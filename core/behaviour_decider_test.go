package core

import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"testing"
)

func TestRequestsAbove(t *testing.T) {
	e0 := common.Elevator{
		Floor: 0,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := common.Elevator{
		Floor: 1,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := common.Elevator{
		Floor: 2,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := common.Elevator{
		Floor: 3,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    common.Elevator
		expected bool
	}{
		{e0, true},
		{e1, false},
		{e2, false},
		{e3, false},
	}
	for _, test := range tests {
		result := RequestsAbove(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}

}

func TestRequestsBelow(t *testing.T) {
	e0 := common.Elevator{
		Floor: 0,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := common.Elevator{
		Floor: 1,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := common.Elevator{
		Floor: 2,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := common.Elevator{
		Floor: 3,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    common.Elevator
		expected bool
	}{
		{e0, false},
		{e1, false},
		{e2, true},
		{e3, true},
	}
	for _, test := range tests {
		result := RequestsBelow(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}
}

func TestRequestsHere(t *testing.T) {
	e0 := common.Elevator{
		Floor: 0,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := common.Elevator{
		Floor: 1,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := common.Elevator{
		Floor: 2,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := common.Elevator{
		Floor: 3,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    common.Elevator
		expected bool
	}{
		{e0, false},
		{e1, true},
		{e2, false},
		{e3, false},
	}
	for _, test := range tests {
		result := RequestsHere(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}

}

// // COMPILATION ERROR TO FIX
// func TestChooseDirection(t *testing.T) {
// 	e0 := common.Elevator{
// 		Floor: 0,
// 		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
// 			{false, false, false},
// 			{true, false, false},
// 			{false, false, false},
// 			{false, false, false},
// 		},
// 	}
// 	e1 := common.Elevator{
// 		Floor: 1,
// 		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
// 			{false, false, false},
// 			{true, false, false},
// 			{false, false, false},
// 			{false, false, false},
// 		},
// 	}
// 	e2 := common.Elevator{
// 		Floor: 2,
// 		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
// 			{false, false, false},
// 			{true, false, false},
// 			{false, false, false},
// 			{false, false, false},
// 		},
// 	}
// 	e3 := common.Elevator{
// 		Floor: 3,
// 		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
// 			{false, false, false},
// 			{true, false, false},
// 			{false, false, false},
// 			{false, false, false},
// 		},
// 	}

// 	tests := []struct {
// 		input    common.Elevator
// 		expected common.DirnBehaviourPair
// 	}{
// 		{e0, common.DirnBehaviourPair{elevio.MD_Up, common.MOVING}},
// 		{e1, common.DirnBehaviourPair{elevio.MD_Stop, common.DOOR_OPEN}},
// 		{e2, common.DirnBehaviourPair{elevio.MD_Down, common.MOVING}},
// 		{e3, common.DirnBehaviourPair{elevio.MD_Down, common.MOVING}},
// 	}
// 	for _, test := range tests {
// 		result := ChooseDirection(test.input)
// 		if result != test.expected {
// 			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
// 		}
// 	}
// }

func TestRequestShouldStop(t *testing.T) {
	e0 := common.Elevator{
		Floor: 0,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e0_stop := common.Elevator{
		Floor: 0,
		Dirn:  elevio.MD_Stop,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1_up := common.Elevator{
		Floor: 1,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1_cab := common.Elevator{
		Floor: 1,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{false, false, true},
			{false, false, false},
			{false, false, false},
		},
	}
	e2_down := common.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Down,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{false, true, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2_up := common.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{false, true, false},
			{false, false, false},
		},
	}
	e2_up2 := common.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{false, true, false},
			{false, true, false},
		},
	}
	e2_up3 := common.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [common.N_FLOORS][common.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{true, false, false},
			{false, true, false},
		},
	}

	tests := []struct {
		input    common.Elevator
		expected bool
	}{
		{e0, false},
		{e0_stop, true},
		{e1_up, true},
		{e1_cab, true},
		{e2_down, false},
		{e2_up, true},
		{e2_up2, false},
		{e2_up3, true},
	}
	for _, test := range tests {
		result := RequestShouldStop(test.input)
		if result != test.expected {
			t.Errorf("Expected request should stop of common %v to be %v, got %v", test.input, test.expected, result)
		}
	}
}

func TestShouldClearImmediately(t *testing.T) {
	e0 := common.Elevator{
		Floor:               0,
		Dirn:                elevio.MD_Up,
		ClearRequestVariant: common.CV_All,
	}

	e1 := common.Elevator{
		Floor:               1,
		Dirn:                elevio.MD_Up,
		ClearRequestVariant: common.CV_InDirn,
	}

	e2 := common.Elevator{
		Floor:               2,
		Dirn:                elevio.MD_Down,
		ClearRequestVariant: common.CV_InDirn,
	}

	e3 := common.Elevator{
		Floor:               3,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: common.CV_All,
	}

	tests := []struct {
		elevator common.Elevator
		btnFloor int
		btnType  elevio.ButtonType
		expected bool
	}{
		{e0, 0, elevio.BT_HallUp, true},    // CV_All: any request at the current floor should be cleared
		{e1, 1, elevio.BT_HallUp, true},    // CV_InDirn: moving UP and HallUp button -> should be cleared
		{e1, 1, elevio.BT_HallDown, false}, // CV_InDirn: moving UP but HallDown button -> should NOT be cleared
		{e2, 2, elevio.BT_HallDown, true},  // CV_InDirn: moving DOWN and HallDown button -> should be cleared
		{e2, 2, elevio.BT_HallUp, false},   // CV_InDirn: moving DOWN but HallUp button -> should NOT be cleared
		{e2, 2, elevio.BT_Cab, true},       // CV_InDirn: CAB button should always be cleared
		{e3, 3, elevio.BT_HallUp, true},    // CV_All: any request at the current floor should be cleared
		{e3, 3, elevio.BT_HallDown, true},  // CV_All: any request at the current floor should be cleared
	}

	for _, test := range tests {
		result := ShouldClearImmediately(test.elevator, test.btnFloor, test.btnType)
		if result != test.expected {
			t.Errorf("Expected ShouldClearImmediately(%v, %d, %v) to be %v, got %v",
				test.elevator, test.btnFloor, test.btnType, test.expected, result)
		}
	}
}

func TestClearAtCurrentFloor(t *testing.T) {
	//TODO
}
