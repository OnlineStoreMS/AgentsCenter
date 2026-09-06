package model

import "time"

// Agent 一台 WindowsAgent 电脑（执行节点）。
type Agent struct {
	ID            uint64     `gorm:"primaryKey" json:"id"`
	TenantID      uint64     `gorm:"index;not null;default:1" json:"tenantId"`
	MachineID     string     `gorm:"size:128;uniqueIndex;not null" json:"machineId"`
	Name          string     `gorm:"size:128;not null" json:"name"`
	Hostname      string     `gorm:"size:128" json:"hostname"`
	OS            string     `gorm:"size:64" json:"os"`
	AgentVersion  string     `gorm:"size:32" json:"agentVersion"`
	AgentKey      string     `gorm:"size:64;uniqueIndex;not null" json:"agentKey"`
	AgentSecret   string     `gorm:"size:128;not null" json:"-"`
	Status        string     `gorm:"size:16;index;not null;default:offline" json:"status"` // online/offline
	SkillsJSON    string     `gorm:"type:text" json:"skillsJson"`                         // 本机支持的 skill id 列表 JSON
	LastHeartbeat *time.Time `json:"lastHeartbeat"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (Agent) TableName() string { return "agents" }

// AgentShop Agent 上报/录入的已登录电商店铺会话。
type AgentShop struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	TenantID         uint64     `gorm:"index;not null;default:1" json:"tenantId"`
	AgentID          uint64     `gorm:"index;not null" json:"agentId"`
	Platform         string     `gorm:"size:32;index;not null" json:"platform"` // doudian/taobao/pdd/...
	PlatformShopID   string     `gorm:"size:128;index;not null" json:"platformShopId"`
	PlatformShopName string     `gorm:"size:256" json:"platformShopName"`
	BrowserChannel   string     `gorm:"size:32" json:"browserChannel"` // msedge/chrome
	CapabilitiesJSON string     `gorm:"type:text" json:"capabilitiesJson"`
	Status           string     `gorm:"size:16;not null;default:active" json:"status"` // active/inactive
	LastSeenAt       *time.Time `json:"lastSeenAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (AgentShop) TableName() string { return "agent_shops" }

// AgentJob 业务中心提交的任务；中心按店铺匹配在线 Agent 后下发。
type AgentJob struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	TenantID         uint64     `gorm:"index;not null;default:1" json:"tenantId"`
	JobType          string     `gorm:"size:64;index;not null" json:"jobType"` // skill id
	Platform         string     `gorm:"size:32;index;not null" json:"platform"`
	PlatformShopID   string     `gorm:"size:128;index;not null" json:"platformShopId"`
	PlatformShopName string     `gorm:"size:256" json:"platformShopName"`
	ParamsJSON       string     `gorm:"type:text" json:"paramsJson"`
	Source           string     `gorm:"size:64" json:"source"` // aftersales/order/product/manual
	Priority         int        `gorm:"not null;default:100" json:"priority"`
	Status           string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	AgentID          *uint64    `gorm:"index" json:"agentId"`
	ClaimedAt        *time.Time `json:"claimedAt"`
	StartedAt        *time.Time `json:"startedAt"`
	FinishedAt       *time.Time `json:"finishedAt"`
	ResultJSON       string     `gorm:"type:text" json:"resultJson"`
	ErrorMessage     string     `gorm:"type:text" json:"errorMessage"`
	CreatedBy        uint64     `json:"createdBy"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (AgentJob) TableName() string { return "agent_jobs" }

const (
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"

	ShopStatusActive   = "active"
	ShopStatusInactive = "inactive"

	JobStatusPending   = "pending"
	JobStatusClaimed   = "claimed"
	JobStatusRunning   = "running"
	JobStatusSucceeded = "succeeded"
	JobStatusFailed    = "failed"
	JobStatusCancelled = "cancelled"

	PlatformDoudian = "doudian"
	PlatformTaobao  = "taobao"
	PlatformPdd     = "pdd"

	JobTypeDoudianAftersale     = "doudian.aftersale"
	JobTypeDoudianDecryptPhone  = "doudian.order.decrypt-phone"
	JobTypeKdzsRemotePrint      = "kdzs.remote.print"
)
