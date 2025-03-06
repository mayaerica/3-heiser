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
)

var ButtonRequestList [4][2]requests.Request

var elevators = []elevator.Elevator{
	{Id: 8081, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8081
	{Id: 8082, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8082
	{Id: 8083, Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8083
}

var printElevatorCounter int

// Function to print elevator information in a similar format as PrintLastReceivedMessages
func PrintElevators() {
	fmt.Println("############################", printElevatorCounter, "##################################")
	fmt.Println("Elevator Information:")

	// Iterate through the elevators array and print details
	for _, elevator := range elevators {
		// Print elevator ID and active status
		fmt.Printf("Elevator ID: %-5d", elevator.Id)

		// Print the entire Elevator struct for the current elevator
		fmt.Printf("  Floor: %-3d | Direction: %-8s | Behaviour: %-8s | Busy: %-5t | DoorOpenDuration: %-4v | ClearRequestVariant: %-5v\n",
			elevator.Floor, elevator.Dirn, elevator.Behaviour, elevator.Busy, elevator.DoorOpenDuration, elevator.ClearRequestVariant)

		// Print the button requests (assuming this is a list of button presses)
		fmt.Println("  Button Requests: ")
		// Here we're assuming there's a 'Requests' field in the elevator
		// Adjust this based on your actual implementation if necessary
		for _, btnReq := range elevator.Requests {
			fmt.Printf("    Button %v\n", btnReq)
		}

		// Print requests for the elevator
		fmt.Println("  Requests: ")
		for i := 0; i < len(elevator.Requests); i++ {
			fmt.Printf("    Floor %d: %v\n", i+1, elevator.Requests[i])
		}

		// Add a separator line for each elevator
		fmt.Println("#################################################################")
	}

	// Increment the elevator counter
	printElevatorCounter++
}

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

// State of elevator
type HRAElevState struct {
	// DON'T REMOVE `json:"..."`
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"` // if an el has request for at a floor
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
	for _, elevator := range elevators {

		// Floor requests for the elevator
		requests := elevator.Requests

		// Iterate through the rows of the Requests matrix (floor requests)
		for i := 0; i < len(requests); i++ {
			// Accumulate the hall requests (external floor button presses)
			hallRequests[i][0] = hallRequests[i][0] || requests[i][0] // OR the "up" request
			hallRequests[i][1] = hallRequests[i][1] || requests[i][1] // OR the "down" request

			// Accumulate the cab requests (internal elevator button presses)
			cabRequests[i] = cabRequests[i] || requests[i][2] // OR the "cab" request (internal elevator button)
		}

		// Direction
		var direction string
		switch elevator.Dirn {
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
		switch elevator.Behaviour {
		case elevator.IDLE:
			behavior = "idle"
		case elevator.DOOR_OPEN:
			behavior = "doorOpen"
		case elevator.MOVING:
			behavior = "moving"
		default:
			behavior = "unknown"
		}

		hraStates[fmt.Sprintf("%d", elevator.Id)] = HRAElevState{
			Behavior:    behavior,
			Floor:       elevator.Floor,
			Direction:   direction,
			CabRequests: cabRequests, // Placeholder, vous pouvez ajuster selon les besoins
		}

	}

	return HRAInput{
		HallRequests: hallRequests,
		States:       hraStates,
	}
}

func UpdateElevatorsandRequests(msg messageProcessing.Message, requestChan chan requests.Request) {
	id := msg.Elevator.Id
	if id != fsm.Elevator.Id {
		elevators[id-8081] = msg.Elevator
	}

	for floor := 0; floor < 4; floor++ { //floor and button might need to be uiversilizersdfds if other amount of floors and button
		for button := 0; button < 3; button++ {
			// Only send request if the value is true
			if msg.Elevator.Requests[floor][button] { // Check if there is a request for this button on the floor
				// Create a Request and send it to the channel

				requestChan <- requests.Request{
					FloorButton: elevio.ButtonEvent{Button: elevio.ButtonType(button), Floor: floor},
					HandledBy:   -1,
				}
			}
		}
	}

	elevators[fsm.Elevator.Id-8081] = fsm.Elevator

}

func ResourceManager(requestChan chan requests.Request, assignChan chan requests.Request) {

	for {
		input := ConvertRequestToHRAInput(elevators) //, []requests.Request{request})

		// Call function to process requests
		output, err := ProcessElevatorRequests(input)
		if err != nil {
			fmt.Println("Erreur :", err)
			return
		}

		// Display results
		//fmt.Println("Request attributed to :")
		fmt.Println(output)
		//fmt.Println()

		// Process assignments and push requests to assignChan
		//fmt.Println(output)
		/*
			for _, req := range output {
				for _, reqPair := range req {
					assignChan <- reqPair
				}
			}*/

	}

}
