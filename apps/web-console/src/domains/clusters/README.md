# K8s Frontend Module Map

本目录承接 `K8s 管理中心` 的前端能力。后续新增“机器管理”等业务时，不应继续把逻辑堆回这里，而应新建平行模块。

## 入口职责

- `../k8s.js`
  - 页面级编排入口
  - 数据加载、模块接线、全局 K8s 事件绑定
  - 不负责具体页面细节渲染

## 子模块职责

- `business-actions.js`
  - 集群/命名空间/工作负载/资源页的业务动作
  - 刷新、同步、切换、删除、创建入口

- `page-renderers.js`
  - 列表页和摘要页渲染
  - 集群、命名空间、工作负载、资源浏览
  - 分页、筛选、空状态

- `selection-actions.js`
  - 从列表进入对象
  - 选中工作负载/资源
  - 刷新当前选中对象

- `workload-detail.js`
  - 服务详情页渲染
  - Pod、发布状态、事件、日志、终端、YAML

- `workload-ops.js`
  - 工作负载运维动作
  - 扩缩容、镜像更新、exec、terminal 等

- `resource-detail-content.js`
  - 资源详情内容块

- `resource-edit.js`
  - 资源编辑弹窗和提交逻辑

- `detail-panels.js`
  - 详情壳层与资源详情页编排

- `cluster-overview.js`
  - 集群概览页

- `workload-creator.js`
  - K8s 资源/工作负载创建器主逻辑

- `workload-creator-state.js`
  - 创建器默认值、builder state 初始化

- `workload-creator-serialize.js`
  - 创建器表单到 manifest 的序列化

## 后续扩展原则

1. K8s 页的新能力优先拆到现有子模块，不要继续加大 `../k8s.js`
2. 新业务域单独建目录，例如：
   - `src/domains/machines/`
   - `assets/app/cmdb/`
3. 详情页和列表页分离，避免一页里同时塞“列表 + 大量详情 + 大量操作”
