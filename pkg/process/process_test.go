package process

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var waitTimeout = 250 * time.Millisecond

func TestPidFile(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "test.pid")
	process := New("test", pidfile, "test")
	expectedPid := 123
	err := process.WritePidFile(expectedPid)
	assert.NoError(t, err)

	actualPid, err := process.ReadPidFile()
	assert.NoError(t, err)
	assert.Equal(t, expectedPid, actualPid)
}

func TestPidfileMissing(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "test.pid")
	process := New("test", pidfile, "test")
	_, err := process.ReadPidFile()
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPidfileInvalid(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "pid")
	if err := os.WriteFile(pidfile, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	process := New("test", pidfile, "test")
	_, err := process.ReadPidFile()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid pid file")
}

func TestProcess(t *testing.T) {
	sleep, err := build(t, "sleep")
	if err != nil {
		t.Fatal(err)
	}
	pidfile := filepath.Join(t.TempDir(), "test.pid")
	process := New("sleep", pidfile, sleep)
	t.Run("exists", func(t *testing.T) {
		cmd := startProcess(t, sleep)
		pid := cmd.Process.Pid
		err := process.WritePidFile(pid)
		assert.NoError(t, err)
		exists, err := process.Exists()
		assert.NoError(t, err)
		assert.True(t, exists)
	})
	t.Run("terminate", func(t *testing.T) {
		cmd := startProcess(t, sleep)
		pid := cmd.Process.Pid

		err := process.WritePidFile(pid)
		assert.NoError(t, err)
		err = process.Terminate()
		assert.NoError(t, err)
		waitForTermination(t, cmd)
		exists, err := process.Exists()
		assert.NoError(t, err)
		assert.False(t, exists)
	})
	t.Run("kill", func(t *testing.T) {
		cmd := startProcess(t, sleep)
		pid := cmd.Process.Pid
		if pid < 0 || pid > math.MaxInt32 {
			t.Fatal("invalid pid")
		}
		err := process.WritePidFile(pid)
		assert.NoError(t, err)
		err = process.Kill()
		assert.NoError(t, err)
		waitForTermination(t, cmd)
		exists, err := process.Exists()
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func build(t *testing.T, name string) (string, error) {
	source := getSource(t, name)

	resolved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		return "", err
	}
	out := filepath.Join(resolved, name)

	t.Logf("Building %q", name)
	cmd := exec.Command("go", "build", "-o", out, source)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build %q: %v", name, err)
	}
	return out, nil
}

func getSource(t *testing.T, name string) string {

	testDataDir, err := getTestDataDir()

	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(testDataDir, name)
}

func startProcess(t *testing.T, exe string) *exec.Cmd {
	name := filepath.Base(exe)
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Started process %q (pid=%v)", name, cmd.Process.Pid)

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return cmd
}

func getTestDataDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(currentDir, "testdata"), nil
}

func waitForTermination(t *testing.T, cmd *exec.Cmd) {
	name := filepath.Base(cmd.Path)
	timer := time.AfterFunc(waitTimeout, func() {
		t.Fatalf("Timeout waiting for %q", name)
	})
	defer timer.Stop()
	start := time.Now()
	err := cmd.Wait()
	t.Logf("Process %q terminated in %.6f seconds: %s", name, time.Since(start).Seconds(), err)
}
