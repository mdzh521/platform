import { api } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { closeDrawer, confirmAction, openFormModal, showErrorDialog, toast } from "../../core/ui.js";

let ctx = null;

export function configureK8sBusinessActions(deps) {
  ctx = deps;
}

export async function onCreateCluster(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/k8s/clusters", {
    method: "POST",
    body: JSON.stringify({
      name: form.get("name"),
      code: form.get("code"),
      environment: form.get("environment"),
      provider: form.get("provider"),
      api_endpoint: form.get("api_endpoint"),
      auth_type: form.get("auth_type"),
      credential: form.get("credential"),
      description: form.get("description"),
    }),
  });
  event.target.reset();
  closeDrawer("cluster");
  toast("集群已保存");
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID())]);
}

export async function onCreateNamespace(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/k8s/namespaces", {
    method: "POST",
    body: JSON.stringify({
      cluster_id: Number(form.get("cluster_id")),
      name: form.get("name"),
      display_name: form.get("display_name"),
      description: form.get("description"),
      source_type: "manual",
      status: "active",
    }),
  });
  event.target.reset();
  closeDrawer("namespace");
  toast("命名空间已保存");
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID())]);
}

export function switchBusinessView(view) {
  if (!["workloads", "workload-detail"].includes(view)) {
    ctx.stopLogFollow();
    state.workloadInspector.followLogs = false;
  }
  state.business.view = view;
  elements.businessViewButtons.forEach((button) => {
    const activeView = view === "workload-detail" ? "workloads" : view;
    button.classList.toggle("active", button.dataset.businessView === activeView);
  });
  elements.businessPages.forEach((page) => {
    page.classList.toggle("hidden", page.id !== `business-page-${view}`);
    page.classList.toggle("active", page.id === `business-page-${view}`);
  });
  elements.businessContext.textContent = ctx.businessContextText();
  ctx.renderNamespacePageSummary();
  ctx.renderWorkloadPageSummary();
  ctx.renderWorkloadDetailPageSummary();
  ctx.renderWorkloadDetailPanel();
  ctx.renderResourcePageSummary();
  ctx.renderResources();
  ctx.renderResourceExplorerPanel();
  ctx.renderClusterDetailPanel();
}

export async function openClusterNamespaces(cluster) {
  state.business.selectedClusterID = cluster.id;
  state.business.selectedClusterName = cluster.name;
  state.business.clusterListPage = "list";
  state.business.selectedNamespace = "";
  state.business.selectedWorkloadID = null;
  state.business.selectedResourceName = "";
  state.business.workloadListPage = "list";
  state.business.resourceListPage = "list";
  state.business.workloadInspectorView = "overview";
  ctx.resetWorkloadInspector();
  ctx.resetResourceExplorer();
  ctx.renderClusterSelectors();
  await Promise.all([ctx.loadClusterOverview(cluster.id), ctx.loadNamespaces(cluster.id), ctx.loadWorkloads(cluster.id)]);
  switchBusinessView("namespaces");
}

export async function openClusterOverview(cluster) {
  state.business.selectedClusterID = cluster.id;
  state.business.selectedClusterName = cluster.name;
  state.business.clusterListPage = "detail";
  state.business.selectedNamespace = "";
  state.business.selectedWorkloadID = null;
  state.business.selectedResourceName = "";
  state.business.workloadListPage = "list";
  state.business.resourceListPage = "list";
  ctx.resetWorkloadInspector();
  ctx.resetResourceExplorer();
  ctx.renderClusterSelectors();
  switchBusinessView("clusters");
  ctx.switchClusterPage("detail");
  await ctx.loadClusterOverview(cluster.id);
}

export async function openNamespaceWorkloads(namespace) {
  state.business.selectedClusterID = namespace.cluster_id;
  state.business.selectedClusterName = namespace.cluster_name;
  state.business.selectedNamespace = namespace.name;
  state.business.selectedWorkloadID = null;
  state.business.selectedResourceName = "";
  state.business.workloadListPage = "list";
  state.business.resourceListPage = "list";
  state.business.workloadInspectorView = "overview";
  ctx.resetWorkloadInspector();
  ctx.resetResourceExplorer();
  state.pagination.workloads.page = 1;
  ctx.renderClusterSelectors();
  await ctx.loadWorkloads(namespace.cluster_id, namespace.name);
  switchBusinessView("workloads");
}

export async function openNamespaceResources(namespace) {
  state.business.selectedClusterID = namespace.cluster_id;
  state.business.selectedClusterName = namespace.cluster_name;
  state.business.selectedNamespace = namespace.name;
  state.business.selectedResourceName = "";
  state.business.resourceListPage = "list";
  state.pagination.resources.page = 1;
  ctx.resetResourceExplorer();
  ctx.renderClusterSelectors();
  await ctx.loadResources(namespace.cluster_id, namespace.name, ctx.selectedResourceKind());
  switchBusinessView("resources");
}

export async function openNamespaceGovernance(namespace, kind) {
  state.business.selectedResourceKind = kind;
  elements.resourceKindFilter.value = kind;
  await openNamespaceResources(namespace);
}

export function openClusterEditModal(cluster) {
  openFormModal({
    eyebrow: "Cluster",
    title: `编辑集群 ${cluster.name}`,
    copy: "凭据留空时保留现有配置。",
    fields: [
      { label: "集群名称", name: "name", value: cluster.name, required: true },
      { label: "集群编码", name: "code", value: cluster.code, required: true },
      { label: "环境", name: "environment", value: cluster.environment, required: true },
      { label: "供应商", name: "provider", value: cluster.provider, required: true },
      { label: "API 地址", name: "api_endpoint", value: cluster.api_endpoint, required: true },
      { label: "认证方式", name: "auth_type", type: "select", value: cluster.auth_type, options: [{ value: "token", label: "ServiceAccount Token" }, { value: "kubeconfig", label: "Kubeconfig" }] },
      { label: "凭据内容", name: "credential", type: "textarea", placeholder: "留空则保留原配置。EKS kubeconfig 可直接保留 aws eks get-token 的 exec 配置。" },
      { label: "说明", name: "description", value: cluster.description || "" },
    ],
    onSubmit: async (form) => {
      await api(`/api/v1/k8s/clusters/${cluster.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: form.get("name"),
          code: form.get("code"),
          environment: form.get("environment"),
          provider: form.get("provider"),
          api_endpoint: form.get("api_endpoint"),
          auth_type: form.get("auth_type"),
          credential: form.get("credential"),
          description: form.get("description"),
        }),
      });
      toast("集群已更新");
      await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID())]);
    },
  });
}

export function openNamespaceEditModal(namespace) {
  openFormModal({
    eyebrow: "Namespace",
    title: `编辑命名空间 ${namespace.name}`,
    fields: [
      { label: "所属集群", name: "cluster_id", type: "select", value: String(namespace.cluster_id), options: state.clusters.map((item) => ({ value: String(item.id), label: item.name })) },
      { label: "Namespace 名称", name: "name", value: namespace.name, required: true },
      { label: "显示名", name: "display_name", value: namespace.display_name || "" },
      { label: "说明", name: "description", value: namespace.description || "" },
      { label: "状态", name: "status", type: "select", value: namespace.status, options: [{ value: "active", label: "启用" }, { value: "disabled", label: "禁用" }] },
    ],
    onSubmit: async (form) => {
      await api(`/api/v1/k8s/namespaces/${namespace.id}`, {
        method: "PUT",
        body: JSON.stringify({
          cluster_id: Number(form.get("cluster_id")),
          name: form.get("name"),
          display_name: form.get("display_name"),
          description: form.get("description"),
          source_type: namespace.source_type,
          status: form.get("status"),
        }),
      });
      toast("命名空间已更新");
      await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID())]);
    },
  });
}

export async function testCluster(cluster) {
  const payload = await api(`/api/v1/k8s/clusters/${cluster.id}/test`, { method: "POST" });
  toast(payload.data.server_version ? `检测成功，版本 ${payload.data.server_version}` : (payload.data.message || "集群检测完成"));
  await ctx.loadClusters();
}

export async function syncClusterNamespaces(cluster) {
  const payload = await api(`/api/v1/k8s/clusters/${cluster.id}/sync-namespaces`, { method: "POST" });
  toast(payload.data.message ? `${payload.data.message}，共 ${payload.data.synced_namespaces} 个` : "命名空间同步完成");
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID())]);
}

export async function syncSelectedClusterWorkloads() {
  const clusterID = ctx.selectedWorkloadClusterID() || ctx.selectedClusterID();
  if (!clusterID) {
    await showErrorDialog({ title: "无法同步", copy: "请先选择一个集群" });
    return;
  }
  const payload = await api(`/api/v1/k8s/clusters/${clusterID}/sync-workloads`, { method: "POST" });
  toast(payload.data.message ? `${payload.data.message}，共 ${payload.data.synced_workloads} 个` : "工作负载同步完成");
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
}

export async function refreshNamespacesView() {
  const clusterID = ctx.selectedClusterID();
  if (clusterID) {
    await api(`/api/v1/k8s/clusters/${clusterID}/sync-namespaces`, { method: "POST" });
  }
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(clusterID)]);
  toast(clusterID ? "命名空间已从集群重新同步" : "命名空间列表已刷新");
}

export async function refreshWorkloadsView() {
  const clusterID = ctx.selectedWorkloadClusterID() || ctx.selectedClusterID();
  if (!clusterID) {
    await showErrorDialog({ title: "无法刷新", copy: "请先选择一个集群" });
    return;
  }
  await api(`/api/v1/k8s/clusters/${clusterID}/sync-workloads`, { method: "POST" });
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
  toast("工作负载已从集群重新同步");
}

export async function refreshBusinessOverview() {
  const namespaceClusterID = ctx.selectedClusterID();
  const workloadClusterID = ctx.selectedWorkloadClusterID();
  const clusterID = workloadClusterID || namespaceClusterID;
  if (clusterID) {
    await Promise.all([
      api(`/api/v1/k8s/clusters/${clusterID}/sync-namespaces`, { method: "POST" }),
      api(`/api/v1/k8s/clusters/${clusterID}/sync-workloads`, { method: "POST" }),
    ]);
  }
  await Promise.all([
    ctx.loadClusters(),
    ctx.loadNamespaces(namespaceClusterID),
    ctx.loadWorkloads(workloadClusterID, ctx.selectedWorkloadNamespace()),
    ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind()),
  ]);
  toast(clusterID ? "业务数据已重新同步" : "业务视图已刷新");
}

export async function deleteCluster(cluster) {
  if (!(await confirmAction({ eyebrow: "Cluster", title: "删除集群", copy: `确认删除集群 ${cluster.name} ? 该集群下的平台命名空间记录也会一起删除。` }))) return;
  await api(`/api/v1/k8s/clusters/${cluster.id}`, { method: "DELETE" });
  toast("集群已删除");
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID()), ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace()), ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind())]);
}

export async function deleteNamespace(namespace) {
  if (!(await confirmAction({ eyebrow: "Namespace", title: "删除命名空间", copy: `确认删除命名空间 ${namespace.name} ?` }))) return;
  await api(`/api/v1/k8s/namespaces/${namespace.id}`, { method: "DELETE" });
  toast("命名空间已删除");
  await Promise.all([ctx.loadClusters(), ctx.loadNamespaces(ctx.selectedClusterID()), ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace()), ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind())]);
}

export async function deleteWorkload(workload) {
  if (!workload?.id) {
    await showErrorDialog({ title: "删除失败", copy: "未找到可删除的工作负载" });
    return;
  }
  const ok = await confirmAction({
    eyebrow: workload.kind || "Workload",
    title: `删除 ${workload.name}`,
    copy: `确认删除 ${workload.kind || "工作负载"} ${workload.name} ? 删除后会同步刷新当前命名空间的工作负载列表。`,
    confirmText: "确认删除",
  });
  if (!ok) return;
  const payload = await api(`/api/v1/k8s/workloads/${workload.id}`, { method: "DELETE" });
  if (state.business.selectedWorkloadID === workload.id) {
    state.business.selectedWorkloadID = null;
    state.business.workloadListPage = "list";
    state.business.workloadInspectorView = "overview";
    ctx.resetWorkloadInspector();
  }
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
  if (ctx.selectedResourceClusterID() && ctx.selectedResourceNamespace() && ctx.selectedResourceKind()) {
    await ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind());
  }
  toast(payload.data.message ? `${payload.data.message}：${workload.name}` : "工作负载已删除");
}

export async function deleteNamespaceResource(kind, name) {
  const clusterID = ctx.selectedResourceClusterID() || ctx.selectedWorkloadClusterID() || ctx.selectedClusterID();
  const namespace = ctx.selectedResourceNamespace() || ctx.selectedWorkloadNamespace() || state.business.selectedNamespace;
  if (!clusterID || !namespace || !kind || !name) {
    await showErrorDialog({ title: "删除失败", copy: "当前资源定位信息不完整，无法删除" });
    return;
  }
  const ok = await confirmAction({
    eyebrow: kind,
    title: `删除 ${name}`,
    copy: `确认删除 ${kind} ${name} ? 删除后会刷新当前命名空间下的资源和工作负载视图。`,
    confirmText: "确认删除",
  });
  if (!ok) return;
  const params = new URLSearchParams({ cluster_id: String(clusterID), namespace, kind, name });
  const payload = await api(`/api/v1/k8s/resources/detail?${params.toString()}`, { method: "DELETE" });
  const deletingSelectedResource = state.business.selectedResourceName === name && (state.resourceExplorer.detail?.kind || ctx.selectedResourceKind()) === kind;
  if (deletingSelectedResource) {
    state.business.selectedResourceName = "";
    state.business.resourceListPage = "list";
    ctx.resetResourceExplorer();
  }
  const deletingLinkedDetail = state.workloadInspector.resourceDetail
    && state.workloadInspector.resourceDetail.name === name
    && ctx.normalizeResourceKindName(state.workloadInspector.resourceDetail.kind) === kind;
  if (deletingLinkedDetail) {
    state.workloadInspector.resourceDetail = null;
  }
  if (ctx.selectedWorkloadClusterID() && ctx.selectedWorkloadNamespace()) {
    await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
  }
  await ctx.loadResources(ctx.selectedResourceClusterID() || clusterID, ctx.selectedResourceNamespace() || namespace, ctx.selectedResourceKind());
  if (state.business.selectedWorkloadID && state.business.workloadInspectorView === "resources") {
    state.workloadInspector.resources = null;
    await ctx.switchWorkloadInspector("resources");
  } else {
    ctx.renderWorkloadDetailPanel();
  }
  toast(payload.data.message ? `${payload.data.message}：${name}` : "资源已删除");
}

export async function onWorkloadClusterChange() {
  state.business.selectedClusterID = ctx.selectedWorkloadClusterID() || null;
  const cluster = state.clusters.find((item) => item.id === state.business.selectedClusterID);
  state.business.selectedClusterName = cluster?.name || "";
  state.business.selectedNamespace = "";
  state.business.selectedWorkloadID = null;
  state.business.workloadListPage = "list";
  state.business.workloadInspectorView = "overview";
  ctx.resetWorkloadInspector();
  state.pagination.workloads.page = 1;
  ctx.renderWorkloadNamespaceOptions();
  elements.businessContext.textContent = ctx.businessContextText();
  if (!state.business.selectedClusterID) {
    await ctx.loadWorkloads(undefined, "");
    return;
  }
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
}

export async function onNamespaceClusterChange() {
  state.business.selectedClusterID = ctx.selectedClusterID() || null;
  const cluster = state.clusters.find((item) => item.id === state.business.selectedClusterID);
  state.business.selectedClusterName = cluster?.name || "";
  if (!state.business.selectedClusterID) {
    state.business.selectedNamespace = "";
  }
  state.business.selectedWorkloadID = null;
  state.business.selectedResourceName = "";
  state.business.workloadListPage = "list";
  state.business.resourceListPage = "list";
  state.business.workloadInspectorView = "overview";
  ctx.resetWorkloadInspector();
  ctx.resetResourceExplorer();
  elements.businessContext.textContent = ctx.businessContextText();
  await ctx.loadNamespaces(ctx.selectedClusterID());
}

export async function onWorkloadNamespaceChange() {
  state.business.selectedNamespace = ctx.selectedWorkloadNamespace();
  state.business.selectedWorkloadID = null;
  state.business.workloadListPage = "list";
  state.business.workloadInspectorView = "overview";
  ctx.resetWorkloadInspector();
  state.pagination.workloads.page = 1;
  elements.businessContext.textContent = ctx.businessContextText();
  await ctx.loadWorkloads(ctx.selectedWorkloadClusterID(), ctx.selectedWorkloadNamespace());
}

export async function onResourceClusterChange() {
  state.business.selectedClusterID = ctx.selectedResourceClusterID() || null;
  const cluster = state.clusters.find((item) => item.id === state.business.selectedClusterID);
  state.business.selectedClusterName = cluster?.name || "";
  state.business.selectedNamespace = "";
  state.business.selectedResourceName = "";
  state.business.resourceListPage = "list";
  state.pagination.resources.page = 1;
  ctx.resetResourceExplorer();
  ctx.renderResourceNamespaceOptions();
  elements.businessContext.textContent = ctx.businessContextText();
  if (!state.business.selectedClusterID) {
    await ctx.loadResources(undefined, "", ctx.selectedResourceKind());
    return;
  }
  await ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind());
}

export async function onResourceNamespaceChange() {
  state.business.selectedNamespace = ctx.selectedResourceNamespace();
  state.business.selectedResourceName = "";
  state.business.resourceListPage = "list";
  state.pagination.resources.page = 1;
  ctx.resetResourceExplorer();
  elements.businessContext.textContent = ctx.businessContextText();
  await ctx.loadResources(ctx.selectedResourceClusterID(), ctx.selectedResourceNamespace(), ctx.selectedResourceKind());
}
