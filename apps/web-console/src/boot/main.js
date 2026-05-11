import { initApp, surfaceFatalError } from "./app.js";

document.addEventListener("DOMContentLoaded", initApp);
window.addEventListener("error", (event) => surfaceFatalError(event.error || event.message || "前端初始化失败"));
window.addEventListener("unhandledrejection", (event) => surfaceFatalError(event.reason?.message || event.reason || "前端异步初始化失败"));
