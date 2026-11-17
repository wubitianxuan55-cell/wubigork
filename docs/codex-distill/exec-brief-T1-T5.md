# 施工单 · codex 0.165.0 → gaea 蒸馏（第一批 T1–T5）

> 交办对象：执行 agent（在 `C:\AI\wubigrok` 内动手）。
> 本文件由逆向方编写，**不要求你读逆向过程**；每条任务都自带「现状 / 改哪里 / 怎么验收 / 禁区」。
> 逆向证据（需要溯源时再看）：`C:\AI\re\codex-0.165.0\docs\hotspot-01..11.md`、
> `distill-checklist-v3.md`。

## 0. 交付纪律（先看，违反等于白做）

1. 先读 `AGENTS.md` 与 `.gaea/AGENTS.md`；涉及构建/装机时另读 `docs/gaea-build-guide.md`。
2. **不 commit、不 push、不 wails build、不装机**，除非用户明确要求。
3. 源码 UTF-8 **无 BOM**；Go 注释用中文、与现有文件风格一致（短句、讲为什么）。
4. 每个任务先跑一次基线测试记下数字，改完再跑，**只许持平或增加**，不许减少用例。
5. 每项独立提交面（一次只做一项），改完在 `.gaea/progress.md` 顶部加一段：
   根因 / 做法 / 门禁数字 / 未做项。
6. 三条硬禁止：不新增外部依赖；不改既有测试的断言意图（可加新用例）；不为通过测试而放宽生产逻辑。

## 1. 五个任务总表

| 编号 | 任务 | 规模 | 用户可感知收益 |
|---|---|---|---|
| T1 | 三档行匹配（编辑工具可靠性） | 小 | 文件被改动后编辑不再整块失败 |
| T2 | 执行环境固定注入（10 个变量） | 小 | 工具输出可比、不卡分页器、缓存前缀不漂 |
| T3 | 审批：拒绝理由回喂 + 超时挂起 | 小 | 无人值守不中断，模型能换法子 |
| T4 | `Never` 语义 + Windows 危险命令 | 小-中 | 夜间任务防误删 |
| T5 | 工具 schema 字节预算 | 中 | 本地小模型首轮上下文下降 |

**建议顺序：T2 → T1 → T3 → T4 → T5**（T2 零风险先落地；T5 需要先量基线）。

---

## T1 · 三档行匹配（apply_patch 的 seek_sequence 同构）

**目标**：编辑工具在目标文件被小改动后，仍能按上下文定位，而不是整块失败。

**现状**（`internal/gaea/boot/boot.go`，工具实现集中在 55 KB 的 boot.go；注册契约见 `internal/gaea/tool/registrymap.go`）
`edit_file` / `multi_edit` / `edit_lines` 走「摘要比对（新在旧亡）」+ `expected_revision` CAS；
**没有上下文逐行定位**，模型给的 old_block 只要与现状不完全一致就失败。

**改动**
1. 新增 `internal/gaea/tool/seek.go`（纯函数、零依赖）：
   ```go
   // FindSequence 在 lines 中自 at 行起找 pat 的最早匹配，三档放宽。
   // 返回命中起始行与档位（0 精确 / 1 尾部空白 / 2 两侧空白），未命中返回 -1。
   func FindSequence(lines []string, pat []string, at int) (idx, tier int)
   ```
   三档判定（逐行比较，命中要求整段连续）：
   - tier0：`lines[i+p] == pat[p]`
   - tier1：`strings.TrimRight(lines[i+p], " \t") == strings.TrimRight(pat[p], " \t")`
   - tier2：`strings.TrimSpace(lines[i+p]) == strings.TrimSpace(pat[p])`
2. `boot.go` 的编辑工具改为：先按 `expected_revision` CAS（保留现有行为），再 `FindSequence` 定位；
   命中 tier>0 时在工具结果里带一句「按第 N 档（忽略行尾/两侧空白）定位到第 X 行」。
3. 未命中时的错误必须可操作：`文件 <path> 第 <at> 行起未找到期望上下文；最接近的候选在第 <x> 行`，
   候选用「首行匹配度最高」的前 3 处给出。

**验收**
- 新增 `tool/seek_test.go`：精确命中 / 行尾空白 / 两侧空白 / 跨段不误命中 / 空 pat / pat 长于文件 六例。
- 新增 `boot` 侧用例：文件被追加无关行后，`edit_file` 仍成功且结果内容正确。
- 既有 `edit_tools_agent_test.go`、`write_scope`、`publishguard` 全绿。
- **禁区**：不许绕过 `write_scope` / `publishguard`；CAS 语义保持（revision 不符仍要拒）。

---

## T2 · 执行环境固定注入（确定性输出）

**目标**：子进程输出不因终端、本地化、分页器而变化；长输出不被 pager 卡死。

**现状**：`internal/gaea/proc/` 只有 `proc.go`(336B) + 各平台 `kill_*` / `hide_*` / `priority_*`；
**没有统一环境注入**，命令继承父进程环境。

**改动**
1. 新增 `internal/gaea/proc/env.go`：
   ```go
   // DeterministicEnv 返回注入给所有工具子进程的固定变量（覆盖同名父变量）。
   func DeterministicEnv() []string
   ```
   固定 10 个（顺序保持，便于比对）：
   ```
   NO_COLOR=1  TERM=dumb  LANG=C.UTF-8  LC_CTYPE=C.UTF-8  LC_ALL=C.UTF-8
   COLORTERM=   PAGER=cat  GIT_PAGER=cat  GH_PAGER=cat  CODEX_CI=1
   ```
   `COLORTERM` 必须是空值（不是删除），因为下游会读「存在且为空」。
2. 在**唯一的命令构造点**合并：`os.Environ()` 打底 → 覆盖上述 10 项 → 交给 `exec.Cmd.Env`。
   若 `internal/gaea/agent/batch_executor.go` 与 boot 侧各有一处构造，抽一个内部函数统一，禁止两处漂移。
3. 允许一个转义阀门（默认关）：`GAEA_EXEC_ENV_RAW=1` 时跳过注入，便于排查。

**验收**
- 新增 `proc/env_test.go`：注入后 `PATH` 保留、`TERM=dumb`、`COLORTERM` 存在且为空。
- 手工验证：同一命令（例如 `git log -3`）注入前后——注入后不进入分页器、输出无 ANSI 转义。
- 既有 bash 工具用例全绿。

**禁区**：不许删父环境（会破坏用户 PATH / 代理）；不许改 shell 解析逻辑。

---

## T3 · 审批：拒绝理由回喂 + 超时挂起

**目标**：拒绝要带上理由进下一轮模型上下文；无人值守时超时进入挂起态而不是卡死。

**现状**（`internal/gaea/control/controller_approval.go`）
- 已有五决策归一（`approvalOutcome`，:164-180）、会话 glob 规则（`grantRuleLocked`/`ruleGrantedLocked`，:262-280）、
  `RequestPermission` 升级通道（:208-256）、`emitAutoApproval` 机器放行账本（:283-292）。
- **缺**：拒绝只有 outcome 字符串，没有理由文本进模型；超时/取消与拒绝混同；无「先拒后追认」。

**改动**
1. `approvalPrompt` 增加 `rejection` 字段；前端拒绝时可带理由（空则用默认「用户拒绝」）。
2. `approvalOutcome` 增加 `OutcomeTimedOut`：`ctx` 超时（不是用户取消）单独归类；
   挂起态写进会话状态，恢复时提示「有 N 条待审」。
3. 拒绝结果写进下一轮工具结果文本：`已拒绝：<理由>`（一句话，不要贴全量参数）。
4. 追认入口：`Controller.ApproveDenied(id)` —— 把已拒动作按原参数重放一次，走正常闸门。

**验收**
- 新增用例：拒绝带理由 → 下一轮消息里出现该理由；超时 → 事件为 TimedOut 且不产生 deny 账本；追认 → 工具被重新执行一次。
- 既有审批用例（`controller_approval`、`request_permission`、`schedule_approval`）全绿。
- **禁区**：不许把超时自动当 allow；不许改五决策的既有语义。

---

## T4 · `Never` 语义 + Windows 危险命令

**目标**：`never` 模式下命中「需批准」的规则要**拒绝并给理由**，而不是静默执行或静默跳过；
Windows 侧对破坏性删除有硬拦截。

**现状**（`internal/gaea/permission/`）
- `Decision` 已是 `Allow/Ask/Deny`（`permission.go:17-30`），`Policy.Decide` 按 allow/ask/deny 三组 glob 判定（:116-158）。
- 未命中默认 `Ask`。
- **缺**：`never` 模式下 `Ask` 的处理；Windows 危险命令清单。

**改动**
1. `permission` 增加模式归一：
   ```go
   // NormalizeAsk 按审批模式把 Ask 归一到最终决策。
   // mode=never → Deny 并返回理由；mode=on-request / untrusted → 保持 Ask。
   func NormalizeAsk(mode string, tool string) (Decision, string)
   ```
   理由文案必须可区分来源，例如：
   `approval required by policy, but approval mode is never`。
2. 新增 `permission/dangerous_windows.go`：
   ```go
   // DangerousWindowsCommand 判定 Windows 下的破坏性命令。
   // 认不出的一律返回 false（保守：不确定不等于危险）。
   func DangerousWindowsCommand(argv []string) (bool, string)
   ```
   判定要点：删除类 cmdlet 集合 `remove-item / ri / rm / del / erase / rd / rmdir`；
   `-Force` 组合判高危；`cmd /c` 先剥引导参数，遇到无法识别的 token 直接返回 false。
3. 把 `DangerousWindowsCommand` 接到执行前闸门（与现有 bash 只读判定同一处），命中即 `Deny` 并给理由。

**验收**
- 新增用例：`never` 模式下 Ask → Deny 且理由含 `never`；`Delete-Item -Force x` → 拦截；未知命令 → 不拦截。
- 既有 `permission` / `bash_readonly` 用例全绿。

---

## T5 · 工具 schema 字节预算

**目标**：工具多时控制首轮 schema 体积，且**同输入同截断**（可缓存、可比对）。

**现状**（`internal/gaea/tool/`）
- `Registry` 已有 `Hide / HideUnlessOnly / RemovePrefix / FilteredSchemas(names)`；
  每个工具可选实现 `CompactDescriptor{CompactDescription, CompactSchema}`，且有 `compact_drift_test.go` 守「紧凑 schema 是子集」。
- **缺**：没有任何**字节预算**——要么全量、要么按名字过滤。

**改动**
1. 新增 `internal/gaea/tool/budget.go`（纯函数）：
   ```go
   // TruncateRows 在共享字节预算内裁描述：名字优先全额保留，
   // 剩余预算在描述间按轮次均分，超出以 "..." 截断，并预留 omission 空间。
   // 返回与 rows 同序的保留描述字节数；nil 表示整行省略，0 表示只留名字。
   func TruncateRows(rows []Row, byteBudget, omissionReserve int) []int
   ```
   算法（照 codex `tools_budget.rs`）：
   - 每个名字行固定成本 `len("- ") + len(name) + len("\n")`，**先全额保留**；
   - 若名字总和 > 预算：从尾部整行省略，直到放下，省略部分用一行 `...还有 N 个工具` 占位（占 omission 预留）；
   - 剩余预算在描述间**按轮次均分**（每轮每行 +1 字节直到用尽），保证公平且确定；
   - 截断按 UTF-8 字符边界，追加 `"..."`。
2. 在 schema 下发处（`FilteredSchemas` 之后）套一层预算；预算值来自配置，默认给一个保守值并写注释说明来源。
3. **默认关闭**（开关 `tools.schema_budget.enabled`），关时行为与现状逐字节等价。

**验收**
- 新增 `tool/budget_test.go`：名字总超预算 / 描述均分 / UTF-8 中文截断不切字符 / 同输入两次调用结果逐字节相同 → 四例。
- 开启后跑一次真实会话，记录首轮 schema token 的**改前/改后**数字（写进 progress.md）。
- 既有 `registrymap` 守卫、`compact_drift` 用例全绿。

---

## 2. 共用门禁

```powershell
# Go 侧（按改动面包）
go build ./...
go vet ./internal/gaea/...
go test ./internal/gaea/tool/... ./internal/gaea/boot/... ./internal/gaea/control/... ./internal/gaea/permission/... ./internal/gaea/agent/... ./internal/gaea/proc/...
```

- 前端未改则不需要 `tsc -b` / `npm run build`；
- **不要**跑 wails build / smoke.ps1（本批五刀都是纯后端 + 单测可覆盖）；
- 若某个包测试基线本来就有红，先记录再动手，别拿它当自己改坏的证据。

## 3. 回报格式（写完贴在会话里）

```
T<n> <任务名>
基线：<包> <N 例通过，M 例失败（既有）>
改动：<文件:行> 一句话
门禁：go build 绿 / go vet 绿 / go test <包> N+? 例全绿
未做 / 风险：<一句>
```

## 4. 与逆向方的接口

- 需要原始证据（codex 的字段顺序、常量、prompt 文本）：`C:\AI\re\codex-0.165.0\docs\`。
- 需要确认「codex 到底怎么做的」再动手：看 `hotspot-09`（exec 策略）、`hotspot-10`（执行与压缩）、
  `hotspot-05`（apply_patch）、`hotspot-06`（工具面预算）。
- **不要**把逆向产物的原文（提示词文本等）直接复制进 gaea 源码；机制照抄、文案自己写。
