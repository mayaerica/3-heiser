package resource

import (
	"time"
	req "Driver-go/requests"
	el "Driver-go/elevator"
)

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

// Modify RequestDispatcher to follow the new assignment logic
func RequestDispatcher(req req.Request, elevators []el.Elevator) el.Elevator {
	candidates := []el.Elevator{}

	// Find eligible elevators
	for _, e := range elevators {
		if e.Busy == false || (e.Dirn == 1 && e.Dirn == req.Dirn && e.Floor <= req.Floor) || (e.Dirn == -1 && e.Dirn == req.Dirn && e.Floor > req.Floor) {
			candidates = append(candidates, e)
		}
	}

	// Assign request to the nearest eligible elevator
	if len(candidates) > 0 {
		bestElevator := candidates[0]
		minDist := abs(bestElevator.Floor - req.Floor)
		for _, e := range candidates {
			dist := abs(e.Floor - req.Floor)
			if dist < minDist {
				bestElevator = e
				minDist = dist
			}
		}
		return bestElevator
		
	} else {
		return el.Elevator{Id: 0}// Return empty elevator or handle the case as needed
	}
}	

func ResourceManager(requestChan chan req.Request, assignChan chan req.Request, elevators []el.Elevator) {
    queue := Queue{}

    for {
        if !queue.Empty() {
            request := queue.Front()
            assignedElevator := RequestDispatcher(request, elevators)
            
            if assignedElevator.Id != 0 {
                queue.PopFront()
                request.HandledBy = assignedElevator.Id
                assignChan <- request
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


