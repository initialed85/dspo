package probe

import (
	"testing"
	"time"

	"github.com/initialed85/dspo/test"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	probeHarness := test.NewProbeHarness("default")

	ready := false

	t.Run("TransitionFromNotReadyToReadyToNotReady", func(t *testing.T) {
		p := New(
			time.Millisecond*300,
			time.Millisecond*100,
			3,
			probeHarness.GetExecutablePath(),
			nil,
			true,
			func() {
				ready = true
			},
			func() {
				ready = false
			},
			"test",
		)
		require.NoError(t, p.Start())
		defer func() {
			_ = p.Stop()
		}()

		go func() {
			time.Sleep(time.Millisecond * 500)
			probeHarness.SetReady()
		}()
		require.Eventually(
			t,
			func() bool {
				return ready == true
			},
			time.Second*5,
			time.Millisecond*100,
		)

		go func() {
			time.Sleep(time.Millisecond * 500)
			probeHarness.SetNotReady()
		}()
		require.Eventually(
			t,
			func() bool {
				return ready == false
			},
			time.Second*5,
			time.Millisecond*100,
		)
	})
}
