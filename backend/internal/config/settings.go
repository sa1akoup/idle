// 应用运行配置：集中读取环境变量，区分本地开发与生产启动行为。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Settings 描述 HTTP 服务、数据库和启动阶段的运行配置。
type Settings struct {
	Environment    string
	HTTPAddr       string
	DatabaseDriver string
	DatabaseURL    string
	TrustedProxies []string
	AllowedOrigins []string
	MigrateOnStart bool
	SeedOnStart    bool
}

// LoadSettings 从环境变量加载配置；开发环境保留当前本地启动体验。
func LoadSettings() (Settings, error) {
	environment := strings.ToLower(getenv("APP_ENV", "development"))
	databaseDriver := strings.ToLower(getenv("DATABASE_DRIVER", "sqlite"))
	databaseURL := getenv("DATABASE_URL", "")
	if databaseDriver == "sqlite" && databaseURL == "" {
		databaseURL = "idle.db"
	}
	if databaseDriver == "postgres" && databaseURL == "" {
		return Settings{}, fmt.Errorf("DATABASE_URL 不能为空（DATABASE_DRIVER=postgres）")
	}
	if databaseDriver != "sqlite" && databaseDriver != "postgres" {
		return Settings{}, fmt.Errorf("不支持的 DATABASE_DRIVER: %s", databaseDriver)
	}

	migrateOnStart, err := getBool("MIGRATE_ON_START", environment != "production")
	if err != nil {
		return Settings{}, err
	}
	seedOnStart, err := getBool("SEED_ON_START", environment != "production")
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		Environment:    environment,
		HTTPAddr:       getenv("HTTP_ADDR", ":8081"),
		DatabaseDriver: databaseDriver,
		DatabaseURL:    databaseURL,
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
		AllowedOrigins: splitList(getenv("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		MigrateOnStart: migrateOnStart,
		SeedOnStart:    seedOnStart,
	}, nil
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("环境变量 %s 必须是布尔值: %w", key, err)
	}
	return parsed, nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
