# 小说板块 · 书源取书→拆书导入接通（刀路规格）

> 状态：**t1 已落地（2026-09-13，非版本刀）**；t2 前端「在线搜书」/t3 打磨未开工 · 类型：板块接线规格
> 关联代码：`internal/booksource/`（引擎，已落地零消费者）、`internal/app/novel_import_handler.go`、
> `internal/bookimport/`、`internal/app/bindings_novel.go`、`frontend/src/pages/HomePage.tsx`
> 关联决策：`docs/gaea-sin-booksource-distill-2026-09.md`（书源引擎蒸馏规格，§7 t5 预留「拆书联动」）、
> `docs/distill/02-book-import.md`（拆书导入）、`docs/gaea-novel-ohstory-distill-2026-09.md`（T4 导入结构映射与篇幅路由）、
> `docs/gaea-sin-board-design-2026-09.md` §8（原罪硬隔离——本线不触碰 sin 面）

## 1. 需求与口径

用户拍板（2026-09-13）：小说板块与原罪板块不合并（两条在案拍板：原罪单独板块 + 硬隔离），
改做「接通」——**书源引擎（搜书→选书→下载正文）作为小说板块拆书导入的取书上游**，
在书架导入向导里补「在线搜书」半边：搜书 → 目录预览 → 范围下载 → 直接入书架成项目。

拆成可验收口径：

1. **入口在小说侧**：书架导入区新增「在线搜书」，不新增板块、不动原罪页面；
2. **复用既有导入端**：下载的章节走与文件导入同一落库链（项目目录 + outline + chapters/），
   导入报告口径（编码/切分/告警直显）沿用 v4.279；
3. **内存直通**：引擎抓到的章节直接进项目文件，不出中间 TXT/EPUB 文件（对齐书源规格 D7）；
4. **规则资产共享**：书源规则是引擎数据资产（同角色库=跨板块共享资产层先例），
   落 `%配置目录%/gaea/booksource/{rules,websearch-engines.json}`，原罪线未来同源取用，不各养一份。

## 2. 接缝实测（两端 API，file:line）

**取书端（internal/booksource，引擎 34 例已绿，全仓零消费者）：**

| 能力 | API | 落点 |
|---|---|---|
| 规则装载 | `LoadDir(dir) []Loaded` / `Parse(raw)` / `EnsureTemplate(dir)`（双模板幂等） | rule.go:361-421、template.go:20 |
| 书源聚合搜索 | `Aggregate(ctx, opt, keyword, rules...) → ([]SearchResult, []SourceError)` | search.go:277 |
| 泛搜索（免书源） | `WebSearcher.DiscoverySearch(ctx, eng, keyword, opt) → []WebCandidate`（带 HasRule 标记） | websearch.go:163 |
| 详情/目录 | `Engine.BookDetail(ctx, detailURL) → BookInfo`；`Engine.Toc(ctx, detailURL, start, end) → []TocEntry` | assemble.go:29、toc.go:113 |
| 整本下载 | `Engine.Download(ctx, detailURL, DownloadOptions{Start,End,OnProgress}) → DownloadReport{Chapters []ChapterText, Failed []FailedChapter}`（有界并发、失败不占位、进度回调） | chapter.go:143-208 |
| 成书组装 | `AssembleTXT(info, chapters, encoding) → []byte`（本线**不用**，内存直通） | assemble.go:87 |

**导入端（novel_import_handler.go）：**
`ImportNovelBook(filePath, title, genre, style)` = 文件校验 → `parseNovelFile` →
**后半：`project.Create` → 逐章 `pm.WriteChapter` → `pm.WriteOutlines` → `NovelImportResult`**（:67-102）。
后半与「章节从哪来」无关——`bookimport.Parse` 吃 `[]byte`、`ChapterText{Title,Content}` 与
`importChapter{Title,Content}` 同构，接缝天然干净。

**前端入口**：书架 `HomePage.tsx:100-140`（选文件→导入 Modal→报告直显），在线搜书与其并列。

## 3. 架构决策

1. **重构不改行为**：把 `ImportNovelBook` 后半抽成
   `createImportedProject(novelsDir, title, genre, style string, chapters []importChapter) (NovelImportResult, error)`；
   文件导入路径行为零变化（既有测试原样过），在线导入与其共用同一落库链。
2. **绑定面 NovelB +4**（650→654，以在制 HEAD 实测为准，drift 门禁同步）：
   - `NovelBookSourceSearch(keyword)` → 聚合候选（书源 Aggregate + 泛搜索 DiscoverySearch 同表，带来源/HasRule/相似度序）；
   - `NovelBookSourceToc(sourceRef)` → 目录预览（章数 + 首末章标题样例，供范围选择）；
   - `NovelBookSourceImport(sourceRef, start, end, title, genre, style) → jobID`（后台起跑，进度走事件）；
   - `NovelBookSourceImportCancel(jobID)`（ctx 取消，SinCancel 同款先例）。
3. **进度事件**：`novel-import-progress:<jobID>`（done/total/failed 计数，终态 done/error 带 NovelImportResult 或失败清单），`a.emit` 惯例同 `sin-stream`/`create-chapter-stream`。
4. **失败语义沿引擎**：单章失败不中断整本，`Failed` 清单随终态事件带出、前端直显（t3 再做失败章重试）。
5. **范围导入章号**：范围下载 `Start/End` 落项目时**从 1 重编号**（书源规格 §7 观察项已点名错位问题，本线落地即修——导入端章号 = 项目章号，连续自洽）。
6. **超时与体量护栏**：单次导入章数上限沿用引擎目录页 20 页上限与有界并发；job 级 ctx 超时给宽上限（如 30 分钟）防挂死；同项目目录 `uniqueProjectDir` 防覆盖（既有件）。

## 4. 边界与撞车声明（双会话合流纪律）

| 线 | 归属 | 本线动作 |
|---|---|---|
| sin 书源线 t2~t5（SinBookSource* 绑定/面板/工具/拆书联动） | 并行线（DSH）拍板池 | **不碰**：不动 bindings_sin.go、不动 OriginalSinPage/sin/*；其 t5「成书 TXT 进导入向导」未来直接消费本线 t1 的 `createImportedProject` |
| 规则目录 | 本规格定义共享层 `%配置目录%/gaea/booksource/` | 若并行线 t2 先落 `sinRoot()/rules`，合流时二选一收敛到共享层（以先落地方案为准去重） |
| oh-story T4（导入结构映射与篇幅路由） | oh-story 刀路（bookimport 续刀） | 不塞进本线：本线章节原样进项目（与文件导入同口径）；T4 落地后在线导入自动受益同一 receiving end |
| 原罪硬隔离（sin 设计 §8） | 在案拍板 | 不破：规则目录是引擎资产层（角色库先例），非 sin 数据面；本线零 sin 引用 |

## 5. 分刀计划（每刀独立提交、可回退）

| 刀 | 内容 | 验收判据 |
|---|---|---|
| **t1**（后端，非版本刀可先行） | `createImportedProject` 重构 + 共享资产目录（EnsureTemplate 幂等）+ NovelB +4 绑定 + 进度/取消事件 | ✅ 2026-09-13：go build/vet/test 全绿（internal/app 全量 84s 过）；绑定面 654 drift OK；文件导入回归零变化（TestImportNovelBook_RefactorRegression）；13 新例（规则装载剔除/搜索合并去重/目录截断样例/导入落库/Failed 清单/范围重编号/全败报错/无规则拒绝/取消登记簿/起跑预检）；实施偏差见 §8 |
| **t2**（前端，版本刀） | 书架「在线搜书」入口：搜索→候选表（来源/HasRule 标注）→目录预览→范围选择→进度条→完成入书架 + 报告直显（复用 v4.279 报告面） | tsc -b 0、eslint 0 error、vitest 全绿（面板新例）、真机走查一轮 |
| **t3**（打磨，按反馈） | 失败章重试、搜索历史、泛搜索引擎规则用户可编辑 | 观察池驱动，不预承诺 |

## 6. 不做清单（防过度设计）

- 不做断点续传/章节缓存目录（书源规格 D7 同判：内存够用，t3 若有需求再评估）；
- 不接 cf-bypass 外挂、不做反爬对抗（书源规格 D6 红线）；
- 不做 EPUB 在线导出（sin 线 t5 的成书侧，非导入侧）；
- 不动原罪任何文件；不把 T4 结构映射塞进导入链；
- 规则策展数据不出厂（D1：模板即示例，规则由用户/后续策展供给）。

## 7. 合规口径

沿书源蒸馏规格 §1 定位：**个人本地使用**的素材获取层（拆书参考/导入学习），
规则 fail-closed（CSS only、禁 @js:）已在引擎层锁死；本线不新增任何网络对抗能力。

## 8. t1 实施记录（与 §2/§3 预估的三处偏差，均以实测定型）

1. **`ChapterText` 是 `Paragraphs []string` 不是 Content 单串**（chapter.go:30）——
   落库时 `strings.Join(Paragraphs, "\n")` 成正文；导入报告 `SplitStrategy="booksource"`
   （Encoding 留空：在线逐页解码无单一编码），前端 t2 的 strategyLabel 需补映射。
2. **免规则下载边界比规格预估更严**：引擎 `Chapter` 无 ChapterRule 直接
   `ErrChapterUnsupported`（正文必填，同上游边界）——故 **Toc 与 Import 绑定均
   fail-closed 拒绝无规则来源**（不只 Import）；泛搜索候选只作参考，`HasRule`
   是唯一可导入通道。
3. **HasRule 打标宿主口径** = 引擎 `hostOf`（小写 Host 含端口）；规则侧宿主取
   `BaseURI`（缺省 Search.URL）同口径归一，泛搜索 `KnownRules` 回调按此匹配。

进度事件节流：每 20 章 + 完成必报（`booksourceProgressStep`）；job 语义沿
SinCancel 先例（句柄登记簿 + 精确取消 + 完成移除）。
