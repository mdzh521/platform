# Phase4 Enrollment Runtime

这份文档只回答 phase4 的 3 个问题：

1. `cluster-enrollment-worker` 现在应该跑在哪里
2. provider 凭据怎么进
3. 真正实跑 EKS / ACK 时，证据链怎么留

## 当前推荐 runtime

P0 真实 provider enrollment：

- `cluster-enrollment-worker` 优先跑在 `本机进程`
- `machine-enrollment-worker` 可以继续跑在 compose 容器或本机

当前不把 `cluster-enrollment-worker` 的 compose 容器定义成首选实跑位置，原因很直接：

- 当前代码仍依赖 CLI fallback
- compose 镜像已经接好运行入口，但镜像本身没有内置 `aws` / `aliyun`
- 所以真跑 provider 时，最短路径仍是本机进程 + 本机 CLI + 本机凭据文件

## 运行位置结论

### P0 实跑推荐

- `terraform-runner`：compose 容器
- `resource-sync-worker`：compose 容器
- `cluster-enrollment-worker`：本机进程
- `machine-enrollment-worker`：compose 容器或本机都可

### 可选 compose 入口

当前 compose 入口 [docker-compose.yml](/Users/alex/ops/platform-center/ops/compose/docker-compose.yml:1) 已加入：

- `machine-enrollment-worker`
- `cluster-enrollment-worker`

用途：

- 提供统一运行面
- 方便后面把 provider adapter 切到 SDK / OpenAPI 后直接容器化

当前阶段不要把它误认为“已经具备真实 provider CLI 环境”。

## Provider 凭据

### AWS

支持两类来源：

1. 环境变量
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `AWS_SESSION_TOKEN`
   - `AWS_REGION`
   - `AWS_DEFAULT_REGION`
2. 本机凭据文件
   - `~/.aws/credentials`
   - `~/.aws/config`

### Alicloud

当前 CLI fallback 侧建议准备：

1. 环境变量
   - `ALIBABA_CLOUD_ACCESS_KEY_ID`
   - `ALIBABA_CLOUD_ACCESS_KEY_SECRET`
   - `ALICLOUD_REGION_ID`
2. 本机配置文件
   - `~/.aliyun/config.json`

## 最小前置检查

执行：

```bash
zsh /Users/alex/ops/platform-center/ops/scripts/check_enrollment_runtime.sh
```

这一步要回答：

- backend 能不能通
- worker config 在不在
- `aws` / `aliyun` CLI 在不在
- provider 凭据有没有准备

## 启动方式

### 推荐：本机进程跑 cluster enrollment

```bash
cd /Users/alex/ops/platform-center/apps/cluster-enrollment-worker
BACKEND_API_BASE_URL=http://127.0.0.1:18080 \
BACKEND_API_TOKEN=runner-dev-token \
go run ./cmd/cluster-enrollment-worker
```

### compose 方式

```bash
cd /Users/alex/ops/platform-center
docker compose up -d --build cluster-enrollment-worker machine-enrollment-worker
```

## 当前调用路径

### EKS

当前主路径：

- `aws-sdk` adapter：第一版真实实现，优先调用 `eks:DescribeCluster`
- `aws-cli` adapter：fallback，SDK 不可用时再降级

### ACK

当前主路径：

- `alicloud-openapi` adapter：占位骨架
- `alicloud-cli` adapter：fallback，当前可执行实现

## 证据链模板

真实实跑后，回报里必须给出：

1. `source job id`
2. `cloud resource id`
3. `cluster-enrollment claim`
4. `cluster-enrollment result`
5. 最终 `k8s cluster` record
6. UI 状态说明

建议记录格式：

```text
provider:
  aws-cli / aws-sdk / alicloud-cli / alicloud-openapi

runtime:
  worker location:
  credential source:

evidence:
  source job id:
  cloud resource id:
  claim:
  result:
  final cluster record:
```

## 取证脚本

真实环境完成 apply / enrollment 后，可直接运行：

```bash
cd /Users/alex/ops/platform-center
zsh scripts/collect_enrollment_evidence.sh <source_job_id> [cloud_resource_id]
```

当前脚本会从运行中的 `mysql` 容器读取：

- `deployment_jobs`
- `cloud_resources`
- `clusters`

如果运行时数据库还没应用最新 schema，脚本会明确指出缺失的列，例如：

- `cloud_resources.cluster_enrollment_status`
- `clusters.source_resource_id`

这代表当前 runtime 还不能给出 phase4 要求的完整证据链，需要先让 backend 用最新代码启动并完成自动迁移。

## 最新状态

截至 2026-04-26：

- compose runtime 已完成一次基于当前仓库代码的 backend/schema 对齐
- `collect_enrollment_evidence.sh` 不再报告 schema 缺列
- 但这仍不等于 `compose 已完成真实 provider 闭环`

当前边界仍然成立：

- schema / internal API：已对齐
- 真实 EKS / ACK provider 证据链：未完成
- P0 推荐运行位置：仍优先 `host process`

正式操作说明见 [AWS_ENROLLMENT_RUNBOOK.md](/Users/alex/ops/platform-center/docs/runbooks/AWS_ENROLLMENT_RUNBOOK.md:1)。
