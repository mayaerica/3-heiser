package main

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/localip"
	"elevatorlab/pkg/network/peers"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// ────────────────────────────────
	// Step 1: Get Elevator ID and Port
	// ────────────────────────────────
	var myID, port string
	flag.StringVar(&myID, "id", "", "Elevator ID to use")
	flag.StringVar(&port, "port", "15657", "Port to use for I/O device")
	flag.Parse()

	if myID == "" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Could not get local IP:", err)
			return
		}
		myID = fmt.Sprintf("%s-%d", ip, os.Getpid()) // Unique ID for each process
		fmt.Println("Auto-generated ID:", myID)
	}
	fmt.Println("Starting elevator with ID:", myID)

	// ────────────────────────────────
	// Step 2: Initialize elevator hardware
	// ────────────────────────────────
	elevio.Init("localhost:"+port, elevio.N_FLOORS)

	initial := common.Elevator{
		ID:                  myID,
		Floor:               elevio.GetFloor(),
		Dirn:                elevio.MD_Stop,
		Behaviour:           common.IDLE,
		ClearRequestVariant: common.CV_InDirn,
		DoorOpenDuration:    3 * time.Second,
	}

	if initial.Floor == -1 {
		fmt.Println("[BOOT] Between floors, moving down to find one...")
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			if f := elevio.GetFloor(); f != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				initial.Floor = f
				break
			}
		}
	}

	elevio.SetFloorIndicator(initial.Floor)

	// ────────────────────────────────
	// Step 3: Define Channels
	// ────────────────────────────────
	hallButtonPress := make(chan elevio.ButtonEvent, 10)      // All hall calls
	orderComplete := make(chan elevio.ButtonEvent, 10)        // Signals a hall call was completed
	existingOrders := make(chan [common.N_FLOORS][2]bool, 10) // Confirmed orders from sync
	allElevators := make(chan map[string]common.Elevator, 10) // Shared elevator state
	assignments := make(chan common.Elevator, 10)             // HRA-assigned elevator state
	elevTx := make(chan common.Elevator, 10)                  // Outbound elevator info to others
	peerTxEnable := make(chan bool)
	elevSet := make(chan control.ElevSetMsg, 10)
	elevGet := make(chan control.ElevGetMsg, 10)

	// ────────────────────────────────
	// Step 4: Shared State
	// ────────────────────────────────
	go peers.Transmitter(15680, myID, peerTxEnable)
	go control.RunElevState(myID, initial, 1650, allElevators, elevSet, elevGet)

	// ────────────────────────────────
	// Step 5: FSM
	// ────────────────────────────────
	go control.InitFSM(myID, initial, orderComplete, hallButtonPress, assignments, elevSet, elevGet)

	// ────────────────────────────────
	// Step 6: Door FSM (runs obstruction + close timer)
	// ────────────────────────────────
	// go control.DoorFSM(control.DoorOpenChan, control.DoorCloseChan, initial.DoorOpenDuration)

	// ────────────────────────────────
	// Step 7: Synchronizer (hall order consensus logic)
	// ────────────────────────────────
	go control.RunSynchronizer(hallButtonPress, orderComplete, existingOrders, myID)

	// ────────────────────────────────
	// Step 8: HRA Coordinator (load balancing of hall calls)
	// ────────────────────────────────
	go hra.Coordinator(allElevators, existingOrders, myID, assignments, elevGet)

	// ────────────────────────────────
	// Step 9: Broadcast Initial Elevator State
	// ────────────────────────────────
	elevTx <- initial

	select {} // Prevent main from exiting
}
