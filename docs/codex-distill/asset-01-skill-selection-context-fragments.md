# 资产 01 · 技能动态选择器 与 上下文碎片契约

> 两条与 gaea 直接对口的可复用资产，均来自 `rust-v0.165.0` 源码，且**在二进制里有对应实现**
> （字面量 `dynamic_skill_selector` / `rrf_lexical_char` / `character_ngram` / `multi_query_lexical`
> 在 `evidence/strings.txt` 命中 7 / 5 / 2 / 3 次）。
> 位置更正：这两个模块在 `codex-rs/ext/skills/src/dynamic_skill_selector/` 与 `codex-rs/context-fragments/`，
> 不是 `skills/src/`（早期文档写错，已在本文件更正）。

---

## 一、技能选择：一个「方法即实现」的实验族

`ext/skills/src/dynamic_skill_selector/` 下 20 个文件，除测试外每个文件就是一种选择策略，
统一实现 `method() -> &'static str` 与 `select(query, documents, limit)`：

| 文件 | 策略 | 方法名（线上标识） |
|---|---|---|
| `character_ngram.rs` | 字符 n-gram 字段加权 | 字符级检索基座 |
| `fielded_bm25.rs` | 分字段 BM25 | 词级检索基座 |
| `weighted_lexical.rs` | 加权词法 | 词级变体 |
| `multi_query_lexical.rs` | 查询多视图拆分后合并 | `multi_query_lexical_v1` |
| `routing_card_lexical.rs` | 路由卡（技能自带检索线索） | 词法变体 |
| `character_routing_card.rs` | 字符 + 路由卡 | 字符变体 |
| `lru.rs` / `lru_plus_lexical.rs` / `lru_plus_character_routing.rs` | 带 LRU 偏置 | 会话内复用变体 |
| `rrf_lexical_char.rs` | 词法 + 字符 **RRF 融合** | 融合终选 |

### 1.1 RRF 融合（可直接照抄的公式）

```rust
const MAX_FUSION_CANDIDATES: usize = 50;
const RRF_K: usize = 60;

// 每路排名（1 起）→ score += 1.0 / (RRF_K + rank)
*scores.entry(id).or_default() += 1.0 / (rank_constant + rank) as f64;
// 同分时用「最好名次」再 id 排序，保证确定性
```

**要点**：`K=60` 是常数不是超参；候选截断 50；同分必须用确定性 tie-break（best rank → id），
否则同一查询两次结果可能不同，破坏缓存与可复现性。

### 1.2 字符 n-gram 的硬上限（防长输入炸内存）

```rust
MAX_QUERY_BYTES = 4 KiB      MAX_DOCUMENT_BYTES = 4 KiB
MAX_QUERY_TERMS = 64         MAX_DOCUMENT_TERMS = 256
MAX_QUERY_GRAMS = 512        MAX_DOCUMENT_GRAMS = 512
MIN_GRAM_CHARS = 2           MAX_GRAM_CHARS = 5
MAX_CANDIDATES = 1_000       MAX_RESULTS = 50
FIELD_WEIGHTS = [8.0, 4.0, 1.0]      // 字段权重，名称/关键词/正文三档
STOP_WORDS = [...]                    // 内置停用词表
```

**要点**：**每一维都有硬上限**（字节/术语数/gram 数/候选数/结果数），超限是截断而不是报错；
权重三档（8/4/1）说明「技能名命中」远重于「正文提及」。

### 1.3 多查询拆分

```rust
MAX_QUERY_VIEWS = 8
CONNECTORS = [" and then ", " and ", " then ", " also "]
// 原查询 + 按连接词切出的子查询各检索一次，合并时保留 full_query_rank
```

**要点**：切分只按固定连接词、最多 8 个视图；合并时**原查询的名次有优先权**
（`full_query_rank` 参与排序），避免子句把主查询挤出。

### 1.4 对 gaea 的用法

gaea 已有 `internal/gaea/skill`（18 文件）、`bm25`（4）、`textsim`（1），缺的是「多路融合 + 硬上限」：

1. 新增 `skill/select.go`：把一个查询拆成 ≤8 视图，分别用现有 BM25 与新增字符 n-gram 检索，各取前 50；
2. 用 `k=60` 的 RRF 融合，同分用「最好名次 → 名称字典序」定序；
3. 所有输入按上表上限截断（这条比算法本身更重要：gaea 的 skill 索引是本地文件，长文档一进来就爆）；
4. 字符 n-gram 对中文天然友好（2–5 字窗口），比分词更适合「技能名 + 中文描述」的匹配。

**验收**：同一查询两次结果逐字节相同；技能 ≥30 个时 top-3 命中率对比现有关键词匹配有提升（人工标 30 条即可）。

---

## 二、上下文碎片契约（context-fragments）

`context-fragments/src/fragment.rs` 定义一个 trait，所有「注入给模型的上下文」都必须实现它：

```rust
pub trait ContextualUserFragment {
    fn role(&self) -> &'static str;                       // 消息角色
    fn content_metadata(&self) -> ContentItemMetadata;    // 归属（默认 harness，生产时绑定）
    fn with_metadata(self, m: ContentItemMetadata) -> AttributedFragment<Self>;
    fn content_kind(&self) -> ContentItemKind;            // 稳定分类 <feature>.<name>
    fn markers(&self) -> (&'static str, &'static str);    // 起止标记
    fn body(&self) -> ContentItem;
    fn type_markers() -> (&'static str, &'static str);
    fn matches_text(text: &str) -> bool;                  // 用 type_markers 判定「这段是不是我」
    fn render(&self) -> ContentItem;                      // markers 为空则直出 body
}
```

### 2.1 三个关键设计

1. **标记对（markers）**：注入内容被 `<start>…<end>` 包起来，因此可以
   ① 从历史里精确识别并剥离；② 压缩时只压标记内；③ 重注入时按类型替换而不是追加。
2. **归属在生产时绑定**（`content_metadata` + `with_metadata`），**绝不从渲染文本反推**。
   这与 Guardian 的 `client_authored` / `harness_authored_configuration` 是同一条原则。
3. **稳定分类** `content_kind` = `<feature>.<name>`，供统计、审计、按类裁剪。

### 2.2 对 gaea 的用法（价值最大的一条）

gaea 现在把上下文拼成字符串（`internal/gaea/prompt` 4 文件 + `contextview` 20 文件）。
若改成碎片契约，能直接解决三个既有痛点：

| 痛点 | 现状 | 用碎片契约后 |
|---|---|---|
| 压缩后重注入不干净 | 摘要 + 重新拼上下文，容易重复/漏 | 按 `content_kind` 精确替换同名碎片 |
| 说不清「这段是谁写的」 | 只有角色（user/assistant/tool） | 再加归属（用户 / 工具 / 继承 / harness） |
| 事实核查与提示注入防御 | 无法区分工具输出与用户指令 | `matches_text` + 归属即可判定「工具输出不得建立授权」 |

**落地最小版**（不必引入 20 个类型）：先只做三件事——
① 给现有注入片段加一对标记（如 `<gaea:world_state>` / `</gaea:world_state>`）；
② 加 `Kind` 字段（`world_state` / `memory` / `skill` / `attachment`）；
③ 让压缩与重注入按 `Kind` 匹配替换。

**验收**：同一会话两次压缩后，注入片段数量与类型集合不增长（不重复）；能列出「本轮上下文里每一段的归属与类型」；
工具输出片段无法被当作授权证据（与 T3/A3 的 Guardian 判定对齐）。

---

## 三、为什么这两条值得先做

- 都不碰审批、压缩、执行这些敏感路径，纯增量；
- 都有**硬上限**和**确定性**要求，正好治 gaea 的两个老毛病（长输入爆内存、两次结果不一致）；
- 与本地小模型组合收益更大：字符 n-gram 不需要分词与向量，RRF 融合便宜；
  碎片契约让上下文可裁剪，直接省 token。
