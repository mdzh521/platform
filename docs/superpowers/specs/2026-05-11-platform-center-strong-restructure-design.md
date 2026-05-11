# Platform Center 强重构设计

## 1. 背景

`platform-center` 当前尚未正式投入使用，也没有生产数据负担。现有数据主要为测试数据，这给了我们一次相对完整的结构性重构窗口。

当前项目的主要问题不是功能缺失，而是代码组织方式已经明显落后于项目复杂度：

- 仓库顶层是“多服务目录并列 + 脚本 + 文档”混合形态
- 旧控制面目录中的 `cloud` 模块承担过多职责
- 多个 worker 虽然都属于 claim/execute/report 模式，但结构尚未统一
- 前端已经开始模块化，但仍缺少稳定的代码分层
- 运维脚本、运行文档、历史材料混在项目主视图里

本次重构目标不是小修补，而是将仓库升级为一个结构稳定、可持续演进的 monorepo。

## 2. 目标

本次重构目标如下：

1. 将仓库重组为以产品域为中心的 monorepo
2. 将控制面后端重组为明确的领域边界结构
3. 将所有 worker 统一到同一种运行骨架
4. 将前端重组为清晰的 `boot/core/domains/shared/pages` 分层
5. 将脚本、compose、运行手册和归档文档迁移到更清晰的位置
6. 为后续 UI 重设计和文档清理创造稳定基础

## 3. 非目标

本次阶段明确不做以下事情：

- 不在本阶段优先重做 UI 视觉样式
- 不在本阶段优先删除历史文档和脚本，只做分类迁移与标记
- 不强行抽一切共享代码到公共库
- 不尝试把所有服务合并为单个可执行程序
- 不在本阶段引入额外前端框架替换当前技术路线

## 4. 目标顶层结构

目标仓库结构如下：

```text
platform-center/
  apps/
    control-plane/
    web-console/
    terraform-runner/
    resource-sync-worker/
    cluster-enrollment-worker/
    cluster-addon-worker/
    machine-enrollment-worker/

  packages/
    domain/
    contracts/
    runtime/
    provider-kit/

  ops/
    compose/
    scripts/
    bootstrap/
    diagnostics/

  docs/
    architecture/
    runbooks/
    cleanup/
    archive/
    superpowers/

  .local/ or tmp/
```

说明：

- `apps/` 只放真正可运行单元
- `packages/` 只放跨服务、语义稳定、值得复用的能力
- `ops/` 收纳 compose、脚本与运行辅助设施
- `docs/` 收纳正式文档、runbook 与归档材料
- 本地运行残留与缓存从主代码视图退出

## 5. 控制面结构设计

旧控制面根目录将统一收敛到 `apps/control-plane`，内部重组目标如下：

```text
apps/control-plane/
  cmd/server/

  internal/
    app/
    config/
    infra/
      db/
      cache/
      queue/
      security/
      web/

    domains/
      iam/
      platform/
      delivery/
      inventory/
      clusters/
      addons/
      machines/
      projects/
      graph/

    contracts/
      api/
      internal/
      events/

    adapters/
      persistence/
      http/
      queue/
```

关键变化：

- 用 `domains` 替代现在的 `modules`
- 将原 `cloud` 大包拆成：
  - `delivery`
  - `inventory`
  - `clusters`
  - `addons`
- 将 `machines` 提升为一等领域
- 将 HTTP、持久化与队列适配层从领域实现中逐步抽离

## 6. Worker 统一骨架

所有 worker 统一迁移至 `apps/<worker-name>`，目标结构如下：

```text
apps/<worker-name>/
  cmd/<worker-name>/

  internal/
    app/
    config/
    runtime/
    client/
      controlplane/
    domain/
      claim.go
      result.go
      task.go
    worker/
      runner.go
      executor.go
      handlers/
```

统一原则：

- 每个 worker 都显式体现 `poll -> claim -> execute -> report`
- `client/controlplane` 统一承接后端内部接口调用
- `domain` 保留 worker 本身的任务契约
- provider-specific 逻辑仅保留在 `handlers/`

这使五个 worker 从“结构相似但不统一”转为“一套模板的不同实例”。

## 7. 前端代码层结构设计

旧前端控制台根目录将统一收敛到 `apps/web-console`，代码层目标结构如下：

```text
apps/web-console/
  index.html
  assets/
    styles/
    vendor/

  src/
    boot/
    core/
      api/
      state/
      events/
      router/
      runtime/
      ui-shell/

    domains/
      auth/
      overview/
      delivery/
      clusters/
      machines/
      system/

    shared/
      components/
      forms/
      tables/
      drawers/
      dialogs/
      terminal/
      utils/

    pages/
      login/
      overview/
      delivery/
      clusters/
      machines/
      system/
```

关键变化：

- 历史前端脚本树统一收敛到 `src`
- `main.js` 负责的逻辑拆入 `boot/core/pages`
- 领域命名与后端统一：
  - `cloud` -> `delivery`
  - `k8s` -> `clusters`
- 强化共享 UI 层，避免模块私有实现持续复制

## 8. 迁移策略

采用“双轨迁移”而不是“一次性全量搬迁”。

### 阶段 1：建立新骨架

- 创建 `apps/ packages/ ops/ docs/` 目录
- 放入迁移说明文件
- 建立统一命名约定

### 阶段 2：迁移控制面

- 先整体收口旧控制面根目录到 `apps/control-plane`
- 保持功能可运行
- 再逐步拆 `domains`
- 以 `delivery -> inventory -> clusters -> addons -> machines` 为优先顺序调整内部结构

### 阶段 3：迁移 worker

- 先迁移 `resource-sync-worker`，作为标准模板
- 再迁移 `machine-enrollment-worker`
- 再迁移 `cluster-enrollment-worker`
- 再迁移 `cluster-addon-worker`
- 最后迁移 `terraform-runner`

理由：

- `resource-sync-worker` 结构最接近标准模板
- `terraform-runner` 特殊性最高，放在最后更稳

### 阶段 4：迁移前端

- 将旧前端控制台根目录迁移到 `apps/web-console`
- 保留现有页面行为
- 先做结构迁移，再做 UI 重设计

### 阶段 5：迁移 ops 和 docs

- `docker-compose.yml` 等迁入 `ops/compose`
- 现有脚本迁入 `ops/scripts` 或 `ops/diagnostics`
- 正式文档迁入 `docs/runbooks` / `docs/architecture`
- 历史遗留文档迁入 `docs/archive`

### 阶段 6：删除旧结构

- 所有引用修正完成后
- 删除旧顶层目录
- 删除过渡性兼容路径

## 9. 验收标准

本次强重构完成后，应满足以下标准：

### 结构层

- 仓库顶层只保留新的 `apps/packages/ops/docs`
- 原顶层 legacy 运行目录不再作为主路径
- 所有主要运行单元都有明确 README 或结构说明

### 控制面层

- `control-plane` 内不再存在职责失控的 `cloud` 超大包
- API、内部契约、领域逻辑、持久化层的边界变清晰

### Worker 层

- 五个 worker 具备统一骨架
- claim/result/client/runtime 组织方式一致

### 前端层

- 入口逻辑从单个 `main.js` 迁出
- 领域模块与页面装配层分离
- 公共 UI 和业务域的边界清晰

### 运行层

- compose 启动路径可重新定位
- 主要运行文档和脚本仍可支撑本地调试
- 关键业务链路仍能被 smoke 验证

## 10. 风险与控制

### 风险 1：路径迁移造成启动失败

控制：

- 每个迁移阶段后立即修 compose / config / Dockerfile
- 保持旧路径到新路径的短期兼容映射，直到下一阶段收敛

### 风险 2：后端内部引用大量断裂

控制：

- 先整体迁移目录，再拆内部领域
- 每个领域拆分后做最小编译检查

### 风险 3：前端结构迁移影响页面行为

控制：

- 前端先只迁结构，不改视觉
- 页面级事件和 API 调用顺序先保持一致

### 风险 4：文档和脚本在强重构中被误删

控制：

- 先迁分类，再决定是否删除
- 删除动作放到结构稳定之后执行

## 11. 推荐实施顺序

推荐按以下顺序执行：

1. 建立新顶层骨架
2. 迁移 `control-plane`
3. 统一 worker 骨架
4. 迁移 `web-console`
5. 迁移 `ops/docs`
6. 删除旧目录与过渡路径

这个顺序的原因是：

- 控制面是最重的结构中心
- worker 依赖控制面契约
- 前端依赖后端稳定命名与业务域
- 脚本与文档清理应放在主代码路径稳定之后

## 12. 下一步

当这份设计确认后，下一步不是直接“边想边改”，而是进入明确实施计划：

- 第一轮只做目录重构和路径迁移
- 第二轮做控制面领域拆分
- 第三轮做 worker 统一化
- 第四轮做前端结构迁移
- 第五轮再进入 UI 重设计与文档脚本清理

这能确保强重构不是一次不可控的全仓震荡，而是一系列可验证的结构收敛。
