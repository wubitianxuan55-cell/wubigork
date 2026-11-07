# gaea 聊天板块联网搜索可靠性计划（2026-11-06）

> 依据众神殿最新会诊
> `pan_1793963799148_6aecf8f40f0a`（2026-11-06 20:40 completed）：
> 「如何完善聊天板块，使其具有成功的搜索功能，刚刚测试，多次失败」。
>
> 该场结论状态是 **部分解决**：确定性代码断点已定位，四行验收门已收敛，
> 但修复、红测和安装产物实弹尚未完成。本计划承接 2026-11-05 已落地的
> 统一检索核 / 主源保留窗 / 状态契约，不推翻 B5-B7，只修用户可见交付闭环。
>
> 规划基线：`09722cc2`。当前工作树已有 14 个在途文件，主要是
> `bindings_*.go` / `app.go`，另含一个探针测试；它们属其他在途工作；
> 搜索修复必须在独立分支或 worktree 中进行，
> 不得混入这些文件。

---

## 一、目标、范围与完成定义

### 1.1 本计划解决什么

聊天板块的一次联网搜索回合必须稳定完成下面三条路径之一：

1. 模型判断无需联网：正常交付一版非空回答，并明确本回合未检索；
2. 模型执行联网：查询词、四态结果和来源反馈进入同一条可观测链路，
   最终正文可保存、可重放、无工具标记；
3. 检索或工具调用失败：界面如实显示降级 / 空结果 / 全败，模型基于
   已有知识作答或给出明确失败，不伪装成“实时搜索结果”。

“搜索成功”不等于接口返回非空，也不等于某一次答案看似正确。完成定义是：

- 拼接后的 `delta`、`done.reply`、数据库 assistant 正文三者完全一致且非空；
- 三者均不含 `<tool_call>`、`<function=` 或未识别工具标记；
- `searches` 与 `chatWebSearch` seam 的实际调用次数一致；
- 搜索状态为 `not_requested / ok / degraded / empty / all_failed` 中明确的一态；
- 搜索期间有可感知进度，前端不会把仍在推进的回合误判为 30 秒超时；
- 最终构建绑定明确 commit、构建时间和 exe SHA256，并在安装产物上复跑。

### 1.2 范围锁定

本计划只处理聊天板块的“联网搜索”：

- 普通对话 `plain`；
- 角色对话 persona；
- 文字聊天流式入口；语音 low-latency 路径单列回归，不默认宣称同步修复。

以下“搜索”不在本计划内，避免三种搜索再次混为一谈：

- 左侧会话标题 / 预览过滤；
- `Ctrl+K` 全局搜索；
- 历史会话正文全文检索与跳转。

若用户需要“找回以前聊过的内容”，另立会话检索计划，按消息级 FTS、
命中片段、跳回原消息和键盘导航单独设计。

---

## 二、最新论证对当前代码的结论

| 编号 | 已核验事实 | 当前后果 | 优先级 |
|---|---|---|---|
| A1 | `chat_tool_loop.go` 在带工具轮把正文写入 `res.Reply` 后直接返回，没有调用 `onDelta`；`runChatStreamPlain` 只累计 `onDelta` 并使用这份累计值落库 | 模型在仍有工具可用时直接作答，会出现 `done.reply` 与落库为空，随后触发空 assistant 保存失败 | P0 |
| A2 | 末轮 `bufferContent=false`，原始正文先 `onDelta`，之后才提取 / 清洗标记；清洗结果只写回 `res.Reply`，流式调用方不读 | 末轮吐工具调用文本时，raw 标记会进入界面、`done.reply` 和数据库 | P0 |
| A3 | `toolLoopResult.Searches` 从未赋值；聊天 adapter 的四态诊断被压成工具结果正文；`done` 没有搜索状态 | 无法区分“模型没搜 / 搜成空 / 全源失败 / 搜到后正文丢失” | P0-P1 |
| A4 | 现有工具循环单测主要走阻塞 `ChatSend`；`ChatStreamPlain` 测试关闭搜索；手跑探针默认 Skip、只 `Logf` | 默认 UI 路径正好缺少覆盖，空回复时探针仍可能 PASS | P0 |
| A5 | 前端 30 秒计时在 runID 到达时只启动一次，`delta` / `reasoning` 不重置；超时还会清掉已有正文 | 18.3 秒成功样本已占约 61% 预算，稍慢或主线程卡顿就会被误杀 | P0-P1 |
| A6 | 顶栏“允许联网”和输入框“回答前先搜索”是两个独立开关，文案仍残留“自动检测搜索意图” | 同一界面可同时出现互斥承诺，用户与模型都不知道谁负责触发 | P1 |
| A7 | 文本工具调用解析只覆盖 `<tool_call><function=...>` 一种方言 | JSON 形态、代码围栏或跨界 `<tool` 尾巴仍可能泄漏 | P0 |
| A8 | 2026-11-05 的统一检索核、主源保留窗、四态 adapter 已存在 | 当前主矛盾不在“没有搜索引擎”；P0 通过前不改源、不调排序 | 约束 |

结论：先把搜索回合收敛为 **一个权威终态**，再处理可观测性和内容质量。
直接换引擎、扩触发词、继续堆提示词，都会掩盖已经确定性存在的交付断点。

---

## 三、总体路线

### P0：搜索回合可靠交付（现在做）

1. 固定源码快照、构建身份和失败样本；
2. 先写默认运行的 `Search + ChatStreamPlain` 红测并确认旧代码失败；
3. 统一正文权威：`delta`、`done`、落库只从同一份过滤后的最终字符串产生；
4. 带工具轮无工具调用时，正确交付已生成正文；空正文不得伪成功；
5. 搜索模式正文在全量过滤后外发，raw 不进入任何用户面；
6. 最小搜索进度事件与可重置超时同批考虑，避免修好空回复后变成长时间沉默。

### P1：可解释、可诊断、语义一致

1. `done` 携带 `searches / search_state / search_rounds / search_summary`；
2. 结构化日志记录入口、开关、轮次、工具调用、查询、四态、耗时和正文长度；
3. 聊天界面显示“正在理解 / 正在搜索 / 已找到来源 / 降级 / 空结果 / 失败”；
4. 两个搜索开关收敛为一个明确的三态控件；
5. 强制搜索必须具备真实兜底，否则删除“必须先搜”的承诺；
6. 搜索摘要随消息持久化，刷新后仍可解释本轮是否联网。

### P2：内容质量与引用

1. 建立聊天级评测矩阵，而不是只看执行器返回；
2. 评估查询清洗、相对日期、结果相关性、来源新鲜度和引用；
3. 只有证据指向引擎或排序时，才调整主源、reranker、TTL 或并发；
4. 为最终回答提供可点击来源和“截至日期”，区分事实与模型推断。

---

## 四、批次执行计划

### B0 · 固化现场与红测（先证旧代码会红）

### 动作

1. 从明确 commit 创建独立分支 / worktree，记录：
   - commit hash；
   - 工作树状态；
   - Go / Node 版本；
   - 预期前端与桌面版本号。
2. 将现有手跑探针降为辅助观察，新增默认运行的确定性表驱动测试。
3. 固定 `runID`，订阅 `chat-stream:<runID>`，同时捕获：
   - 所有 `delta` 的拼接值；
   - `done.reply`；
   - `ChatMessagesList` 中的 assistant 正文；
   - `chatWebSearch` seam 调用次数、查询词和四态；
   - `error` / `cancelled` / 超时终态。
4. 在旧代码上实际跑红，保存失败断言后再进入修复。

### 必测四行

| 用例 | 假模型行为 | 硬断言 |
|---|---|---|
| T1 自动模式直答 | 搜索开启，但模型不调用工具，直接返回正文 | 界面 / done / DB 非空且相等；searches=0；状态 `not_requested` |
| T2 一次搜索后散文 | 第一轮调用 `web_search`，第二轮直接交付 | 只搜索 1 次；正文非空；界面 / done / DB 相等；无标记 |
| T3 末轮固定 raw | 最后一轮仍返回 `<tool_call>...` | 任何 delta、done、DB 均不出现工具标记；最终正文或明确降级非空 |
| T4 跨 chunk 尾巴 | 标记拆成 `<tool` / `_call>`，末块停在 `<tool` | 过滤器正确挂起并在终态丢弃；不泄漏半标记 |

### 增补变体

- 文本调用 JSON 形态；
- tool_call 代码围栏形态；
- 清洗后正文为空；
- 工具本体未注册；
- 搜索执行返回 `ok / degraded / empty / all_failed` 四态；
- 带工具轮直接散文发生在第 1 轮和第 2 轮；
- persona 路径同样满足 delta / done / DB 一致。

### 门禁

- 新测试默认运行，不依赖 `GAEA_PROBE`；
- 不允许只 `t.Logf`；
- 空字符串必须让测试失败，不能用 `strings.Contains("", marker)==false` 假绿；
- 旧代码失败记录、修复后全绿记录均留档。

---

### B1 · P0：一个权威终态

### 实现原则

引入一条明确不变量：

> 每个搜索回合的最终正文恰好交付一次；`delta` 序列、`done.reply`
> 与落库正文由同一份过滤后的 canonical string 产生。

### 后端改动

1. `runBoundedSearchToolLoop`
   - 所有模型正文先进入本轮缓冲；
   - 完成完整解析 / 清洗后，才允许进入 `onDelta`；
   - 带工具轮没有 `tool_calls` 时，交付已缓冲的干净正文，不再只写 `res.Reply`；
   - 末轮不再先发 raw 后清洗；
   - `res.Reply` 作为 canonical 正文，逐轮累计且恰好一次；
   - 任意清洗结果为空时，显式重试一次无工具直答，或返回可识别错误；
   - 工具本体不可用时，回退到普通无工具调用，不返回空成功；
   - 设置 `res.Searches = searches`。
2. `runChatStreamPlain`
   - 不在循环结束后继续信任“delta 累加值”作为唯一结果；
   - 搜索分支使用 `toolLoopResult.Reply` 作为 `done.reply` 和落库正文；
   - 保存前执行最终校验：非空、无已知工具标记；
   - 失败时 emit `error`，不得进入空 assistant 落库。
3. persona / whisper 搜索入口
   - 复用相同 canonical 结果，不复制第二套规则；
   - 保证普通聊天和角色聊天对 raw / 空回复 / 落库的口径一致。
4. 标记清洗
   - 保留并强化 `markupDeltaFilter`，但搜索工具循环不能在未过滤时调用；
   - 覆盖 XML、JSON、代码围栏三种已知形态；
   - 对未知 `<tool_call>` 残余采用安全失败，不把“清洗为空”当成功；
   - 不粗暴删除正常 HTML / Markdown。

### 取舍

搜索模式下允许牺牲部分逐字流式：模型正文先完整缓冲，再一次性或少量
安全分片外发。这是换取“不泄漏、可保存、可核对”的合理代价。无搜索回合
保持现有逐字流式，不受影响。

### 验收

- B0 四行测试及增补变体全绿；
- `searches` 与 seam 次数一致；
- 不出现空 assistant 持久化错误；
- `delta`、`done`、DB 三者一致；
- 普通 / 角色两条路径均覆盖。

---

### B2 · P0 安全配套：进度与可重置超时

最新会诊对“进度帧是否进 P0”仍有分歧。本计划建议：

**最小进度事件与 canonical 交付同批发布。** 原因是搜索模式改为完整缓冲后，
用户可能在 10 秒以上只看到空白；修好空回复却留下“沉默回合”，仍会被当成失败。

### 后端事件

新增轻量 `status` 事件，只发状态转换，不发高频心跳：

```json
{"type":"status","phase":"thinking","round":1,"elapsed_ms":120}
{"type":"status","phase":"searching","round":1,"search":1,"query":"上海明天天气"}
{"type":"status","phase":"search_done","search":1,"state":"degraded","reason":"primary-timeout","sources":4,"elapsed_ms":2100}
{"type":"status","phase":"finalizing","searches":1}
```

建议起点：

- runID 到达后立即发“正在理解问题”；
- 每轮模型调用发一次 `round_started`；
- 工具调用参数完整后发 `searching`；
- 检索返回后发 `search_done`；
- 进入最终正文时发 `finalizing`。

查询词属于用户输入，可在进度中展示；日志需避免无界保存网页正文。

### 前端超时

推荐 B 方案，待产品最终确认：

- **30 秒无真实进度超时**；
- **120 秒绝对硬上限**；
- `delta / reasoning / status(searching|search_done|finalizing)` 重置静默计时；
- 单纯 keep-alive 不重置；
- 硬上限不因事件重置；
- 超时不再无提示地清空已有正文；保留部分内容并明确显示“等待超时”；
- 取消、切题、卸载仍立即取消后端 run，迟到事件不得串入新题。

若产品坚持 A 方案（runID 后固定 30 秒总时长），则必须用 18.3 秒成功样本
重新计算重试余量，并明确放弃较慢本地模型的成功率目标。

### UI 呈现

流式 assistant 占位区内增加一行轻量状态，不新建独立消息：

- `正在理解问题`
- `正在联网检索：{query}`
- `已找到 {n} 个来源，正在整理`
- `主搜索源超时，已使用备用来源`
- `未找到相关结果，正在基于已有知识回答`
- `联网搜索不可用，正在基于已有知识回答`
- `最终整理中`

状态使用图标 + 文案 + 辅助色三重表达，`aria-live="polite"`，
查询词单行省略；`prefers-reduced-motion` 下只保留静态文字。

### 验收

- 首个进度在 runID 后 1 秒内出现；
- 持续 delta 不会触发 30 秒静默超时；
- 真正无进度 30 秒仍会失败；
- 绝对硬上限能兜底；
- 超时后部分正文不丢；
- 搜索状态不造成布局跳动，桌面与 390px 无横向溢出。

---

### B3 · P1：可观测性和搜索开关语义

### 结果契约

`done` 帧新增最小字段：

```json
{
  "type": "done",
  "reply": "最终正文",
  "search_state": "ok",
  "searches": 1,
  "search_rounds": 2,
  "search_summary": {
    "queries": ["成都明天天气"],
    "degraded_reason": "",
    "sources": 5,
    "elapsed_ms": 2310
  }
}
```

字段约定：

- `search_state`：`not_requested | ok | degraded | empty | all_failed`；
- `searches`：实际执行的检索次数；
- `search_rounds`：模型调用轮数；
- `search_summary`：只放解释所需摘要，不保存整页网页；
- 旧前端忽略新增字段时保持兼容。

### 诊断日志

每次聊天至少记录：

- `runID / topicID / mode`；
- `searchEnabled / forceSearch`；
- `round / useTools / tool_calls / searches`；
- 模型原始 query 与实际 query；
- 四态、降级原因、来源数、耗时；
- `delta` 长度、`done` 长度、落库长度；
- `ttft / duration / finalType`。

日志必须能回答“模型没搜、搜空、全败、搜到但没交付”四种问题。

### 搜索开关

推荐把两个布尔开关收敛为一个三态控件：

| 模式 | 行为 | 请求映射（兼容旧后端） |
|---|---|---|
| 关闭 | 不提供 `web_search` 工具 | `searchEnabled=false, forceSearch=false` |
| 自动 | 模型自行判断是否搜索 | `searchEnabled=true, forceSearch=false` |
| 必须联网 | 本回合必须产生一次真实检索后才能回答 | `searchEnabled=true, forceSearch=true` |

交互要求：

- 顶栏只显示当前模式状态，不再提供第二个语义重叠按钮；
- 输入器内一个控件完成切换；
- 文案、aria-label、提示文案三处同义；
- 删除“自动检测搜索意图”等 B7 已退役描述；
- 设置按会话记忆，切题时不悄悄继承错误模式。

“必须联网”不能继续只是提示词。必须实现确定性兜底：

1. 模型首轮调用工具：正常执行；
2. 模型首轮不调用工具：后端用清洗后的用户查询执行一次搜索，
   将结果回填后继续生成；
3. 如果产品不接受该兜底，则删除“必须联网”态，只保留“关闭 / 自动”，
   不得继续显示“回答前先搜索”。

### 错误与空态

- 每个错误必须有恢复动作：重试、关闭联网重答或打开搜索设置；
- 不显示内部枚举名，转为用户语言；
- 空结果不能说“实时搜索结果”；
- 降级结果必须带降级原因和“可能不完整”的诚实提示；
- 全败时允许基于已有知识回答，但必须标注“未获取到实时信息”。

---

### B4 · P2：聊天级质量评测与引用

### 评测矩阵

至少覆盖：

- 普通 / 角色；
- 关闭 / 自动 / 必须联网；
- 有主源 / 仅公共兜底；
- 搜索成功 / 降级 / 空结果 / 全失败；
- 无工具直答 / 一次搜索 / 两次搜索 / 末轮标记。

固定集沿用 2026-11-05 的 30 query 骨架，并补充聊天回合字段：

- 是否应该搜索；
- 实际搜索次数；
- 原始 query 与实际 query；
- 状态四态；
- 首条进度延迟、首正文延迟、总耗时；
- `delta / done / DB` 一致性；
- 来源数、来源域名、引用是否可点；
- 答案是否覆盖 `must_hit`；
- 事实是否新鲜、日期是否诚实；
- 模型是否忽略搜索结果。

### 质量门

- 搜索回合完成率 100%（非用户取消）；
- raw 泄漏 0；
- `delta / done / DB` 不一致 0；
- `searches` 与实际 seam 不一致 0；
- 应搜未搜率、搜索 query 明显错误率用固定集单独统计；
- 答案引用源不得只写“根据搜索”，应给出可核对来源；
- 搜索结果恢复后显示来源 chips / 链接和截至日期。

### 暂不做

- 在 P0 未过前更换主搜索引擎；
- 以单次查询成功调整 reranker；
- 用没有固定样本的“感觉更准”作为验收；
- 同时展开会话正文 FTS、语音搜索和联网搜索三条战线。

---

## 五、代码落点

### 后端

- `internal/app/chat_tool_loop.go`
  - canonical reply、分轮缓冲、完整清洗、搜索计数、最小事件；
- `internal/app/chat_service.go`
  - plain 流结果权威、done 元数据、空正文防线；
- `internal/app/whisper_handler.go`
  - persona / whisper 入口同口径；
- `internal/app/chat_search_adapter.go`
  - 暴露实际 query、四态、降级原因、耗时、来源摘要；
- `internal/app/chat_tool_loop_test.go`
  - 默认运行的表驱动回归；
- `internal/app/chat_service_test.go`
  - `ChatStreamPlain + search=true` 端到端事件与落库；
- `internal/app/whisper_probe_tool_loop_live_test.go`
  - 保留为实弹观察，不再错当合并门。

### 前端

- `frontend/src/hooks/useChatStream.ts`
  - 静默超时、硬上限、status 事件、终态 reconcile；
- `frontend/src/pages/chat/constants.ts`
  - 超时契约常量化；
- `frontend/src/pages/chat/types.ts`
  - 搜索状态与消息元数据类型；
- `frontend/src/pages/ChatPage.tsx`
  - 三态映射、状态装配、发送快照；
- `frontend/src/components/chat/MessageList.tsx`
  - 流式搜索状态行；
- `frontend/src/components/chat/ChatComposer.tsx`
  - 单一搜索控件；
- `frontend/src/components/chat/ChatModeBar.tsx`
  - 移除重复语义，保留准确状态；
- `frontend/src/pages/ChatPage.test.tsx`
  - 进度重置、硬上限、部分正文保留、搜索状态渲染；
- 本地化文件
  - 状态、错误与恢复文案三语同步。

### 设计与记录

- `design-system/gaea/pages/chat.md`
  - 补充搜索状态带、三态控件、超时和无结果规范；
- 本计划与发布验收记录
  - 记录 commit、构建时间、exe SHA256、实跑 query 与三次结果。

---

## 六、发布门禁

### 合并前

- 四行默认运行测试 + 增补变体全绿；
- Go 定向测试、全包测试、`go build`、`go vet`；
- `tsc`、定向 ESLint、相关 Vitest；
- 前端生产构建 / 桌面构建；
- 文档、绑定漂移检查不因本刀新增红项。

### 宣告用户可见修复前

必须绑定同一 commit 和明确构建产物：

1. 记录 commit、构建时间、exe 路径与 SHA256；
2. 安装该 exe；
3. 搜索开启发送“你好”一次：
   - 正文非空；
   - 未搜索时状态可为 `not_requested`，不得报失败；
4. 使用原失败 query 连跑至少 3 次：
   - UI 终态正常；
   - 无 raw；
   - `done.reply` 与 DB assistant 正文一致；
   - `searches > 0` 且查询可解释；
   - 四态与界面提示一致；
5. 保留失败样本，不用单次成功收口。

### 最终验收矩阵

| 维度 | 必过 |
|---|---|
| 路径 | plain / persona |
| 模式 | 关闭 / 自动 / 必须联网（若保留） |
| 结果 | 直答 / ok / degraded / empty / all_failed |
| 流面 | 0 工具 / 1 搜索 / 2 搜索 / 末轮 raw / 跨 chunk |
| 持久化 | delta = done = DB，非空、无标记 |
| 交互 | 进度、超时、取消、重试、无布局跳动 |
| 产物 | 同 commit + 同 exe 哈希 + 同 query ×3 |

---

## 七、风险与纪律

1. **不要先换引擎。** 当前有代码证明的交付断点尚未关闭，换源会混入新变量。
2. **不要让手跑探针继续假绿。** 默认 Skip、只 Logf、丢弃返回值的测试不具备门禁资格。
3. **不要把单次成功当修复。** 18.3 秒那次成功只覆盖两轮工具后末轮散文。
4. **不要用平均回复长度判断质量。** 只自动评分“非空 / 无 raw / 一致 / 状态 / 延迟”，质量由固定题集与人工抽检复核。
5. **不要让提示词承担结构兜底。** 强制搜索、空正文、标记泄漏都必须有代码级确定性防线。
6. **不要污染共享工作树。** 当前 14 个 binding 在途改动不属于本计划。
7. **不要混淆“连接测试通过”和“聊天搜索成功”。** 连接测试只证明引擎链可达。
8. **不要牺牲诚实性换体感。** 降级、空结果、全败均应被看见，不能包装成实时结果。

---

## 八、待拍板项（带推荐值）

| 决策 | 推荐 | 若拒绝 |
|---|---|---|
| 搜索开关 | 单一三态：关闭 / 自动 / 必须联网 | 若只保留两态，则删除 Must 承诺，不得保留“回答前先搜”文案 |
| 超时 | 30 秒静默超时 + 120 秒硬上限 | 选 30 秒总时长必须重新校准本地 35B 的成功率目标 |
| 进度帧 | 最小进度状态与 P0 同批 | 若延后，必须先接受搜索期长时间空白不算“搜索已修好” |
| 搜索摘要持久化 | 保存状态/次数/来源摘要，不保存整页网页 | 至少保留 done 帧与结构化日志，否则无法复盘 |
| 会话正文检索 | 另立计划 | 不要塞进本次联网搜索修复 |

---

## 九、Definition of Done

满足以下全部条件才算“聊天板块联网搜索已稳定”：

- 默认搜索开启的 UI 流式路径有四行以上自动回归保护；
- 搜索回合不会再因缓冲正文、末轮 raw 或空清洗导致空回复 / 标记泄漏；
- `delta / done / DB / searches` 四面对齐；
- 搜索进度可感知，超时只按真实静默与硬上限触发；
- 用户能看懂本回合是未搜、成功、降级、空结果还是全败；
- 搜索开关只有一个清晰语义；
- 同一固定题集在明确构建产物上稳定通过；
- 最后才讨论并调整查询质量、来源排序和引用呈现。

---

## 十、执行记录（2026-11-06）

执行分支：`codex/chat-search-reliability`（独立 worktree；未混入主工作树
现有 bindings 在途改动）。

### 已完成

- ✅ **B0 红测**：新增默认运行的 `TestChatStreamSearch_*`，在旧代码上实际
  复现四类失败：
  - 自动模式不调用工具 → 空 assistant 保存失败；
  - 一次搜索后散文 → 空 assistant 保存失败；
  - 末轮 raw `<tool_call>` → delta / done / DB 三面同时泄漏；
  - 跨 chunk `<tool` 尾 → delta / done / DB 三面同时泄漏。
- ✅ **B1 一个权威终态**：
  - 工具循环每轮正文先完整缓冲、过滤，再外发；
  - 带工具轮无调用时交付已缓冲散文；
  - `done.reply` 与落库改为直接使用 `toolLoopResult.Reply`；
  - `Searches` 已真实赋值；
  - JSON / OpenAI function wrapper / tool_call 代码围栏三种文本形态可解析；
  - 清洗为空时明确失败，不再进入空 assistant 落库。
- ✅ **B2 搜索进度与超时**：
  - 后端发出 `thinking / searching / search_done / finalizing` 状态；
  - `done` 新增 `search_state / searches / search_rounds / search_summary`；
  - assistant `extra` 持久化搜索摘要；
  - 聊天流显示轻量搜索状态行，完成后显示可展开的搜索来源卡；
  - 前端改为 30 秒真实静默超时 + 120 秒绝对硬上限；
  - delta / reasoning / status 会重置静默计时，超时保留已生成正文。
- ✅ **回归覆盖**：
  - `internal/app` 全包通过（164.0s）；
  - `internal/whisper` 全包通过；
  - `go vet ./internal/app ./internal/whisper` 通过；
  - `tsc -b` 通过；
  - 定向 ESLint 通过；
  - ChatPage 16/16、MessageList 15/15。
- ✅ **主工作树集成与桌面产物**：搜索状态、来源卡、canonical 正文与超时修复已并入
  `C:\AI\wubigrok`；按构建文档独立执行 `npm run build` 与 `wails build -s` 后，
  `build/bin/gaea.exe` 和桌面副本直接启动均通过 `/api/health` 200；
  SHA256=`B68AF7460610CC23C9CF81E81423FEA2C4172D94EA91D146C0956F92DF73926A`。

### 尚未完成

- ⏸ 三态搜索控件与 `forceSearch` 服务端兜底（当前仍保留两个布尔开关的
  兼容映射，尚未收敛 UI）；
- ⏸ 30 query 聊天级质量矩阵；
- ⏸ 同一产物的原失败 query ×3 实跑与人工界面验收。
