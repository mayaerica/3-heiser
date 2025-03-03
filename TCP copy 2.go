package tcp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	req "Driver-go/requests"
	el "Driver-go/elevator"
	"Driver-go/fsm" // Assuming fsm package where Elevator is defined
)

// Mutex for shared data
var mu sync.Mutex
var UpdateIntervall = 1000 * time.Millisecond

var printMessageCounter = 0


// Message struct with Active status
type Message struct {
    Elevator      el.Elevator      // Elevator field
    Active1       bool             // Is Elevator 1 active?
    Active2       bool             // Is Elevator 2 active?
    Active3       bool             // Is Elevator 3 active?
    Requests []req.Request // List of button requests
}


// Store last received messages & timestamps
var elevatorStatus = map[int]bool{
	8081: true, // Elevator 1 initially active
	8082: true, // Elevator 2 initially active
	8083: true, // Elevator 3 initially active
}

var lastReceivedMessages = map[int]Message{
	8081: {Elevator: el.Elevator{Id: 8081}, Active1: false, Active2: false, Active3: false},
	8082: {Elevator: el.Elevator{Id: 8082}, Active1: false, Active2: false, Active3: false},
	8083: {Elevator: el.Elevator{Id: 8083}, Active1: false, Active2: false, Active3: false},
}

var elevatorAddresses = map[int]string{
	8081: "localhost:8081",
	8082: "localhost:8082",
	8083: "localhost:8083",
}


//##################################THIS JUST PRINTS STUFF EVERY ELEVATOR FROM THIS ELEVATORS PERSPECTIVE############################################################################

func PrintLastReceivedMessages(messages map[int]Message) {
    fmt.Println("############################", printMessageCounter, "##################################")
    fmt.Println("Last Received Messages:")

    // Print the local elevator (fsm.Elevator) first
    if fsm.Elevator.Id != 0 { // Make sure the local elevator has a valid ID
        // Print elevator ID and active status
        fmt.Printf("Elevator ID: %-5d | Active: %-5t\n", fsm.Elevator.Id, elevatorStatus[fsm.Elevator.Id])

        // Print the entire Elevator struct for the current elevator (fsm.Elevator)
        fmt.Printf("  Floor: %-3d | Direction: %-3s | Behaviour: %-3s | Busy: %-5t | Dod: %-4v | ClearRequestVariant: %-5v\n",
            fsm.Elevator.Floor, fsm.Elevator.Dirn, fsm.Elevator.Behaviour, fsm.Elevator.Busy, fsm.Elevator.DoorOpenDuration, fsm.Elevator.ClearRequestVariant)
        
        // Print Requests for the local elevator
        fmt.Println("  Requests: ")
        for i := 0; i < len(fsm.Elevator.Requests); i++ {
            fmt.Printf("    Floor %d: %v\n", i+1, fsm.Elevator.Requests[i])
        }
    }

    // Ensure consistent order for elevators (8081, 8082, 8083)
    elevatorOrder := []int{8081, 8082, 8083}

    // Print other elevators in a fixed order
    for _, id := range elevatorOrder {
        if id != fsm.Elevator.Id { // Skip the current elevator (fsm.Elevator) from being printed again
            msg, exists := messages[id]
            if exists {
                // Print elevator ID and active status for each elevator
                fmt.Printf("Elevator ID: %-5d | Active: %-5t\n", id, elevatorStatus[id])

                // Print the entire Elevator struct for the target elevator
                fmt.Printf("  Floor: %-3d | Direction: %-8s | Behaviour: %-8s | Busy: %-5t | Dod: %-4v | ClearRequestVariant: %-5v\n",
                    msg.Elevator.Floor, msg.Elevator.Dirn, msg.Elevator.Behaviour, msg.Elevator.Busy, msg.Elevator.DoorOpenDuration, msg.Elevator.ClearRequestVariant)
                
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
            }
        }
    }

    // Print the closing line for separation
    fmt.Println("#################################################################")

    // Increment the message counter
    printMessageCounter++
}


//########################################################################################################################################################


// Function to handle incoming messages
func handleClient(conn net.Conn) {
	defer conn.Close()

	// Read message
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	// Decode JSON
	var receivedMsg Message
	err = json.Unmarshal(buffer[:n], &receivedMsg)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Ignore messages from itself
	if receivedMsg.Elevator.Id == fsm.Elevator.Id { // Directly using fsm.Elevator.Id
		return
	}

	// Update the local elevator's status based on received message (but don't modify the local elevator's own active status)
	mu.Lock()
	lastReceivedMessages[receivedMsg.Elevator.Id] = receivedMsg
	mu.Unlock()

	//fmt.Printf("Updated message from Elevator %d: %+v\n", receivedMsg.Id, receivedMsg)
}


func SendMessage(msg Message) {
	msg.Elevator = fsm.Elevator
	msg.Active1 = false
	msg.Active2 = false
	msg.Active3 = false
	

	mu.Lock()
	switch fsm.Elevator.Id {
	case 8081:
		msg.Active1 = true
	case 8082:
		msg.Active2 = true
	case 8083:
		msg.Active3 = true
	}
	mu.Unlock()

	// Synchronize to wait for both elevators (3 elevators total, minus the current one)

	// Synchronize goroutines using doneCount
	doneCount := 0

	for id, addr := range elevatorAddresses {
		if id != fsm.Elevator.Id { // Avoid sending to itself
			// Launch goroutine for each elevator
			go func(addr string, targetID int) {

				// Step 1: Try to connect to the target elevator
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					// If connection fails, mark the target elevator as inactive
					mu.Lock()
					elevatorStatus[targetID] = false
					mu.Unlock()

					// Increment doneCount as this goroutine is done
					mu.Lock()
					doneCount++
					mu.Unlock()

					return
				}

				// Step 2: If connection is successful, mark the elevator as active
				mu.Lock()
				elevatorStatus[targetID] = true
				// Update Active status in the message for the other elevators
				switch targetID {
				case 8081:
					msg.Active1 = elevatorStatus[8081]
				case 8082:
					msg.Active2 = elevatorStatus[8082]
				case 8083:
					msg.Active3 = elevatorStatus[8083]
				}
				mu.Unlock()

				// Increment doneCount as this goroutine is done
				mu.Lock()
				doneCount++
				mu.Unlock()

				// Wait until both goroutines are done before proceeding
				for doneCount < 2 {
					//wait
				}

				// Step 3: Send the message to the connected elevator
				defer conn.Close()

				// Serialize and send the message
				data, err := json.Marshal(msg)
				if err != nil {
					fmt.Println("Error serializing message:", err)
					return
				}

				_, err = conn.Write(data)
				if err != nil {
					fmt.Println("Error sending message:", err)
				} 
			}(addr, id) // Launch goroutine to handle this elevator
		}
	}

	// Wait for all goroutines to complete before continuing

}

// Function to get the last received message
func GetLastReceivedMessages() map[int]Message {
	mu.Lock()
	defer mu.Unlock()

	return lastReceivedMessages
}



// Function to start the server for an elevator
func StartServer() {
	listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", fsm.Elevator.Id))
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
	defer listen.Close()

	fmt.Printf("Elevator %d listening...\n", fsm.Elevator.Id)

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClient(conn)
	}
	
}
