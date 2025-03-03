package main

import (
	"fmt"
	"net"
	"time"
)

const (
	primaryAddress  = "localhost:9999"  // Address used by the primary
	backupAddress   = "localhost:9998"  // Address used by the backup
	deltaT   = 1 * time.Second
	timeout         = 3 * deltaT // Timeout for failure
	initialCounter  = 1
)

func runPrimary(counter int) {
	fmt.Println("Starting primary process...")

	// Primery listen to 9999 port
	conn, err := net.ListenPacket("udp", primaryAddress)
	if err != nil {
		fmt.Println("Error starting primary UDP listener:", err)
		return
	}
	defer conn.Close()

	for {
		fmt.Println("Primary counter:", counter)
		counter++
		// Send "alive" to backup port
		conn.WriteTo([]byte("alive"), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9998})
		time.Sleep(deltaT)

		if counter == 5 {
			// Simulation of the primary failure
			fmt.Println("Simulating primary failure...")
			return
		}
	}
}

func runBackup() {
	fmt.Println("Starting backup process...")

	// Backup listen to 9998 port
	conn, err := net.ListenPacket("udp", backupAddress)
	if err != nil {
		fmt.Println("Error starting backup UDP listener:", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	counter := initialCounter

	for {
		conn.SetReadDeadline(time.Now().Add(timeout))
		_, _, err := conn.ReadFrom(buffer)
		if err != nil {
			// If the primary is dead, the backup take over
			fmt.Println("Primary process seems dead, taking over as primary...")
			runPrimary(counter)
			return
		}
		counter++
	}
}

func main() {
	go runPrimary(initialCounter)
	runBackup()
}
