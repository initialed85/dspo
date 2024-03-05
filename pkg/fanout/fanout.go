package fanout

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/initialed85/dspo/pkg/managed_process"
)

const (
	depth = 1024
)

type Fanout struct {
	messages             chan managed_process.Log
	consumerByConsumerID map[uuid.UUID]chan managed_process.Log
	mu                   sync.Mutex
	ctx                  context.Context
	cancel               context.CancelFunc
}

func New(messages chan managed_process.Log) *Fanout {
	f := Fanout{
		messages:             messages,
		consumerByConsumerID: make(map[uuid.UUID]chan managed_process.Log),
	}

	f.ctx, f.cancel = context.WithCancel(context.Background())

	go f.runPublish()

	return &f
}

func (f *Fanout) runPublish() {
	for {
		select {
		case <-f.ctx.Done():
			return
		case message := <-f.messages:
			f.mu.Lock()
			consumerByConsumerID := f.consumerByConsumerID
			f.mu.Unlock()

			for _, consumer := range consumerByConsumerID {
				select {
				case consumer <- message:
				default:
				}
			}
		}
	}
}

func (f *Fanout) Close() {
	f.cancel()

	f.mu.Lock()
	defer f.mu.Unlock()

	f.consumerByConsumerID = make(map[uuid.UUID]chan managed_process.Log)

	f.ctx, f.cancel = context.WithCancel(context.Background())
}

func (f *Fanout) Subscribe() (chan managed_process.Log, func()) {
	consumerID := uuid.New()
	consumer := make(chan managed_process.Log, depth)

	f.mu.Lock()
	f.consumerByConsumerID[consumerID] = consumer
	f.mu.Unlock()

	unsubscribe := func() {
		f.mu.Lock()
		delete(f.consumerByConsumerID, consumerID)
		f.mu.Unlock()
	}

	return consumer, unsubscribe
}
