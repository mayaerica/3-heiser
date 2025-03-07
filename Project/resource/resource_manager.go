package resource

import (
	"elevatorlab/elevator"
	"elevatorlab/elevio"
	"elevatorlab/fsm"
	"elevatorlab/messageProcessing"
	"elevatorlab/requests"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv" 
	"time"
)

var ButtonRequestList [4][2]requests.Request

var elevators = []elevator.Elevator{
	{Id: 8081, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8081
	{Id: 8082, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8082
	{Id: 8083, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8083
}

//var printElevatorCounter int

// Function to print elevator information in a similar format as PrintLastReceivedMessages
func PrintElevators() {
	for _, elevator := range elevators {
		message := messageProcessing.Message{
			Elevator: elevator,
			Active1: messageProcessing.ElevatorStatus[8081],
			Active2: messageProcessing.ElevatorStatus[8082],
			Active3: messageProcessing.ElevatorStatus[8083],
			//Requests: elevator.Requests,
		}
		messageProcessing.PrintLastReceivedMessages(message)
	}
}

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

// State of elevator
type HRAElevState struct {
	// DON'T REMOVE `json:"..."`
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"` 
}

// State of the system
type HRAInput struct {
	// List of request for each floor : [2]bool represents 2 buttons [Up, Down]
	HallRequests [][2]bool `json:"hallRequests"`
	// Dict of the state of each el, identified with key "one", "two", ...
	States map[string]HRAElevState `json:"states"`
}

func getHRAExecutable() string {
	switch runtime.GOOS {
	case "linux":
		return "hall_request_assigner"
	case "windows":
		return "hall_request_assigner.exe"
	default:
		panic("OS non supporté")
	}
}

func ProcessElevatorRequests(input HRAInput) (map[string][][2]bool, error) {
	hraExecutable := getHRAExecutable()

	// Conversion into JSON
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %v", err)
	}

	// Execution of hall_request_assigner
	ret, err := exec.Command("./hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %v\nOutput: %s", err, string(ret))
	}

	// Conversion from JSON output to go structure
	var output map[string][][2]bool // Map linking each el to a list of requests
	err = json.Unmarshal(ret, &output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %v", err)
	}
	return output, nil
}

func ConvertRequestToHRAInput(elevators []elevator.Elevator) HRAInput {
	hraStates := make(map[string]HRAElevState)

	var hallRequests = make([][2]bool, 4) // Initialize hallRequests to match the number of floors
	var cabRequests = make([]bool, 4)     // Initialize cabRequests to match the number of floors

	// Iterate through the rows of the Requests matrix (floor requests)
	for _, _elevator := range elevators {
		// Floor requests for the elevator
		requests := _elevator.Requests
		hallCalls := _elevator.HallCalls

		// Iterate through the rows of the Requests matrix (floor requests)
		for i := 0; i < len(requests); i++ {
			// Accumulate the hall requests (external floor button presses)
			hallRequests[i][0] = hallRequests[i][0] || hallCalls[i][0] // OR the "up" request
			hallRequests[i][1] = hallRequests[i][1] || hallCalls[i][1] // OR the "down" request

			// Accumulate the cab requests (internal elevator button presses)
			cabRequests[i] = cabRequests[i] || requests[i][2] // OR the "cab" request (internal elevator button)
		}

		// Direction
		var direction string
		switch _elevator.Dirn {
		case elevio.Dirn(0):
			direction = "stop"
		case elevio.Dirn(1):
			direction = "up"
		case elevio.Dirn(2):
			direction = "down"
		default:
			direction = "stop"
		}

		// Behaviour
		var behavior string
		switch _elevator.Behaviour {
		case elevator.IDLE:
			behavior = "idle"
		case elevator.DOOR_OPEN:
			behavior = "doorOpen"
		case elevator.MOVING:
			behavior = "moving"
		default:
			behavior = "unknown"
		}

		hraStates[fmt.Sprintf("%d", _elevator.Id)] = HRAElevState{
			Behavior:    behavior,
			Floor:       _elevator.Floor,
			Direction:   direction,
			CabRequests: cabRequests, // Placeholder
		}
	}

	return HRAInput{
		HallRequests: hallRequests,
		States:       hraStates,
	}
}

func UpdateElevatorHallCallsAndButtonLamp(msg messageProcessing.Message, requestChan chan requests.Request)  {
	id := msg.Elevator.Id
	if id != fsm.Elevator.Id{
		elevators[id-8081] = msg.Elevator
	}
	
	for floor := 0; floor < 4; floor++ { 
		for button := 0; button < 2; button++ {
			// Only send request if the value is true
			if msg.Elevator.HallCalls[floor][button] && !fsm.Elevator.HallCalls[floor][button]{ 
				//Check if request at floor and request not alreadu accounted for
				fsm.Elevator.HallCalls[floor][button] = true
				elevio.SetButtonLamp(elevio.ButtonType(button), floor, true) //sets hallcall light

				// Sends requests from other elevators to requestchan and lights up button
				requestChan <- requests.Request{
					FloorButton: elevio.ButtonEvent{Button: elevio.ButtonType(button), Floor:  floor},
					HandledBy: -1,
					} 
			
			//removes HallCall if request is done
			} else if msg.Elevator.Done[floor][button] {
				fsm.Elevator.HallCalls[floor][button] = false
				elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
			}

			//toggles own Done if all elevators have removed HallCall
			if fsm.Elevator.Done[floor][button] && !elevators[0].HallCalls[floor][button] && !elevators[1].HallCalls[floor][button] && !elevators[2].HallCalls[floor][button] {
				fsm.Elevator.Done[floor][button] = false
			}
		}
	}
	//updates elevators with self
	elevators[fsm.Elevator.Id-8081] = fsm.Elevator
}


func ResourceManager(requestChan chan requests.Request, assignChan chan requests.Request, TimerStartChan chan time.Duration) {
	for {
		input := ConvertRequestToHRAInput(elevators)

		// Call function to process requests
		output, err := ProcessElevatorRequests(input)
		if err != nil {
			fmt.Println("Error :", err)
			return
		}

		//Copies requests from output to elevator requests:
		for floor := 0; floor < 4; floor++ {
			for btn := 0; btn < 2; btn++ {
				if output[strconv.Itoa(fsm.Elevator.Id)][floor][btn] && !fsm.Elevator.Requests[floor][btn]{
					fsm.Elevator.Requests[floor][btn] = output[strconv.Itoa(fsm.Elevator.Id)][floor][btn]
					fsm.Elevator.Requests[floor][btn] = true
					fsm.OnRequestButtonPress(floor, elevio.ButtonType(btn), TimerStartChan)
				}
			} 
		}
	}
}
