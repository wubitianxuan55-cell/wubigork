# 原罪板块 · 书源引擎蒸馏（so-novel）——规格与刀路

> 状态：t1 引擎 P0 已落地（非版本刀）· 类型：外部项目蒸馏规格（机制清单 + gaea 落点 + 刀路）
> 上游：https://github.com/freeok/so-novel（Java/Spring Boot CLI/Web 网文下载器，本地克隆 `clones/so-novel`，**AGPL-3.0**）
> 关联板块：原罪（`docs/gaea-sin-board-design-2026-09.md`，闲庭 `sin`）
> 关联代码：`internal/booksource/`（t1 本刀）；后续刀见 §5
> 关联决策：`docs/distill/02-book-import.md`（拆书导入，书源是其「取书」上游）

## 1. 用户口径与定位

用户原话：**「so-novel，把这个蒸馏给 gaea 的原罪板块」**。

拆解：so-novel 的本体价值不在「又一个下载器」，而在**把「网站结构」从代码里搬进数据文件**——换站点不改代码，改一份 JSON 规则即可。把这个机制搬给原罪板块，就是给原罪（以及整条小说线）补上**素材获取层**：搜书 → 选书 → 下载正文 → 清洗归一 → TXT，往下游喂拆书导入（`internal/bookimport`，v4.279/280）或作为故事参考素材。

法务红线（先于一切设计）：so-novel 是 **AGPL-3.0**，gaea 是私有版权（All Rights Reserved）——**代码、规则 JSON 一行都不搬**，只取机制（思路/字段语义/启发式常数自行重写并在码内注明「重推导」）。这与屋里「蒸馏=取道不取器」惯例重合，本刀从惯例升级为硬约束。**so-novel 自带的 18 个书源规则文件不随 gaea 分发**（见 §7 拍板池）。

## 2. so-novel 机制清单（file:line 对齐，供逐条验收）

以下 `S/` = `clones/so-novel/src/main/java/com/pcdd/sonovel/`。

### 2.1 书源规则数据化（核心机制）

规则 = 每站点一份 JSON（`bundle/rules/*.json` + `bundle/rules/rule-template.json5` 模板），五段式：

| 段 | 关键字段（so-novel 命名） | 语义 |
|---|---|---|
| 顶层 | `url/name/comment/language/disabled` | 站点根、名称、备注、语言、禁用位 |
| `search` | `url/method/data/cookies/result/bookName/author/category/latestChapter/lastUpdateTime/status/wordCount/nextPage/disabled` | 搜索 URL（`%s` 填关键字或 `@js:` 计算）、POST 表单（值为 `%s` 的槽位填关键字）、结果行选择器、字段选择器、分页 |
| `book` | `url/bookName/author/intro/category/coverUrl/latestChapter/latestChapterUrl/lastUpdateTime/status` | 详情页；`url` 是**从详情 URL 正则捕获书 id**（`S/parser/TocParser.java:47` `ReUtil.getGroup1`），供 `toc.url` 的 `%s` 拼目录页 |
| `toc` | `baseUri/url/list/item/isDesc/nextPage` | 目录页；`item` 为章节链接选择器；`list` 先取容器 HTML 再二次解析；`isDesc` 倒序源反转 |
| `chapter` | `title/content/paragraphTagClosed/paragraphTag/filterTxt/filterTag/nextPage/nextPageInJs/nextChapterLink` | 正文页；`filterTxt` 广告正则（`\|` 连缀）、`filterTag` 整元素移除、分页三件套 |
| `crawl` | `concurrency/minInterval/maxInterval/maxAttempts/retryMinInterval/retryMaxInterval`（秒） | **限流书源自带抓取纪律**，构造时覆盖全局（`S/core/Source.java:44-54`） |

规则文件按**能力/风控分档**：`main`（默认 11 源）/ `proxy-required`（需代理 4 源）/ `rate-limit`（限流 4 源，带 crawl 段）/ `no-search`（只凭详情 URL 下载）/ `cloudflare`（需 CF 绕过）。`active-rules` 配置项激活哪一份（`bundle/config.ini [source]`）。

### 2.2 声明式抽取

`S/core/HtmlExtractor.java`：选择器前缀 `(/|//|(` 判 XPath，否则 CSS（jsoup）；查询串含 `@href/@src` 抽属性、`meta[` 开头抽 content、否则 text；`@js:` 后缀交 DSL 引擎做响应体/取值二跳（`S/dsl/`，JS 执行器 GraalJS——**gaea 不取**，见 §6 偏离 D4）。

### 2.3 抓取纪律

`S/utils/CrawlUtils.java` + `S/core/Crawler.java`：

- 每请求随机抖动间隔：默认 200~400ms（`config.ini [crawl]`），书源 crawl 段覆盖；
- 重试：失败后间隔 = `random(retryMin,retryMax) × attempt` **线性放大**，默认 3 次（`S/parser/ChapterParser.java:66-99`）；
- 并发：IO 密集不绑 CPU 数，默认 50、全局钳制 100（`Crawler.java:116-118`），虚拟线程 per 任务 + 限流器；
- 每请求独立 timeout（秒，规则可覆盖）；UA 随机池 + `Referer=host`（`CrawlUtils.request`）。

### 2.4 搜索面

`S/parser/SearchParser.java`：

- URL `@js:` 时 JS(keyword)→完整 URL，否则 `%s` 格式化（:262-266）；POST body 按 `{k: %s}` 槽位填关键字（`CrawlUtils.buildData`）；
- `nextPage` 有值：一次性抓全部分页 URL（锚点集合去重并行抓，:104-113），否则只取首页；
- 每源取前 `search-limit`（默认 30）条，空书名行跳过；
- **完全匹配直接跳详情页**的站点（结果空 && 详情书名非空）→ 用详情页构造单条结果返回（:130-150）；
- 聚合搜索：全部可搜索书源并发（virtual thread per task），**单源异常静默降级空列表**不拖垮整体（`S/actions/AggregatedSearchAction.java:47-71`）。

### 2.5 相似度过滤排序

`S/handler/SearchResultsHandler.java`：对聚合结果按书名、作者各算相似度（hutool `StrUtil.similar` = 1 − Levenshtein/maxLen）；**自动判别关键字是书名还是作者**（两边相似度总和各自加权，短查询 ≤4 字对完全命中加权 12、长查询 ≥10 字降权），按胜方相似度降序；`> 0.25` 过滤低相似，过滤后为空回退 `> 0`；并列按**另一字段**字典序。

### 2.6 目录面

`S/parser/TocParser.java`：

- 详情/目录分页时 `book.url` 正则捕获 id → `toc.url` 的 `%s`；
- 目录分页：`nextPage` 选择器若命中带 value/href 的**下拉菜单（option 集合）一次性取全**（去重保序、后写覆盖，:73-88）；按钮式逐页递归兜底（:90-104）；
- 目录页并发抓取（5 并发）后**按页序归并**保证章序稳定（:132-149）；
- `isDesc` 源反转；`parse(url, start, end)` 范围下载（**尾部 N 章**的底层能力）。

### 2.7 正文面

`S/parser/ChapterParser.java`：

- 单页 / 分页两路；分页循环：正文 selector 抽 HTML → `nextPage` 按钮取 `absUrl(href)` 或 `nextPageInJs`（JS 里的链接，gaea v1 不取）；
- 末页判定（`isLastPage:186-195`）：规则 `nextChapterLink` 正则命中即末页；或通用启发式——URL 不以 `[-_]\d.html` 结尾 **且** 按钮文本含「下一章|没有了|>>|书末页」；
- **正文抽空 = 限流/风控信号**，如实报错不静默（`Assert.notEmpty`）；
- 简繁转换放**最后**（`ChineseConverter`，HanLP t2s/s2tw——gaea v1 不做，见 §6 D5）。

### 2.8 清洗链（顺序敏感）

`S/core/ChapterFilter.java`（Builder 四步 + 固定两步）：

1. 不可见字符清理：`\p{C}\p{Cf}\p{Co}\p{Zl}\p{Zp}\u200B\uFEFF`——**私有区 PUA 字符是中文乱码根源**（`CrawlUtils.cleanInvisibleChars:62`）；
2. HTML 实体引用 `&..;` 全删（ibooks 兼容）；
3. `filterTxt` 广告正则删除 + `filterTag` 元素**连内容整删**（jsoup `.remove()`，`HtmlUtils.removeTags:39-48`）；
4. 正文开头的重复标题剥离：`Pattern.quote` 先转义标题再拼正则（防章节名里的 `[(` 破坏正则），去「`^(空白|标签)*(标题|去空格标题)`」；顺带 `1.章节名` → `第1章 章节名`（阅读器目录解析兼容，`TITLE_NUMBER_PATTERN`）；
5. 空标签清理（`<p></p>` 类）。

### 2.9 排版归一

`S/core/ChapterFormatter.java`：全属性清除后两路——闭合标签源：非 `<p>` 闭合标签统一改名 `<p>`；非闭合源（`<br>+` 分隔）：按 `paragraphTag` 正则切段逐段包 `<p>`。

### 2.10 导出面

章节先落缓存目录（文件名 `序号补零_标题`，补零保证字典序=章序，`Crawler.java:158-172`）→ 后处理合并：TXT（首页书籍信息头；可选 GBK 编码兼容旧设备）/ EPUB（go-epub 同款模板路线）/ PDF / 带目录 HTML。`preserve-chapter-cache` 可留缓存增量。

### 2.11 杂项

CF 检测=页面 title 精确匹配集合（`Just a moment...` 等 4 个，`CrawlUtils:36-40`），绕过走**外挂服务** `cf-bypass` URL（gaea 不接，见 §6 D6）；下载进度事件每 50 章/完成时推 SSE（gaea 对应物=事件流，t2 接线）。

## 3. gaea 落点设计

### 3.1 定位与数据根（原罪板块，硬隔离合规）

- 书源 = 原罪的**素材获取层**：搜书 → 下载 → TXT 落 `<用户配置目录>/gaea/sin/books/<书名> (<作者>).txt`；规则放 `<用户配置目录>/gaea/sin/rules/*.json`（用户自备/自编，**不出厂规则集**，§7）；
- 与办公硬隔离零破坏：数据全在 `sinRoot()`（`internal/app/sin_store.go:39-41`，同 `art/`、`cast.json` 惯例）；不进办公检索面（`.gaea/play` 已整分区判噪）；
- 引擎包 `internal/booksource`：**纯规则零 AI、Fetcher/Sleeper/Rand 三依赖注入**（HTTP 客户端、时钟抖动、随机源全部可桩），与 `internal/bookimport` 同风格——网络与文件 IO 只在 app 层（t2）出现；
- 复用：网页解码直接用 `bookimport.Decode`（utf-8→utf-8-sig→gb18030→gbk→big5，含替换符误判续探）——GBK 站点正文页是这条链的第二消费方；导出编码用 `golang.org/x/text` 反向编码器。

### 3.2 规则 schema（gaea 版，字段语义与 so-novel 对齐）

对齐到**手工移植一份 so-novel 规则 = 机械翻译**的程度，但命名/形状按 gaea 口径重定：

```jsonc
{
  "name": "示例书源",              // 必填
  "baseUri": "https://example.com/", // 相对链接补全基准
  "search": {
    "url": "…/search?q=%s",        // 必填；%s=URL 转义后的关键字（@js: 不支持，校验器报）
    "method": "get",               // get（默认）| post
    "data": {"searchkey": "%s"},   // post 表单；值为 "%s" 的槽位填关键字
    "result": ".result-item",      // 必填；结果行选择器（CSS）
    "bookName": ".title",          // 必填
    "link": ".title@href",         // 详情链接；缺省=对 bookName 元素取 href
    "author": "…", "category": "…", "latestChapter": "…", "lastUpdateTime": "…",
    "status": "…", "wordCount": "…",
    "nextPage": ".pager a",        // 锚点集合一次性取全（so-novel 同语义）
    "limit": 30                    // 每源上限，缺省 30
  },
  "book": {
    "url": "…/book/(.*?)/",        // 可选：从详情 URL 正则捕获书 id（组 1）供 toc.url
    "bookName": "…", "author": "…", "intro": "…", "category": "…",
    "coverUrl": "…", "latestChapter": "…", "lastUpdateTime": "…", "status": "…"
  },
  "toc": {
    "url": "…/xiaoshuo/%s/",       // 可选：详情/目录分页时必填
    "list": "…",                   // 可选：先取容器 HTML 二次解析
    "item": "#list dd a",          // 必填
    "reverse": false,              // so-novel isDesc 同义
    "nextPage": "#indexselect option"
  },
  "chapter": {
    "title": "h1",                 // 必填
    "content": "#content",         // 必填
    "paragraphTagClosed": true,
    "paragraphTag": "<br>+",       // paragraphTagClosed=false 时的切段正则
    "filterTxt": "广告一|广告二",    // 单条正则（| 连缀），逐条 DeleteAllString
    "filterTag": "div, script",    // 元素连内容整删
    "nextPage": ".bottem a:last-child",
    "endPattern": "…/\\d+\\.html$" // 末页 URL 正则；另带通用启发式（§2.7）
  },
  "crawl": { "concurrency": 5, "minIntervalMs": 1000, "maxIntervalMs": 2000,
             "maxRetries": 3, "retryMinMs": 2000, "retryMaxMs": 4000 }
}
```

不支持即校验失败（fail-closed，带字段名原因）：`@js:` 步骤、XPath 选择器、`@src` 以外属性后缀只收 `@href`。选择器语法 = **CSS only**（goquery/cascadia；上游 XPath 主要出现在需 `@js:` 的源，本就 v1 不支持）。

### 3.3 引擎形状（t1 已落地，`internal/booksource`）

| 文件 | 内容 |
|---|---|
| `rule.go` | schema 结构 + `LoadDir/LoadFile/Validate`（fail-closed 校验器：必填段/必填字段/CSS 可编译/不支持语法点名报错）+ 默认抓取参数 |
| `html.go` | `extractText/extractHTML/extractAttr`（goquery）+ `@href` 后缀解析 + absURL 补全 |
| `fetch.go` | `Fetcher` 接口 + `HTTPFetcher`（随机 UA 池、Referer=host、每请求超时、`netclient` 代理语义沿用 app 层注入）+ 抖动/重试（`Sleeper/Rand` 注入）+ `ErrCloudflare`（title 集合检测） |
| `search.go` | 单源搜索（含分页集合、limit、空结果构造详情单条的完全匹配分支）+ `Aggregate`（并发、单源失败降级）+ 相似度（Levenshtein 重写）过滤排序（0.25/回退>0/短查询精确加权/另一字段 tie-break） |
| `toc.go` | 目录解析（list 二跳、item、分页 option/锚点集合并发抓按页序归并、reverse、`Parse(start,end)` 范围） |
| `chapter.go` | 正文抓取（单页/分页走 next 按钮、endPattern + 通用末页启发式、正文空=显式错误） |
| `clean.go` | §2.8 全链（PUA 不可见字符/实体/filterTxt/filterTag/开头标题剥离+标题序号归一/空标签）+ §2.9 排版归一（DOM 改名路线） |
| `assemble.go` | TXT 组装（首页信息头 + `第N章 标题` + 段落行；UTF-8 / GBK 可选） |
| `rules/rule-template.json` | 出厂唯一内置规则文件：**模板**（全字段注释），供用户照抄自编 |

## 4. 与上游的有意偏离（全部码内注记）

| # | 偏离 | 理由 |
|---|---|---|
| D1 | 不搬任何代码/规则 JSON，机制与常数（0.25 阈值、抖动区间、权重 12/6/3）**重推导** | 上游 AGPL-3.0 vs gaea 私有版权，硬约束 |
| D2 | `link` 显式字段；so-novel 复用 bookName 选择器取 href | 显式优于隐式；规则可读 |
| D3 | 排版归一闭合标签路线用 **DOM 改名**（直接子级）而非带回引用正则 | Go RE2 无反向引用，DOM 路线更稳 |
| D4 | `@js:` 步骤、XPath、`nextPageInJs` 不做，校验器/解析器**如实报不支持** | 引 JS 引擎（goja/GraalJS 同类）是独立刀路，先立门槛再拍板 |
| D5 | 简繁转换不做 | HanLP 级依赖面；繁体源 v1 不覆盖，观察池 |
| D6 | CF 只检测（title 集合）+ 显式 `ErrCloudflare`，**不接外挂绕过服务** | 合规边界：检测是诚实报错，绕过是对抗，不做 |
| D7 | 章节缓存目录+合并的「文件即中间态」不做，引擎内存直通组装 | 桌面单机内存够用；t5 若做增量/断点再评估 |
| D8 | 解码用 `bookimport.Decode` 五级链，无 HTML meta charset 探测 | 链尾兜底已覆盖 GBK 站点主流；meta 嗅探观察池 |

## 5. 刀路

| 刀 | 内容 | 状态 |
|---|---|---|
| t1 | `internal/booksource` 引擎 P0（本档 §3.3 全表 + 夹具单测；零绑定零前端零 app 接线） | ✅ 本刀（非版本刀，避开并行在途 release 的绑定面） |
| t2 | app 绑定 `SinBookSource*`（List/Search/Toc/Download + 下载进度事件流；`sinRoot()/rules|books` 落盘；netclient 代理注入） | ✅ 2026-09-14：**该线自 DSH 移交本线执行（用户拍板）**。落地=SinB +6→20（Search/Toc/Download/DownloadCancel/BooksList/BookDelete，总面 660→666）；**规则目录合流收敛**：小说侧 t1 先落地共享层 %配置目录%/gaea/booksource/rules，按「以先落地为准」弃本档 sinRoot()/rules 口径，成书产物落 sinRoot()/books（sin 数据面）；搜索/目录/取消登记簿与小说侧同源复用（searchBookSources/tocBookSource/bookImportRuns，job 前缀 bss_）；成书=AssembleTXT utf-8（详情名缺省回填与文件名同源）、同名追加序号、单章失败不中断 Failed 如实带出、删除有路径护栏（限成书目录内 .txt）；进度事件 sin-booksource:<jobId> 每 20 章一报；测试 +8（成书落盘/兜底名/Failed/无规则拒绝/同名序号/清单排序/路径护栏/起跑预检），全量 app 回归绿 |
| t3 | 原罪右栏「书源」卡：搜书 → 结果表（相似度序/书源标注）→ 下载进度 → 成书入口 | ✅ 2026-09-14（v4.294.0）：右栏第五页签「书源」（sinPanelState 兼容旧值）；SinBookSourcePanel 自包含流程——候选表 HasRule=可下载/免规则「仅参考」禁用（t2 fail-closed 的 UI 同款）、warnings 直显、目录预览+范围选择（默认全本钳 [1,total]）、进度每 20 章上报+取消（关面板不打断后台）、事件 sin-booksource:<jobId> 三路必退订、成书清单（新→旧+Popconfirm 删除）；dev mock 诚实空态；vitest +6（坑复训：antd 两字按钮插空格、listitem 可访问名来自 title 属性） |
| t4 | 原罪工具接线：`book_search`/`book_download` 进 `sinToolOrder`（sin_tool_web.go 委托同款），故事中可拉素材 | ✅ 2026-09-14（非版本刀）：sinToolOrder 联网区插入两工具（7→9）——`book_search` 只读，紧凑候选清单（可下载排前、≤12 行、warnings 计数化，空态如实说明规则目录与模板路径）；`book_download` 落盘成书进 sin 数据面并**回带 ~5000 rune 正文节选**（模型无文件工具，节选才是能直接化用的部分；外层另有 6000 rune 工具闸）、长书建议给范围、无规则来源 fail-closed 拒绝并引导改选；网络通道 `sinBookToolOpts` 注入（生产零值/测试假站）；下载超时 10 分钟（故事拉素材不建议整本等） |
| t5 | 拆书联动：成书 TXT 一键进小说板块导入向导（tail 引擎已就绪）；EPUB 导出（go-epub 已在依赖） | ✅ 2026-09-14（v4.296.0，线收官）：成书行「送入小说」=sin 前端直调既有 ImportNovelBookEx（full 全本走同一落库链，零新后端耦合）+成功后派发 NAVIGATE {page:novel} 跳书架（sin/novel 同 play 可达）；ImportNovelBookEx 按「legacy 直调转正」升进 bridge NovelBindings 契约面（NovelImportResult 载荷，facets 归 play）；SinBookSourceBookExportEpub（SinB +1→21，面 667）=TXT→同名 .epub 同目录覆盖语义，sinBookParseTxt 确定性逆解析自组装格式→go-epub；护栏抽 sinGuardBookPath 共用；删除成书连带清同名 .epub；mock 诚实拒绝两件 |

## 6. 拍板池 / 风险

1. **出厂规则集**：gaea v1 不带任何真实站点规则（AGPL 数据 + 站点合规双重考虑），用户自备。若后续要出厂，路线=照 `rule-template.json` **自行编写并实测**的目标站点规则，逐源入库；不回移 so-novel 文件。
2. **站点合规**：规则指向的站点多为转载聚合站；引擎是通用机制（与 web_fetch 同性质），站点选择权与使用责任在用户。原罪板块定位（成人向创作）与书源内容的关系由用户自持。
3. **@js: 书源缺口**：quanben5 类加密搜索源需要 JS 步骤引擎（goja）。需求出现时独立拍板，不默认进内核。
4. **CF 站点**：只报错不绕过；用户若自备绕过服务属于壳外行为。

## 7. 验收（t1）

夹具驱动（`internal/booksource/*_test.go`，测试内嵌假站点 HTML，零真实网络）：

- 规则：模板文件可解析通过校验（防模板与 schema 漂移）；坏规则逐类报错（缺必填/@js:/XPath/坏正则/坏 CSS）；
- 抽取：text/html/`@href`/absURL 补全；
- 搜索：单源解析 + limit 截断 + POST 表单槽位 + 分页集合；聚合单源失败降级；相似度排序（书名搜/作者搜自动判别、0.25 过滤、回退>0、tie-break）；
- 目录：分页 option 集合 + 按页序归并 + reverse + `Parse(start,end)` 范围；
- 正文：单页/分页拼装 + endPattern 与通用末页启发式 + 正文空报错；
- 清洗：PUA/实体/广告正则/整删标签/开头标题剥离（含正则元字符标题）/`1.章节名` 归一/空标签/两路排版归一；
- 抓取纪律：抖动区间与重试线性放大（注入 Sleeper/Rand 断言调用序列）、并发钳制、ctx 取消零重试、CF title 检测；
- 组装：TXT 信息头 + 章序 + GBK 编码回环（`bookimport.Decode` 反解）。

## 8. t1 验收对拍记录（2026-09-13，Codex 侧补测）

> 完整对拍报告（红→绿证据、负向对照、逐条验收表）：`.gaea/reports/booksource-t1-review-2026-09-13.md`

- **结论**：主干符合 §2 机制 / §3.2 schema / §4 偏离口径。补测 18 例（23 → **41 例全绿**），覆盖率 **81.9% → 88.0%**；`go build ./...`、`go vet`、`gofmt -l` 全绿，`go test ./internal/...` **118 包全 ok（0 FAIL）**；未执行任何丢弃工作区改动的 git 命令。
- **实修 6 项**：①闭合路线只认 `#text`（`Children()` 不含文本节点，分支不可达）致 `<div>` 逐段源被压成一整段 → 改 `Contents()` 按文档序逐子级成段；②单章分页触顶**静默截断正文** → 改为显式报错（负向对照已验证测试有牙）；③`chapter.nextPageInJs` 被 JSON 未知键**静默丢弃**（§4 D4 要求如实报不支持）→ `Parse` 前置点名拒收；④`search.url`/`toc.url` 被当正则校验 → 合法 URL 含 `[]`/`(` 时误拒，改回按 URL 模板对待；⑤`baseUri` 声明后无消费方（死字段）→ 接入相对链接补全基准（页址优先、缺失回落，向后兼容）；⑥目录分页去重删除元素后旧索引未失效（仅因上游去重侥幸不爆）+ 已抓文档被重复抓取 → 索引重建 + 文档复用。
- **观察项（交 t2 接线拍板）**：范围下载 `Order` 从 1 重编号（tail 模式成书章号会与真实章号错位，改点一行）；CF 挑战页在非 200 时只报 `HTTP 403`（要精确分类需改 Fetcher 契约）；`Disabled` 是否覆盖下载面；POST 的 URL 槽位未转义。
