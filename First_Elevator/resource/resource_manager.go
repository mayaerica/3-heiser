package resource

import (
	el "Driver-go/elevator"
	req "Driver-go/requests"
	tcp "Driver-go/TCP"
	fsm "Driver-go/fsm"
	eio "Driver-go/elevio"
	"fmt"
	"time"
)

var elevators = []el.Elevator{
    {Id: 8081, Floor: 0, Dirn: eio.Dirn(0), Behaviour: el.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8081
    {Id: 8082, Floor: 0, Dirn: eio.Dirn(0), Behaviour: el.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8082
    {Id: 8083, Floor: 0, Dirn: eio.Dirn(0), Behaviour: el.ElevatorBehaviour(0), Busy: false, DoorOpenDuration: 0, ClearRequestVariant: 1}, // Elevator 8083
}

type Queue struct {
    requests []req.Request
}

func (q *Queue) Empty() bool {
    return len(q.requests) == 0
}

func (q *Queue) Front() req.Request {
    return q.requests[0]
}

func (q *Queue) PopFront() {
    q.requests = q.requests[1:]
}

func (q *Queue) Insert(r req.Request) {
    q.requests = append(q.requests, r)
}


func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}


func UpdateElevators(messages map[int]tcp.Message, requestChan chan req.Request)  {
	for id, msg := range messages {
		if id != fsm.Elevator.Id{
			elevators[id-8081] = msg.Elevator
		}

		for _, request := range msg.Requests {
			fmt.Printf("Request: Button %v on Floor %d\n", request.FloorButton.Button, request.FloorButton.Floor)
			if request.HandledBy == fsm.Elevator.Id {
				requestChan <- request
			}
		}
	}

	

	

	elevators[fsm.Elevator.Id-8081] = fsm.Elevator
}


// Modify RequestDispatcher to follow the new assignment logic
func RequestDispatcher(requests req.Request) el.Elevator { // this should take in A button dirn pair, as it needs not a candidate as it find the candidates
	candidates := []el.Elevator{}

	// Find eligible elevators
	for _, e := range elevators {
		if e.Busy == false || (e.Dirn == 1 && requests.FloorButton.Button == eio.ButtonType(0) && e.Floor <= requests.FloorButton.Floor) || (e.Dirn == -1 && requests.FloorButton.Button == eio.ButtonType(1) && e.Floor > requests.FloorButton.Floor) {
			candidates = append(candidates, e)
		}
	}

	// Assign request to the nearest eligible elevator
	if len(candidates) > 0 {
		bestElevator := candidates[0]
		minDist := abs(bestElevator.Floor - requests.FloorButton.Floor)
		for _, e := range candidates {
			dist := abs(e.Floor - requests.FloorButton.Floor)
			if dist < minDist {
				bestElevator = e
				minDist = dist
			}
		}
		//fmt.Println("Best Elevator:", bestElevator)
		return bestElevator
		
	} else {
		return el.Elevator{Id: 0}// Return empty elevator or handle the case as needed
	}
}	

func ResourceManager(requestChan chan req.Request, assignChan chan req.Request) {
    queue := Queue{}

    for {
        if !queue.Empty() {
            requests := queue.Front()
            assignedElevator := RequestDispatcher(requests)
			requests.HandledBy = assignedElevator.Id
            
            if assignedElevator.Id != 0 {
				fmt.Print() //FJERN
                queue.PopFront()
                assignChan <- requests
            }
        }

        select {
        case request := <-requestChan:
            queue.Insert(request)
        default:
            time.Sleep(10 * time.Millisecond) // Prevents high CPU usage
        }
    }
}


