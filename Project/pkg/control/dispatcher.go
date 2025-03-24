package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"fmt"
	"sync"
	"time"
)

var Mutex sync.Mutex
var Mutex2 sync.Mutex
var ElevatorStates map[string]common.Elevator
var HallRequests [common.N_FLOORS][2]common.OrderState
var orderID [common.N_FLOORS][2]string


func InitDispatcher(myID string, localElev common.Elevator) {
	ElevatorStates = make(map[string]common.Elevator)
	UpdateLocalElevatorState(localElev)
	go Synchronizer(myID)
}

func UpdateLocalElevatorState(e common.Elevator) {
	ElevatorStates[e.ID] = e
}

func UpdateOrderState(floor int, button int, state common.OrderState) {
	Mutex.Lock()
	defer Mutex.Unlock()
	HallRequests[floor][button] = state
}

func GetOrderState(floor int, button int) common.OrderState {
	Mutex.Lock()
	defer Mutex.Unlock()
	return HallRequests[floor][button]
}

func HallRequestsToBool() [common.N_FLOORS][2]bool {
	var out [common.N_FLOORS][2]bool
	Mutex.Lock()
	defer Mutex.Unlock()
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			out[floor][btn] = HallRequests[floor][btn] == common.Assigned
		}
	}
	return out
}

// AssignRequest decides which elevator should handle a new hall call (button press).
//
// 1. It marks the request as Unassigned if it’s new.
// 2. It creates an input for the Hall Request Assigner (HRA), which knows the state of all elevators.
// 3. The HRA runs and returns a plan: which elevator should take which hall calls.
// 4. If this elevator is chosen for the current request, we:
//   - Mark the request as Assigned.
//   - Add the request to this elevator’s list.
//   - Let the rest of the system know it was assigned successfully.
func AssignRequest(floor int, button elevio.ButtonType, elevatorID string) bool {
	Mutex.Lock()
	if HallRequests[floor][button] == common.NotRequested || HallRequests[floor][button] == common.Unknown {
		HallRequests[floor][button] = common.Unassigned
	}
	Mutex.Unlock()

	hraInput := hra.CreateHRAInput(ElevatorStates, HallRequestsToBool())
	hraOutput := hra.HRAProcessor(hraInput)
	if hraOutput == nil {
		fmt.Println("[HRA] Error during assignment")
		return false
	}

	for elevatorID, floorAssignments := range *hraOutput {
		for floor := 0; floor < len(floorAssignments); floor++ {
			for button := 0; button < len(floorAssignments[floor]); button++ {
				if floorAssignments[floor][button] {
					fmt.Printf("[HRA] Assigned request: floor=%d, button=%d to elevator=%s\n", floor, button, elevatorID)
					Mutex2.Lock()
					orderID[floor][button] = elevatorID

					Mutex2.Unlock()
					UpdateOrderState(floor, int(button), common.Assigned)
				} else {
					Mutex2.Lock()
					orderID[floor][button] = ""
					Mutex2.Unlock()
				}
			}
		}
	}

	fmt.Printf("[HRA] No assignment for elevator %s (Floor=%d, Button=%d)\n", elevatorID, floor, button)
	return false
}

// Synchronizer makes sure all elevators agree on which hall requests exist.
//
// It does this by:
// 1. Listening to all local and remote button presses.
// 2. Sharing this elevator’s view (Perspective) with the others over the network.
// 3. Collecting the other elevators' perspectives.Variant: common.CV_All,DoorOpenDuration:    3 * time.Second,
// 4. Promoting requests to 'Assigned' only when ALL elevators agree a button has been pressed.
// 5. When a request is completed (an elevator serves the call), it clears it across the network.
//
// Keeps all elevators in sync, even when one elevator restarts or rejoins.
func Synchronizer(myID string) {
	perspectiveTx := make(chan common.Perspective)
	perspectiveRx := make(chan common.Perspective)
	peerUpdateChan := make(chan peers.PeerUpdate)

	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)
	go peers.Receiver(15680, peerUpdateChan)

	ours := [common.N_FLOORS][2]common.OrderState{}
	perspectiveMap := make(map[string]common.Perspective)
	peerList := peers.PeerUpdate{}
	ticker := time.NewTicker(20 * time.Millisecond)

	for {
		select {
		case buttonPress := <-HallCallRequestChan:
			if ours[buttonPress.Floor][buttonPress.Button] == common.NotRequested ||
				ours[buttonPress.Floor][buttonPress.Button] == common.Unknown {
				ours[buttonPress.Floor][buttonPress.Button] = common.Unassigned
			}

		case completed := <-OrderCompleteChan:
			if len(peerList.Peers) > 1 {
				ours[completed.Floor][completed.Button] = common.NotRequested
			} else {
				ours[completed.Floor][completed.Button] = common.Unknown
			}

		case theirs := <-perspectiveRx:
			perspectiveMap[theirs.ID] = theirs
			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					switch theirs.Perspective[floor][btn] {
					case common.NotRequested:
						if (ours[floor][btn] == common.Assigned || ours[floor][btn] == common.Unknown) && theirs.OrderID[floor][btn] == "Done" {
							ours[floor][btn] = common.NotRequested
						}

					case common.Unassigned:
						if ours[floor][btn] == common.NotRequested || ours[floor][btn] == common.Unknown {
							ours[floor][btn] = common.Unassigned
							//requestChan <-elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
						}
						
					case common.Assigned:
						agreeCount := 0
						for _, p := range perspectiveMap { 
							if p.OrderID[floor][btn] == myID {
								agreeCount++
							}
						}

						if agreeCount == len(peerList.Peers) { 
							//elevator[floor][btn] = common.Assigned
							
						}
						ours[floor][btn] = common.Assigned

					case common.Unknown:
						if ours[floor][btn] == common.NotRequested {
							ours[floor][btn] = common.Unknown
						}
					}
				}
			}

		case peerList = <-peerUpdateChan:
			for _, lost := range peerList.Lost {
				delete(perspectiveMap, lost)
			}

		case <-ticker.C:
			Mutex.Lock()
			Mutex2.Lock()
			common.UpdateGlobalPerspective(common.Perspective{
				ID:          myID,
				OrderID:    orderID,
				Perspective: ours,
			})
			Mutex2.Unlock()
			Mutex.Unlock()
			perspectiveTx <- common.GetGlobalPerspective()
			UpdateOrderStateFromSync(ours)
		}
	}
}

func CreateAssignedHallRequests(perspective [common.N_FLOORS][2]common.OrderState) [common.N_FLOORS][2]bool {
	var hallReq [common.N_FLOORS][2]bool
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			hallReq[floor][btn] = (perspective[floor][btn] == common.Assigned)
		}
	}
	return hallReq
}

func UpdateOrderStateFromSync(syncData [common.N_FLOORS][2]common.OrderState) {
	Mutex.Lock()
	defer Mutex.Unlock()

	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			HallRequests[floor][btn] = syncData[floor][btn]
		}
	}
}

func StartDispatcherLoop(myID string, requestChan <-chan elevio.ButtonEvent, assignChan chan<- elevio.ButtonEvent) {
	go func() {
		for btn := range requestChan {
			success := AssignRequest(btn.Floor, btn.Button, myID)
			if success {
				assignChan <- btn
			}	
			
		}
	}()
}
