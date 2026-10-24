# 交接任务书：办公板块 W1 执行（交给 ZCode）

> 2026-10-24 · 签发：DSH（乙方）· 执行：ZCode（甲方，本机 CLI v0.17.0）
> 依据：[决策备忘](zcode-office-decision-memo-2026-10.md) / [交流档](zcode-distill-dialogue-2026-10.md) 第 7 轮
> 状态：三条待签项按"双方共同建议"视为已定，用户已授权交由 ZCode 执行。

---

## 0. 边界（先看这条，越界即停）

**允许改**
- 新增 `internal/office/typography/`（K2）
- 新增 `internal/office/cjk/`（CJK 归一）
- 新增 `internal/office/standard/` 下的检查器文件（Q1，后续刀）
- 新增 `internal/office/qa/`（Q1 规则集，后续刀）

**禁止**
- 改任何既有 `.go` 的行为（只允许"新增文件 + 在主链路加一行调用"这种最小接线）
- 改 `frontend/`（本轮无 UI 任务）
- 改 `internal/gaea/agent/`、`internal/gaea/permission/`（F1/F2 另行派发）
- 跑 `git commit` / `git push`
- 新建任何 `.md` 文档（包括"完成报告"——结果写进最终回复即可）

**每刀的共同验收**
1. `go build ./...` 通过；
2. 新增包的 `go test ./internal/office/<pkg>/...` 通过，且测试是**表驱动 + 正反例**；
3. 不新增第三方依赖；
4. 汇报格式：改了什么文件 → 怎么验证的 → 未验证的部分明说。

---

## 1. 第一刀（本次派发）：K2 中文排版阈值表

**目标**：把散落在技能文本里的排版数值，收敛成一个可断言的 Go 包。

**落点**：`internal/office/typography/`

**必须提供（导出）**

| 名称 | 语义 | 值 |
|---|---|---|
| `TwipsPerCm` | 1cm = 567 twips | 567 |
| `TwipsPerInch` | 1 inch = 1440 twips | 1440 |
| `HalfPointsPerPt` | 1pt = 20 half-points | 20 |
| `CmToTwips(cm float64) int` | 换算 | — |
| `PtToHalfPoints(pt float64) int` | 换算 | — |
| `PtToTwips(pt float64) int` | 1pt = 20 twips | — |
| `A4WidthTwips` / `A4HeightTwips` | A4 版面 | 11906 / 16838 |
| `LineSpacing13x` | 1.3 倍行距（Word `line` 值） | 312 |
| `BodyIndentSimSun` / `BodyIndentYaHei` | 正文首行缩进 2 字符 | 480 / 420 |
| `GBMarginTopMm` / `Bottom` / `Left` / `Right` | GB/T 9704 页边距 | 37 / 35 / 28 / 26 |
| `BannerTableHeightTwips` | 封面外层表高度预算 | 16838 |
| `CoverContentMaxTwips` | 封面内容高度上限 | 15638 |
| `CNSizeNames` | 中文字号 ↔ pt 表（初号~八号 + 小号档） | 见下 |

**中文字号表**（号 → pt，必须完整，供字号转换与机检共用一个真相源）

```
初号 42 / 小初 36 / 一号 26 / 小一 24 / 二号 22 / 小二 18
三号 16 / 小三 15 / 四号 14 / 小四 12 / 五号 10.5 / 小五 9
六号 7.5 / 小六 6.5 / 七号 5.5 / 八号 5
```

并提供 `CNSizeToPt(name string) (float64, bool)` 与 `PtToCNSize(pt float64) (string, bool)`（后者在区间内取最近档，超出返回 false）。

**测试要求**（`typography_test.go`，表驱动）
- 单位换算正例 3 条 + 反例 2 条（负数/零）；
- 字号双向转换：16 个号全量往返一致；`10.5` → 五号；`99` → false；
- GB 页边距与 A4 版心自洽：`A4WidthTwips - CmToTwips(2.8+2.6) ≈ 版心宽`（允许 ±2 twips 误差，测试里写清容差来源）。

**风格**
- 纯常量 + 纯函数，无 I/O，无第三方依赖；
- 注释里写清数值的出处（GB/T 9704-2012 第 5 节 / ZCode `docx/references/common-rules.md`）；
- 中文注释（与仓库现有风格一致）。

---

## 2. 后续刀（等第一刀验收后再派，不要提前做）

| 刀 | 内容 | 落点 | 验收 |
|---|---|---|---|
| K2b | 把 `internal/gaea/skill/gongwen_body.go` 里的硬编码数值改为引用 typography 常量 | 技能层 | 生成文本不变（快照对比） |
| CJK | `NormalizeCJKCompat(string) string`：康熙部首 U+2F00–2FD5 按 NFKC，部首补充 U+2E80–2EF3 按等价表（只收 BMP 100 条） | `internal/office/cjk/` | `⽂`→`文`；不改全角标点；`knowledgeimport.ExtractText` 接线后 PDF 检索命中 |
| K3 | WPS/Word 兼容黑名单（`ShadingType.SOLID`、`\* decimal`、DXA 列宽、缺 `allNoBorders`、exact 行 `verticalAlign`） | `internal/office/qa/` 规则数据 | 5 个反例 docx 全部命中 |
| Q1 | docx 机检 6~7 条（只做可从 OOXML 断言的结构不变量） | `internal/office/qa/` + `standard.Registry` | 每条正反例；对 3 份真实交付物跑不崩 |

---

## 3. 完成标准（第一刀）

- `go build ./...` 绿；
- `go test ./internal/office/typography/...` 绿；
- 新增文件 ≤ 3 个（`typography.go`、`size.go`、`typography_test.go` 或等价拆分）；
- 最终回复给出：文件清单、测试命令与输出摘要、以及"没有验证什么"。

---

## 4. 执行记录（2026-10-24，第一刀已交付）

**结果：K2 已落地并验证。**

| 阶段 | 事实 |
|---|---|
| 首轮（主仓直跑） | ZCode 越界：删除 `internal/gaea/context/provider_adapter.go`、`provider_test.go`（已恢复），并留下半截 `steerItem` 重构导致 `go build` 红。已 `git stash` 存档为 `zcode-aborted-run-2026-10-24`，主仓恢复绿。 |
| 隔离重派 | 改在 `git worktree`（`.tmp/zcode-w1`，分支 `zcode/w1`）里执行，主仓零风险。 |
| 产出 | 新增 `internal/office/typography/{typography.go,size.go,typography_test.go}`，未动既有文件、未跑 git。 |
| 验证 | ZCode **无法执行 Bash**（headless 无权限客户端，只放行 `pwd/ls/echo/find/head`），故由 DSH 代跑：`go test ./internal/office/typography/...` 绿、`go vet` 干净、`go build ./...` 绿。 |
| 修 bug | **任务书自身的错**：`HalfPointsPerPt` 应为 2（1pt=2 半点），原写 20；`PtToTwips` 改用新增的 `TwipsPerPt=20`。已在源码注释里写明这个坑，防止再犯。 |

**结论：后续刀按"ZCode 写代码 → DSH 跑验证"的两段式推进；worktree 隔离是必须项，不是可选项。**

