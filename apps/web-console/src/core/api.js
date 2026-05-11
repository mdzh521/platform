import { state } from "./state.js";

export async function api(url, options = {}) {
  const apiBaseURL = String(state.runtimeConfig?.frontend?.api_base_url || "").trim();
  const headers = { "Content-Type": "application/json", ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const response = await fetch(`${apiBaseURL}${url}`, { ...options, headers });
  const payload = await parseResponsePayload(response);
  if (!response.ok) {
    const message = payload?.error || payload?.message || response.statusText || "请求失败";
    if (response.status === 401 && message === "invalid token") {
      state.token = "";
      state.currentUser = null;
      state.permissions = [];
      sessionStorage.removeItem("bc_token");
      sessionStorage.removeItem("bc_current_user");
      sessionStorage.removeItem("bc_permissions");
      localStorage.removeItem("bc_token");
      localStorage.removeItem("bc_current_user");
      localStorage.removeItem("bc_permissions");
      localStorage.setItem("bc_boot_error", "登录已失效，请重新登录");
      window.location.replace("/");
      throw new Error("登录已失效，请重新登录");
    }
    throw new Error(message);
  }
  return payload;
}

async function parseResponsePayload(response) {
  const contentType = String(response.headers.get("Content-Type") || "").toLowerCase();
  const raw = await response.text();
  if (!raw) return {};
  if (contentType.includes("application/json")) {
    try {
      return JSON.parse(raw);
    } catch (_error) {
      return { error: raw };
    }
  }
  try {
    return JSON.parse(raw);
  } catch (_error) {
    return { error: raw };
  }
}

export function buildAPIURL(path) {
  const apiBaseURL = String(state.runtimeConfig?.frontend?.api_base_url || "").trim();
  return `${apiBaseURL}${path}`;
}

export function buildWebSocketURL(path, params = {}) {
  const apiBaseURL = String(state.runtimeConfig?.frontend?.api_base_url || "").trim();
  const base = apiBaseURL
    ? new URL(apiBaseURL, window.location.origin)
    : new URL(window.location.origin);
  const protocol = base.protocol === "https:" ? "wss:" : "ws:";
  const wsURL = new URL(`${protocol}//${base.host}${path}`);
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    wsURL.searchParams.set(key, String(value));
  });
  return wsURL.toString();
}
