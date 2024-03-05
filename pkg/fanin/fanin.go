package fanin

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/initialed85/dspo/pkg/managed_process"
)

type Fanin struct {
	messages                chan managed_process.Log
	consumerByConsumerID    map[uuid.UUID]chan managed_process.Log
	unsubscribeByConsumerID map[uuid.UUID]func()
	mu                      sync.Mutex
	ctx                     context.Context
	cancel                  context.CancelFunc
}

func New(consumer chan managed_process.Log) *Fanin {
	f := Fanin{
		messages:                consumer,
		consumerByConsumerID:    make(map[uuid.UUID]chan managed_process.Log),
		unsubscribeByConsumerID: make(map[uuid.UUID]func()),
	}

	f.ctx, f.cancel = context.WithCancel(context.Background())

	go f.runConsume()

	return &f

}

func (f *Fanin) runConsume() {
	for {

		select {
		case <-f.ctx.Done():
			return
		default:
		}

		f.mu.Lock()
		consumerByConsumerID := f.consumerByConsumerID
		f.mu.Unlock()

		for _, consumer := range consumerByConsumerID {
			select {
			case message := <-consumer:
				f.messages <- message
			default:
			}
		}
	}
}

func (f *Fanin) Close() {
	f.cancel()

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, unsubscribe := range f.unsubscribeByConsumerID {
		unsubscribe()
	}

	f.consumerByConsumerID = make(map[uuid.UUID]chan managed_process.Log)
	f.unsubscribeByConsumerID = make(map[uuid.UUID]func())

	f.ctx, f.cancel = context.WithCancel(context.Background())
}

func (f *Fanin) Consume(consumer chan managed_process.Log, unsubscribe func()) func() {
	consumerID := uuid.New()

	wrappedUnsubscribe := func() {
		f.mu.Lock()
		_, ok := f.consumerByConsumerID[consumerID]
		if !ok {
			f.mu.Unlock()
			return
		}

		f.mu.Lock()
		delete(f.consumerByConsumerID, consumerID)
		delete(f.unsubscribeByConsumerID, consumerID)
		f.mu.Unlock()

		unsubscribe()
	}

	f.mu.Lock()
	f.consumerByConsumerID[consumerID] = consumer
	f.unsubscribeByConsumerID[consumerID] = wrappedUnsubscribe
	f.mu.Unlock()

	return wrappedUnsubscribe
}
