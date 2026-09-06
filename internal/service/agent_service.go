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
	repos      *repo.Repository
	aftersales *AfterSalesClient
}

func NewAgentService(repos *repo.Repository, aftersales *AfterSalesClient) *AgentService {
	return &AgentService{repos: repos, aftersales: aftersales}
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
				if job.JobType == model.JobTypeDoudianAftersale && !paramsHavePluginCreds(job.ParamsJSON) {
					if s.aftersales == nil {
						continue
					}
					cred, ferr := s.aftersales.FetchCredential(agent.TenantID, job.Platform, job.PlatformShopID)
					if ferr != nil {
						continue
					}
					merged, merr := mergeAftersaleParams(job.ParamsJSON, cred)
					if merr != nil {
						continue
					}
					job.ParamsJSON = merged
					if job.PlatformShopName == "" {
						job.PlatformShopName = cred.PlatformShopName
						if job.PlatformShopName == "" {
							job.PlatformShopName = cred.ShopName
						}
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

	paramsJSON := strings.TrimSpace(in.ParamsJSON)
	shopName := strings.TrimSpace(in.PlatformShopName)
	if jobType == model.JobTypeDoudianAftersale {
		existing, err := s.findPendingJob(tenantID, jobType, platform, shopID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if err := s.refreshAftersaleJobParams(existing, tenantID, platform, shopID, &shopName, paramsJSON); err != nil {
				return nil, err
			}
			return existing, nil
		}
		enriched, name, err := s.enrichAftersaleParams(tenantID, platform, shopID, shopName, paramsJSON)
		if err != nil {
			return nil, err
		}
		paramsJSON = enriched
		shopName = name
	}

	job := model.AgentJob{
		TenantID:         tenantID,
		JobType:          jobType,
		Platform:         platform,
		PlatformShopID:   shopID,
		PlatformShopName: shopName,
		ParamsJSON:       paramsJSON,
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

// findPendingJob 仅复用尚未被领取的执行单，便于参数更新后重新下发。
func (s *AgentService) findPendingJob(tenantID uint64, jobType, platform, shopID string) (*model.AgentJob, error) {
	var job model.AgentJob
	err := s.repos.DB.Where(
		"tenant_id = ? AND job_type = ? AND platform = ? AND platform_shop_id = ? AND status = ?",
		tenantID, jobType, platform, shopID, model.JobStatusPending,
	).Order("id desc").First(&job).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *AgentService) enrichAftersaleParams(tenantID uint64, platform, shopID, shopName, paramsJSON string) (string, string, error) {
	base := strings.TrimSpace(paramsJSON)
	// 业务 app 创建任务时已带上报地址与凭证：直接采用，中心不做覆盖
	if paramsHavePluginCreds(base) {
		return base, shopName, nil
	}
	if s.aftersales == nil {
		return "", shopName, fmt.Errorf("%w: 售后任务缺少采集参数（apiBase/pluginKey/pluginSecret）且无法回源售后", ErrBadRequest)
	}
	cred, err := s.aftersales.FetchCredential(tenantID, platform, shopID)
	if err != nil {
		return "", shopName, fmt.Errorf("%w: 拉取售后采集凭证失败: %v", ErrBadRequest, err)
	}
	if shopName == "" {
		shopName = cred.PlatformShopName
		if shopName == "" {
			shopName = cred.ShopName
		}
	}
	merged, err := mergeAftersaleParams(base, cred)
	if err != nil {
		return "", shopName, err
	}
	return merged, shopName, nil
}

// refreshAftersaleJobParams 更新 pending 执行单参数：优先使用业务 app 本次下发的 params。
func (s *AgentService) refreshAftersaleJobParams(job *model.AgentJob, tenantID uint64, platform, shopID string, shopName *string, incomingParams string) error {
	name := ""
	if shopName != nil {
		name = *shopName
	}
	if name == "" {
		name = job.PlatformShopName
	}
	incoming := strings.TrimSpace(incomingParams)
	if paramsHavePluginCreds(incoming) {
		job.ParamsJSON = incoming
		if name != "" {
			job.PlatformShopName = name
			if shopName != nil {
				*shopName = name
			}
		}
		return s.repos.DB.Save(job).Error
	}
	enriched, name, err := s.enrichAftersaleParams(tenantID, platform, shopID, name, incoming)
	if err != nil {
		return err
	}
	job.ParamsJSON = enriched
	if name != "" {
		job.PlatformShopName = name
		if shopName != nil {
			*shopName = name
		}
	}
	return s.repos.DB.Save(job).Error
}

func mergeAftersaleParams(existing string, cred *AfterSalesCredential) (string, error) {
	m := map[string]any{}
	if strings.TrimSpace(existing) != "" {
		if err := json.Unmarshal([]byte(existing), &m); err != nil {
			return "", fmt.Errorf("%w: paramsJson 非法", ErrBadRequest)
		}
	}
	m["apiBase"] = cred.APIBase
	m["shopId"] = cred.ShopID
	m["shopName"] = cred.ShopName
	m["platform"] = cred.Platform
	m["pluginKey"] = cred.PluginKey
	m["pluginSecret"] = cred.PluginSecret
	m["platformShopId"] = cred.PlatformShopID
	m["platformShopName"] = cred.PlatformShopName
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func paramsHavePluginCreds(paramsJSON string) bool {
	if strings.TrimSpace(paramsJSON) == "" {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(paramsJSON), &m); err != nil {
		return false
	}
	key, _ := m["pluginKey"].(string)
	secret, _ := m["pluginSecret"].(string)
	return strings.TrimSpace(key) != "" && strings.TrimSpace(secret) != ""
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
		var shops []model.AgentShop
		_ = s.repos.DB.Where("agent_id = ? AND status = ?", a.ID, model.ShopStatusActive).
			Order("id asc").Find(&shops).Error
		briefs := make([]dto.AgentShopBrief, 0, len(shops))
		for _, sh := range shops {
			briefs = append(briefs, dto.AgentShopBrief{
				Platform:         sh.Platform,
				PlatformShopID:   sh.PlatformShopID,
				PlatformShopName: sh.PlatformShopName,
				BrowserChannel:   sh.BrowserChannel,
				Status:           sh.Status,
			})
		}
		item := dto.AgentListItem{
			ID:           a.ID,
			MachineID:    a.MachineID,
			Name:         a.Name,
			Hostname:     a.Hostname,
			OS:           a.OS,
			AgentVersion: a.AgentVersion,
			Status:       a.Status,
			SkillsJSON:   a.SkillsJSON,
			ShopCount:    int64(len(briefs)),
			Shops:        briefs,
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

func (s *AgentService) ListShops(tenantID uint64, page, pageSize int, platform string, onlineOnly bool) ([]dto.ShopListItem, int64, error) {
	s.refreshOfflineAgents()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := s.repos.DB.Model(&model.AgentShop{}).Where("agent_shops.tenant_id = ? AND agent_shops.status = ?", tenantID, model.ShopStatusActive)
	if p := strings.TrimSpace(platform); p != "" {
		q = q.Where("agent_shops.platform = ?", p)
	}
	if onlineOnly {
		q = q.Joins("JOIN agents ON agents.id = agent_shops.agent_id").
			Where("agents.status = ?", model.AgentStatusOnline)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AgentShop
	if err := q.Order("agent_shops.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
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
		{
			ID:                     model.JobTypeDoudianAftersale,
			Name:                   "抖店售后单抓取",
			Platform:               model.PlatformDoudian,
            Description:            "抓取抖店售后工作台；上报地址与凭证由售后中心创建任务时写入 params",
			RunPolicies:            []string{model.RunPolicyInterval, model.RunPolicyOnDemand},
			DefaultIntervalMinutes: 30,
		},
		{
			ID:          model.JobTypeDoudianDecryptPhone,
			Name:        "抖店订单解密真实手机号",
			Platform:    model.PlatformDoudian,
			Description: "在已登录抖店后台申请查看真实收件手机号",
			RunPolicies: []string{model.RunPolicyOnDemand},
		},
		{
			ID:          model.JobTypeKdzsRemotePrint,
			Name:        "快递助手远程打单",
			Platform:    model.PlatformDoudian,
			Description: "快递助手桌面端远程打单（Shipping 下发）",
			RunPolicies: []string{model.RunPolicyOnDemand},
		},
	}
}

func (s *AgentService) skillByID(id string) *dto.SkillCatalogItem {
	for _, sk := range s.SkillCatalog() {
		if sk.ID == id {
			cp := sk
			return &cp
		}
	}
	return nil
}

func skillSupports(sk *dto.SkillCatalogItem, policy string) bool {
	if sk == nil {
		return false
	}
	for _, p := range sk.RunPolicies {
		if p == policy {
			return true
		}
	}
	return false
}

func (s *AgentService) ListAssignments(tenantID uint64, page, pageSize int, jobType string) ([]dto.AssignmentListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := s.repos.DB.Model(&model.TaskAssignment{}).Where("tenant_id = ?", tenantID)
	if jt := strings.TrimSpace(jobType); jt != "" {
		q = q.Where("job_type = ?", jt)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.TaskAssignment
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	nameByID := map[string]string{}
	for _, sk := range s.SkillCatalog() {
		nameByID[sk.ID] = sk.Name
	}
	out := make([]dto.AssignmentListItem, 0, len(rows))
	for _, a := range rows {
		item := dto.AssignmentListItem{
			ID:               a.ID,
			JobType:          a.JobType,
			JobTypeName:      nameByID[a.JobType],
			Platform:         a.Platform,
			PlatformShopID:   a.PlatformShopID,
			PlatformShopName: a.PlatformShopName,
			Enabled:          a.Enabled,
			RunPolicy:        a.RunPolicy,
			IntervalMinutes:  a.IntervalMinutes,
			CreatedAt:        a.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        a.UpdatedAt.Format(time.RFC3339),
		}
		if a.LastEnqueuedAt != nil {
			t := a.LastEnqueuedAt.Format(time.RFC3339)
			item.LastEnqueuedAt = &t
		}
		if a.NextRunAt != nil {
			t := a.NextRunAt.Format(time.RFC3339)
			item.NextRunAt = &t
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *AgentService) UpsertAssignment(tenantID, userID uint64, in *dto.UpsertAssignmentInput) (*model.TaskAssignment, *model.AgentJob, error) {
	jobType := strings.TrimSpace(in.JobType)
	platform := strings.TrimSpace(in.Platform)
	shopID := strings.TrimSpace(in.PlatformShopID)
	if jobType == "" || platform == "" || shopID == "" {
		return nil, nil, fmt.Errorf("%w: jobType/platform/platformShopId 必填", ErrBadRequest)
	}
	sk := s.skillByID(jobType)
	if sk == nil {
		return nil, nil, fmt.Errorf("%w: 未知任务类型 %s", ErrBadRequest, jobType)
	}

	runPolicy := strings.TrimSpace(in.RunPolicy)
	if runPolicy == "" {
		if skillSupports(sk, model.RunPolicyInterval) {
			runPolicy = model.RunPolicyInterval
		} else {
			runPolicy = model.RunPolicyOnDemand
		}
	}
	if !skillSupports(sk, runPolicy) {
		return nil, nil, fmt.Errorf("%w: 技能 %s 不支持运行策略 %s", ErrBadRequest, jobType, runPolicy)
	}

	var interval *int
	if runPolicy == model.RunPolicyInterval {
		mins := sk.DefaultIntervalMinutes
		if mins <= 0 {
			mins = 30
		}
		if in.IntervalMinutes != nil && *in.IntervalMinutes > 0 {
			mins = *in.IntervalMinutes
		}
		interval = &mins
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	var row model.TaskAssignment
	err := s.repos.DB.Where(
		"tenant_id = ? AND job_type = ? AND platform = ? AND platform_shop_id = ?",
		tenantID, jobType, platform, shopID,
	).First(&row).Error
	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		row = model.TaskAssignment{
			TenantID:         tenantID,
			JobType:          jobType,
			Platform:         platform,
			PlatformShopID:   shopID,
			PlatformShopName: strings.TrimSpace(in.PlatformShopName),
			Enabled:          enabled,
			RunPolicy:        runPolicy,
			IntervalMinutes:  interval,
			CreatedBy:        userID,
		}
		if enabled && runPolicy == model.RunPolicyInterval {
			// 若立即触发，调度器从下一次间隔开始；否则尽快由调度器领取
			if in.TriggerNow {
				next := now.Add(time.Duration(*interval) * time.Minute)
				row.NextRunAt = &next
			} else {
				row.NextRunAt = &now
			}
		}
		if err := s.repos.DB.Create(&row).Error; err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		prevInterval := 0
		if row.IntervalMinutes != nil {
			prevInterval = *row.IntervalMinutes
		}
		row.Enabled = enabled
		row.RunPolicy = runPolicy
		row.IntervalMinutes = interval
		if v := strings.TrimSpace(in.PlatformShopName); v != "" {
			row.PlatformShopName = v
		}
		if enabled && runPolicy == model.RunPolicyInterval && interval != nil {
			// 间隔或参数变更时重算下次执行时间
			intervalChanged := in.IntervalMinutes != nil && *interval != prevInterval
			if row.NextRunAt == nil || intervalChanged || strings.TrimSpace(in.ParamsJSON) != "" {
				next := now.Add(time.Duration(*interval) * time.Minute)
				if !intervalChanged && row.LastEnqueuedAt != nil {
					cand := row.LastEnqueuedAt.Add(time.Duration(*interval) * time.Minute)
					if cand.After(now) {
						next = cand
					} else {
						next = now
					}
				}
				if in.TriggerNow {
					next = now.Add(time.Duration(*interval) * time.Minute)
				}
				row.NextRunAt = &next
			}
		} else {
			row.NextRunAt = nil
		}
		if err := s.repos.DB.Save(&row).Error; err != nil {
			return nil, nil, err
		}
	}

	var job *model.AgentJob
	if in.TriggerNow {
		var err error
		job, err = s.triggerAssignmentLocked(&row, userID, in.ParamsJSON, in.Source, in.Priority)
		if err != nil {
			return &row, nil, err
		}
	} else if jobType == model.JobTypeDoudianAftersale && strings.TrimSpace(in.ParamsJSON) != "" {
		// 参数变更：同步刷新尚未领取的执行单，下一次领取即用新参数
		if pending, err := s.findPendingJob(tenantID, jobType, platform, shopID); err == nil && pending != nil {
			name := strings.TrimSpace(in.PlatformShopName)
			_ = s.refreshAftersaleJobParams(pending, tenantID, platform, shopID, &name, in.ParamsJSON)
		}
	}
	return &row, job, nil
}

func (s *AgentService) TriggerAssignment(tenantID, userID, assignmentID uint64, in *dto.TriggerAssignmentInput) (*model.AgentJob, error) {
	var row model.TaskAssignment
	if err := s.repos.DB.Where("id = ? AND tenant_id = ?", assignmentID, tenantID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !row.Enabled {
		return nil, fmt.Errorf("%w: 订阅已停用", ErrBadRequest)
	}
	params := ""
	source := "manual"
	priority := 100
	if in != nil {
		params = in.ParamsJSON
		if strings.TrimSpace(in.Source) != "" {
			source = in.Source
		}
		if in.Priority > 0 {
			priority = in.Priority
		}
	}
	return s.triggerAssignmentLocked(&row, userID, params, source, priority)
}

func (s *AgentService) SetAssignmentEnabled(tenantID, assignmentID uint64, enabled bool) (*model.TaskAssignment, error) {
	var row model.TaskAssignment
	if err := s.repos.DB.Where("id = ? AND tenant_id = ?", assignmentID, tenantID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	row.Enabled = enabled
	if enabled && row.RunPolicy == model.RunPolicyInterval && row.IntervalMinutes != nil && *row.IntervalMinutes > 0 {
		now := time.Now()
		if row.NextRunAt == nil || row.NextRunAt.Before(now) {
			row.NextRunAt = &now
		}
	}
	if err := s.repos.DB.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AgentService) triggerAssignmentLocked(row *model.TaskAssignment, userID uint64, paramsJSON, source string, priority int) (*model.AgentJob, error) {
	if source == "" {
		source = "manual"
	}
	job, err := s.CreateJob(row.TenantID, userID, &dto.CreateJobInput{
		JobType:          row.JobType,
		Platform:         row.Platform,
		PlatformShopID:   row.PlatformShopID,
		PlatformShopName: row.PlatformShopName,
		ParamsJSON:       paramsJSON,
		Source:           source,
		Priority:         priority,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	row.LastEnqueuedAt = &now
	if row.RunPolicy == model.RunPolicyInterval && row.IntervalMinutes != nil && *row.IntervalMinutes > 0 {
		next := now.Add(time.Duration(*row.IntervalMinutes) * time.Minute)
		row.NextRunAt = &next
	}
	_ = s.repos.DB.Save(row).Error
	return job, nil
}

// DispatchDueAssignments 已废弃：定时间隔由业务 app（如售后）触发下发。
// 保留空实现以免旧调用方编译失败。
func (s *AgentService) DispatchDueAssignments() (int, error) {
	return 0, nil
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
