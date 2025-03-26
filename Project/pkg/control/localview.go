package control

import (
	"elevatorlab/common"
	"encoding/json"
)

// This function safely updates *your own* elevator inside the global map.
//
// You give it a function `fn`, and it runs that function on your elevator,
// and then saves it back into the shared map.
//
// It’s like saying:
//
//	"Hey control room, please modify my entry in this way."
// func WithMyElevator(myID string, fn func(e *common.Elevator)) {
// 	ElevSet <- ElevSetMsg{
// 		Fn: func(m map[string]common.Elevator) {
// 			e := m[myID] // get a copy of your elevator
// 			fn(&e)       // apply the changes
// 			m[myID] = e  // save it back into the map
// 		},
// 	}
// }

// /////////////////////////////
// DEEP COPY VERSION TO TEST //
// /////////////////////////////
func deepCopyElevator(e common.Elevator) common.Elevator {
	var newElevator common.Elevator
	data, _ := json.Marshal(e)
	json.Unmarshal(data, &newElevator)
	return newElevator
}

func WithMyElevator(myID string, fn func(e *common.Elevator)) {
	ElevSet <- ElevSetMsg{
		Fn: func(m map[string]common.Elevator) {
			e, exists := m[myID]
			if !exists {
				return
			}
			eCopy := deepCopyElevator(e)
			fn(&eCopy)
			m[myID] = eCopy
		},
	}
}

// This function reads *your own* elevator from the shared map.
//
// You send a read request to the control room, and it sends back the entire map.
// You pick out your own elevator from the reply.
func GetMyElevator(myID string) common.Elevator {
	reply := make(chan map[string]common.Elevator) // where the answer will come
	ElevGet <- ElevGetMsg{Reply: reply}            // ask for the map
	return (<-reply)[myID]                         // extract your elevator
}
