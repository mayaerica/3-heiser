package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"elevatorlab/pkg/control/indicators"
	"fmt"
	"time"
)

// Channel for transitioning FSM states (IDLE, MOVING, DOOR_OPEN)
var StateChan = make(chan common.ElevatorBehaviour)

// Signal to open the elevator door
var DoorOpenChan = make(chan struct{})

// Signal that the door has closed and FSM can proceed
var DoorCloseChan = make(chan struct{})

// New hall call request received from button press
var HallCallRequestChan = make(chan elevio.ButtonEvent)

// A hall call has been assigned to this elevator by the dispatcher
var AssignedHallCallChan = make(chan elevio.ButtonEvent)

// A hall call has just been completed. Please remove it from the network's shared state
var OrderCompleteChan = make(chan elevio.ButtonEvent)

func InitFSM(elevatorID string) {
	detectedFloor := elevio.GetFloor()

	e := common.Elevator{
		ID:                  elevatorID,
		Behaviour:           common.IDLE,
		Dirn:                elevio.MD_Stop,
		Floor:               detectedFloor,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3 * time.Second,
	}

	if detectedFloor == -1 {
		fmt.Println("Starting inbetween floors -> moving down to find floor...")
		elevio.SetMotorDirection(elevio.MD_Down)
		e.Behaviour = common.MOVING
		e.Dirn = elevio.MD_Down
	} else {
		fmt.Println("Starting at floor", detectedFloor)
		elevio.SetFloorIndicator(detectedFloor)
		e.Behaviour = common.IDLE
		e.Dirn = elevio.MD_Stop
	}

	SetLocalElevator(e)
	backup.LoadCabRequests(&e)
	SetLocalElevator(e)

	go StateMachineLoop()
	go executionLoop()
	go DoorFSM(DoorOpenChan, DoorCloseChan, e.DoorOpenDuration)
}

func StateMachineLoop() {
	for {
		select {
		case state := <-StateChan:
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Behaviour = state
			})
			handleState()
		}
	}
}

func executionLoop() {
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			PrintElevatorState()
		}
	}()

	var prevDirn elevio.Dirn = elevio.MD_Stop

	for {
		select {
		case buttonPress := <-buttonPressChan:
			fmt.Printf("[BTNPRESSED] Floor: %d, Button: %v\n", buttonPress.Floor, buttonPress.Button)
			handleButtonPress(buttonPress, &prevDirn)

		case assignedBtn := <-AssignedHallCallChan:
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Requests[assignedBtn.Floor][assignedBtn.Button] = true
			})
			indicators.UpdateAllLights(GetLocalElevator(), common.GlobalPerspective.Perspective)

			if GetLocalElevator().Behaviour == common.IDLE {
				e := GetLocalElevator()
				next := ChooseDirection(e, prevDirn)
				UpdateLocalElevator(func(e *common.Elevator) {
					e.Dirn = next.Dirn
				})
				StateChan <- next.Behaviour
				if next.Dirn != elevio.MD_Stop {
					prevDirn = next.Dirn
				}
			}

		case floor := <-floorSensorChan:
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Floor = floor
			})
			elevio.SetFloorIndicator(floor)

			if RequestShouldStop(GetLocalElevator()) {
				StopElevator()
				UpdateLocalElevator(func(e *common.Elevator) {
					e.Behaviour = common.DOOR_OPEN
					e.Dirn = elevio.MD_Stop
				})
				SendCompletedHallRequests(GetLocalElevator())
				ClearRequestsAtCurrentFloor()
				indicators.UpdateAllLights(GetLocalElevator(), common.GlobalPerspective.Perspective)
				DoorOpenChan <- struct{}{}
			}

		case <-DoorCloseChan:
			e := GetLocalElevator()
			next := ChooseDirection(e, prevDirn)
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Dirn = next.Dirn
			})
			StateChan <- next.Behaviour
			if next.Dirn != elevio.MD_Stop {
				prevDirn = next.Dirn
			}
		}
	}
}

func handleButtonPress(buttonPress elevio.ButtonEvent, prevDirn *elevio.Dirn) {
	switch buttonPress.Button {
	case elevio.BT_Cab:
		UpdateLocalElevator(func(e *common.Elevator) {
			e.Requests[buttonPress.Floor][elevio.BT_Cab] = true
		})
		backup.SaveCabRequests(GetLocalElevator())
		indicators.UpdateAllLights(GetLocalElevator(), common.GlobalPerspective.Perspective)

		if GetLocalElevator().Behaviour == common.IDLE {
			dirnPair := ChooseDirection(GetLocalElevator(), *prevDirn)
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Dirn = dirnPair.Dirn
			})
			StateChan <- dirnPair.Behaviour
			if dirnPair.Dirn != elevio.MD_Stop {
				*prevDirn = dirnPair.Dirn
			}
		}
	case elevio.BT_HallUp, elevio.BT_HallDown:
		HallCallRequestChan <- buttonPress
	}
}

func handleState() {
	switch GetLocalElevator().Behaviour {
	case common.IDLE:
		handleIdleState()
	case common.MOVING:
		handleMovingState()
	case common.DOOR_OPEN:
		// no-op
	}
}

func handleIdleState() {
	e := GetLocalElevator()
	next := ChooseDirection(e, e.Dirn)
	UpdateLocalElevator(func(e *common.Elevator) {
		e.Dirn = next.Dirn
	})
	StateChan <- next.Behaviour
}

func handleMovingState() {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			UpdateLocalElevator(func(e *common.Elevator) {
				e.Floor = newFloor
			})
			elevio.SetFloorIndicator(newFloor)

			if RequestShouldStop(GetLocalElevator()) {
				StopElevator()
				DoorOpenChan <- struct{}{}
				<-DoorCloseChan
				SendCompletedHallRequests(GetLocalElevator())
				ClearRequestsAtCurrentFloor()
				indicators.UpdateAllLights(GetLocalElevator(), common.GlobalPerspective.Perspective)
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func SendCompletedHallRequests(e common.Elevator) {
	floor := e.Floor
	if e.Requests[floor][elevio.BT_HallUp] {
		OrderCompleteChan <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
	}
	if e.Requests[floor][elevio.BT_HallDown] {
		OrderCompleteChan <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
	}
}

func PrintElevatorState() {
	e := GetLocalElevator()
	fmt.Println("========== Elevator State ==========")
	fmt.Printf("ID: %s | Floor: %d | Direction: %v | Behaviour: %v\n",
		e.ID, e.Floor, e.Dirn, e.Behaviour)

	fmt.Println("Requests: ")
	for floor := 0; floor < common.N_FLOORS; floor++ {
		fmt.Printf("  Floor %d: [Cab: %v, Up: %v, Down: %v]\n",
			floor,
			e.Requests[floor][elevio.BT_Cab],
			e.Requests[floor][elevio.BT_HallUp],
			e.Requests[floor][elevio.BT_HallDown],
		)
	}
	fmt.Println("====================================")
}
