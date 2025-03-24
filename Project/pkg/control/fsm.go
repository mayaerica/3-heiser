package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"elevatorlab/pkg/control/indicators"
	"sync"
	"fmt"
	"time"
)

var elevator_mutex sync.Mutex
var Elevator common.Elevator

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

	Elevator = common.Elevator{
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
		Elevator.Behaviour = common.MOVING
		Elevator.Dirn = elevio.MD_Down
	} else {
		fmt.Println("Starting at floor", detectedFloor)
		elevio.SetFloorIndicator(detectedFloor)
		Elevator.Behaviour = common.IDLE
		Elevator.Dirn = elevio.MD_Stop
	}

	backup.LoadCabRequests(&Elevator)
	InitDispatcher(Elevator.ID, Elevator)
	
	go StateMachineLoop()
	go executionLoop()
	go DoorFSM(DoorOpenChan, DoorCloseChan, Elevator.DoorOpenDuration)

	
}

func StateMachineLoop() {
	for {
		select {
		case state := <-StateChan:
			fmt.Println(1,1)
			elevator_mutex.Lock()
			fmt.Println(1,1)
			Elevator.Behaviour = state
			fmt.Println("what")
			handleState()
			elevator_mutex.Unlock()
			fmt.Println(1,2)
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
			fmt.Println(2,1)
			elevator_mutex.Lock()
			fmt.Println(2,1)
			PrintElevatorState()
			elevator_mutex.Unlock()
			fmt.Println(2,2)
		}
	}()

	var prevDirn elevio.Dirn = elevio.MD_Stop

	for {
		select {
		case buttonPress := <-buttonPressChan:
			fmt.Printf("[BTNPRESSED] Floor: %d, Button: %v\n", buttonPress.Floor, buttonPress.Button)
			fmt.Println(3,1)
			elevator_mutex.Lock()
			handleButtonPress(buttonPress, &prevDirn)
			elevator_mutex.Unlock()
			fmt.Println(3,2)

		case assignedBtn := <-AssignedHallCallChan:
			fmt.Println(4,1)
			elevator_mutex.Lock()
			Elevator.Requests[assignedBtn.Floor][assignedBtn.Button] = true
			indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
			if Elevator.Behaviour == common.IDLE {
				next := ChooseDirection(Elevator, prevDirn)
				Elevator.Dirn = next.Dirn
				StateChan <- next.Behaviour
				if next.Dirn != elevio.MD_Stop {
					prevDirn = next.Dirn
				}
			}
			elevator_mutex.Unlock()
			fmt.Println(4,2)

		case floor := <-floorSensorChan:
			fmt.Println(5,1)
			elevator_mutex.Lock()
			Elevator.Floor = floor
			elevio.SetFloorIndicator(floor)

			if RequestShouldStop(Elevator) {
				StopElevator()
				Elevator.Behaviour = common.DOOR_OPEN
				Elevator.Dirn = elevio.MD_Stop
				SendCompletedHallRequests(Elevator)
				ClearRequestsAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GetGlobalPerspective().Perspective)
				DoorOpenChan <- struct{}{}
			}
			elevator_mutex.Unlock()
			fmt.Println(5,2)

		case <-DoorCloseChan:
			fmt.Println(6,1)
			elevator_mutex.Lock()
			next := ChooseDirection(Elevator, prevDirn)
			Elevator.Dirn = next.Dirn
			elevator_mutex.Unlock()
			fmt.Println(6,2)
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
		fmt.Println(7,1)
		elevator_mutex.Lock()
		Elevator.Requests[buttonPress.Floor][elevio.BT_Cab] = true
		backup.SaveCabRequests(Elevator)
		indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)

		if Elevator.Behaviour == common.IDLE {
			dirnPair := ChooseDirection(Elevator, *prevDirn)
			Elevator.Dirn = dirnPair.Dirn
			StateChan <- dirnPair.Behaviour
			if dirnPair.Dirn != elevio.MD_Stop {
				*prevDirn = dirnPair.Dirn
			}
		}
		elevator_mutex.Unlock()
		fmt.Println(7,2)
	case elevio.BT_HallUp, elevio.BT_HallDown:
		HallCallRequestChan <- buttonPress
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
	next := ChooseDirection(Elevator, Elevator.Dirn)
	Elevator.Dirn = next.Dirn
	StateChan <- next.Behaviour
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
				SendCompletedHallRequests(Elevator)
				ClearRequestsAtCurrentFloor(&Elevator)
				indicators.UpdateAllLights(Elevator, common.GlobalPerspective.Perspective)
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
	fmt.Println("========== Elevator State ==========")
	fmt.Printf("ID: %s | Floor: %d | Direction: %v | Behaviour: %v\n",
		Elevator.ID, Elevator.Floor, Elevator.Dirn, Elevator.Behaviour)

	fmt.Println("Requests: ")
	for floor := 0; floor < common.N_FLOORS; floor++ {
		fmt.Printf("  Floor %d: [Cab: %v, Up: %v, Down: %v]\n",
			floor,
			Elevator.Requests[floor][elevio.BT_Cab],
			Elevator.Requests[floor][elevio.BT_HallUp],
			Elevator.Requests[floor][elevio.BT_HallDown],
		)
	}
	fmt.Println("====================================")
}
