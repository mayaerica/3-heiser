package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"time"
)

func RunSynchronizer(
	hallButtonPress <-chan elevio.ButtonEvent,
	orderComplete <-chan elevio.ButtonEvent,
	existingOrders chan<- [common.N_FLOORS][2]bool,
	myID string,
) {
	perspectiveRx := make(chan common.Perspective)
	perspectiveTx := make(chan common.Perspective)

	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)

	peerUpdateCh := make(chan peers.PeerUpdate)
	go peers.Receiver(15680, peerUpdateCh)

	localPerspective := [common.N_FLOORS][2]common.OrderState{}
	peerList := peers.PeerUpdate{}
	perspectiveMap := make(map[string]common.Perspective)

	ticker := time.NewTicker(20 * time.Millisecond)

	for {
		UpdateHallLightsFromPerspective(localPerspective)
		//fmt.Println("Local perspective %v", localPerspective)
		select {
		case btn := <-hallButtonPress:
			if localPerspective[btn.Floor][btn.Button] == common.NotSeen || localPerspective[btn.Floor][btn.Button] == common.Uncertain {
				localPerspective[btn.Floor][btn.Button] = common.SeenBySomeone
			}

		case completed := <-orderComplete:
			if len(peerList.Peers) > 1 {
				localPerspective[completed.Floor][completed.Button] = common.NotSeen
			} else {
				localPerspective[completed.Floor][completed.Button] = common.Uncertain
			}
			existingOrders <- seenByEveryoneMatrix(localPerspective)

		case incomingPerspective := <-perspectiveRx:
			perspectiveMap[incomingPerspective.ID] = incomingPerspective

			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					switch incomingPerspective.Perspective[floor][btn] {

					case common.NotSeen:
						if localPerspective[floor][btn] == common.SeenByEveryone || localPerspective[floor][btn] == common.Uncertain {
							localPerspective[floor][btn] = common.NotSeen
							existingOrders <- seenByEveryoneMatrix(localPerspective)
						}

					case common.SeenBySomeone:
						if localPerspective[floor][btn] == common.NotSeen || localPerspective[floor][btn] == common.Uncertain {
							localPerspective[floor][btn] = common.SeenBySomeone
							existingOrders <- seenByEveryoneMatrix(localPerspective)
						}
						if localPerspective[floor][btn] == common.SeenBySomeone &&
							countPeersWithState(perspectiveMap, floor, btn, common.SeenBySomeone) == len(peerList.Peers) {
							localPerspective[floor][btn] = common.SeenByEveryone
							existingOrders <- seenByEveryoneMatrix(localPerspective)
						}

					case common.SeenByEveryone:
						if localPerspective[floor][btn] != common.SeenByEveryone {
							localPerspective[floor][btn] = common.SeenByEveryone
							existingOrders <- seenByEveryoneMatrix(localPerspective)
						}

					case common.Uncertain:
						// No change
					}
				}
			}

		case peerList = <-peerUpdateCh:
			for _, lost := range peerList.Lost {
				delete(perspectiveMap, lost)
			}

		case <-ticker.C:
			perspectiveTx <- common.Perspective{ID: myID, Perspective: localPerspective}

			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					if localPerspective[floor][btn] == common.SeenBySomeone &&
						countPeersWithState(perspectiveMap, floor, btn, common.SeenBySomeone) == len(peerList.Peers) {
						localPerspective[floor][btn] = common.SeenByEveryone
						existingOrders <- seenByEveryoneMatrix(localPerspective)
					}
				}
			}
		}

	}
}

func countPeersWithState(m map[string]common.Perspective, floor, btn int, state common.OrderState) int {
	count := 0
	for _, p := range m {
		if p.Perspective[floor][btn] == state {
			count++
		}
	}
	return count
}

func seenByEveryoneMatrix(perspective [common.N_FLOORS][2]common.OrderState) [common.N_FLOORS][2]bool {
	var result [common.N_FLOORS][2]bool
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			result[floor][btn] = (perspective[floor][btn] == common.SeenByEveryone)
		}
	}
	return result
}
