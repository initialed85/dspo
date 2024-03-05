package fanin

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	_fanout "github.com/initialed85/dspo/pkg/fanout"
	"github.com/initialed85/dspo/pkg/managed_process"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("FaninOnce", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		fanout := _fanout.New(producer)
		defer func() {
			fanout.Close()
		}()

		consumer := make(chan managed_process.Log, 1024)
		f := New(consumer)

		consumer1, cancel1 := fanout.Subscribe()
		f.Consume(consumer1, cancel1)

		consumer2, cancel2 := fanout.Subscribe()
		f.Consume(consumer2, cancel2)

		consumer3, cancel3 := fanout.Subscribe()
		f.Consume(consumer3, cancel3)

		message := managed_process.Log{
			IsStdout:  true,
			IsStderr:  false,
			Timestamp: time.Now().UTC().UnixMilli(),
			Data:      []byte("some data"),
		}

		producer <- message

		assert.Equal(t, message, <-consumer)
		assert.Equal(t, message, <-consumer)
		assert.Equal(t, message, <-consumer)
	})

	t.Run("FaninInALoop", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		fanout := _fanout.New(producer)
		defer func() {
			fanout.Close()
		}()

		consumer := make(chan managed_process.Log, 1024)
		f := New(consumer)

		consumer1, cancel1 := fanout.Subscribe()
		f.Consume(consumer1, cancel1)

		consumer2, cancel2 := fanout.Subscribe()
		f.Consume(consumer2, cancel2)

		consumer3, cancel3 := fanout.Subscribe()
		f.Consume(consumer3, cancel3)

		for i := 0; i < 1024; i++ {
			message := managed_process.Log{
				IsStdout:  true,
				IsStderr:  false,
				Timestamp: time.Now().UTC().UnixMilli(),
				Data:      []byte(fmt.Sprintf("some data %v", i)),
			}

			producer <- message

			assert.Equal(t, message, <-consumer)
			assert.Equal(t, message, <-consumer)
			assert.Equal(t, message, <-consumer)
		}
	})

	t.Run("FaninForAPeriodOfTime", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		fanout := _fanout.New(producer)
		defer func() {
			fanout.Close()
		}()

		consumer := make(chan managed_process.Log, 1024)
		f := New(consumer)

		consumer1, cancel1 := fanout.Subscribe()
		f.Consume(consumer1, cancel1)

		consumer2, cancel2 := fanout.Subscribe()
		f.Consume(consumer2, cancel2)

		consumer3, cancel3 := fanout.Subscribe()
		f.Consume(consumer3, cancel3)

		deadline := time.Now().Add(time.Second * 1)

		for time.Now().Before(deadline) {
			message := managed_process.Log{
				IsStdout:  true,
				IsStderr:  false,
				Timestamp: time.Now().UTC().UnixMilli(),
				Data:      []byte(fmt.Sprintf("some data %v", rand.Int())),
			}

			producer <- message

			assert.Equal(t, message, <-consumer)
			assert.Equal(t, message, <-consumer)
			assert.Equal(t, message, <-consumer)
		}
	})
}
