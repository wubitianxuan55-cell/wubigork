# 浏览器观察窗（v4.26 A2）市场调研

日期：2026-09-01。范围：agent 驱动浏览器时「机器在看的页面」如何呈现给用户；对标 Manus、OpenAI Operator/ChatGPT agent、Claude computer use、Devin、Playwright/DevTools、Cherry Studio。

## 1. 「机器在看的页面」三种路线

**a) 真实标签页直看（零镜像）**：Manus Browser Operator 的 "My Browser" connector 不做截图流或远程镜像，而是接管本地浏览器真实标签页——"opens a new tab within a tab group named after your current task"，用户 "watch the task unfold in real-time"，看到的就是真页面（[manus.im/blog/manus-browser-operator](https://manus.im/blog/manus-browser-operator)）。成本最低、无延迟，但依赖浏览器扩展且占用用户浏览器会话。

**b) 截图步进流**：Operator 界面为"左聊天面板 + 右侧可见界面"（[simonwillison.net/2025/Jan/23/introducing-operator](https://simonwillison.net/2025/Jan/23/introducing-operator/)）；Claude 参考实现在聊天流里对每次工具结果渲染 `st.image(base64)`，截图本就是模型的观察原语，顺手复用，并提供 "Hide screenshots" 开关（[github.com/anthropics/anthropic-quickstarts](https://github.com/anthropics/anthropic-quickstarts) `computer-use-demo/streamlit.py`）。粒度=每动作一张，延迟高、不连续，但可回放、可入证据链，流量小。

**c) 实时帧流**：Anthropic demo 除聊天截图外另开 noVNC 网页查看器（`http://localhost:6080/vnc.html`）直看桌面（同上 README）；CDP 侧对应 `Page.startScreencast` 帧推送（见 §5）。连续流畅，但带宽、CPU 与实现成本最高。

## 2. 操作时间线、截图联动与权限门

Playwright Trace Viewer 是最成熟的先例：每个 trace "records a screencast and renders it as a film strip"，Actions 列表展示每个动作的 locator 与耗时，DOM 快照分 Before/Action/After 且 Action 快照"highlights both the DOM Node as well as the exact click position"；时间线可拖拽筛选，双击动作即把网络/控制台过滤到该动作时间窗（[playwright.dev/docs/trace-viewer](https://playwright.dev/docs/trace-viewer)）。即：**时间线条目 ↔ 截图帧 ↔ 定位高亮** 三向联动。Chrome DevTools Recorder 以步骤列表为核心（"shows the steps that you have performed so far"，可展开 type/target/selectors 细节），回放支持减速、断点、单步（[developer.chrome.com/docs/devtools/recorder](https://developer.chrome.com/docs/devtools/recorder)，经 GitHub 源核实；其"每步截图 filmstrip"未见于现行官方文档，未核实）。

权限门呈现：Operator 对外部副作用强制确认——系统提示词要求 "Ask the user for final confirmation before the final step of any task with external side effects"（[simonwillison.net/2025/Jan/26/chatgpt-operator-system-prompt](https://simonwillison.net/2025/Jan/26/chatgpt-operator-system-prompt/)），检测到可疑屏幕内容会暂停并让用户 "Mark safe and resume"（[同站 2025/Jan/23 及 01/26 更新](https://simonwillison.net/2025/Jan/23/introducing-operator/)）。gaea 可把权限卡作为时间线上的内联节点（动作条目旁挂"待批/已批/拒绝"状态），不必打断主对话流。Devin 则把 shell/浏览器/IDE 视图挂在进度步骤下，点步骤即跳转对应工具视图（[docs.devin.ai/get-started/devin-intro](https://docs.devin.ai/get-started/devin-intro)）——时间线即工作台导航，与右栏 Tab 注册表思路契合。

## 3. 自动弹出与勿扰

未见任何产品公开"自动弹出+可关"的规范描述（未核实）。可观察到的模式：Operator/ChatGPT agent 是常驻可见界面，Willison 批评其"主要防御是期待用户全程盯着"（[simonwillison.net/2025/Oct/22/openai-ciso-on-atlas](https://simonwillison.net/2025/Oct/22/openai-ciso-on-atlas/)）；Manus 用任务命名标签组在用户自己的浏览器里承载观察，不打弹窗、"simply close the dedicated tab" 即停止（manus.im，同 §1）。对 gaea 的启示：自动操作进行中自动弹出观察窗（可关），关闭后以角标/时间线红点承接注意力，是合理差异化；权限卡弹出时同步置顶观察窗。

## 4. 人工接管（远期参考）

- Operator："take control" 用于人工完成敏感步骤（发布演示中人工填信用卡，simonwillison.net/2025/Jan/23）。
- ChatGPT agent/Atlas："you can pause, interrupt, or take over the browser at any time"（simonwillison.net/2025/Oct/22）。
- Manus：点进标签页即接管，关标签即停（manus.im）。
- Devin："jump in to help Devin navigate... via the Interactive Browser"（docs.devin.ai）。
- Claude demo：桌面本身经 VNC 可交互，但无显式接管按钮（github.com/anthropics/anthropic-quickstarts README）。

## 5. Wails 技术可行性注记

- **轮询（建议首版）**：CDP `Page.captureScreenshot` 单帧返回 base64（支持 jpeg/png/webp、clip、optimizeForSpeed），按动作触发+空转低频轮询，Go 侧 `EventsEmit` 推 base64 JPEG 到前端 `<img>`。延迟=轮询间隔，流量可控，截图同时沉淀为时间线/证据链素材（[chromedevtools.github.io/devtools-protocol/tot/Page](https://chromedevtools.github.io/devtools-protocol/tot/Page/)）。
- **帧推送**：`Page.startScreencast` 经事件推 JPEG 帧，参数含 quality/maxWidth/everyNthFrame，带 `screencastFrameAck` 流控（maxFramesInFlight 默认 3）与 sendLastFrame 低延迟选项；注意标注 Experimental，需自实现 ack 与限帧。Claude 参考实现的循环本身无节流 sleep，靠"每动作一张截图"天然限速（github.com/anthropics/anthropic-quickstarts `loop.py`）。
- **Wails 通道**：Go↔JS 统一事件系统，`EventsEmit(ctx, name, data...)` payload 为 `interface{}`，前端 `EventsOn` 接收（[github.com/wailsapp/wails](https://github.com/wailsapp/wails) website docs/reference/runtime/events.mdx；wails.io 站点在本网络 403）。

## 6. 未核实项

「Context providers 浏览器扩展」：多渠道检索无果，未能确认所指产品（未核实）。Cherry Studio README 未含任何 agent 浏览器/观察窗能力（[github.com/CherryHQ/cherry-studio](https://github.com/CherryHQ/cherry-studio)）。Operator/Claude 官网原文因 403/区域屏蔽未直接核实，均经上述二手或源码来源转述。browser-use 文档快速上手页未记载观察 UI（docs.browser-use.com）。
