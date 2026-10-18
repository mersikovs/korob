package audit

import (
	"context"
	"sync"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

type observerRunner struct {
	id       string
	observer Observer
	queue    chan models.AuditEvent
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	ctx      context.Context
	logger   logger.Logger
}

func newObserverRunner(obs Observer, bufferSize int, logger logger.Logger) *observerRunner {
	ctx, cancel := context.WithCancel(context.Background())

	r := &observerRunner{
		id:       obs.GetID(),
		observer: obs,
		queue:    make(chan models.AuditEvent, bufferSize),
		cancel:   cancel,
		ctx:      ctx,
		logger:   logger,
	}

	r.wg.Add(1)
	go r.run()

	return r
}

func (r *observerRunner) run() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case event, ok := <-r.queue:
			if !ok {
				return
			}

			r.observer.Notify(r.ctx, event)
		}
	}
}

func (r *observerRunner) Shutdown() error {
	r.cancel()
	close(r.queue)
	r.wg.Wait()

	return r.observer.Close()
}
