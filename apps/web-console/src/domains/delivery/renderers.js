import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";

export function renderCloudSummary() {
  const view = buildCloudOverview();
  renderSummaryCards(view);
  renderCurrentStage(view);
  renderFoundationStatus(view);
}

function renderSummaryCards(view) {
  if (!elements.cloudSummaryCards) return;
  elements.cloudSummaryCards.innerHTML = `
    <article class="cloud-summary-card">
      <span class="muted-label">基础交付上下文</span>
      <strong>${view.activeAccounts}</strong>
      <p>${view.activeAccounts > 0 ? "已有可用账号，可以继续基础交付。" : "还没有可用云账号。"}</p>
    </article>
    <article class="cloud-summary-card">
      <span class="muted-label">基础网络</span>
      <strong>${view.networkPlans}</strong>
      <p>${view.succeededFoundationJobs > 0 ? `${view.succeededFoundationJobs} 条基础交付任务已成功。` : "基础网络可以先创建，成功执行后才算真正落地。"}</p>
    </article>
    <article class="cloud-summary-card">
      <span class="muted-label">交付对象</span>
      <strong>${view.deliveryReadyCount}</strong>
      <p>${view.deliveryHeadline}</p>
    </article>
    <article class="cloud-summary-card">
      <span class="muted-label">结果</span>
      <strong>${view.jobCount}</strong>
      <p>${view.resourceCount} 条资源结果，${view.queuedJobs} 条任务仍在排队或执行中。</p>
    </article>
  `;
}

function renderCurrentStage(view) {
  if (!elements.cloudCurrentStage) return;
  elements.cloudCurrentStage.innerHTML = `
    <article class="cloud-current-stage-card">
      <span class="cloud-stage-pill ${view.stageTone}">${view.stageLabel}</span>
      <h4>${view.stageTitle}</h4>
      <p>${view.stageCopy}</p>
      <div class="cloud-current-stage-meta">
        <div>
          <span class="muted-label">下一步</span>
          <strong>${view.nextActionTitle}</strong>
          <p>${view.nextActionCopy}</p>
        </div>
        <div>
          <span class="muted-label">当前阻塞</span>
          <strong>${view.blockerTitle}</strong>
          <p>${view.blockerCopy}</p>
        </div>
        <div>
          <span class="muted-label">最近结果</span>
          <strong>${view.latestResultTitle}</strong>
          <p>${view.latestResultCopy}</p>
        </div>
      </div>
      <div class="action-row">
        <button class="primary-button" data-cloud-shortcut="${view.primaryAction}">${view.primaryActionLabel}</button>
        ${view.secondaryAction ? `<button class="ghost-button" data-cloud-shortcut="${view.secondaryAction}">${view.secondaryActionLabel}</button>` : ""}
      </div>
    </article>
  `;
}

function renderFoundationStatus(view) {
  if (!elements.cloudFoundationStatus) return;
  elements.cloudFoundationStatus.innerHTML = `
    <article class="cloud-status-card ${view.activeAccounts > 0 ? "ready" : "blocked"}">
      <span class="muted-label">基础交付账号</span>
      <strong>${view.activeAccounts > 0 ? "已就绪" : "缺少账号"}</strong>
      <p>${view.activeAccounts > 0 ? `当前有 ${view.activeAccounts} 组可用账号。` : "至少需要一组可用账号，基础交付才能真正落地。"}</p>
    </article>
    <article class="cloud-status-card ${view.networkPlans > 0 ? "active" : "blocked"}">
      <span class="muted-label">基础网络</span>
      <strong>${view.networkPlans > 0 ? "已定义" : "未建立"}</strong>
      <p>${view.networkPlans > 0 ? `当前已保存 ${view.networkPlans} 条基础网络记录。` : "先创建基础网络，收口命名、角色和 provider-specific refs。"}</p>
    </article>
    <article class="cloud-status-card ${view.succeededFoundationJobs > 0 ? "ready" : "waiting"}">
      <span class="muted-label">基础交付执行</span>
      <strong>${view.succeededFoundationJobs > 0 ? "已成功落地" : "等待首次落地"}</strong>
      <p>${view.succeededFoundationJobs > 0 ? `${view.succeededFoundationJobs} 条基础交付已成功完成，可继续复用 outputs。` : "有基础网络还不够，至少要有一条成功的基础交付记录。"}</p>
    </article>
  `;
}

function buildCloudOverview() {
  const workbench = state.cloud.workbench || {};
  if (workbench.stage) {
    return buildCloudOverviewFromWorkbench(workbench);
  }

  const summary = state.cloud.summary || {};
  const accounts = state.cloud.accounts || [];
  const jobs = state.cloud.jobs || [];
  const blueprints = state.cloud.blueprints || [];
  const activeAccounts = accounts.filter((item) => String(item.status || "").toLowerCase() === "active").length;
  const blueprintByID = new Map(blueprints.map((item) => [String(item.id), item]));
  const networkJobs = jobs.filter((job) => blueprintByID.get(String(job.blueprint_id))?.category === "network");
  const computeJobs = jobs.filter((job) => blueprintByID.get(String(job.blueprint_id))?.category === "compute");
  const clusterBlueprints = blueprints.filter((item) => item.category === "cluster");
  const bastionJobs = computeJobs.filter((job) => String(job.blueprint_name || "").toLowerCase().includes("bastion"));
  const latestJob = [...jobs].sort((left, right) =>
    new Date(right.created_at || 0).getTime() - new Date(left.created_at || 0).getTime()
  )[0];
  const succeededFoundationJobs = networkJobs.filter((job) => String(job.status || "").toLowerCase() === "succeeded").length;
  const bastionSucceeded = bastionJobs.some((job) => String(job.status || "").toLowerCase() === "succeeded");
  const foundationComplete = activeAccounts > 0 && Number(summary.networks || 0) > 0 && succeededFoundationJobs > 0;
  const clusterContractReady = clusterBlueprints.some((item) => String(item.schema_json || "").includes("network_ref") || String(item.schema_json || "").includes("cluster_node_refs"));
  const sharedServicesReady = blueprints.some((item) => item.category === "shared-services");

  if (activeAccounts === 0) {
    return {
      activeAccounts,
      networkPlans: Number(summary.networks || 0),
      succeededFoundationJobs,
      deliveryReadyCount: 0,
      deliveryHeadline: "当前还没有可用账号，网络和机器交付都无法启动。",
      jobCount: Number(summary.jobs || 0),
      resourceCount: Number(summary.resources || 0),
      queuedJobs: Number(summary.queued_jobs || 0),
      foundationComplete,
      bastionSucceeded,
      clusterContractReady,
      sharedServicesReady,
      stageLabel: "Stage 1",
      stageTone: "blocked",
      stageTitle: "先新增云账号",
      stageCopy: "当前第一步还没完成。没有可用账号，后面的网络和机器交付都做不了。",
      blockerTitle: "没有可用云账号",
      blockerCopy: "先新增并测试至少一组 AWS 或阿里云账号。",
      latestResultTitle: latestJob ? `${latestJob.name || "最近任务"} · ${statusLabel(latestJob.status || "-")}` : "还没有任务记录",
      latestResultCopy: latestJob ? `${latestJob.blueprint_name || "-"} · ${actionLabel(latestJob.action || "-")}` : "从新增云账号开始。",
      nextActionTitle: "点“新增云账号”",
      nextActionCopy: "账号建好以后，再去创建基础网络。",
      primaryAction: "create-account",
      primaryActionLabel: "新增云账号",
      secondaryAction: "refresh",
      secondaryActionLabel: "刷新阶段状态",
    };
  }

  if (Number(summary.networks || 0) === 0) {
    return {
      activeAccounts,
      networkPlans: Number(summary.networks || 0),
      succeededFoundationJobs,
      deliveryReadyCount: 0,
      deliveryHeadline: "账号已就绪，但基础网络还未建立。",
      jobCount: Number(summary.jobs || 0),
      resourceCount: Number(summary.resources || 0),
      queuedJobs: Number(summary.queued_jobs || 0),
      foundationComplete,
      bastionSucceeded,
      clusterContractReady,
      sharedServicesReady,
      stageLabel: "Stage 1",
      stageTone: "active",
      stageTitle: "下一步创建基础网络",
      stageCopy: "账号已经有了，现在直接创建基础网络就行。",
      blockerTitle: "还没有基础网络",
      blockerCopy: "没有基础网络，就不能创建真正可复用的服务器、Bastion 或 Cluster。",
      latestResultTitle: latestJob ? `${latestJob.name || "最近任务"} · ${statusLabel(latestJob.status || "-")}` : "还没有任务记录",
      latestResultCopy: latestJob ? `${latestJob.blueprint_name || "-"} · ${actionLabel(latestJob.action || "-")}` : "先创建第一条基础网络。",
      nextActionTitle: "点“创建基础网络”",
      nextActionCopy: "先完成一条基础网络记录，再从网络继续创建服务器或其他交付对象。",
      primaryAction: "create-network",
      primaryActionLabel: "创建基础网络",
      secondaryAction: "create-account",
      secondaryActionLabel: "管理账号",
    };
  }

  if (!foundationComplete) {
    return {
      activeAccounts,
      networkPlans: Number(summary.networks || 0),
      succeededFoundationJobs,
      deliveryReadyCount: 1,
      deliveryHeadline: "基础网络已建立，但还没有成功执行记录。",
      jobCount: Number(summary.jobs || 0),
      resourceCount: Number(summary.resources || 0),
      queuedJobs: Number(summary.queued_jobs || 0),
      foundationComplete,
      bastionSucceeded,
      clusterContractReady,
      sharedServicesReady,
      stageLabel: "Stage 1",
      stageTone: "active",
      stageTitle: "先确认基础网络已经真正落地",
      stageCopy: "基础网络虽然已经建了，但还没形成可复用的成功结果。先看网络详情和任务状态，确认底座落地。",
      blockerTitle: "还没有成功的基础网络任务",
      blockerCopy: "没有成功记录，后续蓝图就复用不到 outputs。",
      latestResultTitle: latestJob ? `${latestJob.name || "最近任务"} · ${statusLabel(latestJob.status || "-")}` : "还没有最近结果",
      latestResultCopy: latestJob ? `${latestJob.blueprint_name || "-"} · ${actionLabel(latestJob.action || "-")}` : "先确认基础网络为什么还没成功。",
      nextActionTitle: "先看基础网络和任务状态",
      nextActionCopy: "基础网络创建时会自动发起执行；当前先确认它为什么还没成功。",
      primaryAction: "scroll-networks",
      primaryActionLabel: "查看基础网络",
      secondaryAction: "scroll-jobs",
      secondaryActionLabel: "查看任务状态",
    };
  }

  if (!bastionSucceeded) {
    return {
      activeAccounts,
      networkPlans: Number(summary.networks || 0),
      succeededFoundationJobs,
      deliveryReadyCount: 2,
      deliveryHeadline: "基础网络已经可用，下一步可以从具体网络创建服务器或 Ops Bastion。",
      jobCount: Number(summary.jobs || 0),
      resourceCount: Number(summary.resources || 0),
      queuedJobs: Number(summary.queued_jobs || 0),
      foundationComplete,
      bastionSucceeded,
      clusterContractReady,
      sharedServicesReady,
      stageLabel: "Stage 2",
      stageTone: "active",
      stageTitle: "基础网络已完成，开始创建机器或 Ops Bastion",
      stageCopy: "现在最短路径是从某条基础网络继续创建服务器或 Ops Bastion，不要先跳到脱离网络的通用任务。",
      blockerTitle: "还没有成功的 ops bastion",
      blockerCopy: "目前已有基础网络，但还没有 bastion 成功记录。",
      latestResultTitle: latestJob ? `${latestJob.name || "最近任务"} · ${statusLabel(latestJob.status || "-")}` : "还没有最近结果",
      latestResultCopy: latestJob ? `${latestJob.blueprint_name || "-"} · ${actionLabel(latestJob.action || "-")}` : "下一步从网络创建服务器或 Ops Bastion。",
      nextActionTitle: "从具体网络继续创建交付对象",
      nextActionCopy: "优先从基础网络卡片点“创建服务器”或“创建 Ops Bastion”。",
      primaryAction: "scroll-networks",
      primaryActionLabel: "从网络继续创建",
      secondaryAction: "scroll-jobs",
      secondaryActionLabel: "查看最近任务",
    };
  }

  return {
    activeAccounts,
    networkPlans: Number(summary.networks || 0),
    succeededFoundationJobs,
    deliveryReadyCount: 3,
    deliveryHeadline: "基础网络和第一条 compute 路径已打通，可以继续扩展更多机器或 Cluster。",
    jobCount: Number(summary.jobs || 0),
    resourceCount: Number(summary.resources || 0),
    queuedJobs: Number(summary.queued_jobs || 0),
    foundationComplete,
    bastionSucceeded,
    clusterContractReady,
    sharedServicesReady,
    stageLabel: "Stage 3",
    stageTone: "ready",
    stageTitle: "基础路径已经跑通",
    stageCopy: "基础网络和第一条 compute 交付已经有成功记录。现在可以继续从已有网络扩展服务器、Ops Bastion 或 Cluster。",
    blockerTitle: clusterContractReady ? "Cluster contract 待落地实现" : "Cluster contract 仍是下一阻塞",
    blockerCopy: clusterContractReady ? "contract 已有基础，但还没有收成真正可交付路径。" : "当前 cluster 仍偏 quickstart 占位，需要先统一 network / refs / provider-specific 参数。",
    latestResultTitle: latestJob ? `${latestJob.name || "最近任务"} · ${statusLabel(latestJob.status || "-")}` : "还没有最近结果",
    latestResultCopy: latestJob ? `${latestJob.blueprint_name || "-"} · ${actionLabel(latestJob.action || "-")}` : "可以继续从已有网络扩展交付对象。",
    nextActionTitle: "继续从具体网络扩展交付对象",
    nextActionCopy: "先确认结果没问题，再从已有基础网络扩展服务器、Bastion 或 Cluster。",
    primaryAction: "scroll-networks",
    primaryActionLabel: "从网络继续创建",
    secondaryAction: "scroll-jobs",
    secondaryActionLabel: "查看任务状态",
  };
}

function buildCloudOverviewFromWorkbench(workbench) {
  const summary = workbench.summary || {};
  const stage = workbench.stage || {};
  const firstBlocker = Array.isArray(workbench.blockers) ? workbench.blockers[0] : null;
  const secondaryAction = stage.secondary_action || firstSecondaryAction(workbench.actions || []);
  return {
    activeAccounts: Number(workbench.active_accounts || metricValue(workbench, "active_accounts") || 0),
    networkPlans: Number((summary.networks ?? metricValue(workbench, "foundation_networks")) || 0),
    succeededFoundationJobs: Number(workbench.succeeded_foundation_jobs || 0),
    deliveryReadyCount: Number(workbench.delivery_ready_count || metricValue(workbench, "delivery_ready") || 0),
    deliveryHeadline: metricCopy(workbench, "delivery_ready") || stage.copy || "基础交付状态已由后端整理。",
    jobCount: Number(summary.jobs || 0),
    resourceCount: Number(summary.resources || 0),
    queuedJobs: Number(summary.queued_jobs || 0),
    foundationComplete: Boolean(workbench.foundation_complete),
    bastionSucceeded: Boolean(workbench.bastion_succeeded),
    clusterContractReady: Boolean(workbench.cluster_contract_ready),
    sharedServicesReady: false,
    stageLabel: stage.label || "Stage",
    stageTone: stage.tone || "active",
    stageTitle: stage.title || "基础交付工作台",
    stageCopy: stage.copy || "当前阶段由后端 workbench API 计算。",
    blockerTitle: stage.blocker_title || firstBlocker?.title || "暂无阻塞",
    blockerCopy: stage.blocker_copy || firstBlocker?.copy || "可以继续按当前阶段推进。",
    latestResultTitle: stage.latest_result_title || "还没有任务记录",
    latestResultCopy: stage.latest_result_copy || "从新增云账号和创建基础网络开始。",
    nextActionTitle: stage.next_action_title || "查看当前阶段",
    nextActionCopy: stage.next_action_copy || "按后端规划的下一步继续推进。",
    primaryAction: stage.primary_action || primaryAction(workbench.actions || [])?.key || "refresh",
    primaryActionLabel: stage.primary_action_label || primaryAction(workbench.actions || [])?.label || "刷新状态",
    secondaryAction,
    secondaryActionLabel: stage.secondary_action_label || secondaryActionLabel(workbench.actions || []),
  };
}

function metricValue(workbench, key) {
  const item = (workbench.metrics || []).find((metric) => metric.key === key);
  return item?.value;
}

function metricCopy(workbench, key) {
  const item = (workbench.metrics || []).find((metric) => metric.key === key);
  return item?.copy || "";
}

function primaryAction(actions) {
  return actions.find((item) => item.primary) || actions[0] || null;
}

function firstSecondaryAction(actions) {
  return actions.find((item) => !item.primary)?.key || "";
}

function secondaryActionLabel(actions) {
  return actions.find((item) => !item.primary)?.label || "";
}

function actionLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "plan":
      return "Plan";
    case "apply":
      return "Apply";
    case "destroy":
      return "Destroy";
    default:
      return String(value || "-");
  }
}

function statusLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "queued":
      return "排队中";
    case "claimed":
      return "已领取";
    case "planning":
      return "规划中";
    case "planned":
      return "已完成 Plan";
    case "applying":
      return "执行中";
    case "destroying":
      return "销毁中";
    case "succeeded":
      return "已成功";
    case "failed":
      return "已失败";
    case "destroyed":
      return "已销毁";
    case "cancelled":
      return "已取消";
    default:
      return String(value || "-");
  }
}
