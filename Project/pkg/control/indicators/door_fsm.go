package indicators

import (
	"elevatorlab/elevio"
	"time"
)

func DoorFSM(doorOpen <-chan struct{}, doorClosed chan<- struct{}, duration time.Duration) {
	obstructionChan := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionChan)

	for{
		select {
		case <-doorOpen:
			elevio.SetDoorOpenLamp(true)
			timer := time.NewTimer(duration)

			for {
				select {
				case abnormality := <-obstructionChan:
					if abnormality {
						timer.Reset(duration)
					}
				case <-timer.C:
					if !elevio.GetObstruction() {
						elevio.SetDoorOpenLamp(false)
						doorClosed <- struct{}{} //notify fsm
					} else {
						timer.Reset(duration) //stay open until obstruction is removed
					}
				}
			}
		}
	}
}
