package requests

import (
	el "Driver-go/elevator"
	eio "Driver-go/elevio"
	eiod "Driver-go/elevator_io_device"
)

// ---------------------------------h.file---------------------------------
type DirnBehaviourPair struct {
	Dirn      eio.Dirn
	Behaviour el.ElevatorBehaviour
}


///LORIES REQUEST
type Request struct {
	FloorButton eio.ButtonEvent
	HandledBy int
}


//func ChooseDirection(e elevator.Elevator) DirnBehaviourPair
//func requestShouldStop(e elevator.Elevator) bool
//func request_shouldClearImmediatley(e elevator.Elevator, btnFloor int, btnType elevator.Button) bool
//func request_ClearAtCurrentFloor(e elevator.Elevator) elevator.Elevator

//------------------------------------------------------------------------

func requestsAbove(e el.Elevator) bool {
	for f := e.Floor + 1; f < eiod.N_FLOORS; f++ {
		for btn := 0; btn < eiod.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e el.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < eiod.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e el.Elevator) bool {
	for btn := 0; btn < eiod.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {type Dirn int
			return true
		}
	}
	return false
}

func ChooseDirection(e el.Elevator) DirnBehaviourPair {

	switch e.Dirn {

	case eio.MD_Up:

		if requestsAbove(e) {
			return DirnBehaviourPair{eio.MD_Up, el.MOVING}
		}
		if requestsHere(e) {
			return DirnBehaviourPair{eio.MD_Down, el.DOOR_OPEN}
		}
		if requestsBelow(e) {
			return DirnBehaviourPair{eio.MD_Down, el.MOVING}
		}

		return DirnBehaviourPair{eio.MD_Stop, el.IDLE}

	case eio.MD_Down:

		if requestsBelow(e) {
			return DirnBehaviourPair{eio.MD_Down, el.MOVING}
		}
		if requestsHere(e) {
			return DirnBehaviourPair{eio.MD_Up, el.DOOR_OPEN}
		}
		if requestsAbove(e) {
			return DirnBehaviourPair{eio.MD_Up, el.MOVING}
		}

		return DirnBehaviourPair{eio.MD_Stop, el.IDLE}

	case eio.MD_Stop:

		if requestsHere(e) {
			return DirnBehaviourPair{eio.MD_Stop, el.DOOR_OPEN}
		}
		if requestsAbove(e) {
			return DirnBehaviourPair{eio.MD_Up, el.MOVING}
		}
		if requestsBelow(e) {
			return DirnBehaviourPair{eio.MD_Down, el.MOVING}
		}

		return DirnBehaviourPair{eio.MD_Stop, el.IDLE}

	default:
		return DirnBehaviourPair{eio.MD_Stop, el.IDLE}

	}
}

func requestShouldStop(e el.Elevator) bool {
	switch e.Dirn {

	case eio.MD_Down:

		return e.Requests[e.Floor][eio.BT_HallDown] ||
			e.Requests[e.Floor][eio.BT_Cab] ||
			!requestsBelow(e)

	case eio.MD_Up:
		return e.Requests[e.Floor][eio.BT_HallUp] ||
			e.Requests[e.Floor][eio.BT_Cab] ||
			!requestsAbove(e)

	case eio.MD_Stop:
		return true

	default:
		return true
	}
}

func ShouldClearImmediatley(e el.Elevator, btnFloor int, btnType eio.ButtonType) bool {
	switch e.ClearRequestVariant {
	//CV stands for Clear Variant
	//CV_All: clear all requests at a specific floor regardless of direction?
	//CV_InDirn: clear requests in the direction of the elevator
	case el.CV_All:
		return e.Floor == btnFloor

	case el.CV_InDirn:
		return e.Floor == btnFloor &&
			((e.Dirn == eio.MD_Up && btnType == eio.BT_HallUp) ||
				(e.Dirn == eio.MD_Down && btnType == eio.BT_HallDown) ||
				e.Dirn == eio.MD_Stop ||
				btnType == eio.BT_Cab)

	default:
		return false
	}
}

func ClearAtCurrentFloor(e el.Elevator) el.Elevator {
	switch e.ClearRequestVariant {
	case el.CV_All:
		for btn := 0; btn < eiod.N_BUTTONS; btn++ {
			e.Requests[e.Floor][btn] = false
		}
	case el.CV_InDirn:
		e.Requests[e.Floor][eio.BT_Cab] = false
		switch e.Dirn {
		case eio.MD_Up:
			if !requestsAbove(e) && !e.Requests[e.Floor][eio.BT_HallUp] {
				e.Requests[e.Floor][eio.BT_HallDown] = false
			}
			e.Requests[e.Floor][eio.BT_HallUp] = false

		case eio.MD_Down:
			if !requestsBelow(e) && !e.Requests[e.Floor][eio.BT_HallDown] {
				e.Requests[e.Floor][eio.BT_HallUp] = false
			}
			e.Requests[e.Floor][eio.BT_HallDown] = false

		case eio.MD_Stop:
			fallthrough //fallthrough means that the code will continue to the next case
		default:
			e.Requests[e.Floor][eio.BT_HallUp] = false
			e.Requests[e.Floor][eio.BT_HallDown] = false
		}
	default:
		//do nothing
	}

	return e
}

//Lorie's old code for choosing direction:
//  if e.HasRequestsAbove(e.Floor) {
//      return eio.MD_Up
//  } else if e.HasRequestsBelow(e.Floor) {
//      return eio.MD_Down
//  }
//  return eio.MD_Stop
// }

//if we want to use this function it can replace requestHere.
// func ShouldStop(e elevator.Elevator) bool {
//  return e.ShouldStop(e.Floor)
// }
