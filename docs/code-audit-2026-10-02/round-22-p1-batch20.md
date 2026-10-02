# 全仓审计第 20 批 · 语义冲突五条（4 线 + 主代理直做 1）· 2026-10-03

> 接续 [round-21（批次十九）](round-21-p1-batch19.md) §〇 的对账地图「语义冲突 ~5 条（潜在真 bug，下批优先预核）」。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `182c672e`（批次十九）；工作树干净。

---

## 〇、主代理预核（2026-10-03，读码实测）

| # | 审计项 | 预核结论 | 关键证据 |
|---|---|---|---|
| 1 | **AP3-03** 证据链空间红线三入口恒等于 work | **成立**：三处守卫读 `gaeaEffectiveSpace()`（xlsx_edit.go:170/pptx_edit.go:85/docx_edit.go:143）但落地写死 `Space: "work"`（:200/:101/:159）——判据与写入值两源 | 主代理直做（space 取一次、守卫与落库同源） |
| 2 | **IN1-04** gate 与 review 判据重复且门槛相反 | 待线 3 现场复核（gate 句级 telegraph vs review 段级 telegraph，粒度差异可能是设计分层也可能是漂移） | 线 3 |
| 3 | **IN2-07** 图片 backendType 双套口径 | 待线 1 现场复核（注册表 kind vs SetImageBackend 自由串 vs 内部判断口径） | 线 1 |
| 4 | **IN3-09** 三种 OCR 口径各自独立客户端 | 待线 4 现场复核（架构题，允许三档判定：接入/口径单源/证伪） | 线 4 |
| 5 | **FE3-04** mock-only 绑定进真类型面 | **成立且格局已部分变化**（批 17 引入 MOCK_ONLY_NAMES）：四名中 `SetSubagentTemperature/SetEffort/SetSubagentModel` **零组件调用者=纯死类型面**；`Compact` 有一处真实调用（controller.ts:798）且 Go 无绑定——真机恒抛可重试假错误 | 线 2 |

## 一、逐线落地

### 1.1 线 1 · IN2-07 图片 backendType 口径（子代理 + 主代理复核）

**现场复核：审计行号全部漂移**——批 10 的 IN2-03 已把 app 层五份手写构造收敛进 `resolveImageBackend`，直传字面量站点已不存在。当前两轨：A 轨注册表 kind（openai/comfyui/glm，构造用）vs B 轨运行时类型名（xai/comfyui/herdsman/ollama/glm——对外标识，前端 ≥7 文件分支+配置落盘+内部 24 处判据）。**关键事实**：herdsman/ollama 同一具体类型、xai 无实例（nil→内置管线），引擎身份只由 B 轨名承载。

**审计 fix 双双被证据否杀，走最小收窄**：(b) 类型断言推导——herdsman/ollama 同类型不可分；(a) 归并 ImageBackendKindOpenAI——①配置文件 `image_backend` 存量值断裂（启动恢复 switch 键为 B 轨名）②前端 7+ 处分支全破（前端禁改）③引擎名无 engineID 承载点。**落地**：B 轨取值域收成 `ai.ImageBackendType*` 五常量 + 常量块内**唯一口径对照表**（B 轨名↔A 轨 kind↔实例构造↔消费方）；24 处判据字面量→常量（字节等值，`TestImageBackendTypeConstants_LiteralConformance` 钉死）；空值回退判据单点 Set/Get 两处。**彻底统一需配置迁移=拍板另立**。

**主代理复核与变异**：diff 亲读；亲手重做变异「:816 comfyui 空模型归位判据→Herdsman」→ `TestImageBackendInfoBackendJudgments/comfyui_空模型归位_krea2` 红（`got "grok-imagine-image-quality"`）→ 还原复绿。**顺带发现**：`:54` 暖启判据同型但无测试覆盖（登记）。

### 1.2 线 2 · FE3-04 mock-only 四名收口（子代理 + 主代理复核）

**现场复核 + 一处新发现**：四名现状与审计一致（类型面/facets/mock 三处齐、Go 无绑定）；**但 `compact` 回调当前零 UI 调用者**（App.tsx 未解构、ChatPane 不用、无测试取）——审计「有真实 UI 调用」的前提已过时（手动压缩入口应已在前批移除）。修法在两种前提下都正确，照走；「compact 回调+CoreBindings.Compact+mock Compact 整体摘除」入拍板池。

**落地**：①三名零调用者（SetSubagentTemperature/SetEffort/SetSubagentModel）从类型面/facets/mock 三处整体摘除（MOCK_ONLY_NAMES 4→1，理由注释更新；contract.test 经单源自动仍绿零改动）；②Compact 走 realApp get 陷阱 fail-fast：MOCK_ONLY 名返回显式拒绝函数（`BridgeError code=MockOnlyBinding`，文案含「mock-only」「真机不可用（上下文压缩由后端会话事件自动执行）」）；③controller.ts:798 文案诚实化（专用 catch + notice，**去掉「请重试」**——重试永不成功的假提示消灭）；dev mock 路径行为不变。

**主代理复核与变异**：tsc 0 error（摘除态两次跑=零调用者反证）、proxy.mockonly 4/4 + contract 5/5 独立复跑；亲手重做变异「fail-fast 行 `if (false &&` 短路」→ `真机路径调 Compact → BridgeError(MockOnlyBinding)` 红 → 还原 4/4 绿。

### 1.3 线 3 · IN1-04 gate/review 判据口径（子代理 + 主代理复核）

**判定：刻意分层，非漂移**。证据链：上游 rubric 本就是两个维度行（句长节奏=句级 FAIL 语义 / 格式可读性=段级）；资产镜像 `.gaea/skills/novel-review/rubrics/generic.json` 的 `sentence_rhythm`（measurable，不在 deterministicEngine）vs `format_readability`（engine=true）——句级资产上登记、引擎侧由 gate 承担；git 史两包同日并行落地互非祖先。**审计「判据重复且门槛相反」的预设修正为「同域不同粒度的刻意分层」**；但其风险指认为真：此前代码零互引、两侧都用「电报体」措辞，后人极易顺手对齐。

**落地（无争议层 + 最低档）**：①新包 `internal/noveltext`（79 行纯函数零依赖）：`SplitParagraphs`（单源 + **新增 Line 字段**——旧 gate 数所有行、review 只数非空段，同段两边编号不同且无人写明，现 Idx/Line 双口径一次切分）+ `SplitSentences`（gate 原实现 verbatim）；②双侧互引注释写明「同域不同粒度、门槛独立、勿对齐」（gate 包注释+判据处、review 文件头+常量+判据处）；③review telegraph 阈值魔法数→命名常量（同值 12/12 行为不变）；④**矛盾样本双向钉死**：方向① 四大段全短句墙→gate 报 telegraph_style/S2 而 review APPROVE；方向② 20 段长句碎段→review warn 点名电报体而 gate 零发现。**rubric 接入=留池申报**（三理由：rubric.json 有用户整体替换机制会改 gate 行为违反冻结 / gate 无平台轴结构不兼容 / 缺省回退破坏 fail-closed）。**修正审计一处预设**：splitSentences 并非两包各一份（只有 gate 有）；实际重复是段落切分；app 另有自己的 splitSentences/Paragraphs 但语义不同非重复。

**主代理复核与变异**：三包独立复跑 ok；亲手重做变异「SplitSentences 分隔类加逗号」→ `TestChapterQualityIssues_EmptyAndClean` 红（正常文本误报 telegraph）+ `TestIN104_ReviewParagraphTelegraphGateClean` 红 → 还原复绿。

### 1.4 线 4 · IN3-09 OCR 三口径（子代理 + 主代理复核）

**档位判定：证伪审计修法（接入+删串联），按档 2 落口径单源**。理由：①docmd seam 有三组「本地、文件不出机」消费方（扫描件 PDF/报价单图片/腿4），herdsman 注册进 auto 链=三组流量被远端接管（行为+隐私面变更），不进=死 kind；②装不进 provider 形状（docmd 无参 env 驱动 vs herdsman 的 engineMgr+config.json+HERDSMAN_*，且 MinerU 是文档解析非单图 OCR、探测模型不同构）；③「删四路串联」=删产品语义（腿1 是模型中心「设为 OCR」的兑现路径，腿序+空文本穿透+逐腿 errors.Join 是 IN3-08 刚修的行为）。**审计「三层/四路」形态属实但第三层是编排非第三客户端**（gaea_ocr.go 自身零 HTTP 代码）。

**落地（纯注释零删除）**：docmd seam 注释新增权威「口径边界」段（kind 取值域=本地引擎、三消费方、不许接入完整理由、**四组同名不同物配置对照表**——`GAEA_OCR_MODEL`=gguf 文件路径 vs herdsman 模型 ID 等）；herdsman 包注释分工边界；gaea_ocr.go 四腿图（每腿配置源）+「编排非第三客户端、勿折进 seam」互锁。**新钉 6 个**（kind 取值域冻结 / 绑定腿1 优先+模型名分流 / env 不劫持远端腿 / PaddleOCR→MinerU 腿内顺序 / 腿4 透传 fail-closed，全部 httptest 假服务）。

**主代理复核与变异**：三文件 diff 确认纯注释；亲手重做变异「docmd 注册 herdsman kind（审计第一步）」→ `TestOCRKindDomain_LocalOnly` 红（`kind "herdsman" 属远端链，不得注册进本地 seam`）+ `TestOCREngineOrder_RejectsRemoteKinds` 红 → 还原复绿。

### 1.5 AP3-03 · 证据链空间单源（主代理直做）

**落地**：新单点 `appendOfficeEvidence(tool, target, before, after, baseline, opsJSON)`（gaea_docx_edit.go）——空间守卫与 `ChangeRecord.Space` **同取 `gaeaEffectiveSpace()` 一处**；三入口（appendXlsx/Pptx/DocxEvidence）变薄壳（差异=载荷构建与 xlsx 的 opsJSON，作参数传）；红线不变（非 work 不落账）、journal 不可用静默不变、`Space: "work"` 写死消灭。**钉子两方向**（`TestGaeaDocxApplyEdit_SpaceSingleSourced`）：play 空间零证据卡（红线）/ work 记录 Space==gaeaEffectiveSpace()（同源断言）。

**主代理反向变异**：拍掉守卫（space 落库但不再拒）→ 方向 1 红（`play 空间证据卡数 = 1, want 0`）→ 还原复绿。顺带修正：gaea_pptx_edit.go 的 evidence import 随收敛移除。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **五条语义项四种结局**：单源（AP3-03）/最小收窄+拍板（IN2-07）/分层判定+留池（IN1-04）/证伪修法+口径单源（IN3-09）/摘除+fail-fast（FE3-04） | **语义/口径题的正确粒度=先判定性质再定刀深**——五条无一照搬审计 fix 原文，但五条审计的风险指认全部成立 |
| 2 | **审计前提过时两处**（FE3-04 的「有真实 UI 调用」、IN2-07 的行号站点） | 批 10 IN2-03 与前批 UI 移除改变了现场——审计快照语义仍成立（compact 真机假错误仍在），修法随现场走 |
| 3 | **「归并常量」双杀证据**（IN2-07：配置存量值+前端 7 分支） | 彻底统一=配置迁移+前端联动=拍板另立；最小收窄（常量化+对照表）先行，两轨口径对照表唯一落点 image_backend.go |
| 4 | **IN1-04 的「分层 or 漂移」判定法**：上游资产（rubric 维度行+generic.json 的 measurable/engine 标记）+ git 史（同日并行落地）双证 | 判定为刻意分层后，交付转为互引注释+矛盾样本双向钉死——「矛盾可见」比「强行一致」诚实 |
| 5 | **四组变异主代理亲手重做**（预算减半/2013 偏移… 不对——本批：comfyui 归位判据/fail-fast 短路/切句分隔符/herdsman kind 注册/守卫拍除，五组） | 五线五组全红→绿独立取得（线 1 额外发现 ：54 暖启判据无测试覆盖，登记） |
| 6 | **同包不并行纪律的代价与守约**：AP3-03 押后至线 1 交回 | internal/app 唯一占用期间主代理不动该包；押后成本≈0（三行级改动） |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台终值）

**本地快闸（`scripts/ci.ps1` 前台单次，CI OK / exit 0）**：version drift OK（4.454.0）· `go build ./...` 0 · `go vet ./...` 0 · **golangci v2.14.0 0 issues** · **`go test ./... -count=1` exit 0**（app 112s/ai 7.4s/docmd 9.4s 等）· 前端 lint 0 error / build ok · **vitest 431 文件 3735 例全绿**（批 19 为 430/3731，+1 文件 +4 例：proxy.mockonly 4）· E 系列 OK · 卫生四查 OK。

**守卫**：`check-primitives --strict` 无新增 · `check-test-ctors` 364 零新增 · `check-contract-drift` 无新增 · `check-bindings-drift` OK@744（**死绑定检测口径随 FE3-04 收缩：MOCK_ONLY_NAMES 4→1，facets 565→562**——三名死成员摘除后认领集-3，legacy 清单 184→187，守卫自动跟随再生口径，零人工干预）。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| AP3-03 | 三入口「守卫读空间、落库写死 work」两源并存 → `appendOfficeEvidence` 单点同源；play 红线钉 + 同源断言钉 |
| IN2-07 | 24 处判据字面量 → `ImageBackendType*` 五常量 + 全仓唯一口径对照表；两轨口径「人肉对齐」→ 机器钉死；彻底统一留拍板 |
| IN1-04 | 判定=刻意分层（非漂移）；新包 `internal/noveltext`（段落双口径 Idx/Line 单源——两边行号不同从无人写明）；互引注释防顺手对齐；**矛盾样本双向钉死**；rubric 接入留池（三理由） |
| IN3-09 | 审计修法证伪；口径边界文档单源（**四组同名不同物配置对照表**）+ 互锁注释 + 6 钉（kind 域冻结/四腿编排语义） |
| FE3-04 | 三名死类型面成员摘除（零调用者 tsc 反证）；Compact 真机 fail-fast 诚实化（MockOnlyBinding，去「请重试」假提示）；MOCK_ONLY_NAMES 4→1 |
| 测试 | 新增 5 Go 测试文件 + 1 前端测试文件；线 2 既有契约/绑定测试经单源零改动 |

### 3.4 余量（如实登记）

1. **IN2-07 彻底统一**（herdsman/ollama/xai 并入注册表 kind）：需配置迁移+前端 7+ 文件联动——**拍板另立**。
2. **FE3-04 拍板候选**：compact 回调+CoreBindings.Compact+mock Compact 整体摘除（当前零 UI 调用者）vs Go 侧补 GaeaCompact 绑定。
3. IN1-04 rubric 接入（gate 判据进数据资产）：需覆盖文件兼容方案+app 接线，单独立刀。
4. `:54` 暖启判据无测试覆盖（线 1 发现）；IN3-09 若未来出现第三条 OCR 链，seam 接入决策重开。

## 四、审计原文纠错（本批 6 条）

| # | 审计原文 | 实测 | 处置 |
|---|---|---|---|
| 1 | IN2-07 行号站点（image_handler.go:863/873 等） | 全部漂移——批 10 IN2-03 已收敛构造站点 | 按当前现场清点 |
| 2 | IN2-07 fix (a)/(b) | 双双被证据否杀（配置存量值+前端分支/同类型不可分） | 最小收窄 + 拍板登记 |
| 3 | IN1-04「判据重复且门槛相反」预设 | 刻意分层非漂移（上游资产两维度行+git 史双证）；splitSentences 并非两包各一份 | 互引注释+矛盾样本钉死；rubric 留池 |
| 4 | IN3-09 fix「herdsman 注册进 provider seam、删四路串联」 | 证伪——三组本地消费方隐私边界+装不进 provider 形状+串联=产品语义 | 档 2 口径单源 + 6 钉 |
| 5 | FE3-04「真机 controller.ts:798 恒抛」 | 现象仍在但「有真实 UI 调用」前提过时（compact 回调零 UI 调用者） | 摘除+fail-fast；整体摘除入拍板 |
| 6 | FE3-04 路径 `markdownFence` 式的 drift.ts:37-41 | 行号漂移；批 17 MOCK_ONLY_NAMES 已存在，本批是它的收缩与 fail-fast 落地 | 按当前现场落刀 |
