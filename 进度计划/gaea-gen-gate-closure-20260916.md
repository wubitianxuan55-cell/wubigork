# gaea GenerationGate 闭环收口（生成后自动分析门 + 全书体检）规格书

> 2026-09-16 立项。来源：用户指令「继续」。阶段七 §4 在册项（docs/gaea-stage7-
> plan-2026-09.md）：「GenerationGate 闭环编排收口：写前契约→AI 味闸→评审→
> 修复→审批→结算串成主轴（revolution 文档刀 7/8 未落部分）；全书体检=触发式
> 全量编译（作者点按，跑完出报告——无常驻、无夜间）」。
> 7.3-2 仍等零功能周窗口（≈09-23），本刀为窗口期在册清欠最大项。
> 目标版本：v4.326.0。

## 1. 论点与缺口核对

- revolution 文档未勾项「GenerationGate 完整闭环（生成后自动 Analyze/…）」：
  现状=生成 done 事件只挂去味+AI 味分+角色提取；**分析路（V2 落盘/伏笔同步/
  记忆回填）从不自动发生**——t1-P2 伏笔自动回收、t3-P1 记忆回填两条已建成
  的管道，水源只能靠用户手动在 t7 面板点「分析本章」，闭环名存实亡。
- 全书体检：单章门 RunChapterGate（v4.282）与伏笔 Lint（v4.298）各自孤立，
  没有「点一次、全书确定性体检出一份报告」的聚合面。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 生成后自动门=**异步非阻断**（extractCharactersAfterChapter 同款 `go` 位），默认武装——分析是加法数据（V2/伏笔同步/记忆回填均幂等/带审计），与角色提取同性质；不做开关（用户提成本异议再上一行 kill-switch） |
| Q2 | 自动门只跑**确定性三路 + 分析路**：outlineContract+ChapterQualityIssues+aiTaste（零 LLM）+ analysisAgent.Analyze（一次 LLM）；review/consistency 两路仍留单章门手动（成本纪律，不自动烧模型） |
| Q3 | **分支章跳过分析路**（只跑确定性三路）：V2 落盘/伏笔登记是主线口径，分支 upsert 会污染主线条目（角色提取的历史行为不在本刀修） |
| Q4 | 自动门结果 emit `chapter-gate` 事件（紧凑报告）+ slog；V1 零前端消费——t7 章节分析面板天然是消费面（V2 保证新鲜） |
| Q5 | 全书体检 `RunBookHealthCheck()` =**纯确定性零 LLM**（作者点按，秒级）：逐章 outlineContract+质量+AI 味（白名单豁免后）+字数；伏笔 Lint 复用；V2 覆盖率计数；无常驻无定时 |
| Q6 | 报告 camelCase typed struct（ForeshadowLintReport 同款纪律）；逐章行自然序，头部给聚合（契约问题章数/质量问题章数/最差 AI 味/伏笔 findings/分析覆盖率） |

## 3. 线 B：internal/app

- `create_chapter_handler.go`：done 后 `go a.runAutoGateAfterGeneration(pm, targetNum, content, branch)`。
- 新 `novel_book_health.go`：
  - `runAutoGateAfterGeneration`（Q1-Q3/Q4 口径；analysisAgent nil/失败=warn 降级）；
  - `RunBookHealthCheck() (BookHealthReport, error)`（Q5/Q6；countWrittenChapters
    枚举口径；无项目报错；空书=零章报告不算错）。
- NovelB 门面 +RunBookHealthCheck（704→705）。
- 测试：auto gate（正常/分支跳分析/agent nil 降级三例，断言事件与不 panic）；
  book health（fixture 两章：一干净一脏〔超长段落+契约缺项〕，断言聚合与逐章行；
  空项目零章）。

## 4. 线 C：前端

- bridge/novel.ts：`RunBookHealthCheck(): Promise<BookHealthReportView>` + 视图类型。
- mock/novel.ts：+1 桩（两章样本）。
- 新 `BookHealthPanel.tsx`（CreatePage rail「全书体检」）：头部聚合卡（总章数/
  契约问题章/质量问题章/最差 AI 味/伏笔 findings/分析覆盖）+ 逐章表（章号/
  字数/契约/质量/AI 味，红标越线值）+ 伏笔 findings 列表（复用 Lint 行形状）。
- spaceBindings +1（527→528）；bindingNames 再生（705）。
- 测试：面板渲染聚合与逐章行/mock 契约。

## 5. 验收与门禁

- Go：新测试全绿 + `go build ./...`/vet；
- 前端：tsc -b/eslint/vitest 全量/drift OK@705/计数锁 528；
- 出口：生成一章后 V2/伏笔同步/记忆回填自动发生（测试钉住分析被调）；
  全书体检查一次出聚合报告零 LLM。

## 6. 观察池（本刀不做）

chapter-gate 事件的 UI 消费（toast/面板）；auto 分析 kill-switch；review/
consistency 自动化（成本拍板）；体检含 LLM 深检档；体检报告导出。
