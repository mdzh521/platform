package machine

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

func (s *Service) sshClientConfig(payload terminalTicketPayload) (*ssh.ClientConfig, error) {
	authMethod, err := buildSSHAuthMethod(payload)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            payload.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: s.hostKeyCallback(payload),
		Timeout:         8 * time.Second,
	}, nil
}

func (s *Service) hostKeyCallback(payload terminalTicketPayload) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		host, port := strings.TrimSpace(payload.Address), payload.Port
		sha256Fingerprint := ssh.FingerprintSHA256(key)
		md5Fingerprint := ssh.FingerprintLegacyMD5(key)

		var trust SSHHostTrust
		err := s.db.Where("address = ? AND port = ?", host, port).First(&trust).Error
		switch {
		case err == nil:
			if trust.FingerprintSHA256 != sha256Fingerprint {
				return fmt.Errorf("SSH host key mismatch for %s:%d", host, port)
			}
			now := time.Now()
			_ = s.db.Model(&SSHHostTrust{}).Where("id = ?", trust.ID).Update("last_verified_at", &now).Error
			return nil
		case err != nil && err != gorm.ErrRecordNotFound:
			return err
		}

		now := time.Now()
		record := SSHHostTrust{
			Address:           host,
			Port:              port,
			Algorithm:         key.Type(),
			FingerprintSHA256: sha256Fingerprint,
			FingerprintMD5:    md5Fingerprint,
			FirstSeenAt:       now,
			LastVerifiedAt:    &now,
		}
		if err := s.db.Create(&record).Error; err != nil {
			return err
		}
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "host_key_trusted", "info", "首次记录 SSH 主机指纹", fmt.Sprintf("%s:%d %s", host, port, sha256Fingerprint))
		return nil
	}
}

func (s *Service) recoverStaleSessions(maxAge time.Duration) {
	if s.db == nil || maxAge <= 0 {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	var stale []Session
	if err := s.db.Where("status IN ? AND started_at < ? AND ended_at IS NULL", []string{"prepared", "active"}, cutoff).Find(&stale).Error; err != nil {
		return
	}
	for _, session := range stale {
		detail := "系统启动时回收超时会话，避免历史假活跃记录继续保留。"
		s.markSessionClosed(session.ID, detail)
		s.recordSessionEvent(session.AssetIDValue(), session.ID, "session_recovered", "warning", "回收历史超时会话", detail)
	}
}
