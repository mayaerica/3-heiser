package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)


//checking if should stop at current floor
func RequestShouldStop(e common.Elevator) bool {
	switch e.Dirn {
	case elevio.MD_Down:
		return e.Requests[Elevator.Floor][elevio.BT_HallDown] ||
			e.Requests[Elevator.Floor][elevio.BT_Cab] ||
			!RequestsBelow(e)

	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] ||
			e.Requests[e.Floor][elevio.BT_Cab] ||
			!RequestsAbove(e)

	default:
		return true
	}
}


func RequestsAbove(e common.Elevator) bool {
	for f := e.Floor + 1; f < common.N_FLOORS; f++ {
		for btn := 0; btn < common.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e common.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < common.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e common.Elevator) bool {
	for btn := 0; btn < common.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}
