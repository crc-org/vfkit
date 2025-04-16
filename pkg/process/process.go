package process

import (
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

type Manager interface {
	Pid() (int, error)
	Name() string
	PidFilePath() string
	Exists() bool
	Kill() error
	FindProcess() (*os.Process, error)
	WritePidFile(pid int) error
}

type Process struct {
	name        string
	pidFilePath string
}

func New(name, pidFilePath string) (*Process, error) {
	return &Process{name: name, pidFilePath: pidFilePath}, nil
}

func (p *Process) Name() string {
	return p.name
}

func (p *Process) PidFilePath() string {
	return p.pidFilePath
}

func (p *Process) Pid() (int, error) {
	data, err := os.ReadFile(p.PidFilePath())
	if err != nil {
		return -1, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	return pid, errors.Wrap(err, "invalid pid file")
}

func (p *Process) FindProcess() (*os.Process, error) {
	pid, err := p.Pid()
	if err != nil {
		return nil, errors.Wrap(err, "cannot find process")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find process")
	}
	return proc, nil
}

func (p *Process) Exists() bool {
	proc, err := p.FindProcess()
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func (p *Process) Kill() error {
	proc, err := p.FindProcess()
	if err != nil {
		return errors.Wrap(err, "cannot find process")
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return errors.Wrap(err, "cannot kill process")
	}
	_, err = proc.Wait()
	if err != nil {
		return errors.Wrap(err, "failed to wait for process termination")
	}
	return nil
}

func (p *Process) WritePidFile(pid int) error {
	return os.WriteFile(p.pidFilePath, []byte(strconv.Itoa(pid)), 0600)
}
