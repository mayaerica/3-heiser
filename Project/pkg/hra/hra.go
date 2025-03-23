package hra

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/utils"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

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

func HRAProcessor(currentInput HRAInput) *map[string][][2]bool {

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux" :  hraExecutable = "hall_request_assigner"
	case "windows": hraExecutable = "hall_request_assigner.exe"
	default:        panic("Unsupported OS")
	}

	jsonBytes, err := json.Marshal(currentInput)
	if err != nil {
		fmt.Println("json.Marshal error:", err)
		return nil
	}

	fmt.Println("[HRA] Running binary:", hraExecutable)
	ret, err := exec.Command(hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error:", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error:", err)
		return nil
	}

	fmt.Printf("HRAoutput:\n")
	for k, v := range *output {
		fmt.Printf("  %6v  %+v\n", k, v)
	}

	return output
}


func extractCabRequests(e common.Elevator) []bool {
	cabRequests := make([]bool, common.N_FLOORS)
	for f := range common.N_FLOORS {
		cabRequests[f] = e.Requests[f][elevio.BT_Cab]
	}

	return cabRequests
}
