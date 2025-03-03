package main

import (
	tcp "Driver-go/TCP"
	el "Driver-go/elevator"
	eiod "Driver-go/elevator_io_device"
	eio "Driver-go/elevio"
	fsm "Driver-go/fsm"
	req "Driver-go/requests"
	res "Driver-go/resource"
	rm "Driver-go/resource"
	timer "Driver-go/timer"
	"fmt"
	"time"
)

const numElevators = 3

func main() {

	//message sendt to other elevators
	msg := tcp.Message{
		Elevator: fsm.Elevator,
		Active1:  false,
		Active2:  false,
		Active3:  false,
	}

	//////////////////REQUEST///////////////////////////

	//var elevatorStatusChan = make(chan el.Elevator, numElevators*5)
	var requestChan = make(chan req.Request, numElevators*5)
	var assignChan = make(chan req.Request, numElevators*5)

	// Example of elevator instances

	// Start request dispatcher

	// Simulate new requests
	requestChan <- req.Request{Floor: 2, HandledBy: 0}
	requestChan <- req.Request{Floor: 5, HandledBy: 0}

	time.Sleep(5 * time.Millisecond) // Allow processing time

	// Elevator identification (e.g., "8081", "8082", "8083")

	const maxDuration time.Duration = 1<<63 - 1

	//var elevator Elevator
	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000 * time.Millisecond
	fsm.Elevator.Behaviour = el.IDLE
	fsm.Elevator.Dirn = eio.MD_Stop
	fsm.Elevator.Id = 8081

	eio.Init("localhost:15657", eiod.N_FLOORS) //15657

	rm.UpdateElevators(tcp.GetLastReceivedMessages())

	//var d elevio.MotorDirection = elevio.MD_Up

	//elevio.SetMotorDirection(d)

	BtnEventChan := make(chan eio.ButtonEvent)
	FloorChan := make(chan int)
	ObstructionChan := make(chan bool)
	StopChan := make(chan bool)
	TimerStartChan := make(chan time.Duration)
	//timer_out:=make(chan bool)

	maintimer := time.NewTimer(time.Duration(fsm.Elevator.DoorOpenDuration))
	maintimer.Stop()

	go res.ResourceManager(requestChan, assignChan)
	go tcp.StartServer()
	go eio.PollButtons(BtnEventChan)
	go eio.PollFloorSensor(FloorChan)
	go eio.PollObstructionSwitch(ObstructionChan)
	go eio.PollStopButton(StopChan)
	go timer.Start(maintimer, TimerStartChan)
	x := <-FloorChan

	ticker := time.NewTicker(tcp.UpdateIntervall) // Adjust time as needed
	defer ticker.Stop()

	/*if x == -1 {
	    x = <-drv_floors
	    eio.SetMotorDirection(1)
	    fmt.Printf("x=-1")
	}*/

	if x != -1 {
		eio.SetMotorDirection(0)
		eio.SetFloorIndicator(x)
		fsm.Elevator.Behaviour = el.IDLE //REMOVE THIS
		fsm.Elevator.Floor = x

	}

	//send req to reqchan
	//assignChan har request.Handeled by

	for {

		select {
		case <-ticker.C:

			tcp.SendMessage(msg)
			messages := tcp.GetLastReceivedMessages()
			tcp.PrintLastReceivedMessages(messages) //enable this for nice print of every elevator state
			//fmt.Println("Flur",<-FloorChan)
			fmt.Println(<-BtnEventChan)
			rm.UpdateElevators(messages)

		case a := <-BtnEventChan:
			//fmt.Printf("%+vfor{\n", a)
			eio.SetButtonLamp(a.Button, a.Floor, true)
			//if !fsm.Elevator.Requests[a.Floor][a.Button]{

			//}
			//fmt.Println("button set")
			fsm.Elevator.Requests[a.Floor][a.Button] = true
			//fmt.Println("request in queue")
			//fmt.Println(fsm.Elevator.Requests)
			fsm.OnRequestButtonPress(a.Floor, a.Button, TimerStartChan)
			//fmt.Println(fsm.Elevator.Behaviour)

		case a := <-FloorChan:
			//fmt.Printf("floor %+v\n", a)

			eio.SetFloorIndicator(a)
			//requests_clearAtCurrentFloor(elevator)

			if a != -1 {
				//fmt.Printf("floor %+v\n", a)
				fsm.OnFloorArrival(a, TimerStartChan)
			}

		case a := <-ObstructionChan: //this should be the right solution for obstruction
			fmt.Printf("obstruction: %+v\n", a)
			if a {
				//  onDoorTimeout(timer_start)
				if fsm.Elevator.Behaviour == el.DOOR_OPEN {
					TimerStartChan <- maxDuration
				}
				//eio.SetMotorDirection(eio.MD_Stop)
			} else {
				//eio.SetMotorDirection()
				TimerStartChan <- fsm.Elevator.DoorOpenDuration
			}

		case a := <-StopChan:
			fmt.Printf("%+v\n", a)
			for f := 0; f < eiod.N_FLOORS; f++ {
				for b := eio.ButtonType(0); b < 3; b++ {
					eio.SetButtonLamp(b, f, false)
				}
			}

		case <-maintimer.C:
			//fmt.Println("On door timeout")
			fsm.OnDoorTimeout(TimerStartChan)
		}
	}
}
