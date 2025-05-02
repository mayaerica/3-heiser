package core

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/backup"
	"fmt"
	"time"
)

func InitFSM(myID string,
	initial common.Elevator,
	OrderCompleteChan chan elevio.ButtonEvent,
	HallButtonPressChan chan elevio.ButtonEvent,
	FromHRAChan chan common.Elevator,
	ElevSetChan chan ElevSetMsg,
	ElevGetChan chan ElevGetMsg,
	ExistingOrdersChan chan [common.N_FLOORS][2]bool) {

	ElevSetChan <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}

	DoorOpenChan := make(chan struct{})
	DoorCloseChan := make(chan struct{})

	go StateMachineLoop(myID, DoorOpenChan, DoorCloseChan, ExistingOrdersChan, OrderCompleteChan, HallButtonPressChan, FromHRAChan, ElevSetChan, ElevGetChan)
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration)
	go PrintElevatorState(myID, ElevGetChan)
}

func StateMachineLoop(
	myID string,
	DoorOpenChan chan struct{},
	DoorCloseChan chan struct{},
	ExistingOrdersChan chan [common.N_FLOORS][2]bool,
	OrderCompleteChan chan elevio.ButtonEvent,
	HallButtonPressChan chan elevio.ButtonEvent,
	FromHRAChan chan common.Elevator,
	ElevSetChan chan ElevSetMsg,
	ElevGetChan chan ElevGetMsg) {

	buttonPressChan := make(chan elevio.ButtonEvent, 100)
	floorSensorChan := make(chan int, 100)

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	for {
		PrintElevatorState(myID, ElevGetChan)
		select {
		case btn := <-buttonPressChan:
			e := GetMyElevator(myID, ElevGetChan)

			switch e.Behaviour {
			case common.DOOR_OPEN:
				if e.Floor == btn.Floor &&
					((e.Dirn == elevio.MD_Up && btn.Button == elevio.BT_HallUp) ||
						(e.Dirn == elevio.MD_Down && btn.Button == elevio.BT_HallDown) ||
						e.Dirn == elevio.MD_Stop ||
						btn.Button == elevio.BT_Cab) {
					e = clearAtCurrentFloor(e, OrderCompleteChan)
					UpdateCabLights(e)
					WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
					openDoor(myID, DoorOpenChan, ElevSetChan)
				} else {
					if btn.Button == elevio.BT_Cab {
						e.Requests[btn.Floor][btn.Button] = true
						UpdateCabLights(e)
						WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
					} else {
						HallButtonPressChan <- btn
					}
				}

			case common.MOVING:
				if btn.Button == elevio.BT_Cab {
					e.Requests[btn.Floor][btn.Button] = true
					backup.SaveCabRequests(e)
					UpdateCabLights(e)
					WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
				} else {
					HallButtonPressChan <- btn
				}

			case common.IDLE:
				if btn.Floor == e.Floor {
					e.Behaviour = common.DOOR_OPEN
					WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
					openDoor(myID, DoorOpenChan, ElevSetChan)
				} else {
					if btn.Button == elevio.BT_Cab {
						e.Requests[btn.Floor][btn.Button] = true
						backup.SaveCabRequests(e)
						UpdateCabLights(e)
						WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
						trigger(myID, ElevGetChan, OrderCompleteChan, ElevSetChan, DoorOpenChan)
					} else {
						HallButtonPressChan <- btn
					}
				}

			}

		case floor := <-floorSensorChan:
			elevio.SetFloorIndicator(floor)
			WithMyElevator(myID, ElevSetChan, func(e *common.Elevator) {
				e.Floor = floor
			})

			e := GetMyElevator(myID, ElevGetChan)

			if e.Behaviour == common.MOVING && RequestShouldStop(e) {
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				UpdateCabLights(e)
				elevio.SetMotorDirection(elevio.MD_Stop)
				e.Behaviour = common.DOOR_OPEN
				WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
				openDoor(myID, DoorOpenChan, ElevSetChan)
			}
			backup.SaveCabRequests(e)

		case <-DoorCloseChan:
			WithMyElevator(myID, ElevSetChan, func(e *common.Elevator) {
				e.Behaviour = common.IDLE
			})
			elevio.SetDoorOpenLamp(false)
			trigger(myID, ElevGetChan, OrderCompleteChan, ElevSetChan, DoorOpenChan)

		case assigned := <-FromHRAChan:
			WithMyElevator(myID, ElevSetChan, func(e *common.Elevator) {
				for f := 0; f < common.N_FLOORS; f++ {
					e.Requests[f][elevio.BT_HallUp] = assigned.Requests[f][elevio.BT_HallUp]
					e.Requests[f][elevio.BT_HallDown] = assigned.Requests[f][elevio.BT_HallDown]
				}
			})
			e := GetMyElevator(myID, ElevGetChan)
			if e.Behaviour == common.IDLE {
				trigger(myID, ElevGetChan, OrderCompleteChan, ElevSetChan, DoorOpenChan)
			} else if e.Behaviour == common.DOOR_OPEN {
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				UpdateCabLights(e)
				WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
				openDoor(myID, DoorOpenChan, ElevSetChan)
			}

		}
	}
}
func trigger(myID string,
	ElevGetChan chan ElevGetMsg,
	OrderCompleteChan chan elevio.ButtonEvent,
	ElevSetChan chan ElevSetMsg,
	DoorOpenChan chan struct{}) {

	e := GetMyElevator(myID, ElevGetChan)
	next := ChooseDirection(e, e.Dirn)
	e.Dirn = next.Dirn
	e.Behaviour = next.Behaviour

	if next.Behaviour == common.DOOR_OPEN &&
		(next.Dirn == elevio.MD_Up || next.Dirn == elevio.MD_Down) {
		e.Dirn = elevio.MD_Stop
		e.Behaviour = common.DOOR_OPEN
	}

	if e.Behaviour == common.DOOR_OPEN {
		e = clearAtCurrentFloor(e, OrderCompleteChan)
		UpdateCabLights(e)
		WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })
		openDoor(myID, DoorOpenChan, ElevSetChan)
	}

	WithMyElevator(myID, ElevSetChan, func(me *common.Elevator) { *me = e })

	elevio.SetMotorDirection(e.Dirn)
}

func openDoor(myID string, DoorOpenChan chan struct{}, ElevSetChan chan ElevSetMsg) {
	WithMyElevator(myID, ElevSetChan, func(e *common.Elevator) {
		e.Behaviour = common.DOOR_OPEN
	})
	DoorOpenChan <- struct{}{}
}

func PrintElevatorState(myID string, ElevGetChan chan ElevGetMsg) {
	e := GetMyElevator(myID, ElevGetChan)
	time.Sleep(50 * time.Millisecond)
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
