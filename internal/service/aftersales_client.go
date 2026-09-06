package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AfterSalesClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type AfterSalesCredential struct {
	TenantID         uint64 `json:"tenantId"`
	ShopID           uint64 `json:"shopId"`
	ShopName         string `json:"shopName"`
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	PluginKey        string `json:"pluginKey"`
	PluginSecret     string `json:"pluginSecret"`
	APIBase          string `json:"apiBase"`
}

func NewAfterSalesClient(baseURL, token string) *AfterSalesClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(token) == "" {
		return nil
	}
	return &AfterSalesClient{
		BaseURL: baseURL,
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *AfterSalesClient) FetchCredential(tenantID uint64, platform, platformShopID string) (*AfterSalesCredential, error) {
	if c == nil {
		return nil, fmt.Errorf("AfterSales 未配置")
	}
	q := url.Values{}
	q.Set("tenantId", fmt.Sprintf("%d", tenantID))
	q.Set("platform", platform)
	q.Set("platformShopId", platformShopID)
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/internal/agent-shops?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Token", c.Token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("AfterSales HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    AfterSalesCredential `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data.PluginKey == "" || envelope.Data.PluginSecret == "" {
		return nil, fmt.Errorf("售后中心未返回采集凭证")
	}
	return &envelope.Data, nil
}
