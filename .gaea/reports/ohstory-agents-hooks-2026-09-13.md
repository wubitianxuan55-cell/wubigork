# oh-story 蒸馏简报 · 子代理 / 钩子 / 状态追踪（2026-09-13）

> 上游=只读仓 `clones/oh-story-claudecode`（MIT，网文写作插件，11 skill + 7 子代理 + 多宿主 hooks）。行号口径=按 `\n` 切分（非 PowerShell `Get-Content`）。
> **任务书路径勘误**：`skills/story-import/scripts/tracking-transaction.py` 不存在——事务语义在 `skills/story-import/references/tracking-transaction.md`，实现只有 `skills/story-import/scripts/tracking_commit.py`（1229 行）。
> gaea 基线（本仓实测）：hook 引擎 `internal/gaea/hook/`、任务中心 `internal/gaea/tasks/`、子代理 `internal/gaea/agent/`、技能 `internal/gaea/skill/`、生成门 `internal/app/novel_gate_handler.go`、文风/去味 `internal/novelstyle/`。

## A. 子代理（4 条机制）

### A1 七角色职责分工 + 权限矩阵（真增量=角色卡内容；机制 gaea 已有）
- 证据：`.../templates/agents/chapter-extractor.md:2-10`、`consistency-checker.md:2-13`、`story-explorer.md:2-14`、`narrative-writer.md:2-14`、`story-architect.md:2-13`、`character-designer.md:2-11`、`story-researcher.md:2-12`；Codex 变体用 `sandbox_mode` 表达同一权限（`references/codex/agents/consistency-checker.toml:9`、`chapter-extractor.toml:8`、`story-explorer.toml:11` = `read-only`）。
- 规则：只读三员（chapter-extractor / consistency-checker / story-explorer）显式 `disallowedTools: [Write, Edit, Bash]` 且**故意不设** `memory: project`（注释理由：memory 会隐性开启 Write，与只读矛盾，见 `consistency-checker.md:11-12`）；创作四员按成本分模型（architect=opus、writer/designer=sonnet、抽取=haiku），`maxTurns` 12~30 封顶。
- 落点：**技能资产**——7 张角色卡提示词按 gaea 形态重写为 `.gaea/skills/novel-agents/`，权限以 gaea 的 permission/sandbox 面表达；不新建子代理引擎（`internal/gaea/agent/agent.go` + `internal/gaea/cache/spawn.go:99-102` 的 `subagent_explore/research/review/security` 模板已提供机制）。

### A2 交接产物是**固定结构**而非自由文本（真增量）
- 证据：细纲蓝图 11 字段 + 五段式内容概括 + 情节点表 + 复沓锚句 `story-architect.md:68-117`；抽取员 12 项输出契约 `chapter-extractor.md:289-305`；冲突报告 S1~S4 分级 `consistency-checker.md:119-139`。
- 规则：上游把「写手从架构师手里拿什么」写死成可机械校验的文件格式（细纲在 `大纲/细纲_第XXX章.md`），写手铁律 1 明令「细纲是唯一剧情蓝图，每项落地演足、不许合并交代」（`narrative-writer.md:26`）。
- 落点：**内核纯函数（校验）+ 数据资产（模板）**——gaea 的 `types.OutlineNode` 已有 summary/scene_ideas/key_points/emotion；增量是把「必填字段齐备性」做成写前契约检查器（对应 A2 与 B1 联动），存疑字段如「复沓锚句」需按 gaea 大纲结构裁剪后再收。

### A3 产出不达标 → 主线程**升级重试一次**（真增量）
- 证据：`chapter-extractor.md:289-291`：12 条自检清单同时是主线程的升级重试触发条件，任一不达标则用 sonnet 覆盖该 agent 默认 haiku 重新 spawn，**仅 1 次**；`narrative-writer.md:12-14` 注明 subagent 不允许嵌套 spawn（故不加载会 spawn reviewer 的 skill）。
- 规则：质量兜底不靠「让弱模型自证」，而是「父流程按清单复验 + 限额升配重跑」；同时禁嵌套 spawn 以控制爆炸半径。
- 落点：**内核机制**——可作 gaea 任务子代理的通用策略（失败升配一次）；与 gaea 现有 `SubagentStop` 钩子事件天然契合，无需改 UI。

### A4 职责边界 + 三档越界处置（真增量，写作域特有）
- 证据：`narrative-writer.md:28-36` 新增物三档（一档自由裁量／二档写并申报／三档 blocking 不写并报偏纲，拿不准按二档）；`narrative-writer.md:42` 不写追踪文件、不改大纲；`consistency-checker.md:162-167` 只读、不做创作判断，设定矛盾升级给 architect、角色不一致升级给 designer。
- 规则：「申报—收编」由主会话判定，子代理自己不建档；越界三档 = 明确的停机条件（停止交付、不自动修、报告偏纲）。
- 落点：**技能资产 + 内核校验闸**——三档语义适合做成写后闸的判定表（与 gaea 现有 `novel_gate_handler.go:14,27` RunChapterGate 的审查链合并），避免另起一套。

## B. 钩子（7 条机制）

### B1 写前门：无细纲/大纲不许写正文（真增量核心）
- 证据：`.../templates/hooks/guard-outline-before-prose.sh:2-12`（PreToolUse 挂 Bash|Write|Edit|MultiEdit，拦截三类）、`:145-161`（长篇首建要求同书 `大纲/细纲_第N章*.md`，容忍补零与标题后缀）、`:117-122`（短篇要求同目录 `小节大纲.md`，且必须有 `设定.md` 信号才拦）、`settings-hooks.json:43-51`。
- 规则：`exit 2` 即 BLOCKING；细纲门**只在首建时判**，追踪门首建与续写都判；设计原则写死「宁可漏拦不可误伤——任何不确定都 exit 0」，非正文路径静默放行。
- 落点：**内核校验闸**——gaea 已有 hook 引擎（`internal/gaea/hook/hook.go:34-70`：事件枚举、`exit 2` 阻断、`IsBlocking` 仅 PermissionRequest/PreToolUse/UserPromptSubmit 可拦），只缺这条**领域规则**；写前契约检查应做成纯函数 + 数据资产，不做成 shell。

### B2 写前欠账门：上一章毒句式没清，不许开下一章（真增量，跨章债务）
- 证据：`guard-outline-before-prose.sh:181-208`。
- 规则：写第 N 章首建前扫描上一章，命中「毒句式」即拦并列出前 8 条；用户显式豁免写作 `<!-- 去味:跳过 -->`（标题行下 6 行内）；判定无状态（现算自上一章文件）。
- 落点：**内核校验闸**——gaea 去味词表已外置且 `internal/novelstyle` 有 DeSlopRewrite，这条把「事后再洗」前移成「先清再写」，工程成本低、收益直接。

### B3 写后网：只抓硬信号、永不卡流程（机制 gaea 部分已有）
- 证据：`hooks/check-prose-after-write.sh:2-20`（PostToolUse on Write|Edit|MultiEdit；只兜截断/拒绝语/AI 自指/工程词/紧邻复读/毒句式，advisory、无发现完全静默、node 缺席静默放行）、`:86-92`（<200 字节判落盘失败）、`:96-97`（内容网走共享核 `prose-net`）、`:99-107`（exit 0）。
- 规则：钩子只报「漏跑最伤、弱模型自己发现不了」的硬信号；碎句号/长段落等 advisory 留给 workflow 全量跑，钩子不部署重量级检测器。
- 落点：**内核纯函数 + 事务后置提示**——gaea 的「章节体检」在 `RunChapterGate`（应用内手动触发）；增量是把「生成后自动复扫硬信号」接到生成链尾（可复用 `internal/gaea/hook` 的 PostToolUse 语义，或任务完成回调）。

### B4 会话生命周期四钩：SessionStart 注入 / Pre-PostCompact / SessionEnd（机制 gaea 已有）
- 证据：`settings-hooks.json:3-28,65-86`；`hooks/session-start.sh:130-139`（注入 `追踪/上下文.md` 前 18 行「当前位置」）、`:141-152`（未完成拆文计数）、`:154-188`（更新检查 24h 节流 + 失败负缓存）；`pre-compact.sh:18-28`（只报摘要与 git 变更计数，不 dump 内容）、`post-compact.sh:14-19`（提示回读上下文恢复）；`session-end.sh:12-16`（默认静默、`STORY_SESSION_LOG=1` 才落日志）。
- 规则：钩子在「无可用信息时完全静默」以免污染上下文；compact 前后用**外部文件**（上下文.md）而非会话记忆承载写作状态。
- 落点：gaea 已有 `SessionStart/SessionEnd/PreCompact` 事件（`internal/gaea/hook/hook.go:51-55`）与记忆系统；**真增量只有「写作状态卡注入」这一条数据契约**，其余不采纳。

### B5 缺口检测（SessionStart 第二钩）：5 类结构性欠账（部分真增量）
- 证据：`hooks/detect-story-gaps.sh:53-55`（正文 >10 章而设定 <3 个）、`:58-77`（伏笔表状态列异常/过期）、`:83-88`（长篇缺 `大纲/`、短篇缺 `小节大纲.md`）、`:112-129`（跨批连续性：追踪 staleness + 两章撞名，走 node 共享核 `continuity`）。
- 规则：阈值写死、只报书目级摘要、按「最终状态」过滤已完成拆文（裸数文件会把拆完的书永久误报）；连续性用 mtime +1s 容差，明示为启发式 advisory。
- 落点：**内核纯函数**（缺口=结构化查询）——gaea 侧最接近的是 `/api` 统计与任务中心，可先做「写作前体检」面板，避免每次会话刷屏。

### B6 commit advisory（PreToolUse on git commit）：WARNING only（真增量小）
- 证据：`hooks/validate-story-commit.sh:2,66-69,102-105,117-118`；`settings-hooks.json:30-41` 用 `if: Bash(git commit*)` 收窄。
- 规则：仅查正文硬编码角色属性（身高/体重/年龄 + 数字）与角色卡缺 `name/名字` 字段；命中打印警告，**永远 exit 0**；`git diff --cached -z` 取改动集，避免空格路径问题。
- 落点：**内核校验闸**——gaea 有 git 面板与提交链路，可作为「提交前提醒」轻量接入；注意 gaea 已有更严格的 CI 门禁（`ci.ps1`），别重复造。

### B7 多宿主适配：一个共享核 + N 个薄适配（真增量=部署与降级纪律）
- 证据：同一份核被五端调用——Claude `settings-hooks.json`、Codex `references/codex/hooks/hooks.json:8,22,34,48,62,76`（每个事件都转成 `run-story-hook.* <子命令>`，另给 Windows 用 PowerShell `commandWindows` 启动器）、OpenCode `opencode/plugin.ts`、ZCode `zcode/hooks/hooks.json`、Antigravity `antigravity/hooks/hooks.json`；部署脚本 `story-setup/scripts/{merge-codex-hooks.py,generate-codex-hooks.py,merge-claude-settings.py,deploy-antigravity-skills.py}`。
- 规则（宿主差异，按 gaea 现状最该取的思路）：**Codex 没有 PostToolUse**，故内容网改挂回合末的 Stop，按 git 改动集复扫（`references/codex/hooks/story_codex_hook.py:402-403,1522-1546`）；Codex 的写前拦截用 JSON 输出 `permissionDecision: deny` 而非 `exit 2`（`:1304-1316`）；custom agents 部署后须新开会话才注册、`.codex/` 需被信任并在 `/hooks` 中 review（`references/codex/AGENTS.md.tmpl:35-36`）；部署用 sentinel 记版本并做自检（`hooks/lib/sentinel.sh:6-43`；`session-start.sh:42-53` 缺件告警、`:64-79` `agents_version` 低于 v30 要求重新部署、`:29-37` `.agents-pending-restart` 一次性确认）。
- 落点：**内核校验闸 + 数据资产**——gaea 已有 hook 引擎（`internal/gaea/hook/`）与「事件→命令」装配，缺的是①领域规则、②每条规则在**无该事件/无运行时**时的降级承诺（上游一律「宁可不拦不可误伤」，且会话起点自报降级）。另可借鉴其验证纪律：跨端 parity 由 `scripts/test-prose-net-parity.sh`、`check-hook-regex-sync.sh`、`check-codex-adapter.sh`、`test-agent-permissions.py` 锁死——gaea 的 `ci.ps1` 门禁可挂同类跨实现一致性测试。

## C. 状态追踪（4 条机制）

### C1 唯一权威 + 多派生视图，工具不回读 Markdown（真增量最大）
- 证据：`story-import/references/tracking-transaction.md:3,9-14`（权威 `追踪/_tracking-state.json`；派生 `上下文.md`/`角色状态/{名}.md`/`伏笔.md`/`时间线/{作者真相,读者已知}.md`；派生禁止手改、不作为程序输入）；`tracking_commit.py:1159-1188` `check` 逐字重渲染比对；`:132-142` 续写状态卡固定 7 栏（活跃角色≤6、活跃伏笔≤8、近章≤3）。
- 规则：模型只提交**一份**语义 JSON，禁止分别 Write/Edit 多个追踪文件；不一致时报错并给出修复路径（重交该章 `mode=revision` 整份重建）。
- 落点：**内核（Go）+ 数据契约**——gaea 已有伏笔 lint、角色状态水位契约（`internal/types/character_state.go`）与导入落库（`internal/app/novel_import_ai.go`），但没有「单权威 + 派生视图可校验」的协议；这是最值得学的一条。

### C2 事务原子性与并发语义（真增量）
- 证据：`tracking_commit.py:214-238`（每书 `追踪/.tracking-commit.lock`，`O_CREAT|O_EXCL` 自旋 + 10s 超时 + 写 pid）、`:41`（`TRACKING_SCHEMA_VERSION=4`）、`:180-193`（tempfile + `fsync` + `os.replace` 原子写）、`:1140-1149`（**唯一权威文件最后落盘**；append 重跑要求同章内容完全一致）、`tracking-transaction.md:35,37,39`（`expected_state_revision` 拒绝陈旧事务；写失败可按同一份事务重跑；校验失败按报错改事务）。
- 规则：禁止 dirty/pending 状态机——靠「锁 + revision + 最后落盘 + 幂等重跑」保证事务性；退役条目必须显式声明（漏写不当删除，见 `:994-1008`）。
- 落点：**内核机制**——gaea 若要落「长书导入/续写状态台账」，直接照此模型（对应 gaea 待办里的 `tasks` 表 `KindBookImport`；`internal/gaea/tasks/tasks.go:25-45,153-177` 已有 Kind/Status/Handler 骨架）。

### C3 容量预算是硬门（真增量，防上下文膨胀）
- 证据：`tracking_commit.py:42-47`（逐章记录目标 1536B / 硬限 3072B；上下文 8192/12288B；角色快照 4096/8192B）、`:120`（单字段文本上限 768B）、`:1167-1173`（文件名规范 + 体积上限在任何写入前拒绝）。
- 规则：**超硬限在任何写入前拒绝**（不是写后警告）；超目标是警告；`上下文.md` 的 7 栏就是下一章热上下文预算，而非全量状态容量。
- 落点：**内核校验闸 + 数据资产阈值**——阈值应外置为 JSON 资产，便于按平台/题材调档（与 T1 评审 rubric 同层）。

### C4 角色状态反推与「本节速记」筛选（真增量）
- 证据：`references/character-state-reverse.md:7`（只从已落盘拆书产物反推，不重读原文）、`:23-34`（以最后完整章为截面定 8 字段，各列表≤8 条，只取当前值不塞变化史）、`:72-73`（残稿不提前生效；分批导入则重跑完整导入，不追加历史）；`references/state-tracking.md:9,21`（本节速记只保留「不知道就会写错」的信息）、`:70`（只为会复用的角色建快照）。
- 规则：**作者真相 ≠ 角色已知**（`:31,79`）；未结事项必须有正文证据、未来设计留大纲（`:32,80`）；动态快照与静态人设分离（`设定/角色/*.md` 不动）。
- 落点：**内核纯函数 + 技能资产**——gaea 拆书导入（v4.281.0）已能把章节摘要写回大纲，缺的是「按最后完整章截面生成角色当前状态」这一步；建议作为 bookimport 续刀，与 t5 角色职业/状态机线合并。

## D. gaea 等价物对照（先查再抄）

| 上游机制 | gaea 现状 | 结论 |
|---|---|---|
| 钩子引擎（事件/阻断/超时/信任） | `internal/gaea/hook/hook.go:2-6,34-70`（Pre/PostToolUse、UserPromptSubmit、Stop、SessionStart/End、SubagentStop、PreCompact；exit 2 阻断；`IsBlocking`；`trust.go`） | **已有**，只补领域规则与文案 |
| 后台任务态/进度/强杀 | `internal/gaea/tasks/tasks.go:24-44,153-177`（`Kind` 是 handler 注册键、现有 3 个内置；`Status` 六态含 stopping；Handler/Progress/force kill；DB 持久化） | **已有**，长书导入只需新增 `Kind` |
| 子代理机制 | `internal/gaea/agent/agent.go`、`agent_run.go`；`internal/gaea/cache/spawn.go:99-102` 四种 spawn 模板 | **已有**（缺 writing 域角色卡内容） |
| 技能系统/权限/沙箱 | `internal/gaea/skill/skill.go`、`internal/gaea/permission/`、`internal/gaea/sandbox/` | **已有** |
| 生成后校验 | `internal/app/novel_gate_handler.go:14,27`（RunChapterGate）+ `internal/novelstyle/rewrite.go` 去 AI 味 | **部分**：缺「生成前置契约闸」与「自动复扫」 |
| 写作状态台账 | 无单权威 + 派生视图协议（现为各文件分散） | **真增量** |
| 7 角色提示词资产 | 无 | **真增量**（纯资产，低风险） |

## Top 3 首刀建议

1. **写前大纲契约闸（B1 + A2）**——规模：中。纯函数 + 数据资产，吃 `types.OutlineNode` 必填字段与章节文件存在性；gaea hook 引擎可直接挂 PreToolUse 语义。收益：把「跳纲写作」从习惯问题变成机械拦截。
2. **去 AI 味分层 + 欠账门（B2 + B3）**——规模：小-中。规则表外置为 skill 资产（禁词/句式/标点/门槛四类），落「上一章未清账不许开新章」与「生成后硬信号复扫」。与 T2 同层，可一次发版。
3. **写作状态台账（C1 + C2 + C3）**——规模：中-大。单权威 JSON + 派生视图 + `expected_state_revision` + 容量硬门；先服务拆书导入续写（衔接 v4.281.0 的 `KindBookImport` 任务态），再扩到日更写作。
   备选（若偏好纯资产快赢）：**7 角色卡资产化（A1 + A4）**，规模：小，但需先拍板 gaea 侧「角色卡=技能资产还是内置子代理提示词」。
