# 书源引擎 t1 规格符合性核查 + 补测 + 对拍（2026-09-13）

> 对象：`internal/booksource/**`（原罪·书源引擎 t1，非版本刀）
> 规格：`docs/gaea-sin-booksource-distill-2026-09.md`（§2 机制 / §3.2 schema / §4 偏离 / §7 验收）
> 方式：只读通读全部 11 个文件 + 新增 §7 缺口补测（**先红后绿**）+ 负向对照 + 定向门禁
> 纪律：未执行 `git checkout/stash/restore`；未改动实现者既有夹具（补测独立成文件）

## 0. 结论

| 项 | 结果 |
|---|---|
| 规格主干符合性 | **符合**：三级抓取纪律、声明式抽取、搜索/聚合/相似度、目录分页、正文分页与末页启发式、清洗链六步、两路排版归一、TXT 组装与 GBK 回环、fail-closed 校验器齐备 |
| 缺陷 | **6 项实修**（1 项正确性 / 1 项静默失真 / 1 项静默忽略 / 1 项 fail-closed 误拒 / 1 项死字段 / 1 项隐性缺陷），全部带回归锁与负向对照 |
| 测试 | 23 → **41 例**全绿；覆盖率 **81.9% → 88.0%** |
| 门禁 | `go build ./...` exit 0、`go vet ./internal/booksource/` exit 0、`gofmt -l` 空 |
| 观察项 | 4 项留给 t2 接线拍板（含 tail 模式章号语义、CF 非 200 分类），不阻塞 t1 |

## 1. 核查方法（命令即证据）

| 步骤 | 命令 | 结果 |
|---|---|---|
| 基线 | `go test -count=1 -cover ./internal/booksource/` | ok，coverage 81.9%，23 例 |
| 缺口补测（红） | `go test -count=1 ./internal/booksource/` | **8 处红**（下节逐条） |
| 修复后（绿） | 同上 | ok，41 例全绿 |
| 覆盖率 | `go test -coverprofile` + `go tool cover -func` | 88.0% |
| 编译/静态 | `go build ./...` / `go vet ./internal/booksource/` / `gofmt -l` | exit 0 / exit 0 / 空 |
| 负向对照 | 临时退回「分页上限静默截断」→ 复跑目标用例 | 精确 FAIL（测试有效，非空跑） |
| 全仓回归 | `go test -count=1 ./internal/...`（后台跑完） | **118 包全 ok，0 FAIL，EXIT=0** |

## 2. 红 → 绿清单（6 项修复）

### F-1【P2 正确性】闭合路线只认 `#text`，`<div>` 逐段源被压成一整段

- 现象：`paragraphsFromHTML("正文前<div>块中</div>正文后", true, "")` → `["正文前块中正文后"]`（三段黏连）。
- 根因：`children` 路线用 `Children()`（**只含元素节点**）+ `NodeName(s)=="#text"` 判定，该分支事实不可达；`Contents()` 才含文本节点（goquery v1.13.0 `traversal.go:65` `siblingAllIncludingNonElements`）。
- 与规格冲突：§2.9 / §4 D3「闭合标签路线=非 `<p>` 闭合标签（直接子级）统一改名 `<p>`」——改名等价物应「每个直接子级各成一段」。
- 处置：改 `root.Contents()` 按文档序成段，跳过 `#comment/script/style`，文本节点内部换行仍按行切（`clean.go`）。
- 回归锁：`TestParagraphsClosedRouteNonPBlocks`（div 双段 / 文夹块三段 / 标准 `<p>` 路线不回退）。

### F-2【P2 静默失真】单章分页触顶静默截断正文

- 现象：环形/超长分页时 `page > 50` 直接 `break`，返回**部分正文且无错误**（`TestChapterPaginationCapIsExplicitError` 红）。
- 与规格冲突：§2.7 的诚实报错原则（正文抽空都要显式报错，静默截断更甚）。
- 处置：`maxChapterPages=50` 触顶返回显式错误「本章分页超过 50 页上限（疑似环形分页链接）」；`Download` 侧按既有语义记入 `Failed`（不中断整本）。
- 负向对照：临时改回 `break` → 目标用例精确 FAIL，恢复后绿。

### F-3【P2 静默忽略】`chapter.nextPageInJs` 被 JSON 未知键静默丢弃

- 现象：`Parse` 对带 `nextPageInJs` 的规则**返回成功**（`TestParseRejectsNextPageInJs` 红）→ 表现为「分页抓不到 → 章节被静默截断」。
- 与规格冲突：§4 D4 明确要求「`@js:` 步骤、XPath、`nextPageInJs` 不做，校验器/解析器**如实报不支持**」。
- 处置：`Parse` 前置 `rejectUnsupportedFields`（schema 外显式点名，`rule.go`），报错形如 `chapter.nextPageInJs 不支持：需 JS 步骤引擎…`。

### F-4【P2 fail-closed 误拒】URL 模板被当正则校验

- 现象：`search.url = https://x.example/search?tags[]=1&q=%s` → 拒收，报 `search.url 正则不可用: missing closing ]`（`TestValidateAcceptsLegitURLTemplates` 红）。
- 根因：`search.url` / `toc.url` 是**带 `%s` 槽位的 URL 模板**（运行期走 `strings.ReplaceAll`），却被 `checkRegex` 当正则编译；合法 URL 里的 `[]`、`(` 会误判。
- 处置：移除两处 `checkRegex`；真正则字段（`book.url`/`filterTxt`/`paragraphTag`/`endPattern`）校验保留并注释划界。

### F-5【P3 死字段】`baseUri` 声明后无任何消费方

- 现象：注入的 Fetcher 不回报页址（`Page.URL=""`）时，相对链接原样返回 `/book/9/`（`TestBaseURIUsedWhenPageURLAbsent` 红）。
- 与规格冲突：§3.2 明确 `baseUri` = 「相对链接补全基准」。
- 处置：新增 `(*Engine).base(pageURL)`（页址优先、缺失回落 `baseUri`），接入搜索链接/详情/目录三处；`baseUri` 为空时行为与旧实现完全一致（向后兼容）。

### F-6【P3 隐性缺陷】目录分页去重的索引在删除后未失效

- 现象：`tocPageURLs` 的「去重保序、后写覆盖」实现按旧索引 `i` 删元素，删除后 `seen` 里其余 URL 的下标全部失真——只因上游 `absLinks` 自身去重（至多触发一次删除）才没爆；一旦 `absLinks` 语义变化（或下拉里出现多重复）就会**删错页 / 丢章 / 重复章**。
- 附带：被挪到队尾的 URL 已抓到的文档被丢弃，同一页**重复抓取**（`TestTocPaginationDedupesFirstPage` 断言 2 次 → 1 次）。
- 处置：删除后重建索引；已抓文档随行复用（`toc.go`）。回归锁含「首页在下拉里重复出现 → 不丢章不重复 + 只抓一次 + 后写覆盖序」。

## 3. 测试自纠 2 处（**非实现缺陷**，记录以免误判）

- 并发取消断言：初版假设「取消后等待槽位的槽位数恒定」，实测因 `runBounded` 的槽位释放竞争，非确定（12/20）；改为**先占满信号量再取消**的确定性写法，断言 `canceled=18 / calls=2`。实现行为本身正确（不外泄第三种错误、不超限起跑）。
- 并列 tie-break：初版用「甲/乙」假设码点序，实际 Go 字符串比较是**字节序**（乙 < 甲）；改用 ASCII 名并补「书名搜 → 按作者字典序」反向用例。

## 4. §7 验收逐条对拍

| §7 验收项 | 覆盖测试（既有 + 补测） |
|---|---|
| 规则：模板可解析 / 坏规则逐类报错 | `TestTemplateParsesAndValidates`、`TestValidateFailClosed`(10 类)、**`TestValidateAcceptsLegitURLTemplates`**、**`TestParseRejectsNextPageInJs`** |
| 抽取：text/html/`@href`/absURL 补全 | **`TestExtractHelpersAndAbsURLResolution`**（含协议相对、缺省 href、无属性、去重保序）、**`TestBaseURIUsedWhenPageURLAbsent`** |
| 搜索：单源解析 / limit / POST 槽位 / 分页集合 || 聚合降级 | `TestSearchParseLimitAndPagination`、`TestFetchDisciplineHeaders`（POST+cookie+UA+Referer）、`TestAggregateDegradedAndSorted`、**`TestSearchGuardsAndEmptyNameRows`** |
| 相似度：判别 / 0.25 / 回退 >0 / tie-break | `TestSimilarityBasics`、`TestFilterSortDetectsAuthorQuery`、**`TestSimilarityBoostBranchesAndTieBreak`**（12/6/3 加权、双向 tie-break、0.25 严格下界） |
| 目录：option 集合 / 页序归并 / reverse / 范围 | `TestTocPaginationMergedInOrderAndRange`、`TestTocReverseAndEmpty`、**`TestTocListContainerTwoHopParsing`**、**`TestTocPaginationDedupesFirstPage`** |
| 正文：单页/分页 / endPattern / 启发式 / 空正文报错 | `TestChapterPaginatedEndHeuristics`、`TestChapterEndPatternRule`、`TestChapterEmptyContentIsExplicitError`、**`TestChapterPaginationCapIsExplicitError`**、**`TestDownloadCancelRecordsFailuresWithoutPanic`** |
| 清洗：PUA/实体/广告/整删/标题剥离/归一/空标签/两路归一 | `TestCleanChainUnits`、`TestChapterCleanChainAndTitleNormalize`、`TestChapterNonClosedParagraphRoute`、**`TestParagraphsClosedRouteNonPBlocks`**、**`TestParagraphsNonClosedBrRoute`** |
| 抓取纪律：抖动 / 线性退避 / 并发钳制 / 零重试 / CF | `TestFetchDisciplineRetryBackoffLinear`、`TestCancelledCtxZeroCalls`、`TestCloudflareDetectedNotBypassed`、**`TestCrawlConfigDefaultsOverrideAndClamp`**、**`TestJitterUsesCrawlOverrideAndDefaults`**、**`TestRunBoundedEnforcesConcurrencyLimit`**、**`TestHTTPFetcherPostCookieAndNon200`** |
| 组装：信息头 / 章序 / GBK 回环 | `TestAssembleTXTAndGBKRoundtrip`、**`TestAssembleTXTIntroAndOrderFallbacks`** |
| 详情页（§3.3 assemble.go 表） | **`TestBookDetailMetaFallbacksAndCoverSrc`**（选择器优先 → og meta 回落 → `@src` 补全） |

## 5. 未改动观察项（交 t2 接线时拍板）

1. **范围下载的章号语义（与 t2「tail 模式出口」直接相关）**：`Toc(start,end)` 切片后 `Order` 从 1 重编号（`toc.go` 现有测试 `toc2[0].Order == 1` 锁定了该口径）。后果：下载末 N 章时 `AssembleTXT` 会输出「第1章…」，与真实章号错位，下游按「第N章」分段的链路会跟着错。若要让成书章号忠实，改点只有一行（`toc[i].Order = start + i`），但属语义变更，**需 t2 拍板**后再改。
2. **CF 挑战页的状态码分类**：`HTTPFetcher` 对非 200 直接报错，故 403/503 上承载的 CF 挑战页只会得到 `HTTP 403`，拿不到 `ErrCloudflare`（诚实报错，但分类不同）。要精确分类需让 Fetcher 在非 2xx 时也回传 body（**接口契约变更**），留给 t2。
3. **`Disabled` 的适用范围**：目前只作用于聚合与单源搜索（`Aggregate` 跳过、`Search` 返回 `ErrSearchUnsupported`）；直接 `Toc/Chapter/Download` 仍可跑。规格 §2.1 只说「禁用位」，未定义是否含下载面——t2 绑定时应明确（建议：下载入口也尊重 `Disabled`，否则「禁用源」在 UI 上语义不完整）。
4. **POST 路径的 URL 槽位未转义**：GET 走 `url.QueryEscape`（优于上游直填），POST 的 URL 槽位仍填原文（表单值由编码器转义）。站点若在 POST URL 上也带关键字且含空格/`&`，行为未验证——t2 真机跑源时如遇异常优先看这里。

## 6. 改动文件（本刀）

| 文件 | 改动 |
|---|---|
| `internal/booksource/rule.go` | F-3 不支持下字段点名、F-4 去 URL 正则误检、`xpathPrefix` 去重复分支 |
| `internal/booksource/html.go` | F-5 `(*Engine).base` |
| `internal/booksource/search.go` / `toc.go` / `assemble.go` | F-5 接入基准；F-6 去重索引重建 + 文档复用 |
| `internal/booksource/chapter.go` | F-2 分页上限显式报错；`Failed[].Err` 空因兜底（不再依赖 `ctx.Err()` 非空） |
| `internal/booksource/clean.go` | F-1 闭合路线 `Contents()` 逐子级成段 |
| `internal/booksource/booksource_acceptance_test.go` | 新增 18 例（§7 缺口 + 6 项回归锁） |
