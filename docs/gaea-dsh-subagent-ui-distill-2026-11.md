# DSH 0.2.2-alpha.2 子代理 UI/UX 蒸馏 —— 调研与刀池（2026-11-02）

> **补课缘起**：后端/模型面已两轮蒸馏（2026-10 判定级+刀池、2026-11 机制深挖，P0/P1 已落地），**UI/UX
> 面此前从未系统蒸馏**——gaea 前端的子代理面（SubagentThread/useTaskCards/ChatTabs 圆点/queued 徽标/Ban
> 图标）全部是随刀顺手认领，不是从上游 UI 蒸馏出来的。本档补上。
> **源**：`clones/deepseek-harness` @ dsh-v0.2.2-alpha.2。上游 UI 面=packages/client 三件：
> `ui-subagent`（SubagentHeaderLineage 823 行谱系树+SubagentReadOnlyComposer 只读态+locales 词汇表 1055 行）、
> `ui-settings-subagent`（SubagentCard/limits/model-selection 设置卡）、`ui-chat` conversation-nodes
> （notice/relay 节种+turn-process 聚合+process-activity 活动分组+turn-trigger 源徽标）。
> **gaea 侧已核**：SubagentThread.tsx（四态徽标/3s 轮询不可见门控）、useTaskCards（args 文本唯一命中匹配
> 宁缺勿错配）、subagentRender.ts（toolCallId 配对纯映射层+孤儿工具行诚实降级）、types/subagent.ts
> （run meta 带 createdAt/updatedAt）、event.go TriggerSubagent 词汇。

## 一、上游 UI 机制清单与判定

**U1 会话头部谱系树（SubagentHeaderLineage）**：触发器 popover 内 role=tree 树（treeitem/group/
aria-level/current/expanded 全套），StateDot 三态（ongoing/done/idle），分支折叠=显式 expanded Set（默认
收起、只渲染展开层递归），hover 开（进出定时器）+**点击钉住**（点击开的菜单无视 hover-out，只认显式
关闭），portal 贴 viewport 边。词汇：「N 个子智能体/正在运行/已完成/当前未运行/收起 X 的下级子智能体/
在侧边栏打开」。
判定：gaea=ChatTabs tab+SubagentThread 面板（单层清单+ tab 圆点），无树（gaea 无嵌套面，单层够用）。
**谱系树=嵌套若开的 UI 预案**，留档不立项。**点击钉住 vs hover-out** 的弹层交互纪律与 gaea 办公弹层
键盘可达线（v4.516）同族，已做。

**U2 指标行**：token 计数（<100 保留一位小数、≥100 取整、K/M 档）+**总活跃耗时**（秒→分→时→天→月→年
8 档时长格式化，`duration.exactTitle='总活跃耗时：{duration}'`）。判定：gaea SubagentThread 头部=任务
标题+状态徽标+模型+消息数+刷新，**无活跃耗时**——run meta 有 createdAt/updatedAt，成本一行。**UA-1
小刀候选**。

**U3 只读态诚实（SubagentReadOnlyComposer）**：三态词汇——「一次性子智能体记录（不支持后续消息，可在
这里查看完整执行记录）」「此子智能体暂时只读（父会话当前不在线，重新打开父会话后即可继续发送消息）」
「读取子会话后才能确定是否可继续」。第二轮补遗：组件形态=**替换 composer 并解释为什么不可用**
（`role="status"`+按 reason='one-shot'|'parent-unavailable'|'unknown' 三路选文案）。判定：gaea task 全部可
continue_from，无「不可续」概念；外部 provider 生态在册排除。销号，词汇留档；**「不可用态要解释原因」
纪律对照 gaea 禁用态控件通用纪律，已达标**。

**U4 catalog 态**：loading/migration-required（「需要迁移」）/error+retry 三态，external 行 aria-disabled
不可展开。判定：gaea runs store 读失败已有诚实空态（subagent_list 同款）；schema 迁移态 gaea meta 侧车
无版本迁移需求。销号。

**U5 结算通知呈现（turn-trigger）**：消息 source.kind→触发徽标词汇表：agent-message→agent 图标、
subagent-settled→subagent 图标、goal/team/webhook（github 细分）/schedule/tool-jobs/plugin 各有图标
标题；**未名源默认态=保持可见但不虚构归属**（"Custom sources remain visible without attributing
unrecorded identity or success"）。notice form=折叠行不展开。max-tokens notice 锚定在收尾 assistant
与 turn-tail 之间（按 turn/end seq 定锚不漂移）。判定：gaea 结算走 task 工具结果行（ToolCard），
无 user 消息形态结算故无触发徽标面；**「谁触发本回合」徽标**对照 gaea event.go TriggerSubagent 词汇
（subagent/dream 等）——前端是否已渲染 trigger 徽标=待核；未名源不虚构归属纪律与 AnsweredByLine
「未知值原样兜底展示」同族，已做。

**U6 过程聚合（turn-process + process-activity）**：委派工具调用**不计入工具调用数**（`subagent ? 0 : 1`
独立 subagentCount），回合披露行 data-turn-process-subagents 单列；活动分组词汇 read/search/write/
edit/commands/code/webSearch/webFetch/**subagents**/plan/questions/tools——subagent/subagent_* 独立成组
不混入 tools。live 活动行=工具实时「正在做什么」。判定：gaea ChatRow 轨迹 items[]（task 卡已独立成卡
+useTaskCards 配对）+分工面板同构；「委派不计工具数」对应 gaea task 卡不入普通工具折叠——语义已对齐。
活动分组词汇表可对照 gaea 轨迹折叠摘要，留档。

**U7 live 活动词提取（LIVE_TOOL_DETAIL_KEYS）**：实时工具行的「正在做什么」文本按**键优先级清单**提取：
title→description→objective→task→task_name→name→question→questions→prompt→message→command→cmd→
queries→query→pattern→url→uri→file_path→path→target→action→status；空格归一+**160 字符 grapheme
正确截断**（Intl.Segmenter 不劈字形）。判定：gaea task 卡摘要取 args 任务描述原文；**UA-2 取道候选**：
subagentRender 的 tool 行摘要若做截断/选键，照此优先级清单+grapheme 截断（中文 grapheme 安全）。

**U8 设置卡三件（ui-settings-subagent）**：SubagentCard（启用/上限）+SubagentLimitsFields+ModelSelection
Fields。判定：gaea 子代理配置在 config 文件 `[agent] subagent_max_parallel`，无设置 UI。前端池需求
gated，不立项。

## 二、gaea 现状盘点（本轮已核，蒸馏对照基准）

- **SubagentThread**：头部=返回分工+任务标题+状态徽标+模型+消息数+手动刷新；四态 queued/running/
  completed/failed 各有色；实时=running 3s 轮询（页面不可见门控）+事件驱动刷新。
- **useTaskCards**：ToolCard 透传 args 任务文本与 run.task 唯一命中匹配（≥2 候选宁缺勿错配）；tab 预填
  +打开后 5s 轮询自校正。
- **subagentRender**：transcript→渲染项纯函数映射（toolCallId 配对；孤儿工具行降级独立卡诚实展示）；
  assistant live 判定=最后一条。
- **runs meta**：createdAt/updatedAt 在（UA-1 数据就绪）；ref/状态/标题/模型在。
- **AnsweredByLine**：消息级回答者归因（engine/model/source/cost/tps），未知值原样兜底——与上游
  turn-trigger 未名源纪律同族，gaea 自有实现已达标。
- **TaskInboxHost**：后台任务收件箱（jobs 面，非本档范围）。

## 三、刀池（UA 系）

| # | 刀 | 性质 | 说明 |
|---|---|---|---|
| UA-1 | SubagentThread 头部活跃耗时 | 小刀 | meta createdAt/updatedAt 就绪；8 档时长格式化照上游词汇（秒→分→时→天…），「总活跃耗时」tooltip |
| UA-2 | 工具行摘要活动词优先级 | 取道 | U7 清单+grapheme 截断；先核 subagentRender 现行 args 展示宽度策略再动 |
| — | 谱系树 | 留档 | U1=嵌套若开的 UI 预案（树+StateDot+分支折叠+侧边栏打开词汇） |
| — | trigger 徽标 | 待核 | U5：gaea 前端是否渲染 event trigger 词汇（subagent/dream），缺则与 U5 词汇表对照 |
| — | 设置卡 | 需求 gated | U8，[subagent] 段有 UI 需求时立项 |

销号：U3 只读三态（gaea 全可续）、U4 catalog 迁移态、U8 设置卡（gated）、U6 过程聚合（语义已对齐）。
留档：U2 token 计数格式（随 UA-1 顺手）、U7 活动词清单（随 UA-2）、U5 未名源纪律（已达标，词汇留档）。

## 四、方法学注记

- 上游 UI 蒸馏的入口=**locales 词汇表**（zh 字典是 key-set 权威，一行一词全见 UX 面），比读组件快——
  先读字典再按 key 反查组件。
- gaea 前端子代理面的既有纪律盘点：诚实降级（孤儿工具行/宁缺勿错配/未知值兜底）、页面不可见门控空转、
  tab 预填自校正——与上游同族但全部是刀驱动独立长出来的，本档后正式入蒸馏对照系。
