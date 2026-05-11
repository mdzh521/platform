# Phase6 Cluster Addon Runbook

日期：2026-04-27

## 当前结论

这份 runbook 只覆盖 `platform-managed Helm addon`。

当前已经确认：

- `platform_addon_contract -> cluster_addon_execution -> cluster-addon-worker -> result`
  这条执行链已经真实跑通
- `job 55` 的 `aws_load_balancer_controller` contract 已经真实下发
- `cluster_addon_executions.id=1` 也已经被 worker 真实 claim 并执行 Helm 安装

当前真实阻塞不是代码主链断掉，而是运行网络面：

- 目标 EKS 是 `private endpoint`
- compose 里的 `cluster-addon-worker` 无法直连集群私网地址
- 真实失败证据已经落库：
  - `cluster_addon_executions.id=1`
  - `status=failed`
  - `error_message` 为 `Kubernetes cluster unreachable ... i/o timeout`

所以当前结论必须写清楚：

- `platform-managed Helm addon` 执行链已完成
- `AWS Load Balancer Controller` 已完成真实安装尝试
- 但 `compose addon worker` 还不能作为 private-only EKS 的正式安装运行面

## 已确认的真实证据

- `source job id = 55`
- `cloud resource id = 252`
- `cluster id = 7`
- `addon execution id = 1`
- `addon key = aws_load_balancer_controller`

当前失败原因：

- `helm upgrade --install aws-load-balancer-controller ...`
- 访问 `https://9F1819D53EC41BBB5F7F7C2261B4E8CF.gr7.ap-southeast-1.eks.amazonaws.com/version`
- `dial tcp 10.10.100.76:443: i/o timeout`

这说明：

- IAM / addon contract / Helm 调用入口已经成立
- 失败点在 `worker -> private EKS endpoint` 的网络可达性

## 推荐运行面

### P0 真实安装取证

优先推荐：

- 在宿主机直接运行 `cluster-addon-worker`
- 或把 worker 放进能访问该 VPC 私网的 runtime

不再推荐：

- 直接把本地 compose 容器当作 private-only EKS 的正式 addon 安装面

## 推荐启动方式

### 1. backend 与基础 runtime

```bash
cd /Users/alex/ops/platform-center
docker compose up -d backend terraform-runner resource-sync-worker
```

### 2. 推荐 host process 跑 addon worker

```bash
cd /Users/alex/ops/platform-center/apps/cluster-addon-worker
BACKEND_API_BASE_URL=http://127.0.0.1:18080 \
BACKEND_API_TOKEN=runner-dev-token \
AWS_REGION=ap-southeast-1 \
go run ./cmd/cluster-addon-worker
```

### 3. 重新触发 addon 安装

可直接把目标 execution 重新置为 `queued`，或走 API retry。

当前已验证的 execution：

- `cluster_addon_executions.id=1`

## metrics-server 第一版结论

当前决定：

- `metrics-server` 不进入第一版标准集群

原因不是功能价值不足，而是当前 phase6 真实验证已经证明：

- `platform-managed Helm addon` 的正式运行面还需要先解决 private EKS 网络可达性
- `AWS Load Balancer Controller` 作为更高优先级业务 addon 还未完成最终安装闭环

所以当前策略保持为：

- `metrics-server = future_contract`
- 等 `AWS Load Balancer Controller` 在正式 runtime 上安装闭环后，再决定是否进入第一版标准集群

## 下一步

1. 用 host process 或 VPC 内 worker 重新跑 `aws_load_balancer_controller`
2. 回报最终成功证据：
   - addon execution
   - Helm release
   - deployment
   - service account / IRSA annotation
   - controller pods
3. 在 ALB Controller 真实闭环后，再评估 `metrics-server`
