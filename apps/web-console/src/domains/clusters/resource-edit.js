import { api } from "../../core/api.js";
import { state } from "../../core/state.js";
import { confirmAction, openFormModal, toast } from "../../core/ui.js";

let ctx = null;

export function configureResourceEditing(deps) {
  ctx = deps;
}

export function supportsResourceEdit(kind) {
  return ["ConfigMap", "Secret", "Service", "Ingress", "ResourceQuota", "LimitRange", "ServiceAccount", "PersistentVolumeClaim"].includes(String(kind || ""));
}

export async function openEditSelectedResourceModal() {
  const detail = state.resourceExplorer.detail;
  if (!detail || !supportsResourceEdit(detail.kind)) {
    toast("当前资源类型暂不支持编辑");
    return;
  }
  const proceed = await confirmImpactBeforeEdit(detail);
  if (!proceed) {
    return;
  }
  if (detail.kind === "ConfigMap") {
    await openEditConfigMapModal(detail);
    return;
  }
  if (detail.kind === "Service") {
    await openEditServiceModal(detail);
    return;
  }
  if (detail.kind === "Secret") {
    await openEditSecretModal(detail);
    return;
  }
  if (detail.kind === "ResourceQuota") {
    await openEditResourceQuotaModal(detail);
    return;
  }
  if (detail.kind === "LimitRange") {
    await openEditLimitRangeModal(detail);
    return;
  }
  if (detail.kind === "ServiceAccount") {
    await openEditServiceAccountModal(detail);
    return;
  }
  if (detail.kind === "PersistentVolumeClaim") {
    await openEditPersistentVolumeClaimModal(detail);
    return;
  }
  await openEditIngressModal(detail);
}

async function confirmImpactBeforeEdit(detail) {
  const impact = describeResourceImpact(detail);
  if (!impact) {
    return true;
  }
  return confirmAction({
    eyebrow: "影响提示",
    title: `编辑 ${detail.kind} / ${detail.name}`,
    copy: impact,
    confirmText: "继续编辑",
  });
}

function describeResourceImpact(detail) {
  const data = detail.data || {};
  if (detail.kind === "ConfigMap") {
    const count = (data.config_map?.referenced_by || []).length;
    return count ? `这个 ConfigMap 当前被 ${count} 个已同步工作负载引用，修改后可能触发配置漂移或需要重启工作负载才能生效。` : "";
  }
  if (detail.kind === "Secret") {
    const count = (data.secret?.referenced_by || []).length;
    return count ? `这个 Secret 当前被 ${count} 个已同步工作负载引用，更新后可能影响认证、镜像拉取或运行时连接信息。` : "";
  }
  if (detail.kind === "Service") {
    const count = (data.service?.selected_by || []).length;
    return count ? `这个 Service 当前选中了 ${count} 个工作负载，修改 selector、端口或类型可能直接影响流量转发。` : "";
  }
  if (detail.kind === "Ingress") {
    const count = (data.ingress?.backend_services || []).length;
    return count ? `这个 Ingress 当前关联了 ${count} 个后端 Service，修改规则或 TLS 可能影响入口流量和域名访问。` : "";
  }
  if (detail.kind === "ServiceAccount") {
    const count = (data.service_account?.referenced_by || []).length;
    return count ? `这个 ServiceAccount 当前被 ${count} 个工作负载使用，修改 imagePullSecrets 或 annotations 可能影响镜像拉取或云权限映射。` : "";
  }
  if (detail.kind === "PersistentVolumeClaim") {
    const count = (data.persistent_volume_claim?.mounted_by || []).length;
    return count ? `这个 PVC 当前被 ${count} 个工作负载挂载，扩容或修改 annotations 前需要确认底层存储类是否支持在线变更。` : "";
  }
  if (detail.kind === "ResourceQuota") {
    const count = (data.resource_quota?.impacted_workloads || []).length;
    return count ? `这个 ResourceQuota 会影响当前命名空间内 ${count} 个已同步工作负载，收紧配额后可能导致扩容或新建失败。` : "";
  }
  if (detail.kind === "LimitRange") {
    const count = (data.limit_range?.impacted_workloads || []).length;
    return count ? `这个 LimitRange 会影响当前命名空间内 ${count} 个已同步工作负载，修改默认资源限制后可能改变后续发布行为。` : "";
  }
  return "";
}

async function openEditConfigMapModal(detail) {
  const item = detail.data?.config_map || {};
  const entries = (item.entries || []).map((entry) => `${entry.key || ""}=${entry.value || ""}`).join("\n");
  openFormModal({
    eyebrow: "ConfigMap",
    title: `编辑 ${detail.name}`,
    copy: "每行一个 key=value，保存后会原位刷新资源详情。",
    fields: [
      { label: "键值内容", name: "entries_text", type: "textarea", rows: 14, value: entries, placeholder: "APP_ENV=prod" },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      config_map: {
        entries_text: String(form.get("entries_text") || ""),
      },
    }),
  });
}

async function openEditSecretModal(detail) {
  const item = detail.data?.secret || {};
  openFormModal({
    eyebrow: "Secret",
    title: `编辑 ${detail.name}`,
    copy: "出于安全考虑，现有密文值不会回显。这里支持新增/覆盖 key=value，或按 key 删除指定条目，同时维护 annotations。",
    fields: [
      { label: "新增或覆盖键值", name: "entries_text", type: "textarea", rows: 10, value: "", placeholder: "username=demo\npassword=s3cret" },
      { label: "删除这些键", name: "remove_keys_text", type: "textarea", rows: 4, value: "", placeholder: "old-password\nlegacy-token" },
      { label: "Annotations", name: "annotations_text", type: "textarea", rows: 6, value: serializeStringMap(item.annotations || {}), placeholder: "sealedsecrets.bitnami.com/managed=true" },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      secret: {
        entries_text: String(form.get("entries_text") || ""),
        remove_keys_text: String(form.get("remove_keys_text") || ""),
        annotations_text: String(form.get("annotations_text") || ""),
      },
    }),
  });
}

async function openEditServiceModal(detail) {
  const item = detail.data?.service || {};
  const portsText = (item.ports || []).map((port) => [
    port.port ?? "",
    port.target_port ?? "",
    port.name || "",
    port.protocol || "TCP",
    port.node_port || "",
  ].join(",")).join("\n");
  openFormModal({
    eyebrow: "Service",
    title: `编辑 ${detail.name}`,
    copy: "端口格式为 port,targetPort,name,protocol,nodePort。后两项可留空。",
    fields: [
      { label: "Service 类型", name: "type", type: "select", value: item.type || "ClusterIP", options: [{ value: "ClusterIP", label: "ClusterIP" }, { value: "NodePort", label: "NodePort" }, { value: "LoadBalancer", label: "LoadBalancer" }] },
      { label: "Session Affinity", name: "session_affinity", type: "select", value: item.session_affinity || "None", options: [{ value: "None", label: "None" }, { value: "ClientIP", label: "ClientIP" }] },
      { label: "Selector", name: "selector_text", type: "textarea", rows: 5, value: (item.selector || []).join("\n"), placeholder: "app=hello-nginx" },
      { label: "External IPs", name: "external_ips_text", type: "textarea", rows: 4, value: (item.external_ips || []).join("\n"), placeholder: "10.0.0.10" },
      { label: "Ports", name: "ports_text", type: "textarea", rows: 8, value: portsText, placeholder: "80,80,http,TCP," },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      service: {
        type: String(form.get("type") || ""),
        session_affinity: String(form.get("session_affinity") || ""),
        selector_text: String(form.get("selector_text") || ""),
        external_ips_text: String(form.get("external_ips_text") || ""),
        ports_text: String(form.get("ports_text") || ""),
      },
    }),
  });
}

async function openEditIngressModal(detail) {
  const item = detail.data?.ingress || {};
  const rulesText = (item.rules || []).flatMap((rule) => {
    const host = rule.host || "";
    return (rule.paths || []).map((path) => [host, path.path || "/", path.path_type || "Prefix", path.service || "", path.service_port || ""].join(","));
  }).join("\n");
  const tlsText = (item.tls || []).map((entry) => `${entry.secret_name || ""}:${(entry.hosts || []).join("|")}`).join("\n");
  openFormModal({
    eyebrow: "Ingress",
    title: `编辑 ${detail.name}`,
    copy: "规则格式为 host,path,pathType,service,port；TLS 格式为 secretName:host1|host2。",
    fields: [
      { label: "Ingress Class", name: "ingress_class", value: item.ingress_class || "", placeholder: "nginx" },
      { label: "Annotations", name: "annotations_text", type: "textarea", rows: 6, value: serializeStringMap(item.annotations || {}), placeholder: "nginx.ingress.kubernetes.io/rewrite-target=/" },
      { label: "Rules", name: "rules_text", type: "textarea", rows: 8, value: rulesText, placeholder: "demo.local,/,Prefix,hello-nginx,80" },
      { label: "TLS", name: "tls_text", type: "textarea", rows: 5, value: tlsText, placeholder: "demo-tls:demo.local|www.demo.local" },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      ingress: {
        ingress_class: String(form.get("ingress_class") || ""),
        annotations_text: String(form.get("annotations_text") || ""),
        rules_text: String(form.get("rules_text") || ""),
        tls_text: String(form.get("tls_text") || ""),
      },
    }),
  });
}

async function openEditResourceQuotaModal(detail) {
  const item = detail.data?.resource_quota || {};
  openFormModal({
    eyebrow: "ResourceQuota",
    title: `编辑 ${detail.name}`,
    copy: "Hard 指标按 key=value 多行填写，Scopes 按每行一个 scope 填写。",
    fields: [
      { label: "Hard Metrics", name: "hard_text", type: "textarea", rows: 8, value: serializeStringMap(item.hard || {}), placeholder: "pods=12\nrequests.cpu=2\nrequests.memory=2Gi" },
      { label: "Scopes", name: "scopes_text", type: "textarea", rows: 4, value: (item.scopes || []).join("\n"), placeholder: "BestEffort" },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      resource_quota: {
        hard_text: String(form.get("hard_text") || ""),
        scopes_text: String(form.get("scopes_text") || ""),
      },
    }),
  });
}

async function openEditLimitRangeModal(detail) {
  const item = detail.data?.limit_range || {};
  openFormModal({
    eyebrow: "LimitRange",
    title: `编辑 ${detail.name}`,
    copy: "规则支持 JSON 数组或 YAML 数组。建议基于当前内容修改，避免丢失字段。",
    fields: [
      {
        label: "Rules",
        name: "limits_text",
        type: "textarea",
        rows: 16,
        value: JSON.stringify(item.limits || [], null, 2),
        placeholder: "[{\"type\":\"Container\",\"default\":{\"cpu\":\"500m\",\"memory\":\"512Mi\"}}]",
      },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      limit_range: {
        limits_text: String(form.get("limits_text") || ""),
      },
    }),
  });
}

async function openEditServiceAccountModal(detail) {
  const item = detail.data?.service_account || {};
  openFormModal({
    eyebrow: "ServiceAccount",
    title: `编辑 ${detail.name}`,
    copy: "这里主要维护 imagePullSecrets 和 annotations，适合和私有镜像仓库集成一起使用。",
    fields: [
      {
        label: "Image Pull Secrets",
        name: "image_pull_secrets_text",
        type: "textarea",
        rows: 6,
        value: (item.image_pull_secrets || []).join("\n"),
        placeholder: "regcred\nharbor-robot",
      },
      {
        label: "Annotations",
        name: "annotations_text",
        type: "textarea",
        rows: 6,
        value: serializeStringMap(item.annotations || {}),
        placeholder: "eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/demo",
      },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      service_account: {
        image_pull_secrets_text: String(form.get("image_pull_secrets_text") || ""),
        annotations_text: String(form.get("annotations_text") || ""),
      },
    }),
  });
}

async function openEditPersistentVolumeClaimModal(detail) {
  const item = detail.data?.persistent_volume_claim || {};
  openFormModal({
    eyebrow: "PersistentVolumeClaim",
    title: `编辑 ${detail.name}`,
    copy: "这里只开放容量请求和 annotations。StorageClass、accessModes 这类字段多数场景不可在线修改，不在这里暴露。",
    fields: [
      {
        label: "Requested Storage",
        name: "requested_storage",
        value: item.requested || "",
        placeholder: "10Gi",
      },
      {
        label: "Annotations",
        name: "annotations_text",
        type: "textarea",
        rows: 6,
        value: serializeStringMap(item.annotations || {}),
        placeholder: "volume.beta.kubernetes.io/storage-class=standard",
      },
    ],
    submitText: "保存资源",
    onSubmit: async (form) => updateSelectedResource({
      persistent_volume_claim: {
        requested_storage: String(form.get("requested_storage") || ""),
        annotations_text: String(form.get("annotations_text") || ""),
      },
    }),
  });
}

async function updateSelectedResource(body) {
  const clusterID = ctx.selectedResourceClusterID();
  const namespace = ctx.selectedResourceNamespace();
  const detail = state.resourceExplorer.detail;
  if (!clusterID || !namespace || !detail) {
    throw new Error("当前资源定位信息不完整");
  }
  const params = new URLSearchParams({
    cluster_id: String(clusterID),
    namespace,
    kind: detail.kind,
    name: detail.name,
  });
  const payload = await api(`/api/v1/k8s/resources/detail?${params.toString()}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
  state.resourceExplorer.manifest = "";
  await ctx.loadResources(clusterID, namespace, ctx.selectedResourceKind());
  await ctx.loadResourceDetail(detail.name);
  if (state.business.selectedWorkloadID && state.business.workloadInspectorView === "resources") {
    state.workloadInspector.resources = null;
    await ctx.switchWorkloadInspector("resources");
  }
  toast(payload.data.message ? `${payload.data.message}：${detail.name}` : "资源已更新");
}

function serializeStringMap(items) {
  return Object.entries(items || {}).map(([key, value]) => `${key}=${value ?? ""}`).join("\n");
}
