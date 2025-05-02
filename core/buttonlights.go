package core

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

func UpdateAllLights(e common.Elevator, hall [common.N_FLOORS][2]common.OrderState) {
	UpdateCabLights(e)
	UpdateHallLightsFromPerspective(hall)
}

func UpdateCabLights(e common.Elevator) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, e.Requests[floor][elevio.BT_Cab])
	}
}

func UpdateHallLightsFromPerspective(perspective [common.N_FLOORS][2]common.OrderState) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ {
			shouldBeLit :=
				perspective[floor][btn] == common.SEEN_BY_SOMEONE ||
					perspective[floor][btn] == common.SEEN_BY_EVERYONE

			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, shouldBeLit)
		}
	}
}
