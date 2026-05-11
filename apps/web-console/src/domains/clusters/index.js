// K8s frontend orchestrator.
// This file should only coordinate page-level flows, data loading, and module wiring.
// Detailed rendering, object actions, and creator logic should stay in ./k8s/* submodules.
import { api, buildWebSocketURL } from "../../core/api.js";
import { loadProjectCatalog } from "../delivery/api.js";
import { elements } from "../../core/dom.js";
import { configureClusterOverview, renderClusterDetailPanel, switchClusterPage } from "./cluster-overview.js";
import {
  configureK8sBusinessActions,
  deleteCluster,
  deleteNamespace,
  deleteNamespaceResource,
  deleteWorkload,
  onCreateCluster,
  onCreateNamespace,
  onNamespaceClusterChange,
  onResourceClusterChange,
  onResourceNamespaceChange,
  onWorkloadClusterChange,
  onWorkloadNamespaceChange,
  openClusterEditModal,
  openClusterNamespaces,
  openClusterOverview,
  openNamespaceEditModal,
  openNamespaceGovernance,
  openNamespaceResources,
  openNamespaceWorkloads,
  refreshBusinessOverview,
  refreshNamespacesView,
  refreshWorkloadsView,
  switchBusinessView,
  syncClusterNamespaces,
  syncSelectedClusterWorkloads,
  testCluster,
} from "./business-actions.js";
import { configureK8sDetailPanels, renderResourceExplorerPanel, switchResourcePage } from "./detail-panels.js";
import {
  configureK8sPageRenderers,
  normalizeInspectorView,
  renderClusterSelectors,
  renderClusters,
  renderNamespacePageSummary,
  renderNamespaces,
  renderResourceNamespaceOptions,
  renderResourcePageSummary,
  renderResources,
  renderWorkloadNamespaceOptions,
  renderWorkloadPageSummary,
  renderWorkloads,
  selectedBusinessCluster,
  selectedClusterID,
  selectedResourceClusterID,
  selectedResourceKind,
  selectedResourceNamespace,
  selectedWorkloadClusterID,
  selectedWorkloadNamespace,
} from "./page-renderers.js";
import {
  configureK8sSelectionActions,
  normalizeResourceKindName,
  openLinkedNamespaceResource,
  openReferencedWorkload,
  openResourceDetail,
  refreshSelectedResource,
  refreshSelectedWorkload,
  resetResourceExplorer,
  resetWorkloadInspector,
  selectResource,
  selectWorkload,
} from "./selection-actions.js";
import { configureWorkloadCreator, onWorkloadCreatorClick, onWorkloadCreatorInput, openCreateResourceDrawer, openCreateWorkloadDrawer, renderWorkloadCreator } from "./workload-creator.js";
import { ensureWorkloadCreatorDefaults } from "./workload-creator-state.js";
import { renderWorkloadDetailPanel, switchWorkloadPage } from "./workload-detail.js";
import { configureResourceEditing } from "./resource-edit.js";
import { configureWorkloadOps } from "./workload-ops.js";
import { state } from "../../core/state.js";
import { bindAction, bindSearch, closeDrawer, confirmAction, openDrawer, openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, findById, normalizeTerminalChunk } from "../../shared/utils.js";

let logFollowTimer = null;
let interactiveTerminalSocket = null;
let interactiveTerminalResizeTimer = null;
let k8sGraphLinkBound = false;

export async function loadK8sData() {
  elements.hideSystemNamespaces.checked = state.business.hideSystemNamespaces;
  syncResourceScopeButtons();
  if (!(state.cloud.projects || []).length || !(state.cloud.environments || []).length) {
    const catalog = await loadProjectCatalog();
    state.cloud.projects = catalog.projects || [];
    state.cloud.environments = catalog.environments || [];
    state.cloud.stacks = catalog.stacks || [];
  }
  await Promise.all([loadClusters(), loadNamespaces(), loadWorkloads()]);
  switchBusinessView(state.business.view);
}

export function bindK8sEvents() {
  if (!k8sGraphLinkBound) {
    k8sGraphLinkBound = true;
    window.addEventListener("bc:open-k8s-cluster", async (event) => {
      const clusterID = Number(event.detail?.clusterID || 0);
      if (!clusterID) return;
      try {
        await loadClusters();
        const cluster = state.clusters.find((item) => Number(item.id) === clusterID);
        if (!cluster) return;
        await openClusterOverview(cluster);
      } catch (_error) {
      }
    });
  }
  configureK8sBusinessActions({
    businessContextText,
    loadClusterOverview,
    loadClusters,
    loadNamespaces,
    loadResources,
    loadWorkloads,
    normalizeResourceKindName,
    renderClusterDetailPanel,
    renderClusterSelectors,
    renderNamespacePageSummary,
    renderWorkloadDetailPageSummary,
    renderResourceExplorerPanel,
    renderResourceNamespaceOptions,
    renderResourcePageSummary,
    renderResources,
    renderWorkloadDetailPanel,
    renderWorkloadNamespaceOptions,
    renderWorkloadPageSummary,
    resetResourceExplorer,
    resetWorkloadInspector,
    selectedClusterID,
    selectedResourceClusterID,
    selectedResourceKind,
    selectedResourceNamespace,
    selectedWorkloadClusterID,
    selectedWorkloadNamespace,
    stopLogFollow,
    switchClusterPage,
    switchWorkloadInspector,
  });
  configureK8sSelectionActions({
    loadResourceDetail,
    loadResources,
    loadWorkloadInspector,
    loadWorkloads,
    normalizeInspectorView,
    openNamespaceResources,
    renderClusterSelectors,
    renderResources,
    renderWorkloadDetailPanel,
    renderWorkloads,
    selectedResourceClusterID,
    selectedResourceKind,
    selectedResourceNamespace,
    selectedWorkloadClusterID,
    selectedWorkloadNamespace,
    setResourceKindFilter: (kind) => { elements.resourceKindFilter.value = kind; },
    switchBusinessView,
    switchResourcePage,
    switchWorkloadPage,
  });
  configureK8sPageRenderers({
    deleteCluster,
    deleteNamespace,
    deleteNamespaceResource,
    deleteWorkload,
    ensureWorkloadCreatorDefaults,
    openClusterEditModal,
    openClusterNamespaces,
    openClusterOverview,
    openNamespaceGovernance,
    openNamespaceEditModal,
    openNamespaceResources,
    openNamespaceWorkloads,
    renderClusterDetailPanel,
    renderDetailCard,
    renderResourceExplorerPanel,
    renderWorkloadCreator,
    renderWorkloadDetailPanel,
    runK8sAction,
    selectResource,
    selectWorkload,
    syncClusterNamespaces,
    testCluster,
  });
  configureClusterOverview({
    businessContextText,
    openNamespaceResources,
    openNamespaceWorkloads,
    renderDetailCard,
    runK8sAction,
  });
  configureK8sDetailPanels({
    closeInteractiveTerminal,
    deleteNamespaceResource,
    deleteSelectedWorkload,
    openLinkedNamespaceResource,
    openReferencedWorkload,
    openResourceDetail,
    renderDetailCard,
    runK8sAction,
    sendInteractiveTerminalInput,
    toggleLogFollow,
    refreshWorkloadLogs,
    switchManifestMode,
    switchResourceManifestMode,
    switchLogPod,
    switchWorkloadInspector,
  });
  configureResourceEditing({
    loadResources,
    loadResourceDetail,
    selectedResourceClusterID,
    selectedResourceNamespace,
    selectedResourceKind,
    switchWorkloadInspector,
  });
  configureWorkloadOps({
    openInteractiveTerminal,
    renderWorkloadDetailPanel,
    refreshSelectedWorkload,
    selectedWorkloadClusterID,
  });
  configureWorkloadCreator({
    loadClusters,
    loadNamespaces,
    loadResources,
    loadWorkloads,
    renderClusterSelectors,
    runK8sAction,
    selectResource,
    selectWorkload,
    selectedClusterID,
    selectedResourceKind,
    selectedResourceNamespace,
    selectedWorkloadClusterID,
    selectedWorkloadNamespace,
    switchBusinessView,
  });
  elements.createClusterForm.addEventListener("submit", (event) => runK8sAction(() => onCreateCluster(event)));
  elements.createNamespaceForm.addEventListener("submit", (event) => runK8sAction(() => onCreateNamespace(event)));
  elements.openCreateCluster.addEventListener("click", () => openDrawer("cluster"));
  elements.businessOpenCreateCluster.addEventListener("click", () => openDrawer("cluster"));
  elements.openCreateNamespace.addEventListener("click", () => openDrawer("namespace"));
  elements.openCreateWorkload.addEventListener("click", openCreateWorkloadDrawer);
  elements.openCreateResource?.addEventListener("click", openCreateResourceDrawer);
  document.getElementById("refresh-clusters").addEventListener("click", () => runK8sAction(loadClusters));
  document.getElementById("refresh-namespaces").addEventListener("click", () => runK8sAction(refreshNamespacesView));
  document.getElementById("refresh-workloads").addEventListener("click", () => runK8sAction(refreshWorkloadsView));
  document.getElementById("refresh-resources").addEventListener("click", () => runK8sAction(() => loadResources(selectedResourceClusterID(), selectedResourceNamespace(), selectedResourceKind())));
  document.getElementById("sync-workloads").addEventListener("click", () => runK8sAction(syncSelectedClusterWorkloads));
  elements.businessRefreshAll.addEventListener("click", () => runK8sAction(refreshBusinessOverview));
  window.addEventListener("resize", () => {
    if (interactiveTerminalResizeTimer) {
      window.clearTimeout(interactiveTerminalResizeTimer);
    }
    interactiveTerminalResizeTimer = window.setTimeout(() => {
      sendInteractiveTerminalResize().catch(() => {});
    }, 120);
  });
  elements.hideSystemNamespaces.addEventListener("change", () => {
    state.business.hideSystemNamespaces = elements.hideSystemNamespaces.checked;
    if (state.business.resourceScope === "all") {
      syncResourceScopeButtons();
    }
    renderClusterSelectors();
    renderSummary();
    renderNamespaces();
    renderWorkloads();
    renderResources();
  });
  elements.resourceScopeAll.addEventListener("click", () => {
    state.business.resourceScope = "all";
    syncResourceScopeButtons();
    renderResources();
  });
  elements.resourceScopeBusiness.addEventListener("click", () => {
    state.business.resourceScope = "business";
    syncResourceScopeButtons();
    renderResources();
  });
  elements.resourceScopeSystem.addEventListener("click", () => {
    state.business.resourceScope = "system";
    syncResourceScopeButtons();
    renderResources();
  });
  elements.namespaceClusterFilter.addEventListener("change", () => runK8sAction(onNamespaceClusterChange));
  elements.workloadClusterFilter.addEventListener("change", () => runK8sAction(onWorkloadClusterChange));
  elements.workloadNamespaceFilter.addEventListener("change", () => runK8sAction(onWorkloadNamespaceChange));
  elements.resourceClusterFilter.addEventListener("change", () => runK8sAction(onResourceClusterChange));
  elements.resourceNamespaceFilter.addEventListener("change", () => runK8sAction(onResourceNamespaceChange));
  elements.resourceKindFilter.addEventListener("change", () => {
    state.business.selectedResourceKind = elements.resourceKindFilter.value;
    state.pagination.resources.page = 1;
    state.business.resourceListPage = "list";
    resetResourceExplorer();
    runK8sAction(() => loadResources(selectedResourceClusterID(), selectedResourceNamespace(), selectedResourceKind()));
  });
  elements.workloadKindFilter.addEventListener("change", () => {
    state.business.selectedWorkloadKind = elements.workloadKindFilter.value;
    state.pagination.workloads.page = 1;
    renderWorkloads();
  });
  elements.workloadDetailBack.addEventListener("click", () => {
    stopLogFollow();
    closeInteractiveTerminal(true);
    state.workloadInspector.followLogs = false;
    switchBusinessView("workloads");
    switchWorkloadPage("list");
  });
  elements.resourceDetailBack.addEventListener("click", () => switchResourcePage("list"));
  elements.clusterDetailBack.addEventListener("click", () => switchClusterPage("list"));
  elements.businessViewButtons.forEach((button) => {
    button.addEventListener("click", () => switchBusinessView(button.dataset.businessView));
  });
  bindSearch(elements.searchClusters, (value) => {
    state.filters.clusters = value;
    renderClusters();
  });
  bindSearch(elements.searchNamespaces, (value) => {
    state.filters.namespaces = value;
    renderNamespaces();
  });
  bindSearch(elements.searchWorkloads, (value) => {
    state.filters.workloads = value;
    state.pagination.workloads.page = 1;
    renderWorkloads();
  });
  bindSearch(elements.searchResources, (value) => {
    state.filters.resources = value;
    state.pagination.resources.page = 1;
    renderResources();
  });
  elements.workloadCreatorRoot.addEventListener("click", onWorkloadCreatorClick);
  elements.workloadCreatorRoot.addEventListener("input", onWorkloadCreatorInput);
  elements.workloadCreatorRoot.addEventListener("change", onWorkloadCreatorInput);
}

async function loadClusters() {
  const payload = await api("/api/v1/k8s/clusters");
  state.clusters = payload.data || [];
  renderClusterSelectors();
  renderSummary();
  renderClusters();
  if (state.business.selectedClusterID && state.business.clusterListPage === "detail") {
    await loadClusterOverview(state.business.selectedClusterID);
  }
}

async function loadNamespaces(clusterID) {
  const url = clusterID ? `/api/v1/k8s/namespaces?cluster_id=${clusterID}` : "/api/v1/k8s/namespaces";
  const payload = await api(url);
  state.namespaces = payload.data || [];
  renderSummary();
  renderNamespaces();
  renderWorkloadNamespaceOptions();
}

async function loadWorkloads(clusterID, namespace) {
  const params = new URLSearchParams();
  if (clusterID) params.set("cluster_id", clusterID);
  if (namespace) params.set("namespace", namespace);
  params.set("include_pods", "false");
  const query = params.toString();
  const payload = await api(`/api/v1/k8s/workloads${query ? `?${query}` : ""}`);
  state.workloads = payload.data || [];
  if (!state.workloads.length) {
    stopLogFollow();
    closeInteractiveTerminal(true);
    state.workloadInspector.detail = null;
    state.workloadInspector.rollout = null;
    state.workloadInspector.resources = null;
    state.workloadInspector.resourceDetail = null;
    state.workloadInspector.pods = [];
    state.workloadInspector.events = [];
    state.workloadInspector.logs = "";
    state.workloadInspector.logSourcePod = "";
    state.workloadInspector.manifest = "";
    state.workloadInspector.interactiveTerminal = null;
    state.workloadInspector.interactiveTerminalOutput = "";
    state.workloadInspector.interactiveTerminalConnected = false;
    state.workloadInspector.interactiveTerminalError = "";
  }
  renderWorkloads();
  renderWorkloadDetailPageSummary();
  if (state.business.selectedWorkloadID && state.workloadInspector.detail?.id !== state.business.selectedWorkloadID) {
    await loadWorkloadInspector(state.business.selectedWorkloadID);
  }
}

async function loadResources(clusterID, namespace, kind) {
  if (!clusterID || !namespace || !kind) {
    state.resourceExplorer.items = [];
    resetResourceExplorer();
    renderResources();
    return;
  }
  const params = new URLSearchParams({ cluster_id: String(clusterID), namespace, kind });
  const payload = await api(`/api/v1/k8s/resources?${params.toString()}`);
  state.resourceExplorer.items = payload.data.items || [];
  renderResources();
  if (state.business.selectedResourceName && (!state.resourceExplorer.detail || state.resourceExplorer.detail.name !== state.business.selectedResourceName || state.resourceExplorer.detail.kind !== kind)) {
    await loadResourceDetail(state.business.selectedResourceName);
  }
}

async function loadWorkloadInspector(id) {
  stopLogFollow();
  closeInteractiveTerminal(true);
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
  state.workloadInspector.interactiveTerminal = null;
  state.workloadInspector.interactiveTerminalOutput = "";
  state.workloadInspector.interactiveTerminalConnected = false;
  state.workloadInspector.interactiveTerminalError = "";
  const [detail, pods, events] = await Promise.all([
    api(`/api/v1/k8s/workloads/${id}`),
    api(`/api/v1/k8s/workloads/${id}/pods`),
    api(`/api/v1/k8s/workloads/${id}/events`).catch(() => ({ data: { events: [] } })),
  ]);
  state.workloadInspector.detail = detail.data;
  state.workloadInspector.pods = pods.data.pods || [];
  state.workloadInspector.events = events.data.events || [];
  state.workloadInspector.selectedPodName = state.workloadInspector.pods[0]?.name || "";
  elements.workloadDetailTitle.textContent = `${detail.data.kind} / ${detail.data.name}`;
  renderWorkloadDetailPageSummary();
  renderWorkloadDetailPanel();
}

async function loadResourceDetail(name) {
  const clusterID = selectedResourceClusterID();
  const namespace = selectedResourceNamespace();
  const kind = selectedResourceKind();
  if (!clusterID || !namespace || !kind || !name) return;
  state.resourceExplorer.detail = null;
  state.resourceExplorer.manifest = "";
  const params = new URLSearchParams({ cluster_id: String(clusterID), namespace, kind, name });
  const payload = await api(`/api/v1/k8s/resources/detail?${params.toString()}`);
  state.resourceExplorer.detail = { kind, name, data: payload.data.detail };
  elements.resourceDetailTitle.textContent = `${kind} / ${name}`;
  renderResourceExplorerPanel();
}

async function loadClusterOverview(id) {
  state.clusterInspector.overview = null;
  const payload = await api(`/api/v1/k8s/clusters/${id}/overview`);
  state.clusterInspector.overview = payload.data;
  elements.clusterDetailTitle.textContent = `${payload.data.cluster.name} / 集群概览`;
  renderClusterDetailPanel();
}

function businessContextText() {
  if (state.business.view === "clusters") {
    if (state.business.clusterListPage === "detail" && state.business.selectedClusterName) {
      return `当前页：集群概览 / ${state.business.selectedClusterName}`;
    }
    return "当前页：集群列表";
  }
  if (state.business.view === "namespaces") {
    const clusterName = state.business.selectedClusterName || "全部集群";
    return `当前页：命名空间 / ${clusterName}`;
  }
  if (state.business.view === "workload-detail") {
    const clusterName = state.business.selectedClusterName || "全部集群";
    const namespaceName = selectedWorkloadNamespace() || state.business.selectedNamespace || "全部命名空间";
    const workload = state.workloads.find((item) => item.id === state.business.selectedWorkloadID) || state.workloadInspector.detail;
    const workloadName = workload ? `${workload.kind || "Workload"} / ${workload.name || "-"}` : "服务详情";
    return `当前页：服务详情 / ${clusterName} / ${namespaceName} / ${workloadName}`;
  }
  if (state.business.view === "resources") {
    const clusterName = state.business.selectedClusterName || "全部集群";
    const namespaceName = selectedResourceNamespace() || state.business.selectedNamespace || "全部命名空间";
    return `当前页：资源浏览 / ${clusterName} / ${namespaceName} / ${selectedResourceKind()}`;
  }
  const clusterName = state.business.selectedClusterName || "全部集群";
  const namespaceName = selectedWorkloadNamespace() || state.business.selectedNamespace || "全部命名空间";
  return `当前页：工作负载 / ${clusterName} / ${namespaceName}`;
}

function renderDetailCard(item) {
  const label = escapeHtml(String(item?.label ?? "-"));
  const value = escapeHtml(String(item?.value ?? "-"));
  return `<article class="cluster-meta-card"><span class="muted-label">${label}</span><strong>${value}</strong></article>`;
}

function renderSummary() {
  if (!elements.k8sSummaryCards) return;
  const selectedClusterIDValue = state.business.selectedClusterID;
  const currentNamespace = state.business.view === "resources"
    ? (selectedResourceNamespace() || state.business.selectedNamespace || "")
    : (selectedWorkloadNamespace() || state.business.selectedNamespace || "");
  let visibleNamespaces = selectedClusterIDValue
    ? state.namespaces.filter((item) => item.cluster_id === selectedClusterIDValue)
    : state.namespaces;
  if (state.business.hideSystemNamespaces) {
    visibleNamespaces = visibleNamespaces.filter((item) => !String(item.name || "").startsWith("kube-"));
  }
  let visibleWorkloads = currentNamespace
    ? state.workloads.filter((item) => item.namespace_name === currentNamespace)
    : state.workloads;
  if (state.business.hideSystemNamespaces) {
    visibleWorkloads = visibleWorkloads.filter((item) => !String(item.namespace_name || "").startsWith("kube-"));
  }
  const readyClusters = state.clusters.filter((item) => item.status === "ready").length;
  const activeNamespaces = visibleNamespaces.filter((item) => item.status === "active").length;
  const systemNamespaces = (selectedClusterIDValue
    ? state.namespaces.filter((item) => item.cluster_id === selectedClusterIDValue)
    : state.namespaces).filter((item) => String(item.name || "").startsWith("kube-")).length;
  const controllerKinds = new Set(["Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"]);
  const controllerCount = visibleWorkloads.filter((item) => controllerKinds.has(item.kind)).length;
  const unhealthyWorkloads = visibleWorkloads.filter((item) => {
    const desired = Number(item.replicas || 0);
    const ready = Number(item.ready_replicas || 0);
    return desired > 0 && ready < desired;
  }).length;
  const cards = [
    { label: "集群", value: `${readyClusters}/${state.clusters.length || 0}`, meta: "ready / total" },
    { label: "命名空间", value: `${activeNamespaces}/${visibleNamespaces.length || 0}`, meta: "active / visible" },
    { label: "工作负载", value: String(controllerCount), meta: "controllers" },
    { label: "待关注对象", value: String(unhealthyWorkloads), meta: "replica gap" },
  ];
  elements.k8sSummaryCards.innerHTML = cards.map((item) => `<article class="cluster-meta-card"><span class="muted-label">${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong><span class="table-meta">${escapeHtml(item.meta)}</span></article>`).join("");
}

function renderWorkloadDetailPageSummary() {
  if (!elements.workloadDetailPageSummary) return;
  const workload = state.workloads.find((item) => item.id === state.business.selectedWorkloadID) || state.workloadInspector.detail;
  if (!workload) {
    elements.workloadDetailPageTitle.textContent = "服务详情";
    elements.workloadDetailPageCopy.textContent = "选择一个工作负载后，这里会集中展示该服务的完整信息。";
    elements.workloadDetailPageSummary.innerHTML = emptyState("当前还没有选中的服务");
    return;
  }
  const clusterName = state.business.selectedClusterName || state.workloadInspector.detail?.cluster_name || "-";
  const namespaceName = workload.namespace_name || workload.namespace || state.business.selectedNamespace || "-";
  elements.workloadDetailPageTitle.textContent = `${workload.kind} / ${workload.name}`;
  elements.workloadDetailPageCopy.textContent = "这里集中查看服务状态、Pod、日志、终端、发布和资源关系。";
  elements.workloadDetailPageSummary.innerHTML = [
    renderDetailCard({ label: "命名空间", value: namespaceName }),
    renderDetailCard({ label: "所属集群", value: clusterName }),
    renderDetailCard({ label: "副本", value: `${workload.ready_replicas ?? "-"} / ${workload.replicas ?? "-"}` }),
    renderDetailCard({ label: "状态", value: workload.status || "-" }),
  ].join("");
}

function syncResourceScopeButtons() {
  const scope = state.business.resourceScope || "all";
  elements.resourceScopeAll?.classList.toggle("active", scope === "all");
  elements.resourceScopeBusiness?.classList.toggle("active", scope === "business");
  elements.resourceScopeSystem?.classList.toggle("active", scope === "system");
}

async function deleteSelectedWorkload(workload) {
  await deleteWorkload(workload);
}

async function switchWorkloadInspector(view) {
  state.business.workloadInspectorView = view;
  if (view !== "logs") stopLogFollow();
  if (view !== "logs") closeInteractiveTerminal(true);
  const id = state.business.selectedWorkloadID;
  if (!id) {
    renderWorkloadDetailPanel();
    return;
  }
  if (view === "events" && !state.workloadInspector.events.length) {
    const payload = await api(`/api/v1/k8s/workloads/${id}/events`);
    state.workloadInspector.events = payload.data.events || [];
  }
  if (view === "rollout" && !state.workloadInspector.rollout) {
    const payload = await api(`/api/v1/k8s/workloads/${id}/rollout`);
    state.workloadInspector.rollout = payload.data;
  }
  if (view === "resources" && !state.workloadInspector.resources) {
    const payload = await api(`/api/v1/k8s/workloads/${id}/resources`);
    state.workloadInspector.resources = payload.data;
  }
  if (view === "logs" && !state.workloadInspector.logs) {
    await refreshWorkloadLogs();
    if (state.workloadInspector.followLogs) {
      startLogFollow();
    }
  }
  if (view === "manifest" && !state.workloadInspector.manifest) {
    const payload = await api(`/api/v1/k8s/workloads/${id}/manifest?mode=${state.business.workloadManifestMode}`);
    state.workloadInspector.manifest = payload.data.manifest_yaml || "";
  }
  renderWorkloadDetailPanel();
}

async function switchManifestMode(mode) {
  state.business.workloadManifestMode = mode;
  state.workloadInspector.manifest = "";
  await switchWorkloadInspector("manifest");
}

async function switchResourceManifestMode(mode) {
  state.business.resourceManifestMode = mode;
  state.resourceExplorer.manifest = "";
  await loadResourceManifest();
  renderResourceExplorerPanel();
}

async function switchLogPod(podName) {
  state.workloadInspector.selectedPodName = podName;
  state.workloadInspector.logs = "";
  await switchWorkloadInspector("logs");
}

async function refreshWorkloadLogs() {
  const id = state.business.selectedWorkloadID;
  if (!id) return;
  const params = new URLSearchParams({ tail_lines: "200" });
  if (state.workloadInspector.selectedPodName) {
    params.set("pod_name", state.workloadInspector.selectedPodName);
  }
  const payload = await api(`/api/v1/k8s/workloads/${id}/logs?${params.toString()}`);
  state.workloadInspector.logs = payload.data.logs || "";
  state.workloadInspector.logSourcePod = payload.data.source_pod || "";
  state.workloadInspector.selectedPodName = payload.data.source_pod || state.workloadInspector.selectedPodName;
  state.workloadInspector.pods = payload.data.pods || state.workloadInspector.pods;
  renderWorkloadDetailPanel();
}

function stopLogFollow() {
  if (logFollowTimer) {
    window.clearInterval(logFollowTimer);
    logFollowTimer = null;
  }
}

function startLogFollow() {
  stopLogFollow();
  if (!state.workloadInspector.followLogs || state.business.workloadInspectorView !== "logs") return;
  const intervalMs = Math.max(2, Number(state.workloadInspector.followIntervalSeconds || 5)) * 1000;
  logFollowTimer = window.setInterval(() => {
    refreshWorkloadLogs().catch(async (_error) => {
      stopLogFollow();
      state.workloadInspector.followLogs = false;
      await showErrorDialog({ title: "日志刷新失败", copy: "日志自动跟随已停止，日志刷新失败" });
      renderWorkloadDetailPanel();
    });
  }, intervalMs);
}

async function toggleLogFollow() {
  state.workloadInspector.followLogs = !state.workloadInspector.followLogs;
  if (state.workloadInspector.followLogs) {
    await refreshWorkloadLogs();
    startLogFollow();
    toast(`已开启日志自动跟随，每 ${state.workloadInspector.followIntervalSeconds} 秒刷新一次`);
    return;
  }
  stopLogFollow();
  toast("已关闭日志自动跟随");
  renderWorkloadDetailPanel();
}

async function openInteractiveTerminal({ workloadID, podName, containerName, shell, timeoutSeconds }) {
  closeInteractiveTerminal(true);
  const screen = elements.workloadDetailPanel.querySelector("#interactive-terminal-screen");
  const rect = screen?.getBoundingClientRect();
  const cols = Math.max(80, Math.floor((rect?.width || 960) / 8));
  const rows = Math.max(20, Math.floor((rect?.height || 360) / 18));
  const wsURL = buildWebSocketURL(`/api/v1/k8s/workloads/${workloadID}/terminal`, {
    pod_name: podName,
    container_name: containerName,
    shell,
    timeout_seconds: timeoutSeconds,
    cols,
    rows,
  });
  state.workloadInspector.interactiveTerminal = {
    workload_id: workloadID,
    source_pod: podName,
    container_name: containerName,
    shell,
    cols,
    rows,
  };
  state.workloadInspector.interactiveTerminalOutput = "";
  state.workloadInspector.interactiveTerminalConnected = false;
  state.workloadInspector.interactiveTerminalError = "";
  state.workloadInspector.interactiveTerminalLastEvent = "opening";
  state.workloadInspector.interactiveTerminalOutputBytes = 0;
  renderWorkloadDetailPanel();
  const socket = new WebSocket(wsURL, ["bc-terminal", state.token]);
  interactiveTerminalSocket = socket;
  socket.addEventListener("open", () => {
    state.workloadInspector.interactiveTerminalConnected = true;
    state.workloadInspector.interactiveTerminalError = "";
    state.workloadInspector.interactiveTerminalLastEvent = "ws-open";
    renderWorkloadDetailPanel();
    elements.workloadDetailPanel.querySelector("#interactive-terminal-screen")?.focus();
  });
  socket.addEventListener("message", (event) => {
    try {
      const message = JSON.parse(String(event.data || "{}"));
      if (message.type === "output") {
        state.workloadInspector.interactiveTerminalOutput += normalizeTerminalChunk(message.data || "");
        state.workloadInspector.interactiveTerminalOutputBytes += String(message.data || "").length;
        state.workloadInspector.interactiveTerminalLastEvent = "output";
        syncInteractiveTerminalView();
        return;
      } else if (message.type === "status") {
        if (message.status === "connected" && state.workloadInspector.interactiveTerminal) {
          state.workloadInspector.interactiveTerminal = {
            ...state.workloadInspector.interactiveTerminal,
            ...message,
          };
          state.workloadInspector.interactiveTerminalConnected = true;
          state.workloadInspector.interactiveTerminalLastEvent = "status-connected";
        } else if (message.status === "closed") {
          state.workloadInspector.interactiveTerminalConnected = false;
          state.workloadInspector.interactiveTerminalError = message.reason || "";
          state.workloadInspector.interactiveTerminalLastEvent = "status-closed";
        }
      } else if (message.type === "error") {
        state.workloadInspector.interactiveTerminalError = message.message || "终端连接失败";
        state.workloadInspector.interactiveTerminalLastEvent = "status-error";
      } else if (message.type === "pong") {
        state.workloadInspector.interactiveTerminalLastEvent = "pong";
        return;
      }
    } catch (_error) {
      state.workloadInspector.interactiveTerminalOutput += normalizeTerminalChunk(event.data || "");
      state.workloadInspector.interactiveTerminalOutputBytes += String(event.data || "").length;
      state.workloadInspector.interactiveTerminalLastEvent = "raw-output";
      syncInteractiveTerminalView();
      return;
    }
    renderWorkloadDetailPanel();
  });
  socket.addEventListener("close", () => {
    state.workloadInspector.interactiveTerminalConnected = false;
    state.workloadInspector.interactiveTerminalLastEvent = "ws-close";
    interactiveTerminalSocket = null;
    renderWorkloadDetailPanel();
  });
  socket.addEventListener("error", () => {
    state.workloadInspector.interactiveTerminalConnected = false;
    state.workloadInspector.interactiveTerminalError = "终端连接失败";
    state.workloadInspector.interactiveTerminalLastEvent = "ws-error";
    renderWorkloadDetailPanel();
  });
}

async function sendInteractiveTerminalInput(data) {
  if (!interactiveTerminalSocket || interactiveTerminalSocket.readyState !== WebSocket.OPEN) {
    throw new Error("终端连接已断开");
  }
  interactiveTerminalSocket.send(JSON.stringify({ type: "input", data }));
  state.workloadInspector.interactiveTerminalLastEvent = "input-sent";
}

async function sendInteractiveTerminalResize() {
  if (!interactiveTerminalSocket || interactiveTerminalSocket.readyState !== WebSocket.OPEN) {
    return;
  }
  const screen = elements.workloadDetailPanel.querySelector("#interactive-terminal-screen");
  const rect = screen?.getBoundingClientRect();
  if (!rect) return;
  const cols = Math.max(80, Math.floor(rect.width / 8));
  const rows = Math.max(20, Math.floor(rect.height / 18));
  const terminal = state.workloadInspector.interactiveTerminal;
  if (terminal && terminal.cols === cols && terminal.rows === rows) return;
  if (terminal) {
    terminal.cols = cols;
    terminal.rows = rows;
  }
  interactiveTerminalSocket.send(JSON.stringify({ type: "resize", cols, rows }));
}

async function closeInteractiveTerminal(silent = false) {
  if (interactiveTerminalSocket) {
    const socket = interactiveTerminalSocket;
    interactiveTerminalSocket = null;
    if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
      socket.close();
    }
  }
  state.workloadInspector.interactiveTerminalConnected = false;
  state.workloadInspector.interactiveTerminal = null;
  state.workloadInspector.interactiveTerminalOutput = "";
  state.workloadInspector.interactiveTerminalError = "";
  state.workloadInspector.interactiveTerminalLastEvent = "";
  state.workloadInspector.interactiveTerminalOutputBytes = 0;
  if (!silent) {
    renderWorkloadDetailPanel();
  }
}

function syncInteractiveTerminalView() {
  const output = elements.workloadDetailPanel.querySelector(".interactive-terminal-output");
  const screen = elements.workloadDetailPanel.querySelector("#interactive-terminal-screen");
  if (!output || !screen) {
    renderWorkloadDetailPanel();
    return;
  }
  output.textContent = state.workloadInspector.interactiveTerminalOutput || "终端已建立，等待输入…";
  screen.scrollTop = screen.scrollHeight;
}

async function loadResourceManifest() {
  const clusterID = selectedResourceClusterID();
  const namespace = selectedResourceNamespace();
  const kind = selectedResourceKind();
  const name = state.business.selectedResourceName;
  if (!clusterID || !namespace || !kind || !name) return;
  const params = new URLSearchParams({ cluster_id: String(clusterID), namespace, kind, name, mode: state.business.resourceManifestMode });
  const payload = await api(`/api/v1/k8s/resources/manifest?${params.toString()}`);
  state.resourceExplorer.manifest = payload.data.manifest_yaml || "";
}

async function runK8sAction(action) {
  try {
    await action();
  } catch (error) {
    await showErrorDialog({ title: "Kubernetes 操作失败", copy: error?.message || "Kubernetes 操作失败" });
  }
}
