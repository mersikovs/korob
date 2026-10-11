package audit

import (
	"context"
	"sync"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

type Dispatcher struct {
	mu         sync.RWMutex
	observers  map[string]Observer
	queue      chan models.AuditEvent
	maxWorkers int
	wg         sync.WaitGroup
	logger     logger.Logger
	closed     bool

	ctx    context.Context
	cancel context.CancelFunc
}

func NewDispatcher(queueSize int, maxWorkers int, logger logger.Logger) *Dispatcher {
	if queueSize <= 0 {
		queueSize = 1000
	}
	if maxWorkers <= 0 {
		maxWorkers = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Dispatcher{
		observers:  make(map[string]Observer),
		queue:      make(chan models.AuditEvent, queueSize),
		maxWorkers: maxWorkers,
		logger:     logger,

		ctx:    ctx,
		cancel: cancel,
	}

	d.startWorkerPool()
	return d
}

func (d *Dispatcher) startWorkerPool() {
	for i := 0; i < d.maxWorkers; i++ {
		d.wg.Go(func() {
			for event := range d.queue {
				d.processEvent(event)
			}
		})
	}
}

func (d *Dispatcher) processEvent(event models.AuditEvent) {
	d.mu.RLock()
	observers := make([]Observer, 0, len(d.observers))
	for _, obs := range d.observers {
		observers = append(observers, obs)
	}
	d.mu.RUnlock()

	for _, obs := range observers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if d.logger != nil {
						d.logger.Info("observer panicked", "observer_id", obs.GetID(), "error", r)
					}
				}
			}()
			obs.Notify(d.ctx, event)
		}()
	}
}

func (d *Dispatcher) Register(o Observer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.closed {
		d.observers[o.GetID()] = o
	}
}

func (d *Dispatcher) Publish(event models.AuditEvent) {
	d.mu.RLock()
	isClosed := d.closed
	d.mu.RUnlock()

	if isClosed {
		return
	}

	select {
	case d.queue <- event:
	default:
		if d.logger != nil {
			d.logger.Info("audit queue full, dropping event", "ts", event.Ts)
		}
	}
}

func (d *Dispatcher) Shutdown(ctx context.Context) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.cancel()
	d.mu.Unlock()

	close(d.queue)

	d.wg.Wait()

	if d.logger != nil {
		d.logger.Info("audit worker pool shut down gracefully")
	}
}
