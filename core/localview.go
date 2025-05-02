package core

import (
	"elevatorlab/common"
	"encoding/json"
	"sync"
)

var mu sync.Mutex

func deepCopyElevator(e common.Elevator) common.Elevator {
	var newElevator common.Elevator
	data, _ := json.Marshal(e)
	json.Unmarshal(data, &newElevator)
	return newElevator
}

// Apply a function `fn` to an elevator of ID `myID`and saves the output back into the shared map.
func WithMyElevator(myID string, ElevSetChan chan ElevSetMsg, fn func(e *common.Elevator)) {
	mu.Lock()
	defer mu.Unlock()
	ElevSetChan <- ElevSetMsg{
		Fn: func(m map[string]common.Elevator) {
			e, exists := m[myID]
			if !exists {
				return
			}
			eCopy := deepCopyElevator(e)
			fn(&eCopy)
			m[myID] = eCopy
		},
	}
}

func GetMyElevator(myID string, ElevGetChan chan ElevGetMsg) common.Elevator {
	reply := make(chan map[string]common.Elevator)
	ElevGetChan <- ElevGetMsg{Reply: reply}
	return (<-reply)[myID]
}
