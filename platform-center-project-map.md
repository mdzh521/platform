# Platform Center 项目地图

这份文档不是按目录解释 `platform-center`，而是按“它到底在解决什么问题、业务对象怎么流转、每个模块为什么存在”来梳理。

适合在这几种场景下阅读：

- 刚接手这个项目，想快速建立全局认知
- 已经能跑起来，但看不清 backend 和 worker 的分工
- 要给团队新人讲解这个平台，而不是只讲代码目录

## 一、项目定位

`platform-center` 不是单体应用，而是一个多服务控制面。

它试图把几件事情串成一条完整链路：

1. 云资源交付
2. 交付结果回填
3. 平台对象纳管
4. 后续运维操作

从业务上看，它既不是单纯的 Terraform 平台，也不是单纯的 K8s 面板，更不是单纯的堡垒机。  
更准确地说，它是一个把“交付 -> 纳管 -> 运维”串起来的平台控制面。

## 二、统一业务模型

先看最重要的一张抽象图：

```text
用户操作前端
    |
    v
backend API
    |
    v
deployment_jobs
    |
    v
terraform-runner 执行
    |
    v
resource-sync-worker 回填
    |
    v
cloud_resources
   / \
  /   \
 v     v
cluster enrollment   machine enrollment
  |                    |
  v                    v
k8s.Cluster         machine.Asset
  |                    |
  v                    v
addon execution      account / credential / terminal
  |
  v
K8s 运维操作
```

这张图表达的是整个项目最核心的分层。

### 1. 任务层：`deployment_jobs`

这一层表达的是“平台想做什么”。

它负责记录：

- 用哪个云账号
- 用哪个蓝图
- 对哪个项目 / 环境 / 网络生效
- 当前动作是 `plan` / `apply` / `destroy`
- 当前任务执行到什么状态

这一层是任务抽象，不是资源抽象。

### 2. 执行层：`terraform-runner`

这一层负责把任务真正执行掉。

它不关心前端页面，也不关心 K8s 或机器资产，只关心：

- 认领 job
- 执行 Terraform
- 上报心跳、日志和结果

### 3. 资源中间层：`cloud_resources`

这一层是整个项目的关键桥梁。

Terraform 成功不代表平台已经理解资源。  
所以需要一层统一的 provider 资源视图，把不同云厂商、不同蓝图产生的对象先收口为平台可识别的资源记录。

这层通常会标出：

- provider
- account
- resource type
- resource role
- lifecycle state
- source job
- enrollment status

可以把它理解成：

“平台对云世界资源的统一中间态建模。”

### 4. 平台对象层：`k8s.Cluster` / `machine.Asset`

真正让前端和后续运维模块围绕其工作的，不是 `cloud_resources`，而是平台对象。

`cloud_resources` 会继续分叉成两个主要终点：

- 集群资源 -> `k8s.Cluster`
- 计算资源 -> `machine.Asset`

这两个对象才是平台真正运营的核心对象。

### 5. 运维能力层

平台对象生成后，才会进一步挂载操作能力。

集群对象往下挂：

- addon execution
- namespace / workload / pod
- logs / exec / shell
- scale / restart / image update

机器对象往下挂：

- group
- credential
- account
- terminal ticket
- session / event
- SFTP

## 三、两条最重要的业务链

### 1. 云集群从创建到出现在 K8s 页面

这条链路可以压缩成：

```text
创建 deployment job
-> terraform-runner 执行
-> resource-sync-worker 生成 cloud_resources
-> 识别 cluster 资源
-> cluster-enrollment-worker 纳管
-> backend 写入 k8s.Cluster
-> cluster-addon-worker 补 addon
-> 前端 K8s 页面可见
```

关键判断点不是 Terraform apply 成功，而是：

- 是否生成了 cluster 类型的 `cloud_resources`
- `cluster_enrollment_status` 是否成功
- `k8s.Cluster` 是否真正写入

只有到 `k8s.Cluster` 写出来，集群工作台才真正“有东西可管”。

### 2. 机器资源从云回填到前端可连接

这条链路可以压缩成：

```text
创建 deployment job
-> terraform-runner 执行
-> resource-sync-worker 生成 compute 类型 cloud_resources
-> machine-enrollment-worker 纳管
-> backend 写入 machine.Asset
-> 前端机器工作台可见
-> 用户补齐 account / credential
-> 签发 terminal ticket
-> 远程连接
```

这里要特别注意：

- 机器出现在资产列表里，不等于已经可远程连接
- `machine enrollment` 解决的是“纳入资产管理”
- `credential / account / terminal` 解决的是“真正连接”

## 四、模块职责地图

这个项目最适合按 7 个模块来理解。

### 1. `web-console`

职责：

- 提供统一控制台界面
- 组织系统治理、交付网络、集群工作台、机器工作台四大工作台
- 调用后端 API 并展示结果

一句话理解：

“它是平台的交互入口，不是业务大脑。”

### 2. `control-plane`

职责：

- 统一 API 面
- 鉴权和系统治理
- 维护平台主状态
- 编排 job / resource / enrollment / addon 流转
- 暴露内部接口给各 worker

一句话理解：

“它是整个平台的统一控制面和状态中心。”

### 3. `terraform-runner`

职责：

- 认领 deployment job
- 建 workspace
- 跑 Terraform
- 回传日志、心跳和结果

一句话理解：

“它是基础设施交付执行器。”

### 4. `resource-sync-worker`

职责：

- 将 Terraform 交付结果翻译成 `cloud_resources`
- 给资源打类型、角色、生命周期和后续 enrollment 状态

一句话理解：

“它是云资源标准化翻译器。”

### 5. `cluster-enrollment-worker`

职责：

- 认领 cluster 类型资源
- 解析 provider 信息
- 生成或更新 `k8s.Cluster`

一句话理解：

“它把云里的集群变成平台里的集群对象。”

### 6. `cluster-addon-worker`

职责：

- 认领 cluster addon execution
- 安装或修复平台约定的集群 addon
- 上报 heartbeat 和结果

一句话理解：

“它是集群标准化补全器。”

### 7. `machine-enrollment-worker`

职责：

- 认领 compute 类型资源
- 将资源翻译成 `machine.Asset`
- 推进机器资产纳管状态

一句话理解：

“它把云里的机器变成平台里的资产对象。”

## 五、为什么 `cloud_resources` 这一层这么重要

如果只看功能，很容易误以为：

- Terraform 成功 -> K8s 集群可见
- Terraform 成功 -> 机器资产可见

但这个项目不是这么设计的。

它中间故意多加了一层 `cloud_resources`，目的是把：

- provider 资源模型
- 平台领域模型

清晰地隔开。

因此它的真实链路是：

```text
Terraform 成功
-> 平台先“看见资源”
-> 再“理解资源”
-> 最后“运营资源”
```

这三步分别对应：

- `terraform-runner`
- `resource-sync-worker`
- `cluster/machine enrollment + 后续模块`

## 六、排障时应该怎么想

这个项目排障最怕“从页面现象直接跳到某个 worker 猜问题”。  
更稳的方式是按分层查。

### 集群问题

如果集群没有出现在 K8s 页面，按这个顺序查：

1. `deployment_jobs` 是否成功
2. `cloud_resources` 是否已生成 cluster 资源
3. `cluster_enrollment_status` 当前是什么
4. `k8s.Cluster` 是否已写入
5. `ClusterAddonExecution` 是否失败或卡住

### 机器问题

如果机器没有出现在机器工作台，按这个顺序查：

1. `deployment_jobs` 是否成功
2. `cloud_resources` 是否已生成 compute 资源
3. `machine_enrollment_status` 当前是什么
4. `machine.Asset` 是否已写入
5. 账号 / 凭据 / 默认账号是否缺失，导致“可见但不可连”

## 七、建议的阅读顺序

如果你要继续往代码里深入，我建议按下面顺序看。

### 第一步：看整体启动和编排

- [STARTUP.md](/Users/alex/ops/platform-center/docs/runbooks/STARTUP.md:1)
- [docker-compose.yml](/Users/alex/ops/platform-center/ops/compose/docker-compose.yml:1)

### 第二步：看统一 API 面

- [router.go](/Users/alex/ops/platform-center/apps/control-plane/internal/infra/web/router.go:1)

### 第三步：看后端启动装配

- [app.go](/Users/alex/ops/platform-center/apps/control-plane/internal/app/app.go:1)

### 第四步：看三条最关键的 cloud 流转代码

- [deployment_service.go](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/delivery/cloud/deployment_service.go:1)
- [resource_service.go](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/delivery/cloud/resource_service.go:1)
- [addon_service.go](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/delivery/cloud/addon_service.go:1)

### 第五步：再分别看 K8s 和机器域

- [service.go](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/clusters/k8s/service.go:1)
- [service.go](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/machines/machine/service.go:1)

## 八、一句话总结

`platform-center` 的本质，不是“会跑 Terraform 的后台”，而是一个把：

- 云资源交付
- 资源标准化
- 平台对象纳管
- K8s / 机器运维能力

串成一条控制链的多服务平台。

如果只记一句话，可以记这个：

```text
deployment_jobs 决定做什么，
cloud_resources 统一看见了什么，
k8s.Cluster / machine.Asset 决定平台最终管理什么。
```
