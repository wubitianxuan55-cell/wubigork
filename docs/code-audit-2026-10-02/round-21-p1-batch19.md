# 全仓审计第 19 批 · 对账存量四线（8 条 duplication 收敛）· 2026-10-03

> 接续 [round-20（批次十八）](round-20-p1-batch18.md)。本批来源=**全量对账**：512 条发现中 P0+P1 共 241 条，与 round-3~20 台账逐一对照后，**106 条从未进入任何批次视野**（此前各批按 round-15 §4.1 四簇候选推进，未做全量扫描）。本批从 106 条中取 13 条可执行 duplication 收敛（四线）。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `305993c4`（批次十八）；工作树干净。

---

## 〇、全量对账（106 条存量分类，后续批次的地图）

| 类别 | 数量（约） | 处置路线 |
|---|---|---|
| **duplication/consistency 收敛（可执行）** | ~28 | 本批吃 13 条，余量后续批次（BM25/分词族 3+5、导出章节循环四份、章节编号四套、estimateTokens 四份〔已在拍板池〕、记忆检测两套等） |
| **god-file/god-func 大拆** | ~27 | 每条=独立刀（runSinStream 265 行、SaveConfig 220 行、controller.go 963 行、CostLibraryView 1132 行、renderNode 1097 行等），排独立批次 |
| **coupling 架构向** | ~20 | 多数需设计决策/拍板（httpbridge 反射面 1045 方法、core 反向依赖 CLI、全局单例族、两套配置生态等），整批拍板后处理 |
| **dead-code** | ~6 | FE2-03（本批）、GA4-08 SpawnPolicy 全族、X1-04、AP9-06 主脑死绑定+口径打架；X1-09 whisper 196 项已拍板-gated |
| **consistency 语义冲突** | ~5 | IN1-04（gate vs review 门槛相反）、AP3-03（证据链空间红线恒 work）、IN2-07、IN3-09、FE3-04——潜在真 bug，下批优先预核 |
| **perf-smell / doc-drift / safety** | ~4 | AP9-04、X1-08（eslint 门禁）、FE5-04、IN4-06（=已拍板-gated 的 download_and_install 真修法） |

## 〇.1 本批 13 条的审计原文要点（全量对账首次纳入视野）

线 1（internal/app）：AP1-06 大纲树递归 5 份 · AP2-02 书源任务模板三份 · AP5-08 配置快照兜底 4 处 · AP7-02 生成链两份平行循环
线 2（schedule+office）：AP4-11 MPP 版本偏移/魔数双轨 · AP3-02+IN3-06 docx↔pptx zip 读写与纯函数整套复制
线 3（internal/ai）：AP7-03 ComfyUI 工作流成对复制+Warmup 绕过分发 · AP7-10 进度双通道+超时收割复制 · IN2-14 思考预算守护两处
线 4（frontend）：FE2-03 死函数声明 · FE6-05 Markdown 渲染两套通道 · FE7-06 导出层函数副本

**纪律**：这批条目此前从未预核——四线任务书均要求「现场先复核，不成立就证伪」，审计行号按移动靶对待。

## 一、逐线落地

### 1.1 线 1 · internal/app 四条（AP1-06/AP2-02/AP5-08/AP7-02）（子代理 + 主代理复核）

**现场复核（四处全成立，三处审计表述与现场有偏差）**：AP1-06 站点数 **6** 非「5 份」（等价两处被并作一份）；AP5-08 行号漂移（gaeaDreamMode :274）且「兜底行为三处不同」**夸大**（三处形状一致、差异在下游），实际收敛 **5 处**（审计未列的 spaces 内 2 处同型一并收）；AP7-02「只有 internal 链有内存压力预检」**已失效**（两链都有 noteImageGenMemoryPressure），真实差异=override 分支/comfyui 双重试（仅 internal）/Kind 提取/落盘分支/登记口径/终态文案。陷阱抓到一处：`sync` 原实现「无变化」时**仍会 WriteOutlines**（逐字段保留）。

**收敛形态**：`findOutlineNode(nodes, pred)`（前序遍历，命中序同改前；Branch 认否进谓词显式化）；`runBookJob(emit, bookJobSpec)`（spec 七字段：jobPrefix/eventPrefix/panicPrefix/doneType/errWithFailed/doneWithFailed/work；failed 载荷三形保留）；`gaeaCfgSnapshotOrLoad()`（5 处改调）；`runImageGenLoop(imageGenLoopSpec)`（能力开关 `comfyRetries` 仅 internal / `wantKind` 仅 media——**行为冻结**，Media 链补重试=行为新增只申报）。

**主代理复核与变异**：diff 亲读（findOutlineNode 包装形态/能力开关注释/`gaeaCfgSnapshotOrLoad` 锁模式同源）；亲手重做变异「Children 递归注释掉」→ `TestFindOutlineNodeByNumNested` + `TestFindOutlineNode_DeepChildrenRecursion` 双红（`卷下主线节点 ch3 应命中，得到 <nil>`）→ 还原复绿。

**申报单列**：①Media 链补 comfyui 重试（行为新增）；②AP1-06 审计未列的 `findOutlineNodeByNumAny`/`findOutlineNodeByID` 两同型函数未收敛（超清单，留池）；③并行编译坏期用 `go build -overlay` 钉 HEAD 版本完成自验（临时物已删）——并行期自验技巧进 §二。

### 1.2 线 2 · AP4-11 MPP 布局表 + AP3-02/IN3-06 ooxxml 共用包（子代理 + 主代理复核）

**mppLayout 表**：单 struct（行布局/字段偏移/Var2Data 键/asgMode 三态/calandar 键）× 四张表（V9/V12/V14/V14×2013），选择键 `mppLayoutFor(major, appVer)`；魔数/口令位/props 键具名化；switch 分支删除。

**ooxml 包**（`internal/office/ooxml`）：ReadZip/WriteAtomic（条目序+Deflate+CreateTemp+RenameWithRetry 原样）+ 纯函数五件 + `TextElement/EscapeXML`（**方言参数钉死两套历史字节**——`textElement/xmlEscape` 实测字面不同构：docx tab 触发 preserve/`"`→`&quot;` vs pptx 仅空格/`"`→`&#34;`，审计「同构」按字节等价口径成立）。**xlsx 证伪**：全程 excelize 无自有 zip 层，不接入（留池理由在包注释）。段落结构不下沉（docx 侧 link 字段属 WML 语义）。

**字节冻结硬证据**：临时镜像仓以 `git show HEAD` 旧版三文件与新版同跑 17 项探针（docx 条目序+document.xml 字节/pptx 全文件字节/错误原文/4 真实 MPP 样本 JSON SHA）——old/new 全一致。

**测试盲区两处（如实上报）**：①既有 MPP9 真实样本门控对 V9 偏移变异**不红**（只断言 >0 类弱判据，变异后总工期 344→1275769 天仍绿）⇒ 改用 2013 样本硬断言门控反向；②docx/pptx 既有测试对 zip 条目序不敏感（按 part 名读回）⇒ ooxml 新测试补位。

**主代理复核与变异**：diff 亲读（四张表字段/选择键/方言参数）；亲手重做变异「2013 taskDurOff 84→85」→ `TestMppRealSamples2013` 红（`总工期=447406, want 136`）→ 还原复绿。

### 1.3 线 3 · internal/ai 三条（AP7-03/AP7-10/IN2-14）（子代理 + 主代理复核）

- **AP7-03**：`comfyQwenProfile` 表（unet/clip/vae/latent/sampler/auraShift/steps 钳制）+ 两共享构建器；原四构建器转单行转发；`txt2imgWorkflowBuilder` 加 steps 形参（真实生成 8/Warmup 1）；Warmup 改调 `lookupTxt2imgBuilder`（**逐 case 等价表**：空串归一 krea2/前缀回退/z-image-turbo 精确/flux 闭包忽略 steps/default 同文案——全部无差异，新测试假 ComfyUI 捕获 /prompt 请求体钉死）。
- **AP7-10 收窄（审计 overstated）**：「两通道」实测**无重复逻辑可收**（/history ticker 与 /ws 是设计上不同载荷的两个数据源；ticker percent=-1 哨兵 elapsed 保底、ws 报 percent/node）——强行合并必动回调时序（审计自标历史雷区）。只做 `collectResult(ctx, promptID)` 抽取供正常/超时两路共用；取消链一字未动。
- **IN2-14**：`minThinkingBudget=4096` + `clampThinkingBudget(req, wantThinking)`；默认 maxTokens 同源；新增守护测试 4 组（断言写死 4096 字面量防同源循环）。

**主代理复核与变异**：diff 亲读；亲手重做变异「minThinkingBudget 4096→2048」→ `TestClampThinkingBudget_*` 两分支 + `TestStreamDefaultMaxTokensIsThinkingBudgetFloor` 红 → 还原复绿（`ok 8.1s`）。

### 1.4 线 4 · 前端三条（FE2-03/FE6-05/FE7-06）（子代理 + 主代理复核）

- **FE2-03 证伪（零改动）**：`setAllSelected/clearSelection` 现状有正确调用点（:1121-1122）且正确委托 props（:1162/:1165），tsc 0 error——审计描述的「错误调用点+掩盖 FE2-01」在 HEAD 不成立（审计自带「可能已被后续批次清理」条款适用）。
- **FE6-05 成立但「仅差一层 div」不成立**：三路渲染分支**均不同构**（路径 2 多 `data-genui-host` 包裹+行内 code 裸元素依赖 `.md-content` CSS；路径 3 有 mermaid 独占/无 isBlock 闸/整文档管线）⇒ **拒绝审计原 fix**（合并会改 DOM：行内 code 退化无样式、pre 解包语义变化），只统一三者逐字相同的样板为 `extractFence()`（真实路径 `frontend/src/genui/markdownFence.tsx`——审计写的 gaea/lib/ 路径不存在），三处全改接，差异逐处注释写明。
- **FE7-06 照审计收敛**：两份 expFlagSvg 机械 diff 字节级相同、isLinkBinding 语义逐字同；`ganttExport`/`PdmView` 改 export，networkExport 删副本改 import（审计的 pages/ 路径实为 schedule/）。

**主代理复核与变异**：tsc 0 error、eslint 0、vitest 定向 139 文件 1230 例独立复跑；亲手重做变异「extractFence 正则 language-→langauge-」→ **三路全红**（ChatMarkdown.genui 4 + Markdown 3 + extractFence 3，共 10 失败）→ 还原复绿 5/5。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥按 Go 包划**（app / schedule+office / ai / 前端） | **有效**：并行期编译坏（线 2 在改 mpp.go）用 `go build -overlay` 钉 HEAD 自验（线 1 技巧），收线后全树 build/vet 0 |
| 2 | **审计表述与现场的偏差密度**：13 条中 4 条有偏差（AP1-06 站点数、AP5-08 兜底差异、AP7-02 预检差异已失效、FE2-03 整条证伪）+ AP7-10 收窄 | 对账纳入的存量条目**移动靶属性更强**（从未预核过）——「先复核后动手、证伪也是交付」在从未预核的存量上必须升级为默认假设 |
| 3 | **「不同构就保留」两次拒绝审计原 fix**（FE6-05 的 DOM 变化证据、AP7-10 的无重复可收证据） | 收敛的正确粒度再次获胜：同型合并（extractFence/collectResult），语义差异注释钉死；审计 fix 是方向不是图纸（批 17 教训第三次实证） |
| 4 | **测试盲区两处被发现**（MPP9 门控弱判据/office 条目序不敏感） | 反向变异不仅验证收敛，还暴露既有门控强度不足——新钉子补位（ooxml 条目序 pin / mppLayoutFor 表 pin） |
| 5 | **四组反向变异主代理亲手各重做一组** | 递归短路/2013 偏移+1/预算减半/正则 typo——红→绿独立取得；线 4 的正则 typo 变异**三路全红**同时证明 extractFence 单源真的被三路消费 |
| 6 | **线 1 的 `-overlay` 自验技巧** | 并行线破坏编译期时，`go build -overlay` 把他线在途文件钉到 HEAD 完成本包自验（不动他人工作区）——进在册配方 |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台，终值）

**本地快闸（`scripts/ci.ps1` 前台单次，CI OK / exit 0）**：version drift OK（4.454.0）· `go build ./...` 0 · `go vet ./...` 0 · **golangci v2.14.0 0 issues** · **`go test ./... -count=1` exit 0**（app 131.8s/ai 7.7s/schedule/office 全绿）· 前端 lint 0 error / build ok · **vitest 430 文件 3731 例全绿**（批 18 为 428/3723，+2 文件 +8 例：extractFence 5 + exportFlagShared 3）· E 系列 OK · 卫生四查 OK。

**过程事故（如实登记）**：首轮 ci 的 `go test` 步红（exit 1），**失败明细未及捕获日志即被清理（主代理过程疏漏）**；同树零改动复跑全绿（CI OK / exit 0，430/3731）。Go 在册 flaky 清单为空、清单外失败按纪律应按真回归处理——但复跑绿 + 零改动排除了代码回归，定性为**环境性瞬时**（疑似 herdsman localhost:8080 服务在跑时的端口/资源碰撞族，与本批改动无因果）。教训入册：**ci 重定向日志必须留到终值抄录之后**；若再现，先留日志再复跑。

**守卫**：`check-primitives --strict` 无新增 · `check-test-ctors` 364 零新增 · `check-contract-drift` 无新增 · `check-bindings-drift` OK@744 + 死绑定检测 109 在册零新增。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| AP1-06 | 大纲树递归 6 站点 → `findOutlineNode(nodes, pred)` 一份（分支语义谓词显式化；深层递归 3 钉） |
| AP2-02 | 书源任务模板三份 → `runBookJob(spec)`（panic 防线/注销/节流/载荷四类钉子；failedCount 死代码顺删） |
| AP5-08 | 配置快照兜底 5 处（审计 4+同型 2-持锁写回 1）→ `gaeaCfgSnapshotOrLoad()` |
| AP7-02 | 生成链两份 ~100 行循环 → `runImageGenLoop(spec)`（能力开关 comfyRetries/wantKind；语义差异 8 项保留清单） |
| AP4-11 | MPP 三轨 switch+裸字面量 → 4 张 mppLayout 表+具名常量（+198/−92） |
| AP3-02+IN3-06 | docx↔pptx zip 读写+纯函数两套 → `internal/office/ooxml`（方言参数钉死；各包 −200 行；xlsx 证伪不接入） |
| AP7-03 | ComfyUI 四份节点表 → profile 表+共享构建器（Warmup 分发等价表钉死） |
| AP7-10 | 超时收割复制收口 `collectResult`（双通道收口经实测拒绝——无重复可收） |
| IN2-14 | 思考预算守护两处 → `clampThinkingBudget`+常量（默认值同源） |
| FE2-03 | 证伪零改动 |
| FE6-05 | 三路围栏样板 → `extractFence()` 单源（渲染 DOM 不变；审计合并方案被 DOM 证据拒绝） |
| FE7-06 | 导出层两副本 → 单源 import（字节钉证零漂移） |
| 测试 | 新增 4 Go 测试文件 + 2 前端测试文件；字节冻结镜像仓 17 探针双证 |

### 3.4 余量（如实登记）

1. AP1-06 审计未列的两同型函数（`findOutlineNodeByNumAny`/`findOutlineNodeByID`）未收敛（超清单，下批顺手）。
2. Media 链补 comfyui 重试（行为新增，申报未做）；`MarkdownContent.tsx` 第四份围栏样板（白名单外）；xlsx 不接 ooxml（证伪留池）；FE6-05 的 DOM 统一属产品级视觉决策（拍板候选）。
3. MPP9 真实样本门控弱判据（只断言 >0 类）——补强属测试增强，留池。
4. 对账余量：duplication 收敛 ~15 条、god-file 大拆 ~27 条（每条独立刀）、coupling 架构 ~20 条（拍板）、语义冲突 ~5 条（下批优先预核：IN1-04 gate vs review 门槛相反、AP3-03 证据链空间红线恒 work）。

## 四、审计原文纠错（本批 7 条）

| # | 审计原文 | 实测 | 处置 |
|---|---|---|---|
| 1 | AP1-06「5 份」 | 站点 6（两处逐行等价被并算） | 按实收敛 |
| 2 | AP5-08「兜底行为三处各不相同」 | 三处形状一致，差异在下游；实际同型 5 处 | 收敛 5 处 |
| 3 | AP7-02「只有 internal 链有内存压力预检」 | 已失效——两链都有 | 差异清单更正；预检对齐已自然成立 |
| 4 | AP3-02「xlsxedit Recalc 再叠一层」同构 | xlsx 全程 excelize，无自有 zip 层——证伪 | 不接入，包注释记因 |
| 5 | AP7-10「两通道并行驱动」可收敛 | 两通道无重复逻辑（不同数据源不同载荷） | 收窄为 collectResult |
| 6 | FE6-05「仅差一层 data-genui-host div」 | 三路渲染分支均不同构（样式/解包语义/管线） | 只统一样板；合并方案被 DOM 证据拒绝 |
| 7 | FE2-03 死函数 | 现状有正确调用点——证伪（已被后续清理） | 零改动登记 |
