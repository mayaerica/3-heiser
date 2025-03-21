package backup

import (
	"encoding/json"
	"elevatorlab/common"
	"elevatorlab/elevio"
	"os"
)

const backupFile = "backup.json"

//only cab requests are saved
func SaveCabRequests(elevator common.Elevator){
	var cabRequests [common.N_FLOORS]bool
	for floor := 0; floor < common.N_FLOORS; floor++ {
		cabRequests[floor] = elevator.Requests[floor][elevio.BT_Cab]
	}
	data, _:= json.Marshal(elevator.Requests)
	os.WriteFile(backupFile, data, 0644)
}

//load the saved cab requests and merge it into elevator state
func LoadCabRequests(elevator *common.Elevator){
	data, err:= os.ReadFile(backupFile)
	if err != nil {
		return
	}
	var cabRequests [common.N_FLOORS]bool
	if err := json.Unmarshal(data, &cabRequests); err != nil {
		return
	}
	for floor := 0; floor < common.N_FLOORS; floor++ {
		if cabRequests[floor] {
			elevator.Requests[floor][elevio.BT_Cab] = true
		}
	}
}
	
