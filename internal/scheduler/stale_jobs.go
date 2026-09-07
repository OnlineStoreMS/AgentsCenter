package scheduler

import (
	"log"
	"time"

	"agentscenter/internal/service"
)

// StaleJobScheduler 定期回收卡在 claimed/running 的僵尸任务。
type StaleJobScheduler struct {
	svc          *service.AgentService
	staleMinutes int
	stopCh       chan struct{}
}

func NewStaleJobScheduler(svc *service.AgentService, staleMinutes int) *StaleJobScheduler {
	if staleMinutes <= 0 {
		staleMinutes = 60
	}
	return &StaleJobScheduler{
		svc:          svc,
		staleMinutes: staleMinutes,
		stopCh:       make(chan struct{}),
	}
}

func (s *StaleJobScheduler) Start() {
	go s.loop()
}

func (s *StaleJobScheduler) Stop() {
	close(s.stopCh)
}

func (s *StaleJobScheduler) loop() {
	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			n, err := s.svc.RecoverStaleInFlightJobs(s.staleMinutes)
			if err != nil {
				log.Printf("[job-stale] recover failed: %v", err)
			} else if n > 0 {
				log.Printf("[job-stale] failed %d stale claimed/running job(s) older than %d minute(s)", n, s.staleMinutes)
			}
			timer.Reset(2 * time.Minute)
		}
	}
}
