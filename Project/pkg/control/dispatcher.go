package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"sync"
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
					ElevatorStates[elevatorID].Requests[floor][button] = true
					return true
				}
			}
		}
	}
	return false
}

func Synchronizer() {
	stateTx := make(chan common.Elevator) //sends state updates
	stateRx := make(chan common.Elevator) //recieves state updates

	go bcast.Transmitter(1600, stateTx) //broadcast elevator state
	go bcast.Receiver(1600, stateRx)    //recieve state updates

	peerUpdateChan := make(chan peers.PeerUpdate)
	go peers.Receiver(15000, peerUpdateChan)

	ticker := time.NewTicker(200 * time.Millisecond)

	for {
		select {
		case newState := <-stateRx:
			mutex.Lock()
			ElevatorStates[newState.ID] = newState
			mutex.Unlock()
		case peerList := <-peerUpdateChan:
			for _, lostPeer := range peerList.Lost {
				mutex.Lock()
				delete(ElevatorStates, lostPeer)
				mutex.Unlock()
			}
		case <-ticker.C:
			mutex.Lock()
			for _, elevator := range ElevatorStates {
				stateTx <- elevator //broadcast state updates
			}
			mutex.Unlock()
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
