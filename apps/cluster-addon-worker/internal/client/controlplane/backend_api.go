package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cluster-addon-worker/internal/domain"
)

type BackendAPI struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewBackendAPI(baseURL, token string) *BackendAPI {
	return &BackendAPI{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (b *BackendAPI) Claim(ctx context.Context, workerName string) (*domain.Claim, error) {
	payload, err := b.request(ctx, http.MethodPost, "/api/v1/cloud/internal/cluster-addons/claim", map[string]any{"worker_name": workerName})
	if err != nil || payload.Data == nil {
		return nil, err
	}
	var claim domain.Claim
	raw, _ := json.Marshal(payload.Data)
	if err := json.Unmarshal(raw, &claim); err != nil {
		return nil, err
	}
	if claim.ExecutionID == 0 {
		return nil, nil
	}
	return &claim, nil
}

func (b *BackendAPI) Heartbeat(ctx context.Context, executionID uint, workerName, status string) error {
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/cluster-addons/%d/heartbeat", executionID), map[string]any{
		"worker_name": workerName,
		"status":      status,
	})
	return err
}

func (b *BackendAPI) Report(ctx context.Context, executionID uint, result domain.Result) error {
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/cluster-addons/%d/result", executionID), map[string]any{
		"worker_name":   result.WorkerName,
		"status":        result.Status,
		"error_message": result.ErrorMessage,
		"result":        result.Result,
	})
	return err
}

type responsePayload struct {
	Data  map[string]any `json:"data"`
	Error string         `json:"error"`
}

func (b *BackendAPI) request(ctx context.Context, method, path string, body map[string]any) (responsePayload, error) {
	var payload responsePayload
	if strings.TrimSpace(b.token) == "" {
		return payload, errors.New("missing backend api token")
	}
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, reader)
	if err != nil {
		return payload, err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return payload, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		return payload, err
	}
	if resp.StatusCode >= 400 {
		return payload, errors.New(strings.TrimSpace(payload.Error))
	}
	return payload, nil
}
