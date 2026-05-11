package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cluster-enrollment-worker/internal/domain"
)

type aliyunCommandAttempt struct {
	Args   []string
	Source string
}

type ackClusterResponse struct {
	Name              string               `json:"name"`
	ClusterID         string               `json:"cluster_id"`
	ClusterType       string               `json:"cluster_type"`
	RegionID          string               `json:"region_id"`
	CurrentVersion    string               `json:"current_version"`
	InitVersion       string               `json:"init_version"`
	State             string               `json:"state"`
	ClusterSpec       string               `json:"cluster_spec"`
	Profile           string               `json:"profile"`
	PrivateZone       bool                 `json:"private_zone"`
	InitVersionAlias  string               `json:"init_version_alias"`
	APIServerEndpoint string               `json:"api_server_endpoint"`
	Endpoints         ackEndpointSection   `json:"endpoints"`
	Connections       ackConnectionSection `json:"connections"`
	MasterURL         string               `json:"master_url"`
	MetaData          any                  `json:"meta_data"`
	Tags              any                  `json:"tags"`
	VPCID             string               `json:"vpc_id"`
	VSwitchID         string               `json:"vswitch_id"`
	VSwitchIDs        []string             `json:"vswitch_ids"`
	WorkerVSwitchIDs  []string             `json:"worker_vswitch_ids"`
	PodVSwitchIDs     []string             `json:"pod_vswitch_ids"`
	ControlPlane      any                  `json:"control_plane_config"`
	Network           any                  `json:"network"`
	Raw               map[string]any       `json:"-"`
}

type ackEndpointSection struct {
	Private  string `json:"private_ip"`
	Internet string `json:"internet"`
	Internal string `json:"intranet"`
}

type ackConnectionSection struct {
	APIServerInternet string `json:"api_server_internet"`
	APIServerInternal string `json:"api_server_intranet"`
}

func enrollAlicloudCluster(ctx context.Context, claim *domain.Claim) domain.Result {
	clusterID := firstNonEmptyString(
		strings.TrimSpace(claim.CloudID),
		strings.TrimSpace(stringValue(claim.Metadata["cluster_id"])),
		strings.TrimSpace(stringValue(claim.Metadata["clusterId"])),
		strings.TrimSpace(claim.ResourceName),
	)
	if clusterID == "" {
		return domain.Result{Status: "error", ErrorMessage: "missing ACK cluster id"}
	}
	output, source, err := describeACKCluster(ctx, claim, clusterID)
	if err != nil {
		return domain.Result{Status: "error", ErrorMessage: err.Error()}
	}
	response, err := parseACKClusterResponse(output)
	if err != nil {
		return domain.Result{Status: "error", ErrorMessage: err.Error()}
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
			"region_id":    firstNonEmptyString(response.RegionID, claim.Region),
		},
	}
}

func describeACKCluster(ctx context.Context, claim *domain.Claim, clusterID string) ([]byte, string, error) {
	attempts := []aliyunCommandAttempt{
		{Args: []string{"cs", "GET", fmt.Sprintf("/clusters/%s", clusterID), "--region", claim.Region}, Source: "aliyun cs GET /clusters/{id}"},
		{Args: []string{"cs", "DescribeClusterDetail", "--ClusterId", clusterID, "--RegionId", claim.Region}, Source: "aliyun cs DescribeClusterDetail"},
		{Args: []string{"cs", "DescribeCluster", "--ClusterId", clusterID, "--RegionId", claim.Region}, Source: "aliyun cs DescribeCluster"},
	}
	var failures []string
	for _, attempt := range attempts {
		args := append([]string{}, attempt.Args...)
		if !containsArg(args, "--output") {
			args = append(args, "--output", "json")
		}
		cmd := exec.CommandContext(ctx, "aliyun", args...)
		cmd.Env = append(os.Environ(), flattenEnvMap(claim.Environment)...)
		output, err := cmd.CombinedOutput()
		if err == nil && strings.TrimSpace(string(output)) != "" {
			return output, attempt.Source, nil
		}
		message := strings.TrimSpace(string(output))
		if message == "" && err != nil {
			message = err.Error()
		}
		failures = append(failures, fmt.Sprintf("%s: %s", attempt.Source, message))
	}
	return nil, "", errors.New(strings.Join(failures, " | "))
}

func parseACKClusterResponse(output []byte) (ackClusterResponse, error) {
	var raw map[string]any
	if err := json.Unmarshal(output, &raw); err != nil {
		return ackClusterResponse{}, err
	}
	payload := unwrapACKPayload(raw)
	normalized, err := json.Marshal(payload)
	if err != nil {
		return ackClusterResponse{}, err
	}
	var response ackClusterResponse
	if err := json.Unmarshal(normalized, &response); err != nil {
		return ackClusterResponse{}, err
	}
	response.Raw = payload
	return response, nil
}

func unwrapACKPayload(raw map[string]any) map[string]any {
	for _, key := range []string{"cluster", "Cluster", "data", "Data"} {
		if nested, ok := raw[key].(map[string]any); ok && len(nested) > 0 {
			return nested
		}
	}
	return raw
}
