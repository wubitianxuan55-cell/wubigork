# 模块制市场调研 · 2026-09-01（v4.24.0 基线）

> 3 分模块子代理调研（浏览器观察窗 / 文件版本时间线 / pptx 与工作台动向），原始稿在
> `docs/research-2026-09-01/`（browser-observation.md · version-timeline-diff.md ·
> pptx-workbench-trends.md），来源随句标注，未核实项在原始稿内明示。目的：为
> v4.26「浏览器与版本」（A2 观察窗 + B1 版本时间线 + B2 pptx + C2/C3）及后续供弹药。
> 网络限制：openai.com / anthropic 官网 / google 域本环境 403 或超时，相关结论均经
> 二手（simonwillison.net）与官方源码/文档（github.com、chromedevtools.github.io）转述核实。

## 一、浏览器观察窗（A2）

- **呈现路线三分**：真实标签页直看（Manus Browser Operator，零镜像、按任务命名标签组）/ 截图步进流（Operator、Claude computer use：每动作一图，聊天流内可关）/ 实时帧流（Claude demo noVNC；CDP 对应 Page.startScreencast）。成本与延迟依次升高。
- **时间线联动最成熟先例 = Playwright Trace Viewer**：screencast filmstrip + Actions 列表 + Before/Action/After DOM 快照 + 点击坐标高亮 + 拖拽过滤。Devin 把工具视图挂到进度步骤下（点步骤切视图）——「时间线即工作台导航」与 gaea 右栏 Tab 注册表同构。
- **权限门呈现**：Operator「外部副作用最后一步前强制确认」、可疑内容 "Mark safe and resume"；设计推断：权限卡做成时间线内联节点（待批/已批/拒绝），不打断主对话流。
- **自动弹出/勿扰无公开规范**（未核实）：Operator 类常驻界面被批「期待用户全程盯着」，Manus 关标签即停；gaea「自动操作时自动弹出可关 + 收起后角标承接」属差异化设计。
- **接管（远期）**：Operator "take control"、ChatGPT agent "pause/interrupt/take over"、Devin "jump in"。
- **技术起版建议**：CDP `Page.captureScreenshot` 单帧轮询（jpeg、optimizeForSpeed）+ Wails EventsEmit 推送，截图兼作证据链素材；`Page.startScreencast` 帧流（everyNthFrame 限频 + FrameAck 流控，Experimental）仅在「自动操作中且观察窗打开」时启用。

## 二、文件版本时间线（B1）

- **最强参照 = Claude Code checkpoints / Cursor checkpoints**：按用户 prompt（gaea 按轮次）自动建点、捕获全部被改文件、点时间轴预览、Restore 只回文件不动对话、本地内容寻址存储独立于 Git。Claude Code 每 prompt 一点、留 100 点/30 天。
- **gaea 条件更优**：Claude Code 明确「bash 改的文件不追踪」是暗区；gaea 全部改动走自家工具+v4.24 权威登记表，无此暗区——**登记表就是版本索引**，vN 计数=快照数。
- **入口先例三分**：Notion 🕘 图标 popover（最贴 vN 徽标场景：点版本看样貌+变更高亮→Restore 一键）、VS Code Timeline 侧栏（按文件列历史）、Claude Code /rewind 命令。建议 gaea 双入口：**vN 徽标 popover 快翻（Notion 式）+「查看全部」切右栏时间线（与变更 tab 并列）**。
- **恢复 UX 不变量**：丢当前态、不丢历史——恢复=写入新版本，永不覆盖删除；Notion/Cursor 无二次确认弹窗（预览即护栏）。多文件批量回退：checkpoint 模式天然整区批量 → gaea 取「按产物逐个 + 按轮次整批」两层。
- **格式语义**：docx 行业标准=Word 修订（gaea 已对齐，纯文本 diff 只是降级）；xlsx 单元格级 diff 无 Office 原生入口（gaea Plan→Apply 属先行形态）；二进制/图像：GitHub 对图像提供 2-up/swipe/onion-skin 视觉对比，不渲染文件仅版本化。
- **差异化领先位**：Claude 文件产物（docx/xlsx/pptx）官方确认无版本历史——gaea 做文件级时间线在消费 AI 中无对标。

## 三、pptx 与工作台新动向（B2 + 品类复扫）

- **页级指令标准做法（Copilot in PowerPoint）**：两通道并存——"this slide"（相对引用）与 "Update slides 4–6"（显式页码）；Skills 菜单把页级命令预制化（Review this presentation / Visualize this slide）。
- **大纲确认前置**：Beautiful.ai outline-first 生成前强制逐页大纲确认，迭代不破结构。gaea 落点：大纲确认做成生成前硬门槛；点页→指令同时支持显式页码与「当前选中页」两通道。
- **缩略图技术路线**：先做 **python-pptx 结构化大纲卡**（原生读 slides/shapes/text_frame，零依赖毫秒级），像素版用 soffice headless→PDF→PyMuPDF 逐页光栅化按页懒渲染+缓存（坑：中文字体漂移、soffice 冷启动/单实例 profile 锁，社区经验未核实原始帖）。
- **品类新标杆 Claude Cowork**（2026-02→08 滚动更新，claude.com）：桌面 App + 侧栏内置浏览器 + 文件夹级读写授权（直接处理 pptx/xlsx/docx）+ 定时任务 + Skills/Connectors/Sub-agents + 步骤可视——侧栏品类共识已从「聊天侧栏」进化为「代理工作台」（授权可见+步骤可视+定时+并行），gaea v4.23 起方向一致。
- **M365 Copilot 动向**：2026-04 Word/Excel/PPT agentic 能力 GA；2026-05-28 全新设计（side pane=「会建议也会直接改」的 editing partner，可在段落/单元格/幻灯片内就地唤起）上线后 PPT 使用 +43%；06 Work IQ APIs + 常驻代理 Scout；微软自列下一步 = **"more transparency / preview of changes"** —— 正是 gaea Verify→Journal→一键回滚的先发位，「改了哪几页、可还原」应成 PPT 工作台默认 UI。
- 国内：Kimi Agent 侧栏一级入口「PPT/文档/表格」；WPS AI 完整 AI 在客户端（网页版仅基础）；飞书官网定位改「AI 工作平台」。Gemini in Slides / Tome / 钉钉 2026 细节未核实（域被过滤）。

## 四、对刀序的影响（v4.26 收敛建议）

1. **A2 观察窗**：截图步进流起版（captureScreenshot 轮询+事件推送，截图进证据链）；操作时间线对标 Playwright Trace Viewer 的 Actions 列表；权限卡做时间线内联节点；自动弹出+角标承接保持差异化；帧流仅在观察窗打开时启用（可后置）。
2. **B1 版本时间线**：实现=写前旧内容按 hash 入本地内容寻址对象库，**与 v4.24 登记表同源**（登记表即索引，vN=快照数）；恢复=新增版本+恢复前必见预览；双入口=vN 徽标 popover + 右栏时间线；docx 走修订语义、xlsx 走 Plan diff、二进制整档版本化。
3. **B2 pptx**：结构化大纲卡先行（python-pptx，零依赖），像素缩略图懒渲染+缓存后置；页级指令两通道（显式页码+当前选中页）；大纲确认前置硬门槛。
4. **C2/C3 不变**；远期新增候选：浏览器人工接管、screencast 帧流、图像 2-up/swipe 对比、按轮次整批回退。
