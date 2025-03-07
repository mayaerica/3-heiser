package requests

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
)

type DirnBehaviourPair struct {
	Dirn      elevio.Dirn
	Behaviour elevator.ElevatorBehaviour
}

type Request struct {
	FloorButton elevio.ButtonEvent
	HandledBy   int
}

func requestsAbove(e elevator.Elevator) bool {
	for f := e.Floor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e elevator.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e elevator.Elevator) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func ChooseDirection(e elevator.Elevator) DirnBehaviourPair {
	switch e.Dirn {
	case elevio.MD_Up:
		if requestsAbove(e) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.MOVING}
		}
		if requestsHere(e) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.DOOR_OPEN}
		}
		if requestsBelow(e) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.MOVING}
		}
		return DirnBehaviourPair{elevio.MD_Stop, elevator.IDLE}

	case elevio.MD_Down:
		if requestsBelow(e) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.MOVING}
		}
		if requestsHere(e) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.DOOR_OPEN}
		}
		if requestsAbove(e) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.MOVING}
		}
		return DirnBehaviourPair{elevio.MD_Stop, elevator.IDLE}

	case elevio.MD_Stop:
		if requestsHere(e) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.DOOR_OPEN}
		}
		if requestsAbove(e) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.MOVING}
		}
		if requestsBelow(e) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.MOVING}
		}
		return DirnBehaviourPair{elevio.MD_Stop, elevator.IDLE}

	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.IDLE}

	}
}

func RequestShouldStop(e elevator.Elevator) bool {
	switch e.Dirn {
	case elevio.MD_Down:
		return e.Requests[e.Floor][elevio.BT_HallDown] ||
			e.Requests[e.Floor][elevio.BT_Cab] ||
			!requestsBelow(e)

	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] ||
			e.Requests[e.Floor][elevio.BT_Cab] ||
			!requestsAbove(e)

	case elevio.MD_Stop:
		return true

	default:
		return true
	}
}

func ShouldClearImmediatley(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) bool {
	switch e.ClearRequestVariant {
	case elevator.CV_All:
		return e.Floor == btnFloor

	case elevator.CV_InDirn:
		return e.Floor == btnFloor &&
			((e.Dirn == elevio.MD_Up && btnType == elevio.BT_HallUp) ||
				(e.Dirn == elevio.MD_Down && btnType == elevio.BT_HallDown) ||
				e.Dirn == elevio.MD_Stop ||
				btnType == elevio.BT_Cab)

	default:
		return false
	}
}

func ClearAtCurrentFloor(e elevator.Elevator) elevator.Elevator {
	switch e.ClearRequestVariant {
	case elevator.CV_All:
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			e.Requests[e.Floor][btn] = false
		}
	case elevator.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dirn {
		case elevio.MD_Up:
			if !requestsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
				e.HallCalls[e.Floor][elevio.BT_HallDown] = false
				e.Done[e.Floor][elevio.BT_HallDown] = true
			}
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			e.HallCalls[e.Floor][elevio.BT_HallUp] = false
			e.Done[e.Floor][elevio.BT_HallUp] = true

		case elevio.MD_Down:
			if !requestsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
				e.HallCalls[e.Floor][elevio.BT_HallUp] = false
				e.Done[e.Floor][elevio.BT_HallUp] = true
			}
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			e.HallCalls[e.Floor][elevio.BT_HallDown] = false
			e.Done[e.Floor][elevio.BT_HallDown] = true

		case elevio.MD_Stop:
			fallthrough
		default:
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			e.Requests[e.Floor][elevio.BT_HallDown] = false

			e.HallCalls[e.Floor][elevio.BT_HallUp] = false
			e.HallCalls[e.Floor][elevio.BT_HallDown] = false
			e.Done[e.Floor][elevio.BT_HallUp] = true
			e.Done[e.Floor][elevio.BT_HallDown] = true

		}
	default:
	}
	return e
}
