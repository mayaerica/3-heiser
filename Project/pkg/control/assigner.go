package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/bcast"
	"elevatorlab/pkg/network/peers"
	"fmt"
	"time"
)

type AssignerMsg struct {
	Type string // "hall_call", "complete", "assign"
	Data elevio.ButtonEvent
}

var AssignerInput = make(chan AssignerMsg)

func InitAssigner(myID string) {
	go assigner(myID)
}

func assigner(myID string) {
	var hallRequests [common.N_FLOORS][2]common.OrderState
	var orderID [common.N_FLOORS][2]string
	perspectiveMap := make(map[string]common.Perspective)

	perspectiveTx := make(chan common.Perspective)
	perspectiveRx := make(chan common.Perspective)
	peerUpdateChan := make(chan peers.PeerUpdate)

	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)
	go peers.Receiver(15680, peerUpdateChan)

	ticker := time.NewTicker(20 * time.Millisecond)
	var peerList peers.PeerUpdate

	for {
		select {
		case msg := <-AssignerInput:
			f := msg.Data.Floor
			b := int(msg.Data.Button)

			switch msg.Type {
			case "hall_call":
				if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
					hallRequests[f][b] = common.Unassigned
				}

			case "complete":
				if len(peerList.Peers) > 1 {
					hallRequests[f][b] = common.NotRequested
				} else {
					hallRequests[f][b] = common.Unknown
				}

			case "assign":
				if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
					hallRequests[f][b] = common.Unassigned
				}

				reply := make(chan map[string]common.Elevator)
				ElevGet <- ElevGetMsg{Reply: reply}
				allElevs := <-reply

				hraInput := hra.CreateHRAInput(allElevs, AssignedHallRequests(common.Perspective{
					Perspective: hallRequests,
				}))

				hraOutput := hra.HRAProcessor(hraInput)
				if hraOutput == nil {
					fmt.Println("[HRA] error: assignment failed")
					break
				}

				for elevID, assignments := range *hraOutput {
					for floor, buttons := range assignments {
						for btn, assigned := range buttons {
							if assigned {
								hallRequests[floor][btn] = common.Assigned
								orderID[floor][btn] = elevID
								if elevID == myID {
									AssignedHallCallChan <- elevio.ButtonEvent{
										Floor:  floor,
										Button: elevio.ButtonType(btn),
									}
								}
							}
						}
					}
				}
			}

		case theirs := <-perspectiveRx:
			perspectiveMap[theirs.ID] = theirs

			for f := 0; f < common.N_FLOORS; f++ {
				for b := 0; b < 2; b++ {
					switch theirs.Perspective[f][b] {
					case common.NotRequested:
						if (hallRequests[f][b] == common.Assigned || hallRequests[f][b] == common.Unknown) && theirs.OrderID[f][b] == "Done" {
							hallRequests[f][b] = common.NotRequested
						}
					case common.Unassigned:
						if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
							hallRequests[f][b] = common.Unassigned
						}
					case common.Assigned:
						if EveryoneAgreesAssignedToMe(perspectiveMap, myID, f, b) {
							hallRequests[f][b] = common.Assigned
						}
					case common.Unknown:
						if hallRequests[f][b] == common.NotRequested {
							hallRequests[f][b] = common.Unknown
						}
					}
				}
			}

		case peerList = <-peerUpdateChan:
			for _, lost := range peerList.Lost {
				delete(perspectiveMap, lost)
			}

		case <-ticker.C:
			perspectiveTx <- common.Perspective{
				ID:          myID,
				Perspective: hallRequests,
				OrderID:     orderID,
			}
		}
	}
}

func AssignedHallRequests(p common.Perspective) [common.N_FLOORS][2]bool {
	var out [common.N_FLOORS][2]bool
	for f := 0; f < common.N_FLOORS; f++ {
		for b := 0; b < 2; b++ {
			out[f][b] = p.Perspective[f][b] == common.Assigned
		}
	}
	return out
}

func EveryoneAgreesAssignedToMe(
	perspectives map[string]common.Perspective,
	myID string,
	floor int,
	button int,
) bool {
	if len(perspectives) == 0 {
		return false
	}
	for _, p := range perspectives {
		if p.OrderID[floor][button] != myID {
			return false
		}
	}
	return true
}

func IsAssignedInAny(perspectives map[string]common.Perspective, floor int, button int) bool {
	for _, p := range perspectives {
		if p.Perspective[floor][button] == common.Assigned {
			return true
		}
	}
	return false
}