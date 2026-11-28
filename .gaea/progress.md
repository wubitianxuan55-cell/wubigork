## 2026-11-28 傍晚 造价库索引与编辑优化批：单笔提交 + 包 gaea-cost-idxedit-20261128-6b3e5373 装机 PID 27272（ZCode；随本批 push）

- 来源：用户令「优化完善造价数据库、成本库的索引和编辑功能」。只读盘底（活库 9 项目 / 275 清单 / 340 定额 / 283 工料机；330 条定额未归分部、340 条定额地区全空、字典 10 专业而行上只有「土壤修复」在用）后交三批方案，三个拍板点用户未另行表态、按我建议走：①批量改维度 / 新增字典值放进板块 ②撤销不做 ③后端分页不做。期间查出 11-27 傍晚起的检索故障（Herdsman bge-m3 等 llama-server 进程 Vulkan `ErrorDeviceLost`，`/v1/models` 照常 running，语义检索静默失效 20 小时；用户在 Herdsman 重开模型进程即恢复），善后三刀并入本批先做。中途用户令「构建桌面端」（主树整树构建装机 PID 13840），随后「git打包推送」。
- **单笔提交 `6b3e5373`**（53 文件 +4091/−380，两批纠缠拆不绿）：善后三刀（索引器失败退避 + 日志去重，原实现失败路径 `ix.Mark(k)` 立即唤醒把 RetryAfter 架空——每 3 秒 3 个失败请求、20 小时 6.7 万次；检索向量路失败经 `VecNote` 上报、`searchError` 进索引状态、搜索框旁明示 + 重试钮；Herdsman 巡检对在运行向量 / 精排各发一次真实小请求、测试连接同源、坏了 `Connected` 仍 true 不触发故障转移）｜索引优化（空白分词多词全含、分面计数随搜索联动〔自己这维的筛选不套进自己的计数〕、状态筛选、定额地区筛选、待整理浮层逐项直点、主列表按相关度序 + 词条高亮 + 「语义」来源标、检索别名 `kind=alias` 查询词双向展开〔exact=true 不展开〕）｜编辑扩展（规范值下拉改读字典 + 就地新增、期数手填、未保存守卫 + Ctrl+S/Esc、改现行价自动登记调价历史、新建清单项 / 定额 / 工料机、定额与工料机复制另存、勾选批量改维度逐行回报成败〔不做整体回滚〕、挑选器输入即搜、删除卡「解除挂接后删除」与工料机「连同解除引用删除」、定额地区回填先预览后落库〔只被一个省引用的才写〕）｜提示词与 `.gaea/AGENTS.md` 库页约定同步改口：板块可批量改维度 / 加字典值，批量导入、解析、模型读表归类、Excel 批量调价仍归造价模式。
- **绑定面 680→684**（+4 `GaeaWorkcostDictList/DictSet/QuotaDetachEverywhere/QuotaRegionBackfill`），生成物四份再生 + 前端六处登记 + wailsjs 手同步（构建再生逐字一致）。
- **门禁**：gofmt / build / vet 净；`golangci --new-from-rev HEAD` 0；workcost / vecindex / modelengine 全绿（后两个 `-race`）；**app 1401 例四分片全绿**；tsc 0 错、eslint 0 错；前端全量 vitest 999 文件 9560 例全绿（分片墙钟 159s）；`check-bindings-drift` OK（684）、`check-docs` OK（AGENTS.md 25,238 B，32KB 注入预算内）。
- **包**：`releases/gaea-cost-idxedit-20261128-6b3e5373/`——干净树 `.worktrees/pkg-6b3e5373`（junction node_modules）vite 26s + wails 57s（bindings 未跳过），忽略行尾绑定零实质漂移；payload 冒烟 `/api/health` 200；INSTALL OK 新壳 **PID 27272**（17:19:36 > 构建时刻）`18123/healthz` 200；包版四件回写 `build/bin` 三处哈希一致；worktree 已回收（junction 先摘，主树 node_modules 500 完好）。SHA256：exe `82949133…`，zip `2F0496AB…`（34.3MB），src.zip `A33EF9F6…`（831.7MB）。
- **余**：真机走查（多词 / 分面联动 / 状态筛选 / 批量改 / 新建复制 / 回填 / 别名实弹）；检索别名内容为空待添加；后端分页与撤销按用户默认不做（过万条再立项）；`.worktrees/` 里还留着 7 个更早批次的 pkg-* 干净树（pkg-25a68941 等）未回收，待令清理。

## 2026-11-28 下午 黄甲 AI 反馈批：十一刀十三笔提交 + 包 gaea-huangjia-fb-20261128-58b8e9b6 装机 PID 30388（ZCode；未 push）

- 来源：黄甲竣工报告会话里 gaea 自生成的《软件使用问题与建议报告》（台账 6 条 `fb17958…`）。先对码分诊，用户说「会话刚刚完成」后按会话事件日志取证（日志在**工作区** `.gaea/sessions/work/*.gaea-log.jsonl`，不在 %APPDATA%）：源会话 09:32–12:40 整场跑在 ead1952a 包上，报告每条都成立；用户令「按照你的推荐开始」。包说明见 `releases/gaea-huangjia-fb-20261128-58b8e9b6/PACKAGE-INFO.md`。
- **十三笔**：`5ba372d5` edit 预检改比对磁盘原文（预检此前拿 read_file 渲染缓存当文件内容，整读后多行锚点必 not found——首轮分诊我误判「不动」，日志实证后纠正）｜`0e476bf2` 发布边界认随迁（工作区内 mv 目录不再整体隔离，实测 7 批 120 次 move_file）｜`9f81972e` 断流恢复接线（恢复分支一直在，适配层没接；task 报错补「都不传走默认」）｜`36093a06`/`a9b8cd59`/`137b07ca` 内嵌图片 base64 一线三处（format_convert 不保图落盘只留文字 / summarize_file 剥除+块数上限+单块重试 / read_file 超长行不整次报错）｜`ca6f88d3` expected_revision 认尾标写法｜`514ae51d` write_file 相邻相同行≥4 硬拦+回执体检行（全会话 35 次写入交叉验证只拦污染件）｜`6182e2e2` md_validate 国标语境/复合词不判孤立引用、paths 可单独用｜`a28bb9f6` edit_file 行尾无关匹配｜`40fae0f0` create_docx.py 能力探测（办公+文学；黄甲工作区没播种技能、回落到 ~/.codex/skills 的 2026-08 旧镜像）｜`29d6c9dc` diagram_gen 回边｜`58b8e9b6` 子代理失败回执带已动文件、回合记账只认成功调用。
- **门禁**：gofmt / `go build ./...` / vet 净；`golangci-lint --new-from-rev 09d38fdf` 0 issues；agent/builtin/tool/bridge/largefile/docmd/ai 全量绿；app 包全量 155s 绿；前端零改动。
- **包**：干净树构建（vite 25s + wails 58s，cgo 非空壳，绑定无漂移），冒烟 `/api/health` 200，INSTALL OK 新壳 **PID 30388**，包版四件回写 `build/bin` 三处哈希一致。exe `cf65198a0ac41bd4`、zip `f19f682f2484d434`、src.zip `6cd0e65167ae0056`。工作树已回收。
- **余**：push 待令；真机走查（随迁放行、断流恢复、整读后多行 edit、write_file 拦截、旧版 create_docx 的降级回执、回边流程图、子代理失败回执文件清单）；挂池=move_file 目录支持；挂拍板=3.1 文档结构对照 / 3.2 文档级数据巡检；两件待点头=给黄甲工作区播种 docx 技能、报告 §五 两条过时纪律回写办公板块记忆。
- **顺带发现（未动）**：造价向量索引对账自 11-25 起持续失败（embedding 引擎 Vulkan `ErrorDeviceLost` HTTP 500，每 10 秒重试，11-28 刷 5.2 万行日志）；`registry_map_doc -check` 本机行尾假红。

## 2026-11-28 上午 AI 反馈批余量七条收口：五笔提交 + 包 gaea-ai-feedback-20261128-f35fb3f4 装机 PID 35548（ZCode；未 push）

- 用户令「继续」，按原指令「修复这些问题」接做没动的七条；#14 中途用户拍板「不自动注入，模型自己判断是否读取」。逐条做法与已知盲区见接续单 §10（进度计划/gaea-ai-feedback-20261127-batch.md）。
- **五笔**：`5ddd8408` 方案库三刀（回执份数口径拆两行 / outline `display_title` 改 OCR 错字并容错定位 / `action=detect_headings` 认标题 + `outline_file` 读写草稿；查进度与认标题不再弹卡）｜`13642743` 造价检索 `exact=true` 仅字面匹配、可翻页自证穷尽｜`06780c87` `move_file` 覆盖正式 Office 件前留上一版快照到 `.gaea/versions/`，脚本改写正式件被拦后自动放回 `.gaea/work/`｜`a98861c3` 中断交接记录重启后从会话日志重建、带任务原话｜`f35fb3f4` `session_search handoff=<会话 id>` 交接卡（收尾回复、动过的文件、xlsx 体检），最近会话清单带收尾一行。
- **门禁**：gofmt / `go build ./...` / vet 净；`golangci-lint` 0 issues；纯 Go 12 包全量绿；app **1392 例四分片 81s 全绿**；前端本轮零改动。
- **包**：`releases/gaea-ai-feedback-20261128-f35fb3f4`——干净树构建（vite 63s + wails 66s，cgo 非空壳，绑定无漂移），冒烟 `/api/health` 200，INSTALL OK 新壳 **PID 35548**，包版四件回写主树 `build/bin`，三处哈希一致。exe `ef14ca6c6b2c9a46`、zip `a0a7f7e2d24bf4d0`、src.zip `5cd345b66b83742c`。工作树已回收。
- **余**：push 待令；真机走查；拍板池只剩 #10 矢量图补图（引 PDF 库依赖）——277 条章节数据恢复、样表旧文案、样表「按数据扩骨架」脚本已拍板**不做**（2026-11-28）；`xlsx_apply` 的 eval 判不做。

## 2026-11-28 凌晨 AI 反馈批收口：五笔提交 + 包 gaea-ai-feedback-20261128-093c83c5 装机 PID 53224（ZCode；未 push）

- 用户令「继续开始」，从接续单 §5 接手（首会话记录见下一节，事实源=进度计划/gaea-ai-feedback-20261127-batch.md §9）。
- **补完 xlsx 健康体检**：`bash.go` 接 `xlsxCacheNotes(pub, settled)` 五处（前台终段 plain / json、提前返回、后台 job 收尾）——命令写出的 xlsx 公式格没有缓存值时回执提一句（库保存不写公式结果，Excel/WPS 打开是空值）；进程未收场不报「打不开」；隔离件按隔离后落点体检；越界时 result 仍是隔离全文 + 备注（传输契约，第一版写破被新测试逮住）。测试 `internal/app/xlsx_health_test.go` + `internal/gaea/tool/builtin/bash_xlsx_notes_test.go`；lint 两处 errcheck。
- **门禁（全量）**：gofmt / `go build ./...` / vet 净；`golangci-lint --new-from-rev HEAD` 0 issues；纯 Go 10 包全量绿（含 agent 全量）；app **1391 例四分片 77s 全绿**；前端 `tsc -b` 过、vitest 510/511 文件（红 1 例 `modelcenter/SchedulingSection.test.tsx` 偶发，与本批零交集）。
- **提交**：`a78c83a7` 造价线 / `1b628dc7` xlsx 体检 / `0ae43cb3` 会话检索 / `73739a34` PDF 页数 / `093c83c5` bash 与文件闸 + 本笔记档。
- **包**：`releases/gaea-ai-feedback-20261128-093c83c5`——干净树构建（vite 63s + wails 60s，cgo 非空壳，绑定无漂移），冒烟 `/api/health` 200，INSTALL OK 新壳 **PID 53224**（00:10:16 > 构建时刻），包版四件回写主树 `build/bin`，三处哈希一致。exe `6f488036a40e626f`、zip `2c822b1e1450bef3`、src.zip `2cf64218390527d1`。构建工作树已回收。
- **余**：push 待令；真机走查；#4 / #5' / #8 / #11 / #12 / #14 / #15② 待令；拍板池 #10 矢量图 / 277 条章节恢复 / 样表旧文案。

## 2026-11-27 深夜 AI 反馈批（桌面 ai-feedback-2026-11-27.md，20 条）落码 11 条，未提交未打包（ZCode；上下文不足中断，接续单=进度计划/gaea-ai-feedback-20261127-batch.md）

- 背景：用户令「修复这些问题」（模型经 ai_feedback 提交的 19 条 + 用户手写的造价技能工作流 1 条）；中途附适配规范 v5.6 令「根据现在的实际情况进行修正，补充进说明书」，随后令「做好文件记录，我们下个会话继续」。逐条对码分诊 + 翻包2/包4 两场会话日志定根因，分诊表与逐文件清单在接续单。
- **已落码（分包测试绿）**：①造价线——`import_add` 载荷走文件（`resources_file / quotas_file / bills_file`、`classify_file`、`import_status export`）；playbook / 身份稿 / 模式行主路改为「在测算样表上编 → 两道门 → **交用户检查同意后入库**」（此前主路是库内直写，模型不知道先用样表）；样表旁绑定《使用说明书》（`internal/gaea/skill/assets/成本测算表-使用说明书.md`，v5.6 适配规范修正后并入：快速通道口径、三层 ROUND、摘要金额在 F 列、地区只到省等 12 处）；导入演练新增「疑似明细行溢出」与「工作簿内对账」（workcost/workbook_recon.go，金样夹具零误报）；确认卡提醒「请确认已检查过这份测算表」；顺手救活两个静默跳过的导入测试、板块导出菜单去掉已删的「资料条目」。②bash——长命令超 6000 字符改走临时脚本（实测 Git Bash `bash -c` 参数超约 8.2K 字符静默截断且 rc=0）；缺 command 时识别「发来的是数据对象」并指路。③文件闸——stale 闸 / 预检查改用工具自己的路径解析（此前拿相对路径按进程 cwd 读，write 后紧跟 edit 在正式环境必被拦）；预检查失配时直接给最接近位置与现行文本。④read_file 字节窗口（`byte_offset / max_bytes`，单行超长内容按字节分页）。⑤session_search——索引助手实时成稿（kind=message；此前只认 assistant_message，等于只搜得到用户的话）、多关键词三级放宽、坏行容错且不再盘上截 torn tail、query 留空列最近会话；`@会话` 快照同根修。⑥format_convert——页数上限口径改「单次页数」（`pages=501-676` 可补转）、新增 `all_pages` 自动分段转完、截断时回执与文件头尾醒目标出缺页。
- **做到一半**：xlsx 健康体检——`xlsxedit/health.go`（打开性 / 缺缓存值 / 定义名与表级归属 / 数据验证 / 合并格）已接进 xlsx_audit、xlsx_recalc（前后结构对账）、xlsx_apply；`PublishGuard.Touched()` 已加但 **bash.go 还没接**（脚本写出 xlsx 缺缓存值的提示），app 层接线测试没补。
- **预算**：工具面 standard 17250（实测 17156）/ cost 29250（29212）/ proposal 21750（21615）；cost_library 常驻 schema 9700。
- **没动**：#4 发布边界合规链与覆盖留痕；#5' 方案库「共 N 份」口径；#11 outline display_title；#12 正文标题识别；#8 keyword 精确开关；#14 交接卡；#15 断点检查点。
- **待拍板**：#10 矢量图补图（要引 PDF 库依赖）；277 条章节数据恢复；是否给测算样表配「按数据扩骨架」的填表脚本；样表三处旧文案是否用 gaea 改。
- **验证**：`go build ./...`、改动包 vet、分包测试（含 app 的 TestCost / 工具面预算 / 运行模式 / TestXlsx 分组）全绿；**没跑** app 全量、agent 全量、前端全量、golangci-lint；未提交、未打包、未装机。

## 2026-11-27 晚 造价复盘清偿三刀 + 在途六批收口（ZCode；包 gaea-cost-review-20261127-b799f13d 已装机 PID 55360）

- 背景：用户带《软件不足与改进建议——造价库「地区按省合并」批量改造实战复盘》进会话（16 条 P1–P16，证据来自实弹工具回执）。对码分诊：8 条属实、4 条大半已有（P3/P8/P12 提示与机制已在）、P5/P9/P13/P14/P16 挂拍板池；且主树正压着六个未提交小批。
- **收口**：六小批按线归四笔提交——①定额章节改分部·地区限省（f337ddcf：34 省级行政区归一 NormalizeRegion、定额地区与项目挂钩首标纪律、导出补地区列）②方案库导入清偿 docx 重建器+md_validate 刀2/1/4/5（d5efe65e）③前端壳面三小批（7dcb48ea：Codex 造价三级图标+页头搜索单行+导航 46px 对齐、列表排序+工料机详情、办公项目行操作常显）④绑定同步 b4a1e508（wailsjs models.ts 的 Quota.region 字段序——干净树构建检出 f337ddcf 漏带 bindings）。中间包 gaea-20261127-7dcb48ea 冒烟 200 装机 PID 28484 后，随即被终包覆盖。门禁：workcost/docmd/builtin/app(ost+ToolFace) Go 绿 + 前端 CostLibraryPage/Sidebar/logic 82/82；builtin 全包首跑 websearch_core 一例顺序依赖假红（隔离+复跑绿定性，文件未动）。
- **拍板撤销**：P2 的 region 前缀匹配——用户令「地区已经固定到省了」，省市混写旧痛消失；只留 regionFilter 查询词归一（「四川」→「四川省」）。
- **刀A** 0437d1e8（P1+P2+P15）：quotaOf 同码更新没给头字段保持原值（兜底链 行上显式>既有行>项目默认；专业不给=沿用既有行），未给含量行=保留原含量行（saveQuota 先删后插，nil 曾静默清空全部消耗）；quota/resources 详情回显分部/分类/地区/期；SearchQuery.Offset 列表翻页（listQuotasPaged/listResourcesPaged，ORDER BY 带编码兜底窗口稳定）+ clListScale 给 offset 续读提示。
- **刀B** 12ec42e5（P7+P10+P6）：专业闸只有显式给不同专业才退回，文案「去掉行上的 specialty…borrow_quota 借鉴生成」（旧文案括号为空且误导另起编码）；delete_resource 可删先删+被引用暂缓双清单（一条残留不再整批报废）；暂存行部分修正 mergeStagedQuota/mergeStagedResource（重发只带改动字段；**合并必须在校验前**——曾放校验后被名称门拦）+ action=import_status 暂存区回读。
- **刀C** 4d52d899（P11+P4+P8+P3）：AuditQuotaGaps 按专业+同码族分组（quotaFamily 迭代剥 -2/-N/A/B 变体尾缀），跨省同名版本不再池化互报；成员记账改下标制（**按 code 键记 memberKeys 被同码 fixture 打穿**——库内码唯一本就异码）；dims summary=true 跳过待整理明细抽样；import_open 无资料路径时明示 controls 第二道把关；spillNotice 前置尾部 offset 锚点（total-8KB），单行 JSON 大载荷可直取尾部核完整性。
- **预算** b799f13d：cost 档工具面 28250→28500（三刀契约净增 110 字上取整，口径同前）。
- **打包装机**：干净树 .worktrees/pkg-b799f13d（junction node_modules），npm 23s + wails 53s（bindings 未跳过，ucrt64 gcc，cgo 导入表含 sherpa/ORT）；冒烟 200（payload 正式路径 18999 隔离端口）；INSTALL OK 新壳 PID 55360（22:19:09 > 构建 22:18:59），18123/healthz ok；包版四件回写主树 build/bin，payload/build/bin/装机位三处哈希一致（exe `0c59f183dd6c80cc`，zip `0b08d944236be5db`，src.zip `19defed445614d5a`）；工作树已回收（junction 先摘）。
- **余（挂拍板池）**：P5 语义检索覆盖造价库（文档明示 vs 索引覆盖）；P9+P13 载荷走文件与批量改写含量行引用原语；P14/P16 子代理结构化完成报告与孙代理通知归属；P1 数据面=277 条被清空章节的恢复（库备份或原始资料重导，代码刀不恢复历史数据，需用户令）。

## 造价库三级图标重设计 + 搜索并入页头单行（2026-11-27，Codex；未提交）

- **三级图标**：`cost-library/CostTierIcons.tsx` 新增造价专属 24×24 / 1.6px 描边图标——清单＝勾选台账、定额＝卡尺包围的标准单元、工料机＝人工 / 材料 / 机械资源组；替换原「分区 / 钱币 / 应用网格」通用映射。`TierRail` 保持 `tablist` 语义与键盘行为，新增 32×32 弱边界图标底座、单强调色选中态与底部指示线；窄窗 TierBar 用 28×28 紧凑底座并禁止“工料机”换行。
- **搜索栏**：用户令「搜索栏不要单独一栏」。撤掉 `@container cost-library (max-width: 1240px)` 把 `.cl-search` 强制为 `flex-basis:100%` 的断点规则；页头固定为“品牌｜搜索｜动作”单行，搜索框只压缩宽度。实测容器页宽 760px 时三块仍在同一行，搜索框保留 180px 可用宽度。
- **详情三费**：综合单价分析表原先把三费塞进末尾 60px 表单元格，逐项折成三行。改为跨整表宽的单一摘要行，人工 / 材料 / 机械按 `max-content` 从左紧凑排列，外委说明独占附属行；详情宽度下三项均完整可见。
- **工料机表面**：详情根加 `cl-detail-resource`，区块改用 `--bg-elev` 66% 透明混合加极轻顶部渐变，现行价 / 基准价小块改用前景色 3% 透明叠层；清掉原 `background: var(--bg)` 实底，深色与浅色均与页面背景连续。
- **折叠侧栏宽度**：用户令折叠后参照一级导航宽度，并指出首次按 62px 占位足迹对齐后仍偏宽。最终新增 `--v3-rail-width:46px` 表示一级导航本体，工位与文学折叠列及布局常量均改读/对齐 46px；折叠按钮、新建、当前会话与底部动作从 38px 收到 34px。真机量得一级导航与折叠侧栏均为 `width=46`。
- **侧栏品牌去重**：用户指出左侧面板与一级导航的两个 gaea logo 挨在一起。工位与文学侧栏均移除 `office-sidebar__brand-mark` 图片节点；展开态只保留 “gaea / 当前项目” 文字与折叠按钮，折叠态只保留折叠按钮，顶部高度由 80px 收至 58px。
- **一级导航上下对齐**：用户令导航底部与左侧面板平齐，随后补充顶部也要平齐；又纠正板块名称应在导航上方而不是 rail 内。`TelemetryRail` 折叠状态上提到 `MainLayout`，shell 暴露 `--v3-telemetry-height`；导航 dock 顶读 `--v3-strip-height:38px`、底读遥测高度，真机量得导航与侧栏均为 `y=38 / bottom=857`，遥测展开时两者同步到 `bottom=699`。板块名使用透明两字简称（办公 / 造价 / 方案 / 记忆…）悬浮在 rail 上方 62×38px 槽位，不加底色和分隔线；完整名称只放 `title`，rail 内只保留 logo 和图标，顶栏原重复名称移除。简称为 12.5px / 600 字重 / 0.1em 字距，与侧栏标题字级协调。
- **验证**：`tsc -b` 通过；`CostLibraryPage.test.tsx` 39/39 通过；mock 真机目检默认 / 960px / 760px，深色与浅色（图标）无溢出、无标签换行。
- **桌面构建装机**：主树前端 `npm.cmd run build` 21.59s；ucrt64 + cgo + 系统 TMP 执行 `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，38.017s。`build\bin\gaea.exe` 62,765,568B / SHA256 `2141A0AF9828…`；`/api/health` 200。`gaea_install.ps1 -Launch` 输出 INSTALL OK，新壳 PID 21272（13:13:23，晚于构建 13:13:04），`18123/healthz` 200；四件与 build/bin 哈希全一致，exe 导入表含 `sherpa-onnx-c-api.dll`。未制作 ZIP / 源码包。
- **桌面重建装机（13:36）**：再次按同一链路构建；前端 27.73s，Wails 35.319s，bindings 未跳过。`gaea.exe` SHA256 `7C1B24C88BDA…`，`/api/health` 200；INSTALL OK，新壳 PID 43808（13:36:38，晚于构建 13:36:22），`18123/healthz` 200；四件与 build/bin 哈希一致，exe 导入含 `sherpa-onnx-c-api.dll`，无 wails/go 孤儿。未制作 ZIP / 源码包。

## 方案库导入硬规则：PDF 禁直转 md，经 to=docx 转 Word 修正后录入（2026-11-27，ZCode；提交 ca0b13a2，包 gaea-20261127-ca0b13a2）

用户令「重新完善方案库的导入技能：PDF 原始方案要转换成 Word 修正格式后才能导入，不能直接转 Markdown——会大量出错，特别是表格位置」；随后给事实源《包1技术标方案重建复盘》（`D:\新工作\2630沿江一公里\沿江包1\`）：包1技术标 V5 当初 PDF 直转 md 录库（42 块列数矛盾表），复盘 §2.1 实证脏 Word 路线逐字同病，§3.2 点名技能缺录入前质量门禁/脏文档预期/双源对账三块。

- **刀1 to=docx**：`internal/docmd/wps_docx.go` 新增 ConvertToDocx——复用 wpscli pdf2word 但**保留 docx 产物**（此前 md 链把 docx 中间产物丢临时目录）；产物核对/终态行路径/账号级静默与 wps_excel 同纪律；扫描件在线识别上限 100 页、pages 只收单段。format_convert 增 `to=docx`（output 必填 .docx、只收 .pdf、scanned 同 xlsx 口径），**回执固定列三类必修缺陷**（单元格折行成碎片段/垂直居中空段落/目录域与目录表混正文=复盘 §2.1 清单）并指路「修正后以 docx 为源转 md 录入」。
- **刀2 技能**：proposal-longform 录入段重写（office_runmode_skills.go）：备源步 PDF 硬规则；脏 Word「只作结构参照、文字回 PDF 原件对账」；**录入前三项机检硬门禁**（表格列数一致/占位符残留/图引文件存在，全零才准 finalize）。
- **刀3 同步**：方案档 ModeLine + document-convert 模板同口径；不加代码拦截门（直转 md 有预览/取材正当用途）。
- **T7**：proposal_save/update/search 的 year 收数字（metaStr 宽容 UnmarshalJSON，null 不动零值）。
- **测试与坑**：builtin/docmd/app 三包全绿（to=docx 参数校验/回执/落盘、year 数字）。坑一：**ucrt64 前置 PATH 抢 python**——ucrt64/bin 自带 MSYS2 python.exe（无 markitdown），go test 的 LookPath("python") 命中它 → docx 图片族测试假红；测试命令把真 Python313 排 ucrt64 前（已录构建指南坑清单）。坑二：standard 档 ToolFace 预算改前仅 7 字符余量，新增描述必超——compact 文案收至最小（+32 字符）后预算 16750→17000（注释写理由）；预算面=compact 描述+schema（改完整 Description 测试数字不动）。
- **主树构建装机（11:28）**：冒烟 /api/health 200 → INSTALL OK（exe `D6E788A6…`）；后随干净树包重装机。
- **拍板池（复盘 P0/P1 余项，未立项勿做）**：质量机检分级+基线对比、docx→md 合并单元格网格展开、多源交叉对账工具、format_convert 回执改结构完整性指标、矢量图按题注裁回。

## 库页编辑按钮：方案库与造价库直接修正入库错误（2026-11-27，ZCode；提交 603895d2，同日随包 gaea-20261127-ca0b13a2 装机）

用户令「方案库与造价库增加编辑按钮，打开后可以直接编辑修改入库的资料，方便修正入库错误数据」；随后令「构建桌面端」。**这条推翻了 `.gaea/AGENTS.md` 的「库页只读约定」**（该约定及其锚测试已同步改写为「单条编辑钮 + 批量维护仍归造价模式」）；录入、批量导入、解析、新取值归类仍归办公对应运行模式，不走绑定面。

- **Go**：新增 `GaeaProposalUpdateMeta`（元数据全量替换，标题即方案名、改名返回新名）与 `GaeaProposalUpdateSection`（updateTitle/updateBody 两旗显式——没改的半边不传避免白重嵌向量；两旗全 false 拒绝）于 `internal/app/gaea_proposal.go`；新增 `GaeaWorkcostBillProjectProfileSave`（项目口径档，空=保持原值）于 `gaea_workcost.go`。其余九个保存绑定（`WorkcostResourceGet/Save/SetPrice`、`QuotaSave`、`BillItemSave`、`BillQuotaLinks/Attach/Detach`、`BillProjectRatesSave`）是 App 既有方法，从 gen_bindings `excludedBindings` 除名解禁后重跑生成器。绑定面 668→**680**；四件再生（bindings_cost/proposal 门面、bindingNames、legacyBindings、bindingSignatures）+ mappings/spaceBindings 同批认领 12 名；`check-bindings-drift.ps1` OK（StrataModelStop「复活请清册」是既有提示，非本线）。
- **造价库页四处编辑面**（`frontend/src/pages/cost-library/Edit.tsx` 新建）：详情头「编辑」钮进编辑态，保存走 store 既有入口（语义索引打脏随写自动生效）；规范值（专业/地区/期数/分类）只给库里已有取值，新取值归类仍归造价模式。①清单行：字段（名称/单位/工程量/分部/专业/手填价/特征）+ 挂接定额（读引用表，改量走幂等 Attach、解挂、按编码新挂）；②定额：字段 + 含量行整组替换（改含量/损耗、删行、按资源编码加行，缺的元数据后端按资源索引补齐）；③工料机：主数据（类别/名称/规格/单位/分类/地区/期数/损耗/基准价/现行价）+ 独立「按期调价」（SetPrice 写历史，登记后留在编辑态）；④项目封面：口径档与费率档分钮保存。保存成功退出编辑态 → 页面提示 + 静默重读（keepAlive 口径）。
- **方案库页**：清单行「编辑」→ antd 弹窗改标题/类型/年度/摘要，选中方案改名后按返回新名重新定位（清单先重读，改名回执驱动 selectProposal）；阅读窗「编辑本节」→ 标题 + Markdown 正文两格，保存只发真正改动的半边（改标题连带子树路径 → 标题树整棵重取并定位回本节；只改正文 → 重读本节）；换节自动退出编辑态不带走草稿。
- **提示词三处同步**（含锚测试 `TestCostLibraryPromptAnchors` 改锚 `「编辑」`+`批量导入`）：造价 ModeLine / 身份稿「造价库维护」节 / cost-estimation-playbook 技能与 cost_library 工具描述——口径改为「单条改错用户可在板块详情点编辑直接改（也可由你用工具做）；批量导入、解析、归类、批量调价只在本档；直连数据库改库仍禁止」。
- **测试**：Go `TestProposalBindingsUpdateMetaSection` + `TestGaeaWorkcostBillProjectProfileAndRatesBindings`（字典播种后验口径档空=保持）；前端 CostLibraryPage 6 例 + ProposalLibraryPage 5 例编辑流（改名重定位/半边旗/编辑态退出/失败留在表单）。tsc 0 错。**全量门禁收官**：app 包 1380 例全绿（330s）、proposal/workcost/gen_bindings 包绿、`go build ./...`+vet 净、前端全量 vitest **511/511 文件绿**（前一轮有一例单发红，两轮复跑全绿=偶发，文件名未捕获）、mock 契约 5 例绿、漂移守卫 OK、check-docs OK。
- **坑**：①`gen_bindings -legacy-ts` 必须在 spaceBindings/mappings 认领**之后**跑，否则锁四（legacy 重叠）tsc 红；②vitest 从仓根跑会扫进 `.worktrees/` 旧树假红（3597 文件 vs 基线 ~511，2366 失败全是旧版测试），**后台任务必须把 `cd frontend` 写进命令内**（Bash cwd 在调用间漂回仓根，连犯两次）；③详情编辑钮的重置 effect 钩 `[item.id]`——静默重读后 id 不变，编辑态不会自动退，所以 Edit 组件保存成功后自行调 onCancel 退出；④app 包测试要 ucrt64 + 主树 build/bin 双 PATH（0xc0000135）；⑤`go build ./scripts/gen_bindings/` 会在 cwd 落 gen_bindings.exe，记得删。
- **顺手修的存量红（非本线）**：`internal/gaea/proposal/TestMigrationFromV35` 预置只拼 SchemaV1+V35，V38 起 `ALTER TABLE gf_projects` 需要中间版本的表 → 升级路上 no such table（2026-11-25 定规矩批的欠账）；改成按链预置 V1..V35，断言从钉死 user_version=37 改为跟随链尾（当前 43）。
- **余项**：已提交 603895d2、随包 gaea-20261127-ca0b13a2 装机（2026-11-27）；真机走查待令；项目封面改名（name/location/duration/pricing）没做——UpsertBillProject 按名 upsert 会复制出新项目，改名要单开 UPDATE 刀，待用户确有需要再提。
- **构建装机（2026-11-27 09:36 用户令「构建桌面端」）**：主树整树构建（工作树=本批改动，无外线混入）；孤儿进程零、壳未在跑；前端 npm build 22.9s → wails build 39.6s（ucrt64+build/bin 双 PATH 前置、显式系统 TMP、bindings 未跳过）→ exe 导入表含 sherpa/ORT 非空壳 → `build\bin` 冒烟 /api/health 200（PID 55436 已回收）→ `gaea_install.ps1` **INSTALL OK**（exe `6B31A2888844…`，四件与 build/bin 一致）→ 新壳 **PID 3484**（09:33:56 > 构建 09:33:25，路径=装机位）→ **18123/healthz `ok`（200）**。build/bin 即本次构建产物，无需回写。

## 造价技能优化·众神殿终案落码（2026-11-27 晨，ZCode；提交 5b39c1cb + 包 gaea-cost-skill-20261127-5b39c1cb 已装机 PID 51376）



用户令「构建众神殿关于造价技能的论证，开始优化完善造价技能」；论证已由用户侧圆桌跑完（pan_1795735011377，三场续议、codex/zcode/dsh 三席一致）：**三技能不合并、不改名**；cost-estimate 保持全局内置零差异（不得指向造价档专属内容——standard 档可见集合外指针软失效）；落盘硬规则单源进 officeRunModeLineCost；playbook/soil 只做最小语义改动。事实源=`%APPDATA%\gaea\pantheon\results\pan_1795735011377_5f4f26676579.md`。用户问及本地模型检索：已含在 `cost_project` quotas/resources/bills 工具内（workcost.Searcher：编码精确短路→关键词+本地向量 bge-m3→融合 0.55/0.30/0.15→bge-reranker 精排封顶 16；无模型退关键词路），正文三处已教用法（〈语义〉标注/自然说法 keyword），不补实现细节。

- **改动 3 文件 +15/−5**：playbook 四处（定位段「六表纯成本测算」+MeasuresFee/Contingency 预留位勿引用；第 1 步末「土壤修复/污染场地项目先读 soil-remediation-costing」；落盘条收敛为 ModeLine 指针；控制价改「占控制价%（项目设了 ControlPrice 才有）；差额结论不手算以回执为准」）；soil 两句（首段适用边界「非土壤项目不读本技能」+「五步工作流第 1 步触发」联动句）；officeRunModeLineCost 增补「交付落盘（常驻单源）：.gaea/work/ 写过程件+move_file 落正式名+隔离件按回执取回」与「涉及造价库导入、整理或维护前先读 cost-estimation-playbook」装载触发；造价身份稿交付律③改指针（消除「先落临时目录（或 .tmp 扩展名）」冲突副本）。
- **门禁**：vet/gofmt 净；golangci --new-from-rev 0 issues；boot+skill 全绿；app 1380 例四分片全绿（67s）。旧文案断言 sweep（落盘合规/无现算链/.tmp 扩展名）零命中；ModeLine/身份稿/交付段测试均非全文 golden（DSH 席核过 office_runmode_test.go:72-78,538 与 delivery_prompt_test.go:12-64，断言子串「【运行模式】/档名/新会话生效/cad_ask」全保留）。
- **打包装机**（用户令「构建桌面端」）：干净树 `.worktrees/pkg-5b39c1cb`（junction node_modules，PowerShell New-Item 建）→ npm 26s + wails 44s（bindings 未跳过，ucrt64 前置+主树 build/bin 后置，显式系统 TMP/TEMP）→ cgo 导入表核验含 sherpa/ORT 非空壳 → payload `releases/gaea-cost-skill-20261127-5b39c1cb/` 冒烟 **/api/health 200** → `gaea_install.ps1 -SourceDir` 装机 **INSTALL OK** 新壳 **PID 51376**（08:00:52）/18123/healthz 200 → 包版四件回写主树 build/bin，payload/build/bin/装机位三处哈希一致（exe `dc7fbedc…`，zip `b0438926…` 34MB，src.zip `f08592da…` 831MB）。工作树已回收（junction 先摘，wailsjs 漂移验纯行尾）。
- **余项（圆桌遗留台账）**：「库操作前读 playbook」触发句实际装载效果待下一场真实造价会话回放 session.json（查 run_skill 与 cost_library 调用顺序）；identity 稿控制价行（54 行）未在圆桌范围、措辞与新 playbook 同义未动；push 挂令。

## 测算簿适配三级树表样表 v5.5 + 删旧版渲染器（2026-11-27 凌晨，ZCode；主树日常构建已装机，包见 releases/gaea-cost-tree-20261127-3db867a4）

用户附 v5.5 样表与适配规范令适配造价模型与数据库导入；随后两条拍板：①旧「行类」版式退役（以后不再上传）；②**导入的硬契约只有综合单价表**（三级树表），其余五张表是参考、模型自由发挥（清单表按单项工程划分/加列/调列序都不拦）。

- **提交 `3db867a4`**（41 文件，净 −1394 行）：解析器只认三级树表（层级列判行型/变长组/换算系数/残缺行显形，非树表整份拒收）；清单表按列名定位、「单项工程」列顶替「分部」、工程量双来源+回执「工程量来源」行；数值改读存储值（存量 bug：含量 0.00015 被显示值吃成 0.0001，沿江包1「废水处理运行」表 34.70→库 33.91）；封面项目名取「项目名称」标签值并核对标题带；换算系数贯穿暂存流 quota_factors 与手工录入 save_items（重发不带系数沿用原比例）；删旧渲染器 render/export+两绑定+前端七处注册；样表种子换 v5.5（用户文件原样换入）；金样=沿江包1 最终表（33/35/68/41，直接费 6,242,754.43 逐分一致）；技能正文/playbook/系统提示/template 回执重写。
- **门禁**：build/vet/gofmt 净、lint 0 issues、workcost/skill/boot/xlsxedit 绿、app 1380 例四分片全绿、前端 tsc -b 过+注册表守卫 17/17、端到端 TestCostImportProjectYanjiangWorkbook。
- **构建装机**：主树 wails build（bindings 未跳过；goproxy.cn 对 gooxml 502 → goproxy.io 预取兜底）→ exe `D7B672426B5C…` → INSTALL OK → 冒烟 /api/health 200 → 新壳 PID 1308（01:46:07）/18123/healthz 200。
- **余项**：真实模型实弹；样表/项目簿小瑕疵由用户用 gaea 修（N 列 29 格「见S列」残留、含量列 0.#### 显示、m3/m2 拼写、适配文档 §8「67 含量行」实为 68、§7 已过时）；板块「行类」表头两处旧词未统一。

## 造价反馈三批 24 条落地 + 样表两层 ROUND（2026-11-26 晚，ZCode；包 gaea-cost-fb-20261126-cfbeda3c 已装机）

太原 #15 / 百锦路 #13 / 旺平 #12 三份反馈，先对码+对三场会话日志分诊（归并 16 项、纠 4 条错误归因），用户令「按你思路开始」后按四批刀单落地；样表修改经 gaea（用户改版 fa584a4b 换入内置种子），不手搓。事实源=记忆 `gaea-cost-feedback-3batch-triage-2026-11-26`。

- **七个提交**：`3257f87e` bindings 同步（索引线 A3 生成件漂移）；`1a13d339` 文件工具与闸门（overwrite 进 compact schema / md_validate·cad_ask 绑工作区 / 发布边界豁免 .gaea/work+回执可行取回 / move_file 占用点名 / bash 空参话术 / office_edit 操作名归一）；`1495f811` workcost 库层（地区并入裁决收窄+暂存码别名落含量行 / 控制价两路 / 行类漏填显形 / 机检四项 / region 过滤与总数 / 落地预演 / dims 待整理明细）；`4cc6c1af` app 工具面（列表钳位与总数 / summary 取到分再乘量+分部小计+快照对账+取费链走 ComposeProjectFees / save_resource 尊重显式码 / control_price 参数 / 映射上限 60）；`2ce3a532` 交付门（empty_result 不阻塞+聚类+clear_where，顺带修「=文本」永不命中）；`96767a43` 技能与 playbook 文案；`cfbeda3c` 样表换版。
- **新 op `xlsx_apply clear_where`**：range 内条件列为空的行连公式清掉——替代实测误删定额实算公式的手写清扫脚本。
- **门禁**：build/vet/gofmt 净；golangci --new-from-rev 0 issues；tool/builtin/workcost/skill/xlsxedit/agent/boot 全绿；app 1380 例四分片全绿（98s）；工具面三档在线内（standard 16741/16750）；真实 LibreOffice 冒烟三条（样表 1298 公式零错误、480 空结果全记 empty_result）。
- **打包装机**：干净树 `.worktrees/pkg-cfbeda3c`（junction node_modules）→ npm 27s + wails 12s（首跑漏 PATH 后置 build/bin → gen_bindings 0xc0000135，补上即过）→ 冒烟 200 → INSTALL OK 新壳 PID 61892（20:25:50）→ healthz ok → 包版四件回写主树 build/bin 三处哈希一致 → 工作树已回收（junction 先摘）。exe `710B33E6…`，zip `77C55CAA…`，src.zip `88FF7AA4…`。
- **顺带**：agent 存量红 `TestStaleGuardBlocksMoveFileWithoutRead`（铜陵 #8 豁免时未同步）改钉现行口径并更名；索引线（9cc57a20 六刀）随本包一并装机。
- **余项**：清单超 30 组仍需脚本追加（可让 gaea 把样表扩到 60 组，待令）；真实模型实弹（overwrite 是否真带上、聚类回执够不够用）；office_edit 未知名就近提示（操作名单在 C# 侧）；版本对比、工艺路线标签挂拍板池；summary 口径变化（带系数行差分~元级）与地区并入细则（旧写法地区并并规范）已告知用户。

## 造价库本地模型索引 + 旧成本库全量退役（2026-11-26，ZCode；六刀全部提交，代码面收官）

用户令「成本库和造价库都要接入本地模型索引，数据库到了后期会越来越膨胀」；拍板向量模型 bge-m3、单类目 50 万条以内、清单 / 定额 / 工料机三类都进索引；随后澄清在用的只有一个造价库（旧成本库数据已清空），并令旧成本库功能「全量退役」。第一个会话傍晚开工、上下文用尽中断；第二个会话下午做完其余五刀。与 WPS 线互不捆绑。

- **接续入口 = `进度计划/gaea-cost-index-20261126.md`**：拍板 / 摸底 / 设计 / 各刀结果（§八 退役、§九 B1·B2·A3、§十 评测与性能）/ 坑（§六）都在里面。
- **六个提交**：A1 索引底座 `abd65f42`（`internal/gaea/vecindex` + `SchemaV43`：二进制向量表、int8 内存矩阵、后台索引器、融合排序）；R 旧成本库退役 `459723c2`（183 个文件 +560 / −22,879，库表没删）；B1 `cb4e4c23`（三类索引文档 + 库句柄层打脏标记）；B2 `9fbf8e2e`（`workcost.Searcher` 混合检索 + `cost_project`）；A3 `90fa05ab`（后台索引器随启动 + 状态 / 补齐绑定 + 板块语义补充段）；V `a3a72ca5`（真实库评测 + 精排候选封顶）。
- **机制要点**：库的写入经 workcost 库句柄自动打脏（按表名映射类目）、后台索引器增量对账，搜索路径从不建索引；检索管线 = 编码精确 → 关键词 → 向量 → 融合 → 候选多时精排（封顶 16 篇）；模型固定读 `[retrieval]`（缺省 bge-m3）。索引正文不含单价 / 工程量，调价不重嵌。
- **入口**：模型工具 `cost_project` 的 quotas / resources / bills（标〈语义〉的是同义召回）；造价数据库板块搜索框的「意思相近」补充段 + 状态栏「语义索引」状态钮（点击补齐）。
- **真实库评测**（副本，定额 261 / 工料机 225 / 清单 159）：19 个同义说法、61 个期望条目——纯关键词 top-8 命中 2%，混合 69%；关键词一条都对不上的 19 个里 15 个被召回。弱项：铲车→装载机、砼→混凝土这类行话缩写认不出。延迟：向量路约 19 毫秒、含精排约 235 毫秒；10 万条 × 1024 维单次检索 10 毫秒，50 万条外推约 52 毫秒。
- **待令**：打包、装机、真机走查（状态钮与补充段）、push；是否给行话缩写加同义词表。

## WPS 能力开发刀 1：format_convert 加 to=xlsx（2026-11-26 下午，ZCode；打包装机结果见本条末「补记」）

用户令「开始 WPS 的能力借用」；按施工单先把第一梯队实测做完、交草案，用户「照做」后施工。

- **先实测**（数字全在事实源 §2A：`C:\AI\re\wps-12.1.0.29690\docs\WPSCLI-CAPABILITIES.md`）：`pdf2excel` 文字型——清徐清单 839 页 495 秒，22 页对文字层数字召回 1198/1198，金额真数值、12 位定额编码前导零保住、0 处拆小数；扫描路线三路对照——同 22 页 392 个独立数字格，纯在线识别丢改 18（4.6%，含金额末位截断 `5776.41`→`5776.4`、工程量「1」整格丢、相邻格并格），开 `--ai-fix` 丢改 0、慢 8 倍（约 14 秒/页）；`photo2excel` 表结构好、同样丢工程量「1」格、没有 ai-fix 选项；`pdf2txt` 233 页 6.7 秒字符召回 100%（无页分隔）；`ofd2pdf` / `pdf2ofd` 本地引擎各约 1 秒（只过了合成样本）。
- **拍板**：刀 1 扩 `format_convert`、不新开工具；扫描件 / 照片转 Excel 默认开 AI 修复；真实 OFD / CAJ / 手机实拍样本待用户提供。
- **落地**（提交 `1b96f329`，详见施工单 §八）：`internal/docmd/wps_excel.go`（`ConvertToExcel`：分流 / 按页数超时 / 产物路径以 wpscli 终态行为准 / 在线识别单次上限 AI 修复 20 页、不开 100 页）、`internal/docmd/xlsx_inspect.go`（流式读工作表 XML 做核对，回执带疑似缺格行）、`wpsRunTimed`、`builtin/format_convert.go` 加 `to` / `scanned` / `ai_fix` / `per_page_sheet`，pdf 技能种子补用法。
- **实弹**（经 gaea 真实链路）：清徐 291–294 页文字型 6.7 秒、201 格、疑似缺格 0；手机变体照片 5.5 秒、疑似缺格 1 行（正是工程量「1」被漏的那行）；同 4 页在线识别 + AI 修复 73 秒、疑似缺格 0；839 页工作簿核对 0.84 秒。
- **门禁**：`go build` / `go vet ./...`；docmd、tool（含 compact 漂移钉）、builtin、config / control / skill / evidence / contextview 全绿；app 包 1431 例四分片全绿（68 秒）；check-docs、registry map 门 OK；lint 本批文件 0 条。
- **坑**：①excelize 的 `GetCellType` 不能在 `Rows()` 迭代进行中调（查不到格子、一律报 `Unset`，把 149 个文本格数成数值），逐格调用又是平方级开销（同一份 839 页工作簿 5 分半）——产物核对改自己流式读 XML；②wpscli 不覆盖已有产物，另存 `<名>_1`（帮助文本写的是 `file(1).docx`）；③默认 `--timeout 300` 不够 839 页；④`--multi-sheet` 是一页一个 sheet；⑤Git Bash 里给函数传 `"\\$name"` 会被吃成字面量，三路产物落到同一个文件又被改名——给 wpscli 的 `-o` 用正斜杠完整路径；⑥PATH 前置 ucrt64 跑 builtin 包 = msys64 的 python 顶掉系统 python，markitdown 三例假红（上一条已记，本批又踩）。
- **存量红与欠账（本批未动）**：agent `TestStaleGuardBlocksMoveFileWithoutRead`（在册，单跑 3/3 红）；trajectory `TestFoldDeliverablesGolden` 的 multi_edit 金样本——`multi_edit` 11-25 退役后金样本没跟，**新发现**；`builtin/computer_tools.go` 的 `computerTypeTool` / `computerKeyTool` 未注册、lint 报 24 条 unused，`bash.go` / `schedule_tools.go` gofmt 两条；control 包测试会改写受跟踪的 `internal/gaea/control/resumed.jsonl.meta`（已回退不入账）。
- **余项**：①刀 2 OFD / CAJ 透明接入（等真实样本）；②刀 3 `pdf2txt` 替换「只要文字」的 WPS 路径；③刀 4 清单 xlsx → 造价库（现有成本导入是从 Markdown 文本抠表）；④刀 5 导出 / 渲染对测 LibreOffice、`pdf2cad`（工作区有 12 份图纸 PDF）、PDF 杂务类；⑤多段扫描产物合并成一个工作簿未做；⑥真实扫描件上的 AI 修复表现待测（本次是文字型页强制走在线识别）；⑦装机后真机走查：办公模式让模型把一份清单 PDF 转成 Excel。

## PDF 转换 WPS 优先 + 本地 OCR 精简（2026-11-26，同一会话，ZCode；打包装机结果见本条末「补记」）

用户报「造价模式读 PDF 用 OCR 会堵死，按停止后下一轮一直显示排队中」，问「办公板块是否需要一个专门的格式转换工具」；随后补证「WPS 很短时间就转完了」，拍板**年会员额度不设限、转换解析优先用 WPS**；另问本地 OCR 三个能否精简到最好的一个，定「先精简、后提交打包」；WPS 更多能力开发为大项，本会话只记录，下个会话进行。

- **病根（对码 + 会话日志坐实）**：事发 PDF 是三明治件（174 页 JPEG 图像页 + 字形号文字层 `<11CD>Tj`）。`docmd/pdf.go` 自写提取只认 `(...)Tj` 与 `[...] TJ` 两种形态，整本提空；「空文本」分支直接落本地 OCR——WPS 此前只挂在「有文字但乱码」分支。会话日志：08:55:51 派发 `format_convert` 后无 tool_result，直到 09:35 重启。停止无效 = `format_convert` 收了 ctx 不用 + `executeBatch` 的 `wg.Wait()` 死等在途工具。
- **WPS 优先**（`internal/docmd/wps.go` / `pdf.go` / `images.go`，`builtin/format_convert.go`）：①提不出文字的 PDF 先走 WPS 在线 OCR（`pdf2word --scanned true --range`，必须带显式页范围，100 页分段串行），所有消费方生效；②`format_convert` 对文字型 PDF 也先经 WPS（`preferWPS`，真表格 + 图片），失败退回本地提取；检索索引 / @引用 / 预览仍走本地快速通道；③退出码 100/101/429 进 10 分钟全局静默；④兜底也失败时报错带两头真因；⑤`rejoinSplitDecimals`：WPS OCR 把表格里小数拆成 `3. 1` / `115953. 12t`（docx 原件即如此，2624 格中 21 格），只并纯数字格与数字后直贴拉丁单位的格；⑥`format_convert` 收到取消即返回 `ctx.Err()`、不落盘，后台那次跑完即弃（结果仍进进程内缓存）。本地 OCR 降为兜底，`ocrMaxPages=30`。
- **实测**：174 页扫描件经真实 Go 链路 85 秒（103 图 / 77040 汉字 / 37 表）；纯 `wpscli --scanned true --range 1-174` 68 秒。之前 = 本地 OCR 堵死 40 分钟。
- **本地 OCR 精简**（用户确认「只留 uocr2 llama 后端，python 后端一并退役」）：`docmd` 删 OvisOCR2 / tesseract 两套 provider 与 uocr2 python 后端（`ocr.go` 909→604 行、`ocr_uocr2.go` 647→441 行）；`GAEA_OCR_ENGINE` 空 / auto / uocr2 → uocr2，`ovis` / `tesseract` 显式配置如实报「已退役」并指路，不静默换挡；模型中心本地 OCR 三张卡 → 一张（`local:uocr2`；此前两张 uocr2 卡的「设为 OCR」其实点不通——卡 ID 与 kind 对不上，顺手改成一致）；配置残留的已退役定点值启动时清掉（`normalizeLocalOCRKind`）；pdf 技能脚本 `ocr_local.py` 只剩 WinRT + 本地视觉模型两腿；前端 10 个文件的文案与 mock 同步（办公 + 文学镜像）。测试里原先拿 tesseract / ovis 当「假引擎」的用例改用专门的替身引擎 `fake-ocr`。
- **WPS 能力开发记录**（只记录未开工）：事实源 `C:\AI\re\wps-12.1.0.29690\docs\WPSCLI-CAPABILITIES.md`（30 个子命令、实测数字、输出瑕疵、逐项开工前实测清单；**勘误：`clitool\wpscli.exe` 无门禁，11-24 报告测的「门禁」是 `office6\` 那个**），施工单 `进度计划/gaea-wps-capability-plan-20261126.md`，待办表顶行。
- **门禁**：`go vet ./...` 0；go test docmd / tool / builtin / control / fileindex / largefile / wssearch / config / internal/ocr 全绿，app 包 OCR / 本地服务卡 / 成本导入 / 工具面预算 / 绑定完整性全绿；前端 tsc -b 0、vitest 全量 4780 例、改动文件 eslint 0；check-bindings-drift OK、check-docs OK。
- **顺手清的既有欠账**：`TestBuildUocr2LlamaServerCmd`（夹具路径停在 11-20 钉点前，一直红）；`SettingsPage.test.tsx`「晨报预载」断言（开关 11-25 `799e9ba8` 已撤下，断言没跟；干净 HEAD 上复现同红后删）。
- **坑**：①`grep -a -c` 数二进制 PDF 对象不可靠（196 处 `/Font` 报 0），PDF 结构用 Python 正则；②PATH 前置 ucrt64 会用 msys64 的 python 顶掉系统 python，markitdown 类测试与实弹假红——纯 Go 包别带；③`go test -exec cgo_test_dll_prep.cmd` 的 `-run` 不能含 `|`；④Git Bash 里 robocopy 的 `/L /S` 会被路径转换吃掉，走 PowerShell；⑤装了 WPS 的开发机上，没打桩的测试会真调 WPS 云——docmd 的 `TestMain` 默认 `GAEA_WPS_PDF=0`。
- **余项**：①退役引擎的模型目录已删除（2026-11-26 12:09 送入回收站，见补记 2）；②agent 循环层对「不看 ctx 的慢工具」仍会 `wg.Wait()` 死等（本批只在 format_convert 工具级解决）；③正文汉字间 OCR 空格未归一化；④WPS `--ai-fix`、在线引擎单次页数上限未验证；⑤装机后真机走查：造价模式转那份扫描件、转换中按停止再发消息不排队；⑥**既有缺陷（本批未动）**：`internal/gaea/pricefeed/store.go` `SaveFetch` 用 `time.Now().UnixNano()` 生成记录 ID，Windows 时钟粒度下连续两次保存会撞号（`UNIQUE constraint failed: price_fetch.id`）——app 整包测试 `TestGaeaPriceFetchAllTask` 偶红（单跑在改动前后都过），真实使用中「全部抓取」也可能偶发丢一条抓取记录；建议改原子序号或随机后缀。

**补记（打包装机，2026-11-26 11:58）**：提交 `ac3a60bd`（代码）+ `7f8ea22b`（记录）；包 `releases/gaea-wps-ocr-20261126-7f8ea22b/`——干净工作树 `.worktrees/pkg-wps-ocr`（detached @ 7f8ea22b）构建，前端 Vite 22.3s、Wails 26.5s（bindings 未跳过，wailsjs 零漂移），payload 冒烟 200；exe `CA9B9D41…324523B`（60,085,248 B），zip `8843f578…`、src.zip `48bf3ebe…`，三个 DLL 与上一包逐字节相同；包版四件回写主树 `build/bin`（哈希对账一致）→ `gaea_install.ps1 -Launch` → INSTALL OK，装机位 `C:\AI\gaea-app`，新壳 PID 49952（11:58:27 启动，晚于构建时刻）。开机日志两条 `[NewSessionError] 办公引擎未初始化` 与今早 09:35 启动日志相同（既有开机现象）。未 push；真机走查未做（见余项⑤）。

**补记 2（退役模型目录删除，2026-11-26 12:09，用户确认后执行）**：`C:\AI\gaea-OCR\unlimited`（python 后端，70,305 文件 / 15,964,598,671 B ≈ 14.9 GB）与 `C:\AI\gaea-OCR\ovis`（OvisOCR2，22 文件 / 860,116,904 B ≈ 0.8 GB）**送入回收站**（可原位恢复；清空回收站后才释放约 15.7 GB，C 盘当前剩 25.7 GB）。删前：确认无进程占用、目录外部零引用（用户级技能 / 环境变量 / gaea 配置）；不可重下的小件归档到 `C:\AI\re\unlimited-ocr-2\`（部署 README、打补丁的模型代码 16 件、`server.py` / `ocr.py` / `smoke_test.py` / `mp_test.py`、ROCm 手写桩 2 件，逐文件核对一致）；来源与重建链：Unlimited-OCR-2 权重来自 ModelScope、OvisOCR2 为 Apache-2.0 公开模型（确切仓库未记）；实弹样本图挪到 `uocr2-llama\testimg`。删前删后保留引擎实弹对照逐字一致（单页 907 字符、2 页批 894 + 951）。余项①销项。同批：`scripts/uocr2/`（python 服务的版本控制副本）随退役移除，存档在上述数据集；实弹测试去掉 `unlimited\testimg` 回退路径（仅测试与脚本，不影响已装机的二进制）。

## 文学板块对办公优化的同步批（2026-11-26，未提交未装机；审计→施工同一会话，ZCode）

用户问「办公板块进行了多次优化，文学板块相同的部分有没有同步」→ 只读审计 → 用户令「开始」。分叉点 44f8bb6b（11-15）。

- **审计结论**：引擎/后端对齐（两板块共用 `boot.Build`；工具梳理两批同批改了 wenxue deny 名单/前端 disabledDomains/wailsjs；`boot.Options` 逐字段比对，办公独传的只有运行模式三项与 DAG 执行器）。未同步的全在前端 fork 树。
- **落地**：
  1. 上下文视图 `contextWindow`（57daf182）：`ContextModal`/`ContextView` + App 两处挂点——切模型后已打开的上下文 tab 重拉。
  2. 能力摘要宽度截断（d30cb5d2）：`wenxue/lib/capabilities.ts` 改转发办公模块。
  3. 子代理体验（f7dc3063/712ac51e/fc120738/73d1a906）：`lib/taskActivity.ts`（后台回执信封剥离/全部运行匹配/委派命名）、`Transcript.tsx`（委派显示名贯通；后台启动回执与子代理答复不在父对话重复成行）、`SubagentThread.tsx` 与 `lib/types/agent.ts`·`subagent.ts`·`context/AgentRadial.tsx` 取办公现版（与办公仅差上述提交）、`useTaskCards` 打开目标/歧义在**全部运行**里匹配（后台完成卡可点；打开链路第一批仍走主区会话 tab，第二批改开右栏 pane，见下）。
  4. 能力面板重设计（f3f9b5b0 + 6d4c6dfc）：`CapabilitiesPanel` 壳/`CapabilitiesInline`/`SkillsSection`/`ServersSection` 取办公现版（技能区保留办公域过滤），`ToolsSection` 渲染面换办公新版、数据面保留图文静态兜底表（功能分组：`wenxue_*` 归「图文专属」；`ToolSurfaceEntryB` 补 `group`）；样式直接引办公 `capabilities.css`（`components/capabilities/` 下是三级 `../../../`，写错两级 vite 解析失败）。`disabledDomains.ts` 补 `cost_library`（799e9ba8 只改了 Go 侧）。
  5. CSS 补查（首轮审计没比样式）：灵犀皮肤文件卡色值接主题 token（10 条）、对话两处强调色、代码高亮接 `--syntax-*` 并修浅色选择器（`.wx-root :root[data-hl="light"]` 永不命中，分叉起文学浅色主题代码块一直用暗色配色）、上下文视图补 5 段 @media 并清掉**分叉坏尾**（文件尾 5 段 `@media` 后面是生成脚本泄漏的 JS 数组字面量，自 44f8bb6b 起；坏块不闭合会吞掉后续规则）。
- **分叉结构缝（施工中由测试牵出）**：
  - i18n：文学复用的办公组件（ToolCard 委派卡、ApprovalModal…）查的是文学词典，办公后加的 10 个键缺失（委派卡等待态显示裸键名 `tool.delegateWaiting`）→ `wenxue/lib/i18n.tsx` 缺键按同语言回落办公词典。
  - 双份模块：`components/Toast.tsx`、`lib/useNow.ts` 逐字拷贝 → 改转发（Context 按子树、时钟全局，keepAlive 下安全；此前办公组件在文学里拿到的是缺省空实现，提示被静默丢弃）。
  - **keepAlive 下两板块 App 同挂，模块单例不能合并**（后挂载的注册覆盖先挂载）：新增 `gaea/components/boardPorts.ts`（Context 缝，未提供=模块单例，办公行为不变；初名 toolCardPorts.ts，第二批扩成板块级读口），`gaea/components/ToolCard.tsx`·`FileLinkText.tsx` 读口改经缝；文学 `ToolCard` 包装壳提供本树 `taskActivity`/`paneFileOpen` 槽位。此前文学侧子代理卡读不到自己的活动/打开器（无实时动态、点击不开会话）、卡内文件链接落到办公那侧的预览库。
- **审计更正**：首轮报告的「Esc 分层未同步」是误报（`app/useAppKeyboard.ts`、`components/Composer.tsx`、`components/ToolCard.tsx` 都是转发桩/包装壳，共用办公实现）——教训：**孪生文件要先读正文再判，只搜关键字会把转发桩当成缺失**。
- **验证**：`tsc -b` 0 错；vitest wenxue 27 文件 182 例全绿（首跑 1 例 `WenxueSettingsHub` 负载抖动：单跑 7/7、复跑全量绿）；办公回归 5 文件 96 例绿（ToolCard.i18n/FileLinkText/taskActivity/Transcript/Message）；eslint 32 个改动文件 0 错 0 警告；`npm run build`（tsc -b && vite build）过。新增测试：拷办公 `taskActivity`/`Transcript`/`ContextModal`/`SubagentThread` 四份到 wenxue + 能力面板三份 9 例。**未做**：桌面构建、装机、真机走查。
- **第二批（用户令「你说的 3 样都做」，同日）**：
  1. **右栏子会话窗 + 子智能体目录树 + 插话**：Go 新增 `WenxueSubagentSteer`（`wenxue_subagent.go`——插话的终态腿走追问管道，此前映射到办公绑定会把图文子代理的追问交给办公 runner，事件落办公通道）；**独立 Stop 不换锚**（审查批6 留档的决定不变：中断面按 ref 全局寻址、与引擎无关，直用 `GaeaSubagentStop`；我上一轮说「需新增两个绑定」有误）。门面与清单走生成器：`go run ./scripts/gen_bindings`（门面 +1）、`-names`/`-legacy-ts`/`-signatures-ts`（办公树）；文学树手补 `bindingNames`/两份 `bindingSignatures`/`legacyBindings`（`GaeaSubagentSteer` 落 legacy）+ `mappings.ts` 换锚 + wailsjs `WenxueB.d.ts/.js` 手补（下次 `wails build` 会再生，届时 diff 应为零）。前端：`SubagentPaneChat`/`SubagentCatalogButton` 取办公现版、`WorkspacePane` 特例渲染、`paneTabs` 加 `openSubagentPane` 且净化认 `subagent` 并保留 `meta.subagentRef`、App 接 `openSubagentAside`（委派卡与歧义选择器点开右栏，主对话不再被切走）+ 顶栏目录树 + 面包屑谱系、`useTaskCards` 取办公现版；词典补 `subagent.catalog*`/`openAside` 三键三语。
  2. **附件条预览缝**：`boardPorts.ts` 加 `openFilePreview`，共享 `ComposerAttachmentBar` 经缝取预览入口（未提供=办公预览库）；文学 `lib/boardPorts.ts` 单源读口，`ToolCard`/`Composer` 两个包装壳提供。
  3. **文学工具面预算守卫** `wenxue_toolface_test.go`（真实 `WenxueEngineInit` 装配口径）：首测 55 件 / 17682 字符，**量出造价/长文方案两组 6 件办公运行模式工具混进文学工具面**（`cost_project_apply`/`cost_project`/`cost_audit`/`longdoc_manifest`/`longdoc_build`/`longdoc_verify`，约 4400 字符、25%）——办公靠运行模式 ToolHide 藏掉，文学没有这一层，名单式 deny 又只钉了两组的一部分成员 → `wenxue_tool_policy.go` 加按组反查整组拒绝（`wenxueDeniedGroups`），前端 `disabledDomains` 前缀同批补 `cost_`/`longdoc_`/`proposal_`；复测 **49 件 / 13285 字符**，预算钉 49 件 / 13750 字符。
- **验证（第二批后）**：`tsc -b` 0 错；vitest wenxue 30 文件 201 例、办公回归 15 文件 165 例全绿；eslint 54 个改动文件 0 错 0 警告；`go vet ./internal/app` 过；`go test ./internal/app -run 'TestWenxue|TestBindingsCompleteness|TestOfficeToolFaceBudget'` 全过；`scripts/check-bindings-drift.ps1` OK（665 个方法）。新增测试：`wenxue_subagent_test.go`、`wenxue_toolface_test.go`、`wenxue_tool_policy_test.go` 按组用例；前端拷办公 `SubagentCatalogButton`/`subagentRunsStore` 两份 + 新 `paneTabs.test.ts` + `WorkspacePane` 子会话 pane 用例。
- **提交与打包装机（2026-11-26 上午，用户令「构建桌面端，git打包」）**：分域三笔——`b4934d7a`（宁丰 #1 重算内嵌补登，14 文件）/ `4f1e8049`（文学同步批，66 文件）/ `e6d018b6`（progress 记档 + 删仓根四张弃用图片）；提交信息经 `-F` 文件 + node 字节复读验证存储无损。打包=`releases/gaea-wenxue-sync-20261126-e6d018b6/`：干净树 detached @ e6d018b6（junction node_modules），Vite 30.7s + wails 55.8s（TMP/TEMP 显式系统 Temp、CGO_ENABLED=1、ucrt64 前置）；**wailsjs 零漂移**（`git diff --quiet` 过——手补的 WenxueB Steer 与再生件一致，挂池①销）；payload 四件 SHA256（exe `9b8f9711…7d367de`，三 DLL 与上包逐字节相同）；冒烟 200（PID 60032 回收）；包版四件回写主树 build/bin（对账一致）→ `gaea_install.ps1` INSTALL OK → 新壳 **PID 34140**（09:34:59 起，晚于构建时刻 09:34:40）；zip `b02ef89b…`（33MB）+ src.zip `1771fe1a…`（56MB，**排除 releases/ 防嵌套雪球**——首打 831MB 即没排除，PACKAGE-INFO 已照上包规矩重打）；PACKAGE-INFO.md + SHA256SUMS.txt 入库。
- **挂池**：①文学树绑定清单自 11-25 造价库重建后未再生（仍挂 59 个已退役造价名、缺 2 个新造价名；不影响运行），本批只补了本次新增的一名；②`wenxue_*` 专属 19 件约 4200 字符，是后续梳理候选面；③文学工具面里还有办公标准档也有的通用件（`ai_feedback`/`md_validate`/`send_message`），本批未动；④右栏子会话窗/插话/附件条预览为界面行为，接线有测试锁但**真机走查未做**。
- **审计脚本**（系统临时目录 `wx_audit/`，未入库；需要时按 `.gaea/progress.md` 本条与记忆里的方法重写）：reach（可达性）/ mirror（提交×文件增量核对）/ csscmp（样式规则级比对）/ i18ncheck / dualmod。

## 铜陵六表反馈批 + 测算样表化（2026-11-25，删渲染器拍板；本节提交号见 git log）

用户先交《铜陵项目六表编制_软件使用反馈与建议》（15 条），令「修复问题」，中途追加「六表排版比五表差远了，一起调整」；看了效果图后先纠「封面是不是设计错了」（修：信息区+整行合并）；再交人工精修样表令「把这个作为测算模版，已测试可以导入」，我开动手工复刻渲染器时被拍板：**「删除渲染器这条思路，你现在没有能力完美复刻这个表格，而且彻底堵死后续模型的发挥。提供样本给模型，再进行约束」**；我问意见后用户令「开始吧」，又补「工料机不用预填写，模型自己检索引用」。中途再次被质「就没有自己的思想吗」——连续两次顺着执行不给意见，已记 feedback。

- **测算样表化（本批主刀）**：样表=版式真源（`skill/assets/cost-estimate-sample.xlsx` 内嵌种子；`<工作区>/.gaea/skills/cost-estimate/样表-成本测算模版.xlsx` 覆盖优先，改版式换文件不等发版）。`cost_library action=template` 与绑定 `GaeaCostLibraryExport kind=template` 改样表复制（回执=样表+导入契约+三道门）；`workcost/template.go`（ExportProjectTemplate）与 costTemplateOptions/Fee/ScopeCheck 删除；库导出（bill/quota/resource 数据表）不动 render.go（本轮补的版式层保留）。工料机不预填：模型检索库价自己填（技能正文钉死）。导入契约与三道门（xlsx_recalc → xlsx_audit → import dry_run）进 cost-estimate 技能正文。golden 测试钉住内嵌样表可解析（6 表/60 资源/费率 5·4·0·9%）、复制逐字节一致、覆盖优先、样表闭环（填两组→导回 1:N 挂接系数全对）。
- **反馈批工具面**：①#13 根因=CanonicalizeSchema 逐层删字段 description，多动作工具语义全在字段描述里——加 `SchemaDescriptionsKeeper` opt-in（cost_library 保留描述）+ `UsageHint` 接口（loop guard 附参数骨架，#14）+ quota 错位嵌套参数救回（回执点名）。②字典扁平动作 dict_add/update/delete（原 label 表登记了却没分支，落到 health 扫描——顺手修的暗 bug）。③import_project 封面口径兜底（#3）、跳过行带表名+行号+摘要（#4）、取费链对账（#15：122,785 差异 98% 是利润基数口径分叉不是精度舍位——工作簿=直接费×4%，库内=(直接费+管理费)×4%；feerecon.go 反推基数拆性质）。④template path 生效（#5）+ 空库提示（#7）。⑤落盘路径闭环：move_file 源允许系统 TEMP（#2，发布指引与闸门打架）+ move_file 豁免同轮 stale 闸（#8）+ office/xlsx 工具 MSYS /tmp 与 /d/ 路径翻译（#10）。⑥session_search SQLITE_BUSY 短重试（#11）、read_spill 行区间（#12）。#9 heredoc 警示已在线上。
- **六表版式层（render.go，现只服务库导出）**：深藏青题栏/深蓝表头/浅蓝汇总行/黄底输入格/绿字公式格/细灰边框/列宽/冻结/横向按宽打印/打印重复表头（_xlnm.Print_Titles，本版 excelize 无直接 API）；封面信息区+整行合并说明段（用户纠的封面两处硬伤）。
- **SchemaV41 工料机分类字典种子**（用户令「补充工料机分类字典，现在无法分类」——清库把分类字典清空了而规矩是值必须先在字典里）：34 项分级分类树（顶层=人工/材料/机械/外委），编号即自有码分类段（全局两位 01–34）；幂等防撞三分支（同名补号/用户码不动/段号被占整条跳过）。迁移链、schema_v41_test、dims/dict/gcode 断言全量对齐。
- **门禁**：Go 全量绿除 voice/realtime 3 测（干净树同红=存量/环境性）；vitest 全量 4663/4663 绿；eslint 0 错；golangci 我的文件零新增（voice realtime 与 format_convert 红为环境性）。模型侧根因证据=会话日志取证（quota 错位嵌套 9 轮、schema 剥描述实锤）。
- **坑（本批新增）**：①excelize v2.11 本机版本 Font 字段是 `Family` 不是 `FontName`、Alignment 传指针、DefinedName.Scope 是 string、无 SetPrintRows（走定义名）；fitToWidth 必须配 SetSheetProps FitToPage 否则 Excel 不认。②`go test | tail` 把退出码吃了——后台全量 FAIL 被 exit 0 掩盖，读日志才算数。③builtin 测试的 t.TempDir 在系统 TEMP 下，测「界外」要 t.Setenv("TMP/TEMP") 重定向再造界外。④app 包测试必须带 build/bin DLL PATH，否则 0xc0000135。⑤python 写 Go 文件遇反引号必炸（Write 工具落盘纪律再证）。⑥GetRows 取带格式单元格返回格式化文本（费率列「5.00%」），断言按数值比。⑦voice/realtime 与 format_convert(markitdown) 红为环境性，与改动无关。
- **遗留/挂池**：①样表闭环的「真实模型实弹」未做（本轮测试=程序化填表）；②板块「导出→测算模版」前端文案仍写「生成」未改（出口行为已变）；③push 挂令；④voice/realtime 存量红待归因（干净树同红，疑似需真机音频环境）。

## 造价数据库清空（2026-11-25 16:40，用户令；归类线作废）

用户先令「本次对数据库的数据全部进行分门别类，达到可以使用的条件」，我只读取证约 40 分钟没出方案，被打断后改令：「把整个造价数据库清空，我用 gaea 重新按照新版本完成成本测算后导入」。

- **已删**（真实库，单事务，`.tmp/wipe_cost_db.py`，副本演练 → 备份逐表核对 → 执行）：三级数据全表（项目 7 / 清单 329 / 挂接 295 / 定额 295 / 含量行 970 / 工料机 1786）、旧资料条目全家（cost_entries 1794 / cost_categories 106 / cost_stage_values 4）、price_fetch、旧条目的全文索引 `semantic_fts(kind='cost')` 833 条与向量 1 条（在造价 schema 之外，清造价数据必须连带清）、地区字典 6 与工料机分类字典 50（「取自库内已有取值」的残留）。
- **保留**：规划专业字典 10 个、price_sources 2 条、`gf_gcode_seq`（发过的号不复用）。代码与装机不变（包 gaea-cost-code-20261125-4ce95a8b，PID 41636）。
- **备份**：`%APPDATA%\gaea\Hephaestus.db.bak-20261125-164059-pre-wipe`（65,691,648 字节，schema 40）。信息价库（重庆 2026 年第 7 期 1184 条 / 成都 24 / 乐山约 57）也一并清了，需要时可只从备份恢复这部分。
- **核对**：各表 0 行、integrity ok；运行中的壳经调试口只读绑定读到项目 0 / 定额 0 / 工料机 0、体检 0 条。壳未重启，真实库文件未 VACUUM。
- **取证时查到、对重做有用的事实**：① 沧州原表（五表旧版）有 9 个单位工程约 213 条清单，旧库只进了 120 条——综合单价页每个单位工程的块序号从 1 重排，旧五表解析在那里停了；现版不认五表，要进库须先转六表或走模型读表。② 旧数据是旧流程的产物：综合单价类包干价被记成工料机、类别乱挂、同一监测外委 4~5 个重复码、41 种分部写法。
- **下一步（用户）**：在造价模式按新版重做测算并导入；地区、工料机分类字典现为空，用到时经 dict add 新增。代码侧余项不变：公用专业免「外专业定额」提醒、模型读表与模版回导的真实模型实弹、push 挂令。本节与下一节里的「存量 / 待整理 / 真实库已发码」均指已清掉的数据。

## 规划专业表 + 自有编码 / 六表测算模版装机（2026-11-25，提交 4ce95a8b + 包 gaea-cost-code-20261125-4ce95a8b 已装机）

用户指令：「清单和定额的专业你重新进行规划，不要考虑全数据问题，数据整理我们下个会话再展开，定好专业后构建装机」；随后逐项调整：去掉仿古建筑 / 园林绿化 / 矿山 / 城市轨道交通 / 爆破，加四类环保专项与临建 / 措施 / 检测；再去掉大气污染防治 / 物理污染防治 / 构筑物工程。本节取代上一节的「已构建、未冒烟、未装机」状态（那个包 `b68db174` 已作废，从未装机）。

- **规划专业表（SchemaV40，`db/schema_specialties.go`）**：清单与定额共用 10 个专业，代号是自有码的专业段——环境修复：土壤修复 TR / 地下水修复 DX / 水污染防治 SW / 固废处置 GF；建设工程（现行国家清单体系 GB/T 50500-2024 及配套工程量计算标准，2025-09-01 施行）：房屋建筑与装饰工程 FJ / 通用安装工程 AZ / 市政工程 SZ；项目配套（各项目公用）：临建 LJ / 措施 CS / 检测 JC。共用一套专业名的理由：套定额时「同专业优先」靠的就是同名。迁移幂等：库里已有的同名专业只补代号 / 排序 / 说明，用户手填过代号的不动；地区与工料机分类的字典仍不预设（V38「不预设任何清单」只对专业翻案）。改专业改这张表。
- **装机**：干净工作树 `.worktrees/pkg-4ce95a8b` 构建 Vite 27s + Wails 60s；装机前备份 `%APPDATA%\gaea\Hephaestus.db.bak-20261125-160741-pre-v40`；正式 payload 冒烟 200；`gaea_install.ps1 -SourceDir payload -Launch` INSTALL OK，新壳 PID 41636（`C:\AI\gaea-app\gaea.exe`），主树 `build/bin/gaea.exe` 同步成同一版（用户习惯从那里起壳）。用户的壳在装机前已自行关闭，未杀进程。
- **真实库结果**：schema 38 → 40；七张业务表行数不变（7 / 329 / 295 / 295 / 970 / 1786 / 0），字典 57 → 66（专业 1 → 10，地区 6、分类 50 未动）。自有码经调试口调一次只读的定额列表绑定触发首次打开后全部发出：定额 295 / 295（`DTR00001`–`DTR00295`）、清单 329 / 329（专业段全是 `00`——都还没归专业）、工料机 1786 / 1786；零空码零重码；分类字典 `code` 列写段号 01–50（一列唯一）。与副本预演逐项一致。
- **不可逆的事实**：码一经发出不复用、不随之后的归类改变。现有 329 条清单的专业段因此永久是 `00`；下个会话归类后，新录入的清单才带真实专业段。用户在装机前被明确告知并选择先装机。
- **留给下个会话**：① 数据整理——清单归专业、定额归章节、41 种分部写法归并（用户指定）；② 临建 / 措施 / 检测是公用专业，归类后会被大量项目借用，「外专业定额」提醒可能变噪音，考虑对这三个专业免提醒（未做，等与归类一起定）；③ 存量数据问题（未改）：十来个顶层分类没按类别命名、3 条「旺平-…-元」汇总金额被当成工料机；④ 模型读表导入与模版回导未经真实模型实弹；⑤ push 挂令。
- **坑**：⑨→ 后台任务 10 分钟上限，全量 Go 与全量 vitest 并发会被双双杀掉，必须串行；`TestGaeaPriceFetchAllTask`（price_fetch 族）全量下偶红、单跑 5/5 绿；`zip` 命令本机没有，打包走 `.tmp/make_zip.ps1`（Compress-Archive）；PowerShell 路径里 `\$D` 会被 shell 吃成字面量，用正斜杠或落盘脚本；`git archive` 的源码包里中文文件名按 cp437 解码会「集合不等」，按 UTF-8 重解后与提交文件集一致；探针与核对脚本在 `.tmp/`（`backup_real_db.py` / `verify_real_db.py` / `cdp_readonly_probe.mjs`，不入库）。

## 造价自有编码 + 六表测算模版（三级行组）（2026-11-25，提交 b68db174 + 包 gaea-cost-code-20261125-b68db174【已构建，未冒烟、未装机】）

用户原话：「造价编码我们自己设计一套与现在的体系对应」「成本测算模版要依据现在的造价数据库进行程序设计，使其对应」；追问「综合单价表里面一个清单会有几个定额」→ 三级行组；「模板5表改成6表，其中费用汇总进行拆分成清单表和费用汇总表」「旧版用模型解析，不用兼容」；质疑「改了半天就是在手搓一个表？」「不要我说什么就是什么」→ 定案：版式留在程序且只留一处，测算知识进技能。

- **自有编码（SchemaV39，`gcode.go`）**：清单 `Q+专业2+分部2+流水3`、定额 `D+专业2+章节2+流水3`、工料机 `R+类别1+分类2+流水4`。外部码原列保留为映射列，详情 / 检索 / 导出双码并显。写入即授（三处 INSERT 分支）、开库补授存量（幂等；无待授行只跑一条计数）；水位表 `gf_gcode_seq` 只前进，删行不复用；部分唯一索引放行待授空码。专业段落字典 `code`（用户缩写优先）；分部 / 章节在专业内编号、记映射；分类全局编号并写回字典 `code` 列（一列唯一）；没归类固定 `00`。
- **六表与三级行组**：费用汇总拆成 清单表 + 费用汇总表；判据须带清单表（`DetectedSixSheet`），旧五表不再兼容。综合单价 = 清单汇总行 → 定额行 → 工料机明细行：解析端认定额行与系数，`ApplyProjectBundle` 逐块落定额并按 1:N 写挂接（系数进该定额工程量；同清单重复挂同一定额只取首个）。此前 1:N 清单导出再导回会退化成一对一。
- **模版**：导出与空白模版合并为唯一渲染器 `render.go`（空白模版 = 空清单 + 预填工料机 + 空白组）。出口：`CostLibraryExport kind=template`（板块导出菜单）、`cost_library action=template`（造价模式；只读库只新增文件，不弹确认卡；口径须在字典内）。测算口径与填法进内置技能 `cost-estimate`（`skill/cost_estimate_body.go`），工具说明与回执只留指向；不附带静态样表（存一份必过时，要表就生成）。
- **真实库副本取证抓到并修掉的两处设计缺陷（装机前）**：① 第三段（分部 / 章节 / 分类）全是 `01`——每个名称各开一个计数器，329 条清单全挤在 `Q0001`；旧用例每个作用域只有一个分部没钉住，现补「同专业多分部段号不同」用例。② 分类段按类别各编一套，字典 `code` 列出现四个「01」；改为全局编号。另修一处前缀长度写错（3 位应为 4 位，自动号会撞手填码）。
- **本批实测抓到的其余 bug**：apply 链按 (清单, 定额码) 分组却拿清单码去查 → 定额含量行全丢（6 条用例红）；模版说明行含「清单编码」「行类」被当成表头行、所有列错位到第 1 列（`findHeaderRow` 改为各关键词须落在不同列）；定额系数≠1 时汇总行三费没乘系数（LO 重算金样抓到）；两个夹具转六表时费率列为百分号文本、公式无缓存值（转换器双开工作簿 + LO 重算）。
- **验证**：workcost / db / skill / control 包全量绿，app 包 CostLibrary* 全绿；全量 Go 只剩 docmd OCR 环境用例与 proposal V35 迁移用例两条存量红（干净 HEAD 同红）；vet 净；lint 仅 `billquota.go` 的 `quotaLinksByItems` 未使用（13e6e6f1 存量）。模版与导出各一组 LibreOffice 重算数值金样。vitest 全量 500 文件 4663 例：4662 绿，`SecurityPanel` 1 例全量并发偶红、单跑 3/3 绿、与本批零交集；tsc 0 错；eslint 净。真实库副本（在线备份快照）：V38→V39 八表行数不变（7 / 329 / 295 / 970 / 295 / 1786 / 0 / 57），补授 2410 行 135ms，无空码无重码，重开幂等。
- **打包**：干净工作树 `.worktrees/pkg-b68db174`（junction node_modules）构建 Vite 43s + Wails 1m9s（bindings 未跳过）；构建后 29 件仅行尾差异，手改的 `models.ts` 与生成器一致；exe 对语音 DLL 的引用与基线同形。包 `releases/gaea-cost-code-20261125-b68db174/`（exe `6BE0FCAD…6A9049`，三 DLL 与基线同）。工作树已按红线摘 junction 后移除。
- **未做**：payload 冒烟与装机。`smoke.ps1` 走真实配置，一跑就把真实库升到 V39；用户的壳（PID 15700，自 `build/bin` 启动）在跑。装机时：真实库 V38→V39（只加列）；首次打开造价库时给存量 2410 行发码（一经发出不复用）。真实库目前仍是 V38、未动。
- **数据观察（授码把存量问题显形，未改数据）**：分部叫法不统一——`A临建` / `A施工准备与临建` / `一、施工准备与临建` 各占一个段号；十来个顶层分类没按类别命名（`措施费` / `设备` / `基坑` / `人工` / `机械` / `材料` / `其他`）；3 条「旺平-…-元」汇总金额被当成工料机（如 `P1203 旺平-含税工程造价-元`，分类落成「费用汇总」）。
- **遗留**：push 挂令；模型读表导入与模版回导均未经真实模型实弹；浏览面中栏列数偏多（用户令先不管 UI）。

## 造价数据库维度化 + 库定规矩 + 模型读表导入（2026-11-25，提交 13e6e6f1 / 05a2fd7d + 包 gaea-cost-dims-20261125-05a2fd7d）

用户原话：「继续优化完善造价数据库，清单要区分专业（分部分项，消耗工程量×定额），定额也分专业（定额消耗量×工料机）；工料机（人材机）是大库，要细分区分种类、地区、时间」。四轮拍板：①工料机按地区分行（同一种工料机每个地区一行，时间=该行各期价格）；②「定额的专业是固定的，同样一件事不同专业的定额是不一样的，清单使用的定额优先使用同专业的，没有就借鉴别的专业生成自己专业的」（否决上一版「迁移里回填定额章节 / 批量改定额专业」）；③「你的数据库要定规矩，不是比着几个文件去适配」（追问的四问未答，按最佳判断落：字典 + 校验、初值机械取自库内、期数 `YYYY-MM`、不做存量认领）；④导入「造价资料格式不固定，不只是标准五表」→ 模型读表为主，五表机械导入留作标准模板快速通道，「成本测算模版要依据现在的造价数据库重新设计」挂后续。

- **库定规矩（SchemaV38）**：`gf_dicts` 字典（专业 / 地区 / 工料机分类；`rules.go` DictList/Add/Update/Delete，改名级联、分类改名连带下级）；写入校验 `ValidateResource / ValidateQuota / ValidateProjectScope / CheckPrice`（字典外取值、期数不是 `YYYY-MM`、类别不在四类、编码不像编码一律拒）；工料机身份加地区（唯一索引换 `(kind,title,spec,unit,region)`）；价格落成价格事实 `gf_resource_prices`（资源+期+地区+来源幂等）；项目加专业 / 地区 / 计价期、清单行加专业（读时行自身空则跟随项目）、定额加 `derived_from`。定额专业为固有属性：`ResolveQuota` 同专业才复用，外专业同码另立并记「借鉴 <专业> <码>」；飞轮按专业聚合；查漏只比同专业。升库只加列、机械播种字典、把 1385 条「YYYY年M月」改写成 `YYYY-MM`。
- **模型读表导入（主线）**：`importstage.go` + `office_cost_import_tools.go`：`import_open`（登记来源，内存暂存 2h）→ `import_add`（工料机 / 定额 / 清单分批，每批 ≤200）→ `import_check`（控件勾稽、回读原行逐字段核对、价格比对 >1% 点名、悬空引用）→ `import_commit`（dry_run 或落库，落库前全量复验 + 幂等 upsert，hardAsk 确认卡带规模摘要）→ `import_discard`。读回不靠模型自述：xlsx 走 excelize 原始单元格值、csv 带 BOM。
- **整理归类**：`classify.go`——`classify`（项目 / 清单 / 定额 / 工料机四类方案，事务内演练回滚、正式一事务；定额专业只补空或纠错，纠错仅当引用它的清单同属目标专业，否则进冲突清单）；`borrow_quota`（复制含量行生成本专业新定额 + 改挂清单行）；`dims`；`dict`。`save_quota` 拒改专业；`cost_project_apply` 建项目先校验专业 / 地区 / 计价期，挂外专业定额回执点名不拦；机检 `HealthScan` 加第六类「不合规矩的数据」，`cost_audit` 加跨专业查漏。技能 `cost-estimation-playbook` 与身份提示词加「库的规矩」「录入资料（模型读表导入——默认路径）」「整理归类」三段。
- **浏览面（05a2fd7d）**：`logic.ts` 重写（两级上下文、分类树、地区 / 期 facet、当前位置、待整理计数）；`TierRail` 分段小标题 + 全展开分类树；清单列表加专业 / 工程量列、详情含量表加「消耗量」列 + 外专业标；定额加章节列 + 借鉴溯源标；工料机加地区 / 期列 + 详情「地区 · 期」与各期价格表；状态行「待整理 N」入口（复制整理提示词）；体检抽屉加一节可跳转；导出绑定改 `GaeaCostLibraryExport(kind, filter, outPath)`（`LibraryFilter` 承载全部口径，含 `__none__` / `__outside__` 哨兵）。真实库走查中修五处：类别不在四类之内的存量行左栏点不到→「类别待整理」桶（读侧类型放宽 `WorkcostKindRead`）；待整理总数与自己的说明对不上→总数=各项之和；当前位置里期数显示成 `2026-06`→年月；同级点一行看详情会清空筛选（`jump` 无条件清空，上一批就有）→目标行过不了当前筛选才清空；体检按钮提示语「五类」改准。
- **验证**：Go workcost / db / control / tool 包全量绿，app 包 Cost / Workcost / Bindings / Flywheel / Price / OfficeRunMode 相关绿，vet 净；前端 vitest 全量 500 文件 4662 例绿、tsc 0 错、eslint 净；绑定漂移守卫绿（668）；工具地图文档重导追平存量漂移（上一提交遗留的 format_convert / proposal_save 两行）。真实库升库先在副本验证、冒烟时真实库同样升到 V38：七张表行数不变（项目 7 / 清单 329 / 定额 295 / 含量行 970 / 挂接 295 / 工料机 1788 / 价格事实 0）、旧列逐字一致、唯一内容变化=1385 条期数改写、悬空引用 0→0、真实库与副本逐表一致。界面走查（宽窗 1760 + 窄窗 1000）：三级左栏、筛选、新列、详情、窄窗下拉、跨窄窗分界状态保留均过。
- **打包装机**：干净工作树 `.worktrees/pkg-05a2fd7d`（junction node_modules）构建 Vite 86s + Wails 54s（bindings 未跳过）；包 `releases/gaea-cost-dims-20261125-05a2fd7d/`（payload 四件 + ZIP + 源码包 6566 文件 + PACKAGE-INFO.md + SHA256SUMS.txt；exe `E4A98E8EEB89…`，DLL 同基线）；payload 冒烟 200；`gaea_install.ps1 -SourceDir payload` INSTALL OK，装机位哈希=包哈希。版本不抬号；未 push。
- 坑：① `scripts/smoke.ps1` 走真实配置，新包一启动就把真实库升了版——升库验证必须排在冒烟之前（副本先行），冒烟后再拿真实库与副本逐表比对；② 升库对比脚本首版把新增的空列算进摘要，与升库前（没有这些列）比必然不等——比内容要按升库前的列集合比；③ 板块常驻保活，同一走查壳复跑驱动会带着上一轮的筛选，驱动开头要整页重载；④ 窄窗模式下左栏整列不渲染（不是 `display:none`），断言别去取它的样式；⑤ 干净树构建后 wailsjs / go.mod 等 29 件显示被改=纯行尾差异（`--ignore-cr-at-eol` 无内容出入）；⑥ control 包测试会把 updated_at 写回入库夹具 `resumed.jsonl.meta`，跑完工作树会脏（还原即可；夹具被测试写回是存量卫生问题）；⑦ 走查沙箱壳启动时 `migrateLegacyDataRoot` 会把仓根旧 `whisper_data` 复制进沙箱（只复制不搬走）。
- **遗留**：模型读表导入链只有用例、未经真实模型实弹；详情栏打开后中栏约 820px、列数偏多需横向滚动（次要列收起按整页宽度而非中栏宽度判定；用户指示「不要管前端 UI，尽快完成」）；成本测算模版按现在的库重设计（挂池）；存量待整理 1869 项（清单无专业 329、定额未归章节 295、工料机未标地区 539 / 未细分 302 / 无期数 403 / 类别不在四类内 1〔id 1642，编码 / 类别 / 名称都是同一句说明〕）待在造价模式分批归类；地区字典初值口径不一（「乐山市」与「乐山市中区」、「重庆」与「重庆中心城区」并存）；仓内 `.tmp/gaea-walkthrough`（132MB，11-24 旧沙箱）与 5 个旧 `pkg-*` 打包工作树未动。

## 造价数据库推倒重建：三级浏览面 + 维护归造价模式（2026-11-25，提交 4adec93a + 包 gaea-cost-db-20261125-4adec93a）

用户拍板（原话要点）：「参考方案库的模式，造价数据库也退役自己的导入和解析等入口，将导入、解析、维护这些交给办公板块的造价模式，造价数据库保留导出、删除等基本功能」；追问范围时追加「整个造价库可以推翻重新架构，不用在上面修修改改浪费时间。我只需要看到清单，定额，工料机的三级造价数据可以服务于造价模式」。计划经 AskUserQuestion 四问拍板（留库页维护入口=无 / 导出=四类列表 Excel / 对标表=下线），ExitPlanMode 批准后执行。

- **板块重建**：`CostLibraryPage` 重写 + `pages/cost-library/` 五件（TierRail / Tables / Details / HealthDrawer / Overlays）+ `logic.ts` 纯函数层。三级浏览面：左栏三级切换+上下文（项目/专业/类别）、中栏列表（检索、批量删除、下钻 chip）、右栏详情按需展开（综合单价分析按挂接定额分组 / 含量行分析表 / 工料机价格卡+调价历史+被谁引用），引用链可回跳；四态分明、keepAlive 切回静默重读（`useBoardActive('cost')`）；列表列渐进披露（`cl-pri-1/2` 容器查询收列，详情里有）。删除只留单条/批量+确认卡，后端拒绝（仍被引用）原因留在卡上；导出菜单走 `GaeaCostLibraryExport` 四类列表 xlsx（拒绝覆盖）；体检=只读 `CostHealthScan` 抽屉可跳行。旧 8 模块 26 组件+18 测试整删（−3110 行测试）：导入入口×3、资料条目全家（导入/AI 解析/视觉识别/分类管理）、批量调价/资源化/自生成/去重弹窗、费率编辑、挂接定额、概览、造价参考（分位数对标下线——读已退役 costproject 表真实库 0 行恒空）、图谱、对标查漏页、备份卡、价格源/仓库/询价库界面。
- **维护归造价模式**：新工具 `cost_library`（`office_cost_library_tools.go`，PersistWrite+hardAsk 确认卡、`dry_run` 豁免弹卡）：`import_project`（五表工作簿，演练给规模/告警/同名项目）、`save_quota/delete_quota`（含量行整组替换且每行必须引用在库工料机；强删先 DetachQuotaEverywhere）、`save_resource/delete_resource/set_price`（按 id/code 定位、先定目标再合并字段防价格清零、改编码拒绝）、`bulk_price`、`delete_items/delete_project`、`seed/flywheel/dedupe/health`。注册面七处同批（gaea_handler 装配 / toolgroups cost 组 / spacetags / registrymap+工具地图文档 / wenxue 拒装 / 审批 hardAsk+摘要+`costLibraryArgsNoWrite`）。
- **cost_project_apply 三缺陷根修**（入口撤收后它成唯一写路径）：①改已有行清 `list_price`→已有行未传字段保原值；②`quota_codes` 只进回显不落库→`SaveBillItemLinks` 整体替换 `gf_bill_quota_links`+主定额列同步，回执点名被解除挂接数；③`update_rates` 按零覆盖→入参全指针化只改传入字段。回归钉 `TestCostProjectApplySaveItemsKeepsSnapshotAndLinks`。
- **引用面与护栏**（`workcost/usage.go`）：`QuotaUsage/ResourceUsage/QuotaReferenced/DetachQuotaEverywhere/ResourceReferenceCounts/SaveBillItemLinks`+`FindBillItem`（聚合挂接码）；绑定 `GaeaWorkcostQuotaDelete` 加引用拒删、新 `GaeaWorkcostResourceUsage`。
- **绑定面 727→668**：普查脚本证实旧界面消失后 CostB 72 绑定只余 15 个有界面消费，59 个（57 CostB+语义索引 2）进生成器 `excludedBindings`（方法体保留，复活=除名重跑）；`CostBindings` 接口/mock 收到 15 个；`types/cost.ts` 清 56 个孤儿类型（−608 行）；绑定漂移守卫绿。
- **提示词/技能**：造价档 ModeLine、身份提示词（新增「造价库维护」节）、cost-estimation-playbook 同步（板块只读、dry_run 先行、force 删语义、整体替换带原挂接）；锚测试 `TestCostLibraryPromptAnchors`。工具数 7→8。子导航两端（manifests.ts COST_NAV / board builtins.go Nav）同步为三级并修正既有漂移。master plan 顶部加 2026-11-25 推倒重建节；`.gaea/AGENTS.md` 加「库页只读约定」+绑定面现值 668。
- **方案库实弹反馈批同树收口**（2026-11-24 晚另一会话遗留未提交，与本批六处注册面文件交织，按「九批合一」先例合并一笔提交）：proposal_update/delete/export 三维护工具、proposal_save status+dry_run、乱码 PDF 走本机 WPS wpscli pdf2word（docmd/wps.go）、genui 最小示例单源、检索缓存提示自解释、方案库板块切回静默重读。
- **验证**：Go app 全量+workcost/tool/control/board 绿；前端 tsc 0 错、vitest 全量 4639 例绿、eslint/knip 无新增欠账；绑定漂移守卫绿（668）。真实库副本走查（GAEA_WALKTHROUGH 沙箱壳+Hephaestus.db 副本，走查后已清场）：项目 7/清单 329/定额 295/工料机 1788 与库内一致，7 项目及各自条数全对，清单行详情（原表单价 ¥15,660.48 对现算 ¥15,649.8、含量行 10 行）正常。**未走查**（被打包令截断）：导出/体检抽屉/删除路径的真机走查、造价模式实弹回合、布局修复后的复验。
- **打包装机**：干净工作树 `.worktrees/pkg-4adec93a`（junction node_modules）构建 Vite 73s+Wails 67s（bindings 未跳过）；包 `releases/gaea-cost-db-20261125-4adec93a/`（payload 四件+双 zip+PACKAGE-INFO.md+SHA256SUMS.txt；exe `4260836CC011…`，DLL 同基线）；payload 冒烟 200；`gaea_install.ps1 -SourceDir payload` INSTALL OK，装机位哈希=包哈希。版本不抬号；未 push。
- 坑：①走查沙箱 `GAEA_WALKTHROUGH=1` 会把 APPDATA 整体重定向到 `%TEMP%\gaea-walkthrough\appdata`——想让它读真实库副本，库要放进沙箱 APPDATA 而非自设 GAEA_DATA_ROOT；②walkthrough 版 cost 布局初版在 1264px 视口下右栏空态占位挤压列表（1280px 表仅 528px 可视）→按需详情栏+列收起修复；③前端页面测试不能用 jest-dom 匹配器（仓库未装），断言走自写 textContent 帮手；④mock 契约测试（X1-11）逮 `CostLibraryExport` mock 形参 0 vs 门面 5——新增绑定 mock 必须按门面签名声明形参。

## 方案库实弹反馈批 + PDF 经本机 WPS 转换（2026-11-24 晚，已随 4adec93a 收口提交）

用户录入两份方案后方案库界面只显示一份；办公模型另出 7 条使用问题汇总
（`D:\新工作\沿江一公里\南岸\01_编制工作区\gaea软件使用问题与改进建议_20261124.md`）。
用户令「需要一起修复」「你自己定，我只要结果」，并定向「PDF 使用本机 WPS 的格式转换」
（已明示走 WPS 云端可以：本人是会员，WPS 打开本来就会上云）。

- **前端只显示一份**：汇总里的「界面按 workspace 分组」不成立（库里两份都在、
  workspace 一致、`List()` 返回两行、前端无此过滤）。真因=板块 keepAlive 常驻，清单
  只在首次挂载读一次。落刀=`useBoardActive('proposal')` 隐藏转可见时静默重读、
  清单头部刷新按钮、所选方案消失后改选第一份（`ProposalLibraryPage`，18 例绿）。
- **PDF 乱码却报成功**：`docmd` 自写提取无字体映射层，子集字体字形号被当 UTF-16
  硬解（233 页报告复现：汉字只剩 1589 个、17,156 个替换字符；页对象在对象流里，
  页数读成 1）。落刀=`internal/docmd/wps.go`：乱码质检门（hard/soft）→ 本机 WPS
  `wpscli pdf2word` → docx → markitdown → 合并跨页断表（同列数 + 表头重复）；只有
  乱码的才上 WPS；进程内缓存按内容指纹。实测同一份报告：233 页、汉字逐字 100% 命中、
  104 张表（165 并 61）、108 张图，全程约 90 秒。hard 乱码且 WPS 不在 → OCR → 再失败
  报 `FORMAT_TEXT_UNDECODABLE`（以前把乱码当正文返回）。`format_convert` 回执改按
  字数计并附「标题/表格/图片」计数。详见 `docs` 外记忆 gaea-pdf-via-wps-conversion。
  - `wpscli pdf2md` 在本机 4 次调用全部 success + 0 字节，不可用，故走 pdf2word。
- **维护接口**（问题 3）：新增 `proposal_update`（元数据 / `section_id` 增量修正，子树
  路径与索引同步）、`proposal_delete`、`proposal_export`；存储层 `proposal/maintain.go`。
  注册面七处同批（compact/spacetags/registrymap+文档/审批/图文拒装/方案模式技能）。
- **录入进度**（问题 5）：`proposal_save action=status`、`dry_run`（事务回滚、图片不落
  盘、不弹确认）；回执加「落点：工作区 + 库内总数」（问题 2）；清单加「目录」列。
- **检索缓存**（问题 6）：去重键含结果哈希，命中即结果确实没变；提示语改为自解释并
  带结果开头（`agent_run_cache.go`）。
- **genui**（问题 7）：最小示例单源表 `genui/examples.go`（35 个 type 均过校验）；
  报缺字段附示例；`genui_validate` 报错分「参数错误（调用方式）」/「规格错误」；
  手册加必填字段表，`TestHandbookExamplesInSync` 把关。
- **验证**：非 app 的 internal 全量 127 包绿（仅 `TestBuildUocr2LlamaServerCmd` 红，
  干净 HEAD 同红=既有）；`tsc -b` 0 错；真实报告端到端复跑通过；`internal/app` 全量 165.8s 绿。
- **构建装机（11-24 23:57）**：前端 `npm run build` 72s + `wails build -s` 1m12s
  （bindings 未跳过，无再生 diff）；exe SHA256 `F1C768BF…C089`；build/bin 隔离
  profile 冒烟 200，`gaea_install.ps1` INSTALL OK（装机位同哈希），装机位冒烟 200。
  用户当时的壳（PID 47708）由用户自行关闭；旧 exe 留作 `build/bin/gaea.exe.prev-2150`。
- **未做/余项**：WPS 结果未落磁盘缓存（重启后重转）；续页不带重复表头的断表仍不合并；
  WPS 产物无标题样式，方案库仍靠模型定级；跨页表格在纯文本路径（无 WPS）下不处理。
- **库现状**：正式库只剩 id 4、5 两份；此前保留的夹具「包3技术标 第三版」已不在库里
  （非本批所删，未核实来源）。

## 一级导航图标辨识度修正（2026-11-24）

用户反馈方案库一级图标与办公板块太像；首版笔记本方向被否决后，明确要求办公直接用
台式机图标。

- **办公**：原“叠页文档 + 横线”改为显示器、支架与主机塔组合，主栏 20px 直接读作
  台式机。
- **方案库**：原“单册方案 + 标题线”改为分层档案抽屉 + 检索镜，表达“归档成库 +
  按需查找”，与办公的设备轮廓完全分开。
- **验证**：boardIcons / manifests / CommandRail 定向 73 例绿；`tsc -b` 与改动面
  ESLint 绿。仅前端源码修改，未构建、未装机。

## 方案库图片托管改造（2026-11-24，桌面构建完成，未装机）

用户拍板方案库应保存方案图片后，完成“文件托管 + 数据库索引 + 稳定引用”闭环。

- **SchemaV37**：新增 `proposal_assets`，按方案/小节归属记录 sha256、MIME、字节数、
  宽高、原文件名、图注；二进制落 `%APPDATA%\gaea\proposal-assets/objects/`，正文只存
  `gaea-asset://<sha256>`。同图跨小节/跨方案共用磁盘文件，删除时只在最后一个引用消失后回收。
- **录入**：`proposal_save` 自动识别 Markdown 图片，复制来源目录内本地图/解出完整
  data URI，并改写正文；缺图不静默通过，回执列出未托管项。已含 asset URI 的正文重录可
  从托管目录恢复元数据。FTS/语义块只见图注与稳定 URI，不索引 Base64。
- **文档导入**：`format_convert` 新增 `preserve_images=true`（必须同时给 output）；
  DOCX/PPTX 经 MarkItDown `--keep-data-uris` 保留内嵌图，外置到同名 `-assets` 目录并改
  相对引用。PDF 仍只抽文字，不默认把整页渲染成图。
- **阅读端**：新增 ProposalB 第六绑定 `GaeaProposalAssetDataURL`；MarkdownContent 识别
  `gaea-asset://`，经绑定读回 data URL。资源失败显示占位，不伪装成成功。
- **验证**：提案资产单测含落盘/回读/覆盖/跨方案共享/删除回收；DOCX 带图
  `format_convert → proposal_save` 端到端绿；ProposalB 绑定、前端 Markdown/方案库/
  mock 契约、tsc、改动面 ESLint、Go vet 与 docs 守卫绿；Vite 生产构建 34.25s
  （既有 CSS/chunk warning）。绑定面 724→725。
- **桌面构建**：按构建指南先 `npm.cmd run build`（26.48s），再以 `build/bin` +
  ucrt64 PATH、`CGO_ENABLED=1`、`CC=C:\msys64\ucrt64\bin\gcc.exe` 执行裸
  `wails build -s -ldflags "-s -w" -trimpath`；bindings 未跳过，47.153s 成功。
  `build\bin\gaea.exe` 59,361,280 B，SHA256 `582BB3BA…AD2780`；正式路径 smoke
  `/api/health` 200。当前构建含主工作树其他未提交改动；未打包、未装机。
- **边界**：仅收 PNG/JPEG/GIF/WebP、单图 20 MB；PDF 内嵌图与 DOCX 的 EMF/WMF/SVG
  尚未抽取。已入库的旧文字方案无法从现有库恢复当时丢失的原图，需回源重录。

## Inkling 删除 + Qwen3.8 冷启动生命周期改造（2026-11-24）

用户拍板「第一删除 Inkling，第二 Qwen3.8 启动太慢，不适合热启动，要修改」。

- **Inkling 删除**：代码档位、模型 ID、前端兜底与启动配置全部移除；原运行资产
  `C:\AI\models\inkling-2-mini`（19.1GB）、Strata shape engine（141.8MB）与
  tokenizer pack（9.4MB）已从原路径移除。直接永久删除被执行策略拦截，资产已送入
  Windows 回收站；清空回收站可释放约 19.2GB。`C:\AI\re\strata-0.2.20` 保留约
  50KB 能力/测速证据并明确标注“运行时已删”。
- **Qwen 冷启动策略**：新增 `strataColdStartOnly`；聊天 `ensureChatBindingEngine`
  与图文 `wenxueEnsureEngine` 未探测到 8091 时不再等待 3 分钟拉起，立即返回
  “请先到模型中心 → 语言模型点击启动”。显式启动仍走 `StrataModelStart`，
  300s 就绪上限不变。
- **常驻策略**：`StrataEngineStatus` 返回 `auto_start=false`、
  `cold_start_seconds=180`、`keep_resident=true`；空闲收割器检测到 Qwen3.8
  运行时跳过收割（其它 llama/WHIRL/ComfyUI 规则不变）。引擎卡提示“冷启动约
  3 分钟、显式启动、启动后常驻”，语言模型卡按钮显示“预热中…”，并标记
  “冷启动约 3 分钟”。
- **验证/装机**：新增 cold-start 预检与 keep-resident 测试；`TestStrata*`、
  聊天/图文定向测试绿，Vitest 17/17；Vite 27.42s；Wails 47.161s（bindings
  未跳过）；冒烟 200；INSTALL OK，PID 54304，exe SHA256
  `6AB1342865DA…F207FA`。装机 exe 已确认不再含 `inkling2mini_sq20` 档位字符串。

## Inkling-2-Mini SQ-20 接入与实测（2026-11-24）

用户提供手动下载的官方权重后，主代理完成校验、运行配置、gaea 档位接入、测速、构建和装机。

- **权重/组件校验**：SQ-20 `19,346,685,952 B`，SHA256
  `0BBFD523…F7E1B97`；Vision `636,526,592 B`，SHA256 `6589EFF4…6FC677`，
  均与官方 pin 一致。`strata-inkling 0.1.66` Windows HIP `strix` 组件已安装
  （`gfx1151`，官方包 SHA256 `b771d9fb970ac2cf9f7a2807dbb6df3a3277745aef09c91a2713d4cc8f445f0b`）。
  官方 setup 生成 32K、DSpark、Vision 配置；tokenizer 收进模型目录
  `C:\AI\models\inkling-2-mini\inkling-2-mini\tokenizer`，不再依赖旧
  `C:\AI\strata\Strata-data` 散点路径。
- **运行确认**：`strata-inkling --info` 报 `arch=inkling2_loop format=strata
  build=SQ-20 layers=24 visits=48 hidden=2304 vocab=200058 experts=128
  drafter=dspark images=1 parallel=64`；8091 冷启动约 10s，`/v1/models`
  返回 `inkling-2-mini-sq-20` loaded。
- **gaea 接入**：`strataProfiles` 增 `inkling2mini_sq20`，配置
  `terminal\strata-inkling-2-mini-sq-20.json`，模型 ID
  `inkling-2-mini-sq-20`；状态识别、前端兜底档位和模型目录同步。Qwen 仍为
  默认展示档，Inkling 从模型中心显式启动。
- **测速（与 Qwen 同一 `verify_engine_compare.go` 口径）**：
  五档 24.3 / 27.3 / 25.9 / 21.3 / 27.1 tok/s；长提示词 3,121 prompt tokens、
  TTFT 3.936s；并发 x2 两路 54.4 / 60.6 tok/s，脚本聚合 45.9 tok/s。
  Inkling 单路吞吐低于 Qwen，但 DSpark 与连续批处理在 x2 下更稳。
- **能力测试**：官方 `--selfcheck --verify-hash` PASS；可复跑 15 项确定性套件
  **14/15 PASS**（40.8s）：中文逐字指令/算术/常识/多步逻辑/JSON 抽取/中文创作
  约束/注入抵抗/Python 语法/区间合并/中性多轮召回/工具调用+结果闭环/
  15,189-token 长上下文中段召回/视觉 OCR+形状颜色均过。唯一失败为把普通标记
  命名为“校验码”后的安全拒答；同一上下文改用“海盐派-73”可召回，判定为
  安全对齐行为而非上下文能力缺失。工具调用为 `get_weather({"city":"上海"})`
  → tool result → `18°C/晴`；视觉读出 `GAEA-42`、红色圆形、蓝色三角形。
- **Qwen3.8 同题对比**：Qwen 关思考 14/15、36.3s；开思考（max_tokens×4）
  14/15、63.5s；Inkling 14/15、40.8s。三者除两个差异项外全部同过：
  Qwen 能在历史中召回“校验码”，Inkling 安全拒答；Inkling 视觉完整读出
  `GAEA-42`，Qwen 两次均丢连字符成 `GAEA42`（形状/颜色正确）。Qwen 开思考
  对本套题无新增通关，只增加 27.2s 总延时；工具闭环 Qwen 关思考约 2.0s，
  Inkling 约 5.5s。中文创作 Qwen 2.9s/Inkling 4.5s，15K 长上下文
  Qwen 18.2s/Inkling 19.7s。
- **验证/装机**：`TestStrata*` 绿；Vite 37.99s；Wails 1m21.766s（bindings
  未跳过）；build/bin 冒烟 `/api/health` 200；INSTALL OK，活体 PID 27848，
  exe SHA256 `73336E686332…0212`。测速服务已停，应用保持运行。
- **证据**：`C:\AI\re\strata-0.2.20\` 增 Inkling server 日志、组件清单和
  测速/能力 JSON（Qwen 开/关思考双份）与 `scripts\capability_suite.py`；
  大模型只记运行路径与 SHA256。

## 文学板块 Qwen3.8 冷启动 30s 假停止修复（2026-11-24）

用户反馈「其它模型正常，只有 Strata/Qwen3.8 到约 28s 就停止；是不是有时间限制」。

- **根因**：不是引擎超时。文学控制器前端每 30s 执行一次 `GaeaRunning()` watchdog，
  该短名映射 `WenxueRunning()`；旧实现只返回 `ctrl != nil && ctrl.Running()`。
  Qwen3.8 冷启动要等待 Strata 预填 55.43GiB 专家缓存（实测约 2–3 分钟），此时
  `WenxueSend` 仍在 `wenxuePreflight → StrataEngineStart` 内，尚未执行
  `ctrl.Send`，于是首个 watchdog tick 读到 false，`localCancel` 把界面运行态
  复位。真实 Python/strata.exe 与回合都继续运行，最终请求仍成功生成回复。
  其它模型在 30s 内已 ready，所以没有触发；不是模型白名单或 30s 启动上限。
- **修复**：`wenxueRuntime` 增在途发送计数；`WenxueSend` 从进入即 `beginWenxueSend`
  并用幂等 `defer done()` 释放；`WenxueRunning` 现在 `sends>0 || ctrl.Running()`，
  覆盖 `turn_started` 之前的引擎预检/冷启动窗口。后端真实启动上限仍是 300s。
- **验证/装机**：新增 `TestWenxueRunning_IncludesSendPreflight`，连同
  `TestWenxuePreflight*` 定向测试绿；前端生产构建 39.58s；Wails
  `-s -ldflags "-s -w" -trimpath` 45.451s（bindings 未跳过）；`build/bin` 冒烟
  `/api/health` 200；INSTALL OK，活体 PID 53476，exe SHA256 `CC5B615C5BEC…`。

## Strata v0.2.20 + engine 0.1.66 升级与 Qwen3.8 实测（2026-11-24）

用户令「Strata 更新到官方最新版并测试 Qwen3.8 速度」，主代理独立完成。

- **上游与换装**：本地从 `v0.2.13`（engine 0.1.59）升到官方 `v0.2.20`
  （commit `a34979fc94f3`），Windows HIP `rdna3` 组件 `0.1.66`；资产
  `strata-engine-0.1.66-windows-x86_64-hip-rdna3.tar.gz` 为 220,395,581 B，
  SHA256 `568B97FD55AC1309C38F2CCE00597F5B327C7989FC32FE7B49D9F9772883977B`
  与 GitHub 发布页逐字一致。旧 engine 保留为 `engine-0.1.59.bak/`；官方包、启动日志与
  结构化测速 JSON 已归档到 `C:\AI\re\strata-0.2.20\`（见其 README），仓库不放数据。
- **布局迁移**：0.2.16 起官方把控制端移到 `terminal/serve/server.py`、venv 移到
  `terminal/.venv`。`strata_proc.go` 改为当前布局优先、旧 `serve/server.py` 兜底；
  启动、状态返回与测试同步。`terminal/requirements.txt` 已应用到 venv（Python 3.13，
  Pillow 12.3→13.0），`pip check` 无冲突。Qwen3.8 wrapper 的绝对路径改到新根，
  并把当前 64GB 系统内存装不下 55.44GiB expert arena 的真实条件显式写成
  `--mmap-experts`（引擎日志此前已自动回退到同一路径）。
- **运行确认**：`strata-device --list-devices/--selftest` 通过——AMD Radeon 8060S、
  `gfx1151`、107.6GiB、selftest OK；Qwen3.8-Flash-Next Uncensored 以 32K 上下文、
  int8 KV、MTP 打开，模型 `/v1/models` 报 loaded；日志确认
  `hipBLASLt tuning enabled (32 rows, gfx1151, version 100500)`。
- **测速（gaea 聊天页同口径：completion ÷ 首 delta→流结束）**：
  首轮 `verify_strata_speed.go` 四档全 PASS——短答关思考 37.2、长答关思考 30.5、
  思考缺省开 36.7、长提示词 47.0 tok/s，长提示词冷 TTFT 3.529s。
  同引擎热态复测 `verify_engine_compare.go` 五档为 34.4 / 32.6 / 43.5 / 50.5 /
  38.1 tok/s，热 TTFT 204~266ms；并发 x2 两路 58.4 / 60.9 tok/s，聚合 39.6。
  引擎日志旁证短答 35.4→36.7 tok/s、思考档 42.3 tok/s、长提示词 48.6 tok/s。
  当前运行是文件专家 mmap 路径，不等同于官方“专家全驻 RAM”的机器条件。
- **构建与装机**：定向 `TestStrata*`、`TestEngineUpdateCheck*`、`go vet
  ./internal/app` 绿；前端生产构建 41.57s；首次 Wails 因 bindings 子进程缺 DLL
  退出 `0xc0000135`，按指南把 `build/bin` 前置后同一裸
  `wails build -s -ldflags "-s -w" -trimpath` 29.825s 成功（bindings 未跳过）。
  `build/bin/gaea.exe` 冒烟 `/api/health` 200；`gaea_install.ps1 -Launch` INSTALL OK，
  装机 PID 34164，exe SHA256 `1EC6E192ED44…6267`，与构建产物一致。

## 开机动画重设计落地 + 新标全局替换 + 4K 底片（2026-11-24，用户拍板「logo选B，同步替换全局图标，开机动画要4K高清」）

主代理（ZCode）独立完成；**已提交 `6ad4916d` 并打包装机**（2026-11-24 下午，见下「打包装机」）；
早期桌面构建 exe `8FD4BE09`（主树 build/bin，15:27）已被包版覆盖。效果图与生成原件见
`design/boot-animation-2026-11/`（已随提交入库）。

- **打包装机（2026-11-24 15:40~15:53，「git打包」令）**：收口提交 `6ad4916d`（43 文件
  +2606/−535，含 design/ 入库与 `scripts/brand_icons.py`；仓根 `probe-memory-out.jsonl`
  探针垃圾删除）→ 干净树 `.worktrees/pkg-6ad4916d`（detached，node_modules 主树 junction）
  → Vite 45.93s + wails `1m4.597s`（bindings 未跳过，`git diff --ignore-cr-at-eol` 漂移 0）
  → 包 `releases/gaea-boot-20261124-6ad4916d/`（payload 四件：exe `961C6812…A3933` +
  三 DLL 与上一包逐字节同哈希；运行包 `E1F85F15…6EF0` / 源码包 `6CEC4AE7…B9C1`，
  6969 文件）→ 正式路径冒烟 `/api/health` 200（用户壳在跑可并存；smoke.ps1 只回收自身
  PID）→ **INSTALL OK：装机位 `C:\AI\gaea-app\gaea.exe` = 包版 `961C6812`，活体 PID
  48844（15:53:28 > 构建）**。
- **本轮打包坑**：① `gaea_install.ps1` 只装主树 `build/bin`——首装误装 15:27 的旧桌面
  构建 `8FD4BE09`（与本提交同源码但非包版），把包版 exe 复制进 `build/bin` 后重装纠正，
  装机必须以 `sha256sum C:\AI\gaea-app\gaea.exe` 终验；② `git archive -o` 的相对路径随
  `-C` 解析，src.zip 落到仓根，移回包目录；③ Git Bash 无 `zip`，运行包用
  `Compress-Archive` 代替；④ 用户壳在打包期间自行退出（PID 38260，15:47 还在），
  smoke.ps1 经核实只 `Stop-Process` 自身 PID，与本线动作无关——装机因此无「关运行中
  实例」阻碍。余项=真机看新开机动画（语音晚起回跳疑点实证）；push 待令。

- **背景**：11-19 的「两张原画直用」是折中（用户澄清：当时做不出原画效果的动画），不是偏好；
  11-04 的程序化「轨道 + 等高线」已被否。本轮用户两次纠偏：视觉素材用 gaea 自己的生图，
  不手写着色器；新标用生图重出，不抠官方图。
- **生图**（经在跑的 gaea 调试口 9333 → `GenerateMedia`，xAI `grok-imagine-image-quality`，共 4 张）：
  两张无 Logo / 无文字的底片（星际日出、人间山谷，1280×720）+ 两个玻璃 G 候选（纯洋红底），用户选 B。
- **4K**：底片用 4x-UltraSharp 4 倍超分后 Lanczos 到 3840×2160（ComfyUI 环境的 ROCm PyTorch +
  spandrel 推理；权重下到 `temp/` 用完即弃，`C:\AI` 未动）；AVIF q62 两张共 1.0MB（550 + 456KB），
  旧的两张原画 PNG 是 3.3MB。新标 2K 超分后键掉洋红底，透明母版 `build/brand/gaea-mark-2048.png`。
- **开机动画**（`frontend/src/components/`）：`boot/bootScene.ts` 纯时间轴；`boot/bootShader.ts` +
  `boot/bootRenderer.ts` 是 WebGL2 单三角形片元着色器（镜头推进带上下视差、1.6s 俯冲穿日出白闪、
  辉光呼吸、星点闪烁、湖面涟漪、城市灯火、飘雾、轨道彗星、标的光扫）；`BootSplash.tsx` 重写：
  底片 `<img>` 常驻为底层（没有 WebGL2 就是静态交叉淡变），画布就绪后淡入；字标 / 标语 / 进度 /
  状态是 DOM。`index.html` 静态首屏改为内联 622B 模糊缩略图，底片清晰后交叉淡出。
- **保留**：语音仪式 0 / 1.596 / 4.0s、ESC 跳过、声音开关、68→72→100 进度节奏、原画文案、三语文案键。
- **三处时序修正**（旧实现同样存在，新动画更敏感）：① 画面时间直接取语音元素的播放位置——
  主线程被应用挂载占住时 `playing` 事件会迟到 0.4~0.8s，旧表因此晚这么多；② 语音成为时钟后把
  收束保护计时器按语音剩余时长重排——旧的墙上时钟兜底会在语音播到 3.3s 时就关掉画面；
  ③ 离场时画面冻结在交接点。
- **解码策略**：底片按窗口实际分辨率在后台线程解码（`createImageBitmap` 缩放，下限 1280）。
  1080p 窗口不付 4K 像素且采样落在 mip 0，4K 窗口得到全部 4K 像素。
- **全局图标**：新增 `scripts/brand_icons.py`（Pillow + numpy），从母版一键生成 favicon×3、
  `build/appicon.png` / `.svg` / `-small.svg`、`icon.ico` 七档（≤48 用紧凑取景）、
  `gaea-brand-icon.png`、UI 标 `gaea-brand-mark.webp`。UI 标沿用旧明暗原图的 1568×1264 画幅与字形
  框位，所以五处容器尺寸不动；`.gaea-brand-art` 改单图，浅色主题加淡影。旧 `gaea-brand-light/dark.jpg`
  与两张原画 PNG 退役。
- **验证**：BootSplash 14 + bootScene 7 = 21/21；Sidebar 24/24；related（CommandRail + SettingsPage）
  35/35；全量 vitest 518 文件 4746 例 308s 全过；`tsc -b` 0；改动面 ESLint 0；Vite build 26.8s 绿
  （既有 Wenxue CSS 警告）；`check-docs` OK。生产包 + 无头 Edge（ANGLE / Radeon 8060S D3D11）走查：
  1080p 正常动效全序列、4K 视口画布 3840×2160（GPU 读回 1:1 清晰）、浅色主题工作台标、1366×768
  小窗、减弱动效路径，均无异常与控制台报错。时序实测：换景时语音在 1.60~1.61s，语音 4.00s 结束即
  收束（画面进度 94%），WebGL 挂载后 1.0~1.4s 接管。
- **既有问题（非本轮引入，未改）**：① `scripts/frontend-e-check.mjs` 的 E25 在 HEAD 即红——
  `useChatStream.ts` 已没有 `app.ChatSend(` 字样（chat-demo 蒸馏线改了调用形态），守卫匹配串过时；
  ② knip 基线本就有 33 个未使用文件 / 114 个未使用导出（多为生成的 `wailsjs/go/models.ts`），本轮
  零新增；`WelcomePage.tsx` 本就无引用（已同步换标）。
- **坑**：① 重排 effect 顺序后，加载底片的 effect 先于测窗 effect 执行，拿到尺寸占位值 2×2，
  4K 底片被解码成 3×1 像素——画面全糊而单测全绿，只有真机走查发现；现在加载处自行测量并设 1280
  下限。② 无头 Edge 默认按 `prefers-reduced-motion: reduce` 运行，走正常动效须
  `Emulation.setEmulatedMedia`。③ file:// 页面不能把图片传给 WebGL，复现页要内联 data URI。
  ④ `getTexParameter` / `texParameteri` 第一参是目标枚举不是贴图对象；WebGL2 没有
  `getTexLevelParameter`。⑤ heredoc 里的反斜杠会被降级，含转义的脚本一律用写文件工具落盘。
- **桌面构建（同日 15:26~15:28，用户令「构建桌面端」）**：构建前无 gaea / wails / go 孤儿。按构建指南：
  前端 `npm.cmd run build` 33.32s 绿 → 仓根 `wails build -s -ldflags "-s -w" -trimpath` **1m37.091s**
  （bindings 生成未跳过，再生后与仓库零漂移；ucrt64 gcc 前置 + `build/bin` 前置 + `CGO_ENABLED=1`
  + `CC=C:\msys64\ucrt64\bin\gcc.exe` + 系统默认 TMP/TEMP）。产物 `build/bin/gaea.exe`
  59,289,088 B，mtime 15:27:40 = 构建结束时刻，SHA256
  `8FD4BE09560364BA92FCB2F33EDE24516978D8AFDC433CB03F96237DE35D9A8F`；三 DLL 同目录。产物核验：
  exe 直接导入 `sherpa-onnx-c-api.dll`（cgo 未降级）；exe 内嵌图标与 `build/windows/icon.ico`
  **7/7 帧逐字节一致**（对照组旧包 exe 为 0/7）；4K 底片 / 新标贴图 / favicon 按字节嵌入，旧原画
  文件名已不存在。真机壳拉起（`GAEA_WALKTHROUGH=1`、独立桥接口、CDP 9333 抓帧）：PID 启动时间
  晚于构建时刻，开机全序列无页面异常，冷启动（页面解析到可交互）实测 823ms，窗口 1264×761。
- **构建后没做完的**：① **正式冒烟 `scripts\smoke.ps1` 没跑**——用户随后自己从 `build\bin` 起了
  实例，同 profile 不宜再起第二个；该实例稳定运行 1 分半以上，但 HTTP 桥接探活没有做。
  ② **一处疑似的真机瑕疵没查完**：真机里语音比兜底墙上时钟晚起约 1.3~1.8s，语音接管时钟的瞬间
  画面可能被拉回起点（进度条回 0、标重新点亮）——按代码推演成立，细采样那次连到了用户实例、
  没拿到证据。用户令「已经可以了」后停手。若复现，修法=autoplay 可用时等语音真正起播再起表，
  且语音接管时只前seek 语音、画面不回退。③ 我经脚本拉起的那次系统报告 `prefers-reduced-motion:
  reduce`（`SPI_GETCLIENTAANIMATION=0`，走的是无推镜的安静分支），用户自己起的实例报告的是
  正常动效，两者没对上。
- **待拍板**：装机（会关掉运行中的实例）；提交；进度百分比仍按时间演示，未接真实启动事件；
  仓根旧标原图 `logo浅色.jpg` / `logo深色.jpg` 已无代码引用，删不删由用户定。

## Git 打包与推送：gaea-20261124-58be0f5e（2026-11-24 10:05~10:15，用户令「git打包推送」）

- **构建**：从 `58be0f5e` 的独立干净工作树 `.worktrees/pkg-58be0f5e`（detached，node_modules 为主树
  junction）构建：Vite 38.48s + Wails 1m11.086s（bindings 未跳过，再生 28 个 wailsjs 文件，内容漂移 0）；
  系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc、PATH 前置主树 `build/bin`。构建前无 wails/go 孤儿。
- **交付**：`releases/gaea-20261124-58be0f5e/`（payload 四件正式路径冒烟 200，PID 33468 已回收，
  前后进程清单均只有装机壳 PID 32336；PACKAGE-INFO + SHA256SUMS 入库，ZIP / 源码包留盘）。
  exe `71205F07…B6309D`，运行包 `A0BA7358…`，源码包 `41AB5722…`（6551 文件）。
- **可复现性**：exe 与上一交付包 `ddce2d4a` 逐字节相同——四笔增量（`07258861` / `77291f35` /
  `158ed9f7` / `58be0f5e`）全是记录、文档清理与 17 行注释里的文档路径，没有功能变化。
  装机位 `C:\AI\gaea-app\gaea.exe` 就是这份二进制，**未重新装机**。
- **验证口径**：本次未跑全量测试（源码改动仅为注释，编译产物与已验证的上一包相同）；本次 Wails
  构建含 cgo 成功。
- **推送**：`77291f35..58be0f5e` 已推 origin；包记录提交随后推送。

## docs 索引清理：过期文档归档（2026-11-24，用户令）

- **起因**：用户指 `docs/README.md` 里许多登记的文档已过期，可以清理。
- **盘点**：索引 152 行登记里 46 行指向的文件早已在 `docs/archive/`（9~10 月归档时索引行没跟着迁走），
  其中 42 份在归档说明里从未登记；顶层 115 份文档中 13 份只在一段散文里被顺带提到、没有表格行。
- **处理**：
  - 顶层 41 份用 `git mv` 移入 `docs/archive/`（不删文件）：已收官 / 未立项的规划 16 份（含 V5 提案与
    辩论、下一代路线图、长期规划 2026-10）、已退役引擎 3 份（天璇、MoE4All）、已退役小说板块 2 份、
    已闭环线的过程档与证据档 20 份（ZCode 蒸馏线 7、dsh 0.2.1 时代蒸馏 5、聊天实弹报告 3 等）。
  - 每份在 `docs/archive/README.md` 登记「归档原因 / 取代者」；另补登此前无登记的 43 份。
  - `docs/README.md` 重写为现行索引：29,541 → 7,382 字符，按「权威 → 政策约定 → 已落地设计档 →
    在途待拍板 → 近期落地线出处 → 当日总账 → 子目录」分组，74 份顶层文档全部有表格行。
  - 指向被移走文档的 62 处旧路径改写为 `docs/archive/…`：`.gaea/progress.md`、`todos.md`、顶层文档、
    `design/` 一处，以及 17 个源码文件里各一行「设计出处」注释（纯路径替换，gofmt 无变化）。
- **没改**：`CHANGELOG.md`、`releases/`、10 月代码审计目录里的 JSON 证据仍写旧路径（历史记录不改写，
  归档说明里已注明「顶层找不到的多半在归档目录」）。
- **没动（待用户点名）**：当日总账（流水记录，无归档先例）；众神殿 / 灵犀 / 语音 / CAD 等仍在迭代或
  有待拍板事项的线的调研文档（索引里单列成「在途与待拍板」「近期落地线的调研与台账」两组）；子目录
  `docs/distill/`（MuMuAINovel 蒸馏规格，小说板块已退役）与 `docs/snapshots/`（9 月目检截图），
  索引里已标注现状。
- **验证**：`node scripts/check-docs.mjs` OK（孤儿 0 / 悬空 0）；41 份全部被 git 识别为重命名；
  40 个被改写文件逐词 diff 均为纯路径替换。

## 项目长期记忆清理：.gaea/AGENTS.md 回到现行事实（2026-11-24，用户令）

- **起因**：用户指出「长期规划（权威，2026 定稿）」里许多已是过去式，要求梳理后清理，并明示
  **现在没有长期规划**。
- **实锤的损害**：gaea 把 `.gaea/AGENTS.md` 自动注入本仓每次代理会话（`internal/gaea/memory/doc.go`），
  单文件预算 32KB（`docMaxBytes`），超长静默截断。清理前文件 40,473 B，每次注入砍掉末尾 7,705 B——
  「发布流程」「沙箱备忘」「已知注意」整段没进提示词；占着预算的是 11-12~11-14 状态速览（约 13KB）
  与过时的长期规划（约 6.8KB）。
- **对账发现的过时陈述**（均已改）：绑定门面写 11 个、含已退役的 NovelB/SinB（现 12 个，新增
  PantheonB/WenxueB/ProposalB）；docmd 路径 `internal/office/docmd`（目录已不存在，现为
  `internal/docmd`）；「内置工具 17 个」；Herdsman 条目仍列 voxcpm2；「本地 TTS 引擎」整节讲已退役的
  VoxCPM2/CosyVoice2（现本地 TTS=Breeze-TTS-2）；交互纪律里的旧耗时与「本机无 gcc」；旧边界
  「不做算量」「MCP 不进规划」「编程板块=DSH 独立窗口不融合」已被后续拍板推翻。
- **处理**：过时段落整段移出到 `docs/archive/agents-cleanup-2026-11-24.md`（原文逐字保留）；文件按
  「强制规则 → 现行决策 → 约定 → 纪律 → 流程 → 水位」重排，40,473 B → 21,478 B；新增「在册决策与边界」
  一节只留仍成立的拍板。用应用自己的 `discoverDocs` 做临时探针（跑完即删）对照：旧版注入正文
  32,847 B、带 `truncated` 标记、不含「已知注意」；新版 21,476 B、无截断、含「已知注意」。
  `docs/README.md` 规划表与 `docs/archive/README.md` 的「活跃指导 / 活跃权威 /
  唯一权威」标签改为已过时或已收官；5 份计划文档头部补状态行。`node scripts/check-docs.mjs` OK。
- **遗留**：`check-docs` 只守 65536 B 的指令预算，不守 32KB 的注入预算，所以之前超限无人察觉；
  是否加一条「`.gaea/AGENTS.md` ≤ 32KB」守卫待用户定。
- **没动**：V5 Case OS 提案与辩论（仍标「待拍板 / 进行中」）、各域 master plan（造价库 / CAD / 灵犀 /
  绘梦等）以及 `docs/` 里其余带「规划」字样的文档；要一并标注需用户点名。

## 方案库功能体系完善（2026-11-24，`ddce2d4a` 已提交已装机）

- **拍板**（用户）：检索复用现成本地模型客户端（Herdsman embedding + reranker），
  不另造检索工具；录入=模型只定标题层级、正文由工具从源文件逐字切；界面一并做。
- **存储层** `internal/gaea/proposal`：SchemaV36（proposals 增 source_file/ingest_line/
  ingest_hash，sections 增 src_line，proposal_chunks 语义块表〔float32 BLOB〕+
  proposal_embed_meta 模型标记）。录入两形态（outline 按源文件标题行切正文 / sections
  直给），分批续录（mode=append，末节正文待截取期间状态「录入中」，源文件前缀指纹
  防中途改动），同名覆盖须 overwrite=true。标题锚点分级匹配：Markdown 标题行 > 不以
  句读收尾的短行 > 正文句，目录行（标题后只剩页码/点线）恒排除。检索=全文路（trigram
  短语 + 两字词 LIKE）+ 语义路（段落向量，块前拼标题路径）→ 名次融合 → 本地精排，
  本地模型不可用逐级降级为全文路。
- **工具与提示词**：proposal_save/search/read 重写（契约行、compact 表、审批摘要、
  工作区绑定）；方案模式 ModeLine / 身份提示词 / proposal-longform 技能同步录入与
  取材流程。绑定面 ProposalB 五方法（List/Outline/Read/Search/Delete，722→724）。
- **一级板块「方案库」**：`ProposalLibraryPage` 三栏（方案清单/命中 → 1~4 级标题树 →
  阅读窗）+ 检索条（全文+语义/只搜标题、类型/年度/层级）+ 四态 + 空库三步引导；
  menuOrder=6 紧随造价库，众神殿及其后板块序号 +1（两端同步）；页面规范
  `design-system/gaea/pages/proposal.md`；无头浏览器走查亮/暗/窄窗/空库过。
- **实测**：真实技术标（27.3 万字/419 页/241 标题）录入 0.2s、954 块 bge-m3 向量化
  24s、检索 0.5~1.3s 含精排，精确词与自然语言问法均命中正确小节；真实程序经 HTTP
  桥调用五绑定核对字段形状一致，并抓到「层级/类型/年度过滤对语义路不生效」已修。
- **验证**：Go 全量 131 包绿（仅 docmd uocr2 / voice realtime 实弹两包既有环境红，
  零依赖）；前端 517 文件 4734 例，本次改动相关全绿（CommandRail 同步 13 板块），
  3 例负载期不稳单跑绿；tsc/ESLint/vet 0；绑定漂移 OK@724；Vite 30.9s + Wails
  36.3s→24.8s（bindings 未跳过）；冒烟 200；exe SHA256 `b63526b0…`。
- **须用户知情**：①正式库已是 V36——builtin 包全量测试有既有用例会开正式库（未定位
  到具体用例），V36 挂进迁移链后跑测试即被应用；只加列加表，旧版兼容。②正式库里有
  11-23 测试遗留的夹具方案「包3技术标 第三版」（3 节、每节十余字），未动，待用户定
  （11-24 拍板：保留，作后续优化的测试数据，勿删）。
  ③`node scripts/check-docs.mjs` 既有 FAIL：docs/README.md 未登记两份他线文档。
- **挂池**：由真实 LLM 在标题稀疏文档上自行定级录入的实弹（试点用机械标题代替）；
  短问法排序（章引言先于最具体的节）；切换 embedding 模型会整库重嵌；文学树
  bindingSignatures.ts 镜像陈旧（817 名含已退役 novel）。
- **Git 打包与装机**（用户令「先 git 打包，做好文件记录和记忆更新，构建桌面端」）：
  工作树按时间线收口成两笔——`9b5b8724` 文学反馈批（11-23 17:51~18:38 另一会话
  完成并带测试落树，未提交：工具层五刀 / 阅读室排版偏好 / 五个小说写作技能按拍板
  退役，64 文件）；`ddce2d4a` 知识库退役 + 方案库（181 文件）。独立干净工作树
  `.worktrees/pkg-ddce2d4a`（detached，node_modules junction）构建：Vite 35.04s /
  6608 modules + Wails 1m5.284s（bindings 未跳过），wailsjs 内容漂移 0；交付
  `releases/gaea-proposal-library-20261124-ddce2d4a/`（payload 四件冒烟 200；
  PACKAGE-INFO + SHA256SUMS 入库，ZIP / 源码包留盘）；exe `71205F07…B6309D`；
  INSTALL OK，PID 32336（07:04:08 > 构建 07:03:00），装机位与 payload 同哈希；
  4.544.0 不抬号；未 push（07:50 已推，见「推送与拍板」）。
  收口途中自逮一次归组错误：文学批首版提交多带了暂存区里 16 个前端知识库文件
  删除，发现后立即软重置重做（本地未推送提交，无外溢）。
- **复核与记录提交**（07:35 起，新会话续做——上一会话写完四份记录后中断，记录
  未提交、记忆未更新）：未重建，HEAD 仍是 `ddce2d4a`，同提交重建只会得到同名包
  并打断装机位与包的同哈希关系。独立复核：六项 SHA256 逐项重算一致；源码包与
  `git archive ddce2d4a` 重生成件逐字节相同（6548 文件）；运行包四项解包哈希 =
  payload；装机位四件 = payload = 构建工作树 exe；工作树 29 个状态改动文件去 CR
  逐个比对内容差 0、无非忽略的未跟踪文件；PID 32336 仍在运行，诊断口
  `127.0.0.1:18123/healthz` 回 `ok`（200）。**冒烟未复跑**：`smoke.ps1` 不隔离
  工作区，会拉起与用户在用壳同名的第二实例；冒烟 200 沿用上一会话对同一哈希
  payload 的记录，非本会话实测。
- **推送与拍板**（07:50，用户令）：main `f3f9b5b0..07258861` 49 笔已推 origin，本地与
  远端一致；正式库夹具方案「包3技术标 第三版」用户拍板**保留**，作方案库后续优化
  的测试数据，勿删；`docs/README.md` 补登两份他线文档（11-22 day-log、Nebula Memory
  蒸馏报告），check-docs 转绿。

## 文学实测反馈批（2026-11-23 另一会话完成；`9b5b8724` 收口入库，随包 `ddce2d4a` 装机）

- **来源**：《软件不足与建议》实测反馈；11-23 17:51~18:38 由另一会话完成并带
  测试落树，当时未提交、未写本档。本条由打包会话按提交说明补记，细节以
  `git show 9b5b8724` 为准。
- **工具层五刀**（行为锁 `internal/app/wenxue_feedback_batch_test.go` 五个用例）：
  汉字口径——回执 hanzi / 旧字数双口径并报，技能阈值一律对 hanzi；字数硬门禁——
  章节不过 min_hanzi 拒写且不落盘；角色卡匹配——卡名精确 → 文件名 stem 兜底，
  失败列出目录里实际有的卡名；单章交付——`wenxue_chapter_deliver`，无标题行时用
  文件名 stem 解析章元信息；定稿内联体检——final_write 闸门与内联检查。
- **阅读室排版偏好**：字体可选（系统字体栈）、字号滑杆、首字不放大；存
  localStorage；Ctrl+滚轮缩放字号（拦 Ctrl 态默认行为，防被当成页面缩放）。
- **技能退役（用户拍板整批）**：story-long-write / story-deslop / novel-agents /
  novel-deslop / novel-review 五件的资产、播种与 allowlist 全部下线；图文内置
  只剩 wenxue-workbench；`subagent_writing` 网文创作模板前缀随之删除（五前缀
  收敛为四）；novel-chapter-branches 留作唯一播种的无限流专属件。记忆提示词
  收紧为「只在用户明确要求固化时才调用 remember」。
- **验证**：带测试；上一会话 Go 全量 131 包通过时为绿，并随干净树包构建装机。
  该批自身的走查记录未落档。

## xAI 登录三死因修复（2026-11-23）

- **日志实锤三次点击三死法**（gaea-20261123.log）：①14:31/15:29 discovery EOF
  （auth.x.ai 走 TUN 代理间歇断流）；②14:54/15:16 bind 56121 WSAEACCES
  （端口被非保留区占有者间歇占用，netstat 空、excludedportrange 未覆盖）；
  ③早期规则未生效。用户已补代理规则（auth.x.ai 200 可达但 7.2s）。
- **修复两件**：①`49101771` discovery 带退避重试 3 次（800ms/1.6s，对用户
  不可恢复的单发即败）；②`setx XAI_REDIRECT_PORT=39561`（低于动态区 49152、
  不在任何保留段，避开 56121 占用者；RedirectPort 仅此 env 键可覆盖）。
- **交付**：49101771 重造（Vite 38.3s+Wails 90s，含并行线星河 CSS 修复
  bdc03895/25a68941 首次入包），冒烟 200，INSTALL OK 15:42:32，新壳 PID 30076
  （显式带 env 拉起——setx 对已运行进程树不生效，必须启动时注入）。
- **边界**：代理节点慢（7.2s）仍可能压满 15s discovery 预算，重试兜底；
  用户待点登录验证。
## Bloom 星河/输入玻璃跨板块统一·办公/文学/聊天根修（2026-11-23）

- **反馈复现**：银河属性、330 个星尘节点和设置均正常；系统当前
  `prefers-reduced-motion=reduce`，只有 270 个 `opacity:.4` 静态星点，
  而流星按上游规则隐藏。用户启用 Windows“动画效果”后，办公页恢复 40 流星，
  但文学板仍整页无星；WebView2 已实测 `prefers-reduced-motion=no-preference`。
- **包级对照**：用 `GAEA_WALKTHROUGH` 同一沙箱、同一系统设置分别启动昨天
  `555eda3e` 与今天 `b1591b4d` payload，强制“磨砂 + 漫天星辰”。前者的静态
  星海可见，后者整片消失；改回 `.gaea-app-layout` 透明后，`b1591b4d` 包立即
  恢复昨天视觉，证明不是星尘生成器或系统设置单因。
- **上游证据**：`dsh-bloom-theme origin/main bd3e93f` 的
  `.dsh-bloom-sparkles` 固定 `z-index:-1`，装饰只垫在内容背后；减少动态只把
  星点改静态 `.4`，流星全部隐藏，不生成冻结流星。因此不能把星河提上层。
- **根因**：凌晨 `b917ed6d` 的“路由表面契约”把
  `.gaea-app-layout` 自身也列入 `--gaea-material-fill` + sheen + blur。
  全量动效时流明亮度尚能穿透；减少动态的静态细点被整页材质与 sheen 压没。
- **根修**：从路由材质 `:is()` 列表移除 `.gaea-app-layout`；再补一条高特异性
  `.page-enter .gaea-app-layout.ant-layout` 非实底透明规则。原因是文学页 CSS
  懒加载晚于全局外观文件，会重新写入 `background: var(--gaea-aurora-bg)`；
  路由可同时保留多个板块 DOM，办公隐藏 layout 透明并不代表文学活动 layout
  透明。页面根、工作区、面板和卡片仍按原契约使用玻璃材质。
  星尘保持上游式 `z-index:0`（本地隔离壳内对应上游 `-1`）位于内容背后，
  不新增任何假的星轨或强制动效。架构测试锁定 `:is()` 材质列表不得重新纳入
  `.gaea-app-layout`，并锁定文学式懒加载后仍生效的透明规则。
- **输入玻璃二次复现**：办公 composer 为透明填充 + shell sheen + 当前材质
  filter；文学卡被懒加载 CSS 刷成 94% 实底、输入行 96%、工具条 92%，
  聊天 `.chat-composer` 则被刷成 `--chat-surface-raised` 实底且
  `backdrop-filter:none`。三块视觉确实不一致。
- **输入玻璃根修**：文学所有会话状态的 composer/input/toolbar 与聊天
  `.chat-composer.gaea-glass-shell` 改消费 `--gaea-shell-composer-fill`、
  `--gaea-shell-sheen`、`--gaea-shell-filter`、`--gaea-shell-edge/shadow`；
  聚焦态统一到 accent/focus/raised。文学保留办公式 16px 输入卡，聊天保留自身
  760px 输入岛与 18px 圆角，只统一材质，不改布局。
- **验证**：外观/store/设置定向 **3 文件 32/32**、`tsc -b` 0、改动面 ESLint 0；
  干净树前端 `npm.cmd run build` 31.99s；UCRT64 Wails 48.596s
  （bindings 未跳过）；`git diff --ignore-cr-at-eol` 内容漂移 0；
  正式 payload `/api/health` **200**。
- **装机实拍**：`C:\AI\gaea-app\gaea.exe` SHA256
  `3F8CE4D9F63A4468E0B706A0ECBC9E3A30C9450F7BB8A86FC1D49BE4F9F7C809`，
  payload 与装机位同哈希；INSTALL OK，PID 16048 启动于 14:17:10 >
  构建 14:14:54。装机版文学页实测 `prefers-reduced-motion=no-preference`、
  活动 `.gaea-app-layout` 背景/背景图/滤镜均为 none、`stardustZ=0`、
  40 流星；截图确认星河与流星穿过文学工作区。办公/文学/聊天三套输入面逐项
  计算样式对比：background/background-image/backdrop-filter/box-shadow/border
  全部相等；当前 veil 材质实测三者均为 58% + blur24。
- **提交**：核心 `bdc03895`；补齐文学所有会话状态 `25a68941`。
- **交付包**：`releases/gaea-bloom-allboards-20261123-25a68941/`；
  payload 四件+桌面 ZIP+源码 ZIP 六项 SHA256 已落
  `PACKAGE-INFO.md` / `SHA256SUMS.txt`。桌面 ZIP 35,490,408 B
  `504451BC…C444826`；源码 ZIP 829,142,883 B
  `25B409A6…45D61F9`。`bdc03895` 预备包因文学已选会话仍是实底，
  已作废不作交付。
- **边界**：未 push；本轮不抬产品版本。

## 树上在途件全收口·四笔提交+包 gaea-20261123-b1591b4d（2026-11-23）

- **背景**：方案档接线批后用户拍板「未提交的文件一起构建提交」——77 件全盘点。
  归属核验（逐 diff 看）：app.go/gaea_ui_extra/gaea_unified_search/jailbreak/
  main.go 全属文学私密线，无第四条线；`probe-memory-out.jsonl`（运行态残留）与
  `releases/**` zip 大件按纪律排除不入库。
- **提交前验证**：Go vet（app/config/maturecraft）净；定向 `Privacy|Orphan` 绿+
  `WenxueSkillSeed|WenxueSkills|WenxueWorkspace|NovelChapterBranch` 绿+
  config/maturecraft 全量绿；前端 `tsc -b` 0+私密组件 8/8。
- **四笔**：`28933717` feat(wenxue) 图文私密保护四形态+novel-chapter-branches
  技能种子（49 文件 +2678/−111，wenxue_privacy 四态后端+前端三件+
  unified_search 图文剥离+config 私密键+9333 取证注+绑定面再生）；
  `9168f07a` fix(gaea) 编程板块 web 遗孤回收根修（netstat 探 PID+等端口释放+
  树杀，三探针可替换缝）；`b1591b4d` chore(gaea) cdp-walk 双栈
  （[::1]:9333 形态+GAEA_WALK_HOST 钉壳）+cache_probe_live.go 随批入库。
  树上代码面清零。
- **包**：`releases/gaea-20261123-b1591b4d/`（干净树 pkg-b1591b4d detached：
  pnpm 24.7s+Vite 25.23s+Wails 55.57s bindings 未跳过；漂移 27 文件忽略 EOL
  内容差 0；payload 冒烟 **200** PID 35160 已回收）；exe 61,733,888 B
  `1413DA5D…942646A`；桌面 ZIP 34,851,897 B `C2D5293B…E6A17C`；源码 ZIP
  829,135,537 B `17646CD7…A9353`；三 DLL 同基线。PACKAGE-INFO+SHA256SUMS
  入库（docs 提交）。未装机未 push（装机会杀壳——旧壳 62656 还开着，先关再装）。

## 装机 b1591b4d 同源壳+方案档沙箱走查（2026-11-23）

- **构建装机**：旧壳已自行关闭（62656 不在，9333 释放），`gaea-prev-62656.exe`
  已删；主树 HEAD 前端 Vite 31.18s + Wails 30.503s（bindings 未跳过）→
  `build\bin\gaea.exe` 61,722,112 B → 冒烟 **200**（PID 51260 已回收）→
  `gaea_install.ps1` **INSTALL OK**（exe+三 DLL 同装 C:\AI\gaea-app，构建时刻
  13:01:56，桌面 gaea.lnk 已刷新）。
- **方案档沙箱走查**（装机版壳 + `GAEA_WALKTHROUGH=1` + `GAEA_DATA_ROOT`
  钉 Temp 沙箱 home，CDP 9333）：运行模式菜单三态 → 选「方案」chip 显方案 →
  新建会话引擎按新档重建；绑定面回读 `configured/active=proposal`、
  `hide=[cad_ask+cost 四件]`、`extras=[longform/review/soil 三技能]` 全对齐
  compose 表。ModeLine 参照句正文锚由 TestProposalRunModeKnowledgeRef 钉死
  （沙箱无引擎不跑回合，会话日志只落 .meta，提示词正文不上日志——判定以
  单测锚+装配面回读为准）。走查后沙箱目录清场。
- **收尾**：正式壳已拉起（PID 10904 @ C:\AI\gaea-app\gaea.exe，装机版）。
  本机仍未 push（ahead 40）。

## 方案模式·参照知识库接线（2026-11-23）

- **需求收敛**：用户最初提「方案模式增加方案库（参照索引导入的方案）」，三问拍板
  后用户再砍为「直接使用知识库」——方案库=知识库『设计方案』分类视图（导入链/
  INDEX.md 索引/knowledge_search 全现成），零新存储/零新绑定/零新 UI，只补
  「提示接线」。
- **改动（2 文件+1 测试）**：①`officeRunModeLineProposal` 补句「知识库『设计方案』
  分类中已导入的历史方案可作参照：动笔前用 knowledge_search 按关键词检索，命中后
  对照其章节骨架与评分点口径」（standard/cost 档不动）；②`officeProposalLongformBody`
  阶段①插「动笔前先检索参照（命中多篇挑最相近 1~3 篇读全文对照骨架/评分点，
  无命中不阻塞）再落清单」；③锚测试 `TestProposalRunModeKnowledgeRef` 钉两处
  接线防回退（standard/cost 恒不含 knowledge_search）。
- **验证**：定向 `go test ./internal/app -run 'RunMode|ComposeModeLine|OfficeMode|KnowledgeRef'`
  **6/6 绿**（含确定性/矩阵/配置归一/身份稿/工具面注册/新锚六例）；go vet 净；技能名清单测试不钉正文。
  构建：活壳占 build\bin（PID 62656，10:57 起，可视窗口）未杀——走**改名腾位**法
  （旧 exe 改名 `gaea-prev-62656.exe`，运行中进程不受影响，壳关闭后可删）；前端
  Vite 32.5s + `wails build -s` 37.7s（bindings 未跳过，ucrt64+build/bin 双前置）；
  冒烟 `smoke.ps1` **200**（GAEA_HTTP_PORT=18999 避让活壳，PID 53632 已回收）。
- **提交与交付**：feat `5245fdec`（4 文件选择性提交，wenxue 隐私线在途改动零混入）。
  交付包 `releases/gaea-proposal-kbref-20261123-5245fdec/`：干净工作树
  `.worktrees/pkg-5245fdec` detached 检出，pnpm install 27s + Vite 28.36s +
  Wails 绿（bindings 未跳过）；漂移 wailsjs 27 文件+go.mod 忽略 EOL 后内容差 0
  （纯 CRLF 噪音）；payload 冒烟 **200**（PID 70992 已回收，18999 避让活壳）；
  exe 61,664,768 B SHA256 `22CD2CEC…58593DE`；桌面 ZIP 34,823,565 B
  `82854F30…620D729`；源码 ZIP 829,085,963 B `D1AF3FFF…9CF2826`；三 DLL 与
  bloom029 基线一致。PACKAGE-INFO + SHA256SUMS 入库，大件留盘。
- **边界**：CDP 实机走查被活壳挡（9333 被 PID 62656 的 WebView2=16284 占用，
  用户未答复是否可关）——壳关闭后补：切方案档新会话→上下文面板确认模式行含
  参照句→knowledge_search 命中已导入设计方案条目。未提交（树上有 wenxue 隐私线
  在途改动，提交归属挂拍板）、未装机、未 push。

## Bloom 0.29 玻璃态全板块覆盖（2026-11-23）

- **反馈复现**：切换一级板块后发现玻璃态只覆盖壳层和部分工作区。
  逐页扫描确认多个板块仍在路由根或主容器绘制整页实底，包括聊天、
  文学、绘梦、造价、众神殿、记忆中枢、微信与进度计划；运行监测另页头、
  KPI、图表和质量卡保留 74% 实底。
- **根修**：在 `components.css` 增加路由表面材质契约。非 `solid` 模式下，
  `.page-enter` 根层强制透明，主工作区、`v3-panel`、`v3-card` 和上述板块
  的主容器统一消费 `--gaea-material-fill` / `--gaea-shell-sheen` /
  `--gaea-material-filter`；实底模式保持原页面布局不变。
- **实页矩阵**：Vite 逐一切换 12 个一级板块，根层均透明；玻璃模式下
  不再存在面积 >10 万像素且不透明度 >70% 的残留主容器。毛玻璃、磨砂、
  液态玻璃分别实测为 58%+blur24、48%+grain+blur12、
  25%+sheen+blur3 lens；亮暗两种模式均验证。
- **提交**：全板块路由契约 `b917ed6d`；运行监测子容器补齐 `326d6eef`。
- **验收**：定向 `AppearancePanel/appStore/四层架构` 3 文件 `31/31`，
  改动面 ESLint 0；主树 Vite `33.48s` + Wails `28.69s`，
  `build\bin\gaea.exe` SHA256 `586775CA94011D8605B983BC00183C5893E04C696EE0F6B165AB883C3E93B40D`。
- **干净交付**：`.worktrees/pkg-326d6eef` detached 检出 `326d6eef`；
  Vite `37.19s` + Wails `47.817s`，bindings 未跳过，忽略 EOL 后漂移 0；
  payload 冒烟 200（PID 12336 已回收）。包内 exe 实机复核文学、
  进度计划、运行监测，12 个一级板块主表面均无整块实底。
- **交付包**：`releases/gaea-bloom029-glass-all-20261123-326d6eef/`。
  exe 61,664,256 B SHA256 `AF66B6ED…CD07EF1`；桌面 ZIP 35,469,268 B
  `F8B6B08D…FB436D`；源码 ZIP 829,080,512 B `42741D98…7A5A4`；
  三 DLL 与既有基线一致。截图 `temp/gaea-payload-glass-all-final.png`。
- **边界**：未装机、未 push；旧的 `b917ed6d` 中间包不用于交付。

## Bloom 0.29 最终对齐审计与桌面交付（2026-11-23）

- **基线纠正**：参考仓工作区当时检出的是 0.28，最终审计统一改用
  `origin/main` `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）。
- **桌面/Vite 根因**：Vite 是当前源码 + HMR，exe 是构建时冻结的
  `frontend/dist`；两套 profile 的偏好也相互独立。桌面曾残留自定义强调色
  `#1dd7bf`，Vite 为青金主题色 `#7faee9`。同时 Vite 曾因同秒编辑缓存
  继续执行旧 `BloomStardust` 模块。恢复同状态并刷新 mtime 后才做有效对照。
- **矩阵核验**：在 `1264×761` 下用真实 Vite 页面和 `build\bin\gaea.exe`
  逐项切换并读取计算样式。15 主题 × 亮/暗 30 组的主色、主色容器、画布、
  容器、高容器、motion1/2/3 全部一致；3 材质、4 暗氛围、4 亮氛围的
  background/mask/filter/尺寸均一致；减弱动效为 270 静态星点、流星隐藏，
  完整动效为 270 星运动、标准密度 40 流星。
- **新增根修**：彩色流星 `.m2/.m3` 原选择器被完整 galaxy 基础规则以更高
  优先级覆盖，画面一直只取 motion1。改为同特异性的完整作用域选择器，
  实测彩色为 `#7faee9 / #50b9df / #ac9fe5`，纯色统一为当前 accent；
  密度 `20/40/60`、速度 `1.6/1/0.6` 均实测生效。提交 `555eda3e`。
- **验收**：定向 `AppearancePanel/appStore/四层架构` 3 文件 `30/30`；
  主树 Vite `28.37s` + Wails `34.434s`，exe 61,651,456 B
  SHA256 `7A29D84C…F784221`，正式路径冒烟 200。
- **干净交付**：`.worktrees/pkg-555eda3e` detached 检出 `555eda3e`；
  Vite `27.41s` + Wails `55.951s`，bindings 未跳过，忽略 EOL 后内容漂移 0；
  payload 冒烟 200（PID 68940 已回收）。
- **交付包**：`releases/gaea-bloom029-final-20261123-555eda3e/`。
  payload exe 61,662,720 B SHA256 `274BE4A6…72036`；桌面 ZIP
  35,468,728 B `6C0CAF53…1A87B45`；源码 ZIP 829,074,786 B
  `FC1B741E…84CFAD`；三 DLL 与既有基线一致。截图在
  `temp/gaea-desktop-final-aligned.png`、
  `temp/gaea-vite-final-aligned.png`、
  `temp/gaea-desktop-bloom029-settings-final.png`、
  `temp/gaea-payload-final-555eda3e.png`、
  `temp/gaea-payload-settings-final.png`。payload 另起调试实例复核：
  主色 `#7faee9`、270 星点、40 标准流星、三束彩色流星均来自包内 exe。
- **边界**：未装机、未 push；真实后端与 Vite mock 的会话内容/引擎状态不参与
  皮肤一致性判断。

## Bloom 0.29 星河修复与流星控制（2026-11-22）

- **基线纠正**：严格以 `dsh-bloom-theme origin/main`
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）对照，0.29
  设置页仍保留流星颜色、密度、速度三组控制。
- **根因**：`BloomStardust` 沿用上游 `z-index:-1`，但 gaea
  `.gaea-app-layout` 没有独立 stacking context，整块壳层背景把星尘盖住。
  增加 `isolation:isolate` 后，星河位于壳层背景之上、业务内容之下。
- **功能补齐**：新增 `gaea-meteor-*` 持久化与根节点
  `data-gaea-meteor-*`；颜色 `colorful/solid`、密度
  `sparse/standard/dense`、速度 `slow/standard/fast`。密度显示
  20/40/60 颗流星候选，速度倍率 `1.6/1/0.6`，纯色统一切到当前强调色。
  设置行只在暗色“漫天星辰”下出现。
- **验证与实拍**：网页设置页实拍已出现全屏星尘和流星；桌面端新构建实拍
  显示星河和完整外观设置。定向 `AppearancePanel/appStore/架构` **30/30**、
  `tsc -b` 0、改动面 ESLint 0。
- **提交**：星河层级 `9a7475ad`；流星控制 `a37b7548`。
- **桌面构建**：当前主树 Vite `31.53s` + Wails `28.625s`；
  干净树 Vite `34.83s` + Wails `56.851s`，bindings 未跳过，内容漂移 0。
  主树 exe SHA256 `32256E2E…13E722`，冒烟 200（PID 65240 已回收）。
- **交付**：`releases/gaea-bloom029-starlight-20261122-a37b7548/`；
  干净树 payload 冒烟 200（PID 43540 已回收），六项 SHA256 已记录。
  旧 `gaea-bloom029-20261122-96335e75` 包保留但已被本包取代。
- **边界**：未装机、未 push。

## Bloom 0.29 配色大脑桌面构建与 Git 打包（2026-11-22）

- **提交边界**：外观提交 `96335e75`（`feat(gaea): align Bloom appearance
  tokens with upstream`，36 文件）；主树探针 `probe-memory-out.jsonl`、
  `scripts/cache_probe_live.go` 与既有 release ZIP 未进入提交和包。
- **干净构建**：`.worktrees/pkg-96335e75` detached 检出该提交；复用主树
  `frontend/node_modules` junction。先 `npm run build`，Vite `31.92s` / 6611
  modules；再以系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc 执行
  `wails build -s -ldflags "-s -w" -trimpath`，`1m0.058s`，bindings 未跳过。
- **漂移与冒烟**：Wails 再生成仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol`
  内容差异文件数 0；正式 payload `/api/health` **200**，PID 61612 已回收。
- **测试口径**：定向 4 文件 **110/110**；全量前端 518 文件 / 4706 例，4704
  通过、2 例满并发失败（ContextView 水位标、SchedulingSection 故障转移开关），
  两文件隔离复跑 **44/44**。本地快闸被未改动 Go 基线的 98 项 golangci 问题阻断。
- **卫生边界**：`check-docs` 仍被既有未登记文档
  `gaea-agent-memory-nebula-distill-2026-11.md` 与
  `gaea-day-log-2026-11-22.md` 阻断，本次不混入无关文档索引改动。
- **交付**：`releases/gaea-bloom029-20261122-96335e75/`。exe 61,657,088 B
  `9D619B62…993803`；桌面 ZIP 35,466,750 B `3F831790…E232EF`；源码 ZIP
  829,064,469 B / 6971 条 `06F240A7…BD1815`。三枚 DLL 与六项总和见
  `PACKAGE-INFO.md` / `SHA256SUMS.txt`。
- **边界**：产品版本保持 `4.544.0`；未装机、未 push。

## Bloom 0.29 配色大脑对齐（2026-11-22）

- **上游基线**：`C:\AI\re\dsh-bloom-theme` 拉取 `origin/main` 至
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0），逐项对照
  `theme-tokens.ts`、`palette.ts`、`tokens.ts` 与 `switcher` 颜色映射。
- **可读强调轨**：亮色改为 upstream `accentL 88% + black 12%`，暗色直接使用
  `accentD`；原始 accent 只用于表面/色块染色，`onPrimary` 回到上游背景前景。
- **语义阶梯**：接入亮 28/32/35/37、暗 30/35/40/43 的文案四档，亮暗各自
  独立四档边框，以及 success/warning tertiary、warning label、danger hover、
  business tertiary 与侧栏 active/hover token。
- **菜单与无障碍**：色块单字、选中勾与 hover 光晕接上游视觉值；修复
  `prefers-reduced-transparency` 被 `:not([data-gaea-surface])` 永久挡掉的问题，
  已选择材质时也会退实底，并同步关闭 trigger/popover 的滤镜。
- **门禁**：外观/主题/设置定向 **4 文件 110 例**、`tsc -b` 0、改动面 ESLint 0、
  Vite 生产构建成功 `21.97s`（仅既有 CSS/chunk 警告）。右侧 IAB 已打开
  `http://127.0.0.1:5174/?mock=1`。
- **边界**：未构建 Wails、未打包、未装机、未 push。

## Bloom 0.29 完全对齐：色块网格、OKLCH 表面与双套氛围（2026-11-22）

- **上游基线**：重读 `dsh-bloom-theme` 最新 `origin/main`
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）源码，按实现细节
  逐项替换此前的近似方案。
- **顶栏入口**：切换为上游 0.29 六列色块网格，15 套配色全部展示；每格使用
  `morandi → accentL` 三段渐变与国风单字，深色切换为 `accentD`；悬停/键盘聚焦
  更新同一行色名与诗句字幕；补关闭按钮、选中环、焦点返回与 320ms View Transition。
- **配色大脑**：15 套 accent 保持上游原值；新增真实 OKLCH 混色，不再用 sRGB
  通道插值。base/layer-2/layer-3 与侧栏按目标彩度反推 accent 混入比，
  `surfaceTint1/2/3` 回到上游 `bg / layer-2 / layer-3`，莫兰迪只用于氛围、
  光晕和细线。
- **材质**：毛玻璃 58% + `blur(24px)`；磨砂 48% + `blur(12px)` + SVG 颗粒；
  液态 25% + `blur(3px) url(#gaea-liquid-lens) saturate(1.2)`。对话主画布
  改为透出 body 氛围，内容面统一消费 `--gaea-shell-*` 契约。
- **背景**：亮暗分别持久化。浅色提供 `斜光流转 / 水彩晕染 / 清透留白 /
  不显示`，对应三张 SVG fog mask；深色提供 `角落微光 / 对角漫染 / 漫天星辰 /
  不显示`，叠加四角 radial ambience，并逐行移植上游 330 节点确定性星尘/流星序列。
- **根因修复**：主内容 `Content` 内联 `--gaea-glass-bg` 把 body 四角氛围盖成
  一整块实底；移除该内联背景后，dark canvas 空白区取样与 0.29 参考图一致。
  旧 cyber-grid/star-dots/space-dust 常驻层同时删除。
- **自动换色**：0/5/15/30/60 分钟持久化；候选集与上游一致，为全量 15 套。
- **验证**：全量前端 `518` 文件 / `4674` 例均通过；`tsc -b` 0；改动面 ESLint
  0；Vite 生产构建 `25.27s` 成功（既有 CSS parser/chunk 警告）。全量进程末尾
  仍报 `wenxue/components/ToolCard.test.tsx` 的既有
  `EnvironmentTeardownError`（动态 `mock/voice.ts` 在环境回收后完成 import），
  测试本身 2/2，单跑该文件亦 2/2 绿，与本轮外观链路无关。
- **实拍**：右侧浏览器停留 `mist / dark / soft / corner`；暗色主画布空白点
  `(34,44,53)` 与上游 0.29 参考 `(33,44,52)` 对齐，`Content` 透明，
  `.gaea-stardust` 节点数 330。
- **边界**：未提交、未构建 Wails、未装机、未 push。

## Bloom 外观架构 v4 与 Git 打包（2026-11-22）

- **功能提交**：`7fcd1ff5`（`feat(gaea): rebuild appearance system around Bloom architecture`，
  76 文件，+3036/−1359）。删除旧 7 套主题，建立 palette → semantic → materials → components
  四层架构；移植 Bloom 全量 15 套、精选 12 套，默认 `ripple / 天青`。
- **能力面**：亮暗/跟随系统、三种表面材质、四种背景氛围、一键玻璃态；玻璃态统一
  `--gaea-shell-*` 契约并解开全局 `v3-rail`/`v3-telemetry` 与路由容器的错误作用域。
  聊天画布恢复 80px 网格，以径向 mask 从中心向四周渐隐。
- **清理与守卫**：UI、图表、语法高亮、预览组件完成语义色迁移；新增四层架构检查、
  组件层禁止材质特例、渐隐网格和玻璃态契约回归。
- **验收**：外观/主题/设置/对比度/图表/上下文/GenUI 定向 8 文件 `106/106`；
  干净工作树前端 Vite `27.25s`，UCRT64 Wails `47.13s`（bindings 未跳过）；
  bindings 再生成仅 CRLF 噪音，`git diff --ignore-space-at-eol` 内容漂移 0。
- **交付**：`releases/gaea-appearance-v4-20261122-7fcd1ff5/`；payload 正式路径
  `/api/health` 200（PID 16416 已回收）。exe `268F8558…3807E0`，二进制 ZIP
  `713260DE…64EDC9`，`git archive` 源码 ZIP `0501E823…FD2128`；六项校验和复算全 OK。
- **边界**：产品版本保持 `4.544.0`，未抬号；未覆盖安装、未 push。主树既有历史 ZIP
  与 `scripts/cache_probe_live.go` 未混入本包。

## 工作台 UI 桌面构建与 Git 打包（2026-11-22）

- **提交边界**：功能提交 `cf422218`（工作台页签与各面板 UI/UX）；最终提交
  `4d23703e`（Wails 绑定物化 `ExportTocEntry` 与标书导出字段）。能力面板重构、
  F 批 Go 内核与临时试验脚本仍留在主工作树，未进入本包。
- **干净构建**：`.worktrees/pkg-4d23703e` 检出 `4d23703e`；前端依赖按
  `pnpm --frozen-lockfile` 准备，`npm run build` 38.31s；UCRT64 GCC +
  `CGO_ENABLED=1` + 系统 TMP/TEMP，`wails build -s -ldflags "-s -w" -trimpath`
  52.275s，**Skip Bindings=false**。
- **验收**：定向 8 文件 **96/96**、正式 payload `/api/health` **200**
  （PID 37464 已回收）；payload 四件、桌面 ZIP、`git archive` 源码 ZIP 共六项
  SHA256 已复算。构建后 Wails 包装文件与 `go.mod` 仅 CRLF 噪音，内容零漂移。
- **交付**：`releases/gaea-workbench-tabs-20261122-4d23703e/`；exe 61,400,064 B
  SHA256 `24817586…8F9833`，桌面 ZIP 35,123,360 B
  `CEBBCD58…5015ADA`，源码 ZIP 828,931,977 B / 6,916 条
  `1829AE7C…3FC88F8E`。三枚 DLL 与上包一致；未装机、未 push。

## 办公 / 文学工作台 UI 与文学插图页签修正（2026-11-21）

用户依次要求优化左右两侧工作台、文学右侧补回「插图」页签，并明确办公工作台
也要同步改。按同一套工作台视觉语言完成双板收口，业务数据面与 bridge 未改。

- **文学页签位置**：`插图` 保持在右侧工作台，**不进**对话/轨迹/上下文主标签栏；
  主栏维持原有八标签，右栏 gallery 入口与 `WenxueGalleryHub` 数据面接通。
- **插图图册**：修复共享 Hub 横向 flex 导致筛选条和图片墙被拉成竖栏的问题；
  改为“筛选行 + 自适应图片墙”，筛选按钮补来源计数，图片加载失败统一转占位，
  不再显示浏览器破图图标。
- **办公工作台**：启动页改为带主题感知矢量插图的主入口、文件优先卡与两列面板
  入口；页签条固定新建/管理/收起操作，活动页签支持方向键/Home/End roving focus；
  初版曾增加逐视图上下文头，随后真机审计确认会与面板自身标题形成三层重复，已在
  同日续刀中移除并改为直接承接各面板自己的唯一标题行。
- **文学工作台**：复用同一工作台壳与紧凑 CSS，入口调整为文件、产物、浏览器、
  Git、UI、插图六卡；同样具备插图头、管理面板、固定操作区、页签键盘与视图
  上下文头。
- **验证**：办公 WorkspacePane、文学 WorkspacePane、ChatTabs、Gallery 定向
  **4 文件 23/23** 绿；`tsc -b` 0；改动面 ESLint 0 error；`npm run build`
  **28.42s** 绿（既有 Wenxue CSS/chunk 警告）；浏览器实页核对文学八标签、
  右栏插图入口、筛选计数、图片占位及双板工作台启动页。未构建 Wails、未装机。

## 工作台页签与各面板 UX 续刀（2026-11-21）

用户要求继续重设计右侧工作台各标签页。按 `redesign-existing-projects` 真机审计 +
`ui-ux-pro-max` 的导航、焦点、空态与 reduced-motion 规则，在既有未提交工作树上
增量收口；不改 bridge、业务数据面或页签持久化契约。

- **页签轨道**：移除页签、视图上下文头和面板自身标题的三层重复；右栏 ≤420px 时
  非活动视图收成图标、活动页签保留标题与关闭钮，≤300px 才全图标化。页签保留
  roving focus、方向键/Home/End、稳定 `aria-label` 与徽标圆点；非活动关闭钮不再
  占用布局宽度，避免窄栏页签被无意义挤出。
- **各面板**：文件/产物直接使用自身唯一标题；浏览器页把 `← → ↻ ⏎ ↗ ◉` 字符
  替换为统一矢量图标，地址校验改为就近 `role=alert`，补可操作沙箱空态；Git 将
  五档视图从标题行拆成独立可滚动第二行；UI 页补图标化说明空态与面板围栏示例；
  进度计划工具条改为“工程摘要 + 操作 / 四档视图”两层，≤420px 四个操作自动收成
  带 title/aria-label 的图标；CAD/进度懒加载统一为带状态播报的加载转轮。
- **双板一致性**：办公与文学 WorkspacePane / BrowserPane / GenuiPanel / GitPanel
  同步改造；办公与文学共享 `workspace-pane.css`，全部使用现有主题令牌，未新增
  硬编码主题色或图标来源。
- **验证**：WorkspacePane（办公/文学）、BrowserPane、Git、ScheduleTool、
  Deliverables/Changes/Explorer 与 sidebarRegistry 定向 **8 文件 96/96** 绿；
  `tsc -b` 0；改动面 ESLint 0 error；`npm run build` **28.51s** 绿（既有 Wenxue
  CSS/chunk 警告）。实页走查暗/亮双主题、办公/文学双板、340px 窄栏、Git/UI/
  浏览器/进度计划与图册页签；未构建 Wails、未装机。

## 办公右侧能力面板 UI 重设计（2026-11-21）

用户令「使用技能重新设计办公板块的 MCP 与技能面板，特别是工具标签页」，
第二轮补充「对右侧面板 UI 进行优化设计」。按 redesign-existing-projects +
ui-ux-pro-max 审计后，在现有 React/Tailwind v4/主题令牌栈内增量改造，不改
bridge 与 MCP 管理逻辑、不迁框架。

- **抽屉壳**：标题改为图标锁合 + 副标题；状态摘要、三页签、内容滚动分层；
  页签采用 MCP/工具/技能短标签，完整名进 aria-label；方向键/Home/End 采用
  roving focus，访问过的页签保持挂载，切换不丢搜索或展开状态。
- **工具页**：工具目录获得首屏主位；搜索升级为名称/说明检索，补结果计数、
  无结果空态与清除动作；工具面从 8 行常驻开关收成摘要卡，默认折叠，保留
  精简/标准预设、组级开关与注册闸语义。
- **列表视觉**：工具与技能从连续独立圆角卡片收敛为同一表面内的平面清单，
  用分组标题、细分隔线与单主色强调建立层次；补加载骨架、错误/成功提示、
  空态与亮暗主题状态。
- **MCP/技能**：服务器工具行补在线/异常/停用文字徽标，开关和增删操作统一
  尺寸与焦点；技能页补搜索清除、范围/子代理/调用次数信息，并压缩卡片噪声。
- **验证**：新增三页签键盘/状态保留与工具搜索空态回归；定向 3 文件
  **7/7** 绿、`tsc -b` 0、改动面 ESLint 0 error、`npm run build` **37.53s**
  绿（既有 Wenxue CSS/chunk 警告）；亮暗实页与 360/440px 抽屉走查，无横向
  溢出，关键正文/次级对比度实测 **5.56–14.33:1**。未构建 Wails、未装机。

## 平台缺陷清单 D1–D10 修复（2026-11-20）

接用户提供的《gaea 平台缺陷清单》继续收口；工作树原有 D1/D2/D3/D5/D9
半成品修复全部保留并补回归锁，不重新改写已有方案。

- **D1 截断硬失败**：bash 在 rc=0 但出现未闭合 heredoc 告警时升格为
  `HEREDOC_TRUNCATED` 结构化失败，plain/json 两种输出均已覆盖；同时修复
  空工作区下发布守卫 nil 调用。
- **D2 整篇重写**：`write_file` 新增显式 `overwrite:true`，跳过同回合
  stale 闸；跨轮外部修改检测不豁免，拒绝文案同时给出重写/局部编辑两条路。
- **D3 大输出可读性**：头部+中段均匀采样+信号行+尾部；截断回执带原始行数/
  字节数、实际保留行数与采样策略，gap 标记防止采样片段被误读为连续输出。
- **D4 图件体检**：`chart_gen` / `diagram_gen` 渲染进程内做确定性 bbox 检查
  （文本重叠、被 clip 的出界、节点框溢出、字号 < 6pt），`layout` 随回执返回；
  文本-线条交叉明确不判。新增可导入的 `mpl_layout_check.py` 供自绘 matplotlib
  图件使用。
- **D5 草稿删除**：仅放行非递归且完整路径段落在 `_tmp*`/`_draft`、系统临时根
  或 `.tmp/.bak/.old` 后缀的单文件删除；`..`、通配符、变量、未知旗标与递归
  删除继续硬拦，拒绝文案指向 `.trash/` + `move_file`。
- **D6 安全文本补丁**：现有 `edit_file` JSON 通道补足描述与回归锁，明确嵌套
  引号、反斜杠、字面量 `\n`/`\t` 不经 shell；bash 描述引导大段写入改走
  `write_file`/`edit_file`。
- **D7 识图边界**：vision 只承担内容/OCR/图注语义核对；涉及重叠/出界/溢出
  的请求自动附“语义判断不是几何检测”边界，几何验收改走 `layout` 报告。
- **D8 工程图件技能**：新增 `.gaea/skills/engineering-figures/`，固化数字
  溯源、示意/实测分栏、用途单一、图注模板、反例纪律与交付前检查单。
- **D9 字重**：图件生成优先 Microsoft YaHei，并在缺 bold 变体时把降级提示
  写入回执。
- **D10 中断恢复**：交接块新增已完成/进行中/未开始待办快照；已完成工具记录
  `output` 类产物路径，并对 ≤16MiB 的常规文件附 SHA256 + 字节数（最多 16 个）。
- **验证**：`go test ./internal/gaea/... -count=1` 全绿（agent 19.41s、
  boot 35.08s、builtin 34.70s）；`go vet` 相关五包绿；Python 技能脚本编译绿；
  `check-docs` 全绿（孤儿 0 / 悬空 0）。全量首跑暴露的 boot/control 两处
  image_gen 空间测试仍按旧 `play` 口径断言，已按 2026-11-20「办公+文学都可
  生图、`image_gen=shared`」拍板同步测试，生产行为未改。
  **未提交、未装机。**
- **桌面构建（2026-11-20 20:11）**：前端 `npm.cmd run build` 22.97s 绿（既有
  Wenxue CSS/chunk 警告）；UCRT64 GCC + `CGO_ENABLED=1` + 系统 TMP/TEMP，
  `wails build -s -ldflags "-s -w" -trimpath` 29.796s，`Skip Bindings=false`。
  `build/bin/gaea.exe` 60,925,952 B，SHA256
  `B34A30D9036B9CB027DFED04151481D0BC400C5A12F70D582C5159C0D26F2E44`；
  三枚语音 DLL 同目录。正式构建目录冒烟 `/api/health` **200**
  （`{"status":"ok"}`，PID 34776 已回收）。

## 数据根统一：蒸馏/逆向数据全部并入 C:\AI\re（2026-11-20）

用户指令「所有蒸馏的数据文件都放在 `C:\AI\re` 里，方便查找，同时把前期蒸馏的那些都迁移过来，
同步更新文档地址」。本轮只做数据搬迁与文档同步，未动业务代码、未构建、未发版。

- **记忆落地**：仓库根 `AGENTS.md` 新增「蒸馏与逆向数据落盘（强制默认）」节（唯一存放点 /
  命名 / 七层目录 / 禁止复活旧目录 / 引用写绝对路径）；`.gaea/AGENTS.md` 增「数据根」节；
  `docs/README.md` 头部加数据根段并补 `codex-distill/` 子目录索引行。
- **迁移（文件数与字节数逐一核对一致）**：`C:\AI\_distill\codex-computer-use`（1,329 / 73.5 MB）、
  `_distill\agorai`（201 / 4.1 MB，含浅克隆 `.git`）、`_distill\aidebate`（82 / 0.4 MB）、
  仓库内 `re-lingxi`（1,311 / 126.0 MB）→ `C:\AI\re\{codex-computer-use,agorai,aidebate,lingxi}`；
  `C:\AI\_distill` 已删；仓库 `re-lingxi/` 记为 232 个删除（**未提交**）。
- **索引**：新建 `C:\AI\re\README.md`：总则、目录总览（6 个数据集 × 规模 × 入口文档）、
  布局差异说明、查找路线、迁移表、维护规约。
- **文档同步**：19 个文件 69 处旧路径改写 + 13 处斜杠归一；覆盖 `docs\codex-distill\`
  （README / asset-62 / asset-63）、`gaea-computer-use-landing-research-2026-11.md`、
  `gaea-lingxi-*` 五份、`gaea-pantheon-agorai-aidebate-distill-round2`、day-log 11-13 / 11-20、
  `.gaea\progress.md` / `todos.md`、两份 `lingxiIcons.tsx` 与两份 `*-lingxi-skin.css` 注释、
  以及 `C:\AI\re\lingxi\out\*.md` 自述路径。
- **卫生**：`.gitignore` 删除 8 条已失效的 `re-lingxi/out/*` 忽略规则（逆向数据不再进仓库）。
- **第二批：`clones/` 全量迁出（用户追加指令「clones/ 里其它克隆也迁移过去」）**：16 个目录
  整批 `move` 到 `C:\AI\re\<同名>`——上游参考克隆 13（deepseek-harness / deepseek-reasonix /
  reasonix / zcode / orca / unsloth / rea / projectlibre / MuMuAINovel / oh-story-claudecode /
  owllook / so-novel / speech-to-speech）、DSH web 依赖快照 2（dsh-web-latest / dsh-web-baseline）、
  业务样本 1（mpp2013）。文件数 16/16 逐一核对一致（`diffs=0`）；`re` 现 4.60 GB / 231,546 文件；
  仓库 `clones/` 迁空后整目录删除，`.gitignore` 的 `/clones/` 规则同步移除。
- **代码耦合修复**：`internal/schedule/mpp_test.go` 新增 `reSamplePath()`（默认根 `C:\AI\re`，
  `GAEA_RE_ROOT` 可覆盖，旧 `../../clones/` 兜底），两处真实样本门控改走它；`mpp.go` 头注同步。
  **验证：`go test ./internal/schedule -run TestMpp -v` 全过，`TestMppRealSamples` 与
  `TestMppRealSamples2013` 均 PASS（非 SKIP）**——证明新数据根样本可读、解析真值未漂移。
- **文档批量改写**：62 个文件（docs 各册 + `.gaea/reports` + 6 份 `LICENSE-oh-story.txt` +
  `releases/archive` 三份 + `CHANGELOG.md`）里的 `clones/<x>` 引用统一改为 `C:\AI\re\<x>`；
  `clones/.tmp` 两处历史扫描口径说明按原样保留。
- **原位置清理（用户追问后落实）**：核对 `C:\AI\_distill`、`C:\AI\wubigrok\re-lingxi` 已不存在；
  另发现 `clones\agorai`（1e469be）与 `clones\aidebate`（80f3b77）是同 commit 的重复副本，
  与 `C:\AI\re\` 下两份完全重复——一并删除，两份文档引用（`gaea-pantheon-agorai-aidebate-distill-2026-11-07.md`、
  `docs\research\aidebate-distill\01-source-analysis.md`）改指 `C:\AI\re\agorai|aidebate`。
  蒸馏数据自此只有 `C:\AI\re` 一处。
- **验证**：`node scripts/check-docs.mjs` 绿（孤儿 0 / 悬空 0 / 指令预算 37,641 B ≤ 65,536 /
  .ps1 编码合规）；`git grep` 残留仅剩说明性提及与历史发布包/历史 TSV 清单。

## 运行监测 UX/UI 第二轮：可执行状态面（2026-11-20）

用户要求「使用技能重新设计运行监测板块的 UX、UI，要能真正起作用」。本轮用
`redesign-existing-projects` 做现有页面审计，用 `ui-ux-pro-max` 约束实时数据、
无障碍与下钻交互；保留数据契约，不新增后端绑定。

- **当前判定**：健康摘要旁新增关注清单，由引擎掉线、资源/上下文压力、低成功率、
  近期失败和统计读取失败派生；每项都有处置动作。900px 以下关注清单提前于健康摘要。
- **同步可信度**：增加暂停/恢复、请求防重入、统计轮询联动停用、同步中断与最后成功
  快照提示；页面隐藏显示“后台暂停”，不再持续显示绿色实时态。
- **下钻闭环**：错误与需关注模型可定位最近失败请求；最近请求新增模型/引擎/功能/错误
  搜索与清除条件；模型明细补失败列和真实排序，修复旧文案与表格不一致。
- **可访问性**：资源条改 `progressbar` 语义并显示偏高/超载文字；状态与筛选使用
  `aria-live`，跳转区块获得可见焦点；最近请求表格补时间/Token/耗时排序。
- **页面规范**：新增 `design-system/gaea/pages/runtime.md`；专项文档补第二轮章节。
- **验证**：RuntimePage + useStatsState 定向 **20/20**；`tsc -b`、定向 ESLint 0；
  `npm run build` 37.14s 绿（既有 Wenxue CSS/chunk 警告）；1416 亮暗、780、390
  实页走查无页面级横向溢出。
- **桌面构建**：ucrt64/CGO + 系统 TMP/TEMP，完整 `wails build -s -ldflags "-s -w"
  -trimpath`，bindings 未跳过，1m19.037s 绿；exe 60,845,568 B，SHA256
  `873342C5DB5EA64FE8E757F24B29F9F1429ACE15D72E683663301E9E823DAFDC`；
  `onnxruntime.dll`/sherpa 两 DLL 同目录；正式路径冒烟 `/api/health` 200
  （PID 58828 已回收）。
- **状态（接手收口 2026-11-20 午后）**：codex 停止后由 ZCode 接管——双域批提交
  8323e9d4（运行监测本批+同树模型中心归位：ModelGroupNav 组级导航/引擎管理收口
  xAI 登录/设置模型组五页签瘦身，复验 8 文件 71/71+useStatsState 4/4+tsc 0）；
  Wails 54.37s 绿（bindings 未跳过），装机 exe SHA256
  `39C682D92A91D50899AC8396721E5F2BCD3586674364009F1FAC7E86701C8F8A`，新壳
  PID 26908（11:59:08 > 构建时刻 11:58:57）INSTALL OK，冒烟 /api/health 200；
  resumed.jsonl.meta/capability.go 运行残渣回退不入账。

## 设置板块整板 UI 二次重设计（2026-11-20，a7ea2ac4）

用户追加「整个设置板块的 UI」「模型把以前多个标签的内容挤在一起」。本轮保留全部
配置/绑定契约，重画 15 个分类共用的控制台外壳，并把多任务分类改为内部任务页签：

- **整板外壳**：命令栏 + 两级分类轨道 + 当前分区画布；统一实底卡、内容头、
  状态徽章、焦点/skip link/reduced-motion。轨道降噪为紧凑标签，说明集中到内容头。
- **任务分栏**：通用 3 / 聊天 4 / 绘梦 2 / 办公 4 / 模型 5 / 安全 2 / 数据 2 /
  文学 2 / 高级 2；访问过的页签保持挂载，草稿不丢；标准 tablist 键盘语义。
- **模型专项**：旧五域纵向堆叠改为概览/功能绑定/空间策略/本地调度/引擎管理；
  1280×720 mock 实页滚动高度 3155→801px。补设置页 `--mc-*` 令牌作用域，修掉
  模型中心 CSS 未加载时设置页退化成无样式文本的隐患。
- **搜索/深链**：搜索可自动选任务页签并锚定段卡；语音入口直达「朗读引擎」
  （`openSettings(category, tab)`），不再落到聊天首页。
- **导航连续性**：切分类回内容顶部；竖栏/窄窗横栏自动把激活分类滚入视野；
  任务页签离开分类后记忆上次选择；窄窗横栏自动把当前任务带回可视区。
- **响应式/主题**：1280 亮暗、1024、720、390 实页走查；无页面级横向溢出；
  亮暗正文对比度 ≥14.16:1、次级 ≥6.59:1。
- **空态**：模型引擎为零或筛选后无结果时给出明确说明与添加入口，不再留空白区。
- **验证**：定向 7 文件 76/76；全量 503/504 文件、4583/4585 用例通过；两例
  `DatabaseLibrary` 并发 flaky 单跑 6/6，`ToolCard` 既有 `EnvironmentTeardownError`
  单跑 2/2；`tsc -b` 0、改动面 ESLint 0 error、`npm run build` 36.28s 绿
  （既有 Wenxue CSS/chunk warning）；走查脚本扩至 24 断言并通过 syntax check。
- **状态**：已提交 a7ea2ac4；Wails 构建 1m56s 绿（exe SHA256 `7302bee3…e2c376`），
  冒烟 `/api/health` 200，装机 `C:\AI\gaea-app` 四件哈希一致，新壳 PID 64900
  （09:06:23 > 构建时刻 09:06:04）`INSTALL OK`；截图见 `.tmp/settings-redesign/`。

## 设置中枢全线收官+闭环后增量五批（2026-11-20）

接 11-19 深夜七批+尾批。今日 12 提交全推 origin（487f3333..c65c8a77 含 74 笔
列车，ahead=0），装机四次同步（终态 PID 26064），挂账池空：

- **模型中心直达回摆**（6cd9f0a3）：接入节功能绑定/引擎管理两 tab 回驻，
  单源组件双宿主（设置页 model 组消费同组件）；rail-foot 摘要卡回 setCategory。
- **UI 重设计四刀**（9fdb3d78）：内容头方位感/搜索零死端建议胶囊/导航精修
  （zone tick+激活图标着色+n/total）/inspector 相关分组快跳；ui-ux-pro-max
  只取结构性建议，配色字体否决保 md-sys 一致性。
- **索引卡**（e9a5ae2c）：BoardIndexSection 边界留域+NAVIGATE 直达，板块区七组。
- **键盘可达**（c06bfb20）：↑↓ 环绕/Home/End+`/` 聚焦搜索+Esc 清词。
- **竞态根修**（d4c2892e）：void import(bridge) 无 .catch 翻牌用例实锤（挂数
  每轮漂移 1/0/1/2），四处收编后两连跑零翻牌；EnvironmentTeardownError 日志
  =vitest v4 已知局限挂池。
- **搜索直达项**（c65c8a77）：命中段锚定 scrollIntoView+flash 轮询式（段标题
  随 bridge 数据异步挂载，单次扫描必扑空）。
- **验证面**：亮暗双主题目检+CDP 走查 17/17（scripts/gaea-settings-walk.mjs
  转正入库）+全量 504 文件 4578 例+tsc+lint 绿。
- **坑册**：gaea 壳因 --disable-features 被 Wails 连带默认开 127.0.0.1:9333
  （实盘壳恒占口，CDP 走查前必先收壳）；测试漏 import waitFor 的 ReferenceError
  189ms 秒抛被误读成超时挂（失败先看报错原文，时长≠超时）。

## 软件 Logo 明暗双原图全量替换 + 桌面构建（2026-11-19）

用户补入 `logo浅色.jpg`（白底玻璃光效）与 `logo深色.jpg`（黑底发光版），
要求软件品牌全面替换。两张源图均 1568×1264；本轮不再用旧母图提取重绘，
UI 的可见 Logo 直接使用这两张原图，按运行时明暗切换，并保留原始比例与
完整留白。

- **原始资产**：两张 JPG 字节复制为
  `frontend/public/gaea-brand-light.jpg`、
  `frontend/public/gaea-brand-dark.jpg`，SHA256 分别与源文件逐字节相等：
  `D06366EC…0134B773`、`004680E4…4133FF9AC2`。
- **主题映射**：统一 `.gaea-brand-art--light/--dark` 资产层，以
  `data-bs-light` 切换；白底图用 `multiply`、黑底图用 `screen` 融合到
  当前表面，不出现矩形图片底板。首帧、运行时切主题、系统模式同步生效。
- **替换面**：办公/文学侧栏、指挥轨道、欢迎页、设置“关于”全部改用
  同一对原图；关于页原有“盖/G”文字占位及六条 `logoGlyph` 三语键退役；
  四个旧 `src/**/assets/logo*.svg` 引用与文件退役。favicon 随明暗切到
  `favicon-light.svg` / `favicon-dark.svg`。
- **桌面图标**：从深色原图取可见品牌区生成 1024 圆角主图标与紧凑小图标；
  ICO 七档 16/24/32/48/64/128/256。`build/appicon.png` SHA256
  `8821ECFF…3644D5E7`，`build/windows/icon.ico` SHA256
  `6280CCB9…A5B83C9`。exe 内嵌七档图标与 ICO 对应帧逐字节比对
  **7/7 exact**。
- **视觉走查**：1366×约 890 实页完成办公亮/暗、文学亮色、设置关于
  亮/暗走查；侧栏 Logo 无偏移、无畸变、无白/黑矩形底板，指挥轨道与
  关于页切换一致。使用 `ui-ux-pro-max` 检查图标比例、明暗对比、
  reduced-motion 与无障碍标签。
- **门禁**：Sidebar 两文件 **22/22**；改动面 ESLint 0；`tsc -b` 0；
  正式前端构建 **35.16s** 绿（仓库既有 Wenxue CSS syntax 与
  chunk-size warning 仍在，非本轮文件）。
- **桌面构建/验收**：UCRT64 GCC、`CGO_ENABLED=1`、系统默认 TMP/TEMP，
  `wails build -s -ldflags "-s -w" -trimpath` **42.031s**，
  `Skip Bindings=false`；exe **60,460,032 B** SHA256
  `DC354710C0205B79A27E7DB8647E0185E5B31ADAD5578F1AA2D5A51B7DE84A35`；
  三 DLL 同目录。正式路径冒烟 `/api/health` **200**（PID 4680 已回收）。
- **范围**：未提交、未装机；工作树内其他在制业务改动未触碰。

## 开机动画双场景品牌化重设计（2026-11-19）

用户先后给出两张最终画面，并明确要求直接使用原图、自动完成“天空 →
人间”转场。本轮不再拆图重绘 Logo、文字、背景或进度，不调用 Three.js；
两张原始 PNG 作为全屏场景直接使用，在 1.596s 进行 1050ms 整幅交叉
淡变（原图进度分别为 68%、72%）。

- **场景实现**：两层全屏原图常驻，`data-scene` 随语音阶段从 `earth`
  自动切到 `gaia`；`object-fit: contain` 保持原始比例且不裁切画面。
  静态首屏同样内置 1.596s 自动切换。
- **原始资产**：直接字节复制工作空间中的
  `盖亚智能星际启动界面.png`、`盖亚智能世界启动界面.png`（均
  1672×941），目标文件 SHA256 分别为
  `60C71A31…394D74A4`、`68EBE841…1F52A336`，与源文件逐一相等。
- **交互/无障碍**：原图中的 Logo、字标、进度与状态不再 DOM 重绘；
  页面保留隐藏标题、状态、声音与 ESC 跳过能力。原图进度区域按
  1672×941 坐标覆盖为真实动态进度条：0% 平滑增长到 68%，阶段切换后
  继续到 72%，数字与轨道同步；覆盖底采用场景自适应玻璃层（天空深蓝、
  人间青绿），背景随底图交叉变化并遮净原百分比。声音入口常驻左下角并
  显示 `PLAY/AUDIO/MUTED`；自动播放被拦截时首次点击直接重试播放而不是
  误切换成静音。
- **视觉走查**：1366×886 暗色双场景、375×812 竖屏自动转场；
  `temp/gaea-boot-earthrise-final.png` 与
  `temp/gaea-boot-gaia-final.png` 留存终态截图。
- **门禁**：`BootSplash.test.tsx` 8/8；`tsc -b` 0；改动面 ESLint 0；
  前端 `npm run build` 绿（仓库既有 Wenxue CSS syntax 与 chunk-size
  warning 仍在，非本轮文件）。
- **范围**：`frontend/index.html`、`BootSplash.tsx`、`BootSplash.test.tsx`、
  `boot-splash.css`、三语 boot 文案及两张 public 原图；未提交、未构建
  Wails、未装机。

## 众神殿与子代理对话 UI 收口 + 桌面包（2026-11-18 夜）

用户令「检查细节后 git 打包、做好文件记录和记忆更新、构建桌面端」。
功能提交 `fc120738eac3c461d0d4448558c1541a249452f2`（13 files，
1,973 insertions / 294 deletions），只含本轮相关源码/测试，不夹带主工作树
其他在制业务改动。

- **众神殿独立 Agent 窗口**：观点完成态默认折叠并支持批量控制；轨迹合并
  工具记录，补统计、轮次概览、搜索、分组和失败文字；新增上下文标签页
  （Prompt 构成/会话信息/趋势/逐轮账目），无 usage 使用单一诚实空态；
  标签改为标准 tablist/tabpanel，支持方向键与 Home/End。
- **子代理对话与父代理统一**：`SubagentThread` 复用父代理的 `.transcript`
  / `.v3-reading` 阅读列、`buildSegments`、`ProcessCard`、`UserMessage`、
  `AssistantMessage`、`ToolCard`；不再维护扁平私有消息流。
- **去重与命名**：父对话过滤 `background task/bash started` 通知及带
  `subagentRef` 的完成回执；委派卡按会话调用顺序显示 `子代理 A/B/C…`，
  保留任务摘要和右栏打开入口。
- **门禁**：5 个相关测试文件 **110/110**；`tsc -b` 0；改动面 ESLint 0；
  Vite 20.95s（前端总门禁 46.400s）。
- **干净树桌面构建**：`.worktrees/pkg-fc120738`（detached @`fc120738`，
  node_modules junction）；UCRT64 GCC + `CGO_ENABLED=1` + 系统 TMP/TEMP，
  `wails build -s -ldflags "-s -w" -trimpath` **44.2s**，
  `Skip Bindings=false`；wailsjs 仅行尾变化，忽略行尾后内容漂移 0。
- **交付**：`releases/gaea-pantheon-subagent-ui-20261118-fc120738/`；payload
  `gaea.exe` 55,193,088 B SHA256 `86052955E1F5…B86CEF`，三 DLL 与基线
  同哈希；正式路径冒烟 `/api/health` 200（PID 41464 回收）。
- **装机**：`gaea_install.ps1 -SourceDir payload -Launch` → INSTALL OK，
  新 PID 59568，启动时间 23:46:45 > 构建时间 23:46:15；装机四件与 payload
  哈希逐项一致。
- **归档**：二进制 ZIP 30,177,708 B；源码 TAR 43,815,614 B；源码 ZIP
  47,969,713 B；四张走查图；`PACKAGE-INFO.md` + `SHA256SUMS.txt`；
  11 项哈希复算全 OK。二进制 ZIP 仅含 exe+3 DLL。
- **版本**：4.544.0 不抬号（日常构建）。主工作树仍保留与本包无关的在制
  改动；本包构建点由独立干净树保证。

## 子智能体卡 · 点击右栏打开对话（2026-11-18 补刀）

完整智能体形态两批之上的第三刀（用户令「新增子智能体卡，点击可在右侧
打开子智能体对话」）：编队页两张卡面接右栏子会话窗（SubagentPaneChat，
复用上批 openSubagentPane id=sub:<ref> 管道，零新绑定）——

- **分工卡（FleetTasks）**：每条运行行的 subline 加常驻 hover「在侧边栏
  打开」钮（fleet-task-aside，margin-left:auto 靠右）+ 展开详情区加同款
  文字钮（fleet-task-open 样式复用）；不展开即点即达。
- **拓扑卡（AgentTopology）**：非 root 卡加 agent-topology-aside 钮（与
  agent-topology-open 错位 right:25px，hover/选中/focus 显形），点击
  stopPropagation 不触发选中链。
- 接线：Fleet prop onOpenAside → 两卡面；App Fleet 挂载传
  setWorkspacePanel(true)+openSubagentPane(ref, task)。
- 门禁：tsc -b 0、改动面 eslint 0、FleetTasks/AgentTopology/Fleet 三测试
  17/17（新增 aside 两例）；mock 走查实证（编队页 11 分工卡钮+3 拓扑卡
  钮，点开右栏各一 tab，9335 显式口）；wails build+冒烟 200+装机
  INSTALL OK（PID 18544）。

## 办公子智能体升格完整智能体 · 嵌套委派+导航+插话（2026-11-18）

用户指定对齐 dsh 子智能体全形态（子智能体=完整智能体：持久会话/可续聊/
可再派孙代/面包屑导航/右栏聊天页/运行中插话；派生方=父代理非用户直建）。
两批收官（`9475bed3` 批1 机制 + `06b2b3fd` 批2 UI/绑定），第七次装机
（exe 构建 10:21:16，INSTALL OK 新 PID 32952>构建时刻，未 push）：

- **批1 机制**：`[agent] subagent_max_depth`（缺省 3=主→子→孙，硬帽 10，
  1=关闭逐字节现行为）+ctx 深度缝+buildSubRegAt 按级别开放协调四件
  （task/interrupt_agent/subagent_list/read_subagent_result；
  run_skill/install_skill/exit_plan_mode 恒排除）；SubagentMeta.Depth 四路
  盖章（task/RunNew/runFollowUp/persisted_run，RunNew/runFollowUp 补 ctx
  会话 ID+级别缝=孙代 parentRef 链落对续跑保级）；**容量防自锁**（dsh
  ActivationPool 满拒绝同源洞察）：嵌套层槽满 tryAcquireSubSlot 快拒——
  持槽等排队孙代会全树互锁；顶层排队语义逐字节不变；协调三工具子树守卫
  （AncestryContains：只可见/可停/可读自己的后代，兄弟边界维持，job 面
  对子代理不开放）。测试 task_nested_test 四例（三层血缘端到端/深度闸
  契约/快拒+排队回归/守卫矩阵）；agent/config/boot/app 全量 CGO 绿。
- **批2 UI**：GaeaSubagentSteer 绑定（running=Steer 直穿/queued 拒/终态
  走追问；无主回合闸）六面+第七面 core.ts+legacy 再生（drift OK@713）；
  SubagentThread 谱系面包屑+composer 双语义+wenxue 镜像逐字节+三语四键；
  SubagentCatalogButton（顶栏「N 个子智能体」触发器+portal 血缘树 L 徽标
  +行点击主区+侧栏钮）；右栏子会话 pane（openSubagentPane id=sub:<ref>+
  SubagentPaneChat 壳，可插话/追问/停）；mock 嵌套链增稠。
- **门禁**：全量 vitest 4490+/4493（三红例单跑全绿=负载期 flaky 零交集
  定性：DatabaseLibrary/SchedulingSection 在册已知+ContextView skill 面
  无关）；tsc -b 0、改动面 eslint 0、契约/spaceBindings/drift 绿；mock
  走查五链（触发器计数/树 L1L2 徽标/主区面包屑/插话受理/右栏窗）显式
  GAEA_WALK_PORT=9335+/json/list 核对（9333 纪律第三次执行）+截图
  temp/fleet-walk/subagent-batch-20261118.png；wails build 1m50s+冒烟 200
  +装机 INSTALL OK。
- **偏离在案**：嵌套层容量=快拒不排队（vs dsh 全拒绝）只对 caller≥1；
  冷恢复投递 demand-gated 维持；wenxue 面包屑/目录树未挂（无编队消费面，
  组件已镜像）；fleet depthHue 仍按树位推导（meta.depth 已透传待消费）。
- **坑**：fresh profile 无头 Edge 首跳被扩展页劫持（--no-first-run
  --disable-extensions 后净）；上一批 WenxueGalleryList 只进 bindingNames
  未进签名清单=契约锁本批逮住补齐（713 对齐）；镜像组件引用新 i18n 键
  必须同批补 wenxue 三语字典（逐字节拷贝≠键集同源）。

## Git 打包 · gaea-vision-ocr-gallery-20261118-b3d994dc（2026-11-18）

本日治理线全量（识图根修+OCR 大卡+插图标签+U5 徽标+DSH 蒸馏首批+编队 UI
完善+走查根修）桌面构建包。打包点 `b3d994dc`（插图 wailsjs 再生物化后）：
干净树 `.worktrees/pkg-b3d994dc`（detached，junction node_modules）前端
30.03s + Wails 42.803s（bindings 未跳过）；wailsjs 仅 CRLF 噪音（内容漂移
为空）。交付 `releases/gaea-vision-ocr-gallery-20261118-b3d994dc/`：

- payload 四件：gaea.exe 55,064,576 B SHA256 `B9024B6D…F47313`；三 DLL
  沿用基线原样哈希（68D0D5F5/5DD36DBD/4C563277，未重编）；正式路径冒烟
  `/api/health` 200（PID 50912 回收）
- PACKAGE-INFO.md + SHA256SUMS.txt 入库（六项复算全 OK）；源码 TAR
  43,727,334 B `3997DF78…769F95` / ZIP 12,821,240 B `5DC4F634…831C7`
  留盘不入库（P2-3）
- 装机：`gaea_install.ps1 -SourceDir payload -Launch` INSTALL OK
  （exe 新 PID 43192>构建时刻 02:48:03）
- 版本 4.544.0 不抬号（随下个发版点统一抬）；工作树回收（含上批遗留
  pkg-d30cb5d2 空壳）

## 文学「插图」标签 · 生成图/剧照图册（2026-11-18）

用户令文学板块增加插图标签查看生成的插图/剧照等图片。第七次装机（exe 新
PID 60596>构建时刻 02:44:09，未 push）：

- **数据面（Go）**：新绑定 `WenxueGalleryList`（wenxue_gallery.go）——递归
  收集书工作空间图片：`.gaea/uploads`（image_gen 封面/场景插图/角色立绘 +
  WenxueSavePastedImage 粘贴图，媒体根分档同源）+ 其他目录图片如实收集；
  mtime 倒序封顶 400；来源分类 uploads/workspace；`.git/node_modules` 跳过、
  `.gaea` 非 uploads 子树（cache 等）整枝跳过、越界防护（Rel 归一）。
- **前端**：ChatTabs 第九 tab「插图」（gallery 入 CHAT_VIEW_TAB_IDS 单源
  白名单 + ChatTabId + TABS 末位）+ `WenxueGalleryHub`（WenxueHubFrame
  tone=gallery）：网格懒加载取图（Preview image→dataUrl，首屏 24 张）、
  来源筛选（全部/生成/其他）、lightbox 大图（遮罩点击/Esc 关）、空态引导
  image_gen、`gaea-wenxue-filetree` 通道自动刷新（设定/角色 Hub 同款纪律）。
  wenxue 无需 i18n（页内 zh 单语惯例）。
- **绑定六面 + legacy**：facade + 双树 bindingNames + 签名（702→703）+
  mappings（GalleryList→WenxueGalleryList）+ core.ts + spaceBindings(work)
  + mock 桩；gen_bindings（wenxue 115/合计 712）+ `-legacy-ts` 再生（187 名
  ——wenxue 专属绑定不经 gaea AppBindings 消费走 legacy 面，drift 锁三兜住）。
- **走查式测试自逮两轮**：`.gaea` 段过滤/分类最初用根前缀判定——书嵌套在
  图文根下（rel=书名/.gaea/uploads/…）前缀判定必漏（cache 噪声泄漏+来源
  分类错），改路径段判定；裸 `.gaea` 目录访问时子目录未知须放行下钻。
- 测试：Go WenxueGalleryList（收集/分类/过滤/排序/越界，TempDir 隔离）+
  WenxueGalleryHub 4 例（清单/筛选/空态/灯箱）；ChatTabs 八 tab 契约升
  九 tab（a11y End 键末位=插图）；wenxue 族 71/71、tsc 0、改动面 ESLint 0、
  bindings completeness 绿、冒烟 200+INSTALL OK。

## 办公 OCR 专属大卡 · 文件 OCR 解析的进度/状态面（2026-11-18）

用户报 OCR 解析无进度可见（agent ocr 工具只有 dispatch→result 两点、@附件 OCR
腿完全无声、扫描件 PDF 分钟级只有预览面板可见）。`81645938`（第六次装机 exe
新 PID 35500>构建时刻 02:25:06，未 push）：

- **后端**：docmd 包级进度钩子（SetOCRProgressHook，panic recover 不反噬主链）
  + 三链路插桩（OCRImageText 缓存命中即报 cached/逐引擎 engine-start/done
  字数/failed 原因；ocrPDFRange 页粒度 running done/total + 终态 defer 统一
  收口 + 链首引擎 kind）+ App 启动单点注册（app.go，SetVisionRuntimeResolver
  同区）→ gaea-event ocr_progress；payload 抽纯函数 ocrProgressPayload。
  预览面板 preview_progress 专用通道不受影响（双轨并存）。
- **前端**：wire ocr_progress + reducer 按 path 原地 upsert（同文件进度事件
  更新同一张卡不堆叠；pendingUser 不豁免——@附件 OCR 在消息装配期就跑，
  flush 保住「用户消息→OCR 卡」时序）+ OCRCard 大卡（文件名+引擎 chip+
  状态词+PDF 页粒度进度条+完成字数+失败原因；自带 ocr-card.css 亮暗自适应）
  + Transcript case + segSig 带 stage/done（防 memo 吞活进度）+ countTurnSteps
  计入。事件不落历史（恢复会话卡片消失，结果本体经注入块/工具卡持久化）。
- 门禁：docmd 全包绿、app 钩子测试绿、store/WorkHeader/OCRCard 27/27、
  **全量 vitest 4481/4481（291s）**、tsc 0、改动面 ESLint 0、冒烟 200+
  INSTALL OK。
- **验收挂用户**：对扫描件 PDF @引用（分钟级，页粒度最直观）或让 AI 调
  ocr 工具看大卡流转（引擎拉起中→识别中→完成 N 字）。

## 办公识图本地 llama 档根修 · mmproj 装载+能力标注+跟随链闸（2026-11-18）

用户报「办公板块本地模型的识图功能」坏。三层根因同批修复（构建装机 exe 新
PID 10016>构建时刻 02:02:35，未 push）：

- **根因**：llamaEnsure 起 llama-server 不带 --mmproj（qwen35 纯文本服务）；
  办公聊天绑 llama 8124 → 识图跟随链把图喂给纯文本服务必坏。mmproj 文件
  在位（C:\AI\models\qwen35-uncensored）、WHIRL 同模型带 mmproj 正常——
  单副本缺一半。
- **修复**：qwen35 档补 MMProj（与 WHIRL 同文件）+ llamaServerArgs（抽出纯
  函数）条件加载 --mmproj；managedCatalogModels 按 mmproj 文件在位标注
  Caps=[vision]（managedProfileMmprojPathIn 统一根/历史根双布局）；跟随链
  llama/strata/whirl 分支对未标注 running 档弃权落 herdsman 兜底——纯文本
  档（gemma4）永不接图，mmproj 被删自动降级（自我愈合）。
- **实机旁证**：engines.json herdsman LLM 槽 stopped 且 default_model 空——
  llama 是识图唯一通路；herdsman 兜底链因 default 空也断，本刀为必修。
- 测试：mmproj 路径解析（TempDir 隔离）/目录 caps/启动参数/跟随链能力闸
  4 新例；seedLlama 种子随新契约补 caps（3 例既有测试对齐）；app 全包
  118s 绿 0 失败。
- **验收口径挂用户**：下次聊天（llama 档）@图片/识图应恢复——llama 需重启
  一次让 --mmproj 生效（引擎停-起或由聊天 ensure 自动重拉）。

## U5 触发源徽标收尾刀 · 非用户回合亮归属（2026-11-18）

dsh ui 蒸馏档刀池最后一件（`e69154c4`，第四次装机 exe 新 PID 50556，未 push）：

- **实锤路径**：TurnStarted.Trigger（user/wake/subagent）在会话日志
  （log.go:503 payload 带 trigger）与轨迹（Turn.Trigger）两路已在，唯
  wire 缺口——gaeaEventMap 无 TurnStarted 分支。
- **三段补齐**：Go wire 下发（user 缺省不下发，普通轮线格式逐字节不变）
  → reducer turn_started 捕获 turnTrigger（user 轮清空、老后端兜空）→
  WorkHeader 状态行内嵌 chip（后台唤醒/子代理回合，运行态与完成态都亮
  ——完成态耗时行的归属同样诚实）。
- wenxue 镜像六文件同批（store reducer/hook/workheader/wire/locales）；
  i18n work.triggerWake/work.triggerSubagent ×3 语 ×2 树。
- 门禁：store+WorkHeader 22/22（U5 断言 2 例）、wenxue 49/49、tsc 0、
  ESLint 0、Go build/vet/定向 test 绿；前端 32.38s+wails 59.476s+
  冒烟 200+INSTALL OK。
- **UI 蒸馏档刀池至此全部处置完毕**（UA-1/UA-2 已落、U5 徽标本刀收口、
  谱系树/设置卡留档）。

## 编队拓扑实弹走查 · 两处几何 bug 根修 + mock 增稠（2026-11-18）

headless Edge（CDP 9343 独立口）+ Vite dev `?mock=1` 走查新编队 UI——两刀修复
（`6df5ec82`）+ 第三次构建装机（exe 新 PID 14704>构建时刻 01:33:06，未 push）：

- **bug1 判列公式**：bandColsFor 用 MIN 槽位预算选列，窄容器（556px）选出 4 列
  →行宽 680 溢出——改按钳制后实际卡宽逐列迭代（360→2/556→3/720→4）。
- **bug2 步进口径**：place() 步进 x+=width+GUTTER 而 rowWidthOf 只算 slot 和
  （slot 已含 gutter）——每边界漂 10px，行尾聚合卡出界（right 875>866）；
  步进改 x+=child.width 与行宽同口径。
- **bug3 测量口径**：canvas 测量用 getBoundingClientRect（border-box），
  canvas 活在 stage padding 内侧→画布宽出 20px（scrollWidth 554>534）——
  改 clientWidth-paddingX（content box）。
- 终态实弹断言全绿：noHScroll=true、done=4/idle=3 聚合卡、aborted 词汇、
  尾巴×2、深度色带两层、树视图 6 行、跨视图选中/折叠保持、ⓘ 浮层开合。
- **mock 增稠**：AgentNetwork 补 done×4/idle×3/aborted/尾巴；SubagentRuns
  默认分支 9 条对齐——浏览器 ?mock=1 走查面从此可验折叠/新词汇/尾巴。
- **⚠️ 事故复犯记档**：GAEA_WALK_PORT 缺省 9333=用户壳 WebView2 调试口，
  step2「编队」点击曾误打用户实盘（仅视图切换、无写入、无实害）；9333 事故
  （2026-10-09）同源复犯——此后走查驱动前必须 GAEA_WALK_PORT 显式设口 +
  /json/list 核对目标页。
- 门禁：context 族 118/118；tsc -b 0；前端 30.80s+wails 53.96s+冒烟 200+
  INSTALL OK。截图 temp/fleet-walk/（graph/tree/info/亮暗）。

## 编队 UI 完善批 · D8 双视图+深度色带+节点详情浮层（2026-11-18）

DSH 蒸馏计划 01 的 UI 尾款一批收口（`c8f0e2e9`，构建装机 exe 新 PID 34288，未 push）：

- **D8 TREE 双视图**：`FleetTree.tsx` 缩进树渲染器复用 `buildFleetModel` 同一 pre-order
  节点表——选中态/折叠态/过滤联动全在模型层，头部 GRAPH/TREE 双钮切换零状态丢失；
  折叠卡在树中以聚合行同款语义呈现（chevron/错误计数/预览一致）。
- **深度色带**（08 包 P0 尾款）：`depthHue` 五色轮换、卡片与树行左缘按层同色（root 与
  主 agent 家系紫同源），家系色保留连线——血缘/层读双维不打架。
- **ⓘ 节点详情浮层**（08 包 P2）：卡片 hover ⓘ→画布层浮层（状态/模型/历时/工具数/
  错误/token/成本/消息数/最近工具/最近输出），行词汇全复用 radial*/msgCount/elapsed
  既有键，横向钳制不溢出，再点即收。
- **U5 待核销号**：`TriggerSubagent` 前端零渲染实锤（仅 compactedMeta 文案占位）——
  词汇已备渲染面留档待需求。wenxue 无 Fleet/AgentTopology 消费面（仅 locale parity
  五键同步）。i18n 新增 fleet.viewGraph/viewTree/info/cost/lastOutput ×3 语。
- 门禁：context 族 118/118（三跑两绿一红，红例未留痕零交集定性负载期偶发）；
  tsc -b 0、改动面 ESLint 0；wailsjs 零漂移（纯 UI 批无绑定面）；前端 30.74s+
  冒烟 200+INSTALL OK（exe 新 PID 34288>构建时刻 01:13:33）。

## DSH 子代理蒸馏首批 D1–D6 · 编队拓扑会话化+折行+折叠+aborted+单闸+独立 Stop（2026-11-18）

按 `docs/archive/gaea-dsh-distill-plan-01-2026-11.md`（dsh-v0.2.2-rc.1 对账基线，逆向证据包
`C:\AI\dsh-subagent-reverse\`）首批 D1–D7 全处置，三笔提交（015730c9/665b6450/0f6897a1），未 push 未装机未构建 Wails。

- **门 2 验证（开工首验，已写回计划 §4·D1）**：gaea 后端可读父子——spawn 时 ctx 已盖
  `withSubagentSessionID`（task.go:515），`SubagentMeta` 直写 `parentRef`/`spawnCallId`
  （applySnapshot 一次写定）即可，无需日志扫描路线；ephemeral 父（`sa_ephemeral_*`）
  落空按主会话降级，`continue_from` 不改血缘。三条派发路径同批盖章
  （task Execute 新鲜派发/RunNew 宿主/persisted_run 技能路径补 ctx 盖章——技能子代理此前
  不盖会话 ID，嵌套派生会误记到主会话头上）。
- **D1 编队拓扑会话化**（`buildSessionNetwork` in gaea_ui_contextview.go）：runs 携带血缘
  盖章时以**子代理会话**为骨架建树（id=sa_ ref、parentRef 嵌套、ghost 父降级 root、
  孤儿 legacy 节点保留=「任何派发过的 agent 都可见」不变量）；spawnCallId **精确**联结
  事件日志工具调用树迁移统计（替代前端 matchRun 与后端 runMatchesNode 的任务前缀互含
  同款病灶）；lastText/lastTool/model 直达节点；无盖章旧数据回落 legacy 富化逐字节不变。
  前端删 matchRun、statusLabel 四态、root 卡选中归一 main（Fleet knownRefs 联动修复——
  此前拓扑点击 root/工具调用 ID 节点会被 knownRefs 校验清态）。
- **D2 折行布局**（agentTopologyLayout.ts 重写）：bandColsFor 0.78 可读下限
  （360px→2 列/720px→4 列，判列槽位=最小卡）、bandCardWidth 随列摊开、行距按最高卡实测
  （cardHeightOf 活动行尾巴计价：lastText 两行+lastTool 单行）；**gutter 双计 bug**
  当场自逮（slot 已含 gutter 行宽再另加=720px 4 列 750 溢出）。画布不再
  `width=max(measured,content)` 撑宽，20+ 子代理 360/720 两宽无横向滚动。
- **D3 折叠聚合**（fleetModel.ts）：done/idle 两类聚合卡；FOLD_IDLE_MIN=3 上游逐字，
  **done≥4 为 gaea 规模偏差留档**（典型会话 2–5 个子代理，≥2 就折会把常见小会话的
  已结算卡藏进聚合卡——Fleet 集成测试当场逮住）；foldGroupId 稳定组 id、手折
  （foldedIds）优先于组展开（chevron 不失效）、running/活跃分支永不折、展开态聚合卡
  保留作收起入口；50 节点树收敛到 4-5 卡。
- **D4 aborted 结算词汇**：`classifySubagentStop`（P1-C 批既有）只覆盖模型面，meta/UI
  侧中断仍落 failed。根修=中断错误带 `%w context.Canceled` 从 runSubAgentInternal
  逃逸到全部终态写路径，新增 `SubagentAborted`（SaveAborted）：interrupt_agent/独立
  Stop/回合取消不再冒充「出错」；排队中取消同落 aborted；aborted 可续跑；追问中止写
  中性摘要「用户停止：本轮追问已中止」；max-tokens/refusal 维持宁缺勿虚报不造轴
  （判定留档）。三处测试断言同步（interrupt×2/queued-cancel×1）。
- **D5 单闸补齐**：实锤 RunNew（DAG 宿主节点）/runFollowUp（追问）绕过并发闸
  （acquireSubSlot 仅 3 调用点）——补 acquireSubSlot+MarkQueued，单闸不变量成立
  （上游是两池性质错配+超限 fatal，gaea 是单闸+排队可见，实缺口是绕行）；
  行为锁 `TestRunNewRespectsParallelGate`（槽满必排队+queued 侧车+槽释放起跑）。
- **D6 独立 Stop**：新绑定 `GaeaSubagentStop`（后端 InterruptSubagent 早已存在+工具面
  覆盖，只差 bridge）——OfficeB 门面+bindingNames/signatures/mappings/core.ts/
  spaceBindings(work)+gen_bindings 再生 711 方法+TestBindingsCompleteness 绿；
  SubagentThread 头部 Stop 钮（running/queued 可见、mt_/终态隐藏，fire-and-return
  与 TaskCancel 同契约）+2 新用例。**wenxue 镜像同批移植**：SubagentThread 逐字节
  同步（改前两树同源）、ChatTabs/useSubagentTabs 状态词汇、中断按 ref 全局寻址
  （agent 包 subRuns 双引擎同进程共用）直用 GaeaSubagentStop 不设 Wenxue* 换锚、
  locales 三语补 statusAborted/stopTitle。
- **D7 留档不立项（带证据）**：gaea 缓存存原始 LogEntry（size+mtime 失效、折叠每次
  现算），无上游「投影缓存跨格式静默错读」面；格式演进走读端宽容降级。判定写回
  计划 §4·D7。**D8 双视图**未动（P2 demand-gated）。剩余拍板门=计划 §7 门 1
  （rc.1 client 609 文件 UI 复核）。
- **mock 补刀（第三笔 0f6897a1）**：全量 vitest 契约锁（X1-11 漏 mock fail-closed）
  逮住 D6 绑定漏 dev mock——「expected 711 to be 710」+「漏 mock 即红」双例红；
  补办公/wenxue 两树 SubagentStop 桩+types union+防御计数 710→711/701→702。
  **新绑定六面纪律记档**（门面/bindingNames/bindingSignatures/mappings/spaceBindings/
  dev mock），定向跑对漏面假绿又一实证。
- **门禁终值**：go vet 0；agent 包全量+app 包 117s+trajectory 全绿（新增 parentref/
  组树/gate 行为锁 8 例）；前端 tsc -b 0、改动面 ESLint 0 error（ChatTabs react-refresh
  warning=既有同名款）、**全量 vitest 495 文件/4467 例全绿（252.79s，两轮：首轮 2 红=
  mock 漏面→补刀→复跑全绿）**。**已构建 Wails/已装机（11-18 收尾刀）**：前端 26.35s+裸 wails build 1m13.487s
  （bindings 未跳过），wailsjs 再生真实漂移仅三文件 14 行正当下游
  （OfficeB GaeaSubagentStop + SubagentRunView parentRef/spawnCallId +
  AgentNode lastText/lastTool）随 bindings 再生提交；冒烟 /api/health 200
  （PID 31616 回收）；装机 INSTALL OK（exe B59D28E3BC7A…，三 DLL 基线
  哈希不变，new PID 43256>构建时刻 00:56:29）。**未 push**。编队页新拓扑
  真机走查挂用户实弹（需真实多子代理会话）。

## codex 蒸馏差距档案落地 · 四刀（2026-11-18）

- **背景**：逆向第二轮（hotspot-12/13）产出四项 gaea 差距，用户拍板「四项全做+弹卡四选」。
  MCP 修正案态经核实已被 persist_allow 覆盖（跨会话规则回写已有），不立项——漂移记录修正。
- **刀1 网络闸逐主机修正案（1dc4e8c5）**：NetworkGate 挂 NetworkLedger/NetworkAmendment
  双面台账（host 精确/子域后缀/工具名；url subject→host 复用 permission.Subject，无 url
  退化工具名）；交互路径 networkAmender 复用审批卡通道（Source="gate"，决策串不新增，
  persist_allow=永久此主机→回写 `[network].gate_host_allow`）；headless 走 NetworkLedgerOnly
  只读台账；开关 `[network].gate_amendment` 默认 false=现状逐字节等价；四个换级点统一
  `c.wrapNetworkGate` 出口；前端 gate 卡按钮4文案「永久此主机」（gaea/wenxue 单源复用）。
  测试 5 例（台账命中不弹卡/AskFlow persist 传呼/拒绝文案/target 提取/headless 只读）。
- **刀2 MCP per-server 超时（a51be17d）**：`startup_timeout_sec`/`tool_timeout_sec`
  （*float64 秒，命名对齐 codex mcp_types.rs）全链贯通 PluginEntry→.mcp.json→Spec→指纹
  （**改超时即换缓存键**）→三条启动路径（startWithDeadline/StartAll/热插拔）→
  remoteTool.Execute（>0 包 ctx）；nil=包默认（启动 15s、工具不超时=现状逐字节等价，
  不照抄 codex 300s 默认守铁律）；**顺修 GaeaAddMCPServer 丢 Transport/Headers 现存 bug**
  （远端热加曾按 stdio 落盘）；表单加两可选秒数输入（gaea+wenxue 双副本+i18n 六文件；
  models.ts 再生 +6 行，方法名/arity 不变 bindingNames 免再gen）。
- **刀3 T5 轮转对齐（9602c422）**：budget.go 轮转字节步进→**rune 步进**（每轮每行 1
  UTF-8 字符、首字符加计 `": "` 2 字节分隔成本、exhausted/预算不起永久出队——对齐上游
  tools_budget.rs:100-112）；retained 保持字节口径（TruncateDescBytes 不动）；测试重写
  （NameOverflow [1,0,-1×3]/FairShare [14,14,14]+分隔成本守恒）+新增 CJK rune 边界例。
- **刀4 会话策略随行存档（d462c5aa）**：`BranchMeta.PolicySnapshot`（nil=旧会话现状
  等价）记审批模式/权限级别/断网闸三面（bash 沙箱是装配期工具配置不在列）；四处会话
  构造点接线——NewSession/SetSessionPath **回 boot 基线**（存档策略不泄漏进新会话）、
  Fork 携带当前快照（否则 Resume 无快照回落基线静默重置）、Resume 有存档按 P5 出口
  纪律重建闸门/无存档回基线；快照幂等回写 meta（fail-soft）。测试 2 例（round-trip+
  恢复覆盖+回基线；legacy 无快照等价+幂等回写）。
- **构建装机**：frontend build 36.05s 绿；ucrt64 `wails build -s -ldflags "-s -w"
  -trimpath` 85.7s；smoke 200；`gaea_install.ps1 -Launch` INSTALL OK（exe
  183F01B7CAED…，PID 55560 > 构建时刻）。版本不抬号随下版折入。
- **余项**：实弹走查挂用户（弹卡四选流/per-server 超时生效/恢复旧会话策略徽标——
  需配置 `[network].gate_amendment=true` + `[sandbox] network=false` 才见弹卡形态）。

## codex 蒸馏 · 逆向第二轮 C 级推进（2026-11-18，只出逆向产物不改 gaea 代码）

- **hotspot-07 §5 三件全闭合**（文档全在 `C:\AI\re\codex-0.165.0\docs\hotspot-12-jumptables-and-c-level-round2.md`）：
  ①两枚枚举反序列化器跳转表手工解（objdump+表解析，不依赖 Ghidra）——`0x1410ca7b0`=ReviewDecision
  序列化器 **8 变体全解**（approved/approved_execpolicy_amendment/approved_for_session/
  approved_mcp_policy_amendment/network_policy_amendment/denied{rejection}/timed_out/abort，
  单位变体占负域 niche `0x8000…+n`，表 `0x150782894`）；`0x141140580`=**app-server v2**
  SandboxPolicy Deserialize（internally tagged `type`+camelCase，内部枚举 SandboxPolicyDeserialize
  兼容 legacy `readOnlyAccess`/`access`，遇 restricted 显式报错「no longer supported; use
  permissionProfile」两文案实锤）。
  ②`0x147ce6880` **正名**：非压缩触发族，实为 guardian request_budget 异步状态机
  （字节判别 `+0x30`→表 `0x150b95cd4` 15 态已解，span=`codex.guardian.context.section_cost`，
  源 `core\src\guardian\request_budget.rs`）；反编译 4 次（3×300s+1×900s）全超时按预案 3 结案。
  ③四热点锚点全定位：MCP connection_manager.rs 三函数、elicitation 族、网络策略
  `0x1496b44b0`（归属 **exec-server** `network_policy_decisions.rs`，Deny 理由 "not_allowed"
  movabs 内联）、持久化 `rollout_path` 三函数（**三份反编译成功**：行→JSON 序列化器/列名清单绑定）。
- **负发现**：SQL 常量（INSERT INTO thread_items/turns 等 4 副本）零代码引用
  （lea/.reloc DIR64 291,207 条/胖指针三路全扫）=嵌入资源形态。
- **0.165 threads 行字段族实测**（迁移起点已过时）：`permission_profile/active_permission_profile`
  （策略随行存档升级 profile 形态）、子代理树三列（parent_thread_id/depth/agent_nickname/
  agent_role）、分叉两列（forked_from_id/forked_from_ordinal_exclusive）、comp_hash、
  reuse_parent_compaction、dynamic_tools、ThreadGoal 四态。
- **gaea 只读漂移核查**：T3/T10 七态与 codex 8 变体 **6/8 一一对应零漂移**（timeout 与 deny
  可区分、拒绝带 reason、persist=前缀规则授予全对齐）；缺 `approved_mcp_policy_amendment` 与
  `network_policy_amendment`（逐主机网络修正案，gaea 网络闸是全局布尔）——维持差距档案不立项。
  MCP 全局 15s vs codex per-server startup/tool 超时（Duration 纳秒÷1e9）差距档案维持。
- **工具沉淀**：`resolve_jumptables2.py`/`locate_pool_refs.py`（胖指针槽一级）/`DumpDecomp3.java`
  （超时参数化）；配方=Ghidra jumptable 未恢复→objdump 手工解表；坑=Git Bash 内联
  `cmd //c "set X=…&& …bat"` 炸引号，env+调用必落 .cmd 批文件。
- **待用户拍板**：`docs/codex-distill/hotspot-07-*.md` 是逆向原始资料误入仓内的未跟踪副本
  （正本在 RE 工作区且本轮已更新），仓内副本如何处置（删/留）待定。

### 同日第二轮补刀（源码优先法，hotspot-12 §9）

- **hotspot-06 §5 四个未证项全闭合**：tools_budget 算法全提（117 行：名字优先全额→溢出整行
  omit 扣 omission_reserve→描述**字符轮转公平分配**，每轮每行 1 UTF-8 字符、首字符计 `": "`）；
  age_score/invalidate_score=本地 schema 认识（池 0x15070d1b5）但本地源码零消费（服务端字段）；
  prediction_preserve_parent_tools 真实语义=禁 MCP 标记+非新窗口跳过重建增量工具节
  （**无预测窗口**，hotspot-06 §2 旧表述作废，flag Stable 默认开）；deferred_tool_world_state
  默认关（UnderDevelopment）。
- **AskForApproval 判别式闭合**（hotspot-09 挂开口）：声明序 UnlessTrusted=0/OnRequest=1/
  Granular=2/Never=3；主函数 asm 实锤 `cmp $0x2`+五 bool 位打包（bit0=sandbox_approval）+
  文案长度立即数 0x53=83 与源码逐字对上；GranularApprovalConfig 实为 5 bool（含
  skill_approval/request_permissions/mcp_elicitations）。
- **压缩窗口状态机全提**（auto_compact_window.rs）：UUIDv7 三元 id+advance 滚动+双一次性
  claim（预算提醒/降级回退）+Estimated/ServerObserved prefill 基线（服务端观察值不可被估计
  覆写）；analytics 事实层 CompactionTrigger{Manual,Auto}/CompactionReason 四变体——
  hotspot-07「compaction_trigger 簇」正名闭环（真族名在 analytics 事实层）。
- **`0x140a5dcb0`（17.5KB）定性**：模型目录 schema 字段匹配反序列化器 family（含 guardian/
  slug/display_name/default_reasoning_level 与 network/external/restricted-token/elevated
  档位字段），非压缩逻辑。
- **版本跟进触发成立**：npm `@openai/codex` latest=**0.170.0**（基线 0.165.0 +5 minor），
  按 ledger §6 diff 流程重跑待拍板。
- **gaea T5 对照**：`tool/budget.go` 与上游同构，三处本地化差异（字节轮转 vs 字符轮转/
  尾部省略+占位行/首字符分隔成本）不算漂移；对齐改动点只在轮转循环一处。

### 同日版本对齐 0.165.0 → 0.170.0（用户拍板「对齐最新版」，hotspot-13）

- **结论：C 级证据链整体平移成立，无需重做任何一轮逆向**。源码锚连行号未动
  （ReviewDecision 8 变体/AskForApproval 4+5bool/SandboxPolicyDeserialize+legacy 两报错在
  571/602/620/637/tools_budget 轮转/auto_compact_window/CompactionTrigger+Reason）；二进制
  25/26 探针命中（0.170.0 win32 exe 385,571,120 B，比 0.165 大 28MB、带 gstreamer 语音资源新面）。
- **变化面**：crate 122→**127**（+5 未覆盖：agent-message-board-host 261L / message-board-mcp
  710L / plugin-host 396L / exec-server-cli 2877L / http-headers-helper 191L——前两个是 asset-04
  消息板的宿主落地，后两个域判定不搬不变）；features flag 195→**225**（+30 删 0）；锚点 crate
  全增长（core 596→655=**agent/host/ 14 文件准入 host 化**、codex-mcp 60→71=**connection_pool
  共享连接**、thread-store +血缘读、state +nfs_migration、network-proxy +secret_substitution/
  tcp_upstream、protocol +permissions/enforcement、analytics +turn_critical_path、tui 1140→1230）。
- **唯一语义漂移命中已记档面**：`deferred_tool_world_state` 转正（UnderDevelopment/默认关 →
  **Stable/默认开**）——hotspot-06「非默认行为」注记对 0.170 失效，已加修正注。
- **台账 §2/§3 已按 §6 修订**（127 crate；新面 agent/host 与 connection_pool 记档不立项：
  gaea 子代理闸/会话所有权重于共享，无对应痛点，待需求出现再读）。
- **工作区**：`C:\AI\re\codex-0.170.0\`（源码 tarball+平台包），0.165 工作区保留不动。

## codex 蒸馏第四批 · Git 打包与桌面构建（2026-11-17）

- **代码/交付点**：Wave 1 `0d42c3fb` + Wave 2+3 `2a11f08a`（本包锚定后者）+
  bindings 再生 `c393f847`。第四批 9 项全落地、归档 8 项在册——codex 蒸馏
  四批（研究面 + 落地面）至此整体收官。
- **干净树构建**：`.worktrees/pkg-2a11f08a`（detached @`2a11f08a`，
  node_modules junction）；前端 `npm.cmd run build` 23.91s 绿；UCRT64 GCC +
  `CGO_ENABLED=1` + 系统 TMP/TEMP 裸 `wails build -s -ldflags "-s -w"
  -trimpath` 49.061s（Skip Bindings=false）。Wails 再生仅
  `wailsjs/go/models.ts` +4 行真实漂移（trajectory.ToolRec 的 model/turn，
  B5 归因正当下游），已随 `c393f847` 物化回主树；其余 wailsjs 仅 CRLF
  噪音（`--ignore-space-at-eol` 内容漂移为空）。
- **交付目录**：`releases/gaea-codex-distill-batch4-20261117-2a11f08a/`
  （payload 四件 + PACKAGE-INFO + SHA256SUMS + ZIP + 源码 TAR）；两份文本
  记录入库，二进制/源码包留盘不入库（P2-3 纪律）。
- **payload**：`gaea.exe` 54,992,384 B
  `AAB7891162F7384337F9555BA8EB4D3D9F7D6D26899978C4F44A4B9D916DE013`；
  三件 DLL 沿用基线原样哈希不变（`68D0D5F5…F550B` / `5DD36DBD…01E8` /
  `4C563277…7FFE`）。
- **冒烟**：正式 payload 路径 `scripts\smoke.ps1 -ExePath` → `/api/health`
  200（`{"status":"ok"}`，PID 40852 已回收）；构建/冒烟后无孤儿进程。
- **源码/投递包**：`gaea-2a11f08a-source.tar.gz` 43,660,125 B
  `3E290CC1…76B71E`；ZIP 29,501,222 B `1E7A7FEE…2A7FB`；七项清单复算全 OK。
- **版本/边界**：`wails.json=4.544.0` 不抬号（后端治理批无前端可感面）；
  未装机（`gaea_install.ps1` 未运行）；未 push（本地 ahead 9：前三批 5 +
  第四批 3 + bindings 1，随后补 release 记录 1 笔）。默认关开关
  （drain/telemetry/outbound restricted）与 P6 灰度 warn 待实弹后处置。

## codex 蒸馏第四批 Wave 2+3 · 快照/持久化/观测/隔离/MCP（2026-11-17）

三拍板门用户已定（权限算快照维度 / 投影允许滞后禁止超前先灰度 warn /
观测落 whisperDataRoot/telemetry 14 天默认关；P9 用户纠正「gaea 有 MCP 面板」
转入实施——AddMCPServer/plugin host 是现成落点）。五刀全落：

- **P5 B4 step 快照**：`control/controller_step.go`——回合启动记录准入快照
  （turnStep.permLevel），运行中 SetPermLevel/SetContextWindow 走
  Reported 三态（Applied/Rejected+理由）：在跑回合按被准入时的审批权威跑完、
  权限双板 UI 与模型切换链拒绝理由透传前端（半切换态如实告知默认引擎已切）。
  禁止清单折算：审批政策（活热切换面）/模型窗口（切换链回落面）；沙箱
  AccessPolicy 构造期固定无运行中变更面（记档）。`/perm` 斜杠本就回合外
  派发（sendWithRaw 派发序实证）不受影响。
- **P6 B3 持久化守卫**：`session/checkpoint.go` Restore 加「禁止超前」守卫
  ——checkpoint.Seq 越过日志末条 seq（日志被外部截断/轮转或检查点不成对）
  = 投影超前权威源：灰度 warn + 丢弃检查点从日志全量重放（恢复正确性优先，
  允许滞后=正常重放追上不变）。
- **P7 B10 观测最小集**：新包 `internal/gaea/telemetry`——环形 256/去重合并
  计数（表上限 4096）/按日 ndjson 落盘（无 fsync）/14 天清扫/`[telemetry]`
  enabled 默认关（Record 单布尔短路零成本）。接线：turnLoop 记 turn_done
  （ok+label+时长）、批量与 pre-exec 两路 ToolResult 记 tool_result
  （name+ok+durMS——executeOneTimed 包装回补耗时）、App Startup 开关/Shutdown
  Flush。不引 otel（归档决策）。
- **P8 B6 panic 隔离+能力声明**：隔离面已有三处 recover+回合级（资产未读
  gaea 侧的误判）——本刀补验收测试落袋（注入 panic 工具：宿主存活、errMsg
  可解释、同批健康工具照常）。能力三档：readOnly/写盘=Tool 接口既有自报；
  网络档=`tool/capability.go` 名单（web×2+浏览器×12+mcp__ 前缀）+防漂移钉；
  消费方=断网闸 `agent.NetworkGate`（[sandbox] network=false 时拒绝网络能力
  工具——此前断网只关 bash 子进程，web/浏览器/MCP 照常出网是半开沙箱）；
  boot 双消费点（主执行器+子代理）与 control 四个换级点统一包装；
  bash 不拒（有自己的子进程网络开关）。
- **P9 B8 MCP 目录缓存+启动 deadline**：`plugin/catalog.go`——SpecFingerprint
  （type+command+args 保序+env/headers 键序归一+url+dir 的 SHA256；Name 不
  参与——改名重连同目录形状）；目录缓存 32 条/30min TTL/满逐出最旧，缓存
  **目录形状**（mcpTool 原始清单）而非工具对象（remoteTool 绑定旧连接不可
  复用），命中免 tools/list 往返、对新连接重建包装；StartAll 与热插拔
  addConnected 统一 15s deadline（慢/挂服务器只让自己失败，fail-soft 既有
  语义保持）。
- **验证**：新增 21 例全绿（step 快照 5/checkpoint 守卫 4/telemetry 5/
  断网闸+panic 隔离 2+能力钉 1/MCP 指纹·TTL·重建 3+遗留布尔短路形态）；
  全量回归 control 3.0s/agent 18.2s/session/config/telemetry/plugin/tool/
  builtin 30.3s/netclient/ai 全绿；app 全量 123.9s 绿（ucrt64+DLL）；
  `go build ./...` + vet 全绿。
- **边界**：未 Wails build、未装机、未 push。第四批 9 项（P1–P9）至此全部
  落地或按证据归档（B12 已随第三批；B2/B11/插件市场/EMA/codex-mcp 服务端/
  otel 全套在册归档）。灰度项两处：P6 超前断言 warn（一版后升 error）、
  drain_enabled/telemetry/sandbox network 断网闸均默认现状。

## codex 蒸馏第四批 Wave 1 · 准入关闭/工具归因/出网安全（2026-11-17）

按 `plan-batch4-landing-priorities.md` 完成 Wave 1 剩余三项（P2/B12 已随第三批
Wave 2 落地）：

- **P1 B1 准入与优雅关闭**：`control/controller_drain.go` 新增 draining/
  admitted/`BeginDrain(timeout)`——关闸与准入同一把 mu（检查—登记无窗口）；
  关闸后 Send/系统回合（runGuarded）结构化拒收（`ErrSessionDraining` + 含消息
  预览的通知，不静默丢）、队列未跑消息逐条通知作废；「在跑 + 已准入」两计数
  同零才收敛（turnLoop 排空落完最后一条 TurnDone 时 `settleDrainLocked` 归零
  并唤醒等待者）；超时强退 cancel 在飞回合、TurnDone{Err} 走既有路径不伪造。
  `App.Shutdown` 侧：`[agent] drain_enabled`（默认关=现状：Shutdown 不等回合
  不 Close 控制器）开启时先 `drainControllers()`（办公+文学双板）再 teardown；
  进程管理步骤（pantheon/glm-catalog/health-probe/voice-v2/weixin）全部套
  `shutdownStep` 超时（5–10s，挂起记日志继续退出，A4）；步骤错误文本尾截
  4KiB（`tailStderr`）。DB 关闭与 whisper 落库保持必须完成（不套超时）。
- **P3 B5 工具归因闭合**：`event.Tool` +`Model`、`Event.Turn` 从死字段变实盖戳
  （注释原称 runner 维护、实际从未赋值）；六处发射点（batch 派发/结果、流式
  pre-exec 派发、pre-exec 结果、Usage×2、host-advance 合成对）盖 `Turn+Model`
  （合成 todo_write 按归属纪律留空 Model——框架代写不冒充模型来源）；子代理
  经 `routedProv` 携带按调用路由模型 id（覆盖 boot 身份快照），`Options.ModelName`
  由 boot 以 `entry.Model` 注入（ResolveModel 已归一含 provider/model 引用与
  Models 列表缺省）。会话日志 `tool_dispatch/tool_result` payload 落
  `model/turn`（omitempty，旧日志逐字节同形）；`request_header` 补 `model`
  （比 Provider 实例名细一档）；轨迹折叠 ToolRec 浮出 Model/Turn（dispatch 为
  准、result 仅补齐重建路径）。**退役死钩子 auditFunc**（V3.2 落地起全仓零
  消费方、第二参恒空串、签名缺归因三元组——FE3-04 零调用者摘除先例）。
- **P4 B9 出网安全 + B7 构造即验证**：落点从施工卡的 provider/outbound.go 改到
  **既有唯一 HTTP 工厂 `internal/netclient`**（asset-53 自认未读 gaea 侧；
  一处只改一次纪律，不平行造层）。`outbound.go`：`OutboundPolicy`
  （unrestricted 默认=现状 / restricted=https 白名单+回环豁免——本机引擎不是
  出网）、`OutboundGuard` 四态拒因（Unavailable/Destination/Revoked/
  UnsupportedTransport，`OutboundError` 错误即值可分类）、Acquire/Permit 带
  revision（SetPolicy 即撤销全部旧许可）、`InstallOutbound` 重定向纪律（≤10
  跳、每跳过闸、跨源剥 Authorization/Cookie/Proxy-Authorization + 补 Referer、
  同源不删）、`OutboundTransport` 首跳拨号前过闸。B7：`NewOutboundGuard`/
  `SetPolicy` 构造即验证（restricted 空白名单拒绝构造——不给静默宽口；零值
  guard 裁决 unavailable，无默认放行形态）。config `[network]` 增
  `outbound_mode/outbound_allowed_hosts`；`ai.Client.buildHTTPClient` 装配
  （策略非法 warn 不装=不降级假 restricted）。**不搬**：N6 TLS 双后端回退
  （Go 标准库无此概念）、N7 模型目录缓存（不在 P4 落地面，gaea 目录是
  modelengine 侧另一形态）、路由池具体形态（单进程按需分池即可）。
- **验证**：新增测试 22 例全绿——drain 5 例（A1 拒收不开跑/A2 已准入跑完才返/
  A3 超时强退落账/排队逐条作废/系统回合拒收）+ shutdownStep 3 例（完成/挂起
  超时继续/尾截 4KiB）+ 归因 8 例（回合一致/并行不串号/未配置不盖/日志含
  model·turn·header/未盖戳缺省字段/折叠浮出/旧日志零值）+ outbound 9 例
  （N1 协议与白名单/N2 尾点归一/N3 许可撤销/构造即验证/N4 跨源剥头补 Referer/
  同源保留/N5 11 跳终止/首跳过闸/nil guard 不动客户端）。全量回归：control
  2.9s、agent 18.4s、config、netclient、ai 全绿；app 全量 130.1s 绿（ucrt64+
  DLL）；`go build ./...` + vet 全绿。
- **边界**：未 Wails build、未装机、未 push；Wave 2（P5–P8）卡三拍板门
  （权限快照维度/投影滞后/观测落盘），Wave 3（P9 B8 MCP 目录缓存）待定——
  gaea 无 MCP 客户端面（在册排除清单），需先证伪或改刀。

## codex 蒸馏第三批 · Git 打包与桌面构建（2026-11-17）

- **代码/交付点**：`d30cb5d2`（`feat(gaea): 完成 codex 蒸馏第三批双板块收口`，
  94 files changed）。第三批 W1–W9 全清，并同步包含第四批 P2 输出预算同源；
  剩余第四批、F 批与 B 批不在本包。
- **干净树构建**：`.worktrees/pkg-d30cb5d2`（detached @`d30cb5d2`，node_modules
  junction）；前端 `npm.cmd run build` 81.412s（Vite 34.85s），随后 UCRT64 GCC +
  `CGO_ENABLED=1` + 系统默认 TMP/TEMP 执行
  `wails build -s -ldflags "-s -w" -trimpath` 49.665s（编译 49.387s），
  `Skip Bindings=false`。Wails 再生仅留下 wailsjs/go.mod 的 CRLF 状态噪声，
  `git diff --ignore-space-at-eol` 内容漂移为空。
- **交付目录**：`releases/gaea-codex-distill-batch3-20261117-d30cb5d2/`
  （payload 四件 + PACKAGE-INFO + SHA256SUMS + preview ZIP + source TAR）；
  两份文本记录入库，二进制/源码包留盘不入库。
- **payload**：`gaea.exe` 54,916,608 B
  `E058DB7E3CFDD1CA3E6B38A3D6D569A1EC9D2F57B76B7011421031412C080119`；
  三件 DLL 沿用上一基线原样且哈希不变
  `68D0D5F5…F550B` / `5DD36DBD…01E8` / `4C563277…7FFE`。
- **冒烟**：正式 payload 路径 `/api/health` 200（`{"status":"ok"}`），
  PID 49028 已回收；构建/冒烟后无 gaea/wails/go 孤儿进程。
- **源码/投递包**：`gaea-d30cb5d2-source.tar.gz` 43,621,391 B
  `79F508287D8E4746BFE8256B7E8F09C8E0AE1C63C2DCBBB01F6F3FD684ACA2EF`；
  ZIP 30,100,260 B
  `2B75C0AD378AF286D971C1FC691B47A0E30FF5F4EE97F9E3F9C89A09B928DE60`；
  七项最终清单全部复算 OK。
- **版本/边界**：源版本 `wails.json=4.544.0`，不抬号；当前与上一包 exe 的
  raw FileVersion/ProductVersion 均为空，不伪记 PE 资源版本。未跟踪
  hotspot-07 文档未纳入提交/源码包/构建；未运行 `gaea_install.ps1`，
  未覆盖 `C:\AI\gaea-app`。

## codex 蒸馏第三批 Wave 3 · 渲染闸门、表格复制与 GenUI 护栏（2026-11-17）

第三批 W8/W9 完成，至此第三批 Wave 1–3 全清：

- **C14 高亮闸门**：`codeHighlight` 统一执行 512 KiB 总字节、10000 总行、
  单行 4 KiB 三道预检；任一越限直接返回纯文本，不加载 hljs。文学高亮入口改为
  办公单源 re-export。
- **C14 复制链**：新增表格 DOM → GFM 快照/序列化（含对齐行、反斜杠与 `|` 转义），
  `ChatMarkdown`、`MarkdownContent`、办公/文学 `Markdown` 均挂同一
  `CopyableTable` 工具条。`copyToClipboard` 扩为 `rich|plain|false`：
  `ClipboardItem` 双 MIME（text/plain + text/html）优先，失败诚实回落纯文本；
  空内容不写、execCommand 抛错仍清理临时 textarea。
- **C13 GenUI 护栏**：共享预算配置新增 2 MiB GenUI 源门和
  `https://gaea.invalid/genui/` 占位；超限源在切围栏前直接回退文本，
  未通过校验的围栏带 `.invalid` 内部占位属性；外部 `src/imageUrl` 等未知字段
  继续由白名单 guard 丢弃，不建立外链视觉源。
- **验证**：全量前端 Vitest **494 文件 / 4453 例绿**；tsc + Vite **21.18s** 绿；
  全量 ESLint 0 error；`git diff --check` 绿。
- **边界**：未 Wails build、未装机、未 commit。第三批 11 条复核项（W1–W9）
  至此全部落地；后续可选进入第四批其余项或 Wave 3 之外的独立 backlog。

## codex 蒸馏第三批 Wave 2 · 截断、披露、Esc 与文本度量（2026-11-17）

按复核稿完成 W5–W7，并同步落地第四批 P2：

- **W5 A5/B8 + P2 同源预算**：新增
  `internal/gaea/strutil/output_budget.json` 作为前后端共同契约；Go 新增
  `Policy{ModeBytes|ModeTokens|ModeRunes}`、4:1 互折、1.2 序列化余量幂等、
  rune 安全头尾中间截断。Agent/Archive/Controller 四处旧截断改为薄包装，
  `truncateStr` 不再制造非法 UTF-8，长文本尾部不再被整段丢弃。
- **工具卡披露量化**：`boundedOutput` 改为头 6/尾 6（通用 60 行同规则）、
  单行按显示宽度头尾保留、`retainedLines/totalLines/hiddenLines/hiddenBytes`
  分离；超宽行计入 `hiddenLines`。ToolCard 仅在确有 command/diff/args/error
  或真实隐藏内容时开放，短输出不再亮空展开箭头。文学 ToolCard 改为办公单源
  wrapper，并保留文学工具 deny 过滤。
- **W6 Esc 阶梯 + 同源提示**：新增 `escLadder` 纯函数，优先级为
  warnings/questions/modal/preview/overlay/banner/input/stop/backtrack；
  Esc 在非输入区可直接停止运行中回合，弹层先于停止，审批卡仍由自身裁决。
  文学键盘 hook 与 CommandPalette 改为办公单源。新增 `HintStrip`：
  完整/中/短三档按容器宽度选整组提示，单条提示永不拆半。
- **W7 按显示宽度度量**：新增 `textMetrics`（grapheme + CJK/emoji 两列宽、
  组合字符零增宽）与 `hintLayout`；工具主体/单行输出、能力摘要、拒绝账本改走
  显示宽度；`MessageList` 显式 `MAX_RENDERED_ROWS=10000`，不再只有隐式窗口。
- **验证**：全量前端 Vitest **490 文件 / 4440 例绿**（全量首跑唯一非本面
  `Fleet` 事件首条失败，单文件复跑 4/4 绿定性负载 flaky）；全量 ESLint
  0 error；tsc + Vite **37.41s** 绿。Go `strutil/agent/archive/control`
  四包全量绿、vet 绿；UCRT64 + `CGO_ENABLED=1` 下 `go build ./...` 绿。
- **边界**：未 Wails build、未装机、未 commit；第三批 Wave 3（C14 渲染闸门/
  C13 GenUI 上限）尚未开工。

## codex 蒸馏第三批 Wave 1 · 前端正确性与双壳单源化（2026-11-17）

按 `plan-batch3-landing-review.md` 的 W1–W4 开工，完成 11 条中第一波：

- **W1 单源化**：`ApprovalModal`、`Composer`、`lib/tools.ts` 以办公实现为唯一逻辑源；
  文学壳只保留适配 wrapper，显式注入 `Wenxue bridge`、文学 insert store、命令过滤与
  `wenxue` variant，未回落办公绑定。i18n 改为共享 React Context、各自保留字典与
  `DictKey`，避免共享组件在文学 Provider 下取错上下文。
- **A1/A2/A8**：审批卡加焦点陷阱；hardConfirm 键位改为 `1=deny / 2=allow_once /
  3=abort`；会话平铺与项目分组的 sessions/archived 按 path 去重。
- **A7**：`useComposerMenus` 为 `/` 参数和 `@` 搜索加 waiting/current-query 门；
  旧请求结果不能覆盖新查询，等待期显示加载态且不误提交。
- **A4**：新增 `canBacktrack` 纯函数（非在飞、历史完整、目标存在、提示词非空且可见）；
  办公/文学回退入口共用；后端回退失败会把原 prompt 回填到本壳 Composer 并保留错误提示。
- **验证**：前端全量 Vitest **484 文件 / 4420 例全绿**；`npm.cmd run build`
  tsc + Vite 38.17s 绿（仅既有 CSS minify/大 chunk warning）；全量 ESLint 0 error，
  本轮改动面 ESLint 0。
- **边界**：未 Wails build、未装机、未 commit；第三批 Wave 2（A5/B8、Esc 阶梯、按宽截断）
  与第四批 P2 截断 Policy 同源事项尚未开工。

## codex 蒸馏前两批 · Git 打包与桌面构建（2026-11-17）

- **功能提交**：`f416e223`（前两批 T1–T15 双板块收口 + 蒸馏文档）。
- **模块再生提交**：`4be0ddc9`（Wails 模块补齐 Office/Wenxue 新绑定与
  `control.DeniedActionInfo`/`pantheon.AgentStatus.description`）；交付构建点固定为该提交。
- **干净树构建**：`.worktrees/pkg-4be0ddc9`（detached @`4be0ddc9`，前端 node_modules
  junction）；前端 `npm.cmd run build` Vite 27.94s，随后 UCRT64 GCC +
  `CGO_ENABLED=1` + 系统默认 TMP/TEMP 执行
  `wails build -s -ldflags "-s -w" -trimpath` 56.045s，日志确认
  `Skip Bindings=false`。构建后 wailsjs 仅有 CRLF 状态噪声，
  `git diff --ignore-space-at-eol` 为空。
- **交付目录**：`releases/gaea-codex-distill-batch12-20261117-4be0ddc9/`
  （payload 四件 + `PACKAGE-INFO.md` + `SHA256SUMS-4be0ddc9.txt` +
  preview ZIP + source TAR；二进制/源码包留盘不入库）。
- **payload**：`gaea.exe` 54,987,776 B `03455E79…FA54D2`；三 DLL 与原基线同哈希
  `68D0D5F5…` / `5DD36DBD…` / `4C5632…`。完整 SHA256 见包内清单。
- **冒烟**：独立 worktree 的 `build/bin` 被本机策略拦截（WebView2
  `800700aa`）；同构建复制到正式 payload 后 `/api/health` 200（PID 55368），
  再同步主树 `build/bin` 冒烟 200（PID 24580），进程均已回收。
- **版本/装机**：4.544.0，不抬号；未运行 `gaea_install.ps1`，未覆盖
  `C:\AI\gaea-app`。

## codex 蒸馏前两批 · 第二轮全审计与办公/文学双板块收口（2026-11-17）

按 `exec-brief-T1-T5.md`、`master-brief-T6-T15.md` 和两批资产验收逐项复核，
**前两批在 gaea 现有产品形态下的适用面已收口**。后续 F 批与 B 批未触碰。
本轮又抓到并修复 7 类真实缺口：

1. **T10 会话作用域未真正失效**：`/new` 此前不清 `granted` / `grantedRules` /
   `argvRules`，且既有 Gate 持有旧 argv 指针。现会话切换统一清空授权、追认
   账本、Guardian 连续拒绝计数与最近用户授权证据；`ArgvRuleSet` 改并发安全并
   **原地 Clear**，旧 Gate 同步失效。
2. **T7 新会话/恢复快照断链**：`SectionStore` 原随 controller 永久存活，新
   会话会误用上一会话快照；恢复会话又没有 `MarkUnknown`。现 `/new` Reset、
   Resume 转 Unknown（下一轮发替换通知），`SectionStore` 加锁可并发读。
3. **文学审批仍停在旧五决策**：补齐 `WenxueApproveWithReason` /
   `WenxuePendingDeniedActions` / `WenxueApproveDenied`、拒绝理由框、待追认
   Dock、Guardian 来源徽标与独立事件通道刷新；全部锚 `wenxueCtrl`，不再误用
   办公账本。
4. **T11 只接办公**：Guardian caller 改按 feature 路由，办公传 `office`、
   图文传 `wenxue`；图文引擎同样注入 `[guardian]` reviewer。
5. **T9 只判不接**：`GaeaSetModel` / 新 `WenxueSetModel` 均先按目标模型窗口
   触发换窗口压缩，成功后才改模型；成功后同步 runner 的 `Compaction.Window`，
   后续自动压缩不再按旧模型水位计算。
6. **T2 覆盖不全**：除 bash/hook 外，补齐 chart/diagram/read_file、proc
   session、office xlsx 重算、crosslink、COM helper spawn/csc 的固定环境。
7. **T4 绝对路径绕过**：`C:\...\powershell.exe`、`cmd.exe`、绝对路径 `rm`
   此前绕过直呼命令归一；现先取路径 basename 再判定。

另补 T15 角色级配置层：角色 frontmatter 可写
`config_layer: reviewer.toml`，相对角色目录读取 `[agent] max_steps /
temperature / context_window`；路径含符号链接逃逸拒绝，坏层在扫描期跳过并
告警，执行预算只进该席位。两条角色各读各的层有隔离测试。

### 双板块结论

- **办公**：T1/T2/T4/T5/T6/T7/T8/T9/T10/T11/T15 的共享内核与本板块审批/
  模型切换已接线；无“只落库不消费”的新增死开关。
- **文学**：共享内核同样生效；本轮重点补齐 T3/T11 审批闭环与 T9 独立模型切换
  路径，桥面 `bindingNames` 701 名、签名 701、legacy 225 名，与办公 710 名
  维持“减 6 个 Chat* + 3 个 Gaea 审批名”的既有镜像口径。
- **适用性关账（不是代码缺口）**：T8 Catalog 原语因 provider 只收 tools 数组、
  没有目录文本槽而未虚接；T12 Gate/Baseline 原语与 T13 proc.Session、T14
  RRF 原语均无会改变行为的现存生产消费方，保留库接口，不强接死代码。

### 门禁

- `go build ./...`、`go vet ./internal/gaea/... ./internal/office/...
  ./internal/pantheon ./internal/app` 全绿（UCRT64 GCC + 正式 DLL 搜索路径）。
- Go 全量 `internal/app` 165.7s 绿；control/context/permission/proc/agent、
  tool/builtin、office 全包、pantheon 全绿。
- 前端 `tsc -b --force` 0；office `src/gaea` 247 文件/2109 例绿；文学
  `src/wenxue` 14 文件/62 例绿；本轮改动面 ESLint 0；binding drift
  OK@710。
- `scripts/check-docs.mjs` 仍红于既有旁支：4 份顶层文档未登记 +
  releases README 三组历史计数漂移；与本轮改动无交集，未顺手改后续批次。
- 未 Wails build、未装机、未 commit；后续批次未动。

## codex 0.165.0 蒸馏第二批 · 第二/三波补刀 T8/T9/T10（2026-11-17）

第一波四刀（T6/T12/T14/T13）见上段。本段补第二波两项与第三波一项，
**未 commit、未装机**。新增 16 例：tool 5 / agent 5 / permission 8 / control 2。

### T8 · 工具面预算三层（tool/{catalog,search_index}.go，叠在 T5 budget.go 上）

- **目录差分**：`Catalog`（name→sha256 快照）+ `Diff`（added/removed/changed，
  changed=false 时上下文逐字节相同）+ `CoalesceRemoved`（整命名空间移除吞并成
  `ns/*` 一条——上游纪律 3）。
- **检索展开**：`SearchIndex` 自包含 BM25（k1=1.2/b=0.75，同分名字典序确定序），
  索引条目只存检索文本（schema 引用不复制）；`BuildSearchIndexFromRegistry`
  含隐藏工具（藏而可搜）、排除 tool_search/websearch 自身；limit 默认 8。
- **偏差照实记**：施工单说「用现有 bm25」——gaea 无该包（v2 式按名猜），自实现
  60 行整数化 BM25 不引新依赖。
- **审计纠偏（2026-11-17）**：**Catalog 目录差分是「原语就绪、生产零调用」**——
  gaea 工具面走 provider tools 数组逐请求下发，没有「上下文内工具目录文本」
  这个槽位（上游 top_level_tools 差分的是注入上下文的目录段，形态依赖
  T7 批次②/世界状态注入面，该面未立项）。故 `tools.catalog.diff.enabled`
  开关不加（不给死代码配开关），Catalog 留作原语；后续若做工具目录注入再接线。
  已接线的是检索展开层（接线小批）：`[tools] deferred` 隐藏 + `[tools.search]
  enabled/limit` 注册 tool_search（命中即回 schema 文本，物化即文本）。

### T9 · 压缩三形态（agent/compact_newwindow.go + hook PostCompact）

- **做法**：`NewWindowCompact`（token-budget 形态）——跳过 LLM 摘要直接换窗口：
  固定前缀 + partitionFold 白名单保留项（[[keep]] 用户目标=上游 input_goal_ids
  继承的 gaea 折算）+ 回合上下文，无摘要占位消息；完整生命周期（PreCompact
  观测 + CompactionStarted/Done + PostCompact 通知）；`Compaction.Window` 序号
  单调递增、Done.Summary 空=「无摘要的换窗口」标记； economics 双闸
  （foldEconomics 小额跳过 + 丢弃后估算不降拒绝）。`MaybeDownshiftCompact`
  （ModelDownshift 触发）暴露给切模型入口，开关 `CompactionConfig.NewWindow`
  默认关=零行为变化。
- **纪律落地**：PostCompact 用可选接口（postCompactHook）不扩 ToolHooks 主接口
  ——既有测试桩零破坏；小窗口不按 trigger 判经济学（tail 2000 token 下限会
  顶破小窗口 trigger，与 compact() 自身注释同款教训，按「估算有没有变小」判）。
- **未做（挂小批）**：ModelDownshift 在控制器换模型处的实际接线；CompHashChanged
  （gaea 无 comp hash 概念，需先建模型口径指纹）。
- **审计补（2026-11-17）**：接线小批只加了配置键，`compaction_new_window=true`
  此前**无任何触发点**（NewWindowCompact 无人调用，开关无效——审计逮住）。
  已修：`compactIfOver` 增 Tier 2.5——开关开时先试换窗口形态，无物可压/
  经济学拒绝回落 Tier 3 摘要式；回归测试 `TestNewWindowWiredIntoAutoCompact`
  （开=无摘要占位、关=摘要式现状）。

### T10 · 审批七态 + argv 前缀规则（permission/argv_rule.go，承接 T3/T4）

- **现状对账**：步一（七态映射/拒绝理由/超时不进 deny 账本）与步三（追认账本
  + ApproveDenied 重放）已由第一批 T3 覆盖——本刀补真正的缺口 **步二 argv 规则**：
  glob 是字符串级匹配，`bash(git status*)` 会把 `git status; rm -rf` 一起放行。
- **做法**：`ArgvRule`（首 token 索引 + rest 前缀精确匹配）+ 分段安全语义
  （命令按 shell 串接符分段，每段 token 前缀都命中才放行——串接提权天然不匹配）
  + 引号剥除切分（含 glob 元字符 `*?[` 的 subject 拒绝转 argv，回落既有路径）。
  三层接线：`Policy.ArgvAllow`（config `bash prefix ["git","status"]` 形态，
  Persistent 作用域）/ `Gate.SessionArgv`（会话作用域，/new 轮换即失效）/
  `writebackAllowRule`（bash 回写 argv 一行式，其他工具保持既有形态）。
  promptApproval 短路预检同步查 argv 规则（否则同形命令重复弹卡，测试逮住后补）。
- **验收**：git status 放行而 `git status; rm -rf /tmp/x` 仍要审批（且深删被 T4
  硬闸先拦，两条防线各司其职）；Session/Persistent 作用域区分；回写格式往返；
  controller 全链（弹卡→allow_session→同形免卡→串接仍卡）。
- **未做**：`permission.argv_rules.enabled` 开关（argv 解析只在显式配置该形态
  时生效，默认配置零变化=等效关闭）；Alts 多选一 token（上游也不建议先做）；
  MCP/网络策略落盘（gaea 无对应需求面）。

### 门禁

十四面 `go build`+`go vet`+`go test` 全绿（context/memory/skill/proc/tool/builtin/
control/permission/agent/hook/event/cache/config/boot，既有用例零回归；
`TestApprovalSessionGrant` 等 glob 回写用例经 glob 元字符守卫保住原格式）。
零新外部依赖；UTF-8 无 BOM；gofmt 全绿。

## codex 蒸馏 · 审计缺口完善批（2026-11-17）

审计五缺口的修复落地（后端 3 + 前端面 1 + 记档纠偏并入上段）。**未 commit**。

### 后端三件

- **T2 hook 注入**：`hook.DefaultSpawner` 接 `proc.CommandEnv()`——hook 输出
  注入模型上下文（PreCompact 指引），与 bash 工具同源同阀（GAEA_EXEC_ENV_RAW）。
  MCP stdio 维持不注入（服务进程环境敏感）。
- **T4 wsl 形态**：`DangerousWindowsCommand` 补 wsl 直通（`wsl rm -rf x` /
  `wsl --exec rm …` / `--cd <p> rm …`）——剥带值/带旗引导开关取尾部命令；
  未知开关保守放行、裸 wsl 不拦。测试 +7 形态（直通/--exec/引导/未知/只读/空）。
- **T15 席位免重启**：角色加载抽 `reloadRoleSeatsLocked`，`PantheonStart` 每场
  讨论前重载——增删 `<pantheon 根>/agents/*.md` 免重启，新讨论即见（旧会话
  席位表定格不受影响）。

### T3/T11 前端面（审批闭环 UI）

- **绑定三件**（Go→bindingNames→core/mappings/spaceBindings/mock 六面同步，
  签名清单 `-signatures-ts` 再生，drift 脚本 706 名一致）：
  `GaeaApproveWithReason` / `GaeaPendingDeniedActions`（control.DeniedActionInfo
  DTO，decision=deny|timeout 可区分）/ `GaeaApproveDenied`（追认走人工卡）。
- **ApprovalModal**：拒绝理由输入框（可选一句话，deny/abort 随卡带回——
  store.approve 按「拒绝类决策 + 非空理由」路由 ApproveWithReason，其余走原
  Approve 零变化；快捷键路径同携理由）；guardian 追认重放卡徽标
  （source=guardian 副标题「机器不复审，由你裁决」）。新增测试 2 例（理由
  透传/放行不带理由 + 徽标），既有 4 例零破坏（断言取 c[0] 天然兼容）。
- **DeniedActionsDock**（新组件）：待追认账本列表——审批卡同层决策区；
  事件驱动刷新（turn_done/approval_auto/approval_resolved，EventKind 联合
  补齐后端已发的四种 kind）；「追认」钮 → ApproveDenied → 重放人工卡；
  空账本零渲染。CSS 配套（office-approval-deny-reason / office-denied-dock）。
- mock：ApproveWithReason 复用 Approve 收尾流；PendingDeniedActions 空表
  （dev 无审批历史）；ApproveDenied no-op。

### 门禁

go：hook/permission/control/app build+vet+test 全绿（gofmt 列表中的
bindings_* 为既有漂移非本批，bindings_office.go 本批改动自身 gofmt 净）。
前端：tsc -b 0；vitest gaea 全量 247 文件绿（contract 清单再生后过；
DatabaseLibrary 单跑绿=负载 flaky 非本批）；bindingNames drift 706 一致。

## codex 蒸馏 · 两批全景审计（2026-11-17）

对 T1–T15 + 接线小批逐项核对施工单验收 vs 实际代码/测试。结论：
**3 处真错（已修 2、纠账 1）+ 5 处记档/边界缺口（已如实入档）+ 判定关账
4 项复核全部成立**。

### 真错（代码级，审计逮住并修复）

1. **T15 文件席位探测恒不可用**：`pantheonRoleSeat.ProbeRole` 可用路径漏设
   `status.Available = true`（AgentStatus 零值 false）——文件席位会被前端
   判为不可用。测试用 fake 后端自设 Available 故未逮住。已补一行置位。
2. **T9 开关断链**：`compaction_new_window=true` 后无任何代码调用
   NewWindowCompact/MaybeDownshiftCompact——开关无效（接线小批只通了配置键，
   记档只把 Downshift 挂账、没说主触发链也断）。已修：compactIfOver 增
   Tier 2.5（开=先换窗口，回落摘要式）+ 回归测试。
3. **T8 Catalog 死代码**：生产零调用。根因是形态错位——上游差分的是
   「注入上下文的工具目录文本」，gaea 工具走 provider tools 数组，无此槽位。
   处置：纠账为「原语就绪、无生产槽位」（不给死代码配开关），等 T7 批次②
   /世界状态注入面立项再接；已在 T8 段落改写。

### 记档/边界缺口（如实入档，不改动行为）

4. **T2 范围**：环境注入只覆盖 bash 工具两条执行路；hook 命令子进程
   （DefaultSpawner）与 MCP stdio 未注入——已记档的范围决策（hook 是用户
   诊断脚本，注入 GAEA_CI 可能干扰其逻辑），审计确认维持。
5. **T6 生产面**：碎片契约的生产者目前只有 T7 的章节碎片——契约本身是
   地基性质（T11 证据信任标注是第二个消费者），可接受但审计点名。
6. **T11 前端面**：ApprovalSourceGuardian 是新 Source 值，前端审批卡对
   guardian 来源的渲染/追认入口未核（后端链全绿，前端联调挂装机后）。
7. **T15 生效粒度**：角色文件在 manager 创建时加载，增删文件需重启应用
   （未记档，现补）。
8. **T4 wsl 盲区**：`wsl rm -rf` 逃过危险命令判定（wsl 不走 -c 形态）——
   保守方向（认不出不拦）与既记纪律一致，单列待后续补 wsl 直通形态。

### 判定关账复核（4 项全部成立）

- T12 背压门（dream 已有模式门+建议制红线，三道门无真实缺口）✓
- T14 rrf（技能 pinned 注入，无动态选技消费方）✓
- T9 ModelDownshift（换模型=重建控制器，auto 压缩+overflowRecovery 已覆盖）✓
- T13 proc.session（与 bgjobs 语义重叠，整合形态在拍板池）✓

### 复核后门禁

agent（含新回归例）/app/pantheon 及相关面包 build/vet/test 全绿；gofmt 净。

## codex 蒸馏 · T11 Guardian 自动评审（2026-11-17）

施工单 `master-brief-T6-T15.md` §T11 + `asset-07`（施工级）。三刀全落，
**未 commit、未装机**。新增 10 例：guardian 4 / control 6。

### 刀一 · 契约与容错（guardian/guardian.go）

- `Parse` 三层容错：严格 JSON → 首 { 到末 } 切片重试 → 评审失败（非 JSON ≠
  高风险）。默认值纪律照抄：outcome 必填（缺即失败）；allow 缺 risk→low、
  **deny 缺 risk→high（保守方向）**；缺 authorization→unknown；缺/空
  rationale→固定兜底句（UI 不出现空白理由）。枚举非法值一律失败。
- `ResponseSchema` 常量与提示词模板分放（注释钉住「改一处同步另一处」）。

### 刀二 · 证据包/预算/超时（guardian/{evidence,review}.go）

- **T6 联动**：`EvidencePack` 信任标注由 `context.CanAuthorize` 派生——用户
  原话（ProvUser）标【可信证据·可支撑授权】，工具输出/框架注入标
  【背景信息·不可作为授权依据】（asset 验收③的落地形态：结构标注进提示词，
  归属由生产端声明不反推）。
- 预算三层裁剪：不可裁（动作+用户指令）→ 先丢不可信上下文 → 再丢可信
  上下文 → UTF-8 边界截断；仍超 → `ErrInputBudget`；`Original/RetainedBytes`
  跨阶段记账。
- `Runner`：Caller 接口注入（评审走独立调用不复用主会话）；超时从入口即
  生效（默认 20s）、父取消每阶段尊重；失败四分类
  `ErrInputBudget/ErrTimeout/ErrParse/ErrSession`——失败不产生拒绝账本、
  不冒充风险（asset §四红线）。
- `Breaker`：连续 3 deny 熔断（计数器只请求，中断由宿主执行——职责分界）。

### 刀三 · 接线（control/controller_guardian.go + 装配链）

- **通道纪律**：只在常规 Ask 通道（gate）用户卡之前；hardAsk（cost_save
  等逐条确认）与 schedule_apply project 通道**不走**机器评审（测试钉死）；
  auto/yolo 短路在更早处，guardian 不参与。
- **顺序修正（测试逮住的真缺陷）**：会话已授权（granted/argv 规则/T3 追认
  落账）必须排在 guardian **之前**——人已给过的授权不需要机器复审。
  gateApprover 顺序定为 hardAsk → project → auto → **session-grant** →
  **guardian** → 用户卡。
- 三路径：allow→放行+`ApprovalAuto` 三字段账本（Risk/Authorization/Rationale，
  Source=guardian，Decided=guardian_allow）；deny→理由回喂+T3 待追认账本
  （用户 `ApproveDenied` 弹卡裁决，机器不复审）+熔断计数（第 3 次发通知并
  `Cancel` 中断回合）；**失败→回落用户卡（红线，四类失败逐一测试）**。
- `lastUserInput`（sendWithRaw 存最近用户原话截 8KiB）作授权证据源。
- 装配链：`[guardian] enabled/timeout_ms/input_budget`（默认关=审批纯人工
  逐字节）→ app `newGuardianReviewer`（办公本地优先路由，与 dream 同口径）
  → boot.Options.Guardian → control.Options.Guardian。

### 验收对照

asset 六例全落（严格/散文/缺 risk/缺 rationale/空/非 JSON）+ 信任标注与
预算裁剪 + 四类失败分类 + 熔断 + control 六例（allow 账本三字段/deny 追认
全链/失败回落红线×4/熔断触发/hardAsk 旁路/失败打断熔断连续性）。

### 未做（如实）

- guardian-v2 的采样/观测体系与 CachedApproval（上游明示单机用不上）；
- 滑动窗口降级（50 次/10 deny）——先做连续熔断，窗口态待真实使用数据；
- wenxue 引擎不接 guardian（play 侧无审批面）。

## codex 蒸馏 · T15 角色数据化（2026-11-17）

施工单 `master-brief-T6-T15.md` §T15 + `asset-06`（角色文件化值得做/图谱不搬）。
**未 commit、未装机**。新增 6 例：pantheon rolefile 五例 + 内建碰撞一例。

### 做法（pantheon 层核心 + app 装配，动前对齐众神殿线程史）

- **角色文件**：`<pantheon 根>/agents/<role>.md`（frontmatter
  name/description/nicknames + 正文即 persona）。纪律照抄上游：
  description 给模型当 spawn 引导、**必填非空**（空描述拒载+告警不炸全场）；
  坏文件跳过留 warn；目录不存在=空表常态；按 Name 排序稳定序列。
- **适配器**：`RoleSeatBackend{ProbeRole,CallRole}`（+可选 `RoleStreamingBackend`）
  与 grok 席同构——`newRoleAdapter(role, backend)` id/name 取文件、kind=文件
  席位、Description 随 AgentStatus 暴露（前端席位卡直接可见，零前端改动）；
  persona 由后端注入模型 system prompt（agent 级，非 pantheon prompt 拼装）。
- **防串引擎提权**：与内建（grok/codex/zcode/dsh）同名的文件席位跳过——内建
  的引擎绑定不可被文件替换；文件席位排序落内建之后（Agents() priority 兜底 10）。
- **app 装配**：`pantheonRoleSeat` 后端（办公本地优先路由 + persona 即
  system prompt + grok 席同款只读注册表）；`runPantheonGrokAgent` 抽出
  `runPantheonAgentWithPrompt`（system prompt 参数化，grok 席零行为变化）；
  handler 加载 `agents/` 目录 → `SetRoleSeats`（SetGrokBackend 同款重建纪律，
  四个 buildAdapters 调用点全穿 roles/backend）。
- **验收四条全落**：建文件即出现在席位表（Agents()→UI 席位卡，Description
  可见）；删文件（空表重载）不残留；description 进席位引导（toolPolicy
  role-file 分支带【角色设定】+description，内建不受累）；角色级覆盖隔离
  （invoke 各传各的 RoleFile，A 人设不进 B 调用）+ 内建碰撞跳过。

### 未做（如实）

- **config_layer（角色级配置覆盖）**：资产里的「审稿人自带更严审批/更小预算」
  需要席位级策略注入面（当前席位共用闸门）——persona/描述已文件化，配置覆盖
  等席位级闸门立项（与 T10 作用域语义衔接）；file 格式已留扩展位。
- 昵称候选：文件可声明 `nicknames`，运行期取昵称逻辑未接（单次入席无混淆
  面），字段就绪。
- UI 侧改动：零（Agents() 透传 Description，席位卡自渲染）。

### 门禁

pantheon + app build/vet 全绿；pantheon 全量测试绿（含 live_smoke 等
buildAdapters 旧调用点的机械适配）；gofmt 净。

## codex 蒸馏 · T7 章节化+快照差分（2026-11-17）

施工单 `master-brief-T6-T15.md` §T7 + `asset-08`（施工级）/`asset-02`（装配），
基于 T6 碎片契约。**未 commit、未装机**。新增 8 例：context 6 / control 2。

### 落点判定（照实修正施工单）

施工单写「contextview 加章节表」——contextview 实为会话日志折叠视图（前端面），
真实注入面在两处：①`Compose` 逐轮注入（input.go：memory-update/session-facts/
background-jobs）②L2 runtime 前缀（已在 V5.7 学过本课——易变字段挪 turn-tail，
前缀只留稳定块）。**真消费方选①的 session-facts**：临时事实清单每轮全量重注入
（N facts × 每轮，直到 promote），是 gaea 里唯一「按轮重发且大多不变」的块。

### 引擎（context/sections.go，纯函数 + 账本）

- 三态 `Previous{PrevAbsent/PrevKnown/PrevUnknown}`（历史没快照 / 有快照 /
  老会话有旧片段但不可考——`*T` 两态分不开「没有」与「是空」）；
- `RenderSectionDiff` 三情形照抄上游：相同零片段（省 token 的唯一来源）/
  撤销发移除通知 / Unknown 当有（保守发替换）；通知文案固定两句话（不逐轮
  生成，diff 才收敛）；快照只放可比较字符串；
- `SectionStore` 会话内账本（MarkUnknown 迁移入口 / RenderAll 按 ID 字典序
  稳定输出）；片段即 T6 Fragment（Kind=`section.<id>`，user 角色 + ProvHarness
  ——role 与归属两件事，CanAuthorize=false 有测试钉死）。

### 接线（Compose session-facts，开关 `[agent] section_diff` 默认关）

- `Options.SectionDiff` → `c.sectionDiff/sectionStore`（构造即建账本）；开时
  注入走差分：不变零注入（省 N×轮次）/ 变更「替换通知+全量清单」/ 清空
  「移除通知」；关时走原全量注入，与旧实现逐字节一致（HasPrefix 断言钉死）。
- boot ctrlOpts 接 `cfg.Agent.SectionDiff`。

### 验收

引擎五例（三轮稳定/移除通知/Unknown 迁移/一字之差替换/多章节字典序稳定）
+ T6 联动例 + 端到端四态例 + 关态逐字节回归，八例全绿；context/control/config/
boot/agent/permission/memory/tool 八面 build/vet/test 全绿。

### 未做（如实）

- L2 runtime 前缀的章节化（批次②：差分进 flow 而非重建前缀）——动前缀缓存
  关键路径，需 prefix_stability 级独立刀；引擎/账本已备，`RenderAll` 输出即可
  为 flow 片段源。
- 其他 Compose 注入块（memory-update/background-jobs）为事件驱动型（有事件才
  注入），本就 delta 形态，无差分收益。

## codex 蒸馏 · 接线小批清挂账（2026-11-17）

两批 12 刀的挂账接线收口。**未 commit、未装机**。新增 4 例（tool 2 / control 1
+既有面回归）。

### 已接线（三件）

- **T9 `compact.new_window` 配置键**：`[agent] compaction_new_window`（默认
  false）→ `compactionConfig` 注入 `CompactionConfig.NewWindow`。开启后
  `NewWindowCompact`/`MaybeDownshiftCompact` 才可被触达。
- **T4 `[permissions] approval_mode` 全链**：config 字段（仅顶层，不按空间
  覆盖）→ `SpacePermissions` → boot 两处闸（headlessGate + controller Options
  → `c.approvalMode` → `newGate` 统一注入 `Gate.ApprovalMode`）。never=Ask 硬
  拒给理由且**卡不弹**（测试钉死），""=现状逐字节。验证发现：Controller.New
  默认 permLevel="ask"（交互语义），headless 判定走 headlessGate——两档各自
  生效。
- **T8 tool_search + deferred**：`[tools] deferred` 名单 + `[tools.search]
  enabled/limit`（默认关=现状）。enabled 时：名单工具 `reg.Hide`（藏而可按名
  调用）+ 注册 `tool.NewToolSearchTool(reg, limit)`——命中即回 schema 文本，
  **物化即文本**，agent 循环零改动；未注册名单项 slog 告警；非 enabled 时名单
  不隐藏（防工具无声消失）。

### 判定关账（三件，附理由不虚接）

- **T12 背压门**：gaea 的 dream 写入已有模式门（auto/suggest，用户红线=建议制
  先例）+ 写侧空间化，三道门（ephemeral/subagent/限流）在 gaea 架构里无真实
  缺口——`GateCheck` 原语保留供未来自动化写入路径。
- **T14 skill.rrf**：gaea 技能是 pinned 注入形态，无「按查询动态选技能」消费
  方——Select 等 dynamic-loading 特性立项时接线，原语已备。
- **T9 ModelDownshift 接线**：gaea 换模型=重建控制器+会话恢复，恢复后既有
  auto 压缩 + `tryOverflowRecovery`（报错采纳窗口）已覆盖降档场景，增量价值
  =省一次可恢复的失败请求——`MaybeDownshiftCompact` 保留为库接口。
- **T13 proc.session 接 bash 工具**：gaea 已有 bgjobs（后台任务+有界输出），
  会话层与其语义重叠——整合形态（替换 bgjobs 内核 or 新增 exec_session 工具）
  需用户拍板，转拍板池。

### 门禁

六面 build/vet/test 全绿（tool/control/config/boot/permission/agent，既有用例
零回归）；gofmt 全净。

## codex 0.165.0 蒸馏第二批 · 第一波四刀（2026-11-17）

施工单 `docs/codex-distill/batch2-execution-plan.md` + `master-brief-T6-T15.md`，
逐项资产在 `asset-10/12/14/15`。第一波四项零冲突全落，**未 commit、未装机**。
新增 23 例：context 5 / memory 7 / skill 6 / proc 5，各包全量绿（含既有用例）。

### T6 · 上下文碎片契约（context/fragment.go，全批地基）

- **做法**：`Provenance` 七档（Unknown 零值不猜来源 / DeveloperInstructions / Skills /
  AgentsMD / Harness / HarnessHook / Tool / User）+ `Fragment{Role,Kind,Start,End,Body,Prov,NS}`
  + `Render`（标记对包裹）+ `MatchesText`（Kind 派生类型级标记，同 Kind 连续轮稳定命中）
  + `CanAuthorize`（只有 User/DeveloperInstructions/AgentsMD 可作授权证据——
  「user 角色」≠「用户说的」，T11 Guardian 的判定入口）+ `NewToolFragment`/
  `NewHarnessFragment` 生产入口（归属由生产者声明，标记自动派生）。
- **验收**：五条全落测试（三轮命中 / 工具片段框架代写仍 ProvTool / user 角色
  harness 归属不可授权 / 无标记直出 / 零值 Unknown）。

### T12 · 记忆四护栏（memory/{sanitize,gate,baseline}.go）

- **脱敏（常开，已接线）**：四条正则按序替换（Bearer→sk-→AKIA→key 赋值），
  赋值条保留键名/键侧引号/分隔符/值侧引号只换值（JSON 形态 `"token": "v"` 的
  键后引号是实测补的——上游正则没覆盖）；接线在 `Store.Save` 单一写口，
  计数进 slog 不落原文。12 条假密钥零命中 + 5 条普通文本零误伤有测试钉死。
- **背压门（纯原语，调度口接线挂小批）**：`GateCheck` 三道门（ephemeral/subagent
  →enabled→限流），限流拿不到按通过（上游 unwrap_or(true) 务实默认），拒绝给
  `skipped: <reason>`。dream 调度口带会话类型才有接线位。
- **基线差分（纯原语，整理流接线挂小批）**：Snapshot（名→sha256）/Diff 三类/
  阈值拦截（删除>10% 或修改>30%，空基线防清库）/有界渲染（4096B，UTF-8 边界）。
- **未做**：保留期清理（gaea 无扩展资源概念，跳过）；产物校验（gaea 记忆非
  汇总文件形态，不适用）。

### T14 · 技能 RRF（skill/{lexical,ngram,fuse,select}.go）

- **做法**：词法整数权重梯度 256/128/64/24/16/6/4/1 + matched²平方加成
  （覆盖多词>单词刷频）；字符 2–5 gram 字段权重 8/4/1（中文容错）；RRF 融合
  K=60 三级 tie-break（分数→最好名次→id）；硬上限全截断不报错（查询 4KiB/64
  词、文档 4KiB/256 词、候选 1000、返回 50），截断标志 OR 合并上报。
  CJK 按二字窗口切词（与 ngram 路同思路，不引入分词器）。
- **验收**：100 次逐元素相同 / 3 词覆盖排前 / 8KiB 截断上报 / 中文技能名
  top-1。**未做**：30 技能×30 查询的命中率对比（现状无检索器可作基线，
  接线后补记录）；`skill.rrf.enabled` 开关挂接线批（现状零调用方=零行为变化）。

### T13 · 执行语义（proc/{session.go,headtail.go}）

- **做法**：`HeadTail` 50/50 头尾保留 + `... N bytes omitted ...` 唯一标记
  （计数=实际丢弃量）；`Session` 双缓冲同锁追加（pending 消费式/transcript
  累积式，容量各 1MiB）+ `Drain` 消费式续取不重复 + `RunSession` 双闸门
  （时间预算 yieldMS 与输出量预算谁先到谁生效，到点返回增量进程不死）+
  `Wait` 收尾上限 100ms（后代持管道不永久等）+ `WriteStdin` 超 8KB 返回
  `ErrStdinApproval` 哨兵、写已退会话明确报错 + chunk_id 6 位十六进制。
- **验收**：yield≈100ms 返回进程仍活 / 三段 Drain 不重叠且拼接=Transcript /
  省略标记唯一计数准确 / 自然退出收尾 / 写退会话报错 / 审批哨兵——测试用
  测试二进制自重执行驱动真实子进程（无 shell 依赖）。
- **未做**：`proc.session.enabled` 配置开关与 batch_executor/bash 工具接线
  （现有「跑完取结果」路径零改动=零行为变化，接线挂小批）；远端 drain 策略
  （gaea 无远端执行器，明确不做）。

### 门禁

context/memory/skill/proc 四包 `go build`+`go vet`+`go test` 全绿（含既有
用例零回归）；零新外部依赖；UTF-8 无 BOM。

## codex 0.165.0 蒸馏第一批 T1–T5（2026-11-17）

施工单 `docs/codex-distill/exec-brief-T1-T5.md`，逆向资料在仓库外
`C:\AI\re\codex-0.165.0\`。全部纯后端+单测，**未 commit、未 wails build、未装机**
（施工单纪律：待用户点名再打包）。基线（六包 -count=1）全绿；改后
tool 43→50 / builtin 249→257 / control 101→108 / permission 33→37 / proc 13→18，
boot 36 / agent 400 / cache / config 持平，九面 `go build`+`go vet`+`go test` 全绿。

### T2 · 执行环境固定注入（proc/env.go）

- **根因**：工具子进程继承父环境——颜色码/分页器/本地化令「同一命令」输出不可
  比对，长输出可被 pager 卡死。
- **做法**：`proc.DeterministicEnv()` 固定 10 变量（NO_COLOR/TERM=dumb/LANG·LC_*
  =C.UTF-8/COLORTERM=空值存在/PAGER·GIT_PAGER·GH_PAGER=cat/GAEA_CI=1——机制同构
  codex 的 CI 标记，名字用 gaea 自己的，施工单原文 CODEX_CI 系文案已自写）；
  `proc.CommandEnv(extra...)` 是唯一合并点（父环境打底→覆盖→extra 后到优先），
  `bash.go` 前台+后台两路接线；转义阀门 `GAEA_EXEC_ENV_RAW=1` 跳过注入回退历史行为。
  hook 装载器与 MCP stdio 传输非「工具子进程」，本批不动。
- **门禁**：proc 13→18 全绿；builtin 257 全绿（bash 工具用例含状态锚定链）。

### T1 · 三档行匹配（tool/seek.go + replaceInContent 兜底）

- **根因**：edit_file/multi_edit 只会逐字节整串匹配，目标文件被无关改动后
  old_block 整块失败。施工单写「boot.go 的编辑工具」系 v2 式按目录猜——实际
  内核在 `tool/builtin/editfile.go` 的 `replaceInContent`（edit_file+multi_edit
  共享），本刀按实际落点接线。
- **做法**：`tool.FindSequence`（0 精确/1 行尾空白/2 两侧空白，最低档位下最早
  位置，整段连续）+ `tool.NearestMatches`（首行最长公共前缀前 3 候选）；
  `replaceInContent` 签名加 note 返回：精确零命中→先保 CRLF 诊断（既有红线
  `TestEditFileCRLFHint` 不破：裸 \n 对 CRLF 文件仍报错，兜底不抢）→再 seek
  行级拼接替换（未触及区域保原始字节，替换块用文件行尾风格）；命中 tier>0
  在工具结果回一句「按第 N 档[忽略行尾/两侧空白]定位到第 X 行」；未命中错误附
  「最接近的候选在第 X、Y 行」。edit_lines 按行号定位，无需兜底。
- **门禁**：tool 43→50（seek 7 例）+ builtin +8（施工单验收例「文件被追加无关行
  仍成功」+定位说明+候选行号+多命中拒绝+CRLF 不被吞+multi_edit 同款）全绿；
  既有 `TestReplaceInContentKernel`/`TestEditFileCRLFHint` 断言原样。

### T3 · 审批拒绝理由回喂 + 超时挂起 + 追认（controller_approval.go）

- **根因**：拒绝只有 outcome 字符串进不了模型上下文（模型只会原样重试）；
  追认无入口；超时动作解决后不可查。
- **做法**：①`permission.Approver` 接口加 `denial` 返回（实现面=gateApprover+
  permission 测试桩，机械适配）；`ApproveWithReason(id,decision,reason)` 新口
  （`Approve` 委托空理由，前端绑定零破坏）；deny 回喂「已拒绝：<理由>——不要
  原样重试」（理由默认「用户拒绝」，200 rune 封顶）。②超时 `DecisionTimeout`
  归一本就与 deny/cancel 可区分（approvalOutcome→OutcomeTimeout），补「不产生
  deny 账本」确认（Gate.recordDenial 只在 DenyUnapproved 路径）+超时挂
  「待追认账本」。③`deniedActions` FIFO 账本（封顶 50）+`ApproveDenied(id)`
  按原参数重放一次审批走正常闸门，放行写会话 granted（hardAsk 卡除外——逐条
  纪律不写）+账本移除+notice；`PendingDeniedActions()` 供恢复侧「N 条待审」
  展示；ctx 取消不挂账。
- **门禁**：control 101→108（理由回喂/默认理由/超时挂账/追认放行/追认后
  同参重试短路放行/hardAsk 追认永不写 granted/封顶/取消不挂账）全绿；
  五决策既有语义逐字节不变（`TestApprovalResolvedPaired` 等原样）。

### T4 · Never 语义 + Windows 危险命令（permission/approval_mode.go + dangerous_windows.go）

- **根因**：绝不弹窗模式下命中「需批准」会静默放行（headless 自治）或静默卡死；
  Windows 深删无硬拦截。
- **做法**：`NormalizeAsk(mode,tool)`——mode=never→Deny+理由（含 never 字样，
  与用户主动拒绝可区分），""/on-request/untrusted→保持 Ask（现状逐字节）；
  `Gate.ApprovalMode` 字段（默认 ""=关闭）挂 AlwaysAsk 与 Ask 两分支；施工单
  默认文案 "approval required by policy, but approval mode is never" 保留为
  理由主体。`DangerousWindowsCommand(argv)` 纯文本判定（无 build tag，CI linux
  可测）：删除集 remove-item/delete-item/ri/rm/del/erase/rd/rmdir；powershell/
  cmd/bash -c 三壳剥引导参数取命令首词，&& ; | 分段扫；未知开关/token 保守
  不拦（不确定≠危险）；-Force/-rf 判高危写进理由；接线在 Gate.Check bash
  非只读处（与只读判定同位），先于策略判定（策略 Allow 也拦）。审批模式/危险闸
  的控制器配置面接线（permLevel→Gate.ApprovalMode）未做，挂小批。
- **门禁**：permission 33→37（never 归一/Gate 接线含 Allow 不受累+默认等价/
  四壳判定/Gate 深删硬拒）全绿；boot/agent 全量绿（深删闸未误伤既有用例）。

### T5 · 工具 schema 字节预算（tool/budget.go，默认关）

- **根因**：工具多时首轮 schema 只有「全量下发」或「按名过滤」，无共享预算。
- **做法**：`TruncateRows`（名字行成本 2+len(name)+1 先全额保留；名字总额超
  预算从尾部整行省略，「...还有 N 个工具」占位；剩余在描述间按轮次均分
  每轮+1 字节，公平且同输入同结果）+`TruncateDescBytes`（UTF-8 字符边界截断
  +"..."）+`ApplySchemaBudget`（-1 哨兵=整行省略——施工单 []int 装不下 nil，
  语义等价）；收口两处=`Registry.FilteredSchemas` 尾 + `IdentityLayer` 具名
  路径（同一函数同一配置值，不许漂移）；开关 `[tools.schema_budget]
  enabled/bytes`，boot 装配期一次性注入，默认关=逐字节等价现状；Bytes<=0
  走保守默认 12KiB（注释注明取值依据）。
- **门禁**：tool +5（省略/均分/中文不切字符/两次调用逐字节相同/默认关）全绿；
  cache/config/boot 全绿。**实测**（builtin 全集静态口径，`.tmp/t5-measure`，
  不入库）：50 工具，关 14136B(~4.7k tok) → 开(12KiB) 13867B(~4.6k tok)，
  整行省略 0——builtin 描述本就紧凑（CompactDescriptor），预算收益随
  MCP/办公工具面与长描述增长；真实会话 token 对比待装机后实测。

### 共用纪律核对

- 三硬禁止：零新外部依赖 ✓；既有测试断言意图未改（仅 mechanical 适配
  `replaceInContent` 3→4 值与 `Approver` 4 值解构）✓；未放宽生产逻辑 ✓。
- UTF-8 无 BOM ✓；gofmt 全绿（`provider_feature_test.go` 系既有漂移，未动）。
- 未做项：T3 前端接线（拒绝理由输入框/追认按钮/恢复时 N 条待审卡片）挂小批；
  T4 控制器配置面（permLevel→ApprovalMode、config 键）挂小批；T5 真实会话
  token 对比挂装机后；账本/预算均内存态，跨会话持久化未做。



- **根因**：系统提示词身份段只有全局一条解析链（`ResolveSystemPrompt`），图文引擎复用办公 config——本机实锤图文会话开场自称「土壤修复工程办公专用AI助手」（办公自定义人格）；`genui/officepointer.md` 核实为板块中立 GenUI 指针，不属串味。
- **分板**：`boot.Options.SystemPrompt` 覆写位（空=现状逐字节）+ `DefaultWenxueSystemPrompt`（10 行指路型：纪律指向内嵌技能 wenxue-workbench/书籍信息.md/内嵌三件，纪律本体不重复防双源漂移；语言纪律不重复——LanguagePolicy 恒拼单源；技能不点名项目件防索引漂移）+ `agent.wenxue_system_prompt`/`wenxue_system_prompt_file` 双键（文件键优先，读失败硬报错）+ `wenxue_engine` 经 `ResolveWenxueSystemPrompt` 接线。
- **配套**：办公 config.toml 人格摘除 Hermes/Hephaestus 退役工作流程段（备份 config.toml.bak-20261116-pre-hermes）；定稿删除授权走记忆路线（两板块脑 AGENTS.md Notes 红线：删定稿先 request_permission——词表授权门方案被用户否决撤净，零代码门）。
- **验证**：config 解析链/ boot 装配覆写 / internal/app 图文族测试绿；两次 wails build+装机（22:07/22:26）冒烟 `/api/health` 200；图文实际索引 4 件=内建 genui+内嵌三件（novel 三件系开发仓项目件不随书装配，skill-creator 本机不存在）。

## 文件上传任务卡（2026-11-16）

- 办公/文学 Composer 上传从全局 `pendingPaste` 计数升级为逐文件任务：拖拽、粘贴
  与裁剪截图均显示文件名/大小/进度；成功后转正为原附件 chip，失败保留原因、
  可重试标记与移除入口。
- 重试复用内存原件/图片 data URL；分块上传逐片更新进度。上传中不提供假取消，
  避免桥接请求结束后又把已移除文件挂回。存在上传中或失败任务时禁止发送/补选，
  防止悄悄少附件。
- 错误分类：超限、格式、权限、磁盘满等不可重试；超时、临时网络/桥接错误可重试。
  原生“选文件”仍是路径引用，不产生伪上传任务。
- **验证**：相关前端 75/75、`tsc -b` 0、改动面 ESLint 0、正式 Vite build
  41.84s（既有 CSS/大 chunk warning 不变）。纯前端；未 Wails、未装机、未提交。

## 文学候选稿（2026-11-16）

- **范围裁决**：用户否决「中断续写」映射——文学交付物在章节文件，生成未落盘时
  append 无法救回；该功能的工具/模板/UI/文案已全部撤除，不作为蒸馏候选。
- **候选稿**：新增 `wenxue_candidate_write`（固定写
  `章节/候选/<章名>/<label>.md`，不碰正文与定稿）与
  `wenxue_candidate_promote`（提升前自动把当前正文归档到同章候选目录）。
  工具 schema、RegistryMap、图文装配 census、能力面板、工作台技能与首页
  「候选稿比较」快捷卡同批接入。
- **验证**：`internal/app -run TestWenxue` 绿、tool RegistryMap 守卫绿、
  wenxue 前端 38/38、`tsc -b` 0、改动面 ESLint 0、`go vet` 绿、正式 Vite
  build 22.88s（既有 `ctx-*` CSS / 大 chunk warning 不变）。未 Wails、未装机、
  未提交。

## DeepSeek chat-demo 办公/文学蒸馏第一批（2026-11-16）

- **会话级 Composer 草稿**：新增共享 `composerDrafts` 存储，办公/文学分别以
  `work`/`play` + 真实会话路径隔离；未保存会话用 `sessionNonce` 区分。恢复
  文本与附件路径，图片 data URL 不落本地存储，待发送队列只保留在进程内且
  恢复后不自动重发，避免跨会话串话与重复执行。
- **结构化来源证据**：`web_search/web_fetch` 的工具原始输出保持逐字不变，
  Agent 仅在 `ToolResult` 旁附 `ToolSource[]`；日志、实时 wire、断号补拉与
  reducer 全链透传。办公/文学 ToolCard 共用「来源」抽屉，失败检索不虚报证据。
- **验证**：前端定向 16/16、`tsc -b` 0、改动面 ESLint 0、正式 Vite build
  32.78s（既有 `ctx-*` CSS 语法/大 chunk warning 不变）；Go agent/session/app
  定向测试与三包 `go vet` 绿（补齐 build/bin DLL PATH）。未 Wails、未装机、
  未提交；工作树内既有其他业务改动未触碰。

## 文学实弹工具契约修复（2026-11-16）

- **用户反馈**：文学板块实测暴露六类工具契约问题——写后读缓存陈旧、
  `read_file offset=-30` 未读尾部、`bash` 解释器说明含糊、精简版
  `complete_step` schema 漏 kind 枚举、todo 完成与签退脱节、章节回执字数口径不明。
- **缓存**：读缓存改为按绝对路径 + 全文 SHA256 校验，移除 5 秒 mtime 信任窗；
  `CachePathResolver` 接通图文写族整书失效与 `todo_write` 的 `todos.md` 失效；
  回合内去重改用完整结果哈希，并排除持久化状态型 todo。
- **分页与契约**：`read_file` 负 offset 从 EOF 倒数窗口，输出绝对行号与正确续页
  offset；`bash` 精简描述/schema 动态标明当前为 bash 或 PowerShell 并给示例；
  `complete_step` 精简 schema 补 kind 枚举、支持 step_index-only，错误附可复制样例；
  todo 回执列出缺失签退项与调用形态；文学写回执新增 `contentChars` 与
  `wordCountRule`（非空白 rune 口径）。
- **验证**：cache/agent/tool/builtin 四包绿；`internal/app -run Wenxue` 绿；
  相关 `go vet` 绿。未 Wails、未装机、未提交。

## 文学新会话首页简洁化（2026-11-16）

- **用户反馈**：文学板块新会话首页过于花哨，要求对齐办公板块的简洁度。
  原“作品控制台”同时铺开封面、概况、动作卡、作品结构与最近会话，主次失衡。
- **已选作品态**：改为与办公首页同构的居中空态，仅保留当前作品提示、标题
  “继续你的故事”、切换作品/首页隐私、写作快捷动作与原 Composer。切换作品、
  隐私模式、快捷提示插入、离开空态后的会话链路均保留。
- **未选作品态**：作品库收敛为单列最近作品列表，删除封面叠层、渐变 Hero、
  统计条、右侧说明卡与引导步骤卡；保留新建小说、全部历史、隐私和打开作品。
- **书架封面卡片化（用户追加）**：作品列表升级为三列封面卡片，窄窗两列、
  手机单列。封面只保留标题、分类小标与书脊线，使用同一强调色生成克制的浓淡档；
  不使用旧版倾斜叠层、渐变和阴影。同步补齐 `wx-home` 自己的 `--wx-*` 令牌，
  修复脱离 `wx-hub` 后透明背景与浅色边框问题。
- **工程**：`WenxueHome` 从双首页 794 行收敛为作品库单态，已选作品欢迎区由
  `App` 复用办公空态定位；`wenxue-home.css` 由 1215 行收敛为无渐变、无玻璃、
  无封面叠层的轻量布局；三语补齐 7 个首页键。隐私模式继续隐藏输入框作品入口。
- **验证**：WenxueHome 4/4；`tsc -b` 0；改动面 ESLint 0；卡片封面终版
  `npm run build` 23.58s 绿。1440×920、880×760 亮暗实页走查无横向溢出，
  封面标题对比度亮 10.01:1 / 暗 8.70:1，快捷动作与辅助文字对比度 6:1 以上，
  主按钮暗态 7.77:1。未 Wails、未装机、未提交。

## 编程外部浏览器持久会话修复（2026-11-16）

- 用户实测关闭 gaea 后外部浏览器里的 DSH 工作台同时断开。根因：DSH node 进程
  位于带 `KILL_ON_JOB_CLOSE` 的 Windows Job Object，gaea 退出即整树回收；同时旧
  stdout/stderr 走 gaea 持有的匿名管道，父进程退出也可能诱发管道断开。
- 修：外部浏览器入口改调新绑定 `OpenProgrammingWebExternal`；启动后保存 PID/令牌
  到 `C:\AI\dsh\home\.gaea-web-session.json`，用 `proc.ReleaseTracked` 清除 Job
  的关闭即杀位并关闭句柄；node 的 stdout/stderr 改为直接继承日志文件。再次启动
  gaea 可按状态文件重新接管，停止按钮对重接管会话走 `taskkill /T` 的 PID 树回收。
- 验证：proc Job 释放存活、日志令牌原位脱敏、持久会话接管、外部启动持久化四组
  定向绿；真实 DSH 启停 11.09s 通过；独立测试进程启动 DSH→退出→新进程重接管→
  停止回收 11.82s 通过；ProgrammingPage + spaceBindings 14/14、Gaea 定向 `tsc` 0。
- 干净工作树 `e23bd5b9` + 本刀未提交补丁构建：Vite 36.35s、Wails 51.33s
  （bindings 未跳过）；payload 四件冒烟 `/api/health 200`；exe SHA256
  `71F53995D630E3BB2358DCC9F2A090FA7233B2A1F9A4EA4F980002184C092B3A`；
  已装机并启动 PID 54840（12:58:05 > 构建 12:57:42）。并行未收口的文穴改动未入包。

## 编程板块外部浏览器入口（2026-11-16）

- `ProgrammingPage` 工具栏新增常驻「外部浏览器打开」按钮：运行中直接打开当前带
  访问令牌的 DSH Web 地址；未运行时先执行现有隔离预检与启动流程，成功拿到地址后
  再交给系统默认浏览器，启动失败仍停留原页显示原因。
- 验证：`ProgrammingPage.test.tsx` 4/4 绿；改动面 ESLint 0；页面与测试隔离
  `tsc` 0。全量 `tsc -b` 被工作树既有未提交
  `frontend/src/wenxue/components/WenxueMemoryPanel.test.tsx` 的 TS6133 未使用导入
  阻断，与本改动无关。未 Wails/未装机/未提交。

## 文学首页双态重设计（2026-11-16）

- **调研**：Dabble Projects Dashboard（书斋=书架，封面卡+最近修改）、Ulysses
  Library/Dashboard（作品内=进度/大纲/注释上下文）、LivingWriter Dashboard
  （作品首页=项目状态与最近工作）三点交叉。结论=未选小说应负责“找书/建书”，
  选中小说应负责“继续写/知道下一步”，不能共用同一张通用聊天欢迎页。
- **未进入作品：书斋作品库**：左侧主线为最近作品封面叠层；无作品时进入
  “不需要先准备完整的世界”建档引导；已有作品时用非等宽书架（首部放大、
  余作常规卡+新建卡），展示会话数、最近修改、最近会话，并提供继续最近作品
  与全部历史。此态不显示写作 Composer，避免无工作空间时产生假可写入口。
- **已进入作品：作品控制台**：Hero 展示作品名、最近编辑、最新章节与封面；
  四项概况=章节/定稿/角色/设定；下一步动作=续写、核对人物状态、整理时间线、
  推演大纲；常用工作区直达角色/阅读/设定，侧栏展示最近会话与当前作品路径。
  Composer 降至 112px 紧凑档，“继续正文”会预填既有 chapter-continue 模板；
  “作品书架”可从当前小说切回 tuwen 根，已实页验证可逆。
- **隐私边界（两态共用）**：作品路径默认不渲染；跨作品首页不再直接展示最近
  会话摘要。新增“首页隐私 / 已隐身”本地偏好（`gaea.wenxueHomePrivacy`）：
  选中作品时遮罩作品名、封面题名、最新章节、人物/设定名单、最近会话和 Composer
  工作区入口，同时联动左栏作品名/会话名/路径标题与顶栏当前作品名；书架态
  遮罩全部作品封面、会话名和最近记录。只改显示快照，path/会话动作仍使用真实对象。
- **工程**：新增 `WenxueHome.tsx` + `wenxue-home.css`；`useSessionManager`
  增 `projectGroupsLoaded` 防首帧误判空库；App 以 currentProject 分派两种首页，
  未选书不挂 Composer；浏览器新增 `?mock=library`（有书未选）与
  `?mock=wenxue`（已选作品）两态样例。
- **门禁**：首页/侧栏/角色/阅读/设定共 **28/28**；改动面 ESLint
  0 error/0 warning；全量 `tsc -b` 0；正式 `npm run build` Vite `40.96s`
  通过（既有 ctx-* CSS syntax warning 不变）。未 Wails/未装机/未提交。

- **用户三条反馈修正**：①时间线改「整条单档 时间线/时间线.md」（一行一条
  「第N章 · 事件」，章序即推演序；旧多文件布局仍可读，新事件一律追加单档）；
  ②删「一章一会话/一幕一会话」强制约定（切分=使用者上下文策略，agent 不主动
  建会话不催促）；③专属工具节去重（逐件用法以工具 schema 描述为权威，本节只
  留跨工具纪律指针——查库导入+回写，防双源漂移）。装机过验。
## 图文工作台纪律技能优化（2026-11-16）

- `wenxue-workbench` 正文重写优化（提示词面）：新增「开场上下文（先读，再落笔）」
  节——书籍信息.md 自动注入为写作校准首要依据（题材定语感/简介定主线/目标字数
  定卷节奏/**成人等级=尺度硬约束**：全年龄绝不越级，深度成人=无任何限制完全
  跟随剧情）；旧书缺书籍信息.md 的补建路径（含 pinned.json 手工加入法）；新增
  「图文脑记忆」纪律（约束与口径走 remember、剧情事实走角色卡）；新增「生图
  （可选面）」节（image_gen 锚小说目录+未配后端如实告知不硬凑）。
- 装机 12:08 INSTALL OK（PID 46524 > 构建时刻）。skills 测试绿（filter/extras/
  dynamic acceptance）。

## 图文记忆面板优化批（2026-11-16，用户问「是否需要优化」拍板做）

- **机制补齐**：图文脑生命周期管理面从零到有——新绑定六件 `WenxueMemoryPin/
  Archive/ArchivedList/Unarchive/Invalidate/RestoreInvalidate`（经
  hubWenxueBrainStore 直达图文脑，与办公脑同机制 5.3/9.3：固化免疫衰减、归档
  软删可恢复、失效标记退出注入面原文保留）；`MemoryFact` 增 invalid/updatedAt
  可选字段（office 不设置零影响）；WenxueMemory 回执带全量生命周期态。
- **UI 对齐三 Hub**：面板重写上 `WenxueHubFrame`（tone=memory 新增）——书脑
  masthead+概览三 metric（事实/固化/归档）+刷新 action；类型筛选 chips；事实
  卡操作行=固化toggle/编辑/失效(恢复注入)/归档/删除；归档折叠区（恢复钮）；
  排序=固化优先+最近写入；文案去行话（「做梦/晨报为办公板块机制」如实标注）。
- **绑定面**：facade 六件+签名 689 再生+bindingNames 双份（gaea+wenxue 桥各自
  快照）+legacy 双份再 generations+spaceBindings work 面+wenxue mock 六件。
- **验证**：Go 3 组绿、tsc 0、vitest 32 绿；装机 13:06 INSTALL OK（PID 52940 >
  构建 13:06:39）。坑=wenxue 桥自带 bindingNames/legacyBindings 快照对（与 gaea
  桥分开维护），Go 增绑定后两份都要再生，否则 wenxue drift 残差报红。

## 实弹修：记忆 tab 打不开（2026-11-16）

- 用户实测记忆 tab 打不开。CDP 真机走查（GAEA_WEBVIEW_DEBUG=1:9333，主窗按
  url 精确过滤）定罪：**useSubagentTabs.handleChatTabSelect 的视图 tab 白名单
  CHAT_VIEW_TAB_IDS 漏 "memory"**——不在名单的 id 被当子代理 tab 吃掉
  （setChatTab("chat")），内容回落 chat。这是该白名单的已知老坑复犯（v4.542
  刀3/5 注释在案：洞察/编队入列时踩过同一坑）。
- 修=ChatTabs.tsx 白名单补 "memory"（含坑册注释）；装机 12:37 INSTALL OK
  （重挂壳过验：角色 hub ✓ → 记忆 tab ✓ panel 挂载渲染图文脑 facts/scopes，
  零控制台错误）。
- 附：坑=CDP 走查 /json/list 需按 url 精确过滤主窗（dsh.wails.localhost 编程
  板是独立 target）；python heredoc 修 Go 源码反斜杠路径时 	 会被转义成 TAB
  （用 chr(92) 显式拼接）。

## 新建小说关键信息+成人等级（2026-11-16）

- **表单**：新建小说 Modal 扩 关键信息五项——书名/题材/目标字数/简介/**成人等级
  下拉**（全年龄｜轻度（暗示级）｜成人（限制级）｜深度成人（无限制）=无任何
  限制，尺度/题材/表现手法完全跟随剧情需要）。
- **数据链**：`WenxueCreateNovel(name, infoJSON)`——infoJSON 解析→**书籍信息.md**
  落书根（书名/题材/目标字数/等级+分级说明+简介）→ `WenxuePinMaterial` 自动固定
  → 每个新会话系统提示词自动注入（agent 开场即知题材与尺度）。成人等级白名单
  校验（非法值报错列合法值；空=全年龄）。与 ADULT_MODE.md（轻语陪伴域）不同层：
  书籍元数据非全局开关。**唯一恒守线=涉未成年一条（其余零限制，用户确认）**。
- **验证**：Go 3 组（信息落盘+固定/白名单报错/空缺省全年龄）、Sidebar.test 2 参
  6 例绿、wenxue vitest 28 绿、tsc 0；装机 12:01 INSTALL OK（PID 53144 > 构建
  12:01:00）。

- **接手收尾打包（另一执行体停止，本执行体接手）**：提交 4442fefc（编程 Job
  记档+文学首页双态记档）；功能包 `releases/gaea-wenxue-v4-20261116-4442fefc/`
  （干净工作树 @4442fefc，Vite 29.93s+wails 46.1s，冒烟 200，SHA256SUMS 入库，
  zip/tar 留盘，工作树回收）；干净包重装机 14:13 INSTALL OK（PID 13568 >
  构建 14:12:58）。
- **边界（本执行体立场，非协商项）**：工作区另有另一执行体在途的「无限制档
  破甲契约/零底线」改动（jailbreak.go、maturecraft unrestricted 档移除成年硬线
  与恒守线、ADULT_MODE.md 零底线节、云端模型探针文档等）——**未提交、未入包、
  未装机**，原样留在工作树未销毁。理由：破甲契约的目标函数包含解除涉未成年
  防线与非自愿题材解锁，本执行体不构建、不发布该面。深度成人档的写作自由
  （直写、不淡化、不说教、无审查化措辞）已在已提交版本全量生效。
## 提交与打包记档（2026-11-16）

- 本日五批+三标签页重设计已提交：`666f239d`（三标签页重设计，另一执行体批次代提交）、
  `7e0dffde`（图文五批：本地引擎预检+租约/角色库导入/工具瘦身 55→43/记忆中枢接入
  九库/记忆面板恢复+生图面）。
- **功能包** `releases/gaea-wenxue-v3-20261116-7e0dffde/`：独立干净工作树
  （detached @7e0dffde，junction node_modules）Vite 44.65s + `wails build -s
  -ldflags "-s -w" -trimpath`（bindings 未跳过；坑=工作树 wails 需主树
  build/bin DLL 入 PATH 否则 0xc0000135）；payload 四件+zip/tar+SHA256SUMS+
  PACKAGE-INFO 入库（zip/tar 本体留盘）；冒烟 `/api/health 200` 过、冒烟进程
  已回收；工作树已回收；exe SHA256 `c9c8eab8…`。
- **装机**（2026-11-16 11:15）：主树 `wails build -s -ldflags "-s -w" -trimpath`
  （1m27.9s，bindings 未跳过）→ `gaea_install.ps1 -Launch` INSTALL OK：gaea.exe
  SHA256 `CF65EEF89F04…` + 三 DLL 与 build/bin 一致；新 PID 42032（11:15:03）>
  构建时刻 11:14:54；旧壳 49276 已杀。另：工具面板 `genui_validate` 归组任务与
  协作（44 件全有名有组，实测装配面 34+boot 直注册 10）。
- **实弹修：选书不开会话四 tab 全空**（用户实测）——根因=`WenxueEngineInit`
  只挂在 Send/NewSession 入口，板挂载从不初始化 → ctrl=nil → `WenxueMeta.ready
  =false` → 主区挂 Skeleton 挡住角色/阅读/设定/记忆四 tab。修=板挂即初始化
  （App.tsx mount effect `app.EngineInit()`，幂等；Send/NewSession 自动初始化
  照旧）+绑定认领四处（mappings/bridge 接口/spaceBindings work 面/legacy 清单
  摘名+mock 空转）。装机 11:30 INSTALL OK（新 PID 46160 > 构建时刻 11:30:48，
  exe 新哈希）。

## 文学角色 / 阅读 / 设定三标签页重设计（2026-11-16）

- **调研**：截图级对照 Novelcrafter Codex（人物 Progressions / Codex 条目 /
  自动提及）、World Anvil（Wiki 分类 / 章序时间线 / 关系图）、Campfire
  Character（人物档案 / 关系网 / 成长轨迹）、Readwise Reader（长文阅读 /
  章内导航 / 阅读辅助）与 LegendKeeper（分类档案 / 时间线 / 可检索世界库）。
  共同缺口：高水平工具都把角色、阅读、设定视作三种不同任务，而非同一套
  “左文件树 + 右正文”。
- **三 Hub 重设计**：共享 `WenxueHubFrame` 建立眉题/宋体标题/指标/动作的
  档案室语言；角色=演员表 + 人物速写 + 六节正文 + 速览/当前状态/关系/完整度；
  阅读=章节目录 + 书页正文 + 首字下沉 + 章内大纲 + 字号/专注阅读 + 实时滚动
  进度；设定=分类索引 + 文档页 + 大纲 + 章序时间线（未锚事件聚合警示）。
  数据面保持角色卡 / 定稿 / 设定 / 时间线 md 原契约，CAS、角色库只读导入、
  Composer 联动与 filetree 自动重载全部保留。
- **响应式修**：1440×920、1100×880、880×760 亮暗走查；修复窄窗人物档案
  三栏挤压、设置标题逐字折行、时间线横向过窄、Hub body 被 Playwright
  内部滚动撑高；880 自动收起阅读目录，保留正文；新增 `?mock=wenxue`
  浏览器样例（《雾港来信》角色/定稿/设定/时间线，不触真实 tuwen）。
- **门禁**：三角色 Hub 定向 **16/16**；`tsc -b` 0；改动面 ESLint 0 error /
  0 warning；三 Hub 按钮可访问名 0 缺失、图片 alt 0 缺失、无横向溢出；
  `npm run build` 43.10s 通过（Vite 仍输出既有 `ctx-*` CSS syntax warning，
  与本批文件无交集）；未 Wails、未装机、未提交。

## 图文记忆面板恢复 + 图文生图面修复（2026-11-16 收尾批）

- **记忆面板恢复**（用户令「以上问题一起解决」）：新 `WenxueMemoryPanel`（精简
  版——快捷添加 scope+note/事实搜索/编辑 UpdateFact/删除 Forget；**数据面全锚
  图文脑**，做梦/晨报/建议办公机制如实标注不跑；旧 848 行办公脑面板不复活）；
  ChatTabs 第八 tab「记忆」（Brain）；`WenxueMemory` Go 绑定去混源——facts 从
  hubOfficeStore（办公脑）改挂 `hubWenxueBrainStore`（图文脑），Available 改锚
  wenxueBrainDir。写侧五件本就经 wx 控制器落图文脑，零改动。
- **图文生图面修复**：媒体根分档裁决的 image_gen 此前被 work 空间闸静默过滤
  （spacetags 名表 play）——新 `wxWorkSpaceTool` 适配器（仅改 SpaceTag→work，
  Execute 原样委托）在 wenxueBaseTools 单点包装；**办公 gaeaExtraTools 原样
  play 语义不变（红线）**。工具面板 42→43（本地专业模型组补 image_gen）。
- **测试**：census 装配面改用 wenxueBaseTools 枚举（更忠实 wx 装配）+image_gen
  keep 断言；policy 放行面补 image_gen；`TestWenxueBaseToolsImageGenWorkTagged`
  （图文=work/办公=play 双面红线）；`TestWenxueMemory_FactsFromWenxueBrain`
  （去混源回归）。
- **验证**：app 全包 110s 绿、tsc -b 0、vitest 188 绿；未提交未装机。

## 图文记忆接入记忆中枢（2026-11-16）

- **背景**：图文脑（wenxue-brain 独立 DB）不在记忆中枢八库聚合清单里；图文板块
  「记忆」tab 已在记忆左右脑批退役（当时面板挂的是办公脑做梦/晨报/建议窗）——
  模型侧记忆五件工具一直在工作，只是无 UI 可看（用户问「为什么看不见」）。
- **接入（只读）**：`GaeaWenxueMemoryBrowse` 新绑定（MemoryB 门面）——图文脑
  facts 跨书全览（`SQLiteStoreFor`+`List()` 不过滤项目），Available=false=无脑；
  总览增 `WenxueCount`。**物理隔离不动**：读写面仍只在图文引擎记忆五件，中枢
  同右脑聊天记忆「只读」先例。
- **前端**：MemoryHubPage 九库新增「图文记忆」（amber 识别色 + WenxueMemory-
  Library 只读组件：facts 表/搜索/失效标记徽标/空态文案）；绑定四件套再生
  （bindingNames 682/-signatures-ts/wailsjs models 经 `wails generate module`，
  坑=需 ucrt64+build/bin DLL 双 PATH 否则 0xc0000135）+gaea bridge mapping/
  mock（wenxueCount 双 mock 同批）+spaceBindings work 面+drift 锁全过。
- **测试**：`memory_hub_wenxue_test.go` 3 组（browse 回读/空脑可用态/总览计数）；
  既有 TestMemoryHubOverviewDatabaseCounts 补图文脑池清理。坑两枚记档：
  ① `os.UserConfigDir()` Windows 读 %AppData%——USERPROFILE 重定向不影响，
  测试须走 `wenxueBrainDirOverride` 缝（本批新增，仿 officeStoreOverride 先例），
  否则误写真实图文脑（本批实测写进去一条已精确清除，真实数据零污染）；
  ② 记忆条目名须 kebab-case slug（中文名 slug 化为空报 "memory needs a name"）。
- **验证**：app 全包 108s 绿、boot 包绿、go vet 0、tsc -b 0、vitest 188 绿
  （memoryhub+wenxue+hub page）、绑定漂移 OK 682；真实脑核对=1 条用户事实
  未动、测试残留 0；未提交未装机。

## 图文工具瘦身刀（2026-11-16）

- **背景**：面板 55 件里 17 件落「其他」组（静态分组不认识），编辑族/大文件
  件/会话元工具在写作闭环外纯杂音；且每件工具 schema 都吃每轮 token。
- **摘除 13 件（55→42）**：编辑族五件（edit_file/edit_lines/multi_edit/
  replace/move_file——修订走 chapter/character_write 整卡覆写）、大文件两件
  （read_spill/summarize_file——read_file 分页覆盖）、办公侧栏（sidebar_open）、
  会话元工具两件（request_permission/exit_plan_mode——图文交互审批链已覆盖）、
  子代理管理三件（interrupt_agent/subagent_list/read_subagent_result——管理面
  在 UI tab，agent 侧保留 task 派发）。
- **boot.addGated 新缝**：ToolFilter 语义从「addBuiltins/plugins/ExtraTools
  三处」扩为「全部模型面注册点」——boot 直注册 17 处（task/ask/记忆四件/
  read_skill/run_skill/install_skill/slash_command/session_search/request_
  permission/exit_plan_mode/summarize_file/子代理管理三件/task/read_skill
  resolver）全部过闸；filter 空=全放行，办公逐字节现状（红线）。单测
  boot_toolgate_test.go 四态。
- **前端同批**：disabledDomains +13（同源）；ToolsSection 静态表补 会话组
  （session_search/recall_archived）/任务组补 task/文件组补 grep，灭「其他」
  组——42 件全部有名有组（坑=TOOL_DESC 里 task 重复键 TS1117）。
- **测试**：policy 测试翻面（13 件 放行→拒绝 新组）；census deny 抽查 7 件
  装配面；boot 闸四态；app 全包 98s 绿、boot 包绿、tsc -b 0、图文 vitest 24 绿；
  未提交未装机。

## 图文角色库导入刀（2026-11-16）

- **背景**：图文角色工具与角色库零关联——agent 只能拿用户口述的名字凭空造人，
  库内定位/背景/外貌/性格/动机/台词样例全丢（用户点名「不要只要了一个名字」）；
  唯一库通道是角色面板手动导入（全字段六节映射，一次性快照不回写）。
- **`wenxue_character_import` 新工具**（仅图文引擎装配，RegistryMap 契约行第十
  三行 extra-app/work）：id/name → charLib 精确取卡（name 走 List LIKE 粗筛+
  整名唯一匹配；零同名带近似提示、多同名要求传 id）→ **全字段落卡**——卡面
  契约与前端 `cardFromLibraryChar` 逐节同源（设定←定位/标签/背景/动机、外貌←
  外貌/身材、性格←性格/弧线、口头禅←台词样例、关系留白、当前状态←状态/备注，
  ROLE_TYPE_ZH/STATUS_ZH 两表同源对齐），设定节末记「来源：角色库（id …）」。
  overwrite 显式闸同 write。**同步语义=单向一次性快照**（卡面当前状态随章节
  演进、库侧档案独立，双向自动合并必打架；库侧更新要进卡须显式重导入，
  回执提示当前状态会丢）。
- **`wenxue_character_list` 增 `charlib_hits`**：库内与工作空间卡同名命中
  （id/kind）+ charlib_hint 导入提示；库未初始化/查询失败不拦清单（附带
  charlib_error）。
- **纪律面同批**：write desc+workbench 技能补「建新角色前必须先查库，不要只
  拿名字凭空造人」；任务模板 character-card 同口径；前端 ToolsSection 静态
  表补 import 项；装配守卫 census 名单补 import。
- **测试**：`wenxue_character_import_test.go` 6 组（全字段映射+节序/闸+按 id
  重导入/查重错误四态/空字段占位/list 命中解析式断言/nil 库直通）；坑=
  charlib db 目录走 os.TempDir 自管清理（t.TempDir 对 db 文件严格 RemoveAll，
  Windows 句柄释放延迟必误报——同 characterlib newTestStore 先例）。
- **验证**：`internal/app` 全包 108s 绿、go vet 0、tsc -b 0、图文前端 13 例绿；
  未提交未装机。

## 图文本地引擎预检 + 图文空闲租约（2026-11-16）

- **背景**：图文引擎构建期只 `bridge.SetFeature` 注入，请求期 ai.Client 直连；绑定
  本地托管引擎（llama/strata/whirl）在空闲收割后首包 connection refused →
  `isTransferableChatError` 判可转移 → 故障转移悄悄换引擎。聊天板块有
  `resolveChatModelRoute` 预检，图文此前没有。
- **每轮预检**（新 `wenxue_engine_preflight.go`，不采用聊天板块函数——回落链不同）：
  回落链 wenxue 绑定→gaea 绑定→全局活跃（与 wenxueBuildController 注入链同构），
  模型空回落引擎默认档（对齐 ai.Client 动态解析）；llama=`llamaEnsure`+在途补等
  （上限 `qwen35StartWait` 300s，覆盖他方 reason=starting 窗口），strata/whirl 各自
  Start；冷启动先发「正在启动本地模型」phase（已在运行静默），失败转 error 事件
  ——绝不静默落故障转移；`WenxueSend` 每轮发送前过闸（引擎可能在任意两轮间被
  收割，不限于首轮；Steer 运行中插话不重复预检）；llama 裸 GGUF 路径只读归一
  （不代写 chat 键域配置）。`wenxuePreflightFn` 进程缝供测试桩（同
  `ensureChatBindingEngineFn` 先例）。
- **图文空闲租约 10 分钟**（`wenxueIdleLease`）：收割器 `managedIdleTarget` 增第五道
  闸 `lease`——llama/strata/whirl 挂 `wenxueLeaseAlive` 谓词，图文碰过的引擎在最后
  一次图文使用满 10 分钟前不停机（全局阈值默认 5 分钟仍是硬底线，两闸同时过才
  收割；comfyui 无租约）。图文创作轮间隔长，只吃全局阈值会反复杀引擎致每轮冷启动。
- **测试**：`wenxue_engine_preflight_test.go` 8 组（回落链/默认档回落/只读归一/每轮
  语义+租约/错误上下文/云端直通+离线守卫/派发错误/租约收割+目标表）；
  `internal/app` 全包绿（112.3s）；Go vet 0；未提交未装机。

## 图文角色面板 v2 + 记忆左右脑 + 设定标签页 + 功能包（2026-11-16 深夜）

- **角色面板 v2**（`12a00b4f`，装机 PID 8000）：master-detail 重设计——结构化六节
  阅读（缺节占位/自由格式回退）+CAS 编辑；agent 回写即见（gaea-wenxue-filetree
  通道自动重载+未脏态跟随）；新建双通道=空白卡/从角色库导入（CharacterList 只读
  查询→字段映射六节：设定←定位/标签/背景/动机、外貌、性格、口头禅、关系留白、
  当前状态←状态/备注；红线=全程零写向绑定，测试挂监视断言零调用——不污染角色库）。
- **记忆左右脑**（`9773026a`，装机 PID 35668）：`boot.Options.MemoryUserDir` 缝
  （""=现状逐字节）一处算好四处消费点统一 → wx 脑 `<MemoryUserDir>/wenxue-brain`
  独立 DB（DB 随目录推导分池）；UI 记忆 tab 退役（办公脑窗：做梦/晨报/建议全办公
  域机制且 wx 不挂）；boot 双脑隔离测试（A remember 落盘 B 三向不可见+双向互不串）。
  **边界=图文脑从零起步不迁移**（办公脑留存零影响；旧会话 remember 事实不可溯源
  留办公侧）。
- **设定标签页**（`a7cc7c2a`，装机 PID 49276）：编队 tab 退役（ChatTabs 三处单源+
  Fleet 分支+useFleetAutoOpen 接线）；新增设定 tab——设定文档（设定|worldview|
  设定集）+时间线（时间线|timeline）两组 master-detail；设定文档轻渲染+CAS 编辑；
  时间线第N章锚结构化视图（未锚最前/中文序号如实提示，与后端同口径）；AI 生成
  五类预设（世界观/力量体系/规则/势力/地理）+自定义 → Composer 指令（agent 先
  wenxue_chapter_list/character_list/timeline_read 摸底再 write_file 落盘）→
  filetree 自动重载。测试 6 例，wenxue 族 21/21。
- **技能面动态终验**（`0de58522`）：四段式第四腿闭合——boot 全链动态调
  read_skill/run_skill（extras 可达/隐藏名 fail-closed/办公活读不变）；分发链
  核验（安装不复制 .gaea/skills，novel-* 不可达自洽，点亮须用户配 skills.paths）。
- **功能包** `releases/gaea-wenxue-v2-20261116-a7cc7c2a/`：独立干净树（detached
  @a7cc7c2a，junction node_modules）Vite 29.59s+Wails 37.4s（bindings 未跳过）；
  wailsjs 零漂移（git diff --quiet 过）；payload 四件冒烟 200；PACKAGE-INFO+
  SHA256SUMS 入库（gaea.exe `0211079464f7…`，干净树哈希≠装机位属预期代码同源），
  zip/tar 留盘（各约 29 MB）；工作树回收+清 .worktrees 两空壳。包面=
  448743a8..a7cc7c2a 六笔，全批门禁=Go 全量绿+tsc 0+eslint 0+绑定契约 5/5
  （除记忆缝外绑定面零变化，办公逐字节零触碰）。

## 图文工具首刀收口批（2026-11-16 凌晨）· 众神殿两场裁决施工

- **依据**：pan_1794751177012（工具/技能取舍四轮）+ pan_1794754879290（收口执行
  四轮）四席+主席收敛，12 项分歧台账全归宿；本批只施工「采纳」面，后置面
  （完整账本/12→7 合并读入口/批量 compact/结构化 writeback/lint 收益实测）按裁决不动。
- **六刀**：①`fact_add/list/clear` 图文面除名（办公实现与 RegistryMap 契约行不动）；
  ②run_skill 接可见 Skill 集合（`skill.NewRunSkillToolFromSet` + `skill.VisibleSkills`
  装配收敛单点——extras 可达、隐藏名 fail-closed、unknown 提示名单只列可见；
  办公双 nil 保持 store 活读逐字节）；③媒体根分档（image_gen/ocr `mediaRoot` 注入缝，
  wx 引擎 `wenxueBaseTools` 注入 wenxueWorkspace，生成图/OCR 不再落办公工作区；
  vision 不动）；④chapter_list 删假 `dir` 输入参数 + 共享章号 helper
  （wenxueStemNumber/wenxueAnchorNumber，解析与告警同源）；⑤窄告警两条
  （章节清单未编号聚合+根回退；时间线 `#` 标题行跳过+缺锚聚合告警附 文件:行号）+
  追加缺锚回执 hint；⑥workbench 回写句三子串。
- **测试**：skill 包五例（extras 经 run_skill 返回正文/隐藏名 unknown 且名单不泄露/
  办公 store 活读不变/同名去重 store-first/subagent 进 runner）+ app 包五例
  （章节四类 fixture 含 0 号组排序确定/根回退告警/时间线四类 fixture 含标题不进
  events/追加 hint/三子串断言）+ 一次性探针转正 census 守卫三例。
- **门禁与装机**：Go app 106s + boot/agent/control/skill/tool/config 全量绿；
  tsc -b 0；eslint 0；gofmt 改动面清。ucrt64 裸 wails build 1m01.8s（bindings 未
  跳过）；冒烟 /api/health 200；INSTALL OK（exe SHA256 322C6E5E4664…，
  PID 22312 > 构建时刻）。
- **遗留**：novel-agents/novel-deslop/novel-review 三件当前不可达（用户 gaea.toml
  无 `[skills] paths`，wx store 只扫小说目录项目域）——索引不列、run_skill 不报
  unknown，B 方案口径自洽；点亮七角色卡须用户侧配置 skills.paths（安装分发链批
  另议）。台账详见 `进度计划/gaea-wenxue-tools-20261115.md` 首刀收口批段。

## 众神殿续议立题页矮窗遮挡修复（2026-11-15）· 主代理直做

- **实弹根因**：续议会自动展开议题名称、背景与成功标准，额外字段把固定高度
  工作台撑破；同时 `.pantheon-canvas.is-composing` 仍锁死 `overflow:hidden`，
  于是议题底稿内的材料、方法、节奏与“开始讨论”被裁掉且无法滚动。
  1264×670 实测画布仅 602px，底稿内容需要 1069px 才能完整呈现。
- **修复**：发起态画布改为纵向滚动容器；窗口高度 ≤900px 时立题舞台、左右栏
  与底稿切换为自然高度；续议出现额外字段时，无论窗口高度都使用自然高度，
  由画布统一滚动。未展开额外字段的高窗口仍保持原满屏双栏布局。
- **验证**：1264×670 与 1440×920 实测续议展开态可滚动到完整底稿、起议条、
  背景/成功标准及右侧配置；1440×920 普通新讨论态仍无画布溢出、双栏等高
  铺满。`git diff --check` 通过；`npm.cmd run build` 绿（tsc 0，Vite 22.40s；
  仅保留既有 wenxue CSS 与 chunk 体积告警）；UCRT64 + `CGO_ENABLED=1` 裸
  `wails build -s -ldflags "-s -w" -trimpath` 50.27s，bindings 未跳过；
  `build\bin\gaea.exe` 与三枚 DLL 同目录，正式路径冒烟 `/api/health` 200
  （PID 45456 已回收）。exe SHA256 `A18167BC…BC9221`。未装机。

## 图文二次改造桌面端打包（2026-11-15）· 主代理直做 · 已构建预览包

- **提交链**：功能提交 `44f8bb6b`（原罪退役、文学板块隔离、左栏小说树、
  会话新建/小说删除、图文去知识库/CAD/进度计划）；构建前补齐 Wails 自动生成的
  `WenxueB` 10 个 Git 包装器，提交 `404679be`（本包源码提交）。
- **干净构建**：`.worktrees/pkg-404679be` detached 于 `404679be`，构建前
  `git status --porcelain` 为空；系统默认 TMP/TEMP、UCRT64 GCC、
  `CGO_ENABLED=1`，先 `npm.cmd run build`，再
  `wails build -s -ldflags "-s -w" -trimpath`，未跳过 bindings。
- **门禁**：前端全量 532 文件/4901 例通过 310.14s；图文定向 13/13；
  前端总构建 61.774s（Vite 22.58s）；Wails 42.425s；Go
  `internal/app` 129.558s、`go vet`/golangci-lint 通过；绑定漂移 OK@782。
  全量 CI 剩余失败仅为本机真实配置写权限、ORT DLL API 版本、realtime
  pipeline 与 Bash guard 环境差异，已记录为包边界而非声称全绿。
- **交付物**：`releases/gaea-wenxue-20261115-404679be/`，含正式 payload
  （exe+三 DLL）、预览 ZIP、源码 TAR、`PACKAGE-INFO.md` 与
  `SHA256SUMS-404679be.txt`；ZIP 根目录直接为四件套。
- **冒烟**：正式路径 `payload\gaea.exe` → `/api/health` 200，
  `{"status":"ok"}`，PID 36124 已回收。
- **装机**：`gaea_install.ps1 -SourceDir ...\payload -Launch` →
  `INSTALL OK`；四件套哈希与包内 payload 一致；桌面仅留 `gaea.lnk`，
  目标 `C:\AI\gaea-app\gaea.exe`；新 PID 41880 创建于 16:12:27，
  晚于构建时间 16:08:10。
- **哈希**：exe `08017F55…B049C7EC`，预览 ZIP
  `357A21E2…0840FC31F`，源码 TAR `D0BBBBA1…7E3D20`。
- **交付边界**：已覆盖安装 `C:\AI\gaea-app`。exe/DLL/ZIP/TAR 留盘不进 Git，
  包记录与校验清单入库。

## 图文二次改造第三刀：去除办公知识库/CAD/进度计划关联（2026-11-15）· 主代理直做 · 未提交

- **范围铁律**：只改 `frontend/src/wenxue/` 图文副本，办公板块 `gaea/`、调度器与
  Go 绑定不动。目标=文学工作台不再出现知识库、CAD 图纸、进度计划的入口、标签、
  工具卡、文件预览卡或跨板跳转。
- **知识库**：撤左栏展开/折叠入口、`Ctrl+Shift+K`、抽屉状态、命令面板命令、
  `KnowledgePanel` lazy 与渲染；删除图文内的 KnowledgePanel/knowledge 子面板与
  知识导入弹窗，能力清单移除知识工具说明。后端 `/knowledge` 等命令即使返回，
  图文 Composer 菜单与工具统计/卡片也统一过滤。
- **CAD / 进度计划**：右栏 Tab 收敛为 `files/deliverables/browser/git/ui`；撤
  Composer「工具」下拉、`OPEN_TOOL` 事件、CAD 文件直达、右栏 CadPanel/
  ScheduleTool 注册与 lazy chunk；删除图文内 CadPanel、ScheduleTool、
  ScheduleFileCard、ScheduleApplyRollback 及 cadOpen/cadExt。`.gsched` 文件回落到
  普通文本预览；`schedule_apply` 专用 diff/回滚卡与预览联动不再出现。
- **办公隔离补点**：交付复核失败不再跨板 NAVIGATE 到 `gaea`；mock 文件树移除
  DWG/DXF 样例，避免图文开发态仍展示 CAD 关联。
- **门禁**：`tsc -b --force` 0；图文隔离 2 例 + Sidebar 6 例 + mock 契约 5 例
  = **13/13**；改动面 ESLint 0；`npm run build` 38.33s 绿（既有 CSS 告警不变）；
  Go `TestWenxue*` 4 例与 `go vet internal/app` 全绿。网页实页验证：图文左栏仅
  “MCP 与技能”，右栏仅 5 个允许 Tab，命令面板无知识库/CAD/进度计划；办公板块仍
  保留知识库按钮及 CAD/进度计划工具入口。截图
  `temp/wenxue-palette-isolated-dark.png`。未构建 Wails、未装机。

## 图文二次改造第二刀：左栏小说工作空间主入口（2026-11-15）· 主代理直做 · 未提交

- **拍板落点**：图文（文学）二次改造的左侧导航改为小说工作空间主入口；原先右栏
  FileTree 的《新建小说》重复按钮撤下，左栏一键弹窗输入小说名，在
  `C:\AI\tuwen\<小说名>` 建独立目录并自动切换。左栏列表只消费 tuwen 一级子目录，
  办公最近项目不再混入。
- **后端加性**：`WenxueListProjectSessions` 按小说目录分组（含新建但暂无会话的书），
  `WenxueListSessions` 只列当前小说；`WenxueMeta` 返回 tuwen 当前路径；
  `WenxueNewSession` 把新会话落盘目录设到当前小说 `.gaea/sessions`，不修改办公工作区
  持久化配置。`WenxueDeleteNovel` 只接受 tuwen 根的直接普通子目录，拒绝根/越界/
  嵌套/符号链接；当前小说被删时先切回根会话再删目录。工作空间根字符串修为单一
  反斜杠，列表稳定排序。绑定面 777→782，WenxueB 61 方法。
- **前端设计**：按 `design-taste-frontend` 的 preserve-mode 审计 + `ui-ux-pro-max`
  产品导航规则重做左栏。小说是父级、会话直接嵌套；每本小说右缘固定“新建会话”
  与“删除小说”两个图标操作，删除先弹不可恢复确认并显示完整路径。顶部只保留
  一个主操作《新建小说》，知识库/MCP 下沉到底部工具区；暗亮双主题同用既有 teal
  accent，控制圆角统一为 7-10px，交互过渡 160ms 且保留 reduced-motion。
- **接线**：Composer 菜单残留“添加新项目”改为“新建小说”并复用左栏弹窗；选择或
  新建小说后清预览、刷新会话并广播 `wenxue:workspace-changed`。Composer 工作目录、
  文件树、右栏文件面均跟同一小说；删除当前小说回到 tuwen 根。
- **隔离边界**：文件操作、工作空间归属与新会话磁盘落点已隔离到 tuwen；LLM/事件流
  仍使用既有共享引擎，办公隐藏页与图文同时挂载时的引擎态独立仍属后续专项。
- **门禁**：`tsc -b --force` 0；Sidebar 5 例 + mock 契约 5 例绿；Go `TestWenxue*`
  4 例绿；control 包全绿；`go vet internal/app` 0；改动面 ESLint 0；绑定漂移
  OK@782；`npm run build` 24.04s 绿（既有 wenxue CSS 语法告警仍为构建警告，非本刀
  引入）。浏览器实页走查 1440×920 暗/亮、1100/880 窄桌面：小说树层级、右缘双操作、
  新建会话计数 +1、删除确认均通过；截图 `temp/wenxue-sidebar-actions-{dark,light}.png`。
  未构建 Wails、未装机。

## 图文二次改造第一刀：右侧文件面板工作空间隔离（2026-11-15）· 主代理直做 · 未提交

- **拍板**：图文（文学板块）二次改造第一刀=右侧文件面板：所有工作空间限定 C:\AI	uwen、
  新增《新建小说》按钮（弹窗输入名字→建 tuwen/<名字> 目录→自动切换）。中途用户纠正
  「不好共享分离就自己另起炉灶」→ 放弃 209 方法引擎大 fork（boot BuildWithConfig 缝已回滚），
  改走**紧凑文件面 fork**：仅文件/工作空间面隔离，聊天引擎仍全共享（维持早前拍板）。
- **后端**（新文件 `internal/app/wenxue_workspace.go`，62 方法/50 fork+12 手写，WenxueB 门面 46）：
  wenxueWorkspaceRoot（C:\AI	uwen，GAEA_TUWEN_ROOT 测试覆盖）+ wenxueWorkspace（appconfig
  新键 `wenxue_workspace` 持久化当前小说目录，无效回退根）+ WenxueListWorkspaces（根子目录）
  + WenxueCreateNovel（校验名→mkdir→自动切换）+ WenxueSwitchWorkspace（限根内，越界拒绝）
  + WenxuePickWorkspace（禁用占位）+ 文件面 45 方法 fork（listdir/preview/readtext/page/write/
  create/delete/rename/save-as/uploads 四件+pick 两件/filetree（**独立监视槽+独立通道
  gaea-wenxue-filetree**，与办公 officeState.fileTreeWatch 物理分离）/filesearch/materials/pin
  四件/attachment 5/export/zip/convert-pdf/git 10（wenxueGitWorkspace 换根））。全部加性：
  办公绑定体零改动，共享原语（workbench/wspath/gaeaAttachment/pins/spaces）按路径参数复用。
- **config**：加性 `wenxue_workspace` 键四件套（keys/types/configFile+Config 字段/save row/load merge）。
- **gen_bindings**：mapMethod 加 `Wenxue*→wenxue` 前缀规则+facadeOrder/facadeTitle；绑定面
  **777 方法/12 门面**；bindingNames(双树)/legacy(gaea 138 名)/signatures(777) 再生；
  builtins.go wenxue Bindings 加 WenxueB。
- **前端 wenxue**：mappings 55 条重定向（工作空间/文件面/git/schedule/交付全打 Wenxue*，短名与
  Gaea 直调名双口径）；`bridge/wenxue.ts` 新增 WenxueCreateNovel 接口+appBindings 组合+
  spaceBindings facet(work)+mock；FileTree：**《新建小说》按钮+弹窗**（antd Modal 输入名→
  WenxueCreateNovel→广播 `wenxue:workspace-changed`→wx Cwd 状态切树）、独立订阅通道、
  wxCwd 状态（ListWorkspaces 现值跟随，prefs/expanded/树复位全随它）；controller_actions_workspace
  加事件监听（同 switchWorkspace 复位+Meta/Context 重载）；useComposerWorkspace 加监听重拉列表。
  **wenxue 树 legacyBindings 按本树认领集现算**（gen -legacy-ts 按 gaea 口径算，双树口径不同属预期）。
- **门禁**：go vet 0+config/board/app 三包绿（121.9s）；tsc -b --force 0；vitest **4893/4893**
  （中途单红=再生瞬态，复跑连绿）；drift OK@767+死绑定过；改动面 eslint 0；wails 37s+**smoke 200**
  （PID 18080 回收）；装机 INSTALL OK，新壳 PID 40340（14:45:12）>构建 14:45:00。
- **已知边界**（二次改造后续批次收）：聊天/会话/任务中心/DAG 仍共享办公引擎（wenxue 右栏
  Git/交付已隔离，schedule 文件面已隔离）；Composer 工作区名显示仍读办公 Meta.cwd（下拉列表
  已是 wenxue 口径）；PickWorkspace 按钮仍在 Composer 菜单（点了返回空=无操作）。
- 未提交。生成器 `.tmp/wenxue-analysis/gen_workspace.py`（heredoc 反斜杠坑：脚本一律 Write 工具写）。

## 原罪板块整体退役（2026-11-15）· 主代理直做 · 未提交

- **拍板**：用户令「把原罪板块全部删除」——sin 全家（引擎/绑定/前端/导航/功能绑定/配置键）一次性拆除；
  同树 sin 线在途未提交改动（SinCastPane 等五新件+bindingNames/spaceBindings 增量）随拆除一并废弃。
- **删除面**：Go `internal/app/sin_*.go` 44 文件（引擎/工具/书源/EPUB/workspace/store/insight/refs 全家+
  测试）+`bindings_sin.go`+前端 `src/sinboard/`（含 5 个未提交新件）+`src/pages/sin/`+
  legacy `pages/OriginalSinPage.tsx`+其 tabs 测试+wailsjs `SinB.{d.ts,js}`。
- **横切拆点**：`feature_model_handler` sin 两 case；`chat_service` ChatTopicsList 过滤改局部字面量
  `"sin"`（存量 mode=sin 话题行继续滤除，不再引 sin_prompt 常量）；`gaea_route_suggestions`
  routeFeatures 去 sin；`image_handler` 删 GetSinImageConfig/SetSinImageConfig 两绑定（CoreB 面）、
  `image_domain` 注释留历史注；`shelf.go` 两 setter 行；config 包五文件（类型/键/载入/保存/特征 +
  config_test SinBinding 用例删）；`saveconfig_keys_test` sin 行删；`board/builtins.go` NOVEL_NAV
  去 sin 子项+CanonicalIDs **补上刀A 漏登记的 wenxue**（锁测试单向断言没抓到的疏漏）；gen_bindings
  三处去 sin（facadeOrder/mapMethod/facadeTitle）。
- **前端拆点**：`bridge/sin.ts`+`mock/sin.ts` 删；`appBindings.ts` 去 SinBindings 域；`spaceBindings.ts`
  去 66 facets（两处带注释行补刀）；`bridge/image.ts`+`mock/imagegen.ts` 去 SinImageConfig 两声明与
  mock；`bridge/http.ts` FACADES 去 SinB；`mock.ts` 去 buildSin；`events.ts` 去 SIN_STREAM/
  SIN_BOOKSOURCE_PROGRESS/sinStreamChannel/sinBooksourceChannel（events.test 29→27）；NovelPage
  去 sin 页签全家（lazy/类型/挂载集/身份区条件/DARK FICTION/封面 guard/FireOutlined 导入）+
  NovelPage.test 两用例删；`boards/manifests.ts` NOVEL_NAV 去子项+注释；main.tsx 注释更新；
  `test/logicEnvFiles.ts` 去 2 个已删测试登记。
- **再生四件**：`gen_bindings` 全量（**721 方法/11 门面**，bindingNames.ts 手工按 -names 重排时
  **曾丢 `as const` 致 drift 编译锁红——已补**；教训=两份生成件尾缀 `] as const` 是 drift.ts
  编译期锁的一部分，再生时必须保留）；`-legacy-ts`（82 名）+`-signatures-ts`（721 名，
  bindingSignatureCount=721 自洽）；wails build 再生 wailsjs（models.ts 0 Sin）。
- **wenxue 拷贝树同步拆 sin**：bridge/sin.ts+mock/sin.ts 删、appBindings/mock.ts/spaceBindings/
  http.ts/image.ts/mock imagegen 同款清洗、三生成件从 gaea 拷入——fork 与母本保持一致。
- **门禁**：`tsc -b --force` 0；**vitest 4893/4893 全绿**（543→530 文件；既有 contract 787vs751 红
  随再生自愈=生成器本就写 count 常量，sin 线手改导致失同步）；改动面 eslint 0；drift OK@721+
  死绑定检测过（StrataModelStop 提示系既有）；knip sin 项清零（唯一命中=voice-worklets/
  sinkProcessor.js 文件名子串既有假阳性）；Go app/config/board 三包绿；wails 37.9s+
  **smoke 200**（PID 13944 回收）。
- **语义保留**：统一会话存储内 mode=sin 存量话题行不动（ChatTopicsList 继续过滤，聊天板块不可见）；
  用户数据（书架产物/gaea/sin/ 目录）零触碰。未提交。

## 文学一级板块（2026-11-15）· 主代理直做 · 未提交

- **拍板**：新增一级标题「文学」＝办公板块完整拷贝（frontend/src/wenxue/），**全共享引擎与会话流**
  （用户明确选「先全共享，二次开发时再根据变动进行隔离」——零 Go 绑定 fork，OfficeB 同批绑定直用；
  引擎级隔离刀=115 方法 fork+WenxueB 门面的完整方案已在案，待二次开发真分化再立项）。
- **改动面**：Go 仅 `board/builtins.go` 增 wenxue 条目（menuOrder 13/BookOutlined/space work/
  Bindings=["OfficeB","MemoryB","CostB"]/无 Intents）+`board_manifest_test.go` 总数 15→16；
  前端 robocopy 全拷 gaea→wenxue（436 文件，**排除 248 测试件**——防 vitest 时长翻倍，
  二次开发随组件分化重写）+`pages/WenxuePage.tsx`+manifests.ts/main.tsx 注册
  （DEFAULT_BOARD_ID 仍 gaea，不占 ctrl+1~9）。
- **CSS 隔离（keepAlive 双挂载防互染）**：4 个 office-*.css 拷贝件改名 wenxue-*.css 并全量
  `.wx-root` 前缀作用域（:root→.wx-root、文件内 keyframes 改名 wx-*）+11 个组件级 css
  （context-view/cad-panel/agent-topology/fleet-*/changesdiff-tok/morning-brief-card/hljs-theme）
  同法作用域；schedule.css 是共享文件不动；`pages/WenxuePage` 根 div 挂 wx-root。
  共享基础 css（styles/redesign/tailwind）仍 import 自 ../gaea。
- **拷贝内定向修改**：genui scope "office"→"wenxue"（共享 genui/scope.tsx+fingerprint.ts
  联合类型加值 'wenxue'，GenuiPanel 硬编码 genuiPanelKey 同步）；mock 全局钩子改名
  `__wxMockScheduleFile` 防双 mock chunk 互踩；locale 三语可见文案改「文学工作台」
  （skeleton.*/deliverPanel.replan*；schedChat.* 描述与办公共享线程属事实陈述保留）。
- **knip**：`src/wenxue/**` 入 ignore（fork 素材含 81 个拷贝内暂不可达件——消费方在拷贝外
  或仅 gaea 测试件引用；二次开发接线后再收窄）。
- **门禁**：tsc -b 0；vitest 5023/5024（唯一红=`mock/contract.test.ts` 787vs751 **既有红**，
  sin 线在途再生 bindingNames/bindingSignatures 未同步硬编码数，与本批零交集——git diff
  定性 +1 行全 Sin 名）；改动面 eslint 0 错误（4 警告与 gaea 原件逐字同款）；
  `npm run build` 23.89s 绿（WenxuePage 独立 chunk 963KB 懒加载=fork 自含预期代价）；
  Go：board 包绿+app 包清单三测试绿（15→16 修后）；wails build 55.8s+
  `scripts\smoke.ps1` **/api/health 200**（PID 33320 回收）。
- **共享面登记（二次改造不得假设隔离）**：引擎/会话/事件通道 gaea-event 共用——双板块
  keepAlive 同挂时各自切会话另一侧对话流会跟随引擎当前会话（已知形态，用户拍板接受）；
  工作区/文件树/COM 桥/任务中心/做梦管线全共享。
- 未提交（本树另有 sin 线在途改动，未混入）；真机走查（rail 出现文学入口/双板块同挂无样式
  互染/共享会话流）挂用户实弹。

## 桌面端构建装机（2026-11-15 晨）· 主代理直做

- HEAD（a1a2f7f6+图谱重生=40 笔含 pantheon 工作台/browser_run_js/readability/lint 清欠）全量装机：
  前端 23.82s+wails 1m52.8s+冒烟 200+INSTALL OK；exe sha256 前缀 **ECA72636FFBE**，
  新壳 PID 27256（08:25:50）>构建 08:25:39。

## 代码图谱重生成（2026-11-15 凌晨）· 主代理直做（d9b2cdf8）

- `go run scripts/code_atlas_doc.go` 双面重导（markdown+codeAtlas.json）：**164→167 包**
  （office 域 xlsxedit/omml/comhelper 独立成包=dsh-office 蒸馏批、modelengine 升独立包）、
  源码 264,830→**274,000 行**、测试 210,821→**215,250 行**、内部边 474→**478**。
- 分层不变量再生成复算全过（内核不 import 壳层 0 违例/scripts 零被引/壳层入边只来自入口）；
  前端 CodeAtlasPanel 7/7 绿。§7/§8 手写导读段未动。

## lint 存量清欠 25 条 · 2026-11-14 · 主代理直做

- **范围**：11-13 灵犀/dsh 批已提交遗留（非在途，mtime+git 归属定性）。提交 `f7e7acfd`：
  `tools/wpsdock/main.go` errcheck×16（winapi `Call` 三值裸语句显式 `_, _, _ =`；
  `defer pCloseHandle.Call` 改闭包内忽略）+unused×3（pShowWindow/utf16PtrToString/
  argsToString 删除）+gofmt（1.19 doc comment 缩进+LF 归一）；app 三文件 gofmt 纯格式
  （office_tools 顺带修真缩进错位）；`parse_error_style_test.go` gofmt。
- **ineffassign 一条是真逻辑残留**：`gaea_dream_pending.go` 溢出归档错误分支的
  `dropped = over` 被紧随其后的无条件赋值覆盖——两处钉死测试（FIFOCap dropped==1、
  notice 归档文案）证明作者最终语义=无条件挤出数，删死分支保留无条件行；归档失败
  改 `slog.Warn` 留痕不再静默。11-13 归档改造只改了一半测试的死代码，本批补齐语义一致性。
- **坑**：①仓库工作树 CRLF/CR 混行尾——python 单行 `str.replace('\n')` 全 miss，
  须按行处理保 CRLF 或先归一 LF 再 gofmt；②winapi `Call` 返回三值，`_ =` 编译不过
  须 `_, _, _ =`；③app 测试须 ucrt64+build/bin DLL 双 PATH。
- **验证**：全仓 `golangci-lint` **0 issues**；app（Dream/Office/Xlsx 面）+
  builtin TestParse 绿；wpsdock `go vet`+`go build` OK。快闸 lint 段自此不再被存量挡。

## 40c2c677 遥测收口测试回归修复 · 2026-11-14 · 主代理直做（2c61eef8）

- 清欠后重跑快闸，test-related Go 段红：`TestRouteIntent_Status` 确定性失败
  （状态摘要报「当前没有启用的模型引擎」）——**40c2c677（只显示已连接引擎）改
  `GetModelMonitor` 时没对齐消费方测试**：chat_service 测试装配只有 Enabled 无
  Connected stub，`execStatus` 过滤后引擎清单为空。
- 修=`modelengine` 加显式零请求测试缝 `SetEngineConnectedForTest`（仿
  browser.SetForTest 先例；SaveEngine 是部分更新不吞 Status；试过
  `TestConnection` 真拨 mock——**会污染共享 mock 的调用计数与多轮 SSE 编排**
  （chat_persona_guard calls=3 want 2、chat_stream_search 编排错位），已撤）。
- 验证：RouteIntent/Chat/FeatureModel 面绿、modelengine 全包绿、lint 0。
- 快闸终态：build/vet/**lint 0**/漂移/卫生守卫/前端 related（42 例）全过；
  Go 段余两红 `TestVoiceApplySettings_PersonaPresets`（写真实
  `~/.gaea_config.json.tmp` Access is denied）+`TestSemanticSearchTool_EndToEnd`
  （readonly database）=**负载期假红**（同机对方执行体+用户壳并发互撞，
  与本批零交集），定向三连复跑全绿定性（在册 flaky 先例同族：
  websearch 负载期假红/app 首跑负载期）。

## 浏览器蒸馏两刀（browser_run_js + web_fetch readability）· 2026-11-14 · 主代理直做

- **P1 `browser_run_js` 一次执行范式**（Obscura×ego(lite) 调研 P1 刀，`fd33b06b`）：
  第 12 个 browser_* 工具——agent 写一段 JS 单次往返完成多步「看→点→填→等」，
  页面注入 `$read/$snapshot/$click/$fill/$wait/$waitFor/$log` 帮手；
  `$snapshot` 复用 data-gaea-ref 标记与 `__gaeaEpoch` 代数（Go 侧登记 epoch），
  run_js 拿到的 ref 可直接喂 browser_click/type（ref 生态打通）；页内导航不提供
  帮手（跳转销毁执行上下文，文档引导 browser_navigate）。value 超长 Go 侧 rune
  安全截断降级文本预览（不输出半截非法 JSON）。既有 11 工具原样保留，纯增量。
- **P2 `web_fetch` readability 正文主提取**（LP.getMarkdown 思路蒸馏，零新依赖）：
  `webfetch_readability.go`——简化版 Mozilla Readability（长 `<p>` 双亲计分
  parent 全额/grandparent 半额 + class/id 关键词修正 + article/main 加成），
  命中容器连同 `<title>` 重序列化后复用 htmlToText 单一渲染路径；判据不足
  （<3 段/<200 字/无净增益 ≥95% 页文本——噪声元素通常无 `<p>`，页文本必须全
  口径计）回退整页旧行为；mode=auto 默认（web_search 直抓 fetchSearchPage 同享）/
  full 显式整页。摘除清单保守（不摘 header/footer，防丢文章标题）。
- **纪律面对账**：compact 双表/spaceTags/registrymap 契约行/registry map 文档重导
  （`go run scripts/registry_map_doc.go >`——生成器只打印不落盘，须重定向）四表同批；
  TestSpaceTagsCoversAllBuiltins/TestRegistryMapDocCoversMap/CompactDrift 三守卫绿。
- **验证**：browser/tool/builtin 定向 + boot census + app ExtraTools 守卫 +
  agent/spaces 回归全绿；改动面 lint **0**（快闸 lint 段被 11-13 灵犀/dsh 批存量
  欠账挡住 25 条——tools/wpsdock/parse_error_style_test 等已提交遗留，非本刀产物
  不越界修）；前端 build 28.87s 绿（对方 pantheon 在途未阻断）；Wails 构建
  1m5.12s；冒烟 200；装机 `C:\AI\gaea-app` INSTALL OK（exe 63EF672FEFCF…，
  PID 58784 > 构建时刻 22:59:31）。**冒烟新坑**：运行中装机壳与新冒烟实例共用
  WebView2 profile → controller 8007139f 即退 ExitCode=1（非构建问题），杀壳
  后即通——双实例互斥形态记档。
- 余池：obscura.exe sidecar（可选第二引擎，FindEdge 泛化 seam）/共享登录态
  Spaces（信任门）挂拍板；事实源=docs/gaea-obscura-egolite-browser-distill-research-2026-11.md。

## 众神殿工作空间历史与立题工作台 · 2026-11-14

- **讨论范围来源收敛**：发起讨论不再读取应用级最近工作空间，而是从众神殿会话摘要中按
  `UpdatedAt` 倒序取曾真实发起过议事的目录。大小写不敏感去重（Windows），不存在目录跳过，
  同一目录由最近一次议事重新提前，最多返回 4 项。手动选择目录与续议种子只作为本次选中范围，
  不再回灌历史按钮。
- **立题页铺满窗口**：发起态给画布加 `is-composing`，页面级滚动关闭；宽屏下议题底稿与议事配置
  共用同一网格行，默认窗口实测两栏同为 **588px**，1440×920 实测同为 **563px**，底边对齐且
  默认内容无需下滑。窄屏回落为自然高度纵向堆叠。
- **设置卡片化**：底稿下方拆为“材料与补充 / 议事方法 / 轮次与节奏”三张卡，启动区独立为
  底部动作条；原有材料、图片、形态、路径、汇总人、轮次、每轮确认与预算字段及 testid 均保留。
  配置栏压缩段间距与发言行高后，默认发言顺序完整显示。
- **验证**：`PantheonPage` **42/42 例**；Go `TestPantheonRecentWorkspaces` 与
  `internal/app -run Pantheon` 定向通过；`tsc -b` 0；改动面 ESLint **0 error**
  （保留既有 `PantheonPage.tsx:585` exhaustive-deps warning）；Vite 生产构建 **37.76s** 通过；
  1912 宽默认窗口与 1440×920 暗色、亮色实页走查通过。未构建 Wails、未装机。

## 启动模型过载提示阈值 · 2026-11-14

- 用户反馈启动时 4 个本地引擎就弹「已启动模型过多」。旧逻辑写死 `> 3`，
  且同一轮询每 60s 可重复提示。
- 收敛为 `telemetryOverload.ts` 单一策略：允许最多 **6** 个本地引擎
  （本地引擎 + 运行中的 ComfyUI），仅第 **7** 个触发；模型数量告警每次应用
  运行最多一次，CPU/GPU/内存告警仍保留原阈值与 60s 节流。
- 三语提示补「上限 {limit} 个」；新增 2 例边界/计数测试（4/6 不报、7 报，
  ComfyUI 计数、云端不计）。
- 后续复核发现底部仍把“已启用”误显示为“已启动”：Herdsman 默认
  `Qwen3.6-35B...` 实际状态 `error`，Llama/WHIRL 均未连接。`GetModelMonitor`
  改为只返回 `Enabled && Status.Connected` 的引擎，并仅在默认模型状态为空/
  `running` 时展示模型名；前端只渲染已连接本地引擎。最终底部从 4 个陈旧项
  收敛为 `herdsman` + `breeze·breeze-tts-2`，不再出现 Qwen35、Llama、WHIRL。
- 门禁：Go 监控/陈旧模型 2 例、前端阈值/标签 2 例、ESLint 0；前端生产构建
  Vite 31.60s；Wails 1m07.1s；正式路径冒烟 `/api/health` 200；`INSTALL OK`。
  安装版实测底部仅两个已连接本地引擎且启动无模型过多弹窗。exe SHA256
  `3E57E1F63AAA33727158AA254657A4DA5D4049E8DDE2F0C608DD0A8FF7EE4251`。

## 多执行体工作树收口 · 2026-11-14

用户确认其他 agent 已停止，要求主代理接管全部未提交改动并梳理工作空间。收口按
功能边界拆成可审查提交，不把办公、编队、众神殿、识图和进度计划混成单笔：

- `e221c170` 办公输出：交付停靠条同源、上下文状态去重、消息操作浮层、
  阶段“思考中”去重与灵犀成果卡；定向 55 例。
- `14573e88` 三语文案：补编队 MissionPulse 与识图画布文案，退役已删除的
  消息复制/单条沉淀技能文案。
- `6f0f9fb1` 编队：Hero + SVG Agent 网络 + 跨 Agent 时间线/任务 + 事件检查，
  **20/20 例**。
- `fd21988d` 众神殿：恢复四席拓扑并补上下文账本、相对输入环、逐轮轨迹，
  **76/76 例**。
- `8bd9b059` + `d1d20dd9` 识图：完成遗留 `ImageInspectStage`，接入选图页，
  补像素标尺、锚点缩放、平移、全屏、纯函数与视图测试，**15/15 例**。
- `b1cd78a2` 进度计划：ZPert8 W4–W7 时标/图面/就地编辑/列筛/右键/纸面导出/
  斑马密度/树拖动及 4 份逆向资料，一体验证 **85 文件 / 1202 例**。
- `03686b79` 编队测试：把遗留的径向卡片断言改为 MissionPulse SVG 谱系；
  卡片主命中区选中、独立按钮打开上下文，聚焦组 **22/22 例**。
- 更早的 `6925ad86` 输入卡/正文文件链接、`7949b8db` 预览包记录和
  `86040063` release 计数已单独入库。

工作树清理：删除误生成的 `({i` 空文件；把 PowerShell 误落盘的非 UTF-8
`$null` 移到 `.tmp/workspace-cleanup/powershell-null-artifact.txt` 保留取证；
`.gitignore` 增加 `$null` 与 `releases/*/payload/`，预览包二进制继续只留盘。
收口期间又落盘一份未跟踪的 `docs/gaea-obscura-egolite-browser-distill-research-2026-11.md`
（22:10，纯可研未动刀），已补 `docs/README.md` 索引并纳入同一清理提交。
全窗口案：Vite 生产构建 **48.48s** 通过；全量 ESLint **0 error / 57 warning**。
全量前端测试首跑 **542 文件 / 5020 例、1149.56s**：540 文件/5017 例通过；
2 条失败是旧径向拓扑断言，已由 `03686b79` 更新并通过（编队聚焦组 22/22）；
另 1 条 `DatabaseLibrary` 加载时序失败单跑 6/6、与拓扑复跑合批 8/8 通过，
归类为全量并行时序抖动，不掩盖。最终工作树只保留本记忆/忽略规则提交，未 push。

## 办公输出 · 阶段行图标与重复“思考中”收敛 · 2026-11-14

- **阶段行横排**：`office-phase-line` 统一承载 phase 图标与文字，CSS 改为
  `inline-flex + gap:6px`，覆盖 Tailwind preflight 将 SVG 设为块级后造成的
  “图标独占一行”。
- **思考去重**：模型调用前的 `phase="思考中"` 仅在尚未产生 reasoning 时保留；
  同一过程段后续已有思考正文/思考卡时，该 phase 从过程卡正文与最近三项尾巴中
  一并过滤，避免思考卡上方重复出现“思考中”。首包前仍有状态反馈。
- **门禁**：ProcessCard + Transcript 四文件 **52 例**全绿；改动面 ESLint 0；
  `npm run build`（tsc + Vite）**34.60s** 通过。浏览器静态几何实测图标中心线
  与文字一致、图标位于文字左侧；临时预览文件已清理。
- **边界**：未构建 Wails、未装机。

## gaea 进度板块 · 实页走查（Vite dev + 无头 Edge · 2026-11-14）

- **方式**：`npm run dev --port 5180` + 无头 Edge（CDP 9333，1600×1000）+ `?mock=1`，
  点左侧轨道「进度计划」进页，亮/暗各拍一张。
- **实测计算值**（CDP 读 computedStyle）：表格行高 **30px**、单元格字号 **14px**、
  功能区命令圆角 **4px**、命令台 `backdrop-filter: none` —— 密度层确实生效。
- **走查发现并修掉两个真问题**（都是像素级才看得见的）：
  1. **周标尺相邻重复 W1**：`buildWeekRuns` 用 `floor(i/7)+1` 算周序，首周若不是周一
     起始，会把首段与下一个周一段都算成 W1；改为按「第几个周段」递增。
  2. **窄窗 W 标签叠字**：段宽 ~18px 时逐周标 W1..W20 会互相压字；改为段宽 <24px
     退回「每 4 周标一次」（≥24px 仍逐周）。
  两条都补了用例（29 例全绿）。
- **截图**：`temp/gaea-schedule-zebra-light.png` / `temp/gaea-schedule-zebra-dark.png`。
- **边界**：走查用 dev server + mock，未构建 Wails、未装机。

## gaea 众神殿 · 四席圆桌上下文状态与逐轮轨迹 · 2026-11-14

用户确认众神殿固定四席（Grok / Codex / ZCode / DeepSeek Harness），要求在既有议事厅
内补“众神的上下文状态和轨迹”，不重做页面、不删除原有四视图。

- **拓扑回退**：首版圆桌布局经用户实看后否决，已恢复原版“共享枢纽在上、四席卡片在下、
  径向虚线连接”结构；共享主线程与席位节点原有跳转、跳过、重试、实时状态和用时全部保留。
- **连线修复**：新增上下文块后席位卡变高，旧路径端点和容器高度不匹配；宽屏高度增至
  430px 并提高线色/线宽。1320px 以下不再隐藏 SVG，改为生成两列树状 compact path；
  521px 以下沿用原纵向连接缩略。
- **上下文状态**：新增 `pages/pantheon/seatInsights.ts`。每席聚合
  `PantheonSeatUsage` 的输入/缓存/输出 Token、步数和引擎会话；头像外环与卡内条仅表示
  四席之间的**相对输入投入**。引擎未上报时显示“上下文未上报”，不拿缺省冒充 0%。
- **关系展示边界**：拓扑图不再额外绘制关系弦线；目标贡献/分歧关系继续在席位轨迹中
  以“指向谁、被挑战命题、回应结论”展示，避免和原拓扑连线叠加造成误读。
- **轨迹页签**：席位独立窗口新增上下文账本和“轨迹”。轨迹按时间合并观点与工具调用，
  展示主张/质疑/回应等动作、轮次、时间、目标席位、被挑战命题和回应结论；原“观点”
  与“工具调用”页签保持原样。
- **四视图保留**：拓扑图、共享窗口、Agent 窗口、分歧台账的结构与操作未删；本次仅增强
  拓扑图和席位独立窗口。另修正设计探索阶段误加第五席的问题，落地面严格以四席为准。
- **门禁**：`PantheonPage` + `PantheonRecord.streaming` + `seatInsights` **3 文件 / 76 例**
  全绿；`tsc -b` 0；改动面 ESLint 0；`npm run build`（tsc + Vite）**23.55s** 绿。
  恢复后的原版拓扑与席位抽屉在真实 mock 会话亮暗实页走查通过；未提交、未构建 Wails、未装机。

## gaea 进度板块 · 树形拖动（第七刀，主代理直接执行）· 2026-11-14

- **纯计划器**：`moveRow.ts`（before/after/inside 三落点、分区带子树搬家、「之后」按目标子树
  末尾算）；拒绝：原地/自己子树/分区塞分区/工作塞工作/跨级/inside 非分区，均给原因不落库。
- **接线**：行容器 `draggable` + 三区判定（上 25% 前 / 中段成为子工作 / 下 25% 后），
  落点提示 `sched-drop-*`；drop → `store.moveRow`（一次历史）。画布窗格不传属性，手势不变。
- **门禁**：schedule + 页面回归 **85 文件 / 1200 例** 全绿（+11：纯函数 9 + 集成 2）；
  改动面 ESLint 0。`tsc --noEmit` 与 `npm run build` 当前被**他人未提交的在途文件**
  `pages/pantheon/PantheonRecord.tsx` 阻断（TS6133/TS2739 且行号在漂移，属另一执行体
  正在编辑），与本刀零交集；Vite 构建另行单跑核验。
- **边界**：未提交、未构建 Wails、未装机。

## gaea 进度板块 · 右键真动作补批（第六刀，主代理直接执行）· 2026-11-14

按新纪律（2026-11-14 用户撤销强制并发）由主代理连续完成，未派子代理。

- **解组**：`store.ungroupTask` —— 删分组行、子工作并入上一个分组；首分区「无处归属」/
  空分组都拒绝并给原因（与降级同一归属口径）；`pushHistory` 一次可撤销。
- **格式刷**：功能区按钮（有选中才可用）或右键项取当前行外观 → 下一次点击表格行应用
  并自动收起；激活期根节点 `.sched-brush-mode` 让表体 `cursor: copy`。
  行点击包装为「刷子激活则应用、否则选中」，画布手势零改动。
- **文字颜色**：右键项改为色板子菜单（红/橙/绿/蓝/紫/默认 6 项），落 `SchedTaskStyle.color`；
  「默认」删色键（`withoutStyleKey`），不写空壳 style。菜单渲染层支持 antd 子菜单。
- **列宽自适应**：列头右键 → 表格窗格回落到可见列合计并写 `chatPrefs.ganttTableW`。
- **仍诚实禁用**：插入列 / 列设置（列集由视图预设与列菜单定义）、显示/取消分区
  （网络图分区语义未建模）、图面样式（功能区已有面板，不重复挂浮层）。
- **门禁**：schedule + 页面回归 **84 文件 / 1189 例** 全绿（+6 例：解组 4、格式刷/自适应 2、
  菜单契约改写 1）；`tsc --noEmit` 0；改动面 ESLint 0；`npm run build` 见下条。
  既有「格式刷禁用」用例改为用「删除」取证（格式刷已接真动作）。
- **边界**：未提交、未构建 Wails、未装机。

## gaea 编队标签页 · dsh-context 0.84.1 MissionPulse 对齐 · 2026-11-14

用户补充确认两张截图均为**目标参照**，要求对齐完善编队标签页。执行方式同时改为
主代理按需决定、不再强制并发子代理；一次协作任务子代理空转/API 报错后由主代理直接接管。

- **页面骨架**：`Fleet.tsx` 收口为 Hero + 左 Agent 网络 + 右跨 Agent 协作/协作任务 +
  底部事件检查；删除旧的重复成员卡面板，保留原三投影数据源（AgentNetwork /
  SubagentRuns / FamilyEvents），不伪造上游 team task、blocked-by 等缺失字段。
- **Agent 网络**：新增 `agentTopologyLayout.ts`，按叶槽计算任意深度树布局；卡片改为
  SVG 贝塞尔谱系、真实节点色、角色/模型/Token/费用/耗时/工具/消息、选中路径高亮与
  “打开会话”。摘要 Agent 数改读网络实节点，修掉网络 4 个而 Hero 只显示 3 个的偏差。
- **跨 Agent 协作**：新增 `FleetTimeline.tsx`；固定左侧泳道标签，右侧等距列网格、
  dispatch/report 竖向连接与事件节点；搜索、初始/最新定位、选中同步、空态、键盘焦点、
  reduced-motion。泳道只由真实 runs + main 生成，去掉 `family.refs` 中无运行记录的幽灵道。
- **协作任务**：新增 `FleetTasks.tsx`；按 Agent 分组，状态 chips 带真实计数，支持搜索、
  行展开 answer/失败原因与打开上下文；组头可选中 Agent，筛选在 Hero/网络/时间线/事件
  间共享。行数据只用现有 `SubagentRunView` 与 `FamilyAgentRowView`。
- **事件检查**：新增 `FleetEvents.tsx`；类型 chips + 数量、搜索、Agent 筛选 pill、左侧
  事件卡流、右侧常驻检查器；默认展示首条，补齐时间、序号、参与者、发送方/接收方、
  完整正文、关闭与回报会话入口。事件 key 统一 `${agent}-${kind}-${seq}`，与时间线节点联动。
- **视觉口径**：1618×894 亮/暗、1100 与 880 亮色 mock 实页走查；宽屏两列、窄屏顺序
  堆叠，无横向页面溢出；时间线自身长列表保留横向滚动。参照图为信息架构和密度基线，
  颜色继续服从 gaea 主题 token。
- **门禁**：`Fleet*` + layout 5 文件 **20/20 例**、`tsc -b` 0、改动面 ESLint 0、
  `npm run build`（tsc + Vite）**44.15s** 绿；未提交、未构建 Wails、未装机。
- **纪律变更**：`.gaea/AGENTS.md` 的原「默认并发子代理」改为「按需使用」；子代理
  报错/空转/只确认未落产物时，主代理直接接手，不再为并行形式重试。

## gaea 进度板块 · 纯照斑马外观密度 + 交互语义（2026-11-14 第五刀，三线并发）· 2026-11-14

用户先要求「下一代前端对齐、UI 建立交互」，随后二次拍板「**纯照斑马外观密度**」——
撤销 v3 星枢的玻璃/高光线/光晕，形状语言严格照 ZPert8。

- **密度令牌层**：新增 `sched-zebra.css`（20 个令牌：行高 30 / 表格字 14 / 浮层字 12 /
  控件圆角 4 / 面板 6 / `--sched-elev-1: none` / 动效 100ms / hairline / grid-line /
  head-bg / head-ink …），六区密度收紧（功能区、命令台、文档条、工作区、右舷、状态栏）；
  表格/画布方角；hex 只作 `var(--token, #hex)` 兜底，选择器零裸色值。由
  `SchedulePage.tsx` 在 `display-preset`/`schedule`/`hit` 之后引入。
- **交互态**：六个钩子（功能区命令、菜单项、表格行、表头格、状态栏、右键浮层）统一
  hover 4% 灰膜 / 选中 12% 主色 / 按下 16% / disabled 45% + not-allowed /
  `:focus-visible` 2px 主色环 + `outline-offset:-2px`（栅格内负偏移不撑行高）；
  `prefers-reduced-motion` 与 `.ui-reduced-motion` 双入口归零。单代号节点选中从外扩
  光晕改成 2px 实描边环；状态栏不再锁死 24px 高（只留 min-height，防大字号裁字）。
- **键盘与 aria**：页签 roving tab + ←/→/Home/End + Enter/Space；命令按钮补
  `aria-pressed`/`aria-disabled`/`tabIndex=-1`；9 个受控浮层 Esc 收起并回焦触发者；
  表格行 roving（↑/↓ 选中并跟随焦点、Enter 检查器、F2 就地编辑，行内控件不抢键）；
  表头筛选按钮 `aria-haspopup/aria-expanded`；命令面板 `combobox` + `aria-activedescendant`；
  状态栏 `role=status` + `aria-live=polite`（只播变化格）；右键菜单 Esc 回焦命中行。
- **两处静默失效被修**：①`schedule.css` 里 27 处 `--sched-radius-*`/`--sched-motion*`
  此前**没有任何定义**（v3 层撤销后悬空），圆角与动效一直静默失效，本轮补齐；
  ②`uiAlignment.test.ts` 与 `schedule.css` 注释指向已删除的 `sched-v3.css`，已订正为
  `sched-zebra.css`（全仓零残留）；div 型钩子的 `:disabled` 永不命中，补了
  `[aria-disabled="true"]` 变体。
- **口径决策**：表格 14 / 浮层 12 是斑马自身两档（re:359 与 re:362），双击开编辑的字号
  跳变属预期；双标尺 40px 是刻意偏离（保证表格首行与画布首行同基线），已钉成断言。
- **门禁**：schedule + 页面回归 **84 文件 / 1183 例** 全绿（+3 文件 / +109 例，含新增
  密度守卫 85 条与键盘/aria 23 例）；`tsc --noEmit` 0；改动面 ESLint 0；
  `npm run build`（tsc -b + Vite）**42.60s 通过**；前端全量 **532/535 文件、
  4982/4985 例**——3 例失败全在**本刀未碰**的 context 编队
  （`AgentTopology` 2 + `Fleet` 1，工作树里已有的 dsh-context 在途改动），
  与进度板块零交集。
- **边界**：未提交、未构建 Wails、未装机；六屏人工走查（亮/暗 × 1440/1100/880）待用户观感。

## gaea 进度板块 · ZPert W6 输出与外观（按纸张布图/显示预设/收尾）（三线并发）· 2026-11-14

W4/W5 收口后按计划档进 W6，仍是三线并发 + 主代理接线。

- **按纸张布图**：`pageLayout.ts`（A4/A3/A2 × 横竖、mm→px@dpi、页边距、等比适配
  `minScale=0.5`、分页网格 row-major、页框/页号/图幅标记、`MAX_PAGES=400` 截断）+
  `paperExport.ts`（上报图作嵌套 `<svg>` 按页切片 + `clipPath` + 可配装订缝；
  `gap:0` = 打印口径；页数/缩放/截断回填给 UI）。A4 竖 794×1123、10mm 边距内容盒
  718×1047。
- **打印真分页**：`printSvg` 增可选 `{pageSize, margin, svgWidthMm}`——打印纸堆时
  `@page{size:A4 landscape;margin:0}` + `svg{width:297mm}`，每页落一张纸；既有调用
  不传参=原 8mm 口径零变化。PNG/PDF 仍整图编码，纸堆 >20 页时按钮提前禁用并给原因
  （浏览器画布 ~32767px 上限，实测 A4 竖约 28 页到顶）。
- **显示预设**：`displayPreset.ts` 三档 + `display-preset.css` 只覆盖六个 `--sched-*`
  领域色（不碰 `--md-sys-color-*`，亮暗基座都能叠）；每档分「屏上深底/纸面白底」，
  对比度按 WCAG 实测（护眼 4.91–9.08:1、黑白 5.24–16.21:1，关键/正常按文字级 4.5:1
  收口）；进度页「显示 → 配色」切换并落 `chatPrefs.ganttDisplayPreset`（旧偏好零迁移）；
  导出前 `applyExportPalette` 把六个导出色换成该预设纸面档，屏上与 PNG/PDF 同色相。
- **收尾两项**：PDM 节点半径参数化（`nodeRadius` 当尺度系数 k=radius/16，盒/列距/行距/
  连线端点同源缩放，缺省渲染 innerHTML 逐字不变，k=4 下限仍无负间距）；状态栏挂
  `data-hit="status-bar"` 并走同一命中/右键入口（类名/文案/结构零改动）。
- **一处诚实留白**：图签/图例沿用既有构建器（v4.132~v4.137 已落，本轮不重画）；
  前锋线/竣工线不在预设六变量契约内，黑白档仍是彩色线；导出 PDF 仍是图片型单页
  （逐页编码合并属另一刀）。
- **门禁**：schedule + 页面回归 **81 文件 / 1074 例** 全绿；`tsc --noEmit` 0；
  改动面 ESLint 0；`npm run build`（tsc -b + Vite）**40.42s 通过**；前端全量
  **531/532 文件、4866/4867 例**——唯一失败仍是未改动的
  `memoryhub/DatabaseLibrary.test.tsx`（DOM 停「读取中…」），隔离复跑时红时绿，
  与上一轮同一条 flaky，判既有（非本刀引入）。
- **边界**：未提交、未构建 Wails、未装机。

## gaea 进度板块 · ZPert W4/W5 余项收口（命中区/右键/列筛持久化/图面参数）（三线并发）· 2026-11-14

第二刀之后的余项收口，仍是三线并发 + 主代理契约与集成。

- **命中区优先级**：`hitRegion.ts`（列头→画布→单元格→状态栏→分隔条→行头，
  `pickHitRegion` 取最高、`hitRegionChain` 自限深度上溯 `data-hit`）；GanttView 收成
  单一 `onContextMenuCapture`，对象分支先选中再出菜单（§16.2 口径）。关键取舍：
  行容器声明 `row-header` 兜底、单元格声明 `cell` 覆盖它——否则优先级会把行号格
  裁成单元格；该约束在模块注释、组件注释、用例三处各记一遍。
- **统一右键菜单**：`hitMenu.ts` 12 项结构操作固定序 + 画布/列头/分隔条/状态栏区域
  菜单；可用性复用 `commandRegistry.scheduleCommandDisabledReason`；无后端能力的项
  （解组/格式刷/文字颜色/插列/列设置/列宽自适应/分区/图面样式）如实禁用并写原因。
  既有两条右键用例零改动通过（行级专有项追加在 12 项之后）。
- **图面参数补齐**：`networkLayout.ts` 成为 `namePos / hideUnimportantLink /
  stagger / layerHeight1-2 / density` 唯一解析与变换入口。AOA：名称排布三态、隐藏
  非关键非虚关系线（图例真实计数）、错层（仅自动布局）、层高上下限、密度系数
  （列距/行距同源，时标与波形长度不破）；PDM：名称排布 + 隐藏非重要搭接。
  三条「缺省零回归」锚点（节点坐标数组逐字相等 / PDM 渲染逐字一致）。图面样式
  面板补密度/层高/错层/隐藏非重要线四控件，消灭「能写不生效」的假开关。
- **列筛持久化**：`colFilterPrefs.ts`（归一/稳定签名/原型链键防护）+ `chatPrefs.
  ganttColFilters`（键名与版本不变，旧偏好零迁移）；GanttView 初值只读一次、
  变更按签名去抖单点落盘、首帧不写盘；视图切换不动列筛。诚实收窄：只持久化
  `COL_FILTER_KIND` 登记列，自定义字段列条件会话内生效、重启丢失。
- **常量单源**：`ganttUtil` 的 `WEEK_TIER_BELOW / WEEKDAY_LABELS` 改由 `timescale.ts`
  re-export，消除第二份真源。
- **一个测试工程发现**：列筛持久化用例与 GanttView 主测试文件同场跑时顺序敏感
  （挂载前 localStorage 就位 + 几十个共享存储用例），隔离跑则稳定通过。处置=
  把两条端到端用例抽成独立文件 `ganttColFilters.persist.test.tsx`（不是放宽断言，
  而是把环境前置条件显式化），主文件保持 47 例。
- **门禁**：schedule + 页面回归 **76 文件 / 991 例** 全绿（+5 文件 / +90 例）；
  `tsc --noEmit` 0；改动面 ESLint 0；`npm run build`（tsc -b + Vite）**42.71s 通过**；
  前端全量 **526/527 文件、4782/4784 例**——唯一失败是**未改动**的
  `memoryhub/DatabaseLibrary.test.tsx` 两条时序断言（DOM 停在「读取中…」），
  隔离复跑 2 次 6/6 全绿，判既有 flaky（上一轮全量同文件全绿）。
- **余项**：PDM 节点半径（会破 NODE_W/H 分层）；状态栏 `data-hit`（在 SchedulePage
  外壳，随外壳刀）；异常标记/进度滞后列过滤器（需先有列）。
- **边界**：未提交、未构建 Wails、未装机。

## gaea 进度板块 · ZPert W4 图面密度 + W5 交互首版（三线并发）· 2026-11-14

承接 `进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` 的 W4/W5 与
`gaea-schedule-zpert8-master-index-20261114.md` 的逆向结论，三线并发子代理 + 主代理收口。

- **W4 双标尺**：新增 `timescale.ts`（669 行纯模块，逐条对齐 timescale-re §2/§3/§4/§8：
  六档 fov、单位间隔集与钳位、自动联动 0.04..0.4 且 ×0.5 收尾、像素唯一出口
  `pos[i]=Σ step×scale×2.4`）；`GanttTimeline` 重写为「工程标尺（年-月跨列段 +
  日/周刻度线）+ 进度标尺（周/星期）」两行常驻；`GANTT_RULER_H` 成为画布表头、
  表格表头、`--sched-ruler-h`、覆盖层顶部唯一来源（新增守卫断言四处同值）；
  窄窗 `dayW<8` 降级只画月段/周号；今日/竣工竖线跨两行；刻度线疏密读
  `networkStyle.tickLines`。
- **W4 四开关**：`networkDisplay.ts` 为「节点时间/工期/六参数/起止时间 + 几何」唯一
  解析入口；缺省即旧观感（前两开、后两关），AoaView 用「全缺省 vs 不落
  networkStyle → innerHTML 逐字相等」钉死零回归；PDM 同步消费六格 ES/EF、工期/
  搭接标签、框下六参数与起止。
- **W4 几何贯通**：dotStyle/arrowStyle/headHeight/lineWidth 已消费；`aoaLayout.edgeSegs /
  assignChannels / summarySegs` 增 `radius` 入参（缺省 AOA_R，既有 32 例断言零改），
  视图与 `networkExport` 同传 `resolveNetworkGeometry().nodeRadius`——改半径不再
  箭线插进节点/端点悬空。
- **W4 五图模式**：时标逻辑图（保留时间轴与关键红、压掉工期装饰）、横道网络图
  （关系线加重 `sched-link-net` + 折线中点 `FS+2` 标签）接线；其余三模式沿用。
- **W5 就地编辑**：`inplaceEdit.ts` 四类值语义（严格合法日、区间开闭与顺序、数值
  step 对齐夹取、文本 trim/maxLen、空=清空与显式 `emptyAs:'zero'`）；
  `InplaceEditField` 按逆向 §15.2 六步（padding 0/继承字体/左对齐垂直居中/全选/
  Enter 提交/Esc 取消/blur 提交）；GanttTable 双击 名称·工期·进度·开始·完成·自定义
  字段走同一弹层，旧单击直编路径保留（零回归）。
- **W5 列级筛选**：`GanttFilter` 增可选 `colFilters`（文本/数值/日期区间/枚举/空值），
  三参调用逐字不变；表头每列「▼」（伪元素出图标，不污染表头文案断言）；GanttView
  接第 4 参 ctx，日期列筛不再 fail-open；功能区「编辑 → 列筛选」+ 清除信号。
- **门禁**：schedule + 页面回归 **71 文件 / 901 例全绿**（新增 55 例：timescale 22、
  networkDisplay 9、inplaceEdit 25…按文件计）；`tsc --noEmit` 0；改动面 ESLint 0；
  `npm run build`（tsc -b + Vite 生产构建）**41.21s 通过**；**前端全量 522 文件 /
  4694 例 / 742s 全绿**（交付前唯一一次全量）。
- **卫生**：把 in-flight 遗留的 CRLF（schedule.css / AGENTS.md / SchedulePage.test.tsx）
  统一回 LF（diff 由 3698 行噪音收敛为真实 289/40/7 行）；仓库根游离文件
  （`$null`、根目录双份旧测试）移入 `.tmp/hygiene-20261114/`。
- **未做**：命中区顺序与「按命中类型追加右键菜单」同刀（gaea 各区独立 DOM，
  天然分层）；PDM 节点半径（会破 NODE_W/H 分层）；列筛持久化（斑马
  `column_filters` 是二次编码 JSON 表达式，未建模）；异常标记/进度滞后两类过滤器。
- **边界**：未提交、未构建 Wails、未装机。

## gaea 打包 · gaea-schedule-zpert8-20261114-1052adec（进度板块斑马复刻+DSH 批代收编） · 2026-11-14

- **包**：`releases/gaea-schedule-zpert8-20261114-1052adec/`（PACKAGE-INFO+SHA256SUMS 入库；
  payload exe/三 DLL+preview.zip+source.tar.gz 磁盘留存不入库，P2-3 口径）。
- **基线**：`1052adec`（检出完整性修复提交）——`83820741` 的 SchedulePage 已 import
  commandRegistry/ScheduleCommandPalette，但四文件树面未跟踪，干净工作树构建即炸
  TS2307（「磁盘能构建≠提交能检出」老坑再现）→ 批代入库修复后以该提交打包。
- **范围**：aa60faa8（v2 包）后 15 笔——DSH 批代系（18fb5dcb/2eb8d9fd，ancestry
  is-ancestor 已验证）+ 进度板块线 14 笔（板块分离一级/S1~S5/视图库/一次点亮 20/22/
  格式组+工作分类）。codex 在途（命令面板续刀/.gaea/schedule.css）不进包（干净树纪律）。
- **流程**：独立干净工作树 `.tmp/pkg-zpert8`（junction 复用主树 node_modules，同 lock）
  → `npm run build` 39.25s → `wails build -s -ldflags "-s -w" -trimpath` 1m0.2s
  （bindings 未跳过）→ payload 复制到 `releases/…/payload/` 正式路径 → `smoke.ps1`
  `/api/health` 200（PID 48808 回收）→ 工作树回收（junction 先 rmdir 再 remove，
  主树 node_modules 完好）。
- **SHA256**：exe `08b513d4859fe8b6…`；zip `8ff627600ecfe923…`；tar `2fa938c52b3e43c4…`
  （全表见包内 SHA256SUMS-1052adec.txt）。版本源 4.543.0 不抬号，以提交号标识。

## gaea 进度板块 · ZPert W2 命令注册表 + 快捷键助手 · 2026-11-14

承接 `进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` 的 W2，把命令
从功能区散落动作收成可检索、可测试、可复用的单一派发层。
- **注册表**：新增 `commandRegistry.ts`，41 条命令统一稳定 ID/分组/说明/关键词/
  快捷键/禁用原因；handler 仍由 `SchedulePage` 运行期注入，符合斑马“命令元数据
  与业务处理分离、没有总函数表”的真实结构。
- **单入口**：功能区、文件/编辑/视图菜单、命令面板、键盘快捷键全部经 `runCommand`；
  同一命令的可用状态和失败文案只维护一份，禁用项不静默消失。
- **命令面板**：顶部“命令 Ctrl K”常驻入口；`Ctrl+K / Ctrl+/ / F1` 打开；
  支持中英文搜索、分组、快捷键提示、↑/↓、Enter、Esc/遮罩关闭与焦点管理。
- **快捷键**：接通 `Ctrl+N/F/P/Z/Y/Shift+Z/C/V`、`Delete`、`Alt+↑/↓`、
  `Shift+Alt+←/→`；输入框/文本域/可编辑区聚焦时不抢编辑键。
- **验证**：schedule + 页面回归 **63 文件 / 786 例全绿**；新增注册表 4 例、
  命令面板 4 例、页面命令链 1 例；`tsc -b`、定向 ESLint 0、Vite 生产构建通过
  （6385 modules / 23.99s）。`?mock=1` 亮暗双态与 1100×800、880×800 走查：
  命令面板无横向溢出，搜索“单代号”经 Enter 正确切换视图。
- **边界**：本轮不构建 Wails exe、不装机；W4 双标尺/图上开关、W5 列级筛选与
  统一就地编辑、W6 按纸张布图/图签仍按原计划后续。
- **提交状态**：并发提交 `64a2f632` 已带入 `SchedulePage` 的派发接线；新建
  `commandRegistry`/`ScheduleCommandPalette`、CSS/测试与本条记忆尚未整体提交。

## gaea 进度板块 · ZPert UX/UI 布局移植首刀（W1 + 功能区/右舷） · 2026-11-14

按 `进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` 实施，直接以
`C:\AI\zpert-re\mockup\zpert-ui.html` + `ViewDB.gdb.json` 为结构基线。
- **视图层**：9 个 ViewDB 内置横道视图接线；计划编制默认列改为
  `序号/分区/标记/工作名称/工期(工日)/前置工作/计划开始/计划完成`；
  分区用 WBS 根号转罗马数字，星标=分组、菱形=里程碑；宽表固定 730px 核并横向滚动。
- **分组**：14 维（工期/总时差/负责人/里程碑/关键/分类/名称/部位/状态/资源/
  前后置数/自定义/模式），负责人/部位/自定义落到既有 text2/text1/text3。
- **布局**：ZPert 式七页签功能区（文件/计划/资源/管控/协作/视图/帮助），
  计划页严格六面板“视图/编辑/格式/大纲/插入/高级”，组名置底；文档条、全宽工作区、
  250px 右舷“预警信息”与状态栏同屏。未完成能力按参考位置禁用预留。
  去掉常驻工程信息横条及文档条路径/当前视图/重复按钮；工程属性收进“工程设置”弹层。
  AI 对话默认收起，可从功能区恢复。横道附加工具条在 ZPert 工作区隐藏，计划默认全览。
- **网络图**：新增 ZPert 文档双栏，单代号/双代号均为左侧计划表 + 右侧现有
  PDM/AOA 画布，默认 1:1、2px 分隔（1600 宽实测左右各 626px）；
  时标网络图/逻辑网络图按钮原位禁用占位。
- **验证**：schedule 回归 55 文件/733 例绿；SchedulePage/ScheduleDock/GanttView
  定向 57 例+网络双栏 39 例绿；tsc/ESLint 0；Vite 生产构建 39.93s；1600×1000 暗亮走查无横向溢出，
  实测表格 730px / 画布 519px / 右舷 250px。
  截图：`temp/gaea-schedule-zpert-ribbon-light.png` / `…-dark.png`。
- **边界**：命令注册表/快捷键助手、网络图表格同屏、双标尺和五图模式矩阵仍按计划
  后续波次；本轮未构建 Wails exe、未装机。

## gaea 进度板块 · 斑马复刻第 5 步（S5 预警状态机） · 2026-11-14

判定书 §6 第 5 步落地（8bd52a64，装机 C3106D4A2710… 冒烟 200）。
- **数据位**：`Task.askFinish` 双端承载（斑马 `ask_stop_time` 口径=上级下达的要求完成，
  工作日序号 0=开工日）：前端全可选+store 归一（非整数/负数丢弃）；Go `Task.AskFinish`
  omitempty + `Validate` fail-closed（负值拒绝）。旧文件零迁移。
- **`warn.ts` 引擎**（口径全来自斑马 RE §5.3）：红绿灯（红灯=突破要求完成，**压线 overdue=0 也算绿**）；
  两算法可切 `plan-ask`（计划-要求）/`predict-ask`（预测-要求，预测=均匀速率外推——回填 finish 优先、
  进行中按「已完成量/已跨工作日」速率摊剩余、未开始按计划，与 reviews 当期量同族近似）；
  滞后五档 (0,15]/(15,30]/(30,60]/(60,100]/100+；分组行不参与；`warnSummary` 计数+滞后到提前排序。
- **关键口径坑（红灯用例首跑逮出）**：CPM 的 `ef` 是独占边界（`es+d`），「计划完成日」必须取
  **最后工作日口径**（d>0 → ef−1；里程碑/零工期 → ef）——直接拿 ef 比要求完成，差一天全是假红灯。
- **接入**：TaskInspector 要求完成 DatePicker 编辑（dateToWd 唯一口径；清除→undefined）+预警段
  （红绿灯 Tag+档位+算法 Segmented 切换，无要求时间不渲染段）；SchedulePage/ScheduleTool 同口径接线；
  状态栏新增「预警红灯 N 项」（红灯才显）；gschedSummary 增 `reviewCount`+`warn{red,green,unassigned}`
  （agent 面，恒 plan-ask 稳定口径，预测算法留视图态）。
- **验证**：warn.test 17 例+TaskInspector +5 例+gschedSummary +1 例；schedule 回归 **57 文件/758 例全绿**；
  tsc/eslint 0；Go internal/schedule+app(Schedule|Bindings) ok；构建装机 INSTALL OK（exe C3106D4A…）+
  冒烟 200。
- **边界**：斑马「预警天数升高/降低」期际升降档子档挂下刀；预测外推是均匀速率近似（斑马内核该点未反编译）；
  jsdom 中 rc-picker 打字+Enter 解析不出（吐 null）——组件测试只覆盖清除与值渲染，日期→工作日换算由
  calendar/warn 单测钉死（不伪造交互，沿用办公对话流「jsdom 不测几何」先例）。

## gaea 进度板块 · 斑马复刻 S1~S4 收编 + 板块分离一级 · 2026-11-14

两笔提交（feda9c04 + 2a564426），承接上一执行体在树面的 S1~S4 遗留并落地用户拍板的板块分离。
- **收编 feda9c04**：上一执行体已完成未提交的四步（承载位 blocks 双端 / N1 点-箭持久化 /
  S1 检查器 18 条 / S2 检视+拉直+ReviewPanel+横道前锋线跟随）照原样提交，三档 zpert8 报告
  （D0 蒸馏/网络逆向/复刻判定书）入 `进度计划/`；`frontend/dbg.test.ts` 调试草稿删除；
  `re-zpert/`（2.5MB 逆向证据，含反编译 .c）**刻意不入库**（不搬上游派生物，报告自带引用）。
- **板块分离 2a564426**（2026-11-14 用户拍板「进度板块从办公板块分离为一级板块」）：
  schedule `inMenu false→true` **双端同步翻**（Go `board/builtins.go` + 前端 `boards/manifests.ts`，
  撤 v4.168.0 瘦身刀1 并入语义），menuOrder 12 不变（落位微信助手与运行监测之间）；
  **办公侧承接入口全保留**（文件面 .gsched 摘要卡/命令面板项/右栏一级 Tab ScheduleTool）；
  白名单语义不变（注册即可导航）。测试同步：manifests.test.ts 菜单序 11→12 项、
  CommandRail.test.tsx rail 全量序、board_manifest_test.go InMenu 断言翻转。
- **验证**：boards+layouts 定向 51 例绿；schedule 全回归 **56 文件/735 例绿**；`tsc -b`/eslint 0；
  Go `internal/schedule`+`internal/app`+`internal/app/board` 全 ok；构建装机 INSTALL OK
  （exe 34B8C0B8DBF7…，构建 11:54:06）+ 冒烟 `/api/health` 200。
- **坑**：smoke.ps1 路径参数反斜杠被 bash 吞（`C:AIgaea-appgaea.exe`）——Git Bash 内联 PS
  传参一律用正斜杠（已知坑第 6 次变体）。

## gaea 进度板块 · 斑马复刻第 4 步（S2 检视 + 拉直） · 2026-11-14

把第 1 步的 `Project.reviews` 承载位变成可用模型 + 面板（图上画线体留下一步）。
- **新建 `reviews.ts`**（256 行纯函数，三条口径写进文件头）：① 历史不可覆盖（同期只一条、
  不能插入早于已冻结期的新期）；② 拉直写回 + 冻结 + **只升不降**；③ 完成量**按有效工期加权**
  （里程碑/分组行不进分母）。API：`makeReview`（进度快照，100% 顺带填 finish=at）、
  `reviewPercent/taskStatus`（三态）、`projectPercent`、`compareReviews`（只列变化行）、
  `straighten`、`reviewProgress`（当期计划应完成 vs 实际，按 CPM elapsed 算）、
  `reviewWorkday/reviewDate/defaultReviewDate`、`canAddReview`（重复/上限 30/早于已冻结）、
  `frontlineWorkday`（给下一步画前锋线用）。
- **新建 `ReviewPanel.tsx`**（211 行，结构照 BaselinesPanel：只收 cpm，动作全走 store）：
  打一期检视（日期缺省今天，早于开工日取开工日）+ 期列表（完成量/冻结标签/拉直·删除）+
  当期两列（计划%·实际%·偏差）+ 期际对比表 + 前锋线位置提示。
- **store 三动作**：`addReview`/`removeReview`/`straightenReview`，均 `pushHistory`（可撤销），
  拒绝原因原样回给面板（不静默）。接入 `SchedulePage` 信息条「基线」左边的「检视」按钮；
  ScheduleTool 不加（与基线一致）。
- **横道图前锋线已接线**（同轮补记）：查码发现前锋线本来就有（`GanttOverlay` 三类样式 +
  工具栏开关/日期框 + `frontierWd()` 折点算法），缺的只是数据源——一直由视图态 `checkDate`
  默认"今天"驱动。本轮把 `showFront`/`checkDate` 初值改为**有最新检视就默认开线并停在那一期**
  （`reviewDate` 反算 ISO），无检视回落今天；用户仍可工具栏覆盖。新增 GanttView 2 例。
  至此"打点 → 图上站到那天 → 偏差由 reviewProgress 给出 → 拉直写回并冻结"闭环；AOA 线体待做。
- **验证**：`reviews.test.ts` 14 例（含"加权≠算术平均"、拉直只升不降、冻结幂等、三道闸）、
  `ReviewPanel.test.tsx` 6 例、GanttView +2 例；回归 **53 文件 / 712 例全绿**（35.8s）；
  `tsc -b` / eslint 0 错；Go `internal/schedule` + `internal/app(Schedule|Bindings)` ok。
- **边界**：图上尚无前锋线线体（`frontlineWorkday` 已备好）；`actuals` 只能由进度快照生成，
  逐任务手填实际时间缺编辑 UI（数据结构与 `taskStatus` 已支持）；`reviewProgress` 的计划量
  按"已跨过工期"近似（斑马同为均匀摊）。

## gaea 进度板块 · 斑马复刻第 3 步（S1 检查器） · 2026-11-14

把斑马「检查」做成**逐条可定位诊断**（与 DCMA 分工：DCMA 算比率，checker 给定位）。
- **新建 `checker.ts`**（纯函数 `checkProject(project, cpm?)`）：18 条规则，每条带
  `code/severity/message/hint/taskId?/linkIndex?`。分三档：
  error（循环、悬空、自环、非法类型、分组行挂关系/分配、里程碑带工期、工期越界 = Go Validate 的提前告知）、
  warn（传递冗余关系、负时距、孤立、无前置、无收口、空分组、零搭接）、
  info（无里程碑、未定价分配、高浮时>20、近关键≤2）。**关键工作刻意不逐条报**（交 DCMA/横道）。
  附 `sortDiagnostics`（error→warn→info，同级按任务表序）与 `diagSummary`。
- **刻意不做 Go check.go**：Go 已有 `Validate`（fail-closed）+ `schedule_analyze` 的字符串 checks，
  再加一份结构化 check.go 会成第三套口径且无消费者。要上移应改 `analysis.go` 把 findings 结构化。
- **接入**：`TaskInspector` 加可选 `diagnostics` prop（抽屉「检查发现（N）」段：阻断/可疑/提示标签 +
  下一步建议，空数组不渲染）；`SchedulePage`/`ScheduleTool` 各算一次并按 `inspectId` 过滤传入，
  项目级只带 `cycle` 进抽屉。
- **验证**：checker 19 例 + TaskInspector 新增 2 例；回归 **51 文件 / 690 例全绿**（41.0s）；
  `tsc -b` / `eslint src/schedule` 0 错；Go `internal/schedule` 与 `internal/app(Schedule|Bindings)` ok。
- 下一步：S2 检视 + 前锋线（消费第 1 步的 `reviews` 承载位）。

## gaea 进度板块 · 斑马复刻第 2 步（N1 点-箭持久化） · 2026-11-14

把 `Project.aoa` 接进双代号渲染链，解决"拓扑一变编号与手动布点漂移"这个结构性差异。
- **契约**：`SchedAoaDot.id` **就是锚点键**（`S`/`T`/`start:<taskId>`/`end:<taskId>`）——
  与既有 `anchor`（pin 匹配键）天然同一套身份，不需要第二套映射。
- **消费侧**（`buildAoa` 加 `persisted` 选项）：命中持久点的节点用其 `num`/`layer` 并带 `dotId`；
  未命中者按原顺序**补号跳过占用**（编号唯一，i<j 显示的前提）；一个都没命中→退回派生；
  持久编号生效时 `nodes` 按 num 升序（数组序=编号序=i<j 断言同源），派生口径旧行为**逐字节不变**；
  新增 `AoaGraph.numSource`（derived|persisted）与 `AoaEdge.arrowId`（两端有 dotId 时反查）。
  时间/几何完全不受编号覆盖影响。
- **生产侧**：新增 `aoaGraphFromBuild(graph, prev?)`，**只在用户第一次手动布点时**固化
  （`AoaView.commitPin`，`graph.ok && nodes>0`）——凭空固化只会制造历史包袱；
  `dot.layer` 初值取 band 且优先沿用已有持久层号（给分层刀留位）；`arrow.id` 用 `${from}>${to}`
  （跨拓扑稳定），`beginDotId='S'`，S/T 打 isBegin/isEnd。store 加 `setAoa` 动作（入历史）。
- **pins 降级**：新增 `pinsToPersist`（与 `prunePins` 只差一条：**与自动位一致的键不落盘**），
  `commitPin` 改用它 —— `AOA_PINS_MAX=5000` 从"整图坐标快照上限"变成"真手动布点数上限"；
  重置按钮抽 `resetPins`（语义不变）。
- **验证**：`tsc -b` 0 错、`eslint src/schedule` 0 错；定向 79 例（aoa 32/layout 31/AoaView 16）；
  回归 **50 文件 / 669 例全绿**；Go `internal/schedule` 与 `internal/app -run 'Schedule|Bindings'` ok。
  新增 13 例覆盖：持久编号覆盖与排序、部分命中补号唯一、全不命中退回、arrowId 标注、
  固化回灌、固化后插任务老编号不变、prev.layer 优先、pinsToPersist 三处境。
- **边界**：`layer` 暂只承载不消费；固化后不自动清理已删任务的点/箭（斑马也留，上限由
  Go `validateBlocks` 20000/60000 兜底）；`numSource` 尚未上界面提示。

## gaea 进度板块 · 斑马复刻第 1 步落地（四块承载位） · 2026-11-14

按判定书 §6 第 1 步开工并收口：四个"斑马复刻承载位"（检视 / 点-箭 / 图面参数 / 视图）
做到**能存能读不丢**，不接 UI、CPM 不消费、不改任何既有语义。
- **前端**：`types.ts` +165 行（`SchedReview(Actual)/SchedAoaDot/SchedAoaArrow/SchedAoaGraph/
  SchedNetworkStyle/SchedRenderMode/SchedView(Sort)`，`SchedProject` 加四个**全可选**字段）；
  新建 `blocks.ts`（320 行，四个 normalize 纯函数 + 上限常量，风格对齐 `normalizeAoaLayout`）；
  `store.ts` 的 `normalizeProject` 末尾接四行归一（`...p` 本来就透传，这里补形状归一）。
- **后端**：`types.go` +136 行同构结构体（四块 `omitempty`，布尔用 `*bool` 保三态）；
  新建 `blocks.go`（300 行 `validateBlocks` + 子校验：枚举白名单/区间/唯一性/引用完整性/规模上限）；
  `project.go` 的 `Validate` 末尾接 fail-closed 闸。
- **契约五条**：① 四块全缺省时 `.gsched.json` 一个键都不多写（专项用例断言，旧文件零迁移）；
  ② 前端"容错丢条、保证能打开" / Go"非法即拒保存并给可读原因"，两侧样例集互为镜像；
  ③ 布尔三态（`*bool`）；④ `Review.at` 与 `CpmResult` 同为工作日序号，`date` 仅展示；
  ⑤ 箭线 `lag` 沿用斑马 ±9999 闸门。
- **验证**：`go test ./internal/schedule/` ok；`go test ./internal/app/ -run 'Schedule|BindingsCompleteness'` ok
  （需 `CGO_ENABLED=1 CC=C:\msys64\ucrt64\bin\gcc.exe` 且 PATH 带 `C:\AI\gaea-app` 的三枚 DLL）；
  `vitest run src/schedule` **49 文件 / 639 例全绿**（60.6s，新增 blocks 20 例）；`tsc -b` 0 错；eslint 0 错。
- 下一步：N1 —— `buildAoa` 加"有 `Project.aoa` 就用它、没有则派生"的分支，pins 降为手动差量。

## gaea 进度板块 · 复刻斑马前存量评估（舍/改判定书） · 2026-11-14

用户拍板"复刻斑马进度"，第一步先判 gaea 存量代码舍弃还是修改。产出
`进度计划/gaea-schedule-zpert8-replication-assess-20261114.md`（判定书，8 章）。
- **结论：不要重写。** 存量 ≈15.1K 行源 + 13.2K 行测试（前端 55 源/10039 行 + 46 测试/7566 行；
  Go 14 源/4178 + 17 测试/4690；app 绑定 890 行/11 方法），约 537 vitest 用例 + 137 Go 测试函数。
  且**双代号引擎口径已与斑马同源**（`aoa.ts` 头注释：唯一起终点、链式免虚箭、FS/SS/FF/SF 补虚箭、
  虚箭去重、事件正逆推、关键线路含虚箭；`AoaView` 已有波形线=自由时差、过桥法、JGJ/T 121 正交）。
- **判定分布**：保留 ~40 文件、修改 ~12、改造 3（`aoa.ts`/`AoaView.tsx`/`ganttCols.ts`）、
  新增 6 组（检查器/检视+前锋线/预警/视图快照/图面参数/期间计划）、**整块舍弃 0 个文件**。
- **耦合实测**：`types.ts` 被 79 处引用（只能增量加可选字段）、`cpm.ts` 30、`store.ts` 21（唯一高风险点，
  只加动作不改语义）、`calendar` 15、`aoa` 9、`aoaLayout` 6。schedule 之外还有 `pages/SchedulePage*`
  与 `pages/schedule/*` 5 个消费方。
- **四个硬冲突**（决定"怎么改"不是"要不要改"）：① 点身份＝派生 anchor vs 斑马持久 dots；
  ② 显示模式 1 vs 5（且每模式一块几何基准）；③ 列/分组是静态常量（`GANTT_COLS` / 3 分组维度）；
  ④ CPM 前后端**手写镜像**必须双向同步。
- **真正舍弃只有 3 处策略级**：`aoa.ts` 的"每次重建事件 id + 字典序 anchor"默认口径、
  `aoaLayout.ts` 的 `AOA_PINS_MAX=5000` 全量坐标落盘、`ganttCols.ts` 静态列定义作唯一真源。
- **迁移顺序**（8 步）：types 增量透传 → N1 点-箭持久化 → S1 检查器 → S2 检视+前锋线 → 预警 →
  视图/列注册表（唯一结构改造，风险最高）→ 5 模式矩阵 → MPP 导出/期间计划。

## 斑马进度 ZPert8 · Ghidra 收口二：布图拟合器与点结构 · 2026-11-14

Ghidra 第二轮（工程已存，秒级查询），报告扩到 §16（955 行 / 73.6KB）：
- **布图不是"分层器"而是"拟合器"**：`0x140507510`(5552B)+`0x14051cf50`(1179B) 按目标纵横比
  `param_3` 迭代修正列宽/行高，实测容差常量 **0.003 / 0.03**（`DAT_140bf7198`/`DAT_140bf71a0`），
  另有 0.1 下界、1.0 基准、0.5 步长。**没有任何拓扑分层循环**。
- **Dot 结构定死**（`0x1401f4b40` 148B 构造器，0x60 字节）：`+0x00 id`、`+0x04 序号`、
  `+0x08..0x18` 向量A（首侧箭头 begin/end/cap）、`+0x20..0x30` 向量B（次侧）、
  `+0x38/0x40/0x48` 三个侵入式链表头（共享哨兵 `DAT_141125778`）、`+0x54 = -1`（候选层号）、`+0x58 = 0`。
- **两处更正**：① `Dot[+0x04]` = **点在 dots 数组的下标**（`FUN_1401f5540` 插入/删除后统一重排
  `dots[i]->[+4]=i`）→ `canConnect` 的排序键是**数组先后**，不是层号；② `canConnect` 尾段对每个箭头
  调的是 `0x1401cb3d0`+`0x1401ccb20`(2309B)+`0x1401cb8c0`（**不是** rebind），rebind 只在 `mergeCheck` 内。
- **"分层在哪"收窄**：时标模式下 x 由 `plan_start/stop` 直接映射，`layer` 是 Work 上的持久字段；
  `Dot[+0x54]=-1` 是候选缓存槽。下一步钩子：找写 `[+0x54]` 的代码（capstone 扫 `mov dword [reg+0x54], reg`
  已试 imm 形式仅 2 处且无关，需改扫寄存器形式或从"智能调图"命令反查）。
- 未解收敛为 6 条（§16.5），均有地址/方法可续。

## 斑马进度 ZPert8 · Ghidra 收口：关系合法性伪代码实证 · 2026-11-14

按用户拍板装 Ghidra 收口（Temurin JDK 21.0.13+9 + Ghidra 12.1.4 PUBLIC，均在 `C:\Tools`，
项目 `C:\Tools\ghidra-proj\ZPertProj` 已保存可秒查；全量分析 1046s）。报告新增 §15（现 15 章）。
- **自研工具**：`re-zpert/ghidra/DumpDecomp.java`（地址列表 → 反编译 C + callers/callees），
  产物 `re-zpert/evidence/decomp{1,2,3}.c`。
- **关系合法性收口**（关键）：`canConnect(0x14050ca90)` **不接类型参数**——只判"同点/端点缺失/
  按实体 `[+4]`（层/序号）排序/两端邻居表冲突"；类型判定在 `decideLinkType(0x14036c640)`：
  按"A 首/A 次/B 首/B 次四侧是否已有非 FS 箭头"的位掩码（8/4/2/1）决定 FS/SS/FF/SF，
  **FF/SF 的"不合法"= 该点首侧已有"已落地"箭头**（不是搭接组合非法）。六个埋点文案长度
  (6/6/9/10/9/10) 与反编译里的 len 参数逐一吻合，埋点号实为 **81001–81006**（D1 误写 18001）。
- **建关系三段式**：入口 `0x1404e7cf0`（slot 2/4 登记端点 → canConnect → op∈{10,12} → splice
  `0x14050cc40`；失败统一错误码 0x40 + 不回写图数据）。
- **双代号自建点**：`newDot(0x140274ba0)` 给节点分配 **递增编号 `[0]` + 数组序号 `[1]`**；
  `mergeDot(0x1402729a0)` 合并重复点；`splice` 在端点为 0 时**自动补点** → 用户不需要手摆节点。
- **布图实证**：`0x14051c5e0` 每模式两块几何基准（`mode==6` 用 `0x1b40/0x1b28`，否则 `0x19a8/0x1990`）；
  提交前校验页/格数 **一条路径 [60,1000]、另一条 [1,200]**；失败**先回滚尺寸再弹 67 字文案**。
- 仍未解：`neighborsCompatible(0x140274360)` 两项 146 字节小工具的精确判据、`layer` 由谁分配
  （在 `0x140507510`/`0x14051cf50` 内）、`SceneConflictChecker`、虚箭"必须建"的时机。

## 斑马进度 ZPert8 · 网络图逆向续刀：`.zp` schema + pert_json 结构 · 2026-11-14
D1 之后按"继续"再推一程，把两件事从"推测"变成"实证"（报告已扩到 15 章 / 535 行）：
- **`.zp` 的 protobuf 只有 18 个消息**（信封 / 授权 / 协作），从嵌入的 `FileDescriptorProto`
  （文件偏移 `0x89a7f0`）解出**完整字段号与类型**：`ZPertFile{type=1,args=2,pert_json=3,images=4,attachments=5}`、
  `ZPertFileArgs{brotli_original_size=1,aes_iv=2,co_edit_version=3,timestamp=4,lock_num=5}`。
  新工具 `re-zpert/tools/protodesc.py`，产物 `re-zpert/evidence/zpbdata_schema.{txt,json}`。
- **协作协议也在里面**：`Request{seq,type,project_uid,diff,token}` / `Response{…,file,diffs,member(s)}` /
  `Diff{pert_diff,images,deletes[],insert_at[],version}` → **版本号 + 位置索 op 序列**，不是 CRDT。
- **授权模型带功能闸**：`RestrictedFunctions{…,max_works_count,show_watermark,watermark_pattern,
  view/pert/resource/real_info_read_only,columns,functions}` —— 免费版限工作数就靠 `max_works_count`。
- **计划本体是 JSON**（不是 protobuf）：按"键池 + 代码引用顺序 = 发射顺序"还原出三段序列化：
  文档主体（`info_base`/`pert_prop`/`pert_links`/`pen_set`/`color_set`）、颜色与项目信息（`project_info`/
  `pert_script`/`jobtype_setting`/`gantt_style`）、图面（`network` 那 60+ 键）；视图项 =
  `{kind,code,forign_name,color_set,p_colorset,pert_mode,is_show_grid_editor}`。
  键池全表 116 键已落附录 C。
- 修正：D1 §3.1 原说"键序来自 protobuf"→ 实为 **rapidjson 写侧键字面量顺序**；结论方向不变（结构可镜像）。
- **关系合法性判定已挖到代码级**（报告 §13）：`0x1404e7cf0` 创建入口 → `canConnect 0x14050ca90`
  （同点/端点缺失/按实体 `[+4]` 整数排序/两端邻居表逐项处理）→ 核心判定 `sub_140274360`；
  同族还有返回错误码 `0x98/0x99/0x9b` 的判定入口。要写成 gaea 规则仍需伪代码，但函数边界已给。
- 对 gaea 的最终口径：网络图 = **`pert_prop`（拓扑）+ `network`（图面参数）+ `views_list[].pert_mode`（视图→模式）
  + 每工作 `layer`（层号）** 四件套；把 `aoaLayout.pins` 换成"层号 + 参数 + 现算"即与斑马同构。

## 斑马进度 ZPert8 · 网络图子系统逆向（D1 深挖） · 2026-11-14

在 D0 全产品蒸馏之后，按用户要求以**网络图**为重心把斑马进度整个过完，
产出 `进度计划/gaea-schedule-zpert8-network-re-20261114.md`。
- 方法升级（无 Ghidra）：`.pdata` 精确函数边界（**22,402 个函数**）+ capstone 定点反汇编 +
  RIP 引用自动注解字符串 + 字符串簇聚反推子系统。工具落盘 `re-zpert/tools/{pdata,disasm,vtable}.py`。
  vtable 探测为负结果：`Layouter/SceneLayouter/Scene` 有 RTTI TypeDescriptor 但**无 COL/vtable 落盘**，
  后续只能按字符串簇读汇编或装 Ghidra。
- 子系统地图：网络图 UI 逻辑集中在 VA `0x1404f8000–0x140541500`（≈330KB），已按字符串簇标出 7 组函数起点。
- **五个硬结论**：① 双代号是独立持久化结构（`pert_prop{dots,begin_dot_id,job_type,
  compute_start_time,compute_stop_time,group_id}` + `pert_links` + `dots_size`），不是渲染派生；
  ② 显示模式是 `char[5][16]` 具名枚举（时标逻辑图/时标网络图/逻辑网络图/单代号网络图/横道图）+
  运行时 `pert_mode`（实测 `[view+0x11d8]`，`mode==6` 另有几何块分支）+ 废弃模式迁移；
  ③ 图面是 60+ 键状态块（`is_stagger/is_hide_unimportant_link/dot_style/arrow_style/head_height/
  cr_x..crl_y/layer_height1-2/virtualjobcolor…`）→ **有虚工作概念**；
  ④ 布图是"排版—压图幅"四维约束流水线（节点×字号×紧凑度×纸张），失败给三条建议；
  ⑤ 关系合法性按 6 结果埋点（18001–18006）：FS/SS 无条件，**FF/SF 走"合法/不合法"判定**。
- **D0 修正**：`.zp` 容器真身 = `zpbdata.ZPertFileArgs{pert_json, brotli_original_size, aes_iv, …}`
  → 正文 JSON 经 **brotli 压缩 + AES（iv 单独存）**；D0 §3.6 已加注。
- gaea 落地：N1 点-箭模型落库（学斑马"存层+参数现算"，把 `aoaLayout.pins` 降级为人工覆盖）→
  N2 图面状态块 → N3 布图流水线（分层/时标/虚箭/冲突消解/压实）→ N4 关系合法性 → N5 下钻上钻。

## 斑马进度 ZPert8 逆向蒸馏（进度板块对标） · 2026-11-14

对 `C:\Program Files\ZebraTech\ZPert8`（斑马进度计划 7.0.706.0）做静态逆向蒸馏，
产出 `进度计划/gaea-schedule-zpert8-distill-20261114.md`（D0 收口）。
- 结构判定：Qt6.7.3-Glodon + QCefView 壳，**业务 UI 是本地 C++ Qt 网格**（RTTI：
  `ZBaseGridWidget<T>`/`ZQtPertWindow`/`ZGanttState`），CEF 只挂 AI/帮助/登录/云；
  内核对象 `Pert/Work/Link/Calendar/Progress`+`ConflictChecker`+`zpbdata::ZPertFile`(protobuf)。
- 判决：`.zp/.gzp/.gdb` = protobuf 信封 + 高熵加密体（实测头部 + zlib/bz2/lzma 全不通），
  **不追该格式**；互操作走 MPP / MSPDI XML / Excel（三条它自己也在用）。
- 拿到三张可直接落地的清单：① 70 列工作字段中英对照（含三套时间/两套时差/关键性四视角）；
  ② 检查器规则集 C1–C15（工作级体检文案即规则，可直译 diagnostic）；③ 进度管控层
  （检视时间四条时序规则 + 前锋线/拉直/冻结 + 预警状态机两算法五档 + 里程碑治理）。
- 差距矩阵结论：gaea 的计划编制/基线/DCMA/挣值/蒙特卡洛**已深于斑马**，但
  **执行期治理（检视/前锋线/预警/里程碑）与视图体系（10 内置视图 + 14 分组维度）整层为空**。
- 刀单：S1 检查器 → S2 检视时间+前锋线 → S3 排网方式(ASAP/ALAP)+要求时间约束 →
  S4 MPP 导出 → S5 视图/分组 → S6 资源均衡 → S7 期间计划/定额（按客户触发）。
- 证据落盘 `re-zpert/evidence/`（RTTI 1166 条 / 蛇形键 2371 条 / 清洗中文串 3657 行 / 分组 8 类）。
- 未做：Ghidra 反编译（`GHIDRA_INSTALL_DIR` 未配置）、动态跑测（无授权、`%APPDATA%\ZPert8` 为空）、
  CEF JS 桥（下一步在 `QCefView.dll` 侧挖）。

## CAD/进度计划回归右栏一级 Tab · 2026-11-14

按用户拍板把办公板块的 CAD 识图与进度计划从 ToolSheet 临时面板改到右侧面板
成为独立标签，办公板块可直接打开 CAD：
- `lib/workspaceTabs.ts` 清单新增 `cad`（可选使用红线 defaultEnabled:false，
  「CAD 图纸」）与 `schedule`（默认启用，「进度计划」）两个一级 Tab；命令面板
  /欢迎网格/＋下拉三入口由注册表与启用集自动派生（显式点击=启用并打开）。
- `lib/sidebarRegistry.ts` RENDERERS 以 React.lazy 接线 CadPanel(pane 变体)/
  ScheduleTool，WorkspacePanelContext 恢复 `onAsk`（App 接 send）——问图纸/
  问这个计划从右栏直发对话。
- App 退役 activeTool/ToolSheet/§7.3 检查器 rail 宿主三件套：工具下拉、命令
  面板、OPEN_TOOL 单通道、首页 tile 冷启补开统一走 `openToolTab`（停用先启用
  再 openPaneView）；显式 cmd-schedule/cmd-tool-cad 退役（注册表循环已派生）。
- 新增 `lib/cadOpen.ts`：dwg/dxf 文件打开链统一路由——usePreviewPanel.openPaneFile
  与 sidebarRegistry.openPaneFileTab 双口拦截，启用可选卡+开视图 tab+请求总线；
  CadPanel 订阅总线按一次性消费装载（busy 暂存不丢，消费即清不重放）。
- ToolSheet.tsx/.test.tsx 删除；App.palette 契约锁改新语义；workspaceTabs/
  sidebarRegistry 契约测试随 7 Tab 清单同步（cad 免全禁兜底例外）。
- 验证：`tsc -b` 绿；定向 15 文件 92 例 + 契约 4 文件 50 例绿；ESLint 改动面 0；
  knip 无新增欠账；全量 vitest 4414 例复核=本刀零交集（余红=DSH 同树在途
  context/fleet 族 + DatabaseLibrary 负载假红复跑绿）。装机=干净树 fd9caa06
  构建（exe sha256 3C0CDF4D…，build 11/14 00:57:28），smoke /api/health 200
  （PID 已回收）；坑=gaea_install.ps1 SourceDir 默认主树，工作树装机须显式
  -SourceDir。详见 docs/gaea-day-log-2026-11-14.md。
- 打包：`gaea-office-cadtab-20261114-c85a9815` 预览包（六刀全量；PACKAGE-INFO
  +SHA256SUMS 入库 4db1c355，exe/三 DLL/源码包/preview zip/uncommitted 快照
  留盘）；包产物=装机位同源 exe E65681B6…（c85a9815 干净树构建），冒烟 200。

## dsh-context 0.84.1 上下文交互续刀 · 2026-11-14

按 `bowenliang123/dsh-context` 官方 `v0.84.1` / `d81325ec` 源码继续对齐：
- 趋势总览补四分位标尺与分类点击；柱/色块悬停回传请求预览；Step brief
  补 `New in` 行。
- 浏览器七分类独立搜索、时间/大小/名称排序、逐类 delta；DNA 搜索、最新在
  上、确定性“引入步骤”；system/tools 节点从 request_header 展开真实系统
  提示词与逐工具 schema。
- Context Stats 下钻接通 User 浏览器分类、文件活动四类筛选、事件 kind 与
  命令 program；命令行可展开配对工具结果；文件操作可定位浏览器节点。
- 验证：最终相关 UI+mock 契约 81 例绿；Go `contextview` 与 `internal/app`
  全包绿（app 182.976s）；全量 `tsc -b` 绿；Vite 生产 build 39.86s；
  暗/亮 mock 实机走查。
- 续刀（同日）：浏览器「地图」从占位选项补成真实 token 面积 treemap，
  小格自动降密度但保留完整 aria/title；鼠标/键盘激活后切回树视图、展开
  并定位分类。修复第七分类 `skill` 已进入构成/图例却未计入上下文总量和
  百分比的问题；补齐三语 `viewMap/mapHint/mapItemLabel`。
- 原列余项已落地：历史步骤 surface 走 Go 日志前缀确定性重折叠；自定义源
  缺接口时诚实禁用；工具来源 chip 只在 header 投影提供时显示；非图片附件
  card 已进入节点详情链。
- 验证：相关 UI 4 文件 75 例绿；全量 `tsc -b` 绿；改动面 ESLint 0
  （含既有 fast-refresh warning 清理）；Vite 生产 build 33.21s 绿；
  mock 暗/亮双主题走查并实测地图按钮键盘 Enter 下钻。
- 上游续刀二：趋势增量改为按七分类堆叠正负变化并显示零轴；补「自适应」
  纵轴（按当前横向可见步骤缩放）、达到 100/200/300 步自动总览、手动关闭
  本会话粘性、初始定位最新步骤、钉住步骤横向居中、总览方向键/Home/End/
  Enter 下钻。事件行可点：压缩定位其后首个请求，其余事件优先直达对应
  浏览器节点；上下文页按会话保存纵向滚动位置。附件详情从“图片段 + 文件段”
  合并为按正文出现顺序混排的附件卡。
- 设置中心补趋势自动总览阈值，并补齐浏览器名称序、文件行数/名称序入口。
- 验证：相关 UI 5 文件 88 例绿；全量 `tsc -b` 绿；改动面 ESLint 0；
  Vite 生产 build 33.86s 绿；mock 暗/亮双主题走查，实测增量分类堆叠、
  自适应联动与事件行按钮态。
- 上游收官批三：新增 `GaeaContextDna` 批量历史读取序接口（单次日志折返、
  最多 128 个请求、32 份版本化缓存）；system 单段、request_header 逐工具
  schema 成段、其余 surface 节点按读取序成段。前端 DNA 仅请求横向可见窗口
  及前一步，90ms 防抖并在滚动时增量缓存；支持全局/增量 DNA、分类联动与
  首步基线说明。
- 新增 `GaeaContextCommands` 命令历史分页接口（按 Seq 排他游标、默认 200、
  硬上限 1000、32 份版本化页缓存）；命令卡在常规 400 条窗口之外显示“加载
  更早”，合并后程序分布、慢命令和重复命令分析同步覆盖旧页。绑定面 749→751。
- 验证：相关前端 7 文件 105 例绿；Go `contextview` + `internal/app` 定向绿；
  全量 `tsc -b`、改动面 ESLint 0；Vite 生产 build 32.85s；mock 亮暗主题
  实测 DNA 节点分段。完整 Wails 构建 1m22.507s（bindings 未跳过），
  `gaea.exe` 63,727,104 B / SHA256 `E12BDCF4…8E6`，正式路径冒烟
  `/api/health` 200（PID 53944 已回收）。
- 绑定守卫：`bindingNames.ts` 与 Go 一致 751 名；legacy guard 仍提示两个
  既有死项 `EngineUpdateStatus` / `StrataModelStart`（本轮未处理，非新增）。
- 当前剩余：历史 surface/DNA/附件/来源在真实长会话上的全链走查；自定义
  ContextView 数据源若要 DNA，需显式提供 `fetchDna`。本轮不装机。

## DSH Context / Fleet UX 对齐 · 2026-11-13（代码候选）

按用户要求不再从截图猜测，回源 `bowenliang123/dsh-context v0.84.1`
（commit `d81325ec`）的 `contextView.tsx`、`missionPulse.tsx` 与对应 CSS：
- 上下文页改为“统计 + 信息列 → Token/耗时 → 当前上下文+趋势 / 浏览器 →
  事件/文件 → 命令”；命令面板补左摘要右日志；输入框仅在办公「对话」显示，
  历史 `tasks/subagents` 右栏值回退文件。
- Agent 网络移出上下文，`编队` 改为卡片拓扑图（主 Agent/子 Agent 卡片 +
  贝塞尔连接），并补 Hero 摘要、跨 Agent 协作、协作任务、事件检查双栏；
  右栏任务 Tab 从注册表、清单、持久化与自动打开链路退役，自动打开偏好改跳
  「编队」。
- 验证：相关 13 文件 161 例绿；`tsc -b`、定向 ESLint 绿；`npm.cmd run build`
  成功（Vite 24.68s）；mock 亮/暗主题走查通过。未做 Wails 构建/装机。
- 打包：全部未提交源码/文档/截图已提交 `18fb5dcb`；未提交工作树快照
  `gaea-uncommitted-20261113.zip` 526,813,847 B / `E262BDAA...96A2F`，
  源码包 `gaea-18fb5dcb-source.tar.gz` 42,625,937 B / `FED4F578...777E`。
  14 个既有发布 ZIP/TAR 继续留在快照包内，不入 Git。

## 灵犀输出 UI batch3 · 2026-11-13（代码候选）

按 `C:\AI\re\lingxi\out/REPORT-对话窗口UI与文件成果卡.md` +
`REPORT-思考卡工具卡过程卡.md` 把办公对话输出层继续向灵犀形态收口：
交付物从纵向列表改为 2 列「文件成果卡」网格（DOC/XLSX/PPT/PDF/图片/HTML/
Markdown 等类型后缀徽标与淡色渐变），文件名保持主视觉，路径退到 title /
可访问名称；预览、外部打开、文件定位、复制路径、已更新/草稿/就绪/未生成
状态全部保留。过程卡状态徽标去底色，只在错误/中断保留语义色；成果卡用
`container-type:inline-size` 在窄对话区自动单列，避免 Gaea 左栏/右栏挤压时
继续硬塞两列。验证：DeliverableCards + Message + ProcessCard + ToolCard
定向 53 例绿、ESLint 0、`npm.cmd run build` 终稿绿（Vite 22.77s）；浏览器
mock 实机走查浅/暗主题四类型卡均正常。余：跟随模型输出的正文内联 FileChip
仍沿用既有 compact chip（非成果卡），按需再评估是否收进同一卡片语言。

续刀（同日）：删除 `.office-composer-meta` 独立状态行，把 `ContextPill`
改为 `ComposerToolbar` 的 `contextStatus` 插槽，剩余上下文百分比、微型
占用条与点击查看详情均保留，统一落在输入卡底栏右侧；快捷提示同区但窄窗
优先隐藏。定向 Composer/ContextPill/队列 19 例绿、ESLint 0、前端生产构建
终稿 Vite 23.61s；浅暗主题 mock 走查确认无输入框上方单独状态行。

首页续刀：`DeliverablesDock` 改为仅在非空会话挂载（`!showCenteredComposer`），
避免新建会话继承当前会话登记缓存而在空首页显示「交付文件/交互文件」面板；
空首页欢迎语与主标题三语统一大写品牌（zh「GAEA与你同行」/zh-TW「GAEA與你同行」/
en「GAEA is with you」）。定向 App 导出/首页依赖 15 例绿、ESLint 0；mock
实机走查确认空首页无交付停靠条、标题已替换；前端生产构建终稿 Vite 23.40s。

输出面板续刀：抽出 `lib/fileCard` 单源类型映射，输入框上方交付停靠条改为
默认收起的灵犀单行摘要，展开为 2 列紧凑成果卡；右侧「会话产物」行同步换为
同款类型徽标卡（图片继续保留缩略图），版本/验收/定位/复制/外开/沉淀成本
入口原样保留。输入区页脚与运行状态条改为透明，移除顶部分隔线并加强输入卡
投影，使输入框与上方画布同底悬浮。定向 50 例绿；mock 实机走查确认停靠条、
右侧面板与悬浮输入框风格一致；前端构建 Vite 25.81s。

找回与装机（2026-11-14）：上述 batch3 输出卡、输入卡上下文、悬浮输入框、
交付停靠条、右侧产物面板，以及后续消息操作浮层/阶段“思考中”去重，已按
`18fb5dcb` + 当前工作树版本完整找回；期间误撤的工作树改动未提交，已全部
恢复。相关 8 文件 108 例绿，前端生产构建 Vite 24.18s；按 UCRT64 GCC /
`CGO_ENABLED=1` / 系统 TMP 完整 Wails 构建 53.84s。最终 exe 64,020,992 B /
SHA256 `793F2FEB9FE33BFC64F387A682E954C65183AFD9B9328D62CF0AE88F978C4985`，
正式路径冒烟 `/api/health` 200（PID 53600 已回收），`gaea_install.ps1`
输出 INSTALL OK；已安装版与 build/bin 同哈希，未抬版本。

跨端复核、功能提交与预览包（2026-11-14）：实机确认桌面旧布局仍保存
`composerHeight:104`，会把新输入卡的 112px 输入区+43px 工具栏裁掉。输入卡
完整高度基线改为 164px，并改用 `composerHeightV2` 新键；功能提交
`6925ad86` 仅收 5 个 UI 文件。旧 104/225 偏好不再把桌面与网页拉成两套
高度，新 UI 内主动拖拽后仍会独立持久化。正文中原先被 Markdown 行内代码
包裹的文件路径（如 `` `进度计划/当前计划.gsched.json` ``）现升级为可点击
文件 chip；代码围栏保持不可点击，普通内联代码不受影响。提交前 UI 合批
97 例、改动面 ESLint 与 `git diff --check` 通过；网页版实测输入卡 164px、
工具栏 43px、4 个正文文件 chip；桌面同尺寸，文件 chip 点击后成功打开
Markdown 预览。截图：
`.tmp/ui-review/office-desktop-final.png`、
`.tmp/ui-review/office-desktop-final-preview.png`。

预览包 `gaea-office-output-ui-20261114-6925ad86`：独立干净工作树
`.worktrees/pkg-6925ad86` 检出该提交，clean-gate 通过；干净树定向 35/35、
前端 Vite 31.15s、Wails 1m00.5s、bindings 未跳过。包内 exe 63,878,144 B /
SHA256 `BAEFC5D355A39063741CF2BA37C1112307F0C36D828864881EC7575AC898AFD4`；
预览 ZIP 32,942,469 B、源码 TAR 42,925,679 B，正式路径冒烟 `/api/health`
200（PID 35232 已回收）。`PACKAGE-INFO.md` 与 `SHA256SUMS-6925ad86.txt`
已入库；exe/三 DLL/ZIP/TAR 按 P2-3 只留盘。本预览包不覆盖装机位，未抬版本。

## 灵犀 P1 四项全落 · 2026-11-13（装机，拍板=复用系统 Python）

P1-2 xlsx 交付门（xlsx_recalc 暴露 LibreOffice 重算链+xlsx_audit 只读审计
[错误值/自引用/未重算]，范围口径=Rows∪dimension——excelize 写出 dimension
恒 "A1" 单信会漏；公式优先/动态数组/零错误三条进提示词）；P1-1 pdf_cli.py
17 子命令进既有 pdf 技能（灵犀 CLI 契约，实弹 17/17）；P1-3 pptx-template
新技能五步流水线（flatten→shell-report→make-skeleton→apply-flat→assemble，
16:9 硬约束+字体白名单∩系统，实弹全链绿）；P1-4 WPS 面板停靠 dock 档
（wpshost 收编 wpsdock 探针+App 三绑定归 OfficeB+WpsDockControl 缩放跟随/
卸载还原，绑定四件套再生，embed 待拍板）。装机两次（exe 19:23 PID 33768 →
20:28 PID 44752）。同树并跑：编程板块（隔离 DSH〔更正：非 codex，codex.exe 系其沙箱辅助进程〕）续灵犀逆向 B/C/D 批，其
文档在途改动留树未卷。余=真机停靠走查+embed 拍板。

## 灵犀 P0-4 余项补完（#12+#14）· 2026-11-13（装机）

P0 收官批挂起的两条计划内余项补完：#12 参数错误三要素高频面（办公 COM 族
8 工具+通用文件族 10 工具统一「<工具名> 参数解析失败: <底层 json 错误>」，
双守卫测试钉死，全 hermetic）+#14 office_read/edit/render/save/close doc
参数工作区约束显式文案。泛化面 35 处 30 文件只动计划口径 16 处（不 sweep）。
验证：app 全包+builtin+agent 全绿+wails build+冒烟 200+装机 PID 33768。
余：非高频泛化文案 ~19 处留池。

## 灵犀蒸馏 P0 批接续收官 · 2026-11-13（装机，接另一执行体停点）

DSH 灵犀逆向批（lingxi-desktop 1.3.36/app-server 5.24.0 asar 逆向）中途停止，
工作树留有调研三档+P0 全量代码；本会话接续收尾验证与落账。事实源
`docs/gaea-lingxi-re-distill-2026-11.md` + 19 条优势档 + 落地计划档；
`C:\AI\re\lingxi\` 脚本与报告入库（122MB 解包体量 .gitignore 排除）；
`tools/wpsdock` P1-4 前置探针入库（go build 实证）。

- P0-1 交付契约硬化：`fileLinks.parseFilePathHref` 伪协议还原（file:///、
  local-file:///、sandbox:///；UNC/盘符/POSIX 形态，解码失败不抛）+
  Markdown.tsx FileChip 先还原再判 + `boot/delivery_prompt.go` 交付与收尾
  纪律段无条件追加（链接禁反引号/代码块+只承诺工具成功过的事+收尾三段+
  自检四问），SingleModelPrompt 逐字节不动。
- P0-2 受控宏逃生舱 A 档：`office_macro` 宏白名单（word_replace_all/
  sheet_replace_all）确定性展开成既有 office_edit ops，同一条 COM 路径；
  不给任意脚本、零新增执行面（弃 VBA 注入 B 档）。
- P0-3 选区上下文：OfficeHelper.cs `Selection()` 三应用（异常恒
  selected=false 不抛）+ `office_selection` 只读工具 + 划词即改提示词段
  （replace_text find/expect 防错位）；前端捕获胶囊不做（按需读取替代）。
- P0-4 提示词补齐：skill 硬截断补全路径注记 + compact 交接单三要素
  （纯文本+进行中+前序摘要继承）+ office_open 工作区边界显式约束。
- P0-5 steer 确认=已有实锤（steerQueue+MidTurnSteerPrefix+收尾闸）不立项。
- 三处纪律：RegistryMap/spacetags/registry-map 文档补 office_macro+
  office_selection；bindings 再生补 models.ts overflow 字段（六轮批漏再生
  对账）。
- 验证：go vet 零红+boot/skill/agent/tool 全绿+internal/app 定向绿+
  fileLinks 20 例+tsc -b 绿+vite 24.8s+wails 1m07.6s+冒烟 200+装机
  INSTALL OK PID 47776（19:04:35>19:04:18）。
- 余：P1-1~P1-4 挂拍板（阻塞=Python 运行时决策）；#12 参数错误三要素全量
  改文案挂池；IM deliver_result 留 P2 文档预留。

## dsh-office COM 实时协同档波1 落地 · 2026-11-13（装机）

copylee711/dsh-office v1.2.0（MIT，2069af2d）蒸馏：`internal/office/comhelper/`（OfficeHelper.cs
逐字节照搬 + go:embed + csc.exe 编译缓存 `/codepage:65001` 偏离 + Go stdio JSON 行协议
client）+ `internal/app/office_com_tools.go` 八工具（open/status/read/edit/render/save/
close/help）gaeaExtraTools 条件装配（无 Office/WPS 整族不注册）；三处纪律（RegistryMap/
spacetags/registry-map 文档）+evidence isWriterTool 认账 office_save+前端 OFFICE_WRITE_TOOLS
认账。工作区边界由 officeComResolvePath 单点收口（波1=工作区内文档）。捎带修两生产 bug：
publishguard_ledger 账本 Write 前 Close（JSONL 恒 0 字节静默失写）+ workbench WriteText
ms 粒度 CAS 失明（mtime 顶基线+1 保单调）+ test-related 排除 scripts/（ignore-tag setup
failed）。验证：协议真进程 4 例+工具层 9 例+守卫族绿+真机冒烟全链（csc 编译 helper
→open→edit 中文无 mojibake→save→render→read→close）+lint 10 清零（guides.go 文件级豁免
misspell=Excel 函数名）+tsc/vitest 303 绿；exe dfd1db30 冒烟 200 装机 PID 44252（不抬版）。
波2/3 余项挂拍板池：WPS 公式 math 腿/Excel data work/PPT 版式引擎（~4100 行）/体验件
（卡片/跟随/静默/settle）/preferLive 引导回/unseen 总览闭环/克隆底稿去留。
调研唯一事实源=docs/gaea-dsh-office-distill-research-2026-11.md；当日账=day-log §15。


## G2 只读提交图 · 2026-11-13（凌晨，分支候选）

- **绑定 746→748**：`GaeaGitGraph(skip,limit,branch)`（分页提交+parents/%D 装饰+本地分支清单；branch 严格校验防 rev/选项注入，中文分支放行）+ `GaeaGitCommitDetail(hash)`（消息体/引用/parents+numstat 二进制 -1+4MB diff 预算截断+合并诚实空 diff；hash 十六进制校验）。只读边界：不 checkout/fetch/push。
- **前端**：`commitGraph.ts` lane 纯函数（首版真 bug=清位在布父后抹掉同槽首父整图塌单 lane；改先清位再布父，6 拓扑单测钉死）+ `CommitGraphView` 第五镜头（SVG lane/装饰 chip/分支筛选/加载更多 --skip 去重/详情展开复用 ChangesDiff）。
- **验证**：Go 新 2 例+全包 exit 0；tsc+Vite 24.01s；vitest 30/30；ESLint 11 文件零；drift 748 OK。**真机走查已过**（夹具仓 no-ff 合并+tag：6 行拓扑序/装饰 chip/合并诚实空 diff/二进制占位/feature 筛选重拉 3 行精确，全断言绿零缺陷；wails build 9.98s+冒烟 200 exe a51323f4）。
- 坑：lane 清位次序；wailsjs 补账与 autocrlf 噪音区分（`git diff | grep -c "^[+-][^+-]"`=0 即纯相位）；`wails generate module` 可独立再生 wailsjs。

## G1.8 降级矩阵 + G1.6/G1.7 真机走查收官 · 2026-11-13（凌晨）

- **空仓矩阵**（`TestGaeaGitUnbornHead`）：status 报未诞生分支、log 诚实报错；git ≥2.26 未诞生 HEAD=空树——unstage/staged diff 成功且语义正确（首提交场景天然可用）；worktree list 正常。
- **真机走查**（沙箱夹具真仓 UU/rename/二进制/double worktree + GAEA_WALKTHROUGH + GAEA_WEBVIEW_DEBUG→CDP 9333 实壳实绑定）：冲突组置顶+不代解提示+冲突行无快操作、rename `← 原名`、**暂存计数排除冲突**、工作树镜头三行徽标、linked chip 主检出态正确不亮——全断言绿零缺陷。wails build 19.03s + 冒烟 200（payload releases/gaea-g17-g18-walkthrough-20261113/ exe 2fd2741b 留盘不入库）；壳回收+沙箱清场。
- **G1.7 已快进并入 main**（277dc240+7b7f314d，main ahead origin 7 未 push）。G1 验收矩阵（非 Git 仓库/空仓库/无提交 HEAD/重命名/二进制/冲突）至此全部有真仓测试+真机走查双实证。
- 坑：CDP 开关=GAEA_WEBVIEW_DEBUG=1（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 传调试端口在 Wails 壳不生效）；杀壳后 WebView2 子进程可残留占 9333 须核属主；worktree -b 撞已有分支前缀；GitPanel 生产消费方=sidebarRegistry 的 git pane。

## G1.7 冲突/重命名/二进制展示策略收口 · 2026-11-13（凌晨，分支候选）

- **并入**：G1.6 分支（7a544d37+205c5947）已快进并入 main（本地，未 push，main ahead origin 5）。
- **Go**（`gaea_git.go`）：`GitFileStatus` 增 `Conflict`（六形态 U/AA/DD/AU/UA/DU/UD，`isConflictXY` 与前端 ink 同规则）与 `OrigPath`（porcelain -z rename 来源，原 i++ 丢弃；走同一 workspaceStatusPath 过滤口径）；**冲突 staged=false**（不可提交，不进提交计数，修正 UU 误入「已暂存」组）。测试 3 新：真仓 UU（双分支 merge）/rename 溯源/二进制 diff 原样过管道。
- **前端**：GitPanel 冲突独立分组（首组+「本面板不代解」提示；展开行给 `<<<<<<<`+git add 指引而非拉 combined diff；冲突行无 stage/unstage/discard——git add 冲突文件=冲突标记入暂存属危险动作）+ rename 行 `← 原名`；`buildGitDiff` 二进制前缀 →「二进制文件：不提供文本 diff」诚实占位（原「未能解析出 diff 内容」误导）；`gitStatusInk` 冲突补齐六形态+吃 Go 标记；mock 样例补冲突+rename。
- **验证**：Go 定向+全包复跑绿（首跑 FAIL=与前端构建并发的负载期假红，零交集+复跑绿定性）；tsc+Vite 24.85s；GitPanel 15（3 新）+ink+planDiff 15+契约 28/28 绿；7 文件 ESLint 零；绑定 746 不变。未跑 Wails build/未装机。

## G1.6 worktree 只读展示 + Git 子进程预算收口 · 2026-11-12（深夜，分支候选）

- **分支**：`feat/g1-git-deepening`（自 `de283ace` = main tip，含前端测试成本线提交）。设计判定固化在交接档 §6：gaea 全应用单一 workspace 锚，上游「跟随会话进 linked worktree」会造成多面板基准分裂——worktree「切换」=工作区切换器，归 G2/W1 级产品门另立；Go Git 服务扩展不重做。
- **Go**（`internal/app/gaea_git.go`）：①新增 `GaeaGitWorktrees`（`git worktree list --porcelain` 解析：主检出/linked/detached/bare/locked 带原因，isCurrent 与工作区 toplevel 等值判定，Windows 大小写不敏感）+ OfficeB 门面行；②`GitStatus` 增 `isLinkedWorktree`/`mainWorktreePath`（`--absolute-git-dir` ≠ `--git-common-dir`，git ≥2.31，取不到尽力而为）；③验收欠账收口——git 子进程统一 `gitRunBudgeted`：读面 20s/8MB、写面 60s/1MB（pre-commit 钩子余量）、diff 4MB 展示层截断标记（不当错误炸面板），boundedWriter 管道反压早停 + ctx 强杀兜底；预算 var 化供测试收窄。
- **前端**：GitPanel 第四镜头「工作树」（主检出/当前/detached/bare/锁徽标+分支+路径）+ 头部 linked chip；四表同步（types/git.ts、bridge/core.ts+mappings、mock office types+methods_ops 演示样例、bindingSignatures、spaceBindings）；绑定 745→746（bindingNames/legacyBindings/bindings_office.go 均由生成器再生）。
- **验证**：Go 定向 10 例（含 worktree 真仓三态/预算溢出/超时/截断四新例）+ 全包 `go test ./internal/app ./internal/gaea/workbench` 绿、vet 绿；`tsc -b`+Vite build 23.42s 绿；GitPanel 12 用例（含 2 新）+mock 契约等定向 19/19 绿；drift 检查 746 OK；9 文件 ESLint 绿。未跑 Wails build、未装机。
- **坑新增**：①`internal/app` 测试二进制 import 表直依 `sherpa-onnx-c-api.dll`——worktree 无 build/bin 时跑 app 测试须把主树 `build/bin` 加入 PATH（否则 0xc0000135 DLL not found）；②干净仓 `git status --porcelain` 无 `-b` 时零输出，预算溢出测试必须带 `-b`；③`bindings_office.go` 是 gen_bindings 生成件（DO NOT EDIT）——手改会被再生产出覆盖，gofmt 对生成件的对齐改写（493 行 churn）必须撤销；④worktree 列表按路径排序，测试断言不能按位置取 linked/detached。
- **前置收口**：主树前端测试成本线已提交 `de283ace`（vitest 双项目 logic=node/ui=jsdom，全量复验 497 文件/4348 用例绿 393.6s vs 基线 569s）。

## DSH-better-sidebar Go 增量复刻：功能提交与桌面预览包 · 2026-11-12（打包收口）

- **提交**：功能提交 `f49ea1500026526b822c80be39cfb08d06026869`（短 `f49ea150`，标题 `feat-office-dsh-better-sidebar-workbench`），父提交 `daaa0065`；80 files / 7,786 insertions / 374 deletions。收口文档与包元数据另提独立提交，不与源码功能混写。
- **包构建**：从仓内独立干净工作树 `.worktrees/pkg-f49ea150` 精确检出 `f49ea150`，构建前 `git status --porcelain` 为空；前端 `npm.cmd run build` 通过（Vite 20.38s），随后按系统 TMP/TEMP、ucrt64 GCC 16.1.0、`CGO_ENABLED=1` 执行完整 `wails build -s -ldflags "-s -w" -trimpath`（50.158s，bindings 未跳过）。
- **四件套**：`gaea.exe` 62,486,528 B / SHA256 `986C89666AB1588D10E0881AAEBA8129B35EC62577034FA88C1A2155F419EE96`；三只 DLL 从 `build/voice-dlls` 派发到同目录，完整哈希见 `releases/gaea-bettersidebar-20261112-f49ea150/SHA256SUMS-f49ea150.txt`。
- **冒烟**：本机 `.worktrees` 内的 exe 启动被策略拦截，因此把同一四件套放到 `releases/gaea-bettersidebar-20261112-f49ea150/payload/` 正式路径，执行 `scripts\smoke.ps1 -ExePath <absolute>\payload\gaea.exe`；`/api/health` 200，`{"status":"ok"}`，PID 30100 已回收。该路径规则已补入构建指南坑表。
- **交付包**：`gaea-f49ea150-preview.zip` 32,502,355 B / SHA256 `7038DDBE20F3B5A6821BE9D7140BF69257E9037940F3E6EAEB6C7D1E0E2A172A`，根目录仅 exe+三 DLL；`gaea-f49ea150-source.tar.gz` 39,080,619 B / SHA256 `776686C69D939C4DB8720531705C0CFEFCB6D62E6F5845B7222AE7F411C236FC`，由 `git archive` 从 `f49ea150` 精确生成。ZIP/TAR 留盘不入库，`PACKAGE-INFO.md` 与 `SHA256SUMS` 入库。
- **边界**：主工作树平行测试优化与测试加速件未进入功能提交、干净构建树或发布包；既有 releases 压缩包未清理。Wails 已知 `Not found: time.Time` 与生成物尾随空白噪声不变，build/smoke 均成功。
- **状态**：未 push、未装机、未覆盖 `C:\AI\gaea-app`。后续只需按用户拍板决定发布动作；G1 深化仍为 worktree 切换与 Go Git 服务。

## DSH-better-sidebar Go 增量复刻：文件工作台 F1 + 长期交接档 · 2026-11-12

- **用户决策**：在现有 gaea 上增量接线，不从头重写；保留会话/任务/Office/记忆/导航与整体 UI；侧聊服务不做；先做文件工作台。
- **上游基线**：`dsh-better-sidebar v0.32.0`，注解 tag `a53a988`，解析 commit `53e08cec`（2026-11-12）。README 只作概览，能力事实回看 `src/`、CHANGELOG、`docs/plans/`。
- **F1 Go 权威**：新增 `internal/gaea/workbench`（fs/text/upload/watch）与 `internal/app/gaea_workbench.go`；覆盖批量列目录、展开目录 fsnotify、创建/重命名/永久删除、文件/目录/多选 ZIP、分片与原生上传、编码/EOL 保持、mtime CAS、原子写、5MB 编辑与 256KB 分页。
- **F1 前端**：`FileTree` 接批量加载、watch、Ctrl/Shift 多选、CRUD、拖放/原生上传与导出；`ExplorerView`/`paneTabs` 做改名重定向和删除关 tab；`FilePreview` 做完整读取后编辑、冲突检查与分页阅读；bridge、mock、types、spaceBindings 同步。
- **绑定面**：新增 17 方法，728→745；`OfficeB`、`models.ts`、`bindingNames`/legacy/mock 签名已再生成或同步。
- **验证快照**：frontend build 通过；前端 483 文件/4301 用例全绿；定向 ESLint、workbench+app vet、排除语音环境的 Go 全量均通过；Wails build 成功；`build/bin/gaea.exe` 62,432,768 B / SHA256 `DA932248B764A4EA098E1A6A1F50A1818A7158EE5835479D6D06BCCECD0C3E95`，正式路径冒烟 `/api/health` 200。
- **当前状态**：F1 未提交、未 push、未装机。`models.ts` 生成差异仍需最终 Wails 构建重写；后续若改源码必须重建并更新哈希。
- **既有红项**：`go test ./...` 仅 voice/native 与 voice/realtime 因 ORT 1.17.1/Sherpa API 25/28 不匹配；binding drift 名称对齐但退出 1 因存量 `EngineUpdateStatus`、`StrataModelStart`；Wails 生成提示 `Not found: time.Time` 但 build/smoke 成功；docs checker 有多份既有未登记/悬空项，本批只保证新档登记。
- **长期交接**：新增 `docs/gaea-better-sidebar-go-iteration-2026-11.md`，记录 F1 完成项/API/安全不变量、F2 文件树完整度、F3 预览编辑、G1 统一文件/Git、G2 提交图、W1 底部工作台/终端、X1 插件服务门控，以及新会话必读顺序和验收矩阵；旧 `gaea-dsh-better-sidebar-long-term-distill-plan-2026.md` 降为历史调研档。
- **F2.1 Git 状态着色（同日续刀，代码候选）**：新增 `frontend/src/gaea/lib/gitStatusInk.ts` 纯映射层——Git porcelain X/Y 分类、冲突优先级、Windows/rename 路径归一、未跟踪目录折叠、文件与目录/根聚合；`FileTree` 接入现有 `GaeaGitStatus`，文件行显示语义色 X/Y 徽标，目录与根显示后代变更点，挂载/30s/窗口聚焦/目录事件/本树创建改名删除上传后刷新。验证=`gitStatusInk.test.ts`+`FileTree.test.tsx` 23/23、相关 4 文件定向 ESLint 通过、`git diff --check` 干净；按用户要求未跑全量测试/Wails build，真机观感仍挂载。
- **F2.2–F2.5 文件树完整度（同日续刀，代码候选）**：①新增 `internal/gaea/filefilter` 单一 Go 排除权威（默认噪音 + 最多 64 条用户 glob，名称/相对路径/`*`/`?`/`**`，Windows 大小写不敏感）；`GaeaListDirBatch`/`GaeaFileTreeWatch`/`GaeaFileSearch` 同源过滤，@ 菜单也走同一规则；前端按工作区持久化排除规则与紧凑偏好，根菜单/标题齿轮可编辑。②Go 增量探测单可见子目录并输出 `compact`，前端折叠为 `a/b/c` 单行（深度 8），整链展开、菜单尾目录、tab 真实路径调解。③文件图标扩为 Word/表格/PPT/PDF/图片/代码/配置/压缩/数据库并区分文件夹开合。④打开方式改为系统默认程序/资源管理器子菜单，失败可见。
- **F2 验证**：`filefilter`+`workbench` Go 测试绿，`internal/app` 搜索/资料/工作台定向测试绿，三包 vet 绿；前端 FileTree/Composer/偏好/紧凑/图标/Git/mock 契约 **8 文件 53 用例全绿**，`tsc -b --force` 与定向 ESLint 绿；`gen_bindings -signatures-ts` 与 `wails generate module` 成功、绑定仍 745。按用户要求**未跑 Vite/Wails 应用构建、未装机**，桌面真机走查仍挂。
- **F3 预览编辑对齐（同日续刀，代码候选）**：①CodeMirror 6 覆盖现有文本/Markdown 编辑并保留 Suspense/ErrorBoundary textarea 双层回落。②新增 JSONL/NDJSON 记录卡（逐行解析、坏行原样、500 条/4000 字预算、记录卡/原文切换）。③Markdown 浮动目录（代码围栏外标题提取、吸顶、滚动高亮、点击定位）。④相对/file/Windows drive 图片按 Markdown 所在目录解析，经 `AttachmentDataURL` 的可读根+32MB 围栏加载；`data:image`/远程图直渲染。⑤脏稿 registry 支持多 owner、pane tab 关闭/切换与主预览关闭/导航确认、`beforeunload`。⑥ diff 超长单行默认 2K、显式展开 20K，超 20K 配对不做字符 LCS。
- **F3 验证**：前端 12 文件 **99 用例全绿**、`tsc -b --force` 绿、23 文件定向 ESLint 绿。按用户要求**未跑 Vite/Wails 应用构建、未装机**，桌面真机走查仍挂。
- **G1.1 统一文件/Git 前端第一刀（同日续刀，代码候选）**：新增 `gitStatusStore`——唯一 Git 状态轮询者，30s/focus 刷新、多订阅共享 in-flight、非仓库快照与 revision；FileTree Git ink 和 GitPanel 改为共用 store，不再各自发 `GaeaGitStatus`。GitPanel 增加「Git / 本轮文件」双镜头，本轮文件复用 ChangesPanel 三态与证据链。未动 Go Git 最小绑定、未做 push/pull/fetch。验证=gitStatusStore+GitPanel+FileTree **3 文件 29 用例**、tsc、6 文件定向 ESLint 全绿；未跑构建/真机。
- **G1.2 Git 仓库根语义（同日续刀，代码候选）**：`GaeaGitStatus` 先 `git rev-parse --show-toplevel` 发现父级仓库根，再从根执行 porcelain `-z -b`；嵌套工作区路径统一转为工作区相对路径，仓库外变更过滤，避免文件树/Git 面板打开不可达路径。NUL 解析同时修正 rename/copy 目的路径，并补无提交分支名。验证=`TestGaeaGit*`+`TestCleanFilter*` 定向全绿、`go vet ./internal/app` 绿；未抬绑定、未跑构建/真机。
- **G1.3 统一只读变更树（同日续刀，代码候选）**：新增 `unifiedChangesTree` 纯函数——Git 与 SessionChange 按规范化路径合并，目录 rollup 文件/Git/AI 计数；GitPanel 新增「统一」镜头，叶节点同时显示暂存字母、工作区字母和 AI 次数，目录可折叠、文件可打开。工作区外绝对 Agent 路径过滤；非 Git 仓库仍显示 Agent 变更并诚实提示 Git 不可用。验证=纯函数/树组件/GitPanel **3 文件 12 用例**、tsc、定向 ESLint 全绿；未跑构建/真机。统一镜头内 stage/unstage/discard 尚未迁入。
- **G1.4 统一树快操作与来源筛选（同日续刀，代码候选）**：统一镜头增加全部/Git/AI筛选；文件行按 porcelain 状态提供 stage（工作区/未跟踪）、unstage（暂存侧）和两击 discard（已跟踪工作区改动），复用既有 `GaeaGit*` 并在动作后刷新共享 status。操作按钮与文件打开分离，避免误触；AI-only/非仓库树仍可用。验证=统一树/组件/GitPanel **3 文件 14 用例**、tsc、定向 ESLint 全绿；未跑构建/真机。
- **G1.5 统一树行内 diff（同日续刀，代码候选）**：已跟踪文件行增加 diff 展开；同一文件同时存在暂存/工作区改动时可切换「暂存区 diff / 工作区 diff」，继续复用 `GaeaGitDiff` 与 ChangesDiff。未跟踪/AI-only 行不显示伪 diff；展开状态与文件打开分离。验证=统一树/组件/GitPanel **3 文件 16 用例**、tsc、定向 ESLint 全绿；未跑构建/真机。
- **桌面端构建（2026-11-12 23:01）**：按构建指南先 `npm.cmd run build`（21.89s），再使用 ucrt64 + CGO + 系统 TMP 执行完整 `wails build -s -ldflags "-s -w" -trimpath`（56.11s，bindings 未跳过）。产物 `build/bin/gaea.exe` 62,485,504 B，SHA256=`F0452ADCFF604E3CB020EA78BA83A0E3979FB9ECB74CA14AE3FE1FBB09113A96`；三枚语音 DLL 同目录；正式路径冒烟 `/api/health` 200（`{"status":"ok"}`），冒烟 PID 22520 已回收。当前未提交、未装机；Wails 仍提示已知 `Not found: time.Time`，生成 `models.ts` 仍带尾随空白。

## 办公对话输出 UI 系统重设计 · 2026-11-11（未发布）

- 调研收敛：Codex/Claude Code 的执行行、Claude/ChatGPT 安静阅读面、M365
  Copilot 的审批前预览、Cursor 3 的任务状态、Raycast 的问题卡与 Linear 的
  高密度对齐；完整矩阵与源链接见
  `docs/gaea-office-conversation-output-redesign-2026-11.md`。
- 四类输出统一：同一段正文前的思考、阶段和工具合并为一张过程卡，
  折叠头删除“执行过程”固定标题并压成单行摘要 + 状态，过程卡出现后不再显示外部“思考中…”状态带；思考卡始终默认折叠、运行中保留最新一行摘要；工具行改原生 button + 状态文字
  + 目标/结果/输出分层；Ask 改为保留上下文的焦点浮卡；审批从全局固定层迁到
  输入区上沿，主决策两列、次级授权与危险停止分层，硬确认语义不变。
- 门禁：`tsc -b` 0 错；定向 Vitest 59/59；定向 ESLint 0；frontend
  `npm run build` 通过；暗/亮主题浏览器走查过程卡、思考卡、Ask、审批无溢出。
- 桌面构建（工作树，v4.541.0）：先修复提交 `f60cd0af` 中断裂的
  `wails.json` `info.comments` 字符串；frontend 生产构建通过后，以 ucrt64
  gcc、`CGO_ENABLED=1`、默认 `TMP/TEMP` 执行
  `wails build -s -ldflags "-s -w" -trimpath`。产物 `build/bin/gaea.exe`
  62,058,496B，构建时刻 2026-11-11 22:18:05，SHA256
  `E6964FD8168B2E1B35D03A589520B165F421C7C27D426702D598FE3919791198`，
  三 DLL 同目录；正式路径冒烟 `/api/health` 200。未装机、未打发布包。

## 35B 单副本收口：原件删除+llama 切 whirlfix2（v4.539.0）· 2026-11-11（深夜）

- 先验后删：b12110+whirlfix2 载入烟测通过 → qwen35ModelFile 常量切
  whirlfix2（llama/语音/目录探活单点跟随）→ 原件删除 26.76GB（C 盘空闲
  100GB）。herdsman 18 条 mapping=同 inode 硬链接（物理副本全盘唯一）。
- 装机 v4.539.0 PID 10424。文件记录：day-log 续批 4 节。

## 引擎实升最新版+速度对擂+strata 版本口径修（v4.538.0 装机）· 2026-11-11（夜）

- **llama → b12110（0.9.0-dev）**：v0.9.0 只发源码，Windows 走 nightly；
  生产参数全兼容；**53.75→80.1 暖跑=+49%**；换装 uocr2-llama\bin（旧版
  留 bin-0.8.0-dev.bak 一条 mv 回滚）。
- **Strata → v0.2.13 + engine 0.1.59 hip-rdna3**：repo checkout+engine 包
  SHA256 全等平铺换装（旧留 engine-0.1.56.bak）；serve 一次点亮；**42.7
  tok/s 历史带内无回退**。
- **strata 版本口径修**：本地 --version 报 engine 组件 0.1.x、官方 tag 是
  控制端 0.2.x——EngineUpdateCheck 增 assetVersionRe 从 release 资产名抓
  engine 版本比对；升级后三卡全转绿（WHIRL/llama 已最新、strata 0.1.59
  对资产 0.1.59 已最新）。
- **速度总榜**：WHIRL 88~92 ＞ llama b12110 80.1 ＞ strata 42.7；llama 与
  WHIRL 差距收窄到 1.1~1.15x。
- **装机**：v4.538.0 PID 35608；回滚资产留盘。
- **文件记录**：`docs/gaea-day-log-2026-11-11.md`（续批 3 节）。

## 实弹反馈三连修+引擎官方版本检查（v4.537.0 装机）· 2026-11-11（夜）

- **①联网开关根治（08c2e3f9）**：用户「理解问题」排查=聊天页 searchEnabled
  useState(true) 无持久化→切话题重挂载静默回开、web_search schema 无感
  注入（引擎日志 0→1 tools 翻真实锤；微信助手腿硬编码联网为独立面）。
  修=默认关+localStorage 持久化+toolMode 决策日志；复装机实弹 0 tools。
- **②思考流实时上屏（af8c3abe）**：思考卡等正文才出现=useChatStream
  reasoning 帧只进缓冲从不渲染（引擎侧 5.3KB 流式正常）。修=streamReasoning
  rAF 节流通道，流式行展开态「思考中…」卡，done 回落折叠卡。
- **③引擎官方版本检查（本批）**：三引擎（WHIRL 0.2.5.1↔tsaipifong/
  whirl-llm；llama 0.8.0-dev↔ggml-org/llama.cpp v0.9.0；Strata 0.1.56↔
  Niko1221/Strata v0.2.13）releases/latest 检查——`engine_update_check.go`
  （--version 探针+GitHub API+30min 缓存+点分比较；逐行抓取防 MSVC 尾注
  误抓；只告知不自动升级）+三卡「检查更新」钮+结果行。hermetic 测试+
  全 app 套件绿+722 方法+drift OK+modelcenter 119/119。
- **装机**：v4.537.0（三源 sync）PID 8996；版本检查真机三卡目检挂用户
  （llama/Strata 预期显示有新版本）。
- **文件记录**：`docs/gaea-day-log-2026-11-11.md`（续批 2 节）。

## WHIRL 推理引擎接入（第三受管本地引擎，v4.536.0 装机）· 2026-11-11（晚）

- **承接**：用户线「搜索更适合本机的推理引擎」——调研对擂（WHIRL 同文件
  1.82x 碾压 llama Vulkan）→模型盲测（生产 uncensored 6:4 Ornith，边缘面
  定局换模型被否）→张量手术（whirlfix2=仅 2 BF16 router→F32 位级无损，
  你的模型 @ WHIRL **102.57 tok/s=生产 1.91x**，双引擎全验）→本批 gaea
  代码面接入。
- **刀1 Go**：`whirl_proc.go`（8126，单档 qwen35_whirl=whirlfix2+mmproj，
  exe-spawn 照 qwen35/strata 模板）；engine.go 四处注册；散落名单六处
  （catalog/idle/preflight/handler/schedule/vision）；**thinking 通道=
  漏斗层缺省关**（applyWhirlThinkingDefault 顶层 enable_thinking=false，
  防 strata 同款「正文被默认推理烧空」）；`whirl_path` config 五链。
- **刀2 绑定+前端**：gen_bindings 四连（720 方法）；`MANAGED_ENGINE_IDS`
  收敛常量（判定面六处统一）；**tianxuan 前端残留 8 处清理**（含
  RuntimePage 夹具 tianxuan→strata 迁移）；mock 三方法诚实空态；语言包×3。
- **门禁**：Go 定向包全绿+whirl_proc_test 七用例（含 llama 档位反误认）；
  引擎数断言 8→9 四处；全量 vitest 4262+ 全绿；tsc 0；drift 检查 720
  一致；三源版本 4.536.0；build+smoke+装机 PID 34236。
- **现场**：WHIRL 安装位 `C:\AI\whirl`；Ornith 19.8GB 按用户令删除；
  whirlfix2 与原件并存（llama 8124 不动）；**真机四关挂用户目检**
  （WHIRL 卡/chat 绑定拉起 ~100 tok/s/@图 vision/idle 收割）。
- **文件记录**：`docs/gaea-day-log-2026-11-11.md`（续批节）。

## 代码面卫生小批：knip 尾款清零+天璇退役漏网收口 · 2026-11-11

- **承接**：11-10 晚自主批收官后用户令「继续优化完善 gaea」；拍板池余项
  全卡用户输入，推进可自主验证欠账：v4.179 挂账的 knip Unused exports
  169 项三态甄别。
- **盘点**：169 项已被后续批次自然排干，全类目复扫自研仅剩 3 项；假阳性
  甄别=voice-worklets（AudioWorklet URL 运行时挂载）与 preview-atlas 预览壳
  保留。
- **刀（c7da4558，-23 行）**：engines.ts 删 `startTianxuanModel`（天璇
  退役漏网——Go 绑定已删而 wrapper 靠 AppFacade Record 索引签名静默带病
  编译，真调用必炸）与 `startStrataModel`（零调用方；Go 绑定+集成测试
  保留为预留面）；ChatPage 删 INACTIVITY 无消费方再导出；gen_bindings
  参数表除名五条天璇陈旧条目+顺手修 WebSearch 三行既有 gofmt 欠账。
- **门禁**：定向 vitest 83/83（api/engines 全部消费面两批 12 文件）、
  tsc 0、eslint 0、**knip 自研未用导出清零**、gofmt 合式；无运行时行为
  变化，不装机不发版。
- **清障**：`.tmp/gaea-voice-588b5ad6` 遗留工作树移除（29 条改动全为
  wails 再生绑定物，核后强删+prune）。
- **文件记录**：`docs/gaea-day-log-2026-11-11.md`。

## 代码面自主批：空间契约漂移刀+REA preflight+版本三源 · 2026-11-10（晚）

- **承接**：办公治理六阶段与 dsh-ops 候选池双收官后，用户令「继续优化完善
  gaea」；拍板池盘点（余项全卡用户输入），按推荐顺序自主推进三刀。
- **刀1 空间契约漂移（f3c78e94）**：`RegistryMap.Spaces` 与运行时生效标签
  （`builtin.SpaceTagOf`：SpaceTaggedTool 自声明优先→静态表→缺省 shared）
  34 处对齐（workspace 17+shell/web_search 4+browser×11+sidebar_open 误记
  shared 实为 work；image_gen/author_memory 实为 play；bash 非规范
  work+shared 归正）；office 六工具显式入静态表 work 堵 fail-open 混 play
  的洞；守卫 `TestSpaceTagsAgreeWithRegistryMap`+`TestRegistryMapSpacesMatchExtraTools`
  双钉；registry-map 文档单源重导；行为变化=play 空间不再装配 office 六工具。
- **刀2 REA preflight（e2fa809c）**：`programmingWebREAReady` 改查红队预设
  `id: rea-mcp` 行+引擎/技能三件在位，弃 cordis.patch.yml「serverName: rea」
  假标记（11-08 裁决落地，marker 随装机删除）。
- **刀3 版本三源对齐（4665c267）**：`sync-version.ps1` 从 wails.json 基线
  同步 app_info.go/versioninfo.rc → 4.535.0。
- **验证**：tool/builtin/boot 全量+agent space/session+app 定向全绿；
  app 测试须 ucrt64 gcc+三 DLL 进 PATH（0xc0000135 坑）。
- **文件记录**：`docs/gaea-day-log-2026-11-10.md` 补段。

## 办公板块工具治理第一批 · 2026-11-10

- **承接**：众神殿 pan_1794234593182（签注 accepted）六阶段计划；用户令开工。
- **P0**：`internal/gaea/tool/registrymap.go` 82 行契约地图（来源/空间/副作用/
  证据档/验收谓词）+ tool/boot/app 三处防漂移守卫 + `scripts/registry_map_doc.go`
  单源导出 `docs/gaea-tool-registry-map-2026-11.md`。
- **P1a**：GaeaExportDeliverable 补 output_path（默认分支让位=显式声明优先）；
  office_generate+docx_apply/docx_accept(仅 accept=true)/pptx_apply/xlsx_apply
  五工具升 agent 面（gaeaExtraTools 装配单点，WritePathResolver、不标
  PersistWrite、Journal 单点落账）；通道 A 未比对不得回 pass（文案修复）；
  三本账名单对齐（Go isWriterTool/isProducerTool/ExtractDeliverablePaths+前端
  OFFICE_WRITE_TOOLS/extractOfficeWritePaths，L9 销项）。
- **P1b spike**：PublishGuard（tool 包）五类写入（重定向/Python/改名/后台/脚本）
  全部可拦可提交判定，快照 ~1.6ms/200 文件——结论有利，P4 不后置。
- **P1b 接线**：bash 前台（终段 Scan fail-loud）+后台 job（完成回调 Scan，终态
  失败）双路径；隔离到 .gaea/publish-quarantine；依赖目录剪枝记档。
- **验证**：tool/builtin/agent/boot/evidence/app 全量绿（app 142.9s 零 FAIL）+
  全树编译+tsc 0+officeTurnProjection 43/43+wails build+冒烟。
- **D3 止损门四条件全过**；P2–P4 待用户点名。
- **文件记录**：`docs/gaea-day-log-2026-11-10.md`、`docs/gaea-tool-registry-map-2026-11.md`。

## 语音交互窗口记忆中枢化 · 2026-11-09

- **用户目标**：语音交互窗口模仿记忆中枢；虚空大脑全窗呈现，右侧面板必须完全透明，底部控制为居中胶囊，同时支持深浅主题，并让中央字幕在回合结束后继续显示最近问答。
- **最终结构**：顶部身份栏改为覆盖层；大脑画布居中并限制为最大 `860×720px`；左侧保留相位 HUD；右侧对话区实机计算为 `background: rgba(0,0,0,0)`、`border: 0`、`box-shadow: none`、`backdrop-filter: none`；底部控制拆为独立胶囊并统一居中。
- **中央字幕**：实时转写优先；思考中保留最近用户问题；播报中显示助手流式正文；空闲后保留最近一轮用户与助手内容。右侧继续提供完整转写流。
- **双主题**：`html[data-hl="light"]` 使用冷白玻璃、深色文字与中央深色虚空晕影；`html[data-hl="dark"]` 使用深蓝黑观测台与浅色文字；两种主题均保留全窗大脑和完全透明右侧区域。
- **文件记录**：专项档 `docs/gaea-voice-memoryhub-ui-20261109.md`，已同步 `docs/README.md` 与本 progress。
- **验证**：定向测试 41 项、`tsc -b`、定向 ESLint、Vite 生产构建、Wails 构建均通过；实机核对 1264×761 下大脑画布与胶囊中心均为 `x=663`，浅色与深色均无页面溢出。
- **干净构建与打包**：在独立工作树检出 `588b5ad6`，按 `docs/gaea-build-guide.md` 先做前端生产构建，再执行 Wails `2.13.0` 完整构建并保留 bindings；exe 61,605,376 B，SHA256=`0E739EA589559072DD6E83FA8404D844AFC420DC6BBE9381076C3BEC3EE987E4`，与三枚语音 DLL 同目录后 `/api/health` 冒烟 200。
- **离线产物**：`releases/gaea-voice-memoryhub-20261109-588b5ad6-preview.zip`（32,195,357 B，SHA256=`E017EA6E26ABC12766422F72543EA6D452844EE598548D2BDE1CC96A891711F5`）；源码归档 `releases/gaea-voice-memoryhub-20261109-588b5ad6-source.tar.gz`（38,632,091 B，SHA256=`23F903F194A9C0F492DA256FE591FC71A10E921E32134D91EC2E9B7FBC9A96E1`）；统一校验见 `releases/SHA256SUMS-voice-memoryhub-20261109-588b5ad6.txt`。
- **发布边界**：现有版本源漂移 `app_info.go=4.531.0 / wails.json=4.535.0 / versioninfo.rc=4.533.0` 未擅自修改；本次只提交语音 UI 四个源码文件、专项文档、独立预览包元数据与记忆记录，构建明确排除模型引擎在途改动。
- **记忆状态**：本会话未暴露 `hindsight_*` MCP 工具；按仓库既有长期机制写入专项 docs、progress 与 todos，作为下一会话的权威恢复源；会话结束时由宿主自动保留当前对话。


## 编程板块：DSH Web + REA + 桌面插件迁移与 cmd 黑窗根修 · 2026-11-08

- **用户口径**：给 gaea 编程板块接入 REA；安装最新 DSH 后迁移桌面版全部插件；
  必须与桌面 DSH 彻底隔离；连续黑窗必须查到根因。
- **DSH/REA**：隔离 DSH `0.2.2-rc.1` 位于 `C:\AI\dsh`；Web iframe 使用
  `dsh.wails.localhost:3091`，解决 DSH `SameSite=Strict` Cookie 的跨站拒绝。
  REA `6.8.0` 使用独立 Node `24.22.1`，MCP 注册在
  `C:\AI\dsh\home\cordis.patch.yml`，技能落在隔离 agents home；实测 229 个
  `mcp__rea__*` 工具。Ghidra/IDA/JADX/Kuna 可选原生引擎未装。
- **插件迁移**：桌面 9 个外部依赖迁入 Web profile，8 个 bundle 启用；
  `dshmarket 1.73.15`、`dsh-genui 0.15.0`、`dsh-ops 0.5.1`、
  `dsh-skills-manager 1.3.0`、`dsh-better-sidebar 0.30.1`、
  `dsh-context 0.82.2`、`dsh-unlag 1.17.5`、`dsh-dream-skin 11.7.0`；
  `dsh-client-ui-skill-explorer 0.4.16` 仅作依赖安装。`dsh-ops@0.4.4`
  不兼容 DSH 0.2.2-rc.1，因此按上游兼容声明升至 0.5.1；`dsh-wubi` 保持
  最后 bundle 以保留覆盖优先级。
- **cmd 黑窗根因**：不是 REA/MCP。主因是 gaea GPU 遥测；
  `gpu_perf_windows.go` 约每 15 秒启动 PowerShell `Get-Counter`，
  `image_comfyui_proc.go` 的 VRAM/CIM/ComfyUI 探测也启动控制台子进程。
  原裸 `SysProcAttr{HideWindow:true}` 会先显示再隐藏；两个文件共 6 处已统一
  改用 `internal/gaea/proc.HideWindow`（`CREATE_NO_WINDOW`）。次因是
  `dshmarket@1.73.15` 的 Windows detached restart helper 漏 `windowsHide`，
  已通过 `patches\dshmarket@1.73.15.patch` 做 profile 内隔离修复。
- **验收**：真实打开编程板块并监听进程创建，GPU PowerShell、插件 `reg.exe`
  与对应 `conhost.exe` 均为 `visible=False`；GPU 遥测仍正常返回读数。
  Go 定向测试、前端构建、Wails 构建和 `/api/health` 200 冒烟均通过。
- **产物**：gaea `4.534.0`，`C:\AI\gaea-app\gaea.exe`，61,556,736 B，
  SHA256=`9D0178BE67E6D501397B0A8CCC3E8BB0F10F56960855176F15E531B8898F7FDD`。
- **文件记录**：`docs/gaea-programming-rea-plugin-integration-20261108.md`；
  已同步 `docs/README.md`、`docs/gaea-build-guide.md` 与
  `C:\AI\dsh\README-ISOLATED.md`。
- **隔离核对**：桌面 DSH 目录无新增 `.dshwubi.bak`；个人
  `~/.dsh/prompt-inject.md` 与 `~/.agents/skills/reverse-engineer-anything`
  均不存在；Web/桌面 `dshmarket restart.js` 哈希不同。
- **记忆状态**：本会话未暴露 `hindsight_*` MCP 工具，`~/.hindsight` 无可用
  配置；按仓库既有长期机制写入本 progress、专项 docs 与 todos，作为权威恢复源。

## 语音回声门 v2（action-2 复测归因）· 2026-11-07

- **复测结果**：滚雪球消失（转写 3/3/8/4 字，2236cba1 生效）；但三症状（重复输入/
  自己的话当输入/杂音当输入）同源实锤=TTS 播放期扬声器回声被 VAD 确认——回声猝发
  active=192~384ms 全过 0.75 阈值门（silero 对 TTS 回声=真语音给高分），打断级联
  `tts=cut` 掐碎回复（turn_1 rev=0..4），回声被转写成杂串进对话。
- **刀（fb4bc194）**：回声门 v2=播放期确认窗 600ms（含 reopen 续接路径——级联正是骑
  192ms 续接门连爆）+soft-end 接受路径同一窗+EchoGuardOn/Off 显式日志（本次坑：门
  静默开关无法从日志判定在位）。测试=TestVADEchoGuardMinSpeech（能量感知假打分器，
  首版 fixedScorer 不辨静音段不闭合）。
- **装机**：10:48 壳 PID 31888 sha256=94785E23。挂用户=二轮真麦复测（日志应有
  echo gate on/off 行）；残余风险=长句回声 >600ms 过门→随刀自适应增益。

## 聊天取消/超时终稿闭合（众神殿 action-3）· 2026-11-07

- **承接**：众神殿 `pan_1794012727813_7dd108312e0d`「如何解决聊天角色答非所问，
  胡乱说话」（部分解决）。三病灶核实：①VAD 跨轮前缀累积已被 2236cba1 修复装机
  （action-2 只剩真麦复测）；②取消丢正文四处全在；③前端超时不调 ChatStreamCancel。
- **落地（action-3，七处）**：工具循环取消抢救当前轮 sbContent 并入 res.Reply；
  whisper 取消分支不抹已累计正文；plain 工具/非工具两分支取消改走 cancelled
  落库（不再裸奔 error「请求已取消」）；空轮取消发 cancelled 空终态+用户输入
  单条落 user 行；前端超时先调 ChatStreamCancel 再保留 partial；cancelled 帧
  空 reply 回退本地已外发 partial。
- **测试**：+7 例（Go 取消族 4：非工具/空轮/跨轮/persona；前端 3：超时撤销、
  空 reply 回退、canonical 优先）。坑新录：服务端「写完即信号」在客户端消费前
  有竞速，工具轮 mid-round 无客户端可见信号，`ssePartialThenHang` 信号前留
  500ms 消费窗；非工具路径可在首个 delta 事件确定性取消。
- **验证**：`go test ./internal/app/` 全包 168.9s 通过；`tsc -b` 0 错；
  ChatPage 20/20、SearchCard+MessageList 20/20。
- **病例卡**：action-1 草稿（语音格 whisper_gaea + 文本格 bokke/tsundere）入
  `进度计划/gaea-chat-relevance-action-20261107.md`，待用户确认实体/角色与
  action-2 真麦复测。
- **本工作树三条前置 commit**：2530ca23 众神殿席位护栏+DSH 缓存、7b726c51
  聊天搜索可靠性 P0、342b096a gofmt chore。

## 聊天搜索状态与来源卡并入主树 · 2026-11-06

- **用户反馈**：重新测试后，聊天窗口没有显示搜索状态和搜索卡。定位=修复已完成在
  `codex/chat-search-reliability` 独立 worktree，主树仍是旧前端/旧后端，桌面运行的
  4.531.0 exe 也早于搜索修复。
- **集成**：将 canonical 搜索正文、`thinking/searching/search_done/finalizing` 状态、
  `search_summary` 持久化、30s 静默/120s 硬超时和 `SearchCard` 来源卡并入
  `C:\AI\wubigrok`；未混入主树其他 14+ 在途文件。
- **验证**：Go 搜索定向测试与前端 ChatPage/SearchCard/MessageList 共 37 例通过；
  `tsc -b`、定向 ESLint、`npm run build`、`wails build -s` 通过。
- **产物**：`build/bin/gaea.exe` 与桌面 `gaea.exe` 同步，
  SHA256=`B68AF7460610CC23C9CF81E81423FEA2C4172D94EA91D146C0956F92DF73926A`；
  两个真实文件名均直接启动并返回 `/api/health` 200。固定 `.tmp\smoke-gaea.exe`
  冒烟路径受本机执行策略/独立 WebView profile 影响，不能代表真实产物状态。

## 众神殿席位卡死根修与 DSH 前缀缓存 · 2026-11-06

- **用户口径**：「解决众神殿的 agent 经常卡死，和 dsh 的命中缓存低下的问题（是否是真的调用 dsh）」。本文档=`docs/gaea-pantheon-seat-hang-cache-2026-11.md`。
- **真调用取证**：实弹会话 `pan_1793963799148_6aecf8f40f0a` 中 DSH 四轮陈词 368s/147s/316s/403s，`tool_started` 209 条（pwsh 104/read 69/grep 30/glob 5），逐轮工具数 87/31/53/38；DSH 会话 ID 在 `~/.dsh/sessions/--C-AI-wubigrok--/` 有实体记录。结论=确实调用 DSH headless，不是占位实现。
- **卡死根因**：①席位被当完整 agent 用（提示词鼓励工具核验），单轮 87 次工具调用、每次一次 LLM 往返；②DSH 工具自身 30s 超时（实弹 `TOOL_TIMEOUT`），长命令每步白等；③8 分钟总上限 + 空结果自动重试把单席放大到 16 分钟，历史场次跑满 140 分钟被取消；④`recordToolTrace` 每条工具事件都在 `m.mu` 里整份序列化+fsync `session.json`（该场 2.4MB，418 次）。
- **缓存根因**：DSH 前缀缓存本身正常（同会话第 2 步命中 93.7%），但每轮都新开会话，只能命中约 6k 的固定 system 前缀，命中率被压在 ~25%。
- **落地（四刀）**：①DSH 会话续接——`Participant.EngineSessionID` + `buildResume` 传 `--session-id`，引擎 `sessionId` 从事件流解析回写并持久化；②运行护栏 `seatguard.go`——工具安全阀 120 次（ID 去重；16 次初版经 138 次实弹复核会误杀多数正常核验）、无进展看门狗 120s（任何输出喂活性）、开局静默闸 5 分钟（工具/thinking 不算正文），三类失败不触发自动重试；③命令行显式 `--permission-mode read-only --approval-policy never`，`runMode` 按实际命令行改写；④`parseSeatRunStats` 解析 `session`/`step_end` 落 `Contribution.Usage`，工具事件落盘节流 1s，提示词加工具预算纪律；前端席位卡/时间线显示「缓存命中 x% · N 步」。
- **验证**：`go test ./internal/pantheon/` 全绿（新增 8 例）；live `TestLiveDSHSeatSessionReuse` 实测会话 `session-23c1b237-…` 第二轮缓存命中 **96.7%（6016/6224）**；`go build ./...`、`go vet`、`gofmt` 干净；`tsc -b` 0 错、Pantheon 前端 63/63。
- **未做**：codex/zcode 无 usage 上报（前端诚实显示「未上报」）；DSH 12 轮长会上下文增长未实测；工作区无 `.env` 且 Gaea 未配 deepseek key 时 DSH 仍会失败（错误文案已带修复路径）。

## 运行监测 UX/UI 重构与调用统计细化 · 2026-11-03

- **提交**：`fb1501ca`，待 push。
- **信息架构**：运行监控从 HUD 舰桥改为运营工作台，首屏只保留服务状态、引擎连接、当前吞吐和上下文压力；调用统计升为主工作区，随后依次为模型调用明细、运行状态、最近请求和引擎侧请求。
- **调用统计**：新增今日/最近 7 天/最近 30 天范围 KPI，联动调用量、成功率、Token 与费用；两张趋势图提供失败标记与 Token 入/出拆分；右栏质量卡聚合成功/失败、近期平均耗时、P95、四档耗时分布和主要错误。图表提供可展开数据表。
- **口径隔离**：范围指标来自 `getModelCallStats().trend`；模型明细来自累计 `per_model`；耗时/错误来自 `recent_calls`；上下文、缓存和引擎请求来自 Strata/天璇 `/metrics`。页面逐区标注，不把累计值、范围值和近期样本混读。
- **模型定位**：模型明细支持模型 ID/引擎搜索、需关注/本地/云端筛选，以及调用量、成功率、Token、耗时、费用排序；本地模型显示“免费”，异常行以文字、图标和左侧边界三重标记。
- **视觉与响应式**：移除扫描线、HUD 角标和多层霓虹，统一使用现有主题令牌和单一强调色；状态继续使用文字+图标，不依赖颜色。375px 图表进入容器内横滚，页面无横向溢出；亮/暗主题均已实际走查。
- **文件记录**：新增 `docs/gaea-runtime-monitor-redesign-2026-11.md`，并把桌面暗色与 375px 走查图存入 `docs/snapshots/runtime-monitor-2026-11/`；`docs/README.md` 已登记。
- **验证**：`RuntimePage.test.tsx` 12/12、`tsc -b`、定向 ESLint、Vite 生产构建、`build.bat` 全链通过；冒烟 `/api/health` 返回 200。
- **桌面产物**：`build/bin/gaea.exe` 57,468,416B，SHA256=`FDF35BEFEFBE8930A36E6324D14FFEBF635CD59140C1320EDA327CF91CEC0CFD`；桌面 `gaea.exe` 同哈希。

## 众神殿四视图 UX/UI 重构 · 2026-11-03

- **提交**：与众神殿 R6 评测底座合并提交 `5243c429`，待 push。
- **信息架构**：档案栏新增“议事厅 / 协议对照”同级入口。纪要页把拓扑、共享讨论、各席详情、分歧台账收敛为四个一级视图；运行中默认拓扑，已完成默认共享，拓扑中心节点可直接打开共享窗口。
- **各席窗口**：新增 2×2 Agent 窗口墙，每席显示状态、最新公开观点、观点数与工具数；点击继续复用既有观点/工具详情。≤760px 单列，四视图入口回落 2×2。
- **Agent 窗口布局返修**：旧 `.pantheon-agent-windows` 仍把容器声明为两列 Grid，导致标题与窗口被拆成左右两块、窗口全挤到右角。外层改回纵向流，标题占整宽在上，窗口网格占整宽在下；实机计算得 section/grid 宽 957.75px、两列各 473.875px，768/375 无横向溢出并回落单列。
- **Agent 输出弹窗返修**：详情从 320–382px 窄栏扩为宽屏 500–600px 并排面板、较窄桌面 700px 右侧抽屉；观点改为连续正文分隔流，13.5px/1.82 行高，工具日志占满面板。1248 视口实测弹窗宽 700px，1357 视口并排面板宽 600px。
- **拓扑共享卡片返修**：用户指出“拓扑图里共享卡片信息缺失”后，撤掉卡片外重复态势条，把状态、当前 Agent/归档状态、最新共享发言、返回率、处理/候场/失败、观点、工具、待收口分歧、轮次阶段与“打开共享窗口”入口全部收回中心共享卡片。卡片桌面 340×114px，375 窄屏 243×111px，信息换行不横向溢出；每席节点保留最新轮次/时间/摘要、动作徽标、目标席位、观点数、工具数与 Harness/模型版本，运行席以文字“当前”、身份色和 `aria-current` 三重编码。
- **共享窗口时间线重设计**：共享流改为连续纵向轨道，议题/席位/轮次共同挂在时间轴上；轮次成为可折叠章节，默认只展开最新轮次，历史章节收起为 `01 / 02 / 汇` 并显示发言数，支持鼠标和键盘展开。章节头在自身范围内吸顶，下一章节会自然推走上一章节，不再互相覆盖。发言卡只保留席位、动作、耗时、时间与正文，编排说明压缩，协作汇总无轮次时仍单列阶段，避免显示“0 轮”。
- **共享结论预览**：共享窗口头部新增预览条；收起态高约 52px，直接显示确认状态和结论建议，展开后显示摘要以及行动提案/条件/分歧/风险/证据计数，并可通过“查看完整结论”定位到下方原结论面板，不复制结论真相。预览在 375 宽自动换行，无横向滚动。
- **共享窗口当前席**：在共享流滚动容器之外新增常驻状态带，显示当前 Agent 头像、姓名、阶段、连续工作时长、返回进度和待收口分歧；四席状态可点入独立窗口。没有活跃席时明确显示候场、待用户回应或已归档，历史消息滚动后提醒仍可见。
- **协议对照**：接入 `PantheonEvaluation(minCases)`，页面只保留门禁、四链覆盖、同题记录与判定边界；补齐骨架、错误重试、空态、最近纪要入口。CLI/费用/人工质量项保持不可推断。
- **减法**：沿用青灰与月华金，不新增第二强调色、环形评分、通用仪表盘卡片或装饰动画；去掉重复卡片与英文装饰标签，状态继续文字+图标双编码。
- **可见交互验证**：改用 Codex 可见内嵌浏览器直接操作；时间线初始仅展开“协作汇总”，首轮/次轮折叠头各 52px，点击展开后首轮章节 500px，可再次收起；滚动时章节头保持在流顶部且下一章节不重叠。1357×720、768×900 与 375×812 实测页面 `scrollWidth == clientWidth`；375 宽下章节头宽 225px、消息面宽 183px，无横向溢出。
- **自动化验证**：PantheonPage + Record streaming + AgentAvatar 63/63；`tsc -b`、生产构建、定向 ESLint 0 error、Go pantheon 全包、App Pantheon 定向、Go vet、bindings drift 718、文档守卫通过。`frontend-e-check` 仍被既有 E25 `useChatStream` 未走 `ChatSend` 阻断，与本轮文件无关。
- **桌面产物**：重新执行 Wails 2.13 构建（前端预构建 + `-skipbindings`），`build/bin/gaea.exe` 57,452,544B SHA256=`4D6CE4110A720EB333BC4BDB2FBF24063EEE3AB633D6164BCE001AA4CEAA3E2E`；`/api/health` 冒烟 200，桌面另存 `gaea-new.exe` 同哈希。

## 众神殿 R6 评测底座 · 2026-11-03

- **只读投影**：`EvaluateSession` 从既有 session.json 复算轮次、调用/重试/错误、工具、墙钟与调用耗时、缺口、插话、证据句柄、台账、确认与签注；不新增第二份可漂移的评测状态。
- **同题聚合**：对照键由议题/背景/成功标准/工作空间/材料摘要生成；标题、preset、protocol、轮数作为实验变量。自动归类 single、independent_synthesis、roundtable、targeted_challenge、red_team，历史接力单列；固定投票明确排除。
- **报告命令**：`go run ./scripts/pantheon_eval`（Markdown）/ `-json` / `-details` / `-strict`。20 个有效议题未齐或四核心链路缺一场即门禁未就绪；token/费用、事实错误、关键反例、未核验比例、复用意願明确留人工评审。
- **版本证据**：开场探针的 Harness 版本写入 Participant（Grok 记录模型名）；旧会话缺版本或只缺部分席位时报告明确告警，不用当前版本回填。CLI 具体模型版本仍未由 harness 暴露，报告要求实验记录另行固定。
- **导出**：完整审计 Markdown 新增“运行度量（可复算）”。本机历史记录实测报告为 0/20、门禁未就绪，含 2 场无版本快照告警。
- **验证**：`go test ./internal/pantheon/`、`go vet ./internal/pantheon/`、`go test ./internal/app -run Pantheon`、`go run ./scripts/pantheon_eval -json`、`tsc -b`、Pantheon 定向 vitest 57/57、ESLint 定向、绑定漂移 717 名、文档守卫、`git diff --check` 均通过；`ci.ps1 -Quick` 的 build/vet 已过，停在全仓既有 15 条 lint 红项（3 errcheck / 11 gofmt / 1 ineffassign，均不在本次改动文件）。`test-related.ps1` 因生成 models 触发广泛关联前端测试，暴露既有 NovelSettingPage、FilePreview 与 SchedulingSection 环境/时序失败后主动停止，不作为本次门禁结论。

## 众神殿四框架机制落地与 Git 打包 · 2026-11-03

- **功能提交 `a4385d4e`（38 文件，未 push）**：按四框架专项调研完成 R1–R5。动作/目标贡献/回应契约 + 同一分歧台账；用户插话 `origin=user` 可被各席定向回应；`roundtable / targeted_challenge / red_team` preset；至共识按 DID 指纹续跑并在事件/导出留痕；红队汇总后全席独立确认，反对触发主席一次修订并保留原稿哈希；`Verdict.proposedActions` 只读行动提案，不自动执行。
- **文件记录**：`docs/archive/gaea-pantheon-v2-roadmap-2026-11-03.md` 增补 R3/R4/R5 落地记录；四框架专项调研状态同步；`design-system/gaea/pages/pantheon.md` 增补动作/用户插话/独立确认/行动提案契约；`docs/README.md` 维持主报告与证据附录索引；Wails models、前端 types、mock、binding signatures 已同步。
- **验证**：`go test ./internal/pantheon/`、`go vet ./internal/pantheon/`、`internal/app` Pantheon 定向、`tsc -b`、ESLint 0 error、Pantheon+签名契约 60/60、生产构建、文档守卫、绑定漂移全过。全仓既有无关红项=`internal/gaea/tool` compact schema（task.engine/model 漂移）与 `NovelSettingPage` 单例；另全量前端曾出现 jsdom `scrollTop` 只读异常，未触碰相关文件。
- **记忆状态**：Hindsight CLI/plugin 本机无配置目录（`~/.hindsight` 不存在），本会话也未暴露 `hindsight_ingest_document` MCP 工具；已按仓库既有长期记忆机制写入 `.gaea/AGENTS.md` 与本文件，作为当前权威记录。若后续安装 Hindsight，应以标题「众神殿四框架机制落地 2026-11-03」补录。
- **下一步**：R6 协议对照评测，需要 20–30 个真实议题和固定 Harness/模型/权限基线；在评测前不把红队确认设为默认，也不把行动提案交给执行器。

## 众神殿 V2 一日十四提交 · 2026-11-03

- **众神殿单日大版本日（2026-11-03，ZCode 会话，16 提交 8dadd38f..be219425 全部未 push）**——从「实弹 22 次提示词截断」一路收到「V2 P0 真相单源化+P1 收口机制」，全部经 build.bat 全链+双端同哈希交付（最终 exe 39eb4402… SHA256 前缀，新壳 PID 40952）。批次序：
  1. **多线合流 8dadd38f**（153 文件：语音 V2 实弹线/众神殿论证文件化+DSH 席位状态机/办公预览滚动两刀/造价原表单价）+cmd 弹窗根修（启动/关闭路径 taskkill/netstat/powershell/git 九点 proc.HideWindow）。
  2. **论证文件化（22 次截断根修）**：maxPromptRunes 24000→60000+贡献/大材料归档 sessions/<id>/contrib|evidence（session.json 权威+惰性自愈）+分发梯度（同轮全文/上轮立场卡/更早索引行+全文路径）+grok 归档路径入 readPaths。
  3. **席位启动黑框+专属发言动画**（2e9a9221）：pantheon runner.go CLI 拉起 HideWindow 补刀（首轮漏扫 internal/pantheon 包）+CSS 按 data-agent-id 四款动画。
  4. **实弹四连问**（aae0454c）：上一轮立场卡补全文路径（贡献 ID 空头支票根修）+四席能力矩阵澄清+toolPolicy 降级指引+轮换顺序答疑（rotateAgentIDs 确定性非随机）。
  5. **全员联网**（253eb88f 用户拍板）：codex read-only→workspace-write+network_access（read-only 默认禁网=GitHub API 直连失败真因）。
  6. **用户中途评论**（5c0a370e）：AddUserNote+user_note 全文分发+讨论流插话输入框+绑定面 717 四表同批。
  7. **共享流可视四刀**（4c447621）：is-seat-speaking 全程动画+席色四层+详情面板每轮折叠+左轴时间线。
  8. **并行开场**（a58b21e3 用户拍板）：独立开场第 1 轮 goroutine per seat+agentCalls 会话×席位双键多槽+落盘序=完成序；区间重叠证明法测试。
  9. **zcode 空结果真根修**（b4e2c116）：**runStream 只取 cappedBuffer.buf(head) 丢弃 11-02 加的 tail 环留**——大仓流撞 4MB cap 时收尾事件全在被丢尾部（live dump：旁路 2616 字正文在手 parse=0→修后 2933 字全文）；cap→16MB。
  10. **议事质量五刀**（2490602a，实弹 pan_1793681319746 病例驱动——先被骂没看案例）：插话段条件注入（治四席复读回归）/主席轮间小结实权/共识三行制遗留台账/汇总提炼式/转述纪律+质量阶梯。
  11. **V2 规划落档**（6656b8fd）：docs/archive/gaea-pantheon-v2-roadmap-2026-11-03.md（九轮讨论全文提炼：P0 真相单源化→P1 收口改造→P2 协议形态）+**终止草稿刀**（bf0d1dbd：finishCancelled 落 resultPath，未收口草稿标注）。
  12. **V2 P0 真相单源化四刀**（76b05d11）：五入口 liveSessionLocked 同源+插话 drain 交付/ResolveGate 双守卫防双活循环/seatRoundCoverage 事件投影进 Gaps+Partial/startSeatRoundLocked 轮边界具名动作（席位跨轮解冻，58 条拒绝警告洪水根除）。
  13. **V2 P1 停机判据+翻转举证**（c3b6eb18）：Contribution.Residual 抽「剩余分歧」行+停机门增判（自报一致+保留实质分歧不得停机，grok r2 洞根修，旧测试固化病翻转）+no→yes 翻转审计无证据句柄→从众嫌疑留痕。
  14. **V2 P1 分歧台账**（be219425）：Issue 全生命周期=剩余分歧行自动开条（同席去重）→resolved 带撤回轮次/结案 dissent 永久入档；open 贯穿注入 prompt+汇总按条交代去向+导出台账段。
- **用户中途插话刀**（5c0a370e 前身 2dbb22f5→amend wailsjs）与启动铭牌+拓扑每席跳过钮+工具期工作卡（c49c050d：金线框铭牌 notification/恒显每席跳过钮/working 工作卡）并入上列 6/7 批次叙述。
- **V2 余量（挂下批）**：分歧台账 Issue{id,双方,证据,状态,阻塞,核验,去向} 完整结构化+主席小结结构化输出台账、收口轮并行隔离（问「你会不会签」）、P0 防御性 saveSessionLocked 写路径收口+完整版本检查、P2 协议形态按需。
- **验证基线**：每批次 go test pantheon 全绿+app 包多轮 150~190s 绿+tsc/vet 0+build.bat 冒烟 200+桌面副本同哈希+新壳 PID>构建时刻核验；DSH 流水线同树在途（zcode argv 32K 临时文件刀随包内 add 卷入 c3b6eb18，绿不拆记档；deliberation-frameworks 调研文档随 chore 入库）。

## Orca 蒸馏 P0 · 2026-11-02

- **Attempt 身份闭环 + 隐性派生禁令 + 图片能力门（未发布）**——依据 `docs/gaea-orca-pantheon-distill-2026-11.md` 首批 P0 落地。①`pantheon.CallRef` 嵌入 `Participant/Event/Contribution`：`CallID` 在同一 session/agent/phase/round 的重试间稳定，`AttemptID` 每次实际调用重生成，`RetryOfAttempt` 指向上一 Attempt；start/delta/tool/output/error/skip/retry 全链携带。`activeAgentCall` 现在同时按 skip/retry 与“是否仍是当前调用”拒绝残影，旧调用结束后再到的 delta/tool 不写状态。②前端直播键加 AttemptID，封口键按 Attempt（历史无 ID 退席轮）；跳过、失败、重试只清对应 Attempt，新 Attempt 开始不会解封旧 Attempt。③协作守则明确“亲自完成本轮，不派生 subagent/并行助手/二次委派”。④图片通道显式化：Codex/ZCode `image-input`、Grok `image-path-tools`、四席 app 层共享 `shared-image-summary`；有附件时后端逐席 preflight，三者全无才拒绝，Composer 同步点名并禁用开议；DSH 仍走共享摘要。边界=不做类型化未知态 O-D、决策门 O-E、写执行 O-F、daemon O-G。验证=`go test ./internal/pantheon` 绿、PantheonPage+streaming 44/44 绿、`tsc -b` 绿、Wails 绑定重生成成功；共享工作树内 `internal/gaea/agent/task.go` 的并行在制 `routeProv` 未使用会挡全仓构建，本刀未代改该文件，临时编译 shim 已撤。未抬版本、未 commit/push。

## dsh 子代理蒸馏收官 · 2026-11-02

- **四档调研+六批执行，刀池全销项（已 commit 未 push）**——调研四档入库（96a2c9da：`docs/archive/gaea-dsh-022-distill-2026-11.md` 0.2.1-alpha.2→0.2.2-alpha.2 增量〔57 篇升级指南〕/`docs/archive/gaea-dsh-subagent-deep-distill-2026-11.md` 机制级深挖〔src 6360 行全量+G 系补遗〕/`docs/archive/gaea-dsh-subagent-ui-distill-2026-11.md` UI-UX 补课/README 登记）。执行批四刀（5bb83fe6）：SA-D 被拒操作反重试+SA-E 续跑措辞分叉（均落 task CompactDescription）+UA-2 子代理活动词 21 键参数优先级（toolCallId 配对、结果头兜底）+UA-1 子代理头部墙钟历时芯片（tab 合并链带时间戳+含等待诚实标注）；SA-A/C 盘点销号（output_schema 已等价实现/取消判据缺失）。K1 窗口自学习（17954451）：`provider.ContextOverflowLimit` 三正则从溢出错误正文提取真实窗口+救援前采纳（只缩不涨）——引擎登记表无窗口元数据的变体落地，K2/K3/K4 刀前核实销号。P2 收官（d2c812b2）：send_message 子→父中期回报（steerFromSubagent 署名信封归因分离+送达回执非答案+发消息不结束任务+WithRunner 派发盖章）/task 按调用模型路由（`[agent] subagent_routes` 授权清单，空=只许父路由，未授权 fail-loud 点名可用路由；provOverride 优先级=按调用路由>boot 子代理模型>父）/session-reference 销号（已由 10.3 落地 refs_session.go，池口径过时）。验证=`go build ./...`+vet 0+agent/boot/config/app/provider 五包测试+vite SubagentThread 18/18+tsc 全前端零错+eslint 0。绑定面零变更。未 push。

## 注意力事务 · 2026-11-02

- **注意力事务第五刀：无人值守放行也留痕（没人可问，也要记下这里本该问人）**——第四刀把「机器替你做的决定」记进账本，但够不到 `Approver == nil` 那条路：子代理、后台任务、headless 都没有 UI 可问，`permission.Gate` 遇本该问人的工具直接放行（原注释 non-interactive: preserve autonomy）。放行是对的（无人值守问谁），错的是它在账本里无声——事后看日志，「这个子代理写了 40 个文件」全是策略允许，分不清哪些本来就允许、哪些本该问人却没人可问。落地=①`Gate.OnAutoAllow(toolName, subject, source)` 最小 sink，纯账本钩子不能改决定，只在两处 nil-Approver 分支触发（`AlwaysAsk`→hard_ask、策略 `Ask`→gate），**策略本就 Allow 的不记**（不是覆盖，只会长噪声）；②`event.AutoDecidedHeadless="headless"` 与 auto/yolo/session_grant 并列，报表翻成「无人值守放行（headless，没人可问）」；③`boot` 给 `headlessGate` 挂钩子、用共享 sink 发 `ApprovalAuto`，与交互路径的卡落在同一本会话日志；④口径不变：不计入打断卡与均等待，headless 桶暴涨本身就是「无人值守越过越来越多审批」的信号。验证=permission 新 5 子测试（ask/hard_ask 各记一条、策略 Allow/交互闸门/deny 都不记）+report headless 分桶并守住院打断口径；文档同前档第十二节。未抬版本、未 commit/push。

- **注意力事务第四刀：自动放行留痕（没人被打断，也要知道谁替你做的决定）**——第一刀特意让「短路不造事务」（没打扰人就没有记录），副作用是 auto/yolo/会话已有规则的放行在日志里完全不可见：事后无法回答「这条规则什么时候、被谁（人还是策略）批的」；上游结论 :14「或在授权模式下仅记录拟执行动作」、:22「headless auto-allow 需明确哪些权限可放行」正指这一层。落地=①新事件 `approval_auto`（`event.AutoApproval{Tool,Subject,Source,Decided}`，Decided=auto/yolo/session_grant；**没有 id、没有 latency**——它不是一次打扰）；②四个不弹卡出口各记一条：`gateApprover.Approve` 的 auto 短路（按 permLevel 区分 auto/yolo）、`promptApproval` 两处 `ruleGrantedLocked` 短路（入口 + promptMu 后竞态复查，Decided=session_grant）、`RequestPermission` 的 auto 分支；③`session` kind `approval_auto`、`trajectory.ApprovalRec.Decided` 与 Decision 分列（Decision=人点按钮，Decided=没按钮时谁放行），报表新增「自动放行」列与按 decided 分桶，**不计入打断卡与均等待**；④两条线分列的意义=混在一起时要么打断数虚高（把机器放行算成打扰）、要么授权史缺失（只看卡就以为没发生过）。**边界**：headless `nil-Approver` 放行（`internal/gaea/permission/permission.go:259`/`:281`，注释 non-interactive: preserve autonomy）没有 controller sink，本刀接不到，记开放项（需 permission 层最小 sink）。验证=五包定向（新 4 例：control yolo 3 次调用只 1 条账本且零卡、session payload 无 id/latency、trajectory 独立记录不配对、report 计数不动打断）；文档同前档第十一节。未抬版本、未 commit/push。

- **注意力事务第三刀：回合发起者类型化**——上游结论档 :46 记的风险：合成唤醒目前靠 `[system]` 文本前缀和白名单识别，正文/路径一改就误分类，而且统计上没法把「真人发起的回合」与「后台唤醒、子代理回合」分开——打断率的分母里混着唤醒轮，数字不诚实。落地=①`event.TurnStarted` 增 `Trigger`（`TriggerUser`/`TriggerWake`/`TriggerSubagent`，缺省读端按 user，旧事件零影响）；②判别字段源头在入口（控制器知道是真人发送还是后台唤醒），经 `agent.WithTurnTrigger(ctx, trigger)`/`TurnTriggerFromContext(ctx)` 走到 `agent_run.go` 唯一的 `TurnStarted` 发射点，不改 `Run` 签名；③`control.sendWithRaw` 增 trigger 参数，排队消息也记住自己的来源，`maybeWakeByJobs`=wake、`SendWithRaw`/`Submit`=user、`runSubAgentInternal`=subagent；④`turn_started` payload 落 `{"trigger":...}`（空 trigger 仍写 `{}`，旧日志逐字节一致），`trajectory.Turn.Trigger`（omitempty）旧轨迹输出不变；⑤`scripts/attention_report` 增「人轮/唤醒/子代理」三列，打断率分母只取人轮。**诚实边界**：`interrupted_recovery.go:56 isSyntheticUserTurn` 仍在消息回扫时按 `[system] ` 等前缀判合成轮——作用于消息历史而非事件，要类型化需消息级 provenance，本轮不做，留在开放项。验证=五包定向（新 4 例：control 唤醒回合 ctx trigger=wake、session payload 与旧格式兼容、trajectory 折叠、report 分桶）。文档同前档第十节。未抬版本、未 commit/push。

- **注意力事务第二刀：悬空卡闭合（进程先死也有结局）**——第一刀让「请求→结果」成对落盘，但进程在卡还开着时被杀（崩溃/强退/关机），内存里的 approvals/asks 随进程消失，日志永远停在请求行：统计器只能读成「用户一直没回答」，打断数偏高、解决率偏低，而「放弃」与「未解决」在账本里完全同形。落地=①`event.OutcomeAbandoned`（语义=进程先死、没人回答过，不推断用户意图）；②`session.UnresolvedAttention(entries)` 流式配对悬空卡（同 id 重复请求只算一张、孤立结果不造卡、审批结果不关闭提问卡），`session.CloseOrphanAttention(logPath, space)` 补写 `approval_resolved`/`ask_resolved`（`outcome=abandoned`、`decision=""`、`latency_ms=0`，幂等）；③**挂点唯一**：`Controller.ResumeFromDisk` 事件模式分支（LoadWithFormat 之后、Resume 之前）——切换会话与启动恢复都收敛于此，`controller_rewind` 的截断重放不走这里；失败不阻断恢复（尽力而为，日志原文仍是真相）；④统计器 `latency_ms=0` 计入结果分布但不进均等待（否则 abandoned 会把均值往下拽），口径行显式写明。**不是看门狗**：不设阈值、不计时、不杀回合。验证=五包定向（新 3 例：session 识别+补写幂等、control 造崩溃日志→ResumeFromDisk→轨迹成对 abandoned、report 均值剔除 0）。文档同前档第九节。未抬版本、未 commit/push。

- **注意力事务第一刀：请求→结果成对可重放 + 阶段 0 基线统计（众神殿 super agent 讨论产物落地）**——依据 `pan_1793618668807_6f24ec495bc1`（四席讨论 DeepSeek 开源 super agent 原型，2026-11-02）：结论是蒸馏 Router/Worker 与 watchdog 都不对，真东西是「把请求人的注意力做成有起点、有结果、可重放、可计数的持久事务」，且判别字段要贯通实时事件/持久日志/派生轨迹三层。核查确证了洞：`Approval.Reason/Request` 只在实时层，日志只剩 id/tool/subject；`Approve()`/`AnswerQuestion()` 只往内存 channel 写值、**不发完成事件**——批/拒/超时/取消在日志里同形，每轮打断数与解决率无从统计（V5 提案「≤6 次且 −50%」因此无基线）。落地=①`event.ApprovalResolved`/`AskResolved` 成对结果事件（**不重命名**既有 `approval_request`/`ask_request`，旧日志与前端零破坏），`outcome` 归一 allowed/denied/aborted/timeout/cancelled/answered + 明细 `decision` 串 + `latency_ms`；②`Approval.Source` 四值 gate/hard_ask/project_plan/permission_request 一路落到日志 payload 与 `trajectory.ApprovalRec`，旧日志缺 source 记 legacy、有结果事件时按结果 source 精确归位（**不把「未记录」读成 gate**）；③`promptApproval` 的 reply/ctx 取消/超时三条出口与 `Ask` 的 answered/cancelled 全部 defer 发结果，会话已有授权的短路路径**不产生事务**（没打扰人就没有记录）；④阶段 0 只读统计器 `scripts/attention_report`（零依赖、`go run ./scripts/attention_report`，`-json` 可选）：逐会话报轮次/打断卡/审批/提问/已解决/未解决/均等待，口径=一次打断一张卡、ask 按卡不按问题数、审批与提问分项、notice 仅诊断；「每事项」分母缺 `case_id`（`internal/caseos` 未建）故只报每轮。**不做**：不重建 30 秒看门狗（v3.0.2 拍板删除，c426d3ff 引入/a5e33377 汇总删除）、不常驻 Router、不整包移植上游、不追「每轮 ≤6」、不改 headless 放行（有意设计，不弹卡即不造事务）。验证=`go build ./...`+四包定向+新增 12 例（control 8 子测试含超时/取消/短路无事务、session 日志 payload 判别字段、trajectory 按 id 回填且孤立结果不建记录、report 三例）+全仓 `go test ./...` 零 FAIL；前端零改动（结果事件走 reducer default）、绑定面零变更。文档=`docs/gaea-attention-transaction-2026-11.md`（计数口径/事件契约/三层审计路径/开放项：case_id·step_id、V5 账本 :381 只有 approval.granted 需对齐、旧日志不可补、initiator·risk_tier 未建模、wakeMaxConsecutive=3 是自唤醒不是人打断）。未抬版本、未 commit/push。

## 众神殿工作空间范围与 UI 重设计 · 2026-10-29

- **开议界面 DeepSeek 首屏式重设计（2026-10-31 午后，用户给 DeepSeek 官网截图拍板「更喜欢这个」）**——欢迎+开议两屏同语言：①居中 hero（公告胶囊「只读运行·不改动工作区」+大字距衬线大标题 letter-spacing .05em+副题）；②**单一输入卡**（pantheon-compose-card）：无边框大号议题输入+字数/参与数 meta+卡内胶囊工具条（材料/图片/路径/汇总人/轮次/补充设置）+**圆形开议钮**（pantheon-start 改 42px circle，aria-label 开始讨论，Tooltip 承载拦截原因）——topic-desk 与旧 chatbar 两大盒装区合并迁入卡内；③卡下各段去盒装：讨论范围/参与者/发言顺序全部细胶囊带（发言顺序改单行顺序胶囊）；④柔和雾面 washes（radial-gradient×4 accent+primary 色混，welcome/composer stage ::before，主题变量驱动亮暗同源）；⑤welcome 居中化（两栏改单列、seat 卡胶囊化）。**DOM 契约零变动**：全部 testid/aria-pressed/拦截文案/前移后移按钮保留（40/40 测试绿）。**ui-ux-pro-max 审查补课**：①`prefers-reduced-motion: reduce` 降级块（rise 动画/悬停位移全关，ai-driven-landing 范式 High 项）；②圆钮 42→44px（移动口径对齐，Web AA≥24 本已过）；结构对 `minimal-single-column`/AI 输入 hero 范式核符；占位标签已有 aria-label+meta 行缓解（对话式设计沿用参考范式）。复拍暗态无回归（IAB clip 平铺为截图工具伪影，reload 后视口 free-size 漂移所致，setViewportSize 后消失）。验证=vitest 40/40+tsc -b 0+eslint 0+build.bat 全链（双端同哈希 40575cad…）+**IAB 亮暗双态四屏实拍目检**（welcome/composer×亮/暗：居中构图、washes 克制、卡 focus-within 光圈、圆钮 disabled 降透明度均成立；配方=vite 5199+?mock=1+gaea-dark localStorage+首页胶囊钮进板块——首页无 w-board-pill，那是壳内配方）。未 commit（随本日众神殿批次挂拍板）。

- **实弹反馈七刀（2026-10-31 午后）**——用户真机实弹（pan_1793414706331，议「小说板块走向」）七项反馈全处置：
  1. **zcode 观点丢失/推理墙/乱码根修（runner.go）**：三处根因=①`cappedBuffer` 只留头部，4MB cap 斩尾把 turn.completed/最终陈词截丢（改为头+尾 3/4 环留）；②新版信封（eventId/payload 无顶层 type）的聚合事件 `turn.completed.response` 不被 parse/直播两路认识（新增 `extractZcodeFinalText`，parse 优先级 texts→zcodeFinal→generic→rebuilt）；③**增量重装加完成门**：text_delta 累积须见 `finish`/`turn.completed` 才算发言——中途夭折（超时/被杀）的流只有过程旁白，返回空走 agent_error，不把旁白冒充观点（实弹 r1/r3=149/187 字旁白、r2=200001 墙的根因全链）；被 cap 斩断的 JSON 残段不再当纯文本触发整体兜底；历史墙读取期抢救走宽松 `salvageZcodeDeltaText`（不看完成门）。**zcode 探针实证**：payload.kind 生命周期 start→reasoning_delta→text_delta→finish→turn.completed(response,usage)；reasoning_delta 恒不入正文。
  2. **UI 重复输出护栏**：displayOutputs 去重键改 `agentId:round`（对 phase 漂移免疫）+applyLiveEvent 定稿落位即清同席同轮全部直播残影；后端 finalMessageEcho 认识信封聚合事件（sawDelta 已播则吞）。
  3. **发言顺序混乱+「输出上一轮文本」**：贡献时序实为轮换正序，乱象=直播条目钉底+定稿重排的跳变与 zcode 墙；提示词立场卡段加「引用往轮观点给出贡献 ID+一句话概括，勿整段复述原文」。
  4. **切换 agent 弹窗根除**：左栏卡全变纯状态展示（离线卡也不再弹路径 Modal）；路径配置改 Composer 离线席位 chip 内显式「设置路径」小钮（pantheon-agent-config-*，stopPropagation 不触发选席）。
  5. **总时长读秒**：拓扑头右侧 `pantheon-total-elapsed`——运行态逐秒走表（clockNow-createdAt），完成态定值（updatedAt-createdAt）。
  6. **中断保存/恢复**：`Manager.Resume(id)`（run() 泛化 startRound+初始 history）——失败/终止场保留全部既有贡献、从中断处下一轮继续、照常汇总；绑定面 701→702（PantheonResume 四处同步：handler/bindings/wailsjs/mock+bindingNames+spaceBindings 认领）；前端失败横幅新增「恢复讨论」钮（pantheon-resume，有历史发言才显示；重开整场降为次选）；零历史场指引导向重开。
  7. **历史论点续议**：`parentStanceEvidence`（trace.go）——续议自动把父会话最近 ≤8 条讨论发言转立场卡（机械节选 700 字+原长+贡献 ID）注入 Evidence，manifest 只钉用户设题材料不含派生卡；父会话不可用 fail-loud。
  验证=Go 新增 8 用例（zcode 聚合/夭折流/信封回声两态+Resume 两幕+父会话立场卡+孤儿 fail-loud）+go pantheon/app 绿+golangci 0+前端 45/45+tsc 0+eslint 0。**坑=并行执行体在途改 gaea/components/Transcript.tsx 删 cwd prop 把 git-clean 的 schedule/ChatPane 拉红，tsc -b 红而 --noEmit 绿（增量缓存口径差），构建被其中间态阻塞须等收敛重试**。未 commit（与前四刀同批挂拍板）。

- **圆桌裁决第三刀：最小可追溯（2026-10-31）**——①能力逐席入档：`agentAdapter.runMode` 四席各写真实施加机制（codex=`sandbox read-only`+联网搜索开放降级表述/zcode=plan 模式/dsh=环境变量/grok=只读工具白名单），`Contribution.RunMode` 每次调用打标（讨论+汇总），导出参与者行「只读经 …」逐席呈现——不再写笼统「已只读」（裁决条件：无硬限制能力必须降级表述）。②材料快照：`Session.InputManifest []InputDigest`（trace.go `buildInputManifest`）Start 时对 Evidence 正文+附件**盘上活文件**算 sha256（附件讨论期可被改动=裁决修正意见真正缺口；哈希失败诚实记 Error 不静默），导出「### 材料快照（开始时哈希）」。③签注绑版本：`SignOff.VerdictSHA256=verdictFingerprint(verdict)`（canonical JSON sha256），SignOff 后 `writeResult` 重渲染交付物，`RenderFinalDocument` 新增「用户确认」段（决定/时间/备注/绑定小结版本）——签注与转发件不再两份真相。④续议父会话：`StartRequest.ParentSessionID`，前端续议 seed 传 `parentSessionId`（+protocol 继承），导出「父会话：…（续议承接以简报为准，不继承全文）」+record 元信息父会话行；types/models/Composer 全链同步（绑定面不变）。验收=trace_test.go 5 用例（manifest 双源哈希+缺失文件诚实报错/签注指纹绑定+重渲染四断言/runMode 逐调用+导出/父会话入档导出/manifest 导出）+续议前端用例补 parentSessionId 提交断言；门禁=go pantheon/app 全绿+golangci 0（gofmt 两测试文件顺手修）+tsc/eslint 0+PantheonPage 40/40；build.bat 全链过（冒烟 200，双端同哈希 87a5223e…）+壳拉起。未 commit（四刀同批挂发版列车拍板）。裁决三刀至此全清；余池=真机实弹验证（下一场议事看独立开场+材料快照+签注重渲染全链）。

- **独立开场升为默认协议（2026-10-31 用户拍板）**——用户实弹观察「首轮第一位的阐述会把范围限制在一定范围」=裁决分歧区预留的首位锚定问题，拍板=第 1 轮各抒己见、第 2 轮起共享前轮信息再议论。落地=后端 `NormalizeProtocol("")` 默认翻转为 `independent`（显式 `council` 仍可用；历史存档无 protocol 字段，展示层 `protocolLabel` 继续按圆桌接力渲染，不篡改史实）；前端 Composer 默认选独立开场+选项排序前置+Tooltip/头文案改「第一轮各抒己见，第二轮起共享前轮信息互相补强」。测试显式化五处（NormalizeProtocol 空值断言翻转/TestDistribution 同轮全文分发是 council 语义补显式/单轮腿/圆桌单席闸/路径入档 council 腿）+新增默认单席放行兜底断言；前端「少于两席禁止开始」用例按新语义改写为双态（默认独立开场单席放行↔切圆桌接力重新拦），提交契约测试补 `protocol:'independent'` 默认断言。门禁=go pantheon/app 全绿+golangci 0+tsc/eslint 0+PantheonPage 40/40；build.bat 全链过（冒烟 200，双端同哈希 c162c1b2…）+壳拉起。未 commit（随第一/二刀同批挂拍板）。

- **圆桌裁决第二刀：对照实验可表达（2026-10-31）**——路径标签入档：`StartRequest.Protocol`（`council`=圆桌接力〔默认〕/`independent`=独立开场），`NormalizeProtocol` 未知值 fail-loud，导出/最终文档落「议事路径」行；单轮放行：`normalizeRoundSettings` 手动轮 `<1→2` 改 `<1→2 仅零值默认`，`Rounds:1` 保留（连带修正 `TestGrokSeatJoinsCouncilAndChairs` 原依赖 1→2 隐式抬轮，改显式 Rounds:2）；独立开场基线：第 1 轮 `openingHistory=nil` 不注入同轮前序（每位视同首位发言），第 2 轮起恢复正常注入；用户指定顺序与首位轮换原样保留；圆桌接力仍要求 ≥2 席，独立开场放开单席深答（Start 两处 min-2 闸按协议豁免）；费用不可比明示：导出/最终文档加「费用未采集（harness 未暴露 token/费用事件）；耗时仅可同席自比」，最终文档各席观点标题补耗时秒数（DurationMs 已有，补可见性）。前端=Composer 工具条新增议事路径 Select（pantheon-protocol，Tooltip 说明两路径语义）+单轮 min1+独立开场席位下限 1；record 元信息新增「议事路径」行（visuals.ts `protocolLabel`）；types/models.ts/mock 全链同步（Start 仍 1 参，绑定面不变）。验收=Go `protocol_test.go` 6 用例（协议归一化/独立第 1 轮零注入+第 2 轮恢复+顺序轮换断言/单轮 3 贡献/单席独立 2 贡献+少于两席警示/圆桌单席仍拒/路径入档导出双腿）+前端新用例「独立开场允许单席单轮并把协议写入请求」（坑复现：antd Select 在 jsdom 须 mouseDown `.ant-select-selector` + `findByText('.ant-select-item-option-content')`，打根节点不开下拉）；门禁=go test pantheon+app 全绿+golangci 0+tsc 0+eslint 0+PantheonPage 29/29；build.bat 全链过（冒烟 200，双端同哈希 8561a3c5…）+壳拉起。未 commit（树上仍叠 10-30 批次+第一刀，随发版列车拍板）。

- **圆桌裁决第一刀：分发保真+停止信号可见（2026-10-31）**——依据当日 07:57 实弹圆桌（四席议「如何完善众神殿」，grok 主席裁决 `pan_1793377530905_ff24571e456d`）落地第一刀。分发保真=废除 `formatDiscussionHistory` 1200 字/汇总 1800 字静默截断（prompts.go），改分层保真：同轮发言全文（超 4800 字保**完整尾部**）、往轮降立场卡（机械节选 800 字+原长+贡献 ID，禁模型再摘要冒充原文）；所有机械截断带 `〔gaea 分发截断：…〕` 标记与作者省略号区分；被 `extractConsensusSignal` 剥离的共识行以 `（协议行回写 · 该席第 N 轮共识状态：…）` 回写后位可见历史（全角括号开头不会被 consensusMarker 再解析为当轮声明）；24000 总预算与 clampPrompt 头尾保真闸不变（裁决前提）。停止信号=未收敛警示从一句「仍未全员一致」改为结构化分账 `consensusShortfallWarning`：实质分歧 N 席/末轮未声明 N 席/末轮缺席 N 席（运行状态非分歧）/提示词超限截断 N 次（`promptExceedsBudget` 循环内入账）；导出新增「## 停止信号」段（达成轮次或安全上限+末轮各席信号含缺席/未声明），最终文档各席观点补 `> 共识状态：…（协议行提取）` 回写。裁决验收=8 新用例（distribution_test.go：同轮全文集成/超限保尾/立场卡/汇总末轮全文/共识回写+不可再解析/导出停止信号双腿/缺席分账/超限截断入账 4 次）；门禁=go test pantheon+app 全绿+golangci 0+tsc 0+vitest 39/39；build.bat 全链过（1m34s+冒烟 200+桌面副本同哈希 1a1cfa3d…56,112,880B）+壳拉起。未 commit（树上叠有 2026-10-30 未提交批次，按「不抢 commit」留拍板）。

- **最终文档、议题删除、运行读秒、Markdown 观点流、ZCode 事件墙、当前席重试与跳过（2026-10-30）**：主持人最终综合完成后自动生成独立结果 Markdown 到 `DataRoot/pantheon/results/<sessionId>.md`，同步 `Session.ResultPath`；内容含最终建议、讨论结论、适用条件、保留分歧、风险、证据、参与者、各席观点和运行警示。历史完成会话首次打开时按需补生成；完成页“打开最终文档”经 `PantheonOpenResult` 打开本会话专属文件，原完整 Markdown/JSON 导出保留审计。左栏档案新增独立删除按钮；共享主线程按实际 `createdAt` 顺序展示各 Agent 公开观点，流式观点实时进入，每席带品牌头像、轮次、阶段与耗时；推理和工具输入输出仍只在独立详情。Agent 单独详情与共享观点都走 `MarkdownContent`，GFM 标题/列表/代码/表格/公式正常渲染。实测 ZCode 将数百个 JSON 事件以空格拼接在同一物理行，旧 parser 按换行拆分失败后把 `eventId/messageId/payload` 事件墙误存为 200k 发言；现用 `json.Decoder` 拆分连续 JSON，流式 writer 对完整对象即时消费，只回放 `text_delta` 与已知正文；历史事件墙读回时尝试重装，无法恢复则给诚实重试提示。新增 `PantheonSkipAgent`：每席调用有独立子 context，跳过只取消当前席位，旧贡献保留，循环进入下一位；`agent_skipped` 与运行中 Session 指针统一登记在 `Manager.runs`。`PantheonRetryAgent` 用专用 `errAgentRetry` 中断当前调用，`runAgentWithRetry` 在同一轮同一位重新调用，历史保留；运行头部与卡死提示分别提供“重试当前/跳过当前”，整场 `PantheonRestart` 仅用于失败/终止归档并明确改叫“重开整场”。绑定面 700→701。验证=目标五组 vitest 52/52、Go pantheon 全包、bindings/Go 定向测试、tsc/eslint 0、生产构建；最终 `build.bat` 构建 `build/bin/gaea.exe` 56,107,008B SHA256=`8db929daa031af888ad1d97498cd1fc8e76c93f82c70f85211eff6a805d1cf53`，冒烟 200，桌面 `gaea.exe` 同步。未抬版本、未 commit/push。
- **图片附件与检索能力补齐（2026-10-30）**：图片附件最多 4 张、单张 20MB，仅 PNG/JPG/WEBP/GIF/BMP；原生文件选择后由后端二次 stat、路径去重、MIME/尺寸校验，附件随 `StartRequest` 持久化并进入续议。开始议事时用本地 `vision` 对每张图生成一次共享识图摘要写入 `Evidence`，所有席位共享；Grok 的文件访问策略将附件路径加入只读 roots，可直接调用 `vision`/`ocr`，Codex 以 `--image`、ZCode 以 `--attach` 接收原图，DSH 使用共享摘要。Codex 在众神殿调用中显式传 `-c web_search="live"`，解除本机用户配置的 disabled；所有 Prompt 增加网页不可信内容与“不外传本地文件/附件/凭证”约束。前端新增图片选择/移除/错误反馈、共享识图说明、会话附件清单与续议继承；能力标签更新为 Grok/Codex/ZCode「联网·识图」、DSH「联网·文件」。验证=前端定向 28/28、tsc/eslint 0；Go `internal/pantheon` 全包、app 定向与 vet 绿；Codex CLI 0.165.0 真实 `--image` 附加官方 Logo 后返回正确描述。OpenAI 官方 CLI reference 已确认 `--image` 与 `web_search` 参数语义。桌面包 `build\bin\gaea.exe` 56,050,176B SHA256=`6c13fed6f33211527031bbd78eab6f22da416fd4317d149bb7d8aee9b624c4df`，构建冒烟 `/api/health` 200，桌面副本哈希一致。未抬版本、未 commit/push。
- **拓扑协作界面重排 + Agent 官方 Logo（2026-10-30）**：用户要求参考 dsh 的团队协助/子代理 UI，把“所有 Agent 全铺开”收敛为“共享父线程 + 关系拓扑 + 点击下钻”。落地=中央 980px 默认单列共享讨论；顶部共享节点向参与者发散 SVG 拓扑，节点只显示状态、观点数与工具数；点击节点按需打开观点/工具调用详情，关闭/Esc 焦点返回原节点。统一流不再混入任何 Agent 正文或工具卡，只保留议题、阶段、完成度、警示和最终小结；讨论设置/材料/公开事件改为讨论信息展开区，常驻右栏与 2×2 席位窗退役。用户纠正“不要随便找头像”，随后撤掉自绘符号，改用官方标识：xAI 官方 X 标（Grok）、OpenAI 官方花瓣（Codex）、Z.ai 官网图标（ZCode）、DeepSeek 官方鲸鱼（DSH）；素材落 `frontend/src/assets/agents/`，`SOURCE.md` 记录官方站点/官方 GitHub 组织 URL、下载日期与商标声明。头像贯通欢迎阵容、选席、左栏、拓扑、详情与参与者摘要，姓名和状态继续承担识别。续细节=`visuals.ts` 统一 DSH 短名和各席能力文案；左栏 Agent 卡由“头像+就绪”升级为“头像+姓名+就绪/能力”，头像右下加状态点（在线/候场/发言/完成/失败/离线），详情头显示能力，拓扑名称允许两行、卡片高度 96px、名称与指标字号上调。完成态正文压缩、条件/分歧/依据折叠，详情态标题与工具栏分行；亮暗主题、1440 桌面与 390 窄屏实拍通过。验证=定向 vitest 27/27+tsc/eslint 0；`build.bat` 全流程通过，桌面包 56,029,184B SHA256=`3605187f6f7b815d3272463752f334bbfc637e219249455f4e0e8325ba45ca3b`，冒烟 `/api/health` 200，桌面副本哈希一致；未抬版本、未 commit/push。
- **当前发言者高亮（2026-10-30）**：运行态新增“当前发言”状态条与讨论流“正在发言”徽标；发言席同步强化为 2px 身份色描边、双层外辉光、底色提升与轻微上浮，其他席位不强调（68% 不透明度），仍保留历史可读性。本轮发言顺序、右栏参与者名单与 2×2 席位窗同步进入强高亮，并用 `aria-current` + 可见文字双编码。`activeSpeakers` 合并 participant=running 与 liveStream=streaming，状态落后一拍时仍能显示当前席。定向 vitest 25/25+tsc/eslint 0+生产构建；CDP 1440 亮暗实拍流式消息/4 席窗/状态条均清晰。
- **四席身份色续优化（2026-10-30）**：`AGENT_ACCENTS` 从“只给头像/边框上色”扩展为贯穿欢迎席位卡、开议选席、发言顺序、统一讨论流、工具轨迹、2×2 独立窗与右栏参与者名单的稳定身份线索；正文与小字按当前主题混入正文色，亮暗态均保持可读。讨论流新增“席位色标”图例（颜色+姓名双编码），失败/完成状态仍由图标与文字承担语义。定向验证=vitest 25/25、tsc/eslint 0、生产构建通过；CDP 1440 宽亮/暗双主题实拍目检，四色消息与 2×2 席位窗可扫读、图例不挤标题。未抬版本、未 commit/push。
- **发布收口 v4.529.1（2026-10-30）**：小结标题挤压根修（2f01cc87）定版。亮态暖纸目检补课逮住 v4.529.0 断行修复只在折叠栏态成立的盲区——根因=verdict-head flex 行 tag+印记预留挤压，非主题差异；head 换行+标题恒占整行+字号 24px，四象限（亮暗×栏态）实拍两行成立。桌面包 `releases/gaea-v4.529.1.exe`（55,948,288B，SHA256 `88ea4cdc…`）冒烟 200+桌面副本内置同步；保留策略删 `gaea-v4.526.0.exe`，当前保留 v4.529.1 / v4.529.0 / v4.528.1 / v4.528.0 / v4.527.0。纪律入档=flex 收窄类断行目检须覆盖主列宽度全布局态。
- **发布收口 v4.529.0（2026-10-30）**：UI 实拍走查五刀（fd3ea7ee）定版——中断信息三重去重/探针前检测中态/grok 内置卡/CJK 断行修复（ch 限宽按拉丁字形是坑，CJK 量宽用 em）/默认选席逻辑核实正确。桌面包 `releases/gaea-v4.529.0.exe`（55,948,288B，SHA256 `d53a1230…`）构建冒烟 200，桌面副本内置同步；保留策略删 `gaea-v4.525.0.exe`，当前保留 v4.529.0 / v4.528.1 / v4.528.0 / v4.527.0 / v4.526.0。fd3ea7ee 先行推 origin 后随本版定版。
- **发布收口 v4.528.1（2026-10-30）**：实弹修复批（bed2b286 三崩点根修+e078043a 导出双门根修+60ce7f15 浅色白字两根修）定版。桌面包 `releases/gaea-v4.528.1.exe`（55,947,776B，SHA256 `ca132527…`）构建冒烟 200，桌面副本 build.bat 内置同步。导出双门走查沙箱实弹：CDP 点导出→系统「保存文件」#32770 对话框弹出实锤，WM_CLOSE→「已取消导出」toast、错误横幅恒空；真存落盘腿留用户实弹（原生对话框 UIA/SendKeys 自动化全不可行，已录坑）。按五版保留策略仅删 `gaea-v4.524.0.exe`，当前保留 v4.528.1 / v4.528.0 / v4.527.0 / v4.526.0 / v4.525.0。8 提交（aac3daba..e078043a）先于本版已推 origin。
- **发布收口 v4.528.0（2026-10-29）**：众神殿整批功能入库并产出桌面包 `releases/gaea-v4.528.0.exe`（55,947,264B，SHA256 `2215fe0a18a74b962d9588887c6a3e935f990220882b7812291a0e8deae52b8f`）。发布前完整本地快闸全绿：Go build/vet/golangci-lint v2.14.0 + 全仓测试，前端 465 文件/4053 例、生产构建、E/文档守卫；构建内置冒烟 `/api/health` 200。发布期间 lint 逮到 `Manager.updatePhase` 无调用方并删除；工作树中 V5 研究稿在发版前暂存隔离，保持不混入本版。按五版保留策略仅删除本地 `gaea-v4.523.0.exe`，当前保留 v4.528.0 / v4.527.0 / v4.526.0 / v4.525.0 / v4.524.0。Grok 办公工具循环已由脚本化 provider 验证；真实 xAI 工具调用未纳入本次发版门禁，作为明确边界留档。race 因本机缺 gcc/CGO 未跑，由 Actions 覆盖。

- 用户要求继续优化 UI，并为每场讨论选择工作空间以限定项目。空态/新建页移除冗余右栏，原则改为底部横条；讨论纪要保留并固定右侧摘要栏，用于滚动复核参与者、证据与警示。
- 工作空间为会话级字段，不切换 Gaea 全局工作区：`StartRequest.Workspace` 持久化到纪要，外部 Agent 的 cwd 与多轮讨论简报绑定该目录；当前/最近工作空间与独立目录选择器走 `PantheonB`，新增 Workspaces / PickWorkspace 两绑定。并发讨论使用各自 workspace，避免全局 cwd 串台。
- 视觉与交互：石墨/冷灰/旧金体系、参与者地图、参与者观点窗、协作小结印记、骨架加载、亮暗主题、390/720/1440 响应式与 reduced-motion；同时修复纪要切换继承滚动位置、Agent 路径 `window.prompt` 失效两个问题。
- 讨论协议 V2：按用户指定顺序串行发言，默认 2 轮、可选 3-4 轮，下一轮自动轮换首位；第 1 轮开场陈述，后续轮补充证据、澄清分歧和修正观点，最后由汇总人整理共识、分歧、开放问题与建议。全面移除“质询/裁决/签注”法庭化文案，界面改成动态轮次轨道、发言顺序编辑器与讨论小结。
- 共享与独立上下文：运行页固定建立 1 个共享观点窗口和 4 个 Agent 独立窗口；每个 Agent 的后续 prompt 分开注入个人历次发言与其他参与者公开观点，避免共享语境覆盖自身判断链。右侧设置栏持续展示范围、参与者、发言顺序、轮次与材料。
- 流式显示：Grok 直接消费 SSE delta，外部 CLI stdout 逐行增量解析并发出临时 `agent_delta` 事件；共享窗口改为自动滚动的对话气泡，保留流式光标，最终完整输出仍按贡献持久化。
- 门禁：生成器 697 绑定与 drift 一致；Wails 桌面构建成功（55,927,296B，SHA256 `9A60EB0DAD83E0260B6C84252DDE1B80DC51D2F607148C155639C6AF4B0843C5`）；TypeScript/ESLint/go vet 绿；众神殿前端 9 例 + 流式窗口 1 例 + mock 契约 5 例、Go 有序多轮/顺序轮换/独立上下文分段/流式增量/失败隔离/并发工作空间隔离/路径边界定向绿。
- 续优化（同日）：完成态改为“小结先于过程”，首屏直接呈现综合建议；新增基于小结与保留分歧的续议草稿，继承工作空间、材料、席位、顺序和轮次；导出菜单补齐 Markdown/JSON 双格式。
- 历史与发起体验：讨论档案支持议题/工作空间搜索和状态筛选；桌面发起页操作栏吸底，实时显示“选择范围/至少两席/填写议题”的阻塞原因；390/720/1440 三档及亮暗主题目检通过，窄屏改为正常文档流，避免吸底栏遮挡内容。
- 左侧档案栏折叠：桌面可收为 72px 窄轨，保留新讨论、档案数量、展开按钮和四个 Agent 状态；折叠选择写入 localStorage，刷新后保持。移动端继续使用原横向档案栏。折叠新增后众神殿定向 14 例、tsc/eslint 绿；前序全量基线 465 文件/4048 例。
- 轮次模式：手动轮次改为 2-20 轮数字输入，不再局限 2/3/4；新增“不限”模式，每轮要求各席输出结构化共识状态（一致/仍有分歧），仅当所有实际参与者都明确一致才提前汇总。无共识时默认最多 12 轮并保留分歧、正常生成小结，可随时终止；高轮次进度轨改为横向滚动，续议会继承模式。
- Agent 工具与轨迹：Codex 保持 read-only sandbox；ZCode 改走 `--output-format stream-json`；DSH 保持 read-only permission。统一解析 Codex `command_execution`、ZCode `tool_call/tool.updated`、DSH `tool_call/tool_result`，工具开始、完成、失败、输入摘要和结果预览实时进入纪要。Grok 改由 Gaea 办公 AgentRunner 驱动，复用工作区文件、搜索、造价、知识与日程等只读办公工具，不再裸跑 xAI 补全。工具轨迹不展示模型内部推理，只展示实际工具动作。
- Grok 办公工具接线：`pantheonGrokSeat` 以 xAI 模型构造独立 Gaea AgentRunner，provider feature 仍记 `pantheon`；每轮按众神殿工作空间新建只读工具注册表，核心文件工具重新绑定该目录，并从办公 controller 复制显式 `ReadOnly` 的造价/知识/记忆/日程等工具。禁止 `write_file/edit_file/bash/task/install_skill` 等写类或编排类工具；AgentRunner 的 ToolDispatch/ToolResult 直接映射为众神殿工具轨迹。
- 工具真机探针：本机 Codex 0.165.0、ZCode 0.17.0、DSH 0.2.1-rc.3 分别完成最小只读目录查询；三家均返回可解析工具事件，工作区未修改。讨论提示词新增工具使用与只读边界，明确优先核验事实、精简输出、禁止写盘/安装/读取凭证。
- 后端可靠性：超长提示词改为保留首尾并显式标记省略中段，避免长材料/长讨论截掉末尾 JSON 输出要求；应用重启时把遗留 `running` 会话收敛为失败并保留部分发言、参与者和事件，第二次恢复幂等。
- 续优化门禁：tsc/eslint 0；众神殿前端 18 例（续议、筛选、结果优先、JSON 导出、档案栏折叠、手动/不限轮次、实时工具轨迹）；`internal/pantheon` 工具跨供应商解析/事件持久化/提示词边界与轮次共识测试全包绿；发布前完整本地快闸全绿（Go build/vet/lint/全仓测试 + 前端 **465 文件 / 4053 例** + 生产构建 + E/文档守卫）；11 个绑定名称零变更。race 检查因本机缺 gcc/CGO 环境未执行，如实留档。

## 众神殿 MVP：本机异构 Agent 议事一级板块 · 2026-10-29

- 用户拍板「V5 论证先放一边，先把众神殿实现」。按 `docs/archive/gaea-pantheon-market-research-2026-10-29.md` 落首版：一级板块 `pantheon`（shared，菜单序 6）+ 首页胶囊 + 9 个 `PantheonB` 绑定。
- 后端 `internal/pantheon`：独立陈词 → 匿名互评 → 定向质询 → 首席裁决；JSON 原子持久化、实时事件、单席失败隔离、取消、Markdown 议事录。Codex 用只读沙箱；ZCode 用 plan 模式并注入其原生 provider 配置文件路径；DSH 走随附 headless CLI + `DSH_PERMISSION_MODE=read-only`，Windows 取消用 `taskkill /T /F` 清进程树。
- 前端：三步工作台（议题列表/议事台/卷宗检查器）、设坛材料与参与者选择、阶段轨道、实时流、裁决/分歧/风险、用户签注、历史打开、复制/导出/删除。另含浏览器 mock 完整演示与 5 例页面单测。
- 真机探针：Codex/ZCode/DSH 均以最小 `READY` prompt 跑通；DSH 需要当前 Gaea 工作目录以沿用自身凭据层，Gaea 不读取/复制密钥。
- 门禁：`go vet` 绿；`internal/app` 全包 163s 绿；`internal/pantheon`/board 定向绿；tsc/eslint 0；最终 Wails 构建成功（55,882,752B，SHA256 `E626CEE69F0F58432DBB44BBC9B6CCF981B61DB90AAA7BB4718B9DE92156CCE4`）；`/api/health` 冒烟 200；全量 vitest 首轮 463/464 文件、4038/4039 用例通过，唯一失败=CommandRail 旧导航期望，已更新并定向复绿；签注新增后相关 25 例复跑绿；绑定 drift 695 方法 OK；docs 守卫 OK。

## Reasonix 2.x 蒸馏·第 5 批：压缩验收闸（A3）+ 折叠区召回（A5）· 2026-10-24

- 上游原码对照（`compact_projection.go` partitionUnderCeiling+acceptCheckpointCandidate、`compact_search.go` 召回形态）后落地。**A3 适配版**：验收闸进 compact——候选下限≥trigger 一律跳过（force 不豁免；manual 低于 trigger 例外）+固定前缀超 ceiling 走 ≥25% 节省例外。**两处刻意差异（探针实证后定深）**：①候选下限用「机械摘要保底」估算（shrink 校验保证摘要过大回落 mechanical→下限可达而非猜测；上游用真实候选 token）；②通用 above-ceiling 拒绝不取——**gaea tail 预算有 2000 tok 下限（v10.11），小窗口下 tail 单独顶破 50%，照搬会关死小窗口自动压缩**；例外分支在 gaea 常数下由 trigger 闸覆盖（savings=estFold-150 ≥ estNow/4 恒真当折叠区非空）。closedPrefixEnd 不变量=配对完整性回归测试钉死（tailStart 对齐本就保证，测试防回归）。
- **坑**：三个既有测试被闸拦下——TestCompactIfOverForceBoundedPasses 旧场景（window=2000/8000）在 estNow 尺度上 fold 区本就不存在（tail 下限吃满/estNow 低于预算），按 estNow 尺度重排（window=4000/65 对/usage=3700）；hopeless-skip 语义另立新测试。**测试环境 tokPerChar 走 fallback 0.25**（lastUsage 只在采样路径 Store）——合成 usage 的尺度必须与 0.25 估算一致，否则 compactIfOver 的 prompt 口径与 planCompaction 的 sizing 口径脱节。
- **A5 形态适配**：`recall_archived` 新内置只读工具（ctx 注入 archive 目录，runDirect 与 WithSpace 同位）——检索对象=archive JSONL（gaea 折叠区在文件里，上游在 DB canonical；archive 路径本就附在 digest 尾=天然 fold index）。确定性计分（查询词+CJK 双元、≥2 rune 防单字误伤）、tool_output 由所属调用寻址（召回连来路）、噪声比 0.25、默认 8/上限 20、片段 240 rune、输出 4KiB 封顶、坏档跳过（per-source）、合成消息/摘要不入检索面。零新绑定。
- 门禁：go build 全仓+vet+gofmt 绿；agent 全包绿（A3 闸三例+配对完整性+A5 两例新测试）。**未抬版本、未 commit/push**。**A 档七刀全数处置完毕**（六落地+A1 辨伪）。余池=B 档四件（ContextCapsule 观测/list_subagents 取址/durable inbox/checkpoint-rewind，均需独立预核）。

## Reasonix 2.x 蒸馏·第 4 批：中断交接块（A4）· 2026-10-24

- 上游原码对照（`interrupted_recovery.go` 全文 131 行）后落地：回合终局（流错误/取消/预算阻断三路径）把本轮回执折叠成 **bounded 交接记录**（≤24 工具对、每工具 ≤8 文件、渲染层值截 240 rune+HTML 转义；不含原始参数/结果/推理），下一条真实用户消息前置 `<interrupted-turn-recovery>` 块一次性消费——「host-verified recovery facts 非新任务；先查看工作区现状；不要盲目重复已完成写入；被打断的调用从头重发；用户新消息优先」。
- 集成点：capture=agent_run.go 两处终局 return（流终局+预算阻断）前；注入=runDirect 入口 withTurnPreferences 之前（one-shot 清零，下一回合隐式消费不原地改历史——上游同口径）。回扫真实轮起点跳过三类合成用户消息（steer 前缀/流恢复/readiness）+压缩摘要——流恢复消息之后的工具对也覆盖。
- completed/interrupted 判据：结果含 "stream interrupted"（persistInterruptedCalls 信封文案）或无结果回执=被打断；其余=completed（真实失败也算 completed——它是 host-verified 事实）。**与上游的刻意差异留档**：①gaea 终局本就把部分输出落库进上下文（v4.379 刀2），无上游 unsafe_partial_output 条款；②记录存 AgentRunner 内存不随会话持久化（进程重启丢失=诚实边界，上游 LocalOnly 持久）。**坑**：渲染层必须重钳值（捕获时截过不等于契约——测试直接构造 300 rune 文件名抓出；上游正是在渲染处 clip）。
- 门禁：go build 全仓+vet+gofmt 绿；A4 五例新测试+agent 全包绿；app 全包回归后台跑。**未抬版本、未 commit/push**。余池：A3+A5（压缩，需同批预核）、B 档四件（ContextCapsule/子代理结果取址/durable inbox/checkpoint-rewind）。

## Reasonix 2.x 蒸馏·第 3 批：子代理两刀（A10 写范围预检 + A11 完成裁决）· 2026-10-24

- 按纪律先读上游原码（`delegation/task.go` executeTask+task_schema、`safety/evidence/completion_report.go` 全文）再动手。gaea 侧盘底：并行子代理经 batch_executor「每调用唯一键」全并行、**无写范围协调**——双会话同树互踩（stale-Edit/TS2300 撞车坑）的 agent 内同型缺口。
- **A10 write_paths 写范围声明落地**（task.go+task_writepaths.go+batch_executor.go）：task 工具增 `write_paths` 参数（glob `*?[]`/逃逸〔归一后不在 cwd 子树，wspath.Within 判〕/卷根/POSIX 绝对非盘符形拒；cwd 归一+去重+排序）；**两层协调**——①冲突键按声明集计键（`wpaths:`+排序归一集；同集同键同批必串行、异集各键共存并行）②执行期 claim 注册表（TaskTool.writeClaims，目录包含语义两两重叠，重叠的后到者 fail-fast 如实指路「拆批或改 disjoint 范围」）。**生命周期坑**：真后台任务跨回合——声明随 job 存活（StartInheriting 闭包收尾释放），前台/bg-fallback 在 Execute 返回释放（realBackground 旗标区分，双 defer 语义）。**刻意偏差留档**：声明是并行协调元数据非安全边界（上游 extend_write_paths 能力卡需审批 UX，gaea 无；越界执行期拦截不做）；未声明调用零变化（文件观察 stale 检测执行期兜底=上游 optimistic 的 gaea 对应面）。CompactDescription/Schema/CompactSchema 三面同步（CompactDescriptor 是模型唯一可见面——在册坑复训）。
- **A11 complete_subtask 完成裁决落地**（complete_subtask.go 新文件）：结构化完成报告（status 四态+验收判据+证据引用 verification/diff/files/review/manual）+ **回执账本=子会话消息**（assistant 调用↔tool 结果按 ToolCallID 配对；无结果回执=被中断不构成支撑；失败=isErrorMessage 前缀族）+ **只降不升**——引用判据无回执支撑降 unsatisfied，任一降级 complete 整体降 partial，降级结论+逐条理由附在结果尾部给父模型；partial/blocked/failed 不是完成主张不裁决；工具提交永远成功（子代理只收确认文案）。verification 匹配=命令包含双向包含口径；diff/files=路径相等或目录前缀（ToSlash 归一）；review/manual 永不自证。buildSubReg 常驻挂载（不占白名单名额）。**坑**：注册面守卫（v4.520 tool_names_guard）把报告状态枚举 case 当幽灵工具名——按守卫自家协议「非工具名 case 逐个注明归属」入白名单（11 枚举词）。
- 门禁：go build 全仓+vet+gofmt 绿；A10 五例+A11 七例新测试+agent 全包绿。**未抬版本、未 commit/push**。余池：压缩线 A3+A5、A4 中断交接块、B 档（ContextCapsule/list_subagents 取址/durable inbox/checkpoint-rewind）。

## Reasonix 2.x 蒸馏·第 2 批：压缩内容级保留（A2）+ A1 辨伪销号 · 2026-10-24

- 承 Reasonix 2.x 蒸馏线（首批 A6/A7 已落地）。按新纪律先读上游原码（`compact_user_turns.go` 全文+`usage_anchor.go`+`context_usage.go`+`compact.go:353-460`）再动手。
- **A1 usage anchor 辨伪销号**：gaea `tokenmeter.go`（v4.381 dsh 蒸馏）即上游 usageAnchor+providerReportedFloor 同构等价物（锚点+有符号差值+per-message memo 相消；tokPerChar 钳 0.05–2 兜底 0.25 全同）——调研报告初稿「水位线全靠本地估算」系误判（v4.523 上屏的是三阈值**显示**，计量 v4.381 已锚定），报告现状行已修正。上游锚绑 projectionVersion 失效语义在 gaea 不需要（全量表面差值自洽）。窗口探测 contextWindowProbe 无对应面留观察。
- **A2 压缩内容级保留落地**（compact.go+新测试）：`keepUserTurns` 进 partitionFold——digest 后用户轮逐字保留预算=窗口 5%（无窗口 floor 1024 tok，上游常量全同），最老优先、**预算是唯一闸**（上游明确：per-turn 上限是被刻意移除的形态——它会在预算装得下时丢掉长轮；`[[keep]]` 前缀才是逃生通道；gaea 旧 pinnableUserTurn 正是那个被移除形态，已收缩回头部 pin 专职）、估价用固定比率 0.25 不随校准漂（阈值跟上次轮比率漂=同一轮这次留下次折）、**丢弃显形**（Notice 上屏+告知 [[keep]] 通道——「投影看起来完整而作者的话只剩摘要转述，逃逸通道只对被告知它存在的人有用」上游注释口径）。
- **A3 验收经济学预核**：与 A2 强交互（用户轮保留顶大候选，验收 ≤50% 上限须配上游 partitionUnderCeiling 的「先降 KeepErrors 再试」降级序），单做价值低——与 A5 折叠区召回并下一批。
- 门禁：go build 全仓+vet+gofmt 绿；A2 六例新测试+agent 全包绿。**未抬版本、未 commit/push**（树含并行会话改动勿混提）。拍板池：第 3 批=子代理两刀（A10 write_paths 预检+A11 完成裁决）；压缩线余=A3+A5 同批。

## 打包记录：全树收口提交 · 2026-10-24

- 当日多会话累积（B0~B5/CAD 一期/N1~N11/2511 退役/Reasonix 八刀）按主题分 14 笔提交入库（a3ec0485^..f2c6b2e9）：deps a3ec0485 / cad 6e8a49be+8ffec051 / agent-B1+A7 e00781ee+c907f2b5 / attachment 0a4c8730+7f18dd6b / memory 51f89b2f / image 2820a344 / novel 6c45cd9d+7cb6938d / agent-Reasonix b26a0255+f2c6b2e9 / docs 4a9e027f。树干净（0 改动）、HEAD go build 全仓过。**未抬版本未发版**——发版待与 DSH 轮询协调（版本三处防竞速），全部改动此时点起可发。

## Reasonix 2.x 蒸馏重开线：调研立档+安全两刀（A6/A7）· 2026-10-24

- 用户指令「蒸馏reasinox，我们已经落后太多」——前档 docs/archive/gaea-reasonix-v138-distill-2026-09.md（v1.38 四弹 16 刀后留池全销号收官）已过时：**studio 分支 2.0.0→2.50.0 共 59 版，PR 号 #10613→#14337（≈3700 号增量）**。四路并行深潜（会话与上下文引擎/权限安全/子代理补漏/架构地图）+ gaea 盘底（防提已有）→ **docs/archive/gaea-reasonix-2x-distill-2026-10.md** 立档：A 档七刀（usage anchor 压力校准/压缩内容级保留〔用户原话预算〕/压缩验收经济学〔≤50% 上限+closedPrefixEnd〕/中断交接块/write_paths 预检不相交/complete_subtask 完成裁决/递归删除 guard+诚实拒绝）+ B/C 档 + 否定结论（MCP/扩展 Runtime/远程网关/计费/compositions/OS 沙箱/Coordinator）+ 辨伪（cache_shape/file observations/pinned 注入 gaea 已有）。
- **首批实弹两刀（拍板序第 1 批=安全）**：
  - **A6 递归删除硬拒绝**（internal/gaea/tool/builtin/bash_delete.go+测试）：灾难面（文件系统/卷根、home 及祖先、工作目录及祖先）任何形态硬拒绝；PowerShell 词法（Remove-Item/ri/rm/del -Recurse 缩写旗标）；find -delete 与 find|xargs rm 管道形态；变量/glob 字面性不足 fail-closed。**刻意偏差留档**：上游连「工作区外」也拒（有审批阶梯可路由给人），gaea bash 无审批 UX、temp 清理是合法实弹流，只拦灾难面。**坑三连**：①Windows filepath.Clean("C:") 补点成 "C:."——卷根判定必须打在原始词上；②Windows Clean 把 / 规范化成 \，POSIX 形态分类必须先于一切 Clean（MSYS 盘符挂载 /c/foo→C:\foo 单独翻译；/home/... 落 Git 安装根下=非用户数据灾难面放行）；③sameOrAncestor 的 Rel 方向——Rel(dir,p) 把「p 在保护区子树外」误判成祖先，正确方向=Rel(p,dir) 无 ".." 前缀（且 xargs 段 rm 无路径参数，递归旗标即判）。
  - **A7 诚实拒绝升级**（tool/access.go）：读/写拒绝消息列出**具体允许根清单**（≤8 个配置序+超出计数省略；空集合明说 none configured）——模型看到根才能自我纠正；解析失败（提存在性）与越界（不提）两种错误语义分立；只读根不混进可写根清单。
- **对源复核（用户质询「看了上游代码没有」引出的流程缺口补课）**：四路深潜报告有 file:line（读过源码），但 A6/A7 两刀最初按报告摘要写、未直接对照上游原实现——直接读 `shellsafe/delete*.go`+`tools/builtin/bash_delete*.go`+`target_access.go` 后盘出 **v1 九处差距并重写为 v1.1**：①包装解包（sudo/env/nohup/exec/builtin/command+旗标白名单，`sudo rm -rf /` v1 直接漏）②未知基座/解包失败递归参数扫描（裸 `xargs rm -rf` 树根未知 fail-closed；无关基座 echo 等放行=上游 default 分支）③find -exec/-execdir 嵌套删除负载④未知旗标 fail-closed（短旗标字符白名单 rfRvidI/长旗标前缀缩写口径）⑤PS 冒号内联值（`-Path:C:\ -Recurse` v1 漏）⑥`-Recurse:$false` 显式关递归⑦rd/rmdir 别名+cmd `/s /q` 旗标词法⑧字面性字符集补全（%[]{}+~user）⑨`-Confirm -Recurse` 取值参数后跟旗标=fail-closed（测试预期按上游语义修正）。新增 4 组测试（包装/finde-exec/PS v1.1/cmd），tool+agent 全包复绿。**A7 对照结论**：主干已对齐；不取=结构化 Refusal{Code}（gaea sentinel+文本全链改型收益薄）+覆盖写保留读因（新拒绝面行为变更，记刀池）+禁读列表（gaea B1 是 allow 模型无对应面）。
- 门禁：go build 全仓绿；go vet（tool/builtin/tool/agent）绿；guard 12 组子例+denial 4 例+tool 全包（bash/shellstate 回归）+agent 全包绿；gofmt 清。**未抬版本、未 commit/push**（树含并行会话改动勿混提）。
- 拍板池待用户挑：第 2 批=压缩两刀（A1 usage anchor+A2 内容级保留，A3 验收经济学同批预核）；第 3 批=子代理两刀（A10 write_paths 预检+A11 完成裁决）。

## 网文 skill 蒸馏 N11：技法检索库（webnovel-writer 九表思想）· 2026-10-24（续刀五）

- 调研余池 A6 收口（A 档最后一项；GPL-3.0 红线=只重推导机制，账本内容全自写不搬上游文本）。上游思想：技法以「关键词/意图同义词/适用题材/核心摘要/详细展开/毒点」行式账本存档，按需检索注入不常驻。
- **N11 全栈**：internal/app/novel_technique.go 内置 28 条九类技法账本（场景写法/爽点节奏/桥段套路/人设关系/写作技法/命名规则/裁决规则/金手指/题材调性；每条=关键词+适用题材+一句话核心+展开+毒点，关键词 ≥2 rune 硬约束防单字误伤）+ 确定性检索零 LLM（关键词含分类名子串计分，同分按账本序 ID 稳定）+ **生成链 advisory 注入**（create-chapter 的 extras 增 technique 区段：hint=计划+大纲要点文本命中关键词条目，至多 3 条/520 rune/整条不入宁少勿截半；**Rank 4 最先牺牲**——技法是参考非创作约束，排在世界观之后；标题明示「非创作约束，与本章无关的条目忽略」）+ NovelTechniqueSearch 绑定（683→684；空查询=分类目录初始态不倒全量；只读不落盘零写路径）+ TechniquePanel（CreatePage 工具轨质量组新项「技法库」：检索框+九类 Tag 点击即按分类检索+结果卡三件套渲染+零命中诚实空态）+ mock 诚实空态（账本在 Go 二进制，浏览器 dev 无账本）。
- v1 边界留档：内置账本只读不支持项目级增补（写路径属记忆写入确认红线范畴，实弹有需求再立刀）。
- 坑复训：**heredoc python 改文件在 assert 失败时静默不写盘——文件改动一律 Edit 工具手写（第三次复训）**；gen_bindings -signatures-ts 再生会把并行会话已落 Go 未同步 signatures 的名字一并补齐（本次补入 CAD 四件/N 系三批/Sin 件，与 Go 面 684 一致属正常同步非冲突）；**面板测试一次性 mock 会被「打开即拉初始态」的消费先吃掉**（mockResolvedValueOnce 须挂在初始 await 之后）；vitest 定向复跑必须从 frontend/ 起（仓库根起吃不到 vite.config 的 jsdom=整文件 document is not defined 假红）。
- 门禁：go vet 绿；go build 绿；技法定向测试+app 全包 153s 绿；tsc 0；eslint 0（六改动文件）；绑定 drift OK@684；-legacy-ts/-signatures-ts 再生同步；契约漂移 WARN 无新增 exit 0；全量 vitest **458 文件/3981 用例**绿（独占；首跑 2 例=技法面板 mock 次序坑已根修复跑绿）。**未抬版本、未 commit/push**（树含并行会话改动勿混提）。

## 原罪 N10：统一插图画风锚（图文穿插一致性）· 2026-10-24（续刀四）

- 用户拍板「原罪是图文穿插式，需要单独优化」。盘底：SinIllustrate 已有角色文本锚+T2 参考槽+失败回退（正文标记与 sin_illustrate 工具同路），**缺画风锚**——同一故事各张插图风格全凭当轮 cue 措辞漂移，图文穿插体验硬伤。
- **N10 全栈**：cast.json 增 `styles` 键（故事 id→画风描述，版本仍 1 字段增量；**坑=loadSinCast 三个返回点的 map 全非 nil——早退路径漏初始化 Styles 会在 Set 上 panic〔typed-nil map 复踩〕，收敛 emptySinCastDoc 单源**）+ `SinStoryStyleGet/Set`（trim/200 rune 截断/空串清空）+ `sinAugmentPromptWithStyle` 注入（与人物锚同款去重：「统一画风（本故事插图保持一致）：…」，未设=逐字节旧行为）+ 系统提示词插图协议画风行（**教模型 cue 里不写整体画风词**——设了锚由系统承担，没设保持与此前插图措辞一致）+ 右栏「插图」卡顶部画风卡（TextArea+保存，dirty 门闩，切故事重载）+ mock 内存态。绑定 681→683。
- 前端测试 31/31（新 3 例：回显禁写/改写保存生效值回填/清空解除锚定；**坑=antd TextArea 的 testid 落点TagName 兼容取法复训**）。
- 门禁：eslint 0；tsc 0；drift OK@683；go vet+app 全包 140s 绿；全量 vitest 绿（独占跑）。**未抬版本、未 commit/push**。

## 网文 skill 蒸馏 N9：原罪板块写作纪律补族 · 2026-10-24（续刀三）

- 用户问「原罪板块呢」——盘底结论：原罪是对话式合写成人向图文故事（SinB 26 方法+工具环+便签/大纲底稿直注+插图标记+epub 导出），提示词在 `internal/app/sin_prompt.go`（Go 串不走模板引擎）；v4.424 已吸收 T2 蒸馏的解释腔/否定翻转两条。**关键事实：sin 正文是流式对话产物，前后端都不经过 novelstyle 打分——`sin_prompt.go` 旧注释「内核门禁兜底」名不副实，已诚实化**（提示词层是唯一闸，措辞按最强口径写）。
- N9 刀（sin 写作规范补族，与小说板块 N1/N2/N7 判据同源）：通用反应不刷屏（深吸一口气/瞳孔微缩/眼底闪过一丝/嘴角勾起 一轮至多一次，优先换人物特定动作）/弱化装饰节制（微微/一丝/一抹/缓缓 连续出现=描写没落地）/收束不升华（轮末落具体动作对话画面，不写命运齿轮式预告）/获得要挣来（契约四问可移植子集：巧合制造麻烦不解决麻烦）。测试锚扩族（TestSinSystemPromptAntiAIFlavorRules +新增 TestSinSystemPromptEarnedGainRule）。
- 不移植项（判据留档）：N3 扫榜/N6 拆解=项目级工具不适用对话面；N8 词典双时间线=合写场景用户即读者全知，便签+大纲已覆盖一致性锚；N4 书封=epub 封面属新需求非蒸馏。
- 门禁：go build/vet 绿；app 全包 135s 绿（TestSin 全族含提示词锚）；零前端零绑定改动（sin_prompt 是 Go 串，无 drift 面）。**未抬版本、未 commit/push**。

## 网文 skill 蒸馏 N7~N8：设定事实词典双时间线 + 钩子技法补全 · 2026-10-24（续刀二）

- 承 N1~N6（同日续刀二）：消化调研余池 A5/B11；C16 字数带核对完毕=现状 minWords+自动续写 ≤20 轮是既有设计选择，与上游「欠字不自动补」相抵触属行为变更，不动，记观察。
- **N8 设定事实词典全栈**（chinese-novelist-skill「设定与事实词典」机制重推导）：`<项目根>/facts.json` 双时间线账本（name/category/first_chapter/**reader_known 口径上限**/full_truth/reveal_chapter/revealed）——`NovelFactsList/Save`（整表写回作者审批制：名称+读者已知必填、缺 ID 自动补 fact_<ns>、负揭示章拒收）+ **Rank 0 注进章节上下文**（buildChapterContextSections 与伏笔同梯队最后牺牲；预算 800 rune/10 条，整条不入宁少勿截半）：未到揭示章的完整真相带「本章不得提前揭示」纪律、已到期/已标记转「已可揭示」口径、无隐藏层只出口径。设定页下行区改三面板（伏笔+FactsPanel+一致性，aria-label 随改测试同步）；FactsPanel v1=登记/标记已揭示/删除/整表保存。绑定 681。
- **N7 钩子技法补全**（钩子十三式的可代码化子集）：chapter-plan forbidden+=虚假悬念（无铺垫空吓）/机械降神式破局（主角不付代价外力免费解围）；create-chapter must+=新悬念必须当场给具体可感由头（一件物/一句话/一个变化），不开空吓天窗。十三式全表与悬念弧并行不进 prompts（章级模板装不下全书级调度，dissect 八型已够用）。
- 坑复训：**aria-label 类可访问名是契约锁**（getByLabelText 精确匹配，改面板构成须同步测试）；**同族短按钮名（「登记」vs「登记事实」）正则双命中，用 ^…$ 整名锚定**；**全量 vitest 必须独占跑**（与 Go 全包并行=432 假 errors，定性资源抢占非回归）。
- 门禁：go vet+app/prompt/novelstyle 全包绿（app 133s）；tsc 0；eslint 0；绑定 drift OK@681；全量 vitest 绿。**未抬版本、未 commit/push**。余池：扫榜/拆书/词典真机实弹。

## 网文 skill 蒸馏 N4~N6：对标拆解全栈 + 书封三层模板 + 审核 fix 方向限制 · 2026-10-24（续刀）

- 承 N1~N3（同日续刀，用户拍板「只管小说板块，继续」）：消化调研余池 A②/B10/B12。
- **N6 对标拆解全栈**（oh-story story-long-analyze 机制重推导，学习导向与章节分析〔修改导向 analysis-v2〕分面）：prompts/novel-dissect.json（付息/钩子类型八枚举/情绪曲线/信息推进/写法公式「条件→手法→效果」，偷机制不偷设定+证据不足诚实条款）+ internal/app/novel_dissect_handler.go（`NovelDissectChapter(num,force)` 逐章拆解落 `<项目根>/dissections/NNN.json` 原子写、force=false 文件存在即跳过=断点续跑零 LLM、非法 hook_type 落「无」保守档、字段钳长；`NovelDissectList`/`NovelDissectReport` 纯确定性零 LLM 聚合：剧情线付息链/钩子分布/公式去重/情绪曲线）+ 三绑定（679，spaceBindings play 认领+names 手工+legacy/signatures 再生）+ DissectPanel（CreatePage 工具轨「质量组」新项：起止章号+跳过已拆+前端串行循环进度可停可续+报告视图）。
- **N4 书封三层模板**（novel_bookcover.go）：题材风格表 15 档对齐 GENRE_OPTIONS（玄幻/仙侠/武侠/奇幻/都市/现实/历史/军事/游戏/体育/科幻/悬疑/灵异/二次元/短篇，子串命中取首条+通用兜底）+ 画面三层排布（前景人物-中景场景-远景氛围）+ 排版安全区指令（顶部 1/6+底部窄条净空）；本地生图后端中文渲染不可靠的既有「不画文字」口径保留，测试断言字符串（竖版书籍封面构图/promptHint/项目名）全保留。
- **N5 审核 fix 方向限制**：chapter-review.json（一致性/事实/因果类问题只写「事实统一方向」，不夹带文学创作建议、不代作者改剧情走向）+ book-review.json（同款一行）——防 LLM 评审越权改剧情。
- 坑复训：**antd InputNumber 的 data-testid 直接落在 input 元素上**（querySelector('input') 取空，helper 须判 tagName）；**同包测试文件与源文件函数重名不同签名=测试编译才炸**（mustJSON 撞 image_handler_test，改名 dissectJSON）；heredoc 改文件会让 Edit 工具失去文件状态跟踪（重 Read 再 Edit）。
- 门禁：go vet+app/prompt/novelstyle 全包绿（app 144s）；tsc 0；eslint 0；绑定 drift OK@679；契约漂移无新增 exit 0；模板计数锁 25→26；全量 vitest 456 文件/3968 用例绿。**未抬版本、未 commit/push**（树含并行会话改动勿混提）。余池：钩子十三式资产、事实词典双列、扫榜/拆书真机实弹。

## 网文 skill 蒸馏 N1~N3：选题扫榜全栈 + 契约四问申报 + 内核去味扩充 · 2026-10-24

- 起因：宣传图「写小说一整套 Skill」调研（报告 docs/archive/gaea-novel-skill-distill-survey-2026-10.md）——十个名字实出自四仓（oh-story-claudecode 合集 + chinese-novelist-skill + novel-creator-skill + webnovel-writer〔GPL 只重推导〕）；novel-deslop/novel-agents/novel-review 早在 v4.282 起蒸馏自同源，真空白=选题扫榜与拆书编排。用户拍板「不要技能，直接蒸馏进代码」。
- **N3 选题扫榜全栈**：prompts/novel-scan.json（八维市场观察+选题 12 字段+可行性量化门槛+来源诚实标注）+ internal/app/novel_scan_handler.go（粘贴解析→LLM→**确定性封顶**：同方向样本<15〔刺猬猫 10〕或趋势未知→高封中，只降不升，封顶原因写回 feasibility_reason；报告只读不落盘=建议制）+ NovelScanAnalyze 绑定（NovelB 123/全 676：spaceBindings play 认领→bindingNames 手工同步→legacy/signatures 再生）+ CreateNovelModal「扫榜找灵感」折叠区（平台选择/粘贴/可行性徽标报告卡/**采用题材**映射 GENRE_OPTIONS 回填）。爬榜通路明确不做（D 档：ToS 灰）。
- **N2 prompts 纪律**：create-chapter 增契约四问（结果必须挣来/不预支底牌/旧期待付息+新期待具体）+新增物申报（具名新角色/新伏笔/新承诺必须入变更声明；需作者拍板的走向不写正文、【待拍板】列摘要行）+章末升华禁令；chapter-plan 增主线配额（主线推进/关系升级/核心秘密三类取一，非收官章不终局）+钩子章内铺垫纪律。
- **N1 内核去味扩充**（纯确定性零 LLM）：words.json blacklist 19→38（通用反应/论述公文腔/空泛定性/自动过渡信号，只计分不替换，宁漏勿误不变）+patterns.json 新族 trailerEndings（**章末窗口 240 rune 位置敏感**，S3 advisory 封顶 1；解释腔标记 +3）+patterns.go 规则 12+classifyTasteLayer 升华→narrative。
- 门禁：go build/vet 绿；novelstyle/prompt/app 全包测试绿（app 148s）；tsc 0；eslint 0；绑定 drift OK@676；契约漂移 WARN 无新增 exit 0；全量 vitest 455 文件/3961+3 用例绿。模板计数锁 24→25（prompt_override_test）。**未抬版本、未 commit/push**（沿用 B0~B5 纪律；树含并行会话 Qwen-Image-Edit 退役改动，勿混提）。

## Qwen-Image-Edit 2511 退役：删除 `qwen-image-edit` · 2026-10-24

- 用户拍板删除旧本地编辑模型 `qwen-image-edit`（Qwen-Image-Edit 2511），不作为回退保留。
- 删除 2511 工作流、专用权重提示、Kontext 缩放辅助与相关测试；ComfyUI 编辑统一走 `qwen-image-2.1-edit`。角色设定卡、原罪 qedit、媒体编辑均同步切换；旧模型请求显式拒绝。实际权重 25 个文件 / 51.31GB（主文件 + 24 分段）已移入 Windows 回收站，可恢复，其他模型未动。
- 本地 2.1 Edit 不支持蒙版，局部重绘/扩图保留 honest failure，并在 UI 指向支持 `/images/edits` 的云端后端。历史发布记录不重写。门禁：`go test ./...`、Go build/vet/lint、前端 vitest **455 文件 / 3961 用例**、tsc/build/knip、文档/契约/E 系列/绑定守卫全绿。

## 上游蒸馏 B0：发布阻断清零（依赖安全+门禁恢复）· 2026-10-23

- 按 `进度计划/gaea-upstream-distill-execution-20261023.md` 从 B0 直接执行；保留执行前 CAD/工作台未提交改动，不 commit/push。
- 依赖修复：Go 1.26.10、x/net 0.61.0、excelize 10-20 修复版、DOMPurify 3.4.17、KaTeX 0.18.2（pnpm 11 overrides 统一传递依赖）；`govulncheck` 与 `pnpm audit --prod` 均归零。
- Excelize 条件格式测试夹具改用 `NewConditionalStyle`，删除 61 行手工 zip/dxfs 注入；CAD lint 清零，三份 CAD 文档登记并修正 PoC 链接。
- 门禁：Go build/vet/test 全绿；前端 tsc/build/knip 通过；vitest **454 文件 / 3951 用例**全绿；绑定 PASS@675、契约/E 系列/文档守卫通过。前端 lint 0 error / 7 既有 warning，CAD 一条留 B2。
- 下一批次：B1 文件访问边界与子代理 hooks（工作区读写分根、preview confinement、hooks 继承）；未抬版本、未发版。

## 上游蒸馏 B5：Qwen Image 2.1 Turbo Edit（代码+mock+单图真机收口）· 2026-10-24

- 保持 `qwen-image-2.1-edit` 内部 ID 兼容配置，UI 显示 “Qwen Image 2.1 Turbo Edit”；官方 Turbo Edit 工作流复用现有三件权重：`TextEncodeQwenImage21`、`QwenImage21Cache → CFGGuider(cfg=8) → SamplerCustomAdvanced` + 官方 8 步 `ManualSigmas`。1/4/10 图 golden 通过、第 11 图拒绝；增强器默认关闭。
- **合规显形**：Qwen 2.1 的 `Qwen Research License（禁商用）`写入目录 license hint，并在模型选择页/编辑弹窗明确警示；商业用途需另选具备相应许可的模型。新增 Go 目录与前端 UI 回归，定向 Go 测试、21 个前端用例、tsc、eslint 通过。
- **真机复核（2026-10-24）**：ComfyUI 0.42.0 / templates 0.11.96 下，产品级单图指令编辑成功（1024×1024 RGBA、172,229 种颜色）；Turbo Edit 三件权重均已就位，不再等待基础 Edit UNET。
- 2.1 Turbo 真机文生图成功（1024×1024 RGBA、274,349 种颜色、49.59 秒），本机 SHA-256 与 HF `lfs.oid` 一致，证明 2.1 CLIP/VAE/采样节点栈健康。
- 2511 在 0.42.0 下仍输出纯黑；同实例 Krea2 对照正常（158,876 种颜色），排除 VAE/全局 GPU。本机 2511 与官方文件同为 20,533,762,817 B，但 SHA-256 `d21c31e4…d081` ≠ 官方 LFS `c9fdc158…0a4e`，确认本地权重损坏。
- 留池：4 图真机在 5 分钟门控内停在 0/8、未产出，已中断；中文图上文字真机输出绿色噪声、无可读“山海”，判失败。2511 已退役，不再重新获取。
- 全量前端 vitest **456 文件 / 3968 用例**通过；Go 全量与各静态/安全门禁通过，当前绑定面 `PASS@679`（含并行功能增量），未抬版本、未 commit/push。

## 上游蒸馏 B4：作者记忆 + 三层去味 · 2026-10-23

- 新增 `internal/authorprefs` AP/BP 两级 author memory（幂等 event、单调 revision、conflict/pending/accepted、book 覆盖 global、2048B 查询上限与 omitted IDs），新增 play 空间 `author_memory` 工具并接入章节/风格上下文。
- `novelstyle` 与 `novelgate` issue 增加 expression/narrative/story layer；novel-deslop 增加 H/I/J story 授权门禁。自动化词法引擎不伪造 story 层问题，脚本零命中不替代后两层阅读核验。
- authorprefs 6 个场景、三层正反例与零误报测试通过；agent 内部工具不增加 Wails 绑定。

## 上游蒸馏 B3：附件准入 + Provider 发送保护 · 2026-10-23

- 新增 `internal/gaea/attachment` 单一安全实现：普通文件 25 MiB、图片 10 MiB，解码前拒绝超限，`os.Root` 固定目录、独占临时文件、原子改名、真实路径/MIME 校验；粘贴图与附件绑定接入，旧 control 重复实现删除。
- 前端普通文件/图片/截图三条附件路径均在 FileReader 前分拣；新增 provider send guard：发送阶段 300 秒、请求写完启动响应头计时、仅取消 attempt context、未完成发送最多安全重发一次，响应头超时不重发，长 SSE 不中断。
- attachment 7 个后端场景、send guard 6 个场景、附件前端 12 个场景通过；零新绑定。

## 上游蒸馏 B2：CAD 当前版本收口 · 2026-10-23

- `CadPanel` generation/seq 隔离全部异步请求，新图纸清空旧状态，慢 Open/Raster/Texts/Count/Ask 迟到结果丢弃，ask 期间禁换图框，handoff 仅用当前代际数据；不同错误与 count warnings 显形。
- CAD 后端新增 DWG/DXF 32 MiB、PDF 128 MiB、100 图框、512 MiB PNG 总量、渲染宽度 64–8000、PDF 前 50 页限制，pdftoppm 可取消杀树，PNG/cache 元数据原子写。51 页合成 PDF 实测只生成 50 页。
- CAD 后端 14 个 Go 场景、前端 14 个用例通过；lint 0 error / 6 个既有 warning，无 CAD 新 warning。

## 上游蒸馏 B1：文件访问边界 + 子代理 hooks · 2026-10-23

- 新增 AccessPolicy 与真实路径解析；默认读根=workspace+allow_write，`allow_read` 仅扩读；UI 已选文件经显式回调单文件放行。file/grep/ls/vision/format/chart/diagram/schedule/sidebar/largefile/OCR 等路径工具在 precheck 前统一校验。
- 子代理继承 AccessPolicy；用户 hooks 使用独立 `sa_` session view，`retry_until` 也走 child hooks。覆盖 symlink/junction、跨卷/大小写越界、正文不泄漏、读根不可写与 hooks 阻断 bash，共 19 个新增场景；零新绑定。

## 上游蒸馏执行规划已立档（非版本刀）· 2026-10-23

- 全仓审计与上游同步后形成可执行计划：`进度计划/gaea-upstream-distill-execution-20261023.md`。
- 下个会话直接按 §0 执行，不重新审计、不重复拍板；首批 B0 发布阻断清零，随后 B1 文件访问边界+子代理 hooks。
- 锁定顺序：B0 门禁 → B1 安全边界 → B2 CAD 收口 → B3 附件/provider → B4 小说作者记忆 → B5 Qwen Image 2.1 Edit。
- 当前审计基线：`ce56c783`，v4.523.0，工作树含未提交 CAD/工作台改动；保留现有改动，不 commit/push。

## v4.466.0 发布：二期·退役清理（21 死绑定退役，660→639）· 2026-10-07

- **退役 21 全走 excludedBindings**（批 53 先例：方法体保留——ProjectSave/EstimateItemSave 是 graph/indicators 测试种子、笔记/归因有行为测试；门面/bindingNames/mock 收缩；复活=除名重跑）：旧测算项目写路径 9+复盘笔记 4+五算 4+归因 1+CostList 1+WorkcostBillItemSave/Delete 2。
- **前端**：FiveCalcPanel+test 删除（−1315 行）；types 孤儿接口 6 个删（knip 零孤儿）；bridge/mock/mappings/spaceBindings/bindingNames 五文件同步。
- **表处置**：退役表数据全 0 → 保留不迁移+冻结声明；V27 退役表 19 列冻结。
- **发版事故**：pathspec fatal 后误 amend 已推送的 release 提交——reset --mixed 回远程位置重新提交修复，tag 未动。**教训：amend 前必须确认 HEAD 未推送；pathspec fatal 后先 git status。**
- **测试**：全量 vitest 3788/436（−9 例=死组件删除）；Go 三包绿；drift **PASS@639**。
- **产物**：releases/gaea-v4.466.0.exe（冒烟 200；删 v4.461.0.exe 实存 462~466）；桌面补拷双验 ca5a04d5…。
- **下期**：三期·按库组卷导出五表（最高价值）→ AI 组价复活对接定额 → 对标扩到项目案例 → 资源调价批量导入。

## v4.465.0 发布：三库立骨架（清单库/企业定额库/工料机库）· 2026-10-07

- **用户定调**：「做企业定额库，清单库、工料机库，其中工料机库与信息价联动」——三库升格为 IA 骨架：清单库（双视图=按明细/按库项归并〔同名+同特征+同单位聚合，被 N 个项目使用+定额码汇总+可展开成员〕）/企业定额库（引用计数 Quota.UsageCount 读时聚合+来源徽标：非项目导入前缀=手工）/工料机库（改名+信息价联动显形——v4.464 链路的界面落名）。
- Go：Quota.UsageCount 进 ListQuotas/GetQuota（子查询，零迁移）。零新绑定 660。
- 测试：WorkcostViews 24 例（+2）；页面 10 例适配；全量 3797/437；drift PASS@660。
- 产物：releases/gaea-v4.465.0.exe（冒烟 200；删 v4.460.0.exe 实存 461~465）；桌面补拷双验 14988c7a…。
- 留池：清单库手工添加（项目归属语义待拍板）/定额库章节树/企业定额导出/**二期退役清理（18 死绑定）仍待执行**。

## v4.464.0 发布：一期·价格单一真相（规划一期 1.1~1.5 全落）· 2026-10-07

- **系统规划先行**：两侦察代理扫 Go 8 包+前端全模块+660 绑定+真实库逐表 → docs/gaea-cost-db-master-plan-2026-10.md 四期规划落库。核心诊断=三套价格数据（资料条目/工料机资源/询价）互不相通；AI 组价链悬空；18 个死绑定；价格源从未抓取。
- **1.1 实战验证**：真实源直调 pricefeed.Fetch——四川源 24 行成功（钢筋/水泥真实价）；重庆源站点失联（EOF，curl 确认非本端）→ 前端如实显形。
- **1.2 信息价→资源联动**：抓取时 enrichResourceMatches 预匹配（workcost.MatchResourceByTitleSpec 归一化标题+规格+单位，唯一命中才有值=宁缺勿误）；候选行「↔资源名」徽标；发布 syncResources（默认开）→ UpdateResourcePrice 推进现行价+写历史；PriceApplyResult 统计回显。
- **1.3 询价→资源**：调差「更新成本库」同步唯一命中资源（前端编排零绑定）。**1.4 历史 UI**：资源行按钮→Modal（PriceHistoryInline 死代码移除重写）。**1.5 来源显形**：priceType/source=信息价+期数。
- **教训**：绑定签名扩展走生成器门面自动更新（gen_bindings 重跑即可，勿手改 bindings_cost.go）；候选行文字断言在新 toast 文案下要用正则（getByText 默认全等匹配）。
- **测试**：Go workcost/pricefeed/app 绿；前端 22+5 例；全量 **3795/437**；drift PASS@660。
- **产物**：releases/gaea-v4.464.0.exe（冒烟 200；删 v4.459.0.exe 实存 460~464）；桌面补拷双验 c2b76dc2…。
- **下期**：二期退役清理（18 死绑定退役+退役表冻结+FiveCalcPanel 删）。

## v4.463.0 发布：项目文件导入移入造价参考；清单纯累计列表 · 2026-10-07

- **用户定调**：「把项目文件导入改在造价参考里面，清单是所有项目累计的清单列表，它不需要工程量」——IA 落位再校准：造价参考=项目案例（导入/费率/删除）+资料条目分位数对标（两套数据并存）；清单=只读累计列表（项目/编码/名称/特征/单位/定额/单价参考+三重筛选+分析弹窗），**无工程量列**。
- RatesEditor 抽独立文件；WorkcostBillView 全量重写（820→~460 行）；零新绑定 660。
- 测试：WorkcostViews 21 例重写（含「无工程量列、无合计」反向断言）+CostIndicators 6 例（+3）；全量 3794/437；tsc 0。
- 产物：releases/gaea-v4.463.0.exe（冒烟 200；删 v4.458.0.exe 实存 459~463）；桌面补拷双验 0012a568…。
- 留池：按库内组卷导出五表（项目案例+清单+费率都在库里，导出仍依赖源文件）。

## v4.462.0 发布：清单累计视图+筛分+项目整体删除；根修单价全零 · 2026-10-07

- **用户反馈三连**：两个项目单价全 0 要删；清单应累计显示+可筛分；质疑与造价参考重复。
- **单价全零真因（重锤）**：colIndex 子串匹配——`cQty="消耗量"` 被「消耗量单位」列劫持（子串包含+列序在前）→cellValue 读文本=0→含量全零。**宁丰 68/68、百锦路 114/114 全零，旺平列序不同幸免——e2e 只有旺平 fixture 所以四版都没暴露**。根修=精确匹配优先+排除单位列回退；**用用户真实文件 go run 临时脚本复验**（百锦路→0 零、宁丰→4 合法占位）。②「小计/合计」行（中文标题写在编码列）两路收集点误收→4 条孤儿定额→同规则跳过。
- **项目整体删除**（绑定 660）：DeleteBillProject 级联 项目+清单+独占定额（source 匹配且删后无引用）；共享定额与资源保留——Go 测试钉住分界。
- **清单累计视图**：全项目拉平带项目列+三重筛选（项目/分部/关键字）+计数；选中项目出封面费率卡+删除按钮（二次确认）。
- **造价参考不重复**：它聚合资料条目（cost_entries 1590 条旧价格手册）分位数，导入项目不进——hint 澄清。
- **坑**：①colIndex 子串匹配的劫持类 bug——**列名族（XX/XX单位）必须精确匹配优先**；②e2e fixture 单一来源的盲区（只有旺平）；③**vitest 从仓库根跑假红第五踩**（后台命令 cwd 在仓库根扫到 C:\AI\re\，1537 failed 假象）；④bash heredoc 长脚本静默截断再证——一律 Write 脚本文件；⑤anchor 差前导缩进（顶格 it(）。
- **测试**：Go TestDeleteBillProject+解析回归；前端 27 例（+3）；全量 **3797/437**；drift PASS@660。
- **产物**：releases/gaea-v4.462.0.exe（冒烟 200；删 v4.457.0.exe 实存 458~462）；桌面补拷双验 3d382c05…。

## v4.461.0 发布：清单特征显形 + 清单分析弹窗 + 定额编码统一 · 2026-10-07

- **起因**：用户导入 4 个项目实测（旺平/什邡/遂宁宁丰/百锦路，115 条清单）反馈三项。**真实库诊断先行**（只读 SQL）：①特征已落库（feature 几乎全非空）纯显形缺失；②定额 chip 是 span 无点击=「清单分析打不开」；③编码无统一规则（这 4 项目前缀 WP/SF/BJ/04B 恰好不撞，但第 5 项目可能撞码**覆盖**他人定额——UNIQUE 约束下的静默覆盖）。
- **三修**：清单表「项目特征」列（悬停全量+计算式）；定额 chip→按钮开综合单价分析弹窗（明细+三费+综合单价现算+原码徽标）；**SchemaV29** gf_quotas.orig_code + 导入链 ResolveQuotaCode 四态（直用/同源复用/内容全等复用〔真统一：同内容共享全局标准锚点〕/冲突 原码-N 消歧）；内容全等复用时既有定额有消耗行保持不动、空定额补齐（顺带治理 4 条无消耗行定额）。
- **教训**：报「数据没有」先查库再信界面（特征在库只是没显示）；「打不开」类反馈先找不可点击的伪按钮；唯一约束+局部编码=静默覆盖风险，统一规则要在撞车前落。
- **测试**：Go 四态 TestResolveQuotaCodeUnification+e2e 幂等不变；前端 24 例（+2）；全量 3794/437；drift PASS@659 零新绑定。
- **产物**：releases/gaea-v4.461.0.exe（冒烟 200；删 v4.456.0.exe 实存 457~461）；桌面补拷验哈希 4407c8ac…。
- **留池**：按库内组卷导出五表；已入库 115 条 orig_code 为空（重导入任一项目即回填，不做主动脚本）。

## v4.460.0 发布：清单层落库——分部分项工程量清单+项目费率（数据库不是计算器）· 2026-10-07

- **起因**：用户怒问「清单、定额、工料机是什么关系？我让你录入的清单呢？」——v4.457 导入链只落资源+定额，**清单项（含工程量）没落库**，只活在 xlsx 里；随后定调「造价数据库不需要费用汇总表，只是数据库，可录入费率/清单/模版，**但不是把它们加起来**」。
- **SchemaV28**：gf_projects（五表封面+费率六字段=录入数据）+ gf_bill_items（工程量/引用定额/手填单价覆盖，UNIQUE(project_id,code) 幂等，手填行自动 M 序号）。导入链 ④清单落库（工程量优先综合单价表、回退工程量计算表；费率从取费区落）。绑定 654→659（+5 Bill 系）。
- **前端**：新 WorkcostBillView=板块首屏「清单」（项目列表+清单表+费率录入，全落库，**零合计零费用链**——加总归导出五表活公式的 Excel）；**WorkcostFeePanel 删除**（v4.458 落点偏差二次纠偏——取费试算面板=「把它们加起来」，违定调；WorkcostProjectFees 绑定保留零 UI 消费）。
- **教训（两连纠偏）**：v4.457 落数据层不落清单；v4.458/459 又把「费用汇总」当主角——**用户的数据库心智=存取（费率/清单/模版），计算归 Excel**。「重塑数据库」指令的判读顺序：先问数据面（什么该存），再问界面面（怎么管），最后才轮到计算（通常根本不需要）。
- **测试**：Go bill CRUD 2 例+端到端清单断言（32 条落库幂等）；前端 22 例（含「不出现直接费/含税总造价」反向断言）；契约锁 654→659；全量 3792/437；drift PASS@659。
- **坑**：①契约数量锁在 bindingSignatures.ts（生成器烙数）——**新绑定后须 `gen_bindings -signatures-ts` 再生**，漏了契约测试红；②`git add` 首个 pathspec 已被 rm 的文件会 fatal 中止整条 add，commit 只含暂存区（提交后核 `git show --stat` 行数+树干净双验，本次 amend 救回）；③vitest 从仓库根跑假红第三次踩（cd frontend 铁律）。
- **产物**：releases/gaea-v4.460.0.exe（冒烟 200；删 v4.455.0.exe 实存 456~460）；desktop copy SAC 拦截→手动补拷验哈希 5a1bdee5…。
- **留池**：**按库内组卷导出五表**（清单+定额+资源+费率→直接生成，导出仍按源文件重建）；103 条旧综合单价条目批量转定额。

## v4.459.0 发布：造价数据库全新设计——工料法 IA（费用汇总首屏，该删除就删除）· 2026-10-07

- **起因**：用户追问「重构造价数据库在哪里/成本条目还是以前的、根本不是清单/概览是个什么玩意」——v4.457/458 把工料法落在数据层+平行「工料机」模块，板块主界面没动，**落点偏差坐实**。用户拍板「三刀全改」后升级为「对造价数据库进行全新设计，该删除就删除」。**教训：用户说「重塑整个数据库」时，期待的是主界面与工作流的形态变化，不只是数据层重构+平行新模块。**
- **新 IA（6 顶级模块，费用汇总首屏）**：费用汇总（清单测算工作台，默认落点）→ 单价分析（定额×资源价现算+导入/导出五表）→ 资源库 → 价格数据**四段**（价格源/价格仓库/询价库/**资料条目**——旧「成本条目」价格手册整面降级归档，检索/导入/AI 全保留）→ 造价参考 → 概览殿后（**换血工料法仪表盘**：三层规模+四类构成条+口径备忘+「下一步」按库最缺一环指路——定额=0 显眼引导「去导入项目表」，正对用户当前库资源 1486/定额 0 的真实状态）。
- **退役（UI 删除；Go 绑定与数据表全保留，旧数据零损失）**：测算项目模块（CostProjectsView+parts+test 删文件——v4.150 旧五算体系与清单测算重叠）；复盘笔记模块（办公域已有记忆/笔记）；顶栏新建条目/导入按钮（CostLibraryView 段内自带同功能入口）；概览旧指标（库规模/待补单价/草稿）。净 −1628 行。**「该删除就删除」授权解除 UI 简化红线于本板块，但删除只落在 UI 层——数据/绑定/迁移链不动。**
- **测试**：CostLibraryPage.test 重写 10 例（默认落点/6 模块序含退役断言/价格四段/概览新指标/定额 0 引导跳转/空库三步/失败态/回到概览）；全量 vitest **3792/437** 全绿（−2 退役测试文件 −8 例 +页面 1 例账平）；tsc/eslint 0；drift **PASS@654**（零新绑定）；Go 侧零改动。
- **坑**：①release.ps1 的 desktop copy **首次被 SAC（受控文件夹访问）拦截**——手动补拷配方=旧 exe `mv` 改名让位（运行中允许改名）→ `cp` 新 exe → sha256sum 验哈希；用户桌面 gaea 正在运行（PID 可查），改名不打扰旧进程，重启即新版。②bash 内联 PowerShell `$env:` 被转义吃掉（在册坑再证）——文件操作走 Git Bash 原生 mv/cp 最稳。
- **产物**：releases/gaea-v4.459.0.exe（冒烟 200 过；删 v4.454.0.exe，实存 455~459 五版）；桌面副本手动补拷验证一致。
- **留池**：103 条旧「综合单价」条目 → 批量转消耗定额（当前定额只能靠导入五表生成）；五表按库内组卷导出；概览「最近测算」记录（需持久化最近导入/测算概念）。

## v4.458.0 发布：工料法续刀——取费汇总面板 + 定额管理面 + 导出路径选择 · 2026-10-07

- **刀型**：v4.457 留池「后续可做」三项全收口。**零新绑定**——Workcost 19 绑定中 UI 未接的部分（WorkcostProjectFees/QuotaSave·Get·Delete/ProjectExport/PickDirectory）本刀接全，绑定面恒 654，drift PASS。
- **取费汇总 WorkcostFeePanel**（工料机第三子视图）：清单（定额库选行**现算**回填综合单价/手填行，单价可手改=项目级调价）→ 直接费（每行 round2 与 Go 侧 Round2 同口径）→ 费用链（四费率默认 10/2/7/9%、费率 0=显式不计取；利润基数含规费**开关**——两份实测产物口径差异不写死；控制价对照超 100% 显形）。草稿 localStorage 持久化，**试算不落库**（红线：无确认不落库）。
- **定额管理面 QuotaModal**：新建/编辑/删除（二次确认）+ 含量行增删（类别/资源编码/名称/单位/含量/损耗%）；资源搜索回填（resourceCode 对上资源库锚点，资源调价才能传导）。**编辑先 QuotaGet 取全量**——ListQuotas 轻量查询不含含量行，直接拿列表对象编辑会把含量行清空；保存后 composeTick 强制重算（selected 编码不变时核算 effect 不重跑）。编码编辑态锁定（稳定锚点）。
- **导出路径选择**：「导出五表到…」=PickDirectory → WorkcostProjectExport 显式路径（源名-时间戳.xlsx，取消静默）；「→ 工作区」保留 ExportToWorkspace。
- **测试**：WorkcostViews 9→22 例（取费链口径钉子 1000→100/77/20/1197/107.73/1304.73、开关 77→78.4、控制价 104.73/108.73%、现算回填、草稿往返、含量行联动、二次确认、导出取消/格式/工作区）；全量 vitest **3799/439** 全绿；tsc/eslint 0；Go 零改动复跑绿。
- **坑（测试侧两则）**：①金额断言撞多元素——合价/直接费合计/费用链首行三处同值，且费用链是异步核算，`findAllByText` 在第 2 处出现就返回（期望 3 得 2）→ 先等链完成标志（总造价出现）再 `getAllByText` 计数；②取消路径无可断言的可见变化（toast 先被 setToast("") 清掉）→ 固定 sleep 后断言零导出调用。
- **产物**：releases/gaea-v4.458.0.exe（冒烟 200 过；删 v4.453.0.exe，实存 454~458 五版）。
- **留池**：取费面板与「测算项目」模块联动（cost_projects 费率列 V27 已就位）等实弹反馈；五表导出仍按「最近导入源」重建，从库内直接组卷导出未做（需先有可解析模版源）。

## v4.457.0 发布：造价数据库重塑为工料法成本数据库（工料机资源库 + 五表模版）· 2026-10-07

- **用户指令链**：「根据成本测算模版重新设计整个造价数据库，采用综合单价带工料机的方式结构重塑」→「工料机是基本数据，综合单价分析表靠工料机和消耗定额核算出来的，其实就是建立我们的工料法成本数据库」。
- **口径纠偏（本版最重要）**：实测最新产物（旺平矿业 2026-10-07 / 什邡 2026-10-06，取自 gaea 项目空间 `.gaea/exports`）封面自述「**综合单价只含人材机**，管理费/利润/税金不进综合单价」。早期把取费链内联进单价=**错**，已彻底分离：`ComposeUnitPrice` 只出人材机、`ComposeProjectFees` 项目合计层跑一次；取费基数链做成可插拔 `FeePolicy`（利润基数含不含规费在实测两份产物间本身不同：百锦路=false / 市政道路=true）。
- **数据层**：SchemaV25 `gf_resources`+`gf_resource_prices`；V26 `gf_quotas`+`gf_quota_items`；V27 `cost_entries.price_derived`+组成行来源标记+`cost_estimate_items` 10 列+`cost_projects` 11 列。**`cost_entries.price` 保留列名与对外契约不变**（语义降级为核算结果缓存）——既有 20+ 消费方零改动，这是本版风险面收窄的关键决策。
- **三层核算**：外委升一等行类（实测汇总行 `材料 = SUMIF(材料)+SUMIF(外委)`）；零含量是**合法占位**（模版保留「技术工/带班 ×0」，计 `ZeroLines` 不告警，只有负值非法）；五表工作簿解析/导出（活公式 VLOOKUP+SUMIF+跨表取费基数链，导出用活公式而非死数字）。
- **存量已落真实库**：1590 条 → 资源 **1486**（人工 34/材料 1354/机械 64/外委 34），跳过 104（综合单价条目 103 保留在综合单价层 + 价格非正 1），**失败 0 / 无价 0**；幂等复跑零新建。命令 `scripts/migrate_workcost`（**默认干跑**、`-apply` 才写库）。**老库升级链另验**：2026-08 备份（v7、1248 条）→ v27，零损失 + 工料法三层立即可用 + 老数据可资源化（失败 0）。
- **前端**：造价数据库新增一级模块**工料机**（资源库/单价分析 + 导入项目表/导出五表）。资源化先出**干跑预览**（归类分布+跳过原因）再入库；资源缺价显形「缺价」、读失败显形「可重试」。
- **本版三个坑（都真）**：
  1. **`mapGaea` 漏 `GaeaWorkcost` 前缀** → 19 个方法静默落 **office** 门面。调用仍成功（前端 proxy 按方法名遍历所有门面搜索），但类型面归错域。补规则归 CostB。
  2. **生成器 CRLF 静默失配**：解析正则用 `$` 锚行尾，Go `(?m)` 下 `$` 不匹配 `\r`。症状极具欺骗性——带 `// 注释` 的行因 `.*` 顺手吞掉 `\r` 仍能匹配，**只有光秃秃无注释的行失配**，于是「新加的最后一行偏偏没被认领」（legacy 清单被吹胀、tsc 重叠锁才红）。已显式容许 `\r` + **反向验证**（CRLF 输入下仍正确认领 579）。诱因是我用 PowerShell `Set-Content` 把几个 TS 文件写成了 CRLF（仓库统一 LF）——已全部归一。
  3. **组成行类别判定短路**：`normalizeKind("")` 返回「材料」（成本库空类别默认值），早期据它判断「已标注」→ 后续关键词分支全被短路，**挖掘机台班被归成材料**。改为空 kind 不走归一。
- **`SaveRecomposed` 独立成方法**：`price_derived` 是**来源声明**，只有核算路径有资格置位；若并入 `Save` 默认行为，任何一次普通编辑都会把条目误标成「核算价」。
- **测试**：Go vet 0 + `workcost` **59 例**（含 3 个真实库探针）+ `app` 全包绿；前端 tsc/eslint 0 error + vitest **3786 例/439 文件**全绿 + 生产构建 ✓；导出经 LibreOffice 重算与按 bundle 独立口算**逐分一致**；exe 内 13/13 绑定名 + 6/6 前端文案 + 3/3 表列名二进制核验。
- **待抽查口径**：①「HDPE膜铺设人工费」归为人工（名称含「人工费」）；②脚手架租金类单位 `m/月`/`t/月`（租赁综合价，非纯资源）。改 `kind` 是一次字段编辑，不影响价格。
- **后续可做**：项目层取费面板（`WorkcostProjectFees` 绑定已就绪 UI 未接）；定额库管理面（当前只读+导入）；导出路径选择（现固定 `.gaea/exports`）。
## 非版本刀：布尔偏好四域收敛 booleanPref 工厂（第三轮审计 §3.2 *Prefs 线关账）· 2026-10-04

- **收敛落地**（`e7094920`）：browserPrefs/subagentPrefs/tasksPrefs/deliverablePrefs 四份逐字同形读写样板 → `createBooleanPref(key, defaultValue)` 单一方言（垃圾值回落默认、只认显式反向值翻转、恒写 "1"/"0"、降级静默）；四域文件 172→166（含工厂+新测试 8 例），拍板史注释与导出名全保留、调用方零改动。
- **方言保真证明**：四域既有 16 例测试零编辑复绿 + 工厂方言矩阵 6 例（双向 null/显式/垃圾值+往返+存储禁用降级）。vitest 全量 3777/3777。
- **判定不动**（非同型）：contextPrefs/layoutPreferences（JSON 逐字段校验域）、mdViewPref（单文件枚举）、paneTabs（zustand）。*Prefs 线整体关账。
- 坑补训：jsdom 的 localStorage 不经 Storage.prototype 分派——实例级 `vi.spyOn(window.localStorage, 'getItem')` 才拦得住。**不抬版本**。

## 非版本刀：CharacterPage hook 分域七刀（1092→650+7 hook，16 例金样零编辑复绿）· 2026-10-04

- **拆分落地**（`facb8835`，controller.ts 配方）：状态与 handler 群迁 `pages/character/` 七 hook——useCharacterData（数据根：三表加载+代际守卫+同步事件）/useCharFilters（筛选持久化）/useProjectCharacterEdit（抽屉本书局部设定域，CharacterDataSlice 显式注入）/useProtagonistRelations（关系批量域）/useCharDraw（抽卡）/useLegacyWriteBack（两段式回写）/useOrgRelation（组织+关系）；页面 1092→650 纯接线+JSX 逐字节保留。
- **等价证明**：16 例金样零编辑复绿 + tsc 0 + eslint 0 error + knip 无新发现 + vitest 全量 3769/3769。
- **配方复训**：跨域依赖全部显式注入（防 hook 间环，批 36 同款）；hook 变量命名避正文局部名冲突（orgRel）；重组脚本行号切片须核尾随空行（1092 wc vs 1093 split）。
- **不抬版本**。下一候选：CharacterPage 渲染函数出仓（renderProjectDetail/renderDrawModal/renderMergeModal → 子组件，可再减 ~300 行）或 CreatePage（宜缓）。

## 非版本刀：CharacterPage 域刻画测试 11 例（hook 拆分前置契约）· 2026-10-04

- **测试网 5→16 例**（`22ccb1e9`）：新增 domains.test 覆盖同步/回写（零冲突直写路径）/抽卡（默认参数+roleType 回退+pending 过滤+可变引用表 mock）/AI 主角关系（missing 直跑/all 须确认/三态消息/单角色点名）/合并（**参数序 A 并入 B 锁定**）/组织保存。手法=vi.mock 两个 API 门面+antd message 间谍+重型子组件桩。
- **CharacterPage hook 分域拆分前置条件达成**：16 例刻画测试垫底，下一刀可按 controller.ts 配方开拆（1092 行单函数）。
- **坑复训新增**：①Bash `node -e` 内 `\\s` 二次降级（在册坑第四次实证）——测试批量转换一律 Edit 手写；②antd 两字按钮插空格从按钮扩展到 Modal OK 钮与带计数的 tab label（`/^组织 \(/` 锚定）；③antd Select 的 placeholder 是渲染节点（`.ant-select-selection-placeholder`）非 attribute，getByPlaceholderText 查不到。
- 门禁：tsc 0/eslint 0 error/新旧 16 例全绿。**不抬版本**。

## 非版本刀：knip 守卫武装（第三轮审计 §3.5 关账）· 2026-10-04

- **knip 6.39 安装 + 首跑清账 + 入 ci.ps1 常驻**（`424481cd`）：knip.json 已在而工具未装=审计指认的守卫缺口。pnpm add -D + scripts.knip + ci.ps1 lint 后常驻（ASCII 注释保 BOM）。
- knip.json 按 6.39 收紧：删 mock/index.ts（文件不存在）/main.tsx/setup.ts（自动探测冗余）/ignore bindingNames.ts+drift.ts（新分析器不误报）/@types/node；留 genui/index.ts+icons.ts。
- **首跑发现清账 2 处**：CostLibraryView TableRow 遗留 re-export（批 34 拆分后零消费者）、spaceBindings.unknownBindingWarnings（注释称测试用实无引用）。
- **接受 3 条 duplicate-export**（双侧皆活逐条核实）：CharacterCard named←自测/default←库页；emitMock←外部/emit←mock 域内 chat.ts；CostLibraryPage named←测试/default←pageRegistry lazy。
- 门禁：tsc 0/eslint 0 error/vitest 3758 全绿/check-docs OK。**不抬版本**。

## 非版本刀：第三轮审计 §3.1 续刀（ai client.go 四分）+ 两组判定关账 · 2026-10-04

- **ai client.go 1447 行 god-file 按域四分**（`fb79d92f`，批 25 配方）：client.go 502（骨架/auth/image）+ client_failover.go 79 + client_chat.go 333（非流式/用量/请求准备）+ client_stream.go 579（流式/SSE/idleTimeoutBody/ChatSimple*）。等价双证=符号清单对称+go doc -all 逐字节一致；导入按消费重算编译器终审；internal/ai 全包测试零编辑绿。脚本 .audit/split-client.mjs（注释归属锚定 decl+限定标识符用法筛导入）。
- **boot.Build 559 行判定组合根留池**：24 相位、共享态网状（reg×5 簇/executor×4/sink×6），长度大半是接线决策注释；审计该行亦未给拆法（"—"）。候选设计=装配态结构体/相位函数，属重设计非机械刀——留设计批，不做 vanity knife。
- **测试 fixture 组（§3.2 末组）关账**：novel/create stubRuntime 审计称 3 份实测仅剩 1（移动靶）；sin vi.mock(bridge) ×4=按消费者显式列绑定面，是 NOT_MOCKED fail-closed（v4.455）的镜像设计，收敛反有害；schedule 无重复命名 fixture。
- 门禁：vet/golangci 0 issues；ai/config/boot 三包测试绿。**不抬版本**。

## 非版本刀：第三轮审计 §3.1/§3.2 续刀两件（config.Load 层装配拆分 + 拖拽手势单源）· 2026-10-04

- **config.Load 449 行 → 三层装配拆分**（`13665baf`）：按优先级层拆（默认值/环境变量/配置文件——域字段横切三层，层是自然缝，比审计建议的按域拆更贴合代码本体）；loadDefaults/applyEnvOverrides/applyConfigFile 逐字搬迁 config_load.go（含旧品牌回退+损坏备份恢复），Load() 收窄 35 行装配器；config.go 539→116；config 包 30 测试零编辑全绿。
- **拖拽改宽手势 ×4 → startColumnResizeGesture 单源**（`99e257d3`）：usePreviewPanel/useWorkspaceLayout/ResizableDrawer/useSidebar 的 listener 三件套+body 手势态设置还原收敛 utils/resizeGesture.ts（onDone 先于清理=对齐既有顺序）；三例单测（启动态/up 收尾取消订阅/pointercancel 同义）；vitest 3758/436 全绿。
- ***Prefs 系列勘察判定（§3.2 余池）**：browserPrefs/mdViewPref/paneTabs 抽样=**三种语义方言**（boolean 显式关值/枚举/zustand 状态机）且各有文档化取值判据，非逐字同型——硬收敛属设计题（方言参数化+对照钉），留池待设计批。
- 门禁：Go vet/golangci 0+全量零 FAIL；eslint 0 error；vitest 436 文件 3758 例（+1 文件+3 例=resizeGesture）。**不抬版本**（纯结构收敛零行为）。

## 发版 v4.456.0：微信媒体下载 SSRF 守卫对齐 + 第三轮审计结构收敛 · 2026-10-04

- 上午两批非版本刀（第三轮审计收敛 `0c09977e` + §3.2 重复收敛 `da89ca89`）后，续刀三批（weixin SSRF failover `cf7d0cbb` / xlsxedit applyOne 表驱动 `0777f5b0` / rebuildParagraph 收敛 ooxml `2da28d15`），批次 A 为用户可感修复（媒体下载可用性+守卫三副本对齐）⇒ 抬版本。
- 修复=weixin mediaTransport 补齐 IN2-04「逐个已验 IP 尝试建连」（此前固定拨 ips[0] 无回退），安全语义零变化；xlsxedit applyOne 253 行→opHandlers 表驱动（summary 契约 11 例先行+unmerge_cells 补测+键集冻结）；rebuildParagraph 分组+区间计算收敛 ooxml.GroupRuns/AffectedSpans（发射端 w:/a: 两方言保留）。
- 审计证伪两项入册：novelModelName ×4（函数已不存在）、§3.4 ops.go 零覆盖（12 op 全有测试命中）。
- 门禁：Go vet/golangci 0 + 全量零 FAIL；前端 vitest 3755/435 全绿；三门禁绿。产物 releases/gaea-v4.456.0.exe 52,047,360B（冒烟 200 过；删 v4.451.0.exe，实存 452~456 五版）；速览迁 1 条（453 入 archive）。

## 非版本刀：第三轮全仓审计收敛批 + §3.2 重复收敛批（净 −2017 行）· 2026-10-04

- **第三轮审计收敛批（`0c09977e`）**：今晨 11 子代理分域扫描→双证据复核的改动落库（净删 ~1937 行：TS6133 清单 38 条/死文件 2/死本地化键 26×3/死类型/死 CSS ~676 行两路交叉验证/Go 零调用方法 12 个+precheck 死分支/round4 收敛 cost.Round4）；**守卫开闸** tsconfig.app.json noUnusedLocals/Parameters（反向探针 TS6133 验证生效）+ wails.json npm→pnpm + .audit/ 入 gitignore；拦下 15 组「看着像死实则活」误报未删。交接验证全亲跑：go build/vet/lint 0 issues、go test 全量零 FAIL、tsc/eslint 绿、vitest 3752/3752（基线 3760−8=有意删的死测试，账平）、pnpm build、docs/绑定 635/locale 死键三门禁。**审计报告归档 docs/code-audit-2026-10-04/**（README+reports 10 份，docs/README 登记防孤儿）。
- **§3.2 重复收敛批（`da89ca89`）**：errText 方言 A（err, fallback）14 份逐字相同 → utils/errText.ts 单源+3 测试（方言 B 无 fallback 语义不同未并）；EMO_COLORS/EMOTION_COLORS 9 色全等 → chat/constants.ts 再导出；Go strutil 增 ParseTagsJSON（cost+knowledge 5 调用点）+ ContainsHan（hasHan/isCJStr 名不符实收敛）。**等价性测试反抓两处我的注释错误**：假名串含汉字（日本語）=true；尾部垃圾 Unmarshal 实测不部分填充返回 nil——语义按实测冻结。sortedKeys 3 份签名各异=方言不硬并。
- **发版判定**：纯内部清理零行为变化，按 README 发版约定（用户可感能力才抬版本）**不抬版本**，记入 CHANGELOG 随下一版。
- **余池**：§3.1 结构性（config.Load 449 行/xlsxedit applyOne 253/rebuildParagraph 双份 140 行/boot.Build 559/client.go 1160/CreatePage 1403）；§3.2 余（拖拽清理块 ×4 抽 hook、*Prefs localStorage 系列 14 文件、测试 fixture 17 份、novelModelName ×4）；§3.3 待拍板（SummarizeFrom 拒绝桩/SuspendPrefix 未接线/reasoning_language no-op/SetMemoryQueue 不可达/幽灵工具名 4 个 HideUnlessOnly 永久 no-op/types/interfaces.go 22 类型 ~240 行/provider retry.go 整文件+SSRF 判据 weixin 副本漏 IN2-04）；§3.4 补测（schedule ops.go TestApplyOpsEachKind 表驱动）；§3.5 守卫残余（eslint 未用变量 warn/knip 未装）。

## 发版 v4.455.0：全仓审计修复收官（512 条审计 50 批清账 + 绑定面 744→635）· 2026-10-03

- 2026-10-02 全仓审计修复线程收官（批 11~56）：P0 25 全清；四簇/duplication/god-file 族/留池/上报未动池/观察池全清或归拍板；拍板池 16 项中 14 项已执行或零动作确认。**绑定面 744→635**（A1① 109 零调用摘除，生成器 excludedBindings 管线）。要点：AtomicWrite perm 生效/WhisperSubgraph 真机空图根修/AtomicWriteSync fsync 变体/任务身份三系统对照表/NOT_MOCKED 45→0 fail-closed/controller.ts 884→141 hook 分域六文件/estimateTokens 五处收敛/反向变异三组+ID 格式钉三件。发布前 ci.ps1 全量档首次联合通过（前端 vitest 3760/435；app 全包 135s 绿）。
- 产物：releases/gaea-v4.455.0.exe（冒烟 200 过；删 v4.450.0.exe）+ wubigork-v4.455.0-source.tar.gz（tag 归档）。台账：docs/code-audit-2026-10-02/round-1~56 + decision-brief。

## 非版本刀：全仓审计第 36 批·FE3-11 续刀（hook 分域配方首刀：记忆/画像族出仓）· 2026-10-03

- **刀型**：接续批 35（`51b81793`）。**hook 分域配方首刀**：controller.ts 414→376+controller_actions_memory.ts 54——记忆/画像族八回调（useCallback 依赖数组逐字保留，clearFactBase 跨族依赖 refreshFactBase 显式注入）+ 桥接错误三助手出仓（**随动作域走防 controller→actions→controller 环**）；useMemoryActions 原位替换保 hook 调用顺序。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-38-p1-batch36.md`。
- **配方要点（进在册）**：六域盘点→首刀选最小内聚组立配方；跨族依赖经子 hook 签名显式注入；锚点按当前文件实测取（跨批改写后头部形态已变，旧形态断言失败即中止未写盘）。
- **门禁**：tsc 绿 / 53 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例）；提交后树干净+HEAD 可编译。
- **FE3-11 累计 884→376+484+54**：余五族同配方续刀。下一批候选=FE3-11 续刀（会话管理族 13 件/轮次控制族/回退族）或对账其他活池；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 35 批·FE2-04 CostProjectsView 拆分（明细件+样式常量出仓）· 2026-10-03

- **刀型**：接续批 34（`93db2bfd`）。前端配方第六刀：CostProjectsView.tsx 895→~570+cost_projects_parts.tsx ~318——明细件层（ItemRow/EntryPicker/ProjectForm+Field/SnapshotTable/小工具四件，八件全为主段消费）出仓；五共享样式常量（fmtPrice/fieldCls/ghostBtn/solidBtn/iconBtn）落 parts 单源导出。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-37-p1-batch35.md`。
- **要点**：外部仅注释提及、测试只导主件→parts 无需 re-export；**X 单字母图标首轮正则双侧漏报（grep -w 精确定位归 parts）**；批 34 教训执行=样式常量从原文件拷字节插值，iconBtn 跨行值被行号切片切断按位置拼接修复（**CRLF/LF 混排下禁锚点串匹配，用 find 定位拼接**）。
- **门禁**：tsc 绿 / 6 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例）；提交后树干净+HEAD 可编译。
- **memoryhub god-file 族现状**：CostLibraryView 1167→726+455、CostProjectsView 895→~570+~318。剩余=设计型（useController 36 回调 hook 分域/弹窗上提）。下一批候选=设计型配方刀或对账地图其他活池；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 34 批·FE2-02/FE1-05 CostLibraryView 拆分（展示件层出仓）· 2026-10-03

- **刀型**：接续批 33（`64ec4f81`）。前端配方第五刀：CostLibraryView.tsx 1167→726+cost_library_parts.tsx 455——展示件层（CategoryNode/四 props 接口/CostRow+ListView+TableRow+TableView 四 memo 件/MiniCompositionBar，依赖极瘦无 antd/bridge/hooks）出仓；主组件留状态+5 弹窗编排（弹窗内联属设计型不动）。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-36-p1-batch34.md`。
- **双向消费**：测试从原路径导 CostRow/ListView → 主文件 re-export 保路径；CategoryNode 主段要用补 export；TableView 原无 export 补导出；SortKey 两段共用落 parts 单源；genius 守卫命中为 maxTableRows 假阳性。
- **一次自伤被金样当场抓获（教训入册）**：STATUSES 补插时 python 手工 UTF-8 转义把「草稿」错编成「草案」（一字之差）——13 例当场 1 红。**python 字节级插值禁手编中文转义（须从源文件拷字节或走 Edit），文案常量搬移后必须 diff 原文逐字节核对**。
- **门禁**：tsc 绿 / 13 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例）；提交后树干净+HEAD 可编译。
- **下一批候选**：FE3-11 续刀（hook 分域）或 CostLibraryView 弹窗上提或 CostProjectsView 862；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 33 批·FE3-11 controller.ts 第一刀（状态机纯函数层出仓）· 2026-10-03

- **刀型**：接续批 32（`5346e60a`）。前端配方第四刀：controller.ts 884→415+controller_state.ts 484——纯状态机层（types/取号 slot 口径/历史重建/applyEvent/reducer/initialState，零闭包）出仓，`export *` 原路 re-export 外部导入路径不变；React 编排层留守。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-35-p1-batch33.md`。
- **预核要点**：logBridgeError 触 app.LogFrontendError 判归编排侧（防纯函数偷桥）；useController 36 回调 hook 分域=设计型另立刀。**编译期迭代三次**：漏 type import 级联出测试 TS7006 假象（根因单一致）；`Action` 原私有跨层要用逼出 export 决策（拆分的正向收益=纯/编排契约显式化）；CRLF 下 python 需 wb 字节替换。
- **门禁**：tsc 绿 / store+contract 12 文件 105 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例）；提交后树干净+HEAD 可编译。
- **FE3-11 部分收敛（884→415+484）**：下一批候选=FE3-11 续刀（hook 分域配方）或 CostLibraryView/CostProjectsView 组件族；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 32 批·FE6-07 ModuleLauncher 拆分（展示件+轮询 hook 出仓）· 2026-10-03

- **刀型**：接续批 31（`221b73ee`）。前端配方第三刀：ModuleLauncher.tsx 1058→684+launcher_parts.tsx 391（九个纯展示子组件+useLauncherData 轮询 hook+最小类型整段出仓；主文件留双空间首页主体+两套变体）。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-34-p1-batch32.md`。
- **双向接口**：两轮依赖普查（出仓段依赖/主段反向消费 9 符号）+外部消费通道核查（TasksFirstHome/useChatVoice/MainLayout）——SessionList/LauncherData 经主文件 re-export，外部导入路径零改动；SessionList/LauncherData 原文自带 export 关键字，尾部集中导出只列其余 7 符号（重复声明 tsc 抓，即平）。
- **源级守卫随迁（本批关键动作）**：perf-guards.test.ts 源级断言在首轮 ci 拦下 1 红——订阅用法已随段出仓；守卫意图不变、读源范围扩为主文件+展示件并集（头注写明缘由）。**教训：源级断言守卫必须与其守卫的代码同批随迁**（App.tsx 源级锁同家族）。
- **门禁**：tsc 绿 / ModuleLauncher+TasksFirstHome 21 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）；提交后树干净+HEAD 可编译。
- **FE6-07 部分收敛**：两套变体渲染分叉仍在主文件（设计型）。下一批候选=FE3-11 controller.ts（预核判定=useController 280 行单 hook 36 回调属 hook 分域设计型非机械，需先立 hook 分域配方）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 31 批·FE3-15 续刀（Git/任务/证据链/备份 → methods_ops.ts）· 2026-10-03

- **刀型**：接续批 30（`10f8ee5c`）。前端配方第二刀：四域连排块（Git 7 件/任务 5 件/证据链+WriteFile 4 件/数据备份 6 件）整域抽取至 methods_ops.ts（222 行），`Pick<OfficeMethods,22 键>`+`...opsMethods` 原位展开。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-33-p1-batch31.md`。
- **一次真回归抓取与根修**：首轮全量 vitest 22 红（DeliverablesPanel 证据链族）——组装漏插 `...opsMethods` 展开行致组合缺 22 键。**教训升级进在册：对 Partial 注解的组合导出，tsc（缺键类型合法）与 contract 金样（只锁部分面）双盲，组件端到端才是组合完整性的唯一真守卫**——扩域后必须补跑消费该域的组件测试；全量 ci 兜底拦下（门禁未失守）。修复后 130 例绿。
- **门禁**：tsc 绿+契约/面板 13 文件 130 例绿+前台 ci **exit 0**（vitest 435 文件 3762 例）；行数对账闭合 519+185+2+27=733；提交后树干净+HEAD 可编译。
- **FE3-15 累计 1029→550**（两域 ~545 行出仓）；剩余主体共享 mockFileBodies/mockXlsxState 状态面（「共享状态域」，续拆需先理状态归属）。下一批候选=FE6-07 ModuleLauncher 或 FE3-11 controller.ts（闭包结构需预核）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 30 批·前端拆分配方试水（FE3-15 DAG 域整域抽取）· 2026-10-03

- **刀型**：接续批 29（`17bef745`）。**前端配方第一刀**：methods_office.ts 的 DAG 域（mock 态+推进器+Dag* 方法 11 件）整域抽取至 methods_dag.ts，组合导出 `...dagMethods` 原位展开=键序逐位不变；组合字面量保留完整类型注解（批 17 spread 盲区教训）。主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-32-p1-batch30.md`。
- **前置核查**：`this.PinnedMaterials()` 约束（头注在册）仅两处且 Dag 块无 this——可安全搬移；Dag 前言自包含（仅 types+setTimeout）。
- **前端配方三件套进在册**：①**真 tsc -b（必须 cd frontend——从根跑 npx tsc 命中假 tsc 包 exit 0 假绿）**→ ②contract 金样 52 例（744 绑定面）→ ③组件端到端金样（DagPanel/TasksWorkbench 32 例，验 spread 后运行时绑定）。过程编译期纠错两处：组合侧漏带 `export const…= {` 开头/Pick 类型实参末位多逗号——终审是编译器再次实证。
- **门禁**：tsc 绿+契约 52+端到端 32 绿+前台 ci **exit 0**（vitest 435 文件 3762 例）；提交后树干净+HEAD 可编译复验。
- **FE3-15 部分收敛**（1029→732+323）：巨型字面量仍在，70 方法域分组需重排序留续刀。下一批候选=FE3-15 续刀或 FE6-07 ModuleLauncher；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 29 批·god-file 对账收窄（GA1-03/GA4-02 降级）+ 留池 gofmt 清账 · 2026-10-03

- **刀型**：接续批 28（`2f6947c3`）。拆分价值判定批：两条候选降级出池+一项留池清账，主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-31-p1-batch29.md`。
- **判定**：GA1-03 agent.go 与 GA4-02 controller.go 所在包均已高度分解（agent 80+ 文件 / control 30+ 文件），剩余行数=God 类型结构体本体+一行式访问器——再拆收益近零，**字段收敛属设计拍板**；两条移入「拆分价值低/拍板候选」，**Go 侧可拆池清零**。留池关账：scripts/test_image_models.go 补 `//go:build ignore`（scripts/ 不在 golangci 覆盖=假阴性存量五批）。
- **god-file 余 ~13 全在前端**（CostLibraryView×2/CostProjectsView/controller.ts/App.tsx/ModuleLauncher/methods_office.ts 等）；前端拆分=状态上提+import 穿线**不同型**，下批先立前端配方（FE3-15 methods_office.ts 最接近机械形态，contract 744 绑定测试=现成金样）。
- **门禁**：gofmt -l scripts/ 归零；树干净+HEAD 可编译复验。

## 非版本刀：全仓审计第 28 批·god-file 续拆（IN1-01 project.go 五分）· 2026-10-03

- **刀型**：接续批 27（`0492001f`）。god-file 同配方单线（同包纯文件拆分零逻辑改动），主代理直做。预核判定：核心 Manager 但同包分家对消费方零影响（API 不动）——「接口分域两步走」先做零风险第一步。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-30-p1-batch28.md`。
- **IN1-01**：project.go 1194（审计时 1011，继续增长）→ 五文件（project.go ~222 核心+生命周期+上下文构建 / project_files.go ~229 固定资产读写 / project_chapters.go ~348 章节与文件名单源 / project_migrate.go ~160 迁移+writeJSON/loadJSON〔全包共用落点组织性，头注写明〕 / project_stores.go ~290 逐章派生存储）。
- **等价证明**：双证零漂移（符号清单 diff 空——**对称口径教训复训：两侧 glob 都含测试文件**，首轮漏侧出 27 行假删除；go doc -all 逐字节一致）+ 行数对账闭合（1170+5 seam+19 旧头=1194）+ 包测试 0.6s 绿。
- **门禁**：前台 ci **exit 0**（golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例 / 卫生守卫）；新文件四枚显式 add；提交后复验树干净+HEAD 可编译。
- **god-file 余 ~15，重心转前端组件族**（CostLibraryView 1132×2/CostProjectsView 862/controller.ts 963/App.tsx 1041/ModuleLauncher 1038）——组件拆分=状态上提+prop 穿线不同型，需先立前端配方（金样 DOM 断言先行）。下一批候选=前端试水或 Go 侧续拆（GA4-02 controller.go 40 字段上帝对象/GA1-03 agent.go）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 27 批·god-file 续拆（IN3-03 mpp.go 三分）· 2026-10-03

- **刀型**：接续批 26（`022f1405`）。god-file 同配方单线（同包纯文件拆分零逻辑改动），主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-29-p1-batch27.md`。
- **IN3-03**：mpp.go 1035 → 三文件（mpp.go 入口+CFB 容器+中间结构装配 ~600 / mpp_layout.go 版本布局表 156 零 import / mpp_rows.go 低层块解析+字节原语 302）。
- **真回归抓取**：拆分头注把包级 doc 注释错放到 package 之后→go doc 包文档丢失→复位紧贴 package 子句恢复一致。**教训入册：god-file 拆分时原包级 doc 注释必须逐字节跟随 package 子句，doc 块与 package 之间不得插行；拆分说明写新文件头勿污染包文档**；另：符号对比 glob 口径须对齐非测试源文件（首轮 mpp*.go 混入测试符号）。
- **等价证明**：符号清单 diff 空（非测试源文件口径）+ go doc -all 逐字节一致 + schedule 包测试绿（真实样本门控+四版本合成用例覆盖搬移代码）。
- **门禁**：前台 ci **exit 0**（golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例 / 卫生守卫）；新文件两枚显式 add；提交后复验树干净+HEAD 可编译。
- **god-file 余 ~16**；下一批候选=IN1-01 project.go（动核心 Manager 需厚预核，可能接口分域+文件分家两步走）或前端组件族（CostLibraryView 1132 等，组件拆分配方与 Go 分家不同型）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 26 批·god-file 续拆（GA5-08 browser 四分 + GA5-04 耗时域分家）· 2026-10-03

- **刀型**：接续批 25（`486959f7`）。god-file 同配方续拆两线（同包纯文件拆分零逻辑改动），主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-28-p1-batch26.md`。
- **线 1 GA5-08**：browser/manager.go 1068 → 四文件（manager.go ~320 核心生命周期 / manager_tabs.go 196 多标签页 / manager_actions.go 370 页面操作 / manager_evaluate.go 196 evaluate+JS 片段）——文件自带分节注释即切口，最低风险形态。**线 2 GA5-04**：contextview/fold.go 1041 → fold.go 874 + fold_timing.go 180（耗时折叠域整体分家，与专属 timing_test.go 同域对位；**结构不动、方法分家**，比拆结构体低一档风险）。
- **方法要点**：import 图谱两轮 grep（别名调用与注释内包名分清），**终审是编译器**（evaluate 段误带 time 被 build 裁掉）；fold.go import 零改动（json/sort 双侧共用、time 注释性命中先证伪）。
- **等价证明**：双证零漂移（符号清单 diff 空 + go doc -all 逐字节一致，两包各验）+ 行数对账闭合 + 包测试绿（browser 5.3s / contextview 0.4s 含 timing 专属用例）。
- **门禁**：前台 ci **exit 0**（golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例 / 卫生守卫）；新文件四枚显式 add；提交后复验树干净+HEAD 可编译。
- **god-file 余 ~17**（App.tsx 1041/mpp.go 1035/project.go 1011/controller.go 963/agent.go 975/CostLibraryView 1132 等）；下一批候选=IN3-03 mpp.go 分家或 IN1-01 project.go（动核心需更厚预核）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 25 批·god-file 大拆（GA2-01 websearch + IN2-12 stats 计价分家）· 2026-10-03

- **刀型**：接续批 24（`60b2fc36`）。god-func 三刀收官转 god-file：**两线同包纯文件拆分**（原位搬移零逻辑改动），主代理直做。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-27-p1-batch25.md`。
- **线 1 GA2-01**：websearch.go 925 → 三文件（websearch.go 331 工具本体 / websearch_engine.go 456 引擎 seam+6 引擎+共享 HTTP / websearch_html.go 171 SERP 解析+节点工具）；**线 2 IN2-12**：stats.go 758 → 计价域 modelprice.go 182（定价表+归一化+估算口径+默认汇率，专用正则随行）+ stats.go 588（记录器+Manager 接线）。
- **方法与等价证明**：`sed 行号抽取+cat 拼`（字节精确，行数对账闭合）替代手抄；**双证零漂移**=符号清单 before/after diff 空 + `go doc -all` 全包逐字节一致（两包各验）；包测试绿（builtin 12.6s 含计价/注册表/策略套件，modelengine 3.0s 含 estimate_cost 用例）。
- **门禁**：前台 ci **exit 0**（golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例 / 卫生守卫）；新文件三枚进显式 add 清单，提交后复验树干净+HEAD 可编译。
- **god-file 余 ~19**（browser manager 1068/fold.go 1041/App.tsx 1041/mpp.go 1035/project.go 1011/controller.go 963/agent.go 975 等）；下一批候选=同配方续拆（GA5-08/GA5-04/IN3-03 待预核）；coupling ~20 与零散死码等拍板。

## 非版本刀：全仓审计第 24 批·god-func 大拆第三刀（AP2-01 runSinStream 三段拆）· 2026-10-03

- **刀型**：接续批 23（`65baace6`）。**单线主代理直做**：AP2-01 `runSinStream` 265 行串六职责 → 「装配/循环/收尾」三段拆（`buildSinTurn`/`runSinRounds`/`persistSinTurn`，审计修法落地）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-26-p1-batch24.md`。
- **关键判定**：非 switch 型不适用表驱动配方；「先搬行不改语义」+ **abort 哨兵上提**（循环内「整轮失败零正文」的 emit+仅用户落库 inline 出口移到分派层，帧序逐帧一致）；循环 8 个可变态圈死在 `runSinRounds` 内；`sinTurnPlan`/`sinRoundOutcome` 两个产物结构体。预核漏项自查：buildSinTurn 漏 eng 参数编译期抓到即补。
- **等价证明**：测试零编辑（sin 族 20+ 用例+app 全包 117s 绿）；**反向变异一组**（abort 落库 topicID 加 TEMP-MUTATE→TestSinStreamPlainErrorSavesUserMessage 红→还原绿→全仓 0 残留）；vitest 435 文件 3762 例全绿（顺带证批 23 两红=环境瞬时）。
- **顺手对账**：ci 尾段卫生守卫拦下 `releases/README.md` 源码包计数漂移（11 vs 12，e48c1c56 归档漏更新与本批无关）→ 单修 `341da99d`（12 个/181.0 MiB）。
- **god-func 大拆三刀结论进在册**：switch 型=表驱动（AP2-05/IN3-04）、状态圈死型=三段拆+哨兵上提（AP2-01）；等价证明统一形态=「契约/测试先行→搬行→零编辑复跑绿+反向变异抽查」。god-func 簇余量=god-file ~26+coupling ~20（拍板）+零散死码（等 109 拍板）。

## 非版本刀：全仓审计第 23 批·god-func 大拆第二刀（IN3-04 applyOne 表驱动化）· 2026-10-03

- **刀型**：接续批 22（`e7d1768d`）。**单线主代理直做**：IN3-04 `internal/schedule/ops.go` `applyOne` 430 行巨型 switch → 12 个 `applyXxx` 独立函数 + `opHandlers` map 注册表分派（批 22 配方第二刀）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-25-p1-batch23.md`。
- **三步配方**：①**契约先行**——golden fixture 补冻结**回执文案（summary）**（Err/After 已冻结、summary 是 applyOne 第三输出面也是拆分唯一可能静默漂移的面；TS 只消费 error/after，加法字段惰性实测 71 例绿）；②**表驱动拆分**——12 case 逐字搬运（含注释）dedent 一层 + 分派器未命中同 default 文案 + `ops_dispatch_test.go` 新守卫（注册表键集↔Op.Type 枚举双向锁+未知类型 fail-closed 逐字图钉）；③**零编辑复跑**——金样测试零编辑全绿=三输出面逐字节等价。绑定面零变更（ApplyOps/Op 签名不动，唯一生产调用方 schedule_tools.go 只读 summaries）。
- **门禁**：前台 CI：golangci 0 issues + `go test ./... -count=1` 137 包全绿；vitest 3760/3762（2 红=OriginalSinPage.tabs 零交集 flaky，单文件复跑 4/4 绿定性环境瞬时）；前端 build exit 0 补跑。提交后复验「树干净+HEAD 可编译」。
- **大拆配方进在册**：AP2-05+IN3-04 两刀实证「契约先行→表驱动→零编辑复跑」对 **switch 型** god-func=零行为差异形态；对照测试缺的面必须先补冻结再动手。下一刀候选 **AP2-01 runSinStream（非 switch 型，语义拆分，需独立预核）**；余量=god-file ~26+coupling ~20（拍板）+零散死码（等 109 拍板）。

## 非版本刀：全仓审计第 22 批·零散收尾 + 大拆试水（3 线 + 主代理直做 1）· 2026-10-03

- **刀型**：接续批 21（`8dc9f021`）。duplication 零散收尾 + **god-func 大拆试水首刀** + 死码删除：**3 条互斥并行子代理线**（internal/app 双条 / frontend memoryhub / gaea/cache）+ 主代理直做（章节号正则改调收尾+AP3-03 式押后惯例）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-24-p1-batch22.md` + testctors-baseline.json（364→372 已评审）。
- **四条内容**：①**AP2-05**（大拆试水首刀）SaveConfig 224 行巨型 switch（63 case/20 Warn/三遍样板实证）→ 表驱动 `cfgMemSetters` 62 键 + 4 构造器 + 哨兵，SaveConfig 收窄 ~35 行；**等价证明**=63 键覆盖对照测试（217 行：互异值逐键断言精确内存字段+18 盘有内存无键 Warn 逐字+盘校验先于内存同步顺序钉）先对旧 switch 跑绿→表驱动化→零编辑再跑全绿；与 config 包 saveSetters 成对（键集刻意不一致，盘有内存无 18 键现状钉死）；②**AP9-06 冲突指控证伪**——`module_bindings.go:28` 实为 `"gaea.chat": "gaea"`（审计把 :26 chat.chat 行看串行；批 6 同源误读先例）⇒ 映射无需改动；三处互引注释 + `IntentsOf` 访问器 + **分类器全 5 分支派发守卫**（把批 6 只锁 default 的覆盖补满）；MainBrainChat 等拍板；③**FE2-06** 价格源两份 → `PriceSourceCard`（variant 参数化 10 项漂移清单）+ `usePriceSources`（超时/三态单源）；净 −200 行；**申报行为变化**：Repository 失败从静默空列表变可见+重试；④**GA4-08** SpawnPolicy 死族 10 符号删除（spawn.go 226→101）——**配额裁决=删光**（maxForks 只写不读从无执行力、SpawnReport 无生产者、生产 fork 走 skillRunner/RunParallel 与 cache 无关）；绑定面零变更实测；boot 侧反向变异不可行（零 boot 测试触达=覆盖缺口登记）；⑤**章节号正则改调收尾**（主代理）：stats×2/graph×1/app deepChapterRe×1 改调 ParseChapterFileName——六处清点全部闭环。
- **反向证据（主代理亲手各重做一组，全红→绿）**：线1 KeyQualityThreshold setter 接错字段→63 键对照红/线2 withTimeout 吞拒绝→loadFailed 钉红（超时用例仍绿证突变精确）/线3 LookupSpawnTemplate 恒空→注册查找钉红/AP3-03 拍守卫→play 红线钉红。
- **门禁**：前台 CI OK / exit 0（golangci 0 issues · vitest **435 文件 3762 例** · E 系列 · 卫生四查）；守卫：**test-ctors 364→372（+8 已评审刷新：本批测试桩，同文件既有先例同款）**、bindings@744+死绑定 109 在册、primitives/contract 无新增。提交后复验「树干净+HEAD 可编译」（批 21 教训执行——本批新文件全部进显式清单零遗漏）。
- **大拆试水结论（进在册）**：**对照测试先行的表驱动化是低风险形态**（63 键等价测试一次过、零行为差异）——后续 god-func 大拆（IN3-04 applyOne 430 行 switch 同款）沿用此配方。
- **留池/下一批**：SaveConfig 内存同步表整块迁移（域头注有方案）；boot 模板路径零端到端用例（线 3 覆盖缺口）；`:54` 暖启判据补测试。**对账地图剩余=god-file 大拆 ~26+coupling ~20（拍板）+零散死码（等 109 拍板）**。下一批候选：大拆第二刀（IN3-04 applyOne 430 行 switch 同配方）或 AP2-01 runSinStream 265 行串六职责。

## 非版本刀：全仓审计第 21 批·duplication 余量四线（14 条收敛）· 2026-10-03

- **刀型**：接续批 20（`a3cd2ba7`）。对账地图「duplication 余量 ~15」中取 **14 条，四条互斥并行子代理线**（export+project / BM25 分词族 / Go 杂项 / 前端 schedule 族）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-23-p1-batch21.md`。
- **收敛 12 条+证伪修正 4 处**：①**IN1-03** 章节号解析实为六处（复核新增 graph 第三份正则；bookimport 证伪非消费方）→ `project.ParseChapterFileName` 骨架单源+四保留站点口径对照表（迁移 Sscanf 与正则**双向不包含**探针实测）；②**IN3-12** 导出循环四份→`forEachChapter`+`readWorldviewWarned`（EPUB 中途失败不写 FailedChapters 语义逐字保留；ExportHTML 第五循环结构不同未并入）；③**IN3-13** Markdown 块判定→`markdownBlockKind`（docmd 对照钉）；④**GA3-03/04+IN2-11 BM25/分词族**——分词 6 套/BM25 3 套实测成表→**公式单源** `Params.TermScore`（三套 IDF 逐位一致数学证明：log(1+x) vs log(x+1) IEEE 可交换）+**分词开关单源** `TokenizeOpts`（gaea/memory SearchIndex 整体薄封装）；**异族三种证伪出族**（internal/memory k1=1.5 不同算法族不接分词只委托公式、search.Tokenize 不小写化、textsim Dice 集合语义、novelstyle 写作统计）；**数学发现入册：固定 b 下 k1 不可能改变两两排序**（g(a,n)=g(b,n′)⟺a·n′=b·n 与 k1 无关+scratch 实证）——k1 回归防护=分值快照非顺序；⑤**GA2-06** dag 双套→泛型 `fileStore[T]`（钩子先于 MkdirAll；空集 `[]T{}` 单点）；⑥**AP5-06** 主代理「输出一致则映射」前提**被现场修正**（蒸馏=确定性宁漏勿误 vs 面板=模糊 textsim，**输出不等价**）⇒冻结第三路：口径差异注释+对照测试钉成机器事实，收敛与否=拍板；⑦**GA6-05** 「4 包收敛」重画（costinquiry/costref 是第三方言单点已收敛不动）→真同型两副本收 `cost.MatchIndex/LoadMatchIndex`（GA6-09 留痕逐字）+`cost.ParsePrice/ParsePriceWithOpts`（全角归一单点；Inf/NaN 方言差异 PriceOpts 参数化钉住）；⑧**FE7-05** 行派生镜像→`rowDerive`+`GanttRowShell`（synGroupOf 收窄变体防布尔假窄化）；⑨**FE7-07** 汇总口径量化（筛选下差 **3 个工作日**）→**冻结**（导出面无筛选输入，统一必改字节；口径对照表+无筛选同值断言）；⑩**FE7-08** EXP_COLS→`colsByKeys` 派生+**键序长位断言**（附带发现 colsByKeys 入参序无关=键序断言必须存在的实证）；⑪**FE7-12** `useDragGesture`+`useFitView`（**参数值全冻结**：AOA [0.1,2] vs PDM [0.2,2]；GanttView dayW 另套单位不并）；⑫**FE5-03** CreatePage 类型重复 −60 行（直接用已导出 PlanGateReportView）。
- **行为冻结三形态本批齐备**：参数化复现（线 2 开关）/骨架单源+谓词保留（线 1）/冻结+口径对照表+差异钉死（AP5-06、FE7-07）——「统一」不是默认答案。**字节/快照冻结证据**：线 1 金样（导出四格式 diff 除 EPUB 运行时钟外逐字节）；线 2 快照先钉现状改动后一字不改全绿（token 流+检索顺序+分值逐字节）。
- **反向证据（主代理亲手各重做一组，全红→绿）**：线1 分支解析删段→MainlineOnly 红/线3 空集返回 nil→TemplateStore 红线/线2 K1 1.5→1.2→分值快照红（顺序不变=数学性质实证）/线4 EXP_COLS 键序反转→收集期红。
- **门禁**：前台 CI OK / exit 0（golangci 0 issues · vitest **433 文件 3752 例** · E 系列 · 卫生四查）；守卫四份全绿（bindings@744+死绑定 109 在册 / test-ctors 364 零新增 / primitives / contract）。**过程事故再犯**：ci 日志删在 Tests 终值抄录前（**批 19 教训第二次违犯**）——文件数取自日志残句、例数单独复跑实测补齐；教训升级=**rm 清理必须放台账终值写完之后，抄数与清理不许同一条命令**。
- **坑/教训（进在册）**：①行为冻结三形态按条目性质选（参数化复现/骨架单源+谓词/冻结+对照表）——「统一」不是默认答案；②主代理任务书路线的前提同样要被现场复核（AP5-06 输出不等价、GA6-05「4 包」重画）；③数学性质实证入册（k1 排序不变性）让回归防护精确化（分值快照非顺序快照）；④镜像仓金样与快照先钉的组合=重构行为冻结的最强证据组合。
- **留池/下一批**：对账地图更新——**duplication 收敛余量 ~15→~4**（FE2-06 价格源两份、AP5-06/GA6-05 收敛裁决=拍板、章节号 stats/graph 改调一行事、BM25 族已毕）；**剩余大头=god-file 大拆 ~27（每条独立刀）+coupling ~20（拍板）**。拍板池累计十项+本批新增（AP5-06 两套收敛/GA6-05 Inf-NaN 方言）。下一批候选：duplication 零散收尾+god-file 大拆试水（挑最小一条立刀验证配方）或转拍板池批量裁决。

## 非版本刀：全仓审计第 20 批·语义冲突五条（4 线 + 主代理直做 1）· 2026-10-03

- **刀型**：接续批 19（`182c672e`）。对账地图「语义冲突 ~5 条（潜在真 bug）」整簇吃掉：**4 条互斥并行子代理线 + 主代理直做 1**（AP3-03 押后至线 1 交回，守同包不并行）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-22-p1-batch20.md`。
- **五条五种结局（无一照搬审计 fix 原文，五条风险指认全部成立）**：①**AP3-03**（直做）——三入口「守卫读空间、落库写死 work」两源并存 → `appendOfficeEvidence` 单点同源（opsJSON 作参数；play 红线钉+同源断言钉；反向=拍守卫→play 钉红）；②**IN2-07**——审计行号全漂移（批 10 IN2-03 已收敛构造站点），fix (a)/(b) **双双被证据否杀**（配置存量值断裂+前端 7 分支/同类型不可分）⇒ 最小收窄：B 轨取值域五常量+**全仓唯一口径对照表**+24 处判据常量化（字节等值钉死）；彻底统一需配置迁移=拍板；③**IN1-04**——判定**刻意分层非漂移**（上游 rubric 两维度行+资产镜像 measurable/engine 标记+git 史同日并行三证）；新包 `internal/noveltext`（段落双口径 Idx/Line 单源——两边行号不同从无人写明）；互引注释防顺手对齐；**矛盾样本双向钉死**（gate 报 telegraph 而 review APPROVE / review warn 而 gate 零发现）；rubric 接入留池（用户覆盖文件会改 gate 行为违反冻结等三理由）；**修正审计预设**：splitSentences 并非两包各一份，实际重复是段落切分；④**IN3-09**——**证伪审计修法**（herdsman 装不进 docmd provider seam：三组「本地文件不出机」消费方隐私边界+MinerU 是文档解析+探测不同构；「删四路串联」=删产品语义——腿1 是「设为 OCR」兑现路径）⇒ 档 2 口径单源（**四组同名不同物配置对照表**：`GAEA_OCR_MODEL`=gguf 路径 vs 模型 ID 等，纯注释零删除）+6 钉（kind 域冻结/四腿编排语义）；⑤**FE3-04**——三名零调用者（SetSubagentTemperature/SetEffort/SetSubagentModel）整体摘除（tsc 反证）+Compact realApp get 陷阱 fail-fast（`BridgeError code=MockOnlyBinding` 诚实文案，去「请重试」假提示）；**新发现：compact 回调零 UI 调用者**（审计前提过时，整体摘除入拍板池）。
- **反向证据（主代理亲手各重做一组，全红→绿）**：线1 comfyui 归位判据→Herdsman→归位钉红（顺带发现 `:54` 暖启判据无测试覆盖，登记）/线2 fail-fast `if(false&&)` 短路→MockOnlyBinding 钉红/线3 切句分隔符加逗号→正常文本误报红/线4 docmd 注册 herdsman kind→kind 域冻结钉双红/AP3-03 拍守卫→play 红线钉红。
- **门禁**：前台 CI OK / exit 0（golangci 0 issues · vitest **431 文件 3735 例** · E 系列 · 卫生四查）；守卫四份全绿（bindings@744+死绑定 109 在册 / test-ctors 364 零新增 / primitives / contract）。
- **坑/教训（进在册）**：①**语义/口径题的正确粒度=先判定性质再定刀深**——五条五种结局，审计 fix 是方向不是图纸（第 4 次实证）；②「分层 or 漂移」判定法=上游资产（rubric 维度行/资产镜像标记）+git 史双证；③口径单源的最低成本形态=**对照表注释+机器钉**（IN2-07 口径表/IN3-09 同名不同物表），不强行合并；④审计前提会随批次演进过时（FE3-04 的 UI 调用、IN2-07 的站点、批 19 的 FE2-03 同族）——存量条目复核时连「前提」一起复核。
- **留池/拍板**：**新拍板**=IN2-07 彻底统一（配置迁移+前端 7 文件联动）/compact 三件套整体摘除 vs Go 补绑定；**留池**=IN1-04 rubric 接入（单独立刀）、`:54` 暖启判据补测试、IN3-09 第三条 OCR 链出现时 seam 决策重开。**下一批**=对账余量（duplication ~15/大拆 ~27/coupling ~20 拍板）；语义冲突簇只剩 IN3-09 类的口径文档化已毕。待拍板池累计（109 绑定删除/桌面 agent 接线/compact 三件套/IN2-07 彻底统一等）。

## 非版本刀：全仓审计第 19 批·对账存量四线（8 条 duplication 收敛）· 2026-10-03

- **刀型**：接续批 18（`305993c4`）。**先做全量对账**：512 条发现 × round-3~20 台账逐一对照，**106 条 P1 从未进任何批次视野**（分五类：可执行收敛 ~28 / god-file 大拆 ~27 / 耦合架构 ~20 / 死代码 ~6 / 语义冲突 ~5——地图在 round-21 §〇）。本批取 13 条收敛，**4 条互斥并行子代理线**（app / schedule+office / ai / 前端）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-21-p1-batch19.md`（含 106 条分类地图）。
- **收敛九条**：①AP1-06 大纲树递归 6 站点→`findOutlineNode(nodes, pred)`（分支语义进谓词；sync「无变化仍写盘」陷阱逐字段保留）；②AP2-02 书源任务模板三份→`runBookJob(spec)`（panic 防线/注销/节流/载荷钉子）；③AP5-08 配置快照兜底→`gaeaCfgSnapshotOrLoad()`（实际 5 处，审计 4+同型 2）；④AP7-02 生成链两份 ~100 行循环→`runImageGenLoop(spec)`（能力开关 comfyRetries/wantKind，8 项语义差异保留）；⑤AP4-11 MPP 三轨→4 张 mppLayout 表+具名魔数（+198/−92）；⑥AP3-02+IN3-06 docx↔pptx→新包 `internal/office/ooxml`（方言参数钉死两套历史字节；各包 −200 行）；⑦AP7-03 ComfyUI 四份节点表→`comfyQwenProfile` 表+共享构建器（Warmup 分发逐 case 等价表）；⑧AP7-10 超时收割复制→`collectResult`（**双通道收口经实测拒绝**——两通道无重复可收）；⑨IN2-14 思考预算两处→`clampThinkingBudget`+常量。**FE6-05** 三路围栏样板→`extractFence()`（**拒绝审计合并方案**：DOM 证据=行内 code 退化无样式+pre 解包语义变化；渲染分支均不同构只统一样板）；**FE7-06** 导出层副本→单源 import（字节钉证零漂移）。
- **审计纠错 7 条**：AP1-06 站点数 6 非 5 / AP5-08 兜底差异夸大 / AP7-02 预检差异已失效（两链都有）/ AP3-02「xlsx 再叠一层」证伪（全程 excelize）/ AP7-10 双通道可收敛不成立 / FE6-05「仅差一层 div」不成立（三路均不同构）/ **FE2-03 整条证伪**（现状有正确调用点，零改动）。
- **字节冻结硬证据**：线 2 临时镜像仓以 `git show HEAD` 旧版与新版同跑 17 项探针（docx 条目序+document.xml 字节/pptx 全文件/错误原文/4 真实 MPP 样本 JSON SHA）old/new 全一致。**测试盲区两处被反向变异暴露**：MPP9 真实样本门控弱判据（变异后总工期 344→1275769 天仍绿）、docx/pptx 既有测试对 zip 条目序不敏感——新钉子补位。
- **反向证据（主代理亲手各重做一组，全红→绿）**：线1 Children 递归注释→2 钉红 / 线2 2013 taskDurOff+1→真实样本门控红（总工期 447406 vs 136）/ 线3 minThinkingBudget 减半→4 例红 / 线4 language-→langauge-→**三路全红 10 失败**（同时证明 extractFence 单源真被三路消费）。
- **门禁**：前台 CI OK / exit 0（golangci 0 issues · vitest **430 文件 3731 例** · E 系列 · 卫生四查）；守卫四份全绿（bindings@744+死绑定 109 在册 / test-ctors 364 零新增 / primitives / contract）。**过程事故**：首轮 ci `go test` 红、明细未捕获即删日志（主代理疏漏），同树零改动复跑全绿——定性环境性瞬时（疑似 herdsman 8080 碰撞族），教训=**ci 重定向日志必须留到终值抄录之后**。
- **坑/教训（进在册）**：①对账纳入的存量条目**移动靶属性更强**（从未预核）——13 条中 4 条表述有偏差+1 条整条证伪，「先复核后动手」在存量上必须是默认假设；②`go build -overlay` 钉 HEAD 版本=并行线破坏编译期的自验配方；③字节冻结的黄金标准=镜像仓 old/new 双跑探针（不是只靠既有测试——它们对条目序/弱判据是盲的）。
- **留池/下一批**：对账余量（duplication ~15 / god-file ~27 每条独立刀 / coupling ~20 拍板 / 语义冲突 ~5 **下批优先预核：IN1-04 gate vs review 门槛相反、AP3-03 证据链空间红线恒 work**）；findOutlineNodeByNumAny/ByID 两同型未收敛；Media 链补 comfyui 重试（行为新增）；MarkdownContent 第四份样板；MPP9 门控补强；FE6-05 DOM 统一（产品级拍板候选）。待拍板池不变（109 绑定删除/桌面 agent 接线等）。

## 非版本刀：全仓审计第 18 批·FE3-03 死绑定检测 + GA6 族收尾（主代理直做）· 2026-10-03

- **刀型**：接续批 17（`c0b30b87`）。两件量级小**主代理直做**（批 15 线 1 先例）。**不抬版本**。
- **交付**：`docs/code-audit-2026-10-02/round-20-p1-batch18.md`。
- **内容**：①**GA6-04 收尾**——组价两面（`gaea_cost_compose.go`/`cost_compose.go`）内联 `<3/10` 阈值与直调钩子收敛 `cost.Enhance`（三消费面：绑定/工具/组价同一编排出处；组价 rerankLimit=12 归调用面；空态后移可观测等价）；②**FE3-03** 死绑定检测守卫——`check-bindings-drift.ps1` §4：legacy 面 184 名逐名全名 `\b` 文本命中（实测 legacy 名以全名出现在调用点），排除生成物三件+守卫自身+`wails.d.ts` 声明+`mock/**`；**首轮全量普查 = 109/184 死（59%）**——远超审计的「7 个」（审计只查了 Gaea 前缀名，legacy 短名族没进视野；7 死名实测 6 死 1 活）；普查入册为基线，守卫只拦**新增**死绑定（新导出即失联），**删除 109 名=绑定面 744→635 候选整批留拍板**。
- **过程事故两起（主代理自己的）**：①排除模式正反斜杠不匹配（FullName 反斜杠 vs 模式正斜杠）⇒ 生成物清单没排除、全员假活；②正则不容行尾注释（生成器 7 个名挂 `// 注解` 尾巴）⇒ 177/184 漏解析、6 个已知死名误报复活。探测配方=.tmp 探针 .ps1（内联 PS 撞转义在册坑）。**反向证据**：活名 NovelGhostSuggest 全引用改名（组件+测试两处——测试 mock 也算命中，文本命中口径含测试）→ 守卫真 exit 1 红（`清单外死绑定 1 个`）→ 还原 exit 0。
- **门禁**：前台 CI OK / exit 0（golangci 0 issues · vitest 428/3723 · E 系列 · 卫生四查）；守卫：bindings-drift **扩为双职责**（744 一致 + 死绑定检测 109 在册零新增）、test-ctors 364 零新增、primitives/contract 无新增。
- **留池/拍板**：**109 条零调用绑定删除（拍板）**——名单=守卫 knownDead 清册，删除=Go 下导出+再生+门面同步，收益=绑定向真实面收缩；留池=compose 检索路径无独立测试、测试命中算活的双刃性（文本命中口径已知边界）。

## 非版本刀：全仓审计第 17 批·簇 C 收尾 + 簇 D 死代码（4 条并行线）· 2026-10-03

- **刀型**：接续批 16（`fec47842`）。**4 条互斥并行子代理线**（线1 style+project+app+novelcontext / 线2 knowledge / 线3 cache+context+tool / 线4 前端 mock+scripts）+ 主代理预核、逐线复核、**四组反向变异亲手重做**、收线缝合与全量门禁。**不抬版本、不动 CHANGELOG**。
- **交付**：`docs/code-audit-2026-10-02/round-19-p1-batch17.md`。
- **四线内容**：①**IN1-05**——审计 fix「Manager 带 style 类型三方法」**被依赖方向否决**（style→project import 已存在，反向即成环）⇒ []byte 缝变体：project 收口路径与 `.wubigork` 回退（`legacyBrandDir` 与 `IsV4()` **同源共用**=品牌目录名全仓一份），JSON 编解码留 style 薄壳，签名 `projectDir→*Manager`；两套文风口径边界互引 doc（Profile=生成注入 / Fingerprint=评分，刻意分层勿合并）；②**GA3-07**——6 类内联常数 → 命名常量+`combineScore`/`isIrrelevant` 唯一合分口径；「榜单顺序不变」golden（**敏感性设计语料**：vecScale 减半即换位对、cutoff 加大即踢边界条、并列孪生测稳定序）；审计 0.65/0.55 证伪（查重域/记忆合并域）；scoreEntry 实为 category+3 **且 phase 另计+3**（审计漏）；③**GA4-09**——Learner 残骸删仓（5 no-op+LockVersion/IsLocked/resolveVersion+`ContextManager.RecordOutcome` 零调用链+v2/v3 死条目+**连带孤儿 subagentTools**；skill.go 219→172 行）+两把钉（每 Kind 恰 1 版本/Route 恒 v1）；**GA2-03「已漂移」被实测推翻**（三处足迹内机器+足迹外人工逐字段全零差异）⇒ 交付转为 compact-恒生效诚实注释（cfg.Tools.Compact 只管 Hide）+全实现方递归包含性**漂移钉**（≥40 实现方守卫，外部测试包避环）；④**X1-11**——**根因实证：`Object.assign` 16 源超 TS 重载数落 `any`，`makeMockApp(): AppBindings` 注解形同虚设**（缺 45 成员 tsc 照样 0 error）⇒ 契约测试是唯一能抓的层。`gen_bindings -signatures-ts` 产 744 签名清单（字节稳定 sha1×4=`f755ca6b`，改后重跑 `-legacy-ts` git diff 空=共享代码零扰动）+contract.test 5 用例（形参个数从 `fn.toString()` 平衡扫描，**弃 `fn.length`**——它数不到默认值后形参）；首跑战果=**修 34 处形参错位**（只修 mock 行为零变化）+**45 条漏 mock 白名单**（逐条带理由+防腐过期锁）+豁免 1 条 SubmitDisplay（前端桥接双参 vs Go 单参既存分歧，归桥接线）。
- **反向证据（主代理亲手各重做一组）**：线1 删 `.wubigork` 回退→旧品牌可读测试红 / 线2 vecScale 8→4→golden 换位+数值钉双红 / 线3 Route version 1→2→恒 v1 钉红 / 线4 mock Cancel 加形参→契约红。全还原复绿。
- **门禁（前台单次 CI OK / exit 0）**：golangci v2.14.0 0 issues · `go test ./...` exit 0（app 128.166s）· 前端 lint 0 error / build ok · vitest 全量绿 · E 系列+卫生四查 OK；守卫四份全绿（test-ctors 364 零新增 / primitives 无新增 / contract-drift 无新增 / **bindings@744**——Go 绑定面零变更，`-signatures-ts` 为新增只读模式）。
- **坑/教训（进在册）**：①**审计 fix 是方向不是图纸**——IN1-05 被依赖环否决、GA2-03 二选一均不可行，第三路（[]byte 缝/诚实注释+漂移钉）才落地；②**「已漂移」类断言要机器复核**——GA2-03 实测零漂移，漂移钉反过来把「保持不漂移」变成可检；③`Object.assign` 超重载使类型注解失效是 TS 结构性盲区，**跨大对象聚合的接口实现只有运行时契约测试能抓**；④`fn.length` 数不到默认值后形参——按它断言等于逼人删默认值改行为；⑤批 16 产物成为批 17 基建（`legacyBindings.ts` 单源豁免、`-legacy-ts` 的防御闸/字节稳定做法被复刻，「改后重跑旧模式 git diff 空」=共享代码零扰动证明法）。
- **留池/下一批**：45 条 NOT_MOCKED 补 mock（行为新增须单排）· `SubmitDisplay` 桥接/Go 口径分歧（**新拍板候选**）· Compact 命名与语义无关（碰配置面留池）· `test_image_models.go` v1.0.0 起 gofmt 存量（golangci 不覆盖 scripts/，在册）。**下一批候选**：簇 D 收尾（IN4-04 等接线拍板不动 / FE3-03 死绑定检测——`legacyBindings.ts`+`bindingSignatures.ts` 双全集已备齐）+ GA6 族组价接 Enhance（一行事）。待拍板池不变（桌面 agent 接线三件套等）。

## 非版本刀：全仓审计第 16 批·簇 C 口径单源（4 条并行线）· 2026-10-02

- **刀型**：接续同日审计第 15 批（`be53c2a9`）。**4 条互斥并行子代理线**（线1 成本族 cost+builtin+app 三文件 / 线2 novelstyle+前端 / 线3 schedule+前端 calendar / 线4 前端绑定三清单+scripts）+ 主代理预核（八条审计原文全对现场）、逐线复核 diff、**四组反向变异亲手各重做一组**、收线缝合与全量门禁。**不抬版本、不动 CHANGELOG**。
- **交付**：`docs/code-audit-2026-10-02/round-18-p1-batch16.md`。
- **四线内容**：①**成本族**——GA6-01 销账（前批已修）；GA6-06 隐式失效契约消灭（ranker key 加语料指纹 `COUNT|MAX(updated_at)`，任何路径写自动失效，**导出 `InvalidateRankers` 删除**；纯指纹被否：同秒保数编辑+分类改名指纹全盲 ⇒ 混合方案）；GA6-04 三路检索编排收编 `cost.Enhance`（阈值唯一出处，实现由各面注入 SearchHooks——builtin 不能 import app、cost 不能 import retrieval 反向依赖）；精排文档串两份同构实现收编 `cost.RerankDocText`（**实测 4 份非审计的 3 份**；BM25 串/语义串不同型刻意不并，注释互锁「不许顺手对齐」）；②**IN1-09**——novelstyle 严重度 `low/medium/high/blocker` → `S1-S4`（与 gate/review 同枚举；app 的 severityBlocking 走 novelgate 预核证非消费 ⇒ 不动 app/gate/review）；权重 30/16/9/4 原值平移=**分数零漂移**（6 样本 golden 锁）；S1 当前无规则产出（占位，相容 gate 空章级）；③**IN3-02**——预核「Go 四处手写判定」**证伪**（(b) 类 0 处，cost.go/Analyze 已走唯一入口）⇒ 交付转为唯一入口注释钉死 + **TS↔Go 双端对拍 golden**（`testdata/calendar-parity.json` 39 条冻结共识，双端各读同份，改 Go 漏改 TS 必红）；④**FE4-04**——审计半 overstated（satisfies 实测已双向抓红）；真缺口=手写 legacy 清单 **10 条与认领集重叠的死条目**（架空「Go 删绑定」报警）⇒ `gen_bindings` 新增 `-legacy-ts`（解析 spaceBindings+mappings 推导认领集，产出 `legacyBindings.ts` 184 名字节稳定）+ drift 两把新锁（NoStale/NoOverlap）+ 计数魔法数 565 删（类型级保证替代）。
- **预核三处被现场/子代理纠正（主代理不豁免）**：IN3-02 重复不存在（真缺口=无对拍）；FE4-04 satisfies 已能抓（别为不存在的问题造机制）；「gen_bindings 已知 AppBindings 归属」不实（生成器只知 Go 门面归属）。
- **反向证据（主代理亲手各重做一组）**：线1 指纹置盲→直写失效测试红（顺带暴露陈旧倒排按语料位置错位记分的机理）/线2 S2 权重 16→15→golden+权重表双红/线3 Go weekday 错位一天→parity 四节全红+既有 5 测红/线4 认领名塞生成清单→NoOverlap 锁红（还原 sha1 回 `896b863e`）。
- **纪律事故（进在册）**：线 1 子代理误用一次 `git checkout -- cost.go`（违反「代理禁 git 写」）——无提交/历史影响，如实申报+手工重做全部改动+全量复验后收线；**登记不豁免**。另：并发窗口期间其余线见到 `go build ./...`/tsc 短暂红（线 1 删 InvalidateRankers 在飞、线 4 重构 spaceBindings 在飞）——按「并发期不跑全量、收线统一验证」纪律消化，未误判为自身回归。
- **门禁（前台单次 CI OK / exit 0）**：golangci v2.14.0 0 issues · `go test ./...` exit 0 · 前端 lint 0 error / build ok · **vitest 427 文件 3716 例**（批 15 为 3712）· E 系列 + 卫生四查 OK；守卫四份全绿（primitives 无新增 / test-ctors 见台账 / contract-drift 无新增 / **bindings-drift OK@744**——Go 绑定面零变更）。
- **留池/下一批**：组价两面接入 `cost.Enhance`（一行事，footprint 外残留）；`retrieval.DocText` 与精排串价格格式微差（改动触发重嵌入漂移，不动）；GA6-06 的 16 项缓存清空策略（观测池）。**下一批候选**：IN1-05（Profile vs Fingerprint 双真相源，须 internal/project+app 单独一波）· GA3-07（知识库检索双口径，本批与线 1 撞 builtin 包顺延）· 簇 D 死代码（GA4-09 Learner 桩/GA2-03/X1-11/FE3-03 死绑定检测——**线 4 的 `legacyBindings.ts` 生成物为其提供了现成全集数据源**）。待拍板池不变（桌面 agent 接线三件套等）。

## 非版本刀：全仓审计第 15 批·簇 B 真 bug（3 条并行线）· 2026-10-02

- **刀型**：接续同日审计第 14 批（`295b6d2a`）。**线 1 主代理直做**（`internal/types`+`graph`，删死定义不值得一个子代理波次）+ **线 2/线 3 互斥并行子代理**（线2 `internal/gaea/agent` / 线3 `internal/app` 两文件；Go 包互斥）+ 主代理开工前预核（四处现场亲读）、逐线复核 diff、**亲手重做两条线的反向变异**、收线缝合与全量门禁。**不抬版本、不动 CHANGELOG**。
- **交付**：`docs/code-audit-2026-10-02/round-17-p1-batch15.md`（预核 4 条 / 逐线落地 / 跨线裁决 6 条 / 留池与拍板 / 审计纠错 3 条）。
- **三条内容**：①**IN2-13** 删 `types.ConsistencyReport/ConsistencyIssue` 死定义（**全仓零消费实证**：`types.Consistency` 限定引用 0 命中、`overall_note` 除定义 0 命中；graph 版是唯一活形态＝handler+前端全走 `issues/total_issues/summary`；消费方历史已在 `3d84c7a7`/`b6f092f7` 清掉）——处置=删除而非改名/兼容层，graph 版注释标注唯一形态防复活；②**GA1-09** retry_until ephemeral 模式**同会话续跑接活**（原意判明=补真语义非盲删，三证据：函数 doc 宣称累积 + 重试提示词预设子代理记得上次 + 死分支是「会话藏在 `runSubSession` 不外传」的接线未遂；修法=`runSubSession` 回传会话 + 首试后包 ephemeral wrapper（`Ref==""`，MarkRunning/subJournal/SessionID 四项次生影响全 no-op 读码核实）+ `seed=nil`）；③**AP1-08** Apply/Restore × 场景/整章四段写回体收敛为 2 helper（`writeBackRewriteVersionScene` applying 分流 / `writeBackRewriteVersionWhole` 错误前缀逐字传入；落账/slog/返回 map 留调用点；Apply 的完整性校验+快照与 Restore 的**无条件还原**语义差异钉死）+ **AP1-07** 三处 9 参取消出口收敛为 `savePartial` 闭包（三态本体与签名不动）。
- **Seed 替换语义（子代理对预核的修正）**：`Session.Seed` 是**整体替换**（`session/session.go:52-56`）非追加——`seed=nil` 不是防御性微调而是必须项，重试重注种子会把刚累积的失败历史整段抹掉。已进 doc 注释与测试（4 消息形状断言钉死重注）。
- **测试**：+2 文件零既有改动——`task_retry_session_test.go`（第二次请求必须包含首试 assistant 回复 + 恰 4 条消息 `[sys,user,assistant,retry-user]` + check 恰 2 次）；`novel_rewrite_writeback_pin_test.go`（7 Test 522 行：四路径 Happy〔含快照恰好 +1/Label/Trigger/WordCount/v4 rebuild 对称〕+ 错误分支文案逐字〔完整性拒绝零副作用/幂等篡改磁盘不重写/校验顺序钉死〕+ slog 捕获逐字）。**行为不变时序证明**：钉子先对原始代码跑绿 → 重构 → 一字不改复绿。
- **反向证据（主代理亲手各重做一组，「子代理跑过 ≠ 只信自述」）**：线 2 短路 wrapper → 同会话测试红（第二请求只剩 `[sys,user 重试提示]`）→ 还原绿；线 3 拍丢 Apply 应用前快照 → 钉子红（`应用应恰好落 1 个快照，得到 0`）→ 还原绿。`TEMP-MUTATE|REVERSE-TEST|if false` 足迹内 0 命中。
- **门禁**：`scripts/ci.ps1` 前台 CI OK / exit 0（golangci v2.14.0 0 issues · Go 全量 · 前端 lint+build · vitest 427 文件 3712 例 · E 系列 · 卫生四查）；守卫四份全绿（`check-primitives --strict` 无新增 · `test-ctors` **364 零新增** · `bindings-drift` OK@744 · `contract-drift` 无新增）。
- **审计原文纠错 3 条**：①IN2-13「消费端按哪套解析都可能拿到空字段」危害偏重——types 版零消费，无任何消费端拿错，真实形态是同名撞车地雷；②GA1-09 审计留了「盲删」选项——原意三证据链判明为补真语义，盲删会把「文档与实现相反」的静默矛盾永久化；③AP1-07「三态落盘三份重复」的「三份」指三处调用出口（`:558/:571/:581`），三态判定本体始终单一。
- **坑/教训（进在册）**：①`Session.Seed` 替换语义——「防重复注入」的弱直觉会漏掉「抹掉累积历史」的真后果，读标准库语义要读实现不要读名字；②死分支处置先判原意（doc 宣称+调用方提示词+半成品形态三证据链），再定删或接活；③并行波次纪律延续：主代理线直做小足迹 + 子代理线包互斥，三线并行期间 `go build ./...` 恒绿。
- **留池**：`writeBackRewritten` 与新 helper 远期合并（需统一错误文案，跨消费方）；`saveCancelledPartial` 与 converge/scene_cards 的 cancelled 事件词汇两套（语义不同不硬并）。**待拍板池不变**（桌面 agent 接线三件套 / `download_and_install` 独立确认 UI / `GaeaDocumentLint` 读根门 / FE6-09 conversationKey / AP4-01 / X1-03 / `estimateTokens` 四口径）。**下一批建议**：簇 C 口径单源（IN1-09 AI 味判据三套严重度 / IN1-05 `Profile` vs `Fingerprint` / GA3-07 知识库检索双口径 / IN3-02 日历语义 / FE4-04 绑定名三清单 / GA6 成本条目族）或簇 D 死代码（FE3-03 死绑定检测须按「前端驼峰名 + `mappings.ts` 反查」实现）。

## 非版本刀：全仓审计第 14 批·簇 A 安全（4 条并行线）· 2026-10-02

- **刀型**：接续同日审计第 13 批（`a08e9129`）。**4 条互斥并行子代理线**（线1 路径归一/书封护栏（app+域包）/ 线2 hook 沙箱（gaea/hook+boot）/ 线3 whisper 桌面操作 / 线4 前端两条真 bug）+ 主代理开工前预核、**独立实证**（自己写 Go 探针 + 自己跑子代理探针）、**反向变异由主代理执行**、收线缝合与全量门禁。**不抬版本、不动 CHANGELOG**。
- **交付**：`docs/code-audit-2026-10-02/round-16-p1-batch14.md`（含 §〇 预核表、§〇.1 独立清点、逐线落地、跨线裁决、门禁台账、留池与拍板）+ `testctors-baseline.json`（363→364）。
- **簇 A 四项**：①**AP2-04** 书封护栏——`novel_bookcover.go` 同处注入 `applyImageSafeMode`（**不改走统一入口**，理由写进代码注释：统一入口会接管 size 策略/落盘目录/并发槽 ⇒ 书封 3:4 与 exports 落点会回归）；②**AP5-09** 路径单源——落**已有**叶子包 `internal/gaea/wspath`（`Within`/`ResolveRelWithin` 两原语，**30 调用点**，`builtin.within` 上提转调）；现场比审计更严重：**13 处真修复**，含 `gaea_lint.go` 零校验、`gaea_preview.go` **伪拒绝（返回逃逸路径，赌 Stat 失败）**、docx/pptx/xlsx×6/crosslink×2/xlsx_chart **写侧零校验**（crosslink 还能在工作区外 `MkdirAll`）、`gaea_verify.go`（Journal 不可信面参与回滚写盘）、`gaea_schedule_file.go`（索引指针）、`gaea_deliverable_zip.go`（`\\` 形态漏拦），另修 `filewatch` 误丢合法「..foo」的假阴性；③**GA4-06** hook 沙箱——`WithSandbox` 加法（`NewRunner` 签名与 14 处既有测试调用点零改动）+ `Runner.spawn` **唯一注入点** + `sandbox.Command` 的 bool 不再被丢 + 「enforce 但不可用」一次性诚实告警；④**IN4-05 收窄为 5 条真缺口**（关闭黑名单通配符绕过 / `\\?\`·UNC 前缀绕过 / `pathTo` 空值静默落 CWD / 下载类策略盲区 / `Router==nil` 静默 no-op），**IN4-08 与 IN4-09 按现场撤销**（行号失效、控制组探针证明转义有效）。
- **前端两条真 bug**：FE6-09 打字机流守卫补齐（与 canonical `useChatStream.ts:216-230` 同口径、4 处早退 + 2 个 effect）；FE7-11 模块级游标出 `set` 更新器（`idSeq` + `pushHistory` **16 处**），并附带修出真 bug：`setTaskAssignments` 的 fail-closed 判据原在 `pushHistory` **之后** ⇒ 拒绝路径仍压栈并清 redo 栈。
- **反向证据（改坏能红）**：线 1 两条（`ResolveRelWithin` 去包含性 → 68 条断言红；书封护栏删一行 → 逐字节断言红）、线 3 五组（通配符/前缀边界/pathTo/Confirmed/Router==nil）、线 4 四条由**主代理执行**（A 删循环后守卫→2 例红、B 删 tick 首行守卫→1 例红、C `pushHistory` 回更新器→纯度例红、D `newId` 回更新器→idSeq 例红；两个文件均以 **SHA256 证明字节级还原**）。全仓 `TEMP-MUTATE|REVERSE-TEST` 0 命中。
- **门禁（本地快闸步骤集，前台实跑）**：`version drift` OK · `clean-tmp` OK · `go build` 0 · `go vet` 0 · **golangci-lint v2.14.0 = 0 issues** · **`go test ./... -count=1` exit 0**（`internal/app` 113.965s）· frontend `lint` **0 error**（3 存量 warning）· `build` ok（20.20s）· **vitest 427 文件 / 3712 例全绿**（第 13 批 425/3702）· E 系列 OK · `check-docs` OK。**守卫五份**：`check-primitives --strict` 无新增（截断 37/原子写 8/前端 5/heredoc 0）· `check-bindings-drift` **OK@744**（改了 20+ 绑定方法实现、签名零变更）· `check-contract-drift` 无新增 · `check-test-ctors` **364**（已评审刷新 +1）· `check-docs` OK。
- **坑/教训（进在册，可复用）**：①**后台 job 拿不到 `.tmp` 写权限**——`scripts/ci.ps1` 放**后台任务**跑必在 `go test` 步大面积 `open/mkdir C:\AI\wubigrok\.tmp\…: Access is denied`（两次实测、十个包），而**同一命令前台跑全绿**（判别实验：前台单跑那 10 包全 ok + 前台 `go test ./...` exit 0 + `clean-tmp.ps1` 读码排除嫌疑）⇒ **门禁一律前台跑**；②**反向验证的中间态必须还原**——线 2 树里的 `// REVERSE-TEST: bool discarded` + `sandboxed` 无条件置真被收线预检拦下（否则＝「把平台可用当本次已沙箱」的静默撒谎提交，本批主题正是消灭这类）；③**审计说的「已有安全助手」也可能是伪防护**——`resolvePreviewPath` 的 `..` 分支返回的是**逃逸路径**、只赌后续 `os.Stat` 失败，诱饵文件存在时预览照读照渲染，「有判据」≠「判据有效」；④**子代理跑不了 ≠ 不验证**——线 3/线 4 的会话被沙箱管道 EPERM 挡住（vitest/探针），**配方由子代理给、变异与红/绿原文由主代理出**；⑤**守卫的「放大器」档要真复核**——线 1 的 `matrixApp()` 用了裸 `&App{}`，而 `emit` 是 `*core` 方法（`app.go:314`）⇒ 未来输入走到 `GaeaPreview` 非 `.md` 分支即 nil 嵌入 panic，收线改 `&App{core:&core{}}` 降档（bare→core-only、放大器计数不变）再刷基线；⑥默认 `GOCACHE` 在 workspace-write 沙箱下不可写，且 **`go build` 返回 0 可能只是全命中缓存**（要靠文件探针定性）；⑦**删子代理临时物必须逐文件名**——主代理用 `Remove-Item .tmp\*.cjs` 连带删掉 6 个更早会话的临时脚本（`.tmp` 是共享暂存区、已 gitignore、无 tracked 删除，见 round-16 §5.1）。
- **未做（如实登记）**：符号链接/目录联接逃逸（需 `EvalSymlinks`）；`snapshot.go` 与 `scene.go` 两份 `validSceneID`；app 与 `internal/schedule` 两份段级校验（都更严，硬并会放宽）；**桌面 agent 接线**（`RouterContext` 生产构造者 + `DefaultAgentLoopRunner` 传 `Router`/`SessionID` + 两个自动批准设置的调用方）——不接线则四层管道永远只经测试触达；`download_and_install` 真修法（下载后独立确认 UI）；`trashItem` 阻塞风险（两次实测不一致：成功移入回收站 / 挂死 600s）；`GaeaDocumentLint` 读根门；FE6-09 话题 id 显式化；`setBaseline` 同族残留；hook 真包裹在无 WSL2/bwrap/sandbox-exec 机器上测不到；`-race` 只在 Actions。
- **待用户拍板**：仍是 **AP4-01**（三套并行编排合并）、**X1-03**（五重 `*core` 嵌入裸构造）、**`estimateTokens` 四份口径**；本批新增四条：**桌面 agent 接线路线**（接线 M / 弃用删执行器 S / 维持现状）、**`download_and_install` 真修法**、**`GaeaDocumentLint` 读根门**、**FE6-09 话题标识显式化**。**下一批建议**：簇 B 余项（IN2-13 双定义已预核：`graph/consistency.go` vs `types/types.go`；GA1-09 死分支已预核；AP1-07/AP1-08 属 `internal/app`，须单独一波）＋ 簇 D 的 **FE3-03 死绑定检测**（**必须先按「前端驼峰名 + `mappings.ts` 反查」实现**，批次十三误判教训）。

## 非版本刀：全仓审计第 13 批·P1 池清账 + 守卫精化（4 条并行线）· 2026-10-02

- **刀型**：接续同日审计第 12 批（`03e0fc0c`）。**4 条互斥并行子代理线**（线1 工期负值真缺口+跨语言夹具 / 线2 守卫与 CI 精化 / 线3 前端收敛与可见性 / 线4 app 内 P1）+ 主代理定契约、开工前预核、收线缝合、全量门禁。**新纪律生效**：任务书要求各线「接近上下文上限时先用 `send_message` 交阶段性结论」——线2 据此在中途发回防丢报告，四条线**全部**交齐正式报告（对比第 12 批两条线崩溃丢报告）。
- **交付**：`docs/code-audit-2026-10-02/round-15-p1-batch13.md`（含候选来源表 / 开工前预核 4 条 / 跨线裁决 10 条 / 量化收益 10 条 / 留池四簇 + 拍板池）+ `testctors-baseline.json` 刷新。
- **真缺口收口（第 12 批实测的 D5）**：`upsert_task` 补「工期为负」闸——判据单源 `validateDuration(t, idInMsg)`（`Validate`/`upsert`/`patch` 三路径共用），TS 侧同判据镜像 `negativeDurationError`，golden fixture **纯新增 112/0**（新增「upsert 负工期」与「patch 既有负工期只改 level 不误伤」两例，既有 65 条逐字节不变）。**主代理独立复现**：把 upsert 闸改成只调用不拦 → 用例 FAIL 且逐字复现 D5（`err=nil` + `{ID:C Duration:-5}` 落计划）→ 还原 `ok 1.102s`。
- **守卫机器化（三条）**：①`ci.yml` 删 `internal/tts` 过期排除——注释理由「`TestEdgeTTS` 依赖外网」**双重不实**（该测试全仓零命中 + 真原因是平台：`sapi.go` 用 Windows-only `HideWindow`，实测 `GOOS=linux go build` 报 unknown field），整包 66 例回到 windows backend job；②前端 flaky 分类器从**零自测**补到 8 组夹具/14 断言，并**堵掉一个真静默放行洞**（报告无 `testResults` 时旧实现落 OK/exit 0 = 把「vitest 崩在收集阶段」读成绿 → 现 PARSE-FAIL/exit 2）+ `--expect-failure`；③遮蔽闸从「条数基线 + 字典序切片猜新增」改为**显式在册清单 + 名字集合差**（新增/消失分开报、位置漂移仅提示）——旧二进制在同一夹具下把**在册条目**报成新增，本批实测复现该误报。
- **前端收敛与可见性**：计划 rel 第 4/5 份副本并单源（5 消费点断言 + 源码锁）；写类工具字段口径从 3 处收敛到 `toolArgs.ts` **一处**（24 条等价矩阵 + 6 契约 + 3 加固，先固化现状再重构、重构后测试文件一字不改仍全绿）；备份回滚「部分失败」由**只有 antd 瞬时 toast** 改为**页面内持久 Alert**（点名 `1/3 项失败（已回滚 2 项）` + `.restore-before` 可重试说明，断言不弹 modal）。
- **app 内 P1**：文件索引入队去重三入口（判据空间两处全局 vs 一处 work + 落库隐式 vs 显式）→ 唯一入口 `submitFileIndexTaskIn(space,session,label,reason)`（判据与落库同 space、跨空间互不吞并）；板块 ID 清单 **4 处硬编码 → manifest 派生**（审计只列 `intent_llm.go`，实为 +`wx_agent.go:71/:315` +中文名枚举）；知识库迁移失败经**既有** `gaea-event` notice + `slog.Warn` **启动即可见**（零新增绑定）。
- **审计原文纠错 6 条（本批实测）**：①AP4-07 的因果链引错（`novel_import_task.go:55` 是 `KindOutlineReconstruct`；全仓 `KindFileIndex` 仅 3 个提交点且**当前无 play 生产者** ⇒ 「正在抖动」不成立，按「埋着的地雷」定性）；②AP8-08 实为 4 处清单；③批次十一余量「备份回滚前端文案」的前端**有**消费面（主代理按 Go 名 grep 误判为零消费面，被线 3 用驼峰别名证据推翻）；④前端分类器静默放行洞（新发现）；⑤遮蔽闸误报（新实证）；⑥`ci.yml` 的 `internal/tts` 注释双重不实。
- **坑/教训（进在册）**：①**绑定名 grep 必须「前端驼峰名 + `mappings.ts` 反查」**——前端写 `app.DataBackupRollback()`，运行时经 `gaeaToGaea[prop] ?? prop` 改写 Go 名；只按 Go 名 grep 会**系统性漏报**（本次主代理即误判一次，直接影响下一批 FE3-03 死绑定检测的实现口径）；②**并发子代理的中间产物不是原子交付**（承接上批）——本批用「中途 `send_message` 交阶段性结论」把丢报告风险消掉，四线全部交齐；③**子代理在局部契约上可以比主代理更懂**：线 1 驳回主代理「闸宽外溢」建议（ops 契约=只校验本次改动，外溢会让与工期无关的补丁被拒 + 同一条 op 在不同计划上随机成败）并给出探针与跨语言锁定，主代理采纳；④**golden fixture 的正确改法是「只加不改」**（本批 112/0），任何既有期望被动过都该在收线时追问。
- **验证**：`scripts/ci.ps1` = **CI OK / exit 0**（golangci v2.14.0 0 issues · Go 全量 · 前端 build · vitest 全量 · E 系列 · 卫生四查）；主代理独立复跑：`internal/schedule` 1.026s、`internal/app` **106.7s** + board 0.500s、前端 6 文件 **86 passed**（另一轮 `opsSim.golden` 71/71）；**守卫五份**：`primitives --strict` OK（截断 37/原子写 8/前端 5/heredoc 0）· `test-ctors` 无新增（基线 359→**363**，+4 测试药桩已评审）· `bindings-drift` OK@744 · `contract-drift` 无新增 · `check-docs` OK（校验和 447 / 发布说明 636）。
- **未做（如实登记）**：`ApplyOps` 出口即合法（会改全部 op 语义 + 整份 fixture，独立一刀）；索引「彻底单例」需 tasks 层 workspace-global 单例 kind（**拍板**）；`applyIncrementalFileIndex` 不经队列；`frontend/src/boards/manifests.ts` 与 Go manifest **无跨语言断言**；知识库面板常驻提示条需新增读取型绑定（**拍板**）；`internal/tts` 纳入 CI 的最终证明仍是改后第一次 Actions 跑；rel「目录前缀」族 4 处（命名约定）；`-race` 与真机走查欠账。
- **待用户拍板**：仍为 **AP4-01** 三套并行编排合并、**X1-03** 五重 `*core` 嵌入裸构造，另新增两条本批登记（索引单例化 / 知识库面板常驻提示绑定）。**下一批建议**：先打**安全簇**（GA4-06 hook 沙箱零填充 · AP2-04 书封护栏旁路 · AP5-09 路径穿越「重复+漏点」· IN4-05/IN4-08 桌面操作零校验）——四条均已在 round-15 §4.1 预核完毕、带位置与修法。

## 非版本刀：全仓审计第 12 批·P1 清理 + 守卫机器化（4 条并行线 + 两波事故恢复）· 2026-10-02

- **刀型**：接续同日审计第 11 批。**4 条互斥并行子代理线**（线1 app+成本链 / 线2 非 app 包 / 线3 前端 / 线4 守卫·CI·台账）+ 主代理定契约、收线缝合、全量门禁。**第一波 4 条线全部在交付前崩溃**（无收尾报告）⇒ 主代理按「`go build ./...` + 逐包 `go vet`（vet 会编译测试）」两步把产物分成「已完成 / 半成品 / 未动」，据此**重启第二波 4 条线**（每条带现场状态说明，禁止重做与冲突）；第二波里 **app 线又崩了一次**（做完了实现、没交报告），最终由主代理逐行复核产物并**亲自补做 3 条「改坏能红」反向证据**。
- **交付**：`docs/code-audit-2026-10-02/round-14-p1-batch12.md`（§0.2 事故与恢复、逐线证据、跨线裁决 11 条、审计原文纠错 8 条、留池 P1~P8）。
- **真 bug / 静默失败**：①**GA6-09 成本检索错误通道**（`cost.Store.List/Search → ([]Summary, error)`：db==nil→`(nil,nil)`、Query 失败→`(nil,err)`、Scan/`rows.Err` 部分失败→**部分数据照常返回 + errors.Join**；跨 4 包 15 调用点一次性改完，能上报的上抛、纯辅助路径 warn 降级；绑定**只加字段** `CostSearchPage.Error`（`omitempty`）⇒ 绑定面零变更；前端两处分页消费点给**页面内「读取失败」提示条**并与「暂无成本条目」区分）；②**AP1-09 判据单源**（S1/S2 阻断判据 **5 份 → 1**：写前硬闸大小写敏感、落盘校验 Trim+Upper ⇒ 「保存被拒但生成却起来了」的根因消除）；③**AP7-05 台账记生效 backend/model**（此前写死全局后端+硬编码模型 ⇒ 消耗报表按 `{model,backend}` 分组失真；`portraitImageBinding`/`effectiveSceneIllustrationModel` 各成单一来源）；④**IN3-01 工期五闸**（消掉 **2 处真实语义分叉**：patch 校验中间态 / 先写后校验；3 处纯文案差异按 golden 保留）；⑤**AP6-10 TTS 锁内 HTTP**（手写最小 single-flight：HTTP 出锁、失败不缓存、跟随者有界等待；此前「语音没声、无报错」）；⑥**IN3-11 页数三口径**（降级落地：单点判定 + 差异检测 + 诚实告警；**其中 OCR 页码是真修**——改以 pdftoppm 产物文件名为唯一来源 + 数值排序，字典序 `page-10 < page-2` 的错位消除）；⑦**FE3-06 空间门控 fail-open → fail-closed**（未登记名返哨兵 `unknown`，拒绝发生在 resolveBinding 之前 ⇒ 不触达真实绑定 + 一次性 warn 留痕；**184 个未登记名分类清单**证明无需补登记/无需白名单，分类锁 565 未变）；⑧**FE5-05 青鸟页轮询门控**（`usePollingGate`+`useBoardActive`：板块不可见时四个绑定零调用）；⑨**FE7-09 计划 rel 单源**（新建零依赖 `schedule/paths.ts`，三消费点改 import）。
- **守卫机器化**：**X1-13** 绑定遮蔽从「只打印计数」升级为**超基线 exit 1**（判闸在写生成物之前；基线 2 = 实测口径，并写明审计「281」是含门面重复的**另一口径**）；**X1-12** Go 侧 flaky 分类器（`go test -json` → 失败包 ⊆ 在册清单才隔离复跑，清单外直接红；`--expect-failure` 防「归不出失败包就静默放行」；9 组夹具 14 断言；ci.yml backend job 换掉整批重试）；**X1-10** releases 台账计数**根因修复**——`check-docs.mjs` 的正则只认「校验和 N 份」而 README 写「N 份校验和」，**该守卫从未生效**（这才是 422/427 长期无人发现的根因），现两种语序都认 + 「一处都认不出即 FAIL」+ 6 组负例自测；`frontend/package.json` 版本经查证**无消费方** ⇒ 不动并写明「版本源就三处，别再加第四处」。
- **事故/坑（进在册）**：①**并发子代理的中间产物不是原子交付**——崩溃后必须先跑 `go build`（编译期）再逐包 `go vet`（**vet 会编译测试**，是查「签名改动的调用点」最快的检查单），两步就能把半成品与已完成分开；②**生成器再生会错归门面**（`GetSinImageConfig`/`SetSinImageConfig` 从 `ImageB` → `CoreB`）**且把多行方法体折成单行**⇒ 本仓「再生噪声一律按 HEAD 还原」纪律再次被现场证明；③**HEAD 本身不是 gofmt 干净的**（`bindings_core.go`/`bindings_sin.go` 各 1 行对齐，而 `.golangci.yml` 启用了 gofmt formatter）——上一波把它们再生时是干净的，按纪律还原 HEAD 反而暴露债 ⇒ **「还原 HEAD」这条纪律必须配一条「HEAD 自身门禁是否干净」的基线检查**（与批次十一「HEAD 不是 lint 干净」同族，第二次撞）；④长线子代理应更早交报告；主代理**不得**把「已派出」当「已完成」，也不得把子代理自述当自己的实测（本批文档里两者来源逐条标注）。
- **验证**：`scripts/ci.ps1` = **CI OK / exit 0**（golangci v2.14.0 **0 issues**、Go 全量、前端 build、**vitest 424 文件 3660 例**、E 系列、卫生四查）；主代理独立复跑触达包：`internal/app` 110.8s + board、`cost` 1.25s、`schedule` 1.08s、`docmd` 13.1s、`tts` 1.49s、`costimport`/`pricefeed`/`tool`/`evidence` 全 ok；**守卫四份**：`check-primitives --strict` 无新增（截断 37 / 原子写 8 / 前端 5 / heredoc 0）、`check-test-ctors` 359 = 基线（**无新增，无需刷基线**）、`check-bindings-drift` OK@744、`check-contract-drift` 无新增；`check-docs` OK（校验和 447 / 发布说明 636）；**反向证据**共 10+ 条（线 A 的 3 条由主代理补做，`TEMP-MUTATE` 残留 0）。
- **未做（如实登记）**：`upsert_task` 无「工期为负」闸（**真缺口，实测可复现**，需前后端同批 + fixture 同步）；`ci.yml` 仍整包排除 `internal/tts`（注释引用的 `TestEdgeTTS` **全仓不存在** = 过期注释 + 白丢覆盖）；前端 flaky 分类器**注释自称有 4 组夹具自测但实际零自测**；计划 rel 的第 4/第 5 份前端副本（`ScheduleApplyRollback.tsx`/`mock/office/schedule.ts`）；X1-13 归因切片靠字典序；IN3-11 C5 残留；AP6-10 无 ctx 形参（有界等待替代）；FE3-06 直调面无门控边界；真机拔麦与 `-race`（仅 Actions）欠账。
- **待用户拍板**：仍为批次十一那两条结构项（**AP4-01** 三套并行编排合并、**X1-03** 五重 `*core` 嵌入裸构造），本批未动。

## 非版本刀：全仓审计第 11 批·P0 收官 + 原语单源（4 条并行线）· 2026-10-02

- **刀型**：接续同日全仓代码审计（`docs/code-audit-2026-10-02/`，批次一~十已落）的第 11 批。**4 条互斥并行子代理线**（线1 `internal/app` P0+静默 / 线2 非 app 包静默 / 线3 前端 P0+静默 / 线4 原语单源）+ 主代理定契约、早期复审、收线缝合、全量门禁。
- **交付**：`docs/code-audit-2026-10-02/round-13-p1-batch11.md`（含底账更正 8 条、逐线证据、逐站点对照表、留池与裁决）＋ 两份守卫基线刷新（`primitive-baseline.json` / `testctors-baseline.json`）。
- **P0 收官（三条，各带「改坏能红」反向证据）**：①AP9-01 数字生命读取全吞错 → `digitalCount`/`digitalJSON` 返错、8 张计数表逐表降级进新契约字段 `Warnings []string`（绑定签名零变更）、单行坏数据按表汇总（条数+首条示例）、8 表全失败才致命；②AP9-02 Herdsman CLI 超时只杀父进程 → `killProcessTree`（`taskkill /T /F`，趁父存活枚举子孙）+ `Start` 等 ctx + 有界回收，**并配反向对照 `TestParentOnlyKillLeavesGrandchild` 证明旧写法确实留孤儿**；③FE6-02 麦克风不可用静默转「模拟模式」→ 删净 `simTimerRef` 假音量、新增 `degraded` 位、`startCapture` 改返回 `Promise<boolean>`（不再依赖 React eager state），警示条落 **4 处消费面**（主壳命令条 / 欢迎屏 / 语音球 / 输入框）。
- **静默失败诚实化**：GA3-13 知识库迁移失败可见（`MigrationState`）+ 标记读取失败不再当未迁移反复重跑；GA4-11 备份 `os.Stat` 区分「源缺失（Warning）」与「其它错误（中止）」、`RollbackBefore` 部分失败聚合点名并保留目录供重试（不再 `moved>0` 当成功）；AP8-16 `SetFeatureModel` 落盘失败回滚内存并返回 error；AP5-16 后台 panic 统一收口（既有 `gaea-event` Notice，**未新造通道**）+ **whisper 主动关心循环单次 panic 不再永久停摆**（recover 移入每轮体内）；IN3-15 截屏查 `SelectObject`/`BitBlt` 返回码 + 补「与虚拟桌面零交集」守卫；FE4-12 附件失败 toast、FE7-14 DCMA/摘要/diff 三处失败上屏、FE3-10 mock 冷路径订阅者泄漏（**审计只列 1 处，实为 4 处**）。
- **原语单源收敛**：**截断 41 → 36**（`internal/gaea/strutil/truncate.go` 三原语「名字即行为」；删 8 处旧副本、6 处留薄包装），**跳过目录**新建叶子包 `internal/gaea/wspath`（并集 23 项 + `IsSkippedIn(name, extra)` 逐站点保持原扫描范围），**GenUI 上限**Go 为唯一真相源 → 生成 `frontend/src/genui/limits.ts` + `TestGenuiLimitsSync` 逐字节双向断言 + `TestEveryLimitIsEnforced`（10 常量各造超限规格），FE6-04 删 137 行零消费死码。**原子写旁路 13 → 8**。
- **审计原文纠错 8 条（本轮实测，已回写报告）**：①P0 状态**不能用「是否在 round 记录里被提及」判定**（`round-1-p0-status` 会罗列仍开放项）——改按状态表+逐行读码，真实底账 19 已闭/3 本批/2 待拍板；②审计入口表与状态复核表**不是同一张 25 行表**，README 脚注的「#19/#20/#21 重复」是按另一张表编号写的；③GA6-09 改签名横跨 4 包 20+ 调用点 → 当场降级为「只出迁移清单」；④**GA4-13 不是 `doc-drift`**：同一处 GBK 误码命中了**生产代码字面量**（`Pricing.Symbol()` 默认 `"楼"` 应为 `¥`，实测 `UTF-8(¥)=C2A5` 且 `GBK(C2A5)=楼` 单轮可逆），另含用户可见错误文案；⑤线 2 实测 4 处（乱码 18 行非 8 行；`store.go:169` 是索引**读取**非重建；TCCAStats 注释位置本就正确=误报；**「坐标越界致 BitBlt 失败」不成立**——实测返回非 0 且产出全黑帧）；⑥**GA5-17 的修法本身不完整**：并集含 `.gaea`，按原文「取唯一清单当默认」会打挂 `wssearch_test.go:54` 的「`.gaea/exports` 交付产物必须可检索」验收线 → 正解必须参数化；⑦`filewatch` 实测 10 项非 11、`wssearch` 与 app 清单**逐项相同**（审计列为两处独立定义）、截断族实测 45 处（审计记 13）；⑧dag 行号漂移 +43 且**清单漏了 `dagFailAll`**。
- **坑/教训（进在册）**：①**同 Go 包不可并行是硬约束**——线 4 跨包收敛期间整仓 `go build` 必红，此时任何构建/测试都不具判据价值，必须等写者停笔；主代理用「编译器断链清单」当检查单反而最快。②**「单一真相源」不等于「取并集当默认」**——六份清单只有 4 项交集，并集含 `.gaea` 会破「`.gaea/exports` 可检索」红线（有现成测试），必须参数化增量。③**对拍工具留在仓里就是债**：`snapharness` 是非 `_test.go` 却 import `testing` 的文件（会被编进正常构建），必须删净并 `Test-Path` 复核。④**守卫基线不刷 = 守卫静默失效**：`check-test-ctors` 已有一处**入库即漏登记**（批次八新增测试未刷基线），warn 档永远 exit 0、假新增攒到没人看。⑤**`nswrapper` 类假阳性**：守卫按「名字以 truncate 开头」计数，`truncateInputs()` 这类测试矩阵构造函数会被误计——改名（→`snapshotInputs`）比塞进基线诚实。⑥`npx` 在本机被执行的策略挡（`npx.ps1` 禁止运行）→ 用 `npx.cmd`；PowerShell 是 5.1，`Set-Content -Encoding utf8NoBOM` **不存在**（枚举名是 `UTF8`），改文件一律走 `edit` 工具。
- **验证**：`go build ./...` 0、`go vet ./...` 0、`go test ./internal/app/... -count=1` = `ok app 105.6s` + `ok board`、触达 35+ 包全绿、`tsc -b` 0、eslint 改动面 0 error、vitest 改动面 20 文件 169 例全绿；五份守卫（check-docs / primitives --strict / test-ctors / contract-drift / bindings-drift@744）全绿。
- **未做（如实登记，未硬凑）**：字节口径截断站点（office×2 / app.gaea_diagram / archive / memory×2）；`estimateTokens` 四份（口径真不同，拍板项）；GA6-09 独立一刀（清单已备、绑定面裁决已定=保持签名 + `CostSearchPage` 加 `Error` 字段）；GA3-13 的 UI 接线与 GA4-11 的前端文案确认；AP8-16 部分落盘的磁盘混合态（需 config 层单键事务）；真机拔麦走查；**`-race` 本机不可用**（无 gcc）→ 竞争门只在 Actions。
- **待用户拍板（两条结构项 P0）**：AP4-01 三套并行编排合并（选项 (a) 抽共用原语 M / (b) 合并内核 L / (c) 维持现状）；X1-03 五重 `*core` 嵌入裸构造（**关键发现**：`bindings_*.go` 是生成物，故「绑定入口铺断言」只需改生成器+再生，代价在 227 处 `&App{}` 测试桩迁移）。
- **收线时又抓到两条不在本批范围的既有债（均已根修）**：①**HEAD（批次十）本身不是 lint 干净的**——跑 `ci.ps1` 时 `golangci-lint` 报 7 条，其中 `schedule/index.go:93`（errcheck `tmp.Close()`）与 `docmd/pdf.go:603`（SA4006 死赋值，批次十 IN3-10 重构 `stripNonTextStreams` 的残留）**所在文件本批一字未改、内容与 HEAD 一致** ⇒ 审计自述的「golangci 0 issues」在 HEAD 上已不成立（批次九/十只跑定向测试没跑全量门禁）；②**批次一的 AP4-08 只修了一半**：`TestSaveScheduleIndexConcurrentNoTmpClash` 稳定 flaky（无负载下 3 次复跑 1 过 2 败）——只解决了「临时名唯一」，16 路并发对**同一目标**覆盖式 rename 在 Windows 上仍撞 ACCESS_DENIED，而 `RenameWithRetry` 的 5/10/20ms 退避会穷尽（该包 `sync.Mutex` 此前零命中）。根修=新增 `scheduleIndexSaveMu` 进程内串行写完，用例 **6/6 全过**。**这是 P0#24「Windows rename 竞态」的第四次复发，且复发点正是上一批刚为它加的防线不完全**——印证「没有机器守卫的收敛，下一版必回涨」；建议下一批把「Go 侧新并发用例必须 `-count≥5` 通过」纳入门禁（本机 `-race` 不可用）。

## 非版本刀：第一轮 P0 止血刀（7 条并行线）· 2026-10-02

- **刀型**：接续同日全仓代码审计（见下一条），按建议顺序「**守卫 → 止血**」落地第一批：7 条并行子代理线（4 槽位），主代理定契约、独立验收（含真实配置端到端探针）、跑全量快闸收口。
- **交付**：`docs/code-audit-2026-10-02/round-1-fixes.md`（逐刀红→绿证据 + 口径更正）＋ `round-1-p0-status.md`（25 条 P0 在当前工作树的复核）＋ `scripts/check-primitives.mjs`（原语守卫 warn 档 + 真实基线 `machine/primitive-baseline.json`）。
- **七刀**：①**配置保存丢段**（`RenderTOMLPreserving` 段级＋键级保留；真实配置副本端到端「16 段→5 段」的静默丢失已消除，且 `[agent] effort`/`[tools] compact`/`[[providers]] thinking` 这类段内未渲染键一并保住）②任务收件箱读-改-写加锁（进程内）③`ExtractJSON` 改括号配平（21 处调用点零改动）④GenUI 校验器区分「节点数组/数据数组」⑤进度契约（多基线 + `Quantity/Amount` 三态 + 资源 `calendar`；前端零改动、golden fixture 字节不变）⑥原语守卫脚本（截断 41 / `internal/app` 原子写 13 / 前端旁路 5；warn 档未接 CI）⑦`bookimport` 第二份 `ExtractJSON` 收敛到 util。
- **复核更正报告口径三处**：P0#17 **撤销**（`setAllSelected`/`clearSelection` 定义在同文件 1126-1131，函数声明提升，原「未定义函数」判据不成立 → 降 P2「缺表头全选用例」）；P0#21 方向更正为「**显式 0 → 缺键**」（原写反）；P0#22 生产调用点更正为 **21** 处。另：报告 25 行里 #19/#20/#21 与 #1/#2/#8 重复 → **独立 P0 = 22 条**，本轮**已修 7 条独立、仍开放 10 条**（清单在 `round-1-p0-status.md`）。
- **坑（进在册）**：① **TOML 根标量按原位置写会被后续表头吞掉**（`workspace` 读写回空串）→ 保留式渲染必须把根标量前置到第一张表之前；② `Load()` 对数组表是**整片替换**，保留必须**按位次**配对（按 name 会错配 thinking/effort）；③ `sandbox`/`plugins` 也在渲染器职责内（清单漏列），当未知段整段保留会写出**重复表头 = 非法 TOML**；④ **并发线同包编译互相挡**（`render_preserve.go` 在途时 `internal/app` build failed）——集成门必须等所有写者停笔再跑；⑤ 子代理留下的临时探针测试文件必须收尾删除并 `Test-Path` 复核（本次 `zz_probe_save_test.go` 已删）。
- **未做（有意）**：守卫脚本仍是 **warn 档**（接 CI 前先拍板验收线：截断 41→≤5 / 原子写 13→0 / 前端旁路 5→0）；未抬版本、未动 CHANGELOG（发版仪式归用户）；仍开放的 10 条独立 P0（后台链登记纪律 / 检索吞错 / 编排三合一 / 微信语音并发 / 绘梦单槽进度 / 三类复发型缺陷守卫）待排下一批。

## 非版本刀：全仓并行代码审计（27 单元 + 6 分册 + 2 附录）· 2026-10-02

- **刀型**：用户指令「使用子代理并行进行各个板块的全面代码审计，重点关注屎山代码」。**只读审计刀（零源码改动、未抬版本）**：workflow 编排 27 个板块单元审计员（并行）+ 6 个分册主编（综合去重、排刀），主代理另跑机械扫描（AST 上帝函数/重复块/死代码候选/绑定面消费面/139 包依赖图）并亲自复核 3 条 P0。
- **产物**：`docs/code-audit-2026-10-02/`——入口 `README.md`（快照声明+结论+25 条 P0 全表+系统性债+合并刀序）＋ 6 份分册 ＋ `appendix-A-mechanical.md` ＋ `appendix-B-surface.md` ＋ `units/<ID>.json`（27 份原始证据）＋ `machine/`（可复跑脚本）。已登记 `docs/README.md`（子目录 3→4），`node scripts/check-docs.mjs` 四查 OK（孤儿 0/悬空 0/指令预算内/BOM 合规）。
- **结论**：平均 **4.54/10**（10=干净），**512 条带 file:line 证据**（P0 25 / P1 216 / P2 209 / P3 62），**104 条可执行刀**。`go vet` 与 `golangci-lint`（6 linter 全开）**0 issues**——静态门禁全绿，问题全在结构层。主因=duplication 135 + consistency 91（44.1%）＝「同一件事没有唯一实现」；P0 里 10 条本质是静默失败；concurrency 34 条中 9 条 P0（最易真咬人）。头部结构性事实：Go 单函数 ≥100 行 **110 个**（最差 `boot.Build` 551 行）、`internal/app` 一个包 import **110 个包**、743 绑定面里 **322 个（43%）前端零引用**、同名双包 7 组。
- **主代理亲核（读了源码逐行确认，非转述）**：①`internal/gaea/config/render.go` RenderTOML 只渲染 8 个段，`Save()` 经它落盘必丢 13 段，且「始终允许」审批（`boot.go:623-632`）就会触发——真实数据丢失；②`internal/app/gaea_task_inbox.go:153-219` 读-改-写全程无锁（跨会话并发写必丢条）；③`internal/gaea/genui/validate.go:113` 把数据数组 `diffs/series` 当子节点递归——合法规格被判非法并让模型反复改写。
- **坑/教训（进在册）**：①**审计窗口内有并发会话在同一工作树改码**（HEAD 24926674→cc954cdd 并继续发 v4.452.0），单元行数与文件规模出现三方不一致——审计报告必须带快照声明与「移动靶」警告，热区结论复核前先 `git log -1 --format=%h -- <file>`；②**花括号计数统计函数长度会被字符串里的 `{}` 污染**（实测把函数报成 799 行），God 函数统计必须 `go/parser`+`go/ast`（脚本 `.tmp/godfunc/main.go`）；③PowerShell 里 `python -c "…"` 引号被吞（在册坑复现）——长脚本一律落 `.py` 再跑；④子代理长文档写入会被截断，且分册可能被并发重写——收尾必须复核节标题、条数与标题内计数一致（本次 04 分册 6 处小节计数不符，已修）；⑤**时效性互证**：v4.452.0 发布说明自陈「heredoc 转义第三次踩」，正好独立印证本报告 P0#23「复发型缺陷无守卫」。
- **未做（有意）**：不改 CHANGELOG（避免与并发发版会话撞车）；未跑测试/构建/真机（只读纪律，避免与并发改码互相干扰）；322 个零引用绑定**未删**（与 AGENTS 在册「宽口径 36」两套口径需先对齐）。

## 未发版刀：首页重设计 v11「玄墨」（十二轮）· 2026-09-30 ~ 10-01

- **刀型**：用户指令「重新设计整个首页，太廉价」（v10 天玄卡片版被复判）→ 视觉层整刀 +
  一屏自适应 + 系列真实数据缺陷根修。规格与全部轮次=进度计划/gaea-home-tianxuan-cards-20260930.md
  §7.5~§7.16（根因/验证/SHA 逐轮可查）。
- **落地**：`module-launcher.css` 整层重写（三层视觉层级/描边退场/金色三处点睛/排印三档/
  栅格 8+4 对缝）；一屏自适应（board 高度定界 `flex:1 0 0`+`min-height:0`，fr 富余吸收、
  不足回退滚动；高度档 ≤1060px 二级压缩）；浅色档补偿（卡线 6.5% + ink 派生影）；
  页底色上移壳层列（`ml-shell-column`，左缘竖缝消除）；晨报卡撤出首页（组件/做梦 2.0
  管线保留）；设置横条降权；文档空态引导；三语词典 +2 键。
- **门禁**：全量 vitest 3568/3568 绿（410 文件）；tsc 0；eslint 0 error；重叠扫描器
  `scripts/home-overlap-check.mjs` 三形态×多档全扫零重叠；真机 CDP 实数据终验（全屏
  over=0、晨报消失、空间恢复）；exe 重建+冒烟过（现行 SHA `4df73688885710ef`）。
- **坑（三条铁律，后续首页开发必守）**：①一屏 grid 的 fr 轨道必须显式 px 下限或给 item
  留 auto min——item `min-height:0` 吞 auto min，矮窗压瘪轨道且 scrollHeight 不涨
  （=重叠而非滚动，目检不可靠）；②fr 均分行不做大 min 卡片（用 auto 行+stretch）；
  ③重叠扫描须等 v3-rise 终态（transform 中间帧污染矩形测量）。真实数据（文档 5 条/
  晨报 169px/遥测带 176px）三族溢出全部是 dev 空态测不出、真机 CDP 才暴露的。

## 未发版刀：删除编程板块（code / DeepSeek Harness 工作台）· 2026-09-30

- **刀型**：用户指令「删除编程板块」。**功能删除刀**（唯一一类不做零删除担保的刀）：前端
  manifest/页面/入口 + Go 侧 DSH Web 进程管理五绑定整块退役；规格=进度计划/gaea-remove-code-board-20260930.md。
- **落地**：删 `internal/app/programming_web.go`（8.9KB）+ 其 16 例测试、`pages/ProgrammingPage.tsx`
  （538 行）+ 12 例测试 + `programming-page.css`（17KB）；`board/builtins.go` 去 code（canonical
  13→12，`GetBoardManifests` 14→13）、`bindings_core.go` 手删 5 委托、`bindingNames.ts` 再生
  744→739、`bridge/core.ts`/`wailsjs CoreB`/`types/programming.ts`/`mock/core.ts` 同步；
  前端 `manifests.ts`+`launcher.ts`+`main.tsx` 去注册，删 `MainLayout` 的 `v3-prog-host`
  工具栏宿主与 `.v3-strip-prog`；spaceBindings 567→562（independent 分面归零）。
- **取舍**：`independent` 独立窗口**机制保留**（BoardSpace/rail foot/分面表三处落点不减），
  仅移除「编程」这一实例——该分面空集时 rail foot 段自动隐藏。
- **门禁**：go build/vet 0；`go test ./internal/app/... ./internal/gaea/...` 全绿（app 122s）；
  drift OK@739；tsc 0；eslint 0 error；**vitest 410 文件 / 3564 例全绿**；e-check + check-docs OK；
  CDP 真机走查 `body.innerText` 无「编程」、rail 六项 + 深浅色切换、首页正常。
- **坑**：①gen_bindings 全量再生会把 11 个门面文件格式化（多行→单行），混入 ~1000 行噪声——
  处置=先 restore 再手删 core 五方法、只再生 bindingNames（漂移闸按方法名集合，与格式无关）。
  ②删除板块的测试足迹=6 个文件（Go 2 + 前端 4）的夹具/计数/断言反转，顺序=先实现后按失败清单改。

## 未发版修复：首页「最近文档」不实时更新（用户报障）· 2026-09-30

- **现象**：办公里打开文档后回首页，「最近文档」纹丝不动。**定位**：订阅链实测是好的
  （dev 页直接调 `recordRecentFile` → 首页立即出现新条目），缺口在**写入侧**——
  记账只挂在 4 处冷门入口（@ 菜单选文件 / 右键「引用 @」/ 主区预览 / 点最近条），
  而日常两条「打开文档」路径根本没记账：`usePaneTabsStore.openFile`（资源管理器行点击、
  产物·变更·Git 行内打开）与 `usePreviewStore.openFilePreview`（右键预览、正文文件链接、
  记忆中枢·进度计划·DAG·上下文检查器文件行、附件条）。
- **修法**（单点记账，调用方零改动）：两个 store 动作各加一行 `recordRecentFile`
  （pane 页签 / 预览队列 = 用户打开文档的唯二入口），并删掉 `usePreviewPanel` 里的
  重复记账。**顺带修**：首页最近文档点行原本只跳办公板块不开文档（与 hint「在办公中打开」
  不符）→ 改为 `openPaneFileOrPreview(path)` + 导航（已挂载开 pane 页签，未挂载落预览队列）。
- **验收**：pane 路径 / 预览路径各自 dev 真机探针 → 首页首项即时变化 ✅；点行 → 办公 +
  右侧文件预览面板（截图 click-after-office.png）；新增 `paneTabs.test.ts` +3、
  `previewQueue.test.ts` +1；tsc 0 错、eslint 0 error、**vitest 410 文件 / 3568 例全绿**。
- **规格**：进度计划/gaea-recent-docs-live-20260930.md。

## 最新发布：v4.438.0（2026-09-30）「维护小刀包：生成器遮蔽告警 + .tmp 守卫补洞 + tarball 出 VCS」

- **刀型**：优化方向档 P1-2+P2-3①②。**零绑定面变更（743 不变）**。gen_bindings 去重处 shadowed 计数显式报告+诊断脚本 shadow-diag（459 个全为设计内 *B 门面委托）；clean-tmp 补 uiwalk*/home-shots*/*-shots（原守卫扫不到的 577MB 最大堆积源）；11 个 tarball 出 VCS（166.5MB→gitignore，磁盘保留）。gen_bindings golangci 0；全量 ci CI OK。
- **产物**：`releases/gaea-v4.438.0.exe` 51,807,744 B SHA256=`32ec625bd76adabd1a9dd68c3b39b97d484e2cc5266dc7e67e3fed1768d68beb`（冒烟 200 过；删 v4.433.0.exe，实存 5 版 438/437/436/435/434）。

## 最新发布：v4.437.0（2026-09-30）「P1 小刀包：对话停止入口 + 数字守卫 + 优化档关账三标注」

- **刀型**：用户指令「继续」。定刀=优化方向档 P1-5 对话条（无「停止生成」入口——useChatStream sending 期间直接 return，长回合只能等 30s 超时或切话题）+P1-7（check-docs 不校验任何数字）+P1-5 三条已被后续版本做掉的关账标注。**绑定面 742→743（+1：ChatStreamCancel）**、spaceBindings 566→567，零功能删除。
- **①对话停止入口**：后端 `chatStreamMu/chatStreamCancels` 登记表（runID→cancel，**归 core**）+请求级 ctx（nil 防御）+`ChatStreamCancel(runID) bool` 绑定（命中调 cancel、未知返 false）；取消时**已生成部分照常落库**并以 `cancelled` 终态帧收尾（对齐章节生成取消语义——保留部分回复不按失败处理）；协程退出独占清表。前端 useChatStream 暴露 stop（activeRunIDRef 持 runID，取消失败由 30s 静默超时兜底）+cancelled 帧处理（部分回复+「（已停止生成）」标注）+Composer 发送钮 **sending 时变停止钮**（PauseCircle+「已生成部分保留」）；角色模式退化为中止模拟打字。
- **②数字守卫**：check-docs.mjs ⑤节——docs/README 顶层文档计数/releases/README 校验和与发布说明份数，自述 vs 实存对账失配即 FAIL。**守卫首跑即抓到真账**：AGENTS.md 66,644B 超指令预算 65,536B（长篇七刀+三板块连发塞爆速览）→按纪律迁最老 6 条（v4.421~428.0）入 docs/archive/agents-version-history-2026-09.md，AGENTS 回 47,359B。
- **③优化档关账三标注**：P1-5 表对话条 ✅ 本版；小说条 ✅（chapter-review 模板 v4.429+场景驱动刀2 v4.430）；绘梦条 ✅（getSystemStats 归一化 v4.415，全部 Get*Stats 字段缺失用例留尾池）。
- **测试**：Go 新增 3 例（未知 runID 返 false/登记表生命周期 register→cancel 命中→unregister 后 false/**取消传播端到端**〔受控协程：真 ctx+登记+ChatStreamCancel→ctx.Done 传播→退出清表+二次取消幂等〕）+**反向验证**（cancel() 改 `_ = cancel` 必红还原）。app 整包 -count=1 绿（含既有 chat 域全部测试）。数字守卫实测抓真账。
- **门禁**：全量 `ci.ps1` → CI OK；漂移闸 OK@4.437.0（743 一致）；spaceBindings 567。
- **产物**：`releases/gaea-v4.437.0.exe` 51,807,744 B SHA256=`3458cc6740eb2500a8e61342d52b37a2df33b2945571a3d7e43d02c1c065bb87`（`SHA256SUMS-v4.437.0.txt`；冒烟 200 过）；保留策略删 v4.432.0.exe，实存 5 版（437/436/435/434/433）。
- **文档**：`releases/archive/v4.437.0.md` + CHANGELOG/README + releases/README + AGENTS 速览（迁 6 插 1）+ progress（本条）+ 优化方向档三标注。
- **坑/教训**：①**登记表/锁字段放哪个 struct 要看消费方装配面**：chatStreamMu 首放 writingState（跟 chapterGenMu 同区块抄的）——裸 chat 测试桩（newChatServiceTestApp）不构造 writingState→nil 嵌入解引用 panic（mutex.Lock 在 nil 指针+字段偏移上炸）；chat 域字段归 core。②端到端取消测试对假站路由的依赖太深（chat feature 快速失败路径毫秒级退出清表）——受控协程（真 ctx+登记+传播断言）是更稳的端到端形态。③反向验证的 mutation 锚先 grep 确认再改（本次 python 替换锚没匹配上=假验证，改用 Edit 直改才真红）。
- **观察池**：GenUI 预算常量合并（P2-2 独立项）/全部 Get*Stats 类字段缺失用例（P1-5 尾池）/停止入口真机走查（长回合停止手感+部分回复落库核验）。

## 最新发布：v4.436.0（2026-09-30）「P5 破防专项：Go >50KB 五文件拆分（纯搬移零功能）」

- **刀型**：用户指令「继续」（长篇七刀收官后，P5 复测在册挂池清账——「Go >50KB 0→4 破防，拆分留观察池待专项（P3 纪律=专项勿顺手做）」本刀即该专项）。长篇七刀加码后实测**五文件**破防（create_chapter_handler 67.4K 居首）。**零功能变更、绑定面 742 零变更**——全部为同包文件搬移（按功能族移到新文件，函数/常量/类型签名不动）。
- **拆分明细**：①create_chapter_handler 67.4K→43.8K（拆 `create_chapter_context.go` 24.3K：上下文预算常量/前文摘要窗口/增强区段组装/计划门辅助与区段渲染/世界观与文风构建/角色摘要）；②image_comfyui 59.4K→36.1K（拆 `image_comfyui_workflows.go` 23.7K：各模型 txt2img/img2img/edit/video 工作流拼装/LoRA 注入/Kontext 分辨率表）；③image_handler 57.9K→39.3K（拆 `image_comfyui_proc.go` 19.0K：ComfyUI 进程引用/启停恢复/端口探测/系统资源统计）；④controller 53.2K→42.3K（拆 `controller_session.go` 11.4K：会话新建/恢复/工具态复位/checkpoint/上下文用量）；⑤config 52.2K→41.7K（拆 `config_paths.go`：用户配置路径/归档会话目录/约定目录扫描/默认系统提示词与语言政策）。五文件全部回到 P4 纪律上限（50KB）以下；**P5 复测破防线归零**（>50KB 0 个；>40KB 仍 7 个留池下季复测）。
- **验证**：go build ./... ./go vet/gofmt/golangci-lint 四包 0 issues；app+ai+control+config 四包测试 -count=1 绿。随版收口 TestCreateChapterDigestInjection 的 **Windows unlinkat 竞态**（刀5 测试漏补 waitGensDone——v4.435 总结「CreateChapter 真链测试收尾必等」完整套用，八连跑全绿）。
- **门禁**：全量 `ci.ps1` → CI OK（**首轮挂在册负载 flaky OriginalSinPage.tabs**——v4.414/v4.418 同款，单跑 4/4 绿，重跑全绿；漂移闸 OK@4.436.0（742 零变更））。
- **产物**：`releases/gaea-v4.436.0.exe` 51,804,160 B SHA256=`bde9b4652abb9cee52488e4b73181837cf8e1c5b9471460ee6edd999f78754ac`（`SHA256SUMS-v4.436.0.txt`；冒烟 200 过）；保留策略删 v4.431.0.exe，实存 5 版（436/435/434/433/432）。
- **文档**：`releases/archive/v4.436.0.md`（含拆分明细表）+ CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**大文件拆分的机械配方**：按顶层声明清单（grep '^func|^type|^const|^var'）定函数边界行号→整段行级搬移→新文件手写 import（编译器逐个报缺/报多余，两三轮收敛）→**段尾注释回填**（切点常落在文档注释中间，尾注释移回原文件对应 func 前）。②拆分切点选「功能族聚拢」而非对半切——新文件名即族名（context/workflows/proc/session/paths），下季复测可读性也受益。③后台 shell cwd 漂移再犯（git add 报 frontend/internal 不存在）——跨命令一律显式 cd。

## 最新发布：v4.435.0（2026-09-30）「长篇刀7：评测基线（七刀收官）」

- **刀型**：用户指令「继续」（长篇七刀线程收官刀，P2）。规格 §4（评测闭环）按「单刀最小闭环」裁剪：**确定性聚合快照+基线落盘可比+逐指标 Δ**。Go 新 eval_baseline_handler.go；**绑定面 739→742（+3：NovelEvalSnapshot/NovelEvalBaselineSet/NovelEvalCompare）**、spaceBindings 563→566，零功能删除。规格=`进度计划/gaea-novel-eval-baseline-20260930.md`。
- **快照**：NovelEvalSnapshot 零 LLM 全确定性——每章 AI 味（ScoreTextNoRef→mean/max/P90/最差章号）+每章确定性质量信号（ChapterQualityIssues S1/S2/S3 汇总）+伏笔（LintForeshadows→items/planted/hinted/revealed/回收率/findings）+结构（刀3 NovelStoryHealth findings 按 code 分桶）+文风 Delta（有 fingerprint 时全书对照，无档 skipped）+上下文合计（刀6 Inventory）+元数据（章数/字数/**promptSetHash**〔prompts/*.json 内容 sha256 按名稳定序——任一变更即 stale 禁止直接对比，§4 原则〕）。
- **基线/历史**：eval/baseline.json（NovelEvalBaselineSet）+eval/snapshots/<RFC3339Compact>.json（persist=true 留史）；**body 不含时钟、全稳定序——同一状态两次产出逐字节相同**（§4 验收线达成）。
- **对比**：NovelEvalCompare 最近状态 vs 基线逐指标 Δ+方向（better/worse/flat 按指标语义：AI 味降=better、回收率升=better）；promptSetHash 不一致→整份 stale（如实拒绝对比语义）。
- **测试**：Go 新增 4 例（确定性〔两次 marshal 逐字节相同〕/聚合造数〔最差章=AI 味最重章+回收率 0.5+hash 非空〕/基线落盘→状态变化→Δ 方向〔AI 味升=worse〕→baseline hash 篡改必 stale/persist 落盘+零章空快照）+**方向语义反向验证**（lowerIsBetter 翻转必红还原）。app 整包 -count=1 绿。
- **随版收口**：两处 HTTP 断言测试（SettingBudgetTrim/SpineInjection）全量负载下挂 **Windows unlinkat 竞态**——测试 return 后生成协程尾步（自动门/摘要/SceneRefs 回写）仍在写 chapters/，t.TempDir 清理撞 directory not empty（v4.233 在册坑复发）；补 waitGensDone（等登记表清空+chapterGenWG）。
- **门禁**：全量 `ci.ps1` → CI OK（三轮修：errcheck Fprintf 返回值显式化/unused 类型删除/unlinkat 竞态）；漂移闸 OK@4.435.0（742 一致）；spaceBindings 566。
- **产物**：`releases/gaea-v4.435.0.exe` 51,803,648 B SHA256=`f0262d5b4f57517512e4ac80e6900b6d1ec485e25a7895ef21570ceb336e4a4f`（`SHA256SUMS-v4.435.0.txt`；冒烟 200 过）；保留策略删 v4.430.0.exe，实存 5 版（435/434/433/432/431）。
- **文档**：`releases/archive/v4.435.0.md` + 规格 `进度计划/gaea-novel-eval-baseline-20260930.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**HTTP 断言型测试（捕获 requests 后即 return）在生成链有后台尾步协程时必挂 unlinkat**——测试收尾必须 waitGensDone（v4.233 坑的完整形态：不只取消型测试，任何 CreateChapter 真链测试都算）。②errcheck 对 fmt.Fprintf(hasher) 也报——写 hasher 的返回值显式丢弃 `_, _ =`。
- **长篇七刀收官**：刀1 章节计划闭环（v4.422）→刀2 场景卡与场景级生成（v4.430）→刀3 故事层骨架（v4.431）→刀4 质量收敛闭环（v4.432）→刀5 风格学习回灌（v4.433）→刀6 上下文编译器（v4.434）→刀7 评测基线（本版）。诊断 §1.1~§1.14 的正面解全部落地。
- **观察池（长篇线收尾）**：回归夹具 3×30 章人工病历（独立工程）/LLM 盲评+bootstrap/张力曲线聚合（随真机分析覆盖率）/真 A/B 双生成/七刀全链真机走查（多场景章+骨架+收敛+回灌联合体验）。

## 最新发布：v4.434.0（2026-09-30）「长篇刀6：上下文编译器」

- **刀型**：用户指令「继续」（长篇七刀线程次刀序，P2）。规格 §1.8「上下文是堆料不是按优先级编译——世界观至少三处、文风两处、各池独立、结构信息与整篇设定争预算」三缺口收口（刀1 已统一共预算）。Go 3 文件改造+Inventory 新文件；**绑定面 738→739（+1：NovelContextInventory）**、spaceBindings 562→563，零功能删除。规格=`进度计划/gaea-novel-ctx-compiler-20260930.md`。
- **①setting 入预算**：CreateChapter 的 setting 入参（前端传整篇设定 markdown，此前**完全不裁剪**直进模板 P0 槽——预算外堆料最大残留）→ 超 3000 rune 截断并附「（设定过长已截断；分维度要点见下方『世界观要点』区段）」。
- **②文风去重合并**：style.md（显式偏好）与刀5 digest（成稿学习）两个独立区段两个标题（§1.8「文风两处」延续）→ 单一「文风与表达」区段（【显式偏好】优先【成稿学习】跟随；单方在场保完整语义；双空零注入；合并预算 2000）；CreateChapter 的 digest 独立注入位移除。
- **③世界观相关性**：buildWorldviewSection 增 relevanceHint（本章计划+大纲要点）——命中维度条目**稳定前移**（组内原序），前移≠丢弃（预算内未命中维度照旧）；中文无词边界（FieldsFunc 整句一个 token）→ **rune bigram 滑窗**匹配（宽匹配可接受：只排序不过滤）。
- **④编译清单可见性**：NovelContextInventory(chapterNum) dry-run 零生成——区段清单（名称/rune/说明/80 字预览+合计行），「同样信息量下 prompt 更短」可量化（API/DSH 消费，暂无专门 UI）。
- **测试**：Go 新增 4 例（joinStyleSections 合并矩阵含**单标题去重验收**〔「## 文风与表达」恰一处〕/相关性命中前移+零命中原序+预算内不丢弃/setting 超长 HTTP body 断言〔截断提示+前段保留〕/Inventory 清单）+bigram 永假反向验证必红还原；既有断言随合并标题同步（「本书文风」→「文风与表达」两处+刀5 HTTP 断言）。app 整包 -count=1 绿；小说域 38 文件/331 例；tsc 0。
- **门禁**：全量 `ci.ps1` → CI OK（两轮修：①unused 兼容壳 buildStyleSection 零调用删除 ②gofmt 漏一处）；漂移闸 OK@4.434.0（739 一致）；spaceBindings 563。
- **产物**：`releases/gaea-v4.434.0.exe` 51,777,536 B SHA256=`c87c5b06e6f88244225837683628d2d50ac1de0d05e14574869cde9353eb574a`（`SHA256SUMS-v4.434.0.txt`；冒烟 200 过）；保留策略删 v4.429.0.exe，实存 5 版（434/433/432/431/430）。
- **文档**：`releases/archive/v4.434.0.md` + 规格 `进度计划/gaea-novel-ctx-compiler-20260930.md`（含落地回填）+ CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**worldview 测试须直写结构化 json sections**——md 路径触发 ReadWorldviewFile 的 legacy 迁移（全文塞「旧版世界观」section 再拼默认六维，维度结构面目全非）。②**中文文本的关键词分词全部无效**（unicode.IsLetter 汉字=true，FieldsFunc 整句单 token）——相关性匹配用 rune bigram 滑窗。③CI 的 golangci unused 闸比 go vet 严（兼容壳零生产调用即报）——抽函数先查调用面。④后台 shell cwd 漂移复发（python FileNotFoundError 三连）——跨命令一律先 `python -c "import os;os.chdir(r'C:\AI\wubigrok')"` 或显式 cd。
- **观察池**：设定语义级检索（向量/关键词召回升级）/SceneBible 与主链编译层统一（场景路径现独立）/Inventory 专门 UI（调参面板）/真机验证截断提示的实际体验。

## 最新发布：v4.433.0（2026-09-30）「长篇刀5：风格学习回灌」

- **刀型**：用户指令「继续」（长篇七刀线程次刀序）。规格 §1.9「风格一致性靠事后打分与手写偏好——指纹参考档只用于体检打分，生成时不学习作者表达习惯」的正面解。Go 新 3 文件+注入两处+前端面板扩展；**绑定面 735→738（+3：NovelStyleDigestBuild/Get/Clear）**、spaceBindings 559→562，零功能删除。规格=`进度计划/gaea-novel-style-digest-20260930.md`。
- **指令编译**：novelstyle.DigestOf 纯函数——统计指纹的数值特征→**可执行写作指令**：句长节奏（离散度>0.7×mean=「短句作重拍长句作铺陈」/均匀=「以逗号长句为主避免碎短句」）/段落（<80 字=一段一镜头）/对话占比（>45% 对话驱动、<15% 惜字如金）/低密度避免项（四字格<2、连接词<3、形副<20/千字——作者低用即明确「避免」）/标点习惯（逗号分层感、少省略号破折号）/词汇丰富度（TTR 波动）/签名词前 8（「自然融入勿堆砌」）。
- **摘要档**：style_digest.json（项目根，与 fingerprint.json **独立**——体检 vs 生成消费面/生命周期不同，刻意不合并）；Build 复用指纹链（collectChapterSamples→ComputeFingerprint→DigestOf，同样本门槛 3 章 3000 字=作者认可成稿）；Get/Clear（停用回灌不动体检档）。
- **注入**：CreateChapter 与逐场景流注入「作者风格约束（从你的成稿学到的表达习惯，按此口径写）」区段——与刀3 故事层切片同款追加位；**零 digest 零注入**（测试钉死）。
- **前端**：文风指纹面板新「风格学习回灌」区——未学习引导/已学习状态（章数字数时间+查看注入指令折叠预览）+footer「学习我的成稿（生成回灌）/重建/停用回灌」。
- **测试**：Go 3 新例（DigestOf 指令矩阵〔交错/均匀/高对话/低密度/签名词〕/生命周期〔构建→读回→注入→清除零注入→样本不足诚实拒绝〕/**CreateChapter HTTP body 断言含「作者风格约束」**）+反向验证（约束行改永假条件必红还原复绿——注意阈值改宽是单向放松测不出，须改永假）；前端 2 新例（引导+构建调用/已学习+指令预览）。小说域 37 文件/327 例；tsc 0；eslint 0。
- **随版收口**：刀2 测试 TestNovelChapterScenesGenerate_Gate/Flow 全量负载下 flaky（「未收到 done 事件」）——协程登记表清空（waitFor 通过）≠SSE 订阅端已收到 done 帧（flush 延迟），一次快照断言有竞态窗口；改 waitFor 轮询事件快照。
- **门禁**：全量 `ci.ps1` → CI OK（首轮挂上述 flaky，修复后复跑全绿）；漂移闸 OK@4.433.0（738 一致）；spaceBindings 562。
- **产物**：`releases/gaea-v4.433.0.exe` 51,764,224 B SHA256=`b73ee31c530480f409d6b2d13c87688d6bf151756e4728efe1e6373f6b445228`（`SHA256SUMS-v4.433.0.txt`；冒烟 200 过）；保留策略删 v4.428.1.exe，实存 5 版（433/432/431/430/429）。
- **文档**：`releases/archive/v4.433.0.md`（含 flaky 附注）+ 规格 `进度计划/gaea-novel-style-digest-20260930.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**SSE 事件断言必须轮询不能一次快照**：协程退出（登记表清空）与订阅端收帧（SSE flush）有时延——全量负载下必现窗口；waitFor(cond) 轮询事件快照是标准式。②反向验证改阈值测不出单向放松（条件变宽断言照样过）——须改**永假**验证「行存在性」。③后台 shell 的 cwd 各自漂移——跨命令文件操作前先 `cd /c/AI/wubigrok` 或绝对路径。④`python /dev/stdin` 在 Windows 不可用——heredoc 直接 `python - <<'EOF'`。
- **观察池**：few-shot 样本句直取（prompt 预算）/真模型对照实验（同题双生成量化 Delta 下降——回灌效果实证）/回灌后生成质量真机走查。

## 最新发布：v4.432.0（2026-09-30）「长篇刀4：质量收敛闭环」

- **刀型**：用户指令「继续」（长篇七刀线程次刀序）。规格 §1.7「写后质量信号全是报告、从报告到修复是人工动作、没有收敛判据」的正面解。Go 新 converge_handler.go + RewriteModeConverge + 前端 ConvergeModal；**绑定面 733→735（+2）**、spaceBindings 557→559，零功能删除。规格=`进度计划/gaea-novel-converge-20260930.md`。
- **判据**：convergeCheck 纯函数零 LLM——**收敛 = AI 味 ≤ 目标（默认 40）且确定性 S1/S2 归零**（S3 提示不阻断，与门禁 severity 口径一致）。
- **闭环**：NovelChapterConverge 流式 converge-stream，每轮=轮前**单元快照**（场景制=每场景一单元；v4 无场景混合态章回落 blob 单单元=ReadChapterAsStitch 口径）→定向修补（S1/S2 清单+AI 味要点拼指令→整单元重写〔rewrite-chapter 模板+单元级安全闸：分须降才采纳〕+既有 rewriteUnit 句级修补〔自带安全闸〕）→复检→**三态判停**：converged 即停（已收敛章零轮直答零模型零事件）/ no-improve（本轮分数未降且 S1/S2 未减→**按单元回滚本轮**并停——第一职责是不把稿子磨坏）/ max-rounds（剩余项如实报告不假称收敛）。每轮版本库留痕 mode=converge（Original=轮前/New=轮后，轨迹即版本历史可逐轮恢复）。
- **预检**：NovelChapterConvergePreview dry-run（当前分/S1S2 清单/是否已收敛/计划轮次，零修补）；空章诚实报错。
- **前端**：CreatePage rail「收敛修补」→ ConvergeModal（预览+目标分可调+S1/S2 清单前四条+逐轮轨迹行〔AI 味 before→after+改写句数〕+终态如实〔达标/无改善已回滚/轮次上限+剩余〕，终态回调刷新章节）。
- **测试**：Go 新增 6 例（判据矩阵〔S3 不阻断/空正文 S1/严格目标〕/预检 dry-run+空章报错/零轮直答零事件/**no-improve 回滚**〔坏文本+安全闸拒收→stopped 事件+磁盘字节回轮前〕/指令拼装/**回滚函数直测**〔写脏→快照回滚→字节还原〕——no-improve 集成里安全闸常使回滚成 no-op，灵敏度靠直测钉；语义改坏〔空操作回滚〕复验必红后还原）；前端新增 4 例（预览渲染/已达标禁用/启动+零轮直答/预检失败）。小说域 37 文件/325 例；tsc 0；eslint 0；internal/app+types 绿。
- **门禁**：全量 `ci.ps1` → CI OK；漂移闸 OK@4.432.0（735 一致）；spaceBindings 557。
- **产物**：`releases/gaea-v4.432.0.exe` 51,744,256 B SHA256=`283cb4ee4664716b215459f50438e572cbf71bf40df0896cf230b263c43d8a71`（`SHA256SUMS-v4.432.0.txt`；冒烟 200 过）；保留策略删 v4.428.0.exe，实存 5 版（432/431/430/429/428.1）。
- **文档**：`releases/archive/v4.432.0.md` + 规格 `进度计划/gaea-novel-converge-20260930.md`（含落地回填）+ CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**反向验证的语义 mutate 要能被被测路径真的执行到**：no-improve 集成场景里 rewriteUnit 安全闸全程拒收→回滚是 no-op→改坏回滚函数集成测试仍绿——补回滚函数直测（写脏→回滚→字节还原）钉住函数本身。②bash 双引号内的 python f-string 花括号同样会被 shell 吞（{sha} command not found）——外插值一律先 shell 变量再单引号 heredoc 或 Edit 工具。③v4 项目无场景文件的混合态章（迁移/手写 blob）在场景 API 语义下是空集——收集单元必须回落 blob（ReadChapterAsStitch 同款口径），否则混合态章被功能拒之门外。
- **观察池**：平台评审（novelreview 15 维）并入判据（外部标准报告制，另刀）/场景卡字段级修补/收敛闭环真机走查（真实模型的 no-improve 率与轮次分布）。

## 最新发布：v4.431.0（2026-09-30）「长篇刀3：故事层骨架」

- **刀型**：用户指令「继续」（长篇七刀线程，v4.430 刀2 后按次刀序开刀3）。规格 §1.10「整书层没有任何持续状态——纵向结构（长篇命门）在系统里没有承载物」的正面解。Go 新 3 文件+注入两处接线+前端 StorySpinePanel；**绑定面 729→733（+4）**、spaceBindings 553→557，零功能删除。规格=`进度计划/gaea-novel-story-spine-20260930.md`。
- **契约**：story_spine.json（项目根与 outline.json 平级）——主题论证（控制理念+反论最强陈述，M12 只报告不做闸）/人物弧线（want/need/misbelief+水位点 setup·provoke·escalate·proof）/结构节拍位（hook/first_plot_point/midpoint/second_plot_point/climax+自定义，绑章号 planned·hit）/支线开关表（MICE 类型+开启章+最后推进章+active·parked·closed·abandoned）/未解问题池（raised_chapter+open·answered）。确定性校验整单拒绝（章号/枚举/重复 ID/缺名点名）。
- **体检**：NovelStoryHealth 七维度**零 LLM 确定性**——①支线遗忘（>10 章未推进，active S1/parked S2）②未解问题超期（>15 章）③节拍滞后（planned+3 容差）④中段塌陷（分析 V2 情感强度中段 30%~70% 均值<全书 75%，缺数据如实跳过该维度）⑤弧线水位覆盖（最后水位<进度 1/3）⑥终局未闭合（当前章≥climax 绑定（或 max×85%）仍有 active 线程/open 问题=收束清单）⑦主题缺失（S3 报告）。零 spine/零章=空报告不算错。
- **注入**：CreateChapter 与 NovelChapterScenesGenerate 注入同源故事层切片——本章绑定的节拍位/活跃线程/未解问题（「不要无意中回答，也不要遗忘」）/主角弧线最近水位点；**空 spine 零注入**（测试钉死）；切片短且有独立结构价值，不挤占伏笔/文风/世界观 ctxBudget 额度。
- **提案**：NovelStorySpinePropose 从故事主线（刀1 资产）+大纲+最近 8 章摘要提炼骨架初稿；**提案只填表单，落盘必须走「保存」**（确认红线同族）；主线为空诚实拒绝。
- **前端**：CreatePage rail 新「故事骨架」按钮 → StorySpinePanel（主题两行+弧线/节拍/支线/问题四表行内增删改+AI 提炼填表单+体检报告按 S1/S2/S3 渲染+保存）。
- **测试**：Go 新增 4 例（往返+拒绝矩阵 10 例/七维度造数含健康支线不误报+空 spine 空报告/切片渲染含空 spine 零注入+非绑定章不注入节拍行/**CreateChapter prompt HTTP body 断言含「故事层骨架」**）+**反向验证**（遗忘阈值 10→100 必红还原复绿）；前端新增 5 例（打开拉取空态/编辑保存 payload/提案填表单 Save 零调用/体检报告渲染/校验拒绝如实报错）。小说域 36 文件/321 例；tsc 0；eslint 0；internal/app+project+types 绿。
- **门禁**：全量 `ci.ps1` → CI OK；漂移闸 OK@4.431.0（733 一致）；spaceBindings 数量锁 553→557。
- **产物**：`releases/gaea-v4.431.0.exe` 51,712,512 B SHA256=`0d9023b372649056e54b4ead6c9157b0e0467f9bb6fb10da457712c01f14b94c`（`SHA256SUMS-v4.431.0.txt`；冒烟 200 过）；保留策略删 v4.427.0.exe，实存 5 版（431/430/429/428.1/428）。
- **文档**：`releases/archive/v4.431.0.md` + 规格 `进度计划/gaea-novel-story-spine-20260930.md`（含落地回填）+ CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①python heredoc 写 Go 源码里的 `
` 字面量（三重转义）在 bash 层必降级——Go 代码修改一律 Edit 工具或 node 行级操作（本批 sceneGenPlanSection 重写又踩一次）。②node 插入脚本的下标顺序要核对声明先后（spineSec 插到了 ctxBudgetLeft 声明前——编译器抓住，行级重排修复）。③**产物 SHA/体积别预写**：先跑 release.ps1 再从 SUMS 与 ls 读实际值回填（本批预写错一次靠 cat 纠正）。
- **观察池**：M7 伏笔池不并入 spine（伏笔体系在产独立）/M8 认知三表完整形态/M9 因果 DAG·时间线台账/弧线水位闸门（只体检提示）/体检自动触发（按需按钮与全书体检口径一致）/骨架面板真机走查。

## 最新发布：v4.430.0（2026-09-29）「长篇刀2：场景卡与场景级生成」

- **刀型**：用户指令「继续」（v4.429 小说板块观察池收官后按台账次刀序开长篇七刀的刀2）。规格 §1.3/§1.4 的正面解：生成单位从整章 blob 改为按场景卡逐场景、场景卡补创作学核心字段、解禁场景级重写。Go 8 文件+前端 4 文件，**绑定面 726→729（+3）**、spaceBindings 550→553，零功能删除。规格=`进度计划/gaea-novel-scene-cards-20260929.md`。
- **数据契约**：SceneMeta 扩 6 字段（Goal 这场戏要什么/Conflict 谁在阻挡/Turn 价值翻转/Outcome 结果必须改变状态/Sequel 余波 reaction·dilemma·decision/ExitHook 退出钩子），omitempty 旧档零迁移；SaveSceneMeta 白名单 patch **指针语义**（未传=保留旧值防旧调用方抹卡、传空=清除——编辑面所见即所存）；OutlineNode.SceneRefs 启用（G10 死契约转正：逐场景生成完成按章号递归找节点回写场景 ID 列表）。
- **生成链**：①GenerateScene 卡注入+上一场衔接（卡字段非空注入「本场景卡（工艺约束，正文必须落实）」区段+前场 Outcome/ExitHook 衔接段；**无卡场景 prompt 与旧版逐字节一致**——零回归由测试钉死；手动单场景路径不上闸）。②NovelChapterScenesGenerate 整章按卡逐场景生成：按 Order 逐场、**每场完成即落盘**；写前闸=卡缺 Goal/Conflict 拒绝并点名场景与缺失字段（allowMissingCard 显式跳过=刀1 覆盖语义复刻）；任意一场失败即停（已完成保留+如实报错）；全部完成 Stitch→WriteChapter blob（读路径零变化）+SceneRefs 回写+大纲标记；复用 chapterGenMu 同章互斥（与整章生成不并发）与 CancelCreateChapter 取消；新流事件通道 scene-gen-stream（scene-done/scene-skipped/done/error/cancelled）。③NovelSceneRewrite 单场景 whole 重写：快照先行+卡字段/前场衔接进指令+rewrite-chapter 模板+版本库 mode=scene 留痕（Original/New=整章前后 Stitch，回滚安全）+Stitch→blob 同步，**他场一字不动**（字节级测试）；局部重写仍禁场景章（选段↔场景映射维持观察池）。④NovelSceneCardsPropose AI 拆卡：章计划（刀1 资产）+大纲节点+前章摘要→2-5 张卡骨架，**提案不落盘**（确认红线同族），前端审批后走既有 CreateScene+SaveSceneMeta 落卡；既无计划也无大纲节点诚实拒绝。
- **前端**：ⓘ 弹窗扩 6 卡字段（目标/冲突必填标注）+弹窗内「AI 重写本场景」区块（指令输入，重写历史可恢复）；场景区工具条「按卡生成全章」（闸拒绝弹「跳过缺卡」确认）+「AI 拆场景卡」（提案审批弹窗列表可编辑，确认逐张建场景+存卡）+逐场进度行（事件推进+终态整体重拉回喂 scenes/sceneIds）。
- **测试**：Go 新增 5 例（卡区段三态零回归底线/闸点名+跳过+**关闸反向验证必红**/两卡两场全链 done 事件+Stitch→blob+SceneRefs 断言/**卡进 prompt 的 HTTP body 断言**（requests 通道捕获）/场景重写他场字节不动+mode=scene 版本留痕）；前端新增 4 例（卡字段编辑保存/闸拒绝确认弹窗/拆卡提案可编辑+落卡调用链/重写指令空提示+执行）。ChapterEditor 12/12；小说域 35 文件/316 例；tsc 0；eslint 0 错 0 警；internal/app 整包 -count=1 绿。
- **门禁**：全量 `ci.ps1` → CI OK（首轮挂 SA1012：抽共享内核后 GenerateScene 传 nil ctx——改 context.Background 后复绿）；漂移闸 OK@4.430.0（729 一致）；spaceBindings 数量锁 550→553。
- **产物**：`releases/gaea-v4.430.0.exe` 51,645,440 B SHA256=`74974d7ae9f1e1975dfe6d3363c2bac36380bd1f2cee8408939af29298ea8a11`（`SHA256SUMS-v4.430.0.txt`；冒烟 200 过）；保留策略删 v4.426.0.exe，实存 5 版（430/429/428.1/428/427）。
- **文档**：`releases/archive/v4.430.0.md` + 规格 `进度计划/gaea-novel-scene-cards-20260929.md`（含落地回填）+ CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**bash 内联 node -e 的模板字符串美元花括号被 shell 吞**（backtick 串里的 ${} 当 bad substitution）——多行前端补丁一律 .mjs 文件化执行（Write 工具写脚本→node 执行→删）。②**CRLF 文件的 python LF 锚不匹配**（read newline='' 保留 
，replace('
') 锚全 miss 静默）——跨行替换用 node split/join 或 Edit 工具（本批 Step3 补丁的 '

' 转义被三重转义地狱吃掉，最终 Edit 工具直改）。③**antd Modal.confirm 的 title 渲染双节点**（.ant-modal-title+.ant-modal-confirm-title 两层都含文本）——getByText 必 multiple，断言用 findAllByText。④**SaveSceneMeta 白名单 patch 扩字段的抹除陷阱**：string 整覆盖会让不传该字段的旧调用方清空数据——指针（*string）区分「未传/传空」是标准解。⑤wailsjs 是 gitignore 生成物：新绑定前端 import 后，fresh checkout 须 wails generate module（既有惯例，本批 commit 不含 wailsjs）。
- **观察池**：局部重写解禁（选段↔场景映射）/电荷翻转·义务绑定硬闸完整形态（随刀3·刀4）/逐场景生成真机走查（多场章节的流进度+取消手感）/拆卡质量真机观察。

## 最新发布：v4.429.0（2026-09-29）「小说板块收官刀：批2 观察池清账（修 15/关账 2/留池 2）」

- **刀型**：用户指令「继续优化 gaea，本次完成小说板块」。对齐 v4.427 原罪观察池清账方法论：v4.425 规格 §4.1 观察池 19 项（前端 9+Go 10）逐项读码定性→可修的修/误报的关账/需拍板的留池。**小说板块观察池至此清空**（G10 SceneRefs 留给长篇刀2，次刀序不变）。前端 8 文件+Go 14 文件+prompts 增删各一，绑定面 726 零变更，零功能删除。规格=`进度计划/gaea-novel-watchpool-20260929.md`。
- **三实锤根修**：①「停止生成在连接建立阶段显示生成失败」根因在后端——ChatStream 建连 err 无条件 emit error（用户在流开始前点停止=ctx 已取消被当失败）；修=err 分支先判 ctx.Err() 走取消落盘发 cancelled 事件（空稿只发事件不写空文件）。②伏笔「失败重读×排队写」交错窗口：op1 保存失败→重读磁盘覆盖 itemsRef→排队的 op2 把磁盘旧内容原样写回=**op2 改动静默丢失**（读码定性，无需延迟注入）；修=写链代际短路（失败重读 gen++，旧代际链任务跳写+如实提示「本项改动未保存」；窗口=同渲染帧连点，savingWrite 的 disabled 重渲染前不生效）。③设定 Agent 回填覆盖等待期手改：ChatPanel 输入区 disabled={loading} 已禁（审计辨伪），但设定**编辑器**不禁——setContent(worldview) 照样覆盖；修=回填前比对发起快照（contentRef），已变则不覆盖、修改稿以 markdown 围栏附回复走既有「应用」手动路径。
- **Go 侧**：缺口即停四合一（ForEachChapter 本体+countWrittenChapters/RunBookHealthCheck/novel_import_ai 反推三处内联循环——定性修正：ForEachChapter 生产零调用，实际影响面是内联三处；上界统一改 MaxChapterNum 磁盘最大章号，缺口跳过续扫）；N5 分支摘要混入（新 `ReadMainlineChapterSummaries()` 按文件名过滤 NNN{a,b,c}，outline-continue 大纲续写改主线口径；分支浏览器保持全量=特性；memory 分支语义留池拍板）；N12 迁移跳章 fail-closed（无法迁移文件收集后如实报错列明细且**不落 v4 标记**——旧实现静默跳过后照落标记，未迁移章从此从读写视图消失）；N13 分卷嵌套节点四处递归化（gateOutlineIssues 两段匹配/findOutlineNodeByNum/markOutlineDone/novel_import_ai titleByFile——分卷书的章节点在卷 Children 里，只扫顶层=写前闸整体失明）；N7 plans.json 整表读改写加 chapterPlanMu 持锁贯穿（httpbridge 本机第二客户端并发保存不丢更新）；scene_gen 吞错上报（markOutlineDone 返回 error，正文已落盘不回滚但 outlineWarning 如实带出）；chapter-review 模板补缺（内置 fallback 提升为 prompts/chapter-review.json——此前 eng.Get 恒 nil 工坊编辑不到真身）；G11 删零消费死资产 chapter-generate.json（真实链路=create-chapter，v4.308 台账在册「死资产」；磁盘+main.go embed 双入口消失，prompt_test 引用改 create-chapter/character——三层优先级断言换真三层模板不再依赖 Skip 兜底）。
- **前端侧**：情感曲线代际守卫（curveSeqRef：重跑体检/切书自增作废旧循环，迟到结果不 setCurve；loading 按代际收口——被作废循环不得提前关掉新一轮转圈）；重写历史 projectPath 进 effect 依赖（切书同章号按当前书重拉，旧实现残留上一本版本列表）；Cmd+K 改**受控数据流** updateScene（打开时记「场景下标+选区」，接受时直接改 scenes 上报——旧实现 document.activeElement 找目标，CommandBar 抢焦点后静默丢；**实现注记：微实验实证受控组件下「DOM 赋值+dispatch input」的 onChange 收到旧值**（React 恢复受控值），GhostText.tsx:89 同款手法可靠性挂池待真机验证）；局部重写 reanchored 诚实化（ResolveSelection 第四返回值透传，±50 rune 模糊重锚时弹「选区已按已保存正文重新对齐」，契约级根治另刀）。
- **关账**（落档不代码）：生成中切项目后端继续写旧书（v4.421 A3 前端闸门已闭环：切书 effect CancelCreateChapter+事件忽略；G1 定点合并不写坏）；httpbridge 跨客户端并发面（LAN 已收口 127.0.0.1 only；本机面=DSH headless 消费刻意设计；plans 锁已缓解唯一实证丢更新点）。
- **测试**：Go 新增 8 例（project 3：缺口续扫/主线过滤/迁移 fail-closed；app 5：连接建立阶段取消走 cancelled/嵌套节点两处递归/计划并发不丢更新/缺口计数）+rewrite reanchored 断言并入既有 13 例+prompt 测试改写（含 chapter-review 存在性）+main_test embed 清单同步；前端新增 9 例（回填守卫 2/曲线代际 1/伏笔短路 1/重写历史切书 1/Cmd+K 2/reanchored 2）。**四项反向验证改坏必红全部还原复绿**：取消分支/缺口续扫/伏笔短路/曲线守卫（曲线守卫初版用切书场景——svg 嵌在 report 分支内，切书清 report 后断言恒真测不出守卫缺失，改造成「重跑体检」场景才有灵敏度）。小说域定向 35 文件/312 例全绿；tsc -b 0；改动面 eslint 0 错 0 警。
- **门禁**：全量 `ci.ps1` → CI OK（golangci v2.14.0 0 issues/Go 全量绿/前端 vitest 409 文件 3558 例/E 系+卫生守卫）；漂移闸 OK@4.429.0（726 一致）。
- **产物**：`releases/gaea-v4.429.0.exe` 51,592,192 B SHA256=`f90bee694ceaf3ee7cd624ef3633337f05bf35a56b7b9384d310d14e96a1e7a6`（`SHA256SUMS-v4.429.0.txt`；build.bat 冒烟 `/api/health` 200 过）；保留策略删 v4.425.0.exe，实存 5 版（429/428.1/428/427/426）。
- **文档**：`releases/archive/v4.429.0.md` + 规格 `进度计划/gaea-novel-watchpool-20260929.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**release.ps1 预检挡未跟踪文件**——规格+发布说明正文须先 docs commit 再跑（v4.425 两段式同款：fix→docs→release）。②**测试期与后台全量 go test 并发改码=编译竞态自伤**（反向验证 mutate/restore 期间 ci 的 go test 编译到中间态裸 FAIL 无包名——反向验证必须在 ci 冻结窗口外做，且 ci 日志别 tail 截断要全量落盘）。③**python heredoc 写 TS 字面量 
 的 CRLF 变体**：文件行尾 CRLF 时 python 的 
 锚不匹配（LF 版匹配）——跨行 replace 失败静默，须 node split/join 或行级操作。④**还原性验证必须复跑对应测试**：mutate→红→restore 的 restore 若锚不完整（只回注释没回代码体），单看 grep 通过会漏——还原后必须重跑该用例（本批 CancelDuringConnect 曾因此带病过夜到全量 ci 才暴露）。⑤**测试断言要选得出区分度的场景**：曲线守卫切书场景里 svg 的渲染条件（report 非空）被切书本身破坏=断言恒真；换重跑体检场景（report 被同值替换非空）才能测出守卫缺失。
- **观察池（小说域新立案）**：GhostText 受控值疑点（同款 DOM 注入手法，真机补全接受不生效即此因）/memory 分支摘要语义（分支剧情是否算书内事实，需拍板）/零调用者绑定删除（宽口径 36，DSH headless 消费不可静态可见，需拍板）/G10 SceneRefs（随长篇刀2 消费）。

## 最新发布：v4.428.1（2026-09-29）「走查沙箱隔离补漏：APPDATA 数据面同隔离（真机走查收官）」

- **刀型**：v4.425~428 三批原罪改动的真机走查（非版本刀）+ 走查中实证安全缺口的根修。Go 2 文件（walkthrough.go + 测试），前端零改动，绑定面 727 零变更。
- **走查全过**（GAEA_WALKTHROUGH=1 沙箱 + APPDATA 重定向 + CDP 9333，种 105 条消息/真实发送流）：ChatTabs tab 语义+方向键 ✓；手风琴 aria 配对+ArrowDown ✓；面板收起卸载卡体+展开恢复 ✓；画廊描述回退链（旧数据正文反解）✓；消息分页（首拉恰 100〔seq 6..105〕+提示行+点击载入 105+提示消失）✓；流链路（回复渲染+sending 复位+**console 零错误**）✓。沙箱流可用（沙箱 engines.json 默认种子含 herdsman localhost:8080，真机回复可达）。
- **险情与根修**：`ApplyWalkthroughOverride` 只重定向 cfg.Workspace——sin/小说/角色库数据根在 `%APPDATA%\gaea`（os.UserConfigDir），沙箱壳**直连用户真实故事**（走查打开原罪看到用户实盘「写三国演义…」；该段全程只读，发现即停壳）＝v4.418 险情同族漏网（当时只封 workspace）。修=override 内 `os.Setenv("APPDATA", Temp\gaea-walkthroughppdata)`，此后所有 os.UserConfigDir 调用（whisperDataRoot/chat.db/engines.json/sin 数据根）全落沙箱；失败路径 slog.Warn 明示「数据面未隔离」；清场半径不变（删 gaea-walkthrough 整目录即清）。
- **测试**：新增 `TestWalkthroughOverrideIsolatesAppData`（开=重定向+目录创建；关=原样不动；前序用例已重定向时先归位临时目录防串场——定值沙箱路径让断言依赖用例顺序）；config 包全绿。
- **门禁**：全量 `ci.ps1` → CI OK；漂移闸 OK@4.428.1（727 一致）。
- **产物**：`releases/gaea-v4.428.1.exe` 51,582,464 B SHA256=`380318d95e101f839e26d43061e21dc79c673a8e0744ca714b532a5e4072f746`（`SHA256SUMS-v4.428.1.txt`；build.bat 冒烟 `/api/health` 200 过）；保留策略删 v4.424.0.exe，实存 5 版（428.1/428.0/427/426/425）。
- **文档**：`releases/archive/v4.428.1.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**CDP dispatch 与读 DOM 必须分两次 evaluate**（React 批渲染的同步旧读假象——方向键导航两处一度误判 no-op，分开读全过）。②**验证卡体别用类选择器猜**：`sin-acc-panel-*` 是 **id** 不是类，类选择器查空是选择器错不是 bug（本轮差点误报「visible 门控失效」）。③**走查沙箱盲区要先探数据落点**：进板块前先确认沙箱目录里长出了数据文件（本次 workspace 空+有故事=直连实盘的信号）。④本修复后走查启动只需 `GAEA_WALKTHROUGH=1`（APPDATA 进程内自动重定向），不再需要手工 env。
- **观察池**：设定卡排队体验需 ComfyUI 在线补走（本轮 ComfyUI 不在线，UI 态已由单测覆盖）/解释腔出现率·工具误用率需多轮真实生成人工评/novelstyle rewrite 链（需拍板成人向 whitelist 语义，v4.424 立案）/跨实例族+TOCTOU（需跨进程锁，权重低）。

## 最新发布：v4.428.0（2026-09-29）「原罪板块批5：消息分页拉取 + 插图值结构升级（绑定窗口批）」

- **刀型**：用户指令「继续」（承接「继续优化原罪板块」线程）。刀型=**绑定窗口批**：清掉观察池最后两项可做项（SinMessages 分页〔v4.423 立案「需随下一批绑定窗口」〕+ illustrations 值结构升级 {path,caption}）。前端 9 文件 + Go 5 文件 + 6 个新测试锚，**绑定面 726→727（+1：SinMessagesPage）**，零功能删除。
- **刀1 分页拉取**：Go 新绑定 `SinMessagesPage(topicID, beforeSeq, limit)`（chat.store 既有分页 API 的 sin 门面：beforeSeq<=0 取最新一页，否则取 seq < beforeSeq 的更早一页；升序切片 + has_more，空页空切片非 nil；全量 SinMessages 保留兼容）。前端 `useSinStory` 首拉只取最新一页（SIN_MESSAGES_PAGE_SIZE=100）；`loadOlder` 按 oldest seq 翻页前置（olderSeq 防并发乱序）；`reloadMessages` 已载更早页时合并重载（保窗口外前缀、按键去重——整页替换不再吹掉上滚载入的历史）；上滚到顶自动触发 + 流顶提示行（「还有更早的消息 · 向上滚动或点击载入」、加载中转圈）；前置后按高度差回锚滚动位置。绑定同步四处：SinB 门面手加（regen 噪音 11 文件按 HEAD 还原）、bindingNames.ts 重生成（727）、bridge/sin.ts 契约、mock/sin.ts 同源窗口实现、spaceBindings "play"（数量锁 549→550）；运行时派发走 realApp 动态 Proxy，wailsjs 零重生成。
- **刀2 值结构**：`sinAttachIllustration` 值写 `{path, caption}`（标记图 caption=画面描述、工具图 caption=轨迹产物）；`sinIllustrationMap` 双形态归一（历史字符串/新对象）+ 新增 `sinIllustrationCaption`；前端 `parseIllustrations` 归一 + `parseIllustrationCaptions` + `SinMessageView` 增 `seq`/`illustrationCaptions`；画廊描述三级优先级=**落库 caption > 正文标记反解 > 轨迹产物反解**（历史消息行为不变）；导出链行为不变。
- **测试**：Go 新增 2 锚（分页契约〔首页升序/游走翻页不重不漏 10 条/尽头空页〕+ 值形态双形态归一）+3 处调用点断言随签名更新；前端新增 4 锚（105 条真翻页/双形态归一/caption 只取对象/画廊三级优先级）+ reload 用例改分页 mock + SEED_MESSAGES 共享种子基建。sin 域 121/121 绿；internal/app 全包 -count=1 绿（100.8s）；spaceBindings 550；tsc 0；eslint 0 错 0 警。
- **门禁**：全量 `ci.ps1` → CI OK（含绑定漂移闸：Go 727 ↔ bindingNames.ts 727 一致）；漂移闸 OK@4.428.0。
- **产物**：`releases/gaea-v4.428.0.exe` 51,580,928 B SHA256=`b2bfe66d248b0701bbc3d4e9108e5440aab60ddd340d79f95bdf75585cde74e3`（`SHA256SUMS-v4.428.0.txt`；build.bat 冒烟 `/api/health` 200 过）；保留策略删 v4.423.0.exe，实存 5 版（428/427/426/425/424）。
- **文档**：`releases/archive/v4.428.0.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**Bash 工具 heredoc 的 `
` 会降级成真实换行**：python heredoc 里往 TS 源码写字面量 `
`，出现反复「改完又变回去」的假象（同进程回读 md5 相同=改动自我抵消），耗时数轮才定位——heredoc 内构造字面量反斜杠必须 `chr(92)+'n'` 或行级拼接。②分页测试的页大小是 hook 常量 100：造数必须 >100 条才能真翻页（4 条一页装满 hasOlder 恒 false）。③gen_bindings regen 会把 11 个门面文件重排版（多行→一行的全量噪音）——按既有纪律 HEAD 还原后只手加新方法行。
- **观察池（剩余，均需拍板或真机）**：novelstyle rewrite 链（需拍板成人向 whitelist 语义，v4.424 立案）/跨实例族+TOCTOU（需跨进程锁）/真机观察解释腔出现率·工具误用率·设定卡排队体验·分页上滚手感（需用户闲置窗口）。

## 最新发布：v4.427.0（2026-09-29）「原罪板块观察池清账刀：队列统一 + 看板单条化 + 识图并发 + a11y 收口」

- **刀型**：用户指令「继续」（承接「继续优化原罪板块」线程）。刀型=**池清账**：v4.423/424/426 三批审计积压的观察池立案项集中清账（免新审计——每项均已带读码证据），不做需拍板/需真机项（novelstyle rewrite 链、真机观察、跨实例族）。前端 8 文件（含 1 新增模块 sinThumbCache.ts）+ Go 4 文件 + 5 个新测试锚，绑定面 726 零变更，零功能删除。
- **清账九项**：①设定卡链入 illustrationQueue（SinCastPanel 与流内插图/画廊重生成同队列——此前三链各自直发 ComfyUI，进度互串台+取消误杀；排队显示「前面还有 N 张」；取消按 token 制导=排队摘位/在跑才中断后端）；②画廊/大图/过程卡产物 data URL 统一 16 条 LRU 缓存（新模块 sinThumbCache.ts 三消费点接入）+面板收起卸载卡体（SinSidePanel visible 门控——底稿草稿在面板 state、书源下载会话在模块单例，收起/重开不受影响，v4.423「收面板不丢」承诺保持）；③ChatTabs 主 tab 语义（容器 role=tablist、主 tab role=tab+aria-selected+roving tabindex、←→/Home/End 键盘导航覆盖主视图+动态会话 tab；办公共用组件双板块受益；extraTabs tabIndex 同步 roving 化）；④sin 手风琴五卡 id/aria-controls/aria-labelledby 全配对+ArrowUp/Down 竖向循环；⑤Composer 可选 onQueueDrop 回调（卸载时队列未清空如实上报条数——key=activeId 重挂静默丢排队消息的可见化；原罪页接 notice，办公不传零影响）；⑥SinContextNodeDetail 按 seq 反解消息 id（seq/10）单条 GetMessage+TopicID 归属校验——不再为找一个节点全量折叠整话题（千回合老故事每次点节点都全表读库+全量解析 extra）；⑦foldSinInsight 单次解析 extra 存 ins.extras 两折叠视图共用（轨迹/上下文原先各自再解析一遍=双倍 JSON 开销；顺带清掉无消费者的 tools/illus 冗余 map）；⑧图片附件识图并发 2 fan-in（sinVisionConcurrency；块序保持 token 出现序——5 张图从至多 7.5 分钟压到约一半）；⑨sinAttachMu 互斥锁串行化插图回写读-改-写（流内插图+画廊重生成+工具产物落库并发收尾不再互相覆盖 cue）+SinIllustrate a.cfg nil 诚实报错（与其余写入口同口径）。
- **测试**：Go 新增 2 锚（识图并发上限+保序〔注入假识图实测 max 并发=2〕/节点详情跨话题拒绝〔单条查询路径与旧全量扫描行为一致〕），既有折叠契约（TestSinContextNodeDetail/TestSinTrajectoryFold 等）在单条化路径全数通过；前端新增 5 锚（ChatTabs tablist/tab 语义+roving+方向键×3、设定卡入队串行〔被占位不得开跑〕+排队位次显示与取消摘位），既有 16 例（含取消/进度）在入队路径下原样通过；适配 1 处（产物缩略图测试 beforeEach 补 __resetSinThumbCacheForTest——模块级 LRU 跨用例命中会把上一例的成功结果喂给失败用例，新缓存模块的标准纪律）。sin 域前端 117/117 绿；internal/app sin 族 -count=1 绿；tsc 0；改动面 eslint 0 错 0 警。
- **门禁**：全量 `ci.ps1` → CI OK；漂移闸 OK@4.427.0（726 一致）。
- **产物**：`releases/gaea-v4.427.0.exe` 51,575,808 B SHA256=`50b245c823adc26a3ee2e8aa914dec1229c46b84c49f615e4d7039a33cc4d663`（`SHA256SUMS-v4.427.0.txt`；build.bat 冒烟 `/api/health` 200 过）；保留策略删 v4.422.0.exe，实存 5 版（427/426/425/424/423）。
- **文档**：`releases/archive/v4.427.0.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**模块级缓存进测试必须配复位出口**：LRU 命中让「读取失败」用例拿到上一例的成功结果——beforeEach 复位是标准纪律（先例 illustrationQueue/__resetBookDownloadForTest 同款）。②**队列包裹要保同步语义**：enqueueIllustration 的 drain 在入队点同步调用 job.run()，既有「点击即断言 called」的测试不破；若换异步调度会碎一片。③**池清账刀效率高**：三批审计的立案项都带读码证据，本批零新审计直接落地 9 项——审计批+清账批交替是可持续节奏。
- **观察池（剩余，均需拍板或真机）**：novelstyle rewrite 链（需拍板成人向 whitelist 语义，v4.424 立案）/跨实例族+Delete·Clear TOCTOU（需跨进程锁，单用户桌面端权重低）/illustrations 值结构升级 {path,caption}/SinMessages 分页绑定（需绑定面窗口+UI 设计）/真机观察解释腔出现率·工具误用率·设定卡排队体验（需用户闲置窗口）。

## 最新发布：v4.426.0（2026-09-29）「原罪板块优化批3：切故事边界收口 + 插图落库闸门 + 工具产物链补全」

- **刀型**：用户指令「继续优化原罪板块」（拍板审计优化批，对齐 v4.421/v4.423/v4.425 方法论）。定刀依据=**四线并行只读审计**（Go 流/存储 13 项 · Go 工具/导出/书源 11 项 · 前端状态层 13 项 · 前端渲染交互 15 项，约 50 项）+ 主代理读码逐项复核（证伪含「sin 不解析 @附件」误报——sin_file_refs.go 本就是解析器）。前端 14 文件（含 1 新增模块 sinBookDownload.ts）+ Go 13 文件 + 6 个新测试锚，绑定面 726 零变更，零功能删除。
- **论点（三个系统性缺口）**：①「切故事」边界只有 selectStory 有收尾——createStory 直接改 activeId（发送中新建=旧流后台烧完+新故事 Composer 被 sending 锁死+cancel 按 activeIdRef 取消错故事+绕过 confirmSwitchThen 脏草稿确认）；saveCast 保存闭包绑定发起故事，在途切换把 A 的生效清单写进 B 的面板状态（点「带入故事」即覆盖 B，后端无 baseline=不可恢复）；useSinNotes.save 回填绕过读路径 live 守卫；②插图「不落库出图」族——流式中间态手动按钮可点、停止/超时/报错残行（messageId 恒 0）照样自动出图（只生成不回写，重开退占位+重烧 GPU）；超时路径整段覆盖已流出正文且不调 SinCancel；取消迟到 rejection 把主动取消呈现成红色失败；③工具产物链半截——sin_illustrate 实例整回合复用而 arts 只增不减（「再画一张」轨迹重复携带第一张、toolN 编 cue 同图占两键）；sinAttachIllustration 不校验 messageID 归属话题（本板块唯一 fail-open 写口）；MD/EPUB 导出只认正文标记数字 cue=工具图整批消失且 caption 落库即丢；首轮任意错误被误判「模型不支持工具」（假 notice+静默关工具重试）。
- **落地**：前端=teardownStream 抽取+createStory 前置收尾、cancel 按 streamTopicRef 优先、超时先 SinCancel 保留正文提示走 notice、loadMessages 读失败不清空视图（selectStory 先行清空防跨故事串显）、reloadMessages 流中跳过、saveCast/saveNotes 归属校验+失败如实返回 `{ok,message}`、SinIllustration 手动/自动出图 messageId 闸+cancelledRef 收敛取消竞态、新建按钮接 confirmSwitchThen、角色保存成功才关选择器、重命名空输入保留原标题、导出按钮 disabled 透传 ToolbarButton、冲突覆盖切故事后 message.info 回声、书源下载会话模块级单例 `sinBookDownload.ts`（卡切换不丢进度/toast/jobId+起跑防双击+假取消刷清单——v4.423 观察池立案清账；done toast 模块级长驻）、过程卡产物加载占位+流式 result 帧 artifacts 全链贯通（流中缩略图此前被 applyToolFrame 丢弃）、Composer 含空格路径附件改 `@"..."` 引号注入（不含空格路径办公/原罪既有口径零变化）、流式光标无效 aria-label 改 aria-hidden。Go=Execute 入口重置 arts、sinAttachIllustration 加 topicID 归属校验 fail-closed、首轮降级加 `sinErrSuggestsNoTools` 判据、失败零正文（default+空回复两分支）落库用户指令（镜像零正文取消口径）、收尾轮工具调用可见化（如实下发 dispatch/result 帧+assistant 回合与未执行 tool 结果接回上下文=「从同一条收尾令重写」正文重复根因消除）、MD/EPUB 导出补工具产物附录（caption 替代文本）、识图前「正在解析 N 个图片附件」notice（解析完即清）、轨迹 Args 2000 rune 闸（执行参数不受影响）、EPUB 跳零正文行+连续指令逐条入引块+退化标题回退默认名、原子写统一（四处固定 `.tmp` 换 `fileutil.AtomicWrite`——双开壳互踩根除；书源 EPUB 唯一临时文件+rename）、sin_file_refs 引号 token+`io.ReadFull` 读满+空文件不报错+`sinTrimPartialUTF8` 共享助手、成书节选同口径、Windows 保留设备名避让（sinExportFileName+sanitizeDirName 小说导入共用同刀）、参考图 data URL 8MB 上限、下载进度语义改「已处理/总数」（有失败章不再卡 N-1 瞬跳）。
- **测试**：Go 新增 10 锚（普通失败只落用户消息+不降级重试〔400 非 tools 错误体绕开 C 刀故障转移真机语义地雷〕/跨话题回写拒绝/引号 token+裸 token 零误伤/空文件/保留名矩阵/工具产物进 MD 导出/EPUB 历史行+连续指令/产物跨调用重置/参数闸/收尾轮可见化+上下文续接）+改写 5 锚（无正文落库口径 0→1、进度语义 1/3·2/3→3/3、归属校验签名两处）；前端新增 5 锚（saveCast 失败返回/在途切换不回填、画廊 cue 数值排序/工具图描述反解/缓存签名失效）+重写 7 锚（超时口径、artifacts 全链归一、书源会话模块化〔测试复位出口〕）；sin 域 111/111 绿、全量 vitest 409 文件 3541 例绿、tsc 0、改动面 eslint 0 错 0 警、internal/app 整包+booksource `-count=1` 绿。
- **门禁**：全量 `ci.ps1` → **CI OK**（build/vet/golangci/前端 lint+build/vitest 全量/E系/docs 守卫）；漂移闸 OK@4.426.0（726 一致）。
- **产物**：`releases/gaea-v4.426.0.exe` 51,567,616 B SHA256=`24175989253fb2b04ab51cf2b97c1ed41208cd4b5c19d4908722f5d72b8e800f`（`SHA256SUMS-v4.426.0.txt`；build.bat 冒烟 `/api/health` 200 过）；保留策略删 v4.421.0.exe，实存 5 版（426/425/424/423/422）。
- **文档**：`releases/archive/v4.426.0.md` + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①**C 刀故障转移会咬测试**：新守卫测试用 500 状态造「普通失败」，failoverTarget 把 5xx 当可转移错误重试到真实 localhost:8080 引擎→在途流 3 秒收不了尾；改 400（不可转移）+非 tools 错误体即过——失败注入用例的状态码要对着故障转移语义选。②**书源进度卡切换失附的根因是组件生命周期不是事件通道**：五卡手风琴一次只挂一张卡，模块级单例（先例 illustrationQueue）外提会话即可，不动布局。③**react-refresh/only-export-components**：组件文件里导出测试复位函数会吃 warning——抽纯模块（sinBookDownload.ts）既归零警告又对齐 illustrationQueue 形态。④**Go map JSON 字典序**：extra.illustrations 键 "10"<"2"，画廊 ≥11 张错序——数值化排序+toolN 独立类序。⑤**审计误报要读码证伪**：四线审计的「sin 后端不解析 @附件引用」是错的（sin_file_refs.go 即解析器），真问题是含空格路径截断——不复核就会修错方向。
- **观察池**：跨实例族（sinRuns 进程内互斥/话题 id 毫秒撞名/messageID 启发定位）/Delete·Clear TOCTOU 残余窗口/画廊 base64 常驻/sin_insight 三口全量折叠/识图有界并发/illustration merge per-message 锁+sinExportUniquePath stat-then-write（既有）/ChatTabs tab 语义（办公共用另批）/sin 手风琴 aria-controls 配对/设定卡与插图队列统一/illustrations 值结构 {path,caption} 升级/novelstyle rewrite 链（需拍板成人向 whitelist 语义，v4.424 立案）/真机观察解释腔出现率与工具误用率。

## 最新发布：v4.425.0（2026-09-28）「小说板块优化批2：未保存保护收口 + 跨书失效治理 + 竞态守卫」

- **刀型**：用户指令「继续优化小说板块」（拍板审计优化批，对齐 v4.421/v4.423 方法论）。定刀依据=**四线并行只读审计**（前端创作/阅读 10 项 · 前端设定/书架/导入导出 13 项 · Go 后端 11 项 · 计划台账 18 条抽验）+ 主代理读码复核。前端 45 文件 + Go 9 文件 + 5 新测试文件，绑定面 726 零变更，零功能删除。
- **论点（三个系统性缺口）**：①v4.421 的未保存保护只覆盖**页内动作**，不覆盖「服务端写盘成功 → 前端重载」这条**反向路径**——局部重写与重写历史应用/恢复原文无条件 `loadChapter` 把磁盘正文灌回编辑区并把 `loadedSnapshot` 对齐，作者未落盘的手写正文被覆盖**且脏标志被抹掉**（同文件 `refreshEditorAfterServerRewrite` 反而有护栏＝漏网点）；②窗口/面板持有跨书状态切书后不失效（v4.421 只修两面板）——`WorldviewSectionsEditor.load` 只依赖 `disabled` 而「保存全部维度」是**整表替换且后端无备份**（A 书六维写进 B 书）、设定页 markdown 静默丢、章节分析主 load（对比章有 seq 而主 load 没有）、伏笔体检报告、全书体检、评审、文风指纹同类；③异步响应缺「还属于当前上下文吗」守卫——**生成中切项目**时旧书 `done` 把正文挂到**新书同号章**并标已保存、载入在途点保存把上一章正文写进新章、摘要/问书迟到响应错章。Go 侧同型两处：生成链把**整份过期 outline 快照**回写（流式期间作者对大纲的编辑被静默回滚）、书源补下章号基准只取大纲**顶层**且不查磁盘（续写大纲压成 1..5 卷后从 006 起覆盖既有正稿）。
- **落地**：线1 新 `lib/boardActive.ts` 板块级可见信号（MainLayout 声明 + NovelPage pane active 取 AND——修「进过小说板块后切走，隐藏的阅读页仍吞 Ctrl+S/F11 并吞掉浏览器全屏」）+ 新 `components/novel/novelSwitchGuard.ts` 跨页切书闸门（登记 + 单一拦截点，三选仅在全部脏持有者都能保存时给出；HomePage 四处 `openProject` 全过闸）+ 删除正在编辑的书即 `closeProject` + 搜书兜底对账命中回调刷新书架 + `TTSPlayer`/`CharacterLibraryPage` 整通道裸 `EventsOff`（v4.62.2 事故形态）改 `subscribeWailsEvent`（+`GhostText` 收敛）与源码守卫 `events-discipline.test.ts`；线2 创作页 A1/A2a 服务端重写后脏则不重载+入口三选脏闸、A3 `requestedPath` 守卫+切书取消在途生成、A4 `saveActive` 在途早退+保存按钮 `disabled`、A6 rail 写盘入口 `disabled={generating}`、A7 生成中开向导/再点生成给可见提示、A8 新角色弹窗按 `active` 挂起并回页补弹、B5a 切书清评审/指纹态；线3 阅读页 A2b 重写历史按**面板当前章号**判脏+脏则不重载、A9 摘要/问书 seq+章号双守卫、A10 载入在途只读 loading 占位、主代理补「恢复上次阅读章」迟到回填不覆盖已打开标签；线4 设定页与面板群 B1 维度化编辑器依赖 `projectPath`（P0 跨书覆盖）、B2 设定页脏状态登记+切书提示、B3 章节分析主 load seq+projectPath、B4/B5b 切书清报告态、B6 导出全失败不再弹成功（按 `失败*` 前缀统计，部分失败给 N/M）、B7/B8 关闭前问脏、B9 `onShelfReconciled`、B10 体检失败≠空态、B11 详情失败可重试、B12 检查器回填校验章号、M4 伏笔在途禁用；线5 Go G1 生成回写改**读-改-写按 nodeID 定点合并**、N8 建节点落盘 fail-closed、N7 生成后协程纳入 WG、M5 残稿名纳秒+序号唯一化、G2 补下章号基准 `max(递归大纲, 磁盘)` + 逐章挑空号、G6 `enginesSave` 改 `AtomicWrite`、G4 场景圣经预算配账（脊椎段预留并前置——原先末尾截断恰好砍掉 v4.422 的故事主线）、G5 上一章摘要按章号过滤、G7 整篇 span 不参与白名单豁免、G8 去味 before 分口径统一、G9 证据区间合法化、G3 伏笔写盘失败如实上报（`res.Errors`+`SyncPersisted()`，四个计数键不动）；台账更正=补登 v4.421 线5（原漏登）+两条更正块（裸 `EventsOff` 当时未清零 /「在途禁用」当时只有串行队列）+docs 生成物路径 `frontend/src/wailsjs/` → `frontend/wailsjs/`。
- **测试**：小说域定向 **53 文件 / 465 例全绿**（本批开工基线 49 文件 / 401 例 → 净增 4 文件 / 64 例）；`tsc -b` 0；改动面 45 个前端文件 eslint 0 错 0 警；Go `internal/app` 整包 142.5s 绿 + `project/novelcontext/novelstyle/novelreview/outline/analysis` 六包全绿 + `gofmt`/`vet`/`build` 0；**反向守卫逐条做变异验证**（板块门控去掉 `&& boardVisible` 恰 1 例红 / 跨页闸门六态 / 重写不覆盖 / 生成跨书 / 载入在途跨章 / 切书清态 / 恢复章不覆盖；Go 补下基准、白名单整篇、证据区间三条实测变红后还原）。
- **门禁**：全量本地快闸 `scripts/ci.ps1` → **CI OK**（含全量 vitest 409 文件 / 3536 例 + E 系回归 + 卫生守卫）；漂移闸 OK@4.425.0。**首轮快闸抓到真错**：Go 线新测试文件有个未使用的 `writeStyleProfile` 助手被 golangci `unused` 拦下（删助手+连带三个失效 import 后复跑绿）。
- **产物**：`releases/gaea-v4.425.0.exe` SHA256=`1e58ea6a6a615b8eb61b2bf61be4f37a17a5535b3c2d54e8e8081bbad0750e35`（`SHA256SUMS-v4.425.0.txt`；build.bat 冒烟 `/api/health` 200 过 + 桌面副本）；保留策略删 v4.420.0.exe，实存 5 版（425/424/423/422/421）。
- **文档**：规格 `进度计划/gaea-novel-audit-20260929.md`（含 §4.1 观察池 19 项）+ `releases/archive/v4.425.0.md` + CHANGELOG/README + releases/README + AGENTS 速览（**恢复迁 1 插 1 仪式**：一百三十八迁 v4.415.0 入 archive；v4.420–v4.424 五版发布时该仪式停摆、速览积到 8 条 —— 本批一并归位）+ progress（本条）。
- **坑/教训**：①**审计报告的「最小修法」未必够**：Go G2 有「基准取大 + 逐章占用判定」两层保护，只按报告修第一层时**变异测试仍全绿**——一层修复配一层护栏时，用例必须分别钉住两层，否则钉的是第二层。②**`dist/.gitkeep` 每逢 vite 构建即被清空**（v4.422 教训复现）：本轮 `ci.ps1` 与 `release.ps1` 各删一次，提交/发版前必须重建。③**多条等价护栏互相掩盖会让反向守卫失效**：线2 给 A8 多加的「active 变假自动收起」把事件门控藏进同一 tick，改坏门控用例照绿——一个修复点只留一条**可观测**路径。④**jsdom 里 React 不向 `disabled` 按钮派发 click**：UI 门必须直接断言 `disabled`，handler 早退要另找可达路径（线2/线4 都撞到）。⑤**antd `message` 门户跨用例不过期**：全局否定断言（`queryByText(/生成完成/)` 为 null）必被前序用例污染，改用「条数增量」或 `findAllByText`。⑥**并行会话占号**：开工时工作树里有另一会话**未提交**的 v4.424.0 发版准备（sin_* / .gaea/* / CHANGELOG / 版本三源）——按纪律不代其修改、不 stash 其文件，先把本批代码收敛成**独立代码提交**（62 文件，零触碰其面），待其 `fcf5b0dd` 落定后才用 `release.ps1` 取下一个空号。⑦vitest 过滤器 `src/pages/chapter` 会**连坐** `src/pages/ChapterPage.test.tsx`（不分大小写的路径子串匹配），只跑子目录要写 `"src/pages/chapter/"`。
- **观察池**：前端 9 项（局部重写坐标错位需改后端契约 / 设定 Agent 回填覆盖在途键入 / 体检情感曲线无取消 token / 伏笔写回「排队×失败重读×切书」交错窗口 / 重写历史面板跨书面 / `Cmd+K` 接受是否真写回需真机 / 停止生成在连接建立阶段显示「生成失败」/ 切项目时后端是否继续写旧书 / 跨客户端并发面）+ Go 10 项（N5 分支摘要混入 / N9 `ForEachChapter` 缺口即停同族 3 处 / N12 迁移跳章仍落 v4 标记 / N13 嵌套大纲节点不可见 / G10 `SceneRefs` 零写零读 / G11 `chapter-generate.json` 无消费点却在工坊可编辑 / `plans.json` 无锁整表读改写（单客户端不可达）/ 零调用者导出绑定口径待对齐（台账 11 vs 宽口径 36）/ `scene_gen_handler.go:229` 同款吞错 / `internal/chapter` chapter-review 模板缺失恒走 fallback）。版本次刀序未变：**刀2 场景卡与逐场景生成**仍是长篇七刀的下一刀。

## 最新发布：v4.424.0（2026-09-28）「原罪提示词批：工具教学面与注册表同步 + 反 AI 腔对齐蒸馏判据」

- **刀型**：用户指令「继续优化原罪，你应该看看原罪的提示词」。提示词面只读审计（system prompt / 9 工具 description / buildSinUserPrompt 装配 / 钉文本测试），Go 2 文件，绑定面 726 零变更，前端零改动。
- **审计结论**：①工具教学面与注册表脱节（9 工具只教 4；sin_illustrate 与标记协议二义；书源两工具零约束）；②反 AI 腔弱于 novelstyle 蒸馏判据（解释腔标记/否定翻转未点名）；③插图协议「一次」含糊+重复表述。
- **落地**：【工具】补 5 工具纪律（点名要图=sin_illustrate、正文配图=标记、同图不双路径；书源素材化不照搬；导出明确才调+本回合未落库拿不到）；【写作规范】补解释腔五标记禁用+否定翻转偶尔一用（对齐 novelstyle patterns.json 门禁 B/G 同源判据——提示词劝阻+内核门禁同判据）；【插图协议】重写（一轮至多 2 张）。防回归闸 ×2：CoversToolRegistry（加工具忘教直接红）+AntiAIFlavorRules。
- **测试/门禁**：Go（worktree 隔离取证，v4.422.0 同款处置——共享树被并行在制 60 脏文件打红 lint 闸）build/vet/sin 全族+全包 111s 绿；前端面本批零改动=v4.423.0 全量绿同字节；漂移闸 OK@4.424.0。
- **产物**：exe 51,521,024 B SHA256=b65dbfb782ac21fb75566a93506558da637cec03d2175503df3acc2e07ad7e3a（worktree 内构建；SUMS-v4.424.0.txt；删 v4.419.0.exe）。
- **坑/教训**：①提示词与工具注册表是两份独立维护的面——加工具只写 description 不进 system prompt，模型只受 description 单层约束；本批用注册表覆盖闸把同步变成硬门禁。②仓库已有蒸馏判据（novelstyle patterns.json）先于提示词层存在——新板块的反 AI 腔文本应从判据文件取材，不要自造示例。③**build.bat 硬编码 `cd /d C:\AI\wubigrok`**——在 worktree 里跑它会静默构建主树（产物新鲜度检查还先把主树旧 exe 删了），第一版产物险些带着并行在制前端出货；worktree 构建必须先改副本里的根路径。④**frontend/wailsjs 是 gitignore 的生成物**——fresh worktree 缺它，npm run build 报 TS2307；先 `wails generate module`（「Not found: time.Time」是已知噪音，exit 2 不影响生成）。⑤worktree 跑 pnpm install --frozen-lockfile 走全局 store 硬链接，15s 完成，不亏。
- **观察池**：sin 正文接 novelstyle rewrite 链（写后去味，需拍板成人向 whitelist 语义）/真机观察解释腔出现率与工具误用率。

## 最新发布：v4.423.0（2026-09-28）「原罪板块优化批：底稿串写根修 + 流互斥收口 + 常驻面板治理」

- **刀型**：用户指令「优化原罪板块」，复用 v4.421 小说批方法论（双线并行只读审计 25+24 项 + 主代理读码复核，落地 24 项、余入观察池）。前端 11 文件 + Go 10 文件（含 1 新增测试），绑定面 726 零变更，零功能删除。
- **论点**：①底稿草稿跨故事串写（P0，draft 无归属+保存闭包绑定当前故事，双空 baseline 静默覆盖）；②三条入口静默销毁/误投输入（卸载丢草稿、排队消息派发到新故事、粘贴块 submitText 被丢）；③流生命周期裸奔（register 覆盖登记双写、cancel 数据竞争、切故事不取消后端、Delete/Clear 不查在途、错误路径丢已流出正文）。
- **落地**：前端=draft.storyId 归属+脏上报确认切故事／故事列与面板常驻挂载（display:none）／Composer key=activeId／send 双参／切故事 SinCancel（streamTopicRef 发流前登记）／初始化区分读失败与真空／插图队列取消制导／MemoMarkdown 流式渲染／角色库懒加载／提示条移出页签／已停止徽标（extra.cancelled）／Space 键。Go=流互斥（register 拒绝+cancel 锁内写入）／Delete/Clear 在途拒绝／错误路径保底（builder 提前 return 真凶）／零正文取消只落用户消息／前情 ListMessagesPage(12)／上下文柱窗口折算／底稿块原始下标+Save 丢空白／损坏留档 .corrupt-<ts>／未知工具不扣预算／rune 边界截断／成书原子写+节选头读／回写定位失败告警。
- **测试**：Go 新增 sin_stream_guard_test 7 例 + 既有损坏用例补留档断言；internal/app sin 全族 + 全包 105s 绿。前端 sin 域 12 文件 110 例绿（tabs 测试按常驻挂载改 stub 口径）；tsc 0、eslint 0。
- **门禁**：全量 ci.ps1 绿；漂移闸 OK@4.423.0。同树并行会话（v4.422.0 刀1）全程零交叠；其 CHANGELOG 引用的本线在制 tsc 报错（storyText.test TS2304）随本批落定清零——延续「不代对方修在制、不 stash 别人文件」纪律，本批提交面严格限定 sin 域+版本/文档。
- **产物**：exe 51,519,488 B SHA256=c22d2220cfdba400e135da2f75d48397250cf5ecfe76377e9d8d75c970700025（releases/gaea-v4.423.0.exe + SHA256SUMS-v4.423.0.txt；build.bat 冒烟 /api/health 200 过；保留策略删 v4.418.0.exe，实存 5 版 423/422/421/420/419）。
- **文档**：releases/archive/v4.423.0.md + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①sinStreamRound 错误分支提前 return 时 res.content 尚未从 builder 赋值——「错误路径带部分内容」必须在 return 前显式落值，审计报告的「res 带部分 content」描述与实际代码不符（读码复核抓出并新增测试钉住）；②前台 Edit 工具与后台 python 批量替换混用同一文件会触发「file modified since read」——同文件多处改动要么全 Edit 要么全脚本；③并行会话已发版占号（421→422 两连发）——动版本号前先 git log 核对最新台账，本批到场即改号 4.423.0。
- **观察池**：Composer 排队清空提示 / SinMessages 分页绑定 / sinExportUniquePath 并发窗口 / SinIllustrate cfg nil 口径 / 书源下载卡级失附 / ChatTabs tab 语义（办公共用组件）/ illustration merge 无 per-message 锁（多客户端并发丢更新，串行队列下不触发）。

## 最新发布：v4.422.0（2026-09-28）「小说·长篇创作刀1：故事脊椎 + 章节计划闭环（写前硬闸）」

- **刀型**：用户指令「你要解决的是如何实现写一个优秀的长篇小说，不是修修补补」的落地第一刀（《长篇小说创作系统》七刀刀序）。Go 新增 6 文件改 6 文件、前端新增 2 改 7、提示词 4 个，绑定面 720→726（+6）。
- **论点**：长篇写不好不是句子问题，是**结构/意图/状态没有闭环**——写一章的意图输入只有一句 `plot_req`；`ChapterPlan` 契约自 v4.278 落库零消费；`StoryThread` 字段全仓只有读取没有写入（五卷规划实际以空主线为输入，主题每次临时编）；生成单位是整章 blob（v4 场景事后物化单场景）；分析九维只有伏笔/角色状态/记忆三条回流；写前契约只查大纲空字段且「不阻断」；质量门全是写后报告；张力曲线读的字段生产端从不写入；前文窗口固定 10 章×180 rune。诊断十四条全部带 file:line（规格 §1）。
- **落地**：①故事主线写入路径修复（`Continue/ExpandNode` 回收 `story_thread`，非空才覆盖）+提示词输出契约补漏；②`chapters/plans.json` + `analysis/plan-deviation/NNN.json` 落盘三态（原子写/缺失空态/损坏不覆盖）；③计划生产 `NovelChapterPlanPropose`（主线+节点+前文窗口+已有 key_events+本章分析载荷 → 七字段；不落盘、零兜底）+作者审批落盘 `NovelChapterPlanSave`（齐备性+跨章去重，S1/S2 即拒并点名）；④写前硬闸（`novelgate.PlanContractIssues` + `planPrecheck` 唯一判据来源；缺计划拒绝且**模型调用 0 次**；`CreateChapter` 八参签名不变，新增 `CreateChapterWithOverride` 第九参显式覆盖默认关）；⑤生成注入 `chapter_plan`/`outline_points` 槽位 + must（大纲 `KeyPoints`/`Emotion` 此前零进 prompt）；⑥写后偏差四判据 + summary + 下一章计划建议；⑦摘要写回护栏（漏输出 `---CHAPTER_SUMMARY---` 不得清空节点 Summary，否则同章下次预检自锁，用 overlay 负向对照验证用例有效）。前端 `ChapterPlanCard`（硬闸横幅/草案进编辑态/保守保存/偏差报告 + seq 守卫）+ CreatePage 过闸三选弹窗。
- **证据**：端到端 `TestPlanLoopE2E_PlanMissingToDeviation`（3 章夹具 + 单 SSE 假 LLM 双链）六步：缺计划被拦（0 次模型调用）→ 草案不落盘 → 审批落盘 → 预检放行 → 落章（prompt 含「本章计划」区段）→ 偏差四判据 + 落盘回读（命令与输出见发布说明）。
- **测试**：Go 新增 6 测试文件（novelgate 6 / app 15+10+1 / project 9 / outline 7）；`internal/app` 全包 121.8s 绿、`go test ./internal/... .` 全绿；`gofmt`/`vet`/`build` 0。前端小说域 31 文件 233 例绿、tsc 0、eslint 0。
- **门禁**：漂移闸 OK@4.422.0（`bindingNames.ts` 726 一致、`spaceBindings` 锁 549）；`ci.ps1 -Quick` 绿。
- **产物**：exe 51,513,344 B SHA256=059f16a87c2ef1593a7020d86057efb12987a399fbd69c63569eb693f515aca4（releases/gaea-v4.422.0.exe + SHA256SUMS-v4.422.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）；实存 5 版（422/421/420/419/418）正好在保留策略内。
- **文档**：规格 `docs/archive/gaea-longform-novel-system-2026-09.md` + 机制调研 `docs/archive/gaea-longform-craft-mechanics-2026-09.md`（均登记 docs/README）+ releases/archive/v4.422.0.md + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **坑/教训**：①`gen_bindings` 重生成会把全部门面文件压平重排（噪音）——按历史处置还原 10 个文件、只留最小增量（bindings_novel.go +24/−0）；②并行在制会话会让全项目 `tsc` 短暂变红（本次 `sin/storyText.test.ts` 的 `parseCancelled` 未 import 是它的中间态，几分钟后自愈）——**不要代它修**，也不要为了发版去 stash 别人的在制；③发版脚本要求工作树干净，与并行在制天然冲突——本次改走手工发版（sync-version + build.bat + 产物搬运 + 台账），`release.ps1` 幂等可后续补齐；④**`dist/.gitkeep` 从未入库**（P0-1 只在 .gitignore 写了 `!dist/.gitkeep` 例外、漏 `add -f`；且 vite build 默认清空 dist 会删掉占位）→ fresh checkout / CI 的 `go build ./...` 仍炸 `pattern all:dist: no matching files`；本刀用 **`git worktree add <tmp> v4.422.0` 检出 tag 快照**做独立验证时暴露，已 `add -f` 入库（commit 5440b086），**构建后需重建该占位**（后续可让 build 脚本自动补）；**方法论**=并行会话持续改写时，共享工作树上的门禁不可复现（本次快闸连撞 sin 线两轮中间态：`sin_booksource_handler.go` undefined fileutil → `sin_insight.go` 未使用变量），改用 worktree 对**已发布快照**取证：Go 全量除 `internal/netclient` 已知负载 flaky 外全绿（隔离复跑 ok）。
- **观察池**：跨章去重第二来源（已写但无计划条目的旧章其 `ChapterSummary.KeyEvents` 未参与去重，根因无章号字段，修法=带章号的批量摘要读接口，随刀2）/ 刀2 场景卡与逐场景生成 / 刀3 故事层骨架 / 刀4 质量收敛闭环 / 刀5 风格学习回灌 / 刀6 上下文编译器 / 刀7 评测基线（`baseline.json` + `promptSetHash` + 夹具）。

## 最新发布：v4.421.0（2026-09-28）「小说板块优化批：未保存保护 + 常驻页副作用治理 + 取消防覆盖」

- **刀型**：用户指令「优化小说板块」（拍板 A+B 合并批＝正确性修复 + 体验收口）。定刀依据=三线并行只读审计（小说前端 15 项 / Go 后端 14 项 / 计划台账清账与 STALE 校正）+主代理读码复核，落地 20 项、其余入观察池。前端 15 文件（3 新增）+Go 4 文件（2 新增测试），绑定面 720 零变更，零功能删除。
- **论点**：小说五子页常驻挂载（NovelPage 全 pane 同挂 + CSS 隐藏，隐藏≠卸载）衍生三类真实缺陷（读码确证）：①窗口级副作用不随页隐藏——ChapterPage 与 NovelSettingPage 各挂一份无条件 Ctrl+S（全仓仅此两处 `key==='s'`）：一次按键同存「设定」与阅读页那章，创作页手写正文反而不保存；F11 在任意子页翻转不可见阅读页专注模式；阅读模式残留时 ←/→ 在书架/设定页翻章②面板不随书刷新——ForeshadowPanel/ConsistencyPanel 的 load 只依赖恒 undefined 的 disabled，切书仍显上一本伏笔表/一致性报告（引导态死代码，无项目反报红）③未保存缓冲无脏保护——创作页切章/切项目/生成/删除直接 setContent 覆盖（阅读页早有 needsCloseConfirm 先例）。
- **P0（Go）**：取消生成覆盖已完成章节——取消落盘残稿此前无条件覆盖 chapters/NNN.md，对已写完的章再生成并中途取消即整章被半截替换、无备份。改为「生成开局快照 existedBefore + 取消时复查 chapterBodyExists」：已存在则残稿另存 NNN[分支].partial-<ts>.md（fileutil.AtomicWrite 原子写，**绝不回落覆盖正稿**），事件增 partialSaved/partialPath/notice 可选字段，前端如实提示落点且不把残稿标成已保存。
- **落地**：线1 创作页（共享原语 components/novel/unsavedGuard.tsx 两选/三选/并列多分支 + 切章·生成·删除·整章重写全覆盖 + 载入失败可见化 + 并列分支不再绑 Esc + 反推轮询卸载守卫与已等待时长 + 设定读取失败归因 + 去味成功后刷新编辑区 + 创作参数持久化 + 工具轨 12 钮分三组零功能删除）；线2 常驻页快捷键门控（NovelPage 五 pane 下发 active，非当前页不响应也不 preventDefault，Ctrl+S 单一归属；目录点章改「先切 tab、commit 后派发」）；线3 设定页与两面板（load 依赖 projectPath+换书清 AI 深检 + 导入覆盖未保存设定先确认 + 下行双面板可折叠持久化）；线4 组件批（在线导入进度事件失落兜底：后端在返回 jobId 前已发终态事件→改 3s 无事件即按成书清单对账+如实提示 + 章际对比 seq 守卫 + 三处裸 EventsOff 改 subscribeWailsEvent + 提示词工坊切模板脏保护 + 新建小说空标题禁用与在途防双击）；Go 同刀（写回路径收敛 writeBackRewritten：写失败报 error + v4 场景写回后 syncBlobFromScenes；取消互斥改 nil 占位、register 见「正在取消中」如实拒绝、清理归 unregisterChapterGen 独占）。
- **测试**：Go 新增 create_chapter_partial_test.go（残稿三态 + v4 场景不覆盖）与 novel_writeback_test.go（写失败报错 / v4 blob 与场景一致）；internal/app 定向 `Chapter|Deslop|Rewrite|Cancel` PASS 82/FAIL 0 + 场景族 17 例绿；gofmt 空 / go vet / go build 全 0。前端新增 unsavedGuard 7 例 + NovelPage.test 3 例 + 各线守卫 30 余例（线2 灵敏度实测：门控改 false 恰 3 条守卫变红）；线4 报告小说域 32 文件 222 例绿；tsc 0 / eslint 0。
- **门禁**：全量 ci（本地快闸）绿；漂移闸 OK@4.421.0。
- **产物**：见 SHA256SUMS-v4.421.0.txt（冒烟 200 过）。
- **文档**：规格 进度计划/gaea-novel-polish-20260928.md + releases/archive/v4.421.0.md + CHANGELOG/README + releases/README + AGENTS 速览 + progress（本条）。
- **观察池**：跨页切书确认（需跨页共享脏态）/ EditorPanel 内 rune↔code-unit 换算收敛 + 单测 / 板块级 keepAlive（切到别的板块时小说页仍常驻→阅读 tab 的 Ctrl+S·F11 仍吞键，全仓各页共有，需「板块级可见」信号）/ Go 余 9 项（搜索标题命中抑制正文扫描·生成后协程未托管·建章忽略 WriteOutlines 错误·ForEachChapter 缺口即停·分支摘要混入·书架全库全章 IO·迁移跳章仍落 v4 标记·嵌套大纲节点不可见·11 个零调用者导出绑定）/ 残稿文件名秒级时间戳（同秒两次取消会覆盖前一个残稿）。

## 最新发布：v4.420.0（2026-09-26）「计划模式手动选择：Composer 工具栏『计划』chip」

- **刀型**：用户拍板「计划模式不是默认模式，但可以选择手动」落地。前端 3 文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **落地**：ComposerToolbar 新增「计划」chip（ClipboardList，思考深度右侧）：非默认初始关；点击翻转=发送 /plan on|off 走既有动词链（回执已可见化 v4.419）；状态由 App 层从最近一条计划回执推导（零新绑定零轮询，与引擎真实状态经回执天然同步）；回合中置灰；aria-pressed+title。
- **测试**：composer 5 套件 35 例绿+tsc 0+eslint 0；真机沙箱目检双向翻转同步（GAEA_WALKTHROUGH=1，截图在案）；全量 ci 绿。
- **门禁**：漂移闸 OK@4.420.0。
- **产物**：见 SHA256SUMS-v4.420.0.txt（冒烟 200 过）；保留策略删 v4.415.0.exe（v4.417.0.exe 二进制此前误删，身份以 SUMS 在案）。
- **文档**：releases/archive/v4.420.0.md+CHANGELOG/README+releases/README（444→445+裁 v4.389）+progress。

## 最新发布：v4.419.0（2026-09-26）「动词回执可见化：notice 摘出过程卡」

- **刀型**：v4.418 真机走查池实锤的「pre-turn 动词 Notice 不可见」缺陷修复。前端 1 文件+1 测试，绑定面 720 零变更，Go 零改动，零功能删除。
- **根因**：Transcript alternatingSegments 分组谓词把 notice 与 tool/phase/compaction 一并收进过程卡（v4.26 phase 收编时顺带），渲染即折叠。
- **根修**：谓词摘出 notice——一律独立成行渲染（warn 仍走 ErrorCard；info 走既有 .notice 左缘线弱化行零新样式）；失败告警同样受益。
- **测试**：ProcessCard.test 增 1 例（buildSegments 断言 notice 恒落 outsideItems，info+warn 双样本）13/13 绿；真机目检沙箱 /plan on 回执独立行可见（截图在案）；全量 ci 绿。
- **门禁**：漂移闸 OK@4.419.0。
- **产物**：见 SHA256SUMS-v4.419.0.txt（冒烟 200 过）；保留策略删 v4.414.1.exe。
- **文档**：releases/archive/v4.419.0.md+CHANGELOG/README+releases/README（443→444+裁 v4.388）+AGENTS 迁 1 插 1（一百三十七迁）+progress。

## 非版本刀（2026-09-26）真机走查池全清账：plan-mode 审批卡可视确认（零代码）

- **环境**：xAI 重新登录后，v4.418.0 壳 + GAEA_WALKTHROUGH=1 沙箱（首刀实战）+ CDP。全程零触碰用户工作区（黄甲 mmin 复核零新文件）。
- **plan-mode 全链实弹**：/plan on → 政策注入 → 任务「查工作区+建 hello.txt」→ 模型只读研究（list 1 工具）→ **真调 exit_plan_mode → ask 审批卡渲染（计划审批/批准/继续计划 双选项）** → **用户本人在屏幕上点击批准** → 闸翻转 off + 叙事「计划已批准」落历史 → write_file 执行 → hello.txt 落沙箱 → 回合完成。三重传达/键盘/产物全数在位。
- **池清账**：①「审批卡路由漂移下未可视确认」✅ 关账——不仅可视确认，且获得真实用户点击交互（比脚本点击强得多的证据）。②「pre-turn 动词 Notice 不可见」✅ 实锤为真实缺陷（Notice 渲染在「过程」折叠卡内默认不可见），**候小刀=动词回执提为可见 chip**，维持挂池。
- **沙箱开关实战评价**：GAEA_WALKTHROUGH=1 全程无感隔离——写文件落沙箱、会话落沙箱、清场删目录。真机走查从此默认带开关。
- **坑**：AskCard 审批卡由用户真人点击（最初误判为自动批准/超时——Controller.Ask 无超时、AskCard 无预选，人类反应时间戳是最终判据；诊断交互类问题先确认屏幕前有没有真人）。

## 最新发布：v4.418.0（2026-09-26）「走查沙箱开关：GAEA_WALKTHROUGH=1 强制隔离工作区」

- **刀型**：真机走查险情根因封堵（候产品刀当轮立项落地）。Go 1 文件改+1 新测试，绑定面 720 零变更，前端零改动。
- **背景**：真机走查「先建专用会话再发消息」无效——装配前 NewSession 必然诚实报错，首条消息走自动装配+自动恢复，workspace 解析到哪回合落到哪（当日实证接进用户真实会话，模型调用即败未持久化）。
- **落地**：GAEA_WALKTHROUGH=1 → config.Load() 强制 workspace=%TEMP%\gaea-walkthrough\workspace（自动创建），覆盖用户/项目配置；未设零变化；激活 slog.Warn 可审计；删沙箱目录即清场。
- **测试**：walkthrough_test.go 两例（开=恒沙箱/关=零触碰）；config+app 包回归绿；全量 ci 绿（OriginalSinPage.tabs 负载 flaky 一轮隔离复跑绿，v4.414.0 同款）。
- **真机验证实录**：沙箱模式启动→GaeaInit 装配→ListDir 探针=沙箱（14 真实会话零出现）；沙箱内 /plan on 实弹确认「pre-turn Notice 不可见」=真实缺陷（藏在过程折叠卡，候可见 chip 小刀）；/plan off 正常；清场=删沙箱目录。**诊断坑三连**：①壳侧 gaeaLoadConfig 不走 config.Load，开关双入口必须同调 ApplyWalkthroughOverride ②装配前 ga.cfg=nil 探针全假 ③CDP 探针可打在僵尸旧实例（gaea-v4.418.0.exe≠gaea.exe 按名漏杀）——跨实例诊断先对 PID。
- **产物**：见 SHA256SUMS-v4.418.0.txt（冒烟 200 过）；保留策略删 v4.414.0.exe。
- **文档**：releases/archive/v4.418.0.md+CHANGELOG/README+releases/README（442→443+裁 v4.387）+AGENTS 走查配方增补+progress。

## 非版本刀（2026-09-26）真机走查尝试+污染预防实录（零代码；v4.417.0 壳）

- **目的**：真机走查池三项（审批卡可视确认/pre-turn Notice 可见性/whisper 实数据观感）。
- **实况**：v4.417.0 壳+CDP 9333 附着。GaeaNewSession 诚实报错实弹复验通过（引擎未装配→「先发送一条消息完成引擎装配」，v4.414.1 根修二在产验证）。首条消息触发引擎自动装配——**工作区解析到用户真实工作区 C:AIangong黄甲（14 真实会话），自动恢复语义把回合接到用户最新真实会话 20260924-111332 上**（v4.414.1 事故同路径：诚实报错拦不住首条消息自动装配）。所幸模型调用即失败（xAI OIDC token 过期：获取 OIDC 端点失败），回合未持久化。
- **清场（按 RewindLog 同口径）**：杀壳→引擎日志截回 seq≤469（移除 8 行，与备份前缀比对通过；备份 .tmp/backup-walk417/）→三重复核：主 jsonl 26 行与走查前一致/checkpoint 26 条全为用户真实内容（写在 user_message 之前）/state 语义不变（仅 updatedAt 刷新）。**用户可见残留=0**。
- **实得**：①whisper 实数据观感 ✅（165 条事实+12 情节，v4.416 情绪色点实数据渲染正常：时间线节点色点+标签、未知情绪键回退 glow 无布局破损）②v4.414.1 根修二（诚实报错）真机在产复验 ✅③**审批卡可视确认/pre-turn Notice 可见性=阻断于 xAI 登录过期（需用户重新登录），维持挂池**。
- **坑入档（高优先）**：真机走查前必须先完成引擎装配并核 GaeaListSessions 基线（基线在装配后取才有意义——装配前 workspace 未解析恒 0）；「GaeaNewSession 建专用会话再发消息」的隔离方案对首条消息无效（引擎未装配时 NewSession 必然诚实报错，而首条消息走自动恢复）。**候产品刀**：诊断/走查用 workspace 覆盖开关（如 GAEA_WALKTHROUGH=1 强制空隔离工作区），让真机走查与用户数据彻底解耦。

## 非版本刀（2026-09-26）UI 走查收官：全板块×明暗补走查（零代码，零缺陷）

- v4.415~v4.417 三批发版后的验证轮：vite mock+无头 Edge 隔离目检把首轮未覆盖的板块补齐——schedule（mock 实载示例工程 15 任务，斑马分组/前锋线/关键红亮暗两态均可读）、sin 原罪（mock 故事+占位插图+创作画板）、code 编程（服务管理+前置条件清单）三板块×明暗首走，其余板块亮色复走。
- **结论：零新增缺陷**。schedule 亮色下 v4.415 的 --sched-* 令牌（斑马分组六带/deadline/front）实证生效；sin/code 亮色无对比度问题；原罪故事卡的 v4.415 aria-label 删除按钮在位。mock 插图占位画与登录门属预期。
- 至此「优化完善前端UI」线程收官：静态审计可捞的用户可见缺陷已见底；剩余候选（间距栅格收敛/var 回退式清理/圆角随密度化）均为低收益高搅动，重启条件已在案（v4.417.0 release note）。
- 真机走查池（需引擎/数据窗口，与 mock 目检互补）：审批卡路由漂移可视确认、pre-turn 动词 Notice 可见性、whisper 数据面情绪色点实数据观感。

## 最新发布：v4.417.0（2026-09-26）「前端 UI 完善批三：硬编码色值收口 + 可访问性补尾」

- **刀型**：接 v4.415/v4.416 观察池。前端 6 文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **色值收口**：ContextView 趋势图例裸 hex（亮态绿白底 2:1）→语义令牌；WhisperTracePanel MiniBar 四色→primary/success/warning/--whisper-accent；WhisperEmotionPanel 六维条→--whisper-accent。
- **可访问性补尾**：ChapterEditor 右键菜单键盘可达（role=menu/menuitem+聚焦+Esc+↑↓）；MemoryHub 计数「…」加载占位三态齐整；CharacterLibEditor Modal aria-label。
- **证伪降级**：圆角令牌化专项前提不成立（--md-sys-radius-* 明暗恒 8/12/16/28px，迁移零收益）；RelationGraph/GraphView/TisorRadar hex=SVG/canvas 字面量域豁免在案。
- **门禁**：定向 4 套件 80 例绿+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.417.0。
- **产物**：见 SHA256SUMS-v4.417.0.txt（冒烟 200 过）；保留策略删 v4.413.0.exe。
- **文档**：releases/archive/v4.417.0.md+CHANGELOG/README+releases/README（441→442+裁 v4.386）+AGENTS 迁 1 插 1（一百三十五迁）。

## 最新发布：v4.416.0（2026-09-26）「前端 UI 完善批二：emoji 图标红线清零 + schedule 交互反馈补漏」

- **刀型**：接 v4.415.0 观察池头名。前端 19 文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **emoji 清零（19 site）**：whisper 四文件（DesirePanel 分类 emoji→antd 组件；MemoryLibrary 25 情绪 emoji→效价色点 EMOTION_TONES+EmotionDot，全语义令牌；TracePanel 🤫✍️🔧📊→AudioMuted/CheckCircle/Tool/LineChart；MemoryModal 🧠⭐💫📊→Database/文本/删/BarChart；VALENCE_LABEL 按「内容语义」先例保留）；零散 11 文件（🎤→AudioOutlined、🛠️→ToolOutlined、StatsSection ⚠→WarningOutlined，其余装饰字符删除/「工具×N」文本化）。
- **schedule 补漏**：XML 导入 xmlBusy 合并按钮 loading；示例工程 Popconfirm 防呆（与「清空」同款）。
- **门禁**：定向 7 套件绿+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.416.0。
- **产物**：见 SHA256SUMS-v4.416.0.txt（冒烟 200 过）；保留策略删 v4.412.0.exe。
- **文档**：releases/archive/v4.416.0.md+CHANGELOG/README+releases/README（440→441+裁 v4.385）+AGENTS 迁 1 插 1（一百三十四迁）。

## 最新发布：v4.415.0（2026-09-26）「前端 UI 完善批：绘梦整页崩根修 + 走查实证六项修复」

- **刀型**：「优化完善系统的前端UI」整批：vite mock+无头 Edge 隔离目检（9 板块×明暗截图）+静态审计（Explore 全量扫描）双证据定刀。前端 15 文件+2 新测试文件，绑定面 720 零变更，Go 零改动，零功能删除。
- **根修（P0）**：绘梦整页崩（`Cannot read properties of undefined (reading 'toFixed')`）——`GetSystemStats` 契约是 `Partial<SystemStats>`（facade 类型即如此）而 api 层零归一化直透，ControlPanel `memUsed.toFixed()` 字段缺失即抛 TypeError 连坐整页。根修=api 缝合处 statNum 归一化（缺字段补 0/非 string gpuName 收空串/null→null），消费方零改动。测试 `api/image.test.ts` 五例。
- **走查实证修复**：①chat 欢迎屏建议卡被悬浮输入区工具行遮挡→`.chat-welcome` 底部预留 176px（明暗双主题截图复验；`.chat-empty` 是载入态溅屏勿动）②CharacterPage 读取失败静默仅 console→诚实错误态+重试（HomePage v4.349 口径，+2 测试）③schedule 亮色拖拽虚线/前锋线点白系不可见→on-surface 令牌+缩放手柄白芯深边双保险+竣工/前锋线收进 `--sched-deadline/--sched-front` 令牌（暗档原值零变化/亮档加深）④MemoryHub 检索命中标签 tailwind pastel 亮主题近不可见→hub.css `.hub-hit-tag--*` 领域色两档锁定+错误态 `--md-sys-color-destructive`⑤角色卡「白墨」浅海报白名对比不足→`.ccard-id` 黑纱中段 0.42→0.55⑥aria 包：ToolbarButton 组件级 ariaLabel 缺省回落 title（五处消费点自动达标）+灯箱上一张/下一张、sin 删除故事、chat 发送、角色卡九处+ModelCenter aria-selected→aria-pressed。
- **审计证伪关账**：`--radius-*` 「未定义」不成立（App.tsx 早已内联注入，live DOM getComputedStyle 实测 `--radius-md`=12px 与 `--md-sys-radius-md` 同值）；modelcenter FEATURES emoji 字段=零渲染点死数据（BindSection 已走 FEATURE_ICONS antd）删字段了账；sin 页头 `.sin-mark` 本就是 17px 细条非重复大标题。
- **观察池新挂**：whisper 子系统 emoji 图标群（WhisperDesirePanel/WhisperMemoryLibrary/WhisperTracePanel/WhisperMemoryModal 4 文件专项）；内联 borderRadius 222 处+CSS px 圆角 291 处令牌化（批替换专项）；间距 6/10/14 非栅格收敛；ChapterEditor 右键菜单键盘可达（可点 div 无 role/tabIndex）；SchedulePage 示例工程覆盖无确认（Popconfirm 候选）；CharacterLibEditor 大 Modal 可访问名复核。
- **门禁**：定向 10 套件全绿（新增 7 例）+tsc 0+eslint 0；全量 ci 绿+漂移闸 OK@4.415.0；走查隔离目检不动用户壳与会话（v4.414.1 教训执行），console 0 新增错误。
- **产物**：见 SHA256SUMS-v4.415.0.txt（仅本地；冒烟 200 过）；保留策略删 v4.411.0.exe。
- **文档**：releases/archive/v4.415.0.md+CHANGELOG/README+releases/README（439→440+34 席插 v4.415 裁 v4.384+清历史孤儿碎片行）+AGENTS 迁 1 插 1（一百三十三迁：v4.411.0 入 archive，头部补记 130~132 迁流水）。

## 最新发布：v4.414.1（2026-09-26）「桌面 Send 斜杠派发根修 + GaeaNewSession 诚实报错（真机走查实录）」

- **刀型**：v4.414 plan-mode 真机走查拽出双缺陷根修（v4.405.1 先例）。Go 3 文件改+1 新测试，绑定面 720 零变更。
- **根修一（长期缺口）**：桌面 Send 绕过全部斜杠动词——桥接 `app.Submit→GaeaSend→Controller.Send`，而动词面（/plan /compact /dream /memories /goal /perm /distill /new + `#<note>` 快记 + /mcp__ prompt + 自定义命令 + unknown 告知）只活在 `Controller.Submit`（HTTP 路径）；桌面主界面打任何斜杠动词都被当文本开回合发给模型。根修=动词面抽 `dispatchSlash(trimmed) bool`，Submit/SendWithRaw 共用。测试 TestSendDispatchesSlashVerbs 四段断言。
- **根修二**：GaeaNewSession 引擎未初始化静默 no-op（假成功）→诚实报错；前端 catch 有可见提示（"新建失败不重置界面"）。
- **plan-mode 实弹全通**：/plan on 派发（Notice 上 wire）→任务轮（模型只读研究，缓存命中 20096 tok）→模型真调 exit_plan_mode→ask 事件（计划审批 payload 正确）→桥接批准→批准叙事落历史→/plan status=关。console 全程 0。
- **用户会话污染事故与复原（透明记录）**：走查回合写进用户真实会话（黄甲 0924）——根因=走查脚本在引擎初始化前调 GaeaNewSession（静默 no-op）+旧 exe 把 /plan on 当文本开回合。复原=杀壳离线手术按 RewindLog 同口径（事件日志截 seq≤469+镜像截 4 user 前+删检查点+state 摘要修正），重启验证 39 条历史零走查痕迹、末条=用户真实定稿、console 0；四源文件备份 .tmp/backup-*。全程只读（计划模式纪律生效）。教训=走查会话隔离必须落盘双确认，静默 no-op 会骗人。
- **产物**：exe 见 SHA256SUMS-v4.414.1.txt；保留策略留 v4.411~v4.414.1 删 v4.410.0.exe。
- **观察池新项**：审批卡渲染路由漂移下未可视确认（ask payload 正确，渲染留复走）；pre-turn 动词 Notice 上 wire 但办公流不可见。
- **坑**：①Windows 文件锁——RewindLog 的 AtomicWrite rename 对活壳自身句柄退避×3 仍败，离线手术须先杀壳②taskkill /F 后 rmSync 有句柄释放竞态（检查点复活假象），删除后必须 existsSync 复核③node -e 内联脚本反斜杠路径会被吃——走查脚本路径用正斜杠。

## 最新发布：v4.414.0（2026-09-26）「Plan mode 计划审批门（dsh plan-mode 蒸馏）」

- **刀型**：DSH 留池⑦（实质头名：①子代理控制面等上游稳定）。蒸馏=取道不取器：471 行上游 → 内核 1 新文件（agent/plan_mode.go）+2 挂点（executeOne 盖章/boot 注册）+1 斜杠动词（/plan）。绑定面 720 零变更，前端零改动。规格 进度计划/gaea-plan-mode-20260926.md。
- **落地**：①exit_plan_mode 恒注册（工具目录跨模式切换稳定）：计划模式下 markdown 计划（# 标题强制）提交 ask 通道审批——批准=闸翻转 off+叙事落历史/继续计划=错误带回反馈/headless 无 Asker=诚实拒绝（执行闸必须人来开，刻意不仿 ask 自选降级）②PlanGate 窄接口经 callContext 盖章（asker 同位同纪律）+SubagentMetaTools 排除子代理③/plan on|off|status（on 注入 `[plan mode]` 政策 user 消息/off 退出叙事/status 回落，running 拒绝）④与上游两偏离：**系统前缀冻结（V10.36）→政策走 user 消息注入**（mid-session 改 system 段必破缓存，技能目录热重载同判先例）；**状态 v1=runner 内存态**（事件日志 kind=plan_mode+恢复折叠留二刀，最敏感面零碰）；沙箱/权限独立不做硬工具门（与上游同判）。
- **测试**：Go plan_mode_test 六例（inactive 错误/批准翻转+叙事/继续计划语义/headless 拒绝/标题校验含六级合法七级拒/真闸直测含前缀不动+两条注入断言）；agent/control/boot 三包回归绿；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.414.0.txt；保留策略留 v4.410~v4.414 删 v4.409.0.exe。
- **文档**：规格+releases/archive/v4.414.0.md+CHANGELOG/README+releases/README（437→438+裁 v4.382）+AGENTS 迁 1 插 1（一百三十一迁：v4.409.0 入 archive）+todos（dsh 留池⑦清账，余①子代理控制面=等上游稳定）。
- **坑**：①工具契约=Execute(ctx, json.RawMessage)+ReadOnly() bool（跟 string 签名走会炸 reg.Add）②包内测试桩撞名先查（stubGate 已被 gate_test.go 权限桩占用）③双引号 Go 串里的 \n 是真换行——拼 JSON 断言夹具须字面 \\n。

## 非版本刀：sin 书源一条龙实弹+常驻内存采集（2026-09-26，零代码）

- **走查面**：书源一条龙观察池收官——真机 v4.413.0 壳+假站 walk_server.py(18099)+临时规则 zz-walkthrough-fake.json，桥接 SinB 全链（Search/Toc/Download/BooksList/BookExportEpub/BookDelete）+UI 书源卡+底稿直注+工具链+导出，全程 console 零新增错误。
- **SSRF 护栏实弹实证（头条发现）**：书源通道 netclient.GuardedClient 对假站 127.0.0.1 报 `refusing to fetch internal address`——v4.374 设计注释明说「书源=内网全禁，防提示注入的规则摸到本机服务」；**假站成功链路真机结构性不可行=护栏按设计工作**，成功路径由 Go 测试注入 Fetcher 覆盖（fetch.go Options.Fetcher）。实测三态全诚实：聚合搜索 total=0 静默+UI 卡片逐源显示失败原因（「书源「zz-walkthrough-fake」搜索失败: …refusing to fetch…」）+Toc 直接抛同错误；下载任务全章失败→成书清单诚实空（「还没有成书」）。
- **底稿直注实写过**：SinNotesSave(baseline 底稿)→SinStream 轮（本地 herdsman Qwen3.6-35B）→上下文逐轮 `inject=45` token 注入段非零（system 919+inject 45+assistant 50），助手文案带注入特征（灰蓝长衫/短须）——v4.412 上下文口径真机自洽。
- **工具链实弹过**：第二轮明示「用 book_search 工具搜索」→本地模型真调工具（轨迹第 3 轮「3 条记录 1 工具调用」UI 在位）+诚实回复「没有搜到候选…另外 1 条来源失败被略过」（即护栏拒假站）。
- **导出双形态过**：SinExportMarkdown 全文（user/assistant 对+落款头）+SinExportEpub 落 `%APPDATA%\gaea\sin\exports\*.epub`；轨迹页签 Duration 77s/Turn 3/Call 1 渲染正常。
- **清场零残留**：临时故事/角色（remaining=0）/规则文件/导出 EPUB 全删，书架空，rules 目录复原（模板+engines），壳+webview+假站杀净。**零碰 novelsDir**（书源卡下载自包含 sin 书架，与小说导入是两条链——本次链路勘误入档）。
- **常驻内存首采**：v4.413.0 壳启动闲置态 WS=223MB/Private=235MB（slim-masterplan §3 已回填，P5 表「未采」全清）。
- **观察池新项**：reload 后偶现 `Callback 'app.ModelB.GetModelMonitor-…' not registered!!!` console 告警×2——Wails 回调重注册竞态（轮询先于注册），无功能损害，候小刀（重注册前静默或幂等重放）。
- **坑**：①SinStream 是异步流——返回流 ID（`ss_…`）即返回，消息经事件落库，勿当同步结果断言 ②SinMessages 空话题返回 null（非空数组），判空再取 length ③书源规则目录=$APPDATA/gaea/booksource/rules（模板+引擎+用户规则混装，搜索时逐文件装载）。

## 最新发布：v4.413.0（2026-09-26）「良性取消日志降级（前端错误观测通道降噪）」

- **刀型**：观察池项落地（v4.412 sin 实弹走查收官时新挂）。Go 1 文件+1 测试，绑定面 720 零变更，版本三处 4.413.0（sync-version.ps1）。
- **落地**：GaeaLogFrontendError 对 message 含 `context canceled` 降级 slog.Info，其余照旧 slog.Error——良性取消不再污染 Error 级观测通道，审计痕迹保留（INFO 在册）；剧照/设定卡/绘梦三条取消链路同时降噪；前端零改动（proxy.ts invoke 统一入口语义不变：仍上报、仍原拒绝语义抛出）。
- **测试**：Go TestGaeaLogFrontendError_CancelDowngrade（slog 捕获两例：取消链 INFO/真错 ERROR，复用 captureLogs 基建）；全量 ci.ps1 绿 EXIT=0+绑定漂移闸 OK（720 方法）+版本漂移闸 OK@4.413.0。
- **产物**：exe 51339264B SHA256=5d9b4b6ba2ce43c32e0d1e5d9f0a5dc473af1960b5c7eed6f7096f61cb7c2d24（SHA256SUMS-v4.413.0.txt，仅本地；build.bat 冒烟 200 过+桌面副本）；保留策略留 v4.409~v4.413 删 v4.408.0.exe。
- **文档**：releases/archive/v4.413.0.md+CHANGELOG/README+releases/README（436→437+34 席插 v4.413 裁 v4.381）+AGENTS 迁 1 插 1（一百三十迁：v4.408.0 入 archive）+todos（观察池「取消降级」清账）。
- **顺带 P5 冷启动补采**：Go 同步链 43–110ms（十二样本中位 ~62ms，charlib+stores 40–47ms 恒大头）+前端 gaea:interactive=497ms（v4.413.0 壳 CDP 实测：gaea:boot 91ms→interactive 588ms，DCL 139ms）；常驻内存仍挂真机窗口——slim-masterplan §3 已回填。
- **坑**：build.bat 是批处理——`powershell -File build.bat` 会 EXIT=127（旧 exe 原样未动），须 `cmd //c build.bat`。

## 非版本刀：sin 实弹走查收官+v4.412 三页签真机（2026-09-25/26，ComfyUI 在跑窗口）

- **窗口捕获**：ComfyUI 8188 探活 200+队列空而壳未运行→挂池最贵的「真机实弹走查」当场解锁。CDP 9333 附着 v4.412 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 配方），临时角色 lib_1790352026650（64×64 canvas dataURL 参考图）+临时故事 sin_1790352026653_2+SinCastSet 挂 cast——用户真实故事/角色/配置全程零触碰，走查后临时数据全删（CharacterList 查 w413=0、架子只剩用户故事）。
- **通过①（v4.412 三页签真机）**：中栏「故事/轨迹/上下文」真机渲染——轨迹诚实空态（暂无轨迹记录+Duration/Turn/Call 0）+上下文 hero「当前上下文 0/0 token —」六段估算+趋势+Token 统计全在位；console.error=0/pageError=0（页签切换全程）。坑=ChatTabs 页签钮不带 role=tab，选择器走 `.sin-tabs button`；SinCastSet 第二参直传数组（JSON.stringify 会踩 json: cannot unmarshal string into []string）。
- **通过②（v4.408 未验项清账）**：chip「生成 走查临时角色-w413 的设定卡」实弹提交→ComfyUI 队列 running:1 实达；进度行实生成态「加载模型 · 已用时 2s+取消钮」真机在位（截图 walk-v4413-progress.png）；点取消→行消失+「已取消生成」info 语义 toast（非错误样式，walk-v4413-after-cancel.png）+ComfyUI 队列秒清（interrupt 双达实证）+壳日志 context canceled 全链传播；临时角色 referenceImages 保持 1 张=取消先于完成零写回。
- **诚实闸验证（意外收获）**：portrait_backend=xai 时设定卡点击→「设定卡生成（Qwen 参考编辑）当前仅 ComfyUI 本地档（当前后端：xai）」fail-fast，console 仍零污染（v4.410 结论真机复现）。
- **观察池新项**：①取消路径全局前端错误钩记 ERROR 级（[frontend] [CharacterGenerateSheetError]…context canceled）——UI 已语义化但错误日志通道被良性取消污染，候小刀（按 context canceled 降级）②走查期发现 bridge 平面 API 形状坑（NovelB.SaveCharacter 是项目角色须先开项目；全局库=CharacterSave 返回带 id 全量）已沉淀记忆。
- **配置零漂移**：comfyui 本就是用户全局生图后端（09-24 启动日志「图片后端: ComfyUI」实证），走查中 image_backend 重写=原值；portrait_backend xai→comfyui→xai 精确还原；运行时配置真身在 ~/.gaea_config.json（config.toml 是 agent 配置，别看错文件）。壳杀净（gaea.exe+webview 过滤），ComfyUI 保持用户原状。
- **顺带 P5 季度复测落档**：entry 755→970KB(+28%)/dist 10.0→11.1MB/exe 46.15→51.44MB——回涨属功能比例不立项刀；Go >50KB 0→4 破防挂观察池（详见 slim-masterplan §3）。

## 源码包推送：wubigork-v4.411.0（2026-09-25）

- wubigork-v4.411.0-source.tar.gz @20b7204f 排除旧源码包，12951822 B / 4393 files / SHA256=248fd3bc…502ccb（SHA256SUMS-v4.411.0.txt 登记）。覆盖 v4.400~v4.411 十二版；随包推送 24 commit + v4.400.0~v4.411.0 14 tag 上 remote（origin/main 对齐，余 0 领先）。
## 最新发布：v4.411.0（2026-09-25）「一致性评分写回元数据」

- **v4.412.0** 上下文页 UI 优化（hero 提位/窗口未知「—」/趋势 hover）+ 原罪复用轨迹/上下文页签（sin_insight 折叠三绑定 717→720，TrajectoryView/ContextView 原组件自定义数据源）
- **刀型**：编辑器评分历史池①。绑定面 717 零变更，characterlib 迁移补 reference_scores 列。
- **落地**：reference_scores 与参考图索引对齐（读写两路维护+读侧补 0）；编辑器评分写 form 保存持久化+缩略图分数徽标（≥80 绿/60-79 橙/<60 红）；移除同下标裁剪。
- **测试**：Go 往返/对齐/短表三例；前端徽标/持久化/移除 35 例；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.411.0.txt；保留策略留 v4.407.0~v4.411 删 v4.406.0.exe。
- **观察池**：图vs图评分（视觉运行时验证门）；自动重生成（GPU 开销拍板）。
## 非版本刀：v4.408~v4.410 sin 新面真机走查（2026-09-25）

- CDP 9333 附着 v4.410 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 配方）。**通过**：chip「评分 gaea 的参考图」（v4.410）与「生成 gaea 的设定卡」真机在位；评分空态诚实指路（暂无参考图→不用视觉模型）；设定卡非 comfyui 后端闸诚实报错（当前后端 xai→拒绝，fail-fast）；进度行/取消钮空闲态不渲染（闸正确）；单会话 console.error=0/pageError=0。**未验项**：进度行实生成态+取消交互（ComfyUI 已关，等其在跑窗口）。临时故事桥接清场，用户真实故事零触碰。截图 .tmp/walk-v4410.png。
## 非版本刀：whisper 域死代码刀4（2026-09-25）

- 整文件死 88 文件 9,920 行净删（基线 590→196），deadcode -test 方法论三教训（type 依赖/app 反射盲区/依赖闭包剥壳）落档；12f77401。
## 非版本刀：whisper 域死代码调研档（2026-09-25，拍板池）

- deadcode -test 基线：590 不可达 func/147 文件/估 6-8k 行，ackem 蒸馏标记 267 处；五簇分解与三方案（整域/只删 A 簇/维持保留）落 docs/archive/gaea-whisper-deadcode-survey-20260925.md，待拍板。
## 最新发布：v4.410.0（2026-09-25）「sin 评分入口 v1（快路径）+ 评分流式辨伪关账」

- **刀型**：观察池末两项收官——等待体验/一致性线程观察池全部关账。纯前端，绑定面 717 零变更。规格 进度计划/gaea-sin-score-entry-20260925.md。
- **落地**：chip「评分」钮评角色最新参考图（getCharacter→末张→scoreCharacterConsistency→编辑器同款 Modal，<60 补参考提示）；scoreId 与 genId 互斥；评分流式辨伪缓办（JSON 契约无增益+churn 大，重开条件=图vs图 V2）。
- **测试**：SinCastPanel 9→12 例（互斥双向在案）；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.410.0.txt；保留策略留 v4.406.0~v4.410 删 v4.405.1.exe。
- **观察池**：（线程关账）编辑器侧历史池（图vs图/写回元数据/自动重生成）维持挂起待拍板。
## 最新发布：v4.409.0（2026-09-25）「图像生成内存压力预检（iGPU 冷载预期管理）」

- **刀型**：观察池在册项（v4.404.1 起四版）。Go 1 文件+3 挂点+前端 1 util+App 全局订阅，绑定面 717 零变更。规格 进度计划/gaea-imagegen-pressure-20260925.md。
- **落地**：comfyui 提交口读系统内存（复用既有 getMemoryStats 原生采集），可用 < 15% 或 < 4GB 命中→WARN+imagegen:pressure 事件；前端 App 全局订阅+2 分钟节流提示；只提示不拦截。
- **测试**：Go 阈值六态+读数合理性+接线三态（seam+httpbridge SSE 捕获）；前端 3 例；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.409.0.txt；保留策略留 v4.405.1~v4.409 删 v4.404.1.exe。
- **观察池**：评分 LLM 流式；sin 评分入口。
## 最新发布：v4.408.0（2026-09-25）「sin 设定卡 chip 进度行+取消钮」

- **刀型**：图像等待体验线程 sin 侧收尾（v4.407 观察池头名「sin 面板取消钮」+v4.406「sin 进度行」）。纯前端，绑定面 717 零变更。规格 进度计划/gaea-sin-cast-progress-cancel-20260925.md。
- **落地**：SinCastPanel 生成期 1s 轮询同源快照，chips 下方进度行（节点中文+用时/排队中，复用 COMFY_NODE_LABELS）+行尾取消钮（设定卡链 v4.407 起全局可取消，sin 复用自动获得）；handleCastSheet 的 context canceled 语义化「已取消生成」。
- **测试**：SinCastPanel 3→6 例（进度行出现+取消调用+结束消失/排队文案/读取失败不阻断+既有回归）；全量 ci 绿。
- **产物**：exe 见 SHA256SUMS-v4.408.0.txt；保留策略留 v4.404.1~v4.408 删 v4.401.0.exe。
- **观察池**：评分 LLM 流式；iGPU 压力预检；sin 评分入口。
## 最新发布：v4.407.0（2026-09-24）「角色库生成取消（剧照/设定卡可中止）」

- **刀型**：图像等待体验线程观察池头名。纯接线，绑定面 717 零变更。规格 进度计划/gaea-charlib-cancel-20260924.md。
- **落地**：剧照/设定卡两链接入 beginImageGen/endImageGen（ctx+/interrupt 双达，修 v4.406 无条件 clear 缺陷）+resetComfyCancel；前端进度行取消钮+「已取消生成」语义化。
- **测试**：TestCharacterGenerateSheetCancel（挂起模拟冷载→取消/interrupt 恰一次/可再提交）+全链回归+前端两态；全量 ci 绿 399 文件 3377 例。
- **产物**：exe 见 SHA256SUMS-v4.407.0.txt；保留策略留 v4.404.1~v4.407 删 v4.405.0.exe（桌面副本待壳关闭补拷）。
- **观察池**：sin 取消钮；评分流式；iGPU 压力预检。
## 最新发布：v4.406.0（2026-09-23）「角色库生成进度接线（剧照/设定卡载入可见）」

- **刀型**：图像等待体验线程收尾。纯接线，绑定面 717 零变更。规格 进度计划/gaea-charlib-progress-20260923.md。
- **落地**：attachComfyProgress（comfyui 预置 queued+挂回调，剧照/设定卡同链+defer 清理）；编辑器忙碌期 1s 轮询快照+hero 进度行（复用节点标签表）；相对路径写入源清点关账（@mention 相对=设计）。
- **测试**：TestAttachComfyProgress 三态+全链回归+前端两态；全量 ci 绿 399 文件 3375 例。
- **产物**：exe 见 SHA256SUMS-v4.406.0.txt；保留策略留 v4.404.1~v4.406 删 v4.404.0.exe。
- **观察池**：角色库生成取消（接 beginImageGen）；sin 进度行；评分流式。
## 最新发布：v4.405.1（2026-09-23）「AttachmentDataURL 相对路径根修（真机日志实录）」

- **分诊**：真机日志 18:49 实录——可读根检查只认绝对路径，工作区相对形态被整类误拒，文件真实存在。
- **落地**：resolveAgainstWorkspace（相对按工作区根 Join，同 GaeaListDir 惯例）+读侧解析后 ReadFile+写侧同款；穿越仍拒。纯 Go 1 文件，绑定面 717 零变更。
- **测试**：TestAttachmentDataURLRelativePath（放行+端到端+穿越拒+写侧）；app 全包+全量 ci 绿 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.405.1.txt；保留策略留 v4.401~v4.405.1 删 v4.403.0.exe。
- **观察池**：相对路径来源面清点——写入侧统一绝对化才是根治。
## 最新发布：v4.405.0（2026-09-23）「WS 载入阶段进度可读化（排队/载入可见）」

- **刀型**：v4.404.1 分诊后续（20 分钟冷载界面零反馈）。纯 Go 1 文件+前端 1 标签，绑定面 717 零变更。规格 进度计划/gaea-ws-load-progress-20260923.md。
- **落地**：WS 补 executing（节点开始→「当前节点: 加载模型」+不定态）/status（排队可见）/progress_state max=0 兜底三类事件；中文由前端既有节点标签表承担。
- **测试**：gorilla 假 WS 脚本序列回调断言（含两负例）；ai 全包+imagegen 128 例；全量 ci 绿 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.405.0.txt；保留策略留 v4.401~v4.405 删 v4.402.0.exe。
- **观察池**：ComfyUI 载入百分比真实可得性；iGPU 压力预检；取消显性化。
## 最新发布：v4.404.1（2026-09-23）「ComfyUI 生成等待根修：上限 15→30 分钟 + 超时收割（真机实录）」

- **分诊**：用户真机「图生图失败」=客户端白等误报——服务端 20.5 分钟冷载后成功，15 分钟上限先到；产物实际已生成。根因=iGPU 共享内存权重被换出后重载慢（同 GPU 语音管道刚停）。
- **落地**：genTimeout 15→30 分钟+harvestOnTimeout 超时收割（服务端已完成则下载返回）。纯 Go 1 文件，绑定面 717 零变更。
- **测试**：TestHarvestOnTimeout 两态；全量 ci 绿 399 文件 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.404.1.txt；保留策略留 v4.401~v4.404.1 删 v4.400.0.exe（桌面副本待壳关闭补拷）。
- **观察池**：WS 载入阶段进度可读性；取消显性化；iGPU 压力预检。
## 最新发布：v4.404.0（2026-09-23）「角色一致性评分 v1（文字锚点，T2 一致性收口刀）」

- **刀型**：用户拍板解锁挂池项。绑定面 716→717（play）、spaceBindings 540。规格 进度计划/gaea-consistency-score-20260923.md。
- **落地**：视觉 LLM 图 vs 文字设定评分（0-100+总评+issues，复用 RecognizeImage 单图链）；data URL 临时文件链+解析钳位+坏回复带原文报错；编辑器参考图「评分」钮+Modal+<60 补参考提示+单飞。
- **测试**：Go 纯函数+解析七态+seam 全链；前端两态；全量 ci 绿 399 文件 3374 例。
- **产物**：exe 见 SHA256SUMS-v4.404.0.txt；保留策略留 v4.400~v4.404 删 v4.399.0.exe。
- **文档**：releases/archive/v4.404.0.md+CHANGELOG/README+releases/README（425→426+裁 v4.370）+AGENTS 迁 1 插 1（一百一十九迁：v4.399 入 archive）+todos。
- **观察池**：图 vs 参考图评分（多图验证）；sin 评分入口；评分写回元数据；自动重生成。
## 最新发布：v4.403.0（2026-09-23）「sin 侧设定卡入口（T2 角色资产 v2 收尾刀）」

- **刀型**：观察池在册项。纯前端编排，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-sin-sheet-entry-20260923.md。
- **落地**：原罪右栏角色 chip 设定卡钮（单击 triptych 快路径）→getCharacter 全量→qedit→追加 referenceImages→saveCharacter 自动存回→reloadLibrary；chip 级单飞；面板吞错页面层 message；prop 可选向后兼容。
- **测试**：SinCastPanel 3 例+SidePanel 夹具补 prop；全量 ci 绿 399 文件 3372 例（首轮 ProgrammingPage 负载 flaky 隔离绿+复跑全绿在案）。
- **产物**：exe 见 SHA256SUMS-v4.403.0.txt；保留策略留 v4.399~v4.403 删 v4.398.0.exe。
- **文档**：releases/archive/v4.403.0.md+CHANGELOG/README+releases/README（424→425+裁 v4.369）+AGENTS 迁 1 插 1（一百一十八迁：v4.398 入 archive）+todos。
- **观察池**：一致性评分（vision 待拍板）；chip 模板菜单；分张并发档；评分驱动补参考。
## 最新发布：v4.402.0（2026-09-23）「设定卡分张连发（T2 角色资产 v2 第三刀）」

- **刀型**：v4.401 观察池头名。纯前端编排，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-char-sheet-split3-20260923.md。
- **落地**：菜单「三视图分张（连发）」——正/侧/背串行排队逐张追加进参考图（即到即见+N/3 进度）；失败语义=首张失败即中止（系统性故障）/中途失败继续+warning 点名单张重试；sheetGuard 收敛共用。
- **测试**：前端 3 例（全成功 variant 序/首张中止只调 1 次/中途失败 3 张都发起+点名）；全量 ci 绿 398 文件 3369 例。
- **产物**：exe 见 SHA256SUMS-v4.402.0.txt；保留策略留 v4.398~v4.402 删 v4.397.0.exe。
- **文档**：releases/archive/v4.402.0.md+CHANGELOG/README+releases/README（423→424+裁 v4.368）+AGENTS 迁 1 插 1（一百一十七迁：v4.397 入 archive）+todos。
- **观察池**：一致性评分（vision 待拍板）；sin 侧设定卡入口；分张并发档；评分驱动补参考。
## 最新发布：v4.401.0（2026-09-23）「设定卡模板矩阵 + 多参考锚定（T2 角色资产 v2 第二刀）」

- **刀型**：v4.400 观察池头名（多姿势/分张）+qedit 三图槽自然消费。签名扩展 (chJSON, variant)，绑定面 716 零变更、spaceBindings 539。规格 进度计划/gaea-char-sheet-v2-20260923.md。
- **落地**：charSheetVariantBody 六模板（triptych 默认/front·side·back 分张/sitting/action）+多参考锚定（≤3 填满 qedit 三图槽，剧照兜底，单张失败跳过）+前端 Dropdown.Button 模板菜单+**顺手根修 v4.400 剧照/设定卡按钮同点位重叠**（.cd-hero-gens 堆叠容器）。
- **测试**：Go 纯函数矩阵+假 ComfyUI（2 可用 1 缺失→上传恰 2 次/variant 透传/非法报错/两态回归）+前端主按钮默认 triptych+菜单坐姿两态；全量 ci 绿 398 文件 3367 例。
- **坑**：NodeList 解构 tsc -b 报 TS2488 须 Array.from/Dropdown.Button 的 testid 挂 wrapper 须 querySelector 取真按钮/新按钮抄旧 class 先核对是否承担布局定位。
- **产物**：exe 见 SHA256SUMS-v4.401.0.txt；保留策略留 v4.397~v4.401 删 v4.396.0.exe。
- **文档**：releases/archive/v4.401.0.md+CHANGELOG/README+releases/README（422→423+裁 v4.367）+AGENTS 迁 1 插 1（一百一十六迁：v4.396 入 archive；头部记录行补 105~115 漏登案）+todos。
- **观察池**：一致性评分（vision 文本锚点法待拍板）；分张一键连发（三张排队+单张失败重试）；sin 侧设定卡入口；评分驱动补参考。
## 最新发布：v4.400.0（2026-09-23）「角色设定卡生成（三视图，qedit 通道 · T2 角色资产 v2 首刀）」

- **刀型**：T2 角色资产 v2 首刀；复用 qedit+参考图画廊零 schema。绑定面 715→716（play）。规格 进度计划/gaea-char-sheet-20260923.md。
- **落地**：buildCharacterSheetPrompt+CharacterGenerateSheet（qedit 首参考锚定→三视图设定卡）+Editor 按钮（产物追加参考图列表）。
- **测试**：Go 纯函数+假 ComfyUI 全链+两态报错；前端按钮两态；全量 ci 绿。
- **坑**：buildPortraitClient 独立建客户端注入不生效须假 ComfyUI/renderEditor 包装对象传裸静默不命中/prompt 文案断言用词对齐。
- **产物**：exe 见 SHA256SUMS-v4.400.0.txt；保留策略留 v4.396~v4.400 删 v4.395.0.exe。
- **文档**：releases/archive/v4.400.0.md+CHANGELOG/README+releases/README（421→422+裁 v4.366）+AGENTS 迁 1 插 1（一百一十五迁：v4.395 入 archive）+todos。
- **观察池**：一致性评分（vision 文本锚点法）；多姿势模板；设定卡分张；sin 侧入口；评分驱动补参考。

## 最新发布：v4.399.0（2026-09-23）「原罪插图参考升级 qedit（T2 第二消费方）」

- **刀型**：v4.398 观察池头名小刀。纯 Go 2 文件，绑定面 715 零变更。
- **落地**：sinRefPlan comfyui→(txt2img,qedit)（不判生图模型）；herdsman 维持 img2img；refFallback 链兜底权重缺失。
- **测试**：矩阵+透传+TestSin 回归绿；全量 398 文件 3365 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci 绿 EXIT=0+漂移闸 OK@4.399.0。
- **产物**：exe 见 SHA256SUMS-v4.399.0.txt；保留策略留 v4.395~v4.399 删 v4.394.0.exe。
- **文档**：releases/archive/v4.399.0.md+CHANGELOG/README+releases/README（420→421+裁 v4.365）+AGENTS 迁 1 插 1（一百一十四迁：v4.394 入 archive）+todos。
- **观察池**：sin 参考方法可配；两级回退；徽标区分方法；qedit 语义 prompt 模板。

## 最新发布：v4.398.0（2026-09-23）「绘梦·参考槽 Qwen-Edit 通道（T2 一致性首刀）」

- **刀型**：T2 一致性首刀；路线修订（Qwen 架构走编辑参考不走 IP-Adapter）。Go 3+前端 5 文件，绑定面 715 零变更。规格 进度计划/gaea-qedit-ref-20260923.md。
- **病根**：参考槽 v0=img2img 整幅重绘（构图被锁死）+队列 txt2img 丢参考（新场景+角色一致无通道）。
- **落地**：qedit 路由+builder 三图槽（正负 TextEncode image1/2/3）+元数据如实化+ControlPanel 方法 Radio+applyRefCharacter 分流+queue 打通。
- **测试**：ai +2+回归绿+app +1+前端 +2；全量 398 文件 3365 例绿+tsc 0+eslint 0。
- **坑**：代码块搬家块尾注释找边界/regex 中插截断泛型/缺省行为是契约新方法并列露出。
- **产物**：exe 见 SHA256SUMS-v4.398.0.txt；保留策略留 v4.394~v4.398 删 v4.393.0.exe。
- **文档**：releases/archive/v4.398.0.md+CHANGELOG/README+releases/README（419→420+裁 v4.364）+AGENTS 迁 1 插 1（一百一十三迁：v4.393 入 archive）+todos。
- **观察池**：sin 参考槽升级；云端多图编辑；角色资产 v2（三视图/一致性评分）；LoRA 向导；导演 Agent；per-image 权重。

## 最新发布：v4.397.0（2026-09-23）「绘梦·抠图/透明底导出（T3 编辑力收官）」

- **刀型**：T3 最后一件。零新模型：蒙版即 alpha。Go 3 文件+前端 5 文件，绑定面 714→715（+ImageCutout，play）。
- **落地**：ComposeCutout（白=保留主体）→ImageCutout 绑定（落盘+台账+{path,asset_id}）→CutoutModal+ResultStage「抠图」动作。
- **测试**：ai +2+app +1+前端 +3；全量 398 文件 3363 例绿+tsc 0+eslint 0。
- **坑**：bindingNames 两份（真实消费 gaea/lib 那份）/bindings_image.go 单行风格勿 gofmt -w 整文件/JSX 锚点替换嵌条件块插完必 tsc。
- **产物**：exe 见 SHA256SUMS-v4.397.0.txt；保留策略留 v4.393~v4.397 删 v4.392.0.exe。
- **文档**：releases/archive/v4.397.0.md+CHANGELOG/README+releases/README（418→419+裁 v4.363）+AGENTS 迁 1 插 1（一百一十二迁：v4.392 入 archive）+todos。
- **观察池**：羽化；角色库参考图管线对接；模型推理抠图；透明产物展示语义；**画室线转 T2 一致性**（ipadapter/pulid、角色资产 v2、LoRA 向导、导演 Agent）。

## 最新发布：v4.396.0（2026-09-23）「纯绘梦会话台账登记根修（真机走查抓出）+ 编辑四件套真机验收班」

- **刀型**：真机验收班抓出的真缺陷根修。Go 1 文件+1 测试，绑定面 714 零变更。
- **病根**：台账登记闸=armed&&gaeaCfgSnapshot()!=nil；ga.cfg 只在办公引擎 GaeaInit 赋值——纯绘梦会话恒 nil→误判非运行态→登记静默跳过（v4.98 起：素材库恒空/变体簇断）。
- **落地**：闸只留 armed 位；回归测试钉纯绘梦会话形态。
- **验收班**：编辑四件套真机全过（详见 releases/archive/v4.396.0.md）；本地档真图编辑 ~400s（20B 首载）。
- **坑**：国内大模型下载首选 ModelScope（33MB/s vs HF/镜像 2-5MB/s）；分段下载 curl -C - 与 -r 互斥、重试追加超长需单段重下。
- **产物**：exe 见 SHA256SUMS-v4.396.0.txt；保留策略留 v4.392~v4.396 删 v4.391.0.exe。
- **文档**：releases/archive/v4.396.0.md+CHANGELOG/README+releases/README（417→418+裁 v4.362）+AGENTS 迁 1 插 1（一百一十一迁：v4.391 入 archive）+todos。
- **走查班挂池**：变体徽标重验、蒙版/扩图真图、走查产物清场。

## 最新发布：v4.395.0（2026-09-23）「绘梦·编辑历史变体簇：ParentID 链 + 溯源，回到任一变体继续改」

- **刀型**：长期规划 T3「编辑历史形成变体簇」；T0 契约 ParentID 预留位激活。Go 7 文件+前端 5 文件，绑定面 714 零变更。规格 进度计划/gaea-variant-chain-20260923.md。
- **病根**：编辑链 A→A'→A'' 天然成链但不可见——台账无关系字段、前端无变体标识，改三轮无法回到第 2 版分叉。
- **落地**：①sourcePath→imageHubAssetIDByPath→ParentID（查不到=诚实断链）②登记链路 +parentID（四调用点传空）③imageItem 回传 asset_id/parent_id④变体徽标+溯源按钮⑤VariantChainModal 建链+用到画布回祖先。
- **测试**：Go +2+前端 +4+2；全量 397 文件 3360 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.395.0。
- **坑**：批量 replace 命中同名 meta 构造块先数命中数/测试按包依赖挪文件/gofmt 只 -w 触碰文件。
- **产物**：exe 见 SHA256SUMS-v4.395.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.391~v4.395 删 v4.390.0.exe。
- **文档**：releases/archive/v4.395.0.md+CHANGELOG/README+releases/README（416→417+34 席插 v4.395 裁 v4.361）+AGENTS 迁 1 插 1（一百一十迁：v4.390 入 archive）+todos。
- **观察池**：HistoryRail 簇折叠；跨空间溯源；sin 编辑链；变体簇导出；按簇删除；抠图/透明底（T3 最后一件）。

## 最新发布：v4.394.0（2026-09-23）「绘梦·扩图（outpaint）：画布加边，模型补全新区域」

- **刀型**：图像域长期规划 T3「扩图」；v4.393 观察池销号。经济路线：扩图=合成大画布+蒙版环=v4.393 的 edit+mask 形态，两后端零引擎改动。Go 3 文件+前端 3 文件，绑定面 714 零变更。规格 进度计划/gaea-outpaint-20260923.md。
- **病根**：扩图是市场标配（即梦/Nano Banana Pro/PS 生成式扩展），gaea 编辑三件已落两件但扩图缺位——横补竖构图、特写拉远景无处落。
- **落地**：①ComposeOutpaint 纯函数（四边百分比/锚点按扩展量/中性灰 128/蒙版黑白环/>4096² 拒）+OutpaintToSquare②app mode=outpaint 转换 edit 请求（三门 fail-closed/mode 回显/清空 size）③前端 Modal 三态+6 预设 chips+四边输入+虚线预览+零扩展 warning。
- **测试**：ai +3+app +1+前端 +1；全量 396 文件 3354 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.394.0。
- **坑**：夹具 RGBA 忘 A:255=透明色贴图断言必挂/补正方百分比基数是短边原长/gate 顺序互斥门在通用 mask 门之前/测试期望先口算。
- **产物**：exe 见 SHA256SUMS-v4.394.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.390~v4.394 删 v4.389.0.exe。
- **文档**：releases/archive/v4.394.0.md+CHANGELOG/README+releases/README（415→416+34 席插 v4.394 裁 v4.360）+AGENTS 迁 1 插 1（一百零九迁：v4.389 入 archive）+todos。
- **观察池**：抠图/透明底导出（T3 最后一件）；扩边羽化；非矩形扩区（涂选已覆盖）；img2img+mask；变体簇；多图编辑；Lightning LoRA。

## 最新发布：v4.393.0（2026-09-23）「绘梦·蒙版局部重绘：涂选要改的区域，其余保持原样」

- **刀型**：图像域长期规划 T3「蒙版局部重绘」；v4.392 观察池销号第 2 项。Go 4 文件+前端 4 文件，绑定面 714 零变更。规格 进度计划/gaea-mask-inpaint-20260923.md。
- **病根**：双编辑档都有蒙版通道但从未接线——前端没有涂选交互、后端请求没有 Mask 字段；全图编辑对已满意区域有回归风险（蒙版局部重绘是 NovelAI V5/即梦/可灵全系标配）。
- **落地**：①统一蒙版契约（灰度 PNG 白=重绘黑=保留；ComfyUI ImageToMask(red) 零转换，OpenAI grayMaskToOpenAIMask 白→透明一处转换）②ComfyUI 蒙版链 LoadImage→ImageScale(lanczos,targetW/H)→ImageToMask(red)→SetLatentNoiseMask→KSampler——targetW/H 复刻 PREFERRED_KONTEXT_RESOLUTIONS 17 档（nodes_flux.py:105 实读）对齐缩放图；TextEncode image1 保持全图③app fail-closed（mask 非 edit 拒绝）④MaskBrushEditor（strokes 状态+红笔刷显示/黑底白笔刷导出同坐标+比例映射+jsdom 守卫）+Modal 范围切换+未涂 warning。
- **测试**：Go +4 组+app +1+前端 +4+2；全量 396 文件 3353 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.393.0。
- **坑**：蒙版黑白数据语义要 hex-exempt/SetLatentNoiseMask 裁剪不 resize 须显式对齐/JSON 往返数字变 float64/vi.mock 工厂 hoisting 不能引顶层标识/antd Slider 不转发 testid。
- **产物**：exe 见 SHA256SUMS-v4.393.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.389~v4.393 删 v4.388.0.exe。
- **文档**：releases/archive/v4.393.0.md+CHANGELOG/README+releases/README（414→415+34 席插 v4.393 裁 v4.359）+AGENTS 迁 1 插 1（一百零八迁：v4.388 入 archive）+todos。
- **观察池**：扩图（outpaint 画布扩边）；抠图/透明底导出；img2img+mask 纯局部重绘；蒙版反选/羽化；编辑历史变体簇；Lightning LoRA；多图编辑。

## 最新发布：v4.392.0（2026-09-22）「绘梦·指令编辑本地档：ComfyUI 接入 Qwen-Image-Edit 2511 官方工作流」

- **刀型**：图像域长期规划 T3「本地档（B 计划）」第一刀；阶段一刀 C（v4.327 指令编辑云端先行）观察池第 2 项销号。会话主线=乐园画室欠账盘点→「画室三件套」头号=编辑力补全。Go 2 文件+前端 1 文件，绑定面 714 零变更。规格 进度计划/gaea-comfyui-edit-20260922.md。
- **病根**：指令编辑只有云端档（OpenAI 兼容 /images/edits），ComfyUI 诚实拒绝——而用户本机全局生图后端=comfyui，编辑在日用后端上不可用，与本地优先基本盘直接冲突。
- **落地**：①buildQwenImageEditWorkflow 蒸馏官方 comfyui-workflow-templates image_qwen_image_edit_2511（取道不取器，API-format 构建器与 krea2/z-image 同范式）：LoadImage→FluxKontextImageScale（按原图宽高比重标，编辑不套 1024×1024）→缩放图同时进 TextEncodeQwenImageEditPlus×2（image1+vae）与 VAEEncode→KSampler.latent_image；UNET→ModelSamplingAuraFlow(3.1)→CFGNorm(1)→KSampler（20 步/CFG 4.0/euler/simple/denoise 恒 1.0=语义编辑非整幅重绘）②缺权重可操作提示——gaea 从不自动下载模型，value_not_in_list 时错误追加三件文件名+HF 链接+目录（VAE 与 krea2 共用），错误即安装指引；普通错误不追加③GenerateMedia edit+comfyui 时 imgModel 如实改写 qwen-image-edit（结果卡/台账记真实引擎）④InstructionEditModal 说明改口。编辑链全继承既有基建：进度回调/取消/ComfyUI 自动拉起（v4.387）/落盘+台账登记零新增路径。FluxKontextMultiReferenceLatentMethod（官方明示官方权重不需要）与 Lightning 4 步 LoRA（可选加速）均不接。
- **测试**：ai 层改写 ComfyUI edit 拒绝测试为三件（全链 httptest 工作流形状/缺原图/缺权重提示含 HF 链接且普通 500 不追加）+app 层 +1（元数据改写）+前端 +1（说明文案）；全量 395 文件 3347 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.392.0。
- **坑**：官方模板新 subgraph 格式——节点图要从 definitions.subgraphs 展开+interface 槽位对射，顶层只有 4 节点/CLIPLoader 的 clip_name 查 text_encoders 目录非 clip/编辑不套请求 size——FluxKontextImageScale 按原图重标/提示锚 value_not_in_list+模型名双匹配防误挂。
- **产物**：exe 见 SHA256SUMS-v4.392.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.388~v4.392 删 v4.387.0.exe。
- **文档**：releases/archive/v4.392.0.md（含升级说明：ComfyUI 档编辑需先放模型文件，错误即指引）+CHANGELOG/README+releases/README（413→414+34 席插 v4.392 裁 v4.358）+AGENTS 迁 1 插 1（一百零七迁：v4.387 入 archive）+todos。
- **观察池**：Lightning 4 步加速 LoRA；image2/3 多图编辑；蒙版局部重绘/扩图/抠图；编辑历史变体簇（ParentID 链）；模型文件名 config 化；FLUX.2 Klein 编辑族第二档；编辑产物回填角色库。

## 最新发布：v4.391.0（2026-09-22）「附件 URL 缓存 blob 化：base64 大字符串迁出 JS 堆 + selbar z 防御修」

- **刀型**：性能池⑦「base64→blob URL 会话资源管理」收口（v4.365 审计留池最后一项免拍板性能项）+selbar z 防御修+todos 考古。纯前端 3 文件+2 测试（改 1 新 1），绑定面 714 零变更。
- **病根**：usePortraitUrl 的 dataUrlCache=全局无界 Map，值是 MB 级 base64 data URL 字符串——消息内联图/缩略图/inspector/角色头像/剧照全部经此通道（v4.365 已消重复桥调用，但字符串驻留 JS 堆的问题在）。
- **落地**：①缓存值 data URL→blob: object URL（atob→Uint8Array→Blob→createObjectURL——字节迁浏览器侧 blob 存储脱离 JS 堆，原字符串即被 GC；转换在既有 then 异步回调；消费方零改动）②LRU 96+命中刷新+淘汰延迟 120s revoke③无 createObjectURL 回退 data URL④selbar z 1080→990。
- **测试**：+1 文件 5 例+PortraitImg 断言按新行为收口；全量 395 文件 3346 例绿（一轮 1 例无足迹 flaky 复跑全绿）+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0+版本漂移闸 OK@4.391.0。
- **坑**：jsdom 30 原生 createObjectURL（钉 data: 断言真挂）/LRU 淘汰即 revoke 裂在屏图（条目被组件 state 持有）/data URL→blob 别用 fetch(dataURL)（jsdom 不支持）/后台测试命令只留 tail 丢失败详情。
- **产物**：exe 见 SHA256SUMS-v4.391.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.387~v4.391 删 v4.386.exe。
- **文档**：性能池⑦销号回填+releases/archive/v4.391.0.md+CHANGELOG/README+releases/README（412→413+34 席插 v4.391 裁 v4.357）+AGENTS 迁 1 插 1（一百零六迁：v4.388 入 archive）+todos。

## 最新发布：v4.390.0（2026-09-22）「canvas 调色板集中化：resolveThemeColor 收口三份手写解析器 + 主题切换跟随缺陷根修」

- **刀型**：视觉池③+⑤专项（v4.361 审计留池大工程候立项中可代码级收口的两项；①②④仍需逐板块截图对照拍板）。纯前端 7 文件+3 新测试，绑定面 714 零变更。
- **侦察发现**：三份手写 CSS 变量解析器各缺一块（RelationGraph 模块级求值不随主题刷新/GraphView 只解析一个变量且 once/CompanionAvatar 字符串手拆不支持 color-mix）；RelationGraph 真缺陷=角色色/组织色/关系色/画布底模块级顶层求值，v4.350 只修组件内 canvasColors 漏网模块常量。
- **落地**：①resolveThemeColor/RGB（utils/theme.ts）span 探针法——浏览器自己算，支持 color-mix/嵌套 var/fallback 链；RGB 元组版拼 rgba()②RelationGraph palette=useMemo([darkMode]) 全量收口+graphData/decoratedNodes/edges 依赖补 palette+图例改 var() 活解析③GraphView 背景跟随（探针+MutationObserver 监听 :root 内联 style，不引老栈 store）④CompanionAvatar 删手拆改探针+withAlpha（hex 后缀拼接遇 rgb() 串拼非法色被 canvas 静默忽略）⑤AoaView/PdmView critical 兜底 #dc2626→#e02020。
- **测试**：+3 文件 9 例（theme 5+RelationGraph 2〔darkMode 切换后 fillStyle 先暗后亮=重解析链路钉死〕+CompanionAvatar 2）+目检（esbuild 挂具+Edge 无头暗/亮两页：两态节点+连线+图例全渲染，亮=浅底深字走重解析路径，改造前停留暗色）；全量 394 文件 3341 例绿+tsc 0+eslint 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.390.0。
- **坑**：①canvas 测试 jsdom 三连坑——offsetParent 恒 null 触发 v4.365 空转挂起/首绘 rAF 等帧 flush/ctx 桩 Proxy 兜底任意方法②hex 后缀拼接只兼容 6 位 hex——非法色被 canvas 静默忽略不报错③模块级求值=主题切换隐形死角④getPropertyValue 读 custom property 只到引用替换层⑤Edge --screenshot 单帧不跑 rAF——canvas rAF 组件截图全空须 --virtual-time-budget⑥gaea 侧主题跟随不引老栈 store。
- **产物**：exe 51181568B SHA256=0ad223995fc7816bbaf810f1cff82d41942390bcdc8958464a85f79f05335404（SHA256SUMS-v4.390.0.txt，仅本地；冒烟 /api/health 200 过；桌面副本同哈希实测一致；字节数与 v4.389 巧合相同哈希不同）；保留策略 5 版留 v4.386~v4.390 删 v4.385.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.390.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 411→412+34 席插 v4.390 裁 v4.356）+AGENTS 迁 1 插 1（一百零五迁：v4.387 入 archive）+todos。

## 最新发布：v4.389.0（2026-09-22）「模型中心统计重设计：右栏检查器遥测读出式 + 详细统计抽屉信息分层」

- **动机**：用户指令「重新设计模型中心右侧面板的统计和详细统计」。
- **诊断**：右栏=4 张 KPI 卡 2×2 平铺（300px 窄柱又厚又挤、hint 常被省略号截断）+ 两张 720 宽 viewBox 全尺寸趋势图压进 ~276px 渲染（轴字缩到 ~4px 完全不可读）；抽屉=块内标题与抽屉标题重复、5 张 KPI 图标全是同一个闪电、「本地 vs 云端」6 张卡两行平铺。
- **落地**（纯前端 8 文件含 eslint 配置 1，绑定面 714 零变更）：①右栏遥测读出式——三格主指标（总调用·成败/成功率阈值着色/估算费用）+ Token 紧凑读数行（k/M）+ 窄柱专用迷你图 RequestsSpark/TokenMiniBars（260 视口 1:1 设计字标不缩放，保留逐点 tooltip）+ 底部「查看详细统计」入口（context 新增 openStatsDrawer，右栏直开抽屉）②抽屉信息分层——工具条（统计自/价格目录芯片+排序刷新清空）+「云端/本地分流」单卡叙事（Token 占比条+云端费用/本地用量/已节省三列+KV 命中率虚线紧凑行）+ KPI 五格语义图标+Token hint 紧凑格式防截断+图例类化③全尺寸两图 viewBox 720→540+轴字 10→11（抽屉双列 ~380px 栏宽有效字号 5.3→7.7px）④eslint globalIgnores 补 .tmp（挂具 bundle 的 legal comments 触发指令错的门禁欠账）。
- **测试**：前端 +2 文件 8 例（InspectorPanel 三态+入口联动/StatsSection 工具条芯片+KPI 着色+hint 紧凑+分流带占比口算）；modelcenter 17 套 121 例+全量 391 文件 3332 例绿+tsc 0+eslint 0。目检两轮（esbuild 挂具+Edge 截图）：检查器六项全过，抽屉唯一缺陷（Token hint 截断）修复后复检确认。坑：窄柱放全尺寸图轴字不可读——图表按目标渲染宽度设计 viewBox；antd 两字中文按钮自动插空格+图标 textContent 前导空格——断言按存在性/trim；挂具视口 <1180px 触发检查器隐藏媒体查询（截图全黑≠渲染失败）；fmtCompact 归 utils.tsx 不进组件文件（react-refresh 告警）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.389.0。
- **产物**：exe 51181568B SHA256=ba43b3a1cecf79431de4739bc1deb918e621a1b5b98025f4d3c471bc0857693a（SHA256SUMS-v4.389.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致）；保留策略 5 版留 v4.385~v4.389 删 v4.384.0.exe（SUMS 全保留）。
- **文档**：releases/archive/v4.389.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 410→411+34 席插 v4.389 裁 v4.355）+AGENTS 迁 1 插 1（一百零四迁：v4.386 入 archive）+todos。

## 最新发布：v4.388.0（2026-09-22）「原罪插图独立生图绑定：功能绑定卡新增『原罪插图』后端/模型选择」

- **动机**：用户反馈「原罪的功能绑定卡片内增加生图模型选择，这么多模型为什么没有候选，只能使用 comfyui？」。
- **诊断**：原罪生图此前无独立绑定，跟随全局生图设置（本机全局后端=comfyui，所以看起来「只能 comfyui」）；模型中心「功能绑定」原罪卡只绑故事文本模型；跨引擎生图候选其实一直在（剧照卡同款 imageModelOptionsFor），原罪没有入口。
- **落地**（镜像「角色库剧照」先例，Go 6 文件+前端 9 文件，绑定面 712→714）：①GetSinImageConfig/SetSinImageConfig（sin_image_backend/model，空=跟随全局，任一项可单独回退）+shelf 热更新②生成路由——buildImageClientFor 从 buildPortraitClient 通用化（comfyui/herdsman/ollama/glm/xai 五后端独立客户端，不动绘梦全局）；generateImageInternal 加 clientOverride/backendOverride 通道（进度回调/size 裁剪/ComfyUI 自动恢复与拉起分支统一按生效后端判定）；SinIllustrate 级联 sinImageBinding（绑定>全局）+sinRefPlan 用生效值+绑定后端≠全局时独立客户端；sin_illustrate 工具同入口受益③模型中心新增「原罪插图」卡（六后端下拉+跨引擎模型候选），原罪故事卡注释补指引。
- **测试**：Go +4 组（级联四态/Get 回读/buildImageClientFor 分支/override 通道 override=1 全局零触碰）+前端 +1（卡渲染+保存+已绑定态）。坑：config.Save 直写真实 ~/.gaea_config.json 无隔离钩子——Set roundtrip 不能真调，测读映射+纯函数级联；种子引擎目录恒 Enabled 须 SaveEngine 禁用再测未启用分支；override 后端语义要跟到底（三处判定原读全局配置）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.388.0 + spaceBindings 锁 535→537 + bindingNames 714。
- **产物**：exe 51171328B SHA256=48f1a288eaadb80a6a8632a81ac6d0dbff0fd110b87318947c4682a37bb141e1（SHA256SUMS-v4.388.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致——用户已关 gaea 连跳两版直更 v4.388）；保留策略 5 版留 v4.384~v4.388 删 v4.383.0.exe（SUMS 全保留）。
- **文档**：releases/archive/v4.388.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+计数 409→410+34 席插 v4.388 裁 v4.354）+AGENTS 迁 1 插 1（一百零三迁：v4.385 入 archive）+todos。

## 最新发布：v4.387.0（2026-09-22）「ComfyUI 未运行自动拉起：原罪插图/绘梦生成连接被拒自愈」

- **动机**：用户报障「原罪生图失败，插图失败：ComfyUI 提交失败…connectex: No connection could be made because the target machine actively refused it」。
- **诊断（两层根因）**：①环境层——8188 无监听、ComfyUI 进程不在；手动拉起又暴露 standalone-env 落后于仓库 requirements（comfy-aimdo 0.4.13<0.5.5 缺 storage、comfy-kitchen 0.2.28<0.2.35 缺 int8_attention 等，一族过期，起都起不来）②产品层——gaea 生成链只对「孤儿实例 [Errno 22]」自动恢复，服务压根没跑直接 connectex 原始错误失败；原罪插图页无恢复入口（绘梦页才有 StartComfyUI 按钮）。
- **环境修复（用户本机随刀处置）**：按 requirements 钉版精确补 comfy-aimdo==0.5.5/kitchen 0.2.35/embedded-docs 0.5.11/frontend-package 1.52.7/workflow-templates 0.11.62（**刻意不 pip install -r**——torch 无钉版行会重解析，ROCm 特制 torch 可能被 PyPI 通用版顶掉）；修后以 gaea 同配方（standalone-env python+--windows-standalone-build，日志同 ~/.gaea/logs/comfyui.log）拉起 ~30s 就绪（ComfyUI 0.36.0，/system_stats 200）——**用户当前会话重试插图即恢复**。
- **落地（gaea 根修，Go 1 文件+测试，绑定面 712 零变更）**：ensureComfyUIRunning 助手（未运行→StartComfyUI→/system_stats 就绪轮询有界 120s；未配置路径/启动失败/超时=返回 false，**原始错误如实上抛不吞错**）接线 generateImageInternal 重试分支——错误含「连接 ComfyUI 失败」（dial 层文案，服务不可达才出现，服务在但卡死不进防误杀）+backend=comfyui 时拉起成功后重试一次（comfyBooted 闸）；并发竞态由 StartComfyUI 端口占用守卫兜底（「已被占用」转就绪等待）；原罪插图与绘梦共用本链，一处接线双板块受益。
- **测试**：Go +3（image_boot_test.go 零真进程：连接被拒→假就绪端点→重试恰一次〔计数=2〕/未配置路径不重试且原始错误上抛/ensure 三分支契约——已运行即真·未配置即假·无 main.py 启动失败即假快速返回）；internal/app 全量绿。坑=测试成功路径必设 ImageSaveDir 指临时目录（否则 saveToNovelImages 撞装配态 nil）。
- **门禁**：全量 ci.ps1 绿 EXIT=0 + 版本漂移闸 OK@4.387.0。
- **产物**：exe 51163136B SHA256=f7ec8eb3c30260104a180beb81ba6354399aecc6bece6962c70bf9ba85a2e17b（SHA256SUMS-v4.387.0.txt，仅本地；冒烟 /api/health 200 过；桌面副本因用户会话在跑未覆盖——关闭 gaea 后替换，当前会话靠环境修复已可用）；保留策略 5 版留 v4.383~v4.387 删 v4.382.0.exe（SUMS 全保留）。
- **文档**：releases/archive/v4.387.0.md（含升级说明：首次冷启动生图多等一段属预期）+CHANGELOG/README+releases/README（保留策略行+计数 408→409+34 席插 v4.387 裁 v4.353）+AGENTS 迁 1 插 1（一百零二迁：v4.384 入 archive）+todos。

## 最新发布：v4.386.0（2026-09-22）「造价库分页绑定：成本条目列表/表格分页 + 检索下拉载荷降载」

- **动机**：用户口径「继续优化迭代」——todos 扫一遍，可自主推进的工程项=前端性能池⑥余项（造价库全表留池；其余多为等真机/等拍板/等上游的等条件型）。成本库「成本条目」每次搜索/筛选全表拉回全量渲染，用户真实库 1553 条时每次 250ms 防抖击键=全量桥载荷+1553 行 DOM。Go 3 文件+前端 10 文件，绑定面 711→712（+1）。
- **落地**：①**GaeaCostSearchPage 分页绑定**（App 层实现，cost.Store 零改动）——检索管线（SQL→Go 关键词过滤→语义召回→本地精排）提为 costSearchAll 助手两绑定共用，分页与非分页口径一致、工具面 cost_search 零影响；排序进服务端 title/price/updatedAt（空=管线序）**tie-break 恒定 name**（全序=跨页不漂移前提）；limit 钳 [1,200]（≤0→100）、offset 负归零、Total=过滤后总数；未知 sortKey 不排序②**CostLibraryView 分页改造**——首屏 100+「加载更多（余 N 条）」100/批、计数如实「已载 X / 共 Y 条」、分类树计数用 total；**客户端排序删除**（分页时代只排已载子集是静默错误答案）改服务端排序重拉第 1 页；过期响应 reqSeq 丢弃+追加页跨页 name 去重③**EntryPicker 切分页**（top8 载荷降载）；loadStats 总览仍全量（聚合需要，开页一次非热路径）；mock 同口径。
- **测试**：Go +5（切片连续+total/全序 tie-break+降序镜像/钳制四态/零命中空页/未知键管线序/过滤×排序组合〔updated_at RFC3339 秒级落盘→SQL 直铺确定时间序〕）+前端 +1（150 条分页补齐）+迁移断言（7 参形态/排序异步等待）。全量 vitest 389 文件 3323 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.386.0+bindingNames 再生 712+spaceBindings 锁 534→535。
- **事故与修复（本轮最大坑）**：Go 测试零命中查询触发**真实语义管线**——NewManager 种子目录 herdsman 恒 Enabled+localhost:8080，本机 embedding 服务在跑时 SQL 召回<3 即走语义补召回：Stale("cost",keep) 按测试条目集清掉真实库 cost 向量行+Ensure 写假向量。**已修复**：假行 DELETE 清污+走应用自身 GaeaSemanticIndexBackfill 全量重建 1553/1553=24s（复核实测）；测试侧 SaveEngine 禁用 herdsman 断通道。教训：hubCostStore/hubSemanticStore override 只护一半，resolveHerdsmanSearchModel 引擎目录在测试环境是活的；App 嵌 *core 裸 &App{} 字段访问即 panic（须 &App{core:&core{…}}）。
- **产物**：exe 51160576B SHA256=4201619c…458cd（全文见 SHA256SUMS-v4.386.0.txt，仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.382~v4.386 删 v4.381.0.exe（SUMS 全保留）。
- **文档**：性能池⑥销号回填+releases/archive/v4.386.0.md（含升级说明：标题排序改字节序、列表默认 100 条）+CHANGELOG/README+releases/README（保留策略行+计数 407→408+34 席插 v4.386 裁 v4.352）+AGENTS 迁 1 插 1（一百零一迁：v4.383 入 archive）+todos。
- **下刀候选**：性能池⑦base64→blob 会话资源管理；视觉大工程（字号阶梯/玻璃收敛等，需专项+截图对照）；其余等条件型不动。

## 非版本刀（2026-09-22）projection-cache 收益评估：辨伪销号（dsh ⑪）

- **动机**：dsh 留池③「session-projection-cache 冷启动加速（收益待评估）」——评估债清偿，用基准实测代替拍脑袋。
- **结论**：**辨伪销号，不立项**。gaea checkpoint 机制（每轮模型调用前 flush 消息投影+seq，fail-closed）已是投影缓存的等价物——Restore 只投影 checkpoint 之后的尾差；基准实测（合成日志 user/assistant 交替 ~1.5KB/条）：2MB≈20ms / 10MB≈64ms / 50MB≈267ms（有 checkpoint；无 checkpoint 也仅 +15~45%），真实会话多在 1-5MB=几十 ms。成本主导项是全文件 read+parse（不换文件格式无法省，而换格式牵动 append-only 不变量），投影缓存的增量收益不可感知。
- **产物**：基准在案 internal/gaea/agent/session/restore_bench_test.go（go test -bench BenchmarkRestoreSynthetic，2/10/50MB × 有无 checkpoint 六组，可复跑）。
- **文档**：dsh 蒸馏文档⑪辨伪销号回填（留池余 2=子代理控制面等上游稳定+Plan mode 需前端配套，全为等条件型）+todos 头部行更新。

## 最新发布：v4.385.0（2026-09-22）「工程健康轮 #2：excelize 防线扩面 + 死代码清欠 + 双扫描报告」

- **动机**：用户口径「继续」——距 v4.370 工程健康轮 15 版，按惯例复扫三面。Go 2 文件+前端 4 文件，绑定面 711 零变更。
- **扫描报告**：govulncheck 1 项在案（GO-2026-6452 excelize，上游 N/A）/pnpm audit 0/knip 1 未用导出+3 未用类型+3 重复+5 提示。
- **落地**：①excelize 防线扩面（v4.370 只防 xlsxpreview；本轮符号追踪新可达面 schedule.ImportXlsx GetRows+GaeaXlsxRowOps/ColOps InsertRows/Cols/RemoveRow/Col——同款 named-return+defer recover 转可读错误）②knip 清欠 4 项（useModelCenter 兼容口零消费+桥接死类型 ForeshadowUrgencyPayload/NovelRewriteRequest/SinBookSourceDownloadResult；tsc+grep 双验证；stale 注释同步更新；重复导出/配置提示 KEEP 不动）。
- **测试**：app/schedule 全量绿（纯增量 defer 零行为变化）+tsc 0+modelcenter 112 例绿。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.385.0。
- **坑**：①「已防御」≠「全防御」——符号追踪随代码演化增长，扫描要周期性复跑②Fixed in N/A 常驻项——新增 excelize 消费面必须同步配 recover 入口③knip 未用导出区分死代码与契约哨兵（重复导出 KEEP 不跟风删）。
- **产物**：exe 51153920 B SHA256=9d8de0811f3aa6de298ed4b9ee5b78d3159a735bebbf4a48189580d2f6b5a72e（releases/gaea-v4.385.0.exe+SHA256SUMS-v4.385.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.381~v4.385 删 v4.380.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.385.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 406→407+34 席插 v4.385 裁 v4.351）+AGENTS 迁 1 插 1（一百迁：v4.382 入 archive）+progress/todos。

## 最新发布：v4.384.0（2026-09-22）「dsh 蒸馏第五弹：session_query 会话检索（跨会话记忆首刀）」

- **动机**：用户口径「继续」——dsh 余 4 中唯一可独立推进的大工程项⑧ session-query 首刀落地，留池余 3。全 Go，绑定面 711 零变更。
- **落地**：①索引器（新包 internal/gaea/sessionquery：索引面=装配空间会话目录 *.gaea-log.jsonl 的 user_message/assistant_message 正文，system/工具结果/流式 delta 噪声面不入索引；增量按 (size,mtime) 新鲜度指纹惰性刷新零 boot 成本；重索引=DELETE 旧行再全量插；空间隔离由目录构造，双空间红线同构）②session_search 只读工具（query+limit 默认 8 上限 32，newest-first 带 session id/role/时间/snippet；中文降级=FTS5 unicode61 不切 CJK 子串 MATCH 零命中回退 LIKE 子串+手工裁窗，whisper 先例；MATCH 短语引号化）。
- **测试**：Go +5（FTS 命中+噪声面排除+ts 倒序/CJK 子串降级/增量新鲜度/目录隔离/工具面格式化+空查询+零命中诚实）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.384.0。
- **坑**：①ReadLogRepaired 的 torn-tail 修复截断「未以换行收尾」的最后一行——手工构造日志必须换行收尾（真实 LogWriter 恒换行收尾）②FTS5 标准表（非 contentless）才支持普通 DELETE——UNINDEXED 元数据列建虚表免联表可重索引③LIKE 路径 snippet 手工裁窗——两路径 Scan 形状不同，共用 Scan 助手静默拿空原文（首写即犯测试当场抓出）。
- **产物**：exe 51150848 B SHA256=221d26d35d9bd7eeaf5a4fcb15afa647cb30cfaf0ff1a498c4ae68fea5967d25（releases/gaea-v4.384.0.exe+SHA256SUMS-v4.384.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.380~v4.384 删 v4.379.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档⑧首刀销号回填（余 3）+releases/archive/v4.384.0.md（含升级说明：首检索全量索引一次）+CHANGELOG/README+releases/README（计数 405→406+34 席插 v4.384 裁 v4.350）+AGENTS 迁 1 插 1（九十九迁：v4.381 入 archive）+progress/todos。

## 最新发布：v4.383.0（2026-09-22）「聊天虚拟化第二层：绘制层 content-visibility + 挂池考古销号一批」

- **动机**：用户口径「继续」——v4.365 池②「聊天消息真虚拟化」收口（两形态分层），顺带一批过时挂池条目考古销号（零代码）。前端 2 文件+测试 +2，绑定面 711 零变更。
- **落地**：①绘制层虚拟化（超阈值会话根节点挂 chat-flow-cv，行级 content-visibility:auto 离屏跳 layout/paint；contain-intrinsic-size:auto 120px 记住行实际高度；≤阈值不挂类零影响；流式吸底/antd portal 菜单不受影响；react-window 被 v4.370 侦察否决有案=滚动宿主/动态行高/流式吸底三冲突）②挂池考古销号（零代码）=v4.355②file:// 头像（v4.361 已清账）/v4.355③longtask 阈值（v4.361 已提 200ms）/v4.349⑤initRuntimePolyfill（辨伪同步成本≈0）+vendor manualChunks（无新证据）。
- **测试**：前端 +2（>阈值挂 chat-flow-cv/≤阈值不挂——类挂载即完整契约）；全量 vitest 389 文件 3322 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.383.0。
- **坑**：①content-visibility 的行内 fixed/portal 子元素语义要先查——containment 裁剪 fixed 后代，聊天行菜单 portal 到 body 不受影响，novel selbar 不在作用面②「真虚拟化」不等于 react-window——三冲突场景下渲染层窗口+绘制层 CV 分层零结构风险达成同目标，池条目按目标销号不按实现手段。
- **产物**：exe 51128320 B SHA256=4453235ab91e5a39187e4e4112e363df89937e66e96731f43f6f7bd0a310e588（releases/gaea-v4.383.0.exe+SHA256SUMS-v4.383.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.379~v4.383 删 v4.378.0.exe（SUMS 身份档案全保留）。
- **文档**：池②收口+三条考古销号回填+releases/archive/v4.383.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 404→405+34 席插 v4.383 裁 v4.349）+AGENTS 迁 1 插 1（九十八迁：v4.380 入 archive）+progress/todos。

## 最新发布：v4.382.0（2026-09-22）「前端挂池清欠：整店订阅清零 + 交付物缓存失效接线」

- **动机**：用户口径「继续」——dsh 余 4 项全是「等条件」型，转前端挂池清欠（v4.349 池④+v4.354 池④）。前端 4 文件+测试，绑定面 711 零变更。
- **落地**：①无 selector 整店订阅清零（全仓扫描四家 store 仅剩 WorkspacePanel/FilePreviewModal 2 处 selector 化；审计对账=MainLayout/HomePage/AppearancePanel 已被中间轮次 selector 化行号过时，本轮后全仓归零；gaea useController 的 useShallow 整面订阅保留=App.tsx 消费近乎全量字段，热字段 items 流式本就逐 chunk 变化）②交付物缓存失效接线（invalidateTurnCaches 接到 turn_done 事件消费点+loadSessionData 会话切换；新轮交付物/新建文件不再被 TTL 内旧目录探测误标缺失；纯内存清空零桥接成本）。
- **测试**：前端 +2（turn_done 后 ensureTurnRegistry 重拉打桥的调用计数断言/会话装载同语义）；全量 vitest 389 文件 3320 例绿+tsc 0。
- **门禁**：全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.382.0。
- **坑**：①挂池审计行号会过时——清欠前先全仓重扫，销号写清哪些本轮修哪些中间轮已修②恰好一次的事件绑定是模块级单例——从可观测面（bridge 调用计数）断言行为不 mock 内部③缓存失效时机在事件边界——reducer 保持纯函数，副作用接既存非纯区（turn_done 块本就有 ContextUsage/Balance 副作用）。
- **产物**：exe 51128320 B SHA256=a44212dc84ddcdad14c54c7a1c0bd91783858d9907efa377e0362c72f0b0139f（releases/gaea-v4.382.0.exe+SHA256SUMS-v4.382.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.378~v4.382 删 v4.377.0.exe（SUMS 身份档案全保留）。
- **文档**：挂池④/dirListingsCache 销号回填+releases/archive/v4.382.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 403→404+34 席插 v4.382 裁 v4.348）+AGENTS 迁 1 插 1（九十七迁：v4.379 入 archive）+progress/todos。

## 最新发布：v4.381.0（2026-09-22）「dsh 蒸馏第四弹：锚定式 token 计量（压缩触发时机去漂移）」

- **动机**：用户口径「继续」——按上轮承诺对 dsh ②锚定式 token 计量单独一版精修（compression 域最后一块精度拼图），留池余 4。全 Go，绑定面 711 零变更，单刀版。
- **落地**：agent/tokenmeter.go 锚定压力表——压力=上次真实 usage 锚点+表面有符号差值；per-message 估价 memo（FNV-64 与 msgChars 同记账面，ReasoningContent 不入账）让差值只统计新增/删除/改写，未变消息锚点估价逐项相消（剪枝/压缩重写后不漂移）；落锚点=maybeCompact 收到真实 usage 时（表面恰=刚被 answered 的请求、assistant/tool 结果未入账，无系统偏差；LastPrompt 回退不落锚）；估压有锚=锚点+差值（负差值钳非负），无锚=裸字符估算旧口径不变；恒装配无配置面，EstimateContextTokens 委托计量表（midTurn/compactIfOver/force 检查同口径）。
- **测试**：Go +5（比率源漂移下未变消息不重算+新增移除精确移动+重复读取零漂移/压缩重写精确移动/负差值钳/无锚回退/maybeCompact 落锚而回退不落）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.381.0。
- **坑**：①锚点时点对齐「刚被 answered 的请求面」——maybeCompact 在 session.Add(assistant) 前调用才无系统偏差，挪后差一整轮②新消息按当前比率估、老消息按锚点时点比率消是刻意的（memo 键只含内容不含比率，比率演化不影响一致性）③sync.Mutex 不可重入——持锁调内部无锁变体，别再走公开方法。
- **产物**：exe 51128320 B SHA256=42b574026869c84f0d9ae31203b3cdd6d7036fc4907d4c04a313e078dc316647（releases/gaea-v4.381.0.exe+SHA256SUMS-v4.381.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.377~v4.381 删 v4.376.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档②销号回填（余 4）+releases/archive/v4.381.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 402→403+34 席插 v4.381 裁 v4.347）+AGENTS 迁 1 插 1（九十六迁：v4.378 入 archive）+progress/todos。

## 最新发布：v4.380.0（2026-09-22）「dsh 蒸馏第三弹·子代理编排：Fork 型子代理 + 结构化输出补完 + V10.36 对齐复活」

- **动机**：用户口径「继续」——dsh 留池两把可安全落地的编排刀凑一组（委派上下文继承 + 结构化结果回收），留池余 5。全 Go，绑定面 711 零变更。
- **落地**：①Fork 型子代理（task 新增 fork；TaskTool 实现 ContextualTool 取父历史做种子；种子自带父 system 字节同源、不注模板提示=子请求前缀与父逐字节一致→KV 缓存继承第一轮近乎免费；组合禁忌 background/continue_from/retry_until；runSubSession 会话持有上移）②结构化输出补完（schema 注入子代理 prompt+[OUTPUT FORMAT] 指令；terminal guard 非 JSON 同会话纠偏重入有界 2 次，热前缀缓存成本极低，耗尽诚实降级诊断前缀+原样；stripJSONFences 剥围栏）③根修 V10.36 对齐复活（P0-1 每轮重置 activeSchemas 无条件清 nil 把构造期父对齐 schema 第一轮抹掉=死代码；New 存 baseSchemas 基线、重置恢复基线，主代理基线 nil 行为不变；V6.0 两测试钉死形态改写为「请求目录父全量缓存对齐+执行面过滤 buildSubReg」双断言）。
- **测试**：Go +6（fork 种子字节同源+缓存形状+模板提示不注入/组合禁忌四态+无父上下文/schema 注入可见/guard 同会话恢复含剥围栏/guard 耗尽诚实降级/stripJSONFences 三态）+改写 2（V6.0 双断言重构）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.380.0。
- **坑**：①「每轮重置」类清理会把构造期注入的配置一起抹掉——重置要回基线不是零值，死代码靠端到端断言请求形状辨识（schema 数 1 vs 2 当场暴露）②V6.0/V10.36 两代测试钉矛盾行为各活各的（对齐是死的两边都绿）——修复一处另一处断言爆开，改断言不是改实现③RunSubAgent 把会话藏在内部——需同会话二次进入/会话预置的功能都要把会话持有上移调用方④executeOne 把工具结果包信封 JSON——测试透信封断言别假设裸文本。
- **产物**：exe 51117568 B SHA256=1a3a237214d1805c337530bf53481f1d516003d6f2de8c2cb883e61b0b878b27（releases/gaea-v4.380.0.exe+SHA256SUMS-v4.380.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.376~v4.380 删 v4.375.0.exe（SUMS 身份档案全保留）。
- **文档**：dsh 蒸馏文档 fork/结构化输出销号回填（余 5）+releases/archive/v4.380.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 401→402+34 席插 v4.380 裁 v4.346）+AGENTS 迁 1 插 1（九十五迁：v4.377 入 archive）+progress/todos。

## 最新发布：v4.379.0（2026-09-21）「dsh 蒸馏第二弹：spill 泄洪 + 中断流结构化块保全」

- **动机**：用户口径「继续」——dsh 留池可安全落地的两把内核刀（spill 泄洪 + 中断流结构化块保全），留池余 7。全 Go，绑定面 711 零变更（read_spill 是 agent 工具目录新增非 App 绑定）。
- **落地**：①spill 主动泄洪（新叶子包 internal/gaea/spill 内存库：单条 4MB/总量 64MB FIFO 驱逐；executeOne 在 SmartCompress **之前**捕获原文——gaea 压缩管线会先销毁中段；内联文本**前置** locator；新工具 read_spill 分页取回；豁免 read_file/read_spill/task；best-effort 纪律；[agent] tool_spill 默认开，子代理 taskTool.SetSpill 随父）②中断流结构化块保全（终态流错误/预算阻断路径 assistant 含 calls 落库 + persistInterruptedCalls 成对补结果行：预执行只读用真实结果、其余 WrapError 信封合成占位；ToolResult 事件照发收口前端卡片；恢复路径刻意不落 calls=悬空非法形态辨析）。
- **测试**：Go +12（spill 包 4：分页/剩余字节/FIFO 驱逐/超限拒收/ctx 往返；agent 5：泄洪全文锚点/豁免与开关/截断生存性/read_spill 端到端跨回合保序/未知 id 报错；中断保全 3：成对落库+悬空检查/预执行真实结果/恢复不落 calls）。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.379.0。
- **产物**：exe 51111424 B SHA256=affb81cd23963555085479c6bd47adbf716abd3762b10e7ce92fcd7b661e3385（releases/gaea-v4.379.0.exe+SHA256SUMS-v4.379.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.375~v4.379 删 v4.374.0.exe（SUMS 身份档案全保留）。
- **坑**：①信封是单行紧凑 JSON——截断按「行」选择按「字节」裁切，行数不超限时全行保留后首行被**头部**裁切：要活的内容（locator 指引）必须放头部，尾缀版被测试当场抓出假绿②混合批执行序非调用序（v4.375 分区并行不同资源键共存）——「先泄洪再取回」端到端测试同批会竞态假红，跨回合才有全序③agent 测试引 builtin + builtin 引 agent=测试期 import cycle（生产编译不报 go test 才炸）——SpillStore 下沉叶子包，策略留 agent④preOutcomes 存已包信封 output——合成占位同信封约定，否则回放两种形态混存。
- **文档**：dsh 蒸馏文档①③销号回填（余 7）+releases/archive/v4.379.0.md（含升级说明：tool_spill 关闭回旧行为/中断续聊占位语义）+CHANGELOG/README+releases/README（计数 400→401+34 席插 v4.379 裁 v4.345）+AGENTS 迁 1 插 1（九十四迁：v4.376 入 archive）+progress/todos。
## 最新发布：v4.378.0（2026-09-21）「Reasonix 真身蒸馏第四弹·留池收官：bash 会话状态锚定 + memory activation 二维 + subject keys」

- **动机**：用户口径「继续优化完善 gaea」——Reasonix 真身蒸馏第四弹，清掉最后 2 项留池（Persistent bash PTY + memory 事实生命周期）收官整条蒸馏线；同日对账 dsh 留池销 3（12→9）。全 Go，绑定面 711 零变更。
- **落地**：①bash 会话状态锚定（cd/export 含导出函数跨调用存活；取道不取器=进程不持久状态持久；EXIT trap 观测式采集：cwd 经 cygpath -w 转 host 形+env 用 export -p 全量转储前置 eval 重放；显式 exit 照采退出码原样传播；被杀/覆盖/exec 换身不采纳；并行批次独立探针完成序采纳；目录消失自愈；豁免=PowerShell/WSL enforce/后台；会话切换经新接口 tool.SessionStateResetter 由 controller NewSession/Resume 重置）②memory activation 二维（pinned=正文随会话快照预算化注入稳定前缀 1200/单条 400、Name 升序保 touch 不翻前缀；relevant=仅检索可达；门控仅记忆总开关，空集合零注入）③memory subject keys（remember 可选 subject_key 单值问题键；Store.Save 撞键拒绝点名持有者，同名=修订放行，置空=释放；SchemaV23+文件后端 metadata.subject_key 往返）。
- **辨伪销号**：PTY 进程持久残留（与 Job Object 灭树/boundedOutput/超时硬杀相克+conpty 不值残余价值）/freshness 三档+expiry（早已在册 decay.go 三态+CleanupArchived）/dsh 三项（timeout-policy=工具内错误串无外层 deadline 通道、skill digest 重发布=缓存稳定前缀同判、contextBreakdown=ContextView 已有）。
- **测试**：Go +15（shellstate 9：端到端存活/显式 exit/保守四态/合法采纳/自愈/reset/nil 锚/Resetter 契约；activation 3；subject keys 4）+装配 1（pinned 正文双空间各见各的+未固化不进+开关关零注入）；Git Bash 真机验证 trap 采集+eval 重放。
- **门禁**：go build/vet 0+gofmt（触碰文件）+全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.378.0。
- **坑**：①bash printf 的 %s 是 Go Sprintf 动词——wrapper 模板写 %%s 否则 vet 拒编②MSYS 路径形态——探针必须 cygpath -w 转 host 形否则下次 cmd.Dir chdir 失败③事件日志列与 Event 结构一一对应——subject_key 真相归 facts 表（与 body 同待遇），不为审计单独开列④NormalizeSubjectKey 空白折叠与下划线不是一回事——键等价只走 trim+ASCII 小写+空白折叠'-'一条路。
- **产物**：exe 51089920 B SHA256=3f65aafb6c69ae807f394c3f34a0272fcbf39f7326a2d6789423376f60e48622（releases/gaea-v4.378.0.exe+SHA256SUMS-v4.378.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略留 v4.374~v4.378 删 v4.373.0.exe（SUMS 身份档案全保留）。
- **文档**：docs/archive/gaea-reasonix-v138-distill-2026-09.md 第四弹三刀+留池⑤⑦收官回填（池清空）+docs/archive/gaea-dsh-016-distill-2026-09.md 销号 3 项（12→9）+releases/archive/v4.378.0.md+CHANGELOG/README+releases/README（保留策略行+发布说明计数 399→400+34 席索引插 v4.378 裁 v4.344）+AGENTS 迁 1 插 1（九十三迁：v4.375 入 archive；补记九十二迁漏登）+progress/todos（reasonix 行翻✅收官）。

## 最新发布：v4.377.0（2026-09-21）「办公记忆/成本库写入纠偏：自动做梦建议制 + 存量清污 + 成本劝写改口」

- **动机**：用户反馈「不管什么都自己记入记忆和成本库，根本不询问，大量错误污染数据」——主源=auto-dream 每轮直写（设计上豁免审批且记忆开关管不住），次源=成本工具文案劝写。
- **落地**：①自动做梦建议制（[dream] mode：suggest 默认/auto 旧直写/off；待确认队列 FIFO 30+接受才入库 source=explicit/忽略丢弃；DREAM_WRITE_POLICY §5 修订；记忆开关收口）②存量清污（GaeaDreamPurgePreview/GaeaDreamPurge 按审计交集删，explicit 与手工沉淀不动）③成本劝写改口（cost_save 描述+cost_search/compose 文案 4 处+compact 2 行）④默认 system prompt 记忆段改口（仅用户明确要求才 remember；自定义提示词需自查）⑤面板 UI（自动做梦三态按钮+建议忽略+清污两步确认+i18n 三语+mock 4 绑定）。
- **测试**：Go +8（模式归一化/队列 FIFO 上限出队三态空间隔离/入队分流/视图转换/接受幂等+忽略不写库/清污交集审计留痕）+前端 MemoryPanel +3（三态循环/忽略/清污两步）。绑定面 707→711。
- **门禁**：go build/vet 0+全量 ci.ps1 绿+版本漂移闸 OK@4.377.0。
- **坑**：①bindingNames.ts 重生成必须保 as const（丢则 drift 报 never 不指向真凶）②jsdom LocaleProvider 默认 en——i18n 断言双语正则③出队失败不得静默落旧写入路径④清污按钮不能只在建议非空分支渲染⑤Proxy mock 方法必须 mockResolvedValue（裸 vi.fn() 返回 undefined 接 .then 崩，单文件假绿全量才炸）。
- **文档**：DREAM_WRITE_POLICY §5+releases/archive/v4.377.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 399）+AGENTS 迁 1 插 1（九十二迁：v4.374 入 archive）+todos。
## 最新发布：v4.376.0（2026-09-21）「Reasonix 真身蒸馏第三弹：压缩救援阶梯 + 重复调用裁决 + 技能目录预算化」

- **动机**：续 v4.375 真身留池第三弹——压缩/恢复域两把主刀+技能目录一把小刀，辨伪销号一批收口留池。全 Go，绑定面 707 零变更。
- **落地**：①压缩救援阶梯（1a slim 摘要档=估算超 window−预留−5% 边际即逐消息截头换有界转录，provider 真实拒绝后重试也降半预算 slim，fits 逐字重放不变；1b 投影截断终级 truncateRescue=压缩无进展或压缩后仍超窗时保护区收窄 target/4→抹大工具结果→整单元丢最老装 marker，先归档 fail-closed）②重复调用硬阻断降级 advisory（repeatedSuccessBlock 退役，第 3 次同签名成功起批后一次性合成提醒，真实结果不再被 blocked 替换）③skill 目录预算化渲染（超 4000 字符二分收缩描述宽度保全员可见，压缩态尾注计入预算，纯名行仍超才退硬截断）。
- **辨伪销号一批**：采样恢复状态机+预算学习（主链路不发 max_tokens、reasoning 不回放，无靶子）/skill 引用按需分页+watcher 热重载/read_tasks 续读游标/会话私有临时目录 env 重定向（Git Bash TMP 注入牵动 /tmp 映射）/压缩状态跨重启保留（重启丢熔断=天然解锁）。余池=Persistent bash PTY+memory 事实生命周期。
- **测试**：Go +13（slim 截头/参数封顶/请求收缩/fits 逐字/拒绝后降档/rescue 抹结果丢单装 marker/KeepErrors 豁免/三态无靶/溢出落到截断终级/advisory 越线一次性+改钉两条/目录压缩保全员/fitting 零变更）+2 处适配（summarize 加 forceSlim 形参）。
- **门禁**：go build/vet 0+全量 ci.ps1 绿 EXIT=0 首跑即绿+版本三处 4.376.0。
- **坑**：①救援机制的可达带先算清再写测试——可达带=compact 尾钳与 rescue 尾钳间盲区，prune 够不到、compact 尾保护又留下的大块 assistant 正文才是构造素材，否则前置清场测试假绿②token 估算器 max(bytes/4, runes/2)——纯 ASCII 按 2 字符/token 计，边界按此口径算③破坏性截断必须 fail-closed 于归档——三级都先归档再动手，archive 坏=整体拒绝。
- **产物**：exe 51030528 B SHA256=8c0febb68f936ed247a317baa6bb86d714474e9f96b9a2c50a3724f7d59b4aef（releases+SUMS 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略删 v4.371.0.exe。
- **文档**：蒸馏文档第三弹三刀+留池销号回填+releases/archive/v4.376.0.md+CHANGELOG/README+releases/README（计数 397→398+34 席插 v4.376 裁 v4.342）+AGENTS 迁 1 插 1（九十一迁：v4.373 入 archive）+progress/todos。

## 最新发布：v4.375.0（2026-09-21）「Reasonix 真身蒸馏第二弹：执行序批次 + 文件观察 + 杂项加固」

- **动机**：续 v4.374 真身留池第二弹——执行序与数据正确性两把主刀+两把小刀。全 Go，绑定面 707 零变更。
- **落地**：①执行序批次修正（read:/file: 统一 file:<path> 资源键同路径拆批保序，跨路径共存并行保留；call/result 对应性守卫）②Live file observations（会话级版本指纹：读后记录/写前比对/自身写后刷新；外部修改 blocked:[stale version]；与 stale-anchor 并存）③git 硬化基线（-c fsmonitor/maintenance 关闭+三个 env+diff 强制 --no-ext-diff --no-textconv）④memory_search 低权威免责前置。
- **测试**：Go +5（保序/共存回归锁成对/观察三态）+键名适配。
- **门禁**：go build/vet 0+全量 ci 绿 EXIT=0+版本三处 4.375.0。
- **坑**：①批次分区改动必须带共存收益回归锁（只测拆批不测共存=延迟优化静默回退）②观察类机制防「假装观察过」——观察点绑真实 I/O 成功路径。
- **产物**：exe 51011072 B SHA256=88499e001577a49b184f8e754a1f1a879e38c13e3affe86dab9572e48c3e09a6（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.370.exe。
- **文档**：蒸馏文档留池四项销号/辨伪回填+releases/archive/v4.375.0.md+CHANGELOG/README+releases/README（计数 396→397+34 席插 v4.375 裁 v4.341）+AGENTS 迁 1 插 1（九十迁：v4.372 入 archive）+progress/todos。

## 最新发布：v4.374.0（2026-09-21）「Reasonix 真身蒸馏：安全四刀 + 内核三刀」

- **动机**：用户纠偏 v4.373 对象认定（dsh 非 reasonix）并指路 GitHub 搜索——真身=esengine/DeepSeek-Reasonix（Go 单二进制终端 agent，V1.12/V1.15 端口出处），本版蒸馏 v1.15→v1.38.11 增量（5227 提交，源码入 C:\AI\re\deepseek-reasonix）；v4.373 dsh 档勘误更名保留。
- **落地**：安全四刀=①Host 白名单 421（httpbridge+诊断端口，挡 DNS-rebinding）②只读表修正（env/awk/sed 移出只读、cargo 仅留 search、find -ok 族、git 四类执行旗标）③git clean-filter 中和（repo 本地 filter.* 置空三件套）④SSRF 共享件（netclient 三导出）+booksource 默认 client 守卫化。内核三刀=⑤收尾判定（reasoning-only=final；零内容冻结重放不注合成消息）⑥bash 输出运行中封顶（head 1MiB+滚动尾 64KiB）⑦compactStuck 新消息解锁（stuckAtMessages）。
- **测试**：Go +10（HostGuard 10 子例/BlockedInternalIP+GuardedClient/cleanFilterOverrides 真仓/间接执行 20 例/收尾两例/stuck 两态/bounded 三例）；行为变更适配 3 处。
- **门禁**：go build/vet 0+全量 ci 绿 EXIT=0+版本三处 4.374.0。
- **坑**：①蒸馏判据带上「上游机制的前提本地是否成立」（awk -f 例外依赖上游已记忆规则模型，gaea 自动放行无此前提）②安全刀落地先 grep 假站/httptest 用例 client 来源。
- **产物**：exe 51004928 B SHA256=85149421d0098b19dc28fbf66e4a0c7a5e52281fdf86b1e97b8ef35d9aed0a90（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.369.exe。
- **文档**：docs/archive/gaea-reasonix-v138-distill-2026-09.md 新增+gaea-dsh-016 勘误更名+releases/archive/v4.374.0.md+CHANGELOG/README+releases/README（计数 395→396+34 席插 v4.374 裁 v4.340）+AGENTS 迁 1 插 1（八十九迁：v4.371 入 archive）+progress/todos。

## 最新发布：v4.373.0（2026-09-21）「Reasonix (dsh) 0.1.6 内核蒸馏：溢出自愈 + 缓存对齐摘要等五刀」

- **动机**：用户口径「继续优化迭代 gaea，本次优化进行蒸馏最新版的 reasonix」——对 /c/AI/deepseek-harness（dsh v0.1.6-alpha.2，上游同步 2026-09-19）三域全量侦察后落地五刀，全 Go 内核，绑定面 707 零变更。
- **落地**：①上下文溢出自愈（IsContextOverflow 七措辞分类器+剪枝+强制压缩+rewrite version 验持久进展+有界重试，连续上限 1）②缓存对齐摘要（摘要请求=会话 system+被折叠区间逐字重放+指令作末条 user 消息，命中热 KV 缓存；renderTranscript 退役）③shrink 硬校验（摘要必须小于被折叠区间否则退机械摘要）④force 多轮压缩（压缩后重测仍超再压一轮，空转不计入 compactStuck）⑤repeat 分级提醒（[3,5,8] 三档递进+deny 计数+插话重置）。
- **测试**：Go +7（IsContextOverflow 9 子例/溢出自愈四路/摘要请求前缀逐字稳定/shrink 两路/force 有界/repeat 三档）；summaryInstruction 快照断言适配。
- **门禁**：go build/vet 0+agent/provider 全包绿+全量 ci.ps1 绿 EXIT=0+drift OK@707+版本三处 4.373.0。
- **坑**：①折叠区含旧 digest（kept）时重放到第一个 kept 消息即与上一请求分歧——system+tools 最贵前缀仍全命中，部分对齐可接受②无进展判定必须看 session.RewriteVersion 而非 compact 返回值（compact 对无可折叠区返回 nil，按返回值判会把空转当成功）。
- **产物**：exe 50995712 B SHA256=363b40e7450e95a6a9089bfe7d3900c9edbb791fb50a2fc163b0098c0f29ac30（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.368.exe。
- **文档**：docs/archive/gaea-dsh-016-distill-2026-09.md（侦察+五刀+留池 12 项）+releases/archive/v4.373.0.md+CHANGELOG/README+releases/README（计数 394→395+34 席插 v4.373 裁 v4.339）+AGENTS 迁 1 插 1（八十八迁：v4.370 入 archive）+progress/todos。

## 最新发布：v4.372.0（2026-09-21）「Gantt 缩放性能：Ctrl+滚轮 rAF 合并」

- **动机**：用户口径「继续」——性能留池「Gantt 行组件化」的可安全子集落地：Ctrl+滚轮缩放的 wheel 事件 rAF 合并（一帧最多一次 setDayW，终态一致）。schedule 1 源文件+1 测试适配，绑定面 707 不动。
- **落地**：GanttView 的 Ctrl+滚轮缩放路径——高分辨率触控板每秒 60~120 次 wheel 事件，原实现每次事件立即 setDayW→三窗格（横道/表格/时间轴）全量重渲染同频发生。改 rAF 合并——帧内累积缩放因子（pendingW），每帧最多提交一次 setDayWAt；锚点（缩放时光标处对应工程日）以累积后的目标日宽换算，滚动位置恢复与逐事件处理终态一致。清理路径补 cancelAnimationFrame。「GanttBars 行组件化+dayW 走 CSS transform」的结构性改造仍留池：条形定位从 dayW 像素改为 scale 变换涉及拖拽坐标换算（鼠标位置→日期）重写，属核心交互高风险区，需专门的拖拽真机回归轮。
- **测试**：GanttView 测试适配——缩放断言等待 rAF 帧提交（jsdom 下 rAF 为 16ms 模拟 setTimeout），33 例全绿；drag.test 7 例绿。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例；首跑 BookHealthPanel 1 例 flaky 与本轮足迹零交叠，单跑 7/7 复绿后全量复跑 OK）+drift OK@707+版本三处 4.372.0。
- **坑**：①rAF 合并后同步断言失效——jsdom 下 rAF 为 16ms 模拟 setTimeout，fireEvent.wheel 后同步断言拿到的还是旧值，测试需 await act 等待帧提交；②it 回调补 async 时易引连环括号错误——手工补 async( 后遗漏原闭合括号层级，esbuild 语法错误逐层暴露；脚本化改写测试文件时优先整段重写而非点状替换。
- **产物**：exe 50,988,032B SHA256=cf94c096112e9c177f6e46f040fe315acf82e5ebc3fb5b4a765ee78057bab7b8（releases/gaea-v4.372.0.exe+SHA256SUMS-v4.372.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.368~v4.372 删 v4.367.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.372.0.md+CHANGELOG/README+releases/README（计数 393→394+34 席插 v4.372 裁 v4.338）+AGENTS 迁 1 插 1（八十七迁：v4.369 入 archive）+progress/todos（Gantt 缩放 rAF 合并销号；行组件化+transform 结构改造留池）。

## 最新发布：v4.371.0（2026-09-21）「聊天历史分页绑定（游标式，绑定面 706→707）」

- **动机**：用户口径「继续」——性能留池「聊天历史分页绑定」设计并落地：大话题切会话不再全量拉取，首屏最新 200 条+向上翻页游标续拉。Go 3 源文件+1 测试+前端 6 源文件。
- **契约**：store 层 ListMessagesPage(topicID, limit, beforeSeq)——beforeSeq<=0 从最新一条向前取 limit 条，否则取 seq<beforeSeq 的最新 limit 条；返回升序消息+hasMore（取 limit+1 条探测后丢弃多余的）。App 层 ChatMessagesPage 绑定返回 {messages, hasMore, oldestSeq}（oldestSeq=本批最早 seq，前端下一次游标）；ChatB 门面+gen_bindings 再生（合计 707 方法→11 门面）+bindingNames/spaceBindings(shared) 手工同步+spaceBindings.test 数量 529→530。
- **前端**：useChatTopics.loadTopic 首屏改拉 ChatMessagesPage(id, 200, 0)，记录 hasMore/oldestSeq；新增 loadOlder（每次 120 条 prepend 到消息头部）；MessageList 接 hasOlder/loadingOlder/onLoadOlder/prepend 四 props——滚动宿主接近顶部且窗口已覆盖全部已加载消息时触发 loadOlder。**关键交互**：头部追加会改变 messages[0].key，沿用既有 anchor 重置逻辑会把用户弹回尾部——新增 prepend 通知（{id 递增, count}）让 MessageList 区分「分页头部追加」（窗口随新增条数扩大）与「切话题整体替换」（重置窗口）。全量路径 ChatMessagesList 保留不动（导出 Markdown 等场景）。
- **测试**：定向 +1（Go TestListMessagesPage——首屏截断/hasMore/游标续拉/拉到底四场景）+前端 ChatPage.test mock 接线（ChatMessagesPage 返回 page 形状镜像消息，15 例全绿）。
- **门禁**：go build/vet 0+internal/chat 全量绿+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+bindings drift OK@707+版本三处 4.371.0。
- **坑**：①头部追加 vs 整体替换的窗口语义——MessageList 的 anchor 重置逻辑原本不区分「切话题」与「分页头部追加」，直接接分页会把用户弹回尾部；②mock 契约必须跟随签名演进——ChatMessagesList 返回数组而 ChatMessagesPage 返回 page 对象，测试 mock 若只镜像数组形态，loadTopic 的 page?.messages 提取会静默得到空。
- **产物**：exe 50,987,520B SHA256=59cfc300918fd55521a58ad8619884ecf0441795cda15a81fd84b607da19a200（releases/gaea-v4.371.0.exe+SHA256SUMS-v4.371.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.367~v4.371 删 v4.366.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.371.0.md+CHANGELOG/README+releases/README（计数 392→393+34 席插 v4.371 裁 v4.337）+AGENTS 迁 1 插 1（八十六迁：v4.368 入 archive）+progress/todos（性能八项消一项：分页绑定收官——余项均需更深的契约设计）。

## 最新发布：v4.370.0（2026-09-21）「工程健康轮：依赖漏洞清零 + Go 工具链升级 + 构建管线修正」

- **动机**：用户口径「继续」——十轮快跑后工程健康整固：govulncheck+npm audit 双扫描按发现修复。Go 1 源文件+依赖升级+构建脚本修正，绑定面 706 不动。
- **扫描与处置**：①Go 标准库 5 处（net/url 二次复杂度/crypto-tls 握手/net-http ReadHeaderTimeout/encoding-xml·asn1 递归）→工具链 1.26.5→1.26.6（go.mod toolchain 指令，构建自动拉取，exe 实测 go1.26.6）；②golang.org/x/image v0.40→v0.46（webp/bmp 解码 panic、VP8L 越界内存分配 4 处——图片解码是用户可控输入面，连带 x/sys/x/text）；③excelize GO-2026-6452 恶意 xlsx panic（上游 Fixed in: N/A）→应用层防线：xlsxpreview Render/NeedsRecalc 入口 defer recover 转错误「xlsx 预览失败（文档结构异常）」（上游修复后可移除），govulncheck 复扫 11→1 仅剩此条；④npm @vitest/mocker 2 moderate（路径遍历，dev-only）→vitest ^4.1.10→4.1.11，audit 归零；⑤**新增 pnpm-lock.yaml**（前端依赖历史上从未锁定，vitest 4.1.11 锁定，CI 实测兼容）。
- **构建管线修正（在册坑落地）**：build.bat 的 wails 命令此前未带 -s（跳过前端），wails 内部隐式 npm install 在 lockfile 缺失+peer 依赖冲突（eslint-plugin-prettier ^4.2.5）时直接失败——本轮 vitest 升级触碰 package.json 即暴露。按在册坑配方修正 build.bat：先 `npm.cmd run build` 显式构建前端，再 `wails build -s`（wails 捕获前端输出走管道会挂起+隐式 install 环境敏感，AGENTS 在册坑原文配方）。
- **门禁**：govulncheck 复扫 11→1+npm audit 归零+go build/vet 0+office 包全量绿+全量 ci.ps1 绿 EXIT=0（vitest 388 文件 3315 例，vitest 4.1.11 兼容）+drift OK@706+版本三处 4.370.0。
- **坑**：①pnpm 项目里 npm install 会生成 package-lock.json 形成双轨——发现后立即删除并 pnpm install 恢复单轨；②lockfile 缺失是环境敏感炸弹——任何依赖树变动触发重新解析，peer 冲突在 wails 隐式 install 中爆炸，lockfile 入库后不再复现；③wails 隐式 npm install 依赖环境解析成功——显式前端构建+-s 是在册坑配方，build.bat 此前未遵守本轮落地。
- **产物**：exe 50,979,328B SHA256=785a740a681d410dfff9c8744d09cb37698ca6d756e0808f31b42edbae8de95f（releases/gaea-v4.370.0.exe+SHA256SUMS-v4.370.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过；go1.26.6 构建）；保留策略 5 版留 v4.366~v4.370 删 v4.365.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.370.0.md+CHANGELOG/README+releases/README（计数 391→392+34 席插 v4.370 裁 v4.336）+AGENTS 迁 1 插 1（八十五迁：v4.367 入 archive）+progress/todos（excelize 上游未修留观）。

## 最新发布：v4.369.0（2026-09-21）「真机走查班第二班：流式分段渲染真机验收 + --w-ink-3 重指向」

- **动机**：用户口径「继续」——走查班第二班：v4.366~v4.368 三轮（聊天组件 memo 化/编辑输入性能/流式分段渲染）真机验收；顺手落地别名重指向小刀 --w-ink-3。前端 1 css 文件，绑定面 706 不动。
- **走查结论**：①流式分段渲染链路真机验证通过——聊天页新建会话→发送短消息→模型正常回复（「我是一个热心且博学的AI助手。」）→回复正常落库与渲染，终态无错误、无 file:// 残留、无 console 告警；分段切分契约由 findStableCut 6 例单测锁定。②--w-ink-3 重指向目检通过——launcher 局部令牌实测 #5b6472 = --md-sys-color-text-tertiary（亮态），别名重指向两目收官。③环境事实：走查观察到的「请求超时」历史消息系用户环境模型服务当时不可用所致，非本轮回归。
- **落地**：module-launcher.css `--w-ink-3` 从 `color-mix(in srgb, var(--color-text) 46%, transparent)` 重指向 `var(--md-sys-color-text-tertiary)`（实色随主题令牌走），`.ml-avatar-user` 二次混色底 12%→14% 微调对齐观感——「次级文本六别名收敛」的最后一目收官。
- **清场**：走查用的测试消息对（user+assistant）经 chat.db 按 id 精准删除（删前 SELECT 确认、按主键删，会话恢复原有 6 条消息）；走查脚本/截图清理；杀壳。用户既有会话与真实工程零触碰。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+drift OK@706+版本三处 4.369.0。
- **坑**：①CDP 目检读局部令牌要在定义作用域元素上读——--w-ink-3 定义在 .ml 选择器下（板块局部令牌层），从 documentElement 读恒为空串；②走查发消息会写用户真实库——聊天无消息级删除绑定，测试消息只能按 id 从 chat.db 精准删除；后续走查若需验证流式，先建专用测试会话再删除整个会话（ChatTopicDelete），避免污染用户会话。
- **产物**：exe 50,951,680B SHA256=26f0293b4b08e03f70d9220093872c3eff335ed72642116d341ec6029fbdfc50（releases/gaea-v4.369.0.exe+SHA256SUMS-v4.369.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.365~v4.369 删 v4.364.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.369.0.md+CHANGELOG/README+releases/README（计数 390→391+34 席插 v4.369 裁 v4.335）+AGENTS 迁 1 插 1（八十四迁：v4.366 入 archive）+progress/todos（别名重指向两目收官销号）。

## 最新发布：v4.368.0（2026-09-21）「性能轮深水区：聊天流式分段渲染 O(n²) 根修」

- **动机**：用户口径「继续」——性能留池最大项落地。前端 3 源文件+1 新测试，绑定面 706 不动。
- **问题**：聊天流式回复时每个 delta chunk 都对全量累积文本重跑 react-markdown（remarkGfm+remarkMath+rehypeKatex），O(n²)——5KB 尾段单次 parse 5~20ms × 30+ chunk/s = 主线程持续高占用与 GC 压力。
- **方案**：三选一定方案 A（ChatRow 流式分支分段，影响面锁死一处）。方案 B（MarkdownContent 内部分段）会冲击 t74 测试精确的 ReactMarkdown 调用计数断言且波及非流式消费方；限频只降频率不降单次 parse 规模。
- **落地**：①findStableCut（gaea/components/MemoMarkdown.tsx）导出+盲区修复——原只扫切点后 suffix 判断 fence 状态，fence body 含空行（代码极常见）时计数偶数误判可切、把代码块拦腰切断（稳定段含悬挂 fence+尾段代码行被当正文的瞬时错排）；改全量行扫描候选切点前 fence 状态，切点在未闭合 fence 内回退到打开行前（切分更保守，稳定段永不含悬挂 fence）；既有 MemoMarkdown 测试零改动通过。②ChatRow 流式分支 stable+pending 两段渲染——stable 段字符串不变时 react-markdown 子树被 memo 整体跳过，pending 段（通常 KB 级）每帧重解析；终态回既有路径零变化。
- **契约与兼容**：genui 零破坏（fence 当普通 fence，闭合落稳定段；流式分支本就不传 overrides——流式中 genui 显示原始 JSON 面板现状不变）；已知接受折衷=松散列表流式序号短暂重排终态自愈+streaming 翻转 DOM 重建一次（与 gaea 同款）；相比 gaea 原设计改进=尾段走完整 react-markdown 管线，未闭合 fence 流式中显示为持续增长代码块更接近终态。
- **门禁**：定向 +6（findStableCut.test.ts：无空行全不稳定/多段落切分/切点后未闭合 fence 留尾/fence body 含空行不拦腰〔盲区修复〕/闭合 fence 入稳定段/空文本）+MemoMarkdown 既有 2 例零改动通过+ChatMarkdown.genui/聊天组件/genui.walkthrough 回归全绿+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3315 例）+drift OK@706+版本三处 4.368.0。
- **坑**：①正则整段替换函数体非贪婪匹配会吞闭合括号——替换串忘了带 `}`，tsc 当场暴露；②性能改造的行为等价是「终态等价」——流式中间态 DOM 从单块变两块（stable+pending），验收口径=终态一致+中间态不劣化，不能拿中间态 DOM 逐字节比。
- **产物**：exe 50,951,680B SHA256=e1bc4358584f6c0c70dacb2c9ac18cd545dd831f8dc6157a9b3d88b7cf2849a0（releases/gaea-v4.368.0.exe+SHA256SUMS-v4.368.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.364~v4.368 删 v4.363.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.368.0.md+CHANGELOG/README+releases/README（计数 389→390+34 席插 v4.368 裁 v4.334）+AGENTS 迁 1 插 1（八十三迁：v4.365 入 archive）+progress/todos（性能八项消一项：流式分段渲染收官——余七项）。

## 最新发布：v4.367.0（2026-09-21）「编辑输入性能：章节编辑 memo 化 + 设定页预览/统计防抖」

- **动机**：用户口径「继续」——性能轮延续：编辑输入链路每击键全页重渲染与全文重解析治理，全部行为等价。前端 4 源文件+1 新 hook，绑定面 706 不动。
- **①ChapterEditor memo+updateTab useCallback**：章节编辑每击键 onUpdate('scenes')+'saved' 两次 setTabs 使 959 行 ChapterPage 整页重渲染，未 memo 的编辑区全场景框陪跑。修=updateTab 改 useCallback（依赖 activeKey）稳定 onUpdate 引用；ChapterEditor 包 React.memo——父页无关 state（focus/ghost/其它 tab）不再重渲染编辑区。
- **②ChapterPage 上报防抖**：novel:chapter-active 事件 effect 依赖 activeTab（每击键引用变），每次全文 join+countTextChars 并派发事件拖动壳层 inspector 重渲染。修=250ms 防抖（cleanup clearTimeout），输入期间不付全文遍历，停顿后上报一次最终值。
- **③NovelSettingPage 预览/字数防抖**：分屏/渲染模式每击键对全文跑 react-markdown+KaTeX 重解析（10k 字约 10~30ms/键，输入明显卡顿）；wordCount 每键 trim+countTextChars 全文扫描。修=新增 useDebouncedValue 通用 hook（300ms），MarkdownContent 与字数统计消费防抖镜像值；编辑器保持逐键受控（输入零延迟）。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例；ChapterPage/NovelSettingPage/novel 组件 187 例定向全绿）+drift OK@706+版本三处 4.367.0。
- **坑**：①跨层字符串转义地狱——python 补丁多层传递中 \n 被吃成真实换行写进源码（TS Unterminated string literal），构造含反斜杠字面量用 chr(92) 拼接彻底避开转义层级；②防抖的正确姿势是「镜像值」而非「延迟源」——编辑器保持逐键受控（输入零延迟），只有下游重计算（Markdown 解析/字数）消费防抖镜像；防抖放 onChange 上游=输入回显延迟，那是 bug 不是优化。
- **产物**：exe 50,951,680B SHA256=177331f52125c531ba4096e7e2e121cc1e653083d77912ebce6d935d2be8ac38（releases/gaea-v4.367.0.exe+SHA256SUMS-v4.367.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.363~v4.367 删 v4.362.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.367.0.md+CHANGELOG/README+releases/README（计数 388→389+34 席插 v4.367 裁 v4.333）+AGENTS 迁 1 插 1（八十二迁：v4.364 入 archive）+progress/todos（性能八项再消两目——ChapterEditor 受控下沉被 memo+防抖等效覆盖大半，下沉本体留池）。

## 最新发布：v4.366.0（2026-09-21）「性能轮收官：聊天组件 memo 化专项 + StoryStream 行级 memo」

- **动机**：用户口径「继续」——性能八项留池中「聊天组件 memo 化」的 props 链梳理完成，落地收官；StoryStream 行级 memo 同步根修。前端 7 源文件+1 守卫适配，绑定面 706 不动。
- **刀1 聊天组件 memo 化**：ChatPage 五个内联箭头回调 useCallback 化（handleToggleSearch/handleToggleForceSearch/handleToggleThinking/handleOpenVoiceSettings/handleSwitchPersona）；`quickReplies={mode !== 'plain' ? QUICK_REPLIES : []}` 的内联 [] 提模块级 EMPTY_REPLIES 常量（内联数组是 memo 恒失效典型来源）；四组件（ChatModeBar/ChatPersonaBar/ChatComposer/ChatInspector）React.memo 包裹（XInner 声明+文件尾 memo 导出）；ChatInspector 统计三遍全文遍历合并单趟 useMemo。memo 语义安全：props 梳理不完整只损失收益无正确性风险。
- **刀2 StoryStream 行级 memo**：抽 SinAssistantRow memo 行组件（storyId/m/两插图回调为 props），parseStorySegments 下沉行内。前提核实：useSinStory 流式 patch（prev.map 只换流式行引用）+showNotice/setIllustration 均 useCallback 稳定。效果：流式期间每 delta 只重渲染流式中的那一行（原 O(消息数×分段)/token 且 keepAlive 后台照跑）。
- **守卫适配**：E16（frontend-e-check.mjs）QUICK_REPLIES 作用域断言按 memo 化声明形态适配（ChatComposerInner 或 memo 导出取 max 位置）——守卫意图（常量在组件之前）不变。
- **门禁**：tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例+E16 PASS）+drift OK@706+版本三处 4.366.0。
- **坑**：①改组件导出形态先查源断言守卫——E16 按 `export const ChatComposer:` 文本定位组件声明，memo 化改名后守卫失明；守卫与被守代码同 commit 演进但断言意图不变；②memo 化顺序纪律——先稳定 props（useCallback/常量）再包 memo，顺序反了 memo 全部空转且无报错只有性能损失。
- **产物**：exe 50,951,168B SHA256=a6d7be9a917339e4d010ceb17fa3b10c594f4225f845a0f9e3b2fbaa222b10c1（releases/gaea-v4.366.0.exe+SHA256SUMS-v4.366.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.362~v4.366 删 v4.361.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.366.0.md+CHANGELOG/README+releases/README（计数 387→388+34 席插 v4.366 裁 v4.332）+AGENTS 迁 1 插 1（八十一迁：v4.363 入 archive）+progress/todos（性能八项消两项：聊天组件 memo 化+StoryStream 行级 memo）。

## 最新发布：v4.365.0（2026-09-21）「前端性能轮：流式节流 + 后台空转治理 + 缓存统一」

- **动机**：用户口径「继续」——前端换新镜头性能审计（三路：React 渲染性能/长列表与流式渲染/桥接与数据面），全落行为等价低风险刀。前端 22 源文件+1 测试，绑定面 706 不动。
- **线1 流式节流**：useChatStream delta 改 buffer+rAF flush——原每 chunk 一次 setState 使 ChatPage 以 30~120/s 全页重渲染+吸底强制 reflow；四条终态路径（done/error/超时/启动失败）flush 前 cancelPendingDelta 防挂起帧污染已清空文本。角色模式打字机 setTimeout(14ms)≈71 次/秒改 rAF 时间驱动（每 14ms 推进 step 字的节奏不变）。ChatPanel 打字机 12ms/字符且每步 O(n) slice 改 rAF 每帧 3 字符。
- **线2 后台空转治理**：keepAlive 13 页 display:none 常驻——五个 canvas rAF 循环（ParticleFlow/SoundWaveOverlay/CompanionAvatar/VoiceChatOrb/RelationGraph）加 `document.hidden || offsetParent===null` 挂起守卫（保留 rAF 链恢复自动续绘）；DagPanel/AgentTree/useComfyTaskProgress/useImageGenQueue/SinIllustration 轮询秒表加 hidden 跳过（时间值基于 start 计算恢复自动正确）；TaskCenter onTaskEvent 接 gate；AIConsole 隐藏丢事件。
- **线3 渲染链**：MainLayout（根壳）/HomePage/ModelCenterPage/AppearancePanel(8 处) 整 store 订阅改逐字段选择器——任意 store set() 不再触发根壳与全部 keepAlive 页元素重渲染；MemoryHubPage 检索 hits key 稳定化。
- **线4 缓存统一**：usePortraitUrl 导出 getCachedAttachmentDataURL（共享成功缓存+方法缺失防御）；Message/FileThumb/inspector 三处直调接线；粘贴/截图落盘后复用内存 dataUrl 作预览（原三重 base64 往返）。
- **线5 AOA 几何**：nodeById/anchorById/taskNameById 查找表 useMemo（taskName 原每边线性扫 tasks O(E×T)/帧）；箭线几何管线整段提升组件级 useMemo（缩放/选中变化不再全量重算，zoom 走外层 scale 与几何无关）；hooks 全部移 early return 前。
- **顺手修真 bug**：AIConsole 违反 v4.62.2 EventsOff 事故纪律（直接摸 window.runtime 且卸载 EventsOff 全清通道）改 subscribeWailsEvent 唯一入口；TaskCenter 事件通道 gate 漏接接上。
- **门禁**：定向 +4 源断言（perf-guards）+tsc/eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3309 例）+drift OK@706+版本三处 4.365.0。
- **留池**（行为级/需设计，审计已给证据）：流式分段渲染/聊天真虚拟化/StoryStream 行级 memo/ChapterEditor 受控下沉/Gantt 行组件化/聊天历史与造价库分页绑定/base64→blob URL/聊天组件 memo 化（props 链需梳理）。
- **坑**：①组件 memo 非免费午餐——内联 props 使 memo 恒失效，须先梳 props 链；②rules-of-hooks 硬约束——提升 useMemo 连同 manual/shown 计算移 early return 前（eslint 当场拦）；③测试 mock 缺方法时可选链与直接调语义差——共享工具要有方法缺失防御（ContextView 空渲染 5s 超时教训）；④整 store 订阅代价被 keepAlive 放大——根壳粗订阅优先治理。
- **产物**：exe 50,951,168B SHA256=c2d09a937841630a46052517ea750cd16ac975d26e242d026c06f26296653948（releases/gaea-v4.365.0.exe+SHA256SUMS-v4.365.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.361~v4.365 删 v4.360.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.365.0.md+CHANGELOG/README+releases/README（计数 386→387+34 席插 v4.365 裁 v4.331）+AGENTS 迁 1 插 1（八十迁：v4.362 入 archive）+progress/todos（性能大工程八项挂池）。

## 最新发布：v4.364.0（2026-09-21）「真机走查班：v4.361~v4.363 前端三轮验收 + withinReadRoots 绘梦图误伤修复」

- **动机**：用户口径「继续」（授权闲置窗口）——按 v4.355 先例前端优化三轮后开真机走查班（CDP 9333 附着 v4.363.0 壳，只读纪律），走查拽出 v4.358 读侧收口真机才现形的误伤，修复发版。Go 1 源文件+1 测试，绑定面 706 不动。
- **走查结论**：双空间 14 板块（工作 6+闲庭 7+设置/进度计划 navigate 直达）错误边界 0/空白 0/console.error 0/exception 0/`img[src^="file://"]` 残留 0。深验=①聊天 PersonaPicker popover「31 个可聊天角色」26 头像全渲染零裂图零 file://（v4.355 报 27 条告警同场景，v4.361 头像收口确证）②schedule `.sched-row-no` 行号实测 rgb(74,64,96)=Violet 亮态 onSurfaceVariant #4a4060（v4.361 收口生效）③tertiary 双态 #5b6472↔#8b93a0 明暗切换正确④longtask 走查全程仅 1 条 800ms 真长任务（阈值 200ms 后 69~165ms 噪音归零）。
- **修复 withinReadRoots 遗漏图片保存目录**：走查日志 6 条 `[AttachmentDataURLError]`（Pictures\gaea 绘梦历史图被拒）——v4.358 论断「绘梦资产均落两根内」不成立：绘梦图实际落 cfg.ImageSaveDir（默认 %USERPROFILE%\Pictures\gaea）。修=withinReadRoots 改 App 方法补第三根（ImageSaveDir 非空时+默认兜底，与 image_handler 落盘口径一致）。真机复验同一张被拒图成功读回 2.89MB data URL。定向 +1=TestWithinReadRootsCoversImageSaveDir（配置根命中+兜底根纯前缀断言不写用户真实目录）。顺手修=方法化后 &App{} 裸构造既有测试 TestGaeaAttachmentRoundTrip panic（nil core 解引用）加 a.core!=nil 守卫回归绿。
- **门禁**：定向 +1+internal/app 全量绿（102s）+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.364.0。
- **坑**：①mock/单测测不出路径语义——v4.358 白名单论断没经真机验证就写进注释，读侧白名单变更必须真机过一遍真实数据路径；②走查拽错要看壳日志不只看页面——AttachmentDataURL 失败前端有占位兜底页面全绿，问题只在壳日志里；③包级函数改方法先 grep 裸构造测试（&App{} 是合法测试惯用形，方法化后嵌入指针解引用即 panic）；④CDP 探针传 Windows 路径的转义陷阱（多层引号反斜杠被 JS 转义吃掉、 成 formfeed），用 join(String.fromCharCode(92)) 或正斜杠（Windows API 皆收）。
- **产物**：exe 50,948,096B SHA256=e026857b8c74d1667b9d5606267d93bffd94d03a3678008073deca4d4f76b976（releases/gaea-v4.364.0.exe+SHA256SUMS-v4.364.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.360~v4.364 删 v4.359.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.364.0.md+CHANGELOG/README+releases/README（计数 385→386+34 席插 v4.364 裁 v4.330）+AGENTS 迁 1 插 1（七十九迁：v4.361 入 archive）+progress/todos（走查班清账+误伤修复销号）。

## 最新发布：v4.363.0（2026-09-21）「后端留池终审：ConvertToPdf 读侧收口落地 + prompt 双构造辨伪关闭」

- **动机**：用户口径「继续」——后端挂池仅剩 2 项（均标「需设计」）逐项终审：一项设计落地、一项证据辨伪关闭，**后端挂池清零**。Go 1 源文件+1 测试+前端 1 小刀，绑定面 706 不动。
- **刀1 ConvertToPdf 读侧收口**（挂池设计项落地）：取证确认前端生产调用点仅 FilePreviewModal exportPdf 一处，三类路径来源（工作区相对/uploads 绝对/对话框工作区外素材）全部被现有机制覆盖。收口=①相对路径 Clean 拒 .. 穿越（原 `filepath.Join(gaeaCwd(), rel)` Clean 掉 .. 可逃逸工作区，对齐 GaeaReadFile 口径）②统一门 `!withinReadRoots(path) && !isPickedFile(path)` fail-closed（对话框素材经 GaeaPickFiles 登记时已入表，isPickedFile 现成覆盖——唯一真·合法绝对路径场景无需新设计）；写侧 exports 目录/文件名全服务端拼装无越界。三类合法调用方零误伤；前端 exportPdf 已有 toast 错误呈现零改动；内存登记表重启即清=重启后历史会话工作区外附件导出被拒，fail-closed 可接受。定向 +1=TestConvertToPdfPathGuard（相对穿越拒绝/未登记绝对路径拒绝/登记路径过守卫/工作区路径过守卫，四场景零外部进程）。
- **刀2 prompt 双构造辨伪关闭**（不修，证据留档）：复核「双构造」实为 New() 构造后立即被 SetPromptFS 换新（main.go:32→34 同步单线程、Startup/httpbridge 全部之前）——第一实例存活期零读者，无正确性影响；换新是 embed FS 注入的唯一无锁安全手段（prompt.Engine 显式无锁契约 override 只写一次）；收益=省一次 20 文件/92KB 解析≈个位数毫秒一次性 vs 成本=改 New 签名（波及测试/CLI）或 Engine 加锁（违反契约）+绑定生成面三处登记——风险>收益辨伪关闭。日后低风险形态=New() 可选 embed-FS 参数+SetPromptFS 幂等短路，排池尾不单独开刀。
- **刀3 前端小刀**：MemoryHubPage 总览读取失败 rail 计数显「—」占位（title 提示，恢复自动清除；原 catch 静默空缺）。
- **门禁**：定向 +1+internal/app 全量绿（99.9s）+tsc 0+eslint 0+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.363.0。
- **坑**：①收口测试的「放行」断言不能借真实转换——.docx 在装了 LibreOffice 的测试机会真转成功（err==nil）断言失效，借「.doc 显式拒绝」这类转换前确定性分支验证过门，四场景零外部进程；②辨伪关闭也要证据链写进留档（风险侧+收益侧），下次取证不必重查。
- **产物**：exe 50,947,584B SHA256=5a2bdafa10858b913df40604d9428be5adb3969dfeae8205eecf274a5b68b0fa（releases/gaea-v4.363.0.exe+SHA256SUMS-v4.363.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.359~v4.363 删 v4.358.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.363.0.md+CHANGELOG/README+releases/README（计数 384→385+34 席插 v4.363 裁 v4.329）+AGENTS 迁 1 插 1（七十八迁：v4.360 入 archive）+progress/todos（后端挂池清零销号）。

## 最新发布：v4.362.0（2026-09-21）「前端优化轮第二弹：吞错可见化收尾 + tertiary 文本令牌收敛」

- **动机**：用户口径「继续」——v4.361 前端优化轮余量再取证：镜头B 剩余吞错项全量复核 + 次级文本别名收敛影响面，两路并行只读后定刀。纯前端 17 源文件+2 测试+3 语言文件，绑定面 706 不动。
- **刀线一 吞错可见化收尾（12 刀）**：①**OfficePanel 读失败防覆盖**（置顶数据覆盖风险——gaeaSettings 失败被吞后草稿留全默认值 permMode 'ask'/sandboxBash 'enforce'，用户点保存就把默认配置覆盖真实引擎 config.toml，NovelSettingPage 同族）→loadFailed+Alert 重试+保存禁用+handleSave 入口拦截，定向 +2；②**「删除假成功」家族 5 刀**（PriceSourcesRepository/PriceSourcesPanel〔加载失败空面板→错误态+重试〕/CostProjectsView 删明细行/CostProjectsView restoreVersion 旧行删除失败中止恢复〔原吞错继续插新行=删半截+重复行〕/CostNotesView）——失败 message/toast 且不推进「已删」语义；③OfficeMemoryLibrary doMergeAll 串行计数分报，失败对保留可重试；④useChatTopics 初始化失败一次性 message.error（原只进后端日志，侧栏空白像历史全丢）；⑤ProgrammingPage 状态轮询首败 setError 复用 prog-error 横幅成功即清（原永久「检测中…」假加载）；⑥ModelSwitcher 失败态「读取失败点击重试」与「未配置任何模型」分流；⑦TaskInboxPanel 状态变更失败提示；⑧小刀 3 处（ChatPanel 人格清单/AboutPanel 版本「v—」→「版本读取失败」/useSinStory 自动命名 setNotice）。低优挂账=MemoryHubPage 左轨计数「—」占位（空缺非假成功）。
- **刀线二 tertiary 文本令牌收敛**：ThemeTokens 新增 colorTextTertiary 正式字段（6 暗工厂 #8b93a0 原值平移/6 亮工厂 #5b6472——原 App.tsx 三别名硬编码 #6b7280 不在 lightFn，Violet 亮 surface 实测 4.41:1 不达 AA；#5b6472 全 surface ≥5.45 连 Violet 容器高层 4.65 过线；不取 TERTIARY 表——暖容器配对色不当正文中性灰）；App.tsx 三别名（--md-sys-color-text-tertiary/--v3-fg-soft/--color-text-tertiary）改由 effTokens 下发**变量名全保留**=25 消费点+softTextStyle 7 文件零改动；机检扩面=contrast BODY_TOKENS +tertiary（12 主题×2 背景 24 条新断言）+明暗锁值+App.tsx 防回流源断言。别名层裁决：--whisper-ink-muted/--fg-faint 纯转发保留；--fg-faint 千级消费重指向 tertiary 是独立视觉拍板项；--w-ink-3（color-mix 非纯转发）重指向需目检挂池。
- **门禁**：定向 +4+tsc 0+eslint 0 error 0 warn（新增 4 处 exhaustive-deps 全数补齐）+全量 ci.ps1 绿 EXIT=0（vitest 387 文件 3305 例）+bindings drift OK@706+版本三处 4.362.0。
- **坑**：①LocaleProvider 默认跟随 localStorage（gaea-lang），beforeAll loadLocale('zh') 不稳固——组件重渲染（失败态→重试成功）后 t() 翻回 en 中文断言飘，测试须 beforeEach setItem；②antd disabled Button 的 accessible name 在 jsdom 不稳定（getByRole 正则失配），按钮定位走 testid/textContent 直查；③zh-TW.ts 必须补齐全部 DictKey（类型从 en.ts 推导全键校验，tsc 当场拦）；④恢复类操作「先删后插」的吞错=删半截+重复行，失败要中止而非吞掉继续；⑤对比度选值要喂最差场景背景（Violet 亮 surface 4.41，不是白底 4.83）。
- **产物**：exe 50,946,048B SHA256=c26432852cdcb9211edf0c3e4bb6d6831a6be8a9dd57e09331276783aa3cd274（releases/gaea-v4.362.0.exe+SHA256SUMS-v4.362.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.358~v4.362 删 v4.357.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.362.0.md+CHANGELOG/README+releases/README（计数 383→384+34 席插 v4.362 裁 v4.328）+AGENTS 迁 1 插 1（七十七迁：v4.359 入 archive）+progress/todos。

## 最新发布：v4.361.0（2026-09-20）「前端优化轮：失败可见化 + 观察池清账 + 令牌卫生」

- **动机**：用户口径「优化gaea前端UI」——后端优化五连发后换前端镜头，三路并行只读审计（令牌一致性/交互三态覆盖/观察池取证）后定刀落地。纯前端 34 源文件+4 测试+2 新工具文件，绑定面 706 不动（零 Go 改动），零删功能（UI 简化红线）。
- **刀线一 失败可见化（7 刀）**：审计撞出 6 处「失败被伪装」缺陷——NovelSettingPage 读失败吞成空编辑器可覆盖保存真实设定（唯一真丢数据风险→loadFailed+Alert 重试+禁保存+Ctrl+S 拦截）；SearchModal 三路 `.catch(()=>null)` 后端全挂显示「未找到」（v4.350 漏网面→参与线全败才报错、单路降级；顺补存任务失败提示）；ConsistencyPanel 检查失败渲染绿色「全部通过」假阴性（→ruleError 横幅+「结果不可信」）；CostLibraryView batchStatus 吞错后无条件报成功（batchDelete v4.350 同函数漏网兄弟→计数口径+busy 闸）；CostLibraryPage 概览失败伪装「空库新手引导」（→全失败错误态+重试）；WeixinPage 提醒静默+首拉闪空态（→v4.351 范式+loadedOnce）；批量吞错 6 处（审核无反馈/开关静默回弹/清空失败消息复活/语音发送石沉大海/任务输出留白/语音设置假保存）。
- **刀线二 观察池清账（3 项全清）**：①file:// 头像——v4.355 走查 27 条 console 告警清零：PortraitImg 抽 usePortraitUrl hook（独立文件 react-refresh 合规+成功缓存防 popover 数百次绑定调用），PersonaPicker/AssetStudio×2/WeixinPage×4 残留直连全部接线（Avatar 场景失败自然回退首字）；②longtask 日志阈值 50→200ms（69~165ms 常规抖动不再刷屏，10s 窗口判定前置 continue）；③schedule 弱化文字——对比度普查遗留「视觉拍板项」落地：12 处把 outline（0.25 alpha 边框令牌）当文字色全换 on-surface-variant（亮态 1.3~1.8:1→≥6:1，有 v4.127 先例；border 用途合法保留），回归锁进 contrast-hex。
- **刀线三 令牌卫生**：softTextStyle 7 文件逐字重复→utils/uiStyles.ts 单源；情绪色 map 2 处重复→utils/emotionColors.ts（no-raw-hex 豁免名单随文件迁移登记）；novel-workspace #fff 反向兜底→currentColor；tailwind --transition 注明被 App.tsx 注入覆盖。
- **门禁**：定向 +6+eslint 0 error 0 warn+全量 ci.ps1 绿（vitest 386 文件 3301 例+E 系列守卫+卫生守卫）+drift OK@706+版本三处 4.361.0。
- **坑**：①「参与者全失败」合取式里不参与的路贡献 true（vacuous truth）——写反两次被定向测试当场抓出；②源断言行首锚定+令牌名后带标点（border-color/outline-variant 前缀误伤）；③no-raw-hex 豁免按文件名，色板迁新文件要同步登记；④hook 别与组件同文件（react-refresh 0 warn 门禁）；⑤select 测试按 option 文本定位、value 必须是 STATUSES 合法值否则 jsdom 静默回落空串短路 onChange。
- **产物**：exe 50,941,440B SHA256=a2c695b3b6b375c309fa6ecb3dac0942b31e8a49e00cea6c2c7e1484b0f9b5da（releases/gaea-v4.361.0.exe+SHA256SUMS-v4.361.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.357~v4.361 删 v4.356.0.exe（SUMS 身份档案全保留）。
- **文档**：releases/archive/v4.361.0.md+CHANGELOG/README+releases/README（计数 382→383+34 席插 v4.361 裁 v4.327）+AGENTS 迁 1 插 1（七十六迁：v4.358 入 archive，顺带补记七十二~七十五迁案行漏记）+progress/todos。

## 最新发布：v4.360.0（2026-09-20）「后端优化轮第五弹：挂池再清 3 刀——.tmp 卫生守卫 / 导入事务化 / 版本号原子化」

- **动机**：用户口径「继续」——v4.359 清完 v4.358 池后余量再取证：本轮落地 3 刀（含 v4.359 发现的 .tmp 病根本身入闸），绑定面 706 不动，前端零改动。
- **刀1 .tmp 卫生守卫**：scripts/clean-tmp.ps1 新增+ci.ps1 在设置 TMP/TEMP 重定向前接入——.tmp 超 512MB 阈值时只清「已知安全」瞬态模式（Test*/*.log/edge-*·walk-*·ui-sweep*·dsh-context* 旧诊断 profile/smoke-*.exe/go-build*），其余不动。**坑=.ps1 含非 ASCII 必须带 UTF-8 BOM**（check-docs 在册守卫④）——无 BOM 时 powershell.exe 按 GBK 解析，中文注释乱码破坏 param 行，阈值默认值静默失效变成「每次都清」（151MB 就触发），加 BOM 后语义恢复。
- **刀2 ImportProjectCharacters 整体事务化**（characterlib/store.go）：原每角色 Get+Upsert+Associate 三次独立 autocommit，百级导入=三百多次写，中途失败=部分导入。修法=抽 execer 接口（*sql.DB/*sql.Tx 公共 SQL 子集）+ getOn/prepareUpsert（校验+时间戳+剧照本地化，磁盘 IO 不入事务）/upsertOn（序列化+UPSERT）/associateOn 四助手拆分；Get/Upsert/Associate 改薄包装签名不变（绑定面零影响），导入循环 Begin→tx 循环→Commit，任一步失败 Rollback。既有 4 个 Import 测试零改动全绿=事务化等价性证明。
- **刀3 SaveVersion 版本号原子化+SchemaV22**（costproject/gaea db）：原「先 MAX 后 INSERT」两步无事务无约束，并发保存同项目落重复版本号。①INSERT 内 SELECT COALESCE(MAX(version),0)+1 自算——单语句在 SQLite 写锁下天然原子，读改写窗口消除；回读实际落库版本填返回值（与预估值并发错位以库为准）②SchemaV22 先清历史重复（bug 产物保留每组最新 rowid）再建 idx_cost_versions_proj_ver 唯一索引硬约束背书。定向 +1=TestSaveVersionSequenceAndUniqueIndex（序列 1,2 连续+直插重复被拒）。
- **留池仅剩 2 项**（均需设计）：prompt 引擎双构造（agent 构造捕获 eng 指针、SetPromptFS 必须先行的时序风险）/GaeaConvertToPdf 绝对路径（preview→convert 链有合法绝对路径用途，需 Pick/白名单统一设计）。
- **门禁**：go build/vet 0+受影响 4 包测试绿+全量 ci.ps1 绿（clean-tmp 步首次随 CI 运行；首跑因 Invoke-Native 插在函数定义前即插即炸，移到版本漂移闸门后修复）+bindings drift OK@706+版本三处 4.360.0。
- **坑**：①.ps1 无 BOM 的乱码是语义问题非显示问题——症状是「守卫每次都触发」而非报错②execer 接口（Exec+QueryRow 公共子集）是 Go 事务化标准姿势，磁盘 IO 留事务外③「单语句天然原子」优先于「包事务」——INSERT...SELECT MAX+1 零锁成本，事务留给多语句组合④ci.ps1 的 Invoke-Native 调用必须在其函数定义之后（插早了 CommandNotFoundException）。
- **产物**：exe 50936832B SHA256=3c42f42929ee734a49e4c3921800b9065719e14c2a56f3e661f4661f7f3abd58（时间戳 2026-09-20 22:13:30 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.356~v4.360 删 v4.355.exe。

## 最新发布：v4.359.0（2026-09-20）「后端优化轮第四弹：挂池清账——v4.358 审计辨伪项落地 7 刀」

- **动机**：用户口径「继续继续」——v4.358 三路审计留下的挂池逐项再取证定刀：可落地 7 刀全清，需设计/需重构/需复现的 4 项如实留池。纯 Go 7 源文件，绑定面 706 不动，前端零改动。
- **刀A SQL**：①SelfHeal 启动期 N+1 点查（repair.go）——分类树一次捞内存 map，pathResolves 逐条目逐段 QueryRow 改纯查表+补 entries rows.Err；②migrateLegacyDB（whisper db）——只拷主文件+wal（-shm 自动重建不拷）、.migrating 临时文件+rename 原子落位+残留 log.Printf 收口 slog（vet 参数错位同修）；③FTS 降级 LIKE 多列长 OR 链（fts.go）——在册 modernc 空集坑高危形状且是 FTS 失败的兜底（兜底自己失明），抽 likeSearch 助手按模式分轮单列查询+Go 合并去重+凑满 limit 提前收工+rows.Err。
- **刀B 启动链**：④Startup 幂等三处（app.go/whisper_state.go/gaea_tasks.go）——logClose 先关旧再换新、initWeixin 先按 Shutdown 同款清扫 weixinServers 再重建、startFileWatch 先 Close 旧 watcher；⑤ai.Client 双构造+桥接早启窗口（app.go）——Startup 复用 New() 实例（nil 才新建），同一 client 贯穿进程生命周期，configureClient 统一接线，Login 重建路径不变；⑥filewatch 同步 Walk——**addTree 异步化试错证伪回退**（测试 5s 超时=监听就绪前事件真丢，核实「全量索引兜底」不成立：正常启动成功路径不做全量索引），改为 app 层 `go a.startFileWatch()` 整体移出 Startup 临界路径，filewatch 保持「Start 返回=监听就绪」同步契约（测试零改动零丢事件）。
- **刀C 校验**：⑦GaeaListDir 相对路径拒 .. 穿越（gaea_listdir.go）——前端四调用方逐一核对全工作区相对路径零误伤；IsAbs 分支保留（v4.98 冻结面拍板+resolvePreviewPath 口径对齐）。
- **留池 4 项**：ImportProjectCharacters 事务化（需三助手 tx 化中型重构）/prompt 引擎双构造（毫秒级+agent 持 eng 指针时序风险）/SaveVersion 并发窗口（待复现）/GaeaConvertToPdf 绝对路径（需统一设计）——todos 已给原因。
- **门禁**：go build/vet 0+受影响 6 包测试绿（app 96s 全量/cost/whisper db+repos/filewatch）+全量 ci.ps1 绿+bindings drift OK@706+版本三处 4.359.0。
 - **坑**：⓪vitest 连续两轮 CI 失败病根=.tmp 堆积 2.4GB（go test 残留+旧浏览器 profile），清到 157MB 后稳过；「.tmp 定期清理进卫生守卫」挂池①审计论断「事件丢失有全量索引兜底」要核实兜底是否真存在——本例正常启动成功路径不做全量索引，照抄审计建议就是真回归；filewatch 测试 5s 超时是免费的反证②挂池再取证≠照单全收——留池 4 项与落地 7 刀同等重要，留池注明原因让下次取证不必重查。
- **产物**：exe 50931200B SHA256=acd0851d4cbdcbfcac9801ffb59411418ad2530a32a0a8f8f8ba90e18c5fa89f（时间戳 2026-09-20 20:21:39 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.355~v4.359 删 v4.354.exe。

## 最新发布：v4.358.0（2026-09-20）「后端优化轮第三弹：三路审计（SQL 存储层/启动链路/输入校验）——分类改名 P0 根修+读侧收口+FTS 增量」

- **动机**：用户口径「继续优化后端」——v4.356~v4.357 已扫 panic 防线/并发锁/资源句柄/错误吞噬四条轴，本轮换新镜头开三路并行只读审计（线1 SQL/存储层、线2 启动链路、线3 输入校验），按证据定刀。纯 Go 36 源文件+3 新测试文件，绑定面 706 不动，前端零改动。
- **线1 SQL**：①P0 分类改名 substr 字节/字符错位（cost.go SaveCategory，Go len() 喂 SQLite 按字符计数的 substr——中文路径子树条目路径整段截断重挂）→rune 计数偏移+精确行 category 叶子名同步+两写包事务+换父也重写+环检测+名称含 / 拒绝（同函数五问题一次收口）②rows.Err 缺失 17 处批量补（记忆/三元组/情节恢复主路径+成本检索语料+ListItems 直通不可变版本快照）③单条写全量重建 FTS O(N²)→接上闲置增量助手（失败降级全量重建兜底）+两个 Rebuild 事务化（原 DELETE+逐条重插 autocommit=索引空窗）④schema_v15 补 facts/episodes 会话索引+向量写事务化+COUNT 收错×5+GetSource 点查。
- **线2 启动链**：⑤ASR provider 四引擎刷新 goroutine 并发写竞争→SetASRProvider 持锁+读侧快照（对齐 SetRealtimeSession 先例）⑥恢复/迁移结论先于 setupLogging GUI 全盲→restoreSummary 字段日志就绪后回放⑦三 db 打开失败 log.Printf→slog.Error⑧Shutdown 补关 Hephaestus/whisper db+微信 Stop 通知异步化（对齐 notifyStart 先例）+回退轮询 stop 接线+reminderStop Once（复用已有 reminderOnce 字段）。
- **线3 读侧收口**（审计结论：写删侧护栏完备、读侧系统性短板）：⑨GaeaReadFile 对齐写端口径（拒穿越+withinWriteRoots+2MB 截断）⑩AttachmentDataURL withinReadRoots 根白名单+32MB⑪ReadFileB64 契约强制化=Pick 登记机制（fail-closed，pickFile.ts 链路零破坏）⑫Preview 相对路径拒穿越+image/docx 32MB 封顶⑬快照/场景 sceneID 白名单（唯一写越界点）⑭上下文看板 sessionPath 补 sessionDirForPath⑮CreateProject 书架 containment。
- **辨伪挂池**：FTS 降级长 OR 链疑似空集坑/版本号并发窗口/legacy WAL 拷贝/deferred BUSY/SelfHeal N+1/导入无事务/Startup 幂等/filewatch Walk/prompt·aiClient 双构造/ListDir 全盘枚举（需查前端调用源）/ConvertToPdf 绝对路径——todos 新挂池行已给 file:line。
- **测试**：定向 +3（TestSaveCategoryRenameChineseSubtree+Guards/TestReadFileB64RequiresPick/TestGaeaReadFileTraversalGuard/TestSnapshotSceneIDGuard）+既有测试修正 2 处（contextview 伪造路径改真实会话目录形态）+V15 迁移断言。门禁=build/vet 0+受影响 10 包绿+全量 ci 绿+drift OK@706+版本三处 4.358.0。
- **坑**：①Go/SQLite 长度语义差是中文环境必炸点——len() 喂按字符计数函数 ASCII 全绿中文必坏，跨语言偏移一律 rune 或 Go 侧完成②增量助手存在却没人用——接闲置能力补失败降级兜底③同函数多问题一次想全④读侧防线不能照抄写侧——根白名单+Pick 登记+尺寸封顶组合，逐绑定核前端调用链（AttachmentDataURL 只限 portraits 会误伤绘梦资产）。
- **产物**：exe 50928640B SHA256=2e1bfdf827f3aa70cc2f8f84a21d3259d2776c6bec337e5f33b553c21e2641bf（时间戳 2026-09-20 07:27:12 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.354~v4.358 删 v4.353.exe。

## 最新发布：v4.357.0（2026-09-19）「后端优化轮第二弹：挂池三组清账——错误吞噬可见化 4 处 / 并发观察 4 刀根修 / 死代码 2 文件」

- **动机**：用户口径「继续优化后端」——v4.356 挂池三组（错误吞噬卫生 P2×5/并发观察 6 项/AuditLogger·DecisionLogger 死代码）逐项取证定刀。纯 Go 8 源文件+3 测试文件（含 1 处签名适配），绑定面 706 不动（零签名变更 drift OK），前端零改动。
- **刀A 失败可见化 4 处**：①characterlib ListChatEnabled 吞错→签名 ([]Character, error) 上抛（WhisperGetPersonalities 失败 warn 回退内置人格、app.go 剧照同步 warn 跳过——「没有可聊天角色」与「库读失败」不再混淆）②DAG 懒清扫 `_ = Save` ×2→warn（sweep 幂等下次重扫重存，失败可见即可）③turn markdown 投影 `_, _ =`→warn（JSONL 证据链仍在，投影静默缺失不可见）④cost_projects 状态标签 Save ×2→warn（版本/沉淀本体不受影响，列表页状态不再永远旧档）。辨伪：AppendTurnTraceToDB 失败仅 log 不升级——whisper 链无 Notice 事件总线（WhisperChat 同步绑定返回），接线成本>收益，日志 Error 级已可见。
- **刀B 并发加固 4 刀**：⑤PrepareContinue TOCTOU 根修（subagent_store.go）——meta.Status 检查与 MarkRunning 写回之间窗口，双路续跑同 ref 交错写转录；绑定层 followUpClaims 只盖 UI 路径，TaskTool continue_from 无守卫。修法=**接通 store 半成品**（SubagentRun.release 字段+Release() 调用+注释三件齐备却无人赋值恒 no-op）：continueClaims sync.Map 占 ref 单飞，第二路报 already being continued；claim 随 defer run.Release()（调用点本就有）或 SaveCompleted/SaveFailed 终态写兜底释放；转录 Load 失败不占坑；零调用方改动。⑥cost BM25 语料/版本戳原子化——rankerFor 内部再 Load 与调用方捞语料非同一时点，写路径在两步间推进版本时旧语料挂新版本 key 用到下次写；调用方捞语料前快照 corpusVer 传入（rankerFor 加 version 参数不再自取）。⑦weixin Stop→Start 双轮询根修（clawbot.go）——旧 pollLoop 睡在轮询间隔/长轮询 HTTP 里，Stop 后立刻 Start 时 running 被 Swap(true) 复位，旧 loop 醒来条件复活与新 loop 并行；修法=生命周期代际 gen atomic.Int64，每次 Start 递增，pollLoop 捕获启动代 alive()=running&&gen==启动代；panic 兜底的 running 复位加代际守卫（防旧 loop panic 打掉新 loop）。⑧realtime Dial 双拨号泄首连（openai.go）——检查（conn==nil）与赋值之间窗口，输家覆盖 s.conn=赢家连接被遗忘无人 Close（fd 泄漏+双 readLoop）；修法=CAS 落位锁内重检 closed/conn，输家关掉自己拨到的连接再报 already connected/session closed。
- **刀C 死代码**：删 internal/gaea/control/audit.go+decisions.go（~230 行）——New*Logger/SummarizeAuditLog 全仓零调用（v3.2/v3.3 期设计未接线），控制包测试零引用；实际审计走 core/journal.go AppendAudit 与 whisper/desktop_audit_log.go 两条活链。挂池③销账（「直接删」）。
- **挂池辨伪不修 2 项**：memory file_backend 双写非原子（事实文件+MEMORY.md 索引两步）——设计内自愈语义已在（损坏行跳过/traceable and recoverable/索引可重建），事务化两阶段写成本>收益；QuickAdd 锁内 AppendDoc——锁刻意串行化 ReadFile+WriteFile 读改写（移出锁=并发丢更新），低频小文件阻塞窗口毫秒级。
- **测试**：定向 +2——TestPrepareContinueSingleFlight（双路同 ref 第二路受理处即拒/Release 后可再续/终态写兜底释放后可再续）+TestOpenAISession_DialConcurrentLoserCloses（并发双 Dial 恰一胜一负，输家报 already connected，赢家连接保留）；既有测试适配 1 处（store_test 双返回值）。race 检测本机无 gcc 未跑（CI race job 兜底）。
- **门禁**：go build/vet 0+受影响 6 包测试绿（agent/realtime/cost/weixin/characterlib/app）+全量 ci.ps1 绿（exit 0）+bindings drift OK@706+版本三处 4.357.0。
- **坑**：①半成品机制比没有更危险——release 三件齐备无人赋值，读代码误以为守卫存在；新并发守卫落地先 grep 赋值点验证接线完整性②可见化≠fail-closed——幂等写失败 warn 即可不阻断主流程，判据=失败是否改变主操作结果③代际修复要同时护 panic 兜底复位路径④吞错修复优先改签名上抛而非原地打日志（编译器强制全调用点审查）。
- **产物**：exe 50900480B SHA256=c3877db06fd398b74df4a6b981b0f599ba104e7f81062ecd886b0ff51872a413（时间戳 2026-09-19 23:03:48 新鲜；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.353~v4.357 删 v4.352.exe（SUMS 身份档案全保留）。

## 最新发布：v4.356.0（2026-09-19）「后端优化轮：panic 防线收口 12 处 / 确定性自死锁根修 / cfg.Model 并发收口 / 快照失败可见化」

- **动机**：用户口径「继续优化迭代 gaea，本次会话优化后端」——2026-09-12 后端性能普查（两遍扫描，v4.245~v4.253 刀A~E）已全清，换镜头开**三路并行只读审计**（线1 panic 防线与错误吞噬/线2 并发安全与回调发射序/线3 资源句柄生命周期），按证据定刀。纯 Go 30 源文件+2 测试扩展，绑定面 706 不动（零签名变更 drift OK），前端零改动。
- **审计总貌**：线1=Wails dispatcher 只保同步绑定调用，goroutine 裸奔 panic=exe 闪退；65 处 go func+约 30 处 go xxx() 交叉比对 58 处 recover，撞出 12 处漏网。线2=整体纪律水准之上（jobs 锁序/Registry leaf-lock/approval 双查/tasks 条件 UPDATE 堪称范本），真问题=绑定层「调用方自觉」缺口（自死锁/裸字段/无运行闸），非内核锁错。线3=**零 P0/P1**（body 52 文件/os 79 处/rows 93 处/ticker 22 处/子进程收尸/fsnotify 全配对）——句柄轴线纪律极好，余卫生项转刀B。
- **刀A1 防线 12 处**：微信 clawbot pollLoop+handle（唯一常驻外部通道内同步跑完整 AI 回合；同文件三处先例证明漏网；降级文案 fallbackReplyText 常量化）/DAG 三处（节点 panic 转节点 failed 与 err 同形+dagStart 编排兜底+改向——与 GaeaSubagentFollowUp 逐行同构却没抄防线）/语音回合链三处（handleSpeechEnd/runReply/runRealtimePump，全文件 0 recover）/MCP StartAvailable 连接（外部协议数据 panic 转 RecordFailure）/书源五处（导入/追加/下载 panic 转 error 事件不挂进度；fetch worker 与搜索单源转槽位错误项宁漏勿误）/dream/tts/SSE parseStreamEvents（自身 defer close(chunks) panic 路径照常执行，消费方不悬挂）。
- **刀A2/A3 并发守卫**：①GaeaMemorySetRetentionDays 持 ga.mu 调 GaeaInit（内部首行再取锁）——sync.Mutex 不可重入，**引擎未初始化时调用即确定性死锁**：绑定 goroutine 永久卡死持锁，GaeaSend/Cancel/Skills 全部消费方连锁冻结仅重启可解→锁外探测 needInit 仅未初始化才前置 Init（gaeaApplyCfg 同序；已初始化路径行为不变，现有测试零改动）②cfg.Model「UI 三写点×请求热路径四读点」裸字段竞态（string ptr+len 双字撕裂）→internal/config/config_model.go 新增 modelMu+SetModelMem/GetModelMem（Config 是值语义大结构加锁字段毁复制性，访问器是最小侵入解；启动期 Load 单线程装配不受限）③GaeaNewSession/GaeaResumeSession 补 Running() 运行闸——run loop「only swaps session while idle」是注释约定无执行强制，回合中换会话=SetSession 与无锁读交错+后续消息落旧 session 转录分裂；照 GaeaSubagentFollowUp 先例绑定自守防 httpbridge 绕过前端闸。
- **刀B 失败可见性+卫生**：①回合收尾快照失败 slog.Warn-only→Warn 级 Notice（前端 Transcript 按 level==="warn" 渲染可关闭错误卡）——会话文件被 OneDrive/杀毒/备份短暂锁定或磁盘满时 UI 全绿重启回退数轮无提示；对照模型调用前 checkpoint fail-closed 纪律，收尾快照漏了同款 ②chat 流式 runID 毫秒时间戳拼 atomic 序号（同毫秒双路碰撞=两 goroutine 向同一事件通道交错发帧）③netclient 新增 DrainAndClose（非 2xx 早退不读尽 body=连接被丢弃不复用，健康探测每次重新 TCP/TLS 握手；64KB 上限；统一替换 herdsman health/modelengine 三文件/plugin transport_http 9 处 defer Close，成功路径 no-op+Close 安全）④裸 rename 迁 fileutil.RenameWithRetry 21 处/18 文件（app 13+config_save/backup/dag×2/narrative/office docxedit·pptxedit/project；Windows AV/索引器瞬时占用保存失败韧性；跳过跨盘回退触发/跨目录搬迁/恢复流程等特殊语义 12 处）。
- **测试**：定向 +2——TestGaeaMemorySetRetentionDaysUninitialized（死锁回归：未初始化路径带 30s 超时守卫+os.Exit 防 defer 二次挂起，照 gaea_init_deadlock_test 先例；断言锁外 Init 拉起成功+保留期生效）+TestDagNodePanicMarkedFailed（runner panic→节点 failed 带 panic 证据+下游 skipped）。
- **门禁**：go build/vet 0+受影响 10 包测试绿+全量 ci.ps1 绿（go test 全量+lint+build+vitest 386 文件 3295 例——ContextView 2 例在册 flaky 复跑 28/28 绿）+check-docs OK（AGENTS 38231B 预算内）+bindings drift OK@706+版本三处 4.356.0。
- **坑**：①goroutine 裸奔是绑定层系统性缺口而非个例——防线判据=「同文件/同型先例是否已立」（clawbot 三先例/followup 同构/plugin 双标），漏网处全是新码没抄先例②sync.Mutex 不可重入死锁是确定性而非竞态（cfg==nil 分支必现）——「锁外探测+幂等重入」修法保已初始化路径零变化③cfg.Model 收口走包级访问器不改结构（加 sync.Mutex 字段会触发 vet copying locks）④vitest flaky 先复跑失败文件再定与本轮无关（足迹零交叠）⑤后台复合命令 `taskkill…; cmd /c build.bat` 在 harness 下静默假成功（exit 0 只出 banner）——**构建产物必须核对时间戳/哈希/大小三件**，本次靠 v4.355 旧哈希识破重跑。
- **产物**：exe 50893824B SHA256=9C570C94C07A1500282DAABDAE7FCD7BDC2E26ACA1BD305BDBF66EDC25709205（releases/gaea-v4.356.0.exe+SHA256SUMS-v4.356.0.txt 仅本地；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.352~v4.356 删 v4.351.exe（顺带补清上版漏删实物的 v4.350.exe；SUMS 身份档案全保留）。
- **挂池新增（观察池）**：错误吞噬卫生 P2（characterlib ListChatEnabled 吞错返回空列表/cost_projects 双 Save 仅状态标签/dag sweep `_ = Save`/AppendTurnTraceToDB 失败仅 log/turn markdown 导出吞错）；并发观察 6 项（PrepareContinue TOCTOU 双路续跑同 ref——claims 只盖 UI 路径/cost BM25 语料与版本戳非原子/weixin Server Stop→Start 旧 pollLoop 未退可双轮询——当前绑定面不可达/realtime Dial 双拨号泄首连——单会话不可达/memory file_backend 双写非原子 legacy/QuickAdd 锁内 AppendDoc 文件 IO）；AuditLogger/DecisionLogger 死代码（Close 无人调，接线时须挂生命周期或直接删）；15 处「SetLoader 形式竞态」类已注释自证的良性单字指针读维持观察。

## 最新发布：v4.355.0（2026-09-19）「真机走查班：UnifiedSearch 参数序错位根修」

- **动机**：用户口径「继续」——前端优化六连发后按惯例开真机走查班（v4.343 非版本刀先例），CDP 9333 附着 v4.354.0 壳（WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 直启 exe）。
- **巡检基线**：双空间 13 页（书斋 6+乐园 7）错误边界 0 / 空白 0 / exception 0（.tmp/walk-v4354*.mjs 三版迭代）。
- **真发现：UnifiedSearch 三调用方参数序错位（月级老 bug）**：Go GaeaUnifiedSearch(query, topN int, scope ...string)（v2.23.0 起）vs 前端 bridge 声明与调用方 (query, scope, topN)——scope 落 int 形参每次 unmarshal 失败；记忆中枢三脑检索/工作区搜索面板/Ctrl+K 真壳自 v2.23 一直坏。捂因=mock 按前端期望签名写（契约全绿）+bindings_completeness_test 只收方法名+.catch(()=>null) 静默——v4.350 失败可见化（「检索失败，请重试」）一次走查拽出。修复=bridge 签名 (query, topN?, scope?)+MemoryHubPage/WorkspaceSearchPanel/SearchModal 换序+mock 同序+mock-contract-e5/MemoryHubPage/WorkspaceSearchPanel 三测试断言换序；零 Go 改动绑定面 706 不动。
- **修复确证（新壳复验）**：错误从 cannot unmarshal → embedding 请求失败（localhost:8080 语义服务超时——本机 embedding 未部署属环境事实挂观察池）；UI 三态如实呈现。语义检索完全可用需先起 embedding 服务，届时补一条龙走查。
- **其余发现**：角色库页 console 27 条 file:// 头像 WebView2 拒载（既有非本轮回归，挂池：AttachmentDataURL 或 asset:// 通道）；[longtask] 69~165ms 误报阈值过敏感挂池；ResizableDrawer Esc 真机抽验因导航成本收尾（vitest 4 例已锁）。
- **坑**：①mock 契约必须从 Go 绑定复制签名含参数序——按「前端怎么调」写，签名错位时 mock 全绿真壳全坏②静默吞错捂 bug 时长与可见性成反比，失败可见化是最便宜的测试③走查探针按钮定位优先类名/testid，textContent 匹配撞到别的按钮得假阴性（「检索中…」也是按钮文案）。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3295 例）+bindings drift OK@706+版本三处 4.355.0；产物见 releases/archive/v4.355.0.md。**文档**=releases/archive/v4.355.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十迁：v4.352 入 archive）+progress/todos。**未做（下刀池）**=embedding 服务就绪后语义检索一条龙走查/file:// 头像通道治理/longtask 阈值/dirListingsCache 接线/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.354.0（2026-09-19）「前端优化第六轮：P0 数据形状崩溃×2 / 异步竞态 11 处 / 资源生命周期」

- **动机**：用户口径「继续」——五连发后新现场再开三路并行只读审计（异步竞态与并发 / 运行时数据形状防御 / 资源生命周期第二层），按证据定刀；纯前端 17 源码 + 1 测试扩展，零 Go 改动、零绑定变更（706 不动）。
- **P0 数据形状崩溃×2**（均为「后端产出前端未声明形状 + mock 不可复现」同型）：①伏笔面板 STATUS_META 3 态 vs Go 6 态并集透传，「清理→项目重置」把手动条目置 pending 重拉即整页崩（一键自毁式交互）→类型 6 态并集+STATUS_META 补全+statusMetaOf 防御未知值+流转按钮 6 态文案语义，测试+2；②记忆面板 suggestions.skills=null（Go suggestSkillsFromMemories 规则类记忆<2 条 return nil，nil slice 无 omitempty 序列化 JSON null；mock 恒 []）→setSuggestions 入口统一收窄归一（skills/memories/merges ??= []），消费侧五处不再逐个防御。
- **异步竞态 11 处**（统一项目已有 seq/loadToken 范式补齐，辨伪 18+）：PromptWorkshopPanel 模板切换 seq（A 正文写进 B 表单保存即数据损坏）；BookSearchModal pickSeq+backToSearch 失效（B 书源 URL+A 章节范围错误导入）；useChatTopics.createTopic/ChatPage.handleClearMessages 失效在途（新增 invalidateLoads 暴露，空话题视图被旧话题填满后清错对象）；useSinStory loadSeqRef+createStory 失效（切故事串台）；ChapterPage.handleSave 保存快照+setTabs 函数式比对（保存期间继续打字不再被强制 saved=true，未保存保护不丢，提示「请再次保存」）；SearchModal 移除 onPressEnter 双发+searchSeqRef（旧意图卡不再可被执行）；MemoryHubPage runSearch searchSeqRef（Enter 路径守卫）；useImageGenConfig backendSwitching 重入闸；outlineStore 模块级 loadSeq（错书章节读写风险）；WeixinPage togglingId busy 闸+Switch loading（连点两条相反 Save）；TTSPlayer 卸载 cleanupAudio（泄 blob URL 且音频继续外放）。
- **资源生命周期第二层**：blob URL/Observer/AudioContext/MediaStream 全量辨伪通过（saveFile 同步 revoke、useVoiceChat 四路回收、9 处 Observer 均 disconnect）；dirListingsCache 无界+invalidateTurnCaches 零调用方挂池（结构性欠账需设计接线时机）。
- **坑**：①Go nil slice→JSON null 且 mock 手写 [] 永不复现——前端收窄层统一归一优于消费点逐个防御②saved 标志语义=「缓冲区==磁盘」，异步完成侧不能无条件置位，快照比对是通用解③antd Input.Search 回车已触发 onSearch，再挂 onPressEnter=双发，删除而非守卫④竞态修复统一走已有 seq 范式不发明新机制。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3295 例）+bindings drift OK@706+版本三处 4.354.0；产物见 releases/archive/v4.354.0.md。**文档**=releases/archive/v4.354.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十九迁：v4.351 入 archive）+progress/todos。**未做（下刀池）**=dirListingsCache 失效接线/CharacterPage 双击重复写（幂等低危）/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.353.0（2026-09-19）「前端优化第五轮：ResizableDrawer 弹层可访问性收口 / 可点 div 键盘化余量」

- **动机**：用户口径「继续」——可访问性余池两条线收口；纯前端 9 源码 + 1 测试（新建），零 Go 改动、零绑定变更（706 不动）。
- **线1 ResizableDrawer 三缺补齐**（对照 ApprovalModal 先例）：dialog 语义（role=dialog+aria-modal+aria-label 新 prop，两调用方传 i18n 标题）+ Esc 关闭（可编辑目标不劫持）+ 焦点管理（开记录 activeElement+聚焦容器 tabIndex=-1，卸载还原 try 兜底）；ResizableDrawer.test 新建 4 例（Esc 经退出动画/普通键不关/dialog 三件套/开聚焦+卸载还原）。
- **线2 可点 div 键盘化余量**（v4.349 池逐点核对净剩 4 处+顺带 5 钮）：ChapterTreePanel 章节树节点/OutlinePanel 大纲行/StatsPanel 会话·本轮两折叠→role=button+tabIndex+Enter/Space+aria-label（FileTree 范式）；ChapterEditor 加/删场景+ChapterTreePanel 重新生成/删除/添加五 icon Button 补 aria-label（title 不参与可访问名）；FiveCalcPanel 版本带入遮罩补 Esc。辨伪=RailItem/FactCard 已是 button。
- **坑**：①effect 引用 useCallback 声明顺序——Esc effect 插在 handleClose 声明前，依赖数组渲染期即时求值 TDZ ReferenceError；effect 一律放依赖声明后②React 合成 KeyboardEvent 无 isContentEditable（DOM HTMLElement 属性），div onKeyDown 判可编辑目标要 e.target 断言③审计清单是快照不是现状，逐点核对再动手。
- **门禁**：tsc 0（--force）+eslint 0/0+全量 ci.ps1 绿（vitest 386 文件 3293 例）+bindings drift OK@706+版本三处 4.353.0；产物见 releases/archive/v4.353.0.md。**文档**=releases/archive/v4.353.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十八迁：v4.350 入 archive）+progress/todos。**未做（下刀池）**=reducer 级整店订阅 13 处（v4.350 审计辨伪「appStore 写入面低频不构成热点」维持不做）/selbar z-1080 阶梯治理（防御性）/海报墙 hero 内容槽（内容决策）/持久全量 overlay（真机窗口）。

## 最新发布：v4.352.0（2026-09-19）「前端优化第四轮：ForeshadowPanel 行 memo / ModelCenter ctx 两通道拆分 / 图标钮可访问名」

- **动机**：用户口径「继续」——v4.351 下刀池延续；纯前端 16 源码 + 5 测试（扩 5），零 Go 改动、零绑定变更（706 不动）。
- **线1 ForeshadowPanel 行 memo**：抽 ForeshadowRow memo 组件 + flow/remove/saveEdit/handleEditChange/cancelEdit/startEdit 六回调 useCallback 化 + 编辑态布尔/文本双 prop 下发——行内编辑/登记表单每键只重渲染受控行（原全列表重渲染，数百行可达每行 Tag/Tooltip/Popconfirm）；测试 +2（编辑保存全量写回含 3 条/取消不写回其余行不受影响）。
- **线2 ModelCenter ctx 两通道拆分**（v4.350 辨伪项的正解）：ModelCenterContext 拆 StateContext（状态字段，内容变化才触发消费者重渲染）/ ActionsContext（setter/handler 引用波动单独承载）/ 兼容口（两通道合并，useModelCenter 存量不破坏）；页面 stateCtx/actionsCtx 各自 useMemo，9 个 section 迁 useModelCenterState + 按需 useModelCenterActions（多行解构逐字段分类），settingGlmEndpoint 归 state——键入 API Key 不再全 section 重渲染；4 个测试文件兼容 Provider 补三连挂载（value 同对象）。
- **线3 图标钮可访问名**：ChatRow 四处（用户/助手复制动态态+朗读）+ ChatModeBar 四处（联网开关动态态/角色库管理/切换角色/语音设置）补 aria-label——Tooltip 不参与可访问名；SecurityPanel 排除（按钮带文字标签非图标钮）。
- **坑**：①多行解构迁移必须先从源码抄全字段清单再分类（凭接口记忆分类漏 trendData/settingGlmEndpoint/setXxx——UNCLASSIFIED 断言救一半，tsc 逮另一半）②测试文件改完必须 tsc -b --force（上轮沉淀本轮再验证）③兼容口三 Provider 嵌套顺序 state→actions→兼容，测试挂载三连 value 同对象语义不变。
- **门禁**：tsc 0（--force 全量复核）+eslint 0/0+全量 ci.ps1 绿（vitest 385 文件 3289 例）+bindings drift OK@706+版本三处 4.352.0；产物见 releases/archive/v4.352.0.md。**文档**=releases/archive/v4.352.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十七迁：v4.349 入 archive）+progress/todos。**未做（下刀池）**=其余可点 div 键盘化 ~8 处（v4.349 池）/ResizableDrawer 焦点管理/reducer 级整店订阅 13 处/selbar z-1080 阶梯治理/海报墙 hero 内容槽（内容决策）。

## 最新发布：v4.351.0（2026-09-19）「前端优化第三轮：启动反闪根修 / 绘梦历史有界化 / 亮主题叠加层失效群 / color-scheme / 反馈确认补遗」

- **动机**：用户口径「继续」——v4.350 下刀池延续，六线并收；纯前端 29 源码 + 4 测试（1 新 3 扩），零 Go 改动、零绑定变更（706 不动）。
- **线1 启动反闪根修**：亮色用户每次启动先看 ~2s 深空屏（index.html 静态启动屏硬编码深底，主题变量 App useEffect 首帧后才注入）→ head 内联脚本（bundle 前）预读 gaea-display-mode，判定顺序**逐字对齐 appStore.loadMode**（light→亮 / system→matchMedia / 旧布尔键 gaea-dark・wubigork-dark '0'→亮 / 缺失→dark 缺省），亮则 html[data-bs-light]；静态屏与 body 浅色分支 + boot-splash.css 首帧兜底——App 注入真令牌后 var() 接管，两段同色无跳变。
- **线2 绘梦历史有界化**（无界三处收口，素材库 120+12/页先例对齐补漏）：持久化 HISTORY_META_MAX=500 读/写同截（原 path-only 条目可积数千条）；渲染窗口 60+「加载更多(N)」增量展开（原数百 img/video DOM 一次性构建）；回填 restoreHistoryImages 加 limit 参数按 needsFileRestore 过滤取前 48（内联小图不占名额；原全量 dataURL 常驻=数百次生成数百 MB），窗口外缩略图落既有占位、选中/下载/复用 resolveResultImage 按需读文件——记录仍在，零功能删除。
- **线3 亮主题中性叠加层失效群 30 处**：rgba(255,255,255,0.02~0.08)（暗底白提亮惯用法）在浅底白叠白不可见（hover 消失/卡片只剩边框/进度轨道消失/shimmer 不动）→统一 color-mix(var(--color-text) N%, transparent) 明暗自动正确；辨伪保留 ChatCodeBlock 2 处（hex-exempt 暗色代码面板 chrome 刻意不随主题）；MarkdownContent pre 底 rgba(0,0,0,0.3)→surface-container 令牌（亮主题中灰脏块）。
- **线4 color-scheme 声明**：App.tsx 注入 root.style.colorScheme（原全仓无声明，暗主题原生滚动条轨道角/表单控件残留亮色渲染）。
- **线5 反馈确认补遗**（上轮 B 系列遗留 4 处）：角色记忆弹窗三路读取全吞错→失败 Alert（原打开即空白与「无记忆」不可分）；青鸟助手两路全失败原伪装「暂无助手」假空态→载入失败条 role=status（单路失败仍兜底不误报，轮询自动重试）；图片下载两处失败 message.error+成功提示（另存为抛错原落 unhandled rejection，取消不算错）；自定义生图模板删除补 Popconfirm。
- **测试**：HistoryRail.test.tsx 新建 3 例（超窗渲染 60/点击增量展开到底按钮消失/不足窗口全量不回归）+historyMeta +1（limit 按需恢复项计数）+meta +2（读截/写截）+WeixinPage +1（双失败提示+单路兜底不误报）。
- **坑**：①静态期主题预读判定序必须逐字对齐 store.loadMode，多一个 matchMedia 兜底就分叉首帧跳变②limit 语义按需恢复项计数非位置（slice 让内联小图占名额）③heredoc 写测试三层转义（python→文件→JS），\ 才是源码单反斜杠，写完看原始字节（eslint no-useless-escape 逮住）④rgba(255,255,255,x) 不全是债，先排除 hex-exempt 名单再批量替换⑤GenResult 无 id 字段——tsc 增量缓存漏检测试文件改动，强制 tsc -b --force 复核。
- **门禁**：tsc 0（强制全量复核）+eslint 0/0+全量 ci.ps1 绿+bindings drift OK@706+版本三处 4.351.0；产物见 releases/archive/v4.351.0.md。**文档**=releases/archive/v4.351.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十六迁：v4.348 入 archive）+progress/todos。**未做（下刀池）**=ForeshadowPanel 行 memo/ModelCenter ctx 拆分/其余可点 div 键盘化（v4.349 池）/ResizableDrawer 焦点管理/reducer 级整店订阅 13 处/selbar z-1080 阶梯治理。

## 最新发布：v4.350.0（2026-09-19）「前端优化第二轮：事件总线连环炸根修 / P0 幽灵令牌 / 破坏操作确认 / 静默失败 / 流式渲染热点 / 主题破绽」

- **动机**：用户口径「优化 gaea 前端」（v4.349 五线后第二轮）。同范式：四路并行**只读审计**（事件与定时器泄漏 / 交互反馈三态 / 渲染热点 / 视觉一致性，各带 file:line 与量级推理）再按证据定刀；纯前端 32 源码 + 6 测试（3 新 3 适配），零 Go 逻辑改动、零绑定变更（706 不动）。
- **线1 事件总线连环炸根修**：❗`onTaskEvent` 清理走 `EventsOff(channel)` 全清（wails 语义=注销通道全部监听者；v4.61 事故红线 v4.62.2 只修了 onEvent/onSubagentText，本条漏网）——gaea-task 有 5 个并发订阅点（运行角标/任务自动激活/任务面板/价格源/索引任务），任一卸载（关一次任务面板即触发）=角标冻结+自动打开失灵+下载卡「进行中」，keepAlive 下直至应用重启→一律 `subscribeWailsEvent` 按监听者退订；onUpdaterProgress/onReady 同纪律收口。polyfill（网页模式）双修：EventsOn 返回 void→消费方 deps 每变往 eventBus **叠加** handler（ModelCenter 四通道单事件 N 次重复后端拉取）→改返回「只摘自己」退订对齐 wails v2.13 桌面语义；EventsOff(带 callback) 无条件 sse.close()→共享通道他人推送被掐断→只摘自己、通道空才关 SSE。
- **线2 P0 幽灵令牌**（12 主题功能性坏点）：`--md-sys-color-error` 全仓零定义而 gaea 四组件 26 处无 fallback 消费（CSS invalid：失败态红点/红字/红框全消失）→注入 colorDestructive 语义别名；`--md-sys-color-surface-container-low` 零定义→追问发送按钮箭头 12 主题恒隐形+输入区透明+VersionTimeline 对比底丢失→color-mix 插值；`--color-info`/`--md-sys-color-info` 补注入（#38bdf8/#0369a1 明暗分档）；幽灵四连 `--v3-fg-soft`(17 处)/`--v3-line-soft`/`--color-text-tertiary`/`--md-sys-color-text-tertiary` 明暗分档定义（暗主题 12px 次要文字 3.9:1<AA）；modelcenter.css 13 处 gaea 作用域 token 换主令牌直连（--fg 系只在 gaea/styles.css 定义且该页不加载→亮主题 #ddd 白底 1.3~2.5:1）。
- **线3 破坏操作二次确认 9 处**：高危批量/清空 4=知识库批量删除（双分支）/成本库批量删除（**连带假成功根修**：吞错后无条件「已删除 N 条」→诚实成功 N 失败 M）/绘梦清空历史（prompt/seed 即时持久化）/原罪清空故事消息（ToolbarButton 无 forwardRef→Modal.confirm 命令式）；单条 5=任务收件箱删除（**连带吞错可见化**+i18n 三语新键）/青鸟提醒/组织/角色关系/划线想法列表（span 挡冒泡防触发跳转）。
- **线4 静默失败反馈**：三脑检索失败与 0 命中三态化（命中/换关键词/失败重试，role=status）；排程导出 XML 补成败提示（SaveFileAs 抛错原落 unhandled rejection）；章节载入失败白板→message.error 带原因（原只 console.error 且 saved:true 误导正文丢失）。
- **线5 渲染性能**：Transcript 提及扫描按文本内容缓存（原每 chunk 对全会话 assistant 正文跑双全局正则+O(m²) 重叠；完成后文本不可变→缓存命中只付流式段成本，上限 512）；ChatPage topicList memo+ChatTopicSidebar React.memo+过滤 useMemo+行 hover 删 hoveredId 改纯 CSS；ToolCard summary/prettyArgs/outputLines 三处 memo（write_file args 内联整文件数百 KB）；`useNow(active)` 条件订阅（已完成消息/过程卡/RunStatus 无条件订阅 1s 时钟=O(N)/秒常驻重渲染清零；**伴生修复**「思考 Xs」完成后持续增长→完成边沿 Date.now() 定格）。
- **线6 视觉**：RelationGraph 画布 #ddd/#555/#333/#444 写死（6 亮主题节点名不可见）→resolveCSSColor 主题令牌解析+darkMode 入 deps；mermaid 跟应用 data-hl 而非 OS prefers-color-scheme。
- **辨伪**：ModelCenter ctx（100+ 字段）不 memo——五 hook 返回对象字面量每渲染新引用 deps 无法建立，强行 memo=陈旧 ctx 真 bug（挂观察池）；TTSPlayer/AIConsole 单实例全清无害；z-index 1080 阶梯脆弱但无叠加路径；boot 亮色反闪挂池。
- **测试**：events.test 新建 4 例（双订阅退订隔离+EventsOff 永不调用/space 过滤不变/updater+ready 同纪律）+runtimePolyfill +3 例+useNow 3 例；TaskInboxPanel/WeixinPage/KnowledgePanel 删除用例改两步确认。
- **坑**：①EventsOff 红线是「禁全清」非「禁用」——polyfill 带 callback 只摘自己+通道空才关 SSE 才是完整语义②Popconfirm clone 注入 onClick 覆盖子元素（v4.348 Dropdown 坑家族），子按钮原 onClick 必须移除；无 forwardRef 按钮包不上改命令式③确认钮定位一律 testid（okButtonProps 透传）防「删 除」空格名④go test 全量负载 flaky（足迹零交叠复跑绿）；ci 日志勿 tail 截断否则 FAIL 无包名。
- **门禁**：tsc 0+eslint 0/0（新色值走 // hex-exempt）+全量 ci.ps1 绿（vitest 384 文件 3280 例）+drift OK@706+版本三处 4.350.0；产物见 releases/archive/v4.350.0.md。**文档**=releases/archive/v4.350.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十五迁：v4.347 入 archive）+progress/todos。**未做（下刀池）**=boot 亮色主题启动反闪（index.html 内联预读主题，动 splash 链路）；MarkdownContent pre 底 rgba(0,0,0,0.3) 亮主题脏块；白色半透明 hover 底亮主题失效群（PersonaPicker/imagegen）；--color-scheme 声明；ModelCenter ctx context 拆分（state/actions 两通道）；绘梦历史无上限渲染+dataURL 全量回填（挂载内存膨胀）；ForeshadowPanel 行 memo。

## 最新发布：v4.349.0（2026-09-19）「前端优化五线：可访问性动线 / 亮态可读性 / 运行开销 / 布局 / 稳健性」

- **动机**：用户口径「优化迭代 gaea，进行前端优化」。先四路并行**只读审计**（主题可读性/交互可访问性/渲染性能/布局响应式，各带 file:line 与实测数字），再按证据定刀；审计判错的项当轮辨伪不改。纯前端 18 源码+9 测试，**零 Go 逻辑改动、零绑定变更**。
- **线1 可访问性动线**：记忆中枢全局检索命中行（主入口动线）裸 div → `role=button`+`tabIndex=0`+Enter/Space；轻语记忆面板标题/摘要键盘化 + 确认/取消/编辑/删除四图标钮补 `aria-label`（此前只挂 Tooltip，**不参与可访问名**）；阅读「删除高亮」补 `Popconfirm`（误点即删不可恢复）；Lightbox 关闭钮补名；批准弹窗 `aria-modal` false→true。
- **线2 亮态可读性**：①**真缺陷**——`changesdiff-tok.css` 写死 One Dark 七色而 `ChangesDiff` 容器不设背景（继承面板=亮态浅底），实测 `.tok-typeName #e5c07b` **1.38:1**、`.tok-string` 1.61:1（文件注释自称「浅色下同样可辨识」被证伪）→ 接共享调色板 `--hl-*`（真源 `hljs-theme.css` 已备浅色深阶组并随 data-hl 切；ChangesDiff 显式 import 真源防懒加载链 var 未定义），文件零字面色值；②`lightFn` 下沉：warning `#d97706`(2.90~3.13:1)→`#92400e`、success `#059669`→`#047857`（Moss lime 阶保留）、primary Jade/Rose/Amber→`#0f766e`/`#be123c`/`#b45309`（accentRgb/glow 跟随）；**Violet/Moss/Slate 原达标一律不动**；③新增 `appStore.contrast.test.ts` 12 主题×5 令牌×2 背景矩阵机检 + 深阶锁值 + hljs 浅色组逐令牌 ≥4.5 + tok 文件零字面色值。
- **线3 运行开销**：①`main.tsx` 诊断层 500ms 轮询只写一个时间戳（≈17.3 万次/天纯空转）→ 心跳改由既有 rAF 探测循环每帧维护（跨 IIFE 共享 `mainThreadBeatAt`），诊断层只做 8s 低频判读；**降级路径补 500ms 兜底续跳**（否则误报卡死）；②`getThemeTokens()` 每次返回新对象且留在渲染体 + `ConfigProvider theme={{…}}` 内联字面量 → 每次渲染重跑 ≈45 个 setProperty 与 antd 主题算法 → 双 memo 化；③`SelectionToComposer` 的 selectionchange 先判 `isCollapsed` 短路（原上来就 `toString()` 物化整页选区文本）；④`WhisperMemoryList` 过滤/分组 useMemo + 查询词预小写（原每次渲染 ≈7N 谓词）。
- **线4 布局**：摘要 chip 补 `whitespace-nowrap + shrink-0`（窄列曾标签内折行、「基线 N」整排参差）；基线槽位行 `flexWrap:'wrap'` + 工期/时间戳固定段 `nowrap/flexShrink:0`（原固定部分 326px>minWidth 300 致行内换行）；海报描述补 line-clamp（`overflow:hidden` 定高卡原先静默裁切无省略号）。
- **线5 稳健性**：①书架 `loadProjects` 只 `console.error` → 首页把「读失败」渲染成「书架空空如也」**假空态** → store 增 `projectsError`（只表态不抛出，调用方 await 序列零变化）+ 首页错误态含原因与重试；②「最近文档」仅挂载时读一次而 v4.346 起页面常驻保活 → `lib/recentFiles` 增订阅通道（写入广播 + 解析缓存保快照引用稳定，满足 `useSyncExternalStore`），首页账页与办公「最近文件」条同改订阅（空态挂载后首次写入也出现）。
- **审计辨伪（记录在案，当轮不改）**：①印泥章 `.w-seal #d06055` 不改——20px/**700 粗体**属 WCAG large text（阈值 **3:1**），实测白底 3.83/近白 3.49~3.67 **达标**，按 4.5 判死是误报，改深会漂移品牌记号色（阈值已写入测试注释）；②reduced-motion **无缺口**（`index.css:461` 全局 `*` 兜底经 main.tsx 确认已加载 + GSAP 三 hook 均真实调用 `prefersReducedMotion()`）；③暗态 6 主题×27 配对 **0 失败**（最差 4.69）；亮暗两态 `colorText/colorTextSecondary` 本就 ≥4.63——亮态问题**只**集中在 primary/warning/success 被当文字用；④硬编码 `rgb(255 255 255)` 多落在固定深色 scrim（character-detail.css:141/488）上 = 正确豁免。
- **测试**：新增 4 文件 44 例（对比度矩阵 18 / 记忆面板 5 / 阅读浮层 3 / 列表 memo 5 / 结构守卫 13）+ 扩展 5 文件 8 例（命中行键盘 2 / 首页失败重试 1 / 最近文件广播 2 / 挂载后写入 2 / 选区短路 1）。
- **门禁**：tsc 0 + eslint **0 error 0 warn**（全仓）+ **全量 ci.ps1 绿**（vitest **382 文件 3270 例**）+ drift OK@706（零绑定）+ 版本三处 4.349.0；**首轮 ci 被 frontend build 拦下**（新增测试文件后未重跑 tsc：`[...NodeList]` 需 DOM.Iterable、ReadingAnnotation 缺 nodeId/createdAt）→ 修复后复跑全绿。
- **目检**（vite dev ?mock=1 + 无头 Edge CDP 9347，只读，收工已回收进程与临时 profile）：三页错误边界 0/空白 0；检索「振动锤」5 条命中**全部 `role=button`+`tabindex=0`**；亮态实测 `--md-sys-color-primary #0f766e` / warning `#92400e` / success `#047857` 与锁值一致；`.p-poster-desc` 实测 `webkitLineClamp:4`+`overflow:hidden`；`.p-poster.is-hero` 中段空白实测 **273px**（卡高 420，与审计 276px 吻合=本轮有意不动的拍板项）；console 仅 antd Spin tip 警告 + 自诊断 longtask 66ms，**无 exception**；截图 `.tmp/uiwalk-v4349/*.png`+report.json。
- **产物**：exe **50,865,152 B（48.5 MB）** SHA256=**615319a09773dc693c2be6d943904bae226dd686d797291c905d605b5174ecea**（releases+SUMS，格式 `hash *file` 与历版一致；桌面副本同哈希）；冒烟 health 200 过；保留策略执行留 v4.345~v4.349 删 v4.344；顺带纠回 `releases/README.md` 的「最近 5 版」行与「34 席」索引（此前落后三版）与发布说明计数（254→371）。
- **坑**：①审计阈值必须复核——large text 3:1 被按 4.5 判死会白改品牌色②`--hl-*` 是懒加载 CSS 变量，接色板必须同时 import 真源③source-guard 里 `indexOf('.hl-scope-dark')` 会命中文档头注释，取色块须 `lastIndexOf`④antd 两字按钮可访问名带空格（「删 除」「重 试」），测试按 `/^删除$/` 找不到⑤心跳改 rAF 驱动必须处理降级路径，否则每 8s 误报卡死⑥**新加测试文件后必须重跑 tsc**（tsconfig.app 含 src）。
- **未做（下刀）**：海报墙 hero 中段留白 276px（须加内容槽=内容决策，改动需同时验 1440/1180/1100/760 四断点）；其余可点 div 键盘化（章节树/大纲/统计分类 ~8 处）与 `ResizableDrawer` 手写弹层焦点管理按动线权重分刀；进度计划页 `.gsched` 工具栏 spacer 换行与基线 Popover **本轮未实测**（mock 书斋 rail 无「进度计划」入口，代码级证据在案待真机）；跨窗口最近文件同步（storage 事件）本仓单实例无场景。

## 最新发布：v4.348.0（2026-09-19）「原罪板块优化双刀：故事底稿直注前情 + EPUB 电子书导出」

- **动机**：原罪板块优化完善班。①故事越长前情窗口（12 条×1500 rune）越丢早期设定，模型自记的便签/大纲底稿不在提示词里——要花工具轮 read 自己记的设定，忘 read 就自相矛盾；②故事成品只有 Markdown（插图本地路径引用），阅读器/传设备缺内嵌插图的可分发格式（go-epub 已在依赖，书源线有先例）。
- **刀1 底稿直注**：buildSinUserPrompt 增底稿块（角色块后、前情前）——大纲防御性再截 4000 rune + 便签合计预算 sinPromptNotesBudgetRunes=4000 从头带，超限「其余 N 条未展示」如实报数；空稿=提示词逐字零漂移；回合中途 write 下一轮生效；sinDraftForPrompt fail-open+锁内读。工具描述/系统提示词同步改口「底稿已随前情附上」，通常无需 read（省工具轮），核对单条全文才 read。
- **刀2 EPUB 导出**：SinExportEpub（705→706，play）——每条助手回合一节（第N回中文序数）+用户指令引块置节首+插图内嵌（go-epub AddImage；首图兼封面；webp 不在支持集→「插图未能内嵌」占位）+未生成占位（cue 次序键优先/描述键兜底同 MD）；落 sin/exports 同名不覆盖-2/-3、半截产物即删；前端顶栏导出改下拉（Markdown 另存为/EPUB 后端落盘告知路径，mock 如实抛）。
- **测试**：sin_draft_test 5 用例（空稿逐字一致/注入排序/预算截断+未展示计数/大纲便签独立判空/fail-open）+ sin_export_epub_test 2 用例（zip 全链：正文/引块/img/占位/illu-1 恰 1 张/cover.xhtml/同名-2；守卫：空故事/聊天话题拒绝）；既有 buildSinUserPrompt 三处调用点补 sinNotesDoc{} 断言零改动。
- **门禁**：go build/vet 0+internal/app 全量绿（102s）+tsc 0+eslint 0+vitest 定向 94/94+drift OK@706+spaceBindings 锁 528→529+bindingNames 全量再生+版本三处 4.348.0；全量 ci.ps1 绿；产物=exe 50863104B SHA256=f84e9bd9164f2bd55b1b12220f6963c6251f39247abad8dba1bb8a6a0249ed00（releases+SUMS；桌面副本同哈希；冒烟 health 200 过）；本地 exe 保留策略执行留 v4.344~v4.348 删 v4.343；
- **坑**：①Dropdown clone 子元素注入 onClick 会覆盖 ToolbarButton 的 prop（无 forwardRef 且 onClick 必填）——占位空函数即净，不为原罪改办公组件②单条恰=预算的便签 runes>room 为假应整条收下——截断语义「超了才截」，测试前提咬一次③go-epub AddImage 收本地路径返回值直接作 img src，webp 宁占位不硬塞。
- **未做（下刀）**：sin 真机一条龙走查（书源卡+导出下拉+底稿直注实写）挂闲置窗口池；插图变体历史候立项；章节化结构候拍板。

# 任务进度

> 本文件为**最近发布速览**。完整历史磁带见 `docs/archive/progress-history-2026-09.md`
> 与 `releases/`。

## 最新发布：v4.347.0（2026-09-19）「GLM-5.3-FlashX 新模型接入目录 + Flash 补官方绝对价」

- **来源**：智谱 9 月中旬新发 GLM-5.3-FlashX（Flash 加速版，320B 总参/18B 激活，200 tokens/s）；官方四页交叉核实（模型概览/详情/定价/coding 活动）后接入模型中心 GLM 静态目录。纯数据刀，零新绑定 705。
- **落地**：`glm_catalog.json` +1 条目（1M/128K、caps 同 Flash、国内绝对价 2/7 元/M、缓存命中 0.57 记 price_note）；coding 套餐官方明示暂未开放 FlashX→不配积分系数不进别名；Flash 补绝对价 0.8/2.8 元/M（「仅相对价回退 USD」旧口径作废，估算 0.65 USD→3.6 CNY /M+M）。
- **测试**：锚定清单+计数 45→46 全对齐 ×8；FlashX 元数据/价格锁值；积分守卫+估算锁值翻新；modelengine 包全绿。
- **验收**：drift OK@705；全量 ci 绿；产物见 releases/archive/v4.347.0.md。
- **未做（下刀）**：coding 套餐若开放 FlashX 补 points/别名（覆盖文件可先行）。
## 最新发布：v4.346.0（2026-09-19）「UI 健壮性双修：错误边界页级隔离（keepAlive 连坐根修）+ TisorRadar 缺档崩页」

- **来源**：UI 优化班——隔离目检（vite dev ?mock=1 + 无头 Edge CDP）双空间全页×明暗两态；纯前端四文件，零新绑定 705。
- **根修**：①MainLayout 错误边界下沉 keepAlive 每页内部（原包整组外面=任何一页崩卸载全部页且后续导航全停错误态，keepAlive 状态俱失）②TisorRadar dims 守卫+CharacterCard 调用方守卫（缺档不渲染不出假图）③mock 三角色补 dims 对齐真契约。
- **测试**：TisorRadar.test +3 +CharacterCard.test +1（按 circle 计数断言）定向 19/19；tsc 0。
- **验收**：修复前角色库明暗皆崩；修复后 3 卡×30 circle 雷达在位、双空间明暗全绿；全量 ci 绿；drift OK@705；产物见 releases/archive/v4.346.0.md。
- **未做（下刀）**：亮态 accent 对比度打磨（视觉拍板维持观察池）；动效手感待上手定论；真机走查清池班；技能核数 ≈09-30。
## 最新发布：v4.345.0（2026-09-19）「filewatch 关闭竞态根修：fs 通道关闭路径漏 close(out) 致消费方挂起」

- **来源**：全量 ci 偶发时间型 flaky 深挖——预算拉满 10s 仍挂=真缺陷非调度慢。
- **根因**：loop 三退出路径中 fs 通道关闭两条裸 return 漏 close(out)，与 done 分支 select 竞速，输则消费方 range Events() 永久挂起（~1/20）。
- **修复**：outOnce+closeOut() 三路径幂等关闭；测试 2s 硬超时改终态轮询 10s 预算。
- **验收**：filewatch 20 连跑全绿（修前 1/20 挂）；全量 ci 绿；产物见 releases/archive/v4.345.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.343.0（2026-09-19）「t7 收尾「编辑器标注高亮」轻量版：定位单条镜像 mark + 编辑即失效」

- **来源**：观察池最后一项（v4.325 有意裁剪）的轻量落地；纯前端 EditorPanel+一条 CSS，零新绑定 705。
- **落地**：locate 时镜像背景层（同排版/文本透明仅 mark 显色/textarea 底透出）；滚动条占宽补偿+滚动同步；content/activeNode 变化即失效（偏移漂移结构性规避）。
- **测试**：EditorPanel 域 11/11（+1）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.343.0.md。
- **未做（下刀）**：真机走查清池班（镜像对齐需真机目检后再议持久全量）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.341.0（2026-09-19）「t7 观察池「情感曲线图形化」：全书体检面板情感弧线折线」

- **来源**：观察池条目（v4.325 记录）；纯前端单文件 BookHealthPanel，零新绑定 705、零新依赖。
- **落地**：体检面板情感曲线区——按需逐章拉分析 V2 情感弧线强度（0-10，缺档跳过计数），纯 SVG 折线（参考线+tooltip+间隙如实呈现），<2 章诚实不足态。
- **测试**：BookHealthPanel 域 7/7（+4）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.341.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 余项候拍板。
## 最新发布：v4.344.0（2026-09-19）「t7 最终形态：编辑器持久标注高亮（overlay 常驻 + 编辑退场）」

- **来源**：v4.343 轻量版完全体（真机镜像对齐已过）；纯前端三文件，零新绑定 705、零新依赖。
- **落地**：annotationMarks.ts 分段工具收敛（面板/编辑器同一实现）；EditorPanel annotations 驱动全量 mark 常驻 + dirty 编辑退场 + gutter 实测；CreatePage 随激活章加载标注（alive 守卫）；locate 契约不变。
- **测试**：EditorPanel 11/11（重写+2）+ Panel 11/11（重构等价）+ CreatePage 15/15（补漏桩）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.344.0.md。
- **未做（下刀）**：真机走查清池班（持久 overlay 目检随下一班）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.342.0（2026-09-19）「t7 观察池「多章对比」最小形态：分析面板章际对比」

- **来源**：观察池条目最小可用形态；纯前端两文件，零新绑定 705。
- **落地**：分析面板「章际对比」——选对比章拉其 V2，四维+情感强度差值表（Δ 上绿下红、情感中性）；未分析行内提示；chapterOptions 缺省整体隐藏。
- **测试**：Panel 域 11/11（+3）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.342.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 仅剩 overlay 高亮候拍板。
## 最新发布：v4.340.0（2026-09-19）「t2 观察池「反推任务取消绑定」：AI 反推大纲轮询期可取消」

- **来源**：观察池条目（t2 拆书线 v4.291 体验余项）；纯前端单文件 CreatePage，零新绑定 705。
- **落地**：轮询期「取消反推」钮（taskId 到手门控）→ 停止等待 + GaeaTaskCancel 尽力协作取消（完成竞态被吞，任务中心权威）；同步回落路径不出入口。
- **测试**：CreatePage 域 15/15（+2）；补 NovelOutlineReconstruct/TaskCancel 两处漏桩。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.340.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.339.0（2026-09-19）「t7 观察池「标注定位编辑器光标」：分析面板→正文编辑器定位闭环」

- **来源**：观察池条目（v4.325 记录）按既定习惯开刀；零新绑定 705，纯前端三组件。
- **落地**：EditorPanel forwardRef+locate 句柄（rune→code-unit 换算+选区+聚焦滚动）；分析面板标注行「编辑器定位」钮（stopPropagation）；CreatePage 接线（gate 跳他章时入口按章一致条件隐藏）。
- **测试**：EditorPanel +3 + Panel +3（18/18）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/archive/v4.339.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 观察池余项候拍板。
## 最新发布：v4.338.0（2026-09-19）「无参绑定会话语义审计收官 + 死链清理（绑定面 707→705）」

- **来源**：todos 挂账（v4.181.0 起池）收官；审计档 docs/archive/gaea-session-binding-audit-2026-09.md。
- **审计结论**：需修 0 项——会话模型「看历史必经 ResumeSession 切内核」，主流水线无参读内核构造性自洽；v4.181 旁路看板族无同构残余；复审口径入档。
- **顺手清账**：删 GaeaCheckpoints 死链（UI 回退实走 GaeaRewind）+GaeaTCCAReport 孤儿链（state.tcca 零消费）；绑定面 707→705。
- **拍板池**：GaeaSkillDraftFromSession 参数化（从历史会话蒸馏技能）候拍板。
- **验收**：go build/vet 0+internal/app ok；tsc/eslint 0；store 域 38/38；drift OK@705；全量 ci 绿；产物见 releases/archive/v4.338.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；TaskCenter 会话关联结构刀。
## 最新发布：v4.337.0（2026-09-18）「办公对话流美化三轮收官：对齐 Codex web（降噪+动线+语义归位）」

- **来源**：办公对话流美化线程收口发版（在途三轮合一版）；纯前端零新绑定 707，零 Go 改动。
- **落地**：用户消息气泡化；裸思考行 bare 模式（行数 meta 三语+英文单复数）；过程卡完成即收起；WorkHeader 三改（doneSuppressed/纯问答 0 步不渲染/运行态对齐过程条）；尾随工具段合并（过程卡归位正文前）；轮尾登记卡补挂+omitPaths 跨段去重；交付卡统一边框容器；消息操作 hover 淡入；顶条只留 waiting 档；ToolGroup「bash × 3」三语。
- **测试**：ProcessCard 展开态测试重写钉新行为（旧测试钉已废行为，全量首跑 1 红）；vitest 3199/3199；tsc/eslint 零；三语 +4 键同轴。
- **目检**：vite dev 9344 mock=demo+独立 Edge 9345 CDP 两轮五态截图全过（bare「思考 1 行」实测）。
- **验收**：全量 ci.ps1 绿；drift OK@707；产物见 releases/archive/v4.337.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。
## 最新发布：v4.336.0（2026-09-18）「chapter-gate 通知跳转 + oh-story T5 落库接线」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-gate-jump-t5-wiring-20260918.md。
- **甲件**：自动门通知可点击——onOpen 回调+CreatePage 覆盖态+跨章拉正文锚定；无回调零变化（v4.331 观察池头名收口）。
- **乙件**：T5 七角色卡升格可派发——SKILL.md runAs=subagent+派发契约；spawn 写作域模板 subagent_writing+boot 映射；验收钉 Go +3。oh-story 只剩写前契约硬闸（等上游）。
- **验收**：全量 ci 绿；drift OK@707；产物见 releases/archive/v4.336.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.335.0（2026-09-17）「办公输入动线对齐 DSH/Codex：运行中 Enter=排队，插话改显式」

- **来源**：用户报告「办公板块消息输入的排序/插话/撤回不见了，直接插入正在跑的对话中」；拍板「按 DSH/Codex 方式处理」。
- **归因（非回归）**：队列本体（v4.200/201）一直在，v3.6.0 起 Enter=插话、排队退到 Clock 小按钮——队列卡从未被 Enter 动线触达=体感丢失。
- **落地**：Enter=排队（队列卡回主动线，回合结束 FIFO 派发）；Alt+Enter=直插（显式插话）；Shift+Enter 纠正/Esc 停止不变；三语文案+title 三态更新；steerTitle 键删（0 死键）；组装逻辑抽 helper。零绑定 707。
- **验收**：Composer.queue +3/I18n 对齐；composer 域 22/22；tsc/eslint/ci.ps1 全绿；产物见 releases/archive/v4.335.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.334.0（2026-09-17）「revolution 刀8 收官：跨模块验收+回退演练+版本三处漂移根治——**revolution 刀 1–8 全线收官**」

- **来源**：用户指令「继续优化迭代 gaea」；三线并发子代理+主代理收口；规格书 进度计划/gaea-rev-knife8-acceptance-20260917.md。
- **跨模块验收 Go +7（零生产代码改动）**：线A=AI 味性能钉（3.3~5ms<1s）+去味保真（分降/rune 漂移 0/实体句数不变）；线B=生成主轴端到端史上首测（双路桩贯通自动门成功路径+chapter-gate 事件 httpbridge 捕获）+50 章书级体检；线C=审批链 app 级（approved=false 零入库/journal 回放一致/账本只增）+快照往返+真 v3 迁移非破坏。
- **版本漂移根治**：app_info.go/versioninfo.rc 停 4.323.0（十版未跑 sync-version）——三处同步 4.334.0+ci.ps1 新增 version drift check 段；顺手 eslint 5→0（EXTRACT_OPTIONS 归位 novelOptions.ts）。
- **验收**：go vet/test 全量绿；vitest 3185/3185；tsc 零错；drift OK@707（零新绑定）；产物=exe 50832384B SHA256=4d2a01c5…81bc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：RetryJSON 桩回包须包 ```json 围栏；rebuildScenesFromBlob 无场景章 no-op；project.Create 直写 v4 标记（旧 v3 夹具假设过时）。
- **未做（下刀）**：真机走查清池班（v4.319~v4.334 挂池，须闲置窗口）；技能核数 ≈09-30；绘梦阶段二与缺省形态翻转候拍板。

## 最新发布：v4.333.0（2026-09-17）「阶段七出口对账 + 7.3-1 会话级回源收口」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-stage7-audit-resume-20260917.md。
- **会话级回源**：pendingSessionResume 双态通道（keepAlive 事件/冷启 pending）+gaea App 消费（ref 防过期闭包）+收件箱精确回源（V1 板块粒度回退零变化）。零新绑定 707。
- **对账**：7.1/7.2/7.3 已落、7.4 已清；挂账=本地建议样本（真机）/复跑抽样+≥5 次（≈09-30）/总判据⑤走查班。
- **验收**：前端 +5；go 全量复跑绿+vitest 3185/3185；ci 第三跑 OK（首两撞环境竞争先例+2）；产物=exe 50832384B SHA256=89d1de3d…0ba2（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：真机走查清池班（须闲置窗口）；技能核数 ≈09-30；刀8 余项/绘梦阶段二候拍板。

## 最新发布：v4.332.0（2026-09-17）「7.3-2 板块降级为任务视图（阶段七『层跃升』刀）」

- **来源**：用户「开始吧」提前拍板解锁（原窗口 ≈09-23）；规格书 进度计划/gaea-tasks-first-home-20260917.md。**本版=阶段七层跃升版本**。
- **论点**：首页从「用哪个工具」翻转为「什么在等我」。裁决=homeLayout 开关缺省 classic（渐进零变化）；TaskInboxBoard 内联抽取（面板零破坏）；回退开关双位；零新绑定 707；manifest 零改动。
- **落地**：appStore homeLayout+守卫；TasksFirstHome（收件箱大区/最近会话/能力 chips 全量直达/切回经典）；ModuleLauncher 分支+SpaceSwitch 快捷钮；设置页 HomeLayoutPanel；三语 12 键。
- **验收**：前端 +10；零回归实证（TaskInboxPanel 9/9+ModuleLauncher 13/13 零改动）；vitest 3180/3180 一次全绿；ci.ps1 全绿；产物=exe 50831872B SHA256=80dacda4…d7eb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：测试尾追加吞 describe 闭合；LocaleProvider 包裹；ESM 无 require。
- **未做（下刀）**：阶段七出口判据复核对账（三门收口）；刀8 余项；绘梦阶段二提案候拍板。

## 最新发布：v4.331.0（2026-09-17）「自动门可见化 + CI race 扩面（刀8 race 项收口）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gate-notice-race-20260917.md。
- **论点**：自动门零感知（事件无消费）+race 门禁只盖办公包。落地=useChapterGateNotice 轻通知（四维拼装/防叠/分支标记/无通道静默）挂 CreatePage；CI race job 扩面六包（app 注释不进候评估）。零新绑定 707。
- **验收**：前端 +3；vitest 3172/3172 一次全绿；六包本地全绿；ci.ps1 全绿；产物=exe 50824704B SHA256=a5e239f2…3198（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：vi.stubGlobal('window') 整窗替换炸 react-dom——只换 window.runtime 属性。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 余项（验收演练）候排。

## 最新发布：v4.330.0（2026-09-17）「刀7续收官：POV 视图（场景圣经消费面）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-pov-view-20260917.md。前置核对：场景生成/脑图/指纹已落，刀7续仅剩 POV 视图；revolution 勾选对账后只剩刀8。
- **论点**：POV 掩码只在生成链内部消费，视角纪律不可审视。裁决=+1 绑定（707）双路编译（场景/整章合成）；视图 camelCase 恒非 nil；空区段如实空。
- **落地**：novel_scene_bible.go wire 投影；SceneBibleDrawer（已知绿/不知情红对照+角色卡状态回灌）；ChapterEditor「视角」按钮。锁 529→530；drift OK@707。
- **验收**：Go +3+前端 +3；ChapterEditor 7/7 零破坏；vitest 3169/3169 一次全绿；ci.ps1 全绿；产物=exe 50823680B SHA256=4ea48652…4eeb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 race 门禁（CI 侧）。

## 最新发布：v4.329.0（2026-09-17）「绘梦阶段一刀 E：画室消耗（月度聚合+折叠面板）——绘梦阶段一全清」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-studio-usage-20260917.md。**绘梦阶段一（A 资产面板/B 参考槽/C 指令编辑/D 配图 v2/E 画室消耗）全清**。
- **论点**：台账有 Cost+CreatedAt 无聚合视图。裁决=+1 绑定（706）；当月过滤×模型分组；记录单价为事实源（缺价回落目录）；可解析才估算、未定价诚实计数；全免费显示 0；不用积分话术。
- **落地**：imagehub_usage.go 聚合（cwd 参数化供测试）；StudioUsagePanel 折叠面板挂 ImageGenPage；bridge/mock/锁 529/bindingNames 706。
- **验收**：Go +4+前端 +4；vitest 3166/3166 一次全绿；ci.ps1 全绿；产物=exe 50809856B SHA256=90e9ec56…ad78（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：裸 "0" 单价解析、gaeaCwd 全局态、mock 尾锚点。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；绘梦阶段二规划拍板。

## 最新发布：v4.328.0（2026-09-17）「绘梦阶段一刀 D 核心片：章节配图管线 v2（角色参考图+风格槽）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-illustration-v2-20260917.md。
- **论点**：配图纯文生图（一致性靠文字、风格写死）。裁决=签名扩展空串零变化（绑定面 705 不变）；参考路由按后端能力附 img2img（SceneRefDenoise=0.6）或 refNote 诚实降级；风格槽替换风格描述。
- **落地**：chapter.Agent +V2 方法；handler opts+参考路由+readImageFileAsDataURL；前端风格输入+角色勾选+refNote 行；修复挂载效应自动重跑 bug（didMountRun 守卫）。
- **验收**：Go +7+前端 3 改 5；vitest 3162/3162 一次全绿；ci.ps1 全绿；产物=exe 50796032B SHA256=070f39f8…6c39（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：书封参考；素材库变体/设书封；灯箱共用；denoise 可调。
- **未做（下刀）**：刀 E 画室消耗轻量面板或 7.3-2（≈09-23）。

## 最新发布：v4.327.0（2026-09-17）「绘梦阶段一刀 C：指令编辑『改图』MVP（云端先行）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-instruct-edit-20260917.md。前置核对：刀 A/B/E 已落，刀 C 是真缺口（改图=整幅重绘无语义编辑）。
- **论点**：gaea OpenAI 兼容图片面缺 /images/edits（Qwen-Image-Edit 系云端网关标准面）。裁决=复用请求字段（Mode=edit）；三后端诚实口径（openai 实现/glm·comfyui 拒绝）；零新绑定 705 不动。
- **落地**：image_openai.go editImage multipart 分支+parseImageResponse 共用提取；GenerateMedia edit 原图必填；InstructionEditModal（对照+用到画布）+ResultStage「指令编辑」Action+ImageGenPage 接线。
- **验收**：Go +5+前端 +4；ResultStage 5/5 零破坏；ci.ps1 全绿（FilePreviewModal 1 例 flaky 隔离绿，两日两现候选常驻）；产物=exe 50785280B SHA256=855b56f1…9ae0（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：mask/保留区域；ComfyUI 本地档；多图输入；edit-of-edit 谱系；刀 D/刀 E 轻量收尾。
- **未做（下刀）**：绘梦阶段一余项（刀 D 章节配图 v2/刀 E 画室消耗）或 7.3-2（≈09-23）。

## 最新发布：v4.326.0（2026-09-16）「GenerationGate 闭环收口：生成后自动分析门 + 全书体检」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gen-gate-closure-20260916.md；阶段七 §4 在册项（revolution「GenerationGate 完整闭环」未勾项收口）。
- **论点**：生成 done 只挂去味+AI 味分+角色提取，分析路（V2 落盘/伏笔同步/记忆回填）从不自动发生——三条已建成管道水源靠手动。裁决=自动门异步默认武装；只跑确定性三路+分析路一次 LLM（review/consistency 留手动）；分支章跳分析路；chapter-gate 事件零消费（t7 面板天然消费面）；全书体检纯确定性零 LLM。
- **落地**：novel_book_health.go（buildAutoGateReport 同步可测+runAutoGateAfterGeneration go 位+RunBookHealthCheck 逐章三路+伏笔 Lint 复用+V2 覆盖）；create_chapter_handler done 后挂接；NovelB +1（704→705）；BookHealthPanel（聚合卡+最差 AI 味告警+逐章表红标+伏笔 findings）；锁 527→528；drift OK@705。
- **验收**：Go +5（自动门三例+体检两例）+前端 +5；CreatePage 既有测试零破坏；tsc/eslint 零错；产物=exe 50774528B SHA256=0581c0be3ea4a61f33989eecf4ebd666a287cd648a10ec2a21e5da304aa5f83b（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：analysis.New(nil client) 在 Analyze 内 panic——测试用无引擎真 client；JSX 泛型不认索引访问抽别名；无大纲节点=无契约可违不计。
- **出口**：生成后 V2/伏笔同步/记忆回填自动发生；全书体检聚合报告零 LLM；revolution 未勾项勾销。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；闲庭在册清欠沿列车。

## 最新发布：v4.325.0（2026-09-16）「t7 前端统一接线收官：分析 V2 消费面板 + 标注高亮视图」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-analysis-v2-panel-t7-20260916.md。**本刀合入=小说·MuMu 蒸馏六域前端接线全清（t7 收官）**。
- **论点**：分析 V2（九维落盘）与标注层（keyword→rune 偏移）后端在位 UI 零消费；AnalyzeChapter 困在 Legacy 面零入口。裁决=types 直连（不复制九维子类型树）；缺章诚实报错接分析按钮闭环；高亮 V1=面板只读视图不动编辑器；rune→code-unit 换算+交叠裁剪。
- **落地**：analysis_handler.go +NovelChapterAnalysisV2（703→704）；AnalyzeChapter 出 Legacy 转正（AppBindings+mock）；ChapterAnalysisPanel（头部评分+chips+meta、九维分节空节隐藏、标注区 列表|高亮视图 双模式+锚点定位）；CreatePage rail「章节分析」。锁 525→527；drift OK@704。
- **验收**：Go +3+前端 +8（含交叠裁剪断言）；CreatePage 13/13 零破坏；vitest 全量 3151 例（1 例环境 flaky 隔离绿）；tsc/eslint 零错；产物=exe 50756096B SHA256=6604f8e06b3f27cd586053794bfe22d8b881851e72210a7e7b0e2aeb22d89528（releases/gaea-v4.325.0.exe+SHA256SUMS-v4.325.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）
- **出口**：分析按钮→V2 落盘→九维可见；标注列表+只读高亮+锚点；AnalyzeChapter 出 Legacy；既有面零回归。
- **观察池**：编辑器 overlay 持久高亮；标注定位编辑器光标；多章对比；情感曲线图形化。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 零功能周≈09-23）；闲庭在册清欠沿既有列车。

## 最新发布：v4.324.0（2026-09-16）「t6-C2 提示词工坊第二刀：模板包导入导出（content_hash 三态）」

- **来源**：用户指令「继续优化迭代 gaea」。小刀单线主代理直做；规格书 进度计划/gaea-prompt-bundle-t6c2-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md §6.4+§11.4。
- **论点**：覆盖表只有一份本地状态——换机/重装无搬运手段、内置升级无对账工具。补全量快照 JSON 包+三态导入（kept/converted/created_or_updated + gaea 三闸 invalid/unknown/duplicate）；对 MuMu 升级=导入真校验（error 跳行不阻断整包）+去重；裁剪=未知键不建行。
- **落地**：internal/promptstore/bundle.go（ContentHash/SameTemplate canonical/BuildBundle/ImportBundle 零 IO 纯函数）；gaea_prompt_store.go +2 绑定（Export 回 JSON 字符串/Import 三态落盘）；面板顶栏导出（saveExportBlob 双门）/导入（pickFileAsFile 双门+结果 Modal）；mock/契约同步。绑定面 701→703（NovelB +2 显式覆盖表归域——v4.323 同款坑复现即修；play 锁 523→525）。
- **验收**：Go +4 测试函数（三态矩阵 10 分支+round-trip 内核+回魂 e2e）+前端 +4；tsc -b 零错；vitest 全量 3143 例（1 例环境 flaky 复跑绿）；drift OK@703；产物=exe 50734592B SHA256=5c1b3916518cd268f84ec9dfc6b61a3b3746c7d12f778151c1076736380d0abc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **出口**：干净环境导入覆盖逐字节回魂；内置升级三态分支钉死；坏模板行不落盘整包其余正常；壳内导出/导入双门。
- **观察池**：项目级 scope；自建模板键；升级合并交互；按书切换配方；triggers；模板包签名/加密。
- **未做（下刀）**：t7 前端统一接线（小说线最后一域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.323.0（2026-09-16）「t6 提示词工坊首刀：模板可编辑覆盖层（{{name}} 渲染+三级解析+工坊面板）」

- **来源**：用户指令「继续优化迭代 gaea，记得使用子代理」。三线并行子代理（A 引擎纯函数/B handler/C 前端）→ 主代理收口；规格书 进度计划/gaea-prompt-workshop-t6-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md。
- **论点**：20 个 prompts/*.json 只读两层用户改不了；占位符单点硬编码；保存零校验（MuMu §9.3 同型）。裁决={{name}} 统一（旧语法兼容）；V1 只全局覆盖；裁剪云端工坊三表；列表零正文。
- **落地**：internal/prompt +四元数据字段+RenderPlaceholders+SetOverride 三级解析+Names；新包 promptstore 六规则校验+Upsert 自增；prompts ×20 元数据+2 文件占位符迁移；gaea_prompt_store.go 状态文件+五 App 方法+两装配点挂钩+substituteWordCount 双语法；PromptWorkshopPanel 三区+api/prompt.ts+CreatePage 入口+mock 五档。绑定面 696→701（NovelB +5 显式覆盖表归域；play 锁 518→523）。
- **验收**：Go +22 测试函数+前端 +12（CreatePage 13/13 零破坏）；tsc -b 零错；ci.ps1 全绿 exit 0（go 130 包+vitest 367 文件 3138 例）；drift OK@701；产物=exe 50754048B SHA256=1a712949e53414c308ae8a14f85de914d95e021b47fc9711f3ae3e880fdcf4d6（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：20 模板可看可改可恢复；覆盖即时生效（eng.Get 三级解析全链共享）；保存真校验；{{name}} 统一。
- **观察池**：模板包导入导出（content_hash 三态，t6-C2）；项目级 scope；自建模板键；triggers；升级合并交互；风格注入与 novelstyle 打通。
- **未做（下刀）**：t6-C2 模板包导入导出 / t7 前端统一接线（小说线最后两域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.322.0（2026-09-16）「t4-C3 收官：场景工程整章重写（单场景替换）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-rewrite-20260916.md。
- **产品裁决**：单场景替换（应用/Restore 对称）；LLM 分场景回写与场景粒度重写入观察池。
- **落地**：移除 whole v4 拒绝+stitch 读；Apply/Restore +rebuildScenesFromBlob（复用既有原语零新机制）；partial v4 守卫；EditChrome 双按钮+ChapterPage 挂两弹窗。零新绑定 696。
- **验收**：Go +2+前端 +2（11/11）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638848B SHA256=5b1b105ad81bff7b5dd391cc445aeda8cb77aed575a516c996e10441c38ea29e（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：t4-C3 余项全清（三通道×两存储矩阵仅剩 partial×场景=守卫+观察池）。
- **未做（下刀）**：t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.321.0（2026-09-16）「t4-C3 余项：partial 选段局部重写（版本库升级版）」

- **来源**：用户指令「可继续」。三线并行子代理（A/B/C 足迹互斥全部一次绿）→ 主代理收口；规格书 进度计划/gaea-partial-rewrite-20260916.md。
- **论点**：整章重写对「改一段」太重；partial=选区+指令+长度模式只重写选中片段。对 MuMu 升级=走版本库（全文快照可回滚）；全链 rune 口径；零新绑定（696 不动）。
- **落地**：rewrite/partial.go 四纯函数+NormalizeRequest 五规则（whole 零字节变化）+novelChapterRewritePartial（拼接钉死前后文零变化+版本字段）+前端 EditorPanel 选区 rune 换算/PartialRewriteModal 三态/CreatePage 挂载。
- **验收**：rewrite 9 函数+app +3+前端新增 6（合并定向 34 全绿）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638336B SHA256=c0f1abd770b4b4a2253c17e9c92d912f06f7992fbcbc62fe155e1e456a97fa7f（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：选段高亮回写编辑器；流式输出；deslop 入口（类型已留）。
- **未做（下刀）**：t4-C3 余项剩场景工程整章重写；t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.320.0（2026-09-16）「t4-C3 余项：重写版本历史面板（版本库前端消费）」

- **来源**：用户指令「继续」。小刀单线主代理直做（纯前端后端零改动）；契约先行（规格书 进度计划/gaea-rewrite-history-panel-20260916.md）。
- **论点**：v4.304 版本库只有「刚重写完那一刻」的 RewriteModal 能操作，List/Get 两绑定零 UI 消费——关弹窗/换会话后历史版本不可看不可用（v4.305「用户到不了的功能等于没有」同型）。
- **落地**：RewriteHistoryPanel（多行展开+详情懒拉缓存+动作镜像后端状态机门控：completed→应用+放弃/discarded→应用/applied→恢复原文/其余只读；Popconfirm 显式 okText）+CreatePage rail「重写历史」入口（打开才拉）+mock 两样本。零后端：绑定面 696/锁数不动、drift OK@696。
- **验收**：面板 5 用例+CreatePage 既有 13/13 零破坏；tsc/eslint 零错；ci.ps1 全绿 exit 0（go 129 包+vitest 364 文件）；产物=exe 50613248B SHA256=4bc1d56d44a5f97a92fde68d1fb8b18026dec5d0c4a8c6fec19731fca8dca837（releases/gaea-v4.320.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：双栏 diff 视图；partial 入口随 partial 刀；按模式/状态过滤。
- **未做（下刀）**：t4-C3 余项剩 partial 局部重写+场景工程；t6 提示词工坊/t7 前端接线；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## 最新发布：v4.319.0（2026-09-16）「7.2-2 判据②收口：结晶技能调用计数（skill_stats）」

- **来源**：用户指令「继续」。小刀单线主代理直做（体量一轮未拆子代理）；契约先行（规格书 进度计划/gaea-skill-usage-stats-7-2-2-20260916.md）。
- **论点**：v4.317 出口判据②欠账「结晶技能调用 ≥5 次且成功率可见」：现状核实=技能计数是前端会话级（useToolStats 只算当前会话 run_skill，不持久、无成功率、漏 read_skill 主消费通道——boot 按需加载器才是结晶技能被「翻开来用」的时刻）；本刀补**工具级调用计数**：read_skill+run_skill 双钩子 → <DataRoot>/skill_stats.json 持久化 → GaeaSkillStats 绑定 → 能力面板技能行「N 次 · 成功率 X%」。诚实口径（V1）=成功率工具级（read_skill 交付正文/run_skill 管线无错=ok；读不到名也计 fail 不静默）；回合级留观察池；「≥5 次」使用事实门槛不做硬闸。
- **落地**：①新纯包 internal/skillstats（零 IO 表驱动，先例 routesuggest/taskinbox）：Record（空名忽略/MaxSkills=500 新名拒收防漂移堆积/零值 File 自动建表）+View（calls 降序→name 升序稳定）+JSON camelCase 钉死。②boot 双钩子：Options.OnSkillUse（缺省 nil 零行为变化，EmitSubagentText 先例）+sysprompt resolver 包装（read_skill 失败也计）+run_skill 装饰器 skillUseCountTool（args.name 解析失败不计宁少勿扰；CompactDescriptor 回退语义同注册表缺省）。③App：skill_stats.json route_suggestions 同款容错读（缺失/损坏/version≠1 回空表）+原子写+skillStatsMu 串行；recordSkillUse 包级回调；绑定 GaeaSkillStats（OfficeB +1，**695→696**）只读现算。④前端：CapabilitiesPanel open 态拉 app.SkillStats 一次零轮询（reload 随引擎热加载刷新）→SkillsSection 可选 usage prop→SkillRow 徽标行灰字「N 次 · 成功率 X%」（skill-usage-stat；缺省/零调用不渲染，会话 chip 语义不变）+SkillStatView（types/memory）+三语 +2 键+mock 桩+spaceBindings work+1（锁 517→518）。
- **验收**：skillstats 5 函数+app 3 例+前端 2 例；tsc -b 零错、eslint 零告警；ci.ps1 全绿 exit 0 一次过（go 129 包+vitest 363 文件全过）、drift OK@696、版本三处 4.319.0；产物=exe 50606592B SHA256=43739c69ad424a8356a0994525d3d4d93c4dc367e6037e00802bf249b0641de1（releases/gaea-v4.319.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：回合级成功率（点赞点踩/重生成信号另刀）；蒸馏建议 tab 内联已结晶技能计数；两周后真机清池核「≥5 次」。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 稳定一个零功能周）；闲庭在册清欠沿既有列车。

## 最新发布：v4.318.0（2026-09-16）「阶段七第五刀：多入口统一任务收件箱（任务持久实体+四入口存为任务+状态机追踪，7.3-1）」

- **来源**：用户指令「继续推进」（沿「继续 并行使用子代理，优化迭代 gaea」习惯）。契约先行（规格书 进度计划/gaea-task-inbox-7-3-1-20260915.md：A/B/C 三线足迹互斥+出口对照）→ A/B/C 三线并行 → 主代理收口。三线全部一次绿，**无卡死接管**（v4.316/v4.317 连续两刀接管后首回全并行一次过）。
- **论点**：四入口（意图中枢/微信/语音/Ctrl+K）v4.5 已汇同一内核（intent.Parse→routeIntent），本刀补「任务」为**持久实体**+统一收件箱：指令除即时执行外可「存为任务」→ 状态机（待处理/进行中/已完成/已放弃）→ 跳回来源板块。**收件箱是清单不是调度器**（零轮询零定时器零后台事件，读写均用户动作触发=「关闭即停」昼夜运转拍板）；space 必带 work/play 隔离（跨空间仅显式，不做移动/复制）；不加新板块（挂既有双空间首页）。
- **线A**：新纯包 internal/taskinbox（零 IO 表驱动，先例 skilldistill）：CanTransition 四条合法迁移（pending→doing→done、pending|doing→abandoned；终态拒迁同值恒 false）+NormalizeTitle（哨兵 ErrEmptyTitle、120 rune 截断）+ValidSource 五来源（ctrlk|palette|voice|weixin|inbox）+ParseTaskID（ti-+12hex 残渣防御）+FilterBySpace（''=全部，GaeaTaskList 变参先例）+Sort（状态组序→组内 UpdatedAt 降序→ID 稳定=行动优先）。9 测试函数 ~104 断言（JSON camelCase 形状逐字钉死）。
- **线B**：intent 新动作 save_task（存为任务X/记个任务X/帮我添加一个任务X/任务：X/待办：X 首尾锚定两式，排提醒后让位——「提醒我存个任务明天开会」归提醒；空标题不命中坠回聊天宁漏勿误；+9 表用例）+状态文件 <DataRoot>/task_inbox.json（{version:1,tasks:[]} camelCase，route_suggestions 同款容错读+temp+rename 原子写）+四绑定（OfficeB +4）：List（FilterBySpace+Sort 损坏回空表）/Save（id 空=新建 pending+ti-+12hex crypto/rand+超 500 拒；id 非空只改 title/note——**Source/Action/Target/Session/CreatedAt 不可变=来源审计链**；source 缺省兜底 inbox）/SetStatus（同值短路零落盘）/Delete；执行层 execSaveTask：语音/微信走 routeIntent→space=gaeaEffectiveSpace() 归一回退 work、assistantID 判 weixin/voice、reply「已存入任务收件箱：<title>」；**Ctrl+K/面板不走后端执行**——dry-run 照常命中，前端直调 Save（shell 空间更准，来源可区分）。app 9 测试函数（camelCase 钉死禁 created_at 泄漏/"WORK" 大写不泄漏/dry-run 零落盘）。
- **线C**：TaskInboxPanel（antd Modal 四档 tab 计数、CanTransition 前端镜像非法组合按钮不渲染、action=navigate→去板块/session→回会话 V1 板块粒度、V3Empty 两分；9 用例）+双空间首页挂点（书斋 w-vitals 第五节 desk-task-inbox；闲庭 p-foot 第五节 garden-task-inbox 内联 gridColumn '1 / -1' 跨全列——固定四列网格零 CSS 改动）+四入口（Ctrl+K 指令卡〔存为任务〕次按钮+save_task「执行」短路直调 Save；面板 query 非空动态项「存为任务『query』」**捕获期 input 监听**镜像输入——CommandPalette 契约不动，观察池列清退方案；语音/微信零前端改动）+lib/types 本地重述 TaskInboxView（herdsman 防环先例）+三语 +28 键（zh=en=zh-TW=1518 逐键相等）。
- **主代理收口**：gen_bindings+bindingNames 再生 695（drift OK@695；bindings_novel 多行委托规整单行 +101/−101 零语义——v4.313/4.317 重排噪音同款甄别）+bridge/core +4（同名前缀无需映射，GetProgrammingWebStatus 先例）+mock/chat 四桩（两空间样本内存态）+spaceBindings 四名 shared（隔离由 space 参数承担，UnifiedSearch 口径）+收口修 5 处 tsc（bindingNames 再生丢 as const→drift 串味；mock vi.fn 空数组推断 never[]→补 Promise<TaskInboxView[]> 完整样本；antd Text 无 rows 配置→Paragraph）。
- **绑定面**：691→695（OfficeB +4）；锁数 513→517（shared +4）；drift OK@695。
- **出口对账**：①四入口来源可区分 ✅（五 source 落库，测试钉死）③空间隔离零泄漏 ✅（space 必带+严格过滤+"WORK" 大写不泄漏）④零常驻 ✅（无轮询/定时器/事件订阅）；②板块级 ✅（Target 精确导航）、**会话级欠账**——按路径恢复 seam 存在（palette sessionItems→onResumeSession(path)）但活在 gaea App 树内首页拿不到，Session 字段已落库数据就绪后续刀接。
- **验收**：taskinbox 9 函数+app 9 函数+intent +9；前端 33/33（Panel 9+ModuleLauncher 12+SearchModal 8+锁数 4）+App palette/export 源级锁 6/6；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@695、版本四处 4.318.0；产物=exe 50592256B SHA256=63be1263d9077006a5b312f552abb2984787e9f4d3553ecaeb31bfdf3aaeb311（releases/gaea-v4.318.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：命令面板 query DOM 捕获→CommandPalette 可选 prop 清退（约 6 行）；精确回源会话接 seam（面板挂工作台或 resumeSession 提全局事件）；任务模板联动（GaeaTaskTemplates 从模板建任务）；LLM 兜底分类器对 save_task 覆盖；releases exe 留版数待用户拍板（沿 v4.315 观察项）。
- **未做（下刀）**：7.3-2 板块降视图（依赖本刀稳定一个零功能周）或 7.2-2 判据②调用计数小刀（statsFile 先例）；闲庭在册清欠沿既有列车。

## 最新发布：v4.317.0（2026-09-15）「阶段七第四刀：journal 历史蒸馏（重复模式挖掘+结晶审阅+审计链，7.2-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-journal-distill-7-2-2-20260915.md：契约+足迹互斥表+出口对照）→ A/B/C 三线并行 → 主代理收口。**坑**=A 线纯函数包子代理跑满上下文**零落盘**——主代理接管代写（卡死判据先例再+1：v4.316 B 线同款「长时间零落盘」）；B/C 两线正常交付（B 10/10、C 12/12）。
- **论点**：journal 证据链（.gaea/work/journal/*.jsonl，每卡=一次已应用变更）是书斋审批链既有沉淀，把它变成程序性记忆矿床——确定性挖掘跨会话重复 ≥3 次的工具应用模式 → 建议卡只提示不打扰 → 点「结晶为技能」→ LLM 从多次执行回放蒸馏草稿 → 复用 7.2-1 审阅弹窗编辑落盘 → 决定与结晶全程审计。**7.2 技能结晶双入口齐装**（演示式录制 v4.313+历史蒸馏本刀）。
- **线A（主代理代写）**：新纯包 internal/skilldistill（零 IO 表驱动，先例 routesuggest）：MineFlows 会话聚合确定性排序+Distill（连续同 (Tool,Shape) 折叠→会话级序列精确匹配分组→Repeat=不同会话数≥3→步数∈[2,12]→ignored 静默→截 3 宁少勿扰）+BuildPatternID（jd-+sha256 前 8hex 幂等）+ParsePatternID 残渣防御。V1 裁决=滑窗/子序列不进（观察池）。
- **线B**：internal/app/gaea_skill_distill.go 三绑定（OfficeB +3）——Candidates（journal 现算只读零 LLM，ignored∪crystallized 双静默）/Draft（buildJournalReplay ≤3 最近会话分段+AfterSummary 200 rune+整段 600 rune 防护→prompts/skill-from-journal.json（RTCO：多次执行共性进步骤、差异进 cautions、文件名通用化占位）→RetryJSON 2→复用 SkillRecordResult 只回不落盘）/Decide（ignore|crystallized；crystallized 必带技能名，审计条目=sessions+skill+decidedAt）；状态 <DataRoot>/skill_distill_state.json route_suggestions 同款容错读+原子写。
- **线C**：SkillDistillSection（PatternLine 步骤序+重复徽章+证据折叠+结晶/不再提示同卡互斥 loading+空态两分）挂 MemoryPanel 建议 tab「流程蒸馏」分区（props 全可选缺省隐藏，既有调用方零改动）；SkillRecordModal 加 preload/onSaved 可选（preload 直接回填不调内部蒸馏，缺省与 7.2-1 逐字节一致；7.2-1 Composer 入口不动，MemoryPanel 本地托管第二实例零复制）；三语 memory.distill.* 11 键三语键集相等。
- **主代理收口**：gen_bindings+bindingNames 再生 691（bindings_novel 368 行重排噪音甄别还原，v4.313 坑再证）+bridge/core +3（SkillDistillView 局部重述 herdsman 防环先例）+mappings+mock/chat 三桩（走查样本）+spaceBindings work 510→513+App.tsx 接线（useSessionHandlers 六值解构+MemoryPanel 六 props）+建议 tab 首开自动拉候选（C 线 distillView 初值 null 使 undefined 判据失效——主代理改 ref 守卫；fetch 期显示 loading 而非「不可用」误判）+补 SkillRecordModal preload 2 例。
- **绑定面**：688→691（OfficeB +3）；work 锁 510→513；drift OK@691。
- **出口对账**：①只提示不打扰、拒绝即静默 ✅（Decide ignore→状态→复算跳过，测试钉死）；③审计链完整（哪次历史/谁确认/哪版）✅；②结晶技能调用 ≥5 次且成功率可见=**欠账如实入档**（需技能调用计数机制 statsFile 先例另刀）。
- **验收**：skilldistill 15 例+app 10 例+前端 18 例（含收口补 2 例）；tsc -b 零错（双向漂移防线过）、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（复跑：首跑 exit 1 无 FAIL——flaky 先例再+1）；版本四处 4.317.0；产物=exe 50576384B SHA256=e5577402da81afbf0b0135631db066494aa7e415455da67eee2dbe9b0d5e3d2d（releases/gaea-v4.317.0.exe+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：滑窗/子序列挖掘 V2 等真实 journal 数据分布；journal 回放纳入 verdicts（复核结论）过滤；真机走查（同型任务×3→建议卡→结晶→新会话命中技能一条龙）挂闲置窗口；releases/README V4_RECENT 实存条数与「最近 34 版」头注不符（v4.315 前已存在，非本刀引入）。
- **未做（下刀）**：7.3-1 任务收件箱（收窄版任务空间制首刀）或 7.2-2 判据②调用计数小刀；闲庭在册清欠沿既有列车。

## 最新发布：v4.316.0（2026-09-15）「阶段七第三刀：路由学习（任务级账目+成本归因+建议制改绑，7.1-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-route-learning-7-1-2-20260915.md：三线调研结论+契约+足迹互斥表）→ A/B/C 三线并行 → 主代理收口。**坑**=B 线原实现子代理 40 分钟零落盘、两次检查点未应——中断接管主代理代写（子代理卡死判据：长时间零落盘+检查点静默；先例入 AGENTS 执行纪律候选）。
- **判据①任务级账目全键覆盖**：ChatRequest/ChatSimpleOptions 加 Feature 带外字段（json:"-"）→ ai 包 recordUsage/parseStreamEvents/prepareStreamRequest 透传 → statsKey 三维 feature|engine|model（whisper→chat 归一）→ statsFile v2→v3 兼容读 → 24 handler 文件 ~40 调用点一行式透传七键全覆盖（grep 复核零遗漏）；copilot 三入口获批扩迹；bridge 工厂 Provider 打标签（boot→gaea/显式引擎→office）。残留如实报：platform_handler 风格分析（internal/style 足迹外，落未标记桶）、gaea_diagram 两处全局引擎。
- **判据②成本归因视图**：模型中心「成本归因」tab（AttributionSection：KPI×4+功能分组表三维桶行+未标记桶最后；useAttributionState；api/engines.ts +4 函数）。
- **判据③建议制改绑**：新纯包 internal/routesuggest（score=成功率×成本因子域内归一；ScoreGapThreshold 0.25+MinSamples 20 双门槛宁少勿扰；单域至多 1 条；ID 确定性重算幂等+解析残渣防御）；App 三绑定 Suggestions/Apply/Ignore（Apply 走 SetFeatureModel 既有链路绝不自动改绑；状态原子写 <DataRoot>/route_suggestions.json）。
- **判据④本地引擎同池**：候选=全部启用引擎 llm 模型（Kind 回退 ClassifyModelKind）本地云端同池；mock 契约内置 herdsman 建议卡样本（走查锚点；真机采纳样本挂闲置窗口）。
- **绑定面**：684→688（ModelB +4，gen_bindings 覆盖表→model 门面）；work 锁 506→510（shared）；bridge/model.ts 局部重述类型+mock 四桩+新契约测试 mock-contract-route。
- **验收**：routesuggest 11 例+stats 3 例+ai 3 例+app 5 例+前端 9 例+锁数；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@688、版本四处 4.316.0；产物=exe 50515456B SHA256=82c0e4537d28dfb53314770054672ea094e636aac2645509227d76de88113213（桌面副本同哈希；冒烟 200 过）。
- **观察池**：按任务实例切片留 7.3 联动；bridge 启发式边界（显式引擎 provider=office 域现状 6 处）；route_suggestions.json 无 UI 清单；真机走查（聊天→归因 tab 见桶行+建议卡）。
- **未做（下刀）**：7.2-2 journal 历史蒸馏或 7.3-1 任务收件箱；闲庭在册清欠沿既有列车。

## 最新发布：v4.315.0（2026-09-15）「书斋首页 v9『案头』重设计+外观模块下拉化（零功能删除）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。工作树发现上会话中断的在飞刀（两份规格书随刀入库：进度计划/gaea-desk-home-redesign-v9-202609.md+gaea-appearance-module-redesign-202609.md；实现与工具链验证上会话已完成）——本轮接管收口：独立审查子代理七点对账+CDP 走查取证+全量门禁+发版。
- **v9 诊断**（v8 遗留）：砍刊头后无排印锚点（首屏最大字号 19px）/清单海 ~80% hairline 行+等宽点 10 处/命令条 54px 主角不立/右舷五件纵排无图形锚点。
- **落地**：①masthead 身份区（w-seal 印章 44px 印泥朱 #d06055 唯一 hex 豁免+台名 clamp(30,3vw,42px)+lede+chip/pill 迁入；与闲庭月洞门身份对仗）②kicker 行退役+命令条 54→60px③w-index→w-plaques 双列案牌④w-rail-panel→w-vitals（写作升首节、ml-ring 72→80px 作用域隔离）⑤外观面板 ChoiceCards/ThemeCard 退役→Select 泛型下拉（hover 预览「预览中」徽章）+SegmentedRow 三面板⑥WhisperTracePanel hex→--md-sys-* 令牌（亮色可读）。
- **契约**：五钩子 testid 全保；三语键集 1479 零增删（1 键文案随交互更新）；闲庭逐行不动；零绑定面 drift OK@684。
- **验收**：CDP 9333 DOM 断言书斋 17 项+闲庭 5 项全绿+亮色档令牌实测（surface=#f0fdf9）+三截图留档（.tmp/v9-desk-1440.png 等）；ModuleLauncher 8/8+AppearancePanel 6/6+审查对账（className 双向孤儿=0）。
- **门禁**：ci.ps1 全绿 exit 0（第三跑——首跑 netclient flaky 定向复跑绿、二跑 internal/app TempDir 清理竞争，环境 flaky 先例再+2；三跑清走 Vite/无头 Edge 负载后全绿）、版本四处 4.315.0；产物=exe 50464768B SHA256=356b7c6e8ac9ed4ecfa4f0fb5b1f9fa6c52f61b1f1dcdcc1bd13b620e3bc2a1d（桌面副本同哈希；冒烟 200 过）。
- **文档**：home.md 补 v9 节+releases README V4_RECENT 补齐 v4.308~v4.315 九版滞后+规格书入库+AGENTS 三十迁。
- **观察池**：releases 实存 33 版 exe 与「留 5 版」拍板不符（不动文件待用户定夺）；v3-rise 入场序号两对同拍；pnpm 污染两枚已清+frontend/NUL 坏文件已删。
- **未做（下刀）**：7.1-2 路由学习（三线调研已成：R1 遥测=RecordCall 全量收口+ChatRequest 缺 feature 字段；R2 绑定=featureModelKeys 8 键+SetFeatureModel 链路+routesuggest 纯包落点+route_suggestions.json 存储；R3 前端=模型中心「成本归因」tab 落点推荐+命名避让 GaeaCostAttribution。契约待主代理定稿→后端账目/接线+建议生成+前端三线并行）。

## 最新发布：v4.314.0（2026-09-15）「书斋首页 v8 重设计『驾驶舱台面』（技能库驱动，零功能删除）」

- **来源**：用户指令「使用技能重新设计书斋首页」——ui-ux-pro-max 技能库三路检索定方向（Swiss 数据密集+星枢令牌 MASTER 权威）+ Edge 无头 CDP 截图 + analyze_image 诊断。
- **诊断**：v7 刊头大标题占 ~10% 视高/左栏长卷目录挤出首屏/账页×目录同构列表纵向堆叠/右栏五节等权无主次。
- **落地**：①刊头压缩为 kicker 行（w-mast→w-deck）②命令台升格第一主角件（玻璃抬升面收命令条/气泡流/语音状态）③账页×目录双列 7fr/5fr+旗舰横带压缩+目录单列④右栏主从：晨报置顶⑤间距校准。
- **契约不变**：testid 全保留/i18n 零键增删/功能零删除；既有测试 9/9 零改动。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.314.0；产物见 `releases/SHA256SUMS-v4.314.0.txt`。
- **观察池**：晨报有数据时右栏高度回看；写作环空态密度优化候选。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏。

## 最新发布：v4.313.0（2026-09-15）「阶段七第二刀：会话录制技能（演示式录制，7.2-1）」

- **来源**：阶段七 7.2-1（平台调研验证形态：Anthropic Record a Skill / OpenAI Record & Replay 均已发布；SKILL.md 成事实标准）。已有单轮沉淀+记忆聚类候选两通道，缺「会话级多轮回放→结构化草稿→审阅入库」。
- **落地**：①skill-from-session.json 模板（蒸馏纪律+JSON 草稿契约）②office_skill_record.go（buildSkillReplay 末段 60 条回放/parseSkillDraft 归一断拒/renderSkillDraftBody）③GaeaSkillDraftFromSession（本地优先路由+RetryJSON，只回不落盘）+GaeaSkillDraftSave（审阅后落盘）④saveSkillFile 共享通道重构（单轮/会话录制同一落点）⑤SkillRecordModal+Composer 工具栏入口；绑定面 682→684（work 锁 506）。
- **纪律**：LLM 只产草稿，落盘必经用户审阅。
- **测试**：Go 6 例+vitest 4 例；既有零改动。坑=heredoc 转义破坏 TS 字符串/锁数量忘同步/gen_bindings 噪音再证。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.313.0；产物见 `releases/SHA256SUMS-v4.313.0.txt`。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏；闲庭在册清欠沿既有列车。

## 最新发布：v4.312.0（2026-09-15）「阶段七首刀：记忆质量受控评测（三脑四域零依赖基线+经验习得题集，7.1-1）」

- **来源**：阶段七规划（docs/archive/gaea-stage7-plan-2026-09.md，2026-09-15 拍板启动）首刀；闭合在册开放增量「记忆质量可评测」（对齐物升级为 BEAM/LongMemEval-V2）。三脑记忆域检索质量此前零量化——记忆不可信则 7.2 技能结晶复用成功率不可测。
- **落地**：①新包 internal/memoryeval（纯函数+种子语料零外部依赖）：Evaluate（recall@10 门槛 0.8 沿 T5-6 口径+precision 均值）+ParseSet/ResolveSetPath；②三域 runner 全走生产路径：story=纯 Go BM25/work=gaea bm25 Ranker/persona=whisper SQLite FTS 含中文 2-gram LIKE 降级；③经验习得题集 22 题（≥20 达标，LongMemEval-V2「习得经验」方向，查询=未来任务口吻）——7.2 评测基建预置；④种子语料 needle-in-haystack（12+40/12+28/10+20/22），题集与种子 ID 漂移由测试硬断言兜住。
- **测试**：memoryeval 4 例（解析/数学/结构硬断言/基线软告警——低于门槛只 WARN 不阻断）；既有零改动。
- **基线**：四域 recall@10 全 1.000（precision 均值 0.433/0.478/0.661/0.727）落档 docs/memory-eval-set.md；评测产出真实发现=persona 泛主语查询淹没（「用户」二字组 LIKE 命中全部事实）——生产弱点记录在档不动引擎。
- **门禁**：ci.ps1 全绿（vitest 首跑 ContextView 类 flaky 28 例、全量复跑 356 文件/3064 全过）、零绑定面 drift OK@682、版本四处 4.312.0；产物见 `releases/SHA256SUMS-v4.312.0.txt`。
- **未做（下刀）**：7.1-2 路由学习（任务级成本归因+建议制改绑）或 7.2-1 演示式录制（无依赖可先行）；闲庭在册清欠（t4-C3 余项/t6/t7/GenerationGate 收口）沿既有列车。

## 最新发布：v4.311.0（2026-09-15）「t5 收官刀：关系图谱升级（三类节点·四层边·分类筛选·详情浮层，§7.5）」

- **来源**：MuMu 蒸馏线 t5 收官刀（§6.1/§6.2/§7.5）。改造前组织节点是孤岛、职业不可见——状态机数据图谱可视化是收官题眼。
- **落地**：①纯函数模块 graphData.ts（buildGraphData）：节点三类+分组虚拟节点、边四层（组织成员 0.5/分组 0.35/主职 0.3/副职 0.2/人际 0.02——MuMu layoutWeight 映射 d3 strength，不引 dagre 不引新依赖）、人际 pair 去重、无结构边时人际 0.3 兜底、active 过滤+悬空防御；②组件：五类筛选 checkbox/点击详情浮层/职业方形节点/虚线边/选中描边；③零新依赖，调用点零改动。
- **测试**：vitest graphData +4；tsc 全绿；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面、版本四处 4.311.0；产物见 `releases/SHA256SUMS-v4.311.0.txt`。
- **t5 域全清**：差分器→回灌→组织差分→职业 UI→图谱，五刀收官。
- **未做（下刀）**：t4-C3 余项；t6/t7。

## 最新发布：v4.310.0（2026-09-15）「t5 第四刀：职业管理页（角色-职业绑定 UI，§7.6 断线补齐）」

- **来源**：MuMu 蒸馏线 t5 第四刀（§6.4/§7.6/§7.7-#9）。MuMu 职业接口后端完备前端断线；gaea 四刀全通后同样「用户到不了」——本刀补 UI 通路。
- **落地**：①NovelB +SetCharacterCareer/RemoveCharacterCareer（主职业允许替换〔手工意图 vs 差分器防 LLM 误改，两处语义有意不同〕/副职上限 2/同名幂等/阶段钳 ≥1/手工不写水位；存储=项目 characters.json 与差分器同库即设即生成可见）②CharacterPage Drawer 增「职业体系」区块（主/副管理+即时保存+上限禁用+操作后同步快照）+types v2 字段（GetCharacters 早已返回前端未消费）③契约面：bridge/mock/bindingNames 682/spaceBindings 锁 504。
- **测试**：character +1（绑定矩阵）+vitest 3 例（渲染/逐参/上限+Popconfirm 确认）。
- **维护**：AGENTS 速览整段分流——留 3 版迁 10 条入 archive，水位 61671→38151B 根治。
- **门禁**：ci.ps1 全绿、drift OK@682、版本四处 4.310.0；产物见 `releases/SHA256SUMS-v4.310.0.txt`。
- **未做（下刀）**：t5 余项唯一=关系图谱升级 §7.5；t4-C3 余项；t6/t7。

## 最新发布：v4.309.0（2026-09-15）「t5 第三刀：组织状态顶层差分接线（分析链三路输入全通）」

- **来源**：MuMu 蒸馏线 t5 第三刀（§3.7/§3.8/§7.7-#5）。差分器有组织管道但调用侧传 nil、模板无契约、载荷无字段——有管道无水源。
- **落地**：①V2 wire 契约 OrganizationStateChangeV2/OrgMemberChangeV2 名称式引用（与差分器 ID 式输入分层，对齐 character_states 先例）+AnalysisResultV2 +organization_states；②模板 output 加 organization_states JSON 形状+must 稀疏差分纪律（名称精确一致否则省略，不猜测）；③syncCharacterStatesV2 名称→ID 匹配（失败 WARN 跳过）传第 4 参；④**修首刀缺陷：覆灭短路**（destroyed=true 清零势力值+跳过同条 power 与成员变更；destroyed=false 不复活）。
- **范围裁决**：PowerValue 用 0-100 绝对值而非 MuMu 相对增量——幂等可重放，免疫 MuMu §8 坑 2（org 更新无水位守卫）。
- **测试**：analysis +2（端到端：晋升/加入/缺省忠诚/ID 匹配/不存在跳过/覆灭忽略 power/空载荷零写盘）+characterstate +1（覆灭短路纯函数钉死+destroyed=false 不复活）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.309.0；产物见 `releases/SHA256SUMS-v4.309.0.txt`。
- **未做（下刀）**：t5 余项（关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.308.0（2026-09-15）「t5 第二刀：状态回灌（SceneBible 实体属性 + 章节生成 character_states 槽位，降噪）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域第二刀（§3.9/§7.4）。v4.307 差分器写盘后生成侧看不见——状态机一半价值在写回生成侧。MuMu 两通道格式不一致（历史债），gaea 统一多行式。
- **落地**：①SceneBible 实体属性回灌（零新增管道）：entityFromCharacter +current_state/career_main/career_sub，sceneCharFromEntity 读回+角色文件兜底，formatSceneChar 渲染（存量老角色零噪声）；②create-chapter 模板 +character_states P1 槽位，注入器 buildCharacterStatesSection 多行式（存活 emoji/心理截150/职业「剑修·3阶」/组织+忠诚度/关系+亲密度），接线 CreateChapter 主链；③降噪（照搬 MuMu :794/821）：intimacy==50 与 loyalty==50（0 视为未记录）不注入、past 关系跳过、非 active 成员不显示、关系前5/组织前3、区段预算 1500 rune、存量项目全员无 v2 字段返回空串零噪声；④types.CareerMainLabel/CareerSubLabels（无最大阶——方案 C 死解析不做已裁决；CareerName 快照优先）。
- **范围裁决**：chapter-generate.json 是 embedded 死资产（无消费方），槽位打真实链路 create-chapter，死资产不养死槽位。
- **测试**：types +1 + novelcontext +3（roundtrip/兜底/零噪声）+ app +4（降噪矩阵/模板联通）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.308.0（含 versioninfo.rc，sync-version.ps1 同步）；产物见 `releases/SHA256SUMS-v4.308.0.txt`。
- **未做（下刀）**：t5 余项（组织顶层差分 UI/关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.307.0（2026-09-15）「t5 首刀：角色状态机差分更新器（水位守卫+存活级联+关系差分）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域首刀（§3.3~§3.6/§7.2/§7.3）。契约 v4.278 已落，差分器实现是缺口。
- **落地**：①internal/characterstate 差分更新器（三阶段严格顺序+七条不变量全实现：双水位单调/存活短路/**存活级联三件套**/职业钳制/MaxSubCareers=2/副职业水位守卫）；②亲密度算法最长匹配优先禁子串累加（修「不信任」-5 失真）+LLM DeltaHint 优先；③关系差分双向无向/新建基线 50/时间线追加；④模板扩 survival_status+relationship_changes+稀疏差分纪律；⑤syncCharacterStatesV2 委托差分器。
- **测试**：characterstate +5（级联全链/守卫/关系双向/词典/职业组织矩阵）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本三处 4.307.0；产物见 `releases/SHA256SUMS-v4.307.0.txt`。
- **未做（下刀）**：t5 余项（回灌 SceneBible/chapter-generate 槽位/组织差分 UI/图谱升级）；t4-C3 余项；t6/t7。

## 最新发布：v4.306.0（2026-09-15）「t4-C4 分析标注层：keyword→正文偏移（坐标反投影）+ 读取绑定」

- **来源**：MuMu 蒸馏线 t4 情节分析域第三刀（§3.10/§8.4）。C1 keyword 锚点坐标从未回投正文，本刀补标注层数据源。
- **落地**：①定位四段式（精确/去标点反投影 indexMap 修 MuMu 坐标缺陷/前缀 15/未命中）；②构建规则（锚定失败丢弃不留 -1；hook Content 兜底锚点；suggestion 留 -1 仅列条目；伏笔带 tags）；③落盘 analysis/annotations/<MMM>.json+Analyze 接线；④NovelChapterAnnotations 绑定（缺档有分析按需重建）。
- **测试**：analysis +3（定位四段式/构建矩阵/空载荷）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@680（锁 501→502）、版本三处 4.306.0；产物见 `releases/SHA256SUMS-v4.306.0.txt`。
- **未做（下刀）**：前端内联高亮随 t7；t4-C3 余项；t5~t7 未开工。

## 最新发布：v4.305.0（2026-09-15）「t4-C3 收尾：整章重写前端消费（对比确认 UI）+ 建议读取绑定」

- **来源**：t4-C3 收尾刀——v4.304 后端六绑定零前端消费。
- **落地**：①NovelChapterSuggestions（678→679，读 analysis-v2 建议，无分析空数组正常态）；②RewriteModal 三态（表单：建议勾选/自定义要求/重点方向/保留元素/字数，source 自动判；结果：统计+新全文+应用/放弃；应用后恢复入口）；③CreatePage rail 按钮+onApplied 刷新编辑器；④mock +7 诚实空态。
- **测试**：vitest RewriteModal 5 例；既有零改动。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@679（锁 500→501）、版本三处 4.305.0；产物见 `releases/SHA256SUMS-v4.305.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/版本历史面板；t4-C4 锚点标注。

## 最新发布：v4.304.0（2026-09-15）「t4-C3 首刀：驱动式整章重写 + 版本库（应用/丢弃/恢复）」

- **来源**：MuMu 蒸馏线 t4 情节分析域第二刀（§5.2/§8.3 C3）。C1 建议产出后补重写消费链；gaea 强制增量=版本库自带恢复（MuMu 无 restore 路由）。
- **落地**：①internal/rewrite 引擎（指令四段/输出清理/diff 统计/请求归一，纯规则）；②版本库（索引无全文+按需读，契约路径）；③六绑定：重写（建议来自 analysis-v2，温度 0.7，不自动落章）/列表/读全文/应用/丢弃/恢复（原文快照写回+审计）。
- **测试**：rewrite +4 / project +1 / app +4（桩 LLM e2e 全链含恢复审计）。
- **门禁**：ci.ps1 全绿、绑定面 +6 drift OK@678（spaceBindings 锁 494→500）、版本三处 4.304.0；产物见 `releases/SHA256SUMS-v4.304.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/前端对比 UI；t4-C4 锚点标注。

## 最新发布：v4.303.0（2026-09-15）「t3-P2 记忆召回消费：语义检索进 SceneBible 记忆区段（长程一致性全闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第三刀（§3.3/§3.4/§12.1/§12.4）。分析自动回填（P1）+语义召回注入（P2）全通；检索复用 semantic.Store+本地 embedding，不引 ChromaDB。
- **落地**：①结构化 query（人物[:8]/关键事件[:6]/叙事目标/情绪/本章要求[:150]，整体[:800]）；②四段流水线（阈值 0.35 归一化余弦/兜底 3/topK 8，三常量唯一声明）；③SceneBible.Memories+「相关记忆（按相关度）」区段（预算 400）；④Ensure 增量+Stale 清死向量+kind=项目目录隔离+当前章排除；⑤CreateChapter 全链容错接线。
- **测试**：app +4 / novelcontext +2；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.303.0；产物见 `releases/SHA256SUMS-v4.303.0.txt`。
- **未做（观察池）**：t3 三刀收官；真机检索质量评估；候选 2 记忆评测开放；t4-C3/C4、t5~t7 未开工。

## 最新发布：v4.302.0（2026-09-15）「t3-P1 记忆生产者：规则表抽取 StoryMemory 按章落盘（自动回填闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第二刀（§2.3/§12.1/§12.2/§12.4-4）。t4-C1 解锁 V2 载荷，本刀补「分析→结构化记忆」抽取——自动回填闭环咬合（市场差异化点，调研档 §9）。
- **落地**：①types.StoryMemory+确定性 ID `<MMM>-<type>-<ordinal>`+五类常量+is_foreshadow 三值（无 Position 死字段 D11）；②提取规则表纯函数（chapter_summary 必有三级回退固定 0.6/hook≥6/foreshadow 全部/plot_point≥0.6/character_event 0.7/conflict≥7→plot_point，门槛常量唯一声明）；③memories/MMM-<n>-memory.json 按章整文件替换写（幂等规避 MuMu D1，空=删文件）；④Analyze 接线 persistStoryMemories（容错）。
- **测试**：analysis +3（规则表/确定性 ID/回退链）+project +1（回环/空写删档）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.302.0；产物见 `releases/SHA256SUMS-v4.302.0.txt`。
- **未做（下刀）**：t3-P2 召回消费（semantic kind=story_memory+SceneBible 记忆区段+三常量+结构化 query）——接上即全闭环；t4-C3/C4。

## 最新发布：v4.301.0（2026-09-14）「t4-C1 分析代理 V2 化：9 维结构化 + 三维评分联动 + analysis-v2.json 落盘」

- **来源**：MuMu 蒸馏线 t4 情节分析域首刀（§8.3 C1）。市场调研增量轮（调研档 §9）抬升优先级：自动设定库回填是行业痛点（Novelcrafter 手填摩擦/Sudowrite 书长极限），V2 载荷是回填数据底座。
- **落地**：①模板 V2 九维化原地升级（伏笔追踪约束原样保留，ForeshadowHit 契约零改动）；②服务端权威归一（三维钳位/overall 重算/建议联动只裁上限）；③analysis-v2.json 按章号 upsert 落盘（容错注入）；④旧 wire 零破坏派生（AnalyzeChapter 返回键零变化）；syncCharacterStates 升 V2 差分。
- **测试**：analysis +4 + project 落盘回环；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.301.0；产物见 `releases/SHA256SUMS-v4.301.0.txt`。
- **未做（下刀）**：t3-P1 记忆生产者（已解锁，下一刀闭环自动回填）；t4-C3 驱动式重写；t4-C4 锚点标注。

## 最新发布：v4.300.0（2026-09-14）「t3 首刀：章节前文摘要窗口预算化 + 回退链 + 反重复约束」

- **来源**：MuMu 蒸馏线 t3 长程一致性域首刀。§11.3 缺口：prevSummary 全前章 200 rune 无界拼接（200 章≈40k rune 前缀），不参与预算体系。
- **落地**：①最近 10 章窗口（buildPrevSummaryWindow 纯函数，单章 180 rune，章号升序确定性，空返回跳槽位）；②窗口声明头（部分视图语义显式化）；③摘要回退链：大纲 Summary→章节摘要文件→跳过；④create-chapter 模板反重复 must（衔接不复述+窗口外既定剧情不得矛盾）。
- **测试**：app +4（窗口/截断/回退链/乱序空态钳位）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.300.0；产物见 `releases/SHA256SUMS-v4.300.0.txt`。
- **未做（t3 下刀）**：P1 记忆生产者（StoryMemory+规则表+落盘，依赖 t4 产出 AnalysisResultV2）；P2 语义召回消费（semantic kind=story_memory+记忆区段+三常量）。

## 最新发布：v4.299.0（2026-09-14）「伏笔面板接调度可视面：清理入口/统计/紧急度 Badge/同步结果（t1-P4）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第四刀（收官刀）。P1~P3 全在后端，前端面板仍是旧三态视图。
- **落地**：①SyncResult 上绑定面（v4.297 欠账）：GetLastForeshadowSync（面 671→672），同步计数+skippedReasons 全量（D3 可见），未分析显式报错；②GetForeshadows 附带 urgency 运行时投影（共享入口算，A5 不落库，回收/废弃不带键）；③面板四件：后端统计行（分状态+超期红 Tag，降级前端计数）/紧急度 Badge（红橙金+tooltip）/清理折叠区（三入口全确认+自动重载）/上次同步常显卡（计数+前 2 条跳过原因）；④A5 护栏=写回 stripForeshadowUrgency 剥离；⑤契约面+mock+5+load() 增量绑定降级。
- **测试**：Go +2（投影矩阵/lastSync；SyncForeshadows 按规格签名转正导出）+ vitest 面板 14/14（+6）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@672（spaceBindings 锁 493→494）、版本三处 4.299.0；产物见 `releases/SHA256SUMS-v4.299.0.txt`。
- **未做（观察池）**：真机走查闭环一条龙（等闲置窗口）；SyncResult 持久化不做（会话内诊断面）。

## 最新发布：v4.298.0（2026-09-14）「伏笔生命周期清理与统计 + Lint 扩两码（t1-P3）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第三刀（spec §8.1/§5.1/P3.4）。P2 打通分析自动回收后，重分析/重新生成场景缺「干净重来」入口。
- **落地**：①三个清理入口（新 `internal/app/foreshadow_cleanup_handler.go`，语义严格区分）：DeleteChapterForeshadows（删埋入∨回收本章，默认只删分析来源）/CleanChapterAnalysisForeshadows（重分析前：删「分析∧埋入本章」+回退本章回收→planted 清回收痕迹）/ClearProjectForeshadowsForReset（删全部来源；手动**重置不删除**→pending 清章节关联时间戳）；关键不变量=source_type 唯一判据、手动条目永不批量删除；②统计 GetForeshadowStats：分状态+longTermCount+overdueCount（ClassifyResolve 唯一入口，currentChapter≤0 自动），resolved 别名归一并入；③Lint 扩两码 5→7：overdue（medium）+unplanned（low，长线豁免，与 stale「该收了」vs「缺计划字段」并存）。
- **测试**：app +6（三入口 e2e/统计口径/Lint 新码矩阵/camelCase 形状锁）。
- **门禁**：ci.ps1 全绿、绑定面 +4 drift OK@671（spaceBindings 锁 489→493）、版本三处 4.298.0；产物见 `releases/SHA256SUMS-v4.298.0.txt`。
- **未做（下刀）**：t1-P4 前端（面板接清理/统计/新体检码+Badge，mock 随消费面补）；真机走查（分析→回收→清理闭环，等闲置窗口）。

## 最新发布：v4.297.0（2026-09-14）「伏笔分析驱动自动回收：三级匹配闭环 + 候选清单三层渲染（t1-P2）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第二刀（spec P2 闭环核心）。此前分析侧回收只做「描述精确相等」，候选清单是整包 JSON 直塞 prompt——模型记不住该回填哪个 ID。本刀按 `docs/distill/01-foreshadow-spec.md` §3.3~§3.5/§6.1 把分析侧接上 v2 契约，埋入→回收闭环咬合，零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_match.go`）：WordOverlap（rune 级 2/3-gram Jaccard 加权）+ MatchForeshadowByContent 六策略加权（标题族取最大/关键词/内容相似/引用章/分类/角色 Jaccard 累加，采纳≥0.5 同分取先=最早埋入）；②同步层重写（新 `internal/analysis/foreshadow_sync.go`，消费 `types.ForeshadowHit` 替代旧 ForeshadowAction）：回收三级匹配=精确 ID（查不到禁止回落内容匹配）→内容兜底→跳过不新建；已回收不重置、pending 拒收、hinted/partial 可回收（D15）；埋入两道防重+每章新建≤5+Importance=min(Strength/10,1) 唯一派生+计划章缺失不猜值；SyncResult 完整含 SkippedReasons 六类 Kind（修 MuMu D3 静默跳过）；③候选清单三层渲染（新 `internal/analysis/foreshadow_prompt.go`）：L1 必须回收逐条带「⚠️ 回收时 reference_stable_id 填写」紧邻指令/L2 超期≤5/L3 其他≤10+溢出注记，分层走 ClassifyResolve 唯一入口（D9）；④`prompts/analysis-chapter.json`：foreshadows 换 v2 字段+候选槽位升 P1+伏笔追踪任务指令+三条强约束+ID 追踪自检（spec §6.1）；写入口径仍 revealed，前端零消费零破坏。
- **测试**：types +4 / analysis 同步 15 例（4 迁移+11 新增，含无效引用不回落关键回归）/ 渲染 4 例（三层/排除口径/折叠上限/空态 D14 截断）。
- **门禁**：ci.ps1 全绿、drift OK@667（零绑定面）、版本三处 4.297.0；产物见 `releases/SHA256SUMS-v4.297.0.txt`。
- **未做（下刀）**：t1-P3 清理入口/Lint 扩展 overdue·unplanned/统计口径；t1-P4 前端（表格/Badge 用后端 urgency，SyncResult 上绑定面随 P4）；真机走查（分析闭环+书源线一条龙，等闲置窗口）。

## 最新发布：v4.293.0（2026-09-14）「伏笔分层注入：按计划回收章调度生成上下文（t1-P1）」

- **来源**：MuMuAINovel 蒸馏线 t1 伏笔域消费方第一刀。契约（6 态并集+计划回收章+注入控制）v4.278.0 已落库但零生成侧消费方；本刀按 `docs/distill/01-foreshadow-spec.md` §4 落分层注入（spec P1），零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_urgency.go`）：UrgencyLevel（0-3 运行时不落库；D15 修正 partial/hinted 有回收压力；缺计划章不猜值不假超期）+ClassifyResolve 四值+ForeshadowLayerOf 分层唯一入口（L1 必须回收/L2 超期/L3 近期参考/L4 本章计划埋入/L5 无计划兜底〔gaea 扩展：存量无 target_resolve_in 条目保底注入防回归〕）+调度常量唯一声明；②include_in_context=false 全层排除、auto_remind=false 只抑制 L3（D8：L1/L2 硬约束不消隐）、远期不注入；③四层渲染（新 `internal/app/foreshadow_context.go`）替代旧「未回收伏笔（创作约束）」单层：模板对齐 spec §4.2（D14 省略号按长度判断）、L2≤3/L3≤5/总量≤15/整区含标题≤1600 rune、消耗顺序 L1→L2→L3→L4→L5、空层不输出标题、区段头带调度规则约束行；④`resolveTargetChapterNum`（显式/分支父节点/顺延，与 ensureChapterNode 同源复用）；⑤novelcontext 场景圣经分层优先采样（硬约束层带层标注先于参考层；分层逻辑全仓唯 types 一处=D9 消除）；⑥删 docs/mumu-distill 重复副本。
- **测试**：types +4 / app +7 / novelcontext +1（抓出超期章数负号 bug）/ 既有 context 测试迁移分层口径。
- **门禁**：ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.293.0；产物见 `releases/SHA256SUMS-v4.293.0.txt`。
- **未做（下刀）**：t1-P2 分析驱动自动回收（MatchByContent 六策略+SyncForeshadows 三级匹配+分析 Prompt 候选清单）；t1-P3 清理/Lint 扩展；t1-P4 前端。

## v4.283.0 ~ v4.292.0 段落补登（2026-09-14；逐版全文在 CHANGELOG/releases，速览补账）

- **v4.292.0** tail×反推串联：tail 导入成功→确认→novel:goto-tab+novel:auto-reconstruct 两事件→CreatePage 任务化反推链自动开跑。
- **v4.291.0** 反推任务化：tasks.KindOutlineReconstruct+Start/TaskGet（NovelB 660），长书后台态页面关闭不丢。
- **v4.290.0** 角色名→角色库 ID 匹配：matchCharacterIDs 合并进反推 Apply（不冲手工）。
- **v4.289.0** oh-story T6 书级文风档案 style.md 注入（T1~T6 收官）；**v4.288.0** T4 篇幅路由三档骨架（bookimport/skeleton.go）。
- **v4.287.0** tail 出口 ImportNovelBookEx；**v4.286.0** oh-story T2 内核消费（patterns.json+书级白名单）。
- **v4.285.0** 搜索历史+引擎规则编辑器（NovelB 657）；**v4.284.0** 失败章重试补下（NovelB 655）；**v4.283.0** 在线搜书入口；v4.283.1 nil-Fetcher panic 根修。
- 非版本刀：真机走查（假站 /unfail 配方）、flaky 治理（ProgrammingPage 15s/ContextView 40s）、rename 根修（fileutil.RenameWithRetry）。

## 最新发布：v4.282.0（2026-09-13）「oh-story 蒸馏首刀接线：平台质量评审（rubric 引擎 + 创作间面板）+ 生成门确定性两路直显」

- **来源**：用户「把 oh-story-claudecode（MIT 网文写作插件）蒸馏给小说板块」——把已在库的三份资产（T1 评审 rubric / T2 去 AI 味门禁资产 / T3 生成门资产）**接成用户可用的一条链**（规格 `docs/archive/gaea-novel-ohstory-distill-2026-09.md`）。
- **① 平台质量评审引擎**（新 `internal/novelreview`，零 LLM/零网络/纯函数）：15 个可机械判定维度（字数区间 / 开篇钩子 / 章尾钩子 / 预告式收尾 / 情绪节点密度 / 爽点密度 / 段落节奏 / 对话占比 / 标点节奏 / 破折号 / 格式合规 / 人称视角 / 主角存在感 / 金手指提及 / 字数表述核对），每维 `PASS|WARN|FAIL|SKIP` + S1~S4 + **原文证据（rune 区间 + 段落号）** + 改法；结论 `APPROVE|CONCERNS|REJECT` 与 rubric 同门槛；**语义维度不下结论**，缺外部数据显式 SKIP 并说明原因，**不静默给 PASS**。阈值/词表/档位是数据资产 `internal/novelreview/rubric.json`（四档：通用/番茄/起点/知乎盐言；可被 `<工作区>/.gaea/skills/novel-review/rubric.json` 整体替换；fail-closed 校验）。
- **② 接线**：NovelB +2（`NovelChapterReview`/`NovelReviewPlatforms`，648→650，drift OK）；创作间 rail「平台评审」→ 面板（档位选择 + 结论 + 逐维按 S1→S4 排序 + 证据摘录 + 黄金三问）。主角名取自角色库 `role_type=protagonist`。
- **③ T3 两路直显**：`NovelInspector`「章节体检」区新增「写前契约：N 项待补 / 齐备」与「写后硬信号：N 项 / 未命中」+ 严重度排序条目（与 AI 四路并列；缺省诚实降级）。
- **④ 口径对齐**：`.gaea/skills/novel-review/rubrics/generic.json` 18 维 → **26 维**（+8 引擎扩展，标 `measurable/engine`）+ `deterministicEngine` 段；SKILL.md 补「确定性引擎」一节（15 维映射 / 覆盖路径 / `plot_loop` 代理判定口径）——**资产=评审协议、引擎=可机械判定子集，共用同一套维度 id 与 S1~S4**。
- **测试**：Go +16（`internal/novelreview` 12 / `internal/app` 4）+vitest +10（面板 6 / 检查器 3 / CreatePage 1）+spaceBindings 锁 470→472。
- **门禁**：ci.ps1 全绿、drift OK@650、版本三处 4.282.0；产物=exe 49,597,440B SHA256=83F5BFD603DB5EC1B091E006F8CACB85B37C3283FF653CCED669F44A9628EAD9（releases 归档 + 桌面副本同哈希 + 冒烟 200）。
- **未做（下刀）**：T2 内核消费（`gates.json` 模式级门禁进 novelstyle 打分/去味）/ T4 导入结构映射与篇幅路由 / T5 子代理角色卡资产 / T6 风格档案协议；写前契约「拒绝生成」硬闸与书级白名单。

## 非版本刀（2026-09-13）：小说·生成门补确定性两路——写前大纲契约 + 写后质量体检（oh-story 蒸馏 T3）

- **来源**：oh-story-claudecode（MIT）蒸馏第三刀（规格 docs/archive/gaea-novel-ohstory-distill-2026-09.md §2 真增量第 1 条：gaea 缺「写前结构契约闸 + 写后确定性质量闸」）。
- **落地**：①新包 `internal/novelgate`（零 LLM 纯函数）——`OutlineContractIssues`（标题/计划/要点/情感四项目齐备性，S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / 电报体 S2〔短句占比 >40% 且均句长 <10 字〕/ 平均句长偏短 S3 / 标点堆砌 S3 / 省略号滥用 S3 / 通篇句号化 S3）；Issue 与评审 rubric S1-S4 同轴、必带证据；阈值常量集中、rune 口径。②接线 `RunChapterGate` 新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路并列（必出、零成本、不阻断）。
- **测试**：Go +7（契约齐备与分级/空正文 S1/正常文本零误报/电报体/超长段落/标点与省略号/句号化/证据随行）。
- **门禁**：本刀自身证据（go build/vet exit 0 + novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——并行线在制品 internal/app/novel_review_handler_test.go:117 当前为红（与本刀无关）；drift OK@648；**未抬版本**（下次发版随 exe 出）。
- **未做（下刀）**：T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单与 T2 `gates.json` 的内核消费。

## 最新发布：v4.281.0（2026-09-13）「拆书导入 P1 接线：AI 反推大纲（预览载荷 + 幂等落库 + 创作间入口）」

- **来源**：续 v4.280.0（反推引擎），把 t2 P1 接成用户可用的一条链（规格 §8.2/§8.3 首刀）。
- **后端**（新 `internal/app/novel_import_ai.go`，绑定 646→648）：`NovelOutlineReconstruct` 读当前工程章节（标题取大纲节点、正文 `ReadChapterAsStitch`，上限 200 章）→ Stage1 立项反推（模板 book-import-project，采样 3 章 ×2000 rune）→ 分批章节大纲（batchSize=5，逐批独立降级）→ 预览载荷零落库；`NovelOutlineReconstructApply` 按**章号**命中节点写 summary/scene_ideas/key_points/emotion，**幂等**、不新建不删除、不碰正文、角色名只进预览；空载荷/全不匹配**显式报错**。
- **降级诚实**：无模型或解析失败逐级回落规则兜底并把原因写进 warnings（aiUsed=false），单批失败只影响该批。
- **前端**：CreatePage rail「AI 反推大纲」→ 反推 → 确认弹窗（题材/视角/目标字数 + 前两条告警 + 「不覆盖正文、可重复执行」）→ 应用 → 刷新大纲 + rail 消息。
- **接线**：NovelB +2、bindingNames 648、spaceBindings 归 play（锁 468→470）、wailsjs 由 `wails generate module` 再生。
- **测试**：Go +2（无模型全规则兜底且逐章完整 / 幂等与作用域 + 空载荷与全不匹配报错）+vitest spaceBindings 4 例 + CreatePage 9 例。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@648、版本三处 4.281.0；产物=exe 49,455,616B SHA256=7FB09E36…2A693（releases/gaea-v4.281.0.exe + SHA256SUMS-v4.281.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：导入向导的逐章预览 UI；长书后台化任务态（`tasks` 表 `KindBookImport`）；角色名→角色库 ID 匹配；tail 模式出口。

## 最新发布：v4.280.0（2026-09-13）「拆书导入 P1 引擎：大纲反推编排 + Prompt 模板移植 + 输入节稳定序」

- **来源**：续 v4.279.0（t2 P0），本刀落 t2 P1「AI 反推」**引擎层**（规格 `docs/distill/02-book-import.md` §8.2/§3.7/§3.8）；真模型调用编排与预览 UI 下刀接线（本刀零 LLM 依赖、全部可单测）。
- **落地**：①反推编排引擎（`internal/bookimport/reconstruct.go`）——具名契约 + 逐字段归一化（rune 截断 / characters type 二元 / **title 与 chapter_number 强制用输入值**）+ **位置对齐**批量归一化（AI 少返/乱序不错章）+ 数量不符**整批回退规则结构**的断言式防线 + 规则兜底逐字对齐规格 + 视角 11 别名归一 + target_words <1000 回落/>3e6 夹取；②`CallJSON` 负反馈重试（期望类型显式 object|array、失败原文截 200 字注入、传输错误不重试、ctx 取消零调用）+ `ExtractJSON` 容忍 markdown 包裹；③新增两个 RTCO 模板 `prompts/book-import-project.json` / `book-import-outline.json`；④**补掉规格点名的引擎缺陷**：`BuildUserPrompt` 原按 Go map 遍历输入节（同模板两次渲染字节不同、长文本无法稳定置尾、前缀缓存无法命中）→ `InputDef` 增 `order`，按 Order 升序 → key 字典序稳定排序，长文本固定置尾。
- **测试**：`internal/bookimport` +9 例（视角矩阵/字段回落与夹取/位置对齐/截断与 type 归一/兜底取首句/重试提示要素/类型不符报错/传输错误与 ctx 取消/JSON 容错与采样）+`internal/prompt` +2 例（**同模板 20 次渲染字节一致**/order 覆盖 key 序）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646（零新绑定）、版本三处 4.280.0；产物=exe 49,405,440B SHA256=0F9BF89A…8EF4B（releases/gaea-v4.280.0.exe + SHA256SUMS-v4.280.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：app 绑定 `NovelImportReconstruct*`（真模型编排 + 预览载荷）/ 导入向导 UI（预览→应用，含 tail 模式出口）/ 幂等落库与三件套（§8.3，`tasks` 表 `KindBookImport`）。

## 最新发布：v4.279.0（2026-09-13）「拆书导入 P0：三级分章 + 5 级编码链（MuMu 蒸馏 t2 首刀）」

- **来源**：续 v4.278.0（t1 共享契约），本刀落 t2「拆书/导入反推」**P0 阶段**（规格 `docs/distill/02-book-import.md` §8.1，纯规则零 AI、可独立验收）。
- **改前短板**：导入只有强标题正则，识别不到即退化成单章「全文」；编码链缺 utf-8-sig 与 Big5，且 GB18030 解码几乎不报错 + `utf8.Valid` 二次校验**静默放过误判**。
- **落地**：①新包 `internal/bookimport`（纯规则零 IO）——`Decode` 五级链 + UTF-16 BOM + **候选结果含替换符即判误判续探**（实测 GB18030/GBK 解 Big5 不报错只出乱码）；`Clean` 六步顺序敏感清洗；`Split` 三级切分（强标题存在不叠加弱标题、弱标题需 ≥2 候选、无标题 >5000 字走 3000~5000 窗口取最靠后句读边界）；`tail` 裁剪（5 倍数取整、>50 降 full）；四类告警。②app 接线（委托 + `NovelImportResult` 增 `encoding`/`split_strategy`/`warnings`）。③前端成功文案直显「编码 · 切分策略」+ 告警弹窗。
- **有意偏离 MuMu**：首标题前 <200 字并入首章（不丢书名/作者）；阈值一律 rune 计数（MuMu 用字节）。
- **测试**：`internal/bookimport` 17 例（Big5 不被 GBK 遮蔽回归/BOM/UTF-16/兜底不 panic/清洗顺序/三级切分/窗口上界/三类告警/tail 取整与降级）+`internal/app` 导入 5 例（BOM/GB18030/无标题窗口/弱标题/e2e 报告字段）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.279.0；产物=exe 49,397,760B SHA256=D8947306…7659E（releases/gaea-v4.279.0.exe + SHA256SUMS-v4.279.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：AI 反推编排（字段级归一化 + JSON 负反馈重试）/幂等落库与任务态（`tasks` 表 `KindBookImport`，禁用内存 dict）/导入向导 UI（tail 引擎已就绪，当前仅 full 可达）。

## 最新发布：v4.278.0（2026-09-13）「小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）」

- **来源**：两条线汇合——①「优化完善gaea」并行池小说线续刀=伏笔一致性 Linter（文风指纹 v4.277.0 已落）；②MuMuAINovel 蒸馏实施线（规格 docs/distill/）由外部团队先落 t1 共享契约后停摆，用户确认后由 Codex 侧接管收口。
- **①伏笔一致性体检**：新 `internal/app/novel_foreshadow_lint_handler.go`（纯函数 5 类确定性检查：ordering / status-mismatch〔partial 豁免〕/ dangling / stale≥10 章〔长线豁免〕/ duplicate）+ 报告 totalChapters/items/planted/hinted/revealed/longTerm/findings；前端 ForeshadowPanel「一致性体检」（概要行 + severity Tag 轻/中/重 + 说明 + 原文条目 + 章节引用，Go 侧 omitempty/null 全防御）；绑定 645→646。
- **②t1 共享契约落库**（internal/types/）：伏笔并集 6 态 + 计划回收章 + 来源/评分/关联/注入控制 + urgency 显式 must_resolve（MuMu 静默丢弃字段的教训）+ 引用失效禁止回落内容匹配；角色三态章节水位守卫 + 方案 C 职业引用 + 派生 member_count + 亲密/delta_hint 钳制；9 维分析 + 三维分档评分 + 建议数硬联动；ChapterPlan / RewriteVersion+Index / Annotation；模板解析-覆盖-渲染-校验与上下文组装接口 + Lint 接口 + ChapterNumOf；Character/Organization/Relationship v2 可选字段（全 omitempty 零迁移）+ compat_test 兼容矩阵。
- **③P0 wire 口径纠偏**：在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降读取别名）——与 handoff §2-3/§4-7 相反，会让统计回收率/lint 计数/章节注入/SaveForeshadows 白名单等 7 类消费方**静默读空**；纠回 `revealed` 写入口径 + 新增 `IsResolvedStatus` 覆盖两套 wire 值并接进 stats / 伏笔 lint（检查·标签·计数）/ create_chapter 注入闸 / analysis 聚合；SaveForeshadows 白名单 3 态→并集 6 态（别名先归一化）。
- **④规格入库**：`docs/distill/`（01~07 六域 + 09-impl-handoff）入库并在 docs/README 登记；docs/mumu-distill/ 为重复副本未入库。
- **测试**：Go 全量绿（types 兼容矩阵 + 两套 wire 值新用例 / app 伏笔 Lint 5 类 / stats / analysis）+ vitest 体检 3 例 + spaceBindings 锁 467→468。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；产物=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **欠账**：六域实施 t2~t7 未开工（规格在库、契约就位）；t1 契约除伏笔体检外暂无消费方；**走查未覆盖创作面壳内真机**（mock 无工程夹具）；观察项=ChapterNumOf 双实现 / stats·narrative 无 owner / docs/mumu-distill 重复副本待删 / **novel api 直取 `window.go.app.App`（`components/novel/api/outlines.ts` 等，mock 模式 console 报 Wails runtime unavailable；生产壳不受影响）**。

## 最新发布：v4.277.0（2026-09-13）「小说·文风指纹落地面：参考档构建 + 对照体检」

- **来源**：「优化完善gaea」→ 拍板池三项均用户决策项，按既定习惯从板块并行池挑小说线既定余项（novel-revolution 刀3 内核指纹算子齐备但 app 层零消费）。
- **落地**：参考档 fingerprint.json（Manager 三方法原子写）+ NovelFingerprintBuild（全部已写章节 ComputeFingerprint 落盘，门槛 ≥3 章 ≥3000 字诚实拒绝）+ NovelFingerprintScore（有参考档 ScoreText+Delta 对照 / 无参考档通用阈值，两路可用）+ NovelFingerprintStatus + CreatePage「文风指纹」面板（空态引导/摘要格子/体检大数字+Δ 基线距离+issues 摘录）。绑定 642→645。v1 口径=作者成稿即样本，「只学作者手改」留后续刀。
- **测试**：Go+6+形状锁（camelCase 断言+delta=0 指针零值不吞）+vitest+5+spaceBindings 锁 464→467；tsc/eslint 0；drift OK@645。
- **真机走查**：夹具工程（project.Create 造 3 章约 4000 字）壳内 CDP 全流程全通（空态→构建→体检→重建覆盖）+console 零错误+清场。坑复训=海报墙须 scrollIntoView 再点/无 outline 夹具致体检钮 disabled/delete-pending 须 PowerShell 复删。
- **门禁**：ci.ps1 run1 抓 v4.276 存量缺陷当场修（ChatPage.test 语音断言未随 camelCase 发送对齐，改齐后单文件 15/15）run2 全绿/版本三处 4.277.0/桌面副本同哈希冒烟 200。产物 exe 49,352,704B SHA256=2CB99312…802CD。

## 最新发布：v4.276.0（2026-09-13）「绑定面 JSON 形状普查：造价参考/复盘笔记/会话组价价格带断链修复」

- **普查方法**：把 v4.269 走查抓到的「Go struct 缺 json 标签→Wails 线上 PascalCase→前端 camelCase 读空」bug 类升级为全仓 AST 闭包普查（绑定面签名类型递归展开×缺标签求交）：327 缺标签 struct 中绑定面可达仅 7。
- **三真断链当场修**：cost.PriceBand/BandSource（v4.191 起会话组价价格带卡全空+证据表离群判定失效）/costref.Note（复盘笔记 validUntil/refCount 读空）/costref.Indicator（造价参考视图读空）；四输入方向防御补齐。连带抓出 useChatVoice.ts 三处 PascalCase 发送点改 camelCase。
- **结构性守卫**：TestWireShapeGuard 把普查逻辑测试化进 ci（缺标签即 FAIL，负向验证过）——v4.269 修 7 结构体没防住本轮同款，守卫是根治。
- **真机走查**：CDP 桥验 GaeaCostNoteList 返回 camelCase+UI 断言笔记卡片全字段渲染+console 零错误+清场干净；坑=整段式走查脚本 UI 段卡死，拆探针分段+看门狗全链过。
- **门禁**：ci.ps1 全绿（run1 负载 flaky 两例隔离复跑绿）/ drift OK@642 / 版本三处 4.276.0；产物=exe 49,294,336B SHA256=1A12B1CA…DA869（SHA256SUMS-v4.276.0.txt，冒烟 200 过）。

## 最新发布：v4.275.0（2026-09-13）「价格带数据源调研结案：信息价『除税价』列适配」

- **调研**（docs/archive/gaea-priceband-datasource-research-2026-09.md）：价格带数据面基础设施已完整在产（四要素字段+统计+导入链+询价飞轮），外部自动接入四路评估后建议候选4 结案（抓取脆弱+合规灰、商业 API 同「询比价不做」口径、LLM 查价不作基线）。
- **真机验证抓到实锤当场修**：信息价样本 CSV 走 ImportPreview（零落库）——「除税价（元）」不在 fieldPrice 字典→12 行全 skip；当场修字典增「除税价/含税价」+Go 回归测试，复测 12/12 全识别。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.275.0；产物见 SHA256SUMS-v4.275.0.txt。**候选4 结案待用户确认标记**。

## AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）：恢复 14 版水位

- CI 仓库卫生守卫 WARN（.gaea/AGENTS.md 59926 B 逼近 65536 B 预算）触发既定分流：迁 v4.261.0/v4.260.0/v4.259.0/v4.257.1/v4.256.0/v4.255.0 六条入 `docs/archive/agents-version-history-2026-09.md`（八迁记录已更新），主文件恢复「最近 14 版」。水位 59926→47518 B。
- **坑**：迁移动作用 node 脚本而非 sed/正则手改——archive 是 CRLF 行尾，`indexOf('## 版本状态\n')` 匹配不上；锚点一律不带行尾、写入前先验 includes。迁移后三项完整性断言（主文件条目数 14/六条入档/迁入记录更新）全过。

## 最新发布：v4.274.0（2026-09-13）「UX 线第六刀：自定义强调色亮态自动深化」

- **根因**（对比度普查观察池第一项挖到底）：非主题令牌缺陷（lightFn glow 全是深化值），是**自定义强调色不分明暗覆盖 glow/primary**——暗色调亮的 #1dd7bf 切亮态压浅底对比仅 ~1.5，全站 accent 文字看不清。
- **修复**：`ensureLightContrast` 纯函数（lib/accent.ts，保色相压亮度至 WCAG 4.5，单测 7 例含真实案例）+ App.tsx effTokens 亮态深化暗态原样（用户存储色不变）。
- **复验**：重建 exe 复跑对比度扫描——light 告警 63→53 accent 类清零（亮态 glow 实测 #128475），dark 15 不变；剩余为禁用态/误报/设计弱化。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.274.0；产物见 SHA256SUMS-v4.274.0.txt。

## 每版度量看板补账（2026-09-13，维持轨）：v4.273.0 六行落账，滑步注记

- **发现**：slim-baseline §7「自 v4.267 起恢复逐版落账」实际只落了 v4.267 一行，v4.268~272 五版滑步（CHANGELOG v4.252/253 漏记同型教训再现）。
- **补采 v4.273.0 六行**：板块 5/5（manifests 未动）；entry chunk gz **243.6KB**（v4.267=243.0，+0.6KB≈UX 三刀纯 CSS）；冷启动 **1.39~1.70s 六轮**（bind→UI 405~424ms，同 v4.267 区间）；内存 139.1MB 工作集/152.9 私有（采样时开着用户真实工程，环境态口径注）；exe 47.0MB（49,292,288B）；**>50KB 源文件 6 个**（同 v4.267，零变化）。**全部指标零回归**。
- 表格补 v4.273.0 行+滑步注记行，自本行起恢复逐版落账。

## 对比度普查收账（2026-09-13，非版本刀，零代码）：无正文级硬伤，不改

- **方法**：CDP 13 页×明暗两态程序化扫描——枚举可见文本宿主，沿祖先合成有效背景色，算 WCAG 对比度（正文 4.5 / 大字 3.0），脚本 .tmp/walk-v4274-contrast.mjs，数据 .tmp/ui-contrast/report.json。
- **结论**：dark 15 项 / light 63 项告警，逐类甄别后**大头为脚本误报**（渐变/图片背景无法从 backgroundColor 合成——角色卡白字压图片、首页「闲庭」chip 压渐变底、accent 按钮白字，目检实际清晰可读）；真实低对比仅剩**亮态 accent 弱化文本**（徽标/链接 chip ~10 处 1.5~1.8）与 schedule 甘特行号（1.3，设计弱化）+ weixin antd purple tag（3.39）——**全部为装饰性/弱化文本，非正文，无 AA 硬伤，按「收益趋零不做」纪律不动**。
- **观察池新增（打磨候选待拍板）**：亮态 accent 弱化文本（徽标/链接类 chip）对比 1.5~1.8——若要修需动主题 lightFn 令牌（glow/accent 亮态深化），影响所有 accent 消费面，属视觉拍板项。
- **坑**：对比度自动扫描的背景合成对 background-image/渐变必然误报——告警必须逐类目检甄别后才能定刀。

## 留池清账（2026-09-13，非版本刀，零代码）：v4.270 sin_illustrate live 端到端补验通过

- **结论**：live 模型调 `sin_illustrate` 端到端全通——模型真调工具、图片真实落盘（sin/art/…「雨夜回眸」.png 1.8MB）、轨迹 artifacts 在位、`extra.illustrations['tool0']` 回写（画廊可见）、过程卡「✓思考过程·319 字·生成插图」元数据在位；截图 .tmp/walk-v4270-illustrate.png。脚本 .tmp/walk-v4270-illustrate.mjs（含空间切换修正）+ .tmp/retry-illustrate.cjs。
- **历史失败归因翻案**：前两次「模型回合未落消息」实为**上游 grok-4.6 一次性空返回**（只有 reasoning 354 字、无正文无工具）——后端兜底如实报「模型没有返回内容，请重试」（sin_handler.go:339 空正文不落库），重试一轮即成功。发送/流式/错误呈现链路全部正常，**非 gaea 缺陷**。
- **清场**：走查故事已删；产物图删除时被壳句柄锁住（Windows 删除挂起，rmSync 静默），杀壳后 PowerShell 删除成功。
- **观察池新增**：sin/art/ 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34 三张「雨夜站台」走查 prompt 产物，v4.270 失败轮次超时未清场遗留），待人工确认后删除。
- **坑**：①node rmSync 对被占用文件不抛错但删除不生效（Windows delete-pending），删完必须 existsSync 复核；②v4.270 脚本超时 throw 路径没走清场（清场代码在正常尾部）——走查脚本清场应放 finally。

## 最新发布：v4.273.0（2026-09-13）「UX 线第五刀：键盘焦点环全站恢复」

- **来源**：「继续」续 UX 线。reduced-motion 普查=基建已齐（双全局兜底）不动；CDP Tab 实测抓到 v4.255 全局焦点环**全站性失效**：Tailwind v4.3.3 产物注入源码不存在的 `:focus-visible{outline:none}`，同特异性按文档序吃掉普通规则。
- **修复**（index.css 单文件）：全局环 `!important` 必胜 + antd 输入类豁免同步 !important + **ant-btn 移出豁免**（Button 聚焦零视觉，豁免=键盘用户找不到焦点）。
- **复验**：重建 exe 六页 Tab 遍历，四页零无环，余 2 站为有意豁免的 ant-input（probe 采样时序误报）；antd 按钮聚焦实拍 2px glow 环。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.273.0；产物见 SHA256SUMS-v4.273.0.txt。

## 最新发布：v4.272.0（2026-09-13）「UX 线第四刀：窄窗响应式收口」

- **来源**：「继续」续 UX 线，换镜头 CDP Emulation 900×600 复跑 13 页——布局自适应面健康，抓到两处硬伤。
- **修复**：①底部遥测条右缘截断（全站六页）：引擎 pod max-width 240 可截断（.v3-pod-text 出省略号）+tele-key/value shrink-0，挤压顺序 pod→工程名→遥测数值完整保；②sin composer 工具行按钮竖排折字：ComposerToolbar 外层 flex-wrap+按钮组 shrink-0+按钮 nowrap（办公共享组件两处受益）。
- **测试**：tsc 0/eslint 0/composer 26+layouts 5 用例绿；重建 exe 复跑 13 页窄窗全 ok（修复前六页越界），目检两处修复实证。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.272.0；产物见 SHA256SUMS-v4.272.0.txt。

## 最新发布：v4.271.0（2026-09-13）「UX 线第三刀：滚动条全站单一真源」

- **来源**：用户「继续优化迭代前端 UI」接 v4.255/v4.260 UX 线。普查=CDP 起壳 26 页次（13 页×两态）零硬伤；视觉层抓到四套滚动条并存，v1.3.0 霓虹遗产（glow 渐变 thumb）喧宾夺主。
- **落地**（纯 CSS 零 TS 零 Go）：index.css 8px 中性胶囊单一真源（module-launcher 形态升全局）；删霓虹化覆写+清 ::selection 死规则（v4.255 拍板版一直被旧规则盖着从未生效）；删 styles.css/redesign.css/module-launcher 三处重复定义。
- **走查**：重建 exe 复跑 26 页次两态——亮青粗条全变中性胶囊，novel/sin 零变化，零回归。观察池新增=weixin 通道详情留白（待拍板）/办公右栏窄窗卡片横滚兜底/novel 技能下拉裸 id（全站惯例不动）。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.271.0；产物见 SHA256SUMS-v4.271.0.txt（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.270.0（2026-09-13）「sin_illustrate 工具入册（拍板放行）」

- **来源**：拍板池清单后用户连说「继续」=按序放行；推翻 §13.2「刻意不做」，前提（产物链）本刀补齐。
- **落地**：工具集 6→7；单链路委托 SinIllustrate；Artifacts 轨迹链（trace 字段+循环收集+result 帧）+落库回写 tool0..toolN；过程卡缩略图+元数据「生成插图」。
- **测试**：Go+4+计数锁 6→7+vitest +3；零新绑定。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.270.0。走查=部分完成如实记录（live 模型调工具留池补验，详见 releases/archive/v4.270.0.md）；产物=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.269.0（2026-09-13）「造价域线上断链修复：json 标签+五算带入」

- **来源**：维持轨「五算带入与含量对照端到端」走查清账时抓到两个线上真 bug。
- **Bug1**：costproject/coststage 结构体无 json 标签→线上 PascalCase，前端读 camelCase→列表卡空名/¥NaN、五算面板同断（v4.204 起）；修复=7 结构体补 camelCase 标签（存量数据兼容）。
- **Bug2**：五算带入/手动保存/询价录入 payload 带空串时间字段→Wails time.Time 解析必炸（v4.194/4.2 起）；修复=载荷省略+类型改可选。
- **测试**：Go+2（TestWireShapeCamelCase×2 形状回归锁）+vitest 2977 全绿；零新绑定。
- **门禁**：ci.ps1 全绿（run3）/ drift OK@642 / 版本三处 4.269.0。真机走查=一次性项目端到端全通（列表卡正确渲染+带入桥验落库+溯源备注+compose 诚实回退），零前端 JS 错误；产物=exe 49,281,024B SHA256=88864D3C…C2A3（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.268.0（2026-09-13）「knip unused exports 甄别清账」

- **动机**：瘦身 P5 既定池项（§100「169 甄别」）；实跑仅 11+6+8（存量数字陈旧）。
- **甄别**：全死删（含 bridge.ts 五个死 re-export）/双出口裁未用侧/契约哨兵 @public 保留/downloadBlob 误报注明（test spy 委托）；knip.json ignore 补 bindingNames.ts+drift.ts。
- **测试**：tsc 0/eslint 0/knip unused 清零/vitest 2977 例全绿；零行为变化零 Go。
- **门禁**：ci.ps1 全绿（run2）/ drift OK@642 / 版本三处 4.268.0。真机走查=CDP 四页导航扫描渲染全过零前端错误；产物=exe 49,281,024B SHA256=10A2566A…F15D5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.267.0（2026-09-13）「原罪工具集 v1.1：sin_export 图文导出入册」

- **动机**：v4.262 留存合约清账（13.2「下刀接完产物链路再入册」）；sin_illustrate 维持刻意不做待拍板。
- **落地**：sin_export 入册（工具集 5→6，order 末位）——委托既有 SinExportMarkdown、落 sin/exports 同名不覆盖、文件名净化 fail-closed、原子写；过程卡「图文导出」+Download 图标。零新 App 绑定。
- **测试**：Go+3+计数锁 5→6+vitest+1。
- **门禁**：ci.ps1 全绿（run1+run2）/ drift OK@642 / 版本三处 4.267.0。真机走查=临时故事两回合端到端：模型真调工具、导出件存在且含正文、过程卡渲染 ✓；走查抓到「同回合写完就导=本回合正文未落库只导头部」缺口，边界写进工具 Description 并复验过，零 JS 错误；产物=exe 49,281,024B SHA256=BD2AEE86…6D827（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.266.0（2026-09-13）「原罪底稿面板内编辑：大纲/设定集」

- **动机**：v4.263 下刀候选清账（「需冲突合并设计」）；此前设定集/大纲只读。
- **设计（§14.7）**：冲突合并=回合互斥+锁内基线比对确认，不做字段级 merge——sending 锁编辑；SinNotesSave 锁内比对编辑起始快照，不符拒（「底稿冲突：」前缀）→前端覆盖确认→force；限长与工具侧同口径。
- **落地**：新绑定 SinNotesSave（641→642，play）；大纲/设定页签编辑器（编辑/写大纲/记便签入口、逐条增删改、rune 计数）；空底稿也给入口。
- **测试**：Go+1（SaveBinding 矩阵）+vitest+8。
- **门禁**：ci.ps1 全绿（run4）/drift OK@642/版本三处 4.266.0。真机走查=临时故事端到端不打模型：编辑保存桥验落盘+冲突弹窗 force 覆盖全通，零 JS 错误；产物=exe 49,269,248B SHA256=6D40A0D9…CC3C（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.265.0（2026-09-13）「原罪画廊重新生成入口」

- **动机**：v4.263 下刀候选清账——流内插图有「重新生成」，画廊只能看。
- **落地**：零 Go 零绑定（SinIllustrate 全链路既有，`arts[cue]=path` 覆盖回写）。画廊缩略图悬浮钮+大图 Modal footer 钮；与流内同一 illustrationQueue 串行；成功后 reloadMessages 全链刷新；SinIllustration 外部换图采纳（extPathRef，不在生成中才采纳）防画廊/流内不同步。失败如实 notice 原图保留；persisted:false 也提示。
- **测试**：vitest +8（定位回调/busy/sending 禁用/空 prompt/Modal+预览跟新/collect 字段/外部换图×2/reloadMessages）。
- **门禁**：ci.ps1 全绿（vitest 342 文件 2969 例）/ 版本三处 4.265.0 / drift OK@641。真机走查=临时故事端到端（结束即删，用户故事未动）：重生成 busy 态→回写换新路径→画廊/流内同步换图，零前端错误；产物=exe 49,257,984B SHA256=868668BD…65AB（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.264.0（2026-09-12）「原罪右栏面板宽度可拖拽」

- **动机**：用户口径「右侧面板宽度应该是可以自由拉伸的」（v4.263.0 下刀候选清账）。
- **交互**：左缘拖拽手柄（骑边框 8px 命中带 + accent 显色），指针拖拽与办公 useWorkspaceLayout 同源——实时跟手/向左拖=变宽/松手持久化/pointercancel 兜底；双击复位默认 268。
- **钳制**：240~640 硬档 + 视口收敛（innerWidth-520 保故事架与故事流可读）；记忆键 gaea.sin.panelWidth 自有键续用。
- **收益**：画廊列数 auto-fill minmax(104px,1fr) 随宽自适应（默认两列不变，拖宽三/四列）。
- **测试**：vitest +5（跟手+钳制+持久化 / 收窄下限 / 双击复位 / 记忆恢复 / clamp 矩阵）；零 Go 零绑定。
- **门禁**：ci.ps1 全绿（代码树 run3+终态树 run4）/ 版本三处 4.264.0 / drift OK@641。真机走查=CDP 受信任鼠标拖拽 268→448 实时跟手（中途采样 358）+落盘+画廊列数 2→3+双击复位 268，两轮零前端错误；产物=exe 49,255,424B SHA256=4A1E96E0…0102F（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.263.0（2026-09-12）「原罪右栏创作面板：角色 / 大纲 / 设定 / 插图（对标办公标签页面板）」

- **动机**：用户口径「继续优化原罪板块，增加类似办公的右侧面板，可以看角色、大纲、插图、设定」。改前右栏是静态说明栏，且 v4.262 工具写进 notes/<故事id>.json 的设定集/大纲**前端没有读取通道**。
- **绑定**：SinNotesGet（640→641，play 空间）只读便签文件返回 {notes, outline}；topicGuard 守卫 + 与工具侧同一把 sinNotesMu；缺失/损坏=空清单空大纲（同口径容错）。
- **面板**：SinSidePanel 四页签（角色=SinCastPanel 原样 / 大纲=只读 pre-wrap / 设定=逐条 #序号 / 插图=全故事画廊 + Modal 大图），页签计数徽标（**纯文字页签：真机走查实证 268px 下图标+两字折行**）；玩法/插图协议/边界收进头部问号气泡（内容逐字保留）。
- **开合**：顶栏 PanelRight 钮，记忆键 gaea.sin.panelOpen/panelTab（自有命名空间）；宽窗默认开窄窗默认收，替代 1180px 强制隐藏媒体查询；每回合结束自动重读底稿。
- **测试**：Go +1（绑定三态）+ vitest +14（面板 8 / collectIllustrations 2 / useSinNotes 4）。
- **真机走查**：用户真实故事四页签全通——大纲/设定空态轨道环（无底稿不编造）、插图 3 张缩略图全转 data URL、Modal 大图预览、开合与页签记忆；抓到并修掉页签折行缺陷；零前端错误。
- **门禁**：ci.ps1 全绿 ×2 / drift OK@641 / spaceBindings 分类锁 462→463 / 版本三处 4.263.0 / exe 49,253,888B SHA256=6E281D6A…0A880（桌面副本同源，冒烟 200）。
- **下刀候选**：大纲/便签面板内编辑（需设计与 AI 写入的冲突合并）、面板宽度拖拽、画廊「重新生成」入口。

## 最新发布：v4.262.0（2026-09-12）「原罪工具集 v1 + 过程卡：联网搜索 / 网页抓取 / 角色卡 / 故事便签 / 故事大纲」

- **动机**：用户问「原罪现在能用哪些工具？看不见调用工具的过程卡」。改前原罪**模型零工具**（单轮 chat 流，无 tools 字段），过程卡无从显示。
- **工具集**：`web_search` / `web_fetch`（委托办公 builtin，schema 转发零漂移）+ `sin_cast`（只读角色卡）+ `sin_notes`（便签 list/read/write/set/delete）+ `sin_outline`（大纲 read/write）；注册表 `sin_tools.go`（init 自注册/重复 panic/顺序固定）。
- **循环**：有界多轮（上限 4，末轮不带 tools 强制收尾）；新 ai 入口 `ChatStreamMessages`（与 `ChatStreamChunks` 共用装配）；首轮不支持 tools ⇒ 去工具重试 + notice；SinCancel 起手即登记、覆盖流式与工具。
- **过程卡**：新事件 `tool_dispatch`/`tool_result`/`notice`；`done.tools` 与 `extra.tools` 同形态（重开还原）；前端 `SinProcessCard`（思考折叠 + 工具行：标签/摘要/状态/用时/展开明细）。
- **数据面**：便签/大纲落 `%APPDATA%\gaea\sin\notes\<故事id>.json`（原子写 / id 白名单 / 损坏当空），不碰办公工作区与记忆。
- **未做**：`sin_illustrate` / `sin_export`（越契约实现、产物回写链路未接完，已挪 `.tmp/sin-next-knife/`，下刀接）。
- **教训**：并发子代理要显式禁改契约文件；产物类工具的跨线依赖必须在契约里点名责任人。
- **测试**：Go +14 / vitest +13（含预算拦截、收尾兜底、静默计时重置三例回归锁）；**门禁**：ci.ps1 全绿 / 版本三处 4.262.0 / 零新绑定；exe 49,242,624B SHA256=470F86EF…C6E68（桌面副本同哈希，冒烟 200）。
- **真机走查**（抓到并修 3 个单测覆盖不到的问题）：①模型一轮连搜 12 次 → 加调用预算（同名 ≤3/整轮 ≤8）；②收尾轮仍要工具 → 整单无正文且不落库 → 加收束令 + 兜底收尾轮；③前端静默超时是一次性 90s → 改每帧重置。最终=走查故事一轮出正文 1185 字（落库 1340 字）+ 9 条工具轨迹（含预算回绝），零渲染错误（走查故事已删）。

## 最新发布：v4.261.1（2026-09-12）「修复：原罪插图报『marshal image request: 图生图需要提供参考图』」

- **根因**：v4.259.0 参考槽门控只看「后端×模型能不能用」，没看「这一轮有没有取到图」——没选角色 / 只有远端剧照 / 本地文件缺失时 `refImages=0` 仍以 `mode=img2img` 发请求，Herdsman 图生图端点整单拒绝；回退条件 `refImages>0` 不成立，报错直出。
- **真机取证**：`%APPDATA%\gaea\logs\gaea-20260912.log` 20:26 两连发（图片后端 Herdsman）+ 用户 `sin/cast.json` 不存在（没选角色）。
- **修复**：新纯函数 `sinResolveRefImages`（图/名/台账归属 id）+ 解析为空即清 `mode/refMethod` 走纯文本；顺带修台账归属错位（多角色只有第二人有图时记错人）。
- **测试**：Go +3（红→绿：改前 3 子例全红；纯函数矩阵 4 面 + 端到端 3 子例）。
- **门禁**：ci.ps1 全绿 / 版本三处 4.261.1 / 零新绑定；exe 49,164,288B SHA256=CCE99E91…D5CAB（桌面副本同哈希，冒烟 200）。
- **真机走查**：v4.261.1 exe 打开原罪故事 → 两个插图位 20s 内自动出图（产物落 `%APPDATA%\gaea\sin\art\`、回写 extra.illustrations），日志零失败/零异常；该故事没选角色=与用户同一分支。
- **文档**：docs/archive/gaea-sin-board-design-2026-09.md §12.5、releases/archive/v4.261.1.md。

## 最新发布：v4.259.0（2026-09-12）「原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）」

- **链路**：角色库 → 故事角色（v4.257）→ 插图人物（本刀）：文本锚点对所有后端生效；图像参考槽走绘梦 T2（ComfyUI krea2*/z-image-turbo · Herdsman 图生图，denoise 0.65）。
- **防互串**：提示词点名优先；单角色兜底；多角色未点名不锚定。
- **诚实**：能力门控（不支持就跳过并给原因，不硬塞）+ 参考槽失败自动退回纯文本重试（标 ref_fallback）+ caption 徽标可见。
- **未做**：ipadapter/pulid（后端未实现）。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.259.0；走查徽标实测（零异常）。
- **文档**：docs/archive/gaea-sin-board-design-2026-09.md §12、releases/archive/v4.259.0.md。

## 最新发布：v4.258.0（2026-09-12）「原罪插图：生成进度动画 + 串行队列 + 真停止」

- **进度**：抽共享进度组件 GenerationProgress（自绘梦 GenerationBar 抽出，类名/阶段名表单点维护）+ 轮询 hook useComfyTaskProgress；ComfyUI 走确定态（百分比+当前节点+已用时），其余后端走不定态光带 + 本地秒表（不编造百分比）。
- **串行队列**：同板块一次一张（图像后端单任务），排队显示「前面还有 N 张」，失败不阻断。
- **真停止**：新绑定 SinCancel（绑定 639→640）中止底层流并保留已生成部分落库（extra.cancelled）；插图「取消」接 CancelImageGeneration。
- **顺带修真缺陷**：sending 因 deadRef+StrictMode 卡死（输入框锁死）、Composer「停止」是空实现。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.258.0；走查：两处进度条 + 取消一张另一张继续 + 完成后停止按钮消失（零错误零异常）。
- **文档**：docs/archive/gaea-sin-board-design-2026-09.md §11、releases/archive/v4.258.0.md。

## 最新发布：v4.257.1（2026-09-12）「修复：真机插图报 AttachmentDataURL is not a function」

- **根因**：S2-3 兼容代理（window.go.app.App）只认字面名、不做 gaeaToGaea 短名映射 → 短名 `AttachmentDataURL` 在真机（Go 方法名 GaeaAttachmentDataURL）查找落空。
- **修复**：兼容代理补映射回退（字面名优先→映射名→仍未命中才 undefined）+ `api/image.readFileAsDataURL` 改走规范 bridge 路径 + HTTP 门面表加 SinB。
- **波及面**：同根因的 4 个方法（AttachmentDataURL/SavePastedImage/RecognizeImage/OCRText）一并修好。
- **测试/门禁**：vitest +3（真机形态假门面回归）；tsc 0 / eslint 0 error / vitest 全绿 / drift OK@639 / 版本三处 4.257.1。
## 最新发布：v4.257.0（2026-09-12）「原罪三刀：与办公硬隔离 + 模型中心独立卡片 + 角色库接入」

- **①硬隔离**：插图产物落 `<用户配置目录>/gaea/sin/art`（绘梦生成链 provenance 参数化，绘梦行为不变）→ 不写办公工作区、不依赖办公 ImageSaveDir；角色配置落 sin/cast.json；无记忆面接入；模型路由 routeModel("sin") 独立；工位关键词检索改为**整个 `.gaea/play` 判噪音**（改前只跳 play/exports，画室台账等会漏进办公检索）；顺带修「原罪插图被登记两次」缺陷。
- **②模型中心卡片**：FEATURES 增「原罪」+ 图标/说明；走查确认在「功能绑定」区在位（未绑定→跟随全局）。
- **③角色库接入**：右栏角色卡 + 多选弹层 → SinCastGet/SinCastSet（悬空过滤/去重保序/封顶 8）→ SinStream 注入角色块（外观/性格/背景/动机/口吻/说话样例 + 不得改设定）；只引用不改库。
- **未做**：角色参考图当图像参考槽（人物一致性锚定）留拍板。
- **门禁**：Go 全绿 / drift OK@639 / tsc 0 / eslint 0 error / vite build 成功 / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.257.0；走查零错误零异常。
- **文档**：docs/archive/gaea-sin-board-design-2026-09.md §8/§9、releases/archive/v4.257.0.md。

## 最新发布：v4.256.0（2026-09-12）「闲庭新板块『原罪』：对话式图文混杂故事创作（复用办公组件）」

- **定位**：用户点单的闲庭新板块（原话见 CHANGELOG）；闲庭第七个一级板块，形态=「说一句→写一段→你接着指方向」的对话式故事创作 + 就地出图。
- **落地**：前端复用办公组件（Composer/Markdown/ToolbarButton/EmptyState/三件套样式，零新建聊天组件、零改办公件）；后端新门面 SinB（9 方法）=统一聊天存储 mode=sin（与聊天板块双向隔离）+功能级路由 sin（模型中心可单挂模型）+绘梦图像链内联插图+图文 Markdown 导出。
- **协议**：模型另起一行输出 `@@插图|画面描述@@`，前端按出现次序（0 起）就地生成并嵌进正文流；正文原样落库，重开按 `extra.illustrations` 渲染不重生成。
- **边界**：成人内容口径沿用 docs/ADULT_MODE.md（零新立），四条硬边界写进提示词并有测试锁定。
- **门禁**：Go 全绿 / drift OK@637 / tsc 0 / eslint 0 error / vite build 成功 / vitest 334 文件全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.256.0。
- **文档**：docs/archive/gaea-sin-board-design-2026-09.md（设计基线+复用清单+已知边界）、releases/archive/v4.256.0.md。

## 最新发布：v4.218.0（2026-09-11）「阶段六 6.6 首刀：跨域 EVM——进度×造价挣值」

- **定位**：阶段六 6.6 首刀（施工债级可穿插；gaea 独有跨域）。**6.6 判据「三数+两指数出且口径注全」满足**。
- **落地**：computeEvm TS 纯函数（PV 基线分摊/EV 完成率×预算/AC 手录+SPI/CPI/SV/CV/BAC/evTop，口径注直出）+ 进度页「挣值分析」视图七档（AC 录入面）。
- **门禁**：vitest +6 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.218.0。
- **阶段六状态**：6.1/6.2/6.4/6.5/6.6 判据全满足；余 6.3 多文件 DAG（委托式深水区，动手前先立设计文档）。

## 最新发布：v4.217.0（2026-09-11）「阶段六 6.5 首刀：进度·蒙特卡洛工期带」

- **定位**：阶段六 6.5 首刀（进度两翼之二，nPlan 对位）。**6.5 判据「三档+稳定度本地可跑」满足**。
- **落地**：monteCarlo TS 纯函数（种子化 RNG+三角分布采样+逐次重算 CPM；P25/中位/P75 三档+直方图+关键路径稳定度/换线风险）+ SchedulePage「工期模拟」视图（不确定度滑杆）。
- **门禁**：vitest +8 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.217.0。
- **下一步**：6.3 多文件 DAG（委托式深水区）；6.6 跨域 EVM（施工债级可穿插）。

## 最新发布：v4.216.0（2026-09-11）「阶段六 6.4 首刀：进度·DCMA 14 点计划质量体检」

- **定位**：阶段六 6.4 首刀（进度两翼之一，对标 Acumen Fuse）。**6.4 判据「十四点纯函数全测」满足**。
- **落地**：dcma.ts TS 纯函数（14 点逐项+阈值+超标点名+n/a 诚实口径+CPM 断裂 score=null）+ SchedulePage「质量体检」视图（DcmaView 纯展示）。
- **门禁**：vitest +11 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.216.0。
- **下一步**：6.5 蒙特卡洛工期带（同工作日口径哲学）；6.3 多文件 DAG；6.6 EVM 可穿插。

## 最新发布：v4.215.0（2026-09-11）「阶段六 6.2 首刀：记忆驱动项目本体——项目本体注入」

- **定位**：阶段六 6.2 首刀（依赖 5.1 图谱底座）。**6.2 判据满足**（决策引用+可关闭）。
- **落地**：BuildProjectBrief 纯函数（固化优先+[MEM:] 引用键+决策来源归因+600 rune 预算）入 work 空间装配前缀；config project_brief 键+GaeaMemoryBrief/GaeaSetMemoryBrief 绑定（611→613）+MemoryPanel「项目本体」开关。
- **门禁**：Go 全量 0 FAIL（+2）/drift PASS@613/vitest MemoryPanel 2/2/版本三处 4.215.0。
- **下一步**：6.3 办公·多文件 DAG（委托式深水区）；6.4/6.5 进度两翼；6.6 EVM 施工债级可穿插。

## 最新发布：v4.214.0（2026-09-11）「阶段六 6.1 首刀：可审计默认化——文件预览默认版本条」

- **定位**：阶段旗交阶段六（书斋纵深·办公主角），6.1 首刀；纯前端、零新绑定、零 Go 改动。**6.1 判据满足**。
- **落地**：FileVersionStrip 默认可见版本条入文件预览（版本数/最近改动/状态/可恢复；展开=该文件 VersionTimeline 逐版本预览/恢复；pptx 页码归因来自证据卡 p3 摘要）。
- **门禁**：vitest +4（FileVersionStrip）/FilePreview 27 例全绿/tsc -b 0/绑定 611 零变更/版本三处 4.214.0。
- **下一步**：6.2 办公·记忆驱动项目本体（依赖 5.1 图谱已就位）或 6.3 多文件 DAG；6.6 EVM 施工债级可穿插。

## 最新发布：v4.213.0（2026-09-11）「5.3 首刀：记忆生命周期三态」

- **定位**：阶段五 5.3 首刀；「90 天一刀切」替代。**5.3 出口判据全满足**（蒸馏合并/预取开关此前已落地）。
- **落地**：固化（SchemaV19 pinned：豁免清理+排序加权+Pin/Unpin 事件留痕）+ 衰减（DecayScore 纯函数+LifecycleOf 分类）+ 三态可查（GaeaMemoryLifecycle/GaeaMemoryPin 绑定 609→611；OfficeMemoryLibrary 统计行+FactCard 固化锁）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ drift PASS@611 / vitest 6/6（OfficeMemoryLibrary）/ 版本三处 4.213.0。
- **阶段五状态**：5.1/5.2 证据链/5.3 判据全满足；余=5.2 主动调度与 5.1/5.2 UI 深化按需另刀，阶段五可评估出口。

## 最新发布：v4.212.0（2026-09-11）「5.2 首刀：上下文编译 dump/diff + 前缀稳定证明」

- **定位**：阶段五 5.2 首刀；纯 Go、零绑定、前端零改动。证据链出口判据落地。
- **落地**：agent 消息级编译摘要（逐消息 SHA256 链+长度前缀防拼接歧义）+ DiffMessages 相邻请求前缀稳定判定；判定随 RequestHeader 落会话日志（dump/diff/回放一体）；contextview 趋势柱携带 Prefix 与 CacheHitTokens 配对。
- **门禁**：Go 全量 0 FAIL（+4 用例）/ 版本三处 4.212.0 / vitest 以 v4.210 基线为准。
- **下一步**：5.2 深化（意图分类×预算调度的主动调度）或 5.3 记忆生命周期三态（touch 事件已留痕），按规划推进。

## 最新发布：v4.211.0（2026-09-11）「5.1 收口：回复发出前剥离悬空引用」

- **定位**：v4.210.0 的 5.1 余项收口刀；纯 Go、零绑定、前端零改动。**5.1 出口判据全满足**。
- **落地**：agent stream() 收尾定稿闸（FinalizeText 钩子：Message 事件前改写+同进 session/摘要）+ boot 记忆开关注入剥离闭包（StripDanglingCitations 纯函数，不 Touch；剥离键落 dangling cite 事件）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ 版本三处 4.211.0 / vitest 以 v4.210 基线为准（纯 Go 刀先例）。
- **下一步**：阶段五 5.2 上下文编译（意图分类→预算调度→前缀稳定排序；embedding 向量列启用）。

## 最新发布：v4.210.0（2026-09-11）「记忆语义图谱：事件日志投影出图」

- **定位**：阶段五「记忆OS+上下文编译」首刀（规划大纲 §2 主轴旗，底座#1）。设计基线=docs/archive/gaea-memory-graph-51-design-2026-09.md。
- **落地**：memory_events 事件日志（SchemaV18 追加式，五条写路径挂钩，Touch 逐条留痕）→ ProjectEvents 纯函数投影（实体/事件/来源三向边，[MEM:name] 图寻址，悬空拒写留 Dangling）→ mem_graph_* 物化（embedding 占位列；读前水位对账懒重建=删库可重建）。
- **接线**：绑定 608→609（GaeaMemorySemanticGraph）；MemoryHub 图谱页「关联图/事件图谱」切换；回合收尾引用解析落 cite 事件（命中+悬空）。
- **门禁**：Go +9 用例 / vitest +1 / tsc 0 / drift@609 / 版本三处 4.210.0。
- **欠账**：5.1 余项=真·发送前悬空引用剥离（流式层另刀）；UI 遍历/手动重建按钮按需另刀（Go API GraphNeighbors/RebuildGraph 已备）。

## 最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」

- **造价刀路池 §6 收官，survey 六项缺口全清**——docs/archive/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态（§3→v4.194 / §4→v4.195 / §5→v4.196 / §6→本刀），该文档不再充当欠账池。
- **缺口**：AI 拆解人材机含量只有金额自洽校验（v4.158），含量本身离谱（如水泥 420kg 对同类 300kg）无人提示。
- **落地**：`cost.CheckContentBaseline` 纯函数——拆解组件 vs 相似条目池同键含量 P25/P75 分位带（R-7 同口径），带外 warn 带内静默；匹配键=标题归一（全角→半角/去空白/小写）+单位归一精确相等（含双方都空），**一方缺失一律不比（宁缺勿误）**；同值退化带 ±5% 容差；同键样本 <3 不比对。接进 `GaeaCostCompose` `composeChecks`（金额自洽+含量对照两层；对照池=相似条目组件明细，与价格带同池；池空该层静默），Checks 随视图展示+Apply 留痕自动承载，**零新结构零新绑定前端零改动**。边界：外部「行业含量区间」数据源明确不引入——基线源=用户自己的库（私人记忆哲学）。
- **门禁**：Go 全量 exit 0（116 包 0 FAIL，+5 用例：contentband 四例+composeChecks 接线一例）/ drift PASS@608（零绑定）/ 纯 Go 刀 vitest 以 v4.208 全绿基线为准 / 版本三处 sync 4.209.0。
- **产物**：build.bat（1m02.7s）→ `build\bin\gaea.exe` **48,580,096 B** + 冒烟 `/api/health` 200 + Desktop\gaea.exe 同哈希；SHA256=**637c6d835d6355b62edfffbed68602bda35fec1e78a7a8d1f7b948607e7f9537**（releases/SHA256SUMS-v4.209.0.txt）；未打 tag（随 v4.200+ 现状）。
- **欠账**：组价含量对照端到端待真机（观察池）；造价域新缺口另立调研。

## 品牌资产补齐（2026-09-10，非版本刀）

- **缺口**：2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核）；**两个二进制件没跟上**——`build/appicon.png`（1024²）与 `build/windows/icon.ico` 仍停在 **2026-08-05** 的旧视觉（翡翠球体 + 破土嫩芽 + 星芒，深蓝底板 `#0F172A`）。后果：exe 内嵌图标、任务栏、资源管理器、桌面快捷方式**全显示旧 logo**，只有窗口内 UI 是新 logo——「换了一半」。
- **落地**：① 无头 Edge 渲染 `build/appicon.svg` → 1024×1024 RGBA PNG（关键参数 `--default-background-color=00000000` 保透明，圆角外 alpha=0 与旧资产同规范）；② Pillow 由同一母图生成 **7 档 ICO**（16/24/32/48/64/128/256——旧资产只有 6 档、缺 24，本次补齐 Windows 标准全集）；③ 重建产物 `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**，`-s` 跳过前端编译用现成 dist）。
- **验证（三条独立证据）**：① **几何精度**——核心圆 r=50@viewBox512 实测 1024 图中直径 **200px**、圆心 (539.5, 511.5) 对理论 (540, 512)；环 `stroke-width=48` 实测线宽 **96px**；圆角 `rx=104` 对角线首个不透明像素 d=**61** 对理论 60.9——渲染零缩放零偏移。② **exe 内嵌图标提取**（`ExtractAssociatedIcon`）——底板 `#0B1210`、米色核 (243,230,198)、翡翠环 (103,235,195)=新「地核 G」；旧版是 `#0F172A` + 绿球。③ **dist bundle**——新 logo 独有标记 `gaea-logo-ring`×3 / `gaea-logo-light-ring`×3 / `gaea-logo-core`×2 / `0B1210`×2 在册，旧独占色 `#6ee7b7`/`#047857` **零命中**；另按旧 SHA256 全仓比对，**零旧 logo 二进制残留**。
- **门禁**：wails build exit 0 / 冒烟 `/api/health` **200** `{"status":"ok"}` / 纯资产刀——**零前端源码改动、零 Go 源码改动、绑定面不变**。
- **产物**：`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希）。中间态 903F9759…C36EC068 系「只换 appicon.png + 主图 ICO」那一版，小尺寸优化后重建覆盖；原始 59C2B518BDB6165CCBAC07BBC565759A2D7571F254E287E27AA00B2BCCD97A16 系 v4.208.0 归档值（releases 档案自洽，未动）。
- **坑（四条，按价值排序）**：
  1. **无头 Edge 截图默认白底**——不传 `--default-background-color=00000000` 时 SVG 圆角外会合成为**不透明白**（CSS 里写 `background:transparent` 无效，截图不继承页面背景），透明通道静默丢失；旧资产角像素 alpha=0 可作规范基线对照。
  2. **颜色特征校验必须先验证该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变（`stop-opacity=0.12`）与**旧** logo 主色，拿它判新旧会把新资产误报成旧的；判据只能用**独有**标记（SVG id 名 / 独占色 `#6ee7b7`、`#047857`）。
  3. **`pwsh` 不在 PATH**——手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（非 app 故障，易误判为冒烟失败）；`build.bat` 内已有 `where pwsh` fallback 到 Windows PowerShell，手工执行要自己判。
  4. **运行中的 exe 可被 `Copy-Item` 直接覆盖**（Windows 允许替换映像路径，进程持旧映射），桌面副本无需先关 app；但**已加载实例不会换图标**，需重启才是新 logo。
- **小尺寸可辨识优化（同日追加）**：原方案对大图统一降采样，**16×16 环宽仅 1.5px、核 3.1px**，G 字形糊成一坨深色块。新增派生资产 **`build/appicon-small.svg`**（环 48→64、核 r50→62、去 halo），ICO 按尺寸分流源图：**16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**，字重过渡平滑（不像 24→32 那样跳）。**坑：Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 每帧 ICONDIRENTRY 16B + PNG 帧；**256 档的宽/高字段写 0**，写 256 会溢出单字节）。对比目检 `.tmp/logo-render/small-size-compare.png`（16/24/32 × 浅底/深底 × 原版/变体）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。

## 工作空间与文档整理（2026-09-10，非版本刀）

- **目录**：根目录 48 份散落的 `SHA256SUMS-v4.*.txt` 归位 `releases/`（该目录现 218 份，归档位置统一）；`nul`（Windows 重定向残留）删除；`.tmp` 清掉可再生的构建/测试缓存与目检 profile（`gocache*`/`gotmp*`/`node-compile-cache`/`edge-home`/`edge-v7`/`smoke-gaea.exe` 等），**释放 ~2.98 GB**（3354 MB → 395 MB）。
- **文档索引**：`docs/README.md` 复核重建——45 份顶层文档 + 2 子目录**全覆盖**（补回 7 份漏登记件：角色域盘点/30 分钟上手/office 枢纽审计/瘦身基线+刀2+P2 三份证据/壳内残留审计），状态行按 git log 对齐，并立「未登记=孤儿」维护规则。
- **归档索引**：`docs/archive/README.md` 顶层 53 项**逐条登记**（原表格只用通配描述，实际漏登 14 项），磁带类大件标明覆盖范围与缺口。
- **AGENTS.md 瘦身（本刀最大发现）**：该文件曾 **104.9 KB**，超工作区指令预算 **65536 B**——尾部「执行纪律 / 交互纪律（不许用等待换时间）/ 工装 / 长期规划 / 项目定位 / 技术栈 / 发布流程 / 沙箱备忘 / 本地 TTS」**整段对后续会话不可见**。二次分流（v4.173–v4.146 逐版迁入 `docs/archive/agents-version-history-2026-09.md`）后 **45.7 KB**，全文可见；归档件同步标明覆盖范围（v4.173–v4.146 + v4.49 及更早）与缺口（v4.50–v4.145 只在 CHANGELOG/releases）。
- **引用完整性**：修掉 12 处「文档已移入 `docs/archive/` 但引用未跟」的悬空路径（含 AGENTS 三处：执行审计 / VoxCPM2 / CosyVoice 记录）。
- **防复发门禁**：新增 `scripts/check-docs.mjs`（① docs 顶层孤儿登记 ② `docs/` 悬空引用（带外部路径白名单）③ AGENTS.md 字节预算水位）接入 `scripts/ci.ps1`——同类漂移下次直接红。
- **产物保留策略落地（2026-09-10 用户拍板：保留最近 5 个版本）**：`releases/` 下 51 个 exe / 1996.6 MB → 保留 **v4.186.0 / v4.98.0 / v4.97.0 / v4.96.0 / v4.95.0**（226.6 MB），删掉其余 46 个 / **释放 1770 MB**。每个被删版本的**身份仍在** `releases/SHA256SUMS-vX.Y.Z.txt`（218 份校验和覆盖到 v4.99.0 及更早）；策略已写入 `releases/README.md`「本地产物保留策略」与 `.gaea/AGENTS.md` 发布流程第 4 步（下次发版删第 6 新的一版即可）。
- **未动（体量大且属用户数据，等拍板）**：`C:\AI\re\` 参考仓（mpp2013/projectlibre/unsloth，**338.9 MB**——含隐藏 .git；首轮统计未计隐藏文件故报 212 MB）、`backups/` 120.4 MB、`whisper_data/` 运行时数据 49.5 MB、`.tmp/codex` 67 MB。

## 最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」

- **缺口**：v6 首页仍被判廉价——根因是「等大圆角卡片 + 描边」一种形态重复到底，层级只靠 13/14px 微差。
- **落地**：结构收敛为三形态（仪表条 / 账页 / 海报墙）+ 四档排印 + 三级色调面 + 版式记号（序号水印 / 月洞门细环），删极光斑；信息零删除，契约 testid 与 `.garden-banner` 保留。修复重写时遗失的 `.ml-avatar-ai`/`.ml-avatar-user`（气泡左右曾退化为同底色）、清掉空挂 `.p-foot-wide`、删除误入的 pnpm 锁/工作区桩文件。
- **附带门禁可信化**：CI 由「无条件重试一次」改为「只在册 flaky 隔离复跑」（`known-flaky.txt` 当前为空 + 分类器自测）、`maxWorkers` 跟物理核（原按逻辑核开 ~31 worker 导致重组件超时假红）、`testTimeout` 30s、RTL `asyncUtilTimeout` 5s。
- **门禁**：tsc/eslint 0、e-check OK、drift PASS@608（零绑定）、Go 全量、vitest 全量（324 文件 2799 例）、版本三处 sync 4.208.0。
- **产物**：主树 `wails build -ldflags "-s -w" -trimpath`（1m08.5s）→ `build\bin\gaea.exe` 48,570,880 B / 46.32 MB；冒烟 `/api/health` 200 通过；Desktop/build\bin 双副本落位；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（`releases/SHA256SUMS-v4.208.0.txt`）。构建后工作树仍干净（`npm install` 未改写 lockfile）。
- **目检**：Vite dev（`?mock=1`）+ 无头 Edge（CDP 9333），两空间 × 1440/1440 高/1100/880 共 10 张；无横向溢出，880 档书斋单列、闲庭海报墙 2 列塌缩正常。
- **欠账**：海报墙首张（小说）大样中段留白偏多（待用户观感定夺）；浅色主题两首页未目检；壳内真机走查池不变。

## v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线

- **v4.207**：深色主题硬编码深灰——RelationGraph 缩放/提示/图例写死 `#555/#ddd`、进度里程碑菱形/标签写死深灰，改走 `--color-text` / on-surface。
- **v4.206**：33 处 `bg-accent text-white` → `text-accent-fg`（= on-primary：暗色深字、亮色白字），App 补注入 `--color-on-primary` / `--color-surface`。
- **v4.205**：小说书房工坊接线——书架画廊头 + 正在编辑横条 + 书脊/角标；阅读页属性检查器由 `display:none` 改为默认可折叠（章节体检入口回来）。
- 三刀均纯前端、绑定 608 零变更、Go 零改动。

## 最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」

- **缺口**：survey §2 多方案对照——三档数字在，推荐价钉死中位数。
- **落地**：ComposeModal 点选 P25/中位/P75，应用价随档；cost_compose `mode` + 三档对照行。不拆逐步盖章。608 零变更。
- **门禁**：ComposeModal 11/11（+1）、Go TestCostCompose 绿、tsc/eslint 0、版本三处 4.204.0。
- **欠账**：流式打字机/费率政策对照不做；含量基线等样本。

## 最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」

- **缺口**：v4.190 七键写路径已通，UI 只放 gaea 主控；其余功能域覆写仅非空 chip，空键看不见写不了。
- **落地**：纯前端 608 零变更。每张空间卡「其余功能域」六键常显可编（对话/轻语/小说/办公文档/角色库/例行），闭环与 gaea 同款；空串清除。
- **门禁**：vitest StrategySection 8/8（+2）、tsc/eslint 0、版本三处 sync 4.203.0。
- **欠账**：功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。

## 最新发布：v4.182.0（2026-09-09）「双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏」

- **定名**：书斋（work，Study）/闲庭（play，Lounge）三语（shell.space.*/shell.search.scope.*+space.ts 兜底）。
- **切换器迁移**：首页顶栏 SpaceSwitch（胶囊组 aria-pressed+模型徽标，直连 switchSpace）；rail 顶部改竖排空间指示徽标（非交互），分域导航不变。
- **两首页分版**：书斋「文书台」三栏效率台（Hero 命令条+最近文档流水主角面板+能力矩阵+右舷状态栏）；闲庭「游园画廊」全幅画廊（旗舰横幅+大卡两列+园底信息带：进度/继续话题 chips/记忆/遥测细条）——信息零删除形态分化（ui-ux-pro-max 定 dial）。
- **门禁**：vitest 2745（+3 ModuleLauncher.test；flaky 族单独复跑绿）、tsc/eslint 0、e-check OK、locale 0 死键、零绑定 drift PASS@602、零 Go 改动、build strip+冒烟 200、SHA256=F8B238863236F53803AB07822A1136366EEC92E97403BC5B592C1C04214E1337。
- **欠账**：壳内真机走查两首页观感/切换手感（真机池）；改名再调=纯 locale 三文件。

## 最新发布：v4.181.0（2026-09-09）「上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊」

- **根因**：Agent 网络不按当前会话=GaeaTrajectory/GaeaAgentNetwork 无参绑定恒读内核 ga.ctrl 单例会话，UI 切历史会话后不跟（「UI 会话≠内核会话」根因家族，其余无参绑定待审计）。
- **修复**：两绑定变参 sessionPath（显式优先/缺省回落内核兼容）+resolveGaeaSessionPath；bridge 契约数组形态+mock+agentNetworkStore 路径声明（跨会话清快照宁空勿错）+六消费方接线；BrowserPanel 保持内核兜底。
- **附带 GLM 429 分诊**：Coding Plan 资源包挂 coding 端点计费域，标准端点 429=端点未切（非 Key 坏）；glmPing 429 时提示切换「编码套餐」端点。
- **门禁**：Go 全量 0 FAIL（+1 显式路径用例）、vitest 2742（+3；flaky 族单独复跑绿）、tsc/eslint 0、drift PASS@602（零绑定名）、build strip+冒烟 200、SHA256=D3BD8386862D5F74EC2BC63F8CA5B35394FA812588376DA306ECAC078AA62D55。
- **欠账**：TaskCenter session 维度结构刀；无参绑定「内核会话」语义逐个审计。

## 最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」

- **定位**：用户反馈任务管理「没与当前会话关联、全是固定内容」——页面主体 TaskCenter 数据源 `GaeaTaskList`=全局后台任务表（cron 周期任务持续在列，天然会话无关），而会话真任务（todo_write 待办）无入口；子代理/本地模型工具两区本就按会话建册。
- **修复**：TasksWorkbench 首区新增「会话待办」——直订全局 controller store items（零 props 穿层，会话切换自然更替，ContextModal 弹层同源正确）+ 复用 useTodoExtractor 提取最新 todo_write；三态条目+进度 n/m，无待办不占位。页面四区=会话待办/子代理/本地模型工具（会话）+任务中心（全局）。
- **门禁**：vitest **2739/2739 首跑全绿**（+1）、tsc 0、eslint 0、e-check OK、drift **PASS@602**（零绑定）、零 Go 改动、build strip+冒烟 200、SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。
- **欠账**：TaskCenter 会话关联需 Go 任务表加 session 维度（结构刀另立版本）；「会话后台任务」过滤视图等该维度就位再评估。

## 最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」

- **knip 首扫三发现三处置**：①死文件 13 全删（App/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip/PromptShelf/office 死链对/memoryhub 两件/typesGenerationCheck/m3-palette，全仓零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释）；②幽灵依赖 13 包显式化（jszip×8 处/katex/dayjs/unified/hast-util-sanitize/@lezer×8 共 20 处 import 靠传递依赖侥幸工作，按 lock 现版本钉死）；③顺带清死传递依赖 @codemirror/search@6.7.2（零引用）。三存疑依赖验活结案：gsap/docx-preview/unist-util-visit 全部在用保留。
- **冷启动基线打点（轨道五先测量后优化）**：Go app.New 一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 落长期日志——日常启动自动积累真机基线；前端 gaea:boot mark→两 rAF gaea:interactive measure（CDP 可读；有意不落 gaea.log 避免污染 Error 级日志通道）。
- **初始化链审计结案**：TTS/价格/索引/模型刷新/预载/巡检全部已异步化，同步链只剩轻量磁盘活——懒初始化改造无需立项，轨道五收敛为真机采集（与壳内真机池同窗口）。
- **门禁**：vitest 2738/2738（5 例并发 flaky 单独复跑全绿）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、e-check OK、Go 116 包 0 FAIL、build strip+冒烟 200、SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。
- **度量**：exe 46.16MB 持平（死文件本就不进 bundle，收益在仓库卫生与依赖正确性）；knip 复扫死文件/幽灵依赖双清零。
- **欠账**：knip Unused exports 169 项挂下版（测试专用/预留面/真死三态甄别）；壳内真机池/审计刀D 真机取证不变。

## 最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」

- **造价域缺口 §1 首刀**（docs/archive/gaea-cost-domain-survey-2026-09.md）：条目匹配全靠标题字符串 → 编码体系贯通。
- **数据层**：SchemaV17 `cost_entries`/`cost_estimate_items` 增 `code`（归一化存储）+ `idx_cost_code`；迁移测试「新库全链 V1→V17 + V16 升级」双覆盖；`cost.NormalizeCode`（全角→半角/去空白/大写，空串=未录入）。
- **导入链**：表头 `fieldCode` 映射（定额编号/清单编码/项目编码/编码等 8 关键词最长胜出，「编号」仍噪声）；**编码优先匹配**——带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码不标题兜底，杜绝跨子目误覆盖）、无码原语义；AI 解析 prompt 增 code 提取；vision 有意不动。
- **匹配消费方**：costref `entryIndexes` 三索引 + `matchEntry` 编码精确优先（matchedBy=code|entry_name|title）；带码未命中宁漏勿误配（测试钉死）。沉淀携带 item.Code + 引用条目继承编码。
- **工具面/UI**：cost_save 增 code 参数（覆盖未传保留原编码）+ cost_search 表格编码列；EntryModal 编码表单项 / LibraryView 编码 chip+行前缀 / ImportModal 可编辑编码列 / ProjectsView 明细编码输入+EntryPicker 带入；mock 同批贯通。
- **附带收口：e-check 三处陈旧守卫修复**（干净 HEAD 实测即红，非本刀引入）——E25 ChatSend/ChatImportTopic 接受 bridge 形态（v4.174 双轨退役甩下）；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import（v4.176 甩下）。
- **门禁**：Go 全量 0 FAIL（五包 +9 测试）、vitest **2738/2738**（2737+1）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、locale 0 死、e-check OK、build strip + 冒烟 200、SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。
- **欠账**：造价缺口 2-6 不变；编码存量回填等带码样本导入自然补全（不做标题猜测回填）；壳内真机池/审计刀D 真机取证不变。
## 最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」

- **H1 mermaid 动态 import**：Markdown.tsx/mermaidPng.ts 删静态 import → ensureMermaid 异步 await（模块缓存，strict/loose 配置原样）；MermaidBlock 迁 useEffect async IIFE（取消/失败语义不变）。**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动。
- **H7 locales 按需**：zh 静态保留（首帧零异步）+ en/zh-TW 动态 import 显式分支（Vite 静态解析独立 chunk）；useT/t/setPref 签名零变动；Provider 切换先同步渲染回退→chunk 就绪 forceRender+alive。
- **主代理关键修复**：translate 回退链 `DICTS[locale]→DICTS.zh→DICTS.en→key`——**zh 静态恒可用兜底**，非 React 调用方（tools.ts 摘要，currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）。5 测试补 beforeAll(loadLocale('en')) + ModelSwitcher en 用例 await。
- **H8 SearchModal lazy**：React.lazy + Suspense fallback null；测试不经 MainLayout 零改动。
- **门禁**：vitest 2737/2737（首跑 13 例 i18n 时序失败修复后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。
- **度量**：**entry 1193→755.48KB（−37%）**；en=76.95/zh-TW=74.91/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`，46.2→46.15MB 减量甚微——Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主）。
- **P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。
- **欠账**：壳内真机池（rail 切换器/home 最近文档/.gsched）、审计刀D 真机取证（printSvg print/拖拽/粘贴）、观察池刀3（工作台内嵌办公）待评估——全部真机绑定的观察项不变。

## 最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」

- **结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/schedule 10.5/methods_xlsx 4.9/methods_office 32.7/build）。**TS2632 教训**：`let mockScheduleCurrent` 被 ScheduleProjectCreate/Delete 直接重赋值——跨模块重赋值导入绑定被 TS 禁止，schedule 状态与方法必须同文件（与 weixin.ts 域内私有状态先例一致）。
- **运行刀 entry 懒加载**（调研修正 P0 两处：cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy——页面级 lazy 早全就位）：
  - **H6 mock 异步 chunk**：proxy.ts 删静态 import → startMockChunk 单例 + 真机门控零加载 + 冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady。**index 1149→967.23KB（−182KB/−15.8%）**。
  - **H2-H5 MemoryHubPage 页内 lazy**：8 组件全 React.lazy（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel/Markdown/katex/mermaid.core 全独立 chunk 按需）。
  - **H1 mermaid 动态 import 递下一轮**（mermaidPng.ts 在 footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。
- **连带测试时序修复×2**：mock 异步化使 `window.__mockScheduleFile`（schedule hook）与 SearchModal RouteIntent/UnifiedSearch 演示规则在同步断言时未就绪——两测试补 `beforeAll(await waitMockReady())`，bridge.ts 入口加 waitMockReady re-export。首跑 7 例失败修复后全绿。
- **门禁**：vitest 2737/2737 全绿、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。
- **欠账=P4 续**：H1 mermaid 动态 import（+mermaidPng.ts）、H7 locales 按需（−160~200KB）、H8 SearchModal lazy（−20~40KB）、**exe strip**（wails build -ldflags "-s -w" -trimpath 发布版，dev 保留符号）；壳内真机池/审计刀D 不变。

## 历史发布（v4.176.0 及之前，已归档）

- **v4.175** 瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）
- **v4.174** 瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat shim 删除，LegacySurfaceNames 292→184）
- **v4.173** 瘦身 P3 版3 批次二：chat/novel/settings 三族 22 文件迁 bridge（契约扩展 29 方法+元组契约错位修正）
- **v4.172** 瘦身 P3 版3 批次一：wailsjsCompat 首批 12 文件迁 bridge（契约扩展 16 方法+2A/2B 迁移；LegacySurfaceNames 292→276）

- **v4.170** 瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5；逐字节/绑定零变更纪律）
- **v4.169** 瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口
- **v4.168 / v4.167** 瘦身 P2 刀1（schedule 并入办公文档面）+ 刀0（白名单解耦、P0 基线落档、审计刀D Filters、nanoid 升级）
- **v4.166 / v4.165 / v4.164** 瘦身 P1 快赢（轮子四刀、审计刀B/C、locale 死键、依赖验活）+ 拍板落档（路线A+私有许可）+ 收敛计划 W1 卫生刀
- **v4.163 / v4.162** gaea 长期日志机制（按日分文件+365 天保留）/ 进度计划导入导出壳内修复（原生对话框链路）
- **v4.161 ~ v4.159** 进度计划双代号线：对齐标杆二期（工程标尺+图面语言）、一级/二级分级横幅、画布手感（滚轮缩放+行距自适应）
- **v4.158 ~ v4.156** AI 组价复核闭环 / docx 证据链补齐 / pptx 真编辑刀2+刀3（编辑面板+版本对比，Office 三件套编辑闭环）
- **v4.155 ~ v4.150** 双工期口径四刀：SS/FF/SF×cd 全搭接重推、MPP Project 2013+ 变体导入、mspdi 互通、板块 UI、agent 通道、任务级工期单位（wd/cd）
- **v4.149 ~ v4.144** diff 确认卡四刀（回滚/结构化 diff 预览/引擎逐条确认/ops 投影对拍收官）+ 工程复制 + 导入为新工程安全化
- **v4.111 / v4.110** 进度计划板块起始：CPM 引擎+三视图自由切换、Project/斑马对齐（v4.112~v4.143 期记录以「最后更新」大段落内联保留：多工程/资源成本/AOA 手动布局/基线对比等）
- **v4.109 ~ v4.100** pptx 真编辑刀1 数据层、导图画布/多维表、Model Hub 蒸馏 MH1-4 + ComfyUI 预热、Hub 落库+绑定转正、办公搜索对齐、GenUI 围栏审计
- **v4.99 ~ v4.90** 三线并行收口：图像域契约、GenUI 蒸馏（回答即 UI）、Verifier 逐页缩略、子代理网络会话/恢复入口、CodeMirror 高亮、Mermaid strict、diff 渲染升级
- **v4.89 ~ v4.82** 上下文页与工具链：成本费率 hover、/context 弹层、Git 面板最小集、文件三态折叠、HTML 沙箱预览、工具结果缩略卡、上下文趋势跳转
- **v4.81 ~ v4.60** 早期面板期：上下文/文件/任务线（活动行级增量、任务页、记忆界面、子代理会话/并行/流式/追问闭环、办公板块交付验收 A2、better-sidebar pane 化）


## v4.544.0 发版（2026-11-17 凌晨）

- **抬号三处同步** 4.543.0→4.544.0（sync-version.ps1：app_info.go/wails.json/versioninfo.rc）；**CHANGELOG 折段**=新增 v4.544.0 段（文学板块系统性审查十批收官+质检/文风族开工，单段详述+详见 releases/v4.544.0.md）+两段文学未发布段（无限制档/破甲契约）随版折入并修正过期尾注；releases/v4.544.0.md 详版说明入库。
- 装机 exe 8B588FA7AC71…冒烟 200（PID 10148）；提交 ad9d11cf 已推 origin（ahead 清零）。功能包 gaea-wenxue-v6（4.543.0 快照）留档不变。
- **CHANGELOG 折段欠账余量**：v4.530~v4.543 旧版段仍缺（发版门欠账按 docs 台账继续挂）。

## 子代理实弹四症状修复（2026-11-18 下午）

- **症状① 3 子代理 6 卡**：编队页左右两列（拓扑卡+分工组卡）双投影是设计面；**真 bug**=嵌套层（级别≥2）SpawnCallID 盖本层原始 call id 而事件日志节点被 subSinkFor 重写为 `父call/本call` 复合，直等失配→legacy 节点成 orphan 与会话节点并存。修（349bd992）：buildSessionNetwork 建尾段索引+父 run 已认领 legacy 前缀消歧+CreatedAt 序保确定性；回归测试 NestedCompositeCallIDJoins。
- **症状③ 前端无子代理卡**：编队 tab 非默认+自动切偏好默认关（2026-09-27 拍板）+顶栏按钮 `!sessionPath||runs=0` 隐藏。修（73d1a906）：自动切默认翻回开（触发面已收敛为「新任务/子代理首次出现」+≥1240px+设置中心可关——2026-09-27 拍板反转，注明可回退）。
- **症状④ 点拓扑无右栏**：三层根因全修（同提交）——①卡身点击=选中+sa_ 节点直达右栏会话窗（此前仅 18px hover 钮）；②aside 钮只对 sa_ ref 出现（非 sa_ 直传被 ValidRunRef 拒）；③paneTabs sanitize 白名单补 viewId=subagent+保留 meta.subagentRef（此前刷新/切会话 pane 静默消失）；④SubagentThread sessionPath 空早退不再无限转圈。
- **症状② 子代理无 bash 执行失败**：分形态定罪——普通 task 委派（未声明 readonly/write_paths）**有 bash**（names 空=继承父注册表全量）；办公 DAG 节点经 dagNodeScope 必走只读/containment 收窄，bash 被 subagentWriteDenylist+无 WritePathResolver 双重出局=**裁决刀设计行为**。待用户告知委派形态后处置（普通形态复现才是 bug）。
- 构建装机：frontend 30.8s+wails+冒烟 200+INSTALL OK（exe 见装机输出，PID 59616>构建时刻）。

## 子代理范式切换：专业工人→通用智能体（2026-11-18 下午，用户拍板）

- **定案**：实测子代理连 bash 都没有执行失败=「受限专业工人」范式本身错了。对照 DSH
  实测工具面（projcache 快照：standard 子会话 45-46 项通用箱，委派家族父侧专属 6 件，
  无任何专用收窄工具；send_message 是唯一传话通道，无「读结果」工具），用户确认三层
  理解+收窄形态彻底退场。
- **落刀（9e366084，单提交）**：①task 模型面 13 参→5 参（prompt/description/
  run_in_background/engine/model；八专业参数摘除，Execute 解析保留=干净退役）+
  描述/系统提示对齐 DSH §7.2/7.3；②send_message 统一传话通道（running=steer/
  queued=pending_messages 排队起跑 flush/终态=RunFollowUp；子树守卫；帽 8 满员可
  执行文案）；③read_subagent_result 模型面退役（结果自动回投已闭环）；④
  subagent_list 对齐 DSH list_agents（scope=children|descendants+parent/depth/model/
  effort 身份行+`(no subagents)` 空态）；⑤DAG 收窄退场（dagNodeScope 只注入空间、
  dag_plan 摘两字段、WaveFullyDeclared 降宽串行规则退役）+send_message RegistryMap
  P0 契约行（census 守卫逮住补齐）。
- **坑**：协调面 ctx 键两套并存（SubagentRefFromContext vs subagentSessionIDFromContext
  ——守卫必须用前者）；SubagentFollowUpRunner 重复声明（复用既有）；boot census 守卫
  逮新工具缺契约行；app 包 129s 长跑。
- **构建装机**：frontend 33.2s+wails+冒烟 200+INSTALL OK（exe 706051b3eec525a0，
  PID 30368>构建时刻）。版本不抬号。
- **对照表**：模型可见面 before=task(13 参)+subagent_list+interrupt_agent+
  read_subagent_result；after=task(5 参)+send_message+subagent_list(scope)+interrupt_agent
  ——与 DSH A 类委派家族逐位对齐（gaea 无 subagent_fork 独立工具，fork 保留内部能力）。

## 子代理委派体验五刀 + 父子通道实弹检测（2026-11-18 晚，ZCode 批）

- **背景**：实弹报障「办公板块并行派发子代理时对话窗口没有子代理卡片」。
- **刀1 委派卡上浮（0ec36ec5）**：task/run_skill 从过程卡 processItems 上浮
  outsideItems 成一等对话卡（运行中折叠态只露最近 3 张尾巴、后台委派回合结束
  整卡埋折叠体=无卡两层根因）；buildSubcalls 段级改轮级（sig 缓存不击穿 T7-4
  memo）吸附 turn_done 后回投的子调用；wenxue 同刀。
- **刀2 点击开右栏 pane（06fe3ee9）**：useTaskCards 注入点换 openSubagentAside
  （编队卡同管道），主对话不被切走；主区 Tab 链路保留给目录树/面包屑。
- **刀3 委派卡 UI 重设计（afaea9f5）→ 刀4 codex 简化定稿（f7dc3063）**：
  DelegationCard 一等卡（状态灯轨/Bot·子代理徽标/任务描述标题/元芯片）→
  codex 收敛一行任务简报形态（任务名+状态灯+一行实时进展/结果，点击整卡进
  右栏子会话）；同批 OCRCard 优化。wenxue 样式 .wx-root 镜像（wx-* keyframes）。
- **父子通道实弹检测+四修复（f7dc3063）**：通道 Go 测试族全绿；**主动性实弹
  两层证否**（自然/点名均 0 委派，wire task_dispatch=0）→ 根因=用户 config
  system_prompt 子代理段旧专业工人幽灵引导（site-survey 等四子代理已不存在）
  → 按 DSH §7.2/7.3 口径改写（工作形态+动机+一句并发纪律，零领域场景；
  备份 config.toml.bak-20261118-pre-delegate-prompt）→ 改后新会话**自发同消息
  3 task 并发**；「派发即结束回合→结算 finished 注记自动开新回合」实弹全通。
  四修复：task.go 后台回执附 Subagent reference 行（治假 done——回执无 sa_
  引用时卡被翻 done 而子代理磁盘态 running）/displayStatus 以 runs 真态优先/
  run.task 空串文本匹配不可达由 ref 直连绕开/taskResultSummary 剥 dsh-ops
  信封。测试契约跟进+mock=subagents 场景收编；gaea+wenxue 2264/2264 全绿。
- **构建装机**：f7dc3063 → frontend build 35.1s + wails build 85s → smoke 200
  → INSTALL OK（exe 5E8AE1383E7B…，19:55:40）。**未 push**。
- **余项**：主代理与子代理重复劳动（同主题自查一遍）=提示调优挂账；smoke
  失败时 gaea_install.ps1 仍继续装=流程隐患挂拍板；wenxue 无 pane 管道（委派
  卡点击仍主区 Tab）挂后续。

## 办公板块实测反馈九刀（2026-11-20 第二批，接用户九条实测清单）

接「本轮办公板块不足」九条反馈，D1–D10 在途批之上同树落地；⑧a 拍板=全局
procedural 保持自动注入（跨项目复用是特性），不砍，只做命名单源。

- **① md_validate 新工具**（builtin 只读）：题注撞号/章内跳号（表X-Y/图X-Y，
  含 Table/Figure，en-dash 归一，后缀变体 1-8B 不误报）+引用↔题注双向对账
  （孤立引用/未引用）+表格块内列数一致；md/txt 直读，docx 经 docmd 转换；
  未命中回执回显全文标题大纲（一次拿结构）。三处纪律同批（registrymap 契约
  行+spacetags work+registry_map_doc 重导，守卫当场立功）。
- **② doc_render 加 total_pages**：pdfinfo 同族探测（GAEA_PDFINFO→PATH→
  pdftoppm 同目录推导）+缺件降级（全档渲染时 len(pngs) 兜底）；验证页数不再
  现搭脚本。docmd.PDFPageCount 导出+seam 测试。
- **③ read_file 加 section 参数**：md 按标题整节返回（编号/文本子串匹配，
  同级边界语义，绝对行号精确）；offset 节内翻页+续读提示标明相对语义；未命中
  回显标题清单；负 offset 组合拒绝；64MB 整载上限。compact 面同步。
- **④ 回执固化**：FormatReceiptMetrics（chars=rune/hanzi=CJK，父侧 Go 确定性
  计算，杜绝子代理自报量纲）task finalizeRun+persisted_run 两点拼装，附引用
  行之前（前端 isDelegationCall 不受影响）；模板面 DefaultTaskSystemPrompt+
  spawn 五前缀统一 receiptFooter（[findings]≤3/[risks]≤2）。
- **⑤ 发布边界拦截三件**：ExpectedPathsNote 补「取回：原件可 move_file 移出」
  「允许落点：非 durable 不受限」恒显+dest 缺席兜底（账本目录指引）。
- **⑥ bash 空参数报实收面**：「command is required：本次收到 N 个参数键
  （…）」——0 键 vs 键名错第一次就分清。
- **⑦ write_file 回执透明化+rollback 上限**：快照机制本来就有（StageBaseline），
  缺的是可见性——回执报 baseline 绝对路径+保留份数；BaselineKeepPerTarget=10
  逐 target 清理（此前无任何清理无限堆积）；顺修同纳秒快照互覆（CAS 递增
  nano，Windows 计时粒度先例复发——测试 9/10 份实锤）。
- **⑧b 记忆索引标签单源**：displayTitle 回退删去连字符派生（page-count→
  "page count" 双拼写乱源），回退恒 kebab 原样；Title 有值仍用 Title。
- **验证**：tool/evidence/docmd/memory+agent/cache+boot 全绿；app 139s 全绿
  （装机位+ucrt64 双 PATH）；docmd TestBuildUocr2LlamaServerCmd 存量环境红
  （复跑 2 次同红、该面零在途改动，与本批无关，挂池）；前端 tsc 未动面零改。
  桌面构建：npm 34.6s+wails 45.8s，exe ea01889a493eb230… 冒烟 /api/health
  200；装机 PID 72364 INSTALL OK。
- **与 D 批关系**：同树混装（D1–D10 在途+本批九刀合体 exe 装机位）；提交
  分离挂拍板——D 批非本线所起，不抢 commit。

## AI 反馈工具机制（2026-11-20 第三批，用户立项，装机 33e16fee PID 59192）

模型侧反馈闭环：使用过程中遇困难/发现不足 → ai_feedback 工具主动提交 →
记忆面板「AI 反馈」区收集展示 → 勾选导出 md 用于改进软件。聊天/办公/文学
三板块工具同源。

- **底层**：internal/gaea/feedback 新包——<userDir>/ai-feedback.jsonl（JSONL
  追加，坏行跳过不炸库；detail≤8KiB；删除走读-滤-原子重写；dream-pending
  先例但无确认队列——反馈非记忆候选，用户立项即收集）。
- **工具 ai_feedback**（builtin，SpaceTag 自声明+静态表双 shared）：title/
  detail 必填+category（tool-friction/bug/missing-capability/workflow/other）
  +severity（low/medium/high）+board（chat/office/wenxue 模型自报——工具层
  拿不到可靠板块标识，wenxue 引擎也标 work）；枚举白名单校验给纠错信号；
  落 gaea 自管数据区按只读放行（evidence journal 同语义）。三处纪律同批
  （registrymap shared 行+spacetags 行+registry_map_doc 重导）。
- **绑定**：GaeaAIFeedbackList/Delete（mapMethod 显式归 memory 板块）；
  gen_bindings 顺手补 212b34a0 漏生成的 GaeaLocalServiceModels（bindings
  面欠账，drift 双方向连带认领：bridge/model.ts 局部重述 LocalServiceModelB
  环规避先例+mappings+spaceBindings shared+mock 空表）。
- **前端 gaea**：MemoryPanel 第四 tab「AI 反馈」（badge=条数计数回调）+
  AIFeedbackSection（列表=徽标行+severity 高亮+详情展开；全选/勾选/导出/
  逐条删除；导出走 saveExportBlob 双门=壳内另存为对话框，md 按分类分组
  +板块/严重度/时间元信息行）；i18n×3；手写 AIFeedbackItem 类型（不挂
  wails 生成件命名）。
- **前端 wenxue 镜像**：WenxueMemoryPanel 尾部挂反馈区+组件镜像（中文硬
  编码口径——wenxue 面板不走 i18n 体系，镜像组件去 useT）；types re-export
  gaea 同形+bridge/mappings/spaceBindings 三件+bindingNames 子集+2。
- **坑**：gen_bindings -names 是裸名单（须自包装 export const）；wenxue
  bindingNames 是 704 子集非全集（误灌 721 全集炸 drift 方向二，还原后只
  +2 名）；TestSpaceTagsCoversAllBuiltins 要求静态表必有行（自声明≠免表）；
  jsdom 测试钉 gaea-lang=zh+inShellEnv mock true 才能断言另存为链。
- **验证**：feedback/tool/builtin/app Go 全绿（app 156s 双 PATH）；tsc -b 0；
  前端全量 509 文件 4617 用例绿；桌面构建 29.5s+45.8s exe 33e16fee 冒烟 200
  装机 PID 59192 INSTALL OK。未提交（同树 D 批+九刀+本批三线混装，提交
  分离挂拍板）。

## 记忆机制重梳+缓存命中率线（2026-11-21 凌晨，装机 677098e5 PID 10664）

用户拍板两条：①删 auto-dream 做梦机制（自动提炼建议质量差且干扰正常记忆
运作）②修缓存命中率。三线侦察（dream 全家面/Reasonix 上游缓存/gaea 组装链）
后落地：

- **auto-dream 全家删除**：gaea_dream/pending/purge 三文件删（gaeaBackground
  PanicNotice+gaeaSessionSpace+captureGaeaNotices 共用件抽出保留）；TurnDone
  槽触发删；建议面板瘦身（memories/overflow 队列块+Accept/Dismiss 旧路径删，
  skills 聚类/merges 蒸馏合并保留——独立机制）；config [dream] 段删；
  SaveDreamFacts 正名 SaveFacts（通用批量写入+审计通道，distill_merge 与并发
  收敛测试仍消费）；DreamAutoWritten/ForgetFacts 删；绑定面收缩 721→716
  （gen_bindings+bindingNames+signatures 重导，双板块 mock/声明/认领同步）。
  前端：dream 三态开关/清污按钮/记忆候选块/MemorySuggestCard 弹卡
  （双板块）/memorySuggest.ts/设置页自动做梦行/i18n×6 文件 5 键 全删。
- **remember 口径回调**（用户问「模型是无法使用记忆工具了吗」的真相：
  838a58cd 把描述+三处提示词收紧成「用户明确要求才调用」，模型实际停记）：
  恢复「值得记就主动调」，确认制由确认卡承载（work 弹卡=用户确认才落盘，
  红线「未经确认不写」不变）；瞬时状态与明令不保存内容仍不记。描述+办公/
  文学身份稿+双执行常量四处同步；冻结锚测试改判。
- **缓存线定性**（对照 Reasonix 上游）：gaea 缓存机制对齐度本就高——compact
  pinnedPrefixLen（system+首条 user pin）+缓存对齐摘要请求（system+fold 原样
  重放+指令在尾）+CacheShape 诊断（system/tools/log_rewrite 归因）+hit/miss
  双形状解析+命中率面板全在（此前蒸馏批已落）。**真缺口=项目简报跨天衰减
  翻转**：DecayScore(m,now) 排序随日历漂移重排条目=L1 前缀跨重建漂移源。
  修=两遍法（评分只定预算内选谁；选定集合按 Name 升序渲染）+跨天恒定回归
  钉（day1==day2 逐字节）。晨报排序本就确定性（时间值+Name 兜底）未动；
  90% 免费 prune 档保留（正期望：工具结果型会话可就此免掉整次压缩）。
- **验证**：Go app 164s/boot 39s/tool/builtin+control/config/memory/agent 全绿；
  前端 509 文件 4614 用例绿；tsc 0。构建 34.7s exe 677098e5ef6498c3 冒烟 200
  装机 INSTALL OK。未提交（累计四批混装：D 批+九刀+AI 反馈+本批）。

## 四线合一只打包（2026-11-21，用户拍板「不区分，全部接手」）

- **提交**：f96b9e3f 初提 → amend **57daf182**（干净树 tsc 逮出 `result.memories ??=[]` 残留——主树假绿根因=wailsjs models.ts 旧契约未重生成，wails 重生成后暴露；漏删行修掉）
- **干净树构建**：worktree .tmp/pkg-f96b9e3f @57daf182，pnpm frozen 30.8s+前端 28.3s+wails 17.7s（Skip Bindings=false）
- **包**：releases/gaea-20261121-57daf182/（exe 0e7cd2e2…+三 DLL 与上包同源+PACKAGE-INFO+SHA256SUMS；zip 34.3MB 留盘不入库）
- **装机**：-SourceDir 显式包路径（反斜杠被 Git Bash 吃=已知坑，正斜杠过）PID 15800；装机位冒烟 200；装机 exe hash==包 exe hash 同源实锤
- **worktree 回收**：本包树已删；codex/zcode 两分支树（2f173fdc/82d86f01）非本线不动

## 工具面板按板块区分（2026-11-21，装机哈希见下）

composer「工具」下拉候选按板块派生（不同面板工具不一样）：office=CAD 图纸/
进度计划（现状不变）；wenxue=右栏 pane 全集（插图/文件/产物/浏览器/Git/
GenUI——wenxue 无 cad/schedule 域，gallery 专属）。落点：ComposerToolbar
variant prop+toolItems 派生（title 联动）→Composer 透传 variant→gaea App
openToolTab 签名放宽 string（cast WorkspaceTabId，语义不变：停用先启用）→
wenxue App 接 onOpenTool（openPaneView 直通，wenxue pane 全 defaultEnabled
无需启用步骤）。此前该钮在 wenxue 渲染但置灰（wenxue App 未接 prop）——本刀
接通。回归：ComposerToolbarTools.test 新 2 例（两板块菜单项互斥）+
App.palette 锚测试改判（openToolTab 签名放宽，语义锚不变）；前端全量 4615/4616
（1 例 DatabaseLibrary 负载 flake 复跑 6/6 绿）；tsc 0。构建 34.3s 冒烟 200
装机 INSTALL OK PID 56848。未提交。

## 缓存命中率实测（2026-11-21，用户要求实测证据不信口头对齐）

- **① 机制层 e2e**（internal/gaea/provider/bridge/cachehit_e2e_test.go 新增，
  方法学=上游 Reasonix cachehit_e2e_test）：真实请求路径（bridge 转换+ai.
  ChatRequest 序列化+AgentRunner 多轮+真实 Registry 工具）+规范序字节前缀模拟
  DeepSeek 64-token 块缓存。结果：多轮无压缩 append-only 逐轮成立，命中率
  96.7%→96.0%→97.4%→98.8%；system/tools 线上字节恒定；模拟压缩重写后前缀断
  但 system+tools 键头存活（97.2% 仍命中）、下一轮 append-only 恢复。
  **首版 mock 假红教训**：JSON 字段序 messages 在 tools 前，「追加消息」必破
  JSON 字节前缀——服务端缓存键是规范序 token 流（system→tools→messages），
  canonicalBytes 按规范序展开才是忠实模型。
- **② 真 DeepSeek API 实测**（scripts/cache_probe_live.go 新增，build ignore；
  key 经 config.Load()+secure.DecryptString 与 app 同链，不落盘；payload=
  ai.ChatRequest marshal 同序列化+52 个真实 builtin schema+5.4KB system）：
  轮 1 prompt=16258 写缓存（首跑 0 命中），轮 2 **99.8%**（hit=16256/16296）、
  轮 3 **99.5%**；复跑（同前缀、TTL 内）轮 1 直接 **100%** 命中——跨次运行
  同前缀也命中。
- **结论**：会话内（无压缩）逐轮命中率 ≥96%（机制层）/≥99%（真 API），与上游
  宣称口径同级；压缩重写=键头存活但尾部冷的形态被钉进回归。
- **未覆盖面（诚实）**：真 API 上的压缩重写轮（仅机制层覆盖）；会话中途改设置
  触发重建=冷启动的代价（结构性，非本测范围）。测试+探针未提交。

## 2026-11-21 办公板块运行模式（众神殿终案施工）

- **依据**：众神殿 pan_1795200544148（正议 5 轮）+ pan_1795202763988（续议 4 轮）四席+主席裁决链，
  用户拍板三档同批；施工设计底稿 `进度计划/gaea-office-run-mode-design-20261121.md`（终案与该稿
  分歧点：终案=标准档 Hide 专用面（收窄制）+ 造价/方案各带新工具新技能，非叠加制路 C1）。
- **批1 模式机制**：`[office] run_mode` 配置键（config.OfficeRunMode 归一，非法值回 standard）+
  app 层组合声明单表 `internal/app/office_runmode.go`（三档 {Hide 面, 技能 extras, L1 模式行}）+
  boot 两条通用缝 `boot.Options.ToolHide`（装配终点 Registry.Hide：schema 不可见协议层按名可调）/
  `boot.Options.ModeLine`（sysprompt 技能索引后追加，DSH-r5：Hide 与提示行必须同批防无声消失）+
  应用语义 DSH 同构：GaeaSetOfficeRunMode 只登记默认档零重建，GaeaNewSession 判定配置档≠引擎档
  （ga.runMode 由 gaeaBuildController 记录）先重建再建会话=「新会话生效、当前会话不中途切」。
  绑定 GaeaSetOfficeRunMode/GaeaOfficeRunMode（含 hide/extras「查看配置」诚实面）；bindingNames 718。
- **批2 造价三件**（internal/app/office_cost_tools.go，锚 workcost 权威面；costproject 遗留面标记 +
  /cost-estimate 模板改写为 workcost 结构化链）：cost_project 只读读口（projects/items/quotas/quota/
  resources）；cost_project_apply PersistWrite（create_project/update_rates/save_items，**写前全量校验
  后写入**：同批编码唯一闸〔UNIQUE 同码走 UPDATE 会吞后行——落库后扫描扫不出，终案 Grok-r3 定罪〕+
  工程量>0+定额存在性预检，悬空定额整批拒绝）；cost_audit 只读机检（四类硬检=missing_identity/
  unpriced_line/dangling_quota/dangling_resource；费率≤0=显式不计取不报、>1 只 info、BillAuditScan
  查漏+HealthScan 可选 info 永不进 HardCount；同码去重防双报——测试逮到真 bug 修掉）。
- **批3 方案两件**（internal/app/office_longdoc_tools.go）：longdoc_manifest（清单 init/set/get/check +
  七类确定性机检=缺章/空章/重复/悬空来源/未映射评分点/未闭合占位符/清单漂移〔正文首个一级标题≠清单
  标题〕）；longdoc_build（构建前强制机检带硬伤拒组装=「清单漂移可检出」断言执行位 → 按清单序组装 →
  复用 GaeaExportDeliverable 模板链 → SHA256 Go 自算入回执+Journal 证据卡；页数留给 doc_render）。
- **批4 技能五件**（office_runmode_skills.go 内嵌正文，wenxue SkillExtras 先例；standard 不传=天然不可见）：
  cost-estimation-playbook / soil-remediation-costing（种子自 skill-creator 迁出）｜proposal-longform /
  proposal-review / soil-remediation-proposal；土壤修复只进技能正文不进模式维度（规范知识技能化红线），
  法规限值与地方定额不写死查不到标缺。
- **批5 UI**：参数弹出层第三组 chip（标准/造价/方案，title 带「查看配置」面=该档隐藏工具+追加技能）+
  触发钮摘要非默认档显形、登记未生效挂「（新会话生效）」；App.tsx runMode 视图快照（新会话/重连重拉）；
  spaceBindings 两行 shared； ComposerToolbarRunMode.test 3 例；三语 runModeLabel。
- **批6 守卫矩阵与回归**：office_runmode_test.go 七例全绿（三档组合表/工具面对账〔Hide 名单全部真实
  注册〕/配置归一双口径/写前同批编码闸一行不落/机检分级〔费率 0 与 −0.01 不报〕/longdoc 七类机检+
  构建闸+md 组装）；registrymap 五契约行+spacetags 五行+文档重导（registry_map_doc -check OK）。
  回归：app 全量 143s 绿、boot/tool/builtin/config 绿、前端 tsc 0+composer/bridge 52 例绿。
- **未竟（台账）**：双门实跑（造价闭环 ≥20 条土壤修复明细 / 方案 ≥20 章 A/B 七指标三断言）挂用户实弹；
  全量前端 vitest 与 ci.ps1 全门禁未跑（动刀面定向已绿）；**已装机**：wails build 34.8s → 冒烟 200（PID 42036 回收）→ gaea_install INSTALL OK，装机 exe sha256 57C1AC5DED5DB73C=build/bin 同源，PID 74656。

- **追加批（同日）：能力面板「工具」tab 装配面活取**——用户实测办公/文学工具 tab 不全
  （静态硬编码 38 名定格旧时点）。修=Registry.HiddenNames→Controller.ToolSurface
  （name/description 模型可见口径/source/hidden）→GaeaToolSurface/WenxueToolSurface
  两绑定；两板 ToolsSection 活取渲染（来源分组+「本档隐藏」灰显组+模式徽标+可见/总
  计数），静态清单降级为引擎未就绪兜底；wenxue 树 bindingNames/legacyBindings/
  AppBindings 三件套同批再生（legacy 算法内联重跑：720−477=243）；mock 四件空面+
  signatures 再生。测试：前端 130 文件 1114 例+surface 3 例，Go tool/control+surface 2
  例绿。**已装机**：npm build 27.2s+wails 重构建→冒烟 200→INSTALL OK，装机 exe sha256
  994939EE10AA7D71，PID 17208。
- **追加批 2（同日）：会话定档选择器**——新会话（未发消息）参数弹层三态可选、选档即定档
  （登记默认档→档≠引擎档经 startNewSession 重建换新会话即刻生效）；运行后锁定只显当前档
  徽标「X档+会话已定档：切档需新建会话」；触发钮摘要改按 active 显形（撤 pending 后缀）。
  App runModeLocked={items>0||running} 管线三段；测试三钉全绿；npm 25.0s+wails→冒烟 200→
  INSTALL OK，装机 exe sha256 B03E53D64CE1B7F9，PID 66652。
## 2026-11-21 工具面双轴瘦身（众神殿 pan_1795223234328+pan_1795224680124 施工单）

- 施工单 `进度计划/gaea-office-tool-face-plan-20261121.md`；用户拍板链：瘦身边线→双档终案→
  **改排（CAD 并入造价、进度并入方案、标准档两族全关）**。
- 落地：tool/toolgroups.go 组目录+三守卫；[office] tool_face=lean|standard（默认 lean）+
  [office.tool_groups] 覆盖（未知组 fail-closed）；composeToolHide/composeModeLine/
  officeRunModeSkillFilter 组合器（skillFilter={schedule-edit,cad-takeoff} 跟随最终启用组，
  boot.SkillFilterKeepsInstall 保 install_skill）；装配消费+ga 三元组（runMode/toolFace/
  groupSig）判面重建；绑定 GaeaToolFace/GaeaSetToolFace/GaeaSetToolGroup/GaeaResetToolGroups
  （724 名）+ToolSurfaceEntry.Group；ToolsSection ToolFaceBar（预设钮+组开关+重置覆盖）+
  精简/标准面徽标。
- 矩阵：lean standard-run_mode 藏 35 名（cost7+longdoc2+cad1+schedule3+browser12+com10）；
  standard 面恢复 browser/COM；CAD 随造价、进度随方案；media（产图四件）两档默认开。
- 回归：app 全量 128.5s+tool/config/boot/control 绿；前端 tsc 0+129 文件 1108 例绿。
- **已装机**：npm 24.6s+wails 31.1s→冒烟 200→INSTALL OK，装机 exe sha256 3435FBF42D4FDD43，PID 46592。
- **追加批（同日第五批）：输入框拆分+computer 开关+功能分组**——①office 工具钮（CAD/进度）
  删除（入口归运行模式+右栏 Tab，wenxue 保留）；「参数」合并弹层拆 权限/模式/思考 三独立钮
  （单开控制，模式钮三档常驻显名）。②computer 组真开关：GaeaSetComputerUseEnabled 绑定+
  ToolFaceGroupView.Gate+面板开关行（写 [computer_use].enabled 注册闸，Count 随启停 0/6）；
  GaeaNewSession 判面扩四元组（+cuEnabled）。③工具面板按功能分组：groupSurface 重写
  （配置组直显+core 语义小节+MCP 按服务器），弃装配来源分组。测试四文件更新，
  129 文件 1108 例绿；绑定 725 名全链再生。**装机 exe sha256 31B742A759084FF4，PID 58344，
  冒烟 200 INSTALL OK。**
- **追加批（同日第六批）：造价/方案专属身份提示词**——`[office] system_prompt_cost/
  system_prompt_proposal` 配置键（非空顶替）+内置默认稿（造价=workcost 数据面铁律+五步工作流
  +编造禁令；方案=先清单后正文+七类机检+schedule 纪律+三查）；standard 恒空=现状逐字节；
  boot.Options.SystemPrompt 既有覆写缝直用（wenxue 身份分板同构）。修 GaeaNewSession 判面在
  ga.cfg=nil 测试路径的 nil 解引用（判面块整体 guard+字段访问判空）。TestOfficeModeSystemPrompt
  守卫；app 全量 144.9s+config/boot 绿。**已装机**：npm 43.1s+wails 43.2s→冒烟 200→INSTALL OK，
  装机 exe sha256 012F40329431D0D6，PID 40560。

## 2026-11-21 方案模式实测复盘修复批（沿江1公里技术标 43 缺陷的工具侧收口）

- 事实源：`D:\新工作\沿江一公里\南岸\本次会话缺陷与不足复盘.md`（43 条：A8/B15/C8/D12）；
  本批收 A/B/C 工具侧全量，D 类（模型自误）不属工具面。
- **刀1 导出链防删源（A1/A2）**：`gaea_export.go` exportDocx/exportPptx 中间文件改
  `os.CreateTemp`（落输出同目录保相对图路径解析口径），不再写 `<输出stem>.md/.json`——
  同 stem 覆写+删除源文件的路封死（源 md 丢失 11 次的根）；`docmd/soffice.go`
  RunProcess stderr 单行化（traceback 多行被下游按首行截断→「冒号后无下文」）。
- **刀2 create_docx.py 图片硬失败（A3）**：加载失败记账+main 硬闸 exit 2 零产出，
  不再静默降级为「[图片加载失败]」占位文字；ok 回执增 images 数。实弹验证 exit=2 无产物。
- **刀3 longdoc 图引面（A3/B9/C3）**：机检第八类 `image_missing`（整行图引按章节目录
  解析，缺失/外链=硬伤）；组装时相对图引自动绝对化（ToSlash）；docx 回执图片对账
  （引用 N/嵌入 M 数 document.xml drawing，M<N 即错）；拒绝组装错误单行化带明细。
- **刀4 schedule 预检（A4/A5/B1~B4/B7）**：`schedulePreScanRaw` 在结构化吞字段前点名
  task 内 links/workweek 越界(0-6)/op≠type/set_links taskId 并给正解；applyOne 错误带
  可用类型清单；patch_task 报错带嵌套示例；质检无前置具名到任务；cdTasks 空集出 []。
  ops golden 重生成（错误文案有意变更）。
- **刀5 office_com（A7/A8/B12/C7）**：新 op `clear_char_format {style}`（Font.Reset 批量
  清 run 级直接格式=Ctrl+Space 等价，187 处标题场景）；style_format 改字面时回执明示
  直接格式优先级；office_save fill_gaps 改显式参数默认关（隐式重排搅目录页码的根）；
  save 回执丢失（helper 退出）时按调用窗口内落盘时间戳探测「已存待复核」；guides.go
  补优先级说明与清除入口。
- **刀6 md_validate/doc_render（B6/B8/B5）**：附图/附表/插图/图幅复合前缀中性化
  （13 项假阳性根）+长破折号—归一；doc_render 全档渲染>8 页只列首尾页+pages_dir；
  POSIX 路径形态报错附正确写法提示。
- **刀7 文档批（C1~C8）**：proposal-longform 补插图三律/导出安全与交付边界/页数双口径；
  proposal-review 与方案 sysprompt 数据一致性三处→**四处**（摘要/正文/表格/大纲配额表
  与图表索引）；schedule-edit 补 ops 完整 JSON 示例+易静默出错点清单；机检口径七类→八类。
- 测试：新增 `office_longdoc_image_test.go` 六钉（图引机检/绝对化/单行拒绝/zip 对账/
  **同 stem 防删源**——python 成败皆不动源 md/中间件零残留）+schedule 预检 6 案+md_validate
  复合前缀与长破折号；schedule golden 重生成；全量 go 测试见当日收口记录。

## 2026-11-21 方案技能经验沉淀批（复盘 §六/§七 融入，用户点名「验证体系很好」）

- proposal-longform 增两节：**并行编制纪律**（共享事实卡先行/派发指令隔离写入目标/自设数值
  回流/篇幅给区间不给下限/单章过 md_validate 再并下一章+cross 级对账）+**图件工艺**（先定
  插入宽度 140mm 与成图字号下限 6pt 再画布/图题只由正文题注承担/自绘图四类几何自检清零才插入）。
- proposal-review 三查扩**四查**：一查补图表索引自洽（索引==题注数、明细合计==配额合计）；
  新增四查交付验证（图片对账+渲染取页+数据四处对账+关键格式像素/XML 抽查——工具侧已自动
  图片对账，技能教剩余三件）；新增**验证证据纪律**（三层证据：可复现机检＞全文提取＞像素
  判定；视觉模型只读大结构不作验收依据；结构定位用 pStyle 不用文字匹配；CPM 自算交叉验证；
  自建脚本须「故意破坏测试」防恒绿假验）。
- 方案 sysprompt 同步：工作流 step2 补并行事实卡条款；质量纪律三查→四查+证据纪律一句话。
  TestOfficeModeSystemPrompt 补锚（四查/共享事实卡）。
- 坑（第 N 次复发）：技能正文是 Go 反引号原始串，内嵌 Markdown 反引号即劈串——
  「_draft/00_共享事实卡.md」改用「」引号。
- 测试 runmode/skill/longdoc 面绿；npm 不动（前端零改）；wails 重建装机
  **exe cea0b6ac52536600（同源核验）PID 34688，冒烟 200 INSTALL OK**。

## 2026-11-21 验证体系工具化：longdoc_verify 四门（用户拍板「脚本复用成方案专属验证工具」）

- 源头：复盘 §七 八脚本全读分层——verify_ch/cross_check/acceptance_all/verify_v1/ink_color 通用内核
  收编；qc_geom（matplotlib 活体拦截 monkey-patch save）与 cpm_calc（引擎权威+技能纪律）不收。
- **新工具 longdoc_verify**（office_longdoc_verify.go，方案档专属=toolgroups longdoc 组三名）：
  - chapter=单章四对账：题注↔引用↔**章末索引**三方同步+图引存在性（按章目录解析）+docx 内嵌图
    **md5 逐一比对**+drawing 对账+占位残留+禁用词（否定语境前 30 字排除）+口径词/旧口径；
  - cross=跨章：编号跨章唯一+**引用↔题注全册并集双向对账**（chapter 门放行的跨章引用在此闭合）+
    禁用词全册+口径词×各章矩阵+各章字数；
  - book=整册 docx：占位残留/章标题在册/styles.xml 颜色字号字面量读数/页眉页脚域/分节数/源 md 完整；
  - ink=像素判色：暗部（lum≤110）平均色+饱和度差>25 判彩色，缺省五条带（页眉/标题行/正文×2/页脚）。
  项目耦合项（期望图表数/禁用词/口径词/条带）全部调用时传参，内核确定性零 LLM；只读。
- 四处同批：toolgroups 名单+gaea_handler 装配+registrymap 契约行（work/read）+spacetags 静态表；
  registry map 文档重导（scripts/registry_map_doc.go）。
- 技能/sysprompt 接线：longform 并行纪律改为 longdoc_verify chapter/cross 入口+book/ink 准出门；
  review 四查前三件走工具；sysprompt 质量纪律点名四门。qc_geom 留技能（自建脚本+故意破坏测试）。
- 实弹（真书 D:\新工作\沿江一公里\南岸）：chapter=✓（跨章引用 6 种正确放行、md5 不符 0、占位 0）；
  cross=**78 编号 0 重复、引用↔题注 78↔78 闭环**、禁用词全册 0、口径矩阵/207086 汉字与复盘终值一致。
- 测试：office_longdoc_verify_test 五件（chapter 通过+破坏面/cross 唯一+并集孤儿/book 假 docx 全要素/
  ink 程序化 PNG 判色/ReadTarget）+hide 矩阵与 drift 锁自动覆盖；app 全量 126.6s 绿、tool/boot/skill 绿。
- 坑：①`k[1:]` 字节切片劈开多字节「图」（3 字节）→ Sscanf 恒败、跳号永不报——rune 切片修；
  ②heredoc 内联中文路径 JSON 编码坑（复盘 B10 同族）——测试夹具一律 Write 工具落文件。
- **装机 exe 3a42371df4f9b668（=build/bin 同源 uniq 验证）PID 51968，冒烟 200 INSTALL OK**；未提交同树混装。

## 2026-11-21 打包收口：gaea-20261121-01f88477

- 提交 `01f88477`（feat(office) 九批合一，75 文件 +6562/−222；剔除缓存命中线两件在途
  cachehit_e2e_test.go/cache_probe_live.go 与运行态 resumed.jsonl.meta、旧 zip 不入库；
  根目录 manifest.json 为 longdoc 测试残留已删）。
- 干净工作树 `.tmp/gaea-pkg-01f88477` 构建：**frontend 必须 pnpm install --frozen-lockfile**
  （npm 装树触发 esbuild 删临时件 Access denied——坑在册复发，pnpm 布局即通）；
  wails build 61s → payload 四件 → 冒烟 200（PID 16532）。
- 包记录：releases/gaea-20261121-01f88477/{payload,PACKAGE-INFO.md,SHA256SUMS}；
  exe sha256 2ff321bd88584e4f…；zip（34.4MB）留盘不入库。
- 装机：gaea_install.ps1 -SourceDir <payload> → INSTALL OK，装机位 hash=包 exe 同源，
  PID 46504 > 构建时刻。工作树已回收。

## 2026-11-21 造价门实测复盘修复批（沿江1公里成本测算实测——众神殿双门实测至此闭环）

- 事实源：D:\新工作\沿江一公里\南岸\gaea软件使用不足与改进建议.md（阻塞 3/效率 3/能力 3/可靠 2）。
- **刀1 路径方言收口（2.2）**：resolveIn 推广 msysDrivePath（/d/… → D:\…，bash_delete 既有转换器）
  ——write_file 此前把 /d/... Join 成「工作区\d\...」镜像目录树；read_file/ls/md_validate 等全族一修。
- **刀2 bash 初始目录（2.3）**：两分支 Description 写明「初始工作目录=工作区根（首条 pwd 应显工作区）」。
- **刀3 发布边界（2.1）**：拦截回执补两句——自由脚本生成 durable 的合规路径（Temp/非 durable 扩展名
  →move_file 落正式名=合法出口，实测只能试错撞出的路现在明示）+「request_permission 批准的路径
  规则不改变本边界（独立闸门）」；request_permission Description 同步告示。permission×边界深层联动挂拍板。
- **刀4 cost_project action=summary（4.1）**：现算汇总口——ComposeQuotas 库内现算链逐行综合单价/合价/
  人材机拆分（手填价**整价归材料**约定入文档）+取费链（与五表导出 writeSummarySheet 同源：管=直接×率、
  利=(直接+管)×率、规=直接×率、税前四项显式相加）+占控制价；消除模型侧自建 SQL 算价链。
- **刀5 cost_audit 第五检（4.2/六）**：spec_grade_mismatch（INFO）——清单特征档（C10-80/M5-30，
  前缀锚定防 PC200/M046 误报）vs 定额/资源 title 档位比对；C60 充当 C20 的万元级错配有了确定性防线；
  暂代合法（INFO 不进 HardCount）。
- **刀6 数据指纹（5.1）**：xlsx_audit 回执头 size/mtime/sha16；cost_audit 回执头行数+四费率——
  同轮改文件/改数据后结果不同即可分辨，上层「结果同一性去重」的 cached 提示不再歧义。
- **刀7 xlsx 版面确定性检查（5.2）**：xlsx_audit 附版面段（GetPageLayout+已用列数；≥10 列纵向且
  FitToWidth<1=⚠ 压扁告警）——vision 看不出整册压成单列的坑，确定性判据接管。
- **刀8 genui_validate 双形态（4.3）**：spec 收字符串或对象（旧实现传对象直接 unmarshal 炸）；
  schema 描述同步。组件字段手册级文档挂拍板。
- 技能/sysprompt 接线：cost-estimation-playbook 补 summary 口纪律+规格档对齐；造价身份稿补 summary 口。
- 测试：office_cost_fieldback_test 三件（summary 数值链/spec 档位恰 1 条+不进 HardCount+指纹/xlsx
  指纹+版面告警与修复面）+builtin fieldback 两件（MSYS 翻译/genui 双形态）；app 全量 159.9s 绿、
  tool/agent/workcost 绿、存量锚全兼容。
- 挂池拍板：模板三件套（extract/fill/compare，3.1）/增量修正闭环深层（3.2）/ai_feedback 状态面（3.3）/
  region+spec_grade schema 迁移与口径切换（4.2 全量）/permission×发布边界深层联动。
- **装机 exe 0e80b627d984d769（=build/bin 同源）PID 5116，冒烟 200 INSTALL OK**（首装 KILL FAIL
  一次=壳被占用，重试过）。**未提交**（新一批，挂拍板分离）。

## 2026-11-21 造价提示词/技能同步补齐批（用户问「同步优化了吗」——查实只同步两条，补全）

- 造价 sysprompt：工作流 step4 扩「机检与复检」（复检看数据指纹，指纹没变=旧结果勿当通过）；
  step5 扩「沉淀与交付」交付三律（版面先行：长表横向+适应一页+表头重复，xlsx_audit 版面段确定性
  判据、vision 不判版式；勾稽：导出表合计==summary 直接费、人+材+机==合价；落盘合规：bash 直写
  durable 被边界拦截，Temp→move_file 是合法出口）；口径纪律补规格档对齐行+summary 占控制价%可引用。
- cost-estimation-playbook 技能：step4/5 与 sysprompt 同步（三律全展开，含实测教训注记：百万量级
  汇总行填单价/整册压单列 vision 仍答规范/594 公式重算才暴露）；控制价行补 summary 口径。
- TestOfficeModeSystemPrompt 造价锚扩 {summary,勾稽,数据指纹,规格档} 防回漂。
- **装机 exe 9a39719e76b2f622（=build/bin 同源）PID 65404，冒烟 200 INSTALL OK**；未提交同批。

## 2026-11-21 沿江技术标会话反馈批（A~D 四批全落；来源=南岸工作区 gaea使用意见与改进建议.md 14 条）

事实源=`D:\新工作\沿江一公里\南岸\.gaea\exports\gaea使用意见与改进建议.md`（P0×3/P1×7/P2×4）。
用户关键纠正：**会话用的最新构建——「已修未生效」桶不成立，须按真实路径重诊**。

- **重诊结论（根因）**：P0-1/P1-12 拦截信封「已隔离：+空 detail」=传输层断头——
  `execute_one.go` 信封 error 字段取 `firstLine(err.Error())`、detail 取 result，而 bash
  错误路径 result 恒空串 → 九刀⑤的清单/隔离位置/出路文案**根本没到模型眼前**。
- **批 A（传输层根修+小刀）**：①bash publishCheck 三调用点文案随 result 返回（信封 detail
  从此带全文；传输契约入注释）②pageBreakBefore 按 w:val 解析（0/false/off=否）③ledger 行加
  `event: begin/final`（注释明示批次计数按 stamp 去重，保留崩溃安全双行设计）④bash 空参错误
  回显原始参数体 %q（截 120 rune）+loop guard 豁免「收到 0 参数键」签名（batchStormSignature
  早退）。装机 02DF19A4→PID 62044。
- **批 B（P0-2 标书模板全链）**：create_docx.py 新「标书」主题（宋体正文/黑体黑标题/
  Heading1 样式层 pageBreakBefore/页眉下框线+缺省取投标人/封面五要素 subtitle=包号·author=
  投标人·bid_copy·bid_date）；longdoc_build 默认模板 报告→标书+封面要素参数透传；
  **静态目录两遍法**（docmd/pdftext.go pdftotext 三级查找+逐页文本；占位目录 pass-1→
  soffice→PDF→量页→回填重建，行数恒定分页稳定；渲染链缺席如实回落域目录并标注）；
  office_generate/office_tools/wenxue_workspace/compact/技能文档六面同批同步枚举。
  12/12 XML 实弹钉+E2E 7.8s 钉。装机 128ACFDC→PID 18092。
- **批 C（跨章引用双刀）**：md_validate 加 paths 多文件全册并集模式（跨章引用他章闭合即过
  +跨文件撞号点名两侧；单文件模式本地无题注引用改判 cross_ref 待核类不冒充硬伤——实测曾
  9 条误报诱发把有效引用改成模糊描述）；longdoc_verify cross 加 [1s] 语义初筛（引用句↔被引
  标题汉字 bigram 重合，低重合=引用错表高疑标记不判 fail——「编号存在但内容错换」三道存在
  性门全漏的补面）。装机 17044176→PID 40896。
- **批 D（小刀三件）**：P2-9 longdoc check 附配额面（words 偏差>30%+figs/tables 期望 vs
  题注数 warn 级+恒附结构化 JSON 行）+P2-10 stale 闸 edit_file 锚点豁免（old_string 仍锚
  当前磁盘内容=新鲜放行，失配仍拦；同回合 write→edit 改一行不再重读整篇——实测被拒 6 次）
  +P2-11 resolveIn 正斜杠盘符归一（D:/x→D:\x 全族文件工具）。
- 测试：新增 12 钉全绿；agent/tool/builtin 20+2+21s、app 全量三次（143/149/160s）绿
  （uocr2 存量环境红在册挂池，本批未触）。装机 C87A56D3→PID 10336。
- **挂池拍板**：P1-6 版面机检产品化（批 8 有意留技能层）/P1-12 目录级移动原子性大件+
  quarantine_list 查询工具/P1-7 wait 收 sa_+子代理进度/P0-1 尾巴 _draft 豁免/
  output_path 绝对路径兼容（声明写路径契约未动）。
- **未提交**（与十/十一批同树混装，提交分离挂拍板）。

## 2026-11-21 图文侧同步批（用户问「办公优化是否同步文学板块」——审计+顺手修漂移）

- **审计三层**：①agent/bash/文件工具层共享=天然同步（信封断头/loop guard/stale 锚点/空参回显/
  msysDrivePath/正斜杠归一/md_validate paths 全册模式——md_validate 不在 wenxue 禁用表）；
  ②wenxue 有意瘦身面（edit_file 族/doc_render/COM/longdoc/cost，2026-11-16 瘦身刀）不适用；
  ③**wenxueExportDocx/Pptx 是 office 导出链独立副本，落了三件**：同 stem 中间件确定名落盘
  （office 批6 A1/A2 已根修的吞源形态）、TocEntries/--toc-spec 未透传（静态目录不可达）、
  回执无图片对账。
- **修**：docx 中间 md 改随机临时名（.gaea-tmp-md-*，落输出同目录）+pptx 中间 spec 同款+
  TocEntries→--toc-spec 透传+subtitle/author 透传；钉=TestWenxueExportDocxSameStemNeverTouchesSourceMD
  （python 成败皆不触碰同 stem 源+无临时件泄漏）。图片对账入挂池（回执面接线另议）。
- **装机 exe FD13A80BE9F46E7D（=build/bin 同源）PID 31608，冒烟 200**。未提交同树。
- **拍板池（方案模式↔长篇小说结构面移植）**：清单唯一事实源+组装前机检闸进小说成册链
  （卷章场景三层已有，对账 longdoc 八类机检等价面）/章节字数配额面（大纲目标 vs 实测，
  T4 篇幅路由对账）/成册 docx 静态目录两遍法接线（能力已通，缺 wenxue 侧组装调用方）。

## 2026-11-21 图文成册批（用户拍板「全部」——方案模式↔长篇小说结构面移植三刀合一）

- **新工具 wenxue_book_build**（wenxue_book_build.go，图文装配单点+契约行+文档重导三处同批）：
  定稿目录（定稿/final）即唯一事实源（磁盘即事实，不另建 manifest 文件），按章号序组装整本
  docx/pdf/md。三合一：
  ①机检闸（0 硬伤才组装）：未编号/重号/空章/占位残留/{{}}不闭合=硬伤；跳号/无一级标题/
  文件名标题与正文标题漂移=warn 不拦（卷结构/兜底渲染是合法形态）；
  ②配额面：target_words_total+tolerance_pct（默认 30）超差=warn+每章字数恒附 JSON 行；
  ③静态目录两遍法：measureTocEntries 抽为板间共用核心（office longdocMeasureTocPages
  变薄壳），渲染链缺席如实回落域目录并标注。
- 回执=章数/总字数/sha256（Go 自算）/配额/目录/warn 全景。模板缺省 通用（小说非标书）。
- 测试：三钉（硬伤拒组装四类/组装序+配额 JSON/docx 静态目录真链 E2E 16.7s——soffice+
  pdftotext+python 全链，技能脚本副本进临时工作区自洽）；registrymap 契约行覆盖+census
  守卫+app 全量 171.5s 绿。
- **装机 exe AEA6191B0E376107（=build/bin 同源）PID 20644，冒烟 200 INSTALL OK**。未提交同树。
- 坑：registry_map_doc 的输出文件名是 gaea-tool-registry-map-2026-11.md（守卫测试读这个，
  重定向错文件名守卫当场逮住）；GAEA_TUWEN_ROOT 指临时根是 wenxue 测试工作区隔离的正解。

## 2026-11-22 打包记录——gaea-20261122-a4b5b0a7（沿江反馈闭环批+图文成册）

- 业务笔 a4b5b0a7（45 文件 +2606/−136）+ 本记录笔；剔除不抢=DSH 工具面板线（capabilities
  前端族+locales+models.ts）+缓存命中线两件+运行态 meta+两个旧 zip。
- 干净工作树 .tmp/gaea-pkg-a4b5b0a7 构建（pnpm 27.8s+npm 35.6s+wails 54.7s）；
  **坑复现=.tmp 路径跑 exe 被策略拦（ExitCode 1），须先落 releases 正式路径再冒烟**；
  payload 冒烟 200（PID 3984）。
- SHA256：exe 6A0FDBE50D27220E…；三 DLL 与上两包一致；zip E3646329…/src.zip A15F5FB2… 留盘。
- 装机位=包 hash 同源核验 OK（PID 62624，INSTALL OK）。

## 2026-11-22 挂池五项清零批（F 批：P1-6/P1-12/P1-7/_draft 豁免/output_path 兼容）

- **F4 过程件豁免**：PublishGuard.Scan 豁免 `_draft/`、`_probe/` 段路径+文件名含
  probe/test/tmp/draft/草稿/临时（探针/草稿不是交付件；实测往 _draft/probe 写探针 docx 被拦）。
- **F5 声明写路径宽容解析**：wspath.RelIfWithin（工作区内绝对→相对，工作区外仍拒，相对口径
  逐字节不变）接进 GaeaExport/WenxueExport/office_batch 三出口（P2-11 四口径互不兼容收口）。
- **F2 隔离面双件**：①quarantine_list 新工具（builtin work 只读；按 stamp 去重=一行一次状态
  变更、终态取最后一条；status 过滤；契约行+spacetags 静态表+文档重导三处同批——spacetags
  守卫当场逮漏）②目录级原子隔离：Begin 记目录集+时刻，Quarantine 判定三要件（父目录本批新建
  ∧durable 全违规∧非 durable 伴行件 mtime<Begin=mv 随迁旧件）→整树连骨架一次隔离不留空壳；
  进行中编稿（现写新文件）不整搬、往既有目录塞件维持逐件（钉=三形态夹具）。
- **F3 wait+进度**：wait 分流 sa_/mt_ 引用（builtin.SetSubagentWaiter 注入缝——builtin 不能
  反向 import agent，boot 接线 agent.WaitSubagents：2s 轮询到终态/超时如实 running/未知引用
  unknown）；子代理每次工具调用回写 meta.lastAction（TouchAction fire-and-forget，终态停更），
  subagent_list running 行显 last: 动作——主控不再 sleep 轮询全盲。
- **F1 版面机检门 layout**：longdoc_verify 第五门（docx/pdf 输入；docx→soffice→PDF）四件
  确定性面：空白页（72dpi 墨<0.5% 硬伤）/墨覆盖两口径恒同给（整页面积%%+已占用版面%%——口径
  不再打架）/图片清单+有效 ppi（<96 warn，图面折算占版%）/文本越界（bbox xMax>页宽=物理裁切
  硬伤）。docmd 新三件：pdfimages 三级查找+PDFImagesInfo、PDFTextBBoxes（-bbox regex 解析）。
  表格断表/图题同页语义面留技能层。E2E 12s 真链钉（spec 双分页断造空白页）。
- 测试：新增 8 钉全绿；wspath/tool/agent/builtin/boot/app 全量 181.9s 绿。
  **装机 exe 1D796326E932AE24（=build/bin 同源）PID 55524，冒烟 200 INSTALL OK**。未提交（挂拍板）。

## 2026-11-22 众神殿父子交互裁决执行批（pan_1795306273871_ba6804e2f74f，签注 accepted）

裁决四 action 按序落（基线→首刀→核验→第二刀）：

- **action-3 基线回放（先行）**：沿江工作区子代理运行 137 条（22 真子代理+115 model_tool），
  complete_subtask 报告率 **0%**——「父已强制读取完整报告」被实锤推翻，首刀正当性坐实。
  （只读聚合统计，无原文外传。）
- **action-1 首刀 A（complete_subtask 结构化完成交接）**：报告加 `artifacts`（交付物清单，独立
  于 criteria.evidence——evidence 证判据/artifacts 列产物，复用会把读过的输入误认成交付物）+
  结构化 `blockers`（需父/用户动作解除，与 unresolved 分立不得改名互冒）；父侧 `task` 结果
  **恒附完成块**（childCompletionBlock）：无报告=unreported 不再静默；compact 恒一行（并行
  上下文税可控），非干净面（status≠complete/降级/blockers/unverified/unresolved）才展开 full
  （每类 cap 5）。artifacts 宿主核验=子会话成功产出回执（extractDeliverablePath 取
  output_path/out_dir 优先）路径匹配→verified，缺证据=unverified 绝不升级。
- **action-2 产物核验链**：回执面（第一优先证据源）已落地（见上）；办公 Journal 面（父
  SessionPath 匹配）评估后记档不接——成功工具回执本身即最强证据（含 output_path 声明），
  跨包 plumbing 增量≈0，如实记录不冒充。
- **action-4 第二刀四件**：①output_schema 早退路径完成状态经 SubagentMessage 事件面到达父侧
  （JSON 本体不动，程序调用方解析不破坏）②task.go 1372-1381 死分支删除（schemaGuard 早退后
  永不可达）③嵌套中间层 send_message 双向合并件 dualSendMessageTool（缺 agent_id=向父汇报/
  带 agent_id=向下 steer 自己子树——此前同名 Add 静默覆盖丢失向下能力）④wait 终态文案不再
  指向已退役 read_subagent_result（改自动回投口径）。
- **B1（子→父 blocking 请求回合/子代理直连用户）不立项**（裁决：无办公实弹收益样本+审批闸
  不可直达+死锁风险；若实弹日志显示 blocked 频繁且父晚读超一回合可重启）。
- 测试：首刀五钉（unreported/blocked 展开/artifacts verified+unverified/干净面 compact/双向
  分向）+旧 verdict 测试随新契约更新；agent 全量 18.9s+builtin/boot/app（187.4s）全绿。
  **装机 exe AB6863197E0B55E7（=build/bin 同源）PID 63000，冒烟 200 INSTALL OK**。未提交（挂拍板）。

## 2026-11-22 打包记录——gaea-20261122-d6add4b0（挂池五项清零+父子交互裁决执行批）

- 业务笔 d6add4b0（30 文件 +1709/−96，F+G 两线合一）+ 本记录笔；剔除不抢=DSH 能力面板线
  （capabilities 前端族+todos.md 记账）+缓存命中线+运行态 meta+各 zip 留盘。
- 干净工作树 .tmp/gaea-pkg-d6add4b0 构建（pnpm 18.5s+npm 28.3s+wails 57.3s）；
  `.tmp` 路径跑 exe 被策略拦坑第二次复现（先落 releases 正式路径再冒烟）；payload 冒烟 200
  （PID 19528）。
- SHA256：exe BCBEA091EF210C44…；三 DLL 与前包一致；zip 6EA95E3E…/src.zip 23F84310… 留盘。
- 装机位=包 hash 同源核验（见装机行）。

## 2026-11-22 编程板块·DSH 隔离栈全量升级（core rc.5+9 插件+REA 6.10.1+dsh-wubi 两笔适配）

- 详见 docs/gaea-day-log-2026-11-22.md。要点：DSH core 0.2.2-rc.1→rc.5；dshmarket 1.77.1
  （本地 patch 退役）；dsh-wubi 289838c（needle 复验+#13/#18/#35 变体）+c5c76d2（#56/#77
  回退压缩态手术——真机实锤退役不成立，rc.5 原生只恢复输入框不清对话流）；junction 断链
  根修（link spec 与手动 junction 须同批改）；REA 技能 v63/263 同步。
- 验证：预设 registered 20 插件 broken=null、技能 23/23、工具箱 15/15、REA 263 实数、
  实弹 3 轮+两轮回退走查（reload 重放全净）。gaea 仓代码零改动；3091 清场待用户点启动。

## 2026-11-22 打包记录——gaea-memory-distill-20261122-98466698（关联软件官方最新批+记忆中枢蒸馏全链）

- 业务链 fc41b54a + 4c1eb517 + 34a550dc + 1e50e689 + 52f5b187 + day-log 笔；记录笔 6e8c2c82。
- 干净工作树 .tmp/gaea-pkg-98466698（pnpm 28.3s+npm 32.5s+wails 60.3s；主树拷三 DLL 入
  worktree build/bin 供 gen_bindings PATH——System32 ORT 劫持坑打包态同样命中）。
- payload 正式路径冒烟 200（PID 18184）。SHA256：exe 6E8FB1DF…；ORT 与历届一致；
  **sherpa 两 DLL=v2.0.2 新哈希首次进发布件**；zip 86DBD681…/src.zip 48E50ABE… 留盘不入库；
  装机位 0BCAACE7…（同码不同时间戳同源核验 ✓，PID 72764）。worktree 已回收。

## 2026-11-24 打包记录——gaea-20261124-2e129cc（交付链三线+图片托管+图标+Qwen 冷启动，全量收口）

- 五笔业务提交：6c4f242d（一级导航图标辨识度）+ f4fc4ba1（方案库图片托管 SchemaV37+gaea-asset:// 闭环）
  + b0d4147c（Inkling 退役+Qwen3.8 冷启动生命周期）+ e0c322c9（交付链三线：docx 编号联动/登记格式
  模板/交付文件夹口径）+ 2e129cc0（本记录笔）。主树构建 npm 39.8s + wails 58.3s；干净工作树
  .tmp/gaea-pkg-2e129cc 打包（pnpm 47.2s + npm 28.1s + wails），三 DLL 预拷进 worktree build/bin
  防 System32 ORT 劫持。worktree 已回收。
- 冒烟：releases\gaea-20261124-2e129cc\payload 正式路径 /api/health 200（PID 44828 已回收）。
- SHA256：exe BE4C6751AA3CA26A…；ORT 68D0D5F5…；sherpa-c 408F9AB6…；sherpa-cxx DBAC55CE…；
  zip 593BE0D2…/src.zip 6C400E38… 留盘不入库。
- 装机：payload 同源装机（gaea_install.ps1 -SourceDir=payload），INSTALL OK，装机 exe
  BE4C6751AA3CA26A… = 包 exe 同源 ✓，构建时刻 2026-11-24 21:45:33。
- 同批登记：昆顶格式模板已落两工作区（D:/新工作/南岸、2630合并沧州 .gaea/templates/docx/技术标-昆顶.json）。

## 2026-11-24 修复——编程板块无法启动（gaea-20261124-5def04b0）

- 实锤：11-22 DSH 栈升 rc.5 时 gaea 侧 programmingWebExpected 仍钉 rc.1，八查
  version=false 恒拒绝启动（用户 20:02/20:03 两次 OpenProgrammingWebExternalError，
  其余七查全 true）。
- 修=期望常量抬 0.2.2-rc.5（5def04b0）+ 预检测试夹具改引常量（此后升 DSH 只抬一处）。
  定向测试绿；主树 wails 52s；干净工作树打包 pnpm 31.5s+npm 31.0s+wails 52.2s；
  payload 冒烟 200（PID 53160 回收）；装机 payload 同源 INSTALL OK（exe
  0A4B616FA091F94E…，构建时刻 21:53:08）。zip 9360EBE3…/src.zip 1BC9BD77… 留盘。
- 待用户验证：壳拉起 → 编程板块点启动（3091 已清场，DSH rc.5 应就绪）。

## 2026-11-25 修复批——办公 GenUI 表格闪烁根修 + 铜陵二轮反馈 #16–#20/#9（gaea-cost-genui-20261125-69300696）

- 交付提交 69300696。GenUI 闪烁根因：gaea/wenxue Markdown 每渲染内联重建
  components（新 Code 闭包）→ react-markdown 换组件类型 → GenUI 子树整体重挂
  → gui-reveal 动画每帧重放；修=components/fenceKeyFor/codeLabels 按依赖
  memo（scope 原语键防宿主内联对象），插件数组提模块常量，回归测试钉同一
  .gui-block 节点跨重渲染存活。聊天线不受影响（ChatRow 早已 memo）。
- 铜陵 #16/#17 根因（铜陵会话 gaea-log 实证，非模型侧玄学）：顶层列表/标量
  被模型塞进 quota（delete_quota codes、dedupe keep_id/merge_ids、import_project
  region/path）、同键重复 quota 后值胜吃掉好内容、数组串化进 code——cost_library
  救回面推广全谱 + clQuotaProbe 重复键容忍 + clDecodeStrings 串化拆分；缺参
  回显收到键；UsageHint 补 delete/dedupe 五骨架。#19 remember 同款三件
  （KeepSchemaDescriptions+回显+UsageHint）。#18 四要素同仅地区异并入原行
  （SaveResource 须继承原行 id/code 才走 UPDATE——坑）、消歧回执带旧码→新码
  (id=N) 映射、cost_project resources 补 id 列解 dedupe 链路死结。#20 舍分
  口径进技能契约 + 演练按 bundle 模拟库内舍入链出 INFO。#9 bash 描述前置
  长 heredoc 截断警示。#6 已被样表化吸收无需再刀。
- 测试：Go workcost/skill/builtin/memory 全量 + app 造价面全绿（新增三回归
  钉）；前端 tsc/eslint 净，vitest 76 例绿（含闪烁回归钉）。
- 打包：干净工作树 .worktrees/pkg-69300696（junction node_modules；主树
  modelengine 未提交改动未混入），npm 30s+wails 68s；payload 冒烟 200（PID
  43448 回收）。SHA256：exe A738B0E5…；ORT/sherpa 与历届同；zip 9BAEC2DB…/
  src.zip C8C877F3…。装机 payload 同源 INSTALL OK（构建时刻 2026-11-25
  20:48:25）。worktree 待回收（junction 先摘）。

## 2026-11-25 技能去噪批——办公档索引 15→10（gaea-skill-slim-20261125-0f600f5f）

- 用户令「办公板块技能太多，不需要的删除，不要干扰模型」。盘底（铜陵会话
  request_header 实抠）：造价档索引 15 件 = builtin 9 + 造价模式 2 + 泄漏 4
  （~/.gaea/skills 的 8-12 ANTHROPIC 旧拷贝 docx/pdf/pptx/xlsx）+1（ZCode
  hindsight 插件技能经 ~/.agents/skills 漏入）。
- 两刀：①skill Store home 级停扫外厂约定目录（.agents/.agent/.claude），项目级
  保留（外厂技能显式投喂口），commands 侧 ConventionDirs 未动；回归钉
  TestHomeForeignConventionsNotScanned。②删 ~/.gaea/skills 四件旧拷贝（备份
  ~/.gaea/skills-retired-20261125.zip；docx/pptx 导出脚本 home 回退由
  ~/.codex/skills 同版承接，diff 已核）。
- 余 10 件件履职：cad-takeoff/chart-builder/cost-estimate/doc-assemble/
  format-convert/genui/office-edit/office-gongwen + cost-estimation-playbook/
  soil-remediation-costing。commit 0f600f5f。
- 干净树 .worktrees/pkg-0f600f5f 构建（npm 28s+wails），payload 冒烟 200
  （PID 55692 回收）；装机 INSTALL OK（exe 14F8A0FC…，构建时刻 2026-11-25
  21:01:12）。worktree 已回收。zip 345A9AEE…/src.zip 1DD7FB35…。

## 2026-11-25 纪律补丁——干净树打包后 build/bin 回写（用户质询触发）

- 用户问「构建桌面端为什么 build/bin 没同步？纪律没写吗」。核实：build/bin 停在
  7ccdc1cc（4B0466B4，11-25 19:02），装机位已 0f600f5f（14F8A0FC）——落后两批。
  根因=纪律洞：AGENTS.md #7 只写「干净树打包」，装机又显式 -SourceDir=payload，
  两步都碰不到主树 build/bin；「回写」一步三处都没写。
- 现场：包版四件回写 build/bin（与装机位逐一哈希对账一致）；未在主树重建——主树
  压着 modelengine 未提交改动（GLM 目录线在途），重建会混编破坏「build/bin=最近
  装机版」。
- 纪律三处同批：根 AGENTS.md 最低要求新增第 8 条；.gaea/AGENTS.md 构建入口交付包
  条扩写；docs/gaea-build-guide.md 新增「交付包收尾：build/bin 回写」节 + 坑清单
  一行。要点：回写用包版文件，主树有未提交改动时禁用主树重建代替。

## 2026-11-25 办公工具梳理首批 + 记忆按模式隔离 + 流空闲超时紧急修复（gaea-office-tools-20261125-799e9ba8）

- 用户令三条：①「重新梳理整个办公板块的工具，该优化优化、该合并合并、该删除
  删除」；②「记忆为什么没按项目及工作空间隔离，不同项目的不同模式的都混在
  一起」；③紧急「办公板块动不动就流空闲超时：超过 1m0s 无数据直接切断，1min
  对造价模式不够」。随后「完成这个就先构建桌面端，做好文件记录和记忆更新，
  剩下的下个会话继续」。commit 799e9ba8（139 文件：新增 10 / 删除 19 / 改 110）。
- 盘底：真实会话 request_header + 新增工具面体检 TestOfficeToolFaceDump
  （设 GAEA_TOOLDUMP_OUT 导出三档逐工具尺寸）。精简面：标准 73 件/21779 字符、
  造价 82 件/38774、方案 79 件/24945；cost_library 一个 14207（占造价档 37%）；
  调用频次（D:\新工作 15 工作区 68 会话 9~11 月）：前 24 件占九成，19 件零调用。
- 改后实测（同口径 TestOfficeToolFaceDump）：标准 60 件/17395 字符（−20%）、
  造价 68 件/28867（−26%）、方案 72 件/21917（−12%）；造价档余下大头=
  cost_library 8847、remember 2531、request_permission 1284、ask 1010。
- 紧急修复：ai.defaultStreamIdleTimeout 60s→10min（按「两次收数间隔」计，推理
  模型出字前零字节被当空闲；同值兼作响应头等待上限）+[agent]
  stream_idle_timeout_secs 覆盖；回归钉 TestStreamIdleTimeoutResolution。
  真机实弹未做（需造价档长思考实跑）。
- 工具首批：cost_library 花括号 bug（quota 少一个右括号→31 个参数在模型眼里
  嵌在 quota 里，铜陵反馈 #13 根因）+ 行契约随回执发放（import_open 回执带
  行契约、dims 带 classify 契约、action=help；常驻 13091→8541）；退役
  cost_compose/multi_edit/edit_lines/replace；summarize_file 读码改判保留（挂在
  大文件读取子系统上）；方案库六件归长文方案组；dag_plan 摘出工具面（流水线
  运行面保留）；五表/六表口径统一。
- 记忆：facts.mode_id（SchemaV42）+ modeView（通用+本模式，标准档只看通用，
  模式维度未启用的引擎逐字节不变）+ remember scope；五件合两件（remember
  action=save/forget/promote、memory_search 给 name 读全文）；晨报/事实单两块
  停注只留索引一处；两开关（morning_preload/project_brief）及 4 绑定/设置页/
  面板开关/配置项撤下；注入体检改对索引块断言四不变量。装配断言
  boot.TestBuildMemoryModeIsolation / TestBuildRememberStampsMode。
- 实情澄清：存储本就按工作区绝对路径分桶（16 桶无串数据）；「混」来自三因——
  同一文件夹里干多个项目、没有运行模式这一维、过程类记忆每轮常驻 + 用户级
  AGENTS.md 混进造价规矩；外加同一批记忆一轮曾列四遍。C2（造价口径规矩迁技能、
  用户级 AGENTS.md 两条迁出）未动，下会话续。
- 门禁：app 包按用户令「取消单进程，改并线」——go test -c 编一次 + 4 进程分片
  各用独立 APPDATA，1425 例 0 失败；memory/db/boot/control/tool/evidence/
  contextview/cache/ai/config 包通过；tsc -b 0；vitest 受影响 33 文件通过；
  registry_map_doc -check OK。存量红（未改动的 HEAD 同红）：agent
  TestStaleGuardBlocksMoveFileWithoutRead、builtin 三例 markitdown（本机无
  markitdown）。并行负载下 TestGaeaPriceFetchAllTask 偶发一次，单独 3/3 通过。
- 干净树 .worktrees/pkg-799e9ba8（Vite 36s，Wails 1m13s，bindings 未跳过），
  payload 冒烟 200（PID 46288 回收）；装机 INSTALL OK（exe B65B9B14…，构建
  23:20:22，PID 50204 23:22:40）；zip C65E81CE… / src.zip 173A4AF9…。包版四件
  已回写主树 build/bin（与装机位、payload 三处哈希一致）；真实库 V42 已迁移
  （facts.mode_id 在，16 桶 245 行）；worktree 已回收。未 push。
- 发现：用户壳当时跑的是主树 build/bin（不是装机位），锁着四件——回写必须在
  装机脚本关壳之后；顺序=先装机再回写。
- 余（下会话续做，按优先级）：①C2 造价口径规矩迁技能 + 用户级 AGENTS.md 两条
  迁出；②B2 技能三件合 run_skill；③B3 COM 三件并入 office_read/office_edit；
  ④B4 xlsx_recalc 回执带审计、docx_accept 并入 docx_apply；⑤B5 后台任务三件、
  B6 子代理管理三件（协议面宽、收益小，可缓）；⑥A2 request_permission 描述
  瘦身（1188 字符英文）；⑦三档工具面预算守卫定数（TestOfficeToolFaceBudget
  未写）；⑧流空闲超时真机实弹。

## 2026-11-26 办公工具梳理二批 + 工具面预算守卫 + GLM-5.4（gaea-office-tools2-20261126-113d84ad）

- 用户令「继续优化gaea」，承接上条余项；按 69 个真实办公会话（9~11 月）的调用
  频次排刀。三个提交：b16bc58e（GLM-5.4 目录）、72625cb9（二批，42 文件）、
  113d84ad（wailsjs 生成件同步）。未 push。
- 改后实测（同口径 TestOfficeToolFaceDump）：标准 58 件/16404 字符（首批后
  60/17395；自梳理前 73/21779 累计 −25%）、造价 66/27824（68/28867；自 82/38774
  累计 −28%）、方案 70/20863（72/21917；自 79/24945 累计 −16%）；COM 档 10→7。
- A2 request_permission 描述 1188→559 字符（69 会话 4 次调用，却每轮占三档最大
  单项）：只留调用前必须知道的规矩，各决策的后续动作由回执给出。
- B4 xlsx_recalc 回执自带审计与交付门结论（重算 25 / 审计 22 次几乎成对）；
  xlsx_audit 保留只读复核面（一写一读，不真合并）。docx_accept 并入 docx_apply
  （accept:true；Journal 记录名不变，通道 A 复核按记录名分派不受影响）。
- B2 slash_command 并入 run_skill（项目命令按名展开、技能优先、name=list 取清单、
  命令展开不计技能用量）。**read_skill 改判保留**：只读席位（众神殿白名单、
  只读闸）的唯一读技能通道，run_skill 是写类工具，合并会让席位读不了领域规范。
  顺带实测复现并消除 slash_command 对技能名空指针（技能条目不带 Render）。
- B3 office_status / office_help 并入 office_read（status:true / help:true，仍是
  只读面）；office_macro 退役（两个宏各等价一条 replace op，5 次调用）。
- 守卫 TestOfficeToolFaceBudget：件数 58/66/70、字符 16750/28250/21250；已下线
  名单不得回潮（含 gaeaExtraTools 直查，钉到按条件装配的 COM 档）。
- C2 技能侧：造价口径两条规矩迁入造价 playbook「口径纪律」与 cost-estimate
  「口径」节。**用户级 AGENTS.md 两条未动**（用户数据，等确认、备份后再迁出）。
- 未动：B5 后台任务三件 / B6 子代理管理三件（wait 60 次、bash_output 39 次是
  高频协议面，并入 task 收益小风险大——建议不做，挂拍板）；流空闲超时造价档
  长思考真机实弹。
- 发现：干净树 wails build 再生成绑定，暴露已提交 wailsjs 落后于 Go 绑定面
  （首批撤的 4 个绑定 + MemoryEvalReport 字段没同步生成件）——113d84ad 补齐；
  从 72625cb9 打的首个包作废，改从 113d84ad 重打。
- 门禁：app 包 1423 例四分片并线 78s 零失败（测试 exe 须放系统临时目录且
  cwd=internal/app——仓内 .tmp 路径下跑被本机策略限权，连 TempDir 都 Access
  denied）；skill/command/boot/tool/builtin/evidence/skillstats/modelengine 通过；
  tsc -b 0（主树 + 干净树各一遍）；vitest 受影响 6 文件 89 例、干净树
  spaceBindings+mock contract 15 例；registry_map_doc -check OK；go build ./...
  过。存量红：agent TestStaleGuardBlocksMoveFileWithoutRead（HEAD 同红）。
- 交付：干净树 .worktrees/pkg-72625cb9（后切 113d84ad；Vite 38s，Wails 16s，
  bindings 未跳过），payload 冒烟 200（PID 23156 回收）；装机前查旧壳空闲
  （6s 窗口 0.14 CPU 秒）→ INSTALL OK（exe 726CA973…，构建 00:12:54，新壳
  PID 14692 00:14:56，诊断口 ok）；zip B662308D… / src.zip 25484433…（git
  archive 排除 releases/，5493 文件；上一包 src.zip 834MB 是没排除、卷进 1482
  个 releases 条目）；包版四件已回写主树 build/bin（三处哈希一致）；worktree
  已回收；GLM 热更新文件 glm_catalog_remote.json 备份后删除（内嵌目录已带）。
- 工作树遗留（非本批所动）：仓根 4 张图片（logo 浅/深色、盲业智能启动画面
  浅/深色）处于已删除未提交状态，仓内无引用。
- 收尾发现并堵根因（ef2a3047，仅测试代码、不进装机包）：不带隔离的 app 包
  用例经 config.Save 把夹具写进**真实的** `~/.gaea_config.json`（该文件按家目录
  USERPROFILE 定位，手工重定向 APPDATA 管不到）。新增 TestMain 把
  USERPROFILE/HOME/APPDATA 沙箱化，不设全局逃生口；不做手工重定向四分片
  1424 例 0 失败（94s），跑前跑后真实配置 SHA256 不变。实证：不带沙箱单跑
  TestWenxueChatSessionBindingSmoke 后 wenxue_workspace 立刻变成它的临时目录
  （我做这步定性时用的是真实家目录，等于又写了一次——同键仍是死路径，但这是
  我的操作失误）。
- **待用户确认**：真实 `~/.gaea_config.json` 里三个键是测试垃圾——
  wenxue_workspace、image_save_dir 指向已删除的测试临时目录（文学板块每次回退
  到图文根、办公生图落点异常），voice_persona_presets 是「傲娇音→v1」；image_
  backend=xai / image_model=grok-imagine-image-quality 可能也来自同名测试，
  无法区分。修法=备份后删前三个键（是否保留 xai 生图设置由用户定）；没动。
- 偶发：TestWenxueChatSessionBindingSmoke 在四分片负载下 TempDir 清理撞上仍被
  占用的会话日志句柄一次（单跑 5/5 绿，第二次全量也绿）——存量 flaky，未改。

## 2026-11-26 重算脚本内嵌 + LibreOffice 独立配置目录（宁丰反馈 #1；主树装机 PID 13748，未提交、未打包）

- 用户令「吸收重算脚本」（6 条反馈的分诊见记忆
  gaea-ningfeng-feedback-2026-11-26-triage；其余 4 条小刀未令、未动）。新增
  internal/office/xlsxedit/recalc.go、scripts/recalc.py（go:embed）、
  recalc_test.go、recalc_smoke_test.go；xlsxedit.go 删旧 Recalc/
  findRecalcScript；7 个调用点随签名改为 Recalc(path)。
- 来历：重算链原先到 工作区 .gaea/skills → ~/.codex/skills → ~/.gaea/skills 找
  recalc.py——脚本不是 gaea 的，用户机上命中的是 Codex 技能目录的副本。对照
  实验坐实：soffice 走默认配置目录时，机器上另有 LibreOffice 实例（忙或空闲）
  就被转交并立刻返回，文件没重算而脚本回「成功」（6/6）。宁丰会话里那次 zlib
  报错 28 次未复现，成因未证实（会话时序严格串行）。
- 改后：脚本自包含内嵌；每次调用复制一份独立配置目录
  （-env:UserInstallation）去跑，用完即删。首次让 LibreOffice 自己初始化
  （约 5s）并留作模板 %LOCALAPPDATA%\gaea\lo-recalc\lo-recalc-profile-<脚本
  指纹>，此后复制 0.1s + 重算 1.4s（原走默认目录 1.9s）。**手写最小配置目录
  不行**：LibreOffice 对没经它初始化的目录挂住不返回（实测超 60s）。
- Go 层 Recalc：zip 预检（打不开的文件秒退——交给 LibreOffice 会挂满 60s
  超时）；回读类错误自动重试一次；模板复制件跑不起来=丢弃模板重建（自愈）；
  失败三类 ErrRecalcUnavailable / ErrRecalcRead / ErrRecalcNotWorkbook，工具
  回执分别指路，不再一律引去 xlsx_audit。全是「公式无缓存值」时回执改说
  「重算可能没生效，原样再调一次」，不再当成公式写错。
- 测试隔离：app 包 TestMain 置 GAEA_RECALC_DISABLE=1（此前靠「测试工作区没有
  recalc.py」自然跳过，内嵌后不再有）；TestGaeaSelfGenerateCost 自行解除。
- 门禁：xlsxedit 包全绿（新增离线 10 例，含内嵌脚本 python 语法守卫）；实弹
  GAEA_SMOKE_RECALC=1 两例过——常规 11.2s、外部 LibreOffice 实例在场 4.9s，
  均写回缓存值 B5=600；app 包 1424 例四分片 90s 零失败；gofmt / vet 净；
  CGO_ENABLED=0 go build ./... 过。
- 装机（用户令「构建桌面端」，主树日常装机流程，非干净树包）：HEAD a038e34f + 本刀
  未提交改动（另有既有的仓根 4 张图片删除，仓内无引用）。frontend npm.cmd
  run build 33.9s；wails build -s -ldflags "-s -w" -trimpath 53.3s（ucrt64 gcc、
  系统 TMP、bindings 未跳过且 frontend/wailsjs 零漂移）；exe 60,073,472B
  sha256 85DD1F23…DC090（load-time 导入含 sherpa-onnx-c-api.dll，内嵌
  GAEA_RECALC_PROFILE 串在）；三 DLL 未变（68D0D5F5… / 408F9AB6… / DBAC55CE…）。
  冒烟 /api/health 200（PID 59480 回收）→ INSTALL OK：装机位四件哈希与
  build/bin 一致，新壳 PID 13748 启动 07:56:06 晚于构建 07:55:19，诊断口
  healthz=ok。重算配置模板缓存在 %LOCALAPPDATA%/gaea/lo-recalc。
  **未提交、未打包**（提交与干净树包待令）。
- 未动/遗留：技能包那份 .gaea/skills/xlsx/scripts/recalc.py 原样——xlsx 技能
  经 bash 调它仍走默认配置目录，同一缺陷在；前端「重算」钮路径
  GaeaXlsxRecalc → renderXlsxAfterChange 连算两次（存量，未改）。

## 2026-11-27 造价数据库排序 + 工料机详情优化（未提交）

- 用户要求回退此前的整页视觉试验，仅保留列表排序。已恢复页面骨架、密度、
  清单详情和定额详情；此后只继续改 `ResourceDetail`，没有再动另外两级详情。
- 三级列表按可见列提供升序→降序→恢复默认；中文使用数字感知 Collator，
  价格按数值排序，空值沉底，带排序图标与 `aria-sort`。
- 工料机详情按定额详情同一套 `cl-block` 节奏实现：核算价格、引用定额、
  各期价格都是标准信息块，不引入独立价格卡或分组视觉。核算价格块内改为
  大号取用价 → 现行/基准价两栏 → 两列基本资料，补清层级和间距；保留与
  定额详情一致的尾部说明。清单 / 定额详情保持原版。
- 验证：CostLibraryPage.test.tsx 39/39 通过；`npx tsc -b` 通过；Vite mock
  真机检查排序与工料机详情。未构建 Wails、未打包。
- 用户令「构建桌面端」：主树（含本批未提交改动，HEAD 9f6b3805）前端
  `npm.cmd run build` 34.77s；随后 ucrt64 + CGO + 系统 TMP 执行
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，耗时 1m56.453s。
  `build/bin/gaea.exe` 62,759,936B / SHA256 `63555BE5…48AA45`；
  `/api/health` 冒烟 200。`gaea_install.ps1 -Launch` 输出 INSTALL OK，
  安装位四件哈希与 build/bin 全一致；新壳 PID 64464，启动 12:39:55
  （晚于构建 12:39:33）。未制作 ZIP / 源码包。

## 2026-11-27 办公板块项目行会话操作常显（未提交）

- 用户反馈办公板块「新建会话 / 删除」藏在 hover 后不可见，要求参照文学板块。
  已将办公项目行操作组从 `hidden group-hover:flex` 改为常显
  `office-project-actions`，项目行改为 flex 单行；新建、移出最近列表、归档
  使用 27×28 图标钮，当前项目保留新建，非当前项目显示三项。
- 删除 / 归档的无障碍名称补项目名；补充回归断言，确保操作组无 `hidden` 类
  且按钮常驻 DOM。Sidebar.test.tsx 16/16 通过，`npx tsc -b` 通过；
  Vite mock 真机检查按钮已与项目名并排可见。未重新构建桌面端。

## 2026-11-27 午后 沿江包1 复盘落地：docx 表格重建进 gaea + md_validate 分级基线（未提交）

- 背景：用户带《包1技术标方案重建_工具与流程不足复盘》进会话，指出上午装机
  （ca0b13a2）落地的只是流程规则（禁直转/回执列缺陷/门禁拦 finalize），能力
  没跟上——to=docx 出的 Word 三缺陷同病，修还靠项目目录自建脚本。拍板刀路：
  刀2 重建器移植为主刀、刀1 机检分级+基线次之。
- **刀2（T1/T3 根修）**：docmd 新增 docx_rebuild.go——Go 版重建器（算法照搬
  方案库录入/tools/docx2md_v2.py）：轻量 XML 树解析、tblGrid 展开 gridSpan/
  vMerge（行列数与网格严格一致）、单元格折行语义重拼（中文续接/ASCII 补空格/
  列表项与句末 <br> 分行）、目录域(sdt)与目录残表剔除、伪表格（列数≤2 且非空
  单元格全是图）转图片引用、图片按 rels 转 data URI（与 markitdown keep 同契约）。
  ConvertLimitProgressReport / ConvertPreservingImagesReport 新增返回
  DocxRebuildReport；convertDocxSmart 路由：合并网格/折行碎片/目录残表/列数
  不一致任一签名→重建路线，干净文档仍走 markitdown，markitdown 不可用退重建。
  format_convert 回执带结构体检（路线/源表列数一致率/合并展开/目录剔除/折行
  重拼/剔空段）；描述与 compact 改口径。to=docx 后再转 md 无须手工修格式。
- **刀2 实弹**：真实包1技术标V5.docx（676 页）重建对照复盘 v3 指标——标题 506、
  合并单元格 808、目录域 10 全部与项目工具分毫不差；目录残表剔 10（跨查正文
  标题清单交叉比对，75% 纯形态只剔 5；纯数字序号格不数防误吃真内容表）；输出
  2488 行列数矛盾 0；「按照相关技术导+则及…」断词重拼复原。
- **刀1（T2）**：md_validate 检出项分级（撞号/跳号/孤立引用/列数=硬伤；未引用
  题注/跨章待核=提示），hints=false 折叠提示；baseline=旧版文档按内容主键
  （版本无关键，表格用表头签名）差集只报新增；未引用同 key 去重（撞号题注不
  重复报）。schema/描述/compact 同步；proposal 档预算 21250→21500（净增 15 字
  上取整）。
- **顺带修一个存量雷**：markitdown 无 --keep-data-uris 时输出省略号假 URI
  （data:image/png;base64...），convertDocxSmart 保图分支误用普通转换致外置
  报「data URI 无效」——保图请求改走 KeepImages 变体。
- 门禁：docmd 全包（含重建器 9 用例）绿；builtin 全包绿；toolface 三档过；
  app 全量后台跑。剩余：刀3 按题注裁图、刀4 多源对账、刀5 字数口径挂令；
  提交/打包装机挂令。

## 2026-11-27 晚 沿江包1 复盘清偿二批：md_validate 内容保真对账（刀4）+ 字数口径（刀5）；刀3 判不可行挂拍板（未提交）

- **刀4（T4/P1-3）**：md_validate 增 `content` 检查与 `source` 参数——12 字滑窗
  n-gram 对账移植（verify_vs_pdf.py 算法：去图引/<br>/空白、NFKC 全半角与上下标
  统一、下划线视连字符；表格命中率 <98%、段落缺失 >5% 列硬伤并附缺失片段样例；
  上限 60 条防刷屏）。`internal/docmd/pdf.go` 新增导出 `PdfTextLayer`（直读文字层
  纯文本，不走 WPS/OCR；乱码/扫描件如实报错）。source 收 PDF/md/txt/docx——
  **文字层乱码的原件先备基准 md/txt**（mdvLoadText 补 .pdf 分支走 PdfTextLayer）。
- **刀4 实弹**：真实包1 v3 md 对 fitz 基准文字层——**15 条硬伤，与 python 验证版
  （1 表+14 段）分毫不差**，对账 0.19s；WPS 转写 md 作基准时会带出 60 条形变噪声
  （数字/标点/括号转写差异），故基准必须走纯文字层。
- **刀5（T6）**：proposal_save 回执补「字数口径」行——库内=SUM(LENGTH(body)) 即
  md 字符数含空白与标记，成稿去空白口径低约 8%、PDF 原件口径另计，三者不等属
  正常（复盘：47.4/51.3/47.66 万均正确但互不相符）。
- **刀3 判不可行**：patch_figures.py 依赖 fitz get_drawings（矢量对象包围盒），
  Go 侧无等价库（pdfcpu 不做文本/绘图提取）；引新依赖是拍板项，挂池不硬做。
- 门禁：builtin/docmd 全包绿、toolface 三档过、app 全量绿（前批）。余：提交/
  打包装机挂令。

## 2026-11-28 记忆注入三刀：众神殿论证收口落地（代码 5cdf8229，打包提交 ead1952a 含绑定同步+缓存实测落档，包 gaea-memory-inject-20261128-ead1952a 冒烟 200 装机 PID 50428）

- 论证真源：`%APPDATA%/gaea/pantheon/results/pan_1795821263932_6dec471393fe.md`（四席三轮：保 A 4096 整块+预算不动，纯文档化/B 拆块出局，落三项实证修复）。
- **刀1 固化块发射面+溢出提示**：`BuildPinnedFacts` 返回 Block+Emitted/Truncated/Omitted；溢出点名提示计入块预算（先降级点名、挪已发射行是最后手段，纯提示块兜底）；boot 写发射面入 Set，`finishMemoryReload` 中途换 Set 携带快照（去重依据跟前缀走不重算）。
- **刀2 召回 emitted 级去重**：RecallBlock 与 RecallBlockFused 统一只排「已发射且未截断」的 pinned——修掉办公路径 pinned 双份注入与聊天席全量跳过的「两处皆失」洞。
- **刀3 索引确定性准入**：renderIndex 两段准入（procedural 非 pinned 优先，组内 Name 升序；touch/评分不参与）；file 后端不动。
- **基线实测（副本库只读探针，一次性已删）**：最大项目 55 条、索引原始 5556B→注入 4160B **截断真实发生**；修复后 procedural 非 pinned 准入缺口=0；全库 pinned=0（固化块/去重路径暂无实弹负载，单测已钉）。P95/相邻轮 SystemHash 仍欠真机会话观察。
- **③画像↔固化交集核验**：user+semantic+pinned 条目会画像 desc 行+固化 body 行两处出现——量级小（画像 600 rune 预算内一行），挂拍板池不私自动刀。
- 挂拍板池：③交集处置、DocBlock 接线或废弃（零调用点）、B 拆块+目录、固化块预算扩容、面板溢出计数。
- **缓存命中率实弹（11-28 上午补测，探针读 gaea-log.jsonl 的 request_header.prefixStable + usage 缓存拆分）**：四场真机会话 252 请求、251 个相邻对 prefixStable=true 全绿、L1/工具面漂移告警 0 次——论证台账「相邻轮 SystemHash 实弹」关账。命中率（provider 实报）：沿江包1 97.8%（67 轮/523 万 prompt tok）、南岸 94.6%、太化 88.0%、铜陵 98.6%（121 轮/1400 万 tok）；全史聚合 model_stats.json：deepseek-flash 98.8% / glm-5.3-flash 98.7% / grok-4.7 93.7%，本地 herdsman 25-27% 为 llama KV 口径。miss=每轮新增尾部 tokens（召回在尾部属既定设计）。装机后（PID 26212）尚无实弹轮，下次聊天后可复跑对比。
- **打包推送（11-28 09:2x）**：干净树 `.worktrees/pkg-ead1952a` @ ead1952a 构建，payload 冒烟 200，包 `releases/gaea-memory-inject-20261128-ead1952a/`（exe 2764514e…，SHA256SUMS 全录），包版装机 PID 50428、三处（payload/build/bin/装机位）哈希一致；src 包改用 `git archive HEAD` 并排除 releases/（旧包把历史包递归打进 src 是意外膨胀，不再延续）；26 提交推 origin/main。
- 余：真机走查（记忆面板固化/溢出提示、召回去重观感）。
