package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control/movement"
	"elevatorlab/pkg/control/dispatcher"
	"time"
	"strconv"
)

var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)

func InitFSM(elevatorID int) {
	Elevator = common.Elevator{
		ID:                  strconv.Itoa(elevatorID),
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
	}

	go StateMachineLoop()
}

func StateMachineLoop() {
	for {
		select {
		case state := <-StateChan:
			Elevator.Behaviour = state
			handleState()
		}
	}
}

func handleState() {
	switch Elevator.Behaviour {
	case common.IDLE:
		handleIdleState()
	case common.MOVING:
		handleMovingState()
	case common.DOOR_OPEN:
		handleDoorOpenState()
	}
}

// Wait for new request or HRA assignment
func handleIdleState() {
	// Wait for full confirmation before assigning requests
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for button := 0; button < 2; button++ {
			if GetOrderState(floor, button) == common.HalfExisting {
				return // Wait for confirmation before acting
			}
		}
	}

	nextDirn := dispatcher.ChooseDirection(Elevator)
	if nextDirn.Behaviour == common.MOVING {
		Elevator.Behaviour = common.MOVING
		Elevator.Dirn = nextDirn.Dirn
	}
}

// Move until reaching a request
func handleMovingState() {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			Elevator.Floor = newFloor
			elevio.SetFloorIndicator(newFloor)

			if movement.RequestShouldStop(Elevator) {
				movement.StopElevator()
				elevio.SetDoorOpenLamp(true)
				Elevator.Behaviour = common.DOOR_OPEN
				time.Sleep(Elevator.DoorOpenDuration)
				dispatcher.ClearRequests(Elevator)
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// hold door open, then determine next action

func handleDoorOpenState() {
	elevio.SetDoorOpenLamp(true)
	time.Sleep(Elevator.DoorOpenDuration)
	elevio.SetDoorOpenLamp(false)

	// Only clear if fully confirmed
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for button := 0; button < 2; button++ {
			if GetOrderState(floor, button) == common.Existing {
				ClearHallRequest(floor, button)
			}
		}
	}

	StateChan <- common.IDLE
}

func OnRequestButtonPress(btn_floor int, btn_type elevio.ButtonType) {
	dispatcher.AssignRequest(btn_floor, btn_type, Elevator.ID)
	Elevator.Requests[btn_floor][btn_type] = true
}

func UpdateOrderState(floor int, button int, state OrderState) {
	common.GlobalPerspective.Perspective[floor][button] = state
}

func GetOrderState(floor int, button int) OrderState {
	return common.GlobalPerspective.Perspective[floor][button]
}

func HandleHallRequest(floor int, button int) {
	currentState := GetOrderState(floor, button)
	if currentState == common.NonExisting || currentState == common.Unknown {
		UpdateOrderState(floor, button, common.HalfExisting)
	}
}

func ClearHallRequest(floor int, button int) {
	if GetOrderState(floor, button) == common.Existing {
		UpdateOrderState(floor, button, common.NonExisting)
	}
}