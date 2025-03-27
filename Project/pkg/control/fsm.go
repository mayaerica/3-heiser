package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"fmt"
	"time"
)

var (
	DoorOpenChan       = make(chan struct{})
	DoorCloseChan      = make(chan struct{})
	OrderCompleteChan  = make(chan elevio.ButtonEvent)
	ExistingOrdersChan = make(chan [common.N_FLOORS][2]bool)
)

type AssignerMsg struct {
	Data elevio.ButtonEvent
}

var AssignerInput = make(chan AssignerMsg)

func InitFSM(myID string, initial common.Elevator) {
	// backup.LoadCabRequests(&initial)
	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}
	go StateMachineLoop(myID)
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration)
	go PrintElevatorState(myID)
	fmt.Print("\n [FSM]: init done")
}

func StateMachineLoop(myID string) {
	fmt.Print("\n [FSM]: Enter in state machin loop")
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)
	//nexMove:= make....

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	var prevDirn elevio.Dirn = elevio.MD_Stop

	// go func() {
	// 	for orders := range ExistingOrdersChan {
	// 		fmt.Print("[FSM]: orders recieved")
	// 		WithMyElevator(myID, func(e *common.Elevator) {
	// 			for f := 0; f < common.N_FLOORS; f++ {
	// 				for b := 0; b < 2; b++ {
	// 					e.Requests[f][b] = orders[f][b]
	// 				}
	// 			}
	// 		})
	// 	}
	// }()

	for {
		select {
		case btn := <-buttonPressChan:
			fmt.Print("[FSM]: button pressed")
			handleButtonPress(myID, btn, &prevDirn)

		case floor := <-floorSensorChan:
			fmt.Print("[FSM]: Arrived at floor\n")
			elevio.SetFloorIndicator(floor)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = floor
			})
			e := GetMyElevator(myID)
			if RequestShouldStop(e) {
				StopElevator()
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				WithMyElevator(myID, func(me *common.Elevator) {
					*me = e
				})
				UpdateCabLights(e)
				openDoor(myID)
			}

		case orders := <-ExistingOrdersChan:
			fmt.Print("[FSM]: orders received")

			// Directly use the `orders` array here without looping over it (assuming `orders` is a single item)
			WithMyElevator(myID, func(e *common.Elevator) {
				for f := 0; f < common.N_FLOORS; f++ {
					for b := 0; b < 2; b++ {
						e.Requests[f][b] = orders[f][b]
					}
				}
			})

		case <-DoorCloseChan:
			e := GetMyElevator(myID)
			next := ChooseDirection(e, e.Dirn)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = next.Dirn
				e.Behaviour = next.Behaviour
			})
			if next.Behaviour == common.MOVING {
				elevio.SetMotorDirection(next.Dirn)
			}
		}
	}
}

func handleButtonPress(myID string, btn elevio.ButtonEvent, prevDirn *elevio.Dirn) {
	if btn.Button == elevio.BT_Cab {
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Requests[btn.Floor][btn.Button] = true
		})
		// backup.SaveCabRequests(GetMyElevator(myID))
		UpdateCabLights(GetMyElevator(myID))
	} else {
		AssignerInput <- AssignerMsg{Data: btn}
	}
}

func openDoor(myID string) {
	WithMyElevator(myID, func(e *common.Elevator) {
		e.Behaviour = common.DOOR_OPEN
	})
	DoorOpenChan <- struct{}{}
}

func PrintElevatorState(myID string) {

	e := GetMyElevator(myID)
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
