# Nebula Memory（yb-zhai/agent-memory）→ gaea 记忆中枢 对齐蒸馏报告

> 2026-11-22 · 上游锚 `C:\AI\re\agent-memory` @ eeb378a · 逆向说明（引擎机制 file:line 锚）在
> `C:\AI\re\agent-memory\docs\蒸馏-gaea记忆中枢对齐-2026-11-22.md`，本文只做对齐与提案。
> 提案只交拍板池，不抢开工。

## 一、装了什么（已落机，与本文同批）

| 面 | 位置 | 状态 |
|---|---|---|
| DSH 插件 v1.3.8 | `C:\AI\dsh\home\plugins\dsh-memory-nebula` + profile web manifest | 安装器事务通过，健康检查 ✓ |
| 面板服务 | 8898（web host 半拉起，端口冲突 13 连试） | 「记忆星云」侧边栏入口已挂 |
| MCP 会话侧 | `cordis.patch.yml` id:agent-memory-mcp（4 工具 ≈1k tokens，全局挂载） | MCP stdio 探针 + 红队会话端到端写入均 ✓ |
| 记忆库 | `C:\AI\dsh\home\memory-nebula-data`（nodes/edges JSONL + index.json） | 双面同源实锤 |

## 二、gaea 记忆中枢现状盘点（对齐基准）

绑定面 `internal/app/bindings_memory.go`（MemoryB 门面）：MemoryView / Archive 族（ArchivedList、
CleanupArchived、Unarchive、SetRetentionDays）/ Duplicates(min) / Merge / EvalRun / Feedback /
Graph + SemanticGraph / HubOverview / Lifecycle / MorningBrief / Pin / Suggestions /
SemanticIndexBackfill·Status·Search / Profile 族（Save、Conflicts、ResolveConflict）/ Whisper 族
（Anchors、Episodes、CausalExplain、Replay）。数据面：三脑统一事实 `brain_store.go`
（brain.main/left/right × Fact{Entity,Attribute}，与 1.x 记忆中枢同构）+ internal/gaea/factbase +
internal/gaea/bm25（词法臂已有）+ 语义索引（SemanticSearch）。

## 三、Nebula 值得借的四件（按价值排序）

1. **四臂召回 + 逐条 arm 归因**（lexical/activation/graph/profile）。gaea 现状=词法（bm25）+语义向量
   两臂独立存在、结果无归因；四臂融合分 + `arm` 字段让「为什么想起这条」可解释、可评测（EvalRun 直接吃）。
2. **访问动力学（FSRS 思想）**：命中=访问→稳定性增长+act 写回；dormant（60d 未访问 ×0.55）与
   archive（R<0.15 且闲置>90d）是同一条 R 曲线的两个阈值。gaea 的 Lifecycle/RetentionDays 是
   按天数的粗闸，无「记得越牢越难忘」的动力学。
3. **consolidate：episode→fact 自动聚类**（≥0.85、簇 2..12、候选对上限防 O(n²)）。gaea 的
   Duplicates(min)+Merge 是人工触发；三脑 Fact 是结构化终点，缺「散记忆自动长成事实」的中间层。
4. **层次 event/fragment/summary**：写记忆时先落 fragment 层，聚合后才升 summary 层；gaea 记忆
   单层平铺，Merged/Archived 是事后状态而非生成期层次。

## 四、重叠面（不用抄）

- 记忆图谱 3D：gaea 已有「真 3D 脑网格」；Nebula 的 canvas 星云（零依赖手写）只是实现参考。
- 去重：gaea 已有 Duplicates 评审制（人工确认），Nebula 是写入即静默去重（0.92）——评审制更贴
  gaea 的「未经确认不写持久化数据」红线，不跟。
- 知识库 review/merge：gaea Knowledge 有 Review(approve) 门，比 Nebula 严。

## 五、拍板池提案（均未开工）

| # | 提案 | 一句话 | 约束 |
|---|---|---|---|
| P1 | 四臂召回进 gaea | bm25 + 语义 + 图扩散（三脑边）+ 置顶 四臂融合，结果带 arm 归因 | EvalRun 同批出分 |
| P2 | Lifecycle 接 FSRS 动力学 | 稳定性/act 写回 + dormant ×0.55 + archive 双阈值 | 现有 RetentionDays 语义保留 |
| P3 | consolidate 自动事实化 | episode 聚类 → 三脑 Fact 候选 → 走既有评审门 | 不绕 Knowledge/Memory 评审 |
| P4 | 编程板块记忆写入确认门 | MCP remember 直写无确认，与办公「未经确认不写持久化数据」红线对齐（observe 事件层可豁免） | 改 cordis 行或 dsh-wubi 侧 |
| P5 | 面板/MCP 双实例同库的教训 | gaea 记忆中枢保持单进程单实例事实源（Nebula 的 units 不刷新实锤在逆向说明 §四） | 设计约束，非遗刀 |

## 六、红线核对

- 未动 gaea 记忆中枢任何代码（本批只装编程板块 + 落报告）。
- Nebula MIT 允许借鉴；零依赖实现是自有代码，参考思路即可，不搬码。
- DSH 数据全部在 `C:\AI\dsh` 隔离根内（README-ISOLATED 边界未破）。
