import { state } from "../core/state.js";

export async function loadRuntimeConfig() {
  try {
    const response = await fetch("/api/v1/runtime/config");
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || "运行时配置加载失败");
    state.runtimeConfig = payload.data || state.runtimeConfig;
  } catch (error) {
    console.warn("运行时配置加载失败，降级使用默认配置", error);
  }
}
