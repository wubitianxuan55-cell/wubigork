# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

- **最新发布：v4.164.0（2026-09-08）「收敛计划 W1 卫生刀：供应链可复现+测试抗抖+过期导航 id 修复+壳内残留审计」**——收敛计划立项（docs/gaea-convergence-plan-2026-09.md）首轮，三线并发子代理足迹互斥。①package-lock.json 入库+CI npm ci（**坑：npm 11.6.2 --package-lock-only 产物缺条目，须 npm 10 生成或 dry-run 验证**）；②vitest testTimeout 15s+CI 前端 flaky retry（动因=Go/vitest 同机满并发 32 例 5s 超时假红、隔离复跑全绿——**门禁假红一次就会教人无视红灯**）；③修 DeliverablesPanel「回办公面板重新规划」失效：NAVIGATE 载荷沿用历史模块名 `page:"office"` 而 manifest 板块 id=gaea，MainLayout 白名单静默丢弃（**规约：NAVIGATE.page 是壳层板块 id，发送方必须用 manifest id**）；④WebView2 残留审计落档（P0×2：BaselinesPanel CSV/CharacterLibEditor 参考图；刀序 A-D）。零绑定 drift PASS@602；Go 116 包绿、vitest 2678 复跑全绿、tsc/eslint 0、build+冒烟过。欠账=nanoid high 升级刀、审计刀 A-D、LICENSE/性质拍板。

- **最新发布：v4.163.0（2026-09-08）「gaea 长期日志机制」**——此前=单文件 whisper_data/gaea.log（目录误植/无轮转/无上限）。机制（internal/app/logging.go）：**按日分文件 `<DataRoot>/logs/gaea-YYYYMMDD.log`**（当日 O_APPEND），启动清理过期（**保留 365 天**）+总量兜底（>1GB 从最旧删），旧 gaea.log 一次性迁入 logs/gaea-legacy.log；启动头写 version+Go 版本，Shutdown 记运行时长；前端诊断（GaeaLogFrontendError→slog）自动落同一文件；新绑定 **GaeaOpenLogsDir**+设置→数据「打开日志目录」按钮。**⚠附带修 v4.162 缺陷：ReadFileB64/SaveFileAs 漏 facade 委托行致运行时不可达——Wails 只暴露 Bind 里门面结构体的方法，App 直挂方法必须同文件在任一门面（bindings_*.go）加委托行；gen_bindings -names 只扫方法名探测不到漏委托，接手后新绑定必须壳内实测一次调用可达。****坑：①壳内页面调试=WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 + CDP 受信任点击（合成 el.click() 在 rc-trigger 行为不同）；②原生对话框探测 UIA #32770 漏报，须 P/Invoke EnumWindows 按 pid 枚举；③壳内首页入口是 [role=button] div 非 button 标签，冷启动 >10s 要轮询。**绑定 601→603、vitest 2677、tsc/eslint 0、drift PASS@603、版本三处 4.163.0、build+冒烟过。欠账：日志查看 UI 按需另立小刀。
- **最新发布：v4.162.0（2026-09-08）「进度计划导入导出壳内修复：原生对话框链路」**——用户实测「导入导出点击没有反应,包括菜单栏这些也是」。**根因=Wails WebView2 壳内 `<input type=file>.click()` 不弹文件对话框 + `<a download>` 下载不落盘**（浏览器 dev/prod 全正常,故测试全绿不可见;菜单/下拉本身正常）。修复:新绑定 **GaeaReadFileB64**(读所选文件,64MB 上限)+**GaeaSaveFileAs**(系统另存为+写盘) 599→601;导入四路壳内走 GaeaPickFiles+ReadFileB64 还原 File 喂原解析器(浏览器回退 input.click,hidden input 保留 jsdom 口径);导出五路统一 saveExportBlob(壳内另存为/浏览器 download)。**诊断配方（复用价值）：①分层排除=浏览器 dev/prod ✓ 壳内 ✗ → 壳特有;②壳内 DOM 调试=WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333 启动 + Node25 内置 WebSocket CDP,受信任点击必须 Input.dispatchMouseEvent（el.click() 合成事件在 rc-trigger 行为不同）;③原生对话框探测=UIA #32770 RootElement 搜索漏报,须 P/Invoke EnumWindows 按 pid 枚举;④壳内首页入口是 [role=button] div 非 button 标签,冷启动渲染 >10s 要轮询。**绑定 599→601、vitest 2671→2677、tsc/eslint 0、drift PASS@601、版本三处 4.162.0、build+冒烟过+壳内 CDP 实测对话框弹出 ✓。欠账：window.print 壳内行为未走查;全 app `<input type=file>`/`<a download>` 残留扫尾（其余入口多为绑定通路）。
- **最新发布：v4.161.0（2026-09-08）「双代号对齐标杆二期：工程标尺 + 图面语言」**——继续对齐重庆干休所标杆件。①**时间坐标框架进视图**：新纯函数 `aoaRuler.ts`（顶部工程日刻度/月/日+底部星期/工程周,列=工作日/步进按总工期 5-10-20/月标签落本月首个工作日/工程周每 7 列一档），月界竖线贯通图面；标尺与图面同一 scale 容器随缩放对齐；手动布局不画（x 与时间解耦口径）。②**波形线独立着色**：`segsToPathSplit` 拆独立 path 着绿 #22c55e（标杆图例语言）,`segsToPath` 保留兼容包装（d+' '+wave 逐字符等价旧输出,PdmView/既有用例零改动）,导出件同步。③**图面语言**：关键事件红圈(es=ls)/名称工期蓝色系/非关键线加深 #475569,图例同步。④**修 v4.160 带内打包阶梯化 bug**（目检实锤：按节点序打包把合并链拆成阶梯①②③逐级下降）——改按**全局重心行道 gRow** 压缩打包（带内行号=该带升序 gRow 序号）,链保持一条水平线。**坑：①v4.160 的行距「恰按公式」断言与打包模型耦合——布局断言写不变量（等距/放大/封顶）;②esbuild 目检 harness 必须放 frontend/ 内（node_modules 解析）,--bundle 对 css import 自动产同名 .css 手写占位会被覆盖;③Edge 无头截图（--window-size）是 SVG/组件目检的最保真轻量路径,MuPDF 直渲 SVG 黑白反转只核结构。**目检方式=esbuild 隔离挂载 AoaView+Edge 截图（releases/v4.161.0.md 有配方）。零绑定 599、vitest 2671→2677、tsc/eslint 0、版本三处 4.161.0、build 过。欠账：汇总总线双线样式观察池;标尺真机观感走查挂池。
- **最新发布：v4.160.0（2026-09-08）「双代号分级横幅：一级/二级分区布局」**——用户以重庆干休所上报件 PDF 为标杆（PyMuPDF 渲染逐象限目检提炼画法）：**每个分部工程一条横幅**——分部汇总线=横幅顶部通长线（分部名标线上，summarySegs 三段正交：竖起→横贯→竖落）、二级工序只在横幅内排布、跨分部一律虚工作竖向衔接、交替底纹分带。落地：①buildAoa band 归属（一级分组行=横幅,事件归成员最小 band=跨组共用界点归前组;组前未分组并头部合成横幅,组后并最后分组横幅;无分组=单一横幅退化旧行为）+band 内保序打包（沿用全局重心序为带内相对序,带间 1 行空档+首行半行帽）,行距自适应改按打包总行数;②**跨横幅不合并事件**（FS/SS 合并限同带+非首横幅无前置自设开始事件+START 虚工作补入）——否则后学分部挂前学分部事件→任务边跨带穿行标注压字（目检实锤）;0 工期虚工作不改时间参数/关键判定。**坑：①造样例数据任务 id 必须唯一,重名→byId 后者覆盖→链接解析成环假报「循环依赖」;②MuPDF 直渲 SVG 黑白反转文字糊只可核结构,保真目检用 Edge 无头截图（--force-device-scale-factor + window-size 控制幅面）;③v4.159 行距「恰按公式」断言在打包后失效——布局断言写不变量（等距/放大/封顶）勿复刻公式。**视图/导出同口径接底纹（sched-aoa-band-N testid）。零绑定 599、vitest 2665→2671、tsc/eslint 0、版本三处 4.160.0、build 过。欠账：用户真实计划横幅观感真机走查挂池;工程标尺四行视图侧未加（导出侧已有,按需另立小刀）。
- **最新发布：v4.159.0（2026-09-08）「进度计划画布手感：滚轮缩放 + 双代号行距自适应」**——用户实测反馈两处（附截图）。①**三块画布补滚轮缩放**：新共用钩子 `schedule/wheelZoom.ts`（普通滚轮=光标为锚缩放,Shift+滚轮=原生横移;**锚定=记光标下图面坐标、缩放提交后 useLayoutEffect 回写 scrollLeft/Top,同步回写会被旧内容尺寸钳制;监听器必须原生挂 {passive:false},React 合成 onWheel 是 passive preventDefault 无效;applyZoom 走 ref 中转免监听器反复拆挂**）;横道=Ctrl+滚轮缩日宽（MS Project 口径,**普通滚轮保持滚动行**,dayG 锚定+dayWRef 镜像）。②**双代号布局松绑**：AOA_ROW_H 74→112（旧 74px 行距扣掉节点上下时间标注 56px 只剩 18px 空隙=「全挤在一起」根因）+新 AOA_ASPECT_MAX=7 行距自适应（时标宽被工期锁定,低并行长计划天然 20:1 扁条——宽高比超 7 放大行距,clamp [ROW_H,2×ROW_H],y 与时间无耦合波形语义不变）+通道层距 12→16（层距须容「名称上/工期下」两标注,12px 叠压=截图红字互压根因）;PdmView 补 clientWidth≤0 适配防护（v4.143 jsdom 0 宽纪律同款）。**坑：el.dispatchEvent(WheelEvent) 不经 React act,断言读不到 setState 渲染——滚轮用例须 fireEvent.wheel;snapPt 网格随常量变 55×56,历史 pin 绝对坐标不受影响。**零绑定 599、vitest 2658→2665、tsc/eslint 0、版本三处 4.159.0、build 过（Go 仅版本常量,go build+app 包绿）。欠账：滚轮手感/自适应行距真机走查挂池。
- **最新发布：v4.158.0（2026-09-09）「AI 组价复核闭环：确认留痕 + 拆解合理性校验」**——造价板块接续（两路并行调研子代理：域代码盘点 file:line/需求材料核对）。**重要校正：AI 组价 v4.2 已在产**（组价→检索→价格带→LLM 拆解→确认回写全链路,gaea_cost_compose.go/priceband.go/ComposeModal）,roadmap §15「无嵌入索引/无异常检测」等欠账描述已过时——现状基线落档 **docs/gaea-cost-domain-survey-2026-09.md**（含刀路池:条目匹配键/流式分步/五算贯通/询价库级扫描/索引主动维护/含量对照基线）。本刀销两真缺口:①**SchemaV16 cost_compose_records**（契约写 V11 但实际迁移链已到 V15,子代理正确顺延——**接手契约先核对迁移链实际编号**）确认即留痕完整视图快照,无确认不落库红线不变,留痕尽力而为不阻断回写;GaeaCostComposeRecords(entryName 空=全库最近 20)回看,坏 snapshot 行如实 nil;CostEntryModal「组价依据(N 次)」只读折叠区,证据链表抽共享 ComposeEvidenceTable(ComposeModal 零视觉变化迁用);SQL 落点=app 层直写+cost.Store.DB() 一行句柄委托(**hubCostStore 测试注入路径与生产一致是关键约束,勿绕 store 直取 db.GetDatabase 打真实用户库**)。②**cost.CheckComposeComponents** 纯函数(R1 金额一致性/R2 非正值(不进 R1R3 防双重误报)/R3 合计×1.05 warn ×0.5 info,恰阈值不触发)进 View.checks,ComposeModal 行级徽标(同行归并 warn 主导)+全局提示,校验随留痕入档=回看时见「当时提示过什么」。**Go 全量绿(+18)、vitest 2650→2658、tsc/eslint 0、drift PASS@599、版本三处 4.158.0、build+冒烟过;flaky=app 包 TempDir 清理竞态+GitPanel 全量并发 28 败(均单跑/复跑绿,在册同族)。欠账:组价依据区 mock 下恒空真机走查挂池;刀路池见 survey 文档。**
- **最新发布：v4.157.0（2026-09-08）「Office 编辑链一致性小刀：docx 证据链补齐 + pptx 面板字符级对比」**——两个小项收口 v4.156 三件套编辑线（零新绑定 598）。①**docx_apply 补证据链**（pptx 设计 §6 拍板项 4 推荐项）：GaeaDocxApplyEdit 落盘前快照（.gaea/work/rollback/docx-<ts>.before,尽力而为不阻断）+Journal（Tool=docx_apply,摘要 truncateStr 120,BaselinePath 如实）——**ApplyTrackedReplace 编辑语义零改动**,docx 编辑从此可经 GaeaRollbackRecord 回滚（VersionTimeline kind:"docx" 已支持）;GaeaDocxAcceptChanges 仅 accept==true（清除修订标记=不可逆整理）加同款,accept==false 不加（拒绝=恢复原文天然回滚）。②**PptxEditPanel 对比区升级 ChangesDiff**（刀3 层3 在面板侧接通）：docxTextDiff 句级 LCS 输出归一 DiffRow——changed=相邻 del+add 对交 pairModifications 改蓝配对+字符级高亮,未变句给 ctx 行 foldContext 折叠不伪造全量;删「句级对比（无字符级高亮）」降级标注,「不支持修订标记可恢复」常驻标注保留;状态机/应用逻辑零改动。**坑：gaeaEffectiveSpace() 单测缺省返回 ""≠"work",守卫注入先例=直赋 ga.cfg=&gaeaConfig.Config{}（零值→mode on+space work,gaea_pptx_test.go:248 先例）,Workspace 留空使 gaeaCwd() 回退 os.Getwd()=t.Chdir 目录。**Go app 包全绿（+3:apply 快照+Journal 字段逐项/120 截断口径/accept 两分支对称）、vitest 2650 全绿、tsc/eslint 0、drift PASS@598、版本三处 4.157.0、build+冒烟过。**欠账:pptx 刀2 真机走查仍挂池;刀4 修改队列待反馈;Verifier 通道 B 对 docx_apply 的复核口径未动。**
- **最新发布：v4.156.0（2026-09-08）「pptx 真编辑刀2+刀3：编辑面板 + 版本对比（Office 三件套编辑闭环）」**——docs/gaea-pptx-edit-design-2026-09.md 刀2/刀3 三线并行交付（Go 绑定线/前端刀2 线/前端刀3 线,足迹互斥;刀1 数据层早于 v4.109.0 落地,设计文档状态行曾过时——**接手先核对设计文档状态 vs git log**）。**绑定 597→598**（+`GaeaPptxSlideText`=pptxedit.LocateText 段落全文,纯 Go 零 python）：动机=大纲 texts 是 200 rune 截断预览,直接当 apply 的 target 必匹配失败（ApplyTextReplace 单段落精确匹配、宁拒不误改）——段落粒度即 target 粒度。①刀2 编辑面：PptxOutline 两级导航（页→文本框,页锚点/composer 插入/降级全保留,未接线宿主编辑入口不渲染）;新组件 **PptxEditPanel**（P1=xlsx 同款 Plan→Apply:载入段落全文失败诚实降级→段落单选→预设四动作+自定义指令（附加约束「短句化、长度相近防溢出」拼 instruction,GaeaOfficeEditText 零改动复用）→双栏对比（docxTextDiff LCS 句级整块着色,诚实标注无字符级）→应用（后端错误原样透出,成功用返回 PreviewResult 刷宿主+静默重拉）→常驻「PPT 不支持修订标记,应用即生效可回滚」）;FilePreview pdf 分支挂面板（DocxQueuePanel 宿主先例）。②刀3 对比：lib/pptxTextDiff.ts（JSZip 解 slides 文件名自然序=显示层近似（后端 sldIdLst 放映序,注释明示）;页签名 LCS 对齐+changed 页复用 diffDocxParagraphs 段落级;坏 zip 结构化 {ok:false} 降级）;versionCompare kind:"pptx" 字段对齐 XlsxDiff;**取数通道偏离 docx=GaeaAttachmentDataURL**（GaeaPreview 对 pptx 渲染成 PDF 逐页缩略且基线快照 .before 扩展名不走 Preview 分派——docx 的 Preview-dataUrl 通道两侧都拿不到字节,照抄即死代码）;VersionTimeline PptxCompareBody 页摘要+差异页 hunk（改蓝配对+字符高亮白得）。**坑：①gaea 域无 lib/api.ts——组件直调 app.*（DocxPreview 先例）,契约偏差如实记录;②mock Preview 无 .pptx 分支（v4.28 起即如此）→?mock 下编辑链路不可达,真机走查挂真机池（组件状态机 23 测试覆盖）;③bindings_office.go 头部标注 generated——手加委托后 gen_bindings 再生成一致（drift PASS@598 实证）;④TestCreateChapter_SameChapterConcurrentRejected 全量跑偶发 TempDir 清理竞态（Windows unlinkat not empty）,单跑×3+整包复跑绿=在册 flaky 同类。**Go 全量绿（+3）、vitest 2617→**2650**（+33）、tsc -b/eslint 0、drift PASS@598、版本三处 4.156.0、build+冒烟过。**欠账：真机走查（mock 不可达）;刀4 修改队列泛化待刀2 反馈拍板;docx_apply 不落证据链（设计 §6 推荐另立小刀）;面板内字符级高亮未接（句级诚实降级）。**
- **最新发布：v4.155.0（2026-09-08）「双工期欠账放开：SS/FF/SF×cd 全搭接（§3.2.5 重推，零不动点）」**——双工期设计 §6 欠账池首项销账：v4.150 刀1 曾 fail-closed 收窄「cd 任务仅允许 FS 搭接」，本刀按 §3.2.5 要求实施前重推语义矩阵——**原矩阵四格「⚠ 需不动点」均系 wd 代数形误判，实际零迭代可解，八格全放开**。①新原语 `cdEarliestStart`/`CdEarliestStart`（单调逆查：最小 s 使 cdToEf(s)≥T；估锚 max(0,T−cd)+有界双侧校正，与 cdToEf/cdLatestStart 互为镜像三件套）解 FF/SF-to——fwd 关于 es 单调不减有平段，逆像确定存在；②SS/SF-from **直接 ls 下界**（SS/SF 约束本就落在开始上 `ls≤to.ls−lag`，不经 lf+durFrom 换算——换算才是原矩阵担心的不动点源）+后继 **effLf**=cd 后继的 cdToEf(ls)（FF/SF 对前驱传播读实际最迟完成才诚实；v1 全 FS 组合永不触及，旧 pin 零风险）；③逆推 from=cd 拆双上界：lf 收 FS/FF、lsBound 收 SS/SF，`ls=min(cdLatestStart(lf), lsBound)`、lf 保留 raw（兼容「fwd(ls)≤lf」pin）；④wd 路径逐位不变（FS 项无条件读 to.ls 与旧式恒等，快路径用例零回归）。⑤拆闸三处：Go Validate「仅 FS」/引擎防御拒绝/analyze 防御 finding，cd 余四闸不动；文案四处同步（scheduleApply 描述/技能条款（锚点锁 37 不变）/types 注释/甘特 chip title）。**测试：锚点表×6（cdToEf/cdEarliestStart/cdLatestStart×cd=5/7，平台 0-2→5、5-7→10）+用例 A-E 两侧镜像断言 es/ef/ls/lf/tf/critical/duration；golden 62→65（cd+SS(lag2)/SF(lag1)/upsert cd+FF 成功例）。坑：①契约用例 D 原稿把 SS 正推期望误写成 FS 式数字（es=5 只有 from.ef+lag 能得出）——两侧子代理独立发现按定稿语义收敛同组修正值，镜像无损；写引擎测试契约时正推/逆推期望必须按同一搭接型自洽；②既有「cd 非 FS fail-closed」用例物理上不可能零改动，原位翻转为 ok=true 是正路。**Go 全量绿（+2 函数/5 子用例）、vitest 2601→**2617**（+13 TS+3 golden 双跑）、tsc -b/eslint 0、drift PASS@597、版本三处 4.155.0、build+冒烟过。设计文档 §3.2.5/§6 已回写重推结论；欠账池剩=导入任务日历转 cd 迁移助手/斑马导入口径探测/等效跨度灰显双读数（按需）。
- **最新发布：v4.154.0（2026-09-08）「MPP 导入支持 Project 2013+ 变体（真机样本驱动修复）」**——用户真实工程文件（重庆干休所总进度计划，Project 2013+ 保存）此前被 ParseMpp 拒收（「缺 CompObj」），强拆后系统性错乱。**字节级取证钉死 2013+ 变体五处漂移**：①新版无 CompObj 流→工程目录编号判版回退（`   114`=MPP14；无 CompObj ⇒ appVer≥15=lag@14 口径）；②任务工期 i32@42→**@84**（@88 恒同值；全行偏移扫描唯一全行命中且全为整天）；③Var2Data 回链键 uid@0→**id@4**（uid 是陈旧键与名字表大面积错位——27 空名根因）；④parentUID@36 恒 0 失效→**大纲层级 i16@172**（0 项目/1 总账/2 分部/3 分项与 WBS 逐行吻合）+层级栈重建父链；⑤FixedMeta byte8 恒 0xe3 失效→里程碑=零工期语义。搭接 pred/succ 确认同为 ID 键空间（45 行全命中），lag@14 数值合理（14d/27d）。**样本现已完整导入：38 任务/8 分组/42 搭接/总工期 136 天/任务名零缺失，CPM 自洽（192 自然日≈136 工作日×7/5）。**诚实降级：2013+ 资源/分配行键位未钉死（解析出垃圾资源），宁缺勿错暂不解析留观察池；日历例外日仍不解析（v4.140 口径，原文件 150 天 vs 引擎 136 天差异主因）。**坑：①Fixed2Data/Fixed2Meta 不是任务第二表（是字段定义/陈旧快照，勿当主数据源）；②全行偏移扫描=定位漂移字段的最快取证法（候选偏移取「全行命中+业务数值合理」交集）；③gated 样本落 clones/mpp2013/（git-ignored）。**TestMppRealSamples2013 门控落档（38/8/42/136 钉死），旧三样本（MPP9/12/14-2010）零回归；Go 全量绿、版本三处 4.154.0、drift PASS@597、build+冒烟过。
- **最新发布：v4.153.0（2026-09-08）「双工期口径刀4：mspdi 互通」**——双工期四刀**全部收官**（纯前端 597 零变更，Go 零改动）。①导出：cd 叶任务 `<DurationFormat>8</DurationFormat>`（elapsed 日历天）+ `PT{自然日×24}H0M0S`（28 自然日=PT672H）；wd 恒 7+PT{d×8}H 逐位不变；分组行恒工作日口径；Start/Finish 本就 wdToDate 日历日期自动吻合。②导入：DurationFormat=8 → `durationUnit:'cd'`（分钟÷1440）；其余/未知格式维持 ÷480 工作日折算宽松容错；里程碑/汇总不标 cd（normalize+引擎 fail-closed 双保险）；`parseDurationDays`→`parseDurationMinutes` 按格式分流（FORMAT_*/MINUTES_PER_ELAPSED_DAY 常量显式化）。③**真机池（本机未装 MS Project，沿用「等真机」先例）：Project 打开导出文件对 8 的显示与重算、`PT×24` 序列化假说钉死（XSD xsd:duration 语义允许，若有出入仅影响 Export 端常数——Import 端 ÷1440 与分钟解析两口径兼容）、WPS/斑马行为；真机走查通过前 Export 端对 Project 实际兼容性视为待验证，Import 端是失真修复方向性收益确定。**vitest 2596→**2601**（+5：导出口径/roundtrip/Format8 导入/未知格式回落/elapsed 里程碑不标 cd）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。**双工期口径设计（durationUnit wd|cd）四刀全落地：模型引擎/agent 通道/板块 UI/mspdi 互通；拍板池三项（diff 确认卡/多工程/双工期）全部收官。**
- **最新发布：v4.152.0（2026-09-08）「双工期口径刀3：板块 UI」**——纯前端 597 零变更，Go 零改动。①工期列「日历」chip：cd 行常显（primary ghost 点击切回工作日）、**wd 行 hover 行时浮现**（CSS 渐进披露=「wd 行不显后缀」条款落地）、分组行/里程碑无入口；点击循环 wd↔cd **不换算数值**（拍板项 7，28 wd→28 自然日）；工期列宽 48→84。②拖拽缩放：drag.ts 新 **`resizeToNaturalDuration`**（右缘=新 ef 工作日吸附不变，回写**自然日差**=offsets[newEf]−offsets[es]，引擎复算 ceil 吸附不动=所见即所得）；**右缘起点 `dayNo(origEs+origDur)`→`dayNo(row.ef)` 顺带修复 cd 条形画过宽**（自然日数曾被当工作日序号加——条宽改按 dayNo(ef)−dayNo(es) 真实日历跨度，wd 两值逐位相等零回归）；拖拽浮标「日历 X → Y 天（自然日）」按自然日直绘；move 语义不变（manualStart 恒工作日）。③口径透明：条形悬停「日历天 28（等效 20 工作日，日期区间）」等效只读（不做双值列拍板项 7）；前锋线 frontierWd 的 dur 换源 ef−es；检查器工期行「28 天（日历，等效 20 工作日）」；状态栏「总工期 N 天（工作日）」；单代号节点/双代号箭线 cd 标「(日历)」+两图图例注记（**AOA 网格恒工作日刻度，cd 箭线长度仍自然日数=计算不经视图诚实标注**）。**坑：①交互测试必须 setState store project 与渲染 prop 同源（拖拽/点击写 store，渲染用 prop——不同源则断言打到旧工程）；②全量 vitest 并发 flaky 28 败（在册先例），单跑+复跑两轮全绿再发布。**vitest 2589→**2596**（+7）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。**刀4 mspdi 互通（DurationFormat 8 导出导入，绑真机池——不真机不发布）待续。**
- **最新发布：v4.151.0（2026-09-08）「双工期口径刀2：agent 通道」**——刀1 引擎能算 cd 但 agent 说不了这个话，本刀打通对话链路（设计 §3.5，零新绑定 597）。①ops 通道：patchTask.DurationUnit 指针三态（缺 key=不动、**空串 no-op 同 Mode 先例**、非法枚举报错原文）；**单位分支先于工期分支**——「工期 28→30」词尾随新单位（cd 显「日历天」），cd 态工期超 3650 当场拒；回执口径词「工期口径→日历天（自然日定时）」/「工期口径→工作日」；upsert_task 的 Task 直带 durationUnit（零值缺省=wd，校验同 Validate 五闸，新增回执「工期 28 日历天」）。②**Go/TS 对拍 golden +12 例（49→61）**：单位切换/显式 wd/空串 no-op/分组行/里程碑/超上限/非法枚举 + 「cd 全链 upsert+FS+基线」（cd 进 SnapshotBaseline 的等效跨度口径由对拍钉死），opsSim.golden.test.ts 63/63 过——语义漂移即炸对端。③三工具：schedule_get taskView 带 durationUnit（omitempty，wd 不出=旧客户端零噪音）；schedule_analyze 增 **cdTasks 清单**（id/name/duration 自然日数/span 等效工作日跨度/anchor 锚点日期=es 所在工作日）+ qualityChecks 增 cd 非 FS 防御 finding（闸在 Validate/apply，疑旧口径残留提示改 FS）；schedule_apply 回执「写入后总工期 X 天（工作日）」补词。④schedule-edit 技能：工期分节增 cd 纪律（养护/干燥/成活期自然日定时才标 cd/搭接仅 FS/成本按等效跨度 ef−es=元/工日不因日历天放大/定额合同工期禁误标）；汇报口径补「总工期 X→Y 天（工作日）；混排计划先讲 cdTasks」；**锚点锁 35→37**。compact 三工具同步。**Go 全量绿（+3：ops 口径词/引擎 cdTasks/工具链路 TestScheduleDualDurationChain）、vitest 2576→**2589**、tsc -b 0、drift PASS@597、build+冒烟过。**刀3 板块 UI（工期列单位切换+chip+悬停等效提示+拖拽 resize 换算）→刀4 mspdi 互通（绑真机池，不真机不发布）待续。**
- **最新发布：v4.150.0（2026-09-08）「双工期口径刀1：任务级工期单位（模型+引擎）」**——拍板池最后一项欠账启动实施（docs/gaea-schedule-dual-duration-design-2026-09.md 推荐项全采纳）：`task.durationUnit?: 'wd'|'cd'` 缺省 wd=旧文件零迁移零行为变化；cd=日历天（养护/干燥类自然日定时，对齐 mspdi DurationFormat 8），28 个自然日误录 28 工作日的 +8 工作日失真消除。①换算纯函数对 `calendar.ts cdToEf`（正推边界=es 所在工作日+cd 自然日，**ceil 吸附**到首个 ≥边界的工作日序号；26/27/28cd 整周工作制下同 ef=20 吸附折叠）/`cdLatestStart`（逆推最大 s 使 fwd(s)≤lf：镜像回退+有界双侧校正，性质 fwd(ls)≤lf<fwd(ls+1) 钉死；平段取最大 s；负时差极端截 0）。②cpm 增可选 ctx `{calendar?, startDate?}`（Go 先例式双入口 `ComputeCpmCal`/`ComputeCpmPlanCal` 委托同一 `computeCpmFull`）：**快路径铁律**=无 cd 任务分支全短路逐位一致（既有用例零改动全绿+逐位等价用例）；有 cd 而**缺开工日期 → ok=false**（日历缺省回落周一~五=对设计 §3.2.2 的有意识细化，与全部 wdToDate 读点同口径）；cd 涉非 FS 引擎层同样拒（Validate 之外防直接调用）；逆推 FS 上界统一 `to.ls−lag`（wd 后继与旧式逐位相等，cd 后继必须用已换算的 ls）。③成本/基线换源：work 分配与基线快照/漂移 dur 改用 **cpm 行等效工作日跨度 ef−es**（wd 两值恒等有 pin；养护期周末不记工日，28cd×100 元/工日=2000 非 2800；日历变化改 cd 等效跨度时 durDrift 如实呈现=诚实口径）。④校验：Go Validate 五闸（枚举 wd|cd/分组行禁 cd/里程碑禁 cd/数值上限 3650/搭接仅 FS）+TS normalizeProject 容错回落（违规丢字段回 wd）。⑤接线零行为变化：TS 六处 computeCpm 调用点传 ctx（SchedulePage/store/applyDiff/gschedSummary/mock/opsSim——opsSim 与 Go set_baseline→SnapshotBaseline 镜像对齐），Go 四处（Save/Analyze/SnapshotBaseline/schedule_tools Before 摘要）。**坑：①ComputeCosts 收 Project 值不收指针（测试样板函数别返回 *Project）；②grep 中文偶发 GBK 显示乱码——file/python 复核字节，非真损坏勿慌；③发布索引四处（releases/README/README 表/CHANGELOG 头插/AGENTS 速览+一行账）+版本三处（sync-version.ps1 一把梭）。**刀2 agent 通道（patch_task.durationUnit+cdTasks 清单+技能条款）→刀3 板块 UI→刀4 mspdi 互通（绑真机池）按序待续。Go 全量绿（+10）、vitest 2550→**2576**（+26）、tsc -b 0、drift PASS@597、版本三处 4.150.0。
- **最新发布：v4.149.0（2026-09-08）「diff 确认卡刀D：ops 通道投影模拟器 + Go/TS 对拍收官」**——diff 确认闭环设计**四刀全部收官**（A 回滚/B diff 卡/C 引擎强制/D 投影对拍）。①`schedule/opsSim.ts` `simulateOps(before, ops)`=Go internal/schedule/ops.go ApplyOps 的逐语义 TS 镜像：深拷贝顺序应用 fail-closed（单条失败即中止、半途状态不外泄）；指针三态（缺 key=不动、显式 0/空串=改、mode 空串 no-op）、upsert_task 零值语义（缺 level=0 分组/缺 progress=0）、remove_task 扁平数组 level 子孙级联+搭接清理、set_links 整体替换缺省 FS+自前置/悬空拒绝、set_meta 日期 10 字口径+deadline 空串清除/null 不动、auto_chain 手动跳过+无可补报错、set_baseline 缺 savedAt 拒绝+无 planFinish CPM 口径、资源级联/重复对/外来 taskId/坏金额全对齐。②**Go/TS 对拍主阵地**：internal/schedule/ops_golden_test.go 生成并校验 `frontend/src/schedule/ops_golden.fixture.json`（49 例=21 成功+28 失败路径，覆盖 12 op 全守卫分支；`go test -run TestApplyOpsGolden -update-golden` 重生成），opsSim.golden.test.ts 同文件双跑逐例比对「是否失败+after 状态」——**任何一侧语义漂移都炸对端测试**；canonical 比对消化 Go omitempty 与 TS undefined 表达差（空数组双侧同丢）。③卡体：ops 通道缺省路径即 simulateOps 投影→diffProjects 同管线真 diff（「ops 投影」来源标注），三级诚实降级（非缺省路径/读取失败/模拟失败→意图清单+原因）；落盘权威永远在 Go。**坑：①jsdom 里 import.meta.url 非 file 协议读不了 fixture——改 resolveJsonModule（tsconfig.app.json）+直接 JSON 导入；②canonical 不丢空数组则 Go omitempty 空切片与 TS [] 必炸；③control 包同包测试的 strPtr 等指针助手已存在，新增要换名（gStrPtr）；④if 语句头复合字面量加括号 (gateApprover{c}).Approve。**Go 全量绿、vitest 2499→**2550**（+51）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。
- **最新发布：v4.148.0（2026-09-08）「diff 确认卡刀C：project 整量替换引擎强制逐条确认」**——schedule_apply project 通道（args.project 非空=整计划替换/生成）在 `gateApprover.Approve` 增分支：hardAskSet 判定之后、auto 快捷判定之前 → `requestApproval(alwaysPrompt=true)`——任何权限级别（含 auto/yolo）前置逐条弹卡、不读不写会话放行（拍板项 2 禁会话记忆自然落地）；ops 增量通道维持现状闸门（拍板项 3：auto/yolo 豁免，后置回执+回滚兜底）。`approvalSubjectFor` 增 project 分支 `scheduleApplyProjectSubject`：解析 args.project→`schedule.Project.Analyze()`（同一引擎口径）→「整计划替换（目标文件）：N 项工作，总工期 Y 天，关键 K 项」；CPM 不过标注「批准也将被引擎拒绝」；before 侧对比不进 subject（闸门无工作区上下文，由前端 diff 卡承担）。前端 ApprovalModal：`scheduleApplyIsProjectChannel`（applyDiff.ts 镜像判定）→ project 卡按 hardAsk 三钮形态渲染（hardConfirm 统一分支），ops 五钮不变。schedule-edit 技能 Body 增 **「确认与回滚」** 分节（project 逐条批准/被拒=未写入且禁止原样重发/超时=拒绝如实告知/以回执为准/回滚入口=工具卡「回滚本次」+变更 tab），锚点锁 **31→35**；scheduleApply.Description 尾部+compact.go compactDesc 同步确认机制。**坑：①if 语句头用复合字面量 gateApprover{c} 必须加括号 (gateApprover{c}).Approve（vet 抓）；②control 包 import internal/schedule 无环（引擎是叶包）可放心；③不动 event.Approval 的设计红线——前端 hardAsk 形态用 args 判定镜像，强制权威在 Go。**刀D（simulateOps TS 镜像+Go/TS 对拍）待续。Go 全量绿（+5 闸门用例）、skill 锚点过、tsc -b/eslint 0、vitest 2499、drift PASS@597、build+冒烟过。
- **最新发布：v4.147.0（2026-09-08）「diff 确认卡刀B：schedule_apply 审批卡升级为结构化 diff 预览」**——diff 确认闭环设计刀B（§2.2）。①纯函数层 `schedule/applyDiff.ts`：`diffProjects(before, after)` 两侧 normalizeProject 归一后纯比较——任务 id 键控增删改（字段级 from→to 中文 label：任务名/工期/进度/层级/里程碑/模式/手动开始/固定成本）、搭接按 to 分组入边比对（对齐 set_links 整体替换语义）、资源删除标级联分配数、分配 (taskId,resourceId) 键控增删改、meta 四项；汇总行=总工期/关键/总成本双方 computeCpm/computeCosts+倒排校核+基线漂移预览；**after CPM 不过如实标注「引擎将拒绝」（预览层复刻 fail-closed）**；empty 判定。`scheduleApplyArgsOf(items)`=宿主从 Transcript 取 running 工具卡 args（§1.3 ToolDispatch 先于闸门）。②`ScheduleDiffCard` 卡体 + ApprovalModal 新 prop `scheduleApplyArgs`（approval.tool==="schedule_apply" 时替换通用参数原文区，决策/快捷键/hardAsk 不动；双宿主各传各的 items）。降级三级：非缺省路径（loadScheduleFile 只服务缺省）/读取失败/ops 通道（simulateOps=刀D）→ 意图清单+「无现状对比」诚实标注；卡头「预览数字，落盘以回执为准」；60 行截断+展开全部。**坑：①i18n schedDiff.* 键位有意识偏离不做了——卡体是 schedule 域内容层（§405 zh-only），未新增 shell 键；②组件文件只许导出组件（react-refresh），scheduleApplyArgsOf 放 applyDiff.ts 纯函数层；③全角空格进 JSX 文本踩 no-irregular-whitespace；④CpmResult 从 types 导入非 cpm.ts；SchedCalendar 字段名 holidays 非 exceptions（TS 会抓）。**刀C（Go approvalSubjectFor project 分支=整计划替换 hardAsk 化+锚点锁 31→N）→刀D（simulateOps+Go/TS 对拍）待续。vitest 2483→**2499**（+16）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。
- **最新发布：v4.146.0（2026-09-08）「对话改计划回滚闭环（diff 确认卡刀A）」**——v4.125 落档的 diff 确认闭环设计（docs/gaea-schedule-diff-confirm-design-2026-09.md）按刀序启动，刀A=后置回滚补全（设计原文：即使确认卡不做也独立成立）。①变更 tab：WRITE_TOOL_NAMES/WRITE_ONLY_TOOL_NAMES +schedule_apply（三态完备性锁同步满足）；`extractChangedPaths(args, tool?)` 增可选 tool 参——schedule_apply 缺省 path 回填「进度计划/当前计划.gsched.json」（与 Journal target 同口径，缺省路径整计划替换不漏记）；buildChangeDiff 显式降级说明不伪造行级 diff。②新组件 `ScheduleApplyRollback`：schedule_apply 回执卡 done 后常驻「回滚本次」行（ToolCard 内 TaskLiveRow 同位、不进折叠区）——Journal 按 target 匹配**最新一条带基线快照**的记录（ChangesPanel 同款语义），点击调既有 GaeaRollbackRecord（「已被手工修改」守卫在 Go 侧）；无匹配整行不渲染；多次 apply 同路径不逐卡区分（Journal 无调用级关联键，标题带 turn，设计 §5 欠账另案）。**坑：①新刀插入文档头又吃掉上版标题（第三次！）——「old_string=上版标题行」的插入方式必须把标题原样带回 new_string 尾部，已两次靠事后核对 head 挽救；②缺省路径常量在 changes/组件本地镜像不 import gschedSummary（防拉起 schedule store 图谱，DEFAULT_SCHEDULE_PATH 先例）；③旧测试锁「WRITE⊆EDIT∪WRITE_ONLY」，加白名单必须同步选边（schedule_apply→WRITE_ONLY）。**刀B（diffProjects+ScheduleDiffCard+ApprovalModal 变体）→刀C（Go 审批闸门 project 分支+锚点锁）→刀D（ops 投影+Go/TS 对拍）按设计刀序待续。vitest 2477→**2483**（+6）、tsc -b/eslint 0、drift PASS@597、build+冒烟过。

- **近期三版一行账**：v4.156.0 pptx 真编辑刀2+刀3（编辑面板+版本对比,Office 三件套编辑闭环,绑定 598）· v4.155.0 双工期欠账放开（cd 全搭接 SS/FF/SF,§3.2.5 重推零不动点）· v4.154.0 MPP 导入 Project 2013+ 变体（真机样本五处漂移修复）——观察池=mppdi 刀4 真机 Project/WPS/斑马走查、2013+ 资源/分配键位、E1 负数对称、list_projects 真机、**pptx 刀2 真机走查**；双工期欠账池剩=导入任务日历转 cd 迁移助手/斑马导入口径探测/等效跨度灰显双读数（按需）；pptx 欠账=刀4 修改队列泛化/docx_apply 证据链另立小刀。
- **历史存档**：v4.110 之前的逐版全文与本文件原「版本状态」历史段（v2.x~v4.135，2100+ 行）已于 2026-09-09 迁往 docs/archive/agents-version-history-2026-09.md；长期有效结论已沉淀在下方各专节（执行纪律/发布流程/已知注意）。

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

## 长期规划（权威，2026 定稿）

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
- **执行审计（2026-08-30）**：`docs/audit-2026-08-30-v4-execution-review.md` 记录
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
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`
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

- `docs/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

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
