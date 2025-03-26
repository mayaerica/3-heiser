package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"
	"fmt"
)

// Determine whether to stop at current floor

// Determine whether to stop at current floor
func RequestShouldStop(e common.Elevator) bool {
	f := e.Floor
	/*fmt.Printf("[DEBUG] RequestShouldStop: floor=%d dir=%v cab=%v up=%v down=%v\n",
		f, e.Dirn,
		e.Requests[f][elevio.BT_Cab],
		e.Requests[f][elevio.BT_HallUp],
		e.Requests[f][elevio.BT_HallDown],
	)*/
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

// Clear requests at current floor and notify network
func ClearRequestsAtCurrentFloor(myID string) {
	WithMyElevator(myID, func(e *common.Elevator) {
		f := e.Floor
		d := e.Dirn

		switch e.ClearRequestVariant {
		case common.CV_All:
			for btn := 0; btn < common.N_BUTTONS; btn++ {
				e.Requests[f][btn] = false
				if e.Requests[f][elevio.BT_HallUp] {
					AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallUp}}
				}
				if e.Requests[f][elevio.BT_HallDown] {
					AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallDown}}
				}
			}

		case common.CV_InDirn:
			e.Requests[f][elevio.BT_Cab] = false

			switch d {
			case elevio.MD_Up:
				if !RequestsAbove(*e) {
					e.Requests[f][elevio.BT_HallDown] = false
					AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallDown}}
				}
				fmt.Println("\n\n\nSHOULD CLEAR THIS\n\n\n")
				e.Requests[f][elevio.BT_HallUp] = false
				AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallUp}}

			case elevio.MD_Down:
				if !RequestsBelow(*e) {
					e.Requests[f][elevio.BT_HallUp] = false
					AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallUp}}
				}
				e.Requests[f][elevio.BT_HallDown] = false
				AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallDown}}

			case elevio.MD_Stop:
				e.Requests[f][elevio.BT_HallUp] = false
				AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallUp}}
				e.Requests[f][elevio.BT_HallDown] = false
				AssignerInput <- AssignerMsg{Type: "complete", Data: elevio.ButtonEvent{Floor: f, Button: elevio.BT_HallDown}}
			}
		}

		// Print once the clearing process is completed
		fmt.Println("\n\n\nCOMPLETE CLEARING\n\n\n\n")

		// Save the elevator's updated request state
		backup.SaveCabRequests(*e)
	})
}


// Instant clear check for newly pressed button
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

// Choose next direction + behaviour
func ChooseDirection(e common.Elevator, prevDirn elevio.Dirn) common.DirnBehaviourPair {
	//fmt.Printf("[DEBUG] ChooseDirection: floor=%d, prevDirn=%v\n", e.Floor, prevDirn)
	//fmt.Println("RequestsHere:", RequestsHere(e))
	//fmt.Println("RequestsAbove:", RequestsAbove(e))
	//fmt.Println("RequestsBelow:", RequestsBelow(e))
	fmt.Print()

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
