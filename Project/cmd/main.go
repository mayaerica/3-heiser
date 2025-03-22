package main

import (
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/network/localip"
	"fmt"
	"os"
	"strconv"
	"flag"
)

func main() {
	id:= flag.String("id","","Elevator ID")
	port:= flag.String("port","15657","Elavator port")
	flag.Parse()

	if *id == ""{
		fmt.Println("")
		os.Exit(1)
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		os.Exit(1)
	}
	fmt.Printf("Elevator %s running on IP: %s\n", *id, localIP, *port)

	elevatorID, err:= strconv.Atoi(*id)
	if err != nil {
		fmt.Println("Invalid elevator ID:")
		os.Exit(1)
	}


	elevio.Init("localhost:" + *port, 1)
	
	dispatcher.InitDispatcher(elevatorID)
	go dispatcher.StartDispatcherLoop(
		control.HallCallRequestChan,
		control.AssignedHallCallChan,
		elevatorID,
	)

	control.InitFSM(*id)

	select {}
}