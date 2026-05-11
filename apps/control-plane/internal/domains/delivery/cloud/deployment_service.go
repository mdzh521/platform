package cloud

import (
	"backend-center/internal/domains/shared/scopeutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeploymentService struct {
	base *baseService
}

const cloudJobHeartbeatTimeout = 3 * time.Minute

func (s *DeploymentService) RecoverTimedOutJobs() error {
	cutoff := time.Now().Add(-cloudJobHeartbeatTimeout)
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var jobs []DeploymentJob
		if err := tx.
			Where("status IN ?", []string{"claimed", "planning", "applying", "destroying"}).
			Where("ended_at IS NULL").
			Where("heartbeat_at IS NULL OR heartbeat_at < ?", cutoff).
			Find(&jobs).Error; err != nil {
			return err
		}
		for _, job := range jobs {
			now := time.Now()
			job.Status = "failed"
			job.EndedAt = &now
			job.ErrorMessage = fmt.Sprintf("任务心跳已超时，超过 %s 未收到 runner 更新", cloudJobHeartbeatTimeout)
			job.LogExcerpt = "runner 心跳超时，任务已自动标记失败。"
			if err := tx.Save(&job).Error; err != nil {
				return err
			}
			if err := tx.Create(&DeploymentJobLog{
				JobID:     job.ID,
				Stage:     "result",
				Level:     "error",
				Message:   job.LogExcerpt,
				CreatedAt: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DeploymentService) Get(id uint) (DeploymentJobView, error) {
	_ = s.RecoverTimedOutJobs()
	var job DeploymentJob
	if err := s.base.db.First(&job, id).Error; err != nil {
		return DeploymentJobView{}, err
	}
	accountNames, _ := loadNameMap[CloudAccount](s.base.db)
	blueprintNames, _ := loadNameMap[DeploymentBlueprint](s.base.db)
	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	resourceCounts, _ := loadJobResourceCounts(s.base.db)
	view := jobToView(job, accountNames[job.AccountID], blueprintNames[job.BlueprintID], networkLabel(job.NetworkPlanID, networkNames), resourceCounts[job.ID])
	view.Resources, _ = loadJobResources(s.base.db, job.ID)
	return view, nil
}

func (s *DeploymentService) ListLogs(jobID uint) ([]DeploymentJobLogView, error) {
	_ = s.RecoverTimedOutJobs()
	var items []DeploymentJobLog
	if err := s.base.db.Where("job_id = ?", jobID).Order("created_at desc, id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]DeploymentJobLogView, 0, len(items))
	for _, item := range items {
		result = append(result, DeploymentJobLogView{
			ID:        item.ID,
			JobID:     item.JobID,
			Stage:     item.Stage,
			Level:     item.Level,
			Message:   item.Message,
			CreatedAt: item.CreatedAt,
		})
	}
	return result, nil
}

func (s *DeploymentService) List() ([]DeploymentJobView, error) {
	_ = s.RecoverTimedOutJobs()
	var jobs []DeploymentJob
	if err := s.base.db.Order("created_at desc").Find(&jobs).Error; err != nil {
		return nil, err
	}

	accountNames, err := loadNameMap[CloudAccount](s.base.db)
	if err != nil {
		return nil, err
	}
	blueprintNames, err := loadNameMap[DeploymentBlueprint](s.base.db)
	if err != nil {
		return nil, err
	}
	networkNames, err := loadNameMap[NetworkPlan](s.base.db)
	if err != nil {
		return nil, err
	}
	resourceCounts, err := loadJobResourceCounts(s.base.db)
	if err != nil {
		return nil, err
	}

	result := make([]DeploymentJobView, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, jobToView(
			job,
			accountNames[job.AccountID],
			blueprintNames[job.BlueprintID],
			networkLabel(job.NetworkPlanID, networkNames),
			resourceCounts[job.ID],
		))
	}
	return result, nil
}

func (s *DeploymentService) Create(input DeploymentJobInput) (DeploymentJobView, error) {
	_ = s.RecoverTimedOutJobs()
	if strings.TrimSpace(input.Name) == "" {
		return DeploymentJobView{}, errors.New("任务名称不能为空")
	}
	if input.AccountID == 0 {
		return DeploymentJobView{}, errors.New("请选择云账号")
	}
	if input.BlueprintID == 0 {
		return DeploymentJobView{}, errors.New("请选择蓝图")
	}

	action := normalizeField(input.Action, "apply")
	if action != "plan" && action != "apply" && action != "destroy" {
		return DeploymentJobView{}, errors.New("不支持的任务动作")
	}
	if (action == "apply" || action == "destroy") && !input.Confirmed {
		return DeploymentJobView{}, errors.New("执行 apply 或 destroy 前必须完成明确确认")
	}

	var blueprint DeploymentBlueprint
	if err := s.base.db.First(&blueprint, input.BlueprintID).Error; err != nil {
		return DeploymentJobView{}, errors.New("蓝图不存在")
	}
	var account CloudAccount
	if err := s.base.db.First(&account, input.AccountID).Error; err != nil {
		return DeploymentJobView{}, errors.New("云账号不存在")
	}

	if input.Provider == "" {
		input.Provider = blueprint.Provider
	}
	if err := scopeutil.Validate(s.base.db, scopeutil.ScopeInput{
		AccountID:           &input.AccountID,
		ProjectID:           input.ProjectID,
		EnvironmentID:       input.EnvironmentID,
		StackID:             input.StackID,
		FoundationNetworkID: input.NetworkPlanID,
		Provider:            input.Provider,
	}); err != nil {
		return DeploymentJobView{}, err
	}
	if action == "apply" && !blueprintSupportsApply(blueprint.Capability) {
		return DeploymentJobView{}, fmt.Errorf("蓝图 %s 当前成熟度为 %s，不支持真实 apply", blueprint.Name, normalizeBlueprintMaturity(blueprint.Maturity))
	}
	if action == "destroy" && !blueprintSupportsDestroy(blueprint.Capability) {
		return DeploymentJobView{}, fmt.Errorf("蓝图 %s 当前成熟度为 %s，不支持真实 destroy", blueprint.Name, normalizeBlueprintMaturity(blueprint.Maturity))
	}

	job := DeploymentJob{
		Name:          strings.TrimSpace(input.Name),
		Provider:      normalizeProvider(input.Provider),
		AccountID:     input.AccountID,
		BlueprintID:   input.BlueprintID,
		NetworkPlanID: input.NetworkPlanID,
		ProjectID:     input.ProjectID,
		EnvironmentID: input.EnvironmentID,
		StackID:       input.StackID,
		JobType:       normalizeJobType(input.JobType),
		Status:        "queued",
		Action:        action,
		InputJSON: mustJSONString(map[string]any{
			"provider":        input.Provider,
			"action":          action,
			"account_id":      input.AccountID,
			"blueprint_code":  blueprint.Code,
			"network_plan_id": input.NetworkPlanID,
			"project_id":      input.ProjectID,
			"environment_id":  input.EnvironmentID,
			"stack_id":        input.StackID,
			"job_type":        normalizeJobType(input.JobType),
			"input":           normalizeDeploymentInput(blueprint.Code, input.Input),
		}),
		LogExcerpt: "部署任务已创建，等待 Terraform runner 接管执行。",
	}
	if action == "destroy" && input.SourceJobID != nil && *input.SourceJobID > 0 {
		var source DeploymentJob
		if err := s.base.db.First(&source, *input.SourceJobID).Error; err != nil {
			return DeploymentJobView{}, errors.New("销毁来源任务不存在")
		}
		job.RetryOfJobID = &source.ID
		job.WorkspacePath = source.WorkspacePath
	}
	if err := s.base.db.Create(&job).Error; err != nil {
		return DeploymentJobView{}, err
	}
	if job.NetworkPlanID != nil || job.ProjectID != nil || job.EnvironmentID != nil {
		var plan *NetworkPlan
		if job.NetworkPlanID != nil && *job.NetworkPlanID > 0 {
			var loadedPlan NetworkPlan
			if err := s.base.db.First(&loadedPlan, *job.NetworkPlanID).Error; err == nil {
				plan = &loadedPlan
			}
		}
		if err := bindJobScope(s.base.db, &job, &blueprint, plan); err != nil {
			return DeploymentJobView{}, err
		}
	}

	if s.base.queue != nil {
		_ = s.base.queue.Publish(context.Background(), "cloud.deployment.request", mustJSON(map[string]any{
			"job_id":         job.ID,
			"provider":       job.Provider,
			"action":         action,
			"blueprint_code": blueprint.Code,
		}))
	}

	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	return jobToView(job, account.Name, blueprint.Name, networkLabel(job.NetworkPlanID, networkNames), 0), nil
}

func normalizeBlueprintMaturity(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "apply ready":
		return "apply ready"
	case "plan only":
		return "plan only"
	default:
		return "experimental"
	}
}

func normalizeJobType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "provision", "operation", "recovery":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "provision"
	}
}

func blueprintSupportsApply(capability string) bool {
	switch strings.TrimSpace(strings.ToLower(capability)) {
	case "apply_destroy_ready", "apply_ready", "experimental_apply":
		return true
	default:
		return false
	}
}

func blueprintSupportsDestroy(capability string) bool {
	switch strings.TrimSpace(strings.ToLower(capability)) {
	case "apply_destroy_ready":
		return true
	default:
		return false
	}
}

func (s *DeploymentService) ClaimNext(ctx context.Context, input JobClaimRequest) (*JobClaimView, error) {
	_ = s.RecoverTimedOutJobs()
	runnerName := strings.TrimSpace(input.RunnerName)
	if runnerName == "" {
		return nil, errors.New("runner_name 不能为空")
	}

	var (
		job       DeploymentJob
		blueprint DeploymentBlueprint
		account   CloudAccount
		now       = time.Now()
	)

	err := s.base.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", "queued").
			Order("created_at asc").
			First(&job).Error; err != nil {
			return err
		}
		if err := tx.First(&blueprint, job.BlueprintID).Error; err != nil {
			return err
		}
		if err := tx.First(&account, job.AccountID).Error; err != nil {
			return err
		}
		job.Status = "claimed"
		job.RunnerName = runnerName
		job.StartedAt = &now
		job.HeartbeatAt = &now
		if strings.TrimSpace(job.WorkspacePath) == "" {
			job.WorkspacePath = buildWorkspacePath(job.ID)
		}
		job.LogExcerpt = "任务已被 terraform-runner 抢占。"
		return tx.Save(&job).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var inputPayload map[string]any
	_ = json.Unmarshal([]byte(job.InputJSON), &inputPayload)
	if inputPayload == nil {
		inputPayload = map[string]any{}
	}
	topology := loadNetworkTopology(s.base.db, job.NetworkPlanID)
	inputPayload = enrichExecutionInput(inputPayload, account, topology)
	inputPayload = s.resolveFoundationExecutionInput(blueprint, account, job.NetworkPlanID, inputPayload, topology)
	executionEnv, err := s.resolveExecutionEnvironment(account, job.Provider, executionRegion(inputPayload, account.Region))
	if err != nil {
		return nil, err
	}
	return &JobClaimView{
		ID:            job.ID,
		Name:          job.Name,
		Provider:      job.Provider,
		Action:        job.Action,
		AccountID:     job.AccountID,
		BlueprintID:   job.BlueprintID,
		NetworkPlanID: job.NetworkPlanID,
		BlueprintCode: blueprint.Code,
		TemplatePath:  blueprint.TemplatePath,
		Input:         inputPayload,
		Environment:   executionEnv,
		WorkspacePath: job.WorkspacePath,
		ClaimedAt:     job.StartedAt,
	}, nil
}

func (s *DeploymentService) Heartbeat(jobID uint, input JobHeartbeatInput) error {
	runnerName := strings.TrimSpace(input.RunnerName)
	if runnerName == "" {
		return errors.New("runner_name 不能为空")
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var job DeploymentJob
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		if job.Status == "cancelled" {
			return nil
		}
		if strings.TrimSpace(job.RunnerName) != "" && job.RunnerName != runnerName {
			return errors.New("runner 不匹配")
		}
		now := time.Now()
		job.RunnerName = runnerName
		job.HeartbeatAt = &now
		if status := normalizeHeartbeatStatus(input.Status); status != "" {
			job.Status = status
		}
		return tx.Save(&job).Error
	})
}

func (s *DeploymentService) Cancel(jobID uint) (DeploymentJobView, error) {
	_ = s.RecoverTimedOutJobs()
	var job DeploymentJob
	now := time.Now()
	if err := s.base.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		switch job.Status {
		case "queued", "claimed", "planning":
		default:
			return errors.New("当前任务阶段不允许取消")
		}
		job.Status = "cancelled"
		job.EndedAt = &now
		job.LogExcerpt = "任务已被用户取消。"
		job.ErrorMessage = ""
		if err := tx.Save(&job).Error; err != nil {
			return err
		}
		return tx.Create(&DeploymentJobLog{
			JobID:     job.ID,
			Stage:     "result",
			Level:     "warn",
			Message:   job.LogExcerpt,
			CreatedAt: now,
		}).Error
	}); err != nil {
		return DeploymentJobView{}, err
	}
	accountNames, _ := loadNameMap[CloudAccount](s.base.db)
	blueprintNames, _ := loadNameMap[DeploymentBlueprint](s.base.db)
	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	resourceCounts, _ := loadJobResourceCounts(s.base.db)
	view := jobToView(job, accountNames[job.AccountID], blueprintNames[job.BlueprintID], networkLabel(job.NetworkPlanID, networkNames), resourceCounts[job.ID])
	view.Resources, _ = loadJobResources(s.base.db, job.ID)
	return view, nil
}

func (s *DeploymentService) Retry(jobID uint, input RetryJobInput) (DeploymentJobView, error) {
	_ = s.RecoverTimedOutJobs()
	var source DeploymentJob
	if err := s.base.db.First(&source, jobID).Error; err != nil {
		return DeploymentJobView{}, err
	}
	var blueprint DeploymentBlueprint
	if err := s.base.db.First(&blueprint, source.BlueprintID).Error; err != nil {
		return DeploymentJobView{}, errors.New("蓝图不存在")
	}
	desiredAction := normalizeField(input.Action, source.Action)
	if desiredAction != "plan" && desiredAction != "apply" && desiredAction != "destroy" {
		return DeploymentJobView{}, errors.New("不支持的任务动作")
	}
	if err := validateJobRerun(source, blueprint, desiredAction, input.Confirmed); err != nil {
		return DeploymentJobView{}, err
	}

	payload := parseInputJSONMap(source.InputJSON)
	payload["action"] = desiredAction
	payload["provider"] = source.Provider
	payload["account_id"] = source.AccountID
	payload["blueprint_code"] = blueprint.Code
	payload["network_plan_id"] = source.NetworkPlanID
	normalizedInput := payload["input"]
	if input.Input != nil {
		normalizedInput = input.Input
	}
	payload["input"] = normalizeDeploymentInput(blueprint.Code, normalizedInput)

	now := time.Now()
	retryMessage := buildRerunLogExcerpt(source, desiredAction)
	if err := s.base.db.Transaction(func(tx *gorm.DB) error {
		source.Name = firstNonEmpty(strings.TrimSpace(input.Name), source.Name)
		source.Action = desiredAction
		source.Status = "queued"
		source.RunnerName = ""
		source.InputJSON = mustJSONString(payload)
		source.PlanSummaryJSON = ""
		source.OutputJSON = ""
		source.ErrorMessage = ""
		source.LogExcerpt = retryMessage
		source.ResourceSyncStatus = "pending"
		source.ResourceSyncWorker = ""
		source.ResourceSyncError = ""
		source.ResourceSyncedAt = nil
		source.StartedAt = nil
		source.EndedAt = nil
		source.HeartbeatAt = nil
		if err := tx.Save(&source).Error; err != nil {
			return err
		}
		return tx.Create(&DeploymentJobLog{
			JobID:     source.ID,
			Stage:     "result",
			Level:     "warn",
			Message:   retryMessage,
			CreatedAt: now,
		}).Error
	}); err != nil {
		return DeploymentJobView{}, err
	}
	if s.base.queue != nil {
		_ = s.base.queue.Publish(context.Background(), "cloud.deployment.request", mustJSON(map[string]any{
			"job_id":         source.ID,
			"provider":       source.Provider,
			"action":         source.Action,
			"blueprint_code": blueprint.Code,
		}))
	}
	accountNames, _ := loadNameMap[CloudAccount](s.base.db)
	blueprintNames, _ := loadNameMap[DeploymentBlueprint](s.base.db)
	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	resourceCounts, _ := loadJobResourceCounts(s.base.db)
	view := jobToView(source, accountNames[source.AccountID], blueprintNames[source.BlueprintID], networkLabel(source.NetworkPlanID, networkNames), resourceCounts[source.ID])
	return view, nil
}

func validateJobRerun(source DeploymentJob, blueprint DeploymentBlueprint, action string, confirmed bool) error {
	status := strings.TrimSpace(strings.ToLower(source.Status))
	switch status {
	case "failed", "cancelled":
	case "succeeded":
		if strings.TrimSpace(strings.ToLower(source.Action)) != "apply" {
			return errors.New("当前任务状态不支持原地重跑")
		}
		if action != "apply" && action != "destroy" {
			return errors.New("已成功的任务当前只支持重新 apply 或执行清理")
		}
	default:
		return errors.New("当前任务状态不支持重试")
	}
	if action == "apply" && !blueprintSupportsApply(blueprint.Capability) {
		return fmt.Errorf("蓝图 %s 当前成熟度为 %s，不支持真实 apply", blueprint.Name, normalizeBlueprintMaturity(blueprint.Maturity))
	}
	if action == "destroy" && !blueprintSupportsDestroy(blueprint.Capability) {
		return fmt.Errorf("蓝图 %s 当前成熟度为 %s，不支持真实 destroy", blueprint.Name, normalizeBlueprintMaturity(blueprint.Maturity))
	}
	if (action == "apply" || action == "destroy") && !confirmed {
		return errors.New("执行 apply 或 destroy 前必须完成明确确认")
	}
	return nil
}

func buildRerunLogExcerpt(source DeploymentJob, action string) string {
	switch action {
	case "destroy":
		return fmt.Sprintf("任务 #%d 已切换为清理流程，等待 Terraform runner 执行 destroy。", source.ID)
	case "apply":
		if strings.TrimSpace(strings.ToLower(source.Status)) == "succeeded" {
			return fmt.Sprintf("任务 #%d 已更新配置并重新排队，等待 Terraform runner 再次 apply。", source.ID)
		}
		return fmt.Sprintf("任务 #%d 已重新排队，等待 Terraform runner 再次 apply。", source.ID)
	default:
		return fmt.Sprintf("任务 #%d 已重新排队，等待 Terraform runner 再次执行。", source.ID)
	}
}

func (s *DeploymentService) AppendLog(jobID uint, input JobLogInput) error {
	runnerName := strings.TrimSpace(input.RunnerName)
	if runnerName == "" {
		return errors.New("runner_name 不能为空")
	}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return errors.New("日志内容不能为空")
	}
	level := normalizeField(input.Level, "info")
	stage := normalizeField(input.Stage, "plan")
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var job DeploymentJob
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		if job.Status == "cancelled" {
			return nil
		}
		if strings.TrimSpace(job.RunnerName) != "" && job.RunnerName != runnerName {
			return errors.New("runner 不匹配")
		}
		now := time.Now()
		job.RunnerName = runnerName
		job.HeartbeatAt = &now
		job.LogExcerpt = message
		if err := tx.Create(&DeploymentJobLog{
			JobID:     jobID,
			Stage:     stage,
			Level:     level,
			Message:   message,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Save(&job).Error
	})
}

func (s *DeploymentService) ReportResult(jobID uint, input JobResultInput) (DeploymentJobView, error) {
	runnerName := strings.TrimSpace(input.RunnerName)
	if runnerName == "" {
		return DeploymentJobView{}, errors.New("runner_name 不能为空")
	}
	status := normalizeResultStatus(input.Status)
	if status == "" {
		return DeploymentJobView{}, errors.New("不支持的任务状态")
	}

	var job DeploymentJob
	now := time.Now()
	if err := s.base.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		if job.Status == "cancelled" {
			return nil
		}
		if strings.TrimSpace(job.RunnerName) != "" && job.RunnerName != runnerName {
			return errors.New("runner 不匹配")
		}
		job.RunnerName = runnerName
		job.Status = status
		job.HeartbeatAt = &now
		job.EndedAt = &now
		job.LogExcerpt = strings.TrimSpace(input.LogExcerpt)
		job.ErrorMessage = strings.TrimSpace(input.ErrorMessage)
		if input.Output != nil {
			job.OutputJSON = mustJSONString(input.Output)
		}
		if input.PlanSummary != nil {
			job.PlanSummaryJSON = mustJSONString(input.PlanSummary)
		}
		job.ResourceSyncStatus = "pending"
		job.ResourceSyncWorker = ""
		job.ResourceSyncError = ""
		job.ResourceSyncedAt = nil
		if job.LogExcerpt == "" {
			if status == "succeeded" {
				job.LogExcerpt = "任务已完成。"
			} else {
				job.LogExcerpt = normalizeField(job.ErrorMessage, "任务执行失败")
			}
		}
		if err := tx.Save(&job).Error; err != nil {
			return err
		}
		if err := syncJobResources(tx, job.ID, input.PlanSummary, input.Output); err != nil {
			return err
		}
		return tx.Create(&DeploymentJobLog{
			JobID:     job.ID,
			Stage:     "result",
			Level:     resultLevel(status),
			Message:   job.LogExcerpt,
			CreatedAt: now,
		}).Error
	}); err != nil {
		return DeploymentJobView{}, err
	}

	accountNames, _ := loadNameMap[CloudAccount](s.base.db)
	blueprintNames, _ := loadNameMap[DeploymentBlueprint](s.base.db)
	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	resourceCounts, _ := loadJobResourceCounts(s.base.db)
	return jobToView(job, accountNames[job.AccountID], blueprintNames[job.BlueprintID], networkLabel(job.NetworkPlanID, networkNames), resourceCounts[job.ID]), nil
}

func syncJobResources(tx *gorm.DB, jobID uint, planSummary, output map[string]any) error {
	items := extractResourceList(planSummary)
	if len(items) == 0 {
		items = extractResourceList(output)
	}
	if err := tx.Where("job_id = ?", jobID).Delete(&DeploymentResource{}).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	resources := make([]DeploymentResource, 0, len(items))
	for _, item := range items {
		resourceType := strings.TrimSpace(fmt.Sprint(item["type"]))
		resourceName := strings.TrimSpace(fmt.Sprint(item["address"]))
		if resourceName == "" {
			resourceName = strings.TrimSpace(fmt.Sprint(item["name"]))
		}
		if resourceType == "" || resourceName == "" {
			continue
		}
		resources = append(resources, DeploymentResource{
			JobID:        jobID,
			ResourceType: resourceType,
			ResourceName: resourceName,
			CloudID:      optionalString(item["cloud_id"]),
			MetadataJSON: mustJSONString(item),
		})
	}
	if len(resources) == 0 {
		return nil
	}
	return tx.Create(&resources).Error
}

func extractResourceList(source map[string]any) []map[string]any {
	if len(source) == 0 {
		return nil
	}
	raw, ok := source["resource_list"].([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		row, ok := item.(map[string]any)
		if ok {
			result = append(result, row)
		}
	}
	return result
}

func optionalString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func extractBlueprintCode(inputJSON string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(payload["blueprint_code"]))
}

var hostnameSequencePattern = regexp.MustCompile(`^(.*)\[(\d+),(\d+)\]$`)

func normalizeDeploymentInput(blueprintCode string, raw any) any {
	code := strings.TrimSpace(strings.ToLower(blueprintCode))
	if code != "aws-ec2-server" && code != "alicloud-ecs-server" {
		return raw
	}
	input, ok := raw.(map[string]any)
	if !ok || input == nil {
		return raw
	}
	serverCount := maxIntFromAny(input["server_count"], 1)
	serverName := strings.TrimSpace(toString(input["server_name"]))
	if expanded, matched := expandHostnamePattern(serverName, 0); matched {
		input["server_name"] = expanded
		serverName = expanded
	}
	if instances, ok := input["server_instances"].([]any); ok {
		normalized := make([]any, 0, len(instances))
		for index, entry := range instances {
			item, ok := entry.(map[string]any)
			if !ok {
				normalized = append(normalized, entry)
				continue
			}
			if expanded, matched := expandHostnamePattern(strings.TrimSpace(toString(item["name"])), index); matched {
				item["name"] = expanded
				if index == 0 {
					serverName = expanded
				}
			}
			normalized = append(normalized, item)
		}
		input["server_instances"] = normalized
	}
	if serverName == "" {
		serverName = fmt.Sprintf("server-%02d", serverCount)
	}
	input["server_name"] = serverName
	return input
}

func normalizeRetryInputJSON(inputJSON string) string {
	payload := parseInputJSONMap(inputJSON)
	blueprintCode := strings.TrimSpace(toString(payload["blueprint_code"]))
	if nested, ok := payload["input"].(map[string]any); ok {
		payload["input"] = normalizeDeploymentInput(blueprintCode, nested)
	}
	return mustJSONString(payload)
}

func parseInputJSONMap(raw string) map[string]any {
	result := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return result
	}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func expandHostnamePattern(value string, offset int) (string, bool) {
	matches := hostnameSequencePattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) != 4 {
		return value, false
	}
	base := matches[1]
	startLiteral := matches[2]
	widthLiteral := matches[3]
	start, err := strconv.Atoi(startLiteral)
	if err != nil {
		return value, false
	}
	width, err := strconv.Atoi(widthLiteral)
	if err != nil {
		return value, false
	}
	if width < len(startLiteral) {
		width = len(startLiteral)
	}
	current := start + offset
	return fmt.Sprintf("%s%0*d", base, width, current), true
}

func toString(value any) string {
	switch current := value.(type) {
	case string:
		return current
	default:
		return fmt.Sprint(value)
	}
}

func maxIntFromAny(value any, fallback int) int {
	switch current := value.(type) {
	case int:
		if current > 0 {
			return current
		}
	case int64:
		if current > 0 {
			return int(current)
		}
	case float64:
		if int(current) > 0 {
			return int(current)
		}
	case json.Number:
		if parsed, err := current.Int64(); err == nil && parsed > 0 {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(current)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func buildWorkspacePath(jobID uint) string {
	return fmt.Sprintf("/runner-data/jobs/job-%d", jobID)
}

func normalizeHeartbeatStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "claimed", "planning", "applying", "destroying":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeResultStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "planned", "succeeded", "failed", "cancelled", "destroyed":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func resultLevel(status string) string {
	if status == "failed" {
		return "error"
	}
	return "info"
}

func loadNetworkTopology(db *gorm.DB, networkPlanID *uint) map[string]any {
	if networkPlanID == nil {
		return map[string]any{}
	}
	var plan NetworkPlan
	if err := db.First(&plan, *networkPlanID).Error; err != nil {
		return map[string]any{}
	}
	var topology map[string]any
	if err := json.Unmarshal([]byte(plan.TopologyJSON), &topology); err != nil {
		return map[string]any{}
	}
	topology["region"] = plan.Region
	topology["vpc_cidr"] = plan.VPCCIDR
	return topology
}

func executionRegion(input map[string]any, fallback string) string {
	if region, ok := input["region"].(string); ok && strings.TrimSpace(region) != "" {
		return region
	}
	if nested, ok := input["input"].(map[string]any); ok {
		if region, ok := nested["region"].(string); ok && strings.TrimSpace(region) != "" {
			return region
		}
	}
	return strings.TrimSpace(fallback)
}

func enrichExecutionInput(input map[string]any, account CloudAccount, topology map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range input {
		result[key] = value
	}
	nestedInput := ensureNestedInput(result)
	if _, ok := nestedInput["region"]; !ok || strings.TrimSpace(fmt.Sprint(nestedInput["region"])) == "" {
		if value, ok := topology["region"]; ok {
			nestedInput["region"] = value
		} else {
			nestedInput["region"] = account.Region
		}
	}
	result["region"] = nestedInput["region"]
	if _, ok := nestedInput["default_tags"]; !ok {
		nestedInput["default_tags"] = parseStringList(account.DefaultTagsJSON)
	}
	if _, ok := nestedInput["network_ref"]; !ok || strings.TrimSpace(fmt.Sprint(nestedInput["network_ref"])) == "" {
		switch {
		case strings.TrimSpace(fmt.Sprint(topology["foundation_stack_name"])) != "":
			nestedInput["network_ref"] = topology["foundation_stack_name"]
		case strings.TrimSpace(fmt.Sprint(topology["vpc_name"])) != "":
			nestedInput["network_ref"] = topology["vpc_name"]
		}
	}
	for _, field := range []string{"foundation_stack_name", "name_prefix", "environment", "vpc_name", "vpc_cidr", "availability_zones", "availability_zone_count", "public_subnets", "private_subnets", "subnet_groups", "nat_gateway_count", "create_bastion_subnet", "bastion_subnet_cidr", "security_baseline", "network_role_refs", "internet_entry_refs", "egress_refs", "workload_refs", "data_refs", "ops_refs", "cluster_node_refs", "pod_network_refs", "provider_network_refs"} {
		if _, exists := nestedInput[field]; exists {
			continue
		}
		if value, ok := topology[field]; ok {
			nestedInput[field] = value
		}
	}
	result["input"] = nestedInput
	return result
}

func ensureNestedInput(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if nested, ok := payload["input"].(map[string]any); ok && nested != nil {
		return nested
	}
	nested := map[string]any{}
	payload["input"] = nested
	return nested
}

func (s *DeploymentService) resolveFoundationExecutionInput(blueprint DeploymentBlueprint, account CloudAccount, jobNetworkPlanID *uint, payload map[string]any, topology map[string]any) map[string]any {
	blueprintCode := strings.TrimSpace(strings.ToLower(blueprint.Code))
	if blueprintCode != "aws-bastion" &&
		blueprintCode != "aws-ec2-server" &&
		blueprintCode != "aws-eks-quickstart" &&
		blueprintCode != "alicloud-bastion" &&
		blueprintCode != "alicloud-ecs-server" &&
		blueprintCode != "alicloud-ack-quickstart" {
		return payload
	}
	nestedInput := ensureNestedInput(payload)
	hasVPCID := optionalStringValue(nestedInput["vpc_id"]) != ""
	hasSubnetID := optionalStringValue(nestedInput["subnet_id"]) != "" || optionalStringValue(nestedInput["vswitch_id"]) != ""
	hasSubnetIDs := len(extractStringSlice(nestedInput["subnet_ids"])) > 0
	if hasVPCID && (hasSubnetID || hasSubnetIDs) {
		return payload
	}
	switch blueprintCode {
	case "alicloud-bastion", "alicloud-ecs-server":
		vpcID, vswitchID := s.resolveAliCloudFoundationNetworkResources(account, jobNetworkPlanID, payload)
		if !hasVPCID && vpcID != "" {
			nestedInput["vpc_id"] = vpcID
		}
		if optionalStringValue(nestedInput["vswitch_id"]) == "" && vswitchID != "" {
			nestedInput["vswitch_id"] = vswitchID
		}
	case "alicloud-ack-quickstart":
		vpcID, workerVswitchIDs, podVswitchIDs, slbVswitchIDs := s.resolveAliCloudFoundationClusterResources(account, jobNetworkPlanID, payload)
		if !hasVPCID && vpcID != "" {
			nestedInput["vpc_id"] = vpcID
		}
		if !hasSubnetIDs && len(workerVswitchIDs) > 0 {
			nestedInput["subnet_ids"] = workerVswitchIDs
		}
		if len(extractStringSlice(nestedInput["pod_vswitch_ids"])) == 0 && len(podVswitchIDs) > 0 {
			nestedInput["pod_vswitch_ids"] = podVswitchIDs
		}
		if len(extractStringSlice(nestedInput["slb_vswitch_ids"])) == 0 && len(slbVswitchIDs) > 0 {
			nestedInput["slb_vswitch_ids"] = slbVswitchIDs
		}
	default:
		vpcID, subnetID, subnetIDs := s.resolveAWSFoundationNetworkResources(account, jobNetworkPlanID, payload, topology, blueprintCode)
		if !hasVPCID && vpcID != "" {
			nestedInput["vpc_id"] = vpcID
		}
		if optionalStringValue(nestedInput["subnet_id"]) == "" && subnetID != "" {
			nestedInput["subnet_id"] = subnetID
		}
		if !hasSubnetIDs && len(subnetIDs) > 0 {
			nestedInput["subnet_ids"] = subnetIDs
		}
	}
	payload["input"] = nestedInput
	return payload
}

func optionalStringValue(raw any) string {
	if raw == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" || text == "<nil>" || strings.EqualFold(text, "null") {
		return ""
	}
	return text
}

func (s *DeploymentService) resolveAliCloudFoundationNetworkResources(account CloudAccount, jobNetworkPlanID *uint, payload map[string]any) (string, string) {
	networkPlanID := jobNetworkPlanID
	if networkPlanID == nil {
		networkPlanID = optionalUint(payload["network_plan_id"])
	}
	if networkPlanID == nil {
		return "", ""
	}

	var foundationJob DeploymentJob
	err := s.base.db.
		Table("deployment_jobs").
		Joins("JOIN deployment_blueprints ON deployment_blueprints.id = deployment_jobs.blueprint_id").
		Where("deployment_jobs.provider = ?", "alicloud").
		Where("deployment_jobs.account_id = ?", account.ID).
		Where("deployment_jobs.network_plan_id = ?", *networkPlanID).
		Where("deployment_jobs.status = ?", "succeeded").
		Where("deployment_blueprints.category = ?", "network").
		Order("deployment_jobs.ended_at desc, deployment_jobs.id desc").
		Select("deployment_jobs.*").
		First(&foundationJob).Error
	if err != nil {
		return "", ""
	}

	vpcID, opsVswitchIDs := extractAliCloudFoundationOutputs(foundationJob.OutputJSON)
	vswitchID := ""
	for _, key := range sortedStringKeys(opsVswitchIDs) {
		if value := strings.TrimSpace(opsVswitchIDs[key]); value != "" {
			vswitchID = value
			break
		}
	}
	return vpcID, vswitchID
}

func (s *DeploymentService) resolveAliCloudFoundationClusterResources(account CloudAccount, jobNetworkPlanID *uint, payload map[string]any) (string, []string, []string, []string) {
	networkPlanID := jobNetworkPlanID
	if networkPlanID == nil {
		networkPlanID = optionalUint(payload["network_plan_id"])
	}
	if networkPlanID == nil {
		return "", nil, nil, nil
	}

	var foundationJob DeploymentJob
	err := s.base.db.
		Table("deployment_jobs").
		Joins("JOIN deployment_blueprints ON deployment_blueprints.id = deployment_jobs.blueprint_id").
		Where("deployment_jobs.provider = ?", "alicloud").
		Where("deployment_jobs.account_id = ?", account.ID).
		Where("deployment_jobs.network_plan_id = ?", *networkPlanID).
		Where("deployment_jobs.status = ?", "succeeded").
		Where("deployment_blueprints.category = ?", "network").
		Order("deployment_jobs.ended_at desc, deployment_jobs.id desc").
		Select("deployment_jobs.*").
		First(&foundationJob).Error
	if err != nil {
		return "", nil, nil, nil
	}

	return extractAliCloudFoundationClusterOutputs(foundationJob.OutputJSON)
}

func extractAliCloudFoundationOutputs(outputJSON string) (string, map[string]string) {
	if strings.TrimSpace(outputJSON) == "" {
		return "", nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(outputJSON), &payload); err != nil {
		return "", nil
	}
	outputs, _ := payload["outputs"].(map[string]any)
	vpcID := terraformOutputString(outputs, "vpc_id")
	opsVswitchIDs := terraformOutputStringMap(outputs, "ops_vswitch_ids")
	if len(opsVswitchIDs) == 0 {
		opsVswitchIDs = terraformOutputStringMap(payload, "ops_vswitch_ids")
	}
	return vpcID, opsVswitchIDs
}

func extractAliCloudFoundationClusterOutputs(outputJSON string) (string, []string, []string, []string) {
	if strings.TrimSpace(outputJSON) == "" {
		return "", nil, nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(outputJSON), &payload); err != nil {
		return "", nil, nil, nil
	}
	outputs, _ := payload["outputs"].(map[string]any)
	vpcID := terraformOutputString(outputs, "vpc_id")
	if vpcID == "" {
		vpcID = terraformOutputString(payload, "vpc_id")
	}
	workerVswitchIDs := sortedMapValues(terraformOutputStringMap(outputs, "ack_node_vswitch_ids"))
	if len(workerVswitchIDs) == 0 {
		workerVswitchIDs = sortedMapValues(terraformOutputStringMap(payload, "ack_node_vswitch_ids"))
	}
	podVswitchIDs := sortedMapValues(terraformOutputStringMap(outputs, "ack_pod_vswitch_ids"))
	if len(podVswitchIDs) == 0 {
		podVswitchIDs = sortedMapValues(terraformOutputStringMap(payload, "ack_pod_vswitch_ids"))
	}
	slbVswitchIDs := sortedMapValues(terraformOutputStringMap(outputs, "slb_vswitch_ids"))
	if len(slbVswitchIDs) == 0 {
		slbVswitchIDs = sortedMapValues(terraformOutputStringMap(payload, "slb_vswitch_ids"))
	}
	return vpcID, workerVswitchIDs, podVswitchIDs, slbVswitchIDs
}

func sortedMapValues(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := sortedStringKeys(values)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (s *DeploymentService) resolveAWSFoundationNetworkResources(account CloudAccount, jobNetworkPlanID *uint, payload map[string]any, topology map[string]any, blueprintCode string) (string, string, []string) {
	networkPlanID := jobNetworkPlanID
	if networkPlanID == nil {
		networkPlanID = optionalUint(payload["network_plan_id"])
	}
	if networkPlanID == nil {
		return "", "", nil
	}
	var foundationJob DeploymentJob
	err := s.base.db.
		Table("deployment_jobs").
		Joins("JOIN deployment_blueprints ON deployment_blueprints.id = deployment_jobs.blueprint_id").
		Where("deployment_jobs.provider = ?", "aws").
		Where("deployment_jobs.account_id = ?", account.ID).
		Where("deployment_jobs.network_plan_id = ?", *networkPlanID).
		Where("deployment_jobs.status = ?", "succeeded").
		Where("deployment_blueprints.category = ?", "network").
		Order("deployment_jobs.ended_at desc, deployment_jobs.id desc").
		Select("deployment_jobs.*").
		First(&foundationJob).Error
	if err != nil {
		return "", "", nil
	}

	vpcID, privateSubnetIDs := extractAWSFoundationOutputs(foundationJob.OutputJSON)
	if vpcID == "" {
		vpcID = extractAWSFoundationVPCIDFromResources(s.base.db, foundationJob.ID)
	}
	subnetIDs := resolveAWSSubnetIDsForBlueprint(s.base.db, foundationJob.ID, topology, privateSubnetIDs, blueprintCode)
	subnetID := ""
	if len(subnetIDs) > 0 {
		subnetID = subnetIDs[0]
	}
	return vpcID, subnetID, subnetIDs
}

func extractAWSFoundationVPCIDFromResources(db *gorm.DB, foundationJobID uint) string {
	if foundationJobID == 0 {
		return ""
	}
	var item DeploymentResource
	if err := db.
		Where("job_id = ? AND resource_type = ?", foundationJobID, "aws_vpc").
		Order("id asc").
		First(&item).Error; err != nil {
		return ""
	}
	if strings.TrimSpace(item.CloudID) != "" {
		return strings.TrimSpace(item.CloudID)
	}
	if strings.TrimSpace(item.MetadataJSON) == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(item.MetadataJSON), &payload); err != nil {
		return ""
	}
	metadata, _ := payload["metadata"].(map[string]any)
	return firstNonEmpty(
		strings.TrimSpace(fmt.Sprint(metadata["id"])),
		strings.TrimSpace(fmt.Sprint(metadata["cloud_id"])),
		strings.TrimSpace(fmt.Sprint(payload["cloud_id"])),
	)
}

func extractAWSFoundationOutputs(outputJSON string) (string, map[string]string) {
	if strings.TrimSpace(outputJSON) == "" {
		return "", nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(outputJSON), &payload); err != nil {
		return "", nil
	}
	outputs, _ := payload["outputs"].(map[string]any)
	vpcID := terraformOutputString(outputs, "vpc_id")
	privateSubnetIDs := terraformOutputStringMap(outputs, "private_subnet_ids")
	if len(privateSubnetIDs) == 0 {
		privateSubnetIDs = terraformOutputStringMap(payload, "private_subnet_ids")
	}
	return vpcID, privateSubnetIDs
}

func terraformOutputString(outputs map[string]any, key string) string {
	if outputs == nil {
		return ""
	}
	row, ok := outputs[key].(map[string]any)
	if !ok {
		return strings.TrimSpace(fmt.Sprint(outputs[key]))
	}
	return strings.TrimSpace(fmt.Sprint(row["value"]))
}

func terraformOutputStringMap(outputs map[string]any, key string) map[string]string {
	result := map[string]string{}
	if outputs == nil {
		return result
	}
	raw := outputs[key]
	row, ok := raw.(map[string]any)
	if ok {
		raw = row["value"]
	}
	typed, ok := raw.(map[string]any)
	if !ok {
		return result
	}
	for itemKey, itemValue := range typed {
		text := strings.TrimSpace(fmt.Sprint(itemValue))
		if strings.TrimSpace(itemKey) != "" && text != "" {
			result[itemKey] = text
		}
	}
	return result
}

func sortedStringKeys(items map[string]string) []string {
	if len(items) == 0 {
		return nil
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func resolveAWSSubnetIDForBlueprint(db *gorm.DB, foundationJobID uint, topology map[string]any, privateSubnetIDs map[string]string, blueprintCode string) string {
	subnetIDs := resolveAWSSubnetIDsForBlueprint(db, foundationJobID, topology, privateSubnetIDs, blueprintCode)
	if len(subnetIDs) == 0 {
		return ""
	}
	return subnetIDs[0]
}

func resolveAWSSubnetIDsForBlueprint(db *gorm.DB, foundationJobID uint, topology map[string]any, privateSubnetIDs map[string]string, blueprintCode string) []string {
	selectors := [][]string{}
	switch strings.TrimSpace(strings.ToLower(blueprintCode)) {
	case "aws-eks-quickstart":
		selectors = append(selectors,
			plannedNamesForAWSSelector(topology, "cluster_node_refs", "workload_subnet_refs"),
			plannedNamesForAWSSelector(topology, "pod_network_refs", "private_subnet_refs"),
			plannedNamesForAWSSelector(topology, "workload_refs", "workload_subnet_refs"),
			plannedNamesForAWSSelector(topology, "ops_refs", "ops_subnet_refs"),
		)
	case "aws-ec2-server":
		selectors = append(selectors,
			plannedNamesForAWSSelector(topology, "workload_refs", "workload_subnet_refs"),
			plannedNamesForAWSSelector(topology, "ops_refs", "ops_subnet_refs"),
		)
	default:
		selectors = append(selectors, plannedNamesForAWSSelector(topology, "ops_refs", "ops_subnet_refs"))
	}

	var items []DeploymentResource
	_ = db.Where("job_id = ? AND resource_type = ?", foundationJobID, "aws_subnet").Find(&items).Error

	collected := make([]string, 0)
	seen := map[string]struct{}{}
	appendSubnet := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		collected = append(collected, value)
	}

	for _, plannedNames := range selectors {
		for _, item := range items {
			if subnetID := matchSubnetByPlannedName(item.MetadataJSON, plannedNames); subnetID != "" {
				appendSubnet(subnetID)
			}
		}
	}

	if len(collected) > 0 {
		return collected
	}

	keys := make([]string, 0, len(privateSubnetIDs))
	for key := range privateSubnetIDs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		appendSubnet(privateSubnetIDs[key])
	}
	return collected
}

func plannedNamesForAWSSelector(topology map[string]any, refKey, providerKey string) []string {
	plannedNames := plannedNamesFromRefs(topology[refKey])
	if len(plannedNames) > 0 {
		return plannedNames
	}
	return plannedNamesFromProviderRefs(topology["provider_network_refs"], providerKey)
}

func plannedNamesFromRefs(value any) []string {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0)
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, extractStringSlice(item["planned_names"])...)
	}
	return result
}

func plannedNamesFromProviderRefs(value any, key string) []string {
	rows, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return plannedNamesFromRefs(rows[key])
}

func matchSubnetByPlannedName(metadataJSON string, plannedNames []string) string {
	if strings.TrimSpace(metadataJSON) == "" || len(plannedNames) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &payload); err != nil {
		return ""
	}
	metadata, _ := payload["metadata"].(map[string]any)
	tags, _ := metadata["tags"].(map[string]any)
	name := strings.TrimSpace(fmt.Sprint(tags["Name"]))
	if name == "" {
		return ""
	}
	for _, plannedName := range plannedNames {
		if strings.TrimSpace(plannedName) == name {
			return firstNonEmpty(
				strings.TrimSpace(fmt.Sprint(metadata["id"])),
				strings.TrimSpace(fmt.Sprint(payload["cloud_id"])),
			)
		}
	}
	return ""
}

func optionalUint(value any) *uint {
	switch typed := value.(type) {
	case uint:
		if typed == 0 {
			return nil
		}
		return &typed
	case float64:
		converted := uint(typed)
		if converted == 0 {
			return nil
		}
		return &converted
	case int:
		if typed <= 0 {
			return nil
		}
		converted := uint(typed)
		return &converted
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return nil
		}
		var converted uint
		if _, err := fmt.Sscanf(typed, "%d", &converted); err != nil || converted == 0 {
			return nil
		}
		return &converted
	default:
		return nil
	}
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

func (s *DeploymentService) resolveExecutionEnvironment(account CloudAccount, provider, region string) (map[string]string, error) {
	accessKey, err := s.base.cipher.Decrypt(account.AccessKeyEncrypted)
	if err != nil {
		return nil, err
	}
	secretKey, err := s.base.cipher.Decrypt(account.SecretKeyEncrypted)
	if err != nil {
		return nil, err
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = strings.TrimSpace(account.Region)
	}
	env := map[string]string{}
	switch normalizeProvider(provider) {
	case "aws":
		env["AWS_ACCESS_KEY_ID"] = accessKey
		env["AWS_SECRET_ACCESS_KEY"] = secretKey
		env["AWS_REGION"] = region
		env["AWS_DEFAULT_REGION"] = region
		env["AWS_EC2_METADATA_DISABLED"] = "true"
	case "alicloud":
		env["ALICLOUD_ACCESS_KEY"] = accessKey
		env["ALICLOUD_SECRET_KEY"] = secretKey
		env["ALICLOUD_REGION"] = region
	}
	return env, nil
}
