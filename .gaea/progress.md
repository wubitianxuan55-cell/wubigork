# 任务进度

> 本文件为**最近发布速览**。完整历史磁带见 `docs/archive/progress-history-2026-09.md`
> 与 `releases/`。

# 任务进度

> 本文件为**最近发布速览**。完整历史磁带见 `docs/archive/progress-history-2026-09.md`
> 与 `releases/`。

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
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.270.0。走查=部分完成如实记录（live 模型调工具留池补验，详见 releases/v4.270.0.md）；产物=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）。

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
- **文档**：docs/gaea-sin-board-design-2026-09.md §12.5、releases/v4.261.1.md。

## 最新发布：v4.259.0（2026-09-12）「原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）」

- **链路**：角色库 → 故事角色（v4.257）→ 插图人物（本刀）：文本锚点对所有后端生效；图像参考槽走绘梦 T2（ComfyUI krea2*/z-image-turbo · Herdsman 图生图，denoise 0.65）。
- **防互串**：提示词点名优先；单角色兜底；多角色未点名不锚定。
- **诚实**：能力门控（不支持就跳过并给原因，不硬塞）+ 参考槽失败自动退回纯文本重试（标 ref_fallback）+ caption 徽标可见。
- **未做**：ipadapter/pulid（后端未实现）。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.259.0；走查徽标实测（零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §12、releases/v4.259.0.md。

## 最新发布：v4.258.0（2026-09-12）「原罪插图：生成进度动画 + 串行队列 + 真停止」

- **进度**：抽共享进度组件 GenerationProgress（自绘梦 GenerationBar 抽出，类名/阶段名表单点维护）+ 轮询 hook useComfyTaskProgress；ComfyUI 走确定态（百分比+当前节点+已用时），其余后端走不定态光带 + 本地秒表（不编造百分比）。
- **串行队列**：同板块一次一张（图像后端单任务），排队显示「前面还有 N 张」，失败不阻断。
- **真停止**：新绑定 SinCancel（绑定 639→640）中止底层流并保留已生成部分落库（extra.cancelled）；插图「取消」接 CancelImageGeneration。
- **顺带修真缺陷**：sending 因 deadRef+StrictMode 卡死（输入框锁死）、Composer「停止」是空实现。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.258.0；走查：两处进度条 + 取消一张另一张继续 + 完成后停止按钮消失（零错误零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §11、releases/v4.258.0.md。

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
- **文档**：docs/gaea-sin-board-design-2026-09.md §8/§9、releases/v4.257.0.md。

## 最新发布：v4.256.0（2026-09-12）「闲庭新板块『原罪』：对话式图文混杂故事创作（复用办公组件）」

- **定位**：用户点单的闲庭新板块（原话见 CHANGELOG）；闲庭第七个一级板块，形态=「说一句→写一段→你接着指方向」的对话式故事创作 + 就地出图。
- **落地**：前端复用办公组件（Composer/Markdown/ToolbarButton/EmptyState/三件套样式，零新建聊天组件、零改办公件）；后端新门面 SinB（9 方法）=统一聊天存储 mode=sin（与聊天板块双向隔离）+功能级路由 sin（模型中心可单挂模型）+绘梦图像链内联插图+图文 Markdown 导出。
- **协议**：模型另起一行输出 `@@插图|画面描述@@`，前端按出现次序（0 起）就地生成并嵌进正文流；正文原样落库，重开按 `extra.illustrations` 渲染不重生成。
- **边界**：成人内容口径沿用 docs/ADULT_MODE.md（零新立），四条硬边界写进提示词并有测试锁定。
- **门禁**：Go 全绿 / drift OK@637 / tsc 0 / eslint 0 error / vite build 成功 / vitest 334 文件全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.256.0。
- **文档**：docs/gaea-sin-board-design-2026-09.md（设计基线+复用清单+已知边界）、releases/v4.256.0.md。

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

- **定位**：阶段五「记忆OS+上下文编译」首刀（规划大纲 §2 主轴旗，底座#1）。设计基线=docs/gaea-memory-graph-51-design-2026-09.md。
- **落地**：memory_events 事件日志（SchemaV18 追加式，五条写路径挂钩，Touch 逐条留痕）→ ProjectEvents 纯函数投影（实体/事件/来源三向边，[MEM:name] 图寻址，悬空拒写留 Dangling）→ mem_graph_* 物化（embedding 占位列；读前水位对账懒重建=删库可重建）。
- **接线**：绑定 608→609（GaeaMemorySemanticGraph）；MemoryHub 图谱页「关联图/事件图谱」切换；回合收尾引用解析落 cite 事件（命中+悬空）。
- **门禁**：Go +9 用例 / vitest +1 / tsc 0 / drift@609 / 版本三处 4.210.0。
- **欠账**：5.1 余项=真·发送前悬空引用剥离（流式层另刀）；UI 遍历/手动重建按钮按需另刀（Go API GraphNeighbors/RebuildGraph 已备）。

## 最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」

- **造价刀路池 §6 收官，survey 六项缺口全清**——docs/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态（§3→v4.194 / §4→v4.195 / §5→v4.196 / §6→本刀），该文档不再充当欠账池。
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
- **未动（体量大且属用户数据，等拍板）**：`clones/` 参考仓（mpp2013/projectlibre/unsloth，**338.9 MB**——含隐藏 .git；首轮统计未计隐藏文件故报 212 MB）、`backups/` 120.4 MB、`whisper_data/` 运行时数据 49.5 MB、`.tmp/codex` 67 MB。

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

- **造价域缺口 §1 首刀**（docs/gaea-cost-domain-survey-2026-09.md）：条目匹配全靠标题字符串 → 编码体系贯通。
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
