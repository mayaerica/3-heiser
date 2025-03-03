package tcp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	eio "Driver-go/elevio"
	"Driver-go/fsm" // Assuming fsm package where Elevator is defined
)

// Mutex for shared data
var mu sync.Mutex
var UpdateIntervall = 300 * time.Millisecond

// Message struct with Active status
type Message struct {
	Floor   int      // Floor number
	Dir     eio.Dirn // Direction
	Busy    bool     // Elevator status
	Id      int      // Elevator ID
	Active1 bool     // Is Elevator 1 active?
	Active2 bool     // Is Elevator 2 active?
	Active3 bool     // Is Elevator 3 active?
}

// Store last received messages & timestamps
var elevatorStatus = map[int]bool{
	8081: true, // Elevator 1 initially active
	8082: true, // Elevator 2 initially active
	8083: true, // Elevator 3 initially active
}

var lastReceivedMessages = map[int]Message{
	8081: {},
	8082: {},
	8083: {},
}

var elevatorAddresses = map[int]string{
	8081: "localhost:8081",
	8082: "localhost:8082",
	8083: "localhost:8083",
}

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
	if receivedMsg.Id == fsm.Elevator.Id { // Directly using fsm.Elevator.Id
		return
	}

	// Update the local elevator's status based on received message (but don't modify the local elevator's own active status)
	mu.Lock()
	lastReceivedMessages[receivedMsg.Id] = receivedMsg
	mu.Unlock()

	fmt.Printf("Updated message from Elevator %d: %+v\n", receivedMsg.Id, receivedMsg)
}

func SendMessage(msg Message) {
	// Always set the local elevator's active status to true
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

	// Create a WaitGroup to wait for both goroutines to finish
	var wg sync.WaitGroup

	// Send message to the other two elevators concurrently
	for id, address := range elevatorAddresses {
		if id != fsm.Elevator.Id { // Avoid sending to itself
			go func(addr string, targetID int) {

				conn, err := net.Dial("tcp", addr)
				if err != nil {
					// If connection fails, mark the target elevator as inactive
					mu.Lock()
					// Update elevator status in a thread-safe manner
					elevatorStatus[targetID] = false
					mu.Unlock()
					return
				}

				// If connection is successful, mark the elevator as active
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

				defer conn.Close()

				// Serialize and send message
				data, err := json.Marshal(msg)
				if err != nil {
					fmt.Println("Error serializing message:", err)
					return
				}
				_, err = conn.Write(data)
				if err != nil {
					fmt.Println("Error sending message:", err)
				}
			}(address, id) // Launch goroutine to handle this elevator

			wg.Wait()

		}
	}

	// Wait for both goroutines to complete before continuing

}

// Function to get the last received message
func GetLastReceivedMessage(id int) (Message, bool) {
	mu.Lock()
	defer mu.Unlock()
	msg, exists := lastReceivedMessages[id]
	return msg, exists
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
