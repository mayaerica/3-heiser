package indicators

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
)

func UpdateAllLights(e common.Elevator, perspective [common.N_FLOORS][2]common.OrderState){
	UpdateCabLights(e)
	UpdateHallLightsFromPerspective(perspective)
}

//pending cab requests
func UpdateCabLights(e common.Elevator){
	for floor := 0; floor < common.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, e.Requests[floor][elevio.BT_Cab])
	}
}

//regarding hall buttons - if their order state is not NonExisting
func UpdateHallLightsFromPerspective(perspective [common.N_FLOORS][2]common.OrderState){
	for floor:=0; floor < common.N_FLOORS; floor++ {
		for btnType:=0; btnType < 2; btnType++{
			shouldBeLit := perspective[floor][btnType] != common.NonExisting
			elevio.SetButtonLamp(elevio.ButtonType(btnType), floor, shouldBeLit)
		}
	}
}
