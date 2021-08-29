package main

import (
	"os"

	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runCommandWithOutput(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runK3dCommand() error {
	cmd := exec.Command("k3d")

	return cmd.Run()
}

func TestImageCaching(t *testing.T) {
	err := runCommandWithOutput("k3d", "cluster", "create", "foobar", "--no-lb")

	if !assert.NoError(t, err) {
		return
	}

	t.Cleanup(func() {
		if err := runCommandWithOutput("k3d", "cluster", "delete", "foobar"); err != nil {
			t.Errorf("failed to cleanup test cluster: %v", err)
		}
	})

	err = runCommandWithOutput("docker", "build", "-f", "Dockerfile.warden", "-t", "warden:ci", ".")

	if !assert.NoError(t, err) {
		return
	}
}
