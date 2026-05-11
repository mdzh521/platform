import { state } from "../../core/state.js";
import { escapeHtml, formatDateTime } from "../../shared/utils.js";

export function getFilteredAssets() {
  return filterItems(state.machine.assets, "", ["name"]);
}

export function paginateItems(items, page, pageSize) {
  const normalizedPageSize = Math.max(1, Number(pageSize || 10));
  const totalItems = items.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / normalizedPageSize));
  const normalizedPage = Math.min(Math.max(1, Number(page || 1)), totalPages);
  const startIndex = totalItems === 0 ? 0 : (normalizedPage - 1) * normalizedPageSize;
  const endIndex = Math.min(startIndex + normalizedPageSize, totalItems);
  return {
    items: items.slice(startIndex, endIndex),
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalItems,
    totalPages,
    startIndex,
    endIndex,
  };
}

export function renderStatus(status) {
  const mapping = {
    online: "在线",
    warning: "需关注",
    offline: "离线",
  };
  return mapping[String(status || "").toLowerCase()] || "未知";
}

export function renderSessionStatus(status) {
  const mapping = {
    prepared: "已准备",
    active: "进行中",
    closed: "已关闭",
  };
  return mapping[String(status || "").toLowerCase()] || "会话";
}

export function renderEventLevel(level) {
  const mapping = {
    info: "信息",
    warning: "关注",
    error: "错误",
  };
  return mapping[String(level || "").toLowerCase()] || "事件";
}

export function renderEventLevelClass(level) {
  const normalized = String(level || "").toLowerCase();
  if (normalized === "warning") return "warning";
  if (normalized === "error") return "offline";
  return "prepared";
}

export function renderEventType(type) {
  const mapping = {
    quick_connect: "快速登录",
    ticket_issued: "终端票据",
    terminal_connected: "终端连接",
    terminal_closed: "终端关闭",
    terminal_error: "终端异常",
    terminal_signal: "终端信号",
    terminal_command: "命令执行",
  };
  return mapping[String(type || "").toLowerCase()] || "操作事件";
}

export function renderAccountAuthType(authType) {
  const mapping = {
    password: "密码",
    ssh_key: "SSH 私钥",
  };
  return mapping[String(authType || "").toLowerCase()] || "托管账号";
}

export function renderMetaPill(label, value) {
  return `<span class="meta-pill"><span>${escapeHtml(label)}</span><strong>${escapeHtml(value)}</strong></span>`;
}

export function renderAssetGroup(groupName) {
  return String(groupName || "").trim() || "未分组";
}

export function resolveAssetLoginPolicy(asset) {
  const assetPolicy = String(asset?.login_policy || "").toLowerCase();
  if (assetPolicy && assetPolicy !== "inherit_group") {
    return assetPolicy;
  }
  const groupName = String(asset?.group_name || "").trim();
  const group = (state.machine.groups || []).find((item) => String(item.name || "").trim() === groupName);
  if (group?.default_login_policy) {
    return String(group.default_login_policy).toLowerCase();
  }
  return "managed_first";
}

export function renderLoginPolicy(policy) {
  const mapping = {
    managed_first: "优先默认托管账号",
    manual_only: "仅手动认证",
    managed_only: "仅托管账号",
    inherit_group: "继承资产组策略",
  };
  return mapping[String(policy || "").toLowerCase()] || "优先默认托管账号";
}

export function renderAssetAccessHint(asset) {
  if (String(asset.access_mode || "").toLowerCase() === "via_gateway") {
    const gateway = String(asset.gateway_asset_name || asset.gateway_address || "").trim();
    if (gateway) {
      return `当前资产需要先经入口节点 ${gateway} 跳转访问`;
    }
    return "当前资产需要先经入口节点跳转访问";
  }
  const policy = resolveAssetLoginPolicy(asset);
  if (String(asset.status || "").toLowerCase() === "offline") {
    return "当前资产离线，建议先排查网络或主机状态";
  }
  if (policy === "manual_only") {
    return "当前资产要求手动认证登录";
  }
  if (policy === "managed_only") {
    return asset.account ? "当前资产优先使用托管账号登录" : "当前资产要求托管账号，建议先录入托管账号";
  }
  if (asset.account) {
    return "当前资产可直接尝试登录入口";
  }
  return "建议先补充登录账号或托管账号";
}

export function renderAssetAccessMode(accessMode) {
  return String(accessMode || "").toLowerCase() === "via_gateway" ? "通过入口节点" : "直连";
}

export function assetAccessModeOptions() {
  return [
    { value: "direct", label: "直连" },
    { value: "via_gateway", label: "通过入口节点 / 跳板机" },
  ];
}

export function gatewayNodeOptions(currentAssetID = 0) {
  return [{ value: "", label: "不指定入口节点" }, ...(state.machine.assets || [])
    .filter((item) => Boolean(item.is_gateway_node) && Number(item.id) !== Number(currentAssetID || 0))
    .map((item) => ({
      value: String(item.id),
      label: `${item.name} · ${item.address}`,
    }))];
}

export function renderAssetRiskTone(asset) {
  const status = String(asset.status || "").toLowerCase();
  if (status === "offline") return "danger";
  if (status === "warning") return "warn";
  if (!asset.account) return "warn";
  return "ok";
}

export function renderAssetRiskLabel(asset) {
  const status = String(asset.status || "").toLowerCase();
  if (status === "offline") return "连接不可用";
  if (status === "warning") return "存在风险";
  if (!asset.account) return "待补账号";
  return "可连接";
}

export function renderFoldSection(title, body, options = {}) {
  const meta = options.meta ? `<span class="section-meta">${escapeHtml(options.meta)}</span>` : "";
  const open = options.open ? " open" : "";
  return `<details class="fold-section"${open}><summary><span>${escapeHtml(title)}</span>${meta}</summary><div class="fold-body">${body}</div></details>`;
}

export function platformOptions() {
  return [
    { value: "linux", label: "Linux" },
    { value: "windows", label: "Windows" },
    { value: "network", label: "Network" },
    { value: "database", label: "Database" },
  ];
}

export function protocolOptions() {
  return [
    { value: "ssh", label: "SSH" },
    { value: "rdp", label: "RDP" },
    { value: "vnc", label: "VNC" },
    { value: "mysql", label: "MySQL" },
  ];
}

export function statusOptions() {
  return [
    { value: "online", label: "在线" },
    { value: "warning", label: "需关注" },
    { value: "offline", label: "离线" },
  ];
}

export function loginPolicyOptions() {
  return [
    { value: "inherit_group", label: "继承资产组策略" },
    { value: "managed_first", label: "优先默认托管账号" },
    { value: "manual_only", label: "仅手动认证" },
    { value: "managed_only", label: "仅托管账号" },
  ];
}

export function assetGroupOptions() {
  const groups = flattenAssetGroups(state.machine.groups || []).map((group) => ({
    value: group.name,
    label: `${"　".repeat(Number(group.depth || 0))}${group.name}`,
  }));
  return [{ value: "", label: "未分组" }, ...groups];
}

export function flattenAssetGroups(groups, depth = 0) {
  if (!Array.isArray(groups) || !groups.length) return [];
  return groups.flatMap((group) => {
    const current = { ...group, depth };
    const children = flattenAssetGroups(group.children || [], depth + 1);
    return [current, ...children];
  });
}

export function renderUpdatedAt(value) {
  return formatDateTime(value);
}
