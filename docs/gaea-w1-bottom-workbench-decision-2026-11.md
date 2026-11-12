# W1 底部工作台/终端 · 拍板决策备忘（2026-11-13）

> 状态：**调研备忘，待用户拍板**（交接档 §6 W1 的四问前置）。本文只交事实与选项，不动刀。

## 一、上游事实（dsh-better-sidebar v0.32.0，commit 53e08cec）

1. **底部工作台 = 底部 Tab 区（TabBar → TabContent → EditorHost）+ 终端页**。两块能力：
   编辑器页签（文件在底部以 tab 打开，与右侧栏同组件同消费 `line`）+ xterm.js 终端
   （`TerminalView.tsx` + `@xterm/xterm` 6.0 + `@xterm/addon-fit`）。
2. **上游不拥有 PTY**（v0.21.1 起明确让渡）：终端驱动**宿主**的会话级终端服务
   （`@deepseek-ai/dsh-api-terminal-controller` 0.2.0-rc.1 的 client 半区，`ctx.webTerminals`
   结构性探测按名注入，缺服务→终端行降级 pending 而非崩）。宿主负责进程寿命、屏幕恢复、
   重连；插件只是模拟器+管道。
3. **宿主终端服务的真实复杂度**（照抄即抄这个子系统，不是抄一个组件）：
   - 帧式流而非裸流：`render` 帧待 `acknowledge`（模拟器 write 回调放行下一帧），
     未确认帧永久停流——背压控制；
   - `snapshot` 帧=整屏序列化重放（reset+resize+write），`output` 帧追加；
   - 进程跨 DOM 卸载存活（tab 切换/面板折叠/会话切换 remount 复用同进程），显式 close 才结束；
   - `(sessionId, contentId) → terminalId` 持久绑定（localStorage）——页面重载后**收养同一
     shell**；输入按挂载窗口串行化。
4. **工程卫生**：xterm 数百 KB 必须懒 chunk（`chunk-artifact.spec.ts` 钉死不进核心 bundle）。
5. 维修史提示成本：底部页签的改名/删除调解要覆盖**后台会话布局**（#929），草稿注册表键含
   会话维度；终端样式与 xterm 样式表注入顺序的 CSS 特异性坑（#168 条）。

## 二、gaea 现状对照（W1 四问之 Q2/Q3/Q4）

| 问题 | 事实答案 |
|---|---|
| Q2 交互终端比现有任务面多什么 | gaea 任务中心=只读观察+强制终止（单向）；终端=**双向**（交互输入）+ **持久 shell 态**（cd/venv/env 跨命令存活）+ **任意程序**（不止任务系统拉起的命令）+ **TUI 保真**（vim/htop 类需 ConPTY）。这是真实差距，但只在「用户想亲手敲命令」时成立 |
| Q3 窄屏折叠 | 上游做法=折叠钮+`bottomHeight` 状态+≤窄屏降级；gaea 可沿用既有 pane 折叠模式（与本仓 FileTree/右栏一致），成本小 |
| Q4 用户交互面还是 Agent 工具面 | 上游=**会话级用户交互面**（与 Agent 的 exec 工具完全分立）。gaea 建议同口径：终端=用户的手，**不走审批闸**（用户本人操作），但必须吃 workspace 围栏（cwd 锚定 gaeaCwd，走查态=沙箱 workspace）与 proc.HideWindow 纪律 |

## 三、gaea 若做的真实成本（与上游关键差异）

上游可以「不做 PTY」是因为宿主已有终端服务；**gaea 没有宿主服务，PTY 栈要自建**：
- Go 侧 ConPTY（creack/pty 的 windows 实现）+ 进程寿命管理 + 帧式输出/背压 + 会话恢复持久化；
- 前端 xterm.js 懒 chunk + ack 流控；
- 这是 W1 的成本大头，编辑器页签反而不大（gaea 已有 paneTabs/文件预览/编辑 CAS 全套可复用）。

## 四、拍板选项

- **A. 不做（维持现状）**：任务中心+浏览器面板+文件工作台已覆盖只读观察面。触发重议条件=
  用户实际需要亲手在 workspace 里交互敲命令。
- **B. 轻量做（无终端）**：底部=编辑器页签区（复用 paneTabs 与预览编辑 CAS），无 PTY 成本；
  对「多 pane 编辑」有价值的场合可选。
- **C. 全量做（含 ConPTY 终端）**：自建 PTY 子系统（Go ConPTY+帧式流+会话恢复）+xterm 懒
  chunk。按上游维修史，这是 W1 的 80% 成本所在。

## 五、建议

若 Q1（是否需要多 pane/多终端）没有明确的「要」，**建议 A**（观察项，触发条件见上）；
确有编辑分栏需求但终端需求弱，**B** 是性价比点；**C** 只在终端被高频想象使用时立项。
拍板后按交接档 §7 流程立刀。
