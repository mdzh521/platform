package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"cluster-addon-worker/internal/domain"
)

func TestBackendAPIClaimReturnsDomainClaim(t *testing.T) {
	t.Parallel()

	api := NewBackendAPI("http://backend.example", "token")
	api.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/cloud/internal/cluster-addons/claim" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		body, err := json.Marshal(map[string]any{
			"data": map[string]any{
				"execution_id": 42,
				"cluster_name": "demo",
				"addon_key":    "aws_load_balancer_controller",
				"install_mode": "platform_managed_helm",
			},
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Header:     make(http.Header),
		}, nil
	})}
	claim, err := api.Claim(context.Background(), "worker-a")
	if err != nil {
		t.Fatalf("Claim returned error: %v", err)
	}
	if claim == nil {
		t.Fatal("Claim returned nil")
	}
	if claim.ExecutionID != 42 {
		t.Fatalf("unexpected execution id: %d", claim.ExecutionID)
	}
	if claim.ClusterName != "demo" {
		t.Fatalf("unexpected cluster name: %s", claim.ClusterName)
	}
}

func TestBackendAPIReportSendsDomainResult(t *testing.T) {
	t.Parallel()

	var got map[string]any
	api := NewBackendAPI("http://backend.example", "token")
	api.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/cloud/internal/cluster-addons/7/result" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		body, err := json.Marshal(map[string]any{"data": map[string]any{}})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Header:     make(http.Header),
		}, nil
	})}
	err := api.Report(context.Background(), 7, domain.Result{
		WorkerName:   "worker-a",
		Status:       "succeeded",
		ErrorMessage: "",
		Result: map[string]any{
			"release": "alb",
		},
	})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if got["worker_name"] != "worker-a" {
		t.Fatalf("unexpected worker_name: %#v", got["worker_name"])
	}
	if got["status"] != "succeeded" {
		t.Fatalf("unexpected status: %#v", got["status"])
	}
	result, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected result payload: %#v", got["result"])
	}
	if result["release"] != "alb" {
		t.Fatalf("unexpected result.release: %#v", result["release"])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
