import { api, buildAPIURL, buildWebSocketURL } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { applyTerminalChunk, escapeHtml, formatDateTime } from "../../shared/utils.js";

const terminalSockets = new Map();
const terminalHeartbeats = new Map();
let machineTerminalResizeTimer = null;
let machineTerminalResizeBound = false;
let machineTerminalFullscreenBound = false;
let machineTerminalVisibilityBound = false;
let machineTerminalDelegatedBound = false;
let machineXterm = null;
let machineXtermKey = null;
let machineTerminalCtx = null;
const MACHINE_TERMINAL_RENDER_MODE = "xterm";
const MACHINE_TERMINAL_HEARTBEAT_MS = 20000;

export function renderMachineTerminalDock() {
  if (!elements.machineTerminalDock) return;
  const terminals = state.machine.terminals || [];
  if (!terminals.length) {
    elements.machineTerminalDock.innerHTML = `
      <div class="machine-terminal-empty">
        <div class="machine-terminal-chrome machine-terminal-chrome-static">
          <div class="machine-terminal-shell-title">terminal-workbench</div>
          <div class="machine-terminal-toolbar-spacer"></div>
        </div>
        <div class="machine-terminal-empty-body">
          <strong>选择左侧机器后即可直接打开 SSH 终端</strong>
          <p>工作台会保留多个连接标签，你可以在不同机器之间快速切换。</p>
        </div>
      </div>
    `;
    return;
  }
  const active = getActiveTerminal();
  if (!active) {
    elements.machineTerminalDock.innerHTML = '<div class="empty-state">当前没有活动终端。</div>';
    return;
  }
  ensureMachineTerminalSelection(active.key);
  const errorText = active.error ? `<p class="table-meta terminal-status-error">${escapeHtml(active.error)}</p>` : "";
  const disconnectedNotice = !active.connected ? `
    <div class="machine-terminal-disconnect-notice">
      <div class="machine-terminal-disconnect-copy">
        <strong>${escapeHtml("终端连接已断开")}</strong>
        <p>${escapeHtml(active.error || "虚拟机连接已断开，请点击重连恢复。")}</p>
      </div>
      <div class="action-row">
        <button class="primary-button" data-machine-terminal-action="reconnect" data-terminal-key="${active.key}">重连</button>
      </div>
    </div>
  ` : "";
  const paneOpen = Boolean(state.machine.terminalPaneOpen);
  const paneTab = String(state.machine.terminalPaneTab || "files");
  const selectedQuickCommand = (state.machine.quickCommands || []).find((item) => Number(item.id) === Number(state.machine.selectedQuickCommandID || 0)) || null;
  elements.machineTerminalDock.innerHTML = `
    <div class="machine-terminal-dock ${active.fullscreen ? "is-fullscreen" : ""}">
      <div class="machine-terminal-chrome">
        <div class="machine-terminal-shell-title">
          <strong>${escapeHtml(active.assetName)}</strong>
          <small>${escapeHtml(active.username || active.address)}</small>
        </div>
        <div class="machine-terminal-chrome-status">${active.connected ? "connected" : "closed"}</div>
      </div>
      <div class="machine-terminal-body">
        <div class="machine-terminal-main">
          <div class="machine-terminal-header">
            <div class="machine-terminal-tabs">
              ${terminals.map((terminal) => `
                <button class="machine-terminal-tab ${terminal.key === active.key ? "active" : ""}" data-machine-terminal-action="activate" data-terminal-key="${terminal.key}">
                  <span>${escapeHtml(terminal.assetName)}</span>
                  <small>${escapeHtml(terminal.username || terminal.address)}</small>
                  <i data-machine-terminal-action="close" data-terminal-key="${terminal.key}">×</i>
                </button>
              `).join("")}
            </div>
            <div class="machine-terminal-actions">
              <button class="machine-terminal-tool ${paneOpen && paneTab === "files" ? "active" : ""}" data-machine-terminal-action="toggle-pane" data-pane-tab="files" title="文件面板">文件</button>
              <button class="machine-terminal-tool ${paneOpen && paneTab === "search" ? "active" : ""}" data-machine-terminal-action="toggle-pane" data-pane-tab="search" title="搜索">搜索</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="copy-output" data-terminal-key="${active.key}" title="复制终端输出">复制</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="paste-input" data-terminal-key="${active.key}" title="从剪贴板粘贴到终端">粘贴</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="clear-screen" data-terminal-key="${active.key}" title="向终端发送清屏">清屏</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="focus" data-terminal-key="${active.key}" title="聚焦终端">聚焦</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="reconnect" data-terminal-key="${active.key}" title="重新连接">重连</button>
              <button class="machine-terminal-tool" data-machine-terminal-action="fullscreen" data-terminal-key="${active.key}" title="${active.fullscreen ? "退出全屏" : "全屏"}">${active.fullscreen ? "退出" : "全屏"}</button>
              <button class="machine-terminal-tool danger" data-machine-terminal-action="close" data-terminal-key="${active.key}" title="关闭终端">关闭</button>
            </div>
          </div>
          <div class="machine-terminal-stage">
            ${errorText}
            <div class="interactive-terminal-toolbar">
              <div class="interactive-terminal-pill">${escapeHtml(active.username || active.address)}</div>
              <div class="interactive-terminal-pill subtle">${escapeHtml(active.address)}</div>
            </div>
            <div class="interactive-terminal-statusline">
              <span>状态：${active.connected ? "已连接" : "未连接"}</span>
              <span>渲染：${active.renderer || MACHINE_TERMINAL_RENDER_MODE}</span>
              <span>输出：${active.hasServerOutput ? "已收到" : "等待中"}</span>
              <span>事件：${escapeHtml(active.lastEvent || "-")}</span>
            </div>
            ${disconnectedNotice}
            <div id="machine-terminal-screen" class="interactive-terminal-screen" tabindex="0" aria-label="machine interactive terminal">
              <div class="interactive-terminal-surface"></div>
              <div class="interactive-terminal-fallback">
                <div class="interactive-terminal-shell">
                  <div class="interactive-terminal-output">${escapeHtml(resolveTerminalDisplayOutput(active))}<span class="interactive-terminal-cursor ${active.connected ? "" : "is-hidden"}" aria-hidden="true"></span></div>
                </div>
              </div>
            </div>
          </div>
          <div class="machine-terminal-commandbar">
            <div class="machine-terminal-commandbar-top">
              <select id="machine-terminal-quick-command-select" class="toolbar-input">
                <option value="">选择已保存快捷命令</option>
                ${(state.machine.quickCommands || []).map((command) => `
                  <option value="${command.id}" ${state.machine.selectedQuickCommandID === command.id ? "selected" : ""}>${escapeHtml(command.name)} · ${escapeHtml(command.kind === "script" ? "脚本" : "命令")}</option>
                `).join("")}
              </select>
              <button class="ghost-button" data-machine-terminal-action="save-command">保存</button>
              <button class="primary-button" data-machine-terminal-action="run-command">批量执行</button>
            </div>
            <div class="machine-terminal-commandbar-meta">
              ${selectedQuickCommand ? `
                <div class="machine-terminal-commandbar-current">
                  <strong>${escapeHtml(selectedQuickCommand.name)}</strong>
                  <span>${escapeHtml(selectedQuickCommand.kind === "script" ? "脚本" : "命令")} · ${escapeHtml(selectedQuickCommand.description || "未填写说明")}</span>
                </div>
                <div class="action-row">
                  <button class="ghost-button" data-machine-terminal-action="edit-command" data-command-id="${selectedQuickCommand.id}">编辑</button>
                  <button class="ghost-button danger-soft" data-machine-terminal-action="delete-command" data-command-id="${selectedQuickCommand.id}">删除</button>
                </div>
              ` : `<span class="muted-label">支持保存脚本或命令，并批量下发到多个已打开终端。</span>`}
            </div>
            <textarea id="machine-terminal-command-editor" class="machine-terminal-command-editor" placeholder="输入一段命令或脚本。支持同时下发到多个已打开终端。">${escapeHtml(state.machine.terminalCommandDraft || "")}</textarea>
            <div class="machine-terminal-targets">
              ${(state.machine.terminals || []).map((terminal) => `
                <label class="machine-terminal-target-chip">
                  <input type="checkbox" data-machine-terminal-target="${terminal.key}" ${isMachineTerminalSelected(terminal.key) ? "checked" : ""} />
                  <span>${escapeHtml(terminal.assetName)}</span>
                </label>
              `).join("")}
            </div>
          </div>
        </div>
        ${paneOpen ? renderMachineTerminalSidePane(active, paneTab) : ""}
      </div>
    </div>
  `;
  mountMachineTerminalRenderer();
  updateMachineTerminalOutput();
  focusTerminalSoon();
}

function renderMachineTerminalSidePane(active, paneTab) {
  if (paneTab === "search") {
    const query = String(state.machine.terminalSearchQuery || "").trim();
    const results = searchMachineTerminalOutput(active, query);
    return `
      <aside class="machine-terminal-sidepane">
        <div class="machine-terminal-sidepane-head">
          <strong>搜索</strong>
          <button class="machine-terminal-pane-close" data-machine-terminal-action="toggle-pane" data-pane-tab="search">×</button>
        </div>
        <div class="machine-terminal-sidepane-body">
          <label class="machine-terminal-sidepane-search">
            <span>关键字</span>
            <input id="machine-terminal-search-input" class="toolbar-input" placeholder="搜索当前终端输出" value="${escapeHtml(query)}" />
          </label>
          <p class="muted-label">当前终端输出中命中 ${results.length} 条。</p>
          <div class="machine-terminal-search-results">
            ${results.length ? results.map((item) => `
              <article class="machine-terminal-search-hit">
                <strong>第 ${item.line} 行</strong>
                <pre>${escapeHtml(item.text)}</pre>
              </article>
            `).join("") : `<div class="muted-label">${query ? "没有匹配内容" : "输入关键字后，会在当前终端输出中搜索。"} </div>`}
          </div>
        </div>
      </aside>
    `;
  }
  const sftp = active.sftp || { path: "/", entries: [], loading: false, error: "" };
  const parentButton = sftp.path && sftp.path !== "/" ? `
    <button type="button" class="ghost-button machine-terminal-sftp-nav is-text" data-machine-terminal-action="sftp-open" data-terminal-key="${active.key}" data-sftp-path="${escapeHtml(sftp.parent || "/")}" title="返回上一级">上一级</button>
  ` : "";
  return `
    <aside class="machine-terminal-sidepane">
      <div class="machine-terminal-sidepane-head">
        <strong>SFTP</strong>
        <button class="machine-terminal-pane-close" data-machine-terminal-action="toggle-pane" data-pane-tab="files">×</button>
      </div>
      <div class="machine-terminal-sidepane-toolbar">
        <div class="machine-terminal-sidepane-breadcrumb">
          <span class="machine-terminal-sidepane-path-label">路径</span>
          <strong>${escapeHtml(sftp.path || "/")}</strong>
        </div>
        <div class="action-row machine-terminal-sftp-actions">
          ${parentButton}
          <button type="button" class="ghost-button machine-terminal-sftp-nav is-text" data-machine-terminal-action="sftp-refresh" data-terminal-key="${active.key}" title="刷新目录">刷新</button>
          <button type="button" class="ghost-button machine-terminal-sftp-nav is-primary is-short" data-machine-terminal-action="sftp-upload" data-terminal-key="${active.key}" title="上传文件">上传</button>
        </div>
      </div>
      ${sftp.error ? `<p class="muted-label machine-terminal-sidepane-note">${escapeHtml(sftp.error)}</p>` : ""}
      <div class="machine-terminal-sidepane-meta">
        <span>${escapeHtml(active.assetName)}</span>
        <span>${Array.isArray(sftp.entries) ? sftp.entries.length : 0} 项</span>
      </div>
      <div class="machine-terminal-sidepane-table">
        ${renderMachineSFTPEntries(active, sftp)}
      </div>
      <input id="machine-sftp-upload-input" class="machine-terminal-upload-input" type="file" />
    </aside>
  `;
}

export async function openAssetAccess(machineCtx, assetID) {
  const asset = state.machine.assets.find((item) => Number(item.id) === Number(assetID)) || state.machine.detail;
  if (!asset) return;
  const accounts = await getAccountsForAsset(asset.id);
  const defaultAccount = accounts.find((item) => item.is_default) || (asset.default_account_id ? accounts.find((item) => Number(item.id) === Number(asset.default_account_id)) || null : null);
  const policy = machineCtx.resolveAssetLoginPolicy(asset);
  if (policy === "managed_only" && !accounts.length) {
    await showErrorDialog({ title: "无法登录", copy: "当前资产要求使用托管账号登录，但还没有绑定任何登录凭据" });
    return;
  }
  await openTerminalTicketModal(machineCtx, asset.id, accounts, defaultAccount);
}

export async function openManagedAccountTerminal(machineCtx, asset, accountID) {
  const account = state.machine.accounts.find((item) => Number(item.id) === Number(accountID));
  if (!account) return;
  const response = await api(`/api/v1/machines/assets/${asset.id}/terminal-ticket`, {
    method: "POST",
    body: JSON.stringify({ account_id: accountID }),
  });
  openMachineTerminal(machineCtx, asset, response.data, { accountID });
  toast(`已使用托管账号 ${account.name} 打开 SSH 终端`);
}

export async function openTerminalTicketModal(machineCtx, assetID, accounts = [], defaultAccount = null) {
  const asset = state.machine.assets.find((item) => Number(item.id) === Number(assetID)) || state.machine.detail;
  if (!asset) return;
  if (String(asset.protocol || "").toLowerCase() !== "ssh") {
    await showErrorDialog({ title: "暂不支持", copy: "当前阶段只支持 SSH 资产进入在线终端" });
    return;
  }
  const accountOptions = [{ value: "", label: "手动输入账号和认证信息" }, ...accounts.map((item) => ({
    value: String(item.id),
    label: `${item.name} · ${item.username}${item.is_default ? " · 默认" : ""}`,
  }))];
  openFormModal({
    eyebrow: "SSH Terminal",
    title: `打开 SSH 终端 · ${asset.name}`,
    copy: accounts.length
      ? "先选择这台机器已经绑定的登录凭据，再决定是否手动输入。提交后会在右侧终端工作区打开连接。"
      : "当前机器还没有绑定凭据。你可以临时手动输入账号密码或 SSH 私钥，提交后会在右侧终端工作区打开连接。",
    submitText: "连接终端",
    fields: [
      { label: "已绑定凭据", name: "account_id", type: "select", value: defaultAccount ? String(defaultAccount.id) : "", options: accountOptions },
      { label: "机器地址", name: "address", value: `${asset.address}:${asset.port}` },
      { label: "登录账号", name: "username", value: defaultAccount?.username || asset.default_account_username || asset.account || "", required: true, placeholder: "root / ops-admin" },
      { label: "认证方式", name: "auth_type", type: "select", value: "password", options: [{ value: "password", label: "密码" }, { value: "ssh_key", label: "SSH 私钥" }] },
      { label: "登录密码", name: "password", type: "password", placeholder: "密码型登录填写这里" },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 6, placeholder: "私钥型登录填写完整 PEM 内容", upload: { accept: ".pem,.key,.txt,*/*", hint: "支持上传本地私钥文件后自动填入" } },
      { label: "私钥口令", name: "passphrase", type: "password", placeholder: "如果私钥带口令，在这里填写" },
    ],
    onSubmit: async (form) => {
      const selectedAccountID = Number(form.get("account_id") || 0);
      const reconnect = selectedAccountID ? { accountID: selectedAccountID } : {};
      const response = await api(`/api/v1/machines/assets/${asset.id}/terminal-ticket`, {
        method: "POST",
        body: JSON.stringify({
          account_id: selectedAccountID || undefined,
          username: form.get("username"),
          auth_type: form.get("auth_type"),
          password: form.get("password"),
          private_key: form.get("private_key"),
          passphrase: form.get("passphrase"),
        }),
      });
      openMachineTerminal(machineCtx, asset, response.data, reconnect);
      toast(response.data?.message || "终端票据已创建");
    },
  });
}

export function openMachineTerminal(machineCtx, asset, ticketPayload, reconnect = {}) {
  const key = `${asset.id}-${ticketPayload.ticket}`;
  const existing = state.machine.terminals.find((item) => item.key === key);
  if (existing) {
    state.machine.activeTerminalKey = key;
    state.machine.view = "terminal";
    machineCtx.renderMachineWorkspace();
    focusTerminalSoon();
    return;
  }
  const terminal = {
    key,
    assetID: asset.id,
    assetName: asset.name,
    ticket: ticketPayload.ticket,
    address: `${asset.address}:${asset.port}`,
    username: ticketPayload.session?.account || asset.account || "-",
    session_id: ticketPayload.session?.id,
    cols: 120,
    rows: 32,
    connected: false,
    output: "连接已建立。\n",
    rawOutput: "连接已建立。\r\n",
    hasServerOutput: false,
    renderer: MACHINE_TERMINAL_RENDER_MODE,
    lastEvent: "ticket-created",
    sftp: {
      path: "/",
      parent: "/",
      entries: [],
      loading: true,
      error: "",
    },
    reconnect,
    lastActivityAt: Date.now(),
    idleExpired: false,
    closingManually: false,
    error: "",
    fullscreen: false,
  };
  state.machine.terminals = [...(state.machine.terminals || []), terminal];
  state.machine.activeTerminalKey = key;
  state.machine.view = "terminal";
  machineCtx.renderMachineWorkspace();

  const screen = elements.machineTerminalDock?.querySelector("#machine-terminal-screen");
  const { cols, rows } = computeMachineTerminalSize(screen);
  patchTerminal(key, { cols, rows });

  const socket = new WebSocket(buildWebSocketURL(`/api/v1/machines/assets/${asset.id}/terminal`, {
    ticket: ticketPayload.ticket,
    cols,
    rows,
  }), ["bc-terminal", state.token]);
  terminalSockets.set(key, socket);

  socket.addEventListener("open", () => {
    patchTerminal(key, { connected: true, error: "", lastEvent: "ws-open" });
    registerMachineTerminalActivity(key);
    startMachineTerminalHeartbeat(key);
    machineCtx.renderMachineWorkspace();
    loadMachineTerminalSFTP(key);
    focusTerminalSoon();
  });

  socket.addEventListener("message", (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === "output") {
        patchTerminal(key, { lastEvent: "output" });
        registerMachineTerminalActivity(key);
        appendTerminalOutput(key, message.data || "");
        return;
      }
      if (message.type === "pong") {
        patchTerminal(key, { lastEvent: "pong" });
        return;
      }
      if (message.type === "status") {
        if (message.status === "connected") {
          patchTerminal(key, {
            address: `${message.address}:${message.port}`,
            username: message.username || getTerminalByKey(key)?.username || "-",
            connected: true,
            lastEvent: "status-connected",
          });
          registerMachineTerminalActivity(key);
          machineCtx.renderMachineWorkspace();
        } else if (message.status === "closed") {
          const reason = String(message.reason || "终端连接已断开，可点击重连恢复。");
          patchTerminal(key, { connected: false, error: reason, lastEvent: "status-closed" });
          if (state.machine.activeTerminalKey === key) {
            toast(reason);
          }
          machineCtx.renderMachineWorkspace();
          machineCtx.refreshMachineData();
        }
        return;
      }
      if (message.type === "error") {
        const reason = String(message.message || "终端连接失败");
        patchTerminal(key, { connected: false, error: reason, lastEvent: "status-error" });
        if (state.machine.activeTerminalKey === key) {
          toast(reason);
        }
        machineCtx.renderMachineWorkspace();
      }
    } catch (_error) {
      patchTerminal(key, { lastEvent: "raw-message" });
      appendTerminalOutput(key, event.data || "");
    }
  });

  socket.addEventListener("close", () => {
    terminalSockets.delete(key);
    stopMachineTerminalHeartbeat(key);
    const current = getTerminalByKey(key);
    if (current?.closingManually) {
      patchTerminal(key, { connected: false, lastEvent: "ws-close" });
    } else {
      const reason = current?.error || "终端连接已断开，可点击重连恢复。";
      patchTerminal(key, { connected: false, error: reason, lastEvent: "ws-close" });
      if (state.machine.activeTerminalKey === key) {
        toast(reason);
      }
    }
    machineCtx.renderMachineWorkspace();
    machineCtx.refreshMachineData();
  });

  socket.addEventListener("error", () => {
    stopMachineTerminalHeartbeat(key);
    patchTerminal(key, { connected: false, error: "SSH 终端连接失败", lastEvent: "ws-error" });
    machineCtx.renderMachineWorkspace();
  });

}

export function bindMachineTerminalInteractions(machineCtx) {
  machineTerminalCtx = machineCtx;
  const root = elements.machineTerminalDock;
  if (!root) return;
  if (!machineTerminalDelegatedBound) {
    root.addEventListener("click", (event) => {
      const actionButton = event.target.closest("[data-machine-terminal-action]");
      if (!actionButton || !root.contains(actionButton)) return;
      const action = String(actionButton.dataset.machineTerminalAction || "");
      if (!action.startsWith("sftp-")) return;
      const key = String(actionButton.dataset.terminalKey || state.machine.activeTerminalKey || "");
      if (action === "sftp-refresh") {
        event.preventDefault();
        loadMachineTerminalSFTP(key);
        return;
      }
      if (action === "sftp-open") {
        event.preventDefault();
        const targetPath = String(actionButton.dataset.sftpPath || "/");
        loadMachineTerminalSFTP(key, targetPath);
        return;
      }
      if (action === "sftp-download") {
        event.preventDefault();
        event.stopPropagation();
        const targetPath = String(actionButton.dataset.sftpPath || "/");
        downloadMachineTerminalSFTP(key, targetPath);
        return;
      }
      if (action === "sftp-upload") {
        event.preventDefault();
        const input = root.querySelector("#machine-sftp-upload-input");
        if (!input) return;
        input.value = "";
        input.click();
      }
    });
    machineTerminalDelegatedBound = true;
  }
  root.querySelectorAll("[data-machine-terminal-action='activate']").forEach((button) => {
    button.addEventListener("click", () => {
      state.machine.activeTerminalKey = String(button.dataset.terminalKey || "");
      machineCtx.renderMachineWorkspace();
      focusTerminalSoon();
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='close']").forEach((button) => {
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      closeMachineTerminal(machineCtx, String(button.dataset.terminalKey || ""));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='copy-output']").forEach((button) => {
    button.addEventListener("click", async () => {
      await copyMachineTerminalOutput(String(button.dataset.terminalKey || state.machine.activeTerminalKey || ""));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='paste-input']").forEach((button) => {
    button.addEventListener("click", async () => {
      await pasteMachineTerminalInput(String(button.dataset.terminalKey || state.machine.activeTerminalKey || ""));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='clear-screen']").forEach((button) => {
    button.addEventListener("click", () => {
      clearMachineTerminalScreen(String(button.dataset.terminalKey || state.machine.activeTerminalKey || ""));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='focus']").forEach((button) => {
    button.addEventListener("click", () => {
      registerMachineTerminalActivity(String(button.dataset.terminalKey || state.machine.activeTerminalKey || ""));
      root.querySelector("#machine-terminal-screen")?.focus();
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='reconnect']").forEach((button) => {
    button.addEventListener("click", async () => {
      await reconnectMachineTerminal(machineCtx, String(button.dataset.terminalKey || state.machine.activeTerminalKey || ""));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='fullscreen']").forEach((button) => {
    button.addEventListener("click", async () => {
      const key = String(button.dataset.terminalKey || state.machine.activeTerminalKey || "");
      const terminal = getTerminalByKey(key);
      if (!terminal) return;
      const next = !terminal.fullscreen;
      patchTerminal(key, { fullscreen: next });
      renderMachineTerminalDock();
      bindMachineTerminalInteractions(machineCtx);
      focusTerminalSoon();
      if (next) {
        await requestMachineTerminalFullscreen();
      } else {
        await exitMachineTerminalFullscreen();
      }
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='toggle-pane']").forEach((button) => {
    button.addEventListener("click", () => {
      const paneTab = String(button.dataset.paneTab || "files");
      const shouldClose = state.machine.terminalPaneOpen && state.machine.terminalPaneTab === paneTab;
      state.machine.terminalPaneOpen = !shouldClose;
      state.machine.terminalPaneTab = paneTab;
      machineCtx.renderMachineWorkspace();
      if (!shouldClose && paneTab === "files") {
        loadMachineTerminalSFTP(state.machine.activeTerminalKey);
      }
      focusTerminalSoon();
    });
  });
  const uploadInput = root.querySelector("#machine-sftp-upload-input");
  if (uploadInput) {
    uploadInput.addEventListener("change", async (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      await uploadMachineTerminalSFTP(state.machine.activeTerminalKey, file);
      event.target.value = "";
    });
  }
  root.querySelectorAll("[data-machine-terminal-action='save-command']").forEach((button) => {
    button.addEventListener("click", () => {
      openMachineQuickCommandSaveModal(machineCtx);
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='edit-command']").forEach((button) => {
    button.addEventListener("click", () => {
      openMachineQuickCommandEditModal(machineCtx, Number(button.dataset.commandId || 0));
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='delete-command']").forEach((button) => {
    button.addEventListener("click", async () => {
      const commandID = Number(button.dataset.commandId || 0);
      if (!commandID) return;
      await deleteMachineQuickCommand(machineCtx, commandID);
    });
  });
  root.querySelectorAll("[data-machine-terminal-action='run-command']").forEach((button) => {
    button.addEventListener("click", async () => {
      await runMachineTerminalBatchCommand();
    });
  });
  root.querySelectorAll("[data-machine-terminal-target]").forEach((input) => {
    input.addEventListener("change", () => {
      const key = String(input.dataset.machineTerminalTarget || "");
      const selected = new Set(state.machine.terminalSelectedTargets || []);
      if (input.checked) {
        selected.add(key);
      } else {
        selected.delete(key);
      }
      state.machine.terminalSelectedTargets = Array.from(selected);
    });
  });
  const commandSelect = root.querySelector("#machine-terminal-quick-command-select");
  if (commandSelect) {
    commandSelect.addEventListener("change", () => {
      const commandID = Number(commandSelect.value || 0);
      state.machine.selectedQuickCommandID = commandID || null;
      const command = (state.machine.quickCommands || []).find((item) => Number(item.id) === commandID);
      if (command) {
        state.machine.terminalCommandDraft = String(command.content || "");
        rerenderMachineTerminalDock();
        focusTerminalSoon();
      }
    });
  }
  const commandEditor = root.querySelector("#machine-terminal-command-editor");
  if (commandEditor) {
    commandEditor.addEventListener("input", () => {
      state.machine.terminalCommandDraft = commandEditor.value;
    });
  }
  const searchInput = root.querySelector("#machine-terminal-search-input");
  if (searchInput) {
    searchInput.addEventListener("input", () => {
      state.machine.terminalSearchQuery = searchInput.value;
      rerenderMachineTerminalDock();
      window.requestAnimationFrame(() => {
        const nextInput = elements.machineTerminalDock?.querySelector("#machine-terminal-search-input");
        if (nextInput) {
          nextInput.focus();
          nextInput.setSelectionRange(state.machine.terminalSearchQuery.length, state.machine.terminalSearchQuery.length);
        }
      });
    });
  }
  const screen = root.querySelector("#machine-terminal-screen");
  if (!screen) return;
  screen.addEventListener("mousedown", () => focusTerminalSoon());
  screen.addEventListener("keydown", (event) => {
    const active = getActiveTerminal();
    if (!active?.connected) return;
    const data = translateMachineTerminalKeydown(event);
    if (!data) return;
    event.preventDefault();
    event.stopPropagation();
    patchTerminal(active.key, { lastEvent: `raw:${String(event.key || "").toLowerCase() || "key"}` });
    if (state.machine.activeTerminalKey === active.key) {
      updateMachineTerminalOutput();
    }
    sendMachineTerminalInput(data);
  }, true);
  mountMachineTerminalRenderer();
  observeMachineTerminalResize();
  observeMachineTerminalVisibility(machineCtx);
}

export function closeMachineTerminal(machineCtx, terminalKey = state.machine.activeTerminalKey) {
  const key = String(terminalKey || "");
  if (!key) return;
  const terminal = getTerminalByKey(key);
  if (terminal) {
    patchTerminal(key, { closingManually: true });
  }
  const socket = terminalSockets.get(key);
  if (socket) {
    terminalSockets.delete(key);
    stopMachineTerminalHeartbeat(key);
    socket.close();
  }
  if (terminal?.fullscreen) {
    exitMachineTerminalFullscreen();
  }
  state.machine.terminals = (state.machine.terminals || []).filter((item) => item.key !== key);
  if (state.machine.activeTerminalKey === key) {
    state.machine.activeTerminalKey = state.machine.terminals[0]?.key || null;
  }
  if (!state.machine.terminals.length || machineXtermKey === key) {
    destroyMachineTerminalRenderer();
  }
  if (!state.machine.terminals.length && state.machine.view === "terminal") {
    state.machine.view = "access";
  }
  machineCtx.renderMachineWorkspace();
}

async function getAccountsForAsset(assetID) {
  if (state.machine.selectedAssetID === assetID && Array.isArray(state.machine.accounts) && state.machine.accounts.length) {
    return state.machine.accounts;
  }
  const response = await api(`/api/v1/machines/assets/${assetID}/accounts`);
  return response.data || [];
}

function ensureMachineTerminalSelection(activeKey) {
  const terminals = state.machine.terminals || [];
  const existing = new Set((state.machine.terminalSelectedTargets || []).filter((key) => terminals.some((item) => item.key === key)));
  if (activeKey && !existing.size) {
    existing.add(activeKey);
  }
  state.machine.terminalSelectedTargets = Array.from(existing);
}

function isMachineTerminalSelected(key) {
  return (state.machine.terminalSelectedTargets || []).includes(key);
}

function getSelectedMachineTerminalKeys() {
  const terminals = state.machine.terminals || [];
  const selected = (state.machine.terminalSelectedTargets || []).filter((key) => terminals.some((item) => item.key === key));
  if (selected.length) return selected;
  return state.machine.activeTerminalKey ? [state.machine.activeTerminalKey] : [];
}

function searchMachineTerminalOutput(active, query) {
  if (!active) return [];
  const lines = String(active.output || "").split("\n");
  const keyword = String(query || "").trim().toLowerCase();
  if (!keyword) return [];
  const matches = [];
  lines.forEach((line, index) => {
    if (String(line || "").toLowerCase().includes(keyword)) {
      matches.push({ line: index + 1, text: line });
    }
  });
  return matches.slice(0, 50);
}

function getActiveTerminal() {
  const terminals = state.machine.terminals || [];
  return terminals.find((item) => item.key === state.machine.activeTerminalKey) || terminals[0] || null;
}

function getTerminalByKey(key) {
  return (state.machine.terminals || []).find((item) => item.key === key) || null;
}

function patchTerminal(key, patch) {
  state.machine.terminals = (state.machine.terminals || []).map((item) => (item.key === key ? { ...item, ...patch } : item));
}

function startMachineTerminalHeartbeat(key) {
  stopMachineTerminalHeartbeat(key);
  const timer = window.setInterval(() => {
    const socket = terminalSockets.get(key);
    const terminal = getTerminalByKey(key);
    if (!socket || socket.readyState !== WebSocket.OPEN || !terminal?.connected) {
      stopMachineTerminalHeartbeat(key);
      return;
    }
    try {
      socket.send(JSON.stringify({ type: "ping" }));
    } catch (_error) {
      stopMachineTerminalHeartbeat(key);
    }
  }, MACHINE_TERMINAL_HEARTBEAT_MS);
  terminalHeartbeats.set(key, timer);
}

function stopMachineTerminalHeartbeat(key) {
  const timer = terminalHeartbeats.get(key);
  if (timer) {
    window.clearInterval(timer);
    terminalHeartbeats.delete(key);
  }
}

function appendTerminalOutput(key, data) {
  const terminal = getTerminalByKey(key);
  if (!terminal) return;
  patchTerminal(key, {
    rawOutput: `${terminal.rawOutput || ""}${data || ""}`,
    output: applyTerminalChunk(terminal.output || "", data),
    hasServerOutput: true,
    renderer: MACHINE_TERMINAL_RENDER_MODE,
  });
  if (state.machine.activeTerminalKey === key) {
    if (machineXterm && machineXtermKey === key) {
      machineXterm.write(String(data || ""));
      return;
    }
    updateMachineTerminalOutput();
  }
}

function updateMachineTerminalOutput() {
  const screen = elements.machineTerminalDock?.querySelector("#machine-terminal-screen");
  const surface = elements.machineTerminalDock?.querySelector(".interactive-terminal-surface");
  const output = elements.machineTerminalDock?.querySelector(".interactive-terminal-output");
  const cursor = elements.machineTerminalDock?.querySelector(".interactive-terminal-cursor");
  const statusline = elements.machineTerminalDock?.querySelector(".interactive-terminal-statusline");
  const active = getActiveTerminal();
  if (!screen || !active) return;
  const rendererMounted = MACHINE_TERMINAL_RENDER_MODE === "xterm" && mountMachineTerminalRenderer();
  if (surface) {
    surface.classList.toggle("hidden", !rendererMounted);
  }
  const fallback = elements.machineTerminalDock?.querySelector(".interactive-terminal-fallback");
  if (fallback) fallback.classList.remove("hidden");
  if (rendererMounted) {
    resizeMachineTerminal();
  }
  if (output) {
    output.innerHTML = `${escapeHtml(resolveTerminalDisplayOutput(active))}<span class="interactive-terminal-cursor ${active.connected ? "" : "is-hidden"}" aria-hidden="true"></span>`;
  }
  if (statusline) {
    statusline.innerHTML = `
      <span>状态：${active.connected ? "已连接" : "未连接"}</span>
      <span>渲染：${escapeHtml(active.renderer || MACHINE_TERMINAL_RENDER_MODE)}</span>
      <span>输出：${active.hasServerOutput ? "已收到" : "等待中"}</span>
      <span>事件：${escapeHtml(active.lastEvent || "-")}</span>
    `;
  }
  if (cursor) cursor.classList.toggle("is-hidden", !active.connected);
  screen.scrollTop = screen.scrollHeight;
}

function sendMachineTerminalInput(data) {
  const active = getActiveTerminal();
  if (!active) return;
  const socket = terminalSockets.get(active.key);
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  registerMachineTerminalActivity(active.key);
  patchTerminal(active.key, { lastEvent: `send:${describeMachineInput(data)}` });
  if (state.machine.activeTerminalKey === active.key) {
    updateMachineTerminalOutput();
  }
  socket.send(JSON.stringify({ type: "input", data }));
}

function translateMachineTerminalKeydown(event) {
  if (!event) return "";
  const key = String(event.key || "");
  if (!key || key === "Unidentified" || event.isComposing || event.metaKey) return "";
  if (event.ctrlKey && key.length === 1) {
    const lower = key.toLowerCase();
    if (lower >= "a" && lower <= "z") {
      return String.fromCharCode(lower.charCodeAt(0) - 96);
    }
  }
  if (event.altKey && key.length === 1) {
    return `\u001b${key}`;
  }
  const specialMap = {
    Enter: "\r",
    Backspace: "\u007f",
    Tab: "\t",
    Delete: "\u001b[3~",
    Escape: "\u001b",
    ArrowUp: "\u001b[A",
    ArrowDown: "\u001b[B",
    ArrowRight: "\u001b[C",
    ArrowLeft: "\u001b[D",
    Home: "\u001b[H",
    End: "\u001b[F",
  };
  if (specialMap[key]) return specialMap[key];
  if (key.length === 1 && !event.ctrlKey && !event.altKey) return key;
  return "";
}

function resizeMachineTerminal() {
  const active = getActiveTerminal();
  if (!active) return;
  const screen = elements.machineTerminalDock?.querySelector("#machine-terminal-screen");
  if (!screen) return;
  const { cols, rows } = computeMachineTerminalSize(screen);
  if (active.cols === cols && active.rows === rows) return;
  patchTerminal(active.key, { cols, rows });
  if (machineXterm && machineXtermKey === active.key) {
    try {
      machineXterm.resize(cols, rows);
    } catch (_error) {}
  }
  const socket = terminalSockets.get(active.key);
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({ type: "resize", cols, rows }));
}

function registerMachineTerminalActivity(key) {
  if (!key) return;
  patchTerminal(String(key), { lastActivityAt: Date.now(), idleExpired: false });
}

function observeMachineTerminalResize() {
  if (machineTerminalResizeBound) return;
  machineTerminalResizeBound = true;
  window.addEventListener("resize", scheduleMachineTerminalResize);
  if (!machineTerminalFullscreenBound) {
    document.addEventListener("fullscreenchange", syncMachineTerminalFullscreenState);
    machineTerminalFullscreenBound = true;
  }
}

function observeMachineTerminalVisibility(machineCtx) {
  if (machineTerminalVisibilityBound) return;
  machineTerminalVisibilityBound = true;
  const restore = () => {
    if (state.machine.view !== "terminal" || !state.machine.activeTerminalKey) return;
    machineCtx.renderMachineWorkspace();
    focusTerminalSoon();
  };
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) {
      window.setTimeout(restore, 80);
    }
  });
  window.addEventListener("focus", () => {
    window.setTimeout(restore, 80);
  });
}

function scheduleMachineTerminalResize() {
  if (machineTerminalResizeTimer) {
    window.clearTimeout(machineTerminalResizeTimer);
  }
  machineTerminalResizeTimer = window.setTimeout(() => {
    resizeMachineTerminal();
  }, 120);
}

function computeMachineTerminalSize(screen) {
  const rect = screen?.getBoundingClientRect?.() || { width: 980, height: 360 };
  const cols = Math.max(80, Math.floor((rect.width - 28) / 8.5));
  const rows = Math.max(20, Math.floor((rect.height - 24) / 18));
  return { cols, rows };
}

function focusTerminalSoon() {
  window.requestAnimationFrame(() => {
    if (machineXterm) {
      machineXterm.focus();
      const active = getActiveTerminal();
      if (active) {
        patchTerminal(active.key, { lastEvent: "focus-xterm" });
        if (state.machine.activeTerminalKey === active.key) {
          updateMachineTerminalOutput();
        }
      }
      return;
    }
    elements.machineTerminalDock?.querySelector("#machine-terminal-screen")?.focus();
  });
}

function describeMachineInput(data) {
  const normalized = String(data || "");
  if (!normalized) return "empty";
  if (normalized === "\r") return "enter";
  if (normalized === "\u007f") return "backspace";
  if (normalized === "\u001b[3~") return "delete";
  if (normalized.length === 1) return normalized;
  return "seq";
}


async function requestMachineTerminalFullscreen() {
  const dock = elements.machineTerminalDock?.querySelector(".machine-terminal-dock");
  if (!dock || document.fullscreenElement === dock || typeof dock.requestFullscreen !== "function") return;
  try {
    await dock.requestFullscreen();
  } catch (_error) {}
}

async function exitMachineTerminalFullscreen() {
  if (!document.fullscreenElement) return;
  try {
    await document.exitFullscreen();
  } catch (_error) {}
}

function syncMachineTerminalFullscreenState() {
  const active = getActiveTerminal();
  const dock = elements.machineTerminalDock?.querySelector(".machine-terminal-dock");
  if (!active || !dock) return;
  const isFullscreen = document.fullscreenElement === dock;
  if (Boolean(active.fullscreen) === isFullscreen) return;
  patchTerminal(active.key, { fullscreen: isFullscreen });
  updateMachineTerminalOutput();
}

function getBrowserTerminalCtor() {
  if (MACHINE_TERMINAL_RENDER_MODE !== "xterm") return null;
  return globalThis.Terminal || globalThis.XTerm?.Terminal || null;
}

function resolveTerminalDisplayOutput(active) {
  if (!active) return "";
  const content = String(active.output || "");
  if (content) return content;
  if (active.connected) return "";
  return "终端未连接。";
}

function renderMachineSFTPEntries(active, sftp) {
  if (sftp.loading) {
    return `<div class="machine-terminal-sidepane-row is-placeholder"><span>正在加载目录...</span><small>请稍候</small></div>`;
  }
  if (!Array.isArray(sftp.entries) || !sftp.entries.length) {
    return `<div class="machine-terminal-sidepane-row is-placeholder"><span>当前目录为空</span><small>没有可显示的文件</small></div>`;
  }
  return sftp.entries.map((entry) => {
    if (entry.type === "dir") {
      return `
        <button type="button" class="machine-terminal-sidepane-row machine-terminal-sidepane-entry is-dir" data-machine-terminal-action="sftp-open" data-terminal-key="${active.key}" data-sftp-path="${escapeHtml(entry.path)}">
          <span class="machine-terminal-sidepane-name">
            <i>DIR</i>
            <strong>${escapeHtml(entry.name)}</strong>
            <small>${escapeHtml(formatDateTime(entry.mod_time) || "-")}</small>
            <small>${escapeHtml(`${entry.owner || "-"}:${entry.group || "-"} · ${entry.mode || "-"}`)}</small>
          </span>
          <span class="machine-terminal-sidepane-detail">
            <em>目录</em>
          </span>
        </button>
      `;
    }
    return `
      <div class="machine-terminal-sidepane-row machine-terminal-sidepane-file">
        <span class="machine-terminal-sidepane-name">
          <i>FILE</i>
          <strong>${escapeHtml(entry.name)}</strong>
          <small>${escapeHtml(formatDateTime(entry.mod_time) || "-")}</small>
          <small>${escapeHtml(`${entry.owner || "-"}:${entry.group || "-"} · ${entry.mode || "-"}`)}</small>
        </span>
        <span class="machine-terminal-sidepane-detail">
          <em>${escapeHtml(formatSFTPSize(entry.size))}</em>
          <button type="button" class="ghost-button machine-terminal-sftp-nav is-text is-download" data-machine-terminal-action="sftp-download" data-terminal-key="${active.key}" data-sftp-path="${escapeHtml(entry.path)}" title="下载文件">下载</button>
        </span>
      </div>
    `;
  }).join("");
}

async function loadMachineTerminalSFTP(terminalKey, targetPath = "") {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal?.assetID || !terminal?.session_id) return;
  const currentSFTP = terminal.sftp || {};
  patchTerminal(key, {
    sftp: {
      ...currentSFTP,
      loading: true,
      error: "",
      path: targetPath || currentSFTP.path || "/",
    },
  });
  if (state.machine.activeTerminalKey === key && state.machine.terminalPaneOpen && state.machine.terminalPaneTab === "files") {
    rerenderMachineTerminalDock();
  }
  try {
    const query = new URLSearchParams({
      session_id: String(terminal.session_id),
      path: targetPath || currentSFTP.path || "/",
    });
    const response = await api(`/api/v1/machines/assets/${terminal.assetID}/sftp?${query.toString()}`);
    patchTerminal(key, {
      sftp: {
        path: response.data?.path || "/",
        parent: response.data?.parent || "/",
        entries: response.data?.entries || [],
        loading: false,
        error: "",
      },
    });
  } catch (error) {
    patchTerminal(key, {
      sftp: {
        ...currentSFTP,
        loading: false,
        error: error?.message || "SFTP 目录加载失败",
      },
    });
  }
  if (state.machine.activeTerminalKey === key && state.machine.terminalPaneOpen && state.machine.terminalPaneTab === "files") {
    rerenderMachineTerminalDock();
    focusTerminalSoon();
  }
}

function showMachineTerminalSFTPError(terminalKey, message) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  if (!key) return;
  const terminal = getTerminalByKey(key);
  if (!terminal) return;
  patchTerminal(key, {
    sftp: {
      ...(terminal.sftp || {}),
      error: message || "",
      loading: false,
    },
  });
  if (state.machine.activeTerminalKey === key && state.machine.terminalPaneOpen && state.machine.terminalPaneTab === "files") {
    rerenderMachineTerminalDock();
  }
}

async function downloadMachineTerminalSFTP(terminalKey, targetPath) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal?.assetID || !terminal?.session_id) {
    const message = "当前会话不可用，无法下载文件";
    showMachineTerminalSFTPError(key, message);
    await showErrorDialog({ title: "下载失败", copy: message });
    return;
  }
  try {
    showMachineTerminalSFTPError(key, "");
    const query = new URLSearchParams({
      session_id: String(terminal.session_id),
      path: targetPath || "/",
    });
    const headers = {};
    if (state.token) headers.Authorization = `Bearer ${state.token}`;
    const response = await fetch(buildAPIURL(`/api/v1/machines/assets/${terminal.assetID}/sftp/download?${query.toString()}`), { headers });
    if (!response.ok) {
      let message = "文件下载失败";
      try {
        const payload = await response.json();
        message = payload.error || message;
      } catch (_error) {}
      showMachineTerminalSFTPError(key, message);
      await showErrorDialog({ title: "下载失败", copy: message });
      return;
    }
    const blob = await response.blob();
    const disposition = response.headers.get("Content-Disposition") || "";
    const match = disposition.match(/filename="([^"]+)"/);
    const filename = match?.[1] || targetPath.split("/").pop() || "download.bin";
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = filename;
    anchor.style.display = "none";
    document.body.appendChild(anchor);
    anchor.click();
    window.setTimeout(() => {
      anchor.remove();
      URL.revokeObjectURL(url);
    }, 1500);
  } catch (error) {
    const message = error?.message || "文件下载失败";
    showMachineTerminalSFTPError(key, message);
    await showErrorDialog({ title: "下载失败", copy: message });
  }
}

async function uploadMachineTerminalSFTP(terminalKey, file) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal?.assetID || !terminal?.session_id || !file) return;
  const form = new FormData();
  form.set("session_id", String(terminal.session_id));
  form.set("path", terminal.sftp?.path || "/");
  form.set("file", file);
  const headers = {};
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const response = await fetch(buildAPIURL(`/api/v1/machines/assets/${terminal.assetID}/sftp/upload`), {
    method: "POST",
    headers,
    body: form,
  });
  if (!response.ok) {
    let message = "文件上传失败";
    try {
      const payload = await response.json();
      message = payload.error || message;
    } catch (_error) {}
    showMachineTerminalSFTPError(key, message);
    await showErrorDialog({ title: "上传失败", copy: message });
    return;
  }
  toast("文件已上传");
  await loadMachineTerminalSFTP(key, terminal.sftp?.path || "/");
}

async function copyMachineTerminalOutput(terminalKey) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal) return;
  const content = String(terminal.output || "");
  if (!content) {
    await showErrorDialog({ title: "无法复制", copy: "当前终端没有可复制的输出" });
    return;
  }
  try {
    await navigator.clipboard.writeText(content);
    toast("终端输出已复制");
  } catch (_error) {
    await showErrorDialog({ title: "复制失败", copy: "复制失败，请检查浏览器剪贴板权限" });
  }
}

async function pasteMachineTerminalInput(terminalKey) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal?.connected) {
    await showErrorDialog({ title: "无法粘贴", copy: "当前终端未连接，无法粘贴" });
    return;
  }
  try {
    const content = await navigator.clipboard.readText();
    if (!content) {
      await showErrorDialog({ title: "无法粘贴", copy: "剪贴板没有可粘贴的内容" });
      return;
    }
    sendMachineTerminalInput(content);
    focusTerminalSoon();
  } catch (_error) {
    await showErrorDialog({ title: "粘贴失败", copy: "粘贴失败，请检查浏览器剪贴板权限" });
  }
}

function normalizeBatchCommandInput(value) {
  const raw = String(value || "");
  if (!raw.trim()) return "";
  let normalized = raw.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  if (!normalized.endsWith("\n")) {
    normalized += "\n";
  }
  return normalized.replace(/\n/g, "\r");
}

async function runMachineTerminalBatchCommand() {
  const payload = normalizeBatchCommandInput(state.machine.terminalCommandDraft || "");
  if (!payload) {
    toast("先输入一段命令或脚本");
    return;
  }
  const keys = getSelectedMachineTerminalKeys();
  if (!keys.length) {
    toast("先选择至少一个终端");
    return;
  }
  let success = 0;
  for (const key of keys) {
    const terminal = getTerminalByKey(key);
    const socket = terminalSockets.get(key);
    if (!terminal?.connected || !socket || socket.readyState !== WebSocket.OPEN) {
      continue;
    }
    registerMachineTerminalActivity(key);
    socket.send(JSON.stringify({ type: "input", data: payload }));
    success += 1;
  }
  if (!success) {
    await showErrorDialog({ title: "无法执行", copy: "当前没有可执行的在线终端" });
    return;
  }
  toast(`已向 ${success} 个终端下发命令`);
}

function openMachineQuickCommandSaveModal(machineCtx) {
  const draft = String(state.machine.terminalCommandDraft || "").trim();
  if (!draft) {
    toast("先输入一段命令或脚本，再保存为快捷命令");
    return;
  }
  openFormModal({
    eyebrow: "Quick Command",
    title: "保存快捷命令",
    copy: "快捷命令会保存在后端数据库里，后续可以在终端页直接选择并批量执行。",
    submitText: "保存命令",
    fields: [
      { label: "命令名称", name: "name", required: true, placeholder: "例如：查看磁盘占用" },
      {
        label: "命令类型",
        name: "kind",
        type: "select",
        value: "command",
        options: [
          { value: "command", label: "命令" },
          { value: "script", label: "脚本" },
        ],
      },
      { label: "说明", name: "description", placeholder: "说明这个快捷命令的用途" },
      { label: "命令内容", name: "content", type: "textarea", rows: 8, value: draft, required: true },
    ],
    onSubmit: async (form) => {
      await api("/api/v1/machines/quick-commands", {
        method: "POST",
        body: JSON.stringify({
          name: form.get("name"),
          kind: form.get("kind"),
          description: form.get("description"),
          content: form.get("content"),
        }),
      });
      toast("快捷命令已保存");
      await machineCtx.refreshMachineData();
      machineCtx.renderMachineWorkspace();
    },
  });
}

function openMachineQuickCommandEditModal(machineCtx, commandID) {
  const command = (state.machine.quickCommands || []).find((item) => Number(item.id) === Number(commandID));
  if (!command) return;
  openFormModal({
    eyebrow: "Quick Command",
    title: `编辑快捷命令 · ${command.name}`,
    copy: "修改后会立即保存到后端数据库，并同步更新终端页下拉选择。",
    submitText: "保存修改",
    fields: [
      { label: "命令名称", name: "name", required: true, value: command.name },
      {
        label: "命令类型",
        name: "kind",
        type: "select",
        value: command.kind,
        options: [
          { value: "command", label: "命令" },
          { value: "script", label: "脚本" },
        ],
      },
      { label: "说明", name: "description", value: command.description || "" },
      { label: "命令内容", name: "content", type: "textarea", rows: 8, value: command.content, required: true },
    ],
    onSubmit: async (form) => {
      const response = await api(`/api/v1/machines/quick-commands/${command.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: form.get("name"),
          kind: form.get("kind"),
          description: form.get("description"),
          content: form.get("content"),
        }),
      });
      state.machine.terminalCommandDraft = String(response.data?.content || "");
      toast("快捷命令已更新");
      await machineCtx.refreshMachineData();
      machineCtx.renderMachineWorkspace();
    },
  });
}

async function deleteMachineQuickCommand(machineCtx, commandID) {
  const command = (state.machine.quickCommands || []).find((item) => Number(item.id) === Number(commandID));
  if (!command) return;
  if (!window.confirm(`确认删除快捷命令「${command.name}」吗？`)) return;
  await api(`/api/v1/machines/quick-commands/${command.id}`, { method: "DELETE" });
  if (Number(state.machine.selectedQuickCommandID || 0) === Number(command.id)) {
    state.machine.selectedQuickCommandID = null;
  }
  toast("快捷命令已删除");
  await machineCtx.refreshMachineData();
  machineCtx.renderMachineWorkspace();
}

function clearMachineTerminalScreen(terminalKey) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal?.connected) {
    showErrorDialog({ title: "无法清屏", copy: "当前终端未连接，无法清屏" });
    return;
  }
  patchTerminal(key, {
    output: "",
    rawOutput: "",
    hasServerOutput: false,
    lastEvent: "clear-screen",
  });
  if (machineXterm && machineXtermKey === key) {
    try {
      machineXterm.clear();
    } catch (_error) {}
  }
  const active = getActiveTerminal();
  if (active?.key === key) {
    updateMachineTerminalOutput();
  }
  sendMachineTerminalInput("clear\r");
  focusTerminalSoon();
}

function formatSFTPSize(size) {
  const value = Number(size || 0);
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function rerenderMachineTerminalDock() {
  renderMachineTerminalDock();
  if (machineTerminalCtx) {
    bindMachineTerminalInteractions(machineTerminalCtx);
  }
}

async function reconnectMachineTerminal(machineCtx, terminalKey) {
  const key = String(terminalKey || state.machine.activeTerminalKey || "");
  const terminal = getTerminalByKey(key);
  if (!terminal) return;
  const asset = state.machine.assets.find((item) => Number(item.id) === Number(terminal.assetID)) || state.machine.detail;
  if (!asset) {
    await showErrorDialog({ title: "无法重连", copy: "未找到对应资产，无法重连" });
    return;
  }
  const reconnect = terminal.reconnect || {};
  try {
    let payload;
    if (reconnect.accountID) {
      payload = {
        account_id: Number(reconnect.accountID),
      };
    } else if (reconnect.username) {
      payload = {
        username: reconnect.username,
        auth_type: reconnect.authType || "password",
        password: reconnect.password || "",
        private_key: reconnect.privateKey || "",
        passphrase: reconnect.passphrase || "",
      };
    } else {
      await openAssetAccess(machineCtx, asset.id);
      return;
    }
    const response = await api(`/api/v1/machines/assets/${asset.id}/terminal-ticket`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    state.machine.view = "terminal";
    openMachineTerminal(machineCtx, asset, response.data, reconnect);
    closeMachineTerminal(machineCtx, key);
    state.machine.view = "terminal";
    machineCtx.renderMachineWorkspace();
    toast("终端已重新连接");
  } catch (error) {
    await showErrorDialog({ title: "重连失败", copy: error?.message || "终端重连失败" });
  }
}

function destroyMachineTerminalRenderer() {
  if (machineXterm) {
    try {
      machineXterm.dispose();
    } catch (_error) {}
  }
  machineXterm = null;
  machineXtermKey = null;
}

function mountMachineTerminalRenderer() {
  const active = getActiveTerminal();
  const screen = elements.machineTerminalDock?.querySelector("#machine-terminal-screen");
  const surface = screen?.querySelector(".interactive-terminal-surface");
  const fallback = screen?.querySelector(".interactive-terminal-fallback");
  const TerminalCtor = getBrowserTerminalCtor();
  if (!active || !screen || !surface || !TerminalCtor) return false;
  if (machineXterm && machineXtermKey === active.key) return true;
  destroyMachineTerminalRenderer();
  surface.innerHTML = "";
  machineXterm = new TerminalCtor({
    cursorBlink: true,
    convertEol: true,
    fontFamily: "IBM Plex Mono, SFMono-Regular, Consolas, monospace",
    fontSize: 14,
    lineHeight: 1.45,
    theme: {
      background: "#242938",
      foreground: "#e6ebf5",
      cursor: "#7c89ff",
      cursorAccent: "#0f1420",
      selectionBackground: "rgba(124, 137, 255, 0.24)",
      black: "#141925",
      red: "#ef6b73",
      green: "#7ad38b",
      yellow: "#e6c563",
      blue: "#7c89ff",
      magenta: "#b28cff",
      cyan: "#6dd9e8",
      white: "#e8edf7",
      brightBlack: "#5b6578",
      brightRed: "#ff8d95",
      brightGreen: "#92ef9f",
      brightYellow: "#f4d982",
      brightBlue: "#96a2ff",
      brightMagenta: "#c9a8ff",
      brightCyan: "#8de9f4",
      brightWhite: "#ffffff",
    },
  });
  machineXterm.open(surface);
  machineXtermKey = active.key;
  machineXterm.onData((data) => {
    const current = getActiveTerminal();
    if (!current?.connected || current.key !== machineXtermKey) return;
    sendMachineTerminalInput(data);
  });
  machineXterm.onKey(({ domEvent }) => {
    const current = getActiveTerminal();
    if (!current?.connected || current.key !== machineXtermKey) return;
    if (!domEvent) return;
    const key = String(domEvent.key || "").toLowerCase();
    if (key && key !== "unidentified") {
      patchTerminal(current.key, { lastEvent: `key:${key}` });
      if (state.machine.activeTerminalKey === current.key) {
        updateMachineTerminalOutput();
      }
    }
  });
  if (active.rawOutput) {
    machineXterm.write(active.rawOutput);
  } else if (active.output) {
    machineXterm.write(active.output);
  } else {
    machineXterm.write("终端已建立，等待输入…");
  }
  const { cols, rows } = computeMachineTerminalSize(screen);
  try {
    machineXterm.resize(cols, rows);
  } catch (_error) {}
  patchTerminal(active.key, { cols, rows });
  window.requestAnimationFrame(() => {
    try {
      machineXterm?.focus();
      patchTerminal(active.key, { lastEvent: "xterm-focus" });
      if (state.machine.activeTerminalKey === active.key) {
        updateMachineTerminalOutput();
      }
    } catch (_error) {}
  });
  return true;
}
