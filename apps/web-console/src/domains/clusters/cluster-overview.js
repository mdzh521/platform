import { elements } from "../../core/dom.js";
import { renderScopeSummary } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { bindAction } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";

let ctx = null;

export function configureClusterOverview(deps) {
  ctx = deps;
}

export function renderClusterDetailPanel() {
  switchClusterPage(state.business.clusterListPage);
  const overview = state.clusterInspector.overview;
  if (!overview || state.business.clusterListPage !== "detail") {
    elements.clusterDetailPanel.innerHTML = emptyState("选择一个集群后，这里会展示节点、最近事件和命名空间治理信息");
    return;
  }
  const summary = overview.summary || {};
  const cluster = overview.cluster || {};
  const nodes = overview.nodes || [];
  const recentEvents = overview.recent_events || [];
  const namespaceGovernance = (overview.namespace_governance || []).filter((item) => !state.business.hideSystemNamespaces || !String(item.name || "").startsWith("kube-"));
  elements.clusterDetailPanel.innerHTML = `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">集群名称</strong><span>${escapeHtml(cluster.name || "-")}</span></div><div class="detail-item"><strong class="muted-label">环境 / 供应商</strong><span>${escapeHtml(`${cluster.environment || "-"} / ${cluster.provider || "-"}`)}</span></div><div class="detail-item"><strong class="muted-label">归属</strong><span>${escapeHtml(renderScopeSummary({ projectID: cluster.project_id, environmentID: cluster.environment_id, stackID: cluster.stack_id, fallbackEnvironment: cluster.environment }))}</span></div><div class="detail-item"><strong class="muted-label">API 地址</strong><span class="code-cell">${escapeHtml(cluster.api_endpoint || "-")}</span></div><div class="detail-item"><strong class="muted-label">最近检测</strong><span>${escapeHtml(formatDateTime(cluster.last_checked_at))}</span></div></div><div class="rollout-grid">${[
    { label: "节点数", value: summary.node_count ?? 0 },
    { label: "Ready 节点", value: summary.ready_node_count ?? 0 },
    { label: "最近事件", value: summary.recent_event_count ?? 0 },
    { label: "命名空间", value: summary.namespace_count ?? 0 },
    { label: "有配额的命名空间", value: summary.namespaces_with_quota ?? 0 },
    { label: "有 LimitRange 的命名空间", value: summary.namespaces_with_limit_range ?? 0 },
  ].map(ctx.renderDetailCard).join("")}</div>${renderClusterNodes(nodes)}${renderClusterEvents(recentEvents)}${renderNamespaceGovernance(namespaceGovernance)}</div>`;
  bindAction(elements.clusterDetailPanel, "open-governance-workloads", (name) => ctx.runK8sAction(() => ctx.openNamespaceWorkloads(findNamespaceByName(String(name)))));
  bindAction(elements.clusterDetailPanel, "open-governance-resources", (name) => ctx.runK8sAction(() => ctx.openNamespaceResources(findNamespaceByName(String(name)))));
}

function renderClusterNodes(nodes) {
  if (!nodes.length) {
    return `<section><p class="eyebrow">Nodes</p>${emptyState("当前集群没有返回节点信息")}</section>`;
  }
  return `<section><p class="eyebrow">Nodes</p><table><thead><tr><th>序号</th><th>名称</th><th>角色</th><th>状态</th><th>IP / PodCIDR</th><th>版本</th><th>系统</th></tr></thead><tbody>${nodes.map((item, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">创建于 ${formatDateTime(item.created_at)}</span></div></td><td>${(item.roles || []).length ? (item.roles || []).map((role) => `<span class="cluster-chip">${escapeHtml(role)}</span>`).join("") : "<span class='muted-label'>worker</span>"}</td><td><span class="status-pill ${item.ready ? "" : "disabled"}">${item.ready ? "Ready" : "NotReady"}</span></td><td><div class="cell-stack"><span>${escapeHtml(item.internal_ip || "-")}</span><span class="table-meta">${escapeHtml(item.pod_cidr || "-")}</span></div></td><td>${escapeHtml(item.kubelet_version || "-")}</td><td>${escapeHtml(item.os_image || "-")}</td></tr>`).join("")}</tbody></table></section>`;
}

function renderClusterEvents(items) {
  if (!items.length) {
    return `<section><p class="eyebrow">Recent Events</p>${emptyState("当前集群没有可展示的事件")}</section>`;
  }
  return `<section><p class="eyebrow">Recent Events</p><div class="event-list">${items.map((item) => `<article class="event-item"><p class="muted-label">${escapeHtml(item.namespace || "-")} / ${escapeHtml(item.type || "-")} / ${escapeHtml(item.component || "kubernetes")}</p><strong>${escapeHtml(item.involved_kind || "-")} / ${escapeHtml(item.involved_name || "-")} / ${escapeHtml(item.reason || "-")}</strong><p>${escapeHtml(item.message || "-")}</p><p class="table-meta">次数 ${escapeHtml(item.count || 1)}，时间 ${escapeHtml(formatDateTime(item.timestamp))}</p></article>`).join("")}</div></section>`;
}

function renderNamespaceGovernance(items) {
  if (!items.length) {
    return `<section><p class="eyebrow">Namespace Governance</p>${emptyState("当前集群没有命名空间治理数据")}</section>`;
  }
  return `<section><p class="eyebrow">Namespace Governance</p><table><thead><tr><th>序号</th><th>命名空间</th><th>状态</th><th>ResourceQuota</th><th>LimitRange</th><th>操作</th></tr></thead><tbody>${items.map((item, index) => `<tr><td>${index + 1}</td><td><strong>${escapeHtml(item.name)}</strong></td><td><span class="status-pill ${item.status === "Active" ? "" : "disabled"}">${escapeHtml(item.status || "-")}</span></td><td>${escapeHtml(String(item.resource_quota_count ?? 0))}</td><td>${escapeHtml(String(item.limit_range_count ?? 0))}</td><td><div class="action-row"><button class="ghost-button" data-action="open-governance-workloads" data-id="${escapeHtml(item.name)}">工作负载</button><button class="ghost-button" data-action="open-governance-resources" data-id="${escapeHtml(item.name)}">资源</button></div></td></tr>`).join("")}</tbody></table></section>`;
}

export function switchClusterPage(view) {
  state.business.clusterListPage = view;
  elements.clusterListPage.classList.toggle("hidden", view !== "list");
  elements.clusterDetailPage.classList.toggle("hidden", view !== "detail");
  elements.businessContext.textContent = ctx.businessContextText();
}

function findNamespaceByName(name) {
  const clusterID = state.business.selectedClusterID;
  return state.namespaces.find((item) => item.cluster_id === clusterID && item.name === name) || {
    cluster_id: clusterID,
    cluster_name: state.business.selectedClusterName,
    name,
  };
}
