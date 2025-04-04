package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dummyProcessName = "sleep"
	dummyProcessArgs = "60"
)

var (
	dummyProcess   *exec.Cmd
	managedProcess *Process
	pidFilePath    = filepath.Join(os.TempDir(), "pid")
)

func startDummyProcess() error {
	dummyProcess = exec.Command(dummyProcessName, dummyProcessArgs)
	err := dummyProcess.Start()
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	err := startDummyProcess()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start process:", err)
		os.Exit(1)
	}

	managedProcess, err = New(dummyProcessName, pidFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create process:", err)
		os.Exit(1)
	}
	err = managedProcess.WritePidFile(dummyProcess.Process.Pid)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	exitCode := m.Run()
	if dummyProcess.Process != nil {
		_ = dummyProcess.Process.Kill()
	}

	os.Exit(exitCode)
}

func TestProcess_Name(t *testing.T) {
	assert.Equal(t, dummyProcessName, managedProcess.Name())
}

func TestProcess_FindProcess(t *testing.T) {
	foundProcess, err := managedProcess.FindProcess()
	assert.NoError(t, err)
	assert.NotNil(t, foundProcess)
	assert.Equal(t, dummyProcess.Process.Pid, foundProcess.Pid)

	assert.True(t, managedProcess.Exists())
}

func TestProcess_KillProcess(t *testing.T) {
	err := managedProcess.Kill()
	assert.NoError(t, err)
	assert.False(t, managedProcess.Exists())

	// Try to kill the non-existent process
	// This should result in an error
	err = managedProcess.Kill()
	assert.Error(t, err)
}

func TestProcess_FindProcess_InvalidPidFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "invalid_pid")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// Write non-numeric content into the file to mimic an invalid pid
	_, err = tmpfile.WriteString("non-numeric")
	require.NoError(t, err)
	tmpfile.Close()

	invalidProcess, err := New("invalid-process", tmpfile.Name())
	assert.NoError(t, err)

	foundProcess, err := invalidProcess.FindProcess()
	assert.Error(t, err)
	assert.Nil(t, foundProcess)
}
