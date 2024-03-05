package process

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type Process struct {
	cmd        *exec.Cmd
	wg         sync.WaitGroup
	err        error
	mu         sync.Mutex
	returnCode int
}

func Run(
	shell string,
	command string,
	env []string,
	inheritEnv bool,
	stdoutPipe io.Writer,
	stderrPipe io.Writer,
) *Process {
	actualEnv := make([]string, 0)

	if inheritEnv {
		for _, v := range os.Environ() {
			actualEnv = append(actualEnv, v)
		}
	}

	for _, v := range env {
		actualEnv = append(actualEnv, v)
	}

	p := &Process{
		cmd: exec.Command(
			shell,
			"-c",
			command,
		),
		returnCode: -1,
	}
	p.cmd.Env = actualEnv
	p.cmd.Stdout = stdoutPipe
	p.cmd.Stderr = stderrPipe

	p.wg.Add(1)
	go func() {
		err := p.cmd.Run()

		p.mu.Lock()
		p.err = err
		if p.cmd.ProcessState != nil {
			p.returnCode = p.cmd.ProcessState.ExitCode()
		}
		p.mu.Unlock()

		p.wg.Done()
	}()
	runtime.Gosched()

	return p
}

func (p *Process) Error() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.err
}

func (p *Process) Wait() error {
	p.wg.Wait()

	return p.Error()
}

func (p *Process) ReturnCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.returnCode
}

func (p *Process) Close() {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}

	return
}
