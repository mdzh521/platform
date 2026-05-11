package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RunnerName        string        `yaml:"runner_name"`
	ListenAddr        string        `yaml:"listen_addr"`
	PollInterval      time.Duration `yaml:"poll_interval"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	BackendBaseURL    string        `yaml:"backend_base_url"`
	BackendToken      string        `yaml:"backend_token"`
	WorkspaceRoot     string        `yaml:"workspace_root"`
	ProviderCacheDir  string        `yaml:"provider_cache_dir"`
	TemplateRoot      string        `yaml:"template_root"`
	MaxParallelJobs   int           `yaml:"max_parallel_jobs"`
	SuccessTTL        time.Duration `yaml:"workspace_success_ttl"`
	FailureTTL        time.Duration `yaml:"workspace_failure_ttl"`
	DestroyedTTL      time.Duration `yaml:"workspace_destroyed_ttl"`
	ConfigFile        string        `yaml:"-"`
}

func Load() Config {
	cfg := defaultConfig()
	if path := resolveConfigFile("RUNNER_CONFIG_FILE", "config/terraform-runner.yaml", "terraform-runner.yaml", "config.yaml"); path != "" {
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
		RunnerName:        "terraform-runner",
		ListenAddr:        ":8091",
		PollInterval:      15 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		BackendBaseURL:    "http://backend:8080",
		BackendToken:      "",
		WorkspaceRoot:     "/runner-data/jobs",
		ProviderCacheDir:  "/runner-data/plugin-cache",
		TemplateRoot:      "/app/templates",
		MaxParallelJobs:   2,
		SuccessTTL:        24 * time.Hour,
		FailureTTL:        72 * time.Hour,
		DestroyedTTL:      time.Hour,
	}
}

func applyEnvOverrides(cfg *Config) {
	cfg.RunnerName = getenv("RUNNER_NAME", cfg.RunnerName)
	cfg.ListenAddr = getenv("RUNNER_LISTEN_ADDR", cfg.ListenAddr)
	cfg.PollInterval = durationEnv("RUNNER_POLL_INTERVAL", cfg.PollInterval)
	cfg.HeartbeatInterval = durationEnv("RUNNER_HEARTBEAT_INTERVAL", cfg.HeartbeatInterval)
	cfg.BackendBaseURL = getenv("BACKEND_API_BASE_URL", cfg.BackendBaseURL)
	cfg.BackendToken = getenv("BACKEND_API_TOKEN", cfg.BackendToken)
	cfg.WorkspaceRoot = getenv("TERRAFORM_WORKSPACE_ROOT", cfg.WorkspaceRoot)
	cfg.ProviderCacheDir = getenv("TERRAFORM_PROVIDER_CACHE_DIR", cfg.ProviderCacheDir)
	cfg.TemplateRoot = getenv("TERRAFORM_TEMPLATE_ROOT", cfg.TemplateRoot)
	cfg.MaxParallelJobs = intEnv("RUNNER_MAX_PARALLEL_JOBS", cfg.MaxParallelJobs)
	cfg.SuccessTTL = durationEnv("RUNNER_WORKSPACE_SUCCESS_TTL", cfg.SuccessTTL)
	cfg.FailureTTL = durationEnv("RUNNER_WORKSPACE_FAILURE_TTL", cfg.FailureTTL)
	cfg.DestroyedTTL = durationEnv("RUNNER_WORKSPACE_DESTROYED_TTL", cfg.DestroyedTTL)
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

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
