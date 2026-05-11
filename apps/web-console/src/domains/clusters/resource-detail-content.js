import { state } from "../../core/state.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";

export function renderResourceDetailContent() {
  const detail = state.resourceExplorer.detail;
  if (!detail) {
    return emptyState("当前资源详情尚未加载");
  }
  const payload = detail.data || {};
  const manifestBlock = renderFoldSection("YAML", `<pre class="detail-code">${escapeHtml(state.resourceExplorer.manifest || "点击上方 YAML 模式加载资源清单")}</pre>`, {
    meta: state.business.resourceManifestMode === "full" ? "完整模式" : "简洁模式",
    open: false,
  });
  if (detail.kind === "Service") {
    const service = payload.service || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(service.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">类型</strong><span>${escapeHtml(service.type || "-")}</span></div><div class="detail-item"><strong class="muted-label">ClusterIP</strong><span>${escapeHtml(service.cluster_ip || "-")}</span></div><div class="detail-item"><strong class="muted-label">Session Affinity</strong><span>${escapeHtml(service.session_affinity || "-")}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(service.created_at))}</span></div></div>${renderNameResources("Selector", service.selector || [])}${renderNameResources("External IPs", service.external_ips || [])}${renderServicePortDetail(service.ports || [])}${renderNameResources("Endpoints", service.endpoints || [])}${renderAnnotationList(service.annotations || {})}${renderReferenceList("Selected Workloads", service.selected_by || [])}${manifestBlock}</div>`;
  }
  if (detail.kind === "Ingress") {
    const ingress = payload.ingress || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(ingress.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">Ingress Class</strong><span>${escapeHtml(ingress.ingress_class || "-")}</span></div><div class="detail-item"><strong class="muted-label">Default Backend</strong><span>${escapeHtml(ingress.default_backend || "-")}</span></div><div class="detail-item"><strong class="muted-label">地址</strong><span>${(ingress.addresses || []).map((item) => escapeHtml(item)).join("，") || "-"}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(ingress.created_at))}</span></div></div>${renderIngressRuleDetail(ingress.rules || [])}${renderIngressTLSDetail(ingress.tls || [])}${renderIngressBackendServices(ingress.backend_services || [])}${renderAnnotationList(ingress.annotations || {})}${manifestBlock}</div>`;
  }
  if (detail.kind === "ConfigMap") {
    const configMap = payload.config_map || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(configMap.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">键数量</strong><span>${escapeHtml(String(configMap.data_count ?? 0))}</span></div><div class="detail-item"><strong class="muted-label">状态</strong><span>${escapeHtml(configMap.immutable ? "Immutable" : "Mutable")}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(configMap.created_at))}</span></div></div>${renderConfigMapEntries(configMap.entries || [])}${renderAnnotationList(configMap.annotations || {})}${renderReferenceList("Referenced By", configMap.referenced_by || [])}${manifestBlock}</div>`;
  }
  if (detail.kind === "ServiceAccount") {
    const serviceAccount = payload.service_account || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(serviceAccount.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">Secret 数量</strong><span>${escapeHtml(String((serviceAccount.secrets || []).length))}</span></div><div class="detail-item"><strong class="muted-label">Image Pull Secrets</strong><span>${escapeHtml(String((serviceAccount.image_pull_secrets || []).length))}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(serviceAccount.created_at))}</span></div></div>${renderNameResources("Secrets", serviceAccount.secrets || [])}${renderNameResources("Image Pull Secrets", serviceAccount.image_pull_secrets || [])}${renderAnnotationList(serviceAccount.annotations || {})}${renderReferenceList("Referenced By", serviceAccount.referenced_by || [])}${manifestBlock}</div>`;
  }
  if (detail.kind === "PersistentVolumeClaim") {
    const claim = payload.persistent_volume_claim || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(claim.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">状态</strong><span>${escapeHtml(claim.status || "-")}</span></div><div class="detail-item"><strong class="muted-label">StorageClass</strong><span>${escapeHtml(claim.storage_class || "-")}</span></div><div class="detail-item"><strong class="muted-label">Volume</strong><span>${escapeHtml(claim.volume || "-")}</span></div><div class="detail-item"><strong class="muted-label">请求容量</strong><span>${escapeHtml(claim.requested || "-")}</span></div><div class="detail-item"><strong class="muted-label">已分配容量</strong><span>${escapeHtml(claim.capacity || "-")}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(claim.created_at))}</span></div></div>${renderNameResources("Access Modes", claim.access_modes || [])}${renderAnnotationList(claim.annotations || {})}${renderReferenceList("Mounted By", claim.mounted_by || [])}${manifestBlock}</div>`;
  }
  if (detail.kind === "ResourceQuota") {
    const quota = payload.resource_quota || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(quota.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(quota.created_at))}</span></div><div class="detail-item"><strong class="muted-label">Scope 数量</strong><span>${escapeHtml(String((quota.scopes || []).length))}</span></div><div class="detail-item"><strong class="muted-label">Hard 指标</strong><span>${escapeHtml(String(Object.keys(quota.hard || {}).length))}</span></div></div>${renderQuotaMetricTable("Hard", quota.hard || {})}${renderQuotaMetricTable("Used", quota.used || {})}${renderNameResources("Scopes", quota.scopes || [])}${renderAnnotationList(quota.annotations || {})}${renderReferenceList("Impacted Workloads", quota.impacted_workloads || [])}${manifestBlock}</div>`;
  }
  if (detail.kind === "LimitRange") {
    const limitRange = payload.limit_range || {};
    return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(limitRange.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(limitRange.created_at))}</span></div><div class="detail-item"><strong class="muted-label">规则数量</strong><span>${escapeHtml(String((limitRange.limits || []).length))}</span></div></div>${renderLimitRangeRules(limitRange.limits || [])}${renderAnnotationList(limitRange.annotations || {})}${renderReferenceList("Impacted Workloads", limitRange.impacted_workloads || [])}${manifestBlock}</div>`;
  }
  const secret = payload.secret || {};
  return `<div class="detail-stack"><div class="detail-list"><div class="detail-item"><strong class="muted-label">名称</strong><span>${escapeHtml(secret.name || detail.name)}</span></div><div class="detail-item"><strong class="muted-label">类型</strong><span>${escapeHtml(secret.type || "-")}</span></div><div class="detail-item"><strong class="muted-label">键数量</strong><span>${escapeHtml(String(secret.data_count ?? 0))}</span></div><div class="detail-item"><strong class="muted-label">创建时间</strong><span>${escapeHtml(formatDateTime(secret.created_at))}</span></div></div>${renderSecretKeySizes(secret.key_sizes || [])}${renderAnnotationList(secret.annotations || {})}${renderReferenceList("Referenced By", secret.referenced_by || [])}${manifestBlock}</div>`;
}

function renderAnnotationList(annotations) {
  const entries = Object.entries(annotations || {});
  if (!entries.length) {
    return renderFoldSection("Annotations", emptyState("当前资源没有 annotations"), { meta: "0", open: false });
  }
  return renderFoldSection("Annotations", `<div class="replicaset-list">${entries.map(([key, value]) => `<article class="pod-item"><div><p><strong>${escapeHtml(key)}</strong></p><p class="table-meta code-cell">${escapeHtml(String(value || ""))}</p></div></article>`).join("")}</div>`, {
    meta: `${entries.length} 项`,
    open: false,
  });
}

function renderConfigMapEntries(entries) {
  if (!entries.length) return renderFoldSection("Entries", emptyState("当前 ConfigMap 没有键值内容"), { meta: "0", open: false });
  return renderFoldSection("Entries", `<div class="replicaset-list">${entries.map((entry) => `<article class="pod-item"><div><p><strong>${escapeHtml(entry.key || "-")}</strong></p><p class="table-meta">${escapeHtml(`${entry.char_count || 0} chars / ${entry.line_count || 0} lines`)}</p><pre class="detail-code compact">${escapeHtml(entry.value || "")}</pre></div></article>`).join("")}</div>`, {
    meta: `${entries.length} 个键`,
    open: entries.length <= 2,
  });
}

function renderSecretKeySizes(items) {
  if (!items.length) return renderFoldSection("Keys", emptyState("当前 Secret 没有可展示的键"), { meta: "0", open: false });
  return renderFoldSection("Keys", `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.key || "-")}</strong></p><p class="table-meta">encoded size: ${escapeHtml(String(item.encoded_size ?? 0))}</p></div></article>`).join("")}</div>`, {
    meta: `${items.length} 个键`,
    open: items.length <= 3,
  });
}

function renderQuotaMetricTable(title, items) {
  const entries = Object.entries(items || {});
  if (!entries.length) return renderFoldSection(title, emptyState(`当前 ${title} 没有指标`), { meta: "0", open: false });
  return renderFoldSection(title, `<div class="replicaset-list">${entries.map(([key, value]) => `<article class="pod-item"><div><p><strong>${escapeHtml(key)}</strong></p><p class="table-meta">${escapeHtml(String(value || "-"))}</p></div></article>`).join("")}</div>`, {
    meta: `${entries.length} 项`,
    open: title === "Hard",
  });
}

function renderLimitRangeRules(items) {
  if (!items.length) return renderFoldSection("Rules", emptyState("当前 LimitRange 没有规则"), { meta: "0", open: false });
  return renderFoldSection("Rules", `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.type || "-")}</strong></p><p class="table-meta code-cell">default: ${escapeHtml(JSON.stringify(item.default || {}))}</p><p class="table-meta code-cell">defaultRequest: ${escapeHtml(JSON.stringify(item.default_request || {}))}</p><p class="table-meta code-cell">min: ${escapeHtml(JSON.stringify(item.min || {}))}</p><p class="table-meta code-cell">max: ${escapeHtml(JSON.stringify(item.max || {}))}</p><p class="table-meta code-cell">ratio: ${escapeHtml(JSON.stringify(item.max_limit_request_ratio || {}))}</p></div></article>`).join("")}</div>`, {
    meta: `${items.length} 条`,
    open: items.length <= 1,
  });
}

function renderReferenceList(title, items) {
  if (!items.length) return renderFoldSection(title, emptyState("当前资源还没有被已同步工作负载引用"), { meta: "0", open: false });
  return renderFoldSection(title, `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.kind || "-")} / ${escapeHtml(item.name || "-")}</strong></p><p class="table-meta">${escapeHtml(item.namespace || "-")} / ${escapeHtml(item.status || "-")}</p><p class="table-meta code-cell">${escapeHtml(item.image || "-")}</p></div><button class="ghost-button" data-action="open-referenced-workload" data-id="${escapeHtml(JSON.stringify({ kind: item.kind || "", name: item.name || "", namespace: item.namespace || "" }))}">进入详情</button></article>`).join("")}</div>`, {
    meta: `${items.length} 个引用`,
    open: items.length <= 2,
  });
}

function renderNameResources(title, items) {
  if (!items.length) return renderFoldSection(title, emptyState(`当前资源没有 ${title}`), { meta: "0", open: false });
  return renderFoldSection(title, `<div class="resource-pill-row">${items.map((item) => `<span class="cluster-chip">${escapeHtml(item)}</span>`).join("")}</div>`, {
    meta: `${items.length} 项`,
    open: items.length <= 3,
  });
}

function renderServicePortDetail(ports) {
  if (!ports.length) return renderFoldSection("Ports", emptyState("当前 Service 没有端口配置"), { meta: "0", open: false });
  return renderFoldSection("Ports", `<div class="replicaset-list">${ports.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name || "default")}</strong></p><p class="table-meta">${escapeHtml(item.protocol || "TCP")} / ${escapeHtml(String(item.port ?? "-"))} -> ${escapeHtml(String(item.target_port ?? "-"))}</p><p class="table-meta">NodePort: ${escapeHtml(String(item.node_port || "-"))}</p></div></article>`).join("")}</div>`, {
    meta: `${ports.length} 个端口`,
    open: true,
  });
}

function renderIngressRuleDetail(rules) {
  if (!rules.length) return renderFoldSection("Rules", emptyState("当前 Ingress 没有规则"), { meta: "0", open: false });
  return renderFoldSection("Rules", `<div class="replicaset-list">${rules.map((rule) => `<article class="pod-item"><div><p><strong>${escapeHtml(rule.host || "-")}</strong></p>${(rule.paths || []).map((path) => `<p class="table-meta">${escapeHtml(path.path || "/")} / ${escapeHtml(path.path_type || "-")} / ${escapeHtml(path.service || "-")} : ${escapeHtml(path.service_port || "-")}</p>`).join("")}</div></article>`).join("")}</div>`, {
    meta: `${rules.length} 条规则`,
    open: true,
  });
}

function renderIngressTLSDetail(items) {
  if (!items.length) return renderFoldSection("TLS", emptyState("当前 Ingress 没有 TLS 配置"), { meta: "0", open: false });
  return renderFoldSection("TLS", `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.secret_name || "-")}</strong></p><p class="table-meta">${(item.hosts || []).map((host) => escapeHtml(host)).join("，") || "-"}</p></div></article>`).join("")}</div>`, {
    meta: `${items.length} 组`,
    open: false,
  });
}

function renderIngressBackendServices(items) {
  if (!items.length) return renderFoldSection("Backend Services", emptyState("当前 Ingress 没有后端 Service 路由"), { meta: "0", open: false });
  return renderFoldSection("Backend Services", `<div class="replicaset-list">${items.map((item) => `<article class="pod-item"><div><p><strong>${escapeHtml(item.name || "-")}</strong></p><p class="table-meta">${escapeHtml(item.status || "-")} / ${escapeHtml(item.type || "-")} / ClusterIP ${escapeHtml(item.cluster_ip || "-")}</p><p class="table-meta">Routes: ${(item.routes || []).map((route) => escapeHtml(route)).join("，") || "-"}</p><p class="table-meta">Ports: ${(item.ports || []).map((port) => escapeHtml(`${port.name || "default"} ${port.port ?? "-"}->${port.target_port ?? "-"}`)).join("，") || "-"}</p>${(item.selected_by || []).length ? `<p class="table-meta">Workloads: ${(item.selected_by || []).map((workload) => escapeHtml(`${workload.kind}/${workload.name}`)).join("，")}</p>` : ""}</div>${(item.selected_by || []).length ? `<button class="ghost-button" data-action="open-referenced-workload" data-id="${escapeHtml(JSON.stringify({ kind: item.selected_by[0]?.kind || "", name: item.selected_by[0]?.name || "", namespace: item.selected_by[0]?.namespace || "" }))}">进入一个工作负载</button>` : ""}</article>`).join("")}</div>`, {
    meta: `${items.length} 个后端`,
    open: true,
  });
}

function renderFoldSection(title, body, { meta = "", open = false } = {}) {
  return `<details class="fold-section"${open ? " open" : ""}><summary><span class="eyebrow">${escapeHtml(title)}</span>${meta ? `<span class="table-meta">${escapeHtml(meta)}</span>` : ""}</summary><div class="fold-section-body">${body}</div></details>`;
}
