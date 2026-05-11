import { elements, drawerMap } from "./dom.js";
import { escapeHtml } from "../shared/utils.js";

export function openDrawer(type) {
  drawerMap[type]?.classList.remove("hidden");
}

export function closeDrawer(type) {
  drawerMap[type]?.classList.add("hidden");
}

export function showLogin() {
  elements.loginScreen.classList.remove("hidden");
  elements.appShell.classList.add("hidden");
}

export function showApp() {
  elements.loginScreen.classList.add("hidden");
  elements.appShell.classList.remove("hidden");
}

export function renderSession(currentUser, permissions) {
  elements.currentUserName.textContent = currentUser ? `${currentUser.display_name} (${currentUser.username})` : "-";
  elements.permissionBadge.textContent = `${permissions.length} 项权限`;
}

export function bindAction(root, action, handler) {
  root.querySelectorAll(`[data-action='${action}']`).forEach((button) => {
    button.addEventListener("click", () => handler(button.dataset.id));
  });
}

export function bindSearch(element, onInput) {
  element?.addEventListener("input", (event) => onInput(String(event.target.value || "").trim().toLowerCase()));
}

export function renderField(field) {
  const wrapperAttrs = [
    field.name ? `data-field-name="${escapeHtml(field.name)}"` : "",
    field.showForProvider ? `data-show-for-provider="${escapeHtml(field.showForProvider)}"` : "",
  ].filter(Boolean).join(" ");
  if (field.type === "section") {
    return `<div class="form-section-heading field-span-2"><span class="eyebrow">${escapeHtml(field.eyebrow || "")}</span><strong>${escapeHtml(field.label)}</strong>${field.copy ? `<p>${escapeHtml(field.copy)}</p>` : ""}</div>`;
  }
  if (field.type === "actions") {
    return `<div class="field-span-2 action-row modal-action-row">${(field.actions || []).map((action) => `<button type="button" class="${escapeHtml(action.tone === "primary" ? "primary-button" : "ghost-button")}" data-cloud-shortcut="${escapeHtml(action.shortcut || "")}">${escapeHtml(action.label || "")}</button>`).join("")}</div>`;
  }
  if (field.type === "note") {
    return `<div class="form-section-note field-span-2">${escapeHtml(field.copy || "")}</div>`;
  }
  if (field.type === "custom") {
    return `<div class="field-span-2" ${wrapperAttrs}>${field.html || ""}</div>`;
  }
  if (field.type === "select") {
    return `<label ${wrapperAttrs}><span>${escapeHtml(field.label)}</span><select name="${escapeHtml(field.name)}" ${field.disabled ? "disabled" : ""}>${field.options.map((option) => `<option value="${escapeHtml(option.value)}" ${option.value === field.value ? "selected" : ""}>${escapeHtml(option.label)}</option>`).join("")}</select></label>`;
  }
  if (field.type === "datalist") {
    return `<label ${wrapperAttrs}><span>${escapeHtml(field.label)}</span><input name="${escapeHtml(field.name)}" list="${escapeHtml(field.listId)}" value="${escapeHtml(field.value || "")}" placeholder="${escapeHtml(field.placeholder || "")}" ${field.required ? "required" : ""} /><datalist id="${escapeHtml(field.listId)}">${field.options.map((option) => `<option value="${escapeHtml(option.value)}">${escapeHtml(option.label || "")}</option>`).join("")}</datalist></label>`;
  }
  if (field.type === "textarea") {
    const uploadBlock = field.upload && !field.readOnly
      ? `
        <div class="field-upload-row">
          <label class="ghost-button field-upload-button">
            <input type="file" class="hidden-file-input" data-file-target="${escapeHtml(field.name)}" ${field.upload.accept ? `accept="${escapeHtml(field.upload.accept)}"` : ""} />
            本地上传
          </label>
          <span class="muted-label">${escapeHtml(field.upload.hint || "选择本地文件后会自动填入")}</span>
        </div>
      `
      : "";
    return `<label ${wrapperAttrs}><span>${escapeHtml(field.label)}</span><textarea name="${escapeHtml(field.name)}" rows="${field.rows || 5}" ${field.placeholder ? `placeholder="${escapeHtml(field.placeholder)}"` : ""} ${field.readOnly ? "readonly" : ""} ${field.disabled ? "disabled" : ""}>${escapeHtml(field.value || "")}</textarea>${uploadBlock}</label>`;
  }
  return `<label ${wrapperAttrs}><span>${escapeHtml(field.label)}</span><input name="${escapeHtml(field.name)}" type="${escapeHtml(field.type || "text")}" value="${escapeHtml(field.value || "")}" ${field.placeholder ? `placeholder="${escapeHtml(field.placeholder)}"` : ""} ${field.required ? "required" : ""} ${field.minlength ? `minlength="${field.minlength}"` : ""} ${field.readOnly ? "readonly" : ""} ${field.disabled ? "disabled" : ""} /></label>`;
}

export function openFormModal({ eyebrow, title, copy, fields, submitText, onSubmit, onOpen, panelClass = "" }) {
  const panel = elements.formModal?.querySelector(".modal-panel");
  if (panel) {
    panel.className = `modal-panel ${panelClass}`.trim();
  }
  elements.formModalEyebrow.textContent = eyebrow || "Edit";
  elements.formModalTitle.textContent = title;
  if (copy) {
    elements.formModalCopy.textContent = copy;
    elements.formModalCopy.classList.remove("hidden");
  } else {
    elements.formModalCopy.classList.add("hidden");
  }
  const showSubmit = submitText !== false;
  elements.formModalForm.innerHTML = `${fields.map(renderField).join("")}${showSubmit ? `<button type="submit" class="primary-button">${escapeHtml(submitText || "保存")}</button>` : ""}`;
  bindFormFileUploads();
  syncConditionalFields(elements.formModalForm);
  elements.formModalForm.onsubmit = async (event) => {
    event.preventDefault();
    if (!onSubmit) {
      closeFormModal();
      return;
    }
    try {
      await onSubmit(new FormData(elements.formModalForm));
      closeFormModal();
    } catch (error) {
      await showErrorDialog({ title: "提交失败", copy: error?.message || "提交失败" });
    }
  };
  elements.formModal.classList.remove("hidden");
  if (onOpen) {
    onOpen(elements.formModalForm);
  }
}

function syncConditionalFields(form) {
  const providerInput = form.querySelector("[name='provider']");
  if (!providerInput) return;

  const applyVisibility = () => {
    const provider = String(providerInput.value || "").trim().toLowerCase();
    form.querySelectorAll("[data-show-for-provider]").forEach((field) => {
      const expected = String(field.dataset.showForProvider || "").trim().toLowerCase();
      const visible = !expected || expected === provider;
      field.classList.toggle("hidden", !visible);
      const input = field.querySelector("input, select, textarea");
      if (input && !visible) {
        input.value = "";
      }
    });
  };

  providerInput.addEventListener("change", applyVisibility);
  applyVisibility();
}

function bindFormFileUploads() {
  elements.formModalForm.querySelectorAll("[data-file-target]").forEach((input) => {
    input.addEventListener("change", async (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      const targetName = input.dataset.fileTarget;
      const target = elements.formModalForm.querySelector(`[name='${targetName}']`);
      if (!target) return;
      try {
        const content = await file.text();
        target.value = content;
        toast(`已载入文件：${file.name}`);
      } catch (_error) {
        showErrorDialog({ title: "读取失败", copy: "读取文件失败" });
      } finally {
        event.target.value = "";
      }
    });
  });
}

export function closeFormModal() {
  elements.formModal.classList.add("hidden");
  elements.formModalForm.innerHTML = "";
  const panel = elements.formModal?.querySelector(".modal-panel");
  if (panel) {
    panel.className = "modal-panel";
  }
}

export function confirmAction({ eyebrow, title, copy, confirmText }) {
  elements.confirmCancel.classList.remove("hidden");
  elements.confirmModalEyebrow.textContent = eyebrow || "Confirm";
  elements.confirmModalTitle.textContent = title || "确认操作";
  elements.confirmModalCopy.textContent = copy || "";
  elements.confirmSubmit.textContent = confirmText || "确认";
  elements.confirmModal.classList.remove("hidden");
  return new Promise((resolve) => {
    const finish = (result) => {
      closeConfirmModal();
      resolve(result);
    };
    elements.confirmSubmit.onclick = () => finish(true);
    elements.confirmCancel.onclick = () => finish(false);
    elements.closeConfirmModal.onclick = () => finish(false);
    elements.closeConfirmModalBackdrop.onclick = () => finish(false);
  });
}

export function showNoticeDialog({ eyebrow, title, copy, confirmText }) {
  elements.confirmCancel.classList.add("hidden");
  elements.confirmModalEyebrow.textContent = eyebrow || "Notice";
  elements.confirmModalTitle.textContent = title || "提示";
  elements.confirmModalCopy.textContent = copy || "";
  elements.confirmSubmit.textContent = confirmText || "知道了";
  elements.confirmModal.classList.remove("hidden");
  return new Promise((resolve) => {
    const finish = () => {
      closeConfirmModal();
      resolve(true);
    };
    elements.confirmSubmit.onclick = finish;
    elements.closeConfirmModal.onclick = finish;
    elements.closeConfirmModalBackdrop.onclick = finish;
    elements.confirmCancel.onclick = finish;
  });
}

export function showErrorDialog({ title, copy, confirmText } = {}) {
  return showNoticeDialog({
    eyebrow: "Error",
    title: title || "操作失败",
    copy: copy || "请求处理失败",
    confirmText: confirmText || "知道了",
  });
}

export function closeConfirmModal() {
  elements.confirmModal.classList.add("hidden");
  elements.confirmCancel.classList.remove("hidden");
  elements.confirmSubmit.onclick = null;
  elements.confirmCancel.onclick = null;
  elements.closeConfirmModal.onclick = null;
  elements.closeConfirmModalBackdrop.onclick = null;
}

export function toast(message) {
  elements.toast.textContent = message;
  elements.toast.classList.remove("hidden");
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => elements.toast.classList.add("hidden"), 2500);
}
