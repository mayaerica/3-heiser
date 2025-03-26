package backup

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"encoding/json"
	"log"
	"os"
)


const backupFile = "backup.json"

func SaveCabRequests(elevator common.Elevator) {
	
	var cabRequests [common.N_FLOORS]bool

	
	for floor := 0; floor < common.N_FLOORS; floor++ {
		
		cabRequests[floor] = elevator.Requests[floor][elevio.BT_Cab]
	}

	
	data, err := json.Marshal(cabRequests)
	if err != nil {
	
		log.Printf("backup: failed to marshal cab requests: %v", err)
		return
	}

	
	err = os.WriteFile(backupFile, data, 0644)
	if err != nil {
		
		log.Printf("backup: failed to write file: %v", err)
	}
}


func LoadCabRequests(elevator *common.Elevator) {
	
	data, err := os.ReadFile(backupFile)
	if err != nil {
		
		log.Printf("backup: no backup file found: %v", err)
		return
	}

	var cabRequests [common.N_FLOORS]bool

	err = json.Unmarshal(data, &cabRequests)
	if err != nil {
		log.Printf("backup: failed to unmarshal cab requests: %v", err)
		return
	}
	
	for floor := 0; floor < common.N_FLOORS; floor++ {
		if cabRequests[floor] {
			elevator.Requests[floor][elevio.BT_Cab] = true
		}
	}
}


