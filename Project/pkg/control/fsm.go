package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"elevatorlab/pkg/control/dispatcher"
	"elevatorlab/pkg/control/indicators"
	"elevatorlab/pkg/control/movement"
	"elevatorlab/pkg/hra"
	"time"
)

var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)
var DoorOpenChan = make(chan struct{})
var DoorCloseChan = make(chan struct{})

func InitFSM(elevatorID int) {
	Elevator = common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3*time.Second,
	}

	backup.LoadCabRequests(&Elevator)

	go StateMachineLoop()
	go executionLoop()
	go indicators.DoorFSM(DoorOpenChan, DoorCloseChan, Elevator.DoorOpenDuration)
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
				movement.StopElevator() //??
				Elevator.Behaviour = common.DOOR_OPEN
				Elevator.Dirn = elevio.MD_Stop

				movement.ClearRequestAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)

				DoorOpenChan <- struct{}{}
			}

		case <-DoorCloseChan:
			StateChan <-common.IDLE
		}
	}
}

func handleButtonPress(buttonPress elevio.ButtonEvent) {
	dispatcher.AssignRequest(buttonPress.Floor, buttonPress.Button, Elevator.ID)
	Elevator.Requests[buttonPress.Floor][buttonPress.Button] = true

	if buttonPress.Button == elevio.BT_Cab {
		backup.SaveCabRequests(Elevator)
	}

	if Elevator.Behaviour == common.DOOR_OPEN &&
		movement.ShouldClearImmediately(Elevator, buttonPress.Floor, buttonPress.Button){
			movement.ClearRequestAtCurrentFloor(&Elevator)
			indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
			DoorOpenChan <- struct{}{}
	}
		
	if Elevator.Behaviour == common.IDLE {
		StateChan <- common.MOVING
	}
}

func handleState() {
	switch Elevator.Behaviour {
	case common.IDLE:
		handleIdleState()
	case common.MOVING:
		handleMovingState()
	case common.DOOR_OPEN:
		// No-op since we have a door fsm
	}
}

func handleIdleState() {
	hraInput := hra.CreateHRAInput(dispatcher.ElevatorStates, dispatcher.hallRequestsBool())
	hraOutput, err := hra.ProcessElevatorRequests(hraInput)
	if err == nil {
		Elevator = hra.HRAMapToElevator(hraOutput, Elevator)
	}
	nextDirn := dispatcher.ChooseDirection(Elevator)
	Elevator.Dirn = nextDirn.Dirn
	StateChan <- nextDirn.Behaviour
}



func handleMovingState() {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			Elevator.Floor = newFloor
			elevio.SetFloorIndicator(newFloor)

			if movement.RequestShouldStop(Elevator) {
				movement.StopElevator()
				DoorOpenChan <- struct{}{} 
				<-DoorCloseChan
				movement.ClearRequestAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
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

