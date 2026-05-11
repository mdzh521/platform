package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"cluster-enrollment-worker/internal/domain"
)

func TestAliCloudOpenAPIAdapterMapsACKResponse(t *testing.T) {
	previousClient := alicloudOpenAPIHTTPClient
	previousNow := alicloudOpenAPINow
	previousNonce := alicloudOpenAPINonce
	previousEndpoint := alicloudResolveCSOpenAPIEndpoint
	t.Cleanup(func() {
		alicloudOpenAPIHTTPClient = previousClient
		alicloudOpenAPINow = previousNow
		alicloudOpenAPINonce = previousNonce
		alicloudResolveCSOpenAPIEndpoint = previousEndpoint
	})

	alicloudOpenAPIHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/clusters/c-demo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "acs test-ak:") {
			t.Fatalf("missing authorization header: %q", r.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"cluster_id":"c-demo","name":"ack-demo","state":"running","current_version":"1.33.3-aliyun.1","api_server_endpoint":"https://ack.demo.local","vpc_id":"vpc-123","vswitch_ids":["vsw-a","vsw-b"]}`,
			)),
		}, nil
	})}
	alicloudOpenAPINow = func() time.Time { return time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC) }
	alicloudOpenAPINonce = func() string { return "nonce-1" }
	alicloudResolveCSOpenAPIEndpoint = func(string) string { return "https://cs.cn-shenzhen.aliyuncs.com" }

	result, err := (AlicloudOpenAPIAdapter{}).Enroll(context.Background(), &domain.Claim{
		Provider: "alicloud",
		CloudID:  "c-demo",
		Region:   "cn-shenzhen",
		Environment: map[string]string{
			"ALICLOUD_ACCESS_KEY": "test-ak",
			"ALICLOUD_SECRET_KEY": "test-sk",
			"ALICLOUD_REGION":     "cn-shenzhen",
		},
	})
	if err != nil {
		t.Fatalf("expected adapter success, got error: %v", err)
	}
	if result.Status != "enrolled" {
		t.Fatalf("expected enrolled status, got %q", result.Status)
	}
	if result.ClusterName != "ack-demo" {
		t.Fatalf("unexpected cluster name: %q", result.ClusterName)
	}
	if result.Endpoint != "https://ack.demo.local" {
		t.Fatalf("unexpected endpoint: %q", result.Endpoint)
	}
	if result.VPCID != "vpc-123" {
		t.Fatalf("unexpected vpc id: %q", result.VPCID)
	}
	if len(result.SubnetRefs) != 2 || result.SubnetRefs[0] != "vsw-a" || result.SubnetRefs[1] != "vsw-b" {
		t.Fatalf("unexpected subnet refs: %#v", result.SubnetRefs)
	}
	if result.Metadata["source"] != "alicloud-openapi-clusters-get" {
		t.Fatalf("unexpected metadata source: %#v", result.Metadata["source"])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestAliCloudOpenAPIAdapterRequiresCredentials(t *testing.T) {
	if _, err := (AlicloudOpenAPIAdapter{}).Enroll(context.Background(), &domain.Claim{
		Provider: "alicloud",
		CloudID:  "c-demo",
		Region:   "cn-shenzhen",
	}); err == nil {
		t.Fatal("expected missing credentials error")
	}
}
