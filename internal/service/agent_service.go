package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agentscenter/internal/dto"
	"agentscenter/internal/model"
	"agentscenter/internal/repo"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const onlineWithin = 90 * time.Second

type AgentService struct {
	repos *repo.Repository
}

func NewAgentService(repos *repo.Repository) *AgentService {
	return &AgentService{repos: repos}
}

func (s *AgentService) Register(tenantID uint64, in *dto.AgentRegisterInput) (*dto.AgentRegisterResult, error) {
	machineID := strings.TrimSpace(in.MachineID)
	if machineID == "" {
		return nil, fmt.Errorf("%w: machineId 必填", ErrBadRequest)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = machineID
	}
	skills := normalizeSkills(in.Skills)
	skillsJSON, _ := json.Marshal(skills)

	var existing model.Agent
	err := s.repos.DB.Where("machine_id = ?", machineID).First(&existing).Error
	if err == nil {
		existing.Name = name
		existing.Hostname = strings.TrimSpace(in.Hostname)
		existing.OS = strings.TrimSpace(in.OS)
		existing.AgentVersion = strings.TrimSpace(in.AgentVersion)
		existing.SkillsJSON = string(skillsJSON)
		existing.Status = model.AgentStatusOnline
		now := time.Now()
		existing.LastHeartbeat = &now
		if err := s.repos.DB.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &dto.AgentRegisterResult{
			AgentID:     existing.ID,
			AgentKey:    existing.AgentKey,
			AgentSecret: existing.AgentSecret,
			Name:        existing.Name,
			Skills:      skills,
		}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	key, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	secret, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	agent := model.Agent{
		TenantID:      tenantID,
		MachineID:     machineID,
		Name:          name,
		Hostname:      strings.TrimSpace(in.Hostname),
		OS:            strings.TrimSpace(in.OS),
		AgentVersion:  strings.TrimSpace(in.AgentVersion),
		AgentKey:      "agk_" + key,
		AgentSecret:   "ags_" + secret,
		Status:        model.AgentStatusOnline,
		SkillsJSON:    string(skillsJSON),
		LastHeartbeat: &now,
	}
	if err := s.repos.DB.Create(&agent).Error; err != nil {
		return nil, err
	}
	return &dto.AgentRegisterResult{
		AgentID:     agent.ID,
		AgentKey:    agent.AgentKey,
		AgentSecret: agent.AgentSecret,
		Name:        agent.Name,
		Skills:      skills,
	}, nil
}

func (s *AgentService) Authenticate(key, secret string) (*model.Agent, error) {
	key = strings.TrimSpace(key)
	secret = strings.TrimSpace(secret)
	if key == "" || secret == "" {
		return nil, ErrAgentAuth
	}
	var agent model.Agent
	if err := s.repos.DB.Where("agent_key = ? AND agent_secret = ?", key, secret).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAgentAuth
		}
		return nil, err
	}
	return &agent, nil
}

func (s *AgentService) Heartbeat(agent *model.Agent, in *dto.AgentHeartbeatInput) (*dto.AgentHeartbeatResult, error) {
	now := time.Now()
	updates := map[string]any{
		"status":         model.AgentStatusOnline,
		"last_heartbeat": now,
		"updated_at":     now,
	}
	if in != nil {
		if v := strings.TrimSpace(in.Hostname); v != "" {
			updates["hostname"] = v
		}
		if v := strings.TrimSpace(in.OS); v != "" {
			updates["os"] = v
		}
		if v := strings.TrimSpace(in.AgentVersion); v != "" {
			updates["agent_version"] = v
		}
		if len(in.Skills) > 0 {
			b, _ := json.Marshal(normalizeSkills(in.Skills))
			updates["skills_json"] = string(b)
		}
	}
	if err := s.repos.DB.Model(agent).Updates(updates).Error; err != nil {
		return nil, err
	}

	if in != nil && in.Shops != nil {
		if err := s.upsertShops(agent, in.Shops, now); err != nil {
			return nil, err
		}
	}

	var pending int64
	_ = s.repos.DB.Model(&model.AgentJob{}).
		Where("tenant_id = ? AND status = ?", agent.TenantID, model.JobStatusPending).
		Count(&pending).Error

	return &dto.AgentHeartbeatResult{
		ServerTime:        now.Format(time.RFC3339),
		PendingJobs:       pending,
		HeartbeatInterval: 20,
	}, nil
}

func (s *AgentService) upsertShops(agent *model.Agent, shops []dto.AgentShopReport, now time.Time) error {
	seen := make(map[string]struct{}, len(shops))
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		for _, sh := range shops {
			platform := strings.TrimSpace(sh.Platform)
			shopID := strings.TrimSpace(sh.PlatformShopID)
			if platform == "" || shopID == "" {
				continue
			}
			key := platform + "|" + shopID
			seen[key] = struct{}{}
			caps, _ := json.Marshal(sh.Capabilities)
			status := strings.TrimSpace(sh.Status)
			if status == "" {
				status = model.ShopStatusActive
			}
			var row model.AgentShop
			err := tx.Where("agent_id = ? AND platform = ? AND platform_shop_id = ?", agent.ID, platform, shopID).
				First(&row).Error
			if err == gorm.ErrRecordNotFound {
				row = model.AgentShop{
					TenantID:         agent.TenantID,
					AgentID:          agent.ID,
					Platform:         platform,
					PlatformShopID:   shopID,
					PlatformShopName: strings.TrimSpace(sh.PlatformShopName),
					BrowserChannel:   strings.TrimSpace(sh.BrowserChannel),
					CapabilitiesJSON: string(caps),
					Status:           status,
					LastSeenAt:       &now,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			row.PlatformShopName = strings.TrimSpace(sh.PlatformShopName)
			row.BrowserChannel = strings.TrimSpace(sh.BrowserChannel)
			row.CapabilitiesJSON = string(caps)
			row.Status = status
			row.LastSeenAt = &now
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		// 本次未上报的店铺标 inactive（保留历史）
		var all []model.AgentShop
		if err := tx.Where("agent_id = ?", agent.ID).Find(&all).Error; err != nil {
			return err
		}
		for i := range all {
			k := all[i].Platform + "|" + all[i].PlatformShopID
			if _, ok := seen[k]; !ok && all[i].Status == model.ShopStatusActive {
				_ = tx.Model(&all[i]).Update("status", model.ShopStatusInactive).Error
			}
		}
		return nil
	})
}

func (s *AgentService) ClaimJobs(agent *model.Agent, limit int) ([]dto.JobClaimResult, error) {
	if limit <= 0 {
		limit = 1
	}
	if limit > 5 {
		limit = 5
	}
	s.refreshOfflineAgents()

	var shops []model.AgentShop
	if err := s.repos.DB.Where("agent_id = ? AND status = ?", agent.ID, model.ShopStatusActive).Find(&shops).Error; err != nil {
		return nil, err
	}
	if len(shops) == 0 {
		return []dto.JobClaimResult{}, nil
	}

	skills := parseSkills(agent.SkillsJSON)
	skillSet := map[string]struct{}{}
	for _, sk := range skills {
		skillSet[sk] = struct{}{}
	}

	var out []dto.JobClaimResult
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		for _, shop := range shops {
			if len(out) >= limit {
				break
			}
			var jobs []model.AgentJob
			q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("tenant_id = ? AND status = ? AND platform = ? AND platform_shop_id = ?",
					agent.TenantID, model.JobStatusPending, shop.Platform, shop.PlatformShopID).
				Order("priority ASC, id ASC").
				Limit(limit - len(out))
			if err := q.Find(&jobs).Error; err != nil {
				return err
			}
			now := time.Now()
			for i := range jobs {
				job := &jobs[i]
				if len(skillSet) > 0 {
					if _, ok := skillSet[job.JobType]; !ok {
						continue
					}
				}
				aid := agent.ID
				job.Status = model.JobStatusClaimed
				job.AgentID = &aid
				job.ClaimedAt = &now
				if err := tx.Save(job).Error; err != nil {
					return err
				}
				out = append(out, dto.JobClaimResult{
					ID:               job.ID,
					JobType:          job.JobType,
					Platform:         job.Platform,
					PlatformShopID:   job.PlatformShopID,
					PlatformShopName: job.PlatformShopName,
					ParamsJSON:       job.ParamsJSON,
					Source:           job.Source,
					Priority:         job.Priority,
				})
			}
		}
		return nil
	})
	return out, err
}

func (s *AgentService) ReportJob(agent *model.Agent, jobID uint64, in *dto.JobReportInput) error {
	var job model.AgentJob
	if err := s.repos.DB.First(&job, jobID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return err
	}
	if job.AgentID == nil || *job.AgentID != agent.ID {
		return fmt.Errorf("%w: 任务不属于该 Agent", ErrJobState)
	}
	status := strings.TrimSpace(in.Status)
	now := time.Now()
	switch status {
	case model.JobStatusRunning:
		job.Status = model.JobStatusRunning
		job.StartedAt = &now
	case model.JobStatusSucceeded:
		job.Status = model.JobStatusSucceeded
		job.ResultJSON = in.ResultJSON
		job.ErrorMessage = ""
		job.FinishedAt = &now
	case model.JobStatusFailed:
		job.Status = model.JobStatusFailed
		job.ErrorMessage = strings.TrimSpace(in.ErrorMessage)
		job.ResultJSON = in.ResultJSON
		job.FinishedAt = &now
	default:
		return fmt.Errorf("%w: status 仅支持 running/succeeded/failed", ErrBadRequest)
	}
	return s.repos.DB.Save(&job).Error
}

func (s *AgentService) CreateJob(tenantID, userID uint64, in *dto.CreateJobInput) (*model.AgentJob, error) {
	jobType := strings.TrimSpace(in.JobType)
	platform := strings.TrimSpace(in.Platform)
	shopID := strings.TrimSpace(in.PlatformShopID)
	if jobType == "" || platform == "" || shopID == "" {
		return nil, fmt.Errorf("%w: jobType/platform/platformShopId 必填", ErrBadRequest)
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "manual"
	}
	priority := in.Priority
	if priority <= 0 {
		priority = 100
	}
	job := model.AgentJob{
		TenantID:         tenantID,
		JobType:          jobType,
		Platform:         platform,
		PlatformShopID:   shopID,
		PlatformShopName: strings.TrimSpace(in.PlatformShopName),
		ParamsJSON:       in.ParamsJSON,
		Source:           source,
		Priority:         priority,
		Status:           model.JobStatusPending,
		CreatedBy:        userID,
	}
	if err := s.repos.DB.Create(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *AgentService) ListAgents(tenantID uint64, page, pageSize int) ([]dto.AgentListItem, int64, error) {
	s.refreshOfflineAgents()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	q := s.repos.DB.Model(&model.Agent{}).Where("tenant_id = ?", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Agent
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dto.AgentListItem, 0, len(rows))
	for _, a := range rows {
		var shopCount int64
		_ = s.repos.DB.Model(&model.AgentShop{}).Where("agent_id = ? AND status = ?", a.ID, model.ShopStatusActive).Count(&shopCount).Error
		item := dto.AgentListItem{
			ID:           a.ID,
			MachineID:    a.MachineID,
			Name:         a.Name,
			Hostname:     a.Hostname,
			OS:           a.OS,
			AgentVersion: a.AgentVersion,
			Status:       a.Status,
			SkillsJSON:   a.SkillsJSON,
			ShopCount:    shopCount,
			CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		}
		if a.LastHeartbeat != nil {
			t := a.LastHeartbeat.Format(time.RFC3339)
			item.LastHeartbeat = &t
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *AgentService) ListShops(tenantID uint64, page, pageSize int, platform string) ([]dto.ShopListItem, int64, error) {
	s.refreshOfflineAgents()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := s.repos.DB.Model(&model.AgentShop{}).Where("tenant_id = ?", tenantID)
	if p := strings.TrimSpace(platform); p != "" {
		q = q.Where("platform = ?", p)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AgentShop
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dto.ShopListItem, 0, len(rows))
	for _, sh := range rows {
		var agent model.Agent
		_ = s.repos.DB.Select("id, name, status").First(&agent, sh.AgentID).Error
		item := dto.ShopListItem{
			ID:               sh.ID,
			AgentID:          sh.AgentID,
			AgentName:        agent.Name,
			Platform:         sh.Platform,
			PlatformShopID:   sh.PlatformShopID,
			PlatformShopName: sh.PlatformShopName,
			BrowserChannel:   sh.BrowserChannel,
			CapabilitiesJSON: sh.CapabilitiesJSON,
			Status:           sh.Status,
			AgentOnline:      agent.Status == model.AgentStatusOnline,
		}
		if sh.LastSeenAt != nil {
			t := sh.LastSeenAt.Format(time.RFC3339)
			item.LastSeenAt = &t
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *AgentService) ListJobs(tenantID uint64, page, pageSize int, status, jobType string) ([]dto.JobListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := s.repos.DB.Model(&model.AgentJob{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if jobType != "" {
		q = q.Where("job_type = ?", jobType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AgentJob
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dto.JobListItem, 0, len(rows))
	for _, j := range rows {
		item := dto.JobListItem{
			ID:               j.ID,
			JobType:          j.JobType,
			Platform:         j.Platform,
			PlatformShopID:   j.PlatformShopID,
			PlatformShopName: j.PlatformShopName,
			ParamsJSON:       j.ParamsJSON,
			Source:           j.Source,
			Priority:         j.Priority,
			Status:           j.Status,
			AgentID:          j.AgentID,
			ErrorMessage:     j.ErrorMessage,
			CreatedAt:        j.CreatedAt.Format(time.RFC3339),
		}
		if j.AgentID != nil {
			var agent model.Agent
			if s.repos.DB.Select("name").First(&agent, *j.AgentID).Error == nil {
				item.AgentName = agent.Name
			}
		}
		if j.FinishedAt != nil {
			t := j.FinishedAt.Format(time.RFC3339)
			item.FinishedAt = &t
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *AgentService) SkillCatalog() []dto.SkillCatalogItem {
	return []dto.SkillCatalogItem{
		{ID: model.JobTypeDoudianAftersale, Name: "抖店售后单抓取", Platform: model.PlatformDoudian, Description: "抓取抖店售后工作台数据并回传售后中心"},
		{ID: model.JobTypeDoudianDecryptPhone, Name: "抖店订单解密真实手机号", Platform: model.PlatformDoudian, Description: "在已登录抖店后台申请查看真实收件手机号"},
		{ID: model.JobTypeKdzsRemotePrint, Name: "快递助手远程打单", Platform: model.PlatformDoudian, Description: "快递助手桌面端远程打单（Shipping 下发）"},
	}
}

func (s *AgentService) refreshOfflineAgents() {
	cutoff := time.Now().Add(-onlineWithin)
	_ = s.repos.DB.Model(&model.Agent{}).
		Where("status = ? AND (last_heartbeat IS NULL OR last_heartbeat < ?)", model.AgentStatusOnline, cutoff).
		Update("status", model.AgentStatusOffline).Error
}

func normalizeSkills(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return []string{
			model.JobTypeDoudianAftersale,
			model.JobTypeDoudianDecryptPhone,
			model.JobTypeKdzsRemotePrint,
		}
	}
	return out
}

func parseSkills(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
