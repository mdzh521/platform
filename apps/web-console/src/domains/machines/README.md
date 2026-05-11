# Machine Frontend Module

机器管理前端按职责拆成以下模块：

- `index.js`
  - 门面文件
  - 只负责事件绑定、数据加载、模块接线
- `workspace.js`
  - 资产列表、资产组、摘要卡、最近会话
- `detail.js`
  - 资产详情页渲染
  - 详情页动作绑定
- `terminal.js`
  - 在线终端
  - ticket 打开、WebSocket、输入输出、关闭
- `modals.js`
  - 资产、资产组、快速登录、托管账号弹窗
- `helpers.js`
  - 纯映射和渲染辅助

约束：

- 新的机器管理展示逻辑优先放到对应子模块，不要继续堆回 `index.js`
- 终端链路只放在 `terminal.js`
- 详情页交互只放在 `detail.js`
- 列表页筛选、统计、资产组视图只放在 `workspace.js`
