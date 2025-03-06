package messageProcessing

import (
	"elevatorlab/elevator"
	"elevatorlab/requests"
	"fmt"
)


var printMessageCounter = 0


// Store last received messages & timestamps
var elevatorStatus = map[int]bool{
	8081: true, // Elevator 1 initially active
	8082: true, // Elevator 2 initially active
	8083: true, // Elevator 3 initially active
}

type Message struct {
    Elevator      elevator.Elevator      // Elevator field
    Active1       bool             // Is Elevator 1 active?
    Active2       bool             // Is Elevator 2 active?
    Active3       bool             // Is Elevator 3 active?
    Requests [] requests.Request // List of button requests
}


func PrintLastReceivedMessages(message Message) {
    fmt.Println("############################", printMessageCounter, "##################################")
    fmt.Println("Last Received Messages:")
    // Ensure consistent order for elevators (8081, 8082, 8083)

    // Print other elevators in a fixed order
        msg := message
                // Print elevator ID and active status for each elevator
        fmt.Printf("Elevator ID: %-5d", msg.Elevator.Id)

                // Print the entire Elevator struct for the target elevator
        fmt.Printf("  Floor: %-3d | Direction: %-8s | Behaviour: %-8s | Busy: %-5t | Dod: %-4v | ClearRequestVariant: %-5v\n", msg.Elevator.Floor, msg.Elevator.Dirn, msg.Elevator.Behaviour, msg.Elevator.Busy, msg.Elevator.DoorOpenDuration, msg.Elevator.ClearRequestVariant)
                
                // Print Button Requests before the existing Requests for the target elevator
        fmt.Println("  Button Requests: ")
        for _, btnReq := range msg.Requests {
            fmt.Printf("    Button %v\n", btnReq)
           }

                // Print Requests for the target elevator
        fmt.Println("  Requests: ")
        for i := 0; i < len(msg.Elevator.Requests); i++ {
            fmt.Printf("    Floor %d: %v\n", i+1, msg.Elevator.Requests[i])
        }

    // Print the closing line for separation
    fmt.Println("#################################################################")

    // Increment the message counter
    printMessageCounter++
}
