import { state } from "../core/state.js";
import { escapeHtml } from "./utils.js";

export function findProject(projectID) {
  return (state.cloud.projects || []).find((item) => Number(item.id) === Number(projectID || 0)) || null;
}

export function findEnvironment(environmentID) {
  return (state.cloud.environments || []).find((item) => Number(item.id) === Number(environmentID || 0)) || null;
}

export function findStack(stackID) {
  return (state.cloud.stacks || []).find((item) => Number(item.id) === Number(stackID || 0)) || null;
}

export function resolveEnvironmentCodeByID(environmentID) {
  return findEnvironment(environmentID)?.code || "";
}

export function renderScopeBadges({ projectID, environmentID, stackID, fallbackEnvironment = "" } = {}) {
  const project = findProject(projectID);
  const environment = findEnvironment(environmentID);
  const stack = findStack(stackID);
  const badges = [];
  if (project) {
    badges.push(`<span class="scope-badge">Project ${escapeHtml(project.name)}</span>`);
  }
  if (environment) {
    badges.push(`<span class="scope-badge">Env ${escapeHtml(environment.code || environment.name)}</span>`);
  } else if (String(fallbackEnvironment || "").trim()) {
    badges.push(`<span class="scope-badge">Env ${escapeHtml(String(fallbackEnvironment).trim())}</span>`);
  }
  if (stack) {
    badges.push(`<span class="scope-badge">Stack ${escapeHtml(stack.name)}</span>`);
  }
  return badges.join("");
}

export function renderScopeSummary({ projectID, environmentID, stackID, fallbackEnvironment = "" } = {}) {
  const project = findProject(projectID);
  const environment = findEnvironment(environmentID);
  const stack = findStack(stackID);
  const segments = [];
  if (project) segments.push(`Project ${project.name}`);
  if (environment) {
    segments.push(`Env ${environment.code || environment.name}`);
  } else if (String(fallbackEnvironment || "").trim()) {
    segments.push(`Env ${String(fallbackEnvironment).trim()}`);
  }
  if (stack) segments.push(`Stack ${stack.name}`);
  return segments.join(" / ") || "未绑定归属";
}
