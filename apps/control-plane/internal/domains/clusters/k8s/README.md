# K8s Backend Module Map

本目录承接 K8s 管理中心的后端能力。后续新增“机器管理”等功能时，应新建平行业务模块，避免继续向本目录塞入无关逻辑。

## 目录职责

- `model.go`
  - 平台侧 K8s 记录模型

- `handler.go`
  - 常规 HTTP 入口

- `handler_terminal.go`
  - WebSocket / terminal 相关入口

- `service.go`
  - Service 构造与共享依赖

- `service_cluster.go`
  - 集群、命名空间、概览

- `service_workload.go`
  - 工作负载、发布、日志、事件、详情

- `service_resource.go`
  - Service / Ingress / ConfigMap / Secret / PVC / Quota / LimitRange 等资源

- `service_exec.go`
  - 非交互式命令执行

- `service_terminal.go`
  - 交互式终端

- `service_runtime.go`
  - cache / queue / runtime health 接入

- `service_builder.go`
  - manifest 构建与创建器相关 helper

- `service_helpers.go`
  - service 通用辅助函数

- `service_types.go`
  - service DTO / 输入输出结构

- `client_cluster.go`
  - cluster / namespace / node / overview 取数

- `client_workload.go`
  - workload / rollout / logs / events / exec 相关取数

- `client_resource.go`
  - resource detail / manifest / resource relationships

- `client_http.go`
  - kube API 请求基础能力

- `client_helpers.go`
  - client 公共 helper

- `client_types.go`
  - kube API payload 结构

- `client.go`
  - package anchor

## 后续扩展原则

1. 新增 K8s 能力时，优先落到已有 `service_*` / `client_*` 分文件
2. 不要把新业务写进本目录；机器管理等能力应该新建模块
3. terminal / exec / workload / resource / cluster 这几条线保持独立
