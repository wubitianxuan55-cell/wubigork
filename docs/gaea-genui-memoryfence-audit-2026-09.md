# GenUI P5 剩余项审计：记忆/压缩侧围栏剥离审计（若影响确认）

> 日期：2026-09-05 · 基线：v4.99.0 工作区（静态审计，未跑真模型）
> 任务来源：docs/gaea-dsh-genui-distill-plan-2026-09.md §P5「记忆/压缩侧围栏剥离（若影响确认）」+ §5.6「记忆/压缩：待确认影响点」。
> 方法：沿「围栏文本/交互状态的存储形态 → 每条链路的读写点」逐处核对代码（file:line 证据），不做印象判断；「无证据=未验证」如实标注。
> 禁读区（Model Hub 并行线）未触碰；除本报告与 frontend/src/genui/**（含测试）外零文件改动；未 commit。

---

## 0. 基础事实：围栏的存储与流转形态（审计前提）

- **围栏 = assistant 正文的一部分**：模型在回答文本里写 \`\`\`genui / \`\`\`dsh-ui 围栏，随 `provider.Message.Content` 原样进会话内存与磁盘 jsonl，**后续轮次原样回喂模型**（Go 侧 session/provider/agent 无任何围栏剥离代码——grep `genui` 于 internal/gaea/{session,provider,agent} 仅命中注释，见 internal/gaea/provider/provider.go:33、internal/gaea/agent/ask.go:13 等）。因此围栏文本是「模型可见文本」，不是模型外内容；**真正的模型外内容只有前端 localStorage 的交互状态**（frontend/src/genui/interaction.ts:4-5,56-68，LRU 200）。
- 渲染缝三处，解析失败一律退化为普通代码块：老聊天栈 frontend/src/components/ChatMarkdown.tsx:70-71；老栈覆盖件 frontend/src/components/chat/genuiAdapter.tsx:21-24；办公/人格栈 frontend/src/gaea/components/Markdown.tsx:429-437。降级点：frontend/src/genui/markdownFence.tsx:32-33。
- 交互状态 → 模型的唯一通道是 `[genui-action]` 信封（≤1200 字符，作为 user 消息入会话）：frontend/src/gaea/App.tsx:160-167。
- 围栏体上限 64KB（frontend/src/genui/spec.ts:13 `maxFenceBody: 65536`）。

---

## 1. 记忆沉淀链 —— 结论：主链路原样无害；3 个低风险吸收点（Go 侧，待裁决）；1 个密码 payload 泄漏（genui 内，已修）

**文本源逐点核对：**

1. **前端用户提交记忆**（MemoryPanel 快速记忆 / quickadd）：文本源 = 用户输入框，非会话文本 → 围栏不会进入。证据：frontend/src/gaea/components/MemoryPanel.tsx:171-177（`note` state 来自输入框）→ internal/app/gaea_ui_meta.go:350 → internal/gaea/memory/quickadd.go:14-43（AppendDoc，oneLine 规整）。
2. **模型显式 remember 工具**：name/description/body 全部是模型手写的工具参数，**没有任何"从会话文本抓取"的自动路径** → 围栏原文不会被整段吸入（除非模型主动抄写，提示词纪律管辖）。证据：internal/gaea/memory/remember.go:63-133（仅解析 args 落库）；沉淀纪律提示词 internal/gaea/agent/single_prompt.go:47。
3. **自动做梦（办公，每轮后台）——吸收点 A（低风险，需修）**：`dreamTurnMessages` 取最后一轮 user+assistant 原文，`dreamInput` 每条截 1500 字符直接拼入提炼输入，**无围栏剥离**；提炼结果经 SaveDreamFacts/QuickAdd 写入长期记忆。若回复以大围栏开头，1500 字符窗口被 JSON 占满——真内容进不了提炼，且 fact/note 可能引用 JSON 噪声。证据：internal/app/gaea_dream.go:245-287（dreamTurnMessages/dreamWorthwhile/dreamInput）、127-190（runDream）、internal/app/gaea_handler.go:100（每轮触发）。**规划 §5.6 的「抽取函数前剥离」未落地**，且实际落地点在 Go（规划的 parse.ts 剥离不适用于此后端链）。
4. **确定性 /dream（BuildCompactSummary）——吸收点 B（很低）**：assistant 正文含 todo/next/pending/follow up/remaining 关键词时，取**整条消息前 160 字符**进 pendingItems → QueueMemory → 记忆笔记；围栏 JSON 开头可被吸入。证据：internal/gaea/agent/compact_summary.go:53-63、internal/gaea/control/dream.go:310-344。
5. **whisper（人格模式）——吸收点 C（低）**：companion reply 日志 fact 摘要 = `"gaea回复：" + 前 200 字符原文`（post_chat_turn.go:197-211），围栏开头即进 FactStore（FactLayer=raw, Weight 0.3）；working memory 原样存全文（post_chat_turn.go:94-102，仅会话内存活）。**用户档案本身有双防线**：抽取提示词明示「gaea（勿抽取）」（internal/whisper/memory_fact_extract.go:33-37）+ 用户事实守卫 FilterExtractedUserFacts（internal/whisper/memory_ingest.go:133-141）→ 围栏正文不会入用户档案，风险限于 companion_reply/episode 类自述条目抄入 JSON。
6. **genui 交互状态（localStorage）**：仅前端自读自写，不进任何后端管道（frontend/src/genui/interaction.ts 无外部消费方）。
7. **发现 D（隐私，genui 内，已修）**：`InputNode.submit` 原先把 password **明文**放进 action payload → `[genui-action]` user 消息 → 会话日志 → 压缩/做梦链（dreamTurnMessages 包含 user 消息）→ 可进长期记忆。这违背 handbook 秘密禁令（internal/gaea/genui/handbook.md:33「不要放 password 输入」）与持久化侧同款纪律（interaction.ts:2「password 值永不落库」、GenuiBlock.tsx:133-140 已过滤持久化，但 emit 通道漏防）。**已在本审计中修复**（见 §7）。

**风险分级**：A=低（内容质量/噪声，无安全面）；B/C=很低（条目噪声）；D=中低（隐私，发生前提是模型违反手册输出 password 输入，纵深防御补齐）。

---

## 2. 上下文压缩链 —— 结论：原样保留且长度如实计入（口径正确，与模型可见性一致）；1 个低风险改进点（Go，待裁决）

- **压缩摘要输入：围栏原样进 summarizer**。`renderTranscript` 对 assistant `Content` 原样输出（含围栏）→ `a.summarize` 的 user 消息。证据：internal/gaea/agent/compact_util.go:68-89（renderTranscript）、internal/gaea/agent/compact.go:581（`transcript := renderTranscript(fold)`）。这与规划 §5.6「建议正文保留（模型需要回看）」一致——围栏是模型可见正文，压缩前不该剥离。
- **长度统计计入**：`estimateMessagesTokens` 对 Content 全量计数（compact_util.go:27-64）；mid-turn 估算 `EstimateContextTokens` = 字节×tokPerChar（internal/gaea/agent/agent.go:594-601）；上下文看板口径 `len(s)*0.25`（internal/gaea/contextview/estimate.go:7-9）+ assistant 节点 `estimateTokens(p.Text)`（internal/gaea/contextview/fold.go:315-334）。三处口径一致且都把围栏计入 → **成本如实可见**（规划 §10 风险对策「token 成本如实可见（上下文页已统计）」成立）。
- **digest 化石化风险（低，待裁决）**：`summarySystemPrompt`（compact.go:56-81）没有对 UI 围栏 JSON 的处理指引，summarizer 可能按「Preserve identifiers」规则把围栏 JSON 片段抄进 digest；digest 被永久 pin（pinnedPrefixLen 保留全部 compaction-summary，compact.go:387-403）→ JSON 噪声长期占上下文、并可能诱导模型复发旧围栏。机械降级路径（mechanicalFoldDigest，compact_util.go:17-23）不含正文，无此问题。
- PruneStaleToolResults 只修剪 tool 结果（internal/gaea/agent/prune.go:47），不动 assistant 正文围栏——正确。

---

## 3. 子代理投影/转录链 —— 结论：原样无害，渲染与 token 估算一致

- **转录持久化**：子代理 transcript（sa_/mt_ 同管线，internal/gaea/agent/subagent_store.go:33、148-161）为原文 jsonl（subagent_store.go:253）→ 围栏作为模型可见正文原样保存，正确。
- **渲染**：SubagentThread → AssistantMessage（SubagentThread.tsx:239-241「正文与思考走主对话 AssistantMessage」）→ MemoMarkdown/Markdown，genuiKey=item.id（frontend/src/gaea/components/Message.tsx:270）。**关键事实**：SubagentThread 渲染在 GenuiScopeProvider 之外（frontend/src/gaea/App.tsx:1501-1510 的 activeSubagent 分支 vs provider 仅包住 chatTab Transcript，App.tsx:1514）→ `useGenuiScope()` 返回 null → `genuiFenceStateKey` 返回 undefined（frontend/src/genui/markdownFence.tsx:41-48）→ 交互组件以 **volatile 态**渲染：不持久化（GenuiBlock.tsx:16 stateKey 无值即 volatile）、不投递右栏面板（Markdown.tsx:553 `if (genuiScope === null || genuiKey === undefined) return`）、action 诚实禁用（frontend/src/genui/GenuiActionContext.tsx 缺省 undefined + renderNode.tsx `hasAction` 闸，573 等）。只读转录语义正确，不会污染主会话面板/状态库。
- **token 估算一致性**：fold 管线按 assistant 文本全量估算（fold.go:315-334），围栏计入=模型可见，一致。**v4.83「官方 patch 口径」经核实是图片 token 估算**（internal/gaea/contextview/imgtoken.go:3-4「28×28px=1 token ⌈w/28⌉×⌈h/28⌉」；CHANGELOG.md:119-121），与围栏块无交集，不存在需要特殊处理的围栏口径。
- 子代理最终答复回投主回合（subagentRef 徽标，Message.tsx:19/215-231）在主会话 scope 内正常渲染，状态按主会话口径管理，无跨会话串写。

---

## 4. 会话恢复/回放链 —— 结论：指纹防线有效，诚实降级（不渲染错状态）；1 个已知限制（状态跨重启不恢复，非危害）

- **stateKey 结构**：`genui:{scope}:{sessionKey}:{slot}:{fingerprint(body)}`（frontend/src/genui/fingerprint.ts:16-23；djb2 32 位，fingerprint.ts:4-10）。sessionKey 稳定：办公 = 会话路径 sanitize（App.tsx:182-186 currentSessionKey），聊天 = topic id（frontend/src/pages/ChatPage.tsx:410）。
- **内容被压缩/裁剪后**：resume 历史来自当前（可能已压缩）session 快照（`Controller.History()` → `Session().Snapshot()`，internal/gaea/control/controller.go:1173-1178）→ 被折叠的围栏消息不再重放；保留尾部消息索引位移 → 重建槽位 `h${index}`（frontend/src/gaea/lib/store.ts:81-87 rebuildHistoryItems）变化 → stateKey miss → `loadBlockState` 返回 null（frontend/src/genui/interaction.ts:50-54）→ **干净默认渲染**（GenuiBlock.tsx:57-61 用 persisted?.x ?? 默认值）。**不存在"压缩后渲染旧状态"的错误路径** ✓。
- **同内容重放恢复**：指纹相同 → 同 key → 恢复。该承诺在「槽位稳定」场景成立：流式→终态用同一 item id；中途 resync 用日志序稳定 id `a<日志seq>`（internal/app/gaea_resync.go:27-28、154）。LRU 逐出（200 条，interaction.ts:62-65）也只是干净降级。
- **已知限制（诚实但不理想）**：实时 item id `a${前端seq}`（store.ts:248,275）≠ resume 重放 `h${index}` ≠ resync `a<日志seq>` → **跨重启 resume 时 LRU 里的交互状态命中不了**（孤儿化直至逐出）。不串状态、无害，但规划 §0.4「历史重放原样恢复」在 resume 场景实际不生效。统一 id 口径属 store.ts/gaea_resync 跨层改动 → 报告待裁决（§7.5）。
- 残余理论风险：djb2 32 位碰撞需要同 sessionKey+slot 内两条不同围栏体碰撞，概率可忽略（未做定量，标注=未验证）。
- **附带发现（很低，越界报告）**：办公面板 `append:true` 规格在**会话中途 resync** 时以新 sourceKey 重复发布 → 面板 tab 重复追加（frontend/src/gaea/lib/genuiPanel.ts:44-72 去重键=sourceKey+spec 指纹，sourceKey 随 resync id 变化；Markdown.tsx:553-562 重放发布）。刷新/重开会话场景面板 store 为空、顺序重放收敛，无此问题。

---

## 5. 导出链 —— 结论：形态明确=原文本，无害

- **办公栈**：`exportAsMarkdown` 把 assistant 正文原样写入（含围栏原文）→ downloadMarkdown / ExportDeliverable（md/docx/pptx/xlsx/pdf 同管线）。证据：frontend/src/gaea/lib/export.ts:11-25（`case "assistant": ... lines.push(it.text)`）、frontend/src/gaea/App.tsx:546-559（exportConversation）、App.tsx:1431。外部查看器自然退化为代码块；**交互状态与面板内容不导出**（只导 items 文本）。与规划决策 #5「chat 导出含围栏：保留原文（自然退化）」一致 ✓。
- 压缩摘要卡片不导出（export.ts 的 switch 无 `compaction` 分支）→ 导出文档无 digest 噪声。
- **老聊天栈**：ChatTopicExportMarkdown 逐行转义反引号/尖括号（internal/app/chat_service.go:344-353、378-411）→ 围栏标记变字面文本，无代码块结构也无 HTML/结构注入面。形态明确 ✓。
- 未验证（如实标注）：docx/pptx 二进制转换管线对围栏文本的具体呈现（本审计只验证了 md 形态原样进入管线）。

---

## 6. 总裁定：**需修（清单）**——1 项已在 genui 内修复，其余越界项待主代理裁决

### 7. 已实施（完全落在 frontend/src/genui/** 内）

| # | 修复 | 位置 | 动机 |
|---|------|------|------|
| 1 | password 输入的 action payload 不再回传明文，改为 `{valueLength, id}` 长度信号 | frontend/src/genui/renderNode.tsx（InputNode.submit） | 明文经 `[genui-action]` 落会话日志 → 压缩摘要 → 自动做梦记忆链（dreamInput 读 user 消息）；与 handbook.md:33 秘密禁令及「password 值永不落库」持久化纪律对齐 |

测试（新增 2 例于 frontend/src/genui/GenuiBlock.test.tsx）：password payload 不含明文只含 valueLength/id；普通 text input payload 行为不变（回归锚）。
结果：`npx vitest run src/genui` → **3 文件 25/25 全绿**；genui 相关缝测试（genui.walkthrough / ChatMarkdown.genui / Markdown.test）→ **3 文件 28/28 全绿**。未跑全量 vitest、未跑 Go 测试（按任务纪律）。

### 8. 待主代理裁决（越界，本审计不实施）

| # | 项 | 位置 | 等级 | 建议 |
|---|----|------|------|------|
| 1 | **自动做梦围栏剥离**（P5 规划项本体）：dreamInput 拼输入前把 genui/dsh-ui 围栏体折叠为一行占位（如 `[genui 组件]`），避免 JSON 占满 1500 字符窗口与噪声入库 | internal/app/gaea_dream.go:276-287（+246-259） | 低-中 | Go 侧可复用现成围栏切分思路（前端 frontend/src/genui/parse.ts:25-73 splitGenuiFences 的行扫描语义，Go 重写约 30 行）；围栏外的正文务必保留（模型回看价值，规划 §5.6） |
| 2 | BuildCompactSummary pendingItems 160 字符截取跳过围栏体 | internal/gaea/agent/compact_summary.go:53-63 | 很低 | 同上剥离函数可共用 |
| 3 | whisper companion_reply 200 字符摘要跳过围栏开头 | internal/whisper/post_chat_turn.go:197-211 | 很低 | 剥离后再截断；或仅在「剥离后为空」时保留占位 |
| 4 | 压缩 summarizer 围栏口径：renderTranscript 折叠围栏为占位，或 summarySystemPrompt 增补「UI 围栏 JSON 不要原样收录进摘要」 | internal/gaea/agent/compact.go:56-81 + internal/gaea/agent/compact_util.go:68-89 | 低 | 注意：压缩输入里剥离围栏与「正文保留供回看」有张力——围栏仍在保留尾部（不进 digest）时可见；仅 fold 区转占位可接受 |
| 5 | resume 场景交互状态槽位口径统一（`h${index}` vs `a${seq}` vs `a<日志seq>`），否则 LRU 持久化在 resume 后永远 miss | frontend/src/gaea/lib/store.ts rebuildHistoryItems + internal/app/gaea_resync.go | 低（设计缺口，非错误） | 倾向 rebuildHistoryItems 改用日志序 id；属共享文件，须主代理排期 |
| 6 | 会话中途 resync 时办公面板 append 规格重复追加 | frontend/src/gaea/lib/genuiPanel.ts:44-72 | 很低 | 去重键改为 sessionKey+spec 指纹（忽略 sourceKey），或 resync 重建时清空 seen |
| 7 | 会话删除时清理 genui 交互状态（`resetInteractionStore`/`clearBlockState` 已导出但全仓无调用方） | 接线点在会话管理（越界） | 低（隐私卫生） | 删除会话时按该会话 stateKey 前缀清理 |

### 9. 风险与未尽项

- 本审计为静态代码审计：吸收点 A/B/C 的实际发生频率未做真模型验证（结构上可能、提示词/关键词门槛降低概率）；djb2 碰撞未定量。
- 导出链只验证 md 形态；docx/pptx 转换对围栏文本的最终呈现未验证。
- 压缩链「围栏是否应从 fold 区剥离」存在产品取舍（回看价值 vs digest 污染），本审计只给口径事实与建议，不拍板。
- Model Hub 并行线禁读文件零触碰；共享文件（i18n/App.tsx/bridge/store.ts）零改动；未 commit。
