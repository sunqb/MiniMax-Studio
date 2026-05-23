package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// MiniMax
	MinimaxAPIKey     string
	MinimaxPaygAPIKey string // 按量付费 Key，优先用于声音复刻等不走 Token Plan 的能力
	MinimaxBaseURL    string

	// Cloudflare R2
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	// 历史记录 SQLite 路径（默认 data/history.db）
	DBPath string

	// Server
	Port           string
	GinMode        string
	SitePassword   string
	ClonedVoiceIDs []string
}

func Load() (*Config, error) {
	cfg := &Config{
		MinimaxAPIKey:     getEnv("MINIMAX_API_KEY", ""),
		MinimaxPaygAPIKey: getEnv("MINIMAX_PAYG_API_KEY", ""),
		MinimaxBaseURL:    getEnv("MINIMAX_BASE_URL", "https://api.minimaxi.com"),
		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		DBPath:            getEnv("DB_PATH", "data/history.db"),
		Port:              getEnv("PORT", "8080"),
		GinMode:           getEnv("GIN_MODE", "debug"),
		SitePassword:      getEnv("SITE_PASSWORD", ""),
		ClonedVoiceIDs:    getCSVEnv("CLONED_VOICE_IDS"),
	}

	// MiniMax 必填校验
	if cfg.MinimaxAPIKey == "" {
		return nil, fmt.Errorf("MINIMAX_API_KEY is required")
	}

	// R2 必填校验
	if cfg.R2AccountID == "" {
		return nil, fmt.Errorf("R2_ACCOUNT_ID is required")
	}
	if cfg.R2AccessKeyID == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID is required")
	}
	if cfg.R2SecretAccessKey == "" {
		return nil, fmt.Errorf("R2_SECRET_ACCESS_KEY is required")
	}
	if cfg.R2BucketName == "" {
		return nil, fmt.Errorf("R2_BUCKET_NAME is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getCSVEnv(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
