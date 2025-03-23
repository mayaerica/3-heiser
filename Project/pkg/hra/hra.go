package hra

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/utils"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
)

type HRAElevState struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

// input format for HRA
type HRAInput struct {
	HallRequests [common.N_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

// create HRA input from elevator states and hall requests
func CreateHRAInput(states map[string]common.Elevator, hall [common.N_FLOORS][2]bool) HRAInput {
	out := HRAInput{
		HallRequests: hall,
		States:       make(map[string]HRAElevState),
	}

	for id, elev:= range states{
		out.States[id] = HRAElevState{
			Behaviour:    utils.BehaviourToString(elev.Behaviour),
			Floor:        elev.Floor,
			Direction:    utils.DirectionToString(elev.Dirn),
			CabRequests:  extractCabRequests(elev),
		}
	}
	return out
}

func ProcessElevatorRequests(input HRAInput) (map[string]map[int][2]bool, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	exe := "Project/pkg/hra"
	if runtime.GOOS == "windows"{
		exe += ".exe"
	}

	cmd:= exec.Command(exe, "-i", string(inputJSON))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("hra execution failed: %v\n%s", err, string(output))
	}
	
	var outputRaw map[string]map[string][2]bool
	err = json.Unmarshal(output, &outputRaw)
	if err != nil{
		return nil, fmt.Errorf("hra output fail: %v", err)
	}

	//convert keys to int
	results:= make(map[string]map[int][2]bool)
	for idStr, floorMap := range outputRaw {
		results[idStr] = make(map[int][2]bool)
		for floorStr, btns:= range floorMap{
			floor, _:= strconv.Atoi(floorStr)
			results[idStr][floor] = btns
		}
	}
	return results, nil
}

// extract cab requests from elevator
func extractCabRequests(e common.Elevator) []bool {
	cabRequests := make([]bool, common.N_FLOORS)
	for f := range common.N_FLOORS {
		cabRequests[f] = e.Requests[f][elevio.BT_Cab]
	}
	return cabRequests
}
