import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { emptyState, escapeHtml } from "../../shared/utils.js";

export function renderBlueprints() {
  if (!elements.cloudBlueprintsTable) return;
  const items = state.cloud.blueprints || [];
  if (!items.length) {
    elements.cloudBlueprintsTable.innerHTML = emptyState("暂无 Delivery Blueprint");
    return;
  }

  const rows = items.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">${escapeHtml(item.code)}</span></div></td>
      <td>${escapeHtml(item.provider)}</td>
      <td>${escapeHtml(categoryLabel(item.category))}</td>
      <td><div class="cell-stack"><span>${escapeHtml(item.version)}</span><span class="status-pill ${maturityTone(item.maturity)}">${escapeHtml(maturityLabel(item.maturity || "-"))}</span></div></td>
      <td><div class="cell-stack"><span class="group-tag">${escapeHtml(capabilityLabel(item.capability || "-"))}</span><span class="table-meta">${escapeHtml(item.description || "-")}</span></div></td>
      <td>${escapeHtml(item.template_path)}</td>
    </tr>
  `).join("");

  elements.cloudBlueprintsTable.innerHTML = `<table><thead><tr><th>序号</th><th>Delivery Blueprint</th><th>云平台</th><th>交付对象</th><th>版本 / 成熟度</th><th>交付能力</th><th>模板路径</th></tr></thead><tbody>${rows}</tbody></table>`;
}

function maturityTone(value) {
  switch (String(value || "").toLowerCase()) {
    case "apply ready":
      return "";
    case "plan only":
      return "disabled";
    default:
      return "disabled";
  }
}

function categoryLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "network":
      return "Foundation Network";
    case "compute":
      return "Ops Bastion";
    case "cluster":
      return "Cluster";
    case "shared-services":
      return "Shared Services";
    default:
      return String(value || "-");
  }
}

function capabilityLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "apply_destroy_ready":
      return "Plan / Apply / Destroy";
    case "experimental_apply":
      return "Plan / Apply";
    case "apply_ready":
      return "Plan / Apply";
    case "plan_only":
      return "Plan Only";
    default:
      return String(value || "-");
  }
}

function maturityLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "apply ready":
      return "Apply Ready";
    case "plan only":
      return "Plan Only";
    default:
      return "Experimental";
  }
}
