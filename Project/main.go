package main
import (
	"elevatorlab/common"
	"elevatorlab/elevio"
	"elevatorlab/pkg/control"
	"elevatorlab/pkg/hra"
	"elevatorlab/pkg/network/localip"
	"elevatorlab/pkg/network/peers"
	"flag"
	"fmt"
	"os"
	"time"
)
func main() {
	channels := initializeChannels()
	myID, port := getElevatorID()
	initial := initializeElevator(port, myID)
	var initializeLights [4][2] common.OrderState
	control.UpdateAllLights(initial, initializeLights)
	startGoroutines(myID, initial, channels)
	channels.elevTx <- initial
	select {}
}
func getElevatorID() (string, string) {
	var myID, port string
	flag.StringVar(&myID, "id", "", "Elevator ID to use")
	flag.StringVar(&port, "port", "15657", "Port to use for I/O device")
	flag.Parse()
	if myID == "" {
		ip, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Could not get local IP:", err)
			os.Exit(1)
		}
		myID = fmt.Sprintf("%s-%d", ip, os.Getpid())
		fmt.Println("Auto-generated ID:", myID)
	}
	fmt.Println("Starting elevator with ID:", myID)
	return myID, port
}
func initializeElevator(port, myID string) common.Elevator {
	elevio.Init("localhost:"+port, elevio.N_FLOORS)
	initial := common.Elevator{
		ID: myID,
		Floor: elevio.GetFloor(),
		Dirn: elevio.MD_Stop,
		Behaviour: common.IDLE,
		ClearRequestVariant: common.CV_InDirn,
		DoorOpenDuration: 1 * time.Second,
	}
	if initial.Floor == -1 {
		fmt.Println("[BOOT] Between floors, moving down to find one...")
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			if f := elevio.GetFloor(); f != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				initial.Floor = f
				break
			}
		}
	}
	elevio.SetFloorIndicator(initial.Floor)
	return initial
}
type Channels struct {
	hallButtonPress chan elevio.ButtonEvent
	cabButtonPress chan elevio.ButtonEvent
	orderComplete chan elevio.ButtonEvent
	existingOrders chan [common.N_FLOORS][2]bool
	allElevators chan map[string]common.Elevator
	assignments chan common.Elevator
	elevTx chan common.Elevator
	peerTxEnable chan bool
	elevSet chan control.ElevSetMsg
	elevGet chan control.ElevGetMsg
}
func initializeChannels() Channels {
	return Channels{
		hallButtonPress: make(chan elevio.ButtonEvent, 3),
		cabButtonPress: make(chan elevio.ButtonEvent, 3),
		orderComplete: make(chan elevio.ButtonEvent, 3),
		existingOrders: make(chan [common.N_FLOORS][2]bool, 10),
		allElevators: make(chan map[string]common.Elevator, 6),
		assignments: make(chan common.Elevator, 10),
		elevTx: make(chan common.Elevator, 10),
		peerTxEnable: make(chan bool),
		elevSet: make(chan control.ElevSetMsg),
		elevGet: make(chan control.ElevGetMsg),
	}
}
func startGoroutines(myID string, initial common.Elevator, ch Channels) {
	go peers.Transmitter(15680, myID, ch.peerTxEnable)
	go control.RunElevState(myID, initial, 1650, ch.allElevators, ch.elevSet, ch.elevGet)
	go control.InitFSM(myID, initial, ch.orderComplete, ch.hallButtonPress, ch.assignments, ch.elevSet, ch.elevGet, ch.existingOrders)
	go control.RunSynchronizer(ch.hallButtonPress, ch.orderComplete, ch.existingOrders, myID, ch.cabButtonPress)
	go hra.Coordinator(ch.allElevators, ch.existingOrders, myID, ch.assignments, ch.elevGet)
}
