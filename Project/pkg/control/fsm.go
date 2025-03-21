package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"elevatorlab/pkg/control/dispatcher"
	"elevatorlab/pkg/control/indicators"
	"elevatorlab/pkg/control/movement"
	"time"
)

var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)
var OrderComplete = make(chan elevio.ButtonEvent) //tracks completed requests
var DoorOpenChan = make(chan struct{})
var DoorCloseChan = make(chan struct{})

func InitFSM(elevatorID int) {
	Elevator = common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3000,
	}

	backup.LoadCabRequests(&Elevator)

	go indicators.DoorFSM(DoorOpenChan, DoorCloseChan, time.Duration(Elevator.DoorOpenDuration)*time.Millisecond)
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
	Elevator.Requests[buttonPress.Floor][buttonPress.Button] = true

	if buttonPress.Button == elevio.BT_Cab {
		backup.SaveCabRequests(Elevator)
	}

	if Elevator.Behaviour == common.IDLE {
		StateChan <- common.MOVING
	}
}

func handleFloorArrival() {
	Elevator.Behaviour = common.DOOR_OPEN
	DoorOpenChan <- struct{}{} //start door FSM
	<-DoorCloseChan            //wait for door to close
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

func handleIdleState() {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for button := 0; button < 2; button++ {
			if GetOrderState(floor, button) == common.HalfExisting {
				return // Wait for confirmation
			}
		}
	}

	nextDirn := dispatcher.ChooseDirection(Elevator)
	if nextDirn.Behaviour == common.MOVING {
		Elevator.Behaviour = common.MOVING
		Elevator.Dirn = nextDirn.Dirn
	}
}


func handleMovingState() {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			Elevator.Floor = newFloor
			elevio.SetFloorIndicator(newFloor)

			if movement.RequestShouldStop(Elevator) {
				movement.StopElevator()
				Elevator.Behaviour = common.DOOR_OPEN
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func handleDoorOpenState() {
	DoorOpenChan <- struct{}{} //trigger door open
	<-DoorCloseChan            //wait till it closes

	for floor := 0; floor < common.N_FLOORS; floor++ {
		for button := 0; button < 2; button++ {
			if GetOrderState(floor, button) == common.Existing {
				ClearHallRequest(floor, button)
			}
		}
	}

	StateChan <- common.IDLE
}



func UpdateOrderState(floor int, button int, state common.OrderState) {
	common.GlobalPerspective.Perspective[floor][button] = state
}

func GetOrderState(floor int, button int) common.OrderState {
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

