package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"fmt"
	"time"
)

var StateChan = make(chan common.ElevatorBehaviour)
var DoorOpenChan = make(chan struct{})
var DoorCloseChan = make(chan struct{})
var AssignedHallCallChan = make(chan elevio.ButtonEvent)
var OrderCompleteChan = make(chan elevio.ButtonEvent)

func InitFSM(myID string, initial common.Elevator) {
	backup.LoadCabRequests(&initial)

	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}

	go StateMachineLoop(myID)
	go executionLoop(myID)
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration)
}

func StateMachineLoop(myID string) {
	for state := range StateChan {
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Behaviour = state
		})
		handleState(myID)
	}
}

func executionLoop(myID string) {
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	var prevDirn elevio.Dirn = elevio.MD_Stop

	for {
		select {
		case btn := <-buttonPressChan:
			handleButtonPress(myID, btn, &prevDirn)

		case assigned := <-AssignedHallCallChan:
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Requests[assigned.Floor][assigned.Button] = true
			})
			UpdateCabLights(GetMyElevator(myID))

			if GetMyElevator(myID).Behaviour == common.IDLE {
				e := GetMyElevator(myID)
				next := ChooseDirection(e, prevDirn)
				WithMyElevator(myID, func(e *common.Elevator) {
					e.Dirn = next.Dirn
				})
				elevio.SetMotorDirection(next.Dirn)
				StateChan <- next.Behaviour
				if next.Dirn != elevio.MD_Stop {
					prevDirn = next.Dirn
				}
			}

		case floor := <-floorSensorChan:
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = floor
			})
			elevio.SetFloorIndicator(floor)

			if RequestShouldStop(GetMyElevator(myID)) {
				StopElevator()
				WithMyElevator(myID, func(e *common.Elevator) {
					e.Behaviour = common.DOOR_OPEN
					e.Dirn = elevio.MD_Stop
				})
				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				DoorOpenChan <- struct{}{}
			}

		case <-DoorCloseChan:
			e := GetMyElevator(myID)
			next := ChooseDirection(e, prevDirn)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = next.Dirn
			})
			elevio.SetMotorDirection(next.Dirn)
			StateChan <- next.Behaviour
			if next.Dirn != elevio.MD_Stop {
				prevDirn = next.Dirn
			}
		}
	}
}

func handleButtonPress(myID string, btn elevio.ButtonEvent, prevDirn *elevio.Dirn) {
	switch btn.Button {
	case elevio.BT_Cab:
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Requests[btn.Floor][elevio.BT_Cab] = true
		})
		backup.SaveCabRequests(GetMyElevator(myID))
		UpdateCabLights(GetMyElevator(myID))

		if GetMyElevator(myID).Behaviour == common.IDLE {
			dirnPair := ChooseDirection(GetMyElevator(myID), *prevDirn)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = dirnPair.Dirn
			})
			elevio.SetMotorDirection(dirnPair.Dirn)
			StateChan <- dirnPair.Behaviour
			if dirnPair.Dirn != elevio.MD_Stop {
				*prevDirn = dirnPair.Dirn
			}
		}

	case elevio.BT_HallUp, elevio.BT_HallDown:
		AssignerInput <- AssignerMsg{Type: "hall_call", Data: btn}
		AssignerInput <- AssignerMsg{Type: "assign", Data: btn}
	}
}

func handleState(myID string) {
	switch GetMyElevator(myID).Behaviour {
	case common.IDLE:
		handleIdleState(myID)
	case common.MOVING:
		handleMovingState(myID)
	case common.DOOR_OPEN:
		// DoorFSM handles this
	}
}

func handleIdleState(myID string) {
	e := GetMyElevator(myID)
	next := ChooseDirection(e, e.Dirn)
	WithMyElevator(myID, func(e *common.Elevator) {
		e.Dirn = next.Dirn
	})
	elevio.SetMotorDirection(next.Dirn)
	StateChan <- next.Behaviour
}

func handleMovingState(myID string) {
	for {
		newFloor := elevio.GetFloor()
		if newFloor != -1 {
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = newFloor
			})
			elevio.SetFloorIndicator(newFloor)

			if RequestShouldStop(GetMyElevator(myID)) {
				StopElevator()
				DoorOpenChan <- struct{}{}
				<-DoorCloseChan
				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				StateChan <- common.IDLE
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func PrintElevatorState(myID string) {
	e := GetMyElevator(myID)
	fmt.Println("========== Elevator State ==========")
	fmt.Printf("ID: %s | Floor: %d | Direction: %v | Behaviour: %v\n",
		e.ID, e.Floor, e.Dirn, e.Behaviour)

	fmt.Println("Requests:")
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


