package utils

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"strconv"
	"fmt"
)


func BehaviourToString(b common.ElevatorBehaviour) string {
	switch b{
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

func ConvertHRAOutput(output map[string][][2]bool) (map[int]map[int][2]bool, error) {
	convertedOutput := make(map[int]map[int][2]bool)
	for key, floorRequests := range output {
		elevatorID, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("strconv.Atoi error: %v", err)
		}
		floorMap := make(map[int][2]bool)
		for floor, buttons := range floorRequests {
			floorMap[floor] = buttons
		}
		convertedOutput[elevatorID] = floorMap
	}
	return convertedOutput, nil
}
