package main

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/network/localip"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var myID string
	var port string

	// Define the command-line flags
	flag.StringVar(&myID, "id", "", "Elevator ID to use")
	flag.StringVar(&port, "port", "15657", "Port to use for the elevator (default is 15657)")
	flag.Parse()

	// Check if the elevator ID is specified, if not we show the usage
	if myID == "" && len(flag.Args()) < 1 {
		fmt.Println("usage: go run main.go -id=[elevatorID] OR --auto")
		return
	}

	// If --auto is provided, generate ID automatically
	if myID == "" && flag.Args()[0] == "--auto" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Could not get local IP:", err)
			return
		}
		myID = fmt.Sprintf("%s-%d", ip, os.Getpid()) //when running multiple elevators on the same machine
		fmt.Println("Auto-generated ID:", myID)
	} else if myID == "" {
		// If no ID is given, print usage
		fmt.Println("usage: go run main.go -id=[elevatorID] OR --auto")
		return
	}

	// Print the elevator starting info
	fmt.Printf("Elevator starting with ID: %s\n", myID)

	// Initialize the elevator I/O
	elevio.Init(fmt.Sprintf("localhost:%s", port), elevio.N_FLOORS)

	initial := common.Elevator{
		ID:                  myID,
		Floor:               elevio.GetFloor(),
		Dirn:                elevio.MD_Stop,
		Behaviour:           common.IDLE,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3000 * time.Millisecond,
	}
	fmt.Println(2)

	if initial.Floor == -1 {
		fmt.Println("Starting between floors. Moving down to find floor...")
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			floor := elevio.GetFloor()
			if floor != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				initial.Floor = floor
				break
			}
		}

		//added during blocking debugging:
		for f := 0; f < common.N_FLOORS; f++ {
			for btn := 0; btn < common.N_BUTTONS; btn++ {
				initial.Requests[f][btn] = false
			}
		}
		fmt.Println("[BOOT] Flushed all requests at startup.")
		elevio.SetFloorIndicator(initial.Floor)
	}

	var emptyHallRequests [common.N_FLOORS][2]common.OrderState
	control.UpdateAllLights(initial, emptyHallRequests)

	initial.ClearRequestVariant = common.CV_InDirn
	go control.RunElevState(myID, initial, 16570) //wrong port?
	control.InitFSM(myID, initial)
	control.InitAssigner(myID)
	go func() {
		for btn := range control.OrderCompleteChan {
			control.AssignerInput <- control.AssignerMsg{
				Type: "complete",
				Data: btn,
			}
		}
	}()

	select {}
}
