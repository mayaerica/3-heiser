package common

import (
	"elevatorlab/elevio"
	"time"
)

const N_FLOORS = elevio.N_FLOORS
const N_BUTTONS = elevio.N_BUTTONS

type ElevatorBehaviour int

const (
	IDLE ElevatorBehaviour = iota
	MOVING
	DOOR_OPEN
)

type Elevator struct {
	ID                  string
	Floor               int
	Dirn                elevio.Dirn
	Behaviour           ElevatorBehaviour
	Requests            [N_FLOORS][N_BUTTONS]bool
	ClearRequestVariant ClearRequestVariant
	DoorOpenDuration    time.Duration
}

type DirnBehaviourPair struct {
	Dirn      elevio.Dirn
	Behaviour ElevatorBehaviour
}

type OrderState int

const (
	NotSeen OrderState = iota          // No elevator has seen this call
	SeenBySomeone                      // At least one elevator has seen it
	SeenByEveryone                     // All elevators agree the call exists
	Uncertain                          
)

type Perspective struct {
	ID          string
	Perspective [N_FLOORS][2]OrderState
}

type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDirn
)

// Checks if the elevator should stop at the given floor based on requests.
func (e *Elevator) ShouldStop(floor int) bool {
	// Iterate through the three possible buttons for this floor (up, down, or internal request)
	for btn := 0; btn < 3; btn++ {
		if e.Requests[floor][btn] {
			return true
		}
	}
	return false
}

// Clears the requests for the given floor.
func (e *Elevator) ClearRequestsAtFloor(floor int) {
	for btn := 0; btn < 3; btn++ {
		e.Requests[floor][btn] = false
	}
}

// Checks if there are any requests for floors above the current one.
func (e *Elevator) HasRequestsAbove(floor int) bool {
	for f := floor + 1; f < 4; f++ {
		for btn := 0; btn < 3; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

// Checks if there are any requests for floors below the current one.
func (e *Elevator) HasRequestsBelow(floor int) bool {
	for f := 0; f < floor; f++ {
		for btn := 0; btn < 3; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}
