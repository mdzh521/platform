import { api } from "../../core/api.js";
import { loadProjectCatalog } from "../delivery/api.js";
import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { bindSearch, confirmAction, toast } from "../../core/ui.js";
import { resolveAssetLoginPolicy } from "./helpers.js";
import { openAssetDetail, renderMachineDetail, resetMachineDetailState, switchMachineView } from "./detail.js";
import { openAccountDetailModal, openAccountModal, openAssetGroupModal, openAssetModal, openBatchAssetCreateModal, openBatchAssetUpdateModal, openCredentialDetailModal, openCredentialLibraryModal, openQuickConnectModal } from "./modals.js";
import { bindMachineTerminalInteractions, closeMachineTerminal, openAssetAccess, renderMachineTerminalDock } from "./terminal.js";
import { renderMachineAssets, renderMachineAccessWorkspace, renderMachineCredentialLibrary, renderMachineGroups, renderMachineSessions, renderMachineSummary } from "./workspace.js";

let machineBound = false;
let machineSplitBound = false;
let machineGraphLinkBound = false;

const machineCtx = {
  openAccountModal: (asset) => openAccountModal(machineCtx, asset),
  openAssetDetail: (assetID, keepView = false) => openAssetDetail(machineCtx, assetID, keepView),
  openAssetGroupModal: (group = null) => openAssetGroupModal(machineCtx, group),
  openAssetModal: (asset = null) => openAssetModal(machineCtx, asset),
  openBatchAssetCreateModal: () => openBatchAssetCreateModal(machineCtx),
  openBatchAssetUpdateModal: (assetIDs = []) => openBatchAssetUpdateModal(machineCtx, assetIDs),
  openAccountDetailModal: (assetID, accountID) => openAccountDetailModal(machineCtx, assetID, accountID),
  openCredentialDetailModal: (credentialID) => openCredentialDetailModal(machineCtx, credentialID),
  openCredentialLibraryModal: () => openCredentialLibraryModal(machineCtx),
  openQuickConnectModal: (assetID = 0) => openQuickConnectModal(machineCtx, assetID),
  deleteAsset: (asset) => deleteMachineAsset(asset),
  refreshMachineData: (message = "") => refreshMachineData(message),
  renderMachineDetail: () => renderMachineDetail(machineCtx),
  renderMachineWorkspace: () => renderMachineWorkspace(),
  resolveAssetLoginPolicy,
};

export function bindMachineEvents() {
  if (machineBound) return;
  machineBound = true;
  if (!machineGraphLinkBound) {
    machineGraphLinkBound = true;
    window.addEventListener("bc:open-machine-asset", async (event) => {
      const assetID = Number(event.detail?.assetID || 0);
      if (!assetID) return;
      try {
        await loadMachineData();
        await openAssetDetail(machineCtx, assetID);
      } catch (_error) {
      }
    });
  }

  elements.machineCreateAsset?.addEventListener("click", () => machineCtx.openAssetModal());
  elements.machineCreateCredential?.addEventListener("click", () => machineCtx.openCredentialLibraryModal());
  elements.machineCreateAssetInline?.addEventListener("click", () => machineCtx.openAssetModal());
  elements.machineCreateGroup?.addEventListener("click", () => machineCtx.openAssetGroupModal());
  elements.machineQuickConnect?.addEventListener("click", () => machineCtx.openQuickConnectModal());
  elements.machineRefresh?.addEventListener("click", () => refreshMachineData("机器资产已刷新"));
  elements.machineRefreshInline?.addEventListener("click", () => refreshMachineData("机器资产已刷新"));
  elements.machineAccessRefresh?.addEventListener("click", () => refreshMachineData("机器资产已刷新"));
  elements.machineAccessOpenTerminalPage?.addEventListener("click", () => switchMachineView("terminal"));
  elements.machineDetailBack?.addEventListener("click", () => switchMachineView("list"));
  elements.machineNavList?.addEventListener("click", () => switchMachineView("list"));
  elements.machineNavAccess?.addEventListener("click", () => switchMachineView("access"));
  bindSearch(elements.machineSearchAssets, (value) => {
    state.machine.filters.search = value;
    state.machine.pagination.page = 1;
    refreshMachineData();
  });
  bindSearch(elements.machineAccessSearchAssets, (value) => {
    state.machine.filters.search = value;
    if (elements.machineSearchAssets) elements.machineSearchAssets.value = value;
    state.machine.pagination.page = 1;
    refreshMachineData();
  });
  elements.machineStatusFilter?.addEventListener("change", (event) => {
    state.machine.filters.status = String(event.target.value || "").trim().toLowerCase();
    if (elements.machineAccessStatusFilter) elements.machineAccessStatusFilter.value = state.machine.filters.status;
    state.machine.pagination.page = 1;
    refreshMachineData();
  });
  elements.machineAccessStatusFilter?.addEventListener("change", (event) => {
    state.machine.filters.status = String(event.target.value || "").trim().toLowerCase();
    if (elements.machineStatusFilter) elements.machineStatusFilter.value = state.machine.filters.status;
    state.machine.pagination.page = 1;
    refreshMachineData();
  });
  elements.machinePageSize?.addEventListener("change", (event) => {
    state.machine.pagination.pageSize = Number(event.target.value || 10) || 10;
    state.machine.pagination.page = 1;
    refreshMachineData();
  });
  elements.machineAssetsTable?.addEventListener("click", async (event) => {
    const sortButton = event.target.closest("[data-machine-sort]");
    if (sortButton) {
      const sortBy = String(sortButton.dataset.machineSort || "").trim();
      if (sortBy) {
        const sameField = state.machine.filters.sortBy === sortBy;
        state.machine.filters.sortBy = sortBy;
        state.machine.filters.sortOrder = sameField && state.machine.filters.sortOrder === "asc" ? "desc" : "asc";
        state.machine.pagination.page = 1;
        await refreshMachineData();
      }
      return;
    }
    const button = event.target.closest("[data-machine-action]");
    const batchButton = event.target.closest("[data-machine-batch-action]");
    const row = event.target.closest("[data-asset-row]");
    if (batchButton) {
      const action = String(batchButton.dataset.machineBatchAction || "");
      if (action === "create") {
        machineCtx.openBatchAssetCreateModal();
        return;
      }
      if (action === "clear") {
        state.machine.selectedAssetIDs = [];
        renderMachineWorkspace();
        return;
      }
      if (action === "update") {
        const ids = (state.machine.selectedAssetIDs || []).map((item) => Number(item)).filter(Boolean);
        if (ids.length) machineCtx.openBatchAssetUpdateModal(ids);
        return;
      }
      if (action === "delete") {
        await deleteMachineAssetsBatch((state.machine.selectedAssetIDs || []).map((item) => Number(item)).filter(Boolean));
        return;
      }
    }
    if (button) {
      const assetID = Number(button.dataset.assetId || 0);
      const asset = state.machine.assets.find((item) => Number(item.id) === assetID);
      if (button.dataset.machineAction === "connect") {
        await openAssetAccess(machineCtx, assetID);
        return;
      }
      if (button.dataset.machineAction === "credentials") {
        if (asset) machineCtx.openAccountModal(asset);
        return;
      }
      if (button.dataset.machineAction === "detail") {
        await machineCtx.openAssetDetail(assetID);
        return;
      }
      if (button.dataset.machineAction === "delete") {
        if (asset) await machineCtx.deleteAsset(asset);
      }
      return;
    }
    if (event.target.closest("[data-machine-batch-toggle]")) {
      return;
    }
    if (row) {
      await machineCtx.openAssetDetail(Number(row.dataset.assetRow || 0));
    }
  });
  elements.machineAssetsTable?.addEventListener("change", (event) => {
    const toggle = event.target.closest("[data-machine-batch-toggle]");
    if (!toggle) return;
    const currentIDs = new Set((state.machine.selectedAssetIDs || []).map((item) => Number(item)));
    if (toggle.dataset.machineBatchToggle === "page") {
      const checked = Boolean(toggle.checked);
      (state.machine.assets || []).forEach((asset) => {
        if (checked) {
          currentIDs.add(Number(asset.id));
        } else {
          currentIDs.delete(Number(asset.id));
        }
      });
    }
    if (toggle.dataset.machineBatchToggle === "asset") {
      const assetID = Number(toggle.dataset.assetId || 0);
      if (assetID) {
        if (toggle.checked) {
          currentIDs.add(assetID);
        } else {
          currentIDs.delete(assetID);
        }
      }
    }
    state.machine.selectedAssetIDs = Array.from(currentIDs);
    renderMachineWorkspace();
  });
  elements.machineAccessAssetsTable?.addEventListener("click", async (event) => {
    const button = event.target.closest("[data-machine-access-action]");
    const row = event.target.closest("[data-access-asset-row]");
    if (button) {
      const assetID = Number(button.dataset.assetId || 0);
      const asset = state.machine.assets.find((item) => Number(item.id) === assetID);
      if (button.dataset.machineAccessAction === "connect") {
        await openAssetAccess(machineCtx, assetID);
        switchMachineView("access");
        return;
      }
      if (button.dataset.machineAccessAction === "detail") {
        await machineCtx.openAssetDetail(assetID);
      }
      return;
    }
    if (row) {
      const assetID = Number(row.dataset.accessAssetRow || 0);
      if (assetID) {
        await openAssetAccess(machineCtx, assetID);
        switchMachineView("access");
      }
    }
  });
  elements.machineAccessGroupPanel?.addEventListener("click", async (event) => {
    const assetNode = event.target.closest("[data-access-asset-row]");
    if (assetNode) {
      const assetID = Number(assetNode.dataset.accessAssetRow || 0);
      if (assetID) {
        await openAssetAccess(machineCtx, assetID);
        switchMachineView("access");
      }
      return;
    }
    const summary = event.target.closest("[data-machine-access-group-toggle]");
    if (!summary) return;
    event.preventDefault();
    const groupName = String(summary.dataset.machineAccessGroupToggle || "").trim();
    if (!groupName) return;
    const details = summary.closest(".machine-access-group-node");
    const current = details ? details.hasAttribute("open") : false;
    state.machine.expandedGroups[groupName] = !current;
    renderMachineWorkspace();
  });
  bindMachineSplitPane();
}

export async function loadMachineData() {
  const catalogPromise = (state.cloud.projects || []).length && (state.cloud.environments || []).length
    ? Promise.resolve(null)
    : loadProjectCatalog();
  const params = new URLSearchParams();
  if (state.machine.filters.search) params.set("search", state.machine.filters.search);
  if (state.machine.filters.group) params.set("group", state.machine.filters.group);
  if (state.machine.filters.status) params.set("status", state.machine.filters.status);
  params.set("page", String(state.machine.pagination.page || 1));
  params.set("page_size", String(state.machine.pagination.pageSize || 10));
  params.set("sort_by", state.machine.filters.sortBy || "updated_at");
  params.set("order", state.machine.filters.sortOrder || "desc");
  const [catalog, summaryPayload, groupsPayload, credentialsPayload, quickCommandsPayload, assetsPayload, sessionsPayload] = await Promise.all([
    catalogPromise,
    api("/api/v1/machines/summary"),
    api("/api/v1/machines/groups"),
    api("/api/v1/machines/credentials"),
    api("/api/v1/machines/quick-commands"),
    api(`/api/v1/machines/assets?${params.toString()}`),
    api("/api/v1/machines/sessions"),
  ]);
  if (catalog) {
    state.cloud.projects = catalog.projects || [];
    state.cloud.environments = catalog.environments || [];
    state.cloud.stacks = catalog.stacks || [];
  }
  state.machine.summary = summaryPayload.data || state.machine.summary;
  state.machine.groups = groupsPayload.data || [];
  state.machine.credentials = credentialsPayload.data || [];
  state.machine.quickCommands = quickCommandsPayload.data || [];
  state.machine.assets = assetsPayload.data?.items || [];
  state.machine.selectedAssetIDs = (state.machine.selectedAssetIDs || []).filter((id) => state.machine.assets.some((item) => Number(item.id) === Number(id)));
  state.machine.assetTotalItems = Number(assetsPayload.data?.total_items || 0);
  state.machine.assetTotalPages = Number(assetsPayload.data?.total_pages || 1);
  state.machine.pagination.page = Number(assetsPayload.data?.page || state.machine.pagination.page || 1);
  state.machine.pagination.pageSize = Number(assetsPayload.data?.page_size || state.machine.pagination.pageSize || 10);
  state.machine.sessions = sessionsPayload.data || [];
  if (elements.machineSearchAssets && elements.machineSearchAssets.value !== state.machine.filters.search) {
    elements.machineSearchAssets.value = state.machine.filters.search;
  }
  if (elements.machineAccessSearchAssets && elements.machineAccessSearchAssets.value !== state.machine.filters.search) {
    elements.machineAccessSearchAssets.value = state.machine.filters.search;
  }
  if (elements.machineStatusFilter) {
    elements.machineStatusFilter.value = state.machine.filters.status;
  }
  if (elements.machineAccessStatusFilter) {
    elements.machineAccessStatusFilter.value = state.machine.filters.status;
  }
  if (state.machine.selectedAssetID) {
    const stillExists = state.machine.assets.some((item) => Number(item.id) === Number(state.machine.selectedAssetID));
    if (!stillExists) {
      resetMachineDetailState(machineCtx);
    }
  }
  renderMachineWorkspace();
}

export function activateMachineWorkspace() {
  if (state.machine.activeTerminalKey && state.machine.view !== "detail") {
    state.machine.view = "access";
  }
  renderMachineWorkspace();
}

async function refreshMachineData(message = "") {
  await loadMachineData();
  if (message) toast(message);
}

async function deleteMachineAsset(asset) {
  if (!asset?.id) return;
  const confirmed = await confirmAction({
    eyebrow: "Delete Asset",
    title: `删除服务器 · ${asset.name}`,
    copy: "删除后会同时移除这台服务器的托管账号、会话记录和操作轨迹，当前打开的终端标签也会被关闭。",
    confirmText: "删除服务器",
  });
  if (!confirmed) return;
  const terminalKeys = (state.machine.terminals || [])
    .filter((item) => Number(item.assetID) === Number(asset.id))
    .map((item) => item.key);
  for (const key of terminalKeys) {
    closeMachineTerminal(machineCtx, key);
  }
  await api(`/api/v1/machines/assets/${asset.id}`, { method: "DELETE" });
  if (Number(state.machine.selectedAssetID) === Number(asset.id)) {
    resetMachineDetailState(machineCtx);
  }
  if (state.machine.detail && Number(state.machine.detail.id) === Number(asset.id)) {
    state.machine.detail = null;
  }
  await refreshMachineData(`服务器 ${asset.name} 已删除`);
}

async function deleteMachineAssetsBatch(assetIDs = []) {
  const ids = assetIDs.map((item) => Number(item)).filter(Boolean);
  if (!ids.length) return;
  const confirmed = await confirmAction({
    eyebrow: "Batch Delete",
    title: `批量删除服务器 · ${ids.length} 台`,
    copy: "删除后会同时移除这些服务器的托管账号、会话记录和操作轨迹，当前已打开的终端标签也会被关闭。",
    confirmText: "批量删除",
  });
  if (!confirmed) return;
  const selectedSet = new Set(ids);
  const terminalKeys = (state.machine.terminals || [])
    .filter((item) => selectedSet.has(Number(item.assetID)))
    .map((item) => item.key);
  for (const key of terminalKeys) {
    closeMachineTerminal(machineCtx, key);
  }
  await api("/api/v1/machines/assets/batch-delete", {
    method: "POST",
    body: JSON.stringify({ ids }),
  });
  if (state.machine.selectedAssetID && selectedSet.has(Number(state.machine.selectedAssetID))) {
    resetMachineDetailState(machineCtx);
  }
  state.machine.selectedAssetIDs = [];
  await refreshMachineData(`已批量删除 ${ids.length} 台服务器`);
}

function renderMachineWorkspace() {
  applyMachineAccessSplitWidth();
  renderMachineSummary();
  renderMachineAssets(refreshMachineData);
  renderMachineAccessWorkspace(refreshMachineData);
  renderMachineGroups({
    openAssetGroupModal: machineCtx.openAssetGroupModal,
    refreshMachineData: machineCtx.refreshMachineData,
  });
  renderMachineCredentialLibrary({
    openCredentialLibraryModal: machineCtx.openCredentialLibraryModal,
    openCredentialDetailModal: machineCtx.openCredentialDetailModal,
    refreshMachineData: machineCtx.refreshMachineData,
  });
  renderMachineSessions();
  switchMachineView(state.machine.view || "list");
  renderMachineTerminalDock();
  renderMachineDetail(machineCtx);
  bindMachineTerminalInteractions(machineCtx);
}

function applyMachineAccessSplitWidth() {
  const shell = elements.machineAccessShell;
  if (!shell) return;
  const width = Math.max(280, Math.min(620, Number(state.machine.accessSidebarWidth || 380)));
  shell.style.setProperty("--machine-access-sidebar-width", `${width}px`);
}

function bindMachineSplitPane() {
  if (machineSplitBound) return;
  machineSplitBound = true;
  const splitter = elements.machineAccessSplitter;
  if (!splitter) return;

  const startResize = (clientX) => {
    const shell = elements.machineAccessShell;
    if (!shell) return;
    const rect = shell.getBoundingClientRect();
    const onMove = (moveClientX) => {
      const next = Math.max(280, Math.min(620, moveClientX - rect.left));
      state.machine.accessSidebarWidth = next;
      sessionStorage.setItem("bc_machine_access_sidebar_width", String(next));
      applyMachineAccessSplitWidth();
    };
    const handleMouseMove = (event) => onMove(event.clientX);
    const handleTouchMove = (event) => {
      if (!event.touches?.length) return;
      onMove(event.touches[0].clientX);
    };
    const stop = () => {
      document.removeEventListener("mousemove", handleMouseMove, true);
      document.removeEventListener("mouseup", stop, true);
      document.removeEventListener("touchmove", handleTouchMove, true);
      document.removeEventListener("touchend", stop, true);
      document.body.classList.remove("is-machine-resizing");
    };
    document.body.classList.add("is-machine-resizing");
    document.addEventListener("mousemove", handleMouseMove, true);
    document.addEventListener("mouseup", stop, true);
    document.addEventListener("touchmove", handleTouchMove, true);
    document.addEventListener("touchend", stop, true);
    onMove(clientX);
  };

  splitter.addEventListener("mousedown", (event) => {
    event.preventDefault();
    startResize(event.clientX);
  });
  splitter.addEventListener("touchstart", (event) => {
    if (!event.touches?.length) return;
    event.preventDefault();
    startResize(event.touches[0].clientX);
  }, { passive: false });
  splitter.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    const delta = event.key === "ArrowLeft" ? -24 : 24;
    const next = Math.max(280, Math.min(620, Number(state.machine.accessSidebarWidth || 380) + delta));
    state.machine.accessSidebarWidth = next;
    sessionStorage.setItem("bc_machine_access_sidebar_width", String(next));
    applyMachineAccessSplitWidth();
  });
}
