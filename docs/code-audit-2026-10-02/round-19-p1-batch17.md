# 全仓审计第 17 批 · 簇 C 收尾 + 簇 D 死代码（4 条并行线）· 2026-10-03

> 接续 [round-18（批次十六）](round-18-p1-batch16.md) §4.1 的候选：IN1-05（上批让位 project+app 足迹）、GA3-07（上批撞 builtin 包顺延）、簇 D 死代码四条中的 GA4-09/GA2-03/X1-11。
> 本批**不抬版本**、不动 CHANGELOG（发版仪式归用户）；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `fec47842`（批次十六）；工作树干净。

---

## 〇、主代理开工前预核（2026-10-03，读码实测）

| # | 审计项 | 主代理预核结论 | 关键证据（当前工作树） |
|---|---|---|---|
| 1 | **IN1-05** 两套文风互不相识：Profile 与 Fingerprint | **成立（本批只做路径与兼容分支单源，不合两套口径）**：`internal/style/profile.go:134-161` SaveProfile 手拼 `.gaea` 路径+MkdirAll+WriteFile，LoadProfile 自带 `.wubigork` 旧品牌回退（:149-152）= 第三份品牌兼容逻辑；消费方 `internal/app/platform_handler.go:168/:192/:214` + `internal/novelcontext/novelcontext.go:301`（novelcontext 是独立包）。Profile=生成注入口径、Fingerprint=评分口径是刻意分层，只文档化边界不合并 | 线 1：路径/回退收进 project 层，四处消费方改调 |
| 2 | **GA3-07** 知识库检索双口径合分，权重与阈值硬编码 | **成立，且审计阈值清单部分证伪**：合分点 `search.go:61` `kw + vec*8` 裸常数、淘汰 `:62` `vec<0.1`、minScore `:51` `0.02`、scoreEntry 权重（:93 起）——这些成立；**审计列的「0.65/0.55 相似度提示」证伪**：0.65=knowledgeimport.FindSimilar 查重（app/gaea_knowledge_meta.go:139）、0.55=记忆合并建议（app/gaea_memory_meta.go:25），两个不同域不属检索过滤，不动。`knowledge.Search` 唯一外部调用方=builtin/knowledge_search.go:62 | 线 2：单一组合函数+常量命名+「榜单顺序不变」golden |
| 3 | **GA4-09** SkillLayer.Learner 桩与版本晋升全是空壳 | **成立，删仓**：Profiles 多 Kind 定义 v1/v2/v3（skill.go:63-66 等）但 `resolveVersion` 恒返 1；五个方法 no-op 空壳（:138-147）；`LockVersion` 全仓零生产调用；`context/manager.go:109-112` `ContextManager.RecordOutcome` 转调空壳且**自身也是零生产调用**——Learner V5.0 移除后没删干净的骨架 | 线 3：删桩+删 v2/v3 条目+调用点清零证明 |
| 4 | **GA2-03** CompactDescriptor 无条件生效，全量 Schema 死资产且已漂移 | **成立；审计的两个修法选项均不可行，走第三路**：FilteredSchemas（tool.go:422-431）对实现方无条件取 compact；(a)「按 cfg.Tools.Compact 切换」=引入无人要的开关，(b)「删全量 Schema」与 Tool 接口约束冲突（Schema() 是接口方法）。实现方：agent/task.go、boot/skilluse.go、plugin/plugin.go。**主代理裁决走 (c)**：compact-恒生效 是已发布行为保留+注释诚实化（cfg.Tools.Compact 只管 Hide 与 compact 无关）+漂移钉（实现方 CompactSchema 字段集 ⊆ 全量 Schema 字段集的机器可检测试） | 线 3：先实测三处是否已漂移，逐字段差异申报 |
| 5 | **X1-11** 前端 mock 是第三份手写契约，已实际分叉咬人 | **成立（审计数字过时）**：mock 现 17 文件/5335 行（审计说 23/6771，板块删除后已缩）仍是手写同源。全量生成不现实（5335 行行为面），走审计给的「至少加签名逐参数比对契约测试」路线：gen_bindings 新增签名清单模式（Go 侧全集为权威）+ vitest 逐方法 arity 比对 | 线 4：`-signatures-ts` 模式 + 契约测试 + 抓现有错位 |

**线切分（Go 包互斥）**：线 1 = `internal/style`+`internal/project`+`internal/app/platform_handler.go`+`internal/novelcontext`；线 2 = `internal/gaea/knowledge`（builtin 至多注释）；线 3 = `internal/gaea/cache`+`internal/gaea/context`+`internal/gaea/tool`（tool.go，非 builtin）+agent/boot/plugin 的 CompactDescriptor 实现方；线 4 = `frontend/src/gaea/lib/mock/`+`scripts/gen_bindings`。前端仅线 4。四线足迹不相交。

## 一、逐线落地

### 1.1 线 1 · IN1-05 Profile 路径收进 Manager（子代理 + 主代理复核）

**依赖方向分析否决审计原方案**：`style → project` import 已存在（profile.go:12），project 反向持有 `*style.Profile` 即成环 ⇒ 审计 fix 的「Manager StyleProfilePath/Read/Write 三方法（带 style 类型）」不可行。**落地方案 ③**：project 收口路径与品牌兼容（`StyleProfilePath`/`ReadStyleProfileFile`/`WriteStyleProfileFile`，走 `[]byte`），JSON 编解码留在 style 薄壳；签名 `projectDir string → *project.Manager`（4 个消费方手里都有 pm）。**附带收益**：`legacyBrandDir` 单源后与 `IsV4()` 的旧品牌判断**同源共用**（project.go:304/:771）——品牌目录名全仓一份。

**行为逐字段不变**：落点 `.gaea/style-profile.json`、JSON 两空格缩进、权限 0755/0644、`.wubigork` 回退仅 `os.IsNotExist` 放行、错误不包装、写侧刻意非原子（注释写明「勿顺手升级，评分口径 writeJSON 才是原子语义」）。边界互引 doc 两侧对偶（Profile=生成注入 / Fingerprint=评分，勿顺手合并）。

**主代理复核与变异**：diff 亲读；亲手重做变异「删 `.wubigork` 回退分支」→ `TestLoadProfileFallsBackToLegacyBrandDir` 红（`旧品牌目录（.wubigork）应可读`）→ 还原复绿。style/project/novelcontext 三包独立复跑 ok。

### 1.2 线 2 · GA3-07 检索合分单源 + golden（子代理 + 主代理复核）

**常量清点**（search.go 内联 → 命名）：`vecScale=8`（合分 `kw + vec*8`）、`irrelevantVecCutoff=0.1`（淘汰线）、`indexMinScore=0.02`（索引最低相似度，注释写明刻意低于 gaea/search 默认 0.05 的召回放宽语义）、`maxSearchResults=20`、scoreEntry 权重 `10/5/3/3/1/1`。**审计核实纠错**：scoreEntry 是 category+3 **且 phase 另计+3**（审计只写 category）；`combineScore`/`isIrrelevant` 为包内唯一合分口径（grep 证实包内无第二打分路径）。

**golden（敏感性设计语料）**：9 条固定语料 × 4 查询行——`v-body-strongvec`(kw=1,vec=0.594) 刻意压过 `t-tag-only`(kw=5,vec=0)，**vecScale 8→4 即换位**；`b-vec-only`(vec=0.137) 卡在淘汰线上，**cutoff 0.1→0.3 即被踢**；孪生条目同文异名测并列稳定序；另有精确数值钉（`combineScore(0,1)=8`）与边界钉（vec=0.1 恰好不淘汰）。

**主代理复核与变异**：diff 亲读；亲手重做变异「vecScale 8→4」→ golden 榜单换位红 + 数值钉红 → 还原复绿（`ok 1.706s`）。0.65/0.55 证伪与预核一致（查重域/记忆合并域，未触碰）。

### 1.3 线 3 · GA4-09 桩删仓 + GA2-03 漂移钉（子代理 + 主代理复核）

**GA4-09 删仓**（每符号删前全量 grep 清零证明）：`LockVersion`/`IsLocked`+`lockedVer/lockedKind` 字段/`resolveVersion`/四个 no-op（PromoteVersion/RecordOutcome/RecordOutcomeWithReason/DemoteVersion）/`ContextManager.RecordOutcome`/Profiles 全部 v2/v3 条目；**连带发现孤儿 `subagentTools`**（cache/domain.go:56，唯一消费者是被删的 v2/v3）。保留 `CurrentVersion`/`version`（活跃消费 SetL3Version）；`RuntimeLayer.IsLocked` 是另一符号未动。Route 简化为 `const version = 1`。**运行时行为零变化**（v2/v3 本不可达，桩本为 no-op）。

**GA2-03 漂移实测：审计「已漂移」被推翻**——三处足迹内实现方（agent/task 9 字段逐一对应、boot/skilluse 纯转发、plugin remoteTool 结构恒等）+ 足迹外（builtin 全表 45 工具、install_skill、summarize_file、app 包 9 工具）人工逐字段复核**全部零漂移**。交付转为：①tool.go 注释诚实化（compact 恒生效是已发布行为、cfg.Tools.Compact 只管 Hide、全量 Schema 降级为 compact 的手工派生源）；②**漂移钉** `TestCompactSchemaIsSubsetOfFullSchema`（外部测试包避导入环；递归断言 properties/required/type/items；≥40 实现方枚举守卫；覆盖 builtin 全表 blank import；不可机械触达方注释豁免）。

**主代理复核与变异**：diff 亲读；亲手重做变异「Route const version 1→2」→ `TestRouteAlwaysSelectsVersionOne` 红（`CurrentVersion = 2, want 1`）→ 还原复绿（cache/context/tool 三包 ok）。

### 1.4 线 4 · X1-11 mock 签名契约测试（子代理 + 主代理复核）

**根因实证（tsc 为何挡不住）**：`makeMockApp()` = `Object.assign` 16 源聚合，**超出 TS 内建重载数**落 `any` ⇒ `: AppBindings` 注解形同虚设——mock 缺 45 个必需成员 tsc 照样 0 error，浏览器 dev 一调即 TypeError。这就是审计「已实际分叉咬人」的机理。

**落地**：①`gen_bindings` 新增 `-signatures-ts`（`method` 结构加 `Argc/Variadic`，产出 `mock/bindingSignatures.ts`：744 名 → `{argc, variadic, params}`，Go 全集口径，防御下限闸 500，**字节稳定 sha1 四次一致 `f755ca6b`**，改后重跑 `-legacy-ts` git diff 空=共享代码零扰动）；②`contract.test.ts` 5 用例：逐方法经 `gaeaToGaea` 映射对 Go 名（与 proxy.ts 路由口径逐字一致）、形参个数从 `fn.toString()` 平衡扫描（**弃用 `fn.length`**——它数不到默认值后的形参，按它断言等于逼人删默认值）；变参 7 名按门面收窄口径放宽。

**契约测试首跑战果**：①**修 34 处形参错位**（只修 mock，实参只收不用，回退语义不变）；②**45 条漏 mock 白名单**（NOT_MOCKED，逐条带理由——小说 AI 创作流 23/轻语 4/子代理事件 5/charlib 2/组价 2/DAG 2/零散 7），带防腐过期锁（被实现或 Go 删除即红）；③豁免 1 条 `SubmitDisplay`（前端桥接双参 vs Go 单参的**既存口径分歧**，处置归桥接线，豁免同样受过期锁约束）。legacy 面 ~188 名以批 16 的 `legacyBindings` 单源豁免（不在测试手抄副本——手抄即第四份手写契约）。

**主代理复核与变异**：tsc 0 error、contract 5/5、签名清单 sha1 与子代理四次一致、`-legacy-ts` 零扰动独立复跑；亲手重做变异「mock Cancel 加形参」→ 契约红（`Cancel（mock 1 参 vs 门面 0 参）`）→ 还原复绿 5/5。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥按 Go 包划**（style+project+app+novelcontext / knowledge / cache+context+tool / 前端 mock+scripts） | **有效**：四线并行期间主代理多次 `go build ./...` exit 0；收线 app 包 128.166s 全绿 |
| 2 | **审计 fix 两处被依赖方向/接口约束否决，主代理定向第三路** | IN1-05 的「Manager 带类型三方法」会成环（style→project 已存在）⇒ []byte 缝；GA2-03 的二选一均不可行 ⇒ 诚实注释+漂移钉。**审计 fix 是方向不是图纸** |
| 3 | **两处审计断言被实测推翻** | GA2-03「已漂移」⇒ 全部实现方零漂移（机器+人工双复核）；GA3-07 的 0.65/0.55 属查重/记忆合并域。**推翻审计也要机器证据**（漂移钉反过来把「不再漂移」变成可检） |
| 4 | **X1-11 根因比审计更深**：不只是「mock 手写同源」，是 `Object.assign` 超重载使类型注解失效——**契约测试是唯一能抓住的层**（tsc 结构性失明） | 线 4 的根因实证进台账；45 条漏 mock 清单即「分叉咬人」的完整实证 |
| 5 | **四组反向变异主代理亲手各重做一组** | 线 1 删回退分支 / 线 2 vecScale 减半 / 线 3 Route 版本改 2 / 线 4 mock 加形参——红→绿独立取得（延续双轨纪律） |
| 6 | **批 16 产物成为批 17 基建** | `legacyBindings.ts`（批 16 线 4）在线 4 的豁免逻辑中单源复用；`-legacy-ts` 的防御闸与字节稳定做法被 `-signatures-ts` 复刻；共享代码零扰动以「改后重跑旧模式 git diff 空」证明 |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台）

**本地快闸（`scripts/ci.ps1` 前台单次，CI OK / exit 0）**：version drift OK（4.454.0 不抬版本）· `.tmp` OK · `go build ./...` 0 · `go vet ./...` 0 · **golangci-lint v2.14.0 0 issues** · **`go test ./... -count=1` exit 0**（app 128.166s）· 前端 lint 0 error / build ok · **vitest 428 文件 3723 例全绿**（批 16 为 427/3716，+1 文件 +7 例：contract.test 5 用例 + 日历/钉子增量）· E 系列 OK · 卫生四查 OK（AGENTS.md 53873B 分流后预算内）。

**过程事故（如实登记）：首轮全量 vitest 红 1 例——批 13 钉的 DataPanel 回滚测试自带竞态**。`findAllByText` 拿到任一匹配即提前返回，满载下即时 toast 晚渲染被漏数成 1 条（期望 ≥2：toast+持久条两通道）；单跑恒绿、在册 flaky 清单为空。**按清单规则「能修的就修掉」修测试竞态而非登记**：分容器断言（toast 限定 `.ant-message` 容器 `waitFor`、持久条限定 `settings-rollback-failure` testid 容器 `within`）——断言意图不变（两通道都带后端原文）、早退竞态消除。修后单跑 7/7 + 全量 CI OK。`scripts/test_image_models.go` 的 gofmt 存量（v1.0.0 起，golangci 不覆盖 scripts/）与本红无关，如实排除。

**守卫（收线复跑）**：`check-primitives --strict` 无新增 · `check-test-ctors` 364 零新增 · `check-contract-drift` 无新增 · `check-bindings-drift` **OK@744**（Go 绑定面零变更；`-signatures-ts` 为新增只读模式不影响门面）。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| IN1-05 | 品牌兼容路径第三份手写分支消灭——`.wubigork` 回退与 `IsV4` 同源 `legacyBrandDir`；Profile 路径单源 project.Manager（[]byte 缝避环）；两套文风口径边界互引 doc |
| GA3-07 | 检索合分 6 类内联常数 → 命名常量 + `combineScore`/`isIrrelevant` 唯一口径；「榜单顺序不变」golden（敏感性设计语料：换位对/边界对/并列孪生） |
| GA4-09 | Learner 残骸删仓：5 no-op + LockVersion/IsLocked/resolveVersion + ContextManager.RecordOutcome + v2/v3 死条目 + 孤儿 subagentTools（skill.go 219→172 行）；两把钉（每 Kind 恰 1 版本 / Route 恒 v1） |
| GA2-03 | 「已漂移」证伪；compact-恒生效诚实注释 + 全实现方递归包含性漂移钉（≥40 实现方守卫） |
| X1-11 | mock 契约从「无」到「机器可检」：`-signatures-ts` 生成 744 签名清单（字节稳定）+ contract.test 5 用例；**修 34 处形参错位**；45 条漏 mock 实证清单+防腐锁；根因实证（Object.assign 超重载 ⇒ 类型注解失效） |
| 测试 | 新增 5 文件（profile_io_test / search_golden_test / skill_test / compact_drift_test / contract.test.ts）+ 1 生成物；删 1 个钉桩用例（manager_test 的 RecordOutcome） |

### 3.4 余量（如实登记，未硬凑）

1. 45 条 NOT_MOCKED（mock 缺实现）——补 mock=行为新增，超出「契约测试」刀的范围，**留池待排**（浏览器 dev 对这些名字仍会 TypeError，但从此有清单有锁）。
2. `SubmitDisplay` 桥接面双参 vs Go `GaeaSend` 单参的既存口径分歧——豁免留痕+过期锁，处置归桥接线（**新登记拍板候选**）。
3. GA2-03 的 cfg.Tools.Compact 命名与 compact schema 语义无关（只管 Hide）——改名会碰配置面，留池。
4. `scripts/test_image_models.go` v1.0.0 起未 gofmt（golangci 配置不覆盖 scripts/，历届 CI 均绿）——存量在册，不在本批足迹。

## 四、留池与待拍板

### 4.1 下一批候选（批 18）

- **簇 D 收尾**：IN4-04（桌面能力 0 生产调用路由——仍等「接线 vs 弃用」拍板，不接线不动）；FE3-03（死绑定检测——`legacyBindings.ts` + 本批 `bindingSignatures.ts` 双全集已备齐，实现只剩「前端驼峰名反查」比对逻辑）。
- **簇 C 残量**：GA6 族组价两面接入 `cost.Enhance`（一行事）；FE4-04 余量（分类语义错误无测试能抓——审计原话，本批未展开）。
- **既有拍板池不变**：桌面 agent 接线三件套、`download_and_install` 独立确认 UI、`GaeaDocumentLint` 读根门、FE6-09 conversationKey、AP4-01 三编排合并、X1-03 裸构造、`estimateTokens` 四口径；**本批新增**：`SubmitDisplay` 桥接/Go 口径分歧。

### 4.2 本批新增留池

见 §3.4（45 条 NOT_MOCKED 补 mock、SubmitDisplay、Compact 命名、test_image_models.go 存量）。

## 五、审计原文纠错与遗留登记

| # | 审计原文 | 本批实测（谁测） | 处置 |
|---|---|---|---|
| 1 | IN1-05 fix「Manager StyleProfilePath/Read/Write 三方法」 | project 反向持有 style 类型即成环（style→project import 已存在）（线 1 读码） | []byte 缝变体落地，边界 doc 达成同一目标 |
| 2 | GA3-07「0.65/0.55 的相似度提示」属检索阈值 | 0.65=导入查重域、0.55=记忆合并建议域，非检索过滤（线 2 + 主代理预核双证） | 不动；审计清单纠错 |
| 3 | GA3-07「category+3」 | scoreEntry 实为 category+3 且 phase 另计+3（线 2 实测） | 常量注释写明 |
| 4 | GA2-03「全量 Schema 已漂移」 | **全部实现方零漂移**（三处足迹内机器实测+足迹外人工逐字段复核）（线 3） | 断言推翻；漂移钉把「不再漂移」变成机器可检 |
| 5 | GA2-03 修法二选一（加开关 / 删全量 Schema） | 均不可行：开关无人要；Schema() 是接口方法删不得（线 3 + 主代理预核） | 第三路：诚实注释+漂移钉 |
| 6 | X1-11「23 文件/6,771 行」 | 现 17 文件/5,335 行（板块删除后缩水）（线 4 实测） | 数字过时，缺陷本体成立 |
| 7 | GA4-09「SkillLayer.Learner 桩」 | Learner 字段不存在（仅注释残留）；真残骸=5 no-op+锁版本链+v2/v3 死条目（线 3 grep） | 按实际残骸删仓 |
