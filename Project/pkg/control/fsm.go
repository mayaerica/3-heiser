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
	StateChan            = make(chan common.ElevatorBehaviour) // Tell the FSM to switch state (IDLE, MOVING, DOOR_OPEN)
	DoorOpenChan         = make(chan struct{})                 // Tell the door to open
	DoorCloseChan        = make(chan struct{})                 // Door has closed, resume FSM
	AssignedHallCallChan = make(chan elevio.ButtonEvent)       // Assigner tells FSM: "You're responsible for this hall call"
	OrderCompleteChan    = make(chan elevio.ButtonEvent)       // FSM tells assigner: "I completed this request"
)

func InitFSM(myID string, initial common.Elevator) {
	// Load any saved cab calls from disk (in case of crash recovery)
	backup.LoadCabRequests(&initial)

	// Register our local elevator state in the shared state map
	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}

	// Start FSM components
	go StateMachineLoop(myID)                                         // Controls what to do in each state
	go executionLoop(myID)                                            // Reacts to events: buttons, floor sensor, assignments
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration) // Door opens and closes
}

func StateMachineLoop(myID string) {
	for state := range StateChan {
		// Update our elevator's behavior state (IDLE, MOVING, DOOR_OPEN)
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Behaviour = state
		})
		// Decide what to do now that the behavior has changed
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
		// A user pressed a button (cab or hall)
		case btn := <-buttonPressChan:
			handleButtonPress(myID, btn, &prevDirn)

		// Received a hall call assignment from the assigner
		case assigned := <-AssignedHallCallChan:
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Requests[assigned.Floor][assigned.Button] = true
			})
			UpdateCabLights(GetMyElevator(myID))

			// If we’re idle, start moving or open the door immediately
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
		//Elevator arrived at a new floor
		case floor := <-floorSensorChan:
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = floor
			})
			elevio.SetFloorIndicator(floor)

			// Decide whether we should stop and open the door
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

		// Door closed after timeout — pick next action
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
	fmt.Printf("[BTN] Button pressed: floor=%d, type=%v\n", btn.Floor, btn.Button)

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
			})

			switch pair.Behaviour {
			case common.DOOR_OPEN:
				fmt.Println("[IDLE] Opening door immediately at current floor.")
				ClearRequestsAtCurrentFloor(myID)
				UpdateCabLights(GetMyElevator(myID))
				elevio.SetDoorOpenLamp(true)
				StateChan <- common.DOOR_OPEN
				DoorOpenChan <- struct{}{}

			case common.MOVING:
				fmt.Println("[IDLE] Starting to move.")
				elevio.SetMotorDirection(pair.Dirn)
				StateChan <- common.MOVING
				*prevDirn = pair.Dirn

			case common.IDLE:
				fmt.Println("[IDLE] No direction chosen. Staying idle.")
				StateChan <- common.IDLE
			}

		case common.MOVING:
			// Cab call will be handled later during movement
		}

	case elevio.BT_HallUp, elevio.BT_HallDown:
		// Forward hall calls to assigner to let it decide who handles it
		fmt.Println("[HALL] Forwarding hall request to assigner.")
		AssignerInput <- AssignerMsg{Type: "hall_call", Data: btn}
		AssignerInput <- AssignerMsg{Type: "assign", Data: btn}
	}
}

// Reaction when StesteChan is updated
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
				<-DoorCloseChan // Wait for door to fully close before moving again
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
