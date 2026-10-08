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

type ShippingClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type ShippingKdzsLogin struct {
	TenantID    uint64 `json:"tenantId"`
	Mobile      string `json:"mobile"`
	Password    string `json:"password"`
	AccountCode string `json:"accountCode"`
	AccountName string `json:"accountName"`
}

func NewShippingClient(baseURL, token string) *ShippingClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(token) == "" {
		return nil
	}
	return &ShippingClient{
		BaseURL: baseURL,
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *ShippingClient) FetchDefaultKdzsLogin(tenantID uint64) (*ShippingKdzsLogin, error) {
	if c == nil {
		return nil, fmt.Errorf("发货中心未配置")
	}
	if tenantID == 0 {
		return nil, fmt.Errorf("%w: tenantId 无效", ErrBadRequest)
	}
	q := url.Values{}
	q.Set("tenantId", fmt.Sprintf("%d", tenantID))
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/internal/kdzs/default-login?"+q.Encode(), nil)
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
		return nil, fmt.Errorf("Shipping HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    ShippingKdzsLogin `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if strings.TrimSpace(envelope.Data.Mobile) == "" || envelope.Data.Password == "" {
		return nil, fmt.Errorf("发货中心未返回可用的默认快递助手账号")
	}
	return &envelope.Data, nil
}
