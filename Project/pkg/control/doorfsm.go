package control

import (
	"elevatorlab/elevio"
	"time"
)

func DoorFSM(doorOpen <-chan struct{}, doorClosed chan<- struct{}, duration time.Duration) {
	obstructionChan := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionChan)

	for {
		select {
		case <-doorOpen:
			elevio.SetDoorOpenLamp(true)
			timer := time.NewTimer(duration)

			for {
				select {
				case obstructed := <-obstructionChan:
					if obstructed {
						timer.Reset(duration)
					}
				case <-timer.C:
					if !elevio.GetObstruction() {
						elevio.SetDoorOpenLamp(false)
						doorClosed <- struct{}{}
					} else {
						timer.Reset(duration)
					}
				}
			}
		}
	}
}
