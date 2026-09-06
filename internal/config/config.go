package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	CORS       CORSConfig
	AfterSales AfterSalesConfig `mapstructure:"aftersales"`
	Jobs       JobsConfig       `mapstructure:"jobs"`
}

type ServerConfig struct {
	Port int
	Mode string
}

type DatabaseConfig struct {
	Driver      string
	SQLitePath  string
	PostgresDSN string `mapstructure:"postgres_dsn"`
}

type AuthConfig struct {
	Enabled       bool
	JWTSecret     string `mapstructure:"jwt_secret"`
	InternalToken string `mapstructure:"internal_token"`
}

type AfterSalesConfig struct {
	BaseURL       string `mapstructure:"base_url"`
	InternalToken string `mapstructure:"internal_token"`
}

type JobsConfig struct {
	// RetentionDays 执行记录保留天数；默认 3。进行中的 pending/claimed/running 不删。
	RetentionDays int `mapstructure:"retention_days"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8107
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.PostgresDSN == "" {
		cfg.Database.PostgresDSN = "host=127.0.0.1 user=agentscenter password=agentscenter dbname=agentscenter port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = "./data/agentscenter.db"
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "change-me-in-production-use-long-random-string"
	}
	if cfg.Auth.InternalToken == "" {
		cfg.Auth.InternalToken = cfg.Auth.JWTSecret
	}
	if cfg.AfterSales.BaseURL == "" {
		cfg.AfterSales.BaseURL = "http://localhost:5176"
	}
	if cfg.AfterSales.InternalToken == "" {
		cfg.AfterSales.InternalToken = cfg.Auth.InternalToken
	}
	if cfg.Jobs.RetentionDays <= 0 {
		cfg.Jobs.RetentionDays = 3
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{
			"http://localhost:5192",
			"http://127.0.0.1:5192",
			"http://localhost:5174",
			"http://127.0.0.1:5174",
		}
	}
	return &cfg, nil
}
