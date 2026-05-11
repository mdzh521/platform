package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppName             string               `yaml:"app_name"`
	Port                string               `yaml:"port"`
	RunMode             string               `yaml:"run_mode"`
	JWTSecret           string               `yaml:"jwt_secret"`
	InternalRunnerToken string               `yaml:"internal_runner_token"`
	DefaultAdmin        string               `yaml:"default_admin_username"`
	DefaultPassword     string               `yaml:"default_admin_password"`
	MySQLHost           string               `yaml:"mysql_host"`
	MySQLPort           string               `yaml:"mysql_port"`
	MySQLDatabase       string               `yaml:"mysql_database"`
	MySQLUser           string               `yaml:"mysql_user"`
	MySQLPassword       string               `yaml:"mysql_password"`
	TrustedOrigins      []string             `yaml:"trusted_origins"`
	CloudAccounts       []CloudAccountConfig `yaml:"cloud_accounts"`
	Frontend            FrontendConfig       `yaml:"frontend"`
	Cache               CacheConfig          `yaml:"cache"`
	Queue               QueueConfig          `yaml:"queue"`
	ConfigFile          string               `yaml:"-"`
}

type CloudAccountConfig struct {
	Name         string   `yaml:"name"`
	Provider     string   `yaml:"provider"`
	AccessKey    string   `yaml:"access_key"`
	SecretKey    string   `yaml:"secret_key"`
	Region       string   `yaml:"region"`
	RoleARN      string   `yaml:"role_arn"`
	DefaultTags  []string `yaml:"default_tags"`
	DefaultZones []string `yaml:"default_zones"`
}

type FrontendConfig struct {
	Mode       string `json:"mode" yaml:"mode"`
	APIBaseURL string `json:"api_base_url" yaml:"api_base_url"`
}

type CacheConfig struct {
	Driver   string `yaml:"driver"`
	RedisURL string `yaml:"redis_url"`
}

type QueueConfig struct {
	Driver   string `yaml:"driver"`
	RedisURL string `yaml:"redis_url"`
}

type RuntimeConfig struct {
	AppName  string         `json:"app_name"`
	RunMode  string         `json:"run_mode"`
	Frontend FrontendConfig `json:"frontend"`
	Cache    RuntimeDriver  `json:"cache"`
	Queue    RuntimeDriver  `json:"queue"`
}

type RuntimeDriver struct {
	Driver  string `json:"driver"`
	Enabled bool   `json:"enabled"`
}

func Load() Config {
	cfg := defaultConfig()
	if path := resolveConfigFile(); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Errorf("read config file %s: %w", path, err))
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			panic(fmt.Errorf("parse config file %s: %w", path, err))
		}
		cfg.ConfigFile = path
	}
	applyEnvOverrides(&cfg)
	validateSecurityConfig(cfg)
	return cfg
}

func defaultConfig() Config {
	return Config{
		AppName:             "backend-center",
		Port:                "8080",
		RunMode:             "single",
		JWTSecret:           "change-me",
		InternalRunnerToken: "runner-dev-token",
		DefaultAdmin:        "admin",
		DefaultPassword:     "admin123456",
		MySQLHost:           "mysql",
		MySQLPort:           "3306",
		MySQLDatabase:       "backend_center",
		MySQLUser:           "backend",
		MySQLPassword:       "backend123",
		TrustedOrigins:      []string{},
		CloudAccounts:       []CloudAccountConfig{},
		Frontend: FrontendConfig{
			Mode:       "embedded",
			APIBaseURL: "",
		},
		Cache: CacheConfig{
			Driver:   "memory",
			RedisURL: "",
		},
		Queue: QueueConfig{
			Driver:   "memory",
			RedisURL: "",
		},
	}
}

func applyEnvOverrides(cfg *Config) {
	cfg.AppName = getenv("APP_NAME", cfg.AppName)
	cfg.Port = getenv("APP_PORT", cfg.Port)
	cfg.RunMode = getenv("RUN_MODE", cfg.RunMode)
	cfg.JWTSecret = getenv("JWT_SECRET", cfg.JWTSecret)
	cfg.InternalRunnerToken = getenv("INTERNAL_RUNNER_TOKEN", cfg.InternalRunnerToken)
	cfg.DefaultAdmin = getenv("DEFAULT_ADMIN_USERNAME", cfg.DefaultAdmin)
	cfg.DefaultPassword = getenv("DEFAULT_ADMIN_PASSWORD", cfg.DefaultPassword)
	cfg.MySQLHost = getenv("MYSQL_HOST", cfg.MySQLHost)
	cfg.MySQLPort = getenv("MYSQL_PORT", cfg.MySQLPort)
	cfg.MySQLDatabase = getenv("MYSQL_DATABASE", cfg.MySQLDatabase)
	cfg.MySQLUser = getenv("MYSQL_USER", cfg.MySQLUser)
	cfg.MySQLPassword = getenv("MYSQL_PASSWORD", cfg.MySQLPassword)
	cfg.TrustedOrigins = getenvList("TRUSTED_ORIGINS", cfg.TrustedOrigins)
	cfg.Frontend.Mode = getenv("FRONTEND_MODE", cfg.Frontend.Mode)
	cfg.Frontend.APIBaseURL = getenv("FRONTEND_API_BASE_URL", cfg.Frontend.APIBaseURL)
	cfg.Cache.Driver = getenv("CACHE_DRIVER", cfg.Cache.Driver)
	cfg.Cache.RedisURL = getenv("CACHE_REDIS_URL", cfg.Cache.RedisURL)
	cfg.Queue.Driver = getenv("QUEUE_DRIVER", cfg.Queue.Driver)
	cfg.Queue.RedisURL = getenv("QUEUE_REDIS_URL", cfg.Queue.RedisURL)
}

func validateSecurityConfig(cfg Config) {
	if cfg.RunMode == "single" {
		return
	}
	if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me" {
		panic("JWT_SECRET must be set to a strong external value outside single mode")
	}
	if cfg.DefaultPassword == "" || cfg.DefaultPassword == "admin123456" {
		panic("DEFAULT_ADMIN_PASSWORD must be set to a strong external value outside single mode")
	}
	if cfg.InternalRunnerToken == "" || cfg.InternalRunnerToken == "runner-dev-token" {
		panic("INTERNAL_RUNNER_TOKEN must be set to a strong external value outside single mode")
	}
}

func resolveConfigFile() string {
	if explicit := os.Getenv("APP_CONFIG_FILE"); explicit != "" {
		return explicit
	}
	candidates := []string{
		"config/platform-center.yaml",
		"platform-center.yaml",
		"config.yaml",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		for _, candidate := range candidates {
			path := filepath.Join(base, candidate)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQLUser,
		c.MySQLPassword,
		c.MySQLHost,
		c.MySQLPort,
		c.MySQLDatabase,
	)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (c Config) Runtime() RuntimeConfig {
	return RuntimeConfig{
		AppName:  c.AppName,
		RunMode:  c.RunMode,
		Frontend: c.Frontend,
		Cache: RuntimeDriver{
			Driver:  c.Cache.Driver,
			Enabled: c.Cache.Driver != "" && c.Cache.Driver != "memory",
		},
		Queue: RuntimeDriver{
			Driver:  c.Queue.Driver,
			Enabled: c.Queue.Driver != "" && c.Queue.Driver != "memory",
		},
	}
}
