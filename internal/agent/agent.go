package agent

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/mersikovs/korob.git/internal/config"
)

const DefaultPollInterval = 2
const DefaultReportInterval = 10
const NameRandomField = "randomValue"

type Agent struct {
	lastMetrics map[string]string
	pollCount   int
	cnf         *config.Config
	mu          sync.RWMutex
}

func NewAgent(cnf *config.Config) *Agent {
	return &Agent{
		cnf:         cnf,
		lastMetrics: make(map[string]string),
		pollCount:   0,
	}
}

func (a *Agent) Run() {
	go a.getMetricsFromRuntime()
	go a.sendRuntimeMetrics()
}

func (a *Agent) saveMetrics(metrics map[string]string, pollCount int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastMetrics = metrics
	a.pollCount = pollCount
}

func (a *Agent) getSavedMetrics() (map[string]string, int) {
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
		metrics[NameRandomField] = fmt.Sprintf("%v", rand.Float64())

		time.Sleep(time.Duration(a.cnf.PollInterval) * time.Second)
	}
}

func (a *Agent) sendRuntimeMetrics() {
	for {
		time.Sleep(time.Duration(a.cnf.ReportInterval) * time.Second)
		lastMetrics, pollCount := a.getSavedMetrics()

		if len(lastMetrics) > 0 {
			url := fmt.Sprintf("http://%s/update", a.cnf.Address)

			err := PostMetrics(url, lastMetrics, "gauge")
			if err != nil {
				fmt.Println("Произошли ошибки при отправке метрик:", err)
			}

			err = PostMetrics(url, map[string]string{"pollCount": fmt.Sprintf("%v", pollCount)}, "counter")
			if err != nil {
				fmt.Println("Произошли ошибки при отправке метрик:", err)
			}
		}

	}
}
