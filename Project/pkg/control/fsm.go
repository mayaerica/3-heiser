package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control/dispatcher"
	"elevatorlab/pkg/control/movement"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network"
	"time"
)

var Elevator common.Elevator
var StateChan = make(chan common.ElevatorBehaviour)
var ElevatorStateUpdate = make(chan common.Elevator) //this guy will send state updates to the network

func InitFSM(elevatorID int) {
	Elevator = common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
	}

	go StateMachineLoop()
	go network.BroadcastElevatorState(ElevatorStateUpdate) // elevator state is shared across network
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
	hraInput := hra.CreateHRAInput() //get current elevator and hall requests
	hraOutput, err := hra.ProcessElevatorRequests(hraInput)
	if err == nil {
		Elevator = hra.HRAMapToElevator(hraOutput, Elevator) //update elevator with HRA results
	}

	nextDirn := dispatcher.ChooseDirection(Elevator)

	if nextDirn.Behaviour == common.MOVING {
		Elevator.Behaviour = common.MOVING
		Elevator.Dirn = nextDirn.Dirn
		movement.StartMoving(Elevator.Dirn)
		ElevatorStateUpdate <- Elevator // broadcast state update
	} else if nextDirn.Behaviour == common.DOOR_OPEN {
		Elevator.Behaviour = common.DOOR_OPEN
		elevio.SetDoorOpenLamp(true)
		time.Sleep(Elevator.DoorOpenDuration)
		dispatcher.ClearRequests(Elevator)
		StateChan <- common.IDLE // transition back to idle after door closes
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

	nextDirn := dispatcher.ChooseDirection(Elevator)

	if nextDirn.Behaviour == common.MOVING {
		Elevator.Behaviour = common.MOVING
		Elevator.Dirn = nextDirn.Dirn
		movement.StartMoving(Elevator.Dirn)
	} else {
		Elevator.Behaviour = common.IDLE
	}
}

func OnRequestButtonPress(btn_floor int, btn_type elevio.ButtonType) {
	dispatcher.AssignRequest(btn_floor, btn_type, Elevator.ID)
	Elevator.Requests[btn_floor][btn_type] = true
}
