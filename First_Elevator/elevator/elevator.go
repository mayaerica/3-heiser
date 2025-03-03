package elevator

import (
	eio "Driver-go/elevio"
	"time"
)

type Elevator struct {
	Id        int 		//busy, added from lories code
	Floor     int
	Dirn eio.Dirn         // moving up, down, or emergency stop
	Behaviour ElevatorBehaviour // idle, door open, moving
	Requests  [4][3]bool        // Requests for each floor and direction (3 buttons per floor)
	Busy	  bool 				//Busy, added from lories code

	DoorOpenDuration time.Duration
	ClearRequestVariant ClearRequestVariant
}

// Elevator's states

type ElevatorBehaviour int
const (
	IDLE     ElevatorBehaviour = 0
	DOOR_OPEN = 1
	MOVING    = 2
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


type ClearRequestVariant int 

const (
	CV_All = 0
	CV_InDirn = 1
)