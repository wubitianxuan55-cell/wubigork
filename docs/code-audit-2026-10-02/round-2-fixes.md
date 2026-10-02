# 第二轮修复记录（P0 清账刀）· 2026-10-02

> **承接**：`round-1-fixes.md`（第一轮 7 刀）+ 另一执行体 13:05 批次（P0#6/#7/#8/#12 + `gen_discipline_test.go` 线A 红测试）。
> **本轮范围**：开放 P0 中足迹互斥的 8 条实现 + 1 条定性 + 1 条结构项登记；**顺带根修第一轮保留渲染器的一个边界漏**（新测试抓出）。
> **验证**：`go vet ./...` 0 issues；`gofmt` 干净；`internal/app` 全包 + 12 个涉改包 `go test -count=1` 全绿；`tsc -b` 0；三份守卫脚本（primitives --strict / contract-drift / test-ctors）全绿。

## 一、本轮交付

### 刀1 · P0#4/#5 生成链纪律补全（converge + 逐场景）

- **#5 逐场景**：`scene_cards_handler.go` 逐场景生成协程补 `chapterGenWG.Add(1)/Done()`（登记/注销/ctx 判停已在场）。
- **#4 收敛闭环**：`converge_handler.go` 的 `NovelChapterConverge` 后台协程接入同一纪律——`registerChapterGen` 登记（同章互斥 + `CancelCreateChapter` 可命中）+ WG 记账 + 循环头 ctx 判停。
- **ctx 贯通（另一执行体红测试钉出的深修）**：`convergeRound`/`convergeRewriteUnit`/`rewriteUnit`/`llmRewriteSentences` 全链接 ctx——此前 `rewriteUnit→llmRewriteSentences` 用 `context.Background()` 发请求，**取消对句级修补链完全无效**（S1S2 为空时它还是收敛轮唯一的 LLM 调用）。批量重写绑定 `RewriteChapterAiTaste` 传 `a.ctx`。
- **验收**：`gen_discipline_test.go` 两用例（收敛登记/互斥/取消不续写/登记表清空 + 逐场景 WG 等待）全绿；`-run 'TestConverge|TestRewrite|TestChapter|TestScene|TestNovel|TestCreate'` 全绿。

### 刀2 · P0#14 图像生成并发拒绝

- `image_handler.go` `beginImageGen` 锁内 `imageGenRunning` 拒绝：并发第二生成返回 `(nil, nil, 0)` 哨兵，两个调用方（`generateImageInternal`/`GenerateMedia`）回中文 busy 错误，不再覆盖先者的取消句柄/进度单槽。
- 回归锁：`TestBeginImageGen_BusyRejected`（首占/拒绝/收尾恢复）。

### 刀3 · P0#13 whisperSessions 跨锁窗口

- `whisper_handler.go` 入库改写锁内二次查表：并发同键双建时先建者让位、已入库者胜出（构建段无全局副作用，双建只浪费一次构建）；读路径 RLock 语义不变。

### 刀4 · P0#10 DAG 波内并发额度

- `gaea_dag.go` 新增 `dagWaveSemaphore`：`[tasks].max_concurrent` 显式 >0 时波内节点按同一上限排队（额度内 pending、取消优先于排队），不再无上界绕过任务闸口；**缺省 0 = 不设限，保持 v4.221 波内并行产品语义**（`TestDagParallelWaveAndSessionAttribution` 锁定的真并发不受影响；分册 §C-1 本就标注该条 P0 依据需真机确认）。
- 回归锁：`TestDagWaveConcurrencyCap`（额度=1 时同波两节点 CAS 定角色、不得重叠）。

### 刀5 · P0#11 成本导入双写路径收编

- `internal/gaea/cost/cost.go` 抽 `saveTx` 内核并导出 `SaveTx(tx, e)`（UPSERT + tags marshal + 组成整组替换唯一实现）；`Save` 复用之。
- `gaea_cost_import.go` 删除第二份 `costEntryUpsertSQL` 常量与 `marshalCostTags`，整批事务改调 `SaveTx`。语义统一：空标题组成行按 Save 口径跳过（原 app 侧副本不跳）。
- 验收：`internal/gaea/cost` 全包 + app 侧 CostImport 族全绿。

### 刀6 · P0#23 heredoc 守卫（第 4 项检测）

- `scripts/check-primitives.mjs` 新增 ④：`.go` 内容里的 heredoc 形态（重定向 `<<-?['"]?(EOF|PYEOF|GOEOF|JSONEOF|PY|PYTHON|EOS)['"]?` + 泄漏定界行）——在册坑五犯的复发守卫；**基线=0**，探针自检命中、干净树 `--strict` exit 0。接 CI 仍属拍板项（脚本自述）。

### 刀7 · P0#24 TempDir 竞态守卫

- 新增 `waitGensBeforeTempDirRemove(t, a)`（create_chapter_cancel_test.go）：Cleanup 里非 Fatal 等 `chapterGenWG` 归零再放行 TempDir 删除（LIFO 保证先于目录删除），挂进三个会发起生成的构造助手（`newCreateChapterSuspendingApp`/`newChapterGateLLMAppReply`/`newGatedLLMApp`）。审计原名的 `newTestAppWithTempDir` 以「守卫挂进既有构造助手」形态落地（足迹更小、根因相同）。

### 刀8 · P0#2 审批链路保段断言 + **第一轮修复器的边界漏根修**

- 新增 `TestSaveRoundTripKeepsProfileGuardrailsAndPlugins`：审批链（Load → AddPermissionRuleForSpace → Save）往返后 `space_profiles.*.guardrails`、permissions 的 hard_ask/approval_timeout_secs、`[[plugins]]`（含 headers 内联表/auto_start）零漂移。
- **该测试抓出真 bug**：`RenderTOMLPreserving` 在「无顶层保留内容」时提前 return，**键级合并被整段跳过**——纯 owned 段配置（只有 space_profiles/plugins 等）内渲染器未建模的键照丢。根修：合并无条件执行（无合并无保留时输出逐字节等价 RenderTOML）；`joinLines` 改为对已 "\n" 收尾的输入不重复追加（split→join 逐字节稳定）。
- `TestSaveRoundTripKeepsAllSections` 补「原文段头逐个仍在」通用断言。

### 刀9 · P0#17 定性 + P0#6 尾波测试隔离修复

- **#17 证伪**：`tsc -b` EXIT=0，`CostLibraryView` 的 `setAllSelected/clearSelection` 定义在场（1126-1131），「引用未定义函数」不成立——按误报关闭。
- **#6 尾波**：P0#6 上抛库读错误后暴露潜伏的测试隔离缺陷——知识全局服务默认后端即 `db.GetDatabase(MemoryUserDir())`，cost 三测关闭该句柄不重置全局缓存，`TestSemanticSearchTool_EndToEnd` 顺序依赖失败（`sql: database is closed`）。修复：三处 cost 测试 Cleanup 补 `knowledge.ResetForTest()`；EndToEnd 装配时亦重置（顺序无关）。

## 二、登记不动（需拍板/结构项）

- **P0#25**（五重嵌入裸构造 panic 三犯）：属结构重构（assertAssembled 全绑定入口铺开 或 收敛唯一构造函数），与 AP4-01 同族；本轮以另一执行体的 `check-test-ctors.mjs` 守卫（基线 356 处，新增即报）作过渡防线，重构方案待拍板。
- **P0#6 跨进程双开**、**AP4-01 三套编排合并**：维持「另立一刀」结论。

## 三、口径备忘

- 另一执行体 13:05 批次（#6/#7/#8/#12）不在本记录重复展开；其 `gen_discipline_test.go` 线A 红测试由本轮刀1 转绿。
- `check-test-ctors.mjs` 基线按脚本自述工作流以 `--write-baseline` 刷新（+2 处 = 本轮 DAG 测试既有同款模式）。
