package control

import (
	"elevatorlab/elevio"
	"time"
)

func DoorFSM(doorOpen <-chan struct{}, doorClosed chan<- struct{}, duration time.Duration) {
	obstructionChan := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionChan)

	var obstructed bool

	for {
		<-doorOpen
		elevio.SetDoorOpenLamp(true)
		timer := time.NewTimer(duration)

	doorOpenLoop:
		for {
			select {
			case obstructed = <-obstructionChan:
				timer.Reset(duration)

			case <-timer.C:
				if obstructed {
					timer.Reset(duration)
				} else {
					elevio.SetDoorOpenLamp(false)
					doorClosed <- struct{}{}
					break doorOpenLoop
				}
			}
		}
	}
}

/*	for {
		select {
		case <-doorOpen:
			elevio.SetDoorOpenLamp(true)
			timer := time.NewTimer(duration) // Timer for door closing duration

		Loop:
			for {
				select {
				case obstructed := <-obstructionChan:
					if obstructed {
						// If obstruction is detected, reset the timer to keep the door open

					} else {
						// If obstruction is cleared, close the door
						timer.Reset(duration)
					}
				case <-timer.C:
					// If the timer expired and no obstruction, close the door
					if !elevio.GetObstruction() {
						elevio.SetDoorOpenLamp(false)
						doorClosed <- struct{}{}
						break Loop // Exit the loop and stop checking further
					} else {
						timer.Reset(duration)
					}
				}
			}
		}
	}
}
*/
