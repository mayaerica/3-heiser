package main

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/network/localip"
	"fmt"
	"os"
)

func main() {
	var myID string

	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go [elevatorID] OR --auto")
		return
	}

	if os.Args[1] == "--auto" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Could not get local IP:", err)
			return
		}
		myID = ip
	} else {
		myID = os.Args[1]
	}

	fmt.Printf("Elevator starting with ID: %s\n", myID)

	elevio.Init("localhost:15657", elevio.N_FLOORS)

	initial := common.Elevator{
		ID:                  myID,
		Floor:               elevio.GetFloor(),
		Dirn:                elevio.MD_Stop,
		Behaviour:           common.IDLE,
		ClearRequestVariant: common.CV_All,
		DoorOpenDuration:    3,
	}

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
	}

	elevio.SetFloorIndicator(initial.Floor)

	go control.RunElevState(myID, initial, 16570)
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
