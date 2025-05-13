package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

type Manager interface {
	ReadPidFile() (int, error)
	Name() string
	PidFilePath() string
	Exists() bool
	Terminate() error
	Kill() error
	FindProcess() (*os.Process, error)
	WritePidFile(pid int) error
}

type Process struct {
	name           string
	pidFilePath    string
	executablePath string
}

func New(name, pidFilePath, executablePath string) (*Process, error) {
	return &Process{name: name, pidFilePath: pidFilePath, executablePath: executablePath}, nil
}

func (p *Process) Name() string {
	return p.name
}

func (p *Process) PidFilePath() string {
	return p.pidFilePath
}

func (p *Process) ExecutablePath() string {
	return p.executablePath
}

func (p *Process) ReadPidFile() (int, error) {
	data, err := os.ReadFile(p.PidFilePath())
	if err != nil {
		return -1, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return -1, fmt.Errorf("invalid pid file: %v", err)
	}
	return pid, nil
}

func (p *Process) FindProcess() (*process.Process, error) {
	pid, err := p.ReadPidFile()
	if err != nil {
		return nil, fmt.Errorf("cannot find process: %v", err)
	}

	exists, err := process.PidExists(int32(pid))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("process not found")
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("cannot find process: %v", err)
	}
	if proc == nil {
		return nil, fmt.Errorf("process not found")
	}
	name, err := proc.Name()
	if err != nil {
		return nil, fmt.Errorf("cannot find process name: %v", err)
	}
	if name != p.Name() {
		return nil, fmt.Errorf("pid %d is stale, and is being used by %s", pid, name)
	}
	exe, err := proc.Exe()
	if err != nil {
		return nil, fmt.Errorf("cannot find process exe: %v", err)
	}
	if exe != p.ExecutablePath() {
		return nil, fmt.Errorf("pid %d is stale, and is being used by %s", pid, exe)
	}
	return proc, nil
}

func (p *Process) Exists() bool {
	proc, err := p.FindProcess()
	return err == nil && proc != nil
}

func (p *Process) Terminate() error {
	proc, err := p.FindProcess()
	if err != nil {
		return fmt.Errorf("cannot find process: %v", err)
	}
	return proc.Terminate()
}

func (p *Process) Kill() error {
	proc, err := p.FindProcess()
	if err != nil {
		return fmt.Errorf("cannot find process: %v", err)
	}
	return proc.Kill()
}

func (p *Process) WritePidFile(pid int) error {
	return os.WriteFile(p.pidFilePath, []byte(strconv.Itoa(pid)), 0600)
}
