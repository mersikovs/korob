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
	mu          sync.Mutex
}

func NewAgent(cnf *config.Config) *Agent {

	return &Agent{
		cnf:         cnf,
		lastMetrics: make(map[string]string),
		pollCount:   0,
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
		time.Sleep(time.Duration(a.cnf.PollInterval) * time.Second)
	}
}

func (a *Agent) SendMetrics() {
	for {
		time.Sleep(time.Duration(a.cnf.ReportInterval) * time.Second)
		a.mu.Lock()
		if len(a.lastMetrics) > 0 {
			url := fmt.Sprintf("http://%s/update", a.cnf.Address)

			err := PostMetrics(url, a.lastMetrics, "gauge")
			if err != nil {
				fmt.Println("Ошибка отправки метрик:", err)
			}

			err = PostMetrics(url, map[string]string{"pollCount": fmt.Sprintf("%v", a.pollCount)}, "counter")
			if err != nil {
				fmt.Println("Ошибка отправки счетчика:", err)
			}
		}
		a.mu.Unlock()
	}
}
