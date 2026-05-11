import { api } from "../../core/api.js";
import { elements } from "../../core/dom.js";
import { state } from "../../core/state.js";
import { bindAction, bindSearch, closeDrawer, confirmAction, openDrawer, openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { emptyState, escapeHtml, filterItems, findById, formatDateTime, renderTagList } from "../../shared/utils.js";

export async function loadSystemData() {
  await Promise.all([loadUsers(), loadGroups(), loadRoles(), loadSources(), loadBindings(), loadSettings(), loadAuditEvents()]);
  renderAccessControlPage();
  renderAuditCenterPage();
}

export function bindSystemEvents() {
  elements.createUserForm.addEventListener("submit", onCreateUser);
  elements.createRoleForm.addEventListener("submit", onCreateRole);
  elements.createGroupForm.addEventListener("submit", onCreateGroup);
  elements.createSourceForm.addEventListener("submit", onCreateSource);
  elements.createSettingForm.addEventListener("submit", onUpsertSetting);
  elements.assignRoleForm.addEventListener("submit", onAssignRole);
  elements.assignSubjectType.addEventListener("change", renderSubjectOptions);

  elements.openCreateUser.addEventListener("click", () => openDrawer("user"));
  elements.openCreateRole.addEventListener("click", () => openDrawer("role"));
  elements.openCreateGroup.addEventListener("click", () => openDrawer("group"));

  document.getElementById("refresh-users").addEventListener("click", loadUsers);
  document.getElementById("refresh-groups").addEventListener("click", loadGroups);
  document.getElementById("refresh-sources").addEventListener("click", loadSources);
  document.getElementById("refresh-bindings").addEventListener("click", loadBindings);
  document.getElementById("refresh-settings").addEventListener("click", loadSettings);

  bindSearch(elements.searchUsers, (value) => {
    state.filters.users = value;
    renderUsers();
  });
  bindSearch(elements.searchGroups, (value) => {
    state.filters.groups = value;
    renderGroups();
  });
  bindSearch(elements.searchRoles, (value) => {
    state.filters.roles = value;
    renderRoles();
  });
  bindSearch(elements.searchSources, (value) => {
    state.filters.sources = value;
    renderSources();
  });
  bindSearch(elements.searchBindings, (value) => {
    state.filters.bindings = value;
    renderBindings();
  });
  bindSearch(elements.searchSettings, (value) => {
    state.filters.settings = value;
    renderSettings();
  });
  bindSearch(elements.searchAccessRoles, (value) => {
    state.filters.accessRoles = value;
    renderAccessControlPage();
  });
  bindSearch(elements.searchAccessBindings, (value) => {
    state.filters.accessBindings = value;
    renderAccessControlPage();
  });
  elements.accessBindingTypeFilter?.addEventListener("change", (event) => {
    state.filters.accessBindingType = String(event.target.value || "");
    renderAccessControlPage();
  });
  bindSearch(elements.searchAuditSessions, (value) => {
    state.filters.auditSessions = value;
    renderAuditCenterPage();
  });
  bindSearch(elements.searchAuditChanges, (value) => {
    state.filters.auditChanges = value;
    renderAuditCenterPage();
  });
  elements.auditChangeKindFilter?.addEventListener("change", (event) => {
    state.filters.auditChangeKind = String(event.target.value || "");
    renderAuditCenterPage();
  });
  elements.auditEventTypeFilter?.addEventListener("change", (event) => {
    state.filters.auditEventType = String(event.target.value || "");
    renderAuditCenterPage();
  });
  elements.auditEventLevelFilter?.addEventListener("change", (event) => {
    state.filters.auditEventLevel = String(event.target.value || "");
    renderAuditCenterPage();
  });
  elements.refreshAccessControl?.addEventListener("click", async () => {
    await Promise.all([loadUsers(), loadGroups(), loadRoles(), loadBindings()]);
    renderAccessControlPage();
    toast("访问控制视图已刷新");
  });
  elements.refreshAuditCenter?.addEventListener("click", async () => {
    await refreshAuditCenterData();
    toast("审计视图已刷新");
  });
}

async function loadUsers() {
  const payload = await api("/api/v1/users");
  state.users = payload.data || [];
  renderUsers();
  renderSubjectOptions();
}

async function loadGroups() {
  const payload = await api("/api/v1/groups");
  state.groups = payload.data || [];
  renderGroups();
  renderSubjectOptions();
}

async function loadRoles() {
  const payload = await api("/api/v1/roles");
  state.roles = payload.data || [];
  renderRoles();
  elements.assignRoleId.innerHTML = state.roles.map((role) => `<option value="${role.id}">${escapeHtml(role.name)} (${escapeHtml(role.code)})</option>`).join("");
}

async function loadSources() {
  const payload = await api("/api/v1/identity-sources");
  state.sources = payload.data || [];
  renderSources();
}

async function loadBindings() {
  const payload = await api("/api/v1/role-bindings");
  state.bindings = payload.data || [];
  renderBindings();
}

async function loadSettings() {
  const payload = await api("/api/v1/system-settings");
  state.settings = payload.data || [];
  renderSettings();
}

async function loadAuditEvents() {
  const payload = await api("/api/v1/machines/events?limit=40");
  state.auditEvents = payload.data || [];
}

async function onCreateUser(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/users", {
    method: "POST",
    body: JSON.stringify({
      username: form.get("username"),
      display_name: form.get("display_name"),
      email: form.get("email"),
      password: form.get("password"),
      mfa_required: String(form.get("mfa_required")) === "true",
    }),
  });
  event.target.reset();
  closeDrawer("user");
  toast("用户已创建");
  await loadUsers();
}

async function onCreateRole(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/roles", {
    method: "POST",
    body: JSON.stringify({ name: form.get("name"), code: form.get("code"), description: form.get("description") }),
  });
  event.target.reset();
  closeDrawer("role");
  toast("角色已创建");
  await loadRoles();
}

async function onCreateGroup(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/groups", {
    method: "POST",
    body: JSON.stringify({ name: form.get("name"), code: form.get("code"), description: form.get("description") }),
  });
  event.target.reset();
  closeDrawer("group");
  toast("用户组已创建");
  await loadGroups();
}

async function onCreateSource(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/identity-sources", {
    method: "POST",
    body: JSON.stringify({
      name: form.get("name"),
      type: form.get("type"),
      host: form.get("host"),
      port: Number(form.get("port") || 0),
      base_dn: form.get("base_dn"),
      bind_dn: form.get("bind_dn"),
      bind_password: form.get("bind_password"),
      user_filter: form.get("user_filter"),
      issuer_url: form.get("issuer_url"),
      client_id: form.get("client_id"),
      client_secret: form.get("client_secret"),
      redirect_url: form.get("redirect_url"),
      enabled: true,
    }),
  });
  event.target.reset();
  toast("身份源已保存");
  await loadSources();
}

async function onUpsertSetting(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  await api("/api/v1/system-settings", {
    method: "POST",
    body: JSON.stringify({ key: form.get("key"), value: form.get("value"), description: form.get("description") }),
  });
  event.target.reset();
  toast("系统设置已保存");
  await loadSettings();
}

async function onAssignRole(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  const subjectType = String(form.get("subject_type"));
  const subjectName = String(form.get("subject_name") || "").trim();
  const subject = subjectType === "user" ? state.users.find((item) => item.username === subjectName) : state.groups.find((item) => item.name === subjectName);
  if (!subject) {
    await showErrorDialog({
      title: "分配失败",
      copy: subjectType === "user" ? "请选择有效用户名" : "请选择有效用户组",
    });
    return;
  }
  await api("/api/v1/role-bindings", {
    method: "POST",
    body: JSON.stringify({ subject_type: subjectType, subject_id: String(subject.id), role_id: Number(form.get("role_id")) }),
  });
  event.target.reset();
  renderSubjectOptions();
  toast("角色分配完成");
  await Promise.all([loadUsers(), loadGroups(), loadBindings()]);
}

function renderUsers() {
  const users = filterItems(state.users, state.filters.users, ["username", "display_name", "email", "source_type", "status"]);
  if (!state.users.length) return (elements.usersTable.innerHTML = emptyState("暂无用户"));
  if (!users.length) return (elements.usersTable.innerHTML = emptyState("没有匹配的用户"));
  elements.usersTable.innerHTML = `<table><thead><tr><th>序号</th><th>用户名</th><th>显示名</th><th>用户组</th><th>角色</th><th>MFA</th><th>来源</th><th>状态</th><th>操作</th></tr></thead><tbody>${users.map((user, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><strong>${escapeHtml(user.username)}</strong><span class="table-meta">创建于 ${formatDateTime(user.created_at)}</span></div></td><td><div class="cell-stack"><span>${escapeHtml(user.display_name)}</span><span class="table-meta">更新于 ${formatDateTime(user.updated_at)}</span></div></td><td>${renderTagList(user.groups || [], "name", "group-tag")}</td><td>${renderTagList(user.roles || [], "name", "role-tag")}</td><td><div class="cell-stack"><span class="${user.mfa_required ? "role-tag" : "group-tag"}">${user.mfa_required ? "强制开启" : "未强制"}</span><span class="table-meta">${user.mfa_enabled ? "已启用" : "未启用"}</span></div></td><td>${escapeHtml(user.source_type)}</td><td><span class="status-pill ${user.status !== "active" ? "disabled" : ""}">${escapeHtml(user.status)}</span></td><td><div class="action-row"><button class="ghost-button" data-action="edit-user" data-id="${user.id}">编辑</button><button class="ghost-button" data-action="password-user" data-id="${user.id}">改密</button><button class="ghost-button" data-action="toggle-user" data-id="${user.id}">${user.status === "active" ? "禁用" : "启用"}</button><button class="ghost-button danger-soft" data-action="delete-user" data-id="${user.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.usersTable, "edit-user", (id) => openUserEditModal(findById(state.users, id)));
  bindAction(elements.usersTable, "password-user", (id) => openPasswordModal(findById(state.users, id)));
  bindAction(elements.usersTable, "toggle-user", (id) => toggleUserStatus(findById(state.users, id)));
  bindAction(elements.usersTable, "delete-user", (id) => deleteUser(findById(state.users, id)));
}

function renderGroups() {
  const groups = filterItems(state.groups, state.filters.groups, ["name", "code", "description"]);
  if (!state.groups.length) return (elements.groupsTable.innerHTML = emptyState("暂无用户组"));
  if (!groups.length) return (elements.groupsTable.innerHTML = emptyState("没有匹配的用户组"));
  elements.groupsTable.innerHTML = `<table><thead><tr><th>序号</th><th>名称</th><th>编码</th><th>成员</th><th>操作</th></tr></thead><tbody>${groups.map((group, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><span>${escapeHtml(group.name)}</span><span class="table-meta">创建于 ${formatDateTime(group.created_at)}</span></div></td><td><div class="cell-stack"><span>${escapeHtml(group.code)}</span><span class="table-meta">${escapeHtml(group.description || "-")}</span></div></td><td>${group.members?.length ? group.members.map((m) => `<span class="mini-line">${escapeHtml(m.username)}<button class="inline-link" data-action="remove-member" data-group-id="${group.id}" data-user-id="${m.id}">移除</button></span>`).join("") : "<span class='muted-label'>暂无成员</span>"}</td><td><div class="action-row"><button class="ghost-button" data-action="add-member" data-id="${group.id}">加成员</button><button class="ghost-button" data-action="edit-group" data-id="${group.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-group" data-id="${group.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.groupsTable, "add-member", (id) => openAddMemberModal(findById(state.groups, id)));
  bindAction(elements.groupsTable, "edit-group", (id) => openGroupEditModal(findById(state.groups, id)));
  bindAction(elements.groupsTable, "delete-group", (id) => deleteGroup(findById(state.groups, id)));
  elements.groupsTable.querySelectorAll("[data-action='remove-member']").forEach((button) => button.addEventListener("click", () => removeGroupMember(button.dataset.groupId, button.dataset.userId)));
}

function renderRoles() {
  const roles = filterItems(state.roles, state.filters.roles, ["name", "code", "description"]);
  if (!state.roles.length) return (elements.rolesTable.innerHTML = emptyState("暂无角色"));
  if (!roles.length) return (elements.rolesTable.innerHTML = emptyState("没有匹配的角色"));
  elements.rolesTable.innerHTML = `<table><thead><tr><th>序号</th><th>名称</th><th>编码</th><th>说明</th><th>操作</th></tr></thead><tbody>${roles.map((role, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><span>${escapeHtml(role.name)}</span><span class="table-meta">创建于 ${formatDateTime(role.created_at)}</span></div></td><td><div class="cell-stack"><span>${escapeHtml(role.code)}</span><span class="table-meta">更新于 ${formatDateTime(role.updated_at)}</span></div></td><td>${escapeHtml(role.description || "-")}</td><td><div class="action-row"><button class="ghost-button" data-action="edit-role" data-id="${role.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-role" data-id="${role.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.rolesTable, "edit-role", (id) => openRoleEditModal(findById(state.roles, id)));
  bindAction(elements.rolesTable, "delete-role", (id) => deleteRole(findById(state.roles, id)));
}

function renderSources() {
  const sources = filterItems(state.sources, state.filters.sources, ["name", "type", "host", "issuer_url", "base_dn"]);
  if (!state.sources.length) return (elements.sourcesTable.innerHTML = emptyState("暂无身份源"));
  if (!sources.length) return (elements.sourcesTable.innerHTML = emptyState("没有匹配的身份源"));
  elements.sourcesTable.innerHTML = `<table><thead><tr><th>序号</th><th>名称</th><th>类型</th><th>连接</th><th>密钥</th><th>操作</th></tr></thead><tbody>${sources.map((item, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><span>${escapeHtml(item.name)}</span><span class="table-meta">更新于 ${formatDateTime(item.updated_at)}</span></div></td><td>${escapeHtml(item.type)}</td><td>${escapeHtml(item.host || item.issuer_url || "-")}</td><td>${item.has_secret ? "已配置" : "未配置"}</td><td><div class="action-row"><button class="ghost-button" data-action="test-source" data-id="${item.id}">测试</button><button class="ghost-button" data-action="edit-source" data-id="${item.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-source" data-id="${item.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.sourcesTable, "test-source", (id) => testSource(Number(id)));
  bindAction(elements.sourcesTable, "edit-source", (id) => openSourceEditModal(findById(state.sources, id)));
  bindAction(elements.sourcesTable, "delete-source", (id) => deleteSource(findById(state.sources, id)));
}

function renderBindings() {
  const bindings = filterItems(state.bindings, state.filters.bindings, ["subject_type", "subject_name", "subject_display_name", "role_name", "role_code"]);
  if (!state.bindings.length) return (elements.bindingsTable.innerHTML = emptyState("暂无角色绑定"));
  if (!bindings.length) return (elements.bindingsTable.innerHTML = emptyState("没有匹配的绑定关系"));
  elements.bindingsTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>名称</th><th>说明</th><th>角色</th><th>操作</th></tr></thead><tbody>${bindings.map((item, index) => `<tr><td>${index + 1}</td><td>${item.subject_type === "user" ? "用户" : item.subject_type === "group" ? "用户组" : escapeHtml(item.subject_type)}</td><td>${escapeHtml(item.subject_name || item.subject_id)}</td><td>${escapeHtml(item.subject_display_name || "-")}</td><td>${escapeHtml(item.role_name)} (${escapeHtml(item.role_code)})</td><td><button class="ghost-button danger-soft" data-action="delete-binding" data-id="${item.id}">解绑</button></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.bindingsTable, "delete-binding", (id) => removeBinding(findById(state.bindings, id)));
}

function renderSettings() {
  const settings = filterItems(state.settings, state.filters.settings, ["key", "value", "description"]);
  if (!state.settings.length) return (elements.settingsTable.innerHTML = emptyState("暂无系统设置"));
  if (!settings.length) return (elements.settingsTable.innerHTML = emptyState("没有匹配的系统设置"));
  elements.settingsTable.innerHTML = `<table><thead><tr><th>序号</th><th>Key</th><th>Value</th><th>说明</th><th>操作</th></tr></thead><tbody>${settings.map((item, index) => `<tr><td>${index + 1}</td><td><div class="cell-stack"><span>${escapeHtml(item.key)}</span><span class="table-meta">更新于 ${formatDateTime(item.updated_at)}</span></div></td><td class="code-cell">${escapeHtml(item.value || "")}</td><td>${escapeHtml(item.description || "-")}</td><td><div class="action-row"><button class="ghost-button" data-action="edit-setting" data-id="${item.id}">编辑</button><button class="ghost-button danger-soft" data-action="delete-setting" data-id="${item.id}">删除</button></div></td></tr>`).join("")}</tbody></table>`;
  bindAction(elements.settingsTable, "edit-setting", (id) => openSettingEditModal(findById(state.settings, id)));
  bindAction(elements.settingsTable, "delete-setting", (id) => deleteSetting(findById(state.settings, id)));
}

export function renderAccessControlPage() {
  renderAccessControlSummary();
  renderAccessRoleCatalog();
  renderAccessRoleDetail();
  renderAccessBindingExplorer();
}

export function renderAuditCenterPage() {
  renderAuditSummary();
  renderAuditEvents();
  renderAuditSessions();
  renderAuditChanges();
  renderAuditTimeline();
}

async function refreshAuditCenterData() {
  const [sessionsPayload, eventsPayload] = await Promise.all([
    api("/api/v1/machines/sessions"),
    api("/api/v1/machines/events?limit=40"),
    loadUsers(),
    loadGroups(),
    loadRoles(),
    loadBindings(),
    loadSettings(),
    loadSources(),
  ]);
  state.machine.sessions = sessionsPayload.data || [];
  state.auditEvents = eventsPayload.data || [];
  renderAuditCenterPage();
}

function renderAccessControlSummary() {
  if (!elements.accessControlSummary) return;
  const roles = state.roles || [];
  const bindings = state.bindings || [];
  const directUserCount = new Set(bindings.filter((item) => item.subject_type === "user").map((item) => String(item.subject_id || item.subject_name))).size;
  const groupBindingCount = new Set(bindings.filter((item) => item.subject_type === "group").map((item) => String(item.subject_id || item.subject_name))).size;
  const privilegedBindings = bindings.filter((item) => /admin|manage|owner/i.test(`${item.role_code || ""} ${item.role_name || ""}`)).length;
  elements.accessControlSummary.innerHTML = `
    <article class="summary-card">
      <span class="muted-label">角色总数</span>
      <strong>${roles.length}</strong>
      <p>当前平台纳管的角色数量。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">绑定关系</span>
      <strong>${bindings.length}</strong>
      <p>用户和用户组承接角色的总绑定数。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">直接授权用户</span>
      <strong>${directUserCount}</strong>
      <p>直接绑定角色的用户数，不含组继承。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">授权用户组</span>
      <strong>${groupBindingCount}</strong>
      <p>通过用户组承接角色的范围。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">高权限绑定</span>
      <strong>${privilegedBindings}</strong>
      <p>名称中包含 admin / manage / owner 的绑定数。</p>
    </article>
  `;
}

function renderAccessRoleCatalog() {
  if (!elements.accessRolesTable) return;
  const query = state.filters.accessRoles || "";
  const roles = filterItems(state.roles, query, ["name", "code", "description"]);
  if (!state.roles.length) return (elements.accessRolesTable.innerHTML = emptyState("暂无角色"));
  if (!roles.length) return (elements.accessRolesTable.innerHTML = emptyState("没有匹配的角色"));
  if (!roles.some((item) => Number(item.id) === Number(state.selectedAccessRoleID))) {
    state.selectedAccessRoleID = Number(roles[0]?.id || 0) || null;
  }
  const rows = roles.map((role, index) => {
    const directUsers = (state.bindings || []).filter((item) => item.role_id === role.id && item.subject_type === "user").length;
    const directGroups = (state.bindings || []).filter((item) => item.role_id === role.id && item.subject_type === "group").length;
    const permissions = Array.isArray(role.permissions) ? role.permissions : [];
    return `
      <tr>
        <td>${index + 1}</td>
        <td><div class="cell-stack"><strong>${escapeHtml(role.name)}</strong><span class="table-meta">创建于 ${formatDateTime(role.created_at)}</span><span class="table-meta"><button class="inline-link" data-access-role-select="${role.id}">${Number(state.selectedAccessRoleID) === Number(role.id) ? "当前角色" : "查看权限"}</button></span></div></td>
        <td><div class="cell-stack"><span>${escapeHtml(role.code)}</span><span class="table-meta">${escapeHtml(role.description || "-")}</span><span class="table-meta">${permissions.length ? escapeHtml(permissions.map((item) => `${item.resource}.${item.action}`).join(" / ")) : "未绑定权限点"}</span></div></td>
        <td>${directUsers}</td>
        <td>${directGroups}</td>
        <td><div class="cell-stack"><span>${formatDateTime(role.updated_at)}</span><span class="table-meta"><button class="inline-link" data-access-role-filter="${role.id}">查看绑定</button> · <button class="inline-link" data-access-role-jump="${role.id}">去编辑</button></span></div></td>
      </tr>
    `;
  }).join("");
  elements.accessRolesTable.innerHTML = `<table><thead><tr><th>序号</th><th>角色</th><th>编码 / 说明</th><th>直接用户</th><th>用户组</th><th>最近更新</th></tr></thead><tbody>${rows}</tbody></table>`;
  elements.accessRolesTable.querySelectorAll("[data-access-role-select]").forEach((button) => button.addEventListener("click", () => {
    state.selectedAccessRoleID = Number(button.dataset.accessRoleSelect || 0) || null;
    renderAccessRoleCatalog();
    renderAccessRoleDetail();
  }));
  elements.accessRolesTable.querySelectorAll("[data-access-role-filter]").forEach((button) => button.addEventListener("click", () => {
    const roleID = Number(button.dataset.accessRoleFilter || 0);
    const role = (state.roles || []).find((item) => Number(item.id) === roleID);
    state.filters.accessBindings = String(role?.name || role?.code || "").trim().toLowerCase();
    if (elements.searchAccessBindings) elements.searchAccessBindings.value = state.filters.accessBindings;
    renderAccessBindingExplorer();
  }));
  elements.accessRolesTable.querySelectorAll("[data-access-role-jump]").forEach((button) => button.addEventListener("click", () => {
    window.dispatchEvent(new CustomEvent("bc:navigate", {
      detail: {
        topModule: "system",
        systemModule: "user-management",
        systemTab: "roles",
      },
    }));
  }));
}

function renderAccessRoleDetail() {
  if (!elements.accessRoleDetail) return;
  const role = (state.roles || []).find((item) => Number(item.id) === Number(state.selectedAccessRoleID))
    || (state.roles || [])[0];
  if (!role) {
    elements.accessRoleDetail.innerHTML = emptyState("暂无可查看的角色权限");
    return;
  }
  const permissions = Array.isArray(role.permissions) ? role.permissions : [];
  const grouped = permissions.reduce((result, item) => {
    const resource = item.resource || "unknown";
    if (!result[resource]) result[resource] = [];
    result[resource].push(item.action || "-");
    return result;
  }, {});
  const bindings = (state.bindings || []).filter((item) => Number(item.role_id) === Number(role.id));
  const bindingTags = bindings.map((item) => ({
    label: `${item.subject_type === "user" ? "用户" : "用户组"} · ${item.subject_name || item.subject_id}`,
  }));
  const permissionBlocks = Object.entries(grouped).map(([resource, actions]) => `
    <details class="summary-card" open>
      <summary class="panel-header">
        <div>
          <span class="muted-label">${escapeHtml(resource)}</span>
          <strong>${actions.length} 个动作</strong>
        </div>
      </summary>
      <div class="cell-stack">
        <span class="table-meta">资源 ${escapeHtml(resource)} 当前绑定的动作集合。</span>
        ${renderTagList(actions.map((action) => ({ label: action })), "label", "role-tag")}
      </div>
    </details>
  `).join("");
  elements.accessRoleDetail.innerHTML = `
    <div class="cell-stack">
      <strong>${escapeHtml(role.name)} <span class="table-meta">(${escapeHtml(role.code)})</span></strong>
      <span class="table-meta">${escapeHtml(role.description || "未填写角色说明")}</span>
      <span class="table-meta">最近更新 ${formatDateTime(role.updated_at)}</span>
    </div>
    <div class="summary-grid">
      <article class="summary-card">
        <span class="muted-label">权限点数量</span>
        <strong>${permissions.length}</strong>
        <p>当前角色实际绑定的资源动作总数。</p>
      </article>
      <article class="summary-card">
        <span class="muted-label">资源类别</span>
        <strong>${Object.keys(grouped).length}</strong>
        <p>按资源归并后的权限范围。</p>
      </article>
    </div>
    <div class="cell-stack">
      <span class="muted-label">承接对象</span>
      ${bindingTags.length ? renderTagList(bindingTags, "label", "role-tag") : `<span class="table-meta">当前没有绑定到用户或用户组</span>`}
    </div>
    ${permissionBlocks ? `<div class="summary-grid">${permissionBlocks}</div>` : emptyState("当前角色未绑定权限点")}
  `;
}

function renderAccessBindingExplorer() {
  if (!elements.accessBindingsTable) return;
  const query = state.filters.accessBindings || "";
  const typeFilter = state.filters.accessBindingType || "";
  const bindings = filterItems(state.bindings, query, ["subject_type", "subject_name", "subject_display_name", "role_name", "role_code"])
    .filter((item) => !typeFilter || item.subject_type === typeFilter);
  if (!state.bindings.length) return (elements.accessBindingsTable.innerHTML = emptyState("暂无授权绑定"));
  if (!bindings.length) return (elements.accessBindingsTable.innerHTML = emptyState("没有匹配的绑定关系"));
  const rows = bindings.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td>${item.subject_type === "user" ? "用户" : item.subject_type === "group" ? "用户组" : escapeHtml(item.subject_type)}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.subject_name || item.subject_id)}</strong><span class="table-meta">${escapeHtml(item.subject_display_name || "-")}</span></div></td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.role_name || "-")}</strong><span class="table-meta">${escapeHtml(item.role_code || "-")}</span></div></td>
      <td><div class="cell-stack"><span>${formatDateTime(item.created_at || item.updated_at)}</span><span class="table-meta"><button class="inline-link" data-access-binding-jump="${item.subject_type}">去授权页</button></span></div></td>
    </tr>
  `).join("");
  elements.accessBindingsTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>对象</th><th>角色</th><th>绑定时间</th></tr></thead><tbody>${rows}</tbody></table>`;
  elements.accessBindingsTable.querySelectorAll("[data-access-binding-jump]").forEach((button) => button.addEventListener("click", () => {
    window.dispatchEvent(new CustomEvent("bc:navigate", {
      detail: {
        topModule: "system",
        systemModule: "user-management",
        systemTab: "roles",
      },
    }));
  }));
}

function renderAuditSummary() {
  if (!elements.auditSummary) return;
  const sessions = state.machine.sessions || [];
  const machineSummary = state.machine.summary || {};
  const disabledUsers = (state.users || []).filter((item) => item.status !== "active").length;
  const changes = buildAuditTimelineItems();
  const machineEvents = state.auditEvents || [];
  elements.auditSummary.innerHTML = `
    <article class="summary-card">
      <span class="muted-label">最近会话</span>
      <strong>${sessions.length}</strong>
      <p>最近机器连接与终端操作记录。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">告警资产</span>
      <strong>${Number(machineSummary.warning_assets || 0)}</strong>
      <p>机器状态为需关注的资产数。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">离线资产</span>
      <strong>${Number(machineSummary.offline_assets || 0)}</strong>
      <p>当前不可用或离线的机器数。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">禁用用户</span>
      <strong>${disabledUsers}</strong>
      <p>当前被禁用的平台用户数量。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">治理变更</span>
      <strong>${changes.length}</strong>
      <p>治理变更与机器事件混合后的最近时间线。</p>
    </article>
    <article class="summary-card">
      <span class="muted-label">机器事件</span>
      <strong>${machineEvents.length}</strong>
      <p>来自终端与文件操作链路的最近事件。</p>
    </article>
  `;
}

function renderAuditEvents() {
  if (!elements.auditEventsTable) return;
  const eventType = (state.filters.auditEventType || "").toLowerCase();
  const eventLevel = (state.filters.auditEventLevel || "").toLowerCase();
  const events = (state.auditEvents || [])
    .filter((item) => !eventType || String(item.event_type || "").toLowerCase().includes(eventType))
    .filter((item) => !eventLevel || String(item.event_level || "").toLowerCase() === eventLevel);
  if (!events.length) return (elements.auditEventsTable.innerHTML = emptyState("暂无机器事件流"));
  const rows = events.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td>${escapeHtml(item.asset_name || "-")}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.event_type || "-")}</strong><span class="table-meta">${escapeHtml(item.summary || "-")}</span></div></td>
      <td><span class="status-pill ${String(item.event_level || "").toLowerCase() === "error" ? "disabled" : ""}">${escapeHtml(item.event_level || "-")}</span></td>
      <td>${escapeHtml(item.detail || "-")}</td>
      <td>${formatDateTime(item.created_at)}</td>
    </tr>
  `).join("");
  elements.auditEventsTable.innerHTML = `<table><thead><tr><th>序号</th><th>资产</th><th>事件</th><th>级别</th><th>详情</th><th>时间</th></tr></thead><tbody>${rows}</tbody></table>`;
}

function renderAuditTimeline() {
  if (!elements.auditTimelineTable) return;
  const timeline = buildUnifiedAuditTimeline();
  if (!timeline.length) return (elements.auditTimelineTable.innerHTML = emptyState("暂无统一时间轴记录"));
  const rows = timeline.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td>${escapeHtml(item.kind)}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">${escapeHtml(item.summary || "-")}</span></div></td>
      <td>${escapeHtml(item.source || "-")}</td>
      <td><div class="cell-stack"><span>${formatDateTime(item.time)}</span><span class="table-meta"><button class="inline-link" data-audit-timeline-jump="${escapeHtml(item.jump)}">去对应页</button></span></div></td>
    </tr>
  `).join("");
  elements.auditTimelineTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>对象</th><th>来源</th><th>时间</th></tr></thead><tbody>${rows}</tbody></table>`;
  elements.auditTimelineTable.querySelectorAll("[data-audit-timeline-jump]").forEach((button) => button.addEventListener("click", () => {
    const target = String(button.dataset.auditTimelineJump || "");
    if (target === "machine") {
      window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "machine" } }));
      return;
    }
    if (target === "settings") {
      window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "system", systemModule: "system-settings" } }));
      return;
    }
    window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "system", systemModule: "user-management", systemTab: target === "source" ? "identity-sources" : "roles" } }));
  }));
}

function renderAuditSessions() {
  if (!elements.auditSessionsTable) return;
  const sessions = filterItems(state.machine.sessions || [], state.filters.auditSessions || "", ["asset_name", "address", "account", "protocol", "status", "detail"]);
  if (!sessions.length) return (elements.auditSessionsTable.innerHTML = emptyState("暂无会话记录"));
  const rows = sessions.map((session, index) => `
    <tr>
      <td>${index + 1}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(session.asset_name || "-")}</strong><span class="table-meta">${escapeHtml(session.address || "-")}</span></div></td>
      <td>${escapeHtml(session.account || "-")}</td>
      <td>${escapeHtml(session.protocol || "-")}</td>
      <td><span class="status-pill ${session.status !== "closed" ? "" : "disabled"}">${escapeHtml(session.status || "-")}</span></td>
      <td><div class="cell-stack"><span>${formatDateTime(session.started_at)}</span><span class="table-meta">${session.ended_at ? `结束于 ${formatDateTime(session.ended_at)}` : "仍在运行或待回收"} · <button class="inline-link" data-audit-session-jump="machine">去机器页</button></span></div></td>
    </tr>
  `).join("");
  elements.auditSessionsTable.innerHTML = `<table><thead><tr><th>序号</th><th>资产</th><th>账号</th><th>协议</th><th>状态</th><th>时间</th></tr></thead><tbody>${rows}</tbody></table>`;
  elements.auditSessionsTable.querySelectorAll("[data-audit-session-jump]").forEach((button) => button.addEventListener("click", () => {
    window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "machine" } }));
  }));
}

function renderAuditChanges() {
  if (!elements.auditChangesTable) return;
  const changes = buildAuditTimelineItems()
    .filter((item) => !state.filters.auditChangeKind || item.kind === state.filters.auditChangeKind);
  const filtered = filterItems(changes, state.filters.auditChanges || "", ["kind", "name", "summary"]);
  if (!filtered.length) return (elements.auditChangesTable.innerHTML = emptyState("暂无治理变更记录"));
  const rows = filtered.map((item, index) => `
    <tr>
      <td>${index + 1}</td>
      <td>${escapeHtml(item.kind)}</td>
      <td><div class="cell-stack"><strong>${escapeHtml(item.name)}</strong><span class="table-meta">${escapeHtml(item.summary)}</span></div></td>
      <td><div class="cell-stack"><span>${formatDateTime(item.time)}</span><span class="table-meta"><button class="inline-link" data-audit-change-jump="${escapeHtml(resolveAuditJumpModule(item.kind))}">去对应页</button></span></div></td>
    </tr>
  `).join("");
  elements.auditChangesTable.innerHTML = `<table><thead><tr><th>序号</th><th>类型</th><th>对象</th><th>时间</th></tr></thead><tbody>${rows}</tbody></table>`;
  elements.auditChangesTable.querySelectorAll("[data-audit-change-jump]").forEach((button) => button.addEventListener("click", () => {
    const kind = String(button.dataset.auditChangeJump || "");
    if (kind === "settings") {
      window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "system", systemModule: "system-settings" } }));
      return;
    }
    window.dispatchEvent(new CustomEvent("bc:navigate", { detail: { topModule: "system", systemModule: "user-management", systemTab: kind === "source" ? "identity-sources" : "roles" } }));
  }));
}

function buildAuditChangeItems() {
  const items = [];
  (state.users || []).forEach((item) => items.push({ kind: "用户", name: item.username, summary: item.display_name || item.email || "用户信息更新", time: item.updated_at || item.created_at }));
  (state.groups || []).forEach((item) => items.push({ kind: "用户组", name: item.name, summary: item.description || item.code || "用户组更新", time: item.updated_at || item.created_at }));
  (state.roles || []).forEach((item) => items.push({ kind: "角色", name: item.name, summary: item.description || item.code || "角色更新", time: item.updated_at || item.created_at }));
  (state.bindings || []).forEach((item) => items.push({ kind: "角色绑定", name: `${item.subject_name || item.subject_id} -> ${item.role_name || item.role_code}`, summary: item.subject_type === "user" ? "直接授权给用户" : "授权给用户组", time: item.updated_at || item.created_at }));
  (state.settings || []).forEach((item) => items.push({ kind: "系统设置", name: item.key, summary: item.description || item.value || "系统设置更新", time: item.updated_at || item.created_at }));
  (state.sources || []).forEach((item) => items.push({ kind: "身份源", name: item.name, summary: item.type || "身份源更新", time: item.updated_at || item.created_at }));
  return items
    .filter((item) => item.time)
    .sort((left, right) => new Date(right.time).getTime() - new Date(left.time).getTime())
    .slice(0, 20);
}

function buildAuditTimelineItems() {
  const governance = buildAuditChangeItems();
  const machineEvents = (state.auditEvents || []).map((item) => ({
    kind: "机器事件",
    name: item.asset_name || "-",
    summary: `${item.event_type || "event"} · ${item.summary || item.detail || "-"}`,
    time: item.created_at,
  }));
  return [...machineEvents, ...governance]
    .filter((item) => item.time)
    .sort((left, right) => new Date(right.time).getTime() - new Date(left.time).getTime())
    .slice(0, 20);
}

function buildUnifiedAuditTimeline() {
  const governance = buildAuditTimelineItems().map((item) => ({
    ...item,
    source: item.kind === "机器事件" ? "machine-event" : "governance",
    jump: resolveAuditJumpModule(item.kind),
  }));
  const sessions = (state.machine.sessions || []).map((item) => ({
    kind: "机器会话",
    name: item.asset_name || "-",
    summary: `${item.protocol || "-"} · ${item.account || "-"} · ${item.status || "-"}`,
    source: "session",
    time: item.started_at || item.ended_at,
    jump: "machine",
  }));
  return [...sessions, ...governance]
    .filter((item) => item.time)
    .sort((left, right) => new Date(right.time).getTime() - new Date(left.time).getTime())
    .slice(0, 30);
}

function resolveAuditJumpModule(kind) {
  if (kind === "机器事件") return "machine";
  if (kind === "系统设置") return "settings";
  if (kind === "身份源") return "source";
  return "roles";
}

function renderSubjectOptions() {
  const subjectType = elements.assignSubjectType.value;
  const options = subjectType === "user" ? state.users.map((item) => ({ value: item.username, label: `${item.display_name} · ${item.email || "no-email"}` })) : state.groups.map((item) => ({ value: item.name, label: `${item.code} · ${item.description || "group"}` }));
  elements.assignSubjectName.value = "";
  elements.subjectOptions.innerHTML = options.map((item) => `<option value="${escapeHtml(item.value)}">${escapeHtml(item.label)}</option>`).join("");
}

function openUserEditModal(user) {
  openFormModal({
    eyebrow: "User",
    title: `编辑用户 ${user.username}`,
    fields: [
      { label: "显示名", name: "display_name", value: user.display_name, required: true },
      { label: "邮箱", name: "email", value: user.email || "", type: "email" },
      { label: "强制开启 MFA", name: "mfa_required", type: "select", value: String(Boolean(user.mfa_required)), options: [{ value: "false", label: "否" }, { value: "true", label: "是" }] },
      { label: "MFA 当前状态", name: "mfa_enabled", type: "select", value: String(Boolean(user.mfa_enabled)), options: [{ value: "false", label: "未启用" }, { value: "true", label: "已启用" }] },
      { label: "状态", name: "status", type: "select", value: user.status, options: [{ value: "active", label: "启用" }, { value: "disabled", label: "禁用" }] },
    ],
    onSubmit: async (form) => {
      await api(`/api/v1/users/${user.id}`, { method: "PUT", body: JSON.stringify({ display_name: form.get("display_name"), email: form.get("email"), status: form.get("status"), mfa_required: String(form.get("mfa_required")) === "true", mfa_enabled: String(form.get("mfa_enabled")) === "true" }) });
      toast("用户信息已更新");
      await loadUsers();
    },
  });
}

function openPasswordModal(user) {
  openFormModal({
    eyebrow: "Password",
    title: `重置密码 ${user.username}`,
    copy: "密码至少 8 位。",
    fields: [{ label: "新密码", name: "password", type: "password", required: true, minlength: 8 }],
    submitText: "更新密码",
    onSubmit: async (form) => {
      await api(`/api/v1/users/${user.id}/password`, { method: "PUT", body: JSON.stringify({ password: form.get("password") }) });
      toast("密码已重置");
    },
  });
}

function openGroupEditModal(group) {
  openFormModal({
    eyebrow: "Group",
    title: `编辑用户组 ${group.name}`,
    fields: [{ label: "组名称", name: "name", value: group.name, required: true }, { label: "组编码", name: "code", value: group.code, required: true }, { label: "说明", name: "description", value: group.description || "" }],
    onSubmit: async (form) => {
      await api(`/api/v1/groups/${group.id}`, { method: "PUT", body: JSON.stringify({ name: form.get("name"), code: form.get("code"), description: form.get("description") }) });
      toast("用户组已更新");
      await loadGroups();
    },
  });
}

function openAddMemberModal(group) {
  openFormModal({
    eyebrow: "Group",
    title: `向 ${group.name} 添加成员`,
    fields: [{ label: "用户名", name: "username", type: "datalist", listId: "group-user-options", required: true, options: state.users.filter((user) => !(group.members || []).some((member) => member.id === user.id)).map((user) => ({ value: user.username, label: user.display_name })), placeholder: "搜索唯一用户名" }],
    submitText: "添加成员",
    onSubmit: async (form) => {
      const username = String(form.get("username") || "").trim();
      const user = state.users.find((item) => item.username === username);
      if (!user) throw new Error("请选择有效用户名");
      await api(`/api/v1/groups/${group.id}/members`, { method: "POST", body: JSON.stringify({ user_id: user.id }) });
      toast("组成员已添加");
      await Promise.all([loadGroups(), loadUsers()]);
    },
  });
}

function openRoleEditModal(role) {
  openFormModal({
    eyebrow: "Role",
    title: `编辑角色 ${role.name}`,
    fields: [{ label: "角色名称", name: "name", value: role.name, required: true }, { label: "角色编码", name: "code", value: role.code, required: true }, { label: "说明", name: "description", value: role.description || "" }],
    onSubmit: async (form) => {
      await api(`/api/v1/roles/${role.id}`, { method: "PUT", body: JSON.stringify({ name: form.get("name"), code: form.get("code"), description: form.get("description") }) });
      toast("角色已更新");
      await Promise.all([loadRoles(), loadBindings(), loadUsers(), loadGroups()]);
    },
  });
}

function openSourceEditModal(source) {
  openFormModal({
    eyebrow: "Identity Source",
    title: `编辑身份源 ${source.name}`,
    fields: [{ label: "名称", name: "name", value: source.name, required: true }, { label: source.type === "ldap" ? "Host" : "Issuer URL", name: "endpoint", value: source.type === "ldap" ? (source.host || "") : (source.issuer_url || "") }, { label: "Port", name: "port", value: String(source.port || 389), type: "number" }],
    onSubmit: async (form) => {
      await api(`/api/v1/identity-sources/${source.id}`, { method: "PUT", body: JSON.stringify({ name: form.get("name"), type: source.type, enabled: source.enabled, host: source.type === "ldap" ? form.get("endpoint") : "", issuer_url: source.type === "oidc" ? form.get("endpoint") : "", port: Number(form.get("port") || 389), base_dn: source.base_dn || "", bind_dn: source.bind_dn || "", user_filter: source.user_filter || "", client_id: source.client_id || "", redirect_url: source.redirect_url || "", sync_mode: source.sync_mode || "login_only" }) });
      toast("身份源已更新");
      await loadSources();
    },
  });
}

function openSettingEditModal(item) {
  openFormModal({
    eyebrow: "System Setting",
    title: `编辑设置 ${item.key}`,
    fields: [{ label: "Value", name: "value", value: item.value || "" }, { label: "说明", name: "description", value: item.description || "" }],
    onSubmit: async (form) => {
      await api("/api/v1/system-settings", { method: "POST", body: JSON.stringify({ key: item.key, value: form.get("value"), description: form.get("description") }) });
      toast("设置已更新");
      await loadSettings();
    },
  });
}

async function toggleUserStatus(user) {
  await api(`/api/v1/users/${user.id}`, { method: "PUT", body: JSON.stringify({ display_name: user.display_name, email: user.email, status: user.status === "active" ? "disabled" : "active" }) });
  toast(`用户已${user.status === "active" ? "禁用" : "启用"}`);
  await loadUsers();
}

async function deleteUser(user) {
  if (!(await confirmAction({ eyebrow: "User", title: "删除用户", copy: `确认删除用户 ${user.username} ? 删除后无法恢复。` }))) return;
  await api(`/api/v1/users/${user.id}`, { method: "DELETE" });
  toast("用户已删除");
  await Promise.all([loadUsers(), loadGroups(), loadBindings()]);
}

async function deleteGroup(group) {
  if (!(await confirmAction({ eyebrow: "Group", title: "删除用户组", copy: `确认删除用户组 ${group.name} ? 组成员关系和组授权会一起清理。` }))) return;
  await api(`/api/v1/groups/${group.id}`, { method: "DELETE" });
  toast("用户组已删除");
  await Promise.all([loadGroups(), loadBindings()]);
}

async function deleteRole(role) {
  if (!(await confirmAction({ eyebrow: "Role", title: "删除角色", copy: `确认删除角色 ${role.name} ? 该角色的授权绑定也会一起移除。` }))) return;
  await api(`/api/v1/roles/${role.id}`, { method: "DELETE" });
  toast("角色已删除");
  await Promise.all([loadRoles(), loadBindings(), loadUsers(), loadGroups()]);
}

async function removeGroupMember(groupID, userID) {
  if (!(await confirmAction({ eyebrow: "Group Member", title: "移除组成员", copy: "确认把该用户从当前用户组中移除吗？" }))) return;
  await api(`/api/v1/groups/${groupID}/members/${userID}`, { method: "DELETE" });
  toast("成员已移除");
  await Promise.all([loadGroups(), loadUsers()]);
}

async function testSource(id) {
  const payload = await api(`/api/v1/identity-sources/${id}/test`, { method: "POST" });
  toast(payload.data.message || "测试完成");
}

async function deleteSource(source) {
  if (!(await confirmAction({ eyebrow: "Identity Source", title: "删除身份源", copy: `确认删除身份源 ${source.name} ? 删除后对应登录配置会失效。` }))) return;
  await api(`/api/v1/identity-sources/${source.id}`, { method: "DELETE" });
  toast("身份源已删除");
  await loadSources();
}

async function removeBinding(binding) {
  const subjectLabel = binding.subject_name || binding.subject_id;
  if (!(await confirmAction({ eyebrow: "Binding", title: "解绑角色", copy: `确认解绑 ${subjectLabel} -> ${binding.role_name} ?` }))) return;
  await api(`/api/v1/role-bindings/${binding.id}`, { method: "DELETE" });
  toast("角色绑定已移除");
  await Promise.all([loadBindings(), loadUsers(), loadGroups()]);
}

async function deleteSetting(item) {
  if (!(await confirmAction({ eyebrow: "System Setting", title: "删除系统设置", copy: `确认删除设置 ${item.key} ?` }))) return;
  await api(`/api/v1/system-settings/${item.id}`, { method: "DELETE" });
  toast("设置已删除");
  await loadSettings();
}
