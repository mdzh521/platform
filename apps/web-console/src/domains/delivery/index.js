import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { showErrorDialog, toast } from "../../core/ui.js";
import { loadCloudSnapshot, retryCloudResourceEnrollment } from "./api.js";
import { bindProjectGraphEvents, renderProjectGraph } from "./graph.js";
import { openCreateAccountModal, renderAccounts } from "./accounts.js";
import { renderBlueprints } from "./blueprints.js";
import { openCreateDeploymentJobModal, openCreateServerDeliveryModal, renderJobs } from "./jobs.js";
import { openCreateNetworkPlanModal, openNetworkKeyModal, renderNetworkPlans } from "./network-plans.js";
import { renderCloudSummary } from "./renderers.js";
import { renderResources } from "./resources.js";

let preferredNetworkPlanID = null;
let preferredBlueprintCode = "";

export async function loadCloudData() {
  const snapshot = await loadCloudSnapshot();
  state.cloud.workbench = snapshot.workbench || null;
  state.cloud.summary = snapshot.summary || {};
  state.cloud.accounts = snapshot.accounts || [];
  state.cloud.projects = snapshot.projects || [];
  state.cloud.environments = snapshot.environments || [];
  state.cloud.stacks = snapshot.stacks || [];
  state.cloud.networkPlans = snapshot.networkPlans || [];
  state.cloud.blueprints = snapshot.blueprints || [];
  state.cloud.jobs = snapshot.jobs || [];
  state.cloud.resources = snapshot.resources || [];
  state.cloud.machineCredentials = snapshot.machineCredentials || [];
  state.cloud.machineAssets = snapshot.machineAssets || [];
  state.cloud.machineGroups = snapshot.machineGroups || [];
  renderCloudPage();
}

export function bindCloudEvents() {
  elements.cloudCreateAccount?.addEventListener("click", () => {
    openCreateAccountModal(loadCloudData);
  });
  elements.cloudCreateNetworkPlan?.addEventListener("click", () => {
    openCreateNetworkPlanModal(loadCloudData);
  });
  elements.cloudRefresh?.addEventListener("click", async () => {
    try {
      await loadCloudData();
    } catch (error) {
      showErrorDialog({ title: "云平台自动化刷新失败", copy: error?.message || "请稍后重试" });
    }
  });
  const handleShortcutClick = (event) => {
    const button = event.target.closest("[data-cloud-shortcut]");
    if (!button) return;
    handleCloudShortcut(String(button.dataset.cloudShortcut || ""));
  };
  elements.cloudCurrentStage?.addEventListener("click", handleShortcutClick);
  elements.formModalForm?.addEventListener("click", handleShortcutClick);
  elements.cloudModule?.addEventListener("click", handleShortcutClick);
  bindProjectGraphEvents(loadCloudData);
}

function handleCloudShortcut(action) {
  if (action.startsWith("retry-enrollment:")) {
    const [, target, rawID] = action.split(":");
    const resourceID = Number(rawID || 0);
    if (!resourceID) return;
    retryCloudResourceEnrollment(resourceID, { target })
      .then(loadCloudData)
      .then(() => toast("enrollment 已重新进入待处理队列"))
      .catch((error) => {
        showErrorDialog({ title: "重试 enrollment 失败", copy: error?.message || "请稍后再试" });
      });
    return;
  }
  if (action === "create-account") {
    openCreateAccountModal(loadCloudData);
    return;
  }
  if (action === "create-network") {
    openCreateNetworkPlanModal(loadCloudData);
    return;
  }
  if (action === "create-job") {
    preferredBlueprintCode = "";
    openCreateDeploymentJobModal(loadCloudData, {
      preselectedNetworkId: preferredNetworkPlanID,
    });
    return;
  }
  if (action.startsWith("create-job-with-network:")) {
    preferredNetworkPlanID = Number(action.split(":")[1] || 0) || null;
    preferredBlueprintCode = "";
    openCreateDeploymentJobModal(loadCloudData, { preselectedNetworkId: preferredNetworkPlanID });
    return;
  }
  if (action.startsWith("create-server-with-network:")) {
    preferredNetworkPlanID = Number(action.split(":")[1] || 0) || null;
    preferredBlueprintCode = "aws-ec2-server";
    openCreateServerDeliveryModal(loadCloudData, {
      preselectedNetworkId: preferredNetworkPlanID,
    });
    return;
  }
  if (action.startsWith("create-bastion-with-network:")) {
    preferredNetworkPlanID = Number(action.split(":")[1] || 0) || null;
    preferredBlueprintCode = `${resolveProviderForNetwork(preferredNetworkPlanID)}-bastion`;
    openCreateDeploymentJobModal(loadCloudData, {
      preselectedNetworkId: preferredNetworkPlanID,
      preselectedBlueprintCode: preferredBlueprintCode,
    });
    return;
  }
  if (action.startsWith("create-cluster-with-network:")) {
    preferredNetworkPlanID = Number(action.split(":")[1] || 0) || null;
    preferredBlueprintCode = resolveProviderForNetwork(preferredNetworkPlanID) === "alicloud" ? "alicloud-ack-quickstart" : "aws-eks-quickstart";
    openCreateDeploymentJobModal(loadCloudData, {
      preselectedNetworkId: preferredNetworkPlanID,
      preselectedBlueprintCode: preferredBlueprintCode,
    });
    return;
  }
  if (action.startsWith("network-key:")) {
    preferredNetworkPlanID = Number(action.split(":")[1] || 0) || null;
    if (!preferredNetworkPlanID) return;
    openNetworkKeyModal(preferredNetworkPlanID);
    return;
  }
  if (action === "refresh") {
    elements.cloudRefresh?.click();
    return;
  }
  if (action === "scroll-networks") {
    elements.cloudNetworkPlansTable?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (action === "scroll-jobs") {
    elements.cloudJobsTable?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (action === "scroll-delivery") {
    elements.cloudBlueprintsTable?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (action === "scroll-resources") {
    elements.cloudResourcesTable?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (action === "scroll-graph") {
    elements.cloudProjectGraph?.scrollIntoView({ behavior: "smooth", block: "start" });
  }
}

function resolveProviderForNetwork(networkPlanID) {
  const plan = (state.cloud.networkPlans || []).find((item) => Number(item.id) === Number(networkPlanID || 0));
  return String(plan?.provider || "aws").trim().toLowerCase() || "aws";
}

export function renderCloudPage() {
  renderCloudSummary();
  renderAccounts(loadCloudData);
  renderNetworkPlans(loadCloudData);
  renderBlueprints();
  renderJobs(loadCloudData);
  renderResources();
  renderProjectGraph(loadCloudData).catch(() => {});
}
