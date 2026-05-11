import { elements } from "../../core/dom.js";
import { renderScopeBadges, renderScopeSummary } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { bindAction } from "../../core/ui.js";
import { emptyState, escapeHtml, filterItems, findById, formatDateTime } from "../../shared/utils.js";

let ctx = null;

export function configureK8sPageRenderers(deps) {
  ctx = deps;
}

export function renderClusters() {
  const items = filterItems(state.clusters, state.filters.clusters, ["name", "code", "environment", "provider", "api_endpoint"]);
  if (!state.clusters.length) return (elements.clustersTable.innerHTML = emptyState("暂无集群"));
  if (!items.length) return (elements.clustersTable.innerHTML = emptyState("没有匹配的集群"));
  elements.clustersTable.innerHTML = `<div class="cluster-grid">${items.map((item, index) => `<article class="cluster-card cluster-card-compact"><div class="cluster-card-head"><div><p class="eyebrow">Cluster ${index + 1}</p><h5>${escapeHtml(item.name)}</h5><p class="table-meta">${escapeHtml(displayClusterEndpoint(item.api_endpoint))}</p><p class="table-meta">${escapeHtml(renderScopeSummary({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id, fallbackEnvironment: item.environment }))}</p><div class="cluster-chip-row"><span class="cluster-chip">${escapeHtml(item.environment)}</span><span class="cluster-chip">${escapeHtml(item.provider)}</span><span class="cluster-chip">${escapeHtml(item.auth_type)}</span></div><div class="scope-badge-row">${renderScopeBadges({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id, fallbackEnvironment: item.environment })}</div></div><div class="cell-stack"><span class="status-pill ${!isHealthyClusterStatus(item.status) ? "disabled" : ""}">${escapeHtml(item.status)}</span><span class="table-meta">检测 ${formatDateTime(item.last_checked_at)}</span></div></div><div class="cluster-meta-grid cluster-meta-grid-compact"><div class="cluster-meta-card"><span class="muted-label">Namespace</span><strong>${item.namespace_count}</strong></div><div class="cluster-meta-card"><span class="muted-label">版本</span><strong>${escapeHtml(item.server_version || item.version || "-")}</strong></div><div class="cluster-meta-card"><span class="muted-label">来源</span><strong>${escapeHtml(item.source_resource_id || "-")}</strong></div></div><div class="cluster-actions cluster-actions-compact"><button class="primary-button" data-action="open-cluster-overview" data-id="${item.id}">概览</button><button class="ghost-button" data-action="open-cluster-namespaces" data-id="${item.id}">命名空间</button><button class="ghost-button" data-action="test-cluster" data-id="${item.id}">检测</button><button class="ghost-button" data-action="edit-cluster" data-id="${item.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-cluster" data-id="${item.id}">删除</button></div></article>`).join("")}</div>`;
  bindAction(elements.clustersTable, "open-cluster-overview", (id) => ctx.runK8sAction(() => ctx.openClusterOverview(findById(state.clusters, id))));
  bindAction(elements.clustersTable, "open-cluster-namespaces", (id) => ctx.runK8sAction(() => ctx.openClusterNamespaces(findById(state.clusters, id))));
  bindAction(elements.clustersTable, "test-cluster", (id) => ctx.runK8sAction(() => ctx.testCluster(findById(state.clusters, id))));
  bindAction(elements.clustersTable, "sync-cluster", (id) => ctx.runK8sAction(() => ctx.syncClusterNamespaces(findById(state.clusters, id))));
  bindAction(elements.clustersTable, "edit-cluster", (id) => ctx.openClusterEditModal(findById(state.clusters, id)));
  bindAction(elements.clustersTable, "delete-cluster", (id) => ctx.runK8sAction(() => ctx.deleteCluster(findById(state.clusters, id))));
}

function isHealthyClusterStatus(status) {
  return ["ready", "enrolled", "running", "active"].includes(String(status || "").trim().toLowerCase());
}

function displayClusterEndpoint(rawEndpoint) {
  const text = String(rawEndpoint || "").trim();
  if (!text) return "-";
  if (!text.startsWith("{")) return text;
  try {
    const payload = JSON.parse(text);
    return String(payload.api_server_endpoint || payload.intranet_api_server_endpoint || payload.endpoint || text).trim() || "-";
  } catch {
    return text;
  }
}

export function renderNamespaces() {
  const clusterID = selectedClusterID();
  if (!clusterID) {
    elements.namespacesTable.innerHTML = emptyState("请先选择一个集群，再查看命名空间");
    renderNamespacePageSummary();
    return;
  }
  if (!state.namespaces.length) {
    elements.namespacesTable.innerHTML = emptyState("暂无命名空间");
    renderNamespacePageSummary();
    return;
  }
  const items = filteredNamespaces(state.namespaces);
  if (!items.length) {
    const message = state.business.hideSystemNamespaces
      ? "当前筛选结果为空。你可以关闭“只看业务命名空间”查看 kube-* 系统命名空间。"
      : "没有匹配的命名空间";
    elements.namespacesTable.innerHTML = emptyState(message);
    renderNamespacePageSummary(items);
    return;
  }
  const governanceMap = new Map((state.clusterInspector.overview?.namespace_governance || []).map((item) => [item.name, item]));
  elements.namespacesTable.innerHTML = `<table><thead><tr><th>序号</th><th>Namespace</th><th>所属集群</th><th>来源</th><th>治理</th><th>状态</th><th>操作</th></tr></thead><tbody>${items.map((item, index) => {
    const governance = governanceMap.get(item.name) || { resource_quota_count: 0, limit_range_count: 0 };
    const namespaceBadges = [isSystemNamespace(item.name) ? `<span class="cluster-chip muted">system</span>` : ""].join("");
    return `<tr><td>${index + 1}</td><td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">${escapeHtml(item.display_name || "-")}</span><div class="cluster-chip-row">${namespaceBadges}</div></div></td><td>${escapeHtml(item.cluster_name)}</td><td><span class="${item.source_type === "sync" ? "group-tag" : "role-tag"}">${escapeHtml(item.source_type)}</span></td><td><div class="cell-stack"><span class="table-meta">Quota ${escapeHtml(String(governance.resource_quota_count ?? 0))} / Limit ${escapeHtml(String(governance.limit_range_count ?? 0))}</span><div class="action-row"><button class="ghost-button" data-action="open-namespace-quota" data-id="${item.id}">查看 Quota</button><button class="ghost-button" data-action="open-namespace-limitrange" data-id="${item.id}">查看 LimitRange</button></div></div></td><td><div class="cell-stack"><span class="status-pill ${item.status !== "active" ? "disabled" : ""}">${escapeHtml(item.status)}</span><span class="table-meta">更新于 ${formatDateTime(item.updated_at)}</span></div></td><td><div class="action-row"><button class="primary-button" data-action="open-namespace-workloads" data-id="${item.id}">查看工作负载</button><button class="ghost-button" data-action="open-namespace-resources" data-id="${item.id}">查看资源</button><button class="ghost-button" data-action="edit-namespace" data-id="${item.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-namespace" data-id="${item.id}">删除</button></div></td></tr>`;
  }).join("")}</tbody></table>`;
  bindAction(elements.namespacesTable, "open-namespace-workloads", (id) => ctx.runK8sAction(() => ctx.openNamespaceWorkloads(findById(state.namespaces, id))));
  bindAction(elements.namespacesTable, "open-namespace-resources", (id) => ctx.runK8sAction(() => ctx.openNamespaceResources(findById(state.namespaces, id))));
  bindAction(elements.namespacesTable, "open-namespace-quota", (id) => ctx.runK8sAction(() => ctx.openNamespaceGovernance(findById(state.namespaces, id), "ResourceQuota")));
  bindAction(elements.namespacesTable, "open-namespace-limitrange", (id) => ctx.runK8sAction(() => ctx.openNamespaceGovernance(findById(state.namespaces, id), "LimitRange")));
  bindAction(elements.namespacesTable, "edit-namespace", (id) => ctx.openNamespaceEditModal(findById(state.namespaces, id)));
  bindAction(elements.namespacesTable, "delete-namespace", (id) => ctx.runK8sAction(() => ctx.deleteNamespace(findById(state.namespaces, id))));
  renderNamespacePageSummary(items);
}

export function renderClusterSelectors() {
  const selectedCluster = state.business.selectedClusterID;
  const defaultNamespaceFilter = selectedCluster ? String(selectedCluster) : "";
  const currentFilter = elements.namespaceClusterFilter.value || defaultNamespaceFilter;
  const options = [`<option value="">选择集群后继续</option>`, ...state.clusters.map((item) => `<option value="${item.id}">${escapeHtml(item.name)}</option>`)];
  elements.namespaceClusterFilter.innerHTML = options.join("");
  if (currentFilter && state.clusters.some((item) => String(item.id) === currentFilter)) {
    elements.namespaceClusterFilter.value = currentFilter;
  }
  elements.createNamespaceClusterID.innerHTML = state.clusters.map((item) => `<option value="${item.id}">${escapeHtml(item.name)}</option>`).join("");
  const currentWorkloadCluster = elements.workloadClusterFilter.value || defaultNamespaceFilter;
  elements.workloadClusterFilter.innerHTML = options.join("");
  if (currentWorkloadCluster && state.clusters.some((item) => String(item.id) === currentWorkloadCluster)) {
    elements.workloadClusterFilter.value = currentWorkloadCluster;
  }
  const currentResourceCluster = elements.resourceClusterFilter.value || defaultNamespaceFilter;
  elements.resourceClusterFilter.innerHTML = options.join("");
  if (currentResourceCluster && state.clusters.some((item) => String(item.id) === currentResourceCluster)) {
    elements.resourceClusterFilter.value = currentResourceCluster;
  }
  renderWorkloadNamespaceOptions();
  renderResourceNamespaceOptions();
  if (state.workloadCreator.builder || state.workloadCreator.yaml) {
    ctx.ensureWorkloadCreatorDefaults();
    ctx.renderWorkloadCreator();
  }
}

export function selectedClusterID() {
  const value = Number(elements.namespaceClusterFilter.value || state.business.selectedClusterID || 0);
  return value > 0 ? value : undefined;
}

export function selectedWorkloadClusterID() {
  const value = Number(elements.workloadClusterFilter.value || state.business.selectedClusterID || 0);
  return value > 0 ? value : undefined;
}

export function selectedWorkloadNamespace() {
  return String(elements.workloadNamespaceFilter.value || "").trim();
}

export function selectedResourceClusterID() {
  const value = Number(elements.resourceClusterFilter.value || state.business.selectedClusterID || 0);
  return value > 0 ? value : undefined;
}

export function selectedResourceNamespace() {
  return String(elements.resourceNamespaceFilter.value || "").trim();
}

export function selectedResourceKind() {
  return String(elements.resourceKindFilter.value || state.business.selectedResourceKind || "Service").trim();
}

export function renderWorkloadNamespaceOptions() {
  const clusterID = selectedWorkloadClusterID();
  const namespaces = filterNamespacesForSelection(clusterID ? state.namespaces.filter((item) => item.cluster_id === clusterID) : []);
  const current = elements.workloadNamespaceFilter.value || state.business.selectedNamespace || "";
  elements.workloadNamespaceFilter.disabled = !clusterID;
  elements.workloadNamespaceFilter.innerHTML = [`<option value="">${clusterID ? "全部命名空间" : "先选择集群"}</option>`, ...namespaces.map((item) => `<option value="${escapeHtml(item.name)}">${escapeHtml(item.name)}</option>`)].join("");
  if (current && namespaces.some((item) => item.name === current)) {
    elements.workloadNamespaceFilter.value = current;
  }
}

export function renderResourceNamespaceOptions() {
  const clusterID = selectedResourceClusterID();
  const namespaces = filterNamespacesForSelection(clusterID ? state.namespaces.filter((item) => item.cluster_id === clusterID) : []);
  const current = elements.resourceNamespaceFilter.value || state.business.selectedNamespace || "";
  elements.resourceNamespaceFilter.disabled = !clusterID;
  elements.resourceNamespaceFilter.innerHTML = [`<option value="">${clusterID ? "全部命名空间" : "先选择集群"}</option>`, ...namespaces.map((item) => `<option value="${escapeHtml(item.name)}">${escapeHtml(item.name)}</option>`)].join("");
  if (current && namespaces.some((item) => item.name === current)) {
    elements.resourceNamespaceFilter.value = current;
  }
  elements.resourceKindFilter.value = selectedResourceKind();
}

export function filteredWorkloads() {
  let items = filterItems(state.workloads, state.filters.workloads, ["name", "namespace_name", "image", "status", "kind"]);
  if (state.business.hideSystemNamespaces) {
    items = items.filter((item) => !isSystemNamespace(item.namespace_name));
  }
  if (state.business.selectedWorkloadKind) {
    items = items.filter((item) => item.kind === state.business.selectedWorkloadKind);
  }
  return items;
}

export function workloadReplicaLabel(item) {
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

export function workloadTypeSummary(workloads) {
  const order = ["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"];
  return order
    .map((kind) => ({ kind, count: workloads.filter((item) => item.kind === kind).length }))
    .filter((item) => item.count > 0);
}

export function normalizeInspectorView(kind, view) {
  if (view === "rollout" && !["Deployment", "StatefulSet"].includes(kind)) return "overview";
  if (view === "logs" && !["Deployment", "StatefulSet", "DaemonSet", "Job", "Pod"].includes(kind)) return "overview";
  return view;
}

export function filteredResources() {
  let items = filterItems(state.resourceExplorer.items, state.filters.resources, ["name", "kind", "summary", "status", "namespace"]);
  const scope = state.business.resourceScope || (state.business.hideSystemNamespaces ? "business" : "all");
  if (scope === "business") {
    items = items.filter((item) => !isSystemNamespace(item.namespace));
  } else if (scope === "system") {
    items = items.filter((item) => isSystemNamespace(item.namespace));
  } else if (state.business.hideSystemNamespaces) {
    items = items.filter((item) => !isSystemNamespace(item.namespace));
  }
  return items;
}

export function selectedBusinessCluster() {
  if (!state.business.selectedClusterID) return null;
  return state.clusters.find((item) => item.id === state.business.selectedClusterID) || null;
}

export function renderWorkloads() {
  const clusterID = selectedWorkloadClusterID();
  if (!clusterID) {
    elements.workloadsTable.innerHTML = emptyState("请先选择一个集群，再查看工作负载");
    elements.workloadPagination.innerHTML = "";
    renderWorkloadPageSummary();
    ctx.renderWorkloadDetailPanel();
    return;
  }
  if (!state.workloads.length) {
    elements.workloadsTable.innerHTML = emptyState("暂无工作负载");
    elements.workloadPagination.innerHTML = "";
    renderWorkloadPageSummary();
    ctx.renderWorkloadDetailPanel();
    return;
  }
  const items = filteredWorkloads();
  if (!items.length) {
    elements.workloadsTable.innerHTML = emptyState("没有匹配的工作负载");
    elements.workloadPagination.innerHTML = "";
    renderWorkloadPageSummary(items);
    ctx.renderWorkloadDetailPanel();
    return;
  }
  const pageSize = state.pagination.workloads.pageSize;
  const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
  if (state.pagination.workloads.page > totalPages) state.pagination.workloads.page = totalPages;
  const page = state.pagination.workloads.page;
  const start = (page - 1) * pageSize;
  const currentItems = items.slice(start, start + pageSize);
  elements.workloadsTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>名称</th><th>命名空间</th><th>副本</th><th>镜像</th><th>状态</th><th>操作</th></tr></thead><tbody>${currentItems.map((item, index) => `<tr class="workload-row ${state.business.selectedWorkloadID === item.id ? "active" : ""}" data-action="select-workload" data-id="${item.id}"><td>${start + index + 1}</td><td><span class="cluster-chip">${escapeHtml(item.kind)}</span></td><td><button class="link-button" data-action="select-workload" data-id="${item.id}">${escapeHtml(item.name)}</button></td><td>${escapeHtml(item.namespace_name)}</td><td>${escapeHtml(workloadReplicaLabel(item))}</td><td class="code-cell">${escapeHtml(item.image || "-")}</td><td><span class="status-pill ${!["ready", "Running", "completed", "scheduled"].includes(item.status) ? "disabled" : ""}">${escapeHtml(item.status)}</span></td><td><div class="action-row"><button class="primary-button" data-action="select-workload" data-id="${item.id}">进入服务</button><button class="ghost-button danger-soft" data-action="delete-workload" data-id="${item.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  elements.workloadPagination.innerHTML = renderPaginationControls("workload", page, totalPages, items.length, pageSize);
  bindAction(elements.workloadsTable, "select-workload", (id) => ctx.runK8sAction(() => ctx.selectWorkload(Number(id))));
  bindAction(elements.workloadsTable, "delete-workload", (id) => ctx.runK8sAction(() => ctx.deleteWorkload(findById(state.workloads, id))));
  bindPaginationControls(elements.workloadPagination, "workload", state.pagination.workloads, totalPages, renderWorkloads);
  renderWorkloadPageSummary(items);
  ctx.renderWorkloadDetailPanel();
}

export function renderResources() {
  const clusterID = selectedResourceClusterID();
  if (!clusterID) {
    elements.resourcesTable.innerHTML = emptyState("请先选择一个集群，再查看资源");
    elements.resourcePagination.innerHTML = "";
    renderResourcePageSummary();
    ctx.renderResourceExplorerPanel();
    return;
  }
  if (!state.resourceExplorer.items.length) {
    elements.resourcesTable.innerHTML = emptyState("暂无资源");
    elements.resourcePagination.innerHTML = "";
    renderResourcePageSummary();
    ctx.renderResourceExplorerPanel();
    return;
  }
  const items = filteredResources();
  if (!items.length) {
    elements.resourcesTable.innerHTML = emptyState("没有匹配的资源");
    elements.resourcePagination.innerHTML = "";
    renderResourcePageSummary(items);
    ctx.renderResourceExplorerPanel();
    return;
  }
  const pageSize = state.pagination.resources.pageSize;
  const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
  if (state.pagination.resources.page > totalPages) state.pagination.resources.page = totalPages;
  const page = state.pagination.resources.page;
  const start = (page - 1) * pageSize;
  const currentItems = items.slice(start, start + pageSize);
  elements.resourcesTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>名称</th><th>命名空间</th><th>摘要</th><th>状态</th><th>操作</th></tr></thead><tbody>${currentItems.map((item, index) => `<tr><td>${start + index + 1}</td><td><span class="cluster-chip">${escapeHtml(item.kind)}</span></td><td><strong>${escapeHtml(item.name)}</strong></td><td>${escapeHtml(item.namespace)}</td><td>${escapeHtml(item.summary || "-")}</td><td><span class="status-pill">${escapeHtml(item.status || "-")}</span></td><td><div class="action-row"><button class="primary-button" data-action="select-resource" data-id="${escapeHtml(item.name)}">进入详情</button><button class="ghost-button danger-soft" data-action="delete-resource" data-id="${escapeHtml(item.name)}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  elements.resourcePagination.innerHTML = renderPaginationControls("resource", page, totalPages, items.length, pageSize);
  bindAction(elements.resourcesTable, "select-resource", (name) => ctx.runK8sAction(() => ctx.selectResource(String(name))));
  bindAction(elements.resourcesTable, "delete-resource", (name) => ctx.runK8sAction(() => ctx.deleteNamespaceResource(selectedResourceKind(), String(name))));
  bindPaginationControls(elements.resourcePagination, "resource", state.pagination.resources, totalPages, renderResources);
  renderResourcePageSummary(items);
  ctx.renderResourceExplorerPanel();
}

export function renderNamespacePageSummary() {
  const cluster = selectedBusinessCluster();
  if (!cluster) {
    elements.namespacePageTitle.textContent = "请选择一个集群";
    elements.namespacePageCopy.textContent = "从集群列表进入后，这里会展示当前集群的接入状态、同步状态和命名空间概览。";
    elements.namespacePageSummary.innerHTML = emptyState("当前还没有选中的集群");
    return;
  }
  const namespaces = state.namespaces.filter((item) => item.cluster_id === cluster.id);
  const visibleNamespaces = filteredNamespaces(namespaces);
  const businessNamespaces = namespaces.filter((item) => !isSystemNamespace(item.name));
  const syncedNamespaces = namespaces.filter((item) => item.source_type === "sync").length;
  const manualNamespaces = namespaces.length - syncedNamespaces;
  const systemNamespaces = namespaces.filter((item) => isSystemNamespace(item.name)).length;
  const governance = state.clusterInspector.overview?.namespace_governance || [];
  const quotaCount = governance.reduce((sum, item) => sum + Number(item.resource_quota_count || 0), 0);
  const limitCount = governance.reduce((sum, item) => sum + Number(item.limit_range_count || 0), 0);
  const clusterWorkloads = state.workloads.filter((item) => item.cluster_id === cluster.id);
  const warningWorkloads = clusterWorkloads.filter((item) => !["ready", "Running", "completed", "scheduled"].includes(item.status)).length;
  elements.namespacePageTitle.textContent = cluster.name;
  elements.namespacePageCopy.textContent = cluster.description || `当前集群来自 ${cluster.provider} / ${cluster.environment}，可在下方继续查看和维护命名空间。`;
  elements.namespacePageSummary.innerHTML = [
    renderInsightCard("集群状态", cluster.status, `最近同步 ${formatDateTime(cluster.last_synced_at)}`, cluster.status === "ready" ? "ok" : "warn"),
    renderInsightCard("命名空间视图", `${visibleNamespaces.length} / ${namespaces.length}`, state.business.hideSystemNamespaces ? "当前已折叠系统命名空间" : "当前显示全部命名空间"),
    renderInsightCard("业务 / 系统", `${businessNamespaces.length} / ${systemNamespaces.length}`, "便于区分业务空间和 kube-* 系统空间"),
    renderInsightCard("来源结构", `${syncedNamespaces} 同步 / ${manualNamespaces} 手工`, "看当前集群里哪些空间是平台维护、哪些来自实时同步"),
    renderInsightCard("治理对象", `${quotaCount} Quota / ${limitCount} LimitRange`, "帮助快速判断哪些命名空间已经配置资源治理"),
    renderInsightCard("需关注工作负载", warningWorkloads, warningWorkloads ? "该集群下存在未就绪对象，建议继续进入工作负载页" : "当前已同步对象里没有明显异常状态", warningWorkloads ? "warn" : "ok"),
  ].join("");
}

export function renderWorkloadPageSummary(filteredItems) {
  const cluster = selectedBusinessCluster();
  const namespaceName = state.business.selectedNamespace;
  let workloads = filteredItems || (namespaceName ? state.workloads.filter((item) => item.namespace_name === namespaceName) : state.workloads);
  if (state.business.hideSystemNamespaces) {
    workloads = workloads.filter((item) => !isSystemNamespace(item.namespace_name));
  }
  if (!cluster) {
    elements.workloadPageTitle.textContent = "请选择一个集群";
    elements.workloadPageCopy.textContent = "先锁定集群，再按命名空间继续查看工作负载。";
    elements.workloadPageSummary.innerHTML = emptyState("当前还没有选中的集群");
    return;
  }
  const readyCount = workloads.filter((item) => ["ready", "Running", "completed", "scheduled"].includes(item.status)).length;
  const warningCount = workloads.length - readyCount;
  const deploymentCount = workloads.filter((item) => item.kind === "Deployment").length;
  const statefulCount = workloads.filter((item) => item.kind === "StatefulSet").length;
  const daemonCount = workloads.filter((item) => item.kind === "DaemonSet").length;
  const jobLikeCount = workloads.filter((item) => ["Job", "CronJob"].includes(item.kind)).length;
  const typeCards = workloadTypeSummary(workloads).map((item) => ({ label: item.kind, value: item.count }));
  elements.workloadPageTitle.textContent = namespaceName ? `${cluster.name} / ${namespaceName}` : `${cluster.name} / 全部命名空间`;
  elements.workloadPageCopy.textContent = namespaceName ? `当前命名空间下共有 ${workloads.length} 个工作负载对象，可继续查看详情。` : "当前展示该集群下已同步的全部工作负载。";
  elements.workloadPageSummary.innerHTML = [
    renderInsightCard("工作负载总数", workloads.length, namespaceName ? "当前命名空间范围" : "当前集群范围"),
    renderInsightCard("健康 / 异常", `${readyCount} / ${warningCount}`, warningCount ? "建议优先进入异常对象详情看事件和日志" : "当前列表没有明显异常对象", warningCount ? "warn" : "ok"),
    renderInsightCard("控制器分布", `${deploymentCount} Deploy / ${statefulCount} STS / ${daemonCount} DS`, `Job 与 CronJob 合计 ${jobLikeCount} 个`),
    renderInsightCard("命名空间过滤", state.business.hideSystemNamespaces ? "业务命名空间" : "全部命名空间", state.business.hideSystemNamespaces ? "系统命名空间已折叠" : "包含 kube-* 系统命名空间"),
    ...typeCards.map((item) => renderInsightCard(item.label, item.value, "对象类型分布")),
  ].join("");
}

export function renderResourcePageSummary(filteredItems) {
  const cluster = selectedBusinessCluster();
  const namespaceName = selectedResourceNamespace();
  const items = filteredItems || state.resourceExplorer.items;
  if (!cluster || !namespaceName) {
    elements.resourcePageTitle.textContent = "请选择一个集群";
    elements.resourcePageCopy.textContent = "先选择集群，再继续缩小到命名空间和资源类型。";
    elements.resourcePageSummary.innerHTML = emptyState("当前还没有选中的集群或命名空间");
    return;
  }
  const kind = selectedResourceKind();
  elements.resourcePageTitle.textContent = `${cluster.name} / ${namespaceName} / ${kind}`;
  elements.resourcePageCopy.textContent = `当前展示 ${namespaceName} 下的 ${kind} 资源，共 ${items.length} 条。`;
  elements.resourcePageSummary.innerHTML = [
    { label: "资源总数", value: items.length },
    { label: "资源类型", value: kind },
    { label: "所属集群", value: cluster.name },
    { label: "命名空间", value: namespaceName },
    { label: "资源范围", value: resourceScopeLabel() },
  ].map(ctx.renderDetailCard).join("");
}

function filteredNamespaces(items) {
  let filtered = filterItems(items, state.filters.namespaces, ["name", "display_name", "cluster_name", "source_type", "status"]);
  if (state.business.hideSystemNamespaces) {
    filtered = filtered.filter((item) => !isSystemNamespace(item.name));
  }
  return filtered;
}

function filterNamespacesForSelection(items) {
  if (!state.business.hideSystemNamespaces) return items;
  return items.filter((item) => !isSystemNamespace(item.name));
}

function isSystemNamespace(name) {
  return String(name || "").startsWith("kube-");
}

function resourceScopeLabel() {
  switch (state.business.resourceScope) {
    case "business":
      return "业务资源";
    case "system":
      return "系统资源";
    default:
      return state.business.hideSystemNamespaces ? "全部资源 / 已隐藏系统" : "全部资源";
  }
}

function renderInsightCard(label, value, helper = "", tone = "") {
  return `<article class="detail-card insight-card ${tone ? `insight-card-${tone}` : ""}"><span class="muted-label">${escapeHtml(String(label))}</span><strong>${escapeHtml(String(value))}</strong>${helper ? `<p class="table-meta">${escapeHtml(String(helper))}</p>` : ""}</article>`;
}

function renderPaginationControls(prefix, page, totalPages, totalItems, pageSize) {
  const sizes = [10, 20, 30, 50, 100];
  return `<div class="pagination-group"><div class="pagination-controls"><button class="ghost-button" data-action="${prefix}-prev" ${page <= 1 ? "disabled" : ""}>上一页</button><span class="table-meta">第 ${page} / ${totalPages} 页，共 ${totalItems} 条</span><button class="ghost-button" data-action="${prefix}-next" ${page >= totalPages ? "disabled" : ""}>下一页</button></div><div class="pagination-controls"><label class="pagination-inline"><span class="table-meta">每页</span><select data-pagination-size="${prefix}">${sizes.map((size) => `<option value="${size}" ${size === pageSize ? "selected" : ""}>${size}</option>`).join("")}</select></label><label class="pagination-inline"><span class="table-meta">跳转到</span><input data-pagination-jump="${prefix}" type="number" min="1" max="${totalPages}" value="${page}" /></label><button class="ghost-button" data-action="${prefix}-jump">前往</button></div></div>`;
}

function bindPaginationControls(root, prefix, pager, totalPages, rerender) {
  bindAction(root, `${prefix}-prev`, () => {
    pager.page = Math.max(1, pager.page - 1);
    rerender();
  });
  bindAction(root, `${prefix}-next`, () => {
    pager.page = Math.min(totalPages, pager.page + 1);
    rerender();
  });
  bindAction(root, `${prefix}-jump`, () => {
    const input = root.querySelector(`[data-pagination-jump='${prefix}']`);
    const next = Number(input?.value || pager.page);
    pager.page = Math.min(totalPages, Math.max(1, Number.isFinite(next) ? next : pager.page));
    rerender();
  });
  root.querySelector(`[data-pagination-size='${prefix}']`)?.addEventListener("change", (event) => {
    const next = Number(event.target.value || pager.pageSize);
    pager.pageSize = Number.isFinite(next) ? next : pager.pageSize;
    pager.page = 1;
    rerender();
  });
  root.querySelector(`[data-pagination-jump='${prefix}']`)?.addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    const next = Number(event.target.value || pager.page);
    pager.page = Math.min(totalPages, Math.max(1, Number.isFinite(next) ? next : pager.page));
    rerender();
  });
}
