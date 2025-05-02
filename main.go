package main

import (
	"elevatorlab/backup"
	"elevatorlab/common"
	"elevatorlab/core"
	"elevatorlab/elevio"
	"elevatorlab/hra"
	"elevatorlab/network/localip"
	"elevatorlab/network/peers"
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	HIGH_CHAN_SIZE int = 10
	LOW_CHAN_SIZE  int = 3
)

func main() {
	channels := initializeChannels()
	myID, port := getElevatorID()
	initial := initializeElevator(port, myID)
	elevio.SetFloorIndicator(initial.Floor)
	startGoroutines(myID, initial, channels)
	channels.elevTxChan <- initial
	select {}
}

func getElevatorID() (string, string) {
	var myID, port string
	flag.StringVar(&myID, "id", "", "Elevator ID to use")
	flag.StringVar(&port, "port", "15657", "Port to use for I/O device")
	flag.Parse()
	if myID == "" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Could not get local IP:", err)
			os.Exit(1)
		}
		myID = fmt.Sprintf("%s-%d", ip, os.Getpid())
		fmt.Println("Auto-generated ID:", myID)
	}
	fmt.Println("Starting elevator with ID:", myID)
	return myID, port
}

func initializeElevator(port, myID string) common.Elevator {
	elevio.Init("localhost:"+port, elevio.N_FLOORS)
	initial := common.Elevator{
		ID:                  myID,
		Floor:               elevio.GetFloor(),
		Dirn:                elevio.MD_Stop,
		Behaviour:           common.IDLE,
		ClearRequestVariant: common.CV_InDirn,
		DoorOpenDuration:    3 * time.Second,
	}
	_, err := os.Stat(backup.BackupFile)
	if err == nil {
		backup.LoadCabRequests(&initial)
		if initial.Floor == -1 {
			elevio.SetMotorDirection(initial.Dirn)
			for {
				if f := elevio.GetFloor(); f != -1 {
					elevio.SetMotorDirection(initial.Dirn)
					initial.Floor = f
					break
				}
			}
		}
	} else {
		if initial.Floor == -1 {
			elevio.SetMotorDirection(elevio.MD_Down)
			for {
				if f := elevio.GetFloor(); f != -1 {
					elevio.SetMotorDirection(elevio.MD_Down)
					initial.Floor = f
					break
				}
			}
		}
	}
	var initializeLights [4][2]common.OrderState
	core.UpdateAllLights(initial, initializeLights)
	return initial
}

type Channels struct {
	hallButtonPressChan chan elevio.ButtonEvent
	orderCompleteChan   chan elevio.ButtonEvent
	existingOrdersChan  chan [common.N_FLOORS][2]bool
	allElevatorsChan    chan map[string]common.Elevator
	fromHRAChan         chan common.Elevator
	elevTxChan          chan common.Elevator
	peerTxEnableChan    chan bool
	elevSetChan         chan core.ElevSetMsg
	elevGetChan         chan core.ElevGetMsg
}

func initializeChannels() Channels {
	return Channels{
		hallButtonPressChan: make(chan elevio.ButtonEvent, LOW_CHAN_SIZE),
		orderCompleteChan:   make(chan elevio.ButtonEvent, LOW_CHAN_SIZE),
		existingOrdersChan:  make(chan [common.N_FLOORS][2]bool, HIGH_CHAN_SIZE),
		allElevatorsChan:    make(chan map[string]common.Elevator, LOW_CHAN_SIZE),
		fromHRAChan:         make(chan common.Elevator, HIGH_CHAN_SIZE),
		elevTxChan:          make(chan common.Elevator, HIGH_CHAN_SIZE),
		peerTxEnableChan:    make(chan bool),
		elevSetChan:         make(chan core.ElevSetMsg),
		elevGetChan:         make(chan core.ElevGetMsg),
	}
}
func startGoroutines(myID string, initial common.Elevator, ch Channels) {
	go peers.Transmitter(15680, myID, ch.peerTxEnableChan)
	go core.RunStateMap(myID, initial, 1650, ch.allElevatorsChan, ch.elevSetChan, ch.elevGetChan)
	go core.InitFSM(myID, initial, ch.orderCompleteChan, ch.hallButtonPressChan, ch.fromHRAChan, ch.elevSetChan, ch.elevGetChan, ch.existingOrdersChan)
	go core.RunSynchronizer(ch.hallButtonPressChan, ch.orderCompleteChan, ch.existingOrdersChan, myID)
	go hra.Coordinator(ch.allElevatorsChan, ch.existingOrdersChan, myID, ch.fromHRAChan, ch.elevGetChan)
}
