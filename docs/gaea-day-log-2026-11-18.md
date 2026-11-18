# gaea Day Log · 2026-11-18 · 办公与文学两份独立系统提示词落地（众神殿 v1.0 冻结包）

> 众神殿 pan_1795007052552（续议 pan_1795003767314）四轮论证冻结 v1.0 施工单：
> 「四件常量＋workbench 三行＋显式板块缝」。本批照单施工——文本原样冻结
> （DeepSeek r1 附录 A 稿 ×2 + 第 4 轮 DSH/Codex 同文三句路径终版），运行时
> 快照失败只修装配、不重开文本。

## 1. 刀路总览

- **办公身份常量**（`internal/gaea/config/config_paths.go` DefaultSystemPrompt）：
  替换为「办公 A」冻结稿——身份层瘦身为身份/范围/任务判断/事实纪律/边界五段，
  不再点名具体工具（53 行工具手册淘汰，能力说明单源=工具 schema；发现兜底=
  执行常量两行路由）；敏感文档「本地优先、数据不出机」句与确认制记忆保留。
- **办公执行常量增量**（`internal/gaea/agent/single_prompt.go`）：
  ①记忆句（原 :47）「主动用 remember 沉淀」→ 确认制「只在用户明确要求记住、
  或纠正某项事实并要求以后照做时调用」；②「办公领域质量检查点」段末新增
  「能力发现」两行路由（先看 schema→tool_search 检索→如实说明不猜工具名）；
  ③事实底座/落盘/签退三段原文保留——根段计数红线 `.gaea/exports/`×2、
  `.gaea/work/`×1 不动（S4 play 替换锚点+测试计数对象）。
- **文学身份常量**（同文件 DefaultWenxueSystemPrompt）：替换为「文学 A」冻结稿
  ——范围放宽到各类小说（长篇/中篇/短篇+大纲/人物/世界观/续写/改写/润色/
  评审/阅读检索），明确排除散文/剧本/诗歌/非小说评论与办公域；含先判断任务/
  作品上下文/创作主权与连续性/边界与记忆五段；workbench 指路句保留。
- **文学执行常量**（新文件 `internal/gaea/agent/wenxue_prompt.go`
  WenxueSingleModelPrompt）：独立成文（与办公常量零共用正文、运行期零字符串
  替换）——文学 B 清单分节（角色流程/调研/章节写作/文件落点三句/验证/签退/
  错误恢复/记忆确认制/数据边界）。路径三句=第 4 轮终版：过程文件
  `.gaea/work/<任务名>/` 字面量（work 锚：wenxue_* 工具族 SpaceTag 恒 work、
  write_file 自建父目录，work/mode-off 两态直接可执行）；导出件按宿主当前
  空间默认落点交付（不写固定导出根）；定稿仅 wenxue_final_write。
  **无空间渲染函数**——play 下 work 标签工具被 AllowsSpace 全滤，渲染是给
  不存在的工具指路的死代码（issue-9 裁决淘汰）。
- **workbench 三行+自称**（`internal/app/wenxue_skills.go`）：自称「长篇小说
  创作专属板块」→「小说创作工作台（长篇/中篇/短篇由书籍信息与用户要求决定）」；
  目录约定补过程文件行（`.gaea/work/<任务名>/`）；交付节补导出句（默认落点、
  不写进定稿/）；记忆句改确认制。
- **装配缝**（`internal/gaea/boot/boot.go` + `internal/app/wenxue_engine.go`）：
  Options 新增 `ExecBoard` 字段——`"wenxue"` 时执行纪律换
  WenxueSingleModelPrompt 且跳过 officeLive 段（COM 实时协同是办公域分工，
  图文无 office_ 工具）；中立交付段（deliveryPromptSection）两板块照常单源
  追加（用户拍板 q341：交付链接规则留装配层不复制进两稿）。图文引擎 Build
  传 `ExecBoard: "wenxue"`。

## 2. 验收合同（裁决终版五面，全部落测试）

1. 文学执行常量契约（`agent/wenxue_prompt_test.go`）：在场锚（work 字面量/
   wenxue 四工具/complete_step/确认制/不承诺数据不出机）+ 禁词（`.gaea/exports/`
   fact_* /format_convert/semantic_search/screen_capture/office_/「通用办公助手」
   /「办公规范」）+ 独立成文（办公独有段锚不得出现）。
2. 办公执行常量增量（同文件）：确认制（无「主动用 remember」）+ 能力发现
   四锚；既有 2/1 计数断言保持绿。
3. 两份身份稿冻结锚（`config/wenxue_prompt_test.go` 追加 +
   `config/system_prompt_test.go` 新建）：文学=各类小说/非小说排除/确认制，
   旧自称与办公工具名禁入；办公=五段锚+敏感文档句+确认制，工具手册句禁入。
4. workbench v1.0 正文锚（`app/wenxue_skills_test.go` 追加）：新自称/过程行/
   导出句/确认制在场；「长篇小说创作专属板块」与 `.gaea/exports/` 禁入。
5. executor 最终串快照（`boot/wenxue_exec_assembly_test.go`，mock Build +
   History 过滤 system 消息——ctrl.SystemPrompt() 只回 L1 身份，L2 执行纪律
   在 session）：ExecBoard="wenxue" → 图文常量在场、办公执行锚（事实底座/
   质量检查点/办公角色行/`.gaea/exports/`）与 officeLive 段不在场；空板块 →
   办公执行常量+v1.0 增量在场（现状逐字节）。快照取 space.mode=off 态
   （验收①口径：play 下工具全滤属装配洞另案，不用 play 快照判文本失败）。

## 3. 验证与装机

- `go build ./...` 绿；agent/config/boot/app 四包测试全绿
  （app 跑 Wenxue 面 19s）。
- 装机：`wails build -s -ldflags "-s -w" -trimpath`（ucrt64/CGO）→
  gaea_install.ps1 INSTALL OK（gaea.exe sha256 D7D02D6E662C…）→
  smoke.ps1 `/api/health` 200。

## 4. 余项（裁决明确另案，不阻塞本次文本）

- 办公 schema 正常/budget/deferred 三态实测（ActiveSchemas 导出+三类选择
  案例集成测试）——运行时验收，非提示词文本。
- 文学 play 装配洞：play 下 work 标签工具全被 AllowsSpace 过滤，需 SpaceTag/
  SessionDir/书内账本三处同改才谈渲染（独立装配缺陷台账）。
- `story-long-write` 网文门控（恒在描述 vs 非网文书）与全局
  `remember.Description()` 主动沉淀措辞——各自单独产品刀，不混本批。
- 前缀缓存：办公身份+执行常量与文学身份是计划内常量改动，一次性失效记档。

## 5. 余项四刀同日收口（用户拍板「一起完成」）

- **刀 B·文学 play 装配洞根修**：`Options.SpaceAnchor`（boot.go）——文学引擎
  传 `"work"` 锚定装配空间，全局 `session.space=play` 时基础读写与 wenxue_*
  （SpaceTag 恒 work）不再被 AllowsSpace 全滤；工具过滤/记忆读端/space_
  profiles/权限规则/S4 渲染单源跟随。办公/CLI/TUI 不传=现状逐字节。
  测试=play 全局下文学 census（write_file/read_file/ls 在场+板块缝仍生效）
  与对照组（办公链 play 下照常滤+play 根段渲染）。「文学真正进 play」仍是
  产品另案（SpaceTag/SessionDir/账本三处同改的前置未变）。
- **刀 A·办公 schema 三态验收**：`Controller.Schemas()` 新暴露面（可见面=
  Schemas()，hide/budget 已套用；注册面=Tools() 藏而可按名调用）。三态集成
  测试（office_schema_states_test.go）：正常态 vision 服务完整 compact 描述；
  budget=24 描述截断/整行省略；deferred=[vision]+search.enabled 下 vision 出
  首轮 schema、tool_search 进可见面、注册面保留；三态执行纪律恒带能力发现
  两行（兜底路由与三态正交）。**实测新知**：CompactDescriptor 是无条件替换
  ——模型恒见 compact 短描述，`Description()` 长描述是死输出（众神殿引用的
  「schema 里写着冷启动 20 秒」实际模型看不到，issue-4 兜底路由的必要性
  反而更强）。
- **刀 C·story-long-write 网文门控**：索引描述改「长篇网文写作指导（网文
  题材专用）……书籍信息未声明网文题材时不套用本技能指标」+正文头加适用域
  引用块。保持恒在装配（不破坏技能索引面），以自述边界与文学身份稿同款
  约束对齐。
- **刀 D·remember 确认制**：`memory/remember.go` 描述开头加确认制句
  （Only call when the user explicitly asks…do not proactively save），消除
  与三处提示词确认制的文字张力；确认卡机制本就在，此为描述口径对齐红线。
- **顺手修既有假红**：`gaea_subagents_test.go` 三例——11-18 会话隔离刀
  （`GaeaSubagentRuns` 按 meta.sessionPath 严格过滤）落库时未同步测试
  fixture（meta 未盖章→全滤成 total=0）。stash 干净基线复跑同红定性为 HEAD
  欠账，fixture 三处补 sessionPath 盖章销项。
- 验证：boot/memory/control/agent/config/app 六包全量绿（app 107.9s）；
  装机 exe sha256 `3FD36C5CEBB9`，冒烟 `/api/health` 200。

## 6. 众神殿席位上下文账本断链根修（晚间实弹 pan_1795007052552 定罪）

**症状**：议事时各 agent 节点「上下文账本」恒显「上下文未上报」。实锤（session.json
contributions.usage）：zcode/dsh 席数据完整；**codex 席 token 恒零值**（只有
engineSessionId）；**grok 席 usage 全空**。T15 文件席位同 grok 管道，同断。

**双根因**：
1. codex：`parseSeatRunStats` 只认 session/thread.started/status.step_end/result
   四形态——codex exec --json 的收尾是 `turn.completed`（本机 0.165 实锤
   `{"type":"turn.completed","usage":{"input_tokens":14098,"cached_input_tokens":
   13952,"output_tokens":2,…}}`），蛇形字段从未被接。
2. grok/T15：模型席位走 `runModelSeat`（内部 AgentRunner，返回纯文本）——
   `parseSeatRunStats` 是 CLI 席管道，模型席路径压根没有 usage 采集，seatResult.
   Usage 恒 nil。

**修法（竖切）**：
- pantheon 接口族三签名加 `*SeatUsage` 返回（GrokStreamingBackend/
  GrokToolStreamingBackend/GrokHistoryBackend/RoleStreamingBackend +
  agentAdapter.stream/streamWithTools/streamWithToolsSession + runModelSeat）；
  调用点 `seatResult{Usage: usage}` 进 contribution（落库/前端链已有，零改）。
- app 执行核 `runPantheonAgentWithPrompt`（grok+T15 共用）：sink 捕
  `event.Usage` 累计——steps=模型请求数，input=CacheMiss（缺省
  Prompt-CacheHit），cacheRead=CacheHit，output=Completion；与 CLI 席
  「未命中/命中」两栏同口径。
- seatguard：`turn.completed` case（input_tokens 含 cached，同 ZCode 口径
  归一；多 turn 逐事件累加）。
- 测试：grok agent 实测链断言（两轮 Usage 事件累计 900/13100/200/2 步——
  **坑：AgentRunner 只在 TotalTokens>0 才发 Usage 事件**，mock fixture 须带
  TotalTokens）；codex 断言从「保持零值」升级为归一账目（1287/12800/2/1）。
- 全量：pantheon 20.5s + app 110.8s 绿；装机 exe 22:42:25 构建，冒烟 200。

**验收路径**：下场议事看四席「上下文账本」——输入/缓存/步数四席全有数
（codex 从 turn.completed、grok/T15 从引擎事件、zcode/dsh 原链不动）。

## 7. 众神殿席位窗口三刀（用户实弹反馈：窗口内容简单/缺上下文页/观点不折叠）

- **轨迹 tab 合并**（办公轨迹 tab 同形态）：AgentDetailPanel 独立「工具调用」tab
  退役——工具调用并入「轨迹」时间线（`SeatTrajectoryItem` tool 项扩
  input/output/error 全文），记录行=折叠单行摘要（工具名+状态+预览），点击
  展开检查器看陈词 markdown / 工具入参·回执·错误；ToolTraceList+TOOL_TRACE_LIMIT
  随删。
- **上下文 tab 新增**（办公上下文 tab 对标）：tab 收口为 观点/轨迹/上下文
  三枚。上下文页=账本条（从头部移入，Ledger testid 让位 pantheon-agent-ledger）
  +会话信息卡（引擎会话全文/累计用时/缓存命中率）+逐轮账目表（轮次/阶段/
  新增 in/缓存读/输出/步数/用时+合计行；数据源=各轮 contribution.usage——
  §6 根修后四席有数）。
- **共享窗口观点默认折叠**：SharedAgentBody 已完成观点一律默认折叠（预览+
  展开全文），流式中恒展开；用户手动开合记忆不弹回。FOLDABLE_CHARS/
  FOLD_DEFAULT_CHARS/charCount 随删（失去消费方）。
- 测试：streaming 套件三处断言跟进（账本进 context tab/短发言也折叠/
  markdown 展开后验）+新增正向用例（工具行展开+逐轮表+合计）；83/83 绿。
- **顺手销既有红**：WorkbenchPrefsPanel 三例=73d1a906（11-18 子代理实弹批
  ③任务自动切编队默认翻回开）测试欠账——断言按新默认矩阵更新（任务=开），
  5/5 绿；全量 4509 仅余 0 红。
- 装机 exe 23:10:18 构建，冒烟 200。

## 8. 办公系统提示词「装机位不换新」返工（深夜用户定罪）

- **现象**：提示词分板批装机后办公板块系统提示词仍是旧土壤修复长稿。
- **根因**：装机 exe 已含新稿（二进制 grep 实证），但
  `%APPDATA%\gaea\config.toml` 持久化 `system_prompt`（旧土壤修复稿）按
  `ResolveSystemPrompt()` 优先级（file > 持久值 > 默认）盖住代码默认值。装机
  验收只查了二进制内容+冒烟，没查**生效值**；§6 白天 DSH 委派检测刀撞过同一条
  「config 旧专业幽灵段」却只是往里追加委派段（19:20 写盘）而非清根。
- **清障**：第 6 行 `system_prompt` 置空（置空优于删行：Load 从 Default 起底后
  merge 时 "" 回盖，后续设置保存不会把默认值重新烤回去）+重启 gaea；备份
  `config.toml.bak-20261118-promptfix`。
- **无损核验**：task 工具 CompactDescription §7.3 委派口径全量在（config 委派
  段退役不丢）；旧稿点名的 spec_judge/spec_query 已不存在于代码（旧稿一直在
  教模型猜不存在的工具名，正是 v1.0 明令禁止形态）。
- **结构洞挂拍板**：`Default()` 给 `Agent.SystemPrompt` 播种 → 任何
  `gaeaApplyCfg` 保存把默认值烤进 config 冻结，默认稿再改仍会被盖（本机已用
  "" 键盾牌，他机/新机仍在）。

## 9. 三连提交+收官包 gaea-board-prompts-20261119-cff32e23

- 提交（main，origin 未 push）：`838a58cd` 提示词分板 v1.0 + `f7323e13`
  众神殿 usage 竖切与窗口三刀（§6/§7 落笔）+ `cff32e23` codex hotspot-07
  docs。releases 旧 zip/tar 与 `.gaea/archive/` 运行时产物照旧留盘。
- 装机：23:30 主树构建 `791FE398`，`gaea_install.ps1 -Launch` INSTALL OK，
  PID 44496 启动时刻>构建时刻，冒烟 200（含三提交全部内容，同源树）。
- 收官包：`.tmp/gaea-pkg-cff32e23` 干净树（detached @ cff32e23）构建，
  payload 冒烟 200（PID 51468 回收）；exe `0F7102C8…`，三 DLL 与基线一致；
  ZIP `566451E9…`（29.6MB）、源码 zip `EDB743AC…`；详录
  `releases/gaea-board-prompts-20261119-cff32e23/PACKAGE-INFO.md`。
- **新坑（打包线）**：干净工作树全新 `npm install` 确定性触发
  `[vite:esbuild-transpile] remove %TEMP%\esbuild-*: Access is denied`
  （esbuild 0.25.12 同版中招；主树同刻重跑绿=依赖集差异判别实验定性）。根子=
  **repo 不含 package-lock.json（untracked），每次全新解析依赖集漂移**。处置=
  删工作树 node_modules 后 junction 主树 node_modules 即绿。建议 lockfile 入库。
- 顺手清 `internal/gaea/control/resumed.jsonl.meta` 测试残留（无跟踪夹具）。
