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
	"sync"
)


var numActiveElevators int

var mu sync.RWMutex

var mu2 sync.Mutex

var ButtonRequestList [4][2]requests.Request

var elevators = []elevator.Elevator{
	{Id: "8081", Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8081
	{Id: "8082", Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8082
	{Id: "8083", Floor: 0, Dirn: elevio.Dirn(0), Behaviour: elevator.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8083
}

var printElevatorCounter int

// Function to print elevator information in a similar format as PrintLastReceivedMessages
func PrintElevators() {
    fmt.Println("############################", printElevatorCounter, "##################################")
    fmt.Println("Elevator Information:")

    // Iterate through the elevators array and print details
    for _, elevator := range elevators {
        if elevator.Id == fsm.Elevator.Id {
			num, _ := strconv.Atoi(elevator.Id)
            elevators[num-8081] = fsm.Elevator
        }

        // Print elevator ID, floor, direction, behaviour, busy status, door open duration, clear request variant in a row
		fmt.Printf("Elevator ID: %-5d | Floor: %-3d | Direction: %-8s | Behaviour: %-8s | Busy: %-5t | DoorOpenDuration: %-4v | ClearRequestVariant: %-5v\n",
			elevator.Id, elevator.Floor, elevator.Dirn.String(), elevator.Behaviour.String(), elevator.Busy, elevator.DoorOpenDuration, elevator.ClearRequestVariant)

        // Print the floor number headers for each category, now including Handled
        fmt.Printf("  %-6v %-6v %-6v %-6v\n", "   Hall Calls           ", "Requests           ", "Done         ", "Handled")

        // Loop through all floors and print corresponding Button Requests, Requests, Done, and Handled side by side
        for i := 0; i < len(elevator.HallCalls); i++ {
            fmt.Printf("    %-6v %-6v %-6v %-6v\n", elevator.HallCalls[i], elevator.Requests[i], elevator.Done[i], elevator.HandledBy[i])
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

func UpdateFromMessage(msg messageProcessing.Message, callUpdatesChan chan requests.CallUpdate) {
    id := msg.Elevator.Id
    if id != fsm.Elevator.Id {
        for floor := 0; floor < 4; floor++ {
            for button := 0; button < 2; button++ {
				if msg.Elevator.HandledBy[floor][button]  != "Done"  && msg.Elevator.HallCalls[floor][button] && fsm.Elevator.HandledBy[floor][button] != "Done" { //adds hallcall and handled by to update handledby
						callUpdatesChan <- requests.CallUpdate{
							Floor: floor,
							Button: button,
							HandledBy: "Unchanged", 
							Delete: false,
						}
					
						
					} else if msg.Elevator.HandledBy[floor][button]  == "Done" && msg.Elevator.HallCalls[floor][button] { //removes hallcall if done
						callUpdatesChan <- requests.CallUpdate{
						Floor: floor,
						Button: button,
						HandledBy: "",
						Delete: true,
					}
				}
            }
        }
    

	
	num, error := strconv.Atoi(msg.Elevator.Id)
	if error == nil {
    	elevators[num-8081] = msg.Elevator
	} else {
		fmt.Println("Error in UpdateFromMessage: ", error)
	}
	fmt.Println("doon1)")
	}
	
}


func UpdateElevator(callUpdatesChan chan requests.CallUpdate, TimerStartChan chan time.Duration, requestUpdateChan chan requests.CallUpdate) {
	for {
		select {
		case updatedRequest := <-requestUpdateChan:
			// Locking to ensure safe access to fsm.Elevator
			requests.Mu5.Lock()
			defer requests.Mu5.Unlock()
			
			if updatedRequest.Delete {
				fsm.Elevator.Requests[updatedRequest.Floor][updatedRequest.Button] = false

				if updatedRequest.Button != 2 {
					if updatedRequest.HandledBy == "Done" {
						fmt.Println("\n\n\n\n\nDoon is doon \n\n\n\n\n")
						time.Sleep(1 * time.Second)

						fsm.Elevator.HallCalls[updatedRequest.Floor][updatedRequest.Button] = false
						fsm.Elevator.HandledBy[updatedRequest.Floor][updatedRequest.Button] = ""

						elevio.SetButtonLamp(elevio.ButtonType(updatedRequest.Button), updatedRequest.Floor, false)

					} else if updatedRequest.HandledBy != "Unchanged" {
						fsm.Elevator.HandledBy[updatedRequest.Floor][updatedRequest.Button] = updatedRequest.HandledBy
					}
				}

			} else {
				fsm.Elevator.HandledBy[updatedRequest.Floor][updatedRequest.Button] = updatedRequest.HandledBy
				fsm.Elevator.Requests[updatedRequest.Floor][updatedRequest.Button] = true
				fsm.OnRequestButtonPress(updatedRequest.Floor, elevio.ButtonType(updatedRequest.Button), TimerStartChan, requestUpdateChan)
			}

		case updatedCall := <-callUpdatesChan:
			// Locking to ensure safe access to fsm.Elevator
			requests.Mu5.Lock()
			defer requests.Mu5.Unlock()

			if updatedCall.HandledBy != "Unchanged" {
				fsm.Elevator.HandledBy[updatedCall.Floor][updatedCall.Button] = updatedCall.HandledBy
			}
			if updatedCall.Delete {
				fsm.Elevator.HallCalls[updatedCall.Floor][updatedCall.Button] = false
				elevio.SetButtonLamp(elevio.ButtonType(updatedCall.Button), updatedCall.Floor, false)
			} else {
				// if message and elevator agree on who should handle call, the elevator will take the call
				if fsm.Elevator.HandledBy[updatedCall.Floor][updatedCall.Button] != "Done" {
					fsm.Elevator.HallCalls[updatedCall.Floor][updatedCall.Button] = true
					elevio.SetButtonLamp(elevio.ButtonType(updatedCall.Button), updatedCall.Floor, true)
				}
			}
		}

		num, error := strconv.Atoi(fsm.Elevator.Id)
		if error == nil {
			elevators[num-8081] = fsm.Elevator
		} else {
			fmt.Println("Error in UpdateFromMessage: ", error)
		}
	}
}



func RequestUpdater (TimerStartChan chan time.Duration, callUpdatesChan chan requests.CallUpdate, requestUpdateChan chan requests.CallUpdate) {
	
	for {
		// fmt.Printf("str2: %q\n", elevators[1].HandledBy[2][0]) // This prints "8087" (quoted string)
		
		mu.RLock()
		for floor := 0; floor < 4; floor++ { 
			for button := 0; button < 2; button++ {
				if fsm.Elevator.HandledBy[floor][button] == fsm.Elevator.Id { //Checks if local elevator wants call
					if numActiveElevators > 1 {
						agreedOnFloor := 1
						for _,elevator := range elevators {
							
							if elevator.Id != fsm.Elevator.Id && elevator.HandledBy[floor][button] == fsm.Elevator.HandledBy[floor][button] && fsm.Elevator.HandledBy[floor][button] == fsm.Elevator.Id {
								agreedOnFloor++
							}
						}
						if agreedOnFloor >= 2 && !fsm.Elevator.Requests[floor][button] && fsm.Elevator.HandledBy[floor][button] != "Done" {
							requestUpdateChan <- requests.CallUpdate{
								Floor: floor,
								Button: button,
								HandledBy: fsm.Elevator.Id,
								Delete: false,
							}
						} else {
							requestUpdateChan <- requests.CallUpdate{
								Floor: floor,
								Button: button,
								HandledBy: "Unchanged",
								Delete: true,
							}
						}
					} else {
						requestUpdateChan <- requests.CallUpdate{
							Floor: floor,
							Button: button,
							HandledBy: fsm.Elevator.Id,
							Delete: false,
						}
					}
				}
			}
		}
		mu.RUnlock()

	}
	
}


/*

func UpdateElevatorHallCallsAndButtonLamp(msg messageProcessing.Message, requestChan chan requests.Request, TimerStartChan chan time.Duration)  {
	id := msg.Elevator.Id
	
	requests.Mu5.Lock()
	if id != fsm.Elevator.Id{
		num, _ := strconv.Atoi(id)
		elevators[num-8081] = msg.Elevator
		for floor := 0; floor < 4; floor++ { 
			for button := 0; button < 2; button++ {

				//################Sets floor calls and removes true if all elevators have finished removing buttoncall ####################################
				if fsm.Elevator.HandledBy[floor][button] == "Done" && !elevators[0].HallCalls[floor][button] && !elevators[1].HallCalls[floor][button] && !elevators[2].HallCalls[floor][button] {
					
					fsm.Elevator.HandledBy[floor][button] = ""
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
					
					
				//removes hallcall if done
				} else if msg.Elevator.HandledBy[floor][button] == "Done" {
					fsm.Elevator.HallCalls[floor][button] = false
					fsm.Elevator.HandledBy[floor][button] = "" //This for some reason only changes locally, not in message. This is why it doesnt work
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
				
		// Only send request if the value is true
				} else if msg.Elevator.HallCalls[floor][button] && !fsm.Elevator.HallCalls[floor][button] && fsm.Elevator.HandledBy[floor][button] != "Done"{  
					//Check if request at floor and request not alreadu accounted for
					fsm.Elevator.HallCalls[floor][button] = true
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, true) //sets hallcall light

					// Sends requests from other elevators to requestchan and lights up button
					requestChan <- requests.Request{
						FloorButton: elevio.ButtonEvent{Button: elevio.ButtonType(button), Floor:  floor},
						HandledBy: -1,
						} 
				}
				
			}
			
		}
	}
	//updates elevators with self, this is needed to update information changed above
	num, _ := strconv.Atoi(fsm.Elevator.Id)
    elevators[num-8081] = fsm.Elevator

	requests.Mu5.Unlock()
}

*/

func ResourceManager(TimerStartChan chan time.Duration, callUpdatesChan chan requests.CallUpdate, requestUpdateChan chan requests.CallUpdate) {
	for {
		// Checks if elevators are active before sending them to request
		activeElevators := []elevator.Elevator{}

		// Lock for reading elevators and ElevatorStatus
		mu.RLock()
		messageProcessing.ActiveMu.Lock()
		for _, elevator := range elevators {
			if messageProcessing.ElevatorStatus[elevator.Id] {
				activeElevators = append(activeElevators, elevator)
			}
		}
		// Unlock after reading elevators and ElevatorStatus
		messageProcessing.ActiveMu.Unlock()
		mu.RUnlock()

		// Lock to write to numActiveElevators
		mu.Lock() // Lock to write to shared resource
		numActiveElevators = len(activeElevators)
		mu.Unlock() // Unlock after writing

		if numActiveElevators != 0 {
			// fmt.Println("fem", activeElevators)
			input := ConvertRequestToHRAInput(activeElevators)

			// Call function to process requests
			output, err := ProcessElevatorRequests(input)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// Use requests.Mu5 to protect fsm.Elevator.HandledBy
			for elevatorID, floorButtonStates := range output {
				for floor, buttons := range floorButtonStates {
					for button, buttonState := range buttons {
						if buttonState {
							// Lock fsm.Elevator before checking HandledBy
							requests.Mu5.Lock()
							if fsm.Elevator.HandledBy[floor][button] != "Done" {
								// Ensure elevatorID slicing works correctly
								callUpdatesChan <- requests.CallUpdate{
									Floor:     floor,
									Button:    button,
									HandledBy: elevatorID[len("%!d(string="):len(elevatorID)-1], // Verify this slice logic
									Delete:    false,
								}
							}
							requests.Mu5.Unlock() // Unlock after reading fsm.Elevator
						}
					}
				}
			}
		}
	}
}