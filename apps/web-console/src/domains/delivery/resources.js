import { elements } from "../../core/dom.js";
import { emptyState } from "../../shared/utils.js";

export function renderResources() {
  if (!elements.cloudResourcesTable) return;
  elements.cloudResourcesTable.innerHTML = `
    <div class="tip-box">
      资源结果已经合并到上方每条任务卡片里。
      直接展开某条任务，就能看到对应资源、同步状态和清理入口。
    </div>
    <div class="action-row compact-action-row">
      <button type="button" class="ghost-button" data-cloud-shortcut="scroll-jobs">返回任务卡片</button>
      <button type="button" class="ghost-button" data-cloud-shortcut="scroll-networks">返回 Foundation Networks</button>
    </div>
  `;
}
