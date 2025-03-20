package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	 "elevatorlab/pkg/control/movement"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"sync"
	"time"
)

var mutex sync.Mutex
var ElevatorStates map[int]common.Elevator
var HallRequests [common.N_FLOORS][2]common.OrderState

func InitDispatcher() {
	ElevatorStates = make(map[int]common.Elevator)
	go Synchronizer()
}


func AssignRequest(floor int, button elevio.ButtonType, elevatorID int) bool {
	mutex.Lock()
	defer mutex.Unlock()

	//store the hall request
	if HallRequests[floor][button] == common.NonExisting || HallRequests[floor][button] == common.Unknown {
		HallRequests[floor][button] = common.HalfExisting
	}

	//create HRA input and process it
	var hallRequestsBool [common.N_FLOORS][2]bool
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			hallRequestsBool[floor][btn] = (HallRequests[floor][btn] != common.NonExisting && HallRequests[floor][btn] != common.Unknown)
		}
	}
	hraInput := hra.CreateHRAInput(ElevatorStates, hallRequestsBool)
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

					//confirmed only if all peers ack it
					HallRequests[floor][button] = common.Existing
					return true
				}
			}
		}
	}
	return false
}



var networkAlive bool = true

func Synchronizer() {
	perspectiveRx := make(chan common.Perspective)
	perspectiveTx := make(chan common.Perspective)
	peerUpdateCh := make(chan peers.PeerUpdate)

	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)
	go peers.Receiver(15680, peerUpdateCh)

	ticker := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		case peerUpdate := <-peerUpdateCh:
			// If no peers are connected, assume network failure
			networkAlive = len(peerUpdate.Peers) > 0

		case theirs := <-perspectiveRx:
			// Only update hall requests when peers confirm
			mutex.Lock()
			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					if theirs.Perspective[floor][btn] == common.Existing {
						HallRequests[floor][btn] = common.Existing
					}
				}
			}
			mutex.Unlock()

		case <-ticker.C:
			// Send only hall request states (not full elevator states)
			perspectiveTx <- common.Perspective{Perspective: HallRequests}
		}
	}
}

func ChooseDirection(e common.Elevator) common.DirnBehaviourPair {
	if movement.RequestsAbove(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.MOVING}
	}
	if movement.RequestsHere(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.DOOR_OPEN}
	}
	if movement.RequestsBelow(e) {
		return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.MOVING}
	}
	return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
}

// for the current floor:
func ClearRequests(e common.Elevator) {
	for btn := 0; btn < common.N_BUTTONS; btn++ {
		e.Requests[e.Floor][btn] = false
	}
}
