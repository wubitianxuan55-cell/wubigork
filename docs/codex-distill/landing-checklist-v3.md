# 蒸馏落地清单 v3（对照 gaea 实现后的修正版）

> v2 的落点与规模是「按目录名 + 旧审计文档」推的。本轮**逐个打开 gaea 实现**核对后，
> 有 5 项被高估（gaea 已有一半以上能力）、1 项方向需修正、3 项维持。
> 依据文件：`internal/gaea/{tool,control,permission,agent,hook,sandbox,trajectory,event}`。
> 结论：**A2 与 B2 仍是性价比最高的两刀；A1 从「中」降到「小」；A3（Guardian）是唯一真正的大件。**

## 修正总表

| 项 | v2 规模 | v3 规模 | 判定依据（gaea 现状） |
|---|---|---|---|
| A1 审批七态 | 中 | **小** | 已有五决策 + 会话规则 + 自动放行账本 |
| A2 execpolicy 三值 | 中 | **小-中** | 已有 Allow/Ask/Deny + glob 规则 + 只读 bash 判定 |
| A3 Guardian 自动评审 | 大 | **大（不变）** | 完全没有 LLM 评审层 |
| A4 工具面预算 | 中 | **中（方向修正）** | 已有隐藏/紧凑 schema，缺预算算法与检索 |
| B1 token-budget 压缩 | 中 | **小-中** | 已有三处压缩入口 + 事件 + PreCompact hook |
| B2 执行环境与双闸门 | 小-中 | **小** | `proc` 无环境注入、无 yield，纯新增 |
| B3 apply_patch 三档匹配 | 小 | **小（不变）** | 只有摘要比对 + CAS，无模糊行匹配 |

---

## A1 审批七态 → 降为「小」

**gaea 已有**（`control/controller_approval.go`）
- 五决策语义：`approvalOutcome()` 把「allow_once / allow_session / persist_allow / deny / 未知」归一（:164-180）；
- 会话级规则：`grantRuleLocked(tool, subject)` + `ruleGrantedLocked` 做 **glob 规则匹配**（:262-280）；
- 权限升级：`RequestPermission(ctx, tool, subject, reason)` 走同一条 `ApprovalRequest` 通道（:208-256）；
- 机器放行账本：`emitAutoApproval` + `event.ApprovalAuto`，明确区分「人被打断」与「机器替你决定」（:283-292）；
- 注意力延迟：`attentionLatency(start)`（:156）。

**真实缺口（三处，都是小改）**
1. **拒绝理由回喂模型**：现在拒绝只产生 outcome 字符串（`event.OutcomeDenied`），没有 `rejection` 文本进模型上下文；
2. **`timed_out` 与 `abort` 分流**：现在取消/超时都归一到同一结果，缺「无人值守挂起待审」这个态；
3. **`approve-after-deny` 追认**：没有「先拒后追认同一条动作」的入口。

**落点**：`control/controller_approval.go`（结构体加 `rejection`/`timeout` 分支）+ `event.ApprovalResolved` 增字段。
**验收修正**：不再要求「七态全上」，改为——① 拒绝必须携带可读理由并进下一轮模型上下文；
② 无人值守下超时进入挂起态、可在恢复后统一审；③ 现有五决策行为逐字节不变。

## A2 execpolicy 三值 → 降为「小-中」

**gaea 已有**
- `permission.Decision` 就是三值 `Allow / Ask / Deny`（`permission.go:17-30`），与我们逆向出的
  codex `Decision{Allow,Prompt,Forbidden}` **同构**，只是命名不同；
- `Policy.Decide(toolName, readOnly, args)` + allow/ask/deny 三组 glob 规则 + `RuleMatches`（:116-158）；
- 只读判定：`isReadOnlyBashSubject` + `containsShellSyntax` + `hasUnsafeReadOnlyArgs`（`bash_readonly.go`）——
  已覆盖「这条 bash 是否只读」的判断；
- 未命中/空配置默认 `Ask`（保守姿态，`ParseDecision` 注释）。

**真实缺口**
1. `Never` 语义：现在没有「绝不弹窗 → 把 Ask 升级为拒绝并把理由回给模型」这一条；
2. `Granular{sandbox_approval, rules}` 两个独立开关（分别管沙箱升级申请 / 策略规则命中）；
3. 危险命令清单（Windows：`DELETE_CMDLETS` 七项 + 强制删除 + PowerShell 分词两层 + 未知 token 保守判否）。

**落点**：`permission`（加 `Never` 归一函数 + 两开关）+ 新增 Windows 危险命令判定文件。
**验收修正**：`Never` 下命中 Ask 规则必须产生「拒绝 + 理由」，且与「用户主动拒绝」在事件上可区分。

## A3 Guardian → 维持「大」，但可切片

**gaea 现状**：只有 `gateApprover`（gate 放行、计划内免问）与 `ApprovalAuto` 账本，
**没有任何 LLM 评审层**——这块是真正的空白。

**切片建议（把「大」拆成三小刀）**
1. 刀一（小）：`guardian` 包骨架 + 证据包结构（用户消息 / 工具输出 / 继承消息**分源**），
   先把 `event.ApprovalAuto` 扩成带 `source/risk/authorization` 三字段的账本；
2. 刀二（中）：评审调用（复用 `internal/ai`）+ 三层预算（输入/请求/会话，参照 `INPUT_TOKEN_MARGIN=256`）；
3. 刀三（中）：决策接线（`None` 回落用户流程，绝不隐式放行）+ 追认入口。
**验收不变**，但刀一就能先落地「授权证据分源」，这本身已经解决提示注入防御的一半问题。

## A4 工具面预算 → 方向修正

**gaea 已有**（`tool/tool.go`、`tool/registrymap.go`）
- `Registry.Hide` / `HideUnlessOnly(names, alternatives)` / `RemovePrefix` / `FilteredSchemas(names)`（:230-358）；
- 每工具可选 `CompactDescriptor{CompactDescription, CompactSchema}`，并有 `compact_drift_test.go` 钉住
  「紧凑 schema 字段必须是全量 schema 的子集」——**这已经是「同一工具两套 schema」的基建**。

**真实缺口**
1. 缺**共享字节预算**：现在是「全量下发」或「名字过滤」，没有 codex 的
   「名字全额优先 + 描述轮次均分 + `...` 截断 + omission 预留」；
2. 缺 `tool_search` 延迟展开（按需把隐藏工具放回来）；
3. 缺占位符语义（不可用工具仍出现在列表里，调用被拒）。

**落点**：`tool/` 新增 `budget.go`（纯函数，可直接照 tools_budget 算法实现）+ `agent/discover_tools.go` 接检索。
**验收修正**：不改注册表结构，只加「schema 文本预算」一层；用 `compact_drift_test.go` 同款守卫保证两套 schema 不漂移。

## B1 token-budget 压缩 → 降为「小-中」

**gaea 已有**（`agent/compact.go`）
- 三个入口：`maybeCompact`（采样后）/ `midTurnMaybeCompact`（回合中）/ `compactIfOver(prompt, trigger)`；
- 规划器：`planCompaction`（保留 ≥25% 窗口给近期轮次）、`pinnedPrefixLen`（钉住前 15% 用户窗口）、
  `tailStart`、`keepUserTurns`、`partitionFold`；
- 溢出恢复：`tryOverflowRecovery`（从 provider 报错里采纳 context window）；
- 事件：`event.CompactionStarted` / `CompactionDone`（已存在，与 codex 的 `ContextCompaction` 条目对齐）；
- hook：`hook.PreCompact` 已存在（"fires just before a compaction pass; stdout 注入为额外摘要指引"）。

**真实缺口**
1. **第三种形态**：codex 的 token-budget 压缩「跳过摘要、直接装新窗口」；gaea 只有摘要式；
2. **PostCompact hook**（gaea 事件表里有 PreCompact，没有 PostCompact）；
3. 压缩后「世界状态重注入」与首轮一致性的校验（有 `prefix_stability_test.go` 可扩展）。

**落点**：`agent/compact.go` 加 `newWindowCompact()` + `hook` 加 `PostCompact`。
**验收修正**：不重写压缩管线，只加一种「不摘要」形态；要求新窗口注入与首轮逐字节一致。

## B2 执行环境与双闸门 → 仍「小」，且最划算

**gaea 现状**：`internal/gaea/proc` 只有进程/管道/信号（`kill_windows.go` 等），
**没有**统一环境注入、没有 `yield_time_ms`/`max_output_tokens` 增量语义。

**落点**：`proc` 新增 `env.go`（10 个固定变量）+ 执行侧时间片回收。
**验收**：同一命令在注入后输出与未注入版的差异仅限颜色/分页/本地化；长命令可按时间片分段返回。

## B3 apply_patch 三档匹配 → 「小」，但价值高

**gaea 现状**：`edit_file` / `multi_edit` / `edit_lines` / `replace` 走「摘要比对（新在旧亡）」+
`expected_revision` CAS，**没有上下文模糊匹配**。

**落点**：`tool/` 新增 `seek.go`（精确 → `trim_end` → `trim` 三档，返回命中行区间与偏移）+ 现有工具复用。
**验收**：文件被无关改动后，按上下文仍能定位；失败要给出「文件 + 期望行 + 最接近的候选行」。

## P2 三项的现状修正

| 项 | gaea 现状 | 修正后的判断 |
|---|---|---|
| 会话索引与策略存档 | `trajectory` 是对 `session.LogEntry`（JSONL）的折叠（`fold.go` 20 KB / `family.go` 11 KB），无 SQLite `threads` 表、无策略存档 | 若要「恢复旧会话用旧策略」，**只需在会话元数据里存两份字符串**（`sandbox_policy`/`approval_mode`），不必上 SQLite |
| MCP 治理 | **已存在**：`internal/gaea/plugin/`（`plugin.go` 24 KB 握手 + `tools/list` + `tools/call`；`transport_stdio.go` / `transport_http.go` 双传输；`mcpConnectTimeout = 30s`；`ssrf.go` 出站校验；`resources.go` / `prompts.go`） | **修正**：不是从零建，「先确认是否存在」作废；真实缺口只有两条——① 超时是**全局常量**而非 per-server 配置（codex 是 `startup_timeout_sec`/`tool_timeout_sec`）；② 工具清单无**版本化增量刷新**（codex 用 `tool_list_revision` + 通知后刷新）。规模：**小** |
| 出网策略 | `sandbox.Spec{Mode:"enforce"}` + seatbelt/bwrap/WSL2 封装，无域名白名单、无出网代理 | 域名白名单是新增，出网代理（credential broker/MITM）明确不做 |

---

## 建议顺序（v3 修正后）

1. **B2 环境注入**（半天，纯新增、零风险、立刻改善输出可复现）；
2. **A2 `Never` 语义 + Windows 危险命令**（1 天，改一个归一函数 + 一个判定文件）；
3. **A1 拒绝理由 + 超时挂起**（1 天，改 approval 三处分支）；
4. **A4 schema 字节预算**（1–1.5 天，纯函数 + 守卫测试）；
5. **B1 token-budget 压缩形态 + PostCompact hook**（1 天）；
6. **A3 Guardian 三刀**（3–4 天，唯一大件，先落「证据分源」）；
7. **B3 三档行匹配**（1 天，可与 4 并行）。

## 本轮的方法论教训

v2 的规模估算错在**只看目录名与文件大小**：`registrymap.go` 21 KB 我以为只是注册表，
打开才发现它带装配入口、空间标签、副作用、证据档、验收谓词五列契约；
`controller_approval.go` 23 KB 我以为只有「允许/记住」两态，实际是五决策 + glob 规则 + 自动放行账本。
**结论：落点与规模必须打开实现再写，不能按命名猜。**
