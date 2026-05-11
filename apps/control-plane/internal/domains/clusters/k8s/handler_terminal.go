package k8s

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"backend-center/internal/domains/iam/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var terminalUpgrader = websocket.Upgrader{
	Subprotocols: []string{"bc-terminal"},
}

type terminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func (h *Handler) OpenInteractiveTerminal(c *gin.Context) {
	if _, err := h.authenticateTerminalRequest(terminalRequestToken(c)); err != nil {
		response := terminalErrorFrame("invalid or missing token")
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, terminalErrorFrame("invalid workload id"))
		return
	}

	input := OpenInteractiveTerminalInput{
		PodName:        strings.TrimSpace(c.Query("pod_name")),
		ContainerName:  strings.TrimSpace(c.Query("container_name")),
		Shell:          strings.TrimSpace(c.Query("shell")),
		TimeoutSeconds: parseTerminalInt(c.Query("timeout_seconds"), 1800),
		Cols:           parseTerminalInt(c.Query("cols"), 120),
		Rows:           parseTerminalInt(c.Query("rows"), 32),
	}

	upgrader := terminalUpgrader
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return isAllowedWebSocketOrigin(r, h.trustedOrigins)
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	terminal, meta, err := h.service.OpenInteractiveTerminal(uint(id), input)
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
			n, readErr := terminal.PTY.Read(buffer)
			if n > 0 {
				if err := writeJSON(terminalOutputFrame(string(buffer[:n]))); err != nil {
					return
				}
			}
			if readErr != nil {
				reason := "completed"
				if !errors.Is(readErr, io.EOF) {
					reason = readErr.Error()
				} else if terminal.Cmd != nil {
					reason = formatTerminalCloseReason(terminal.Cmd.Wait())
				}
				_ = writeJSON(terminalClosedFrame(reason))
				return
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
			if _, err := io.WriteString(terminal.PTY, message.Data); err != nil {
				_ = writeJSON(terminalErrorFrame(err.Error()))
				return
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
	if strings.TrimSpace(tokenString) == "" {
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
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
