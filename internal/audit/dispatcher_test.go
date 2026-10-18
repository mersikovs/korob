package audit

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	models "github.com/mersikovs/korob.git/internal/model"
)

type mockObserver struct {
	notifyCalled atomic.Int32
	closeCalled  atomic.Int32
}

func (m *mockObserver) Notify(ctx context.Context, event models.AuditEvent) {
	time.Sleep(5 * time.Millisecond)
	m.notifyCalled.Add(1)
}

func (m *mockObserver) Close() error {
	m.closeCalled.Add(1)
	return nil
}

func (m *mockObserver) GetID() string {
	return "mock-observer"
}

func TestDispatcher_Shutdown(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(t *testing.T, d *Dispatcher) *atomic.Int32
		shutdownTimeout   time.Duration
		wantProcessed     int32
		maxDuration       time.Duration
		callShutdownTwice bool
	}{
		{
			name: "graceful_shutdown_processes_all_items",
			setup: func(t *testing.T, d *Dispatcher) *atomic.Int32 {
				var processed atomic.Int32

				d.wg.Go(func() {
					for {
						select {
						case event, ok := <-d.queue:
							if !ok {
								return
							}

							for _, runner := range d.observers {
								runner.observer.Notify(runner.ctx, event)
								processed.Add(1)
							}
						case <-d.ctx.Done():
							return
						}
					}
				})

				for range 5 {
					d.queue <- models.AuditEvent{
						TS:        time.Now().Unix(),
						Metrics:   []string{"1", "2"},
						IPAddress: "8.8.8.8",
					}
				}
				return &processed
			},
			shutdownTimeout: 2 * time.Second,
			wantProcessed:   5,
			maxDuration:     500 * time.Millisecond,
		},
		{
			name: "timeout_forces_cancellation_and_prevents_hang",
			setup: func(t *testing.T, d *Dispatcher) *atomic.Int32 {
				var processed atomic.Int32

				d.wg.Go(func() {
					for {
						select {
						case <-d.queue:

							time.Sleep(3 * time.Second)
							processed.Add(1)
						case <-d.ctx.Done():
							return
						}
					}
				})

				d.queue <- models.AuditEvent{
					TS:        time.Now().Unix(),
					Metrics:   []string{"stuck"},
					IPAddress: "8.8.8.8",
				}
				return &processed
			},
			shutdownTimeout: 50 * time.Millisecond,
			wantProcessed:   0,
			maxDuration:     200 * time.Millisecond,
		},
		{
			name: "idempotent_shutdown_does_not_panic",
			setup: func(t *testing.T, d *Dispatcher) *atomic.Int32 {
				return &atomic.Int32{}
			},
			shutdownTimeout:   1 * time.Second,
			wantProcessed:     0,
			maxDuration:       100 * time.Millisecond,
			callShutdownTwice: true,
		},
		{
			name: "concurrent_publish_during_shutdown_no_panic",
			setup: func(t *testing.T, d *Dispatcher) *atomic.Int32 {
				var processed atomic.Int32

				d.wg.Go(func() {
					for {
						select {
						case _, ok := <-d.queue:
							if !ok {
								return
							}
							processed.Add(1)
						case <-d.ctx.Done():
							return
						}
					}
				})

				for i := range 100 {
					go func(id int) {
						for range 100 {
							d.Publish(models.AuditEvent{
								TS:        time.Now().Unix(),
								Metrics:   []string{strconv.Itoa(id)},
								IPAddress: "8.8.8.8",
							})
						}
					}(i)
				}

				return &processed
			},
			shutdownTimeout:   500 * time.Millisecond,
			wantProcessed:     0,
			maxDuration:       2 * time.Second,
			callShutdownTwice: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := t.Context()
			ctx, cancel := context.WithCancel(testCtx)

			runnerCtx, runnerCancel := context.WithCancel(context.Background())
			testRunner := &observerRunner{
				id:       "test-runner",
				observer: &mockObserver{},
				queue:    make(chan models.AuditEvent, 10),
				ctx:      runnerCtx,
				cancel:   runnerCancel,
			}

			d := &Dispatcher{
				queue:      make(chan models.AuditEvent, 10),
				maxWorkers: 2,
				queueSize:  10,
				ctx:        ctx,
				cancel:     cancel,
				observers: map[string]*observerRunner{
					"observer1": testRunner,
				},
			}

			processedCount := tt.setup(t, d)

			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()

				shutdownCtx, shutdownCancel := context.WithTimeout(testCtx, tt.shutdownTimeout)
				defer shutdownCancel()

				start := time.Now()
				_ = d.Shutdown(shutdownCtx)
				duration := time.Since(start)

				if tt.callShutdownTwice {
					_ = d.Shutdown(shutdownCtx)
				}

				if duration > tt.maxDuration {
					t.Errorf("Shutdown too long: %v (max: %v)", duration, tt.maxDuration)
				}

				if tt.wantProcessed > 0 {
					if got := processedCount.Load(); got != tt.wantProcessed {
						t.Errorf("processedCount error: got %d, want %d", got, tt.wantProcessed)
					}
				}

				if panicked {
					t.Error("Shutdown panic")
				}

				if tt.name == "timeout_forces_cancellation_and_prevents_hang" {
					select {
					case <-d.ctx.Done():
						// ok, context cancsel
					default:
						t.Error("context don't cansel")
					}
				}
			}()
		})
	}
}
