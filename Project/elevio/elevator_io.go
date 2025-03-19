package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

var _initialized bool = false
var _numFloors int = 4
var _mtx sync.Mutex
var _conn net.Conn

type Dirn int

const (
	MD_Up   Dirn = 1
	MD_Down Dirn = -1
	MD_Stop Dirn = 0
)

func (d Dirn) String() string {
	switch d {
	case MD_Up:
		return "Up"
	case MD_Down:
		return "Down"
	case MD_Stop:
		return "Stop"
	default:
		return "Unknown"
	}
}

type ButtonType int

const (
	BT_HallUp ButtonType = iota
	BT_HallDown
	BT_Cab
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_numFloors = numFloors
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

func SetMotorDirection(dir Dirn) [4]byte {
	motorDirection := [4]byte{1, byte(dir), 0, 0}
	write(motorDirection)
	return motorDirection
}

func SetButtonLamp(button ButtonType, floor int, value bool) [4]byte {
	fmt.Println("hey")
	buttonLamp := [4]byte{2, byte(button), byte(floor), ToByte(value)}
	write(buttonLamp)
	return buttonLamp
}

func SetFloorIndicator(floor int) [4]byte {
	floorIndicator := [4]byte{3, byte(floor), 0, 0}
	write(floorIndicator)
	return floorIndicator
}

func SetDoorOpenLamp(value bool) [4]byte {
	doorOpenLamp := [4]byte{4, ToByte(value), 0, 0}
	write(doorOpenLamp)
	return doorOpenLamp
}

func SetStopLamp(value bool) [4]byte {
	stopLamp := [4]byte{5, ToByte(value), 0, 0}
	write(stopLamp)
	return stopLamp
}

func PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, _numFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < _numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := GetButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func PollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := GetFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func PollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func GetButton(button ButtonType, floor int) bool {
	a := read([4]byte{6, byte(button), byte(floor), 0})
	return ToBool(a[1])
}

func GetFloor() int {
	a := read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func GetStop() bool {
	a := read([4]byte{8, 0, 0, 0})
	return ToBool(a[1])
}

func GetObstruction() bool {
	a := read([4]byte{9, 0, 0, 0})
	return ToBool(a[1])
}

func read(in [4]byte) [4]byte {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = _conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) error {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		return fmt.Errorf("lost connection to Elevator Server: %w", err)
	}
	return nil
	// if err != nil {
	// 	panic("Lost connection to Elevator Server")
	// }
}

func ToByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func ToBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}
