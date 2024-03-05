package managed_process

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/initialed85/dspo/internal"
	"github.com/initialed85/dspo/pkg/process"
)

type RestartPolicy string

const (
	Never            RestartPolicy = "no"
	UnlessStopped    RestartPolicy = "unless-stopped"
	OnFailure        RestartPolicy = "on-failure"
	bufferSize                     = 1024
	internalLogDepth               = 1024
)

type Log struct {
	Name      string
	IsStdout  bool
	IsStderr  bool
	Timestamp int64
	Data      []byte
}

type ManagedProcess struct {
	logs                chan Log
	restartPolicy       RestartPolicy
	shell               string
	command             string
	env                 []string
	inheritEnv          bool
	restartWaitDuration time.Duration
	onExit              func(int)
	internalLogs        chan Log
	stdoutReader        io.ReadCloser
	stdoutWriter        io.WriteCloser
	stderrReader        io.ReadCloser
	stderrWriter        io.WriteCloser
	mu                  *sync.Mutex
	running             bool
	process             *process.Process
	ctx                 context.Context
	cancel              context.CancelFunc
	logger              *slog.Logger
	name                string
}

func New(
	logs chan Log,
	restartPolicy RestartPolicy,
	shell string,
	command string,
	env []string,
	inheritEnv bool,
	restartWaitDuration time.Duration,
	onExit func(int),
	name string,
) *ManagedProcess {
	m := ManagedProcess{
		logs:                logs,
		restartPolicy:       restartPolicy,
		shell:               shell,
		command:             command,
		env:                 env,
		inheritEnv:          inheritEnv,
		restartWaitDuration: restartWaitDuration,
		onExit:              onExit,
		internalLogs:        make(chan Log, internalLogDepth),
		mu:                  new(sync.Mutex),
		running:             false,
		process:             nil,
		logger:              internal.GetLogger(name),
		name:                name,
	}

	return &m
}

func (m *ManagedProcess) runLogger() {

	var l Log

	for {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			break
		}
		m.mu.Unlock()

		select {
		case l = <-m.internalLogs:
			m.logs <- l
		}
	}
}

func (m *ManagedProcess) runReader(name string, stream io.Reader) {
	isStdout := name == "stdout"
	isStderr := name == "stderr"

	for {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			break
		}
		m.mu.Unlock()

		b := make([]byte, bufferSize)

		n, err := stream.Read(b)
		if err != nil {
			if err != io.EOF {
			}
			continue
		}

		if n == 0 {
			continue
		}

		buf := b[0:n]

		l := Log{
			Name:      m.name,
			IsStdout:  isStdout,
			IsStderr:  isStderr,
			Timestamp: time.Now().UTC().UnixMilli(),
			Data:      buf,
		}

		select {
		case <-m.ctx.Done():
			break
		case m.internalLogs <- l:
		default:
		}
	}

	_ = m.Stop()
}

func (m *ManagedProcess) runLifecycle() {
	var p *process.Process
	var returnCode int

	restart := false

lifecycle:
	for {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			break
		}
		m.mu.Unlock()

		m.mu.Lock()
		if m.process == nil {
			m.mu.Unlock()
			break
		}
		p = m.process
		m.mu.Unlock()

		if restart {
			m.mu.Lock()
			m.process.Close()
			m.process = process.Run(
				m.shell,
				m.command,
				m.env,
				m.inheritEnv,
				m.stdoutWriter,
				m.stderrWriter,
			)
			m.mu.Unlock()
			restart = false
		}

		_ = p.Wait()

		returnCode = p.ReturnCode()

		m.onExit(returnCode)

		switch m.restartPolicy {
		case Never:
			break lifecycle
		case UnlessStopped:
			time.Sleep(m.restartWaitDuration)
			restart = true
			continue lifecycle
		case OnFailure:
			if returnCode != 0 {
				time.Sleep(m.restartWaitDuration)
				restart = true
				continue lifecycle
			}
			break lifecycle
		}
	}

	_ = m.Stop()
}

func (m *ManagedProcess) start() error {
	m.stdoutReader, m.stdoutWriter = io.Pipe()
	m.stderrReader, m.stderrWriter = io.Pipe()

	m.process = process.Run(
		m.shell,
		m.command,
		m.env,
		m.inheritEnv,
		m.stdoutWriter,
		m.stderrWriter,
	)

	go m.runLogger()
	runtime.Gosched()

	go m.runReader("stdout", m.stdoutReader)
	runtime.Gosched()

	go m.runReader("stderr", m.stderrReader)
	runtime.Gosched()

	go m.runLifecycle()
	runtime.Gosched()

	return nil
}

func (m *ManagedProcess) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("already running")
	}

	m.running = true
	m.ctx, m.cancel = context.WithCancel(context.Background())

	return m.start()
}

func (m *ManagedProcess) stop() error {
	m.running = false

	if !m.running {
		return fmt.Errorf("not running")
	}

	if m.process != nil {
		m.process.Close()
		m.process = nil
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if m.stdoutReader != nil {
		_ = m.stdoutReader.Close()
	}

	if m.stderrWriter != nil {
		_ = m.stderrWriter.Close()
	}

	if m.stderrReader != nil {
		_ = m.stderrReader.Close()
	}

	if m.stderrWriter != nil {
		_ = m.stderrWriter.Close()
	}

	return nil
}

func (m *ManagedProcess) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.stop()
}

func (m *ManagedProcess) Running() bool {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	return running
}
