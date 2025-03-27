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

	var doorTimeout <-chan time.Time

	for {
		select {
		case <-doorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimeout = time.After(duration)

		case obstructed = <-obstructionChan:
			fmt.Println("[DOORFSM] : Obstruction detected")
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
