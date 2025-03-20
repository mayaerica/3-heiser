package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/control/movement"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"sync"
	"strconv"
	"time"
)

var mutex sync.Mutex
var ElevatorStates map[int]common.Elevator
var HallRequests [common.N_FLOORS][2]bool

func InitDispatcher() {
	ElevatorStates = make(map[int]common.Elevator)
	go Synchronizer()
}

// this guy hopefully use hra correctly and assign the hall requests
func AssignRequest(floor int, button elevio.ButtonType, elevatorID int) bool {
	mutex.Lock()
	defer mutex.Unlock()

	//store the hall request
	HallRequests[floor][button] = true

	//create HRA input and process it
	hraInput := hra.CreateHRAInput(ElevatorStates, HallRequests)
	hraOutput, err := hra.ProcessElevatorRequests(hraInput)
	if err != nil {
		return false
	}

	//then we assign the hall requests based on HRA output
	for assignedElevatorID, floorRequests := range hraOutput {
		if assignedElevatorID == elevatorID {
			for assignedFloor, assignedButtons := range floorRequests {
				if assignedFloor == floor && assignedButtons[button] {
					elevator := ElevatorStates[elevatorID]
					elevator.Requests[floor][button] = true
					ElevatorStates[elevatorID] = elevator
					return true
				}
			}
		}
	}
	return false
}



var networkAlive bool = true

func Synchronizer(existingOrders chan<- [N_FLOORS][2]bool, myID string) {
    perspectiveRx := make(chan control.Perspective)
    perspectiveTx := make(chan control.Perspective)

    go bcast.Transmitter(21478, perspectiveTx)
    go bcast.Receiver(21478, perspectiveRx)

    peerUpdateCh := make(chan peers.PeerUpdate)
    go peers.Receiver(15680, peerUpdateCh)

    ticker := time.NewTicker(50 * time.Millisecond) // Optimized frequency

    for {
        select {
        case peerUpdate := <-peerUpdateCh:
            if len(peerUpdate.Peers) == 0 {
                networkAlive = false
            } else {
                networkAlive = true
            }

        case theirs := <-perspectiveRx:
            for floor := 0; floor < N_FLOORS; floor++ {
                for btn := 0; btn < 2; btn++ {
                    if theirs.Perspective[floor][btn] == control.Existing {
                        control.UpdateOrderState(floor, btn, control.Existing)
                    }
                }
            }

        case <-ticker.C:
            if networkAlive {
                perspectiveTx <- control.GlobalPerspective
            }
        }
    }
}

func ChooseDirection(e common.Elevator) common.DirnBehaviourPair {
	if movement.requestsAbove(e) {
		return common.DirnBehaviourPair{elevio.MD_Up, common.MOVING}
	}
	if movement.requestsHere(e) {
		return common.DirnBehaviourPair{elevio.MD_Stop, common.DOOR_OPEN}
	}
	if movement.requestsBelow(e) {
		return common.DirnBehaviourPair{elevio.MD_Down, common.MOVING}
	}
	return common.DirnBehaviourPair{elevio.MD_Stop, common.IDLE}
}

// for the current floor:
func ClearRequests(e common.Elevator) {
	for btn := 0; btn < common.N_BUTTONS; btn++ {
		e.Requests[e.Floor][btn] = false
	}
}
