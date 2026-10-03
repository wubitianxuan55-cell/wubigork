# 全仓审计第 25 批 · god-file 大拆（GA2-01 websearch + IN2-12 modelengine/stats）· 2026-10-03

> 接续 [round-26（批次二十四）](round-26-p1-batch24.md)。god-func 三刀收官后转入 **god-file 大拆**：本批两线，均为**同包纯文件拆分**（原位搬移零逻辑改动），主代理直做。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `60b2fc36`（批次二十四）；工作树干净。

---

## 〇、选点与预核

- god-file 存量 21 条（round-21 §〇记 ~27 含 god-func 重叠），选点原则：不需拍板、可纯搬移、测试网厚。**GA2-01**（websearch.go 921→925 行：工具+引擎注册表+HTML 解析同文件）与 **IN2-12**（modelengine/stats.go 758 行：计价域与统计记录器混装）胜出。
- 拆分方式=**同包内文件重组**：符号全部包内可见，编译不敏感、纯组织性改动；每文件 import 按各自消费重算。

## 一、线 1 · GA2-01 websearch.go → 三文件（925 → 331+456+171）

- `websearch.go`（331）：工具本体——init 注册/webSearch 实现/Execute 编排（并行扇出+首胜收集）/buildEngines+searchEngineOrder/formatResults/域名策略过滤（searchPolicyRestricted+filterSearchResults）/url 直抓（fetchSearchPage）/truncate。
- `websearch_engine.go`（456）：搜索引擎 seam——SearchEngine 接口+kind 常量+默认序+注册表（Register/New/Kinds）+6 引擎实现（local/public SearXNG、Bing、DDG Lite、Tavily、Brave）+引擎侧共享 HTTP（searchHTTPClient/searchProxyURLFor/doSearchRequest）+SearchResult 契约。
- `websearch_html.go`（171）：Bing/DDG Lite SERP 解析（parseBingResults/bingResultFromBlock/parseDDGLiteResults）+nethtml 节点工具五件。
- 方法：`sed -n '行号区间 p'` 按行号字节精确抽取 + `cat` 拼手写文件头（避免手抄漂移）；行数对账闭合（900 内容行+3 seam 空行+22 行旧 import 块=925）。

## 二、线 2 · IN2-12 stats.go → 计价域分家（758 → 588+182）

- `modelprice.go`（182）：内置定价表 modelPricing+modelPrice+normalizeModelID（含专用正则 reDateSuffix/reDate8，包内无第二消费方实证）+estimatePrice/estimatedCostFor/EstimateCostCNY+defaultUsdCnyRate。目录价/用户价（glmCatalogPrice/engineCatalogPrice/userEnginePrice）本就在 catalog_models.go/user_price.go——计价四域自此各归其位。
- `stats.go`（588）：用量契约类型+statsRecorder（load/save/record/summary/prune）+Manager 接线。import 重算：`regexp` 随计价域走，`math` 双侧共用（EstimateCostCNY 汇率守卫 + usdToCNYRate）。

## 三、等价证明与门禁（两线同配方）

- **双证零漂移**：`grep -hE "^(func|type|var|const)" | sort` 符号清单 before/after diff 空 + `go doc -all` 全包输出 before/after **逐字节一致**（两包分别验证）。
- 包测试：builtin 12.6s 绿（含 registry/policy/websearch 四件套）；modelengine 3.0s 绿（含 estimate_cost_test 计价用例，直接覆盖搬移代码）。
- 前台 ci **exit 0**：golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例全绿 / 卫生守卫绿。
- 提交后复验「树干净 + HEAD 可编译」；新文件（websearch_engine.go/websearch_html.go/modelprice.go）进显式 add 清单。

## 四、god-file 余量与下一批

- 已销：GA2-01、IN2-12。god-file 余 ~19（browser manager 1068/fold.go 1041/App.tsx 1041/mpp.go 1035/project.go 1011/controller.go 963/agent.go 975 等）。
- 下一批候选：同配方续拆（GA5-08 browser manager 五职责 / GA5-04 fold.go 五职 / IN3-03 mpp.go 版本布局分家）；coupling ~20 与零散死码继续等拍板。
