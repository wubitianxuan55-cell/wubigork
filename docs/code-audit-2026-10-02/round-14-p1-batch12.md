# 第十四轮修复记录（批次十二：P1 清理 + 守卫机器化）· 2026-10-02

> **承接**：批次一~十一（`b6f092f7` → `3f056528`）。批次十一已把「一刀可落」的 P0 清零，余两条结构拍板项（AP4-01 三套编排合并、X1-03 五重 `*core` 嵌入裸构造）。
> **本批口径**：P1 中**未被任何 round 记录触碰**的 143 条里，挑「真 bug / 静默失败 / 口径分叉 / 可机器化守卫」类，按 4 条互斥并行线落地。
> **范式**：4 条互斥并行子代理线（线1 app+成本链 · 线2 非 app Go 包 · 线3 前端 · 线4 守卫/CI/台账），主代理定契约、裁决降级、收线缝合、全量验证。
> **范围**（fid）：线1 = GA6-09 / AP1-09 / AP7-05；线2 = GA5-07 / IN3-01 / IN3-11 / AP6-10；线3 = FE3-06 / FE5-05 / FE7-09 + GA6-09 的前端消费；线4 = X1-13 / X1-12 / X1-10。

---

## 〇、动手前的底账核对（候选清单怎么来的）

1. **方法**：以 `machine/findings_all.json`（512 条）为底，逐条用 `round-*.md` 里的 fid 出现情况做**候选分诊**（脚本 `.tmp/audit-status.mjs`，可复跑）：
   - P0：25/25 在 round 记录中出现（批次十一已逐行复核，见其 §〇 更正 1）
   - P1：216 条中 **73 条已被 round 记录覆盖、143 条从未触碰**
   - P2：209 条中 13 条被提及；P3：62 条中 1 条
   **注意口径**：这里只是「分诊」，不等于「已修」——被提及的 73 条仍需逐条读码实测（批次十一已实证这一点）。
2. **挑选偏向**：优先 ①真 bug / 静默失败 / 错误通道缺失（GA6-09、AP7-05、AP1-09、FE3-06）；②口径分叉且**能机器守卫**（X1-13、X1-12、X1-10、FE7-09）；③性能与锁面（AP6-10、GA5-07）；④重复实现收敛（IN3-01）。
3. **前置基线（主代理实测，本批开工时的干净工作树）**：
   - `scripts/check-bindings-drift.ps1` → `OK：bindingNames.ts 与 Go 绑定面一致（744 个方法）`
   - `node scripts/check-primitives.mjs --strict` → `STRICT OK：无新增（截断 37 / 原子写 8 / 前端 5 / heredoc 0）`
   - `git status --short` → 空

### 0.2 执行事故与恢复（如实登记，2026-10-02）

**第一波 4 条并行子代理全部在交付前崩溃**（无收尾报告，`list_agents` 全部 `inactive`）。它们的中间产物留在工作树里，主代理按现场恢复：

| 事故面 | 现场状态 | 处置 |
|---|---|---|
| `internal/app/bindings_*.go` 11 份被再生 | 生成器把所有门面方法体从多行折成单行（**纯格式噪声**），并把 `GetSinImageConfig`/`SetSinImageConfig` 从 `ImageB` **错归** `CoreB`（本仓已知的「再生噪声」：门面归属是手工修过的，生成器口径不同） | 11 份**全部还原到 HEAD**（方法集合逐文件比对：其余仅格式差异，唯一实质差异就是上面那对方法的归属） |
| 线1（成本链）编译未收口 | `costimport/pricefeed/gaea_cost_graph/gaea_cost_rerank` 四处用了 `slog`/`fmt` **未加 import**；cost 包与 app 侧共 25+ 处测试调用点仍是单返回值 | 补齐 import；机械修完全部测试调用点（`x := s.Search(` → `x, _ := s.Search(` 等，逐处 Edit 而非脚本）；`go build ./...`=0、`go vet` 触达包=0、`go test ./internal/gaea/cost/... ./internal/gaea/costimport/... ./internal/gaea/pricefeed/...` 全 ok |
| 线2（非 app）只做完 GA5-07 | `evidence.go` + 新增 `evidence_coverage_test.go` 已就位且 `go test ./internal/gaea/evidence/...` 绿；`schedule`/`docmd`/`tts` 三项未动 | GA5-07 由主代理复核保留；其余三项交第二波线 B |
| 线3（前端）只做了一部分 FE3-06 | `spaceBindings.ts`/`spaceBindings.test.ts`/`bridgeFacades.test.ts` 有未完成改动 | 交第二波线 C 收尾（要求先读 diff 再做完） |
| 线4（守卫）只做完 X1-13 | `gen_bindings/main.go` 遮蔽闸已就位；CI 分类器与台账计数未动 | X1-13 由线 D 复核、其余两项交线 D |

**方法论留存（亏）**：并发子代理的中间产物**不是**原子交付——崩溃后主代理必须能凭 `git status`/`git diff` 判出「完成到哪、编译是否自洽、生成物是否被污染」。本批的判据是「先跑 `go build ./...`，再按包逐个 `go vet`（vet 会编译测试）」，两步就能把「半成品」与「已完成」分开。

---

## 一、已落地

### 线 4 · 守卫 / CI / 台账（X1-13 复核 · X1-12 · X1-10）

**X1-13 · 绑定遮蔽闸（复核上一轮已落地的实现，未重做）**：`scripts/gen_bindings/main.go` 改为 `main() → os.Exit(run(os.Args[1:]))`，新增 `shadowBaselineMax = 2`（含口径/来源/日期，并写明审计「约 281 份」是**含门面重复的全仓同名对**口径，与本闸不同源）、`shadowPair` 明细、`-shadow-check`/`-shadow-diag` 两个诊断 flag；**判闸在写生成物之前**，超基线 `exit 1` 并逐条列出新增遮蔽 + 两条处置路径。
证据：`go build ./...`=0；`-shadow-diag` → `遮蔽基线（在册上限）= 2；本次实测 = 2` + 两条明细（`SetFeatureModel` / `SetFeatureModelEnabled` ← core，`feature_model_handler.go:79/:147`）；**负例**（基线 2→1）→ `exit=1` 且列出 `SetFeatureModelEnabled`，还原→`exit=0`；`git status --short internal/app/bindings_*.go` 全程为空（零生成物噪声）。

**X1-12 · Go 侧 flaky 分类器（对齐前端）**：新增 `scripts/classify-go-failures.mjs` + `scripts/go-known-flaky.txt`（空清单，表头与前端同风格）。判据：解析 `go test -json`（包级/子测试级 `fail`、go1.21+ `build-fail` 去 `[pkg.test]` 后缀），**失败包集合 ⊆ 在册清单**才隔离复跑（绿则过、红则红），清单外失败直接 `exit 1`。比前端多两道防线：`--expect-failure`（首跑非零却归不出失败包 → `UNATTRIBUTED` exit 1，**不许静默放行**）、BOM/UTF-16LE 报告容错（PowerShell 5.1 重定向默认 UTF-16LE，实测踩到）。`ci.yml` backend job 只动该 job：「失败整批重试」→「跑一次落 `-json` → 分类器 → 只复跑清单内包」+ 新增分类器 selftest step。
证据：`--selftest` 9 组夹具 / 14 条断言全过；YAML 解析 OK（jobs 不变，4 个 hunk 全在 backend 段）；抽真实 run 块 `bash -n` 通过；**stub-`go` 端到端 6 路径**（GREEN=0 / FLAKY 复跑绿=0 / FLAKY 复跑红=1 / REGRESSION=1 / UNATTRIBUTED=1 / `go list` 空=1）；仓库外真 `go test -json` 端到端复核（含 UTF-16LE 报告）。

**X1-10 · 台账计数与包版本漂移（根因修复）**：实测口径改对 `releases/README.md`（校验和 **447**（根 427+archive 20）、发布说明 **636**、旧线 md **122**、CHANGELOG 全族 33、exe 根 5+archive 4、**仓库根 0 个**、源码包 11/166.5 MiB、`.git` 244 MiB 中约 68% 是源码包历史对象、V4_RECENT 34 条）。
**根因**：`scripts/check-docs.mjs` 原正则只认「校验和 422 份」，而 README 写「422 份校验和」⇒ **整段守卫从未生效**，这就是 422/427 长期没人发现的根因。现两种语序都认，且「一处都认不出」本身判 FAIL（措辞被改也不会静默失效）；新增对账：校验和份数 / 发布说明份数 / V4_RECENT 条数 / 实存 exe 清单 / 源码包份数（后两项仅在「本机确有该文件」时对账，因不入库）。
证据：6 组负例自测（A 447→446 / B 636→999 / C exe 清单改一项 / D **删掉数量词** / E 源码包 11→12 / F 34→33）逐条 `exit=1`，还原后 `exit=0`（每次字节级还原 byte-identical）。
**`frontend/package.json` 处置 = 无消费方 → 不动 + 明确登记**（不手改中间值）：`release.ps1`/`sync-version.ps1`/`check-version-drift.ps1` 只维护 `app_info.go`/`wails.json`/`versioninfo.rc` 三处；CI 不读；`vite.config.ts` 无 define；全仓 `npm_package_version` 零命中。决定与 grep 证据写进 `releases/README.md` 新增的「版本源清单（就三处，别再加第四处）」。

**线 4 的越界声明（主代理已接受）**：`.gitignore` +3 行（`go-test.json` / `go-test.err` / `scripts/go-retry-pkgs.txt`）——与既有 `frontend/vitest-report.json`、`frontend/vitest-retry-files.txt` 同款处理，否则每次跑门禁都在工作树留未跟踪产物。

**线 4 顺带发现（未改，登记为下一批候选）**：`ci.yml` 前端 job 注释称前端分类器「含 4 组夹具自测」，但 `frontend/scripts/classify-vitest-failures.mjs` **没有任何自测代码**（grep `selftest` 零命中）——注释是不实陈述，前端侧应与后端同款补夹具自测。

---

### 线 2（重启版线 B）· 非 app 包（IN3-01 · IN3-11 · AP6-10）

**IN3-01 · cd 工期五闸校验三处复制（已修，2 处真实语义分叉被消除）**：`internal/schedule/project.go:213` 落唯一实现 `validateDurationUnit(t Task, idInMsg bool)`，`Validate`（`project.go:101`）与 `ops.upsert_task`（`ops.go:128-132`）调用同一实现；`ops.patch_task`（`ops.go:165-183`）改为「先投射生效态（durationUnit/level/isMilestone/duration）→ 校验 → 再赋值」。
**逐段差异实测（四段：Validate / upsert / patch 单位块 / patch 工期块）**：D3 **真实语义**——Validate/upsert 校验任务**终态**，而 patch 校验的是**逐步应用中的中间态**（cd 约束在 level/isMilestone 赋值之前）⇒ 单条 `patch{level:0}` 或 `{isMilestone:true}` 打在 cd 任务上：对话路径放行、文件路径拒绝。D4 **真实语义**——patch 先写 `t.DurationUnit` 再查上限 ⇒ 失败留半改状态（回执/前端指针可见）。**两者均已在本次收敛**；D1/D2/D6 为纯文案差异（被 `frontend/src/schedule/ops_golden.fixture.json` 冻结，本线足迹不含 frontend，按 golden 保留）。
证据：`TestDurationUnitPathsAgree`（7 子例，Save→Validate / upsert / patch 三路径同结论 + 规则文本逐字一致 + 失败不落盘）、`TestDurationUnitSingleImplementation`、`TestPatchDurationUnitNoPartialMutation` + 既有 golden 用例全 PASS；**反向证据** M1（删 patch 的 level/isMilestone 投射）⇒「降为分组行+保留 cd / 设为里程碑+保留 cd」FAIL；M2（注释掉 Validate 的调用）⇒「文件保存路径应拒绝」+「Validate 与 validateDurationUnit 不一致」FAIL；还原后全绿。

**IN3-11 · 页数三口径（按审计允许的降级路径落地，未假装合一）**：`pdf.go:105` 新增**唯一判定点** `pdfTotalPages(raw, content)`（对外 total 仍取原始字节口径 ⇒ 行为不变，同时比对文本流视图并告警）、`pdf.go:46` 取代旧内联逻辑、`pdf.go:59-66` 新增「文本路径页对象数 > 声明总页数」告警；`render.go:71` 抽 `parseRenderedPageNumber`（视觉 diff 与 OCR 共用）；`ocr.go:508-538/579/619` OCR 页码改为**以 pdftoppm 实际产物文件名为唯一来源**（数值排序 + 跳页/越界对账告警 + 不可解析才回退 `first+i` 并告警）——这是**真修**，不是仅加告警。
证据：`TestPDFPageCountViewsDiverge`（探针① raw=4 / 视图=3 / 对外 total=4 + 告警）、`TestPDFPageCountContentViewExceedsRaw`（探针② raw=1 / 视图=2 / 正文被判到第 2 页 + 两条告警）、`TestRenderedPageJobsArtifactNumbers`（`page-1/2/4` → `[1 2 4]`；spec「1-3」只识别 1/2；spec「4」命中 `page-4.png`）、`TestOCRPageOrderNumericNotLexicographic`（12 页字典序输入 → `[1 2 3 10 11 12]`）、`TestPageBoundsSpecIgnoresDeclaredTotal`、`TestRenderedPageJobsFallback`；**反向证据** M6（强制回退 `first+i`）⇒ 两条 FAIL（含 `[1 2 3 4 5 6] want [1 2 3 10 11 12]`）、M7（删差异告警）⇒「实际告警=""」FAIL。
**实测口径差异（C1~C5，已钉现状）**：C1 对外 total = `countPDFPages(raw)`，驱动 capPageSpec/pageBounds/OCR 边界；C2 文本路径页码走 `countPDFPages(stripNonTextStreams(decodeFlateStreams(raw)))` 的页对象序列 ⇒ 两个取值面可不等（伪页抬高 total / 正文错位到第 2 页）；C3 `pageBounds` 只按规格算、**不夹 total**（`pageBounds("3-5",2)=(3,5)`）；C4 旧 OCR 页码是 `os.ReadDir` **字典序位置** `first+i`（`page-10` 排在 `page-2` 前、跳页即错位），新口径已修；C5 残留：不可解析名回退 `first+i`（告警）、文本提取全空时 `extractRawText` 兜底分支无页码归属且忽略 `effSpec`、OCR 渲染边界仍由 C1/C3 决定 ⇒ **「文本路径先于渲染拿页数」的结构性约束未消除**（改它要动 `capPageSpec`/`pageBounds` 与既有 fixture，属拍板项）。

**AP6-10 · SupportedSpeakers 持锁做最长 30s HTTP（已修）**：本包/本仓无 singleflight 原语、`go.mod` 无 `golang.org/x/sync` ⇒ 手写最小 single-flight（`speakerFlight`/`speakerFlights` in-flight 表）：`herdsman.go:379-417` 锁内只读缓存/认领/跟随，`fetchSupportedSpeakers` 在**锁外**发 HTTP（`:433`），`waitSpeakerFlight` 对跟随者有界等待（`:113` = `ttsTimeoutForModel+5s`），**成功才回填缓存（失败不缓存 = 可重试）**，失败路径全部 `slog.Warn`（此前是「语音没声、无报错」）。
证据：`TestSupportedSpeakersSingleFlight`（8 并发 → hits==1）、`TestSupportedSpeakersFailureNotCached`（500→nil；重试 hits==2 并回填；第三次命中缓存）、`TestSupportedSpeakersHTTPOutsideLock`（慢 handler 期间 `speakerCacheMu` 可立即获取 + 异模型不阻塞）、`TestSupportedSpeakersFollowerBoundedWait`、`TestSupportedSpeakersSingleFlightPerModel`（两模型各 4 并发 → 共 2 次）；**反向证据** M3（HTTP 放回锁内）⇒「speakerCacheMu 被整段持有（AP6-10 未修）」FAIL、M4（去跟随分支）⇒「应只发 1 次 HTTP，实际 8 次」FAIL、M5（失败也缓存）⇒「got=[] hits=1」FAIL。
余量：`SupportedSpeakers` 无 `ctx` 形参（`internal/app/voice_model_handler.go:229` 直呼，改签名越足迹）⇒ 跟随者用**有界等待**而非 ctx 取消；`-race` 本机不可用（无 gcc）⇒ 并发正确性靠 mutex + `close(channel)` 的 happens-before 结构与确定性用例；`speakerProbeWaitFn` 是包级测试替换点。

**线 B 的两条如实上报（主代理复核后登记为下一批候选）**：
1. **D5 真缺口（未修）**：`upsert_task` **完全没有「工期为负」闸**（`Validate` 有）。实测 `ApplyOps(upsert_task{Duration:-5})` → `err=nil`、落库 `-5`；同一计划走 `Save` → 报「任务 N 工期为负」。不修理由=不属 durationUnit 规则，且会让 Go ops 与 TS `simulateOps`/golden fixture 产生**未登记分叉**（frontend 不在该线足迹）⇒ 建议与前端 fixture 同批改。
2. **任务书纠错（主代理核实）**：任务书说「`internal/tts` 的 `TestEdgeTTS` 依赖外网」——**全仓 `grep TestEdgeTTS` 零命中**，该测试已不存在；而 `.github/workflows/ci.yml` 仍以此为由 `grep -v /internal/tts` **整包排除**（实测该包 66 例真跑全绿，唯一外网项 `TestLiveVoxCPM2` 自带 `HERDSMAN_LIVE=1` 门自动 SKIP）⇒ **过期的 CI 排除 + 不实注释**，登记为下一批候选。

---

---

### 线 1 + 线 A（重启版）· app 与成本检索链（GA6-09 生产代码 + 测试 · AP1-09 · AP7-05）

> **执行事故**：app 线**两个子代理都崩溃**（第一次死在编译未收口；第二次做完了实现、还没交报告就被终止）。**生产代码与用例由子代理落地、由主代理逐行复核与收尾**：主代理补齐第一次遗留的 4 处 import 与 25+ 处测试调用点、`gofmt` 了两处 HEAD 既有 gofmt 债，并**亲自补做第二次缺失的「改坏能红」反向证据**（线 A 未及交付，见本节末表）。

**GA6-09 · 成本检索错误通道（生产代码 + 测试收口）**
- **契约落地**：`cost.Store.List()/Search(...)` → `([]Summary, error)`。语义=「部分数据仍要返回」——`db == nil`（未配置库）→ `(nil, nil)`；`Query` 失败 → `(nil, err)`；单行 `Scan` 失败 → **计数 + 首条错误（`errors.Join`）**，`rows.Err()` → 一并进 error，两者都**照常返回已读到的部分**。
- **绑定面零变更**：`CostSearchPage` 仅**加字段** `Error string \`json:"error,omitempty"\``（成功恒空串 ⇒ 旧前端零行为变化）；`GaeaCostList`/`GaeaCostSearch` 返回裸切片的签名不动 ⇒ 失败走 `reportCostReadError`（`slog.Warn` + **既有** `gaeaNoticeSink`/`gaea-event` Notice 通道，**未新造通道**）。
- **调用点一次性改完（跨 4 包）**：`internal/app` 9 处（compose/graph/import_vision/inquiry/rerank/semantic_index/semantic_search/memory_hub×2）、`tool/builtin` 5 处、`costimport`、`pricefeed`。处置纪律=能上报就**上抛**（compose/graph/import_vision/semantic_search——语义是「读不全就不该给价格带/比价/图谱/索引面的结论」），纯辅助路径 `slog.Warn` + 降级且写明理由（rerank/costimport/pricefeed）。
- **测试**：`internal/gaea/cost/cost_error_test.go`（4 例：nil-db 契约 / Query 失败返错 / **部分 Scan 失败仍返回部分数据 + error** / 成功无错）+ `internal/app/gaea_cost_search_page_error_test.go`（5 例：成功 Error 空 / 读失败 Error 非空 / 部分数据仍保 Items+Total / 带 limit 同路径 / 字段稳定）。

**AP1-09 · S1/S2 阻断判据五份 → 单一判据（已修）**：新增 `internal/app/plan_severity.go` 唯一实现 `severityBlocking(severity)`（`ToUpper+TrimSpace` 后判 `S1|S2` 阻断、`S3|S4|空|未知` 仅提示，严格照抄 `types.PlanProblem.Severity`/`PlanGateReport.Blocking` 既有契约），**五处全部改调并删净本地实现**：`create_chapter_context.go`（`planSeverityBlocking`，写前硬闸 + `planGateError` 明细）、`novel_plan_handler.go`（`planBlockingSeverity`，落盘校验）、`converge_handler.go`（收敛分桶）、`novel_book_health.go`（两处内联）。此前「写前硬闸大小写敏感、落盘校验 Trim+Upper」⇒ 上游若产出 `"s1"`/`" S2 "`，就会出现「保存被拒但生成却起来了」的无法解释的不一致。测试：`plan_severity_test.go`（形态矩阵 / 三闸一致 / 闸门集成）+ `converge_severity_test.go`（收敛分桶）。

**AP7-05 · 图像台账 backend/model 记生效值（已修）**
- `chapter_handler.go`：新增唯一常量 `auroraSceneImageModel` + `effectiveSceneIllustrationModel(cfg)`（`cfg.ImageModel` 优先、空回落 Aurora 档）——**请求与登记同源**；章节配图登记从写死 `"" / "grok-imagine-image-quality"` 改为 `a.cfg.ImageBackend / effectiveSceneIllustrationModel(a.cfg)`（ComfyUI 后端下此前必然记出「不存在的后端 + 不支持模型」的假记录）。
- `characterlib_gen_handler.go`：新增 `portraitImageBinding(cfg, model)`，作为**剧照「请求下发值」与「台账登记值」的唯一来源**（`PortraitBackend` 空→全局绘梦；`PortraitModel` 空→`ImageModel`；显式入参优先），并把原本四处各写一遍的绑定解析（`characterGeneratePortrait`/`buildPortraitClient`/`CharacterGenerateSheet`/登记侧）统一到它。
- `character_portrait_register.go`：`registerCharacterPortraitAsset(..., backend, model)` 新增两参并真正落台账（此前固定空 backend/空 model ⇒ `imagehub_usage.go` 按 `{model,backend}` 分组时同一后端被拆成多行）。
- `image_handler.go`：`generateImageInternal` 登记传**生效 backendType**（override 客户端通道的解析结果）而非全局 `cfg.ImageBackend`；`image_domain.go` 两条无 override 通道的路径（`recordImageHubGenerated`/`ImageCutout`）写明「恒走全局绘梦后端，与改造前逐字节一致」，抠图标 `model="cutout"` 哨兵并注明语义。
- **如实登记的余量**：角色剧照登记的是**当前生效绑定**、而非该次生成的历史绑定（历史值未落盘，无法回溯）——已写进 `character_portrait_register.go` 注释。
- 测试：`imagehub_binding_ledger_test.go`（override 生效值 / 无 override 回落全局）。

**主代理补做的反向证据（线 A 未及交付；逐条实测 FAIL → 还原 → PASS，`TEMP-MUTATE` 残留 = 0）**

| 变异点 | 红出内容（真实输出） | 还原后 |
|---|---|---|
| `page.Error = ""`（读取失败不入 Error） | `--- FAIL: TestGaeaCostSearchPageReportsReadError`「读取失败时 Error 必须非空（否则用户会误判为无匹配条目）」+ `..._PartialDataKeepsItemsAndTotal`「有行解析失败时 Error 必须非空（结果少了几行）」 | PASS（15 例全绿） |
| `severityBlocking` 去掉 `TrimSpace` | `--- FAIL: TestSeverityBlockingForms/两侧空白_S2_也阻断`「severityBlocking(" S2 ") = false, want true」 | PASS |
| 台账 backend 改回全局 `a.cfg.ImageBackend` | `--- FAIL: TestGenerateImageInternalLedgerRecordsEffectiveBinding`「台账 backend = "comfyui", want override 值 herdsman（改前=全局 comfyui）」 | PASS |

### 线 3（重启版线 C）· 前端（FE3-06 · FE5-05 · FE7-09 · GA6-09 消费面）

**半成品收尾（诚实登记）**：上一轮在 `spaceBindings.ts` 的实现**自洽可用**（哨兵类型分层 `BindingSpaceLookup` + 告警去重集 + `isBindingAllowedInSpace` 短路），缺口在 `bridgeFacades.test.ts`——探针工装（Proxy 记录 `window.go.app` 门面属性读取）**装了却没有任何用例使用**，即「未登记名不触达真实绑定」这条最关键反证是空的（且 `goProbeReads` 只写不读，eslint 会报未用）。线 C 按原设计补完、未推倒重来：新增「未登记名在 work/play/shared 三门面一律被拒**且不触达真实绑定**」用例（**含正控**：先证探针是活的，否则负控永真）+「已登记名零告警」用例 + 把 `console.warn` 捕获收进 `beforeEach`（不再刷屏）；并删掉一条**多余的 `eslint-disable-next-line no-console`**（`no-console` 未启用，eslint 由「0 error 1 warning」归零）。

**FE3-06 · 空间门控 fail-open → fail-closed（已修，safety）**：`bindingSpaceOf` 未登记名改返哨兵 `UNKNOWN_BINDING_SPACE = "unknown"`（独立类型 `BindingSpaceLookup`，**不进 `BindingSpace` 联合** ⇒ 编译期就不可能被当成普通空间），`isBindingAllowedInSpace` 见哨兵即 `false`（拒绝发生在 `resolveBinding` **之前** ⇒ 不触达真实绑定），`isSharedBinding` 同源；未登记名走 `warnUnknownBinding` 一次性留痕（模块级去重，前缀 `[spaceBindings]`）。
**「未登记名 → 处置」分类清单**（可复现口径：`bindingNames.ts` 744 − `GAEA_METHOD_FACETS`(565) 映射后的 Go 名 = **184**）：①「真被生产代码依赖 → 需补登记」= **0 项**（双重依据：fail-closed 打开后**全量 vitest 422 文件 / 3647 例 0 失败**；静态核查 `workApp./playApp./sharedApp.` 生产调用点 **8/8 全部已登记**）；②「测试桩/历史遗留 → 修测试」= 0 项；③「Go 有、前端分类表无 → 保持未登记」= **184 项全部归此类**，处置=**不登记**——它们正是 `bridge/drift.ts` 具名排除集里的 legacy 直调族，未登记集**完全包含于** drift.ts 的 198 个具名（未登记独有=0）；塞进分面表等于给门面扩面，正是审计禁止的「为让测试变绿而放宽判据」的镜像 ⇒ 分类锁 565 **未变**。
**残余风险（余量）**：这 184 个名字经**门面**访问现在会被拒（本次目标），但它们真正被调用的路径是**无门控的通用 `app` 代理**（`api/*` 直调族），该面本就不做空间隔离 ⇒ Play/Work 隔离对直调面仍退化为「无门控」，属既有设计边界，本批未扩大范围。

**FE5-05 · 青鸟页 5s 轮询无板块可见门控（已修）**：照既有形态改 `usePollingGate()` + `useBoardActive('weixin')`，tick 体前置 `if (!pollable || !boardVisible) return`，deps 带两个门控（切回即补拉），删掉原手判 `isPageVisible`。证据：`WeixinPage.test.tsx` 17 例（⑮板块不可见：首拉与 3 个空转周期内四个绑定零调用、切回立即恢复；⑯窗口不可见）。**反向证据**：tick 判据去掉板块门控 ⇒ FAIL「expected vi.fn() to not be called at all, but actually been called 1 times」。**余量**：扫码弹窗内的 2s `WhisperWeixinQRStatus` 轮询未加门控（仅弹窗打开时存在、用户主动触发，不属审计所指的 5s 常驻轮询）。

**FE7-09 · 计划文件默认 rel 四处镜像（前端三处已收敛）**：新建**零依赖** `frontend/src/schedule/paths.ts`（只有 `SCHEDULE_DEFAULT_REL`，模块内零 import ⇒ 转引不产生模块边），`gschedSummary.ts`/`store.ts`/`gaea/lib/changes.ts` 三处消费点改 import；`paths.test.ts` 断言「三处消费点取到同一值 + 与 Go `DefaultRelPath` 同字面量 + paths.ts 零依赖」。**反向证据**：把 `changes.ts` 第三处改回漂移字面量 `…V2.gsched.json` ⇒ FAIL。**余量**：Go 侧第四份（`internal/schedule/project.go`）不在前端足迹 ⇒ 跨语言无编译期锁；另**发现审计未列的第 4/第 5 份前端副本**——`gaea/components/ScheduleApplyRollback.tsx:26` 与 `gaea/lib/mock/office/schedule.ts:16` 各写一份同字面量，**登记为下一批候选**。

**GA6-09 前端消费面（已修）**：`lib/types/cost.ts` 的 `CostSearchPageResult` 加 `error?: string`（**未动 `wailsjs` 生成物**——`frontend/wailsjs/` 在 `.gitignore:45`）；`lib/bridge/cost.ts` 新增 `costReadErrorText`（Go 文案自带「读取失败」时不重复加前缀、非 Go 文案补前缀 ⇒ 恒含该字样）；`CostLibraryView.tsx`（首页 + 加载更多 + catch 路径）与 `CostProjectsView.tsx`（EntryPicker top8）在 `error` 非空时给**页面内提示条**（`role=alert` + `data-testid=cost-read-error`/`-retry`），并**区分空态文案**（「读取失败」≠「暂无成本条目」）；`mock/cost.ts` 保持契约一致（成功态不设 error，与 Go `omitempty` 同口径）。证据：7 文件 55 例定向全绿（含 `bridge/cost.test.ts` 3 例）；**反向证据**：成功路径 `setReadError(costReadErrorText(r?.error))` 改成 `setReadError("")` ⇒ FAIL「Unable to find an element by: [data-testid="cost-read-error"]」，还原后 13 例全绿。

**线 C 的全量证据**：`npx tsc -b` = 0；`npx eslint`（18 个改动文件）= 0 problem；定向 vitest 7 文件 55 例全绿；**全量 vitest 两遍**——fail-closed 打开后第一遍 **422 文件 / 3647 例 0 失败**（⇒ 分类清单无①类），四任务落地后最终一遍 **424 文件 / 3660 例全绿（EXIT=0）**；18 个改动文件 BOM=False；4/4 反向证据齐备。

---

## 二、跨线裁决与缝合（主代理收线清单）

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **GA6-09 跨线契约**（线 1 定契约、线 3 消费） | Go 侧 `Search/List → ([]Summary, error)`、`CostSearchPage` **加字段** `error`（绑定签名零变更、漂移闸保持 @744）；前端按**可选字段** `error?: string` 消费，**不手改 `wailsjs` 生成物**（`frontend/wailsjs/` 在 `.gitignore:45`，无需再生）。两线各自独立验证通过 ⇒ 契约缝合完成。 |
| 2 | **`bindings_*.go` 11 份被再生（事故产物）** | 全部按 HEAD 还原：逐文件比对方法集合，确认其余仅「多行折单行」的格式噪声，唯一实质差异是生成器把 `GetSinImageConfig`/`SetSinImageConfig` 从 `ImageB` **错归** `CoreB`（本仓已知的「再生噪声」，门面归属是手工修过的）⇒ **不改归属**，保持 HEAD。 |
| 3 | **HEAD 既有 gofmt 债（收线时暴露）** | `internal/app/bindings_core.go` / `bindings_sin.go` **与 HEAD 逐字节一致却不过 `gofmt`**（各 1 行对齐），而 `.golangci.yml` 启用了 `gofmt` formatter。上一波把它们「再生」成单行版时是干净的 ⇒ 按规矩还原 HEAD 反而把债还原回来。处置：`gofmt -w` 清掉这 **2 行**（纯对齐、零语义、不动方法归属），否则全量门禁在 HEAD 上必红。**这与批次十一的「HEAD 本身不是 lint 干净」同族，是在册复发项。** |
| 4 | **线 2 崩溃遗留：evidence 两文件未 gofmt** | 主代理 `gofmt -w`（`evidence.go` / `evidence_coverage_test.go`），`go test ./internal/gaea/evidence/...` 复跑仍绿。 |
| 5 | **两条线的报告缺失（子代理崩溃）** | 线 A（app）未交报告 ⇒ 主代理**逐行复核产物 + 亲自补做 3 条「改坏能红」**；线 C 交了完整报告（含全量 vitest）⇒ 主代理只做现场复核。**任何「子代理自述」都在本文件里标明来源**，不混同为主代理实测。 |
| 6 | **线 B 上报 D5（`upsert_task` 无「工期为负」闸）** | **不在本批范围**（不属 durationUnit 规则，且改它会让 Go ops 与 TS `simulateOps`/golden fixture 产生未登记分叉）⇒ 登记为**下一批一刀**，要求「前后端同批 + fixture 同步 + 三路径同结论用例」。 |
| 7 | **线 C 上报：审计未列的第 4/第 5 份 rel 副本** | `gaea/components/ScheduleApplyRollback.tsx:26`、`gaea/lib/mock/office/schedule.ts:16` 各一份 `进度计划/当前计划.gsched.json`；两者不在本线足迹 ⇒ 登记下一批并入 `schedule/paths.ts`（连同 Go 侧 `DefaultRelPath` 的跨语言一致性断言）。 |
| 8 | **线 D 上报：前端分类器「有 4 组夹具自测」是不实注释** | `frontend/scripts/classify-vitest-failures.mjs` 零自测（grep `selftest` 零命中），而 `ci.yml` 前端 job 注释这么写 ⇒ 登记下一批（与后端本批落的 9 组/14 断言同款补齐）。 |
| 9 | **线 D 上报：`ci.yml` 仍整包排除 `internal/tts`** | 注释理由是「`TestEdgeTTS` 依赖外网」，但**该测试全仓不存在**（主代理 `grep TestEdgeTTS` 零命中）；`internal/tts` 66 例真跑全绿（唯一外网项 `TestLiveVoxCPM2` 自带 `HERDSMAN_LIVE=1` 门自动 SKIP）⇒ 过期排除白丢覆盖。**登记下一批**（删除排除 + 改注释；backend job 在 windows-latest 上跑，风险面=该包是否有环境依赖，需先在 runner 形态下实跑一次）。 |
| 10 | **`.gitignore` +3 行（线 D 越界并主动声明）** | 主代理**接受**：`go-test.json`/`go-test.err`/`scripts/go-retry-pkgs.txt` 与既有 `frontend/vitest-report.json`、`frontend/vitest-retry-files.txt` 同款（CI 产物不入库），否则每次跑门禁都留未跟踪文件。 |
| 11 | **守卫基线刷新** | `check-primitives --strict` / `check-test-ctors` 在本批新增测试后需复核并按「人工评审 → `--write-baseline`」刷新（见 §三 表）。 |

---

## 三、本批结论与余量

### 3.1 交付与验收

| 项 | 结果 |
|---|---|
| **真 bug / 静默失败** | GA6-09 错误通道（部分数据+error 双态透出到绑定与前端的「读取失败」提示条）· AP1-09 判据单源（消灭「保存被拒但生成却起来了」）· AP7-05 台账记生效 backend/model · IN3-01 两处**真实语义分叉**（patch 中间态 / 先写后校验）· AP6-10 锁内 HTTP + 失败不缓存 · IN3-11 OCR 页码取产物为准 · FE3-06 空间门控 fail-open → **fail-closed** |
| **守卫机器化** | X1-13 遮蔽超基线即红（判闸先于落盘）· X1-12 Go 侧 flaky 分类器（失败包 ⊆ 在册清单才复跑；`--expect-failure` 防静默放行）· X1-10 `check-docs` 计数守卫**根因修复**（原正则从未生效）+ 6 组负例自测 |
| 静态门禁 | `go build ./...`=0、`go vet ./...`=0、`golangci-lint`（pinned v2.14.0）=0 issues（首轮 2 条已在收线时修掉：SA2001 空临界区 / 未用常量） |
| Go 测试 | 触达包全绿：`internal/app` 110.8s + board + `internal/gaea/cost` 1.25s + `costimport`/`pricefeed`/`tool`/`evidence`/`schedule` 1.08s + `docmd` 13.1s + `tts` 1.49s（均为主代理独立复跑输出） |
| 前端 | `tsc -b`=0；`eslint`（18 改动文件）=0 problem；定向 vitest 7 文件 55 例；**全量 vitest 424 文件 / 3660 例全绿**（线 C 实跑，EXIT=0） |
| 守卫 | `check-docs` OK（校验和 447 / 发布说明 636 / 卫生三查全过）· `classify-go-failures --selftest` 14/14 · `check-primitives --strict` / `check-test-ctors` / `check-contract-drift` / `check-bindings-drift@744` 见下方刷新记录 |
| **全量本地快闸** | **`scripts/ci.ps1` → `CI OK` / `CI_SCRIPT_EXIT=0`**（`version-drift` → `clean-tmp` → `go build` → `go vet` → **golangci-lint v2.14.0 = 0 issues** → `go test ./... -count=1` → 前端 `lint` → `build` → **vitest 424 文件 / 3660 例全绿** → E 系列回归守卫 → `check-docs` 卫生四查）。**首次运行在 golangci 卡 2 条**（`tts` 测试 SA2001 空临界区、`novel_bookcover.go` 未用常量），均为两条崩溃线的收尾残留 ⇒ 已修（改 `TryLock` 探针 / 删冗余常量）后重跑全绿。 |

### 3.2 收敛与负债账（本批量化）

| 项 | 变化 | 说明 |
|---|---|---|
| 成本检索错误通道 | 0 → 1（贯穿 4 包 15 调用点 + 2 个绑定出口） | 此前 `Query` 失败/`Scan` 失败/`rows.Err` 三处静默 |
| S1/S2 阻断判据 | **5 → 1** | 删 5 处本地实现（含 2 处大小写口径不同） |
| 台账 backend/model 填法 | **3 套 → 1 套**（`portraitImageBinding` / `effectiveSceneIllustrationModel` 各为单一来源） | 消耗报表不再按假 backend 拆行 |
| 工期单位校验段 | 4 段 → 1 处实现（2 处纯文案差异按 golden 保留） | 2 处真实语义分叉已消除 |
| rel 常量（前端） | 3 → 1（`schedule/paths.ts`） | 另有 2 份新发现的副本入池 |
| 遮蔽闸 | 只打印计数 → **超基线 exit 1** | 基线 2（实测口径） |
| CI 后端 flaky 策略 | 无条件整批重试 → 在册分类器 | 与前端同款形态 + 后端独有两道防线 |

### 3.3 余量（如实登记，未硬凑）

1. **D5 / P1**：`upsert_task` 无「工期为负」闸（真缺口，实测可复现）——需前后端同批。
2. **IN3-11 C5 残留**：不可解析产物名回退 `first+i`、`extractRawText` 兜底分支无页码归属、「文本路径先于渲染拿页数」的结构性约束未消除（属拍板项）。
3. **AP6-10**：`SupportedSpeakers` 无 `ctx` 形参（改签名越足迹）⇒ 跟随者用有界等待（30s/185s）而非 ctx 取消；本机 `-race` 不可用 ⇒ 并发正确性靠 mutex + `close(channel)` 的 happens-before 结构 + 确定性用例。
4. **FE3-06 边界**：184 个 legacy 直调名经**门面**访问已被拒，但其真实调用路径是无门控的 `app` 代理（既有设计边界）。
5. **FE5-05**：扫码弹窗内 2s 轮询未加门控（非审计所指的常驻轮询）。
6. **FE7-09**：Go 侧 `DefaultRelPath` 无跨语言编译期锁；第 4/第 5 份前端副本未并。
7. **AP7-05**：剧照台账记「当前生效绑定」而非该次生成的历史绑定（历史值未落盘）。
8. **X1-13**：遮蔽归因切片靠字典序（总量超基线必红，但明细可能指错条目）；归零（逐条改名/加说明）未做。
9. **X1-12**：复跑粒度是「按包」（与前端「按文件」同粒度）；包内仍可能有一次环境抖动被复跑洗白。
10. **X1-10**：旧线 md 122 份分布、CHANGELOG 24/33、仓库体积数字不纳入对账（已在 `releases/README.md` 显式登记）。
11. **真机/环境欠账**：`-race` 只在 Actions；FE6-02 真机拔麦走查仍未做（承接批次十一）。

---

## 四、留池与待拍板

### 承接批次十一的两条结构拍板项（**仍未拍板，本批未动**）

- **待拍板 A · AP4-01 三套并行编排合并**（DAG / 任务收件箱 / `gaea/tasks` 调度器各自带身份、状态机、取消、进度、落盘）：选项 (a) 短期抽共用原语（M，无用户可见行为变化）/(b) 统一到一个编排内核（L/XL，须先锁黄金测试）/(c) 维持现状。
- **待拍板 B · X1-03 五重 `*core` 嵌入裸构造 panic**：关键事实不变——`bindings_*.go` 全是**生成物**，故「绑定入口铺 `assertAssembled()`」只需改生成器 + 再生，代价在把 **227 处 `&App{}`** 测试桩迁到统一构造器；选项 (a) 生成器注入 + 测试桩迁移（L）/(b) 只做 nil-safe 接入层（M）/(c) 维持 `check-test-ctors.mjs` 守卫。

### 本批新入池（均为「已实测、已定位、但不在本批切口」）

| # | 项 | 依据（本批实测） | 建议刀型 |
|---|---|---|---|
| P1 | **`upsert_task` 无「工期为负」闸** | `ApplyOps(upsert_task{Duration:-5})` → `err=nil` 且落库 `-5`；同计划 `Save` 报「任务 N 工期为负」 | 前后端同批 + golden fixture 同步（线 B 已给证据） |
| P2 | **`ci.yml` 过期排除 `internal/tts`** | `TestEdgeTTS` 全仓零命中；该包 66 例真跑全绿 | CI 单点（删除排除 + 改注释 + runner 形态实跑一次） |
| P3 | **前端分类器无自测但注释自称有** | `grep selftest frontend/scripts/classify-vitest-failures.mjs` 零命中 | 前端守卫（对齐后端 9 组夹具形态） |
| P4 | **计划文件 rel 的第 4/第 5 份前端副本** | `ScheduleApplyRollback.tsx:26`、`mock/office/schedule.ts:16` | 前端收敛（并入 `schedule/paths.ts`）+ 跨语言一致性断言 |
| P5 | **`estimateTokens` 四份口径**（承接批次十一） | 口径真不同，合一会改变预算行为 | 拍板项，非本批 |
| P6 | **GA3-13 的 UI 接线 / GA4-11 前端文案确认**（承接批次十一余量 4/5） | 后端已导出 `MigrationState`、备份回滚已返回非 nil error | 前端单点 |
| P7 | **X1-13 遮蔽归因切片靠字典序** | 本批新增；`shadow[baseline:]` 在「新名排序靠前」时明细可能指错条目（**总量超基线必红**，不静默） | 生成器守卫精化（改成按名字集合对照在册清单） |
| P8 | **`-race` 与真机走查欠账**（承接批次十一余量 6） | 本机无 gcc（`-race` 只在 Actions）；FE6-02 真机拔麦走查未做 | 环境/真机欠账 |

---

## 五、审计原文纠错与遗留登记

| # | 审计原文 | 本批实测 | 定性 |
|---|---|---|---|
| 1 | X1-13「约 **281** 份被遮蔽实现」 | 生成器同口径实测 **2**（`SetFeatureModel`/`SetFeatureModelEnabled` ← core）。281 是**含门面重复的全仓同名对**口径 | **口径不同源**（已写进 `shadowBaselineMax` 注释）；归零仍是余量 |
| 2 | X1-10「校验和 **424** 份」「`releases/*.md` **617** 份」「仓库根有 `gaea-vX.Y.Z.exe`」 | 校验和 **447**（根 427 + archive 20）、发布说明 **636**、仓库根 exe **0 个**（实存 5 在 `releases/`、4 在 `archive/`） | 数字口径未区分根/含 archive；**根因**=`check-docs.mjs` 的正则只认「校验和 N 份」而 README 写「N 份校验和」⇒ **该守卫从未生效**（已修 + 加「认不出即 FAIL」） |
| 3 | X1-12 隐含「前端已有 flaky 分类器形态可抄」 | 形态可抄，但 `ci.yml` 注释所称「前端分类器含 4 组夹具自测」**不实**（零自测代码） | 后端本批落 **9 组夹具 / 14 断言**；前端补夹具登记为下一批 |
| 4 | `ci.yml` 注释「`internal/tts` 的 `TestEdgeTTS` 依赖外部网络，暂时排除」 | **该测试全仓不存在**（`grep` 零命中）；该包 66 例真跑全绿 | **过期注释 + 过期排除**，白丢覆盖 ⇒ 下一批 |
| 5 | GA6-09 在 `findings_all.json` 里是 **P2**（error-swallow/S） | 本批按 P2 收口（round 13 曾把它列进 P1 语境的「独立一刀」，编号口径需按 fid 表为准） | 口径更正 |
| 6 | IN3-11 修法「totalPages 收进单一函数、三条路径合一」 | 实测三口径确有分歧（C1~C5），但**在不改变行为的前提下不可达**（文本路径必须先于渲染拿页数） | 降级为「单点 + 差异检测 + 诚实告警 + 钉现状」；**其中 OCR 页码是真修**（改以 pdftoppm 产物为唯一来源） |
| 7 | AP6-10「持全局锁做最长 **30s** HTTP」 | 实测 `ttsTimeoutForModel` 上限可达 **180s**（不止 30s） | 影响面比审计描述更大；修复后 HTTP 已在锁外 |
| 8 | FE3-06「`api/` 直调族约 **184** 项未登记」 | 实测 `744 − 565 映射后 = 184`，**完全吻合**；未登记集完全包含于 `drift.ts` 的 198 个具名 | 数字吻合；但补一条边界——直调面本就无空间门控（未登记 ≠ 唯一风险面） |
