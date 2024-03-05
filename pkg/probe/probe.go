package probe

import (
	"log/slog"
	"sync"
	"time"

	"github.com/initialed85/dspo/internal"
	"github.com/initialed85/dspo/pkg/managed_process"
)

type Probe struct {
	startupTolerance  time.Duration
	probeInterval     time.Duration
	permittedFailures int
	onReady           func()
	onNotReady        func()
	mu                sync.Mutex
	managedProcess    *managed_process.ManagedProcess
	ignoreUntil       time.Time
	failureCount      int
	ready             bool
	logger            *slog.Logger
}

func New(
	startupTolerance time.Duration,
	probeInterval time.Duration,
	permittedFailures int,
	command string,
	env []string,
	inheritEnv bool,
	onReady func(),
	onNotReady func(),
	name string,
) *Probe {
	p := Probe{
		startupTolerance:  startupTolerance,
		probeInterval:     probeInterval,
		permittedFailures: permittedFailures,
		onReady:           onReady,
		onNotReady:        onNotReady,
		logger:            internal.GetLogger(name),
	}

	p.managedProcess = managed_process.New(
		nil,
		managed_process.UnlessStopped,
		"/bin/bash",
		command,
		env,
		inheritEnv,
		probeInterval,
		p.onExit,
		name,
	)

	return &p
}

func (p *Probe) onExit(returnCode int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Now().Before(p.ignoreUntil) {
		return
	}

	if returnCode != 0 {
		p.failureCount++

		if p.failureCount > p.permittedFailures {
			p.onNotReady()
			p.ready = false
		}

		return
	}

	p.onReady()
	p.ready = true
	p.failureCount = 0
}

func (p *Probe) SetIgnoreUntil(ignoreUntil time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ignoreUntil = ignoreUntil
}

func (p *Probe) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.managedProcess.Start()
	if err != nil {
		return err
	}

	p.ignoreUntil = time.Now().Add(p.startupTolerance)
	p.failureCount = 0

	p.logger.Debug("started")

	return nil
}

func (p *Probe) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.managedProcess.Stop()
	if err != nil {
		return err
	}

	p.logger.Debug("stopped")

	return nil
}
