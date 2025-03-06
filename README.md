This project is a Go-based elevator control system. It includes multiple Go files that define the behavior of the elevator and its control logic.


## Architecture of the code
The project is organised into different modules used in `main` files :
- `elevator` :  implements the core logic for controlling the elevator, including movement, floor selection, and responses to commands.
- `elevator_io_device` : handles communication with the physical elevator hardware or a simulator. Responsible for sending/receiving signals like button presses, motor control, and sensor data.
- `elevio` : handles communication with the elevator simulator. Responsible for sending/receiving signals like button presses, motor control, and sensor data.
- `fsm` : manages the elevator’s states (e.g., idle, moving, door open, handling requests). Controls transitions between states based on events like button presses or sensor feedback.
- `hall_request_assigner` : d code which dynamically distributes requests among the elevators in a multi-elevator system. Likely optimizes for efficiency, minimizing wait times and unnecessary movement.
- `messageProcessing` : handles inter-module communication, and processing messages exchanged between different parts of the system (e.g., requests or elevators status).
- `network` : implements peer-to-peer (P2P) topology. 
- `requests` : define the requests structure, test if an elevator should stop, cleared it requests, moved and in which direction.
- `resource` : convert go state structure into HRA input and call functions in `hall_request_assigner` module to get the requests distributions among the elevator. Send the tasks to the elevators concerned.
- `TCP` : implements all the functions for TCP communication between elevators.
- `timer` : implements all the time functions used for the simulation and define a timeout.


## User guide
### Requirements
- Go (latest version recommended)
### Command operation
To run the elevator simulation, follow the following steps :

1. Run two simelevator servers with the ports **10003** and **10002** (corrisponding to `main_3.go` and `main_2.go`) (either use exe or write `./SimElevatorServer --port 10002`)
2. Run the real elevatorserver or another sim. If you run another sim you have to edit `main_1.go`'s port on **line 97** from **localhost:15657** to **localhost:10001** (or any other port)

3. run main_1,2 and 3 in seperate terminals


Or replace main_1.go with main_2.go or main_3.go depending on the test case you want to execute.
## Roadmap

Future improvements and features planned:
- For the moment, the user has to run *n* `main_n.go` in seperate terminal to simulate *n* elevators, this will change as soon as possible so that only one main has to be runned.
- As the module `resource` use `hall_request_assigner` both will probably be combined into one module. 
- Some modules have multiple aims or perform actions that other modules should do, this will be fixed
- Even if a ressource manager module was made, each elevator functions as a normal single elevator, without using the output the D code which distribute the request. 
- A P2P model is currently being implemented but not active for the moment.
- For the moment, acceptance tests were written only for `elevator` module. This will be pursue for all the modules.
- Even if we have to check that everything is ok (with acceptance tests and fault tolerance) rather than looking for errors, we will add data type and interval value check at the beginning of each functions. 
- For the moment, nothing was made to handle packet loss. We will study the D code `packetloss.d` in the project ressources, translate and adapt it to our project. 
- Nothing was concerning fault tolerance and we have to think about 


HOWTORUN:
1. run two simelevator servers with the ports 10003 and 10002 (corrisponding to main 3 and main two) (either use exe or write ./SimElevatorServer --port 10002)
2. Run the real elevatorserver or another sim. If you run another sim you have to edit main_1.go's port on line 97 from localhost:15657 to localhost:10001 (or any other port)

3. run main_1,2 and 3 in seperate terminals


For HRA:
chmod a+rwx hall_request_assigner

For packetloss (examples):
sudo packetloss -f
        Removes all packet loss rules, disabling packet loss
        
    sudo packetloss -p 12345,23456,34567 -r 0.25
        Applies 25% packet loss to ports 12345, 23456, and 34567
        
    sudo packetloss -n executablename -r 0.25
        Applies 25% packet loss to all ports used by all programs named "executablename"
        
    sudo packetloss -p 12345 -n executablename -r 0.25
        Also applies 25% packet loss to port 12345

    sudo packetloss -n executablename -f
        Lists ports used by "executablename", but does not apply packet loss
