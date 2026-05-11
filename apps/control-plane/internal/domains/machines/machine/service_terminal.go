package machine

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type interactiveTerminal struct {
	AssetID       uint
	SessionID     uint
	AssetName     string
	Address       string
	Port          int
	Protocol      string
	Username      string
	client        *ssh.Client
	gatewayClient *ssh.Client
	session       *ssh.Session
	stdin         io.WriteCloser
	output        *io.PipeReader
	outputSink    *io.PipeWriter
	resizeMu      sync.Mutex
	auditMu       sync.Mutex
	commandBuf    string
}

func (s *Service) OpenInteractiveTerminal(id uint, ticket string, cols, rows int) (*interactiveTerminal, map[string]any, error) {
	if cols <= 0 {
		cols = 120
	}
	if rows <= 0 {
		rows = 32
	}
	payload, err := s.consumeTerminalTicket(ticket)
	if err != nil {
		return nil, nil, err
	}
	if payload.AssetID != id {
		return nil, nil, errors.New("terminal ticket does not match asset")
	}

	config, err := s.sshClientConfig(payload)
	if err != nil {
		return nil, nil, err
	}
	address := fmt.Sprintf("%s:%d", payload.Address, payload.Port)
	conn, gatewayClient, err := s.openTargetSSHClient(payload, address, config)
	if err != nil {
		s.markSessionClosed(payload.SessionID, "SSH 连接失败: "+err.Error())
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "terminal_error", "warning", "SSH 连接失败", err.Error())
		return nil, nil, err
	}
	session, err := conn.NewSession()
	if err != nil {
		_ = conn.Close()
		s.markSessionClosed(payload.SessionID, "SSH 会话创建失败: "+err.Error())
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "terminal_error", "warning", "SSH 会话创建失败", err.Error())
		return nil, nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = conn.Close()
		s.markSessionClosed(payload.SessionID, "SSH stdin 初始化失败: "+err.Error())
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "terminal_error", "warning", "SSH stdin 初始化失败", err.Error())
		return nil, nil, err
	}
	reader, writer := io.Pipe()
	session.Stdout = writer
	session.Stderr = writer
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = stdin.Close()
		_ = writer.Close()
		_ = reader.Close()
		_ = session.Close()
		_ = conn.Close()
		s.markSessionClosed(payload.SessionID, "SSH PTY 初始化失败: "+err.Error())
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "terminal_error", "warning", "SSH PTY 初始化失败", err.Error())
		return nil, nil, err
	}
	if err := session.Shell(); err != nil {
		_ = stdin.Close()
		_ = writer.Close()
		_ = reader.Close()
		_ = session.Close()
		_ = conn.Close()
		s.markSessionClosed(payload.SessionID, "SSH shell 启动失败: "+err.Error())
		s.recordSessionEvent(payload.AssetID, payload.SessionID, "terminal_error", "warning", "SSH shell 启动失败", err.Error())
		return nil, nil, err
	}

	terminal := &interactiveTerminal{
		AssetID:       payload.AssetID,
		SessionID:     payload.SessionID,
		AssetName:     payload.Address,
		Address:       payload.Address,
		Port:          payload.Port,
		Protocol:      payload.Protocol,
		Username:      payload.Username,
		client:        conn,
		gatewayClient: gatewayClient,
		session:       session,
		stdin:         stdin,
		output:        reader,
		outputSink:    writer,
	}
	s.markSessionActive(payload.SessionID)

	meta := map[string]any{
		"asset_id":   payload.AssetID,
		"session_id": payload.SessionID,
		"address":    payload.Address,
		"port":       payload.Port,
		"protocol":   payload.Protocol,
		"username":   payload.Username,
		"cols":       cols,
		"rows":       rows,
		"message":    "machine interactive terminal opened",
		"terminal":   "ssh",
	}
	if payload.AccessMode == "via_gateway" {
		meta["access_mode"] = "via_gateway"
		meta["gateway_name"] = payload.GatewayName
		meta["gateway_address"] = payload.GatewayAddress
	}
	return terminal, meta, nil
}

func (s *Service) openTargetSSHClient(payload terminalTicketPayload, address string, config *ssh.ClientConfig) (*ssh.Client, *ssh.Client, error) {
	if payload.AccessMode != "via_gateway" || payload.GatewayAssetID == 0 {
		client, err := ssh.Dial("tcp", address, config)
		return client, nil, err
	}
	gatewayConfig, err := s.gatewaySSHClientConfig(payload)
	if err != nil {
		return nil, nil, err
	}
	gatewayAddress := fmt.Sprintf("%s:%d", payload.GatewayAddress, payload.GatewayPort)
	gatewayClient, err := ssh.Dial("tcp", gatewayAddress, gatewayConfig)
	if err != nil {
		return nil, nil, err
	}
	targetConn, err := gatewayClient.Dial("tcp", address)
	if err != nil {
		_ = gatewayClient.Close()
		return nil, nil, err
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(targetConn, address, config)
	if err != nil {
		_ = targetConn.Close()
		_ = gatewayClient.Close()
		return nil, nil, err
	}
	return ssh.NewClient(clientConn, chans, reqs), gatewayClient, nil
}

func buildSSHAuthMethod(payload terminalTicketPayload) (ssh.AuthMethod, error) {
	switch payload.AuthType {
	case "", "password":
		if payload.Password == "" {
			return nil, errors.New("missing SSH password")
		}
		return ssh.Password(payload.Password), nil
	case "ssh_key":
		if payload.PrivateKey == "" {
			return nil, errors.New("missing SSH private key")
		}
		var signer ssh.Signer
		var err error
		if payload.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(payload.PrivateKey), []byte(payload.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(payload.PrivateKey))
		}
		if err != nil {
			return nil, err
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, errors.New("unsupported SSH auth type")
	}
}

func (s *Service) ResizeInteractiveTerminal(terminal *interactiveTerminal, cols, rows int) error {
	if terminal == nil || terminal.session == nil {
		return errors.New("machine interactive terminal is not available")
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	terminal.resizeMu.Lock()
	defer terminal.resizeMu.Unlock()
	return terminal.session.WindowChange(rows, cols)
}

func (s *Service) CloseInteractiveTerminal(terminal *interactiveTerminal) {
	if terminal == nil {
		return
	}
	if terminal.stdin != nil {
		_ = terminal.stdin.Close()
	}
	if terminal.outputSink != nil {
		_ = terminal.outputSink.Close()
	}
	if terminal.output != nil {
		_ = terminal.output.Close()
	}
	if terminal.session != nil {
		_ = terminal.session.Close()
	}
	if terminal.client != nil {
		_ = terminal.client.Close()
	}
	if terminal.gatewayClient != nil {
		_ = terminal.gatewayClient.Close()
	}
	if terminal.SessionID > 0 {
		s.markSessionClosed(terminal.SessionID, "SSH 在线终端已关闭。")
	}
}

func (s *Service) ObserveInteractiveTerminalInput(terminal *interactiveTerminal, data string) {
	if terminal == nil || strings.TrimSpace(data) == "" && !strings.ContainsAny(data, "\r\n\u0003\u0004\u007f") {
		return
	}
	terminal.auditMu.Lock()
	defer terminal.auditMu.Unlock()

	for _, r := range data {
		switch r {
		case '\r', '\n':
			s.flushInteractiveTerminalCommand(terminal)
		case '\u0003':
			s.recordSessionEvent(terminal.AssetID, terminal.SessionID, "terminal_signal", "warning", "发送 Ctrl+C", "用户在交互终端发送了 Ctrl+C。")
			terminal.commandBuf = ""
		case '\u0004':
			s.recordSessionEvent(terminal.AssetID, terminal.SessionID, "terminal_signal", "info", "发送 Ctrl+D", "用户在交互终端发送了 Ctrl+D。")
			s.flushInteractiveTerminalCommand(terminal)
			terminal.commandBuf = ""
		case '\u007f', '\b':
			if len(terminal.commandBuf) > 0 {
				terminal.commandBuf = trimLastRune(terminal.commandBuf)
			}
		case '\t':
			terminal.commandBuf += "\t"
		default:
			if r < 32 {
				continue
			}
			terminal.commandBuf += string(r)
		}
	}
}

func (s *Service) flushInteractiveTerminalCommand(terminal *interactiveTerminal) {
	if terminal == nil {
		return
	}
	command := strings.TrimSpace(terminal.commandBuf)
	terminal.commandBuf = ""
	if command == "" {
		return
	}
	s.recordSessionEvent(terminal.AssetID, terminal.SessionID, "terminal_command", "info", "执行命令", command)
}

func trimLastRune(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	if len(runes) <= 1 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func formatTerminalCloseReason(err error) string {
	if err == nil {
		return "completed"
	}
	text := err.Error()
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return "timeout"
	}
	if text == "" {
		return "completed"
	}
	return text
}

func (s *Service) gatewaySSHClientConfig(payload terminalTicketPayload) (*ssh.ClientConfig, error) {
	gatewayPayload := terminalTicketPayload{
		AssetID:    payload.GatewayAssetID,
		SessionID:  payload.SessionID,
		Address:    payload.GatewayAddress,
		Port:       payload.GatewayPort,
		Username:   payload.GatewayUsername,
		AuthType:   payload.GatewayAuthType,
		Password:   payload.GatewayPassword,
		PrivateKey: payload.GatewayPrivateKey,
		Passphrase: payload.GatewayPassphrase,
	}
	authMethod, err := buildSSHAuthMethod(terminalTicketPayload{
		AuthType:   gatewayPayload.AuthType,
		Password:   gatewayPayload.Password,
		PrivateKey: gatewayPayload.PrivateKey,
		Passphrase: gatewayPayload.Passphrase,
	})
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            gatewayPayload.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: s.hostKeyCallback(gatewayPayload),
		Timeout:         10 * time.Second,
	}, nil
}
