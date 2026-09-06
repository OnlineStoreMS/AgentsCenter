package scheduler

import (
	"log"
	"time"

	"agentscenter/internal/service"
)

// JobRetentionScheduler 定期清理过期执行记录。
type JobRetentionScheduler struct {
	svc           *service.AgentService
	retentionDays int
	stopCh        chan struct{}
}

func NewJobRetentionScheduler(svc *service.AgentService, retentionDays int) *JobRetentionScheduler {
	if retentionDays <= 0 {
		retentionDays = 3
	}
	return &JobRetentionScheduler{
		svc:           svc,
		retentionDays: retentionDays,
		stopCh:        make(chan struct{}),
	}
}

func (s *JobRetentionScheduler) Start() {
	go s.loop()
}

func (s *JobRetentionScheduler) Stop() {
	close(s.stopCh)
}

func (s *JobRetentionScheduler) loop() {
	// 启动后稍等再清一次，之后每小时
	timer := time.NewTimer(2 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			n, err := s.svc.CleanupOldJobs(s.retentionDays)
			if err != nil {
				log.Printf("[job-retention] cleanup failed: %v", err)
			} else if n > 0 {
				log.Printf("[job-retention] deleted %d job(s) older than %d day(s)", n, s.retentionDays)
			}
			timer.Reset(time.Hour)
		}
	}
}
