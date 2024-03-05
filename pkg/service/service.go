package service

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/initialed85/dspo/internal"
	"github.com/initialed85/dspo/pkg/common"
	"github.com/initialed85/dspo/pkg/fanout"
	"github.com/initialed85/dspo/pkg/managed_process"
	"github.com/initialed85/dspo/pkg/probe"
)

type Service struct {
	managedProcess *managed_process.ManagedProcess
	startupProbe   *probe.Probe
	livenessProbe  *probe.Probe
	onStarted      func()
	onLive         func()
	onDead         func()
	logs           chan managed_process.Log
	fanout         *fanout.Fanout
	mu             sync.Mutex
	logger         *slog.Logger
	started        bool
	startupReady   bool
	livenessReady  bool
	name           string
}

func New(
	managedProcessArgs common.ManagedProcessArgs,
	startupProbeArgs *common.StartupProbeArgs,
	livenessProbeArgs *common.LivenessProbeArgs,
	onStartupReady func(),
	onLivenessReady func(),
	onLivenessNotReady func(),
	name string,
) *Service {
	s := Service{
		onStarted: onStartupReady,
		onLive:    onLivenessReady,
		onDead:    onLivenessNotReady,
		logs:      make(chan managed_process.Log),
		logger:    internal.GetLogger(name),
		name:      name,
	}

	s.managedProcess = managed_process.New(
		s.logs,
		managedProcessArgs.RestartPolicy,
		managedProcessArgs.Shell,
		managedProcessArgs.Command,
		managedProcessArgs.Env,
		managedProcessArgs.InheritEnv,
		managedProcessArgs.RestartWaitDuration,
		func(returnCode int) {},
		name,
	)

	if startupProbeArgs != nil {
		s.startupProbe = probe.New(
			startupProbeArgs.StartupTolerance,
			startupProbeArgs.ProbeInterval,
			0,
			startupProbeArgs.Command,
			managedProcessArgs.Env,
			managedProcessArgs.InheritEnv,
			s.startupOnReady,
			common.NoOpFunc,
			fmt.Sprintf("%v_startup", name),
		)
	}

	if livenessProbeArgs != nil {
		s.livenessProbe = probe.New(
			time.Second*60*60*24*365*100,
			livenessProbeArgs.ProbeInterval,
			livenessProbeArgs.PermittedFailures,
			livenessProbeArgs.Command,
			managedProcessArgs.Env,
			managedProcessArgs.InheritEnv,
			s.livenessOnReady,
			s.livenessOnNotReady,
			fmt.Sprintf("%v_liveness", name),
		)
	}

	return &s
}

func (s *Service) startupOnReady() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startupReady {
		return
	}

	s.startupReady = true

	if s.onStarted != nil {
		s.onStarted()
	}

	s.livenessProbe.SetIgnoreUntil(time.Now().Add(-time.Nanosecond * 1))

	s.logger.Debug("startup ready")
}

func (s *Service) livenessOnReady() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.livenessReady {
		return
	}

	s.livenessReady = true

	if s.onLive != nil {
		s.onLive()
	}

	s.logger.Debug("liveness ready")
}

func (s *Service) livenessOnNotReady() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.livenessReady {
		return
	}

	s.livenessReady = false

	if s.onDead != nil {
		s.onDead()
	}

	s.logger.Debug("liveness not ready")
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started
}

func (s *Service) StartupReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startupReady
}

func (s *Service) LivenessReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.livenessReady
}

func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("already started")
	}

	s.started = true
	s.startupReady = s.onStarted == nil
	s.livenessReady = s.onLive == nil && s.onDead == nil

	var err error

	err = s.managedProcess.Start()
	if err != nil {
		return err
	}

	if s.startupProbe != nil {
		err = s.startupProbe.Start()
		if err != nil {
			return err
		}
	}

	if s.livenessProbe != nil {
		err = s.livenessProbe.Start()
		if err != nil {
			return err
		}
	}

	s.fanout = fanout.New(s.logs)

	s.logger.Debug("started")

	return nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("not started")
	}

	if s.livenessProbe != nil {
		_ = s.livenessProbe.Stop()

		if s.livenessReady {
			if s.onDead != nil {
				s.onDead()
			}
		}
	}

	if s.startupProbe != nil {
		_ = s.startupProbe.Stop()
	}

	_ = s.managedProcess.Stop()

	if s.fanout != nil {
		s.fanout.Close()
		s.fanout = nil
	}

	s.started = false

	s.logger.Debug("stopped")

	return nil
}

func (s *Service) SubscribeToLogs() (chan managed_process.Log, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fanout == nil {
		return nil, nil, fmt.Errorf("no fanout to subscribe to (not started?)")
	}

	logs, cancel := s.fanout.Subscribe()

	return logs, cancel, nil
}
