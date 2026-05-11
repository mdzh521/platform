package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type shellSession struct {
	ID            string
	WorkloadID    uint
	ClusterID     uint
	Namespace     string
	PodName       string
	ContainerName string
	Kubeconfig    string
	Cmd           *exec.Cmd
	Stdin         io.WriteCloser
	cancel        context.CancelFunc
	done          chan struct{}
	createdAt     time.Time
	mu            sync.Mutex
	buffer        bytes.Buffer
	closed        bool
	closeReason   string
	deleteOnClose bool
}

func (s *Service) OpenShellSession(id uint, input OpenShellSessionInput) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	pods, err := client.GetRelatedPods(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if len(pods) == 0 {
		return nil, errors.New("no related pods found for current workload")
	}
	selectedPod, err := selectLogPod(pods, input.PodName)
	if err != nil {
		return nil, err
	}
	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 900
	}
	if timeoutSeconds > 3600 {
		timeoutSeconds = 3600
	}
	kubeconfigPath, err := writeTemporaryKubeconfig(*cluster)
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("shell-%d-%d", workload.ID, time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	args := []string{"--kubeconfig", kubeconfigPath, "exec", "-i", "-n", selectedPod.Namespace, selectedPod.Name}
	if strings.TrimSpace(input.ContainerName) != "" {
		args = append(args, "-c", strings.TrimSpace(input.ContainerName))
	}
	args = append(args, "--", "/bin/sh")
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		_ = os.Remove(kubeconfigPath)
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(kubeconfigPath)
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(kubeconfigPath)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		_ = os.Remove(kubeconfigPath)
		return nil, err
	}

	session := &shellSession{
		ID:            sessionID,
		WorkloadID:    workload.ID,
		ClusterID:     cluster.ID,
		Namespace:     selectedPod.Namespace,
		PodName:       selectedPod.Name,
		ContainerName: strings.TrimSpace(input.ContainerName),
		Kubeconfig:    kubeconfigPath,
		Cmd:           cmd,
		Stdin:         stdin,
		cancel:        cancel,
		done:          make(chan struct{}),
		createdAt:     time.Now(),
	}
	s.shellMu.Lock()
	s.shellSessions[sessionID] = session
	s.shellMu.Unlock()

	go s.consumeShellOutput(session, io.MultiReader(stdout, stderr))
	go s.waitShellSession(session)

	return map[string]any{
		"workload_id":     workload.ID,
		"cluster_id":      cluster.ID,
		"cluster_name":    cluster.Name,
		"namespace":       workload.NamespaceName,
		"kind":            workload.Kind,
		"name":            workload.Name,
		"session_id":      session.ID,
		"source_pod":      session.PodName,
		"container_name":  session.ContainerName,
		"timeout_seconds": timeoutSeconds,
		"message":         "shell session opened",
	}, nil
}

func (s *Service) SendShellInput(workloadID uint, sessionID, input string) (map[string]any, error) {
	session, err := s.requireShellSession(workloadID, sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil, errors.New("shell session is closed")
	}
	if _, err := io.WriteString(session.Stdin, input); err != nil {
		return nil, err
	}
	return map[string]any{
		"session_id": session.ID,
		"written":    len(input),
		"message":    "input sent",
	}, nil
}

func (s *Service) ReadShellOutput(workloadID uint, sessionID string, offset int) (map[string]any, error) {
	session, err := s.requireShellSession(workloadID, sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	data := session.buffer.Bytes()
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	return map[string]any{
		"session_id":   session.ID,
		"offset":       len(data),
		"chunk":        string(data[offset:]),
		"closed":       session.closed,
		"close_reason": session.closeReason,
	}, nil
}

func (s *Service) CloseShellSession(workloadID uint, sessionID string) error {
	session, err := s.requireShellSession(workloadID, sessionID)
	if err != nil {
		return err
	}
	session.mu.Lock()
	session.deleteOnClose = true
	session.mu.Unlock()
	s.closeShellSession(session, "closed by user")
	return nil
}

func (s *Service) requireShellSession(workloadID uint, sessionID string) (*shellSession, error) {
	s.shellMu.RLock()
	session, ok := s.shellSessions[sessionID]
	s.shellMu.RUnlock()
	if !ok || session.WorkloadID != workloadID {
		return nil, errors.New("shell session not found")
	}
	return session, nil
}

func (s *Service) consumeShellOutput(session *shellSession, reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			session.mu.Lock()
			_, _ = session.buffer.Write(buffer[:n])
			session.mu.Unlock()
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.closeShellSession(session, err.Error())
			}
			return
		}
	}
}

func (s *Service) waitShellSession(session *shellSession) {
	err := session.Cmd.Wait()
	reason := "completed"
	if err != nil {
		reason = err.Error()
	}
	s.closeShellSession(session, reason)
}

func (s *Service) closeShellSession(session *shellSession, reason string) {
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return
	}
	session.closed = true
	session.closeReason = reason
	deleteOnClose := session.deleteOnClose
	session.mu.Unlock()

	session.cancel()
	_ = session.Stdin.Close()
	_ = os.Remove(session.Kubeconfig)

	if deleteOnClose {
		s.shellMu.Lock()
		delete(s.shellSessions, session.ID)
		s.shellMu.Unlock()
	} else {
		go func(id string) {
			time.Sleep(5 * time.Minute)
			s.shellMu.Lock()
			delete(s.shellSessions, id)
			s.shellMu.Unlock()
		}(session.ID)
	}

	select {
	case <-session.done:
	default:
		close(session.done)
	}
}
