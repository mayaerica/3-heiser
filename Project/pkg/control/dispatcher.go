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
var AssignedTo [common.N_FLOORS][2]int

func InitDispatcher() {
	ElevatorStates = make(map[int]common.Elevator)
	for f:= 0; f<common.N_FLOORS, f++{
		for b:=0;b<2;b++{
			AssignedTo[f][b] = -1
		}
	}
	go Synchronizer()
}

func UpdateLocalElevatorState(e common.Elevator) {
	mutex.Lock()
	ElevatorStates[e.ID] = e
	mutex.Unlock()
}

func UpdateOrderState(floor int, button int, state common.OrderState){
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
			out[floor][btn] = HallRequests[floor][btn] != common.NonExisting && HallRequests[floor][btn] != common.Unknown
		}
	}
	return out
}

func AssignRequest(floor int, button elevio.ButtonType, elevatorID int) bool {
	mutex.Lock()
	if HallRequests[floor][button] == common.NonExisting || HallRequests[floor][button] == common.Unknown {
		HallRequests[floor][button] = common.HalfExisting
	}
	mutex.Unlock()

	hraInput := hra.CreateHRAInput(ElevatorStates, HallRequestsToBool())
	hraOutput, err := hra.ProcessElevatorRequests(hraInput)
	if err != nil {
		return false
	}
	
	mutex.Lock()
	defer mutex.Unlock()
	for id, floorAssignments := range hraOutput {
		for f, btns := range floorAssignments {
			for b, assigned := range btns {
				if assigned {
					AssignedTo[f][b] = id
					if id == elevatorID && f == floor && b == int(button) {
						HallRequests[f][b] = common.Existing
						ElevatorStates[elevatorID].Requests[f][b] = true
						return true
					}
				}
			}
		}
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
		case msg:= <-perspectiveRx:
			mutex.Lock()
			for floor := 0; floor < common.N_FLOORS; floor++{
				for btn:=0; btn<2;btn++{
					if msg.Perspective[floor][btn] == common.Existing{
						HallRequests[floor][btn] = common.Existing
					}
				}
			}
			mutex.Unlock()
		case peers := <-peerUpdateChan:
			mutex.Lock()
			lost := make(map[int]bool)
			exisiting := make(map[int]bool)
			for _, p := range peers.Peers{
				id := parseID(p)
				existing[id] = true
			}

			for id := range ElevatorStates {
				if !existing[id] {
					lost[id] = true
				}
			}
			for id := range lost {
				delete(ElevatorStates, id)
			}
			for f:= 0; f<common.N_FLOORS;f++{
				for b:= 0; b< 2; b++{
					if lost[AssignedTo[f][b]] && HallRequests[f][b] ==common.Existing{
						HallRequests[f][b] = common.HalfExisting
						AssignedTo[f][b] = -1
					}
				}
			}
			mutex.Unlock()

			//reassign
			hraInput := hra.CreateHRAInput(ElevatorStates,HallRequestsToBool())
			hra.ProcessElevatorRequests(hraInput)

		case <-ticker.C:
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


