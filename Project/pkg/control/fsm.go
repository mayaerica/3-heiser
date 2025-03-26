package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"fmt"
	"time"
)

// Channels used in the FSM system
var (
	//StateChan            = make(chan common.ElevatorBehaviour, 1) // Tell the FSM to switch state (IDLE, MOVING, DOOR_OPEN)
	DoorOpenChan         = make(chan struct{})           // Tell the door to open
	DoorCloseChan        = make(chan struct{})           // Door has closed, resume FSM
	AssignedHallCallChan = make(chan elevio.ButtonEvent) // Assigner tells FSM: "You're responsible for this hall call"
	OrderCompleteChan    = make(chan elevio.ButtonEvent) // FSM tells assigner: "I completed this request"
)

func InitFSM(myID string, initial common.Elevator) {
	// Load any saved cab calls from disk (in case of crash recovery)
	backup.LoadCabRequests(&initial)

	// Register our local elevator state in the shared state map
	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}

	// Start FSM components
	//go StateMachineLoop(myID)                                         // Controls what to do in each state
	go StateMachineLoop(myID)                                         // Reacts to events: buttons, floor sensor, assignments
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration) // Door opens and closes
	//go PrintElevatorState(myID)
}

//func StateMachineLoop(myID string) {
//	for state := range StateChan {
// Update our elevator's behavior state (IDLE, MOVING, DOOR_OPEN)
//		WithMyElevator(myID, func(e *common.Elevator) {
//			e.Behaviour = state
//		})
// Decide what to do now that the behavior has changed
//		handleState(myID)
//	}
//}

func StateMachineLoop(myID string) {
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)

	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	var prevDirn elevio.Dirn = elevio.MD_Stop

	fmt.Print("Entered executionloop")
	for {
		select {
		// A user pressed a button (cab or hall)
		case btn := <-buttonPressChan:
			fmt.Println("Buttonpress received", btn)
			handleButtonPress(myID, btn, &prevDirn)

		// Received a hall call assignment from the assigner
		case assigned := <-AssignedHallCallChan:

			fmt.Println("Assingment received", assigned)

			WithMyElevator(myID, func(e *common.Elevator) {
				e.Requests[assigned.Floor][assigned.Button] = true
			})

			// If we’re idle, start moving or open the door immediately
			e := GetMyElevator(myID)
			next := ChooseDirection(e, e.Dirn)

			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = next.Dirn
				e.Behaviour = next.Behaviour
			})

			if next.Dirn == elevio.MD_Stop {
				fmt.Println(("Assigned call is at current floor"))

				ClearRequestsAtCurrentFloor(myID)

				OrderCompleteChan <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
				OrderCompleteChan <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}

				DoorOpenChan <- struct{}{}
				<-DoorCloseChan // Wait for door to fully close before moving again

			} else {
				fmt.Println(("Motor direction set"))
				elevio.SetMotorDirection(next.Dirn)
				prevDirn = next.Dirn
			}
			/*select {
			case StateChan <- next.Behaviour:
			default:
				fmt.Println("StateChan blocked in executionLoop (assigned)")
			}

			if next.Dirn != elevio.MD_Stop {
				prevDirn = next.Dirn
			}*/
			//}

		//Elevator arrived at a new floor
		case floor := <-floorSensorChan:
			fmt.Println("Floor sensor has been activated")
			elevio.SetFloorIndicator(floor)

			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = floor
			})

			// Decide whether we should stop and open the door
			if RequestShouldStop(GetMyElevator(myID)) {
				fmt.Println("Stopping at floor:", floor)
				StopElevator()

				WithMyElevator(myID, func(e *common.Elevator) {
					e.Behaviour = common.DOOR_OPEN
				})

				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				DoorOpenChan <- struct{}{}
				<-DoorCloseChan // Wait for door to fully close before moving again
			}

		// Door closed after timeout — pick next action
		case <-DoorCloseChan:
			fmt.Println("Door closed, choosing next action...")
			e := GetMyElevator(myID)
			next := ChooseDirection(e, prevDirn)

			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = next.Dirn
				e.Behaviour = next.Behaviour
			})

			elevio.SetMotorDirection(next.Dirn)

			//prevent blocking - added select stuff rather than just "StateChan <- someBehaviour" - under debug:
			/*select {
			case StateChan <- next.Behaviour:
			default:
				fmt.Println("StateChan blocked in executionLoop (door closed)")
			}*/

			if next.Dirn != elevio.MD_Stop {
				prevDirn = next.Dirn
			}
		}
	}
}

func handleButtonPress(myID string, btn elevio.ButtonEvent, prevDirn *elevio.Dirn) {
	switch btn.Button {
	case elevio.BT_Cab:
		// Cab calls are local and stored immediately
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Requests[btn.Floor][elevio.BT_Cab] = true
		})
		backup.SaveCabRequests(GetMyElevator(myID))
		UpdateCabLights(GetMyElevator(myID))
		// React based on current state

		switch GetMyElevator(myID).Behaviour {
		case common.DOOR_OPEN:
			// If we're at the same floor and it's clearable, refresh door timer
			if elevio.GetFloor() == btn.Floor && ShouldClearImmediately(GetMyElevator(myID), btn.Floor, btn.Button) {
				fmt.Println("[DOOR_OPEN] Clearing cab request at current floor.")
				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				DoorOpenChan <- struct{}{}
			}

		case common.IDLE:
			// Start elevator activity immediately
			pair := ChooseDirection(GetMyElevator(myID), *prevDirn)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = pair.Dirn
				e.Behaviour = pair.Behaviour
			})
			switch pair.Behaviour {
			case common.DOOR_OPEN:
				fmt.Println("[IDLE] Opening door immediately at current floor.")
				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				DoorOpenChan <- struct{}{}

				//elevio.SetDoorOpenLamp(true)
				//prevent blocking - added select stuff rather than just "StateChan <- someBehaviour" - under debug:
				//select {
				//case StateChan <- common.DOOR_OPEN:
				//default:
				//	fmt.Print("StateChan blocked in handleButtonPress, case dooropen")
				//}

			case common.MOVING:
				fmt.Println("[IDLE] Starting to move.")
				elevio.SetMotorDirection(pair.Dirn)
				*prevDirn = pair.Dirn

				//select {
				//case StateChan <- common.DOOR_OPEN:
				//default:
				//	fmt.Print("StateChan blocked in handleButtonPress, case moving")
				//}

			case common.IDLE:
				fmt.Println("[IDLE] No direction chosen. Staying idle.")

				/*case StateChan <- common.IDLE:
				default:
					fmt.Print("StateChan blocked in handleButtonPress, case idle")
				}*/
			}
		}

	case elevio.BT_HallUp, elevio.BT_HallDown:
		// Forward hall calls to assigner to let it decide who handles it
		fmt.Println("[HALL] Forwarding hall request to assigner.")

		AssignerInput <- AssignerMsg{Type: "hall_call", Data: btn}
		AssignerInput <- AssignerMsg{Type: "assign", Data: btn}
	}
}

func PrintElevatorState(myID string) {

	for {
		e := GetMyElevator(myID)
		time.Sleep(250 * time.Millisecond)
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
}

/* Reaction when StesteChan is updated
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

// Elevator just entered IDLE, check if we should do anything
func handleIdleState(myID string) {
	e := GetMyElevator(myID)
	next := ChooseDirection(e, e.Dirn)
	WithMyElevator(myID, func(e *common.Elevator) {
		e.Dirn = next.Dirn
		e.Behaviour = next.Behaviour
	})
	fmt.Println("help")
	fmt.Println(1, next.Dirn)
	elevio.SetMotorDirection(next.Dirn)
	//prevent blocking - added select stuff rather than just "StateChan <- someBehaviour" - under debug:

}

func handleMovingState(myID string) {
	newFloor := elevio.GetFloor()
	if newFloor != -1 {
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Floor = newFloor
		})
		e := GetMyElevator(myID)

		next := ChooseDirection(e, e.Dirn)
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Dirn = next.Dirn
			e.Behaviour = next.Behaviour
		})

		/*if RequestShouldStop(GetMyElevator(myID)) {
			StopElevator()
			DoorOpenChan <- struct{}{}
			<-DoorCloseChan // Wait for door to fully close before moving again
			ClearRequestsAtCurrentFloor(myID)
			UpdateCabLights(GetMyElevator(myID))
			//prevent blocking - added select stuff rather than just "StateChan <- someBehaviour" - under debug:
			select {
			case StateChan <- common.IDLE:
			default:
				fmt.Println("StateChan blocked in handleMovingState")
			}
			return
		}
	}
	time.Sleep(50 * time.Millisecond)

}*/
