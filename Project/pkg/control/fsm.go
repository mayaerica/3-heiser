package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/backup"

)

var (
	DoorOpenChan         = make(chan struct{})
	DoorCloseChan        = make(chan struct{})
	OrderCompleteChan    = make(chan elevio.ButtonEvent)
	ExistingOrdersChan   = make(chan [common.N_FLOORS][2]bool)
)

type AssignerMsg struct {
	Data elevio.ButtonEvent
}

var AssignerInput = make(chan AssignerMsg)


func InitFSM(myID string, initial common.Elevator) {
	backup.LoadCabRequests(&initial)
	ElevSet <- ElevSetMsg{Fn: func(m map[string]common.Elevator) {
		m[myID] = initial
	}}
	go StateMachineLoop(myID)
	go DoorFSM(DoorOpenChan, DoorCloseChan, initial.DoorOpenDuration)
}

func StateMachineLoop(myID string) {
	buttonPressChan := make(chan elevio.ButtonEvent)
	floorSensorChan := make(chan int)
	go elevio.PollButtons(buttonPressChan)
	go elevio.PollFloorSensor(floorSensorChan)

	var prevDirn elevio.Dirn = elevio.MD_Stop

	go func() {
		for orders := range ExistingOrdersChan {
			WithMyElevator(myID, func(e *common.Elevator) {
				for f := 0; f < common.N_FLOORS; f++ {
					for b := 0; b < 2; b++ {
						e.Requests[f][b] = orders[f][b]
					}
				}
			})
		}
	}()

	for {
		select {
		case btn := <-buttonPressChan:
			handleButtonPress(myID, btn, &prevDirn)

		case floor := <-floorSensorChan:
			elevio.SetFloorIndicator(floor)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Floor = floor
			})
			e := GetMyElevator(myID)
			if RequestShouldStop(e) {
				StopElevator()
				e = clearAtCurrentFloor(e, OrderCompleteChan)
				WithMyElevator(myID, func(me *common.Elevator) {
					*me = e
				})
				UpdateCabLights(e)
				openDoor(myID)
			}

		case <-DoorCloseChan:
			e := GetMyElevator(myID)
			next := ChooseDirection(e, e.Dirn)
			WithMyElevator(myID, func(e *common.Elevator) {
				e.Dirn = next.Dirn
				e.Behaviour = next.Behaviour
			})
			if next.Behaviour == common.MOVING {
				elevio.SetMotorDirection(next.Dirn)
			}
		}
	}
}

func handleButtonPress(myID string, btn elevio.ButtonEvent, prevDirn *elevio.Dirn) {
	if btn.Button == elevio.BT_Cab {
		WithMyElevator(myID, func(e *common.Elevator) {
			e.Requests[btn.Floor][btn.Button] = true
		})
		backup.SaveCabRequests(GetMyElevator(myID))
		UpdateCabLights(GetMyElevator(myID))
	} else {
		AssignerInput <- AssignerMsg{Data: btn}
	}
}

func openDoor(myID string) {
	WithMyElevator(myID, func(e *common.Elevator) {
		e.Behaviour = common.DOOR_OPEN
	})
	DoorOpenChan <- struct{}{}
}
