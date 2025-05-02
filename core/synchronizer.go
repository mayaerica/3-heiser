package core

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/network/bcast"
	"elevatorlab/network/peers"
	"time"
)

func RunSynchronizer(
	HallButtonPressChan <-chan elevio.ButtonEvent,
	OrderCompleteChan <-chan elevio.ButtonEvent,
	ExistingOrdersChan chan<- [common.N_FLOORS][2]bool,
	myID string,
) {
	perspectiveRxChan := make(chan common.Perspective)
	perspectiveTxChan := make(chan common.Perspective)
	peerUpdateChan := make(chan peers.PeerUpdate)

	go bcast.Transmitter(21478, perspectiveTxChan)
	go bcast.Receiver(21478, perspectiveRxChan)
	go peers.Receiver(15680, peerUpdateChan)

	localPerspective := [common.N_FLOORS][2]common.OrderState{}
	peerList := peers.PeerUpdate{}
	perspectiveMap := make(map[string]common.Perspective)

	ticker := time.NewTicker(20 * time.Millisecond)

	for {
		UpdateHallLightsFromPerspective(localPerspective)
		select {

		case btn := <-HallButtonPressChan:
			if localPerspective[btn.Floor][btn.Button] == common.NOT_SEEN || localPerspective[btn.Floor][btn.Button] == common.UNCERTAIN {
				localPerspective[btn.Floor][btn.Button] = common.SEEN_BY_SOMEONE
			}

		case completed := <-OrderCompleteChan:
			if len(peerList.Peers) > 1 {
				localPerspective[completed.Floor][completed.Button] = common.NOT_SEEN
			} else {
				localPerspective[completed.Floor][completed.Button] = common.UNCERTAIN
			}
			ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)

		case incomingPerspective := <-perspectiveRxChan:
			perspectiveMap[incomingPerspective.ID] = incomingPerspective

			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					switch incomingPerspective.Perspective[floor][btn] {

					case common.NOT_SEEN:
						if localPerspective[floor][btn] == common.SEEN_BY_EVERYONE || localPerspective[floor][btn] == common.UNCERTAIN {
							localPerspective[floor][btn] = common.NOT_SEEN
							ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)
						}

					case common.SEEN_BY_SOMEONE:
						if localPerspective[floor][btn] == common.NOT_SEEN || localPerspective[floor][btn] == common.UNCERTAIN {
							localPerspective[floor][btn] = common.SEEN_BY_SOMEONE
							ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)
						}
						if localPerspective[floor][btn] == common.SEEN_BY_SOMEONE &&
							countPeersWithState(perspectiveMap, floor, btn, common.SEEN_BY_SOMEONE) == len(peerList.Peers) {
							localPerspective[floor][btn] = common.SEEN_BY_EVERYONE
							ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)
						}

					case common.SEEN_BY_EVERYONE:
						if localPerspective[floor][btn] != common.SEEN_BY_EVERYONE {
							localPerspective[floor][btn] = common.SEEN_BY_EVERYONE
							ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)
						}

					case common.UNCERTAIN:

					}
				}
			}

		case peerList = <-peerUpdateChan:
			for _, lost := range peerList.Lost {
				delete(perspectiveMap, lost)
			}

		case <-ticker.C:
			perspectiveTxChan <- common.Perspective{ID: myID, Perspective: localPerspective}

			for floor := 0; floor < common.N_FLOORS; floor++ {
				for btn := 0; btn < 2; btn++ {
					if localPerspective[floor][btn] == common.SEEN_BY_SOMEONE &&
						countPeersWithState(perspectiveMap, floor, btn, common.SEEN_BY_SOMEONE) == len(peerList.Peers) {
						localPerspective[floor][btn] = common.SEEN_BY_EVERYONE
						ExistingOrdersChan <- seenByEveryoneMatrix(localPerspective)
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
			result[floor][btn] = (perspective[floor][btn] == common.SEEN_BY_EVERYONE)
		}
	}
	return result
}
