package control

import (
	"elevatorlab/common"
	"sync"
)

var (
	localElevator common.Elevator
	localMutex	  sync.RWMutex
)

func SetLocalElevator(e common.Elevator) {
	localMutex.Lock()
	defer localMutex.Unlock()
	localElevator = e
}

//returns a copy of the elevator state
func GetLocalElevator() common.Elevator {
	localMutex.RLock()
	defer localMutex.RUnlock()
	return localElevator
}

//modify elevator in-place via callback
func UpdateLocalElevator(fn func(e *common.Elevator)){
	localMutex.Lock()
	defer localMutex.Unlock()
	fn(&localElevator)
}