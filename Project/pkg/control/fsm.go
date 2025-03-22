package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"elevatorlab/pkg/control/indicators"
	"time"
)

var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)
var DoorOpenChan = make(chan struct{})
var DoorCloseChan = make(chan struct{})
var HallCallRequestChan = make(chan elevio.ButtonEvent)
var AssignedHallCallChan = make(chan elevio.ButtonEvent)

func InitFSM(elevatorID string) {
	Elevator = common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3 * time.Second,
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
		
		case assignedBtn:= <-AssignedHallCallChan:
			Elevator.Requests[assignedBtn.Floor][assignedBtn.Button] = true
			indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)

		case floor := <-floorSensorChan:
			Elevator.Floor = floor
			elevio.SetFloorIndicator(floor)

			if RequestShouldStop(Elevator) {
				StopElevator() //?? have to make this
				Elevator.Behaviour = common.DOOR_OPEN
				Elevator.Dirn = elevio.MD_Stop
				ClearRequestsAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
				DoorOpenChan <- struct{}{}
			}

		case <-DoorCloseChan:
			StateChan <- common.IDLE
		}
	}
}

func handleButtonPress(buttonPress elevio.ButtonEvent) {
	if buttonPress.Button == elevio.BT_Cab {
		Elevator.Requests[buttonPress.Floor][elevio.BT_Cab] = true
		backup.SaveCabRequests(Elevator)
		indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
	} else {
		HallCallRequestChan <- buttonPress
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
		// no-op
	}
}

func handleIdleState() {
	nextDirn := ChooseDirection(Elevator)
	Elevator.Dirn = nextDirn.Dirn
	StateChan <- nextDirn.Behaviour
}

func handleMovingState() {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			Elevator.Floor = newFloor
			elevio.SetFloorIndicator(newFloor)

			if RequestShouldStop(Elevator) {
				StopElevator()
				DoorOpenChan <- struct{}{}
				<-DoorCloseChan
				ClearRequestsAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}


