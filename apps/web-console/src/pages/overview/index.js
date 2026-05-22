import { state } from "../../core/state.js";

export function renderOverviewDashboard() {
  const root = document.getElementById("overview-summary-grid");
  if (!root) return;
  const health = state.runtimeConfig?.health || {};
  const systemStatus = String(health.status || "unknown");
  const users = Array.isArray(state.users) ? state.users.length : 0;
  const clusters = Array.isArray(state.clusters) ? state.clusters.length : 0;
  const workloads = Array.isArray(state.workloads) ? state.workloads.length : 0;
  const assets = Array.isArray(state.machine.assets) ? Number(state.machine.assetTotalItems || state.machine.assets.length || 0) : 0;
  const sessions = Array.isArray(state.machine.sessions) ? state.machine.sessions.length : 0;
  const statusCopy = systemStatus === "ok" ? "控制面、缓存和任务链路处于稳定状态。" : "存在需要关注的基础设施状态，请优先检查 healthz 与最近任务。";
  root.innerHTML = `
    <article class="overview-summary-card is-primary">
      <span class="muted-label">平台健康</span>
      <strong>${systemStatus === "ok" ? "运行稳定" : "需要关注"}</strong>
      <p>${statusCopy}</p>
    </article>
    <article class="overview-summary-card">
      <span class="muted-label">基础建设</span>
      <strong>${clusters}</strong>
      <p>已接入集群 ${clusters} 个，当前纳管工作负载 ${workloads} 个。</p>
    </article>
    <article class="overview-summary-card">
      <span class="muted-label">机器工作台</span>
      <strong>${assets}</strong>
      <p>机器资产总数 ${assets}，最近会话记录 ${sessions} 条。</p>
    </article>
    <article class="overview-summary-card">
      <span class="muted-label">系统治理</span>
      <strong>${users}</strong>
      <p>平台用户 ${users}，角色 ${(state.roles || []).length}，身份源 ${(state.sources || []).length}。</p>
    </article>
  `;
}
