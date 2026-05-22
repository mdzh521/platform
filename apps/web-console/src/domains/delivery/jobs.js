import { state } from "../../core/state.js";
import { elements } from "../../core/dom.js";
import { renderScopeBadges, renderScopeSummary, resolveEnvironmentCodeByID } from "../../shared/scope.js";
import { confirmAction, openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import { cancelDeploymentJob, createCloudKeyPair, createDeploymentJob, createMachineCredential, getDeploymentJob, getMachineCredential, importCredentialToMachineAsset, listCloudInstanceTypes, listDeploymentJobLogs, retryCloudResourceEnrollment, retryDeploymentJob } from "./api.js";

let refreshJobs = async () => {};

export function renderJobs(onRefresh = async () => {}) {
  refreshJobs = onRefresh;
  if (!elements.cloudJobsTable) return;
  const items = state.cloud.jobs || [];
  const retryGroups = buildRetryGroups(items);
  const rootJobs = retryGroups.rootJobs;
  const jobResources = groupJobResources(state.cloud.resources || [], retryGroups.rootByJobId);
  if (!rootJobs.length) {
    elements.cloudJobsTable.innerHTML = emptyState("暂无交付任务");
    return;
  }

  const cards = rootJobs.map((item, index) => {
    const resources = jobResources.get(Number(item.root_job_id || item.id)) || [];
    return `
      <article class="resource-job-group" data-job-card="${item.id}">
        <details class="resource-job-details">
          <summary class="resource-job-summary">
            <div class="resource-job-summary-main">
              <span class="muted-label">#${index + 1} · ${escapeHtml(item.blueprint_name || "基础交付任务")} · ${escapeHtml(item.account_name || "-")}</span>
              <strong>${escapeHtml(item.name)}</strong>
              <p>${escapeHtml(statusLabel(item.status))} · ${escapeHtml(actionLabel(item.action))} · ${escapeHtml(item.provider)} · ${escapeHtml(item.network_plan || "-")}</p>
              <p class="table-meta">${escapeHtml(renderScopeSummary({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id }))}</p>
              <p class="table-meta">${escapeHtml(renderRetryMeta(item))}</p>
            </div>
            <div class="resource-job-summary-side">
              <span class="resource-state-badge">${escapeHtml(statusLabel(item.status || "-"))}</span>
              <span class="resource-count-badge">${resources.length} 条资源</span>
              <span class="table-meta">${formatDateTime(item.created_at)}</span>
            </div>
          </summary>
          <div class="resource-job-body">
            <div class="action-row compact-action-row">
              <button class="ghost-button" data-job-detail="${item.id}">详情</button>
              <button class="ghost-button" data-job-summary="${item.id}">查看摘要</button>
              ${canReconfigure(item) ? `<button class="ghost-button" data-job-reconfigure="${item.id}">调整配置</button>` : ""}
              ${canCleanup(item) ? `<button class="ghost-button" data-job-destroy="${item.id}">清理资源</button>` : ""}
              ${canCancel(item) ? `<button class="ghost-button" data-job-cancel="${item.id}">取消</button>` : ""}
              ${canRetry(item) ? `<button class="ghost-button" data-job-retry="${item.id}" data-job-action="${escapeHtml(item.action || "")}">重试</button>` : ""}
            </div>
            <div class="table-inline-grid">
              <article>
                <span class="muted-label">基础网络</span>
                <strong>${escapeHtml(item.network_plan || "-")}</strong>
                <p>${escapeHtml(item.provider || "-")} · ${escapeHtml(item.account_name || "-")}</p>
              </article>
              <article>
                <span class="muted-label">同步状态</span>
                <strong>${escapeHtml(resourceSyncStatusLabel(item.resource_sync_status || "-"))}</strong>
                <p>${escapeHtml(item.resource_sync_error || item.log_excerpt || "等待或查看详情")}</p>
              </article>
              <article>
                <span class="muted-label">最近结果</span>
                <strong>${escapeHtml(item.error_message ? "存在错误" : "查看摘要")}</strong>
                <p>${escapeHtml(item.error_message || item.log_excerpt || "无额外说明")}</p>
              </article>
            </div>
            <div class="action-row compact-action-row">
              <span class="muted-label">资源结果</span>
            </div>
            ${resources.length ? renderEmbeddedResourceList(resources) : `<div class="tip-box">这条任务当前还没有回填资源结果。先看任务详情里的执行输出和同步状态。</div>`}
          </div>
        </details>
      </article>
    `;
  }).join("");

  elements.cloudJobsTable.innerHTML = `<div class="resource-job-gallery">${cards}</div>`;
  bindJobActions();
}

export function openCreateDeploymentJobModal(onCreated, options = {}) {
  const accounts = state.cloud.accounts || [];
  const blueprints = state.cloud.blueprints || [];
  const networks = state.cloud.networkPlans || [];
  const deliveryBlueprints = blueprints.filter((item) => String(item.category || "").toLowerCase() !== "network");
  if (!accounts.length || !deliveryBlueprints.length) {
    showErrorDialog({ title: "缺少基础数据", copy: "请先确认云账号和下游交付蓝图 已准备完成。" });
    return;
  }
  if (!networks.length) {
    showErrorDialog({ title: "缺少基础网络", copy: "请先创建并执行至少一条基础网络，然后再创建基础交付任务。" });
    return;
  }

  const preferredNetwork = networks.find((item) => Number(item.id) === Number(options.preselectedNetworkId || 0)) || networks[0];
  const defaultNetwork = preferredNetwork;
  const initialProvider = normalizeProvider(defaultNetwork?.provider || deliveryBlueprints[0]?.provider || accounts[0]?.provider || "aws");
  const initialBlueprints = filterByProvider(deliveryBlueprints, initialProvider);
  const defaultAccount = accounts.find((item) => Number(item.id) === Number(defaultNetwork?.account_id || 0)) || accounts[0];
  const defaultBlueprint = resolveDefaultBlueprint({
    preferredCode: options.preselectedBlueprintCode,
    provider: initialProvider,
    deliveryBlueprints,
    initialBlueprints,
  });
  if (!defaultBlueprint) {
    showErrorDialog({
      title: "缺少可执行蓝图",
      copy: buildMissingBlueprintCopy(options.preselectedBlueprintCode, defaultNetwork),
    });
    return;
  }
  openFormModal({
    eyebrow: "Cloud",
    title: "创建基础交付任务",
    copy: `${describeJobCopy(initialProvider)} 当前基础交付任务会直接继承基础网络的云平台、云账号和基础网络输入；你主要只需要确认交付对象和执行动作。不同 provider、账号、环境不会在这里混用。`,
    fields: [
      { label: "基础交付任务名称", name: "name", required: true },
      { label: "基础网络", name: "network_plan_id", type: "select", value: defaultNetwork ? String(defaultNetwork.id) : "", options: networkOptions(networks) },
      { label: "云平台", name: "provider", type: "select", value: initialProvider, options: providerOptions(blueprints), disabled: true },
      { label: "云账号", name: "account_id", type: "select", value: String(defaultAccount?.id || ""), options: accountOptions(defaultAccount ? [defaultAccount] : []), disabled: true },
      { label: "Project", name: "project_id", type: "select", value: String(defaultNetwork?.project_id || ""), options: projectOptions() },
      { label: "Environment", name: "environment_id", type: "select", value: String(defaultNetwork?.environment_id || ""), options: environmentOptions(defaultNetwork?.project_id), disabled: true },
      { label: "蓝图", name: "blueprint_id", type: "select", value: String(defaultBlueprint.id), options: blueprintOptions(initialBlueprints) },
      { type: "note", copy: blueprintMaturityCopy(defaultBlueprint) },
      { label: "执行动作", name: "action", type: "select", value: "plan", options: allowedActionOptions(defaultBlueprint) },
      { label: "实例付费方式", name: "instance_charge_type", type: "select", value: "PostPaid", options: [{ value: "PostPaid", label: "按量付费" }, { value: "PrePaid", label: "包年包月" }] },
      { label: "购买时长", name: "period", type: "select", value: "1", options: [{ value: "1", label: "1 个月" }, { value: "3", label: "3 个月" }, { value: "6", label: "6 个月" }, { value: "12", label: "12 个月" }] },
      { label: "时长单位", name: "period_unit", type: "select", value: "Month", options: [{ value: "Month", label: "Month" }] },
      { label: "执行输入 JSON", name: "input_json", type: "textarea", rows: 14, value: JSON.stringify(defaultInputForBlueprint(defaultBlueprint, defaultAccount, defaultNetwork), null, 2), readOnly: true },
    ],
    onSubmit: async (form) => {
      let parsedInput = {};
      try {
        parsedInput = JSON.parse(String(form.get("input_json") || "{}"));
      } catch (_error) {
        throw new Error("执行输入 JSON 格式不正确");
      }
      const action = String(form.get("action") || "plan");
      let confirmed = false;
      if (action === "apply" || action === "destroy") {
        confirmed = await confirmAction({
          eyebrow: "Cloud Safety",
          title: action === "apply" ? "确认执行真实 Apply" : "确认执行真实 Destroy",
          copy: action === "apply"
            ? "这会使用当前云账号对 AWS 发起真实资源创建或变更，可能产生费用。确认后才会进入队列。"
            : "这会使用当前云账号对 AWS 发起真实资源销毁。确认后 runner 会继续执行 destroy。",
          confirmText: action === "apply" ? "确认 Apply" : "确认 Destroy",
        });
        if (!confirmed) {
          return;
        }
      }

      const selectedNetwork = (state.cloud.networkPlans || []).find((item) => String(item.id) === String(form.get("network_plan_id")));
      if (!selectedNetwork) {
        throw new Error("请选择有效的基础网络");
      }
      const selectedAccount = (state.cloud.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork.account_id || 0));
      if (!selectedAccount) {
        throw new Error("当前基础网络未绑定有效云账号");
      }
      const selectedBlueprint = (state.cloud.blueprints || []).find((item) => Number(item.id) === Number(form.get("blueprint_id")));
      if (!selectedBlueprint) {
        throw new Error("请选择有效的交付蓝图");
      }

      await createDeploymentJob({
        name: form.get("name"),
        provider: normalizeProvider(selectedNetwork.provider),
        account_id: Number(selectedAccount.id),
        blueprint_id: Number(selectedBlueprint.id),
        network_plan_id: Number(selectedNetwork.id),
        project_id: Number(selectedNetwork.project_id || form.get("project_id") || 0) || null,
        environment_id: Number(selectedNetwork.environment_id || form.get("environment_id") || 0) || null,
        action,
        confirmed,
        input: parsedInput,
      });
      await onCreated();
      toast("基础交付任务已创建");
    },
    onOpen: (form) => bindDeploymentJobProvider(form, { accounts, blueprints, networks, preferredBlueprintCode: options.preselectedBlueprintCode }),
  });
}

export function openCreateServerDeliveryModal(onCreated, options = {}) {
  const networks = state.cloud.networkPlans || [];
  const accounts = state.cloud.accounts || [];
  const blueprints = state.cloud.blueprints || [];
  const network = networks.find((item) => Number(item.id) === Number(options.preselectedNetworkId || 0)) || networks[0];
  if (!network) {
    showErrorDialog({ title: "缺少基础网络", copy: "请先创建并执行至少一条基础网络。" });
    return;
  }
  const provider = normalizeProvider(network.provider);

  const account = accounts.find((item) => Number(item.id) === Number(network.account_id || 0));
  if (!account) {
    showErrorDialog({ title: "缺少云账号", copy: "当前基础网络未绑定有效云账号。" });
    return;
  }

  const serverBlueprintCode = provider === "alicloud" ? "alicloud-ecs-server" : "aws-ec2-server";
  const blueprint = (blueprints || []).find((item) => String(item.code || "").toLowerCase() === serverBlueprintCode);
  if (!blueprint) {
    showErrorDialog({ title: "缺少服务器蓝图", copy: `当前没有可用的 ${provider === "alicloud" ? "阿里云" : "AWS"} Server 蓝图。` });
    return;
  }

  const placementOptions = buildServerPlacementOptions(network, provider);
  const defaultProject = resolveDefaultProjectName(network, account);
  const machineGroupSuggestions = flattenMachineGroups(state.cloud.machineGroups || []);
  const defaultMachineGroup = resolveDefaultMachineGroupName(network, defaultProject, machineGroupSuggestions);
  const defaultPrefix = resolveDefaultServerPrefix(network, defaultProject);
  const projectCredentialOptions = listProjectCredentialsForNetwork(network.id);
  const bastionOptions = listBastionAssetsForNetwork(network.id, provider);
  const initialAction = supportsApply(blueprint) ? "apply" : "plan";
  const existingJobID = Number(options.existingJobId || 0) || 0;
  const initialInput = options.initialInput || {};

  openFormModal({
    eyebrow: "服务器基础交付",
    title: `${existingJobID ? "调整服务器配置" : "创建服务器"} · ${network.name}`,
    copy: existingJobID
      ? "当前会直接修改这条已有服务器任务的执行输入，并在同一个任务上重新发起 apply 或 destroy。"
      : "当前服务器创建会直接绑定到选中的基础网络。云平台、账号、区域、VPC 都继承自这条网络，避免不同环境串在一起。",
    submitText: existingJobID ? "保存并重新执行" : "创建服务器任务",
    panelClass: "modal-panel-wide",
    fields: [
      {
        type: "section",
        eyebrow: "Context",
        label: "基础交付上下文",
        copy: `基础网络：${network.name} ｜ Provider：${String(network.provider || "-").toUpperCase()} ｜ 账号：${account.name} ｜ 区域：${network.region || account.region || "-"}`,
      },
      {
        type: "custom",
        html: renderServerWizardForm({
          network,
          account,
          provider,
          defaultProject,
          defaultMachineGroup,
          machineGroupSuggestions,
          defaultPrefix,
          placementOptions,
          initialAction,
          projectCredentialOptions,
          bastionOptions,
        }),
      },
    ],
    onSubmit: async (form) => {
      const modalForm = elements.formModalForm;
      const action = String(form.get("action") || initialAction);
      const serverCount = Math.max(Number(form.get("server_count") || 1), 1);
      const instanceType = String(form.get("instance_type") || "").trim();
      if (!instanceType) {
          throw new Error("请选择实例规格");
      }
      const selectedSubnetIDs = form.getAll("subnet_ids").map((item) => String(item || "").trim()).filter(Boolean);
      if (!selectedSubnetIDs.length) {
        throw new Error("请至少选择一个子网");
      }

      const hostnameMode = String(form.get("hostname_mode") || "prefix");
      const projectName = String(form.get("project_name") || "").trim();
      const machineGroupName = String(form.get("machine_group_name") || "").trim();
      const projectID = Number(form.get("project_id") || network.project_id || 0) || null;
      const environmentID = Number(form.get("environment_id") || network.environment_id || 0) || null;
      const environmentCode = String(form.get("environment_code") || resolveEnvironmentCodeByID(environmentID) || "dev").trim() || "dev";
      const prefix = String(form.get("hostname_prefix") || "").trim() || defaultPrefix;
      const hostnames = resolveServerHostnames({
        mode: hostnameMode,
        count: serverCount,
        prefix,
        manualText: String(form.get("hostnames") || ""),
      });
      if (hostnames.length !== serverCount) {
        throw new Error("主机名数量和机器数量不一致");
      }

      let confirmed = false;
      if (action === "apply") {
        confirmed = await confirmAction({
          eyebrow: "Cloud Safety",
          title: "确认执行真实 Apply",
          copy: `这会在当前基础网络对应的${provider === "alicloud" ? "阿里云 VPC / vSwitch" : "AWS VPC / Subnet"}中真实创建服务器，可能产生费用。`,
          confirmText: "确认 Apply",
        });
        if (!confirmed) return;
      }

      const serverInstances = hostnames.map((name, index) => ({
        name,
        subnet_id: selectedSubnetIDs[index % selectedSubnetIDs.length],
      }));
      const defaultInput = defaultServerInput(account, network);
      const payloadInput = {
        ...defaultInput,
        project_id: projectID,
        environment_id: environmentID,
        environment: environmentCode,
        project_name: projectName,
        machine_group_name: machineGroupName || projectName || network.name,
        server_name: serverCount > 1 ? prefix : hostnames[0],
        server_count: serverCount,
        server_instances: serverInstances,
        subnet_ids: selectedSubnetIDs,
        subnet_id: selectedSubnetIDs[0],
        instance_type: instanceType,
        system_disk_size_gb: Math.max(Number(form.get("system_disk_size_gb") || 40), 20),
        system_disk_type: String(form.get("system_disk_type") || defaultInput.system_disk_type || "gp3").trim() || defaultInput.system_disk_type || "gp3",
        system_disk_iops: Math.max(Number(form.get("system_disk_iops") || defaultInput.system_disk_iops || 0), 0),
        system_disk_throughput: Math.max(Number(form.get("system_disk_throughput") || defaultInput.system_disk_throughput || 0), 0),
        data_disks: modalForm ? collectServerDataDisks(modalForm) : [],
        key_pair_name: String(form.get("key_pair_name") || "").trim(),
        allocate_eip: String(form.get("allocate_eip") || "true") === "true",
        ingress_cidrs: parseDelimitedValues(String(form.get("ingress_cidrs") || ""), ["0.0.0.0/0"]),
        default_tags: {
          ...(defaultInput.default_tags || {}),
          ...(projectName ? { project: projectName } : {}),
          ...parseTagMapFromText(String(form.get("host_tags") || "")),
        },
      };
      if (provider === "alicloud") {
        payloadInput.instance_charge_type = String(form.get("instance_charge_type") || defaultInput.instance_charge_type || "PostPaid");
        if (payloadInput.instance_charge_type === "PrePaid") {
          payloadInput.period = Math.max(Number(form.get("period") || defaultInput.period || 1), 1);
          payloadInput.period_unit = String(form.get("period_unit") || defaultInput.period_unit || "Month").trim() || "Month";
        } else {
          delete payloadInput.period;
          delete payloadInput.period_unit;
        }
      }

      const jobName = String(form.get("name") || `${projectName || network.name}-servers`).trim();
      if (existingJobID) {
        await retryDeploymentJob(existingJobID, {
          name: jobName,
          action,
          confirmed,
          input: payloadInput,
        });
      } else {
        await createDeploymentJob({
          name: jobName,
          provider,
          account_id: Number(account.id),
          blueprint_id: Number(blueprint.id),
          network_plan_id: Number(network.id),
          project_id: projectID,
          environment_id: environmentID,
          action,
          confirmed,
          input: payloadInput,
        });
      }
      const shouldRegisterBastion = String(form.get("register_bastion_credential") || "false") === "true";
      const bastionAssetID = Number(form.get("bastion_asset_id") || 0) || 0;
      const projectCredentialID = Number(form.get("project_credential_id") || 0) || 0;
      const keyStrategy = String(form.get("key_strategy") || "existing");
      const privateKeyMaterial = String(form.get("private_key_material") || "").trim();
      if (shouldRegisterBastion) {
        if (!bastionAssetID) {
          throw new Error("如果选择录入堡垒机，请先选择目标 Ops Bastion");
        }
        if (keyStrategy === "project_credential" && projectCredentialID) {
          await importCredentialToMachineAsset(bastionAssetID, {
            credential_id: projectCredentialID,
            is_default: true,
          });
        } else {
          if (!privateKeyMaterial) {
            throw new Error("勾选录入堡垒机时，必须提供 SSH 私钥内容");
          }
          const response = await createMachineCredential({
            name: String(form.get("key_note") || `${projectName || network.name}-ssh-key`).trim(),
            username: String(form.get("credential_username") || defaultLoginUsername(provider)).trim() || defaultLoginUsername(provider),
            auth_type: "ssh_key",
            private_key: privateKeyMaterial,
            passphrase: String(form.get("private_key_passphrase") || "").trim(),
            source_type: "cloud_project_key",
            provider,
            scope_kind: "foundation_network",
            scope_ref: String(network.id),
            scope_name: network.name,
            asset_id: bastionAssetID,
            description: `Cloud Server Wizard · ${network.name} · ${String(form.get("key_pair_name") || "").trim()}`,
          });
          const credential = response?.data || null;
          if (credential?.id) {
            await importCredentialToMachineAsset(bastionAssetID, {
              credential_id: Number(credential.id),
              is_default: true,
            });
          }
        }
      }
      await onCreated();
      toast(existingJobID ? "服务器任务已更新并重新排队" : "服务器任务已创建");
    },
    onOpen: (form) => {
      bindServerWizardForm(form, { account, network, provider, placementOptions });
      if (initialInput && Object.keys(initialInput).length) {
        applyServerWizardInitialValues(form, initialInput, {
          network,
          provider,
          initialName: options.existingJobName || "",
          initialAction: options.initialAction || initialAction,
        });
      }
    },
  });
}

function resolveDefaultBlueprint({ preferredCode, provider, deliveryBlueprints, initialBlueprints }) {
  if (preferredCode) {
    return initialBlueprints.find((item) => String(item.code || "").toLowerCase() === String(preferredCode).toLowerCase()) || null;
  }
  return initialBlueprints[0] || deliveryBlueprints[0] || null;
}

function buildMissingBlueprintCopy(preferredCode, network) {
  const networkName = network?.name || "当前基础网络";
  switch (String(preferredCode || "").toLowerCase()) {
    case "aws-ec2-server":
      return `${networkName} 当前没有可用的“服务器创建”蓝图。请先确认使用的是 AWS 基础网络，或者改用通用基础交付任务。`;
    case "aws-bastion":
    case "alicloud-bastion":
      return `${networkName} 当前没有可用的“Ops Bastion”蓝图。请改用通用基础交付任务，或先补齐对应 provider 的 Bastion 蓝图。`;
    case "aws-eks-quickstart":
    case "alicloud-ack-quickstart":
      return `${networkName} 当前没有可用的 Cluster 蓝图。请改用通用基础交付任务，或先补齐对应 provider 的 Cluster 蓝图。`;
    default:
      return `${networkName} 当前没有匹配的 交付蓝图。`;
  }
}

function accountOptions(items) {
  return items.map((item) => ({ value: String(item.id), label: `${item.name} · ${item.provider} · ${item.region}` }));
}

function providerOptions(items) {
  const unique = [...new Set(items.map((item) => item.provider).filter(Boolean))];
  return unique.map((item) => ({ value: item, label: item.toUpperCase() }));
}

function blueprintOptions(items) {
  return items.map((item) => ({ value: String(item.id), label: `${item.name} (${item.provider}) · ${maturityLabel(item.maturity || "experimental")}` }));
}

function networkOptions(items) {
  return items.map((item) => ({ value: String(item.id), label: `${item.name} (${item.provider})` }));
}

function bindJobActions() {
  elements.cloudJobsTable.querySelectorAll("[data-job-detail]").forEach((button) => {
    button.addEventListener("click", async () => {
      await openDeploymentJobDetailModal(Number(button.dataset.jobDetail || 0));
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-job-summary]").forEach((button) => {
    button.addEventListener("click", async () => {
      await openDeploymentJobSummaryModal(Number(button.dataset.jobSummary || 0));
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-job-cancel]").forEach((button) => {
    button.addEventListener("click", async () => {
      const id = Number(button.dataset.jobCancel || 0);
      if (!id) return;
      const confirmed = await confirmAction({
        eyebrow: "Cloud Control",
        title: "确认取消任务",
        copy: "当前只允许取消 queued、claimed、planning 阶段任务。已进入 applying 或 destroying 的任务不会被伪装取消。",
        confirmText: "确认取消",
      });
      if (!confirmed) return;
      await cancelDeploymentJob(id);
      await refreshJobs();
      toast("任务已取消");
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-job-retry]").forEach((button) => {
    button.addEventListener("click", async () => {
      const id = Number(button.dataset.jobRetry || 0);
      const action = String(button.dataset.jobAction || "plan");
      if (!id) return;
      let confirmed = false;
      if (action === "apply" || action === "destroy") {
        confirmed = await confirmAction({
          eyebrow: "Cloud Retry",
          title: action === "apply" ? "确认重试 Apply" : "确认重试 Destroy",
          copy: action === "apply"
            ? "重试会再次对 AWS 发起真实资源创建或变更。"
            : "重试会再次对 AWS 发起真实资源销毁。",
          confirmText: action === "apply" ? "确认重试 Apply" : "确认重试 Destroy",
        });
        if (!confirmed) return;
      }
      await retryDeploymentJob(id, { confirmed });
      await refreshJobs();
      toast("当前任务已重新排队");
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-job-reconfigure]").forEach((button) => {
    button.addEventListener("click", async () => {
      const id = Number(button.dataset.jobReconfigure || 0);
      if (!id) return;
      await openUpdateServerJobModal(id);
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-job-destroy]").forEach((button) => {
    button.addEventListener("click", async () => {
      const id = Number(button.dataset.jobDestroy || 0);
      if (!id) return;
      await createDestroyJobFromExisting(id);
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-resource-retry-enrollment]").forEach((button) => {
    button.addEventListener("click", async () => {
      const resourceID = Number(button.dataset.resourceRetryEnrollment || 0);
      const target = String(button.dataset.enrollmentTarget || "all");
      if (!resourceID) return;
      await retryCloudResourceEnrollment(resourceID, { target });
      await refreshJobs();
      toast("enrollment 已重新进入待处理队列");
    });
  });
  elements.cloudJobsTable.querySelectorAll("[data-resource-enrollment-detail]").forEach((button) => {
    button.addEventListener("click", async () => {
      const resourceID = Number(button.dataset.resourceEnrollmentDetail || 0);
      if (!resourceID) return;
      const resource = (state.cloud.resources || []).find((item) => Number(item.id) === resourceID);
      if (!resource) return;
      await openResourceEnrollmentDetailModal(resource);
    });
  });
}

function groupJobResources(items, rootByJobId = new Map()) {
  return (items || []).reduce((map, item) => {
    const sourceJobID = Number(item.source_job_id || 0);
    const key = rootByJobId.get(sourceJobID) || sourceJobID;
    if (!key) return map;
    if (!map.has(key)) map.set(key, []);
    map.get(key).push(item);
    return map;
  }, new Map());
}

function renderEmbeddedResourceList(items) {
  return `
    <div class="resource-inline-list">
      ${items.map((item) => `
        <article class="resource-inline-item">
          <div>
            <span class="muted-label">${escapeHtml(item.resource_type || "-")} · ${escapeHtml(resourceCategoryLabel(item.category || "-"))}</span>
            <strong>${escapeHtml(item.resource_name || "-")}</strong>
            <p>${escapeHtml(item.account_name || "-")} · ${escapeHtml(item.provider || "-")} · ${escapeHtml(item.region || "-")}</p>
            <div class="scope-badge-row">
              ${renderScopeBadges({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id })}
            </div>
            <div class="scope-badge-row">
              ${renderEnrollmentStatusBadge("machine", item.machine_enrollment_status, item.machine_enrollment_worker)}
              ${renderEnrollmentStatusBadge("cluster", item.cluster_enrollment_status, item.cluster_enrollment_worker)}
            </div>
            ${renderEnrollmentErrorCopy(item)}
          </div>
          <div class="resource-inline-meta">
            <span class="resource-state-badge">${escapeHtml(item.lifecycle_state || "-")}</span>
            <span class="table-meta">${escapeHtml(item.cloud_id || "-")}</span>
            <span class="table-meta">${formatDateTime(item.last_synced_at)}</span>
            <button class="ghost-button" data-resource-enrollment-detail="${item.id}">纳管详情</button>
            ${renderEnrollmentRetryButton(item)}
          </div>
        </article>
      `).join("")}
    </div>
  `;
}

function renderEnrollmentStatusBadge(kind, status, worker) {
  const normalizedKind = kind === "cluster" ? "Cluster" : "Machine";
  const normalizedStatus = String(status || "not_applicable").trim().toLowerCase();
  if (!normalizedStatus || normalizedStatus === "not_applicable") {
    return "";
  }
  const workerText = String(worker || "").trim() ? ` · ${worker}` : "";
  return `<span class="scope-badge enrollment-badge enrollment-${escapeHtml(enrollmentTone(normalizedStatus))}">${escapeHtml(`${normalizedKind} ${normalizedStatus}${workerText}`)}</span>`;
}

function renderEnrollmentErrorCopy(item) {
  const machineError = String(item.machine_enrollment_error || "").trim();
  const clusterError = String(item.cluster_enrollment_error || "").trim();
  const message = machineError || clusterError;
  if (!message) return "";
  return `<p class="table-meta enrollment-error-copy">${escapeHtml(message)}</p>`;
}

function renderEnrollmentRetryButton(item) {
  if (String(item.resource_role || "").toLowerCase() === "compute") {
    return `<button class="ghost-button" data-resource-retry-enrollment="${item.id}" data-enrollment-target="machine">重试机器纳管</button>`;
  }
  if (String(item.resource_role || "").toLowerCase() === "cluster" && isClusterControlPlaneResourceType(item.resource_type || "")) {
    return `<button class="ghost-button" data-resource-retry-enrollment="${item.id}" data-enrollment-target="cluster">重试集群纳管</button>`;
  }
  return "";
}

async function openResourceEnrollmentDetailModal(item) {
  openFormModal({
    eyebrow: "Enrollment",
    title: `纳管详情 · ${item.resource_name || item.cloud_id || item.id}`,
    copy: `${item.resource_type || "-"} · ${item.provider || "-"} · ${item.region || "-"}`,
    submitText: false,
    fields: [
      { label: "归属", name: "scope_summary", value: renderScopeSummary({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id }), readOnly: true },
      { label: "Machine Enrollment", name: "machine_enrollment_json", type: "textarea", rows: 6, readOnly: true, value: JSON.stringify({ status: item.machine_enrollment_status || "-", worker: item.machine_enrollment_worker || "", error: item.machine_enrollment_error || "", enrolled_at: item.machine_enrolled_at || null }, null, 2) },
      { label: "Cluster Enrollment", name: "cluster_enrollment_json", type: "textarea", rows: 6, readOnly: true, value: JSON.stringify({ status: item.cluster_enrollment_status || "-", worker: item.cluster_enrollment_worker || "", error: item.cluster_enrollment_error || "", enrolled_at: item.cluster_enrolled_at || null }, null, 2) },
      { label: "Resource Metadata", name: "resource_metadata_json", type: "textarea", rows: 10, readOnly: true, value: JSON.stringify(item.metadata || {}, null, 2) },
      {
        type: "actions",
        actions: enrollmentActions(item),
      },
    ],
  });
}

function enrollmentActions(item) {
  const actions = [];
  if (String(item.resource_role || "").toLowerCase() === "compute") {
    actions.push({ label: "重试机器纳管", shortcut: `retry-enrollment:machine:${item.id}`, tone: "primary" });
  }
  if (String(item.resource_role || "").toLowerCase() === "cluster" && isClusterControlPlaneResourceType(item.resource_type || "")) {
    actions.push({ label: "重试集群纳管", shortcut: `retry-enrollment:cluster:${item.id}`, tone: "primary" });
  }
  return actions;
}

function enrollmentTone(status) {
  switch (String(status || "").toLowerCase()) {
    case "enrolled":
      return "ok";
    case "syncing":
    case "created":
    case "pending":
    case "access_pending":
      return "warn";
    default:
      return "error";
  }
}

function resourceCategoryLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "network":
      return "基础网络";
    case "compute":
      return "Compute";
    case "cluster":
      return "Cluster";
    case "shared-services":
      return "Shared Services";
    default:
      return String(value || "-");
  }
}

export async function openDeploymentJobDetailModal(jobID) {
  if (!jobID) return;
  const [jobResponse, logsResponse] = await Promise.all([
    getDeploymentJob(jobID),
    listDeploymentJobLogs(jobID),
  ]);
  const job = jobResponse.data || {};
  const logs = logsResponse.data || [];

  openFormModal({
    eyebrow: "Cloud",
    title: `基础交付任务详情 · ${job.name || `#${jobID}`}`,
    copy: `${job.blueprint_name || "-"} · ${job.account_name || "-"} · ${statusLabel(job.status || "-")} · ${renderScopeSummary({ projectID: job.project_id, environmentID: job.environment_id, stackID: job.stack_id })}`,
    submitText: false,
    fields: [
      {
        type: "actions",
        actions: jobDetailActions(job),
      },
      { label: "执行概览", type: "section", eyebrow: "Overview", copy: `动作：${actionLabel(job.action || "-")} ｜ 基础网络：${job.network_plan || "-"} ｜ 归属：${renderScopeSummary({ projectID: job.project_id, environmentID: job.environment_id, stackID: job.stack_id })} ｜ Runner：${job.runner_name || "-"} ｜ Resource Inventory：${resourceSyncStatusLabel(job.resource_sync_status || "-")}` },
      { label: "执行输入", name: "input_json", type: "textarea", rows: 12, readOnly: true, value: JSON.stringify(job.input || {}, null, 2) },
      { label: "执行摘要", name: "plan_summary_json", type: "textarea", rows: 12, readOnly: true, value: JSON.stringify(job.plan_summary || {}, null, 2) },
      { label: "Resource Inventory 条目", name: "resources_json", type: "textarea", rows: 10, readOnly: true, value: JSON.stringify(job.resources || [], null, 2) },
      { label: "Resource Inventory 同步状态", name: "resource_sync_json", type: "textarea", rows: 4, readOnly: true, value: JSON.stringify({ status: resourceSyncStatusLabel(job.resource_sync_status || "-"), worker: job.resource_sync_worker || "-", synced_at: job.resource_synced_at || null, error: job.resource_sync_error || "" }, null, 2) },
      { label: "执行输出", name: "output_json", type: "textarea", rows: 12, readOnly: true, value: JSON.stringify(job.output || {}, null, 2) },
      { label: "执行日志", name: "logs_text", type: "textarea", rows: 12, readOnly: true, value: formatLogs(logs) },
    ],
  });
}

export async function openDeploymentJobSummaryModal(jobID) {
  if (!jobID) return;
  const [jobResponse, logsResponse] = await Promise.all([
    getDeploymentJob(jobID),
    listDeploymentJobLogs(jobID),
  ]);
  const job = jobResponse.data || {};
  const logs = logsResponse.data || [];

  openFormModal({
    eyebrow: "Cloud Summary",
    title: `任务摘要 · ${job.name || `#${jobID}`}`,
    copy: `${job.blueprint_name || "-"} · ${job.account_name || "-"} · ${statusLabel(job.status || "-")} · ${actionLabel(job.action || "-")} · ${renderScopeSummary({ projectID: job.project_id, environmentID: job.environment_id, stackID: job.stack_id })}`,
    submitText: false,
    panelClass: "modal-panel-wide",
    fields: [
      {
        type: "section",
        eyebrow: "Overview",
        label: "执行概览",
        copy: `基础网络：${job.network_plan || "-"} ｜ 资源 ${Array.isArray(job.resources) ? job.resources.length : 0} 条 ｜ Runner：${job.runner_name || "-"} ｜ 创建于 ${formatDateTime(job.created_at)}`,
      },
      {
        label: "关键信息",
        name: "summary_text",
        type: "textarea",
        rows: 8,
        readOnly: true,
        value: formatJobSummaryText(job),
      },
      {
        label: "失败原因 / 关键提示",
        name: "failure_highlights",
        type: "textarea",
        rows: 12,
        readOnly: true,
        value: extractFailureHighlights(job.log_excerpt || ""),
      },
      {
        label: "执行摘录",
        name: "excerpt_text",
        type: "textarea",
        rows: 24,
        readOnly: true,
        value: formatLogExcerpt(job.log_excerpt || formatLogs(logs)),
      },
    ],
  });
}

function jobDetailActions(job) {
  const blueprintName = String(job?.blueprint_name || "").toLowerCase();
  const networkPlanID = Number(job?.network_plan_id || 0);
  const provider = normalizeProvider(job?.provider);
  const actions = [
    { label: "查看 Resources", shortcut: "scroll-resources", tone: "primary" },
    { label: "返回 Recent Jobs", shortcut: "scroll-jobs" },
  ];
  if ((blueprintName.includes("基础网络") || blueprintName.includes("foundation network")) && networkPlanID) {
    if (provider === "aws") {
      actions.unshift({ label: "创建服务器", shortcut: `create-server-with-network:${networkPlanID}`, tone: "primary" });
    }
    actions.unshift({ label: "创建 Ops Bastion", shortcut: `create-bastion-with-network:${networkPlanID}` });
  } else if (blueprintName.includes("ops bastion") && networkPlanID) {
    actions.unshift({ label: "准备 Cluster", shortcut: `create-cluster-with-network:${networkPlanID}`, tone: "primary" });
  }
  return actions;
}

function buildResourceCountByJob(items, rootByJobId = new Map()) {
  return (items || []).reduce((map, item) => {
    const sourceJobID = Number(item.source_job_id || 0);
    const key = rootByJobId.get(sourceJobID) || sourceJobID;
    if (!key) return map;
    map.set(key, (map.get(key) || 0) + 1);
    return map;
  }, new Map());
}

function buildRetryGroups(items) {
  const jobById = new Map((items || []).map((item) => [Number(item.id), item]));
  const rootByJobId = new Map();
  const groups = new Map();

  const resolveRoot = (job) => {
    let current = job;
    const visited = new Set();
    while (current?.retry_of_job_id && !visited.has(Number(current.id))) {
      visited.add(Number(current.id));
      current = jobById.get(Number(current.retry_of_job_id)) || current;
      if (!current?.retry_of_job_id) break;
    }
    return Number(current?.id || job.id);
  };

  (items || []).forEach((job) => {
    const rootId = resolveRoot(job);
    rootByJobId.set(Number(job.id), rootId);
    if (!groups.has(rootId)) groups.set(rootId, []);
    groups.get(rootId).push(job);
  });

  const rootJobs = Array.from(groups.entries())
    .map(([rootId, attempts]) => {
      const ordered = attempts.sort((left, right) => new Date(right.created_at || 0).getTime() - new Date(left.created_at || 0).getTime());
      const rootJob = attempts.find((item) => Number(item.id) === Number(rootId)) || ordered[ordered.length - 1];
      const latestAttempt = ordered[0];
      return {
        ...rootJob,
        root_job_id: Number(rootId),
        attempt_count: ordered.length,
        latest_attempt_id: latestAttempt?.id,
        latest_attempt_action: latestAttempt?.action,
        latest_attempt_status: latestAttempt?.status,
        latest_attempt_created_at: latestAttempt?.created_at,
      };
    })
    .sort((left, right) => new Date(right.latest_attempt_created_at || right.created_at || 0).getTime() - new Date(left.latest_attempt_created_at || left.created_at || 0).getTime());

  return { rootByJobId, groups, rootJobs };
}

function renderRetryMeta(job) {
  const retryOfJobID = Number(job.retry_of_job_id || 0);
  const attemptCount = Number(job.attempt_count || 1);
  const latestAttemptID = Number(job.latest_attempt_id || job.id);
  const latestAction = actionLabel(job.latest_attempt_action || job.action || "-");
  const latestStatus = statusLabel(job.latest_attempt_status || job.status || "-");
  if (!retryOfJobID) {
    return attemptCount > 1
      ? `共 ${attemptCount} 次尝试 · 最新 #${latestAttemptID} · ${latestAction} / ${latestStatus}`
      : `当前尝试 #${job.id}`;
  }
  return `重试尝试 · 当前 #${job.id} · 来源 #${retryOfJobID} · 共 ${attemptCount} 次`;
}

function formatJobSummaryText(job) {
  const lines = [
    `状态: ${statusLabel(job.status || "-")}`,
    `动作: ${actionLabel(job.action || "-")}`,
    `基础网络: ${job.network_plan || "-"}`,
    `蓝图: ${job.blueprint_name || "-"}`,
    `云账号: ${job.account_name || "-"}`,
    `云平台: ${job.provider || "-"}`,
    `Resource Sync: ${resourceSyncStatusLabel(job.resource_sync_status || "-")}`,
    `同步错误: ${job.resource_sync_error || "-"}`,
    `输出字段: ${Object.keys(job.output || {}).join(", ") || "-"}`,
  ];
  return lines.join("\n");
}

function extractFailureHighlights(text) {
  const lines = String(text || "")
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "\n")
    .split("\n");
  const blocks = [];

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index].trim();
    if (!line.startsWith("Error:")) continue;
    const block = [line];
    let cursor = index + 1;
    while (cursor < lines.length) {
      const next = lines[cursor];
      const trimmed = next.trim();
      if (!trimmed) break;
      if (trimmed.startsWith("Error:")) break;
      block.push(trimmed);
      cursor += 1;
      if (block.length >= 8) break;
    }
    blocks.push(block.join("\n"));
  }

  if (!blocks.length) {
    return "当前没有提炼出明确的错误块。请直接查看下面的执行摘录。";
  }

  return blocks.join("\n\n--------------------\n\n");
}

function formatLogExcerpt(value) {
  const normalized = String(value || "")
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "\n")
    .trim();
  if (!normalized) {
    return "暂无执行摘录";
  }
  return normalized;
}

function renderServerWizardForm({ network, account, provider, defaultProject, defaultMachineGroup, machineGroupSuggestions, defaultPrefix, placementOptions, initialAction, projectCredentialOptions, bastionOptions }) {
  const normalizedProvider = normalizeProvider(provider || account?.provider || network?.provider);
  const providerLabel = normalizedProvider === "alicloud" ? "阿里云" : "AWS";
  const defaultLoginUser = defaultLoginUsername(normalizedProvider);
  const supportsCloudKeyCreate = normalizedProvider === "aws" || normalizedProvider === "alicloud";
  const systemDiskOptions = normalizedProvider === "alicloud"
    ? `
      <option value="cloud_essd" selected>cloud_essd</option>
      <option value="cloud_ssd">cloud_ssd</option>
      <option value="cloud_efficiency">cloud_efficiency</option>
    `
    : `
      <option value="gp3" selected>gp3</option>
      <option value="gp2">gp2</option>
      <option value="io1">io1</option>
      <option value="io2">io2</option>
    `;
  const projectSuggestions = buildProjectSuggestions(network, account);
  const environmentCode = resolveEnvironmentCodeByID(network.environment_id) || network.topology?.environment || "dev";
  const subnetItems = placementOptions.length
    ? placementOptions.map((item) => `
      <label class="network-plan-badge server-wizard-subnet-chip">
        <input type="checkbox" name="subnet_ids" value="${escapeHtml(item.id)}" data-role="${escapeHtml(item.role || item.kind || "-")}" data-slot="${escapeHtml(item.slot || "-")}" />
        ${escapeHtml(item.label)}
      </label>
    `).join("")
    : `<span class="muted-label">当前没有可复用的子网输出，请先确认基础网络已成功执行并产出 subnet outputs。</span>`;

  return `
    <div class="form-grid">
      <div class="form-section-heading field-span-2">
        <span class="eyebrow">Context</span>
        <strong>当前基础交付上下文</strong>
        <p>当前服务器会固定创建在这条基础网络下，不会串到别的云账号、区域或 VPC。</p>
      </div>
      <label>
        <span>基础网络</span>
        <input value="${escapeHtml(network.name)}" readonly />
      </label>
      <label>
        <span>云账号</span>
        <input value="${escapeHtml(account.name)}" readonly />
      </label>
      <label>
        <span>云平台</span>
        <input value="${escapeHtml(providerLabel)}" readonly />
      </label>
      <label>
        <span>区域</span>
        <input name="region_readonly" value="${escapeHtml(network.region || account.region || "-")}" readonly />
      </label>
      <label>
        <span>VPC / 项目标识</span>
        <input value="${escapeHtml(network.topology?.vpc_name || network.name || "-")}" readonly />
      </label>
      <label>
        <span>项目名</span>
        <input name="project_name" list="server-project-suggestions" value="${escapeHtml(defaultProject)}" readonly />
        <datalist id="server-project-suggestions">
          ${projectSuggestions.map((item) => `<option value="${escapeHtml(item)}"></option>`).join("")}
        </datalist>
      </label>
      <label>
        <span>Project ID</span>
        <input name="project_id" value="${escapeHtml(String(network.project_id || ""))}" readonly />
      </label>
      <label>
        <span>Environment</span>
        <input name="environment_code" value="${escapeHtml(environmentCode)}" readonly />
        <input type="hidden" name="environment_id" value="${escapeHtml(String(network.environment_id || ""))}" />
      </label>
      <label>
        <span>机器管理项目 / 分组</span>
        <input name="machine_group_name" list="machine-group-suggestions" value="${escapeHtml(defaultMachineGroup)}" placeholder="例如 OTC / 项目A / Cloud Imported" />
        <datalist id="machine-group-suggestions">
          ${machineGroupSuggestions.map((item) => `<option value="${escapeHtml(item)}"></option>`).join("")}
        </datalist>
      </label>

      <div class="form-section-heading field-span-2">
        <span class="eyebrow">Sizing</span>
        <strong>机器规格</strong>
        <p>${normalizedProvider === "alicloud" ? "实例规格当前优先用平台内置可用列表兜底，后续再接阿里云实时机型读取。" : "实例规格按当前区域实时读取。CPU 和内存由实例规格决定，系统盘单独设置。"}</p>
      </div>
      <label>
        <span>任务名称</span>
        <input name="name" value="${escapeHtml(`${defaultProject || network.name}-servers`)}" required />
      </label>
      <label>
        <span>执行动作</span>
        <select name="action">
          <option value="apply" ${initialAction === "apply" ? "selected" : ""}>Apply</option>
          <option value="plan" ${initialAction === "plan" ? "selected" : ""}>Plan</option>
        </select>
      </label>
      ${normalizedProvider === "alicloud" ? `
        <label>
          <span>付费方式</span>
          <select name="instance_charge_type">
            <option value="PostPaid" selected>PostPaid · 按量付费</option>
            <option value="PrePaid">PrePaid · 包年包月</option>
          </select>
        </label>
        <label>
          <span>购买时长</span>
          <select name="period">
            <option value="1" selected>1 个月</option>
            <option value="3">3 个月</option>
            <option value="6">6 个月</option>
            <option value="12">12 个月</option>
          </select>
          <input type="hidden" name="period_unit" value="Month" />
        </label>
      ` : ""}
      <div class="field-span-2 server-picker-panel">
        <span class="muted-label">实例规格</span>
        <strong>规格选择器</strong>
        <p>点开后用关键字、最少 CPU、最少内存一起筛选，再选一个规格即可。</p>
        <details class="fold-section" open>
          <summary>选择实例规格</summary>
          <div class="fold-section-body">
            <div class="server-picker-toolbar">
              <label>
                <span>搜索</span>
                <input name="instance_keyword_filter" placeholder="规格名 / CPU / 内存，例如 c7、8vcpu、32g" />
              </label>
              <label>
                <span>最少 CPU</span>
                <input name="instance_vcpu_filter" type="number" min="1" placeholder="例如 4" />
              </label>
              <label>
                <span>最少内存 (GiB)</span>
                <input name="instance_memory_filter" type="number" min="1" placeholder="例如 8" />
              </label>
            </div>
            <label>
              <span>实例规格</span>
              <select name="instance_type">
                <option value="">正在读取当前区域机型...</option>
              </select>
            </label>
            <label>
              <span>规格摘要</span>
              <input name="instance_type_summary" value="等待选择实例规格" readonly />
            </label>
          </div>
        </details>
      </div>
      <label>
        <span>机器数量</span>
        <input name="server_count" type="number" min="1" value="1" />
      </label>
      <label>
        <span>分配公网 IP</span>
        <select name="allocate_eip">
          <option value="true" selected>是</option>
          <option value="false">否</option>
        </select>
      </label>
      <label class="field-span-2">
        <span>主机标签</span>
        <textarea name="host_tags" rows="4" placeholder="每行一个 key=value，例如&#10;service=api&#10;tier=app&#10;owner=platform-team"></textarea>
      </label>
      <div class="field-span-2 server-picker-panel">
        <span class="muted-label">Disk</span>
        <strong>磁盘高级设置</strong>
        <p>默认折叠。这里可以设置系统盘，也可以添加多块数据盘，并按盘配置类型、IOPS、吞吐。</p>
        <details class="fold-section">
          <summary>展开磁盘设置</summary>
          <div class="fold-section-body server-disk-settings">
            <div class="server-disk-card">
              <strong>系统盘</strong>
              <div class="server-picker-toolbar">
                <label>
                  <span>大小 (GB)</span>
                  <input name="system_disk_size_gb" type="number" min="20" value="40" />
                </label>
                <label>
                  <span>类型</span>
                  <select name="system_disk_type">
                    ${systemDiskOptions}
                  </select>
                </label>
                <label>
                  <span>IOPS</span>
                  <input name="system_disk_iops" type="number" min="0" value="${normalizedProvider === "alicloud" ? "0" : "3000"}" />
                </label>
                <label>
                  <span>吞吐 (MiB/s)</span>
                  <input name="system_disk_throughput" type="number" min="0" value="${normalizedProvider === "alicloud" ? "0" : "125"}" />
                </label>
              </div>
            </div>
            <div class="server-disk-card">
              <div class="action-row">
                <strong>数据盘</strong>
                <button type="button" class="ghost-button" data-add-server-disk>添加数据盘</button>
              </div>
              <div class="table-shell compact-table-shell" data-server-data-disks></div>
            </div>
          </div>
        </details>
      </div>

      <div class="form-section-heading field-span-2">
        <span class="eyebrow">Batch</span>
        <strong>数量与命名</strong>
        <p>多台机器时可以自动生成主机名，也可以手工批量填写。系统会在你选中的子网之间轮询分布。</p>
      </div>
      <label>
        <span>主机名模式</span>
        <select name="hostname_mode">
          <option value="prefix" selected>前缀自动生成</option>
          <option value="manual">手工批量填写</option>
        </select>
      </label>
      <label>
        <span>主机名前缀</span>
        <input name="hostname_prefix" value="${escapeHtml(defaultPrefix)}" placeholder="支持 aws-sg-prod-otc-redis-[01,2]" />
      </label>
      <label>
        <span>命名规则说明</span>
        <input value="形如 aws-sg-prod-otc-redis-[01,2]，表示从 01 开始，编号宽度 2 位；会按机器数量自动展开。" readonly />
      </label>
      <label class="field-span-2">
        <span>批量主机名</span>
        <textarea name="hostnames" rows="5" placeholder="每行一个主机名，仅在“手工批量填写”时生效"></textarea>
      </label>
      <label class="field-span-2">
        <span>主机名预览</span>
        <textarea name="planned_hosts_preview" rows="5" readonly></textarea>
      </label>

      <div class="form-section-heading field-span-2">
        <span class="eyebrow">Placement</span>
        <strong>子网与接入策略</strong>
        <p>默认固定到当前基础网络的 VPC。多台机器时，会按你勾选的子网轮询分布。</p>
      </div>
      <label class="field-span-2">
        <span>允许接入 CIDR</span>
        <textarea name="ingress_cidrs" rows="3">0.0.0.0/0</textarea>
      </label>
      <div class="form-section-heading field-span-2">
        <span class="eyebrow">Subnets</span>
        <strong>子网选择</strong>
        <p>建议优先选择 workload 子网；只有需要运维入口时，再把 ops 子网一起勾上。</p>
        <div class="action-row server-wizard-subnet-grid">${subnetItems}</div>
      </div>
      <label class="field-span-2">
        <span>计划摘要</span>
        <textarea name="placement_summary" rows="6" readonly></textarea>
      </label>
      <div class="field-span-2 server-distribution-panel">
        <span class="muted-label">分布预览</span>
        <strong>机器与子网分布</strong>
        <p>下面这张表会告诉你每台机器将落到哪个子网。</p>
        <div class="table-shell compact-table-shell" data-server-distribution-preview></div>
      </div>

      <div class="form-section-heading field-span-2">
        <span class="eyebrow">SSH Key</span>
        <strong>密钥处理</strong>
        <p>密钥就在这里创建或录入。你可以直接使用现有 Key Pair、复用当前基础网络的项目密钥，或者手工录入私钥内容。${supportsCloudKeyCreate ? `也可以直接在 ${providerLabel} 里创建一把新密钥。` : `${providerLabel} 这轮先不支持在平台里直接创建云上密钥。`}若需要后续经堡垒机使用，可选择把这把私钥录入机器凭据库。</p>
      </div>
      <label>
        <span>密钥方式</span>
        <select name="key_strategy">
          <option value="existing" selected>使用已有 Key Pair</option>
          <option value="project_credential">使用当前基础网络已有项目密钥</option>
          <option value="create_cloud">在${providerLabel}创建新密钥</option>
          <option value="manual">手工录入私钥</option>
        </select>
      </label>
      <label>
        <span>已有项目密钥</span>
        <select name="project_credential_id">
          <option value="">不使用项目密钥</option>
          ${projectCredentialOptions.map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("")}
        </select>
      </label>
      <label>
        <span>Key Pair 名称</span>
        <input name="key_pair_name" placeholder="例如 otc-prod-app" />
      </label>
      <label>
        <span>登录用户</span>
        <input name="credential_username" value="${escapeHtml(defaultLoginUser)}" />
      </label>
      <label>
        <span>是否录入堡垒机</span>
        <select name="register_bastion_credential">
          <option value="false" selected>否</option>
          <option value="true">是</option>
        </select>
      </label>
      <label>
        <span>目标 Ops Bastion</span>
        <select name="bastion_asset_id">
          <option value="">${bastionOptions.length ? "请选择要同步密钥的 Ops Bastion" : "当前基础网络下还没有 Ops Bastion 资产"}</option>
          ${bastionOptions.map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("")}
        </select>
      </label>
      <div class="field-span-2 action-row">
        <button type="button" class="ghost-button" data-server-create-key ${supportsCloudKeyCreate ? "" : "disabled"}>在${providerLabel}创建密钥</button>
        <span class="muted-label">${supportsCloudKeyCreate ? "先填写 Key Pair 名称，再点这里创建；创建后私钥会自动回填。" : "当前云平台这轮先不支持平台内直接创建云上密钥，请先在控制台创建后填入 Key Pair 名称，或手工录入私钥。"}</span>
      </div>
      <label class="field-span-2">
        <span>SSH 私钥</span>
        <textarea name="private_key_material" rows="8" placeholder="创建云密钥后会自动回填，或者手工粘贴 PEM 私钥"></textarea>
      </label>
      <label>
        <span>私钥口令</span>
        <input name="private_key_passphrase" type="password" placeholder="如果私钥带口令可填写" />
      </label>
      <label>
        <span>密钥备注</span>
        <input name="key_note" placeholder="例如 otc 应用服务器默认密钥" />
      </label>
    </div>
  `;
}

function bindServerWizardForm(form, { account, network, provider, placementOptions }) {
  const normalizedProvider = normalizeProvider(provider || account?.provider || network?.provider);
  const typeField = form.querySelector("[name='instance_type']");
  const summaryField = form.querySelector("[name='instance_type_summary']");
  const keywordFilterField = form.querySelector("[name='instance_keyword_filter']");
  const vcpuFilterField = form.querySelector("[name='instance_vcpu_filter']");
  const memoryFilterField = form.querySelector("[name='instance_memory_filter']");
  const modeField = form.querySelector("[name='hostname_mode']");
  const countField = form.querySelector("[name='server_count']");
  const prefixField = form.querySelector("[name='hostname_prefix']");
  const hostnamesField = form.querySelector("[name='hostnames']");
  const projectField = form.querySelector("[name='project_name']");
  const diskField = form.querySelector("[name='system_disk_size_gb']");
  const systemDiskTypeField = form.querySelector("[name='system_disk_type']");
  const systemDiskIOPSField = form.querySelector("[name='system_disk_iops']");
  const systemDiskThroughputField = form.querySelector("[name='system_disk_throughput']");
  const actionField = form.querySelector("[name='action']");
  const eipField = form.querySelector("[name='allocate_eip']");
  const hostTagsField = form.querySelector("[name='host_tags']");
  const placementSummaryField = form.querySelector("[name='placement_summary']");
  const hostPreviewField = form.querySelector("[name='planned_hosts_preview']");
  const distributionPreview = form.querySelector("[data-server-distribution-preview]");
  const keyStrategyField = form.querySelector("[name='key_strategy']");
  const projectCredentialField = form.querySelector("[name='project_credential_id']");
  const keyPairField = form.querySelector("[name='key_pair_name']");
  const credentialUserField = form.querySelector("[name='credential_username']");
  const registerBastionField = form.querySelector("[name='register_bastion_credential']");
  const bastionAssetField = form.querySelector("[name='bastion_asset_id']");
  const privateKeyField = form.querySelector("[name='private_key_material']");
  const privateKeyPassphraseField = form.querySelector("[name='private_key_passphrase']");
  const keyNoteField = form.querySelector("[name='key_note']");
  const createKeyButton = form.querySelector("[data-server-create-key]");
  const addServerDiskButton = form.querySelector("[data-add-server-disk]");
  const dataDisksContainer = form.querySelector("[data-server-data-disks]");
  const subnetCheckboxes = () => Array.from(form.querySelectorAll("input[name='subnet_ids']"));
  const bastionWrapper = bastionAssetField?.closest("label");

  let instanceTypes = [];

  const updateServerPreview = () => {
    const count = Math.max(Number(countField?.value || 1), 1);
    const prefix = String(prefixField?.value || "").trim() || "server";
    const mode = String(modeField?.value || "prefix");
    const hostnames = resolveServerHostnames({
      mode,
      count,
      prefix,
      manualText: String(hostnamesField?.value || ""),
    });
    const selectedSubnetInputs = subnetCheckboxes().filter((input) => input.checked);
    const selectedSubnets = selectedSubnetInputs.map((input) => ({
      id: input.value,
      label: input.parentElement?.textContent?.trim() || input.value,
      role: input.dataset.role || "-",
      slot: input.dataset.slot || "-",
    }));
    if (hostPreviewField) {
      hostPreviewField.value = hostnames.length ? hostnames.join("\n") : "当前没有可用主机名";
    }
    if (placementSummaryField) {
      const systemDiskType = String(systemDiskTypeField?.value || "gp3").trim() || "gp3";
      const dataDisks = collectServerDataDisks(form);
      placementSummaryField.value = [
        `项目: ${String(projectField?.value || "-").trim() || "-"}`,
        `执行动作: ${String(actionField?.value || "-").trim() || "-"}`,
        `机器数量: ${count}`,
        `实例规格: ${String(typeField?.value || "-").trim() || "-"}`,
        `系统盘: ${Math.max(Number(diskField?.value || 40), 20)} GB · ${systemDiskType} · ${formatDiskIopsPreview(systemDiskType, Number(systemDiskIOPSField?.value || 0))} · ${formatDiskThroughputPreview(systemDiskType, Number(systemDiskThroughputField?.value || 0))}`,
        `数据盘: ${dataDisks.length ? `${dataDisks.length} 块` : "不挂载"}`,
        `公网 IP: ${String(eipField?.value || "true") === "true" ? "分配" : "不分配"}`,
        `付费方式: ${normalizedProvider === "alicloud" ? String(form.querySelector("[name='instance_charge_type']")?.value || "PostPaid") : "按量 / 默认"}`,
        `主机标签: ${summarizeTagText(String(hostTagsField?.value || ""))}`,
        `密钥方式: ${String(keyStrategyField?.value || "existing")}`,
        `Key Pair: ${String(keyPairField?.value || "-").trim() || "-"}`,
        `录入堡垒机: ${String(registerBastionField?.value || "false") === "true" ? "是" : "否"}`,
        `子网数量: ${selectedSubnets.length}`,
        `子网列表: ${selectedSubnets.length ? selectedSubnets.map((item) => item.label).join(" | ") : "未选择"}`,
      ].join("\n");
    }
    if (distributionPreview) {
      distributionPreview.innerHTML = renderServerDistributionPreview(hostnames, selectedSubnets);
    }
  };

  const renderTypeOptions = () => {
    if (!typeField) return;
    const filtered = filterInstanceTypes(instanceTypes, {
      keyword: String(keywordFilterField?.value || "").trim(),
      vcpu: Number(vcpuFilterField?.value || 0),
      memoryGiB: Number(memoryFilterField?.value || 0),
    });
    if (!filtered.length) {
      typeField.innerHTML = `<option value="">没有匹配当前筛选条件的机型</option>`;
      if (summaryField) summaryField.value = "没有匹配当前筛选条件的机型";
      updateServerPreview();
      return;
    }
    const previous = String(typeField.value || "");
    const preferred = String(typeField.dataset.preferredValue || "").trim();
    typeField.innerHTML = filtered.map((item, index) => `
      <option value="${escapeHtml(item.instance_type)}" ${((preferred && item.instance_type === preferred) || (!preferred && previous ? item.instance_type === previous : index === 0)) ? "selected" : ""}>
        ${escapeHtml(item.instance_type)} · ${escapeHtml(String(item.vcpu || 0))} vCPU · ${escapeHtml(item.memory_gib || "0")} GiB
      </option>
    `).join("");
    const desiredValue = preferred || previous;
    const stillSelected = filtered.some((item) => item.instance_type === desiredValue);
    if (!stillSelected) {
      typeField.value = filtered[0].instance_type;
      delete typeField.dataset.preferredValue;
    } else if (desiredValue) {
      typeField.value = desiredValue;
      delete typeField.dataset.preferredValue;
    }
    const selected = filtered.find((item) => String(item.instance_type) === String(typeField.value));
    if (summaryField) {
      summaryField.value = selected
        ? `${selected.instance_type} ｜ ${selected.vcpu} vCPU ｜ ${selected.memory_gib} GiB ｜ ${selected.architecture || "-"} ｜ ${selected.network || "-"}`
        : "等待选择实例规格";
    }
    updateServerPreview();
  };

  const syncKeyMode = () => {
    const strategy = String(keyStrategyField?.value || "existing");
    const hasPrivateKey = String(privateKeyField?.value || "").trim().length > 0;
    const privateVisible = strategy === "manual" || strategy === "create_cloud" || strategy === "project_credential" || hasPrivateKey;
    privateKeyField?.closest("label")?.classList.toggle("hidden", !privateVisible);
    privateKeyPassphraseField?.closest("label")?.classList.toggle("hidden", !privateVisible);
    keyNoteField?.closest("label")?.classList.toggle("hidden", !privateVisible);
    projectCredentialField?.closest("label")?.classList.toggle("hidden", strategy !== "project_credential");
    if (registerBastionField) {
      registerBastionField.disabled = !privateVisible && strategy !== "project_credential";
      if (!privateVisible) {
        registerBastionField.value = "false";
      }
    }
    const shouldShowBastion = String(registerBastionField?.value || "false") === "true";
    bastionWrapper?.classList.toggle("hidden", !shouldShowBastion);
    if (!shouldShowBastion && bastionAssetField) {
      bastionAssetField.value = "";
    }
    updateServerPreview();
  };

  const syncDiskMode = () => {
    const systemType = String(systemDiskTypeField?.value || "gp3");
    systemDiskIOPSField?.closest("label")?.classList.toggle("hidden", !supportsConfigurableIops(systemType));
    systemDiskThroughputField?.closest("label")?.classList.toggle("hidden", !supportsConfigurableThroughput(systemType));
    syncServerDiskRows(form);
    updateServerPreview();
  };

  if (modeField && prefixField && hostnamesField) {
    const syncHostnameMode = () => {
      const manual = String(modeField.value || "prefix") === "manual";
      prefixField.closest("label")?.classList.toggle("hidden", manual);
      hostnamesField.closest("label")?.classList.toggle("hidden", !manual);
      updateServerPreview();
    };
    modeField.addEventListener("change", syncHostnameMode);
    syncHostnameMode();
  }
  [countField, prefixField, hostnamesField, projectField, diskField, systemDiskTypeField, systemDiskIOPSField, systemDiskThroughputField, actionField, eipField].forEach((field) => {
    field?.addEventListener("input", updateServerPreview);
    field?.addEventListener("change", updateServerPreview);
  });
  [keywordFilterField, vcpuFilterField, memoryFilterField].forEach((field) => {
    field?.addEventListener("change", renderTypeOptions);
    field?.addEventListener("input", renderTypeOptions);
  });
  [hostTagsField, keyPairField, keyStrategyField, registerBastionField, privateKeyField, credentialUserField, projectCredentialField, bastionAssetField].forEach((field) => {
    field?.addEventListener("input", updateServerPreview);
    field?.addEventListener("change", updateServerPreview);
  });
  form.querySelector("[name='instance_charge_type']")?.addEventListener("change", updateServerPreview);
  form.querySelector("[name='period']")?.addEventListener("change", updateServerPreview);
  subnetCheckboxes().forEach((input) => input.addEventListener("change", updateServerPreview));
  keyStrategyField?.addEventListener("change", syncKeyMode);
  registerBastionField?.addEventListener("change", syncKeyMode);
  systemDiskTypeField?.addEventListener("change", syncDiskMode);
  projectCredentialField?.addEventListener("change", async () => {
    const credentialID = Number(projectCredentialField?.value || 0) || 0;
    if (!credentialID) {
      if (keyStrategyField && String(keyStrategyField.value || "") === "project_credential") {
        keyPairField.value = "";
        privateKeyField.value = "";
        privateKeyPassphraseField.value = "";
      }
      syncKeyMode();
      updateServerPreview();
      return;
    }
    try {
      const response = await getMachineCredential(credentialID);
      const credential = response?.data || {};
      if (keyStrategyField) keyStrategyField.value = "project_credential";
      if (keyPairField) keyPairField.value = String(credential.name || "").trim();
      if (credentialUserField) credentialUserField.value = String(credential.username || defaultLoginUsername(normalizedProvider)).trim() || defaultLoginUsername(normalizedProvider);
      if (privateKeyField) privateKeyField.value = String(credential.private_key || "");
      if (privateKeyPassphraseField) privateKeyPassphraseField.value = String(credential.passphrase || "");
      if (keyNoteField && !String(keyNoteField.value || "").trim()) {
        keyNoteField.value = String(credential.description || `${network.name} · ${credential.name || "project-key"}`).trim();
      }
      syncKeyMode();
      updateServerPreview();
      toast("已带入当前基础网络的项目密钥");
    } catch (error) {
      await showErrorDialog({ title: "读取项目密钥失败", copy: error?.message || "请稍后重试" });
    }
  });
  addServerDiskButton?.addEventListener("click", () => {
    appendServerDataDiskRow(dataDisksContainer);
    syncDiskMode();
  });
  createKeyButton?.addEventListener("click", async () => {
    const keyName = String(keyPairField?.value || "").trim();
    if (!keyName) {
      await showErrorDialog({ title: "缺少密钥名称", copy: "请先填写 Key Pair 名称，再执行云上创建。" });
      return;
    }
    createKeyButton.disabled = true;
    try {
      const response = await createCloudKeyPair(account.id, {
        provider: normalizedProvider,
        region: network.region || account.region,
        name: keyName,
      });
      const data = response.data || {};
      if (keyPairField) keyPairField.value = String(data.name || keyName);
      if (privateKeyField) privateKeyField.value = String(data.private_key || "");
      if (keyNoteField && !String(keyNoteField.value || "").trim()) {
        keyNoteField.value = `${keyName} generated by cloud wizard`;
      }
      if (registerBastionField) {
        registerBastionField.disabled = false;
      }
      toast(`${providerLabel} 密钥已创建，私钥已回填`);
      updateServerPreview();
    } catch (error) {
      await showErrorDialog({ title: "创建密钥失败", copy: error?.message || "请稍后重试" });
    } finally {
      createKeyButton.disabled = false;
    }
  });
  if (!typeField) return;

  const handleLoadedInstanceTypes = (items) => {
    instanceTypes = items || [];
    if (!instanceTypes.length) {
      typeField.innerHTML = `<option value="">当前区域没有可用机型</option>`;
      if (summaryField) summaryField.value = "当前区域没有可用机型";
      return;
    }
    const updateSummary = () => {
      const selected = instanceTypes.find((item) => String(item.instance_type) === String(typeField.value));
      if (!summaryField) return;
      summaryField.value = selected
        ? `${selected.instance_type} ｜ ${selected.vcpu} vCPU ｜ ${selected.memory_gib} GiB ｜ ${selected.architecture || "-"} ｜ ${selected.network || "-"}`
        : "等待选择实例规格";
      updateServerPreview();
    };
    typeField.addEventListener("change", updateSummary);
    renderTypeOptions();
  };

  const instanceTypeRequest = normalizedProvider === "alicloud"
    ? listCloudInstanceTypes(account.id, { provider: "alicloud", region: network.region || account.region })
    : listCloudInstanceTypes(account.id, { provider: "aws", region: network.region || account.region });
  instanceTypeRequest
    .then((payload) => {
      const items = payload.data || [];
      if (normalizedProvider === "alicloud" && !items.length) {
        handleLoadedInstanceTypes(listAliCloudInstanceTypeFallbacks());
        return;
      }
      handleLoadedInstanceTypes(items);
    })
    .catch((error) => {
      if (normalizedProvider === "alicloud") {
        handleLoadedInstanceTypes(listAliCloudInstanceTypeFallbacks());
        if (summaryField) {
          summaryField.value = `阿里云实时机型读取失败，已回退平台内置列表：${error?.message || "unknown error"}`;
        }
        updateServerPreview();
        return;
      }
      typeField.innerHTML = `<option value="">读取机型失败</option>`;
      if (summaryField) summaryField.value = error?.message || "读取机型失败";
      updateServerPreview();
    });

  if (dataDisksContainer && !dataDisksContainer.children.length) {
    renderServerDataDisksTable(dataDisksContainer, []);
  }
  syncKeyMode();
  syncDiskMode();
  updateServerPreview();
}

function applyServerWizardInitialValues(form, initialInput, options = {}) {
  if (!form || !initialInput) return;
  const setValue = (name, value) => {
    const field = form.querySelector(`[name='${name}']`);
    if (!field || value === undefined || value === null) return;
    field.value = String(value);
  };
  setValue("name", options.initialName || "");
  setValue("action", options.initialAction || "apply");
  setValue("project_name", initialInput.project_name || options.network?.name || "");
  setValue("machine_group_name", initialInput.machine_group_name || "");
  setValue("server_count", initialInput.server_count || 1);
  setValue("hostname_mode", "manual");
  setValue("hostname_prefix", initialInput.server_name || "");
  setValue("instance_type", initialInput.instance_type || "");
  setValue("system_disk_size_gb", initialInput.system_disk_size_gb || 40);
  setValue("system_disk_type", initialInput.system_disk_type || (normalizeProvider(options.provider || options.network?.provider) === "alicloud" ? "cloud_essd" : "gp3"));
  setValue("system_disk_iops", initialInput.system_disk_iops || (normalizeProvider(options.provider || options.network?.provider) === "alicloud" ? 0 : 3000));
  setValue("system_disk_throughput", initialInput.system_disk_throughput || (normalizeProvider(options.provider || options.network?.provider) === "alicloud" ? 0 : 125));
  setValue("allocate_eip", initialInput.allocate_eip === false ? "false" : "true");
  setValue("instance_charge_type", initialInput.instance_charge_type || "PostPaid");
  setValue("period", initialInput.period || 1);
  setValue("period_unit", initialInput.period_unit || "Month");
  setValue("key_pair_name", initialInput.key_pair_name || "");
  setValue("credential_username", initialInput.login_username || defaultLoginUsername(options.provider || options.network?.provider));
  setValue("ingress_cidrs", Array.isArray(initialInput.ingress_cidrs) ? initialInput.ingress_cidrs.join("\n") : "0.0.0.0/0");

  const typeField = form.querySelector("[name='instance_type']");
  if (typeField && initialInput.instance_type) {
    typeField.dataset.preferredValue = String(initialInput.instance_type);
  }

  const hostnamesField = form.querySelector("[name='hostnames']");
  const serverInstances = Array.isArray(initialInput.server_instances) ? initialInput.server_instances : [];
  if (hostnamesField && serverInstances.length) {
    hostnamesField.value = serverInstances.map((item) => String(item?.name || "").trim()).filter(Boolean).join("\n");
  }

  const subnetIDs = new Set(
    [
      ...(Array.isArray(initialInput.subnet_ids) ? initialInput.subnet_ids : []),
      ...serverInstances.map((item) => item?.subnet_id).filter(Boolean),
    ].map((item) => String(item || "").trim()).filter(Boolean),
  );
  form.querySelectorAll("input[name='subnet_ids']").forEach((checkbox) => {
    checkbox.checked = subnetIDs.has(String(checkbox.value || "").trim());
  });

  const hostTagsField = form.querySelector("[name='host_tags']");
  if (hostTagsField) {
    hostTagsField.value = formatEditableHostTags(initialInput.default_tags || {}, initialInput.project_name || "");
  }

  const dataDisksContainer = form.querySelector("[data-server-data-disks]");
  if (dataDisksContainer) {
    renderServerDataDisksTable(dataDisksContainer, Array.isArray(initialInput.data_disks) ? initialInput.data_disks : []);
  }

  [
    "hostname_mode",
    "server_count",
    "hostname_prefix",
    "hostnames",
    "project_name",
    "machine_group_name",
    "instance_type",
    "system_disk_type",
    "system_disk_size_gb",
    "system_disk_iops",
    "system_disk_throughput",
    "allocate_eip",
    "instance_charge_type",
    "period",
    "key_pair_name",
    "ingress_cidrs",
  ].forEach((name) => {
    form.querySelector(`[name='${name}']`)?.dispatchEvent(new Event("change", { bubbles: true }));
  });
}

function buildAWSServerPlacementOptions(network) {
  const topology = network?.topology || {};
  const outputs = latestFoundationOutputsForNetwork(network, "aws");
  const privateSubnetIDs = terraformOutputStringMap(outputs, "private_subnet_ids");
  const publicSubnetIDs = terraformOutputStringMap(outputs, "public_subnet_ids");
  const providerRefs = topology?.provider_network_refs || {};
  const refConfigs = [
    { key: "public_subnet_refs", ids: publicSubnetIDs, kind: "public" },
    { key: "ingress_subnet_refs", ids: publicSubnetIDs, kind: "ingress" },
    { key: "private_subnet_refs", ids: privateSubnetIDs, kind: "private" },
    { key: "workload_subnet_refs", ids: privateSubnetIDs, kind: "workload" },
    { key: "data_subnet_refs", ids: privateSubnetIDs, kind: "data" },
    { key: "ops_subnet_refs", ids: privateSubnetIDs, kind: "ops" },
  ];
  const items = [];
  refConfigs.forEach(({ key: providerKey, ids, kind }) => {
    const refs = Array.isArray(providerRefs?.[providerKey]) ? providerRefs[providerKey] : [];
    refs.forEach((ref) => {
      const plannedNames = Array.isArray(ref?.planned_names) ? ref.planned_names : [];
      const cidrs = Array.isArray(ref?.cidrs) ? ref.cidrs : [];
      plannedNames.forEach((plannedName, index) => {
        const key = plannedNameToSubnetKey(plannedName);
        const subnetID = ids[key];
        if (!subnetID) return;
        items.push({
          id: subnetID,
          kind,
          role: String(ref.role || ref.description || kind || "subnet"),
          slot: plannedNameToSubnetKey(plannedName) || plannedName,
          label: `${plannedName} · ${cidrs[index] || cidrs[0] || "-"} · ${subnetID}`,
        });
      });
    });
  });
  return dedupePlacementOptions(items);
}

function buildAliCloudServerPlacementOptions(network) {
  const topology = network?.topology || {};
  const outputs = latestFoundationOutputsForNetwork(network, "alicloud");
  const providerRefs = topology?.provider_network_refs || {};
  const refConfigs = [
    { key: "slb_vswitch_refs", ids: terraformOutputStringMap(outputs, "slb_vswitch_ids"), kind: "slb" },
    { key: "application_vswitch_refs", ids: terraformOutputStringMap(outputs, "application_vswitch_ids"), kind: "application" },
    { key: "database_vswitch_refs", ids: terraformOutputStringMap(outputs, "database_vswitch_ids"), kind: "database" },
    { key: "ops_vswitch_refs", ids: terraformOutputStringMap(outputs, "ops_vswitch_ids"), kind: "ops" },
    { key: "ack_node_vswitch_refs", ids: terraformOutputStringMap(outputs, "ack_node_vswitch_ids"), kind: "ack-node" },
    { key: "ack_pod_vswitch_refs", ids: terraformOutputStringMap(outputs, "ack_pod_vswitch_ids"), kind: "ack-pod" },
  ];
  const items = [];
  refConfigs.forEach(({ key: providerKey, ids, kind }) => {
    const refs = Array.isArray(providerRefs?.[providerKey]) ? providerRefs[providerKey] : [];
    refs.forEach((ref) => {
      const plannedNames = Array.isArray(ref?.planned_names) ? ref.planned_names : [];
      const cidrs = Array.isArray(ref?.cidrs) ? ref.cidrs : [];
      plannedNames.forEach((plannedName, index) => {
        const key = plannedNameToSubnetKey(plannedName);
        const vswitchID = ids[key];
        if (!vswitchID) return;
        items.push({
          id: vswitchID,
          kind,
          role: String(ref.role || ref.description || kind || "vswitch"),
          slot: plannedNameToSubnetKey(plannedName) || plannedName,
          label: `${plannedName} · ${cidrs[index] || cidrs[0] || "-"} · ${vswitchID}`,
        });
      });
    });
  });
  return dedupePlacementOptions(items);
}

function buildServerPlacementOptions(network, provider) {
  return normalizeProvider(provider) === "alicloud"
    ? buildAliCloudServerPlacementOptions(network)
    : buildAWSServerPlacementOptions(network);
}

function defaultLoginUsername(provider) {
  return normalizeProvider(provider) === "alicloud" ? "root" : "ec2-user";
}

function listAliCloudInstanceTypeFallbacks() {
  return [
    { instance_type: "ecs.g6.large", vcpu: 2, memory_gib: 8, memory_mib: 8192, architecture: "x86_64", network: "enhanced" },
    { instance_type: "ecs.g6.xlarge", vcpu: 4, memory_gib: 16, memory_mib: 16384, architecture: "x86_64", network: "enhanced" },
    { instance_type: "ecs.c6.large", vcpu: 2, memory_gib: 4, memory_mib: 4096, architecture: "x86_64", network: "enhanced" },
    { instance_type: "ecs.c6.xlarge", vcpu: 4, memory_gib: 8, memory_mib: 8192, architecture: "x86_64", network: "enhanced" },
  ];
}

function listProjectCredentialsForNetwork(networkPlanID) {
  return (state.cloud.machineCredentials || [])
    .filter((item) =>
      String(item.source_type || "").toLowerCase() === "cloud_project_key" &&
      String(item.scope_kind || "").toLowerCase() === "foundation_network" &&
      String(item.scope_ref || "") === String(networkPlanID)
    )
    .map((item) => ({
      value: String(item.id),
      label: `${item.name} · ${item.username || "-"}${item.asset_name ? ` · ${item.asset_name}` : ""}`,
    }));
}

function listBastionAssetsForNetwork(networkPlanID, provider) {
  return (state.cloud.machineAssets || [])
    .filter((item) => isBastionAssetForNetwork(item, networkPlanID, provider))
    .map((item) => ({
      value: String(item.id),
      label: `${item.name} · ${item.address} · ${item.account || "-"}`,
    }));
}

function isBastionAssetForNetwork(asset, networkPlanID, provider) {
  if (!asset || String(asset.source_provider || "").toLowerCase() !== String(provider || "").toLowerCase()) {
    return false;
  }
  const tags = Array.isArray(asset.tags) ? asset.tags.map((item) => String(item || "").toLowerCase()) : [];
  const metadata = parseMachineAssetMetadata(asset.description);
  const blueprintCode = String(metadata.blueprint_code || "").toLowerCase();
  const metadataNetworkPlanID = Number(metadata.network_plan_id || 0);
  return (tags.includes("bastion") || blueprintCode.includes("bastion")) && metadataNetworkPlanID === Number(networkPlanID);
}

function parseMachineAssetMetadata(value) {
  try {
    const parsed = JSON.parse(String(value || "").trim() || "{}");
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch (_error) {
    return {};
  }
}

function dedupePlacementOptions(items) {
  const seen = new Set();
  return (items || []).filter((item) => {
    if (!item?.id || seen.has(item.id)) return false;
    seen.add(item.id);
    return true;
  });
}

function filterInstanceTypes(items, filters = {}) {
  const keyword = String(filters.keyword || "").trim().toLowerCase();
  const minVCPU = Number(filters.vcpu || 0);
  const minMemoryGiB = Number(filters.memoryGiB || 0);
  return (items || []).filter((item) => {
    if (keyword) {
      const memoryGiB = Number(item.memory_gib || 0);
      const vcpu = Number(item.vcpu || 0);
      const keywordTargets = [
        String(item.instance_type || "").toLowerCase(),
        String(item.architecture || "").toLowerCase(),
        String(item.network || "").toLowerCase(),
        `${vcpu}`,
        `${vcpu}vcpu`,
        `${vcpu} vcpu`,
        `${memoryGiB}`,
        `${memoryGiB}g`,
        `${memoryGiB}gb`,
        `${memoryGiB} gib`,
        `${memoryGiB}gib`,
        `${vcpu}vcpu ${memoryGiB}g`,
        `${vcpu} vcpu ${memoryGiB} gi`,
      ];
      const matched = keywordTargets.some((target) => target.includes(keyword));
      if (!matched) {
        return false;
      }
    }
    if (minVCPU > 0 && Number(item.vcpu || 0) < minVCPU) {
      return false;
    }
    if (minMemoryGiB > 0 && Number(item.memory_mib || 0) < minMemoryGiB * 1024) {
      return false;
    }
    return true;
  });
}

function renderServerDistributionPreview(hostnames, subnets) {
  if (!subnets.length) {
    return emptyState("请先至少选择一个子网");
  }
  if (!hostnames.length) {
    return emptyState("当前没有可用主机名");
  }
  const rows = hostnames.map((hostname, index) => {
    const subnet = subnets[index % subnets.length];
    return `
      <tr>
        <td>${index + 1}</td>
        <td>${escapeHtml(hostname)}</td>
        <td>${escapeHtml(subnet.role || "-")}</td>
        <td>${escapeHtml(subnet.slot || "-")}</td>
        <td>${escapeHtml(subnet.label || subnet.id || "-")}</td>
      </tr>
    `;
  }).join("");
  return `<table><thead><tr><th>序号</th><th>主机名</th><th>Role</th><th>Subnet Slot</th><th>目标子网</th></tr></thead><tbody>${rows}</tbody></table>`;
}

function parseTagMapFromText(value) {
  return String(value || "")
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
    .reduce((result, line) => {
      const [key, ...rest] = line.split("=");
      const normalizedKey = String(key || "").trim();
      if (!normalizedKey) return result;
      result[normalizedKey] = rest.join("=").trim();
      return result;
    }, {});
}

function summarizeTagText(value) {
  const keys = Object.keys(parseTagMapFromText(value));
  if (!keys.length) return "无";
  return keys.join(", ");
}

function formatEditableHostTags(defaultTags, projectName) {
  const excluded = new Set(["project", "environment", "managedby", "blueprintcode", "name", "role"]);
  return Object.entries(defaultTags || {})
    .filter(([key, value]) => String(key || "").trim() && String(value || "").trim())
    .filter(([key, value]) => !(String(key).toLowerCase() === "project" && String(value) === String(projectName || "")))
    .filter(([key]) => !excluded.has(String(key || "").toLowerCase()))
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}

function supportsConfigurableIops(volumeType) {
  return ["gp3", "io1", "io2"].includes(String(volumeType || "").toLowerCase());
}

function supportsConfigurableThroughput(volumeType) {
  return ["gp3"].includes(String(volumeType || "").toLowerCase());
}

function formatDiskIopsPreview(volumeType, iops) {
  return supportsConfigurableIops(volumeType) ? `${Math.max(Number(iops || 0), 0)} IOPS` : "默认 IOPS";
}

function formatDiskThroughputPreview(volumeType, throughput) {
  return supportsConfigurableThroughput(volumeType) ? `${Math.max(Number(throughput || 0), 0)} MiB/s` : "默认吞吐";
}

function collectServerDataDisks(form) {
  return Array.from(form.querySelectorAll("[data-server-disk-row]"))
    .map((row, index) => {
      const size = Math.max(Number(row.querySelector("[name='data_disk_size_gb']")?.value || 0), 0);
      const type = String(row.querySelector("[name='data_disk_type']")?.value || "gp3").trim() || "gp3";
      const iops = Math.max(Number(row.querySelector("[name='data_disk_iops']")?.value || 0), 0);
      const throughput = Math.max(Number(row.querySelector("[name='data_disk_throughput']")?.value || 0), 0);
      const deviceName = String(row.querySelector("[name='data_disk_device_name']")?.value || `/dev/xvd${String.fromCharCode(98 + index)}`).trim() || `/dev/xvd${String.fromCharCode(98 + index)}`;
      if (size <= 0) return null;
      return {
        device_name: deviceName,
        size_gb: size,
        volume_type: type,
        iops,
        throughput,
      };
    })
    .filter(Boolean);
}

function syncServerDiskRows(form) {
  Array.from(form.querySelectorAll("[data-server-disk-row]")).forEach((row) => {
    const typeField = row.querySelector("[name='data_disk_type']");
    const iopsField = row.querySelector("[name='data_disk_iops']");
    const throughputField = row.querySelector("[name='data_disk_throughput']");
    const volumeType = String(typeField?.value || "gp3");
    iopsField?.closest("label")?.classList.toggle("hidden", !supportsConfigurableIops(volumeType));
    throughputField?.closest("label")?.classList.toggle("hidden", !supportsConfigurableThroughput(volumeType));
  });
}

function appendServerDataDiskRow(container, defaults = {}) {
  const currentRows = Array.from(container.querySelectorAll("[data-server-disk-row]"));
  const nextIndex = currentRows.length;
  const rows = [
    ...currentRows.map((row) => ({
      device_name: row.querySelector("[name='data_disk_device_name']")?.value || "",
      size_gb: row.querySelector("[name='data_disk_size_gb']")?.value || "",
      volume_type: row.querySelector("[name='data_disk_type']")?.value || "gp3",
      iops: row.querySelector("[name='data_disk_iops']")?.value || "3000",
      throughput: row.querySelector("[name='data_disk_throughput']")?.value || "125",
    })),
    {
      device_name: defaults.device_name || `/dev/xvd${String.fromCharCode(98 + nextIndex)}`,
      size_gb: defaults.size_gb || "100",
      volume_type: defaults.volume_type || "gp3",
      iops: defaults.iops || "3000",
      throughput: defaults.throughput || "125",
    },
  ];
  renderServerDataDisksTable(container, rows);
}

function renderServerDataDisksTable(container, rows) {
  if (!container) return;
  if (!rows.length) {
    container.innerHTML = emptyState("当前没有数据盘。需要时点击“添加数据盘”。");
    return;
  }
  container.innerHTML = rows.map((item, index) => `
    <div class="server-disk-row" data-server-disk-row>
      <div class="action-row">
        <strong>数据盘 ${index + 1}</strong>
        <button type="button" class="ghost-button" data-remove-server-disk="${index}">删除</button>
      </div>
      <div class="server-picker-toolbar">
        <label>
          <span>设备名</span>
          <input name="data_disk_device_name" value="${escapeHtml(String(item.device_name || `/dev/xvd${String.fromCharCode(98 + index)}`))}" />
        </label>
        <label>
          <span>大小 (GB)</span>
          <input name="data_disk_size_gb" type="number" min="1" value="${escapeHtml(String(item.size_gb || 100))}" />
        </label>
        <label>
          <span>类型</span>
          <select name="data_disk_type">
            <option value="gp3" ${String(item.volume_type || "gp3") === "gp3" ? "selected" : ""}>gp3</option>
            <option value="gp2" ${String(item.volume_type || "") === "gp2" ? "selected" : ""}>gp2</option>
            <option value="io1" ${String(item.volume_type || "") === "io1" ? "selected" : ""}>io1</option>
            <option value="io2" ${String(item.volume_type || "") === "io2" ? "selected" : ""}>io2</option>
          </select>
        </label>
        <label>
          <span>IOPS</span>
          <input name="data_disk_iops" type="number" min="0" value="${escapeHtml(String(item.iops || 3000))}" />
        </label>
        <label>
          <span>吞吐 (MiB/s)</span>
          <input name="data_disk_throughput" type="number" min="0" value="${escapeHtml(String(item.throughput || 125))}" />
        </label>
      </div>
    </div>
  `).join("");
  container.querySelectorAll("[data-remove-server-disk]").forEach((button) => {
    button.addEventListener("click", () => {
      const removeIndex = Number(button.dataset.removeServerDisk || -1);
      const nextRows = rows.filter((_item, index) => index !== removeIndex);
      renderServerDataDisksTable(container, nextRows);
      const form = container.closest("form");
      if (form) {
        syncServerDiskRows(form);
        form.querySelector("[name='system_disk_type']")?.dispatchEvent(new Event("change", { bubbles: true }));
      }
    });
  });
  container.querySelectorAll("input, select").forEach((field) => {
    field.addEventListener("input", () => {
      const form = container.closest("form");
      if (form) {
        syncServerDiskRows(form);
        form.querySelector("[name='system_disk_size_gb']")?.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });
    field.addEventListener("change", () => {
      const form = container.closest("form");
      if (form) {
        syncServerDiskRows(form);
        form.querySelector("[name='system_disk_size_gb']")?.dispatchEvent(new Event("change", { bubbles: true }));
      }
    });
  });
}

function buildProjectSuggestions(network, account) {
  const values = [
    resolveDefaultProjectName(network, account),
    String(network?.topology?.vpc_name || "").trim(),
    String(network?.name || "").trim(),
    ...parseTagPairsFromLines(account?.default_tags).map((item) => item.value),
    ...parseTagPairsFromLines(network?.topology?.default_tags).map((item) => item.value),
  ].filter(Boolean);
  return [...new Set(values)];
}

function flattenMachineGroups(groups, result = []) {
  (groups || []).forEach((group) => {
    const name = String(group?.name || "").trim();
    if (name) result.push(name);
    flattenMachineGroups(group?.children || [], result);
  });
  return [...new Set(result)];
}

function latestFoundationOutputsForNetwork(network, provider) {
  const blueprintsByID = new Map((state.cloud.blueprints || []).map((item) => [String(item.id), item]));
  const foundationJob = (state.cloud.jobs || [])
    .filter((job) => normalizeProvider(job.provider) === normalizeProvider(provider))
    .filter((job) => Number(job.network_plan_id) === Number(network?.id || 0))
    .filter((job) => String(job.status || "").toLowerCase() === "succeeded")
    .filter((job) => blueprintsByID.get(String(job.blueprint_id))?.category === "network")
    .sort((left, right) => new Date(right.ended_at || right.created_at || 0).getTime() - new Date(left.ended_at || left.created_at || 0).getTime())[0];
  return foundationJob?.output?.outputs || {};
}

function resolveDefaultProjectName(network, account) {
  const tags = [
    ...parseTagPairsFromLines(network?.topology?.default_tags),
  ];
  const project = tags.find((item) => item.key === "project");
  return project?.value || String(network?.topology?.vpc_name || network?.name || "").trim();
}

function resolveDefaultMachineGroupName(network, defaultProject, suggestions) {
  const candidate = String(defaultProject || "").trim();
  if (candidate && (suggestions || []).includes(candidate)) {
    return candidate;
  }
  return candidate || String(network?.name || "Cloud Imported").trim() || "Cloud Imported";
}

function parseTagPairsFromLines(items) {
  return (Array.isArray(items) ? items : []).map((item) => {
    const [key, ...rest] = String(item || "").split("=");
    return { key: String(key || "").trim(), value: rest.join("=").trim() };
  }).filter((item) => item.key && item.value);
}

function resolveDefaultServerPrefix(network, projectName) {
  const base = slugifyHostPart(projectName || network?.topology?.vpc_name || network?.name || "server");
  return base || "server";
}

function slugifyHostPart(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
}

function resolveServerHostnames({ mode, count, prefix, manualText }) {
  if (mode === "manual") {
    const items = String(manualText || "")
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean);
    if (!items.length) return [];
    const expanded = [];
    items.forEach((item) => {
      const pattern = parseHostnameSequencePattern(item);
      if (!pattern) {
        expanded.push(item);
        return;
      }
      const remaining = Math.max(count - expanded.length, 0);
      const take = remaining > 0 ? remaining : 1;
      for (let index = 0; index < take; index += 1) {
        const current = pattern.start + index;
        expanded.push(`${pattern.base}${String(current).padStart(pattern.width, "0")}`);
      }
    });
    return expanded.slice(0, Math.max(count, 1));
  }
  const sequencePattern = parseHostnameSequencePattern(prefix);
  if (sequencePattern) {
    return Array.from({ length: count }, (_item, index) => {
      const current = sequencePattern.start + index;
      return `${sequencePattern.base}${String(current).padStart(sequencePattern.width, "0")}`;
    });
  }
  return Array.from({ length: count }, (_item, index) => {
    if (count === 1) return prefix;
    return `${prefix}-${String(index + 1).padStart(2, "0")}`;
  });
}

function parseHostnameSequencePattern(value) {
  const match = String(value || "").trim().match(/^(.*)\[(\d+),(\d+)\]$/);
  if (!match) return null;
  const base = match[1];
  const startLiteral = match[2];
  const widthLiteral = match[3];
  const start = Number(startLiteral);
  const width = Math.max(Number(widthLiteral), startLiteral.length, 1);
  if (!base || Number.isNaN(start) || Number.isNaN(width)) {
    return null;
  }
  return { base, start, width };
}

function parseDelimitedValues(value, fallback = []) {
  const items = String(value || "")
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter(Boolean);
  return items.length ? items : fallback;
}

function defaultVpcInput(account) {
  if (normalizeProvider(account?.provider) === "alicloud") {
    return {
      region: account?.region || "cn-hangzhou",
      environment: "dev",
      vpc_name: "platform-ali-dev",
      vpc_cidr: "10.20.0.0/16",
      availability_zone_count: 2,
      subnet_groups: [
        { role: "slb", traffic_profile: "internet-entry", cidrs: ["10.20.0.0/24", "10.20.1.0/24"] },
        { role: "application", traffic_profile: "workload", cidrs: ["10.20.10.0/24", "10.20.11.0/24"] },
        { role: "database", traffic_profile: "data", cidrs: ["10.20.20.0/24", "10.20.21.0/24"] },
        { role: "ack-node", traffic_profile: "cluster-node", cidrs: ["10.20.30.0/24", "10.20.31.0/24"] },
        { role: "ack-pod", traffic_profile: "pod-network", cidrs: ["10.20.32.0/24", "10.20.33.0/24"] },
        { role: "ops", traffic_profile: "ops", cidrs: ["10.20.40.0/24", "10.20.41.0/24"] },
      ],
      nat_gateway_count: 1,
      create_bastion_subnet: true,
      bastion_subnet_cidr: "10.20.90.0/24",
      default_tags: {
        project: "platform-center",
        environment: "dev",
        owner: "platform-team",
      },
    };
  }
  return {
    region: account?.region || "ap-southeast-1",
    environment: "dev",
    vpc_name: "platform-core-dev",
    vpc_cidr: "10.10.0.0/16",
    availability_zone_count: 2,
    subnet_groups: [
      { role: "ingress", tier: "public", cidrs: ["10.10.0.0/24", "10.10.1.0/24"] },
      { role: "middleware", tier: "private", cidrs: ["10.10.10.0/24", "10.10.11.0/24"] },
      { role: "database", tier: "private", cidrs: ["10.10.20.0/24", "10.10.21.0/24"] },
      { role: "k8s", tier: "private", cidrs: ["10.10.30.0/24", "10.10.31.0/24"] },
      { role: "ops", tier: "private", cidrs: ["10.10.40.0/24", "10.10.41.0/24"] },
    ],
    nat_gateway_count: 1,
    create_bastion_subnet: true,
    bastion_subnet_cidr: "10.10.90.0/24",
    default_tags: {
      project: "platform-center",
      environment: "dev",
      owner: "platform-team",
    },
  };
}

function defaultBastionInput(account, network) {
  const topology = network?.topology || {};
  const providerRefs = topology.provider_network_refs || {};
  const opsRefs = topology.ops_refs || [];
  const resolvedNetworkIDs = resolveFoundationNetworkIDs(network, account?.provider);
  const provider = normalizeProvider(account?.provider);
  return {
    region: account?.region || (provider === "alicloud" ? "cn-hangzhou" : "ap-southeast-1"),
    environment: "dev",
    bastion_name: provider === "alicloud" ? "platform-ali-bastion-dev" : "platform-aws-bastion-dev",
    instance_type: provider === "alicloud" ? "ecs.g6.large" : "t3.micro",
    instance_charge_type: provider === "alicloud" ? "PostPaid" : "",
    period: provider === "alicloud" ? 1 : null,
    period_unit: provider === "alicloud" ? "Month" : "",
    key_pair_name: "",
    network_ref: network?.name || (provider === "alicloud" ? "foundation-vpc-ali-dev" : "foundation-vpc-aws-dev"),
    foundation_stack_name: topology.foundation_stack_name || topology.vpc_name || network?.name || "",
    name_prefix: topology.name_prefix || topology.vpc_name || "",
    ops_refs: opsRefs,
    provider_network_refs: providerRefs,
    vpc_id: resolvedNetworkIDs.vpc_id || "",
    subnet_id: resolvedNetworkIDs.subnet_id || "",
    vswitch_id: resolvedNetworkIDs.vswitch_id || "",
    system_disk_category: provider === "alicloud" ? "cloud_essd" : "",
    allocate_eip: true,
    ingress_cidrs: ["0.0.0.0/0"],
    default_tags: {
      project: "platform-center",
      environment: "dev",
      owner: "platform-team",
    },
  };
}

function defaultServerInput(account, network) {
  const topology = network?.topology || {};
  const providerRefs = topology.provider_network_refs || {};
  const resolvedNetworkIDs = resolveFoundationNetworkIDs(network, account?.provider, "server");
  const provider = normalizeProvider(account?.provider);
  return {
    region: account?.region || (provider === "alicloud" ? "cn-hangzhou" : "ap-southeast-1"),
    project_id: network?.project_id || null,
    environment_id: network?.environment_id || null,
    environment: resolveEnvironmentCodeByID(network?.environment_id) || "dev",
    project_name: resolveDefaultProjectName(network, account),
    machine_group_name: "",
    server_name: provider === "alicloud" ? "platform-ali-server-dev" : "platform-aws-server-dev",
    server_count: 1,
    instance_type: provider === "alicloud" ? "ecs.g6.large" : "t3.micro",
    system_disk_size_gb: 40,
    system_disk_type: provider === "alicloud" ? "cloud_essd" : "gp3",
    system_disk_iops: provider === "alicloud" ? 0 : 3000,
    system_disk_throughput: provider === "alicloud" ? 0 : 125,
    data_disks: [],
    key_pair_name: "",
    network_ref: network?.name || (provider === "alicloud" ? "foundation-vpc-ali-dev" : "foundation-vpc-aws-dev"),
    foundation_stack_name: topology.foundation_stack_name || topology.vpc_name || network?.name || "",
    name_prefix: topology.name_prefix || topology.vpc_name || "",
    workload_refs: topology.workload_refs || [],
    ops_refs: topology.ops_refs || [],
    provider_network_refs: providerRefs,
    vpc_id: resolvedNetworkIDs.vpc_id || "",
    subnet_id: resolvedNetworkIDs.subnet_id || "",
    vswitch_id: resolvedNetworkIDs.vswitch_id || "",
    allocate_eip: true,
    ingress_cidrs: ["0.0.0.0/0"],
    default_tags: {
      project: "platform-center",
      environment: "dev",
      owner: "platform-team",
    },
  };
}

function defaultClusterInput(blueprint, account, network) {
  const provider = normalizeProvider(account?.provider || blueprint?.provider);
  const topology = network?.topology || {};
  const providerRefs = topology.provider_network_refs || {};
  const defaultTags = {
    project: "platform-center",
    environment: "dev",
    owner: "platform-team",
  };

  if (provider === "alicloud") {
    return {
      region: account?.region || "cn-hangzhou",
      project_id: network?.project_id || null,
      environment_id: network?.environment_id || null,
      environment: resolveEnvironmentCodeByID(network?.environment_id) || "dev",
      cluster_name: `platform-ack-${Date.now().toString().slice(-6)}`,
      kubernetes_version: "1.33.3-aliyun.1",
      cluster_spec: "ack.pro.small",
      worker_instance_type: "ecs.g6.large",
      desired_capacity: 2,
      min_size: 1,
      max_size: 4,
      node_system_disk_category: "cloud_essd",
      node_system_disk_size: 120,
      node_login_password: "AckNode#20260427",
      key_pair_name: "",
      network_ref: network?.name || "foundation-vpc-ali-dev",
      foundation_stack_name: topology.foundation_stack_name || topology.vpc_name || network?.name || "",
      cluster_node_refs: topology.cluster_node_refs || [],
      pod_network_refs: topology.pod_network_refs || [],
      provider_network_refs: providerRefs,
      ops_refs: topology.ops_refs || [],
      endpoint_access: "private",
      vpc_id: "",
      subnet_ids: [],
      pod_vswitch_ids: [],
      slb_vswitch_ids: [],
      service_cidr: "172.21.0.0/20",
      pod_cidr: "172.20.0.0/16",
        create_ack_service_roles: true,
      cluster_addon_profile: "standard",
      install_ebs_csi: true,
      install_efs_csi: true,
      install_lb_controller: true,
      install_metrics_server: false,
      default_tags: defaultTags,
    };
  }

  return {
    region: account?.region || "ap-southeast-1",
    project_id: network?.project_id || null,
    environment_id: network?.environment_id || null,
    environment: resolveEnvironmentCodeByID(network?.environment_id) || "dev",
    cluster_name: "platform-eks-dev",
    kubernetes_version: "1.33.3-aliyun.1",
    node_instance_type: "t3.large",
    desired_capacity: 2,
    min_size: 2,
    max_size: 4,
    network_ref: network?.name || "foundation-vpc-aws-dev",
    foundation_stack_name: topology.foundation_stack_name || topology.vpc_name || network?.name || "",
    cluster_node_refs: topology.cluster_node_refs || [],
    pod_network_refs: topology.pod_network_refs || [],
    provider_network_refs: providerRefs,
    ops_refs: topology.ops_refs || [],
    endpoint_access: "private",
    cluster_addon_profile: "standard",
    install_ebs_csi: true,
    install_efs_csi: true,
    install_lb_controller: true,
    install_metrics_server: false,
    default_tags: defaultTags,
  };
}

function isClusterControlPlaneResourceType(resourceType) {
  const value = String(resourceType || "").trim().toLowerCase();
  if (!value) return false;
  if (value.includes("node_group") || value.includes("nodegroup") || value.includes("fargate_profile")) return false;
  if (value === "aws_eks_cluster" || value.includes("eks_cluster")) return true;
  if (value.includes("ack") || value.includes("managed_kubernetes")) return true;
  if (value.includes("kubernetes_cluster")) return true;
  return false;
}

function defaultInputForBlueprint(blueprint, account, network) {
  const code = String(blueprint?.code || "").trim().toLowerCase();
  if (code.includes("bastion")) {
    return defaultBastionInput(account, network);
  }
  if (code.includes("server")) {
    return defaultServerInput(account, network);
  }
  if (code.includes("eks") || code.includes("ack")) {
    return defaultClusterInput(blueprint, account, network);
  }
  return defaultVpcInput(account);
}

function bindDeploymentJobProvider(form, datasets) {
  const providerField = form.querySelector("[name='provider']");
  const accountField = form.querySelector("[name='account_id']");
  const blueprintField = form.querySelector("[name='blueprint_id']");
  const networkField = form.querySelector("[name='network_plan_id']");
  const inputField = form.querySelector("[name='input_json']");
  const actionField = form.querySelector("[name='action']");
  const chargeTypeField = form.querySelector("[name='instance_charge_type']");
  const periodField = form.querySelector("[name='period']");
  const periodUnitField = form.querySelector("[name='period_unit']");
  const projectField = form.querySelector("[name='project_id']");
  const environmentField = form.querySelector("[name='environment_id']");
  if (!providerField || !accountField || !blueprintField || !networkField || !inputField || !actionField) return;

  const syncAliCloudBillingFields = (selectedBlueprint) => {
    const isAliCloudBastion = String(selectedBlueprint?.code || "").trim().toLowerCase() === "alicloud-bastion";
    chargeTypeField?.closest("label")?.classList.toggle("hidden", !isAliCloudBastion);
    const isPrePaid = isAliCloudBastion && String(chargeTypeField?.value || "PostPaid") === "PrePaid";
    periodField?.closest("label")?.classList.toggle("hidden", !isPrePaid);
    periodUnitField?.closest("label")?.classList.toggle("hidden", !isPrePaid);
    if (isAliCloudBastion) {
      if (chargeTypeField && !chargeTypeField.value) chargeTypeField.value = "PostPaid";
      if (periodField && !periodField.value) periodField.value = "1";
      if (periodUnitField && !periodUnitField.value) periodUnitField.value = "Month";
    }
  };

  const syncInputJSON = (selectedBlueprint, selectedAccount, selectedNetwork) => {
    if (!selectedBlueprint) return;
    const payload = defaultInputForBlueprint(selectedBlueprint, selectedAccount, selectedNetwork);
    if (String(selectedBlueprint.code || "").trim().toLowerCase() === "alicloud-bastion") {
      payload.instance_charge_type = String(chargeTypeField?.value || "PostPaid");
      if (payload.instance_charge_type === "PrePaid") {
        payload.period = Number(periodField?.value || 1);
        payload.period_unit = String(periodUnitField?.value || "Month");
      } else {
        delete payload.period;
        delete payload.period_unit;
      }
    }
    inputField.value = JSON.stringify(payload, null, 2);
  };

  const syncFromNetwork = () => {
    const selectedNetwork = (datasets.networks || []).find((item) => String(item.id) === String(networkField.value));
    if (!selectedNetwork) {
      showErrorDialog({ title: "缺少基础网络", copy: "请先选择有效的基础网络。" });
      return;
    }
    const provider = normalizeProvider(selectedNetwork.provider);
    const selectedAccount = (datasets.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork.account_id || 0));
    const blueprints = filterByProvider(datasets.blueprints, provider).filter((item) => String(item.category || "").toLowerCase() !== "network");
    if (projectField) {
      projectField.value = String(selectedNetwork.project_id || "");
    }
    if (environmentField) {
      environmentField.innerHTML = environmentOptions(selectedNetwork.project_id).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
      environmentField.value = String(selectedNetwork.environment_id || "");
    }
    providerField.innerHTML = providerOptions(datasets.blueprints).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    providerField.value = provider;
    accountField.innerHTML = accountOptions(selectedAccount ? [selectedAccount] : []).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    if (selectedAccount) {
      accountField.value = String(selectedAccount.id);
    }
    blueprintField.innerHTML = blueprintOptions(blueprints).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    const currentBlueprint = resolveBlueprintSelection(blueprints, blueprintField.value, datasets.preferredBlueprintCode);
    if (!currentBlueprint) {
      inputField.value = "{}";
      actionField.innerHTML = `<option value="plan">Plan Only</option>`;
      syncAliCloudBillingFields(null);
      return;
    }
    blueprintField.value = String(currentBlueprint.id);
    actionField.innerHTML = allowedActionOptions(currentBlueprint).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    updateBlueprintNote(form, currentBlueprint);
    syncAliCloudBillingFields(currentBlueprint);
    syncInputJSON(currentBlueprint, selectedAccount, selectedNetwork);
    const copyNode = form.closest(".modal-panel")?.querySelector(".modal-copy");
    if (copyNode) {
      copyNode.textContent = `${describeJobCopy(provider)} 当前会直接引用基础网络「${selectedNetwork.name}」的账号和网络上下文。`;
    }
  };

  networkField.addEventListener("change", syncFromNetwork);
  blueprintField.addEventListener("change", () => {
    const selectedNetwork = (datasets.networks || []).find((item) => String(item.id) === String(networkField.value));
    const selectedAccount = (datasets.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork?.account_id || 0));
    const selectedBlueprint = (datasets.blueprints || []).find((item) => String(item.id) === String(blueprintField.value));
    if (!selectedBlueprint) return;
    actionField.innerHTML = allowedActionOptions(selectedBlueprint).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    updateBlueprintNote(form, selectedBlueprint);
    syncAliCloudBillingFields(selectedBlueprint);
    syncInputJSON(selectedBlueprint, selectedAccount, selectedNetwork);
  });
  chargeTypeField?.addEventListener("change", () => {
    const selectedNetwork = (datasets.networks || []).find((item) => String(item.id) === String(networkField.value));
    const selectedAccount = (datasets.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork?.account_id || 0));
    const selectedBlueprint = (datasets.blueprints || []).find((item) => String(item.id) === String(blueprintField.value));
    syncAliCloudBillingFields(selectedBlueprint);
    syncInputJSON(selectedBlueprint, selectedAccount, selectedNetwork);
  });
  periodField?.addEventListener("change", () => {
    const selectedNetwork = (datasets.networks || []).find((item) => String(item.id) === String(networkField.value));
    const selectedAccount = (datasets.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork?.account_id || 0));
    const selectedBlueprint = (datasets.blueprints || []).find((item) => String(item.id) === String(blueprintField.value));
    syncInputJSON(selectedBlueprint, selectedAccount, selectedNetwork);
  });
  periodUnitField?.addEventListener("change", () => {
    const selectedNetwork = (datasets.networks || []).find((item) => String(item.id) === String(networkField.value));
    const selectedAccount = (datasets.accounts || []).find((item) => Number(item.id) === Number(selectedNetwork?.account_id || 0));
    const selectedBlueprint = (datasets.blueprints || []).find((item) => String(item.id) === String(blueprintField.value));
    syncInputJSON(selectedBlueprint, selectedAccount, selectedNetwork);
  });
  syncFromNetwork();
}

function projectOptions() {
  return (state.cloud.projects || []).map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }));
}

function environmentOptions(projectID) {
  return (state.cloud.environments || [])
    .filter((item) => String(item.project_id) === String(projectID || ""))
    .map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }));
}

function resolveBlueprintSelection(blueprints, currentBlueprintID, preferredBlueprintCode) {
  if (preferredBlueprintCode) {
    const preferred = (blueprints || []).find((item) => String(item.code || "").toLowerCase() === String(preferredBlueprintCode).toLowerCase());
    if (preferred) {
      return preferred;
    }
  }
  return (blueprints || []).find((item) => String(item.id) === String(currentBlueprintID)) || blueprints[0] || null;
}

function filterByProvider(items, provider) {
  return (items || []).filter((item) => normalizeProvider(item.provider) === normalizeProvider(provider));
}

function normalizeProvider(value) {
  return String(value || "").trim().toLowerCase();
}

function describeJobCopy(provider) {
  return normalizeProvider(provider) === "alicloud"
    ? "当前基础交付任务会围绕阿里云基础网络生成执行输入。绑定基础网络后，系统会自动带入 role-based refs；真正 Apply 前仍需要补全蓝图要求的云侧对象 ID。"
    : "当前基础交付任务会围绕 AWS 基础网络生成执行输入。绑定基础网络后，系统会自动带入 role-based refs；若已有成功的基础网络 outputs，会自动补全 Ops Bastion 所需的 VPC / Subnet ID。";
}

function resolveFoundationNetworkIDs(network, provider, mode = "ops") {
  const normalizedProvider = normalizeProvider(provider);
  if (!network?.id || !normalizedProvider) {
    return {};
  }
  const blueprintsByID = new Map((state.cloud.blueprints || []).map((item) => [String(item.id), item]));
  const candidates = (state.cloud.jobs || [])
    .filter((job) => normalizeProvider(job.provider) === normalizedProvider)
    .filter((job) => Number(job.network_plan_id) === Number(network.id))
    .filter((job) => String(job.status || "").toLowerCase() === "succeeded")
    .filter((job) => {
      const blueprint = blueprintsByID.get(String(job.blueprint_id));
      return blueprint?.category === "network";
    })
    .sort((left, right) => new Date(right.ended_at || right.created_at || 0).getTime() - new Date(left.ended_at || left.created_at || 0).getTime());

  const foundationJob = candidates[0];
  if (!foundationJob) {
    return {};
  }

  const outputs = foundationJob.output?.outputs || {};
  if (normalizedProvider === "alicloud") {
    const vpcID = terraformOutputString(outputs, "vpc_id");
    const opsVswitchIDs = terraformOutputStringMap(outputs, "ops_vswitch_ids");
    const vswitchID = resolveOpsProviderNetworkID(opsVswitchIDs, network.topology || {}, ["ops_vswitch_refs"]);
    return {
      vpc_id: vpcID,
      vswitch_id: vswitchID,
    };
  }

  const vpcID = terraformOutputString(outputs, "vpc_id");
  const privateSubnetIDs = terraformOutputStringMap(outputs, "private_subnet_ids");
  const subnetID = resolveOpsProviderNetworkID(
    privateSubnetIDs,
    network.topology || {},
    mode === "server" ? ["workload_subnet_refs", "ops_subnet_refs"] : ["ops_subnet_refs"],
  );
  return {
    vpc_id: vpcID,
    subnet_id: subnetID,
  };
}

function terraformOutputString(outputs, key) {
  const row = outputs?.[key];
  if (!row || typeof row !== "object") return "";
  return String(row.value || "").trim();
}

function terraformOutputStringMap(outputs, key) {
  const row = outputs?.[key];
  const value = row?.value;
  if (!value || typeof value !== "object") return {};
  return Object.entries(value).reduce((acc, [entryKey, entryValue]) => {
    const text = String(entryValue || "").trim();
    if (String(entryKey || "").trim() && text) {
      acc[entryKey] = text;
    }
    return acc;
  }, {});
}

function resolveOpsProviderNetworkID(providerIDs, topology, providerKeys) {
  for (const providerKey of providerKeys) {
    const refs = Array.isArray(topology?.provider_network_refs?.[providerKey])
      ? topology.provider_network_refs[providerKey]
      : [];
    for (const ref of refs) {
      const plannedNames = Array.isArray(ref?.planned_names) ? ref.planned_names : [];
      for (const plannedName of plannedNames) {
        const key = plannedNameToSubnetKey(plannedName);
        if (providerIDs[key]) {
          return providerIDs[key];
        }
      }
    }
  }
  const firstKey = Object.keys(providerIDs)[0];
  return firstKey ? providerIDs[firstKey] : "";
}

function plannedNameToSubnetKey(value) {
  const parts = String(value || "").trim().split("-").filter(Boolean);
  if (parts.length < 2) return "";
  return parts.slice(-2).join("-");
}

function allowedActionOptions(blueprint) {
  const options = [{ value: "plan", label: "Plan" }];
  if (supportsApply(blueprint)) {
    options.push({ value: "apply", label: "Apply" });
  }
  if (supportsDestroy(blueprint)) {
    options.push({ value: "destroy", label: "Destroy" });
  }
  if (options.length === 1) {
    options[0].label = "Plan Only";
  }
  return options;
}

function supportsApply(blueprint) {
  if (typeof blueprint?.supports_apply === "boolean") {
    return blueprint.supports_apply;
  }
  return ["apply_destroy_ready", "apply_ready", "experimental_apply"].includes(String(blueprint?.capability || "").toLowerCase());
}

function supportsDestroy(blueprint) {
  if (typeof blueprint?.supports_destroy === "boolean") {
    return blueprint.supports_destroy;
  }
  return ["apply_destroy_ready"].includes(String(blueprint?.capability || "").toLowerCase());
}

function blueprintMaturityCopy(blueprint) {
  if (!blueprint) return "请选择蓝图";
  const maturity = maturityLabel(String(blueprint.maturity || "experimental"));
  if (supportsDestroy(blueprint)) {
    return `${blueprint.name} 当前成熟度：${maturity}。允许 Plan / Apply / Destroy，真实执行仍需要显式确认。`;
  }
  if (supportsApply(blueprint)) {
    return `${blueprint.name} 当前成熟度：${maturity}。允许 Plan / Apply；Destroy 仍未放开。绑定基础网络后会自动带入基础网络 refs。`;
  }
  return `${blueprint.name} 当前成熟度：${maturity}。当前只允许 Plan，不允许 Apply / Destroy。`;
}

function updateBlueprintNote(form, blueprint) {
  const notes = form.querySelectorAll(".form-section-note");
  if (notes[0]) {
    notes[0].textContent = blueprintMaturityCopy(blueprint);
  }
}

function formatLogs(items) {
  if (!items.length) return "暂无执行日志";
  return items.map((item) => `[${formatDateTime(item.created_at)}] ${String(item.level || "info").toUpperCase()} ${item.stage || "-"} ${item.message || ""}`).join("\n");
}

function canCancel(item) {
  return ["queued", "claimed", "planning"].includes(String(item?.status || ""));
}

function canRetry(item) {
  return ["failed", "cancelled"].includes(String(item?.status || ""));
}

function canCleanup(item) {
  if (String(item?.action || "").toLowerCase() !== "apply") return false;
  if (String(item?.status || "").toLowerCase() !== "succeeded") return false;
  const blueprint = (state.cloud.blueprints || []).find((entry) => Number(entry.id) === Number(item.blueprint_id || 0));
  return supportsDestroy(blueprint);
}

function canReconfigure(item) {
  if (String(item?.action || "").toLowerCase() !== "apply") return false;
  if (!["succeeded", "failed", "cancelled"].includes(String(item?.status || "").toLowerCase())) return false;
  const blueprint = (state.cloud.blueprints || []).find((entry) => Number(entry.id) === Number(item.blueprint_id || 0));
  return ["aws-ec2-server", "alicloud-ecs-server"].includes(String(blueprint?.code || "").toLowerCase());
}

export async function createDestroyJobFromExisting(jobID) {
  const confirmed = await confirmAction({
    eyebrow: "Cloud Cleanup",
    title: "确认清理资源",
    copy: "这会直接在当前任务上切换到清理流程，并复用原有 workspace 执行 destroy。",
    confirmText: "确认清理",
  });
  if (!confirmed) return;
  await retryDeploymentJob(jobID, {
    action: "destroy",
    confirmed: true,
  });
  await refreshJobs();
  toast("当前任务已切换为清理流程");
}

export async function openUpdateServerJobModal(jobID) {
  if (!jobID) return;
  const response = await getDeploymentJob(jobID);
  const job = response?.data || {};
  const input = job?.input?.input || {};
  await openCreateServerDeliveryModal(refreshJobs, {
    existingJobId: jobID,
    existingJobName: job.name || "",
    initialAction: "apply",
    initialInput: input,
    preselectedNetworkId: job.network_plan_id,
  });
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
      return "Queued";
    case "claimed":
      return "Claimed";
    case "planning":
      return "Planning";
    case "planned":
      return "Planned";
    case "applying":
      return "Applying";
    case "succeeded":
      return "Succeeded";
    case "destroying":
      return "Destroying";
    case "destroyed":
      return "Destroyed";
    case "failed":
      return "Failed";
    case "cancelled":
      return "Cancelled";
    default:
      return String(value || "-");
  }
}

function resourceSyncStatusLabel(value) {
  switch (String(value || "").toLowerCase()) {
    case "pending":
      return "Pending";
    case "syncing":
      return "Syncing";
    case "synced":
      return "Synced";
    case "failed":
      return "Failed";
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
