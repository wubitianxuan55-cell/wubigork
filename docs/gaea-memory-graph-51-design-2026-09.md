# 记忆语义图谱 5.1 设计（2026-09）

> 阶段五「记忆操作系统与上下文编译」首刀（v4.210.0 起）。上位文档：`gaea-next-stage-plan-2026-09.md` §2；路线图对位：`gaea-nextgen-roadmap-2026.md` 底座#1「统一记忆语义图谱（三脑孤岛 → 图谱记忆网格）」。本文件是 5.1/5.2/5.3 施工的域内基线：后续刀在此文档上续写，不另立新档。

## 1. 问题与目标

记忆此前是「存储」：facts 表按名存取，MemoryHub 的 `GaeaMemoryGraph` 是**演示型**关联图（同标签/同分类拼边，无事件事实），三脑数据之间没有可推理的结构。阶段五要记忆成为「操作系统」：一切记忆变更可追溯、互引可遍历、投影可重建，为 5.2 上下文编译（按任务取材）与 5.3 生命周期（衰减/蒸馏）供事实底座。

5.1 出口判据（规划 §2）与落点：

| 判据 | 承担者 | 状态 |
|---|---|---|
| 记忆互引可图遍历 | `memory.GraphNeighbors`（BFS，id/[MEM:] 键/显示名三入口，边双向可走） | ✅ v4.210.0 |
| [MEM:] 引用在回复发出前全部解析到节点 | v4.210.0 落观测侧（回合收尾解析+触达+cite 事件）；**v4.211.0 落执行侧**：agent stream() 收尾定稿闸（`Options.FinalizeText`/`SetFinalizeText`），悬空键在 Message 全文事件发出前剥离（改写值同进 session/摘要），boot 按记忆开关注入闭包，被剥离键落 dangling cite 事件保住可观测性；前端收 Message 整泡替换零改动 | ✅ v4.211.0 |
| 同一事件日志两次投影结果一致 | `ProjectEvents` 纯函数（不读时钟/不落 IO/输出排序），`TestProjectEventsDeterministic` + `TestGraphRebuildFromLog`（物化=投影逐字段一致） | ✅ v4.210.0 |

## 2. 架构（三层）

```
写路径（remember 工具/桌面面板/做梦/蒸馏/引用触达）
   └─ sqliteBackend.Save/Archive/Unarchive/ChangeType/Touch（internal/gaea/memory/sqlite.go）
        ├─ facts 表（内容真相，既有，不动）
        └─ memory_events 表 INSERT 一条事件（事件真相，追加式，绝不 UPDATE/DELETE）
                │
投影：ProjectEvents(events) 纯函数（internal/gaea/memory/graph.go）
                │  entity/event/source 三类节点
                │  produces（source→event）/ affects（event→entity）/ references（entity→entity）
                ▼
物化：mem_graph_nodes/edges/meta（internal/gaea/memory/graphstore.go）
    读取前 EnsureGraphFresh 水位对账（meta.max_seq < MAX(memory_events.seq) 即整体重建）
    → 删掉 mem_graph_* 再读一次即自愈，「日志即真相，删库可重建」
```

- **SQLite+向量列起步**：`mem_graph_nodes.embedding BLOB` 本刀恒 NULL，是 5.2 语义检索的占位；不上 Neo4j（个人尺度数据量不需要，维护面红线）。
- 日志不存全文：save 事件只带投影所需元数据（kind/type/title/desc/tags/space/互引 refs）+ 280 字节 excerpt 供展示；内容真相仍在 facts 表。
- 事件即审计：Touch（每回合引用触达/memory_get）逐条落事件——5.3 衰减的 last_used 事实源就在这里，不用再补埋点。

## 3. 关键口径

- **节点寻址**：实体 id 恒为 `mem:<project>/<name>`（facts 唯一键=(project,name)，跨项目同名是两条记忆，互引不跨项目）。`[MEM:name]` 引用键 → 图上实体 = id 后缀 `/<name>` 匹配。
- **投影确定性**：事件按 seq 升序折算，实体元数据 last-write-wins（最新 save 事件为准），archive/unarchive 只翻状态位；输出节点按 id、边按 (src,tgt,etype)、悬空列表排序去重。两条输入路径（内存构造/日志读回）的 Tags/Refs 统一归一 nil 口径——`TestGraphRebuildFromLog` 锁死「物化=投影」。
- **悬空拒写**：references 边只在两端实体都存在时物化；悬空目标不进图、不建节点，但计入 `Dangling`（meta 持久 + 视图统计），不静默。cite 事件永不创建实体（模型幻觉键名防节点化），只留事件节点与来源边。
- **cite 事件**：回合收尾解析回复中的 `[MEM:x]`，命中（Touch）与悬空各落一条（`Refs:["dangling"]` 标记）。尽力而为，日志不可用静默跳过，绝不影响回合。
- **视图裁剪**：`GaeaMemorySemanticGraph` 实体/来源全保留，事件节点按 seq 取最新 220 个（触达事件随时间无限增长，全量进 3D 力导图既卡也无信息增量）。裁剪只在视图层，物化/投影不动。
- **写路径挂钩尽力而为**：事件追加失败不阻断主写入（与 dream 审计同哲学）；物化读前懒对账兜底最终一致。

## 4. 消费面

- **绑定**（609，+1）：`GaeaMemorySemanticGraph → SemanticGraph`，返回 `SemanticGraphView`（nodes/links + eventCount/entityCount/danglingRefs/projectedSeq）。
- **前端**：MemoryHubPage 图谱页工具条新增「关联图 / 事件图谱」切换（GraphView 新 `source` prop，默认 cloud 旧行为零变化）；domainColors 增 entity/event/source 三语义色；mock 带 8 节点样例（`?mock=1` 可走查）。
- **Go API（未出绑定）**：`memory.GraphNeighbors`（互引遍历）、`memory.RebuildGraph`（强制重建）——UI 遍历路径/手动重建按钮按需另刀。

## 5. 测试与门禁

- Go：投影确定性 / 三向边 / 悬空不入图+Dangling / 实体 fold（末态 archived）/ 删库重建自愈（物化=投影）/ BFS 遍历（跳数、经边、[MEM:] 键解析、未知起点）/ 事件日志读写 / 写路径逐 op 挂钩（未命中触达不落事件）/ cite 事件形态。
- 前端：GraphView `source="semantic"` 数据源切换 + 标题切换；既有 8 例零变更全绿。

## 6. 5.1 余项与后续刀

- ~~**真·「回复发出前」闸**~~ ✅ **v4.211.0 已收口**：闸点=stream() 收尾（Message 全文事件发出前+返回值进 session 前），`memory.Store.StripDanglingCitations` 纯函数剥离（不 Touch，触达职责仍在回合收尾），boot 装配闭包（记忆开关闭/库不可用=不注入，子代理 nil 不改写），剥离键落 dangling cite 事件。流式增量原样透传、由 Message 重渲染收敛——前端零改动。测试：memory 三例（命中保留/悬空剥离+空间隔离/不触达+零值 Store 跳过）+ agent 两例（Message/Summary/session 三路改写、nil no-op）。
- **5.2 上下文编译**：意图分类 → 预算调度 → 前缀稳定排序；`embedding` 向量列在此刀启用。
- **5.3 记忆生命周期**：固化/衰减/归档三态替代 90 天一刀切——touch 事件已留痕，衰减计算直接吃日志。
- **投影水位增量化**：当前懒对账全量重投影（本地量级足够：万级事件 < 百毫秒）；事件到十万级再考虑按 max_seq 增量折算。
