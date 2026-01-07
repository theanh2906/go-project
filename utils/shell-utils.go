package utils

import (
	"fmt"
	"os/exec"
)

func runCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run command %s: %w", command, err)
	}
	return string(output), nil
}

func runPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run PowerShell command %s: %w", command, err)
	}
	return string(output), nil
}
