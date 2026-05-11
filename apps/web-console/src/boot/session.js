import { elements } from "../core/dom.js";

export function migrateSessionAuthState() {
  const token = localStorage.getItem("bc_token");
  const currentUser = localStorage.getItem("bc_current_user");
  const permissions = localStorage.getItem("bc_permissions");
  if (token && !sessionStorage.getItem("bc_token")) {
    sessionStorage.setItem("bc_token", token);
  }
  if (currentUser && !sessionStorage.getItem("bc_current_user")) {
    sessionStorage.setItem("bc_current_user", currentUser);
  }
  if (permissions && !sessionStorage.getItem("bc_permissions")) {
    sessionStorage.setItem("bc_permissions", permissions);
  }
  localStorage.removeItem("bc_token");
  localStorage.removeItem("bc_current_user");
  localStorage.removeItem("bc_permissions");
}

export function restoreLoginErrorFromStorage() {
  const bootError = localStorage.getItem("bc_boot_error");
  if (!bootError) {
    setLoginError("");
    return;
  }
  setLoginError(bootError);
}

export function setLoginError(message = "") {
  const text = String(message || "").trim();
  elements.loginError.textContent = text;
  elements.loginError.classList.toggle("hidden", !text);
}
