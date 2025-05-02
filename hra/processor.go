package hra

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

func HRAProcessor(currentInput HRAInput) *map[string][][2]bool {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "./hra/hall_request_assigner"
	case "windows":
		hraExecutable = ".\\hra\\hall_request_assigner.exe"
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

	return output
}
