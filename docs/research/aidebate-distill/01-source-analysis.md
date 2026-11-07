# aidebate 源码分析（蒸馏输入）

仓库：`clones/aidebate`（orlenko/aidebate，pyproject 记 0.4.7，alpha，Apache-2.0）
阅读方式：只读源码与文档，未安装依赖、未运行测试、未改动仓库。
证据格式：`path:line`（行号对应当前 HEAD 快照）。仓库内路径省略 `src/aidebate/` 前缀时以 `core/`、`web/` 开头。

---

## 一句话判定

**aidebate 是一个"把三个交互式 CLI 塞进 tmux 分屏、用三个文件当回合握手协议、用一条追加式 JSONL 当群聊"的单机玩具级多智能体辩论器；值得蒸馏的是它在"不可靠终端"上硬造可靠性的那几招（功能级 canary、旗标三件套、sidecar 写手、定向路由 + 忙碌合并 + 弃权、静默看门狗、终态事件优先级缓冲、无空格路径不变量），不值得蒸馏的是 tmux/pane 驱动本身、固定 sleep 调参与五阶段里的 roast、以及它那套没有重试也没有超时分级的失败语义。**

一句话补刀：它解决的问题（交互式 REPL 无结构化输出）gaea 用 stream-json 已经绕开了，所以**协议层可抄、传输层不可抄**。

---

## 0. 观察到的整体形状

| 维度 | aidebate | 证据 |
|---|---|---|
| 席位载体 | 一个 tmux session、N+1 个 pane，每个 pane 一个长驻 REPL | `core/session.py:68-92`, `core/session.py:95-157` |
| 回合协议 | prompt.md / answer.md / .done 三文件 + `send-keys` 触发 | `core/turn.py:65-115` |
| 群聊 | `chat.jsonl` 追加，agent 通过随包 `chat-say` 脚本发帖 | `core/crossexam.py:44-74` |
| 编排 | 5 阶段，纯 Python 线程 + 轮询 | `core/debate.py:266-570` |
| 呈现 | FastAPI + SSE + 1s 轮询 pane/answer 文件 | `web/server.py:173-278` |
| 落盘 | `~/.aidebate/sessions/<id>/` 全量 Markdown/JSONL | `core/session.py:18-44` |

整个系统没有事件总线、没有消息中间件、没有状态机库；**可靠性全部来自"文件存在性 + 睡够时间 + 正则读屏"三件事**。这是它最大的价值（极简可移植）也是它最大的债（不可证明）。

---

## 1. 旗标文件协议

### 1.1 完整握手时序（含真实数值）

一个 `run_turn` 的完整动作序列（`core/turn.py:65-115`）：

1. `mkdir -p <phase_dir>`；定义 `{role}.prompt.md` / `{role}.answer.md` / `{role}.done`（`turn.py:72-75`）。
2. 拼 prompt = 正文 + `---` + adapter 的 `answer_instruction`（格式化成 answer/done 绝对路径）（`turn.py:11-18`, `turn.py:77-80`）。
3. 写 prompt 文件（`turn.py:80`）。
4. `time.sleep(3.0)` —— 让 REPL 把上一轮渲染完（注释承认 1s 对 gemini 不够，会吞掉粘贴）（`turn.py:88-91`）。
5. `handle_permission_prompts(duration=2.0)` → `wait_until_ready(8.0)` → 再来一遍 `handle(2.0)` → `wait_until_ready(6.0)`（`turn.py:94-97`）。**故意做两轮**，因为对话框可能在扫描窗口结束后才出现。
6. 发送触发语，只有一行：`Read the file <path> and follow its instructions exactly. The instructions include how to signal completion.`（`turn.py:84-87`, `turn.py:98`）。触发语用 `send_keys(..., literal=True)`，然后睡 `submit_delay`（claude 0.6 / codex 1.0 / gemini 1.2），再单独按 Enter（`core/pane.py:20-33`）。
7. 轮询等待 `.done`：每 **0.4s** 检查一次文件，同时每轮 `capture-pane -p -S -80` 扫权限弹窗；命中则按 `respond`（空串表示只按 Enter）再 Enter，同一 pattern 有 **1.5s 冷却**（`turn.py:21-62`，关键行 `44`, `46`, `50`, `53`, `61`）。
8. 超时（默认 `turn_timeout=900.0`，`debate.py:271`）→ 抛 `TimeoutError`，异常消息里**内嵌最后 80 行 pane 输出**（`turn.py:104-109`）。
9. `.done` 存在但 `answer.md` 不存在 → 抛 `RuntimeError`（`turn.py:111-114`）。
10. 返回 `answer.md` 全文（`turn.py:115`）。

注意：第 4-6 步的 7~21 秒 preamble **不计入 timeout**（timeout 从第 7 步才开始计），所以单回合最坏墙钟 ≈ `turn_timeout + 21s`。

### 1.2 轮询与超时常量全表

| 常量 | 值 | 位置 |
|---|---|---|
| `canary_timeout` | 180s | `debate.py:270` |
| `turn_timeout` | 900s（crossexam 内被压到 `min(turn_timeout, 300.0)`） | `debate.py:271`, `debate.py:468` |
| `crossexam_wallclock` | 300s | `debate.py:272` |
| `crossexam_silence` | 180s | `debate.py:273` |
| `moderator_silence_nudge` | 60s | `crossexam.py:270` |
| done 轮询间隔 | 0.4s | `turn.py:61` |
| 权限重扫冷却 | 1.5s | `turn.py:50` |
| 发送前 sleep | 3.0s | `turn.py:91` |
| canary 最短等待 | `max(max(startup_keys.delay)+2.5, 6.0)` | `turn.py:143` |
| ready 轮询 | 0.25s | `pane.py:61` |
| chat 监听 | 0.75s | `crossexam.py:401` |
| 静默看门狗 | 1.0s | `crossexam.py:408` |
| 结束判定 | 1.0s | `crossexam.py:443` |
| web 采集 | 1.0s | `server.py:266` |
| 阶段收尾 drain | 30s | `crossexam.py:457` |

### 1.3 失败重试：**没有**

全仓库唯一的重试是"权限弹窗反复应答"（`turn.py:47-59`, `pane.py:73-83`）。回合超时 = 该席位在**本阶段**被永久除名：`_run_phase` 收到 error 就 `_record_dropout` + `panes.pop(role)`（`debate.py:420-423`），后续阶段不会再给它机会。没有"重发一次"、没有"降级到更短 prompt"、没有"换引擎重试"。对比 gaea `manager.go:1646-1679` 对空结果有一次自动重试并留痕， aidebate 这块是空白。

### 1.4 什么时候会"假成功"（这是它最脆的地方）

| # | 场景 | 源码事实 | 是否有防护 |
|---|---|---|---|
| A | **stale `.done`** | `run_turn` 只定义路径、**从不 unlink**（`turn.py:72-75`）；而 canary 会先删旧旗标（`turn.py:124-127`）。同一 `turn_dir` 被重入（重跑阶段、手工 smoke、agent 从上下文里重放旧 `touch` 命令）时，`.done` 立刻为真，`run_turn` 会把**上一轮/上一次的 answer** 当成本轮结果返回 | ❌ 无 mtime 校验、无内容校验、无 inode/内容哈希 |
| B | **写了 answer 但没 touch** | 只能等满 900s 超时；没有"answer 已存在但 done 缺失"的旁路检测 | ❌ |
| C | **touch 了但没写 answer** | 有检查（`turn.py:111-114`），抛 `RuntimeError` | ✅（唯一真的存在的检查） |
| D | **done + 空/截断 answer** | `answer_path.read_text()` 直接返回（`turn.py:115`），**不检查空串、不检查长度、不检查 mtime 是否早于 done**；空 answer 会一路流进 verdict prompt 变成 `(no opening)` 之外的真空洞 | ❌ |
| E | **agent 卡在未知权限弹窗** | 只认 adapter 里写死的正则；不匹配的弹窗会把触发语当输入吃掉，然后静默 900s | ❌ 部分（已知弹窗有救，未知弹窗必死） |
| F | **agent 干完活但没退出/还在跑** | done 已存在 → 函数立刻返回；下一回合的文本被粘贴进正在输出的 REPL，被吞或错位 | ⚠️ 只有 `wait_until_ready`，而 claude/codex **没有 `ready_patterns`**，该函数第一行就 `return True`（`pane.py:53-54`）——对这两个主力引擎等于没防护 |
| G | **多个席位共用一个 tmux session** | 每个 pane 独立 send-keys（`session.py:116-137`），不会串台 | ✅ |
| H | **超时异常把整屏终端文本灌进主席 prompt** | `_record_dropout` 存 `str(err)`，而 `TimeoutError` 消息含 last 80 行 pane（`turn.py:106-109`）；该 error 又被原样拼进 verdict prompt（`debate.py:186`） | ❌ 会泄漏终端内容 + 撑爆 token |

结论：**协议只保证"文件出现了"，不保证"文件是本轮的、内容是真的"**。canary 之所以可信，是因为它每次先 `unlink` 再跑（`turn.py:124-127`）；`run_turn` 缺的正是这一行。

---

## 2. tmux 驱动

### 2.1 "就绪"怎么判

三段式，且是**可降级的**：

1. **定时 startup_keys**：进程一起就在后台线程按 `delay` 逐个 `send-keys`（`session.py:145-155`）。claude 1 个 Enter@3.0s，codex 1 个 Enter@3.0s，gemini **2 个** Enter@3.0s/5.0s（`adapters/*.yaml`；理由写在 `gemini.yaml:26-31`：gemini 连续弹两个模态，一个 Enter 只关掉第一个，trust 框会吞掉 canary 文本）。
2. **读屏 ready_patterns**：`capture-pane -p -S -120` + 正则，0.25s 轮询（`pane.py:47-62`）。**只有 gemini 定义了** `Type your message or @path/to/file`（`gemini.yaml:21-25`）；claude/codex 没有 → `wait_until_ready` 立即返回 True（`pane.py:53-54`）。
3. **兜底 sleep**：canary 至少睡 6s（`turn.py:143`），回合至少睡 3s（`turn.py:91`）。注释坦白：claude 冷启动 5s+，4s 时"三块 pane 会有一块错过 canary"（`turn.py:138-142`）。

### 2.2 三个 adapter 字段各自解决的现实问题

- **`submit_delay`**：解决 **bracketed paste 吸收期**。tmux `send-keys -l` 被 TUI 当作粘贴，文本到达后还有一小段时间在"吸收输入"，此刻 Enter 会变成草稿里的换行而不是提交（`pane.py:22-28`）。codex 最严重 → 1.0s；gemini 1.2s；claude 用默认 0.6s（`adapter.py:45`）。
- **`startup_keys`**：解决 **启动期模态对话框**（trust folder / workspace-add / "press enter to continue"）。它在 `time.sleep` 计时器里发，不等读屏，所以是"盲打"，纯经验值。
- **`permission_prompts`**：解决**任务执行中途**的审批弹窗。这是这套系统里唯一有"闭环"味道的东西：task 开始后弹窗可能在 10s 后才出现（gemini 0.38.1 的 WriteFile 确认），一次性扫描窗口必然错过 → 改成"等 done 与扫弹窗交织"（`turn.py:26-38`）。`respond` 允许空串 = 只按 Enter 接受高亮默认项（radio 式对话框）（`pane.py:64-70`）。

### 2.3 多行文本怎么处理：**不处理**

答案是"从不发送多行"。真正的正文永远在文件里，终端里只发一行固定触发语（`turn.py:84-87`）。所有换行、引号、特殊字符都绕开了：路径用单引号包住写进 `answer_instruction`（`claude.yaml:25-28`），`chat-say` 用 Python `repr()` 嵌路径避免 shell 引号地狱（`crossexam.py:45-53`）。没有 bracketed paste 模式、没有 `send-keys -H`、没有 base64 编码。这是**用协议规避终端**，不是"解决终端"。

### 2.4 读屏用来干什么

| 用途 | 位置 |
|---|---|
| 判活/就绪（ready_patterns） | `pane.py:56-62` |
| 发现并应答权限弹窗 | `turn.py:46-59`, `pane.py:73-83` |
| 超时诊断（把屏幕塞进异常） | `turn.py:105`, `turn.py:157` |
| Web 端"看它们思考"的原始窗格流 | `server.py:165-171`, `server.py:223-227` |

**不用于抽取答案**。答案只认 `answer.md`。所以"agent 把答案打在屏幕上但没写文件"= 失败，这是刻意的取舍（`claude.yaml:29`："Do not print the answer to this terminal"）。

### 2.5 读屏的两个反向风险

- `capture-pane -S -80` 带**滚动历史**，弹窗消失后其文本仍在 80 行内，会被当成"仍然存在"再次应答 → 往输入框里注入 `y`+Enter。
- codex 的匹配词是 `Allow` / `Approve`（`codex.yaml:10-11`），而 claude 是 `Allow edits to`。这类宽正则会命中 agent 正常输出的英文句子，把 "y" 打进正在生成的回答里。

---

## 3. 五阶段管线

### 3.1 阶段矩阵

| 阶段 | 输入 | 并发度 | 产物 | 失败语义 |
|---|---|---|---|---|
| canary（前置） | 无 | N+2 线程同时 | `agents/<role>/.canary/{ready,ready.done}` | 主席失败 = **致命**（`debate.py:371-374`）；roastmaster 失败 = 记 dropout 后跳过；辩手失败 = 记 dropout + `panes.pop`，只要剩 ≥1 就继续（`debate.py:385-392`） |
| 1 opening | topic + stance + 其他角色名单 + 群聊说明 | 每存活席位 1 线程，全并行（`phases.py:41-46`） | `phase-1-opening/<role>.{prompt.md,answer.md,done}` | 单席位失败 → dropout，掉到 0 人则 `RuntimeError` 中止（`debate.py:429-430`） |
| 2 crossexam | openings 全文 + stance + topic + chat 路径 | 每角色 1 worker 线程 + 1 chat watcher + 1 silence watcher | `phase-2-crossexam/<role>/turn-00N/{prompt.md,answer.md,done,error.log}` + 共享 `chat.jsonl` | **单回合失败被吞掉**，只写 `error.log`，不记 dropout，不中止（`crossexam.py:302-307`） |
| 3 rebuttal | 对手 opening 全文 + chat 全文 + 自己 stance | 同阶段 1 | `phase-3-rebuttal/<role>.*` | 同阶段 1 |
| 4 verdict | 全部 opening + 全部 rebuttal + chat 全文 + dropout 清单 | 单线程 | `phase-4-verdict/moderator.*` + `verdict.md` | **未捕获**：超时直接冒泡出 `run_debate` → 整场 error，无裁决（`debate.py:498-513`） |
| 5 roast | opening + rebuttal + verdict + chat + dropout | 单线程 | `phase-5-roast/roastmaster.*` + `roast.md` | 非致命，记 dropout 后继续（`debate.py:552-554`） |

阶段 2 与 3 都有"存活席位 <2 就跳过"的门（`debate.py:451`, `debate.py:480`），并 emit `phase_skipped`。

### 3.2 cross-examination 怎么决定谁说话、说几轮、何时停

模型写在 docstring 里（`crossexam.py:1-23`），实现是：

1. **每个角色一个 worker 线程 + 一个 `queue.Queue`**（`crossexam.py:250-256`, `313`）。
2. **开局广播 seed**：给所有角色队列塞同一个哨兵 `"__SEED__"`（`crossexam.py:247`, `431-432`）。辩手的 seed 是"针对每个对手最强的具体主张，提 2 个尖锐问题，用 `--to` 指名"（`crossexam.py:129-142`）；主席的 seed 是"读 openings，向全组提 1 个 framing 问题"（`crossexam.py:157-170`）。
3. **一句话决定一次发言**：chat watcher 每 0.75s 全量重读 `chat.jsonl`，对新行按 `to` 路由——`"*"` 或包含自己角色才入队，跳过发送者自己（`crossexam.py:378-401`, `_is_addressee` 97-99）。
4. **忙碌合并（batching）**：worker 取到第一条后把队列里现存的全 drain 成一个 `batch`（`crossexam.py:319-327`），一次 prompt 里列出"自你上轮以来针对你的全部消息"（`crossexam.py:181-188`）。**这就是它的背压机制**：没人排队就不会重复触发。
5. **可以不说话**：prompt 明确允许 `(pass)`——"若没什么尖锐的，写 `(pass)` 到 answer 文件并 touch done，不发言。沉默优于填充"（`crossexam.py:194-198`）。
6. **主席只在三种情况下介入**：有人回避直接提问 / 原地打转 / 关键假设无人挑战（`crossexam.py:158-163`）。另有 stall 分支：静默 60s 就给主席塞 `{"__stall__": True}`，走 `_stall_prompt_moderator`——"挑最近最弱且未被回答的点，捅一下"（`crossexam.py:405-420`, `223-239`）。
7. **终止是 OR**：`wallclock`（300s）**或** `silence_timeout`（180s）任一到达即 `break`（`crossexam.py:441-452`），然后 `stop.set()` + 最多等 30s 让在飞回合收尾（`crossexam.py:454-461`）。

**轮次上限/预算：没有。** 只有墙钟与静默两个时间闸。每角色 `turn_n` 单调递增（`crossexam.py:297-299`），队列无上限，单回合 timeout 300s（`debate.py:468`）。理论上一个话痨席位可以把别人刷到队列爆炸；实际被 300s 墙钟兜住。

### 3.3 chat.jsonl 的写入：追加，无锁

`chat-say` 是每个 agent cwd 里一份的可执行 Python 脚本（`crossexam.py:44-74`），核心是：

```python
with open(CHAT, "a") as f:
    f.write(json.dumps(msg, ensure_ascii=False) + "\n")
```

（`crossexam.py:63-64`）

- 写者是 **N 个独立进程**（每个 agent 各跑一次），不是同一进程的线程。
- **没有 flock、没有 fsync、没有临时文件 rename**（全仓库只有 canary 用 `unlink`，没有任何锁原语；`grep flock|LOCK_EX|fsync|os.replace` 零命中）。
- 靠的是 POSIX `O_APPEND` 对**单次小 write** 的原子性。单条消息 ≤ 几百字节，实际很难交错；但这是**平台契约，不是代码契约**——一旦某 agent 发超长消息、或文件系统不是本地 POSIX，就可能撕裂成半行 JSON。
- 后果可控：读侧 `_read_chat` 遇到坏行直接 `continue`（`crossexam.py:90-93`），`_format_chat_transcript` 同样跳过（`debate.py:106-109`）。**坏行 = 静默丢消息**，没有任何计数或告警。

对比：`events.jsonl` 有 `threading.Lock()`（`events.py:22`, `events.py:29`），因为它是同进程多线程写；`chat.jsonl` 跨进程，反而没有锁。

---

## 4. 交叉质询的提示词工程

结论先行：**比 gaea 现有 prompt 弱一档**。它强制"具体"，但没有强制"可定位"。骨架如下（提炼，不贴原文）：

| 位置 | 角色 | 结构约束 |
|---|---|---|
| `debate.py:115-127` opening | 辩手 | 主题 / 我的角色 / 我的立场 / 其他辩手名单 / 任务：~250 词，1 个论点 + 3–5 条最强论证 + **具体证据** + 主动承认最强的预期反驳 / 群聊说明 |
| `crossexam.py:129-142` seed | 辩手 | 对手 opening 全文 / 任务：提 **2 个尖锐、具体**的问题，攻击其 opening 中**最弱的具体主张**；**每个问题必须 `--to` 指名一个角色**；每题 1–2 句；"要尖锐，不要修辞" |
| `crossexam.py:157-170` seed | 主席 | 主题 / 辩手与立场清单 / 全部 opening / 任务：**只在三种情况介入**（回避、绕圈、关键假设无人挑战）；首轮只提 **1 个** framing 问题 |
| `crossexam.py:173-220` nudge | 全员 | **逐条转述"发给你的消息"（from / to / text 原文）** / 最近 30 条 chat / 任务：**一条** ≤2 句的消息；要么直接回答，要么一句话说明这是坏问题并改道；否则 `(pass)` |
| `crossexam.py:223-239` stall | 主席 | 无新消息了 / 最近 30 条 / 任务：挑最弱未答点，捅一下；或者 `(pass)` |
| `debate.py:130-164` rebuttal | 辩手 | 每个对手的 opening 全文 / chat 全文 / 任务：**逐个对手**正面接其最强主张；对抗型就挖最伤人的漏洞（事实错、逻辑断、证据弱、未回应的反例），协作型就锐化 + 承认合理处 + 推进仍有争议处；**必须引用交叉质询中的具体时刻**（被回避的问题、某次让步、opening 与 chat 的自相矛盾）；结尾 1–2 句预答；~300 词 |

**它做对了什么**（值得抄的"约束句式"）：

1. **"每个问题必须点名一个角色"** —— 把"泛泛而谈"变成结构性不可能：`--to <role>` 是**机器可校验的落点**（`_is_addressee` 只认 role），发不出指名消息就等于没发。
2. **"攻击最弱的具体主张"** + "1–2 句" + "要尖锐不要修辞" —— 用长度上限和靶点限定压制空话。
3. **"必须引用交叉质询的具体时刻"** 并给了三个可识别的类别（回避的问题 / 让步 / 自相矛盾）—— 这是它唯一接近"引用原话"的要求，但仍然只是**要求描述**，不是要求**摘引**。
4. **`(pass)` 弃权** —— 明确"沉默优于填充"，把"必须发言"从协议里摘掉，这是 gaea 固定轮次模型里没有的自由度。
5. **nudge 里逐条原文转述**（`crossexam.py:183-186`）—— 不是摘要，是把 `text` 原样列出来，减少转述失真。

**它做错了什么**（gaea 不要学）：

- 没有任何**贡献 ID / 主张 ID**，所以"引用某人的某个论点"无法回溯；对比 gaea `prompts.go:559-561` 要求"贡献 ID + 不超过一句的原话摘引"，并区分【工具核验】/【他人核验】/【未核验】三级证据——aidebate 完全没有这层。
- 没有"被挑战命题必须能在目标发言中定位"的约束（gaea `prompts.go:107`）。
- 没有字数/轮次预算，只有墙钟；`~250 词` / `~300 词` 是软约束，无校验。
- chat 全文无上限塞进 rebuttal/verdict prompt（`debate.py:95-112` 的 `limit` 默认 `None`，调用处 `debate.py:475`, `497` 都没传），chat 越长 prompt 越爆。

---

## 5. 裁决（verdict）

**主席看到什么**（`debate.py:167-216`）：

- 每个**存活**席位的 `stance` + opening 全文 + rebuttal 全文（缺席者显示 `(no opening)`）；
- **dropout 清单**：`- <role> (<agent>) — dropped during <phase>: <error>`，并附一段裁决纪律："只评判实际提交的辩手，**不要**因为缺席而扣分，**不要**替缺席者编造立场"（`debate.py:183-195`）；
- **完整** chat transcript（围栏代码块）+ chat 文件在磁盘上的绝对路径（`debate.py:202-204`）；
- **看不到**：pane 原文、events.jsonl、各席位的 crossexam 回合 prompt/answer、token 用量。主席的信息面 = 三个文本产物 + 一条群聊。

**输出格式约束**：三段式编号清单——1. Winner（哪一方更强、为什么，1 段）；2. Scorecard（每席位一句最强贡献 + 一句最大弱点）；3. Open questions（2–3 个未决问题）。并明确"要具体、要敢选赢家，除非真的打平，否则别说'双方都有道理'"（`debate.py:205-215`）。

**有没有"无法裁决/平局"路径**：只有一句提示允许平局（`debate.py:213-215`），**没有结构化的 tie 字段、没有 JSON schema、没有失败兜底**。如果主席超时或崩溃，异常直接冒泡出 `run_debate`（`debate.py:498-511`），整场标记 error，`verdict.md` 根本不存在——**没有"部分裁决"或"程序性裁决"降级路径**。

**落盘与渲染**：`verdict_text = run_turn(...)` → 原文写 `verdict.md`（`debate.py:512-513`）→ manifest 写相对路径（`debate.py:565`）→ emit `verdict_ready`（`debate.py:514`）。前端 `marked.parse` 渲染（`app.js:27-34`, `app.js:714-716`）。服务端 poller 也会在文件一出现时就 emit（`server.py:247-265`），所以裁决不必等 roast 结束就能看。

---

## 6. 会话归档

### 6.1 目录契约（实测拼装，README 的版本已过期）

```
~/.aidebate/sessions/<YYYY-MM-DD-HHMMSS-mmm>/
├── session.json                 # manifest
├── chat.jsonl                   # 追加式群聊，创建时 touch
├── events.jsonl                 # 叙事事件（带锁追加）
├── verdict.md                   # 阶段 4
├── roast.md                     # 阶段 5（可选）
├── agents/<role>/
│   ├── chat-say                 # 可执行 sidecar，chmod 0755
│   ├── CLAUDE.md                # 仅 roastmaster 有（人格旁加载）
│   └── .canary/{ready,ready.done}
├── phase-1-opening/<role>.{prompt.md,answer.md,done}
├── phase-2-crossexam/<role>/turn-00N/{prompt.md,answer.md,done,error.log}
├── phase-3-rebuttal/<role>.{prompt.md,answer.md,done}
├── phase-4-verdict/moderator.{prompt.md,answer.md,done}
└── phase-5-roast/roastmaster.{prompt.md,answer.md,done}
```

证据：`session.py:68-72`（root + chat touch）、`session.py:106-107`（agents/<role>）、`debate.py:444`, `484`, `495`, `528`（阶段目录名）、`crossexam.py:276`, `299`（`turn-{n:03d}`）、`debate.py:335-337`（roastmaster 的 CLAUDE.md）、`crossexam.py:69-74`（chat-say 落位）。

注意两处**编号错位**：代码里 rebuttal 是 `phase-3-rebuttal`、verdict 是 `phase-4-verdict`，而 README 写的是 `phase-2-rebuttal` / `phase-3-verdict`（`README.md:89-95`, `README.md:107-113`）。**以代码为准**：目录里根本没有 `phase-2-rebuttal` 这种东西。

### 6.2 session.json 字段

初次写入（`debate.py:284-296`）：`session_id` / `topic` / `moderator_agent` / `sides[{role,agent,stance}]` / `created_at` / `status:"running"` / `verdict_path:null` / `completed_at:null` / `dropouts:[]`。
过程中追加：`dropouts[{role,agent,phase,error,at}]`（每次 dropout 立即回写整份 manifest，`debate.py:316-317`）。
结束时追加：`status:"done"` / `verdict_path` / `completed_at`（`debate.py:564-567`）；roast 相关 `roast_path` / `roast_enabled`（`debate.py:520-521`, `548`）。

**状态机只有两个持久态**：`running` → `done`。**失败不落盘**：`run_debate` 抛异常时 manifest 永远停在 `running`（`server.py:313-327` 只改内存 `state.status="error"` 并补一条 `debate_completed(status=error)` 事件，从不碰 `session.json`）。所以磁盘上的"失败"只能靠 `events.jsonl` 的最后一条事件 + web 层推断。这是一个**真实的契约缺口**（第 7.4 节的 `stale` 逻辑就是给这个缺口打的补丁）。

### 6.3 恢复 / 重连

- **刷新页面（服务端还活着）**：`app.js:317-347` 路由 `#/live/<sid>` → 重新 `EventSource` → 服务端把 `narrative_events` 全量 + `events` 中非 narrative 的积压先灌一遍再转直播（`server.py:594-608`）。因为订阅注册与快照在同一把锁内（`server.py:594-597`），不会漏也不会重（理论上）。
- **服务端重启过**：`SESSIONS` 是纯内存（`server.py:156`），`/api/debates/{sid}` 404 → 前端 `alert` + 回首页（`app.js:334-337`）；只能进归档视图读磁盘。
- **归档视图**：`GET /api/sessions/{sid}` 一次性返回 manifest / verdict / roast / chat / events / 各阶段 answer（`server.py:395-434`），前端渲染 + 可"Clone this setup"重开同配置（`app.js:472-477`）。
- **tmux 侧无恢复**：`_run_debate_thread` 的 `finally` 里 `session.kill()`（`server.py:332-335`），跑完即销毁 pane；服务端崩了则 tmux session 变成孤儿，只能靠 `debate kill-all` 清（`cli.py:215-230`）。

---

## 7. Web 层

### 7.1 SSE 还是轮询：**两层都有**

- **服务端 → 浏览器**：SSE（`server.py:583-625`），`data:` 帧，15s 心跳注释（`server.py:611-615`）。
- **文件/终端 → 服务端**：**1s 轮询**。`_poll_session` 每轮做四件事：drain `events.jsonl`（按字节 offset 增量，`server.py:198-218`）、`capture-pane` 比对文本变化、glob 所有 `phase-*/*.answer.md` 比对内容、比对 `verdict.md`/`roast.md`（`server.py:220-266`）。

### 7.2 前端怎么知道"某席思考/完成/失败"

- **思考中**：没有显式状态。只有一个 `status: running`（`server.py:292`）+ 该 pane 的 `pane` 事件在不断刷屏（`app.js:706-713`）。没有 per-seat 状态字段。
- **完成**：`participant_completed_phase` 叙事事件（`debate.py:426`），前端渲染成 "✓ <role> finished <phase>"（`app.js:160-164`）。
- **失败**：`dropout` 叙事事件（`debate.py:318`），前端标红 "⚠ role dropped during phase: error"（`app.js:138-143`）；归档页给该席位加 `dropped @ phase` 徽章（`app.js:419-436`）。
- 也就是说：**席位状态是从"叙事事件流"里读出来的，不是一份状态快照**。刷新期间如果事件丢了（缓冲区裁剪），席位状态就丢了。这是它和 gaea 的 `Participant.Status` + 状态转移表（`observe.go:44-82`）最本质的差距。

### 7.3 多会话并发怎么隔离

- 每个会话一个 `SessionState`（内存）+ 一个 `debate-<sid>` tmux session + 一个 `sessions/<sid>/` 目录（`server.py:95-155`, `session.py:68-92`）。
- 每个 SSE 订阅者一个独立 `EventBuffer`（`server.py:589`），会话内 fan-out（`server.py:148-153`）。
- **不隔离也没人管的地方**：`SESSIONS` 永不清理（跑完也不 pop），且同一个 state 会同时注册在两个 key 下（预分配 id + 真实 id，`server.py:502-527`），等于双重泄漏；`EventBuffer` 满了按"最老的非优先事件"驱逐（`server.py:61-73`），优先集合是 `{verdict, roast, status, error, debate_completed}`（`server.py:54`）——这条设计是对的（终态永不被 pane 噪声挤掉），应该抄。
- 并发上限：无。N 个辩论 = N×(席位数+2) 个 tmux pane + N 组线程。

### 7.4 "由磁盘说 running 但没人活着"的降级

`GET /api/sessions` 里做了一次判据：manifest 说 `running` 但 `SESSIONS` 里没有对应的活跃 state → 报 `status:"stale"`（`server.py:361-388`，实现 387-388）。前端把 `stale` 渲染成 error 色（`app.js:357-361`）。这是**用"活体登记表"去校验"持久化声明"**的思路，值得抄——但注意 gaea 已有 `RecoverInterrupted`（`manager.go:193`）+ `StatusInterrupted`，属于已覆盖。

### 7.5 停止/中断入口与语义

**没有整场停止入口。** 全仓库搜索 `cancel|abort|stop` 只命中 crossexam 内部的 `threading.Event`（`crossexam.py:291`）。现有能力只有：

- `POST /api/debates/{sid}/panes/{role}/keys`，body 支持 `{text, enter}` 或 `{key}`（`server.py:552-580`）——本质是**向某个 pane 注入按键**，UI 上暴露为每个 pane 下方一个输入框 + `⏎` + `^C` 按钮（`app.js:595-619`）。
- 语义极其含糊：`^C` 只会打断当前 REPL 的生成，**编排线程完全不知情**，它会继续等那个永远不来的 `.done`，直到 900s 超时；被 Ctrl-C 的那个席位随后会被记成 dropout。
- `debate kill-all` 杀所有 `debate-*` tmux session（`cli.py:215-230`），但**杀不掉正在跑的 Python 线程**，只能等它超时后自己 `kill` 一个已死的 tmux。
- `doc/proposed-features.md:1-23` 自己承认：暂停/注入/踢席位都**没做**，唯一控制面是单向 SSE。

---

## 8. Adapter 抽象

### 8.1 新增一个 CLI 需要给的字段

| 字段 | 必需? | 缺省 | 硬约束 / 经验值 | 证据 |
|---|---|---|---|---|
| `name` | **必需** | — | `data["name"]` 直接下标，缺则 KeyError | `adapter.py:39` |
| `cmd` | **必需** | — | 必需；支持 `{session_root}` / `{agent_cwd}` 占位符 | `adapter.py:40`, `session.py:133-137` |
| `answer_instruction` | **必需** | — | 必需；**必须含 `{answer_path}` 与 `{done_path}`**，且 done 路径要自带单引号（路径可能含空格） | `adapter.py:51`, `turn.py:14-17`, `test_adapter.py:34-35` |
| `submit_key` | 可选 | `"Enter"` | **死字段**：加载了但没人读，`send_enter` 硬编码 `Enter` | `adapter.py:41` vs `pane.py:35-37` |
| `submit_delay` | 可选 | `0.6` | 经验值，按 TUI 的粘贴吸收期调 | `adapter.py:45` |
| `permission_prompts[{match,respond}]` | 可选 | `[]` | 经验值 + 正则；`respond:""` = 只按 Enter | `adapter.py:46-49` |
| `ready_patterns[]` | 可选 | `[]` | 缺省 = **放弃就绪检测，立即返回 ready** | `adapter.py:50`, `pane.py:53-54` |
| `startup_keys[{delay,key}]` | 可选 | `[]` | 纯经验值，盲打；canary 的等待时长由 `max(delay)+2.5` 反推 | `adapter.py:55-58`, `turn.py:137-143` |

启动时 `validate_all_adapters()` 全量试加载，坏的只告警不中止（`adapter.py:69-84`, `cli.py:38-57`）——这个"坏配置不阻断启动、只在使用时炸"的策略值得抄。

### 8.2 可移植性边界（它默默假设了什么）

1. **假设 agent 会遵守"写文件"指令**。整个协议的唯一可靠性来源。没有校验、没有回退到"从屏幕抽答案"。agent 一旦"我直接回答你"（把答案打在终端），这一回合就是 900s 浪费。
2. **假设 agent 有 shell 工具**（要能 `touch`）。如果 CLI 是纯对话（无工具），协议根本不成立。
3. **假设一个 pane 同一时刻只有一个回合**。这个假设在多处被破坏（见 §9-F1）。
4. **假设终端输出是可供正则匹配的稳定英文**。对 i18n、主题、ANSI 重绘、宽字符都很脆。
5. **假设单机、单用户、无鉴权**。服务默认绑 `127.0.0.1:8765`（`cli.py:317-318`），但 pane 注入接口没有任何鉴权（`server.py:552`），一旦暴露即可远程往 agent 终端打字。
6. **假设文件路径不含空格**。这条被写成了系统级不变量（`session.py:20-33`，理由：agent 会发出大量未加引号的 shell 命令；`test_session_paths.py:42-48` 甚至断言路径里不能有空格）。
7. **没有 token/成本核算、没有模型参数、没有输出 schema**。adapter 只是"怎么启动 + 怎么收尾"，不是"怎么理解"。

结论：这个抽象**只能覆盖"长驻交互式 REPL + 有文件系统工具 + 英文 UI"** 这一类 CLI。它是 gaea 的反面：gaea 用 `stream-json` 把"理解"做进了协议，aidebate 用文件把"理解"推给了 agent。

---

## 9. 反面教材：agent 不守规矩时的失效模式清单

按危险度排序。每一条都是**当前代码会真实发生**的，不是假设。

**F1（最危险）跨阶段共用 pane，两个回合同时打字。**
crossexam 结束时只等 30s 让在飞回合收尾，等不到就打个 warning 直接返回（`crossexam.py:456-461`）；紧接着阶段 3 立刻对**同一批 pane** 发新 prompt（`debate.py:480-486`）。若某席位在 crossexam 里卡了 200s，它的 turn 线程还在 `run_turn` 里等 `.done`，而 rebuttal 线程已经往同一个 REPL 粘了第二段触发语 → 两段文本在同一个输入框里交错，两个 `.done` 互相冒充，answer 写错目录。这是**协议层没有"回合所有权"锁**的必然结果。

**F2 stale 旗标假成功。**
`run_turn` 不 unlink（`turn.py:72-75`）。只要目录被重入（手工重跑、未来做 resume、agent 从上下文里重放旧的 `touch` 命令），`.done` 立即为真，返回旧 answer，整场无人察觉。canary 有清场（`turn.py:124-127`），回合没有——这是**同一份协议里两套卫生标准**。

**F3 空/截断 answer 被当成功。**
`read_text()` 直出（`turn.py:115`），无空检查、无 mtime 比较、无长度下限。agent 先 `touch` 后写（或写一半被 Ctrl-C）→ 空 opening 进入 rebuttal/verdict prompt，主席对着 `(no opening)` 之外的一片空白做裁决。

**F4 就绪误判（对 claude/codex 等于没有就绪判据）。**
`wait_until_ready` 在 `ready_patterns` 为空时**第一行就 return True**（`pane.py:53-54`），claude.yaml / codex.yaml 都没有 ready_patterns。于是"等待就绪"退化为 `sleep 3.0`（`turn.py:91`）+ 两次固定 2s 扫描。CLI 冷启动或长输出时，触发语会被粘进正在渲染的界面里，**静默丢失**，然后 900s 超时。

**F5 宽正则权限应答把按键打进正文。**
codex 的 `Allow` / `Approve`（`codex.yaml:10-11`）、claude 的 `Allow edits to`（`claude.yaml:10`）是对**整个 80 行 scrollback** 做搜索（`pane.py:74`）。agent 正常输出里出现 "Allow" 就会被判定为弹窗，往输入框注入 `y` + Enter，把正在生成的回答打断并提交成垃圾。同时 `capture-pane -S -80` 包含历史，消失的弹窗会被反复"应答"。

**F6 超时异常把终端内容灌进主席 prompt。**
`TimeoutError` 消息含最后 80 行 pane（`turn.py:106-109`）→ 存进 `dropouts[].error`（`debate.py:313`）→ 原样拼进 verdict prompt（`debate.py:186`）。一个卡住的席位会把整屏终端（含它读过的文件片段、之前的 prompt 回显）送进主席上下文，既泄密又烧 token。

**F7 无重试 + 无降级，一次抖动 = 永久减员。**
超时即 dropout（`debate.py:420-423`），不重发、不换引擎、不降 prompt 长度。两个席位里死一个，`active_sides` 就只剩 1，阶段 2/3 直接跳过（`debate.py:451`, `480`），整场退化成"独角戏 + 主席裁决"，而 manifest 只会记一条 dropout。crossexam 里的失败更糟：**只写 `error.log`，连 dropout 都不记**（`crossexam.py:302-307`）——主席完全不知道有人在质询阶段哑火。

**F8 静默计时器从"阶段开始"起算，不是从"上一回合结束"起算。**
`last_message_ts` 在 `run_crossexam` 入口初始化（`crossexam.py:375`），只在**新 chat 消息**到达时重置（`crossexam.py:385-386`）。两个席位各跑 90s 的 seed 回合时，时钟一直在走；若 seed 后无人发帖，180s 静默从函数入口算起 → **可能只跑了一轮就被判"安静"结束**。而且"安静"不等于"空闲"：有人正在写 300 字长回答时，chat 也可能安静 180s。

**F9 群聊无锁、无去重、无上限。**
跨进程 append（`crossexam.py:63-64`）无 flock；读侧坏行静默丢弃（`crossexam.py:90-93`）；注释声称"按 (ts, from, text) 去重"（`crossexam.py:351-353`）**但代码里没有任何去重**；rebuttal/verdict prompt 塞入**无长度上限**的全文 chat（`debate.py:95-112`, 调用处 475/497 不传 limit）。

**F10 失败态不落盘 + 内存只增不减。**
manifest 永远停在 `running`（只有 `debate.py:290` 与 `564` 两处写 status）；`SESSIONS` 永不清理且一个 state 注册两个 key（`server.py:502-527`）；跑完 `tmux kill` 后内存里的 `SessionState` 仍持有全部 `events`（上限 500）+ `narrative_events`（无上限）。

**F11 注释承诺了不存在的语义（会误导后来的维护者）。**
`crossexam.py:346-349` 说 seed 同批到达的普通消息"会留到下次 drain 再处理"——实际 `batch` 是局部 list，`continue` 后整批丢弃；`crossexam.py:351-353` 说"按 (ts, from, text) 去重"——没有实现。这类注释比 bug 更难查。

**F12 权限弹窗扫描的两个实现不一致。**
`handle_permission_prompts` 一轮内会对**所有**命中的 pattern 逐个应答（`pane.py:75-82`，无 break）；`_wait_for_done_dismissing_prompts` 命中第一个就 `break` 重扫（`turn.py:59`）。gemini 的 `Do you trust the following` 同时命中 `gemini.yaml:11` 与 `gemini.yaml:12` 两条 → 发送前扫描会连按两次 Enter（第二次可能把空输入提交出去）。

---

## 候选蒸馏点

[01] 功能级 canary 握手 — 开赛前让每个席位做一次"写一个词 + 落完成旗标"的最小往返，用真实 I/O 证明"这个席位现在能用"，失败即判不可用而不是等到正式回合超时 — `core/turn.py:118-162`，调度与致命性分级 `core/debate.py:349-397` — **高**（gaea 目前只有 `--version` 探活，`agents.go:237`, `agents.go:577`，证明不了鉴权/写盘/完成信号）

[02] 回合三件套 prompt/answer/done + 逐回合目录 — 每回合三个文件固定命名、固定落位，prompt 是自解释的（含"写哪里、怎么宣布完成"），产物天然可审计 — `core/turn.py:65-115`，`core/crossexam.py:299` — 中（gaea 已有 contribution + trace，但缺"本回合到底给模型看了什么"的原文快照）

[03] 完成判据双检 — 旗标存在 + 产物存在，两者缺一即失败，而不是只看旗标 — `core/turn.py:111-114` — 中（补的是"agent 报完成但没交付"这一档）

[04] 随包 sidecar 写手（chat-say） — 把"生成合法结构化消息"的责任从 LLM 手里拿走，给它一条命令，路径与角色用 `repr()` 硬编码进脚本，绕开 shell 引号 — `core/crossexam.py:44-74` — **高**（gaea 让模型直接输出结构化 JSON，格式错误全靠解析器兜；sidecar 是把格式责任外置的最便宜办法）

[05] 定向/广播路由 + 每席位队列 + 忙碌合并 — 消息带 `to`（角色列表或 `*`），只有被点名者入队；忙碌时新消息合并进下一次发言的"自你上轮以来针对你的消息"清单；空闲则不触发 — `core/crossexam.py:97-99`, `313-371`, `181-188` — **高**（gaea 是固定轮次全员发言，没有"被点名才回应"和"合并积压"这两种背压）

[06] 弃权语义 `(pass)` — 明确允许"本轮不说话"，且协议上可区分"说了"和"选择不说"（answer 写 `(pass)`，chat 不追加） — `core/crossexam.py:194-198`, `223-239` — **高**（gaea 每轮强制全员产出，纯附和只能靠 prompt 判"无效发言"，不如给一个合法弃权通道）

[07] 静默看门狗 + 主席定向"捅一下" — 静默超过阈值就给主席单独塞一个"挑最弱未答点"的任务，且限流（每窗口一次） — `core/crossexam.py:405-420`, `223-239` — 中（gaea 有 round 推进，缺"这一轮没人有价值发言时怎么救"的机制）

[08] 三重终止条件（墙钟 / 静默 / 外部 stop） — 三个独立闸门取或，结束后给在飞回合固定宽限期再收尾 — `core/crossexam.py:441-461` — 中（简单但有效；注意 gaea 已经有 ctx cancel，价值在"静默闸"）

[09] 掉线清单注入裁决提示词 + 反虚构纪律 — 每个 dropout 记 role/agent/phase/error 并回写 manifest，裁决时显式列出，并要求"只评判提交者、不得替缺席者编立场" — `core/debate.py:183-195`, `307-319` — 中（gaea 有"缺席是运行状态不构成实质分歧"，`manager.go:3127`，可对照补"不得代为虚构"这一句）

[10] 终态事件优先级缓冲 — 有界队列满时驱逐"最老的非优先事件"，`verdict/roast/status/error/completed` 永不丢；全优先时宁可临时溢出 — `web/server.py:40-87`, 优先级集合 `54` — **高**（长跑会话里 pane 噪声挤掉裁决，是最典型的"用户看不到结果"事故）

[11] 订阅注册与快照同锁原子化 — "取积压 + 登记订阅者"在同一把锁内完成，保证不漏不重 — `web/server.py:589-597` — 中（重连/刷新场景的经典竞态，实现只有 4 行）

[12] 叙事流与原始流分层 + 独立自增 id — 原始 pane 流可有界裁剪，叙事事件单独无裁剪保存供重放；事件 id 用独立计数器而不是 `len(buffer)`（后者封顶后会永远相同） — `web/server.py:112-153`, `126-131` — 中（gaea 有 Seq，但"分层裁剪"的思路可借鉴到直播流）

[13] 产物路径"无空格"不变量 — 因为 agent 会大量 emit 未加引号的 shell 命令（touch/cd/printf），所以把"路径不能有空格"提升为系统级约束并用测试钉住 — `core/session.py:20-36`, `tests/test_session_paths.py:42-48` — **高**（gaea 在 Windows 上，用户目录/工作区路径常含空格与中文；这条直接命中）

[14] 适配器声明式契约 + 启动即全量校验 — 一个 YAML 描述"怎么起、怎么收尾、启动按哪些键、弹窗怎么答"，坏配置只告警不阻断启动 — `core/adapter.py:24-84`, `cli.py:38-57` — 中（gaea 的 adapter 是 Go 结构体，硬编码在 `agents.go`；外置成数据能让"加一个 CLI"不用改代码）

[15] 就绪闸门组合拳（定时键 + 读屏正则 + 发送前双轮扫描） — 三层叠加、任一层可缺省，且把"对话框可能晚到"当默认情况处理 — `core/turn.py:88-98`, `core/pane.py:47-83` — 中（仅在 gaea 将来引入 PTY 型席位时才有用；把"晚到的对话框"当默认，这个判断本身值得记）

---

## 不要抄的清单

1. **无重试的 dropout 语义**：一次超时即永久除名，且 crossexam 内部失败连 dropout 都不记（`debate.py:420-423` vs `crossexam.py:302-307`）。gaea 已有"空结果自动重试一次并留痕"（`manager.go:1660-1668`），不要退回到"一次就死"。
2. **回合开始不清理旗标**：`run_turn` 从不 unlink（`turn.py:72-75`），而 canary 会（`turn.py:124-127`）。同一份协议两套卫生标准，等于给"假成功"留正门。
3. **跨阶段共用 pane、收尾只等 30s**：`crossexam.py:456-461` + `debate.py:480-486` 允许两个回合同时对同一 REPL 打字。回合所有权必须显式化（锁/租约/独占句柄），不能靠"应该跑完了"。
4. **固定 sleep 当就绪判据**：3s / 6s / `max(delay)+2.5`（`turn.py:91`, `143`），并且 claude/codex 连 `ready_patterns` 都没有，`wait_until_ready` 直接短路 True（`pane.py:53-54`）。这类"调参型可靠性"换个机器就复现不了。
5. **宽正则自动应答权限弹窗**：`Allow` / `Approve` 对整屏 scrollback 匹配，命中就往输入框注入 `y`+Enter（`codex.yaml:10-11`, `pane.py:74-82`）。宁可卡住等超时，也不要往别人的输入框里打字。
6. **无界上下文塞入**：chat 全文（`limit=None`）进 rebuttal/verdict prompt（`debate.py:95-112`, `475`, `497`），且超时异常里的 80 行终端输出会经 `dropouts[].error` 进入主席 prompt（`turn.py:106-109` → `debate.py:186`）。必须有预算与清洗。
7. **失败态不落盘 + 内存只增不减**：manifest 只有 `running`/`done` 两个持久态（`debate.py:290`, `564`），error 只活在内存（`server.py:313-327`）；`SESSIONS` 永不清理且双 key 注册（`server.py:502-527`）。
8. **把 tmux/PTY 当作席位传输层**：pane 是给人看的，不是给协议用的——没有帧、没有边界、没有 ack、没有背压。gaea 用 stream-json 已经绕开，不要为了"能看它思考"把 PTY 请回来；要看过程，用结构化事件流（`EventAgentDelta`）比抓屏可靠一个数量级。

---

## 文档/README 声称但源码未见实现

| 声称 | 位置 | 源码事实 |
|---|---|---|
| 三阶段管线：opening / rebuttal / verdict，目录 `phase-2-rebuttal`、`phase-3-verdict` | `README.md:89-95`, `README.md:107-113` | 五阶段；目录是 `phase-2-crossexam` / `phase-3-rebuttal` / `phase-4-verdict` / `phase-5-roast`（`debate.py:444`, `484`, `495`, `528`）。README 从未提 cross-exam 与 roast |
| "Agents coordinate via flag files, **not terminal scraping**" | `README.md:100` | 全程读屏：就绪检测（`pane.py:56-62`）、权限弹窗（`turn.py:46-59`）、超时诊断（`turn.py:105`）、Web 直播（`server.py:223-227`）。旗标只用于"回合结束"，其余全靠抓屏 |
| "when every agent's `.done` lands, the phase is complete" | `README.md:105` | 每个回合各自等自己的 done（`turn.py:104`）；crossexam 阶段根本不以 done 为终止条件，而是墙钟/静默（`crossexam.py:441-452`） |
| adapter 的 `submit_key` 可配置提交键 | `README.md:125`, `adapter.py:41` | **死字段**：`send_enter` 硬编码 `"Enter"`（`pane.py:35-37`），全仓库无 `adapter.submit_key` 读取点 |
| `permission_prompts` 处理"任何其他交互" | `README.md:115` | 只处理**写死正则**能匹配到的弹窗；未知弹窗 = 触发语被吞 + 900s 超时（`turn.py:47-59`） |
| "every pane streams into the browser" | `README.md:51` | 1s 轮询 `capture-pane` 后整块替换（`server.py:223-227`, `app.js:706-713`），不是流；丢帧不可见 |
| 依赖 `watchdog>=4.0`（暗示文件系统事件） | `pyproject.toml:31` | 全仓库无 `import watchdog`；`core/watch.py` 是无人调用的死代码（`wait_for_file`/`wait_for_all` 零引用），实际用轮询 |
| 注释："seed 同批到达的消息会留到下次 drain 再处理" | `crossexam.py:346-349` | `batch` 是局部 list，`continue` 后整批**丢弃**，没有回队 |
| 注释："按 (ts, from, text) 去重" | `crossexam.py:351-353` | 无任何去重实现 |
| `doc/proposed-features.md` 说"目前是 fire-and-forget，单向 SSE" | `doc/proposed-features.md:14-23` | 这条**诚实且准确**（唯一的例外是 `server.py:552-580` 的 pane 按键注入，文档没提，属于"做了但没写"，不是"写了没做"） |
| `chat-say` 保证 JSONL "原子追加" | `crossexam.py:11` | 只有 `open(a) + write`，无锁无 fsync；依赖 POSIX `O_APPEND` 对小写入的原子性，跨平台不成立 |

---

## 附：与 gaea 的机制对照（用于判断"我们缺什么"）

| 能力 | aidebate | gaea（internal/pantheon） | 结论 |
|---|---|---|---|
| 席位就绪证明 | 功能级 canary（写+落旗标） | `--version` 探活（`agents.go:237`, `577`） | **gaea 缺** |
| 传输 | tmux pane + 抓屏 | `--output-format stream-json`（`agents.go:408`） | gaea 更优，不要回退 |
| 席位状态机 | 无（状态从事件流推断） | 完整（`types.go:83-93` + 转移表 `observe.go:44-82`） | gaea 远优 |
| 结构化发言约束 | 只有"点名 + 1–2 句" | 贡献 ID + 原话摘引 + 三级证据 + 讨论动作（`prompts.go:102-108`, `559-561`） | gaea 远优 |
| 群聊/定向质询 | `to` 路由 + 合并 + pass | 固定轮次全员发言 | **gaea 缺"被点名才说/可弃权"** |
| 静默救场 | 主席 stall 提示 | 无（靠轮次推进） | **gaea 缺** |
| 失败重试 | 无 | 空结果自动重试一次（`manager.go:1650-1671`） | gaea 更优 |
| 终态事件不丢 | 优先级驱逐缓冲（`server.py:40-87`） | 事件全量存 session + Wails emit | 可借鉴（直播层） |
| 缺席记账 | dropout 清单 + 反虚构纪律 | 缺席不构成分歧（`manager.go:3127`） | 各有一半 |
| 路径卫生 | "无空格"硬不变量 + 测试 | 未见对应约束 | **gaea 值得抄（Windows 尤其）** |
| 中途人工干预 | 单 pane 按键注入 | Gate/Question/Answer 决策门（`types.go:53-55`） | gaea 更优 |
