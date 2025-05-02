
# README : elevator project

**Timeslot :** 08:00--12:00 Wednesday \
**Workstation number :** 10 \
**Group number :** 53

## User guide
### Command operation
To run the project, do the following steps :\
**On physical elevators:**
1. Delete `backup.json`
2. Run `elevatorserver` to start the elevator
3. To run program use `go run main.go -id=*id* -port=*port nr.*`

**On simulators:**
1. Delete `backup.json`
2. Run `./SimElevatorServer --port *port nr.*`
3. To run program use `go run main.go -id=*id* -port=*port nr.*`

**To run acceptance tests:**
1. Go on the same `_test.go` file directory
2. Run `go test -v`

## Contributors

- Maya Erica Frafjord Saint-Victor : <mesaintv@stud.ntnu.no>
- Magnus Arnfinn Aubell :  <magnus.a.aubell@ntnu.no>
- Lorie Turco : <loriet@ntnu.no>
