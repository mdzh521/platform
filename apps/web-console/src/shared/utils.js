export function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function emptyState(text) {
  return `<div class="tip-box">${escapeHtml(text)}</div>`;
}

export function findById(list, id) {
  return list.find((item) => String(item.id) === String(id));
}

export function filterItems(items, keyword, fields) {
  if (!keyword) return items;
  return items.filter((item) =>
    fields.some((field) => String(item[field] || "").toLowerCase().includes(keyword))
  );
}

export function formatDateTime(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function renderTagList(items, key, cls) {
  if (!items?.length) return `<span class="muted-label">未分配</span>`;
  return `<div class="role-tags">${items.map((item) => `<span class="${cls}">${escapeHtml(item[key])}</span>`).join("")}</div>`;
}

export function stripAnsi(value) {
  return String(value ?? "")
    .replace(/\u001b\][^\u0007]*(\u0007|\u001b\\)/g, "")
    .replace(/\u001b\[[0-?]*[ -/]*[@-~]/g, "")
    .replace(/\u001b[@-_]/g, "");
}

export function normalizeTerminalChunk(value) {
  const cleaned = stripAnsi(value)
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "");
  let output = "";
  for (const char of cleaned) {
    if (char === "\b" || char === "\u007f") {
      output = output.slice(0, -1);
      continue;
    }
    output += char;
  }
  return output;
}

export function applyTerminalChunk(currentValue, chunkValue) {
  const cleaned = stripAnsi(chunkValue)
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "");
  let output = String(currentValue ?? "");
  for (const char of cleaned) {
    if (char === "\b" || char === "\u007f") {
      output = output.slice(0, -1);
      continue;
    }
    output += char;
  }
  return output;
}
