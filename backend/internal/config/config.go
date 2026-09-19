package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 汇总服务运行所需的全部配置, 全部来自环境变量(可选 .env 文件)。
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
}

// AppConfig 应用级配置。
type AppConfig struct {
	Name string
	Env  string
	Seed bool
}

// IsProduction 用于控制日志与错误细节的输出方式。
func (a AppConfig) IsProduction() bool {
	return strings.EqualFold(a.Env, "production")
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CORSOrigins  []string
}

// Addr 返回监听地址, 例如 0.0.0.0:8080。
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig 数据库配置, driver 支持 sqlite 与 postgres。
type DatabaseConfig struct {
	Driver   string
	DSN      string
	LogLevel string
}

// Load 读取环境变量并组装配置, 未设置的项使用默认值。
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "streetlight-fault-system"),
			Env:  getEnv("APP_ENV", "development"),
			Seed: getEnvBool("APP_SEED", true),
		},
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			CORSOrigins:  getEnvSlice("CORS_ALLOW_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Driver:   strings.ToLower(getEnv("DB_DRIVER", "sqlite")),
			DSN:      getEnv("DB_DSN", "data/streetlight.db"),
			LogLevel: getEnv("DB_LOG_LEVEL", "warn"),
		},
	}

	if cfg.Database.DSN == "" {
		cfg.Database.DSN = getEnv("DATABASE_URL", "")
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	switch c.Database.Driver {
	case "sqlite", "postgres":
	default:
		return fmt.Errorf("不支持的数据库驱动 DB_DRIVER=%s (可选: sqlite, postgres)", c.Database.Driver)
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		return fmt.Errorf("数据库连接串 DB_DSN 不能为空")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务端口 SERVER_PORT=%d 不合法", c.Server.Port)
	}
	return nil
}

// loadDotEnv 尽力加载 .env, 文件不存在时静默忽略, 真实环境变量优先级更高。
func loadDotEnv() {
	for _, file := range []string{".env", "backend/.env"} {
		if _, err := os.Stat(file); err == nil {
			_ = godotenv.Load(file)
			return
		}
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvSlice(key string, fallback []string) []string {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
