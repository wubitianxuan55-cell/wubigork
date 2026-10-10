# DSH 0.2.1 核心域蒸馏 —— 非子代理子系统侦察与刀池（2026-10-10）

> **源**：`clones/deepseek-harness` @ dsh-v0.2.1-alpha.2（真身 GitHub deepseek-ai/deepseek-harness，
> 核实记录见 docs/gaea-dsh-subagent-distill-2026-10.md——子代理板块已单独蒸馏收官，本档扫**其余子系统**）。
> 背景：距 016 蒸馏（v0.1.6，docs/gaea-dsh-016-distill-2026-09.md）跨 0.1.7/0.2.0/0.2.1 三版；
> 官方升级指南三个版本共 9 篇，主题集中在 schedule 退役/账号登录/插件展示/子代理 activation——对 gaea
> 内核有意义的增量主要在子系统语义层，非版本事件层。纪律：取道不取器；GoalCard/MCP/平台化/跨设备在册排除。

## 一、逐域判定

### 压缩域（compaction，284 行文档）

上游把压缩做成能力 seam：log-only 三事件（start/summary/end，**锁的孤儿可检测**——crash 在操作中段留下
无配对 end 的 start，重启后按「陈旧证据」忽略）、tool-result-pruner（范围选择前的确定性头/中/尾修剪，
**可以只修剪不出摘要**推进表面）、summary-error waterfall（摘要请求失败时的恢复监听器瀑布）、
compactNow/compactRegion（手动/区间压缩）、image offload（多模态图片卸载）。

**判定：等价面已足，销号。** gaea 的压缩线已三轮蒸馏（cache 对齐摘要/shrink 硬校验/force 二轮/
锚定 tokenmeter/泄洪），tool-result 修剪对应 gaea 按工具压缩+全局截断管线；锁孤儿检测与 image offload
在 gaea 无多模态上下文压力，留观察不立项。**唯一留档**：`compactNow`（低于压力阈值的空闲会话整理）
若将来做「/compact 手动命令」可参考其「无有用区间不落盘」语义。

### todo（32 行）

Deliberately minimal（content+三态，整表替换，无 id/优先级）。gaea canonical todo 同构且多 activeForm。
**销号。**

### plan 模式（87 行）

016 留池⑦「需前端 intent 渲染配套」**实际已落地**（internal/gaea/agent/plan_mode.go：政策 user 消息注入
不动 system 前缀 + exit_plan_mode 审批/继续计划闭环 + /plan off），与上游同判「软指导不做硬工具门」。
上游独有的「plan/mode 事件日志持久化+恢复折叠」gaea 刻意留二刀（plan_mode.go 头注释在案）。**销号
（持久化二刀维持既定留池口径）。**

### user-questions / ask（307 行）

上游的 provider-neutral 提问 seam：批量问题+稳定 id 回执路由+typed 呈现意图（plan-review 的 approve
点名选项而非依赖顺序）。**gaea ask 工具已同构**（1-4 问题×2-4 选项批量+首选项推荐位）。**销号。**

### session-title（213 行）——**真增量，本批落地**

持久 latest-wins 标题态 + 异步 LLM 起名 + **用户改名钉住**（explicit rename 后自动生成永停）+ 失败降级
fallback 链。gaea 现状：标题只有用户手动注册表（.titles.json），无命名会话显示首条消息截断（DeriveTitle
60 rune）——列表可读性差且无语义。

**落地（P1-A）**：`internal/app/gaea_session_title.go`——TurnDone 槽（与自动做梦同位）触发，首条 user
消息经**办公本地路由**（routeOfficeLocal，数据不出本机）生成 ≤12 字标题写入既有注册表；三语义对齐：
latest-wins（注册表值即真相）/用户改名钉住（只在无标题时生成）/失败自愈（触发条件「无标题+回合成功」
天然退避，无定时器）。刻意偏离：不做 provider/source 归因持久化、不做随对话演进重生成（留观察池）。
零前端改动零新绑定（消费既有 Title 字段与列表渲染）。

### session-reference（224 行）

跨会话 @提及：host 文件发现 + 结构化引用请求 + 快照注入（label 取最新标题）。gaea 的模型侧等价物是
session_search（v4.384 FTS5）；**用户侧 @旧会话 mention 是 UI+wire 中改**——真实需求待现（用户得先
抱怨「没法把上个会话拉进这个话题」）。**留池 P2。**

### slots / ptc-runtime / agent-team / worktrees

slots=Web Client 的 React 组合系统（gaea UI 架构不同，不适用）；ptc-runtime=沙箱化 Node 程序执行
（host bindings 注入——gaea 办公哲学「数据库不是计算器」，bash 已覆盖计算面，**辨伪不取**）；
agent-team=多代理编队（子代理线程已覆盖核心面）；worktrees=git 工作树（办公域无 git 重度场景）。
**全部销号/不取。**

### schedule（0.2.0 退役）与 goal（在册排除）

上游自己把 schedule bundle 退役了——gaea 任务中心自建且在产，无可取；GoalCard 系撤下勿再提。**销号。**

### token-meter / spill / session-query / session-projection

前三者已蒸馏落地（v4.381/v4.379/v4.384 首刀）；session-projection 维持辨伪（gaea checkpoint 等价）。
**销号。**

### 未深扫域（诚实声明）

core/llm-streaming/tools/system-prompt/skills/deliverables/jobs/approval/shell/sandbox/terminal/filesystem/
workspace/conversation/commands：本轮只做了结构级 skim，未逐机制对表。理由：这些域 gaea 均有在产实现且
历史蒸馏覆盖过主干（core spine=框架性差异无刀；approval 的 TimedOut 已从 codex 蒸馏；skills 三判据在
016 档对过）。**后续若某个域出现真实痛点，按域单独立项侦察，不做全量假扫。**

## 二、刀池

| 序 | 刀 | 状态 |
|---|---|---|
| P1-A | session-title 会话自动命名（本地路由+用户改名钉住+失败自愈） | **本批已落地**（gaea_session_title.go + TurnDone 槽 + 2 测试） |
| P2-B | session-reference 跨会话 @提及 | 留池（UI+wire 中改，等真实需求） |
| P2-C | 自动标题随对话演进重生成（latest-wins 全语义） | 观察池（先看 v1 一次定名的用户反馈） |
| P2-D | /compact 手动压缩命令（参考 compactNow 语义） | 留池（等「上下文焦虑」真实反馈） |
| — | plan 模式持久化二刀 | 维持既定留池（plan_mode.go 头注释口径） |
