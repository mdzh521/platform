import { api } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import { renderScopeBadges } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { confirmAction, showErrorDialog, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import {
  renderAccountAuthType,
  renderAssetAccessMode,
  renderAssetAccessHint,
  paginateItems,
  renderAssetGroup,
  renderLoginPolicy,
  renderAssetRiskLabel,
  renderAssetRiskTone,
  renderSessionStatus,
  renderStatus,
} from "./helpers.js";

let credentialLibraryActions = {
  openCredentialLibraryModal: () => {},
  openCredentialDetailModal: () => {},
  refreshMachineData: async () => {},
};

export function renderMachineSummary() {
  const summary = state.machine.summary || {};
  const visibleAssets = state.machine.assets || [];
  const filteredTotal = Number(state.machine.assetTotalItems || visibleAssets.length || 0);
  const groupCount = Object.keys(summary.group_breakup || {}).filter(Boolean).length;
  const online = Number(summary.online_assets || 0);
  const warning = Number(summary.warning_assets || 0);
  const offline = Number(summary.offline_assets || 0);
  const primaryPlatform = Object.entries(summary.platform_breakup || {}).sort((a, b) => Number(b[1] || 0) - Number(a[1] || 0))[0];
  const primaryProtocol = Object.entries(summary.protocol_breakup || {}).sort((a, b) => Number(b[1] || 0) - Number(a[1] || 0))[0];
  elements.machineSummaryCards.innerHTML = `
    <article class="summary-card summary-card-featured">
      <span class="muted-label">资产总数</span>
      <strong>${summary.total_assets || 0}</strong>
      <p>当前筛选下共 ${filteredTotal} 台，本页展示 ${visibleAssets.length} 台。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">在线 / 需关注</span>
      <strong>${online} / ${warning}</strong>
      <p>离线机器 ${offline} 台，优先处理告警资产。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">资产分组</span>
      <strong>${summary.total_groups || groupCount}</strong>
      <p>按分组查看资产入口。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">已录入登录账号</span>
      <strong>${(state.machine.assets || []).filter((asset) => String(asset.account || "").trim()).length}</strong>
      <p>已录入账号的资产可直接登录。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">近 24 小时会话</span>
      <strong>${summary.recent_sessions || 0}</strong>
      <p>用于查看最近连接轨迹。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">主要平台</span>
      <strong>${escapeHtml(primaryPlatform?.[0] || "未分类")}</strong>
      <p>${primaryPlatform ? `共 ${primaryPlatform[1]} 台，当前主要接入平台。` : "资产平台信息还不完整。"} </p>
    </article>
    <article class="summary-card">
      <span class="muted-label">主要协议</span>
      <strong>${escapeHtml(primaryProtocol?.[0] || "SSH")}</strong>
      <p>${primaryProtocol ? `共 ${primaryProtocol[1]} 台，后续可以继续扩展更多接入协议。` : "当前默认以 SSH 资产为主。"} </p>
    </article>
  `;
}

export function renderMachineGroups({ openAssetGroupModal, refreshMachineData }) {
  if (!elements.machineGroupPanel) return;
  const groups = state.machine.groups || [];
  const selected = state.machine.filters.group || "";
  const totalAssets = Number(state.machine.assetTotalItems || (state.machine.assets || []).length);
  if (!groups.length) {
    elements.machineGroupPanel.innerHTML = `${emptyState("当前还没有独立资产组。先创建一个资产组，再给机器归类。")}<div class="action-row"><button class="ghost-button" data-machine-group-action="create">新增资产组</button></div>`;
    bindMachineGroupActions({ openAssetGroupModal, refreshMachineData });
    return;
  }
  elements.machineGroupPanel.innerHTML = `
    <div class="machine-tree-list">
      <button class="machine-tree-item ${selected === "" ? "active" : ""}" data-machine-group-filter="">
        <span>全部资产</span>
        <strong>${totalAssets}</strong>
      </button>
      ${renderMachineGroupTreeRows(groups, selected)}
    </div>
    <div class="action-row">
      <button class="ghost-button" data-machine-group-action="create">新增资产组</button>
    </div>
  `;
  bindMachineGroupActions({ openAssetGroupModal, refreshMachineData });
}

export function renderMachineCredentialLibrary({ openCredentialLibraryModal, openCredentialDetailModal, refreshMachineData }) {
  credentialLibraryActions = { openCredentialLibraryModal, openCredentialDetailModal, refreshMachineData };
  if (!elements.machineCredentialFeed) return;
  const credentials = state.machine.credentials || [];
  if (!credentials.length) {
    elements.machineCredentialFeed.innerHTML = `${emptyState("当前还没有保存的登录凭据。可以先录入账号密码或 SSH 私钥，后续创建资产时直接选择。")}<div class="action-row"><button class="ghost-button" data-machine-credential-action="create">新增凭据</button></div>`;
    if (elements.machineCredentialPagination) {
    elements.machineCredentialPagination.innerHTML = "";
    }
    bindCredentialActions(credentialLibraryActions);
    return;
  }
  const pagination = paginateItems(credentials, state.machine.pagination.credentialsPage, 5);
  state.machine.pagination.credentialsPage = pagination.page;
  elements.machineCredentialFeed.innerHTML = `${pagination.items.map((credential) => `
    <article class="machine-feed-card">
      <div class="machine-feed-head">
        <strong>${escapeHtml(credential.name)}</strong>
        <span class="machine-status machine-status-prepared">${escapeHtml(renderAccountAuthType(credential.auth_type))}</span>
      </div>
      <p>${escapeHtml(credential.username)} · ${escapeHtml(credential.description || "未填写凭据说明")}</p>
      <span class="muted-label">${formatDateTime(credential.updated_at)}</span>
      <div class="action-row">
        <button class="ghost-button" data-machine-credential-action="view" data-credential-id="${credential.id}">查看内容</button>
        <button class="ghost-button danger-soft" data-machine-credential-action="delete" data-credential-id="${credential.id}">删除</button>
      </div>
    </article>
  `).join("")}<div class="action-row"><button class="ghost-button" data-machine-credential-action="create">新增凭据</button></div>`;
  bindCredentialActions(credentialLibraryActions);
  renderMachineSidePagination(elements.machineCredentialPagination, pagination, "credentials");
}

function bindCredentialActions({ openCredentialLibraryModal, openCredentialDetailModal, refreshMachineData }) {
  elements.machineCredentialFeed?.querySelectorAll("[data-machine-credential-action='create']").forEach((button) => {
    button.addEventListener("click", () => openCredentialLibraryModal());
  });
  elements.machineCredentialFeed?.querySelectorAll("[data-machine-credential-action='view']").forEach((button) => {
    button.addEventListener("click", () => {
      const id = Number(button.dataset.credentialId || 0);
      if (!id) return;
      openCredentialDetailModal(id);
    });
  });
  elements.machineCredentialFeed?.querySelectorAll("[data-machine-credential-action='delete']").forEach((button) => {
    button.addEventListener("click", async () => {
      const id = Number(button.dataset.credentialId || 0);
      if (!id) return;
      await api(`/api/v1/machines/credentials/${id}`, { method: "DELETE" });
      toast("凭据已删除");
      await refreshMachineData();
    });
  });
}

function bindMachineGroupActions({ openAssetGroupModal, refreshMachineData }) {
  elements.machineGroupPanel?.querySelectorAll("[data-machine-group-filter]").forEach((button) => {
    button.addEventListener("click", async () => {
      state.machine.filters.group = String(button.dataset.machineGroupFilter || "").trim();
      state.machine.pagination.page = 1;
      await refreshMachineData();
    });
  });
  elements.machineGroupPanel?.querySelectorAll("[data-machine-group-action='create']").forEach((button) => {
    button.addEventListener("click", () => openAssetGroupModal());
  });
  elements.machineGroupPanel?.querySelectorAll("[data-machine-group-action='edit']").forEach((button) => {
    button.addEventListener("click", () => {
      const id = Number(button.dataset.groupId || 0);
      const group = findMachineGroupByID(state.machine.groups || [], id);
      if (group) openAssetGroupModal(group);
    });
  });
  elements.machineGroupPanel?.querySelectorAll("[data-machine-group-action='delete']").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        const id = Number(button.dataset.groupId || 0);
        if (!id) return;
        const group = findMachineGroupByID(state.machine.groups || [], id);
        const confirmed = await confirmAction({
          eyebrow: "Asset Group",
          title: "删除资产组",
          copy: `确认删除资产组 ${group?.name || ""} ? 如果它还有下级资产组，后端会阻止删除。`,
          confirmText: "删除资产组",
        });
        if (!confirmed) return;
        await api(`/api/v1/machines/groups/${id}`, { method: "DELETE" });
        toast("资产组已删除");
        await refreshMachineData();
      } catch (error) {
        await showErrorDialog({
          title: "删除失败",
          copy: error?.message || "删除资产组失败",
        });
      }
    });
  });
}

function findMachineGroupByID(groups, id) {
  if (!Array.isArray(groups) || !id) return null;
  for (const group of groups) {
    if (Number(group?.id) === Number(id)) return group;
    const nested = findMachineGroupByID(group?.children || [], id);
    if (nested) return nested;
  }
  return null;
}

export function renderMachineAssets(refreshMachineData = async () => {}) {
  const assets = state.machine.assets || [];
  const selectedAssetIDs = new Set((state.machine.selectedAssetIDs || []).map((item) => Number(item)));
  const selectedCount = assets.filter((asset) => selectedAssetIDs.has(Number(asset.id))).length;
  const pagination = {
    items: assets,
    page: Number(state.machine.pagination.page || 1),
    pageSize: Number(state.machine.pagination.pageSize || 10),
    totalItems: Number(state.machine.assetTotalItems || assets.length),
    totalPages: Number(state.machine.assetTotalPages || 1),
    startIndex: assets.length ? (Number(state.machine.pagination.page || 1) - 1) * Number(state.machine.pagination.pageSize || 10) : 0,
    endIndex: assets.length ? ((Number(state.machine.pagination.page || 1) - 1) * Number(state.machine.pagination.pageSize || 10)) + assets.length : 0,
  };
  if (elements.machinePageSize) {
    elements.machinePageSize.value = String(pagination.pageSize);
  }
  if (!assets.length) {
    elements.machineAssetsTable.innerHTML = `${emptyState("暂无机器资产。先用“快速登录”录入一台主机，后续再接账号托管和在线终端。")}<div class="action-row" style="margin-top:12px;"><button class="ghost-button" data-machine-batch-action="create">批量新增服务器</button></div>`;
    if (elements.machineAssetsPagination) {
      elements.machineAssetsPagination.innerHTML = "";
    }
    return;
  }

  elements.machineAssetsTable.innerHTML = `
    <div class="machine-assets-toolbar">
      <div class="machine-assets-toolbar-copy">
        <strong>资产列表</strong>
        <span class="muted-label">当前 ${assets.length} 台，已选 ${selectedCount} 台</span>
      </div>
      <div class="action-row">
        <button class="ghost-button" data-machine-batch-action="create">批量新增</button>
      </div>
    </div>
    ${selectedCount ? `
      <div class="machine-assets-selectionbar">
        <div class="machine-assets-selectioncopy">
          <strong>已选 ${selectedCount} 台服务器</strong>
          <span class="muted-label">可以统一更新标签、端口、账号密码等字段</span>
        </div>
        <div class="action-row">
          <button class="ghost-button" data-machine-batch-action="clear">清空选择</button>
          <button class="ghost-button" data-machine-batch-action="update">批量更新</button>
          <button class="ghost-button danger-soft" data-machine-batch-action="delete">批量删除</button>
        </div>
      </div>
    ` : ""}
    <div class="machine-assets-tableview machine-assets-gridview">
      <div class="machine-assets-listhead">
        <label class="machine-assets-headcheck">
          <input type="checkbox" data-machine-batch-toggle="page" ${selectedCount === assets.length ? "checked" : ""} />
        </label>
        <div class="machine-assets-col machine-assets-col-asset">
          ${renderMachineSortHeader("资产名称", "name")}
        </div>
        <div class="machine-assets-col">分组 / 平台</div>
        <div class="machine-assets-col">状态 / 风险</div>
        <div class="machine-assets-col">登录账号</div>
        <div class="machine-assets-col">登录策略</div>
        <div class="machine-assets-col">最近活跃</div>
        <div class="machine-assets-col machine-assets-col-actions">操作</div>
      </div>
      <div class="machine-assets-listmeta">
        <strong>资产清单</strong>
        <span class="muted-label">按行查看服务器身份、状态和常用操作</span>
      </div>
      ${assets.map((asset, index) => `
        <article data-asset-row="${asset.id}" class="machine-asset-row machine-asset-table-row ${Number(state.machine.selectedAssetID) === Number(asset.id) ? "active-row" : ""}">
          <div class="machine-asset-table-check">
            <label class="machine-asset-select">
              <input type="checkbox" data-machine-batch-toggle="asset" data-asset-id="${asset.id}" ${selectedAssetIDs.has(Number(asset.id)) ? "checked" : ""} />
            </label>
          </div>
          <div class="machine-asset-table-cell machine-asset-table-asset">
            <div class="machine-asset-titleline">
              <span class="table-index">${pagination.startIndex + index + 1}</span>
              <strong>${escapeHtml(asset.name)}</strong>
              ${asset.is_gateway_node ? `<span class="network-plan-badge">入口节点</span>` : ""}
            </div>
            <p class="machine-asset-endpoint">${escapeHtml(asset.address)}:${asset.port}</p>
            <p class="machine-asset-subcopy">${escapeHtml(asset.description || renderAssetAccessHint(asset))}</p>
            <div class="scope-badge-row">
              ${renderScopeBadges({ projectID: asset.project_id, environmentID: asset.environment_id, stackID: asset.stack_id })}
            </div>
          </div>
          <div class="machine-asset-table-cell">
            <strong>${escapeHtml(renderAssetGroup(asset.group_name))}</strong>
            <p class="machine-asset-subcopy">${escapeHtml(asset.platform || "linux")} / ${escapeHtml(asset.protocol || "ssh")} / ${escapeHtml(renderAssetAccessMode(asset.access_mode))}</p>
            ${asset.gateway_asset_name ? `<p class="machine-asset-subcopy">入口节点：${escapeHtml(asset.gateway_asset_name)}</p>` : ""}
          </div>
          <div class="machine-asset-table-cell">
            <div class="machine-asset-head-status">
              <span class="machine-status machine-status-${escapeHtml(asset.status)}">${escapeHtml(renderStatus(asset.status))}</span>
              <span class="machine-signal machine-signal-${renderAssetRiskTone(asset)}">${escapeHtml(renderAssetRiskLabel(asset))}</span>
            </div>
            <p class="machine-asset-subcopy">${escapeHtml(`纳管 ${asset.enrollment_status || "-"}`)}</p>
            ${asset.enrollment_error ? `<p class="machine-asset-subcopy">${escapeHtml(asset.enrollment_error)}</p>` : ""}
          </div>
          <div class="machine-asset-table-cell">
            <strong>${escapeHtml(asset.account || "未录入")}</strong>
          </div>
          <div class="machine-asset-table-cell">
            <strong>${escapeHtml(renderLoginPolicy(asset.login_policy))}</strong>
          </div>
          <div class="machine-asset-table-cell">
            <strong>${formatDateTime(asset.last_seen_at || asset.updated_at)}</strong>
          </div>
          <div class="machine-asset-table-cell machine-asset-table-actions">
            <button class="primary-button" data-machine-action="connect" data-asset-id="${asset.id}">登录</button>
            <button class="ghost-button" data-machine-action="credentials" data-asset-id="${asset.id}">凭据</button>
            <button class="ghost-button" data-machine-action="detail" data-asset-id="${asset.id}">详情</button>
            <button class="ghost-button danger-soft" data-machine-action="delete" data-asset-id="${asset.id}">删除</button>
          </div>
        </article>
      `).join("")}
    </div>
  `;
  renderMachineAssetPagination(pagination, refreshMachineData);
}

export function renderMachineAccessWorkspace(refreshMachineData = async () => {}) {
  renderMachineAccessGroups();
  if (!elements.machineAccessAssetsTable) return;
  elements.machineAccessAssetsTable.innerHTML = "";
  if (elements.machineAccessAssetsPagination) {
    elements.machineAccessAssetsPagination.innerHTML = "";
  }
}

function renderMachineSortHeader(label, sortBy) {
  const active = state.machine.filters.sortBy === sortBy;
  const order = active ? (state.machine.filters.sortOrder || "asc") : "";
  const indicator = active ? (order === "asc" ? "↑" : "↓") : "↕";
  return `
    <button class="machine-sort-button ${active ? "active" : ""}" type="button" data-machine-sort="${sortBy}">
      <span>${escapeHtml(label)}</span>
      <strong>${indicator}</strong>
    </button>
  `;
}

function renderMachineAssetPagination(pagination, refreshMachineData) {
  if (!elements.machineAssetsPagination) return;
  const canPrev = pagination.page > 1;
  const canNext = pagination.page < pagination.totalPages;
  const pageOptions = Array.from({ length: pagination.totalPages }, (_, index) => {
    const value = index + 1;
    return `<option value="${value}" ${value === pagination.page ? "selected" : ""}>第 ${value} 页</option>`;
  }).join("");
  elements.machineAssetsPagination.innerHTML = `
    <div class="pagination-summary">
      <span>共 ${pagination.totalItems} 台资产</span>
      <strong>${pagination.startIndex + 1}-${pagination.endIndex}</strong>
    </div>
    <div class="pagination-actions">
      <button class="ghost-button" data-machine-page-action="prev" ${canPrev ? "" : "disabled"}>上一页</button>
      <select class="toolbar-input machine-page-jump" data-machine-page-action="jump">
        ${pageOptions}
      </select>
      <button class="ghost-button" data-machine-page-action="next" ${canNext ? "" : "disabled"}>下一页</button>
    </div>
  `;
  elements.machineAssetsPagination.querySelectorAll("[data-machine-page-action='prev']").forEach((button) => {
    button.addEventListener("click", async () => {
      state.machine.pagination.page = Math.max(1, pagination.page - 1);
      await refreshMachineData();
    });
  });
  elements.machineAssetsPagination.querySelectorAll("[data-machine-page-action='next']").forEach((button) => {
    button.addEventListener("click", async () => {
      state.machine.pagination.page = Math.min(pagination.totalPages, pagination.page + 1);
      await refreshMachineData();
    });
  });
  elements.machineAssetsPagination.querySelectorAll("[data-machine-page-action='jump']").forEach((select) => {
    select.addEventListener("change", async (event) => {
      state.machine.pagination.page = Number(event.target.value || pagination.page) || pagination.page;
      await refreshMachineData();
    });
  });
}

function renderMachineAccessGroups() {
  if (!elements.machineAccessGroupPanel) return;
  const groups = state.machine.groups || [];
  const selected = state.machine.filters.group || "";
  const assets = state.machine.assets || [];
  const totalAssets = Number(state.machine.assetTotalItems || assets.length);
  elements.machineAccessGroupPanel.innerHTML = `
    <div class="machine-tree-list">
      <button class="machine-tree-item ${selected === "" ? "active" : ""}" data-machine-group-filter="">
        <span>全部机器</span>
        <strong>${totalAssets}</strong>
      </button>
      ${renderMachineAccessGroupTreeRows(groups, assets, selected)}
      ${renderMachineUngroupedAssetRows(assets, selected === "" ? true : false)}
    </div>
  `;
}

function renderMachineGroupTreeRows(groups, selected, depth = 0) {
  if (!Array.isArray(groups) || !groups.length) return "";
  return groups.map((group) => `
    <div class="machine-tree-row ${selected === group.name ? "active" : ""}">
      <button class="machine-tree-item ${selected === group.name ? "active" : ""}" data-machine-group-filter="${escapeHtml(group.name)}" style="padding-left:${16 + depth * 16}px">
        <span>${escapeHtml(group.name)}</span>
        <strong>${group.asset_count}</strong>
      </button>
      <div class="machine-tree-actions">
        <button class="ghost-button" data-machine-group-action="edit" data-group-id="${group.id}">编辑</button>
        <button class="ghost-button danger-soft" data-machine-group-action="delete" data-group-id="${group.id}">删除</button>
      </div>
    </div>
    ${renderMachineGroupTreeRows(group.children || [], selected, depth + 1)}
  `).join("");
}

function renderMachineAccessGroupTreeRows(groups, assets, selected, depth = 0) {
  if (!Array.isArray(groups) || !groups.length) return "";
  return groups.map((group) => `
    <details class="machine-access-group-node" ${isMachineAccessGroupExpanded(group, depth) ? "open" : ""}>
      <summary class="machine-tree-item ${selected === group.name ? "active" : ""}" data-machine-access-group-toggle="${escapeHtml(group.name)}" style="padding-left:${16 + depth * 16}px">
        <span>${escapeHtml(group.name)}</span>
        <strong>${group.asset_count}</strong>
      </summary>
      <div class="machine-access-group-children">
        ${renderMachineAccessAssetRows(assets.filter((asset) => String(asset.group_name || "").trim() === String(group.name || "").trim()), depth + 1)}
        ${renderMachineAccessGroupTreeRows(group.children || [], assets, selected, depth + 1)}
      </div>
    </details>
  `).join("");
}

function renderMachineAccessAssetRows(assets, depth = 0) {
  if (!Array.isArray(assets) || !assets.length) return "";
  return assets.map((asset) => `
    <button class="machine-access-asset-node ${Number(state.machine.selectedAssetID) === Number(asset.id) ? "active" : ""}" data-access-asset-row="${asset.id}" style="padding-left:${28 + depth * 16}px">
      <div class="machine-access-asset-main">
        <strong>${escapeHtml(asset.name)}</strong>
        <span>${escapeHtml(asset.address)}:${asset.port}</span>
        <small>${escapeHtml(asset.default_account_username || asset.account || "未录入登录账号")}</small>
      </div>
      <div class="machine-access-row-status">
        <span class="machine-access-dot machine-access-dot-${escapeHtml(asset.status)}" aria-hidden="true"></span>
      </div>
    </button>
  `).join("");
}

function renderMachineUngroupedAssetRows(assets, showByDefault) {
  const ungrouped = (assets || []).filter((asset) => !String(asset.group_name || "").trim());
  if (!ungrouped.length) return "";
  return `
    <details class="machine-access-group-node" ${showByDefault ? "open" : ""}>
      <summary class="machine-tree-item">
        <span>未分组机器</span>
        <strong>${ungrouped.length}</strong>
      </summary>
      <div class="machine-access-group-children">
        ${renderMachineAccessAssetRows(ungrouped, 1)}
      </div>
    </details>
  `;
}

function isMachineAccessGroupExpanded(group, depth) {
  const key = String(group?.name || "").trim();
  if (!key) return depth < 1;
  if (Object.prototype.hasOwnProperty.call(state.machine.expandedGroups || {}, key)) {
    return Boolean(state.machine.expandedGroups[key]);
  }
  return depth < 1;
}

export function renderMachineSessions() {
  if (!state.machine.sessions.length) {
    elements.machineSessionFeed.innerHTML = emptyState("最近暂无机器会话。通过“快速登录”或资产列表的“登录”可以留下会话轨迹。");
    if (elements.machineSessionPagination) {
      elements.machineSessionPagination.innerHTML = "";
    }
    return;
  }
  const pagination = paginateItems(state.machine.sessions, state.machine.pagination.sessionsPage, 5);
  state.machine.pagination.sessionsPage = pagination.page;
  elements.machineSessionFeed.innerHTML = pagination.items.map((session) => `
    <article class="machine-feed-card">
      <div class="machine-feed-head">
        <strong>${escapeHtml(session.asset_name)}</strong>
        <span class="machine-status machine-status-${escapeHtml(session.status)}">${escapeHtml(renderSessionStatus(session.status))}</span>
      </div>
      <p>${escapeHtml(session.account || "-")} · ${escapeHtml(session.address)}</p>
      <span class="muted-label">${formatDateTime(session.started_at)}</span>
      <p class="machine-feed-copy">${escapeHtml(session.detail || "会话已记录")}</p>
    </article>
  `).join("");
  renderMachineSidePagination(elements.machineSessionPagination, pagination, "sessions");
}

function renderMachineSidePagination(root, pagination, type) {
  if (!root) return;
  if (pagination.totalItems <= pagination.pageSize) {
    root.innerHTML = "";
    return;
  }
  root.innerHTML = `
    <div class="pagination-summary">
      <span>共 ${pagination.totalItems} 条</span>
      <strong>${pagination.startIndex + 1}-${pagination.endIndex}</strong>
    </div>
    <div class="pagination-actions">
      <button class="ghost-button" data-machine-side-page="${type}-prev" ${pagination.page > 1 ? "" : "disabled"}>上一页</button>
      <button class="ghost-button" data-machine-side-page="${type}-next" ${pagination.page < pagination.totalPages ? "" : "disabled"}>下一页</button>
    </div>
  `;
  root.querySelectorAll(`[data-machine-side-page='${type}-prev']`).forEach((button) => {
    button.addEventListener("click", () => {
      if (type === "credentials") {
        state.machine.pagination.credentialsPage = Math.max(1, pagination.page - 1);
        renderMachineCredentialLibrary(credentialLibraryActions);
        return;
      }
      state.machine.pagination.sessionsPage = Math.max(1, pagination.page - 1);
      renderMachineSessions();
    });
  });
  root.querySelectorAll(`[data-machine-side-page='${type}-next']`).forEach((button) => {
    button.addEventListener("click", () => {
      if (type === "credentials") {
        state.machine.pagination.credentialsPage = Math.min(pagination.totalPages, pagination.page + 1);
        renderMachineCredentialLibrary(credentialLibraryActions);
        return;
      }
      state.machine.pagination.sessionsPage = Math.min(pagination.totalPages, pagination.page + 1);
      renderMachineSessions();
    });
  });
}
