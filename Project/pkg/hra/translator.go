package hra

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/utils"
)

// Types needed for JSON encoding/decoding
type HRAElevState struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [common.N_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

// Builds input for the HRA binary
func CreateHRAInput(
	elevators map[string]common.Elevator,
	hallRequests [common.N_FLOORS][2]bool,
) HRAInput {
	hraStates := make(map[string]HRAElevState)

	for id, elev := range elevators {
		hraStates[id] = HRAElevState{
			Behaviour:   utils.BehaviourToString(elev.Behaviour),
			Floor:       elev.Floor,
			Direction:   utils.DirectionToString(elev.Dirn),
			CabRequests: extractCabRequests(elev),
		}
	}

	return HRAInput{
		HallRequests: hallRequests,
		States:       hraStates,
	}
}

// Extracts only this elevator's assignment from HRA output
func HRAMapToElevator(
	assignments map[string][][2]bool,
	elev common.Elevator,
) common.Elevator {
	assigned, exists := assignments[elev.ID]
	if !exists {
		return elev
	}

	for f := 0; f < common.N_FLOORS; f++ {
		elev.Requests[f][elevio.BT_HallUp] = assigned[f][0]
		elev.Requests[f][elevio.BT_HallDown] = assigned[f][1]
	}

	return elev
}

// Helper to extract only cab requests from full request matrix
func extractCabRequests(e common.Elevator) []bool {
	cab := make([]bool, common.N_FLOORS)
	for f := 0; f < common.N_FLOORS; f++ {
		cab[f] = e.Requests[f][elevio.BT_Cab]
	}
	return cab
}
