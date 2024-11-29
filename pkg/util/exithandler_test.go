package util

import (
	"syscall"
	"testing"
	"time"
)

func TestExitHandlerCalled(t *testing.T) {
	setupExitSignalHandling(false)

	ch := make(chan struct{})
	RegisterExitHandler(func() {
		close(ch)
	})

	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	if err != nil {
		t.Errorf("failed at sending SIGINT signal")
	}

	select {
	case <-ch:
		// exit handler was called
	case <-time.After(5 * time.Second):
		t.Errorf("Exit handler not called - timed out")
	}
}
