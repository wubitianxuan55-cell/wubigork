# 全仓审计第 22 批 · 零散收尾 + 大拆试水（3 线 + 主代理直做）· 2026-10-03

> 接续 [round-23（批次二十一）](round-23-p1-batch21.md)。duplication 余量零散（FE2-06、章节号改调）+ god-func 大拆试水（AP2-05 表驱动化）+ 死码删除（GA4-08）+ 口径对齐（AP9-06 可执行部分）。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `8dc9f021`（批次二十一）；工作树干净。

---

## 〇、线切分与主代理预核

| 线 | 条目 | 足迹 |
|---|---|---|
| 线 1 | AP2-05 SaveConfig 220 行巨型 switch → 「key→setter」表驱动（god-func 大拆试水第一刀）+ AP9-06 可执行部分（default 分支与 moduleOfIntent 对齐；MainBrainChat 已在批 18 死绑定清册 109 名，删除归拍板不动） | internal/app |
| 线 2 | FE2-06 价格源两份 → PriceSourceCard + usePriceSources()（Repository 失败静默空列表→loadFailed 可见化=申报行为变化） | frontend memoryhub |
| 线 3 | GA4-08 SpawnPolicy 死族删除（生产路径零调用，boot lookupSubagentTemplatePrefix 承担） | internal/gaea/cache |
| 主代理 | 章节号正则改调单源收尾：internal/stats（stats.go:35 + dashboard.go:70/93）+ internal/graph（consistency.go:90）→ project.ParseChapterFileName；app 侧 deepChapterRe（consistency_deep_handler.go:378）押后至线 1 交回（同包不并行） | internal/stats + internal/graph |

## 一、逐线落地

### 1.0 主代理直做 · 章节号正则改调收尾

- `internal/stats/stats.go`：本地正则 MatchString → ParseChapterFileName ok 判定（等价性由批 21 正则等价钉背书）；regexp import 移除。
- `internal/stats/dashboard.go`：chapterFileRe+FindStringSubmatch+strconv.Atoi → 一次取 num/branch；var 与两个 import 清理。
- `internal/graph/consistency.go:90/119`：chapterFileRe 同改（graph 本就 import project 无环）。
- `internal/app/consistency_deep_handler.go:378` deepChapterRe：押后至线 1 交回后完成改调（deepChapterRef 结构保留，包内私有类型注释如旧）。
- 验证：stats/graph/app build+test 绿。**对账收口**：章节号解析六处清点全部单源化或显式保留（保留者口径对照表在 ParseChapterFileName 注释）。

### 1.1 线 1 · AP2-05 表驱动化 + AP9-06 口径对齐（子代理 + 主代理复核）

**AP2-05 复核成立**（SaveConfig 224 行，switch 占 :129-343，63 个 case+1 default，20 处 slog.Warn；三遍模式实证 int×5/float×4/bool×10/字符串直赋 42/SetModelMem 1/NovelsDir 特判 1）。**表驱动化**：`SaveConfig` 收窄约 35 行；`cfgMemSetters` 62 键（签名即审计建议）+ 4 构造器 `setStr/setInt/setFloat/setBool`（样板三遍→单点）+ `memParseSkipError` 哨兵（三条 msg 与原文案逐字一致）；与 config 包 `saveSetters` 成对（那边落盘校验、这边内存同步，**两表键集刻意不一致**——盘有内存无的 18 键走 default Warn，现状钉死）；迁移落点=shelf.go 内自包含域（域头注写明后续整块迁移方案），novels_dir 永久特判。**等价证明**：63 键覆盖对照测试先对旧 switch 跑绿 → 表驱动化 → 测试零编辑再跑全绿。

**AP9-06 冲突指控证伪**：现树 `module_bindings.go:28` 为 `"gaea.chat": "gaea"`（自 f5ddf623 诞生即此），**不存在审计所称 gaea.chat→whisper**——审计把 :26 的 `"chat.chat": "whisper"` 看串行；且批 6 已证伪过同源误读（回归锁现存且绿）。**落地**：映射无需改动；三处注释互引（分类器=意图空间唯一硬编码生产者等）+ 机器守卫升级 `ModuleRegistry.IntentsOf` 访问器 + `TestMainBrainClassifierBranchesDispatchable`（分类器全部 5 分支输出必须可派发，把批 6 只锁 default 的覆盖补满）。MainBrainChat 本体零改动（等 109 拍板）。

**键覆盖对照测试**（saveconfig_keys_test.go 新增 217 行）：63 同步键全覆盖（互异值逐键断言精确内存字段）+ 18 盘有内存无键逐一断言 Warn 文案逐字+内存不动 + 4 非法值钉「盘校验先于内存同步」顺序。

**主代理复核与变异**：app 包独立复跑（含 116s 全量 by 线 1 + 主代理定向）；亲手重做变异「KeyQualityThreshold setter 接错字段→QualityMaxRetries」→ `TestSaveConfig_AllSyncedKeys` 红（`SaveConfig("quality_threshold","7") 后内存未按现状同步`）→ 还原绿。

### 1.2 线 2 · FE2-06 价格源统一（子代理 + 主代理复核）

**现场复核成立，漂移清单 10 项**（比审计多记数条：freqLabel 形态/timeText useMemo/启用态徽标策略/地区徽标/Coins 图标/时间前缀/抓取按钮/按钮位置尺寸/hover 显现 vs 常显/删除失败提示通道——无证伪项）。**抽取**：`PriceSourceCard`（props `variant: "panel"|"repository"` 分支参数化呈现差异；复制/外链/剪贴板 toast 收单源；onFetch 缺省=只读开关）+ `usePriceSources(opts)`（8 秒 Promise.race 兜底+真失败 loadFailed 置位；withFetches 开关）；两容器只剩工具栏/标题/空态+容器态选择；公开组件签名不变（CostLibraryPage 零改动）。**申报行为变化（已做）**：Repository 失败从静默空列表变 loadFailed 可见+重试（同 Panel 三态、同 data-testid）——失败可见化方向。

**主代理复核与变异**：tsc 0/定向 14 例独立复跑；亲手重做变异「withTimeout 把拒绝吞成兜底」→ `bridge 拒绝 → loadFailed=true` 红 + Repository 失败可见化用例红（超时用例仍绿证突变精确）→ 还原 5/5 绿。

### 1.3 线 3 · GA4-08 SpawnPolicy 删仓（子代理 + 主代理复核）

**删仓清单**：全族 10 符号（ForkMode+三常量/SpawnPolicy/SpawnDomainEntry/ForkConfig/SpawnReport/NewSpawnPolicy/BuildSpawnPrompt/buildSpawnDomain/hashDomain/recordHit）删除，spawn.go 226→101 行；生产调用点清零=编译级证明（删除后 `go build ./...` 绿）。**maxForks 配额裁决：删光不保留**——配额从未有执行力（maxForks 只写不读、forkCount 不对照、SpawnReport 无生产者），生产 fork 路径走 skillRunner/RunParallel 与 cache 包无关，「skillRunner 显式限流」无对象可保。**绑定面零变更**（cache 类型不在 App 门面扫描面；bindings-drift OK@744+109 在册实测）。

**偏差申报（合理）**：任务书指定的 boot 侧反向变异不可行（boot.go 在足迹外且**零个 boot 测试触达该路径**——覆盖缺口如实记录未越权补测）；等效改为 cache 侧 `LookupSpawnTemplate` 打 TEMP-MUTATE 恒 not-found → `TestRegisterAndLookupSpawnTemplate` 红 → 还原绿。**主代理亲手重做同款变异**（恒返回空+跳过原逻辑）→ 同钉红 → 还原复绿。

## 二、跨线裁决与缝合

| # | 项 | 裁决 / 处置 |
|---|---|---|
| 1 | **足迹互斥**（app / frontend memoryhub / cache / 主代理 stats+graph+app 押后） | **有效**：AP3-03 式押后惯例延续（deepChapterRe 押至线 1 交回）；主代理 stats/graph 与三线零冲突 |
| 2 | **AP9-06 冲突指控证伪**（审计把 chat.chat 行看串行）+ 批 6 同源误读先例 | 「审计因果链要按现场复核」第三次撞同族——判读配置映射表必须逐行看键，不能凭语义联想 |
| 3 | **语义题刀深四结局延续**：AP2-05 表驱动（等价证明）/AP9-06 判定证伪+守卫升级（不动本体）/FE2-06 单源+申报行为变化/GA4-08 删光（配额无执行力证据先行） | god-func 大拆试水结论：**对照测试先行的表驱动化是低风险形态**（AP2-05 零行为差异一次过），可作后续大拆模板 |
| 4 | **五组变异主代理亲手重做**：key 接错字段/fail-fast 短路（线 2 已由主代理做超时吞拒绝）/LookupSpawnTemplate 恒空/守卫拍除/EXP_COLS 键序——全红→绿 | 延续双轨纪律 |
| 5 | **线 3 发现 boot 模板路径零端到端用例**（覆盖缺口登记未越权补测） | 死码删除的反向变异顺带测绘了测试覆盖边界——缺口进留池 |

## 三、本批结论与量化收益

### 3.1 门禁与守卫（主代理亲跑，前台终值）

**本地快闸（`scripts/ci.ps1` 前台单次，CI OK / exit 0）**：version drift OK（4.454.0）· `go build ./...` 0 · `go vet ./...` 0 · **golangci v2.14.0 0 issues** · **`go test ./... -count=1` exit 0**（app 全量两次 116s/112s）· 前端 lint 0 error / build ok · **vitest 435 文件 3762 例全绿**（批 21 为 433/3752，+2 文件 +10 例：PriceSourceCard/usePriceSources）· E 系列 OK · 卫生四查 OK。

**守卫**：`check-primitives --strict` 无新增 · **`check-test-ctors` 364→372（+8 已评审刷新）**——新增全为本批测试桩：我的 AP3-03 两钉（SpaceSingleSourced 两子测试 bare，同文件 EvidenceChain 既有 bare 先例同款）/线 1 守卫测试 bare/线 3 对照测试 bare/线 4 OCR 链 core-only×2（engineMgr 注入，matrixApp 同款降档形态）/另 2 处同类；放大器计数 4 不变 · `check-contract-drift` 无新增 · `check-bindings-drift` OK@744 + 死绑定检测 109 在册零新增。

### 3.3 本批量化收益

| 项 | 变化 |
|---|---|
| AP2-05 | SaveConfig 224 行巨型 switch（63 case/20 Warn/三遍样板）→ 表驱动 `cfgMemSetters` 62 键 + 4 构造器 + 哨兵（SaveConfig 收窄 ~35 行；63 键对照测试等价证明；盘有内存无 18 键现状钉死）——**god-func 大拆试水首刀零行为差异一次过** |
| AP9-06 | 冲突指控证伪；三处互引注释 + `IntentsOf` 访问器 + 分类器全 5 分支派发守卫（批 6 只锁 default 的覆盖补满）；MainBrainChat 等拍板 |
| FE2-06 | 价格源两份 → `PriceSourceCard`（variant 参数化）+ `usePriceSources`（超时/三态单源）；净 −200 行；Repository 失败可见化（申报） |
| GA4-08 | SpawnPolicy 死族 10 符号删除（spawn.go 226→101；配额无执行力证据先行；绑定面零变更） |
| 章节号收尾 | stats×2/graph×1/app deep×1 四处正则改调单源——**六处清点全部闭环**（保留者口径对照表在 ParseChapterFileName 注释） |
| 测试 | 新增 Go 测试 3 文件（saveconfig_keys 217 行/telegraph 已计批 21/…）+ 前端 2 文件（price-source 10 例）+ 守卫测试升级 |

### 3.4 余量（如实登记）

1. SaveConfig 内存同步表迁移（shelf.go 自包含域 → settings_mem_sync.go 或 config 包）=后续批次整块剪贴（域头注已写方案）。
2. boot 模板路径零端到端用例（线 3 覆盖缺口登记）。
3. `:54` 暖启判据补测试（批 20 登记，未做）。
4. FE5-03 畸形载荷 "undefined" 显示（审计明示接受项，批 21 申报）。
5. MainBrainChat/109 绑定删除=拍板池。

## 四、对账地图更新

- **duplication 簇全毕**（FE2-06 本批收官）。**剩余=god-file/god-func 大拆 ~26（AP2-05 本批吃 1）+ coupling ~20（拍板）+ 零散死码（AP9-06 本体等拍板）**。
- 下一批候选：god-func 大拆第二刀（IN3-04 applyOne 430 行 switch——同 AP2-05 表驱动配方）或 AP2-01 runSinStream 265 行串六职责；coupling 簇与删除类继续等拍板。
