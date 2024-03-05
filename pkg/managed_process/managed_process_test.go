package managed_process

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	returnCodes := make(chan int)
	onExit := func(returnCode int) {
		select {
		case returnCodes <- returnCode:
		default:
		}
	}

	waitUntilNotRunning := func(m *ManagedProcess, timeout time.Duration) error {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timed out waiting for not running")
		case <-returnCodes:
			return nil
		}
	}

	logs := make(chan Log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	datas := make([]string, 0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case l := <-logs:
				datas = append(datas, string(l.Data))
			}
		}
	}()

	t.Run("RestartPolicyNeverZeroReturnCode", func(t *testing.T) {
		m := New(
			logs,
			Never,
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'",
			nil,
			true,
			time.Second*1,
			onExit,
			"managed_process_test",
		)
		require.NoError(t, m.Start())
		defer func() {
			_ = m.Stop()
			datas = make([]string, 0)
		}()

		require.NoError(t, waitUntilNotRunning(m, time.Second*1))

		assert.Equal(
			t,
			[]string{"first\n", "second\n"},
			datas,
		)
	})

	t.Run("RestartPolicyUnlessStoppedZeroReturnCode", func(t *testing.T) {
		m := New(
			logs,
			UnlessStopped,
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'",
			nil,
			true,
			time.Second*1,
			onExit,
			"managed_process_test",
		)
		require.NoError(t, m.Start())
		defer func() {
			_ = m.Stop()
			datas = make([]string, 0)
		}()

		time.Sleep(time.Second * 2)

		assert.Equal(
			t,
			[]string{"first\n", "second\n", "first\n", "second\n"},
			datas,
		)
	})

	t.Run("RestartPolicyOnFailureZeroReturnCode", func(t *testing.T) {
		m := New(
			logs,
			OnFailure,
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'",
			nil,
			true,
			time.Second*1,
			onExit,
			"managed_process_test",
		)
		require.NoError(t, m.Start())
		defer func() {
			_ = m.Stop()
			datas = make([]string, 0)
		}()

		require.NoError(t, waitUntilNotRunning(m, time.Second*1))

		assert.Equal(
			t,
			[]string{"first\n", "second\n"},
			datas,
		)
	})

	t.Run("RestartPolicyOnFailureNonZeroReturnCode", func(t *testing.T) {
		m := New(
			logs,
			OnFailure,
			"/bin/bash",
			"echo 'first'; sleep 0.1; echo 'second'; exit 1",
			nil,
			true,
			time.Second*1,
			onExit,
			"managed_process_test",
		)
		require.NoError(t, m.Start())
		defer func() {
			_ = m.Stop()
			datas = make([]string, 0)
		}()

		time.Sleep(time.Second * 2)

		assert.Equal(
			t,
			[]string{"first\n", "second\n", "first\n", "second\n"},
			datas,
		)
	})
}
