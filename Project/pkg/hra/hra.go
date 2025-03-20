package hra

import (
	"elevatorlab/common"
	"elevatorlab/pkg/utils"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
)

// to prevent race conditions in HRA processing
var mutex sync.Mutex

type HRAElevState struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

// input format for HRA
type hraInput struct {
	HallRequests [common.N_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

// get correct HRA executable based on OS
func getHRAExecutable() string {
	switch runtime.GOOS {
	case "linux":
		return "hall_request_assigner"
	case "windows":
		return "hall_request_assigner.exe"
	default:
		panic("Unsupported OS")
	}
}

// create HRA input from elevator states and hall requests
func CreateHRAInput(elevatorStates map[int]common.Elevator, hallRequests [common.N_FLOORS][2]bool) hraInput {
	mutex.Lock()
	defer mutex.Unlock()

	hraStates := make(map[string]HRAElevState)

	for id, elevator := range elevatorStates {
		hraStates[fmt.Sprintf("%d", id)] = HRAElevState{
			Behaviour:   utils.BehaviourToString(elevator.Behaviour),
			Floor:       elevator.Floor,
			Direction:   utils.DirectionToString(elevator.Dirn),
			CabRequests: extractCabRequests(elevator),
		}
	}

	return HRAInput{
		HallRequests: hallRequests,
		States:       hraStates,
	}
}

func ProcessElevatorRequests(input hraInput) (map[int]map[int][2]bool, error) {
	hraExecutable := getHRAExecutable()

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %v", err)
	}

	//execute HRA binary
	ret, err := exec.Command("./hall_request_assigner"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %v\nOutput: %s", err, string(ret))
	}

	//convert JSON response to Go structure
	var output map[string][][2]bool
	err = json.Unmarshal(ret, &output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %v", err)
	}
	return output, nil
}

// map HRA output to an elevators state
func HRAMapToElevator(hraOutput map[string][][2]bool, e common.Elevator) common.Elevator {
	mutex.Lock()
	defer mutex.Unlock()

	if floorRequests, exists := hraOutput[fmt.Sprintf("%d", e.ID)]; exists {
		for floor, buttons := range floorRequests {
			e.Requests[floor][0] = buttons[0] //Hall Up
			e.Requests[floor][1] = buttons[1] //Hall Down
		}
	}

	return e
}

// extract cab requests from elevator
func extractCabRequests(e common.Elevator) []bool {
	cabRequests := make([]bool, common.N_FLOORS)
	for floor := 0; floor < common.N_FLOORS; floor++ {
		cabRequests[floor] = e.Requests[floor][2]
	}
	return cabRequests
}
