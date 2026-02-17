package main

import (
	"os"
	"os/exec"
	"syscall"
)

func main() {
	cmd := exec.Command("go", "run", "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		os.Exit(1)
	}
}
