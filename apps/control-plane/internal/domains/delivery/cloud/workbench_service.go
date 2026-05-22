package cloud

import (
	"sort"
	"strconv"
	"strings"
)

type WorkbenchService struct {
	summary     SummaryOperations
	accounts    *AccountService
	networks    *NetworkPlanService
	blueprints  BlueprintOperations
	deployments *DeploymentService
	resources   *ResourceService
}

type cloudWorkbenchData struct {
	Summary      Summary
	Accounts     []CloudAccountView
	NetworkPlans []NetworkPlanView
	Blueprints   []BlueprintView
	Jobs         []DeploymentJobView
	Resources    []CloudResourceView
}

func (s *WorkbenchService) Get() (CloudWorkbenchView, error) {
	accounts, err := s.accounts.List()
	if err != nil {
		return CloudWorkbenchView{}, err
	}
	networks, err := s.networks.List()
	if err != nil {
		return CloudWorkbenchView{}, err
	}
	var blueprints []BlueprintView
	if s.blueprints != nil {
		loadedBlueprints, err := s.blueprints.List()
		if err != nil {
			return CloudWorkbenchView{}, err
		}
		blueprints = loadedBlueprints
	}
	jobs, err := s.deployments.List()
	if err != nil {
		return CloudWorkbenchView{}, err
	}
	resources, err := s.resources.List()
	if err != nil {
		return CloudWorkbenchView{}, err
	}

	summary := Summary{
		Accounts:   int64(len(accounts)),
		Networks:   int64(len(networks)),
		Blueprints: int64(len(blueprints)),
		Jobs:       int64(len(jobs)),
		Resources:  int64(len(resources)),
	}
	for _, job := range jobs {
		if isQueuedOrRunning(job.Status) {
			summary.QueuedJobs++
		}
	}
	if s.summary != nil {
		loadedSummary, err := s.summary.Summary()
		if err != nil {
			return CloudWorkbenchView{}, err
		}
		summary = loadedSummary
	}

	return buildCloudWorkbench(cloudWorkbenchData{
		Summary:      summary,
		Accounts:     accounts,
		NetworkPlans: networks,
		Blueprints:   blueprints,
		Jobs:         jobs,
		Resources:    resources,
	}), nil
}

func buildCloudWorkbench(data cloudWorkbenchData) CloudWorkbenchView {
	accounts := cloneCloudAccounts(data.Accounts)
	networks := cloneNetworkPlans(data.NetworkPlans)
	blueprints := cloneBlueprints(data.Blueprints)
	jobs := cloneDeploymentJobs(data.Jobs)
	resources := cloneCloudResources(data.Resources)
	sortJobsByCreatedAtDesc(jobs)
	sortResourcesByUpdatedAtDesc(resources)

	activeAccounts := countActiveAccounts(accounts)
	networkCount := len(networks)
	if data.Summary.Networks > int64(networkCount) {
		networkCount = int(data.Summary.Networks)
	}
	blueprintByID := blueprintMapByID(blueprints)
	succeededFoundationJobs := countSucceededFoundationJobs(jobs, blueprintByID)
	bastionSucceeded := hasSucceededBastionJob(jobs, blueprintByID)
	foundationComplete := activeAccounts > 0 && networkCount > 0 && succeededFoundationJobs > 0
	clusterContractReady := hasClusterContract(blueprints)
	latestJob := firstJob(jobs)
	deliveryReadyCount := computeDeliveryReadyCount(foundationComplete, bastionSucceeded, clusterContractReady)
	deliveryHeadline := deliveryHeadlineFor(foundationComplete, bastionSucceeded)

	stage := buildWorkbenchStage(workbenchStageInput{
		ActiveAccounts:          activeAccounts,
		NetworkPlans:            networkCount,
		SucceededFoundationJobs: succeededFoundationJobs,
		FoundationComplete:      foundationComplete,
		BastionSucceeded:        bastionSucceeded,
		ClusterContractReady:    clusterContractReady,
		LatestJob:               latestJob,
	})

	return CloudWorkbenchView{
		PayloadVersion:          "2026-05-foundation-delivery",
		ModuleLabel:             "基础建设",
		WorkflowLabel:           "基础交付",
		Summary:                 data.Summary,
		Stage:                   stage,
		Metrics:                 buildWorkbenchMetrics(data.Summary, activeAccounts, networkCount, succeededFoundationJobs, deliveryReadyCount, deliveryHeadline),
		Actions:                 buildWorkbenchActions(stage),
		Blockers:                buildWorkbenchBlockers(stage),
		ActiveAccounts:          activeAccounts,
		SucceededFoundationJobs: succeededFoundationJobs,
		DeliveryReadyCount:      deliveryReadyCount,
		FoundationComplete:      foundationComplete,
		BastionSucceeded:        bastionSucceeded,
		ClusterContractReady:    clusterContractReady,
		RecentJobs:              limitJobs(jobs, 5),
		RecentResources:         limitResources(resources, 8),
		Accounts:                accounts,
		NetworkPlans:            networks,
		Blueprints:              blueprints,
		Jobs:                    jobs,
		Resources:               resources,
	}
}

type workbenchStageInput struct {
	ActiveAccounts          int
	NetworkPlans            int
	SucceededFoundationJobs int
	FoundationComplete      bool
	BastionSucceeded        bool
	ClusterContractReady    bool
	LatestJob               *DeploymentJobView
}

func buildWorkbenchStage(input workbenchStageInput) CloudWorkbenchStage {
	latestTitle, latestCopy := latestJobCopy(input.LatestJob)
	if input.ActiveAccounts == 0 {
		return CloudWorkbenchStage{
			Key:                  "account",
			Label:                "Stage 01",
			Tone:                 "blocked",
			Title:                "先新增并验证云账号",
			Copy:                 "基础交付从账号开始。没有可用账号，基础网络、服务器和集群都不能进入稳定路径。",
			NextActionTitle:      "新增云账号",
			NextActionCopy:       "保存后先测试账号可用性，再继续创建基础网络。",
			PrimaryAction:        "create-account",
			PrimaryActionLabel:   "新增云账号",
			SecondaryAction:      "refresh",
			SecondaryActionLabel: "刷新状态",
			BlockerTitle:         "没有可用云账号",
			BlockerCopy:          "至少需要一组 active 状态的 AWS 或阿里云账号。",
			LatestResultTitle:    latestTitle,
			LatestResultCopy:     latestCopy,
		}
	}
	if input.NetworkPlans == 0 {
		return CloudWorkbenchStage{
			Key:                  "foundation_network",
			Label:                "Stage 02",
			Tone:                 "active",
			Title:                "创建第一条基础网络",
			Copy:                 "账号已经可用，下一步是把 VPC、子网角色、NAT、运维入口这些基础设施约束收口成基础网络。",
			NextActionTitle:      "创建基础网络",
			NextActionCopy:       "基础网络会绑定 provider、账号、区域和环境，后续交付对象都从这里继续。",
			PrimaryAction:        "create-network",
			PrimaryActionLabel:   "创建基础网络",
			SecondaryAction:      "create-account",
			SecondaryActionLabel: "管理云账号",
			BlockerTitle:         "还没有基础网络",
			BlockerCopy:          "没有基础网络，服务器、Ops Bastion 和集群会缺少可复用的网络上下文。",
			LatestResultTitle:    latestTitle,
			LatestResultCopy:     latestCopy,
		}
	}
	if !input.FoundationComplete {
		return CloudWorkbenchStage{
			Key:                  "foundation_apply",
			Label:                "Stage 03",
			Tone:                 "active",
			Title:                "先让基础网络成功落地",
			Copy:                 "基础网络记录已经存在，但还没有成功的基础交付任务。先确认执行状态和输出结果，再继续创建上层对象。",
			NextActionTitle:      "查看基础网络和任务状态",
			NextActionCopy:       "确认网络 apply 是否成功，以及 outputs 是否已经回填到资源结果。",
			PrimaryAction:        "scroll-networks",
			PrimaryActionLabel:   "查看基础网络",
			SecondaryAction:      "scroll-jobs",
			SecondaryActionLabel: "查看任务状态",
			BlockerTitle:         "还没有成功的基础交付任务",
			BlockerCopy:          "没有成功的基础网络 apply，后续蓝图无法稳定复用 outputs。",
			LatestResultTitle:    latestTitle,
			LatestResultCopy:     latestCopy,
		}
	}
	if !input.BastionSucceeded {
		return CloudWorkbenchStage{
			Key:                  "delivery_objects",
			Label:                "Stage 04",
			Tone:                 "ready",
			Title:                "从基础网络继续创建交付对象",
			Copy:                 "基础网络已经成功落地。现在优先从具体网络继续创建服务器、Ops Bastion 或集群，不要绕开底座创建孤立任务。",
			NextActionTitle:      "从网络卡片发起基础交付",
			NextActionCopy:       "选择一条基础网络，再创建服务器、Ops Bastion 或集群。",
			PrimaryAction:        "scroll-networks",
			PrimaryActionLabel:   "从基础网络继续创建",
			SecondaryAction:      "scroll-jobs",
			SecondaryActionLabel: "查看最近任务",
			BlockerTitle:         "还没有成功的 Ops Bastion",
			BlockerCopy:          "基础网络可用后，建议先跑通一条运维接入路径。",
			LatestResultTitle:    latestTitle,
			LatestResultCopy:     latestCopy,
		}
	}

	blockerTitle := "继续扩展集群和交付对象"
	blockerCopy := "基础路径已经跑通，可以继续从现有基础网络扩展服务器、Ops Bastion 或集群。"
	if !input.ClusterContractReady {
		blockerTitle = "集群网络 contract 仍待补齐"
		blockerCopy = "集群蓝图还需要稳定消费 network_ref、cluster_node_refs 和 provider refs。"
	}
	return CloudWorkbenchStage{
		Key:                  "scale",
		Label:                "Stage 05",
		Tone:                 "ready",
		Title:                "基础路径已经跑通",
		Copy:                 "账号、基础网络和第一条计算交付都已有成功记录。现在可以围绕现有网络继续扩展更多对象。",
		NextActionTitle:      "继续复用基础网络扩展",
		NextActionCopy:       "先确认最近结果，再从已有基础网络扩展服务器、Bastion 或集群。",
		PrimaryAction:        "scroll-networks",
		PrimaryActionLabel:   "从基础网络扩展",
		SecondaryAction:      "scroll-jobs",
		SecondaryActionLabel: "查看任务状态",
		BlockerTitle:         blockerTitle,
		BlockerCopy:          blockerCopy,
		LatestResultTitle:    latestTitle,
		LatestResultCopy:     latestCopy,
	}
}

func buildWorkbenchMetrics(summary Summary, activeAccounts, networkCount, succeededFoundationJobs, deliveryReadyCount int, deliveryHeadline string) []CloudWorkbenchMetric {
	return []CloudWorkbenchMetric{
		{
			Key:   "active_accounts",
			Label: "可用账号",
			Value: int64(activeAccounts),
			Copy:  accountMetricCopy(activeAccounts),
			Tone:  readyTone(activeAccounts > 0),
		},
		{
			Key:   "foundation_networks",
			Label: "基础网络",
			Value: int64(networkCount),
			Copy:  networkMetricCopy(networkCount, succeededFoundationJobs),
			Tone:  readyTone(networkCount > 0 && succeededFoundationJobs > 0),
		},
		{
			Key:   "delivery_ready",
			Label: "交付对象",
			Value: int64(deliveryReadyCount),
			Copy:  deliveryHeadline,
			Tone:  readyTone(deliveryReadyCount > 0),
		},
		{
			Key:   "execution_results",
			Label: "任务与资源",
			Value: summary.Jobs,
			Copy:  buildResultMetricCopy(summary),
			Tone:  readyTone(summary.QueuedJobs == 0),
		},
	}
}

func buildWorkbenchActions(stage CloudWorkbenchStage) []CloudWorkbenchAction {
	actions := []CloudWorkbenchAction{
		{Key: stage.PrimaryAction, Label: stage.PrimaryActionLabel, Kind: "shortcut", Primary: true, Enabled: true},
	}
	if strings.TrimSpace(stage.SecondaryAction) != "" {
		actions = append(actions, CloudWorkbenchAction{Key: stage.SecondaryAction, Label: stage.SecondaryActionLabel, Kind: "shortcut", Enabled: true})
	}
	return actions
}

func buildWorkbenchBlockers(stage CloudWorkbenchStage) []CloudWorkbenchBlocker {
	if strings.TrimSpace(stage.BlockerTitle) == "" {
		return []CloudWorkbenchBlocker{}
	}
	severity := "info"
	if stage.Tone == "blocked" {
		severity = "critical"
	} else if stage.Tone == "active" {
		severity = "warning"
	}
	return []CloudWorkbenchBlocker{{
		Title:     stage.BlockerTitle,
		Copy:      stage.BlockerCopy,
		Severity:  severity,
		ActionKey: stage.PrimaryAction,
	}}
}

func countActiveAccounts(accounts []CloudAccountView) int {
	count := 0
	for _, account := range accounts {
		if strings.EqualFold(strings.TrimSpace(account.Status), "active") {
			count++
		}
	}
	return count
}

func countSucceededFoundationJobs(jobs []DeploymentJobView, blueprintByID map[uint]BlueprintView) int {
	count := 0
	for _, job := range jobs {
		if !strings.EqualFold(strings.TrimSpace(job.Status), "succeeded") {
			continue
		}
		blueprint := blueprintByID[job.BlueprintID]
		if strings.EqualFold(strings.TrimSpace(blueprint.Category), "network") {
			count++
		}
	}
	return count
}

func hasSucceededBastionJob(jobs []DeploymentJobView, blueprintByID map[uint]BlueprintView) bool {
	for _, job := range jobs {
		if !strings.EqualFold(strings.TrimSpace(job.Status), "succeeded") {
			continue
		}
		blueprint := blueprintByID[job.BlueprintID]
		if !strings.EqualFold(strings.TrimSpace(blueprint.Category), "compute") {
			continue
		}
		name := strings.ToLower(job.BlueprintName + " " + blueprint.Code + " " + blueprint.Name)
		if strings.Contains(name, "bastion") {
			return true
		}
	}
	return false
}

func hasClusterContract(blueprints []BlueprintView) bool {
	for _, blueprint := range blueprints {
		if !strings.EqualFold(strings.TrimSpace(blueprint.Category), "cluster") {
			continue
		}
		schema := strings.ToLower(blueprint.SchemaJSON)
		if strings.Contains(schema, "network_ref") || strings.Contains(schema, "cluster_node_refs") || strings.Contains(schema, "provider_network_refs") {
			return true
		}
	}
	return false
}

func computeDeliveryReadyCount(foundationComplete, bastionSucceeded, clusterContractReady bool) int {
	count := 0
	if foundationComplete {
		count++
	}
	if bastionSucceeded {
		count++
	}
	if clusterContractReady {
		count++
	}
	return count
}

func deliveryHeadlineFor(foundationComplete, bastionSucceeded bool) string {
	if !foundationComplete {
		return "基础网络还没有成功落地，暂不建议创建上层对象。"
	}
	if !bastionSucceeded {
		return "基础网络已经可用，下一步从网络继续创建服务器或 Ops Bastion。"
	}
	return "基础网络和运维接入路径已经跑通，可以继续扩展更多对象。"
}

func accountMetricCopy(activeAccounts int) string {
	if activeAccounts > 0 {
		return "已有可用账号，可以继续基础交付。"
	}
	return "还没有可用账号，基础交付无法启动。"
}

func networkMetricCopy(networkCount, succeededFoundationJobs int) string {
	if succeededFoundationJobs > 0 {
		return "基础网络已有成功执行结果，可供后续对象复用。"
	}
	if networkCount > 0 {
		return "基础网络已定义，但还需要成功执行。"
	}
	return "还没有基础网络。"
}

func buildResultMetricCopy(summary Summary) string {
	return strings.TrimSpace(strings.Join([]string{
		int64Label(summary.Resources, "条资源结果"),
		int64Label(summary.QueuedJobs, "条任务排队或执行中"),
	}, "，")) + "。"
}

func int64Label(value int64, unit string) string {
	return strings.TrimSpace(strings.Join([]string{formatInt64(value), unit}, " "))
}

func readyTone(ready bool) string {
	if ready {
		return "ready"
	}
	return "waiting"
}

func latestJobCopy(job *DeploymentJobView) (string, string) {
	if job == nil {
		return "还没有任务记录", "从新增云账号和创建基础网络开始。"
	}
	return strings.TrimSpace(job.Name + " · " + jobStatusLabel(job.Status)), strings.TrimSpace(job.BlueprintName + " · " + jobActionLabel(job.Action))
}

func jobStatusLabel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "queued":
		return "排队中"
	case "claimed":
		return "已领取"
	case "planning":
		return "规划中"
	case "planned":
		return "已完成 Plan"
	case "applying":
		return "执行中"
	case "destroying":
		return "销毁中"
	case "succeeded":
		return "已成功"
	case "failed":
		return "已失败"
	case "destroyed":
		return "已销毁"
	case "cancelled":
		return "已取消"
	default:
		return strings.TrimSpace(value)
	}
}

func jobActionLabel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "plan":
		return "Plan"
	case "apply":
		return "Apply"
	case "destroy":
		return "Destroy"
	default:
		return strings.TrimSpace(value)
	}
}

func isQueuedOrRunning(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "queued", "claimed", "planning", "applying", "destroying":
		return true
	default:
		return false
	}
}

func firstJob(jobs []DeploymentJobView) *DeploymentJobView {
	if len(jobs) == 0 {
		return nil
	}
	return &jobs[0]
}

func blueprintMapByID(items []BlueprintView) map[uint]BlueprintView {
	result := make(map[uint]BlueprintView, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func sortJobsByCreatedAtDesc(items []DeploymentJobView) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func sortResourcesByUpdatedAtDesc(items []CloudResourceView) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
}

func limitJobs(items []DeploymentJobView, limit int) []DeploymentJobView {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func limitResources(items []CloudResourceView, limit int) []CloudResourceView {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func cloneCloudAccounts(items []CloudAccountView) []CloudAccountView {
	return append([]CloudAccountView(nil), items...)
}

func cloneNetworkPlans(items []NetworkPlanView) []NetworkPlanView {
	return append([]NetworkPlanView(nil), items...)
}

func cloneBlueprints(items []BlueprintView) []BlueprintView {
	return append([]BlueprintView(nil), items...)
}

func cloneDeploymentJobs(items []DeploymentJobView) []DeploymentJobView {
	return append([]DeploymentJobView(nil), items...)
}

func cloneCloudResources(items []CloudResourceView) []CloudResourceView {
	return append([]CloudResourceView(nil), items...)
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
