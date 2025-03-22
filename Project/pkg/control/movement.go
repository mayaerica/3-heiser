package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
)

// checking if should stop at current floor
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

func StopElevator() {
	elevio.SetMotorDirection(elevio.MD_Stop)
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

func ClearRequestsAtCurrentFloor(e *common.Elevator) {
	floor := e.Floor
	dirn := e.Dirn

	switch e.ClearRequestVariant {
	case common.CV_All:
		for btn := 0; btn < common.N_BUTTONS; btn++ {
			e.Requests[floor][btn] = false
		}
	case common.CV_InDirn:
		e.Requests[floor][elevio.BT_Cab] = false

		switch dirn {
		case elevio.MD_Up:
			e.Requests[floor][elevio.BT_HallUp] = false
		case elevio.MD_Down:
			e.Requests[floor][elevio.BT_HallDown] = false
		case elevio.MD_Stop:
			e.Requests[floor][elevio.BT_HallUp] = false
			e.Requests[floor][elevio.BT_HallDown] = false
		}
	}

	backup.SaveCabRequests(*e)
}

func ShouldClearImmediately(e common.Elevator, btnFloor int, btnType elevio.ButtonType) bool{
	if e.Floor != btnFloor {
		return false
	}

	switch e.ClearRequestVariant{
	case common.CV_All:
		return true
	
	case common.CV_InDirn:
		return btnType == elevio.BT_Cab ||
			e.Dirn == elevio.MD_Stop ||
			(e.Dirn == elevio.MD_Up && btnType == elevio.BT_HallUp) ||
			(e.Dirn == elevio.MD_Down && btnType == elevio.BT_HallDown)
	default:
		return false
	}
}