package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

type interactiveTerminal struct {
	WorkloadID    uint
	ClusterID     uint
	Namespace     string
	WorkloadName  string
	WorkloadKind  string
	PodName       string
	ContainerName string
	Kubeconfig    string
	Cmd           *exec.Cmd
	PTY           *os.File
	cancel        context.CancelFunc
}

func (s *Service) OpenInteractiveTerminal(id uint, input OpenInteractiveTerminalInput) (*interactiveTerminal, map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, nil, err
	}
	pods, err := client.GetRelatedPods(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if len(pods) == 0 {
		return nil, nil, errors.New("no related pods found for current workload")
	}
	selectedPod, err := selectLogPod(pods, input.PodName)
	if err != nil {
		return nil, nil, err
	}

	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 1800
	}
	if timeoutSeconds > 7200 {
		timeoutSeconds = 7200
	}

	cols := input.Cols
	rows := input.Rows
	if cols <= 0 {
		cols = 120
	}
	if rows <= 0 {
		rows = 32
	}

	shell := strings.TrimSpace(input.Shell)
	if shell == "" {
		shell = "/bin/sh -i"
	}
	shellArgs := strings.Fields(shell)
	if len(shellArgs) == 0 {
		shellArgs = []string{"/bin/sh", "-i"}
	}

	kubeconfigPath, err := writeTemporaryKubeconfig(*cluster)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	args := []string{"--kubeconfig", kubeconfigPath, "exec", "-it", "-n", selectedPod.Namespace, selectedPod.Name}
	if strings.TrimSpace(input.ContainerName) != "" {
		args = append(args, "-c", strings.TrimSpace(input.ContainerName))
	}
	args = append(args, "--")
	args = append(args, shellArgs...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		cancel()
		_ = os.Remove(kubeconfigPath)
		return nil, nil, err
	}

	terminal := &interactiveTerminal{
		WorkloadID:    workload.ID,
		ClusterID:     cluster.ID,
		Namespace:     selectedPod.Namespace,
		WorkloadName:  workload.Name,
		WorkloadKind:  workload.Kind,
		PodName:       selectedPod.Name,
		ContainerName: strings.TrimSpace(input.ContainerName),
		Kubeconfig:    kubeconfigPath,
		Cmd:           cmd,
		PTY:           ptmx,
		cancel:        cancel,
	}

	meta := map[string]any{
		"workload_id":     workload.ID,
		"cluster_id":      cluster.ID,
		"cluster_name":    cluster.Name,
		"namespace":       workload.NamespaceName,
		"kind":            workload.Kind,
		"name":            workload.Name,
		"source_pod":      terminal.PodName,
		"container_name":  terminal.ContainerName,
		"shell":           strings.Join(shellArgs, " "),
		"cols":            cols,
		"rows":            rows,
		"timeout_seconds": timeoutSeconds,
		"message":         "interactive terminal opened",
	}
	return terminal, meta, nil
}

func (s *Service) ResizeInteractiveTerminal(terminal *interactiveTerminal, cols, rows int) error {
	if terminal == nil || terminal.PTY == nil {
		return errors.New("interactive terminal is not available")
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return pty.Setsize(terminal.PTY, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func (s *Service) CloseInteractiveTerminal(terminal *interactiveTerminal) {
	if terminal == nil {
		return
	}
	if terminal.cancel != nil {
		terminal.cancel()
	}
	if terminal.PTY != nil {
		_ = terminal.PTY.Close()
	}
	if terminal.Kubeconfig != "" {
		_ = os.Remove(terminal.Kubeconfig)
	}
}

func terminalStatusFrame(status string, data map[string]any) map[string]any {
	frame := map[string]any{
		"type":   "status",
		"status": status,
	}
	for key, value := range data {
		frame[key] = value
	}
	return frame
}

func terminalOutputFrame(data string) map[string]any {
	return map[string]any{
		"type": "output",
		"data": data,
	}
}

func terminalErrorFrame(message string) map[string]any {
	return map[string]any{
		"type":    "error",
		"message": message,
	}
}

func terminalClosedFrame(reason string) map[string]any {
	return map[string]any{
		"type":   "status",
		"status": "closed",
		"reason": reason,
	}
}

func terminalOpenMessage(meta map[string]any) map[string]any {
	return terminalStatusFrame("connected", meta)
}

func formatTerminalCloseReason(err error) string {
	if err == nil {
		return "completed"
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "completed"
	}
	return fmt.Sprintf("process exited: %s", text)
}
