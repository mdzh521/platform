package machine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func (s *Service) ListSFTP(assetID, sessionID uint, requestPath string) (SFTPListResult, error) {
	if sessionID == 0 {
		return SFTPListResult{}, errors.New("missing session id")
	}
	payload, err := s.getSessionAccessPayload(sessionID)
	if err != nil {
		return SFTPListResult{}, err
	}
	if payload.AssetID != assetID {
		return SFTPListResult{}, errors.New("session does not match asset")
	}

	config, err := s.sshClientConfig(payload)
	if err != nil {
		return SFTPListResult{}, err
	}

	address := fmt.Sprintf("%s:%d", payload.Address, payload.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return SFTPListResult{}, err
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return SFTPListResult{}, err
	}
	defer client.Close()

	targetPath := normalizeSFTPPath(requestPath)
	if targetPath == "" {
		targetPath = "."
	}
	if targetPath == "." {
		if wd, err := client.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
			targetPath = wd
		}
	}
	entries, err := client.ReadDir(targetPath)
	if err != nil {
		return SFTPListResult{}, err
	}

	result := make([]SFTPEntryView, 0, len(entries))
	cleanTarget := normalizeSFTPPath(targetPath)
	for _, entry := range entries {
		entryType := "file"
		if entry.IsDir() {
			entryType = "dir"
		}
		owner, group := resolveSFTPOwnerGroup(entry)
		entryPath := path.Join(cleanTarget, entry.Name())
		if cleanTarget == "/" {
			entryPath = "/" + entry.Name()
		}
		result = append(result, SFTPEntryView{
			Name:    entry.Name(),
			Path:    entryPath,
			Type:    entryType,
			Size:    entry.Size(),
			Mode:    entry.Mode().String(),
			Owner:   owner,
			Group:   group,
			ModTime: entry.ModTime(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type == "dir"
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	parent := path.Dir(cleanTarget)
	if cleanTarget == "/" || cleanTarget == "." {
		parent = "/"
	}

	return SFTPListResult{
		Path:    cleanTarget,
		Parent:  parent,
		Entries: result,
	}, nil
}

type sftpDownloadStream struct {
	io.ReadCloser
	closeFn func()
}

func (s *Service) DownloadSFTP(assetID, sessionID uint, requestPath string) (io.ReadCloser, string, error) {
	payload, client, closeFn, err := s.openSFTPClient(assetID, sessionID)
	if err != nil {
		return nil, "", err
	}

	targetPath := normalizeSFTPPath(requestPath)
	file, err := client.Open(targetPath)
	if err != nil {
		closeFn()
		return nil, "", describeSFTPFileError(targetPath, err)
	}
	filename := path.Base(targetPath)
	if filename == "." || filename == "/" || filename == "" {
		filename = payload.Address
	}
	return &sftpDownloadStream{
		ReadCloser: file,
		closeFn:    closeFn,
	}, filename, nil
}

func (s *sftpDownloadStream) Close() error {
	var err error
	if s.ReadCloser != nil {
		err = s.ReadCloser.Close()
	}
	if s.closeFn != nil {
		s.closeFn()
	}
	return err
}

func (s *Service) UploadSFTP(assetID, sessionID uint, targetDir, filename string, reader io.Reader) error {
	if strings.TrimSpace(filename) == "" {
		return errors.New("missing upload file name")
	}
	_, client, closeFn, err := s.openSFTPClient(assetID, sessionID)
	if err != nil {
		return err
	}
	defer closeFn()

	dirPath := normalizeSFTPPath(targetDir)
	targetPath := path.Join(dirPath, path.Base(filename))
	if dirPath == "/" {
		targetPath = "/" + path.Base(filename)
	}

	file, err := client.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	return nil
}

func (s *Service) openSFTPClient(assetID, sessionID uint) (terminalTicketPayload, *sftp.Client, func(), error) {
	payload, err := s.getSessionAccessPayload(sessionID)
	if err != nil {
		return terminalTicketPayload{}, nil, nil, err
	}
	if payload.AssetID != assetID {
		return terminalTicketPayload{}, nil, nil, errors.New("session does not match asset")
	}

	config, err := s.sshClientConfig(payload)
	if err != nil {
		return terminalTicketPayload{}, nil, nil, err
	}

	address := fmt.Sprintf("%s:%d", payload.Address, payload.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return terminalTicketPayload{}, nil, nil, err
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return terminalTicketPayload{}, nil, nil, err
	}
	closeFn := func() {
		_ = client.Close()
		_ = conn.Close()
	}
	return payload, client, closeFn, nil
}

func (s *Service) getSessionAccessPayload(sessionID uint) (terminalTicketPayload, error) {
	if s.cache == nil {
		return terminalTicketPayload{}, errors.New("terminal cache is not configured")
	}
	raw, ok, err := s.cache.Get(context.Background(), machineSessionAccessKey(sessionID))
	if err != nil {
		return terminalTicketPayload{}, err
	}
	if !ok {
		return terminalTicketPayload{}, errors.New("session access expired")
	}
	var payload terminalTicketPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return terminalTicketPayload{}, err
	}
	return payload, nil
}

func normalizeSFTPPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "."
	}
	if strings.HasPrefix(value, "/") {
		return path.Clean(value)
	}
	return path.Clean("/" + value)
}

func describeSFTPFileError(targetPath string, err error) error {
	fileName := path.Base(strings.TrimSpace(targetPath))
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = targetPath
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "permission denied"):
		return fmt.Errorf("无法读取文件 %s：当前登录账号没有读取权限", fileName)
	case strings.Contains(message, "no such file"), strings.Contains(message, "not exist"), strings.Contains(message, "not found"):
		return fmt.Errorf("无法读取文件 %s：文件不存在或路径已变化", fileName)
	case strings.Contains(message, "failure"):
		return fmt.Errorf("无法读取文件 %s：远端 SFTP 服务拒绝访问，通常是权限不足或文件不可读", fileName)
	default:
		return fmt.Errorf("无法读取文件 %s：%s", fileName, err.Error())
	}
}

func resolveSFTPOwnerGroup(entry fs.FileInfo) (string, string) {
	sys := entry.Sys()
	if sys == nil {
		return "-", "-"
	}
	stat, ok := sys.(*sftp.FileStat)
	if !ok || stat == nil {
		return "-", "-"
	}
	owner := "-"
	group := "-"
	if stat.UID != 0 {
		owner = fmt.Sprintf("%d", stat.UID)
	}
	if stat.GID != 0 {
		group = fmt.Sprintf("%d", stat.GID)
	}
	return owner, group
}
