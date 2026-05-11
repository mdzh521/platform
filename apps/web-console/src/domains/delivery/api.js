import { api } from "../../core/api.js";

export async function loadProjectCatalog() {
  const [projects, stacks] = await Promise.all([
    api("/api/v1/projects"),
    api("/api/v1/projects/stacks"),
  ]);
  const projectItems = projects.data || [];
  const environmentResponses = await Promise.all(
    projectItems.map((item) => api(`/api/v1/projects/${Number(item.id)}/environments`))
  );
  return {
    projects: projectItems,
    environments: environmentResponses.flatMap((item) => item.data || []),
    stacks: stacks.data || [],
  };
}

export async function loadCloudSnapshot() {
  const [summary, accounts, catalog, networkPlans, blueprints, jobs, resources, machineCredentials, machineAssets, machineGroups] = await Promise.all([
    api("/api/v1/cloud/summary"),
    api("/api/v1/cloud/accounts"),
    loadProjectCatalog(),
    api("/api/v1/cloud/network-plans"),
    api("/api/v1/cloud/blueprints"),
    api("/api/v1/cloud/jobs"),
    api("/api/v1/cloud/resources"),
    api("/api/v1/machines/credentials"),
    api("/api/v1/machines/assets?page=1&page_size=200&sort_by=updated_at&order=desc"),
    api("/api/v1/machines/groups"),
  ]);
  return {
    summary: summary.data || {},
    accounts: accounts.data || [],
    projects: catalog.projects || [],
    environments: catalog.environments || [],
    stacks: catalog.stacks || [],
    networkPlans: networkPlans.data || [],
    blueprints: blueprints.data || [],
    jobs: jobs.data || [],
    resources: resources.data || [],
    machineCredentials: machineCredentials.data || [],
    machineAssets: machineAssets.data?.items || [],
    machineGroups: machineGroups.data || [],
  };
}

export function testCloudAccount(id) {
  return api(`/api/v1/cloud/accounts/${id}/test`, { method: "POST" });
}

export function createCloudAccount(payload) {
  return api("/api/v1/cloud/accounts", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function deleteCloudAccount(id) {
  return api(`/api/v1/cloud/accounts/${id}`, {
    method: "DELETE",
  });
}

export function listCloudInstanceTypes(accountId, options = {}) {
  const params = new URLSearchParams();
  if (options.provider) params.set("provider", String(options.provider));
  if (options.region) params.set("region", String(options.region));
  if (options.zone) params.set("zone", String(options.zone));
  return api(`/api/v1/cloud/accounts/${accountId}/instance-types?${params.toString()}`);
}

export function createCloudKeyPair(accountId, payload) {
  return api(`/api/v1/cloud/accounts/${accountId}/key-pairs`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function createMachineCredential(payload) {
  return api("/api/v1/machines/credentials", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function deleteMachineCredential(id) {
  return api(`/api/v1/machines/credentials/${id}`, {
    method: "DELETE",
  });
}

export function getMachineCredential(id) {
  return api(`/api/v1/machines/credentials/${id}`);
}

export function importCredentialToMachineAsset(assetId, payload) {
  return api(`/api/v1/machines/assets/${assetId}/accounts/import`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function createNetworkPlan(payload) {
  return api("/api/v1/cloud/network-plans", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateNetworkPlan(id, payload) {
  return api(`/api/v1/cloud/network-plans/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteNetworkPlan(id) {
  return api(`/api/v1/cloud/network-plans/${id}`, {
    method: "DELETE",
  });
}

export function createDeploymentJob(payload) {
  return api("/api/v1/cloud/jobs", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getDeploymentJob(id) {
  return api(`/api/v1/cloud/jobs/${id}`);
}

export function listDeploymentJobLogs(id) {
  return api(`/api/v1/cloud/jobs/${id}/logs`);
}

export function cancelDeploymentJob(id) {
  return api(`/api/v1/cloud/jobs/${id}/cancel`, {
    method: "POST",
  });
}

export function retryDeploymentJob(id, payload = {}) {
  return api(`/api/v1/cloud/jobs/${id}/retry`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function retryCloudResourceEnrollment(id, payload = {}) {
  return api(`/api/v1/cloud/resources/${id}/retry-enrollment`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getProjectGraph(projectID) {
  return api(`/api/v1/graph/projects/${projectID}`);
}
