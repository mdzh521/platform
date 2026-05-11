package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"cluster-enrollment-worker/internal/domain"
)

var alicloudOpenAPIHTTPClient = &http.Client{Timeout: 20 * time.Second}

var alicloudOpenAPINow = func() time.Time {
	return time.Now().UTC()
}

var alicloudOpenAPINonce = func() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

var alicloudResolveCSOpenAPIEndpoint = func(region string) string {
	return fmt.Sprintf("https://cs.%s.aliyuncs.com", strings.TrimSpace(region))
}

func (AlicloudOpenAPIAdapter) Enroll(ctx context.Context, claim *domain.Claim) (domain.Result, error) {
	clusterID := firstNonEmptyString(
		strings.TrimSpace(claim.CloudID),
		strings.TrimSpace(stringValue(claim.Metadata["cluster_id"])),
		strings.TrimSpace(stringValue(claim.Metadata["clusterId"])),
	)
	if clusterID == "" {
		return domain.Result{}, errors.New("missing ACK cluster id")
	}
	accessKey := firstNonEmptyString(
		strings.TrimSpace(claim.Environment["ALICLOUD_ACCESS_KEY"]),
		strings.TrimSpace(claim.Environment["ALIBABA_CLOUD_ACCESS_KEY_ID"]),
	)
	secretKey := firstNonEmptyString(
		strings.TrimSpace(claim.Environment["ALICLOUD_SECRET_KEY"]),
		strings.TrimSpace(claim.Environment["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]),
	)
	region := firstNonEmptyString(
		strings.TrimSpace(claim.Region),
		strings.TrimSpace(claim.Environment["ALICLOUD_REGION"]),
		strings.TrimSpace(claim.Environment["ALICLOUD_REGION_ID"]),
		strings.TrimSpace(claim.Environment["ALIBABA_CLOUD_REGION_ID"]),
	)
	if accessKey == "" || secretKey == "" || region == "" {
		return domain.Result{}, errAdapterNotConfigured
	}

	body, source, err := describeACKClusterOpenAPI(ctx, accessKey, secretKey, region, clusterID)
	if err != nil {
		return domain.Result{}, err
	}
	response, err := parseACKClusterResponse(body)
	if err != nil {
		return domain.Result{}, err
	}

	endpoint := firstNonEmptyString(
		response.APIServerEndpoint,
		response.Endpoints.Internet,
		response.Connections.APIServerInternet,
		response.Endpoints.Internal,
		response.Connections.APIServerInternal,
		response.Endpoints.Private,
		response.MasterURL,
		stringValue(response.Raw["api_server_endpoint"]),
		stringValue(response.Raw["endpoint"]),
	)
	subnetRefs := firstNonEmptyStringSlice(
		response.VSwitchIDs,
		response.WorkerVSwitchIDs,
		response.PodVSwitchIDs,
		extractStringSlice(response.Raw["vswitch_ids"]),
		extractStringSlice(response.Raw["worker_vswitch_ids"]),
		extractStringSlice(response.Raw["pod_vswitch_ids"]),
		extractStringSlice(response.Raw["vswitchIds"]),
	)
	vpcID := firstNonEmptyString(
		response.VPCID,
		stringValue(response.Raw["vpc_id"]),
		stringValue(response.Raw["vpcId"]),
	)
	if response.VSwitchID != "" {
		subnetRefs = appendUniqueString(subnetRefs, response.VSwitchID)
	}
	return domain.Result{
		Status:         deriveEnrollmentStatus(response.State, endpoint),
		ClusterName:    firstNonEmptyString(response.Name, claim.ResourceName, clusterID),
		Endpoint:       endpoint,
		Version:        firstNonEmptyString(response.CurrentVersion, response.InitVersion, stringValue(response.Raw["version"])),
		VPCID:          vpcID,
		SubnetRefs:     subnetRefs,
		AccessMode:     deriveACKAccessMode(endpoint, response),
		ProviderStatus: response.State,
		Metadata: map[string]any{
			"source":       source,
			"cluster_id":   clusterID,
			"cluster_type": response.ClusterType,
			"profile":      response.Profile,
			"region_id":    firstNonEmptyString(response.RegionID, region),
		},
	}, nil
}

func describeACKClusterOpenAPI(ctx context.Context, accessKey, secretKey, region, clusterID string) ([]byte, string, error) {
	endpoint := strings.TrimRight(alicloudResolveCSOpenAPIEndpoint(region), "/")
	path := fmt.Sprintf("/clusters/%s", strings.TrimSpace(clusterID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+path, nil)
	if err != nil {
		return nil, "", err
	}
	signAliCloudROARequest(req, accessKey, secretKey)
	resp, err := alicloudOpenAPIHTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("ack openapi %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, "alicloud-openapi-clusters-get", nil
}

func signAliCloudROARequest(req *http.Request, accessKey, secretKey string) {
	now := alicloudOpenAPINow()
	headers := map[string]string{
		"accept":                  "application/json",
		"x-acs-date":              now.Format(time.RFC3339),
		"x-acs-signature-method":  "HMAC-SHA1",
		"x-acs-signature-nonce":   alicloudOpenAPINonce(),
		"x-acs-signature-version": "1.0",
		"x-acs-version":           "2015-12-15",
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	stringToSign := buildAliCloudROAStringToSign(req)
	mac := hmac.New(sha1.New, []byte(secretKey))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("Authorization", fmt.Sprintf("acs %s:%s", accessKey, signature))
}

func buildAliCloudROAStringToSign(req *http.Request) string {
	canonicalHeaders := canonicalizeAliCloudROAHeaders(req.Header)
	resource := req.URL.EscapedPath()
	if resource == "" {
		resource = "/"
	}
	query := req.URL.Query()
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for key := range query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			values := query[key]
			sort.Strings(values)
			for _, value := range values {
				if strings.TrimSpace(value) == "" {
					parts = append(parts, key)
					continue
				}
				parts = append(parts, key+"="+value)
			}
		}
		resource += "?" + strings.Join(parts, "&")
	}
	return strings.Join([]string{
		req.Method,
		req.Header.Get("Accept"),
		req.Header.Get("Content-MD5"),
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		canonicalHeaders + resource,
	}, "\n")
}

func canonicalizeAliCloudROAHeaders(headers http.Header) string {
	keys := make([]string, 0)
	values := map[string]string{}
	for key, item := range headers {
		lower := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lower, "x-acs-") {
			continue
		}
		keys = append(keys, lower)
		values[lower] = strings.Join(item, ",")
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+":"+strings.TrimSpace(values[key]))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
