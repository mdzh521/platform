package machine

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend-center/internal/domains/iam/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var terminalUpgrader = websocket.Upgrader{
	Subprotocols: []string{"bc-terminal"},
}

const (
	terminalPingPeriod = 20 * time.Second
	terminalReadWait   = 75 * time.Second
	terminalWriteWait  = 10 * time.Second
)

type terminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func (h *Handler) OpenInteractiveTerminal(c *gin.Context) {
	if _, err := h.authenticateTerminalRequest(terminalRequestToken(c)); err != nil {
		c.JSON(http.StatusUnauthorized, terminalErrorFrame("invalid or missing token"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, terminalErrorFrame("invalid asset id"))
		return
	}
	ticket := c.Query("ticket")
	if ticket == "" {
		c.JSON(http.StatusBadRequest, terminalErrorFrame("missing terminal ticket"))
		return
	}

	upgrader := terminalUpgrader
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return isAllowedWebSocketOrigin(r, h.trustedOrigins)
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	_ = conn.SetReadDeadline(time.Now().Add(terminalReadWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(terminalReadWait))
	})

	terminal, meta, err := h.service.OpenInteractiveTerminal(uint(id), ticket, parseTerminalInt(c.Query("cols"), 120), parseTerminalInt(c.Query("rows"), 32))
	if err != nil {
		_ = conn.WriteJSON(terminalErrorFrame(err.Error()))
		_ = conn.Close()
		return
	}
	defer h.service.CloseInteractiveTerminal(terminal)

	var writeMu sync.Mutex
	writeJSON := func(payload map[string]any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(terminalWriteWait))
		return conn.WriteJSON(payload)
	}
	if err := writeJSON(terminalOpenMessage(meta)); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buffer := make([]byte, 4096)
		for {
			n, readErr := terminal.output.Read(buffer)
			if n > 0 {
				if err := writeJSON(terminalOutputFrame(string(buffer[:n]))); err != nil {
					return
				}
			}
			if readErr != nil {
				reason := "completed"
				if !errors.Is(readErr, io.EOF) {
					reason = readErr.Error()
				}
				_ = writeJSON(terminalClosedFrame(reason))
				return
			}
		}
	}()

	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(terminalPingPeriod)
		defer ticker.Stop()
		defer close(pingDone)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeMu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(terminalWriteWait))
				err := conn.WriteControl(websocket.PingMessage, []byte("keepalive"), time.Now().Add(terminalWriteWait))
				writeMu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()

	for {
		var message terminalMessage
		if err := conn.ReadJSON(&message); err != nil {
			break
		}
		switch message.Type {
		case "input":
			log.Printf("machine terminal input asset=%d session=%d len=%d sample=%q", terminal.AssetID, terminal.SessionID, len(message.Data), truncateTerminalSample(message.Data))
			if terminal.stdin != nil {
				if _, err := io.WriteString(terminal.stdin, message.Data); err != nil {
					_ = writeJSON(terminalErrorFrame(err.Error()))
					return
				}
				h.service.ObserveInteractiveTerminalInput(terminal, message.Data)
			}
		case "resize":
			if err := h.service.ResizeInteractiveTerminal(terminal, message.Cols, message.Rows); err != nil {
				_ = writeJSON(terminalErrorFrame(err.Error()))
				return
			}
		case "ping":
			if err := writeJSON(map[string]any{"type": "pong"}); err != nil {
				return
			}
		}
	}

	h.service.CloseInteractiveTerminal(terminal)
	_ = conn.Close()
	<-done
	<-pingDone
}

func isAllowedWebSocketOrigin(request *http.Request, trustedOrigins []string) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	requestHost := websocketRequestHost(request)
	if sameWebSocketHost(originURL, requestHost) {
		return true
	}
	normalizedOrigin := strings.TrimRight(strings.ToLower(origin), "/")
	for _, item := range trustedOrigins {
		if normalizedOrigin == strings.TrimRight(strings.ToLower(strings.TrimSpace(item)), "/") {
			return true
		}
	}
	return false
}

func websocketRequestHost(request *http.Request) string {
	forwardedHost := strings.TrimSpace(request.Header.Get("X-Forwarded-Host"))
	if forwardedHost != "" {
		return strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}
	return strings.TrimSpace(request.Host)
}

func sameWebSocketHost(originURL *url.URL, requestHost string) bool {
	if originURL == nil {
		return false
	}
	requestURL := &url.URL{Host: strings.TrimSpace(requestHost)}
	if requestURL.Host == "" {
		return false
	}
	originHostname := strings.TrimSpace(strings.ToLower(originURL.Hostname()))
	requestHostname := strings.TrimSpace(strings.ToLower(requestURL.Hostname()))
	if originHostname == "" || requestHostname == "" {
		return false
	}
	return originHostname == requestHostname
}

func (h *Handler) authenticateTerminalRequest(tokenString string) (*auth.Claims, error) {
	if tokenString == "" {
		return nil, errors.New("missing token")
	}
	token, err := jwt.ParseWithClaims(tokenString, &auth.Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func terminalRequestToken(c *gin.Context) string {
	header := c.GetHeader("Sec-WebSocket-Protocol")
	if header == "" {
		return c.Query("token")
	}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" || value == "bc-terminal" {
			continue
		}
		return value
	}
	return c.Query("token")
}

func parseTerminalInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func truncateTerminalSample(value string) string {
	if len(value) <= 32 {
		return value
	}
	return value[:32]
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
