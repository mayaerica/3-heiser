package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

func SetAllLights(e common.Elevator) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < common.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
		}
	}
}

func UpdateHallLights(hallRequests [common.N_FLOORS][2]bool) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ { //only hall up and hall down
			elevio.SetButtonLamp(elevio.ButtonType(btnType), floor, hallRequests[floor][btnType])
		}
	}
}

func UpdateCabLights(e common.Elevator) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, e.Requests[floor][elevio.BT_Cab])
	}
}

func UpdateFloorIndicator(floor int) {
	elevio.SetFloorIndicator(floor)
}

func SetDoorOpenLamp(state bool) {
	elevio.SetDoorOpenLamp(state)
}
