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

	"terraform-runner/internal/domain"
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
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (b *BackendAPI) ClaimNextJob(ctx context.Context, runnerName string) (*domain.JobClaim, error) {
	payload, err := b.request(ctx, http.MethodPost, "/api/v1/cloud/internal/jobs/claim", map[string]any{
		"runner_name": runnerName,
	})
	if err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, nil
	}
	claim := domain.JobClaim{
		JobID:         uintValue(payload.Data["id"]),
		Name:          stringValue(payload.Data["name"]),
		Provider:      stringValue(payload.Data["provider"]),
		Action:        stringValue(payload.Data["action"]),
		AccountID:     uintValue(payload.Data["account_id"]),
		BlueprintID:   uintValue(payload.Data["blueprint_id"]),
		NetworkPlanID: optionalUintValue(payload.Data["network_plan_id"]),
		BlueprintCode: stringValue(payload.Data["blueprint_code"]),
		TemplatePath:  stringValue(payload.Data["template_path"]),
		Input:         mapValue(payload.Data["input"]),
		Environment:   stringMapValue(payload.Data["environment"]),
		WorkspacePath: stringValue(payload.Data["workspace_path"]),
	}
	if claim.JobID == 0 {
		return nil, nil
	}
	return &claim, nil
}

func (b *BackendAPI) Heartbeat(ctx context.Context, jobID uint, runnerName, status string) error {
	if jobID == 0 {
		return errors.New("missing job id")
	}
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/jobs/%d/heartbeat", jobID), map[string]any{
		"runner_name": runnerName,
		"status":      status,
	})
	return err
}

func (b *BackendAPI) AppendLog(ctx context.Context, jobID uint, runnerName, stage, level, message string) error {
	if jobID == 0 {
		return errors.New("missing job id")
	}
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/jobs/%d/logs", jobID), map[string]any{
		"runner_name": runnerName,
		"stage":       stage,
		"level":       level,
		"message":     message,
	})
	return err
}

func (b *BackendAPI) ReportResult(ctx context.Context, runnerName string, result domain.JobResult) error {
	if result.JobID == 0 {
		return errors.New("missing job id")
	}
	_, err := b.request(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud/internal/jobs/%d/result", result.JobID), map[string]any{
		"runner_name":   runnerName,
		"status":        result.Status,
		"log_excerpt":   result.LogExcerpt,
		"output":        result.Output,
		"plan_summary":  result.PlanSummary,
		"error_message": result.ErrorMessage,
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

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func uintValue(value any) uint {
	switch typed := value.(type) {
	case float64:
		return uint(typed)
	case int:
		return uint(typed)
	case int64:
		return uint(typed)
	case json.Number:
		raw, _ := typed.Int64()
		return uint(raw)
	default:
		return 0
	}
}

func optionalUintValue(value any) *uint {
	parsed := uintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func mapValue(value any) map[string]any {
	data, ok := value.(map[string]any)
	if ok {
		return data
	}
	return map[string]any{}
}

func stringMapValue(value any) map[string]string {
	items, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	result := make(map[string]string, len(items))
	for key, raw := range items {
		result[key] = stringValue(raw)
	}
	return result
}
