# gaea 项目代码图谱（2026-11）

> 由 `go run scripts/code_atlas_doc.go` 从 `go list -json ./...` + 文件盘点单源生成（module `github.com/gaea/gaea`，go 1.26）。§1–§6 为机械段勿手改；§7 接缝导读与 §8 再生成说明随认知更新改模板重导。统计口径：Go 源码=GoFiles+CgoFiles 行数（非测试），测试=Test/XTestGoFiles；前端=ts/tsx/css 行数。`node_modules` 内杂包已滤。

> 全仓 164 个 Go 包 · 源码 264,830 行 · 测试 210,821 行（测试/源码 0.80）。

## 1. 分层总览

```
                ┌─ scripts 工具带（gen_bindings/评测/迁移，零被引）
 入口 main ──▶ 壳层 internal/app（板块装配/wails bindings）
                ├─▶ 内核·装配执行 internal/gaea/{boot,control,tool,agent,evidence,…}
                │        └─▶ 内核·共享库 internal/gaea/{db,memory,skill,proc,…}
                └─▶ 领域库 internal/{office,novel 系,engine 系,pantheon,schedule,costimport,…}
```

| 层 | 包数 | 源码 LOC | 测试 LOC |
|---|---:|---:|---:|
| 入口 | 1 | 124 | 44 |
| 壳层·板块装配 | 2 | 76,081 | 69,053 |
| 内核·装配执行 | 22 | 46,041 | 38,938 |
| 内核·共享库 | 54 | 45,006 | 35,489 |
| 办公域 | 12 | 4,407 | 2,237 |
| 小说域 | 21 | 34,381 | 16,850 |
| 引擎域 | 8 | 15,662 | 15,572 |
| 众神殿 | 1 | 11,510 | 8,831 |
| 领域库 | 33 | 27,454 | 23,490 |
| scripts 工具带 | 10 | 4,164 | 317 |
| **合计** | **164** | **264,830** | **210,821** |

层间依赖边矩阵（行=引用方 → 列=被引方，共 474 条内部边）：

| ↓从 / 到→ | 入口 | 壳层·板块装配 | 内核·装配执行 | 内核·共享库 | 办公域 | 小说域 | 引擎域 | 众神殿 | 领域库 | scripts 工具带 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 入口 |  | 1 |  |  |  |  |  |  | 1 |  |
| 壳层·板块装配 |  | 1 | 10 | 42 | 7 | 19 | 8 | 1 | 31 |  |
| 内核·装配执行 |  |  | 47 | 69 | 1 |  |  |  | 9 |  |
| 内核·共享库 |  |  | 12 | 57 | 1 |  | 1 |  | 8 |  |
| 办公域 |  |  |  | 2 | 4 |  |  |  | 1 |  |
| 小说域 |  |  |  | 11 |  | 10 | 5 |  | 36 |  |
| 引擎域 |  |  |  | 3 |  |  | 2 |  | 8 |  |
| 众神殿 |  |  |  | 2 |  |  |  |  |  |  |
| 领域库 |  |  |  | 15 |  | 3 | 1 |  | 29 |  |
| scripts 工具带 |  |  | 1 | 3 |  | 2 | 6 | 1 | 3 |  |

## 2. 分层不变量检查（再生成时复算）

- ✅ 内核（internal/gaea/**）不 import 壳层（internal/app）——0 处违例
- ✅ scripts 工具带零被引——0 处违例
- ✅ 壳层入边只来自入口 main（壳层内部互引不计）——实际入边源: (main), internal/app

## 3. 包清单（层内按源码 LOC 降序）

| 包 | 层 | 源码 LOC | 测试 LOC | 被引 | 内部出边 |
|---|---|---:|---:|---:|---:|
| `(main)` | 入口 | 124 | 44 | 0 | 2 |
| `internal/app` | 壳层·板块装配 | 75,660 | 68,882 | 1 | 119 |
| `internal/app/board` | 壳层·板块装配 | 421 | 171 | 1 | 0 |
| `internal/gaea/agent` | 内核·装配执行 | 13,224 | 12,452 | 3 | 20 |
| `internal/gaea/tool/builtin` | 内核·装配执行 | 11,955 | 8,146 | 3 | 28 |
| `internal/gaea/control` | 内核·装配执行 | 4,656 | 4,149 | 2 | 23 |
| `internal/gaea/contextview` | 内核·装配执行 | 2,782 | 1,795 | 1 | 2 |
| `internal/gaea/agent/session` | 内核·装配执行 | 2,684 | 3,290 | 8 | 4 |
| `internal/gaea/tool` | 内核·装配执行 | 1,627 | 1,493 | 12 | 2 |
| `internal/gaea/plugin` | 内核·装配执行 | 1,562 | 748 | 2 | 3 |
| `internal/gaea/boot` | 内核·装配执行 | 1,443 | 1,620 | 1 | 27 |
| `internal/gaea/evidence` | 内核·装配执行 | 1,099 | 1,036 | 4 | 0 |
| `internal/gaea/hook` | 内核·装配执行 | 935 | 825 | 2 | 3 |
| `internal/gaea/context` | 内核·装配执行 | 824 | 957 | 3 | 2 |
| `internal/gaea/permission` | 内核·装配执行 | 556 | 835 | 3 | 1 |
| `internal/gaea/event` | 内核·装配执行 | 448 | 253 | 8 | 2 |
| `internal/gaea/agent/render` | 内核·装配执行 | 400 | 121 | 1 | 3 |
| `internal/gaea/agent/textutils` | 内核·装配执行 | 379 | 113 | 2 | 0 |
| `internal/gaea/command` | 内核·装配执行 | 296 | 455 | 2 | 2 |
| `internal/gaea/sandbox` | 内核·装配执行 | 279 | 211 | 3 | 1 |
| `internal/gaea/agent/toolguard` | 内核·装配执行 | 260 | — | 1 | 0 |
| `internal/gaea/agent/testutil` | 内核·装配执行 | 199 | 56 | 0 | 1 |
| `internal/gaea/agent/cache` | 内核·装配执行 | 196 | 194 | 1 | 1 |
| `internal/gaea/agent/budget` | 内核·装配执行 | 138 | 80 | 1 | 1 |
| `internal/gaea/spaces` | 内核·装配执行 | 99 | 109 | 7 | 0 |
| `internal/gaea/workcost` | 内核·共享库 | 5,714 | 3,426 | 2 | 1 |
| `internal/gaea/memory` | 内核·共享库 | 5,355 | 5,024 | 5 | 5 |
| `internal/gaea/config` | 内核·共享库 | 2,931 | 2,622 | 9 | 5 |
| `internal/gaea/cost` | 内核·共享库 | 2,203 | 2,103 | 8 | 2 |
| `internal/gaea/browser` | 内核·共享库 | 2,060 | 1,437 | 2 | 1 |
| `internal/gaea/skill` | 内核·共享库 | 1,707 | 1,587 | 3 | 5 |
| `internal/gaea/workbench` | 内核·共享库 | 1,655 | 327 | 1 | 4 |
| `internal/gaea/trajectory` | 内核·共享库 | 1,545 | 1,218 | 1 | 3 |
| `internal/gaea/costref` | 内核·共享库 | 1,341 | 558 | 2 | 3 |
| `internal/gaea/tasks` | 内核·共享库 | 1,258 | 1,938 | 1 | 0 |
| `internal/gaea/costimport` | 内核·共享库 | 1,233 | 536 | 1 | 3 |
| `internal/gaea/knowledge` | 内核·共享库 | 1,196 | 1,005 | 3 | 4 |
| `internal/gaea/cache` | 内核·共享库 | 1,186 | 642 | 2 | 3 |
| `internal/gaea/db` | 内核·共享库 | 989 | 763 | 5 | 0 |
| `internal/gaea/cad` | 内核·共享库 | 892 | 366 | 2 | 2 |
| `internal/gaea/provider` | 内核·共享库 | 867 | 1,248 | 17 | 1 |
| `internal/gaea/dag` | 内核·共享库 | 807 | 697 | 1 | 1 |
| `internal/gaea/backup` | 内核·共享库 | 747 | 601 | 1 | 1 |
| `internal/gaea/costinquiry` | 内核·共享库 | 721 | 499 | 2 | 1 |
| `internal/gaea/semantic` | 内核·共享库 | 659 | 364 | 3 | 1 |
| `internal/gaea/pricefeed` | 内核·共享库 | 649 | 333 | 1 | 2 |
| `internal/gaea/i18n` | 内核·共享库 | 581 | 114 | 2 | 0 |
| `internal/gaea/genui` | 内核·共享库 | 578 | 530 | 6 | 0 |
| `internal/gaea/attachment` | 内核·共享库 | 567 | 353 | 1 | 1 |
| `internal/gaea/jobs` | 内核·共享库 | 539 | 673 | 4 | 2 |
| `internal/gaea/knowledgeimport` | 内核·共享库 | 532 | 214 | 1 | 5 |
| `internal/gaea/retrieval` | 内核·共享库 | 470 | 280 | 3 | 1 |
| `internal/gaea/wssearch` | 内核·共享库 | 411 | 236 | 1 | 4 |
| `internal/gaea/largefile` | 内核·共享库 | 402 | 206 | 2 | 3 |
| `internal/gaea/costproject` | 内核·共享库 | 354 | 249 | 3 | 1 |
| `internal/gaea/sessionquery` | 内核·共享库 | 354 | 215 | 1 | 2 |
| `internal/gaea/wspath` | 内核·共享库 | 341 | 386 | 10 | 0 |
| `internal/gaea/coststage` | 内核·共享库 | 333 | 511 | 1 | 0 |
| `internal/gaea/bm25` | 内核·共享库 | 291 | 380 | 3 | 0 |
| `internal/gaea/vision` | 内核·共享库 | 280 | 352 | 4 | 0 |
| `internal/gaea/filewatch` | 内核·共享库 | 279 | 215 | 1 | 1 |
| `internal/gaea/provider/bridge` | 内核·共享库 | 267 | 637 | 1 | 2 |
| `internal/gaea/fileutil/encoding` | 内核·共享库 | 245 | — | 4 | 0 |
| `internal/gaea/archive` | 内核·共享库 | 242 | 221 | 2 | 0 |
| `internal/gaea/pins` | 内核·共享库 | 213 | 117 | 2 | 2 |
| `internal/gaea/billing` | 内核·共享库 | 200 | 382 | 1 | 1 |
| `internal/gaea/factbase` | 内核·共享库 | 195 | 100 | 1 | 1 |
| `internal/gaea/fileindex` | 内核·共享库 | 188 | 64 | 1 | 3 |
| `internal/gaea/strutil` | 内核·共享库 | 187 | 704 | 23 | 0 |
| `internal/gaea/outputstyle` | 内核·共享库 | 180 | 94 | 1 | 1 |
| `internal/gaea/search` | 内核·共享库 | 177 | 107 | 2 | 0 |
| `internal/gaea/filefilter` | 内核·共享库 | 173 | 70 | 2 | 1 |
| `internal/gaea/proc` | 内核·共享库 | 171 | 166 | 9 | 0 |
| `internal/gaea/fileutil` | 内核·共享库 | 136 | 278 | 23 | 0 |
| `internal/gaea/spill` | 内核·共享库 | 128 | 99 | 2 | 0 |
| `internal/gaea/secure` | 内核·共享库 | 114 | 53 | 3 | 0 |
| `internal/gaea/textsim` | 内核·共享库 | 80 | — | 3 | 0 |
| `internal/gaea/frontmatter` | 内核·共享库 | 65 | 157 | 4 | 0 |
| `internal/gaea/nilutil` | 内核·共享库 | 18 | 32 | 5 | 0 |
| `internal/office/xlsxedit` | 办公域 | 821 | 503 | 1 | 1 |
| `internal/office/docxedit` | 办公域 | 625 | 417 | 1 | 1 |
| `internal/office/xlsxpreview` | 办公域 | 594 | 303 | 1 | 0 |
| `internal/office/pptxedit` | 办公域 | 462 | 189 | 1 | 1 |
| `internal/office/qa` | 办公域 | 350 | 176 | 1 | 2 |
| `internal/office/ooxml` | 办公域 | 348 | 184 | 3 | 1 |
| `internal/office/standard` | 办公域 | 329 | 104 | 2 | 0 |
| `internal/office/crosslink` | 办公域 | 285 | 103 | 1 | 0 |
| `internal/office/anchor` | 办公域 | 282 | 119 | 1 | 0 |
| `internal/office/cjk` | 办公域 | 150 | 39 | 1 | 0 |
| `internal/office/typography` | 办公域 | 146 | 100 | 0 | 0 |
| `internal/office` | 办公域 | 15 | — | 0 | 1 |
| `internal/whisper` | 小说域 | 18,039 | 7,628 | 4 | 5 |
| `internal/booksource` | 小说域 | 2,432 | 1,716 | 1 | 2 |
| `internal/whisper/db/repos` | 小说域 | 1,958 | 557 | 1 | 2 |
| `internal/novelstyle` | 小说域 | 1,943 | 993 | 1 | 1 |
| `internal/characterlib` | 小说域 | 1,610 | 1,093 | 3 | 4 |
| `internal/novelreview` | 小说域 | 1,238 | 423 | 1 | 1 |
| `internal/novelcontext` | 小说域 | 1,038 | 620 | 1 | 6 |
| `internal/bookimport` | 小说域 | 1,013 | 688 | 2 | 2 |
| `internal/character` | 小说域 | 836 | 309 | 1 | 8 |
| `internal/whisper/db` | 小说域 | 683 | 167 | 2 | 0 |
| `internal/outline` | 小说域 | 677 | 600 | 1 | 7 |
| `internal/novelgate` | 小说域 | 530 | 609 | 1 | 2 |
| `internal/rewrite` | 小说域 | 412 | 380 | 1 | 2 |
| `internal/characterstate` | 小说域 | 387 | 222 | 2 | 2 |
| `internal/narrative` | 小说域 | 359 | 290 | 1 | 2 |
| `internal/scene` | 小说域 | 279 | 141 | 2 | 2 |
| `internal/worldview` | 小说域 | 252 | 41 | 1 | 5 |
| `internal/chapter` | 小说域 | 222 | — | 1 | 6 |
| `internal/style` | 小说域 | 218 | 159 | 2 | 3 |
| `internal/maturecraft` | 小说域 | 147 | 100 | 3 | 0 |
| `internal/noveltext` | 小说域 | 108 | 114 | 2 | 0 |
| `internal/ai` | 引擎域 | 4,806 | 5,361 | 8 | 6 |
| `internal/modelengine` | 引擎域 | 3,796 | 4,650 | 2 | 3 |
| `internal/voice/realtime` | 引擎域 | 3,682 | 2,318 | 3 | 0 |
| `internal/tts` | 引擎域 | 1,460 | 1,525 | 4 | 1 |
| `internal/herdsman` | 引擎域 | 750 | 915 | 1 | 1 |
| `internal/voice/native` | 引擎域 | 595 | 368 | 3 | 0 |
| `internal/voice` | 引擎域 | 308 | 289 | 1 | 1 |
| `internal/ocr` | 引擎域 | 265 | 146 | 1 | 1 |
| `internal/pantheon` | 众神殿 | 11,510 | 8,831 | 2 | 2 |
| `internal/schedule` | 领域库 | 4,461 | 4,977 | 3 | 1 |
| `internal/docmd` | 领域库 | 3,617 | 3,171 | 8 | 1 |
| `internal/channels/weixin` | 领域库 | 2,296 | 2,810 | 1 | 2 |
| `internal/types` | 领域库 | 2,109 | 1,228 | 20 | 0 |
| `internal/config` | 领域库 | 1,969 | 1,158 | 11 | 2 |
| `internal/project` | 领域库 | 1,703 | 1,013 | 15 | 4 |
| `internal/analysis` | 领域库 | 1,685 | 1,232 | 1 | 8 |
| `internal/graph` | 领域库 | 800 | 456 | 2 | 2 |
| `internal/export` | 领域库 | 762 | 694 | 1 | 3 |
| `internal/auth` | 领域库 | 660 | 842 | 2 | 3 |
| `internal/promptstore` | 领域库 | 616 | 789 | 1 | 1 |
| `internal/authorprefs` | 领域库 | 516 | 164 | 1 | 1 |
| `internal/chat` | 领域库 | 504 | 301 | 1 | 0 |
| `internal/core` | 领域库 | 482 | 137 | 2 | 3 |
| `internal/httpbridge` | 领域库 | 444 | 290 | 2 | 0 |
| `internal/netclient` | 领域库 | 402 | 687 | 15 | 1 |
| `internal/visual` | 领域库 | 389 | 32 | 1 | 2 |
| `internal/intent` | 领域库 | 374 | 308 | 1 | 0 |
| `internal/stats` | 领域库 | 364 | 172 | 1 | 2 |
| `internal/prompt` | 领域库 | 342 | 493 | 7 | 0 |
| `internal/util` | 领域库 | 334 | 522 | 16 | 1 |
| `internal/memory` | 领域库 | 300 | 294 | 1 | 3 |
| `internal/snapshot` | 领域库 | 295 | 127 | 1 | 1 |
| `internal/assistant` | 领域库 | 282 | 240 | 2 | 2 |
| `internal/skilldistill` | 领域库 | 275 | 323 | 1 | 0 |
| `internal/screen` | 领域库 | 242 | 122 | 2 | 0 |
| `internal/context` | 领域库 | 232 | 90 | 1 | 3 |
| `internal/netclient/sysproxy` | 领域库 | 208 | 50 | 1 | 0 |
| `internal/skill` | 领域库 | 191 | 170 | 1 | 0 |
| `internal/routesuggest` | 领域库 | 184 | 171 | 1 | 0 |
| `internal/taskinbox` | 领域库 | 181 | 324 | 1 | 0 |
| `internal/search` | 领域库 | 152 | — | 1 | 2 |
| `internal/skillstats` | 领域库 | 83 | 103 | 1 | 0 |
| `scripts/voice_v2/ws_probe` | scripts 工具带 | 1,402 | — | 0 | 2 |
| `scripts/gen_bindings` | scripts 工具带 | 1,191 | 130 | 0 | 0 |
| `scripts/attention_report` | scripts 工具带 | 518 | 187 | 0 | 1 |
| `scripts/voice_design` | scripts 工具带 | 412 | — | 0 | 3 |
| `scripts/voice_backfill` | scripts 工具带 | 210 | — | 0 | 3 |
| `scripts/migrate_workcost` | scripts 工具带 | 125 | — | 0 | 3 |
| `scripts/voice_v2/vadcheck` | scripts 工具带 | 104 | — | 0 | 1 |
| `scripts/gen_bindings/shadow-diag` | scripts 工具带 | 90 | — | 0 | 0 |
| `scripts/pantheon_eval` | scripts 工具带 | 88 | — | 0 | 2 |
| `scripts/voice_v2/diagserver` | scripts 工具带 | 24 | — | 0 | 1 |

## 4. 内部依赖明细（按层分组；→ 后为引用的内部包）

### 入口

- `(main)` → internal/app, internal/httpbridge

### 壳层·板块装配

- `internal/app` → internal/ai, internal/analysis, internal/app/board, internal/assistant, internal/auth, internal/authorprefs, internal/bookimport, internal/booksource, internal/channels/weixin, internal/chapter, internal/character, internal/characterlib, internal/chat, internal/config, internal/context, internal/core, internal/docmd, internal/export, internal/gaea/agent, internal/gaea/agent/session, internal/gaea/attachment, internal/gaea/backup, internal/gaea/boot, internal/gaea/browser, internal/gaea/cad, internal/gaea/config, internal/gaea/contextview, internal/gaea/control, internal/gaea/cost, internal/gaea/costimport, internal/gaea/costinquiry, internal/gaea/costproject, internal/gaea/costref, internal/gaea/coststage, internal/gaea/dag, internal/gaea/db, internal/gaea/event, internal/gaea/evidence, internal/gaea/factbase, internal/gaea/filefilter, internal/gaea/fileindex, internal/gaea/fileutil, internal/gaea/filewatch, internal/gaea/genui, internal/gaea/i18n, internal/gaea/knowledge, internal/gaea/knowledgeimport, internal/gaea/largefile, internal/gaea/memory, internal/gaea/pins, internal/gaea/pricefeed, internal/gaea/proc, internal/gaea/provider, internal/gaea/provider/bridge, internal/gaea/retrieval, internal/gaea/secure, internal/gaea/semantic, internal/gaea/skill, internal/gaea/spaces, internal/gaea/strutil, internal/gaea/tasks, internal/gaea/textsim, internal/gaea/tool, internal/gaea/tool/builtin, internal/gaea/trajectory, internal/gaea/vision, internal/gaea/workbench, internal/gaea/workcost, internal/gaea/wspath, internal/gaea/wssearch, internal/graph, internal/herdsman, internal/httpbridge, internal/intent, internal/maturecraft, internal/memory, internal/modelengine, internal/narrative, internal/netclient, internal/novelcontext, internal/novelgate, internal/novelreview, internal/novelstyle, internal/ocr, internal/office/crosslink, internal/office/docxedit, internal/office/pptxedit, internal/office/qa, internal/office/standard, internal/office/xlsxedit, internal/office/xlsxpreview, internal/outline, internal/pantheon, internal/project, internal/prompt, internal/promptstore, internal/rewrite, internal/routesuggest, internal/scene, internal/schedule, internal/screen, internal/search, internal/skill, internal/skilldistill, internal/skillstats, internal/stats, internal/style, internal/taskinbox, internal/tts, internal/types, internal/util, internal/visual, internal/voice, internal/voice/native, internal/voice/realtime, internal/whisper, internal/whisper/db, internal/whisper/db/repos, internal/worldview

### 内核·装配执行

- `internal/gaea/agent` → internal/gaea/agent/budget, internal/gaea/agent/cache, internal/gaea/agent/render, internal/gaea/agent/session, internal/gaea/agent/textutils, internal/gaea/agent/toolguard, internal/gaea/archive, internal/gaea/context, internal/gaea/event, internal/gaea/evidence, internal/gaea/genui, internal/gaea/jobs, internal/gaea/memory, internal/gaea/nilutil, internal/gaea/provider, internal/gaea/spaces, internal/gaea/spill, internal/gaea/strutil, internal/gaea/tool, internal/gaea/wspath
- `internal/gaea/tool/builtin` → internal/docmd, internal/gaea/browser, internal/gaea/cad, internal/gaea/config, internal/gaea/cost, internal/gaea/costproject, internal/gaea/costref, internal/gaea/db, internal/gaea/evidence, internal/gaea/fileutil, internal/gaea/fileutil/encoding, internal/gaea/genui, internal/gaea/jobs, internal/gaea/knowledge, internal/gaea/memory, internal/gaea/retrieval, internal/gaea/sandbox, internal/gaea/semantic, internal/gaea/spaces, internal/gaea/spill, internal/gaea/strutil, internal/gaea/tool, internal/gaea/vision, internal/gaea/wspath, internal/netclient, internal/office/anchor, internal/schedule, internal/screen
- `internal/gaea/control` → internal/docmd, internal/gaea/agent, internal/gaea/agent/session, internal/gaea/billing, internal/gaea/command, internal/gaea/config, internal/gaea/context, internal/gaea/event, internal/gaea/hook, internal/gaea/i18n, internal/gaea/jobs, internal/gaea/memory, internal/gaea/nilutil, internal/gaea/permission, internal/gaea/plugin, internal/gaea/provider, internal/gaea/skill, internal/gaea/spaces, internal/gaea/strutil, internal/gaea/tool, internal/gaea/tool/builtin, internal/gaea/vision, internal/schedule
- `internal/gaea/contextview` → internal/gaea/agent/session, internal/util
- `internal/gaea/agent/session` → internal/gaea/event, internal/gaea/fileutil, internal/gaea/provider, internal/gaea/spaces
- `internal/gaea/tool` → internal/gaea/provider, internal/gaea/wspath
- `internal/gaea/plugin` → internal/gaea/provider, internal/gaea/tool, internal/netclient
- `internal/gaea/boot` → internal/gaea/agent, internal/gaea/agent/session, internal/gaea/archive, internal/gaea/cache, internal/gaea/command, internal/gaea/config, internal/gaea/context, internal/gaea/control, internal/gaea/db, internal/gaea/event, internal/gaea/genui, internal/gaea/hook, internal/gaea/jobs, internal/gaea/largefile, internal/gaea/memory, internal/gaea/outputstyle, internal/gaea/permission, internal/gaea/pins, internal/gaea/plugin, internal/gaea/provider, internal/gaea/sandbox, internal/gaea/sessionquery, internal/gaea/skill, internal/gaea/tool, internal/gaea/tool/builtin, internal/gaea/vision, internal/netclient
- `internal/gaea/hook` → internal/gaea/proc, internal/gaea/sandbox, internal/gaea/strutil
- `internal/gaea/context` → internal/gaea/cache, internal/gaea/provider
- `internal/gaea/permission` → internal/gaea/event
- `internal/gaea/event` → internal/gaea/nilutil, internal/gaea/provider
- `internal/gaea/agent/render` → internal/gaea/agent/textutils, internal/gaea/event, internal/gaea/provider
- `internal/gaea/command` → internal/gaea/frontmatter, internal/gaea/tool
- `internal/gaea/sandbox` → internal/gaea/proc
- `internal/gaea/agent/testutil` → internal/gaea/provider
- `internal/gaea/agent/cache` → internal/gaea/provider
- `internal/gaea/agent/budget` → internal/gaea/provider

### 内核·共享库

- `internal/gaea/workcost` → internal/gaea/strutil
- `internal/gaea/memory` → internal/gaea/bm25, internal/gaea/frontmatter, internal/gaea/spaces, internal/gaea/strutil, internal/gaea/tool
- `internal/gaea/config` → internal/gaea/fileutil, internal/gaea/permission, internal/gaea/provider, internal/gaea/spaces, internal/netclient
- `internal/gaea/cost` → internal/gaea/bm25, internal/gaea/strutil
- `internal/gaea/browser` → internal/gaea/proc
- `internal/gaea/skill` → internal/gaea/config, internal/gaea/frontmatter, internal/gaea/genui, internal/gaea/strutil, internal/gaea/tool
- `internal/gaea/workbench` → internal/gaea/filefilter, internal/gaea/fileutil, internal/gaea/fileutil/encoding, internal/gaea/wspath
- `internal/gaea/trajectory` → internal/gaea/agent/session, internal/gaea/evidence, internal/util
- `internal/gaea/costref` → internal/gaea/cost, internal/gaea/costinquiry, internal/gaea/costproject
- `internal/gaea/costimport` → internal/gaea/cost, internal/gaea/fileutil/encoding, internal/gaea/strutil
- `internal/gaea/knowledge` → internal/gaea/config, internal/gaea/db, internal/gaea/search, internal/gaea/strutil
- `internal/gaea/cache` → internal/gaea/provider, internal/gaea/strutil, internal/gaea/tool
- `internal/gaea/cad` → internal/docmd, internal/gaea/fileutil
- `internal/gaea/provider` → internal/gaea/nilutil
- `internal/gaea/dag` → internal/gaea/fileutil
- `internal/gaea/backup` → internal/gaea/fileutil
- `internal/gaea/costinquiry` → internal/gaea/cost
- `internal/gaea/semantic` → internal/gaea/retrieval
- `internal/gaea/pricefeed` → internal/gaea/cost, internal/gaea/fileutil/encoding
- `internal/gaea/attachment` → internal/gaea/wspath
- `internal/gaea/jobs` → internal/gaea/event, internal/gaea/nilutil
- `internal/gaea/knowledgeimport` → internal/docmd, internal/gaea/knowledge, internal/gaea/strutil, internal/gaea/textsim, internal/office/cjk
- `internal/gaea/retrieval` → internal/gaea/cost
- `internal/gaea/wssearch` → internal/docmd, internal/gaea/search, internal/gaea/strutil, internal/gaea/wspath
- `internal/gaea/largefile` → internal/docmd, internal/gaea/provider, internal/gaea/tool
- `internal/gaea/costproject` → internal/gaea/cost
- `internal/gaea/sessionquery` → internal/gaea/agent/session, internal/gaea/tool
- `internal/gaea/filewatch` → internal/gaea/wspath
- `internal/gaea/provider/bridge` → internal/ai, internal/gaea/provider
- `internal/gaea/pins` → internal/gaea/fileutil, internal/gaea/strutil
- `internal/gaea/billing` → internal/netclient
- `internal/gaea/factbase` → internal/gaea/fileutil
- `internal/gaea/fileindex` → internal/docmd, internal/gaea/semantic, internal/gaea/wspath
- `internal/gaea/outputstyle` → internal/gaea/frontmatter
- `internal/gaea/filefilter` → internal/gaea/wspath

### 办公域

- `internal/office/xlsxedit` → internal/gaea/proc
- `internal/office/docxedit` → internal/office/ooxml
- `internal/office/pptxedit` → internal/office/ooxml
- `internal/office/qa` → internal/office/ooxml, internal/office/standard
- `internal/office/ooxml` → internal/gaea/fileutil
- `internal/office` → internal/core

### 小说域

- `internal/whisper` → internal/gaea/fileutil, internal/gaea/genui, internal/gaea/proc, internal/gaea/strutil, internal/util
- `internal/booksource` → internal/bookimport, internal/netclient
- `internal/whisper/db/repos` → internal/whisper, internal/whisper/db
- `internal/novelstyle` → internal/util
- `internal/characterlib` → internal/assistant, internal/gaea/fileutil, internal/types, internal/whisper
- `internal/novelreview` → internal/noveltext
- `internal/novelcontext` → internal/gaea/strutil, internal/graph, internal/project, internal/style, internal/types, internal/util
- `internal/bookimport` → internal/gaea/strutil, internal/util
- `internal/character` → internal/ai, internal/characterstate, internal/config, internal/maturecraft, internal/project, internal/prompt, internal/types, internal/util
- `internal/outline` → internal/ai, internal/config, internal/maturecraft, internal/project, internal/prompt, internal/types, internal/util
- `internal/novelgate` → internal/noveltext, internal/types
- `internal/rewrite` → internal/gaea/textsim, internal/types
- `internal/characterstate` → internal/gaea/strutil, internal/types
- `internal/narrative` → internal/gaea/fileutil, internal/util
- `internal/scene` → internal/gaea/fileutil, internal/types
- `internal/worldview` → internal/ai, internal/config, internal/project, internal/prompt, internal/types
- `internal/chapter` → internal/ai, internal/config, internal/project, internal/prompt, internal/types, internal/util
- `internal/style` → internal/ai, internal/project, internal/util

### 引擎域

- `internal/ai` → internal/auth, internal/config, internal/gaea/config, internal/modelengine, internal/netclient, internal/util
- `internal/modelengine` → internal/gaea/fileutil, internal/gaea/strutil, internal/netclient
- `internal/tts` → internal/netclient
- `internal/herdsman` → internal/netclient
- `internal/voice` → internal/tts
- `internal/ocr` → internal/netclient

### 众神殿

- `internal/pantheon` → internal/gaea/fileutil, internal/gaea/proc

### 领域库

- `internal/schedule` → internal/gaea/fileutil
- `internal/docmd` → internal/gaea/proc
- `internal/channels/weixin` → internal/gaea/strutil, internal/netclient
- `internal/config` → internal/gaea/config, internal/gaea/fileutil
- `internal/project` → internal/gaea/fileutil, internal/scene, internal/snapshot, internal/types
- `internal/analysis` → internal/ai, internal/characterstate, internal/config, internal/gaea/strutil, internal/project, internal/prompt, internal/types, internal/util
- `internal/graph` → internal/project, internal/types
- `internal/export` → internal/gaea/fileutil, internal/project, internal/types
- `internal/auth` → internal/config, internal/gaea/secure, internal/netclient
- `internal/promptstore` → internal/prompt
- `internal/authorprefs` → internal/gaea/fileutil
- `internal/core` → internal/gaea/fileutil, internal/gaea/proc, internal/netclient
- `internal/netclient` → internal/netclient/sysproxy
- `internal/visual` → internal/project, internal/types
- `internal/stats` → internal/project, internal/types
- `internal/util` → internal/gaea/strutil
- `internal/memory` → internal/gaea/bm25, internal/project, internal/util
- `internal/snapshot` → internal/types
- `internal/assistant` → internal/gaea/secure, internal/whisper
- `internal/context` → internal/project, internal/types, internal/util
- `internal/search` → internal/project, internal/types

### scripts 工具带

- `scripts/voice_v2/ws_probe` → internal/voice/native, internal/voice/realtime
- `scripts/attention_report` → internal/gaea/agent/session
- `scripts/voice_design` → internal/characterlib, internal/config, internal/tts
- `scripts/voice_backfill` → internal/characterlib, internal/config, internal/tts
- `scripts/migrate_workcost` → internal/gaea/config, internal/gaea/db, internal/gaea/workcost
- `scripts/voice_v2/vadcheck` → internal/voice/native
- `scripts/pantheon_eval` → internal/config, internal/pantheon
- `scripts/voice_v2/diagserver` → internal/voice/realtime

## 5. 巨石文件 Top 30（非测试，按 LOC）

| 文件 | 包 | LOC |
|---|---|---:|
| `internal/pantheon/manager.go` | `internal/pantheon` | 3,416 |
| `internal/app/whisper_handler.go` | `internal/app` | 1,604 |
| `internal/gaea/agent/task.go` | `internal/gaea/agent` | 1,561 |
| `internal/voice/realtime/pipeline.go` | `internal/voice/realtime` | 1,464 |
| `internal/gaea/contextview/fold.go` | `internal/gaea/contextview` | 1,310 |
| `internal/gaea/tasks/tasks.go` | `internal/gaea/tasks` | 1,258 |
| `internal/gaea/costimport/costimport.go` | `internal/gaea/costimport` | 1,233 |
| `internal/app/gaea_ui_extra.go` | `internal/app` | 1,218 |
| `scripts/gen_bindings/main.go` | `scripts/gen_bindings` | 1,191 |
| `internal/whisper/orchestrator.go` | `internal/whisper` | 1,168 |
| `internal/app/image_handler.go` | `internal/app` | 1,165 |
| `internal/gaea/agent/agent.go` | `internal/gaea/agent` | 1,158 |
| `internal/channels/weixin/clawbot.go` | `internal/channels/weixin` | 1,139 |
| `internal/gaea/config/config.go` | `internal/gaea/config` | 1,109 |
| `internal/gaea/control/controller.go` | `internal/gaea/control` | 1,101 |
| `internal/gaea/cost/cost.go` | `internal/gaea/cost` | 1,090 |
| `internal/ai/image_comfyui.go` | `internal/ai` | 1,078 |
| `internal/app/create_chapter_handler.go` | `internal/app` | 1,078 |
| `internal/app/chat_service.go` | `internal/app` | 1,039 |
| `internal/novelcontext/novelcontext.go` | `internal/novelcontext` | 1,038 |
| `internal/pantheon/runner.go` | `internal/pantheon` | 1,023 |
| `internal/gaea/boot/boot.go` | `internal/gaea/boot` | 1,010 |
| `internal/app/create_chapter_context.go` | `internal/app` | 988 |
| `internal/app/app.go` | `internal/app` | 978 |
| `internal/gaea/agent/subagent_store.go` | `internal/gaea/agent` | 978 |
| `internal/gaea/agent/compact.go` | `internal/gaea/agent` | 972 |
| `internal/app/sin_handler.go` | `internal/app` | 964 |
| `internal/app/gaea_dag.go` | `internal/app` | 929 |
| `internal/app/characterlib_gen_handler.go` | `internal/app` | 915 |
| `internal/app/gaea_handler.go` | `internal/app` | 906 |

## 6. 前端 src 地图（ts/tsx 计数；.test.* 单列）

| 目录 | 源文件 | 源码 LOC | 测试文件 | 测试 LOC |
|---|---:|---:|---:|---:|
| `gaea` | 415 | 100,630 | 242 | 34,195 |
| `pages` | 120 | 54,597 | 59 | 14,666 |
| `components` | 160 | 37,564 | 102 | 14,283 |
| `(顶层)` | 9 | 14,169 | 8 | 641 |
| `schedule` | 68 | 13,239 | 48 | 8,631 |
| `voice` | 10 | 5,034 | 5 | 1,134 |
| `genui` | 14 | 3,738 | 5 | 573 |
| `api` | 6 | 1,969 | 2 | 174 |
| `data` | 15 | 1,782 | 1 | 35 |
| `hooks` | 9 | 1,620 | 2 | 204 |
| `utils` | 23 | 876 | 16 | 662 |
| `layouts` | 1 | 751 | 1 | 82 |
| `stores` | 2 | 607 | 3 | 343 |
| `v3` | 1 | 553 | — | — |
| `types` | 2 | 512 | — | — |
| `boards` | 4 | 502 | 2 | 418 |
| `test` | 2 | 238 | — | — |
| `lib` | 4 | 186 | 2 | 120 |

boards/ 细目：

- `manifests.ts`：1 文件 / 350 行
- `types.ts`：1 文件 / 61 行
- `space.ts`：1 文件 / 58 行
- `pageRegistry.ts`：1 文件 / 33 行

## 7. 关键接缝导读（手写段，动结构时同步更新）

1. **工具面（唯一注册表）**：`boot.Build` 装配单一 `tool.Registry`，四个入口——
   builtin（`RegisterBuiltin` 全集）、ExtraTools（app 层 `gaeaExtraTools` 单点）、
   插件/MCP 热插（`plugins.go`/`controller_mcp.go`，动态 `mcp__<server>__<tool>`）、
   Build 内部直加（skill/memory/task/slash）。契约=`tool.RegistryMap`（P0 五行契约，
   tool/boot/app 三面守卫+文档单源导出）；空间装配期物理过滤=`builtin.AllowsSpace`
   （S1.3-B；2026-11-10 漂移刀后与契约行双守卫钉死）。`/mcp__` 前缀是 prompt 旁路不进注册表。
2. **执行面**：`internal/gaea/agent`（会话/预算/渲染/工具守卫/子代理）、`control`
   （控制器输入流与 MCP prompt 旁路）、`permission`（Approver 审批面，子代理审批放行点）、
   `evidence`（交付证据五档：agent-write/agent-produce/office-journal/publish-guard/none）。
3. **壳层（板块装配）**：`internal/app` 是最大层（600+ 文件），wails bindings 双向
   （`scripts/gen_bindings` 生成，drift 锁）；板块=board manifest + 各 `gaea_*.go` 文件；
   编程板块走隔离 DSH（`C:\AI\dsh`:3091，preflight 查红队预设 `id: rea-mcp`）。
4. **引擎域**：herdsman（OpenAI 兼容 API，本机 8080）、modelengine、tts、voice
   （native=cgo sherpa-onnx，三 DLL 与 exe 同目录铁律，缺 ucrt64 gcc 构建会静默降级空壳）、
   ocr、ComfyUI（图片后端）。外部引擎均有 idle-exit/JobObject 收割线。
5. **数据面**：配置 `.gaea_config.json`（旧品牌 `.wubigork_config.json` 回退）；
   `internal/gaea/db`=modernc.org/sqlite（纯 Go，WAL）；memory/knowledge/factbase/cost
   各自持久化；会话存档 jsonl+gaea-log 双形态；众神殿 sessions 在 `$APPDATA/gaea/pantheon`。
6. **前端**：`frontend/src`（React+TS），boards=板块页、stores=状态、api=wails 绑定包装；
   HTTP 桥接 opt-in（`GAEA_HTTP_PORT`，冒烟/走查用）。

## 8. 再生成方式

```bash
go run scripts/code_atlas_doc.go > docs/gaea-code-atlas-2026-11.md
```

机械段（§1–§6）随代码变化重导即可；接缝导读（§7）与分层口径（脚本顶部
kernelRT/novelTops/engineTops 集合）是人工认知，调整后重导。新增顶层包通常只须
确认它落进的层名是否符合预期（domain 是兜底层）。
