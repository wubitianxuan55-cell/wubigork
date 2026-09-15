# 独立验证员 · 实施前基线（verifier）

> 采集时机：t8 未就绪（依赖 t7）期间，在**任何实施任务落地之前**采集。
> 目的：把后续失败项判定为「本来就失败」还是「本次实施引入」提供不可再生的对照。
> 采集者：verifier（mumu-impl）

## 0. 代码状态锚点

| 项 | 值 |
|---|---|
| branch | `main` |
| HEAD | `dae34beadd394f6fb67186b3b4845b759258b4cb` |
| stash list | 空（无 stash） |
| 未提交改动 | 13 个 ` M` + 8 个 `??`（含 `.agent-teams/`、`docs/distill/` 等） |

`git status --porcelain`（基线，逐字）：

```
 M frontend/src/components/novel/ForeshadowPanel.test.tsx
 M frontend/src/components/novel/ForeshadowPanel.tsx
 M frontend/src/gaea/lib/bindingNames.ts
 M frontend/src/gaea/lib/bridge/novel.ts
 M frontend/src/gaea/lib/mock/novel.ts
 M frontend/src/gaea/lib/spaceBindings.test.ts
 M frontend/src/gaea/lib/spaceBindings.ts
 M internal/app/app_info.go
 M internal/app/bindings_manifest.go
 M internal/app/bindings_novel.go
 M internal/app/bindings_sin.go
 M versioninfo.rc
 M wails.json
?? .agent-teams/
?? .gaea/reports/domain-t2-book-import-distill-2026-09.md
?? .gaea/schedule/
?? docs/distill/
?? docs/mumu-distill/
?? internal/app/novel_foreshadow_lint_handler.go
?? internal/app/novel_foreshadow_lint_handler_test.go
?? "进度计划/"
```

> 注：交接书 §0 称「21 项未提交改动」，实测 `--porcelain` 为 21 行（13 M + 8 ??），**一致**。
> `internal/app/novel_foreshadow_lint_handler.go` 及其测试是**未跟踪新文件**——即 5 类 Lint 是既有的、非本次实施引入。

## 1. 基线命令与结果（全部 exit 0）

| # | 命令 | 结果 | 耗时 |
|---|---|---|---|
| 1 | `go build ./...`（workdir=`C:\AI\wubigrok`） | **exit 0** | 10.4s |
| 2 | `go vet ./...` | **exit 0** | — |
| 3 | `go test -count=1 ./internal/...` | **exit 0，零失败包** | — |
| 4 | `cd frontend && npm test`（vitest run） | **exit 0**，`Test Files 345 passed (345)` / `Tests 2994 passed (2994)` | 155.0s |
| 5 | `cd frontend && npm run build`（`tsc -b && vite build`） | **exit 0**，`✓ built in 28.25s` | 28.3s |

原始输出：
- 后端：`.gaea/baseline-tests-verifier.txt`
- 前端测试：`.gaea/baseline-frontend-test-verifier.txt`
- 前端构建：`.gaea/baseline-frontend-build-verifier.txt`

> 前端基线的 `node.exe : ... NativeCommandError` 文本是 PowerShell 对 vite「chunk > 500 kB」警告经 stderr 输出的包装，**不是失败**；`FRONTEND_BUILD_EXIT=0`。

## 2. 结论：基线是干净的

后端**零失败包**、前端**2994/2994 通过**、build/vet 均 exit 0。

→ **本次实施后出现的任何测试失败，都不能归因于「本来就失败」，必须归因于实施改动**（除非证明确由并发写入的中间态引起，需重跑确认）。

## 3. 「必须保留」机制的基线锚点（已实勘，回归对照用）

| 机制 | 位置（基线已确认存在） |
|---|---|
| `novelcontext` 全局预算 | `internal/novelcontext/novelcontext.go:29` `DefaultMaxRunes = 2000`；`:121` `maxRunes = DefaultMaxRunes` |
| `DeSlopRewrite` 确定性闭环 | `internal/novelstyle/rewrite.go:42` `func DeSlopRewrite(text string, score *TasteScore) (string, *RewriteReport, error)` |
| 稳定伏笔 ID（sha256） | `internal/analysis/analysis.go:213-214` `sha256.Sum256([]byte(category+chapterFile+description))` → `fmt.Sprintf("%s_%s_%x", category, chapterFile[:3], h[:8])` |
| 现有 5 类伏笔 Lint | `internal/app/novel_foreshadow_lint_handler.go`：`ordering`(:76) `status-mismatch`(:82,:86) `dangling`(:92,:97) `stale`(:105) `duplicate`(:113) |
| 章号派生 `chapterNumOf()` | `internal/app/novel_foreshadow_lint_handler.go:45`（交接书 §2.2 指定） |
| F-01 前端真实结构 | `frontend/src/services/` **不存在**（`Test-Path` = False）；真实为 `frontend/src/api/*.ts` 与 `frontend/src/components/novel/**` |

## 4. 待验证清单（t8 就绪后执行）

按 t8 任务书 5 组，逐条附可复现命令与输出证据：
1. 全量编译与静态检查（基线 0 → 必须仍为 0）
2. 全量后端测试（基线零失败 → 新增失败须定位归属）
3. 前端 `npm test` + `npm run build`
4. 规格符合性抽查（每域 3-5 条可机械验证点）
5. 回归风险（上表 6 项必须仍在；`BookReviewResult`/`PlotBranch` 待核）

**严禁**执行任何会丢弃工作区改动的 git 命令（`checkout`/`stash`/`restore`）。
