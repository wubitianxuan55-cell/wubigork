# AI 反馈批（2026-11-27，20 条）施工记录与接续单

> 事实源。下个会话从这里接手，不要重新分诊。
> 来源：`C:\Users\wubi\Desktop\ai-feedback-2026-11-27.md`（模型经 ai_feedback 工具提交的 19 条 + 用户手写 1 条）。
> 用户中途补充：①附《三级树表测算模版-结构说明与适配规范》v5.6（`D:\新工作\2630沿江一公里\沿江包1\.gaea\exports\`），令「根据现在的实际情况进行修正，补充进说明书，它已经是昨天的版本」；②上下文不够，令「做好文件记录，我们下个会话继续」。

## 0. 状态一句话

- **全部收口（2026-11-28）**：首会话落码 11 条 → 五笔代码 `a78c83a7` / `1b628dc7` / `0ae43cb3` / `73739a34` / `093c83c5`（§9）；余量七条 → 五笔 `5ddd8408`（方案库 #5' / #11 / #12）/ `13642743`（#8）/ `06780c87`（#4）/ `a98861c3`（#15②）/ `f35fb3f4`（#14，见 §10）。现行包 `releases/gaea-ai-feedback-20261128-f35fb3f4`，装机 **PID 35548**，包版四件已回写主树 `build/bin`。**未 push**（本地 ahead origin）。
- 拍板落定（2026-11-28）：**277 条章节数据恢复、测算样表三处旧文案、样表「按数据扩骨架」填表脚本——三项不做**；拍板池只剩 #10 矢量图补图（要引 PDF 库依赖）。`xlsx_apply` 的 eval 类操作判不做（等于绕开发布边界）。
- 下文 §1~§8 是首会话（2026-11-27 深夜）的原始记录，保留不改；收口增量在 §9、§10。

## 1. 分诊结论（逐条对码核实）

| # | 条目 | 结论 | 处置 |
|---|---|---|---|
| bug-1 | session_search 刚结束的会话无命中 | 属实。根因：只索引 `assistant_message`，实时落盘的助手成稿 kind 是 `message`；另有多关键词按整句短语匹配、一行坏日志拖垮整个检索 | ✅ 本会话修 |
| bug-2 | bash 长命令被拆成 37 个散键 | 归因错。日志实锤：模型把数据字典 `{"BC01":15506.11,…}` 整个当参数发出（没有 command）；「键序每次不同」是报错里 Go map 的遍历序 | ✅ 本会话修 |
| bug-3 | 详情不回显章节、更新时被清空 | 属实，刀A 已修（0437d1e8）；277 条数据恢复不在代码范围 | ⏸ 拍板：用库备份或原始资料重导 |
| fric-4 | 脚本不能直写正式区 xlsx | 发布边界按扩展名拦、只盯 bash，设计如此 | ❌ 未动（做法见 §4） |
| fric-5 | heredoc 截断 / write 后 edit 被拦 / spill 单行读不全 | 三处都属实，根因各见 §2.2 | ✅ 本会话修 |
| fric-5' | proposal_save「方案库共 N 份」漂移 | 计数是全库行数（含录入中） | ❌ 未动 |
| fric-6 | format_convert 超 500 页静默截断 | 属实 | ✅ 本会话修 |
| miss-7 | xlsx 缓存值丢失 / 转换器结构改写无预警 | 属实：xlsx_apply 只有「环境缺失」一种失败会出备注；recalc 前后没有结构比对 | 🔶 做到一半（§3） |
| miss-8 | 列表缺翻页与精确过滤 | offset 已有（刀A）；地区前缀匹配用户已撤；keyword 精确开关没有 | ❌ 精确开关未动 |
| miss-9 | 批量载荷无法从文件读取 | 属实（复盘 P9 当时挂池，这次用户令修） | ✅ 本会话修 |
| miss-10 | 交叉对账 + 矢量图补图 | 对账=md_validate source（d5efe65e 已落）；矢量图要引 PDF 库依赖 | ⏸ 矢量图挂拍板 |
| miss-10' | md_validate 无分级无 baseline | 已落（d5efe65e） | ✅ 已有 |
| miss-10'' | docx 表格转 MD 还原不足 | 已落（d5efe65e 重建器） | ✅ 已有 |
| miss-11 | outline 标题逐字强校验 | 属实（匹配是子串包含，OCR 错字仍对不上） | ❌ 未动 |
| miss-12 | 扫描件无井号标题时缺标题识别 | 属实 | ❌ 未动 |
| miss-13 | pages 受 500 页上限约束 | 属实：上限卡的是「起始页 ≤ 500」 | ✅ 本会话修 |
| wf-14 | 跨会话交接无质量闸 + 歧义指令无确认 | 属实 | 🔶 「按模板编制先确认口径」已进技能与说明书；交接卡未动 |
| wf-15 | 缺交付前对账 / 断点检查点 / xlsx 体检 | 属实 | 🔶 对账=导入演练的工作簿内对账已做；体检做到一半；检查点未动 |
| wf-16 | 测算模版骨架无标注 | 属实 | 🔶 说明书讲清 + 演练能认溢出；**样表本身不动**（红线） |
| wf-17 | proposal-longform 技能缺两块 | 已落（ca0b13a2 + d5efe65e） | ✅ 已有 |
| wf-18 | （用户手写）造价技能工作流 | 属实：playbook 与身份稿的主路是「建项目→写清单」的库内直写，样表只在交付段出现 | ✅ 本会话修 |

## 2. 已落码（分包测试绿）

### 2.1 造价线（wf-18 / wf-16 / miss-9 / wf-15 的对账）

- **载荷走文件**（`internal/app/office_cost_payload_files.go` 新增）：`import_add` 增 `resources_file / quotas_file / bills_file`，`classify` 增 `classify_file`——脚本从资料直接生成 JSON，程序读文件；数组与文件同给以数组为准；手写数组仍 200 行/批，文件 2000 行/文件；退回行号按文件顺序计。`import_status export=<路径>` 把暂存区导出成同格式 JSON（自校验用）。常驻 schema 预算 9400→9700。
- **工作流重排**（`office_runmode_skills.go` playbook、`office_runmode_systemprompt.go` 身份稿、`office_runmode.go` 模式行、cost_library 描述）：主路改为「对齐口径 → 取样表读说明书 → 查库定价 → 填表 → 两道门 → **交用户检查，同意后入库**」；`cost_project_apply` 降为库内零星维护；写明「样表 2 块 × 4 明细只是示例槽位、数量没有上限」；地区口径统一成只到省（删掉「先加省再加市」的旧话）。
- **说明书**（`internal/gaea/skill/assets/成本测算表-使用说明书.md` 新增，go:embed）：`cost_library action=template` 与板块「导出 → 测算模版」都会把样表和 `<样表名>-使用说明.md` 一起落盘；回执指路先读它；工作区 `.gaea/skills/cost-estimate/成本测算表-使用说明书.md` 同名文件可覆盖。内容 = 用法（总则、编制流程、槽位不够怎么加行）+ v5.6 适配规范按现状修正后并入（六表总览、综合单价 14 列、固定组与紧凑布局、金额链三层 ROUND、公式模板、各表要点、导入边界、核查清单、版式规范）+ 附录 A（样表里的旧文案）+ 附录 B（相对 v5.6 的 12 处修正）。用户工作区里的 v5.6 原件没改。
- **导入演练的两道确定性检查**（`internal/gaea/workcost/workbook_recon.go` 新增；解析器补读缓存值）：①「疑似明细行溢出」——定额行的编码却是本簿工料机价格表里的编码 = 上一块明细溢出盖住了定额头，整块跳过并单列；②「工作簿内对账」——拿工作簿自己的缓存值按金额链逐级复算（定额行 = Σ明细、清单行 = Σ(定额×系数) 与按明细重算、三费拆分、「清单工程量」与清单表引用、直接费合计、工料机空价）。金样夹具（沿江包1）零误报。顺手根修：工料机价格表现在先于综合单价表解析（此前综合单价里的「回退价格表」映射在读取时恒空）。
- **入库确认卡**（`controller_approval.go`）：导入测算工作簿的卡面写「请确认你已检查过这份测算表」。
- 前端：板块导出菜单文案改正（「按库预填」是旧话），去掉后端早已删除的「资料条目」导出项。
- 测试：`office_cost_payload_files_test.go`、`workbook_recon_test.go` 新增；救活两个因旧夹具被删而静默跳过的导入测试（什邡夹具 → 沿江包1，缺夹具即失败）；`TestCostLibraryImportProjectDryRunRecon`、说明书与技能条款断言、vitest 40 例。

### 2.2 bash / 文件闸 / read_file（bug-2 / fric-5）

- **长命令走临时脚本**（`bash.go`）：Windows 上 Git Bash 的 `bash -c` 参数超过约 8.2K 字符被静默截断且 rc=0（探针：8053 字符完整，12043 字符起只剩前 144 行）。超过 6000 字符的命令写临时 .sh 由 bash 执行（60KB 实测完整）；强制沙箱时不改道。
- **缺 command 的报错**（`bash.go`）：键全未知且值都是标量 = 数据对象，直接说「这不是命令，数据写成 JSON 文件再用 bash 执行脚本」；键排序输出。
- **stale 闸路径口径**（`agent/stale_paths.go` 新增，`execute_one.go` / `agent_helpers.go` / `tool_precheck.go`）：闸和版本观察改用工具自己的路径解析（ReadTarget / WritePaths）。此前拿参数里的相对路径按进程 cwd 读，读不到就当锚点失配——write_file 之后紧跟的 edit_file 在正式环境里必被拦（包4 会话日志逐调用回放证实 old_string 与磁盘逐字一致）。预检查同根（读不到文件而整体失效），并改为与 edit_file 工具同口径（空白漂移不拦）；真失配时报错直接给「最接近的位置 + 现行文本」。锚点仍匹配的定点 edit_file 也不再被跨轮版本闸拦。
- **read_file 字节窗口**（`readfile.go`、`compact.go`）：`byte_offset / max_bytes` 按字节严格分页（默认 16K、上限 24K，压在输出层 32KiB 压缩线内），两端对齐字符边界，尾部给续读游标；与 offset/section 互斥。输出层截断标记对单行超长改成说实话并指路（`textutils/budget.go`）。

### 2.3 会话检索（bug-1）

- `sessionquery`：索引助手成稿的两种 kind（`assistant_message` 与 `message`）；读日志只认完整行、坏行跳过、末尾半行忽略，**不再在盘上截 torn tail**；单份坏文件不拖垮整个检索，结果尾部注明。多关键词三级放宽：FTS 各词 AND → 同一条消息里各词都是子串 → 同一会话的不同消息里分别出现。`query` 留空 = 列出最近的会话（标题 / 起止 / 用户与助手条数 / 首条任务），来自 `.titles.json` 与索引。零命中时报告索引规模。
- `control/refs_session.go`：`@会话` 快照同样认 `message` kind（此前实时会话只投影得出 [user] 行）。
- `sessiondir_fence.go`：围栏拒绝的话术改成两步具体做法。

### 2.4 PDF 页数（fric-6 / miss-13）

- `docmd/pagespec.go`：`capPageSpec` 的上限口径从「起始页 ≤ 上限」改成「单次请求的页数 ≤ 上限」；`pages=501-676` 可直接补转；显式超限报错并给分段写法（不再静默裁剪）；删掉 `clampPageSpec`。
- `format_convert`：新增 `all_pages=true`（超限自动分段转完再拼接，段间 HTML 注释记页码，须给 output、与 pages 互斥）；回执带「页数：共 N 页，本次 a–b 页」；截断时文件头尾各放一份醒目告警（缺哪几页 + 续转办法）。续转中途失败点名段落与已完成段数，不留半份文件。

### 2.5 工具面预算

`TestOfficeToolFaceBudget`：standard 17000→17250（实测 17156）、cost 29000→29250（实测 29212）、proposal 21500→21750（实测 21615）；wenxue 13519/13750 不变。涨的是新参数的键、bash / xlsx 审计的描述，以及 cost_library 的描述与模式行。

## 3. 做到一半：xlsx 健康体检（miss-7 / wf-15③）

**已落并有测试**：`internal/office/xlsxedit/health.go`——`Inspect(path)`（zip + 工作簿/工作表 XML 是否完好、公式格里没有可用缓存值的数量、错误值、定义名及表级归属数、数据验证、条件格式、合并格）与 `StructureDiff`；`health_test.go` 用真实样表验（15 个定义名全是表级、0 缺缓存值）。

**已接线、没补测试（→ 续会话已补，见 §9）**：
- `xlsx_audit_tool.go`：回执第一行「健康体检：可打开｜公式 N 格…｜结构一行」；打不开直接报错不去扫单元格。
- `xlsx_recalc_tool.go`：重算前后各体检一次，回执追加「结构对账」（纯函数 `recalcStructureNote`）。
- `gaea_xlsx_edit.go`（xlsx_apply）：重算失败不再静默；缺缓存值的格数如实报出。
- `tool/publishguard.go`：新增 `Touched()`（最近一次 Scan 见到的全部 durable 新增/修改，含豁免面）。

**还没做（→ 续会话已全部做完，落地差别见本节末）**：
1. `bash.go` 接缓存值提示（脚本写出 xlsx 是这条反馈的主场景）。三处 `publishCheck(pub)` 之后（提前返回的 json/plain 两处、正常收尾的 json/plain 两处）与后台 job 收尾处，各追加 `xlsxCacheNotes(pub)`：

```go
// xlsxCacheNotes 事后体检：命令写出的 xlsx 里公式格没有缓存值的，回执里提一句
// （openpyxl / excelize 保存会丢缓存值，Excel/WPS 打开是空值）。
func xlsxCacheNotes(pub *tool.PublishGuard) string {
	if pub == nil {
		return ""
	}
	var lines []string
	for _, v := range pub.Touched() {
		if !strings.EqualFold(filepath.Ext(v.Path), ".xlsx") || strings.Contains(v.RelPath, "~$") {
			continue
		}
		h, err := xlsxedit.Inspect(v.Path)
		switch {
		case err != nil:
			lines = append(lines, fmt.Sprintf("⚠ 脚本写出的 %s 打不开（%v）", v.RelPath, err))
		case h.NoCachedValue > 0:
			lines = append(lines, fmt.Sprintf("⚠ 脚本写出的 %s：%s——交付前 xlsx_recalc", v.RelPath, h.CacheLine()))
		}
	}
	return strings.Join(lines, "\n")
}
```

   `builtin` 包引 `internal/office/xlsxedit` 无环（xlsxedit 不依赖 tool）。只在命令成功或失败都追加（备注，不置错）。

**实际落地与上面这份草稿的差别（2026-11-28）**：
- 函数签名多了 `settled`：提前返回（进程仍在后台跑）传 `false`，只报缺缓存值、不报「打不开」——半写的文件不算坏；后台 job 收尾与前台终段传 `true`。
- 被发布边界隔离的件原位已不在，按隔离后的落点（`LastQuarantineDest()` + 相对路径）体检，不把「被移走」说成「打不开」；原位与隔离位都不在则跳过。
- 挂载点共五处：前台终段 plain / json（json 进 `xlsx_notes` 字段）、提前返回 plain / json、后台 job 收尾（备注进 job 缓冲，bash_output / wait 读得到，不占 result 位）。
- **越界时 result 仍是隔离全文 + 备注**（传输契约）。我第一版把 plain 终段写成 `return out`，把取回办法从 result 里挤掉了——`TestBashXlsxCacheNotesOnQuarantinedFile` 当场逮住。
2. ✅ app 层测试：`recalcStructureNote`、xlsx_audit 健康体检行（含打不开报错）、xlsx_apply 缺缓存值备注（`internal/app/xlsx_health_test.go`）、bash 写出未重算 xlsx 的提示（`internal/gaea/tool/builtin/bash_xlsx_notes_test.go`，含 json 模式 / 隔离件 / 后台 job / 未收场四形态）。
3. 两个测试文件的 gofmt 已处理（`office_toolface_test.go`、`tool_precheck_test.go`）。

## 4. 没动的条目与建议做法

- **#4 发布边界**：①`move_file` 覆盖正式名下的 Office 件时留上一版快照到 `.gaea/versions/` 并在回执里说明（脚本直写覆盖时旧版已经没了，快照只能在合规出口做）；②被拦的如果是「改写已有正式件」，自动把隔离件放一份到 `.gaea/work/<原相对路径>`，省掉手动取回那一步。`xlsx_apply` 不加 eval 类操作——那等于绕开边界。
- **#5'**：回执改成「全库共 N 份（含录入中 M 份；含本方案）」+ 一行本方案进度。
- **#11**：outline 条目增 `display_title`（匹配用 `title`，入库用 `display_title`）。
- **#12**：`proposal_save` 增「辅助认标题」（只回带行号的候选标题清单，不写库），或 format_convert 对无井号输出做标题形态识别。
- **#8**：`cost_project quotas/resources` 增 keyword 精确匹配开关。
- **#14**：会话收尾的「产物交接卡」与开场注入；最小做法是给 `sessionquery.Recent` 加「末条助手回复摘要」，再往上是落盘交接卡。
- **#15②**：断点恢复块只存在于进程内存，重启即丢；可在恢复会话时从持久化日志重建上一轮的工具调用清单（`agent/interrupted_recovery.go`）。

**待用户拍板**：#10 矢量图补图（要引 PDF 库依赖，Go 侧无等价库）；#3 的 277 条章节数据恢复；是否要给测算样表配「按数据扩骨架」的填表脚本（技能脚本，不进 Go；本次没做，说明书里给的是 xlsx_apply 配方与脚本要点）；样表里三处旧文案（N15 起备注「综合单价见 S 列」、封面编制说明「两层 ROUND」、综合单价 E 列上挂错位置的下拉列表）要不要用 gaea 改。

## 5. 接续步骤（✅ 2026-11-28 凌晨续会话六步全部做完，结果见 §9）

1. `git status --short | grep -v '^?? releases/'` 核对 §7 的文件都在；确认没有别的执行体在同一棵树上动过。
2. 做完 §3 的三件（bash.go 提示、app 层测试）。
3. 全量门禁：`gofmt -l`（改过的文件）、`go build ./...`、`go vet`（改动包）、`golangci-lint run --new-from-rev HEAD`、分包测试（§6 的命令），前端 `cd frontend && npx tsc -b && npx vitest run src/pages/CostLibraryPage.test.tsx`。app 全量与前端全量串行跑（并发会被双双杀掉）。
4. 提交（建议六笔：造价线 / bash 与文件闸 / 会话检索 / PDF 页数 / xlsx 体检 / 记档）。中文提交信息落 UTF-8 文件后 `git commit -F`。
5. 打包装机按 AGENTS.md 规约与 `docs/gaea-build-guide.md`：从提交的干净工作树构建，包版四件回写主树 `build/bin`。
6. 同步 `.gaea/progress.md`、`.gaea/todos.md` 与记忆 `gaea-ai-feedback-20261127-batch`。

## 6. 验证结果（2026-11-27 深夜，主工作树）

- `go build ./...` 通过（ucrt64 + `build/bin` 前置 PATH，`CGO_ENABLED=1`）。
- `go vet`：sessionquery / control / docmd / xlsxedit / skill / workcost / builtin / agent / app 全过。
- 分包 `go test` 全绿：sessionquery、xlsxedit、skill、workcost、agent/textutils、builtin（全包）、docmd（`-run 'Page|Spec|ConvertLimit|Truncat'`）、control（全包）、agent（`TestPrecheck|TestStale|TestEditFile|TestMoveFile`）。
- app 分组跑（每个模式单独一次，`-run` 不能带 `|`）：`TestCost`、`TestOfficeToolFace`、`TestOfficeModeSystemPrompt`、`TestWenxueToolFace`、`TestOfficeRunMode`、`TestXlsx` 全绿。
- 改过的 .go 文件 `gofmt -l` 干净；control 包测试写脏的夹具 `resumed.jsonl.meta` 已还原。
- **没跑**：app 全量、agent 全量、前端全量（只跑过 `CostLibraryPage.test.tsx` 40/40）、golangci-lint、真实模型实弹、打包装机。

### 续会话验证（2026-11-28 凌晨，主工作树，收口前的全量）

- `gofmt -l` 净（改动 + 新增 .go）；`go build ./...` 过；改动包 `go vet` 过；`golangci-lint run --new-from-rev HEAD` **0 issues**（首跑逮到 sessionquery 两处 `rows.Close()` 没接返回值，已改 `_ = rows.Close()`）。
- 纯 Go 10 包**全量**绿（不带 ucrt64，系统 Python 排前）：sessionquery 2.2s / xlsxedit 2.0s / skill 1.2s / workcost 9.8s / textutils 0.4s / tool 2.1s / builtin 43s / docmd 18s / control 2.9s / **agent 19s**，合计 48s。
- app 包**全量 1391 例四分片 77s，fail=0**（`go test -c` 编一次放系统临时目录，`.tmp/run_app_shards.mjs` 四片）。
- 前端 `tsc -b` 过（32s）；vitest 全量 **510/511 文件、4806/4807 例**（294s）。红 1 例 `src/pages/modelcenter/SchedulingSection.test.tsx`「切换开关调用 setEngineFailover(true)」waitFor 超时——与本批零交集（本批前端只动 CostLibraryPage），隔离 3 跑 1 红 2 绿，定性既有偶发，记遗留。
- 仍没做：真实模型实弹、真机走查。

## 7. 改动文件清单（首会话时未提交；提交归属见 §9）

- 造价：`internal/app/office_cost_payload_files.go`（新）+ `_test.go`（新）、`office_cost_library_tools.go`、`office_cost_library_schema.go`、`office_cost_library_schema_test.go`、`office_cost_import_tools.go`、`office_cost_template_sample.go`、`office_cost_library_template_test.go`、`office_cost_library_test.go`、`office_cost_library_rules_test.go`、`gaea_cost_library_export.go`、`office_runmode_skills.go`、`office_runmode_systemprompt.go`、`office_runmode.go`、`office_toolface_test.go`；`internal/gaea/control/controller_approval.go` + `cost_library_approval_test.go`；`internal/gaea/skill/cost_estimate_assets.go`、`cost_estimate_body.go`、`cost_estimate_skill_test.go`、`assets/成本测算表-使用说明书.md`（新）；`internal/gaea/workcost/projectimport.go`、`projectimport_sheets.go`、`workbook_recon.go`（新）、`workbook_recon_test.go`（新）；`frontend/src/pages/CostLibraryPage.tsx`、`CostLibraryPage.test.tsx`。
- bash / 文件闸 / 读文件：`internal/gaea/tool/builtin/bash.go`、`bash_longcmd_test.go`（新）、`compact.go`、`readfile.go`、`readfile_bytes_test.go`（新）、`sessiondir_fence.go`；`internal/gaea/agent/stale_paths.go`（新）、`stale_paths_test.go`（新）、`tool_precheck.go`、`tool_precheck_test.go`、`agent_helpers.go`、`execute_one.go`、`edit_tools_agent_test.go`；`internal/gaea/agent/textutils/budget.go`。
- 会话检索：`internal/gaea/sessionquery/sessionquery.go`、`tool.go`、`sessionquery_test.go`；`internal/gaea/control/refs_session.go`、`refs_session_test.go`。
- PDF 页数：`internal/docmd/pagespec.go`、`pagelimit_test.go`、`pdf_pages_test.go`；`internal/gaea/tool/builtin/format_convert.go`、`format_convert_pages_test.go`（新）。
- xlsx：`internal/office/xlsxedit/health.go`（新）、`health_test.go`（新）；`internal/app/xlsx_audit_tool.go`、`xlsx_recalc_tool.go`、`gaea_xlsx_edit.go`；`internal/gaea/tool/publishguard.go`。
- 记档：本文、`.gaea/progress.md`、`.gaea/todos.md`。
- **续会话（2026-11-28）新增**：`internal/app/xlsx_health_test.go`（新）、`internal/gaea/tool/builtin/bash_xlsx_notes_test.go`（新）；`internal/gaea/sessionquery/sessionquery.go` 两处 `rows.Close()` 接返回值（lint）；`bash.go` 的体检挂载点（§3 末）。

## 8. 本批坑（都是当场踩的）

- **Git Bash 的 `bash -c` 参数上限约 8.2K 字符**（Windows）：超过就静默截断、rc 仍为 0；探针脚本与结论见 §2.2。
- **文件闸的路径口径**：工具绑工作区后相对路径由工具自己解析，闸/预检查若拿原始串按进程 cwd 读就全错；写回归用例必须用「工作区绑定工具 + `t.Chdir` 到别处 + 相对路径」的形态，旧用例全用绝对路径所以一直是假绿。
- **`message` vs `assistant_message`**：实时落盘的助手成稿是前者，迁移/子代理提升才是后者；读会话日志的地方两种都要认。
- **`capPageSpec` 口径变更**：显式超限的 `pages` 现在报错而不是裁剪；preview / wenxue 那两条内部调用传的是空规格，行为不变。
- **工具面预算只算键**：compact schema 的描述会被压缩掉，涨的是新参数的键和工具级 CompactDescription。
- **ZCode 工具**：`gofmt -w` 之后要重新 Read 才能 Edit；`for … range X{…}.Y()` 要先把复合字面量赋给变量；包内已有 `min/max`，测试里别重定义；`map[K]V{}` 别漏花括号；未用 Read 读过的文件不能 Edit（`cat`/`sed` 看过不算）。
- **`-run` 正则不含 `|`**（cgo 包装器会把它吃掉——收尾验证自己又踩了一次，app 那组拆成六次单模式才跑起来）；带 cgo 的包用 `scripts\cgo_test_dll_prep.cmd` 绝对路径做 `-exec`。
- 取证脚本（只读会话日志）放在 `%TEMP%\gaeaprobe\`，不入库：`log_forensics.py`、`log_forensics2.py`、`log_trace.py`、`argv_limit_probe.py`、`argv_limit_probe2.py`、`dump_sample*.py`、`gotest.sh`（app 包测试跑法）、`verify_batch.sh`（本批收尾验证脚本）。

续会话（2026-11-28）追加：

- **传输契约一改就破**：bash 的越界回执里 result 必须是隔离全文（信封 detail 取的是它）。体检备注接进 plain 终段时我写成了 `return out`，取回办法被挤掉，模型只会看到「已隔离：」断头句——新测试当场逮住。凡是改 bash 回执拼装，先问「result 里还有没有完整文案」。
- **办公工具回执是 `tool.WrapText` 的 JSON 信封**：正文在 `message` 字段，换行是转义的 `\n`。测试要按行取内容得先解信封，直接 `Split(out, "\n")` 取不到。
- **excelize `SaveAs` 只认 .xlsx / .xlsm 一类扩展名**（别的是 `unsupported workbook file format`）：造「非 durable 名」的种子件要先 `WriteToBuffer` 再 `os.WriteFile`。
- **`golangci-lint --new-from-rev HEAD` 会把未跟踪的新文件算进新增面**——这正是它该干的；errcheck 要求 `rows.Close()` 接返回值。
- **干净树构建后 `frontend/wailsjs` 一片 M 多半是 autocrlf 幻影**：判绑定有没有真漂移用 `git diff --ignore-cr-at-eol --stat`，为空（或只剩同行文本）就是没漂移。
- **control 包全量测试会写脏夹具** `internal/gaea/control/resumed.jsonl.meta`（既有卫生问题）：提交前 `git checkout --` 还原，别把它带进提交。
- **前端全量偶发**：`modelcenter/SchedulingSection.test.tsx` 切换开关一例 waitFor 超时，隔离 3 跑 1 红——零交集 + 复跑不恒红即定性偶发，不拦收尾。
- **PowerShell 建 / 摘 junction 走脚本文件**（`%TEMP%\gaeaprobe\wt_junction.ps1`）：建用 `New-Item -ItemType Junction`，摘用 `cmd /c rmdir <链接>`（不带 `/s`，只删链接），前后核对主树 node_modules 条目数（本次 500→500）。
- **分笔提交的顺序自洽用静态核对**（`%TEMP%\gaeaprobe\check_partition.py`）：后一笔才新增定义的顶层符号，不得出现在更早各笔的新增行里。只认行首顶层定义——放开缩进会把函数内局部 `var` 全算成定义，满屏误报。
- 本轮新增脚本（都在 `%TEMP%\gaeaprobe\`，不入库）：`gate_a.sh`（gofmt / build / vet / lint）、`gate_b.sh`（纯 Go 分包全量）、`gate_c.sh`（app 四分片）、`gate_d.sh`（前端 tsc + vitest 全量）、`pkg_fb_build.sh <commit8> <YYYYMMDD>`（干净树构建 → payload）。

## 9. 收口记录（2026-11-28 凌晨，续会话）

用户令「继续开始」，从 §5 接手。

### 9.1 做完的三件

1. `bash.go` 接体检备注：`xlsxCacheNotes(pub, settled)` 挂五处（前台终段 plain / json、提前返回 plain / json、后台 job 收尾）；与 §3 草稿的差别见 §3 末。
2. 测试：`internal/app/xlsx_health_test.go`（结构对账纯函数、xlsx_audit 体检行三形态、xlsx_apply 重算没做成报缺缓存值 + 无公式不误报）、`internal/gaea/tool/builtin/bash_xlsx_notes_test.go`（plain / json / 重算过不提 / 坏件 / 非 xlsx / 隔离件 / 后台 job / 未收场）。
3. lint 两处 errcheck。

### 9.2 门禁

见 §6「续会话验证」。

### 9.3 提交（未 push）

| 笔 | 提交 | 内容 | 文件 |
|---|---|---|---|
| C1 | `a78c83a7` | 造价线：工作流重排 / 载荷走文件 / 使用说明书 / 导入演练两道检查 / 确认卡 / 前端导出菜单 / 工具面预算（全批终值） | 27 |
| C2 | `1b628dc7` | xlsx 健康体检：`xlsxedit.Inspect` + audit / recalc / apply 三处接线 + `PublishGuard.Touched()` | 7 |
| C3 | `0ae43cb3` | 会话检索：session_search / `@会话` / 会话目录围栏话术 | 6 |
| C4 | `73739a34` | PDF 页数：`capPageSpec` 口径 + `all_pages` | 5 |
| C5 | `093c83c5` | bash 与文件闸：长命令 / 数据对象 / stale 闸路径 / read_file 字节窗口 / xlsx 提示 | 14 |
| C6 | 记档 | 本文、`.gaea/progress.md`、`.gaea/todos.md` | 3 |

切分顺序经静态核对自洽（C2 在 C5 前：bash 用到 `Touched()` 与 `Inspect`；C4 在 C5 前：`compact.go` 里 `all_pages` 的紧凑 schema 随 C5 落，晚于 format_convert 对它的支持）。`office_toolface_test.go` 的预算只设上限，放 C1 带终值，后面各笔中间态只会更小。

### 9.4 包与装机

- 包：`releases/gaea-ai-feedback-20261128-093c83c5/`（payload 四件 + zip 34,194,637 B + src.zip 831,539,059 B + SHA256SUMS + PACKAGE-INFO.md）。
- 构建：干净树 `.worktrees/pkg-093c83c5`（node_modules junction），vite 63s + wails 60s，bindings 未跳过，导入表含 `sherpa-onnx-c-api.dll`。构建后 `frontend/wailsjs` 的 M 全是行尾幻影，绑定无漂移。
- 哈希（前 16）：exe `6f488036a40e626f`；onnxruntime `68d0d5f502be4509`；sherpa-c-api `408f9ab683d1b77c`；sherpa-cxx-api `dbac55ceca8383fe`；zip `2c822b1e1450bef3`；src.zip `2cf64218390527d1`。
- 冒烟：payload 正式路径 `/api/health` **200**，进程已回收。
- 装机：INSTALL OK，新壳 **PID 53224**（00:10:16 > 构建时刻 00:09:27），`18123/healthz` 200。
- 回写：包版四件回写主树 `build/bin`，payload / build/bin / 装机位三处哈希一致。构建工作树已回收（junction 先摘，主树 node_modules 500→500）。
- 版本号未抬。**未 push。**

### 9.5 留给下次的

- **待用户拍板**（不变）：#10 矢量图补图；#3 的 277 条章节数据恢复；测算样表三处旧文案是否用 gaea 改；要不要给样表配「按数据扩骨架」的填表脚本。
- **待令小刀**（§4 有做法）：#4、#5'、#11、#12、#8、#14、#15②。
- **真机走查**：造价技能新主路（样表上编 → 用户检查 → 入库）、载荷走文件、bash 写出 xlsx 的缺缓存值提示、session_search 留空列会话、PDF `all_pages`。
- 测试卫生两条：SchedulingSection 偶发；control 包写脏 `resumed.jsonl.meta`。

## 10. 余量七条收口（2026-11-28 上午，同一续会话的后半段）

用户令「继续」，按原指令「修复这些问题」接做 §4 的七条（4 个拍板项与 push 不碰）；#14 做到一半时用户拍板**「不自动注入，模型自己判断是否读取」**。

### 10.1 逐条落地

| 条 | 做法 | 提交 |
|---|---|---|
| #5' | 回执拆两行：「落点：工作区 X」+「方案库：全库共 N 份（现行 a、录入中 b），含本方案」；演练新方案写「不含本方案；正式写入后 N+1 份」 | `5ddd8408` |
| #11 | outline 条目增 `display_title`：`title` 定位、`display_title` 入库。给了显示名时 title 差一两个字也能对上（指了行号认那一行，否则在标题样行里找最接近的；短于 4 字不容错）；源文件行带编号而显示名没带时补回编号；回执列「标题修正」请核对 | `5ddd8408` |
| #12 | `proposal_save action=detect_headings`（只读）：按井号 / 第X章 / 第X节 / 1.1 / 一、/（一）/ 1. 认标题样行并推建议层级；目录行与代码块排除，表格行里藏的章名单列。再给 `outline_file` 就把拿得准的写成草稿文件（标准 JSON 数组、每行一条）；录入时 `outline_file` 读草稿。认标题与查进度不再弹确认卡（此前查一次进度也弹） | `5ddd8408` |
| #8 | 检索层 `SearchQuery.Exact` + `cost_project exact=true`：只认字面匹配、不走向量与精排，按编码稳定排序；回执「字面匹配共 N 条，本次第 a~b 条」，可翻页自证穷尽 | `13642743` |
| #4 | ① `move_file` 覆盖正式交付面的 Office 件前留上一版到 `.gaea/versions/<原相对路径去扩展名>/<旧文件修改时间>.<扩展名>`，回执给快照路径；② 脚本改写已有正式件被隔离时自动放一份到 `.gaea/work/<原相对路径>`，回执给一步回正式名的 move_file 写法 | `06780c87` |
| #15② | 会话换入（`SetSession`）时最后一轮没收尾就从持久化消息重建交接记录（硬杀形态也认），带任务原话；消费时先把待办从日志重建出来 | `a98861c3` |
| #14 | `session_search handoff=<会话 id>`：交接卡（首条任务、收尾回复、经工具写过的文件及现状，xlsx 走体检钩子）；最近会话清单带「收尾」一行；会话围栏话术指路 | `f35fb3f4` |

### 10.2 没做与已知盲区

- `xlsx_apply` 的 eval 类自定义操作：等于绕开发布边界，判不做。
- 交接卡认不出 bash 脚本内部写出的文件（只认带重定向的）；旧格式日志里读写只能按工具名区分，个别只读的可能混入。
- 交接卡只给 xlsx 体检；「关键勾稽结论 / 已知残留问题」仍是上一场的自述，要靠对应的审计工具（xlsx_audit / cost_library 导入演练 / md_validate）复核。
- 版本快照不自动清理（被覆盖的文件留全部历史），用户可自行清理 `.gaea/versions/`。
- 中断重建的交接块里文件指纹是恢复时的现状，不是中断时的。

### 10.3 门禁 / 包 / 装机

- 门禁（终态 `f35fb3f4`，主工作树）：gofmt 净；`go build ./...` 过；改动包 vet 过；`golangci-lint --new-from-rev 943cb37e` 0 issues；纯 Go **12 包全量**绿（比上一轮多了 proposal、boot，50s）；app **1392 例四分片 81s fail=0**。前端本轮零改动，未重跑 vitest。
- 工具面预算：proposal 21750→22000（实测 21713）、cost 29250→29500（实测 29239）；session_search 描述加交接卡后一度把 standard 档顶到 17339，压紧描述回到 17139，standard 没抬。
- 包：`releases/gaea-ai-feedback-20261128-f35fb3f4/`（payload 四件 + zip 34,237,773 B + src.zip 831,595,452 B + SHA256SUMS + PACKAGE-INFO.md）。干净树 `.worktrees/pkg-f35fb3f4`，vite 63s + wails 66s，导入表含 `sherpa-onnx-c-api.dll`，绑定无漂移（wailsjs 的 M 仍是行尾幻影）。
- 哈希（前 16）：exe `ef14ca6c6b2c9a46`；DLL 三件同上一包；zip `a0a7f7e2d24bf4d0`；src.zip `5cd345b66b83742c`。
- 冒烟 `/api/health` 200；装机 INSTALL OK，新壳 **PID 35548**（07:05:36 > 构建时刻 07:05:06），`18123/healthz` 200；包版四件回写主树 `build/bin` 三处哈希一致；工作树已回收。

### 10.4 本轮新坑

- **提交前先看有没有顺手改了既有行为**：给「模型漏写编号时自动补源文件编号」加了一条规则，撞上既有测试钉的「模型写什么存什么」——超出 #11 范围，撤掉。一刀只管一事。
- **交接块的前置发生在待办重建之前**：重启后首轮的清单此时还是空的，重建来的记录在消费时得先补做一次，否则用过待办工具的轮也会被写成未记录。
- **常驻面只紧不松**：给 session_search 加能力后描述涨了 183 字符、standard 档超预算——压紧描述而不是抬预算。
- **带变量的 Windows 路径别手拼反斜杠**（坑册旧条再犯）：`"$REL\\payload"` 传给 PowerShell 被吃掉、压缩脚本静默无输出；要单引号写死路径或 `cygpath -w`。
- **草稿文件的读写形态要对齐**：草稿按「每行一条」写、读入只认 JSON 数组，测试当场逮住——现在写标准数组（每行一条），两种形态都认。

### 10.5 余项

- **push 待令**；真机走查：方案库认标题 → 草稿 → 录入、display_title 改错字、造价 exact 盘点、move_file 覆盖后 `.gaea/versions/` 有快照、脚本改写正式件被拦后的回执、重启后中断会话的交接块、`session_search handoff`。
- **拍板落定（2026-11-28）**：277 条章节数据恢复、样表三处旧文案、样表「按数据扩骨架」脚本——**不做**；拍板池只剩 #10 矢量图补图（引 PDF 库依赖）。
- 测试卫生两条不变：SchedulingSection 偶发；control 包写脏 `resumed.jsonl.meta`。
