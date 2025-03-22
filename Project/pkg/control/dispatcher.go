package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"strconv"
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

func UpdateLocalElevatorState(e common.Elevator) {
	mutex.Lock()
	id, err := strconv.Atoi(e.ID)
	if err != nil {
		return
	} //change this part somehow
	ElevatorStates[id] = e
	mutex.Unlock()
}

func UpdateOrderState(floor int, button int, state common.OrderState) {
	mutex.Lock()
	HallRequests[floor][button] = state
	mutex.Unlock()
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

func AssignRequest(floor int, button elevio.ButtonType, elevatorID int) bool {
	mutex.Lock()
	if HallRequests[floor][button] == common.NotRequested || HallRequests[floor][button] == common.Unknown {
		HallRequests[floor][button] = common.Unassigned
	}
	mutex.Unlock()

	hraInput := hra.CreateHRAInput(ElevatorStates, HallRequestsToBool())
	hraOutput, err := hra.ProcessElevatorRequests(hraInput)
	if err != nil {
		return false
	}

	if floorAssignments, ok := hraOutput[elevatorID]; ok {
		if floorAssignments[floor][button] {
			UpdateOrderState(floor, int(button), common.Assigned)
			elevator := ElevatorStates[elevatorID]
			elevator.Requests[floor][button] = true
			ElevatorStates[elevatorID] = elevator
			return true
		} //fix this one
	}
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
func StartDispatcherLoop(localElevID int, requestChan <-chan elevio.ButtonEvent, assignChan chan<- elevio.ButtonEvent) {
	go func() {
		for btn := range requestChan {
			success := AssignRequest(btn.Floor, btn.Button, localElevID)
			if success {
				assignChan <- btn
			}
		}
	}()
}
