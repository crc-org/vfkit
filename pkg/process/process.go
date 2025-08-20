/*
Copyright 2025, Red Hat, Inc - All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package process

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

type Process struct {
	Name           string
	PidFilePath    string
	ExecutablePath string
}

func New(name, pidFilePath, executablePath string) *Process {
	return &Process{Name: name, PidFilePath: pidFilePath, ExecutablePath: executablePath}
}

func (p *Process) ReadPidFile() (int, error) {
	data, err := os.ReadFile(p.PidFilePath)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return int(pid), nil
}

func (p *Process) findProcess() (*process.Process, error) {
	pid, err := p.ReadPidFile()
	if err != nil {
		return nil, err
	}
	if pid < 0 || pid > math.MaxInt32 {
		return nil, fmt.Errorf("invalid pid: %d", pid)
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil && err.Error() == "process does not exist" {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	if proc == nil {
		return nil, os.ErrNotExist
	}
	name, err := proc.Name()
	if err != nil {
		return nil, fmt.Errorf("cannot find process name: %w", err)
	}
	if name != p.Name {
		return nil, os.ErrNotExist
	}
	exe, err := proc.Exe()
	if err != nil {
		return nil, fmt.Errorf("cannot find process exe: %w", err)
	}
	if exe != p.ExecutablePath {
		return nil, os.ErrNotExist
	}
	return proc, nil
}

func (p *Process) Exists() (bool, error) {
	proc, err := p.findProcess()
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if proc == nil {
		return false, nil
	}
	return true, nil
}

func (p *Process) Terminate() error {
	proc, err := p.findProcess()
	if err != nil {
		return fmt.Errorf("cannot find process: %w", err)
	}
	return proc.Terminate()
}

func (p *Process) Kill() error {
	proc, err := p.findProcess()
	if err != nil {
		return fmt.Errorf("cannot find process: %w", err)
	}
	return proc.Kill()
}

func (p *Process) WritePidFile(pid int) error {
	return os.WriteFile(p.PidFilePath, []byte(strconv.Itoa(pid)), 0600)
}
