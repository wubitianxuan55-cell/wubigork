# 市场调研：AI 聊天/陪伴产品 UI（2026-08）

> 目的：为 gaea 聊天×轻语合并版 UI 提供设计依据。来源为 2026 年公开资料，均为桌面/移动 AI 对话产品的一手界面模式。

## 1. 行业共识：1:1 AI 对话不再是"多人聊天"布局

- Claude 风格流布局成为主流：内容居中 + 最大宽度约束，用户消息为右侧轻量胶囊，助手消息为通栏流畅文本，弱化头像与气泡边框。根因是"左右气泡 + 头像"是多人聊天模式，不适合 1:1 AI 助手（[clankie redesign issue #110](https://github.com/thiagovarela/clankie/issues/110)）。
- ChatGPT/Claude/Gemini 界面高度趋同（极简画布 + 输入框 + 闪烁光标），差异化价值来自排版、动效与人格化，而非基础结构（[Built In: AI minimalist design](https://builtin.com/artificial-intelligence/ai-minimalist-design)）。
- 2026 UX 基准：Claude NPS 领先 ChatGPT 21 分（28% vs 7%），Gemini 12%、Grok 17% 居中（[MeasuringU 2026](https://measuringu.com/ai-based-chat-software-ux-2026/)）——界面质感与信任直接影响留存。
- Gemini 推出 "Neural Expressive" 设计语言：更流动的布局、活力配色、更新排版、触觉反馈（[Phandroid](https://phandroid.com/2026/05/19/google-brings-more-refined-aesthetics-to-the-gemini-app/amp/)）。

## 2. 陪伴类产品：人格在场感 + 记忆/情绪

- Livia（UX Design Awards 2025）：陪伴体验 = 记忆驱动交互 + 可定制人格 + 情绪追踪 + 主动互动（[Livia](https://ux-design-awards.com/winners/2025-1-livia)）。
- 研究结论：记忆功能显著增强长期陪伴感与情感连接（[Tampere 研究](https://trepo.tuni.fi/bitstream/handle/10024/231892/ChanSamU.pdf)）。
- 可落地的陪伴层：人格头像/名字常驻、情绪状态可见、信任/亲密度有进展反馈、记忆可回看。

## 3. 必须避开的暗模式（CDT 37 项）

CDT 对 ChatGPT/Gemini/Claude/Replika/Character.AI 审计出 37 种对话式暗模式（[CDT taxonomy](https://cdt.org/insights/dark-patterns-in-ai-chatbots-a-taxonomy-to-inform-better-design/)）。gaea 设计红线：

- 诚实标注 AI 身份，不伪造人类在场（无假"在线"、无虚假情感承诺）。
- 情绪/关系数值是透明元数据，不做操纵性诱导（如"再买会员才能继续感情"）。
- 退出/删除路径清晰（删除话题、清除会话一键可达）。

## 4. 视觉语言：Liquid Glass 与玻璃质感

- iOS 26 Liquid Glass 进入消息应用（WhatsApp iOS 版）：透明度 + 背景效果随明暗模式调整（[Digital Trends](https://www.digitaltrends.com/social-media/whatsapp-is-getting-ios-26s-liquid-glass-glow-up-and-its-surprisingly-gorgeous/)）。
- 玻璃拟态在 AI 聊天模板中成为主流（"cosmic liquid glass" + 人格系统，如 [Arcanea chat template](https://github.com/frankxai/arcanea-chat-template)）。
- 落地：保留 gaea 既有 M3 令牌 + aurora/glass，精修为"分层玻璃"（外层壳 + 内层芯，双 bezel 高光），明暗一致。

## 5. 会话管理

- 侧栏会话应显示可读标题 + 首条预览，避免逐个打开寻找（[openclaw sessions sidebar](https://github.com/openclaw/openclaw/issues/28168)）。
- 话题需要分类/归档能力（LobeHub 请求），本轮不实现，侧栏结构预留。
- 用户消息要有明确视觉区分（垂直标记或底色差异）（[ChatGPT 社区讨论](https://community.openai.com/t/improve-navigation-and-context-management-in-chat-interface/1378039)）。

## 6. Chat UI 基础规范（UXPin）

- 输入框多行可编辑；Enter 发送 / Shift+Enter 换行；发送按钮常显。
- 错误就近展示：失败标记在具体消息气泡上 + 说明 + 重试，而非全局 toast（[UXPin Chat UI guide](https://www.uxpin.com/studio/blog/chat-user-interface-design/)）。
- AI 对话要流式渲染、可停止；富文本（代码/表格/列表）在气泡内干净渲染；提供结构化快捷建议。
- 无障碍：ARIA live region 播报新消息、键盘全流程可操作、对比度 ≥4.5:1、触摸目标 ≥44px。

## 7. 对 gaea 聊天合并版的设计决策

| 调研结论 | gaea 落地 |
|---|---|
| Claude 式居中流布局 | 助手消息通栏流畅文本（max-width 768），用户消息右侧轻量胶囊；保留人格头像作"在场感"而非每条消息都带头像 |
| 人格在场感 | persona 模式顶部状态条：头像/名字/情绪/信任/轮次；记忆抽屉回看事实 |
| 诚实与透明 | AI 身份明确标注；情绪/信任为只读元数据；删除话题/清会话直达 |
| Liquid Glass | 输入区与侧栏用分层玻璃（外壳+内芯高光），沿用 M3 令牌，明暗一致 |
| 会话可读性 | 侧栏话题显示标题 + 首条预览 + 模式标记 |
| 错误就近 | 失败消息内联错误 + 重试按钮 |
| 无障碍 | 新消息 aria-live、键盘操作、对比度校验、44px 目标 |
