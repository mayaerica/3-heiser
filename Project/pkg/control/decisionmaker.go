package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

// Determine whether to stop at current floor

// Determine whether to stop at current floor
func RequestShouldStop(e common.Elevator) bool {
	f := e.Floor

	switch e.Dirn {
	case elevio.MD_Down:
		return e.Requests[f][elevio.BT_HallDown] ||
			e.Requests[f][elevio.BT_Cab] ||
			!RequestsBelow(e) || f == 0
	case elevio.MD_Up:
		return e.Requests[f][elevio.BT_HallUp] ||
			e.Requests[f][elevio.BT_Cab] ||
			!RequestsAbove(e) || f == common.N_FLOORS-1
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

func clearAtCurrentFloor(e common.Elevator, orderComplete chan elevio.ButtonEvent) common.Elevator {
	switch e.ClearRequestVariant {

	case common.CV_All:
		for btn := 0; btn < elevio.N_BUTTONS; btn++ {
			e.Requests[e.Floor][btn] = false
		}

	case common.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		UpdateCabLights(e)

		switch e.Dirn {
		case elevio.MD_Up:
			if !e.HasRequestsAbove(e.Floor) && !e.Requests[e.Floor][elevio.BT_HallUp] && e.Requests[e.Floor][elevio.BT_HallDown] {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
				orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
			}
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}

		case elevio.MD_Down:
			if !e.HasRequestsBelow(e.Floor) && !e.Requests[e.Floor][elevio.BT_HallDown] && e.Requests[e.Floor][elevio.BT_HallUp] {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
				orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
			}
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}

		case elevio.MD_Stop:
			if e.Requests[e.Floor][elevio.BT_HallUp] {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
				orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
			}
			if e.Requests[e.Floor][elevio.BT_HallDown] {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
				orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
			}
			// e.Requests[e.Floor][elevio.BT_HallUp] = false
			// orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
			// e.Requests[e.Floor][elevio.BT_HallDown] = false
			// orderComplete <- elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
		}
	}
	return e
}

func ShouldClearImmediately(e common.Elevator, btnFloor int, btnType elevio.ButtonType) bool {
	if e.Floor != btnFloor {
		return false
	}

	switch e.ClearRequestVariant {
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

func ChooseDirection(e common.Elevator, prevDirn elevio.Dirn) common.DirnBehaviourPair {
	switch prevDirn {
	case elevio.MD_Up:
		if RequestsAbove(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.MOVING}
		} else if RequestsHere(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.DOOR_OPEN}
		} else if RequestsBelow(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.MOVING}
		}
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
	case elevio.MD_Down:
		if RequestsBelow(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.MOVING}
		} else if RequestsHere(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.DOOR_OPEN}
		} else if RequestsAbove(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.MOVING}
		}
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
	case elevio.MD_Stop:
		if RequestsHere(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.DOOR_OPEN}
		} else if RequestsAbove(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: common.MOVING}
		} else if RequestsBelow(e) {
			return common.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: common.MOVING}
		}
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
	default:
		return common.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: common.IDLE}
	}
}
