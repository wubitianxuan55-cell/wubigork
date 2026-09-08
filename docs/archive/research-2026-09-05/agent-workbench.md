# 调研：AI 编码代理 / 代理工作台开源项目的 UI/UX 能力面（2026-09-05）

调研维度：上下文用量可视化、子代理/后台任务管理、工具调用结果渲染（含图片）、文件活动追踪、任务生命周期（终止/退出码/追问）。数据来自 GitHub 页面、releases、CHANGELOG 与官方文档快照；查不到的写「未核实」。

## 1. openai/codex
- 链接：https://github.com/openai/codex ｜ 星数：约 121.5k ｜ License：Apache-2.0
- 最近版本：0.153.2（2026-09-03；同日 0.153.0 为功能版）
- 亮点：
  - 上下文可视化：TUI 会话横幅常驻显示「100% context left」，`/status` 查看会话配置；0.153.0 引入实验开关 `features.context_management.experimental_mode`（token 预算上下文 + history notes + `new_context` 工具）。
  - 工具渲染：TUI 历史流展示完整 patch、发往后台终端的输入、逐条完成的命令。
  - 子代理/任务：支持把聚焦工作委派给专用代理后回收结论；`codex cloud` 并行云任务可浏览/回填本地仓库；`/review` 只读审查（不动工作区）。
  - Guardian 审查历史在 compaction/重启/fork 后仍隔离保留子代理历史。
- gaea 可借鉴：把「剩余上下文 %」做成常驻状态栏，而非按需查询；上下文管理作为可开关实验特性（预算+历史笔记）渐进灰度。

## 2. anthropics/claude-code
- 链接：https://github.com/anthropics/claude-code ｜ 星数：约 144k ｜ License：专有（Commercial Terms，非 OSS）
- 最近版本：无 GitHub release，npm 分发；CHANGELOG 最新 2.1.260（条目无日期，2026-09 活跃）
- 亮点：
  - 文件活动：2.1.260 新增 `/diff` 面板，在对话旁并排实时显示 Claude 编辑产生的未提交变更。
  - 子代理：子代理/teammate 的 token 计数器跨 transcript 切换实时更新；后台子代理遇中断自动续跑；停止子代理连带清理其 monitors。
  - 生命周期：修复史显示终止语义在收敛——中止需级联取消派生任务、分离后台命令不得在 stop/exit 后存活。
  - 上下文：接近 1M 上限前提前 compaction；长截图会话的图片尺寸上限与 prompt cache 失配修复。
- gaea 可借鉴：终止必须级联（主任务停→子代理/监控/后台命令全停）；文件 diff 用独立面板而非混入聊天流。

## 3. sst/opencode（仓库现显示为 anomalyco/opencode）
- 链接：https://github.com/sst/opencode ｜ 星数：约 204k ｜ License：MIT
- 最近版本：v1.18.28（2026-09-04）；桌面 App 处于 beta（macOS/Windows/Linux）
- 亮点：
  - 一引擎多前端：同一会话可跑 TUI/桌面/Web，桌面端与 CLI 共享账号设备认证。
  - 双代理：Tab 键在 build（全权限）/plan（只读、bash 需确认）间切换；`@general` 子代理承接多步搜索。
  - 子代理可恢复：v1.18.20「以可续跑的 task_id 呈现失败的子代理工具调用」——失败不是终点而是可恢复对象。
  - 会话 ID 作为请求头随 Copilot 等供应商传递，便于全链路追踪。
- gaea 可借鉴：引擎/前端分离，会话可迁移；失败的子代理调用给出「可恢复 task_id」而非一次性报错。

## 4. cline/cline
- 链接：https://github.com/cline/cline ｜ 星数：约 67.5k ｜ License：Apache-2.0
- 最近版本：Desktop v0.0.23（2026-09-03；同期 CLI v3.0.61、SDK v0.0.82）
- 亮点：
  - 图片结果渲染：工具结果中的图片渲染为内联可点击图片，多图用轮播，替代原始 base64 文本（Desktop v0.0.20）。
  - 子代理体系：独立上下文窗口+独立 token 预算、严格只读；聊天界面实时显示每个子代理的 tool calls/tokens/cost，成本单独核算再汇总。
  - 生命周期：中止任务会级联取消其派生的子代理与 teammates，已取消任务持久保留为 cancelled 状态；拒绝消息明确点名被拒工具、以用户决策而非错误呈现。
  - checkpoint 安全：存在后续 commit 时拒绝恢复 checkpoint，防静默破坏分支。
- gaea 可借鉴：子代理「只读+独立预算+逐个成本面板」是最稳妥落地组合；图片工具结果用轮播组件而非文本占位。

## 5. RooCodeInc/Roo-Code（已停运，作风险样本）
- 链接：https://github.com/RooCodeInc/Roo-Code ｜ 星数：约 24.3k ｜ License：Apache-2.0
- 状态：2026-05-15 归档（read-only），扩展停运，官方推荐 ZooCode/Cline；最终版本号未核实。
- 亮点（停运前）：Code/Architect/Ask/Debug + 自定义模式的「多模式即人格」体系；与 Cline 同源的 checkpoint/编辑 diff 能力。
- gaea 可借鉴：反面教材——依附单 IDE 插件生态风险高；但其「按场景切模式且模式绑定权限/提示词」的 UX 值得吸收。

## 6. block/goose
- 链接：https://github.com/block/goose ｜ 星数：约 53.9k ｜ License：Apache-2.0（Rust；隶属 Linux 基金会 AAIF）
- 最近版本：v1.49.0（2026-09-03）；另有桌面独立版本通道（Desktop v0.0.23，2026-09-03）
- 亮点：
  - 审批透明：先向用户展示工具输入再请求批准（#10932）；稳定 tool_call_id 贯穿工具全生命周期（#11120）。
  - 上下文治理：统一上下文上限解析、handoff memo 限幅保证长会话可恢复、压缩失败给出诚实提示并在无工具响应时快速失败。
  - 桌面 UX：聊天底栏交互式 git 分支指示器、定时任务会话折叠为手风琴、并行子代理通知相互隔离。
- gaea 可借鉴：「审批时看得见输入」+ 稳定 tool_call_id 是信任基础；压缩失败必须显式告知，不做静默降级。

## 7. charmbracelet/crush
- 链接：https://github.com/charmbracelet/crush ｜ 星数：约 27.9k ｜ License：FSL-1.1-MIT
- 最近版本：v0.92.0（2026-08-31）
- 亮点：
  - 工具渲染：Bash 工具输出带 bash 语法高亮、剥离冗余 cd 前缀；修复工具预览缩进丢失。
  - 会话：会话管理器暴露 IsBusy / AttachedClients 状态，会话可跨客户端共享；退出时打印 session ID 便于 `run --session` 续用；会话选择器支持鼠标滚轮/点击。
  - 上下文：LSP 补充代码上下文；权限 allow/deny 白名单 + `--yolo` 全跳过两极可配。
- gaea 可借鉴：工具输出按语言语法高亮渲染；会话「忙/被挂接」状态显式暴露给多客户端。

## 8. google-gemini/gemini-cli
- 链接：https://github.com/google-gemini/gemini-cli ｜ 星数：约 107k ｜ License：Apache-2.0
- 最近版本：稳定版 v0.58.0（2026-09-01；周二周更节奏，nightly 每日）
- 亮点：会话 checkpoint 保存/恢复；1M token 上下文；VS Code 伴随扩展 + GitHub Action（PR 审查/issue 分诊）；headless 模式供脚本化。
- gaea 可借鉴：固定周更节奏培养用户预期；checkpoint 让「复杂会话可存档再恢复」。

## 综合观察：2026 年代理工作台 UX 趋势
1. 上下文可视化从「命令查询」升级为「常驻横幅/实时计数」：codex 常显 context left、claude-code 子代理计数器实时刷新；同时上下文管理被工具化（token 预算、history notes、new_context、handoff memo、auto-compaction）。
2. 终止即级联：主任务中止必须传播到子代理/teammates/monitors/后台命令，且已取消任务持久保留可查（codex、cline、claude-code 的修复史均为证据）；「已取消」是持久状态而非消失。
3. 工具结果富媒体化、按类型渲染：图片内联可点+轮播（cline），patch 完整展示（codex），bash 语法高亮（crush）；审批前先可见工具输入（goose）成为信任标配。
4. 文件活动从聊天流抽出为专属面板/控件：claude-code `/diff` 并排面板、goose git 分支指示器、cline checkpoint+diff 回滚；伴随「恢复的安全性护栏」（有后续 commit 拒绝恢复）。
5. 形态收敛为一引擎多前端（TUI/桌面/Web 共享会话，opencode/goose/crush），失败子代理以可恢复 task_id 呈现；同时警惕单生态依赖——Roo-Code 停运即前车之鉴。
