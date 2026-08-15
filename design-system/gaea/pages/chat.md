# gaea 3.0 板块蓝图 · chat 聊天

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：`frontend/src/pages/ChatPage.tsx`（370 行）+ pages/chat/{ChatComposer,ChatModeBar,MessageList,WelcomeScreen}.tsx
- 侧栏：ChatTopicSidebar（会话列表，react-window 虚拟滚动）；消息流 MessageList
- 样式：chat-*.css + 全局 token；语音：useChatVoice / ChatAppendMessages 持久化
- 流式：SSE 事件（delta/reasoning/done/error）+ 30s 无帧超时 + sending 五路复位

## 目标态
- **视觉性格：沉浸对话**——左侧玻璃会话栏 + 消息流 + 流式发光。
- 会话侧栏（玻璃 `gp-panel`）：
  - 激活会话 = `--color-primary-container` 高亮 + 主色左缘 3px 指示条；
  - 新建按钮 = 主色胶囊；未完成徽标（interrupted）琥珀色 `--color-warning`；
  - 搜索框 focus `--focus-ring`。
- 消息流：
  - 用户气泡 = `--color-primary-container`（主色系，onPrimaryContainer 文字）；
  - AI 消息 = 透明底 + 头像（品牌 orb 小号）+ 主题色发光边（AI 生成态）；
  - 流式光标 `cursor-blink`（已有）；生成中 AI 消息底部发光脉冲 `--gaea-glow`；
  - 消息操作（复制/朗读）hover 浮现，`--transition-fast`。
- Composer：玻璃输入条（`md-glass-strong`），发送按钮主色，语音按钮次级；
  - 附加物 chips、资料引用标签走 `--color-primary-container`。

## 落地清单
- [ ] 侧栏激活态统一（primary-container + 左缘指示条）；焦点环
- [ ] 用户/AI 气泡语义色统一走令牌（清硬编码）
- [ ] Composer 玻璃输入条 + 发送 hover 微交互
- [ ] 流式发光脉冲（AI 生成态）用 `--gaea-glow`

## 验收
- 12 主题明暗下气泡对比度 ≥4.5:1；流式无跳动（无 layout shift）；
- 键盘可达（消息可聚焦、操作可见焦点环）；reduced-motion 无发光脉冲。
