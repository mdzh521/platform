import { api } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import {
  buildManifestTemplate,
  createDefaultBuilderState,
  createDefaultYAMLState,
  createEmptyClaimTemplate,
  createEmptyContainer,
  createEmptyContainerSecurityContext,
  createEmptyDNSOption,
  createEmptyEnvFromSource,
  createEmptyEnvValueSource,
  createEmptyHostAlias,
  createEmptyNodeAffinityRule,
  createEmptyNodeSelector,
  createEmptyPodAffinityRule,
  createEmptyProbe,
  createEmptyToleration,
  createEmptyTopologySpreadConstraint,
  createEmptyVolumeMount,
  ensureWorkloadCreatorDefaults,
  normalizeCreatorValue,
} from "./workload-creator-state.js";
import {
  serializeAffinity,
  serializeContainerSecurityContext,
  serializeDNSConfig,
  serializeEnvFromSources,
  serializeEnvValueSources,
  serializeHostAliases,
  serializeNodeSelectors,
  serializePodSecurityContext,
  serializeProbe,
  serializeTolerations,
  serializeTopologySpreadConstraints,
  serializeVolumeMounts,
} from "./workload-creator-serialize.js";
import { state } from "../../core/state.js";
import { closeDrawer, openDrawer, toast } from "../../core/ui.js";
import { emptyState, escapeHtml } from "../../shared/utils.js";

let ctx = null;

export function configureWorkloadCreator(deps) {
  ctx = deps;
}

export function openCreateWorkloadDrawer() {
  state.workloadCreator.preset = "Deployment";
  state.workloadCreator.mode = "builder";
  state.workloadCreator.step = "basic";
  state.workloadCreator.builder = createDefaultBuilderState(ctx, "Deployment");
  state.workloadCreator.resourceBuilder = createDefaultResourceBuilderState("Service");
  state.workloadCreator.yaml = createDefaultYAMLState(ctx, "Deployment");
  renderWorkloadCreator();
  openDrawer("workload");
}

export function openCreateResourceDrawer() {
  state.workloadCreator.preset = selectedYamlPreset();
  state.workloadCreator.mode = "builder";
  state.workloadCreator.step = "basic";
  state.workloadCreator.builder = createDefaultBuilderState(ctx, "Deployment");
  state.workloadCreator.resourceBuilder = createDefaultResourceBuilderState(state.workloadCreator.preset);
  state.workloadCreator.yaml = createDefaultYAMLState(ctx, state.workloadCreator.preset);
  renderWorkloadCreator();
  openDrawer("workload");
}

function selectedYamlPreset() {
  const kind = String(state.business.selectedResourceKind || "Service").trim();
  return kind || "Service";
}

export function renderWorkloadCreator() {
  ensureWorkloadCreatorDefaults();
  const mode = state.workloadCreator.mode;
  const steps = [
    { id: "basic", label: "基本信息" },
    { id: "containers", label: "容器组" },
    { id: "storage", label: "存储挂载" },
    { id: "advanced", label: "高级设置" },
    { id: "exposure", label: "服务暴露" },
  ];
  const currentIndex = steps.findIndex((item) => item.id === state.workloadCreator.step);
  const builder = state.workloadCreator.builder;
  const resourceBuilder = state.workloadCreator.resourceBuilder;
  const yamlState = state.workloadCreator.yaml;
  const builderContent = isResourcePreset(state.workloadCreator.preset)
    ? `<div class="creator-content">${renderResourceBuilder(resourceBuilder, state.workloadCreator.preset)}</div>`
    : `<div class="creator-layout"><nav class="creator-steps">${steps.map((item, index) => `<button class="creator-step ${item.id === state.workloadCreator.step ? "active" : ""}" data-action="creator-step" data-id="${item.id}"><span>${index + 1}</span><strong>${item.label}</strong></button>`).join("")}</nav><section class="creator-content">${renderBuilderCreatorStep(builder, state.workloadCreator.step)}</section></div>`;
  const builderFooter = isResourcePreset(state.workloadCreator.preset)
    ? `<div class="creator-footer"><div></div><div class="action-row"><button class="ghost-button" data-action="creator-fill-template">查看 YAML</button><button class="primary-button" data-action="creator-submit-builder">创建 ${escapeHtml(shortPresetLabel(state.workloadCreator.preset))}</button></div></div>`
    : `<div class="creator-footer"><button class="ghost-button" data-action="creator-prev" ${currentIndex <= 0 ? "disabled" : ""}>上一步</button><div class="action-row"><button class="ghost-button" data-action="creator-fill-template">填充模板</button><button class="primary-button" data-action="creator-submit-builder">创建 ${escapeHtml(builder.workload_type || "Deployment")}</button><button class="ghost-button" data-action="creator-next" ${currentIndex >= steps.length - 1 ? "disabled" : ""}>下一步</button></div></div>`;
  elements.workloadCreatorRoot.innerHTML = `${renderCreatorCatalog()}<div class="creator-mode-bar"><button class="ghost-button ${mode === "builder" ? "active" : ""}" data-action="creator-mode" data-id="builder">图形化创建</button><button class="ghost-button ${mode === "yaml" ? "active" : ""}" data-action="creator-mode" data-id="yaml">YAML 创建</button></div>${mode === "builder" ? `${builderContent}${builderFooter}` : `<div class="creator-content"><div class="creator-section"><div class="section-heading"><h4>YAML 创建</h4><p>支持多文档 YAML。未声明 namespace 的命名空间级资源会补全为默认命名空间。</p></div><div class="form-grid"><label><span>集群</span>${renderClusterSelect("yaml", yamlState.cluster_id)}</label><label><span>默认命名空间</span><input data-creator-mode="yaml" data-key="namespace" value="${escapeHtml(yamlState.namespace)}" placeholder="default" /></label><label><span>模板类型</span><select data-creator-mode="yaml" data-key="preset_kind">${renderYamlPresetOptions(state.workloadCreator.preset)}</select></label><label class="field-span-2"><span>Manifest YAML</span><textarea rows="20" data-creator-mode="yaml" data-key="manifest_yaml">${escapeHtml(yamlState.manifest_yaml)}</textarea></label></div></div></div><div class="creator-footer"><div></div><div class="action-row"><button class="ghost-button" data-action="creator-fill-template">填充模板</button><button class="primary-button" data-action="creator-submit-yaml">应用 YAML</button></div></div>`}`;
}

function renderCreatorCatalog() {
  const preset = state.workloadCreator.preset || "Deployment";
  const workloadKinds = ["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"];
  const resourceKinds = ["Service", "Ingress", "ConfigMap", "Secret", "PersistentVolumeClaim", "ServiceAccount", "ResourceQuota", "LimitRange"];
  return `<section class="creator-catalog"><div class="section-heading"><h4>资源类型</h4><p>参考 Kuboard，把创建入口先按对象类型分层。工作负载优先走图形化，配置 / 网络 / 治理对象同时支持图形化和 YAML。</p></div><div class="creator-catalog-grid"><article class="creator-catalog-card"><p class="eyebrow">Workload</p><h5>工作负载</h5><p class="table-meta">Deployment、StatefulSet、DaemonSet、Job、CronJob</p><div class="resource-pill-row">${workloadKinds.map((kind) => `<button class="catalog-chip ${preset === kind && state.workloadCreator.mode === "builder" ? "active" : ""}" data-action="creator-preset-builder" data-id="${kind}">${kind}</button>`).join("")}</div></article><article class="creator-catalog-card"><p class="eyebrow">Resource</p><h5>配置 / 网络 / 治理</h5><p class="table-meta">Service、Ingress、ConfigMap、Secret、PVC、ServiceAccount、Quota、LimitRange</p><div class="resource-pill-row">${resourceKinds.map((kind) => `<button class="catalog-chip ${preset === kind && state.workloadCreator.mode === "builder" ? "active" : ""}" data-action="creator-preset-resource" data-id="${kind}">${shortPresetLabel(kind)}</button>`).join("")}</div></article><article class="creator-catalog-card"><p class="eyebrow">Mode</p><h5>创建方式</h5><p class="table-meta">图形化适合常见对象，YAML 适合完整清单和高级字段。</p><div class="creator-catalog-copy"><span class="cluster-chip">${escapeHtml(currentPresetDescription())}</span></div></article></div></section>`;
}

function renderYamlPresetOptions(selected) {
  return ["Deployment", "Service", "Ingress", "ConfigMap", "Secret", "PersistentVolumeClaim", "ServiceAccount", "ResourceQuota", "LimitRange", "CronJob"]
    .map((kind) => `<option value="${kind}" ${selected === kind ? "selected" : ""}>${kind}</option>`)
    .join("");
}

function shortPresetLabel(kind) {
  const labels = {
    PersistentVolumeClaim: "PVC",
    ServiceAccount: "SA",
    ResourceQuota: "Quota",
    LimitRange: "Limit",
  };
  return labels[kind] || kind;
}

function currentPresetDescription() {
  const preset = state.workloadCreator.preset || "Deployment";
  if (state.workloadCreator.mode === "builder") {
    return `当前图形化创建 ${preset}`;
  }
  return `当前 YAML 模板 ${preset}`;
}

function isResourcePreset(preset) {
  return ["Service", "Ingress", "ConfigMap", "Secret", "PersistentVolumeClaim", "ServiceAccount", "ResourceQuota", "LimitRange"].includes(preset);
}

function createDefaultResourceBuilderState(preset) {
  const clusterID = ctx.selectedResourceClusterID?.() || ctx.selectedWorkloadClusterID?.() || ctx.selectedClusterID?.() || state.clusters[0]?.id || "";
  const namespace = ctx.selectedResourceNamespace?.() || ctx.selectedWorkloadNamespace?.() || state.business.selectedNamespace || "default";
  return {
    kind: preset,
    cluster_id: clusterID ? String(clusterID) : "",
    namespace,
    name: `demo-${shortPresetLabel(preset).toLowerCase()}`,
    labels_text: "",
    annotations_text: "",
    service_type: "ClusterIP",
    service_selector_text: "app=demo-app",
    service_ports_text: "80:8080:TCP",
    service_external_ips_text: "",
    service_session_affinity: "None",
    ingress_class_name: "nginx",
    ingress_host: "demo.local",
    ingress_path: "/",
    ingress_service_name: "demo-service",
    ingress_service_port: "80",
    ingress_tls_secret_name: "",
    config_entries_text: "APP_ENV=production\nLOG_LEVEL=info",
    secret_type: "Opaque",
    secret_entries_text: "username=admin\npassword=change-me",
    pvc_storage_class_name: "",
    pvc_size: "5Gi",
    pvc_access_modes_text: "ReadWriteOnce",
    image_pull_secrets_text: "",
    quota_hard_text: "pods=10\nrequests.cpu=2\nrequests.memory=2Gi",
    limit_rules_text: "- type: Container\n  default:\n    cpu: \"500m\"\n    memory: 512Mi\n  defaultRequest:\n    cpu: \"100m\"\n    memory: 128Mi",
  };
}

function renderClusterSelect(mode, value) {
  return `<select data-creator-mode="${mode}" data-key="cluster_id">${state.clusters.map((item) => `<option value="${item.id}" ${String(item.id) === String(value) ? "selected" : ""}>${escapeHtml(item.name)}</option>`).join("")}</select>`;
}

function renderBuilderCreatorStep(builder, step) {
  if (step === "basic") {
    return `<div class="creator-section"><div class="section-heading"><h4>基本信息</h4><p>先确定工作负载类型和基础元数据。Deployment / StatefulSet / DaemonSet / Job / CronJob 都在这里切换。</p></div><div class="form-grid"><label><span>集群</span>${renderClusterSelect("builder", builder.cluster_id)}</label><label><span>命名空间</span><input data-creator-mode="builder" data-key="namespace" value="${escapeHtml(builder.namespace)}" /></label><label><span>工作负载类型</span><select data-creator-mode="builder" data-key="workload_type"><option value="Deployment" ${builder.workload_type === "Deployment" ? "selected" : ""}>Deployment</option><option value="StatefulSet" ${builder.workload_type === "StatefulSet" ? "selected" : ""}>StatefulSet</option><option value="DaemonSet" ${builder.workload_type === "DaemonSet" ? "selected" : ""}>DaemonSet</option><option value="Job" ${builder.workload_type === "Job" ? "selected" : ""}>Job</option><option value="CronJob" ${builder.workload_type === "CronJob" ? "selected" : ""}>CronJob</option></select></label><label><span>名称</span><input data-creator-mode="builder" data-key="name" value="${escapeHtml(builder.name)}" /></label>${builder.workload_type !== "DaemonSet" ? `<label><span>${builder.workload_type === "CronJob" ? "执行次数" : "副本数"}</span><input type="number" min="1" data-creator-mode="builder" data-key="replicas" value="${escapeHtml(builder.replicas)}" /></label>` : `<label><span>控制器说明</span><input value="DaemonSet 自动按节点铺开" disabled /></label>`}${builder.workload_type === "StatefulSet" ? `<label><span>Service 名称</span><input data-creator-mode="builder" data-key="service_name" value="${escapeHtml(builder.service_name)}" placeholder="留空默认同名" /></label><label><span>Service 模式</span><select data-creator-mode="builder" data-key="statefulset_service_mode"><option value="Headless" ${builder.statefulset_service_mode === "Headless" ? "selected" : ""}>Headless</option><option value="ClusterIP" ${builder.statefulset_service_mode === "ClusterIP" ? "selected" : ""}>ClusterIP</option><option value="LoadBalancer" ${builder.statefulset_service_mode === "LoadBalancer" ? "selected" : ""}>LoadBalancer</option></select></label><label><span>Pod Management</span><select data-creator-mode="builder" data-key="pod_management_policy"><option value="OrderedReady" ${builder.pod_management_policy === "OrderedReady" ? "selected" : ""}>OrderedReady</option><option value="Parallel" ${builder.pod_management_policy === "Parallel" ? "selected" : ""}>Parallel</option></select></label><label><span>Rolling Partition</span><input type="number" min="0" data-creator-mode="builder" data-key="statefulset_partition" value="${escapeHtml(builder.statefulset_partition)}" placeholder="0" /></label>` : ""}${builder.workload_type === "CronJob" ? `<label><span>Schedule</span><input data-creator-mode="builder" data-key="schedule" value="${escapeHtml(builder.schedule)}" /></label><label><span>挂起</span><select data-creator-mode="builder" data-key="suspend"><option value="false" ${!builder.suspend ? "selected" : ""}>否</option><option value="true" ${builder.suspend ? "selected" : ""}>是</option></select></label><label><span>Concurrency Policy</span><select data-creator-mode="builder" data-key="concurrency_policy"><option value="Allow" ${builder.concurrency_policy === "Allow" ? "selected" : ""}>Allow</option><option value="Forbid" ${builder.concurrency_policy === "Forbid" ? "selected" : ""}>Forbid</option><option value="Replace" ${builder.concurrency_policy === "Replace" ? "selected" : ""}>Replace</option></select></label><label><span>Starting Deadline</span><input type="number" min="0" data-creator-mode="builder" data-key="starting_deadline_seconds" value="${escapeHtml(builder.starting_deadline_seconds)}" placeholder="0" /></label><label><span>Successful Jobs History</span><input type="number" min="0" data-creator-mode="builder" data-key="successful_jobs_history_limit" value="${escapeHtml(builder.successful_jobs_history_limit)}" placeholder="3" /></label><label><span>Failed Jobs History</span><input type="number" min="0" data-creator-mode="builder" data-key="failed_jobs_history_limit" value="${escapeHtml(builder.failed_jobs_history_limit)}" placeholder="1" /></label>` : ""}<label class="field-span-2"><span>Labels</span><textarea rows="4" data-creator-mode="builder" data-key="labels_text" placeholder="team=platform&#10;tier=backend">${escapeHtml(builder.labels_text)}</textarea></label><label class="field-span-2"><span>Annotations</span><textarea rows="4" data-creator-mode="builder" data-key="annotations_text" placeholder="prometheus.io/scrape=true">${escapeHtml(builder.annotations_text)}</textarea></label></div></div>`;
  }
  if (step === "containers") {
    return `<div class="creator-section"><div class="section-heading"><h4>容器组</h4><p>支持初始化容器和多个工作容器。命令、参数按行填写，环境变量按 <code>KEY=VALUE</code> 填写，卷挂载改成图形化配置。</p></div>${renderContainerGroup("init_containers", "初始化容器", builder.init_containers)}${renderContainerGroup("containers", "工作容器", builder.containers)}</div>`;
  }
  if (step === "storage") {
    return `<div class="creator-section"><div class="section-heading"><h4>存储挂载</h4><p>这里先定义 Pod 级卷。真正挂到哪个容器、挂载到哪里，在“容器组”里分别配置。StatefulSet 还可以额外定义 volumeClaimTemplates。</p></div><div class="action-row"><button class="ghost-button" data-action="creator-add-volume">新增卷</button></div>${builder.volumes.length ? builder.volumes.map((item, index) => renderVolumeCard(item, index)).join("") : emptyState("当前还没有卷定义")}${builder.workload_type === "StatefulSet" ? `<section class="creator-block"><div class="panel-header panel-header-nested"><div><h5>Volume Claim Templates</h5><p class="table-meta">给 StatefulSet 每个 Pod 自动生成独立 PVC，容器挂载时直接使用模板名称。</p></div><div class="action-row"><button class="ghost-button" data-action="creator-add-claim-template">新增模板</button></div></div>${builder.claim_templates.length ? builder.claim_templates.map((item, index) => renderClaimTemplateCard(item, index)).join("") : emptyState("当前还没有 volumeClaimTemplates")}</section>` : ""}</div>`;
  }
  if (step === "advanced") {
    const showServiceAccountName = builder.create_service_account;
    const showUpdateStrategy = ["Deployment", "StatefulSet"].includes(builder.workload_type);
    const showRollingControls = builder.update_strategy === "RollingUpdate";
    const showDNSConfig = builder.dns_policy === "None";
    const showImagePullSecrets = Boolean(String(builder.image_pull_secrets_text || "").trim());
    return `<div class="creator-section"><div class="section-heading"><h4>高级设置</h4><p>把更新策略、调度、DNS、主机别名和安全上下文收在这里，保持和 Kuboard 类似的分块方式。</p></div><div class="form-grid"><label><span>创建 ServiceAccount</span><select data-creator-mode="builder" data-key="create_service_account"><option value="false" ${!builder.create_service_account ? "selected" : ""}>否</option><option value="true" ${builder.create_service_account ? "selected" : ""}>是</option></select></label>${showServiceAccountName ? `<label><span>ServiceAccount 名称</span><input data-creator-mode="builder" data-key="service_account_name" value="${escapeHtml(builder.service_account_name)}" placeholder="留空则默认同名" /></label>` : ""}${showUpdateStrategy ? `<label><span>更新策略</span><select data-creator-mode="builder" data-key="update_strategy"><option value="">默认</option><option value="RollingUpdate" ${builder.update_strategy === "RollingUpdate" ? "selected" : ""}>RollingUpdate</option><option value="OnDelete" ${builder.update_strategy === "OnDelete" ? "selected" : ""}>OnDelete</option></select></label>` : ""}${showUpdateStrategy && showRollingControls ? `<label><span>Max Surge</span><input data-creator-mode="builder" data-key="max_surge" value="${escapeHtml(builder.max_surge)}" /></label><label><span>Max Unavailable</span><input data-creator-mode="builder" data-key="max_unavailable" value="${escapeHtml(builder.max_unavailable)}" /></label>` : ""}<label><span>DNS Policy</span><select data-creator-mode="builder" data-key="dns_policy"><option value="ClusterFirst" ${builder.dns_policy === "ClusterFirst" ? "selected" : ""}>ClusterFirst</option><option value="Default" ${builder.dns_policy === "Default" ? "selected" : ""}>Default</option><option value="ClusterFirstWithHostNet" ${builder.dns_policy === "ClusterFirstWithHostNet" ? "selected" : ""}>ClusterFirstWithHostNet</option><option value="None" ${builder.dns_policy === "None" ? "selected" : ""}>None</option></select></label><label><span>Termination Grace Period</span><input type="number" min="0" data-creator-mode="builder" data-key="termination_grace_period_seconds" value="${escapeHtml(builder.termination_grace_period_seconds)}" placeholder="30" /></label><label><span>Host Network</span><select data-creator-mode="builder" data-key="host_network"><option value="false" ${!builder.host_network ? "selected" : ""}>否</option><option value="true" ${builder.host_network ? "selected" : ""}>是</option></select></label><label><span>Host PID</span><select data-creator-mode="builder" data-key="host_pid"><option value="false" ${!builder.host_pid ? "selected" : ""}>否</option><option value="true" ${builder.host_pid ? "selected" : ""}>是</option></select></label><label><span>Host IPC</span><select data-creator-mode="builder" data-key="host_ipc"><option value="false" ${!builder.host_ipc ? "selected" : ""}>否</option><option value="true" ${builder.host_ipc ? "selected" : ""}>是</option></select></label><label><span>配置拉镜像密钥</span><select data-creator-mode="builder" data-key="image_pull_secrets_enabled"><option value="false" ${!showImagePullSecrets ? "selected" : ""}>否</option><option value="true" ${showImagePullSecrets ? "selected" : ""}>是</option></select></label>${showImagePullSecrets ? `<label><span>Image Pull Secrets</span><textarea rows="3" data-creator-mode="builder" data-key="image_pull_secrets_text" placeholder="regcred&#10;harbor-robot">${escapeHtml(builder.image_pull_secrets_text || "")}</textarea></label>` : ""}<div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Topology Spread</h5><p class="table-meta">按拓扑键分散 Pod，避免集中落到同一节点或可用区。</p></div><button class="ghost-button" data-action="creator-add-topology-spread">新增规则</button></div>${builder.topology_spread_constraints.length ? builder.topology_spread_constraints.map((item, index) => renderTopologySpreadRow(item, index)).join("") : emptyState("当前还没有 topology spread 规则")}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>节点选择器</h5><p class="table-meta">用键值对方式选择节点，不再手写整块文本。</p></div><button class="ghost-button" data-action="creator-add-node-selector">新增规则</button></div>${builder.node_selectors.length ? builder.node_selectors.map((item, index) => renderNodeSelectorRow(item, index)).join("") : emptyState("当前还没有节点选择器")}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>容忍度</h5><p class="table-meta">按字段配置 toleration，避免 YAML 缩进错误。</p></div><button class="ghost-button" data-action="creator-add-toleration">新增容忍度</button></div>${builder.tolerations.length ? builder.tolerations.map((item, index) => renderTolerationRow(item, index)).join("") : emptyState("当前还没有容忍度")}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>节点亲和</h5><p class="table-meta">优先做 nodeAffinity 的必选和优选规则，覆盖最常见的节点调度场景。</p></div></div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Required During Scheduling</h5></div><button class="ghost-button" data-action="creator-add-affinity" data-id="node_affinity_required">新增必选规则</button></div>${builder.node_affinity_required.length ? builder.node_affinity_required.map((item, index) => renderNodeAffinityRow("node_affinity_required", item, index)).join("") : emptyState("当前还没有必选 nodeAffinity")}</div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Preferred During Scheduling</h5></div><button class="ghost-button" data-action="creator-add-affinity" data-id="node_affinity_preferred">新增优选规则</button></div>${builder.node_affinity_preferred.length ? builder.node_affinity_preferred.map((item, index) => renderNodeAffinityRow("node_affinity_preferred", item, index)).join("") : emptyState("当前还没有优选 nodeAffinity")}</div></div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Pod Affinity</h5><p class="table-meta">按 labelSelector 和 topologyKey 配置工作负载之间的靠近调度。</p></div></div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Required</h5></div><button class="ghost-button" data-action="creator-add-pod-affinity" data-id="pod_affinity_required">新增必选规则</button></div>${builder.pod_affinity_required.length ? builder.pod_affinity_required.map((item, index) => renderPodAffinityRow("pod_affinity_required", item, index)).join("") : emptyState("当前还没有必选 podAffinity")}</div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Preferred</h5></div><button class="ghost-button" data-action="creator-add-pod-affinity" data-id="pod_affinity_preferred">新增优选规则</button></div>${builder.pod_affinity_preferred.length ? builder.pod_affinity_preferred.map((item, index) => renderPodAffinityRow("pod_affinity_preferred", item, index)).join("") : emptyState("当前还没有优选 podAffinity")}</div></div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Pod Anti-Affinity</h5><p class="table-meta">按 labelSelector 和 topologyKey 配置工作负载之间的分散调度。</p></div></div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Required</h5></div><button class="ghost-button" data-action="creator-add-pod-affinity" data-id="pod_anti_affinity_required">新增必选规则</button></div>${builder.pod_anti_affinity_required.length ? builder.pod_anti_affinity_required.map((item, index) => renderPodAffinityRow("pod_anti_affinity_required", item, index)).join("") : emptyState("当前还没有必选 podAntiAffinity")}</div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Preferred</h5></div><button class="ghost-button" data-action="creator-add-pod-affinity" data-id="pod_anti_affinity_preferred">新增优选规则</button></div>${builder.pod_anti_affinity_preferred.length ? builder.pod_anti_affinity_preferred.map((item, index) => renderPodAffinityRow("pod_anti_affinity_preferred", item, index)).join("") : emptyState("当前还没有优选 podAntiAffinity")}</div></div>${showDNSConfig ? `<div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>DNS Config</h5><p class="table-meta">把 nameservers、searches 和 options 拆开配置。</p></div><button class="ghost-button" data-action="creator-add-dns-option">新增 Option</button></div><div class="form-grid"><label><span>Nameservers</span><textarea rows="3" data-creator-mode="builder" data-key="dns_nameservers_text" placeholder="10.96.0.10&#10;1.1.1.1">${escapeHtml(builder.dns_nameservers_text || "")}</textarea></label><label><span>Searches</span><textarea rows="3" data-creator-mode="builder" data-key="dns_searches_text" placeholder="svc.cluster.local&#10;cluster.local">${escapeHtml(builder.dns_searches_text || "")}</textarea></label></div>${builder.dns_options.length ? builder.dns_options.map((item, index) => renderDNSOptionRow(item, index)).join("") : emptyState("当前还没有 DNS Options")}</div>` : ""}<div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>主机别名</h5><p class="table-meta">按 IP 和多个 hostname 组织，不再手写 YAML。</p></div><button class="ghost-button" data-action="creator-add-host-alias">新增别名</button></div>${builder.host_aliases.length ? builder.host_aliases.map((item, index) => renderHostAliasRow(item, index)).join("") : emptyState("当前还没有主机别名")}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>Pod Security Context</h5><p class="table-meta">先把最常用的运行用户和文件组配置成图形化字段。</p></div></div><div class="form-grid"><label><span>Run As Non Root</span><select data-creator-mode="builder" data-list="pod_security_context" data-key="run_as_non_root"><option value="false" ${!builder.pod_security_context.run_as_non_root ? "selected" : ""}>否</option><option value="true" ${builder.pod_security_context.run_as_non_root ? "selected" : ""}>是</option></select></label><label><span>Run As User</span><input data-creator-mode="builder" data-list="pod_security_context" data-key="run_as_user" value="${escapeHtml(builder.pod_security_context.run_as_user || "")}" placeholder="1000" /></label><label><span>Run As Group</span><input data-creator-mode="builder" data-list="pod_security_context" data-key="run_as_group" value="${escapeHtml(builder.pod_security_context.run_as_group || "")}" placeholder="1000" /></label><label><span>FS Group</span><input data-creator-mode="builder" data-list="pod_security_context" data-key="fs_group" value="${escapeHtml(builder.pod_security_context.fs_group || "")}" placeholder="2000" /></label></div></div></div>`;
  }
  return `<div class="creator-section"><div class="section-heading"><h4>服务暴露</h4><p>在这里决定是否创建 Service / Ingress，以及 Service 类型是否用 LoadBalancer。</p></div><div class="form-grid"><label><span>创建 Service</span><select data-creator-mode="builder" data-key="create_service"><option value="false" ${!builder.create_service ? "selected" : ""}>否</option><option value="true" ${builder.create_service ? "selected" : ""}>是</option></select></label>${builder.create_service ? `<label><span>Service 类型</span><select data-creator-mode="builder" data-key="service_type"><option value="ClusterIP" ${builder.service_type === "ClusterIP" ? "selected" : ""}>ClusterIP</option><option value="NodePort" ${builder.service_type === "NodePort" ? "selected" : ""}>NodePort</option><option value="LoadBalancer" ${builder.service_type === "LoadBalancer" ? "selected" : ""}>LoadBalancer</option></select></label><label><span>Service 端口</span><input type="number" min="1" data-creator-mode="builder" data-key="service_port" value="${escapeHtml(builder.service_port)}" /></label>` : ""}<label><span>创建 Ingress</span><select data-creator-mode="builder" data-key="create_ingress"><option value="false" ${!builder.create_ingress ? "selected" : ""}>否</option><option value="true" ${builder.create_ingress ? "selected" : ""}>是</option></select></label>${builder.create_ingress ? `<label><span>Ingress Host</span><input data-creator-mode="builder" data-key="ingress_host" value="${escapeHtml(builder.ingress_host)}" placeholder="demo.example.com" /></label><label><span>Ingress Path</span><input data-creator-mode="builder" data-key="ingress_path" value="${escapeHtml(builder.ingress_path)}" /></label><label><span>Ingress Class</span><input data-creator-mode="builder" data-key="ingress_class_name" value="${escapeHtml(builder.ingress_class_name)}" placeholder="nginx" /></label>` : ""}</div></div>`;
}

function renderResourceBuilder(resource, preset) {
  const summary = [
    { label: "类型", value: shortPresetLabel(preset) },
    { label: "名称", value: resource.name || "-" },
    { label: "命名空间", value: resource.namespace || "default" },
    { label: "集群", value: state.clusters.find((item) => String(item.id) === String(resource.cluster_id))?.name || "-" },
  ];
  return `<section class="creator-section"><div class="section-heading"><h4>${escapeHtml(shortPresetLabel(preset))} 图形化创建</h4><p>按资源概念块组织字段。提交时会转换成标准 Kubernetes YAML 并调用统一的 apply 接口。</p></div><div class="creator-resource-layout"><div class="creator-resource-form">${renderResourceBuilderForm(resource, preset)}</div><aside class="creator-resource-side"><div class="creator-card"><h5>创建摘要</h5><div class="detail-stack">${summary.map((item) => `<div class="detail-card"><span class="muted-label">${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong></div>`).join("")}</div></div></aside></div></section>`;
}

function renderResourceBuilderForm(resource, preset) {
  const base = `<div class="form-grid"><label><span>集群</span>${renderClusterSelect("resource-builder", resource.cluster_id)}</label><label><span>命名空间</span><input data-creator-mode="resource-builder" data-key="namespace" value="${escapeHtml(resource.namespace || "")}" placeholder="default" /></label><label><span>资源名称</span><input data-creator-mode="resource-builder" data-key="name" value="${escapeHtml(resource.name || "")}" /></label><label><span>资源类型</span><input value="${escapeHtml(shortPresetLabel(preset))}" disabled /></label><label class="field-span-2"><span>Labels</span><textarea rows="3" data-creator-mode="resource-builder" data-key="labels_text" placeholder="app=demo-app">${escapeHtml(resource.labels_text || "")}</textarea></label><label class="field-span-2"><span>Annotations</span><textarea rows="3" data-creator-mode="resource-builder" data-key="annotations_text" placeholder="owner=platform">${escapeHtml(resource.annotations_text || "")}</textarea></label></div>`;
  if (preset === "Service") {
    const externalIPsEnabled = Boolean(String(resource.service_external_ips_text || "").trim());
    return `${base}<div class="form-grid"><label><span>Service 类型</span><select data-creator-mode="resource-builder" data-key="service_type"><option value="ClusterIP" ${resource.service_type === "ClusterIP" ? "selected" : ""}>ClusterIP</option><option value="NodePort" ${resource.service_type === "NodePort" ? "selected" : ""}>NodePort</option><option value="LoadBalancer" ${resource.service_type === "LoadBalancer" ? "selected" : ""}>LoadBalancer</option></select></label><label><span>Session Affinity</span><select data-creator-mode="resource-builder" data-key="service_session_affinity"><option value="None" ${resource.service_session_affinity === "None" ? "selected" : ""}>None</option><option value="ClientIP" ${resource.service_session_affinity === "ClientIP" ? "selected" : ""}>ClientIP</option></select></label><label class="field-span-2"><span>Selector</span><textarea rows="3" data-creator-mode="resource-builder" data-key="service_selector_text" placeholder="app=demo-app&#10;tier=backend">${escapeHtml(resource.service_selector_text || "")}</textarea></label><label class="field-span-2"><span>端口映射</span><textarea rows="4" data-creator-mode="resource-builder" data-key="service_ports_text" placeholder="80:8080:TCP&#10;443:8443:TCP">${escapeHtml(resource.service_ports_text || "")}</textarea></label><label><span>配置 External IPs</span><select data-creator-mode="resource-builder" data-key="service_external_ips_enabled"><option value="false" ${!externalIPsEnabled ? "selected" : ""}>否</option><option value="true" ${externalIPsEnabled ? "selected" : ""}>是</option></select></label>${externalIPsEnabled ? `<label class="field-span-2"><span>External IPs</span><textarea rows="3" data-creator-mode="resource-builder" data-key="service_external_ips_text" placeholder="10.0.0.10">${escapeHtml(resource.service_external_ips_text || "")}</textarea></label>` : ""}</div>`;
  }
  if (preset === "Ingress") {
    const tlsEnabled = Boolean(String(resource.ingress_tls_secret_name || "").trim());
    return `${base}<div class="form-grid"><label><span>Ingress Class</span><input data-creator-mode="resource-builder" data-key="ingress_class_name" value="${escapeHtml(resource.ingress_class_name || "")}" placeholder="nginx" /></label><label><span>Host</span><input data-creator-mode="resource-builder" data-key="ingress_host" value="${escapeHtml(resource.ingress_host || "")}" placeholder="demo.local" /></label><label><span>Path</span><input data-creator-mode="resource-builder" data-key="ingress_path" value="${escapeHtml(resource.ingress_path || "")}" placeholder="/" /></label><label><span>后端 Service</span><input data-creator-mode="resource-builder" data-key="ingress_service_name" value="${escapeHtml(resource.ingress_service_name || "")}" placeholder="demo-service" /></label><label><span>后端端口</span><input data-creator-mode="resource-builder" data-key="ingress_service_port" value="${escapeHtml(resource.ingress_service_port || "")}" placeholder="80" /></label><label><span>启用 TLS</span><select data-creator-mode="resource-builder" data-key="ingress_tls_enabled"><option value="false" ${!tlsEnabled ? "selected" : ""}>否</option><option value="true" ${tlsEnabled ? "selected" : ""}>是</option></select></label>${tlsEnabled ? `<label><span>TLS Secret</span><input data-creator-mode="resource-builder" data-key="ingress_tls_secret_name" value="${escapeHtml(resource.ingress_tls_secret_name || "")}" placeholder="demo-tls" /></label>` : ""}</div>`;
  }
  if (preset === "ConfigMap") {
    return `${base}<div class="form-grid"><label class="field-span-2"><span>数据条目</span><textarea rows="10" data-creator-mode="resource-builder" data-key="config_entries_text" placeholder="APP_ENV=production&#10;LOG_LEVEL=info">${escapeHtml(resource.config_entries_text || "")}</textarea></label></div>`;
  }
  if (preset === "Secret") {
    return `${base}<div class="form-grid"><label><span>Secret 类型</span><select data-creator-mode="resource-builder" data-key="secret_type"><option value="Opaque" ${resource.secret_type === "Opaque" ? "selected" : ""}>Opaque</option><option value="kubernetes.io/dockerconfigjson" ${resource.secret_type === "kubernetes.io/dockerconfigjson" ? "selected" : ""}>dockerconfigjson</option></select></label><label class="field-span-2"><span>敏感条目</span><textarea rows="10" data-creator-mode="resource-builder" data-key="secret_entries_text" placeholder="username=admin&#10;password=change-me">${escapeHtml(resource.secret_entries_text || "")}</textarea></label></div>`;
  }
  if (preset === "PersistentVolumeClaim") {
    return `${base}<div class="form-grid"><label><span>StorageClass</span><input data-creator-mode="resource-builder" data-key="pvc_storage_class_name" value="${escapeHtml(resource.pvc_storage_class_name || "")}" placeholder="standard" /></label><label><span>容量</span><input data-creator-mode="resource-builder" data-key="pvc_size" value="${escapeHtml(resource.pvc_size || "")}" placeholder="5Gi" /></label><label class="field-span-2"><span>Access Modes</span><textarea rows="3" data-creator-mode="resource-builder" data-key="pvc_access_modes_text" placeholder="ReadWriteOnce">${escapeHtml(resource.pvc_access_modes_text || "")}</textarea></label></div>`;
  }
  if (preset === "ServiceAccount") {
    const pullSecretsEnabled = Boolean(String(resource.image_pull_secrets_text || "").trim());
    return `${base}<div class="form-grid"><label><span>配置拉镜像密钥</span><select data-creator-mode="resource-builder" data-key="image_pull_secrets_enabled"><option value="false" ${!pullSecretsEnabled ? "selected" : ""}>否</option><option value="true" ${pullSecretsEnabled ? "selected" : ""}>是</option></select></label>${pullSecretsEnabled ? `<label class="field-span-2"><span>Image Pull Secrets</span><textarea rows="4" data-creator-mode="resource-builder" data-key="image_pull_secrets_text" placeholder="regcred">${escapeHtml(resource.image_pull_secrets_text || "")}</textarea></label>` : ""}</div>`;
  }
  if (preset === "ResourceQuota") {
    return `${base}<div class="form-grid"><label class="field-span-2"><span>Hard Limits</span><textarea rows="8" data-creator-mode="resource-builder" data-key="quota_hard_text" placeholder="pods=10&#10;requests.cpu=2">${escapeHtml(resource.quota_hard_text || "")}</textarea></label></div>`;
  }
  return `${base}<div class="form-grid"><label class="field-span-2"><span>Limit Rules</span><textarea rows="10" data-creator-mode="resource-builder" data-key="limit_rules_text">${escapeHtml(resource.limit_rules_text || "")}</textarea></label></div>`;
}

function renderContainerGroup(listKey, title, items) {
  return `<section class="creator-block"><div class="panel-header panel-header-nested"><div><h5>${escapeHtml(title)}</h5></div><div class="action-row"><button class="ghost-button" data-action="creator-add-container" data-id="${listKey}">新增</button></div></div>${items.length ? items.map((item, index) => renderContainerCard(listKey, item, index)).join("") : emptyState(`当前还没有${title}`)}</section>`;
}

function renderContainerCard(listKey, item, index) {
  const mounts = item.volume_mounts || [];
  const envFromSources = item.env_from_sources || [];
  const envValueSources = item.env_value_sources || [];
  const postStartEnabled = hasMeaningfulText(item.lifecycle_post_start_text);
  const preStopEnabled = hasMeaningfulText(item.lifecycle_pre_stop_text);
  const envFromEnabled = envFromSources.length > 0;
  const envValueEnabled = envValueSources.length > 0;
  const resourcesEnabled = hasContainerResources(item);
  const securityContextEnabled = hasContainerSecurityContext(item.security_context);
  return `<article class="creator-card"><div class="panel-header panel-header-nested"><div><h5>${escapeHtml(item.name || `${listKey}-${index + 1}`)}</h5></div><button class="ghost-button danger-soft" data-action="creator-remove-container" data-id="${listKey}:${index}">删除</button></div><div class="form-grid"><label><span>容器名称</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="name" value="${escapeHtml(item.name || "")}" /></label><label><span>镜像</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="image" value="${escapeHtml(item.image || "")}" placeholder="nginx:1.27" /></label><label><span>拉镜像策略</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="image_pull_policy"><option value="IfNotPresent" ${item.image_pull_policy === "IfNotPresent" ? "selected" : ""}>IfNotPresent</option><option value="Always" ${item.image_pull_policy === "Always" ? "selected" : ""}>Always</option><option value="Never" ${item.image_pull_policy === "Never" ? "selected" : ""}>Never</option></select></label><label><span>端口</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="ports_text" value="${escapeHtml(item.ports_text || "")}" placeholder="80,8080" /></label><label><span>工作目录</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="working_dir" value="${escapeHtml(item.working_dir || "")}" placeholder="/app" /></label><label><span>STDIN</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="stdin"><option value="false" ${!item.stdin ? "selected" : ""}>否</option><option value="true" ${item.stdin ? "selected" : ""}>是</option></select></label><label><span>TTY</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="tty"><option value="false" ${!item.tty ? "selected" : ""}>否</option><option value="true" ${item.tty ? "selected" : ""}>是</option></select></label><label class="field-span-2"><span>环境变量</span><textarea rows="4" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="env_text" placeholder="APP_ENV=prod">${escapeHtml(item.env_text || "")}</textarea></label><label><span>命令</span><textarea rows="4" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="command_text" placeholder="/bin/sh&#10;-c">${escapeHtml(item.command_text || "")}</textarea></label><label><span>参数</span><textarea rows="4" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="args_text" placeholder="sleep 5">${escapeHtml(item.args_text || "")}</textarea></label><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>生命周期</h5><p class="table-meta">支持 postStart / preStop 的 exec 命令。</p></div></div><div class="form-grid"><label><span>启用 Post Start</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="lifecycle_post_start_enabled"><option value="false" ${!postStartEnabled ? "selected" : ""}>否</option><option value="true" ${postStartEnabled ? "selected" : ""}>是</option></select></label>${postStartEnabled ? `<label class="field-span-2"><span>Post Start</span><textarea rows="4" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="lifecycle_post_start_text" placeholder="/bin/sh&#10;-c&#10;echo started">${escapeHtml(item.lifecycle_post_start_text || "")}</textarea></label>` : ""}<label><span>启用 Pre Stop</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="lifecycle_pre_stop_enabled"><option value="false" ${!preStopEnabled ? "selected" : ""}>否</option><option value="true" ${preStopEnabled ? "selected" : ""}>是</option></select></label>${preStopEnabled ? `<label class="field-span-2"><span>Pre Stop</span><textarea rows="4" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="lifecycle_pre_stop_text" placeholder="/bin/sh&#10;-c&#10;sleep 5">${escapeHtml(item.lifecycle_pre_stop_text || "")}</textarea></label>` : ""}</div></div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>环境引用</h5><p class="table-meta">支持 envFrom 和 ConfigMap/Secret 单键注入。</p></div></div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>envFrom</h5></div><div class="action-row"><label class="compact-inline"><span>启用</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="env_from_enabled"><option value="false" ${!envFromEnabled ? "selected" : ""}>否</option><option value="true" ${envFromEnabled ? "selected" : ""}>是</option></select></label>${envFromEnabled ? `<button class="ghost-button" data-action="creator-add-env-from" data-id="${listKey}:${index}">新增来源</button>` : ""}</div></div>${envFromEnabled ? (envFromSources.length ? envFromSources.map((source, sourceIndex) => renderEnvFromRow(listKey, index, source, sourceIndex)).join("") : emptyState("当前还没有 envFrom 来源")) : ""}</div><div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>单键注入</h5></div><div class="action-row"><label class="compact-inline"><span>启用</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="env_value_sources_enabled"><option value="false" ${!envValueEnabled ? "selected" : ""}>否</option><option value="true" ${envValueEnabled ? "selected" : ""}>是</option></select></label>${envValueEnabled ? `<button class="ghost-button" data-action="creator-add-env-value-source" data-id="${listKey}:${index}">新增引用</button>` : ""}</div></div>${envValueEnabled ? (envValueSources.length ? envValueSources.map((source, sourceIndex) => renderEnvValueSourceRow(listKey, index, source, sourceIndex)).join("") : emptyState("当前还没有单键注入")) : ""}</div></div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>资源限制</h5><p class="table-meta">常用的 CPU / Memory requests 和 limits。</p></div></div><div class="form-grid"><label><span>启用资源限制</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="resources_enabled"><option value="false" ${!resourcesEnabled ? "selected" : ""}>否</option><option value="true" ${resourcesEnabled ? "selected" : ""}>是</option></select></label>${resourcesEnabled ? `<div class="field-span-2 creator-mount-row"><label><span>CPU Request</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="cpu_request" value="${escapeHtml(item.cpu_request || "")}" placeholder="100m" /></label><label><span>Memory Request</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="memory_request" value="${escapeHtml(item.memory_request || "")}" placeholder="128Mi" /></label><label><span>CPU Limit</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="cpu_limit" value="${escapeHtml(item.cpu_limit || "")}" placeholder="500m" /></label><label><span>Memory Limit</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="memory_limit" value="${escapeHtml(item.memory_limit || "")}" placeholder="512Mi" /></label></div>` : ""}</div></div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>健康检查</h5><p class="table-meta">支持 HTTP GET、TCP Socket 和 Exec，并单独提供 startupProbe。</p></div></div>${renderProbeCard(listKey, index, "startup_probe", "Startup Probe", item.startup_probe)}${renderProbeCard(listKey, index, "liveness_probe", "Liveness Probe", item.liveness_probe)}${renderProbeCard(listKey, index, "readiness_probe", "Readiness Probe", item.readiness_probe)}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>卷挂载</h5><p class="table-meta">每个容器单独选择要挂载的卷和路径。</p></div><button class="ghost-button" data-action="creator-add-mount" data-id="${listKey}:${index}">新增挂载</button></div>${mounts.length ? mounts.map((mount, mountIndex) => renderMountRow(listKey, index, mount, mountIndex)).join("") : emptyState("当前容器还没有卷挂载")}</div><div class="field-span-2 creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>容器安全上下文</h5><p class="table-meta">把最常用的能力提升、用户和 capabilities 做成图形化字段。</p></div></div><div class="form-grid"><label><span>启用安全上下文</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="security_context_enabled"><option value="false" ${!securityContextEnabled ? "selected" : ""}>否</option><option value="true" ${securityContextEnabled ? "selected" : ""}>是</option></select></label>${securityContextEnabled ? `<div class="field-span-2">${renderContainerSecurityContext(listKey, index, item.security_context)}</div>` : ""}</div></div></div></article>`;
}

function hasMeaningfulText(value) {
  return Boolean(String(value || "").trim());
}

function hasContainerResources(item) {
  return ["cpu_request", "memory_request", "cpu_limit", "memory_limit"].some((key) => hasMeaningfulText(item?.[key]));
}

function hasContainerSecurityContext(securityContext) {
  const current = securityContext || {};
  return Object.values(current).some((value) => {
    if (typeof value === "boolean") return value;
    return hasMeaningfulText(value);
  });
}

function renderEnvFromRow(listKey, containerIndex, source, sourceIndex) {
  return `<div class="creator-mount-row"><label><span>来源类型</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-from-index="${sourceIndex}" data-key="source_type"><option value="configMap" ${source.source_type === "configMap" ? "selected" : ""}>ConfigMap</option><option value="secret" ${source.source_type === "secret" ? "selected" : ""}>Secret</option></select></label><label><span>名称</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-from-index="${sourceIndex}" data-key="source_name" value="${escapeHtml(source.source_name || "")}" placeholder="app-config" /></label><div></div><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-env-from" data-id="${listKey}:${containerIndex}:${sourceIndex}">删除</button></div></div>`;
}

function renderEnvValueSourceRow(listKey, containerIndex, source, sourceIndex) {
  return `<div class="creator-toleration-row"><label><span>环境变量名</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-value-index="${sourceIndex}" data-key="env_name" value="${escapeHtml(source.env_name || "")}" placeholder="APP_TOKEN" /></label><label><span>来源类型</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-value-index="${sourceIndex}" data-key="source_type"><option value="configMap" ${source.source_type === "configMap" ? "selected" : ""}>ConfigMap</option><option value="secret" ${source.source_type === "secret" ? "selected" : ""}>Secret</option></select></label><label><span>资源名称</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-value-index="${sourceIndex}" data-key="source_name" value="${escapeHtml(source.source_name || "")}" placeholder="app-secret" /></label><label><span>Key</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-env-value-index="${sourceIndex}" data-key="source_key" value="${escapeHtml(source.source_key || "")}" placeholder="token" /></label><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-env-value-source" data-id="${listKey}:${containerIndex}:${sourceIndex}">删除</button></div></div>`;
}

function renderProbeCard(listKey, containerIndex, probeKey, title, probe) {
  const current = probe || createEmptyProbe();
  return `<div class="creator-mount-section"><div class="panel-header panel-header-nested"><div><h5>${title}</h5></div></div><div class="form-grid"><label><span>启用</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="enabled"><option value="false" ${!current.enabled ? "selected" : ""}>否</option><option value="true" ${current.enabled ? "selected" : ""}>是</option></select></label>${current.enabled ? `<label><span>类型</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="type"><option value="httpGet" ${current.type === "httpGet" ? "selected" : ""}>HTTP GET</option><option value="tcpSocket" ${current.type === "tcpSocket" ? "selected" : ""}>TCP Socket</option><option value="exec" ${current.type === "exec" ? "selected" : ""}>Exec</option></select></label>${current.type === "httpGet" ? `<label><span>Path</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="path" value="${escapeHtml(current.path || "")}" placeholder="/healthz" /></label><label><span>Port</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="port" value="${escapeHtml(current.port || "")}" placeholder="8080" /></label><label><span>Scheme</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="scheme"><option value="HTTP" ${current.scheme === "HTTP" ? "selected" : ""}>HTTP</option><option value="HTTPS" ${current.scheme === "HTTPS" ? "selected" : ""}>HTTPS</option></select></label>` : current.type === "tcpSocket" ? `<label><span>Host</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="host" value="${escapeHtml(current.host || "")}" placeholder="127.0.0.1" /></label><label><span>Port</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="port" value="${escapeHtml(current.port || "")}" placeholder="8080" /></label><div></div>` : `<label class="field-span-2"><span>Exec Command</span><textarea rows="3" data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="command_text" placeholder="/bin/sh&#10;-c&#10;curl -f http://127.0.0.1:8080/healthz">${escapeHtml(current.command_text || "")}</textarea></label>`}<label><span>Initial Delay</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="initial_delay_seconds" value="${escapeHtml(current.initial_delay_seconds || "")}" placeholder="5" /></label><label><span>Period</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="period_seconds" value="${escapeHtml(current.period_seconds || "")}" placeholder="10" /></label><label><span>Timeout</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="timeout_seconds" value="${escapeHtml(current.timeout_seconds || "")}" placeholder="2" /></label><label><span>Success Threshold</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="success_threshold" value="${escapeHtml(current.success_threshold || "")}" placeholder="1" /></label><label><span>Failure Threshold</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-probe-key="${probeKey}" data-key="failure_threshold" value="${escapeHtml(current.failure_threshold || "")}" placeholder="3" /></label>` : ""}</div></div>`;
}

function renderContainerSecurityContext(listKey, containerIndex, securityContext) {
  const current = securityContext || createEmptyContainerSecurityContext();
  return `<div class="form-grid"><label><span>Allow Privilege Escalation</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="allow_privilege_escalation"><option value="false" ${!current.allow_privilege_escalation ? "selected" : ""}>否</option><option value="true" ${current.allow_privilege_escalation ? "selected" : ""}>是</option></select></label><label><span>Privileged</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="privileged"><option value="false" ${!current.privileged ? "selected" : ""}>否</option><option value="true" ${current.privileged ? "selected" : ""}>是</option></select></label><label><span>Read Only Root FS</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="read_only_root_filesystem"><option value="false" ${!current.read_only_root_filesystem ? "selected" : ""}>否</option><option value="true" ${current.read_only_root_filesystem ? "selected" : ""}>是</option></select></label><label><span>Run As Non Root</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="run_as_non_root"><option value="false" ${!current.run_as_non_root ? "selected" : ""}>否</option><option value="true" ${current.run_as_non_root ? "selected" : ""}>是</option></select></label><label><span>Run As User</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="run_as_user" value="${escapeHtml(current.run_as_user || "")}" placeholder="101" /></label><label><span>Run As Group</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="run_as_group" value="${escapeHtml(current.run_as_group || "")}" placeholder="101" /></label><label><span>Capabilities Add</span><textarea rows="3" data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="capabilities_add_text" placeholder="NET_BIND_SERVICE">${escapeHtml(current.capabilities_add_text || "")}</textarea></label><label><span>Capabilities Drop</span><textarea rows="3" data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-security-context="true" data-key="capabilities_drop_text" placeholder="ALL">${escapeHtml(current.capabilities_drop_text || "")}</textarea></label></div>`;
}

function renderMountRow(listKey, containerIndex, mount, mountIndex) {
  const volumeChoices = [
    ...state.workloadCreator.builder.volumes.map((item) => ({
      value: item.name || "",
      label: item.name ? `${item.name} (${item.type || "volume"})` : "未命名卷",
    })),
    ...state.workloadCreator.builder.claim_templates.map((item) => ({
      value: item.name || "",
      label: item.name ? `${item.name} (claim template)` : "未命名模板",
    })),
  ];
  const volumeOptions = [`<option value="">选择卷</option>`, ...volumeChoices.map((item) => {
    const value = item.value;
    const label = item.label;
    return `<option value="${escapeHtml(value)}" ${value === String(mount.volume_name || "") ? "selected" : ""}>${escapeHtml(label)}</option>`;
  })].join("");
  return `<div class="creator-mount-row"><label><span>卷</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-mount-index="${mountIndex}" data-key="volume_name">${volumeOptions}</select></label><label><span>挂载路径</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-mount-index="${mountIndex}" data-key="mount_path" value="${escapeHtml(mount.mount_path || "")}" placeholder="/app/config" /></label><label><span>权限</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${containerIndex}" data-mount-index="${mountIndex}" data-key="read_only"><option value="false" ${!mount.read_only ? "selected" : ""}>读写</option><option value="true" ${mount.read_only ? "selected" : ""}>只读</option></select></label><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-mount" data-id="${listKey}:${containerIndex}:${mountIndex}">删除</button></div></div>`;
}

function renderNodeSelectorRow(item, index) {
  return `<div class="creator-mount-row"><label><span>Key</span><input data-creator-mode="builder" data-list="node_selectors" data-index="${index}" data-key="key" value="${escapeHtml(item.key || "")}" placeholder="node-role.kubernetes.io/worker" /></label><label><span>Value</span><input data-creator-mode="builder" data-list="node_selectors" data-index="${index}" data-key="value" value="${escapeHtml(item.value || "")}" placeholder="true" /></label><div></div><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-node-selector" data-id="${index}">删除</button></div></div>`;
}

function renderTolerationRow(item, index) {
  return `<div class="creator-toleration-row"><label><span>Key</span><input data-creator-mode="builder" data-list="tolerations" data-index="${index}" data-key="key" value="${escapeHtml(item.key || "")}" placeholder="node-role.kubernetes.io/control-plane" /></label><label><span>Operator</span><select data-creator-mode="builder" data-list="tolerations" data-index="${index}" data-key="operator"><option value="Equal" ${item.operator === "Equal" ? "selected" : ""}>Equal</option><option value="Exists" ${item.operator === "Exists" ? "selected" : ""}>Exists</option></select></label><label><span>Value</span><input data-creator-mode="builder" data-list="tolerations" data-index="${index}" data-key="value" value="${escapeHtml(item.value || "")}" placeholder="true" /></label><label><span>Effect</span><select data-creator-mode="builder" data-list="tolerations" data-index="${index}" data-key="effect"><option value="" ${!item.effect ? "selected" : ""}>未指定</option><option value="NoSchedule" ${item.effect === "NoSchedule" ? "selected" : ""}>NoSchedule</option><option value="PreferNoSchedule" ${item.effect === "PreferNoSchedule" ? "selected" : ""}>PreferNoSchedule</option><option value="NoExecute" ${item.effect === "NoExecute" ? "selected" : ""}>NoExecute</option></select></label><label><span>Toleration Seconds</span><input data-creator-mode="builder" data-list="tolerations" data-index="${index}" data-key="toleration_seconds" value="${escapeHtml(item.toleration_seconds || "")}" placeholder="3600" /></label><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-toleration" data-id="${index}">删除</button></div></div>`;
}

function renderHostAliasRow(item, index) {
  return `<div class="creator-host-alias-row"><label><span>IP</span><input data-creator-mode="builder" data-list="host_aliases" data-index="${index}" data-key="ip" value="${escapeHtml(item.ip || "")}" placeholder="10.0.0.10" /></label><label><span>主机名</span><textarea rows="3" data-creator-mode="builder" data-list="host_aliases" data-index="${index}" data-key="hostnames_text" placeholder="internal.demo.local&#10;cache.demo.local">${escapeHtml(item.hostnames_text || "")}</textarea></label><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-host-alias" data-id="${index}">删除</button></div></div>`;
}

function renderDNSOptionRow(item, index) {
  return `<div class="creator-mount-row"><label><span>Name</span><input data-creator-mode="builder" data-list="dns_options" data-index="${index}" data-key="name" value="${escapeHtml(item.name || "")}" placeholder="ndots" /></label><label><span>Value</span><input data-creator-mode="builder" data-list="dns_options" data-index="${index}" data-key="value" value="${escapeHtml(item.value || "")}" placeholder="2" /></label><div></div><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-dns-option" data-id="${index}">删除</button></div></div>`;
}

function renderNodeAffinityRow(listKey, item, index) {
  return `<div class="creator-affinity-row"><label><span>Key</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="key" value="${escapeHtml(item.key || "")}" placeholder="kubernetes.io/hostname" /></label><label><span>Operator</span><select data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="operator"><option value="In" ${item.operator === "In" ? "selected" : ""}>In</option><option value="NotIn" ${item.operator === "NotIn" ? "selected" : ""}>NotIn</option><option value="Exists" ${item.operator === "Exists" ? "selected" : ""}>Exists</option><option value="DoesNotExist" ${item.operator === "DoesNotExist" ? "selected" : ""}>DoesNotExist</option></select></label><label><span>Values</span><textarea rows="3" data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="values_text" placeholder="node-a&#10;node-b">${escapeHtml(item.values_text || "")}</textarea></label>${listKey === "node_affinity_preferred" ? `<label><span>Weight</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="weight" value="${escapeHtml(item.weight || "")}" placeholder="50" /></label>` : `<div></div>`}<div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-affinity" data-id="${listKey}:${index}">删除</button></div></div>`;
}

function renderPodAffinityRow(listKey, item, index) {
  return `<div class="creator-pod-affinity-row"><label><span>Label Key</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="label_key" value="${escapeHtml(item.label_key || "")}" placeholder="app" /></label><label><span>Label Value</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="label_value" value="${escapeHtml(item.label_value || "")}" placeholder="demo-app" /></label><label><span>Topology Key</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="topology_key" value="${escapeHtml(item.topology_key || "")}" placeholder="kubernetes.io/hostname" /></label>${listKey.endsWith("_preferred") ? `<label><span>Weight</span><input data-creator-mode="builder" data-list="${listKey}" data-index="${index}" data-key="weight" value="${escapeHtml(item.weight || "")}" placeholder="80" /></label>` : `<div></div>`}<div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-pod-affinity" data-id="${listKey}:${index}">删除</button></div></div>`;
}

function renderTopologySpreadRow(item, index) {
  return `<div class="creator-toleration-row"><label><span>Max Skew</span><input data-creator-mode="builder" data-list="topology_spread_constraints" data-index="${index}" data-key="max_skew" value="${escapeHtml(item.max_skew || "")}" placeholder="1" /></label><label><span>When Unsatisfiable</span><select data-creator-mode="builder" data-list="topology_spread_constraints" data-index="${index}" data-key="when_unsatisfiable"><option value="ScheduleAnyway" ${item.when_unsatisfiable === "ScheduleAnyway" ? "selected" : ""}>ScheduleAnyway</option><option value="DoNotSchedule" ${item.when_unsatisfiable === "DoNotSchedule" ? "selected" : ""}>DoNotSchedule</option></select></label><label><span>Topology Key</span><input data-creator-mode="builder" data-list="topology_spread_constraints" data-index="${index}" data-key="topology_key" value="${escapeHtml(item.topology_key || "")}" placeholder="kubernetes.io/hostname" /></label><label><span>Label Key</span><input data-creator-mode="builder" data-list="topology_spread_constraints" data-index="${index}" data-key="label_key" value="${escapeHtml(item.label_key || "")}" placeholder="app" /></label><label><span>Label Value</span><input data-creator-mode="builder" data-list="topology_spread_constraints" data-index="${index}" data-key="label_value" value="${escapeHtml(item.label_value || "")}" placeholder="demo-app" /></label><div class="creator-mount-actions"><button class="ghost-button danger-soft" data-action="creator-remove-topology-spread" data-id="${index}">删除</button></div></div>`;
}

function renderVolumeCard(item, index) {
  return `<article class="creator-card"><div class="panel-header panel-header-nested"><div><h5>${escapeHtml(item.name || `volume-${index + 1}`)}</h5><p class="table-meta">定义卷本身，容器里的挂载点到“容器组”里填写。</p></div><button class="ghost-button danger-soft" data-action="creator-remove-volume" data-id="${index}">删除</button></div><div class="form-grid"><label><span>卷名称</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="name" value="${escapeHtml(item.name || "")}" /></label><label><span>卷类型</span><select data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="type"><option value="configMap" ${item.type === "configMap" ? "selected" : ""}>ConfigMap</option><option value="pvc" ${item.type === "pvc" ? "selected" : ""}>PVC</option><option value="emptyDir" ${item.type === "emptyDir" ? "selected" : ""}>临时目录</option><option value="hostPath" ${item.type === "hostPath" ? "selected" : ""}>主机路径</option><option value="nfs" ${item.type === "nfs" ? "selected" : ""}>NFS</option><option value="secret" ${item.type === "secret" ? "selected" : ""}>密文</option></select></label>${item.type === "configMap" ? `<label><span>ConfigMap 名称</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="config_map_name" value="${escapeHtml(item.config_map_name || "")}" /></label><label class="field-span-2"><span>ConfigMap 数据</span><textarea rows="4" data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="config_map_data" placeholder="APP_ENV=prod">${escapeHtml(item.config_map_data || "")}</textarea></label>` : ""}${item.type === "secret" ? `<label><span>Secret 名称</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="secret_name" value="${escapeHtml(item.secret_name || "")}" /></label>` : ""}${item.type === "pvc" ? `<label><span>PVC 名称</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="pvc_name" value="${escapeHtml(item.pvc_name || "")}" /></label><label><span>PVC 大小</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="pvc_size" value="${escapeHtml(item.pvc_size || "")}" placeholder="1Gi" /></label><label><span>StorageClass</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="pvc_storage_class_name" value="${escapeHtml(item.pvc_storage_class_name || "")}" /></label>` : ""}${item.type === "hostPath" ? `<label class="field-span-2"><span>HostPath</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="host_path" value="${escapeHtml(item.host_path || "")}" /></label>` : ""}${item.type === "nfs" ? `<label><span>NFS Server</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="nfs_server" value="${escapeHtml(item.nfs_server || "")}" /></label><label><span>NFS Path</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="nfs_path" value="${escapeHtml(item.nfs_path || "")}" /></label>` : ""}${item.type === "emptyDir" ? `<label><span>EmptyDir Medium</span><input data-creator-mode="builder" data-list="volumes" data-index="${index}" data-key="empty_dir_medium" value="${escapeHtml(item.empty_dir_medium || "")}" placeholder="Memory" /></label>` : ""}</div></article>`;
}

function renderClaimTemplateCard(item, index) {
  return `<article class="creator-card"><div class="panel-header panel-header-nested"><div><h5>${escapeHtml(item.name || `claim-template-${index + 1}`)}</h5><p class="table-meta">模板名称会直接成为容器挂载里的卷名，每个 StatefulSet Pod 都会得到一份独立 PVC。</p></div><button class="ghost-button danger-soft" data-action="creator-remove-claim-template" data-id="${index}">删除</button></div><div class="form-grid"><label><span>模板名称</span><input data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="name" value="${escapeHtml(item.name || "")}" placeholder="data" /></label><label><span>容量</span><input data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="size" value="${escapeHtml(item.size || "")}" placeholder="10Gi" /></label><label><span>StorageClass</span><input data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="storage_class_name" value="${escapeHtml(item.storage_class_name || "")}" placeholder="standard" /></label><label><span>Access Modes</span><textarea rows="3" data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="access_modes_text" placeholder="ReadWriteOnce">${escapeHtml(item.access_modes_text || "")}</textarea></label><label class="field-span-2"><span>Labels</span><textarea rows="3" data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="labels_text" placeholder="tier=data">${escapeHtml(item.labels_text || "")}</textarea></label><label class="field-span-2"><span>Annotations</span><textarea rows="3" data-creator-mode="builder" data-list="claim_templates" data-index="${index}" data-key="annotations_text" placeholder="backup=true">${escapeHtml(item.annotations_text || "")}</textarea></label></div></article>`;
}

export function onWorkloadCreatorClick(event) {
  const button = event.target.closest("[data-action]");
  if (!button) return;
  const action = button.dataset.action;
  if (action === "creator-mode") {
    state.workloadCreator.mode = button.dataset.id;
    if (state.workloadCreator.mode === "builder" && !["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"].includes(state.workloadCreator.preset)) {
      state.workloadCreator.resourceBuilder = createDefaultResourceBuilderState(state.workloadCreator.preset);
    }
    if (state.workloadCreator.mode === "yaml") {
      state.workloadCreator.yaml = createDefaultYAMLState(ctx, state.workloadCreator.preset || "Deployment");
    }
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-preset-builder") {
    const preset = button.dataset.id || "Deployment";
    state.workloadCreator.preset = preset;
    state.workloadCreator.mode = "builder";
    state.workloadCreator.step = "basic";
    state.workloadCreator.builder = createDefaultBuilderState(ctx, preset);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-preset-resource") {
    const preset = button.dataset.id || "Service";
    state.workloadCreator.preset = preset;
    state.workloadCreator.mode = "builder";
    state.workloadCreator.resourceBuilder = createDefaultResourceBuilderState(preset);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-preset-yaml") {
    const preset = button.dataset.id || "Service";
    state.workloadCreator.preset = preset;
    state.workloadCreator.mode = "yaml";
    state.workloadCreator.yaml = createDefaultYAMLState(ctx, preset);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-step") {
    state.workloadCreator.step = button.dataset.id;
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-prev" || action === "creator-next") {
    const steps = ["basic", "containers", "storage", "advanced", "exposure"];
    const index = steps.indexOf(state.workloadCreator.step);
    state.workloadCreator.step = steps[Math.max(0, Math.min(steps.length - 1, index + (action === "creator-next" ? 1 : -1)))];
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-container") {
    const listKey = button.dataset.id;
    state.workloadCreator.builder[listKey].push(createEmptyContainer());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-container") {
    const [listKey, index] = button.dataset.id.split(":");
    state.workloadCreator.builder[listKey].splice(Number(index), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-mount") {
    const [listKey, containerIndex] = button.dataset.id.split(":");
    const container = state.workloadCreator.builder[listKey][Number(containerIndex)];
    if (!container.volume_mounts) container.volume_mounts = [];
    container.volume_mounts.push(createEmptyVolumeMount());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-mount") {
    const [listKey, containerIndex, mountIndex] = button.dataset.id.split(":");
    const mounts = state.workloadCreator.builder[listKey][Number(containerIndex)]?.volume_mounts || [];
    mounts.splice(Number(mountIndex), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-env-from") {
    const [listKey, containerIndex] = button.dataset.id.split(":");
    const container = state.workloadCreator.builder[listKey][Number(containerIndex)];
    if (!container.env_from_sources) container.env_from_sources = [];
    container.env_from_sources.push(createEmptyEnvFromSource());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-env-from") {
    const [listKey, containerIndex, sourceIndex] = button.dataset.id.split(":");
    const sources = state.workloadCreator.builder[listKey][Number(containerIndex)]?.env_from_sources || [];
    sources.splice(Number(sourceIndex), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-env-value-source") {
    const [listKey, containerIndex] = button.dataset.id.split(":");
    const container = state.workloadCreator.builder[listKey][Number(containerIndex)];
    if (!container.env_value_sources) container.env_value_sources = [];
    container.env_value_sources.push(createEmptyEnvValueSource());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-env-value-source") {
    const [listKey, containerIndex, sourceIndex] = button.dataset.id.split(":");
    const sources = state.workloadCreator.builder[listKey][Number(containerIndex)]?.env_value_sources || [];
    sources.splice(Number(sourceIndex), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-node-selector") {
    state.workloadCreator.builder.node_selectors.push(createEmptyNodeSelector());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-node-selector") {
    state.workloadCreator.builder.node_selectors.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-toleration") {
    state.workloadCreator.builder.tolerations.push(createEmptyToleration());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-topology-spread") {
    state.workloadCreator.builder.topology_spread_constraints.push(createEmptyTopologySpreadConstraint());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-topology-spread") {
    state.workloadCreator.builder.topology_spread_constraints.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-toleration") {
    state.workloadCreator.builder.tolerations.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-affinity") {
    const listKey = button.dataset.id;
    state.workloadCreator.builder[listKey].push(createEmptyNodeAffinityRule(listKey === "node_affinity_preferred" ? "50" : ""));
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-affinity") {
    const [listKey, index] = button.dataset.id.split(":");
    state.workloadCreator.builder[listKey].splice(Number(index), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-pod-affinity") {
    const listKey = button.dataset.id;
    state.workloadCreator.builder[listKey].push(createEmptyPodAffinityRule(listKey.endsWith("_preferred") ? "80" : ""));
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-pod-affinity") {
    const [listKey, index] = button.dataset.id.split(":");
    state.workloadCreator.builder[listKey].splice(Number(index), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-host-alias") {
    state.workloadCreator.builder.host_aliases.push(createEmptyHostAlias());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-host-alias") {
    state.workloadCreator.builder.host_aliases.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-dns-option") {
    state.workloadCreator.builder.dns_options.push(createEmptyDNSOption());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-dns-option") {
    state.workloadCreator.builder.dns_options.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-volume") {
    state.workloadCreator.builder.volumes.push({ name: "", type: "configMap", config_map_name: "", config_map_data: "", pvc_name: "", pvc_size: "", pvc_storage_class_name: "", secret_name: "", host_path: "", nfs_server: "", nfs_path: "", empty_dir_medium: "" });
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-add-claim-template") {
    state.workloadCreator.builder.claim_templates.push(createEmptyClaimTemplate());
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-volume") {
    state.workloadCreator.builder.volumes.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-remove-claim-template") {
    state.workloadCreator.builder.claim_templates.splice(Number(button.dataset.id), 1);
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-fill-template") {
    if (state.workloadCreator.mode === "builder") {
      state.workloadCreator.mode = "yaml";
    }
    if (isResourcePreset(state.workloadCreator.preset)) {
      const resource = state.workloadCreator.resourceBuilder;
      state.workloadCreator.yaml = {
        cluster_id: resource.cluster_id,
        namespace: resource.namespace,
        manifest_yaml: buildResourceBuilderManifest(resource, state.workloadCreator.preset),
      };
    } else {
      state.workloadCreator.yaml = createDefaultYAMLState(ctx, state.workloadCreator.preset || state.workloadCreator.builder?.workload_type || "Deployment");
    }
    renderWorkloadCreator();
    return;
  }
  if (action === "creator-submit-builder") {
    ctx.runK8sAction(onCreateWorkloadBundle);
    return;
  }
  if (action === "creator-submit-yaml") {
    ctx.runK8sAction(onApplyManifest);
  }
}

export function onWorkloadCreatorInput(event) {
  const target = event.target;
  if (!target.dataset.creatorMode) return;
  const mode = target.dataset.creatorMode;
  const listKey = target.dataset.list;
  const key = target.dataset.key;
  if (!key) return;
  if (mode === "yaml") {
    if (key === "preset_kind") {
      state.workloadCreator.preset = target.value;
      state.workloadCreator.yaml.manifest_yaml = buildManifestTemplate(target.value, state.workloadCreator.yaml.namespace);
      renderWorkloadCreator();
      return;
    }
    if (key === "namespace") {
      state.workloadCreator.yaml[key] = target.value;
      state.workloadCreator.yaml.manifest_yaml = buildManifestTemplate(state.workloadCreator.preset || "Deployment", target.value);
      return;
    }
    state.workloadCreator.yaml[key] = target.value;
    return;
  }
  if (mode === "resource-builder") {
    if (key === "service_external_ips_enabled") {
      state.workloadCreator.resourceBuilder.service_external_ips_text = target.value === "true"
        ? (state.workloadCreator.resourceBuilder.service_external_ips_text || "10.0.0.10")
        : "";
      renderWorkloadCreator();
      return;
    }
    if (key === "ingress_tls_enabled") {
      state.workloadCreator.resourceBuilder.ingress_tls_secret_name = target.value === "true"
        ? (state.workloadCreator.resourceBuilder.ingress_tls_secret_name || "demo-tls")
        : "";
      renderWorkloadCreator();
      return;
    }
    if (key === "image_pull_secrets_enabled") {
      state.workloadCreator.resourceBuilder.image_pull_secrets_text = target.value === "true"
        ? (state.workloadCreator.resourceBuilder.image_pull_secrets_text || "regcred")
        : "";
      renderWorkloadCreator();
      return;
    }
    state.workloadCreator.resourceBuilder[key] = normalizeCreatorValue(key, target.value);
    return;
  }
  if (listKey) {
    const index = Number(target.dataset.index);
    const mountIndex = target.dataset.mountIndex;
    const envFromIndex = target.dataset.envFromIndex;
    const envValueIndex = target.dataset.envValueIndex;
    const probeKey = target.dataset.probeKey;
    const securityContextMode = target.dataset.securityContext;
    const value = normalizeCreatorValue(key, target.value);
    if (mountIndex !== undefined) {
      const container = state.workloadCreator.builder[listKey][index];
      if (!container.volume_mounts) container.volume_mounts = [];
      container.volume_mounts[Number(mountIndex)][key] = value;
      container.volume_mounts_text = serializeVolumeMounts(container.volume_mounts);
      return;
    }
    if (envFromIndex !== undefined) {
      const container = state.workloadCreator.builder[listKey][index];
      if (!container.env_from_sources) container.env_from_sources = [];
      container.env_from_sources[Number(envFromIndex)][key] = value;
      container.env_from_text = serializeEnvFromSources(container.env_from_sources);
      return;
    }
    if (envValueIndex !== undefined) {
      const container = state.workloadCreator.builder[listKey][index];
      if (!container.env_value_sources) container.env_value_sources = [];
      container.env_value_sources[Number(envValueIndex)][key] = value;
      container.env_value_from_text = serializeEnvValueSources(container.env_value_sources);
      return;
    }
    if (probeKey) {
      const container = state.workloadCreator.builder[listKey][index];
      if (!container[probeKey]) container[probeKey] = createEmptyProbe();
      container[probeKey][key] = value;
      if (probeKey === "liveness_probe") {
        container.liveness_probe_yaml = serializeProbe(container.liveness_probe);
      }
      if (probeKey === "readiness_probe") {
        container.readiness_probe_yaml = serializeProbe(container.readiness_probe);
      }
      if (key === "type" || key === "enabled") {
        renderWorkloadCreator();
      }
      return;
    }
    if (securityContextMode) {
      const container = state.workloadCreator.builder[listKey][index];
      if (!container.security_context) container.security_context = createEmptyContainerSecurityContext();
      container.security_context[key] = value;
      container.security_context_yaml = serializeContainerSecurityContext(container.security_context);
      return;
    }
    if (listKey === "pod_security_context") {
      state.workloadCreator.builder.pod_security_context[key] = value;
      return;
    }
    const container = state.workloadCreator.builder[listKey][index];
    if (key === "lifecycle_post_start_enabled") {
      container.lifecycle_post_start_text = target.value === "true"
        ? (container.lifecycle_post_start_text || "/bin/sh\n-c\necho started")
        : "";
      renderWorkloadCreator();
      return;
    }
    if (key === "lifecycle_pre_stop_enabled") {
      container.lifecycle_pre_stop_text = target.value === "true"
        ? (container.lifecycle_pre_stop_text || "/bin/sh\n-c\nsleep 5")
        : "";
      renderWorkloadCreator();
      return;
    }
    if (key === "env_from_enabled") {
      container.env_from_sources = target.value === "true"
        ? (container.env_from_sources?.length ? container.env_from_sources : [createEmptyEnvFromSource()])
        : [];
      container.env_from_text = serializeEnvFromSources(container.env_from_sources);
      renderWorkloadCreator();
      return;
    }
    if (key === "env_value_sources_enabled") {
      container.env_value_sources = target.value === "true"
        ? (container.env_value_sources?.length ? container.env_value_sources : [createEmptyEnvValueSource()])
        : [];
      container.env_value_from_text = serializeEnvValueSources(container.env_value_sources);
      renderWorkloadCreator();
      return;
    }
    if (key === "resources_enabled") {
      if (target.value === "true") {
        container.cpu_request = container.cpu_request || "100m";
        container.memory_request = container.memory_request || "128Mi";
      } else {
        container.cpu_request = "";
        container.memory_request = "";
        container.cpu_limit = "";
        container.memory_limit = "";
      }
      renderWorkloadCreator();
      return;
    }
    if (key === "security_context_enabled") {
      container.security_context = target.value === "true"
        ? (hasContainerSecurityContext(container.security_context) ? container.security_context : createEmptyContainerSecurityContext())
        : createEmptyContainerSecurityContext();
      container.security_context_yaml = serializeContainerSecurityContext(container.security_context);
      renderWorkloadCreator();
      return;
    }
    container[key] = value;
    if ((listKey === "volumes" && key === "type") || (listKey === "claim_templates" && key === "name")) {
      renderWorkloadCreator();
    }
    return;
  }
  if (key === "image_pull_secrets_enabled") {
    state.workloadCreator.builder.image_pull_secrets_text = target.value === "true"
      ? (state.workloadCreator.builder.image_pull_secrets_text || "regcred")
      : "";
    renderWorkloadCreator();
    return;
  }
  state.workloadCreator.builder[key] = normalizeCreatorValue(key, target.value);
  if (["workload_type", "create_service_account", "update_strategy", "dns_policy", "create_service", "create_ingress"].includes(key)) {
    renderWorkloadCreator();
  }
}

async function onCreateWorkloadBundle() {
  if (isResourcePreset(state.workloadCreator.preset)) {
    await onCreateResourceBuilder();
    return;
  }
  const builder = state.workloadCreator.builder;
  const clusterID = Number(builder.cluster_id);
  const namespace = String(builder.namespace || "").trim();
  const containers = (builder.containers || []).map((item) => ({
    ...item,
    env_from_text: serializeEnvFromSources(item.env_from_sources),
    env_value_from_text: serializeEnvValueSources(item.env_value_sources),
    volume_mounts_text: serializeVolumeMounts(item.volume_mounts),
    liveness_probe_yaml: serializeProbe(item.liveness_probe),
    readiness_probe_yaml: serializeProbe(item.readiness_probe),
    startup_probe_yaml: serializeProbe(item.startup_probe),
    security_context_yaml: serializeContainerSecurityContext(item.security_context),
  }));
  const initContainers = (builder.init_containers || []).map((item) => ({
    ...item,
    env_from_text: serializeEnvFromSources(item.env_from_sources),
    env_value_from_text: serializeEnvValueSources(item.env_value_sources),
    volume_mounts_text: serializeVolumeMounts(item.volume_mounts),
    liveness_probe_yaml: serializeProbe(item.liveness_probe),
    readiness_probe_yaml: serializeProbe(item.readiness_probe),
    startup_probe_yaml: serializeProbe(item.startup_probe),
    security_context_yaml: serializeContainerSecurityContext(item.security_context),
  }));
  const nodeSelectorText = serializeNodeSelectors(builder.node_selectors);
  const tolerationsYAML = serializeTolerations(builder.tolerations);
  const dnsConfigYAML = serializeDNSConfig(builder);
  const hostAliasesYAML = serializeHostAliases(builder.host_aliases);
  const podSecurityContextYAML = serializePodSecurityContext(builder.pod_security_context);
  const affinityYAML = serializeAffinity(builder) || builder.affinity_yaml;
  const topologySpreadYAML = serializeTopologySpreadConstraints(builder.topology_spread_constraints);
  const payload = await api("/api/v1/k8s/workloads/bundle", {
    method: "POST",
    body: JSON.stringify({
      cluster_id: clusterID,
      namespace,
      workload_type: builder.workload_type,
      name: builder.name,
      image: builder.containers[0]?.image || "",
      labels_text: builder.labels_text,
      annotations_text: builder.annotations_text,
      replicas: Number(builder.replicas || 1),
      service_name: builder.service_name,
      pod_management_policy: builder.pod_management_policy,
      statefulset_service_mode: builder.statefulset_service_mode,
      statefulset_partition: Number(builder.statefulset_partition || 0),
      schedule: builder.schedule,
      suspend: Boolean(builder.suspend),
      concurrency_policy: builder.concurrency_policy,
      starting_deadline_seconds: Number(builder.starting_deadline_seconds || 0),
      successful_jobs_history_limit: Number(builder.successful_jobs_history_limit || 0),
      failed_jobs_history_limit: Number(builder.failed_jobs_history_limit || 0),
      containers,
      init_containers: initContainers,
      volumes: builder.volumes,
      claim_templates: builder.claim_templates,
      create_service_account: Boolean(builder.create_service_account),
      service_account_name: builder.service_account_name,
      update_strategy: builder.update_strategy,
      max_surge: builder.max_surge,
      max_unavailable: builder.max_unavailable,
      node_selector_text: nodeSelectorText,
      tolerations_yaml: tolerationsYAML,
      affinity_yaml: affinityYAML,
      dns_policy: builder.dns_policy,
      dns_config_yaml: dnsConfigYAML,
      host_aliases_yaml: hostAliasesYAML,
      pod_security_context_yaml: podSecurityContextYAML,
      host_network: Boolean(builder.host_network),
      host_pid: Boolean(builder.host_pid),
      host_ipc: Boolean(builder.host_ipc),
      image_pull_secrets_text: builder.image_pull_secrets_text,
      topology_spread_yaml: topologySpreadYAML,
      termination_grace_period_seconds: Number(builder.termination_grace_period_seconds || 0),
      create_service: Boolean(builder.create_service),
      service_type: builder.service_type,
      service_port: Number(builder.service_port || 0),
      create_ingress: Boolean(builder.create_ingress),
      ingress_host: builder.ingress_host,
      ingress_path: builder.ingress_path,
      ingress_class_name: builder.ingress_class_name,
    }),
  });
  state.business.selectedClusterID = clusterID;
  state.business.selectedNamespace = namespace;
  closeDrawer("workload");
  toast(payload.data.message ? `${payload.data.message}，资源 ${payload.data.created_kinds.join(" / ")}` : "工作负载已创建");
  await Promise.all([
    ctx.loadClusters(),
    ctx.loadNamespaces(clusterID),
    ctx.loadWorkloads(clusterID, namespace),
    ctx.loadResources(clusterID, namespace, ctx.selectedResourceKind()),
  ]);
  await focusCreatedResources(payload.data.created_resources || [], clusterID, namespace);
}

async function onCreateResourceBuilder() {
  const resource = state.workloadCreator.resourceBuilder;
  const clusterID = Number(resource.cluster_id);
  const namespace = String(resource.namespace || "").trim();
  const payload = await api("/api/v1/k8s/manifests/apply", {
    method: "POST",
    body: JSON.stringify({
      cluster_id: clusterID,
      namespace,
      manifest_yaml: buildResourceBuilderManifest(resource, state.workloadCreator.preset),
    }),
  });
  state.business.selectedClusterID = clusterID;
  state.business.selectedNamespace = namespace;
  closeDrawer("workload");
  toast(payload.data.message ? `${payload.data.message}，共 ${payload.data.document_count} 个文档` : "资源已创建");
  await Promise.all([
    ctx.loadClusters(),
    ctx.loadNamespaces(clusterID),
    ctx.loadWorkloads(clusterID, namespace),
    ctx.loadResources(clusterID, namespace, state.workloadCreator.preset),
  ]);
  await focusCreatedResources(payload.data.created_resources || [], clusterID, namespace);
}

async function onApplyManifest() {
  const yamlState = state.workloadCreator.yaml;
  const clusterID = Number(yamlState.cluster_id);
  const namespace = String(yamlState.namespace || "").trim();
  const payload = await api("/api/v1/k8s/manifests/apply", {
    method: "POST",
    body: JSON.stringify({
      cluster_id: clusterID,
      namespace,
      manifest_yaml: yamlState.manifest_yaml,
    }),
  });
  state.business.selectedClusterID = clusterID;
  if (namespace) {
    state.business.selectedNamespace = namespace;
  }
  closeDrawer("workload");
  toast(payload.data.message ? `${payload.data.message}，共 ${payload.data.document_count} 个文档` : "YAML 已应用");
  await Promise.all([
    ctx.loadClusters(),
    ctx.loadNamespaces(clusterID),
    ctx.loadWorkloads(clusterID, ctx.selectedWorkloadNamespace()),
    ctx.loadResources(clusterID, ctx.selectedResourceNamespace(), ctx.selectedResourceKind()),
  ]);
  await focusCreatedResources(payload.data.created_resources || [], clusterID, namespace || ctx.selectedResourceNamespace() || ctx.selectedWorkloadNamespace());
}

function buildResourceBuilderManifest(resource, preset) {
  const labels = parseKeyValueText(resource.labels_text);
  const annotations = parseKeyValueText(resource.annotations_text);
  const metadata = {
    name: resource.name,
    namespace: resource.namespace || "default",
  };
  if (Object.keys(labels).length) metadata.labels = labels;
  if (Object.keys(annotations).length) metadata.annotations = annotations;
  let doc = null;
  switch (preset) {
    case "Service":
      doc = {
        apiVersion: "v1",
        kind: "Service",
        metadata,
        spec: {
          type: resource.service_type || "ClusterIP",
          sessionAffinity: resource.service_session_affinity || "None",
          selector: parseKeyValueText(resource.service_selector_text),
          ports: parseServicePorts(resource.service_ports_text),
        },
      };
      if (String(resource.service_external_ips_text || "").trim()) {
        doc.spec.externalIPs = splitLines(resource.service_external_ips_text);
      }
      break;
    case "Ingress":
      doc = {
        apiVersion: "networking.k8s.io/v1",
        kind: "Ingress",
        metadata,
        spec: {
          ingressClassName: resource.ingress_class_name || undefined,
          rules: [{
            host: resource.ingress_host || "",
            http: {
              paths: [{
                path: resource.ingress_path || "/",
                pathType: "Prefix",
                backend: {
                  service: {
                    name: resource.ingress_service_name || "demo-service",
                    port: normalizeServicePortValue(resource.ingress_service_port || "80"),
                  },
                },
              }],
            },
          }],
        },
      };
      if (resource.ingress_tls_secret_name) {
        doc.spec.tls = [{ secretName: resource.ingress_tls_secret_name, hosts: [resource.ingress_host || ""] }];
      }
      break;
    case "ConfigMap":
      doc = {
        apiVersion: "v1",
        kind: "ConfigMap",
        metadata,
        data: parseKeyValueText(resource.config_entries_text),
      };
      break;
    case "Secret":
      doc = {
        apiVersion: "v1",
        kind: "Secret",
        metadata,
        type: resource.secret_type || "Opaque",
        stringData: parseKeyValueText(resource.secret_entries_text),
      };
      break;
    case "PersistentVolumeClaim":
      doc = {
        apiVersion: "v1",
        kind: "PersistentVolumeClaim",
        metadata,
        spec: {
          accessModes: splitLines(resource.pvc_access_modes_text),
          resources: { requests: { storage: resource.pvc_size || "5Gi" } },
        },
      };
      if (resource.pvc_storage_class_name) {
        doc.spec.storageClassName = resource.pvc_storage_class_name;
      }
      break;
    case "ServiceAccount":
      doc = {
        apiVersion: "v1",
        kind: "ServiceAccount",
        metadata,
      };
      if (String(resource.image_pull_secrets_text || "").trim()) {
        doc.imagePullSecrets = splitLines(resource.image_pull_secrets_text).map((name) => ({ name }));
      }
      break;
    case "ResourceQuota":
      doc = {
        apiVersion: "v1",
        kind: "ResourceQuota",
        metadata,
        spec: { hard: parseKeyValueText(resource.quota_hard_text) },
      };
      break;
    case "LimitRange":
      return `apiVersion: v1\nkind: LimitRange\nmetadata:\n${indentYamlBlock(serializeMetadata(metadata), 2)}\nspec:\n  limits:\n${indentMultiline(resource.limit_rules_text || "", 4)}`;
    default:
      throw new Error(`unsupported resource builder preset: ${preset}`);
  }
  return yamlFromObject(doc);
}

function parseKeyValueText(text) {
  return splitLines(text).reduce((acc, line) => {
    const separator = line.includes("=") ? "=" : ":";
    const idx = line.indexOf(separator);
    if (idx <= 0) return acc;
    const key = line.slice(0, idx).trim();
    const value = line.slice(idx + 1).trim();
    if (key) acc[key] = value;
    return acc;
  }, {});
}

function splitLines(text) {
  return String(text || "").split("\n").map((line) => line.trim()).filter(Boolean);
}

function parseServicePorts(text) {
  return splitLines(text).map((line, index) => {
    const [port, targetPort, protocol] = line.split(":").map((item) => item.trim());
    return {
      name: `port-${index + 1}`,
      port: Number(port || 80),
      targetPort: /^\d+$/.test(String(targetPort || "")) ? Number(targetPort) : (targetPort || Number(port || 80)),
      protocol: protocol || "TCP",
    };
  });
}

function normalizeServicePortValue(value) {
  return /^\d+$/.test(String(value)) ? { number: Number(value) } : { name: String(value) };
}

function yamlFromObject(value, level = 0) {
  const indent = "  ".repeat(level);
  if (Array.isArray(value)) {
    return value.map((item) => {
      if (typeof item === "object" && item !== null) {
        const nested = yamlFromObject(item, level + 1);
        const lines = nested.split("\n");
        return `${indent}- ${lines[0].trimStart()}\n${lines.slice(1).map((line) => `${indent}  ${line.trimStart()}`).join("\n")}`;
      }
      return `${indent}- ${String(item)}`;
    }).join("\n");
  }
  return Object.entries(value || {}).filter(([, val]) => val !== undefined && val !== null && val !== "").map(([key, val]) => {
    if (Array.isArray(val)) {
      return val.length ? `${indent}${key}:\n${yamlFromObject(val, level + 1)}` : `${indent}${key}: []`;
    }
    if (typeof val === "object") {
      const nested = yamlFromObject(val, level + 1);
      return nested ? `${indent}${key}:\n${nested}` : `${indent}${key}: {}`;
    }
    return `${indent}${key}: ${formatYamlScalar(val)}`;
  }).join("\n");
}

function formatYamlScalar(value) {
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (/^[A-Za-z0-9._/-]+$/.test(String(value))) return String(value);
  return JSON.stringify(String(value));
}

function serializeMetadata(metadata) {
  return yamlFromObject(metadata);
}

function indentYamlBlock(text, spaces) {
  return String(text || "").split("\n").map((line) => `${" ".repeat(spaces)}${line}`).join("\n");
}

function indentMultiline(text, spaces) {
  return splitLinesWithEmpty(text).map((line) => `${" ".repeat(spaces)}${line}`).join("\n");
}

function splitLinesWithEmpty(text) {
  return String(text || "").split("\n").filter((line) => line.trim().length > 0);
}

async function focusCreatedResources(createdResources, clusterID, fallbackNamespace) {
  if (!Array.isArray(createdResources) || !createdResources.length || !clusterID) {
    return;
  }
  const workloadKinds = ["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"];
  const resourceKinds = ["Service", "Ingress", "ConfigMap", "Secret", "ServiceAccount", "PersistentVolumeClaim", "ResourceQuota", "LimitRange"];
  const workloadTarget = createdResources.find((item) => workloadKinds.includes(item.kind));
  if (workloadTarget) {
    const namespace = String(workloadTarget.namespace || fallbackNamespace || "").trim();
    state.business.selectedClusterID = clusterID;
    state.business.selectedNamespace = namespace;
    ctx.renderClusterSelectors();
    await ctx.loadWorkloads(clusterID, namespace);
    const workload = state.workloads.find((item) => item.kind === workloadTarget.kind && item.name === workloadTarget.name && item.namespace_name === namespace);
    if (workload) {
      ctx.switchBusinessView("workloads");
      await ctx.selectWorkload(workload.id);
      return;
    }
  }
  const resourceTarget = createdResources.find((item) => resourceKinds.includes(item.kind));
  if (resourceTarget) {
    const namespace = String(resourceTarget.namespace || fallbackNamespace || "").trim();
    state.business.selectedClusterID = clusterID;
    state.business.selectedNamespace = namespace;
    state.business.selectedResourceKind = resourceTarget.kind;
    ctx.renderClusterSelectors();
    elements.resourceKindFilter.value = resourceTarget.kind;
    await ctx.loadResources(clusterID, namespace, resourceTarget.kind);
    if (state.resourceExplorer.items.some((item) => item.name === resourceTarget.name)) {
      ctx.switchBusinessView("resources");
      await ctx.selectResource(resourceTarget.name);
    }
  }
}
