# gaea 3.0 板块蓝图 · gaea 通用办公

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：`frontend/src/gaea/App.tsx`（920 行，办公工作台壳）+ gaea/components/*（Composer/Sidebar/Transcript/
  ProcessCard/ToolGroup/FilePreview/MemoryPanel/CostLibraryView/TaskCenter 等，~40 组件）
- 布局：左侧会话栏（Sidebar，react-window 虚拟化 + 项目分组 + 置顶/归档）+ 中间消息流（Transcript/ProcessCard）
  + 右侧工具/交付物面板（ToolGroup/DeliverablesPanel）；底部 Composer（附件/资料/工具菜单）
- 后端：42 工具 agent；TurnResult 语义；看门狗；审批交互（EnableInteractiveApproval）

## 目标态
- **视觉性格：生产力主战场**——信息密度高但层级清晰；AI 过程可见可回溯。
- 左会话栏（玻璃 `gp-panel`）：
  - 项目分组头（小字 `--color-text-secondary`）；会话行 hover `--color-surface-container-high`；
  - 激活会话 = primary-container + 左缘指示条；置顶/归档/未完成徽标语义色；
  - 专注模式切换按钮。
- 中间消息流：
  - 用户请求 = 主色左缘条 + 文本；AI 过程卡（ProcessCard）= 实底卡
    （`--color-surface-container`），标题行 = 工具图标 + 名称 + 状态色（running=primary 脉冲 /
    done=success / error=destructive / blocked=warning）；
  - 思考卡默认折叠；工具卡/组折叠；大输出卡完成自动展开（现状保留）；
  - 运行中强制跟随底部（现状保留）；流式文本光标 `cursor-blink`。
- 右侧面板（工具/交付物/任务）：Tab 头 = antd tabs（ink-bar 主色）+ 玻璃容器；
  - 交付物卡 = 内容卡 + 缩略图 + 复制路径按钮（hover 浮现）。
- Composer：玻璃输入条；开工前计划卡片（AskCard）= primary-container 高亮卡；
  - 附件 chips / 资料引用 / 工具菜单全部走令牌色。

## 落地清单
- [ ] Sidebar 激活态/分组头/徽标统一令牌
- [ ] ProcessCard 状态色语义化（running/done/error/blocked 四态 + 图标，不只靠颜色）
- [ ] 右侧面板容器统一 `gp-panel` + antd tabs 主色
- [ ] Composer/AskCard 玻璃化 + 焦点环
- [ ] 清组件内硬编码色值

## 验收
- 12 主题下过程卡状态四态可区分（色+图标+文字）；键盘遍历过程卡可见焦点；
- reduced-motion：无脉冲动画；密度切换（standard/compact）间距正常。
