# 全仓审计第 21 批 · duplication 余量四线（14 条收敛）· 2026-10-03

> 接续 [round-22（批次二十）](round-22-p1-batch20.md)。取对账地图「duplication 余量 ~15」中的 14 条（四线）；语义冲突簇批 20 已毕。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `a3cd2ba7`（批次二十）；工作树干净。

---

## 〇、线切分与条目（14 条，Go 包互斥）

| 线 | 条目 | 足迹 |
|---|---|---|
| 线 1 | IN1-03 章节编号解析四套（+迁移路径 Sscanf 第 5 套）· IN3-12 导出章节循环四份 · IN3-13 Markdown 结构识别三份 | internal/export + project + novelcontext（docmd 只评估禁改） |
| 线 2 | GA3-03 两套 BM25 · GA3-04 五套分词/相似度 · IN2-11 三套 BM25 三份分词器 | internal/gaea/bm25+search+memory+gaea/memory + novelstyle(segment) |
| 线 3 | GA2-06 dag 双套档存储 · AP5-06 重复记忆检测两套 · GA6-05 标题/价格匹配 4 包分叉 | internal/gaea/dag+pricefeed+costimport+costinquiry+costref+cost + app/memory_meta |
| 线 4 | FE7-05 表格/画布行派生镜像 · FE7-07 分组汇总两份取数 · FE7-08 上报列定义平行 · FE7-12 拖拽/缩放视图间重复 · FE5-03 硬闸类型重复 | frontend/src/schedule + pages/CreatePage |

**纪律**：全部来自对账存量（从未预核）——先复核后动手、证伪也是交付；线 2 是全批最 delicate（分词/BM25 直接决定检索结果，**行为冻结=第一约束**，统一必须以「参数化复现各自现状」方式落地）。

## 一、逐线落地

### 1.1 线 1 · export 域三条（子代理 + 主代理复核）

**现场复核：解析实为六处（审计五套+复核新增 graph/consistency.go:90 第三份正则）；行号全漂移；证伪一条——bookimport 不是章节号解析消费方（它解析正文「第X章」标题非文件名）**。四套口径容忍度各异实测成表：正则三分支（恰 3 位+可选单小写+严格 .md）/ 纯数字排除分支（N5 刻意收窄）/ 前导数字 / 首段连续数字（宽于 leadDigits）/ 迁移 Sscanf（探针实测与正则**双向不包含**）。

**收敛（按审计放行的收窄形态）**：`project.ParseChapterFileName(name) (num, branch, ok)`（口径对照表注释列四保留站点理由）+ `forEachChapter(scope, fn(ch,label,content))`（EPUB 需 num/branch 拼文件名，三值签名；EPUB 中途失败不写 FailedChapters 的历史行为逐字保留）+ `readWorldviewWarned` + `markdownBlockKind`（html/docx 共用；docmd 禁改，对照钉在 export 侧测试——docmd 发射词汇可被 markdownHeading 回收、`####` 钳制落正文）。ExportHTML 是结构不同的第五循环（摘要覆写/世界观静默跳过），未并入（并入会改行为）。

**金样三重锁定**：临时探针项目产出 TXT/MD/EPUB/DOCX+mainlineOnly 变体，重构前后 diff 除 go-epub 运行时钟外**逐字节相同**。**主代理变异**：分支解析删段 → `TestExportAll_MainlineOnlySkipsBranches` 红（`缺少 "分支甲正文。"`）→ 还原绿。留池：stats.go:33 与 graph:90 可一行改调（等价钉背书）；迁移 Sscanf 与 novelcontext 宽口径刻意保留。

### 1.2 线 2 · BM25/分词族（子代理 + 主代理复核）

**现状对照表（实测）**：分词 6 套 / BM25 打分 3 套。三类族谱：bm25 族（cost 消费，两字去重无停用词）、gaea/memory 族（同算法族+停用词+两字不去重，IDF 逐位一致）、**异族三种**（internal/memory k1=1.5 不同算法族；search.Tokenize 不小写化 TF-IDF 族；textsim Dice 集合语义族；novelstyle 写作统计分词）。

**统一形态（行为冻结第一约束的落地）**：①**BM25 公式单源** `bm25.Params.TermScore`（三套 IDF 逐行对照逐位一致——`log(1+x)` vs `log(x+1)` IEEE 加法可交换，因子式收敛零行为差）；②**分词开关单源** `TokenizeOpts{Stopwords, DedupeTwoChar}`（开关↔消费方映射对照表注释），gaea/memory 的 SearchIndex **整体改 bm25.Ranker 薄封装**（倒排/tf/df/打分全下沉）；③**internal/memory 不接分词**（不同算法族，k1=1.5 真实差异保留），公式委托 `idx.params.TermScore`（四条理由写 Index 注释）；④**证伪出族三种**（口径注释+对照表指针，零逻辑改动）；⑤查明 internal/memory 的 `df==0→0.5` 平滑分支**不可达**（f>0 蕴含 df≥1），保留。

**行为冻结证明**：快照测试先钉现状（三口径 14 句语料含全部分歧维度/跨包 pin 取自 gaea/memory 真实消费方输出/双参数排序+分值 pin）→ 改动后一字不改全绿；cost 批 16 golden 原样绿。

**数学发现（写入测试注释与 Index 注释）**：固定 b 下 k1 **数学上不可能改变两两排序**（g(a,n)=g(b,n′) ⟺ a·n′=b·n 与 k1 无关，scratch 双参数全语料实证一致）——k1 变异的红落在分值快照而非顺序。

**主代理复核与变异**：十触达包独立复跑全绿；亲手重做变异「internal/memory K1 1.5→1.2」→ `TestIndexRetrievalFreezeSnapshot` 红（五查询分值全偏、顺序不变：`Search("通背拳") = [3.13, 2.95], want [3.15, 2.94]`）→ 还原复绿。

### 1.3 线 3 · Go 杂项三条（子代理 + 主代理复核）

- **GA2-06**：泛型 `fileStore[T]`（dir+label+id/createdAt 提取+beforeSave 钩子，save/get/list/del 四方法）；**钩子先于 MkdirAll**（保 Template「坏形状不产生半途目录」语义）；空集 `[]T{}` 单点保留在 list()。**主代理变异**：空目录返回 nil → `TestTemplateStoreRoundtrip` 红（`目录不存在 List 应返回非 nil 空集`）→ 还原绿。
- **AP5-06 第三条路（行为冻结）且前提被修正**：主代理给的「现场证实两套输出实际一致则直接映射」**不成立**——蒸馏=确定性（同名异写/逐字相等/仅同 space/上限 8/保较新/宁漏勿误）vs 面板=模糊 textsim ≥0.55（不分区 space/无上限/保名称靠前），**输出不等价**。落地：函数头注释钉死口径差异+对照测试把分叉钉成机器事实（同批事实：蒸馏只报名异写对、模糊近似文对不报、同名对保序方向相反）；**拍板池登记**：两套是否收敛（映射=改绑定输出/移除=744→743），未拍板前冻结。
- **GA6-05 现场重画**：真同型副本对=pricefeed.matchRows↔costimport.MatchRows（含 GA6-09 留痕各抄一份）+价格解析两份+归一化两份（contentband 实在 cost 包内部）；**costinquiry/costref 是第三方言**（剥括号尾注/带码不回退），已单点收敛于 costinquiry.MatchTitle，强套 MatchIndex 会改行为——**「4 包收敛」口径证伪，审计 fix 落点收窄为两同型副本**。落地：cost 包 `MatchIndex/NewMatchIndex/LoadMatchIndex`（GA6-09 留痕文案逐字）+`ParsePrice/ParsePriceWithOpts`（全角￥/逗号归一单点）+`PriceOpts{RequireDigit,Round2}`（钉住的历史口径：costimport 方言下 "Inf"/"NaN" 会被当价格、pricefeed 数字预检挡掉——本刀只参数化，收敛裁决入拍板池）。

**主代理复核与变异**：六触达包独立复跑全绿（cost 批 16 golden 原样）；亲手重做变异「fileStore 空目录返回 nil」→ `TestTemplateStoreRoundtrip` 红 → 还原绿。

### 1.4 线 4 · 前端 schedule 族五条（子代理 + 主代理复核）

- **FE7-05**：`gantt/rowDerive.ts` `deriveRows()`（一次遍历产出两窗格共用派生序列，判别联合）+ `GanttRowShell` 统一行外壳；判定唯一源=私有 `synGroupOf`（返回收窄变体避免布尔假窄化）。
- **FE7-07 冻结现状（量化先行）**：对照测试量化出筛选下两口径差 **3 个工作日**（导出=全子孙叶 vs 画布=可见叶）；裁决=冻结——导出面 `buildGanttExportSvg` 根本没有筛选/折叠输入，「统一到可见叶」必改导出签名+字节；文件头已申明「图面是发布物，不随工作台交互态漂移」；口径对照表固化+同值断言（无筛选时同值钉死）。
- **FE7-08**：EXP_COLS 改 `colsByKeys(GANTT_REPORT_KEYS)` 派生+**键序长位断言**——附带发现 `colsByKeys` 保 GANTT_COLS 序、入参序无关，既有用例只断 label 存在性，这正是键序断言必须存在的实证。
- **FE7-12**：`useDragGesture`（move/moved/commit/end 四点注入，up=detach→moved 才 commit→end，Esc=detach→end）+ `useFitView`（每视图注入 min/max——**参数值全冻结**：AOA [0.1,2] vs PDM [0.2,2] 未顺手统一）；GanttView dayW 2~40px 是另一套单位机制，按最小行为风险不并入。三处实现序差已论证无观测差并写入注释。
- **FE5-03**：CreatePage 删 `PlanGateBridge/CreateChapterBridge/toPlanGateReport`（净 −60 行），直接用已导出 `PlanGateReportView` 类型化调用；运行时守卫与降级文案逐字保留。申报：畸形载荷（Go json 不会产出）会显 "undefined" 而非旧空串——真实绑定面字段恒字符串，审计 fix 明示的接受项。

**主代理复核与变异**：tsc 0/eslint 0/定向 vitest 160 例独立复跑；亲手重做变异「EXP_COLS 派生键序反转」→ ganttExport 测试收集期红（键序断言在册）→ 还原 18/18 绿。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥按 Go 包划**（export+project+novelcontext / bm25+search+memory+gaea-memory+novelstyle / dag+price 族+app-meta / 前端 schedule+CreatePage） | **有效**：并行期两线互见对方暂态编译坏（线 2 见 project、线 3 见 export），均未触碰、等自愈；收线全树 build/vet 0 |
| 2 | **主代理给出的路线前提被现场修正两处**（AP5-06「输出实际一致则映射」不成立→冻结第三路；GA6-05「4 包收敛」重画为两副本+两第三方言） | 主代理任务书同样要被现场复核——「宁窄勿宽」在存量上再次获胜 |
| 3 | **行为冻结的三种形态本批齐备**：参数化复现（线 2 开关）/骨架单源+谓词保留（线 1）/冻结+口径对照表+差异钉死（线 3 AP5-06、线 4 FE7-07） | 按条目性质选形态；「统一」不是默认答案——FE7-07 量化后冻结、AP7-10（批 19）拒绝合并同族 |
| 4 | **数学性质入册**：固定 b 下 k1 不改变两两排序（BM25） | k1 变异的回归防护=分值快照而非顺序快照；测试注释与 Index 注释双落点 |
| 5 | **四组变异主代理亲手重做**：分支解析删段/空集返回 nil/K1 减半/EXP_COLS 键序反转——全红→绿 | 线 4 的 EXP_COLS 变异落在收集期（比断言红更早）——同样是有效红 |
| 6 | **对账存量批的偏差密度持续**：14 条中证伪/重画 4 处（bookimport 消费方/xlsx 同构族外/costinquiry·costref 第三方言/AP5-06 输出不等价） | 存量复核纪律（批 19 确立）持续有效 |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台终值）

**本地快闸（`scripts/ci.ps1` 前台单次，CI OK / exit 0）**：version drift OK（4.454.0）· `go build ./...` 0 · `go vet ./...` 0 · **golangci v2.14.0 0 issues** · **`go test ./... -count=1` exit 0** · 前端 lint 0 error / build ok · **vitest 433 文件 3752 例全绿**（批 20 为 431/3735，+2 文件 +17 例：rowDerive 6 + gestures 8 + 骨架增量）· E 系列 OK · 卫生四查 OK。

**过程事故再犯（如实登记）**：ci 日志在抄录 Tests 终值前被删（**批 19 刚入册的教训第二次违犯**）——文件数从日志残句取得（433），例数由单独全量 vitest 复跑实测补齐（3752）。教训升级：**rm 清理必须放在台账终值写完之后**，且抄数与清理不许在同一条命令里。

**提交事故（当场发现当场修复）**：首次提交 `d96850e9` **漏了两个新文件**（`internal/gaea/dag/filestore.go`、`internal/gaea/cost/matchindex_test.go` 不在显式 add 清单）——已提交树上 dag 包编译不过（fileStore[T] 被引用但未入库）。发现方式=提交后 `git status` 仍残留两个 `??`；**立即 amend 补齐**（终态 `860144db`，43 文件），HEAD 复验 build+test 绿。教训：**枚举 add 清单必须对照 `git status` 的 `??` 段逐一勾销，提交后必须复验「树干净 + HEAD 可编译」两件事**（本仓禁 `add -A` 的纪律下，新文件漏 add 是显式枚举法的固有风险，勾销清单是唯一防线）。

**守卫**：`check-primitives --strict` 无新增 · `check-test-ctors` 364 零新增 · `check-contract-drift` 无新增 · `check-bindings-drift` OK@744 + 死绑定检测 109 在册零新增。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| IN1-03 | 章节号解析骨架单源 `ParseChapterFileName`（六处清点、四保留站点口径对照表；金样逐字节） |
| IN3-12 | 导出循环四份 → `forEachChapter` + `readWorldviewWarned`（EPUB 中途失败语义逐字保留） |
| IN3-13 | Markdown 块判定 → `markdownBlockKind`（html/docx 共用；docmd 对照钉） |
| GA3-03/04/IN2-11 | 分词 6 套 → `TokenizeOpts` 开关单源（gaea/memory SearchIndex 整体薄封装）；BM25 公式 3 套 → `Params.TermScore`（IDF 逐位一致证明）；异族三种证伪出族；k1 排序不变性数学发现入册 |
| GA2-06 | dag 双套档存储 → 泛型 `fileStore[T]`（钩子先于 MkdirAll；空集语义单点） |
| AP5-06 | 两套检测口径差异钉成机器事实（对照测试）；收敛与否=拍板池 |
| GA6-05 | 匹配两副本 → `cost.MatchIndex/LoadMatchIndex`；价格解析两份 → `cost.ParsePrice/ParsePriceWithOpts`；归一化对照注释；Inf/NaN 方言差异参数化钉住 |
| FE7-05/07/08/12 | 行派生+行外壳单源（rowDerive/GanttRowShell）；汇总口径量化（3 工作日差）后冻结+对照表；EXP_COLS 派生化+键序长位断言；拖拽/缩放 hooks（参数冻结） |
| FE5-03 | CreatePage 类型重复 −60 行（直接用已导出 PlanGateReportView） |
| 测试 | 新增 Go 测试 6 文件 + 前端 4 文件；金样/快照/字节钉三重行为冻结证据 |

### 3.4 余量（如实登记）

1. 章节号解析：stats.go:33 与 graph:90 改调（一行事，留池）；迁移 Sscanf 与 novelcontext 宽口径刻意保留（注释对照表已列）。
2. BM25 族：textsim/novelstyle/search.Tokenize 出族（口径注释）；internal/memory 分词不同族不接入（四理由）；「统一检索用分词器」若未来需要=新刀。
3. AP5-06 两套收敛（映射/移除）=拍板池；GA6-05 的 Inf/NaN 方言收敛=拍板池（PriceOpts 已参数化）。
4. FE7-07 若未来导出面需要筛选输入=签名变更（产品决策）；FE6-05 DOM 统一（既存拍板候选）。
5. GanttView dayW 缩放未并入 useFitView（另一套单位机制，最小风险不并）。

## 四、对账地图更新

- **duplication 收敛余量：~15 → ~4**（本批吃 14 条中的 11 条 + FE2-03 证伪销账；余=BM25 族已毕、AP5-06/GA6-05 收敛裁决=拍板、FE2-06 价格源两份、零散）。
- 语义冲突簇：批 20 已毕。**剩余大头=god-file 大拆 ~27（每条独立刀）+ coupling ~20（拍板）**。
- 下一批候选：duplication 最后的零散（FE2-06 等）+ god-file 大拆试水（挑最小的一条立刀验证配方）或转拍板池批量裁决。
