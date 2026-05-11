package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type JobExecution struct {
	JobID         uint
	Name          string
	Provider      string
	Action        string
	AccountID     uint
	BlueprintID   uint
	NetworkPlanID *uint
	BlueprintCode string
	TemplatePath  string
	Input         map[string]any
	Environment   map[string]string
	WorkspacePath string
}

type JobExecutionResult struct {
	Status      string
	LogExcerpt  string
	PlanSummary map[string]any
	Output      map[string]any
	StageLogs   []StageLog
}

type StageLog struct {
	Stage   string
	Level   string
	Message string
}

type ExecutionError struct {
	Stage   string
	Message string
}

func (e *ExecutionError) Error() string {
	return strings.TrimSpace(e.Message)
}

type TerraformExecutor struct {
	workspaces *WorkspaceManager
}

func NewTerraformExecutor(workspaces *WorkspaceManager) *TerraformExecutor {
	return &TerraformExecutor{workspaces: workspaces}
}

func (e *TerraformExecutor) Execute(ctx context.Context, job JobExecution) (JobExecutionResult, error) {
	result := JobExecutionResult{}
	workspace := e.workspaces.WorkspaceDir(job.JobID)
	if strings.TrimSpace(job.WorkspacePath) != "" {
		workspace = job.WorkspacePath
	}
	template := e.workspaces.ResolveTemplateDir(job.TemplatePath, job.BlueprintCode)
	if _, err := os.Stat(template); err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("init", fmt.Sprintf("template not found: %s", template), nil)
	}
	if job.Action == "destroy" {
		if err := e.workspaces.EnsureWorkspace(workspace); err != nil {
			_ = e.workspaces.RecordStatus(workspace, "failed")
			return result, stageError("init", "ensure destroy workspace failed", err)
		}
	} else {
		if err := e.workspaces.PrepareWorkspace(workspace); err != nil {
			_ = e.workspaces.RecordStatus(workspace, "failed")
			return result, stageError("init", "prepare workspace failed", err)
		}
	}
	if err := e.workspaces.CopyTemplateToWorkspace(template, workspace); err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("init", "copy template failed", err)
	}
	tfvarsPath := filepath.Join(workspace, "terraform.tfvars.json")
	inputPayload := prepareTerraformInput(job.Input)
	if err := writeJSONFile(tfvarsPath, inputPayload); err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("init", "write terraform.tfvars.json failed", err)
	}
	initOutput, err := e.runTerraform(ctx, workspace, job.Environment, "init", "-input=false", "-no-color")
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("init", "terraform init failed", err)
	}
	result.StageLogs = append(result.StageLogs, StageLog{
		Stage:   "init",
		Level:   "info",
		Message: stageMessage("terraform init completed", initOutput),
	})
	switch job.Action {
	case "apply":
		actionResult, err := e.executeApply(ctx, job, workspace, template, tfvarsPath, initOutput)
		result.StageLogs = append(result.StageLogs, actionResult.StageLogs...)
		if err == nil {
			result.Status = actionResult.Status
			result.LogExcerpt = actionResult.LogExcerpt
			result.PlanSummary = actionResult.PlanSummary
			result.Output = actionResult.Output
			_ = e.workspaces.RecordStatus(workspace, actionResult.Status)
		}
		return result, err
	case "destroy":
		actionResult, err := e.executeDestroy(ctx, job, workspace, template, tfvarsPath, initOutput)
		result.StageLogs = append(result.StageLogs, actionResult.StageLogs...)
		if err == nil {
			result.Status = actionResult.Status
			result.LogExcerpt = actionResult.LogExcerpt
			result.PlanSummary = actionResult.PlanSummary
			result.Output = actionResult.Output
			_ = e.workspaces.RecordStatus(workspace, actionResult.Status)
		}
		return result, err
	default:
		actionResult, err := e.executePlan(ctx, job, workspace, template, tfvarsPath, initOutput)
		result.StageLogs = append(result.StageLogs, actionResult.StageLogs...)
		if err == nil {
			result.Status = actionResult.Status
			result.LogExcerpt = actionResult.LogExcerpt
			result.PlanSummary = actionResult.PlanSummary
			result.Output = actionResult.Output
			_ = e.workspaces.RecordStatus(workspace, actionResult.Status)
		}
		return result, err
	}
}

func (e *TerraformExecutor) executePlan(ctx context.Context, job JobExecution, workspace, template, tfvarsPath, initOutput string) (JobExecutionResult, error) {
	result := JobExecutionResult{}
	planFile := filepath.Join(workspace, "tfplan")
	planOutput, err := e.runTerraform(ctx, workspace, job.Environment, "plan", "-input=false", "-no-color", "-lock=false", "-out", planFile)
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("plan", "terraform plan failed", err)
	}
	planSummary, showExcerpt, err := e.inspectPlan(ctx, workspace, job.Environment, planFile)
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("plan", "inspect terraform plan failed", err)
	}
	result = JobExecutionResult{
		Status:     "planned",
		LogExcerpt: fmt.Sprintf("terraform init+plan completed for %s (%s)", job.BlueprintCode, showExcerpt),
		PlanSummary: mergeMaps(planSummary, map[string]any{
			"workspace":   workspace,
			"template":    template,
			"action":      job.Action,
			"init_output": truncate(initOutput, 400),
			"plan_output": truncate(planOutput, 400),
		}),
		Output: map[string]any{
			"workspace":   workspace,
			"tfvars":      tfvarsPath,
			"plan_file":   planFile,
			"plan_output": truncate(planOutput, 400),
		},
		StageLogs: []StageLog{
			{
				Stage:   "plan",
				Level:   "info",
				Message: stageMessage(fmt.Sprintf("terraform plan completed for %s", job.BlueprintCode), planOutput),
			},
		},
	}
	return result, nil
}

func (e *TerraformExecutor) executeApply(ctx context.Context, job JobExecution, workspace, template, tfvarsPath, initOutput string) (JobExecutionResult, error) {
	result := JobExecutionResult{}
	applyOutput, err := e.runTerraform(ctx, workspace, job.Environment, "apply", "-input=false", "-auto-approve", "-no-color")
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("apply", "terraform apply failed", err)
	}
	stateSummary, stateExcerpt, err := e.inspectState(ctx, workspace, job.Environment)
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("apply", "inspect terraform state failed", err)
	}
	outputs, err := e.readOutputs(ctx, workspace, job.Environment)
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("apply", "terraform output failed", err)
	}
	result = JobExecutionResult{
		Status:     "succeeded",
		LogExcerpt: fmt.Sprintf("terraform apply completed for %s (%s)", job.BlueprintCode, stateExcerpt),
		PlanSummary: mergeMaps(stateSummary, map[string]any{
			"workspace":    workspace,
			"template":     template,
			"action":       job.Action,
			"init_output":  truncate(initOutput, 400),
			"apply_output": truncate(applyOutput, 400),
		}),
		Output: mergeMaps(map[string]any{
			"workspace":    workspace,
			"tfvars":       tfvarsPath,
			"apply_output": truncate(applyOutput, 400),
			"outputs":      outputs,
		}, map[string]any{
			"resource_list": stateSummary["resource_list"],
		}),
		StageLogs: []StageLog{
			{
				Stage:   "apply",
				Level:   "info",
				Message: stageMessage(fmt.Sprintf("terraform apply completed for %s", job.BlueprintCode), applyOutput),
			},
		},
	}
	return result, nil
}

func (e *TerraformExecutor) executeDestroy(ctx context.Context, job JobExecution, workspace, template, tfvarsPath, initOutput string) (JobExecutionResult, error) {
	result := JobExecutionResult{}
	destroyOutput, err := e.runTerraform(ctx, workspace, job.Environment, "destroy", "-input=false", "-auto-approve", "-no-color")
	if err != nil {
		_ = e.workspaces.RecordStatus(workspace, "failed")
		return result, stageError("destroy", "terraform destroy failed", err)
	}
	result = JobExecutionResult{
		Status:     "destroyed",
		LogExcerpt: fmt.Sprintf("terraform destroy completed for %s", job.BlueprintCode),
		PlanSummary: map[string]any{
			"workspace":      workspace,
			"template":       template,
			"action":         job.Action,
			"init_output":    truncate(initOutput, 400),
			"destroy_output": truncate(destroyOutput, 400),
			"resource_list":  []map[string]any{},
		},
		Output: map[string]any{
			"workspace":      workspace,
			"tfvars":         tfvarsPath,
			"destroy_output": truncate(destroyOutput, 400),
			"resource_list":  []map[string]any{},
		},
		StageLogs: []StageLog{
			{
				Stage:   "destroy",
				Level:   "info",
				Message: stageMessage(fmt.Sprintf("terraform destroy completed for %s", job.BlueprintCode), destroyOutput),
			},
		},
	}
	return result, nil
}

func prepareTerraformInput(input map[string]any) map[string]any {
	payload := map[string]any{}
	for _, key := range []string{"account_id", "blueprint_code", "network_plan_id", "action"} {
		if value, ok := input[key]; ok {
			payload[key] = value
		}
	}
	if provider, ok := input["provider"]; ok {
		payload["cloud_provider"] = provider
	}

	nestedInput := map[string]any{}
	if raw, ok := input["input"].(map[string]any); ok {
		for key, value := range raw {
			nestedInput[key] = value
		}
	}
	if nestedInput == nil {
		nestedInput = map[string]any{}
	}
	if _, ok := nestedInput["availability_zone_count"]; !ok {
		if value, exists := nestedInput["availability_zones"]; exists {
			nestedInput["availability_zone_count"] = value
		}
	}
	if tags, ok := normalizeTagMap(nestedInput["default_tags"]); ok {
		nestedInput["default_tags"] = tags
	}
	payload["input"] = nestedInput
	return payload
}

func normalizeTagMap(value any) (map[string]string, bool) {
	switch typed := value.(type) {
	case map[string]string:
		return typed, true
	case map[string]any:
		result := map[string]string{}
		for key, raw := range typed {
			text := strings.TrimSpace(fmt.Sprint(raw))
			if strings.TrimSpace(key) != "" && text != "" {
				result[key] = text
			}
		}
		return result, true
	case []any:
		return parseTagPairs(typed), true
	case []string:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return parseTagPairs(items), true
	default:
		return nil, false
	}
}

func parseTagPairs(items []any) map[string]string {
	result := map[string]string{}
	for _, item := range items {
		parts := strings.SplitN(strings.TrimSpace(fmt.Sprint(item)), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

func (e *TerraformExecutor) runTerraform(ctx context.Context, dir string, extraEnv map[string]string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "terraform", args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "TF_PLUGIN_CACHE_DIR="+e.workspaces.ProviderCacheDir())
	for key, value := range extraEnv {
		if strings.TrimSpace(key) == "" {
			continue
		}
		command.Env = append(command.Env, key+"="+value)
	}
	output, err := command.CombinedOutput()
	text := redactSensitiveText(strings.TrimSpace(string(output)))
	if err != nil {
		if text != "" {
			return text, errors.New(text)
		}
		return "", err
	}
	return text, nil
}

func (e *TerraformExecutor) inspectPlan(ctx context.Context, dir string, extraEnv map[string]string, planFile string) (map[string]any, string, error) {
	raw, err := e.runTerraform(ctx, dir, extraEnv, "show", "-json", planFile)
	if err != nil {
		return nil, "", fmt.Errorf("terraform show failed: %w", err)
	}
	raw = extractJSONObject(raw)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, "", err
	}
	resourceChanges := 0
	resourceList := make([]map[string]any, 0)
	if items, ok := payload["resource_changes"].([]any); ok {
		resourceChanges = len(items)
		for _, item := range items {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			change, _ := row["change"].(map[string]any)
			actions, _ := change["actions"].([]any)
			actionNames := make([]string, 0, len(actions))
			for _, action := range actions {
				actionNames = append(actionNames, strings.TrimSpace(fmt.Sprint(action)))
			}
			resourceList = append(resourceList, map[string]any{
				"address":  row["address"],
				"mode":     row["mode"],
				"type":     row["type"],
				"name":     row["name"],
				"provider": row["provider_name"],
				"actions":  actionNames,
			})
		}
	}
	outputChanges := 0
	if changes, ok := payload["output_changes"].(map[string]any); ok {
		outputChanges = len(changes)
	}
	summary := map[string]any{
		"format_version":    payload["format_version"],
		"terraform_version": payload["terraform_version"],
		"resource_changes":  resourceChanges,
		"output_changes":    outputChanges,
		"resource_list":     resourceList,
	}
	return summary, fmt.Sprintf("%d resource changes", resourceChanges), nil
}

func (e *TerraformExecutor) inspectState(ctx context.Context, dir string, extraEnv map[string]string) (map[string]any, string, error) {
	raw, err := e.runTerraform(ctx, dir, extraEnv, "show", "-json")
	if err != nil {
		return nil, "", fmt.Errorf("terraform state show failed: %w", err)
	}
	raw = extractJSONObject(raw)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, "", err
	}
	resourceList := collectStateResources(payload["values"])
	summary := map[string]any{
		"format_version":    payload["format_version"],
		"terraform_version": payload["terraform_version"],
		"resource_count":    len(resourceList),
		"resource_list":     resourceList,
	}
	return summary, fmt.Sprintf("%d managed resources in state", len(resourceList)), nil
}

func (e *TerraformExecutor) readOutputs(ctx context.Context, dir string, extraEnv map[string]string) (map[string]any, error) {
	raw, err := e.runTerraform(ctx, dir, extraEnv, "output", "-json")
	if err != nil {
		return nil, fmt.Errorf("terraform output failed: %w", err)
	}
	raw = extractJSONObject(raw)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func collectStateResources(values any) []map[string]any {
	root, ok := values.(map[string]any)
	if !ok {
		return nil
	}
	rootModule, _ := root["root_module"].(map[string]any)
	return appendModuleResources(rootModule, nil)
}

func appendModuleResources(module map[string]any, current []map[string]any) []map[string]any {
	if module == nil {
		return current
	}
	if resources, ok := module["resources"].([]any); ok {
		for _, item := range resources {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			metadata := extractResourceMetadata(row["values"])
			cloudID := firstNonEmpty(
				stringValue(metadata["cloud_id"]),
				stringValue(metadata["arn"]),
				stringValue(metadata["id"]),
			)
			current = append(current, map[string]any{
				"address":  row["address"],
				"mode":     row["mode"],
				"type":     row["type"],
				"name":     row["name"],
				"provider": row["provider_name"],
				"actions":  []string{"current"},
				"cloud_id": cloudID,
				"metadata": metadata,
			})
		}
	}
	if children, ok := module["child_modules"].([]any); ok {
		for _, child := range children {
			childModule, _ := child.(map[string]any)
			current = appendModuleResources(childModule, current)
		}
	}
	return current
}

func extractResourceMetadata(values any) map[string]any {
	resourceValues, ok := values.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	metadata := map[string]any{}
	copyOptionalField(metadata, resourceValues, "id")
	copyOptionalField(metadata, resourceValues, "arn")
	copyOptionalField(metadata, resourceValues, "name")
	copyOptionalField(metadata, resourceValues, "public_ip")
	copyOptionalField(metadata, resourceValues, "private_ip")
	copyOptionalField(metadata, resourceValues, "availability_zone")
	copyOptionalField(metadata, resourceValues, "cidr_block")
	copyOptionalField(metadata, resourceValues, "domain")
	copyOptionalField(metadata, resourceValues, "instance_type")
	copyOptionalField(metadata, resourceValues, "key_name")
	copyOptionalField(metadata, resourceValues, "vpc_id")
	copyOptionalField(metadata, resourceValues, "subnet_id")
	copyOptionalField(metadata, resourceValues, "version")
	copyOptionalField(metadata, resourceValues, "endpoint")
	copyOptionalField(metadata, resourceValues, "status")
	copyOptionalField(metadata, resourceValues, "security_groups")
	copyOptionalField(metadata, resourceValues, "vpc_security_group_ids")
	copyOptionalField(metadata, resourceValues, "associate_public_ip_address")
	copyNestedVPCConfig(metadata, resourceValues["vpc_config"])
	if tags, ok := resourceValues["tags"].(map[string]any); ok && len(tags) > 0 {
		metadata["tags"] = tags
	}
	if id := firstNonEmpty(stringValue(metadata["id"]), stringValue(metadata["arn"])); id != "" {
		metadata["cloud_id"] = id
	}
	return metadata
}

func copyOptionalField(target map[string]any, source map[string]any, key string) {
	if value, ok := source[key]; ok && value != nil && stringValue(value) != "" {
		target[key] = value
	}
}

func copyNestedVPCConfig(target map[string]any, value any) {
	row := firstMapFromNestedValue(value)
	if len(row) == 0 {
		return
	}
	copyOptionalField(target, row, "vpc_id")
	if subnetIDs := normalizeStringSlice(row["subnet_ids"]); len(subnetIDs) > 0 {
		target["subnet_refs"] = subnetIDs
	}
	copyOptionalField(target, row, "cluster_security_group_id")
	copyOptionalField(target, row, "endpoint_private_access")
	copyOptionalField(target, row, "endpoint_public_access")
}

func firstMapFromNestedValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case []any:
		for _, item := range typed {
			row, ok := item.(map[string]any)
			if ok && len(row) > 0 {
				return row
			}
		}
	}
	return map[string]any{}
}

func normalizeStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return compactStringSlice(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				result = append(result, text)
			}
		}
		return compactStringSlice(result)
	default:
		return nil
	}
}

func compactStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mergeMaps(left, right map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range left {
		result[key] = value
	}
	for key, value := range right {
		result[key] = value
	}
	return result
}

func stageMessage(prefix, output string) string {
	output = redactSensitiveText(strings.TrimSpace(output))
	if output == "" {
		return strings.TrimSpace(prefix)
	}
	return fmt.Sprintf("%s\n%s", strings.TrimSpace(prefix), truncate(output, 1200))
}

func stageError(stage, prefix string, err error) error {
	message := strings.TrimSpace(prefix)
	if detail := errorText(err); detail != "" {
		if message != "" {
			message += ": " + detail
		} else {
			message = detail
		}
	}
	return &ExecutionError{
		Stage:   strings.TrimSpace(stage),
		Message: message,
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return redactSensitiveText(strings.TrimSpace(err.Error()))
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func redactSensitiveText(value string) string {
	result := value
	envPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(AWS_ACCESS_KEY_ID=)([^\s]+)`),
		regexp.MustCompile(`(?i)(AWS_SECRET_ACCESS_KEY=)([^\s]+)`),
		regexp.MustCompile(`(?i)(ALICLOUD_ACCESS_KEY=)([^\s]+)`),
		regexp.MustCompile(`(?i)(ALICLOUD_SECRET_KEY=)([^\s]+)`),
	}
	for _, pattern := range envPatterns {
		result = pattern.ReplaceAllString(result, `${1}***REDACTED***`)
	}
	jsonLikePattern := regexp.MustCompile(`(?i)("?(access_key|secret_key|private_key|passphrase|token)"?\s*[:=]\s*"?)([^"\s,}]+)("?)`)
	return jsonLikePattern.ReplaceAllString(result, `${1}***REDACTED***${4}`)
}
