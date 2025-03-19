package elevio

import (
	"flag"
	"net"
	"testing"
	"time"
)

func TestDirnString(t *testing.T) {
	tests := []struct {
		input    Dirn
		expected string
	}{
		{MD_Up, "Up"},
		{MD_Down, "Down"},
		{MD_Stop, "Stop"},
		{Dirn(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.input.String()
		if result != test.expected {
			t.Errorf("Dirn.String(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

// Start a mock server to handle the connection
func StartMockServer(address string) net.Listener {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic("Failed to start mock server: " + err.Error())
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return listener
}

func TestInit(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)

	port := flag.String("port", "15657", "Elevator port")
	Init("localhost:"+*port, 4)

	if !_initialized {
		t.Errorf("Init() failed, expected _initialized to be true")
	}
}

func TestSetMotorDirection(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)
	Init("localhost:15657", 4)

	tests := []struct {
		input    Dirn
		expected [4]byte
	}{
		{MD_Up, [4]byte{1, 1, 0, 0}},
		{MD_Down, [4]byte{1, 255, 0, 0}},
		{MD_Stop, [4]byte{1, 0, 0, 0}},
	}
	for _, test := range tests {
		result := SetMotorDirection(test.input)
		if result != test.expected {
			t.Errorf("SetMotorDirection(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestSetButtonLamp(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)
	Init("localhost:15657", 4)

	tests := []struct {
		button   ButtonType
		floor    int
		value    bool
		expected [4]byte
	}{
		{BT_HallUp, 1, true, [4]byte{2, 0, 1, 1}},
		{BT_HallDown, 2, false, [4]byte{2, 1, 2, 0}},
		{BT_Cab, 4, true, [4]byte{2, 2, 4, 1}},
	}

	for _, test := range tests {
		result := SetButtonLamp(test.button, test.floor, test.value)
		if result != test.expected {
			t.Errorf("SetButtonLamp(%v, %d, %v) = %v; want %v", test.button, test.floor, test.value, result, test.expected)
		}
	}
}

func TestSetFloorIndicator(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)
	Init("localhost:15657", 4)

	tests := []struct {
		input    int
		expected [4]byte
	}{
		{1, [4]byte{3, 1, 0, 0}},
		{2, [4]byte{3, 2, 0, 0}},
		{3, [4]byte{3, 3, 0, 0}},
	}
	for _, test := range tests {
		result := SetFloorIndicator(test.input)
		if result != test.expected {
			t.Errorf("SetFloorIndicator(%d) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestSetDoorOpenLamp(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)
	Init("localhost:15657", 4)

	tests := []struct {
		input    bool
		expected [4]byte
	}{
		{true, [4]byte{4, 1, 0, 0}},
		{false, [4]byte{4, 0, 0, 0}},
	}
	for _, test := range tests {
		result := SetDoorOpenLamp(test.input)
		if result != test.expected {
			t.Errorf("SetDoorOpenLamp(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestSetStopLamp(t *testing.T) {
	mockServer := StartMockServer("localhost:15657")
	defer mockServer.Close()
	time.Sleep(100 * time.Millisecond)
	Init("localhost:15657", 4)

	tests := []struct {
		input    bool
		expected [4]byte
	}{
		{true, [4]byte{5, 1, 0, 0}},
		{false, [4]byte{5, 0, 0, 0}},
	}
	for _, test := range tests {
		result := SetStopLamp(test.input)
		if result != test.expected {
			t.Errorf("SetStopLamp(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestPollButtons(t *testing.T) {
	// TO DO
}

func TestPollFloorSensor(t *testing.T) {
	// TO DO
}

func TestPollStopButton(t *testing.T) {
	// TO DO
}

func TestPollObstructionSwitch(t *testing.T) {
	// TO DO
}

func TestGetButton(t *testing.T) {
	// TO DO
}
func TestGetFloor(t *testing.T) {
	// TO DO
}

func TestGetStop(t *testing.T) {
	// TO DO
}

func TestGetObstruction(t *testing.T) {
	// TO DO
}

func TestToByte(t *testing.T) {
	tests := []struct {
		input    bool
		expected byte
	}{
		{true, 1},
		{false, 0},
	}

	for _, test := range tests {
		result := ToByte(test.input)
		if result != test.expected {
			t.Errorf("ToByte(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{1, true},
		{0, false},
	}

	for _, test := range tests {
		result := ToBool(test.input)
		if result != test.expected {
			t.Errorf("ToBool(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}
