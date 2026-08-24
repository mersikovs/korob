package agent

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/mersikovs/korob.git/internal/config"
	models "github.com/mersikovs/korob.git/internal/model"
)

const DefaultPollInterval = 2
const DefaultReportInterval = 10
const NameRandomField = "RandomValue"

type Agent struct {
	cnf         *config.Config
	client      Sender
	key         []byte
	lastMetrics map[string]models.Metrics
	pollCount   int

	mu sync.RWMutex
}

func NewAgent(cnf *config.Config, transport Sender) *Agent {
	return &Agent{
		cnf:         cnf,
		client:      transport,
		key:         cnf.Key,
		lastMetrics: make(map[string]models.Metrics),
		pollCount:   0,
	}
}

func (a *Agent) Run() {
	go a.getMetricsFromRuntime()
	go a.sendRuntimeMetrics()
}

func (a *Agent) saveMetrics(metrics map[string]models.Metrics, pollCount int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastMetrics = metrics
	a.pollCount = pollCount
}

func (a *Agent) getSavedMetrics() (map[string]models.Metrics, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastMetrics, a.pollCount
}

func (a *Agent) getMetricsFromRuntime() {
	for {

		metrics := GetRuntimeMetrics()
		lastMetrics, _ := a.getSavedMetrics()
		pollCount := GetCountDiff(lastMetrics, metrics)
		a.saveMetrics(metrics, pollCount)
		metrics[NameRandomField] = models.Metrics{
			ID:    NameRandomField,
			MType: "gauge",
			Value: Float64Ptr(rand.Float64()),
		}

		time.Sleep(time.Duration(a.cnf.PollInterval) * time.Second)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func (a *Agent) sendRuntimeMetrics() {
	for {
		time.Sleep(time.Duration(a.cnf.ReportInterval) * time.Second)
		lastMetrics, pollCount := a.getSavedMetrics()

		if len(lastMetrics) > 0 {
			updateURL := fmt.Sprintf("http://%s/updates/", a.cnf.Address)

			lastMetrics["PollCount"] = models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: int64Ptr(int64(pollCount)),
			}

			err := PostMetricsBatch(a.client, updateURL, lastMetrics)
			if err != nil {
				fmt.Println("Произошли ошибки при отправке метрик:", err)
			}
		}

	}
}
