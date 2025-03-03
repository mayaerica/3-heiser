package tcp

import (
	"fmt"
	"net"
	"os"
	"sync"
	"encoding/json"
	eio "Driver-go/elevio"
)

// Elevator addresses (for 3 elevators)
var elevatorAddresses = []string{
	"localhost:8081",
	"localhost:8082",
	"localhost:8083",
}

// Shared global data to manage communication between elevators
var mu sync.Mutex

type Message struct {
	Floor int    // Floor number (e.g., 0, 1, 2, ...)
	Dir  eio.Dirn  // Direction (e.g., "up", "down", "stop")
	Busy  bool   // Whether the elevator is busy or not
	Id    int    // Elevator ID
}

// Function to handle communication (acting as a server)
func handleClient(conn net.Conn) {
	defer conn.Close()  // This will close the connection once the function completes

	// Read the request from the client
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	// Process the request
	clientMessage := string(buffer[:n])
	fmt.Printf("Elevator received request: %s\n", clientMessage)

	// Simulate processing the request and send the response
	response := "Request received and processed"
	conn.Write([]byte(response))

	// The connection will be closed after the function completes, once the response is sent
}


// Function to send a request to all elevators (broadcasting)
func SendRequestToAllElevators(msg Message) {
	var wg sync.WaitGroup

	// Assume elevatorAddresses is a list of TCP addresses for the elevators
	elevatorAddresses := []string{
		"127.0.0.1:8081", // Example address
		"127.0.0.1:8082",
	}

	// Loop through all elevator addresses and send the message to each one
	for _, addr := range elevatorAddresses {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			conn, err := net.Dial("tcp", address)
			if err != nil {
				fmt.Println("Error connecting to elevator:", err)
				return
			}
			defer conn.Close() // Close the connection after sending the message

			// Serialize the Message struct to JSON
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Println("Error serializing message:", err)
				return
			}

			// Send the serialized message to the elevator
			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending message:", err)
				return
			}

			// Receive the response from the elevator (if needed)
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("Error reading response:", err)
				return
			}

			// Display the response from the elevator
			fmt.Printf("Response from %s: %s\n", address, string(buffer[:n]))
		}(addr)
	}

	// Wait for all goroutines to finish sending the messages
	wg.Wait()
}


// Function to start the server for the elevator (listening)
func StartServer(elevatorID string) {
	// Start listening for requests from other elevators
	listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", elevatorID))
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
	defer listen.Close()

	fmt.Printf("Elevator %s listening...\n", elevatorID)

	// Accept and handle incoming connections
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClient(conn)
	}
}

