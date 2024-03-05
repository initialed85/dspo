package system

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/initialed85/dspo/pkg/common"
	"github.com/initialed85/dspo/test"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		serviceArgs1a := test.NewMockService("service_1a", []string{})
		serviceArgs1b := test.NewMockService("service_1b", []string{})
		serviceArgs2 := test.NewMockService("service_2", []string{serviceArgs1b.ServiceArgs.Name})
		serviceArgs3a := test.NewMockService("service_3a", []string{serviceArgs2.ServiceArgs.Name})
		serviceArgs3b := test.NewMockService("service_3b", []string{serviceArgs2.ServiceArgs.Name})
		serviceArgs4 := test.NewMockService("service_4", []string{serviceArgs3a.ServiceArgs.Name, serviceArgs3b.ServiceArgs.Name})

		s := New(
			[]common.ServiceArgs{
				serviceArgs1a.ServiceArgs,
				serviceArgs1b.ServiceArgs,
				serviceArgs2.ServiceArgs,
				serviceArgs3a.ServiceArgs,
				serviceArgs3b.ServiceArgs,
				serviceArgs4.ServiceArgs,
			},
			"test",
		)
		require.NoError(t, s.Start())
		defer func() {
			require.NoError(t, s.Stop())
		}()

		consumer, unsubscribe, err := s.SubscribeToLogs()
		require.NoError(t, err)
		defer unsubscribe()
		go func() {
			for {
				select {
				case _ = <-ctx.Done():
					return
				case message := <-consumer:
					log.Printf("%v\t @ %v: %v", message.Name, message.Timestamp, string(message.Data))
				}
			}
		}()

		service1a := s.ServiceByName()["service_1a"]
		service1b := s.ServiceByName()["service_1b"]
		service2 := s.ServiceByName()["service_2"]
		service3a := s.ServiceByName()["service_3a"]
		service3b := s.ServiceByName()["service_3b"]
		service4 := s.ServiceByName()["service_4"]

		require.Eventually(t, service1a.Started, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service1b.Started, time.Second*1, time.Millisecond*50)

		serviceArgs1a.StartupProbeHarness.SetReady()
		serviceArgs1a.LivenessProbeHarness.SetReady()
		require.Eventually(t, service1a.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service1a.LivenessReady, time.Second*1, time.Millisecond*50)

		serviceArgs1b.StartupProbeHarness.SetReady()
		serviceArgs1b.LivenessProbeHarness.SetReady()
		require.Eventually(t, service1b.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service1b.LivenessReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service2.Started, time.Second*1, time.Millisecond*50)

		serviceArgs2.StartupProbeHarness.SetReady()
		serviceArgs2.LivenessProbeHarness.SetReady()
		require.Eventually(t, service2.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service2.LivenessReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service3a.Started, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service3b.Started, time.Second*1, time.Millisecond*50)

		serviceArgs3a.StartupProbeHarness.SetReady()
		serviceArgs3a.LivenessProbeHarness.SetReady()
		serviceArgs3b.StartupProbeHarness.SetReady()
		serviceArgs3b.LivenessProbeHarness.SetReady()
		require.Eventually(t, service3a.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service3a.LivenessReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service3b.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service3b.LivenessReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service4.Started, time.Second*1, time.Millisecond*50)

		serviceArgs4.StartupProbeHarness.SetReady()
		serviceArgs4.LivenessProbeHarness.SetReady()
		require.Eventually(t, service4.StartupReady, time.Second*1, time.Millisecond*50)
		require.Eventually(t, service4.LivenessReady, time.Second*1, time.Millisecond*50)
	})
}
