# MuMuAINovel 蒸馏 → gaea 实施交接书

> 本文件是 mumu-distill 团队的全部可复用产出摘要。**实施团队从这里开始，不必重读六份长规格的开头。**
> 事实源：`clones/MuMuAINovel` @ commit `600be7038539e4dd0568b63e6e534fdcfaf91687`（main, tag v1.5.5, clean）
> 目标工程：`C:\AI\wubigrok`（gaea，Go 1.26.3 + Wails v2 + React/TS）

---

## 0. 工程基线（实施前必读，已实测）

| 项 | 值 |
|---|---|
| 编译 | `go build ./...` → **exit 0，约 4.9s**（基线干净可编译） |
| Go 测试 | `go test ./internal/<pkg>/...`；已有 novel 相关测试 33 个（如 `internal/app/novel_import_handler_test.go`、`novel_foreshadow_lint_handler_test.go`、`create_chapter_context_test.go`、`internal/analysis/foreshadow_sync_test.go`） |
| 前端测试 | `cd frontend && npm test`（vitest run）；novel 相关 17 个测试文件 |
| 前端构建 | `cd frontend && npm run build`（`tsc -b && vite build`） |
| Lint | `golangci-lint run`（配置 `.golangci.yml`）、前端 `npm run lint` |
| 代码规模 | `internal/gaea` 626 文件、`internal/app` 404 文件、`internal/modelengine` 25 文件 |

**⚠️ 工作区不是干净的**：`git status --porcelain` 有 **21 项未提交改动**，其中多项正是实施目标文件：
`frontend/src/components/novel/ForeshadowPanel.tsx`、`ForeshadowPanel.test.tsx`、`frontend/src/gaea/lib/bridge/novel.ts`、`frontend/src/gaea/lib/bindingNames.ts`、`frontend/src/gaea/lib/spaceBindings.ts`、`internal/app/app_info.go`、`internal/app/bindings_manifest.go`、`internal/app/bindings_novel.go`、`internal/app/bindings_sin.go`、`versioninfo.rc`、`wails.json`。
→ **实施时不得 `git checkout`/`git stash` 丢弃这些改动**；改这些文件前先读当前内容。用户已明确选择「直接在工作区改」，未新建分支。

---

## 1. 六域规格索引（实施依据，均已审计）

| 域 | 文件 | 行数 | 内容 |
|---|---|---|---|
| t1 伏笔管理 | `docs/distill/01-foreshadow.md` | 831 | 源码分析（数据模型/状态机/时间线/提醒算法/Prompt/API/前端） |
| t1 规格 | `docs/distill/01-foreshadow-spec.md` | 1129 | gaea 落地规格（DDL/API/Prompt/前端/分阶段/测试） |
| t2 拆书导入 | `docs/distill/02-book-import.md` | 1090 | 流水线/三级分章/编码链/幂等/刀序 T2-1..T2-6 |
| t3 长程一致性 | `docs/distill/03-long-range-consistency.md` | 514 | 上下文双构建器/embedding/20 点差距表/改造点 |
| t4 情节分析 | `docs/distill/04-plot-analysis.md` | 1292 | 9 维分析/三维评分/keyword 锚点/重写三通道/C1-C4 缺口 |
| t5 角色职业 | `docs/distill/05-character-career.md` | 1218 | 状态机/差分算法/职业三层/关系图/14 项改动清单 |
| t6 提示词 | `docs/distill/06-prompt-workshop.md` | 1033 | 模板 Schema/两级三源/工坊协议/15 模板迁移 |
| t7 交叉审校 | `docs/distill/07-crosscheck.md` | 176 | findings F-01~F-07 + 11 条已复核断言 |

行数为权威口径（`ReadAllText + split("\n")`）。**禁止用 `Get-Content` 或 `Measure-Object -Line` 统计行数**——PowerShell 5.1 读 UTF-8 中文文件会丢行，且 `Measure-Object -Line` 不计空行。

---

## 2. 已定调的权威决策（不得推翻，除非显式说明理由）

1. **职业体系 = 方案 C**：新建 Career 数据模型**不做**。把「3主+2副职业树」生成为 `worldview.json` 的新 section（`id:"careers"`），角色生成时作 P1 上下文注入，职业名记进 `Character.Notes`。→ 若后续要做「职业成长玩法」再升级。
2. **伏笔存储 = A1**：不引入 SQLite 主表。主存储保持 `foreshadows.json`，章号由 `chapterNumOf()` 从 `PlantedIn` 文件名派生（现成实现：`internal/app/novel_foreshadow_lint_handler.go:45`）。仅当条目 >2000 时加**可重建的只读**索引表。
3. **伏笔状态机 = 并集 6 态，但 wire 值保持 `"revealed"` 不变**（⚠️ 本条曾写错，2026-09 更正）：
   - 新增 `pending` / `resolved` / `partially_resolved` / `abandoned`，与既有 `planted` / `hinted` / `revealed` 共 6 态。
   - **存储与 JSON 的规范值仍是 `"revealed"`**。**不得**把它改名或归一化为 `"resolved"` —— 全仓有 5 类消费者硬编码该字符串：`internal/types/types.go:186-188`、`frontend/src/components/novel/ForeshadowPanel.tsx`（12 处，含 `:164`/`:167` 回收率统计）、`internal/stats/stats.go:72-73`、`internal/analysis/analysis.go:161` 与 `:301`、`evolution.go:136`、既有测试 `internal/analysis/foreshadow_sync_test.go:85`。改名会让回收率与 lint 计数静默归零。
   - t1 必须提供 `types.NormalizeForeshadowStatus()` 与 `types.IsResolvedStatus()`（后者同时覆盖 `"revealed"` 与 `"resolved"`）。**业务代码禁止用字符串字面量比较伏笔状态，一律走助手。**
   - **落地记录（v4.278.0，Codex 侧收口）**：契约已按本口径落库——`NormalizeForeshadowStatus` 方向为
     `resolved → revealed`，`IsResolvedStatus` 覆盖两者，并已接进 stats 回收率 / 伏笔 lint 计数与状态标签 /
     章节上下文注入闸（已回收不再注入）/ 书籍聚合统计；`SaveForeshadows` 白名单从 3 态扩到并集 6 态
     （别名先归一化再校验）。**t1 在制品早期曾是反向口径（写 `resolved`），v4.278.0 已纠回，后续勿再反转。**
4. **前端落点必须用 gaea 真实结构**（见 F-01）：gaea **没有** `frontend/src/services/` 目录，也没有 `pages/{PromptTemplates,PromptWorkshop,WritingStyles,ProjectList,ChapterAnalysis}.tsx`。真实结构为 `frontend/src/api/*.ts` 与 `frontend/src/components/novel/**`（已有 `components/novel/api/character.ts`、`outlines.ts`、`readingAssistant.ts`）。

---

## 3. 「不要抄」清单（MuMu 的真实缺陷，实施时必须规避）

| # | 缺陷 | 证据 | gaea 对策 |
|---|---|---|---|
| 1 | 职业阶段推进**无章节号水位守卫** → 重跑低章节重复升阶 | `career_update_service.py:129-290`（心理 `:293`、存活 `:214` 有守卫，唯独职业没有） | 新增 `UpdatedChapter` 字段做单调守卫 |
| 2 | 亲密度**子串叠加**：`"不信任"` 同时命中 `"信任"+10` 与 `"不信任"-15` → 净 -5 | `character_state_update_service.py:818-823`（纯 `if keyword in desc: delta += adj`） | 改最长匹配优先 + LLM 直出 `delta_hint` |
| 3 | `member_count` 只增不减（离开不减），靠 fix API 兜底 | `character_state_update_service.py:558`、`projects.py:494/543` | 改为**派生值**，不存储 |
| 4 | `stage_progress` 16 处构造点恒 0，唯一消费方 `CharacterCareerCard.tsx` 是**未被 import 的死组件** | `career.py:58`、`CharacterCareerCard.tsx` | 不实现该字段；若要，先接上 UI |
| 5 | 任务态**纯内存 dict**，无持久化无 TTL，进程重启全失效 | `book_import_service.py:89` | 复用 gaea `tasks` 表注册 `KindBookImport` |
| 6 | 取消在 **AI 反推期间完全不生效**（`_check_cancelled` 只在 `_build_preview` 前后调用） | `book_import_service.py:611/617/627/637` | 每批 AI 调用前后各查一次 ctx |
| 7 | 非流式 `apply_import` 是**死代码**且与流式版行为不一致（角色数 `max(...,8)` vs `max(...,5)`） | `:219` vs `:376` | 单一实现，删除另一版 |
| 8 | `must_resolve` 被 FastAPI 响应模型**静默丢弃** | 服务返回 `foreshadow_service.py:808`、上下文 `chapter_context_service.py:1006`/`:1814` 用该键；`schemas/foreshadow.py:190-197` 无该字段 | 显式字段 + 单测覆盖 |
| 9 | `source_analysis_id` 恒 NULL（创建不写、清理却依赖）→ 一条清理分支是死代码 | `:1433-1455` vs `:1019-1020` | 要么写入，要么删除该分支 |
| 10 | `AI_DENOISING` 模板**全库无定义** → `/api/polish` 必然 500（前端路由已注释，是死代码） | `api/polish.py:40`、`:114`（t4/t6 **双向互证**） | 不迁移该链路；去 AI 味改用 gaea 现有 `novelstyle.DeSlopRewrite` |
| 11 | 局部重写**直接拼接落库、无快照无 undo**；整章重写有快照但**无 restore 路由** | `chapters.py:5224`、`RegenerationTask.original_content` | 实施重写必须自带版本恢复 |
| 12 | 用户模板保存**零校验** + 运行时缺失变量即抛错 → 写错一个 `{变量}` 即整条链路 500 | `api/prompt_templates.py:204-282`、`prompt_service.py:2609-2624` | 保存时校验占位符；缺失变量保留原文 + warning，不抛错 |
| 13 | `parameters` 字段**不可信**（36 模板中 11 个与真实占位符不符，7 个有运行时风险） | t6 §14 验证脚本 | 不用该字段做校验；以真实扫描占位符为准 |
| 14 | `len()` 数字节而非字符 → 「≥200 字前言阈值 / <300 字短章告警」在中文上失真 | `txt_parser_service` 相关 | 用 `utf8.RuneCountInString` |

---

## 4. 「gaea 已优于 MuMu、必须保留」清单

1. `novelcontext.Render(maxRunes)` 全局 token/字符预算（`internal/novelcontext/novelcontext.go:29 DefaultMaxRunes=2000`、`:119`）——**MuMu 全后端零预算**（grep `tiktoken|max_context|context_limit|token_budget|count_tokens` 零命中）。
2. `novelstyle.DeSlopRewrite` 确定性闭环（词长降序替换 + 标点归一 + 复测打分）。
3. 句级 LLM 重写安全闸：`internal/app/novel_llm_rewrite_handler.go:125-139`（篇幅漂移 >50% 放弃该句；复测不降不落盘）。
4. 稳定伏笔 ID：`sha256(category+chapterFile+description)[:8]`（MuMu 用序号 id，只能先删后建）。
5. `BookReviewResult`（letter/total_score/peaks/valleys/arc_completions）与 `PlotBranch`（6 字段含 foreshadow_impact）——MuMu 无对应。
6. gaea 现有 5 类伏笔 Lint 检查（ordering / status-mismatch / dangling / stale / duplicate）——MuMu 完全没有。**升级是扩展，不是替换。**
7. **伏笔状态的 `"revealed"` wire 值**（同 §2-3，2026-09 补充）—— 这是 gaea 既有约定，前端回收率、后端统计、分析写回与既有测试都依赖它。任何"归一化为 resolved"的改法都是破坏性变更。

---

## 4.5 文件归属划界（防并发冲突，实施时严格遵守）

系统对 `inScope` 做重叠检测，我在建任务时按「**每个文件只归一个任务**」划分。关键的刻意的划界：

| 文件 / 目录 | 归属 | 说明 |
|---|---|---|
| `internal/types/` | **t1 独占** | 全链前置；其余各域只引用不改 |
| `internal/app/create_chapter_context.go` | t4 | prevSummary 预算修复 |
| `internal/app/create_chapter_handler.go` | t6 | substituteWordCount 同步 |
| `internal/app/novel_import_handler.go` | t2 | 拆书委托改造 |
| `internal/app/gaea_tasks.go` | t2（**批准的最小外延**） | 仅加 `KindBookImport` 注册与 Priority map 一条；不得重构既有逻辑 |
| `internal/app/novel_llm_rewrite_handler.go` | t5 | 重写版本恢复 |
| `internal/plotanalysis/` | t5（**新建包**） | 刻意新建以避开与 `internal/analysis/` 的目录冲突 |
| `internal/analysis/`（除 `foreshadow_sync_test.go`） | t4 | |
| `internal/analysis/foreshadow_sync_test.go` | t3 | |
| `frontend/src/components/novel/` | t7（单一任务统一接线） | 避免多成员并发写同一目录 |
| `prompts/` | t6 | |

---

## 5. 可直接借鉴的高价值机制（实施重点）

- **t2 三级分章**：强正则 → 弱模式（≤25 字、无标点、前后空行）→ 固定窗口 3000-5000 字 + 标点边界兜底（`txt_parser_service.py:15-19,119-133,135-168`）。**gaea 现状只有强正则**（`internal/app/novel_import_handler.go:183-185` 识别不到就退化成单章「全文」）——这是最直接短板。
- **t2 JSON 负反馈重试**：重试时把上次失败回复截 200 字拼进 prompt（`ai_service.py:689-690`）+ 显式 `expected_type`。
- **t2 大纲字段级归一化**：`title`/`chapter_number` 一律丢弃用输入值；其余 `raw or fallback` + 长度夹取；批内**位置对齐**；数量不符则整批回退规则结构（`book_import_service.py:1291-1332,1375-1466`）。
- **t4 M1**：单次 LLM 调用产出「9 维分析 + 三维分档评分（`overall=(P+E+C)/3`）+ keyword 强制 8-25 字原文复制」+ 建议数量与 overall 硬联动（禁止打 7-8 分安全分）。
- **t4 M2**：`AnalysisTask` 三重超时（硬 600s + 独立终态事务幂等重试 3 次 + 陈旧恢复 720s）+ 重试重置 `started_at`。
- **t4 M3**：分批展开 `batch_size=5` + 跨批差异化上下文 + `sub_index` 全局重写 + 双序重排。
- **t6 迁移成本极低**：15 个 gaea 模板全部保留原名原结构；**全库唯一占位符改动**是把 `create-chapter.json` 的 `{word_count}` 改为 `{{word_count}}`，并同步 `internal/app/create_chapter_handler.go:86-90` 的 `substituteWordCount`。

---

## 6. 已知工具陷阱（踩过一次，别再踩）

- **行数统计**：见 §0 末尾。用 `ReadAllText + split("\n")` 或 read 工具。
- **PowerShell 5.1 读 UTF-8 中文**：`Get-Content` 会丢行（本机实测：同一文件 `Get-Content`=131 行 vs 权威 170 行）。涉及中文内容处理请用 read/write 工具。
- **自检义务**：captain 本人曾因上述陷阱误判 t4/t5「行数虚报」，已在 `07-crosscheck.md` 公开撤回。**任何「N 行/N 字节不符」的结论，先怀疑自己的工具。**
