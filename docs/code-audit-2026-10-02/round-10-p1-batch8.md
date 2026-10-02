# 第十轮修复记录（P1 批次八：前端收敛）· 2026-10-02

> **承接**：批次一~七（b6f092f7 → 29be2ed4）。
> **验证**：`tsc -b` 0、eslint（触达文件）0 warning、FilePreview/ContextView/context 族 9 文件 148 测试全绿、gofmt/vet（Go 侧无改动）。

## 已落地（2 条前端收敛 + 1 条证伪）

### FE1-01 · PDF 懒加载四件套收敛
新 hook `components/useLazyPdfPages.ts`：pageElsRef/observer/lazyPdf 状态机/anchor 登记/aspect 测量/IO 接线/scrollTo 强制页——FilePreviewModal（v4.32 C）与 FilePreview（v4.33.0 对齐弹窗）此前逐行同构存在两份、注释互指。两消费方各留一行解构（净 −192 行）。lib/pageLazy.ts 纯函数不动（审计口径）。FilePreview 本地回调名 `handlePageImgLoad` 并入 hook 的 `handlePdfPageLoad`（语义同款：img 变量 vs 解构，等价替换）。
**过程踩坑留档**：文本手术切 IO effect 时头行 `useEffect(() => {` 未纳入删除区间留下 orphan（括号失衡 TS1005），已修复；教训=行级删除区间必须以 effect 头行为起点回溯注释，而非以首条语句行。

### FE1-02 · 上下文六分类单一事实源
新模块 `components/context/cats.ts`：`CONTEXT_CATS`（key+组行全名+节点行短名）与派生 `CONTEXT_CAT_BROWSE`。ContextView 的 `CATS`/`CAT_BROWSE_SHORT` 与 inspector 的 `GROUPS`/`ROW_LABELS` 全部改为派生——审计所指「未导出故本地重声明」的双源漂移面消除。

### AP4-09 · 证伪
「成本账本口径三份实现」不成立：`cloudEngineSet` 云端集合只在 overview 一处消费；`buildRouteLedger` 是顶层汇总纯透传（无重算）；`GaeaRouteSuggestions` 是 per (engine|model) 质量投影（形状独立）。三者无「同一汇总各写一遍」的收敛面，强收视图映射反而是无收益抽象。

## 留池（不变）

AP1-10（归一化超集口径，待拍板）、AP5-01（截断 14 份）、AP6-05（回合收尾四套）、AP2-03（在途登记六套）、FE2-05/06/08、FE7-11。
