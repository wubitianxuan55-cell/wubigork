# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **最新发布：v4.239.0（2026-09-12）「数学公式渲染对齐：伴侣线接入 KaTeX」**——伴侣/聊天线（ChatMarkdown/MarkdownContent）一直没有数学渲染（$..$、$$..$$、\(..\) 裸显），办公板块却早有 KaTeX——同仓双标。**落地**：① 新 gaea/lib/mathText.ts 跨板块共享（normalizeMath/hasMathContent/ensureKatexCss 自 Markdown.tsx 提取，行为零变化）；② ChatMarkdown plain 管线加 remark-math+rehype-katex，text 先过 normalizeMath，有数学内容才注入 KaTeX CSS；③ MarkdownContent companion/流式同款（GenUI 覆盖件路径不受影响）。零新依赖零新 chunk（katex 经既有共享 chunk 复用）。vitest +2（两线 $..$→.katex 真实渲染）。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 625/版本三处 4.239.0。**欠账**=真机抽查伴侣线数学回复（等自然使用窗口）。

- **最新发布：v4.238.0（2026-09-12）「观察池收官：回答反馈（点赞/点踩）落记忆事件」**——**设计取舍使「无消费方」顾虑消解**：反馈不做成孤立数据，而是记忆事件（op=feedback）落 memory_events——事件节点随 v4.210 语义图谱投影自然可查，消费者=事件图谱/事件日志现成面。**落地**：① memory/eventlog.go OpFeedback 扩员（closed set）；② sqliteBackend.AppendFeedbackEvent（摘要 eventExcerptLimit 截断+At/Actor 缺省补齐）；fileBackend 诚实报错（Pin 先例——绝不静默假成功）；spaceView 最小透传；Store 值方法分派（与 Pin 同形，无接口级联）；③ **App 绑定 624→625**（gaea_memory_feedback.go：rating 限 up/down 非法拒绝；事件字段 Name=messageID/Title 点赞回答|点踩回答/Space/SourceSession 当前会话/SourceMessage/Actor=panel；bindings_memory 透传一行）；④ 前端 AssistantMessage 操作行 👍/👎（icons +ThumbsUp/Filled/ThumbsDown/Filled，antd Like/Dislike wrap）——一次有效（事件追加式不可撤回，点击后双钮禁用）、失败回退可重试+error toast；Transcript onFeedback 开关全链透传（App→Transcript→TurnBlock→ctx→AssistantMessage 五环）。**测试**=Go +3（落库全字段含摘要 ≤280B 截断/投影只出事件节点不建实体——cite 悬空拒写同边界/文件后端诚实报错）+vitest +3（落库参数+双钮禁用/失败回退+toast/熄灯口径）。**坑**=①批量接线脚本末位断言崩掉整体不写盘，重做只补 props 声明漏了 ctx/调用点→真机按钮不渲染——**批量接线后必须从调用点正向全链 grep 复核（App→Transcript→TurnBlock→ctx→AssistantMessage 五环）**；②Wails 要求绑定参数逐个显式传（facade 可选省略=args:[] received 0 expected 1），调用点显式传空串（Go 侧空串语义核实安全）。**真机复验**=点👍→已反馈态+语义图 ev:10「feedback · 点赞回答」事件节点含摘要——端到端全链打通。**门禁**=Go app+memory 绿/vitest 87 例相关面绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624→625/版本三处 4.238.0。**观察池全清**；反馈深度消费（按反馈加权重排记忆/注入偏好）属记忆 OS 后续方向按需另刀。

- **最新发布：v4.237.0（2026-09-12）「变参绑定根治：绑定层去变参 + Wails 变参真相定论」**——v4.236 单串化方向对但没修到根：轨迹页走查暴露新错误（unmarshal string→[]string），**Wails v2.13 源码（ParseArgs 严格计数+reflect.Call 按 In(0)=string）+ 真机经验矩阵（七形态×双方法全败：传串 unmarshal 失败/传数组 reflect panic 被 recover 回调永不送达=promise 永久 pending）双重定论=变参绑定在 Wails v2.13 不可用**。修复=bindings_office.go 六处+bindings_cost.go 一处绑定门面 ...string→单 string 透传（核心层保留变参：resolveGaeaSessionPath 跳空回退当前会话/taskListInSpace("")=不过滤语义核实安全）；前端 facade 改必填串+调用点显式传 ""（Wails 要求逐参显式传，省略=count 错真机实锤）。**真机复验**=wire 错误 0+轨迹页红卡消失满载数据+任务管理实时行在位（对比 v4.235 红卡/v4.236 空面板）。**坑固化**=①Wails v2.13 绑定面禁用 ... 参数（核心层可保留）②修 wire 契约后必须重建 exe 再复验③被 .catch 吞掉的接口错误修完报错≠修完功能必须验证数据真到达。**门禁**=Go app 绿/vitest 87 例绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.237.0。**欠账**=观察池剩点赞点踩（等真实需求）。

- **最新发布：v4.236.0（2026-09-12）「修复变参绑定 wire 契约：reflect 参数错误清零」**——清 v4.235 走查观察池项。**根因（桥接约定与 Wails 真实形态不符）**：Go 五个变参绑定（GaeaAgentNetwork/GaeaTrajectory/GaeaContextView/GaeaContextNodeDetail/GaeaTaskList，均 ...string）要求每个变参元素作为独立顶层参数（args:["path"]），facade 却约定 JS 数组整体作单参（args:[["path"]])→reflect 报错；**推断=v4.174 退役的 wailsjsCompat shim 当年会展开数组，退役后数组调用点静默失去展开**，报错被 .catch 吞掉无人深究，直到壳内走查抓日志。**修复（facade 单可选串化——全部调用方只传 0..1 个路径，Go 变参天然支持）**：bridge/core.ts 五签名 string[]→?:string；七调用点同步简化（AgentNetworkCard/agentNetworkStore/TrajectoryView/ContextView/inspector 数组包裹拆除；TaskCenter/useRunningBadge TaskList([])→TaskList()）；mock/core.ts 四实现对齐；agentNetworkStore 补语义（poller 空串=内核会话哨兵，下发前归一 undefined）；UnifiedSearch/CostImportApply 调用方本就字符串/展开传参核实不动。**回归锁**=新 bridge.variadic.test.ts 4 用例（注入门面 spy 钉死顶层 string/零实参形态）；agentNetworkStore/ContextView 三处旧数组断言更新。**真机复验**=重建壳进办公工作台 reflect 错误 0 条（修复前必现）。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.236.0。**欠账**=观察池剩点赞点踩（等真实需求）。

- **最新发布：v4.235.0（2026-09-12）「壳内真机走查：抓到并修复 DAG 模板区 nil slice 崩溃」**——「测试绿≠真机通」再实锤：五刀（高亮×3/regenerate/折叠）壳内 DOM 断言走查（CDP 9333 起壳 + cdp-walk evaluate/--click/--shot），进办公工作台第一步撞上页面级崩溃 `Cannot read properties of null (reading 'length')`——vitest 330 文件全绿没拦住。**根因**=用户机器模板目录不存在 → `DagTemplateStore.List()` IsNotExist 分支 `return nil, nil` → Go nil slice 经 Wails JSON 序列化成 **null** → 前端 `setTpls(null)`（类型标注非空但边界没挡）→ `tpls.length` 炸；测试/mock 给的都是 []——v4.210「nil↔[] 漂移」同族、跨语言面。**修复（双侧归一）**=① Go dag.Store.List/TemplateStore.List 的 IsNotExist 分支返回空集非 nil；② DagPanel 三处绑定边界 `?? []`；③ Go 用例强化空目录 List 断言非 nil（原 len 断言对 nil 也过）。**真机复验**=重建壳重进正常渲染错误 0、办公流水线区在位。**走查其余结论**=data-hl 全链路双主题真机工作（`gaea-display-mode` dark→data-hl dark+body 切深；首查见 light 是翻错键——`gaea-dark` 是被压制的 legacy 键非 bug）；regenerate 按钮真实会话在位。**方法论沉淀**=①抓错误边界底下的堆栈：Runtime.enable 收 `Runtime.exceptionThrown`（边界只给人话）；②minified 崩点按 dist chunk 行:列切上下文反查源码；③`--key` 不带修饰符发不了 ctrl+4（组合键须 Input.dispatchKeyEvent 带 modifiers）。**新观察池**=GaeaTaskList/GaeaAgentNetwork 的 `reflect: cannot use []string as type string`（预置参数形态错，接口报错不崩页面，按需另刀）。**门禁**=Go dag/app 绿/vitest/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.235.0。

- **最新发布：v4.234.0（2026-09-12）「AI 输出渲染：超长代码块折叠 + i18n context value 修正」**——观察池第二项清账（对齐 Claude/ChatGPT 长代码处理）。**落地**：① 新 gaea/components/CodeCollapse.tsx 双板块共享——>30 行块级代码默认收起（max-height 380px+底部渐隐+展开/收起胶囊钮），≤30 行零包装直通；**跨 chunk 共享组件不能假定对方加载了 gaea/styles.css 令牌**——渐隐色调用方显式传（gaea 默认 var(--bg)/聊天线传 #0b0e14 同其恒暗面板），按钮用全局 M3 令牌内联样式，文案调用方注入 labels（gaea 走 useT 三语/聊天线硬编码中文一致）。② 新 useTOptional()：无 LocaleProvider 回退 zh 直译不抛错——Markdown 是被裸渲染的共享组件，useT 会让全部裸渲染用例连坐抛错（首跑 27 例挂）。③ 顺带修 LocaleProvider context value 不稳定（真回归，stash 二分+三步实验定位）：value 原先每渲染造新对象，Markdown 经 useTOptional 成为消费者后 en chunk 就绪的 forceRender 把它拖进重渲染级联、ReactMarkdown 整树重解析撕掉 MemCitationChip 弹层——修复=value useMemo 化（deps: locale/pref/setPref/tt），消费者只在语言真变时重渲染。**教训=Provider 的 context value 不 memo 化是埋雷，任何新消费者上线都可能引爆**。vitest +2+MemCitationChip 3 例转绿。**坑四证（固化铁律）**：python heredoc 追加源码第三次转义丢失——追加源码只走 Edit 工具无例外；后台 CI 管道 tail 截丢归因必须整份落盘。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.234.0。**观察池剩**=点赞点踩/壳内真机走查。

- **最新发布：v4.233.0（2026-09-12）「AI 输出渲染收尾：MarkdownContent 缺省高亮 + 明暗调色板目检」**——高亮三刀收尾。**① 目检补证**：v4.230/231 此前只有 vitest DOM 断言没有视觉验证——配方=Node 同款 highlight.js 语法包生成 go/sql/python/powershell 样例 hljs HTML → check.html 双 link 真实 gaea/hljs-theme.css（双面板=gaea 随主题 / 聊天线恒暗 .hl-scope-dark）→ 无头 Edge --screenshot 明暗两版人工目检：暗色层次清楚/浅色深阶对比良好/**恒暗面板在浅色主题下令牌仍钉暗色组按设计工作**。**② MarkdownContent 缺省高亮**：NovelSettingPage.tsx:189 直用无覆盖=全仓最后一个无着色渲染面——模块级 defaultComponents（引用稳定不破坏 memo）：未传 components 时块级代码走 ChatCodeBlock（与聊天线同款）、行内交还 .md-content code 默认样式、pre 透传防双层；传了 components（GenUI 缝）完全尊重调用方零变化（消费面盘点：无覆盖调用方仅此一处）。vitest +3。**③ 顺带根治 Go 在册 flaky（两天两度打挂 CI）**：TestCreateChapter_SameChapterConcurrentRejected 家族失败从来不是断言而是 t.TempDir() 清理竞态——CancelCreateChapter 取消路径先删登记表，被取消协程仍有「已生成部分落盘」尾步，waitGensDone 只等表空放行即撞 Windows unlinkat（directory not empty）；根治=writingState 增 chapterGenWG sync.WaitGroup（spawn 前 Add/协程首 defer Done，LIFO 故最后触发=协程真退出）+waitGensDone 两级等待（表空快速路径+WG 5s 超时）——**教训：登记表空≠协程退出，取消路径删登记与协程收尾之间有窗口**，-count=10 全绿（修复前定向复跑即可复现）。**坑三证**：python heredoc 追加含反引号 fence 的测试代码再次转义丢失产出语法错 JS——追加源码只走 Edit 工具无例外。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.233.0。**「AI 输出效果对齐同类产品」主线全清**（v4.230 办公高亮/v4.231 聊天线/v4.232 regenerate/v4.233 收尾）；观察池=点赞点踩（个人工具暂无消费方等真实需求）/超长代码块折叠/壳内真机走查等闲置窗口。

- **最新发布：v4.232.0（2026-09-12）「AI 输出渲染第三刀：重新生成 regenerate」**——渲染两刀（v4.230/v4.231）后补齐消息级交互最大差距：对最后一条回答一键重新生成。**复用既有原语零新绑定零 Go 改动**：GaeaRewind(turn,"conversation") 截断该轮（含用户消息）+ GaeaSend 原样重发。**落地**=① controller.regenerate(turn)：守卫（运行中/有排队未决拒绝；**只对最后一轮开放**——更早轮次语义交给回退/分叉）→ rewind 成功后 send(该轮用户文本)；rewind 失败不动现场；rewind 顺带改返回 Promise<boolean>（既有调用方忽略返回值零破坏）。② UI=AssistantMessage 操作行新增「重新生成」（RefreshCw，locale 三语 msg.regenerate），显形=当前会话最后一条 assistant 且非运行中非流式——**半截取消的回复同样可重发**（rewind 把半截正文与 warn notice 一并清掉后干净重来，cancel 后高频动作）；props=canRegenerate 布尔+稳定 onRegenerateTurn 回调（AssistantMessage 内部 useCallback 组装轮号）不击穿 TurnBlock/AssistantMessage memo 链；Transcript canRegenerateId useMemo 从 items 尾扫。vitest +7（store.t74 +4：最后一轮编排 Rewind(1,"conversation")+Send(原文)/rewind 失败不重发且可见/非最后一轮拒绝/运行中拒绝；Message +3：按钮按轮号回调/无资格不渲染/流式中不渲染）。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.232.0。**欠账**=任意轮重生成（其后轮次级联重跑=连续 LLM 调用+费用）按需另刀/SubmitDisplay 形态消息重发以显示文本为准（附件形态诚实降级）/壳内真机走查挂观察池。

- **最新发布：v4.231.0（2026-09-12）「AI 输出渲染对齐同类产品第二刀：聊天线（伴侣/流式）代码块高亮」**——接 v4.230 续刀：ChatPage 两模式代码块全无着色（plain 终态纯文本；companion/流式经 genuiAdapter 后非 genui 代码只落裸 code），且 .hljs-*/--hl-* 规则都在 gaea/styles.css 而**该文件不在 ChatPage 加载范围**（仅 Gaea/CostLibrary/MemoryHub/Schedule 四 lazy chunk 引用）。落地=①hljs 三块（暗调色板/浅覆写/.hljs-* 令牌规则）自 styles.css 迁出为 `gaea/hljs-theme.css` **色值单一真源跨板块共享**，重构双常量组 --hl-dark-*/--hl-light-* → :root 兜底暗 + [data-hl=light] 跟主应用明暗（迁出前核 tailwind 桥接 --color-hl-* 零页面消费方、全仓无第二处 .hljs 定义）；②HlCode 自 Markdown.tsx 抽出 `gaea/components/HlCode.tsx`（css 随组件 import 随用随载），vite 自然 hoist 成办公/聊天共享异步块；③新 `components/ChatCodeBlock.tsx` 聊天线两模式共用=复制头+暗面板+高亮，面板维持「行业标准暗色专用色不随主题」在册决定（hex 同行豁免标记照旧），根节点挂新 `.hl-scope-dark` **元素级钉死暗色组——令牌随面板不随应用主题**；④接线两处=ChatMarkdown 块级分支换 ChatCodeBlock（本地头部移入）+ genuiAdapter 非 genui 块级同走（data-genui-host 透传不落 md-content pre 默认底），companion/流式与 plain 终态同款；⑤白名单 22→27（+ruby/php/kotlin/perl/lua）。vitest +4（plain 暗面板+异步 hljs 令牌/companion 同面板/白名单直通）。**坑**=①python heredoc 往源文件追加代码时反引号十六进制转义在非 raw 串提前变真反引号、产出语法错 JS——**追加源码一律走 Edit 工具**；②eslint no-raw-hex 豁免标记必须与 hex 同行（行尾 // hex-exempt），提行注释不豁免。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.231.0。**欠账**=NovelSettingPage 直用 MarkdownContent 未着色（非聊天面按需另刀）/重新生成·点赞点踩按反馈另刀/壳内真机走查挂观察池。

- **最新发布：v4.230.0（2026-09-11）「AI 输出渲染对齐同类产品首刀：聊天代码块语法高亮」**——现状盘点后最大可见短板=聊天代码块零着色；styles.css 的 hljs 令牌主题（.hljs-* 十组规则+--hl-* 八色）**早已预铺却一直没有生产者**，lang.ts 头注「no highlight.js dependency」是当年刻意留的懒缝，本刀接上。落地=新依赖 highlight.js ^11.12.0 + 新 gaea/lib/codeHighlight.ts 懒加载缝：core+22 语言白名单（go/bash/sql/yaml/powershell/ts·js/py/c 系/rust/java/json/xml/css/md/diff/ini/makefile/nginx/dockerfile）全走动态 import 独立 async chunk（mermaid 566KB 同款先例；构建审计 registerLanguage 入口 chunk 0 命中、核心 21KB）；未注册语言回退纯文本不硬造；围栏语言别名归一复用 lang.ts ALIASES（编辑器/工具卡同源表；toml 无官方语法借 ini 着色、展示标签不变）；hljs 产出（自带转义仅 hljs-* span）插入前再过统一消毒层 DOMPurify 兜底；Markdown.tsx 代码块接 HlCode=首帧纯文本立现、高亮就位整体换 HTML（调用点=稳定分段 MemoMarkdown 段签名缓存，流式增长尾部走简易 HTML 不经此处——无逐 delta 重高亮）；明暗双调色板=暗色既有 One Dark 系 :root 兜底不动+浅色 :root[data-hl="light"] 同色相深阶（白底 ≥4.5:1），data-hl 由主 App 主题 effect **内联**挂载（刻意不 import gaea/lib——sanitize 链会拖进主入口 chunk），darkMode 为 store 派生实际明暗 system 档实时跟切。测试=vitest +11（codeHighlight 9+Markdown 2）。**坑**=hljs 类名运行时 hljs-+scope 拼接，产物 JS 无 "hljs-keyword" 字面量，bundle 归属审计不能 grep 类名，用 registerLanguage 特征串。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.230.0。**欠账**=白名单外语言（kotlin/php/ruby 等）按需扩 REGISTER；重新生成/点赞点踩属消息级交互面按反馈另刀；ChatPage 伴侣线自有渲染链（ChatMarkdown/MarkdownContent）不动；壳内真机走查（明暗两主题代码块）挂观察池。

- **最新发布：v4.229.0（2026-09-11）「任务表会话维度 + gongwen 页码/字色」**——并行双线。**线 A=v4.180 结构欠账收口**：SchemaV20 `tasks.session_id`（NOT NULL DEFAULT '' 旧行零成本回填，迁移编号先查迁移链=migrations 切片 V19 终点+1）+ Task.SessionID（json session_id,omitempty）+ SubmitSpaceSession（既有签名不变委托，INSERT/SELECT/scanTask 全链带列）；**诚实口径=全仓 6 个任务创建点（文件索引/价格抓取）全为 cron/设置类无会话上下文，子代理/DAG 是独立数据系统，session_id 如实全空停在能力层不造假接线**；前端 TaskCenter「本会话/全部」过滤面+三语+4 用例就位，**chip 暂不显形**（TasksWorkbench 不传 sessionPath，等首个会话入口接线再点亮——恒空过滤=半成品面）。零新绑定 624。**线 B=gongwen-9704 余项（v4.223 立项）**：docx_layout.py 增页脚解析（rels 归位 ref_types/\bPAGE\b 防 NUMPAGES 误认/一字线文本）+字色采集（run 级聚合，auto 不采），输出增 footers/colors 旧键零动；SKILL.md 排版表 6→8 项；test_docx_layout.py stdlib 夹具 4 用例过。规范知识零进内核（红线）。**坑**=有恒空风险的新 UI 面先熄灯、能力+测试先行。顺带 AGENTS.md 五迁（本条目同批，v4.208~203 四条 4770B 入 archive 回 WARN 线下）。**门禁**=ci.ps1 全绿/绑定 624 零变更/版本三处 4.229.0。

- **最新发布：v4.228.0（2026-09-11）「办公板块输出列左对齐」**——用户反馈「输出部分还是居中」：`.v3-reading` 基础 `margin:0 auto` 只被前次重设计加宽未去居中。落地全走 `.gaea-app-layout` 作用域覆写（零删基础规则）：输出列/composer-glow 去 auto margin（同一左缘）+ `.phase` 左对齐。**坑**=SchedulePage/MemoryHubPage/CostLibraryPage 均导入 gaea/styles.css，共享规则改动必须 `.gaea-app-layout` 限定（进度计划居中流不动，其输入框居中来自 Composer 内部 mx-auto 也零波及）。顺带清 monte.ts 存量 prefer-const（v4.217.0 入库，十版未跑全量 lint 未暴露）。**测试**=真实 CSS+DOM 静态挂载 Edge 无头截图前后对照+vitest src/gaea 1541 全绿；**门禁**=ci.ps1 全绿/绑定 624 零变更/版本三处 4.228.0。

- **最新发布：v4.227.0（2026-09-11）「规范知识出内核第四刀：造价判定参数数据化」**——拍板池 §5.5 执行，**形态 a=Apply 确认留痕流不动**（CheckComposeComponents/CheckContentBaseline 签名零变更，结论仍随证据链留痕），仅行业判定参数出码。internal/gaea/cost/checkparams.json go:embed 六参数内置默认（R1 绝对下限 0.01/相对容差 0.01、R3 超 1.05/低 0.5、含量最少样本 3/退化带容差 0.05）+ params.go 快照 atomic + `LoadCheckParams` **部分字段覆盖**（≤0 字段忽略，坏 JSON 显式报错，缺失静默）+ 引擎接线全走 `currentCheckParams()`（MinContentSamples 导出常量删除防双源漂移）+ app ensureCostCheckParams 每进程懒加载（composeChecks 入口，惯例路径 .gaea/skills/cost-compose/params.json）+ SKILL 文档（口径表×字段映射+判定纪律）。测试 Go +2（漂移守卫/覆盖全链：R1 容差抬升 3% 偏差翻转+minContentSamples=5 后 3 例池静默+非法值忽略+坏 JSON 报错+cleanup 恢复）。**门禁**=Go cost+app 全量 0 FAIL/绑定 624 零变更/版本三处 4.227.0。**拍板池 §5.4/5.5 全清——「规范知识出内核」审计线收官**：四刀四形态（gongwen-9704 撤植入转技能 / novel-deslop Go 词表数据资产整表替换 / schedule-dcma 前端阈值资产部分覆盖 / cost-compose Go 参数资产部分覆盖）；遗留观察=standard 包 Redhead/CostTable 文本级 checker 同属规范在码，迁移低优先挂观察池。

- **最新发布：v4.226.0（2026-09-11）「规范知识出内核第三刀：DCMA 阈值表数据化」**——拍板池 §5.4 执行，**形态 a=质量体检视图保留**（不退已发布产品面），阈值表出码。前端镜像 novelstyle 模式：dcma-thresholds.json 内置默认 + dcmaAudit 第 4 参 `Partial<DcmaThresholds>` 部分覆盖（判定线+展示串全走生效值 th）+ DcmaView 挂载 ReadFile 懒加载 `.gaea/skills/schedule-dcma/thresholds.json`（缺失/坏 JSON 静默默认）+ SKILL 文档（14 点×阈值对照+字段映射+诚实口径）。兼容导出 DCMA_* 从默认表派生零破坏；顺带清 naCheck 死函数。测试 vitest +2（漂移守卫/覆盖全链：阈值抬升判定翻转+展示串同步+未覆盖字段保默认）。**坑**=①ts 文件全局 perl 替换会改坏兼容导出行（`export const th.x` 语法错），批量替换后必须 tsc 立验；②断言「未覆盖字段」不能挑恰好被覆盖的字段；③DCMA 实测占比 value 随阈值变（offender 数不同），不能断言前后相等。**门禁**=tsc 0/eslint 0/vitest 329 文件 2847 全绿/绑定 624 零变更/版本三处 4.226.0。**拍板池余**：§5.5 造价 R1-R3/含量带技能化（涉 Apply 确认流语义，待拍板）。

- **最新发布：v4.225.0（2026-09-11）「规范知识出内核第二刀：小说 AI 味词表外置」**——红线（v4.224 拍板）审计后首落地（小说域）。**引擎=机制留码，词表=知识出码**：internal/novelstyle/words.json go:embed 内置默认（genui 先例，开箱即用）+ LoadWordsFile 整表覆盖 API（不合并可预测；惯例路径 .gaea/skills/novel-deslop/words.json，改词表不发版）+ 引擎三处接线（rewrite 替换表/segment 分词词表 vocab **原子快照可重建** tokenize 无锁读旧快照/score 规则 9 黑名单）+ app ensureNovelStyleWords 每进程懒加载（两入口；缺失/解析失败静默保默认=增强面不挡主功能）+ SKILL.md 机制说明与改表指引。测试 Go +2（漂移守卫 27/19/覆盖全链/缺失静默/cleanup 恢复内置）。**坑**=①词表快照持有者须包级变量初始化先于一切 init（segment 的 vocab 构建在 init 里读它）；②双 config 坑再证：app 包 config 名被 internal/config 占用，gaea/config 需别名；③ConventionDirs 变量 .gaea 首位即优先序，无 ConventionDirsAt。**门禁**=Go 全量 65 包 0 FAIL/绑定 624 零变更/版本三处 4.225.0。**审计余项入拍板池 §5.4/5.5**：DCMA 14 点形态（阈值数据化 vs 整体转技能）、造价 R1-R3 技能化（内嵌 Apply 留痕链路）——涉产品面待拍板。

- **最新发布：v4.224.0（2026-09-11）「架构修正：规范知识出内核——排版细则改技能按需加载」**——**用户拍板的架构红线，永久有效：领域规范/行业模板/检测规则不应植入代码，应以 skill 技能按需加载防臃肿；内核只留通用机制；新需求先问「能否做成技能」**。动作=①撤销 v4.223 Go 侧 FormatChecker（format.go+test 删、registry/gaea_lint/docxedit.ReadDocumentXML 全回退——无消费者的取数口也不养）；②技能包 .gaea/skills/gongwen-9704（SKILL.md=红头要素表七项+排版细则表六项+换算备查+判定纪律「容差内即符合/不适用≠缺失/实测值进建议/宁缺勿误」；scripts/docx_layout.py=stdlib 解析 docx 排版事实，最小 docx 实测通过）；③既有红头要素/造价表式 Go checker 同属「规范在码」列观察池候选，迁移涉 GaeaDocumentLint 行为变更待拍板。**门禁**=Go 全量 0 FAIL/绑定 624 零变更/版本三处 4.224.0。

- **最新发布：v4.223.0（2026-09-11）「办公·中文文书规范包第二刀：GB/T 9704 排版细则 lint」**——板块并行池开工（roadmap 办公#5；非阶段旗）。纯 Go 零绑定零前端（LintReport 走既有 GaeaDocumentLint/前端规范包分组）。**缺口**=红头要素（v4.1c）只查文本层，GB/T 9704-2012 排版层无人管。**落地**=standard 包 FormatChecker：ParseDocxLayout 纯函数（document.xml 容错子集解析：pgMar/spacing/ind/jc/rFonts/sz/t，run 级 rPr 优先段级缺省，表格内段落计入）+ 六项检查（页边距 37/35/28/26mm±2 容差聚一条/标题二号 22pt 小标宋只建议不判定/正文三号仿宋多数票/行距 28~30 磅固定值多数符合即符合/首行缩进 2 字符/层级标题一、黑体（一）楷体字体名缺失不判宁缺勿误；数据不足=「不适用」不误报缺失，违规带实测值建议）。取数口=docxedit.ReadDocumentXML 新导出；LintDocumentWithLayout 聚合（layout=nil 行为不变）；gaea_lint.go docx 分支解析失败 fail-open。**测试**=Go +4（XML 片段解析事实/合规全过/违规六项实测/不适用+无边距）。**门禁**=Go 全量 0 FAIL（纯 Go 刀 vitest 以 v4.222 基线为准）/绑定 624 零变更/版本三处 4.223.0。**坑**=①range 头里组合字面量方法调用须括号 `(FormatChecker{}).CheckLayout(l)`（语法错 setup failed）；②attr helper 收 xml.StartElement 值非指针。**注**：本刀植入式 checker 已被 v4.224 按用户架构红线撤销并转为技能包 gongwen-9704，规则知识保留在技能资产中。

- **最新发布：v4.222.0（2026-09-11）「6.3 余项：dag_plan 增量改图——run_id 编辑模式」**——纯 Go 零绑定零前端改动（设计档 §6 又清一条，**DAG 线自然收尾**：§6 仅剩等外部设计的「运行中 steer 直穿+分级审批」与「够用」的 DeliverableRegistry）。**缺口**=改单节点指令也要整链重规划：run id 换、既有状态与验收全丢。**落地**=dag_plan 增 `run_id` 可选参=编辑模式（省略=新建原行为不变），核心 internal/gaea/dag `ApplyEdit` 纯函数原地调和：节点按 id 对账，title/prompt/dependsOn 全等→原样保留（状态/产物/ref/运行痕迹延续）；有变/新增→回 pending；运行中拒绝；被移除节点仍被依赖=悬空依赖 fail-closed 不静默断链；**改已验收节点=撤销验收**（产物是旧指令的产出），RevokedAcc 计数透出人读文案。**测试**=Go +2（dag 调和四态+撤销验收+三守卫；app 编辑路径全流程+三拒绝）。**门禁**=Go 全量 0 FAIL（纯 Go 刀 vitest 以 v4.221 基线为准，v4.209/4.212 先例）/drift@624 零变更/版本三处 4.222.0。**坑**=Edit 工具old_string 带函数签名行而 new_string 忘带会静默删函数行——本刀在 dag.go Derived 与 template.go atomic 两处触雷均当轮自愈（go build 立验）。

- **最新发布：v4.221.0（2026-09-11）「6.3 余项：子代理证据落账+按会话归因+波内并行」**——三合一刀（设计档 §6 首条清掉）。**起因=真机归因链核查发现结构性缺口：子代理写盘从不落 Journal**（子代理 Options 无 JournalDir/SessionID→flushJournal 跳过；v4.219「窗口归因」真机恒空集，fake-runner 测试靠手工塞卡绿，壳内真机走查恰挂观察池未跑——**测试绿≠真机通，fake runner 模拟什么归因就是什么**）。**① 子代理证据落账**=TaskTool 增 journalDir（SetSubagentJournalDir，boot 注入与主执行器同目录）+ Options.SessionID=run.Ref 经 runSubSession 下发——task 工具/RunNew/RunFollowUp 三路径全覆盖（汇于 runSubSession 是全覆盖关键）；ephemeral 无 ref 不启用宁缺勿错；顺带收口「task 子代理编辑对版本时间线/回滚不可见」审计缺口（6.1 同向，子代理开始建回滚基线）。**② 按会话归因**=节点 outputs=SessionID==ref 的证据卡 Target（精确归因，主对话同期写盘不再并入；ref 空回退窗口增量）；UI 口径注三语同步。**③ 波内并行**=dagExecute 波内 goroutine 并发（归因已精确不再依赖窗口不重叠；失败级联/终止级联/互斥落盘不变；波内并发由测试互等信号锁死）。**测试**=Go +3（agent 落账往返+独立会话文件/app 并行互等+会话隔离/生命周期 fake-runner 改真实落账形态）。**门禁**=Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.221.0。

- **最新发布：v4.220.0（2026-09-11）「6.3 余项：流水线模板库——存模板一键重建」**——阶段六收官后续刀（设计档 docs/gaea-office-dag-63-design-2026-09.md §6 余项之二清掉）。**模板只取图形状**：internal/gaea/dag/template.go 纯函数包（FromRun=剥状态/产物/ref/运行痕迹+空名回退 goal 截 24 字+限长 40 rune 显式报错；Instantiate=全新草稿 run 不自动起跑——起跑仍人拍板与整链首跑同闸；TemplateStore 落 `<cwd>/.gaea/work/dag/templates/`，**run 档同域子目录**，Store.List 只读顶层 *.json 互不混；Save 前全量 Validate=成环/悬空依赖/坏形状拒入库）。**绑定 620→624**（GaeaDagTemplateSave/List/New/Delete，Office 门面手工补行）；重建=模板当前形状快照，改模板不影响已重建 run；删除只删模板档（已重建 run 不受影响）。前端 DagPanel：模板折叠条（默认收起、有模板才显形）逐条「新建/删」+ run 卡「存模板」内联输入（默认带出 goal 截断，Escape/取消收起）；模板拉取失败静默不挡主列表，dag-retry 顺带重拉模板；?mock=1 预置「月度经营报告」模板。**测试**=Go +6（template_test：剥痕迹/名规则/实例化全新/存储往返+坏形状拒+穿越拒绝/列表倒序+与 run 隔离；app TestDagTemplateFlow=存→列→重建→重建链 fake-runner 跑通全 done→删→再删报错）、vitest +4（显隐折叠/新建重拉/存模板提交/删除+失败静默）。**门禁**=Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.220.0。**坑再证**=gofmt -w 会把 bindings_office.go 整文件重排（import 排序+全文件换行重写 492 行 diff）——门面只手工补行，格式化器/生成器禁过门面文件；bindingNames.ts 由 `gen_bindings -names` **只打印不写盘**，按输出手工补行。**§6 余项剩**：波内并行（等按 ref 归因定案）/运行中 GaeaSteer 直穿+危险操作分级审批（等审批闸分级面）/dag_plan 增量改图/产物登记 DeliverableRegistry。



- **v4.174–v4.178（2026-09-09）瘦身 P3 版3/P4 三刀与造价条目匹配键刀**——全文迁入 docs/archive/agents-version-history-2026-09.md（2026-09-11 三迁腾预算，check-docs 指令预算守卫）。
## 执行纪律：默认并发子代理（2026-09-03 用户强化习惯）

用户已把「并发子代理」从点名指令固化为**默认习惯**：后续任务默认按此执行、
无需再次点名「并行使用子代理」。

1. **开工先拆线**：任务含 ≥2 条互不相交的线时，先列出「线 × 文件足迹」再动工；
   2-4 条独立线用并发子代理分头执行（运行环境 4 并发槽位），主代理负责定契约、
   跑全量门禁、集成收口——v4.54-v4.59 的「三线并行 + 主代理收口」即标准形。
2. **单线也倾向派活**：一条独立成刀的任务（调研/实现/测试/文档）若体量超过一轮，
   优先派子代理并发执行，而不是排队串行。
3. **足迹互斥铁律**：线间文件足迹不相交；契约/生成类文件（types.ts / bridge.ts /
   mock.ts / 三语字典 / gen_bindings 产物）必须指定单一负责人，生成动作由主代理在
   所有后端子代理完成后统一执行，防止并发写覆盖。
4. **每刀回写**：刀末把本次分线/收口经验（含教训）写回本文件与 `.gaea/progress.md`，
   让习惯持续强化；若某刀必须串行，收口时说明原因。

## 交互纪律：不许用「等待」换时间（2026-09-10 用户拍板）

用户原话（对上一轮执行的批评）：「后台运行的东西你为什么要等待」「在浪费我的时间」。
耗时本身不可怕，**干等与重复**才是浪费。后续每一轮工作按下办：

1. **后台任务不阻塞**：起后台任务后立刻去做别的事（改码/读文档/跑定向用例/写文档），
   收到完成通知再取结果。**禁止起完就守着等**——那几分钟是白扔的。
2. **重活先报 ETA 再跑**：实测耗时——全量 vitest ≈ **140s**（324 文件 2799 例）、
   `scripts/ci.ps1` 全门禁 ≈ **5–7 分钟**（Go 全量 + 前端 lint/build/vitest）、
   `go test ./...` ≈ **2–4 分钟**。开跑前一句话说清「跑什么、约多久」，用户可否决。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
4. **超时上限按实际需要写**：不要写 900s 这种夸张数字——那是上限不是耗时，
   只会造成「要卡你十几分钟」的观感。
5. **不把「等待」当进度表达**：进度由已完成的具体产出说话，不由轮询次数说话。
   用户催问时先答「在跑什么、还要多久」，再继续。
6. **能并行就并行**：长任务是可并行的（截图取证 / 读码 / 单测互不依赖），
   串行排队本身就是一种浪费。

> 与上一节的关系：并发是**手段**（把时间省下来），本节是**约束**（别把省下的时间又等回去）。

## 工装：CDP 走查与前端调试（2026-09-10 增补）

- `scripts/cdp-walk.mjs` 支持 **`--target <url 片段>`**（多标签时选定目标页，缺省取第一个
  page）与 **`@文件路径` 传入 JS 表达式**（PowerShell 5.1 传原生命令参数会吃掉内层引号，
  长表达式一律写文件再 `@` 传入）。真机走查配方：无头 Edge
  `--headless=new --remote-debugging-port=9333 --window-size=1440,920 --user-data-dir=<tmp>`
  + Vite dev（`?mock=1`）→ 本脚本 eval/截图。
- **Vite dev 缓存坑**：同一 CSS/源文件在**同一秒内的两次编辑**，mtime 粒度相同会让
  HMR/转换缓存认不出改动（表现为浏览器仍跑旧样式，reload 也无用）。处置=改完
  `(Get-Item file).LastWriteTime = Get-Date` 再 reload。
- **`vitest` worker 上限**：`maxWorkers` 必须按**物理核**（≈逻辑核/2，夹 [4,8]）而不是
  逻辑核——超配会把单用例墙钟拉到 CPU 时间的数倍，重组件首个 `await import` 直接
  撞 `testTimeout` 假红（详见 `frontend/vite.config.ts` 注释）。

## 长期规划（权威，2026 定稿）

- **下一阶段规划 = `docs/gaea-next-stage-plan-2026-09.md`（2026-09-10 活跃指导）**：接棒长期规划（阶段一~四已收官）——阶段五 记忆 OS+上下文编译（主轴）/阶段六 书斋纵深——办公主角（可审计默认化·记忆驱动项目本体·多文件 DAG），进度（DCMA 体检·蒙特卡洛工期带）与跨域 EVM 为两翼/板块并行池/拍板池（信息价接入·MCP 翻案·壳外 computer use）/维持轨。提方向前先对其 §0 在册边界（GoalCard v3.6.0 撤下、MCP/平台化不进规划、不做算量、DSH 独立窗口等）。
- **用户拍板（2026-09-08，收敛计划 §0）：性质路线=A「终极个人工具」**——产品化
  降为期权不作承诺，不为想象中的用户写代码；**LICENSE=私有 All Rights Reserved**
  （根目录 LICENSE），未来开源须另行发布开源许可证覆盖对应模块并与私有部分区隔。
- **瘦身长期总规划（2026-09-08 立项）= `docs/gaea-slim-masterplan-2026-09.md`**：七面
  （认知/资产/结构/轮子/运行/产物/数据知识）×六阶段（P0 基线→P5 维持）；治理规则仍在
  收敛计划。防复发规约自 P1 起生效（新下载走 saveExportBlob/新选取走 pickFile/新 diff
  复用 lib/diff/新 slug 用 strutil.TitleSlug/新解码用 b64ToBytes/NAVIGATE 用 manifest id/
  新域先问「能否是文档」）。
- **用户再拍板（2026-09-08）：工位与乐园并列平级**——办公不是乐园的上位核心，瘦身
  不是「收乐园保办公」；认知面问题=13 板块平铺单一导航面、并列结构未表达。P2 措辞
  已由「乐园折叠」改为「双空间并列落地」，乐园板块在乐园空间内一级可见。
- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  ~~"陪伴×办公融合"（旧 v4.3）已删除~~。
- **本轮关键纠正**：灵犀 = 金山 WPS 独立 AI 办公 Agent（非阿里/通义系）；
  WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台（非 Kimi 系；Kimi Work 是月之暗面的）。
- **i18n 决策（2026 追加）**：采用审计 §405「诚实 zh-only」选项——**壳层 chrome +
  设置外壳三语**（已完成，消灭壳层混合语言根因）；**页面内容层保持 zh 单语**，
  不再逐页铺 en/zh-TW 字典（个人中文工具无国际化受众，~5000 字符边际价值≈0；
  未来需国际化时按 S2.3b WireShape 模式整页迁移）。
- **文档纪律**：docs/ 旧调研/已落地计划已归档至 `docs/archive/`（见其 README）；
  后续会话以本文件 + 长期规划 + `.gaea/progress.md` 为权威，勿引用 docs/archive 结论。
  **新文档必须登记 `docs/README.md`**——守卫 `node scripts/check-docs.mjs` 四查（孤儿登记 /
  `docs/` 悬空引用 / **本文件字节预算 ≤ 65536 B**，超了尾部纪律段就对会话不可见 /
  **含非 ASCII 的 .ps1 必须带 UTF-8 BOM**——2026-09-10 实撞：编辑 ci.ps1 丢 BOM，整条门禁语法错静默不跑），已接入 `scripts/ci.ps1`。
- **执行审计（2026-08-30）**：`docs/archive/audit-2026-08-30-v4-execution-review.md` 记录
  v4.x 全量「承诺 vs 代码」对照——裁决=最小版执行（骨架真、纵深欠账）。红线缺口
  三条（记忆注入跨空间未接线 / 任务分账未启用 / 事件过滤仅 1 处）与补课刀序见该文
  §B/§E；后续每刀验收新增「纵深检查」，发布说明必须列欠账清单。
- **执行状态（v4.8.0 后）**：审计欠账大面收账——读屏纵深（多显示器/OCR 本地
  摘要/截图留档）、intent LLM 兜底分类器（默认关，白名单+置信门+硬超时）、
  生图产物 CardPath 接通、iLink 微信通道离线收敛（限频/下载防线/识别管线/
  防御解析/SendFileCard seam）、全局离线模式总开关（EngineType.IsLocal +
  routeModel 云过滤）、成本知识图谱可视化（BuildGraph + CostGraphView 第 8
  模块，绑定面 533）、实时语音 Realtime S0 铺底（internal/realtime seam）。
  剩余欠账（Realtime S1/S2、iLink 真机窗口、离线模式设置 UI、权限升级请求+
  stubGate 竞态、XlsxPreview 虚拟滚动/生命库可写化=观察项）见
  `releases/v4.8.0.md` 欠账清单。
- **欠账收尾小步（2026-08-30，v4.8.3 后）**：VoiceStart realtime 门小修
  （端到端回复走服务端 response 事件，whisperChatFn=nil 也可启动，拼接
  管线双门逐字节保留）；持久化套件统一（desktop_session 原子写 + archive
  JSONL 单次 Write 落整行）；XlsxPreview 大表格行虚拟滚动（观察项收账，
  300 行以上只渲染可见窗口 ±overscan）。Go 全量绿、vitest 809/809、
  tsc/eslint 0、绑定面 535 不变。
- **下一执行**：v4.8.3 已发布（微信图片双向真协议）；剩余——Realtime 真机
  验证轮（真 key 下端到端对话/打断体感/AEC 实效，S2 骨架已就绪待真机数据）；
  手写体识别质量复测（多模态 Qwen 升级后）；iLink 语音/视频等未探明 item
  维持宁漏勿误静默跳过；生命库可写化=观察项。

## 版本状态（历史存档）

> 2026-09-09 整理：本段逐版历史（v2.x~v4.135，约 2100 行）整体迁往 `docs/archive/agents-version-history-2026-09.md`，按需检索；当前动态见顶部速览，发布物全文见 releases/。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；429 个导出方法拆 10 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1，~110GB 模型：35B 对话 ×2、zimage-turbo、voxcpm2、
  mineru、embedding/reranker、paddleocr、sherpa-onnx 等）；gaea 的聊天/视觉/检索/OCR/解析/ASR/TTS/生图/翻译
  全部依赖它，herdsman 升级可能破坏契约——用 App.HerdsmanProbe 启动探测

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建（本沙箱：`cd frontend; npm run build` → `wails build -s`；本机：`cmd /c build.bat`），
   产物 build/bin/gaea.exe（同时复制到桌面）；本机 build.bat 已内置真实退出码检查 +
   默认自动冒烟（.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200，失败即停；
   `build.bat skip-smoke` 可跳过，发布前不得跳过）
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`；
   **本地产物只保留最近 5 版**（2026-09-10 用户拍板）——发版后删掉第 6 新的那一版，
   更早版本的身份以 `SHA256SUMS-vX.Y.Z.txt` 为准（旧 exe 不入库）
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测不达标（耗时长、音色男女混乱、克隆不稳定）。
> 下方 VoxCPM/Vulkan 相关方法保留为「已废弃教训」，勿重新安装；当前本地 TTS 为 CosyVoice2。
> 注：herdsman 侧实测 voxcpm2 可用（冷启动约 50s，不支持预设音色），qwen3-tts-* 未安装。

本机（Radeon 8060S 核显 / 64GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010。

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）

- `8030`：主后端 `C:\AI\llama-omni\build\bin\llama-tts-server.exe`（llama.cpp-omni，C++/ggml + Vulkan）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 8060S 识别 `KHR_coopmat + bf16`，全量 29 层 offload Vulkan0，加载约 2s
- `8021`：备胎 ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`）
- `8020`：适配层 `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 入口，契约不变）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030/8021/8020）

### CosyVoice2（端口 8010）

- `C:\AI\cosyvoice\server.py`：LLM 段 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 段 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）

- 中文女 `zh_female.wav`（f0≈221Hz）、中文男 `zh_male.wav`（f0≈133Hz）
- 英文女 `en_female.wav`（f0≈191Hz）、英文男 `en_male.wav`（f0≈109Hz）
- 参考音频 ~7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速教训，勿重蹈覆辙）

1. 不要再用纯 ROCm PyTorch 追赶速度：iGPU 共享内存架构下 ROCm 与 CPU 基本相同（RTF ≈1.06~1.12）；
   Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65~0.84）
2. 构建：MSYS2 UCRT64，`cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`
3. 坑 1（端口绑不上）：server-voxcpm2 会构造 SSLServer，空证书导致 is_valid_=false 任何端口 bind 失败；
   本地回环不需 TLS，改普通 httplib::Server
4. 坑 2（克隆近静音）：AudioVAE 参数特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major）
5. 坑 3：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. 坑 4：VoxCPM Python 长文本 CFG 2.0 会「静音+整段重试」（RTF 4.8~7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. 网络：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65~0.84（6 步）、语音设计 0.57~0.60；同 seed 输出确定

### 详细记录

- `docs/archive/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/archive/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）

- gaea 启动时后端 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 → `App.StartLocalTTSService(engineId)`；
  引擎连接测试兜底 ensure（等约 8s）；TTS 合成前兜底 ensure
- 实现：`internal/app/tts_service.go`（core.ensureLocalTTSService 幂等 + 异步轮询，emit `tts-service-status`；
  CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

## 已知注意

- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）
