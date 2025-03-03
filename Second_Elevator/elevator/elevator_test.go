package elevator

import (
	"testing"
)

func TestElevator_ShouldStop(t *testing.T) {
	elevator := Elevator{
		Requests: [4][3]bool{
			{false, false, false}, // No request
			{true, false, false},  // One "up" request at floor 1
			{false, false, false}, // No request
			{false, false, false}, // No request
		},
	}

	tests := []struct {
		floor      int
		expectStop bool
	}{
		{0, false},
		{1, true},
		{2, false},
		{3, false},
	}

	for _, test := range tests {
		t.Run("TestShouldStop", func(t *testing.T) {
			stop := elevator.ShouldStop(test.floor)
			if stop != test.expectStop {
				t.Errorf("Expected stop at floor %d to be %v, got %v", test.floor, test.expectStop, stop)
			}
		})
	}
}

func TestElevator_ClearRequestsAtFloor(t *testing.T) {
	elevator := Elevator{
		Requests: [4][3]bool{
			{true, false, true},
			{false, true, false},
			{false, false, false},
			{false, false, false},
		},
	}

	elevator.ClearRequestsAtFloor(0)

	for btn := 0; btn < 3; btn++ {
		if elevator.Requests[0][btn] != false {
			t.Errorf("Expected request at floor 0 to be cleared, but got %v", elevator.Requests[0][btn])
		}
	}
}

func TestElevator_HasRequestsAbove(t *testing.T) {
	elevator := Elevator{
		Requests: [4][3]bool{
			{false, false, false}, // No request
			{true, false, false},  // Up reqest at floor 1
			{false, true, false},  // Down request at floor 2
			{false, false, false}, // No request
		},
	}

	tests := []struct {
		floor       int
		expectAbove bool
	}{
		{0, true},
		{1, true},
		{2, false},
		{3, false},
	}

	for _, test := range tests {
		t.Run("TestHasRequestsAbove", func(t *testing.T) {
			hasRequests := elevator.HasRequestsAbove(test.floor)
			if hasRequests != test.expectAbove {
				t.Errorf("Expected HasRequestsAbove for floor %d to be %v, got %v", test.floor, test.expectAbove, hasRequests)
			}
		})
	}
}

func TestElevator_HasRequestsBelow(t *testing.T) {
	elevator := Elevator{
		Requests: [4][3]bool{
			{false, false, false}, // No request
			{true, false, false},  // Up request at floor 1
			{false, true, false},  // Down request at floor 2
			{false, false, false}, // No request
		},
	}

	tests := []struct {
		floor       int
		expectBelow bool
	}{
		{0, false},
		{1, false},
		{2, true},
		{3, true},
	}

	for _, test := range tests {
		t.Run("TestHasRequestsBelow", func(t *testing.T) {
			hasRequests := elevator.HasRequestsBelow(test.floor)
			if hasRequests != test.expectBelow {
				t.Errorf("Expected HasRequestsBelow for floor %d to be %v, got %v", test.floor, test.expectBelow, hasRequests)
			}
		})
	}
}
