package requests

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
	"testing"
)

func TestRequestsAbove(t *testing.T) {
	e0 := elevator.Elevator{
		Floor: 0,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := elevator.Elevator{
		Floor: 1,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := elevator.Elevator{
		Floor: 2,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := elevator.Elevator{
		Floor: 3,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    elevator.Elevator
		expected bool
	}{
		{e0, true},
		{e1, false},
		{e2, false},
		{e3, false},
	}
	for _, test := range tests {
		result := requestsAbove(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}

}

func TestRequestsBelow(t *testing.T) {
	e0 := elevator.Elevator{
		Floor: 0,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := elevator.Elevator{
		Floor: 1,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := elevator.Elevator{
		Floor: 2,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := elevator.Elevator{
		Floor: 3,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    elevator.Elevator
		expected bool
	}{
		{e0, false},
		{e1, false},
		{e2, true},
		{e3, true},
	}
	for _, test := range tests {
		result := requestsBelow(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}
}

func TestRequestsHere(t *testing.T) {
	e0 := elevator.Elevator{
		Floor: 0,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := elevator.Elevator{
		Floor: 1,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := elevator.Elevator{
		Floor: 2,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := elevator.Elevator{
		Floor: 3,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    elevator.Elevator
		expected bool
	}{
		{e0, false},
		{e1, true},
		{e2, false},
		{e3, false},
	}
	for _, test := range tests {
		result := requestsHere(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}

}

func TestChooseDirection(t *testing.T) {
	e0 := elevator.Elevator{
		Floor: 0,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1 := elevator.Elevator{
		Floor: 1,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2 := elevator.Elevator{
		Floor: 2,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e3 := elevator.Elevator{
		Floor: 3,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}

	tests := []struct {
		input    elevator.Elevator
		expected DirnBehaviourPair
	}{
		{e0, DirnBehaviourPair{elevio.MD_Up, elevator.MOVING}},
		{e1, DirnBehaviourPair{elevio.MD_Stop, elevator.DOOR_OPEN}},
		{e2, DirnBehaviourPair{elevio.MD_Down, elevator.MOVING}},
		{e3, DirnBehaviourPair{elevio.MD_Down, elevator.MOVING}},
	}
	for _, test := range tests {
		result := ChooseDirection(test.input)
		if result != test.expected {
			t.Errorf("Expected request above at floor %v to be %v, got %v", test.input.Floor, test.expected, result)
		}
	}
}

func TestRequestShouldStop(t *testing.T) {
	e0 := elevator.Elevator{
		Floor: 0,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e0_stop := elevator.Elevator{
		Floor: 0,
		Dirn:  elevio.MD_Stop,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1_up := elevator.Elevator{
		Floor: 1,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{true, false, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e1_cab := elevator.Elevator{
		Floor: 1,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{false, false, true},
			{false, false, false},
			{false, false, false},
		},
	}
	e2_down := elevator.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Down,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{false, true, false},
			{false, false, false},
			{false, false, false},
		},
	}
	e2_up := elevator.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{false, true, false},
			{false, false, false},
		},
	}
	e2_up2 := elevator.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{false, true, false},
			{false, true, false},
		},
	}
	e2_up3 := elevator.Elevator{
		Floor: 2,
		Dirn:  elevio.MD_Up,
		Requests: [elevator.N_FLOORS][elevator.N_BUTTONS]bool{
			{false, false, false},
			{false, false, false},
			{true, false, false},
			{false, true, false},
		},
	}

	tests := []struct {
		input    elevator.Elevator
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
			t.Errorf("Expected request should stop of elevator %v to be %v, got %v", test.input, test.expected, result)
		}
	}
}

func TestShouldClearImmediately(t *testing.T) {
	e0 := elevator.Elevator{
		Floor:               0,
		Dirn:                elevio.MD_Up,
		ClearRequestVariant: elevator.CV_All,
	}

	e1 := elevator.Elevator{
		Floor:               1,
		Dirn:                elevio.MD_Up,
		ClearRequestVariant: elevator.CV_InDirn,
	}

	e2 := elevator.Elevator{
		Floor:               2,
		Dirn:                elevio.MD_Down,
		ClearRequestVariant: elevator.CV_InDirn,
	}

	e3 := elevator.Elevator{
		Floor:               3,
		Dirn:                elevio.MD_Stop,
		ClearRequestVariant: elevator.CV_All,
	}

	tests := []struct {
		elevator elevator.Elevator
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
		result := ShouldClearImmediatley(test.elevator, test.btnFloor, test.btnType)
		if result != test.expected {
			t.Errorf("Expected ShouldClearImmediately(%v, %d, %v) to be %v, got %v",
				test.elevator, test.btnFloor, test.btnType, test.expected, result)
		}
	}
}

func TestClearAtCurrentFloor(t *testing.T) {
	//TODO
	return
}
