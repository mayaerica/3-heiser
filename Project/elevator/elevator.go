package elevator

import (
	"elevatorlab/elevio"
	"time"
)

const N_FLOORS int = 4
const N_BUTTONS int = 3

type Elevator struct {
	Id        int 		
	Floor     int
	Dirn      elevio.Dirn         // moving up, down, or emergency stop
	Behaviour ElevatorBehaviour // idle, door open, moving
	Requests  [4][3]bool        // Requests for each floor and direction (3 buttons per floor)
	HallCalls [4][2]bool   		//Which HallButtons have been made. Used to sync all elevators
	Busy	  bool 				//Busy, added from lories code. Dont think we use this anymore -Magnus
	Done [4][2]bool 			//List for done requests so that all elevators can turn off lights

	DoorOpenDuration time.Duration
	ClearRequestVariant ClearRequestVariant
}

// Elevator's states

type ElevatorBehaviour int
const (
	IDLE       ElevatorBehaviour = 0
	DOOR_OPEN 				     = 1
	MOVING                       = 2
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