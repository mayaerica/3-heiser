package messageProcessing

import (
	"elevatorlab/elevator"
    "elevatorlab/fsm"
	//"elevatorlab/requests"
    "time"
	"elevatorlab/network/peers"
	"fmt"
    "sync"
)


var printMessageCounter = 0

var Mu2 sync.Mutex // used to lock ElevatorStatus

//store last received messages & timestamps
var ElevatorStatus = map[string]bool{
	"8081": false, // Elevator 1 initially active
	"8082": false, // Elevator 2 initially active
	"8083": false, // Elevator 3 initially active
}


type Message struct {
    Elevator      elevator.Elevator      // Elevator field
    Active1       bool             // Is Elevator 1 active?
    Active2       bool             // Is Elevator 2 active?
    Active3       bool             // Is Elevator 3 active?
}

//print the last received messages
func PrintLastReceivedMessages(message Message) {
    // Print elevator ID and active status for each elevator
    fmt.Printf("Elevator ID: %-5d", message.Elevator.Id)

    // Print the entire Elevator struct for the target elevator
    fmt.Printf("  Floor: %-3d | Direction: %-8s | Behaviour: %-8s | Busy: %-5t | Dod: %-4v | ClearRequestVariant: %-5v\n",
        message.Elevator.Floor, message.Elevator.Dirn, message.Elevator.Behaviour,
        message.Elevator.Busy, message.Elevator.DoorOpenDuration, message.Elevator.ClearRequestVariant)
                
    // Print Button Requests before the existing Requests for the target elevator
    fmt.Println("  Button Requests: ")
    for _, btnReq := range message.Elevator.HallCalls {
        fmt.Printf("    Button %v\n", btnReq)
    }

    // Print Requests for the target elevator
    fmt.Println("  Requests: ")
    for i := 0; i < len(message.Elevator.Requests); i++ {
        fmt.Printf("    Floor %d: %v\n", i+1, message.Elevator.Requests[i])
        }

    fmt.Println("  Done: ")
    for i := 0; i < len(message.Elevator.Requests); i++ {
         fmt.Printf("    Floor %d: %v\n", i+1, message.Elevator.Done[i])
        }

}


func UpdateMessage(peerUpdateCh chan peers.PeerUpdate, messageTx chan Message){
    for{
        select {
            case p :=<-peerUpdateCh:

            for elevator := range p.Lost {
                Mu2.Lock()
                ElevatorStatus[p.Lost[elevator]]=false
                Mu2.Unlock()
            }

            if len (p.New)!=0{
                Mu2.Lock()
                ElevatorStatus[p.New]=true
                Mu2.Unlock()
            }

            default:
                
                msg := Message{
                    Elevator:      fsm.Elevator,              // Use the colon to assign a value to the Elevator field
                    Active1:       ElevatorStatus["8081"], 
                    Active2:       ElevatorStatus["8082"],
                    Active3:       ElevatorStatus["8083"],
                }
                messageTx <- msg
                time.Sleep(10 * time.Millisecond)
            }
        }
}