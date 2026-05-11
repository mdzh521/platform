import { api } from "../core/api.js";
import { elements } from "../core/dom.js";
import { state } from "../core/state.js";
import { closeDrawer, closeFormModal, renderSession, showApp, showErrorDialog, toast } from "../core/ui.js";
import { bindK8sEvents, loadK8sData } from "../domains/clusters/index.js";
import { bindMachineEvents, loadMachineData } from "../domains/machines/index.js";
import { bindCloudEvents, loadCloudData } from "../domains/delivery/index.js";
import { bindSystemEvents, loadSystemData } from "../domains/system/index.js";
import { renderDeliveryPage } from "../pages/delivery/index.js";
import { enterClustersPage } from "../pages/clusters/index.js";
import { showLoginPage } from "../pages/login/index.js";
import { enterMachinesPage } from "../pages/machines/index.js";
import { renderOverviewDashboard } from "../pages/overview/index.js";
import { switchSystemModule } from "../pages/system/index.js";
import { loadRuntimeConfig } from "./runtime.js";
import { migrateSessionAuthState, restoreLoginErrorFromStorage, setLoginError } from "./session.js";

window.__bcModuleBooted = false;
let moduleEventsBound = false;

export async function initApp() {
  bindCoreEvents();
  try {
    await loadRuntimeConfig();
    migrateSessionAuthState();
    window.__bcModuleBooted = true;
    if (state.token) {
      await bootstrapApp();
      return;
    }
    showLoginPage();
    restoreLoginErrorFromStorage();
  } catch (error) {
    surfaceFatalError(error);
  }
}

export function surfaceFatalError(error) {
  const message = typeof error === "string" ? error : (error?.message || "前端初始化失败");
  console.error(error);
  if (window.__bcModuleBooted && state.token) {
    showErrorDialog({ title: "操作失败", copy: message });
    return;
  }
  window.__bcModuleBooted = false;
  localStorage.setItem("bc_boot_error", message);
  showLoginPage();
  setLoginError(message);
}

async function bootstrapApp() {
  showApp();
  bindModuleEvents();
  const failures = [];
  const loaders = [
    ["系统治理", loadSystemData],
    ["交付网络", loadCloudData],
    ["集群工作台", loadK8sData],
    ["机器工作台", loadMachineData],
  ];
  await Promise.all(loaders.map(async ([label, loader]) => {
    try {
      await loader();
    } catch (error) {
      console.error(`${label} 初始化失败`, error);
      failures.push(`${label}加载失败：${error?.message || "未知错误"}`);
    }
  }));
  localStorage.removeItem("bc_boot_error");
  renderSession(state.currentUser, state.permissions);
  renderOverviewDashboard();
  switchTopModule("overview", document.querySelector("[data-top-module='overview']"));
  if (failures.length) {
    showErrorDialog({ title: "模块加载失败", copy: failures[0] });
    console.warn("模块启动失败", failures);
  }
}

function bindCoreEvents() {
  elements.loginForm.addEventListener("submit", onLogin);
  elements.closeFormModal.addEventListener("click", closeFormModal);
  elements.closeFormModalBackdrop.addEventListener("click", closeFormModal);
  elements.logoutButton.addEventListener("click", () => logout());
  document.querySelectorAll("[data-close-drawer]").forEach((button) => {
    button.addEventListener("click", () => closeDrawer(button.dataset.closeDrawer));
  });
  document.querySelectorAll(".tab-button").forEach((button) => button.addEventListener("click", () => switchTab(button.dataset.tab)));
  document.querySelectorAll(".nav-item").forEach((button) => button.addEventListener("click", () => switchSystemModule(button.dataset.module, button)));
  document.querySelectorAll("[data-top-module]").forEach((button) => button.addEventListener("click", () => {
    const module = button.dataset.topModule;
    const navButton = document.querySelector(`.top-tab[data-top-module='${module}']`) || button;
    switchTopModule(module, navButton);
  }));
  document.querySelectorAll("[data-system-module-jump]").forEach((button) => button.addEventListener("click", () => {
    switchTopModule("system", document.querySelector(".top-tab[data-top-module='system']"));
    switchSystemModule(button.dataset.systemModuleJump || "user-management", document.querySelector(`.nav-item[data-module='${button.dataset.systemModuleJump || "user-management"}']`));
    if (button.dataset.systemTabJump) {
      switchTab(button.dataset.systemTabJump);
    }
  }));
  window.addEventListener("bc:navigate", (event) => {
    const detail = event.detail || {};
    if (detail.topModule) {
      switchTopModule(detail.topModule, document.querySelector(`.top-tab[data-top-module='${detail.topModule}']`));
    }
    if (detail.systemModule) {
      switchSystemModule(detail.systemModule, document.querySelector(`.nav-item[data-module='${detail.systemModule}']`));
    }
    if (detail.systemTab) {
      switchTab(detail.systemTab);
    }
  });
}

function bindModuleEvents() {
  if (moduleEventsBound) return;
  moduleEventsBound = true;
  for (const [label, binder] of [
    ["系统治理", bindSystemEvents],
    ["交付网络", bindCloudEvents],
    ["集群工作台", bindK8sEvents],
    ["机器工作台", bindMachineEvents],
  ]) {
    try {
      binder();
    } catch (error) {
      console.error(`${label} 事件绑定失败`, error);
    }
  }
}

async function onLogin(event) {
  event.preventDefault();
  const form = new FormData(event.target);
  setLoginError("");
  try {
    const response = await fetch("/api/v1/auth/login/local", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: form.get("username"), password: form.get("password") }),
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || "登录失败");
    state.token = payload.data.token;
    state.currentUser = payload.data.user;
    state.permissions = payload.data.permissions || [];
    sessionStorage.setItem("bc_token", state.token);
    sessionStorage.setItem("bc_current_user", JSON.stringify(state.currentUser));
    sessionStorage.setItem("bc_permissions", JSON.stringify(state.permissions));
    localStorage.removeItem("bc_token");
    localStorage.removeItem("bc_current_user");
    localStorage.removeItem("bc_permissions");
    localStorage.removeItem("bc_boot_error");
    await bootstrapApp();
    toast("登录成功");
  } catch (error) {
    setLoginError(error?.message || "登录失败");
  }
}

function switchTab(tabName) {
  document.querySelectorAll(".tab-button").forEach((button) => button.classList.toggle("active", button.dataset.tab === tabName));
  document.querySelectorAll(".tab-panel").forEach((panel) => panel.classList.toggle("active", panel.id === `tab-${tabName}`));
}

function switchTopModule(module, button) {
  state.currentTopModule = module;
  document.querySelectorAll(".top-tab").forEach((item) => item.classList.remove("active"));
  button?.classList.add("active");
  document.querySelectorAll(".top-module").forEach((section) => section.classList.add("hidden"));
  document.getElementById(`top-module-${module}`).classList.remove("hidden");
  const titles = { overview: "总览", cloud: "交付网络", business: "集群工作台", system: "系统治理 / 用户与身份", machine: "机器工作台" };
  elements.moduleTitle.textContent = titles[module];
  if (module === "overview") {
    renderOverviewDashboard();
  }
  if (module === "cloud") {
    renderDeliveryPage();
  }
  if (module === "business") {
    enterClustersPage();
  }
  if (module === "machine") {
    enterMachinesPage();
  }
}

function logout(errorMessage = "") {
  state.token = "";
  state.currentUser = null;
  state.permissions = [];
  sessionStorage.removeItem("bc_token");
  sessionStorage.removeItem("bc_current_user");
  sessionStorage.removeItem("bc_permissions");
  localStorage.removeItem("bc_token");
  localStorage.removeItem("bc_current_user");
  localStorage.removeItem("bc_permissions");
  if (errorMessage) {
    localStorage.setItem("bc_boot_error", errorMessage);
  } else {
    localStorage.removeItem("bc_boot_error");
  }
  showLoginPage();
  if (errorMessage) {
    setLoginError(errorMessage);
  } else {
    setLoginError("");
  }
}

window.backendCenter = { api };
