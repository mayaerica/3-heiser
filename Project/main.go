package main

import (
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/network/localip"
	"fmt"
	"os"
	"flag"
)

func main() {
	id:= flag.String("id","","Elevator ID")
	port:= flag.String("port","15657","Elavator port")
	flag.Parse()

	if *id == ""{
		fmt.Println("Specify an elevator ID using -id")
		os.Exit(1)
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		os.Exit(1)
	}
	fmt.Printf("Elevator %s running on IP: %s\n", *id, localIP, *port)

	//func Init(address string, numFloors int)
	elevio.Init("localhost:" + *port, 4)
	
	control.InitDispatcher(*id)

	go control.StartDispatcherLoop(
		control.HallCallRequestChan,
		control.AssignedHallCallChan,
		*id,
	)

	control.InitFSM(*id)

	select {}
}