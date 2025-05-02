package conversions

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

func BehaviourToString(b common.ElevatorBehaviour) string {
	switch b {
	case common.IDLE:
		return "idle"
	case common.MOVING:
		return "moving"
	case common.DOOR_OPEN:
		return "doorOpen"
	default:
		return "unknown"
	}
}

func DirectionToString(d elevio.Dirn) string {
	switch d {
	case elevio.MD_Stop:
		return "stop"
	case elevio.MD_Up:
		return "up"
	case elevio.MD_Down:
		return "down"
	default:
		return "unknown"
	}
}
