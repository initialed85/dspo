package service

import (
	"runtime"
	"testing"
	"time"

	"github.com/initialed85/dspo/pkg/common"
	"github.com/initialed85/dspo/pkg/managed_process"
	"github.com/initialed85/dspo/test"
	"github.com/stretchr/testify/require"
)

const (
	startupProbeTestScript = `
#!/bin/bash

set -e

if cat /tmp/startup_probe_test.tmp | grep 'startup_probe_ready: true'; then
	exit 0
fi

exit 1
`
	livenessProbeTestScript = `
#!/bin/bash

set -e

if cat /tmp/liveness_probe_test.tmp | grep 'liveness_probe_ready: true'; then
	exit 0
fi

exit 1
`
)

func TestNew(t *testing.T) {
	startupProbeHarness := test.NewProbeHarness("startup")
	livenessProbeHarness := test.NewProbeHarness("liveness")

	t.Run("HappyPath", func(t *testing.T) {
		started := false
		live := false

		s := New(
			common.ManagedProcessArgs{
				RestartPolicy:       managed_process.UnlessStopped,
				Shell:               "/bin/bash",
				Command:             "while true; do echo 'tick'; sleep 1; done",
				Env:                 nil,
				InheritEnv:          true,
				RestartWaitDuration: time.Millisecond * 50,
			},
			&common.StartupProbeArgs{
				StartupTolerance: time.Millisecond * 250,
				ProbeInterval:    time.Millisecond * 50,
				Command:          startupProbeHarness.GetExecutablePath(),
			},
			&common.LivenessProbeArgs{
				ProbeInterval:     time.Millisecond * 50,
				PermittedFailures: 3,
				Command:           livenessProbeHarness.GetExecutablePath(),
			},
			func() {
				started = true
			},
			func() {
				live = true
			},
			func() {
				live = false
			},
			"test",
		)
		require.NoError(t, s.Start())
		defer func() {
			_ = s.Stop()
		}()

		go func() {
			time.Sleep(time.Millisecond * 125)
			startupProbeHarness.SetReady()
			livenessProbeHarness.SetReady()
		}()
		runtime.Gosched()
		require.Eventually(
			t,
			func() bool {
				return started
			},
			time.Millisecond*500,
			time.Millisecond*50,
		)
		require.Eventually(
			t,
			func() bool {
				return live
			},
			time.Millisecond*500,
			time.Millisecond*50,
		)

		go func() {
			time.Sleep(time.Millisecond * 500)
			livenessProbeHarness.SetNotReady()
		}()
		runtime.Gosched()
		require.Eventually(
			t,
			func() bool {
				return live
			},
			time.Millisecond*500,
			time.Millisecond*50,
		)
	})
}
