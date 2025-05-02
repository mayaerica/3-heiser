package hra

import (
	"elevatorlab/common"
	"elevatorlab/core"
	"reflect"
)

// Reads the elevator map, confirms hall calls,
// runs the HRA optimizer, and sends assigned hall calls.
func Coordinator(
	AllElevatorsChan <-chan map[string]common.Elevator,
	ExistingOrdersChan <-chan [common.N_FLOORS][2]bool,
	myID string,
	FromHRAChan chan<- common.Elevator,
	ElevGetChan chan<- core.ElevGetMsg,
) {
	var elevMap map[string]common.Elevator
	var hallRequests [common.N_FLOORS][2]bool
	var prevAssignmentsMap map[string][][2]bool

	for {
		select {
		case elevMap = <-AllElevatorsChan:
		case hallRequests = <-ExistingOrdersChan:
		}
		hraInput := CreateHRAInput(elevMap, hallRequests)
		hraOutput := HRAProcessor(hraInput)
		if hraOutput == nil || reflect.DeepEqual(prevAssignmentsMap, *hraOutput) {
			continue
		}
		prevAssignmentsMap = *hraOutput
		assignedElev := HRAMapToElevator(*hraOutput, elevMap[myID])
		FromHRAChan <- assignedElev
	}
}
