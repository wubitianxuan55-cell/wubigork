# 版本时间线与产物 Diff · 对标调研（2026-09-01）

> 对象：规划 B1「文件版本时间线」（vN 徽标可点 → 逐版本列表 + 预览 + 恢复）。方法：官方文档与一手博客实际抓取核实；被墙/拒绝抓取处如实标注「未核实」，未编造。

## 1. 消费级 AI 产品的产物版本历史

**Notion 是最完整的先例（全部核实）**：编辑期间每 10 分钟自动记一版、停止编辑 2 分钟后补记一版；页顶 🕘 图标点开即「本页全部修订与评论」的下拉列表；点任一版本可看当时页面样貌并**高亮本次变更**（文本增删、块级增删、部分表格变更）；「Restore」一键把旧版置为当前版，旧版仍留在历史里；保留期按套餐 7/30/90 天/不限（notion.com/help/duplicate-delete-and-restore-content）。启示：版本=时间戳自动聚合，不做里程碑命名；预览+变更高亮是恢复前的护栏；AI 改写与手改同轴记录。

**Claude Artifacts**：官方文档只确认「版本选择器」（"Switch between different versions using the version selector"），每轮修改在原窗口就地生效，编辑历史消息会分叉出带独立产物的新对话分支（support.claude.com/en/articles/9487310）；「回退旧版」按钮官方文档未写，**未核实**；2026-08-31 的发布说明里也检索不到 version history / rewind 条目（support.claude.com/en/articles/12138966）。Claude 的 docx/xlsx/pptx 文件产物官方只写创建/编辑/下载，**没有版本历史功能**（support.claude.com/en/articles/12111783）——gaea 做文件级时间线在消费 AI 中无对标，反而领先。

**ChatGPT Canvas**：定位「共享编辑环境」（simonwillison.net/2025/Jan/25/openai-canvas-gets-a-huge-upgrade）；版本导航/回退细节因 help.openai.com 拒绝抓取，**未核实**。

**M365 Copilot**：版本历史依赖 Office/OneDrive/SharePoint 原生能力（File > Info > Version History），Copilot 产稿遵循 Word 修订；support.microsoft.com 检索被拒，**未核实**。

## 2. 编码工作台 checkpoint：最可迁移的机制

- **Cursor**：Agent 在重大改动前自动建 checkpoint，捕获所有被改文件；时间轴上点任意点即预览当时文件；Restore 只回退文件、不动对话；「本地存储、独立于 Git」（cursor.com/docs/agent/chat/checkpoints）。
- **Claude Code**：每个用户 prompt 自动存一次 checkpoint（追踪其文件编辑工具的全部改动）；`/rewind` 或双 Esc 出菜单，三选：恢复代码+对话 / 仅对话 / 仅代码；每会话保留最近 100 个快照文件、会话按 30 天清理（code.claude.com/docs/en/checkpointing）。**关键局限**：bash 命令改的文件不追踪。
- **Codex（cloud→web）**：不做快照，任务跑隔离环境、直接「还你一个 diff」（simonwillison.net/2025/May/16/openai-codex/），代码评审用临时容器（simonwillison.net/2025/Sep/15/gpt-5-codex/）——是产物级审查先例，不是版本管理先例。
- **Copilot Workspace**：githubnext 项目页仅剩标题、正文 JS 渲染无法核实（githubnext.com/projects/copilot-workspace）；**Windsurf** Cascade checkpoints 文档已 404（docs.windsurf.com 现整体重定向 docs.devin.ai），只能存疑。

**迁移判断**：gaea 条件比 Claude Code 更好——所有写操作都经自家工具且已落产物登记表，不存在「绕过追踪」的暗改动。实现无需 shadow git：**内容寻址即可**——写前把旧内容按 hash 存本地对象库，登记表就是索引，vN 计数=该产物快照数。建点粒度对齐 Claude Code（按轮次=按 prompt 节点）或 Cursor（按显著改动），与登记表现有「轮次/时间/次数」字段天然对齐。

## 3. 非代码文件的 diff 呈现

- **docx**：Word 修订/比较是行业标准语义，gaea 已对齐（接受/拒绝全部修订）；行级纯文本 diff 只是降级选项。GitHub 对不渲染的文件**不给内容 diff**、仅版本化；PNG/JPG/GIF/PSD/SVG 例外，提供 2-up / swipe / onion-skin 三种视觉对比（PSD 不支持比较）（docs.github.com/repositories/working-with-files/using-files/working-with-non-code-files）。
- **xlsx**：单元格级 diff 无 Office 原生入口（Spreadsheet Compare 属 ProPlus Inquire 组件，本次未能核实）；gaea 的 Plan→Apply 单元格 diff 属先行形态。
- **二进制**：只能版本化不能 diff，先例做法是占位说明 + 整档预览旧版，不伪装 diff。

## 4. 恢复 UX 细节

- **确认与否**：Notion「选版本→预览→Restore」无二次弹窗，预览即防误触；Cursor 的 Restore Checkpoint 按钮即执行；Claude Code 用「选节点→选动作」两步菜单，天然一次确认（来源同前）。
- **原版本去向**：三家共同不变量是**丢当前态、不丢历史**——Notion 恢复后旧版仍在历史中；Cursor/Claude Code 恢复前的状态本身就在 checkpoint 序列里可再翻回。gaea 应遵循：恢复=写入新版本，永不覆盖删除。
- **多文件批量回退**：checkpoint 模式天然支持整工作区批量；Notion 反例是数据库恢复不含页内容、需逐页恢复（来源同前）。gaea 取「按产物逐个恢复 + 按轮次整批恢复」两层。

## 5. vN 徽标 → 时间线入口

先例三分：**图标/popover**——Notion 🕘 点开下拉列表，最贴 gaea 徽标场景；**侧栏面板**——VS Code Explorer 的 Timeline view 按文件列历史、可含本地文件改动（code.visualstudio.com/docs/sourcecontrol/overview），Google Docs 版本历史右侧面板（**未核实**，Google 域抓取被网络阻断）；**命令/菜单**——Claude Code `/rewind`。建议 gaea 双入口：徽标点开 popover 快速翻版（对齐 Notion），「查看全部」切入右栏时间线面板承接预览+恢复，与「变更」tab 并列。

## 对 B1 的落点（三条）

1. 版本=自动聚合的时间戳快照（每轮/每次写工具落盘一版），不做里程碑命名；vN 计数与产物登记表同源。
2. 恢复不丢当前版：恢复=新增版本；徽标 popover 快翻 + 右栏时间线双入口，恢复前必见预览。
3. docx 走修订、xlsx 走单元格 diff、二进制只版本化+整档预览，与 v4.24.0 既有语义连续；底层用内容寻址快照库，登记表作索引。
