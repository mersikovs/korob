package agent

import (
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

const NameRandomField = "RandomValue"
const NamePollCountField = "PollCount"

type Agent struct {
	cnf    *config.Config
	client Sender
	logger logger.Logger

	key            []byte
	lastMetrics    map[string]models.Metrics
	pollCount      int64
	pollInterval   time.Duration
	reportInterval time.Duration

	metricsChan chan map[string]models.Metrics
	maxWorkers  int

	resetBeatCh chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	wgProducers sync.WaitGroup
	wgConsumers sync.WaitGroup

	mu sync.RWMutex
}

func NewAgent(cnf *config.Config, transport Sender, logger logger.Logger) *Agent {
	return &Agent{
		cnf:            cnf,
		client:         transport,
		logger:         logger,
		key:            cnf.Key,
		lastMetrics:    make(map[string]models.Metrics),
		pollCount:      0,
		maxWorkers:     cnf.RateLimit,
		pollInterval:   time.Duration(cnf.PollInterval * int(time.Second)),
		reportInterval: time.Duration(cnf.ReportInterval * int(time.Second)),
	}
}

func (a *Agent) Run(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.metricsChan = make(chan map[string]models.Metrics, 100)
	a.resetBeatCh = make(chan struct{}, 1)

	a.wgProducers.Add(1)
	go a.poolRuntimeMetrics()

	a.wgProducers.Add(1)
	go a.poolSystemMetrics()

	a.wgProducers.Add(1)
	go a.heartbeatSender()

	a.wgConsumers.Add(1)
	go a.startSenderPool()

	<-a.ctx.Done()

	a.shutdown()
}

func (a *Agent) saveMetrics(metrics map[string]models.Metrics, pollCount int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	maps.Copy(a.lastMetrics, metrics)
	a.pollCount = pollCount
}

func (a *Agent) heartbeatSender() {
	timer := time.NewTimer(a.reportInterval)
	defer timer.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-timer.C:
			hb := map[string]models.Metrics{}
			pollCount := int64(1)
			hb[NameRandomField] = models.Metrics{
				ID:    NameRandomField,
				MType: models.Gauge,
				Value: new(rand.Float64()),
			}

			hb[NamePollCountField] = models.Metrics{
				ID:    NamePollCountField,
				MType: models.Counter,
				Delta: &pollCount,
			}

			select {
			case a.metricsChan <- hb:
			case <-a.ctx.Done():
				return
			}

			timer.Reset(a.reportInterval)
		case <-a.resetBeatCh:
			timer.Reset(a.reportInterval)
		}
	}
}

func (a *Agent) poolRuntimeMetrics() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			metrics := a.collectRuntimeMetrics()

			select {
			case a.metricsChan <- metrics:
				a.scheduleNextSend()
			case <-a.ctx.Done():
				return
			}
		}
	}

}

func (a *Agent) collectRuntimeMetrics() map[string]models.Metrics {
	metrics := GetRuntimeMetrics()
	pollCount := a.getCountDiff(metrics)
	a.saveMetrics(metrics, pollCount)
	metrics[NameRandomField] = models.Metrics{
		ID:    NameRandomField,
		MType: models.Gauge,
		Value: new(rand.Float64()),
	}

	metrics[NamePollCountField] = models.Metrics{
		ID:    NamePollCountField,
		MType: models.Counter,
		Delta: &pollCount,
	}

	return metrics
}

func (a *Agent) poolSystemMetrics() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			metrics := a.collectSystemMetrics()

			select {
			case a.metricsChan <- metrics:
				a.scheduleNextSend()
			case <-a.ctx.Done():
				return
			}
		}
	}
}

func (a *Agent) collectSystemMetrics() map[string]models.Metrics {
	result := make(map[string]models.Metrics)

	if v, err := mem.VirtualMemory(); err == nil {
		vTotal := float64(v.Total)
		vFree := float64(v.Free)
		result["TotalMemory"] = models.Metrics{
			ID:    "TotalMemory",
			Value: &vTotal,
			MType: models.Gauge,
		}
		result["FreeMemory"] = models.Metrics{
			ID:    "FreeMemory",
			Value: &vFree,
			MType: models.Gauge,
		}
	}

	if perCPUPercent, err := cpu.Percent(0, true); err == nil && len(perCPUPercent) > 0 {
		for i, v := range perCPUPercent {
			result[fmt.Sprintf("%s%d", "CPUutilization", i)] = models.Metrics{
				ID:    fmt.Sprintf("%s%d", "CPUutilization", i),
				Value: &v,
				MType: models.Gauge,
			}
		}
	}

	pollCount := a.getCountDiff(result)
	a.saveMetrics(result, pollCount)
	result[NameRandomField] = models.Metrics{
		ID:    NameRandomField,
		MType: models.Gauge,
		Value: new(rand.Float64()),
	}

	result[NamePollCountField] = models.Metrics{
		ID:    NamePollCountField,
		MType: models.Counter,
		Delta: &pollCount,
	}

	return result
}

func (a *Agent) getCountDiff(newMetrics map[string]models.Metrics) int64 {
	countDiff := 0
	a.mu.RLock()

	for key, newMetric := range newMetrics {
		oldMetric, exists := a.lastMetrics[key]
		if !exists {
			countDiff++
			continue
		}

		if !float64PtrEqual(newMetric.Value, oldMetric.Value) ||
			!int64PtrEqual(newMetric.Delta, oldMetric.Delta) {
			countDiff++
		}
	}

	a.mu.RUnlock()

	return int64(countDiff)
}

func (a *Agent) startSenderPool() {
	var wg sync.WaitGroup
	for i := 0; i < a.maxWorkers; i++ {
		wg.Go(func() {
			for metrics := range a.metricsChan {
				a.sendMetrics(metrics)
			}
		})
	}
	wg.Wait()
}

func (a *Agent) sendMetrics(metrics map[string]models.Metrics) {

	if len(metrics) > 0 {
		updateURL := fmt.Sprintf("http://%s/updates/", a.cnf.Address)

		err := PostMetricsBatch(a.client, updateURL, metrics)
		if err != nil {
			fmt.Println("Произошли ошибки при отправке метрик:", err)
		}
	}

}

func (a *Agent) scheduleNextSend() {
	select {
	case a.resetBeatCh <- struct{}{}:
	default:
	}
}

func (a *Agent) shutdown() {

	producersDoneChan := make(chan struct{})

	go func() {
		a.wgProducers.Wait()
		close(producersDoneChan)
	}()

	select {
	case <-producersDoneChan:
		a.logger.Info("все источники данных завершили работу")
	case <-time.After(30 * time.Second):
		a.logger.Info("все источники данных не завершили работу и прерваны по таймауту")
	}

	close(a.metricsChan)

	consumersDoneChan := make(chan struct{})

	go func() {
		a.wgConsumers.Wait()
		close(consumersDoneChan)
	}()

	select {
	case <-consumersDoneChan:
		a.logger.Info("все получатели данных завершили работу")
	case <-time.After(30 * time.Second):
		a.logger.Info("все получатели данных не завершили работу и прерваны по таймауту")
	}
}
