package backup

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"encoding/json"
	"log"
	"os"
)

const BackupFile = "backup.json"

type ElevatorState struct {
	CabRequests [common.N_FLOORS]bool
	Direction   elevio.Dirn
}

func SaveCabRequests(elevator common.Elevator) {
	var cabRequests [common.N_FLOORS]bool

	for floor := 0; floor < common.N_FLOORS; floor++ {
		cabRequests[floor] = elevator.Requests[floor][elevio.BT_Cab]
	}

	state := ElevatorState{
		CabRequests: cabRequests,
		Direction:   elevator.Dirn,
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("backup: failed to marshal elevator state: %v", err)
		return
	}

	err = os.WriteFile(BackupFile, data, 0644)
	if err != nil {
		log.Printf("backup: failed to write file: %v", err)
	}
}

func LoadCabRequests(elevator *common.Elevator) {
	data, err := os.ReadFile(BackupFile)
	if err != nil {
		log.Printf("backup: no backup file found: %v", err)
		return
	}

	var state ElevatorState
	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Printf("backup: failed to unmarshal elevator state: %v", err)
		return
	}

	for floor := 0; floor < common.N_FLOORS; floor++ {
		if state.CabRequests[floor] {
			elevator.Requests[floor][elevio.BT_Cab] = true
		}
	}

	elevator.Dirn = state.Direction
}
