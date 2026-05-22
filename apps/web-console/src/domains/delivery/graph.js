import { elements } from "../../core/dom.js";
import { renderScopeSummary } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { openFormModal } from "../../core/ui.js";
import { emptyState, escapeHtml } from "../../shared/utils.js";
import { getProjectGraph } from "./api.js";

let refreshCloudData = async () => {};

export async function renderProjectGraph(onRefresh = async () => {}) {
  refreshCloudData = onRefresh;
  if (!elements.cloudProjectGraph || !elements.cloudGraphProjectFilter) return;
  renderProjectFilter();
  const projectID = Number(elements.cloudGraphProjectFilter.value || state.cloud.projects?.[0]?.id || 0);
  if (!projectID) {
    elements.cloudProjectGraph.innerHTML = emptyState("先创建一个 Project，图谱才有作用域。");
    return;
  }
  const payload = await getProjectGraph(projectID);
  const graph = payload.data || {};
  const nodes = Array.isArray(graph.nodes) ? graph.nodes : [];
  const edges = Array.isArray(graph.edges) ? graph.edges : [];
  if (!nodes.length) {
    elements.cloudProjectGraph.innerHTML = emptyState("当前 Project 还没有可视化节点。先从基础网络、机器或集群创建链进入。");
    return;
  }

  const grouped = groupNodesByType(nodes);
  elements.cloudProjectGraph.innerHTML = `
    <div class="graph-summary-strip">
      <span class="scope-badge">Project ${(graph.project && escapeHtml(graph.project.name)) || "-"}</span>
      <span class="scope-badge">${nodes.length} 节点</span>
      <span class="scope-badge">${edges.length} 关系</span>
    </div>
    <div class="graph-lane-grid">
      ${Object.entries(grouped).map(([type, items]) => `
        <section class="graph-lane">
          <div class="graph-lane-head">
            <strong>${escapeHtml(typeLabel(type))}</strong>
            <span class="muted-label">${items.length}</span>
          </div>
          <div class="graph-node-stack">
            ${items.map((item) => renderGraphNodeCard(item)).join("")}
          </div>
        </section>
      `).join("")}
    </div>
    <section class="graph-edge-panel">
      <div class="graph-lane-head">
        <strong>关系</strong>
        <span class="muted-label">${edges.length}</span>
      </div>
      <div class="graph-edge-list">
        ${edges.map((edge) => renderGraphEdge(edge, nodes)).join("")}
      </div>
    </section>
  `;
  bindGraphActions(nodes);
}

export function bindProjectGraphEvents(onRefresh = async () => {}) {
  refreshCloudData = onRefresh;
  elements.cloudGraphRefresh?.addEventListener("click", async () => {
    await renderProjectGraph(refreshCloudData);
  });
  elements.cloudGraphProjectFilter?.addEventListener("change", async () => {
    await renderProjectGraph(refreshCloudData);
  });
}

function renderProjectFilter() {
  if (!elements.cloudGraphProjectFilter) return;
  const current = String(elements.cloudGraphProjectFilter.value || state.cloud.projects?.[0]?.id || "");
  elements.cloudGraphProjectFilter.innerHTML = (state.cloud.projects || []).map((item) => `
    <option value="${escapeHtml(String(item.id))}">${escapeHtml(`${item.name} · ${item.code}`)}</option>
  `).join("");
  if ((state.cloud.projects || []).some((item) => String(item.id) === current)) {
    elements.cloudGraphProjectFilter.value = current;
  }
}

function renderGraphNodeCard(node) {
  const metadata = node.metadata || {};
  const scope = renderScopeSummary({
    projectID: numericMeta(metadata.project_id),
    environmentID: numericMeta(metadata.environment_id),
    stackID: numericMeta(metadata.stack_id),
  });
  const summary = firstNonEmptyText(
    scope !== "未绑定归属" ? scope : "",
    metadata.code,
    metadata.provider && metadata.region ? `${metadata.provider} / ${metadata.region}` : "",
    metadata.endpoint,
    metadata.address,
    metadata.private_ip,
  );
  return `
    <article class="graph-node-card">
      <div>
        <span class="muted-label">${escapeHtml(typeLabel(node.type))}</span>
        <strong>${escapeHtml(node.name || node.key)}</strong>
        <p>${escapeHtml(summary || node.status || "-")}</p>
      </div>
      <div class="graph-node-meta">
        <span class="status-pill ${node.status === "ready" || node.status === "enrolled" ? "" : "disabled"}">${escapeHtml(node.status || "-")}</span>
        <button class="ghost-button" data-graph-node-open="${escapeHtml(node.key)}">查看</button>
      </div>
    </article>
  `;
}

function renderGraphEdge(edge, nodes) {
  const lookup = new Map(nodes.map((item) => [item.key, item]));
  const from = lookup.get(edge.from);
  const to = lookup.get(edge.to);
  return `
    <article class="graph-edge-item">
      <strong>${escapeHtml(from?.name || edge.from)}</strong>
      <span class="scope-badge">${escapeHtml(edge.relation || "links")}</span>
      <strong>${escapeHtml(to?.name || edge.to)}</strong>
    </article>
  `;
}

function bindGraphActions(nodes) {
  const lookup = new Map(nodes.map((item) => [item.key, item]));
  elements.cloudProjectGraph?.querySelectorAll("[data-graph-node-open]").forEach((button) => {
    button.addEventListener("click", async () => {
      const node = lookup.get(String(button.dataset.graphNodeOpen || ""));
      if (!node) return;
      await openGraphNode(node);
    });
  });
}

async function openGraphNode(node) {
  const refID = Number(node.ref_id || 0) || 0;
  if (node.type === "foundation-network" && refID) {
    document.querySelector(`[data-network-plan-card='${refID}']`)?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (node.type === "machine-asset" && refID) {
    window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "machine" } }));
    window.dispatchEvent(new CustomEvent("bc:open-machine-asset", { detail: { assetID: refID } }));
    return;
  }
  if (node.type === "k8s-cluster" && refID) {
    window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "business" } }));
    window.dispatchEvent(new CustomEvent("bc:open-k8s-cluster", { detail: { clusterID: refID } }));
    return;
  }
  openFormModal({
    eyebrow: "Project Graph",
    title: `${typeLabel(node.type)} · ${node.name}`,
    copy: `状态：${node.status || "-"}`,
    submitText: false,
    fields: [
      { label: "节点类型", name: "node_type", value: typeLabel(node.type), readOnly: true },
      { label: "引用 ID", name: "ref_id", value: node.ref_id || "-", readOnly: true },
      { label: "元数据", name: "metadata_json", type: "textarea", rows: 10, readOnly: true, value: JSON.stringify(node.metadata || {}, null, 2) },
    ],
  });
}

function groupNodesByType(nodes) {
  const order = ["project", "environment", "stack", "foundation-network", "machine-asset", "k8s-cluster"];
  return order.reduce((result, type) => {
    const items = nodes.filter((item) => item.type === type);
    if (items.length) result[type] = items;
    return result;
  }, {});
}

function typeLabel(type) {
  switch (String(type || "").toLowerCase()) {
    case "project":
      return "Project";
    case "environment":
      return "Environment";
    case "stack":
      return "Stack";
    case "foundation-network":
      return "基础网络";
    case "machine-asset":
      return "Machine";
    case "k8s-cluster":
      return "K8s Cluster";
    default:
      return type || "Node";
  }
}

function numericMeta(value) {
  const parsed = Number(value || 0);
  return parsed > 0 ? parsed : null;
}

function firstNonEmptyText(...values) {
  for (const value of values) {
    if (String(value || "").trim()) {
      return String(value).trim();
    }
  }
  return "";
}
