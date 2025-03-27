package control

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

// UpdateAllLights sets cab and hall lights based on current state
func UpdateAllLights(e common.Elevator, hall [common.N_FLOORS][2]common.OrderState) {
	UpdateCabLights(e)
	UpdateHallLightsFromPerspective(hall)
}

// UpdateCabLights reflects the elevator's own cab requests
func UpdateCabLights(e common.Elevator) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, e.Requests[floor][elevio.BT_Cab])
	}
}

// UpdateHallLightsFromPerspective sets hall button lights based on shared perspective
func UpdateHallLightsFromPerspective(perspective [common.N_FLOORS][2]common.OrderState) {
	for floor := 0; floor < common.N_FLOORS; floor++ {
		for btn := 0; btn < 2; btn++ { // Only HallUp and HallDown
			shouldBeLit :=
				perspective[floor][btn] == common.SeenBySomeone ||
					perspective[floor][btn] == common.SeenByEveryone

			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, shouldBeLit)
		}
	}
}
