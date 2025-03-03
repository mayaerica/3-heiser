package timer

import (
	"time"
	"fmt"
	//"github.com/golang/protobuf/ptypes/duration"
)

var timerEndTime time.Time	// Will holds the time when the timer will expire
var TimerActive bool

func GetWallTime() time.Time {
	return time.Now()
}

func Start(timer *time.Timer, timer_start chan time.Duration) {
	for{
		select{
		case duration:=<-timer_start:

			fmt.Print()
			//fmt.Println("duration: ", duration)	
			timer.Reset(duration)

			//timerEndTime = time.Now().Add(time.Duration(duration * (time.Second)))
			//TimerActive = true
}
		}	
	}
	

func Stop() {
	TimerActive = false
}

// Checks if the timer has expiredSetMotorDirection
func TimedOut() bool {
	if !TimerActive {
		return false
	}
	
	return time.Now().After(timerEndTime)
	
}
