# 🌍 gaea — 多功能 AI 助手

> gaea（盖亚）——你的通用办公与日常 AI 伙伴：可靠、清晰、有温度。
> 对话、轻语、小说、绘梦、模型引擎、工程办公、微信助手，多模块共生，统一品牌 V1.0.0。

![gaea](frontend/public/favicon.svg)

## 功能

- 💬 **对话** — 通用 AI 对话，流式输出，话题管理，本地存储
- 🌸 **轻语** — 可定制人格的陪伴型 AI（29 人格模板 + 情绪融合 + LLM 记忆管线）
- 📚 **小说创作** — 世界观/角色/大纲 Agent，章节流式创作，一键导出 TXT/MD/EPUB
- 🎨 **绘梦** — 文生图（ComfyUI：Flux / Z-Image-Turbo / Krea2），模型后端可切换
- ⚙️ **模型引擎** — 多供应商模型中心（xAI / DeepSeek / MiMo），OAuth 一键登录
- 🏗️ **通用办公** — 轻量工具集（17 个内置核心工具 + 技能扩展）：文档转换/图表生成/文档拼装、
  记忆与知识库、方案编写；开工前计划卡片 + 资料一键引用、对话内文件链接点击即预览、
  交付物卡片与会话产物面板、会话结束后自动「做梦」整理记忆；工作区全文搜索（轻量 RAG）、
  常用资料固定（新会话自动带入）、任务模板库（欢迎页一键发起 + slash 命令）
- 🧮 **造价数据库** — 一级板块（zaojia-database 蒸馏）：综合单价一级/人材机二级组成、
  测算项目与版本沉淀闭环、造价参考分位数对标、复盘笔记、《市政成本测算手册》整本导入、
  历史数据自愈；「数据库就是数据库」，测算/对标由办公 agent 工具承担
- 💬 **微信助手**（beta，个人使用）— 扫码绑定，ClawBot 远程对话；凭证失效不自动恢复
- 🔑 **SuperGrok OAuth** — PKCE 一键登录，token 持久化

## 技术栈

| 层 | 技术 |
|---|------|
| 桌面框架 | Wails v2 |
| 后端 | Go 1.26 |
| 前端 | React + TypeScript + Ant Design + Vite |
| AI 引擎 | xAI Grok (SuperGrok OAuth PKCE) + 多模型中心 |
| 图像 | ComfyUI (Flux / Z-Image-Turbo / Krea2) |

## 快速开始

```bash
# 安装前端依赖
cd frontend && npm install

# 开发模式（热加载）
wails dev

# 网页版对齐桌面端（HTTP 调试桥接）
设置 `GAEA_HTTP_PORT=8080` 后启动（`wails dev` 或生产 exe），Go 内核会额外
暴露 HTTP 桥接：浏览器访问 `http://localhost:5173`（Vite 已把 `/api` 代理到
桥接）即可驱动同一个内核——模型中心/聊天/记忆/办公引擎全部走真实数据与本地
模型，而不是前端 mock。事件经 `/api/stream` SSE 实时推送。

# 命令行登录
gaea login

# 生产构建（产物 build/bin/gaea.exe）
wails build
```

## 项目结构

```
gaea/
├── main.go                    # Wails 入口
├── wails.json                 # Wails 配置
├── internal/
│   ├── app/                   # App 绑定 + 各模块 handler
│   ├── ai/                    # AI 调用层（SSE 流式 + 图片）
│   ├── auth/                  # OAuth PKCE + token
│   ├── gaea/                  # 办公引擎（agent/tool/control/skill/boot/knowledge）
│   │   ├── cost/              # 成本库（条目/组成行/自愈）
│   │   ├── costproject/       # 测算项目（明细/版本/沉淀闭环）
│   │   └── costref/           # 造价参考（分位数指标/复盘笔记）
│   ├── whisper/               # 轻语模块（人格/记忆/分发）
│   ├── modelengine/           # 模型中心
│   ├── weixin/                # 微信助手
│   ├── project/               # 项目管理
│   ├── worldview/ character/ outline/ chapter/   # 小说 Agents
│   ├── export/                # TXT/MD/EPUB 导出
│   └── config/                # 全局配置（~/.gaea_config.json）
├── frontend/                  # React 前端
│   └── src/
│       ├── pages/             # 对话/轻语/小说/绘梦/模型中心/办公/知识库
│       ├── gaea/              # 办公板块（gaeaW 原生 UI）
│       └── stores/            # Zustand 状态
├── prompts/                   # RTCO Prompt 模板
├── skills/                    # 内置 Skill
└── docs/                      # 设计文档
```

## 版本

| 版本 | 日期 | 说明 |
|------|------|------|
| **v4.69.0** | 2026-09-04 | 上下文页第三轮精修「对齐 dsh-context 仪表」（绑定面 561 零变更，纯前端）：图例 chip hover 联动高亮（悬停分类该段保持、其余段 150ms 淡出）；趋势卡内联请求详情——默认选中最新一步打开即有内容、删独立详情行（master-detail 收进同卡）；事件筛选改 dsh 多选 chips（全亮→单类→再点恢复全选）；四仪表卡头「标题+右副注」单行排版；Token 环心命中率两位小数；死键 contextview.pickHint 三语删除；vitest 224/1683、tsc/eslint 0、冒烟 200 |
| **v4.68.0** | 2026-09-04 | 上下文页对齐 dsh-context（绑定面 561 零变更）：ContextTiming 时长折叠（wall/ttft/gen/tools，诚实近似）；dsh 全宽网格仪表（四仪表卡+当前上下文宽卡含空闲段+浏览器分类折叠组+文件活动聚合树+Agent 径向树+汇总条）；i18n +47 键；judge 三图 pass；vitest 224/1681、冒烟 200 |
| **v4.67.0** | 2026-09-04 | 上下文标签页重设计（绑定面 561 零变更，ui-ux-pro-max 技能刀）：单列 8 卡 → 驾驶舱三区（顶部总览条+左过程轴+右 inspector 三 tab），功能零删减；judge 视觉验收 pass；vitest 221/1647、冒烟 200 |
| **v4.66.0** | 2026-09-04 | 子代理会话提升（绑定面 560→561 +GaeaPromoteSubagent）：transcript 忠实投影为独立新顶层会话（投影往返双校验/诚实降级/原运行不动）+ SubagentThread「保存为新会话」入口；TasksWorkbench 双源轮询迁 store（四消费点全收敛、失败不再吞错）；Go 全量 exit 0、vitest 221/1647、drift PASS（561）、冒烟 200 |
| **v4.65.1** | 2026-09-04 | 三线并行（绑定面 560 零变更）：任务 ExitCode 退出码透出（强杀欠账过期销账）；追问后台失败写 run meta 前端感知 + 修 v4.64.0 RunFollowUp 终态回写真回归；新 agentNetworkStore 收敛 net 轮询 + runsLoadFail 三语键；Go 全量 exit 0、vitest 220/1636、drift PASS（560）、冒烟 200 |
| **v4.65.0** | 2026-09-04 | 三线并行收欠账（绑定面 560 零变更）：追问失败内联错误条+失败气泡保留原文+一键重试/撤销；「办公工作台偏好」设置卡（四个自动展开偏好进设置中心，新 lib/tasksPrefs）；SubagentsPanel/Sidebar 迁共享 subagentRunsStore（3 路重复轮询收敛为 1 路，store 增 loading/error/reload 向后兼容）；releases/README 索引治理；vitest 219/1624、drift PASS（560）、冒烟 200、实机走查 PASS |
| **v4.64.3** | 2026-09-04 | 任务管理树呼吸感优化（绑定面 560 零变更，纯前端）：AgentTree 行高/间距/字号/状态点/层级缩进放宽 + 本地模型工具区块同步；信息零删减；vitest 217/1605、冒烟通过、实机走查 PASS |
| **v4.64.2** | 2026-09-04 | 修复：任务管理树丢失零工具子代理（绑定面 560 零变更）：FoldAgentNetwork 以「拥有子工具记录」定义节点致纯调研子代理（零工具调用）整批不可见；enrichAgentNetwork 对未被承载的 run 补挂合成节点（id=sa_ ref）；+3 回归；同日 v4.62.2～v4.64.1 六版实机验收六项全 PASS；Go 全量 0 FAIL、vitest 217/1605、drift PASS（560）、冒烟通过 |
| **v4.64.1** | 2026-09-04 | 修复：mt_ 信封双层嵌套转义墙（绑定面 560 零变更）：写端 unwrapModelToolOutput 递归拆包（data.result/message/output 逐层）+ 读端 unwrapEnvelopeText 显示侧拆包救历史转录；双侧回归测试；vitest 217/1605、drift PASS（560）、冒烟通过 |
| **v4.64.0** | 2026-09-04 | Side Chat 式追问（绑定面 559→560 +GaeaSubagentFollowUp）：子代理会话 tab 内可持续提问——复用 continue_from 管道带完整工作记忆继续运行；乐观上屏+专用通道流式+快照轮询；守卫双侧对齐（running/mt_/主回合运行中拒绝）；vitest 217/1603、drift PASS（560）、冒烟通过 |
| **v4.63.4** | 2026-09-04 | mt_/长文本输出 Codex 式有界渲染（绑定面 559 零变更）：mt_ 标签页/超 4000 字 assistant 内容默认限高滚动+「展开全部（N 字）/收起」+字数标注，流式行保持跟随；i18n 三语 +3 键；vitest 217/1602、drift PASS（559）、冒烟通过 |
| **v4.63.3** | 2026-09-04 | 对标 dsh（绑定面 559 零变更）：GaeaSubagentRuns 共享单轮询聚合（每会话单定时器/单在途/不可见门控，App 两处独立轮询并入）+ 新子代理 0→N 自动切右栏任务视图（500ms 去抖重臂+偏好开关默认开）；vitest 217/1601、drift PASS（559）、冒烟通过 |
| **v4.63.2** | 2026-09-04 | 并行子代理（绑定面 559 零变更）：修复 task/run_skill 全局冲突键 !spawn 致同回合批量派发逐个串行——改每调用唯一键落同一并行批（≤8 并发）；TaskTool 用量改互斥合并（并行安全）；本地模型推理排队如实说明；新增分区回归测试 3 例；vitest 216/1597、drift PASS（559）、冒烟通过 |
| **v4.63.1** | 2026-09-04 | 主对话子代理卡片整卡可点：单击 task/run_skill 卡直开子代理会话 tab（绑定面 559 零变更）：ref 解析+空 ref 唯一 running 命中回退（宁缺勿错）、tab 预填+轮询自校正、活动行同入口；i18n 三语 +1 键；vitest 216/1597、drift PASS（559）、冒烟通过 |
| **v4.63.0** | 2026-09-04 | 子代理会话 tab 输出对齐主对话 Codex 式渲染（绑定面 559 零变更）：正文/思考走 AssistantMessage、工具按 toolCallId 配对走 ToolCard（subagentRender.ts 纯映射层，孤儿降级/运行中 running）、流式实时行同款；vitest 214/1594、drift PASS（559）、冒烟通过 |
| **v4.62.2** | 2026-09-04 | 热修复：对话标签页实时输出失聪（绑定面 559 零变更）：v4.61 子代理标签页卸载时 EventsOff 连坐炸掉主对话事件订阅——按监听者精确注销修复+3 回归测试；mt_ transcript 信封拆包（消灭字面 
 转义墙）；GaeaTaskList 变参必填修正（任务中心恒空）；vitest 214/1590、drift PASS（559）、冒烟通过 |
| **v4.62.1** | 2026-09-04 | 热修复：子代理流式打断对话窗过程可见性（绑定面 559 零变更）：SubagentText 分道专用通道 gaea-subagent-text（无 seq），修复「wire-only 事件消费 seq 致缺口不可补拉→反复 resync 整体重建对话视图」的 v4.62.0 回归；forwarder 不变量成文 + 回归测试钉死；vitest 214/1587、drift PASS（559）、冒烟通过 |
| **v4.62.0** | 2026-09-04 | 办公板块：子代理逐 token 流式 · 交付验收闭环 A2（绑定面 559 零变更）：SubagentText wire-only 事件通道（EventLogSink 免落盘），持久化子代理运行中文本增量实时渲染到会话 tab（P1 销账）；Word 修改队列（框选攒批去重→批量执行→再定位防错位→诚实跳过/重试→汇总）；版本结构化对比（docx 段级 diff+序号列、xlsx sheet/单元格差异表+截断，收 unsupported 欠账）；i18n 三语 +35 键；vitest 214/1586、drift PASS（559）、冒烟通过 |
| **v4.61.0** | 2026-09-04 | 子代理会话闭环（绑定面 559 零变更）：Word 预览目录侧栏（docxOutline 解析/定位/章节修改模板）；子代理 tab 对齐主代理（Markdown 正文+状态点）；子代理 transcript 真机接线（惰性 SubagentStore + ~1s 快照实时化，后台子代理落盘修复）；本地模型工具同 UI（ModelBacked 标记 vision/summarize_file → mt_ 变相子代理记录，同一行/tab 展示）；子代理入口收敛 task + run_skill 两级（移除三办公包装与 explore 等分类残留）；vitest 211/1535、drift PASS（559）、冒烟通过 |
| **v4.60.0** | 2026-09-03 | better-sidebar pane 化三刀合并发布（绑定面 559 零变更）：右栏工作台 pane 化（欢迎卡→视图/文件/浏览器/任务 tab、点子代理独立会话 tab、旧两级组件删除）；左栏子代理会话入口（父会话展开→子代理子行→独立 tab）；文件打开统一开 pane 文件 tab（产物/正文交互卡/变更行与资源管理器同 tab 条，可并存去重）；并发子代理固化为默认执行纪律；vitest 210/1525、drift PASS（559）、冒烟通过 |
| **v4.59.0** | 2026-09-03 | 继续三线并行（绑定面 559 零变更）：A i18n——设置五面板+SettingsSection 入三语字典 +192 键/语言（682→874，设置中心九面板全量 i18n）；B 小说——搜索命中「落为划线」一键永久标注（区间→摘录口径适配，保留原文大小写）+ReadingPrefsPanel 拆分净减 47 行 +13 用例；C 模型中心——自定义引擎用户价目 v1（EngineConfig 指针三态字段+折算最高优先层+零值语义回归锁，559 不变，models 重生成）；收口抓潜伏雷——ChatPanel 直调 wailsjsCompat 浏览器同步抛致设置聊天分组白屏，try 兜底修复；vitest 204/1457 |
| **v4.58.0** | 2026-09-03 | 继续三线并行收欠账（绑定面 559 零变更）：A 小说——同章搜索重定位缺陷根因实锤+最小修复（searchLocateSeq 入 effect 依赖，回归测试反向验证）+ChapterPage 拆分第三批净减 67 行 +41 用例；B dev mock——补 GaeaBenchmark 五方法（中性空态/诚实失败）+GetModelMonitor+契约 7 用例；C i18n——设置三面板文案入三语字典 34 键/语言（zh 保真，648→682 键）；收口——GetModelMonitor 三消费点迁 getModelMonitor 三态回退（wailsjsCompat 直读绕过 bridge mock 的欠账真身）+mock 补 GetEngines；?mock=1 走查零横幅；vitest 202/1442 |
| **v4.57.0** | 2026-09-03 | 设置中心化繁为简（绑定面 559 零变更）：删四——绘梦纯展示卡（与下拉重复）、小说角色剧照零交互卡（并入存储目录 desc）、关于「系统信息」收成「存储路径」（去与模型分组重复三行）、api/settings.ts 七个零消费死导出；增一——通用分组「界面语言」切换（跟随系统/简体/繁體/English，i18n setPref 首个入口，即时生效）；修一坑——ImageGenPanel 补 comfyui_url/image_save_dir 回填防已存配置被清空；vitest 197/1393 |
| **v4.56.0** | 2026-09-03 | 继续完善三线并行（绑定面 559 零变更）：ChapterPage 拆分第二批搬移（累计净减 ~111 行 +14 用例）；dev mock 补 Herdsman 七方法（引擎管理错误横幅消除，契约 5 用例）；并行 task 卡 provider 升级 (ref,args)+matchRunningRun 唯一命中关联（+13 用例，宁缺勿错）；收口补 GetModelCallStats 空聚合（统计横幅消除） |
| **v4.55.0** | 2026-09-03 | 继续完善三线并行（绑定面 559 零变更）：ChapterPage 拆分补测第一步（阅读高亮三函数抽 chapter/readingHighlight.ts + 首个测试文件 15 用例）；dev mock 补 Get/SetEngineFailover + engines.ts 浏览器回退（调度三开关 mock 下全点亮）；failover toast 文案 engineLabel 化（三级回退）；右舷虚线交叠 DOM 核实证伪销账 |
| **v4.54.0** | 2026-09-03 | 继续完善三线并行收欠账（绑定面 559 零变更）：办公面板任务中心空态文案纠偏+产物/变更分隔线加深；首页矩阵「编程」升格 span 4×1 宽瓦片收末行空位（四响应档整除核算入蓝图）；eslint 三 warning 清零（全量首次 0/0，ChapterPage 链式 useCallback 稳定化）；judge 视觉验收 2/2 pass |
| **v4.53.0** | 2026-09-03 | 办公欢迎界面化繁为简（绑定面 559 零变更）：删对话头部上下文横条（走主区上下文标签页）+删侧栏底部模型卡（入口统一顶栏）；右栏 6→4——产物+变更、任务+分工直接合并为一个面板（上下分区同屏全可见，非二级标签），旧持久化 id 别名收敛，零功能删除；judge 视觉验收 3/3 pass |
| **v4.52.0** | 2026-09-03 | 首页重设计「星枢港 · 双舷驾驶舱」（绑定面 559 零变更）：左舷=紧凑 Hero+命令条五要素+能力矩阵 Bento（编程/设置瓦片化入格），右舷=内核遥测三表+写作进度环+会话+记忆+晨报；v3 五段并三段零功能删除（chips/状态细条/门廊收编），i18n 三语 648 键（新增3/更新3/删22 含 9 个死键）；judge 视觉验收 overall pass |
| **v4.51.0** | 2026-09-03 | 壳层左缘修复（main 区预留收起态 rail 48px，全板块首字符裁切修复）+ 创建青鸟助手深链直通绑定（sessionStorage 焦点 + NAVIGATE crossSpace 跨空间，落青鸟选中并直开扫码；修复 S2.1 同空间守卫静默丢弃缺口）（绑定面 559 零变更，judge 视觉验收 overall pass） |
| **v4.50.0** | 2026-09-03 | 造价数据库化繁为简（绑定面 559 零变更）：导航 8→6（价格源+价格仓库+询价库归并为「价格数据」，知识图谱降为概览镜头）、询价库从成本条目隐藏 icon 视图升格一等子页、概览快捷入口 7→4、CostLibraryView 拆 inquiry 模式与 compact/onInsert 死管线、删不可达 memoryhub/CostLibrary、dev mock 补询价/五算九方法（judge 视觉验收 overall pass） |
| **v4.49.0** | 2026-09-02 | 青鸟助手生命周期补全（绑定面 559 零变更）：①编辑已有助手（改名/换人格，PersonaPickerPanel 新增/编辑共用，自定义人格原值保留）；②角色库 custom 角色卡一键创建未绑定助手；③EnsureAssistants 镜像守卫——custom 角色绝不被镜像覆写（真隐患修复） |
| **v4.48.0** | 2026-09-02 | 青鸟：微信助手板块更名 + 新增助手人格选择器（绑定面 559 零变更）：双栏选择器（轻语预设+角色库可聊天角色、搜索过滤、详情预览）、角色立绘回显、18+ 人格过滤；CharacterList/WhisperGetPersonalities 转正 AppBindings（judge 视觉验收 5/5） |
| **v4.47.0** | 2026-09-02 | 微信助手星枢化「通讯枢纽」工作台（绑定面 559 零变更）：三分区布局（玻璃细条+左通道轨道+主区三视图）、通道状态三重传达（状态点四态+状态字+键值卡）、扫码流三步指示、dev mock 补微信域；功能零删减（judge 视觉验收 6/6） |
| **v4.46.0** | 2026-09-02 | 小说板块第二轮（绑定面 557→559）：①伏笔登记表闭环——SaveForeshadows 写入口+syncForeshadows 按 ID 合并（手工条目不冲掉）+面板登记/流转/编辑/删除；②导出扩展——EPUB 真封面自动探测嵌入、onlyMainline 仅主线参数（默认兼容）、DOCX 导出落地（gooxml）；③伴读划线问书多轮化——historyJSON 六轮+划线窗口 12000 rune+会话式弹窗；④一致性 AI 深检 v0（Continuity Linter）——逐章状态卡+本地跨章比对五类矛盾+规则层合并 source 徽标+诚实降级。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1312/1312、drift PASS（559）。详见 releases/v4.46.0.md |
| **v4.45.0** | 2026-09-02 | 百炼全量下线（绑定面 557 零变更）：用户拍板删干净——DashScope 引擎退出注册表、DashScopeAPIKey 字段与迁移全删、SetImageBackend 第 5 参彻底移除；imagegen 引擎枚举去 dashscope + ResultStage 新增「改图」动作（img2img 走 ComfyUI/Herdsman）；微信改图链（v4.9 起）全删：ActionEditImage 意图解析/执行层/入站图缓存/wx_agent edit_image 工具（7→6）/editImageFromCard；gen_bindings 空白参数名占位修复。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1312/1312、drift PASS（557）。详见 releases/v4.45.0.md |
| **v4.44.0** | 2026-09-02 | 绘梦专项三刀（绑定面 557 零变更）：①百炼模型残留修复（真 bug）——dashscope 后端 GetImageBackendInfo/SetImageBackend 空或残留模型（grok-imagine-*/krea2）归位 qwen-image-edit-plus、前端 modelOptions 固定三档官方编辑模型、切后端归位默认、queue 提交前拦截引擎固有模式残留；②引擎枚举单源化+补 GLM——meta.ts 收敛唯一 BACKEND_OPTIONS（含能力位），ControlPanel/顶条状态 pill/GenerationBar 门禁/启动消息统一走单源，修复「xAI 云端 云端」式拼接重复，GLM 引擎下拉补全（txt2imgOnly 禁用+残留专属警告）；③模板推荐画幅落地——templateSizeToPreset 纯函数把模板 size 比例标签映射到实际画幅（2:3 立绘→自定义 768×1152），applyTemplate/TemplatePicker 全链路消费。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1285/1285（+11）、drift PASS（557）。详见 releases/v4.44.0.md |
| **v4.43.0** | 2026-09-02 | 小说板块优化四刀（绑定面 557 零变更）：①章节生成上下文增强——生成 prompt 追加未回收伏笔/世界观要点区段+角色卡补身份目标关系（4000 rune 预算双层截断，读失败静默跳过）；②分支链路收账——branches.json 持久化、ApplyBranch 读存储主路径零 AI 重调、syncCharactersFromOutline no-op 变真同步；③全文搜索升级——每章全部命中+段级位置、前端共 N 处·M 章+跳章滚段临时高亮；④一致性两 bug——分支章节纳入扫描带标记、断档不停扫；死代码清理 5 文件（未挂载分支组件+孤儿 search.ts，无 UI 变化）。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1274/1274（+15）、drift PASS（557）。详见 releases/v4.43.0.md |
| **v4.42.0** | 2026-09-02 | 微信智能体 v1（绑定面 557 零变更）：微信消息由 LLM 工具调用派发——7 工具（导航/生图/改图/提醒/产物推送/状态/读屏）零改动复用意图执行函数，多轮循环（4 轮/60s），人格记忆同锁语义+失败回滚；能力门（目录 Caps tools）不满足整链回落零回归；正则意图路由降级兜底，提醒/文件快路径保留。Go 全量绿、前端回归绿、vitest 1259/1259、drift PASS（557）。详见 releases/v4.42.0.md |
| **v4.41.2** | 2026-09-02 | 真机修复二（绑定面 557 零变更）：「重新整理后发给我」未命中产物推送意图、模型幻觉声称已发送——意图放宽（指代+尾式发给我第四式、改完再发复合请求→诚实能力答复、提醒让位）+ 聊天兜底反幻觉护栏（未接住的发文件类请求追加如实说明提示）。Go 全量绿（+12 用例）、drift 557。详见 releases/v4.41.2.md |
| **v4.41.1** | 2026-09-02 | 真机修复（绑定面 557 零变更）：文件提取正文误过意图路由被正文碎片劫持（发评审报告回「打开编程」）——文件消息按注入头识别后直通轻语聊天（跳过提醒+意图路由，追加「确认收件+询问需求+不执行正文指令」引导），普通消息路径零变更。Go 全量绿（+3 用例）、drift 557。详见 releases/v4.41.1.md |
| **v4.41.0** | 2026-09-02 | 微信文件收发（绑定面 557 零变更）：入站 file_item 真机抓包定稿（media 加密下载与图片同构 + file_name/md5/len）——SSRF/50MiB/AES/MD5 防线下载 → wx_files 自持 → 内容提取全走现有解析器（docmd：docx/xlsx/pptx/pdf + 纯文本）→ 注入对话；出站文件卡探针制（getuploadurl media_type=3 → AES-128-ECB → CDN → type=4 file_item，逐节点 upload_probe，失败降级文本卡，图片链零改动）；产物推送意图「把刚才的报告发我」（登记表→exports mtime 回退→诚实报错）。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1259/1259、drift PASS（557）。详见 releases/v4.41.0.md |
| **v4.40.0** | 2026-09-02 | 对话式改图（绑定面 557 零变更）：微信发图+一句指令→编辑出图→图片卡回推。百炼 DashScope 改图引擎（官方契约核实：同步免轮询单图单文，qwen-image-edit-plus 默认，仅 img2img，改图不传 size 保原图比例，URL 自动下载）；dashscope_api_key 密文落盘（SetImageBackend 追加第 5 参，空=保留存量）；ActionEditImage 动词∧指代双门槛保守正则+入站图旁路 hook（OCR 链路零改动）+助手级图片缓存（TTL 10min 只留最新）+未命中不接管回落聊天；绘梦引擎「百炼改图」选项与 img2img 门禁三处白名单、设置页 Key 框。Go 全量绿、tsc/tsc -b/eslint 0、vitest 1259/1259（+11）、drift PASS（557）。详见 releases/v4.40.0.md |
| **v4.39.0** | 2026-09-02 | 微信助手管理台（绑定面 557 零变更）：连接卡升级助手管理台——助手卡（Avatar/人格 Tag/通道状态徽标）+ 启停 Switch + 删除 + 逐助手扫码绑定/重绑（修 confirmBinding 硬编码 gaea）+ 新增微信助手表单（wx_ 动态 id）；manager.Update 补回写 WxBotID/PortraitURL/VoiceGuide/Gender/Tags/Dims（空值保留防部分保存清空）；WhisperAssistantSave 空 token/userId 保留旧凭据（启停切换零风险）；同人格多助手 AssistantName 锁外直写数据竞争修复（聊天链重构 whisperChat 内部链，注入移入 LockTurn 窗口，绑定签名零变更）。Go 全量绿（+5 用例）、tsc/tsc -b/eslint 0、vitest 1248/1248、drift PASS（557）。详见 releases/v4.39.0.md |
| **v4.38.0** | 2026-09-02 | 目录通用化（绑定面 557 零变更）：模型元数据目录从 GLM 扩展到 deepseek/xai/opencode-zen（25 条目官方核实）——通用目录 model_catalog.json v1+loader（opencode-go 订阅制无按量售价拍板不入目录防误导）；estimatePrice 目录优先层扩展+估算修正（deepseek-v4-flash 3.0→12.672 CNY 官方 USD 峰价、grok-4.5/4.6 129.6→57.6、zen 8 模型新增计价，claude/gpt/gemini/kimi 旧条目回归锁逐位一致）；fetchModels 动态列表 enrich（id 归一化匹配只填空不覆盖），deepseek/xai/zen 模型卡上下文/价格/免费徽标自动点亮（前端零改动）。官方停用模型（deepseek-chat/reasoner、grok-4/3 系）不加条目走内置表兜底。Go 全量 test exit 0（filewatch 一例负载 flaky 复跑绿）、tsc -b/eslint 0、vitest 1243/1243、drift PASS（557）。详见 releases/v4.38.0.md |
| **v4.37.1** | 2026-09-02 | D 刀收口（绑定面 557 零变更）：模型库卸载确认带释放大小（file_size→「释放 X GB」）；磁盘占用展示与 /health 透出经核实已存在剔除不入刀。详见 releases/v4.37.1.md |
| **v4.37.0** | 2026-09-02 | 健康巡检+故障转移 v0（模型中心调研 C 刀，绑定面 555→557）：巡检 goroutine 10 分钟周期探已启用非本地引擎（GET /models 8s 超时），Status 持久化+「连续 N 次探测失败」前缀+状态变化事件 engine-health-changed，Error 永不含 Key（对抗回显测试锚定）；故障转移开关 engine_failover_enabled（默认关=现状逐字节回归锁），网络类/408/429/5xx 才转移（401 等配置错误不转移），候选=已连接 llm 引擎按 order 取首、用其默认模型重试一次（流式仅首字节前转移），emit model-failover+逐笔记账；前端「本地调度」第三开关卡（降级「未知」禁用）+双事件订阅+总览引擎卡连续失败原文/巡检时间；收口注记 gen_bindings explicitOverrides 登记教训（新方法先登记再生成否则被前缀规则误归 OfficeB）。Go 全量 test exit 0、tsc -b/eslint 0、vitest 1243/1243（+6）、drift PASS（557）。详见 releases/v4.37.0.md |
| **v4.36.0** | 2026-09-02 | GLM 目录 v2·能力/价格元数据+远程热更新（模型中心调研 B 刀=计费三件套本体，绑定面 555 零变更）：glm_catalog.json 升级 44 条目 schema v2（上下文/最大输出/价格/币种/单位/免费档/能力 caps/coding 积分系数，官方免费档 8 个+核实国内价 6 条，查不到的绝对价不编数沿用内置 z.ai USD 估算口径）；远程热更新 v0（glm_catalog_url 默认禁用，24h 拉取+version 比对+缓存兜底，优先级 覆盖文件>远程>内嵌，仅影响展示与估算不碰路由/alias/鉴权）；估算单源化（estimatePrice GLM 目录优先内置表兜底，价格更新只动目录不发版，估算值零回归锁）；glm_alias 与官方现状核对一致补锚定；前端模型卡上下文/能力/价格徽标+StatsSection coding 行官方公式估算积分（含缓存通道）+目录来源小注。Go 全量 test exit 0、tsc -b/eslint 0、vitest 1237/1237（+18）、drift PASS（555）。详见 releases/v4.36.0.md |
| **v4.35.0** | 2026-09-02 | 自定义引擎（模型中心专项调研 A 刀，绑定面 552→555）：OpenAI 兼容服务商任意添加——新类型 EngineCustom + Manager 六方法（添加/更新/删除+Key 注入），engineID=custom-前缀+slug 冲突追加序号，Key 存 config 新键 custom_engine_keys（加密 JSON map+saveSetters 登记）不落 engines.json 不下发前端，baseURL 校验 http(s)+host（v4.9.1 Key 粘错框防线延伸），LoadState 恢复 custom 条目防伪造；聊天路径 BuildChatURL/resolveChatEndpoint custom 分支（流式+非流式，空 Key 不发 Authorization 头），真正可聊天/设活跃/绑功能；前端引擎管理「添加自定义引擎」表单+custom 卡地址框/编辑/删除确认/徽标，内置云端地址框防线一字未动（新增回归锁）。Go 全量 test exit 0、tsc -b/eslint 0、vitest 1219/1219（+9）、drift PASS（555）。详见 releases/v4.35.0.md |
| **v4.34.0** | 2026-09-02 | 子代理气泡恢复（收 v4.26 沿旧欠账，绑定面 552 零变更）：修复恢复会话后「子代理」徽标气泡整条丢失——根因=session.ProjectMessages 投影无 subagent_message case；模型面投影不可动（恢复后模型上下文须与实时语义一致），改走 UI 侧并行投影：session 新导出 ProjectSubagentAnchors 锚点镜像（projection.go 零改动）+ GaeaHistory 读磁盘事件日志按锚点合并子代理气泡（mergeSubagentAnchors 纯函数 + logOffset 校正检查点 system 提示偏移，越界锚点宁漏勿误丢弃），HistoryMessage 加 subagentRef（golden 不变）；前端 rebuildHistoryItems 透传 subagentRef 复用实时徽标渲染。Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest 1210/1210（+3）、drift PASS（552）。详见 releases/v4.34.0.md |
| **v4.33.0** | 2026-09-02 | 细节收口第三刀（绑定面 552 零变更，三并行子代理+主代理集成）：①回滚守卫统一+write_file >8KB 恒误报修复（真 bug）——rollback 卡接入「恢复后已被手工修改」守卫（撤销恢复前校验防覆盖编辑），write_file 守卫改 evidence.ClampSummary 同口径截断比较（原精确比较对 >8KB 未手改文件必然误拒），截断单点化，8KB 窗口外手改不可检为有意取舍（宁漏勿误）；②pdf 占位比按实测精确化——pageLazy 新 nextPageAspect/placeholderAspect（onLoad 读 naturalWidth/Height，首个有效测量为整档比例，无测量回落 A4），占位→真身交换不再跳高；③主区预览 pdf 懒加载对齐弹窗——FilePreview pdf 分支接入 IO 单向懒加载（初始 4 页/800px 预挂/不卸载/大纲跳转强制渲染/无 IO 降级）+测量比例接线。Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest 1207/1207（+11）、drift PASS（552）。详见 releases/v4.33.0.md |
| **v4.32.0** | 2026-09-02 | 细节收口第二刀（绑定面 552 零变更，三并行子代理+主代理集成）：①回滚先快照当前态（收 v4.28 B1 欠账）——GaeaRollbackRecord 恢复前快照目标当前内容（evidence 新导出 StageBaselineTo 命名逻辑单点化），rollback 记录升级为完整证据卡，**恢复动作本身成为时间线里可再恢复的版本（撤销恢复=对 rollback 卡再点恢复）**，目标缺失/快照失败降级不阻断；②产物自动弹出+偏好（收 v4.30 欠账）——deliverablePrefs（gaea.deliverableAutoOpen 默认关）+面板头部胶囊+App 新产物 diff 自动切「产物」tab（尊重 tab 停用态、激活即清零角标），单版本徽标 title 细化为「有 N 个历史快照」；③弹窗 pdf 逐页懒加载（收 v4.31 欠账）——lib/pageLazy 纯函数+IntersectionObserver 单向懒加载（初始 4 页/800px 预挂/不卸载），大纲跳转目标页强制渲染，无 IO 全量降级，顺带修 IO 观察集为空致懒加载永不触发的真 bug；④预览最大化持久化（收 v4.30 欠账）——gaea.previewMaximized 独立键+三处落盘。Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest 1196/1196（+30）、drift PASS（552）。详见 releases/v4.32.0.md |
| **v4.31.1** | 2026-09-02 | -count>1 全量绿化（绑定面 552 零变更）：统一根因=测试写进程级全局状态（provider/billing/boot/app 注册表 kind、whisperSessions 会话缓存），`-count` 多次运行不兼容。修法=注册 kind 改 testKind(prefix)（进程级 atomic 单调计数，19 注册点唯一化）+ app whisper 会话隔离改唯一会话 ID + t.Cleanup 清理（12 调用点）+ **whisper PacedStreamEmitter 真 bug 修复**（streamDone 分支收尾末气泡，修 MarkDone 挂起/末气泡 OnBubbleEnd 永不触发，+3 生产行）。验证=五包 -count=2/-count=5 全绿、发射器 -count=300 全绿、tasks -count=20 仍全绿、**全量 go test -count=2 ./... exit 0**；前端零改动、drift PASS（552）、版本四处 4.31.1。详见 releases/v4.31.1.md |
| **v4.31.0** | 2026-09-02 | 细节收口四线并行（绑定面 552 零变更，四并行子代理足迹互斥+主代理集成）：①产物版本时间线单版本入口（收 v4.28 B1 欠账）——versions>1 按现状 vN 徽标（旧锁不破），versions≤1 但有 journal baselinePath 快照的产物渲染「版本」入口徽标可点开时间线（预览/恢复），无快照保持空态；②FilePreviewModal pdf/pptx 逐页预览（收 v4.28 欠账）——弹窗 kind=pdf 分支补齐逐页缩略（页锚点滚动）+dataUrl 整本回退+诚实空态+PptxOutline 大纲卡（「针对第 N 页修改」composer 插入），FilePreview 本体零改动；③轨迹历史轮耗时（收 v4.26 欠账）——后端 Turn.DurationMs（turn_done−turn_started ×1000，omitempty 向后兼容）+TrajectoryView 轮次头「用时 Ns」，零新增绑定（结构字段级）；④TestCancelConcurrentStress flaky 根治（**实现层真竞态**）——pickNext 不做任务级预留→落选 worker 无条件删 cancelReq→Cancel 已成功返回的任务终态被 succeeded 吞掉；修复=tasks.go 新 clearStaleCancel（只清残留绝不删取消意图）+胜者重登记 cancel，测试改事件驱动等待、断言不削弱。Go 全量 0 FAIL（tasks -count=100 全绿）、tsc/tsc -b/eslint 0、vitest 1166/1166（+9）、drift PASS（552）。详见 releases/v4.31.0.md |
| **v4.30.0** | 2026-09-02 | 办公 UI 化繁为简第二刀（绑定面 552 零变更，红线：简化≠删除功能；收 v4.29.0 欠账四项）：①产物生成自动置前/角标（Devin Auto-open 式）——App diff 会话内新产物路径 → 产物 tab 角标（未查看数，激活即清零）+产物面板行「新」徽标与高亮（data-fresh 可测），会话切换重置基线；②面板行级降噪（Cowork 一行式）——产物/变更/任务三列表次级信息（路径/相对路径/时间/重试）改悬停次行显现（group-hover），title 全保留；③命令面板按当前视图重排（Linear 式）——新 lib/paletteRank 纯函数：当前激活右栏面板 cmd 置顶、chatTab=overview 时概览置顶、其余稳定保序，CommandPalette 零改动；④预览「半幅↔最大化」两档（VS Code Toggle Maximized Panel）——FilePreview 头部最大化/还原按钮（icons 补 Maximize2/Minimize2），最大化占满可用宽度、还原回半幅、拖拽分割条自动退出。Go 0 FAIL（零 Go 变更）、tsc/tsc -b/eslint 0、vitest 1157/1157（+10）、drift PASS（552）。详见 releases/v4.30.0.md |
| **v4.29.0** | 2026-09-02 | 办公 UI 化繁为简（绑定面 552 零变更，红线：简化≠删除功能；弹药 docs/market-research-2026-09-01b.md 两线调研）：①顶栏导出收拢——新 ExportMenu，导出 Markdown/Word/PDF 三常驻文字钮收进单钮「导出 ⌄」下拉（对标 Devin/Linear 只进菜单不加按钮 + VS Code 单点溢出），三出口与统一交付管线原样保留，顶栏常驻操作钮 7→5；②右栏 tab 窄栏自适应图标化——容器 <420px 时 6 tab 文字 CSS 隐藏只显图标（aria-label/title/角标保留，Notion Icon only 式），宽栏恢复文字，6 tab 集合与数量锁不动，340px 基线宽拥挤根治；③预览头部降噪——FilePreview「打开/定位」图标化+头部按钮去边框，「编辑/保存/取消」状态语义文字保留（编辑能力保留红线测试钉住）。Go 110 包 0 FAIL、tsc/tsc -b/eslint 0、vitest 1147/1147（+9）、drift PASS（552）。详见 releases/v4.29.0.md |
| **v4.28.0** | 2026-09-01 | 浏览器与版本（A2+B1+B2/C3，绑定面 550→552 +GaeaPptxOutline/GaeaBrowserObserve）：①A2 浏览器观察窗——右栏新「浏览器」tab：CDP 截图步进流（captureScreenshot jpeg ≤1280，未运行绝不拉起）+URL/标题+操作时间线（browser_* 倒序上限 20）+权限静态行+自动弹出胶囊（gaea.browserAutoOpen，新 browser_* 工具自动切 tab，2.5s 可见门控轮询）；②B1 文件版本时间线——产物 vN 徽标可点→内联时间线（时间/工具/轮次/状态）+基线预览+恢复（RollbackRecord，恢复=新增证据卡不丢历史），零 Go 改动长在证据链上（对标 Notion 版本史/Artifacts rewind，预览即护栏）；③B2/C3 pptx 交互——GaeaPptxOutline（python-pptx 结构化大纲）+GaeaPreview .pptx 分支（soffice→PDF 缓存+poppler 逐页缩略 ≤60 页）+前端逐页预览+大纲侧栏+页锚点滚动+「针对第 N 页修改」指令插入。三并行子代理分线+主代理集成。Go 110 包 0 FAIL、tsc -b/eslint 0、vitest 1138/1138（+43）、drift PASS（552）。详见 releases/v4.28.0.md |
| **v4.27.4** | 2026-09-01 | todo 持久化改名（绑定面 550 零变更）：**勘误**——`.gaea/progress.md` 四次被覆写并非「并行会话」，真凶是 gaea 自身 `todo_write` 的计划进度持久化写 `<工作区根>/.gaea/progress.md`，办公代理以本仓库为工作区跑任务时每次 todo_write 都覆盖同名发布进度文件（文件名撞车）。修复=持久化改名 **todos.md** + compaction 读取端优先新名/回退旧名（存量工作区兼容）。测试 +2（写 todos.md 不碰 progress.md、读取优先/回退）。Go 110 包 0 FAIL、前端零改动。详见 releases/v4.27.4.md |
| **v4.27.3** | 2026-09-01 | markdown 包裹符修复（绑定面 550 零变更）：模型用反引号包裹路径时（`` `安全文明手册/x.docx` ``），fileLinks 匹配把开头反引号吞进路径——交付卡片点击→预览「文件不存在」、「在文件管理器中定位」打开错位（真实会话现场实锤）。根因=路径字符集不排除 `` ` `` 与 *，两者是 Windows 文件名非法字符、应作路径边界（v4.26.1 全角括号盲区第二弹）。修复=PATH_BODY/FIRST_SEG 排除包裹符+PATH_BOUNDARY 纳入+BARE_FILE_RE 允许包裹符前缀；下划线合法字符不受影响；存量消息渲染时实时重提取、重启即恢复。tsc -b/eslint 0、vitest 1095/1095（+5）。详见 releases/v4.27.3.md |
| **v4.27.2** | 2026-09-01 | 细节收口（绑定面 550 零变更）：①subagent_message 端到端收口——v4.26 回投特性此前实际未通（后端发 kind=subagent_message、前端无消费整条被丢），wire 层转译 kind="message"+subagentRef（磁盘日志仍按原始 kind 落），「子代理」徽标气泡真实生效；补拉折叠同步（GaeaResyncItem.subagentRef 恒全键、fold→独立条目+closePending 防误续写）；②轨迹面板子代理记录——TrajectoryRecordKind 加 "subagent"，徽标/Bot 图标/折叠行/详情全文/搜索命中，turns 与 betweenTurns 双落点；③sidebar_open 目录定位（收 v4.25 欠账）——directory→FileTree 树中定位，顺带修 FileTree 目录行无 data-path 锚点导致 reveal 静默失效的暗坑。Go 110 包 0 FAIL（TestCancelConcurrentStress 负载型 flaky 单跑稳定）、tsc -b/eslint 0、vitest 1090/1090（+5）、drift PASS（550）。详见 releases/v4.27.2.md |
| **v4.27.1** | 2026-09-01 | seq 防线 omitempty 失配修复（「运行中只显示思考读秒、无过程卡/文本卡交替」根因收口）：v4.26 补拉防线前后端形状契约失配——Go GaeaResyncItem 全字段 omitempty（流式 assistant 空 reasoning、写类工具 readOnly:false 键被序列化省略），前端 parseResyncItems 严格校验缺键即整快照判坏 → 补拉快照 100% 被拒、防线静默失效；Wails 吞件期间对话窗无物可渲染（WorkHeader 是 store tick 驱动所以活着，轨迹面板读盘不受害）。修复=Go 全字段去 omitempty（TestGaeaResyncItemWireAllKeys 锁「序列化恒全键」契约）+前端缺省键宽容（缺键→零值，类型错/kind/id/status 校验不变）。真机验证：真实应用发只读任务——对话窗 WorkHeader「已完成 · 用时 15s · 7 步」+阶段行+思考块+ls 工具卡+正文交替（elapsed 3s→8s→14s 运行中逐个渲染）。绑定面 550 零变更。Go 110 包 0 FAIL、tsc -b/eslint 0、vitest 1085/1085（+5）。详见 releases/v4.27.1.md |
| **v4.27.0** | 2026-09-01 | 右侧面板对齐 Codex（纯前端，绑定面 550 零变更）：①右栏文件工作台——点文件后预览占满右栏（原顶部 3/5 小窗 + 底部文件树挤压）、文件树收敛为「文件」按钮切换的 260px 侧栏（打开首个文件自动收起）、宽度上限 720→1600（视口−侧栏−400 对话区动态钳制，首次打开自动抬升 560）、编辑器 tab 文件类型图标（lib/fileIcon 单源）、树内高亮当前编辑文件；②标签扁平化——删「资料/成本库」、取消二级标签 → 文件/产物/变更/任务/分工一级平铺，旧存储值自动收敛；③对话输出——用户消息去气泡、第 2 轮起「第 N 轮」回合分隔线、助手消息复制按钮、编辑类工具 +N−N diffstat 芯片；④子代理实时下钻——点击子代理 → 全面板对话（SubagentThread：思考折叠/tool 卡/状态徽标），运行中 3s 轮询 + 事件驱动实时刷新 + 自动跟随底部；⑤上下文标签——总览头部水位分色（≥70% 琥珀/≥90% 红）+缓存/费用/刷新、空态引导、文件活动行点击打开预览、步骤详情「占窗口 %」、趋势图悬停六分类构成。vitest 1082/1082、tsc -b/eslint 0、Go 零改动、drift PASS（550）、版本四处 4.27.0。规划 v4.27「浏览器与版本」顺延。详见 releases/v4.27.0.md |
| **v4.26.1** | 2026-09-01 | 全角括号文件名修复：fileLinks 的 PATH_BODY 把（）当路径终止符——文件名含（修订）（终稿）时正则截断、扩展名拼不上，「交付文件」卡片与内联文件链接整体失配（真实办公会话实证：正文有 C:\…\开工筹备计划（修订）.docx 但卡片未渲染）。修复=路径体允许全角括号、扩展名仍锚定末尾（不吞「（三份）」类补语）。绑定面 550 零变更。tsc -b/eslint 0、vitest 1080/1080（+8：匹配用例 5+组件回归守卫 3）。详见 releases/v4.26.1.md |
| **v4.26.0** | 2026-09-01 | 对话流式重造（对齐 Codex，插刀）：根因=「发送后对话窗静默而轨迹在动」六连——子代理 text/reasoning 有意不进主聊天、TurnStarted 前预处理窗零事件、Wails 事件流吞件、Retrying 未映射、phase 空 seam、TTFT 静默。①工作态头部行 WorkHeader——turn 激活期常驻（spinner+阶段文本+已用时 1s tick+步数，items 为空也渲染，消灭死寂窗口），完成后转 Codex 式「已完成 · 用时 · N 步」耗时行；②后端 phase 事件接线——预处理各阶段（正在启动引擎/解析 @引用/装配首轮上下文/检索记忆/思考中）+Retrying/compaction 转译为 phase（磁盘日志格式不变），phase 收编进过程卡+头部；③子代理活动回投主回合（Codex 2026-08 同款）——新事件 subagent_message 回投子代理最终答复，主区消息带「子代理」徽标，task 卡 running 实时 lastText/lastTool 预览（5s 轮询注入）；④事件序号防线——gaea-event 全量带 seq（转发层原子递增，会话切换归零），跳号→新绑定 GaeaResyncEvents（549→550）从磁盘日志折叠全量快照整体替换（冷却门 5s/在途去重/坏快照保底，golden 逐字节不变）；⑤重复工具折叠「已调用 X · N 次」（Claude Code 式）+StreamingIndicator 兜底重定；⑥顺带修复 weixin_reminder_test 时间炸弹。Go 全量 0 FAIL（golden/fold 原样通过）、tsc -b/eslint 0、vitest 1072/1072（净增 71）、drift PASS（550）。三并行子代理分线+主代理集成；调研 docs/research-2026-09-01/codex-streaming-ux.md。详见 releases/v4.26.0.md |
| **v4.25.0** | 2026-09-01 | 文件工作台（规划第三刀 A3+B3）：①编辑器 tab 化——文件树点开→右栏内多文件编辑器 tab（lib/editorTabs 外部 store：上限 12 LRU/关闭激活相邻/localStorage 持久化坏值兜底），FilePreview embedded 模式，docx/xlsx/md/图片/PDF 能力原样随迁（换壳不换芯红线），双入口保留（树行点击=右栏内开 tab、右键=主区预览 pane），产物行「树中定位」→FileTree 展开父链+滚动+闪烁；②变更 tab diff 化（Git 面板式）——文件行展开→行级红绿 diff（lib/planDiff 三态：edit_file/multi_edit 真 before/after、write_file/edit_lines 写入内容预览+原因、其余诚实不伪造）+回滚接证据链 Journal 最近基线；③B3 选区联动——xlsx 选中单元格→浮动「引用到对话」、docx 框选工具栏补「引用到对话」、docx 渲染失败降级纯文本（docxText 提取正文段落）；④模型主动打开——新内置工具 sidebar_open（work 空间/ReadOnly 直允许/防穿越/envelope path_rel）+前端解析器+App 按事件 id 去重接线（file 开编辑器 tab/directory 亮文件 tab）。绑定面 549→549（零新增，内置工具走事件管线）。Go 0 FAIL（+20 用例）、tsc -b/eslint 0、vitest 1001/1001（净增 74）、drift PASS（549）。三并行子代理分线+主代理集成；调研弹药 docs/market-research-2026-09-01.md。详见 releases/v4.25.0.md |
| **v4.24.0** | 2026-09-01 | 子代理工作台（规划第二刀 A1+C1）：①树形实时拓扑 AgentTree——嵌套 Children 全量渲染（此前只画两层），root 折叠为「主 agent」行/更深层默认收起可展开/新节点自动展开父链，节点量化（状态色点/任务摘要/工具数/模型徽标/耗时——running 实时已用 1s tick/错误数），下钻链：节点→详情卡→完整 transcript→工具调用行点击定位结果消息（收 v4.21 欠账）；②合并活动流（Devin 式单列 feed：running 子代理 lastText/lastTool 按 updatedAt 倒序合并上限 20 空态收起）；③新子代理自动展开（可关默认开：新 ref 出现→App 亮出右栏切「分工」tab，偏好键 gaea.subagentAutoOpen 损坏值回落默认）；④C1 权威产物登记表——trajectory.FoldDeliverables 从事件日志折叠写类 8+生成导出类 3 工具落盘登记（路径/工具/轮次/时间/次数上限 200）+ 新绑定 GaeaDeliverableRegistry（548→549）+ DeliverablesPanel「权威产物登记」只读区补启发式漏登。vitest 927/927（净增 16）、Go 0 FAIL、tsc/eslint 0、drift PASS（549）。详见 releases/v4.24.0.md |
| **v4.23.0** | 2026-08-31 | 工作台框架（右栏工作台化第一刀，对标 DSH-better-sidebar）：①Tab 注册表 lib/sidebarRegistry.ts——元数据复用清单+render 接线单一数据源，右栏渲染/命令面板全派生，新增面板=清单+RENDERERS 各一条（框架/内容解耦）；②工作台外壳三件套——全局宽度键（左缘拖拽 280–720、最后一次拖拽胜出跨会话跟随）、声明式设置（齿轮→侧边卡片每 tab 独立开关，停用即隐藏/至少保留一个/停用不进命令面板，启用集全局键）、会话记录 v2（JSON {v,tab,enabled,width}，v1 裸 id 兼容、坏值逐项兜底）；③主区「概览」tab——ChatTabs 第 4 tab + OverviewPanel 承载原 StatsPanel，右栏统计下线（union 移除，旧 tab:"stats" 宽容收敛回「文件」），右栏收敛 3 主 Tab×7 面板。绑定面 548→548（零新增）。tsc/eslint 0、vitest 911/911（净增 38）、Go 0 FAIL、drift PASS（548）。两并行子代理分线+主代理集成。详见 releases/v4.23.0.md |
| **v4.22.0** | 2026-08-31 | 一次性收官：①轨迹真虚拟化——react-window v2 List + useDynamicRowHeight（视口窗口渲染 ±overscan 12，展开行 ResizeObserver 实测自动重排，超长会话 DOM 恒定，v4.21 分批机制退役；概览跳转走 scrollToRow、搜索回顶）；②transcript 消息定位——序号 #N + 搜索命中自动滚动；③晨报预载 UI 开关（+2 绑定 GaeaMorningPreload/GaeaSetMorningPreload，internal/config.Save 持久化+重建引擎即时生效；记忆面板「晨报预载 开/关」）。绑定面 546→548。Go 全量 0 FAIL、tsc/eslint 0、vitest 873/873（+3）、drift PASS（548）。六轮改动一次合并提交 + tag v4.22.0。详见 releases/v4.22.0.md |
| **v4.21.0** | 2026-08-31 | 长会话与 transcript：①轨迹增量渲染——扁平行流（轮次头+展开记录+Between turns）按批渲染（首批 250，滚动到底自动续载或「加载更多」），搜索词变化回首批，概览跳转同步扩可见区；②子代理 transcript 消息搜索——按正文/推理/工具名/参数/结果过滤 + 命中/总数计数；③ChatTabs 过期注释清理。零新增绑定（546，纯前端）。tsc/eslint 0、vitest 872/872（+2）、Go cached 全绿、drift PASS（546）。详见 releases/v4.21.0.md |
| **v4.20.0** | 2026-08-31 | 剩余收官：①子代理完整 transcript 查看器（+1 绑定 GaeaSubagentTranscript：读 `<sessionDir>/subagents/<ref>.jsonl` 全量消息，ref 安全校验防穿越；前端 Agent 网络详情面板「查看完整 transcript」消息流）②轨迹 Overview 投影 + 轮次跳转 + 收起全部/展开全部（概览条每轮一柱，柱高∝记录密度，工具高亮/报错标红，点击平滑跳转并展开）③迁移/兜底会话趋势补齐（ToLogEntries 每回合合成 request_header——真实 system + 实际工具名集合，顺序同运行期；contextview 回合末估算关闭，estimated 记录 + 前端「估算构成（无用量记录）」诚实标注，不伪造用量）。绑定面 545→546。Go 全量 0 FAIL（+2 用例）、tsc/eslint 0、vitest 870/870（+3）、drift PASS（546）。详见 releases/v4.20.0.md |
| **v4.19.0** | 2026-08-31 | 看板收官：①上下文浏览器——contextview 折叠补全系统/工具节点（request_header 构成变化才入 nodes，每步重复不刷屏），前端 ContextBrowserCard（活跃/归档双页签 + 六分类过滤 + 节点行可展开，归档=被压缩移出带「已压缩」标记），页脚占位整行移除；②`/context` 命令——GaeaCommands 内置 + i18n（zh/en）+ classifyComposerCommand 拦截切上下文标签；③Agent 网络节点点击→子代理详情——AgentNetworkCard 增 sessionPath（App 注入 currentSessionPath），SubagentRuns 按任务前缀匹配固定详情面板（状态/模型/工具数/活动行/最后回答）。零新增绑定（545）。Go 全量 0 FAIL（+1 用例）、tsc/eslint 0、vitest 867/867（+4）、drift PASS（545）。详见 releases/v4.19.0.md |
| **v4.18.0** | 2026-08-31 | 看板补全：①文件活动时间线（上下文标签新卡）——contextview 折叠新增 FileActivity（工具参数确定性提取路径 + 工具→动作白名单，screen_capture 结果补记，bash 等无法确定性取路径者诚实不造数；同轮同步骤同路径合并、上限 200）；②增量（Delta）模式启用（趋势图「增量」按钮去灰置，绿=净增·红=净减图例，柱色改全站语义色）；③运行中实时刷新（新 hook useLiveReload 订阅事件流：节流 1200ms + turn_done 立即刷新，轨迹/上下文/Agent 网络统一接入）。零新增绑定（545）。Go 全量 0 FAIL（+2 用例）、tsc 0、eslint 0、vitest 22/22、drift PASS（545）。详见 releases/v4.18.0.md |
| **v4.17.0** | 2026-08-31 | 轨迹上下文接通：办公板块「轨迹」「上下文」标签此前是空壳（前端完整，根因是事件日志数据源缺省关闭）——本刀 `config.EffectiveLogFormat` 缺省 "event"（显式 legacy 可退回）+ gaea_handler 注入生效值 + boot 创建 EventLogSink；`session.ReadEntriesFor` 旧会话读端兜底（无日志时从 legacy 会话投影折叠条目，纯读不落盘）；`ToLogEntries` 迁移产物带 turn_started/turn_done 回合边界 + 轨迹折叠兼容 assistant_message（内嵌工具调用展开合并）；EventLogSink.Close 挂进 Controller.Cleanup（Windows 文件句柄释放）。零新增绑定（545）。Go 全量 0 FAIL（+6 用例）、tsc 0、看板 vitest 12/12、drift PASS（545）。详见 releases/v4.17.0.md |
| **v4.16.0** | 2026-08-31 | 四刀并行：①persona 侧离线裂缝收口（gaea_whisper_causal/retell/whisper_handler 三处 featureModel→routeModel，全局离线对轻语链路生效）②浏览器键盘级 Input（browser_press 第 11 工具：dispatchKeyEvent+组合键+text）+ iframe 内交互完整实现（read/click/type 加 frame 参数，getFrameTree→createIsolatedWorld，真机验证全通）③Verifier 通道 B 结果进前端（Verdict 增 channelBRatio/Pages/Artifacts + 证据卡「视觉复核」行 + 产物目录按钮）④晨报深度预装配（BuildMorningPreloadBlock 纯函数→sysprompt 注入「工作记忆晨报」块，morning_preload 键默认开，play 不注入）。零新增绑定（545）。Go +20 测试、vitest 861/861、drift PASS（545）。详见 releases/v4.16.0.md |
| **v4.15.0** | 2026-08-31 | 聊天路由归位：plain 聊天三处 `featureModel("chat")` → `routeModel("chat")`——全局离线模式对 plain 聊天生效（修复「总闸不总」裂缝，用户功能绑定语义不变）+ 无绑定时全局/兜底与 persona 一致 + model.route 事件补齐；「由谁回答/为何/花了多少」回显——导出 EstimateCostCNY（本地/未知恒 0，USD 折算 CNY）+ chat done 帧/ChatSend 返回加 answered_by + 前端 AnsweredByLine 消息底部小字（费用 ≤0 隐藏，旧事件向后兼容）。零新增绑定（545）。Go +7 测试、vitest 859/859、drift PASS（545）。详见 releases/v4.15.0.md |
| **v4.14.0** | 2026-08-31 | 三箭并行：①浏览器续刀——空闲 TTL 自动关停（默认 10min + GAEA_BROWSER_IDLE_TTL env，到期自动回收进程+清临时 profile，browser_* 自动重拉闭环）+ 多标签页（tabs map + ListTabs/NewTab/SwitchTab/CloseTab，切 tab 旧 refs 诚实失效）+ 新工具 ×3（browser_tabs/browser_new_tab/browser_switch_tab）+ browser_close 可选 tab_id；②做梦 2.0 主动预取 MVP——纯本地晨报（BuildMorningBrief 纯函数零 LLM + GaeaMemoryMorningBrief 绑定 545 + 首页 MorningBriefCard 仅 work 渲染，绑定面 544→545）；③Verifier 产品化——证据卡三步展开（声明↔实况 diff × GaeaPreview 现取 + 操作回放时间线 + 无基线回滚禁用），纯前端零新增绑定。Go +25 测试、vitest 852/852、tsc/eslint 0、drift PASS（545）。详见 releases/v4.14.0.md |
| **v4.13.0** | 2026-08-31 | 自动操作·浏览器：internal/gaea/browser CDP 客户端（msedge 三段式定位 + 隔离临时 profile + Job Object 绑定 + Ensure 幂等/失联自愈 + URL 白名单 http/https，gorilla/websocket 零新增依赖）+ 7 个 browser_* 工具面（navigate/read/snapshot ref 机制/click/type React 兼容/scroll/close，work 空间，envelope 结构化返回）+ 权限门（只读档恒放行/写档弹卡可记忆 + subject url 键固化为窄规则）+ 事件留痕全链通用。真机 headless Edge 实测 PASS；零新增绑定（544）、前端零改动。Go +25 测试、vitest 825/825、drift PASS（544）。详见 releases/v4.13.0.md |
| **v4.12.0** | 2026-08-31 | 成本透亮：GLM 价格表补全（原仅 glm-4.7，GLM 用量未被计价；官方 USD 价 + usd_cny_rate 折算，免费档计 0，未核实者诚实不入表）+ 编码套餐积分口径（/api/coding/ 调用 billing_mode=coding_points 不按 token 计价，聚合 glm@coding 单列）+ 模型别名注记（coding 家族 4 条官方自动切换，glm-5.2→glm-5.3 等，前端「自动切换」标记 + 记账归一）+ GLM 目录数据驱动（内嵌 JSON 逐字锁定 + glm_catalog_path 覆盖文件 mtime 热更新 + 坏 JSON 回退）。零新增绑定（544）。Go +5 组测试、vitest 825/825、drift PASS（544）。详见 releases/v4.12.0.md |
| **v4.11.0** | 2026-08-30 | GLM 全模态纵深：生图后端接通（官方 images/generations，只发官方 schema 字段，URL 转 data URL，错误体原样透出）+ 官方双端点切换（std=按量/coding=编码套餐，SetGlmEndpoint 只收官方常量，绑定面 543→544）+ 生图模型目录补全（glm-image/cogview 系 18→22）+ glm-5-turbo 误分类修复。Go +15 测试、vitest 821/821、drift PASS（544）。详见 releases/v4.11.0.md |
| **v4.10.0** | 2026-08-30 | GLM 引擎上线（智谱 OpenAI 兼容 paas/v4：Bearer Key 全链路、chat ping 验证、官方文档锚定静态模型目录 glm-5.3 旗舰等 18 模型、地址防呆三防线）+ 做梦 2.0 蒸馏真实合并（确定性重复检测+可逆归档合并，路线图 T0 第一刀）+ 工作人设收口（professional 节奏豁免+办公秘书人格+[SPLIT] 出口净化+搜索触发收窄）+ 多跳因果链（≤2 跳「导致」链）+ Verifier 通道 A 引用级深化（opsJson 随卡+声明↔实况比对）+ Herdsman CLI 错误透明化。绑定面 540→543；vitest 818/818；drift PASS。详见 releases/v4.10.0.md |
| **v4.9.0** | 2026-08-30 | 星枢首页·轻语记忆纵深（基线 v4.8.3 + 15 提交，绑定面 535→540）：轻语记忆回放系列（GaeaWhisperEpisodeReplay 按情节重建原始对话 + GaeaWhisperMemoryRetell LLM 人格重述 + 时间锚点「重访那一天」写路径与纪念日回放）+ 图谱三维度（情绪着色/hermes.db V14、确定性因果三元组入图、event_chain 关联边琥珀虚线）+ 跨事实因果推断 GaeaWhisperCausalExplain「为什么」（确定性证据 + 人格口吻 LLM 人话化，无证据零 LLM 诚实回退）+ 首页重构「星枢指挥所」（启动默认首页 + 两段式启动动画 BootSplash + 命令条/遥测细条/Bento 能力矩阵，i18n 三语 17 键）+ 工程化（build.bat 冒烟自动化、持久化原子写、XlsxPreview 行虚拟滚动、VoiceStart realtime 门、微信识图两修）。Go 全量绿、vitest 818/818、tsc/eslint 0、drift PASS（540）。详见 releases/v4.9.0.md |
| **v4.8.3** | 2026-08-30 | 微信图片双向（真协议实装，三方印证：本机抓包解密+hermes-agent+openilink SDK）：出图回推 getuploadurl→AES-128-ECB→CDN 密文上传→image_item 卡片+caption 补发（真机 delivered）+ 发图识别 type=2/aeskey 解密下载→魔数终审落盘（真机两连发通过）+ 识图模型升级多模态 Qwen3.6-35B 优先（手写体强，与聊天同体零额外显存；PaddleOCR→MinerU→OvisOCR2 降兜底）+ 身份类问题跳过联网搜索（乱字母根因）+ 关键坑锁死（type=2/aes_key=base64(hex)/上传域与扫码 baseurl 无关）。Go 全量绿、零新增绑定（535）、前端零改动。详见 releases/v4.8.3.md |
| **v4.8.2** | 2026-08-30 | 欠账收尾：权限升级请求（request_permission 工具 + 硬纪律三闸——deny 先行/hardAsk 拒升级/批准只写 glob 规则表不绕闸门 + 审批卡 request 形态 reason 原文块 + 会话 glob 规则表补全，零新增绑定）+ 竞态/flake 全治理（Cancel 被 succeeded 吞掉的收尾窗生产竞态 ×10 压力绿、stubGate 测试桩加锁、filewatch 风暴测试时序根治、ProgrammingPage 显式 5s 超时）+ Realtime S2 事件环骨架（Resample16kTo24k 纯函数 + 事件常量 +7 + TurnControl 可选接口 + voice_manager 事件泵/barge-in 三联/24k WAV 冲洗 + 前端 PCM 推送死门打通 + 五重降级护栏，真机欠账如实记账）。Go 全量绿、vitest 807/807、tsc/eslint 0、drift PASS。详见 releases/v4.8.2.md |
| **v4.8.1** | 2026-08-30 | 欠账清尾：全局离线模式设置 UI（SecurityPanel 总闸段 + GaeaGet/SetOfflineMode 绑定 533→535 + shelf 内存同步；与敏感域/办公本地优先的叠加关系文案化）+ Realtime S1（realtime 三键落盘——provider 仅 openai、Key 走 DPAPI 密文仅本机可解；initVoice 注入接线位置测试守护；VoiceApplySettings/GetSettings 扩三键——明文 Key 永不出后端，测试断言无泄漏；VoiceSettingsPanel「实时语音（实验）」入口段：供应商/模型/密码框 Key/保存回读 hasKey）。Go 全量绿、vitest 803/803、tsc/eslint 0、drift PASS。详见 releases/v4.8.1.md |
| **v4.8.0** | 2026-08-30 | 全面铺开·触点纵深（六线并行调研→七刀，多子代理分工）：读屏纵深（多显示器 Monitors/CaptureArea + 「第N屏/主屏/副屏」序数解析 + OCR 本地摘要朗读 + 截图留档）+ intent LLM 兜底分类器（默认关：白名单 navigate/status/read_screen + 0.75 置信门 + 2s 硬超时，dryRun 恒不调用，宁漏勿误）+ 生图产物 CardPath 接通（微信回推数据源）+ iLink 微信通道离线收敛（限频/4KB 截断/多媒体上限/SSRF+魔数下载防线/图片→vision 识别管线/防御解析矩阵/SendFileCard seam）+ 全局离线模式总开关（EngineType.IsLocal + routeModel 三步云过滤，跨版欠账清账）+ 成本知识图谱可视化（costref.BuildGraph 树聚合/条目展开双视角 + CostGraphView 零依赖 SVG，成本库第 8 模块，绑定面 532→533）+ 实时语音 Realtime S0 铺底（internal/realtime seam + VoiceHealth realtimeReady，S1/S2 留欠账）。Go 全量绿、vitest 800/800（146 文件）、tsc/eslint 0。详见 releases/v4.8.0.md |
| **v4.7.0** | 2026-08-30 | 命令面板接内核·读屏（S4.6 完整收口）：GaeaRouteIntent(text,dryRun) 绑定（531→532；dry-run 预览-确认制——搜索词不是整句指令入口，命中出卡点执行才真跑，宁漏勿误）+ SearchModal 指令预览卡（动作标签+预览语+回执内联，导航类复用 INTENT_NAVIGATE 切板块收面板）+ 真·Ctrl+K（MainLayout 全局快捷键，办公板块让位工作台自有 CommandPalette）+ 屏幕感知能力「读一下屏幕」（read_screen 窄规则 + 截屏→临时 PNG→既有 OCR 链→300 字截断回传，语音 TTS 朗读/面板内联/微信回推三入口免费受益，失败诚实回执）。Go 全量绿、vitest 796/796（145 文件）、tsc/eslint 0。详见 releases/v4.7.0.md |
| **v4.6.1** | 2026-08-30 | 微信统一路由·规范包机制·归因对标：微信消息接统一意图路由（routeIntentWithResult，提醒之外生图/状态/导航全命中）+ iLink 图片消息协议第一刀（image_item 防御解析 + 发图提示行）；规范包机制化（Checker 注册表 + GB/T 9704 红头 + 造价工程表式双检查器，OfficePanel 按规范包分组）；成本归因对标（项目明细 vs 参考指标 P25/P75 带宽，差幅等级/贡献金额/主因 TopDrivers，参考池排除自身防自对标；绑定面 530→531，FiveCalcPanel 归因区）。Go 全量绿、vitest 791/791、tsc/eslint 0。详见 releases/v4.6.1.md |
| **v4.6.0** | 2026-08-30 | 执行审计后第一轮补课·双空间收尾+纵深：红线三条全接线（记忆注入 InSpace 视图收窄——work 只注入 work/play 只注入 play；[tasks] 配置段启用按空间分账 {work=1, play=1}+价格抓取优先；onTaskEvent 订阅层空间过滤推广）；前端治理收尾（keepAlive 裸轮询 8 处门控 + CSS 真硬编码 token 化）；纵深三件（Mood→TTS 连续韵律闭环——「听得出她今天低落」；Verifier 通道 B 真视觉 diff（soffice→pdftoppm→像素差异率）+ 失败回 Plan（xlsx 一键重新规划）；询价异常检测分级+线性回归价格预测+OCR 报价单自动幂等入询价库）。绑定面不变（变参兼容）；Go 全量绿、vitest 791/791、tsc/eslint 0。详见 releases/v4.6.0.md |
| **v4.5.0** | 2026-08-30 | 规划修订(§10.4a)后第一刀·指令中枢：统一「意图→能力→结果回传」路由内核落地触点层「同内核多入口」——intent 纯函数解析包(导航/生图/状态/提醒，宁漏勿误纪律) + 能力执行层(navigate 事件/generate_image/status/reminder 复用离线代办) + 语音指令通路(JARVIS 一档：voice 回调分流，命中即能力执行经 TTS 播报，voice 包零改动) + 前端 INTENT_NAVIGATE → navigateBoard 自动切空间；绑定面 530 零新增；Go 108/108、vitest 789/789。详见 releases/v4.5.0.md |
| **v4.4.0** | 2026-08-30 | v4.4「触点」一期·微信遥控器：离线代办——微信对助手说「提醒我 30分钟后 喝水 / 明天早上9点 开会」→ 中文时间解析（相对/日期+段词/裸时刻，中文数字进位）建提醒 → 20s ticker 到点经微信 Push 回推（失败重试 ≤5 次，JSON 持久化重启恢复）；weixin.Server 主动推送通路（最近活跃会话记忆）；WeixinPage 书房板块页落地（扫码绑定流/通道状态/提醒面板，板块 inMenu=true 进 rail 与首页左翼）；绑定面 525→530（WhisperWeixin*/WhisperAssistant* 转正 + WeixinReminder* 新增），spaceBindings 235→247 全归 work；vitest 789/789、Go 107/107。详见 releases/v4.4.0.md |
| **v4.3.2** | 2026-08-30 | 首页「双翼·中庭」重构 + 空间导航收敛：中庭语音+打字一体对话条（VoiceChatText 共用管道 + 放大 orb 磁吸核心，hero 让位细眉）；左翼「书房」2×2 格（办公/造价/记忆/模型）；右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊编程独立窗口入口；命名 工位→书房、乐园→庭院；移除 rail 空间切换按钮，navigateBoard 按板块自动切空间；搜索 scope 文案同步。绑定面不变、vitest 789/789、Go 全量绿。详见 releases/v4.3.2.md |
| **v4.3.1** | 2026-08-30 | v4.3 后续小步：主动关心定时推送频控（ticker 四信号：频控/作息尊重/生日祝福/门控合成 → gaea-whisper-proactive 事件 + GaeaWhisperProactiveConfig 配置绑定）+ 创作间世界模型面板（设定页维度化编辑器 6 维度卡片 + 伏笔登记表状态流转/回收率 + 一致性检查三类告警）+ 角色参考图（SchemaV2 reference/gallery 两列 + CharacterGeneratePortraitWithRef img2img 参考槽 + 前端参考图管理）+ 朗读情绪 UI（9 情绪选择器 + 会话情绪跟随 + TTSSpeakBase64WithParams）。绑定面 525、spaceBindings 235、vitest 789/789、Go 118/118。详见 releases/v4.3.1.md |
| **v4.3.0** | 2026-08-29 | v4.3 乐园做深（阶段 3+ 第二发）：会客厅关系记忆（三表持久化闭环 + ReseedAssociationGraph + QuerySubgraph 多跳子图召回 + WhisperGraphPanel SVG 图谱）+ 主动关心（GaeaWhisperProactiveNow 评估 + 前端「轻语先开口」）+ 情感语音（TTS SynthesizeWithParams 参数扩展 + cosyvoice 修复 + 情绪→参数映射 + 长期心境维 Mood）+ 创作间图文联动（章节配图复活死绑定 + GaeaGenerateBookCover 3:4 书封落 play exports）。绑定面 522、spaceBindings 233、vitest 769/769。详见 releases/v4.3.0.md |
| **v4.2.0** | 2026-08-29 | v4.2 造价 AI 化（阶段 3+ 领域包第一发）：AI 组价（PriceBand 价格带 P25-P75+置信度+证据链 → GaeaCostCompose 相似检索+LLM 人材机拆解 → 一键回写，前端 ComposeModal）+ 询价飞轮（costinquiry 四源归一数据点 + 到期预警 + 调差建议 + 前端询价视图）+ 五算对比（coststage 估/概/预/结/决阶段值 + 对比/偏差三档 + 前端 FiveCalcPanel）。绑定面 517、spaceBindings 229、vitest 759/759。详见 releases/v4.2.0.md |
| **v3.9.0** | 2026-08-29 | 双空间壳（阶段 2 收官：两视图+空间切换持久化+双首页+删旧 10 板块导航+bridge 三门面+types 全量迁移+151 hex token 化+i18n 决策+页面迁入 P1 对话流）+ v4.1 办公信任链（证据链 ChangeRecord/Journal + Verifier 双通道复核 + 基线回滚冲突保护 + GB/T 9704 红头规范体检）。绑定面 506、spaceBindings 218、vitest 738/738。详见 releases/v3.9.0.md |
| **v3.8.0** | 2026-08-29 | 双空间内核（工位/乐园隔离：会话/记忆/任务/产物/模型/工具/权限/护栏全按空间装配，space.mode 可回退）+ 质量地基（并发加固/Registry 锁/gate 原子化/retry_until 门控/edit_file 工具层/前端虚拟化与轮询门控/CI -race）+ 长期规划定稿（编程板块保持独立 DSH 窗口）。绑定面 502、Go 115 包、vitest 全绿。详见 releases/v3.8.0.md |
| **v3.7.0** | 2026-08-29 | 办公蒸馏 codex 清单收官：C2 记忆引用可追溯（注入行引用键 [MEM:name] + 回传解析 Touch + 前端引用徽标弹层展示记忆详情/沉淀来源）；C4 审批决策族（deny/abort 拒绝三分 + approval_timeout_secs 超时 + persist_allow「始终允许」回写 [permissions].allow 策略文件，hardAsk 降级不回写，GaeaApprove 重构决策串五值）；C9 任务输出事件化（gaea-task 事件推输出尾回放，dock 事件即推、轮询兜底）；C5 运行状态行窗口占用百分比 + 压缩前预警（75%/90% 两档）；C6 项目说明文件 32KB 预算 + .gaea/AGENTS.md 发现；C3 自动做梦 no-op 指纹去重。绑定面 499、vitest 681/681、Go 114/114。详见 releases/v3.7.0.md |
| **v3.6.0** | 2026-08-29 | 办公文件编辑审阅制 + 本地优先 + 对话面减负：xlsx AI 编辑两段式（GaeaXlsxPlanEdit 临时副本试运行+单元格级 diff → 批准 → GaeaXlsxApplyEdit，新增 set_style 叠加/合并/列宽）；xlsx 原生图表嵌入工作簿（非截图）；PDF 统一出口（GaeaConvertToPdf，LibreOffice 无头 + md 经 docx 中转 + 顶栏导出）；办公功能级 AI 本地优先（routeOfficeLocal + 安全设置开关，主 agent 不受影响）；运行中插话（GaeaSteer + event.Steer）；回退方案模式 v1、撤下任务目标/验收清单（GoalCard + GaeaRequirement 系）、用户消息 Codex 式收敛 + 超长折叠；修复 whisper 关机排水丢任务 / 空切片 null 崩溃。绑定面 499、vitest 669/669、Go 114/114。详见 releases/v3.6.0.md |
| **v3.5.0** | 2026-08-28 | 办公对话区标签页 + dsh-context Go 移植：对话窗口上方 [对话\|轨迹\|上下文] 三标签；request_header 事件（模型可见必入日志的请求头落点）；上下文标签（contextview 包：六分类组成 + 原生 SVG 趋势图 + 事件流 + 步骤详情，usage 锚定与顶栏同源）；轨迹标签（trajectory 包对齐 DSH ui-trajectory 扁平事件账本：header change/工具 parentId 嵌套/轮间压缩/ask+approval + 检查器/搜索/统计）；Agent 网络（子代理树 + 节点 token 环 + subagents meta 富化）。绑定面 503、vitest 668/668、Go 114/114。详见 releases/v3.5.0.md |
| **v3.4.0** | 2026-08-27 | 记忆统一层第一刀：统一检索后端收口（GaeaUnifiedSearch 增三脑/文件语义两组，hub 搜索 4 绑定前端拼装→1 绑定）+ 生命周期产品化（归档 tab 分页修复 + Unarchive 恢复 + retentionDays 展示）+ 修复漂移脚本单条差异静默放行 bug。eslint 0/0、tsc 0、vitest 654、Go 112/112、绑定面 498。详见 releases/v3.4.0.md |
| **v3.3.0** | 2026-08-27 | 质量收敛：eslint 存量 warnings 366→0（配置显式化 `^_` 前缀/空 catch/常量导出 + 死代码清理 56 处 + exhaustive-deps 40 处含两处 TDZ 重排 + 混合导出显式声明 + 冗余 @ts-ignore 移除）；flaky 治理（filewatch 超时 3s→5s + CI 测试失败重试）；releases/README.md 历史乱码重建；前端性能体检。eslint 0/0、tsc 0、vitest 652、Go 112/112、绑定面 497。详见 releases/v3.3.0.md |
| **v3.2.1** | 2026-08-26 | 工作区内联编辑（C5，蒸馏清单收官）：GaeaWriteFile（相对路径/写根/文本白名单/大小四重校验 + 原子写）+ 文件预览编辑模式（脏标记/Ctrl+S/保存状态机/保存后重读）。vitest 652、绑定面 497 方法。详见 releases/v3.2.1.md |
| **v3.2.0** | 2026-08-26 | 任务可见性：C1 任务实时输出（Progress.Output 环形缓冲 + GaeaTaskOutput + 任务中心输出 dock 2s 轮询尾随滚动 + stopping 结束态细分）+ C2 子代理活动行（lastText/lastTool「正在…」实时线）+ 数据备份计划路径统一修复。vitest 647、绑定面 496 方法。详见 releases/v3.2.0.md |
| **v3.1.1** | 2026-08-26 | 造价数据库闭环补齐：测算项目 UI（明细编辑/版本快照/恢复/沉淀回库）+ 造价参考分位数对标 + 复盘笔记（后端 v3.1.0 已就绪，纯前端 + 导航同步）+ C4 选区转对话（选中正文→转为提问插入输入框）+ 仓储卫生（删根目录临时脚本/旧 exe、releases 索引补全）。vitest 643、绑定面 495 方法零新增。详见 releases/v3.1.1.md |
| **v3.1.0** | 2026-08-26 | 造价数据库一级板块（综合单价架构 + 测算项目/造价参考/复盘笔记 + 手册整本导入 + 数据自愈）+ 办公蒸馏（右侧面板会话持久化/活动角标/预览队列 chip、文件工作台资源管理器）+ 办公初始化死锁修复；绑定面 495 方法、vitest 630。详见 releases/v3.1.0.md |
| **v3.0.8** | 2026-08-17 | 办公板块「表格可交付 + 会话产物打包 + 多智能体分工」+ 界面收敛：表格选中区域→一键图表（柱/线/饼，图表 ▾ 菜单）+ 会话产物一键打包 Zip + 产物缩略图增强（xlsx 迷你表格/md 文本摘要）+ 子代理「分工」面板（状态/任务摘要/回答）；右侧面板 Tab 收敛为 4 主标签（文件/成果/运行/分析）+ Excel 工具栏按上下文收敛（用户决策：不堆功能、聚焦 Word/Excel）。vitest 605 通过、绑定面 480 方法。详见 releases/v3.0.8.md |
| **v3.0.7** | 2026-08-17 | 办公板块文件交互体验（调研 P0-P2 落地）：非图片附件 chip 化 + 行内文件 chip 视觉统一 + 最近文件快捷区 + 多文件预览队列（←/→ 导航）+ 产物版本时间线（vN 徽标）+ 大工具输出有界预览 + 附件上下文占用透明化；内置 prompt 模板兜底（SetPromptFS，exe 单文件分发）。vitest 587 通过。详见 releases/v3.0.7.md |
| **v3.0.6** | 2026-08-16 | 编程板块桌面内嵌工作台（iframe 内嵌 DeepSeek Harness Web + 启动引导：前置条件检查/日志/启动动画）+ 运行中工具栏移入顶栏（仅编程板块显示）；办公板块会话回退/分叉/回退点 + 右侧 Tab 清单化 + 会话统计回填 + mock 场景补全。详见 releases/v3.0.6.md |
| **v3.0.5** | 2026-08-16 | 首页任务指挥中心改版（参照 DeepSeek 首页风格）：Hero 左文右卡 + AI 状态细条 + 「全部模块」办公大卡 + 8 卡 4×2 网格；修复设置卡被遮挡、PageRegistry 补注册编程板块；语音晶核取消粒子星云恢复发光球。详见 releases/v3.0.5.md |
| **v3.0.4** | 2026-08-16 | 办公板块能力加强（任务目标→验收清单 + 目标/待办卡拆分 + 自动追踪）+ 小说阅读体验重构（排版/书签/划线/AI 伴读/朗读同步/全文搜索/导入成品小说）+ 角色库闭环 + 导出合并进阅读面板。详见 releases/v3.0.4.md |
| **v3.0.3** | 2026-08-16 | 小说（设定页「应用到设定」/创作面板字号/默认 story-deslop/剧情构思后台化）+ 模型中心（统计与资源/AMD 显存识别/失败可见+重试）+ 绘梦模板库 19 类 231 个 + ComfyUI 实时进度 + 角色剧照远程 URL 本地化。详见 releases/v3.0.3.md |
| **v3.0.2** | 2026-08-16 | 移除运行态看门狗（反复误杀办公板块回合，用户决策）；模型流空闲超时与回合 panic 恢复保留兜底。详见 releases/v3.0.2.md |
| **v3.0.1** | 2026-08-15 | 小说板块 UX/UI 重构：书架卡片化（封面渐变条/阅读进度条/搜索排序/继续阅读联动）、阅读页单条 chrome + 沉浸阅读模式（居中限宽衬线排版）、AI 控制台 v3 玻璃面板化、创作页令牌化与生成发光统一、全板块零硬编码色值；修复 app_info.go 版本漂移（2.40.0→3.0.1 三处统一）。详见 releases/v3.0.1.md |
| **v3.0.0** | 2026-08-15 | 星枢 Constellation OS · UI 革命性重设计（V3.0 首发）：壳层革命（左侧指挥轨道 + 顶部轨道条 + 底部遥测轨道）；10 板块统一 3 分区工作台；Luminous Glass 2.0 令牌体系；发布前打磨（聊天模式条移入顶栏、输入框两级重设计、首页全屏适配）+ 7 个 CSS 文件注释 `*/` 根因修复（面板「只有上半截」）。详见 releases/v3.0.0.md |
| **v2.40.0** | 2026-08-15 | 3.0 架构主线 Wave 4「Step 3 收官」：semantic_search 工具注册（死代码恢复为办公 agent 可用工具）+ BalanceKind 从 ProviderEntry 贯通（`balance_kind` 配置项，切换余额后端只改配置，未知 kind fail-closed）+ ModuleLauncher 清单化（subscribeBoards 订阅活动清单，后端并入 knowledge 后首页启动器自动多出「知识库」卡） |
| **v2.39.0** | 2026-08-15 | 3.0 架构主线 Wave 3：Step 3b LLM Seam（LLMProvider + NewLLM 配置驱动）+ Step 3c OCR/ASR/TTS Seam（三类注册表 + GAEA_OCR_ENGINE 驱动）+ Step 3d 分类单源化与 8 处硬编码注册表化（websearch/embed/rerank/vision/markitdown/billing）+ 前端 GetBoardManifests 接线（normalize 差集 + KnowledgePage）+ gaea.toml 新配置段 |
| **v2.38.0** | 2026-08-15 | 3.0 架构主线 Wave 2：Step 1 app 层接线（会话事件日志「日志即真相」运行时闭环：Resume→Restore / Save→日志 / 模型调用前 fail-closed 检查点）+ Step 2 板块 Manifest（board 包 10 板块 + module_registry manifest 驱动 + GetBoardManifests + MainLayout 附 B 12 硬编码点清单化 + PageRegistry + events 常量表 + label 单一来源）+ Step 3a Image Seam（图片后端注册表化：openai/comfyui 自注册 + 401 单次重试守卫）。绑定面 464 方法 |
| **v2.37.0** | 2026-08-15 | 正确性纵深收官：T7-2 可见性收口 / T7-3 名实相符 / T7-4 前端性能收尾 + 3.0 Step 0 修债 + Step 1 会话事件日志（append-only 日志 + 投影 + checkpoint + 迁移 + GaeaHistory 黄金测试） |
| **v2.34.0** | 2026-08-15 | 正确性纵深第一刀：轻语会话并发安全（深拷贝/-race 实证修复）+ 任务调度器竞态 + TCCA 指标聚合收敛 + AI 客户端状态与重试 |
| **v2.33.0** | 2026-08-14 | 阶段 6「质量收敛」第十刀·前端收敛：巨型文件拆分（ChatPage 1022→370 / ImageGenPage 911→310 / CapabilitiesPanel 803→178 / Composer 786→406 / mock.ts 1563→50 按域拆 10 文件，11 no-op 落实）；any 清零（no-explicit-any 升 error 进 CI，315→0）；绑定漂移检查恢复（gen_bindings -names + bindingNames.ts + bridge.ts 双向类型守卫 + CI 步骤）；mock 契约对齐（RetrievalEvalRun 12 条真实查询集）；Sidebar react-window 虚拟滚动 + CostLibraryView memo 化 + useDebouncedValue；删除 api/bridge.ts 双桥接合流；测试 354→361 |
| **v2.32.0** | 2026-08-14 | 阶段 6「质量收敛」第九刀·辅助合集名实相符：微信生命周期（Stop 幂等/Start 重启/过期钩子退出空转）+ wxToken DPAPI 加密迁移 + 4 张死表 V13 删除；OCR 兜底名实相符（超时杀进程树 + 单图 tesseract 降级 + 删「Windows 原生 OCR」文案）；配置原子写（fsync+rename）+ 损坏备份恢复；CosyVoice 路径/端口可配置 + 启动退避重试；token 改 header（服务端去 query 兜底 + 前端 fetch 流式 SSE） |
| **v2.31.0** | 2026-08-14 | 阶段 6「质量收敛」第八刀·记忆生命周期与审计：dream 写入审计（决策成文 DREAM_WRITE_POLICY.md，SaveDreamFacts 带 source 落 dream-audit.jsonl）；facts 归档生命周期（90 天保留硬删 CleanupArchived + ListArchivedPaged 分页 + 新绑定 GaeaMemoryCleanupArchived/GaeaMemoryArchivedList + 前端清理按钮 + purge-audit 溯源审计）；索引截断按边界（4096 字节口径统一 + 行边界 + markdown 链接保护，6 测试）；GraphView/WhisperMemoryLibrary 补测 13 用例 |
| **v2.30.0** | 2026-08-14 | 阶段 6「质量收敛」第七刀·小说导出与原子性：export 整改（失败计数/作者元数据/统一分段器/世界观对齐/保留名消毒，13 测试）；生成中断（CancelCreateChapter 绑定 + 章节互斥 + 取消保留部分正文）；writeFileAtomic 落盘原子化；模板 {word_count} 占位符；CreatePage 791→288 行 + 停止按钮 + 判别联合（18 vitest） |
| **v2.29.0** | 2026-08-14 | 阶段 6「质量收敛」第六刀·模型中心密钥与 UI：refresh_token DPAPI 加密（旧明文自动迁移）；汇率配置化（usd_cny_rate 键 + 新绑定 + ModelPanel 输入）；probe 告警文案修复；ModelCenterPage 顶层 useState 42→3（5 hooks 下沉）+ XAI_VOICES 单源；refreshLocalModels 竞态守卫 + 定时器随分类重置 |
| **v2.28.0** | 2026-08-14 | 阶段 6「质量收敛」第五刀·轻语测试与可观测：补测试 146 用例（emotion_fusion 100% 覆盖 + 记忆管线 ≥92%）；异步记忆写错误可观测（WriteErrors 计数 + 四类 phase）；成人模式决策成文（ADULT_MODE.md + 删除死接口 WhisperSetAdultMode）；GetDatabase 签名收敛 (db, err) + PRAGMA 单源 + FTS 失败日志；陈旧占位清理 4 文件 |
| **v2.27.0** | 2026-08-14 | 阶段 6「质量收敛」第四刀·绘梦链路真实生效：取消真实中断 ComfyUI（/interrupt + 本地取消标记）；flux 真实工作流（映射表 + 静默降级消除）；历史图片 file_path 恢复（分级存储 + 回填）；尺寸严格校验（64-2048）；端口命令注入修复（netstat 参数数组）；核心链路 httptest 31 用例 |
| **v2.26.0** | 2026-08-14 | 阶段 6「质量收敛」第三刀·对话流可靠：流订阅竞态修复（订阅先行 + 30s 无帧超时 + sending 五路复位）；落库错误透传（ChatTopicsList/ChatMessagesList 返回 error + 流式 error 终态）；语音消息持久化（新绑定 ChatAppendMessages 单事务）；旧数据迁移一次性化；导出 Markdown 转义 + 文件名消毒；导入事务化 |
| **v2.25.0** | 2026-08-14 | 阶段 6「质量收敛」第二刀·办公引擎正确性：PDF 页数统计/分页过滤修复（/Type /Pages 干扰 + OCR 绝对页码）；TurnResult 语义收紧（blocked/suppressed 计入）；落地后端看门狗（墙钟 10min/停滞 30s，工具执行与审批等待豁免）；运行中 Send 限长队列（8 条 FIFO 排空）；子代理禁写清单注册表化（PersistWrite 标记推导）；TCCA 97%/evidence 91% 覆盖率；docmd.go 1521 行拆 4 文件 |
| **v2.24.0** | 2026-08-14 | 阶段 6「质量收敛」第一刀·基础层可靠性：SSE 流式加固（64KB 行上限消除/连接与 5xx 退避重试/60s 空闲超时）；AI 流量接入系统代理（localhost 强制直连）；前端错误可见性（BridgeError 归一化 + 14 处静默 catch 改记录）；后端吞错清理 + 桥接 token 日志脱敏 |
| **v2.23.0** | 2026-08-14 | 阶段 5「运行纵深」第三刀·进料与质量：成本库 PDF/图片报价单本地识别入表（OCR+本地视觉归一化，sensitive_local 不出机）+ 供应商比价卡；统一检索入口（关键词+跨库语义单框两组）+ 检索质量受控测评（Recall@10≥0.8 门槛，模型中心「检索质量」区） |
| **v2.22.0** | 2026-08-14 | 阶段 5「运行纵深」第二刀·速度与韧性：本地模型调度纵深（保活 keep-warm 轻量探针防卸载、启动自动预载、换模预计等待提示、KV 缓存命中率并入调用统计）；中断续跑（会话「未完成」徽标 + 恢复注入中断摘要、轻语任务计划持久化与恢复入口） |
| **v2.21.0** | 2026-08-14 | 阶段 5「运行纵深」第一刀·调度与异步化：通用任务调度器（持久任务表/进度事件/取消/自动重试/手动重试/重启续跑，价格抓取与文件索引全异步化 + 办公右栏「任务中心」Tab）；fsnotify 实时文件监听（新文件秒级进语义索引，10 分钟轮询降级兜底） |
| **v2.20.1** | 2026-08-14 | 数据可迁移独立审查修复（3 高危+4 中危）：恢复重试幂等、home 配置恢复、SQLite 快照 busy_timeout + checkpoint 回退、失败可见、已有 pending 拒绝、dirSize 缓存、Rollback 回滚 |
| **v2.20.0** | 2026-08-14 | 个人使用收口（不商用）：设置「数据」一键备份/恢复（Hephaestus.db + whisper_data + 配置 + sessions → zip + manifest；SQLite VACUUM INTO 一致性快照；两阶段恢复重启生效、恢复前自动备份当前数据）；微信通道 beta 标注、移动端冻结；发布形态简化（去安装器/签名/自动更新） |
| **v2.19.0** | 2026-08-14 | 阶段 3 第二刀（D3-4 补测评缺口）：测评报告专项分析（每模型对比/长上下文/缓存复用/显存参数/并发）；压力专项任务预设；快速流式探针（SSE 断流/卡顿观察） |
| **v2.18.0** | 2026-08-14 | 长期规划阶段 3 首轮：跨库统一语义检索补齐「资料」+ 索引状态；模型中心「本地 vs 云端」分流统计与节省对比；Herdsman 受控测评产品化（一键发起/逐 case 明细/Markdown 报告导出） |
| **v2.17.0** | 2026-08-14 | 长期规划阶段 2·安全与架构收敛：LAN 暴露全局告警横幅 + 设置「安全」面板；WebView2 远程调试默认关闭（GAEA_WEBVIEW_DEBUG 开关）；HTTP 桥接一次性 token 鉴权；敏感域本地化开关（成本/报价 AI 默认本地 Herdsman）；429 个导出方法按板块拆 10 个绑定门面（脚本生成 + 完备性测试兜底） |
| **v2.16.1** | 2026-08-14 | E1-4 模型中心资源协同：生命周期操作串行化（对齐 herdsman local_concurrency=1）+ 模型库磁盘 KPI（已装占用/卷余量）+ fmtSize TB 档 |
| **v2.16.0** | 2026-08-14 | 长期规划首轮：Herdsman 底座加固（环境探测/健康检查/TTS 默认动态解析/LAN 暴露告警/模型用途提示/思考模式 token 守护）+ 工程门禁（前端 CI 修复、发布冒烟脚本、周版本节奏） |
| **v2.15.7** | 2026-08-13 | 通用办公 P0·开工前计划卡片结构化：计划改出严格 JSON，后端解析为「任务理解/步骤（资料·工具·产出物）/待确认」，Ask 卡片渲染专属计划卡片，解析失败自动回退纯文本；测试 241→243 |
| **v2.15.6** | 2026-08-13 | Herdsman 深挖 P5·数字生命联动：记忆中枢新增「数字生命」库（角色/关系/记忆摘要/时间线/世界事件，只读）；最近 Herdsman 操作列表；路线图六阶段收官；测试 239→241 |
| **v2.15.5** | 2026-08-13 | Herdsman 深挖 P4·检索升级 + 调用统计：语义检索动态切 qwen3-embedding-4b / qwen3-reranker-4b（未装回退 bge）；模型中心新增 Herdsman 本地调用统计面板（model_stats 聚合）；两个模型已下载启动 |
| **v2.15.4** | 2026-08-13 | Herdsman 深挖 P3·本地翻译：优先 Hunyuan-MT / Hy-MT 翻译模型（未安装回退办公模型）；新增 translate_text 办公专业工具；能力面板入口；Go +6 用例 |
| **v2.15.3** | 2026-08-13 | Herdsman 深挖 P2·模型生命周期：模型库卡片直接启动/停止/下载/卸载 Herdsman 模型；读 launch_records 生成「启动预设」徽标；测试 238→239 |
| **v2.15.2** | 2026-08-13 | Herdsman 深挖 P1·模型库：模型中心新增「模型库」分类，接入 Herdsman 完整 90 模型目录（能力/安装/运行/量化/变体/大小），可搜索过滤；后端 HerdsmanModelCatalog（CLI RPC）+ 测试 234→238 |
| **v2.15.1** | 2026-08-13 | 通用办公·产物与资料体验收口：会话产物面板图片缩略图 + 一键复制全部路径；Ctrl+K 新增资料/产物/变更跳转；欢迎页任务模板内置兜底（离线/空库不再空白）；前端测试 229→234 |
| **v2.15.0** | 2026-08-13 | 模型中心 P0/P1/P2 + UI 重设计（引擎状态联动/连接诊断/功能绑定回退态、搜索收藏置顶/资源占用可视化/选择器统一、批量启停/统计抽屉；浅色深色双主题） |
| **v2.14.12** | 2026-08-13 | 绘梦 UI 重构落地（可折叠分区/底部生成栏/任务中心三 Tab/玻璃 HUD）+ 选项改下拉并修复 WebView2 弹层；herdsman 生图修复（size 契约/URL 转 data/图生图接入） |
| **v2.14.11** | 2026-08-13 | 小说板块后端（章节流式/状态/书架/世界观/统计/导出）+ 绘梦生成链路闭环（队列/取消、历史元数据持久化、ComfyUI 进度、模板/绘照、LoRA 重试）；绘梦 UI 重构已立项 |
| **v2.14.10** | 2026-08-13 | 修复办公模型改绑不生效：运行时改绑「办公」后同步重注入 bridge 并重建引擎，不再沿用旧模型 |
| **v2.14.9** | 2026-08-13 | 聊天板块后端：用户/助手消息原子落库，AppendMessage 用 RETURNING 收敛 seq 分配 |
| **v2.14.8** | 2026-08-13 | 聊天板块后端：会话列表预览 N+1 收敛为单条子查询，新增 GetTopic 供创建/导入/导出按 ID 读取 |
| **v2.14.7** | 2026-08-13 | 聊天板块交互收尾：清空对话二次确认、切换话题自动聚焦输入框、标题生成纯函数化；前端测试 194→196 |
| **v2.14.6** | 2026-08-13 | 聊天板块会话导出为 Markdown（落盘用户数据目录，前端一键导出并复制路径） |
| **v2.14.5** | 2026-08-13 | 聊天板块普通对话真实流式输出（后端逐块下发 delta/reasoning/done/error），替换前端模拟打字流 |
| **v2.14.4** | 2026-08-13 | 聊天板块收口：联网搜索不再污染用户历史、上翻「回到底部」、侧栏预览随发送/清空同步 |
| **v2.14.3** | 2026-08-13 | 聊天板块补强：中文输入法 Enter 防误发、快速切话题竞态修复、模式栏收敛；前端测试 190→194 |
| **v2.14.2** | 2026-08-13 | 聊天板块优化：会话最近活跃排序 + 相对时间、侧栏会话搜索、生成中智能滚动（上翻不吸底）、重开聊天正确载入历史消息；前端测试 182→190 |
| **v2.14.1** | 2026-08-13 | 办公板块缺陷收口（会话失败提示/归档删除确认/跨项目最近会话/变更面板汇总排序/归档搜索）+ 前端测试 138→179 + 首轮结构收敛 |
| **v2.14.0** | 2026-08-13 | 办公板块会话化升级：项目分组 + 会话置顶/归档/恢复 + 任务目标（需求→验收）+ 文件变更面板 + 专注模式；修复记忆图谱三元组被成本节点挤掉、会话删除注册表清理不完整、App 层 toast 静默失效 |
| **v2.13.22** | 2026-08-12 | 修复整轮结束后大过程卡折叠：合并大卡独立 key 全新挂载（默认展开），小过程卡实例始终折叠不误展开 |
| **v2.13.21** | 2026-08-12 | 办公板块安全审计：子代理不再继承持久化写入工具（封堵 headless 通道绕过审批），forget/install_skill 纳入硬性确认 |
| **v2.13.20** | 2026-08-12 | 记忆/知识库写入强制确认（remember/knowledge_add/promote_session_facts 与 cost_save 同规则）；记忆索引注入限 3000 runes，控制上下文占用 |
| **v2.13.19** | 2026-08-12 | 成本库写入强制逐条确认：cost_save 任何权限级别（含 yolo）都弹审批卡，显示条目/单价/规格/来源，批准仅本次生效，杜绝 AI 直接入库 |
| **v2.13.18** | 2026-08-12 | 通用办公左侧面板重设计（参考 Codex 会话栏）：紧凑头部、搜索前置、会话行标题+时间+预览、统一区块小字 |
| **v2.13.17** | 2026-08-12 | 修复运行中已完成的小过程卡不折叠：分段小卡（不含文本）默认折叠、段完成收起；含文本的大过程卡保持展开 |
| **v2.13.16** | 2026-08-12 | 办公板块铺满窗口；删除顶部「办公」二级标签；底栏只显示本地模型、云端引擎不再计入超载报警 |
| **v2.13.15** | 2026-08-12 | 删除方案编写模块与办公二级导航，办公板块收敛为单一「通用办公」入口；左脑记忆改接办公事实，三脑检索不受影响 |
| **v2.13.14** | 2026-08-12 | 通用办公工具/技能面板显示文档技能：docx/xlsx/pdf/pptx 安装到用户级全局技能目录（任意工作区可见），工具面板新增「文档技能」分组 |
| **v2.13.13** | 2026-08-12 | 聊天面板左侧栏可折叠（窄栏 + 状态持久化）；折叠时左下角悬浮绑定模型卡一并隐藏 |
| **v2.13.12** | 2026-08-12 | 通用办公布局优化：右侧边栏删除成本库/搜索标签；绑定模型卡移入左侧栏底部，随面板折叠、不再遮挡导航按钮 |
| **v2.13.11** | 2026-08-12 | 修复记忆中枢·用户画像打不开：后端空结果返回 null 导致前端崩溃，统一改为空数组 + 前端 null 兜底 |
| **v2.13.10** | 2026-08-12 | 修复办公文档处理反复弹 cmd 黑窗：markitdown 转换 / Excel 重算 / 报告导出 / 绘图 / 桌面自动化统一隐藏子进程窗口 |
| **v2.13.9** | 2026-08-12 | 每一轮的大过程卡完成后默认展开（不再只保留最新回合）；大过程卡内部思考卡默认折叠，工具卡/工具组保持折叠 |
| **v2.13.8** | 2026-08-12 | 仅最新回合的大过程卡默认展开，旧回合自动折叠，不再全部摊开 |
| **v2.13.7** | 2026-08-12 | 办公上下文窗口修正（1M→256k，超限自动压缩）+ 大上下文处理提示 + 最终回答兜底 |
| **v2.13.5** | 2026-08-12 | 运行中强制跟随底部：长任务实时显示最新输出，修复“卡住没输出”假象 |
| **v2.13.4** | 2026-08-12 | 外层过程卡（ProcessCard）完成后默认展开，过程文本与过程卡直接可见；内部工具卡默认折叠 |
| **v2.13.3** | 2026-08-12 | 过程卡完成后默认展开：大输出卡片（≥10 行或 ≥2000 字符）完成自动展开，用户折叠后不再干预 |
| **v2.13.2** | 2026-08-12 | 办公过程文件落盘规范：中间文件统一 .gaea/work/、交付物 .gaea/exports/，不再与源文件混放 |
| **v2.13.1** | 2026-08-12 | 补丁：修复 @引用 PDF 时二进制注入上下文导致办公输出不可见 |
| **v2.13.0** | 2026-08-12 | 通用办公打磨：方案分节按字数目标续写（修复 100 字截断）、docx 读取乱码修复、运行态看门狗与停止按钮复位、待办自动收尾、模型中心优化、Herdsman/Ollama 本地模型双模式实测 |
| **v2.12.0** | 2026-08-11 | 稳定工程：WebView2 rAF 冻结看门狗、成本库多级分类重设计、剧照文件化、记忆中枢/办公四库增强 |
| **v2.11.0** | 2026-08-10 | 通用办公大优化：工作区全文搜索、常用资料固定装配、任务模板库、记忆生命周期与自进化 |
| **v2.10.0** | 2026-08-09 | 通用办公三阶段闭环正式发布：前期解析（OCR 四级管线 + 事实底座）、中期编辑（Word 框选即改/修订、Excel 单元格级编辑 + 插行插列 + 公式重算）、后期输出（统一交付出口 + 成本测算模板）、Codex 式文件预览布局（v2.7.0–v2.9.4 累积） |
| **v2.7.9** | 2026-08-09 | 通用办公：粘贴图片附件一键「提取文字」（OvisOCR2 常驻服务），识图/提字双入口 |
| **v2.7.8** | 2026-08-09 | 扫描件 OCR 常驻服务（llama-server 共享端口、按需拉起，多页 2.4s）+ format_convert 直连 Ovis，修复扫描件误判文本 |
| **v2.7.7** | 2026-08-09 | 扫描件 OCR 上 OvisOCR2（0.8B 文档解析 GGUF+Vulkan，Markdown/表格/公式），auto 链路 Ovis→Rapid→WinRT→VLM |
| **v2.7.6** | 2026-08-09 | 性能专项：docx→MD 提速约 12x（修复 O(n²) 标签扫描）、PDF 约 4x、xlsx 微调，新增转换基准 |
| **v2.7.5** | 2026-08-09 | 方案校验增强：整改建议 + 原文定位（章节/摘录/一键跳转），AI 覆盖建议透传 |
| **v2.7.4** | 2026-08-09 | 事实底座一键沉淀长期记忆（去重/分类映射），后续对话自动加载 |
| **v2.7.3** | 2026-08-09 | 事实底座一稿多用：fact_add/list/clear 工具 + 侧栏事实底座面板，交付物基于同一底座生成 |
| **v2.6.9** | 2026-08-09 | 移除 VoxCPM2（实测不达标：耗时长、音色男女混乱），本地 TTS 保留 CosyVoice2 |
| **v2.6.8** | 2026-08-09 | 模型中心一键启动本地 TTS 服务：gaea 启动自动保活 CosyVoice2/VoxCPM2，模型卡片「启动」/测试连接/合成前兜底均自动拉起 |
| **v2.6.7** | 2026-08-09 | VoxCPM2 Vulkan GGUF 加速（克隆 RTF 0.65–0.84 / 语音设计 0.60）+ 本地 TTS 4 音色替换（火山引擎样本）；含 v2.6.5/2.6.6（CosyVoice2 GGUF+Vulkan 提速 8–10×、VoxCPM2 本地引擎接入） |
| **v2.4.5** | 2026-08-08 | 通用办公欢迎页重设计：6 大核心能力卡 + 内置技能 chips，替换土壤修复旧内容 |
| **v2.4.4** | 2026-08-07 | 文件预览重设计：对话内点击文件打开大尺寸预览（docx/xlsx/pdf 内联转 Markdown），右侧面板删消息/报告并默认折叠 |
| **v2.4.3** | 2026-08-07 | 精简内置工具集：38→17 个核心工具，文档专项工具移交 ModelScope 技能，清理前端死列表 |
| **v2.4.1** | 2026-08-07 | 设置面板重设计：按功能板块左侧导航 + 全局搜索，清除死代码与重复面板 |
| **v2.4.0** | 2026-08-07 | 网页调试桥接（GAEA_HTTP_PORT：RPC+SSE 驱动同一内核）+ 办公引擎热加载（技能/工具/插件免重启生效） |
| **v2.3.0** | 2026-08-07 | 界面焕新与办公整合：角色库/首页/智能办公/方案编写重设计，随机生成完善（含人格） |
| **v2.2.0** | 2026-08-07 | 统一角色库：全局角色资产（小说×聊天同一模型）、小说抽卡单向引用、聊天内选角色、角色记忆隔离、状态/记忆/追踪归集角色库、取消轻语称谓 |
| **v2.1.0** | 2026-08-07 | 二代完善：模型中心持久化/启停语义、Cmd+K 引擎路由、OAuth 回归、前端 E 系列守卫、小说剧照与模型链路审计 |
| **v2.0.1** | 2026-08-07 | 三脑底座：模型路由降级链、三脑记忆、主脑可选编排、基线加固 |
| **v1.0.0** | 2026-08-01 | 品牌重塑：wubigrok 正式更名 gaea，全量替换品牌名与 logo，版本重新起算 |
