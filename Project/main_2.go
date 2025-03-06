package main

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
	"elevatorlab/fsm"
	"elevatorlab/requests"
	"elevatorlab/resource"
	"elevatorlab/timer"
	"elevatorlab/messageProcessing"
	"elevatorlab/network/bcast"
	"elevatorlab/network/localip"
	"elevatorlab/network/peers"
	"fmt"
	"time"
)


var UpdateIntervall = 1000 * time.Millisecond
const numElevators = 3
var master = true

func main() {
	var id string
	//UDP

	peerUpdateCh := make(chan peers.PeerUpdate)  // Channel for updates related to peers
	peerTxEnable := make(chan bool)              // Channel to enable or disable transmission
	messageTx := make(chan messageProcessing.Message)          // Channel for transmitting "hello" messages
	messageRx := make(chan messageProcessing.Message)          // Channel for receiving "hello" messages

	//between elevators 1 and 2
	go peers.Transmitter(15012, id, peerTxEnable)
	go peers.Receiver(15012, peerUpdateCh)
	go bcast.Transmitter(16612, messageTx)
	go bcast.Receiver(16612, messageRx)
	

	//between elevators 2 and 3
	go peers.Transmitter(15023, id, peerTxEnable)
	go peers.Receiver(15023, peerUpdateCh)
	go bcast.Transmitter(16623, messageTx)
	go bcast.Receiver(16623, messageRx)
	localIP, err := localip.LocalIP()

	fmt.Println(localIP)

	if err != nil {
		fmt.Println("Error serializing message:", err)
	}


	//var elevatorStatusChan = make(chan elevator.Elevator, numElevators*5)
	var requestChan = make(chan requests.Request, numElevators*5)
	var dummyChan = make(chan requests.Request, numElevators*5)
	var assignChan = make(chan requests.Request, numElevators*5)
	dummyChan <- requests.Request{}
	


	time.Sleep(5 * time.Millisecond) // Allow processing time

	// Elevator identification (e.g., "8081", "8082", "8083")
	

	const maxDuration time.Duration = 1<<63 - 1

	//var elevator Elevator
	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000 * time.Millisecond
	fsm.Elevator.Behaviour = elevator.IDLE
	fsm.Elevator.Dirn = elevio.MD_Stop
	fsm.Elevator.Id = 8082



	go func() {
		for {
			msg := messageProcessing.Message{
				Elevator:      fsm.Elevator,              // Use the colon to assign a value to the Elevator field
				Active1:       true, 
				Active2:       true,
				Active3:       true,
				Requests: make([]requests.Request, 0), // Initialize an empty slice for ButtonRequests
			}
			messageTx <- msg

			time.Sleep(1 * time.Second)
		}
	}()


	elevio.Init("localhost:10002", elevio.N_FLOORS) //15657


	


	//var d elevio.MotorDirection = elevio.MD_Up

	//elevio.SetMotorDirection(d)

	BtnEventChan := make(chan elevio.ButtonEvent)
	FloorChan := make(chan int)
	ObstructionChan := make(chan bool)
	StopChan := make(chan bool)
	TimerStartChan := make(chan time.Duration)
	//timer_out:=make(chan bool)

	maintimer := time.NewTimer(time.Duration(fsm.Elevator.DoorOpenDuration))
	maintimer.Stop()
	
	go resource.ResourceManager(requestChan, assignChan, TimerStartChan)
	
	
	for elevio.GetFloor() != 0 {
		elevio.SetMotorDirection(-1)
		elevio.SetFloorIndicator(0)
	}
	elevio.SetMotorDirection(0)

	
	go elevio.PollButtons(BtnEventChan)
	go elevio.PollFloorSensor(FloorChan)
	go elevio.PollObstructionSwitch(ObstructionChan)
	go elevio.PollStopButton(StopChan)
	go timer.Start(maintimer, TimerStartChan)

	
	ticker := time.NewTicker(UpdateIntervall) // Adjust time as needed
	defer ticker.Stop()

	

	//turns off all lights
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			Button := elevio.ButtonType(btn)
				elevio.SetButtonLamp(Button, floor, false)
			
		}
	}

	

	//send req to reqchan
	//assignChan har request.Handeled by
	for {

		select {

		

		case a := <-BtnEventChan:
			
			//Handles local hall calls. RM handles hallcalls from other elevators
			if a.Button != elevio.BT_Cab {
				if !fsm.Elevator.HallCalls[a.Floor][a.Button] { //Checks if request already been called
					requestChan <- requests.Request{
						FloorButton: elevio.ButtonEvent{Button: elevio.ButtonType(a.Button), Floor:  a.Floor},
						HandledBy: -1,
					}
					fsm.Elevator.HallCalls[a.Floor][a.Button] = true
					elevio.SetButtonLamp(a.Button, a.Floor, true) //sets local lights for local cabin calls
				}
				
			} else {

				elevio.SetButtonLamp(a.Button, a.Floor, true)
				fsm.Elevator.Requests[a.Floor][a.Button] = true //CabinRequests are always handled locally
				}
				//Sends local request to requestChan

		// moves elevator
		//	fsm.Elevator.Requests[a.Floor][a.Button] = true
		//	fsm.OnRequestButtonPress(a.Floor, a.Button, TimerStartChan)
			

		
		case a := <-FloorChan:
			//fmt.Printf("floor %+v\n", a)

			

			elevio.SetFloorIndicator(a)
			//requests_clearAtCurrentFloor(elevator)

			if a != -1 {
				//fmt.Printf("floor %+v\n", a)
				fsm.OnFloorArrival(a, TimerStartChan)
			}

		case a := <-ObstructionChan: //this should be the right solution for obstruction
			fmt.Printf("obstruction: %+v\n", a)
			if a {
				//  onDoorTimeout(timer_start)
				if fsm.Elevator.Behaviour == elevator.DOOR_OPEN {
					TimerStartChan <- maxDuration
				}
				//elevio.SetMotorDirection(elevio.MD_Stop)
			} else {
				//elevio.SetMotorDirection()
				TimerStartChan <- fsm.Elevator.DoorOpenDuration
			}

		case a := <-StopChan:
			fmt.Printf("%+v\n", a)
			for f := 0; f < elevator.N_FLOORS; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false) 
				}
			}

		case <-maintimer.C:
			//fmt.Println("On door timeout")
			fsm.OnDoorTimeout(TimerStartChan)

		case a := <-messageRx:
			resource.UpdateElevatorHallCallsAndButtonLamp(a, requestChan) 

		case <-ticker.C:
			resource.PrintElevators()

		

		}
	}
}