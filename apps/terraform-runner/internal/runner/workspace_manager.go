package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WorkspaceManager struct {
	root             string
	providerCacheDir string
	templateRoot     string
	successTTL       time.Duration
	failureTTL       time.Duration
	destroyedTTL     time.Duration
}

func NewWorkspaceManager(root, providerCacheDir, templateRoot string, successTTL, failureTTL, destroyedTTL time.Duration) *WorkspaceManager {
	return &WorkspaceManager{
		root:             root,
		providerCacheDir: providerCacheDir,
		templateRoot:     templateRoot,
		successTTL:       successTTL,
		failureTTL:       failureTTL,
		destroyedTTL:     destroyedTTL,
	}
}

func (m *WorkspaceManager) WorkspaceDir(jobID uint) string {
	return filepath.Join(m.root, fmt.Sprintf("job-%d", jobID))
}

func (m *WorkspaceManager) TemplateDir(blueprintCode string) string {
	return filepath.Join(m.templateRoot, blueprintCode)
}

func (m *WorkspaceManager) ResolveTemplateDir(templatePath, blueprintCode string) string {
	path := strings.TrimSpace(templatePath)
	if path == "" {
		return m.TemplateDir(blueprintCode)
	}
	if filepath.IsAbs(path) {
		return path
	}
	normalized := filepath.ToSlash(path)
	if idx := strings.Index(normalized, "templates/"); idx >= 0 {
		relative := strings.TrimPrefix(normalized[idx+len("templates/"):], "/")
		if relative != "" {
			return filepath.Join(m.templateRoot, filepath.FromSlash(relative))
		}
	}
	if strings.Contains(normalized, "/") {
		return filepath.Join(m.templateRoot, filepath.Base(normalized))
	}
	return filepath.Join(m.templateRoot, normalized)
}

func (m *WorkspaceManager) ProviderCacheDir() string {
	return m.providerCacheDir
}

func (m *WorkspaceManager) PrepareWorkspace(path string) error {
	if err := os.MkdirAll(m.providerCacheDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(path, "terraform.tfstate")); err == nil {
		return os.MkdirAll(path, 0o755)
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func (m *WorkspaceManager) EnsureWorkspace(path string) error {
	if err := os.MkdirAll(m.providerCacheDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func (m *WorkspaceManager) CopyTemplateToWorkspace(templateDir, workspaceDir string) error {
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(workspaceDir, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()
		destination, err := os.Create(target)
		if err != nil {
			return err
		}
		defer destination.Close()
		if _, err := io.Copy(destination, source); err != nil {
			return err
		}
		return destination.Chmod(info.Mode())
	})
}

func (m *WorkspaceManager) RecordStatus(path, status string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	marker := filepath.Join(path, ".job-status")
	if err := os.WriteFile(marker, []byte(strings.TrimSpace(status)), 0o600); err != nil {
		return err
	}
	now := time.Now()
	return os.Chtimes(path, now, now)
}

func (m *WorkspaceManager) CleanupExpired() error {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "job-") {
			continue
		}
		path := filepath.Join(m.root, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		status := m.readStatus(path)
		ttl := m.ttlForStatus(status)
		if ttl <= 0 {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			continue
		}
		if now.Sub(info.ModTime()) >= ttl {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *WorkspaceManager) readStatus(path string) string {
	raw, err := os.ReadFile(filepath.Join(path, ".job-status"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func (m *WorkspaceManager) ttlForStatus(status string) time.Duration {
	switch status {
	case "destroyed":
		return m.destroyedTTL
	case "failed", "cancelled":
		return m.failureTTL
	case "succeeded", "planned":
		return m.successTTL
	default:
		return m.failureTTL
	}
}
