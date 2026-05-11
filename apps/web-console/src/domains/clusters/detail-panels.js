import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { bindAction } from "../../core/ui.js";
import { emptyState, escapeHtml } from "../../shared/utils.js";
import { openEditSelectedResourceModal, supportsResourceEdit } from "./resource-edit.js";
import { renderResourceDetailContent } from "./resource-detail-content.js";
import { configureWorkloadDetailPanel, switchWorkloadPage } from "./workload-detail.js";

let ctx = null;

export function configureK8sDetailPanels(deps) {
  ctx = deps;
  configureWorkloadDetailPanel(deps);
}

export function renderResourceExplorerPanel() {
  switchResourcePage(state.business.resourceListPage);
  const selected = state.resourceExplorer.items.find((item) => item.name === state.business.selectedResourceName) || state.resourceExplorer.detail;
  if (!selected) {
    elements.resourceDetailPanel.innerHTML = emptyState("选择一个资源后，这里会展示详细信息和 YAML");
    return;
  }
  const kind = state.resourceExplorer.detail?.kind || selected.kind;
  const name = state.resourceExplorer.detail?.name || selected.name;
  const editButton = supportsResourceEdit(kind) ? `<button class="ghost-button" data-action="edit-selected-resource">编辑资源</button>` : "";
  elements.resourceDetailPanel.innerHTML = `<div class="detail-stack"><div><p class="eyebrow">${escapeHtml(kind)}</p><h4>${escapeHtml(name)}</h4><p class="table-meta">资源详情页优先展示摘要信息和 YAML，避免把底层对象一次性全量铺开。</p></div><div class="detail-toolbar-shell"><div class="detail-toolbar"><button class="ghost-button active" data-action="resource-overview">基础信息</button><button class="ghost-button ${state.business.resourceManifestMode === "compact" ? "active" : ""}" data-action="resource-manifest-compact">简洁 YAML</button><button class="ghost-button ${state.business.resourceManifestMode === "full" ? "active" : ""}" data-action="resource-manifest-full">完整 YAML</button></div><div class="detail-toolbar detail-toolbar-actions">${editButton}<button class="ghost-button danger-soft" data-action="delete-selected-resource">删除资源</button></div></div><div>${renderResourceDetailContent()}</div></div>`;
  bindAction(elements.resourceDetailPanel, "resource-overview", () => renderResourceExplorerPanel());
  bindAction(elements.resourceDetailPanel, "resource-manifest-compact", () => ctx.runK8sAction(() => ctx.switchResourceManifestMode("compact")));
  bindAction(elements.resourceDetailPanel, "resource-manifest-full", () => ctx.runK8sAction(() => ctx.switchResourceManifestMode("full")));
  bindAction(elements.resourceDetailPanel, "open-referenced-workload", (payload) => ctx.runK8sAction(() => ctx.openReferencedWorkload(String(payload))));
  bindAction(elements.resourceDetailPanel, "edit-selected-resource", () => ctx.runK8sAction(() => openEditSelectedResourceModal()));
  bindAction(elements.resourceDetailPanel, "delete-selected-resource", () => ctx.runK8sAction(() => ctx.deleteNamespaceResource(kind, name)));
}

export function switchResourcePage(view) {
  state.business.resourceListPage = view;
  elements.resourceListPage.classList.remove("hidden");
  elements.resourceDetailPage.classList.toggle("hidden", view !== "detail");
}
