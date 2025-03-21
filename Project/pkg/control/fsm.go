package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"time"
	"strconv"
)


var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)
var OrderComplete = make(chan elevio.ButtonEvent) // NEW: Tracks completed requests

func InitFSM(elevatorID int) {
	Elevator = common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
	}
	backup.LoadCabRequests(&Elevator)
	go StateMachineLoop()
	go executionLoop()
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

func executionLoop() {
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	for {
		select {
		case buttonPress := <-buttonPressChan:
			handleButtonPress(buttonPress)

		case floor := <-floorSensorChan:
			Elevator.Floor = floor
			elevio.SetFloorIndicator(floor)

			// Stop at the floor if necessary
			if movement.RequestShouldStop(Elevator) {
				movement.StopElevator()
				handleFloorArrival()
			}

		case <-OrderComplete:
			dispatcher.ClearRequests(Elevator)
			StateChan <- common.IDLE
		}
	}
}

func handleButtonPress(buttonPress elevio.ButtonEvent) {
	dispatcher.AssignRequest(buttonPress.Floor, buttonPress.Button, Elevator.ID)

	if Elevator.Behaviour == common.IDLE {
		StateChan <- common.MOVING
	}
}

func handleFloorArrival() {
	Elevator.Behaviour = common.DOOR_OPEN
	elevio.SetDoorOpenLamp(true)
	time.Sleep(Elevator.DoorOpenDuration)
	elevio.SetDoorOpenLamp(false)

	// Clear the request based on direction
	dispatcher.ClearRequests(Elevator)

	StateChan <- common.IDLE
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
	if btn_type == elevio.BT_Cab {
		backup.SaveCabRequests(Elevator)
	}
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

func handleDoorOpenState() {
	elevio.SetDoorOpenLamp(true)
	doorTimeout := time.After(Elevator.DoorOpenDuration)

	for {
		select {
		case obstructed := <-elevio.PollObstructionSwitch():
			if obstructed {
				doorTimeout = time.After(Elevator.DoorOpenDuration) // Restart timer
			}

		case <-doorTimeout:
			if !elevio.GetObstructionSwitch() {
				elevio.SetDoorOpenLamp(false)
				StateChan <- common.IDLE
				return
			}
		}
	}
}
