import { api } from "../../core/api.js";
import { state } from "../../core/state.js";
import { confirmAction, openFormModal, toast } from "../../core/ui.js";
import {
  assetAccessModeOptions,
  assetGroupOptions,
  gatewayNodeOptions,
  loginPolicyOptions,
  platformOptions,
  protocolOptions,
  statusOptions,
} from "./helpers.js";

export function openQuickConnectModal(machineCtx, assetID = 0) {
  const asset = state.machine.assets.find((item) => Number(item.id) === Number(assetID));
  openFormModal({
    eyebrow: asset ? "Quick Connect" : "Create Asset",
    title: asset ? `登录 ${asset.name}` : "快速登录机器",
    copy: asset
      ? "基于资产信息创建一次登录会话，后续再接入真实 SSH / RDP 在线代理。"
      : "先录入机器入口，再统一往会话审计和凭据托管演进，整体思路参考 JumpServer 的资产中心。",
    submitText: asset ? "准备登录" : "保存并登录",
    fields: asset ? [
      { label: "资产名称", name: "name", value: asset.name },
      { label: "机器地址", name: "address", value: asset.address },
      { label: "平台", name: "platform", type: "select", value: asset.platform, options: platformOptions() },
      { label: "协议", name: "protocol", type: "select", value: asset.protocol, options: protocolOptions() },
      { label: "端口", name: "port", type: "number", value: String(asset.port || "") },
      { label: "登录账号", name: "account", value: asset.account || "" },
    ] : [
      { label: "资产名称", name: "name", placeholder: "例如 prod-gateway-01" },
      { label: "机器地址", name: "address", required: true, placeholder: "10.0.0.12 / host.example.com" },
      { label: "平台", name: "platform", type: "select", value: "linux", options: platformOptions() },
      { label: "协议", name: "protocol", type: "select", value: "ssh", options: protocolOptions() },
      { label: "端口", name: "port", type: "number", value: "22" },
      { label: "登录账号", name: "account", placeholder: "root / ops-admin" },
      { label: "资产说明", name: "description", type: "textarea", rows: 4, placeholder: "用途、环境、归属团队" },
    ],
    onSubmit: async (form) => {
      const payload = asset ? { asset_id: asset.id } : {
        name: form.get("name"),
        address: form.get("address"),
        platform: form.get("platform"),
        protocol: form.get("protocol"),
        port: Number(form.get("port") || 0),
        account: form.get("account"),
        description: form.get("description"),
        save_to_assets: true,
      };
      const response = await api("/api/v1/machines/quick-connect", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      await machineCtx.refreshMachineData();
      toast(response.data?.message || "机器登录入口已准备");
    },
  });
}

export function openAssetModal(machineCtx, asset = null) {
  const isEdit = Boolean(asset?.id);
  const createFields = [
    { type: "section", eyebrow: "Asset", label: "资产信息", copy: "先录入机器本身的信息，后续登录和审计都围绕这台资产展开。" },
    { label: "资产名称", name: "name", required: true, value: asset?.name || "", placeholder: "例如 prod-app-01" },
    { label: "机器地址", name: "address", required: true, value: asset?.address || "", placeholder: "10.0.0.12 / host.example.com" },
    { label: "平台", name: "platform", type: "select", value: asset?.platform || "linux", options: platformOptions() },
    { label: "协议", name: "protocol", type: "select", value: asset?.protocol || "ssh", options: protocolOptions() },
    { label: "访问方式", name: "access_mode", type: "select", value: asset?.access_mode || "direct", options: assetAccessModeOptions() },
    { label: "入口节点 / 跳板机", name: "gateway_asset_id", type: "select", value: asset?.gateway_asset_id ? String(asset.gateway_asset_id) : "", options: gatewayNodeOptions(asset?.id || 0) },
    { label: "作为入口节点", name: "is_gateway_node", type: "select", value: asset?.is_gateway_node ? "true" : "false", options: [{ value: "false", label: "否" }, { value: "true", label: "是" }] },
    { label: "资产分组", name: "group_name", type: "select", value: asset?.group_name || "", options: assetGroupOptions() },
    { label: "登录策略", name: "login_policy", type: "select", value: asset?.login_policy || "inherit_group", options: loginPolicyOptions() },
    { label: "端口", name: "port", type: "number", value: String(asset?.port || 22) },
    { label: "登录账号", name: "account", value: asset?.account || "", placeholder: "root / ops-admin" },
    { label: "状态", name: "status", type: "select", value: asset?.status || "online", options: statusOptions() },
    { label: "标签", name: "tags", value: Array.isArray(asset?.tags) ? asset.tags.join(", ") : "", placeholder: "prod, app, core" },
    { label: "资产说明", name: "description", type: "textarea", rows: 4, value: asset?.description || "", placeholder: "用途、环境、归属团队" },
  ];

  const credentialFields = [
    { type: "section", eyebrow: "Credentials", label: "登录凭据", copy: "如果你已经知道这台机器的账号密码或 SSH 私钥，可以在这里一起保存。创建完成后会自动绑定到该资产。" },
    { label: "使用已有凭据", name: "credential_library_id", type: "select", value: "", options: [{ value: "", label: "不使用，直接录入新凭据" }, ...(state.machine.credentials || []).map((item) => ({ value: String(item.id), label: `${item.name} · ${item.username}` }))] },
    { label: "凭据显示名称", name: "credential_name", placeholder: "例如 Linux 运维账号" },
    { label: "凭据登录账号", name: "credential_username", placeholder: "ops-admin" },
    { label: "认证方式", name: "credential_auth_type", type: "select", value: "password", options: [{ value: "password", label: "密码" }, { value: "ssh_key", label: "SSH 私钥" }] },
    { label: "登录密码", name: "credential_password", type: "password", placeholder: "密码型账号填写这里" },
    { label: "SSH 私钥", name: "credential_private_key", type: "textarea", rows: 6, placeholder: "私钥型账号填写完整 PEM 内容", upload: { accept: ".pem,.key,.txt,*/*", hint: "支持上传 .pem / .key，本地内容会自动填入" } },
    { label: "私钥口令", name: "credential_passphrase", type: "password", placeholder: "如果私钥带口令，在这里填写" },
    { label: "凭据说明", name: "credential_description", type: "textarea", rows: 3, placeholder: "用途、权限范围、归属团队" },
    { label: "创建后设为默认凭据", name: "credential_is_default", type: "select", value: "true", options: [{ value: "true", label: "是" }, { value: "false", label: "否" }] },
    { type: "note", copy: "密码和私钥都会在后端加密保存，页面不会回显明文。" },
  ];

  openFormModal({
    eyebrow: isEdit ? "Edit Asset" : "Create Asset",
    title: isEdit ? `编辑资产 · ${asset.name}` : "新增机器资产",
    copy: isEdit
      ? "维护主机地址、平台、协议、登录账号和状态，让资产中心保持可维护。"
      : "创建机器时可一并录入登录凭据，包括账号密码或 SSH 私钥。凭据会在后端加密保存。",
    submitText: isEdit ? "保存变更" : "创建资产",
    fields: isEdit ? createFields : [...createFields, ...credentialFields],
    onOpen: (form) => {
      const accessModeField = form.querySelector("[name='access_mode']");
      const gatewayField = form.querySelector("[name='gateway_asset_id']");
      const syncGatewayField = () => {
        const viaGateway = String(accessModeField?.value || "direct") === "via_gateway";
        gatewayField?.closest("label")?.classList.toggle("hidden", !viaGateway);
        if (!viaGateway && gatewayField) {
          gatewayField.value = "";
        }
      };
      accessModeField?.addEventListener("change", syncGatewayField);
      syncGatewayField();
    },
    onSubmit: async (form) => {
      const credentialUsername = String(form.get("credential_username") || "").trim();
      const payload = {
        name: form.get("name"),
        address: form.get("address"),
        platform: form.get("platform"),
        protocol: form.get("protocol"),
        access_mode: form.get("access_mode"),
        gateway_asset_id: Number(form.get("gateway_asset_id") || 0) || null,
        is_gateway_node: String(form.get("is_gateway_node") || "false") === "true",
        group_name: form.get("group_name"),
        login_policy: form.get("login_policy"),
        port: Number(form.get("port") || 0),
        account: form.get("account") || credentialUsername,
        status: form.get("status"),
        tags: form.get("tags"),
        description: form.get("description"),
      };
      if (isEdit) {
        await api(`/api/v1/machines/assets/${asset.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
        toast("资产信息已更新");
        await machineCtx.refreshMachineData();
        await machineCtx.openAssetDetail(asset.id, true);
        return;
      }
      const response = await api("/api/v1/machines/assets", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      const assetID = response.data?.id;
      const selectedCredentialID = Number(form.get("credential_library_id") || 0);
      const credentialAuthType = String(form.get("credential_auth_type") || "password");
      const credentialPassword = String(form.get("credential_password") || "").trim();
      const credentialPrivateKey = String(form.get("credential_private_key") || "").trim();
      const shouldImportCredential = assetID && selectedCredentialID > 0;
      const shouldCreateCredential = assetID && credentialUsername && (
        (credentialAuthType === "password" && credentialPassword) ||
        (credentialAuthType === "ssh_key" && credentialPrivateKey)
      );
      if (shouldImportCredential) {
        await api(`/api/v1/machines/assets/${assetID}/accounts/import`, {
          method: "POST",
          body: JSON.stringify({
            credential_id: selectedCredentialID,
            is_default: true,
          }),
        });
      } else if (shouldCreateCredential) {
        await api(`/api/v1/machines/assets/${assetID}/accounts`, {
          method: "POST",
          body: JSON.stringify({
            name: form.get("credential_name") || credentialUsername,
            username: credentialUsername,
            auth_type: credentialAuthType,
            password: form.get("credential_password"),
            private_key: form.get("credential_private_key"),
            passphrase: form.get("credential_passphrase"),
            description: form.get("credential_description"),
            is_default: String(form.get("credential_is_default")) === "true",
          }),
        });
      }
      toast(shouldImportCredential || shouldCreateCredential ? "机器资产和登录凭据已创建" : "机器资产已创建");
      await machineCtx.refreshMachineData();
      if (assetID) {
        await machineCtx.openAssetDetail(assetID, true);
      }
    },
  });
}

export function openBatchAssetCreateModal(machineCtx) {
  openFormModal({
    eyebrow: "Batch Create",
    title: "批量新增服务器",
    copy: "每行一台服务器，支持 `名称,地址,端口,账号` 或只填地址。其余字段作为统一默认值应用到整批机器。",
    submitText: "批量新增",
    fields: [
      { label: "服务器列表", name: "items", type: "textarea", rows: 10, placeholder: "prod-app-01,10.0.0.11,22,ec2-user\n10.0.0.12\nprod-db-01,10.0.0.21,3306,root" },
      { label: "平台", name: "platform", type: "select", value: "linux", options: platformOptions() },
      { label: "协议", name: "protocol", type: "select", value: "ssh", options: protocolOptions() },
      { label: "资产分组", name: "group_name", type: "select", value: "", options: assetGroupOptions() },
      { label: "登录策略", name: "login_policy", type: "select", value: "inherit_group", options: loginPolicyOptions() },
      { label: "默认端口", name: "port", type: "number", value: "22" },
      { label: "默认登录账号", name: "account", placeholder: "root / ops-admin" },
      { label: "状态", name: "status", type: "select", value: "online", options: statusOptions() },
      { label: "标签", name: "tags", placeholder: "prod, app, core" },
      { label: "资产说明", name: "description", type: "textarea", rows: 3, placeholder: "这批服务器的统一说明" },
    ],
    onSubmit: async (form) => {
      const raw = String(form.get("items") || "");
      const items = raw.split("\n").map((line) => line.trim()).filter(Boolean).map((line) => {
        const parts = line.split(",").map((part) => part.trim());
        if (parts.length === 1) {
          return { name: "", address: parts[0], port: 0, account: "" };
        }
        return {
          name: parts[0],
          address: parts[1] || "",
          port: Number(parts[2] || 0) || 0,
          account: parts[3] || "",
        };
      });
      const response = await api("/api/v1/machines/assets/batch-create", {
        method: "POST",
        body: JSON.stringify({
          items,
          platform: form.get("platform"),
          protocol: form.get("protocol"),
          group_name: form.get("group_name"),
          login_policy: form.get("login_policy"),
          port: Number(form.get("port") || 0),
          account: form.get("account"),
          status: form.get("status"),
          tags: form.get("tags"),
          description: form.get("description"),
        }),
      });
      toast(`已批量新增 ${response.data?.count || items.length} 台服务器`);
      await machineCtx.refreshMachineData();
    },
  });
}

export function openBatchAssetUpdateModal(machineCtx, assetIDs = []) {
  openFormModal({
    eyebrow: "Batch Update",
    title: `批量更新服务器 · ${assetIDs.length} 台`,
    copy: "只会更新你填写的字段。这里既可以统一改标签、端口、登录账号，也可以给所选服务器批量覆盖默认托管凭据。",
    submitText: "批量更新",
    fields: [
      { type: "section", eyebrow: "Asset", label: "资产字段", copy: "留空表示不修改该字段。" },
      { label: "资产分组", name: "group_name", type: "select", value: "", options: assetGroupOptions() },
      { label: "登录策略", name: "login_policy", type: "select", value: "", options: [{ value: "", label: "不修改" }, ...loginPolicyOptions().filter((item) => item.value)] },
      { label: "统一端口", name: "port", type: "number", value: "" },
      { label: "登录账号", name: "account", placeholder: "统一修改资产登录账号" },
      { label: "状态", name: "status", type: "select", value: "", options: [{ value: "", label: "不修改" }, ...statusOptions().filter((item) => item.value)] },
      { label: "标签", name: "tags", placeholder: "prod, app, core" },
      { label: "资产说明", name: "description", type: "textarea", rows: 3, placeholder: "批量追加的统一说明" },
      { type: "section", eyebrow: "Credentials", label: "默认托管凭据", copy: "如果填写登录凭据，会为所选服务器统一创建或覆盖默认托管账号。" },
      { label: "凭据名称", name: "credential_name", placeholder: "例如 Linux 运维账号" },
      { label: "凭据登录账号", name: "credential_username", placeholder: "ops-admin" },
      { label: "认证方式", name: "credential_auth_type", type: "select", value: "password", options: [{ value: "password", label: "密码" }, { value: "ssh_key", label: "SSH 私钥" }] },
      { label: "登录密码", name: "credential_password", type: "password", placeholder: "密码型凭据填写这里" },
      { label: "SSH 私钥", name: "credential_private_key", type: "textarea", rows: 6, placeholder: "私钥型凭据填写完整 PEM 内容", upload: { accept: ".pem,.key,.txt,*/*", hint: "支持本地上传私钥文件" } },
      { label: "私钥口令", name: "credential_passphrase", type: "password", placeholder: "如果私钥带口令，在这里填写" },
      { label: "凭据说明", name: "credential_description", type: "textarea", rows: 3, placeholder: "用途、适用环境" },
    ],
    onSubmit: async (form) => {
      const response = await api("/api/v1/machines/assets/batch-update", {
        method: "POST",
        body: JSON.stringify({
          ids: assetIDs,
          group_name: form.get("group_name"),
          login_policy: form.get("login_policy"),
          port: Number(form.get("port") || 0),
          account: form.get("account"),
          status: form.get("status"),
          tags: form.get("tags"),
          description: form.get("description"),
          credential_name: form.get("credential_name"),
          credential_username: form.get("credential_username"),
          credential_auth_type: form.get("credential_auth_type"),
          credential_password: form.get("credential_password"),
          credential_private_key: form.get("credential_private_key"),
          credential_passphrase: form.get("credential_passphrase"),
          credential_description: form.get("credential_description"),
        }),
      });
      toast(`已批量更新 ${response.data?.count || assetIDs.length} 台服务器`);
      await machineCtx.refreshMachineData();
    },
  });
}

export function openCredentialLibraryModal(machineCtx) {
  openFormModal({
    eyebrow: "Credentials",
    title: "新增凭据库",
    copy: "这里用于提前保存密码或 SSH 私钥。创建资产时可以直接选择已有凭据，不用重复录入。",
    submitText: "保存凭据",
    fields: [
      { label: "凭据显示名称", name: "name", placeholder: "例如 生产 Linux 运维账号" },
      { label: "登录账号", name: "username", required: true, placeholder: "ops-admin" },
      { label: "认证方式", name: "auth_type", type: "select", value: "password", options: [{ value: "password", label: "密码" }, { value: "ssh_key", label: "SSH 私钥" }] },
      { label: "登录密码", name: "password", type: "password", placeholder: "密码型凭据填写这里" },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 6, placeholder: "私钥型凭据填写完整 PEM 内容", upload: { accept: ".pem,.key,.txt,*/*", hint: "支持本地上传私钥文件" } },
      { label: "私钥口令", name: "passphrase", type: "password", placeholder: "如果私钥带口令，在这里填写" },
      { label: "凭据说明", name: "description", type: "textarea", rows: 4, placeholder: "用途、归属团队、适用环境" },
    ],
    onSubmit: async (form) => {
      await api("/api/v1/machines/credentials", {
        method: "POST",
        body: JSON.stringify({
          name: form.get("name"),
          username: form.get("username"),
          auth_type: form.get("auth_type"),
          password: form.get("password"),
          private_key: form.get("private_key"),
          passphrase: form.get("passphrase"),
          description: form.get("description"),
        }),
      });
      toast("凭据库已保存");
      await machineCtx.refreshMachineData();
    },
  });
}

export async function openCredentialDetailModal(machineCtx, credentialID) {
  const approved = await confirmAction({
    eyebrow: "Credentials",
    title: "确认查看凭据内容",
    copy: "将显示已保存的账号密码或 SSH 私钥明文。仅在需要核对时查看。",
    confirmText: "继续查看",
  });
  if (!approved) return;
  const response = await api(`/api/v1/machines/credentials/${credentialID}`);
  const credential = response.data || {};
  openFormModal({
    eyebrow: "Credentials",
    title: `查看凭据内容 · ${credential.name || "凭据"}`,
    copy: "仅管理员或已授权人员应查看这里的明文内容。离开页面后建议及时关闭弹窗。",
    submitText: false,
    fields: [
      { label: "凭据名称", name: "name", value: credential.name || "", readOnly: true },
      { label: "登录账号", name: "username", value: credential.username || "", readOnly: true },
      { label: "认证方式", name: "auth_type", value: credential.auth_type || "", readOnly: true },
      { label: "登录密码", name: "password", value: credential.password || "", readOnly: true },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 8, value: credential.private_key || "", readOnly: true },
      { label: "私钥口令", name: "passphrase", value: credential.passphrase || "", readOnly: true },
      { label: "凭据说明", name: "description", type: "textarea", rows: 3, value: credential.description || "", readOnly: true },
    ],
  });
}

export async function openAccountDetailModal(machineCtx, assetID, accountID) {
  const approved = await confirmAction({
    eyebrow: "Credentials",
    title: "确认查看登录凭据",
    copy: "将显示当前资产保存的账号密码或 SSH 私钥明文。查看动作会记录到资产操作轨迹。",
    confirmText: "继续查看",
  });
  if (!approved) return;
  const response = await api(`/api/v1/machines/assets/${assetID}/accounts/${accountID}`);
  const account = response.data || {};
  openFormModal({
    eyebrow: "Credentials",
    title: `查看登录凭据 · ${account.name || "凭据"}`,
    copy: "这里显示该资产当前保存的账号密码或 SSH 私钥明文。请仅在需要核对时查看。",
    submitText: false,
    fields: [
      { label: "显示名称", name: "name", value: account.name || "", readOnly: true },
      { label: "登录账号", name: "username", value: account.username || "", readOnly: true },
      { label: "认证方式", name: "auth_type", value: account.auth_type || "", readOnly: true },
      { label: "登录密码", name: "password", value: account.password || "", readOnly: true },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 8, value: account.private_key || "", readOnly: true },
      { label: "私钥口令", name: "passphrase", value: account.passphrase || "", readOnly: true },
      { label: "账号说明", name: "description", type: "textarea", rows: 3, value: account.description || "", readOnly: true },
    ],
  });
}

export function openAssetGroupModal(machineCtx, group = null) {
  const isEdit = Boolean(group?.id);
  const selectedParentID = group?.parent_id ? String(group.parent_id) : "";
  const parentOptions = [{ value: "", label: "作为一级资产组" }, ...flattenGroupsWithIDs(state.machine.groups || []).filter((item) => String(item.id) !== String(group?.id)).map((item) => ({
    value: String(item.id),
    label: `${"　".repeat(Number(item.depth || 0))}${item.name}`,
  }))];
  openFormModal({
    eyebrow: isEdit ? "Edit Asset Group" : "Create Asset Group",
    title: isEdit ? `编辑资产组 · ${group.name}` : "新增资产组",
    copy: "资产组用于组织机器归类，并承接分组级默认登录策略。",
    submitText: isEdit ? "保存资产组" : "创建资产组",
    fields: [
      { label: "资产组名称", name: "name", required: true, value: group?.name || "", placeholder: "例如 生产核心" },
      { label: "资产组编码", name: "code", value: group?.code || "", placeholder: "prod-core" },
      { label: "上级资产组", name: "parent_id", type: "select", value: selectedParentID, options: parentOptions },
      { label: "默认登录策略", name: "default_login_policy", type: "select", value: group?.default_login_policy || "managed_first", options: loginPolicyOptions().filter((item) => item.value !== "inherit_group") },
      { label: "说明", name: "description", type: "textarea", rows: 4, value: group?.description || "", placeholder: "这一组资产的用途和范围" },
    ],
    onSubmit: async (form) => {
      const parentID = Number(form.get("parent_id") || 0);
      const payload = {
        name: form.get("name"),
        code: form.get("code"),
        parent_id: parentID > 0 ? parentID : null,
        default_login_policy: form.get("default_login_policy"),
        description: form.get("description"),
      };
      if (isEdit) {
        await api(`/api/v1/machines/groups/${group.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
        toast("资产组已更新");
      } else {
        await api("/api/v1/machines/groups", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        toast("资产组已创建");
      }
      await machineCtx.refreshMachineData();
    },
  });
}

function flattenGroupsWithIDs(groups, depth = 0) {
  if (!Array.isArray(groups) || !groups.length) return [];
  return groups.flatMap((group) => {
    const current = { ...group, depth };
    return [current, ...flattenGroupsWithIDs(group.children || [], depth + 1)];
  });
}

export function openAccountModal(machineCtx, asset) {
  openFormModal({
    eyebrow: "Credentials",
    title: `新增登录凭据 · ${asset.name}`,
    copy: "这里用于保存登录账号、密码或 SSH 私钥。凭据会在后端加密保存，页面不会回显明文。",
    submitText: "保存凭据",
    fields: [
      { label: "显示名称", name: "name", placeholder: "例如 Linux 运维账号" },
      { label: "登录账号", name: "username", required: true, placeholder: "ops-admin" },
      { label: "认证方式", name: "auth_type", type: "select", value: "password", options: [{ value: "password", label: "密码" }, { value: "ssh_key", label: "SSH 私钥" }] },
      { label: "登录密码", name: "password", type: "password", placeholder: "密码型账号填写这里" },
      { label: "SSH 私钥", name: "private_key", type: "textarea", rows: 6, placeholder: "私钥型账号填写完整 PEM 内容", upload: { accept: ".pem,.key,.txt,*/*", hint: "支持本地上传私钥文件" } },
      { label: "私钥口令", name: "passphrase", type: "password", placeholder: "如果私钥带口令，在这里填写" },
      { label: "账号说明", name: "description", type: "textarea", rows: 4, placeholder: "用途、权限范围、归属团队" },
      { label: "设为默认账号", name: "is_default", type: "select", value: "true", options: [{ value: "true", label: "是" }, { value: "false", label: "否" }] },
    ],
    onSubmit: async (form) => {
      await api(`/api/v1/machines/assets/${asset.id}/accounts`, {
        method: "POST",
        body: JSON.stringify({
          name: form.get("name"),
          username: form.get("username"),
          auth_type: form.get("auth_type"),
          password: form.get("password"),
          private_key: form.get("private_key"),
          passphrase: form.get("passphrase"),
          description: form.get("description"),
          is_default: String(form.get("is_default")) === "true",
      }),
      });
      toast("登录凭据已保存");
      await machineCtx.openAssetDetail(asset.id, true);
    },
  });
}
