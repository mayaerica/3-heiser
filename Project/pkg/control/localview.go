package control

/*
Used to modify local elevator's state inside the plantstate map.
This will help avoid repeating code.
*/

import (
	"elevatorlab/common"
)

// To safely update local elevator in the shared state map
// via a callback.
func WithMyElevator(myID string, fn func(e *common.Elevator)) {
	ElevSet <- ElevSetMsg {
		Fn : func(m map[string]common.Elevator) {
			e := m[myID]
			fn(&e)
			m[myID] = e
		},
	}
}

// reads and returns the current state of local elevator.
func GetMyElevator(myID string) common.Elevator {
	reply := make(chan map[string]common.Elevator)
	ElevGet <- ElevGetMsg{Reply: reply}
	return (<-reply)[myID]
}
