


#dmd main.d config.d elevator_algorithm.d elevator_state.d optimal_hall_requests.d d-json/jsonx.d -w -g -ofhall_request_assigner;


dmd main.d config.d elevator_algorithm.d elevator_state.d optimal_hall_requests.d d-json/jsonx.d -I=d-json -of=hall_request_assigner
