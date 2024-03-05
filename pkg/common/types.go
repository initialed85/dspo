package common

import (
	"time"

	"github.com/initialed85/dspo/pkg/managed_process"
)

type ManagedProcessArgs struct {
	RestartPolicy       managed_process.RestartPolicy
	Shell               string
	Command             string
	Env                 []string
	InheritEnv          bool
	RestartWaitDuration time.Duration
}

type StartupProbeArgs struct {
	StartupTolerance time.Duration
	ProbeInterval    time.Duration
	Command          string
}

type LivenessProbeArgs struct {
	ProbeInterval     time.Duration
	PermittedFailures int
	Command           string
}

type ServiceArgs struct {
	Name               string
	DependsOn          []string
	ManagedProcessArgs ManagedProcessArgs
	StartupProbeArgs   *StartupProbeArgs
	LivenessProbeArgs  *LivenessProbeArgs
}

var (
	NoOpFunc = func() {}
)
