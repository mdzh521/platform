import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { bindAction } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import {
  openInteractiveTerminalModal,
  openExecCommandModal,
  openScaleWorkloadModal,
  openStatefulSetSettingsModal,
  openUpdateImageModal,
  restartWorkload,
  rolloutRestartWorkload,
} from "./workload-ops.js";

let ctx = null;

export function configureWorkloadDetailPanel(deps) {
  ctx = deps;
}

export function renderWorkloadDetailPanel() {
  switchWorkloadPage(state.business.workloadListPage);
  const workload = state.workloads.find((item) => item.id === state.business.selectedWorkloadID);
  if (!workload) {
    elements.workloadDetailPanel.innerHTML = emptyState("选择一个工作负载后，这里会展示详细信息");
    return;
  }
  const detail = state.workloadInspector.detail;
  const inspectorTabs = [
    `<button class="ghost-button ${state.business.workloadInspectorView === "overview" ? "active" : ""}" data-action="inspector-overview">基础信息</button>`,
    supportsWorkloadRollout(workload.kind) ? `<button class="ghost-button ${state.business.workloadInspectorView === "rollout" ? "active" : ""}" data-action="inspector-rollout">发布状态</button>` : "",
    `<button class="ghost-button ${state.business.workloadInspectorView === "resources" ? "active" : ""}" data-action="inspector-resources">关联资源</button>`,
    `<button class="ghost-button ${state.business.workloadInspectorView === "events" ? "active" : ""}" data-action="inspector-events">事件</button>`,
    supportsWorkloadLogs(workload.kind) ? `<button class="ghost-button ${state.business.workloadInspectorView === "logs" ? "active" : ""}" data-action="inspector-logs">日志</button>` : "",
    `<button class="ghost-button ${state.business.workloadInspectorView === "manifest" ? "active" : ""}" data-action="inspector-manifest">YAML</button>`,
    state.business.workloadInspectorView === "manifest"
      ? `<button class="ghost-button ${state.business.workloadManifestMode === "compact" ? "active" : ""}" data-action="manifest-compact">简洁模式</button><button class="ghost-button ${state.business.workloadManifestMode === "full" ? "active" : ""}" data-action="manifest-full">完整模式</button>`
      : "",
  ].join("");
  const workloadMeta = [
    { label: "命名空间", value: workload.namespace_name || "-" },
    { label: "类型", value: workload.kind },
    { label: "副本", value: workloadReplicaLabel(workload) },
    { label: "状态", value: workload.status || "-" },
    { label: "镜像", value: workload.image || "-" },
    { label: "更新时间", value: formatDateTime(workload.updated_at) },
  ];
  const mutateActions = [
    supportsWorkloadScale(workload.kind) ? `<button class="primary-button" data-action="scale-workload">扩缩容</button>` : "",
    supportsWorkloadImageUpdate(workload.kind) ? `<button class="ghost-button" data-action="update-workload-image">更新镜像</button>` : "",
    supportsStatefulSetSettings(workload.kind) ? `<button class="ghost-button" data-action="update-statefulset-settings">StatefulSet 设置</button>` : "",
  ].join("");
  const restartActions = [
    supportsWorkloadRestart(workload.kind) ? `<button class="ghost-button" data-action="restart-workload">重启 Pod</button>` : "",
    supportsWorkloadRolloutRestart(workload.kind) ? `<button class="ghost-button" data-action="rollout-restart">Rollout Restart</button>` : "",
  ].join("");
  const dangerActions = `<button class="ghost-button danger-soft" data-action="delete-selected-workload">删除工作负载</button>`;
  const operationSections = [
    mutateActions ? `<div class="operation-group"><p class="muted-label">变更</p><div class="operation-row">${mutateActions}</div></div>` : "",
    restartActions ? `<div class="operation-group"><p class="muted-label">运维</p><div class="operation-row">${restartActions}</div></div>` : "",
    `<div class="operation-group operation-group-danger"><p class="muted-label">清理</p><div class="operation-row">${dangerActions}</div></div>`,
  ].join("");
  elements.workloadDetailPanel.innerHTML = `<div class="detail-stack"><div class="detail-heading"><div><p class="eyebrow">${escapeHtml(workload.kind)}</p><h4>${escapeHtml(workload.name)}</h4></div><div class="detail-meta-row">${workloadMeta.map((item) => `<span class="meta-pill"><span>${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong></span>`).join("")}</div></div><div class="detail-toolbar-shell"><div class="detail-toolbar">${inspectorTabs}</div></div><section class="operation-panel"><div class="operation-panel-head"><div><p class="eyebrow">Operations</p><h5>工作负载运维</h5></div><p class="table-meta">常用变更和运维动作集中在这里，详情标签只负责查看信息。</p></div><div class="operation-grid">${operationSections}</div></section><div>${renderWorkloadInspectorContent(workload, detail)}</div></div>`;
  bindAction(elements.workloadDetailPanel, "inspector-overview", () => ctx.switchWorkloadInspector("overview"));
  bindAction(elements.workloadDetailPanel, "inspector-rollout", () => ctx.runK8sAction(() => ctx.switchWorkloadInspector("rollout")));
  bindAction(elements.workloadDetailPanel, "inspector-resources", () => ctx.runK8sAction(() => ctx.switchWorkloadInspector("resources")));
  bindAction(elements.workloadDetailPanel, "inspector-events", () => ctx.runK8sAction(() => ctx.switchWorkloadInspector("events")));
  bindAction(elements.workloadDetailPanel, "inspector-logs", () => ctx.runK8sAction(() => ctx.switchWorkloadInspector("logs")));
  bindAction(elements.workloadDetailPanel, "inspector-manifest", () => ctx.runK8sAction(() => ctx.switchWorkloadInspector("manifest")));
  bindAction(elements.workloadDetailPanel, "manifest-compact", () => ctx.runK8sAction(() => ctx.switchManifestMode("compact")));
  bindAction(elements.workloadDetailPanel, "manifest-full", () => ctx.runK8sAction(() => ctx.switchManifestMode("full")));
  bindAction(elements.workloadDetailPanel, "toggle-log-follow", () => ctx.runK8sAction(() => ctx.toggleLogFollow()));
  bindAction(elements.workloadDetailPanel, "refresh-workload-logs", () => ctx.runK8sAction(() => ctx.refreshWorkloadLogs()));
  bindAction(elements.workloadDetailPanel, "select-log-pod", (podName) => ctx.runK8sAction(() => ctx.switchLogPod(String(podName))));
  bindAction(elements.workloadDetailPanel, "open-service-detail", (name) => ctx.runK8sAction(() => ctx.openResourceDetail("service", String(name))));
  bindAction(elements.workloadDetailPanel, "open-ingress-detail", (name) => ctx.runK8sAction(() => ctx.openResourceDetail("ingress", String(name))));
  bindAction(elements.workloadDetailPanel, "open-linked-namespace-resource", (payload) => ctx.runK8sAction(() => ctx.openLinkedNamespaceResource(String(payload))));
  bindAction(elements.workloadDetailPanel, "delete-linked-service", (name) => ctx.runK8sAction(() => ctx.deleteNamespaceResource("Service", String(name))));
  bindAction(elements.workloadDetailPanel, "delete-linked-ingress", (name) => ctx.runK8sAction(() => ctx.deleteNamespaceResource("Ingress", String(name))));
  bindAction(elements.workloadDetailPanel, "scale-workload", () => ctx.runK8sAction(() => openScaleWorkloadModal(workload)));
  bindAction(elements.workloadDetailPanel, "update-workload-image", () => ctx.runK8sAction(() => openUpdateImageModal(workload)));
  bindAction(elements.workloadDetailPanel, "update-statefulset-settings", () => ctx.runK8sAction(() => openStatefulSetSettingsModal(workload)));
  bindAction(elements.workloadDetailPanel, "restart-workload", () => ctx.runK8sAction(() => restartWorkload(workload)));
  bindAction(elements.workloadDetailPanel, "rollout-restart", () => ctx.runK8sAction(() => rolloutRestartWorkload(workload)));
  bindAction(elements.workloadDetailPanel, "delete-selected-workload", () => ctx.runK8sAction(() => ctx.deleteSelectedWorkload(workload)));
  bindAction(elements.workloadDetailPanel, "open-interactive-terminal-pod", (podName) => ctx.runK8sAction(() => openInteractiveTerminalModal(workload, String(podName))));
  bindAction(elements.workloadDetailPanel, "exec-workload-command-pod", (podName) => ctx.runK8sAction(() => openExecCommandModal(workload, String(podName))));
  bindInteractiveTerminalBindings();
}

export function switchWorkloadPage(view) {
  state.business.workloadListPage = view;
  elements.workloadListPage.classList.remove("hidden");
  elements.workloadDetailPage.classList.toggle("hidden", view !== "detail");
}

function workloadReplicaLabel(item) {
  switch (item.kind) {
    case "Deployment":
    case "StatefulSet":
    case "DaemonSet":
      return `${item.ready_replicas}/${item.replicas}`;
    case "Job":
      return `完成 ${item.ready_replicas}/${item.replicas}`;
    case "CronJob":
      return `活跃 ${item.ready_replicas}/${item.replicas}`;
    default:
      return `${item.ready_replicas}/${item.replicas}`;
  }
}

function supportsWorkloadRollout(kind) {
  return ["Deployment", "StatefulSet"].includes(kind);
}

function supportsWorkloadScale(kind) {
  return ["Deployment", "StatefulSet"].includes(kind);
}

function supportsWorkloadImageUpdate(kind) {
  return ["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"].includes(kind);
}

function supportsStatefulSetSettings(kind) {
  return kind === "StatefulSet";
}

function supportsWorkloadRestart(kind) {
  return ["Deployment", "StatefulSet", "DaemonSet", "Job", "Pod"].includes(kind);
}

function supportsWorkloadRolloutRestart(kind) {
  return ["Deployment", "StatefulSet"].includes(kind);
}

function supportsWorkloadLogs(kind) {
  return ["Deployment", "StatefulSet", "DaemonSet", "Job", "Pod"].includes(kind);
}

function supportsWorkloadExec(kind) {
  return ["Deployment", "StatefulSet", "DaemonSet", "Job", "Pod"].includes(kind);
}

function renderWorkloadInspectorContent(workload, detail) {
  const activeView = state.business.workloadInspectorView;
  if (activeView === "events") {
    if (!state.workloadInspector.events.length) {
      return emptyState("当前工作负载暂无事件，或还未同步到可见事件");
    }
    return `<div class="detail-stack">${renderEventDiagnosis()}<div class="event-list">${state.workloadInspector.events.map((item) => `<article class="event-item"><p class="muted-label">${escapeHtml(item.type || "-")} / ${escapeHtml(item.component || "kubernetes")}</p><strong>${escapeHtml(item.reason || "-")}</strong><p>${escapeHtml(item.message || "-")}</p><p class="table-meta">次数 ${escapeHtml(item.count || 1)}，时间 ${escapeHtml(formatDateTime(item.timestamp))}</p></article>`).join("")}</div></div>`;
  }
  if (activeView === "rollout") return renderRolloutInspector(workload);
  if (activeView === "resources") return renderResourcesInspector();
  if (activeView === "logs") {
    return `<div class="detail-stack">${renderLogToolbar()}${renderFoldSection("日志来源", `<p class="table-meta">${escapeHtml(state.workloadInspector.logSourcePod ? `当前显示 Pod ${state.workloadInspector.logSourcePod} 的日志` : "当前显示工作负载关联 Pod 的日志")}</p>${renderPodSelector()}`, { open: true })}${renderInteractiveTerminal()}${renderExecResult()}${renderFoldSection("日志内容", `<pre class="detail-code">${escapeHtml(state.workloadInspector.logs || "暂无日志输出")}</pre>`, { open: true })}</div>`;
  }
  if (activeView === "manifest") {
    return renderFoldSection("YAML", `<pre class="detail-code">${escapeHtml(state.workloadInspector.manifest || "暂无 YAML 内容")}</pre>`, { open: true });
  }
  return `<div class="detail-stack">${renderDiagnosisOverview(workload)}${renderFoldSection("关联 Pod", renderRelatedPods(), { open: true, meta: `${state.workloadInspector.pods.length} 个` })}</div>`;
}

function renderRolloutInspector(workload) {
  const rollout = state.workloadInspector.rollout;
  if (!rollout) return emptyState("当前工作负载暂无发布状态数据");
  const primaryCards = [
    { label: "策略", value: rollout.strategy || "-" },
    { label: "期望副本", value: rollout.desired_replicas ?? "-" },
    { label: "就绪副本", value: rollout.ready_replicas ?? "-" },
    { label: "已更新副本", value: rollout.updated_replicas ?? "-" },
  ];
  const extraCards = workload.kind === "Deployment"
    ? [
        { label: "可用副本", value: rollout.available_replicas ?? "-" },
        { label: "不可用副本", value: rollout.unavailable_replicas ?? "-" },
        { label: "Max Surge", value: rollout.max_surge || "-" },
        { label: "Max Unavailable", value: rollout.max_unavailable || "-" },
      ]
    : [
        { label: "当前副本", value: rollout.current_replicas ?? "-" },
        { label: "Current Revision", value: rollout.current_revision || "-" },
        { label: "Update Revision", value: rollout.update_revision || "-" },
        { label: "Observed Generation", value: rollout.observed_generation ?? "-" },
      ];
  return `<div class="detail-stack"><section class="detail-lateral-grid"><div class="detail-stack">${renderFoldSection("发布摘要", `<div class="rollout-grid">${[...primaryCards, ...extraCards].map(ctx.renderDetailCard).join("")}</div>`, { open: true })}${renderRolloutConditions()}</div><div class="detail-stack">${workload.kind === "Deployment" ? renderReplicaSets() : renderRevisionSummary(rollout)}</div></section></div>`;
}

function renderDiagnosisOverview(workload) {
  const groups = categorizeWorkloadEvents(state.workloadInspector.events || []);
  const warningEvents = (state.workloadInspector.events || []).filter((item) => String(item.type || "").toLowerCase() === "warning");
  const topGroup = groups[0];
  const podCount = state.workloadInspector.pods.length;
  const selectedPod = state.workloadInspector.selectedPodName || "";
  const cards = [
    {
      label: "Warning 事件",
      value: String(warningEvents.length),
      meta: warningEvents.length ? "建议先检查事件" : "最近无 Warning",
      tone: warningEvents.length ? "warn" : "ok",
      action: "inspector-events",
      cta: "看事件",
    },
    {
      label: "关联 Pod",
      value: String(podCount),
      meta: selectedPod ? `当前日志 Pod: ${selectedPod}` : "可直接切到日志或终端",
      tone: podCount ? "ok" : "warn",
      action: "inspector-logs",
      cta: "看日志",
    },
    {
      label: "首要信号",
      value: topGroup ? topGroup.label : "稳定",
      meta: topGroup ? topGroup.meta : "未发现明显失败信号",
      tone: topGroup ? topGroup.tone : "ok",
      action: topGroup ? "inspector-events" : "inspector-resources",
      cta: topGroup ? "看诊断事件" : "看关联资源",
    },
    {
      label: "变更入口",
      value: supportsWorkloadRollout(workload.kind) ? "发布与版本" : "配置与资源",
      meta: supportsWorkloadRollout(workload.kind) ? "镜像、版本和发布状态都在这里" : "更适合从关联资源和日志继续排查",
      tone: "ok",
      action: supportsWorkloadRollout(workload.kind) ? "inspector-rollout" : "inspector-resources",
      cta: supportsWorkloadRollout(workload.kind) ? "看发布状态" : "看关联资源",
    },
  ];
  return `<section class="diagnosis-board"><div class="diagnosis-board-head"><div><p class="eyebrow">Diagnosis</p><h5>诊断提示</h5></div><p class="table-meta">只保留排障动作入口，不重复展示上面的基础状态。</p></div><div class="diagnosis-grid">${cards.map((item) => `<article class="diagnosis-card diagnosis-card-${escapeHtml(item.tone)}"><span class="muted-label">${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong><p>${escapeHtml(item.meta)}</p><button class="ghost-button" data-action="${escapeHtml(item.action)}">${escapeHtml(item.cta)}</button></article>`).join("")}</div></section>`;
}

function renderEventDiagnosis() {
  const groups = categorizeWorkloadEvents(state.workloadInspector.events || []);
  if (!groups.length) {
    return `<section class="diagnosis-board"><div class="diagnosis-board-head"><div><p class="eyebrow">Event Diagnosis</p><h5>事件诊断</h5></div><p class="table-meta">当前事件里没有明显的故障模式，更多信息可继续查看日志和发布状态。</p></div></section>`;
  }
  return `<section class="diagnosis-board"><div class="diagnosis-board-head"><div><p class="eyebrow">Event Diagnosis</p><h5>事件诊断</h5></div><p class="table-meta">先按故障类型分组，再回到原始事件明细确认具体原因。</p></div><div class="diagnosis-grid">${groups.map((item) => `<article class="diagnosis-card diagnosis-card-${escapeHtml(item.tone)}"><span class="muted-label">${escapeHtml(item.label)}</span><strong>${escapeHtml(String(item.count))}</strong><p>${escapeHtml(item.meta)}</p><button class="ghost-button" data-action="${escapeHtml(item.action)}">${escapeHtml(item.cta)}</button></article>`).join("")}</div></section>`;
}

function categorizeWorkloadEvents(events) {
  const buckets = {
    scheduling: { label: "调度问题", count: 0, tone: "warn", meta: "节点不可调度、亲和性或资源约束未满足", action: "inspector-events", cta: "检查调度事件" },
    probe: { label: "探针失败", count: 0, tone: "warn", meta: "健康检查失败，优先看探针与容器启动日志", action: "inspector-logs", cta: "看日志" },
    image: { label: "镜像问题", count: 0, tone: "warn", meta: "拉镜像失败或镜像不存在，优先检查镜像仓库与凭据", action: "inspector-events", cta: "看镜像事件" },
    quota: { label: "配额限制", count: 0, tone: "warn", meta: "命名空间 ResourceQuota/LimitRange 可能阻塞创建或调度", action: "inspector-resources", cta: "看治理资源" },
    crash: { label: "运行异常", count: 0, tone: "warn", meta: "容器反复重启或启动失败，优先检查日志和环境配置", action: "inspector-logs", cta: "看容器日志" },
    network: { label: "网络暴露", count: 0, tone: "warn", meta: "Service/Ingress 可能存在暴露或后端选择问题", action: "inspector-resources", cta: "看关联资源" },
  };
  events.forEach((item) => {
    const reason = String(item.reason || "").toLowerCase();
    const message = String(item.message || "").toLowerCase();
    const text = `${reason} ${message}`;
    if (text.includes("failedscheduling") || text.includes("0/") || text.includes("node") && text.includes("available")) {
      buckets.scheduling.count += 1;
      return;
    }
    if (text.includes("unhealthy") || text.includes("readiness probe failed") || text.includes("liveness probe failed") || text.includes("startup probe failed")) {
      buckets.probe.count += 1;
      return;
    }
    if (text.includes("imagepullbackoff") || text.includes("errimagepull") || text.includes("pulling image") && text.includes("failed")) {
      buckets.image.count += 1;
      return;
    }
    if (text.includes("exceeded quota") || text.includes("limitrange") || text.includes("resourcequota")) {
      buckets.quota.count += 1;
      return;
    }
    if (text.includes("backoff") || text.includes("crashloopbackoff") || text.includes("failed") || text.includes("error")) {
      buckets.crash.count += 1;
      return;
    }
    if (text.includes("ingress") || text.includes("service") || text.includes("endpoint")) {
      buckets.network.count += 1;
    }
  });
  return Object.values(buckets).filter((item) => item.count > 0).sort((a, b) => b.count - a.count);
}

function renderRolloutConditions() {
  const conditions = state.workloadInspector.rollout?.conditions || [];
  const body = !conditions.length
    ? emptyState("当前工作负载没有可展示的 rollout conditions")
    : `<div class="event-list">${conditions.map((item) => `<article class="event-item"><p class="muted-label">${escapeHtml(item.type || "-")} / ${escapeHtml(item.status || "-")}</p><strong>${escapeHtml(item.reason || "-")}</strong><p>${escapeHtml(item.message || "-")}</p></article>`).join("")}</div>`;
  return renderFoldSection("Conditions", body, { meta: `${conditions.length} 条` });
}

function renderReplicaSets() {
  const replicaSets = state.workloadInspector.rollout?.replica_sets || [];
  const body = !replicaSets.length
    ? emptyState("当前 Deployment 没有可展示的历史版本数据")
    : `<div class="replicaset-list">${replicaSets.map((item, index) => `<article class="pod-item revision-card"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">Revision ${escapeHtml(item.revision || String(index + 1))}</p><p class="table-meta">Ready ${escapeHtml(String(item.ready))} / Desired ${escapeHtml(String(item.desired))} / Available ${escapeHtml(String(item.available))}</p><p class="table-meta">创建于 ${escapeHtml(formatDateTime(item.creation_stamp))}</p></div></article>`).join("")}</div>`;
  return renderFoldSection("历史版本", body, { meta: `${replicaSets.length} 个版本`, open: true });
}

function renderRevisionSummary(rollout) {
  return renderFoldSection("版本信息", `<div class="detail-list"><div class="detail-item"><strong class="muted-label">Current Revision</strong><span>${escapeHtml(rollout.current_revision || "-")}</span></div><div class="detail-item"><strong class="muted-label">Update Revision</strong><span>${escapeHtml(rollout.update_revision || "-")}</span></div></div>`, { meta: "StatefulSet", open: true });
}

function renderResourcesInspector() {
  const resources = state.workloadInspector.resources;
  if (!resources) return emptyState("当前工作负载暂无关联资源数据");
  return `<div class="detail-stack">${renderWorkloadServiceAccount(resources.service_account || "")}${renderServiceResources(resources.services || [])}${renderIngressResources(resources.ingresses || [])}${renderStatefulSetClaimTemplates(resources.statefulset_claim_templates || [], resources.persistent_volume_claims || [])}${renderPersistentVolumeClaimResources(resources.persistent_volume_claims || [])}${renderWorkloadResourceLinks("ConfigMaps", "ConfigMap", resources.config_maps || [])}${renderWorkloadResourceLinks("Secrets", "Secret", resources.secrets || [])}${renderLinkedResourceDetailPanel()}</div>`;
}

function renderServiceResources(services) {
  const body = !services.length
    ? emptyState("当前工作负载没有匹配到 Service")
    : `<div class="replicaset-list">${services.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">${escapeHtml(item.type || "-")} / ClusterIP ${escapeHtml(item.cluster_ip || "-")}</p><p class="table-meta">Ports: ${(item.ports || []).map((port) => escapeHtml(port)).join("，") || "-"}</p><p class="table-meta">Selector: ${(item.selector || []).map((pair) => escapeHtml(pair)).join("，") || "-"}</p><p class="table-meta">Endpoints: ${(item.endpoints || []).map((endpoint) => escapeHtml(endpoint)).join("，") || "-"}</p></div><div class="action-row"><button class="ghost-button" data-action="open-service-detail" data-id="${escapeHtml(item.name)}">查看详情</button><button class="ghost-button danger-soft" data-action="delete-linked-service" data-id="${escapeHtml(item.name)}">删除</button></div></article>`).join("")}</div>`;
  return renderFoldSection("Services", body, { meta: `${services.length} 个`, open: services.length > 0 });
}

function renderIngressResources(ingresses) {
  const body = !ingresses.length
    ? emptyState("当前工作负载没有匹配到 Ingress")
    : `<div class="replicaset-list">${ingresses.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">Host: ${(item.hosts || []).map((host) => escapeHtml(host)).join("，") || "-"}</p><p class="table-meta">Path: ${(item.paths || []).map((path) => escapeHtml(path)).join("，") || "-"}</p><p class="table-meta">Backend: ${(item.backends || []).map((backend) => escapeHtml(backend)).join("，") || "-"}</p><p class="table-meta">Address: ${(item.addresses || []).map((address) => escapeHtml(address)).join("，") || "-"}</p></div><div class="action-row"><button class="ghost-button" data-action="open-ingress-detail" data-id="${escapeHtml(item.name)}">查看详情</button><button class="ghost-button danger-soft" data-action="delete-linked-ingress" data-id="${escapeHtml(item.name)}">删除</button></div></article>`).join("")}</div>`;
  return renderFoldSection("Ingresses", body, { meta: `${ingresses.length} 个` });
}

function renderNameResources(title, items) {
  if (!items.length) return `<section><p class="eyebrow">${escapeHtml(title)}</p>${emptyState(`当前工作负载没有引用 ${title}`)}</section>`;
  return `<section><p class="eyebrow">${escapeHtml(title)}</p><div class="resource-pill-row">${items.map((item) => `<span class="cluster-chip">${escapeHtml(item)}</span>`).join("")}</div></section>`;
}

function renderWorkloadResourceLinks(title, kind, items) {
  const body = !items.length
    ? emptyState(`当前工作负载没有引用 ${title}`)
    : `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item)}</strong></p><p class="table-meta">${escapeHtml(kind)} 资源</p></div><button class="ghost-button" data-action="open-linked-namespace-resource" data-id="${escapeHtml(JSON.stringify({ kind, name: item }))}">进入资源详情</button></article>`).join("")}</div>`;
  return renderFoldSection(title, body, { meta: `${items.length} 个` });
}

function renderWorkloadServiceAccount(name) {
  const body = !name
    ? emptyState("当前工作负载没有声明 ServiceAccount")
    : `<div class="replicaset-list"><article class="pod-item"><div><p><strong>${escapeHtml(name)}</strong></p><p class="table-meta">${name === "default" ? "默认账号" : "自定义账号"}</p></div><button class="ghost-button" data-action="open-linked-namespace-resource" data-id="${escapeHtml(JSON.stringify({ kind: "ServiceAccount", name }))}">进入资源详情</button></article></div>`;
  return renderFoldSection("ServiceAccount", body, { meta: name || "未声明" });
}

function renderStatefulSetClaimTemplates(items, claims) {
  if (!items.length) return "";
  return renderFoldSection("Claim Templates", `<div class="replicaset-list">${items.map((item) => {
    const generatedClaims = (claims || []).filter((claim) => claim.claim_template === item.name);
    return `<article class="pod-item"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">${escapeHtml(item.requested_storage || "-")} / ${escapeHtml(item.storage_class_name || "default")}</p><p class="table-meta">Access Modes: ${(item.access_modes || []).map((mode) => escapeHtml(mode)).join("，") || "-"}</p><p class="table-meta">已生成 PVC: ${escapeHtml(String(generatedClaims.length))}</p>${(item.labels || []).length ? `<p class="table-meta">Labels: ${(item.labels || []).map((label) => escapeHtml(label)).join("，")}</p>` : ""}${(item.annotations || []).length ? `<p class="table-meta">Annotations: ${(item.annotations || []).map((annotation) => escapeHtml(annotation)).join("，")}</p>` : ""}</div></article>`;
  }).join("")}</div>`, { meta: `${items.length} 个模板` });
}

function renderPersistentVolumeClaimResources(items) {
  const body = !items.length
    ? emptyState("当前工作负载没有挂载 PVC")
    : `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">${escapeHtml(item.status || "-")} / ${escapeHtml(item.requested_storage || "-")} / ${escapeHtml(item.storage_class_name || "default")}</p><p class="table-meta">Volume: ${escapeHtml(item.volume_name || "-")} / Access Modes: ${(item.access_modes || []).map((mode) => escapeHtml(mode)).join("，") || "-"}</p>${item.claim_template ? `<p class="table-meta">来源模板: ${escapeHtml(item.claim_template)}</p>` : ""}${(item.mounted_by || []).length ? `<p class="table-meta">Mounted By: ${(item.mounted_by || []).map((workload) => escapeHtml(`${workload.kind}/${workload.name}`)).join("，")}</p>` : ""}</div><button class="ghost-button" data-action="open-linked-namespace-resource" data-id="${escapeHtml(JSON.stringify({ kind: "PersistentVolumeClaim", name: item.name }))}">进入资源详情</button></article>`).join("")}</div>`;
  return renderFoldSection("PersistentVolumeClaims", body, { meta: `${items.length} 个` });
}

function renderLinkedResourceDetailPanel() {
  const detail = state.workloadInspector.resourceDetail;
  if (!detail) return "";
  if (detail.kind === "service") {
    return renderFoldSection("Service Detail", `<div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(detail.data.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">类型</strong><span>${escapeHtml(detail.data.type || "-")}</span></div><div class="detail-item"><strong class="muted-label">ClusterIP</strong><span>${escapeHtml(detail.data.cluster_ip || "-")}</span></div><div class="detail-item"><strong class="muted-label">Session Affinity</strong><span>${escapeHtml(detail.data.session_affinity || "-")}</span></div></div><div class="detail-stack">${renderNameResources("Selector", detail.data.selector || [])}${renderNameResources("External IPs", detail.data.external_ips || [])}${renderServicePortDetail(detail.data.ports || [])}${renderNameResources("Endpoints", detail.data.endpoints || [])}</div>`, { meta: detail.data.name || detail.name, open: true });
  }
  return renderFoldSection("Ingress Detail", `<div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(detail.data.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">Ingress Class</strong><span>${escapeHtml(detail.data.ingress_class || "-")}</span></div><div class="detail-item"><strong class="muted-label">Default Backend</strong><span>${escapeHtml(detail.data.default_backend || "-")}</span></div><div class="detail-item"><strong class="muted-label">地址</strong><span>${(detail.data.addresses || []).map((item) => escapeHtml(item)).join("，") || "-"}</span></div></div>${renderIngressRuleDetail(detail.data.rules || [])}${renderIngressTLSDetail(detail.data.tls || [])}`, { meta: detail.data.name || detail.name, open: true });
}

function renderServicePortDetail(ports) {
  if (!ports.length) return `<section><p class="eyebrow">Ports</p>${emptyState("当前 Service 没有端口配置")}</section>`;
  return `<section><p class="eyebrow">Ports</p><div class="replicaset-list">${ports.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name || "default")}</strong></p><p class="table-meta">${escapeHtml(item.protocol || "TCP")} / ${escapeHtml(String(item.port ?? "-"))} -> ${escapeHtml(String(item.target_port ?? "-"))}</p><p class="table-meta">NodePort: ${escapeHtml(String(item.node_port || "-"))}</p></div></article>`).join("")}</div></section>`;
}

function renderIngressRuleDetail(rules) {
  if (!rules.length) return `<section><p class="eyebrow">Rules</p>${emptyState("当前 Ingress 没有规则")}</section>`;
  return `<section><p class="eyebrow">Rules</p><div class="replicaset-list">${rules.map((rule) => `<article class="pod-item"><div><p><strong>${escapeHtml(rule.host || "-")}</strong></p>${(rule.paths || []).map((path) => `<p class="table-meta">${escapeHtml(path.path || "/")} / ${escapeHtml(path.path_type || "-")} / ${escapeHtml(path.service || "-")} : ${escapeHtml(path.service_port || "-")}</p>`).join("")}</div></article>`).join("")}</div></section>`;
}

function renderIngressTLSDetail(items) {
  if (!items.length) return `<section><p class="eyebrow">TLS</p>${emptyState("当前 Ingress 没有 TLS 配置")}</section>`;
  return `<section><p class="eyebrow">TLS</p><div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.secret_name || "-")}</strong></p><p class="table-meta">${(item.hosts || []).map((host) => escapeHtml(host)).join("，") || "-"}</p></div></article>`).join("")}</div></section>`;
}

function renderRelatedPods() {
  if (!state.workloadInspector.pods.length) return emptyState("当前工作负载没有关联 Pod，或尚未同步到可见 Pod");
  return `<div class="pod-list">${state.workloadInspector.pods.map((item) => `<article class="pod-item ${item.name === state.workloadInspector.selectedPodName ? "active" : ""}"><div><p><strong>${escapeHtml(item.name)}</strong></p><p class="table-meta">${escapeHtml(item.status)} / ${escapeHtml(item.image || "-")}</p></div><div class="action-row"><button class="ghost-button" data-action="select-log-pod" data-id="${escapeHtml(item.name)}">查看日志</button><button class="ghost-button" data-action="open-interactive-terminal-pod" data-id="${escapeHtml(item.name)}">进入终端</button><button class="ghost-button" data-action="exec-workload-command-pod" data-id="${escapeHtml(item.name)}">执行命令</button></div></article>`).join("")}</div>`;
}

function renderPodSelector() {
  if (!state.workloadInspector.pods.length) return "";
  return `<div class="pod-selector-row">${state.workloadInspector.pods.map((item) => `<button class="ghost-button ${item.name === state.workloadInspector.selectedPodName ? "active" : ""}" data-action="select-log-pod" data-id="${escapeHtml(item.name)}">${escapeHtml(item.name)}</button>`).join("")}</div>`;
}

function renderLogToolbar() {
  const followEnabled = !!state.workloadInspector.followLogs;
  const intervalText = `${state.workloadInspector.followIntervalSeconds || 5}s`;
  return `<section class="log-toolbar"><div><p class="eyebrow">Logs</p><h5>日志查看</h5></div><div class="log-toolbar-actions"><span class="table-meta">${escapeHtml(followEnabled ? `自动跟随已开启，每 ${intervalText} 刷新` : "手动刷新模式")}</span><button class="ghost-button ${followEnabled ? "active" : ""}" data-action="toggle-log-follow">${followEnabled ? "停止自动跟随" : "自动跟随"}</button><button class="ghost-button" data-action="refresh-workload-logs">立即刷新</button></div></section>`;
}

function renderExecResult() {
  const result = state.workloadInspector.execResult;
  if (!result) {
    return renderFoldSection("命令执行", `<p class="table-meta">在上方“工作负载运维”里点击“命令执行”，可以针对当前 Pod 执行一次非交互式排障命令。适合快速查看环境变量、DNS 配置和挂载文件。</p>`, { meta: "未执行", open: false });
  }
  const command = Array.isArray(result.command) ? result.command.join(" ") : String(result.command || "");
  return renderFoldSection("命令执行", `<div class="detail-list"><div class="detail-item"><strong class="muted-label">执行 Pod</strong><span>${escapeHtml(result.source_pod || "-")}</span></div><div class="detail-item"><strong class="muted-label">容器</strong><span>${escapeHtml(result.container_name || "默认容器")}</span></div><div class="detail-item"><strong class="muted-label">命令</strong><span>${escapeHtml(command || "-")}</span></div><div class="detail-item"><strong class="muted-label">执行结果</strong><span>${escapeHtml(result.success ? "成功" : "失败")}</span></div></div><pre class="detail-code">${escapeHtml(result.output || result.error_message || "无输出")}</pre>`, { meta: result.success ? "success" : "error", open: true });
}

function renderInteractiveTerminal() {
  const terminal = state.workloadInspector.interactiveTerminal;
  if (!terminal) {
    return renderFoldSection("交互式终端", `<p class="table-meta">在上方“工作负载运维”里点击“进入终端”，可以像 Kuboard 一样直接进入容器内执行命令。这里使用 WebSocket 持续传输输入和输出。</p>`, { meta: "未连接", open: false });
  }
  const statusText = state.workloadInspector.interactiveTerminalConnected ? "已连接" : "已断开";
  const errorText = state.workloadInspector.interactiveTerminalError
    ? `<p class="table-meta terminal-status-error">${escapeHtml(state.workloadInspector.interactiveTerminalError)}</p>`
    : "";
  return renderFoldSection(
    "交互式终端",
    `<div class="detail-list"><div class="detail-item"><strong class="muted-label">执行 Pod</strong><span>${escapeHtml(terminal.source_pod || "-")}</span></div><div class="detail-item"><strong class="muted-label">容器</strong><span>${escapeHtml(terminal.container_name || "默认容器")}</span></div><div class="detail-item"><strong class="muted-label">Shell</strong><span>${escapeHtml(terminal.shell || "/bin/sh")}</span></div><div class="detail-item"><strong class="muted-label">状态</strong><span>${escapeHtml(statusText)}</span></div><div class="detail-item"><strong class="muted-label">最近事件</strong><span>${escapeHtml(state.workloadInspector.interactiveTerminalLastEvent || "-")}</span></div><div class="detail-item"><strong class="muted-label">输出字节</strong><span>${escapeHtml(String(state.workloadInspector.interactiveTerminalOutputBytes || 0))}</span></div></div>${errorText}<div class="interactive-terminal-toolbar"><span class="table-meta">点击终端区域即可输入，尺寸变化会自动同步。支持粘贴、方向键和 Ctrl+C / Ctrl+D / Ctrl+L。</span><div class="action-row"><button class="ghost-button" data-action="focus-interactive-terminal">聚焦终端</button><button class="ghost-button" data-action="clear-interactive-terminal">清空输出</button><button class="ghost-button danger-soft" data-action="close-interactive-terminal">关闭终端</button></div></div><div id="interactive-terminal-screen" class="interactive-terminal-screen" tabindex="0" aria-label="interactive terminal"><pre class="interactive-terminal-output">${escapeHtml(state.workloadInspector.interactiveTerminalOutput || "终端已建立，等待输入…")}</pre></div>`,
    { meta: statusText, open: true }
  );
}

function renderFoldSection(title, body, options = {}) {
  const meta = options.meta ? `<span class="table-meta">${escapeHtml(String(options.meta))}</span>` : "";
  return `<details class="fold-section"${options.open ? " open" : ""}><summary><span>${escapeHtml(title)}</span>${meta}</summary><div class="fold-section-body">${body}</div></details>`;
}

function bindInteractiveTerminalBindings() {
  bindAction(elements.workloadDetailPanel, "focus-interactive-terminal", () => {
    elements.workloadDetailPanel.querySelector("#interactive-terminal-screen")?.focus();
  });
  bindAction(elements.workloadDetailPanel, "clear-interactive-terminal", () => {
    state.workloadInspector.interactiveTerminalOutput = "";
    const output = elements.workloadDetailPanel.querySelector(".interactive-terminal-output");
    if (output) output.textContent = "终端输出已清空，继续输入命令…";
  });
  bindAction(elements.workloadDetailPanel, "close-interactive-terminal", () => ctx.runK8sAction(() => ctx.closeInteractiveTerminal()));
  const screen = elements.workloadDetailPanel.querySelector("#interactive-terminal-screen");
  if (!screen) return;
  screen.addEventListener("click", () => screen.focus());
  screen.addEventListener("keydown", (event) => {
    const payload = mapTerminalKeyEvent(event);
    if (!payload) return;
    event.preventDefault();
    ctx.runK8sAction(() => ctx.sendInteractiveTerminalInput(payload));
  });
  screen.addEventListener("paste", (event) => {
    const text = event.clipboardData?.getData("text");
    if (!text) return;
    event.preventDefault();
    ctx.runK8sAction(() => ctx.sendInteractiveTerminalInput(text));
  });
}

function mapTerminalKeyEvent(event) {
  if (event.ctrlKey && !event.shiftKey && !event.altKey) {
    const key = String(event.key || "").toLowerCase();
    if (key === "c") return "\u0003";
    if (key === "d") return "\u0004";
    if (key === "l") return "\u000c";
  }
  if (event.key === "Enter") return "\r";
  if (event.key === "Backspace") return "\u007f";
  if (event.key === "Tab") return "\t";
  if (event.key === "Escape") return "\u001b";
  if (event.key === "ArrowUp") return "\u001b[A";
  if (event.key === "ArrowDown") return "\u001b[B";
  if (event.key === "ArrowRight") return "\u001b[C";
  if (event.key === "ArrowLeft") return "\u001b[D";
  if (event.key.length === 1 && !event.metaKey && !event.altKey) {
    return event.key;
  }
  return "";
}
