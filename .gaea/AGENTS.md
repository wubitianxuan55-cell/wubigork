# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。
- **最新发布：v4.261.1（2026-09-12）「修复：原罪插图报『marshal image request: 图生图需要提供参考图』」**——**根因**=v4.259.0 参考槽门控只判「后端×模型能不能用」，不看「这一轮有没有取到图」：①没选角色 ②只有远端剧照（`sinRefDataURL` 跳过）③本地文件缺失 → `refImages=0` 却仍以 `mode=img2img` 发请求，Herdsman 图生图端点整单拒绝（`ai/image_openai.go:buildImg2ImgBody`，被包成 `marshal image request: …`），而回退只在 `refImages>0` 触发。**真机取证**=日志 `gaea-20260912.log` 20:26 两连发（图片后端 Herdsman）+ 用户 `sin/cast.json` 不存在=没选角色（分支①）。**修复**=①新纯函数 `sinResolveRefImages`（参考图优先·其次剧照·远端与缺失跳过，返回图/名/**台账归属 id**）；②解析为空即 `refOK=false` 并**清空 mode/refMethod** 走纯文本（选角色无图给「该角色没有可用的参考图/立绘」，没选角色不弹噪声）——「用不上参考槽」不再等于整单失败；③顺带修**台账归属错位**（多角色里只有第二人有图时记错人）。**测试**=Go +3（红→绿：改前 3 子例全红 `mode=img2img refs=0`；纯函数矩阵 + 端到端 3 子例）。**门禁**=ci.ps1 全绿/版本三处 4.261.1/零新绑定；exe 49,164,288B SHA256=CCE99E91…D5CAB（桌面副本同哈希，冒烟 200）。**真机走查**=v4.261.1 exe 打开原罪故事 → 两个插图位 20s 内自动出图（产物落 `%APPDATA%\gaea\sin\art\`、回写 extra.illustrations），零失败/零异常。**文档**=设计档 §12.5 + releases/v4.261.1.md。
- **最新发布：v4.261.0（2026-09-12）「UX 线第三刀：gaea EmptyState 收敛到 V3Empty——全站空态语言归一收官」**——v4.260.0 避开项清账（彼时 sin 线消费中暂缓，用户确认收尾+工作区干净核实后动手）。gaea/components/EmptyState.tsx（原 13px 40% 灰字）改 **V3Empty 薄壳**（轨道环+on-surface-variant 文字，py-14 保留，**API {message} 不变**），消费方（WhisperGraphPanel 两处 / sin StoryStream）零改动——**小刀坑=跨目录 import 层级：gaea/components→src/components 是 ../../ 不是 ../，tsc 拦下修正**。至此空态三套视觉归一：无图形场景一律轨道环、语义图标场景保留 icon。**测试**=消费面 27 用例+tsc 0+eslint 0；原罪页真机正常（用户已有故事非空态场景，空态渲染由 vitest 覆盖）。**门禁**=ci.ps1 全绿/版本三处 4.261.0/exe 49,164,288B SHA256=38a912c6…03cac。
- **最新发布：v4.260.0（2026-09-12）「UX 线第二刀：一致性收口——--radius 桥接 M3 + modelcenter 空态收敛 + --sched-dim 修复」**——「优化 UX、美化前端 UI」第二刀。普查先行：**24 页次（12 rail 页×明暗两态）CDP 起壳巡检全零错误**、孤岛 CSS 实测近乎零 hex（imagegen/modelcenter/settings/programming/weixin/character-library 均 0；novel 19 处为功能性设计意图不动）——明暗一致性基础好无硬伤；普查「三套圆角并存」实测收窄=定义 3 处消费仅 1 处。三小刀：①**--radius 桥接**=gaea/styles.css `--radius: 9px` 孤值改 `var(--md-sys-radius-md, 12px)`+redesign.css 同值死覆写删除；消费点仅 .context-menu 一处，办公工作台内视觉零变化。②**modelcenter 空态收敛**=EmptyState（API 不变）双形态：**无 icon 时渲染 V3Empty compact 轨道环**（hint 走 children 弱色行）、有 icon 保持语义图标；13 调用点实测=10 带 icon 不变/3 无 icon（Benchmark 加载态/LLM·Specialty 搜索空态）自动获轨道环；V3Empty 加 compact 档（48px）。③**--sched-dim 未定义修复**=.sched-aoa-mode-btn 的 var(--sched-dim,…) 变量从未定义恒走 fallback（v4.255 发现项），桥接 on-surface-variant；**.sched-dim 是同名 CSS 类非残留**（73/777/1012 行）。**避开项**=gaea/components/EmptyState.tsx 收敛暂缓（sin 线消费中，待稳定另刀）。**版本顺延**=原按 v4.256.0 预备，同树另一执行体抢先 commit 四刀（v4.256~259 原罪板块），顺延 v4.260.0。**测试**=modelcenter 107 用例+tsc 0+eslint 0+ci.ps1 全绿+24 页次两态巡检零错误+截图目检无回归。**门禁**=版本三处 4.260.0。
- **最新发布：v4.259.0（2026-09-12）「原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）」**——上刀把生成进度做完后原罪仍缺一环：角色选进来了、插图里的人不一定是她。本刀接绘梦既有 T2 角色参考槽，让「角色库→故事角色→插图人物」闭合。**①文本锚点（所有后端受益）**=sinAugmentPromptWithCast：画面描述未写外观时补「人物锚点（名字）：外观；身材」（≤120 rune），已写过不重复（前 20 字命中即跳过）。**②图像参考槽（仅 ComfyUI/Herdmans 图生图）**=取角色参考图（referenceImages 优先，其次剧照）走 mode=img2img+refImages+refMethod=img2img（denoise 0.65），复用 generateImageInternal 参数化入口（不新增第二条生成实现）；参考图=data URL 原样/本地路径读文件转 data URL（comfyui uploadImage 与 herdsman img2img 只认 data URL）/远端与缺失跳过并 warn。**③锚定谁**=sinPickRefCharacters：提示词点名优先；都没点名时仅当只选一个角色才锚定；多角色未点名→不锚定（防互串）。**④能力门控**=sinRefPlan 纯函数矩阵（comfyui+krea2*/z-image-turbo ✅ / comfyui 其他模型 ⏭ 给原因 / herdsman ✅ / xai·glm ⏭ 给原因）；不硬塞（硬塞会整单报错）。**⑤回退**=参考槽失败自动退回纯文本重试一次 + 标 ref_fallback（插图不因参考图失败）。**⑥可见**=caption 徽标「角色参考：<名字>」（accent）「已补外观锚点」「参考图不可用 · 已按纯文本生成」；无角色可锚时不弹噪声徽标。**未做**=ipadapter/pulid（image_comfyui.go 未实现，comfyResolveRefMode 如实报错）。**测试**=Go +7（能力矩阵/点名与单角色兜底/锚点去重与空值/data URL 转换与远端跳过/参考槽透传/不支持后端跳过且文本锚点仍在/失败回退）+vitest +2。**门禁**=Go 全绿/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@640（零新绑定）/E 系列+仓库卫生守卫绿/版本三处 4.259.0；走查=故事带「林晚」→ caption 显示「角色参考：林晚」「已补外观锚点」（零错误零异常）。**文档**=设计档 §12 + releases/v4.259.0.md。
- **最新发布：v4.258.0（2026-09-12）「原罪插图：生成进度动画 + 串行队列 + 真停止」**——用户提「增加图片生成动画知道完成进度情况」。**改前**=插图只有转圈+「正在生成插图…」：无百分比、无阶段、看不出在跑还是排队，多张并发时进度还无法归属。**①进度抽核共用**=新 `components/imagegen/GenerationProgress.tsx`（自绘梦 GenerationBar 抽出：确定态进度条 +「全部: N%」+「当前节点: 中文阶段名」+ 已用时；类名 `.ig-progress*` 与阶段名表单点维护→绘梦/原罪同实现防漂移；imagegen.css 的进度块与 keyframes 迁入自带 generation-progress.css）+ 新 hook `useComfyTaskProgress`（只在生成中轮询/单次读失败保留上一帧/卸载清定时器）；**诚实口径**=ComfyUI 有实时百分比走确定态，其余后端不提供 → 不定态光带 + 本地秒表，不编造百分比；补 reduced-motion 降级。**②串行队列**=新 `pages/sin/illustrationQueue.ts`：同板块一次只跑一张（图像后端单任务+进度单任务维度，并发会抢同一百分比）；排队显示「排队中 · 前面还有 N 张」，轮到自动切实时进度；失败不阻断队列。**③真停止**=新绑定 `SinCancel(topicID)`（绑定 639→640）：中止底层流（ctx）但**已生成部分照常落库**（extra.cancelled=true，用户消息不丢），无在途流如实报错；插图「取消」接 `CancelImageGeneration`；sinRuns 按话题单飞登记。**④顺带修两个自伤缺陷**=（a）useSinStory 的 deadRef 跳过收尾 + StrictMode 双挂载 → sending 永久 true、输入框被「请先完成或停止当前回合」锁死（v4.256 起潜伏）；（b）Composer「停止」是空实现（onCancel 传 ()=>undefined）→ 现接 useSinStory.cancel（后端取消+本地立即解封）。**测试**=Go +2（取消保留部分+extra.cancelled/无在途流报错+跨板块守卫）；vitest +7（队列串行与位次/进度轮询四态/插图卡确定态与不定态/已有产物不重跑/hook cancel 复位）。**门禁**=Go 全绿/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@640/E 系列+仓库卫生守卫绿/版本三处 4.258.0；浏览器走查=生成中两处进度条（「生成中…」/「排队中 · 即将开始」）+取消一张另一张继续+完成后停止按钮消失（零错误零异常）。**文档**=设计档 §11 + releases/v4.258.0.md。
- **最新发布：v4.257.1（2026-09-12）「修复：真机插图报 AttachmentDataURL is not a function」**——用户报「插图失败：e(...).AttachmentDataURL is not a function」。**根因**=S2-3 兼容代理（`window.go.app.App`，bridge/http.ts::ensureLegacyAppProxy）只按**字面名**在各板块门面查方法，认不出 gaeaToGaea 短名映射——旧调用点写 `AttachmentDataURL`，OfficeB 真名是 `GaeaAttachmentDataURL`，查找落空返回 undefined，调用即抛 not a function。**为何只有部分功能中招**=bridge 代理（app.*）会做映射（PortraitImg/消息内联图正常），而 `api/image.ts` 图像域 helper 走旧直连 `window.go.app.App`，其中 4 个 Gaea 前缀方法全中招：AttachmentDataURL / SavePastedImage / RecognizeImage / OCRText（原罪插图、章节配图历史缩略图、画室识图试用；部分调用点 catch 成空=静默失败）。**修复**=①兼容代理补映射回退（字面名优先→映射名，仍未命中才 undefined，不塞假实现）②`api/image.readFileAsDataURL` 改走规范 bridge 路径（映射+mock 兜底），bridge 未认领才退回旧直连 ③HTTP 模式门面补齐表 +SinB。**测试**=vitest +3（新 legacyAppProxy.test.ts 真机形态假门面：短名映射命中/字面名优先/未知 undefined/readFileAsDataURL 经映射命中 OfficeB）。**门禁**=tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@639/版本三处 4.257.1；浏览器 mock 走查原罪插图仍正常（2 张、零异常）。
- **最新发布：v4.257.0（2026-09-12）「原罪三刀：与办公硬隔离 + 模型中心独立卡片 + 角色库接入」**——v4.256.0 交付原罪后用户追加三条口径，逐条落地。**①硬隔离（组件可复用=渲染层，数据面一律不通）**：插图产物改落 `<用户配置目录>/gaea/sin/art` —— 为此把绘梦生成链「产物归属」参数化（新 `generateFreeImageProvenanced(sourceBoard, saveDir)`，绘梦侧逐字节不变），原罪传自有 saveDir + `source_board=sin`：不写办公工作区 `.gaea/`、不依赖办公 ImageSaveDir 与小说项目目录；**顺带修真缺陷**：改前原罪插图被登记两次（生成链内 source=imagegen + 调用方补 source=sin → 画室同图两条），现登记只在生成链内发生一次且来源正确（`recordImageHubGeneratedFor`）。**工位检索硬边界**=`wssearch.isNoiseRel` 改前只跳 `.gaea/play/exports`，play 分区其余子目录（画室台账/章节配图/原罪产物）漏进办公关键词检索；现**整个 `.gaea/play` 判为噪音**（S2.1 红线；fileindex 本就跳 `.gaea`）。**记忆面零接入**（不读不写 hephaestus facts 与 hermes.db，故事上下文=本次故事历史+已选角色）、**模型路由独立**（routeModel("sin")，办公换模型不影响原罪）、**角色选择自存**（sin/cast.json）。**②模型中心独立卡片**=FEATURES 增 `{key:'sin',label:'原罪'}`+FireOutlined+卡片说明（config 三键上一刀已接线）；走查确认在位。**③角色库接入**=右栏「角色」卡+多选弹层（复用 PortraitImg 三态）→ 按故事持久化 `SinCastGet/SinCastSet`（悬空 id 过滤/去重保序/单故事封顶 8，返回生效清单前端回填）→ `SinStream` 注入角色块（名字/性别/年龄/定位/外观/身材/性格/背景/动机/状态/口吻/≤3 条说话样例，空值整行不写不编造 + 声明「不得改设定或改名」）；只引用不修改角色库档案。**未做**=角色剧照/参考图当图像参考槽（人物一致性锚定，需 img2img+参考槽，留拍板）。**绑定 637→639**（gen_bindings 再生 + office/cost 门面按纪律还原）。**测试**=Go +8（cast 往返/守卫/封顶/角色块注入/数据根隔离/wssearch 整分区）+vitest +5；走查=模型中心卡片在位+角色弹层三人列示+「带入故事（1）」chip 落地（零错误零异常）。**门禁**=Go 全绿/drift OK@639/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/E 系列+仓库卫生守卫绿/版本三处 4.257.0。**文档**=设计档 §8 硬隔离清单+§9 角色库接入+releases/v4.257.0.md。
- **最新发布：v4.256.0（2026-09-12）「闲庭新板块『原罪』：对话式图文混杂故事创作（复用办公组件）」**——用户提「在闲庭增加一个单独的功能板块，复用办公的组件，主要用于与 AI 对话，可以生成图文混杂的 H 故事小说，板块取名原罪」。**形态**=闲庭（play）一级板块 `sin`（layout=full，menuOrder 13）：左「故事」架 + 中故事流/办公输入器 + 右「玩法·插图协议·边界」三卡，窄窗两级收起侧栏（1180/880）。**组件复用零新建**=输入器=办公 `Composer`（附件/粘贴表格转 Markdown/截图/裁剪/识图/OCR/输入历史全继承）、正文=办公 `Markdown`（本地文件 FileChip/代码折叠/mermaid/genui/消毒同链）、`ToolbarButton`/`EmptyState`/`LocaleProvider` + 办公三件套样式；**不改办公组件本身**（原罪只做消费方）。**后端（新门面 SinB，绑定 628→637）**=①会话复用 `internal/chat` 统一 Store（mode=sin）：与聊天板块**同表不同域+双向隔离**（SinTopicsList 只见 sin；ChatTopicsList 过滤 sin——聊天 persona 分支不认识该模式，放进去既污染列表也走错生成路径），跨板块操作 fail-closed（sinTopicGuard 先验 mode）；②文本=既有 `ChatStreamChunks` 流式通道 + 新功能级路由 `routeModel("sin")`（config 三键 func_sin_engine/model/enabled；模型中心新增「原罪」绑定 → 可单挂长上下文/无审查本地模型，未绑定回退全局）；事件 `sin-stream:<runID>` 与 chat-stream 同形；前情装配=最近 12 条（单条 1500 rune）+历史标记替换「（已配图）」；play 护栏温度/上限只降不升；③插图=绘梦 `GenerateFreeImage` 同一条链（image_safe_mode 照常生效）+台账登记 `space=play/source_board=sin`（画室可按来源筛）→ 路径回写消息 `extra.illustrations[cue]`（**重开不重生成**；chat store 新增 GetMessage/UpdateMessageExtra 两个窄方法）；④`SinExportMarkdown` 图文导出（未生成留占位，不静默吞）。**插图协议**=模型另起一行输出 `@@插图|画面描述@@`（一轮≤2 张），前端按出现次序编号（0 起）=cue 键就地生成嵌入正文流；**正文原样落库**（正文=模型输出可回溯），导出时才做标记→图片替换。**内容边界零新立**=沿用 `docs/ADULT_MODE.md`（个人非商用、默认开启、不回避不说教不医学化），提示词写死四条硬边界（全员成年/禁真实在世人物/合意基调/用户叫停即停）并有专项测试锁定。**接线六处**=bridge 分域 SinBindings（AppBindings 第 11 域）+bindingNames（drift 637）+spaceBindings 归 play（锁 450→459）+dev mock（内存故事库+`window.runtime.EventsEmit` 模拟真流式+占位插图，`?mock=1` 可走查图文版面）+manifest（前后端各一处）+PageRegistry/启动器文案/模型中心 FEATURES。**测试**=Go+7（域隔离与守卫/流式落库与消息 id/未知话题拒绝/插图回写与导出/前情装配去标记+封顶/提示词硬边界）+config 往返 1+vitest+27（storyText 10/useSinStory 5/事件频道 2+既有锁更新）。**门禁**=Go 全绿/drift OK@637/tsc 0/eslint 0 error/vite build 成功/vitest 334 文件全绿/E 系列与仓库卫生守卫绿/版本三处 4.256.0。**文档**=docs/gaea-sin-board-design-2026-09.md（设计基线，已登记 docs/README.md）+releases/v4.256.0.md。
- **最新发布：v4.255.0（2026-09-12）「前端 UX 精修一版三刀：进度计划主题桥接 + 全站空态统一 + 全局焦点环」**——用户提「优化 UX、美化前端 UI」，Explore 普查先行（四代令牌/15 页样式矩阵/UX 底座），一版三刀零 Go 零绑定：①**schedule.css 接入主题**——明暗切换无 CSS 类钩子（全靠 App.tsx 换 :root 变量），纯 hex 消费者天然脱离主题；顶部四令牌（关键红/工作蓝/链接灰/分组灰）改**明暗两档**（`:root` 暗色提亮 #f04848/#4d8df0/#8b98ab/#d5dae2，`:root[data-hl="light"]` 亮色锁原值=零回归；**data-hl 是 CSS 侧唯一明暗钩子**，先例 hljs-theme.css）；状态徽章/网络图灰线/非工作日红字等 **12 处桥接** var(--md-sys-color-*, 原值)；**红线保留 17 处**=导出纸张白底/分部横幅六色板(v4.160 拍板)/旗标紫琥珀/深底 tooltip（补轮廓边框）；**四令牌禁绑 primary/error——主题预设漂移会改领域编码色**；补 sched-pulse reduced-motion；文件 hex 36→25。真机**两态令牌 bridged 实锤**（计算值切换+两态零错误）。②**全站空态统一 V3Empty**（新 components/V3Empty.tsx）——轨道环 SVG（虚线外环+斜轨道+活核，呼应「地核 G」logo）+children 动作透传，image 接受即忽略；替换散用 antd Empty 7 文件 9 处（CharacterPage×4/NovelSettingPage/HomePage/PersonaPicker/SearchModal/SkillModal；modelcenter/CostLibrary 已有各自 EmptyState 不动）；真机目检搜索零结果态协调。③**index.css 全局精修**——`:focus-visible` 全局兜底环（2px var(--gaea-glow)，**antd 自管焦点态控件豁免防双环**）补键盘导航可见性；`::selection` 主题化（color-mix primary 30%）；`:root` 立字号阶梯 6 档 --font-size-xs~xl（新样式勿再写裸 px 字号）。**普查修正**：页面动画已有(.page-enter)/toast 已统一(message 47 文件)/novel hex 多为功能色不动。**走查导航配方**=`window.dispatchEvent(new CustomEvent('navigate',{detail:{page:'schedule'}}))` 直达板块（壳层 NAVIGATE 白名单）；空间键 gaea.shell.space、页签键 gaea.shell.page.*（**首启不恢复页签**，useState 只取 home board——localStorage 预置页签无效）。**测试**=schedule 550 用例+tsc 0+eslint 0 errors+V3Empty 面用例绿；三面截图目检。**门禁**=ci.ps1 全绿/版本三处 4.255.0。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。

- **最新发布：v4.254.0（2026-09-12）「办公·公文规范包补全：生成半边——GB/T 9704 公文模板技能资产化」**——板块并行池既定余项（模板内置，形态随 v4.224 红线拍板走技能资产，零进内核）。gongwen-9704 补「生成」半边：docx_gen.py 一键产出合规空白公文骨架 docx（纯标准库，与取数口同口径）——预置项与排版细则表逐项对齐（页边距 37/35/28/26mm 精确值/红头 FF0000 小标宋 36pt+红色分隔线/文号六角括号〔〕/标题二号居中/正文仿宋三号+28磅固定行距+首行缩进 2 字符/「一、」黑体「（一）」楷体/页脚「— 1 —」=4号宋体+PAGE 域+一字线）；**dogfood 自洽测试 9 例=生成→用自家 docx_layout.extract 复检**（校验器即验收器）+CLI 冒烟三面全对上；SKILL.md 加触发词（生成公文/公文模板）+Workflow 步骤 0+生成参数表。**坑=Windows 下 io.open 默认 cp936 读 UTF-8 磁带会乱码，CHANGELOG 补账时三刀连漏（v4.252/253）——io.open 必须显式 encoding='utf-8'，发布检查单加「CHANGELOG 磁带连续性」项**。**门禁**=ci.ps1 全绿/版本三处 4.254.0。

- **最新发布：v4.253.0（2026-09-12）「二遍扫描账本收官：#9 修 + #6 实测结案」**——①jobs children 级联链随 pruneTerminalLocked 淘汰联动回收（改前只增不清随后台派生累积）。②#6 filewatch 同步 WalkDir **实测结案**：skipDirs 已覆盖 node_modules/.git 等重目录，开发工作区 925 目录仅 **46ms**——异步化收益趋零而「启动窗口事件丢失+无周期重扫兜底」风险真实，风险大于收益维持同步。二遍扫描账本终态=**修 7、结案 1、观察 4（#1/#7/#11/#12 均已知低危或需独立设计）**。**门禁**=ci.ps1 全绿/版本三处 4.253.0。

- **最新发布：v4.252.0（2026-09-12）「启动链收尾：远程剧照迁移异步化（普查二遍#3）」**——冷启动关键路径再摘一颗：改前 Startup 内同步串行下载全部远程剧照（xAI 临时图，每张 30s 超时，N×30s 全算进冷启动）；改后台协程（recover 守卫，沿用 app 刷新协程先例）。**异步化安全前提核实**=下载期间不占库连接（先收集 todo 再逐条下载+短 Exec），与启动期其他角色库操作在单连接池上安全交错；迁移完成前角色卡显示远程 URL=迁移前既有状态语义零变化；幂等（未迁移的下轮重试）+本体测试覆盖。二遍扫描账本更新=修 6 观 6（#6 filewatch WalkDir 就绪语义维持观察，#1 browser 状态机维持观察）。**测试**=characterlib 迁移本体+app 全量绿。**门禁**=ci.ps1 全绿/版本三处 4.252.0。

- **最新发布：v4.251.0（2026-09-12）「后端第二遍扫描：goroutine/内存/启动链（修5观7）」**——换镜头复查首遍欠覆盖维度。**goroutine 泄漏零**；12 项净修 5：①MCP StartAvailable 并行化+单 server 30s 上限（改前串行无 deadline 挂死 server 无限期卡装配；**AfterFunc 取消仅失败路径生效——成功连接 transport 绑定该 ctx 定时器必须 Stop 否则误杀**；测试=挂死子进程(**坑=select{} 触发死锁检测器自己崩报 EOF,须 sleep 循环保活**)+双挂死好 server 耗时上界 3.5s 守卫并行性）②weixin apiPost 每请求 ctx 超时替代克隆 Transport（长轮询永久每轮新建+TLS 握手）③notifyStart 异步化（Startup 链 10s×N）④tasks 节流 map 随 LRU 回收⑤copyPath 流式拷贝（迁移内存峰值=最大文件→io.Copy）。观察 7 项入普查档二遍扫描节（#1/#3/#6 触及启动语义需单独设计）。**门禁**=ci.ps1 全绿/版本三处 4.251.0。

- **最新发布：v4.250.0（2026-09-12）「走查遗留收账：语义召回搜索路径加界 + 观察项②撤销」**——刀F 处理 v4.249.1 入池两项观察：①语义召回搜索路径加界——tool/app 两侧 semanticCostRecall 的 st.Search 内含 Ensure 增量向量化，ctx 60s 包住「追赶+查询」（1509 条库首次追赶 >120s，真机实测 60.1s×2）；**实态收窄=Ensure 按批持久化增量收敛，60s×2 属追赶窗口非常态**；刀法=ctx 60s→5s 双侧加界，超时回落关键词结果（语义召回只增不改基线，与改前 ctx 超时语义一致快 12×），追赶由既有显式补齐 GaeaSemanticIndexBackfill 完成；**导入后台钩子提案与档头既有拍板「不做后台守护协程」冲突已撤回**。②~~Available() 探测口径 /models 404~~**撤销**——引擎实盘 base_url 本就含 /v1（connected=true、bge-m3 在册），探测路径正确；走查时 curl 打了缺 /v1 的猜测 URL——**证据未复核即入池，同 #8 教训**。零绑定零前端。**门禁**=ci.ps1 全绿/版本三处 4.250.0。**「优化 gaea 的后端」线程全收**。

- **最新发布：v4.249.1（2026-09-12）「走查补刀：app 侧检索客户端单例化 + 五刀真机走查清账」**——后端性能五刀真机走查（CDP 9333 DOM+绑定断言，配方 .tmp/walk-v4249.mjs：**壳内绑定挂 window.go.app 门面（OfficeB/CostB/MemoryB…）非 window.go.main.App，跨门面解析+Wails 逐参显式传**）——绑定全通（看板绑定亚毫秒=条目缓存真机生效；GaeaCostList 真实库 1509 条 62ms）、记忆中枢/造价数据库零接管、exceptionThrown=0。**新发现一并修**（普查#4 的 app 侧孪生）：localSearchEmbedder/localSearchReranker 每调用 new 新客户端改按 (baseURL,model) 键单例。**观察池新增**=①GaeaCostSearch 语义召回在 UI 绑定内同步 Ensure 全库向量化（60s ctx，关键词召回<3 时可阻塞 60s/次，实测 60.1s×2）②Available() 探测口径 /models 404（herdsman 实际部署不实现）→ Available 恒 false——两项按需另刀。**门禁**=ci.ps1 全绿/版本三处 4.249.1。

- **最新发布：v4.249.0（2026-09-12）「后端性能刀E：面板/杂项收官（看板缓存 408×+预览缓存+无界缓存上限）」**——普查档最后一刀，**五刀 A/B/C/D/E 全清收官**。①**会话条目缓存**（普查#5）：看板/节点详情/轨迹/Agent 网络（+resync 共五处）绑定共用 ReadEntriesFor 全量读+逐行解析，前端轮询每回合同份日志反复解析（基线 2MB 日志 9.1ms/6.7MB 每刷）；新 readEntriesForCached——事件日志 **append-only** ⇒ size+mtime 未变即内容未变，缓存解析结果（上限 8 会话；fold/detail/trajectory 层核实只读共享；resync 保持直读）。**看板刷新 9.1ms→22μs（408×）**。②**预览缓存+轻量解码**（普查#7）：列表点换 previewSessionCached（size+mtime 键上限 64）+解码只取 role、content 仅首条用户消息才反序列化。③wssearch textCache 加 64 条上限（普查#10 无界增长）。④graph linker 循环内正则改字符串操作（QuoteMeta 字面量等价，普查#16）。⑤sanitizeFilename 改 strings.Builder（#18）。⑥三处函数内 MustCompile 提包级（#20）。⑦SemanticRecall 死代码删除（#19 零消费方核实；DocText/Cosine 保留）；#17=legacy file backend 观察不修。**测试**=Go+2（两级缓存失效语义）+app bench 1 组。**门禁**=ci.ps1 全绿/版本三处 4.249.0。

- **最新发布：v4.248.0（2026-09-12）「后端性能刀C：SQLite 连接池放宽 + 批量清理事务化」**——普查档刀C（风险最高单独走）。①**连接池 1→4**（普查#6）：Hephaestus.db 改前 MaxOpenConns(1) 把读也串行化（WAL 本可读写并行），且「rows 未关闭时 Exec 等唯一连接」是整类死锁根源（portrait.go/whisper fts.go 两处在册注脚）；放宽后死锁类整体消除（注脚注释更新、收集再写回习惯保留）；写并发由单写者锁+DSN 逐连接 busy_timeout=5000 兜底（modernc 对每新连接应用 DSN 参数，PRAGMA 不丢）。**Benchmark**（2s 时间制）：并行点查 7.6μs→2.5μs（**3.0×**）。②**批量清理事务化**（普查#11）：CleanupArchived 单事务并收紧原子语义（改前删一半失败仍把整批 doomed 报审计=虚报，现在全删或全不删）；semantic.Stale 同款。**测试**=Go+2（并发写 8×25 全落库零 BUSY；写事务进行中另一连接可读+WAL 快照隔离——改前单连接下此测试死锁=新形态守卫）+db bench 2 组。**门禁**=ci.ps1 全绿/版本三处 4.248.0。**普查档余**=刀E 面板杂项，待拍板。

- **最新发布：v4.247.0（2026-09-12）「后端性能刀D：cost 检索固定成本（BM25 缓存+客户端单例+子查询消除）」**——普查档刀D，benchmark 先行。①**BM25 排序缓存**（普查#3，T7-3 原设计接通）：Search 每查询对命中子集从零重建倒排（2000 条库 20.6ms/22MB）；**刀法修正**=bm25.Cache 原设计语料整库与现行为子集排序语义错配，改**包级缓存**（key=db池+数据版本+过滤形态，语料=该形态 SQL 全捞全量 name 序；Store 即建即弃不能挂实例），子集按语料下标取分 tie-break=name 序与改前一致，**失效全链**=Save/Delete/分类+SelfHeal 实际修复推进版本+app 批量导入 cost.InvalidateRankers 显式失效，map 超 16 清空。②**Embedder/Reranker 单例化**（普查#4）：按 retrievalRuntime 配置键缓存实例（探测缓存从此可命中、keep-alive 复用；SetRetrievalRuntime 变更自动重建，构造失败不缓存）。③**子查询消除**：逐行相关 COUNT 改 LEFT JOIN (GROUP BY) 单趟，ComponentCount 语义不变。**Benchmark**（20x）：Search 500 条 2.4×/2000 条 2.1×（分配 22→10MB）。**测试**=Go+1（失效语义三态）+bench 2 组；TestCostSearchBM25Order 原样过=排序口径守卫。**门禁**=ci.ps1 全绿/版本三处 4.247.0。**剩余刀路**=刀C SQLite 读写分离/刀E 面板杂项，待拍板。

- **最新发布：v4.246.0（2026-09-12）「后端性能刀B：记忆写路径（重载出锁+写路径事务化）」**——「优化 gaea 的后端」刀B（普查档 20 项交拍板池，benchmark 先行）。①**全库重载出全局锁**（普查#2）：九个记忆写点改前在 c.mu 内跑 memory.Load（50 条=2.3ms/300 条=7.0ms）阻塞 Send/Cancel 等全部控制器操作；**刀法修正**（非原案后台刷新）=刷新对调用方仍同步（写后立即可见语义与测试依赖不动），Load 改锁外跑——beginMemoryReloadLocked（锁内收口 Options+推进代号 memGen）→解锁→finishMemoryReload（代号守卫：gen≥已换入才换入，慢旧 Load 被拒不回退快照，被跳过的写必然已被更高代 Load 读到）；applySpace 推进代号使在途旧空间快照失效。②**写路径事务化**（普查#12）：五写方法（Save/Archive/Unarchive/ChangeType/touchInSpace，Touch 在引用解析热路径）commitWithEvent 单事务，事件尽力而为失败回退单语句重放（写入本体不丢）；EventLog 抽 execer（事件必须落事务连接——Hephaestus.db 单连接池否则死锁）。**Benchmark**（20x）：Save 0.95→0.49ms/Touch 0.86→0.45ms（均 1.9×）。**测试**=Go+5（事件失败回落保本体/原子提交/未命中零留痕/并发收敛 4×5 全到达）+bench 3 组。**复核撤销普查#8**（BuildSearchIndex 实际分词 body，勘误在档）。**门禁**=ci.ps1 全绿/版本三处 4.246.0。**下一刀候选**=刀C SQLite 读写分离/刀D cost 检索/刀E 面板杂项，待拍板。

- **最新发布：v4.245.0（2026-09-12）「后端性能刀A：回合边界税清偿（事件日志续接+摘要增量化）」**——「优化 gaea 的后端」首批落地（普查档 docs/gaea-backend-perf-survey-2026-09.md 20 项交拍板池，本刀=刀A 两项，先立 benchmark 锁基线再动刀）。①**事件日志续接**（普查#1）：sink turn_done 干净关闭后记「关闭时 seq+刷盘后文件大小」，重开走新 OpenLogResuming——大小未变即跳过 RepairLogFile+countLogLines 全量读+逐行解析直接续 seq；大小不符（外部追加/轮转/截断/删除）自动回落 OpenLog 全量路径，回合间日志可被外部触碰的设计红线由大小核对保住；OpenLog 本体与 save/migrate/subagent_promote 三处一次性调用方零变化。②**摘要增量化**（普查#14）：stream() 每步 DigestMessages 全量重哈希改 DigestCache——相邻请求前缀**指针级身份**核对（字符串不可变⇒数据指针+长度相等即字节相等；ToolCalls 同底层数组），未变消息复用摘要只重哈希新增/改写，身份不符一律重算（前缀稳定证据链只会重算不会错判），Digest 与 DigestMessages 逐元素等价。**Benchmark**（20x）：日志重开 2MB 19.1ms→0.96ms（20×，与大小无关=O(1)）；摘要步进 300 回合 521μs→49μs（10.6×）。**测试**=Go+9（续接/外部追加回落/截断/删除/sink 跨回合/外部写回落；缓存等价六形态/复用计数/重建切片保守重哈希）+bench 3 组（gaea 热路径首批 benchmark）。零绑定零前端。**门禁**=ci.ps1 全绿/版本三处 4.245.0。**下一刀候选**=刀B 记忆写路径/刀D cost 检索，待拍板。




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
