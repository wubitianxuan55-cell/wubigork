# gaea 3.0 板块蓝图 · chat 聊天

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。
> 更新：T6-10.2（2026-08-15）——模式切换条移入全局顶栏；输入框两级重设计。

## 现状（facts）
- 页面：`frontend/src/pages/ChatPage.tsx` + components/chat/{ChatComposer,ChatModeBar,ChatPersonaBar,MessageList,ChatInspector,WelcomScreen}.tsx
- 侧栏：ChatTopicSidebar（会话列表，react-window 虚拟滚动）；消息流 MessageList
- 样式：chat-board.css + 全局 token；语音：useChatVoice / ChatAppendMessages 持久化
- 流式：SSE 事件（delta/reasoning/done/error）+ 30s 无帧超时 + sending 五路复位

## 目标态
- **视觉性格：沉浸对话**——左侧玻璃会话栏 + 消息流 + 流式发光。
- **模式切换条在全局顶栏**（T6-10.2）：普通对话/角色对话 胶囊段渲染进 MainLayout 的
  v3-strip 轨道条（`#v3-chatmode-host` 宿主 + ChatPage createPortal + ChatModeBar `variant='strip'`；
  仅聊天板块激活时显示，动作按钮如角色库/切换角色/语音/导出/清空同条内嵌）
- 会话侧栏（玻璃 `gp-panel`）：
  - 激活会话 = `--color-primary-container` 高亮 + 主色左缘 3px 指示条；
  - 新建按钮 = 主色胶囊；未完成徽标（interrupted）琥珀色 `--color-warning`；
  - 搜索框 focus `--focus-ring`。
- 消息流：
  - 用户气泡 = `--color-primary-container`（主色系，onPrimaryContainer 文字）；
  - AI 消息 = 透明底 + 头像（品牌 orb 小号）+ 主题色发光边（AI 生成态）；
  - 流式光标 `cursor-blink`（已有）；生成中 AI 消息底部发光脉冲 `--gaea-glow`；
  - 消息操作（复制/朗读）hover 浮现，`--transition-fast`。
- Composer **两级布局**（T6-10.2）：
  - 工具行（`.chat-composer-tools`）：搜索/深度思考/语音 开关（图标 + 12px 标签，激活 =
    主色胶囊 + 柔光，状态三重传达：色/图标/文案），右侧键盘提示「Enter 发送 · Shift+Enter 换行」；
  - 输入卡（`.chat-composer` 玻璃）：仅 textarea + 发送按钮（主色，有字发光），placeholder「输入消息…」；
  - 角色模式：快捷回复 chips 位于工具行上方。

## 落地清单
- [x] 模式切换条移入全局顶栏轨道条（portal 宿主，仅聊天板块显示）
- [x] Composer 两级重设计（工具行独立 + 输入卡精简）
- [x] 侧栏激活态统一（primary-container + 左缘指示条）；焦点环
- [x] 用户/AI 气泡语义色统一走令牌（清硬编码）
- [x] Composer 玻璃输入条 + 发送 hover 微交互
- [x] 流式发光脉冲（AI 生成态）用 `--gaea-glow`

## 验收
- 12 主题明暗下气泡对比度 ≥4.5:1；流式无跳动（无 layout shift）；
- 键盘可达（消息可聚焦、操作可见焦点环）；reduced-motion 无发光脉冲。
- 顶栏模式条：切板块隐藏、回聊天恢复；工具行激活态随 aria-pressed 同步。
