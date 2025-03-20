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
	ClearRequestVariant ClearRequestVariant //could this just be "int"?
	DoorOpenDuration    time.Duration       //could this just be "int"?
}

// this will be used for HRA communication as well!
type DirnBehaviourPair struct {
	Dirn      elevio.Dirn
	Behaviour ElevatorBehaviour
}

type OrderState int

const (
	NonExisting OrderState = iota
	HalfExisting
	Existing
	Unknown
)

type Perspective struct {
	ID string
	Perspective [N_FLOORS][2]OrderState
}

// global state for tracking
var GlobalPerspective Perspective

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

type ClearRequestVariant int

const (
	CV_All    = 0
	CV_InDirn = 1
)
