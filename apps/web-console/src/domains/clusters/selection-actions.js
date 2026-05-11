import { api } from "../../core/api.js";
import { state } from "../../core/state.js";
import { showErrorDialog, toast } from "../../core/ui.js";

let ctx = null;

export function configureK8sSelectionActions(deps) {
  ctx = deps;
}

export function resetWorkloadInspector() {
  state.workloadInspector.detail = null;
  state.workloadInspector.rollout = null;
  state.workloadInspector.resources = null;
  state.workloadInspector.resourceDetail = null;
  state.workloadInspector.pods = [];
  state.workloadInspector.events = [];
  state.workloadInspector.logs = "";
  state.workloadInspector.logSourcePod = "";
  state.workloadInspector.selectedPodName = "";
  state.workloadInspector.manifest = "";
  state.workloadInspector.followLogs = false;
  state.workloadInspector.followIntervalSeconds = 5;
  state.workloadInspector.execResult = null;
  state.workloadInspector.interactiveTerminal = null;
  state.workloadInspector.interactiveTerminalOutput = "";
  state.workloadInspector.interactiveTerminalConnected = false;
  state.workloadInspector.interactiveTerminalError = "";
}

export function resetResourceExplorer() {
  state.resourceExplorer.detail = null;
  state.resourceExplorer.manifest = "";
}

export async function selectWorkload(id) {
  state.business.selectedWorkloadID = id;
  state.business.workloadListPage = "detail";
  state.business.workloadInspectorView = "overview";
  state.business.workloadManifestMode = "compact";
  ctx.switchBusinessView("workload-detail");
  ctx.switchWorkloadPage("detail");
  ctx.renderWorkloads();
  document.getElementById("business-page-workload-detail")?.scrollIntoView({ behavior: "smooth", block: "start" });
  await ctx.loadWorkloadInspector(id);
}

export async function selectResource(name) {
  state.business.selectedResourceName = name;
  state.business.resourceListPage = "detail";
  state.business.resourceManifestMode = "compact";
  ctx.switchResourcePage("detail");
  ctx.renderResources();
  await ctx.loadResourceDetail(name);
}

export async function openReferencedWorkload(payload) {
  const target = JSON.parse(payload);
  if (!target.namespace || !target.name) {
    await showErrorDialog({ title: "定位失败", copy: "工作负载定位信息不完整" });
    return;
  }
  state.business.selectedNamespace = target.namespace;
  ctx.renderClusterSelectors();
  await ctx.loadWorkloads(ctx.selectedResourceClusterID(), target.namespace);
  const workload = state.workloads.find((item) => item.namespace_name === target.namespace && item.name === target.name && (!target.kind || item.kind === target.kind));
  if (!workload) {
    await showErrorDialog({ title: "未找到工作负载", copy: "当前工作负载尚未同步到平台列表" });
    return;
  }
  ctx.switchBusinessView("workloads");
  await selectWorkload(workload.id);
}

export async function refreshSelectedWorkload() {
  const workloadID = state.business.selectedWorkloadID;
  if (!workloadID) return;
  const current = state.workloads.find((item) => item.id === workloadID) || state.workloadInspector.detail || null;
  const previousPage = state.business.workloadListPage;
  const previousInspectorView = state.business.workloadInspectorView;
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
  const next = state.workloads.find((item) => item.id === workloadID)
    || state.workloads.find((item) => current && item.kind === current.kind && item.name === current.name && item.namespace_name === (current.namespace_name || current.namespace));
  if (!next) {
    state.business.selectedWorkloadID = null;
    state.business.workloadListPage = "list";
    ctx.renderWorkloads();
    await showErrorDialog({ title: "未找到工作负载", copy: "当前工作负载在刷新后未找到，已返回列表" });
    return;
  }
  state.business.selectedWorkloadID = next.id;
  state.business.workloadListPage = previousPage;
  state.business.workloadInspectorView = ctx.normalizeInspectorView(next.kind, previousInspectorView);
  ctx.switchWorkloadPage(previousPage);
  await ctx.loadWorkloadInspector(next.id);
}

export async function refreshSelectedResource() {
  const name = state.business.selectedResourceName;
  if (!name) return;
  await ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind());
  await ctx.loadResourceDetail(name);
}

export async function openResourceDetail(kind, name) {
  const id = state.business.selectedWorkloadID;
  if (!id) return;
  if (kind === "service") {
    const payload = await api(`/api/v1/k8s/workloads/${id}/services/${encodeURIComponent(name)}`);
    state.workloadInspector.resourceDetail = { kind, name, data: payload.data.service };
  } else if (kind === "ingress") {
    const payload = await api(`/api/v1/k8s/workloads/${id}/ingresses/${encodeURIComponent(name)}`);
    state.workloadInspector.resourceDetail = { kind, name, data: payload.data.ingress };
  }
  ctx.renderWorkloadDetailPanel();
}

export async function openLinkedNamespaceResource(payload) {
  const target = JSON.parse(payload);
  const clusterID = ctx.selectedWorkloadClusterID();
  const namespace = ctx.selectedWorkloadNamespace();
  const kind = normalizeResourceKindName(target.kind);
  const name = String(target.name || "").trim();
  if (!clusterID || !namespace || !kind || !name) {
    await showErrorDialog({ title: "定位失败", copy: "资源定位信息不完整" });
    return;
  }
  state.business.selectedResourceKind = kind;
  ctx.setResourceKindFilter(kind);
  await ctx.openNamespaceResources({
    cluster_id: clusterID,
    cluster_name: state.business.selectedClusterName || state.workloadInspector.detail?.cluster_name || "",
    name: namespace,
  });
  await selectResource(name);
}

export function normalizeResourceKindName(kind) {
  const value = String(kind || "").trim();
  if (!value) return value;
  const lower = value.toLowerCase();
  const mapping = {
    service: "Service",
    ingress: "Ingress",
    configmap: "ConfigMap",
    secret: "Secret",
    resourcequota: "ResourceQuota",
    limitrange: "LimitRange",
    serviceaccount: "ServiceAccount",
    persistentvolumeclaim: "PersistentVolumeClaim",
  };
  return mapping[lower] || value;
}
