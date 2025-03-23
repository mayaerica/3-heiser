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

var mutex sync.Mutex
var ElevatorStates map[string]common.Elevator
var HallRequests [common.N_FLOORS][2]common.OrderState

func InitDispatcher(myID string, localElev common.Elevator) {
	ElevatorStates = make(map[string]common.Elevator)
	UpdateLocalElevatorState(localElev) //register yourself
	go Synchronizer()
}

func UpdateLocalElevatorState(e common.Elevator) {
	mutex.Lock()
	defer mutex.Unlock()
	ElevatorStates[e.ID] = e
}

func UpdateOrderState(floor int, button int, state common.OrderState) {
	mutex.Lock()
	defer mutex.Unlock()
	HallRequests[floor][button] = state
}

func GetOrderState(floor int, button int) common.OrderState {
	mutex.Lock()
	defer mutex.Unlock()
	return HallRequests[floor][button]
}

func HallRequestsToBool() [common.N_FLOORS][2]bool {
	var out [common.N_FLOORS][2]bool
	mutex.Lock()
	defer mutex.Unlock()
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			out[floor][btn] = HallRequests[floor][btn] != common.NotRequested && HallRequests[floor][btn] != common.Unknown
		}
	}
	return out
}

func AssignRequest(floor int, button elevio.ButtonType, elevatorID string) bool {
	mutex.Lock()
	if HallRequests[floor][button] == common.NotRequested || HallRequests[floor][button] == common.Unknown {
		HallRequests[floor][button] = common.Unassigned
	}
	mutex.Unlock()

	hraInput := hra.CreateHRAInput(ElevatorStates, HallRequestsToBool())
	hraOutput := hra.HRAProcessor(hraInput)
	if hraOutput  == nil {
		fmt.Println("[HRA] Error during assignment")
		return false
	}

	if floorAssignments, ok := (*hraOutput)[elevatorID]; ok {
		if floorAssignments[floor][button] {
			fmt.Printf("[HRA] Assigned request: floor=%d, button=%d to elevator=%s\n", floor, button, elevatorID)

			UpdateOrderState(floor, int(button), common.Assigned)
			
			mutex.Lock()
			elevator := ElevatorStates[elevatorID]
			elevator.Requests[floor][button] = true
			ElevatorStates[elevatorID] = elevator
			mutex.Unlock()

			return true
		}
	}

	fmt.Printf("[HRA] No assignment for elevator %s (Floor=%d, Button=%d)\n", elevatorID, floor, button)
	return false
}

func Synchronizer() {
	perspectiveTx := make(chan common.Perspective)
	perspectiveRx := make(chan common.Perspective)
	peerUpdateChan := make(chan peers.PeerUpdate)

	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)
	go peers.Receiver(15680, peerUpdateChan)

	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case msg := <-perspectiveRx:
			mutex.Lock()
			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					if msg.Perspective[floor][btn] == common.Assigned {
						HallRequests[floor][btn] = common.Assigned
					}
				}
			}
			mutex.Unlock()

		case <-ticker.C:
			perspectiveTx <- common.Perspective{Perspective: HallRequests}
		}
	}
}

func ChooseDirection(e common.Elevator) common.DirnBehaviourPair {
	if RequestsAbove(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.MOVING}
	}
	if RequestsHere(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.DOOR_OPEN}
	}
	if RequestsBelow(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.MOVING}
	}
	return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
}

// listener loop
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