package fsm

import (
	el "Driver-go/elevator"
	eiod "Driver-go/elevator_io_device"
	eio "Driver-go/elevio"
	req "Driver-go/requests"
	//timer "Driver-go/timer"
	"fmt"
	"time"
)

//----------------------- h fil -----------------------

// ----------------- FSM struct -----------------

var Elevator el.Elevator

//var F = FSM{Elevator: el.Elevator{Floor: 1, DoorsOpen: false}}

//func GetFSM() *FSM {
//    return &globalFSM
//}

// ---------------------------------------
func setAllLights(e el.Elevator) {
	for floor := 0; floor < eiod.N_FLOORS; floor++ {
		for btn := 0; btn < eiod.N_BUTTONS; btn++ {
			eio.SetButtonLamp(eio.ButtonType(btn), floor, eio.ToBool(eio.ToByte(e.Requests[floor][btn])))
		}
	}
}

//func OnInitBetweenFloors() {
//	eio.SetMotorDirection(eio.MD_Down)
//	Elevator.Dirn = eio.MD_Down
//	Elevator.Behaviour = el.MOVING
//}

func OnRequestButtonPress(btn_floor int, btn_type eio.ButtonType, timer_start chan time.Duration) {

	switch Elevator.Behaviour {
	case el.DOOR_OPEN:
		if req.ShouldClearImmediatley(Elevator, btn_floor, btn_type) { // If the request should be cleared immediately
			timer_start <- Elevator.DoorOpenDuration // Start the door timer
		} else {
			Elevator.Requests[btn_floor][btn_type] = true // Set the request
		}
		break

	case el.MOVING:
		Elevator.Requests[btn_floor][btn_type] = true // Set the request
		break

	case el.IDLE: // If the elevator is idle
		Elevator.Requests[btn_floor][btn_type] = true // Set the request
		var pair req.DirnBehaviourPair
		pair = req.ChooseDirection(Elevator) //puts directions into the DirnBehaviourPair struct "pair"
		Elevator.Dirn = pair.Dirn
		Elevator.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case el.DOOR_OPEN:
			eio.SetDoorOpenLamp(true)
			timer_start <- Elevator.DoorOpenDuration
			Elevator = req.ClearAtCurrentFloor(Elevator)

		case el.MOVING:
			eio.SetDoorOpenLamp(false)
			eio.SetMotorDirection(Elevator.Dirn)

		case el.IDLE:
			eio.SetDoorOpenLamp(false)
		}
		break

	}

	setAllLights(Elevator)

}

func OnFloorArrival(newFloor int, timer_start chan time.Duration) {
	Elevator.Floor = newFloor
	eio.SetFloorIndicator(Elevator.Floor)

	switch Elevator.Behaviour {
	case el.MOVING:
		if Elevator.ShouldStop(newFloor) {
			eio.SetMotorDirection(eio.MD_Stop) // Stop the elevator
			eio.SetDoorOpenLamp(true)          // turns on the light
			Elevator = req.ClearAtCurrentFloor(Elevator)
			timer_start <- Elevator.DoorOpenDuration // Start the door timer
			setAllLights(Elevator)
			Elevator.Behaviour = el.DOOR_OPEN
		}
	default:
		break
	}
}

func OnDoorTimeout(timer_start chan time.Duration) {
	switch Elevator.Behaviour {

	case el.DOOR_OPEN:
		
		if eio.GetObstruction()==true{
			fmt.Println("obstruction")	//this is the wrong solution :)
			break
		}

		eio.SetDoorOpenLamp(false)

		var pair req.DirnBehaviourPair
		pair = req.ChooseDirection(Elevator)
		Elevator.Dirn = pair.Dirn
		Elevator.Behaviour = pair.Behaviour

		switch Elevator.Behaviour {
		case el.DOOR_OPEN:
			timer_start <- Elevator.DoorOpenDuration
			Elevator = req.ClearAtCurrentFloor(Elevator)
			setAllLights(Elevator)
			break

		case el.MOVING:
			eio.SetMotorDirection(Elevator.Dirn) //there should definetly be something underneath this function
			eio.SetDoorOpenLamp(false)
		case el.IDLE:
			eio.SetDoorOpenLamp(true)
			timer_start <- Elevator.DoorOpenDuration
			eio.SetDoorOpenLamp(false)
			eio.SetMotorDirection(Elevator.Dirn)
			break
		}

		break
	default:
		break

	}
}
