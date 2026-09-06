package dto

type AgentRegisterInput struct {
	MachineID    string   `json:"machineId" binding:"required"`
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	OS           string   `json:"os"`
	AgentVersion string   `json:"agentVersion"`
	Skills       []string `json:"skills"`
}

type AgentRegisterResult struct {
	AgentID     uint64   `json:"agentId"`
	AgentKey    string   `json:"agentKey"`
	AgentSecret string   `json:"agentSecret"`
	Name        string   `json:"name"`
	Skills      []string `json:"skills"`
}

type AgentShopReport struct {
	Platform         string   `json:"platform" binding:"required"`
	PlatformShopID   string   `json:"platformShopId" binding:"required"`
	PlatformShopName string   `json:"platformShopName"`
	BrowserChannel   string   `json:"browserChannel"`
	Capabilities     []string `json:"capabilities"`
	Status           string   `json:"status"`
}

type AgentHeartbeatInput struct {
	Hostname     string            `json:"hostname"`
	OS           string            `json:"os"`
	AgentVersion string            `json:"agentVersion"`
	Skills       []string          `json:"skills"`
	Shops        []AgentShopReport `json:"shops"`
}

type AgentHeartbeatResult struct {
	ServerTime       string `json:"serverTime"`
	PendingJobs      int64  `json:"pendingJobs"`
	HeartbeatInterval int   `json:"heartbeatIntervalSec"`
}

type JobClaimResult struct {
	ID               uint64 `json:"id"`
	JobType          string `json:"jobType"`
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	ParamsJSON       string `json:"paramsJson"`
	Source           string `json:"source"`
	Priority         int    `json:"priority"`
}

type JobReportInput struct {
	Status       string `json:"status" binding:"required"` // running/succeeded/failed
	ResultJSON   string `json:"resultJson"`
	ErrorMessage string `json:"errorMessage"`
}

type CreateJobInput struct {
	JobType          string `json:"jobType" binding:"required"`
	Platform         string `json:"platform" binding:"required"`
	PlatformShopID   string `json:"platformShopId" binding:"required"`
	PlatformShopName string `json:"platformShopName"`
	ParamsJSON       string `json:"paramsJson"`
	Source           string `json:"source"`
	Priority         int    `json:"priority"`
}

type InternalCreateJobInput struct {
	TenantID         uint64 `json:"tenantId" binding:"required"`
	JobType          string `json:"jobType" binding:"required"`
	Platform         string `json:"platform" binding:"required"`
	PlatformShopID   string `json:"platformShopId" binding:"required"`
	PlatformShopName string `json:"platformShopName"`
	ParamsJSON       string `json:"paramsJson"`
	Source           string `json:"source"`
	Priority         int    `json:"priority"`
}

type AgentListItem struct {
	ID            uint64           `json:"id"`
	MachineID     string           `json:"machineId"`
	Name          string           `json:"name"`
	Hostname      string           `json:"hostname"`
	OS            string           `json:"os"`
	AgentVersion  string           `json:"agentVersion"`
	Status        string           `json:"status"`
	SkillsJSON    string           `json:"skillsJson"`
	ShopCount     int64            `json:"shopCount"`
	Shops         []AgentShopBrief `json:"shops"`
	LastHeartbeat *string          `json:"lastHeartbeat"`
	CreatedAt     string           `json:"createdAt"`
}

type AgentShopBrief struct {
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	BrowserChannel   string `json:"browserChannel"`
	Status           string `json:"status"`
}

type ShopListItem struct {
	ID               uint64  `json:"id"`
	AgentID          uint64  `json:"agentId"`
	AgentName        string  `json:"agentName"`
	Platform         string  `json:"platform"`
	PlatformShopID   string  `json:"platformShopId"`
	PlatformShopName string  `json:"platformShopName"`
	BrowserChannel   string  `json:"browserChannel"`
	CapabilitiesJSON string  `json:"capabilitiesJson"`
	Status           string  `json:"status"`
	LastSeenAt       *string `json:"lastSeenAt"`
	AgentOnline      bool    `json:"agentOnline"`
}

type JobListItem struct {
	ID               uint64  `json:"id"`
	JobType          string  `json:"jobType"`
	Platform         string  `json:"platform"`
	PlatformShopID   string  `json:"platformShopId"`
	PlatformShopName string  `json:"platformShopName"`
	ParamsJSON       string  `json:"paramsJson"`
	Source           string  `json:"source"`
	Priority         int     `json:"priority"`
	Status           string  `json:"status"`
	AgentID          *uint64 `json:"agentId"`
	AgentName        string  `json:"agentName"`
	ErrorMessage     string  `json:"errorMessage"`
	CreatedAt        string  `json:"createdAt"`
	FinishedAt       *string `json:"finishedAt"`
}

type SkillCatalogItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
}
