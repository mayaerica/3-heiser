package hra

import (
	"elevatorlab/common"
	"elevatorlab/pkg/control"
	"fmt"
	"reflect"
)

// assigner continuously reads the global elevator state and confirmed hall calls,
// runs the HRA optimizer, and sends assigned hall calls to the local FSM.
func Coordinator(
	allElevators <-chan map[string]common.Elevator,
	existingOrders <-chan [common.N_FLOORS][2]bool,
	myID string,
	eFromHRA chan<- common.Elevator,
	elevGet chan<- control.ElevGetMsg,
) {
	var elevMap map[string]common.Elevator
	var hallRequests [common.N_FLOORS][2]bool
	var prevAssignments map[string][][2]bool

	for {
		select {
		case elevMap = <-allElevators:
		case hallRequests = <-existingOrders:
		}

		// Create input and run external optimizer
		hraInput := CreateHRAInput(elevMap, hallRequests)
		hraOutput := HRAProcessor(hraInput)
		if hraOutput == nil || reflect.DeepEqual(prevAssignments, *hraOutput) {
			continue
		}
		prevAssignments = *hraOutput
		assignedElev := HRAMapToElevator(*hraOutput, elevMap[myID])
		fmt.Printf("[HRA Coordinator] Assignment for %s: %+v\n", myID, assignedElev.Requests)
		eFromHRA <- assignedElev
	}
}
