package agent

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

const DefaultPollInterval = 2
const DefaultReportInterval = 10
const NameRandomField = "randomValue"

type Agent struct {
	lastMetrics    map[string]string
	pollCount      int
	pollInterval   int
	reportInterval int
	mu             sync.Mutex
}

func NewAgent(pollInterval, reportInterval int) *Agent {

	if pollInterval <= DefaultPollInterval {
		pollInterval = DefaultPollInterval
	}
	if reportInterval <= DefaultReportInterval {
		reportInterval = DefaultReportInterval
	}

	return &Agent{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		lastMetrics:    make(map[string]string),
		pollCount:      0,
	}
}

func (a *Agent) Run() {
	go a.GetMetrics()
	go a.SendMetrics()
}

func (a *Agent) GetMetrics() {
	for {
		a.mu.Lock()
		metrics := GetRuntimeMetrics()
		a.pollCount = GetCountDiff(a.lastMetrics, metrics)
		a.lastMetrics = metrics
		metrics[NameRandomField] = fmt.Sprintf("%v", rand.Float64())
		a.mu.Unlock()
		time.Sleep(time.Duration(a.pollInterval) * time.Second)
	}
}

func (a *Agent) SendMetrics() {
	for {
		time.Sleep(time.Duration(a.reportInterval) * time.Second)
		a.mu.Lock()
		if len(a.lastMetrics) > 0 {
			err := PostMetrics("http://localhost:8080/update", a.lastMetrics, "gauge")
			if err != nil {
				fmt.Println("Ошибка отправки метрик:", err)
			}

			err = PostMetrics("http://localhost:8080/update", map[string]string{"pollCount": fmt.Sprintf("%v", a.pollCount)}, "counter")
			if err != nil {
				fmt.Println("Ошибка отправки счетчика:", err)
			}
		}
		a.mu.Unlock()
	}
}
