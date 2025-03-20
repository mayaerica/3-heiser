package utils

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)


func BehaviourToString(b common.ElevatorBehaviour) string {
	switch b{
	case common.IDLE:
		return "IDLE"
	case common.MOVING:
		return "MOVING"
	case common.DOOR_OPEN:
		return "DOOR_OPEN"
	default:
		return "UNKNOWN"
	}
}

func DirectionToString(d elevio.Dirn) string {
	switch d {
	case elevio.MD_Stop:
		return "STOP"
	case elevio.MD_Up:
		return "UP"
	case elevio.MD_Down:
		return "DOWN"
	default:
		return "UNKNOWN"
	}
}

