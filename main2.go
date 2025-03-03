package main

import (
	el "Driver-go/elevator"
	eiod "Driver-go/elevator_io_device"
	eio "Driver-go/elevio"
	fsm "Driver-go/fsm"
	timer "Driver-go/timer"
	"fmt"
	"time"
)

func main() {

	const maxDuration time.Duration = 1<<63 - 1

	//var elevator Elevator
	fsm.Elevator.ClearRequestVariant = 1
	fsm.Elevator.DoorOpenDuration = 1000 * time.Millisecond
	fsm.Elevator.Behaviour = el.IDLE
	fsm.Elevator.Dirn = eio.MD_Stop

	eio.Init("localhost:15657", eiod.N_FLOORS) //15657

	for eio.GetFloor() != 1 {
		eio.SetMotorDirection(-1)
		fmt.Println("x=-1")
		if eio.GetFloor() == 0 {
			break
		}
	}

	//var d elevio.MotorDirection = elevio.MD_Up

	//elevio.SetMotorDirection(d)

	BtnEventChan := make(chan eio.ButtonEvent)
	FloorChan := make(chan int)
	ObstructionChan := make(chan bool)
	StopChan := make(chan bool)
	TimerStartChan := make(chan time.Duration)
	//timer_out:=make(chan bool)

	fmt.Println("1")

	maintimer := time.NewTimer(time.Duration(fsm.Elevator.DoorOpenDuration))
	maintimer.Stop()

	go eio.PollButtons(BtnEventChan)
	go eio.PollFloorSensor(FloorChan)
	go eio.PollObstructionSwitch(ObstructionChan)
	go eio.PollStopButton(StopChan)
	go timer.Start(maintimer, TimerStartChan)
	fmt.Println("2")
	x := <-FloorChan

	//fmt.Println(x)
	//moves the elevator to 0
	if x != -1 {
		eio.SetMotorDirection(0)
		eio.SetFloorIndicator(x)
		fsm.Elevator.Behaviour = el.IDLE
		fsm.Elevator.Floor = x

		BtnEventChan <- eio.ButtonEvent{}
	}

	for {
		select {
		case a := <-BtnEventChan:
			fmt.Printf("%+vfor{\n", a)
			eio.SetButtonLamp(a.Button, a.Floor, true)
			fmt.Println("button set")
			fsm.Elevator.Requests[a.Floor][a.Button] = true
			fmt.Println("request in queue")
			fmt.Println(fsm.Elevator.Requests)
			fsm.OnRequestButtonPress(a.Floor, a.Button, TimerStartChan)
			fmt.Println(fsm.Elevator.Behaviour)

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
				//  onDoorTimeout(TimerStartChan)
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
			fmt.Println("On door timeout")
			fsm.OnDoorTimeout(TimerStartChan)
		}
	}
}
