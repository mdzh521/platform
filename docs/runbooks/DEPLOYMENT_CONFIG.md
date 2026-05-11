# Platform Center Deployment Config

这份文档只回答一件事：`platform-center` 现在有哪些配置文件，分别该配什么。

## 配置入口

当前一共 5 份核心配置文件：

1. 后端主配置
   [platform-center.yaml](/Users/alex/ops/platform-center/apps/control-plane/config/platform-center.yaml)
2. Terraform Runner 配置
   [terraform-runner.yaml](/Users/alex/ops/platform-center/apps/terraform-runner/config/terraform-runner.yaml)
3. Resource Sync Worker 配置
   [resource-sync-worker.yaml](/Users/alex/ops/platform-center/apps/resource-sync-worker/config/resource-sync-worker.yaml)
4. Machine Enrollment Worker 配置
   [machine-enrollment-worker.yaml](/Users/alex/ops/platform-center/apps/machine-enrollment-worker/config/machine-enrollment-worker.yaml)
5. Cluster Enrollment Worker 配置
   [cluster-enrollment-worker.yaml](/Users/alex/ops/platform-center/apps/cluster-enrollment-worker/config/cluster-enrollment-worker.yaml)

加载顺序统一是：

1. 程序内置默认值
2. YAML 配置文件
3. 环境变量覆盖

环境变量优先级最高。

## 1. 后端主配置

文件：

`apps/control-plane/config/platform-center.yaml`

主要负责：

- 应用名、端口、运行模式
- JWT 密钥
- 内部 runner token
- 默认管理员账号
- MySQL
- Redis cache / queue
- 前端模式
- 云账号 bootstrap

最关键字段：

- `jwt_secret`
- `internal_runner_token`
- `default_admin_username`
- `default_admin_password`
- `mysql_*`
- `cache.*`
- `queue.*`
- `cloud_accounts`

### 云账号凭据怎么配

现在云账号可以直接写进这个文件：

```yaml
cloud_accounts:
  - name: aws-demo
    provider: aws
    access_key: AKIA...
    secret_key: your-secret-key
    region: ap-southeast-1
    role_arn: arn:aws:iam::123456789012:role/platform-center
    default_tags:
      - env=dev
      - owner=platform
    default_zones:
      - ap-southeast-1a
      - ap-southeast-1b
```

行为说明：

- 后端启动时会按 `name` 做 upsert
- `access_key / secret_key` 会先加密再写数据库
- 如果账号已存在而你没改密钥，旧的加密值会保留

## 2. Terraform Runner 配置

文件：

`apps/terraform-runner/config/terraform-runner.yaml`

主要负责：

- runner 名称
- claim 轮询频率
- heartbeat 频率
- backend 内部 API 地址和 token
- workspace 路径
- provider cache 路径
- template 根目录
- 最大并发
- workspace 清理 TTL

最关键字段：

- `backend_base_url`
- `backend_token`
- `poll_interval`
- `heartbeat_interval`
- `max_parallel_jobs`
- `workspace_*_ttl`

当前默认值适合本地 compose：

```yaml
backend_base_url: http://backend:8080
backend_token: runner-dev-token
```

如果你后面单独部署 runner，就改这里，不用再去改代码。

## 3. Resource Sync Worker 配置

文件：

`apps/resource-sync-worker/config/resource-sync-worker.yaml`

主要负责：

- worker 名称
- 轮询频率
- backend 内部 API 地址和 token

最关键字段：

- `backend_base_url`
- `backend_token`
- `poll_interval`

## 4. Machine Enrollment Worker 配置

文件：

`apps/machine-enrollment-worker/config/machine-enrollment-worker.yaml`

主要负责：

- worker 名称
- 轮询频率
- backend 内部 API 地址和 token

最关键字段：

- `backend_base_url`
- `backend_token`
- `poll_interval`

## 5. Cluster Enrollment Worker 配置

文件：

`apps/cluster-enrollment-worker/config/cluster-enrollment-worker.yaml`

主要负责：

- worker 名称
- 轮询频率
- backend 内部 API 地址和 token

最关键字段：

- `backend_base_url`
- `backend_token`
- `poll_interval`

注意：

- 这两份 enrollment worker 配置只负责 worker 自身运行参数
- 真实 provider 凭据仍然来自宿主环境变量或宿主凭据文件
- phase4 实跑说明见 [PHASE4_ENROLLMENT_RUNTIME.md](/Users/alex/ops/platform-center/docs/runbooks/PHASE4_ENROLLMENT_RUNTIME.md)

## 生产环境最少要改的字段

至少要改这些：

- `apps/control-plane/config/platform-center.yaml`
- `apps/control-plane/config/platform-center.yaml`
  - `jwt_secret`
  - `internal_runner_token`
  - `default_admin_password`
  - `mysql_password`
  - `cache.redis_url`
  - `queue.redis_url`
- `apps/terraform-runner/config/terraform-runner.yaml`
  - `backend_token`
  - `backend_base_url`
- `apps/resource-sync-worker/config/resource-sync-worker.yaml`
  - `backend_token`
  - `backend_base_url`
- `apps/machine-enrollment-worker/config/machine-enrollment-worker.yaml`
  - `backend_token`
  - `backend_base_url`
- `apps/cluster-enrollment-worker/config/cluster-enrollment-worker.yaml`
  - `backend_token`
  - `backend_base_url`

## 和 docker-compose 的关系

当前 compose 入口 [docker-compose.yml](/Users/alex/ops/platform-center/ops/compose/docker-compose.yml) 已经包含：

- `backend`
- `frontend`
- `terraform-runner`
- `resource-sync-worker`
- `machine-enrollment-worker`
- `cluster-enrollment-worker`

目前还保留的少量环境变量覆盖主要是：

- `backend` 的 `FRONTEND_MODE=external`
- `backend` 的 `INTERNAL_RUNNER_TOKEN=runner-dev-token`
- `terraform-runner` 的 `RUNNER_HEARTBEAT_INTERVAL=2s`

其中：

- `FRONTEND_MODE=external` 是为了当前前后端分容器部署
- `RUNNER_HEARTBEAT_INTERVAL=2s` 是当前测试/验证口径，生产上你可以删掉，让它回到 YAML 默认值

## 推荐做法

本地或单机演示：

- 直接改这 5 份 YAML
- 然后执行：

```bash
cd /Users/alex/ops/platform-center
docker compose up -d --build
```

生产环境：

- 非敏感默认值放 YAML
- 敏感值优先走环境变量或密钥管理系统覆盖
- 保持 `internal_runner_token` 在三边一致：
  - backend
  - terraform-runner
  - resource-sync-worker
  - machine-enrollment-worker
  - cluster-enrollment-worker
