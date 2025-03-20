package backup

import (
	"os"
	"encoding/json"
	"elevatorlab/common"
)

const backupFile = "elevator_backup.json"

func SaveCabRequests(elevator common.Elevator){
	data, _:= json.Marshal(elevator.Requests)
	os.WriteFile(backupFile, data, 0644)
}

func LoadCabRequests(elevator *common.Elevator){
	data, err:= os.ReadFile(backupFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &elevator.Requests)
}
