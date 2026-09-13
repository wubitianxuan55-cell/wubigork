# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。
- **最新发布：v4.278.0（2026-09-13）「小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）」**——两条线汇合收口：①并行池小说线续刀=**伏笔一致性 Linter**（plan §4 并行池小说行后半：文风指纹 v4.277.0 已落）；②另起的 MuMuAINovel 蒸馏实施线（规格 `docs/distill/`）先落 t1 共享契约后**停摆**，用户确认后由 Codex 侧接管，把在制品 + t1 契约一并落库。**落地**=①伏笔一致性体检（`internal/app/novel_foreshadow_lint_handler.go`：ordering/status-mismatch（partial 豁免）/dangling/stale≥10 章（长线豁免）/duplicate 五类确定性检查 + 报告 totalChapters/items/planted/hinted/revealed/longTerm/findings，空登记=空报告正常态；`ForeshadowPanel`「一致性体检」概要行+severity Tag 轻/中/重直显；绑定 645→646）②t1 契约（`internal/types/`：伏笔并集 6 态+计划回收章+来源/评分/注入控制+urgency **显式 must_resolve**（MuMu 静默丢弃字段的教训）；角色三态章节水位守卫+方案 C 职业引用+派生 member_count+亲密/delta_hint 钳制；9 维分析+三维分档评分+建议数硬联动；ChapterPlan/RewriteVersion/Annotation；模板解析-覆盖-渲染-校验与上下文组装接口+Lint 接口+ChapterNumOf；旧 JSON 零迁移）③**P0 口径纠偏**=在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降读取别名，与 handoff §2-3/§4-7 相反，会让统计回收率/lint 计数/章节注入/SaveForeshadows 白名单等 7 类消费方静默读空）→ 纠回 `revealed` 写入口径 + `IsResolvedStatus` 覆盖两套 wire 值并接进 stats/lint/章节注入/analysis + SaveForeshadows 白名单 3 态→并集 6 态（别名先归一化）④`docs/distill/` 六域规格入库并登记 docs/README。**测试**=Go 全量绿（types 兼容矩阵+两套 wire 值新用例/app 伏笔 Lint/stats/analysis）+vitest 体检 3 例+spaceBindings 锁 467→468。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；产物=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**欠账**=**六域实施 t2~t7 未开工**（拆书导入/长程一致性/情节分析重写/角色职业接线/提示词工坊/前端统一接线；规格在库、契约就位）；t1 契约除伏笔体检外暂无消费方（契约先行）；观察项=ChapterNumOf 双实现/stats·narrative 无 owner/`docs/mumu-distill/` 重复副本待删。
- **最新发布：v4.277.0（2026-09-13）「小说·文风指纹落地面：参考档构建 + 对照体检」**——「优化完善gaea」从板块并行池挑小说线既定余项（novel-revolution 刀3 内核指纹算子齐备但 app 层零消费）。**落地**=①参考档 fingerprint.json（Manager 三方法原子写；types 不反向依赖 novelstyle）②NovelFingerprintBuild 全部已写章节（v4 走 ReadChapterAsStitch）ComputeFingerprint 落盘，**门槛诚实拒绝**（≥3 章且 ≥3000 字不足中文报错不落盘）③NovelFingerprintScore 有参考档 ScoreText+Delta（越小越像作者）/无参考档通用阈值两路可用，参考档损坏降级不挡体检④NovelFingerprintStatus 未构建=exists:false 正常态⑤CreatePage rail「文风指纹」+StyleFingerprintPanel 受控 Modal（空态引导/8 项摘要格子+口头禅 Tag/体检大数字+Δ 基线距离+issues 摘录列表）。v1 口径=作者成稿即样本，「只学作者手改」留后续刀。**测试**=Go+6+形状锁（camelCase 键断言+PascalCase 反例+delta=0 不被 omitempty 吞）+vitest+5+spaceBindings 锁 464→467；绑定 642→645。**真机走查**=夹具工程壳内 CDP 全流程（空态→构建 3 章 · 3,999 字→体检 Δ 0.00→重建覆盖→console 零错误）；坑复训=海报墙入口须 scrollIntoView 再点/无 outline 夹具致体检钮 disabled（UI 依赖大纲树须连 outline.json 一起造）/delete-pending 须 PowerShell 复删。**门禁**=ci.ps1 run1 抓 v4.276 存量缺陷当场修（ChatPage.test 语音断言未随 camelCase 发送对齐）run2 全绿/drift OK@645/版本三处 4.277.0；产物=exe 49,352,704B SHA256=2CB99312…802CD（桌面副本同哈希，冒烟 200 过）。
- **最新发布：v4.276.0（2026-09-13）「绑定面 JSON 形状普查：造价参考/复盘笔记/会话组价价格带断链修复」**——「优化完善gaea」：把 v4.269 走查抓到的「Go struct 缺 json 标签→Wails 线上 PascalCase→前端 camelCase 读空」bug 类从撞见修一处升级为**全仓 AST 闭包普查**（绑定面 App 导出方法签名类型递归展开×缺标签 struct 求交）：全仓 327 缺标签 struct 中绑定面可达仅 7，三真断链当场修=cost.PriceBand/BandSource（GaeaCostCompose band 字段，**v4.191 起会话组价价格带卡全空+证据表 IQR 离群判定 NaN 失效**）/costref.Note（复盘笔记 validUntil/refCount/updatedAt 读空）/costref.Indicator（造价参考视图标题/均值/分位读空）；四输入方向（AskAnswer/ChatMessageInput/TTSParams）防御补齐——**形状锁定唯一真源落 Go 定义，不靠 Unmarshal 大小写宽容**。连带抓出 useChatVoice.ts 三处 PascalCase 发送点（线上不炸纯靠 Unmarshal 大小写不敏，wails build 再生绑定类型时 tsc 实锤）改 camelCase。**测试**=Go+6 形状锁（marshal 键断言+PascalCase 反例+旧数据 Unmarshal 兼容）+**结构性守卫 TestWireShapeGuard**（普查逻辑测试化进 ci：绑定面可达闭包内缺标签即 FAIL，豁免清单机制；**负向验证过**——删标签 FAIL 精确报出/恢复 PASS；v4.269 修 7 结构体没防住本轮同款，守卫是根治）。**真机走查**=CDP 桥验 GaeaCostNoteList 返回 camelCase（validUntil/projectType/updatedAt 在位）+UI 断言复盘笔记卡片全字段渲染（「材料·房建·泵送」/「有效期至 2026-12-31」/「引用 0 次」）+console 零错误+清场 remaining=0；坑=整段式走查脚本 UI 段卡死一次，拆探针分段+看门狗全链过——**走查脚本宜分段不要一次性长链**。**门禁**=ci.ps1 全绿（run1 负载 flaky 两例：schedule rename 文件锁+netclient httptest 竞态，隔离复跑绿既有先例；run2 全绿）/drift OK@642/版本三处 4.276.0；产物=exe 49,294,336B SHA256=1A12B1CA…DA869（桌面副本同哈希，冒烟 200 过）。
- **最新发布：v4.264.0（2026-09-12）「原罪右栏面板宽度可拖拽」**——用户口径「右侧面板宽度应该可以自由拉伸」（v4.263 下刀候选清账）。**落地**=①左缘拖拽手柄（骑边框 8px 命中带，悬停/拖拽 accent 显色，role=separator）：指针拖拽与办公 useWorkspaceLayout 同源（window 级 pointermove 实时跟手/向左拖=变宽/body 锁 col-resize+禁选中/pointerup 持久化/pointercancel 兜底）②宽度记忆 gaea.sin.panelWidth 自有键续用③钳制=240~640 硬档+视口收敛（innerWidth-520 保故事架+故事流可读），clampSinPanelWidth 纯函数，非法值回 268④双击手柄复位⑤画廊列数 auto-fill minmax(104px,1fr) 随宽自适应（拖宽多列有实际收益）。**测试**=vitest +5（跟手+钳制+持久化/收窄下限/双击复位/记忆恢复/clamp 矩阵）；零 Go 零绑定。**真机走查**=CDP 受信任鼠标两轮全通：拖拽 268→448px 实时跟手（中途采样 358）+落盘 448+画廊 2→3 列+双击复位 268/268，零前端错误（首轮脚本双击模拟缺 clickCount:2 误报失败，修正后过=应用无缺陷）。**门禁**=ci.ps1 全绿（代码树 run3+终态树 run4）/版本三处 4.264.0/drift OK@641；产物=exe 49,255,424B SHA256=4A1E96E0…0102F（桌面副本同哈希，冒烟 200 过）。
- **最新发布：v4.263.0（2026-09-12）「原罪右栏创作面板：角色/大纲/设定/插图」**——用户口径「增加类似办公的右侧面板」。**改前**=右栏是静态说明栏，且 sin_notes/sin_outline 写进 notes/<故事id>.json 的工作底稿前端**没有读取通道**。**落地**=①新绑定 SinNotesGet（640→641，play）：只读便签文件返回 {notes,outline}（topicGuard+与工具侧同一把 sinNotesMu；缺失/损坏=空，同口径）。②右栏改办公同款标签页面板 SinSidePanel：角色（SinCastPanel 原样）/大纲（只读 pre-wrap）/设定（逐条 #序号）/插图（全故事画廊：extra.illustrations 收集+正文反解 caption+Modal 大图，AttachmentDataURL 通道）；页签计数徽标（**纯文字页签=真机走查实证 268px 下图标+两字折行，去图标+nowrap 修**）。③玩法/插图协议/边界收进头部问号气泡（内容逐字保留）。④开合=顶栏 PanelRight 钮，记忆键 gaea.sin.panelOpen/panelTab（自有命名空间），宽窗默认开窄窗默认收，替代 1180px 强制隐藏。⑤每回合结束自动重读底稿+切故事重读。**测试**=Go+1+vitest+14；**门禁**=ci.ps1 全绿 ×2/drift OK@641/spaceBindings 锁 462→463/版本三处 4.263.0；**真机走查**=用户真实故事四页签全通（空态轨道环诚实/3 张缩略图全转 data URL/Modal 预览/开合记忆），零前端错误；exe 49,253,888B SHA256=6E281D6A…0A880（桌面副本同源，冒烟 200）。
- **最新发布：v4.262.0（2026-09-12）「原罪工具集 v1 + 过程卡」**——用户问「原罪能用哪些工具？看不见调用工具的过程卡」，拍板「A + 网络搜索工具」。**改前**=原罪模型**零工具**（单轮 chat 流，请求无 tools 字段）⇒ 过程卡无从显示。**落地**=①工具集 5 个：`web_search`/`web_fetch`（委托办公 builtin，**schema 转发零漂移**；[search] 扇出/SSRF/代理同源）+ `sin_cast`（只读角色卡）+ `sin_notes`（便签）+ `sin_outline`（大纲）；注册表 `sin_tools.go`（init 自注册/重复 panic/顺序固定）。②**有界工具循环**：新 ai 入口 `ChatStreamMessages`（与 `ChatStreamChunks` 共用 `prepareStreamRequest`）；**调用预算**（同名 ≤3/整轮 ≤8，超限不执行、如实回绝）；**收尾轮**（不带 tools + 「工具阶段结束」收束令，被工具顶掉时允许一次兜底收尾轮）；首轮不支持 tools ⇒ 去工具重试 + notice；SinCancel 起手即登记。③**过程卡**=新事件 `tool_dispatch`/`tool_result`/`notice` + `done.tools` 与 `extra.tools` 同形态（重开还原）+ 前端 `SinProcessCard`（思考折叠 + 工具行 + 展开明细）；静默超时改**每帧重置**（改前一次性 90s，工具长回合被误判超时）。④便签/大纲落 `%APPDATA%\gaea\sin\notes\`（原子写/id 白名单/损坏当空），不碰办公数据面。**未做**=`sin_illustrate`/`sin_export`（越契约实现、产物回写链路未接完，挪 `.tmp/sin-next-knife/` 下刀接）。**教训**=并发子代理必须显式禁改契约文件；产物类工具的跨线依赖要在契约里点名责任人。**真机走查**=联网探针 0.6s 返回 Bing 结果；走查故事一轮出正文 1185 字（落库 1340 字）+ 9 条工具轨迹（含预算回绝），零渲染错误（走查故事已删）。**测试**=Go+14/vitest+13；**门禁**=ci.ps1 全绿/版本三处 4.262.0/零新绑定；exe 49,242,624B SHA256=470F86EF…C6E68（桌面副本同哈希，冒烟 200）。
- **最新发布：v4.270.0（2026-09-13）「sin_illustrate 工具入册（拍板放行）」**——维持轨五算 e2e 清账后向用户列拍板池，连续「继续」=按序放行第一项，推翻 §13.2「刻意不做」（入册前提=产物链本刀补齐）。**单链路委托既有 SinIllustrate**，不复制出图链。**落地**=①工具集 6→7（outline 后 export 前）②`sin_illustrate`（prompt 必填/caption/size 可选，ReadOnly=false，Description 明确与标记协议分工+「不必再为它写插图标记」防同图两生成）③**Artifacts 产物链**（v4.262 留存缺口）：sinToolTrace 增 artifacts 字段+sinToolArtifactProvider 接口+循环收集随 result 帧与轨迹落库 ④落库回写 sinPersistToolArtifacts 按 tool0..toolN 并入 extra.illustrations（cue 前缀防互踩，逐条告警不阻断）⑤过程卡缩略图（附件通道 data URL+caption 图注+失败占位）+元数据「生成插图」。**测试**=Go+4（真出图+Artifacts+目录/坏参数/order/回写+messageID<=0 跳过）+计数锁 6→7+wantReadOnly+vitest +3；零新绑定。**门禁**=ci.ps1 全绿/版本三处 4.270.0/drift OK@642；产物=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）；**live 模型调工具端到端留池补验**（两次尝试环境未走到+遇疑似用户活动按红线停止，脚本已含空间切换修正），见 releases/v4.270.0.md。
- **最新发布：v4.274.0（2026-09-13）「UX 线第六刀：自定义强调色亮态自动深化（对比度保障）」**——对比度普查观察池第一项挖到底：**非主题令牌缺陷**（6 主题 lightFn glow 全是深化值），是**用户自定义强调色（gaea-accent）不分明暗覆盖 glow/colorPrimary**——暗色调亮的 #1dd7bf 切亮态压浅底对比仅 ~1.5，全站 accent 文字（空间 chip/active tab/徽标/链接）看不清。**修复**=①`ensureLightContrast` 纯函数（lib/accent.ts：对白底 WCAG<4.5 时保色相饱和度迭代压 HSL 亮度，下限 0.12、灰兜底；达标/非法原样返回；vitest 7 例含真实案例）②App.tsx effTokens 亮态深化暗态原样（用户存储色不变）。**真机复验**=复跑 13 页×两态：light 告警 63→53 accent 类清零（亮态 glow 实测 #128475=深化输出），dark 15 不变；剩余为禁用态/图片背景误报/占位符/设计弱化。**观察池**=weixin「运行中」antd success 绿字 2.21（语义色惯例不动）/schedule 行号与 placeholder（设计弱化）。**门禁**=ci.ps1 全绿/版本三处 4.274.0/drift OK@642；产物见 SHA256SUMS-v4.274.0.txt。
- **最新发布：v4.275.0（2026-09-13）「价格带数据源调研结案：信息价『除税价』列适配（拍板池候选4 收官）」**——调研（docs/gaea-priceband-datasource-research-2026-09.md）：价格带数据面**基础设施已完整在产**（CostEntry Source/Region/PriceType/PriceDate 四要素+ComputePriceBand+文件/AI/视觉导入链+OCR 询价飞轮），「手动导入信息价」无需开发；外部自动接入四路评估=B 抓取不建议（脆弱+合规灰）/C 商业 API 不做（询比价已拍板不做其上游）/D LLM 查价不作基线——**建议候选4 结案**。**真机验证抓实锤当场修**：信息价样本 CSV 走 GaeaCostImportPreview（预览零落库）——「除税价（元）」不在 fieldPrice 字典→价格列 unmapped、12 行全 skip；当场修字典增「除税价/含税价」+Go 回归测试（TestParseCSV_InfoPriceChushuiColumn）；复测 **12/12 全识别**。**门禁**=ci.ps1 全绿/版本三处 4.275.0/drift OK@642；产物见 SHA256SUMS-v4.275.0.txt。
- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。
- **对比度普查收账（2026-09-13，非版本刀，零代码）**：13 页×明暗两态 WCAG 程序化扫描（.tmp/walk-v4274-contrast.mjs）——dark 15/light 63 告警逐类甄别后**大头为脚本误报**（渐变/图片背景无法合成，目检实际清晰）；真实低对比仅亮态 accent 弱化文本（~10 处 1.5~1.8）+schedule 行号 1.3+weixin purple tag 3.39——全为装饰性文本非正文，**无 AA 硬伤不动**。观察池新增=亮态 accent 弱化文本打磨候选（修则需动 lightFn 令牌，视觉拍板项）。**坑=对比度自动扫描对 background-image/渐变必然误报，告警须逐类目检甄别后才能定刀**。
- **v4.270 留池补验收官（2026-09-13，非版本刀，零代码）**：sin_illustrate live 端到端**全通**——模型真调工具/图片真实落盘（sin/art「雨夜回眸」.png 1.8MB）/轨迹 artifacts 在位/`extra.illustrations['tool0']` 回写画廊可见/过程卡「思考过程·319 字·生成插图」元数据在位。**历史两次失败归因翻案**=上游 grok-4.6 一次性空返回（仅 reasoning 无正文无工具），后端兜底如实报错（sin_handler.go 空正文不落库），重试即成——**非 gaea 缺陷**。清场=故事删+产物图删（壳句柄锁删挂起，杀壳后 PowerShell 删成）。**坑**=①node rmSync 对被占用文件不抛错但删不动（Windows delete-pending），删后必须 existsSync 复核 ②走查脚本清场要放 finally（v4.270 脚本超时 throw 路径跳过清场）。**观察池新增**=sin/art 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34「雨夜站台」走查 prompt 产物），待人工确认删除。
- **最新发布：v4.273.0（2026-09-13）「UX 线第五刀：键盘焦点环全站恢复（可访问性修复）」**——「继续」续 UX 线。普查 reduced-motion=基建已齐（index.css:457 系统级 + :772 应用内 ui-reduced-motion 双全局兜底）不动。CDP `Input.dispatchKeyEvent` Tab 遍历六页逐站采样抓到大问题：**v4.255 全局 :focus-visible 环全站性失效**（home 7 站/cost 5 站/sin 6 站无环）——元素 `matches(':focus-visible')===true` 但 computed `outline:3px none`，枚举 styleSheets 实锤打包产物 tailwind-*.css 有条**源码不存在的** `:focus-visible{outline:none}`（Tailwind v4.3.3 编译链路注入，源码 grep 不到），同特异性按文档序吃掉普通规则。**修复（index.css 单文件）**=①全局环 `!important` 必胜（forced-colors 同款标准做法）②antd 输入类豁免段同步 !important（自管 border/shadow 焦点态防双环）③**ant-btn 移出豁免**（Button text/default 聚焦零视觉变化，豁免=键盘用户找不到焦点）。**真机复验**=六页 Tab 遍历四页零无环，余 2 站为有意豁免 ant-input（probe 60ms 采样早于 antd shadow 0.2s transition，误报）；antd 按钮聚焦实拍 2px glow 环贴合圆角。**观察池**=动效时长令牌化不做（裸 ms/s ~95 处 vs --dur 消费 65 处，节奏差异属设计自由度）。**门禁**=ci.ps1 全绿/版本三处 4.273.0/drift OK@642；产物见 SHA256SUMS-v4.273.0.txt。
- **最新发布：v4.272.0（2026-09-13）「UX 线第四刀：窄窗响应式收口（遥测条 + Composer 工具行）」**——「继续」续 UX 线。换镜头做窄窗：CDP `Emulation.setDeviceMetricsOverride` **900×600 复跑 13 rail 页**——布局自适应面整体健康（办公右栏窄窗自动收起、面板 min-w-0/truncate 到位），抓到两处真硬伤并修：①**底部遥测条右缘截断**（home/chat/novel/imagegen/cost/sin 六页均现）：引擎 pod（nowrap 超长模型名）把 CPU/内存/GPU 数值挤出视口——`.v3-engine-pod` 加 max-width 240+min-width 0+shrink 1，**pod 文本包 `.v3-pod-text` 出省略号（inline-flex 容器上 text-overflow 不生效，必须内层 span truncate）**；tele-key/value 加 flex-shrink:0——挤压优先级 pod→工程名（已有 ellipsis）→遥测数值完整保。②**sin composer 工具行按钮竖排折字**（右栏打开时容器约 320px，权限/思考按钮无 nowrap 被 flex 压成竖条）：ComposerToolbar 外层 `flex-wrap`+按钮组 `shrink-0`+按钮 `whitespace-nowrap`（**办公共享组件，办公+原罪两处受益**）。**测试**=tsc 0/eslint 0/composer 26 用例+layouts 5 用例绿。**真机复验**=重建 exe 复跑 13 页窄窗**全 ok**（修复前六页 off:v3-tele-value），目检 pod 省略号+遥测完整、工具行两行横排无竖字。**坑=后台命令 cwd 漂移致 build.bat「不是内部或外部命令」/sync-version 静默没跑——发版前 grep 版本三处必须显式验证**。**观察池**=办公右栏中窄窗任务面板横滚兜底（内部已 truncate 非破版）/weixin 详情留白沿用待拍板。**门禁**=ci.ps1 全绿/版本三处 4.272.0/drift OK@642；产物见 SHA256SUMS-v4.272.0.txt。
- **最新发布：v4.271.0（2026-09-13）「UX 线第三刀：滚动条全站单一真源（视觉降噪）」**——用户「继续优化迭代前端 UI」接 v4.255/v4.260。普查=CDP 起壳 **26 页次（13 rail 页×明暗两态，code=独立窗口跳过）**横向溢出零/越界仅首页极光装饰球/console 零错误，结构层无硬伤；视觉层抓到**四套滚动条并存**：v1.3.0「滚动条霓虹化」遗产（thumb=glow 渐变+无效 opacity:0.7）盖过中性定义→chat 消息区/modelcenter 中栏/imagegen 左栏亮青粗条喧宾夺主，novel/sin 却是 scrollbar-width:thin 原生细条，同屏两态。**落地（纯 CSS 零 TS 零 Go 零绑定）**=①index.css 全局单一真源：8px 中性胶囊（thumb=on-surface 18% 半透明+2px 内缩+胶囊圆角，hover 30%——取 module-launcher 最得体形态升全局）②删霓虹化覆写+**顺手清死规则：v4.255 拍板的 ::selection（color-mix primary 30%）一直被文件尾 v1.3.0 glow 版覆盖从未生效**，两处合一保留生效形态（零视觉变化）③删三处重复定义（gaea/styles.css 全局 10px/redesign.css .gaea-app-layout 8px/module-launcher.css .ml 10px，各留指回注释）④保留功能性隐藏（novel-subnav 等）与 scrollbar-thin/hidden utility（无消费方）。**真机复验**=重建 exe 同脚本复跑 26 页次两态：亮青粗条全变中性胶囊，novel/sin 零变化，目检零回归。**观察池新增**=weixin 通道详情主区留白（填充需后端消息历史，待拍板）/办公右栏窄窗子代理卡横滚兜底/novel 技能下拉裸 id（技能库同以文件名为题，全站惯例不动）。**坑=巡检脚本切明暗须写 gaea-display-mode（新键），legacy gaea-dark '1' 不映射 dark；跨空间 navigate 被壳忽略须先点 ml-space-{work|play}**。**门禁**=ci.ps1/版本三处 4.271.0/drift OK@642；产物见 SHA256SUMS-v4.271.0.txt。
- **最新发布：v4.269.0（2026-09-13）「造价域线上断链修复」**——维持轨五算 e2e 走查抓到两个线上真 bug 当场修：①costproject/coststage 7 结构体无 json 标签→线上 PascalCase、前端读 camelCase→测算项目列表卡空名/¥NaN、五算面板全断（v4.204 起，保存方向大小写不敏所以录入正常没暴露）；②五算带入/手动保存/询价录入 payload 空串时间字段→Wails time.Time 解析必炸（v4.194/4.2 起）。修复=补 camelCase 标签（存量兼容）+载荷省略时间字段+前端类型改可选；Go+2 形状回归锁（TestWireShapeCamelCase）。**真机走查**=一次性项目端到端全通（列表卡正确渲染+带入桥验 stage:估算/100200/溯源备注落库+compose 诚实回退）。**门禁**=ci.ps1 全绿（run3；run1/2 已知 flaky 复跑绿）/版本三处 4.269.0/drift OK@642；产物=exe 49,281,024B SHA256=88864D3C…C2A3（桌面副本同哈希，冒烟 200 过）。
- **最新发布：v4.268.0（2026-09-13）「knip unused exports 甄别清账」**——瘦身 P5 既定池项（§100「169 甄别」，实跑仅 11+6+8 存量数字陈旧）。甄别原则=全死删（MODEL_PRICES/SPINNER_WORDS/getLocale/nodeStageLabel/CharacterFormHelpers 三件/engines 三接口/bridge.ts 五个死 re-export，本体文件各自健在）+双出口裁未用侧+**契约哨兵保留 @public**（drift/spaceBindings 的 AssertNever 锁零运行时成本）+误报注明（exportArtifact downloadBlob=test vi.mock spy 委托 knip 解析不到）；knip.json ignore 补 bindingNames.ts（drift PS1 文本消费）+drift.ts。接受的重复出口 3 处。**测试**=tsc 0/eslint 0/knip unused 清零/vitest 2977 全绿；零行为零 Go。**真机走查**=CDP 四页导航扫描渲染全过零前端错误。**门禁**=ci.ps1 全绿（run2；run1 rename flaky 复跑绿）/版本三处 4.268.0；产物=exe 49,281,024B SHA256=10A2566A…F15D5（桌面副本同哈希，冒烟 200 过）。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。














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
