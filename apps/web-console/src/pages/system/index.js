import { elements } from "../../core/dom.js";
import { renderAccessControlPage, renderAuditCenterPage } from "../../domains/system/index.js";

export function switchSystemModule(module, button) {
  const normalizedModules = new Set(["user-management", "access-control", "audit-center", "system-settings"]);
  const normalized = normalizedModules.has(module) ? module : "user-management";
  document.querySelectorAll(".nav-item").forEach((item) => item.classList.toggle("active", item.dataset.module === normalized));
  button?.classList.add("active");
  const userPage = document.getElementById("module-user-management");
  const accessPage = document.getElementById("module-access-control");
  const auditPage = document.getElementById("module-audit-center");
  const settingsPage = document.getElementById("module-system-settings");
  const isUserManagement = normalized === "user-management";
  const isAccessControl = normalized === "access-control";
  const isAuditCenter = normalized === "audit-center";
  const isSettings = normalized === "system-settings";
  userPage.classList.toggle("hidden", !isUserManagement);
  accessPage.classList.toggle("hidden", !isAccessControl);
  auditPage.classList.toggle("hidden", !isAuditCenter);
  settingsPage.classList.toggle("hidden", !isSettings);
  if (isUserManagement) {
    elements.moduleTitle.textContent = "系统治理 / 用户与身份";
    return;
  }
  if (isAccessControl) {
    elements.moduleTitle.textContent = "系统治理 / 访问控制";
    renderAccessControlPage();
    return;
  }
  if (isAuditCenter) {
    elements.moduleTitle.textContent = "系统治理 / 审计中心";
    renderAuditCenterPage();
    return;
  }
  if (isSettings) {
    elements.moduleTitle.textContent = "系统治理 / 系统设置";
  }
}
