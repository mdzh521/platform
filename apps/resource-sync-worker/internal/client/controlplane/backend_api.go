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

	"resource-sync-worker/internal/domain"
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
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (b *BackendAPI) ClaimNextSync(ctx context.Context, workerName string) (*domain.Claim, error) {
	payload, err := b.request(ctx, http.MethodPost, "/api/v1/cloud/internal/resource-sync/claim", map[string]any{
		"worker_name": workerName,
	})
	if err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, nil
	}
	var claim domain.Claim
	raw, err := json.Marshal(payload.Data)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &claim); err != nil {
		return nil, err
	}
	if claim.ID == 0 {
		return nil, nil
	}
	return &claim, nil
}

func (b *BackendAPI) ReportSyncResult(ctx context.Context, jobID uint, result domain.Result) error {
	if jobID == 0 {
		return errors.New("missing job id")
	}
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/resource-sync/%d/result", jobID), map[string]any{
		"worker_name":      result.WorkerName,
		"status":           result.Status,
		"error_message":    result.ErrorMessage,
		"entries":          result.Entries,
		"synced_resources": result.SyncedResources,
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
		raw, err := json.Marshal(body)
		if err != nil {
			return payload, err
		}
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
		message := strings.TrimSpace(payload.Error)
		if message == "" {
			message = fmt.Sprintf("backend api returned %d", resp.StatusCode)
		}
		return payload, errors.New(message)
	}
	return payload, nil
}
