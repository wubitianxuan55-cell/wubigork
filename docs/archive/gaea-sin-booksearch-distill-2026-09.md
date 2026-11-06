# 原罪板块 · 泛搜索与免规则解析蒸馏（owllook）——规格与刀路

> 状态：引擎刀已落地（非版本刀）· 类型：外部项目蒸馏规格（机制清单 + gaea 落点 + 偏离）
> 上游：https://github.com/howie6879/owllook（Python/Sanic 小说垂直搜索引擎，本地克隆 `clones/owllook`，**Apache-2.0**）
> 前序：docs/gaea-sin-booksource-distill-2026-09.md（so-novel 蒸馏，`internal/booksource` t1 已落）
> 关联板块：原罪（闲庭 `sin`）；代码落点 `internal/booksource/websearch.go|guess.go|cache.go`

## 1. 定位：与 so-novel 路线的互补关系

用户口径：**「owllook，继续蒸馏这个的搜索引擎给原罪」**。

so-novel 路线（已落 t1）= **逐站搜索**：每个书源自带搜索端点规则，规则到哪搜到哪。owllook 路线 = **泛搜索**：拿书名去通用搜索引擎（360/baidu/bing/duck_go）检索，从 SERP 里滤出小说站候选，再用**极少量规则 + 免规则启发式**解析目录与翻页。两条路线在 gaea 合流为 `internal/booksource` 的双搜索面：

- `Aggregate`（已有）：有规则书源的站内搜索；
- `DiscoverySearch`（本刀）：通用引擎泛搜索，产出**站点候选**（含「该站有无书源规则」标记），无规则站点也能免规则出目录。

法务：owllook 为 Apache-2.0（许可宽松于 so-novel 的 AGPL），机制仍一律重推导；本次无知识资产收录需求（引擎规则是自编功能数据，黑名单域名系站点策展数据**不出厂**，见 §6）。

## 2. owllook 机制清单（file:line 对齐）

以下 `O/` = `clones/owllook/owllook/`。

### 2.1 泛搜索（novels_factory）

- 引擎优先级 `ENGINE_PRIORITY = ['360','baidu','bing','duck_go']`（`O/config/rules.py`）；引擎 URL/参数在 `O/config/config.py`（bing `/search?q=`、ddg `/html`、so `/s`、baidu `wd=`）；
- **query 塑形**：`'{书名} 小说 免费阅读'` / `intitle:{书名} 小说 阅读`（`O/views/api_blueprint.py:87`、`O/fetcher/novels_tools.py:24`）——垂直化靠查询词模板而非引擎 API；
- 每引擎一个 SERP 解析器（`O/fetcher/novels_factory/{bing,baidu,duck_go,so}_novels.py`）：bing/ddg 取 `h2 a`、baidu 取 `h3.t a`、360 各自类名 → {title, url, netloc}；
- **重定向解包两路**：baidu 跳转链 HEAD+allow_redirects 解真实 URL（5s 超时，`baidu_novels.py:53-67`）；ddg 链接解 `uddg` 查询参数（`duck_go_novels.py:28`）；
- **SERP 净化启发式**：剥 `index.html` 尾；仍含 `.html` → 判非目录页丢弃；解析结果只剩站点根 → 丢弃；engine 自家域名/baidu/baike → 丢弃（`bing_novels.py:27-30`）；
- **黑名单域名** ~150 个（正版站/百科/不解析站，`O/config/rules.py:8-30`）——站点策展数据；
- **REPLACE_RULES**：个别站 SERP 链接与真实目录路径不同，按域名做 old→new 串替换（`O/config/rules.py:33-58`）；
- 结果打标：`is_parse`（该域名有解析规则）/`is_recommend`（有追更规则）。

### 2.2 免规则目录（招牌机制）

`O/fetcher/extract_novels.py:16-40 extract_chapters`：

- 章节锚点正则：`<a>…第?[一二两三四五六七八九十○零百千万亿0-9１２３…０-９]{1,6}[章回卷节折篇幕集]…</a>`——对**任意页面**凭文本形态认出章节链接（含全角数字、卷/回/节/折/篇/幕/集）；
- 锚点集合重喂 bs4 抽 href/text，`urljoin` 补全；
- **按 URL 数字尾排序**：`index = int(path 去扩展名后最后一段)`（`extract_novels.py:33`）——上游 `reverse=True` 取最新序（追更场景）；`int()` 对非数字尾直接抛异常（上游缺陷，gaea 防御化）。

### 2.3 免规则翻页/上下章

`extract_novels.py:43-85 extract_pre_next_chapter`：锚点文本正则 `[第上前下后][一]?[0-9]{0,6}[页张个篇章节步]` 匹配「下一页/上一章/下一篇」类链接；噪声词表（`后一个`/`天上掉下个`）剔除；自链（与当前 URL 相同）判末页。

### 2.4 正文解析（仍需规则）与标题

`O/fetcher/cache.py:22-65`：正文按 `RULES[域名].content_selector`（`{id|class|tag}` 三形）抽取——**上游不对正文做免规则猜测**；标题三级回退：`<title>` 正则 `(第X章…)[_,-]` → `h1` → `<title>` 全文。

### 2.5 缓存与规则资产

- aiocache/Redis：搜索结果 TTL 3 天（259200s）、正文/目录 TTL 300s（`cache.py:20,63`）；
- RULES/LATEST_RULES：每域名极简规则（content_url 模式 -1/0/1/netloc + `{class|id|tag}` 选择器，`上游 so-novel 的规则定义文档（规则定义.md）`）；追更走 og:novel:latest_chapter meta（gaea 的 BookDetail og: meta 双路已覆盖，书源规格 §3.2）。

## 3. gaea 落点设计（internal/booksource 三件新增）

| 文件 | 内容 |
|---|---|
| `websearch.go` | `SearchEngineRule`（数据：url/queryFormat/result/title/link/linkParam/redirectHosts/crawl）+ `LoadEnginesFile` + `DiscoverySearch`：SERP 抓取 → 候选抽取 → 净化启发式 → `WebCandidate{Engine,Title,URL,Host,HasRule}`（HasRule=规则目录同 host 有书源，对应上游 is_parse）|
| `guess.go` | `GuessTocEntries`（章节锚点正则含全角数字/卷回节折篇幕集 → urljoin → URL 数字尾升序，非数字尾稳定保序）、`GuessNextPage`（翻页锚点正则+噪声词+自链判末页）、`GuessChapterTitle`（title 正则→h1→title 三级回退） |
| `cache.go` | `TTLCache`（泛型、注入时钟、互斥 map）——DiscoverySearch 结果缓存（默认 30 分钟，0=关），t2 绑定层亦复用 |

集成面（既有 Toc/Chapter 的**回退路线**，规则可选化）：

- `Toc`：`toc.item` 为空 → 免规则路线（首页 GuessTocEntries + GuessNextPage 有限翻页 ≤20 页合并）；
- `Chapter`：`chapter.title` 为空 → GuessChapterTitle；`chapter.autoNext=true` 且 `nextPage` 空 → GuessNextPage 翻页；
- **正文仍必须有规则**（`chapter.content` 必填不变）——与上游同边界：目录与翻页可猜，正文不猜；
- 校验放宽：`toc.item`、`chapter.title` 改可选；规则文件级 fail-closed 语义不变。

## 4. 有意偏离（码内注明）

| # | 偏离 | 理由 |
|---|---|---|
| D1 | 机制重推导，不搬代码；黑名单域名/REPLACE_RULES 不出厂 | 站点策展数据归用户自备（规格 §6）；Apache-2.0 宽松但屋规统一取道不取器 |
| D2 | 排序改**升序**（上游降序=追更口径） | gaea 是整本下载/阅读，第一章在前 |
| D3 | 数字尾解析防御化：非数字尾不 panic、index=0 稳定保序聚前 | 上游 `int()` 裸抛是缺陷，不复制 |
| D4 | 免规则翻页加 20 页上限 | 上游无翻页上限（单页场景）；防环形链接 |
| D5 | 引擎规则数据化（同书源规则一套纪律：CSS only、fail-closed） | 上游每引擎一个硬编码类；数据化后加引擎=加一份 JSON |
| D6 | 缓存改进程内 TTLCache（单用户桌面），不做 Redis | 依赖面；语义等价（搜索结果短 TTL） |

## 5. 刀路

| 刀 | 内容 | 状态 |
|---|---|---|
| 引擎刀 | websearch.go + guess.go + cache.go + Toc/Chapter 回退 + 校验放宽 + 测试 | ✅ 本刀（非版本刀） |
| t2 | app 绑定 `SinBookSource*`：书源列表/聚合搜索/泛搜索合并面（候选带 HasRule 标记）/下载 + 进度事件；数据根 `sin/rules|books`；TTLCache 复用 | 下刀（绑定面 648，需与并行线错峰） |
| t3 | 原罪右栏「书源」卡：搜索（书源结果+全网候选同表）→ 下载 → 成书 | 依赖 t2 |
| t4 | 原罪工具 `book_search`/`book_download` 进 sinToolOrder | 依赖 t2 |
| t5 | 拆书联动 + EPUB；搜索历史/热搜（owllook owl_ranking 对应物，单机本地化） | 待拍板 |

## 6. 拍板池 / 风险

1. **黑名单/REPLACE 策展数据**：不出厂。用户可按 `上游 so-novel 的规则定义文档（规则定义.md）`（owllook）自行整理成选项里的 BlackHosts（t2 暴露配置面时一并定格式）。
2. **通用引擎可达性**：bing/ddg/so 的 HTML SERP 对无头抓取的容忍度随时间漂移，引擎规则是数据、坏了换一份；baidu 需重定向解包（redirectHosts 机制已备）。
3. **免规则目录的误报面**：正文中同名锚点（如评论区「第一章」引用）会混入——目录页场景误报低，详情页场景靠 start/end 人工校对（t2 UI 逐章勾选时消化）。

## 7. 验收（本刀）

夹具驱动（零真实网络）：

- 泛搜索：假 SERP 页解析（result/title/link）、queryFormat 塑形、linkParam 解包、index.html 剥离、.html 尾拒绝、黑名单/自家域名过滤、去重、limit、HEAD 重定向解包（302 跟随）、候选打标 HasRule、TTL 缓存命中与过期（注入时钟）；
- 免规则：混合数字尾/全角数字/卷回节折篇幕集的目录页 → 条目+排序+去重；非数字尾不 panic 保序；翻页锚点（下一页/下一章/自链/噪声词）；标题三级回退；
- 集成：`toc.item` 空走免规则路线（含翻页合并）；`autoNext` 分页章拼装；`title` 空走标题回退；校验放宽后坏规则仍 fail-closed。
