package fanout

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/initialed85/dspo/pkg/managed_process"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("FanoutOnce", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		f := New(producer)
		defer func() {
			f.Close()
		}()

		consumer1, cancel1 := f.Subscribe()
		defer cancel1()

		consumer2, cancel2 := f.Subscribe()
		defer cancel2()

		consumer3, cancel3 := f.Subscribe()
		defer cancel3()

		message := managed_process.Log{
			IsStdout:  true,
			IsStderr:  false,
			Timestamp: time.Now().UTC().UnixMilli(),
			Data:      []byte("some data"),
		}

		producer <- message

		assert.Equal(t, message, <-consumer1)
		assert.Equal(t, message, <-consumer2)
		assert.Equal(t, message, <-consumer3)
	})

	t.Run("FanoutInALoop", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		f := New(producer)
		defer func() {
			f.Close()
		}()

		consumer1, cancel1 := f.Subscribe()
		defer cancel1()

		consumer2, cancel2 := f.Subscribe()
		defer cancel2()

		consumer3, cancel3 := f.Subscribe()
		defer cancel3()

		for i := 0; i < 1024; i++ {
			message := managed_process.Log{
				IsStdout:  true,
				IsStderr:  false,
				Timestamp: time.Now().UTC().UnixMilli(),
				Data:      []byte(fmt.Sprintf("some data %v", i)),
			}

			producer <- message

			assert.Equal(t, message, <-consumer1)
			assert.Equal(t, message, <-consumer2)
			assert.Equal(t, message, <-consumer3)
		}
	})

	t.Run("FanoutForAPeriodOfTime", func(t *testing.T) {
		producer := make(chan managed_process.Log)
		f := New(producer)
		defer func() {
			f.Close()
		}()

		consumer1, cancel1 := f.Subscribe()
		defer cancel1()

		consumer2, cancel2 := f.Subscribe()
		defer cancel2()

		consumer3, cancel3 := f.Subscribe()
		defer cancel3()

		deadline := time.Now().Add(time.Second * 1)

		for time.Now().Before(deadline) {
			message := managed_process.Log{
				IsStdout:  true,
				IsStderr:  false,
				Timestamp: time.Now().UTC().UnixMilli(),
				Data:      []byte(fmt.Sprintf("some data %v", rand.Int())),
			}

			producer <- message

			assert.Equal(t, message, <-consumer1)
			assert.Equal(t, message, <-consumer2)
			assert.Equal(t, message, <-consumer3)
		}
	})
}
