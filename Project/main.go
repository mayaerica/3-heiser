package main

import (
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/network/localip"
	"fmt"
	"os"
)

func main() {
	var myID string

	// Usage:
	//   go run main.go 0        → [manually] set elevator ID to "0"
	//   go run main.go --auto   → [auto-set] ID based on local IP

	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go [elevatorID] OR --auto ")
		return
	}

	if os.Args[1] == "--auto" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("could not get local IP:", err)
			return
		}
		myID = ip
	} else {
		myID = os.Args[1]
	}

	fmt.Printf("Elevator starting with ID: %s\n", myID)

	//func Init(address string, numFloors int)
	elevio.Init("localhost:15657", elevio.N_FLOORS)

	control.InitFSM(myID)
	control.InitDispatcher(myID, control.GetLocalElevator())
	go control.StartDispatcherLoop(
		myID,
		control.HallCallRequestChan,
		control.AssignedHallCallChan,
	)

	select {}
}
