package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// Elevator addresses (for 3 elevators)
var elevatorAddresses = []string{
	"localhost:8081",
	"localhost:8082",
	"localhost:8083",
}

// Shared global data to manage communication between elevators
var mu sync.Mutex

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
func sendRequestToAllElevators(action string) {
	var wg sync.WaitGroup
	for _, addr := range elevatorAddresses {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			conn, err := net.Dial("tcp", address)
			if err != nil {
				fmt.Println("Error connecting to elevator:", err)
				return
			}
			defer conn.Close() // Each goroutine closes its own connection after communication

			// Send the action to the server
			_, err = conn.Write([]byte(action))
			if err != nil {
				fmt.Println("Error sending message:", err)
				return
			}

			// Receive the response from the server
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("Error reading response:", err)
				return
			}

			// Display the response
			fmt.Printf("Response from %s: %s\n", address, string(buffer[:n]))

			// The connection will be closed when this goroutine completes
		}(addr)
	}

	// Wait for all goroutines to finish
	wg.Wait()
}


// Function to start the server for the elevator (listening)
func startServer(elevatorID string) {
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

func main() {
	// Elevator identification (e.g., "8081", "8082", "8083")
	elevatorID := "8083" // Change to "8082" or "8083" for different elevators

	// Start the server for the current elevator
	go startServer(elevatorID)

	// Simulate sending a request to all elevators
	var action string
	for {
		fmt.Println("Enter action (move, open, close, or exit):")
		fmt.Scanln(&action)

		if action == "exit" {
			fmt.Println("Exiting...")
			break
		}

		// Broadcast the request to all elevators
		if action == "move" || action == "open" || action == "close" {
			sendRequestToAllElevators(elevatorID + ":" + action)
		}
	}
}
