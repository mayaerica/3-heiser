package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"fmt"
	"time"
)

// var (
// 	DoorOpenChan       = make(chan struct{})
// 	DoorCloseChan      = make(chan struct{})
// 	// OrderCompleteChan  = make(chan elevio.ButtonEvent)
// 	ExistingOrdersChan = make(chan [common.N_FLOORS][2]bool)
// )

type AssignerMsg struct {
	Data elevio.ButtonEvent
}

var AssignerInput = make(chan AssignerMsg)

func InitFSM(myID string,
	initial common.Elevator,
	orderComplete chan elevio.ButtonEvent,
	hallButtonPress chan elevio.ButtonEvent,
	eFromHRA chan common.Elevator,
	ElevSet chan ElevSetMsg,
	ElevGet chan ElevGetMsg,
	ExistingOrdersChan chan [common.N_FLOORS][2]bool) {
	// backup.LoadCabRequests(&initial)
	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}
	DoorOpenChan := make(chan struct{})
	DoorCloseChan := make(chan struct{})
	

	go StateMachineLoop(myID, DoorOpenChan, DoorCloseChan, ExistingOrdersChan, orderComplete, hallButtonPress, eFromHRA, ElevSet, ElevGet)
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration)
	go PrintElevatorState(myID, ElevGet)

	fmt.Print("\n [FSM]: init done")
}

func StateMachineLoop(
	myID string,
	DoorOpenChan chan struct{},
	DoorCloseChan chan struct{},
	ExistingOrdersChan chan [common.N_FLOORS][2]bool,
	OrderCompleteChan chan elevio.ButtonEvent,
	hallButtonPress chan elevio.ButtonEvent,
	eFromHRA chan common.Elevator,
	ElevSet chan ElevSetMsg,
	ElevGet chan ElevGetMsg) {

	fmt.Println("\n [FSM]: Enter in state machin loop")
	buttonPressChan := make(chan elevio.ButtonEvent, 100)
	floorSensorChan := make(chan int, 100)
	//nexMove:= make....

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	//trigger := make(chan struct{}, 3)
	var prevDirn elevio.Dirn = elevio.MD_Stop

	for {
		select {
		case btn := <-buttonPressChan:
			fmt.Println("[FSM]: Button pressed")
			e := GetMyElevator(myID, ElevGet)

			switch e.Behaviour {
			case common.DOOR_OPEN:
				fmt.Println("[FSM]: DOOR OPEN")
				if e.Floor == btn.Floor &&
					((e.Dirn == elevio.MD_Up && btn.Button == elevio.BT_HallUp) ||
						(e.Dirn == elevio.MD_Down && btn.Button == elevio.BT_HallDown) ||
						e.Dirn == elevio.MD_Stop ||
						btn.Button == elevio.BT_Cab) {
					e = clearAtCurrentFloor(e, OrderCompleteChan)
					UpdateCabLights(e)
					WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
					openDoor(myID, DoorOpenChan, ElevSet)
				} else {
					if btn.Button == elevio.BT_Cab {
						e.Requests[btn.Floor][btn.Button] = true
						UpdateCabLights(e)
						WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
					} else {
						hallButtonPress <- btn
					}
				}

			case common.MOVING:
				fmt.Println("[FSM]: MOVING")
				if btn.Button == elevio.BT_Cab {
					e.Requests[btn.Floor][btn.Button] = true
					UpdateCabLights(e)
					WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
				} else {
					hallButtonPress <- btn
				}

			case common.IDLE:
				fmt.Println("[FSM]: IDLE")
				if btn.Floor == e.Floor {
					e.Behaviour = common.DOOR_OPEN
					WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
					openDoor(myID, DoorOpenChan, ElevSet)
				} else {
					if btn.Button == elevio.BT_Cab {
						e.Requests[btn.Floor][btn.Button] = true
						UpdateCabLights(e)
						WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
						prevDirn = trigger(myID, ElevGet, prevDirn, OrderCompleteChan, ElevSet, DoorOpenChan)
					} else {
						hallButtonPress <- btn
					}
				}

			}

		case floor := <-floorSensorChan:
			fmt.Println("[FSM]: Arrived at floor")
			elevio.SetFloorIndicator(floor)
			WithMyElevator(myID, ElevSet, func(e *common.Elevator) {
				e.Floor = floor
			})

			e := GetMyElevator(myID, ElevGet)

			if e.Behaviour == common.MOVING && RequestShouldStop(e) {
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				UpdateCabLights(e)
				prevDirn = e.Dirn
				elevio.SetMotorDirection(elevio.MD_Stop)
				e.Behaviour = common.DOOR_OPEN
				WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
				openDoor(myID, DoorOpenChan, ElevSet)
			}

		case <-DoorCloseChan:
			fmt.Println("[FSM]: Door close")
			WithMyElevator(myID, ElevSet, func(e *common.Elevator) {
				e.Behaviour = common.IDLE
			})
			elevio.SetDoorOpenLamp(false)

			prevDirn = trigger(myID, ElevGet, prevDirn, OrderCompleteChan, ElevSet, DoorOpenChan)

		case assigned := <-eFromHRA:
			fmt.Println("[FSM]: Recieved assigned tasks")
			WithMyElevator(myID, ElevSet, func(e *common.Elevator) {
				for f := 0; f < common.N_FLOORS; f++ {
					e.Requests[f][elevio.BT_HallUp] = assigned.Requests[f][elevio.BT_HallUp]
					e.Requests[f][elevio.BT_HallDown] = assigned.Requests[f][elevio.BT_HallDown]
				}
			})
			e := GetMyElevator(myID, ElevGet)
			if e.Behaviour == common.IDLE {
				prevDirn = trigger(myID, ElevGet, prevDirn, OrderCompleteChan, ElevSet, DoorOpenChan)
			} else if e.Behaviour == common.DOOR_OPEN {
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				UpdateCabLights(e)
				WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
				openDoor(myID, DoorOpenChan, ElevSet)
			}

		}
	}
}
func trigger(myID string, ElevGet chan ElevGetMsg, prevDirn elevio.Dirn, OrderCompleteChan chan elevio.ButtonEvent, ElevSet chan ElevSetMsg, DoorOpenChan chan struct{}) elevio.Dirn {
	fmt.Println("[FSM]: trigger")
	e := GetMyElevator(myID, ElevGet)
	next := ChooseDirection(e, prevDirn)
	prevDirn = e.Dirn
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
		WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })
		openDoor(myID, DoorOpenChan, ElevSet)
	}

	WithMyElevator(myID, ElevSet, func(me *common.Elevator) { *me = e })

	elevio.SetMotorDirection(e.Dirn)
	fmt.Printf("[FSM] Requests: %+v\n", e.Requests)
	return prevDirn
}

// for {
// 	fmt.Println("[FSM]: Looping")
// 	select {
// 	case btn := <-buttonPressChan:
// 		fmt.Print("[FSM]: button pressed")
// 		handleButtonPress(myID, btn, &prevDirn)

// 	case floor := <-floorSensorChan:
// 		fmt.Print("[FSM]: Arrived at floor\n")
// 		elevio.SetFloorIndicator(floor)
// 		WithMyElevator(myID, func(e *common.Elevator) {
// 			e.Floor = floor
// 		})
// 		fmt.Println("floor @1")
// 		e := GetMyElevator(myID)
// 		if RequestShouldStop(e) {
// 			fmt.Println("floor @2")
// 			StopElevator()
// 			fmt.Println("floor @3")
// 			e = clearAtCurrentFloor(e, OrderCompleteChan)
// 			fmt.Println("floor @4")
// 			WithMyElevator(myID, func(me *common.Elevator) {
// 				*me = e
// 			})
// 			fmt.Println("floor @5")
// 			UpdateCabLights(e)
// 			fmt.Println("floor @6")
// 			openDoor(myID, DoorOpenChan)
// 			fmt.Println("floor @7")
// 		}
// 		fmt.Println("floor done")

// 	case orders := <-ExistingOrdersChan:
// 		fmt.Print("[FSM]: orders received")

// 		// Directly use the `orders` array here without looping over it (assuming `orders` is a single item)
// 		WithMyElevator(myID, func(e *common.Elevator) {
// 			for f := 0; f < common.N_FLOORS; f++ {
// 				for b := 0; b < 2; b++ {
// 					e.Requests[f][b] = orders[f][b]
// 				}
// 			}
// 		})

// 	case <-DoorCloseChan:
// 		fmt.Println("[FSM]: close door")
// 		e := GetMyElevator(myID)
// 		next := ChooseDirection(e, e.Dirn)
// 		WithMyElevator(myID, func(e *common.Elevator) {
// 			e.Dirn = next.Dirn
// 			e.Behaviour = next.Behaviour
// 		})
// 		if next.Behaviour == common.MOVING {
// 			elevio.SetMotorDirection(next.Dirn)
// 		}
// 	}
// }

// CHECK THIS
// func handleButtonPress(myID string, btn elevio.ButtonEvent, prevDirn *elevio.Dirn, ElevSet chan ElevSetMsg, ElevGet chan ElevGetMsg) {
// 	if btn.Button == elevio.BT_Cab {
// 		WithMyElevator(myID, ElevSet, func(e *common.Elevator) {
// 			e.Requests[btn.Floor][btn.Button] = true
// 		})
// 		// backup.SaveCabRequests(GetMyElevator(myID))
// 		UpdateCabLights(GetMyElevator(myID, ElevGet))
// 	} else {
// 		AssignerInput <- AssignerMsg{Data: btn}
// 	}
// }

func openDoor(myID string, DoorOpenChan chan struct{}, ElevSet chan ElevSetMsg) {
	WithMyElevator(myID, ElevSet, func(e *common.Elevator) {
		e.Behaviour = common.DOOR_OPEN
	})
	DoorOpenChan <- struct{}{}
}

func PrintElevatorState(myID string, ElevGet chan ElevGetMsg) {

	e := GetMyElevator(myID, ElevGet)
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
