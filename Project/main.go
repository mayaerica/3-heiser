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
	"elevatorlab/timer"
	"fmt"
	"os"
	"time"
	"flag"

)


/*
Hanled by, requests, done and hallcalls are handled twice. Once by resourcemanager and once by UpdateElevatorHallCallsAndButtonLamp. 
This should be fixed so that only one of them needs to set a value false when needed.
*/
var UpdateInterval = 100 * time.Millisecond

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
	fmt.Printf("Elevator %s running on IP: %s\n", *id, localIP)

	elevatorID := *id

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
		time.Sleep(100*time.Millisecond) 
		//allow the elevator to move before checing the floor again
		       
	}
	 
	init = false
	elevio.SetMotorDirection(elevio.MD_Stop)

	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000*time.Millisecond
	fsm.Elevator.Behaviour = elevator.IDLE
	fsm.Elevator.Dirn = elevio.MD_Stop
	fsm.Elevator.Id = elevatorID //this is converted back and forth, what spaghetti code hehe
	messageProcessing.ElevatorStatus[elevatorID]=true
	
	maintimer := time.NewTimer(time.Duration(fsm.Elevator.DoorOpenDuration))
	maintimer.Stop()
	
	

	//channels
	//updateHallCallsChan := make(chan resource.HallCallUpdate)
	//updateHandledByChan := make(chan resource.HandledByUpdate)
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
	go timer.Start(maintimer, TimerStartChan)
	go resource.RequestUpdater(TimerStartChan)


	ticker := time.NewTicker(UpdateInterval)
	defer ticker.Stop()
	
	

	//continously updating messages:
	go messageProcessing.UpdateMessage(peerUpdateCh, messageTx)

	elevio.SetDoorOpenLamp(false) //I am unsure why this is needed but it works. When initializing on first floor without this, door light is on.
	//event loop
	
	for {
		// fmt.Println("still alive")
		select {
		case btn := <-BtnEventChan:
			// fmt.Println("1")
			if btn.Button != elevio.BT_Cab {
				if !fsm.Elevator.HallCalls[btn.Floor][btn.Button] {
					requestChan <- requests.Request{FloorButton: btn, HandledBy: -1}
					fsm.Elevator.HallCalls[btn.Floor][btn.Button] = true
					elevio.SetButtonLamp(btn.Button, btn.Floor, true)
					}
				} else {
					elevio.SetButtonLamp(btn.Button, btn.Floor, true)
					fsm.Elevator.Requests[btn.Floor][btn.Button] = true
					fsm.OnRequestButtonPress(btn.Floor, btn.Button, TimerStartChan)
				}
				
		case floor := <-FloorChan:
			// fmt.Println("2")
			elevio.SetFloorIndicator(floor)
			if floor != -1 {
				fsm.OnFloorArrival(floor, TimerStartChan)
			}

		case obstructed := <-ObstuctionChan:
			// fmt.Println("3")
			if obstructed && fsm.Elevator.Behaviour == elevator.DOOR_OPEN { 
				TimerStartChan <- time.Duration(1<<63 - 1)	
			} else{
				TimerStartChan <- fsm.Elevator.DoorOpenDuration
			}
		case <-StopChan:
			// fmt.Println("4")
			for f:=0; f < elevator.N_FLOORS; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}

		
		case <-maintimer.C:
			fmt.Println("5")
			fsm.OnDoorTimeout(TimerStartChan)

		
		case <-ticker.C:
			// fmt.Println("6")
			resource.PrintElevators()
	

		case msg := <-messageRx:
			// fmt.Println("7")
			//UpdateElevatorHallCallsAndButtonLamp is the only thing updating requests so if no messages are recieved the system doesnt work
			//fmt.Println("\n\n\n\n\n 7 \n\n\n\n\n")
			resource.UpdateElevatorHallCallsAndButtonLamp(msg, requestChan, TimerStartChan)

		
		}
	}
}