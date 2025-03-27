package hra

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

// HRAProcessor runs the external hall request assigner binary with the given input.
func HRAProcessor(currentInput HRAInput) *map[string][][2]bool {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "./pkg/hra/hall_request_assigner"
	case "windows":
		hraExecutable = ".\\pkg\\hra\\hall_request_assigner.exe"
	default:
		panic("Unsupported OS")
	}

	if _, err := exec.LookPath(hraExecutable); err != nil {
		fmt.Println("Could not find HRA executable:", hraExecutable)
		return nil
	}

	jsonBytes, err := json.Marshal(currentInput)
	if err != nil {
		fmt.Println("json.Marshal error:", err)
		return nil
	}

	fmt.Println("[HRA] Running binary:", hraExecutable)
	ret, err := exec.Command(hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error:", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error:", err)
		return nil
	}

	fmt.Printf("HRAoutput:\n")
	for k, v := range *output {
		fmt.Printf("  %6v  %+v\n", k, v)
	}

	return output
}
