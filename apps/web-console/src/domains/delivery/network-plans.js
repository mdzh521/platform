import { state } from "../../core/state.js";
import { elements } from "../../core/dom.js";
import { renderScopeBadges, renderScopeSummary } from "../../shared/scope.js";
import { confirmAction, openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import { createCloudKeyPair, createDeploymentJob, createMachineCredential, createNetworkPlan, deleteMachineCredential, deleteNetworkPlan, getMachineCredential, importCredentialToMachineAsset, updateNetworkPlan } from "./api.js";

let refreshNetworkPlans = async () => {};

export function renderNetworkPlans(onRefresh = async () => {}) {
  refreshNetworkPlans = onRefresh;
  if (!elements.cloudNetworkPlansTable) return;
  const items = state.cloud.networkPlans || [];
  if (!items.length) {
    elements.cloudNetworkPlansTable.innerHTML = emptyState("暂无 Foundation Network");
    return;
  }

  elements.cloudNetworkPlansTable.innerHTML = `
    <div class="network-plan-gallery">
      ${items.map((item, index) => renderNetworkPlanCard(item, index)).join("")}
    </div>
  `;
  bindNetworkPlanActions();
}

export function openCreateNetworkPlanModal(onCreated) {
  openNetworkPlanFormModal({ onCreated });
}

function openEditNetworkPlanModal(plan, onCreated) {
  openNetworkPlanFormModal({ plan, onCreated });
}

function openNetworkPlanFormModal({ plan = null, onCreated }) {
  const accounts = state.cloud.accounts || [];
  if (!accounts.length) {
    showErrorDialog({ title: "缺少云账号", copy: "请先新增并保存一个云账号，再创建 Foundation Network。" });
    return;
  }
  const topology = plan?.topology || {};
  const isEdit = Boolean(plan);
  const initialProvider = normalizeProvider(plan?.provider || accounts[0]?.provider || "aws");
  const initialAccounts = accounts.filter((item) => normalizeProvider(item.provider) === initialProvider);
  const defaultAccount = initialAccounts[0] || accounts[0];
  const initialPreset = defaultPresetForProvider(initialProvider);
  const projects = state.cloud.projects || [];
  const projectOptions = projects.map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }));
  const initialProjectID = String(plan?.project_id || accounts[0]?.default_project_id || projects[0]?.id || "");
  const initialEnvironmentOptions = listEnvironmentOptions(initialProjectID);
  const initialEnvironmentID = String(plan?.environment_id || accounts[0]?.default_environment_id || initialEnvironmentOptions[0]?.value || "");

  openFormModal({
    eyebrow: "Cloud",
    title: isEdit ? "编辑 Foundation Network" : "新增 Foundation Network",
    copy: isEdit ? describeNetworkPlanCopy(initialProvider) : `${describeNetworkPlanCopy(initialProvider)} 保存后会自动创建并执行 Foundation 交付任务。`,
    submitText: isEdit ? "保存网络方案" : "创建并执行 Foundation",
    fields: [
      { label: "预置模板", name: "preset_template", type: "select", value: initialPreset, options: presetOptions(initialProvider) },
      { type: "note", copy: templateCatalogCopy(initialProvider, initialPreset) },
      { label: "Foundation 名称", name: "name", required: true, value: plan?.name || "" },
      { label: "云平台", name: "provider", type: "select", value: initialProvider, options: providerOptions(accounts) },
      { label: "云账号", name: "account_id", type: "select", value: String(plan?.account_id || defaultAccount.id), options: accountOptions(initialAccounts) },
      { label: "Project", name: "project_id", type: "select", value: initialProjectID, options: projectOptions },
      { label: "Environment", name: "environment_id", type: "select", value: initialEnvironmentID, options: initialEnvironmentOptions },
      { label: "区域", name: "region", value: plan?.region || defaultAccount.region, required: true },
      { label: "环境标识", name: "environment", value: topology.environment || "dev" },
      { label: "VPC / 命名代号", name: "vpc_name", value: topology.vpc_name || defaultVpcNameForProvider(initialProvider), required: true },
      { label: "VPC CIDR", name: "vpc_cidr", value: plan?.vpc_cidr || defaultVPCCIDR(initialProvider), required: true },
      { label: availabilityZoneLabel(initialProvider), name: "availability_zones", type: "text", value: String(topology.availability_zone_count || topology.availability_zones || 2) },
      { label: subnetGroupsLabel(initialProvider), name: "subnet_groups", type: "textarea", rows: 8, value: formatSubnetGroupsEditorValue(topology.subnet_groups, initialProvider) || defaultSubnetGroupsText(initialProvider), placeholder: subnetGroupsPlaceholder(initialProvider) },
      { type: "note", copy: networkPlanGuide(initialProvider) },
      { label: natGatewayLabel(initialProvider), name: "nat_gateway_count", type: "text", value: String(topology.nat_gateway_count ?? 1) },
      { label: "默认标签", name: "default_tags", type: "textarea", rows: 3, value: formatStringList(topology.default_tags) || "project=platform-center\nenvironment=dev\nowner=platform-team" },
      { label: "安全基线", name: "security_baseline", type: "select", value: topology.security_baseline || "standard", options: [{ value: "standard", label: "standard" }, { value: "strict", label: "strict" }] },
      { label: bastionToggleLabel(initialProvider), name: "create_bastion_subnet", type: "select", value: String(topology.create_bastion_subnet ?? true), options: [{ value: "true", label: "是" }, { value: "false", label: "否" }] },
      { label: bastionCIDRLabel(initialProvider), name: "bastion_subnet_cidr", value: topology.bastion_subnet_cidr || defaultBastionCIDR(initialProvider) },
    ],
    onSubmit: async (form) => {
      const payload = {
        name: form.get("name"),
        provider: form.get("provider"),
        account_id: Number(form.get("account_id")),
        project_id: Number(form.get("project_id") || 0) || null,
        environment_id: Number(form.get("environment_id") || 0) || null,
        region: form.get("region"),
        environment: form.get("environment"),
        vpc_name: form.get("vpc_name"),
        vpc_cidr: form.get("vpc_cidr"),
        availability_zones: Number(form.get("availability_zones") || 2),
        subnet_groups: parseSubnetGroups(form.get("subnet_groups"), form.get("provider")),
        default_tags: parseLines(form.get("default_tags")),
        nat_gateway_count: Number(form.get("nat_gateway_count") || 0),
        security_baseline: form.get("security_baseline"),
        create_bastion_subnet: String(form.get("create_bastion_subnet")) === "true",
        bastion_subnet_cidr: form.get("bastion_subnet_cidr"),
      };
      if (isEdit) {
        await updateNetworkPlan(plan.id, payload);
        await onCreated();
        toast("Foundation Network 已更新");
      } else {
        const response = await createNetworkPlan(payload);
        const createdPlan = response?.data;
        await createFoundationExecutionForPlan(payload, createdPlan);
        await onCreated();
        toast("Foundation Network 已创建，并已自动发起执行");
      }
    },
    onOpen: (form) => {
      bindNetworkPlanPreset(form, isEdit);
      bindProjectEnvironmentScope(form);
    },
  });
}

function accountOptions(items) {
  return items.map((item) => ({ value: String(item.id), label: `${item.name} · ${item.provider} · ${item.region}` }));
}

function providerOptions(items) {
  const existing = new Set(items.map((item) => normalizeProvider(item.provider)).filter(Boolean));
  return [
    { value: "aws", label: existing.has("aws") ? "AWS" : "AWS（未配置账号）" },
    { value: "alicloud", label: existing.has("alicloud") ? "阿里云" : "阿里云（未配置账号）" },
  ];
}

function listEnvironmentOptions(projectID) {
  return (state.cloud.environments || [])
    .filter((item) => String(item.project_id) === String(projectID || ""))
    .map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }));
}

function bindProjectEnvironmentScope(form) {
  const projectField = form.querySelector("[name='project_id']");
  const environmentField = form.querySelector("[name='environment_id']");
  const environmentCodeField = form.querySelector("[name='environment']");
  if (!projectField || !environmentField) return;
  const syncEnvironments = () => {
    const options = listEnvironmentOptions(projectField.value);
    environmentField.innerHTML = options.map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    if (!options.find((item) => item.value === String(environmentField.value || ""))) {
      environmentField.value = options[0]?.value || "";
    }
    syncEnvironmentCode();
  };
  const syncEnvironmentCode = () => {
    const selected = (state.cloud.environments || []).find((item) => String(item.id) === String(environmentField.value || ""));
    if (environmentCodeField && selected?.code) {
      environmentCodeField.value = selected.code;
    }
  };
  projectField.addEventListener("change", syncEnvironments);
  environmentField.addEventListener("change", syncEnvironmentCode);
  syncEnvironments();
}

async function createFoundationExecutionForPlan(payload, plan) {
  const provider = normalizeProvider(payload.provider);
  const blueprint = findFoundationBlueprint(provider);
  if (!blueprint || !plan?.id) {
    throw new Error("未找到可执行的 Foundation Blueprint");
  }
  const action = supportsApply(blueprint) ? "apply" : "plan";
  await createDeploymentJob({
    name: `${String(payload.name || "foundation").trim()}-${action}`,
    provider,
    account_id: Number(payload.account_id),
    blueprint_id: Number(blueprint.id),
    network_plan_id: Number(plan.id),
    project_id: Number(plan.project_id || payload.project_id || 0) || null,
    environment_id: Number(plan.environment_id || payload.environment_id || 0) || null,
    action,
    confirmed: action !== "plan",
    input: buildFoundationExecutionInput(payload),
  });
}

function buildFoundationExecutionInput(payload) {
  return {
    region: String(payload.region || "").trim(),
    project_id: Number(payload.project_id || 0) || null,
    environment_id: Number(payload.environment_id || 0) || null,
    environment: String(payload.environment || "dev").trim(),
    vpc_name: String(payload.vpc_name || "").trim(),
    vpc_cidr: String(payload.vpc_cidr || "").trim(),
    availability_zone_count: Number(payload.availability_zones || 2),
    availability_zones: Number(payload.availability_zones || 2),
    subnet_groups: Array.isArray(payload.subnet_groups) ? payload.subnet_groups : [],
    nat_gateway_count: Number(payload.nat_gateway_count || 0),
    create_bastion_subnet: Boolean(payload.create_bastion_subnet),
    bastion_subnet_cidr: String(payload.bastion_subnet_cidr || "").trim(),
    security_baseline: String(payload.security_baseline || "standard").trim(),
    default_tags: parseTagObject(payload.default_tags),
  };
}

function parseTagObject(lines) {
  return (Array.isArray(lines) ? lines : []).reduce((result, item) => {
    const [key, ...rest] = String(item || "").split("=");
    const normalizedKey = String(key || "").trim();
    if (!normalizedKey) return result;
    result[normalizedKey] = rest.join("=").trim();
    return result;
  }, {});
}

function findFoundationBlueprint(provider) {
  return (state.cloud.blueprints || []).find((item) =>
    normalizeProvider(item.provider) === provider && String(item.category || "").toLowerCase() === "network"
  );
}

function supportsApply(blueprint) {
  if (!blueprint) return false;
  if (typeof blueprint.supports_apply === "boolean") {
    return blueprint.supports_apply;
  }
  return ["apply_destroy_ready", "apply_ready", "experimental_apply"].includes(String(blueprint.capability || "").toLowerCase());
}

function parseLines(value) {
  return String(value || "")
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseSubnetGroups(value, provider = "aws") {
  const normalizedProvider = normalizeProvider(provider);
  return parseLines(value)
    .map((line) => {
      const [role, middle, tail] = line.split("|").map((item) => item.trim());
      const hasThreeColumns = typeof tail === "string" && tail !== "";
      const tier = normalizedProvider === "aws" && hasThreeColumns ? String(middle || "") : normalizedProvider === "alicloud" ? "" : String(middle || "");
      const trafficProfile = normalizedProvider === "alicloud"
        ? (hasThreeColumns ? String(middle || "") : "")
        : "";
      return {
        role,
        tier,
        traffic_profile: trafficProfile,
        cidrs: String(hasThreeColumns ? tail : middle || "")
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
      };
    })
    .filter((item) => item.role && item.cidrs.length);
}

function defaultSubnetGroupsText(provider = "aws") {
  return provider === "alicloud"
    ? [
      "slb | internet-entry | 10.20.0.0/24, 10.20.1.0/24",
      "ack-node | cluster-node | 10.20.10.0/24, 10.20.11.0/24",
      "ack-pod | pod-network | 10.20.20.0/24, 10.20.21.0/24",
      "application | workload | 10.20.30.0/24, 10.20.31.0/24",
      "database | data | 10.20.40.0/24, 10.20.41.0/24",
      "ops | ops | 10.20.50.0/24, 10.20.51.0/24",
    ].join("\n")
    : [
      "ingress | public | 10.10.0.0/24, 10.10.1.0/24",
      "middleware | private | 10.10.10.0/24, 10.10.11.0/24",
      "database | private | 10.10.20.0/24, 10.10.21.0/24",
      "k8s | private | 10.10.30.0/24, 10.10.31.0/24",
      "ops | private | 10.10.40.0/24, 10.10.41.0/24",
    ].join("\n");
}

function presetOptions(provider = "aws") {
  if (provider === "alicloud") {
    return [
      { value: "alicloud-ack-standard", label: "阿里云 ACK 标准网络" },
      { value: "alicloud-application-stack", label: "阿里云业务分层网络" },
      { value: "alicloud-data-platform", label: "阿里云数据平台网络" },
    ];
  }
  return [
    { value: "k8s-standard", label: "K8s 标准网络" },
    { value: "application-stack", label: "业务应用分层网络" },
    { value: "data-platform", label: "数据平台网络" },
  ];
}

function bindNetworkPlanPreset(form, preserveExisting = false) {
  const providerField = form.querySelector("[name='provider']");
  const accountField = form.querySelector("[name='account_id']");
  const regionField = form.querySelector("[name='region']");
  const presetField = form.querySelector("[name='preset_template']");
  const subnetGroupsField = form.querySelector("[name='subnet_groups']");
  const natCountField = form.querySelector("[name='nat_gateway_count']");
  const bastionToggleField = form.querySelector("[name='create_bastion_subnet']");
  const bastionCIDRField = form.querySelector("[name='bastion_subnet_cidr']");
  const vpcNameField = form.querySelector("[name='vpc_name']");
  if (!presetField || !subnetGroupsField || !providerField || !accountField || !regionField) return;

  ensureSubnetPreview(form);
  renderNetworkPlanFormMeta(form, providerField.value, presetField.value);

  const applyPreset = (force = false) => {
    if (preserveExisting && !force) {
      renderSubnetPreview(form);
      return;
    }
    const preset = networkPreset(providerField.value, presetField.value);
    if (!preset) return;
    subnetGroupsField.value = preset.subnetGroups;
    if (natCountField) natCountField.value = String(preset.natGatewayCount);
    if (bastionToggleField) bastionToggleField.value = preset.createBastionSubnet ? "true" : "false";
    if (bastionCIDRField) bastionCIDRField.value = preset.bastionSubnetCIDR;
    renderSubnetPreview(form);
  };

  const syncAccountsForProvider = (force = false) => {
    const provider = normalizeProvider(providerField.value);
    const accounts = (state.cloud.accounts || []).filter((item) => normalizeProvider(item.provider) === provider);
    if (!accounts.length) {
      accountField.innerHTML = `<option value="">请先新增${escapeHtml(provider === "alicloud" ? "阿里云" : "AWS")}账号</option>`;
      accountField.value = "";
      regionField.value = "";
    } else {
      accountField.innerHTML = accountOptions(accounts).map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    }
    if (accounts.length && (force || !accounts.some((item) => String(item.id) === String(accountField.value)))) {
      accountField.value = String(accounts[0].id);
      regionField.value = accounts[0].region || "";
    }
    const nextPresetOptions = presetOptions(provider);
    presetField.innerHTML = nextPresetOptions.map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
    presetField.value = defaultPresetForProvider(provider);
    if (force && vpcNameField && !preserveExisting) {
      vpcNameField.value = defaultVpcNameForProvider(provider);
    }
    const subnetGroupsWrapper = subnetGroupsField.closest(".form-field");
    const bastionToggleWrapper = bastionToggleField?.closest(".form-field");
    const bastionCIDRWrapper = bastionCIDRField?.closest(".form-field");
    const availabilityZoneWrapper = form.querySelector("[name='availability_zones']")?.closest(".form-field");
    const natGatewayWrapper = natCountField?.closest(".form-field");
    subnetGroupsWrapper?.querySelector("label")?.replaceChildren(subnetGroupsLabel(provider));
    bastionToggleWrapper?.querySelector("label")?.replaceChildren(bastionToggleLabel(provider));
    bastionCIDRWrapper?.querySelector("label")?.replaceChildren(bastionCIDRLabel(provider));
    availabilityZoneWrapper?.querySelector("label")?.replaceChildren(availabilityZoneLabel(provider));
    natGatewayWrapper?.querySelector("label")?.replaceChildren(natGatewayLabel(provider));
    subnetGroupsField.placeholder = subnetGroupsPlaceholder(provider);
    form.querySelector(".form-section-note")?.replaceChildren(networkPlanGuide(provider));
    if (force && bastionCIDRField && !preserveExisting) {
      bastionCIDRField.value = defaultBastionCIDR(provider);
    }
    renderNetworkPlanFormMeta(form, provider, presetField.value);
  };

  providerField.addEventListener("change", () => {
    syncAccountsForProvider(true);
    applyPreset(true);
  });
  accountField.addEventListener("change", () => {
    const selected = (state.cloud.accounts || []).find((item) => String(item.id) === String(accountField.value));
    if (selected) {
      regionField.value = selected.region || regionField.value;
      if (normalizeProvider(selected.provider) !== normalizeProvider(providerField.value)) {
        providerField.value = normalizeProvider(selected.provider);
        syncAccountsForProvider(false);
      }
      renderNetworkPlanFormMeta(form, providerField.value, presetField.value);
    }
  });
  presetField.addEventListener("change", () => {
    renderNetworkPlanFormMeta(form, providerField.value, presetField.value);
    applyPreset(true);
  });
  subnetGroupsField.addEventListener("input", () => renderSubnetPreview(form));
  vpcNameField?.addEventListener("input", () => renderSubnetPreview(form));
  bastionToggleField?.addEventListener("change", () => renderSubnetPreview(form));
  bastionCIDRField?.addEventListener("input", () => renderSubnetPreview(form));
  syncAccountsForProvider(false);
  applyPreset(false);
}

function networkPreset(provider, code) {
  if (normalizeProvider(provider) === "alicloud") {
    switch (code) {
      case "alicloud-application-stack":
        return {
          subnetGroups: [
            "slb | internet-entry | 10.20.0.0/24, 10.20.1.0/24",
            "application | workload | 10.20.10.0/24, 10.20.11.0/24",
            "middleware | workload | 10.20.20.0/24, 10.20.21.0/24",
            "database | data | 10.20.30.0/24, 10.20.31.0/24",
            "ops | ops | 10.20.40.0/24, 10.20.41.0/24",
          ].join("\n"),
          natGatewayCount: 1,
          createBastionSubnet: true,
          bastionSubnetCIDR: "10.20.90.0/24",
        };
      case "alicloud-data-platform":
        return {
          subnetGroups: [
            "slb | internet-entry | 10.20.0.0/24, 10.20.1.0/24",
            "etl | workload | 10.20.10.0/24, 10.20.11.0/24",
            "warehouse | data | 10.20.20.0/24, 10.20.21.0/24",
            "cache | data | 10.20.30.0/24, 10.20.31.0/24",
            "ops | ops | 10.20.40.0/24, 10.20.41.0/24",
          ].join("\n"),
          natGatewayCount: 2,
          createBastionSubnet: true,
          bastionSubnetCIDR: "10.20.90.0/24",
        };
      case "alicloud-ack-standard":
      default:
        return {
          subnetGroups: defaultSubnetGroupsText("alicloud"),
          natGatewayCount: 1,
          createBastionSubnet: true,
          bastionSubnetCIDR: "10.20.90.0/24",
        };
    }
  }
  switch (code) {
    case "application-stack":
      return {
        subnetGroups: [
          "ingress | public | 10.10.0.0/24, 10.10.1.0/24",
          "application | private | 10.10.10.0/24, 10.10.11.0/24",
          "middleware | private | 10.10.20.0/24, 10.10.21.0/24",
          "database | private | 10.10.30.0/24, 10.10.31.0/24",
          "ops | private | 10.10.40.0/24, 10.10.41.0/24",
        ].join("\n"),
        natGatewayCount: 1,
        createBastionSubnet: true,
        bastionSubnetCIDR: "10.10.90.0/24",
      };
    case "data-platform":
      return {
        subnetGroups: [
          "ingress | public | 10.10.0.0/24, 10.10.1.0/24",
          "etl | private | 10.10.10.0/24, 10.10.11.0/24",
          "warehouse | private | 10.10.20.0/24, 10.10.21.0/24",
          "cache | private | 10.10.30.0/24, 10.10.31.0/24",
          "ops | private | 10.10.40.0/24, 10.10.41.0/24",
        ].join("\n"),
        natGatewayCount: 2,
        createBastionSubnet: true,
        bastionSubnetCIDR: "10.10.90.0/24",
      };
    case "k8s-standard":
    default:
      return {
        subnetGroups: defaultSubnetGroupsText("aws"),
        natGatewayCount: 1,
        createBastionSubnet: true,
        bastionSubnetCIDR: "10.10.90.0/24",
      };
  }
}

function bindNetworkPlanActions() {
  elements.cloudNetworkPlansTable.querySelectorAll("[data-network-plan-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      openNetworkPlanDetailModal(Number(button.dataset.networkPlanDetail || 0));
    });
  });
  elements.cloudNetworkPlansTable.querySelectorAll("[data-network-plan-edit]").forEach((button) => {
    button.addEventListener("click", () => {
      const plan = (state.cloud.networkPlans || []).find((item) => Number(item.id) === Number(button.dataset.networkPlanEdit || 0));
      if (!plan) return;
      openEditNetworkPlanModal(plan, refreshNetworkPlans);
    });
  });
  elements.cloudNetworkPlansTable.querySelectorAll("[data-network-plan-delete]").forEach((button) => {
    button.addEventListener("click", async () => {
      const plan = (state.cloud.networkPlans || []).find((item) => Number(item.id) === Number(button.dataset.networkPlanDelete || 0));
      if (!plan) return;
      const confirmed = await confirmAction({
        eyebrow: "Cloud Cleanup",
        title: "确认删除 Foundation Network",
        copy: `将删除 Foundation Network“${plan.name}”。如果仍有关联的 Delivery Job 或资源台账，系统会拒绝删除。`,
        confirmText: "确认删除",
      });
      if (!confirmed) return;
      await deleteNetworkPlan(plan.id);
      await refreshNetworkPlans();
      toast(`Foundation Network ${plan.name} 已删除`);
    });
  });
  elements.cloudNetworkPlansTable.querySelectorAll("[data-network-plan-key-manage]").forEach((button) => {
    button.addEventListener("click", () => {
      openNetworkKeyModal(Number(button.dataset.networkPlanKeyManage || 0));
    });
  });
}

function openNetworkPlanDetailModal(networkPlanID) {
  const plan = (state.cloud.networkPlans || []).find((item) => Number(item.id) === Number(networkPlanID));
  if (!plan) return;

  const topology = plan.topology || {};
  const subnetGroups = Array.isArray(topology.subnet_groups) ? topology.subnet_groups : [];
  const provider = normalizeProvider(plan.provider);
  const deliveryActions = buildNetworkDeliveryActions(networkPlanID, provider);

  openFormModal({
    eyebrow: "Foundation Network",
    title: `Foundation Network 详情 · ${plan.name || `#${networkPlanID}`}`,
    copy: `${plan.account_name || "-"} · ${plan.provider || "-"} · ${plan.region || "-"} · 创建于 ${formatDateTime(plan.created_at)}`,
    submitText: false,
    fields: [
      {
        type: "actions",
        actions: [
          ...deliveryActions,
          { label: "密钥管理", shortcut: `network-key:${networkPlanID}` },
          { label: "创建通用 Delivery Job", shortcut: `create-job-with-network:${networkPlanID}` },
          { label: "查看 Resources", shortcut: "scroll-resources" },
          { label: "返回 Foundation 列表", shortcut: "scroll-networks" },
        ],
      },
      {
        label: "基础信息",
        type: "section",
        eyebrow: "Overview",
        copy: `VPC：${plan.vpc_cidr || "-"} ｜ 环境：${topology.environment || "-"} ｜ 命名代号：${topology.vpc_name || "-"} ｜ NAT：${topology.nat_gateway_count ?? "-"}`,
      },
      {
        label: "Foundation 引用契约",
        name: "foundation_refs_json",
        type: "textarea",
        rows: 12,
        readOnly: true,
        value: JSON.stringify(buildFoundationContractView(topology), null, 2),
      },
      {
        label: subnetPreviewLabel(provider),
        name: "subnet_preview",
        type: "textarea",
        rows: Math.max(12, subnetGroups.length * 4),
        readOnly: true,
        value: formatSubnetGroupsPreview(topology, provider),
      },
      {
        label: "拓扑 JSON",
        name: "topology_json",
        type: "textarea",
        rows: 14,
        readOnly: true,
        value: JSON.stringify(topology, null, 2),
      },
    ],
  });
}

function renderNetworkPlanCard(item, index) {
  const topology = item.topology || {};
  const provider = normalizeProvider(item.provider);
  const groups = normalizedSubnetGroups(topology, provider);
  const projectCredentials = listProjectCredentials(item.id);
  const roleBadges = groups.slice(0, 6).map((group) => `<span class="network-plan-badge">${escapeHtml(group.role)}</span>`).join("");
  const extraCount = groups.length > 6 ? `<span class="network-plan-badge">+${groups.length - 6}</span>` : "";
  const deliveryActions = buildNetworkDeliveryActions(item.id, provider)
    .map((action) => `
      <button class="${action.tone === "primary" ? "primary-button" : "ghost-button"}" data-cloud-shortcut="${escapeHtml(action.shortcut)}">${escapeHtml(action.label)}</button>
    `)
    .join("");

  return `
    <article class="network-plan-card ${provider}" data-network-plan-card="${item.id}">
      <div class="network-plan-card-head">
        <div>
          <span class="muted-label">#${index + 1} · ${escapeHtml(providerLabel(provider))}</span>
          <h5>${escapeHtml(item.name)}</h5>
          <p>${escapeHtml(item.account_name || "-")} · ${escapeHtml(item.region || "-")} · ${escapeHtml(item.vpc_cidr || "-")} · ${escapeHtml(item.status || "-")}</p>
          <p class="table-meta">${escapeHtml(renderScopeSummary({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id, fallbackEnvironment: topology.environment }))}</p>
        </div>
        <div class="action-row">
          <button class="ghost-button" data-network-plan-detail="${item.id}">详情</button>
          <button class="ghost-button" data-network-plan-key-manage="${item.id}">密钥管理</button>
          <button class="ghost-button" data-network-plan-edit="${item.id}">编辑</button>
          <button class="ghost-button danger" data-network-plan-delete="${item.id}">删除</button>
        </div>
      </div>
      <div class="network-plan-card-meta">
        <span class="network-plan-badge">${escapeHtml(topology.environment || "dev")}</span>
        <span class="network-plan-badge">${escapeHtml(topology.vpc_name || item.name || "foundation")}</span>
        <span class="network-plan-badge">${escapeHtml(item.resource_brief || `${groups.length} groups`)}</span>
        <span class="network-plan-badge">${escapeHtml(`项目密钥 ${projectCredentials.length} 把`)}</span>
      </div>
      <div class="scope-badge-row">
        ${renderScopeBadges({ projectID: item.project_id, environmentID: item.environment_id, stackID: item.stack_id, fallbackEnvironment: topology.environment })}
      </div>
      <div class="network-plan-card-roles">
        ${roleBadges}
        ${extraCount}
      </div>
      <div class="action-row">
        ${deliveryActions}
        <button class="ghost-button" data-network-plan-key-manage="${item.id}">项目密钥</button>
      </div>
    </article>
  `;
}

export function openNetworkKeyModal(networkPlanID) {
  const plan = (state.cloud.networkPlans || []).find((item) => Number(item.id) === Number(networkPlanID));
  if (!plan) return;
  const account = (state.cloud.accounts || []).find((item) => Number(item.id) === Number(plan.account_id || 0));
  if (!account) {
    showErrorDialog({ title: "缺少云账号", copy: "当前 Foundation Network 没有绑定有效云账号。" });
    return;
  }
  const provider = normalizeProvider(plan.provider);
  const topology = plan.topology || {};
  const projectName = String(topology.vpc_name || plan.name || "foundation").trim() || "foundation";
  const environment = String(topology.environment || "dev").trim() || "dev";
  const defaultKeyName = `${projectName}-${environment}-key`;
  const supportsCloudCreate = provider === "aws" || provider === "alicloud";
  const bastionOptions = listBastionAssetsForNetwork(networkPlanID, provider);
  const projectCredentials = listProjectCredentials(networkPlanID);
  const defaultRegisterBastion = "false";

  openFormModal({
    eyebrow: "Cloud Key",
    title: `密钥管理 · ${plan.name}`,
    copy: `当前作用域：${plan.account_name || "-"} · ${providerLabel(provider)} · ${plan.region || account.region || "-"}。建议在每条 Foundation Network 下维护项目级密钥，后续创建服务器和 Bastion 时直接复用。`,
    submitText: "保存密钥",
    fields: [
      {
        type: "section",
        eyebrow: "Scope",
        label: "Foundation 范围",
        copy: `Foundation Network：${plan.name} ｜ 项目标识：${projectName} ｜ 环境：${environment}`,
      },
      {
        label: "密钥来源",
        name: "key_mode",
        type: "select",
        value: supportsCloudCreate ? "cloud_create" : "manual",
        options: supportsCloudCreate
          ? [
              { value: "cloud_create", label: "在云上创建并回填私钥" },
              { value: "manual", label: "手工录入现有私钥" },
            ]
          : [
              { value: "manual", label: "手工录入现有私钥" },
            ],
      },
      { label: "密钥名称 / Key Pair 名称", name: "key_name", required: true, value: defaultKeyName },
      { label: "登录用户", name: "credential_username", value: provider === "aws" ? "ec2-user" : "root" },
      { label: "录入机器管理凭据库", name: "save_credential", type: "select", value: "true", options: [{ value: "true", label: "是" }, { value: "false", label: "否" }] },
      { label: "同时绑定到 Ops Bastion", name: "register_bastion_credential", type: "select", value: defaultRegisterBastion, options: [{ value: "true", label: "是" }, { value: "false", label: "否" }] },
      {
        label: "目标 Ops Bastion",
        name: "bastion_asset_id",
        type: "select",
        value: bastionOptions[0]?.value || "",
        options: [{ value: "", label: bastionOptions.length ? "不绑定 Bastion，只存到机器管理凭据库" : "当前 Foundation 下还没有可绑定的 Ops Bastion，仍可先保存到机器管理凭据库" }, ...bastionOptions],
      },
      {
        type: "section",
        eyebrow: "Machine Access",
        label: "录入说明",
        copy: "项目密钥默认保存到机器管理凭据库。只有在你希望某台 Ops Bastion 直接复用这把密钥时，才需要额外选择“同时绑定到 Ops Bastion”。",
      },
      {
        type: "custom",
        html: supportsCloudCreate
          ? `<div class="cloud-key-inline-actions"><button type="button" class="ghost-button" data-network-key-create>在 ${escapeHtml(providerLabel(provider))} 创建密钥</button><span class="muted-label">创建后会自动回填下面的私钥内容。</span></div>`
          : `<div class="cloud-key-inline-actions"><span class="muted-label">当前云平台暂不支持直接通过平台创建云密钥，请手工录入已有私钥。</span></div>`,
      },
      {
        type: "custom",
        html: renderProjectCredentialSummary(projectCredentials),
      },
      { label: "SSH 私钥", name: "private_key_material", type: "textarea", rows: 10, placeholder: "创建云密钥后会自动回填，或者手工粘贴 PEM 私钥" },
      { label: "私钥口令", name: "private_key_passphrase", type: "password", placeholder: "如果私钥带口令可填写" },
      { label: "备注", name: "key_note", value: `${plan.name} · ${projectName} · ${environment}` },
    ],
    onOpen: (form) => bindNetworkKeyForm(form, { account, plan, provider, projectCredentials }),
    onSubmit: async (form) => {
      const saveCredential = String(form.get("save_credential") || "true") === "true";
      const registerToBastion = String(form.get("register_bastion_credential") || "true") === "true";
      const keyName = String(form.get("key_name") || "").trim();
      const privateKey = String(form.get("private_key_material") || "").trim();
      const bastionAssetID = Number(form.get("bastion_asset_id") || 0) || 0;
      const selectedBastion = (state.cloud.machineAssets || []).find((item) => Number(item.id) === bastionAssetID);
      if (!keyName) {
        throw new Error("密钥名称不能为空");
      }
      if (registerToBastion && !bastionAssetID) {
        throw new Error("如果选择绑定 Ops Bastion，请先选择目标 Ops Bastion");
      }
      let createdCredential = null;
      if (saveCredential) {
        const savedCredentialID = Number(elements.formModalForm?.dataset?.savedCredentialId || 0) || 0;
        if (savedCredentialID) {
          createdCredential = { id: savedCredentialID };
        } else {
          if (!privateKey) {
            throw new Error("录入平台凭据库时，必须提供私钥内容");
          }
          const response = await createMachineCredential({
            name: keyName,
            username: String(form.get("credential_username") || (provider === "aws" ? "ec2-user" : "root")).trim() || (provider === "aws" ? "ec2-user" : "root"),
            auth_type: "ssh_key",
            private_key: privateKey,
            passphrase: String(form.get("private_key_passphrase") || "").trim(),
            source_type: "cloud_project_key",
            provider,
            scope_kind: "foundation_network",
            scope_ref: String(plan.id),
            scope_name: plan.name,
            asset_id: bastionAssetID || null,
            asset_name: selectedBastion?.name || "",
            description: String(form.get("key_note") || `${plan.name} · ${keyName}`).trim(),
          });
          createdCredential = response?.data || null;
        }
      }
      if (registerToBastion) {
        if (!createdCredential?.id) {
          throw new Error("绑定 Ops Bastion 需要先保存到机器管理凭据库");
        }
        await importCredentialToMachineAsset(bastionAssetID, {
          credential_id: Number(createdCredential.id),
          is_default: true,
        });
      }
      toast(registerToBastion ? "项目密钥已保存到机器管理凭据库，并已绑定到选定 Ops Bastion 的默认账号" : "项目密钥已保存到机器管理凭据库");
      await refreshNetworkPlans();
    },
  });
}

function bindNetworkKeyForm(form, { account, plan, provider, projectCredentials = [] }) {
  const keyModeField = form.querySelector("[name='key_mode']");
  const keyNameField = form.querySelector("[name='key_name']");
  const usernameField = form.querySelector("[name='credential_username']");
  const noteField = form.querySelector("[name='key_note']");
  const privateKeyField = form.querySelector("[name='private_key_material']");
  const passphraseField = form.querySelector("[name='private_key_passphrase']");
  const saveCredentialField = form.querySelector("[name='save_credential']");
  const registerBastionField = form.querySelector("[name='register_bastion_credential']");
  const bastionField = form.querySelector("[name='bastion_asset_id']");
  const createButton = form.querySelector("[data-network-key-create]");
  const submitButton = form.querySelector("button[type='submit']");
  const privateKeyWrapper = privateKeyField?.closest(".form-field");
  const bastionWrapper = bastionField?.closest("[data-field-name='bastion_asset_id']");

  const syncKeyMode = () => {
    const mode = String(keyModeField?.value || "manual");
    if (privateKeyWrapper) {
      privateKeyWrapper.querySelector("label")?.replaceChildren(mode === "cloud_create" ? "SSH 私钥（创建后自动回填）" : "SSH 私钥");
    }
  };
  const syncBastionBinding = () => {
    const visible = String(registerBastionField?.value || "false") === "true";
    bastionWrapper?.classList.toggle("hidden", !visible);
    if (!visible && bastionField) {
      bastionField.value = "";
    }
  };

  keyModeField?.addEventListener("change", syncKeyMode);
  registerBastionField?.addEventListener("change", syncBastionBinding);
  syncKeyMode();
  syncBastionBinding();

  form.querySelectorAll("[data-project-key-use]").forEach((button) => {
    button.addEventListener("click", () => {
      const credentialID = Number(button.dataset.projectKeyUse || 0);
      const credential = projectCredentials.find((item) => Number(item.id) === credentialID);
      if (!credential) return;
      if (keyNameField) keyNameField.value = credential.name || "";
      if (usernameField) usernameField.value = credential.username || "";
      if (noteField) noteField.value = credential.description || "";
      if (bastionField) {
        bastionField.value = credential.asset_id ? String(credential.asset_id) : "";
      }
      if (registerBastionField) {
        registerBastionField.value = credential.asset_id ? "true" : "false";
        syncBastionBinding();
      }
      if (privateKeyField) {
        privateKeyField.focus();
        privateKeyField.select();
      }
      toast("已带入当前项目密钥配置，请补充或替换私钥后保存");
    });
  });

  form.querySelectorAll("[data-project-key-view]").forEach((button) => {
    button.addEventListener("click", async () => {
      const credentialID = Number(button.dataset.projectKeyView || 0);
      if (!credentialID) return;
      await openProjectCredentialDetailModal(credentialID);
    });
  });

  form.querySelectorAll("[data-project-key-delete]").forEach((button) => {
    button.addEventListener("click", async () => {
      const credentialID = Number(button.dataset.projectKeyDelete || 0);
      const credential = projectCredentials.find((item) => Number(item.id) === credentialID);
      if (!credentialID || !credential) return;
      const confirmed = await confirmAction({
        eyebrow: "Project Key",
        title: "确认删除项目密钥",
        copy: `将删除项目密钥“${credential.name}”。如果这把密钥已经导入某台 Bastion，平台凭据库记录会删除，但已导入的 Bastion 账号不会自动回收。`,
        confirmText: "确认删除",
      });
      if (!confirmed) return;
      await deleteMachineCredential(credentialID);
      toast(`项目密钥 ${credential.name} 已删除`);
      await refreshNetworkPlans();
      openNetworkKeyModal(plan.id);
    });
  });

  createButton?.addEventListener("click", async () => {
    const keyName = String(keyNameField?.value || "").trim();
    if (!keyName) {
      await showErrorDialog({ title: "缺少密钥名称", copy: "请先填写 Key Pair 名称。" });
      return;
    }
    try {
      createButton.disabled = true;
      const response = await createCloudKeyPair(Number(account.id), {
        provider,
        region: plan.region || account.region || "",
        name: keyName,
      });
      const data = response?.data || {};
      const privateKey = String(data.private_key || "");
      if (privateKeyField) privateKeyField.value = privateKey;
      const shouldSaveCredential = String(saveCredentialField?.value || "true") === "true";
      const shouldRegisterBastion = String(registerBastionField?.value || "false") === "true";
      const bastionAssetID = Number(bastionField?.value || 0) || 0;
      const selectedBastion = (state.cloud.machineAssets || []).find((item) => Number(item.id) === bastionAssetID);

      if (shouldSaveCredential) {
        if (!privateKey) {
          throw new Error("云密钥已创建，但没有返回私钥内容，无法同步到机器管理凭据库");
        }
        const credentialResponse = await createMachineCredential({
          name: data.name || keyName,
          username: String(usernameField?.value || (provider === "aws" ? "ec2-user" : "root")).trim() || (provider === "aws" ? "ec2-user" : "root"),
          auth_type: "ssh_key",
          private_key: privateKey,
          passphrase: String(passphraseField?.value || "").trim(),
          source_type: "cloud_project_key",
          provider,
          scope_kind: "foundation_network",
          scope_ref: String(plan.id),
          scope_name: plan.name,
          asset_id: shouldRegisterBastion ? (bastionAssetID || null) : null,
          asset_name: shouldRegisterBastion ? (selectedBastion?.name || "") : "",
          description: String(noteField?.value || `${plan.name} · ${data.name || keyName}`).trim(),
        });
        const createdCredential = credentialResponse?.data || null;
        if (createdCredential?.id) {
          form.dataset.savedCredentialId = String(createdCredential.id);
          if (shouldRegisterBastion) {
            if (!bastionAssetID) {
              throw new Error("如果选择绑定 Ops Bastion，请先选择目标 Ops Bastion");
            }
            await importCredentialToMachineAsset(bastionAssetID, {
              credential_id: Number(createdCredential.id),
              is_default: true,
            });
          }
        }
        await refreshNetworkPlans();
        if (submitButton) {
          submitButton.textContent = "已保存到机器管理凭据库";
        }
        toast(shouldRegisterBastion ? `云密钥 ${data.name || keyName} 已创建，并已同步到机器管理凭据库和选定 Ops Bastion` : `云密钥 ${data.name || keyName} 已创建，并已同步到机器管理凭据库`);
        return;
      }

      toast(`云密钥 ${data.name || keyName} 已创建`);
    } catch (error) {
      await showErrorDialog({ title: "创建密钥失败", copy: error?.message || "云密钥创建失败" });
    } finally {
      createButton.disabled = false;
    }
  });
}

function listProjectCredentials(networkPlanID) {
  return (state.cloud.machineCredentials || []).filter((item) =>
    String(item.scope_kind || "").toLowerCase() === "foundation_network" &&
    String(item.scope_ref || "") === String(networkPlanID)
  );
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

function renderProjectCredentialSummary(credentials) {
  if (!credentials.length) {
    return `
      <div class="cloud-project-key-summary">
        <div class="cloud-project-key-summary-head">
          <strong>当前项目密钥</strong>
          <span class="muted-label">0 把</span>
        </div>
        <p>当前这条 Foundation Network 还没有维护项目级密钥。建议先在这里创建，再给服务器和 Ops Bastion 复用。</p>
      </div>
    `;
  }
  return `
    <div class="cloud-project-key-summary">
      <div class="cloud-project-key-summary-head">
        <strong>当前项目密钥</strong>
        <span class="muted-label">${escapeHtml(String(credentials.length))} 把</span>
      </div>
      <div class="cloud-project-key-list">
        ${credentials.slice(0, 5).map((item) => `
          <div class="cloud-project-key-item">
            <div>
              <strong>${escapeHtml(item.name || "-")}</strong>
              <p>${escapeHtml(item.username || "-")} · ${escapeHtml(item.asset_name || "未绑定 Bastion")}</p>
            </div>
            <div class="cloud-project-key-item-side">
              <span class="muted-label">${escapeHtml(formatDateTime(item.updated_at || item.created_at))}</span>
              <div class="action-row compact-action-row">
                <button type="button" class="ghost-button" data-project-key-view="${item.id}">查看详情</button>
                <button type="button" class="ghost-button" data-project-key-use="${item.id}">带入配置</button>
                <button type="button" class="ghost-button danger-soft" data-project-key-delete="${item.id}">删除</button>
              </div>
            </div>
          </div>
        `).join("")}
      </div>
    </div>
  `;
}

async function openProjectCredentialDetailModal(credentialID) {
  const response = await getMachineCredential(credentialID);
  const credential = response?.data || {};
  openFormModal({
    eyebrow: "Project Key",
    title: `查看项目密钥 · ${credential.name || "密钥"}`,
    copy: "这里显示项目级密钥的明文内容。仅在需要核对、导入或排障时查看。",
    submitText: false,
    fields: [
      { label: "密钥名称", name: "name", value: credential.name || "", readOnly: true },
      { label: "登录用户", name: "username", value: credential.username || "", readOnly: true },
      { label: "云平台", name: "provider", value: credential.provider || "", readOnly: true },
      { label: "作用域", name: "scope_name", value: credential.scope_name || "", readOnly: true },
      { label: "绑定 Bastion", name: "asset_name", value: credential.asset_name || "", readOnly: true },
      { label: "认证方式", name: "auth_type", value: credential.auth_type || "", readOnly: true },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 10, value: credential.private_key || "", readOnly: true },
      { label: "私钥口令", name: "passphrase", value: credential.passphrase || "", readOnly: true },
      { label: "备注", name: "description", type: "textarea", rows: 3, value: credential.description || "", readOnly: true },
    ],
  });
}

function buildNetworkDeliveryActions(networkPlanID, provider) {
  const actions = [];
  if (hasBlueprint(provider, "server")) {
    actions.push({ label: "创建服务器", shortcut: `create-server-with-network:${networkPlanID}`, tone: "primary" });
  }
  if (hasBlueprint(provider, "bastion")) {
    actions.push({ label: "创建 Ops Bastion", shortcut: `create-bastion-with-network:${networkPlanID}` });
  }
  if (hasBlueprint(provider, "cluster")) {
    actions.push({ label: "创建 Cluster", shortcut: `create-cluster-with-network:${networkPlanID}` });
  }
  return actions;
}

function hasBlueprint(provider, categoryOrCodeFragment) {
  const normalizedProvider = normalizeProvider(provider);
  const target = String(categoryOrCodeFragment || "").toLowerCase();
  return (state.cloud.blueprints || []).some((item) => {
    if (normalizeProvider(item.provider) !== normalizedProvider) return false;
    return String(item.category || "").toLowerCase() === target || String(item.code || "").toLowerCase().includes(target);
  });
}

function formatSubnetGroupsPreview(topology, provider = "aws") {
  const vpcName = String(topology?.vpc_name || "platform-core-dev").trim() || "platform-core-dev";
  const groups = Array.isArray(topology?.subnet_groups) ? topology.subnet_groups : [];
  if (!groups.length) {
    return normalizeProvider(provider) === "alicloud" ? "暂无交换机角色规划" : "暂无子网角色规划";
  }
  return groups.map((group) => {
    const role = String(group.role || "-").trim();
    const tier = String(group.tier || "-").trim();
    const trafficProfile = String(group.traffic_profile || "").trim();
    const cidrs = Array.isArray(group.cidrs) ? group.cidrs : [];
    const lines = cidrs.map((cidr, index) => `  ${vpcName}-${role}-${String(index + 1).padStart(2, "0")}    ${cidr}`);
    const header = normalizeProvider(provider) === "alicloud"
      ? `[${role}] traffic_profile=${trafficProfile || "-"}`
      : `[${role}] tier=${displayTier(provider, tier)}`;
    return [header, ...lines].join("\n");
  }).join("\n\n");
}

function buildFoundationContractView(topology) {
  return {
    foundation_stack_name: topology?.foundation_stack_name || topology?.vpc_name || "",
    name_prefix: topology?.name_prefix || topology?.vpc_name || "",
    network_role_refs: topology?.network_role_refs || {},
    internet_entry_refs: topology?.internet_entry_refs || [],
    egress_refs: topology?.egress_refs || [],
    workload_refs: topology?.workload_refs || [],
    data_refs: topology?.data_refs || [],
    ops_refs: topology?.ops_refs || [],
    cluster_node_refs: topology?.cluster_node_refs || [],
    pod_network_refs: topology?.pod_network_refs || [],
    provider_network_refs: topology?.provider_network_refs || {},
  };
}

function normalizeProvider(value) {
  return String(value || "").trim().toLowerCase();
}

function defaultPresetForProvider(provider) {
  return normalizeProvider(provider) === "alicloud" ? "alicloud-ack-standard" : "k8s-standard";
}

function defaultVpcNameForProvider(provider) {
  return normalizeProvider(provider) === "alicloud" ? "platform-ali-dev" : "platform-core-dev";
}

function defaultBastionCIDR(provider) {
  return normalizeProvider(provider) === "alicloud" ? "10.20.90.0/24" : "10.10.90.0/24";
}

function defaultVPCCIDR(provider) {
  return normalizeProvider(provider) === "alicloud" ? "10.20.0.0/16" : "10.10.0.0/16";
}

function describeNetworkPlanCopy(provider) {
  return normalizeProvider(provider) === "alicloud"
    ? "阿里云 Foundation Network 采用 VPC + 交换机设计。请优先使用 role + traffic_profile 表达，例如 SLB 入口、ACK 节点、ACK Pod、应用、数据库、运维接入，而不是默认使用 public/private。若当前还没有阿里云账号，先到“云账号”里新增。"
    : "这版 Foundation Network 支持按子网角色批量命名。用“角色 | tier | CIDR列表”定义公网入口、中间件、数据库、K8s、运维等子网，后续模板会自动生成子网名称。若当前还没有 AWS 账号，先到“云账号”里新增。";
}

function renderNetworkPlanFormMeta(form, provider) {
  const copyNode = form.closest(".modal-panel")?.querySelector(".modal-copy");
  if (copyNode) {
    copyNode.textContent = describeNetworkPlanCopy(provider);
  }
  const presetField = form.querySelector("[name='preset_template']");
  const templateNote = form.querySelector(".form-section-note");
  if (templateNote && presetField) {
    const notes = form.querySelectorAll(".form-section-note");
    if (notes[0]) {
      notes[0].textContent = templateCatalogCopy(provider, presetField.value);
    }
    if (notes[1]) {
      notes[1].textContent = networkPlanGuide(provider);
    }
  }
  const previewTitle = form.querySelector("[data-subnet-preview] strong");
  const previewCopy = form.querySelector("[data-subnet-preview] p");
  if (previewTitle) {
    previewTitle.textContent = normalizeProvider(provider) === "alicloud" ? "交换机命名预览" : "子网命名预览";
  }
  if (previewCopy) {
    previewCopy.textContent = normalizeProvider(provider) === "alicloud"
      ? "根据当前 VPC 命名代号和交换机角色，系统会自动生成下面这些交换机名称。"
      : "根据当前 VPC 命名代号和子网角色，系统会自动生成下面这些子网名称。";
  }
}

function subnetGroupsLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "交换机分层规划" : "子网角色规划";
}

function bastionToggleLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "创建运维接入交换机" : "创建跳板机子网";
}

function bastionCIDRLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "运维接入交换机 CIDR" : "跳板机子网 CIDR";
}

function subnetPreviewLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "交换机角色与命名预览" : "子网角色与命名预览";
}

function availabilityZoneLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "可用区数量" : "AZ 数量";
}

function natGatewayLabel(provider) {
  return normalizeProvider(provider) === "alicloud" ? "NAT 出口规划数量" : "NAT 网关数量";
}

function subnetGroupsPlaceholder(provider) {
  return normalizeProvider(provider) === "alicloud"
    ? "slb | internet-entry | 10.20.0.0/24, 10.20.1.0/24\nack-node | cluster-node | 10.20.10.0/24, 10.20.11.0/24\nack-pod | pod-network | 10.20.20.0/24, 10.20.21.0/24\napplication | workload | 10.20.30.0/24, 10.20.31.0/24"
    : "ingress | public | 10.10.0.0/24, 10.10.1.0/24\nmiddleware | private | 10.10.10.0/24, 10.10.11.0/24\ndatabase | private | 10.10.20.0/24, 10.10.21.0/24";
}

function networkPlanGuide(provider) {
  return normalizeProvider(provider) === "alicloud"
    ? "阿里云建议至少拆出 SLB 入口交换机、ACK 节点交换机、ACK Pod 交换机、数据库交换机和运维接入交换机；请使用 role + traffic_profile 表达，不再默认使用 public/private。"
    : "AWS 建议按入口子网、应用/中间件子网、数据库子网和运维子网分层；公网入口和 NAT 通常依附在 public/private 子网设计上。";
}

function displayTier(provider, tier) {
  if (normalizeProvider(provider) === "alicloud") {
    return tier === "public" ? "edge" : "internal";
  }
  return tier;
}

function templateCatalogCopy(provider, preset) {
  const item = presetCatalog(provider)[preset] || presetCatalog(provider)[defaultPresetForProvider(provider)];
  if (!item) return "";
  return `${item.name}：${item.copy} 示例分层：${item.roles.join(" / ")}`;
}

function presetCatalog(provider) {
  if (normalizeProvider(provider) === "alicloud") {
    return {
      "alicloud-ack-standard": {
        name: "阿里云 ACK 标准网络",
        copy: "围绕 SLB 入口、ACK 节点和 Pod 网络、应用和运维接入做标准分层。",
        roles: ["slb", "ack-node", "ack-pod", "application", "database", "ops"],
      },
      "alicloud-application-stack": {
        name: "阿里云业务分层网络",
        copy: "适合传统业务系统，突出 SLB 入口、应用、中间件、数据库和运维接入。",
        roles: ["slb", "application", "middleware", "database", "ops"],
      },
      "alicloud-data-platform": {
        name: "阿里云数据平台网络",
        copy: "适合数仓与数据处理场景，突出 ETL、仓库、缓存和运维分层。",
        roles: ["slb", "etl", "warehouse", "cache", "ops"],
      },
    };
  }
  return {
    "k8s-standard": {
      name: "K8s 标准网络",
      copy: "适合 AWS 上的 K8s / EKS 规划，突出入口、中间件、数据库、K8s 和运维分层。",
      roles: ["ingress", "middleware", "database", "k8s", "ops"],
    },
    "application-stack": {
      name: "业务应用分层网络",
      copy: "适合传统业务系统，强调入口、应用、中间件、数据库和运维子网。",
      roles: ["ingress", "application", "middleware", "database", "ops"],
    },
    "data-platform": {
      name: "数据平台网络",
      copy: "适合数据类平台，强调 ETL、仓库、缓存和运维隔离。",
      roles: ["ingress", "etl", "warehouse", "cache", "ops"],
    },
  };
}

function formatSubnetGroupsEditorValue(groups, provider = "aws") {
  if (!Array.isArray(groups) || !groups.length) return "";
  return groups.map((group) => {
    const role = String(group.role || "").trim();
    const tier = String(group.tier || "private").trim();
    const trafficProfile = String(group.traffic_profile || "").trim();
    const cidrs = Array.isArray(group.cidrs) ? group.cidrs.join(", ") : "";
    if (!role || !cidrs) return "";
    return normalizeProvider(provider) === "alicloud"
      ? `${role} | ${trafficProfile || "workload"} | ${cidrs}`
      : `${role} | ${tier || "private"} | ${cidrs}`;
  }).filter(Boolean).join("\n");
}

function formatStringList(items) {
  return Array.isArray(items) ? items.join("\n") : "";
}

function ensureSubnetPreview(form) {
  if (form.querySelector("[data-subnet-preview]")) return;
  const submitButton = form.querySelector("button[type='submit']");
  const block = document.createElement("div");
  block.className = "form-section-heading field-span-2 subnet-preview";
  block.dataset.subnetPreview = "true";
  block.innerHTML = `
    <span class="eyebrow">Naming Preview</span>
    <strong>子网命名预览</strong>
    <p>根据当前 VPC 命名代号和子网角色，系统会自动生成下面这些子网名称。</p>
    <div class="network-topology-preview" data-network-topology-preview></div>
  `;
  if (submitButton) {
    form.insertBefore(block, submitButton);
  } else {
    form.appendChild(block);
  }
}

function renderSubnetPreview(form) {
  const subnetGroupsField = form.querySelector("[name='subnet_groups']");
  if (!subnetGroupsField) return;

  const provider = normalizeProvider(form.querySelector("[name='provider']")?.value || "aws");
  const vpcName = String(form.querySelector("[name='vpc_name']")?.value || "platform-core-dev").trim() || "platform-core-dev";
  const createBastion = String(form.querySelector("[name='create_bastion_subnet']")?.value || "false") === "true";
  const bastionCIDR = String(form.querySelector("[name='bastion_subnet_cidr']")?.value || "").trim();
  const groups = parseSubnetGroups(subnetGroupsField.value, provider);
  const topologyPreview = form.querySelector("[data-network-topology-preview]");

  if (createBastion && bastionCIDR) {
    groups.push({ role: "bastion", tier: "private", cidrs: [bastionCIDR] });
  }

  if (!groups.length) {
    if (topologyPreview) {
      topologyPreview.innerHTML = `<div class="subnet-preview-empty">${escapeHtml(normalizeProvider(provider) === "alicloud" ? "暂无可预览交换机，请先填写交换机角色规划。" : "暂无可预览子网，请先填写子网角色规划。")}</div>`;
    }
    return;
  }

  if (topologyPreview) {
    topologyPreview.innerHTML = renderNetworkTopologyDiagram({
      provider,
      vpcName,
      vpcCIDR: String(form.querySelector("[name='vpc_cidr']")?.value || "").trim(),
      groups: normalizedSubnetGroups({ subnet_groups: groups }, provider),
      createBastion,
      bastionCIDR,
    });
  }
}

function renderNetworkTopologyDiagram({ provider = "aws", vpcName = "foundation", vpcCIDR = "", groups = [], createBastion = false, bastionCIDR = "" }) {
  const normalizedProvider = normalizeProvider(provider);
  const graphGroups = Array.isArray(groups) ? groups : [];
  const accent = normalizedProvider === "alicloud" ? "alicloud" : "aws";
  const chips = graphGroups.map((group) => `
    <article class="network-role-chip ${roleTone(group, normalizedProvider)}">
      <div class="network-role-chip-head">
        <strong>${escapeHtml(group.role)}</strong>
        <span>${escapeHtml(groupLabel(group, normalizedProvider))}</span>
      </div>
      <div class="network-role-chip-lines">
        ${(group.cidrs || []).map((cidr, index) => `<div><span>${escapeHtml(`${vpcName}-${group.role}-${String(index + 1).padStart(2, "0")}`)}</span><code>${escapeHtml(cidr)}</code></div>`).join("")}
      </div>
    </article>
  `).join("");

  const bastionBlock = createBastion && bastionCIDR
    ? `
      <article class="network-role-chip ops">
        <div class="network-role-chip-head">
          <strong>${normalizedProvider === "alicloud" ? "ops-access" : "bastion"}</strong>
          <span>${escapeHtml(normalizedProvider === "alicloud" ? "ops" : "private")}</span>
        </div>
        <div class="network-role-chip-lines">
          <div><span>${escapeHtml(`${vpcName}-${normalizedProvider === "alicloud" ? "ops-access" : "bastion"}-01`)}</span><code>${escapeHtml(bastionCIDR)}</code></div>
        </div>
      </article>
    `
    : "";

  return `
    <section class="network-topology-shell ${accent}">
      <div class="network-topology-head">
        <div>
          <span class="muted-label">${escapeHtml(providerLabel(normalizedProvider))}</span>
          <strong>${escapeHtml(vpcName)}</strong>
          <p>${escapeHtml(vpcCIDR || "-")}</p>
        </div>
        <div class="network-topology-legend">
          <span>${escapeHtml(normalizedProvider === "alicloud" ? "交换机角色图" : "子网角色图")}</span>
        </div>
      </div>
      <div class="network-topology-canvas">
        ${chips}
        ${bastionBlock}
      </div>
    </section>
  `;
}

function normalizedSubnetGroups(topology, provider = "aws") {
  const groups = Array.isArray(topology?.subnet_groups) ? topology.subnet_groups : [];
  return groups.map((group) => ({
    role: String(group.role || "-").trim(),
    tier: String(group.tier || "").trim(),
    traffic_profile: String(group.traffic_profile || "").trim(),
    cidrs: Array.isArray(group.cidrs) ? group.cidrs : [],
  })).filter((group) => group.role && group.cidrs.length);
}

function groupLabel(group, provider = "aws") {
  const normalizedProvider = normalizeProvider(provider);
  if (normalizedProvider === "alicloud") {
    return String(group.traffic_profile || group.tier || "-");
  }
  return String(group.tier || group.traffic_profile || "-");
}

function roleTone(group, provider = "aws") {
  const role = String(group.role || "").toLowerCase();
  const label = groupLabel(group, provider).toLowerCase();
  if (role.includes("ingress") || role.includes("slb") || label.includes("public") || label.includes("internet-entry")) return "edge";
  if (role.includes("db") || role.includes("data") || label.includes("data")) return "data";
  if (role.includes("ops") || role.includes("bastion") || label.includes("ops")) return "ops";
  if (role.includes("k8s") || role.includes("ack") || role.includes("pod") || label.includes("cluster") || label.includes("pod")) return "cluster";
  return "workload";
}

function providerLabel(provider = "aws") {
  return normalizeProvider(provider) === "alicloud" ? "阿里云" : "AWS";
}
