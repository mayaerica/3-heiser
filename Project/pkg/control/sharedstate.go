package control

import (
	"elevatorlab/common"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"time"
)

type ElevSetMsg struct {
	Fn func(map[string]common.Elevator)
}

type ElevGetMsg struct {
	Reply chan map[string]common.Elevator
}

var (
	ElevSet = make(chan ElevSetMsg)
	ElevGet = make(chan ElevGetMsg)
)

func RunElevState(myID string, initial common.Elevator, port int) {
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

		case set := <-ElevSet:
			set.Fn(elevators)

		case get := <-ElevGet:
			get.Reply <- copyElevMap(elevators)

		case update := <-peerUpdates:
			for _, lostID := range update.Lost {
				delete(elevators, lostID)
			}

		case <-ticker.C:
			if me, ok := elevators[myID]; ok {
				tx <- me
			}
		}
	}
}

func copyElevMap(original map[string]common.Elevator) map[string]common.Elevator {
	copy := make(map[string]common.Elevator)
	for id, elev := range original {
		copy[id] = elev
	}
	return copy
}
