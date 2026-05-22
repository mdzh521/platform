import { elements } from "../../core/dom.js";
import { renderScopeSummary } from "../../shared/scope.js";
import { state } from "../../core/state.js";
import { bindAction, confirmAction, openFormModal, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, formatDateTime } from "../../shared/utils.js";
import { createCloudAccount, deleteCloudAccount, testCloudAccount } from "./api.js";

export function renderAccounts(onChanged) {
  if (!elements.cloudAccountsTable) return;
  const items = state.cloud.accounts || [];
  if (!items.length) {
    elements.cloudAccountsTable.innerHTML = emptyState("暂无云账号");
    return;
  }

  const rows = items.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">${escapeHtml(item.provider)}</span></div></td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.region)}</strong><span class="table-meta">${escapeHtml(renderScopeSummary({ projectID: item.default_project_id, environmentID: item.default_environment_id }))}</span></div></td>
      <td>${escapeHtml(item.status)}</td>
      <td><div class="cell-stack"><span>AK ${item.has_access_key ? "已配置" : "未配置"} / SK ${item.has_secret_key ? "已配置" : "未配置"}</span><span class="table-meta">${escapeHtml(item.role_arn || "-")}</span></div></td>
      <td><div class="cell-stack"><span>${formatDateTime(item.last_checked_at)}</span><span class="table-meta"><button class="inline-link" data-action="cloud-test-account" data-id="${item.id}">测试连接</button></span></div></td>
      <td>
        <div class="action-row">
          <button class="ghost-button" data-action="cloud-delete-account" data-id="${item.id}">删除</button>
        </div>
      </td>
    </tr>
  `).join("");

  elements.cloudAccountsTable.innerHTML = `<table><thead><tr><th>序号</th><th>名称</th><th>区域</th><th>状态</th><th>凭据 / 角色</th><th>最近检测</th><th>操作</th></tr></thead><tbody>${rows}</tbody></table>`;
  bindAction(elements.cloudAccountsTable, "cloud-test-account", async (id) => {
    const payload = await testCloudAccount(id);
    const item = payload.data;
    const index = state.cloud.accounts.findIndex((account) => String(account.id) === String(id));
    if (index >= 0) state.cloud.accounts[index] = item;
    onChanged();
    toast(`云账号 ${item.name} 已完成基础校验`);
  });
  bindAction(elements.cloudAccountsTable, "cloud-delete-account", async (id) => {
    const item = state.cloud.accounts.find((account) => String(account.id) === String(id));
    if (!item) return;
    const confirmed = await confirmAction({
      eyebrow: "Cloud Cleanup",
      title: "确认删除云账号",
      copy: `将删除云账号“${item.name}”。如果该账号下还有基础网络、基础交付任务或资源台账，系统会拒绝删除。`,
      confirmText: "确认删除",
    });
    if (!confirmed) return;
    await deleteCloudAccount(id);
    await onChanged();
    toast(`云账号 ${item.name} 已删除`);
  });
}

export function openCreateAccountModal(onCreated) {
  const projects = state.cloud.projects || [];
  const projectOptions = projects.length
    ? projects.map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }))
    : [{ value: "", label: "暂不绑定" }];
  const defaultProjectID = projects[0]?.id ? String(projects[0].id) : "";
  const initialEnvironments = listProjectEnvironments(defaultProjectID);
  openFormModal({
    eyebrow: "Cloud",
    title: "新增云账号",
    copy: "先落最小闭环：记录云平台凭据、默认区域和角色信息，SecretKey 不回显。",
    fields: [
      { label: "云账号名称", name: "name", required: true },
      { label: "云平台", name: "provider", type: "select", value: "aws", options: [{ value: "aws", label: "AWS" }, { value: "alicloud", label: "阿里云" }] },
      { label: "AccessKey", name: "access_key", required: true },
      { label: "SecretKey", name: "secret_key", type: "password", required: true },
      { label: "默认区域", name: "region", required: true, value: "ap-southeast-1" },
      { label: "默认 Project", name: "default_project_id", type: "select", value: defaultProjectID, options: [{ value: "", label: "暂不绑定" }, ...projectOptions] },
      { label: "默认 Environment", name: "default_environment_id", type: "select", value: initialEnvironments[0]?.value || "", options: [{ value: "", label: "暂不绑定" }, ...initialEnvironments] },
      { label: "Role ARN", name: "role_arn", showForProvider: "aws", placeholder: "arn:aws:iam::123456789012:role/platform-center" },
      { label: "默认标签", name: "default_tags", placeholder: "env=dev,owner=platform" },
      { label: "默认可用区", name: "default_zones", placeholder: "ap-southeast-1a,ap-southeast-1b" },
    ],
    onSubmit: async (form) => {
      await createCloudAccount({
        name: form.get("name"),
        provider: form.get("provider"),
        access_key: form.get("access_key"),
        secret_key: form.get("secret_key"),
        region: form.get("region"),
        default_project_id: Number(form.get("default_project_id") || 0) || null,
        default_environment_id: Number(form.get("default_environment_id") || 0) || null,
        role_arn: form.get("role_arn"),
        default_tags: splitByComma(form.get("default_tags")),
        default_zones: splitByComma(form.get("default_zones")),
      });
      await onCreated();
      toast("云账号已创建");
    },
    onOpen: bindAccountProjectScope,
  });
}

function splitByComma(value) {
  return String(value || "").split(",").map((item) => item.trim()).filter(Boolean);
}

function bindAccountProjectScope(form) {
  const projectField = form.querySelector("[name='default_project_id']");
  const environmentField = form.querySelector("[name='default_environment_id']");
  if (!projectField || !environmentField) return;
  const syncEnvironments = () => {
    const options = listProjectEnvironments(projectField.value);
    environmentField.innerHTML = [{ value: "", label: "暂不绑定" }, ...options]
      .map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`)
      .join("");
  };
  projectField.addEventListener("change", syncEnvironments);
  syncEnvironments();
}

function listProjectEnvironments(projectID) {
  return (state.cloud.environments || [])
    .filter((item) => String(item.project_id) === String(projectID || ""))
    .map((item) => ({ value: String(item.id), label: `${item.name} · ${item.code}` }));
}
