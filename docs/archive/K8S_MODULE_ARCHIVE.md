# K8s 模块归档说明

日期：2026-04-20

## 归档目的

当前 `集群工作台` 已经进入稳定阶段。为了避免后续“机器工作台”等新需求继续侵入 K8s 代码，本次将 K8s 相关代码做边界归档。

这不是冻结功能，而是明确：

- 哪些目录属于 K8s
- 哪些文件负责什么
- 后续新需求不应该改到哪里

## 前端边界

K8s 前端代码集中在：

- `apps/web-console/src/domains/clusters/`
- 页面编排入口：`apps/web-console/src/pages/clusters/index.js`

职责划分见：

- [README.md](/Users/alex/ops/platform-center/apps/web-console/src/domains/clusters/README.md)

后续新增模块建议：

- `apps/web-console/src/domains/machines/`
- `apps/web-console/src/domains/system/`
- `apps/web-console/src/shared/`

不要把新业务逻辑再塞进：

- `apps/web-console/src/domains/clusters/index.js`
- `apps/web-console/src/domains/clusters/workload-detail.js`

除非它确实属于 K8s。

## 后端边界

K8s 后端代码集中在：

- `apps/control-plane/internal/domains/clusters/k8s/`

职责划分见：

- [README.md](/Users/alex/ops/platform-center/apps/control-plane/internal/domains/clusters/k8s/README.md)

后续新增业务模块建议：

- `apps/control-plane/internal/domains/machines/`
- `apps/control-plane/internal/domains/nodeops/`
- `apps/control-plane/internal/domains/inventory/`

不要把“机器工作台”直接写进：

- `service_cluster.go`
- `service_workload.go`
- `service_resource.go`

除非该逻辑确实是 K8s 集群对象管理的一部分。

## 当前 K8s 稳定能力

前端：

- 集群列表 / 集群概览
- 命名空间
- 工作负载列表
- 服务详情独立页
- 资源浏览
- YAML / 事件 / 日志 / 终端 / exec
- 图形化资源与工作负载创建

后端：

- 集群接入
- namespace / workload / resource 同步
- rollout / events / logs / manifest
- exec / terminal
- cache / queue 基础接入

## 后续开发约束

1. 新业务域单独建模块目录
2. K8s 只做 K8s 对象和 K8s 运维
3. 复用公共基础设施时，走：
   - `apps/control-plane/internal/infra/*`
   - `apps/web-console/src/shared/*` 的公共层
4. 非 K8s 的页面不要复用集群详情页结构，避免耦合继续扩大
