# gaea 3.0 板块蓝图 · gaea 通用办公

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。
> v4.53（2026-09-03，用户四点拍板化繁为简）：欢迎界面降噪——删对话头部上下文
> 横条（主区「上下文」标签页保留）、删侧栏底部模型卡（顶栏模型入口统一）、
> 右栏「任务+分工」「产物+变更」**直接合并为两个面板**（MergedPanel 上下分区
> 同屏全可见，非二级标签；一级 Tab 6→4：文件/产物/任务/浏览器；旧持久化 id
> changes→deliverables、subagents→tasks 别名收敛，子代理自动展开改切任务面板）。

## 现状（facts）
- 页面：`frontend/src/gaea/App.tsx`（办公工作台壳）+ gaea/components/*（Composer/Sidebar/Transcript/
  ProcessCard/ToolGroup/FilePreview/MemoryPanel/CostLibraryView/TaskCenter 等，~40 组件）
- 布局：左侧会话栏（Sidebar，react-window 虚拟化 + 项目分组 + 置顶/归档）+ 中间消息流（Transcript/ProcessCard）
  + 右侧工作区面板；底部 Composer（附件/资料/工具菜单）
- 右栏清单：`lib/workspaceTabs.ts` 单一数据源（id/label/icon/keywords/启用集/宽度）；
  渲染接线 `lib/sidebarRegistry.ts`；合并壳 `components/MergedPanel.tsx`
  （主区 flex-1 自滚动 + 次区 border-t 分隔、高度随内容封顶 45%、内部滚动）
- 后端：42 工具 agent；TurnResult 语义；看门狗；审批交互（EnableInteractiveApproval）

## 目标态
- **视觉性格：生产力主战场**——信息密度高但层级清晰；AI 过程可见可回溯。
- 左会话栏（玻璃 `gp-panel`）：项目分组头/激活会话左缘指示条/置顶·归档·未完成
  徽标语义色/专注模式；**底部无模型卡**（v4.53 起 FeatureModelBar 移除，
  模型入口 = chat 头部切换器 + 壳层顶栏 pill）。
- 中间消息流：**头部无上下文横条**（v4.53 起 ContextBar 移除，上下文走主区标签页）；
  用户请求 = 主色左缘条；ProcessCard 四态语义色+图标（running/done/error/blocked）；
  思考/工具卡折叠；大输出卡完成自动展开；运行中强制跟随底部；流式光标。
- 右侧面板（4 Tab，声明式设置卡/命令面板随清单派生）：
  - 「产物」= 会话产物(上) + 文件变更·证据链(下)；
  - 「任务」= 任务中心(上) + 分工(下，含自动展开偏好；运行角标挂任务单键)；
  - 「文件」「浏览器」单面板不变；窄栏图标化（WORKSPACE_TAB_COMPACT_WIDTH=420）。
- Composer：玻璃输入条；开工前计划卡片（AskCard）= primary-container 高亮卡；
  附件 chips / 资料引用 / 工具菜单全部走令牌色。

## 落地清单
- [x] v4.53 四点简化（ContextBar/模型卡删除 + 两处面板直接合并 + 旧 id 别名迁移）
- [x] 右栏清单/注册表/设置卡/命令面板单一数据源链路（v4.23 注册制延续）
- [ ] ProcessCard 状态色语义化 12 主题全量抽查
- [ ] Composer/AskCard 玻璃化焦点环全量核查
- [ ] 清组件内硬编码色值（存量核查）

## 验收
- judge 视觉验收 3/3 pass（无 must_fix）：四点诉求落实、上下分区空态/开关完整、
  无截断叠压；vitest 192 文件/1337 用例全绿（workspaceTabs/WorkspaceTabs/MergedPanel
  锁同步更新）。
- 非阻塞欠账：任务中心空态副文案「价格抓取、语义索引」与办公语境不符（占位残留）；
  产物/变更次区分隔线偏淡可加深（设计偏好）。
