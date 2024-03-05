package test

import (
	"fmt"
	"time"

	"github.com/initialed85/dspo/pkg/common"
	"github.com/initialed85/dspo/pkg/managed_process"
)

type MockService struct {
	StartupProbeHarness  *ProbeHarness
	LivenessProbeHarness *ProbeHarness
	StartupDelay         time.Duration
	ServiceArgs          common.ServiceArgs
}

func NewMockService(
	name string,
	dependsOn []string,
) *MockService {
	m := MockService{
		StartupProbeHarness:  NewProbeHarness(fmt.Sprintf("%v_startup", name)),
		LivenessProbeHarness: NewProbeHarness(fmt.Sprintf("%v_liveness", name)),
	}

	m.ServiceArgs = common.ServiceArgs{
		Name:      name,
		DependsOn: dependsOn,
		ManagedProcessArgs: common.ManagedProcessArgs{
			RestartPolicy:       managed_process.UnlessStopped,
			Shell:               "/bin/bash",
			Command:             "while true; do echo 'tick'; sleep 1; done",
			Env:                 nil,
			InheritEnv:          true,
			RestartWaitDuration: time.Millisecond * 50,
		},
		StartupProbeArgs: &common.StartupProbeArgs{
			StartupTolerance: time.Millisecond * 250,
			ProbeInterval:    time.Millisecond * 50,
			Command:          m.StartupProbeHarness.GetExecutablePath(),
		},
		LivenessProbeArgs: &common.LivenessProbeArgs{
			ProbeInterval:     time.Millisecond * 50,
			PermittedFailures: 3,
			Command:           m.LivenessProbeHarness.GetExecutablePath(),
		},
	}

	return &m
}
