# 资产 15 · 技能选择施工级：整数权重词法 + 字符 n-gram + RRF 融合

> 源码：`ext/skills/src/dynamic_skill_selector/`（20 个文件）。
> 本篇取可实现的三件：`weighted_lexical_v1`（整数权重打分）、`character_ngram`（字符窗口，中文友好）、
> `rrf_lexical_char_v1`（RRF 融合，**全文只有 90 行**）。
> 对照：gaea `internal/gaea/skill`（18 文件）+ `bm25` + `textsim`。

---

## 一、RRF 融合：90 行，照抄即可

```rust
const MAX_FUSION_CANDIDATES: usize = 50;
const RRF_K: usize = 60;

// 两路各取 50 个候选，然后：
for ranking in [lexical_ids, character_ids] {
    for (index, id) in ranking.enumerate() {
        let rank = index + 1;                        // 名次从 1 起
        score[id] += 1.0 / (RRF_K + rank) as f64;    // 累加
        best_rank[id] = min(best_rank[id], rank);    // 记最好名次
    }
}
// 排序：分数降序 → 最好名次升序 → id 升序
```

**三个必须照做**：

1. **`K = 60`**：常数，不是超参，不要调；
2. **三级 tie-break**：分数 → 最好名次 → id。缺任何一级，同一查询两次可能给出不同顺序，
   直接破坏缓存与「同一输入同结果」的验收；
3. **元数据合并**：`query_term_count = max(两路)`、`query_truncated = OR`、`candidate_set_truncated = OR`
   —— 截断标志要「或」起来，任何一路截断都必须让上层知道。

## 二、词法打分：整数权重表（可直接实现）

```rust
// name（技能名）
query_phrase 出现在 name 里        → +256
name == query_term                 → +128
name 的分词集合包含 query_term       → +64
name 里有「相关词」                  → +24
// short_description（一句话描述）
包含 → +16      相关词 → +6
// description（正文）
包含 → +4       相关词 → +1
// 收尾
score += matched_query_terms²       // ★ 覆盖的不同词数，平方加成
```

**`matched_query_terms²` 这个设计值得单独说**：它奖励「覆盖了多个**不同**查询词」的文档，
而不是「同一个词出现很多次」。比如查询「导出 excel 表格」：
- 文档 A 只有「excel」出现 5 次 → 匹配 1 个词，加成 1
- 文档 B 同时含「导出」「excel」「表格」 → 匹配 3 个词，加成 9

这在不引入 IDF/TF 的前提下，用一行代码抑制了「单个高频词刷榜」，很适合技能名这种短文本。

**权重是有意的整数梯度**：256 / 128 / 64 / 24 / 16 / 6 / 4 / 1 —— 名命中远大于描述命中，
相关词命中远小于精确命中。全部整数、全部确定性，同一输入永远同分同序。

## 三、硬上限清单（两路一致，必须抄）

| 维度 | 值 |
|---|---|
| 查询字节 | 4 KiB |
| 查询词数 | 64 |
| 文档字节 | 4 KiB |
| 文档词数 | 256 |
| 候选集 | 1,000 |
| 返回结果 | 50 |
| 字符 gram 长度 | 2–5 字符 |
| 字段权重（字符路） | `[8.0, 4.0, 1.0]`（名 / 关键词 / 正文） |

**每条都是截断而不是报错**，并且通过 `query_truncated` / `candidate_set_truncated` 上报。
这条纪律对本地应用很关键：用户粘贴一大段需求进来，检索不能因此卡死或抛错。

## 四、为什么这套适合中文

`character_ngram` 用 2–5 字符窗口在**字节/字符级**匹配，不依赖分词。
中文没有空格，通用分词器在「技能名 + 中文描述」这种短文本上误切率高；
字符 n-gram 天然按字滑动，「导出表格」和「表格导出」能部分命中，代价是索引稍大——
但它有硬上限，实际开销可控。

## 五、gaea 施工方案（1–1.5 天）

1. 新增 `skill/lexical.go`：照 §二 的整数权重实现 `scoreWeighted(queryTerms, doc) uint32`，
   分词用「小写 + 非字母数字视为分隔符」，与 gaea 现有 `textsim` 的规范化对齐；
2. 新增 `skill/ngram.go`：字符 2–5 gram，字段权重 8/4/1，复用 §三 的上限；
3. 新增 `skill/fuse.go`：
   ```go
   const RRFK = 60
   func Fuse(rankings [][]int, limit int) []int   // 分数降序 → 最好名次 → id
   ```
4. `skill/select.go`：两路各取 50 → `Fuse` → 取 top N；
5. **不改** 现有 `skill` 包的加载与索引结构，只在选择环节加一层。

## 六、验收（可断言）

1. 同一查询连续调用 100 次，结果序列**逐元素相同**（确定性）；
2. 查询含 3 个不同词、文档分别覆盖 1 词 / 3 词 → 覆盖 3 词的排前面（验证平方加成生效）；
3. 粘贴 8 KiB 查询 → 不报错，`query_truncated=true`，结果正常；
4. 技能 ≥30 个时，人工标 30 条查询对比「现有关键词匹配」与「RRF 融合」的 top-3 命中率，
   要求不劣于基线（有提升更好，但要记录数字）；
5. 中文技能名（如「表格导出」）在查询「导出 excel」时能进 top-3（字符 n-gram 路径生效）。

## 七、不建议照搬

- **`lru_plus_*` 系列**（会话内 LRU 偏置）：需要跨轮状态，gaea 的选择是单次调用，先不做；
- **`routing_card_lexical`**（技能自带检索线索）：要求每个技能额外维护一份路由卡，
  技能数少时是纯负担，等技能到 50+ 再考虑；
- **`fielded_bm25`**：gaea 已有 `bm25` 包，但技能名这种短文本用整数权重更可控，不必两套并存。

## 八、一句话

这三件里最值得先做的是 **RRF 融合（90 行）**：它不改任何现有检索器，只把「词法」与「字符」
两路结果合并，就能同时拿到「精确命中」与「中文容错」两个好处；
而整数权重打分表是它的输入质量保证，两者一起上才有效果。
