package fsm

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
	"elevatorlab/requests"
	"fmt"
	"time"
)

var Elevator elevator.Elevator

func setAllLights(e elevator.Elevator) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			Button := elevio.ButtonType(btn)
			if Button == elevio.BT_Cab{ //Sets hall button based on HallCalls 
				elevio.SetButtonLamp(Button, floor, elevio.ToBool(elevio.ToByte(e.Requests[floor][btn])))
			} else { //Sets cab button based on requests
				elevio.SetButtonLamp(Button, floor, elevio.ToBool(elevio.ToByte(e.HallCalls[floor][btn])))
			}
			
		}
	}
} 

//func OnInitBetweenFloors() {
//	elevio.SetMotorDirection(elevio.MD_Down)
//	Elevator.Dirn = elevio.MD_Down
//	Elevator.Behaviour = elevator.MOVING
//}

func OnRequestButtonPress(btn_floor int, btn_type elevio.ButtonType, timer_start chan time.Duration) {

	switch Elevator.Behaviour {
	case elevator.DOOR_OPEN:
		if requests.ShouldClearImmediatley(Elevator, btn_floor, btn_type) { // If the request should be cleared immediately
			timer_start <- Elevator.DoorOpenDuration // Start the door timer
		} else {
			Elevator.Requests[btn_floor][btn_type] = true // Set the request
		}
		break

	case elevator.MOVING:
		Elevator.Requests[btn_floor][btn_type] = true // Set the request
		break

	case elevator.IDLE:
		Elevator.Requests[btn_floor][btn_type] = true                            // Set the request
		var pair requests.DirnBehaviourPair = requests.ChooseDirection(Elevator) //puts directions into the DirnBehaviourPair struct "pair"
		Elevator.Dirn = pair.Dirn
		Elevator.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.DOOR_OPEN:
			elevio.SetDoorOpenLamp(true)
			timer_start <- Elevator.DoorOpenDuration
			Elevator = requests.ClearAtCurrentFloor(Elevator)

		case elevator.MOVING:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(Elevator.Dirn)

		case elevator.IDLE:
			elevio.SetDoorOpenLamp(false)
		}
		break

	}
	setAllLights(Elevator)
}

func OnFloorArrival(newFloor int, timer_start chan time.Duration) {
	Elevator.Floor = newFloor
	elevio.SetFloorIndicator(Elevator.Floor)

	switch Elevator.Behaviour {
	case elevator.MOVING:
		if requests.RequestShouldStop(Elevator) {
			elevio.SetMotorDirection(elevio.MD_Stop) // Stop the elevator
			elevio.SetDoorOpenLamp(true)             // turns on the light
			Elevator = requests.ClearAtCurrentFloor(Elevator)
			timer_start <- Elevator.DoorOpenDuration // Start the door timer
			setAllLights(Elevator)
			Elevator.Behaviour = elevator.DOOR_OPEN
		}
	default:
		break
	}
}

func OnDoorTimeout(timer_start chan time.Duration) {
	switch Elevator.Behaviour {

	case elevator.DOOR_OPEN:

		if elevio.GetObstruction() == true {
			fmt.Println("obstruction") //this is the wrong solution :)
			break
		}

		elevio.SetDoorOpenLamp(false)

		var pair requests.DirnBehaviourPair
		pair = requests.ChooseDirection(Elevator)
		Elevator.Dirn = pair.Dirn
		Elevator.Behaviour = pair.Behaviour

		switch Elevator.Behaviour {
		case elevator.DOOR_OPEN:
			timer_start <- Elevator.DoorOpenDuration
			Elevator = requests.ClearAtCurrentFloor(Elevator)
			setAllLights(Elevator)
			break

		case elevator.MOVING:
			elevio.SetMotorDirection(Elevator.Dirn)
			elevio.SetDoorOpenLamp(false)
		case elevator.IDLE:
			elevio.SetDoorOpenLamp(true)
			timer_start <- Elevator.DoorOpenDuration
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(Elevator.Dirn)
			break
		}

		break
	default:
		break

	}
}
