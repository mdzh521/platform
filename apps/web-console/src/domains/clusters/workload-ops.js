import { api } from "../../core/api.js";
import { state } from "../../core/state.js";
import { confirmAction, openFormModal, showErrorDialog, toast } from "../../core/ui.js";
import { formatDateTime } from "../../shared/utils.js";

let ctx = null;

export function configureWorkloadOps(deps) {
  ctx = deps;
}

export async function openInteractiveTerminalModal(workload, preferredPodName = "") {
  const podOptions = (state.workloadInspector.pods || []).map((item) => ({
    value: item.name,
    label: `${item.name} (${item.status || "-"})`,
  }));
  openFormModal({
    eyebrow: "Terminal",
    title: `进入终端 ${workload.name}`,
    copy: "这是持续交互式终端，会直接把键盘输入发送到容器里的 shell，更接近 Kuboard 的进入容器体验。",
    fields: [
      { label: "目标 Pod", name: "pod_name", type: "datalist", listId: "pod-terminal-options", value: preferredPodName || state.workloadInspector.selectedPodName || state.workloadInspector.logSourcePod || podOptions[0]?.value || "", options: podOptions, required: true, placeholder: "选择一个 Pod" },
      { label: "容器名称", name: "container_name", value: "", placeholder: "可选，留空则使用默认容器" },
      { label: "Shell", name: "shell", value: "/bin/sh -i", placeholder: "/bin/sh -i" },
      { label: "会话超时(秒)", name: "timeout_seconds", type: "number", value: "1800", required: true },
    ],
    submitText: "进入终端",
    onSubmit: async (form) => {
      const timeoutSeconds = Number(form.get("timeout_seconds") || 1800);
      state.business.workloadInspectorView = "logs";
      await ctx.openInteractiveTerminal({
        workloadID: workload.id,
        podName: String(form.get("pod_name") || "").trim(),
        containerName: String(form.get("container_name") || "").trim(),
        shell: String(form.get("shell") || "/bin/sh").trim() || "/bin/sh",
        timeoutSeconds: Number.isFinite(timeoutSeconds) ? timeoutSeconds : 1800,
      });
      toast("终端已打开");
    },
  });
}

export async function openExecCommandModal(workload, preferredPodName = "") {
  const podOptions = (state.workloadInspector.pods || []).map((item) => ({
    value: item.name,
    label: `${item.name} (${item.status || "-"})`,
  }));
  openFormModal({
    eyebrow: "Exec",
    title: `执行命令 ${workload.name}`,
    copy: "这是非交互式 Pod 命令执行。适合快速查看环境、配置文件和网络状态，不是持续交互终端。",
    fields: [
      { label: "目标 Pod", name: "pod_name", type: "datalist", listId: "pod-exec-options", value: preferredPodName || state.workloadInspector.selectedPodName || state.workloadInspector.logSourcePod || podOptions[0]?.value || "", options: podOptions, required: true, placeholder: "选择一个 Pod" },
      { label: "容器名称", name: "container_name", value: "", placeholder: "可选，留空则使用默认容器" },
      { label: "命令", name: "command", type: "textarea", rows: 4, value: "printenv", placeholder: "例如 printenv 或 cat /etc/resolv.conf" },
      { label: "超时时间(秒)", name: "timeout_seconds", type: "number", value: "15", required: true },
    ],
    submitText: "执行命令",
    onSubmit: async (form) => {
      const timeoutSeconds = Number(form.get("timeout_seconds") || 15);
      const payload = await api(`/api/v1/k8s/workloads/${workload.id}/exec`, {
        method: "POST",
        body: JSON.stringify({
          pod_name: String(form.get("pod_name") || "").trim(),
          container_name: String(form.get("container_name") || "").trim(),
          command: String(form.get("command") || "").trim(),
          timeout_seconds: Number.isFinite(timeoutSeconds) ? timeoutSeconds : 15,
        }),
      });
      state.workloadInspector.execResult = payload.data || null;
      state.business.workloadInspectorView = "logs";
      ctx.renderWorkloadDetailPanel();
      if (payload.data?.success) {
        toast("命令执行完成");
      } else {
        await showErrorDialog({ title: "命令执行失败", copy: payload.data?.error_message || "命令执行返回错误" });
      }
    },
  });
}

export async function openScaleWorkloadModal(workload) {
  openFormModal({
    eyebrow: "Scale",
    title: `扩缩容 ${workload.name}`,
    copy: "修改期望副本数后，平台会回刷工作负载详情和发布状态。",
    fields: [
      { label: "目标副本数", name: "replicas", type: "number", value: String(workload.replicas ?? 0), required: true },
    ],
    submitText: "执行扩缩容",
    onSubmit: async (form) => {
      const replicas = Number(form.get("replicas"));
      if (!Number.isInteger(replicas) || replicas < 0) {
        throw new Error("副本数必须是大于等于 0 的整数");
      }
      const payload = await api(`/api/v1/k8s/workloads/${workload.id}/scale`, {
        method: "POST",
        body: JSON.stringify({ replicas }),
      });
      toast(payload.data.message ? `${payload.data.message}，当前目标副本 ${payload.data.replicas}` : "扩缩容已提交");
      await refreshInspectorAfterMutation();
    },
  });
}

export async function openUpdateImageModal(workload) {
  const payload = await api(`/api/v1/k8s/workloads/${workload.id}/container-images`);
  const containers = payload.data.containers || [];
  if (!containers.length) {
    throw new Error("当前工作负载没有可编辑的容器镜像");
  }
  openFormModal({
    eyebrow: "Image",
    title: `更新镜像 ${workload.name}`,
    copy: containers.length > 1
      ? "当前工作负载包含多个容器，可以逐个修改镜像。未改动的容器也会按当前值一起提交。"
      : "当前工作负载为单容器，直接修改主容器镜像即可。",
    fields: containers.map((item, index) => ({
      label: `${index + 1}. ${item.name}`,
      name: `container_image_${index}`,
      value: item.image || "",
      required: true,
      placeholder: "例如 nginx:1.28",
    })),
    submitText: "更新镜像",
    onSubmit: async (form) => {
      const updates = containers.map((item, index) => ({
        name: item.name,
        image: String(form.get(`container_image_${index}`) || "").trim(),
      }));
      if (updates.some((item) => !item.image)) {
        throw new Error("目标镜像不能为空");
      }
      const endpoint = updates.length === 1 ? `/api/v1/k8s/workloads/${workload.id}/image` : `/api/v1/k8s/workloads/${workload.id}/images`;
      const body = updates.length === 1
        ? JSON.stringify({ image: updates[0].image })
        : JSON.stringify({ containers: updates });
      const result = await api(endpoint, {
        method: "POST",
        body,
      });
      toast(result.data.message || "镜像更新已提交");
      await refreshInspectorAfterMutation();
    },
  });
}

export async function openStatefulSetSettingsModal(workload) {
  const payload = await api(`/api/v1/k8s/workloads/${workload.id}/statefulset-settings`);
  const current = payload.data || {};
  openFormModal({
    eyebrow: "StatefulSet",
    title: `编辑 ${workload.name} 设置`,
    copy: `当前 Pod Management 为 ${current.pod_management_policy || "OrderedReady"}。这一项在 Kubernetes 里更适合通过重建 StatefulSet 处理，所以先保留只读；当前版本支持修改 Service 模式和 Rolling Partition。`,
    fields: [
      { label: "Service 模式", name: "service_mode", type: "select", value: current.service_mode || "ClusterIP", options: [{ value: "Headless", label: "Headless" }, { value: "ClusterIP", label: "ClusterIP" }, { value: "LoadBalancer", label: "LoadBalancer" }] },
      { label: "Rolling Partition", name: "rolling_partition", type: "number", value: String(current.rolling_partition ?? 0), required: true },
    ],
    submitText: "保存设置",
    onSubmit: async (form) => {
      const rollingPartition = Number(form.get("rolling_partition"));
      if (!Number.isInteger(rollingPartition) || rollingPartition < 0) {
        throw new Error("Rolling Partition 必须是大于等于 0 的整数");
      }
      const result = await api(`/api/v1/k8s/workloads/${workload.id}/statefulset-settings`, {
        method: "POST",
        body: JSON.stringify({
          service_mode: String(form.get("service_mode") || "ClusterIP"),
          pod_management_policy: current.pod_management_policy || "OrderedReady",
          rolling_partition: rollingPartition,
        }),
      });
      toast(result.data.message || "StatefulSet 设置已更新");
      await refreshInspectorAfterMutation();
    },
  });
}

export async function restartWorkload(workload) {
  const ok = await confirmAction({
    eyebrow: "Restart",
    title: `重启 ${workload.name}`,
    copy: "这会删除当前工作负载关联的 Pod，让控制器重新拉起新实例。适合快速重建异常 Pod，但会触发一次实际重建。",
    confirmText: "确认重启",
  });
  if (!ok) return;
  const payload = await api(`/api/v1/k8s/workloads/${workload.id}/restart`, { method: "POST" });
  toast(payload.data.deleted_count ? `已触发重启，删除 ${payload.data.deleted_count} 个 Pod` : (payload.data.message || "重启已提交"));
  await refreshInspectorAfterMutation();
}

export async function rolloutRestartWorkload(workload) {
  const ok = await confirmAction({
    eyebrow: "Rollout",
    title: `Rollout Restart ${workload.name}`,
    copy: "这会更新 PodTemplate 注解，触发一次标准滚动重启。相比直接删 Pod，更适合有序更新 Deployment / StatefulSet。",
    confirmText: "确认执行",
  });
  if (!ok) return;
  const payload = await api(`/api/v1/k8s/workloads/${workload.id}/rollout-restart`, { method: "POST" });
  toast(payload.data.restarted_at ? `已触发滚动重启，时间 ${formatDateTime(payload.data.restarted_at)}` : (payload.data.message || "滚动重启已提交"));
  await refreshInspectorAfterMutation();
}

async function refreshInspectorAfterMutation() {
  const clusterID = ctx.selectedWorkloadClusterID();
  state.workloadInspector.rollout = null;
  state.workloadInspector.resources = null;
  state.workloadInspector.resourceDetail = null;
  state.workloadInspector.pods = [];
  state.workloadInspector.events = [];
  state.workloadInspector.logs = "";
  state.workloadInspector.logSourcePod = "";
  state.workloadInspector.selectedPodName = "";
  state.workloadInspector.manifest = "";
  if (clusterID) {
    await api(`/api/v1/k8s/clusters/${clusterID}/sync-workloads`, { method: "POST" });
  }
  await ctx.refreshSelectedWorkload();
}
