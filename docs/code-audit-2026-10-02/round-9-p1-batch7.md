# 第九轮修复记录（P1 批次七：收敛型开刀）· 2026-10-02

> **承接**：批次一~六（b6f092f7 → 16911149）。
> **验证**：`go build` 全绿、`go vet ./...` 0、app（全包 127s）/visual/memory 测试全绿、gofmt 干净。

## 已落地（3 条）

### AP7-06 · 情绪评分确定性修复
`extractEmotionValue` 两张关键词表从 map 改有序 `[]kwRule` + 共用 `scoreByKeywords`（表序首个命中）——Go map 迭代随机，「紧张而温馨」这类多关键词标签此前每次评分不同，情绪曲线不可复现。表序=原表书写序（语义主从序：强词在前）。回归锁 `TestExtractEmotionValueDeterministic`：多关键词/单关键词/默认值 3 组 × 50 次重复恒等（随机序时代该断言间歇红）。

### AP1-05 · 「下一章章号」算法统一
`ensureChapterNode` 兜底分支 `len(of.Nodes)+1` 改走 `resolveTargetChapterNum(of, 0, "")`（全树 maxOrder+1）——同文件 :768 注释明写 len+1 在卷结构/缺节点跳章下「算出已存在章号反复覆盖」的实弹事故口径，同文件不得并存两套算法。生成链（CreateChapter/Outline/ChapterNode 族）测试全绿。

### AP4-06 · 交付记忆构造收敛
`memory.NewDeliverable(name, title, desc, body, tags)` 单点定义 Type/Kind 组合；`dagAcceptMemoryWrite` 不再内联常量。Name 规则（=节点 ID，重复验收同名 UPSERT）注释钉在构造函数——不改变现口径（改名会让既有已验收节点的记忆条目孤儿化，属产品决策）。

## 留池（需设计拍板）

- **AP1-10**（中文归一 N 份）：三份主实现语义**实质不同**（deepNormalizeText 剥包裹标点+ASCII 小写 / planNormalize 特定空白集+ASCII 小写 / normalizePlanEvent unicode 小写+CJK 标点折叠）——「收敛成哪一套」直接改变各处去重判据的合并结果（哪些事件/条目被视为同一条），属判据行为变更，需拍板超集口径 + 逐处核对去重影响后再动。
- AP4-09（usageAggregate 三视图收敛）、AP5-01（截断 14 份）、AP6-05（回合收尾四套）、AP2-03（在途登记六套）——同族后续批次。
