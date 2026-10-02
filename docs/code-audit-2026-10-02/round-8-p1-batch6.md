# 第八轮修复记录（P1 批次六：功能缺陷 + 收敛开头）· 2026-10-02

> **承接**：批次一~五。本批转 duplication/consistency 类，先挑「真功能缺陷」型。
> **验证**：`go build` 全绿、`go vet ./...` 0、app/knowledge/types 包测试全绿、`tsc -b` 0、KnowledgePanel vitest 绿、gofmt 干净。

## 已落地（5 条修复 + 1 条证伪带回归锁）

### FE1-03 · 知识库 phase 过滤修复（真用户可见 bug）
`KnowledgeSummary` 投影链缺 Phase 字段：Go `EntrySummary`/绑定投影/TS 类型都没有 → 前端选阶段过滤 + 空 query（走 List）时 `e.phase` 恒 undefined，**列表被过滤成空**（靠 `as unknown as` 强转掩盖类型缺字段）。贯通修复：`EntrySummary.Phase`（file/sqlite 双后端，sqlite List 补 SELECT phase 列）→ app `KnowledgeSummary.Phase` → TS `KnowledgeSummary.phase` → 前端过滤去强转；mock 与测试夹具补值。

### AP5-05 · 语义检索 topN 接入
`semanticSearchHitsOnDemand` 加 topN 参数（perKind = topN/4 下限 1、总截 = topN）：绑定面缺省 20（wire 契约零变更）、统一搜索 semantic 组与 file 组同用调用方 topN、检索评测传 `retrievalEvalTopK`——Recall@10 口径从此真实生效。

### AP5-10 · 补拉 afterSeq 增量落地
`GaeaResyncEvents` 按 seq 过滤补缺口（afterSeq<=0 全量）——签名收下参数却整表重放，前端拿到的 items 与已见内容重复。语义依据：v4.62.1 不变量「wire seq 与磁盘日志 1:1 对应」，序号空间同源，裁剪即本意。折叠器按调用重建合并状态，跨缺口 tool_result 独立成条。既有测试改双断言（全量合并 + 增量裁剪）。

### AP5-07 · 子代理 ref 校验收敛单点
app 层 `validSubagentRef`（与 `gaeaAgent.ValidRunRef` 逐字节同语义、注释自认应单点）删除，唯一调用点改调 agent 包实现。

### AP1-11 · 章号解析收敛
app 层 `chapterNumOf` 副本删除（与 `types.ChapterNumOf` 结果完全一致），三个调用点改调；`deepChapterRe` 是文件枚举过滤器（严格三位+分支字母）语义不同，保留。解析矩阵测试落 `internal/types`（补零/非补零/分支字母/无数字/超长），app 层重复用例删除。

### AP9-03 · 证伪 + 回归锁
审计称「意图 handler 被静默覆盖：主脑默认分支永远派发失败」——**两处误读**：①`moduleOfIntent` 实为 gaea.chat→gaea、gaea.create→office（不同模块，无覆盖对象；审计把 "chat.chat"→whisper 看串行）；②即便两意图落同模块，`len(d.intents)!=1` 的 fillFail 会先响亮拦截，handle 覆盖不会静默注册。生产链 `New→initModules→FillFromManifests(Builtins)` 完整。补 `TestMainBrainDefaultBranchDispatchable` 钉死 default 分支可派发。

## 留池（下批继续收敛型）

AP1-05（同文件两套下一章号算法）、AP1-10（中文归一六份）、AP5-01（截断 rune 14 份）、AP6-05（wx 回合收尾四套）、AP2-03（在途登记六套）、AP7-04/05（产物登记 45 行复制+台账三套填法）、FE1-01/02、FE2-05/06（前端批）。
