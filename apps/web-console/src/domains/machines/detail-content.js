import { state } from "../../core/state.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import {
  renderAccountAuthType,
  renderAssetAccessHint,
  renderAssetAccessMode,
  renderAssetGroup,
  renderEventLevel,
  renderEventLevelClass,
  renderEventType,
  renderLoginPolicy,
} from "./helpers.js";
export function renderAssetSessions() {
  if (!state.machine.assetSessions.length) {
    return emptyState("当前资产还没有独立会话记录。通过“打开 SSH 终端”后会在这里留下轨迹。");
  }
  return `<div class="machine-feed">${state.machine.assetSessions.map((session) => `
    <article class="machine-feed-card">
      <div class="machine-feed-head">
        <strong>${escapeHtml(session.account || "终端会话")}</strong>
        <span class="machine-status machine-status-${escapeHtml(session.status)}">${escapeHtml(renderSessionStatus(session.status))}</span>
      </div>
      <p>${escapeHtml(session.address)}</p>
      <span class="muted-label">${formatDateTime(session.started_at)}</span>
      <p class="machine-feed-copy">${escapeHtml(session.detail || "会话已记录")}</p>
    </article>
  `).join("")}</div>`;
}

export function renderAssetEvents() {
  if (!state.machine.assetEvents.length) {
    return emptyState("当前资产还没有操作轨迹。打开 SSH 终端并执行命令后，这里会记录连接、命令和关闭事件。");
  }
  return `<div class="machine-feed">${state.machine.assetEvents.map((event) => `
    <article class="machine-feed-card">
      <div class="machine-feed-head">
        <strong>${escapeHtml(event.summary || "操作事件")}</strong>
        <span class="machine-status machine-status-${renderEventLevelClass(event.event_level)}">${escapeHtml(renderEventLevel(event.event_level))}</span>
      </div>
      <p>${escapeHtml(renderEventType(event.event_type))}${event.session_id ? ` · 会话 #${event.session_id}` : ""}</p>
      <span class="muted-label">${formatDateTime(event.created_at)}</span>
      ${event.detail ? `<p class="machine-feed-copy"><code>${escapeHtml(event.detail)}</code></p>` : ""}
    </article>
  `).join("")}</div>`;
}

export function renderAssetAccounts() {
  const accounts = state.machine.accounts || [];
  if (!accounts.length) {
    return `${emptyState("当前资产还没有登录凭据。建议先保存账号密码或 SSH 私钥，再从凭据直接进入在线终端。")}<div class="action-row"><button class="ghost-button" data-machine-detail-action="add-account">新增凭据</button></div>`;
  }
  return `<div class="machine-feed">${accounts.map((account) => `
    <article class="machine-feed-card">
      <div class="machine-feed-head">
        <strong>${escapeHtml(account.name)}</strong>
        <span class="machine-status machine-status-${account.is_default ? "online" : "prepared"}">${account.is_default ? "默认" : "托管"}</span>
      </div>
      <p>${escapeHtml(account.username)} · ${escapeHtml(renderAccountAuthType(account.auth_type))}</p>
      <span class="muted-label">${formatDateTime(account.updated_at)}</span>
      ${renderProjectKeyBinding(account)}
      <p class="machine-feed-copy">${escapeHtml(account.description || "未填写账号说明")}</p>
      <div class="action-row">
        <button class="ghost-button" data-machine-account-action="connect" data-account-id="${account.id}">用此账号登录</button>
        <button class="ghost-button" data-machine-account-action="view" data-account-id="${account.id}">查看内容</button>
        ${account.is_default ? "" : `<button class="ghost-button" data-machine-account-action="default" data-account-id="${account.id}">设为默认</button>`}
        <button class="ghost-button danger-soft" data-machine-account-action="delete" data-account-id="${account.id}">删除</button>
      </div>
    </article>
  `).join("")}</div><div class="action-row"><button class="ghost-button" data-machine-detail-action="add-account">新增凭据</button></div>`;
}

function renderProjectKeyBinding(account) {
  const credential = (state.machine.credentials || []).find((item) =>
    String(item.source_type || "").toLowerCase() === "cloud_project_key" &&
    String(item.name || "") === String(account.name || "") &&
    String(item.username || "") === String(account.username || "")
  );
  if (!credential) return "";
  const scope = credential.scope_name || credential.scope_ref || "Foundation Network";
  return `<p class="machine-feed-copy"><strong>项目密钥</strong> · ${escapeHtml(credential.name || "-")} · ${escapeHtml(scope)}</p>`;
}

export function renderMachineAccessPanel(asset) {
  return `
    <div class="detail-list">
      <div class="detail-item"><strong class="muted-label">地址</strong><span>${escapeHtml(asset.address)}:${asset.port}</span></div>
      <div class="detail-item"><strong class="muted-label">资产分组</strong><span>${escapeHtml(renderAssetGroup(asset.group_name))}</span></div>
      <div class="detail-item"><strong class="muted-label">访问方式</strong><span>${escapeHtml(renderAssetAccessMode(asset.access_mode))}</span></div>
      <div class="detail-item"><strong class="muted-label">入口节点</strong><span>${escapeHtml(asset.gateway_asset_name || asset.gateway_address || (asset.access_mode === "via_gateway" ? "未指定" : "-"))}</span></div>
      <div class="detail-item"><strong class="muted-label">登录账号</strong><span>${escapeHtml(asset.account || "-")}</span></div>
      <div class="detail-item"><strong class="muted-label">登录策略</strong><span>${escapeHtml(renderLoginPolicy(asset.login_policy))}</span></div>
      <div class="detail-item"><strong class="muted-label">连接建议</strong><span>${escapeHtml(renderAssetAccessHint(asset))}</span></div>
      <div class="detail-item"><strong class="muted-label">连接面板</strong><span>${escapeHtml(asset.access_mode === "via_gateway" ? "打开 SSH 终端后，平台会先连接入口节点，再跳转到当前内网机器。" : "打开 SSH 终端后会进入下方连接面板，可同时保留多台机器的连接标签。")}</span></div>
    </div>
    <div class="action-row">
      <button class="primary-button" data-machine-detail-action="open-terminal">打开 SSH 终端</button>
      <button class="ghost-button" data-machine-detail-action="edit-asset">编辑资产</button>
      <button class="ghost-button" data-machine-detail-action="refresh-detail">刷新详情</button>
      <button class="ghost-button danger-soft" data-machine-detail-action="delete-asset">删除服务器</button>
    </div>
  `;
}

function renderSessionStatus(status) {
  const mapping = {
    prepared: "已准备",
    active: "进行中",
    closed: "已关闭",
  };
  return mapping[String(status || "").toLowerCase()] || "会话";
}
