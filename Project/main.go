package main

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
	"elevatorlab/fsm"
	"elevatorlab/requests"
	"elevatorlab/resource"
	"elevatorlab/messageProcessing"
	"elevatorlab/network/bcast"
	"elevatorlab/network/localip"
	"elevatorlab/network/peers"
	"fmt"
	"os"
	"time"
	"flag"
	"strconv"

)

var UpdateInterval = 1000 * time.Millisecond

const (
	numElevators = 3
	basePeerPort = 15000
	baseBcastPort = 16000
)

func main() {
	id:= flag.String("id","8081","Elevator ID")
	port:= flag.String("port","15657","Elavator port")
	flag.Parse()

	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		os.Exit(1)
	}
	fmt.Println("Elevator %s running on IP:", *id, localIP)

	elevatorID, err:= strconv.Atoi(*id)
	if err != nil {
		fmt.Println("Invalid elevator ID:")
		os.Exit(1)
	}

	//using UDP for communication
	peerUpdateCh := make(chan peers.PeerUpdate) 
	peerTxEnable := make(chan bool)
	messageTx := make(chan messageProcessing.Message)
	messageRx := make(chan messageProcessing.Message)

	//initializing communication for each elevator
	for i := 1; i <= numElevators; i++ {
		if fmt.Sprintf("808%d", i) != *id {
			go peers.Transmitter(basePeerPort + i, *id, peerTxEnable)
			go peers.Receiver(basePeerPort+ i, peerUpdateCh)
			go bcast.Transmitter(baseBcastPort + i, messageTx)
			go bcast.Receiver(baseBcastPort + i, messageRx)
		}
	}

	//setup elevator
	elevio.Init("localhost:" +*port, elevator.N_FLOORS)
		//button lights is turned off
		for floor := 0; floor < elevator.N_FLOORS; floor++ {
			for btn := 0; btn < elevator.N_BUTTONS; btn++ {
				elevio.SetButtonLamp(elevio.ButtonType(btn), floor, false)
			}
		}
		//moves down to reset if not on first floor
		init := true
		for elevio.GetFloor() != 0 && init {
			elevio.SetMotorDirection(elevio.MD_Down)
			//allow the elevator to move before checing the floor again
			time.Sleep(100*time.Millisecond)         
		}
		init = false
		elevio.SetMotorDirection(elevio.MD_Stop)

	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000*time.Millisecond
	fsm.Elevator.Behaviour = elevator.IDLE
	fsm.Elevator.Dirn = elevio.MD_Stop
	fsm.Elevator.Id = elevatorID


	//channels
	requestChan := make(chan requests.Request, numElevators*5)
	assignChan := make(chan requests.Request, numElevators*5)
	BtnEventChan := make(chan elevio.ButtonEvent)
	FloorChan := make(chan int)
	ObstuctionChan := make(chan bool)
	StopChan := make(chan bool)
	TimerStartChan := make(chan time.Duration)

	go resource.ResourceManager(requestChan, assignChan, TimerStartChan)
	go elevio.PollButtons(BtnEventChan)
	go elevio.PollFloorSensor(FloorChan)
	go elevio.PollObstructionSwitch(ObstuctionChan)
	go elevio.PollStopButton(StopChan)

	ticker := time.NewTicker(UpdateInterval)
	defer ticker.Stop()

	//event loop
	for {
		select {
		case btn := <-BtnEventChan:
			if btn.Button != elevio.BT_Cab {
				if !fsm.Elevator.HallCalls[btn.Floor][btn.Button] {
					requestChan <- requests.Request{FloorButton: btn, HandledBy: -1}
					fsm.Elevator.HallCalls[btn.Floor][btn.Button] = true
					elevio.SetButtonLamp(btn.Button, btn.Floor, true)
					}
				} else {
					elevio.SetButtonLamp(btn.Button, btn.Floor, true)
					fsm.Elevator.Requests[btn.Floor][btn.Button] = true
				}
		case floor := <-FloorChan:
			elevio.SetFloorIndicator(floor)
			if floor != -1 {
				fsm.OnFloorArrival(floor, TimerStartChan)
			}
		case obstructed := <-ObstuctionChan:
			if obstructed && fsm.Elevator.Behaviour == elevator.DOOR_OPEN {
				TimerStartChan <- time.Duration(1<<63 - 1)	
			} else{
				TimerStartChan <- fsm.Elevator.DoorOpenDuration
			}
		case <-StopChan:
			for f:=0; f < elevator.N_FLOORS; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}
		
		case <-ticker.C:
			resource.PrintElevators()

		case msg := <-messageRx:
			resource.UpdateElevatorHallCallsAndButtonLamp(msg, requestChan)
		}
	}
}