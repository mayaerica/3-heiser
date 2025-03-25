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

// A wrapper type for different kinds of messages we want the assigner to handle
type AssignerMsg struct {
	Type string             // What type of message: "hall_call", "complete", or "assign"
	Data elevio.ButtonEvent // The button/floor this message refers to
}

// This is the input channel where the FSM and main forward hall button presses or completions
var AssignerInput = make(chan AssignerMsg)

// Initializes the assigner goroutine
func InitAssigner(myID string) {
	go assigner(myID)
}

func assigner(myID string) {
	var hallRequests [common.N_FLOORS][2]common.OrderState // Our view of the hall button matrix (notrequested, unassigned, etc.)
	var orderID [common.N_FLOORS][2]string                 // Tracks who we think is assigned to each button
	perspectiveMap := make(map[string]common.Perspective)  // What we know about other elevators' views

	// Channels for peer communication and periodic broadcasting
	perspectiveTx := make(chan common.Perspective) // What we send out
	perspectiveRx := make(chan common.Perspective) // What we receive from others
	peerUpdateChan := make(chan peers.PeerUpdate)  // Info about who joined/left

	// Set up the networking (broadcast and peer updates)
	go bcast.Transmitter(21478, perspectiveTx)
	go bcast.Receiver(21478, perspectiveRx)
	go peers.Receiver(15680, peerUpdateChan)

	// Every 20ms, we'll rebroadcast our view to the network
	ticker := time.NewTicker(20 * time.Millisecond)
	var peerList peers.PeerUpdate // Tracks who's currently online

	for {
		select {
		// Incoming messages from FSM (button press or complete)
		case msg := <-AssignerInput:
			fmt.Printf("[ASSIGNER] Received: %+v\n", msg) //added during blocking-debug
			f := msg.Data.Floor
			b := int(msg.Data.Button)

			switch msg.Type {
			case "hall_call":
				// A hall button was pressed by a user — mark it as needing assignment
				if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
					hallRequests[f][b] = common.Unassigned
				}
				UpdateHallLightsFromPerspective(hallRequests)

			case "complete":
				// A hall call was served. Depending on peer count, we either mark it gone or unknown
				if len(peerList.Peers) > 1 {
					// If we're in a network, clear it
					hallRequests[f][b] = common.NotRequested
				} else {
					// If alone, keep it as "unknown" (we don't trust we're really done)
					hallRequests[f][b] = common.Unknown
				}
				UpdateHallLightsFromPerspective(hallRequests)

			// We are asked to try assigning all unassigned calls
			case "assign":
				if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
					hallRequests[f][b] = common.Unassigned
				}

				// Get the current global view of elevators
				reply := make(chan map[string]common.Elevator)
				ElevGet <- ElevGetMsg{Reply: reply}
				allElevs := <-reply

				fmt.Println("\n\n\n\n\n")
				fmt.Println(AssignedHallRequests(common.Perspective{
					Perspective: hallRequests,
				}))

				fmt.Println("\n\n\n\n\n")

				// Prepare input for HRA (only currently unassigned hall calls)
				hraInput := hra.CreateHRAInput(allElevs, AssignedHallRequests(common.Perspective{
					Perspective: hallRequests,
				}))
				fmt.Println("\n\n\n\n\n")
				fmt.Println(hraInput)

				fmt.Println("\n\n\n\n\n")

				// Run HRA, the assignment optimizer
				hraOutput := hra.HRAProcessor(hraInput)
				if hraOutput == nil {
					fmt.Println("[HRA] error: assignment failed")
					break
				}

				// Apply the results from HRA
				for elevID, assignments := range *hraOutput {
					for floor, buttons := range assignments {
						for btn, assigned := range buttons {
							if assigned {

								//added under blocking-debug:
								fmt.Printf("[ASSIGNER] Assigning floor %d button %d to %s\n", floor, btn, elevID)

								hallRequests[floor][btn] = common.Assigned
								orderID[floor][btn] = elevID
								if elevID == myID {
									// This assignment is for 'me' → notify FSM
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

		// Another elevator sent us their current view
		case theirs := <-perspectiveRx:
			// Store their view
			perspectiveMap[theirs.ID] = theirs

			// Reconcile their view with ours, button-by-button
			for f := 0; f < common.N_FLOORS; f++ {
				for b := 0; b < 2; b++ {
					switch theirs.Perspective[f][b] {
					case common.NotRequested:
						// If they say it's done, and we thought it was still active — clear it
						if (hallRequests[f][b] == common.Assigned || hallRequests[f][b] == common.Unknown) && theirs.OrderID[f][b] == "Done" {
							hallRequests[f][b] = common.NotRequested
						}
					case common.Unassigned:
						// If they say unassigned, and we don't know about it — mark it unassigned
						if hallRequests[f][b] == common.NotRequested || hallRequests[f][b] == common.Unknown {
							hallRequests[f][b] = common.Unassigned
						}
					case common.Assigned:
						// Only mark it assigned if all peers agree it's assigned to me
						if EveryoneAgreesAssignedToMe(perspectiveMap, myID, f, b) {
							hallRequests[f][b] = common.Assigned
						}
					case common.Unknown:
						//  Another elevator says: "I don't know the state of this hall call."
						// If we thought the request was definitely NOT requested, we now downgrade to UNKNOWN too.
						// This prevents us from being overly confident when peers are unsure — especially useful
						// in recovery situations or after peer drops.
						if hallRequests[f][b] == common.NotRequested {
							hallRequests[f][b] = common.Unknown
						}
					}
				}
			}
		// If a peer disappears, we forget what they thought (temp)
		case peerList = <-peerUpdateChan:
			for _, lost := range peerList.Lost {
				delete(perspectiveMap, lost)
			}

		// Every 20ms, broadcast our current view to the rest of the network
		case <-ticker.C:
			perspectiveTx <- common.Perspective{
				ID:          myID,
				Perspective: hallRequests,
				OrderID:     orderID,
			}
		}
	}
}

// AssignedHallRequests extracts only the currently assigned hall requests
// from a full Perspective. It's used when feeding input into the HRA,
// because HRA only wants to know which buttons are actively being worked on.
//
// This helps avoid trying to reassign buttons that are already assigned.
func AssignedHallRequests(p common.Perspective) [common.N_FLOORS][2]bool {
	var out [common.N_FLOORS][2]bool
	fmt.Println("output:")
	fmt.Println(p.Perspective)
	fmt.Println(common.Assigned)
	for f := 0; f < common.N_FLOORS; f++ {
		for b := 0; b < 2; b++ {
			// Only keep track of buttons currently marked as "Assigned"
			out[f][b] = p.Perspective[f][b] == common.Assigned || p.Perspective[f][b] == common.Unassigned 
		}
	}
	return out
}

// EveryoneAgreesAssignedToMe checks if *all known peers*
// agree that a specific hall call is assigned to *me* (myID).
//
// This is used during state reconciliation when we receive
// other perspectives. It prevents premature assumptions —
// we only mark a call as Assigned if every elevator agrees.
//
// If even one peer thinks the call is assigned to someone else,
// then we don’t treat it as truly Assigned yet.
func EveryoneAgreesAssignedToMe(
	perspectives map[string]common.Perspective,
	myID string,
	floor int,
	button int,
) bool {
	if len(perspectives) == 0 {
		// If we don’t know what others think, we can’t claim consensus
		return false
	}
	for _, p := range perspectives {
		if p.OrderID[floor][button] != myID {
			return false
		}
	}
	return true
}

// IsAssignedInAny checks whether *any* peer has marked
// the given hall call as Assigned — even if they disagree who it's assigned to.
//
// This can be used for decisions like:
// “Should I avoid assigning this request again?” or
// “Is this button still active in the network?”
//
// Useful as a softer check than EveryoneAgreesAssignedToMe.
func IsAssignedInAny(perspectives map[string]common.Perspective, floor int, button int) bool {
	for _, p := range perspectives {
		if p.Perspective[floor][button] == common.Assigned {
			return true
		}
	}
	return false
}
