package core

import (
	"elevatorlab/common"
	"elevatorlab/network/bcast"
	"elevatorlab/network/peers"
	"time"
)

// Used to send instructions to update shared elevator map
type ElevSetMsg struct {
	Fn func(map[string]common.Elevator)
}

type ElevGetMsg struct {
	Reply chan map[string]common.Elevator
}

func RunStateMap(myID string,
	initial common.Elevator,
	port int,
	allElevators chan map[string]common.Elevator,
	ElevSet chan ElevSetMsg,
	ElevGet chan ElevGetMsg) {

	tx := make(chan common.Elevator)
	rx := make(chan common.Elevator)
	peerUpdates := make(chan peers.PeerUpdate)

	go bcast.Transmitter(port, tx)
	go bcast.Receiver(port, rx)
	go peers.Receiver(15680, peerUpdates)

	elevators := map[string]common.Elevator{
		myID: initial,
	}

	ticker := time.NewTicker(20 * time.Millisecond) 

	for {
		select {
		case other := <-rx:
			elevators[other.ID] = other
			allElevators <- copyElevMap(elevators)

		case set := <-ElevSet:
			set.Fn(elevators)
			allElevators <- copyElevMap(elevators)

		case get := <-ElevGet:
			get.Reply <- copyElevMap(elevators)

		case update := <-peerUpdates:
			for _, lostID := range update.Lost {
				delete(elevators, lostID)
			}
			allElevators <- copyElevMap(elevators)

		case <-ticker.C:
			if me, ok := elevators[myID]; ok {
				tx <- me
			}
		}
	}
}


func copyElevMap(original map[string]common.Elevator) map[string]common.Elevator {
	copyMap := make(map[string]common.Elevator)
	for key, value := range original {
		newElevator := common.Elevator{
			ID:                  value.ID,
			Floor:               value.Floor,
			Dirn:                value.Dirn,
			Behaviour:           value.Behaviour,           
			ClearRequestVariant: value.ClearRequestVariant, 
			DoorOpenDuration:    value.DoorOpenDuration,
		}

		newElevator.Requests = [common.N_FLOORS][common.N_BUTTONS]bool{}
		for i := 0; i < common.N_FLOORS; i++ {
			for j := 0; j < common.N_BUTTONS; j++ {
				newElevator.Requests[i][j] = value.Requests[i][j]
			}
		}
		
		copyMap[key] = newElevator
	}

	return copyMap
}
