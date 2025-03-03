package elevator_io_device

import (
	eio "Driver-go/elevio"
)
const N_FLOORS = 4
const N_BUTTONS = 3

// Direction constants (similar to enum in C)


// ElevInputDevice struct (with function types as fields)
type ElevInputDevice struct {
	floorSensor   func() int
	requestButton func(floor int, btn eio.ButtonType) bool
	stopButton    func() bool
	obstruction   func() bool
}


// ElevOutputDevice struct (with function types as fields)
type ElevOutputDevice struct {
	floorIndicator     func(floor int)
	requestButtonLight func(floor int, btn eio.ButtonType, light bool)
	doorLight          func(light bool)
	stopButtonLight    func(light bool)
	motorDirection     func(direction eio.Dirn)
}








// ----------------------------c file----------------------

func _wrap_requestButton(f int, b eio.ButtonType) bool { //
	return eio.GetButton(b, f)
}
func _wrap_requestButtonLight(f int, b eio.ButtonType, v bool) { //
	eio.SetButtonLamp(b, f, v)
}
func _wrap_motorDirection(d eio.Dirn) { //
	eio.SetMotorDirection(d)
}

func elevio_getInputDevice() ElevInputDevice {
	// Return an ElevInputDevice struct with the function fields initialized
	return ElevInputDevice{
		floorSensor:    eio.GetFloor,
		requestButton:  _wrap_requestButton,
		stopButton:     eio.GetStop, //elevator_hardware_get_stop_signal
		obstruction:    eio.GetObstruction,//elevator_hardware_get_obstruction_signal,
	}
}



func elevio_getOutputDevice() ElevOutputDevice {
	// Return an ElevOutputDevice struct with the function fields initialized
	return ElevOutputDevice{
		floorIndicator:     eio.SetFloorIndicator,
		requestButtonLight: _wrap_requestButtonLight,
		doorLight:          eio.SetDoorOpenLamp,
		stopButtonLight:    eio.SetStopLamp,
		motorDirection:     _wrap_motorDirection,
	}
}



func elevioDirnToString(d eio.Dirn) string {
	if d == eio.MD_Up {
		return "D_Up"
	} else if d == eio.MD_Down {
		return "D_Down"
	} else if d == eio.MD_Stop {
		return "D_Stop"
	}
	return "D_UNDEFINED"
}

func elevioButtonToString(b eio.ButtonType) string {
	if b == eio.BT_HallUp {
		return "B_HallUp"
	} else if b == eio.BT_HallDown {
		return "B_HallDown"
	} else if b == eio.BT_Cab {
		return "B_Cab"
	}
	return "B_UNDEFINED"
}
