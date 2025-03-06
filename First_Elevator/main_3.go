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



	//between elevators 2 and 3
	go peers.Transmitter(15023, id, peerTxEnable)
	go peers.Receiver(15023, peerUpdateCh)
	go bcast.Transmitter(16623, messageTx)
	go bcast.Receiver(16623, messageRx)
	

	//between elevators 1 and 3
	go peers.Transmitter(15013, id, peerTxEnable)
	go peers.Receiver(15013, peerUpdateCh)
	go bcast.Transmitter(16613, messageTx)
	go bcast.Receiver(16613, messageRx)
	localIP, err := localip.LocalIP()

	fmt.Println(localIP)

	if err != nil {
		fmt.Println("Error serializing message:", err)
	}


	//var elevatorStatusChan = make(chan elevator.Elevator, numElevators*5)
	var requestChan = make(chan requests.Request, numElevators*5)
	var dummyChan = make(chan requests.Request, numElevators*5)
	var assignChan = make(chan requests.Request, numElevators*5)


	time.Sleep(5 * time.Millisecond) // Allow processing time

	// Elevator identification (e.g., "8081", "8082", "8083")
	

	const maxDuration time.Duration = 1<<63 - 1

	//var elevator Elevator
	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000 * time.Millisecond
	fsm.Elevator.Behaviour = elevator.IDLE
	fsm.Elevator.Dirn = elevio.MD_Stop
	fsm.Elevator.Id = 8083



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


	elevio.Init("localhost:10003", eiod.N_FLOORS) //15657


	


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
	
	go resource.ResourceManager(requestChan, assignChan)
	
	
	go elevio.PollButtons(BtnEventChan)
	go elevio.PollFloorSensor(FloorChan)
	go elevio.PollObstructionSwitch(ObstructionChan)
	go elevio.PollStopButton(StopChan)
	go timer.Start(maintimer, TimerStartChan)
	x := <-FloorChan
	
	ticker := time.NewTicker(UpdateIntervall) // Adjust time as needed
	defer ticker.Stop()

	/*if x == -1 {
	    x = <-drv_floors
	    elevio.SetMotorDirection(1)
	    fmt.Printf("x=-1")
	}*/

	if x != -1 {
		elevio.SetMotorDirection(0)
		elevio.SetFloorIndicator(x)
		fsm.Elevator.Behaviour = elevator.IDLE //REMOVE THIS
		fsm.Elevator.Floor = x

	}


	//send req to reqchan
	//assignChan har request.Handeled by
	for {

		select {

		

		case a := <-BtnEventChan:
			//fmt.Printf("%+vfor{\n", a)
			elevio.SetButtonLamp(a.Button, a.Floor, true)
			
			if master {
				requestChan <-requests.Request{a,0}
			}


			fsm.Elevator.Requests[a.Floor][a.Button] = true

			fsm.OnRequestButtonPress(a.Floor, a.Button, TimerStartChan)
			
			//fmt.Println("button set")
			//fsm.Elevator.Requests[a.Floor][a.Button] = true
			//fmt.Println("request in queue")
			//fmt.Println(fsm.Elevator.Requests)
			//fsm.OnRequestButtonPress(a.Floor, a.Button, TimerStartChan)
			//fmt.Println(fsm.Elevator.Behaviour)
	/*	
		case a :=<- assignChan:
			if a.HandledBy == fsm.Elevator.Id {
				
			} else {
				msg.Requests=append(msg.Requests, a)
			}
			*/
		
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
			for f := 0; f < eiod.N_FLOORS; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}

		case <-maintimer.C:
			//fmt.Println("On door timeout")
			fsm.OnDoorTimeout(TimerStartChan)

		case a := <-messageRx:
			rm.UpdateElevatorsandRequests(a, dummyChan) // Put a dummychan here cause the recieveChan currently interferes with an elevator working alone. This can probably be removed later
		

		case <-ticker.C:
			rm.PrintElevators()

		

		}
	}
}
