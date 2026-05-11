package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	cfg := Config{
		WorkerName:     "machine-enrollment-worker",
		ListenAddr:     ":8093",
		PollInterval:   10 * time.Second,
		BackendBaseURL: "http://127.0.0.1:18080",
	}
	if path := resolveConfigFile("MACHINE_ENROLLMENT_WORKER_CONFIG_FILE", "config/machine-enrollment-worker.yaml", "machine-enrollment-worker.yaml", "config.yaml"); path != "" {
		if err := applyConfigFile(path, &cfg); err != nil {
			panic(fmt.Errorf("parse config file %s: %w", path, err))
		}
		cfg.ConfigFile = path
	}
	if value := os.Getenv("MACHINE_ENROLLMENT_WORKER_NAME"); value != "" {
		cfg.WorkerName = value
	}
	if value := os.Getenv("MACHINE_ENROLLMENT_WORKER_LISTEN_ADDR"); value != "" {
		cfg.ListenAddr = value
	}
	if value := os.Getenv("MACHINE_ENROLLMENT_WORKER_POLL_INTERVAL"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			cfg.PollInterval = parsed
		}
	}
	if value := os.Getenv("BACKEND_API_BASE_URL"); value != "" {
		cfg.BackendBaseURL = value
	}
	if value := os.Getenv("BACKEND_API_TOKEN"); value != "" {
		cfg.BackendToken = value
	}
	return cfg
}

func applyConfigFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "worker_name":
			cfg.WorkerName = value
		case "listen_addr":
			cfg.ListenAddr = value
		case "poll_interval":
			if parsed, err := time.ParseDuration(value); err == nil {
				cfg.PollInterval = parsed
			}
		case "backend_base_url":
			cfg.BackendBaseURL = value
		case "backend_token":
			cfg.BackendToken = value
		}
	}
	return scanner.Err()
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
