package audit

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

// Dispatcher manages a pool of background workers that asynchronously process
// audit events from a buffered queue and distribute them to registered Observers.
type Dispatcher struct {
	mu         sync.RWMutex
	observers  map[string]*observerRunner
	queue      chan models.AuditEvent
	maxWorkers int
	queueSize  int
	wg         sync.WaitGroup
	logger     logger.Logger
	closed     bool

	ctx    context.Context
	cancel context.CancelFunc
}

// NewDispatcher creates a Dispatcher with the specified queue size and worker count.
// It applies safe defaults for invalid values and immediately starts the background worker pool.
func NewDispatcher(queueSize int, maxWorkers int, logger logger.Logger) *Dispatcher {
	if queueSize <= 0 {
		queueSize = 1000
	}
	if maxWorkers <= 0 {
		maxWorkers = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Dispatcher{
		observers:  make(map[string]*observerRunner),
		queue:      make(chan models.AuditEvent, queueSize),
		maxWorkers: maxWorkers,
		queueSize:  queueSize,
		logger:     logger,

		ctx:    ctx,
		cancel: cancel,
	}

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
	defer d.mu.RUnlock()

	for _, runner := range d.observers {
		select {
		case runner.queue <- event:
			// OK
		default:
			if d.logger != nil {
				d.logger.Info("observer queue full, event dropped", "observer_id", runner.id)
			}
		}
	}
}

func (d *Dispatcher) Start() {
	d.startWorkerPool()
}

// Register adds an Observer to the dispatcher.
func (d *Dispatcher) Register(o Observer) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}

	runner := newObserverRunner(o, d.queueSize, d.logger)

	d.observers[o.GetID()] = runner
}

// Publish sends an audit event to the processing queue.
func (d *Dispatcher) Publish(event models.AuditEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	isClosed := d.closed

	if isClosed {
		return
	}

	select {
	case d.queue <- event:
	default:
		if d.logger != nil {
			d.logger.Info("audit queue full, dropping event", "ts", event.TS)
		}
	}
}

// Shutdown initiates a graceful shutdown of the worker pool.
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	var shutdownErr error

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()

	close(d.queue)

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if d.logger != nil {
			d.logger.Info("main worker pool shut down gracefully")
		}

	case <-ctx.Done():
		if d.logger != nil {
			d.logger.Info("main worker pool shutdown timed out", "error", ctx.Err())
		}

		shutdownErr = fmt.Errorf("worker pool shutdown timed out: %w", ctx.Err())
	}
	d.cancel()

	d.mu.RLock()
	observersToShutdown := maps.Clone(d.observers)
	d.mu.RUnlock()

	var closeErrs []error
	for id, runner := range observersToShutdown {
		if runner == nil || runner.observer == nil {
			continue
		}

		if err := runner.Shutdown(); err != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close observer %q: %w", id, err))
		}
	}

	if shutdownErr != nil {
		closeErrs = append([]error{shutdownErr}, closeErrs...)
	}

	if len(closeErrs) > 0 {
		return errors.Join(closeErrs...)
	}

	if d.logger != nil {
		d.logger.Info("all observers shut down gracefully")
	}

	return nil
}
