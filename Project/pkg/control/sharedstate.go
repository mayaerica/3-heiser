package control

import (
	"elevatorlab/common"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"time"
)

// A message type for sending instructions to update the shared elevator map
type ElevSetMsg struct {
	Fn func(map[string]common.Elevator)
}

// A message type for asking for a snapshot (copy) of the shared elevator map
type ElevGetMsg struct {
	Reply chan map[string]common.Elevator
}

// The actual channels used throughout the system
// var (
// 	ElevSet = make(chan ElevSetMsg) // Channel for setting elevator states
// 	ElevGet = make(chan ElevGetMsg) // Channel for getting elevator states
// )

// This function manages the master map of all elevators in the system.
// Think of this as the “control room” where every update request is handled one at a time.
func RunElevState(myID string, initial common.Elevator, port int, ElevSet chan ElevSetMsg, ElevGet chan ElevGetMsg) {
	tx := make(chan common.Elevator)           // What we broadcast to others
	rx := make(chan common.Elevator)           // What we receive from others
	peerUpdates := make(chan peers.PeerUpdate) // Keeps track of who’s online

	go bcast.Transmitter(port, tx)        // Start sending our state to the network
	go bcast.Receiver(port, rx)           // Start listening for other elevators' states
	go peers.Receiver(15680, peerUpdates) // Start tracking connected peers

	// This is the full map of all known elevators and their state
	elevators := map[string]common.Elevator{
		myID: initial, // Start with ourselves
	}

	ticker := time.NewTicker(20 * time.Millisecond) // Every 20ms we broadcast our current state

	for {
		select {
		case other := <-rx:
			// Received a broadcast from another elevator
			// Update their entry in our map
			elevators[other.ID] = other

		case set := <-ElevSet:
			// Someone wants to update the elevator map
			// We run their function on it (e.g., "change floor" or "add request")
			set.Fn(elevators)

		case get := <-ElevGet:
			// Someone wants to read the map (safely)
			// We send them a **copy** so they don’t mess with the real one
			get.Reply <- copyElevMap(elevators)

		case update := <-peerUpdates:
			// A peer disconnected — remove them from the map
			for _, lostID := range update.Lost {
				delete(elevators, lostID)
			}

		case <-ticker.C:
			// Time to broadcast our own state to the network
			if me, ok := elevators[myID]; ok {
				tx <- me
			}
		}
	}
}

// Makes a deep copy of the map so no one touches the original by accident
func copyElevMap(original map[string]common.Elevator) map[string]common.Elevator {
	copy := make(map[string]common.Elevator)
	for id, elev := range original {
		copy[id] = elev
	}
	return copy
}
