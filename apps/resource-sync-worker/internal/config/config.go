package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorkerName     string        `yaml:"worker_name"`
	ListenAddr     string        `yaml:"listen_addr"`
	PollInterval   time.Duration `yaml:"poll_interval"`
	BackendBaseURL string        `yaml:"backend_base_url"`
	BackendToken   string        `yaml:"backend_token"`
	ConfigFile     string        `yaml:"-"`
}

func Load() Config {
	cfg := defaultConfig()
	if path := resolveConfigFile("SYNC_WORKER_CONFIG_FILE", "config/resource-sync-worker.yaml", "resource-sync-worker.yaml", "config.yaml"); path != "" {
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
	return cfg
}

func defaultConfig() Config {
	return Config{
		WorkerName:     "resource-sync-worker",
		ListenAddr:     ":8092",
		PollInterval:   10 * time.Second,
		BackendBaseURL: "http://backend:8080",
		BackendToken:   "",
	}
}

func applyEnvOverrides(cfg *Config) {
	cfg.WorkerName = getenv("SYNC_WORKER_NAME", cfg.WorkerName)
	cfg.ListenAddr = getenv("SYNC_WORKER_LISTEN_ADDR", cfg.ListenAddr)
	cfg.PollInterval = durationEnv("SYNC_WORKER_POLL_INTERVAL", cfg.PollInterval)
	cfg.BackendBaseURL = getenv("BACKEND_API_BASE_URL", cfg.BackendBaseURL)
	cfg.BackendToken = getenv("BACKEND_API_TOKEN", cfg.BackendToken)
}

func resolveConfigFile(envKey string, candidates ...string) string {
	if explicit := os.Getenv(envKey); explicit != "" {
		return explicit
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

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
