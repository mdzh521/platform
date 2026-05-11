import { api } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import { renderScopeSummary } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { bindAction, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import {
  renderAssetGroup,
  renderFoldSection,
  renderLoginPolicy,
  renderMetaPill,
  renderStatus,
} from "./helpers.js";
import {
  renderAssetAccounts,
  renderAssetEvents,
  renderAssetSessions,
  renderMachineAccessPanel,
} from "./detail-content.js";
import {
  openAssetAccess,
  openManagedAccountTerminal,
} from "./terminal.js";

export function switchMachineView(view) {
  const normalizedView = view === "terminal" ? "access" : view;
  state.machine.view = normalizedView;
  elements.machineListPage?.classList.toggle("hidden", normalizedView !== "list");
  elements.machineAccessPage?.classList.toggle("hidden", normalizedView !== "access");
  elements.machineDetailPage?.classList.toggle("hidden", normalizedView !== "detail");
  elements.machineNavList?.classList.toggle("active", normalizedView === "list" || normalizedView === "detail");
  elements.machineNavAccess?.classList.toggle("active", normalizedView === "access");
  elements.machineNavTerminal?.classList.remove("active");
}

export function resetMachineDetailState(machineCtx) {
  state.machine.view = "list";
  state.machine.selectedAssetID = null;
  state.machine.detail = null;
  state.machine.groups = state.machine.groups || [];
  state.machine.accounts = [];
  state.machine.assetSessions = [];
  state.machine.assetEvents = [];
}

export async function openAssetDetail(machineCtx, assetID, keepView = false) {
  if (!assetID) return;
  const [assetPayload, sessionsPayload, accountsPayload, eventsPayload] = await Promise.all([
    api(`/api/v1/machines/assets/${assetID}`),
    api(`/api/v1/machines/assets/${assetID}/sessions`),
    api(`/api/v1/machines/assets/${assetID}/accounts`),
    api(`/api/v1/machines/assets/${assetID}/events`),
  ]);
  state.machine.selectedAssetID = assetID;
  state.machine.detail = assetPayload.data || null;
  state.machine.accounts = accountsPayload.data || [];
  state.machine.assetSessions = sessionsPayload.data || [];
  state.machine.assetEvents = eventsPayload.data || [];
  machineCtx.renderMachineWorkspace();
  switchMachineView("detail");
  window.requestAnimationFrame(() => elements.machineDetailPage?.scrollIntoView({ behavior: "smooth", block: "start" }));
}

export function renderMachineDetail(machineCtx) {
  const asset = state.machine.detail;
  if (!asset) {
    elements.machineDetailTitle.textContent = "机器详情";
    elements.machineDetailPanel.innerHTML = emptyState("选择一台机器后，这里会展示资产详情、最近会话和在线 SSH 终端。");
    return;
  }
  elements.machineDetailTitle.textContent = asset.name;
  elements.machineDetailPanel.innerHTML = `
    <section class="detail-heading machine-object-head">
      <div class="machine-object-identity">
        <div>
          <p class="eyebrow">Machine Detail</p>
          <h4>${escapeHtml(asset.name)}</h4>
        </div>
        <span class="machine-status machine-status-${escapeHtml(asset.status)}">${escapeHtml(renderStatus(asset.status))}</span>
      </div>
      <div class="detail-meta-row machine-object-meta">
        ${renderMetaPill("机器地址", `${asset.address}:${asset.port}`)}
        ${renderMetaPill("资产分组", renderAssetGroup(asset.group_name))}
        ${renderMetaPill("登录账号", asset.account || "-")}
        ${renderMetaPill("归属", renderScopeSummary({ projectID: asset.project_id, environmentID: asset.environment_id, stackID: asset.stack_id }))}
        ${renderMetaPill("登录策略", renderLoginPolicy(asset.login_policy))}
        ${renderMetaPill("纳管状态", asset.enrollment_status || "-")}
        ${renderMetaPill("最近活跃", formatDateTime(asset.last_seen_at || asset.updated_at))}
      </div>
    </section>
    <section class="machine-detail-mosaic">
      <div class="machine-detail-card machine-detail-card-main">
        ${renderFoldSection("连接入口", renderMachineAccessPanel(asset), { open: true })}
        ${renderFoldSection("登录凭据", renderAssetAccounts(), { open: true, meta: `${state.machine.accounts.length} 个` })}
      </div>
      <div class="machine-detail-card machine-detail-card-side">
        ${renderFoldSection("最近会话", renderAssetSessions(), { open: true, meta: `${state.machine.assetSessions.length} 条` })}
      </div>
      <div class="machine-detail-card machine-detail-card-meta">
        ${renderFoldSection("资产说明", `<p class="machine-detail-copy">${escapeHtml(asset.description || "未填写资产说明")}</p>${asset.enrollment_error ? `<p class="machine-detail-copy">${escapeHtml(asset.enrollment_error)}</p>` : ""}`, { open: true })}
      </div>
      <div class="machine-detail-card machine-detail-card-audit">
        ${renderFoldSection("操作轨迹", renderAssetEvents(), { open: false, meta: `${state.machine.assetEvents.length} 条` })}
      </div>
    </section>
  `;
  bindMachineDetailActions(machineCtx, asset);
}

function bindMachineDetailActions(machineCtx, asset) {
  elements.machineDetailPanel.querySelectorAll("[data-machine-detail-action='open-terminal']").forEach((button) => {
    button.addEventListener("click", () => openAssetAccess(machineCtx, asset.id));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-detail-action='add-account']").forEach((button) => {
    button.addEventListener("click", () => machineCtx.openAccountModal(asset));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-detail-action='edit-asset']").forEach((button) => {
    button.addEventListener("click", () => machineCtx.openAssetModal(asset));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-detail-action='delete-asset']").forEach((button) => {
    button.addEventListener("click", () => machineCtx.deleteAsset(asset));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-detail-action='refresh-detail']").forEach((button) => {
    button.addEventListener("click", () => openAssetDetail(machineCtx, asset.id, true));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-account-action='connect']").forEach((button) => {
    button.addEventListener("click", () => openManagedAccountTerminal(machineCtx, asset, Number(button.dataset.accountId || 0)));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-account-action='view']").forEach((button) => {
    button.addEventListener("click", () => machineCtx.openAccountDetailModal(asset.id, Number(button.dataset.accountId || 0)));
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-account-action='default']").forEach((button) => {
    button.addEventListener("click", async () => {
      const accountID = Number(button.dataset.accountId || 0);
      if (!accountID) return;
      await api(`/api/v1/machines/assets/${asset.id}/accounts/${accountID}/default`, { method: "POST" });
      toast("默认托管账号已更新");
      await openAssetDetail(machineCtx, asset.id, true);
      await machineCtx.refreshMachineData();
    });
  });
  elements.machineDetailPanel.querySelectorAll("[data-machine-account-action='delete']").forEach((button) => {
    button.addEventListener("click", async () => {
      const accountID = Number(button.dataset.accountId || 0);
      if (!accountID) return;
      await api(`/api/v1/machines/assets/${asset.id}/accounts/${accountID}`, { method: "DELETE" });
      toast("托管账号已删除");
      await openAssetDetail(machineCtx, asset.id, true);
    });
  });
}
