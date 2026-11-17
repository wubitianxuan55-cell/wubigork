# 资产 13 · 审批七态施工级：规则格式、作用域、超时与追认

> 源码：`protocol/src/protocol.rs:4295`（`ReviewDecision`）、`execpolicy/src/{rule.rs,amend.rs,policy.rs}`、
> `core/src/exec_policy.rs:514`（`add_prefix_rule` 调用点）。
> 对照：gaea `internal/gaea/permission/permission.go`（`Rule{Tool, Subject}` + `ToolName(glob)` 语法）
> 与 `control/controller_approval.go`（五决策 + 会话 glob 规则 + 自动放行账本）。

---

## 一、七态 → 落地动作（施工对照表）

| 决策 | 落盘/内存动作 | gaea 现状 |
|---|---|---|
| `Approved` | 本次放行，不记规则 | ✅ 有（allow_once） |
| `ApprovedForSession` | 写**会话级** granted（进程内），不落盘 | ✅ 有（allow_session） |
| `ApprovedExecpolicyAmendment{proposed_execpolicy_amendment}` | **写规则文件**（见 §二） | ⚠️ 有 persist_allow，但落的是自己的 glob 规则 |
| `ApprovedMcpPolicyAmendment` | 写 MCP 工具策略（跨会话） | ❌ 无 |
| `NetworkPolicyAmendment{network_policy_amendment}` | 写网络规则（host/protocol/decision） | ❌ 无 |
| `Denied{rejection}` | 拒绝 + **理由回喂模型** | ⚠️ 只有 outcome 字符串 |
| `TimedOut` | 无人应答，进挂起态，**不记 deny 账本** | ❌ 与取消混同 |
| `Abort` | 拒绝并终止本轮 | ⚠️ 归一成 denied |

## 二、规则到底怎么存（这块最有实操价值）

codex 的持久化格式是**一行一条规则的文本 DSL**，追加写入策略目录下的 `rules/default.rules`：

```
prefix_rule(pattern=["echo", "Hello, world!"], decision="allow")
prefix_rule(pattern=["ls"], decision="allow")
```

生成逻辑（`blocking_append_allow_prefix_rule`）：把 argv 每个 token 做 JSON 序列化拼成 `pattern=[...]`，
再拼 `decision="allow"`，**追加**到文件末尾（`blocking_append_network_rule` 另有网络规则格式，
带 host / protocol / decision / justification）。

匹配语义（`rule.rs`）：

```rust
pub enum PatternToken { Single(String), Alts(Vec<String>) }
pub struct PrefixPattern { first: Arc<str>, rest: Arc<[PatternToken]> }
// 首 token 固定（策略按首 token 建索引 rules_by_program: MultiMap<String, RuleRef>）
// 其余 token 逐个匹配，支持「多选一」
```

**为什么按首 token 建索引**：命令匹配是热路径，`MultiMap<program, RuleRef>` 让每次只比对同程序下的几条规则，
而不是全表扫描。

## 三、gaea 的规则粒度问题（重要差异）

gaea 现在是 `Rule{Tool, Subject}` + `ToolName(glob)` 语法，subject 是**字符串 glob**。
对 `bash` 这类工具，subject 就是整条命令字符串，于是：

```
Rule{Tool: "bash", Subject: "git status*"}   → 匹配 "git status; rm -rf ~"
```

**glob 是字符串级匹配，无法区分「git status 后面跟的是参数还是另一条命令」**。
codex 的 argv 前缀匹配天然没有这个问题——`prefix_rule(pattern=["git","status"])` 只匹配
「argv 前两个 token 是 git、status」的命令。

**建议**：不动现有 glob 规则（工具级 subject 用它够用），给命令类工具**新增一种规则类型**：

```go
// ArgvRule 按 argv 前缀匹配，用于命令类工具（bash 等）。
type ArgvRule struct {
    First    string      // 首 token，建索引用
    Rest     []ArgToken  // Single / Alts
    Decision Decision
}
```
落盘格式建议沿用 gaea 现有规则文件风格，但语义按 token 比较：
`bash prefix ["git","status"] allow`。**这样「记住这条命令」不会顺手放行整类命令。**

## 四、作用域：会话级 vs 持久级

- `ApprovedForSession`：**只在进程内存**，会话结束即失效，不落盘；
- `ApprovedExecpolicyAmendment`：**落盘**，跨会话生效，作用域是「所有匹配该前缀的命令」；
- MCP 策略与网络策略各自独立的持久化位置，不共用命令规则文件。

gaea 现在把 `allow_session` 与 `persist_allow` 都写进「会话 granted + glob 规则」，
两者差别只是「是否落盘」。**建议把三种作用域显式建模**：`Once` / `Session` / `Persistent`，
避免以后加「项目级」时又要改结构。

## 五、`TimedOut` 与 `Abort` 的区分（语义别混）

- `TimedOut`：**没有人应答**（无人值守 / 通知没人看），不是用户意见。处理是**挂起**，
  恢复后统一审；**不写 deny 账本**，因为它不代表「用户拒绝过」。
- `Abort`：用户明确拒绝**并且**要求本轮什么都别做（区别于 `Denied` 的「这条不行，换个办法」）。

gaea 现在两者都归一成 `denied`，会导致两个后果：无人值守时账本被污染（统计出「用户拒绝了很多次」），
以及模型无法区分「换个办法」还是「停下来」。

## 六、追认（approve denied action）

codex 有 RPC `thread/approveGuardianDeniedAction`，语义是「对已经拒绝的某个动作，重新批准并执行」。
施工要点：
1. 拒绝时**保存动作快照**（工具名 + 参数 + 理由 + 决策来源），带稳定 id；
2. 追认按 id 取快照，**重新走一遍正常闸门**（不能绕过策略与沙箱，只是把「用户拒绝」这个前置条件解除）；
3. 追认本身要记账（谁追认、什么时候、原拒绝理由是什么）。

## 七、gaea 施工方案（三步）

**步一（1 天）**：七态映射 + 拒绝理由回喂
- `approvalOutcome` 增 `OutcomeTimedOut`；`Denied` 携带 `rejection` 文本进下一轮工具结果；
- 事件里区分 `ApprovalDecisionSource`（user / guardian / policy）。

**步二（1–1.5 天）**：作用域三态 + argv 前缀规则
- `GrantScope{Once, Session, Persistent}` 显式化；
- 新增 `ArgvRule` 与匹配函数（首 token 建索引）；
- 「记住这条命令」写 argv 规则而不是 glob。

**步三（1 天）**：追认
- 拒绝时存动作快照（内存 + 会话日志）；`ApproveDenied(id)` 重放一次。

## 八、验收（可断言）

1. 「记住这条命令」后，`git status` 放行，而 `git status; rm -rf /tmp/x` **仍然要审批**（argv 前缀不匹配）；
2. `Session` 作用域的放行在 `/new` 之后**失效**，`Persistent` 作用域仍生效；
3. 无人值守超时 → 事件为 `TimedOut` 且**不产生 deny 账本**；恢复后可统一审；
4. 拒绝理由出现在下一轮模型上下文里（断言字符串包含）；
5. 追认 → 同一动作被执行一次，且账本记录「追认 + 原理由」；
6. `Abort` 之后本轮不再发起任何工具调用（断言无后续 ToolDispatch 事件）。

## 九、不建议照搬

- **Starlark 风格的规则 DSL**（`prefix_rule(...)`）：gaea 已有配置文件体系，用自己的一行式语法即可，
  引入 DSL 等于多一个解析器；
- **`Alts`（多选一 token）**：模式表达能力增强但可读性骤降，先只做 `Single`；
- **网络规则与 MCP 策略的落盘**：等真有出网代理与跨会话 MCP 策略需求时再做。
