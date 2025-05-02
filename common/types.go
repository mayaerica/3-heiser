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
	NOT_SEEN OrderState = iota
	SEEN_BY_SOMEONE
	SEEN_BY_EVERYONE
	UNCERTAIN
)

type Perspective struct {
	ID          string
	Perspective [N_FLOORS][2]OrderState
	CabCalls    [N_FLOORS][1]OrderState
}

type ClearRequestVariant int

const (
	CV_All ClearRequestVariant = iota
	CV_InDirn
)

func (e *Elevator) ShouldStop(floor int) bool {
	for btn := 0; btn < 3; btn++ {
		if e.Requests[floor][btn] {
			return true
		}
	}
	return false
}

func (e *Elevator) ClearRequestsAtFloor(floor int) {
	for btn := 0; btn < 3; btn++ {
		e.Requests[floor][btn] = false
	}
}

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
