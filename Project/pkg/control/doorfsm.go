package control

import (
	"elevatorlab/elevio"
	"fmt"
	"time"
)

func DoorFSM(doorOpen <-chan struct{}, doorClosed chan<- struct{}, duration time.Duration) {
	obstructed := false
	obstructionChan := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionChan)


	var doorTimeout <- chan time.Time

	for {
		select {
		case <-doorOpen :
			elevio.SetDoorOpenLamp(true)
			doorTimeout = time.After(duration)

		/*case <-doorClosed :
			elevio.SetDoorOpenLamp(fqlse)
			timer := time.NewTimer(duration)*/

		case obstructed = <-obstructionChan:
			fmt.Println("here")
			if obstructed && doorTimeout != nil {
				doorTimeout = time.After(duration)
			}
		
		case <-doorTimeout:
			if obstructed {
				doorTimeout = time.After(duration)
			} else {
				elevio.SetDoorOpenLamp(false)
				doorClosed <- struct{}{}
				doorTimeout = nil
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
