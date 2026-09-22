## v4.393.0 · 绘梦·蒙版局部重绘：涂选要改的区域，其余保持原样（2026-09-23）
> 图像域长期规划 T3「蒙版局部重绘」；v4.392.0 观察池销号第 2 项。病根：双编辑档（OpenAI 兼容 /images/edits + ComfyUI Qwen-Image-Edit 2511）都有蒙版通道但从未接线——前端没有涂选交互，后端请求没有 Mask 字段；全图编辑对已满意区域有回归风险。规格 进度计划/gaea-mask-inpaint-20260923.md，绑定面 714 零变更。【落地】①统一蒙版契约：全链 Mask=灰度 PNG data URL 白=重绘区黑=保留区——前端黑底白笔刷导出直观可预览；ComfyUI 零转换（ImageToMask(red) 白→1.0），OpenAI 一处转换 grayMaskToOpenAIMask（白区→alpha=0，OpenAI 语义透明=编辑区；标准库零新依赖）②ComfyUI 蒙版链（尺寸对齐是技术核心）：LoadImage(mask)→ImageScale(lanczos,targetW/H)→ImageToMask(red)→SetLatentNoiseMask(VAEEncode)→KSampler.latent_image——targetW/H 复刻 PREFERRED_KONTEXT_RESOLUTIONS 17 档最近宽高比（本机 ComfyUI 0.36 nodes_flux.py:105 实读），SetLatentNoiseMask 只裁剪不 resize 蒙版必须与缩放图同尺寸否则错位；原图尺寸 DecodeConfig 取得，解析失败诚实报错；TextEncode image1 保持全图（语义参考需全图上下文，重绘区域由 noise_mask 限制）③OpenAI editImage Mask 非空时转换附 mask multipart 字段④app：mediaGenParams +mask 键；fail-closed mask 非空且 mode!=edit 拒绝（img2img+蒙版留观察池不静默忽略）⑤前端新 MaskBrushEditor（strokes 状态驱动：显示层半透明红笔刷/导出层黑底白笔刷同坐标重放；指针坐标 getBoundingClientRect 比例映射自然尺寸 CSS 缩放不影响精度；jsdom 无 2d ctx 导出守卫 null 不崩；renderMaskDataURL 归 ui.tsx 遵 react-refresh 惯例）+InstructionEditModal 范围切换（全图/局部涂选）+笔刷滑杆+清除+已涂 N 笔状态；局部未涂抹 warning 不触发生成。【测试】Go +4 组（grayMaskToOpenAIMask 白→alpha0 黑→255 尺寸不变非 data URL 报错/kontextScaleSize 方图 1024²·2:1→1456×720·退化兜底/OpenAI edit+mask multipart 形状捕获回读断言 alpha/ComfyUI edit+mask 工作流形状：蒙版链四节点+ImageScale 对齐 1456×720+KSampler latent 改接 SetLatentNoiseMask+TextEncode image1 保持全图，无蒙版回归由既有形状测试钉住）+app +1（mask+img2img 拒绝不触后端/mask+edit 透传）+前端 +4+2（editor 初始态/涂抹计数+回调守卫 null/清除/纯函数空笔画；modal 局部未涂 warning+涂后 mask 透传/全图不带 mask 键）；全量 396 文件 3353 例绿+tsc 0+eslint 0。【门禁】全量 ci 绿 EXIT=0+漂移闸 OK@4.393.0。【坑】①蒙版黑白是数据语义非主题色——local/no-raw-hex 要加 hex-exempt 注释②SetLatentNoiseMask 尺寸语义=裁剪不 resize——蒙版与 latent 空间必须显式对齐（复刻 FluxKontextImageScale 的 17 档表）③工作流断言经 JSON 往返数字全变 float64——int 字面量比较必假④vi.mock 工厂不能引用顶层标识（hoisting）——antd 桩改用真实组件+组件侧包 testid⑤antd Slider 不转发 data-testid——外包 span。【产物】exe 见 SHA256SUMS-v4.393.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.389~v4.393 删 v4.388.0.exe。【文档】releases/v4.393.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 414→415+34 席插 v4.393 裁 v4.359）+AGENTS 迁 1 插 1（一百零八迁：v4.388 入 archive）+progress/todos。

## v4.392.0 · 绘梦·指令编辑本地档：ComfyUI 接入 Qwen-Image-Edit 2511 官方工作流（2026-09-22）
> 图像域长期规划 T3「本地档（B 计划）」第一刀，阶段一刀 C（v4.327）观察池第 2 项销号。病根：指令编辑只有云端档（OpenAI 兼容 /images/edits），ComfyUI 诚实拒绝——而用户本机全局生图后端=comfyui，编辑在日用后端上不可用，与本地优先基本盘冲突。规格 进度计划/gaea-comfyui-edit-20260922.md，绑定面 714 零变更。【落地】①internal/ai：mode=edit 从拒绝改真实现——buildQwenImageEditWorkflow 蒸馏官方 comfyui-workflow-templates image_qwen_image_edit_2511（取道不取器，API-format 与 krea2/z-image 同范式）：LoadImage→FluxKontextImageScale（按原图宽高比重标不套 1024×1024）→缩放图同时进 TextEncodeQwenImageEditPlus×2（image1+vae）与 VAEEncode→KSampler.latent_image；UNETLoader→ModelSamplingAuraFlow(3.1)→CFGNorm(1)→KSampler（20 步/CFG 4.0/euler/simple/denoise 恒 1.0=语义编辑非整幅重绘，官方 Note「Comfy」列参数）；FluxKontextMultiReferenceLatentMethod 官方明示官方权重不需要不接；Lightning 4 步 LoRA 留观察池；缺原图报「指令编辑需要原图」；req.Model 在 edit 分支忽略（model 字段是生图模型名不代表编辑引擎）②缺权重可操作提示：gaea 从不自动下载模型，ComfyUI 回 value_not_in_list 时错误追加三件文件名+HF 链接+存放目录（qwen_image_edit_2511_fp8mixed/qwen_2.5_vl_7b_fp8_scaled/qwen_image_vae 与 krea2 共用）——错误即安装指引，普通错误不追加③internal/app：GenerateMedia 在 edit+comfyui 时 imgModel 如实改写 qwen-image-edit（结果卡/台账记真实引擎）④前端 InstructionEditModal 说明改口（本地档已支持+缺权重行为），零交互变更。编辑链全继承既有基建：进度回调/取消/ComfyUI 自动拉起（v4.387）/落盘+台账登记零新增路径。【测试】ai 层改写拒绝测试为三件（全链 httptest 工作流形状：缩放图同进 TextEncode×2 与 VAEEncode+文件名与 type+AuraFlow→CFGNorm 链+KSampler 20/4.0/1.0 四路接线/缺原图/缺权重提示含 HF 链接且普通 500 不追加）+app 层 +1（edit+comfyui 元数据改写）+前端 +1（说明文案）；全量 395 文件 3347 例绿+tsc 0+eslint 0。【门禁】全量 ci 绿 EXIT=0+漂移闸 OK@4.392.0。【坑】①官方模板是新 subgraph 格式——节点图要从 definitions.subgraphs 展开+interface 槽位对射才能还原 API 图，直接读顶层 nodes 只有 4 个节点②CLIPLoader 的 clip_name 查的是 text_encoders 目录不是 clip（官方存放指引 models/text_encoders/）③编辑尺寸不能套请求 size——FluxKontextImageScale 按原图宽高比重标，width/height 在 edit 分支天然无效④提示文案锚 value_not_in_list+模型名双匹配，防普通错误误挂安装指引。【产物】exe 见 SHA256SUMS-v4.392.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.388~v4.392 删 v4.387.0.exe。【文档】releases/v4.392.0.md（含升级说明：ComfyUI 档编辑需先放模型文件，错误即指引）+CHANGELOG/README+releases/README（计数 413→414+34 席插 v4.392 裁 v4.358）+AGENTS 迁 1 插 1（一百零七迁：v4.387 入 archive）+progress/todos。

## v4.391.0 · 附件 URL 缓存 blob 化：base64 大字符串迁出 JS 堆 + selbar z 防御修（2026-09-22）
> 性能池⑦「base64→blob URL 会话资源管理」收口（v4.365 审计留池最后一项免拍板性能项）。usePortraitUrl 的 dataUrlCache=全局无界 Map，值是 MB 级 base64 data URL 字符串——消息内联图/缩略图/inspector/角色头像/剧照全部经此通道，长会话多图 JS 堆驻留可观。纯前端 3 文件+2 测试（改 1 新 1），绑定面 714 零变更。【落地】①缓存值 data URL→blob: object URL（atob→Uint8Array→Blob→createObjectURL，原 base64 字符串即被 GC——字节迁浏览器侧 blob 存储脱离 JS 堆；转换在既有 then 异步回调内；消费方零改动）②LRU 上限 96（Map 迭代序=LRU 序，命中 delete+set 刷新）+淘汰延迟 120s revoke（直接 revoke 会裂仍持有旧 URL 的在屏图，延迟给行卸载→重挂载→缓存 miss→重拉新条目留窗口）③无 createObjectURL/atob 环境回退 data URL 行为与旧版逐字节一致④selbar z-index 1080→990（v4.350 池⑧防御修——原值越过 antd 全部弹层，弹层内选中文本浮条盖弹层；990=内容上/fab 1000 与全部弹层下）⑤todos 考古（--whisper-ink-muted 已在中间轮次重指向 M3 secondary）。【测试】+1 文件 5 例（同路径一次桥/字节与 MIME 忠实/回退/LRU 淘汰+recency+延迟 revoke 恰一次）+改 1（PortraitImg 断言按新行为收口 blob:|data: 双形态——jsdom 30 原生 createObjectURL，blob 化在测试里真实生效）；全量 395 文件 3346 例绿（一轮 1 例无足迹 flaky 复跑全绿）+tsc 0+eslint 0。【门禁】全量 ci 绿 EXIT=0+漂移闸 OK@4.391.0。【坑】①jsdom 30 原生实现 createObjectURL（blob:nodedata:…）——「测试环境必然走 fallback」假设不成立，钉 data: 形态的既有断言真挂；降级路径测试要显式 defineProperty 摘除才触达②淘汰即 revoke 会裂在屏图——缓存条目被组件 state 持有，LRU 淘汰≠无人引用③data URL→blob 别用 fetch(dataURL)——jsdom fetch 不支持 data: scheme，atob 手写两边可控④后台测试命令只留 tail 丢失败详情——全量验证重定向完整日志再 grep。【产物】exe 见 SHA256SUMS-v4.391.0.txt（仅本地；冒烟 200 过）；保留策略 5 版留 v4.387~v4.391 删 v4.386.exe。【文档】性能池⑦销号回填+releases/v4.391.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 412→413+34 席插 v4.391 裁 v4.357）+AGENTS 迁 1 插 1（一百零六迁：v4.388 入 archive）+progress/todos。

## v4.390.0 · canvas 调色板集中化：resolveThemeColor 收口三份手写解析器 + 主题切换跟随缺陷根修（2026-09-22）
> 视觉池③+⑤「canvas 图表调色板集中 resolveToken」专项（v4.361 审计留池大工程候立项中可代码级收口的两项；①②④仍需逐板块截图对照拍板）。侦察发现三份手写 CSS 变量解析器各缺一块，且 RelationGraph 有一处**真缺陷**：角色色/组织色/关系色/画布底是**模块级顶层求值**——模块加载后主题切换永不重解析，亮暗切换后图谱停留在旧主题色（v4.350 只修了组件内 canvasColors，模块级常量漏网）。纯前端 7 文件+3 新测试，绑定面 714 零变更。【落地】①**集中解析器 resolveThemeColor/resolveThemeColorRGB**（utils/theme.ts）——span 探针法：颜色串写进临时 span 的 color 读 computed style，**浏览器自己算**天然支持 color-mix()/嵌套 var()/fallback 链；RGB 元组版供 canvas 派生半透明色拼 rgba()②**RelationGraph 接线+根修**——模块级求值收进组件 palette=useMemo(…,[darkMode])（与 v4.350 canvasColors 合并），graphData/decoratedNodes/edges memo 依赖补 palette——主题切换图谱全量调色板随动；图例色块改直接消费 var() 串（DOM 活解析天然级联）③**GraphView（记忆 3D 图谱）背景跟随**——探针解析（引用链拿计算色更稳）+MutationObserver 监听 documentElement 内联 style 变更重读背景推实例（backgroundColor 运行时可更新）；不引老栈 store 保持 gaea 与主应用仅经 CSS 变量耦合④**CompanionAvatar 接线**——删字符串手拆实现（不支持 color-mix；gaea 令牌链返回非 hex 串时 hex 后缀拼接拼出非法色被 canvas 静默忽略），改探针+withAlpha rgba() 拼接⑤**AoaView/PdmView 关键红兜底统一**——var(--sched-critical,#dc2626) 两处改 #e02020（令牌亮色定义值，导出白底场景消费），与 UsageView 一致。【测试】前端 +3 文件 9 例（theme：纯色短路/var/color-mix 探针/jsdom 空回退/非浏览器；RelationGraph：**darkMode 切换后画布底色重解析** fillStyle 先暗后亮+图例直消费 var 串；CompanionAvatar：探针→rgba 派生+纯 hex 默认路径）+目检（esbuild 挂具+Edge 无头暗/亮两页：两态节点+连线+图例全渲染，暗=深底浅字/亮=浅底深字——亮态走的就是重解析路径，改造前停留暗色）；全量 vitest 394 文件 3341 例绿+tsc 0+eslint 0。【门禁】全量 ci.ps1 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.390.0。【坑】①canvas 测试 jsdom 三连坑——offsetParent 恒 null 触发 v4.365 空转挂起（桩强制可见）/首绘走 rAF 调度（等帧 flush）/ctx 桩 Proxy 兜底任意方法（逐方法列举必漏：先漏 scale 后漏 createRadialGradient）②hex 后缀拼接只兼容 6 位 hex——遇 rgb() 串拼非法色被 canvas **静默忽略**不报错，派生透明度一律解析后拼 rgba()③模块级求值是主题切换隐形死角——canvas 类颜色要么进组件 memo（darkMode 依赖）要么 DOM 直消费 var() 活解析④getPropertyValue 读 custom property 只到引用替换层——span 探针读 .color 才是终值⑤Edge --screenshot 单帧模式不跑 rAF——canvas rAF 组件截图全空（DOM 正常），须加 --virtual-time-budget⑥gaea 侧主题跟随不引老栈 store——MutationObserver 监听 :root 内联 style 是零耦合跟随信号。【产物】exe 见 SHA256SUMS-v4.390.0.txt（exe 仅本地不入库；冒烟 /api/health 200 过）；保留策略 5 版留 v4.386~v4.390 删 v4.385.0.exe（SUMS 身份档案全保留）。【文档】releases/v4.390.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+发布说明计数 411→412+34 席索引插 v4.390 裁 v4.355）+AGENTS 迁 1 插 1（一百零五迁：v4.387 入 archive）+progress/todos。

## v4.389.0 · 模型中心统计重设计：右栏检查器遥测读出式 + 详细统计抽屉信息分层（2026-09-22）
> 用户指令「重新设计模型中心右侧面板的统计和详细统计」。病根两层：右栏「调用统计」=4 张 KPI 卡 2×2 平铺（300px 窄柱里又厚又挤、hint 常被省略号截断）+ 两张 720 宽 viewBox 全尺寸趋势图压进 ~276px 渲染（**坐标轴字缩到 ~4px 完全不可读**）；详细统计抽屉=块内标题与抽屉标题重复、5 张 KPI 图标全是同一个闪电、「本地 vs 云端」6 张卡两行平铺。纯前端 8 文件（含 eslint 配置 1）+2 新测试，绑定面 714 零变更。**落地**=①**右栏遥测读出式**：三格主指标（总调用·成功/失败/成功率阈值着色〔100%=成功色·<90%=危险色〕/估算费用）+ Token 紧凑读数行（k/M 缩写）+ 窄柱专用迷你图 RequestsSpark（面积+失败红点+末点+峰值读数）/TokenMiniBars（入/出堆叠+图例色片）+ 底部「查看详细统计」入口（context 新增 openStatsDrawer 动作，右栏直开抽屉）②**抽屉信息分层**：工具条（统计自 since/价格目录芯片+排序刷新清空，去重复标题）；「云端/本地分流」单卡叙事（Token 占比条+云端费用/本地用量/已节省三列+KV 命中率虚线紧凑行，替代 6 卡平铺）；KPI 五格语义图标（闪电/数字/付费/对勾/时钟）+Token hint 紧凑格式（旧全数字 hint 卡内截断）；图例类化（mc-chart-legend/mc-legend-swatch 与检查器共用）③**全尺寸两图可读性**：viewBox 720→540+轴字 10→11（抽屉双列 ~380px 栏宽有效字号 5.3→7.7px；两图现仅抽屉消费）④**eslint globalIgnores 补 .tmp**（esbuild 目检挂具 bundle 的 legal comments 触发指令错的门禁欠账，v4.358 .tmp 卫生守卫同族）。**测试**=前端 +2 文件 8 例（InspectorPanel：三格读数+迷你图+入口联动+空态/错误态；StatsSection：工具条芯片/KPI 着色+hint 紧凑/分流带占比口算+KV 行/空态）；modelcenter 17 套 121 例+全量 391 文件 3332 例绿+tsc 0+eslint 0。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.389.0。**坑**=①窄柱放全尺寸图=轴字缩到不可读——SVG width:100% 按视口比例缩，图表按目标渲染宽度设计 viewBox（迷你图 260/抽屉双列 540）②antd 两字中文按钮自动插空格（「重 试」）+图标 textContent 前导空格——断言按按钮存在性/trim 收口③目检挂具视口 <1180px 触发检查器隐藏媒体查询（截图全黑≠渲染失败），拍检查器要宽视口+隐藏抽屉④fmtCompact 归 utils.tsx 不进组件文件（react-refresh only-export-components 告警）。**产物**=exe 51181568B SHA256=ba43b3a1cecf79431de4739bc1deb918e621a1b5b98025f4d3c471bc0857693a（全文见 SHA256SUMS-v4.389.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致）；保留策略 5 版留 v4.385~v4.389 删 v4.384.0.exe（SUMS 身份档案全保留）。**文档**=releases/v4.389.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+发布说明计数 410→411+34 席索引插 v4.389 裁 v4.355）+AGENTS 迁 1 插 1（一百零四迁：v4.386 入 archive）+progress/todos。

## v4.388.0 · 原罪插图独立生图绑定：功能绑定卡新增「原罪插图」后端/模型选择（2026-09-22）
> 用户反馈「原罪的功能绑定卡片内增加生图模型选择，这么多模型为什么没有候选，只能使用 comfyui？」——答案两层：原罪生图此前**无独立绑定**跟随全局生图设置（本机全局=comfyui），模型中心原罪卡只绑**故事文本模型**；生图候选一直在（剧照卡同款 imageModelOptionsFor：xAI/ComfyUI 三件/引擎目录图像模型），只是原罪没入口。本轮镜像「角色库剧照」先例补上。Go 6 文件+前端 9 文件，绑定面 712→714（+2）。**落地**=①**GetSinImageConfig/SetSinImageConfig 绑定**：sin_image_backend/model 键（空=跟随全局，任一项可单独回退——只换模型不换后端也行）+shelf 热更新 case②**生成路由**：buildImageClientFor 从 buildPortraitClient 通用化（剧照改薄包装零行为变化，comfyui/herdsman/ollama/glm/xai 五后端独立客户端不改变绘梦全局）；generateImageInternal 加 clientOverride+backendOverride 通道——进度回调/size 裁剪/ComfyUI 自动恢复与自动拉起（v4.387）分支统一按生效后端判定；SinIllustrate 级联 sinImageBinding（绑定>全局）+sinRefPlan 参考槽判定用生效值（绑 Herdsman 按 Herdsman 判 img2img）+绑定后端≠全局时独立客户端（同后端复用全局客户端），sin_illustrate 工具同入口自动受益③**模型中心新增「原罪插图」卡**（镜像剧照卡）：后端下拉（跟随全局/xAI/ComfyUI/Herdsman/Ollama/GLM）+模型下拉（跨引擎候选：引擎目录图像模型过滤+ComfyUI 内置三件+当前值兜底）；原罪故事卡注释补指引。**测试**=Go +4 组（级联四态/Get 空与如实回读/buildImageClientFor 分支〔构建不拨号·禁用报错点名功能·地址缺失报错〕/override 通道〔override 调用=1 且全局客户端零触碰〕）+前端 +1（插图卡渲染+保存+已绑定态）。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.388.0+spaceBindings 锁 535→537+bindingNames 714。**坑**=①config.Save 直写真实 ~/.gaea_config.json 无测试隔离钩子——Set roundtrip 不能真调，测读映射与纯函数级联（抽 sinImageBinding）②种子引擎目录恒 Enabled（ollama/herdsman 带 BaseURL）——测未启用分支须先 SaveEngine 禁用（v4.386 语义坑同族）③override 客户端的后端语义要跟到底——进度回调/size 裁剪/自动拉起分支原来都读 a.cfg.ImageBackend，漏一处就是换了客户端没换行为的隐性错位。**产物**=exe 51171328B SHA256=48f1a288eaadb80a6a8632a81ac6d0dbff0fd110b87318947c4682a37bb141e1（全文见 SHA256SUMS-v4.388.0.txt，仅本地；冒烟 200 过；桌面副本同哈希实测一致——用户已关 gaea，本连跳两版直接更新到 v4.388）；保留策略 5 版留 v4.384~v4.388 删 v4.383.0.exe（SUMS 身份档案全保留）。**文档**=releases/v4.388.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+发布说明计数 409→410+34 席索引插 v4.388 裁 v4.354）+AGENTS 迁 1 插 1（一百零三迁：v4.385 入 archive）+progress/todos。

## v4.387.0 · ComfyUI 未运行自动拉起：原罪插图/绘梦生成连接被拒自愈（2026-09-22）
> 用户报障「原罪生图失败，插图失败：ComfyUI 提交失败…connectex: No connection could be made」。两层根因：环境层=ComfyUI 服务没跑且 standalone-env 落后于仓库 requirements（comfy-aimdo 0.4.13<0.5.5、comfy-kitchen 0.2.28<0.2.35 等，起都起不来）；产品层=gaea 生成链只对「孤儿实例 [Errno 22]」自动恢复，服务压根没跑时直接以 connectex 原始错误失败——原罪插图页无任何恢复入口（绘梦页才有启动按钮）。Go 1 文件+测试，绑定面 712 零变更。**落地**=①**ensureComfyUIRunning 助手+生成链接线**（image_handler.go）：生成失败且错误含「连接 ComfyUI 失败」（dial 层包裹文案，服务不可达才出现——服务在但卡死不进本分支防误杀）且 backend=comfyui 时，拉起+就绪轮询（/system_stats 同口径，有界 120s）成功后重试一次（comfyBooted 闸）；未配置路径/启动失败/超时一律返回 false，**原始错误如实上抛不吞错不换错**；并发拉起竞态由 StartComfyUI 既有端口占用守卫兜底（拿到「已被占用」转就绪等待不放弃）；原罪插图（sin_illustrate/SinIllustrate）与绘梦生成共用 generateImageInternal 一条链，一处接线双板块受益②**环境修复（用户本机随刀处置）**：按 requirements 钉版精确补 comfy-aimdo==0.5.5/kitchen 0.2.35/embedded-docs 0.5.11/frontend-package 1.52.7/workflow-templates 0.11.62（刻意不 pip install -r——torch 无钉版行会重解析，ROCm 特制 torch 可能被 PyPI 通用版顶掉）；修后以 gaea 同配方拉起 ~30s 就绪（ComfyUI 0.36.0），用户当前会话重试即恢复。**测试**=Go +3（image_boot_test.go 零真进程：连接被拒→假就绪端点→重试恰一次成功〔调用计数=2〕/未配置路径不重试且原始错误上抛/ensure 契约三分支）；internal/app 全量绿。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.387.0。**坑**=①ComfyUI 仓库更新不自动升级 python 环境——comfy-* 系包过期是一族的（先爆 aimdo storage 缺失、修完又爆 kitchen int8_attention 缺失），逐包补不如按钉版一次补齐；症状=起不来+import/attribute 错→对照 requirements 与 pip list②修 python 环境别 pip install -r③错误文案是重试路由的锚——新增错误分支先确认文案唯一锚定目标场景④测试成功路径必设 ImageSaveDir 指临时目录——否则 saveToNovelImages 撞装配态 nil。**产物**=exe 51163136B SHA256=f7ec8eb3c30260104a180beb81ba6354399aecc6bece6962c70bf9ba85a2e17b（全文见 SHA256SUMS-v4.387.0.txt，仅本地；冒烟 200 过；桌面副本因用户会话在跑本轮未覆盖——关闭 gaea 后下次发版统一替换）；保留策略 5 版留 v4.383~v4.387 删 v4.382.0.exe（SUMS 身份档案全保留）。**文档**=releases/v4.387.0.md（含升级说明：首次冷启动生图多等一段属预期）+CHANGELOG/README+releases/README（保留策略行+发布说明计数 408→409+34 席索引插 v4.387 裁 v4.353）+AGENTS 迁 1 插 1（一百零二迁：v4.384 入 archive）+progress/todos。

## v4.386.0 · 造价库分页绑定：成本条目列表/表格分页 + 检索下拉载荷降载（2026-09-22）
> 前端性能池⑥余项收口（聊天历史分页 v4.371 已落、造价库全表留池至今）：成本库「成本条目」每次搜索/筛选全表拉回全量渲染——用户真实库 1553 条时每次 250ms 防抖击键=全量桥载荷+1553 行 DOM。Go 3 文件+前端 10 文件，绑定面 711→712（+1）。**三件**=①**GaeaCostSearchPage 分页绑定**（App 层实现，cost.Store 零改动）：检索管线（SQL 过滤→Go 关键词过滤→语义召回→本地精排）提为 costSearchAll 助手，原 GaeaCostSearch 与新绑定共用——分页与非分页口径一致、工具面 cost_search 零影响；排序进服务端 sortKey∈title/price/updatedAt（空=管线序）**tie-break 恒定 name**（全序=跨页翻页不漂移的前提）；limit 钳 [1,200]（≤0→100）、offset 负数归零、Total=过滤后总数；未知 sortKey 不排序保持管线序（防前端传错键）②**CostLibraryView 分页改造**：首屏 100 条+尾部「加载更多（余 N 条）」100 条/批，计数如实「已载 X / 共 Y 条」载完收口，分类树「全部条目」计数用 total；**客户端排序删除**（分页时代只排已载子集是静默错误答案）——表格排序点击改服务端排序重拉第 1 页，箭头语义不变；过期响应按 reqSeq 序号丢弃+追加页跨页 name 去重（翻页期间数据增删的偏移漂移兜底）③**EntryPicker 切分页绑定**：测算项目「单价搜索下拉」只要 top 8，此前全表拉回客户端截断，现 CostSearchPage(…,8,0) 载荷从全表降到 8 行；loadStats 总览仍用全量（聚合统计需要，开页一次非热路径）；mock 同口径实现分页。**测试**=Go +5（分页切片连续无缝+total/排序全序 price 同价 name tie-break+降序=升序精确镜像/钳制四态 limit≤0→100·>200→200·offset<0→0·越界空页 total 仍在/零命中诚实空页/未知 sortKey 管线序/status 过滤×updatedAt 组合——SQL 直铺确定时间序）+前端 +1（150 条首屏 100+加载更多补齐+计数收口+按钮消失）+迁移断言（分类过滤 7 参形态/表格排序异步等待）。**门禁**=全量 vitest 389 文件 3323 例绿+tsc 0+全量 ci 绿 EXIT=0（E 系列守卫 OK+仓库卫生守卫 OK）+版本漂移闸 OK@4.386.0+spaceBindings 锁 534→535。**坑**=①**App 层检索类测试必须断死语义通道**——modelengine.NewManager 种子目录 herdsman 恒 Enabled+localhost:8080，本机 embedding 服务在跑时零命中查询（SQL 召回<3 触发语义补召回）走**真实**管线：Stale("cost",keep) 按测试条目集清掉真实库 cost 向量行+Ensure 写入假向量（本轮开发中实际发生——真实库 1553 条 cost 向量被清，已用应用自身 GaeaSemanticIndexBackfill 通道全量重建 1553/1553=24s，测试侧改 SaveEngine 禁用 herdsman 断通道）；hubCostStore/hubSemanticStore 的 override 注入只护住一半，resolveHerdsmanSearchModel 的引擎目录在测试环境是活的②App 嵌 *core 指针——裸 &App{} 的 a.engineMgr 字段访问即 nil-core panic，测试构造须 &App{core:&core{…}}（既有先例 consistency_deep_handler_test）③分页绑定的排序必须全序（name 收尾 tie-break）；同理客户端排序在分页时代必须删除而非保留④updated_at 落盘 RFC3339 秒级——同一测试内连发保存回读后全同秒，按保存顺序断言时间序不成立，要确定时间序用 SQL 直铺。**产物**=exe 51160576B SHA256=4201619c820784789af1d77452eaf48169af70dedddf25f32024156f54b458cd（全文见 SHA256SUMS-v4.386.0.txt，exe 仅本地不入库；桌面副本同哈希实测一致；冒烟 /api/health 200 过）；保留策略 5 版留 v4.382~v4.386 删 v4.381.0.exe（SUMS 身份档案全保留）。**文档**=性能池⑥销号回填+releases/v4.386.0.md（含升级说明）+CHANGELOG/README+releases/README（保留策略行+发布说明计数 407→408+34 席索引插 v4.386 裁 v4.352）+AGENTS 迁 1 插 1（一百零一迁：v4.383 入 archive）+progress/todos。

## v4.385.0 · 工程健康轮 #2：excelize 防线扩面 + 死代码清欠 + 双扫描报告（2026-09-22）
> 距 v4.370 工程健康轮已 15 版，按惯例复扫三面。Go 2 文件+前端 4 文件，绑定面 711 零变更。**扫描报告**=govulncheck 1 项在案（GO-2026-6452 excelize 负 shared-string 下标 panic，上游 Fixed in N/A）/pnpm audit 0/knip 1 未用导出+3 未用类型+3 重复导出+5 配置提示。**两刀**=①**excelize 防线扩面**：v4.370 只防了 xlsxpreview 入口，本轮符号追踪多出三条未防御可达面——schedule.ImportXlsx（GetRows）、GaeaXlsxRowOps（InsertRows/RemoveRow）、GaeaXlsxColOps（InsertCols/RemoveCol）——同 v4.370 配方 named-return+defer recover 转可读错误（「文档结构异常」），panic 不再外溢②**knip 死代码清欠 4 项**（tsc+引用 grep 双验证）：useModelCenter（v4.352 拆分后兼容口零消费，stale 注释同步更新）+桥接死类型 3 个（ForeshadowUrgencyPayload/NovelRewriteRequest/SinBookSourceDownloadResult）；重复导出 3 项与配置提示 5 项为既有 KEEP/低价值 churn 不动。**测试**=app/schedule 包全量绿（纯增量 defer 零行为变化）+tsc 0+modelcenter 112 例绿。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.385.0。**坑**=①「已防御」≠「全防御」——符号追踪随代码演化增长（v4.370 后新增的 xlsx 编辑/导入引入新可达面），扫描要周期性复跑②Fixed in N/A 的上游漏洞是常驻项——新增 excelize 消费面必须同步配 recover 入口③knip 未用导出要区分死代码与契约哨兵——重复导出是既有 KEEP 形态不跟风删。**产物**=exe 见 SHA256SUMS-v4.385.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.381~v4.385 删 v4.380.exe。**文档**=releases/v4.385.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 406→407+34 席插 v4.385 裁 v4.351）+AGENTS 迁 1 插 1（一百迁：v4.382 入 archive）+progress/todos。

## v4.384.0 · dsh 蒸馏第五弹：session_query 会话检索（跨会话记忆首刀）（2026-09-22）
> dsh 留池⑧「session-query 检索」首刀落地（余 4→3）：过往会话全文 FTS5 索引+session_search 工具——模型可检索本工作区过往会话的 user/assistant 正文，与 memory 事实库互补（memory 存提炼事实，本工具查原始对话）。全 Go，绑定面 711 零变更。**两件套**=①**索引器**（新包 internal/gaea/sessionquery）：索引面=装配空间会话目录的 *.gaea-log.jsonl 中 user_message/assistant_message 正文（system/工具结果/流式 delta 大噪声面不入索引）；增量按 (size,mtime) 新鲜度指纹（session_index_meta 表）——搜索时惰性刷新零 boot 成本；重索引=先 DELETE 该文件旧行再全量插（FTS5 标准表普通 DELETE 可用）；空间隔离由目录构造（Store 绑装配空间会话目录，双空间红线同构）②**session_search 工具**（只读）：query 必填+limit（默认 8 上限 32），newest-first 带 session id/role/时间/snippet 高亮；**中文降级**=FTS5 unicode61 不切 CJK 子串，MATCH 零命中回退 LIKE '%q%' 子串扫描+手工裁窗摘要（对齐 whisper FTS5 先例）；MATCH 短语引号化（内嵌引号双写转义）。**测试**=Go +5（FTS 命中+非消息 kind 不入索引+ts 倒序/CJK 子串 LIKE 降级/增量新鲜度/目录隔离/工具面格式化+空查询+零命中诚实）。**门禁**=go build/vet 0+gofmt（触碰文件）+全量 ci 绿+版本漂移闸 OK@4.384.0。**坑**=①ReadLogRepaired 的 torn-tail 修复会截断「未以换行收尾」的最后一行——手工构造日志必须换行收尾（真实 LogWriter 恒换行收尾）②FTS5 标准表（非 contentless）才支持普通 DELETE——UNINDEXED 元数据列建虚表上免联表可重索引③LIKE 路径 snippet 是手工裁窗——两路径 Scan 形状不同，共用 Scan 助手会静默拿到空原文（首写即犯测试当场抓出）。**产物**=exe 见 SHA256SUMS-v4.384.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.380~v4.384 删 v4.379.exe。**文档**=dsh 蒸馏文档⑧首刀销号回填+releases/v4.384.0.md（含升级说明：首检索全量索引一次）+CHANGELOG/README+releases/README（计数 405→406+34 席插 v4.384 裁 v4.350）+AGENTS 迁 1 插 1（九十九迁：v4.381 入 archive）+progress/todos。

## v4.383.0 · 聊天虚拟化第二层：绘制层 content-visibility + 挂池考古销号一批（2026-09-22）
> v4.365 池②「聊天消息真虚拟化」收口（两形态分层）+ 一批过时挂池条目考古销号（零代码）。前端 2 文件+测试 +2，绑定面 711 零变更。**刀1 绘制层虚拟化**：聊天长历史成本分两层——渲染层尾部窗口 v4.370/371 已落（>50 条渲染最近 80 条+顶部渐进扩载分页衔接），本刀补绘制层：超阈值会话根节点挂 chat-flow-cv，行级 content-visibility:auto 离屏行跳过 layout/paint，窗口扩载到全量历史后深滚动依旧顺滑；contain-intrinsic-size:auto 120px 让浏览器记住行实际高度，滚动条估算与滚动锚定不受影响；≤阈值不挂类零影响；流式吸底不受影响（流式行恒在屏内）；复制/朗读菜单走 antd portal 不受 containment 影响；**为什么不是 react-window**（v4.370 侦察否决有案）=JS 窗口化要求滚动容器即虚拟列表根+行绝对定位，与本页滚动宿主（ChatPage listRef）/动态 Markdown 行高/流式吸底三冲突，CSS 原生方案零结构改动兑现同目标。**挂池考古销号（零代码）**=v4.355②角色库 file:// 头像 27 条（v4.361 已清账，AttachmentDataURL 通道注释在案）/v4.355③longtask 阈值（v4.361 已提 200ms 在案）/v4.349⑤initRuntimePolyfill 同步成本（辨伪≈0 纯对象装配）+vendor manualChunks（无新证据，P4 懒加载已 −37%）。**测试**=前端 +2（>阈值挂类/≤阈值不挂——类挂载即完整契约）；全量 vitest 389 文件 3322 例绿+tsc 0。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.383.0。**坑**=①content-visibility 的行内 fixed/portal 子元素语义要先查——containment 裁剪 fixed 后代，聊天行无（菜单 portal 到 body），novel selbar 不在作用面②「真虚拟化」不等于 react-window——动态行高+流式吸底+外部滚动宿主三冲突下，渲染层窗口+绘制层 CV 分层以零结构风险达成同目标；池条目按「目标」销号不按「实现手段」。**产物**=exe 见 SHA256SUMS-v4.383.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.379~v4.383 删 v4.378.exe。**文档**=池②收口+三条考古销号回填+releases/v4.383.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 404→405+34 席插 v4.383 裁 v4.349）+AGENTS 迁 1 插 1（九十八迁：v4.380 入 archive）+progress/todos。

## v4.382.0 · 前端挂池清欠：整店订阅清零 + 交付物缓存失效接线（2026-09-22）
> 两个小欠账一并清偿：v4.349 挂池④「reducer 级整店订阅」收口销号 + v4.354 记录的 dirListingsCache 失效接线欠账。前端 4 文件+测试，绑定面 711 零变更。**两刀**=①**无 selector 整店订阅清零**：全仓扫描四家 store（appStore/scheduleStore/outlineStore/previewStore）仅剩 2 处无 selector 整店订阅——WorkspacePanel（novelsDir/setNovelsDir）与 FilePreviewModal（previewFile/closeFilePreview/openFilePreview）全部 selector 化；审计对账=v4.349 审计「13 处（MainLayout:414/HomePage:33/AppearancePanel×8）」行号已过时——MainLayout/HomePage/AppearancePanel 已被中间轮次（v4.352⑦ 等）selector 化，本轮清零最后余量后全仓归零；gaea 控制器 useController() 的 useShallow(s=>s) 整面订阅保留（App.tsx 消费近乎全量字段无法局部化，热字段 items 流式本就逐 chunk 变化 selector 化无收益，架构性拆分留专项）②**交付物缓存失效接线**（v4.354 记录欠账）：deliverablesTurn.ts 的 registryCache/dirListingsCache（轮交付物登记+目录存在性探测）此前只有 TTL 与测试用 invalidateTurnCaches，「轮完成/会话切换」失效时机一直是预留注释——接线两处：turn_done 事件消费点（ensureEventsBound）+loadSessionData（会话切换/恢复）；新轮交付物与轮内新建文件不再被 TTL 内旧目录探测误标缺失，会话切换后登记与探测随会话整包刷新；纯内存清空零桥接成本。**测试**=前端 +2（失效接线：turn_done 后 ensureTurnRegistry 重拉打桥的调用计数断言/会话装载路径同语义）；全量 vitest 389 文件 3320 例绿+tsc 0。**门禁**=全量 ci 绿 EXIT=0+版本漂移闸 OK@4.382.0。**坑**=①挂池审计行号会过时——清欠前先全仓重扫，中间轮次已顺手修掉大头，销号要写清哪些本轮修哪些中间轮已修②恰好一次的事件绑定是模块级单例——失效接线没有重新绑定的测试入口，从可观测面（bridge 调用计数）断言行为不 mock 内部③缓存类欠账的失效时机在事件边界——reducer 保持纯函数，失效副作用接在事件消费/装载的既存非纯区。**产物**=exe 见 SHA256SUMS-v4.382.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.378~v4.382 删 v4.377.exe。**文档**=挂池④/dirListingsCache 销号回填+releases/v4.382.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 403→404+34 席插 v4.382 裁 v4.348）+AGENTS 迁 1 插 1（九十七迁：v4.379 入 archive）+progress/todos。

## v4.381.0 · dsh 蒸馏第四弹：锚定式 token 计量（2026-09-22）
> dsh 留池②锚定式 token 计量（packages/llm/token-meter）单独一版精修——compression 域最后一块精度拼图。全 Go，绑定面 711 零变更。**病根**：mid-turn 压力=全量字符×全局比率，比率漂一点全表面跟着漂，压缩触发时机（high-water）漂移=早触发白砸缓存前缀/晚触发靠溢出自愈兜底。**落地**（internal/gaea/agent/tokenmeter.go）：压力=上次真实 usage 锚点+表面有符号差值；per-message 估价 memo（FNV-64=role+content+toolcalls，与 msgChars 同记账面，ReasoningContent 不入账）让差值只统计新增/删除/改写——未变消息锚点估价逐项相消，剪枝/压缩重写后估计不再漂移；落锚点=maybeCompact 收到真实 usage 时（此刻会话表面恰=刚被 answered 的请求，新 assistant/tool 结果未入账，锚点无系统偏差；LastPrompt 回退路径不落锚）；估压有锚=锚点+差值（负差值钳非负），无锚=裸字符估算旧口径不变（midTurn 本就等首锚后才工作）；压缩重写性质=老消息被摘要替换时压力按「删锚点估价、加新估价」精确移动。恒装配无配置面。**测试**=Go +5（比率源漂移下未变消息不重算+新增移除精确移动+重复读取零漂移/压缩重写精确移动/负差值钳/无锚回退/maybeCompact 真实 usage 落锚而回退路径不落）。**门禁**=go build/vet 0+gofmt（触碰文件）+全量 ci 绿+版本漂移闸 OK@4.381.0。**坑**=①锚点时点对齐「刚被 answered 的请求面」——maybeCompact 在 session.Add(assistant) 前调用才无系统偏差，挪后差一整轮②新消息按当前比率估、老消息按锚点时点比率消是刻意的（memo 键只含内容不含比率，比率演化不影响一致性）③sync.Mutex 不可重入——持锁调内部无锁变体，别再走公开方法。**产物**=exe 见 SHA256SUMS-v4.381.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.377~v4.381 删 v4.376.exe。**文档**=dsh 蒸馏文档②销号回填（余 4）+releases/v4.381.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 402→403+34 席插 v4.381 裁 v4.347）+AGENTS 迁 1 插 1（九十六迁：v4.378 入 archive）+progress/todos。

## v4.380.0 · dsh 蒸馏第三弹·子代理编排：Fork 型子代理 + 结构化输出补完 + V10.36 对齐复活（2026-09-22）
> dsh 留池再进两刀（余 5）：委派双向增强凑成「子代理编排」一组。全 Go，绑定面 711 零变更；顺带根修一个被测试当场抓出的既有缺陷。**两刀+一根修**=①**Fork 型子代理**（源 subagent-fork-in-process）：task 新增 fork 参数——子代理以父会话完整历史做种子开跑（带上下文委派，模型不再手工粘前文）；种子自带父 system 首消息不注模板提示→子请求前缀与父请求逐字节一致=KV 缓存直接继承（fork 第一轮近乎免费）；TaskTool 实现 ContextualTool（父历史经 executeOne 盖章注入）；组合禁忌=background/continue_from/retry_until；runSubSession 会话持有上移（fork 种子与 guard 重入共用）②**结构化输出补完**（源 outputSchema/两阶段捕获/terminal guard）：原 output_schema 只在父级事后验、子代理根本不知道要产 JSON——三件补完：schema 注入子代理 prompt（[OUTPUT FORMAT] 指令）+ terminal guard（终答非 JSON 同会话纠偏重入，热前缀缓存成本极低，有界 2 次，仍不合法诚实降级诊断前缀+原样文本）+ stripJSONFences 剥 markdown 围栏③**根修 V10.36 对齐复活**：P0-1 每轮重置 activeSchemas 无条件清 nil，把构造期注入的父对齐 schema 第一轮就抹掉——对齐自重置加入起是死代码（fork 测试抓出：子请求 schema 1 vs 父 2）；修法=New 存构造期基线 baseSchemas、回合重置恢复基线（主代理基线 nil 行为不变）；两个 V6.0 测试钉的是死形态，按复活后设计改写为「请求目录=父注册表全量（缓存对齐）+执行面过滤由 buildSubReg 承担」双断言（一层委派红线不破：子调未授权工具得 unknown tool）。**测试**=Go +6（fork 种子字节同源+缓存形状/组合禁忌四态；schema 注入/guard 恢复含剥围栏/guard 耗尽诚实降级/stripJSONFences 三态）+改写 2（V6.0 双断言重构）。**门禁**=go build/vet 0+gofmt（触碰文件）+全量 ci 绿+版本漂移闸 OK@4.380.0。**坑**=①「每轮重置」类清理会把构造期注入配置一起抹掉——重置要回基线不是零值，死代码靠端到端断言请求形状才能辨识②V6.0/V10.36 两代测试钉矛盾行为各活各的（对齐是死的两边都绿）——修复一处另一处断言爆开，改断言不是改实现③RunSubAgent 把会话藏在内部——凡需同会话二次进入/会话预置的功能都要把会话持有上移调用方④executeOne 把工具结果包信封 JSON——测试透信封断言别假设裸文本。**产物**=exe 见 SHA256SUMS-v4.380.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.376~v4.380 删 v4.375.exe。**文档**=dsh 蒸馏文档 fork/结构化输出销号回填（余 5）+releases/v4.380.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 401→402+34 席插 v4.380 裁 v4.346）+AGENTS 迁 1 插 1（九十五迁：v4.377 入 archive）+progress/todos。

## v4.379.0 · dsh 蒸馏第二弹：spill 泄洪 + 中断流结构化块保全（2026-09-21）
> dsh 留池再进两刀（余 7 项）。全 Go，绑定面 711 零变更（read_spill 是 agent 工具目录新增非 App 绑定）。**两刀**=①**spill 主动泄洪**（源 packages/spill/spill-policy）：成功工具结果 ≥24KB 全文先存会话泄洪库（新叶子包 internal/gaea/spill，runner 内存+单条 4MB/总量 64MB FIFO 驱逐，重启清零即天然会话级），内联文本前置 locator，新工具 read_spill(id,offset,max_bytes) 分页取回——大结果中段不再被截断销毁，经济性从压缩时被动归档提前到执行时主动泄洪；**捕获在压缩前**（gaea 按工具压缩 bash 24KB+全局 48KB 截断会先销毁中段，泄洪保原文、预览仍交既有压缩管线）；**locator 前置=截断生存性结论**（信封是单行紧凑 JSON，超帽时全行保留但首行被头部裁切——尾缀必死、头部必活，TestSpillNoticeSurvivesTruncation 当场抓出尾缀假绿）；豁免 read_file（分页+缓存，上游 read 豁免同义）/read_spill/task；best-effort 纪律照单全收（泄洪失败绝不把成功变错误或藏内联结果）；[agent] tool_spill 配置门默认开，子代理经 taskTool.SetSpill 随父各持独立库②**中断流结构化块保全**（源 agent-loop step catch）：终态流错误/预算阻断路径 assistant（含 calls）落库——此前只存文本，模型已声明的调用从历史消失；assistant(tool_calls) 必须成对补 tool 结果行（否则下次请求非法历史形态 provider 拒绝），新增 persistInterruptedCalls：流式预执行只读工具有真实结果用真实，其余合成「stream interrupted: received but not executed」占位（WrapError 信封同约定），ToolResult 事件照发收口前端半开卡片；**恢复路径刻意不落 calls**（重试采样路径落了就是悬空非法形态）——两路径分流是关键辨析；真零内容不落库纪律不变。**测试**=Go +12（spill 包 4：分页/剩余/FIFO 驱逐/超限拒收/ctx 往返；agent 5：泄洪全文锚点/豁免与开关/截断生存性/read_spill 端到端跨回合保序/未知 id 报错；中断保全 3：成对落库+悬空检查/预执行真实结果/恢复不落 calls）。**门禁**=go build/vet 0+gofmt（触碰文件）+全量 ci 绿+版本漂移闸 OK@4.379.0。**坑**=①信封单行 JSON 截断按行选择按字节裁切——行数不超限时全行保留后首行头部裁切，要活的内容放头部（对所有「结果内嵌指引」机制成立）②混合批执行序非调用序（v4.375 分区并行不同资源键共存）——「先泄洪再取回」端到端测试同批会竞态假红，跨回合才有全序③agent 测试引 builtin + builtin 引 agent=测试期 import cycle（生产编译不报 go test 才炸）——共享类型下沉叶子包 internal/gaea/spill④preOutcomes 存已包信封 output——合成占位也走同信封约定否则回放两种形态混存。**产物**=exe 见 SHA256SUMS-v4.379.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.375~v4.379 删 v4.374.exe。**文档**=dsh 蒸馏文档①③销号回填+releases/v4.379.0.md（含升级说明）+CHANGELOG/README+releases/README（计数 400→401+34 席插 v4.379 裁 v4.345）+AGENTS 迁 1 插 1（九十四迁：v4.376 入 archive）+progress/todos。

## v4.378.0 · Reasonix 真身蒸馏第四弹·留池收官：bash 会话状态锚定 + memory activation 二维 + subject keys（2026-09-21）
> Reasonix（esengine/DeepSeek-Reasonix v1.15→v1.38）真身蒸馏第四弹收官版，全 Go，绑定面 711 零变更。三刀落地后蒸馏档留池 9 项全部销号（落地 8+辨伪并入既有），真身蒸馏线收官；同日对账 dsh 0.1.6 留池销号 3 项（12→9：timeout-policy 域隔离=gaea 工具超时是工具内错误字符串无外层 deadline 误判通道/技能目录 digest 重发布=缓存稳定前缀同判/contextBreakdown=ContextView catSystem·catTools 构成视图已在）。**三刀**=①**bash 会话状态锚定**（留池⑤主值，源 persistentshell）：cd/export（含导出函数）跨调用存活——**取道不取器：进程不持久，状态持久**（上游 PTY+marker+分帧与 gaea 每调用新进程+Job Object 灭树+boundedOutput+超时硬杀处处相克且 Windows 需 conpty）；漂移治理在「观测」不在「推断」（盲缓存前缀注入的病根）——EXIT trap 采集终态：cwd 经 cygpath -w 转 host 形（MSYS $PWD 的 /c/... 形 exec.Cmd.Dir 不认）、env 用 export -p 全量转储下次前置 eval 重放（转储即真相），显式 exit 照采且退出码 exit $__gaea_rc 原样传播；被杀（超时/取消硬杀 trap 不跑）/trap 被覆盖/exec 换身不采纳——缓存永不超前于现实；并行批次独立探针文件完成序采纳；锚定目录消失自愈回落工作空间；豁免=PowerShell 壳/WSL enforce/run_in_background（起始目录仍锚定）；会话切换经新接口 tool.SessionStateResetter 由 controller（NewSession/Resume）整体重置防跨会话泄漏②**memory activation 二维**（留池⑦之一，源 memory/activation.go）：activation 管事实「怎么触达」与 scope 正交——pinned=正文随会话快照装配/relevant=仅检索可达；BuildPinnedFactsBlock 固化正文预算化注入缓存稳定前缀（块 1200 rune/单条 400 截断留标），记忆面板「固化」动作从此有功能后果；排序 Name 升序非 recency（块 ride 稳定前缀 touch 不能翻动）；门控仅记忆总开关，两空间各注入各的收窄视图；空集合零注入③**memory subject keys**（留池⑦之二，源 remember/store_v2）：remember 新增可选 subject_key 点名单值问题（project.package_manager/user.response_style），同空间同键仅一条活跃值——撞键保存**拒绝并点名持有者**（修订走原条目重写非制造自相矛盾新记忆），同名重存=修订放行，置空=释放键；Store.Save 统一写入侧检查（remember/面板/做梦全经此，dream 不带键零影响）；SchemaV23 facts.subject_key 列+双 SELECT 回填；文件后端 metadata.subject_key 往返（仅声明时写出=旧文件逐字节不变）。**测试**=Go +15（shellstate 9：cwd/env 端到端/显式 exit 采纳+退出码传播/保守四态/合法采纳/消失自愈/reset 整树删/nil 锚/Resetter 契约；activation 3；subject keys 4；装配 1：pinned 正文双空间各见各的+未固化不进+开关关零注入）；Git Bash 真机验证 trap 采集+eval 重放。**门禁**=go build/vet 0+gofmt（触碰文件）+全量 ci 绿+版本漂移闸 OK@4.378.0。**坑**=①bash printf 的 %s 是 Go Sprintf 动词——wrapper 模板写 %%s 否则 vet 拒编②MSYS 路径形态——探针必须 cygpath -w 转 host 形否则下次 cmd.Dir chdir 失败③事件日志列与 Event 结构一一对应——subject_key 真相归 facts 表（与 body 同待遇日志只留摘要），不为审计单独开列④NormalizeSubjectKey 空白折叠与下划线不是一回事——键等价只走 trim+ASCII 小写+空白折叠'-'一条路。**产物**=exe 见 SHA256SUMS-v4.378.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.374~v4.378 删 v4.373.exe。**文档**=蒸馏文档第四弹三刀+留池⑤⑦收官回填（池清空）+dsh 档销号 3 项（12→9）+releases/v4.378.0.md+CHANGELOG/README+releases/README（计数 399→400+34 席插 v4.378 裁 v4.344）+AGENTS 迁 1 插 1（九十三迁：v4.375 入 archive）+progress/todos（reasonix 行翻✅收官）。

## v4.377.0 · 办公记忆/成本库写入纠偏：自动做梦建议制 + 存量清污 + 成本劝写改口（2026-09-21）
> 用户反馈「不管什么都自己记入记忆和成本库，根本不询问，大量错误污染数据」——主源=auto-dream（每轮 TurnDone 后台 LLM 提炼直写长期记忆+项目文档，设计上豁免审批且记忆开关管不住）；次源=成本工具文案持续劝写。绑定面 707→711（+4 dream 开关/忽略/清污×2）。**五刀**=①**自动做梦建议制**（主刀）：新增 [dream] mode（suggest 默认|auto 旧直写|off）；suggest 下提炼结果进待确认队列（dream-pending.json，FIFO 上限 30）记忆面板「建议」页逐条**接受才入库**（source=explicit）/忽略即丢弃；notes 同样建议化（play notes 不入队同旧纪律）；异步无法弹卡的旧解法「放行直写」改为「确认从写入时移到接受时」（DREAM_WRITE_POLICY §5 修订——「写入内容低风险」前提被实测推翻）；**记忆开关收口**（enabled=false 任何模式不整理，旧实现只关注入）②**存量清污**：GaeaDreamPurgePreview 返回 auto_dream 审计∩现存事实名（explicit 接受过与手工沉淀不在内），GaeaDreamPurge 仅删该交集（绑定面不可删任意记忆），dream_purge 审计留痕③**成本劝写改口**：cost_save 描述改「仅用户明确要求时调用」，cost_search/cost_compose 返回文案 4 处+compact 简述 2 行同步（work 审批卡原样保留，改的是发起频率）④**默认 system prompt 记忆段改口**：「只在用户明确要求记住时才调用 remember」；自定义提示词不受默认值影响需自查⑤**面板 UI**：记忆面板「自动做梦」三态循环按钮+建议卡片忽略按钮+「清理自动做梦写入」两步确认入口+i18n 三语+dev mock 补 4 绑定。**测试**=Go +8 用例（DreamMode 归一化/队列 FIFO 上限出队三态空间隔离/入队分流 work 全入 play notes 不入/视图转换/接受入库出队幂等+忽略不写库/清污交集圈定交集外跳过审计留痕）+前端 MemoryPanel +3（三态循环/忽略/清污两步）。**门禁**=go build/vet 0+全量 ci 绿+版本漂移闸 OK@4.377.0。**坑**=①bindingNames.ts 重生成必须保 as const——丢失则 BindingName 退化 string、drift 锁报 never（报错不指向真凶先查数组尾）②jsdom LocaleProvider 默认 en——i18n 断言须双语正则③出队失败不得静默落旧写入路径（否则未确认建议入库+队列残留）④清污按钮不能只在建议非空分支渲染。**产物**=exe + SHA256 见 SHA256SUMS-v4.377.0.txt（仅本地；冒烟 200 过）；保留策略留 v4.373~v4.377 删 v4.372.exe。**文档**=docs/DREAM_WRITE_POLICY.md §5 修订+releases/v4.377.0.md（含升级说明：恢复旧行为/清污指引）+CHANGELOG/README+releases/README（计数 398→399+34 席插 v4.377 裁 v4.343）+AGENTS 迁 1 插 1（九十二迁：v4.374 入 archive）+progress/todos。

## v4.376.0 · Reasonix 真身蒸馏第三弹：压缩救援阶梯 + 重复调用裁决 + 技能目录预算化（2026-09-21）
> 续 v4.375 真身留池第三弹。全 Go，绑定面 707 零变更。**三刀**=①**压缩救援阶梯**（源 fold_ladder/truncate/rescueOrFail）：1a slim 摘要档——折叠区大到来不及重放时（溢出场景），摘要请求自己 400→退机械裸计数；估算超 window−预留−5% 边际即逐消息截头（下限 200 字符+标记，工具参数截 160）换有界转录，全量档被 provider 真实拒绝后重试也降半预算 slim；fits 时逐字重放不变（缓存对齐仍是首选档）。1b 投影截断终级 truncateRescue——溢出自愈的剪枝+压缩无进展**或压缩后估算仍在窗口之上**（上游 at-or-above-ceiling）时：保护区收窄 target/4（planCompaction ≥2000 token 尾钳会把可救内容全划成圣地）→抹大工具结果（老→新，KeepErrors/KeepProtected 豁免）→整单元丢最老+装 marker；改动前先归档（archive 坏=整体拒绝 fail-closed）②**重复调用硬阻断降级 advisory**（上游裁决：硬阻断误伤多于收益，全量退役；gaea 观察期 v4.375→v4.376 满）：repeatedSuccessBlock 退役，同签名写工具第 3 次成功起批后注入一次性合成提醒（每签名每轮至多一条），真实结果不再被 blocked 替换，Success 语义更真实③**skill 目录预算化渲染**（源 skill catalog 二分压缩）：目录超 4000 字符先二分收缩描述宽度保**全员可见**（尾部技能被截=模型永远用不到），压缩态尾注说明计入预算；纯名行仍超才退旧硬截断；fitting 目录逐字不变。**辨伪销号一批**=采样恢复状态机+预算学习（中断/溢出两路已在册；超限/thinking-400/预算学习无靶子——主链路不发 max_tokens、reasoning 不回放）、skill 引用按需分页（read_skill 即按需）、watcher 热重载（必破缓存稳定前缀）、read_tasks 续读游标（read_file 已有 offset/limit）、会话私有临时目录 env 重定向（Git Bash TMP 注入牵动 /tmp 映射）、压缩状态跨重启保留（重启丢熔断=天然解锁）。余池=Persistent bash PTY、memory 事实生命周期。**测试**=Go +13（slim 截头/参数封顶/请求收缩/fits 逐字/拒绝后降档/rescue 抹结果丢单装 marker/KeepErrors 豁免/三态无靶/溢出落到截断终级/advisory 越线一次性+改钉两条/目录压缩保全员/fitting 零变更）+2 处适配（summarize 加 forceSlim 形参）。**门禁**=go build/vet 0+全量 ci 绿 EXIT=0+版本漂移闸 OK@4.376.0。**坑**=①救援机制的可达带要先算清再写测试——rescue 可达带=compact 尾钳与 rescue 尾钳之间盲区，大块 assistant 正文是 prune 够不到、compact 尾保护又留下的内容，E2E 须用此类内容构造否则前置清场测试假绿②token 估算器是 max(bytes/4, runes/2)——纯 ASCII 按 2 字符/token 计，边界按此口径算③破坏性截断必须 fail-closed 于归档——三级（prune/compact/rescue）都先归档再动手，archive 坏=整体拒绝。**产物**=exe 51030528 B SHA256=8c0febb68f936ed247a317baa6bb86d714474e9f96b9a2c50a3724f7d59b4aef（SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.372~v4.376 删 v4.371.exe。**文档**=蒸馏文档第三弹三刀+留池销号回填+releases/v4.376.0.md+CHANGELOG/README+releases/README（计数 397→398+34 席插 v4.376 裁 v4.342）+AGENTS 迁 1 插 1（九十一迁：v4.373 入 archive）+progress/todos。

## v4.375.0 · Reasonix 真身蒸馏第二弹：执行序批次 + 文件观察 + 杂项加固（2026-09-21）
> 续 v4.374 真身留池第二弹。全 Go，绑定面 707 零变更。**四刀**=①**执行序批次修正**（源 v1.38 Harness-style scheduling）：批次分区冲突键 read:/file: 分家导致同批「先读 A 再改 A」并发竞态——统一 file:<path> 资源键同路径拆批保序，跨路径共存并行延迟收益保留（取道不取器：gaea 冲突键模型已编码资源隔离，不取上游全序严格执行以免退 v4.63 并行子代理）+call/result 对应性守卫（双重 recover 间逃逸路径合成结构化错误，回放永不悬空）②**Live file observations**（源 fileops/observation.go）：会话级版本指纹（size+mtime 纳秒）——read_file 真实读后记录、写前比对、自身写后刷新；外部修改/删除 blocked:[stale version]、从未观察不拦、缓存命中不假装观察；与 V10.28 stale-anchor 并存（锚点新鲜度 vs 跨轮外部篡改——此前 write_file 会静默覆盖外部修改）③**git 硬化基线**（源 gitcmd）：gitRun 注入 -c fsmonitor/maintenance 关闭+env GIT_TERMINAL_PROMPT=0/GIT_OPTIONAL_LOCKS=0/GIT_CONFIG_NOSYSTEM=1，GaeaGitDiff 强制 --no-ext-diff --no-textconv④**memory_search 低权威免责**（源 auto_recall）：检索结果前置免责声明防旧记忆覆盖新指令；路径抹除辨伪销号（gaea 记忆不含机器路径）。**测试**=Go +5（保序/共存回归锁成对/观察三态）+键名适配。**门禁**=go build/vet 0+全量 ci 绿+版本三处 4.375.0。**坑**=①批次分区改动必须带共存收益回归锁（只测拆批不测共存=延迟优化静默回退）②观察类机制防「假装观察过」——观察点必须绑真实 I/O 成功路径（缓存命中记录当前版本反而掩盖过期）。**产物**=exe 51011072 B SHA256=88499e001577a49b184f8e754a1f1a879e38c13e3affe86dab9572e48c3e09a6（SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.371~v4.375 删 v4.370.exe。**文档**=蒸馏文档留池四项销号/辨伪回填+releases/v4.375.0.md+CHANGELOG/README+releases/README（计数 396→397+34 席插 v4.375 裁 v4.341）+AGENTS 迁 1 插 1（九十迁：v4.372 入 archive）+progress/todos。

## v4.374.0 · Reasonix 真身蒸馏：安全四刀 + 内核三刀（2026-09-21）
> 用户纠偏「蒸馏对象认错了，跑去看 dsh」+「自己去 github 上搜索」——真身=GitHub esengine/DeepSeek-Reasonix（Go 单二进制终端 agent，gaea 注释 V1.12/V1.15 端口出处），本版蒸馏 v1.15→v1.38.11 增量（5227 提交）；v4.373 的 dsh 文档勘误更名 gaea-dsh-016-distill-2026-09.md（五刀本身有效保留）。全 Go，绑定面 707 零变更。**安全四刀**=①**Host 白名单**（HTTP 421，源 serve/hostguard）：loopback 服务被 DNS-rebinding 变同源后可无预检驱动 RPC——Host 头比对 loopback+监听地址名单拒绝；httpbridge Handler 默认挂、ServeWithToken 按监听地址派生（通配豁免）、app 诊断端口（/healthz、/stack 栈倾倒）同挂②**只读表修正**（源 shellsafe/bash_approval）：移除 env/awk/sed（间接执行与可执行 shell 的程序体 fail-closed 交审批；上游 awk -f 例外依赖「已记忆规则」模型、gaea 自动放行无此前提故不照抄）、cargo 仅留 search（check/doc 执行 build.rs）、find 补 -ok/-okdir、git 补 --ext-diff/--textconv/cat-file --filters/grep --open-files-in-pager③**git clean-filter 中和**（源 gitcmd）：diff 是唯一在工作树内容上跑 clean filter 的子命令而 .gitattributes 不可信——gaea_git 枚举 repo 本地 filter.* 注入 -c 置空三件套（恒等直通 diff 仍真实，非 repo fail-open）④**SSRF 共享件+booksource 补洞**（源 installsource parity）：判据提升 netclient.BlockedSensitiveIP/BlockedInternalIP/GuardedClient（webfetch 委托、loopback 放行语义不变），booksource 默认 client 从裸 http.DefaultClient 换守卫版（书源规则可被诱导注入摸内网/云元数据 169.254.169.254/100.100.100.200）。**内核三刀**=⑤**收尾判定**（源 v1.37 Deterministic natural-turn completion）：reasoning-only 干净 stop=最终回答（不再强制可见文本、不再注 nudge 重试，上游实测重试无收益）；真零内容=冻结请求原样重放不注入合成 user 消息（缓存前缀零污染），上限 3 不变超限如实报错⑥**前台输出运行中内存上限**（源 shellrun/bounded）：bash 输出 Write 即封顶 head 1MiB+滚动尾 64KiB+截断标记——`yes`/失控构建不再能打爆内存（与退出后截断本质不同）⑦**compactStuck 新消息解锁**（源 stuckInputHash）：熔断绑定会话形状 stuckAtMessages，新消息=新折叠边界自动解锁重试（旧熔断会话级永久），mid-turn 不再提前短路。**测试**=Go +10（HostGuard 10 子例+桥 421/BlockedInternalIP+GuardedClient 拒 loopback/cleanFilterOverrides 真仓两路/间接执行 20 例/reasoning-only final/零内容冻结重放/stuck 两态/boundedOutput 三例）；行为变更适配 3 处（零值 fetcher 改钉守卫拒绝语义、output_continue_test 裁剪至 length-nudge、TestDownloadChaptersExplicitList 补显式 client）。**门禁**=go build/vet 0+全量 ci 绿 EXIT=0+版本漂移闸 OK@4.374.0+E 系列+卫生守卫 OK。**坑**=①蒸馏判据带上「上游该机制的前提在本地是否成立」——-f 例外在 gaea 自动放行模型下不成立②安全刀落地先全仓 grep 假站/httptest 用例的 client 来源——TestDownloadChaptersExplicitList 未注入 Fetcher 走默认通道被守卫打碎。**产物**=exe 51004928 B SHA256=85149421d0098b19dc28fbf66e4a0c7a5e52281fdf86b1e97b8ef35d9aed0a90（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.370~v4.374 删 v4.369.exe。**文档**=docs/gaea-reasonix-v138-distill-2026-09.md 新增（真身七刀+留池 9 项+否定结论）+docs/gaea-dsh-016-distill-2026-09.md 勘误更名+releases/v4.374.0.md+CHANGELOG/README+releases/README（计数 395→396+34 席插 v4.374 裁 v4.340）+AGENTS 迁 1 插 1（八十九迁：v4.371 入 archive）+progress/todos。

## v4.373.0 · Reasonix (dsh) 0.1.6 内核蒸馏：溢出自愈 + 缓存对齐摘要等五刀（2026-09-21）
> 用户口径「继续优化迭代 gaea，本次优化进行蒸馏最新版的 reasonix」——对 /c/AI/deepseek-harness（dsh v0.1.6-alpha.2，上游同步 2026-09-19）三域全量侦察后落地五刀。全 Go 内核，绑定面 707 零变更。**落地**=①**上下文溢出自愈**（compaction-basic 蒸馏）：provider.IsContextOverflow 七措辞分类器（绝不误判输出侧 max_tokens）命中后剪枝→强制压缩→rewrite version 验持久进展→有界重试；采样成功清零，连续上限 1，无进展如实放弃——「上下文超限」从回合死刑变可恢复路径②**缓存对齐摘要**（summarizer 蒸馏）：摘要请求=会话真实 system+被折叠区间逐字原样重放+指令作末条 user 消息（动态注入只进末条），成上一真实请求真前缀命中热 KV 缓存，按 cache-read 计价不再全价重付；renderTranscript 平铺退役③**shrink 硬校验**（region.ts 蒸馏）：摘要估算必须小于被折叠区间，否则退机械摘要——压缩永不膨胀④**force 多轮压缩**（compactionRetries 蒸馏）：强制水位压缩后重测仍超再压一轮，rewrite version 只计真实落地，空转不计入 compactStuck 熔断⑤**repeat 分级提醒**（repeat-tool-reminder 蒸馏）：[3,5,8] 三档递进（3 通用/5 点名工具+参数预览 500 上限/8 终止指令），4/6/7/9+ 沉默，deny 也计数，用户插话重置链。**测试**=Go +7（IsContextOverflow 9 子例/自愈四路/摘要请求前缀逐字稳定/shrink 两路/force 有界/repeat 三档）；行为变更适配 summaryInstruction 快照断言。**门禁**=go build/vet 0+agent/provider 全包绿+全量 ci 绿+drift OK@707+版本三处 4.373.0。**坑**=①折叠区含旧 digest 时逐字重放到第一个 kept 消息即分歧——system+tools 最贵前缀仍全命中，部分对齐可接受（完全对齐会重摘要旧 digest，破坏不可变设计）②无进展判定必须看 session.RewriteVersion 而非 compact 返回值——compact 对无可折叠区返回 nil，按返回值判会把空转当成功。**产物**=exe 50995712 B SHA256=363b40e7450e95a6a9089bfe7d3900c9edbb791fb50a2fc163b0098c0f29ac30（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.369~v4.373 删 v4.368.exe。**文档**=docs/gaea-reasonix-dsh016-kernel-distill-2026-09.md（侦察+五刀+留池 12 项）+releases/v4.373.0.md+CHANGELOG/README+releases/README（计数 394→395+34 席插 v4.373 裁 v4.339）+AGENTS 迁 1 插 1（八十八迁：v4.370 入 archive）+progress/todos。


## v4.372.0 · Gantt 缩放性能：Ctrl+滚轮 rAF 合并（2026-09-21）
> 用户口径「继续」——性能留池「Gantt 行组件化」可安全子集落地：Ctrl+滚轮缩放 wheel 事件 rAF 合并（一帧最多一次 setDayW，终态一致）。schedule 1 源文件+1 测试适配，绑定面 707 不动。**落地**=高分辨率触控板每秒 60~120 次 wheel 事件原各触发一次 setDayW 全三窗格重渲染；改 rAF 合并——帧内累积缩放因子 pendingW，每帧最多提交一次 setDayWAt；锚点以累积后目标日宽换算，滚动位置恢复与逐事件处理终态一致；清理路径补 cancelAnimationFrame。「GanttBars 行组件化+dayW 走 CSS transform」结构性改造留池（条形定位改 scale 变换涉及拖拽坐标换算重写，核心交互高风险区需专门拖拽真机回归）。**测试**=GanttView 33 例全绿（缩放断言等 rAF 帧提交——jsdom 下 rAF 为 16ms 模拟 setTimeout，fireEvent 后同步断言拿旧值）+drag.test 7 例绿。**门禁**=全量 ci 绿 EXIT=0（vitest 3315 例；首跑 BookHealthPanel 1 例 flaky 零足迹交叠单跑复绿后全量复跑 OK）+drift OK@707+版本三处 4.372.0。**坑**=①rAF 合并后同步断言失效——测试需等待帧提交②it 回调补 async 时连环括号错误——脚本化改写测试优先整段重写。**产物**=exe 50988032B SHA256=cf94c096112e9c177f6e46f040fe315acf82e5ebc3fb5b4a765ee78057bab7b8（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.368~v4.372 删 v4.367.exe。**文档**=releases/v4.372.0.md+CHANGELOG/README+releases/README（计数 393→394+34 席插 v4.372 裁 v4.338）+AGENTS 迁 1 插 1（八十七迁：v4.369 入 archive）+progress/todos（Gantt 缩放 rAF 合并销号）。

## v4.371.0 · 聊天历史分页绑定（游标式，绑定面 706→707）（2026-09-21）
> 用户口径「继续」——性能留池「聊天历史分页绑定」设计落地：大话题切会话不再全量拉取，首屏最新 200 条+向上翻页游标续拉。Go 3 源文件+1 测试+前端 6 源文件。**契约**=store ListMessagesPage(topicID, limit, beforeSeq)（beforeSeq<=0 从最新向前取；返回升序+hasMore 取 limit+1 探测）/App ChatMessagesPage 返回 {messages, hasMore, oldestSeq}/ChatB 门面+gen_bindings 再生（合计 707 方法→11 门面）+bindingNames/spaceBindings(shared) 手工同步+spaceBindings.test 数量 529→530。**前端**=loadTopic 首屏 ChatMessagesPage(id,200,0) 记录 hasMore/oldestSeq+loadOlder（120 条/批 prepend 头部）/MessageList 接 hasOlder·loadingOlder·onLoadOlder·prepend 四 props——头部追加用 prepend 通知扩大窗口而非重置（anchor 重置逻辑区分「分页追加」与「切话题替换」，防用户被弹回尾部）；全量 ChatMessagesList 保留（导出场景）。**测试**=定向 +1（Go TestListMessagesPage：首屏截断/hasMore/游标续拉/拉到底）+ChatPage.test mock 接线 15 例全绿。**门禁**=go build/vet 0+internal/chat 全量绿+tsc/eslint 0+全量 ci 绿（vitest 3315 例）+drift OK@707+版本三处 4.371.0。**坑**=①头部追加 vs 整体替换窗口语义——anchor 重置不区分会把用户弹回尾部②mock 契约跟随签名演进——数组 vs page 对象，page?.messages 提取静默空。**产物**=exe 50987520B SHA256=59cfc300918fd55521a58ad8619884ecf0441795cda15a81fd84b607da19a200（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.367~v4.371 删 v4.366.exe。**文档**=releases/v4.371.0.md+CHANGELOG/README+releases/README（计数 392→393+34 席插 v4.371 裁 v4.337）+AGENTS 迁 1 插 1（八十六迁：v4.368 入 archive）+progress/todos（性能八项消一项：分页绑定收官）。

## v4.370.0 · 工程健康轮：依赖漏洞清零 + Go 工具链升级 + 构建管线修正（2026-09-21）
> 用户口径「继续」——十轮快跑后工程健康整固：govulncheck+npm audit 双扫描按发现修复。Go 1 源文件+依赖升级+构建脚本修正，绑定面 706 不动。**扫描与处置**=①Go 标准库 5 处（net url/tls/http/xml/asn1）→工具链 1.26.5→1.26.6（go.mod toolchain 指令）②x/image v0.40→v0.46（webp/bmp 解码 panic 4 处，用户可控输入面）③excelize GO-2026-6452 恶意 xlsx panic（上游 Fixed in N/A）→应用层防线：xlsxpreview Render/NeedsRecalc 入口 defer recover 转「预览失败（文档结构异常）」错误，govulncheck 复扫 11→1 仅剩此条④npm @vitest/mocker 2 moderate（dev-only）→vitest 4.1.11，audit 归零⑤**新增 pnpm-lock.yaml**（前端依赖历史上从未锁定）。**构建管线修正**=build.bat wails 命令未带 -s，隐式 npm install 在 lockfile 缺失+peer 冲突（eslint-plugin-prettier）时爆炸（本轮 vitest 升级触碰 package.json 即暴露）——按在册坑配方改先 npm.cmd run build 再 wails build -s。**门禁**=govulncheck 复扫 11→1+audit 归零+go build/vet 0+office 全量绿+全量 ci 绿（vitest 388 文件 3315 例，4.1.11 兼容）+drift OK@706+版本三处 4.370.0。**坑**=①pnpm 项目 npm install 生成 package-lock 双轨——立即删并 pnpm install 恢复单轨②lockfile 缺失是环境敏感炸弹③wails 隐式 npm install 依赖环境解析成功——显式前端构建+-s 是在册坑配方 build.bat 此前未遵守。**产物**=exe 50979328B SHA256=785a740a681d410dfff9c8744d09cb37698ca6d756e0808f31b42edbae8de95f（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过；go1.26.6 构建）；保留策略留 v4.366~v4.370 删 v4.365.exe。**文档**=releases/v4.370.0.md+CHANGELOG/README+releases/README（计数 391→392+34 席插 v4.370 裁 v4.336）+AGENTS 迁 1 插 1（八十五迁：v4.367 入 archive）+progress/todos（excelize 上游未修留观）。

## v4.369.0 · 真机走查班第二班：流式分段渲染真机验收 + --w-ink-3 重指向（2026-09-21）
> 用户口径「继续」——走查班第二班：v4.366~368 三轮真机验收；顺手落地别名重指向小刀 --w-ink-3。前端 1 css 文件，绑定面 706 不动。**走查结论**=①流式分段渲染链路真机验证通过：新建会话→发短消息→模型正常回复落库渲染，终态无错误无 file:// 残留无 console 告警（分段切分契约由 findStableCut 6 例单测锁定）②--w-ink-3 重指向目检通过：launcher 局部令牌实测 #5b6472=text-tertiary（别名重指向两目收官）③环境事实：走查观察到的「请求超时」历史消息系用户环境模型服务当时不可用，非本轮回归。**落地**=module-launcher.css --w-ink-3 从 color-mix(text 46%) 改 var(--md-sys-color-text-tertiary)，.ml-avatar-user 二次混色底 12%→14% 对齐观感。**清场**=测试消息对经 chat.db 按 id 精准删除（会话恢复原有 6 条）；脚本/截图清理；杀壳。**门禁**=tsc/eslint 0+全量 ci 绿（vitest 3315 例）+drift OK@706+版本三处 4.369.0。**坑**=①CDP 读局部令牌要在定义作用域元素上读（--w-ink-3 在 .ml 下，documentElement 读恒空）②走查发消息会写用户真实库——聊天无消息级删除，测试消息只能 chat.db 按 id 精准删；后续验证流式先建专用测试会话再删整会话。**产物**=exe 50951680B SHA256=26f0293b4b08e03f70d9220093872c3eff335ed72642116d341ec6029fbdfc50（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.365~v4.369 删 v4.364.exe。**文档**=releases/v4.369.0.md+CHANGELOG/README+releases/README（计数 390→391+34 席插 v4.369 裁 v4.335）+AGENTS 迁 1 插 1（八十四迁：v4.366 入 archive）+progress/todos（别名重指向两目收官）。

## v4.368.0 · 性能轮深水区：聊天流式分段渲染 O(n²) 根修（2026-09-21）
> 用户口径「继续」——性能留池最大项落地：聊天流式每 delta 对全量累积文本重跑 react-markdown（O(n²)，5KB 尾段 5~20ms×30+ chunk/s）。移植 gaea 侧已验证「稳定分段」设计到聊天流式行，每 chunk 实际重解析只剩尾段。前端 3 源文件+1 新测试，绑定面 706 不动。**方案**=三选一定 A（ChatRow 流式分支分段，影响面锁死一处；方案 B 动公共组件冲击调用计数断言+波及非流式方；限频只降频率不降规模）。**落地**=①findStableCut 导出+盲区修复（原只扫切点后 suffix，fence body 含空行计数偶数误判可切拦腰切断；改全量行扫描 fence 状态，切点在未闭合 fence 内回退到打开行前，切分更保守稳定段永不含悬挂 fence；既有 MemoMarkdown 测试零改动通过）②ChatRow 流式分支 stable+pending 两段渲染（stable 段字符串不变时子树被 memo 整体跳过；pending 每帧重解析通常 KB 级；终态回既有路径零变化）。**契约**=genui 零破坏（fence 当普通 fence 闭合落稳定段；流式分支本就不传 overrides）；已知接受折衷=松散列表流式序号短暂重排终态自愈+streaming 翻转 DOM 重建一次（与 gaea 同款）；改进=尾段走完整管线，未闭合 fence 流式中显示为持续增长代码块更接近终态。**门禁**=定向 +6（findStableCut 含盲区修复用例）+MemoMarkdown 既有零改动通过+genui/聊天回归全绿+tsc/eslint 0+全量 ci 绿（vitest 3315 例）+drift OK@706+版本三处 4.368.0。**坑**=①正则整段替换函数体非贪婪匹配会吞闭合括号——替换串忘了带 }，tsc 当场暴露②性能改造行为等价是「终态等价」——流式中间态 DOM 从单块变两块，验收口径=终态一致+中间态不劣化，不能拿中间态 DOM 逐字节比。**产物**=exe 50951680B SHA256=e1bc4358584f6c0c70dacb2c9ac18cd545dd831f8dc6157a9b3d88b7cf2849a0（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.364~v4.368 删 v4.363.exe。**文档**=releases/v4.368.0.md+CHANGELOG/README+releases/README（计数 389→390+34 席插 v4.368 裁 v4.334）+AGENTS 迁 1 插 1（八十三迁：v4.365 入 archive）+progress/todos（性能八项消一项：流式分段渲染收官）。

## v4.367.0 · 编辑输入性能：章节编辑 memo 化 + 设定页预览/统计防抖（2026-09-21）
> 用户口径「继续」——性能轮延续：编辑输入链路每击键全页重渲染与全文重解析治理，全部行为等价。前端 4 源文件+1 新 hook，绑定面 706 不动。**①ChapterEditor memo+updateTab useCallback**=章节编辑每击键两次 setTabs 使 959 行 ChapterPage 整页重渲染、未 memo 的编辑区全场景框陪跑→updateTab 改 useCallback（依赖 activeKey）稳定 onUpdate 引用+ChapterEditor 包 React.memo（父页无关 state 不再重渲染编辑区）。**②ChapterPage 上报防抖**=novel:chapter-active 事件 effect 依赖 activeTab（每击键变）每次全文 join+countTextChars 并拖动壳层 inspector 重渲染→250ms 防抖停顿后上报一次。**③NovelSettingPage 预览/字数防抖**=分屏/渲染模式每击键全文 react-markdown+KaTeX 重解析（10k 字约 10~30ms/键）+wordCount 每键全文扫描→新增 useDebouncedValue 通用 hook（300ms），MarkdownContent 与字数消费防抖镜像值，编辑器保持逐键受控输入零延迟。**门禁**=tsc/eslint 0+全量 ci 绿（vitest 3309 例；ChapterPage/NovelSettingPage/novel 187 例定向全绿）+drift OK@706+版本三处 4.367.0。**坑**=①跨层字符串转义地狱——CDP/python 多层传递 \n 被吃成真实换行写进源码（TS Unterminated string literal），构造含反斜杠字面量用 chr(92) 拼接②防抖正确姿势是镜像值而非延迟源——编辑器逐键受控输入零延迟，只有下游重计算消费防抖镜像；防抖放 onChange 上游=输入回显延迟是 bug 不是优化。**产物**=exe 50951680B SHA256=177331f52125c531ba4096e7e2e121cc1e653083d77912ebce6d935d2be8ac38（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.363~v4.367 删 v4.362.exe。**文档**=releases/v4.367.0.md+CHANGELOG/README+releases/README（计数 388→389+34 席插 v4.367 裁 v4.333）+AGENTS 迁 1 插 1（八十二迁：v4.364 入 archive）+progress/todos（性能八项再消两目——ChapterEditor 受控下沉被 memo+防抖等效覆盖大半，下沉本体留池）。

## v4.366.0 · 性能轮收官：聊天组件 memo 化专项 + StoryStream 行级 memo（2026-09-21）
> 用户口径「继续」：性能八项留池中「聊天组件 memo 化」props 链梳理完成落地收官；StoryStream 行级 memo 同步根修（O(消息数×分段) 流式放大）。前端 7 源文件+1 守卫适配，绑定面 706 不动。**刀1 聊天组件 memo 化**=ChatPage 五个内联箭头回调 useCallback 化+内联 [] 提 EMPTY_REPLIES 常量+四组件（ChatModeBar/ChatPersonaBar/ChatComposer/ChatInspector）React.memo 包裹（XInner+文件尾 memo 导出）；ChatInspector 统计三遍全文遍历合并单趟 useMemo。memo 语义安全：props 梳理不完整只损失收益无正确性风险。**刀2 StoryStream 行级 memo**=抽 SinAssistantRow memo 行组件（parseStorySegments 下沉行内）——前提核实：useSinStory 流式 patch 非流式行引用稳定+插图回调 useCallback 稳定；流式期间每 delta 只重渲染流式中的那一行。**守卫适配**=E16 QUICK_REPLIES 作用域断言按 memo 化声明形态适配（意图不变）。**门禁**=tsc/eslint 0+全量 ci 绿（vitest 3309 例+E16 PASS）+drift OK@706+版本三处 4.366.0。**坑**=①改组件导出形态先查源断言守卫（E16 按声明文本定位）②memo 化顺序纪律——先稳定 props 再包 memo，反了空转无报错。**产物**=exe 50951168B SHA256=a6d7be9a917339e4d010ceb17fa3b10c594f4225f845a0f9e3b2fbaa222b10c1（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.362~v4.366 删 v4.361.exe。**文档**=releases/v4.366.0.md+CHANGELOG/README+releases/README（计数 387→388+34 席插 v4.366 裁 v4.332）+AGENTS 迁 1 插 1（八十一迁：v4.363 入 archive）+progress/todos（性能八项消两项）。

## v4.365.0 · 前端性能轮：流式节流 + 后台空转治理 + 缓存统一（2026-09-21）
> 用户口径「继续」——前端换新镜头性能审计（三路：React 渲染/长列表流式/桥接数据面），全落行为等价低风险刀，前端 22 源文件+1 测试，绑定面 706 不动。**线1 流式节流**=useChatStream delta 改 buffer+rAF flush（原每 chunk 一次 setState 使 ChatPage 30~120/s 全页重渲染+吸底 reflow；四条终态路径 flush 前 cancelPendingDelta）+角色打字机 rAF 时间驱动（节奏不变）+ChatPanel 打字机 rAF 每帧 3 字符。**线2 后台空转治理**=五个 canvas 循环（ParticleFlow/SoundWaveOverlay/CompanionAvatar/VoiceChatOrb/RelationGraph）加 document.hidden||offsetParent===null 挂起守卫/DagPanel·AgentTree·useComfyTaskProgress·useImageGenQueue·SinIllustration 轮询秒表 hidden 跳过/TaskCenter onTaskEvent 接 gate/AIConsole 隐藏丢事件。**线3 渲染链**=MainLayout（根壳）/HomePage/ModelCenterPage/AppearancePanel×8 整 store 订阅改逐字段选择器（任意 set() 不再波及全部 keepAlive 页）/MemoryHubPage hits key 稳定化。**线4 缓存统一**=usePortraitUrl 导出 getCachedAttachmentDataURL（共享缓存+方法缺失防御），Message/FileThumb/inspector 三处直调接线；粘贴/截图复用内存 dataUrl 作预览（省第二次全量回读）。**线5 AOA 几何**=nodeById/anchorById/taskNameById Map 化（原每边线性扫）+几何管线整段组件级 useMemo（缩放/选中不再全量重算），hooks 全部移 early return 前。**顺手修真 bug**=AIConsole 违反 v4.62.2 EventsOff 事故纪律（改 subscribeWailsEvent 唯一入口）/TaskCenter 事件通道 gate 漏接。**门禁**=定向 +4 源断言（perf-guards）+tsc/eslint 0+全量 ci 绿（vitest 387 文件 3309 例）+drift OK@706+版本三处 4.365.0。**留池**=流式分段渲染/真虚拟化/StoryStream 行级 memo/ChapterEditor 受控下沉/Gantt 行组件化/分页绑定/blob URL/聊天组件 memo 化。**坑**=①组件 memo 非免费午餐——内联 props 使 memo 恒失效，须先梳 props 链②rules-of-hooks 硬约束——提升 useMemo 连同 manual/shown 移 early return 前③测试 mock 缺方法时可选链与直接调语义差——共享工具要方法缺失防御④整 store 订阅代价被 keepAlive 放大——根壳粗订阅优先治理。**产物**=exe 50951168B SHA256=c2d09a937841630a46052517ea750cd16ac975d26e242d026c06f26296653948（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.361~v4.365 删 v4.360.exe。**文档**=releases/v4.365.0.md+CHANGELOG/README+releases/README（计数 386→387+34 席插 v4.365 裁 v4.331）+AGENTS 迁 1 插 1（八十迁：v4.362 入 archive）+progress/todos（性能大工程八项挂池）。

## v4.364.0 · 真机走查班：v4.361~v4.363 前端三轮验收 + withinReadRoots 绘梦图误伤修复（2026-09-21）
> 用户口径「继续」（授权闲置窗口）——CDP 9333 附着 v4.363.0 壳走查（只读纪律），走查拽出 v4.358 读侧收口真机才现形的误伤，修复发版。Go 1 源文件+1 测试，绑定面 706 不动。**走查结论**=双空间 14 板块（工作 6+闲庭 7+设置/进度计划直达）错误边界 0/空白 0/console.error 0/exception 0/file:// img 残留 0；深验=①聊天 PersonaPicker popover 31 可聊天角色 26 头像全渲染零裂图零 file://（v4.355 报 27 条告警同场景，收口确证）②schedule 行号色实测 rgb(74,64,96)=Violet 亮态 onSurfaceVariant（v4.361 生效）③tertiary 双态 #5b6472↔#8b93a0 切换正确④longtask 走查全程仅 1 条 800ms 真长任务（阈值 200ms 后噪音归零）。**修复 withinReadRoots 遗漏图片保存目录**=走查日志 6 条 AttachmentDataURLError（Pictures\gaea 绘梦历史图被拒）——v4.358 论断「绘梦资产均落两根内」不成立：绘梦图实际落 cfg.ImageSaveDir（默认 %USERPROFILE%\Pictures\gaea）；修=withinReadRoots 改 App 方法补第三根（ImageSaveDir 非空时+默认兜底，与 image_handler 落盘口径一致）；真机复验同一张被拒图成功读回 2.89MB data URL；定向 +1 TestWithinReadRootsCoversImageSaveDir（配置根+兜底根纯前缀断言不写用户目录）；顺手修=方法化后 &App{} 裸构造既有测试 panic（nil core 解引用）加 a.core!=nil 守卫。**门禁**=定向 +1+app 全量绿 102s+tsc/eslint 0+全量 ci 绿（vitest 3305 例）+drift OK@706+版本三处 4.364.0。**坑**=①mock/单测测不出路径语义——v4.358 白名单论断没经真机就写注释，读侧白名单变更必须真机过真实数据路径②走查拽错要看壳日志不只看页面（AttachmentDataURL 失败前端有占位兜底页面全绿）③包级函数改方法先 grep 裸构造测试（&App{} 合法惯用形，方法化解引用 nil 嵌入指针即 panic）④CDP 探针传 Windows 路径转义陷阱（多层引号反斜杠被吃 + 成 formfeed），用 join(String.fromCharCode(92)) 或正斜杠。**产物**=exe 50948096B SHA256=e026857b8c74d1667b9d5606267d93bffd94d03a3678008073deca4d4f76b976（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.360~v4.364 删 v4.359.exe。**文档**=releases/v4.364.0.md+CHANGELOG/README+releases/README（计数 385→386+34 席插 v4.364 裁 v4.330）+AGENTS 迁 1 插 1（七十九迁：v4.361 入 archive）+progress/todos。

## v4.363.0 · 后端留池终审：ConvertToPdf 读侧收口落地 + prompt 双构造辨伪关闭（2026-09-21）
> 用户口径「继续」：后端挂池仅剩 2 项（均标「需设计」）逐项终审——一项设计落地、一项证据辨伪关闭，**后端挂池清零**。Go 1 源文件+1 测试+前端 1 小刀，绑定面 706 不动。**刀1 ConvertToPdf 读侧收口**（挂池设计项落地）=取证确认三类调用方（工作区相对/uploads 绝对/对话框工作区外素材）全部被现有机制覆盖——收口=①相对路径 Clean 拒 .. 穿越（原 Join Clean 掉 .. 可逃逸）②统一门 `!withinReadRoots && !isPickedFile` fail-closed（对话框素材经 GaeaPickFiles 登记时已入表，isPickedFile 现成覆盖；对齐 GaeaReadFileB64 口径）；写侧 exports 全服务端拼装无越界；三类合法调用方零误伤；前端 exportPdf 已有 toast 零改动；定向 +1 TestConvertToPdfPathGuard（四场景零外部进程——放行断言借 .doc 扩展名拒绝分支，.docx 在装了 soffice 的测试机会真转成功反而无法断言）。**刀2 prompt 双构造辨伪关闭**=复核「双构造」实为 New() 构造后立即被 SetPromptFS 换新（main.go:32→34 同步单线程在 Startup/httpbridge 前），第一实例零读者无正确性影响；换新是 embed FS 注入的唯一无锁手段（Engine 显式无锁契约不支持事后注入）；收益=个位数毫秒一次性 vs 成本=改 New 签名或加锁+绑定生成面三处——风险>收益辨伪关闭，日后低风险形态=New 可选 embed-FS 参数+SetPromptFS 幂等短路排池尾。**刀3 前端小刀**=MemoryHubPage 总览读取失败 rail 计数「—」占位（原静默空缺）。**门禁**=定向 +1+internal/app 全量绿+tsc/eslint 0+全量 ci 绿（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.363.0。**坑**=①收口测试放行断言不能借真实转换（.docx 装了 soffice 会真转成功断言失效），借转换前确定性分支②辨伪关闭也要证据链写进留档下次不必重查。**产物**=exe 50947584B SHA256=5a2bdafa10858b913df40604d9428be5adb3969dfeae8205eecf274a5b68b0fa（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.359~v4.363 删 v4.358.exe。**文档**=releases/v4.363.0.md+CHANGELOG/README+releases/README（计数 384→385+34 席插 v4.363 裁 v4.329）+AGENTS 迁 1 插 1（七十八迁：v4.360 入 archive）+progress/todos（后端挂池清零）。

## v4.362.0 · 前端优化轮第二弹：吞错可见化收尾 + tertiary 文本令牌收敛（2026-09-21）
> 用户口径「继续」：v4.361 余量再取证定刀，纯前端 17 源文件+2 测试+3 语言文件，绑定面 706 不动。**刀线一 吞错可见化收尾 12 刀**=①OfficePanel 读失败防覆盖（置顶数据覆盖风险——gaeaSettings 失败草稿留全默认值，点保存把默认覆盖真实引擎 config.toml；loadFailed+Alert 重试+禁保存+入口拦截，定向 +2）②「删除假成功」家族 5 刀（PriceSourcesRepository/PriceSourcesPanel〔+加载失败空面板错误态〕/CostProjectsView 删明细行/CostProjectsView restoreVersion 旧行删除失败中止恢复〔原吞错继续插新行=删半截+重复〕/CostNotesView）③OfficeMemoryLibrary doMergeAll 串行计数分报失败对保留可重试④useChatTopics 初始化失败一次性 message.error（原只进后端日志侧栏空白像历史全丢）⑤ProgrammingPage 状态轮询首败 setError 复用 prog-error 横幅（原永久「检测中…」假加载）⑥ModelSwitcher 失败态「点击重试」与「未配置」分流⑦TaskInboxPanel 状态变更失败提示⑧小刀 3 处（ChatPanel 人格清单/AboutPanel 版本「v—」→读取失败/useSinStory 自动命名）。低优挂账=MemoryHubPage 计数「—」占位。**刀线二 tertiary 文本令牌收敛**=ThemeTokens 新增 colorTextTertiary（6 暗 #8b93a0 原值平移/6 亮 #5b6472——原 #6b7280 在 Violet 亮 surface 实测 4.41:1 不达 AA 且三别名硬编码在 App.tsx 改不到；#5b6472 全 surface ≥5.45 连容器高层过线）；App.tsx 三别名（--md-sys-color-text-tertiary/--v3-fg-soft/--color-text-tertiary）改由令牌下发变量名全保留=25 消费点+softTextStyle 7 文件零改动；机检扩面=BODY_TOKENS +tertiary（12 主题×2 背景 24 条新断言）+明暗锁值+App.tsx 防回流源断言；别名层裁决=纯转发（--whisper-ink-muted/--fg-faint）保留、--fg-faint 千级重指向独立拍板、--w-ink-3 需目检挂池。**门禁**=定向 +4+tsc 0+eslint 0/0（新增 4 处 exhaustive-deps 补齐）+全量 ci 绿（vitest 387 文件 3305 例）+drift OK@706+版本三处 4.362.0。**坑**=①LocaleProvider 跟随 localStorage（gaea-lang），beforeAll loadLocale 不稳固——重渲染后 t() 翻回 en 中文断言飘，须 beforeEach setItem②antd disabled Button accessible name 在 jsdom 不稳定，按钮定位走 testid/textContent③zh-TW 必须补齐全部 DictKey（tsc 当场拦）④恢复类先删后插的吞错=删半截+重复，失败要中止⑤对比度选值喂最差场景背景（Violet 亮 surface 4.41 而非白底 4.83）。**产物**=exe 50946048B SHA256=c26432852cdcb9211edf0c3e4bb6d6831a6be8a9dd57e09331276783aa3cd274（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.358~v4.362 删 v4.357.exe。**文档**=releases/v4.362.0.md+CHANGELOG/README+releases/README（计数 383→384+34 席插 v4.362 裁 v4.328）+AGENTS 迁 1 插 1（七十七迁：v4.359 入 archive）+progress/todos。

## v4.361.0 · 前端优化轮：失败可见化 + 观察池清账 + 令牌卫生（2026-09-20）
> 用户口径「优化gaea前端UI」：三路并行只读审计（令牌一致性/交互三态/观察池取证）后定刀，纯前端 34 源文件+4 测试文件+2 新工具文件，绑定面 706 不动（零 Go 改动）。**刀线一 失败可见化**=①NovelSettingPage 读失败防覆盖（loadFailed 态+Alert 重试+禁保存+Ctrl+S 拦截——原空编辑器可覆盖保存真实设定，唯一真丢数据风险）②SearchModal 检索失败三态（参与的线全败才报错、单路降级；原失败伪装「未找到」；顺手补存为任务失败提示）③ConsistencyPanel 检查失败不伪装「全部通过」（ruleError 横幅+「结果不可信」空态）④CostLibraryView batchStatus 失败计数（照 batchDelete v4.350 口径+busy 闸；原无条件报假成功）⑤CostLibraryPage 概览失败≠空库（全失败走错误态+重试，单路降级）⑥WeixinPage 提醒失败态+首拉 loading（照 v4.351 范式+loadedOnce 标志）⑦批量吞错 6 处（KnowledgePanel 审核/MemoryPanel 三开关回滚 toast/ChatPage 清空失败恢复快照/ModuleLauncher 语音发送/TaskCenter 输出/VoiceSettingsPanel）。**刀线二 观察池清账**=⑧file:// 头像通道统一（PortraitImg 抽 usePortraitUrl hook+成功缓存；PersonaPicker/AssetStudio×2/WeixinPage×4 残留直连全接线——v4.355 走查 27 条 console 告警清零）⑨longtask 日志阈值 200ms（69~165ms 常规抖动不再刷屏）⑩schedule 弱化文字 outline→on-surface-variant（css 9 处+DIM_COLOR+AoaView 2 fill+PredEditor；亮态 1.3~1.8:1→≥6:1，border/background 用途保留；回归锁进 contrast-hex）。**刀线三 令牌卫生**=⑪softTextStyle 7 文件逐字重复→utils/uiStyles.ts 单源⑫情绪色 map 2 处重复→utils/emotionColors.ts（hex 豁免名单同步登记）⑬novel-workspace #fff 兜底反向→currentColor+tailwind --transition 注释覆盖关系。**门禁**=定向 +6+build 0+eslint 0/0+全量 ci 绿（vitest 386 文件 3301 例）+drift OK@706+版本三处 4.361.0。**坑**=①「参与者全失败」合取式不参与的路贡献 true（vacuous）——写反两次被定向测试抓出②源断言行首锚定+标点排除前缀族（border-color/outline-variant 误伤）③no-raw-hex 豁免按文件名，色板迁移要同步登记④hook 别与组件同文件（react-refresh 0 warn 门禁）⑤测试 select 定位按 option 文本、value 必须是合法状态值。**产物**=exe 50941440B SHA256=a2c695b3b6b375c309fa6ecb3dac0942b31e8a49e00cea6c2c7e1484b0f9b5da（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.357~v4.361 删 v4.356.exe。**文档**=releases/v4.361.0.md+CHANGELOG/README+releases/README（计数 382→383+34 席插 v4.361 裁 v4.326）+AGENTS 迁 1 插 1（七十六迁：v4.358 入 archive）+progress/todos。

## v4.360.0 · 后端优化轮第五弹：挂池再清 3 刀——.tmp 卫生守卫 / 导入事务化 / 版本号原子化（2026-09-20）
> 用户口径「继续」：v4.359 余量再取证落地 3 刀（含 .tmp 病根本身入闸），绑定面 706 不动，前端零改动。**刀1 .tmp 卫生守卫**=scripts/clean-tmp.ps1 新增+ci.ps1 接入：.tmp 超 512MB 只清已知安全瞬态模式（Test*/*.log/edge-*·walk-* 等 profile/smoke-*.exe/go-build*）；坑=.ps1 含非 ASCII 必须带 UTF-8 BOM——无 BOM 时 GBK 误读破坏 param 行，阈值默认值静默失效变「每次都清」，首跑即暴露加 BOM 恢复。**刀2 ImportProjectCharacters 整体事务化**（characterlib）=抽 execer 接口（DB/Tx 公共 SQL 子集）+getOn/prepareUpsert（磁盘 IO 不入事务）/upsertOn/associateOn 四助手拆分，Get/Upsert/Associate 改薄包装签名不变（绑定面零影响），导入循环 Begin→tx→Commit 任一步失败整体回滚；既有 4 个 Import 测试零改动全绿。**刀3 SaveVersion 版本号原子化+SchemaV22**=INSERT 内 SELECT MAX+1 自算版本号（单语句 SQLite 写锁下天然原子，读改写窗口消除）+回读实际落库版本填返回值+SchemaV22 先清历史重复（保留最新 rowid）再建 idx_cost_versions_proj_ver 唯一索引硬约束背书；定向测试=版本序列 1,2 连续+直插重复被拒。**留池仅剩 2 项**（均需设计）=prompt 双构造（agent 持 eng 指针时序）/ConvertToPdf 绝对路径（Pick/白名单统一设计）。**门禁**=定向 +1+build/vet 0+受影响 4 包绿+全量 ci 绿+drift OK@706+版本三处 4.360.0。**坑**=①.ps1 无 BOM 乱码是语义问题非显示问题（守卫每次都触发而非报错）②execer 接口是 Go 事务化标准姿势，磁盘 IO 留事务外③「单语句天然原子」优先于「包事务」（INSERT...SELECT MAX+1 零锁成本，事务留给多语句组合）。**产物**=exe 50936832B SHA256=3c42f42929ee734a49e4c3921800b9065719e14c2a56f3e661f4661f7f3abd58（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.356~v4.360 删 v4.355.exe。

## v4.359.0 · 后端优化轮第四弹：挂池清账——v4.358 审计辨伪项落地 7 刀（2026-09-20）
> 用户口径「继续继续」：v4.358 挂池逐项再取证定刀——可落地 7 刀全清，需设计/重构/复现的 4 项如实留池。纯 Go 7 源文件，绑定面 706 不动，前端零改动。**刀A SQL**：①SelfHeal 启动期 N+1 点查（pathResolves 每条目每段逐条 QueryRow，千条库数千次点查）→分类树一次捞内存 map 纯查表②migrateLegacyDB 逐文件直写非原子→只拷主+wal、.migrating 临时文件+rename 原子落位、-shm 不拷（自动重建）+残留 log.Printf 收口 slog③FTS 降级 LIKE 多列长 OR 链（在册 modernc 空集坑高危形状，兜底自己失明）→likeSearch 助手按模式分轮单列查询+Go 合并去重+凑满提前收工。**刀B 启动链**：④Startup 幂等三处（logClose 先关旧/weixinServers 先清扫/fileWatch 先 Close）⑤ai.Client 双构造+桥接早启窗口→Startup 复用 New() 实例（nil 才新建）⑥filewatch 同步 Walk→app 层 go startFileWatch 整体移出临界路径（addTree 异步化试错证伪回退：监听就绪前事件真丢、全量索引兜底不成立——filewatch 保持 Start 返回=就绪契约）。**刀C 校验**：⑦GaeaListDir 相对路径拒 .. 穿越（四调用方全工作区相对零误伤；IsAbs 分支保留=v4.98 冻结面）。**留池 4 项**=ImportProjectCharacters 事务化（需 tx 化三助手）/prompt 双构造（毫秒级+agent 持 eng 指针时序风险）/SaveVersion 并发窗口（待复现）/ConvertToPdf 绝对路径（需统一设计）。**门禁**=build/vet 0+受影响 6 包绿+全量 ci 绿+drift OK@706+版本三处 4.359.0。**坑**=①审计论断「事件丢失有全量索引兜底」要核实兜底是否真存在——本例不存在，照抄即真回归，测试 5s 超时是免费反证②挂池再取证≠照单全收，留池与落地同等重要。**产物**=exe 50931200B SHA256=acd0851d4cbdcbfcac9801ffb59411418ad2530a32a0a8f8f8ba90e18c5fa89f（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.355~v4.359 删 v4.354.exe。

## v4.358.0 · 后端优化轮第三弹：三路审计（SQL 存储层/启动链路/输入校验）——分类改名 P0 根修+读侧收口+FTS 增量（2026-09-20）
> 用户口径「继续优化后端」：换新镜头三路并行只读审计，纯 Go 36 源文件+3 测试文件，绑定面 706 不动，前端零改动。**线1 SQL**：①分类改名 substr 字节/字符错位（P0，中文路径子树条目路径整段截断重挂）→rune 计数偏移+精确行叶子名同步+事务+换父重写+环检测+名称含 / 拒绝（同函数五问题一次收口）②rows.Err 缺失 17 处批量补（记忆/情节恢复主路径+成本检索语料+ListItems 直通不可变版本快照）③单条写全量重建 FTS O(N²)→接上闲置增量助手（失败降级全量重建兜底）+Rebuild 事务化④schema_v15 补 facts/episodes 会话索引+向量写事务化+COUNT 收错×5+GetSource 点查。**线2 启动链**：⑤ASR provider 四 goroutine 并发写竞争→SetASRProvider 持锁+读侧快照（对齐 SetRealtimeSession 先例）⑥恢复/迁移结论先于 setupLogging GUI 全盲→restoreSummary 字段日志就绪后回放⑦三 db 打开失败 log.Printf→slog.Error（GUI 可见）⑧Shutdown 补关 Hephaestus/whisper db+微信 Stop 通知异步化（对齐 notifyStart 先例）+回退轮询 stop 接线+reminderStop Once。**线3 读侧收口**（写侧防线完备、读侧系统性短板）：⑨GaeaReadFile 对齐写端口径（穿越拒绝+withinWriteRoots+2MB 截断）⑩AttachmentDataURL 根白名单（工作区+数据根）+32MB⑪ReadFileB64 契约强制化=Pick 登记机制（fail-closed，pickFile.ts 链路零破坏）⑫Preview 相对路径拒穿越+image/docx 32MB 封顶⑬快照/场景 sceneID 白名单（唯一写越界点：..evil 项目外建文件）⑭上下文看板 sessionPath 补 sessionDirForPath 校验⑮CreateProject 书架 containment。**辨伪挂池**=FTS 降级长 OR 链疑似空集坑/版本号并发窗口/legacy WAL 拷贝/Startup 幂等/filewatch Walk/prompt·aiClient 双构造/ListDir 全盘枚举（需查前端调用源）。**测试与门禁**=定向 +3（P0 中文路径回归+读侧三防线）+既有测试修正 2 处（伪造路径改真实会话目录形态）+build/vet 0+受影响 10 包绿+全量 ci 绿+drift OK@706+版本三处 4.358.0。**坑**=①Go/SQLite 长度语义差是中文环境必炸点——len() 字节喂按字符计数的 SQL 函数 ASCII 全绿中文必坏②增量助手存在却没人用——接闲置能力时补失败降级兜底③同函数多问题一次想全（分开修会打架）④读侧防线不能照抄写侧——根白名单+Pick 登记+尺寸封顶组合并逐绑定核前端调用链。**产物**=exe 50928640B SHA256=2e1bfdf827f3aa70cc2f8f84a21d3259d2776c6bec337e5f33b553c21e2641bf（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略（5 版）留 v4.354~v4.358 删 v4.353.exe。**文档**=releases/v4.358.0.md+CHANGELOG/README+releases/README+AGENTS 迁 1 插 1（七十三迁：v4.355 入 archive）+progress/todos。

## v4.357.0 · 后端优化轮第二弹：挂池三组清账——错误吞噬可见化 / 并发观察 4 刀根修 / 死代码 2 文件（2026-09-19）
> 用户口径「继续优化后端」：v4.356 挂池三组逐项取证定刀——纯 Go 8 源文件+3 测试文件，绑定面 706 不动（零签名变更 drift OK），前端零改动。**刀A 失败可见化 4 处**：①characterlib ListChatEnabled 吞错→签名 ([]Character, error) 上抛，WhisperGetPersonalities 失败 warn 回退内置人格（不再静默吞成空列表）、app.go 剧照同步 warn 跳过②DAG 懒清扫 _ = Save ×2→warn（sweep 幂等下次重扫，失败可见即可）③turn markdown 投影 _, _ =→warn（JSONL 证据链仍在，投影静默缺失不可见）④cost_projects 状态标签 Save ×2→warn（标签写失败列表页状态永远旧档）；辨伪=AppendTurnTraceToDB 仅 log——whisper 链无 Notice 事件总线（同步绑定返回）接线成本>收益。**刀B 并发加固 4 刀**：⑤PrepareContinue TOCTOU 根修=接通 store 半成品（SubagentRun.release 字段三件齐备却无人赋值恒 no-op）：continueClaims sync.Map 占 ref 单飞，第二路报 already being continued，claim 随 defer Release/终态写兜底释放，绑定层 claims 保留零调用方改动——双路续跑同 ref 交错写转录根修⑥cost BM25 语料/版本戳原子化=调用方捞语料前快照 corpusVer 传入 rankerFor（此前 rankerFor 内部再 Load，写路径在两步间推进版本时旧语料挂新版本 key 用到下次写）⑦weixin Stop→Start 双轮询根修=生命周期代际 gen，Start 递增，pollLoop 只在自己那一代存续（panic 兜底复位也加代际守卫防旧 loop 打掉新 loop）⑧realtime Dial 双拨号泄首连=CAS 落位锁内重检，输家关自己连接报 already connected（赢家连接被遗忘 fd 泄漏+双 readLoop）。**刀C 死代码**：删 control/audit.go+decisions.go ~230 行（New*Logger/SummarizeAuditLog 全仓零调用 v3.2/v3.3 期设计未接线，实际审计走 core/journal 与 desktop_audit_log 两条活链）。**辨伪不修**=file_backend 双写非原子（设计内自愈语义已在）/QuickAdd 锁内 AppendDoc（锁刻意串行化读改写，移出=并发丢更新）。**测试与门禁**=定向 +2（TestPrepareContinueSingleFlight 三段断言+TestOpenAISession_DialConcurrentLoserCloses 恰一胜一负）+既有测试适配 1 处（store_test 双返回值）+build/vet 0+受影响 6 包绿+race 本机无 gcc 跳过 CI race job 兜底+全量 ci.ps1 绿+drift OK@706+版本三处 4.357.0。**坑**=①半成品机制比没有更危险——release 字段+调用+注释齐备却无人赋值，读代码误以为守卫存在，新守卫落地先 grep 赋值点②可见化≠fail-closed——幂等写失败 warn 即可，判据=失败是否改变主操作结果③代际修复要护 panic 兜底复位路径④吞错修复优先改签名上抛而非原地打日志（编译器强制全调用点审查）。**产物**=exe 50900480B SHA256=c3877db06fd398b74df4a6b981b0f599ba104e7f81062ecd886b0ff51872a413（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略（5 版）留 v4.353~v4.357 删 v4.352.exe（SUMS 身份档案全保留）。**文档**=releases/v4.357.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十二迁：v4.354 入 archive）+progress/todos（挂池三组全清销账）。

## v4.356.0 · 后端优化轮：panic 防线收口 12 处 / 确定性自死锁根修 / cfg.Model 并发收口 / 快照失败可见化（2026-09-19）
> 用户口径「继续优化迭代 gaea，本次会话优化后端」：2026-09-12 后端性能普查已全清，换镜头开三路并行只读审计（panic 防线与错误吞噬/并发安全与回调发射序/资源句柄生命周期），按证据定刀——纯 Go 30 源文件+2 测试扩展，绑定面 706 不动（零签名变更 drift OK），前端零改动。**刀A panic 防线收口**（Wails dispatcher 只保同步绑定调用，goroutine 裸奔 panic=exe 闪退；65 处 go func 交叉比对 58 处 recover 撞出 12 处漏网）：①微信轮询主循环+消息处理（唯一常驻外部通道内同步跑完整 AI 回合全链零防线，同文件三处先例证明漏网）②DAG 流水线三处（节点 panic 转节点 failed 与 err 路径同形+编排层兜底+改向 goroutine——与 followup 逐行同构却没抄防线）③语音回合链三处（handleSpeechEnd/runReply/runRealtimePump 全文件 0 recover，文字/TTS/章节流 app 层均有唯独语音缺）④MCP 连接（外部进程协议数据 panic 转 RecordFailure）⑤书源五处（导入/追加/下载 panic 转 error 事件不挂进度；worker/单源转槽位错误项宁漏勿误）⑥dream/tts/SSE 解析。**并发守卫三处**：⑦GaeaMemorySetRetentionDays 持 ga.mu 调 GaeaInit——sync.Mutex 不可重入，未初始化时调用即**确定性自死锁**（绑定 goroutine 永久卡死持锁，办公板块全部 ga.mu 消费方连锁冻结仅重启可解）→锁外探测 needInit 仅未初始化才前置 GaeaInit（gaeaApplyCfg 同序，已初始化路径行为不变）⑧cfg.Model「UI 三写点×请求热路径四读点」裸字段竞态→config 包 modelMu+SetModelMem/GetModelMem 收口（Config 是值语义大结构加锁字段破坏可复制性，访问器是最小侵入解）⑨GaeaNewSession/GaeaResumeSession 补 Running() 运行闸（run loop「only swaps session while idle」是注释约定无强制，回合中换会话=SetSession 与无锁读交错转录分裂；照 GaeaSubagentFollowUp 先例绑定层自守防 httpbridge 绕过前端闸）。**刀B 失败可见性+卫生**：⑩回合收尾快照失败 slog.Warn-only→Warn 级 Notice（前端渲染可关闭错误卡——OneDrive/杀毒锁定会话文件时 UI 全绿重启回退数轮无提示，对照模型调用前 checkpoint fail-closed 纪律）⑪chat 流式 runID 毫秒时间戳拼 atomic 序号防同毫秒双路碰撞交错发帧 ⑫netclient 新增 DrainAndClose（非 2xx 早退不读尽 body=连接不复用，健康探测每次重新 TLS 握手；64KB 上限；替换 9 处 defer Close）⑬裸 rename 迁 fileutil.RenameWithRetry 21 处/18 文件（Windows AV 瞬时占用韧性；跳过跨盘回退/跨目录搬迁等特殊语义 12 处）。**审计辨伪**=Wails 绑定方法（框架 recover）/time.After 循环/用户正则 MustCompile/后台轮询退出条件/tasks·httpbridge 防线全净；线3 句柄轴零 P0/P1（body 52 文件/os 79 处/rows 93 处/ticker 22 处/子进程收尸全配对）。**测试与门禁**=定向 +2（死锁回归 TestGaeaMemorySetRetentionDaysUninitialized 带 os.Exit 超时守卫照 gaea_init_deadlock_test 先例+TestDagNodePanicMarkedFailed panic 转 failed 下游 skipped）+go build/vet 0+受影响 10 包绿+全量 ci.ps1 绿（vitest 386 文件 3295 例——ContextView 2 例在册 flaky 复跑 28/28 绿）+check-docs OK+drift OK@706+版本三处 4.356.0。**坑**=①goroutine 裸奔是绑定层系统性缺口——防线判据=同文件/同型先例是否已立，漏网处全是新码没抄先例②不可重入死锁是确定性非竞态，「锁外探测+幂等重入」保已初始化路径零变化现有测试零改动③cfg.Model 收口走包级访问器不改结构（加 sync.Mutex 字段毁复制语义）④vitest flaky 先复跑失败文件再定与本轮无关（足迹零交叠）⑤后台复合命令 build 假成功——产物必须核时间戳/哈希/大小三件，本次靠旧哈希识破。**产物**=exe 50893824B SHA256=9C570C94C07A1500282DAABDAE7FCD7BDC2E26ACA1BD305BDBF66EDC25709205（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略（5 版）留 v4.352~v4.356 删 v4.351.exe 顺带补清上版遗留 v4.350.exe（SUMS 身份档案全保留）。**文档**=releases/v4.356.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十一迁：v4.353 入 archive）+progress/todos。

## v4.355.0 · 真机走查班：UnifiedSearch 参数序错位根修（2026-09-19）
> 用户口径「继续」：前端优化六连发后按惯例开真机走查班（CDP 9333 附着 v4.354.0 壳，walk-v4354 配方），双空间 13 页巡检错误边界 0/空白 0/exception 0；走查拽出一个月级老 bug（修复即抬版本）。**UnifiedSearch 三调用方参数序错位**=Go 绑定 GaeaUnifiedSearch(query, topN int, scope ...string)（v2.23.0 起），前端 bridge 声明与调用方传 (query, scope, topN)——scope 字符串落进 int 形参，每次 unmarshal 失败；记忆中枢三脑检索/工作区搜索面板/Ctrl+K 三路在真壳上自 v2.23 一直坏。**为何捂一个月**=①mock 按前端期望签名写（mock 契约测试全绿，mock/真契约分叉又一例）②bindings_completeness_test 只收方法名不校验签名（在册坑）③前端 .catch(()=>null) 静默——v4.350 失败可见化（「检索失败，请重试」）一次走查就拽出。**修复**（零 Go 改动绑定面 706 不动）=bridge 签名改 (query, topN?, scope?)+三调用方换序（MemoryHubPage/WorkspaceSearchPanel/SearchModal）+mock 同序+三测试文件断言换序。**修复确证**（新壳 CDP 复验）=错误从 cannot unmarshal→embedding 请求失败（localhost:8080 语义服务超时——本机 embedding 未部署属环境事实挂观察池），UI 三态如实呈现。**其余发现**=角色库页 console 27 条 file:// 头像 WebView2 拒载（既有问题非本轮回归，挂观察池：正确通道=AttachmentDataURL 或 asset://）；[longtask] 69~165ms 误报阈值过敏感挂观察池。**门禁**=tsc 0+eslint 0/0+全量 ci 绿（vitest 386 文件 3295 例）+drift OK@706+版本三处 4.355.0。**坑**=①mock 契约必须从 Go 绑定复制签名（含参数序），按「前端怎么调」写=签名错位时 mock 全绿真壳全坏②静默吞错捂 bug 时长与可见性成反比——失败可见化是最便宜的测试③走查探针按钮定位优先类名/testid，textContent 匹配会撞到别的按钮得假阴性。**产物**=exe 50877440B SHA256=b012287b889a5eceb74915ce8888a70be9344185372b5e32e07d98e5547e173f（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.350~v4.355 删 v4.349。**文档**=releases/v4.355.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十迁：v4.352 入 archive）+progress/todos。

## v4.354.0 · 前端优化第六轮：P0 数据形状崩溃×2 / 异步竞态 11 处 / 资源生命周期（2026-09-19）
> 用户口径「继续」：五连发后新现场再开三路并行只读审计（异步竞态与并发/数据形状防御/资源生命周期第二层），纯前端 17 源码+1 测试扩展，零 Go 改动零绑定变更（706 不动）。**P0 数据形状崩溃×2**（后端产出前端未声明形状+mock 不可复现同型）：①伏笔面板 6 态漂移——前端 STATUS_META 只声明 3 态而 Go 状态机 6 态并集（pending=外部导入初始态/partially_resolved/abandoned）原样透传，「清理→项目重置」把手动条目置 pending 后重拉即 undefined 整页崩（一键自毁式）→类型 6 态并集+STATUS_META 补全+statusMetaOf 防御未知值+流转按钮 6 态文案，测试+2；②记忆面板 skills=null——Go suggestSkillsFromMemories 规则类记忆<2 条 return nil，nil slice 无 omitempty 序列化 JSON null（mock 恒 [] 故 ?mock=1 永不复现），轻量用户点「扫描」即 null.length 崩→setSuggestions 入口统一收窄归一。**异步竞态 11 处**（统一项目已有 seq/loadToken 范式补齐）：PromptWorkshopPanel 模板切换（慢响应 A 正文写进 B 表单，保存即覆盖 B 模板数据损坏）/BookSearchModal 选书（B 书源 URL+A 章节范围错误导入）/useChatTopics.createTopic+ChatPage 清空（旧话题在途响应填满空话题视图，清空对话清错对象→新增 invalidateLoads）/useSinStory.loadMessages（切故事旧消息覆盖+发送串台）/ChapterPage.handleSave（场景章串行保存期间继续打字，完成侧强制 saved=true 打掉未保存保护静默丢稿→保存快照+函数式比对）/SearchModal（Enter 双触发 onSearch+onPressEnter 双挂+无 seq→旧意图卡可被执行）/MemoryHubPage.runSearch（Enter 无守卫旧命中覆盖）/useImageGenConfig 切后端（state await 后才写，二次切换绕过相等检查）/outlineStore（快速换书旧书大纲覆盖，按 A 树章号写 B 书错误章节）/WeixinPage.toggleAssistant（Switch 受控到轮询才翻转诱发连点=两条相反 Save）/TTSPlayer（播放中卸载泄 blob URL 且音频继续外放→cleanup 接入卸载路径）。辨伪 18+（antd Modal async onOk 自带 confirmLoading 等）。**资源生命周期第二层**=TTSPlayer 修完；dirListingsCache 无界+invalidateTurnCaches 零调用方挂池（结构性欠账需设计接线时机）；blob URL/Observer/AudioContext/MediaStream 全量辨伪通过。**门禁**=tsc 0（--force）+eslint 0/0+全量 ci 绿（vitest 386 文件 3295 例）+drift OK@706+版本三处 4.354.0。**坑**=①Go nil slice→JSON null 且 mock 手写 [] 永不复现——收窄层统一归一优于消费点逐个防御②saved 标志语义=「缓冲区==磁盘」，异步完成侧不能无条件置位，快照比对通用解③antd Input.Search 回车已触发 onSearch，再挂 onPressEnter=双发，删除而非守卫④竞态修复统一走已有范式不发明新机制。**产物**=exe 50877440B SHA256=758eba83…（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.349~v4.354 删 v4.348。**文档**=releases/v4.354.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十九迁：v4.351 入 archive）+progress/todos。

## v4.353.0 · 前端优化第五轮：ResizableDrawer 弹层可访问性收口 / 可点 div 键盘化余量（2026-09-19）
> 用户口径「继续」（v4.352 池延续），可访问性余池两条线收口；纯前端 9 源码+1 测试（新建），零 Go 改动零绑定变更（706 不动）。**ResizableDrawer 三缺补齐**（对照 ApprovalModal 先例）：①aside 加 role=dialog+aria-modal+aria-label（新可选 prop，两调用方传现成 i18n 标题 caps.title/history.title）——阻断式弹层此前对读屏是匿名区域；②Esc 关闭（document keydown，可编辑目标内不劫持同 ApprovalModal 纪律）——此前唯一关闭路径是鼠标点遮罩；③焦点管理（开时记录 activeElement+聚焦抽屉容器 tabIndex=-1，卸载还原到打开者 try 兜底）——此前开不聚焦关不还焦键盘用户关闭后落回 body 迷路；测试 ResizableDrawer.test 新建 4 例（Esc 经退出动画/普通键不关/dialog 三件套/开聚焦+卸载还原）。**可点 div 键盘化余量**（v4.349 池逐点核对净剩 4 处+顺带 5 钮）：ChapterTreePanel 章节树节点（主选择动线）/OutlinePanel 大纲行/StatsPanel 会话・本轮两个折叠切换→role=button+tabIndex+Enter/Space+aria-label（StatsPanel 加 aria-expanded，FileTree 范式）；ChapterEditor 加场景/删场景+ChapterTreePanel 重新生成/删除/添加五 icon Button 补 aria-label（title 不参与可访问名）；FiveCalcPanel 版本带入遮罩补 Esc。辨伪=RailItem/FactCard 内嵌跳转已在此前轮次变 button，清单是快照不是现状照单全收会重复改。**门禁**=tsc 0（--force）+eslint 0/0+全量 ci 绿（vitest 386 文件 3293 例）+drift OK@706+版本三处 4.353.0。**坑**=①effect 引用 useCallback 声明顺序：Esc effect 插在 handleClose 声明前——effect 回调虽延迟执行但依赖数组渲染期即时求值 TDZ 直接 ReferenceError，effect 一律放依赖声明后②React 合成 KeyboardEvent 无 isContentEditable（那是 DOM HTMLElement 属性），div onKeyDown 判可编辑目标要 e.target 断言③审计清单逐点核对再动手。**产物**=exe 50875904B SHA256=2f46336262f5a858544408cb9dc19672f0bc8804a9d126de188286ddca5fbea5（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.348~v4.353 删 v4.347。**文档**=releases/v4.353.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十八迁：v4.350 入 archive）+progress/todos。

## v4.352.0 · 前端优化第四轮：ForeshadowPanel 行 memo / ModelCenter ctx 两通道拆分 / 图标钮可访问名（2026-09-19）
> 用户口径「继续」（v4.351 下刀池延续）；纯前端 16 源码+5 测试（扩 5），零 Go 改动零绑定变更（706 不动）。**ForeshadowPanel 行 memo**=抽 ForeshadowRow memo 组件+flow/remove/saveEdit 等六回调 useCallback 化+编辑态布尔/文本双 prop 下发（原 items.map 全量渲染，行内编辑/登记表单每键全列表重渲染，数百行可达）；测试 +2（编辑保存全量写回含 3 条/取消不写回）。**ModelCenter ctx 两通道拆分**（v4.350 辨伪项正解）=ModelCenterContext 拆 StateContext（状态字段，内容变化才重渲染）/ActionsContext（setter/handler 引用波动单独承载）/兼容口（两通道合并，useModelCenter 存量不破坏）；页面 stateCtx/actionsCtx 各自 useMemo，9 个 section 迁 useModelCenterState+按需 useModelCenterActions，settingGlmEndpoint 归 state——键入 API Key 不再全 section 重渲染；4 个测试文件兼容 Provider 补三连挂载。**图标钮可访问名**=ChatRow 四处（用户/助手复制动态态+朗读）+ChatModeBar 四处（联网开关动态态/角色库管理/切换角色/语音设置）补 aria-label（Tooltip 不参与可访问名）；SecurityPanel 排除（按钮带文字）。**门禁**=tsc 0（--force 全量复核）+eslint 0/0+全量 ci 绿（vitest 385 文件 3289 例）+drift OK@706+版本三处 4.352.0。**坑**=①多行解构迁移必须先从源码抄全字段清单再分类（凭接口记忆漏 trendData/settingGlmEndpoint，UNCLASSIFIED 断言+tsc 双保险）②测试文件改完必须 tsc --force（上轮沉淀本轮再验证）③兼容口三 Provider 嵌套顺序 state→actions→兼容，测试挂载三连 value 同对象语义不变。**产物**=exe 50874368B SHA256=32c47fa87e7511fa6b3903690986f701ec9b3f8e1cec571ff87ee6df6d815d23（releases+SUMS 仅本地不入库；桌面副本同哈希；冒烟 200 过）；保留策略留 v4.347~v4.352 删 v4.346。**文档**=releases/v4.352.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十七迁：v4.349 入 archive）+progress/todos。

## v4.351.0 · 前端优化第三轮：启动反闪根修 / 绘梦历史有界化 / 亮主题叠加层失效群 / color-scheme / 反馈确认补遗（2026-09-19）
> 用户口径「继续」（v4.350 下刀池延续），六线并收；纯前端 29 源码+4 测试（1 新 3 扩），零 Go 改动零绑定变更（706 不动）。**启动反闪根修**=亮色用户每次启动先看 ~2s 深空屏（index.html 静态启动屏硬编码深底，主题变量 App useEffect 首帧后才注入）→ head 内联脚本（bundle 前）预读 gaea-display-mode——判定顺序逐字对齐 appStore.loadMode（light→亮/system→matchMedia/旧布尔键 gaea-dark・wubigork-dark '0'→亮/缺失→dark 缺省）亮则 html[data-bs-light]+静态屏与 body 浅色分支+boot-splash.css 首帧兜底（令牌注入后 var() 接管两段同色无跳变）。**绘梦历史有界化**（无界三处收口，素材库 120+12/页先例对齐）：持久化 HISTORY_META_MAX=500 读/写同截（原 path-only 条目可积数千）；渲染窗口 60+「加载更多(N)」增量展开（原数百 img/video DOM 一次性构建）；回填 restoreHistoryImages 加 limit 按 needsFileRestore 过滤取前 48（内联小图不占名额；原全量 dataURL 常驻内存=数百次生成数百 MB），窗口外缩略图落既有占位、选中/下载/复用 resolveResultImage 按需读文件——记录仍在零功能删除。**亮主题中性叠加层失效群 30 处**=rgba(255,255,255,0.02~0.08) 暗底白提亮惯用法在浅底白叠白不可见（hover 消失/卡片只剩边框/进度轨道消失/shimmer 不动）→统一 color-mix(var(--color-text) N%, transparent) 明暗自动正确（PersonaPicker/imagegen 全家/RelationshipModal/CharacterFormHelpers/AppearancePanel/ChapterIllustration/WhisperDesirePanel）；辨伪保留 ChatCodeBlock 2 处（hex-exempt 暗色代码面板 chrome 刻意不随主题）；MarkdownContent pre 底 rgba(0,0,0,0.3)→surface-container 令牌（亮主题中灰脏块）。**color-scheme 声明**=App.tsx 注入 root.style.colorScheme（原全仓无声明，暗主题原生滚动条轨道角/表单控件残留亮色渲染）。**反馈确认补遗**（上轮 B 系列遗留 4 处）=①角色记忆弹窗三路读取全吞错→失败 Alert（原打开即空白与「无记忆」不可分）②青鸟助手两路全失败原伪装「暂无助手」假空态→载入失败条 role=status（单路失败仍兜底不误报，轮询自动重试）③图片下载两处失败 message.error+成功提示（另存为抛错原落 unhandled rejection；取消不算错）④自定义生图模板删除补 Popconfirm。**测试**=HistoryRail.test 新建 3 例（超窗/增量/全量不回归）+historyMeta +1（limit 按需恢复项计数）+meta +2（读截/写截）+WeixinPage +1（双失败提示+单路兜底不误报）。**门禁**=tsc 0（强制全量复核）+eslint 0/0+全量 ci 绿+drift OK@706+版本三处 4.351.0。**坑**=①静态期主题预读判定序必须逐字对齐 store.loadMode，多一个 matchMedia 兜底就分叉首帧跳变②limit 语义按需恢复项计数非位置（slice 会让内联小图占名额）③heredoc 写测试三层转义（python→文件→JS）\ 才是源码单反斜杠，写完看原始字节④rgba(255,255,255,x) 不全是债，先排除 hex-exempt 名单再批量替换。

## v4.350.0 · 前端优化第二轮：事件总线连环炸根修 / P0 幽灵令牌 / 破坏操作确认 / 静默失败 / 流式渲染热点 / 主题破绽（2026-09-19）
> 用户口径「优化 gaea 前端」（v4.349 五线后第二轮）：四路并行只读审计（事件与定时器泄漏/交互反馈三态/渲染热点/视觉一致性）按证据定刀，纯前端 32 源码+6 测试（3 新 3 适配），零 Go 逻辑改动零绑定变更（绑定面 706 不动）。**事件总线连环炸根修**=①onTaskEvent 清理走 EventsOff(channel) 全清（wails 语义=注销该通道全部监听者，v4.61 事故红线 v4.62.2 漏网）：gaea-task 通道 5 个并发订阅点（运行角标/任务自动激活/任务面板/价格源/索引任务），任一卸载=角标冻结+自动打开失灵+下载卡进行中，keepAlive 下直至重启→一律 subscribeWailsEvent 按监听者退订，onUpdaterProgress/onReady 同纪律收口；②runtimePolyfill（网页模式）EventsOn 此前返回 void→消费方 deps 每变往 eventBus 叠加 handler（ModelCenter 四通道单事件 N 次重复后端拉取）→改返回「只摘自己」退订函数对齐 wails v2.13 桌面语义；EventsOff(带 callback) 无条件 sse.close()→共享通道他人推送被掐断→只摘自己、通道空才关 SSE。**P0 幽灵令牌**（12 主题功能性坏点）=--md-sys-color-error 全仓零定义而 gaea 四组件 26 处无 fallback 消费（CSS invalid：失败态红点/红字/红框全消失）→注入 colorDestructive 语义别名；--md-sys-color-surface-container-low 零定义→追问发送按钮箭头 12 主题恒隐形+输入区透明+VersionTimeline 对比底丢失→color-mix 插值注入；--color-info/--md-sys-color-info 补注入（#38bdf8/#0369a1 明暗分档，novel 风格指纹分档/CreatePage 图表分组在 gaea 板块外整条 invalid）；幽灵令牌四连 --v3-fg-soft(17 处)/--v3-line-soft/--color-text-tertiary/--md-sys-color-text-tertiary 补明暗定义（暗主题 12px 次要文字 3.9:1<AA）；modelcenter.css 13 处 gaea 作用域 token 换主令牌直连（--fg 系只在 gaea/styles.css 定义且该页不加载→亮主题 #ddd 白底 1.3~2.5:1）。**破坏操作二次确认 9 处**=高危批量/清空 4：知识库批量删除/成本库批量删除（连带假成功根修：吞错后无条件「已删除 N 条」→诚实成功 N 失败 M）/绘梦清空历史（prompt/seed 即时持久化）/原罪清空故事消息（ToolbarButton 无 forwardRef→Modal.confirm 命令式）；单条 5：任务收件箱删除（连带吞错可见化+i18n 三语新键）/青鸟提醒/组织/角色关系/划线想法列表（span 挡冒泡防触发跳转）。**静默失败反馈**=三脑检索失败与 0 命中三态化（命中/换关键词/失败重试 role=status）；排程导出 XML 补成败提示（SaveFileAs 抛错原落 unhandled rejection）；章节载入失败白板→message.error 带原因。**渲染性能**=Transcript 提及扫描按文本内容缓存（原每 chunk 对全会话正文跑双全局正则+O(m²) 重叠，长会话流式每 chunk 数 ms~数十 ms；完成后文本不可变→缓存命中只付流式段成本，上限 512）；ChatPage topicList memo 化（原每 render 重建 O(T×P)，角色打字机≈70 renders/s）+ChatTopicSidebar React.memo+过滤 useMemo+行 hover 删 state 改纯 CSS；ToolCard summary/prettyArgs/outputLines 三处 memo（write_file args 内联整文件数百 KB，折叠态每次更新全付 JSON round-trip+全文 split）；useNow(active) 条件订阅（已完成消息/过程卡/RunStatus 无条件订阅 1s 全局时钟=O(N)/秒常驻重渲染清零；伴生修复 AssistantMessage「思考 Xs」完成后持续增长→完成边沿 Date.now() 定格）。**视觉**=RelationGraph 画布 #ddd/#555/#333/#444 写死（亮主题节点名不可见）→resolveCSSColor 解析主题令牌+darkMode 入 deps；mermaid 明暗跟应用 data-hl 而非 OS prefers-color-scheme（应用亮 OS 暗=深底图表嵌浅文档）。**审计辨伪**=ModelCenter ctx（100+ 字段）不 memo——五 hook 返回对象字面量每渲染新引用 deps 无法建立，强行 memo=陈旧 ctx 真 bug 挂观察池；TTSPlayer/AIConsole 单实例全清无害；z-index 1080 阶梯脆弱但无叠加路径。**测试**=events.test 新建 4 例（双订阅退订隔离+EventsOff 永不调用/space 过滤不变）+runtimePolyfill +3 例+useNow 3 例；TaskInboxPanel/WeixinPage/KnowledgePanel 删除用例改两步确认。**门禁**=tsc 0+eslint 0/0（新色值走 // hex-exempt）+全量 ci.ps1 绿（vitest 384 文件 3280 例）+drift OK@706+版本三处 4.350.0。**坑**=①EventsOff 红线是「禁全清」非「禁用」——polyfill 侧带 callback 只摘自己+通道空才关 SSE 才是完整语义②Popconfirm clone 注入 onClick 覆盖子元素 onClick（v4.348 Dropdown 坑家族），子按钮原 onClick 必须移除；无 forwardRef 按钮包不上改命令式③antd 两字按钮确认钮定位一律 testid（okButtonProps 透传），按文案找会撞「删 除」空格名④useNow 停订后完成边沿定格有 ≤1s 误差（最后一次 tick 值），显示层可接受⑤go test 全量负载 flaky 本轮又现（足迹零交叠复跑绿），ci 日志别用 tail 截断否则 FAIL 无包名查不了。

## v4.349.0 · 前端优化五线：可访问性动线 / 亮态可读性 / 运行开销 / 布局 / 稳健性（2026-09-19）
> 用户口径「优化迭代 gaea，进行前端优化」；先四路并行**只读审计**（主题可读性/交互可访问性/渲染性能/布局响应式，均带 file:line 与实测数字）再按证据定刀，纯前端 18 源码+9 测试，**零 Go 逻辑改动、零绑定变更**。**可访问性动线**=记忆中枢全局检索命中行（主入口动线）裸 div → role=button+tabIndex+Enter/Space；轻语记忆面板标题/摘要键盘化+确认/取消/编辑/删除四图标钮补 aria-label（此前只挂 Tooltip，不参与可访问名）；阅读「删除高亮」补 Popconfirm 二次确认（误点即删不可恢复）；Lightbox 关闭钮补可访问名；批准弹窗 aria-modal false→true。**亮态可读性**=①真缺陷：changesdiff-tok.css 写死 One Dark 七色而容器继承面板背景（亮态浅底）实测 .tok-typeName 1.38:1 / .tok-string 1.61:1 → 接共享语法调色板 --hl-*（单一真源 hljs-theme.css，ChangesDiff 显式 import 真源防懒加载链 var 未定义），文件零字面色值；②lightFn 语义令牌下沉：warning #d97706(2.90~3.13:1)→#92400e、success #059669→#047857（Moss lime 阶保留）、colorPrimary Jade/Rose/Amber →#0f766e/#be123c/#b45309（accentRgb/glow 跟随；Violet/Moss/Slate 原达标不动）；③新增 12 主题×5 令牌×2 背景对比度矩阵机检+深阶锁值+hljs 浅色组逐令牌 ≥4.5。**运行开销**=①main.tsx 诊断层 500ms 轮询只写时间戳（17.3 万次/天空转）→ 心跳由既有 rAF 探测循环每帧维护、降级路径补 500ms 兜底续跳（否则误报卡死）；②getThemeTokens 每次返回新对象且留在渲染体 + ConfigProvider theme 内联字面量 → 每次渲染重跑 ≈45 个 setProperty 与 antd 主题算法 → 双 memo 化；③SelectionToComposer 的 selectionchange 先判 isCollapsed 短路（原上来就物化整页选区文本）；④WhisperMemoryList 过滤/分组 useMemo+查询词预小写（原每次渲染 ≈7N 谓词）。**布局**=摘要 chip 补 nowrap+shrink-0（窄列曾标签内折行、整排参差，「基线 N」chip 即在此排）；基线槽位行 wrap+工期/时间戳固定段 nowrap/flexShrink:0（原固定部分 326px>minWidth 300 致行内换行）；海报描述补 line-clamp（overflow:hidden 定高卡原先静默裁切无省略号）。**稳健性**=①书架 loadProjects 只 console.error → 首页把「读失败」渲染成「书架空空如也」假空态 → store 增 projectsError（只表态不抛出）+首页错误态含原因与重试；②「最近文档」仅挂载时读一次而 v4.346 起页面常驻保活 → lib/recentFiles 增订阅通道（写入广播+解析缓存保快照引用稳定，满足 useSyncExternalStore），首页账页与办公「最近文件」条同改订阅（空态挂载后首次写入也出现）。**审计辨伪**=印泥章 .w-seal #d06055 不改（20px/700 属 WCAG large text 阈值 3:1，实测 3.49~3.83 达标，按 4.5 判死是误报）；reduced-motion 无缺口（index.css:461 全局兜底已在+GSAP 三 hook 均调 prefersReducedMotion）；暗态 6 主题×27 配对 0 失败。**测试**=新增 4 文件 44 例（对比度矩阵 18/记忆面板 5/阅读浮层 3/列表 memo 5/结构守卫 13）+扩展 5 文件 8 例（命中行键盘 2/首页失败重试 1/最近文件广播 2/挂载后写入 2/选区短路 1）。**门禁**=tsc 0+全量 ci.ps1 绿+drift OK@706（零绑定）+版本三处 4.349.0。**坑**=①审计阈值必须复核（large text 3:1 被按 4.5 判死会白改品牌色）②--hl-* 是懒加载 CSS 变量，接色板必须同时 import 真源③source-guard 里 indexOf('.hl-scope-dark') 会命中文档头注释须用 lastIndexOf④antd 两字按钮可访问名带空格（「删 除」）测试按 /^删除$/ 找不到⑤心跳改 rAF 驱动必须处理降级路径否则每 8s 误报卡死⑥新加测试文件后要重跑 tsc（tsconfig.app 含 src，CI 首轮即被 frontend build 拦下）。**未做（下刀）**=海报墙 hero 中段留白 276px（占卡高 65.7%，须加内容槽=内容决策，改动需验 1440/1180/1100/760 四断点）；其余可点 div 键盘化（章节树/大纲/统计分类 ~8 处）与 ResizableDrawer 手写弹层焦点管理按动线权重分刀；进度计划页 .gsched 工具栏换行与基线 Popover 本轮未实测（mock 无「进度计划」入口）。

## v4.348.0 · 原罪板块优化双刀：故事底稿直注前情 + EPUB 电子书导出（2026-09-19）
> 原罪优化完善班两刀：①长程一致性——模型自记的便签/大纲底稿非空时随每轮前情直注提示词（预算 4000 rune 从头带、超限如实报数引导 read；空稿逐字零漂移；工具描述同步「底稿已随前情附上」省 read 工具轮），掉出 12 条历史窗口的早期设定靠底稿确定性回归；②图文成品出口补全——SinExportEpub（绑定面 705→706）：每条助手回合一节（中文序数）+用户指令引块置首+插图内嵌（go-epub，首图兼封面，webp 不支持则占位不硬塞）+未生成占位，落 sin/exports 同名不覆盖；顶栏导出改下拉（Markdown 另存为/EPUB 落盘告知路径）。**测试**=底稿直注 5 用例+EPUB zip 全链 2 用例，既有三处调用点补参断言零改动；**门禁**=tsc 0+全量 ci 绿+drift OK@706+spaceBindings 锁 528→529+版本三处 4.348.0；**坑**=Dropdown clone 注入 onClick 覆盖子元素 prop（ToolbarButton 占位空函数即净，不动办公组件）/单条恰=预算的便签整条收下（「超了才截」）。

## v4.347.0 · GLM-5.3-FlashX 新模型接入目录 + Flash 补官方绝对价（2026-09-19）
> 智谱 9 月中旬新发 GLM-5.3-FlashX（Flash 加速版，320B 总参/18B 激活，200 tokens/s），官方四页交叉核实（模型概览/详情/定价/coding 活动）后接入模型中心 GLM 静态目录。纯数据刀，零新绑定 705、零代码逻辑变更。
**落地**=①`glm_catalog.json` 追加 `glm-5.3-flashx`：1M/128K、caps 同 Flash（vision/tools/reasoning）、国内绝对价 2/7 元/M（缓存命中 0.57 记 price_note）、尾部追加纪律②coding 套餐官方明示「暂未开放 FlashX」（仅 Flash 上线）→不配积分系数、不进别名表③顺带翻新 Flash：定价页补出绝对价 0.8/2.8 元/M（缓存 0.23）——「仅相对价不填绝对价、估算回退内置表 USD」的 09-02 口径作废，估算从 USD 近似（0.65/M+M）变官方国内价精确值（3.6 CNY/M+M）④`glm_catalog.go` 头注释核实日期增补。
**测试**=锚定清单 +`glmCatalogV2FlashXIDs`；计数 45→46 全对齐（catalog_test ×5+catalog_remote ×3）；FlashX 元数据/价格锁值 +4；积分守卫名单 +FlashX；`TestGLMPricingVerified`/`EstimateCostCNY` 锁值翻新；CatalogSource 日期锚定；modelengine 包全绿。
**坑**=①文档站正文动态渲染抓不到表——`llms.txt` 索引+`.md` 后缀直出是正道②「无绝对价回退内置表 USD」是快照口径，官方补价后须回头改目录+锁值，否则估算长期用错币种近似（Flash 拖了 17 天）③同批发布≠同套餐：FlashX 不在 coding 套餐，逐模型核实再配计费。

## v4.346.0 · UI 健壮性双修：错误边界页级隔离（keepAlive 连坐根修）+ TisorRadar 缺档崩页（2026-09-19）
> UI 优化班——隔离目检（vite dev ?mock=1 + 无头 Edge CDP）双空间全页×明暗两态扫出一处整页崩溃与一处架构级连坐缺陷，全部根修。纯前端四文件，零新绑定 705、零新依赖、零功能删减。
**发现**=角色库整页崩进错误边界（`Cannot read properties of undefined (reading 'T')`，堆栈单点收敛 TisorRadar `dims[k]/100`）；顺带揪出更深一条：边界崩一次后切任何页都停在错误态——目检现场的「原罪页也崩」即此假象（原罪本体无辜）。
**根修**=①MainLayout 错误边界下沉到 keepAlive 每页内部：壳层 visitedPages 全部已访问页常驻挂载，边界包在整组外面时任何一页崩溃会卸载全部页（keepAlive 状态一并丢失）且后续所有导航停在错误态——与「防止单页崩溃拖垮整个应用」的注释意图相反；页级实例后崩溃只瘫自己那页，外层边界保留兜底②TisorRadar 组件级守卫：`dims` 可选+缺档渲染 null（真数据旧档/外部来源可缺，Inspector 的 `c.dims &&` 守卫是先例，卡片漏了）+CharacterCard 调用方守卫对齐（缺档不渲染雷达，不出假图）③mock 三角色补 dims：Go 侧 Dims 值类型必出，mock 对齐真契约——本次崩页正是 mock 与真实形状分叉暴露的洞。
**测试**=新增 TisorRadar.test +3（缺档 null 不崩/有档 5 点 10 圆+五标签/showLabels=false 零标签）；CharacterCard.test +1（chatEnabled 缺 dims → 零 circle 不崩、卡片动作照常；antd 图标也是 svg，按 circle 计数断言）。
**门禁**=tsc 0；定向 19/19 绿；全量 ci.ps1 绿；drift OK@705（零绑定变更）；版本三处 4.346.0（sync-version.ps1）；产物=exe 50844160B SHA256=5c2a7ec30751b31bc6eb739e2ba6b0250eec844b70795a7c8565954c22c24631（releases+SUMS；桌面副本同哈希；冒烟 200 过）。首轮全量撞 BookHealthPanel 负载型 flaky（足迹零交叠，隔离 7/7 绿，复跑全量绿收口）；抬版本手改漏 app_info.go 被漂移门禁逮住——一律走 sync-version.ps1。
**目检**=修复前角色库明暗皆崩「页面渲染出错」；修复后 3 卡×30 circle 雷达在位、错误边界 0，双空间明暗全绿。脚本与截图 .tmp/uiwalk-v4346*。
**坑**=①错误边界+keepAlive 是隐形耦合：边界粒度必须与常驻粒度对齐，否则「隔离」只在注释里②生成类型说必填≠运行时必达，TS 挡不住 mock/旧档缺档，防御守卫落在渲染组件边界③CDP 目检切主题走 rail 按钮 aria-label（Tooltip 不落 title）；无头 Edge 临时 profile 记住主题态，脚本按当前态自适应。
**未做（下刀）**=亮态 accent 弱化文本对比度打磨（视觉拍板项维持观察池）；动效手感待上手定论；真机走查清池班；技能核数 ≈09-30。

## v4.345.0 · filewatch 关闭竞态根修：fs 通道关闭路径漏 close(out) 致消费方挂起（2026-09-19）
> 真机走查班后续——全量 ci 偶发的时间型 flaky 深挖后是真并发缺陷。单文件 Go 修复+测试加固。
**根因**=loop() 三退出路径中，`fs.Close()` 连带关闭 fsnotify Events/Errors 通道的两条路径此前裸 `return` 漏 `close(w.out)`——与 done 分支 select 竞速（随机取胜），输了就永不关闭输出通道，消费方 `for range w.Events()`（工作区语义索引触发链）永久挂起=关闭竞态协程泄漏。压测频率 ~1/20，全量 ci 高负载放大竞速窗口。
**修复**=①outOnce sync.Once + closeOut()：三条退出路径统一幂等关闭②TestCloseStopsEvents 断言 2s 硬超时改终态轮询（10s 预算，v4.335 纪律）——既是加固也是回归锁（修复前 1/20 必挂，修复后 20/20 绿）。
**坑**=flaky 分两级：预算太短（改预算就绿，掩盖真缺陷）与预算拉满仍挂（真缺陷）——治理必须压测到「拉满仍挂」才停，否则只是把缺陷埋深；select 多通道退出的关闭路径逐条清点，主动关闭正确≠被动关闭正确（fsnotify 连带关闭=隐形触发源）。
**门禁**=go vet 0；filewatch 20 连跑全绿；全量 ci.ps1 绿；drift OK@705（零绑定变更）；版本三处 4.345.0；产物=exe 50843648B SHA256=b3befbdc11c7975aa272848ac9deca857bade2b2dca267feca573ba99e465cf0（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.345.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.345；持久 overlay 目检随下一班）；技能核数 ≈09-30；t2 余项按反馈。

## v4.344.0 · t7 最终形态：编辑器持久标注高亮（overlay 常驻 + 编辑退场）（2026-09-19）
> v4.343 轻量版的完全体（真机镜像对齐已过，最后顾虑清掉）；纯前端三文件，零新绑定 705、零新依赖。
**落地**=①分段工具收敛：新建 create/annotationMarks.ts（buildAnnSegments+runeToCodeUnit），面板高亮视图与编辑器镜像同一实现——rune↔code-unit 口径全仓唯一出处（Panel 本地副本删除，11 用例零改动=重构等价证明）②EditorPanel：单条 hl 换 annotations prop 驱动 marks（useMemo 派生），镜像渲染全量段（交叠钳制 Q5 口径）；dirty 纪律——正文编辑整体退场（偏移漂移结构性规避）、切章/标注重载恢复；gutter 镜像显形时 rAF 实测③CreatePage 随激活章加载 NovelChapterAnnotations（alive 守卫防竞态）④locate 句柄契约不变（光标+滚动），与常驻 mark 解耦。
**测试**=EditorPanel 重写+2 共 11/11（常驻双 mark rune 换算断言/编辑整体退场/locate 契约不受 dirty 影响/越界标注无 mark）；Panel 11/11（重构等价）；CreatePage 15/15（补 NovelChapterAnnotations 漏桩）。
**坑**=①effect 引用 200 行后才声明的 const=TDZ，tsc 不抓靠读序发现②测试桩漏项放大成组件树崩（unhandled rejection → 整树卸载 → 后续断言全"找不到元素"），先看 unhandled errors 段。
**门禁**=tsc/eslint 0；三域 38/38；全量 ci.ps1 绿；drift OK@705；版本三处 4.344.0；产物=exe 50843648B SHA256=95ecb91d66a9d1828a26fb8d7c660d1a4f0bba3db7a7c8292c47c654fd74d5fc（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.344.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十九迁）+progress/todos。
**未做（下刀）**=真机走查清池班（持久 overlay 真机目检随下一班）；技能 ≥5 次核数 ≈09-30；t2 余项按反馈。

## v4.343.0 · t7 收尾「编辑器标注高亮」轻量版：定位单条镜像 mark + 编辑即失效（2026-09-19）
> 观察池最后一项（v4.325 有意裁剪的编辑器 overlay）轻量落地；纯前端 EditorPanel + 一条 CSS，零新绑定 705、零新依赖。
**落地**=①镜像背景层：locate 时 textarea 背后垫同排版 div（同衬线栈/font-size 变量/行高 1.9/字距/20×24 padding/1px 透明 border/box-sizing），文本 color transparent 仅 mark 显色（--gaea-glow 32%），textarea 切 `.novel-editor-mirror-hl` 底透出（!important 源码序压过既有底色）②换行对齐：镜像按滚动条占宽（offsetWidth−clientWidth）补右内距；pre-wrap+break-word 与 textarea 默认一致；onScroll 同步 scrollTop+聚焦后 rAF 补同步③失效纪律：content/activeNode 变化（手写/流式/切章）即清除——偏移漂移正确性问题结构性规避；pointerEvents none 纯装饰。
**测试**=EditorPanel +1 共 11/11：真实 state harness——定位前无镜像/locate 后 mark=「一二」（rune[1,3)→code-unit[2,4)）/fireEvent.change 编辑后高亮清除。
**坑**=①rune 区间心算又错：rune[1,3) 在 𝌆 开头串里是「一二」非「二三」（𝌆 占 rune 0）——区间断言先写清每个 rune 是谁②定位句柄里 setState 异步渲染须 findBy/waitFor，setSelectionRange 同步 DOM 立即可断言——同一次 locate 两种时序③textarea 底色 !important 内联压不过，须同优先级类+源码序在后。
**门禁**=tsc/eslint 0；EditorPanel 域 11/11；全量 ci.ps1 绿；drift OK@705；版本三处 4.343.0；产物=exe 50842624B SHA256=49381cbd2b5af04af05a124b6678c4916342743b7758e3ab32d564a790338ecb（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.343.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十八迁）+progress/todos。
**未做（下刀）**=真机走查清池班（镜像对齐真实字体/滚动条下需真机目检，通过后再议持久全量高亮）；技能 ≥5 次核数 ≈09-30；t2 余项按反馈。

## v4.342.0 · t7 观察池「多章对比」最小形态：分析面板章际对比（2026-09-19）
> 观察池条目的最小可用形态；纯前端两文件，零新绑定 705。情感曲线（v4.341）给全书节奏，本刀补「第 N 章比第 M 章强吗」的章际质量对比。
**落地**=①ChapterAnalysisPanel 增可选 chapterOptions：提供且排除本章后可选 ≥1 时渲染「章际对比」区（缺省整体隐藏，其他挂载点零影响）；Select 选对比章拉其 V2 → 差值表（综合/节奏/代入/连贯+情感强度）；Δ 语义=四质量维上绿下红、情感强度中性不判色、缺字段与 0 差值不着色；对比章未分析→行内提示不 toast；V2 无 scores→「对比章分析无评分数据」；切章/关面板重置②CreatePage 传 flatNodes 章号清单。
**测试**=Panel +3 共 11/11（未提供隐藏/下拉排除本章+差值表 7.8 vs 6.0 Δ-1.8+按章号调用/未分析行内提示）。
**坑**=①flattenTree 返回 TreeNode 章号在 .node 下——tsc 抓住 map(n=>n.order_index) 的 undefined 运行期隐患②antd Select 的 mousedown 必须落内层 role=combobox 元素（根 div 不开下拉），dropdown 渲染在 portal 需 findByTitle 等待，选项点击 jsdom 可用。
**门禁**=tsc/eslint 0；Panel 域 11/11；全量 ci.ps1 绿；drift OK@705；版本三处 4.342.0；产物=exe 50841600B SHA256=3ade927983d1b41e7ac1a6351f72302169e0e52d1e1470bec5db61318bec41dd（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.342.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十七迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.342 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；t7 观察池仅剩编辑器 overlay 持久高亮（镜像背景层中刀候拍板）。

## v4.341.0 · t7 观察池「情感曲线图形化」：全书体检面板情感弧线折线（2026-09-19）
> 观察池条目（v4.325 记录）；纯前端单文件 BookHealthPanel，零新绑定 705、零新依赖（纯 SVG 不引图表库）。
**落地**=①体检面板情感曲线区（逐章表与伏笔 findings 之间）：默认「生成曲线」按钮按需触发——N 章=N 次 V2 读盘 RPC，交给作者决定不随体检自动跑②逐章读 NovelChapterAnalysisV2：命中 emotional_arc.intensity（钳 0-10）收集 {章号,强度,主导情绪}；缺档/无弧线/报错跳过并计数（缺档只报「尚未分析」不触发重建，循环安全）③纯 SVG 折线 EmotionCurveSvg：x=章号线性、y=强度反转、0/5/10 参考线（5 虚线）、相邻点直连——未分析章间隙以长段如实呈现不插值、圆点 title tooltip（章号·情绪·强度）、viewBox 自适应④诚实不足态：<2 章可绘制提示「需 ≥2 章有情感弧线数据；本次拿到 N 章，M 章跳过」；跳过数>0 折线下计数说明⑤「重新体检」重置曲线；按钮态生成/刷新。
**测试**=BookHealthPanel +4 共 7/7（零 V2 调用前置断言/SVG 两点+tooltip 文案/不足提示+跳过计数/默认不拉 V2）。
**坑**=①数据点数与跳过数断言须按报告章数编排——2 章报告跳过 1 章后只剩 1 点走「不足」分支而非折线分支②未分析章 V2 绑定报「尚未分析」不触发重建（重建是标注绑定的行为），逐章循环安全不引爆 LLM。
**门禁**=tsc/eslint 0；BookHealthPanel 域 7/7；全量 ci.ps1 绿；drift OK@705；版本三处 4.341.0；产物=exe 50838528B SHA256=9067547b1fa0f7207370d98cc113509307d0dea93650b84e1969ac306e9ffaa9（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.341.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十六迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.341 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；t7 余项（overlay 持久高亮有意裁剪/多章对比形态候拍板）。

## v4.340.0 · t2 观察池「反推任务取消绑定」：AI 反推大纲轮询期可取消（2026-09-19）
> 观察池条目（t2 拆书线 v4.291 任务化体验余项）；纯前端单文件 CreatePage，零新绑定 705——后端取消能力 v4.291 起即备齐（TaskState.TaskID 回传+GaeaTaskCancel 协作取消 ctx 直通 handler），缺的只是前端入口。
**落地**=①取消状态三件：reconstructCancelRef（轮询标志）/reconstructTaskIdRef（Start 回传 taskId）/reconstructCancellable（taskId 到手才 true——精确门控，同步回落路径无任务 id 不出取消钮）②轮询循环加检查点（入循环前+3s sleep 后）：命中→退出轮询+app.TaskCancel 请求协作取消（完成竞态后端报错被吞——等待已停，任务中心可查，消息如实说「已请求取消」）+落「已取消反推等待」return 不弹应用确认③rail「AI 反推大纲」旁 danger「取消反推」钮，finally 复位；novel:auto-reconstruct 自动链路同样受益。
**测试**=CreatePage.test +2（轮询期取消：消息+TaskCancel('tk-c')+确认不弹；同步回落路径不出入口零 Cancel）13 既有全绿；补两处漏桩（NovelB mock 块缺 NovelOutlineReconstruct 同步回落本体/bridge appStubs 缺 TaskCancel）。3s 轮询 sleep 靠全局 asyncUtilTimeout 5000 覆盖无假定时器。
**坑**=①取消请求是「尽力而为」：任务恰在取消前完成是正常竞态，UI 语义=停止等待+尽力取消+如实说请求已发，不做取消结果二次确认（任务中心=任务态权威）②取消入口必须以 taskId 到手门控，不能只看 busy 态（同步回落无 id）。
**门禁**=tsc/eslint 0；CreatePage 域 15/15；全量 ci.ps1 绿；drift OK@705；版本三处 4.340.0；产物=exe 50835456B SHA256=7aaf15a8fcea664ec85969f3048a49ffe1be2ac2054e8d70021fbb2c63c8cb93（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.340.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十五迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.340 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；t2 观察池余项（反向角色补建/骨架 AI 丰富化按反馈）。

## v4.339.0 · t7 观察池「标注定位编辑器光标」：分析面板→正文编辑器定位闭环（2026-09-19）
> 观察池条目（v4.325 记录）按既定习惯开刀；零新绑定 705，纯前端三组件。
**落地**=①EditorPanel 转 forwardRef 暴露 `locate(runeStart,runeEnd)` 命令句柄——rune→code-unit 换算（toRune 逆，代理对按 codePoint 步进）+setSelectionRange+focus（Chromium 聚焦即滚选区入视口），无激活章回 false②ChapterAnalysisPanel 增可选 onLocate：锚定标注行内「编辑器定位」钮（stopPropagation 不触发行内高亮跳转），原「↩定位」标签改「↩高亮」消歧③CreatePage 接线：onLocate 仅面板章=编辑章时提供（gate 跳他章时面板 content 是他章正文，入口隐藏而非点了报错），处理器关面板→复位 gate→locate(pos,pos+length)，未就绪降级提示。
**测试**=EditorPanel +3（𝌆 代理对换算选区聚焦/越界钳文末/无激活章 false）+Panel +3（回传标注+stopPropagation/未锚定无入口/未提供 onLocate 隐藏）；tsc/eslint 0；两文件 18/18。
**坑**=gate 跳转的章≠编辑章（面板 content 可为 gateContent 他章正文）——定位入口按章一致条件提供，不做「点了再报错」。
**门禁**=全量 ci.ps1 绿；drift OK@705；版本三处 4.339.0；产物=exe 50834944B SHA256=447cc8b0452af06a912edbd5009b14a32e830f9f639a0b681b579ca9fa8290ea（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.339.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十四迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.339 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；t7 观察池余项（overlay 持久高亮/多章对比/情感曲线）候拍板或按反馈。

## 非版本刀 · todos 对账清账 + booksource 进度回调乱序根修（2026-09-19）
> 全表对照 git log/CHANGELOG 逐行核查开放项（子代理扫描+人工复核），起因=「TaskCenter 会话维度」行标 ⬜ 实际 v4.229.0 已落、险些重做。
**对账结果**=关 3 行：TaskCenter 会话维度（v4.229.0 SchemaV20+SubmitSpaceSession+过滤面就位 chip 熄灯；chip 显形等首个会话上下文创建点=门控）/审计刀A（v4.165.0 已落仅状态标错）/knip 甄别（v4.268.0 清账，剩余 15 项 KEEP=@public 哨兵在册保留）。改写 2 子句：t4-C4「随 t7」悬置语→已随 t7 收官（面板只读高亮+锚点 v4.325.0；编辑器 overlay=v4.325 有意裁剪入观察池）；造价「剩 §6 等样本」→§6=v4.209.0 已收官（基线源=自有库不引入外部数据=设计裁决）。删 7.3-1「会话级回源欠账」旧注记（v4.333.0 已收口）。补记 7.3-2 缺省 classic→tasks 翻转候拍板（v4.333 出口对账在案 todos 漏记）。**勘误=v4.338.0 未做段「TaskCenter 结构刀」同源过时，按惯例不追改已发条目、以本条为准**。
**booksource 进度回调根修**=fetchChapters 的 OnProgress 曾在计数锁外发射：并发 worker 各持快照乱序到达（全量 ci 实测 [0/3 2/3 1/3]，「单跑绿全量挂」负载 flaky 家族+1）且对消费方构成并发调用（测试侧 append 潜在竞争）。根修=发射互斥（emitMu）下现读计数：回调序列非降、末次=成功总数、消费方免同步；OnProgress 契约注释补纪律（轻量回调）。-count=10 绿；本机无 gcc -race 不可跑（race 门=GitHub Actions）。
**教训**=①「已落地未关」的 todos 行会传导进 release notes 的未做段，下版排刀前先对账②进度类回调的发射序是契约的一部分——计数加锁≠发射有序。

## v4.338.0 · 无参绑定会话语义审计收官 + 死链清理（绑定面 707→705）（2026-09-19）
> todos 挂账「GaeaHistory/ContextView 等无参绑定内核会话语义逐个审计」收官（v4.181.0 起挂池）；审计档 docs/gaea-session-binding-audit-2026-09.md。
**审计结论=需修 0 项，家族关闭**。枚举全部 Gaea* 导出绑定逐个核对 ga.ctrl 会话态读点与前端消费方——核心发现=该应用会话模型构造性保证「看历史必经 ResumeSession 切内核」（currentSessionPath 的 current 标记本身来自 GaeaListSessions 读 c.SessionPath()）：①对话主流水线（History/ResyncEvents/Context/FactBase/ListSessions Current 标记）无参读内核自洽，无「旁路查看另一会话」UI 目标②v4.181 旁路看板族（轨迹/网络/上下文/子代理 runs）无同构残余③提交类/引擎级豁免。**复审口径入档**：新绑定无参读内核会话三条件（跟随实时流的视图/动作类/引擎级），带会话切换的看板一律显式 sessionPath。
**顺手清账两条死链（707→705）**=①GaeaCheckpoints（全仓唯一引用=bridge 声明+mock，UI 回退实走 GaeaRewind）——连带删 CheckpointMeta 类型/门面/TestGaeaCheckpoints/wire 类型/wailsjs 再生，rewind.go 注释自述化（内核能力保留）②GaeaTCCAReport（返回值仅入库 state.tcca 零渲染消费）——连带删 controller 两处拉取+action/reducer/字段+TCCAReport 接口+mock+三处测试 mock 键与 JSON.parse 错误注入用例。
**拍板池新候选**=GaeaSkillDraftFromSession 参数化（唯一无参读全会话做 LLM 加工点；「从历史会话蒸馏技能」功能增量候拍板）。
**坑**=①gen_bindings -names 只打印不写盘（v4.145 在案）bindingNames.ts 手工同步；wailsjs 目录 gitignore 再生后 git status 不显示，验证须看文件本身②删 Go 测试带走 import 唯一使用（_ = strings.Contains）——build 段不测 test 文件，删测试后须 go vet 快验。
**门禁**=go build/vet 0+internal/app 全量 ok；tsc/eslint 0；store 域 38/38；drift OK@705；全量 ci.ps1 绿；版本三处 4.338.0；产物=exe 50833408B SHA256=9d4f70562e1e9e4b82fb4373cc7d4cef2e2de15bca301196baaf67f13ed5b0d5（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=审计档+releases/v4.338.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十三迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.338 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；TaskCenter 会话关联结构刀（另立版本）。

## v4.337.0 · 办公对话流美化三轮收官：对齐 Codex web（降噪+动线+语义归位）（2026-09-18）
> 办公对话流美化线程收口发版（对齐 Codex web 三轮）。纯前端零新绑定 707，零 Go 改动；规格=releases/v4.337.0.md。
**落地十件**=①用户消息气泡化（surface-container-high+细边框，替代无气泡裸文本——轮次边界扫读清晰）②裸思考行 bare 模式（纯思考段不出过程卡，一行裸思考块+行数 meta 进思考行；meta 三语键 reasoning.metaLineOne/metaLines+英文单复数——收口修掉初版硬编码中文「行」）③过程卡完成即收起（运行中自动展开/完成收起一行「已工作 Xs·…」/历史恢复同样折叠/手动干预保留；旧「展开态默认展开」废=垂直噪音主源；耗时角标只在「已工作」缺席时补位）④WorkHeader 三改（轮内有过程卡时完成态「已完成·用时」行让位 doneSuppressed/纯问答轮 0 步完成态不渲染/运行态样式对齐过程条左细线+hover 底）⑤尾随工具段合并（buildSegments 末段纯过程并回前段——store 事件序工具尾追，过程卡按 Codex 语义渲染在正文前，消「答后动作」假象）⑥轮尾登记-only 交付卡补挂（reasoning-first 轮尾无正文段原先丢卡）+omitPaths 跨段去重（防同文件双卡）⑦交付卡统一边框容器（细边框+行分隔线+整卡一圆角）⑧消息操作 hover 淡入（复制/评分后驻留）⑨顶条只留 waiting 档（≤5s 连接档与 WorkHeader spinner 同义不显示；>5s 才出顶条——正常回合运行态无第二行横幅）⑩ToolGroup 文案精简「bash × 3」+title 完整语义（三语新键）+ToolCard 代码块圆角统一。
**测试**=ProcessCard 展开态测试重写钉新行为（完成态折叠 small/非 small 一致+运行中自动展开+bare 无卡头——旧测试钉已废行为且用例数据恰落 bare 分支，全量首跑 1 红）；vitest 3199/3199；tsc 零错；eslint 零告警；三语 +4 键同轴。
**目检**（本班收口补验）=vite dev 9344 ?mock=demo+独立 headless Edge 9345 CDP 驱动真实 UI 两轮五态截图全过（运行中自动展开/完成收起/交付卡容器/用户气泡/顶条无横幅/bare「思考 1 行」实测/Composer 排队提示在位）。
**坑**=①摘要类文案禁硬编码——初版「行」硬编码 en 界面漏中文，一律三语键+注意英文单复数②行为变更须重写钉旧行为的测试（其用例数据可能已落新分支）③杀隔离进程按 user-data-dir/端口过滤勿全杀。
**门禁**=全量 ci.ps1 绿；drift OK@707；版本三处 4.337.0；产物=exe 50837504B SHA256=36d7b3e2ab26886b2f96192d13db16666db7686f18c729ce0c36381f0d89c641（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.337.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十二迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.337 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；绘梦阶段二/缺省形态翻转候拍板。

## v4.336.0 · chapter-gate 通知跳转 + oh-story T5 落库接线（2026-09-18）
> 用户指令「继续」。两件观察池/todos 收口；规格书 进度计划/gaea-gate-jump-t5-wiring-20260918.md。
**甲件 自动门通知可点击**（v4.331 观察池头名）=useChapterGateNotice 增可选 onOpen 回调（ref 保最新回调防过期闭包），通知 onClick 传报告章号；CreatePage gateChapter 覆盖态打开章节分析面板，跨章时 GetChapter 拉正文供标注锚定（失败回退空串）；无回调=纯通知零变化。测试钩子 +1。
**乙件 oh-story T5 落库接线**（todos 欠账本体）=摸底证实资产零代码引用、消费面应为 run_skill/任务子代理（非角色库，数据管道已齐）——①SKILL.md frontmatter `runAs: subagent`+「派发契约」段（子代理执行序=解析 role→read_file `.gaea/skills/novel-agents/agents/<role>.md`→按卡执行；七 role 枚举；主会话 arguments 格式 `role=<角色> <任务>`）②spawn 第五内置模板 `subagent_writing`（写作域角色卡前缀）+boot 映射 novel-agents→subagent_writing（七角色子代理共享 L4 前缀缓存 V5.30）③验收钉 Go +3（skill 包直读仓库资产断言 Scope/RunAs/锚点/七 role；boot 映射；cache 计数 4→5）。**oh-story 余项只剩写前契约硬闸（需先有一键补大纲，等上游）**。零新绑定 707。
**门禁**=全量 ci.ps1 绿；drift OK@707；版本三处 4.336.0；产物=exe 50833920B SHA256=235e3f05ca7ab062ebea99c49e24160f439a599b80db51a648f33bd1419d77ea（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=规格书+releases/v4.336.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十一迁）+progress/todos。
**未做（下刀）**=真机走查清池班（v4.319~v4.336 挂池，须闲置窗口）；技能 ≥5 次核数 ≈09-30；绘梦阶段二/缺省形态翻转候拍板。

## v4.335.0 · 办公输入动线对齐 DSH/Codex：运行中 Enter=排队，插话改显式（2026-09-17）
> 用户报告「办公板块消息输入的排序/插话/撤回不见了，直接插入正在跑的对话中」。归因=**非回归**：队列功能本体（v4.200/201 排序/插话/撤回/锁定）一直在且测试全绿，丢的是入口——v3.6.0（2026-08-28）把运行中 Enter 从排队改成插话（对齐豆包边跑边改），排队退到 Clock 小按钮，用户 Enter→直插当前对话→队列卡从未出现。用户拍板「按 DSH/Codex 方式处理」。
**落地**（纯前端零绑定 707）=①运行中 **Enter=排队**：入发送队列→队列卡（拖拽排序/撤回编辑/单条·全部插话/全部取消/发送中锁定）回主动线，回合结束 FIFO 自动派发②**插话改显式**：Alt+Enter 直插当前回合（边跑边改保留），队列卡 Zap/全部插话不变③Shift+Enter 纠正/Esc 停止/Clock 按钮语义零变化④发送钮 title 三态换 sendQueuedTitle；placeholder 运行中提示三语更新（Enter 排队 · Alt+Enter 插话 · Shift+Enter 纠正）⑤composer.steerTitle 键删（维持 0 死键）；Composer 四处重复组装抽 assembleSubmit/enqueueSubmit/pushInputHistory，入队统一进输入历史。**测试**=Composer.queue +3（Enter 排队不走 steer/Alt+Enter 直插不入队/Shift+Enter 纠正回归锁）+I18n 断言对齐+sendQueuedTitle 三态钉；composer 域 22/22。**顺带修 flaky（v4.334 遗留）**=CI 首跑 TestKnife8_GenerationAxis_EndToEnd 快败（标注 items=0）：门协程落盘序 V2→memories→annotations→伏笔，测试只对 V2 waitFor、③④⑤裸读输给 IO 时序（昨日 3 连跑全赢属侥幸）——③④⑤ 改 waitFor 终态后 -count=3 绿；教训=**异步落盘链上读到 A≠读到 B，断言等终态不赌时序**。**坑=后台跑 ci.ps1 别接管道 tail（吞退出码假绿，v4.334 首跑被 tail 的 exit 0 骗过）**。**观察**=GenUI action 运行中仍走 steer（机器动作语义与用户输入分线，不改）。
**门禁**=tsc 零错/eslint 零告警/全量 ci.ps1 绿/drift OK@707/版本三处 4.335.0；产物=exe 50832384B SHA256=556c906f1efed83273dd72d7b289610abf1c90c1287560393cd0e3ddf446ccd0（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
**文档**=releases/v4.335.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（五十迁）+progress/todos。
**未做（下刀）**=真机走查清池班；技能 ≥5 次核数 ≈09-30；绘梦阶段二/缺省形态翻转候拍板。

## v4.334.0 · revolution 刀8 收官：跨模块验收+回退演练+版本三处漂移根治（2026-09-17）
> 用户指令「继续优化迭代 gaea」。三线并发子代理+主代理收口；契约先行（规格书随刀入库 进度计划/gaea-rev-knife8-acceptance-20260917.md）。**本刀合入=小说 revolution 刀 1–8 全线收官**（race 项 v4.331 已清，刀8 余项=跨模块验收+回退演练本刀收口）。
**论点**=各模块单测在位但跨模块串联零覆盖：生成主轴端到端（CreateChapter→done+aiTaste→自动门→分析 V2 落盘/记忆/标注/伏笔同步→场景物化）全仓无贯通测试，**自动门成功路径（analysisDone=true/foreshadowSync）从未被测**（既有只测 nil-engine 降级）；叙事审批链 app 级 0 测试；回退三径（app 级快照往返/v3 迁移 `_v3_backup` 非破坏）零断言；性能判据（AI 味<1s）与去味保真（字数/实体不变）无钉。
**落地三线 Go +7**（全部零生产代码改动、缝合面现成）：**线A novelstyle**=性能钉（~5200 rune 章 ScoreTextNoRef 单次 3.3~5.0ms，判据<1s 余量 200 倍）+去味保真（100→27 分、rune 漂移 0、实体名逐字保留 3 次、句数恒等）。**线B 生成主轴**=herdsman→httptest SSE **双路桩**（按「小说分析编辑」系统提示词标记分流）贯通全链——章落盘/自动门分析成功/V2 落盘 Overall 归一 8.2/memories 6 条/annotations 4 条 pos≥0 命中/伏笔 planted 入库/rebuildScenesFromBlob 真跑重置物化/**chapter-gate 事件经 httpbridge SSE 订阅捕获**（analysisDone=true、planted/created=1/1）/桩计数一次生成一次分析；50 章书级体检（奇脏偶净 25/25、AnalyzedChapters==3、~0.7s 零 LLM）。**线C 审批链+回退**=Settle(false) **AI 建议无一自动入库**（版本 0/账本空）→Settle(true) 版本 1→journal Replay 与 GetNovelState 逐字段一致→二次结算版本 2 只增→坏 JSON 拒；app 级快照往返；**真 v3 夹具**迁移非破坏（场景 ≥1/Stitch==原文/`_v3_backup` 保留）。
**版本三处漂移根治（独立发现）**=app_info.go/versioninfo.rc 停在 4.323.0（v4.324 起十版未跑 sync-version.ps1，exe 内嵌版本落后十版）而闸门不在 ci.ps1 从未拦截——三处同步 4.334.0+**ci.ps1 新增 version drift check 段**防再犯。**顺手清账**=eslint 5 条存量告警清零（EXTRACT_OPTIONS 归位 novelOptions.ts+4 处未用变量/导入；MonteView `_cpm` 缝合面保留）。
**测试**=Go +7（线A 2/线B 2/线C 3），同包 TestKnife8_* 前缀防撞；既有邻域（TestAutoGateReport|TestBookHealthCheck|TestCreateChapter|TestNovelState 系）全绿零回归。**坑**=①RetryJSON 的 ExtractJSON 对「回复整体即裸 JSON」判未找到——桩回分析必须包 ```json 围栏②rebuildScenesFromBlob 对无场景章是 no-op（物化靠读路径惰性），验「重置+物化」须预置旧场景③project.Create 现直接写 v4 标记，旧夹具 v3 假设过时（newMaterializeApp 里 MigrateV3ToV4() 实为 no-op），真 v3 须手动摘 `.gaea/v4`。
**门禁**=go build/vet 全过、go test ./... 全量全绿、tsc -b 零错、eslint 零告警（5→0）、vitest 3185/3185、drift OK@707（零新绑定）、版本三处 4.334.0（新闸门过）；产物=exe 50832384B SHA256=4d2a01c5c84a8e6696fbe845d3583d0591ec79f26e083840dc01110b31c081bc（releases+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.334.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十九迁）+progress/todos+revolution 刀8 勾选（规划收官对账）。
**出口对照**=长文一致性（50 章体检+深检纯函数既有）✅/去味分降且字数实体不变（线A）✅/作者是上帝无一自动入库（线C）✅/journal 可回放（线C）✅/性能<1s（线A 3.3~5ms）✅/每刀可回退（线C 三径+rewrite Reset 三条 e2e 既有）✅。
**未做（下刀）**=真机走查清池班（v4.319~v4.333 挂池，须用户闲置窗口）；技能 ≥5 次核数 ≈09-30；绘梦阶段二与缺省形态翻转候拍板。

## v4.333.0 · 阶段七出口对账 + 7.3-1 会话级回源收口（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-stage7-audit-resume-20260917.md）。三门刀已全落（7.1 v4.312/316；7.2 v4.313/317/319；7.3 v4.318/332）——本刀=出口判据逐条复核（诚实注记挂账）+收掉 7.3-1 唯一结构欠账。
**会话级回源**=v4.318 遗留：Session 路径已落库（palette 存 currentSessionPath 实证）但 resume seam 活在 gaea App 树内，收件箱在首页拿不到，V1 只回板块粒度。**落地**=①新模块 gaea/lib/pendingSessionResume（模块级单例 pending+requestSessionResume 派发 window CustomEvent「gaea:resume-session」——gaea 板块 keepAlive=true 两态通道：已挂载走事件直达/冷启走挂载消费 pending 兜底）②gaea App 挂载 effect：consume pending+事件监听→resumeRecentSession（内部按 projectGroups 解析跨项目切工作区后恢复）；**ref 保最新回调防过期闭包**（projectGroups 异步加载，[] 依赖订阅一次）③TaskInboxBoard/Panel +onResumeSession 透传——源会话按钮有回调且带路径走精确回源，否则回退 V1 板块粒度（既有行为零变化）；ModuleLauncher/TasksFirstHome 接线（request+onNavigate('gaea')）。零新绑定 707。
**出口对账**（docs/gaea-stage7-plan §5 门表+总判据+7.3-1 判据②全部更新）：7.1 已落（挂账=本地引擎建议真实采纳样本待真机）；7.2 已落（挂账=录制复跑 ≥80% 抽样真机+结晶 ≥5 次真实调用 ≈09-30 清池核数）；7.3 已落（②会话级本刀收口；缺省 classic→tasks 翻转候拍板）；7.4 已清（t6/t7/Gate/画室一）；总判据①②③④✅、⑤真机走查欠账（v4.319~v4.332 挂池，列下一班清池首项）。
**测试**=前端 +5（pendingSessionResume 3：派发/一次性消费/空路径防御；TaskInboxPanel +2：精确回调不再板块粒度/无回调回退零变化）；既有 11 用例零改动。**坑**=①测试文件尾 rfind('})') 追加吞 describe 闭合——**同坑两日三犯**，规约：测试追加一律在末尾按「闭合块后整体 append」并在 append 前数一遍尾括号②闭包过期：订阅 effect [] 依赖引用了依赖异步数据的回调——ref 保最新。
**门禁**=go build 全过、go test ./... 全量复跑全绿、tsc -b 零错、eslint 零告警、vitest 3185/3185、drift OK@707（零新绑定）、版本三处 4.333.0；**ci.ps1 首两跑分别撞 go test/vitest 段瞬时失败（单跑皆绿、清残留 node 进程后第三跑 CI OK——环境竞争先例+2）**；产物=exe 50832384B SHA256=89d1de3d332d19ca77b42952708bef88bfac0cc45c355c28052caa28773c0ba2（releases/gaea-v4.333.0.exe+SHA256SUMS-v4.333.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.333.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十八迁）+progress/todos+阶段七规划对账更新。
**未做（下刀）**=真机走查清池班（⑤欠账+各刀挂池项，须用户闲置窗口）；技能 ≥5 次核数（≈09-30）；刀8 余项与绘梦阶段二候拍板。

## v4.332.0 · 7.3-2 板块降级为任务视图（阶段七「层跃升」刀：任务优先首页+能力视图+回退开关）（2026-09-17）
> 用户指令「开始吧」（在 7.3-2 解锁讨论后明确提前拍板——原零功能周窗口 ≈09-23）。契约先行（规格书随刀入库 进度计划/gaea-tasks-first-home-20260917.md；阶段七规划 §3 7.3-2；身位依据平台调研 §1.1：任务持久化红海、「收件箱第一界面+板块降级」无人发布）。**形态标志：本版=阶段七「层跃升」版本（nextgen §10 口径；版本号续 v4.x 列车不跳号）。**
**论点**=现状首页板块入口优先，打开第一问「用哪个工具」；7.3-2 翻转为「什么在等我」——收件箱+最近上下文成为首屏，板块入口降为「能力的视图」。
**裁决**=形态开关 homeLayout（classic/tasks）入 appStore 持久化（localStorage，缺省 **classic**=现有体验零变化，层跃升由开关启用）；TaskInboxPanel 抽内联 **TaskInboxBoard**（Modal 壳保留薄包装，props/DOM/既有 9 用例零改动）；TasksFirstHome 两空间共用 space 感知；回退开关双位（设置页 HomeLayoutPanel 正式位+首页 SpaceSwitch 条快捷钮互切）；**零新绑定**（GaeaTaskInbox* 四绑定复用）；manifest/pageRegistry 零改动（形态合并引擎零改动原则）；三语 +12 键 zh=en=zh-TW。
**落地**=①appStore +HomeLayout 类型/态/setter/persist 守卫（非法值回退 classic）②TaskInboxPanel.tsx 抽 TaskInboxBoard（active 拉取位+onPendingChange 计数上报，Modal 标题徽标同源）③新 components/TasksFirstHome.tsx：SpaceSwitch+轻 masthead（印章/标题/lede+切回经典钮）→ 收件箱大区（TaskInboxBoard 直嵌）→ 最近会话（SessionList）→ 能力视图（deriveLauncherModules 全量 chips 点击直达——**藏≠删**）→ 纯同步口径脚注④ModuleLauncher：homeLayout 分支（classic 走既有 Desk/Garden 零变化）+SpaceSwitch 条 HomeLayoutToggle 快捷钮⑤设置页 HomeLayoutPanel（SegmentedRow 同款）挂 SettingsPage⑥locales 三语 12 键。
**测试**=前端 +10（TasksFirstHome 4：收件箱直嵌拉取/能力 chips 全量点击直达/切回经典写 store/最近会话；ModuleLauncher 分支 2：缺省 classic 零变化+tasks 形态渲染与快捷钮互切；appStore 3：缺省/持久化/非法值守卫）；**既有零回归实证**：TaskInboxPanel 9/9 零改动、ModuleLauncher 13/13 零改动（缺省 classic 分支原路径）、SettingsPage/appStore 既有全绿。
**坑**=①测试文件尾部 rfind('})') 追加会吞原 describe 闭合（Unexpected EOF）——追加后必须数括号或直接读尾②TasksFirstHome 用 useT 须 LocaleProvider 包裹（ModuleLauncher.test wrap 先例）③ESM 测试无 require——守卫函数走顶部 import。
**门禁**=go build 全过（零 Go 改动）、tsc -b 零错、eslint 零告警、vitest 全量、drift OK@707（零新绑定）、版本三处 4.332.0；产物=exe 50831872B SHA256=80dacda4f818deeb47827006e944293bb045b479087611c275345ec29d9dd7eb（releases/gaea-v4.332.0.exe+SHA256SUMS-v4.332.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.332.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十七迁）+progress/todos+阶段七规划 7.3-2 状态更新。
**出口对照**=①收件箱+最近任务成为首页首屏（tasks 形态）✅；②既有板块全部入口无损（能力 chips 全量直达，测试钉死）✅；③回退开关（设置页+首页快捷，即时切回经典）✅；④classic 零回归（既有测试零改动全绿）✅。
**观察池**=「最近任务」独立区（doing/done 档已覆盖）；能力 chips 分组/搜索；任务拖拽排序；晨报/脉息在 tasks 形态取舍；默认形态翻转（用户试用后拍板）。
**未做（下刀）**=阶段七出口判据复核对账（7.1-7.3 三门收口）；刀8 余项（跨模块验收+回退演练）；绘梦阶段二提案候拍板。

## v4.331.0 · 自动门可见化 + CI race 扩面（刀8 race 项收口）（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-gate-notice-race-20260917.md）。7.3-2 仍锁窗（≈09-23）。两件收口：①v4.326 观察池第一项——chapter-gate 事件零前端消费；②revolution 刀8 第三项——CI race 门禁只盖办公核心包。
**论点A**=自动门（v4.326）异步跑完只落 slog+事件，用户毫无感知——「生成→体检→分析同步」主轴最后一环（知会作者）缺失。**论点B**=race 门禁现盖 gaea/agent·tool·control+novelstyle·novelcontext·narrative；chapter/analysis/promptstore/novelgate/rewrite/characterstate 六个 revolution 期新生包无并发防线（自动门/覆盖缓存/状态机都在其并发路径上）。
**落地前端**=useChapterGateNotice 钩子（useChapterStream 同款 runtime.EventsOn/EventsOff 通道模式）：chapter-gate report → 轻通知「第 N 章 生成后自动体检：契约 X 项 · 质量 Y 项 · AI 味 Z · 分析已同步」（antd message.info key 防叠 6s，空体检显示完成，分支章带标记，非 report 忽略，浏览器无通道静默）；CreatePage 挂载。**零新绑定 707 不动**。
**落地CI**=.github/workflows/ci.yml race job 追加六包（chapter/analysis/promptstore/novelgate/rewrite/characterstate）；internal/app 注释不进（race 下已知 TempDir 清理竞争 flaky 先例，候单独评估）；六包本地普通模式全绿。revolution 刀8 行补注（race 项收口，余=跨模块验收+回退演练候排）。
**测试**=前端 +3（文案拼装三分支/订阅触发+防叠+非 report 忽略+卸载退订/无 runtime 静默——坑=vi.stubGlobal('window') 整窗替换会炸 react-dom 的 document 依赖，只换 runtime 属性）。CreatePage 既有测试零破坏。
**门禁**=go build 全过、六包全绿、tsc -b 零错、eslint 零告警、vitest 全量 3172/3172 一次全绿、drift OK@707（零新绑定）、版本三处 4.331.0；产物=exe 50824704B SHA256=a5e239f24897a858b628dcd3c8a18ccbafbd073f3831172300a3cfa6b8a63198（releases/gaea-v4.331.0.exe+SHA256SUMS-v4.331.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.331.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十六迁）+progress/todos+revolution 刀8 行补注。
**观察池**=通知点击跳 t7 章节分析面板；internal/app race 扩面评估；刀8 余项（跨模块验收+回退演练）候排。
**未做（下刀）**=7.3-2 板块降视图（≈09-23 解锁）；刀8 余项候排。

## v4.330.0 · 刀7续收官：POV 视图（场景圣经消费面）（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-pov-view-20260917.md）。前置核对=场景生成按钮已落（v4 场景工程线）/全文脑图/文风指纹在位——刀7续仅剩 POV 视图，本刀合入即 revolution 只剩刀8。
**论点**=POV 视角掩码（刀2 场景圣经核心差异化：POVView=已知/HiddenFacts=不知情且不得泄露）只在生成链内部消费——用户看不到「这个场景通过谁的眼睛在看、哪些信息被刻意瞒住」，视角纪律不可审视。
**裁决**=+1 绑定 NovelSceneBibleView(chapterNum, sceneID)（706→707，NovelB）；sceneID 非空=该场景编译（不存在报错）/空=整章合成（BuildSceneBibleFromChapter，v3 与纯 blob 章可用）；视图 camelCase typed 切片恒非 nil（前端零判空）；编译静默降级契约——空区段如实空展示不编造。
**落地Go**=novel_scene_bible.go：SceneBibleView/SceneBibleCharView wire 投影+双路编译。
**落地前端**=SceneBibleDrawer（POV 头「通过谁的眼睛」+核心区已知（绿底）/不知情（红字）对照+角色卡含 t5 状态机回灌（currentState/careerMain/Sub）+伏笔约束+时间锚点/主线+确定性编译脚注）；ChapterEditor 场景操作行「视角」按钮（EyeOutlined，sceneIds 门控）挂抽屉；bridge/mock/锁 529→530/bindingNames 707。
**revolution 勾选对账**=GenerationGate 闭环（v4.326 实落）与刀7续（场景生成/脑图/指纹核对+本刀 POV）补记勾选——文档只剩刀8（验收演练/race 门禁 CI 侧）。
**测试**=Go +3（整章视图形状恒非 nil/未知场景报错/无项目报错；POV 掩码语义由 novelcontext 包测试钉死此处只钉 wire 投影）+前端 +3（对照区渲染/空区段全知视角+无隐藏约束/错误态重试）；ChapterEditor 既有 7/7 零破坏。
**门禁**=go build/vet 全过、app+novelcontext 全绿 -count=1、tsc -b 零错、eslint 零告警、vitest 全量 3169/3169 一次全绿、drift OK@707、版本三处 4.330.0；产物=exe 50823680B SHA256=4ea486520055b3ca160c7fe5336414efbc8e5408535020544974e83528494eeb（releases/gaea-v4.330.0.exe+SHA256SUMS-v4.330.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.330.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十五迁）+progress/todos+revolution 勾选对账。
**出口对照**=场景「视角」抽屉可见 POV 已知/不知情对照 ✅；整章合成路 v3 章可用 ✅；空区段诚实 ✅。
**观察池**=Render 全文预览；POV 切换沙盒（改 POVCharID 重编译）；多场景并排；隐藏事实来源溯源。
**未做（下刀）**=7.3-2 板块降视图（≈09-23 解锁）；刀8 验收演练+race 门禁（CI 侧另配）候排。

## v4.329.0 · 绘梦阶段一刀 E：画室消耗（月度聚合+折叠面板，阶段一全清）（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-studio-usage-20260917.md；规格 docs/gaea-dream-studio-nextgen-2026-09.md §4 刀 E 轻量收尾）。**本刀合入=绘梦阶段一（A 资产面板/B 参考槽/C 指令编辑/D 配图 v2/E 画室消耗）全清**。
**论点**=台账已逐条记 Cost（目录单价）+CreatedAt 但没有聚合视图——用户看不到「本月画了几张、花了多少」。
**裁决**=+1 绑定 ImageHubMonthlyUsage(space)（705→706，ImageB）；聚合口径=当月（本地时区 CreatedAt 前缀 YYYY-MM）×model+backend 分组；**单价以台账记录为事实源**（记录缺价回落目录表）；可解析「X CNY/张」才估算（张数×X），裸"0"/"0 CNY/张"=免费，"未定价"/空=unpriced 计数诚实不猜；合计全免费→「0 CNY（本地免费）」，有未定价→注明；文案「画室消耗/创作记录」不用积分话术（规格原文）；面板 space 固定 play（work 空间走成本归因）。
**落地Go**=imagehub_usage.go：ImageHubMonthlyUsage（绑定入口 gaeaCwd 与 ImageHubAssets 同构）+imageHubMonthlyUsageAt（cwd 参数化供测试）+parseUnitCostCNY；空台账=零报告不算错。
**落地前端**=StudioUsagePanel（Collapse 默认收起：标题带总数+合计；展开=按模型行 张数/估算/未定价如实+口径脚注；失败静默收起）；ImageGenPage 顶栏下方挂点；bridge/image.ts +StudioUsageView；mock/imagehub +1；spaceBindings play +1（锁 528→529）；bindingNames 再生（706）。
**测试**=Go +4（当月过滤跨月剔除/分组估算算术〔glm 2×0.1=0.20；krea2 免费 0 CNY；grok 未定价留空〕/记录单价优先目录回落/单价解析矩阵含裸 0）；前端 +4（标题聚合/按模型行/空月引导/失败静默）。
**坑**=①裸 "0" 单价初版没被解析成免费（只认「X CNY/张」形态）——测试矩阵钉住②gaeaCwd 走全局 ga.cfg 不可注入——cwd 参数化内函数供测试（ImageHubAssets 同款口径由绑定入口负责）③mock 尾部 rfind 锚点吞掉前方法闭合括号（插入后必 tsc 即炸即修）。
**门禁**=go build/vet 全过、app 全绿、tsc -b 零错、eslint 零告警、vitest 全量、drift OK@706、版本三处 4.329.0；产物=exe 50809856B SHA256=90e9ec5627b7c7304a11da30194971d528d87cb06cb15f563e3bbb25db7aad78（releases/gaea-v4.329.0.exe+SHA256SUMS-v4.329.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.329.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十四迁）+progress/todos。
**出口对照**=画室侧「本月画室消耗」折叠面板可见（张数+估算+免费/未定价如实）✅；本地免费显示 0 ✅；零计费动作纯只读 ✅；**绘梦阶段一全清** ✅。
**观察池**=按来源板块细分；单价表动态化；消耗趋势；work 空间并入成本归因。
**未做（下刀）**=7.3-2 板块降视图（≈09-23 解锁）；阶段二规划拍板（画室长出手脚：模板市场/资产复用/一致性进阶）。

## v4.328.0 · 绘梦阶段一刀 D 核心片：章节配图管线 v2（角色参考图 + 风格槽）（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-scene-illustration-v2-20260917.md；规格 docs/gaea-dream-studio-nextgen-2026-09.md §4 刀 D）。取核心片=GenerateSceneIllustration 参考图+风格槽贯通（书封参考/素材库变体/灯箱共用留后续片）。
**论点**=场景插图纯文生图：角色一致性只靠外貌文字（刀 B 参考槽没接到配图链），风格句写死。
**裁决**=签名扩展 `GenerateSceneIllustration(chapterNum, optsJSON)`（v4.139 Save 先例，空串零行为变化，**绑定面 705 不变**）；opts={"characterIds":[],"style":""}；参考图来源=项目角色 PortraitURL（data URL 直用/本地路径读盘转 data URL/无立绘诚实跳过）；**参考路由按后端能力**（comfyui/herdsman 附 RefImages img2img+SceneRefDenoise=0.6；其它后端不附+refNote 诚实降级提示不静默）；风格槽替换风格描述（16:9 构图固定保留）；返回 map 增 refNote 键恒回。
**落地Go**=chapter.Agent +GenerateSceneIllustrationV2（refs/style 参；旧签名委托零变化）；chapter_handler 签名扩展+opts 解析+sceneIllustrationRefs 路由+readImageFileAsDataURL（8MB 上限+MIME 推断）。
**落地前端**=ChapterIllustration 弹窗加风格输入+角色勾选（GetCharacters 名单，至多 8 个展示）+refNote 浅色说明行；bridge/mock 双参同步。**修复挂载效应 bug：run 依赖 refIds/style 后改选项会自动重跑（每次键入触发生成）——didMountRun ref 守卫恢复「打开即一次」语义**。
**测试**=Go +7（风格注入/空 opts 零变化/坏 opts 拒绝/参考路由三态〔comfyui 附 img2img·非参考后端降级提示·无立绘跳过说明〕/路径立绘转 dataURL）；前端 3 改 5（双参断言/v2 选项与重试/refNote 有无两态）。坑=测试 fixture 须 a.ctx=context.Background（agent 持 a.ctx 直透后端，nil 在 fake 的 ctx.Err() 解引用 panic）；fakeImageBackend +lastReq 捕获字段。
**门禁**=go build/vet 全过、chapter+app 全绿（app 首跑并行环境 flaky 复跑两次绿先例+1）、tsc -b 零错、eslint 零告警、vitest 全量 3162/3162 一次全绿、drift OK@705、版本三处 4.328.0；产物=exe 50796032B SHA256=070f39f84f964b7d129b59e09ba68023d11a7c892d8d8932c4e736377a656c39（releases/gaea-v4.328.0.exe+SHA256SUMS-v4.328.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.328.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十三迁）+progress/todos。
**出口对照**=配图弹窗选风格+勾角色→opts 透传→参考路由诚实（附/降级/跳过三态可见）✅；空 opts 旧行为零变化 ✅；挂载一次语义保住 ✅。
**观察池**=GaeaGenerateBookCover 参考图与风格槽；素材库变体/替换/设为书封；灯箱共用；denoise 用户可调；多参考权重。
**未做（下刀）**=绘梦阶段一收尾（刀 E 画室消耗轻量面板）或 7.3-2 板块降视图（≈09-23 解锁）。

## v4.327.0 · 绘梦阶段一刀 C：指令编辑「改图」MVP（云端先行）（2026-09-17）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-instruct-edit-20260917.md；规格 docs/gaea-dream-studio-nextgen-2026-09.md §4 阶段一刀 C）。前置核对=刀 A 资产面板（AssetStudio）/刀 B 参考槽（RefImages img2img）/刀 E 模型目录分层均已落——**刀 C 指令编辑是真缺口**（现状「改图」=把结果填回图生图整幅重绘，无「原图+人话指令」语义编辑）。
**论点**=编辑能力是旗舰标配（调研 §1.1/§1.2）；gaea 后端已有 OpenAI 兼容图片面但缺 /images/edits——Qwen-Image-Edit 系云端网关的标准暴露面。
**裁决**=复用 ImageGenerationRequest（Mode="edit"+InitImage=原图+Prompt=指令，零新字段）；OpenAI 兼容后端 multipart POST /images/edits；GLM/ComfyUI 诚实拒绝（本地档 B 计划）；**app 层不加后端门**（透传按后端能力诚实报错——img2img 既有门不动）；**零新绑定**（GenerateMedia paramsJSON 加 mode，绑定面 705 不变）；前端 V1=弹窗对照（原图|新图）+用到画布（并入 results/history 走既有保存/台账链）；mask/保留区域观察池。
**落地线A**=image_openai.go：editImage 分支（data URL 解码→multipart model/prompt/n/size+image 文件名按 MIME）+parseImageResponse 提取（generation/img2img/edit 三分支共用，行为零变化）+decodeDataURLBytes；image_glm.go edit 并入拒绝（文案区分指令编辑）；image_comfyui.go edit 前置拒绝（不静默降级为图生图——编辑语义≠整幅重绘）。
**落地线B**=GenerateMedia：mode="edit" 原图必填校验（「指令编辑需要原图」）+ 既有 prompt 校验；t2v/img2img 门零变化。
**落地线C**=MediaParams.mode +'edit'；新 InstructionEditModal（指令输入+对照区原图|新图并排+meta+错误 Alert+用到画布/关闭；后端落盘+台账后已就绪，前端只并入画布）；ResultStage +onInstructEdit Action「指令编辑」（单图/网格两视图，icon HighlightOutlined 与「改图」并列）；ImageGenPage 接线（instructEditSource 状态+onApply=setResults/setHistory 前插镜像 queue 成功路径）。
**测试**=Go +5（ai 层 4：edit multipart 形状〔httptest 断言端点/表单字段/文件/n+size〕/缺原图与非 data URL 报错/GLM 拒绝/ComfyUI 拒绝；app 层 1：缺原图拒绝+带原图透传出结果落盘带回路径）+前端 +4（生成透传/用到画布回调〔prompt【编辑】前缀〕/错误态/空指令守卫）；ResultStage 既有 5/5 零破坏。
**门禁**=go build/vet 全过、ai+app 全绿 -count=1（156s）、tsc -b 零错、eslint 足迹零告警、vitest 全量、版本三处 4.327.0；**绑定面 705 不动（零新绑定刀）**；产物=exe 50785280B SHA256=855b56f1ee1dc0f05959e6f004d40a828569c9d2ee33d0dcbd155e78a0499ae0（releases/gaea-v4.327.0.exe+SHA256SUMS-v4.327.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**坑**=JSX 三元表达式内首个匹配 `<AssetLibrary` 插挂点会破坏单父结构（挂点必须找弹窗区锚点）；antd Modal open 由父控——组件测试不断言 onClose 后内容卸载。
**文档**=规格书+releases/v4.327.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十二迁）+progress/todos。
**出口对照**=结果卡「指令编辑」→指令→对照→用到画布全链可达 ✅（真机走查挂池）；三后端诚实口径（openai 实现/glm·comfyui 拒绝）✅；零绑定面 ✅。
**观察池**=mask/保留区域；ComfyUI 本地档（Qwen-Image-Edit/FLUX Kontext 工作流）；多图编辑输入；edit-of-edit 谱系；刀 D 章节配图管线 v2 与刀 E 画室消耗轻量收尾。
**未做（下刀）**=绘梦阶段一余项（刀 D 章节配图参考图贯通/刀 E 画室消耗）；7.3-2 板块降视图（≈09-23）。

## v4.326.0 · GenerationGate 闭环收口：生成后自动分析门 + 全书体检（2026-09-16）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-gen-gate-closure-20260916.md）。阶段七 §4 在册项；7.3-2 仍等零功能周窗口（≈09-23）。
**论点**=revolution 未勾项「GenerationGate 完整闭环」：生成 done 只挂去味+AI 味分+角色提取，**分析路从不自动发生**——t1-P2 伏笔自动回收/t3-P1 记忆回填/t4 V2 落盘三条已建成管道的水源只能靠手动点「分析本章」，闭环名存实亡；全书体检=单章门与伏笔 Lint 各自孤立，没有「点一次出全书报告」的聚合面。
**裁决**=自动门**异步非阻断**（extractCharactersAfterChapter 同款 go 位）默认武装——分析是加法数据（幂等/带审计），与角色提取同性质；只跑确定性三路+分析路（一次 LLM），review/consistency 仍留单章门手动（成本纪律）；**分支章跳分析路**（V2 落盘/伏笔登记是主线口径，upsert 会污染主线条目）；结果 emit `chapter-gate` 事件（V1 零前端消费，t7 章节分析面板天然是消费面——V2 保证新鲜）；全书体检=**纯确定性零 LLM**（作者点按秒级出报告，无常驻无定时）。
**落地线B**=①novel_book_health.go：buildAutoGateReport（同步可测：outlineContract+ChapterQualityIssues+aiTaste 白名单豁免后必出；分析路主线章才跑，失败 warn 降级）+runAutoGateAfterGeneration（go 位包装+recover 兜底+emit）；RunBookHealthCheck（逐章确定性三路+字数；伏笔 Lint 复用；V2 覆盖计数；BookHealthReport camelCase typed，S1/S2 记 error 级）②create_chapter_handler done 后挂自动门 go 位③NovelB +RunBookHealthCheck（**704→705**）。
**落地线C**=BookHealthPanel（CreatePage rail「全书体检」）：聚合卡（总章/契约问题章/质量问题章/最差 AI 味/V2 覆盖/伏笔告警数）+最差 AI 味≥60 告警（指引去味复检）+逐章表（契约/质量计数与阻断级红标·AI 味 60+ 红·字数 tabular-nums）+伏笔 findings 列表+零 LLM 说明脚注；bridge+视图类型；mock +1 桩；锁 527→528；bindingNames 再生。
**测试**=Go +5（自动门三例：确定性三路必出/分支跳分析/agent 失败降级不 panic；体检两例：干净+脏章聚合断言〔超长段落 S3 非阻断、契约 S2×1+S3+S4 计数、无大纲节点章不计契约〕+V2 覆盖与空书零章非 nil）；前端 +5（聚合渲染/重新体检/零章空态/失败空态/契约 1）；CreatePage 既有测试零破坏。
**门禁**=go build/vet 全过、internal/app 全绿 147s（-count=1）、vitest 全量、drift OK@705、版本三处 4.326.0；产物=exe 50774528B SHA256=0581c0be3ea4a61f33989eecf4ebd666a287cd648a10ec2a21e5da304aa5f83b（releases/gaea-v4.326.0.exe+SHA256SUMS-v4.326.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**坑**=①analysis.New(nil client) 在 Analyze 内部 nil 解引用 panic——测试须用无引擎真 client（ai.NewClient 空 cfg，LLM 调用走 error 返回）②JSX 泛型不认索引访问（Table<Obj['rows'][number]> 语法错），抽 type 别名③gateOutlineIssues 找不到大纲节点返回空表=无契约可违不计问题章（首版断言想当然写反）。
**文档**=规格书+releases/v4.326.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十一迁）+progress/todos。
**出口对照**=生成一章后 V2/伏笔同步/记忆回填自动发生（分析被调降级路径钉住）✅；全书体检查一次出聚合报告零 LLM ✅；revolution「GenerationGate 完整闭环」勾销 ✅。
**观察池**=chapter-gate 事件 UI 消费（toast/面板）；auto 分析 kill-switch；review/consistency 自动化（成本拍板）；体检 LLM 深检档与报告导出。
**未做（下刀）**=7.3-2 板块降视图（等零功能周≈09-23）；闲庭在册清欠沿既有列车。

## v4.325.0 · t7 前端统一接线收官：分析 V2 消费面板 + 标注高亮视图（2026-09-16）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-analysis-v2-panel-t7-20260916.md）。**本刀合入=小说·MuMu 蒸馏六域前端接线全清（t7 收官）**——伏笔面板 v4.299/重写建议 v4.305 已消费，本刀清掉最后两块零消费资产。
**论点**=分析 V2（v4.301 落盘九维 analysis-v2.json）与标注层（v4.306 NovelChapterAnnotations keyword→rune 偏移）后端早已在位而 UI 面零消费；AnalyzeChapter 绑定困在 drift.ts LegacySurfaceNames 零入口——用户没有任何途径触发分析或看到结果。
**裁决**=新绑定 NovelChapterAnalysisV2 回 types.ChapterAnalysisResult **直连**（PromptTemplateDetail 回 prompt.Template 先例；顶层 snake_case 如实透传前端镜像声明，不为只读面板复制九维子类型树）；缺该章分析诚实 error「尚未分析」（面板空态接分析按钮闭环）；AnalyzeChapter 出 Legacy 转正（AppBindings+mock，t7「接线」字面义；V1 返回忽略，真相源=V2 落盘）；标注高亮 V1=面板内**只读高亮视图**（不动编辑器 textarea——overlay 对 antd 样式同步脆弱入观察池）；rune→code-unit 换算（EditorPanel toRune 逆函数，代理对按 codePoint 步进）+交叠裁剪（后段起点钳前段终点）+越界钳制。
**落地线B**=analysis_handler.go +NovelChapterAnalysisV2（ReadAnalysisV2File 按章查找直连；只读，落盘由 AnalyzeChapter 的 agent 链路完成）；NovelB 门面 +1（703→704）。
**落地线C**=①bridge/novel.ts：AppBindings +AnalyzeChapter（drift.ts LegacySurfaceNames 移除）+NovelChapterAnalysisV2 +ChapterAnalysisV2View/AnalysisV2ResultView 视图（嵌套 snake 同落盘契约，全字段缺省防御）②mock/novel +2 桩（V2 单章九维演示样本）③新组件 ChapterAnalysisPanel（antd Modal）：头部综合分大数+三维 Tag+评分理由+chips（节奏/阶段/对话·叙述占比）+meta；分节=钩子/伏笔（埋设·回收徽标+长线+预计回收章）/冲突（烈度+化解%）/情感/角色变化（存活态异常 Tag）/组织变化（稀疏）/情节推进/场景/建议/小结——空节隐藏；标注区 Segmented 双模式=列表（type 徽标+importance+↩定位）/高亮视图（只读 pre-wrap 正文+分类型软底 mark+title 悬停），列表行点击切视图并 scrollIntoView 锚点④CreatePage rail「章节分析」按钮+挂载（content/activeChapterNum 传入）。
**绑定面**=703→704（NovelB 123→124）；AnalyzeChapter 入 facets play；锁 525→527（AppBindings +2）；bindingNames 再生；drift OK@704。
**测试**=Go +3（命中回九维+meta 直连/缺章「尚未分析」/无 agent「请先打开项目」）；前端 +8（面板 5：九维渲染/缺档空态+分析闭环（AnalyzeChapter→重拉 V2+标注）/高亮段换算+交叠裁剪（两段交叠断言第二段起点钳 5）/锚点切换滚动/无章空态不拉取；契约 3）；CreatePage 13/13 零破坏；tsc -b 零错、eslint 足迹零告警。
**门禁**=go build/vet 全过、internal/app 全绿（146s）、vitest 全量、drift OK@704、版本三处 4.325.0；产物=exe 50756096B SHA256=6604f8e06b3f27cd586053794bfe22d8b881851e72210a7e7b0e2aeb22d89528（releases/gaea-v4.325.0.exe+SHA256SUMS-v4.325.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）
**文档**=规格书+releases/v4.325.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（四十迁）+progress/todos。
**出口对照**=分析本章按钮→V2 落盘→面板九维可见 ✅；标注列表+只读高亮+锚点可达 ✅；AnalyzeChapter 出 Legacy ✅；既有消费面零回归（CreatePage 13/13）✅。
**观察池**=编辑器 textarea 内 overlay 持久高亮；标注点击定位编辑器光标（需 ref 联动）；多章对比；情感曲线图形化；V1 wire 前端消费。
**未做（下刀）**=7.3-2 板块降视图（等 v4.318 零功能周≈09-23）；闲庭在册清欠沿既有列车。

## v4.324.0 · t6-C2 提示词工坊第二刀：模板包导入导出（content_hash 三态）（2026-09-16）
> 用户指令「继续优化迭代 gaea」。小刀单线主代理直做（体量一轮未拆子代理）；契约先行（规格书随刀入库 进度计划/gaea-prompt-bundle-t6c2-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md §6.4——「本域最有借鉴价值的一段算法」+§11.4 裁决）。
**论点**=首刀覆盖表是只有一份的本地状态——换机/重装没有搬运手段，内置升级后也没有对账工具；本刀补全量快照 JSON 包+MuMu 三态导入算法（false+同基线→删覆盖行回落内置 kept_system_default / false+基线不同→转自定义 converted_to_custom / true→直接写行 created_or_updated）。
**对 MuMu 升级**=导入行过 promptstore.Validate 真校验（error 级跳行不阻断整包——MuMu 导入零校验云端人工审核当唯一闸门，单机无审核链坏模板会直接毒化生成链）；重复键跳过（首见生效）；裁剪=未知键不建行（工坊 V1 不支持自建键，与 Save 拒未知键同口径）。
**裁决**=导出范围=引擎 Names() 全量含内置行（备份语义）；行内容=覆盖行存在（无论启停）取覆盖行 Content 保真备份；导入不信包内 version（防倒退，本地 Upsert 自增）；哈希 sha256(TrimSpace)[:16] 只是导出侧冗余诊断，对账用 canonical JSON 逐字比对（MuMu 同口径用内容比对不用哈希）；导出落盘走前端 saveExportBlob 双门（壳内 GaeaSaveFileAs/浏览器 a[download]，v4.162 先例）绑定只回 JSON 字符串；导入选文件壳内 pickFileAsFile(['json'])（GaeaPickFiles 系统对话框+.json 后置校验）浏览器动态 input 回退（SkillModal 刀C-3 同款）。
**落地线A**=internal/promptstore/bundle.go（零 IO 纯函数）：ContentHash+SameTemplate（canonical JSON 稳定，map 键序不干扰）+BuildBundle（names×覆盖表×基线闭包→包+统计 total/customized/systemDefault）+ImportBundle 三态决策合并（入参不改写恒非 nil；写行路径统一过 Validate 闸；BundleImportStats/Outcome/Result 三形状）。
**落地线B**=gaea_prompt_store.go +2：PromptBundleExport（BuildBundle→MarshalIndent 回 JSON 字符串）/PromptBundleImport（坏 JSON·version!=1 报错；Applied 才原子落盘+invalidatePromptOverrides；空包/全跳过不落盘不算错）；NovelB 门面 +2 委托。
**落地线C**=PromptWorkshopPanel 顶栏两钮（空态也可见——导入正是空环境恢复路径）：导出=exportBundle→Blob→saveExportBlob('gaea-prompt-bundle.json')取消静默；导入=双门选文件→importBundle→结果 Modal（统计一行+逐行 action 中文 Tag，skipped_* 红/橙标+原因）→applied 后 refreshList+定向刷新当前键；api/prompt.ts +2 函数+BundleImportResult 类型；mock/novel +2 桩；mock-contract +2 用例。
**绑定面**=701→703（NovelB 121→123；PromptBundle* 挂 *App 被默认路由 CoreB——gen_bindings 显式覆盖表 +2 点名归 novel，v4.323 同款坑当日复现即修）；bindingNames 再生；play 锁 523→525。
**测试**=Go +4 函数（线A 三态矩阵 10 分支+round-trip 内核+哈希/canonical 稳定+BuildBundle 统计与停用行保真；线B 导出→清状态→导入覆盖逐字节回魂引擎生效/kept 删覆盖行/防御四例）；前端 +4（导出 Blob 断言/导入结果弹窗逐行+刷新/取消不触发/契约 2）；tsc -b 零错、eslint 足迹零告警。
**门禁**=go build/vet 全过、promptstore+app 全绿、vitest 全量 3143 例（首跑 PptxEditPanel 1 例环境 flaky 复跑绿——隔离复跑 8/8）、drift OK@703、版本三处 4.324.0；产物=exe 50734592B SHA256=5c1b3916518cd268f84ec9dfc6b61a3b3746c7d12f778151c1076736380d0abc（releases/gaea-v4.324.0.exe+SHA256SUMS-v4.324.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.324.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（三十九迁）+progress/todos。
**出口对照**=导出的 JSON 包在干净环境导入后覆盖层逐字节回魂（生成链路同效果）✅；内置升级场景三态各分支测试钉死 ✅；坏模板行不落盘且整包其余行正常导入 ✅；绑定面 703 三处同步 ✅；壳内导出落盘/导入选文件走系统对话框双门 ✅。
**观察池**=项目级覆盖（scope）；自建新模板键；系统模板版本升级提示合并交互；按书切换配方；Skill triggers；模板包签名/加密（单机明文够用）。
**未做（下刀）**=t7 前端统一接线（小说线最后一域：分析 V2/伏笔面板/标注高亮/重写建议消费面）；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## v4.323.0 · t6 提示词工坊首刀：模板可编辑覆盖层（{{name}} 渲染+三级解析+工坊面板）（2026-09-16）
> 用户指令「继续优化迭代 gaea，记得使用子代理」。三线并行子代理（A 引擎纯函数 / B handler 接线 / C 前端三件，足迹互斥）→ 主代理收口；契约先行（规格书随刀入库 进度计划/gaea-prompt-workshop-t6-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md §5/§6/§9.3/§11）。
**论点**=gaea 模板是「磁盘 JSON+embed 兜底」的只读两层，用户改不了（改了也被发版覆盖）；占位符只有 {word_count} 单点硬编码替换；模板保存零校验（MuMu §9.3 同型缺陷）——本刀补「全局覆盖→磁盘→embed」三级解析+`{{name}}` 渲染+保存校验+工坊面板。**裁决**=占位符统一 `{{name}}`（迁移 create-chapter/rewrite-chapter 两文件，substituteWordCount 双语法兼容旧盘上模板）；V1 只做全局覆盖（项目级 scope/自建模板键入观察池）；风格注入不做（oh-story T6 v4.289 已有 style.md 注入链路，互补部分观察池）；模板包导入导出留 t6-C2。**裁剪**=云端社区工坊三表/审核流/X-Instance-ID 信任模型整域不做（单机无多租户）；列表接口不回传正文（MuMu 1.74MB 全量下发教训 §9.2）。
**落地线A**=internal/prompt：Template +Version/Category/Description/Parameters 四字段（全 omitempty 零迁移）；包级 RenderPlaceholders（只认双层花括号，缺失保留原文+记名不抛错，单层 {x} 不动）；Engine.SetOverride 覆盖钩子+Get 变三级解析+Names() 字典序。新包 internal/promptstore（零 IO 纯函数，taskinbox 先例）：Issue/Override/NormalizeKey（字符集白名单）/Validate 六规则（empty-system/empty-task/brace-unbalanced/too-long=error；undeclared-var/legacy-brace=warn）/ActiveOverride（IsActive 门控）/Upsert（Version 自增+CreatedAt 保留+字典序）/Remove。prompts/*.json ×20 全量元数据补齐（9 组 category 中文映射+一句话 description+version）+2 文件 {word_count}→{{word_count}}+parameters 声明（引擎可见语义零变化，git show 逐文件比对）。
**落地线B**=gaea_prompt_store.go：状态文件 <DataRoot>/prompt_overrides.json（taskinbox 同款容错读+temp+rename 原子写）+覆盖缓存互斥（save/reset 失效惰性重读）+applyPromptOverrides 两装配点挂钩（New/SetPromptFS 重建引擎必重挂）+writingState 懒构造基线引擎（Base 对照与元数据回落）；五个 App 方法：List（列表零正文）/Get（生效模板+内置基线）/Save（NormalizeKey→未知键拒绝→Validate error 阻断·warn 落盘带回→Upsert 落盘）/Reset（幂等回落）/Preview（草稿 system 段 {{}} 渲染+未解析变量警告）；substituteWordCount 双语法（先 {{word_count}} 再旧 {word_count}，用户手改盘上旧语法模板不炸）。
**落地线C**=PromptWorkshopPanel（antd Modal 三区：九类分组列表+来源徽标（自定义 orange/内置 default/已停用灰）+v{version}，open 拉一次零轮询；详情编辑=内置基线 Collapse 对照+System/Task/输出说明/category/description/启用 Switch+Issues 逐条 error 红·warn 橙+恢复内置 Popconfirm「确认恢复」；变量预览=parameters 动态输入列+渲染+warnings Tag）；api/prompt.ts 五函数；CreatePage rail「提示词工坊」入口（既有测试零改动 13/13）；mock/novel 五档（override/builtin 双样本+Base≠Template+Save 三分支+本地 {{}} 替换）；mock-contract-prompt 6 用例；spaceBindings 五名入 play。
**绑定面**=696→701（NovelB 116→121；PromptTemplate* 挂 *App 被默认路由 CoreB，gen_bindings 显式覆盖表 +5 点名归 novel——v4.291 反推任务先例）；bridge/novel.ts +5 签名+载荷类型；bindingNames.ts 再生；play 锁 518→523。
**测试**=Go +22 测试函数（线A 15：渲染五态+边界（未闭合/去重/不二次扫描）/覆盖三态+三级解析 e2e/Names 恰 20 键/六规则矩阵 18 用例+边界 20000·20001 与 65536·65537 rune/Upsert 自增·CreatedAt 保留·乱序字典序/NormalizeKey 19 用例/JSON 形状逐字钉死；线B 7：roundtrip+坏文件容错/校验闸+未知键+isActive=false 回落/引擎覆盖 e2e/双语法矩阵/Preview+警告/缓存失效冒烟）；前端 +12（面板 6+契约 6）+CreatePage 13/13+RewriteHistory 5/5 零破坏；tsc -b 零错；eslint 足迹零告警。
**门禁**=ci.ps1 全绿 exit 0（go 130 包+vitest 367 文件 3138 例；E 系列回归+仓库卫生守卫过）；drift OK@701；版本三处 4.323.0；产物=exe 50754048B SHA256=1a712949e53414c308ae8a14f85de914d95e021b47fc9711f3ae3e880fdcf4d6（releases/gaea-v4.323.0.exe+SHA256SUMS-v4.323.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
**文档**=规格书+releases/v4.323.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（三十八迁）+progress/todos。
**出口对照**=20 模板全部可看/可改/可恢复且来源版本可见 ✅；覆盖即时生效于生成链路（eng.Get 三级解析，全部 agent 取数口共享）✅；保存真校验（error 阻断/warn 提示）✅；占位符统一 {{name}} 且旧语法兼容 ✅。
**观察池**=模板包导入导出（content_hash 三态智能导入，t6-C2）；项目级覆盖（scope 字段）；自建新模板键；Skill triggers 自动匹配；系统模板升级合并交互；风格文本注入与 novelstyle 量化指纹打通。
**未做（下刀）**=t6-C2 模板包导入导出；t7 前端统一接线（小说线最后两域）；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## v4.322.0 · t4-C3 收官：场景工程整章重写（单场景替换）（2026-09-16）
> 用户指令「继续」。小刀单线主代理直做；契约先行（规格书随刀入库 进度计划/gaea-scene-rewrite-20260916.md）。**产品裁决（单场景替换）**=整章重写产出整章新全文与原场景边界无法自动对齐——重写整章本就意味着重划结构，应用=新全文写回该章唯一场景，Restore 同语义对称；LLM 分场景回写/场景粒度重写入观察池。**论点**=NovelChapterRewrite 此前对 v4 场景章直接拒绝——场景工程项目整章重写不可达；根因=版本库 Apply/Restore 只写 blob 而 v4 真源是 scenes/（scene→blob 单向同步）。本刀补 v4 感知写回，**复用既有原语 rebuildScenesFromBlob**（CreateChapter 整章重写完成点同款：删全部场景→从新 blob 物化单场景；v3/无场景 no-op）零新机制零新绑定（696 不动）。**落地**=①Go 四处：移除 whole 入口 v4 拒绝段+original 改 ReadChapterAsStitch（v3 fallback 零变化）；Apply/Restore 各 +rebuildScenesFromBlob；partial 加 v4 守卫（「选段与场景边界无法对齐」）②前端：EditChrome +canRewrite/onRewrite/onRewriteHistory 可选 props+「整章重写」「重写历史」双按钮（仅 sceneBacked 页渲染）；ChapterPage 挂 RewriteModal+RewriteHistoryPanel（onApplied→loadChapterIntoTab 重载场景框）。**测试**=Go +2（v4 两场景夹具全链：重写→快照=stitch→Apply 收敛单场景+blob 同步→Restore 对称；partial 拒绝）既有回归绿+前端 +2（双按钮渲染/时序等 V4 探测；v3 不渲染）合计 11/11；tsc/eslint/gofmt 零错。**门禁**=ci.ps1 全绿、版本三处 4.322.0、drift 696；产物=exe 50638848B SHA256=5b1b105ad81bff7b5dd391cc445aeda8cb77aed575a516c996e10441c38ea29e（releases/gaea-v4.322.0.exe+SHA256SUMS-v4.322.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书+releases/v4.322.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（三十七迁）+progress/todos。**出口对照**=v4 场景章重写全链可达 ✅；**t4-C3 余项全清**（whole v4.304/partial v4.321/历史面板 v4.320/场景 v4.322——三通道×两存储矩阵仅剩 partial×场景一格=守卫+观察池）。**观察池**=LLM 分场景回写；场景粒度重写（对齐去味逐场景先例）；partial 选段→场景映射。**未做（下刀）**=t6 提示词工坊/t7 前端统一接线（小说线最后两域）；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## v4.321.0 · t4-C3 余项：partial 选段局部重写（版本库升级版）（2026-09-16）
> 用户指令「可继续」（沿「继续并行使用子代理，优化迭代 gaea」习惯）。三线并行子代理（A 引擎纯函数/B handler 接线/C 前端三件，足迹互斥，三线全部一次绿无卡死）→ 主代理收口。契约先行（规格书随刀入库 进度计划/gaea-partial-rewrite-20260916.md；蒸馏依据 docs/distill/04-plot-analysis.md §5.3 路径 B）。**论点**=整章重写对「改一段」太重：全章重新生成波及未选段落；partial=选区+指令+长度模式，只重写选中片段前后文原样拼接。**对 MuMu 的升级**=partial 也走 v4.304 版本库（全文快照可回滚——MuMu 局部重写无快照无 undo 系其 F5 缺陷）；全链 rune 口径。**零新绑定**：NovelChapterRewrite reqJSON 契约字段早已先行，绑定面 696 不动。**落地**=①线A internal/rewrite/partial.go（新纯函数）：ResolveSelection（rune 边界校验+±50 窗口模糊重锚，顺手修 MuMu 两处原缺陷：重锚起点负下界/start<0 纵深）+PartialLengthSpec 四档（similar 0.8~1.2×/expand 1.2~2.0×/condense 0.5~0.8×/custom±20% 钳[50,10000]，未知回 similar）+PartialMaxTokens（max(500,min(Max×3,8000))）+BuildPartialInstruction 四段固定序（上下文空占位注明仅参考→选段「选段外一字不动」→指令≤1000 rune 截断→长度提示）；NormalizeRequest partial 分支五规则（Source 恒 custom/指令必填/lengthMode 归一/custom 需字数/选区形状），whole 零字节变化②线B novel_rewrite_handler.go：partial 分叉抽独立方法 novelChapterRewritePartial——重锚→±500 rune 上下文→同一 rewrite-chapter 模板（chapter_content 只给选段、systemPrompt 代词数=区间上限）→MaxTokens=PartialMaxTokens（whole 8192 不动）→CleanRewriteOutput→前缀+新选段+后缀拼接（测试钉死前后文零变化）→版本落盘（Mode=partial+选区字段+全文快照+选段口径相似度）；types.RewriteRequest +SelectedText③线C 前端三件：EditorPanel TextArea ref 读原生 selection（code-unit→rune 换算，代理对样例钉死）+「局部重写」按钮（无选区 warning 引导）；PartialRewriteModal 三态（选段预览+指令必填≤1000+长度模式四选出字数输入→运行态禁关闭→结果态新选段+统计+应用走版本库 Apply/放弃 Discard）；CreatePage 挂载 onApplied 刷新正文。**测试**=rewrite 9 函数（whole 4 回归+partial 新 5，ResolveSelection 13 例含多字节换算证明）+app +3（往返含 partial 版本 Apply/Restore 走既有链路/不匹配零落盘/custom 无字数）+前端新增 6（Modal 4+EditorPanel 2，合并定向 34 全绿）；tsc -b 零错 eslint 零告警。**门禁**=ci.ps1 全绿、drift OK@696、版本三处 4.321.0；产物=exe 50638336B SHA256=c0f1abd770b4b4a2253c17e9c92d912f06f7992fbcbc62fe155e1e456a97fa7f（releases/gaea-v4.321.0.exe+SHA256SUMS-v4.321.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书入库+releases/v4.321.0.md（含线间对账记录）+CHANGELOG/README+.gaea AGENTS 迁 1 插 1（三十六迁）+progress/todos。**观察池**=选段高亮回写编辑器；流式输出（V1 同步等待同 whole）；deslop 模式入口（类型已留）。**未做（下刀）**=t4-C3 余项剩场景工程整章重写；t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## v4.320.0 · t4-C3 余项：重写版本历史面板（版本库前端消费）（2026-09-16）
> 用户指令「继续」（沿「继续并行使用子代理，优化迭代 gaea」习惯）。小刀单线主代理直做（纯前端后端零改动）；契约先行（规格书随刀入库 进度计划/gaea-rewrite-history-panel-20260916.md）。**论点**=v4.304 版本库（rewrites/<n>/index.json+全文快照+审计）只有「刚重写完那一刻」的 RewriteModal 能操作，NovelListRewriteVersions/NovelGetRewriteVersion 两绑定**零 UI 消费**——关弹窗/换会话后历史版本不可看不可用（v4.305「用户到不了的功能等于没有」同型）。本刀补「重写历史」面板。**落地**=①RewriteHistoryPanel（components/novel，antd Modal 硬编码中文=小说域口径）：open 拉一次零轮询；行=时间+模式 Tag（整章/局部/去味）+状态 Tag（排队中/重写中/失败/已完成/已应用/已放弃）+相似度%；**多行可同时展开**（Set 态版本对比友好）；详情懒拉 Get（缓存）=指标行（字数变化/相似度/AI 味分 before→after）+新文预览+原文快照 Collapse②动作镜像后端状态机门控（前端只禁用不裁决）：completed→应用（Popconfirm 显式 okText）+放弃；discarded→应用（后端升级语义=可再应用）；applied→恢复原文；pending/running/failed 只读；动作成功 message+重拉+onApplied 刷新正文（RewriteModal 同口）③CreatePage rail「整章重写」后加「重写历史」（无徽标打开才拉首屏零请求）④mock：List 两样本+Get 配套全文；Apply/Discard/Restore 保持抛错（诚实）。**零后端改动**：绑定面 696 不动锁数不动 gen_bindings 不跑（drift OK@696）。**测试**=RewriteHistoryPanel 5 用例（列表/空态/懒拉/门控镜像/应用闭环 Popconfirm→onApplied→刷新）+CreatePage 既有 13/13 零破坏；tsc -b 零错 eslint 零告警。**门禁**=ci.ps1 全绿、drift OK@696、版本三处 4.320.0；产物=exe 50613248B SHA256=4bc1d56d44a5f97a92fde68d1fb8b18026dec5d0c4a8c6fec19731fca8dca837（releases/gaea-v4.320.0.exe+SHA256SUMS-v4.320.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书入库+releases/v4.320.0.md+CHANGELOG/README+.gaea AGENTS 迁 1 插 1（三十五迁）+progress/todos。**出口对照**=版本库五绑定全部有 UI 消费 ✅ 跨会话找回历史版本可达 ✅。**观察池**=双栏 diff 视图（V1 单侧预览）；partial 入口随 partial 刀；按模式/状态过滤。**未做（下刀）**=t4-C3 余项剩 partial 局部重写+场景工程；t6 提示词工坊/t7 前端接线；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## v4.319.0 · 7.2-2 判据②收口：结晶技能调用计数（skill_stats）（2026-09-16）
> 用户指令「继续」（沿「继续并行使用子代理，优化迭代 gaea」习惯）。小刀单线主代理直做（体量一轮未拆子代理）；契约先行（规格书随刀入库 进度计划/gaea-skill-usage-stats-7-2-2-20260916.md）。**论点**=v4.317 出口判据②欠账「结晶技能调用 ≥5 次且成功率可见」：现状核实技能计数是前端会话级（useToolStats 只算当前会话 run_skill，不持久、无成功率、漏 read_skill 主消费通道——boot 装配的按需加载器才是结晶技能被「翻开来用」的时刻），「≥5 次」的跨会话累计口径支撑不了。本刀补**工具级调用计数**：read_skill+run_skill 双钩子 → <DataRoot>/skill_stats.json 持久化 → GaeaSkillStats 绑定 → 能力面板技能行「N 次 · 成功率 X%」。**诚实口径（V1）**：成功率=工具级（read_skill 交付正文/run_skill 管线无错=ok；读不到名也计 fail——技能名漂移如实计数不静默）；回合级成功率留观察池勿虚构；「≥5 次」是使用事实门槛不做硬闸。**落地**=①新纯包 internal/skillstats（零 IO 表驱动，先例 routesuggest/taskinbox）：Record（空名忽略/MaxSkills=500 新名拒收防漂移堆积/零值 File 自动建表）+View（calls 降序→name 升序稳定）+JSON camelCase 钉死②boot 双钩子：Options.OnSkillUse（缺省 nil 零行为变化，EmitSubagentText 先例）+sysprompt resolver 包装（read_skill 失败也计）+run_skill 装饰器 skillUseCountTool（args.name 解析失败不计宁少勿扰，CompactDescriptor 回退语义同注册表缺省）③App 层：skill_stats.json route_suggestions 同款容错读（缺失/损坏/version≠1 回空表）+原子写+skillStatsMu 串行；recordSkillUse 包级回调（引擎 goroutine 调用零 App 状态）；绑定 GaeaSkillStats（**OfficeB +1，695→696**）④前端：CapabilitiesPanel open 态拉 app.SkillStats（零轮询，reload 随引擎热加载刷新）→SkillsSection 可选 prop usage→SkillRow 徽标行灰字统计（skill-usage-stat；缺省/零调用不渲染，会话计数 chip 语义不变）+SkillStatView（types/memory）+三语 +2 键+mock +1 桩+spaceBindings work+1（锁 517→518）。**测试**=skillstats 5 函数+app 3 例+前端 2 例；tsc -b 零错、eslint 零告警。**门禁**=ci.ps1 全绿、drift OK@696、版本三处 4.319.0；产物=exe 50606592B SHA256=43739c69ad424a8356a0994525d3d4d93c4dc367e6037e00802bf249b0641de1（releases/gaea-v4.319.0.exe+SHA256SUMS-v4.319.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书入库+releases/v4.319.0.md+CHANGELOG/README+.gaea AGENTS 迁 1 插 1（三十四迁）+progress/todos。**观察池**=回合级成功率（点赞点踩/重生成信号另刀）；蒸馏建议 tab 内联已结晶技能计数；两周后真机清池核「≥5 次」。**未做（下刀）**=7.3-2 板块降视图（等 v4.318 稳定一个零功能周）；闲庭在册清欠沿既有列车。

## （非版本刀）真机验证清池首班：v4.316 归因 / v4.317 蒸馏 / v4.318 收件箱 三刀壳内走查全绿（2026-09-16）
> 用户拍板「给真机验证清池留固定车位」后的首班清池。DSH 同树流水线 v4.318 收口期间先备脚本与数据前置核对（不抢代码足迹），commit 后以 build/bin/gaea.exe（v4.318.0）+ CDP 9333 走查，**三刀全绿零产品缺陷**，全程 exceptionThrown=0、console error=0。①**任务收件箱（v4.318）**：语音路径后端闭环（GaeaRouteIntent('存个任务X', false)→handled+「已存入任务收件箱：X」回执+source=voice）✅；dry-run 预览零落盘 ✅；书斋 desk-task-inbox 节+TaskInboxPanel 手建（source=inbox）✅；状态机 doing→done、pending→abandoned、**终态非法迁移被拒**（「任务状态不能从 done 变为 doing」如实报错）✅；Ctrl+K 全链（指令预览卡→〔存为任务〕→source=ctrlk+action=save_task 落库→intentReply 内联回执）✅；空间隔离双向零泄漏（work/play 清单互不可见）✅；闲庭 garden-task-inbox 节 ✅；task_inbox.json camelCase 形状+来源可区分 ✅；清场归零、文件删除还原原始态 ✅。**palette 入口/微信通道/会话级跳转未走**（palette 组件测过、微信需 assistant 配置，入观察池）。②**成本归因（v4.316）**：模型中心 rail「成本归因」→KPI×4 真数据（总成本 ¥2617.95/总 Token 196.5M/总调用 2848/成功率 97.8%）+17 桶行+「未标记」桶如实（model_stats 实盘仍 v2——v4.316 前历史数据兼容读实证）+建议卡空态（MinSamples 20 宁少勿扰）✅。③**流程蒸馏（v4.317）**：Candidates 绑定直调返回 {candidates:[],available:false}（journal 实盘各工作区仅 1 会话 work.jsonl——诚实空态）✅；办公工作台→记忆页签→建议 tab→distill-section 渲染+候选首拉完成（非 loading 卡死）+空态文案「暂无重复流程——多完成几次同类任务后，这里会出现结晶建议」✅。**顺带验证**=意图路由导航三例真机全通（打开模型中心/记忆中枢/办公→navigate 事件→板块切换）。**走查配方沉淀（坑）**：㊀中断重跑必须步首清场（残留终态任务撞状态机非法迁移报错）；㊁Input.insertText 重跑会叠加注入——重跑前 Ctrl+A 清空输入；㊂SearchModal 是 onSearch/回车触发**非防抖**（纯 typing 不触发预览卡）；㊃ml-space 切换按钮在首屏视口外（rect.y=-15）——必须 scrollIntoView 后重取坐标再点，真实类名=.ml-space-btn.is-play（ml-space-play 只是契约注释里的概念名）；㊄ModelCenterPage 页签=自定义 .mc-rail-item 非 antd tab，mc-attribution-kpi 挂容器（4 瓦片=children 数）；㊅MemoryPanel 挂办公工作台 chatTab=memory，记忆中枢 board 是另一页。**残余欠账入池**：蒸馏一条龙（×3 同型会话播种→建议卡→结晶→新会话命中）需真模型调用与播种设计另班；palette/微信入口 UI 走查；journal 实盘厚度=每工作区 1 会话，建议真实使用积累后再清。截图证据=.tmp/walk-v4318-*.png；走查脚本=.tmp/walk-v4318-pool.mjs、patch-v4318-ctrlk.mjs、walk-v4318-attribution.mjs、walk-v4318-distill2.mjs。

## v4.318.0 · 阶段七第五刀：多入口统一任务收件箱（任务持久实体+四入口存为任务+状态机追踪，7.3-1）（2026-09-16）
> 用户指令「继续推进」（沿「继续 并行使用子代理，优化迭代 gaea」习惯）。契约先行（规格书 `进度计划/gaea-task-inbox-7-3-1-20260915.md`：A/B/C 三线足迹互斥+出口对照）→ A/B/C 三线并行子代理实现（三线全部一次绿，无卡死接管）→ 主代理收口（gen_bindings/drift/桥面三件套/发版）。**论点**=意图中枢/微信/语音/Ctrl+K 四入口自 v4.5 汇同一内核（intent.Parse→routeIntent），本刀补「任务」为**持久实体**与**统一收件箱**：指令除即时执行外可「存为任务」→ 状态机（待处理/进行中/已完成/已放弃）→ 跳回来源板块。**收件箱是清单不是调度器**——任务只在被打开/被操作时读写（零轮询零定时器零后台事件，关闭即停=昼夜运转拍板）；space 维度必带（work/play 隔离沿 boards/space.ts，跨空间仅显式，本刀不做移动/复制）；不加新板块（挂既有双空间首页）。**落地**=①新纯包 `internal/taskinbox`（零 IO 表驱动，先例 skilldistill）：CanTransition 四条合法迁移（pending→doing→done、pending|doing→abandoned；终态拒迁、同值恒 false）+NormalizeTitle（空报错哨兵 ErrEmptyTitle、120 rune 截断）+ValidSpace（仅 work|play）/ValidSource（ctrlk|palette|voice|weixin|inbox 五来源）+ParseTaskID（"ti-"+12hex 残渣防御）+FilterBySpace（''=全部，GaeaTaskList 变参先例）+Sort（状态组序→组内 UpdatedAt 降序→ID 升序稳定=行动优先）；9 测试函数约 104 断言（JSON camelCase 形状逐字钉死）②intent 新动作 **save_task**（`存为任务X/记个任务X/帮我添加一个任务X/任务：X/待办：X` 首尾锚定两式，排提醒后让位——「提醒我存个任务明天开会」归提醒；空标题不命中坠回聊天，宁漏勿误；+9 表用例）+状态文件 `<DataRoot>/task_inbox.json`（{version:1,tasks:[]} camelCase，route_suggestions 同款容错读+temp+rename 原子写）+四绑定（OfficeB +4）：List（FilterBySpace+Sort，损坏回空表）/Save（id 空=新建 pending+ti-+12hex crypto/rand+超 500 拒；id 非空=只改 title/note——**Source/Action/Target/Session/CreatedAt 不可变=来源审计链**；source 缺省兜底 inbox）/SetStatus（ParseStatus+CanTransition，同值短路零落盘）/Delete；执行层 execSaveTask：语音/微信走 routeIntentModeForAssistant→space=gaeaEffectiveSpace() 归一回退 work、assistantID 判 weixin/voice、reply「已存入任务收件箱：<title>」（TTS/回推天然可用）；**Ctrl+K/面板不走此路径**——dry-run 预览照常命中，前端直调 Save（带 shell 空间更准，两路来源可区分）；app 9 测试函数（camelCase 钉死禁 created_at 泄漏/500 上限/"WORK" 大写不泄漏/voice·weixin 双例/空间回退/dry-run 零落盘）③前端：**TaskInboxPanel**（antd Modal，v3-panel-head 对齐 TaskCenter；open 拉+变更后重拉**无轮询**；手动新建 source=inbox；四档 tab 带计数；行动作按 CanTransition 前端镜像——非法组合按钮不渲染；action=navigate&&target→〔去板块〕、session→〔回会话〕V1 板块粒度；空态 V3Empty 两分；9 用例）+**双空间首页挂点**（ModuleLauncher：书斋 w-vitals 第五节 desk-task-inbox 待处理计数；闲庭 p-foot 第五节 garden-task-inbox 内联 gridColumn '1 / -1' 跨全列——固定四列网格不动 CSS，既有四节 testid 零变化；面板单例 space 跟随 home 空间）+**四入口**（Ctrl+K=SearchModal 指令卡〔存为任务〕次按钮+save_task「执行」短路直调 Save+intentReply 回执；意图中枢=命令面板 query 非空动态项「存为任务『query』」→source=palette+当前会话路径，捕获期 input 监听镜像面板输入——组件契约不动，观察池列清退方案；语音/微信零前端改动走②）+lib/types 本地重述 TaskInboxView（herdsman 防环先例）+三语 +28 键（zh=en=zh-TW=1518 逐键相等，zh-TW 台湾惯用词收件匣/待處理/前往板塊）；ModuleLauncher「脉息四节」断言按规格更新五节（其余原样）。④收口：bridge/core +4（同名前缀无需映射，GetProgrammingWebStatus 先例）+mock/chat 四桩（两空间样本内存态）+spaceBindings 四名 shared（隔离由 space 参数承担，UnifiedSearch 口径）+bindingNames 再生；gen_bindings 附带 bindings_novel.go 多行委托规整单行（+101/−101 零语义）。**绑定面**=691→695（OfficeB +4）；锁数 513→517（shared +4）；drift OK@695。**出口对账**=①四入口来源可区分 ✅（测试钉死）③空间隔离零泄漏 ✅（space 必带+严格过滤+大写不泄漏）④零常驻 ✅（无轮询/定时器/事件订阅，读写均用户动作触发）；②跳回来源板块 ✅ 板块级、**会话级欠账**——按路径恢复 seam 存在（palette sessionItems→onResumeSession(path)）但活在 gaea App 树内首页拿不到，Session 字段已落库数据就绪后续刀接。**测试**=taskinbox 9 函数+app 9 函数+intent +9；前端 33/33（Panel 9+ModuleLauncher 12=8 既有+4 增量+SearchModal 8=5 既有+3 增量+锁数 4）+App palette/export 源级锁 6/6；tsc -b 零错、eslint 零告警。**门禁**=ci.ps1 全绿 exit 0、drift OK@695、版本四处 4.318.0；产物=releases/gaea-v4.318.0.exe+SHA256SUMS（桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书入库+releases/v4.318.0.md+CHANGELOG/README+.gaea AGENTS 迁 1 插 1（三十三迁）+progress/todos。**观察池**=命令面板 query DOM 捕获→可选 prop 清退；精确回源会话接缝；任务模板联动（GaeaTaskTemplates 从模板建任务）；收件箱首屏化=7.3-2 前置；LLM 兜底分类器对 save_task 覆盖。**未做（下刀）**=7.3-2 板块降级为任务视图（依赖本刀稳定一个零功能周）或 7.2-2 判据②调用计数小刀（statsFile 先例）；闲庭在册清欠沿既有列车。

## v4.317.0 · 阶段七第四刀：journal 历史蒸馏（重复模式挖掘+结晶审阅+审计链，7.2-2）（2026-09-15）
> 用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 `进度计划/gaea-journal-distill-7-2-2-20260915.md`：契约+足迹互斥表+出口对照）→ A/B/C 三线并行实现（**A 线子代理跑满上下文零落盘——主代理接管代写**，卡死判据先例再+1）→ 主代理收口（gen_bindings/drift/桥面三件套/App 接线）。**论点**=journal 证据链（`.gaea/work/journal/*.jsonl`）是书斋审批链既有沉淀，把它变成程序性记忆矿床：确定性挖掘跨会话重复 ≥3 次的工具应用模式 → 建议卡只提示不打扰 → 点「结晶为技能」→ LLM 从多次执行回放蒸馏草稿 → 复用 7.2-1 审阅弹窗编辑落盘 → 决定与结晶全程审计——**7.2 技能结晶双入口齐装**（演示式录制 v4.313 + 历史蒸馏本刀）。**落地**=①新纯包 `internal/skilldistill`（零 IO 表驱动，先例 routesuggest）：MineFlows（会话聚合+(At,Tool,Target) 确定性排序）+Distill（连续同 (Tool,Shape) 折叠→会话级序列**精确匹配**分组→Repeat=不同会话数≥3→步数∈[2,12]→ignored 静默→repeat 降序截 3 宁少勿扰）+BuildPatternID（"jd-"+sha256 前 8hex 确定性幂等）+ParsePatternID 残渣防御；V1 裁决=滑窗/子序列不进（误报风险，观察池）②App 三绑定（OfficeB +3）：GaeaSkillDistillCandidates（journal 现算只读零 LLM，ignored∪crystallized 双静默）/GaeaSkillDistillDraft（buildJournalReplay 纯函数：≤3 最近会话分段+AfterSummary 200 rune+整段 600 rune 防护→`prompts/skill-from-journal.json`（RTCO：多次执行**共性进步骤、差异进 cautions**、文件名通用化占位）→RetryJSON 2→parseSkillDraft→复用 SkillRecordResult 只回不落盘）/GaeaSkillDistillDecide（ignore|crystallized；crystallized 必带技能名，审计条目=sessions+skill+decidedAt「哪次历史/谁确认/哪版」；ID 防御委托 skilldistill.ParsePatternID）；状态 `<DataRoot>/skill_distill_state.json` route_suggestions 同款容错读+原子写③前端：SkillDistillSection（PatternLine 步骤序 code 工具名+「重复 N 次」徽章+最近日期+证据 Collapse+〔结晶为技能〕〔不再提示〕同卡互斥 loading+空态两分）挂 MemoryPanel 建议 tab「流程蒸馏」分区（props 全可选缺省隐藏，既有调用方零改动）；SkillRecordModal 加 preload/onSaved 可选（preload=外部已蒸馏直接回填，缺省与 7.2-1 逐字节一致；7.2-1 Composer 入口不动，MemoryPanel 本地托管第二实例组件零复制）；useSessionHandlers 四 handler（保存成功 Decide("crystallized") 落审计后刷新）；建议 tab 首开自动拉候选（收口补：只读零 LLM，fetch 期 loading 而非「不可用」误判——C 线 distillView 初值 null 使 undefined 判据失效，主代理改 ref 守卫）；三语 memory.distill.* 11 键三语键集相等④收口：bridge/core.ts +3（SkillDistillView 局部重述 types/memory.ts，herdsman 防环先例）+mappings+mock/chat 三桩（走查样本）+spaceBindings work+3+bindingNames 再生 691。**绑定面**=688→691（OfficeB +3）；work 锁 510→513；drift OK@691。**出口对账**=①只提示不打扰拒绝即静默 ✅（测试钉死）③审计链完整 ✅；②结晶技能调用 ≥5 次且成功率可见=**欠账如实入档**（需技能调用计数机制 statsFile 先例另刀）。**测试**=skilldistill 15 例+app 10 例（状态往返含审计/回放分段截断/端到端 3 会话→Repeat=3+最近优先/双静默复算/守卫/Decide 矩阵）+前端 18 例（SkillDistillSection 6+SkillRecordModal 6=4 既有零改动+收口补 preload 2 例+MemoryPanel 2 既有+锁数）；tsc -b 零错（双向漂移防线过）、eslint 零告警。**门禁**=ci.ps1 全绿 exit 0（复跑：首跑 exit 1 尾部无 FAIL 输出——flaky 先例再+1，复跑即绿）；产物=exe 50576384B SHA256=e5577402da81afbf0b0135631db066494aa7e415455da67eee2dbe9b0d5e3d2d（releases/gaea-v4.317.0.exe+SHA256SUMS-v4.317.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书随刀入库+releases/v4.317.0.md+CHANGELOG/README+.gaea AGENTS 速览（迁 1 插 1）+progress+todos。**观察池**=滑窗/子序列挖掘 V2 等真实数据；journal 回放纳入 verdicts 过滤；真机走查（多会话同型任务→建议卡→结晶→新会话命中一条龙）挂闲置窗口；releases/README V4_RECENT 实存条数与「最近 34 版」头注不符（v4.315 前已存在）。**未做（下刀）**=7.3-1 任务收件箱或 7.2-2 判据②调用计数小刀；闲庭在册清欠沿既有列车。

## v4.316.0 · 阶段七第三刀：路由学习（任务级账目+成本归因+建议制改绑，7.1-2）（2026-09-15）
> 用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 `进度计划/gaea-route-learning-7-1-2-20260915.md`：三线调研结论+契约+足迹互斥表）→ A/B/C 三线并行实现（B 线原实现子代理 40 分钟零落盘两次检查点未应——中断接管由主代理代写，先例再+1：子代理卡死判据=长时间零落盘+检查点静默）→ 主代理收口（gen_bindings/drift/前端三件套）。**判据对账**=①任务级账目覆盖全部功能绑定键：`ChatRequest/ChatSimpleOptions` 加 `Feature` 带外字段（`json:"-"` 贴 EngineID 先例，请求体形状零变化）→ recordUsage/parseStreamEvents/prepareStreamRequest 透传 → `statsKey` 扩三维 `feature|engine|model`（whisper→chat 归一防脏桶）→ statsFile v2→v3（旧文件全量按 feature="" 兼容读、首次落盘升 v3）→ 24 个 handler 文件约 40 调用点一行式透传（chat/novel/office/gaea/characterlib/routine/sin 七键+whisper 别名全覆盖，grep 复核零遗漏；copilot 三入口获批扩迹带 feature 参数；bridge 工厂 Provider 打标签：boot 主 agent→gaea、显式引擎→office）；残留如实报：platform_handler 风格分析经 internal/style 构造（足迹外落未标记桶，novel 键另有 9 处覆盖）、gaea_diagram 两处不走路由属如实呈现 ②成本归因视图：模型中心新「成本归因」tab（AttributionSection：KPI 行×4+功能分组表（feature×engine×model 三维桶行，未标记桶最后）+EmptyState+错误条；useAttributionState hook；api/engines.ts +4 函数走 App() 兼容通道）③建议制改绑：新纯包 `internal/routesuggest`（零 IO 表驱动：score=成功率×成本因子（域内归一，本地 0 成本=1）+ScoreGapThreshold 0.25 宁少勿扰+MinSamples 20 证据门槛+单域至多 1 条+ID 确定性重算幂等）；App 三绑定 `GaeaRouteSuggestions/GaeaRouteSuggestionApply/GaeaRouteSuggestionIgnore`（Apply 内部走 `SetFeatureModel` 既有链路保 gaea/office 域即时重建，绝不自动改绑；忽略/采纳状态原子写 `<DataRoot>/route_suggestions.json` temp+rename）④本地引擎同池：候选=全部启用引擎 llm 模型（Kind 缺省回退 ClassifyModelKind），本地云端同池天然满足；mock 契约样本含本地引擎建议卡（走查锚点）。**绑定面**=684→688（ModelB +4：GaeaRouteLedger/GaeaRouteSuggestions/GaeaRouteSuggestionApply/GaeaRouteSuggestionIgnore，gen_bindings 显式覆盖表→model 门面 GaeaUsageOverview 同族先例）；work 锁 506→510（shared，两空间模型中心共用）；bridge/model.ts 签名+局部重述类型（RouteLedgerViewB 族，herdsman 局部重述先例防 api↔bridge 环）；mock/model.ts 四桩（含走查样本数据）+新契约测试 mock-contract-route.test.ts（4 名存在性+形状关键键三方对齐）。**验收**=routesuggest 11 例（阈值边界/无样本静默/样本门槛/单域取最大分差/ignored 静默/本地引擎胜出/全本地成本维度失效/多域独立/重算幂等/ID 往返+残渣防御——第二个 | 藏进字段段的解析 bug 测试抓出后修）/stats 3 例（三维桶/归一/v2 兼容升 v3）/ai 3 例（非流式/流式+归一/纯函数）/app ledger 2 例+suggestions 3 例/前端 AttributionSection 5 例+契约 4 例+锁数；tsc -b 零错（双向漂移防线过）、eslint 零告警。**门禁**=ci.ps1 全绿 exit 0（一次过）、drift OK@688、版本四处 4.316.0；产物=exe 50515456B SHA256=82c0e4537d28dfb53314770054672ea094e636aac2645509227d76de88113213（releases/gaea-v4.316.0.exe+SHA256SUMS-v4.316.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=规格书随刀入库+releases/CHANGELOG/README+AGENTS 速览（迁 1 插 1）。**观察池**=账本三维桶为窗口聚合（与统计同窗口），「按任务实例」切片留待任务收件箱（7.3）联动；bridge 启发式依赖「app 侧显式引擎 provider=office 域」现状（6 处），后续新增显式引擎 NewLLM 需重新审视；route_suggestions.json 无 UI 清单（忽略后悔=改绑回原模型即可，ID 随绑定漂移）。**未做（下刀）**=7.2-2 journal 历史蒸馏或 7.3-1 任务收件箱；真机走查（发一条聊天→归因 tab 见 chat 桶行+建议卡）挂闲置窗口。

## v4.315.0 · 书斋首页 v9「案头」重设计 + 外观模块下拉化（零功能删除）（2026-09-15）
> 用户指令「继续并行使用子代理，优化迭代 gaea」。本轮=接管上会话中断的在飞刀（两份规格书随刀入库：`进度计划/gaea-desk-home-redesign-v9-202609.md` + `gaea-appearance-module-redesign-202609.md`，实现与工具链验证已由上会话完成）+独立审查子代理七点对账+CDP 走查取证+全量门禁+发版收口。**v9 诊断**（v8 遗留四条）：①砍刊头后书斋无排印锚点（首屏最大字号 19px，与闲庭月洞门+34-54px 大字并置成「无签名版」）②清单海（~80% 界面 hairline 文字行+等宽字消费点 10 处）③命令条仅 54px 主角不立④右舷五件纵排无图形锚点。**落地**=①masthead 身份区（w-seal 印章 44px 印泥朱 `#d06055` 全文件唯一 hex 豁免、取空间名首字 aria-hidden -3° 定格+台名 clamp(30,3vw,42px)/650+lede+chip/pill 迁入；与闲庭月洞门成「圆/居中/token 派生色 ↔ 方/左置/实色不随主题」身份对仗）②kicker 行整体退役（mark/rule 退役、chip/pill 迁 masthead、deck-sub 升 w-mast-lede）+命令条 54→60px、输入 14→15px③能力目录 w-index 单列行→w-plaques 双列案牌（图标章 32px+单行描述+hover 抬升；删 w-index-num 等宽序号降 mono 密度）④右舷 w-rail-panel→w-vitals（写作进度升首节、ml-ring 72→80px 用 .w-vitals 作用域选择器隔离不波及闲庭共用件）⑤外观面板重构：ChoiceCards/ThemeCard 全仓退役→`Select<ThemePreset, ThemeSelectOption>` 泛型下拉（ThemeOrb 色点+optionRender hover 预览「预览中」徽章+onOpenChange 兜底）+SegmentedRow 行式三面板（模式/密度/动效），447→421 行⑥WhisperTracePanel 暗色硬编码 hex→`--md-sys-*` 语义令牌（亮色模式可读性，角色库记忆弹窗消费面）。**契约不变**=ml-space-switch/work/play、ml-space-chip（迁位不改名）、desk-recent-docs、.garden-banner 五钩子全保；i18n 键零增删（三语键集 1479=1479=1479，livePreviewDesc 1 键文案随交互三语更新）；闲庭 GardenHome 与 .p-*/.garden-* 逐行不动。**验收**=CDP 9333 DOM 断言书斋 17 项+闲庭 5 项全绿+亮色档令牌实测（--md-sys-color-surface=#f0fdf9 生效）+三截图留档（.tmp/v9-desk-1440.png / v9-garden-1440.png / v9-desk-work-light.png）；ModuleLauncher 8/8（4 既有+4 新增）+AppearancePanel 6/6+独立审查全绿（className 双向对账孤儿=0、三语交叉差集空）。**门禁**=ci.ps1 全绿 exit 0（第三跑：首跑 netclient readLoopPeekFailLocked 定向复跑绿、二跑 internal/app TempDir 清理竞争——均环境 flaky 非断言失败，先例再+2；三跑清走 Vite/无头 Edge 负载后全绿）、零绑定面 drift OK@684、版本四处 4.315.0；产物=exe 50464768B SHA256=356b7c6e8ac9ed4ecfa4f0fb5b1f9fa6c52f61b1f1dcdcc1bd13b620e3bc2a1d（releases/gaea-v4.315.0.exe+SHA256SUMS-v4.315.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=design-system/gaea/pages/home.md 补 v9 节+releases/CHANGELOG/README（releases README V4_RECENT 区块补齐 v4.308~v4.315 共 9 版滞后并裁旧 8 条回 34 席）+AGENTS 速览（迁 1 插 1）。**观察池**=releases/ 实存 33 版 exe 与「留 5 版」拍板不符（不动文件待用户定夺）；v3-rise 入场序号两对同拍（deck/ledger 同 2、cap/rail 同 3，无功能影响）；走查遗留 pnpm 污染两枚已清+frontend/NUL 坏文件已删（rg 全仓报错元凶）。**未做（下刀）**=7.1-2 路由学习（三线调研已完成：RecordCall 全量收口+ChatRequest 带外字段先例+routesuggest 纯包+模型中心归因 tab 落点，契约待主代理定稿）。

## v4.314.0 · 书斋首页 v8 重设计「驾驶舱台面」（技能库 ui-ux-pro-max 驱动，零功能删除）（2026-09-15）
> 用户指令「使用技能重新设计书斋首页」。流程=ui-ux-pro-max 技能库（design-system/product/ux 三路检索定方向：Swiss 数据密集 + 既有星枢令牌 MASTER 为视觉权威，技能输出按契约不覆写项目体系）→ Edge 无头 CDP 截图现状（node 原生 WebSocket 版 shot 脚本，BootSplash 虚拟时间预算不奏效改真实等待）→ analyze_image 四问题诊断 → 设计 → 实现 → 双宽度目检。**诊断**（v7「文书台」）：①杂志刊头大标题占 ~10% 视高信息价值极低（工作台首页第一任务是发起工作不是读刊头）②左栏长卷（刊头→账页→目录）目录大部分挤出首屏③账页与目录两段同构行式列表纵向堆叠扫视疲劳④右栏五节等权（遥测/写作/会话/记忆/晨报）无主次。**落地**=①刊头压缩：w-mast（大标题 clamp(32,3.3vw,50px)+三行 lede）→ w-deck 一条 kicker 行（空间 chip+单行 lede 截断 title 提示+rule+pill）；②**命令台升格第一主角件**：命令条/气泡流/语音状态收进一张玻璃抬升面（accent 渐变+顶部 1px 光线，radius-xl）——视觉锚点从「书斋」二字转移到命令台；③**账页×目录双列**（w-columns 7fr/5fr，≤1180 单列塌缩账页在上）：同构列表并排；旗舰横带压缩适配窄列（padding 20→11/水印 96→68px/grid 46→40）+ w-index 双列改单列；④**右栏主从**：晨报卡（主动作卡）置顶，遥测/写作/会话/记忆四节退后；⑤间距节奏校准（w-cap gap 12→9）。**契约不变**=ml-space-chip/desk-recent-docs/ml-space-* testid 全保留；i18n 零键增删（home.sub 收进 deck 头单行截断不删内容——UI 简化红线）；语音/命令条/账页/目录/右栏五节功能全保留。**验收**=1440×900 左栏一屏收齐（余 ~5-8% 底部空间，目录 01-09 全可见）+1100×800 双列塌缩正常+间距均匀无重叠溢出（analyze_image 复核）；既有测试 9/9 零改动（ModuleLauncher 4+HomePage 5）。**门禁**=ci.ps1 全绿 exit 0、零绑定面 drift OK@684、版本四处 4.314.0；产物=exe 50439168B SHA256=beccb7caf4611b0b21b8b74cf298a9e102de65d45449afc4fdfcc15f6fb58928（releases/gaea-v4.314.0.exe+SHA256SUMS-v4.314.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=design-system/gaea/pages/home.md 补 v8 节（诊断/方案/契约/验收）+releases/CHANGELOG/README+AGENTS 速览（迁 1 插 1）。**观察池**=晨报卡有数据时的右栏高度回看；写作进度环空态（「未打开项目」占 ~90px）密度优化候选（另行拍板）。

## v4.313.1 · 办公板块走查补刀：右栏折叠后对话列不居中（CSS 覆写作用域修复）（2026-09-15）
> 用户报障：办公板块右侧面板折叠后对话不居中。**根因**=styles.css 办公覆写把 `.v3-reading`（对话阅读宽度列）写死 `margin-left: 0`（设计意图=右栏开启时与工作台/产物面板满宽区域共用同一左缘）——右栏折叠后没有可对齐物，正文列仍靠左，而输入框 Composer 根节点本是 `mx-auto` 居中，于是折叠态「输入框居中、正文靠左」错位。**修复**=①默认态改回 `margin: auto` 居中（宽度仍 --maxw=min(1000px,100%)，与输入框同宽同缘）；②左对齐降级为右栏开启态作用域 `.layout--workspace-open`/`.layout--preview-open`（保留原设计意图）；③左对齐包进 `@media (min-width: 1240px)`——与既有 ≤1239px「workspace-pane display:none 但类仍在」网格覆写同断点，窄窗下面板不可见时不左对齐（堵住同款错位的窄窗变体）。**范围**=纯 CSS 三条规则，零 Go/TS 改动、零绑定面；`.phase` 阶段小标在阅读列内随列走，无需另改。**门禁**=ci.ps1 全绿 exit 0（lint 6 warn 0 err 为既有基线）、drift OK@684、版本四处 4.313.1；产物=exe 50439680B SHA256=c37c05a977751a163c9caca922bdac8533e73439315b17d9d667d12927b11230（releases/gaea-v4.313.1.exe+SHA256SUMS-v4.313.1.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=releases/v4.313.1.md+CHANGELOG/README（patch 版按 v4.283.1 先例不动 AGENTS 速览）；真机目检挂观察池（宽窗折叠/展开两态+窄窗 <1240px 一眼）。

## v4.313.0 · 阶段七第二刀：会话录制技能（演示式录制，7.2-1）（2026-09-15）
> 阶段七规划（docs/gaea-stage7-plan-2026-09.md §2）7.2-1 演示式录制——平台调研（docs/gaea-platform-market-survey-2026-09-15.md §1.2）验证的形态：Anthropic Cowork「Record a Skill」（2026-07-21）/OpenAI Codex「Record & Replay」（2026-06-18）均已发布，「演示一遍→沉淀为可复用技能」是大厂兑现位；SKILL.md 成事实标准（40+ 兼容产品）而 gaea skills/ 加载器（frontmatter+applies_to）天然对齐。改造前 gaea 已有两条沉淀通道：单轮 task/solution 手动沉淀（GaeaCaptureSkill+SkillCaptureModal）与 procedural 记忆聚类候选（gaea_memory_suggestions）——缺「会话级多轮回放→结构化草稿→审阅入库」。**落地**=①提示词模板 prompts/skill-from-session.json（RTCO：蒸馏纪律=只依据回放不虚构/步骤祈使句可照做/通用化措辞不写死项目路径；output=JSON {name,description,scenario,steps[],cautions[]}，走既有 prompt 管道）；②internal/app/office_skill_record.go：buildSkillReplay 纯函数（GaeaHistory 多轮含工具事件→末段 60 条回放文本，用户/助手 600 rune 截断、工具条目 200 rune 摘要，system 跳过）+parseSkillDraft（名字清洗归一 kebab-case/条目裁剪上限 15 步/空步骤断然拒绝）+renderSkillDraftBody（适用场景/操作步骤编号/注意事项/调用方式，与单轮模板同构）；③绑定 GaeaSkillDraftFromSession（办公引擎守卫→回放→routeOfficeLocal("office") 本地优先路由→RetryJSON 格式重试→草稿+回放+SKILL.md 预览，**只回不落盘**）+GaeaSkillDraftSave（审阅修订后落盘）；④GaeaCaptureSkill 写盘段抽出共享通道 saveSkillFile/saveSkillFileContent（工作区 .gaea/skills+全局镜像+热加载，单轮与会话录制同一落点，同名覆盖）；⑤前端：SkillRecordModal（打开即蒸馏→可编辑草稿表单〔标识/用途/场景/步骤逐行/注意事项逐行〕+SKILL.md 预览+回放依据折叠→保存 toast 热加载提示）+Composer 工具栏入口（截图按钮旁 Wand2 图标，对标 Cowork「+菜单→Record a skill」位置，三语言键 composer.recordSkill）+桥面 +2（core/mappings/mock union/spaceBindings work）+wails models 再生；绑定面 682→684（OfficeB 197→199）。**纪律**：LLM 只产草稿，落盘必经用户审阅（GaeaSkillDraftSave 是显式动作）——「作者是上帝」同款；入口禁用条件=running。**测试**=Go 6 例（回放构建含 system 跳过/空工具结果剔除/末段窗口与截断、草稿解析含名字归一/上限/空步骤断拒、渲染含条件段、保存落盘含步骤编号与注意事项/缺场景拒绝、引擎未初始化守卫）+vitest 4 例（打开即蒸馏回填/逐行编辑保存逐参传递/空步骤拒绝不落盘/蒸馏失败 toast 可重试）；既有测试零改动（GaeaCaptureSkill 两例经重构后原样通过）。**坑沉淀**=①bash heredoc 内 Python 打补丁的 \\n 转义经两层解释后变成真换行破坏 TS 字符串（mock/chat.ts）——含转义的补丁用 Edit 工具不用 heredoc；②spaceBindings.test 锁总数 504→506 忘同步是 CI 唯一红点（新绑定三件套：bindingNames 再生+spaceBindings 归类+**测试锁数量**）；③go test 首跑 flaky exit 1 复跑全绿先例再+1（本次 go 侧）；④gen_bindings 再生重排 novel 门面 368 行噪音——git checkout 还原（既有坑再证）。**门禁**=ci.ps1 全绿 exit 0（第二次复跑）；绑定面 +2 drift OK@684（work 锁 506）；版本四处 4.313.0；产物=exe 50439168B SHA256=1e1b86665434241e9ced0801fb6d977e7aeb27a0883a3da48ee9d07698f90c6d（releases/gaea-v4.313.0.exe+SHA256SUMS-v4.313.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=releases/v4.313.0.md+CHANGELOG/README+.gaea AGENTS 速览（迁 1 插 1）+progress+todos。**7.2 技能结晶双入口已落其一**；**未做（下刀）**=7.1-2 路由学习（任务级成本归因+建议制改绑）或 7.2-2 journal 历史蒸馏（评测依赖已解锁，观察 Letta/Bionic 逼近速度）；闲庭在册清欠沿既有列车。

## v4.312.0 · 阶段七首刀：记忆质量受控评测（三脑四域零依赖基线 + 经验习得题集，7.1-1）（2026-09-15）
> 阶段七规划（docs/gaea-stage7-plan-2026-09.md，2026-09-15 拍板启动）首刀，闭合在册开放增量「记忆质量可评测」（market-survey 候选2，对齐物升级为 BEAM/LongMemEval-V2）。改造前：书斋四库有运行态检索评测（GaeaRetrievalEvalRun，需 Herdsman embedding 引擎进不了 CI），三脑记忆域（项目 StoryMemory/办公事实知识/轻语人格记忆）的检索质量**零量化**——记忆不可信则阶段七技能结晶（7.2）的复用成功率不可测。**落地**=①新包 internal/memoryeval（纯函数+种子语料，零外部依赖）：通用评测核心 Evaluate（recall@10 门槛 0.8 沿 T5-6 口径 + precision 均值；预期空不计召回分母、无命中不计精度分母、TopK 截断）+ ParseSet/ResolveSetPath（```json 代码块解析+向上 6 级查找，与 retrieval-eval-set 同款约定）；②三域 runner 全走生产检索路径：story=internal/memory 纯 Go BM25、work=internal/gaea/bm25 Ranker、persona=whisper SQLite FTS（ReplaceFactsInDB+RebuildFactsFTS 落临时库后走 SearchFactIDsFTS，含中文 2-gram LIKE 降级——评测即生产真实能力）；③**经验习得题集 22 题**（≥20 达标，对齐 LongMemEval-V2「习得未来任务所需经验」方向）：查询为未来任务口吻（「给领导发周报前注意什么」），期望命中=承载该经验的记忆条目——为 7.2 技能结晶预置评测基建（whisper 侧 procedural_habits 表是远期接口）；④种子语料 needle-in-haystack：story=修真长篇 12 目标+40 市井支线陪衬（人物/地名一致）/work=12 目标+28 行政事务陪衬/persona=10 目标+20 无关事实陪衬/experience=22 条（经验+教训成对）；题集与种子 ID 漂移由测试硬断言兜住。**测试**=memoryeval 4 例：ParseSet 格式与错误/Evaluate 数学（recall 0.5·precision 0.25·空分母·TopK 截断）/结构硬断言（经验≥20·每条有预期·expected ID 与种子零漂移·语料规模下限）/基线软告警（四域数字进测试日志，低于 0.8 只 [WARN] 不阻断——结构性错误才硬失败）；既有测试零改动。**基线**=四域 recall@10 全 1.000（story 0.433/work 0.478/experience 0.661/persona 0.727 precision 均值），落档 docs/memory-eval-set.md；**评测产出真实发现**：persona 泛主语查询淹没（「用户」二字组 LIKE 命中全部事实、top-10 灌满、单条 precision 0.1）——生产路径真实弱点记录在档不动引擎（候选改进：2-gram IDF 加权或跳过通用主语词）。**门禁**=ci.ps1 全绿（vitest 首跑 ContextView 类 flaky 28 例、全量复跑 356 文件/3064 测试全过——并发 flaky 复跑先例再+1）、零绑定面 drift OK@682、版本四处 4.312.0；产物=exe 50414080B SHA256=c41c22f447b1a86d3f4db6464a1c9202b016de1a4a66250694a9f2f1ebbbdb8e（releases/gaea-v4.312.0.exe + SHA256SUMS-v4.312.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**文档**=docs/memory-eval-set.md（题集+匹配规则+基线数字）+docs/README.md 登记（另补 stage7 规划与平台调研两档）+releases/CHANGELOG/README+.gaea AGENTS 速览（迁 1 插 1）。**未做（下刀）**=7.1-2 路由学习（任务级成本归因+建议制改绑）或 7.2-1 演示式录制（无依赖可先行）；闲庭在册清欠（t4-C3 余项/t6 提示词工坊/t7 前端接线/GenerationGate 收口）沿既有列车。

## v4.311.0 · t5 收官刀：关系图谱升级（三类节点·四层边·分类筛选·详情浮层，§7.5）（2026-09-15）
> MuMu 蒸馏线 t5 角色状态机域收官刀（规格 docs/distill/05-character-career.md §6.1/§6.2/§7.5）。改造前图谱组织节点是「孤岛」（组织成员关系没画出来）、职业完全不可见——四刀差分器+回灌+职业 UI 攒下的状态机数据在图谱上一览无余是收官题眼。**落地**=①数据层抽出纯函数模块 RelationGraph/graphData.ts（buildGraphData，单测钉死）：节点三类=角色/组织/职业（方形节点，career-main-{id}/career-sub-{id}）+两个虚拟分组节点（__career_group_main__/__career_group_sub__，MuMu :100-101 同名常量）；边四层=组织成员（org→成员，职位标签，dashed，strength 0.5↔MuMu layoutWeight 8）/职业分组（group→职业，dashed，0.35↔4）/主职业（0.3↔3）与副职业（dashed，0.2↔2）/人际（0.02↔1）——布局权重映射 d3-force 每边 strength（MuMu 用 dagre 分层，gaea 保留 d3-force 不引新依赖，权重语义等价）；人际边 pair 去重（MuMu 无向语义）+「仅前结构边参与布局」：人际 strength 0.02 基本不布局（MuMu :971-974），**无组织/职业边时人际回 0.3 兜底**（MuMu :972）②组织成员过滤口径与 v4.308 回灌区段一致（仅 active/缺省语义）+悬空引用防御（成员指向不存在角色跳过）③组件改造：分类筛选（五类 checkbox，隐藏类不渲染且不参与布局重仿真）/点击节点详情浮层（角色=定位+主职业阶+副职列表；职业=持有人+最高阶；组织=成员；再点空白或同节点关闭）/职业节点方形+分组小圆虚描边/虚线边 setLineDash/选中节点描边强调/悬停与选中高亮并行④零新依赖（d3-force 既有），CharacterPage 调用点零改动（props 向后兼容，organizations 传 member_list 即得组织成员边）。**测试**=vitest graphData +4（节点三类计数与职业汇总含最高阶/边四层标签强度虚线逐条含 retired+ghost 过滤与人际去重/兜底强度回 0.3/副职无名快照退 career_id+标签表五类）；tsc 全绿；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@682、版本四处 4.311.0；产物=exe SHA256=c494d53bc77f91692926f6deba4e0606de180a2eb7f382159710887178130592（releases/gaea-v4.311.0.exe + SHA256SUMS-v4.311.0.txt；桌面副本同哈希；冒烟 200 过）。**t5 角色状态机域四刀+收官全清**：差分器（v4.307）→回灌（v4.308）→组织差分（v4.309）→职业 UI（v4.310）→图谱（v4.311）。**未做（下刀）**=t4-C3 余项（partial 局部重写/场景级工程整章重写/版本历史面板）；t6 提示词工坊/t7 前端统一接线未开工；真机走查（职业设定→生成携带→图谱可视一条龙）等闲置窗口。

## v4.310.0 · t5 第四刀：职业管理页（角色-职业绑定 UI，§7.6 断线补齐）（2026-09-15）
> MuMu 蒸馏线 t5 角色状态机域第四刀（规格 docs/distill/05-character-career.md §6.4/§7.6/§7.7-#9）。§6.4 关键空档：MuMu 角色职业接口后端完备、前端断线（CharacterCareerCard 无人 import）；gaea 差分器四刀全通后同样「用户到不了」——MainCareerID 只能靠分析差分或手改 JSON。本刀补 UI 通路（v4.305「用户到不了的功能等于没有」同款理由）。**落地**=①绑定对（NovelB 门面 680→682）：SetCharacterCareer(charID, reqJSON)（is_main/career_name/stage；阶段 <1 钳 1；**主职业允许替换**——手工意图明确，与差分器拒绝替换防 LLM 误改两处语义有意不同，码内注明；副职业同名引用幂等更新+MaxSubCareers=2 上限沿用；手工设置不写水位=UpdatedChapter 保持 0 最低水位，后续差分正常推进）+ RemoveCharacterCareer（主职业清引用对/副职业按名删行，皆显式报错不静默）②存储落点=项目 characters.json（与差分器/生成注入同库——UI 设置即生成可见，方案 C 名称即 ID 口径）③前端：CharacterPage 角色 Drawer 增「职业体系」区块（主职业 Input+阶段 InputNumber+设定/移除；副职业 Tag 列表+删除 Popconfirm+添加行**达上限 2 禁用**；即时保存不随「保存本书状态」，操作后刷新全量并同步 Drawer 快照）+types CharacterData 扩 v2 字段（current_state/main_career_id/main_career_stage/sub_careers+CareerRefData——GetCharacters 早已返回只是前端未消费）+api 两函数④契约面：bridge/novel.ts +2 签名、mock +2 no-op（名单 union 同步）、bindingNames 再生 680→682、spaceBindings +2 归 play 锁 502→504。**测试**=character +1（矩阵：设主/替换/副职两枚+同名幂等/上限拒绝/空名拒绝/移除主副/不存在报错）+vitest CharacterPage 3 例（当前值与副职 Tag 渲染含无名快照退 ID/「设定」逐参断言+刷新/上限禁用+Popconfirm 二次确认逐参）；既有零改动。**坑沉淀**=antd 两字按钮渲染插空格（「设 定」「添 加」）——断言须 /^设\s*定$/（记忆已有此坑，本次再踩实证）。**门禁**=ci.ps1 全绿、绑定面 +2 drift OK@682（play 锁 504）、版本四处 4.310.0；产物=exe 50407424B SHA256=7bdfbe8a75ab07b2a1c94b260458f712424f8e38466a3bfa809da0e110d21a6a（releases/gaea-v4.310.0.exe + SHA256SUMS-v4.310.0.txt；桌面副本同哈希〔Git Bash sha256sum 输出路径带反斜杠前缀致字符串比对假性不等，哈希本体一致〕；冒烟 200 过）。**维护**=.gaea/AGENTS.md 速览整段分流：留最近 3 版（v4.309~v4.307）、10 条（v4.306~v4.297）迁 archive 段首，水位 61671→38151B 根治（14 版口径废止改 3 版口径）。**未做（下刀）**=t5 余项：关系图谱升级（§7.5 节点三类边四层，唯一余项）；t4-C3 余项（partial 局部重写/场景级工程/版本历史面板）；t6 提示词工坊/t7 前端统一接线未开工；真机走查（职业设定→生成携带一条龙）等闲置窗口。

## v4.309.0 · t5 第三刀：组织状态顶层差分接线（分析链三路输入全通）（2026-09-15）
> MuMu 蒸馏线 t5 角色状态机域第三刀（规格 docs/distill/05-character-career.md §3.7/§3.8/§7.7-#5）。v4.307 差分器已实现组织成员六变更+组织自身更新，但调用侧 analysis_v2.go 第 4 参传 nil、分析模板无 organization_states 输出契约、V2 载荷无字段——组织差分是有管道无水源。本刀接通：分析一章 → 组织成员/势力值/覆灭自动落库。**落地**=①V2 wire 契约（types/analysis_v2.go）：OrganizationStateChangeV2/OrgMemberChangeV2 名称式引用（LLM 只输出名称，MuMu 同款约束；与差分器输入 OrgStateDiff/OrgMemberChange 的 ID 式分层——对齐 character_states 先例），AnalysisResultV2 +organization_states 字段（omitempty，稀疏差分：未提及的组织不输出）；②模板契约：analysis-chapter.json output.description 加 organization_states 完整 JSON 形状（org_name/power_value 0-100 绝对值/destroyed/member_changes 六值 change_type）+must 约束追加稀疏差分纪律（名称必须与角色库精确一致，无法确定就省略该条，不得猜测或模糊匹配；destroyed=true 时其余字段忽略）；③解析链：syncCharacterStatesV2 组织名/角色名→ID 精确匹配（失败 WARN 跳过），传入 ApplyChapterDiff 第 4 参；日志扩 orgStateUpdated/orgMemberUpdated；早退条件扩组织维度；④**修首刀差分器缺陷——覆灭短路**：destroyed=true 时原实现照写 PowerValue 且继续处理成员变更；对齐 MuMu :718-750 改为清零势力值（:725）+continue 跳过同条 power 与成员变更（组织已灭加人晋升无意义），destroyed=false/null 不走短路分支不得复活已覆灭组织。**范围裁决**：PowerValue 用 0-100 绝对值而非 MuMu power_change 相对增量——绝对值幂等可重放，天然免疫 MuMu §8 坑 2（update_organization_states 无章节号单调守卫，重跑低章节覆盖 current_state/势力值）。**测试**=analysis +2（组织差分端到端：晋升带忠诚度+新成员 ID 匹配+缺省忠诚 50+JoinedAt 章号/不存在组织与角色 WARN 跳过/覆灭短路 power 忽略/空载荷零写盘）+characterstate +1（覆灭短路纯函数钉死：清零+成员跳过+净产出计数+destroyed=false 不复活）；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.309.0；产物=exe 50394624B SHA256=40e995648dc78f4bbed5d8ec3c726fd9c1afe2ad5eb9863397e59fbde1502fbf（releases/gaea-v4.309.0.exe + SHA256SUMS-v4.309.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t5 余项：关系图谱升级（§7.5 节点三类边四层）/职业管理页（§7.6）；t4-C3 余项；t6 提示词工坊/t7 前端统一接线未开工；真机走查等闲置窗口。

## v4.308.0 · t5 第二刀：状态回灌（SceneBible 实体属性 + 章节生成 character_states 槽位，降噪）（2026-09-15）
> MuMu 蒸馏线 t5 角色状态机域第二刀（规格 docs/distill/05-character-career.md §3.9/§7.4）。v4.307 差分器把状态写进 characters.json 后，生成侧看不见——状态机的一半价值在写回生成侧（§3.9）。MuMu 有两条回灌通道且格式不一致（分析一行式 vs 生成多行式，历史债），gaea 统一多行式（chapter_context_service.py:31-45 口径）。**落地**=①SceneBible 实体属性回灌（§7.4-1 零新增管道）：entityFromCharacter 追加 current_state/career_main/career_sub 三属性（与既有 status 同一管道），sceneCharFromEntity 读回+角色文件兜底（实体库未回灌时以 characters.json 为准），formatSceneChar 渲染「当前状态(截50)/主职业/副职业」行——存量老角色无状态字段零噪声不渲染空行；②章节生成 character_states P1 槽位（§7.4-2）：create-chapter 模板新增 input_sections.character_states（P1），注入器 buildCharacterStatesSection 按 MuMu 多行式生成（【名】(💀已死亡/❓已失踪/📤已退场 emoji 标记)+当前状态(截150，值内已含章号)+主职业「剑修·3阶」+副职业+所属组织(职位+忠诚度)+关系网络(描述[亲密度])），接线 CreateChapter 主链 BuildUserPrompt——P0 characters 是基础人设、P1 是状态机增量，语义分工不重复；③降噪规则（§7.4-3 照搬 MuMu chapters.py:794/821）：intimacy==50 与 loyalty==50（0 视为未记录）不注入；past 关系（存活级联已结束）整条跳过；非 active 组织成员不显示；关系取前 5/组织取前 3（MuMu :807/:791）；区段整体预算 1500 rune；存量项目角色零 v2 字段整角色跳过、全员无则返回空串（BuildUserPrompt 不渲染空 context，零噪声）；④types.CareerMainLabel/CareerSubLabels 职业标签纯函数：MuMu 渲染「剑修(3/10阶)」带最大阶，gaea 方案 C 不建 Career 主表、worldview careers section 是 markdown 文本——死解析不如不做，渲染「剑修·3阶」，CareerName 冗余快照优先缺失退 career_id。**范围裁决**=chapter-generate.json 是 embedded 死资产（无 Go/TS 消费，仅 prompt_test 断言存在），真实生成链路是 create-chapter——槽位打 create-chapter，死资产不动不养死槽位。**测试**=types +1（CareerLabels 主/副带阶·无阶·快照优先·ID 兜底·全空跳过）+novelcontext +3（实体属性写入读回渲染 roundtrip/角色文件兜底/无状态老角色零噪声）+app +4（降噪矩阵：默认值 50 不注入·past 跳过·死亡 emoji·非 active 成员过滤·存量空串·nil 容错/模板联通：槽位 P1·渲染序 P0 后·空 context 无标题）；既有测试零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.308.0（versioninfo.rc 经 sync-version.ps1 同步，登记性不进 exe）；产物=exe 50386432B SHA256=5da6e324b86a78135a09937952116b229db558d577ccad10b60159a13a15aa65（releases/gaea-v4.308.0.exe + SHA256SUMS-v4.308.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t5 余项：组织状态顶层差分 UI（organization_states 输入契约已备）/关系图谱升级（§7.5 节点三类边四层）/职业管理页（§7.6，CareerDefinition markdown 解析死做不做已裁决）；t4-C3 余项（partial 局部重写/场景级工程/版本历史面板）；t6 提示词工坊/t7 前端统一接线未开工；真机走查等闲置窗口。

## v4.307.0 · t5 首刀：角色状态机差分更新器（水位守卫+存活级联+关系差分）（2026-09-15）
> MuMu 蒸馏线 t5 角色状态机域首刀（§3.3~§3.6/§7.2/§7.3）。契约 v4.278 已落，差分器是缺口——旧 sync 只把 NewState 直写 Status（心理文本写进离散枚举字段），无水位守卫/级联/关系差分/亲密度算法。**落地**=①差分更新器（新 internal/characterstate 纯函数 ApplyChapterDiff）：三阶段严格顺序（存活短路+级联→心理→关系→职业→组织成员→组织自身），七条不变量全实现（两条水位单调守卫/存活短路/**存活级联三件套**〔关系 past+EndedAt 保留行、成员终态+LeftAt+Notes 时间线、心理文本化同步推水位〕/职业钳制/MaxSubCareers=2 修 MuMu 四处不一致/副职业水位守卫 gaea 新增）②亲密度算法：MuMu 全词典+**最长匹配优先禁子串累加**（修「不信任」-5 失真→正确 -15）+LLM DeltaHint 优先词典兜底③关系差分：双向无向语义/新建基线 50+delta/描述追加时间线 History④分析模板 character_states 扩 survival_status+relationship_changes+**稀疏差分纪律**（让 LLM 默认什么都不填）⑤syncCharacterStatesV2 委托差分器（survival 映射 gaea 值域）。**测试**=characterstate +5（级联全链/水位守卫/关系双向/词典修缺陷/职业组织矩阵）；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@680、版本三处 4.307.0；产物=exe 50370048B SHA256=1f2d597969f28dea42b28273398d78cbb8a66fe3d18ba7d6a95c1cecbe7da2b1（releases/gaea-v4.307.0.exe + SHA256SUMS-v4.307.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t5 余项：状态回灌 SceneBible/chapter-generate character_states 槽位/组织顶层差分 UI/关系图谱升级；t4-C3 余项；t6/t7。

## v4.306.0 · t4-C4 分析标注层：keyword→正文偏移（坐标反投影）+ 读取绑定（2026-09-15）
> MuMu 蒸馏线 t4 情节分析域第三刀（§3.10/§8.4 A1/A2，契约 v4.278 已落）。C1 的 keyword 锚点坐标从未回投正文，本刀补标注层为内联高亮备好数据源。**落地**=①定位引擎四段式（LocateKeyword）：精确 rune 匹配→去标点匹配+**坐标反投影**（indexMap 修 MuMu 缺失导致的坐标偏移）→长 keyword 前 15 rune 前缀→未命中 -1；②构建规则（BuildAnnotations）：锚定类 hook/foreshadow/plot_point 定位失败**即丢弃**（对 MuMu「留 -1 错位回查」缺陷 L746 的对策）；hook Content 原文摘录可兜底锚点、foreshadow/plot_point 只认 keyword（转述不可定位）；suggestion 无锚点 Pos=-1 仅列条目；伏笔带 planted/resolved tags；③落盘 analysis/annotations/<MMM>.json+Analyze 主链接线（容错）；④读取绑定 NovelChapterAnnotations（679→680）缺档且有 V2 分析按需重建（存量分析免重跑）；前端内联高亮随 t7。**测试**=analysis +3（四段式定位/构建矩阵含丢弃规则/空载荷）；既有零改动。**门禁**=ci.ps1 全绿、drift OK@680（锁 501→502）、版本三处 4.306.0；产物=exe 50335744B SHA256=f541f3059508380f54c69c79915c3c75b817c8ca4ca262864a40566f79fdd2a3（releases/gaea-v4.306.0.exe + SHA256SUMS-v4.306.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=前端内联高亮随 t7；t4-C3 余项；t5~t7 未开工。

## v4.305.0 · t4-C3 收尾：整章重写前端消费（对比确认 UI）+ 建议读取绑定（2026-09-15）
> v4.304 重写链路只有后端六绑定零前端消费——用户到不了的功能等于没有。**落地**=①NovelChapterSuggestions(chapterNum)（678→679）：读 analysis-v2 该章 Suggestions 供重写勾选，无分析返回空数组正常态；②RewriteModal 三态：表单（建议勾选清单〔空→提示先分析退化自定义〕+自定义要求+重点方向五选+保留元素+目标字数；source 按勾选自动判 custom/suggestions/mixed）→运行态→结果态（相似度 Tag+字数变化+新全文；应用并写回正文〔回调刷新编辑器〕/放弃版本；应用后恢复原文入口）；③CreatePage rail「整章重写」按钮+onApplied 刷新正文；④mock +7 诚实空态。**测试**=vitest RewriteModal 5 例（建议加载/逐参提交/统计渲染/应用恢复/放弃关窗）；既有零改动。**门禁**=ci.ps1 全绿、drift OK@679、spaceBindings 锁 500→501、版本三处 4.305.0；产物=exe 50319872B SHA256=3cc6238656161c4f69022755a06e202079af5fbcd7543563b1f3e4dbfd32fa58（releases/gaea-v4.305.0.exe + SHA256SUMS-v4.305.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=partial 局部重写/场景工程/版本历史列表面板；t4-C4 锚点标注；真机走查等闲置窗口。

## v4.304.0 · t4-C3 首刀：驱动式整章重写 + 版本库（应用/丢弃/恢复）（2026-09-15）
> MuMu 蒸馏线 t4 情节分析域第二刀（§5.2/§8.3 C3，契约 plot_v2.go v4.278 已落）。C1 结构化建议产出后补重写消费链。**落地**=①重写引擎（新 internal/rewrite 纯规则）：NormalizeRequest（来源校验/字数 3000 钳 [500,10000]/未知 focus 丢弃）+BuildModificationInstruction（四段固定顺序：改进问题〔越界静默跳过〕/自定义要求/重点方向五选/保留元素逐条）+CleanRewriteOutput（8 前缀+成对引号 rune 级剥离）+ComputeDiff（difflib 以本地 textsim Dice 替代）；②版本库（rewrites/<n>/index.json 无全文下发+<id>.json 按需读，ID 自动生成，时间倒序）；③六绑定（672→678）：NovelChapterRewrite（建议从 analysis-v2 取〔缺分析显式报错〕→指令→模板 rewrite-chapter→LLM 温度 0.7→清理→diff→版本 completed **不自动落章**）+List/Get/Apply（写回+幂等）/Discard（快照保留）/Restore（写回原文快照+审计=gaea 强制增量，MuMu 无 restore）；④场景级工程章显式拒绝下刀放开；前端消费面随 UI 刀补。**测试**=rewrite +4/project +1/app +4（桩 LLM e2e 全链含恢复审计/守卫/丢弃/形状锁）；既有零改动。**门禁**=ci.ps1 全绿、drift OK@678、spaceBindings 锁 494→500、版本三处 4.304.0；产物=exe 50312704B SHA256=c27ff6d3fb3bf0909ed2d46f7e76a3a7c6afec1d88121200f058a2baa1b625b3（releases/gaea-v4.304.0.exe + SHA256SUMS-v4.304.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=partial 局部重写/场景工程整章重写/前端对比确认 UI；t4-C4 锚点标注；真机走查等闲置窗口。

## v4.303.0 · t3-P2 记忆召回消费：语义检索进 SceneBible 记忆区段（长程一致性全闭环）（2026-09-15）
> MuMu 蒸馏线 t3 长程一致性域第三刀（§3.3/§3.4/§12.1/§12.4-1/§12.4-2）。P1 生产者已按章落盘，本刀接召回——**分析自动回填+语义召回注入的长程一致性闭环全通**。检索底座复用 semantic.Store（Ensure 快照比对）+本地 embedding（Herdsman bge-m3），不引 ChromaDB（§12.3）。**落地**=①结构化 query（§12.4-1）：人物[:8]/关键事件[:6]/叙事目标/情绪/本章要求[:150] 分段限额拼接整体[:800]，不嵌大纲原文；②四段流水线（§3.4）：阈值筛选 storyMemoryThreshold=0.35（归一化余弦语义，不照抄 MuMu 未归一化 L2 的 0.6）→全低分兜底 top3→注入 topK=8，三常量唯一声明；③SceneBible 增 Memories 字段+「相关记忆（按相关度）」区段（行 `- (相关度:0.82) 内容[:120]`，区段预算 400，空不出标题，参考信息非硬约束）；④索引维护：Ensure 增量+Stale 清死向量（规避 MuMu D1 向量残留）+kind=story_memory|<项目目录> 隔离+当前章排除；⑤CreateChapter 接线全链容错（无库/无 embedding/无记忆→nil 不出区段绝不中断生成；20s 超时）；零绑定面。**测试**=app +4（query 限额/流水线三态/行格式/降级）+novelcontext +2（区段渲染/预算/空态）；既有零改动。**门禁**=ci.ps1 全绿、drift OK@672、版本三处 4.303.0；产物=exe 50259456B SHA256=de5cd5ef359df1e3d04bc80b666d23dc405f7d1fdca1e977ce3949c9f78d3485（releases/gaea-v4.303.0.exe + SHA256SUMS-v4.303.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（观察池）**=t3 三刀收官；真机检索质量（相关度分布）待真实长书评估 threshold；记忆质量可评测（候选 2）保持开放；t4-C3/C4、t5~t7 未开工。

## v4.302.0 · t3-P1 记忆生产者：规则表抽取 StoryMemory 按章落盘（自动回填闭环）（2026-09-15）
> MuMu 蒸馏线 t3 长程一致性域第二刀（§2.3/§12.1/§12.2/§12.4-4）。t4-C1 解锁 V2 载荷后补上「分析→结构化记忆」抽取环节——**分析一章，设定库自动回填闭环咬合**（市场痛点：Novelcrafter 手填/Sudowrite 书长极限，调研档 §9）。**落地**=①types.StoryMemory 字段子集+确定性 ID `<MMM>-<type>-<ordinal>`（§12.2，幂等前提）+五类类型常量+is_foreshadow 三值；**无 Position 字段**（依赖未落地的 t4-C4 偏移标注，不留恒 0 死字段 D11）；②提取规则表纯函数 ExtractStoryMemories（六类对齐 MuMu :305-486）：chapter_summary 必有（summary→前 3 推进点→正文前 300 字纯截断三级回退，固定 0.6）/hook ≥6/foreshadow 全部〔planted=1/resolved=2，strength 缺省 5〕/plot_point ≥0.6 原样/character_event 每差分一条固定 0.7/conflict ≥7 记 plot_point；门槛常量唯一声明；③按章落盘 memories/MMM-<n>-memory.json（整章替换写=重分析幂等规避 MuMu D1；空 items=删文件不留陈旧档；缺文件=空正常态）；④Analyze 主链接线 persistStoryMemories（容错注入）。**测试**=analysis +3（规则表逐条含门槛边界/确定性 ID/回退链）+project +1（回环/空写删文件）；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.302.0；产物=exe 50247168B SHA256=02218224c7e46554928d4e641819d25960ab0e82b4cbda9e87215c11c290db4c（releases/gaea-v4.302.0.exe + SHA256SUMS-v4.302.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t3-P2 召回消费（semantic kind=story_memory+SceneBible 记忆区段+三常量+结构化 query）——生产者已就位，召回接上即全闭环；t4-C3/C4；真机走查等闲置窗口。

## v4.301.0 · t4-C1 分析代理 V2 化：9 维结构化 + 三维评分联动 + analysis-v2.json 落盘（2026-09-14）
> MuMu 蒸馏线 t4 情节分析域首刀（§8.3 C1）。**市场调研直接抬升优先级**（调研档 §9 增量轮：Novelcrafter Codex 手填摩擦/Sudowrite 书长极限/蛙蛙设定库——自动设定库回填是行业痛点，V2 载荷是回填数据底座）。**落地**=①模板 V2 九维化（原地升级）：hooks/conflict/emotional_arc/character_states 差分/plot_points/scenes/pacing/文白比+scores 三维+suggestions 联动+summary；t1-P2 伏笔追踪约束与候选槽位原样保留（ForeshadowHit 契约零改动）；②服务端权威归一 normalizeAnalysisV2：三维钳位 [1,10]、**overall 一律重算**（ComputeOverallScore 唯一实现=禁安全分机制保障）、建议条数 SuggestionCountForOverall 只裁上限不代拟；③落盘 analysis-v2.json（UpsertAnalysisV2 按章号覆盖/升序/原子写；容错不影响分析返回）；④旧 wire 零破坏 deriveLegacyAnalysis（AnalyzeChapter 返回键零变化前端零改动）；syncCharacterStates 升 V2 差分（空 NewState 不写）。**测试**=analysis +4（钳位重算裁剪/分档矩阵/legacy 映射逐键/空载荷安全）+project 落盘回环；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.301.0；产物=exe 50229760B SHA256=739399a4a2098e63937246dc2a1444634b9083d7e01f17404523fd901b582f83（releases/gaea-v4.301.0.exe + SHA256SUMS-v4.301.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t3-P1 记忆生产者（已解锁：StoryMemory+规则表+从 analysis-v2 抽取落盘=自动回填闭环）；t4-C3 驱动式重写；t4-C4 锚点标注高亮；真机走查等闲置窗口。

## v4.300.0 · t3 首刀：章节前文摘要窗口预算化 + 回退链 + 反重复约束（2026-09-14）
> MuMu 蒸馏线 t3 长程一致性域首刀。§11.3 对照显形的 gaea 自身缺口：章节生成 prevSummary 遍历**全部**前章、每章 200 rune **无累计上限**（200 章≈40k rune prompt 前缀），且不参与 ctxBudgetTotal 预算体系。**落地**=①最近 10 章窗口（buildPrevSummaryWindow 纯函数）：只注入本章之前最近 10 章（对齐 MuMu :1375），章号升序确定性拼接，单章 180 rune 截断（:1408），空返回跳过槽位；②窗口声明头「仅含第 X～Y 章概要，更早剧情同样有效」——部分视图语义显式化，防模型把窗口外既定剧情当不存在；③摘要回退链（§12.4-5）：大纲 Summary → 章节摘要文件（chapters/NNN-summary.json，原实现从未进前文窗口）→ 该章跳过不占位；④反重复约束（§12.4-6）：create-chapter 模板 must 增「前文摘要仅作衔接参考，直接推进新进展不得复述已完成剧情；窗口外既定剧情同样不得矛盾」。**测试**=app +4（200 章窗口过滤+分隔计数+声明头/单章截断/回退链含整章跳过/乱序升序+三类空态+小 limit 钳位）；既有测试零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.300.0；产物=exe 50211840B SHA256=344b72d9228fe53ea774253b7c7f0fbf0096036382bc8411c30623085b0e6bd6（releases/gaea-v4.300.0.exe + SHA256SUMS-v4.300.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（t3 下刀）**=t3-P1 记忆生产者（StoryMemory 契约+入库门槛规则表+落盘，依赖 t4 产出 AnalysisResultV2，建议与 t4 排序联动）；t3-P2 语义召回消费（semantic.Store kind=story_memory+SceneBible 记忆区段+三常量）；真机走查等闲置窗口。

## v4.299.0 · 伏笔面板接调度可视面：清理入口/统计/紧急度 Badge/同步结果（t1-P4）（2026-09-14）
> MuMu 蒸馏线 t1 伏笔域第四刀（收官刀）。P1~P3 全在后端，前端伏笔面板仍是旧三态视图。**落地**=①SyncResult 上绑定面（v4.297 欠账）：analysis.Agent 捕获最近一轮同步结果，新绑定 GetLastForeshadowSync（面 671→672），回收/新埋/内容匹配/跳过计数+skippedReasons 全量（D3 可见不静默），未分析显式报错；②GetForeshadows 附带 urgency 运行时投影（UrgencyLevel/ClassifyResolve 共享入口；A5 不落库、D7 后端算；回收/废弃不带键；currentChapter 自动）；③ForeshadowPanel 四件：后端统计行（分状态口径+超期红 Tag，失败降级前端计数）/紧急度 Badge（已超期红·急需橙·需关注金+tooltip 时机中文）/生命周期清理折叠区（重分析前清理/删除本章〔只删分析默认开〕/项目重置，全 Popconfirm+自动重载，文案明示手动条目永不批量删除）/上次分析同步常显卡（计数+前 2 条跳过原因+溢出注记）；④A5 护栏=写回经 stripForeshadowUrgency 剥离投影（形状锁测试）；⑤契约面=bridge/novel.ts 四载荷类型+签名、types.urgency?、mock +5 诚实空态、load() 增量绑定缺失降级。**测试**=Go +2（投影矩阵/lastSync 未分析报错+D3 可见；SyncForeshadows 按规格签名转正导出）+vitest 面板 14/14（+6：统计行/Badge/剥离/同步直显/删除逐参/清理逐参）。**门禁**=ci.ps1 全绿、绑定面 +1 drift OK@672（spaceBindings 锁 493→494）、版本三处 4.299.0；产物=exe 50,209,280B SHA256=1209d0f139ac7d47d2f92738f1e7a126585947539b828bb3063dc3b7a28dc623（releases/gaea-v4.299.0.exe + SHA256SUMS-v4.299.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（观察池）**=t1 伏笔域四刀收官；真机走查闭环一条龙等闲置窗口；SyncResult 持久化不做（会话内诊断面）。

## v4.298.0 · 伏笔生命周期清理与统计 + Lint 扩两码（t1-P3）（2026-09-14）
> MuMu 蒸馏线 t1 伏笔域第三刀（spec §8.1/§5.1/P3.4）。P2 打通分析自动回收后，重分析/重新生成场景缺「干净重来」入口——手工条目与分析产物混居一张登记表，批量清理要么删不掉要么误伤作者资产。**落地**=①三个清理入口（新 `internal/app/foreshadow_cleanup_handler.go`，语义严格区分）：DeleteChapterForeshadows(chapterFile, onlyAnalysisSource)（删埋入∨回收本章条目，默认只删分析来源）；CleanChapterAnalysisForeshadows(chapterFile)（重分析前：只删「分析∧埋入本章」+ 回退本章回收条目 → planted 清回收痕迹）；ClearProjectForeshadowsForReset()（删全部来源；手动条目**重置不删除**→pending 清章节关联与时间戳）；关键不变量=source_type 是批量清理唯一判据、手动条目永不批量删除（存量无 source_type 按手动保护）；统一返回 ForeshadowCleanupResult{deleted,rolledBack?,resetManual?}；②统计 GetForeshadowStats(currentChapter)（spec §5.1）：分状态计数+longTermCount+overdueCount（存活×ClassifyResolve 唯一入口，currentChapter≤0 自动按已写章数），resolved 别名归一并入，Total=分桶和；③Lint 扩两码 5→7（spec P3.4）：overdue（medium，计划回收章<已写章数，超期条目正以硬约束进生成上下文应尽早处置）+unplanned（low，无 target_resolve_in 且埋入≥10 章，长线豁免；与 stale 差异=「缺计划字段无法按章调度」，建议动作不同两码并存），既有五类零改动；④有意偏离=不引 SourceAnalysis 引用分支（gaea 不写该字段，引入即 D5 死代码同款）、§8.3 分析记录联动无落盘目标暂不做。**测试**=app +6（三入口 e2e 来源护栏/回退含 partial/重置字段清空、统计别名归一+超期口径 2→3、Lint 新两码矩阵+既有回归、camelCase 形状锁）。**门禁**=ci.ps1 全绿、绑定面 +4 drift OK@671（NovelB 门面 4 委托，spaceBindings 全归 play 锁 489→493）、版本三处 4.298.0；产物=exe 50,193,920B SHA256=489ef004ba0bc2cbd9f54eebd798640413e7283da51e6711d42dfad770a437c5（releases/gaea-v4.298.0.exe + SHA256SUMS-v4.298.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t1-P4 前端（伏笔面板接清理/统计/新体检码+Badge 用后端 urgency，四绑定 mock 随消费面补）；真机走查（分析→回收→清理闭环，等闲置窗口）。

## v4.297.0 · 伏笔分析驱动自动回收：三级匹配闭环 + 候选清单三层渲染（t1-P2）（2026-09-14）
> MuMu 蒸馏线 t1 伏笔域第二刀（spec P2，闭环核心：分析侧此前只会「描述精确相等」匹配回收，模型记不住该回填哪个 ID）。**落地**=①纯函数层（新 `internal/types/foreshadow_match.go`）：`WordOverlap`（rune 级 2/3-gram Jaccard 加权 o2*0.4+o3*0.6）+ `MatchForeshadowByContent` 六策略加权（标题族取最大 1.0/0.95 后缀剥离/0.8/0.75 包含/n-gram×0.7、关键词命中 0.75、内容相似 ×0.6、引用章号+0.15/分类+0.1/角色 Jaccard+0.1 累加；采纳=严格大于且 ≥0.5，同分取先出现=最早埋入）；②同步层重写（新 `internal/analysis/foreshadow_sync.go`，消费 v2 契约 `types.ForeshadowHit` 替代旧 ForeshadowAction 三动作）：回收三级匹配=精确 ID（查不到**禁止回落内容匹配**〔spec §1.2 原则 3，关键回归〕）→内容兜底→跳过不新建（只有埋入会创建记录）；已回收不重置（如实区分本章/历史章）、pending/abandoned 不可回收、hinted/partial 可回收（D15 同源）；埋入两道防重（稳定 ID+同章同题分析条目）+每章新建上限 5（只约束新建，types.ForeshadowMaxNewPerChapter 唯一声明）+评分派生唯一规则 Importance=min(Strength/10,1)（缺省 5/钳 1-10）+计划回收章缺失不猜值；`SyncResult` 完整含 `SkippedReasons` 六类 Kind（修正 MuMu D3 静默跳过），汇总进 slog；③分析侧候选清单三层渲染（新 `internal/analysis/foreshadow_prompt.go`，替代整包 JSON 直塞）：L1 本章必须回收（逐条带「⚠️ 回收时 reference_stable_id 填写: {id}」紧邻指令）/L2 超期 ≤5/L3 其他 ≤10+溢出注记，分层走 `types.ClassifyResolve` 唯一入口（D9），include_in_context=false 不进 prompt 面（登记表同步池不受影响），auto_remind=false 不在分析侧消隐（D8 只管生成侧 L3）；④`prompts/analysis-chapter.json` 升级：foreshadows 输出换 v2 字段（title/content/type/strength/subtlety/category/estimated_resolve_chapter/reference_stable_id/keyword 锚点），候选清单槽位升 P1，任务指令+三条强约束（标题禁加「回收」后缀/reference_stable_id 回收必填/estimated_resolve_chapter 埋入必填）+伏笔 ID 追踪自检条款（spec §6.1）；写入口径状态=revealed 不变，零绑定面 drift OK@667，前端对分析返回 foreshadows 形状零消费零破坏。**测试**=types +4（WordOverlap 表驱动含 rune 安全/后缀剥离/六策略矩阵含 0.5 边界与同分取先/平局确定性）+analysis 同步 15 例（4 例迁移新契约+11 新增：无效引用不回落/无匹配不新建/已回收不重置/pending 拒收/hinted 可回收/上限 5 且更新不占额/空内容/标题防重第二道/同批不双吃/未知类型容错/评分缺省钳制）+渲染 4 例（三层各就各位/排除口径与 D8/折叠上限与溢出注记/空态与 D14 截断）。**门禁**=ci.ps1 全绿、drift OK@667（零绑定面）、版本三处 4.297.0；产物=exe 50,173,440B SHA256=d107e3ae8369c9b250a9976061edcf39eb613788eaae41d66ec8fc69f0bf5306（releases/gaea-v4.297.0.exe + SHA256SUMS-v4.297.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t1-P3 清理入口/Lint 扩展 overdue·unplanned/统计口径；t1-P4 前端（表格/紧急度 Badge 用后端 urgency）；真机走查（书源线+本域，等闲置窗口）。

## v4.296.0 · 原罪·书源线收官 t5：成书「送小说导入」+ EPUB 导出（2026-09-14）
> sin 书源线最后一刀（t2~t5 自 DSH 移交本线执行），t1~t5 至此收官。**落地**=①成书行「送入小说」：确认后 sin 前端直调既有 ImportNovelBookEx（full 全本，TXT 本是干净章节文本走同一落库链），成功报章数并派发 NAVIGATE {page:'novel'} 跳书架（零新后端耦合）；ImportNovelBookEx 按「legacy 直调转正」升进 bridge NovelBindings 契约面（含 NovelImportResult 载荷，facets 归 play，drift legacy 清单移除）；②SinBookSourceBookExportEpub（SinB +1→21，绑定面 667）：成书 TXT→同名 .epub 同目录（覆盖=重导出），转换=自组装格式的确定性逆解析（书名/作者/简介头+「第N章」章头+非空行=段）→ go-epub；护栏抽 sinGuardBookPath 共用；删除成书连带清理同名 .epub；③mock 诚实拒绝两件。**测试**=Go +4（解析回环/导出+护栏/连带清理）+ vitest +2（逐参对齐+NAVIGATE 断言；EPUB 成功提示），sin 域 86/86。**门禁**=ci.ps1 全绿、drift OK@667、版本三处 4.296.0；产物=exe 50,127,872B SHA256=DD72F0781F3E3C9E810EC7F696185730E49DD740532F2E08EB90FE7514A3736E（releases/gaea-v4.296.0.exe + SHA256SUMS-v4.296.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（观察池）**=真机一条龙走查（书源卡+卡片面+工具链，状态化假站配方等闲置窗口）；断点续传/cf-bypass/真实规则出厂维持不做。

## v4.295.0 · 原罪右栏卡片化：五页签并排改手风琴卡片堆（2026-09-14）
> 用户反馈「五个标签并排太多，参考办公板块右侧面板做卡片化」。**落地**=五个功能卡（角色/大纲/设定/插图/书源）卡头常显纵向排列、一次只展开一张（办公右栏「一次聚焦一个视图」同型；展开卡高亮+箭头旋转+计数徽标，书源卡无计数=清单自管）；新增面板头一行（「创作面板」+问号气泡收于此）；展开卡即滚动容器，删旧页签条与 .sin-side-body 包装层；宽度拖拽/底稿编辑/画廊预览行为零变化；展开卡沿用 gaea.sin.panelTab 旧值直接迁移；卡头补 aria-expanded+容器 aria-orientation=vertical（纵向 tablist 合法）。**测试**=sin 域 84/84 全绿（卡头文本与 role 语义保持，既有断言零改动通过）。**门禁**=ci.ps1 全绿、绑定面零变化 drift OK@666、版本三处 4.295.0；产物=exe 50101248B SHA256=（releases/gaea-v4.295.0.exe + SHA256SUMS-v4.295.0.txt；桌面副本同哈希；冒烟 200 过）。**未做**=sin 书源线 t4 工具接线 / t5 送小说导入+EPUB 导出；真机走查等闲置窗口。

## v4.294.0 · 原罪·书源接线 t3：右栏「书源」页签（2026-09-14）
> sin 书源线 t3（t2~t5 自 DSH 移交本线执行）。t2 六绑定接成原罪用户可用的取书面。**落地**=①原罪右栏第五页签「书源」（sinPanelState 页签记忆兼容旧值）；②SinBookSourcePanel 自包含流程：搜书→候选表（HasRule=可下载，免规则「仅参考」禁用=后端 fail-closed 的 UI 同款诚实；warnings 直显）→目录预览（总数+首末样例）→范围选择（默认全本钳 [1,total]）→下载成书（进度每 20 章上报+取消；关面板不打断后台继续；事件 sin-booksource:<jobId> 三路 progress/done/error 必退订）→成书清单（新→旧+Popconfirm 删除）；③dev mock 诚实空态（mock/sin.ts +6）；④events.ts 登记 SIN_BOOKSOURCE_PROGRESS 通道（后端事件 24→25）。**测试**=vitest +6（三路事件/起跑五参对齐/取消/删除确认/免规则禁用）+SinSidePanel 页签断言更新（sin 域 84/84）；坑复训=antd 两字按钮插空格（「取 消」「删 除」）、listitem 可访问名来自 title 属性。**门禁**=ci.ps1 全绿、drift OK@666（t2 已带零新绑定）、版本三处 4.294.0；产物=exe 50,098,176B SHA256=4C0D8C0B38AEC991EAC30D8DEE648DC06A389640521BBACB305F385A22294614（releases/gaea-v4.294.0.exe + SHA256SUMS-v4.294.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t4 book_search/book_download 进 sinToolOrder；t5 送小说导入（直调 ImportNovelBookEx 零后端耦合）+EPUB 导出；真机取书走查（状态化假站配方等闲置窗口）。

## v4.293.0 · 伏笔分层注入：按计划回收章调度生成上下文（t1-P1）（2026-09-14）
> MuMu 蒸馏线 t1 伏笔域消费方第一刀（契约 v4.278 已落，生成侧消费仍是单层平铺——计划回收章这个本域最大增量零消费方）。**落地**=①纯函数层（新 `internal/types/foreshadow_urgency.go`）：`UrgencyLevel`（0-3 运行时算不落库，D15：partially_resolved/hinted 也有回收压力，缺计划章不猜值不假超期）+ `ClassifyResolve` 四值 + `ForeshadowLayerOf` 分层唯一入口（L1 必须回收/L2 超期/L3 近期参考/L4 本章计划埋入/L5 无计划兜底〔gaea 扩展防存量回归〕）+ 调度常量唯一声明（lookahead=5 上限 20、remind 缺省 5、急需 ≤2、每章新建上限 5）；②两层修正：include_in_context=false 全层排除，auto_remind=false 只抑制 L3（D8：L1/L2 硬约束不消隐）；远期不注入；③四层渲染（新 `internal/app/foreshadow_context.go`）：替代旧「未回收伏笔（创作约束）」单层，模板逐条对齐 spec §4.2（D14：省略号按长度判断），L2≤3、L3≤5、总量≤15、整区含标题≤1600 rune，消耗顺序 L1→L2→L3→L4→L5，层内确定性排序，空层不输出标题，区段头带「调度规则」约束行；L5 兜底承接存量无 target_resolve_in 条目（严格四层会让全部已埋伏笔从注入消失——回归）；④接线：`resolveTargetChapterNum`（显式/分支父节点/顺延，与 ensureChapterNode 同源复用）供分层按正确本章章号判定；⑤场景圣经（novelcontext）同步分层优先采样，硬约束层带 `[本章必须回收]`/`[已超期N章]`/`[本章计划埋入]` 标注先于参考层，D9 消除（全仓分层逻辑唯 types 一处）；⑥副本清理：删 `docs/mumu-distill/`（与 docs/distill 逐字节一致，v4.278 观察项销账）。**测试**=types +4 / app +7 / novelcontext +1（抓出超期章数负号 bug）/ 既有 context 测试全量迁移分层口径。**门禁**=ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.293.0；产物=exe 50,061,312B SHA256=DDEFF9A62EC67F470262728874D9E73ED3AA93C44DD8310B42A5BB41F1541E51（releases/gaea-v4.293.0.exe + SHA256SUMS-v4.293.0.txt；桌面副本同哈希；冒烟 200 过）。**未做（下刀）**=t1-P2 分析驱动自动回收（MatchByContent 六策略+SyncForeshadows 三级匹配+分析 Prompt 候选清单）；t1-P3/P4；拆书线观察池照旧。

## （非版本刀）Windows 原子重命名根修：fileutil.RenameWithRetry（2026-09-14）
> 「Access is denied」rename 假红类（TestEditFileBasic/GaeaRewindTurn0/Schedule index.json/config Save 等，本会话 ci 咬两次）根修：Windows 上 AV/索引器/备份进程短暂持有目标文件时，覆盖式 rename 报 os.ErrPermission 且绝大多数瞬时——隔离复跑是止血，本刀根治。**落地**=①`fileutil.RenameWithRetry(src,dst)`：权限类错误（errors.Is os.ErrPermission）按 5/10/20ms 退避重试 3 次（共 4 次尝试），其他错误原样快速失败；②`fileutil.AtomicWrite` 内部改用（config/会话/记忆库等共享写路径全受益）；③schedule index.go/project.go 两处实测假红点迁移。**测试**=fileutil +3（瞬态持锁经重试成功〔Windows 语义〕/持续持锁穷尽报错且错误保留权限类+原数据不破坏/AtomicWrite 覆盖回环；-count=8 压力稳定）。**门禁**=ci.ps1 全绿。

## （非版本刀）测试负载 flaky 治理：ProgrammingPage / ContextView 假红消除（2026-09-14）
> 本会话 ci 三次因满负载假红复跑（ProgrammingPage「重新检查」×2、ContextView 两例超时、builtin 一次性），隔离恒绿——典型负载敏感而非真回归。**①ProgrammingPage.test**：`LOAD` 等待预算 5s→15s（满载时点击→handler→mock→渲染链路实测可逼近 10s+，断言本身确定性不变）；②**ContextView.test**：文件级 `testTimeout` 20s→40s（import 图最重〔recharts 族〕，满负载下 20s 两例超时假红，只影响本文件）。**验证**=两文件隔离 40/40 绿 + 全量 ci exit 0（vitest 段含满负载场景）。**builtin TestWriteToolsWithoutLedgerNoop**（一次性、无时序依赖）不盲改，继续观察。**Windows 原子重命名「Access is denied」类**（本轮 Schedule index.json 与 config 各一例，隔离绿）维持既有「隔离复跑」处置，根修（rename 重试）需动原子写公共路径，另案评估。

## v4.292.0 · tail×反推串联：末 N 章导入完成后一键反推续写大纲（2026-09-14）
> 拆书线按反馈项「tail×AI 反推串联」——tail 的产品意图（用末尾几章反推「接下来该怎么写」）此前只完成一半：末 N 章能导入，反推还得自己进创作间点。**落地**=①HomePage：tail 提取范围导入成功后弹「立即反推这部分大纲？」确认——确认后派发 `novel:goto-tab`（跳创作间）+ `novel:auto-reconstruct`（触发反推）两事件；稍后再说一切照旧。②CreatePage：监听 `novel:auto-reconstruct`，触发既有任务化反推流（Start 入队→轮询→确认→应用，v4.291）；tab 组件常驻渲染，监听器在事件派发时必然在位。**测试**=HomePage +1（tail 选末 10 章导入→确认弹窗→两事件派发）+CreatePage +1（事件触发任务化反推链到 Apply 收骨架条目）。**门禁**=ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.292.0；**产物**=exe 50,041,344B SHA256=7AE305A5C672BB6852F680CDD7CDEE69909470887428E5E892122AD639A244D2（releases/gaea-v4.292.0.exe + SHA256SUMS-v4.292.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=反推任务取消绑定/反向角色补建/骨架 AI 丰富化按反馈；观察池=负载 flaky 三处+SERP 首搜抖动。

## v4.291.0 · 拆书导入线：AI 反推大纲任务化（长书后台态，任务中心可见）（2026-09-14）
> 拆书线在案欠账（v4.280 起历版未做）：AI 反推是 1-3 分钟长操作（上限 10 分钟），同步绑定页面一关结果就丢。本刀接进 tasks 任务队列——任务中心可见、页面关闭不丢、轮询取结果。**① 任务化**=新 `tasks.KindOutlineReconstruct` + handler（注册进启动装配块）跑与同步绑定同一套 `outlineReconstructCore`（篇幅路由/骨架聚合/规则兜底全量语义），预览 JSON 存任务 Result、Progress 上报阶段。**② 绑定**=NovelB +2 → **660**（play，锁 479→481）：`NovelOutlineReconstructStart()`（play 空间提交）+ `NovelOutlineReconstructTaskGet()`（最近任务状态，succeeded 携带预览）；同步绑定 `NovelOutlineReconstruct` 重构出共用内核 `outlineReconstructCore` 原签名保留零变化；**explicitOverrides 点名两绑定归 novel**（*App 接收者按接收者回退会误入 core）。**③ 前端**=CreatePage 反推按钮改 Start → 3 秒轮询 TaskGet（12 分钟上限）→ succeeded 取预览走既有确认弹窗 → failed 透出；队列不可用自动回落同步绑定（老后端兼容）；反推期间页面可自由导航。**测试**=Go +2（真实 tasks 队列 e2e：queued→succeeded→mid 档 6 骨架预览；队列不可用/无任务诚实报错）+vitest +2（轮询到 succeeded 弹卷级确认+Apply 收骨架条目；failed 透出不弹窗）。**门禁**=ci.ps1 全绿、drift OK@660、wailsjs 再生、版本三处 4.291.0；**产物**=exe 50,040,832B SHA256=FC0EF9E4C935601C5543B23BB007EC34375AF58EF0BD0BF09D0C45A5D4F245DE（releases/gaea-v4.291.0.exe + SHA256SUMS-v4.291.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=反推任务取消绑定按需；tail×反推串联按反馈。

## v4.290.0 · 拆书导入线：角色名→角色库 ID 匹配（反推落库补全出场角色）（2026-09-14）
> 拆书线在案欠账（v4.281 起「角色名只进预览，角色 ID 匹配另刀」）：反推条目带角色名，大纲节点 `Characters` 字段收**角色 ID**——名字到预览为止，落库后大纲角色维度始终空白。**落地**=①`matchCharacterIDs`（app 纯函数）：精确名优先+双向包含兜底（反推「林晚」命中库里「林晚儿」）、去重保序、未知名静默跳过（角色库是用户资产不因反推编造）、库空/读失败诚实降级无匹配；②`NovelOutlineReconstructApply` 接线：命中章节点写 summary 时同步把匹配 ID **合并**进 `node.Characters`（既有 ID 在前、新增追加、去重——手工配置不冲掉）；预览不可见该字段（不序列化），只落库生效；零绑定面变化。**测试**=app +3（匹配矩阵 / 端到端：60 章导入工程+建角色库→apply 收 ID / 既有 ID 保留在前）。**门禁**=ci.ps1 全绿、drift OK@658、版本三处 4.290.0；**产物**=exe 50,031,616B SHA256=B3E44A0207695E246EF95010C5FC70454BED2B37AD4F242C42DB04289DCDD6D0（releases/gaea-v4.290.0.exe + SHA256SUMS-v4.290.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=长书后台任务态 / tail×反推串联 / 反向角色补建（按反馈）。

## v4.289.0 · oh-story T6：书级文风档案（style.md 协议对齐）（2026-09-14）
> oh-story 线最后一项 T6「风格档案协议对齐」（上游 story-deslop/references/style-resolution.md ↔ gaea 文风指纹），**T1~T6 至此全部收官**。**落地**=书级文风档案 `<项目目录>/style.md`：作者显式文风偏好注入章节生成上下文——`buildChapterContextSections` 新增文风区段（伏笔→文风→世界观，共享总预算截断）；协议口径逐条对齐上游 style-resolution：无文件不建占位不猜、一句可执行偏好也有效、空白/纯标题/「待补充」不算、区段头声明「只约束表达维度（句长/视角/标点/对话/修辞/收尾），事件事实与信息边界仍以细纲为准」（事实与表达分开裁决）、超长截断 ctxStyleBudget=1200 rune；书级白名单 `.deslop-whitelist`（v4.286 已实现）补**上游注记格式兼容锁**（`# 来源：…；用途：…` 整行忽略）；与文风指纹互补（统计基线 vs 显式偏好）。**测试**=app +3（注入+协议边界声明/无文件与占位无区段/一句偏好/超长截断）+novelstyle 注记格式断言。**门禁**=ci.ps1 全绿、drift OK@658（零绑定面）、版本三处 4.289.0；**产物**=exe 50,027,520B SHA256=4F6E8CB200FA92AF5403C905497904F02C40CE56BB63E84D25BFCE907F840442（releases/gaea-v4.289.0.exe + SHA256SUMS-v4.289.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=style.md 编辑入口按反馈；拆书线余项长书后台任务态/角色名→ID 匹配/tail×反推串联。

## v4.288.0 · oh-story T4：篇幅路由与骨架聚合——长书反推从 200 章细纲变卷级粗纲（2026-09-14）
> oh-story 线 T4（落点 internal/bookimport；设计拍板：三档骨架——短篇 <3 万字或 <5 章=章级细纲现状；中篇=每 10 章聚合 1 节点；长篇 ≥80 万字或 ≥300 章=每 30 章聚合 1 节点。阈值沿上游 length-routing 短篇上界 30000 + 长篇大部头下沿，常量即数据改之即改路由）。**① 纯函数层**=新 `internal/bookimport/skeleton.go`：`RouteTier`（长篇信号优先、零章=短篇）+ `SegmentSize`（0/10/30）+ `AggregateSkeleton`（顺序聚合成「第X-Y章」骨架节点，章标题拼接摘要 200 rune 截断）。**② 反推接线**=NovelOutlineReconstruct 计算 tier/segmentSize 进预览载荷；Stage2 章级材料照旧（AI 批/规则兜底不变），中/长篇预览 Items 替换为骨架条目（chapterFrom/chapterTo 跨度标注）；短篇逐字节不变。**③ 应用扩展**=NovelOutlineReconstructApply 骨架条目（chapterFrom>0）新建卷级参考节点（seg-NNN、根级、planned、OrderIndex 续编）；节点由反推持有——**重复应用先移除旧 seg-* 再落新**（幂等不堆积）；短篇路径零变化；混合载荷兼容（骨架条目不进章号映射）。**④ 前端**=CreatePage 反推确认弹窗按 tier/segmentSize 分叉文案（卷级参考节点 vs 章级合并）、完成消息区分。**测试**=bookimport +3（路由边界矩阵/三档粒度/聚合跨度余数截断）+app +2（60 章×600 字中篇导入工程：tier=mid、6 骨架、apply 新建 seg-001..006 planned、重复应用不堆积；短篇章级回归）+既有反推 2 例原样过。**门禁**=ci.ps1 全绿、drift OK@658（**修正**：v4.287 时 bindingNames.ts 漏登记 ImportNovelBookEx、旧数误报 OK，本刀按生成器流程重产补齐）、版本三处 4.288.0；**产物**=exe 50,025,472B SHA256=9C78EDB2C5B04FE164A96FD8AB008266B32474A596A44F6CC6370E0A9C580C34（releases/gaea-v4.288.0.exe + SHA256SUMS-v4.288.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=骨架节点 AI 丰富化（卷级 summary 升级为 AI 段落概括）按反馈；oh-story 余项 T5/T6；拆书线余项长书后台任务态/角色名→ID 匹配/tail×反推串联。

## （非版本刀）真机走查：在线搜书失败章补下 + 引擎规则编辑器 + tail 出口全链验证（2026-09-14）
> v4.283.1 假站走查配方续用（CDP 9333 + 本地 127.0.0.1 假站 + 状态化失败开关 `/unfail` + 临时规则/标记项目，走查完即删），对 v4.285~v4.287 三刀做真机验收，**全部通过零缺陷**：①**失败章补下**（v4.284）=假站第 7/19 章持续 500 → 整本导入 done 面板直显「2 章下载失败，未入库：第7章、第19章」→ 翻 /unfail 后点「重试补下 2 章」→ 补下成功「已补下 2 章，全部章节已齐」、卷尾续编落盘（007.md=源第8章、019.md=源第21章，重编口语径核实）；**发现记录**=引擎内置重试（maxRetries=2）会把一次性 500 当场救回、进不了 Failed——失败路径验证必须用持续失败开关（走查配方修正）。②**引擎规则编辑器**（v4.285）=搜索区入口 → 三引擎清单渲染 → 拨 so360 启用开关 → 保存落盘实锤（engines 文件 disabled 字段翻转）。③**tail 出口**（v4.287）=12 章 TXT 绑定直调 tail/10 → chapterCount=10 + 裁剪告警「只保留了末尾 10 章（共 12 章）」如实带出。④前端错误横幅零；夹具（规则/两个标记项目/假站）全清理。

## v4.287.0 · 拆书导入 tail 模式出口：长书只取末 N 章入书架（2026-09-14）
> 拆书导入线在案欠账（引擎 `applyExtractMode` 自 v4.279 就支持「只取末尾 N 章」，但绑定只走 full 不可达）；tail 产品意图（MuMu 蒸馏 §0.4）=**用末尾几章反推「这本书接下来该怎么写」**，长书续写参考场景不必整本入库。oh-story T4 调研后暂缓（聚合粒度需先定设计），池序顺延本刀。**① 后端**=新绑定 `ImportNovelBookEx(filePath, title, genre, style, extractMode, tailChapters)`（NovelB +1 → **658**，play）：full|tail 非法值起跑即报错；`ImportNovelBook` 原签名保留=full 委托零变化；opts 全链透传 `parseNovelFile/parseTextChapters`；引擎语义照旧——tail 按 5 的倍数向上取整、>50 或 ≥总章数降级全本不静默乱裁、裁剪告知进 warnings、落库章号从 1 重编号；EPUB 无末尾 N 章语义，tail 时报告如实告知按全本。**② 前端**=导入向导「提取范围」Select（全本/末 5~50 章步进 5）+提示语如实（tail 语义+EPUB 限制）+HomePage 解析 full/tail:N 传参+关闭复位。**测试**=Go +2（12 章取末 10→重编号 1-10+裁剪告警带出；60>50 降级全本/非法模式报错/full 回归）+vitest +3（选项表锁定/当前值渲染/EPUB 提示）。**门禁**=ci.ps1 全绿、drift OK@658、wailsjs 再生、版本三处 4.287.0；**产物**=exe 50,015,744B SHA256=5C212AD239AF68E5E842887C8605961209CC48B7684F44B5D5CF9107520AA7F8（releases/gaea-v4.287.0.exe + SHA256SUMS-v4.287.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=T4 需先定聚合粒度设计（章级 vs 卷级，建议交拍板）；长书后台任务态/逐章预览 UI/角色名→ID 匹配；tail×AI 反推串联按反馈。

## v4.286.0 · oh-story T2 内核消费：模式级判定进 novelstyle + 书级白名单豁免（2026-09-14）
> oh-story 蒸馏线 T2「去 AI 味规则分层」的**内核消费刀**（此前只落资产层 gates.json/ai-patterns.md 模型执行）；与 T1 平台评审、T3 生成门三线在体检面板汇合。零新绑定 drift OK@657、前端零改动（新判定经既有 issues 载荷自动直显）。**① 模式级判定资产**=新 `internal/novelstyle/patterns.json`（go:embed）：只收 gates.json 两族可机械判定模式——`negationFlipHigh`（门禁 B 同句高置信：「不是A，而是B」等四式 RE2 数据化，命中=high）+ `explanatoryMarkers`（门禁 G advisory：之所以/这意味着/她不知道的是/殊不知/多年以后/句首原来，命中=low+承担锚点则保留，**计分封顶 5 条**）；跨段复核制等语义层留 ai-patterns.md 由技能执行；覆盖纪律同 words.json（.gaea/skills/novel-deslop/patterns.json 整体替换，坏正则**整表拒绝不换出**）。**② 打分**=novelstyle 规则 10/11（ruleNegationFlip/ruleExplanatoryMarkers），span rune 口径与合成权重不变。**③ 书级白名单**=`<项目目录>/.deslop-whitelist` 一行一显式授权片段（#注释剔除；**无文件不建空表**）；`ApplyWhitelist` 命中与片段互含即摘 issue 并按剩余重算分数；`DeSlopRewriteEx` 替换出现位置上下文命中白名单原样保留（授权片段连禁用词替换也豁免）、after 复测与 before 同口径应用白名单（「分数不降才落盘」闸语义不变）；`DeSlopRewrite` 原签名保留零变化。**④ 接线**=三处打分/去味调用点（一键去味/生成自动去味+done 体检分/对照体检）词表+模式表+白名单三件套齐挂，自动与手动同口径。**测试**=novelstyle +6（资产自检/同句命中与不误报/advisory 封顶/ApplyWhitelist 重算/白名单文件/豁免替换/覆盖 fail-closed）+app +2（无白名单 changes>0、建白名单后豁免且授权词原样——v4 场景路由夹具；体检命中摘除分数不升）。**门禁**=ci.ps1 全绿、版本三处 4.286.0；**产物**=exe 50,010,624B SHA256=2330CBC9F7C224BF055813C7DA132E64A864B000CF807E1C54FE591379FA1BD4（releases/gaea-v4.286.0.exe + SHA256SUMS-v4.286.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做**=门禁 C/E/F 语义裁决留技能（防过度设计）；白名单 UI 管理按反馈；oh-story 余项 T4/T5/T6/写前契约硬闸。

## v4.285.0 · 书源取书 t3 收官：搜索历史 + 泛搜索引擎规则可编辑（2026-09-14）
> t3 余项两件（规格 §5 观察池驱动），**t3 至此收官**；绑定面 655→**657**（NovelB +2 归 play，锁 477→479）。**① 搜索历史**（前端本地数据）=新 `utils/bookSearchHistory.ts`（localStorage：trim/非空去重最近在前/封顶 8/损坏当空/存储禁用兜底）+ BookSearchModal 搜索区历史 chips（点击即搜）+「清空历史」+ 搜索成功即记录。**② 引擎规则可编辑**（新 `BookSearchEnginesModal`）=引擎规则是用户数据资产此前只能手改 JSON——结构化编辑（启用开关/名称/SERP 地址 `%s`=查询词/查询塑形/结果条目与标题链接选择器/删除/新增模板行），高级字段（link/linkParam/redirectHosts/crawl）不进表单但按行原样往返不丢；新绑定 `NovelBookSourceEnginesGet()`（文件缺失=空清单不报错、损坏显式报错）+ `NovelBookSourceEnginesSave(rulesJSON)`（**整体替换+fail-closed**：逐条 Validate 同装载纪律 CSS only 禁 @js:、引擎名去重、空清单拒绝防误清空、临时文件+改名原子落盘、校验失败原文件不动；规则每次搜索现读保存即生效）；入口=搜索区「搜索引擎规则」钮；mock Get 给演示清单/Save 如实拒绝。**测试**=Go +1（`TestEnginesGetSave_RoundTripAndFailClosed`：缺失空态/保存回读/坏文件显式报错/空清单/缺 name/重复名/@js: 拒绝/校验失败原文件不动）+vitest +10（历史 util 6 + 编辑器 4 + Modal 历史 chips 集成 1）。**门禁**=ci.ps1 全绿、drift OK@657、wailsjs 再生、版本三处 4.285.0；**产物**=exe 49,975,296B SHA256=2100FA0C0E5E46F33EBBEF30892FCCB9A45F61017161AEB514102B54C0A0ED3C（releases/gaea-v4.285.0.exe + SHA256SUMS-v4.285.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**状态**=书源线功能项收官（t1 后端/t2 前端入口/v4.283.1 走查补刀/t3 失败章重试+历史+规则编辑）；观察池=SERP 首搜抖动、线上源站漂移跟踪、真机补下与编辑器走查；sin 书源线 t2~t5 归并行线。

## v4.284.0 · 书源取书 t3：失败章重试（一键补下，项目端续编落库）（2026-09-14）
> 书源线 t3 首项（规格 §5 观察池驱动；用户「继续」按并行池纪律放行）。整本下载常有零星失败章（源站限流/超时），t2 只能如实报「N 章未入库」后整本重来——本刀补成一条用户动作。**① 引擎**=新 `booksource.Engine.DownloadChapters`（显式 URL 清单抓章；`Download` 编排水抽出 `fetchChapters` 共用，有界并发/重试/进度纪律零变化；空清单显式报错）。**② app**=新绑定 `NovelBookSourceImportChapters(source, projectPath, chaptersJSON)`（NovelB +1 → 655，play）：清单=整本导入 done 事件 Failed 原样回传（会话内存活，关闭即弃，不做持久化/断点续传照 §6）；同步预检 fail-closed（规则/清单/项目三查）；追加落库=现有最大 OrderIndex 续编章号 + 大纲节点 imp-NNN 追加且既有节点原样保留（**并发写核实**：outline.Agent 写路径每次重读 outline.json 无缓存副本，外部追加安全）；进度/取消全复用（同 novel-import-progress:<jobID> 通道 + append-done 终态，同 bookImportRuns 登记簿）；done 载荷 NovelBookSourceAppendResult 追加语义词（appended/totalChapters/addedWords/failed 分列，残留可继续重试）。**③ 前端**=BookSearchModal 失败面板（失败章标题直显 +「重试补下 N 章」+ 进度条可取消 + 完成态提示）；**Modal 关闭权收归组件**——全部成功自动关、带失败留面板「完成」手动关（父层不再代关，避免关 Modal 抹掉失败面板）；HomePage.onAppended 刷新书架+增量提示；mock 诚实拒绝。**测试**=Go +2（显式清单保序/空正文进 Failed/进度成功数语义/空清单拒绝；续编与大纲追加/既有节点保留/rune 字数/失败透出/全败不动大纲）+vitest +3（失败面板渲染/重试参数逐参对齐/progress/append-done 回调/完成态/无失败自动关）。**门禁**=ci.ps1 全绿、drift OK@655（spaceBindings 锁 476→477）、wailsjs 再生、版本三处 4.284.0；**产物**=exe 49,975,296B SHA256=2100FA0C0E5E46F33EBBEF30892FCCB9A45F61017161AEB514102B54C0A0ED3C（releases/gaea-v4.284.0.exe + SHA256SUMS-v4.284.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=t3 余项搜索历史/引擎规则可编辑（按反馈）；失败清单跨会话持久化（有意不做 §6）；真机补下走查（复用 v4.283.1 本地假站配方，改 fail 计数即触发）。

## v4.283.1 · 走查补刀：书源取书零注入 nil-Fetcher panic 修复（四绑定全路径）（2026-09-14）
> v4.283.0 发布后补做的 t2 真机走查（壳内 CDP + 本地 127.0.0.1 假站 + 临时规则 + 标记夹具项目，走查完即删）当场抓到 **P0**：书架「在线搜书」点搜索即 nil pointer panic——在线搜书在壳内完全不可用。**根因**=`internal/booksource/fetch.go` `newCrawler` 只兜底 Sleeper/Rand 不兜底 Fetcher，四绑定全传 `Options{}`（Fetcher=nil）→ DiscoverySearch/Engine 首个网络请求 nil 接口解引用；**单测零暴露**（引擎 34 例+app 13 例全注入假站 Fetcher，「零真网络」纪律下生产缺省通道从未被走到）。**修复**=一处根治全路径：`newCrawler` 补 `Fetcher==nil → &HTTPFetcher{}` 缺省（Client==nil 回落 DefaultClient 已内建；app 层未来注入带代理实现仍优先），Engine（Toc/Import）与 WebSearcher（Search）同走 newCrawler 四绑定一次修净。**测试**=Go +2（零注入三构造路径 fetch 必非 nil 回归锁 / 零值 HTTPFetcher 对本地 httptest 夹具真实取页端到端）。**真机复走查全链过**=搜索同表渲染规则候选（可导入+选书钮）与泛搜索候选（免规则禁用）→ 选书 → 目录预览 25 章+首8末4样例 → 范围导入落库（书架 6→7 部、正在编辑 25 章 · 1,241 字、磁盘 25 章+outline 25 节点+UTF-8 正文）→ 前端零错误横幅；夹具全清理。**观察池注记**=一次「首搜零候选零告警、复搜稳定 6 候选」的 SERP 首请求抖动（bing 选择器实测匹配线上），归既有 SERP 漂移观察池。**门禁**=ci.ps1 全绿、drift OK@654（零绑定面变化）、版本三处 4.283.1；**产物**=exe 49,959,424B SHA256=66A9765143487D3D8324AE08AAED5F9F6846B2F9151D51CF46D4DCB4682A174C（releases/gaea-v4.283.1.exe + SHA256SUMS-v4.283.1.txt；桌面副本同哈希；冒烟 /api/health 200 过）。

## v4.283.0 · 书源取书 t2：书架「在线搜书」入口（搜索→候选→目录预览→范围下载→进度→入书架）（2026-09-14）
> 续 t1（书源取书→拆书导入后端四绑定，非版本刀），本刀落规格 `docs/gaea-novel-booksource-import-2026-09.md` §5 既定下刀 **t2（前端，版本刀）**：把 `NovelBookSource*` 四绑定接成书架用户可用的取书链。**① 入口**=书架工具条「在线搜书」（与「导入小说」并列；不新增板块、不动原罪任何文件）。**② BookSearchModal（自包含流程）**=候选表同表呈现书源聚合+泛搜索（来源/kind 标注；**hasRule 是唯一可导入通道**——免规则候选标「正文不可解析·仅参考」禁用，t1 fail-closed 边界的 UI 同款诚实）；各源失败 warnings 直显不静默；目录预览（总章数+首8末4截断样例）；范围下载（起止章默认全本钳制 [1,total]+实时候节数；题材/文风沿用既有选项）；进度与取消=`NovelBookSourceImport` 起跑拿 jobId 订阅 `novel-import-progress:<jobId>`（events.ts 新增 `BOOK_IMPORT_PROGRESS`+`bookImportProgressChannel()`），进度条按每 20 章节流事件更新，取消走 `NovelBookSourceImportCancel`（canceled 类失败如实转写「已取消导入」），error 面板内直显可重试，done 带失败章清单则提示「N 章下载失败，未入库」；导入期关闭受锁、关闭必退订。**③ 报告面收口**=新 `utils/novelImportReport.ts`（`formatImportSuccess/formatImportWarnings/SPLIT_STRATEGY_LABELS` 补 **booksource→在线书源** 映射），文件导入原内联逻辑改共用——完成即 openProject+刷新书架+报告直显，与文件导入同款。**④ dev mock 诚实空态**=mock/novel.ts 补四方法（搜索空候选+说明、目录空态、导入如实抛错、取消 false），浏览器 mock 可用不假装能搜书。**测试**=前端 +15（BookSearchModal 7：HasRule 打标与禁用/告警直显/范围默认与收窄 7 参逐参对齐/progress+done 回调/error 直显可重试/取消；novelImportReport 6；HomePage 入口 1；events 频道 1，后端事件常量 24→25）。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build / vitest 全量 / E 系列 / 仓库卫生守卫）、drift OK@654（零新绑定）、版本三处 4.283.0、wailsjs 再生（gitignore 构建时产物）；**产物**=exe 49,951,744B SHA256=2D42376FF1B23C3EEB165473DE5773343B74AC7DAA751BCE0DA64CBA0266ED47（releases/gaea-v4.283.0.exe + SHA256SUMS-v4.283.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=t3 失败章重试/搜索历史/引擎规则可编辑（按反馈）；真机取书走查等用户闲置窗口（引擎 34 例+app 13 例均假站零真网络，线上 SERP/规则漂移在观察池）。

## v4.282.0 · oh-story 蒸馏首刀接线：平台质量评审（rubric 引擎 + 创作间面板）+ 生成门确定性两路直显（2026-09-13）
> 来源=用户「把 oh-story-claudecode（MIT 网文写作插件）蒸馏给小说板块」——本轮把已在库的三份资产（T1 评审 rubric / T2 去 AI 味门禁资产 / T3 生成门资产）**接成用户可用的一条链**（规格 `docs/gaea-novel-ohstory-distill-2026-09.md`；上游本地只读克隆 `clones/oh-story-claudecode`，机制一律重推导、知识资产随附 MIT 许可）。**① 平台质量评审引擎**（新 `internal/novelreview`，零 LLM/零网络/纯函数，绑定 NovelB +2 → 650）=15 个**可机械判定**维度：字数区间 / 开篇钩子 / 章尾钩子 / 预告式收尾 / 情绪节点密度 / 爽点·升级密度 / 段落节奏 / 对话占比 / 标点节奏 / 破折号 / 格式合规 / 人称视角 / 主角存在感 / 金手指提及 / 字数表述核对；每维给 `PASS | WARN | FAIL | SKIP` + **S1~S4 分级** + **原文证据（rune 区间 + 段落号）** + 改法，结论 `APPROVE / CONCERNS / REJECT` 与评审 rubric 同门槛；**语义维度（核心卖点/角色动机/伏笔回收）引擎不下结论**，外部数据缺失时**显式 SKIP 并写明原因**（如「角色库未标注主角」），**不静默给 PASS**。**② 阈值与词表是数据资产**=`internal/novelreview/rubric.json`（go:embed，四档位：通用/番茄/起点/知乎盐言；判定词表：冲突/情绪/悬念/爽点/金手指/预告收尾），可被 `<工作区>/.gaea/skills/novel-review/rubric.json` **整体替换**（与 novel-deslop 词表同款纪律），**fail-closed 校验**拒绝非法资产（未知维度/非法严重度/坏区间逐类点名且不换出）。**③ 前端**=创作间 rail「平台评审」→ 面板（档位选择器 + 结论大标与 S1~S4 计数 + 逐维结果**按 S1→S4 排序** + 证据摘录 + 黄金三问），主角名取自角色库 `role_type=protagonist`。**④ T3 生成门两路可视化**=`NovelInspector`「章节体检」区新增「写前契约：N 项待补 / 齐备」与「写后硬信号：N 项 / 未命中」+ 按严重度排序的条目（与 AI 四路并列；老后端缺省仍诚实降级「未启用」）。**⑤ 资产口径对齐**=`.gaea/skills/novel-review/rubrics/generic.json` 18 维 → **26 维**（+8 确定性引擎扩展，逐条标 `measurable/engine`）+ 新增 `deterministicEngine` 段（资产=评审协议、引擎=可机械判定子集）与 `SKILL.md`「确定性引擎」一节（15 维映射 / 覆盖文件路径 / `plot_loop` 只做爽点密度代理判定）。**测试**=Go +16（`internal/novelreview` 12 例：资产自检 / 未知档位回落与空文本 / 覆盖文件 fail-closed 与整体替换 / 字数区间双档 / 开篇三态 / 章尾钩子与预告式收尾（含 S1→REJECT）/ 情绪密度与最长平直段 / 破折号（含数字区间豁免）与省略号密度 / 盐言人称 S1 / 主角 SKIP 口径与金手指 / 字数表述双向引号核对 / **合格章节整体 APPROVE 集成夹具**；`internal/app` 4 例：档位清单 / 载荷证据与 REJECT / 错误与未知档位回落 / 主角维度开闭）+vitest +10（面板 6 / 检查器 3 / CreatePage 1）+spaceBindings 锁 **470→472**。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint / vite build / vitest / E 系列 / 仓库卫生守卫）、drift OK@650、版本三处 4.282.0；**产物**=exe 49,597,440B SHA256=83F5BFD603DB5EC1B091E006F8CACB85B37C3283FF653CCED669F44A9628EAD9（releases/gaea-v4.282.0.exe + SHA256SUMS-v4.282.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=T2 内核消费（`gates.json` 模式级门禁进 novelstyle 打分与去味）/ T4 导入结构映射与篇幅路由（长短篇骨架路由）/ T5 子代理角色卡资产 / T6 风格档案协议对齐；写前契约的「拒绝生成」硬闸与书级白名单。

## （非版本刀）oh-story 蒸馏 T3：小说·生成门补确定性两路：写前大纲契约 + 写后质量体检（oh-story 蒸馏 T3）（2026-09-13）
> 续 T1（评审 rubric 资产）与 T2（去 AI 味门禁资产），本刀落 **T3：把上游 hooks 里「写前守契约、写后查质量」的机制补进 gaea 内核**（规格 `docs/gaea-novel-ohstory-distill-2026-09.md` §2 第 1 条真增量：gaea 此前只有事后去味与体检，**没有生成前的结构契约闸、也没有生成后的确定性质量闸**）。**落地**=①新包 `internal/novelgate`（零 LLM 零网络，纯函数）：`OutlineContractIssues`（写前契约：标题/本章计划 summary/关键要点/情感基调四项齐备性，缺席按 S2~S4 分级）+ `ChapterQualityIssues`（写后确定性质量：空正文 S1 / 超长段落 S3（带行号证据）/ **电报体** S2（短句占比 >40% 且平均句长 <10 字——「把逗号长句拆碎」同属 AI 味）/ 平均句长偏短 S3 / 连续堆叠问号感叹号 S3（RE2 无反向引用，用「同类标点 ≥3 连」表达）/ 省略号滥用 S3 / 通篇句号化 S3；阈值集中成常量、口径一律 rune）；`Issue{Code,Severity,Message,Evidence}` 与评审 rubric 的 S1-S4 同轴。②接线：`RunChapterGate`（章节生成门）新增**确定性两路** `outlineContract` + `deterministic`，作为 AI 四路（analysis/review/consistency/aiTaste）的基线对照——必出、零成本、失败不阻断（找不到大纲节点返回空表）。**门禁**=本刀自身证据：go build/vet exit 0、internal/novelgate 7 例全绿、internal/app 生成门定向用例绿（全量 ci.ps1 未取全绿——并行线在制品 internal/app/novel_review_handler_test.go:117 当前为红，与本刀无关）；drift OK@648（零新绑定）；**未抬版本**（非版本刀，随下次发版入 exe）。**测试**=Go +7（契约齐备/空项分级、空正文 S1、正常文本零误报、电报体、超长段落、标点堆砌与省略号、句号化、证据随行）。**未做（下刀）**=T4 导入结构映射与篇幅路由；写前契约的「拒绝生成」硬闸（本刀只报告不阻断，避免作者被空大纲卡住）；上游「书级白名单 .deslop-whitelist」与 `gates.json` 的内核消费。

## v4.281.0 · 拆书导入 P1 接线：AI 反推大纲（预览载荷 + 幂等落库 + 创作间入口）（2026-09-13）
> 续 v4.280.0（反推引擎），本刀把 t2 P1 接成**用户可用的一条链**（规格 `docs/distill/02-book-import.md` §8.2/§8.3 首刀）。**① 后端**（新 `internal/app/novel_import_ai.go`，绑定 646→648）=`NovelOutlineReconstruct`：读当前工程章节（标题取大纲节点、正文走 `ReadChapterAsStitch`，上限 200 章）→ **Stage1 立项反推**（模板 `book-import-project`，采样前 3 章 ×2000 rune）→ **分批章节大纲**（`book-import-outline`，batchSize=5，逐批独立降级）→ 返回 `OutlineReconstructPreview`（`aiUsed` + 立项五字段 + 逐章 summary/scenes/characters/keyPoints/emotion/goal + warnings），**零落库**；`NovelOutlineReconstructApply(itemsJSON)`：按**章号**命中既有大纲节点写 summary/scene_ideas/key_points/emotion，**幂等**（同载荷连跑两次结果一字不差）、不新建/不删除节点、**不碰章节正文**、角色名只进预览不写角色 ID 字段（角色库匹配另刀）；全对不上/空载荷**显式报错不静默成功**。**降级诚实**：无模型/解析失败逐级回落规则兜底并把原因写进 `warnings`（`aiUsed=false`），失败不阻断整单。**② 前端**（CreatePage 创作间 rail「AI 反推大纲」）=一次点击 → 反推（约 1-3 分钟）→ 确认弹窗（显示推断题材/视角/目标字数 + 前两条告警 + 明确「不覆盖正文、可重复执行」）→ 应用 → 刷新大纲 + rail 消息「已应用 N 章大纲」。**③ 接线面**=NovelB 门面 +2、`bindingNames` 648（drift OK）、spaceBindings 归 play（锁 468→470）、`wailsjs` 再生。**测试**=Go +2（无模型时全规则兜底且逐章字段完整 / 幂等与作用域：只命中章号被写、未命中章不动、连跑两次字节一致、空载荷与全不匹配显式报错）+vitest spaceBindings 4 例 + CreatePage 9 例。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@648、版本三处 4.281.0；**产物**=exe 49,455,616B SHA256=7FB09E36…2A693（releases/gaea-v4.281.0.exe + SHA256SUMS-v4.281.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=导入向导的**预览 UI**（逐章核对/单章编辑后再应用——本刀为「一次反推 + 确认应用」的最短闭环）；幂等落库的任务态（`tasks` 表 `KindBookImport`，长书后台化）；角色名 → 角色库 ID 匹配；tail 模式出口。

## v4.280.0 · 拆书导入 P1 引擎：大纲反推编排 + Prompt 模板移植 + 输入节稳定序（2026-09-13）
> 续 v4.279.0（t2 P0 解析引擎），本刀落 **t2 P1「AI 反推」的引擎层**（规格 `docs/distill/02-book-import.md` §8.2 / §3.7 / §3.8）。**① 反推编排引擎**（新 `internal/bookimport/reconstruct.go`，**纯函数 + 可注入调用缝**、零 LLM 依赖）：`ProjectSuggestion`/`OutlineStructure` 具名契约；**逐字段归一化**（summary/scenes[:6]/characters[type 仅 character|organization、无 name 丢弃]/key_points[:8]/emotion[:200]/goal[:300]，一律 rune 截断；**title 与 chapter_number 强制用输入值，AI 返回值丢弃**）；**位置对齐批量归一化**（第 i 槽位取 `ai[i]`，非对象槽位直接 fallback——AI 少返/乱序都不错章）+ 数量不符**整批回退规则结构**的断言式防线；规则兜底结构逐字对齐规格（含「取正文首句作兜底 summary」）；`NormalizePerspective` 归一中文三值 + 11 个英文别名；`target_words` <1000 回落 / >3e6 夹取。**② 负反馈重试**=新 `CallJSON`（期望类型显式 object|array，避免「返对象按数组解包」的静默错；解析/类型失败把**上次失败原文截 200 字**注入下次提示；传输错误直接返回不空转；ctx 取消零调用）+ `ExtractJSON` 容忍 markdown 包裹与前后解说。**③ Prompt 模板移植**=新增 `prompts/book-import-project.json`（立项五字段）与 `prompts/book-import-outline.json`（七字段 + scenes 2-6/key_points 2-6/type 两值硬约束），按 RTCO 结构等价改写 MuMu 的五段式 Prompt。**④ 顺手补掉规格点名的引擎缺陷**=RTCO `BuildUserPrompt` 原按 **Go map 遍历**输入节（`internal/prompt/prompt.go:194`），**同模板两次渲染字节不同**——长文本无法稳定置尾、供应商 prompt 前缀缓存无法命中（阶段五 5.2 的出口判据之一）；改后 `InputDef` 增 `order` 字段并按 **Order 升序 → key 字典序**稳定排序，新模板把长文本（`sampled_text`/`chapters_text`）固定置尾。**测试**=Go：`internal/bookimport` +9 例（视角别名矩阵 / 立项字段回落与夹取 / 位置对齐与强制标题章号 / 逐字段截断与 type 归一 / 兜底结构取首句 / 重试提示携带失败原文与期望类型 / 类型不符报错 / 传输错误不重试与 ctx 取消 / JSON 容错与采样）+`internal/prompt` +2 例（**同模板 20 次渲染字节一致** / order 覆盖 key 序）。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646（零新绑定）、版本三处 4.280.0；**产物**=exe 49,405,440B SHA256=0F9BF89A…8EF4B（releases/gaea-v4.280.0.exe + SHA256SUMS-v4.280.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=app 绑定 `NovelImportReconstruct*`（真模型调用编排 + 预览载荷）与导入向导 UI、幂等落库（`tasks` 表 `KindBookImport`）与三件套生成。

## v4.279.0 · 拆书导入 P0：三级分章 + 5 级编码链（MuMu 蒸馏 t2 首刀）（2026-09-13）
> 续 v4.278.0（t1 契约落库），本刀落 **t2 拆书/导入反推 P0 阶段**（规格 `docs/distill/02-book-import.md` §8.1，**纯规则零 AI**）。**改前短板**=gaea 导入只有强标题正则（识别不到即退化成单章「全文」）；编码链 `utf8.Valid → GB18030 → GBK` 缺 utf-8-sig 与 Big5，且 GB18030 解码几乎不报错+二次校验会**静默放过误判**。**落地**=①新包 `internal/bookimport`（纯规则、零 IO）：`Decode` 5 级编码链 + UTF-16 BOM 识别 + **候选结果含替换符即判误判续探**（实测 GB18030/GBK 解 Big5 字节不返回 error 只出「材�彻」乱码，MuMu 会静默接受）；`Clean` 六步顺序敏感清洗（全角空格→两半角空格，不是删除）；`Split` 三级切分——**强标题存在时不再叠加弱标题**、弱标题需 **≥2 候选**（否则正文短行被切碎 / 整篇短文本被当成一章标题）、无标题且 >5000 字走**兜底窗口**（3000~5000 内取最靠后句读边界，找不到才硬切，标题伪造第N章）；`applyExtractMode` 末尾 N 章裁剪（5 的倍数向上取整、>50 降级 full）；`buildWarnings`（过短 <300 / 过长 >12000 / 标题重复 / 裁剪告知）。②`internal/app` 接线：`parseTextChapters` 委托新引擎并带出 `bookimport.Report`；`NovelImportResult` 增 `encoding`/`split_strategy`/`warnings`（omitempty，与既有 snake_case 形状同风格）。③前端：导入成功文案直显「编码 · 切分策略」并弹告警（≤2 条 + 计数）——「这本书是不是被切错了」当场可见。**与 MuMu 的两处有意偏离**=首标题前正文 <200 字**并入首章**而非丢弃（书名/作者信息不丢，宁留勿删）；阈值一律 `utf8.RuneCountInString`（MuMu 用 `len()`=字节，中文失真）。**测试**=Go：`internal/bookimport` +17 例（编码链含「Big5 不被 GBK 遮蔽」回归、BOM/UTF-16/兜底不 panic、清洗顺序、三级切分、前言门槛、窗口上界、三类告警、tail 取整与降级）＋`internal/app` 导入 5 例（BOM / GB18030 / 无标题 8000 字走窗口 / 弱标题 3 章 / e2e 报告字段）。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.279.0；**产物**=exe 49,397,760B SHA256=D8947306…7659E（releases/gaea-v4.279.0.exe + SHA256SUMS-v4.279.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=AI 反推编排（字段级归一化 + JSON 负反馈重试）、幂等落库与任务态（`tasks` 表 `KindBookImport`，不得用内存 dict）、导入向导 UI（tail 模式引擎已实现，当前仅 full 可达）；观察项=双实现收敛（`app.chapterNumOf` 与 `types.ChapterNumOf`）、novel api 直取 `window.go.app.App`（mock 噪声）。

## v4.278.0 · 小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）（2026-09-13）
> 两条线汇合收口：①「优化完善gaea」并行池小说线续刀=**伏笔一致性 Linter**（plan §4 并行池小说行：文风指纹 v4.277.0 已落 + 伏笔一致性 Linter）；②另一支 MuMuAINovel 蒸馏实施线（规格 `docs/distill/`）先落 t1 共享契约后停摆，本刀由 Codex 侧接管，把**在制品 + t1 契约**一并落库并按契约先行口径对齐。**①伏笔一致性体检**=新 `internal/app/novel_foreshadow_lint_handler.go`（纯函数 5 类确定性检查：ordering 回收早于埋设 / status-mismatch 状态与回收章不一致（partial 豁免）/ dangling 指向未写章 / stale 悬置 ≥10 章（长线豁免）/ duplicate 归一描述重复；报告 totalChapters/items/planted/hinted/revealed/longTerm/findings，登记表空=空报告正常态）+ `ForeshadowPanel` 新增「一致性体检」（概要行 + severity Tag 轻/中/重 + 说明 + 条目描述 + 章节引用；保留上次报告直至下次体检；Go 侧 omitempty/null 全防御）；绑定 645→646。**②t1 契约落库**（`internal/types/`）=`foreshadow_v2.go`（并集 6 态 + 计划回收章 + 来源/评分/关联/注入控制 + `ForeshadowUrgency` 含**显式 must_resolve**〔MuMu 因响应模型缺字段静默丢弃，gaea 不重犯〕+ `ForeshadowAmbiguity`〔引用失效禁止回落内容匹配〕）/`character_state.go`（三态水位守卫 + 方案 C 职业引用 + 派生 member_count + 亲密与 delta_hint 钳制）/`analysis_v2.go`（9 维 + 三维分档评分 `(P+E+C)/3` + 建议数与总分硬联动 + keyword 8–25 字约束）/`plot_v2.go`（ChapterPlan / RewriteVersion+Index / Annotation rune 锚点）/`interfaces.go`（模板解析-覆盖-渲染-校验四接口 + 两级三源 rank + 上下文节稳定排序 + DefaultContextMaxRunes=2000 与 novelcontext 同源 + Lint 接口 + ChapterNumOf）/`types.go`（Character/Organization/Relationship v2 可选字段，全 omitempty 零迁移）+ `compat_test.go`（旧 JSON 往返 + 未知字段前向兼容）。**③wire 口径纠偏（P0）**=t1 在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降为读取别名），与交接书 §2-3/§4-7 相反——会让 7 类既有消费方静默读空（统计/回收率、lint 计数、分析写回、章节上下文注入、SaveForeshadows 白名单拒收新值、narrative、前端 12 处）。按 A 方案纠回：`NormalizeForeshadowStatus` 方向=`resolved → revealed`、新增 `IsResolvedStatus` 覆盖两套 wire 值，并接进 `internal/stats`（回收率）/伏笔 lint（检查·标签·计数）/`create_chapter_handler`（已回收不再注入 prompt）/`internal/analysis`（聚合统计）；`SaveForeshadows` 白名单 3 态→并集 6 态且写入前先归一化（别名落库即 `revealed`）。**④规格入库**=`docs/distill/`（01~07 六域 + `09-impl-handoff.md`）入库并在 `docs/README.md` 登记。**测试**=Go 全量绿（types 兼容矩阵 + 两套 wire 值新用例 / app 伏笔 Lint 5 类 / stats / analysis）+vitest `ForeshadowPanel` 体检 3 例 + `spaceBindings` 锁 467→468；**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；**产物**=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**欠账**=六域实施 t2~t7 未开工（拆书导入/长程一致性/情节分析重写/角色职业接线/提示词工坊/前端统一接线；规格在库、契约就位）；t1 契约中除伏笔体检外的类型尚无消费方（契约先行，实施时按规格接线）；观察项=ChapterNumOf 双实现（建议 t3 委托统一）/`internal/stats`·`internal/narrative` 在 §4.5 归属表无 owner/`docs/mumu-distill/` 重复副本待删（`.agent-teams/` 工作区已入 .gitignore）。

## v4.277.0 · 小说·文风指纹落地面：参考档构建 + 对照体检（并行池小说线既定余项）（2026-09-13）
> 「优化完善gaea」从板块并行池挑小说线既定余项。**现状核实**=内核 `internal/novelstyle` 指纹算子（ComputeFingerprint/Delta/ScoreText 带参考档）自刀3起齐备但 app 层零消费——参考档无处落无处看、打分只走无参考档通用阈值。**落地**=①参考档 `<projectDir>/fingerprint.json`（types.StyleFingerprintFile，types 不反向依赖 novelstyle；Manager 三方法原子写）②`NovelFingerprintBuild` 全部已写章节（v4 走 ReadChapterAsStitch）ComputeFingerprint 落盘，**门槛诚实拒绝**（≥3 章且 ≥3000 字，不足中文报错不落盘）③`NovelFingerprintScore` 有参考档 ScoreText+Delta（函数词 z 向量距离，越小越像作者）/无参考档通用阈值两路可用，参考档损坏降级不挡体检④`NovelFingerprintStatus` 未构建=exists:false 正常态⑤前端 CreatePage rail「文风指纹」+StyleFingerprintPanel 受控 Modal（空态引导/8 项摘要格子+口头禅 Tag/体检大数字+Δ 基线距离+issues 列表含原文摘录；阈值同 aiTaste；全字段防御）。v1 口径=作者成稿即样本，「只学作者手改」结算学习留后续刀（面板文案如实写明不自动漂移）。**测试**=Go+6（门槛/构建回环/双路体检/拒收/遍历口径）+形状锁（camelCase 键断言+PascalCase 反例+delta=0 不被 omitempty 吞）+vitest +5+spaceBindings 锁 464→467；绑定 642→645。**真机走查**=夹具工程（project.Create 造 3 章约 4000 字）壳内 CDP 全流程：空态→构建（3 章 · 3,999 字+摘要格子）→体检（Δ 0.00 顺眼）→重建覆盖→console 零错误；坑复训=海报墙入口须 scrollIntoView 再点（否则被悬浮件拦截）/无 outline 夹具致体检钮 disabled（UI 依赖大纲树须连 outline.json 一起造）/delete-pending 删夹具须 PowerShell 复删。**门禁**=ci.ps1 run1 抓到 v4.276 存量测试缺陷当场修（ChatPage.test 语音落库断言仍锁旧 PascalCase，与 v4.276 已改 camelCase 的 useChatVoice 发送不一致，断言对齐+单文件 15/15 绿）run2 go test 段负载 flaky（日志被 tail 截丢未定包名=后台长输出禁 tail 教训再踩；run3 全量落盘复跑全绿）+lint/build/vitest 分段跑齐（vitest 2991 例全绿）/drift OK@645/版本三处 4.277.0。**产物**=exe 49,352,704B SHA256=2CB99312…802CD 见 SHA256SUMS-v4.277.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.276.0 · 绑定面 JSON 形状普查：造价参考/复盘笔记/会话组价价格带断链修复（2026-09-13）
> 「优化完善gaea」：把 v4.269 走查抓到的「Go struct 缺 json 标签→Wails 线上 PascalCase→前端 camelCase 读空」bug 类从走查撞见修一处升级为**全仓 AST 闭包普查**（绑定面 App 导出方法签名类型递归展开 × 缺标签 struct 求交）：全仓 327 个缺标签 struct 中绑定面可达仅 7 个——三个输出方向真断链当场修：`cost.PriceBand/BandSource`（GaeaCostCompose band 字段，**v4.191 起会话组价价格带卡全空+证据表 IQR 离群判定 NaN 失效**）/`costref.Note`（复盘笔记 validUntil/refCount/updatedAt 读空）/`costref.Indicator`（造价参考视图标题/均值/分位读空）；四个输入方向（AskAnswer/ChatMessageInput/TTSParams）防御补齐——**形状锁定唯一真源应落在 Go 定义，不依赖 Unmarshal 大小写宽容**。连带抓出 `useChatVoice.ts` 三处 PascalCase 发送点（线上不炸纯靠 Unmarshal 大小写不敏，wails build 再生绑定类型时 tsc 实锤），改 camelCase。**测试**=Go+6（TestWireShapeCamelCase ×4 含旧 PascalCase 数据兼容断言+TestAskAnswerWireShape）+**结构性守卫 TestWireShapeGuard**（普查逻辑测试化：绑定面可达闭包内缺标签 struct 即 FAIL，豁免清单机制；**负向验证过**——删标签 FAIL 精确报出/恢复 PASS；v4.269 修 7 个结构体没防住本轮同款，守卫是根治）。**真机走查**=CDP 桥验 GaeaCostNoteList 返回 camelCase（validUntil/projectType/updatedAt 在位）+UI 断言复盘笔记卡片全字段渲染（「材料·房建·泵送」/「有效期至 2026-12-31」/「引用 0 次」）+console 零错误+清场 remaining=0；坑=整段式走查脚本 UI 段卡死一次，拆探针分段+看门狗全链过——**走查脚本宜分段不要一次性长链**。**门禁**=ci.ps1 全绿（run1 负载 flaky 两例隔离复跑绿既有先例）/drift OK@642/版本三处 4.276.0。**产物**=exe 49,294,336B SHA256=1A12B1CA…DA869 见 SHA256SUMS-v4.276.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.275.0 · 价格带数据源调研结案：信息价「除税价」列适配（拍板池候选4 收官）（2026-09-13）
> 拍板池候选4 调研落地（docs/gaea-priceband-datasource-research-2026-09.md）。**现状核实**=价格带数据面基础设施已完整在产（CostEntry Source/Region/PriceType/PriceDate 四要素+ComputePriceBand 统计+文件/AI/视觉导入链+OCR 询价飞轮），「手动导入信息价」路径无需开发；**外部自动接入四路评估**=公开信息价抓取不建议（脆弱+合规灰）、商业行情 API 不做（询比价已拍板不做其上游）、LLM 在线查价不作基线——**建议候选4 结案**。**真机验证抓到实锤当场修**：信息价发布表样本 CSV 走 `GaeaCostImportPreview`（预览零落库）——「除税价（元）」（信息价标准列名）不在 fieldPrice 字典→价格列 unmapped、12 行全 skip「缺少有效单价」；当场修 costimport.go fieldPrice 增「除税价/含税价」+Go 回归测试 TestParseCSV_InfoPriceChushuiColumn；复测 **12/12 行全部识别**。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.275.0。**产物**=exe 见 SHA256SUMS-v4.275.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.274.0 · UX 线第六刀：自定义强调色亮态自动深化（对比度保障）（2026-09-13）
> 「继续」续 UX 线，处理对比度普查观察池第一项。根因挖到底：**不是主题令牌缺陷**（6 主题 lightFn glow 全是深化值），而是**用户自定义强调色不分明暗覆盖 glow/colorPrimary**——暗色调亮的 #1dd7bf 切亮色主题后压浅底对比仅 ~1.5，全站 accent 文字（空间 chip/active tab/运行中徽标/链接）看不清。**修复**=①新增纯函数 `ensureLightContrast`（lib/accent.ts）：对白底 WCAG <4.5 时保色相饱和度迭代压 HSL 亮度至达标（下限 0.12、12 步、灰兜底；达标/非法原样返回）②App.tsx effTokens 亮态用深化值暗态原样——用户存储色与暗态观感均不变。**测试**=vitest accent 7 例（含真实案例）+App.tokens 2 例绿/tsc 0/eslint 0（no-raw-hex hex-exempt 行尾豁免）。**真机复验**=重建 exe 复跑 13 页×两态：light 告警 63→53，accent 类清零（亮态 glow 实测 #128475=深化输出）；剩余逐类甄别仍为禁用态/图片背景误报/占位符/设计弱化；dark 15 不变。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.274.0。**产物**=exe 见 SHA256SUMS-v4.274.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.273.0 · UX 线第五刀：键盘焦点环全站恢复（可访问性修复）（2026-09-13）
> 「继续」续 UX 线。普查 reduced-motion 发现基建已齐（系统级+应用内 ui-reduced-motion 双全局兜底）不动；CDP `Input.dispatchKeyEvent` 实测 Tab 导航抓到大问题：**v4.255 立的全局 :focus-visible 环全站性失效**（home 7 站/cost 5 站/sin 6 站无环）——元素 `matches(':focus-visible')===true` 但 computed `outline: 3px none`，枚举 styleSheets 实锤打包产物 tailwind-*.css 里有一条源码不存在的 `:focus-visible{outline:none}`（Tailwind v4.3.3 编译链路注入），同特异性层叠按文档序吃掉普通规则——**焦点环自 v4.255 起实际只对少数自绘环元素可见**。**修复（index.css 单文件）**=①全局环 `!important` 必胜（forced-colors 模式同款标准做法）②antd 输入类豁免段同步 !important 保持豁免（自管 border/shadow 焦点态防双环）③**ant-btn 移出豁免**：Button text/default 型聚焦零视觉变化，豁免=键盘用户找不到焦点，移出吃全局环。**真机复验**=重建 exe 复跑六页 Tab 遍历：home/cost/memoryhub/sin 零无环，chat/settings 各剩 1 个豁免 ant-input（probe 采样早于 antd shadow transition，非真无指示）；sin 页 antd 按钮聚焦实拍 2px glow 环贴合圆角。**观察池**=动效时长令牌化不做（各页节奏属设计自由度）/焦点陷阱未测（antd 自带 trap）。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.273.0。**产物**=exe 49,292,288B SHA256=45F4B7BF…BD1F（桌面副本同哈希，冒烟 200 过）。

## v4.272.0 · UX 线第四刀：窄窗响应式收口（遥测条 + Composer 工具行）（2026-09-13）
> 「继续优化迭代前端 UI」续刀。换镜头做窄窗：CDP Emulation 900×600 复跑 13 rail 页——布局自适应面整体健康（办公右栏窄窗自动收起、面板 min-w-0/truncate 到位），抓到两处真硬伤并修：①**底部遥测条右缘截断**（全站六页现）：引擎 pod（nowrap 超长模型名）把 CPU/内存/GPU 数值挤出视口——pod 加 max-width 240 + flex-shrink(文本包 .v3-pod-text 出省略号，inline-flex 上 text-overflow 不生效)，tele-key/value 加 flex-shrink:0——挤压优先级 pod→工程名→遥测数值完整保；②**sin composer 工具行按钮竖排折字**（右栏打开时容器约 320px，按钮无 nowrap 被 flex 压成竖条）：ComposerToolbar 外层 flex-wrap + 权限/思考按钮组 shrink-0 + 按钮 whitespace-nowrap（办公共享组件，两处受益）。**测试**=tsc 0/eslint 0/composer 26 用例+layouts 5 用例绿；**真机复验**=重建 exe 复跑 13 页窄窗全 ok（修复前六页 off:v3-tele-value），目检 pod 省略号+遥测完整、工具行两行横排无竖字。**观察池**=办公右栏中窄窗任务面板横滚兜底（内部已 truncate 非破版）/weixin 详情留白沿用待拍板。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.272.0。**产物**=exe 见 SHA256SUMS-v4.272.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.271.0 · UX 线第三刀：滚动条全站单一真源（视觉降噪）（2026-09-13）
> 接 v4.255/v4.260 的「优化 UX、美化前端 UI」线。普查先行：CDP 起壳 26 页次（13 rail 页×明暗两态，code=独立窗口跳过）巡检——横向溢出零/越界仅首页极光装饰球/console 零错误，结构层无硬伤；问题在视觉层：**四套滚动条定义并存**，v1.3.0 霓虹遗产「滚动条霓虹化」（thumb=glow 渐变+无效 opacity）盖过中性定义，chat/modelcenter/imagegen 亮青粗条喧宾夺主，novel/sin 却是 `scrollbar-width:thin` 原生细条——同屏两态。**本刀（纯 CSS，零 TS/零 Go/零绑定）**=①全局单一真源（index.css）：8px 中性胶囊（thumb=on-surface 18% 半透明+2px 内缩+胶囊圆角，hover 30%，取 module-launcher 最得体形态升全局）②删霓虹化覆写+顺手清死规则：v4.255 拍板的 ::selection（color-mix）一直被文件尾 v1.3.0 glow 版覆盖从未生效，两处合一保留生效形态零视觉变化 ③删三处重复定义（gaea/styles.css 全局 10px、redesign.css .gaea-app-layout 8px、module-launcher.css .ml 10px）④保留功能性隐藏与 scrollbar-thin/hidden utility（无消费方）。**真机复验**=重建 exe 同脚本复跑 26 页次两态：亮青粗条全部变中性胶囊，novel/sin 细条区零变化，目检零回归零错误。**观察池**=weixin 通道详情主区留白（填充需后端消息历史，待拍板）/办公右栏窄窗子代理卡横向滚动兜底/novel 技能下拉裸 id（全站惯例）。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.271.0。**产物**=exe 见 SHA256SUMS-v4.271.0.txt（桌面副本同哈希，冒烟 200 过）。

## v4.270.0 · 原罪工具集 v1.2：sin_illustrate 工具入册（拍板放行）（2026-09-13）
> 用户在拍板池清单后连说「继续」=按序放行第一项，推翻设计档 §13.2「刻意不做」；入册前提（产物链）本刀补齐。**单链路委托既有 SinIllustrate**（角色锚点+参考槽门控+退回纯文本+台账+sin/art 目录），不复制出图链。**本刀**=①工具集 6→7（order：outline 后、export 前）②`sin_illustrate`：prompt 必填/caption/size 可选，ReadOnly=false，Description 明确与标记协议分工+提醒「不必再为它写插图标记」防同图两生成 ③**Artifacts 产物链补齐（v4.262 留存缺口）**：sinToolTrace 增 artifacts 字段、工具实现 sinToolArtifactProvider、循环 Execute 成功后收集并随 result 帧与轨迹落库 ④**落库回写**：messageID 拿到后 sinPersistToolArtifacts 按 tool0..toolN 并入 extra.illustrations（cue 前缀防与正文标记数字键互踩，失败逐条告警不阻断）⑤前端过程卡缩略图（附件通道转 data URL，caption 图注，读取失败如实占位）+元数据「生成插图」。**测试**=Go+4（真出图+Artifacts+原罪目录落盘/坏参数报错不留产物/order 位次/回写 tool0+messageID<=0 跳过）+计数锁 6→7+wantReadOnly+vitest +3（缩略图渲染/读取失败占位/标签断言）；零新 App 绑定（drift 642）。**门禁**=ci.ps1 全绿/drift OK@642/版本三处 4.270.0。**走查=部分完成如实记录**：工具集在位+缩略图渲染由 Go/组件测试覆盖；live 模型调工具端到端两次尝试未走到（第 1 次空间持久化致 shelf 定位失败脚本已修；第 2 次模型回合未落消息且遇非脚本发起的壳启动，按「用户在用不动真机」红线停止），**留池补验**（.tmp/walk-v4270-illustrate.mjs 已含空间切换修正）。**产物**=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）。

## v4.269.0 · 造价域线上断链修复：costproject/coststage json 标签 + 五算带入空串时间字段（2026-09-13）
> 维持轨「五算带入与含量对照端到端」走查清账时抓到的**两个线上真 bug**。**Bug1**=costproject/coststage 全部结构体无 json 标签：线上序列化输出 PascalCase（Name/ItemCount/Total/VersionCount），前端契约读 camelCase → 测算项目列表卡空名/空行数/¥NaN、五算面板阶段值/对比/偏差同断（**自 v4.204 起纯展示/读取面断裂**；保存方向 Go 反序列化大小写不敏所以一直正常，录入路径未暴露；组件测试 mock camelCase 拦不住形状错配）。**修复**=两包 7 结构体全量补 camelCase json 标签（Unmarshal 大小写不敏 ⇒ 存量数据天然兼容）。**Bug2**=五算带入/阶段手动保存/询价手动录入 payload 显式带 `createdAt:""`/`updatedAt:""`——Wails 参数解析空串进 time.Time 直接报 `parsing time "" as "2006-01-02T15:04:05Z07:00"`（toast 报错永不落库，**自 v4.194/4.2 起带入按钮线上必炸**）。**修复**=载荷省略时间字段（缺省=零值）+前端类型 CostStageValue/CostInquiryRecord 时间字段改可选；同型排查全仓仅此三处（mock 不算）。**测试**=Go+2（TestWireShapeCamelCase ×2 线上形状回归锁：camelCase 键全在位+无 PascalCase 键+旧 PascalCase 数据兼容断言）；tsc 0/eslint 0/vitest 342 文件 2977 例全绿；零新绑定。**真机走查**=一次性造价项目端到端：建项目→3 明细→版本快照（合计 100,200）→UI 列表卡正确显示名称/行数/v1/合计（Bug1 修复实证）→五算面板估算行带入→版本选择器列快照→确认→**桥验 `{stage:估算, amount:100200, note:"带入:走查-五算e2e v1"}` 落库**（Bug2 修复实证）→对比/偏差出数→compose 含量对照（无相似样本诚实回退 band null/checks 空；非空分支 v4.158/v4.209 单测覆盖）→级联删除清场。**真机走查**=一次性造价项目端到端（级联删除清场）：列表卡正确渲染名称/行数/v1/合计（Bug1 实证）→五算面板估算行带入→版本选择器→确认→桥验 `{stage:估算, amount:100200, note:"带入:走查-五算e2e v1"}` 落库（Bug2 实证）→对比/偏差出数→compose 无相似样本诚实回退（band null/checks 空；非空分支单测覆盖）；零前端 JS 错误。**门禁**=ci.ps1 全绿（run3；run1 ContextView 竞争 flaky、run2 rename 文件锁 flaky，隔离复跑绿）/drift OK@642/版本三处 4.269.0。**产物**=exe 49,281,024B SHA256=88864D3C…C2A3（桌面副本同哈希，冒烟 200 过）。

## v4.268.0 · knip unused exports 甄别清账（瘦身 P5 既定池项）（2026-09-13）
> next-stage-plan §100「knip unused exports 169 甄别」清账：实跑 knip 现报 11 unused exports+6 unused types+8 duplicates（169 为陈旧数字，期间拆解已消化大半）。**甄别原则**=全死删（MODEL_PRICES/SPINNER_WORDS/getLocale/nodeStageLabel/CharacterFormHelpers Block·Field·FieldProps/engines 三接口/bridge.ts 五个死 re-export——本体在 proxy/office/events/drift 各自文件里健在）、双出口裁未用侧（SecurityBanner/DataPanel/SecurityPanel/AnsweredByLine 具名侧、CostGraphView default、WhisperMemoryList MemoryFact 再出口）、**契约哨兵保留 @public**（drift.ts _Check 两方向锁+spaceBindings _No* 三锁+IndependentBindingName+wails.d.ts NovelReadingAskTurn——AssertNever 零运行时成本）、误报挂 @public 注明（exportArtifact downloadBlob=BaselinesPanel.test vi.mock spy 委托，knip 解析不到）；knip.json ignore 补 bindingNames.ts（drift PS1 文本消费）与 bridge/drift.ts（契约档）。接受的重复出口 3 处（两侧均真实使用）。**测试**=tsc 0/eslint 0 error/knip unused 清零/vitest 342 文件 2977 例全绿；零行为变化零 Go 改动。**真机走查**=CDP 四页导航扫描（sin/chat/schedule/home）渲染全过零前端错误。**门禁**=ci.ps1 全绿（run2；run1 rename 文件锁 flaky 隔离复跑绿）/drift OK@642/版本三处 4.268.0。**产物**=exe 49,281,024B SHA256=10A2566A…F15D5（桌面副本同哈希，冒烟 200 过）。

## v4.267.0 · 原罪工具集 v1.1：sin_export 图文导出工具入册（2026-09-13）
> v4.262 留存合约清账（设计档 13.2 注记「下刀接完产物链路再入册」）：`sin_export` 的产物链路（exports 目录落盘 + 轨迹文本路径，过程卡渲染）已被 v4.262 过程卡补齐，前置满足。`sin_illustrate` 维持「刻意不做待拍板」不入册。**本刀**=留存件逐字入册+接线：①工具集 5→6（`sinToolOrder` 末位=收尾动作），`sin_export` 委托既有 `SinExportMarkdown`（与前端「另存为」同一份 Markdown，零第二份导出逻辑）②产物落 `<用户配置目录>/gaea/sin/exports/`（与 art/notes 同数据根，硬隔离不变），**同名不覆盖**（-2…-99），文件名净化 fail-closed（路径分隔/Windows 保留字符/控制符剔除、上跳不留、.md 去重、60 rune 截断），原子写（tmp+rename）③`ReadOnly=false`（新建文件）+schema 仅可选 `filename`+Description 明确「只在用户明确要导出时用」④前端过程卡：标签「图文导出」+ Download 图标 + 行首摘要给 filename。**测试**=Go+3（导出落盘+轨迹给路径+同名 -2 不覆盖+空故事诚实报错/文件名净化矩阵/工具集末位）+wantReadOnly 与 loop 计数锁 5→6+vitest +1（图文导出标签）。**真机走查**=临时故事两回合端到端（结束即删）：回合一只写正文、回合二导出→模型真调 sin_export→轨迹给路径+文件存在且含正文+过程卡「图文导出」行 ✓；**走查抓到并当场修掉语义缺口**=同回合「写完就导」时本回合正文尚未落库只导出头部，已把边界写进工具 Description（刚写的内容下一回合再导）并复验通过；onerror/unhandledrejection=0。**门禁**=ci.ps1 全绿（run1+run2 覆盖 Description 修改）/drift OK@642/版本三处 4.267.0。**产物**=exe 49,281,024B SHA256=BD2AEE86…6D827（桌面副本同哈希，冒烟 200 过）。

## v4.266.0 · 原罪底稿面板内编辑：大纲/设定集（右栏从只读到可写）（2026-09-13）
> v4.263.0 遗留下刀候选清账（「需冲突合并设计」）。此前设定集/大纲只读，写路径只在工具侧。**冲突合并设计（§14.7）=回合互斥+锁内基线比对确认，不做字段级 merge**：①AI 只在回合内写底稿，面板编辑在 sending 中锁定（入口+保存双禁用）→常规路径结构上无并发；②残余窗口（多开壳/他端）由 `SinNotesSave` 在 `sinNotesMu` 锁内做**基线比对**——入参带用户开始编辑时的 doc 快照 JSON，与当前文件不符即拒（固定前缀「底稿冲突：」），前端弹「覆盖确认」后 `force=true` 重试，静默覆盖不可能发生；③限长与工具侧完全同口径（单条 2000 rune/大纲 4000 rune 截断、条数 200 上限拒绝）。**本刀**=新绑定 `SinNotesSave(topicID, baseline, outline, notes, force)`（641→642，play）；大纲页签「编辑/写大纲」→textarea（rune 计数 x/4000）；设定页签「编辑/记便签」→逐条 textarea+删除+添加一条；空底稿也给入口（用户可先于 AI 手动起草）；`useSinNotes.save` 按冲突前缀映射 `conflict=true`。**测试**=Go+1（TestSinNotesSaveBinding：基线一致落盘/过期基线冲突+前缀/force 覆盖/截断/条数上限/坏 JSON/守卫）+vitest+8（大纲保存 payload 带基线/空大纲入口+取消/设定加删改整包/sending 双禁/冲突弹窗走 force/普通失败就地显示/save 映射×2）。**真机走查**=临时故事端到端（不打模型，结束即删）：写大纲/记便签保存桥验落盘 ✓，他端 force 直改后 UI 过期基线保存→「底稿已被更新」弹窗→覆盖→UI 内容胜出 ✓；onerror/unhandledrejection=0。**门禁**=ci.ps1 全绿（run4；run1/run3 负载 flaky 隔离复跑绿，run2 被 E27 拦=gen_bindings 还原 office/cost 变参，checkout 还原后过）/drift OK@642/版本三处 4.266.0。**产物**=exe 49,269,248B SHA256=6D40A0D9…CC3C（桌面副本同哈希，冒烟 200 过）。

## v4.265.0 · 原罪画廊重新生成入口（右栏画廊对齐流内插图能力）（2026-09-13）
> v4.263.0 遗留下刀候选清账；此前只有流内插图卡有「重新生成」，画廊只能看。**零 Go 改动零新绑定**——`SinIllustrate`（641 面既有）本就是全链路：角色锚点+参考槽门控+退回纯文本重试+台账登记+`arts[cue]=path` 覆盖回写，重生成=同一入口再调一次。**本刀**=①`collectIllustrations` 条目扩 `messageId/cue`（回写定位：SinIllustrate 按 messageId+cue 覆盖写）②画廊缩略图**悬浮「重新生成」钮**（默认藏，hover/focus 显形；生成中变「重新生成中…」状态条+缩略图调暗挡重复点击；正文无标记→prompt 空不出钮，SinIllustrate 拒空提示词；回合 sending 中禁用防与在途插图互踩）③大图 Modal footer 同款钮，重生成成功后预览自动跟到新条目④页面回调与流内**同一 `illustrationQueue` 串行**（一次一张）→ SinIllustrate → `reloadMessages()`（useSinStory 新暴露）刷新画廊与流内⑤**外部换图采纳**：SinIllustration 记 `extPathRef`，pathProp 变化且自身不在生成中才采纳——防「画廊新图/流内旧图」不同步，也防把自己在途产物顶掉。失败如实 notice（原图保留：回写只在成功路径发生）；`persisted:false`（图已生成但回写失败）也如实提示不静默。**测试**=vitest +8（画廊钮回调带定位/busy 挡重复+sending 禁用+空 prompt 不出钮/Modal footer+预览跟新/collect 字段/外部换图采纳+生成中不采纳/reloadMessages）。**真机走查**=临时故事端到端（结束即删，用户故事未动）：建走查故事→Composer 打一轮出正文+插图标记→流内自动出图落库→画廊点「重新生成」→busy 态出现→后端回写换新路径+画廊缩略图/流内大图同步换图三项断言全过；window.onerror/unhandledrejection=0。**门禁**=ci.ps1 全绿（vitest 342 文件 2969 例）/drift OK@641/版本三处 4.265.0。**产物**=exe 49,257,984B SHA256=868668BD…65AB（桌面副本同哈希，冒烟 200 过）。

## v4.264.0 · 原罪右栏面板宽度可拖拽（对标办公右栏拖宽交互）（2026-09-12）
> 用户口径「右侧面板宽度应该是可以自由拉伸的」（v4.263.0 遗留的下刀候选）。**本刀**=①**左缘拖拽手柄**（`.sin-side-resizer`，骑边框 8px 命中带，悬停/拖拽中 accent 显色）：指针拖拽范式与办公 `useWorkspaceLayout.startWorkspaceResize` 同源——`window` 级 pointermove 实时跟手（向左拖=变宽）、拖拽中 `body` 锁 `col-resize`+禁选中、pointerup 持久化、pointercancel 兜底收尾；**②宽度记忆** `gaea.sin.panelWidth`（原罪自有键名续用）；**③钳制**=240~640 硬档位之上按视口收敛（`innerWidth-520`，保住左故事架+故事流可读），`clampSinPanelWidth` 纯函数；非法/越界值回默认 268；**④双击手柄复位**默认宽度；**⑤画廊列数随宽自适应**：`repeat(auto-fill, minmax(104px,1fr))`（默认 268px 仍两列，拖宽自然到三/四列——拖宽有实际收益而不是留白）。**测试**=vitest +5（拖拽跟手+视口钳制+松手持久化 / 向右收窄到下限 / 双击复位 / 宽度记忆重挂载恢复 / clamp 纯函数矩阵）；零 Go 改动零绑定。**真机走查**=CDP 受信任鼠标拖拽手柄 268→448px 实时跟手（拖拽中途采样 358 证跟手）、`gaea.sin.panelWidth` 落 448、画廊列数 2→3、双击复位 268/268，两轮 window.onerror/unhandledrejection=0（首轮走查脚本 CDP 双击模拟缺 clickCount:2 误报复位失败，修正模拟后通过=应用无缺陷）。**门禁**=ci.ps1 全绿（代码树 run3+终态树 run4）/版本三处 4.264.0/drift OK@641（零新绑定）。**产物**=exe 49,255,424B SHA256=4A1E96E0…0102F（桌面副本同哈希，冒烟 200 过）。

## v4.263.0 · 原罪右栏创作面板：角色 / 大纲 / 设定 / 插图（对标办公标签页面板）（2026-09-12）
> 用户口径「继续优化原罪板块，增加类似办公的右侧面板，可以看角色、大纲、插图、设定」。**改前**=原罪右栏是静态说明栏（角色卡 + 玩法/插图协议/边界三张固定卡），且 AI 在回合里用 `sin_notes`/`sin_outline` 写下的工作底稿（设定集、大纲）前端**没有读取通道**——工具写进 `%APPDATA%\gaea\sin\notes\<故事id>.json` 后用户看不见。**本刀**=①**新绑定 `SinNotesGet(topicID)`（640→641，play 空间）**：只读同源便签文件返回 `{notes, outline}`（`sinTopicGuard` 守卫 + `sinNotesMu` 与工具侧同一把锁；缺失/损坏=空清单空大纲，与工具侧 loadSinNotes 同口径）。**②右栏改办公同款标签页面板 `SinSidePanel`**：四页签——**角色**（既有 SinCastPanel 原样入页签）、**大纲**（sin_outline 记的章节走向，pre-wrap 只读）、**设定**（sin_notes 设定集逐条卡片带 #序号）、**插图**（本故事全部产物画廊：extra.illustrations 收集 + 正文标记反解 caption + 缩略图点开 Modal 大图；本地路径经 AttachmentDataURL 通道转 data URL，与流内插图同口径）；页签带计数徽标（纯文字页签——真机走查实证 268px 下「图标+两字」会折行成竖排，去图标+nowrap 修复）。**③玩法/插图协议/边界三段静态说明收进面板头部问号气泡**（内容逐字保留，版面降噪——遵守「简化≠删功能」红线）。**④面板开合**：顶栏 PanelRight 开合钮（记忆键 `gaea.sin.panelOpen` 原罪自有命名，不借办公 workbench 空间键；宽窗默认开窄窗默认收，替代原 1180px 强制隐藏媒体查询）；页签记忆 `gaea.sin.panelTab`。**⑤底稿实时性**：每回合结束（sending true→false）自动重读便签/大纲（AI 可能在回合里写了），切故事即重读。**测试**=Go +1（SinNotesGet 绑定：空文档/工具写→读回同源/未知故事 id 拒绝）+ vitest +14（SinSidePanel 8：默认页签与计数徽标/大纲渲染/便签序号/画廊与 Modal 预览/三空态/错误透出/页签持久化/玩法气泡入口；collectIllustrations 2：跨消息收集与 prompt 反解/空路径跳过；useSinNotes 4：读取与坏行过滤/空 id 不读/错误透出/reload 重读）。**真机走查**=用户真实故事四页签全通：大纲/设定空态轨道环（无底稿不编造）、插图 3 张缩略图全转 data URL + Modal 大图预览 + 开合与页签记忆；走查抓到并当场修掉页签折行缺陷。**门禁**=ci.ps1 全绿 ×2（计数锁修复后 + 页签修复后终态树）/drift OK@641/spaceBindings 分类锁 462→463/版本三处 4.263.0；mock 同步 SinNotesGet（便签+大纲种子，浏览器走查可看）。**产物**=exe 49,253,888B SHA256=6E281D6A…0A880（桌面副本同源，冒烟 200 过）。

## v4.262.0 · 原罪工具集 v1 + 过程卡：联网搜索 / 网页抓取 / 角色卡 / 故事便签 / 故事大纲（2026-09-12）
> 用户报「原罪现在可以使用哪些工具？现在看不见调用工具的过程卡」，并拍板「A + 网络搜索工具」。**改前**=原罪文本链**模型零工具**（`SinStream` → `ai.ChatSimpleOptions` + `ChatStreamChunks`，请求体没有 tools 字段；前端 StoryStream 不处理 `tool_dispatch`/`tool_result`），因此过程卡不是被折叠，而是根本没有可显示的工具调用。**本刀**=①**工具集 v1（原罪域内，硬隔离不变）**：`web_search`/`web_fetch`（**委托办公 builtin**——schema 直接转发零漂移，[search] 引擎扇出、SSRF/域名策略、代理同源）+ `sin_cast`（只读角色卡，不改设定/改名）+ `sin_notes`（故事便签 list/read/write/set/delete，≤200 条 × 单条 2000 rune）+ `sin_outline`（大纲 read/write，≤4000 rune）；新注册表 `sin_tools.go`（init 自注册/重复即 panic/顺序固定，形态与 image_backend、search engine 同源）。②**有界工具循环**：`runSinStream` 升级为多轮（上限 4，最后一轮不带 tools 强制收尾成正文）——每轮经新 ai 入口 `ChatStreamMessages`（与 `ChatStreamChunks` 共用 `prepareStreamRequest` 一处装配，两入口零漂移）→ delta/reasoning 照旧实时下发 → `Done.ToolCalls` 非空则执行工具并追加 `assistant(tool_calls)` + `tool` 消息继续；**首轮带 tools 直接失败 ⇒ 去掉 tools 重试一次 + notice「当前模型不支持工具调用，已按纯写作继续」**（工具是增强不是前置条件）；整轮可取消（SinCancel 起手即登记，覆盖流式与工具执行）。③**过程可见**=新事件 `tool_dispatch`/`tool_result`/`notice`；`done.tools` 与消息 `extra.tools` **同一形态**（重开故事按 extra 还原）；前端新增 `SinProcessCard`（「思考过程 · N 字」折叠行 + 工具行：中文标签/目标摘要/状态/用时，展开看参数与输出）+ `useSinStory` 过程帧合并（result 丢帧兜底补行、done 以后端轨迹为权威）+ StoryStream 在正文之上渲染（空则不渲染，零噪声）。④**数据面**=便签/大纲落 `<用户配置目录>/gaea/sin/notes/<故事id>.json`（原子写、故事 id 白名单 fail-closed、损坏当空容错），不写办公工作区、不进办公记忆。⑤**失败纪律**=未知工具如实回错并列可用清单、工具失败只记轨迹不炸整轮、工具输出 6000 rune 截断并留可见标记、全程无正文不落空消息。**未做**=`sin_illustrate`/`sin_export`——并行执行体越过契约实现了这两件（方向不坏：委托既有 `SinIllustrate`/`SinExportMarkdown`），但**产物回写链路**（工具产物 → 过程卡缩略图/重开还原）两个执行体各以为对方会补、实际没人补，属半成品，已挪出树外存 `.tmp/sin-next-knife/`（不删），下刀接完再入册。**过程教训**=并发子代理时「契约文件」必须显式列为禁改 + 产物类工具的跨线依赖要在契约里点名责任人（本刀两处都踩了）。**测试**=Go +12（注册表接线/联网委托逐字节对齐 builtin/角色卡文本矩阵/线格式帧逐键锁定/输出截断/便签与大纲往返+上限+路径守卫+损坏自愈/工具循环四例：多轮执行与 extra 落库、端点不支持 tools 的降级重试、轮次封顶+未知工具回错、超长截断）+ vitest +12（过程卡 8：空态/折叠展开/运行-完成-失败态/参数与输出折叠；轨迹解析 3：snake_case 解析/畸形跳过/失败态推导；事件流 1：dispatch→result 合并 + 丢帧兜底 + notice + 历史还原）。**门禁**=ci.ps1 全绿/版本三处 4.262.0/drift OK@640（零新绑定）。

> **真机走查补记（v4.262.0 exe，2026-09-12）**——联网探针=直调办公 builtin `web_search` 0.6s 返回 Bing 结果（1084 字）；应用内一轮「先查证再写」抓到并当场修掉三个单测覆盖不到的真问题：①**搜索狂欢**=模型一轮连搜 12 次、轮次封顶后正文只剩 62 字 → 加**调用预算**（同名 ≤3/整轮 ≤8，超限不执行、如实回绝并写进轨迹）；②**收尾轮空转**=收尾轮（不带 tools）模型仍回工具调用 → 该轮没有正文、整单报错且什么都没落库 → 加「工具阶段结束」收束令 + 一次兜底收尾轮；③**前端误判超时**=静默超时是一次性 90s 计时器（不随帧重置），工具循环长回合必踩（后端已落库 827 字、前端报超时）→ 改每帧重置的真沉默计时。**最终基线**=新建走查故事一轮「先查证再写」→ UI 正文 1185 字（落库 1340 字）+ 9 条工具轨迹（便签/大纲/角色卡 + 联网搜索×3 + 预算回绝×1 + 抓取网页×2，各带用时），零渲染错误/exceptionThrown=0（走查故事已删）。**测试最终**=Go +14 / vitest +13。**产物**=exe 49,242,624B SHA256=470F86EF0C8DD63C643496D44200FB01E857C693A0849909EDB45375C9DC6E68（桌面副本同哈希，冒烟 200 过）。

## v4.261.1 · 修复：原罪插图报「marshal image request: 图生图需要提供参考图」（2026-09-12）
> 用户报「原罪图片生成失败：marshal image request: 图生图需要提供参考图」。**根因**=v4.259.0 参考槽门控漏了「后端支持、但这一轮拿不到参考图」这一支：`sinRefPlan` 只判「后端×模型能不能用参考槽」，一旦可用（herdsman 恒可用 / comfyui+krea2·z-image-turbo）就把 `mode=img2img` 定死，而参考图解析结果可能为空——①故事没选角色（`picked` 空）②角色只有远端剧照（xAI 临时图，`sinRefDataURL` 如实跳过）③本地文件缺失。此时 `refImages=0` 却仍以 `mode=img2img` 发请求，Herdsman 图生图端点整单拒绝（`ai/image_openai.go:buildImg2ImgBody` 报「图生图需要提供参考图」，被包成 `marshal image request: …`），而既有回退只在 `refImages>0` 时触发 → 用户看到的就是这句报错。**真机取证**=`%APPDATA%\gaea\logs\gaea-20260912.log` 20:26:37/20:26:59 两连发（启动行图片后端 Herdsman）；用户 `%APPDATA%\gaea\sin\cast.json` **不存在**=故事没选过角色，即分支①。**修复**=①解析与门控解耦——新纯函数 `sinResolveRefImages` 负责「参考图优先、其次剧照；远端 URL 与缺失文件跳过」，返回可用图/名与**台账归属 id**；②门控收口——解析结果为空即 `refOK=false` 并**清空 mode/refMethod**（没选角色不弹噪声原因；选了角色但无图给「该角色没有可用的参考图/立绘（角色库可生成剧照后重试）」），绝不把「这次用不上参考槽」变成整单失败；③顺带修**台账归属错位**——改前 `len(refChars)==1` 时记 `picked[0].ID`，多角色里只有第二人有可用参考图会把登记记到第一人头上，现按「真正带图」的角色回溯。**测试**=Go +3（红→绿取证：改前新用例 3 子例全红「mode=img2img refs=0」；新纯函数矩阵=data URL 原样/本地文件转 URL/远端与缺失跳过/归属取带图者；端到端 3 子例=没选角色 / 单角色只有远端剧照 / 多角色未点名，都必须纯文本且不报错，选角色时文本锚点照旧生效）。**门禁**=Go 全绿/tsc 0/eslint 0 error/vitest 339 文件 2929 例全绿/drift OK@640（零新绑定）/ci.ps1 全绿/版本三处 4.261.1；exe 49,164,288B SHA256=CCE99E91…D5CAB（桌面副本同哈希，冒烟 200 过）。**真机走查**=v4.261.1 exe（CDP 9333，图片后端 Herdsman）打开原罪故事 → 两个插图位 20s 内自动出图（新产物落 `%APPDATA%\gaea\sin\art\…png` 2,134,820B 并回写 extra.illustrations），日志零「图片生成失败」/零 SinIllustrateError、页面零渲染错误、exceptionThrown=0；故事 `cast.json` 不存在（没选角色）=与用户同一分支。

## v4.261.0 · UX 线第三刀：gaea EmptyState 收敛到 V3Empty（全站空态语言归一收官）（2026-09-12）
> v4.260.0 避开项清账（彼时 sin 线消费中暂缓；用户确认那边已结束，工作区干净核实后动手）。gaea/components/EmptyState.tsx（原 13px 40% 灰字纯文字形态）改 **V3Empty 薄壳**——轨道环视觉+on-surface-variant 文字，py-14 呼吸间距保留，**API {message} 不变**，消费方（聊天域 WhisperGraphPanel 两处空态 / 原罪 sin StoryStream 故事流空态）零改动。至此全站空态三套（V3Empty / modelcenter EmptyState 双形态 / gaea EmptyState）视觉语言归一：无图形场景一律轨道环，语义图标场景保留 icon。**测试**=消费面 27 用例绿+tsc 0+eslint 0；原罪页真机走查正常渲染（用户已有历史故事非空态场景，vitest 覆盖空态渲染）。**门禁**=ci.ps1 全绿/版本三处 4.261.0/exe 49,164,288B SHA256=38a912c6…03cac。

## v4.260.0 · UX 线第二刀：一致性收口（--radius 桥接 M3 + modelcenter 空态收敛 + --sched-dim 修复）（2026-09-12）
> 「优化 UX、美化前端 UI」第二刀。普查先行：24 页次（12 rail 页×明暗两态）CDP 起壳巡检**全零错误**、各孤岛 CSS 实测近乎零 hex（imagegen/modelcenter/settings/programming/weixin/character-library 均 0，novel 19 处为功能性设计意图不动）——明暗一致性基础好，无硬伤级问题；据此把普查里「三套圆角并存」实测收窄（定义 3 处、消费仅 1 处）后定形三小刀。①**--radius 桥接 M3**：gaea/styles.css 的 `--radius: 9px`（--ds 体系布局令牌，孤值）改 `var(--md-sys-radius-md, 12px)`；redesign.css `.gaea-app-layout` 内同值死覆写（12px）删除——消费点仅 .context-menu 一处（styles.css:1372），办公工作台内视觉零变化，孤值收敛单源。②**modelcenter 空态收敛到 V3Empty**：EmptyState（ui.tsx，API 不变）改双形态——**无 icon 时渲染 V3Empty compact 轨道环**（hint 走 children 弱色行），有 icon 保持语义图标形态；实测 13 调用点/9 文件：10 处带 icon 不变、3 处无 icon（BenchmarkSection 加载态/LLMSection·SpecialtySection 搜索空态）自动获得轨道环视觉；V3Empty 加 compact 档（48px 图形+小间距）。③**--sched-dim 未定义修复**：.sched-aoa-mode-btn 的 `var(--sched-dim, #64748b)` 变量从未定义恒走 fallback（v4.255 刀A 子代理发现），桥接 `var(--md-sys-color-on-surface-variant, #64748b)` 明暗自适配；注意 `.sched-dim` 是同名 CSS 类（73 行）与变量无关，其余 3 处命中为类名非残留。**避开项**：gaea/components/EmptyState.tsx 收敛暂缓——sin 线（v4.256~259 已 commit）正在消费，待其稳定另刀。**版本顺延说明**：本刀原按 v4.256.0 预备，同树另一执行体抢先一笔 commit 四刀（v4.256.0~v4.259.0 原罪板块），故顺延 v4.260.0。**测试**=modelcenter 107 用例绿+tsc 0+eslint 0+ci.ps1 全绿；24 页次两态巡检零错误+cost/modelcenter 截图目检无回归。**门禁**=版本三处 4.260.0。

## v4.259.0 · 原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）（2026-09-12）
> 上刀（v4.258）把生成进度做完后，原罪仍缺一环：**角色选进来了，插图里的人不一定是她**。本刀接上绘梦既有 T2 角色参考槽，让「角色库 → 故事角色 → 插图人物」闭合。**①文本锚点（所有后端受益）**=`sinAugmentPromptWithCast`：画面描述未写到角色外观时补「人物锚点（名字）：外观；身材」（≤120 rune），已写过不重复（前 20 字命中即跳过）；无外观设定则不追空壳。**②图像参考槽（仅 ComfyUI/Herdmans 图生图）**=取角色参考图（referenceImages 优先，其次剧照）走 `mode=img2img + refImages + refMethod=img2img`（denoise 0.65），与绘梦 T2 同链路（复用 `generateImageInternal` 参数化入口，不新增第二条生成实现）；参考图数据形态=data URL 原样 / 本地路径读文件转 data URL（comfyui uploadImage 与 herdsman img2img 都只认 data URL）/ 远端 URL 与缺失文件跳过并记 warn。**③锚定谁（防互串）**=`sinPickRefCharacters`：提示词点名优先；都没点名时仅当故事只选了一个角色才锚定；多角色未点名 → 不锚定。**④能力门控**=`sinRefPlan` 纯函数矩阵——comfyui+krea2*/z-image-turbo ✅（与 image_comfyui.go 图生图工作流支持面一致）、comfyui+其他模型 ⏭ 给原因、herdsman ✅、xai/glm 等 ⏭ 给原因；**不硬塞**（硬塞会整单报错）。**⑤回退**=参考槽失败自动退回纯文本重试一次 + 标 `ref_fallback`（插图不因参考图而失败）；返回 `ref_used/ref_characters/ref_reason/ref_fallback/anchor_added/prompt_used`。**⑥可见**=caption 徽标：accent 色「角色参考：<名字>」、灰「已补外观锚点」、「参考图不可用 · 已按纯文本生成」；无角色可锚时不弹噪声徽标（门控：仅在「选了角色但用不上」或回退时提示）。**未做**=ipadapter/pulid（ai/image_comfyui.go 未实现，`comfyResolveRefMode` 如实报错——沿用已实现的那条路，不假装支持）。**测试**=Go +7（能力矩阵/点名与单角色兜底/文本锚点去重与空值/data URL 转换与远端跳过/参考槽透传/不支持后端跳过且文本锚点仍在/失败回退重试）+vitest +2（参考徽标、回退文案）。**门禁**=Go 全绿/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@640（零新绑定）/E 系列+仓库卫生守卫绿/版本三处 4.259.0；浏览器走查=故事带「林晚」→ 出图 caption 显示「角色参考：林晚」「已补外观锚点」（零渲染错误/零异常）。**文档**=设计档 §12 + releases/v4.259.0.md。
## v4.258.0 · 原罪插图：生成进度动画 + 串行队列 + 真停止（2026-09-12）
> 用户提「增加图片生成动画知道完成进度情况」。**改前**=插图只有一个转圈 + 「正在生成插图…」：没有百分比、没有阶段，也看不出「这张在跑还是排队」，且多张并发时进度无法归属。**①进度抽核共用**=新 `components/imagegen/GenerationProgress.tsx`（自绘梦 GenerationBar 抽出：确定态进度条 + 「全部: N%」+「当前节点: 中文阶段名」+ 已用时；类名 `.ig-progress*` 与阶段名表 `COMFY_NODE_LABELS` 单点维护 → 绘梦/原罪同实现防漂移；`imagegen.css` 的进度块与 keyframes 随之迁入自带 `generation-progress.css`）+ 新轮询 hook `useComfyTaskProgress`（只在生成中轮询、单次读失败保留上一帧、卸载即清定时器）；**诚实口径**=ComfyUI 有实时百分比走确定态，其余后端（xai/glm/herdsman…）后端不提供百分比 → 不定态光带 + 本地秒表，**不编造百分比**；补 reduced-motion 降级。**②串行队列**=新 `pages/sin/illustrationQueue.ts`：同板块一次只跑一张（图像后端单任务 + 进度单任务维度，并发会让两条进度条抢同一百分比）；排队中的插图显示「排队中 · 前面还有 N 张」，轮到自动切实时进度；失败不阻断队列。**③真停止**=新绑定 `SinCancel(topicID)`（**绑定 639→640**）：中止底层流式请求（ctx）但**已生成的部分照常落库**（extra.cancelled=true，用户消息不丢），无在途流时如实报错；插图卡「取消」接既有 `CancelImageGeneration`；`sinRuns` 进程内登记表按话题单飞。**④顺带修两个自伤缺陷**=（a）`useSinStory` 的 `deadRef` 跳过收尾 + React StrictMode 双挂载 → `sending` 永久 true、输入框被「请先完成或停止当前回合」锁死（v4.256 起潜伏，dev 与空间剪枝重挂必现）；（b）Composer「停止」是空实现（`onCancel: () => undefined`，点了没反应）——现在接到 `useSinStory.cancel`（后端取消 + 本地立即解封）。**测试**=Go +2（取消保留部分 + extra.cancelled + 无在途流报错/跨板块守卫）；vitest +7（队列串行与位次、进度轮询四态、插图卡确定态/不定态/已有产物不重跑、hook cancel 复位）。**门禁**=Go 全绿/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@640/E 系列+仓库卫生守卫绿/版本三处 4.258.0；浏览器走查=生成中两处进度条（「生成中…」/「排队中 · 即将开始」）+ 取消一张后另一张继续 + 完成后图片就位且停止按钮消失（零渲染错误/零异常）。**文档**=设计档 §11 + releases/v4.258.0.md。
## v4.257.1 · 修复：真机插图报「AttachmentDataURL is not a function」（2026-09-12）
> 用户报「插图失败：e(...).AttachmentDataURL is not a function」。**根因**=S2-3 兼容代理（`window.go.app.App`，frontend/src/gaea/lib/bridge/http.ts::ensureLegacyAppProxy）只按**字面名**在各板块门面里查方法，认不出 gaeaToGaea 的短名映射：旧调用点写 `AttachmentDataURL`，而 OfficeB 上的真名是 `GaeaAttachmentDataURL` → 查找落空返回 undefined → 调用即抛 not a function。**为什么只有一部分功能中招**=bridge 代理（`app.*`）会做映射，所以 PortraitImg/消息内联图正常；`api/image.ts` 的图像域 helper 走的是旧直连 `window.go.app.App`，其中 4 个 Gaea 前缀方法全中招——`AttachmentDataURL`（读图）/`SavePastedImage`（粘贴落盘）/`RecognizeImage`（识图）/`OCRText`（OCR），受影响的用户面包括原罪插图、章节配图历史缩略图、画室识图试用（历史实现里部分调用点 catch 成空，属静默失败）。**修复**=①兼容代理补映射回退（字面名优先 → gaeaToGaea 映射名；仍未命中才返回 undefined——诚实失败，不静默塞假实现）；②`api/image.readFileAsDataURL` 改走规范 bridge 路径（映射 + mock 兜底），仅当 bridge 未认领时退回旧直连形态；③HTTP 模式门面补齐表加 SinB（移动端/调试页同口径）。**测试**=vitest +3（新 legacyAppProxy.test.ts：真机形态假门面下「短名 → GaeaAttachmentDataURL 命中」/「字面名仍优先」/「未知方法 undefined」/「readFileAsDataURL 经映射命中 OfficeB」）。**门禁**=tsc 0/eslint 0 error/vite build 成功/vitest 全绿/drift OK@639/版本三处 4.257.1；浏览器 mock 走查：原罪插图仍正常出图（2 张、零渲染错误、零异常）。
## v4.257.0 · 原罪三刀：与办公硬隔离 + 模型中心独立卡片 + 角色库接入（2026-09-12）
> v4.256.0 交付原罪板块后用户追加三条口径，本刀逐条落地。**①硬隔离（数据/记忆/模型路由/工作区四面）**——口径=组件可复用（渲染层），数据面一律不通。**插图产物改落原罪自有目录**`<用户配置目录>/gaea/sin/art`：为此把绘梦生成链「产物归属」参数化（新 `generateFreeImageProvenanced(sourceBoard, saveDir)`，绘梦侧行为逐字节不变），原罪传自有 saveDir + `source_board=sin`——不写办公工作区 `.gaea/`、不依赖办公 ImageSaveDir 与小说项目目录；**顺带修一处真缺陷**：改前原罪插图会被登记两次（生成链内 source=imagegen + 调用方补 source=sin，画室同图两条），现在登记只在生成链内发生一次且来源正确（`recordImageHubGeneratedFor`）。**工位检索硬边界**：`wssearch.isNoiseRel` 改前只跳 `.gaea/play/exports`，play 分区其余子目录（画室台账/章节配图/原罪产物）漏进办公关键词检索——现改为**整个 `.gaea/play` 判为噪音**（S2.1 红线「工位搜索搜不到乐园产物」；fileindex 本就整目录跳过 `.gaea`）。**记忆面零接入**（不读不写办公 facts/hephaestus.db 与轻语 hermes.db，故事上下文=本次故事历史+已选角色）；**模型路由独立**（routeModel("sin")，办公换模型不影响原罪）；**角色选择自存**（sin/cast.json）。**②模型中心独立卡片**=FEATURES 增 `{key:'sin',label:'原罪'}` + FireOutlined 图标 + 卡片说明（后端 config 三键 func_sin_engine/model/enabled 上一刀已接线）；走查确认卡片在位（未绑定→跟随全局默认）。**③角色库接入**=右栏「角色」卡 + 多选弹层（复用角色库 PortraitImg 三态渲染）→ 按故事持久化 `SinCastGet/SinCastSet`（悬空 id 过滤/去重保序/单故事封顶 8，返回生效清单前端以此回填）→ `SinStream` 注入角色块（名字/性别/年龄/定位/外观/身材/性格/背景/动机/状态/口吻/最多 3 条说话样例，空值整行不写不编造，并声明「不得改设定或改名」）；只引用不修改角色库档案。**绑定 637→639**（gen_bindings 再生 + office/cost 门面按既有纪律还原，diff 仅 manifest/新文件）。**测试**=Go +8（cast 往返含悬空丢弃/非 sin 话题守卫/封顶/角色块与提示词注入/数据根隔离断言；wssearch 整分区噪音）＋vitest +5（useSinCast 五态）；走查=模型中心卡片在位 + 角色弹层三人列示 + 「带入故事（1）」→ chip 落地（零渲染错误/零异常）。**门禁**=Go 全绿/drift OK@639/tsc 0/eslint 0 error/vite build 成功/vitest 全绿/E 系列+仓库卫生守卫绿/版本三处 4.257.0。**文档**=设计档 §8 硬隔离清单 + §9 角色库接入 + releases/v4.257.0.md。
## v4.256.0 · 闲庭新板块「原罪」：对话式图文混杂故事创作（复用办公组件）（2026-09-12）
> 用户提「在闲庭增加一个单独的功能板块，复用办公的组件，主要用于与 AI 对话，可以生成图文混杂的 H 故事小说，板块取名原罪」。**形态**=闲庭（play）第七板块 `sin`：左「故事」架 + 中故事流/办公输入器 + 右玩法·协议·边界三卡，layout=full。**组件复用（用户口径的正面落地）**=输入器 = 办公 `Composer`（粘贴/附件/截图/表格转换/工作区菜单全继承）、正文渲染 = 办公 `Markdown`（与办公消息同一条渲染链：本地文件链接/代码/mermaid/genui）、`ToolbarButton`/`EmptyState`/i18n `LocaleProvider` + 办公三件套样式（styles/tailwind/redesign，与 GaeaPage 同口径）——**零新建聊天组件**。**后端（新域 sinB，绑定 628→637）**=①会话复用 `internal/chat` 统一 Store（话题 mode=sin）——与聊天板块**同表不同域**：`SinTopicsList` 只见 sin，`ChatTopicsList` 过滤 sin（聊天板块 persona 分支不认识该模式，放进去既污染列表也走错生成路径）；②文本 = `ai.Client.ChatStreamChunks` 既有流式通道 + 新功能级路由 `routeModel("sin")`（config 三键 `func_sin_engine/model/enabled`；模型中心新增「原罪」功能绑定 → 可给故事单挂无审查/长上下文模型，未绑定回退全局）；③插图 = 绘梦图像后端 `mediaState.GenerateFreeImage` 同一条生成链（play 护栏 image_safe_mode 照常生效），产物按 `space=play / source_board=sin` 登记图像域台账（画室可按来源筛选），路径回写该轮消息 `extra.illustrations`（**重开故事直接按映射渲染，不重新生成**；chat store 新增 `GetMessage`/`UpdateMessageExtra` 两个窄方法）；④`SinExportMarkdown` 导出图文 Markdown（标记 → 图片链接／未生成留占位）。**插图协议**=模型在需要配图处另起一行输出 `@@插图|画面描述@@`（可 0～2 张/轮），前端按出现次序编号（0 起）为 cue 键就地生成并嵌进正文流——正文原样落库（`正文 = 模型输出` 可回溯），导出时才做标记→图片替换。**成人内容口径零新立**=沿用 `docs/ADULT_MODE.md` 既有决策（个人非商用桌面应用、默认开启、不回避不说教不医学化）；系统提示词把硬边界写死：性相关角色一律成年人、禁止真实在世人物、合意基调、用户叫停即停（`sinSystemPrompt` 有专项回归测试锁这四条）。**前端接线六处**=bridge 分域 `SinBindings`（AppBindings 第 11 域）+ bindingNames（drift 637）+ spaceBindings 归 play（锁 450→459）+ dev mock（内存故事库 + `window.runtime.EventsEmit` 模拟真流式 + 占位插图路径，`?mock=1` 可走查图文版面）+ manifest（前后端各一处，backend `builtinManifests` 与普通 canonical 同列，前端 `canonicalBoards` menuOrder 13）+ PageRegistry/启动器文案 + 模型中心 FEATURES。**测试**=Go +7（话题域隔离与跨板块守卫/流式落库与消息 id/未知话题拒绝/插图回写与图文导出/前情装配去标记+封顶/提示词硬边界）+ config 绑定往返 1 + vitest +27（storyText 纯函数 10 / useSinStory 状态机 5 / 事件频道 2；boards·CommandRail·events 既有锁更新）。**门禁**=go build/vet/test 全绿 + drift OK@637 + tsc 0 + eslint 0 error + vite build 0 + vitest 334 文件全绿 + E 系列/仓库卫生守卫绿 + 版本三处 4.256.0。
## v4.255.0 · 前端 UX 精修一版三刀：进度计划主题桥接 + 全站空态统一 + 全局焦点环（2026-09-12）
> 用户提「优化 UX、美化前端 UI」，普查先行（四代令牌体系/15 页样式来源矩阵/UX 底座盘点，产出 docs 速览交本刀），一版落地三刀。①**进度计划域接入主题系统**（schedule/schedule.css 单文件）：本应用明暗切换无 CSS 类钩子（全靠 App.tsx 换 :root 变量值），纯 hex 消费者天然脱离主题——顶部四令牌（关键红/工作蓝/链接灰/分组灰）改**明暗两档**：`:root` 默认暗色提亮档 #f04848/#4d8df0/#8b98ab/#d5dae2，`:root[data-hl="light"]` 亮色锁原值 #e02020/#1e6fd9/#94a3b8/#1f2937（亮色零变化=零回归；data-hl 是 CSS 侧唯一可靠明暗钩子，先例 hljs-theme.css）；状态徽章（绿 success/琥珀 warning/蓝 primary/灰 on-surface-variant·outline）+网络图灰线（outline-variant）+非工作日红字（destructive）+手动进度灰共 **12 处桥接** `var(--md-sys-color-*, 原值)` 带兜底，与文件既有桥接风格一致；**红线保留 17 处**=导出纸张白底（纸张语义）/分部横幅六色板（v4.160 拍板）/旗标紫·琥珀功能色/深底 tooltip（补 outline-variant 轮廓边框增强暗色轮廓）/有色板底白字；四令牌**禁绑 var(--md-sys-color-primary/error)**——主题预设漂移会改领域编码色（暗夜紫主题下工作条变紫）；补 sched-pulse 动画 reduced-motion 降级（prefers-reduced-motion + ui-reduced-motion 双口径）。文件内非 var hex 36→25（其中 8 行=两档令牌定义，17 处样式声明全为红线保留项）。真机走查**两态令牌 bridged 实锤**（暗 #f04848/#4d8df0 vs 亮 #e02020/#1e6fd9 计算值切换，两态页面零错误）。②**全站空态统一 V3Empty**（新 components/V3Empty.tsx）：antd Empty 默认灰插画与 v3 玻璃轨道语言割裂；新组件=轨道环 SVG（虚线外环 outline-variant+斜向轨道 primary 0.35+实心活核，呼应「地核 G」logo 语言）+on-surface-variant 描述文字（13px/1.6）+children 动作钮透传，image/imageStyle 接受即忽略（兼容旧调用点）；替换散用 antd Empty 共 **7 文件 9 处**（CharacterPage×4/NovelSettingPage/HomePage/PersonaPicker/SearchModal/SkillModal；modelcenter 各 Section 与 CostLibraryPage 已有各自 EmptyState 体系不动）；真机目检搜索零结果态=轨道环+文案协调渲染。③**全局键盘焦点环+选区+字号阶梯**（index.css）：`:focus-visible` 全局兜底环（2px var(--gaea-glow)+offset，**antd 自管焦点态控件豁免防双环**——btn/input/select/switch/slider/pagination/tabs/picker）补齐键盘导航可见性；`::selection` 主题化（color-mix primary 30%，先例 chat-board.css 已用 color-mix）；`:root` 立字号阶梯令牌 6 档 --font-size-xs~xl（新样式勿再写裸 px 字号，存量渐进收敛；index.css 自身 3 处示范替换）。**普查修正**（Explore 代理报告两处不准）：页面进入动画已有（.page-enter 挂 keepAlive 容器）；toast 全站 message 统一（47 文件）无混用；novel-workspace.css 的 hex 多为功能性设计意图（批注便签四色/三套阅读主题）不动。**测试**=schedule 550 用例绿+tsc 0+eslint 0 errors（4 条存量 warning）+V3Empty 面相关 vitest 绿；真机走查三面截图目检（schedule 明暗两态/搜索空态/Tab 焦点环=2px glow 实锤）。**走查导航配方**=`window.dispatchEvent(new CustomEvent('navigate',{detail:{page:'schedule'}}))` 直达板块（壳层 NAVIGATE 白名单），space 键=gaea.shell.space。**门禁**：ci.ps1 全绿/绑定面零变更/版本三处 4.255.0。

## v4.254.0 · 办公·公文规范包补全：「生成」半边——GB/T 9704 公文模板技能资产化（2026-09-12）
> 板块并行池既定余项执行（「GB/T 9704 公文模板内置（也应走技能/模板资产）」，形态随 v4.224 红线拍板：**规范/模板零进内核，只做技能资产**）。gongwen-9704 技能包此前只有「查」半边（docx_layout.py 取数+SKILL.md 规则表），本刀补「生成」半边：`scripts/docx_gen.py` 一键产出 **GB/T 9704 合规空白公文骨架** docx（纯标准库 zipfile+手写 XML，与取数口同依赖口径）。**预置项与排版细则表逐项对齐**：页边距 37/35/28/26mm 精确值；红头=机关名 FF0000 小标宋 36pt 居中+发文字号下红色分隔线；发文字号六角括号〔〕（默认「×政发〔当年〕1号」）；标题小标宋二号（44）居中；正文仿宋_GB2312 三号+28 磅固定行距+首行缩进 2 字符（firstLineChars=200+firstLine=640）；正文段自动分层——「一、」→黑体、「（一）」→楷体；页脚「— 1 —」形态=4 号宋体+PAGE 域+一字线包数字；版记抄送/印发行参数可覆盖。**dogfood 自洽测试**：test_docx_gen.py 9 例=生成→**用自家 docx_layout.extract 复检**断言逐项命中细则表（校验器即验收器）+CLI 冒烟（边距/页码/红头三面 JSON 全对上）。SKILL.md：description 加「生成公文/公文模板」触发词+Workflow 新增步骤 0（生成→填内容→复检）+新增「生成」段参数表。**红线零违反**：零 Go/内核/绑定/前端改动（纯技能资产）。**测试**=python +9（技能内 stdlib unittest）。**门禁**：ci.ps1 全绿/版本三处 4.254.0。

## v4.253.0 · 二遍扫描账本收官：#9 修 + #6 实测结案（2026-09-12）
> 二遍扫描剩余观察项最后清账。①**#9 修**=jobs 的 children 级联链随 `pruneTerminalLocked` 淘汰联动回收（改前只增不清，已淘汰 job 的映射随进程内后台派生数累积）。②**#6 结案**=filewatch 同步 WalkDir **实测结案**——skipDirs 已覆盖 node_modules/.git 等重目录，开发工作区 925 个目录实测仅 **46ms**（.tmp 一次性测试程序）；异步化收益趋零，而「启动窗口事件丢失+无周期重扫兜底」风险真实——风险大于收益，维持同步。**方法论=「结案」也要拿实测证据，让风险判断从拍脑袋变有据；收益趋零的优化不做就是不做**。二遍账本终态=修 7（#2/#3/#4/#5/#8/#9/#10）、结案 1（#6）、观察 4（#1/#7/#11/#12，均已知低危或需独立设计）。测试 jobs 包全绿。零绑定零前端。**门禁**：ci.ps1 全绿/版本三处 4.253.0。

## v4.252.0 · 启动链收尾：远程剧照迁移异步化（普查二遍#3）（2026-09-12）
> 冷启动关键路径再摘一颗：改前 Startup 内同步串行下载全部远程剧照（xAI 临时图，每张 30s 超时，N×30s 全算进冷启动）；改后台协程（recover 守卫，沿用 app 刷新协程先例）。**异步化安全前提核实**=迁移是「先收集 todo → 逐条下载（网络不占库连接）→ 短 Exec 回写」，与启动期其他角色库操作在 characterlib 单连接池上安全交错，无「持连接等待」死锁面；迁移完成前角色卡显示远程 URL=迁移前既有状态，语义零变化；幂等（本版未迁移的下轮启动重试）+本体测试覆盖（TestMigrateRemotePortraits）。MigratePortraitsToFiles（base64 本地落盘无网络）维持同步。二遍扫描账本更新=修 6 观 6。测试=characterlib 迁移本体+app 全量 67.9s 绿。零绑定零前端。**门禁**：ci.ps1 全绿/版本三处 4.252.0。

## v4.251.0 · 后端第二遍扫描：goroutine/内存/启动链（修 5 观 7，goroutine 泄漏零）（2026-09-12）
> 「优化 gaea 的后端」第二遍（换镜头复查首遍欠覆盖维度：goroutine 生命周期/无界内存/锁跨慢操作/启动链；首遍热路径五刀已全清）。**goroutine 泄漏零**（全部 go func 有退出路径，time.After 循环均有上界；schedule/novelstyle/novelcontext/proc/filewatch 全净）；12 项发现**净修 5 项、观察 7 项**（普查档二遍扫描节全表落档）：①**MCP StartAvailable 并行化+单 server 30s 上限**（#4，最高危）：改前串行连接且无 deadline——一个挂死的 MCP server 无限期卡住会话装配；并行化后单 server 超时用 **AfterFunc 取消仅失败路径生效**形态（成功连接的 transport 绑定该 ctx，定时器必须 Stop 否则健康 server 被误杀；fired 竞态下已连上的也撤下并统一按 timeout 上报）；测试=TestHelperHangProcess 挂死子进程（**坑=select{} 会触发 Go 死锁检测器把子进程自己杀掉报 EOF，须 sleep 循环保活**）+「双挂死+好 server 耗时上界 3.5s」同时守卫超时与并行性（串行 >4s）。②**weixin apiPost 每请求 ctx 超时替代克隆客户端**（#2）：改前 timeout<90s 时每次 new Transport——长轮询循环（30s/轮）永久每轮新建+TCP/TLS 握手，keep-alive 全失效；共享 client+ctx 截止语义等价（较短者生效）。③**notifyStart 异步化**（#5）：Startup 链上同步 POST 网络不可达时每助手 10s×N；通知失败无补救语义。④**tasks 节流 map 随 outputs LRU 联动回收**（#8）：lastEmit/lastOutputEmit 改前只增不删。⑤**copyPath 流式拷贝**（#10）：旧数据根迁移改前整文件读入内存（启动内存峰值=最大文件体积）改 io.Copy。**观察 7 项入档**（#1 browser 全程持锁/#3 Startup 串行远程剧照/#6 filewatch 同步 WalkDir 触及启动语义需单独设计；#7/#9/#11/#12 低危顺手级）。**测试**=Go +1（MCP 超时+并行性，3 遍绿）+既有全绿。**门禁**：ci.ps1 全绿/版本三处 4.251.0。

## v4.250.0 · 走查遗留收账：语义召回搜索路径加界 + 观察项②撤销（2026-09-12）
> 「优化 gaea 的后端」尾刀（刀F）：处理 v4.249.1 真机走查入池的两项观察。①**语义召回搜索路径加界**（观察①收尾）：tool 侧 `semanticCostRecall`（cost_tools.go）与 app 侧（gaea_cost_rerank.go）同形态——`st.Search` 内含 Ensure 增量向量化，`ctx 60s` 包住「追赶+查询」，库大或刚导入时追赶吃满 60s 全算进调用方（模型回合/UI 绑定，真机实测 60.1s×2）。**实态收窄**：Ensure 按批持久化增量收敛（EmbedBatch=64、逐批 upsert），60s×2 属 1509 条库**首次追赶窗口**（>120s 未完），收敛后单次查询约 1s——「每次调用必阻塞 60s」的原始表述不成立。**刀法**=搜索路径 ctx 60s→5s 双侧加界：超时回落关键词结果（语义召回只增不改基线，回落语义与改前 ctx 超时一致、只是快 12×）；追赶由既有**显式补齐 GaeaSemanticIndexBackfill**（10min 增量、幂等、用户看着发生）完成；收敛后搜索路径 Ensure 为 no-op（diff 读+单次查询嵌入 <1s）。**提案撤回**：曾拟「导入后后台向量索引钩子」，与 gaea_semantic_index.go 档头既有拍板「不做后台守护协程，覆盖数字与补齐动作由用户看着发生」冲突——按既有拍板撤回，若要自动化交拍板池。②**观察②撤销**：~~Available() 探测口径 /models 404~~——引擎实盘 `base_url = http://localhost:8080/v1`（本就含 /v1，connected=true、bge-m3 在册），探测路径 `BaseURL+/models` 正确；走查时 curl 打的是缺 /v1 的猜测 URL，**证据未复核即入池——同普查 #8 教训（普查证据/走查快结论都要复核）**。**测试**=tool 侧 TestCostSearchSemanticRecall（fake embedder）+app 包相关面绿。零绑定零前端。**门禁**：ci.ps1 全绿/版本三处 4.250.0。**「优化 gaea 的后端」线程至此全收**。

## v4.249.1 · 走查补刀：app 侧检索客户端单例化 + 五刀真机走查清账（2026-09-12）
> 后端性能五刀（v4.245~v4.249）真机走查（CDP 9333 DOM+绑定断言，不点危险操作不打模型；配方 .tmp/walk-v4249.mjs：**壳内绑定挂 window.go.app 门面非 window.go.main.App，跨门面解析+Wails 逐参显式传**）——绑定全通（GaeaContextView/Trajectory/AgentNetwork/GaeaListSessions/GaeaCostList/GaeaMemoryHubOverview，看板绑定亚毫秒=条目缓存真机生效；GaeaCostList 真实库 **1509 条** 62ms 经 LEFT JOIN 查询正常）、记忆中枢/造价数据库两页零「页面渲染出错」、全程 exceptionThrown=0。**走查新发现一并修**（普查#4 的 **app 侧孪生**）：`localSearchEmbedder/localSearchReranker` 每次调用 new 新客户端（60s Available() 探测缓存按实例存永不命中），改按解析出的 (baseURL, model) 为键单例缓存（引擎配置/环境变量变更自动重建；override 测试注入不动）。**观察池新增两项**：①GaeaCostSearch 语义召回在 UI 绑定内**同步 Ensure 全库向量化**（60s ctx；关键词召回<3 时每次调用可阻塞 60s，真机实测 60.1s×2——语义向量进度未在调用间收敛或全库过大）→异步索引/移出同步链按需另刀；②Available() 探测口径 `/models` 404：herdsman（LM Studio）实际部署不实现 /models → Available 恒 false，探测口径与实际部署不匹配按需另刀。**测试**=app 包相关面绿。**门禁**：ci.ps1 全绿/版本三处 4.249.1。

## v4.249.0 · 后端性能刀E：面板/杂项收官（看板缓存 408×+预览缓存+无界缓存上限）（2026-09-12）
> 「优化 gaea 的后端」刀E 收官（普查档最后一刀，benchmark 先行）。①**会话条目缓存**（普查#5）：上下文看板/节点详情/轨迹/Agent 网络（+resync 共五处）绑定共用 ReadEntriesFor 全量读+逐行解析，前端轮询刷新时同份日志被反复解析（基线 2MB 日志 9.1ms/6.7MB 分配每刷）；新增 `readEntriesForCached`——事件日志是 **append-only**（单写入器整行追加），文件 size+mtime 未变 ⇒ 内容未变，缓存解析结果；四绑定接点切换（resync 保持直读=刻意刷新）；缓存切片跨调用方只读共享（fold/detail/trajectory 层核实不原地改写）；上限 8 会话超限清空。**看板刷新 9.1ms→22μs（408×，分配 6.7MB→928B）**。②**会话列表预览缓存+轻量解码**（普查#7）：`previewSession` 对每个 .jsonl 全量解码（含多 MB 工具输出）只为取首条用户消息预览+轮次数；列表调用点换 `previewSessionCached`（size+mtime 键，transcript 整体重写⇒键变，上限 64）；解码改轻量形态——只取 role 字符串，content 仅在确属首条用户消息时才反序列化（tool_calls/reasoning 等大字段零物化）。③**wssearch 正文缓存上限**（普查#10）：textCache 改前只增不删内存无界，加 64 条上限超限清空（mtime/size 校验照旧）。④**graph linker 正则消除**（普查#16）：循环内每次 `regexp.MustCompile` 两处改纯字符串操作（QuoteMeta 后的字面量模式与 ReplaceAll/Contains 语义等价），O(实体数) 次编译归零。⑤platform sanitizeFilename 循环 `+=` 改 strings.Builder（普查#18）。⑥三处函数内 MustCompile 提包级（memory_hub 引用正则/gaea_export 文件名正则/shelf 章节名正则，普查#20）。⑦**SemanticRecall 死代码删除**（普查#19）：全量重 embedding 候选库的遗留危险实现，零生产消费方核实后删除（DocText/Cosine 有消费方保留）；#17 归档分页=legacy file backend 专属路径，观察不修。**测试**=Go +2（条目缓存失效=追加立即可见；预览缓存失效=重写立即可见）+app bench 1 组入库。**门禁**：ci.ps1 全绿/版本三处 4.249.0。**普查档五刀 A/B/C/D/E 全清收官**。

## v4.248.0 · 后端性能刀C：SQLite 连接池放宽 + 批量清理事务化（2026-09-12）
> 「优化 gaea 的后端」刀C（普查档 #6+#11，风险最高单独走一刀+全量回归）。①**连接池放宽 1→4**（普查#6）：Hephaestus.db 改前 `SetMaxOpenConns(1)` 注释写「串行写入最佳实践」但把**读也串到单连接**——WAL 模式本可读写并行；且「rows 未关闭时 Exec 永久等待唯一连接」是整类死锁的根源（internal/characterlib/portrait.go:244、internal/whisper/db/repos/fts.go:25 两处在册注脚，均为绕死锁的「先收集再写回」 workaround）。放宽后死锁类整体消除（两处注脚注释更新、收集再写回习惯保留）；写并发由 SQLite 单写者锁串行化+DSN 逐连接 `_busy_timeout=5000` 兜底（modernc 驱动对池内每个新连接应用 DSN 参数，PRAGMA 不随连接切换丢失；cache_size=-8000 每连接 8MB，4 连接 ≤32MB）；写事务全部「首语句即写」形态，无 deferred 升级死锁面。**Benchmark**（2s 时间制）：并行点查 7.6μs→2.5μs/op（**3.0×**）。②**批量清理事务化**（普查#11）：`memory.CleanupArchived` 逐条 DELETE 各自提交（N 次 fsync）改单事务，并收紧为**原子语义**（改前删一半失败会把整批 doomed 报给审计=虚报；现在全删或全不删如实上报）；`semantic.Stale` 同款单事务。**测试**=Go +2（并发写 8 goroutine×25 全量落库零 BUSY 逃逸；写事务进行中另一连接可读且 WAL 快照隔离——该测试在改前单连接池下会死锁，即新形态的回归守卫）+db bench 2 组入库。全量回归：app 67s/builtin/cost/memory/semantic/characterlib/whisper 全绿。**门禁**：ci.ps1 全绿/版本三处 4.248.0。**普查档余**=刀E 面板杂项，待拍板。

## v4.247.0 · 后端性能刀D：cost 检索固定成本（BM25 缓存+客户端单例+子查询消除）（2026-09-12）
> 「优化 gaea 的后端」刀D（普查档刀D，benchmark 先行；cost_search 是测算/组价的高频工具调用）。①**BM25 排序缓存**（普查#3，T7-3 原设计接通）：Search 每查询对关键词命中子集从零重建倒排（基线 2000 条库 20.6ms/22MB 分配）；**刀法修正**——bm25.Cache 原设计语料=整库，但 Search 现行为=命中子集排序，直接接通会改 BM25 统计口径（IDF/avgdl 随子集变化）——改为**包级缓存**（key=db 池+数据版本+category|status 过滤形态，语料=该过滤形态下 SQL 全捞全量 name 序；Store 即建即弃所以不能挂实例），命中子集按语料下标取分、tie-break=语料序（=name 序与改前一致）；**失效全链**=Save/Delete/SaveCategory/DeleteCategory 成功即推进版本+SelfHeal 实际修复推进+app 批量导入（直写 UPSERT 不经 Store.Save）新导出 cost.InvalidateRankers 显式失效；map 超 16 项整体清空防任意过滤值撑大。②**Embedder/Reranker 单例化**（普查#4）：costEmbedder/costReranker 改前每次工具调用 new 一个（各自新建 http.Client），60s 的 Available() 探测缓存按实例存永不命中——每次语义召回/精排多做一次 /models 探测且 keep-alive 全失效；改按注入的 retrievalRuntime 配置为键缓存实例（SetRetrievalRuntime 变更自动重建，构造失败不缓存下次重试），override 测试注入路径不动。③**子查询消除**（普查#3 附带）：Search 的逐行相关子查询 `(SELECT COUNT(*) FROM cost_entry_components ...)` 改 LEFT JOIN (GROUP BY) 单趟扫描，ComponentCount 语义不变（COALESCE 0=无明细）。**Benchmark**（20x）：Search 500 条 6.1→2.6ms（2.4×）/2000 条 20.6→9.7ms（2.1×，分配 22→10MB）；List 6.9→6.6ms（子查询本非其瓶颈，DB 池命中为主）。**测试**=Go +1（缓存失效语义：写后新条目立即可查/删除失效/分类过滤形态各自缓存互不污染；TestCostSearchBM25Order 原样通过=排序口径守卫）+bench 2 组入库（cost bench）。**门禁**：ci.ps1 全绿/版本三处 4.247.0。**剩余刀路**=刀C SQLite 读写分离（风险最高单独走）/刀E 面板杂项，待拍板。

## v4.246.0 · 后端性能刀B：记忆写路径（重载出锁+写路径事务化）（2026-09-12）
> 「优化 gaea 的后端」刀B（普查档 docs/gaea-backend-perf-survey-2026-09.md，benchmark 先行）。①**全库重载出全局锁**（普查#2）：QuickAdd/SaveDoc/UpdateFact/ChangeFactType/ForgetMemory/QueueMemory/PromoteSessionFacts/SaveDreamFacts/DistillMerge 九个写点改前都在持控制器全局锁 c.mu 时执行 memory.Load（SELECT 全部 facts 含 body+全量分词+文档发现，基线 50 条=2.3ms/300 条=7.0ms）——一次 remember/quickadd/forget 就阻塞 Send/Cancel/事件等全部控制器操作。**刀法修正**（非拍板池原案的「后台刷新」）：刷新保持**对调用方同步**（写后立即可见的语义与既有测试依赖不动，面板/蒸馏候选集/DistillMerge 后重算零变化），Load 改在锁外跑——`beginMemoryReloadLocked`（锁内收口 Options+推进写侧代号 memGen）→ 释放 c.mu → `finishMemoryReload`（Load 后按代号提交：gen ≥ 已换入代号才换入，**慢的旧 Load 被拒绝换入而非回退快照**；被跳过的写必然已被更高一代 Load 读到——Load 起点晚于该写）；空间切换 applySpace 同步推进代号使在途旧空间快照失效（改前 applySpace 本就不刷新，语义一致）；新增 refreshMemory()（测试/诊断用）。②**写路径事务化**（普查#12）：sqliteBackend 五写方法（Save/Archive/Unarchive/ChangeType/touchInSpace——Touch 在引用解析热路径上，每个被 [MEM:] 引用事实一次）改前 facts UPDATE 与 memory_events 事件是两条独立写事务=WAL 下双倍 commit/fsync；新 `commitWithEvent` 单事务提交，事件仍尽力而为——事务内事件语句失败→回滚并退回单语句重放实体语句（写入本体不因事件失败而丢，语义同改前）；EventLog 抽 execer 接口（appendEventExec）——事件必须落在调用方事务连接上（Hephaestus.db 单连接池，事务持有期间任何 l.DB.Exec 都会死锁）。**Benchmark**（20x）：Save 0.95ms→0.49ms（1.9×）/Touch 0.86ms→0.45ms（1.9×）；Load 本体不变（其收益=不再阻塞全局锁）。**测试**=Go +5（事件失败回落保写入本体/Save+事件原子提交/未命中行零留痕语义不变/Unarchive·ChangeType miss 报错；control 并发收敛=4 goroutine×5 写后 WaitGroup 全返回快照含全部 20 条）+bench 3 组（memory bench 入库）。**复核撤销普查#8**：BuildSearchIndex 实际分词 body（search.go:66），「索引不 SELECT body」会改变 memory_search 语义——证据不成立已勘误。**门禁**：ci.ps1 全绿/版本三处 4.246.0。**下一刀候选**=刀C SQLite 读写分离（风险最高单独走）/刀D cost 检索/刀E 面板杂项，待拍板。

## v4.245.0 · 后端性能刀A：回合边界税清偿（事件日志续接+摘要增量化）（2026-09-12）
> 「优化 gaea 的后端」首批落地（普查档 docs/gaea-backend-perf-survey-2026-09.md 20 项交拍板池，本刀=刀A 两项，均先立 benchmark 锁基线再动刀）。①**回合边界事件日志全量重读清偿**（普查#1）：sink 在 turn_done 干净关闭写入器、下事件懒重开，而 OpenLog 每次=RepairLogFile 全量读+countLogLines 全量读+逐行解析——长会话每条用户消息固定 O(会话总字节) I/O。**修复**=新增 OpenLogResuming 续接形态：干净关闭时 sink 记下「关闭时 seq+刷盘后文件大小」，重开时大小未变 ⇒ 尾部完好行数未变，跳过全量修复+解析直接续 seq；大小不符（外部追加/轮转/截断/删除）自动回落 OpenLog 全量路径，「回合间日志持久、可被外部工具触碰」设计红线由大小核对保住；OpenLog 本体与其他三处一次性调用方（save/migrate/subagent_promote）零变化。②**逐消息摘要增量化**（普查#14）：stream() 每步 DigestMessages 对全部历史消息重哈希 O(会话总字节)/步；新增 DigestCache——相邻请求前缀以**指针级身份**核对（Go 字符串不可变 ⇒ 数据指针+长度相等即字节相等；ToolCalls 同底层数组即同元素），未变消息复用上一请求摘要、仅重哈希新增/改写消息，身份不符一律重算（保守方向，前缀稳定证据链只会重算不会错判）；DigestCache.Digest 与 DigestMessages 逐元素等价，a.lastDigests 链路语义零变化。**Benchmark**（benchtime 20x，AMD RYZEN AI MAX+ 395）：日志重开 256KB 11.1ms→0.84ms（13×）、2MB 19.1ms→0.96ms（20×，与日志大小无关=O(1) 实锤）；摘要步进 50 回合 103μs→4.8μs（21×）、300 回合 521μs→49μs（10.6×），成本从 O(总字节) 降为 O(新增字节)+O(消息数) 指针对比。**测试**=Go +9（续接快路径 seq 连续/外部追加回落/截断回落/删除回落/sink 跨回合端到端/sink 外部写回落；摘要缓存等价六形态/复用计数/重建切片保守重哈希）+bench 3 组基线入库（log_bench/compile_bench，gaea 热路径首批 benchmark）。零绑定零前端。**门禁**：ci.ps1 全绿/版本三处 4.245.0。**下一刀候选**=刀B 记忆写路径（remember 全局锁内全库重载）/刀D cost 检索（接通闲置 bm25.Cache），待拍板。

## v4.244.0 · 记忆中枢办公面板口径对齐：facts 直读管理面（观察池清账）（2026-09-12）
> v4.243 真机走查观察池项清账：办公记忆库面板显示「不可用—未配置」而同库生命周期/注入体检读出 2 条活跃——**口径分裂实锤**。**根因**=GaeaMemory() 的 facts/available 等引擎控制器（gaeaCtrl/c.Memory）构建才有，而数据实体在 SQLite 经 hubOfficeStore 直接可读（与 GaeaMemoryLifecycle/注入体检同路径）；新起壳控制器未建时面板误报不可用。**修复**=facts/available 改为**管理面直读**（hubOfficeStore.List()，与面板上生命周期行同口径；available=用户目录可解析），docs/scopes/storeDir 仍是引擎域文件面、控制器就绪时补充——「不可用」退化为真不可用（用户目录不可解析）。零绑定零前端改动（面板空态分支本就区分 available 与暂无记忆）。**测试**=Go +1（控制器未构建形态：store 可读即 available+facts 直读+Enabled 缺省）。**门禁**：ci.ps1 全绿（vitest 332 文件 2893 例）/版本三处 4.244.0/真机复验=重建壳面板显示事实列表不再误报不可用。

## DAG 6.3 壳内真机走查（2026-09-12，非版本刀）
> §6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 起壳 DOM 断言走查（不点危险操作不打模型）——①注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库端到端打通（通过：五条结构不变量全部成立；晨报预载 2 条·104/600 runes、项目本体 2 引用·181/600 runes、门控四开关透出、固化覆盖「库内无固化条」诚实口径；截图 .tmp/eval-drawer.png）；②办公工作台流水线区 dag-section 在位、空态引导文案正确；③全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径口径不同，既有语义非本线回归，按需另刀。零代码改动不抬版本。

## v4.243.0 · 运行中 steer 直穿 + 危险操作分级审批（DAG §6 末项，绑定 627→628）（2026-09-12）
> DAG 设计档 §6 最后一条功能余项清账（roadmap §12.4「GaeaSteer 运行中改向，危险操作分级审批」）。①**运行中直穿**=主对话 GaeaSteer 同机制下沉子代理：agent 包加在跑登记 `subRunners`（ref=SessionID→AgentRunner，runSubAgentInternal 起跑登记/defer 注销）+ `SteerSubagent(ref,text)`；GaeaDagNodeSteer 放行 running 节点——凭 node.Ref 注入在跑 runner 的 steer 队列（不打断工具执行，下一回合生效），查无登记（恰收跑/ephemeral）如实报错不静默转续跑。②**危险操作分级审批**=dag_plan 节点参数 `risk`（normal 缺省/high=覆盖删除既有文件、批量移动、全局性改动，Validate 拒非法值）；**执行器起跑前置闸**：dagExecute 每节点起跑前 high&&!Approved → 置 StatusHold「待审批」+波收尾**链停**（下游保持 pending 不级联跳过）；新绑定 `GaeaDagNodeApprove`：hold→pending+Approved=true，**只放行不自动跑**（起跑仍人拍板同闸）；已批准重跑不重新挂闸；ApplyEdit 把 Risk 计入形状比较——**风险变更=回 pending+审批归零**（新风险重新批）；FromRun/Instantiate risk 随模板、approved 剥净。前端：hold 徽标 var(--warn) 待审批+「批准」按钮+running 节点直穿改向框（占位文案区分直穿/续跑）。**绑定 627→628**（gen_bindings 再生+office 门面手工补行+checkout 还原单参化，diff 恰 1 行）。**测试**=Go +4（风险值校验/ApplyEdit 风险变更归零审批/模板携带/审批闸端到端=hold+链停+approve 放行+已批准重跑不挂闸；SteerSubagent 寻址三态）+vitest +2（hold 徽标+批准/直穿占位）。**门禁**：tsc 0/eslint 0 error/ci.ps1 全绿（vitest 332 文件 2893 例）/drift OK@628/版本三处 4.243.0。**§6 余项剩一条**：壳内真机走查。

## v4.242.0 · 办公流水线成品直出首刀：一键验收（市场调研候选3，绑定 626→627）（2026-09-12）
> survey §5 候选3 防御性首刀（LM Studio Bionic 竞品信号），DAG 设计档 §6 余项 DeliverableRegistry 用户面清账（续写该档不另立）。**差距**=MS Agent Mode/WPS 把「一句话→成品」做到大众可用；gaea 流水线产物散在各节点证据卡、逐节点验收=N 次盖章。**口径**：成品=done/accepted 节点 outputs（工作区相对路径）的 run 级汇总；**验收语义零变更**——仍人拍板（一键=一次拍板覆盖清单所列节点，两段式按钮首击只武装、再击执行、重拉自动解除防陈旧误击），不自动验收不定时验收（「人拍板回流记忆」红线不动）。**落地**：① Go=提取单节点验收记忆回写核心为 `dagAcceptMemoryWrite`（负载单一真源，单验收/一键验收同语义）+ 新绑定 `GaeaDagAcceptAll(id)`（锁内一次翻转全部 done 节点+save 一次；锁外逐节点回写，单条失败不阻断其余汇总如实上报=「验收已成立不回滚」同哲学；无可验收返回明确信息非错误）；**绑定 626→627**（gen_bindings 再生后 bindings_office **手工补行**+checkout 还原 v4.237 单参化，office 门面 diff 恰 1 行）。② 前端=DagPanel run 头「M 件成品」计数徽标（title=文件清单）+「一键验收 N」两段式按钮（运行中/无 done 隐藏）+五处接线（bridge 接口/mappings/bindingNames/spaceBindings work 域+锁 448→449）。**测试**=Go +2（一键全收 3 节点+产物计数+记忆逐节点回流+空验收提示+混合态只收 done 余量 1 个；单验收回归=重构后状态/记忆/拒绝语义逐条不变）+vitest +2（成品徽标口径/两段式武装与 running 隐藏）。**门禁**：tsc 0/eslint 0 error/ci.ps1 全绿（vitest 332 文件 2891 例）/drift OK@627/版本三处 4.242.0。**§6 余项剩**：运行中 steer 直穿+分级审批/壳内真机走查。

## v4.241.0 · 记忆注入质量可评测（市场调研候选2，绑定 625→626）（2026-09-12）
> survey §5 候选2 落地，与检索评测分立：`GaeaRetrievalEvalRun`（v4.196，Recall@10≥0.8 统计门槛）测**检索面**（查得到），本刀测**注入面**（注得对）——对真实库的 work 视图跑与 boot/sysprompt.go 装配点**完全相同**的两个真实构建器（晨报预载 v4.16 + 项目本体 6.2），断言结构不变量：预算合规×2（600 runes）/归档与跨空间零泄漏（**精确路径**=预载条目名+本体引用键逐点核对；块全文子串兜底明确不做——work 条目正文合法提及归档名不是泄漏，子串必误报）/引用零悬空（悬空注入=模型会引用不存在的键）。**门槛=零违规**（确定性规则无统计门槛，面板注明与 0.8 分立）；固化覆盖（PinnedTotal/InBrief/MissingPinned）是信息面不判死——预算挤兑下固化未全收是「预算内按行收录」设计内行为；门控三开关（memory.enabled/morning_preload/project_brief）随报告透明下发，关态结构体检照跑 Note 注明。**形态**：纯核 `memory.EvalInjectionBlocks`（不读时钟不 IO，同输入同报告）+ 绑定 `GaeaMemoryEvalRun`（gaea_memory_eval.go；门控读取与装配点同源=morningPreloadEnabled/projectBriefEnabled 两函数+gaeaLoadConfig 兜底）+ gen_bindings 再生（**绑定 625→626**）+ 前端六处（types 手写接口 MemoryArchivedPage 先例/bridge 接口/mappings/bindingNames/spaceBindings work 域+锁 447→448/mock 全合规样例）+ 办公记忆库工具栏「注入体检」Drawer（渐进披露不加一级房间）。**测试**=Go +7（纯核五不变量逐形态+固化覆盖信息面+空输入；绑定接线真库全合规+关态 Note）+vitest +3（通过态/违规态/失败回退）。**坑两实锤**：①**gen_bindings 再生会回退 v4.237 的手工单参化**——脚本从 App 方法签名派生门面，手工修的七个变参签名被还原成 ...string，E27 守卫当场拦下（守卫价值实证）；**以后 gen_bindings 后必须 git diff bindings_cost/office 两文件，凡含变参行即手工还原**；②App 的 `a.cfg` 是 *appconfig.Config（偏好域 GetMorningPreload/GetProjectBrief），`ga.cfg` 才是 gaeaConfig（引擎域 Memory.Enabled/SpaceModeIsOn）——两域门禁读取不可混用，且裸 `&App{}` 读 a.cfg 会因 core 未初始化直接 panic（测试须 `&App{core: &core{cfg: ...}}`）。**门禁**：tsc 0/eslint 0 error/ci.ps1 全绿（vitest 332 文件 2889 例）/drift OK@626/E27 PASS/版本三处 4.241.0。

## v4.240.0 · 市场调研落地 + 记忆双时间轴 SchemaV21（2026-09-12）
> 「优化迭代 gaea」先调研后动刀：**八赛道市场调研落档 `docs/gaea-market-survey-2026-09.md`**（办公/记忆/造价/进度/小说/陪伴/本地优先/通用智能体）——微软 Copilot agentic（Word/Excel/PPT 多步原生操作）2026-04 GA=办公 agent 形态被巨头验证且竞争加剧；Zep/Graphiti bi-temporal（事实时间 vs 事务时间）=2026 记忆层共识口径；LM Studio Bionic（本地 agent 直出文档/幻灯片）=本地 agent 工作台正面竞品，「本地+隐私」已是基线不再是差异点，护城河必须压在真编辑×私人记忆×专业域×闲庭组合；陪伴市场高增长验证闲庭粘性价值；进度专业排程（CPM/双代号）AI 化个人级空席位=维持现状即差异化。**候选 4 项交拍板池**（记忆双时间轴/记忆可评测/办公成品直出/价格带数据源观察窗），已过删除史筛——平台化/MCP 大全/跨设备记忆包/算量/清标询比价/PM 任务协作层/GoalCard 系/新板块均不提名。**本版落地候选1=记忆双时间轴**：SchemaV21 `memory_events` 加 `recorded_at`（事务时间=日志写入时刻，`AppendEvent` 服务端盖章、调用方传值一律覆盖不可伪造；`At` 单一语义=事实时间；旧行 0=读取归一回落 At，与 V20「零成本回填+诚实缺省」同口径；对标 Zep valid/transaction time）；投影 `eventTimeNote` 把「（发生 YYYY-MM-DD HH:MM · 记录 …）」标注烘进事件节点 Desc——仅两轴分歧≥1s 才标注（写入路径 At 取写入时刻同毫秒级是常态零噪音），「事实在过去、写入在当下」的回填/导入/做梦衍生类路径才透出审计信息。零绑定零前端改动（GraphView 渲染 desc 现成面）。**测试**=Go +6（V21 升级旧行回填/新库全链/盖章反伪造（伪造 RecordedAt 被覆盖+事实时间保留）/旧行归一/cite 路径盖章/desc 标注含 abs 差与阈值边界）。**坑**=双轴曾计划作为 GraphNode 新字段，被 TestGraphRebuildFromLog 拦下——mem_graph_* 物化表没有这两列，字段化破坏「物化=日志投影」逐字段可比不变量；改烘 Desc（desc 本就随投影物化）不变量天然保持——**给投影产物加字段前先查物化表有没有这列**。**门禁**：db+memory+app 三包绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 625 零变更/版本三处 4.240.0。**下一刀**=候选2 记忆注入质量可评测（与既有检索评测分立）；候选4 价格带数据源待拍板。

## knip Unused exports 甄别首批（2026-09-12，非版本刀）
> 维持轨项（.gaea/todos.md ⬜→🔶）。knip@5 + 新增 frontend/knip.json 固化口径（ignoreExportsUsedInFile + test 入口； genui/icons barrel 白名单）——全量 441 项过滤后 **真死候选 56 项**。**首批删除 14 项零引用声明**（stats.ts×7：priceFor/calcCost/fmtElapsed/filterSteps/withHitRate/mergeCols/DEFAULT_PRICE；subagentRunsStore getSubagentRunsSnapshot；officeTurnProjection OFFICE_READ_TOOLS；workspaceTabs WORKSPACE_TAB_COMPACT_WIDTH；api/engines getSemanticIndexStatus/getBenchmarkDetail；chat/constants CHAT_SIDEBAR_KEY；schedule/customFields customUsedKeys；gsapAnimations DUR_FAST）。**剩余 46 项三态账本**：守卫 KEEP 12（spaceBindings/drift 编译期 canary + wails.d.ts 生成面）+ DEFAULT 组件 7 + 真死待删 27（下一批 surgical 删除）。**第二批（同日）surgical 删除 31 项**：DEFAULT 组件 7（SelectionToComposer/ModelDirectory/BenchmarkSection/CostIndicatorsView/CostNotesView/ChatInspector/EmotionSpeakSelector）+ genui barrel 断链 4（renderGenuiSpec/clearBlockState/setMaxPartialRepairAttempts/tryParseFence + index.ts 再导出行）+ 类型/接口 12（Tone/CanvasChapterData/BackendEventName/BridgeWatchState/FileIndexStatus/XlsxCellChange/MemoryLifecycleItem/MemoryMergeSuggestion/FactView/TraceStep/RetrievalEvalQuery/ColStatsWithRate/SemanticIndexStatus/BenchmarkRunDetail）+ stats hitRate→hitRateColor 勘误（同名前缀撞删误伤，已恢复在用函数）+ StatsPanel 孤儿持久化块清除（loadData/saveData/StoredData/TurnRecord/useEffect/useRef 级联）。**剩余 15 项 KEEP/预留**（编译期 canary×6、wails.d.ts 生成面、SPINNER_WORDS/getLocale/Block/Field/MODEL_PRICES 预留面）。**坑**=①括号配平删除脚本对无分号单行常量吞后续行（git checkout 恢复后单行精确重删）②同名前缀撞删（hitRate 撞 hitRateColor）——声明块删除必须 tsc+git diff+全量 vitest 三验证。**门禁**：tsc 0/eslint 0/受影响面 55 例绿。

## v4.239.0 · 数学公式渲染对齐：伴侣线接入 KaTeX（2026-09-12）
> 「AI 输出对齐」补遗刀：**伴侣/聊天线（ChatMarkdown/MarkdownContent）一直没有数学渲染**——$E=mc^2$、$$..$$、\(..\) 全部裸显。**落地**：① 新 gaea/lib/mathText.ts 跨板块共享（自 gaea Markdown.tsx 提取 normalizeMath/hasMathContent/ensureKatexCss，行为零变化）；② ChatMarkdown plain 终态管线加 remark-math + rehype-katex，text 先过 normalizeMath，有数学内容才注入 KaTeX CSS；③ MarkdownContent companion/流式同款（GenUI 覆盖件路径不受影响）。零新依赖零新 chunk（katex 模块/样式经既有共享 chunk 复用）、无 Go 改动、绑定 625 不变。**测试**=vitest +2（两线 $..$ → .katex 真实渲染）。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 625/版本三处 4.239.0。

## v4.238.0 · 观察池收官：回答反馈（点赞/点踩）落记忆事件（2026-09-12）
> **设计取舍使「无消费方」顾虑消解**：反馈不做成孤立数据，而是**记忆事件**（op=feedback）落 memory_events 日志——事件节点随 v4.210 语义图谱投影自然可查，消费者=事件图谱/事件日志现成面。**Go（记忆域）**：OpFeedback 常量扩员；sqliteBackend.AppendFeedbackEvent（摘要 eventExcerptLimit 截断+At/Actor 缺省补齐）；fileBackend 诚实报错（Pin 先例）；spaceView 最小透传；Store 值方法分派。**App 绑定 624→625**：GaeaMemoryFeedback(messageID, rating, excerpt, space)——rating 限 up/down 非法拒绝；事件字段 Name(messageID)/Title(点赞回答|点踩回答)/Space/SourceSession(当前会话)/SourceMessage/Actor(panel)；bindings_memory 透传一行。**前端**：AssistantMessage 操作行 👍/👎（icons +ThumbsUp/Filled/ThumbsDown/Filled 四件）；一次有效（事件追加式不可撤回，点击后双钮禁用）、失败回退可重试+error toast；Transcript onFeedback 开关全链透传。**测试**=Go +3（落库全字段/投影只出事件节点不建实体/文件后端诚实报错）+vitest +3（落库参数+禁用/失败回退+toast/熄灯口径）。**坑**=①Transcript 批量接线脚本末位断言崩掉整体不写盘，重做只补 props 声明漏了 ctx/调用点——批量接线后必须从调用点正向全链 grep 复核；②Wails 要求绑定参数逐个显式传（facade 可选省略=count 错），调用点显式传空串（Go 侧空串=不过滤/当前会话语义核实安全）。**门禁**：Go app+memory 绿/vitest 87 例相关面绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624→625（drift/completeness/spaceBindings 锁同步）/版本三处 4.238.0。**真机复验**=点👍→已反馈态+语义图 ev:10 "feedback · 点赞回答" 事件节点含摘要——端到端全链打通。**观察池全清**；反馈深度消费（按反馈加权重排记忆）属记忆 OS 后续方向按需另刀。

## 壳内全板块渲染健康巡检（2026-09-12，非版本刀）
> v4.237 收尾巡检：CDP 起壳逐 rail 页点击 + 错误边界断言，覆盖**闲庭空间七页**（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）与**书斋空间六页**（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）——**13 页全部干净**（「页面渲染出错」零接管，办公页内容满载 13834 字符）。巡检配方沉淀 .tmp/walk-pages.mjs 形态（rail 坐标 DOM 定位 + 逐页 4.5s 停留 + 错误文本断言）。零代码改动不抬版本。

## v4.237.0 · 变参绑定根治：绑定层去变参 + Wails 变参真相定论（2026-09-12）
> v4.236 单串化方向对但没修到根：轨迹页专项走查暴露新错误形态（json: cannot unmarshal string into Go value of type []string），追到 **Wails v2.13 源码 + 真机经验矩阵双重定论=变参绑定在 Wails v2.13 根本不可用**（ParseArgs 严格计数+逐参 unmarshal 口径为 []string，而 reflect.Call 按 In(0)=string 校验——传串 unmarshal 失败、传数组 reflect panic 被 recover 后回调永不送达=promise 永久 pending；七形态×双方法×3s 超时经验矩阵全败，对照无参绑定秒回）。**修复**：bindings_office.go 六处+bindings_cost.go 一处绑定门面 ...string→普通单 string（透传核心层；resolveGaeaSessionPath 跳空回退当前会话/taskListInSpace("")=不过滤语义已核实）；前端 facade 签名改**必填串**、调用点显式传 ""（真机日志实锤 Wails 要求逐参显式传，省略=count 错）；回归锁与三处断言跟随。**真机复验**=重建壳 wire 错误 0 条 + 轨迹页红卡消失满载数据（Duration/Turns/Calls+完整事件线+子代理树 7 节点）+ 任务管理实时行在位——对比 v4.235 红卡/v4.236 空面板修复实效确凿。**坑固化**=①Wails v2.13 变参绑定不可用，绑定面禁用 ... 参数（核心层可保留）；②修 wire 契约后必须重建 exe 再复验（旧壳跑旧前端，本轮踩中浪费一轮探针）；③被 .catch 静默吞掉的接口错误，修完报错≠修完功能，必须验证数据真到达。**门禁**：Go app 67s 绿/vitest 87 例相关面绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.237.0。

## v4.236.0 · 修复变参绑定 wire 契约：reflect 参数错误清零（2026-09-12）
> 清掉 v4.235 走查挂出的观察池项。**根因（桥接约定与 Wails 真实形态不符）**：Go 五个变参绑定（GaeaAgentNetwork/GaeaTrajectory/GaeaContextView/GaeaContextNodeDetail/GaeaTaskList，均 ...string）要求每个变参元素作为独立顶层参数（args:["path"]），桥接 facade 却约定 JS 数组整体作单参（args:[["path"]]）→ reflect 报错；**推断=v4.174 退役的 wailsjsCompat shim 当年会展开数组，退役后数组调用点静默失去展开**，接口报错被 .catch 吞掉，直到壳内走查抓日志。已核 UnifiedSearch/CostImportApply 调用方本就按字符串/展开传参不动。**修复（facade 单可选串化，全部调用方只传 0..1 个路径）**：bridge/core.ts 五签名 string[]→?:string；七调用点同步（AgentNetworkCard/agentNetworkStore/TrajectoryView/ContextView/inspector 的数组包裹拆除，TaskCenter/useRunningBadge 的 TaskList([])→TaskList()）；lib/mock/core.ts 四实现对齐；agentNetworkStore 补语义（poller 空串=内核会话哨兵，下发前归一 undefined）。**回归锁**=新 bridge.variadic.test.ts 4 用例钉死 wire 形态（单路径→顶层 string/无参→零实参）；agentNetworkStore/ContextView 三处旧数组断言更新。**真机复验**=重建壳进办公工作台 reflect 错误 0 条（修复前必现）。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.236.0。**欠账**：观察池剩点赞点踩（等真实需求）。

## v4.235.0 · 壳内真机走查：抓到并修复 DAG 模板区 nil slice 崩溃（2026-09-12）
> 「测试绿≠真机通」再实锤：五刀（高亮×3/regenerate/折叠）壳内 DOM 断言走查，进办公工作台第一步撞上页面级崩溃——vitest 330 文件全绿没拦住。**根因**=用户机器模板目录不存在 → `DagTemplateStore.List()` 返回 `nil, nil` → Go nil slice 序列化成 JSON **null** → 前端 `tpls.length` 炸（测试/mock 给的都是 []；v4.210 nil↔[] 漂移同族、跨语言面）。**修复（双侧归一）**：① Go dag.Store.List/TemplateStore.List 的 IsNotExist 分支返回空集非 nil；② DagPanel 三处绑定边界 `?? []`；③ Go 用例强化空目录 List 断言非 nil（原 len 断言对 nil 也过拦不住）。**真机复验**=重建壳重进正常渲染错误 0、流水线区在位。**走查其余结论**：data-hl 全链路双主题真机工作（首查见 light 是翻错键——gaea-dark 是被 gaea-display-mode 压制的 legacy 键非 bug）；regenerate 按钮真实会话在位（DOM=1）；方法论沉淀=Runtime.enable 收 exceptionThrown 抓错误边界底下的堆栈、minified 崩点按 dist chunk 行:列切上下文反查。**新观察池**=GaeaTaskList/GaeaAgentNetwork 的 reflect []string as string 参数错误（预置，接口报错不崩页面，按需另刀）。**门禁**：Go dag/app 绿/vitest/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.235.0。

## v4.234.0 · AI 输出渲染：超长代码块折叠（2026-09-12）
> 观察池第二项清账（对齐 Claude/ChatGPT 长代码处理）。**落地**：① 新 `gaea/components/CodeCollapse.tsx` 双板块共享——>30 行块级代码默认收起（max-height 380px+底部渐隐+「展开全部（N 行）/收起」胶囊钮），≤30 行零包装直通，折叠态保留横向滚动；渐隐色由调用方显式传（gaea 默认 var(--bg)，聊天线传 #0b0e14 同其恒暗面板），按钮用全局 M3 令牌内联样式，文案调用方注入 labels（gaea 走 useT 三语 msg.codeExpand/codeCollapse，聊天线硬编码中文一致）——**跨 chunk 共享组件不能假定对方加载了 gaea/styles.css 令牌**。② 新 useTOptional()：无 LocaleProvider 回退 zh 直译不抛错——Markdown 是被裸渲染的共享组件，useT 会让全部裸渲染用例连坐抛错（首跑 27 例挂即此）；运行态恒有 Provider 真实三语不受影响。**测试**=vitest +2（40 行围栏默认收起+展开/收起切换；短围栏零包装）。**③ 顺带修 LocaleProvider context value 不稳定（真回归，三步实验定位）**：value 原先每渲染造新对象，Markdown 经 useTOptional 成为消费者后 en chunk 就绪的 forceRender 把它拖进重渲染级联、ReactMarkdown 整树重解析撕掉 MemCitationChip 弹层（CI 4 例挂+隔离稳定复现；stash 二分+去包裹/去 hook/memo 化三实验锁定）——修复=value useMemo 化，消费者只在语言真变时重渲染。**坑四证（固化铁律）**：python heredoc 追加含 
/反引号源码本会话第三次转义丢失——追加源码只走 Edit 工具无例外。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.234.0。**观察池剩**=点赞点踩（等真实需求）/壳内真机走查（五刀汇总）。

## v4.233.0 · AI 输出渲染收尾：MarkdownContent 缺省高亮 + 明暗调色板目检（2026-09-12）
> 高亮三刀收尾。**① 目检补证**：v4.230/231 此前只有 vitest DOM 断言没有视觉验证——Node 同款语法包生成 go/sql/python/powershell 样例 hljs HTML → check.html 双 link 真实 hljs-theme.css（双面板=gaea 随主题/聊天线恒暗 .hl-scope-dark）→ 无头 Edge 截图明暗两版人工目检：暗色层次清楚、浅色同色相深阶对比良好、**恒暗面板在浅色主题下令牌仍钉暗色组按设计工作**。**② MarkdownContent 缺省高亮**：NovelSettingPage.tsx:189 直用无覆盖是全仓最后一个无着色渲染面——模块级 defaultComponents（引用稳定不破坏 memo）：未传 components 时块级代码走 ChatCodeBlock（暗面板+高亮+复制头与聊天线同款）、行内交还 .md-content code 默认样式、pre 透传防双层；传了 components（GenUI 缝）完全尊重调用方零变化。**测试**=vitest +3（无覆盖走面板+异步令牌/行内不进面板/有覆盖零变化）。**顺带根治 Go 在册 flaky（两天两度打挂 CI）**：TestCreateChapter_SameChapterConcurrentRejected 家族失败从来不是断言而是 t.TempDir() 清理竞态——CancelCreateChapter 取消路径先删登记表，被取消协程仍有「已生成部分落盘」尾步，waitGensDone 只等表空放行即撞 Windows unlinkat；根治=writingState 增 chapterGenWG（spawn 前 Add/协程首 defer Done）+waitGensDone 两级等待（表空快速路径+WG 5s 超时等真退出），-count=10 全绿。**坑再证**：python heredoc 追加含反引号 fence 的测试代码第二次踩转义丢失坑——追加源码只走 Edit 工具无例外；后台 CI 管道 tail 截丢归因必须整份落盘。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.233.0。**主线全清**：观察池=点赞点踩（个人工具暂无消费方）/超长代码块折叠/壳内真机走查。

## v4.232.0 · AI 输出渲染第三刀：重新生成 regenerate（2026-09-12）
> 「优化 AI 输出效果对齐同类产品」收口刀：渲染两刀（v4.230 高亮/v4.231 聊天线对齐）后补齐消息级交互最大差距——对最后一条回答一键重新生成。**设计=复用既有原语零新绑定零 Go 改动**：GaeaRewind(turn,"conversation") 截断该轮（含用户消息）+ GaeaSend 原样重发，缺的只是编排与入口。**落地**：① controller.regenerate(turn)=守卫（运行中/有排队未决拒绝；**只对最后一轮开放**，更早轮次语义交给回退/分叉）→ rewind 成功后 send(该轮用户文本)；rewind 失败不动现场（既有 failWrite 可见化）；rewind 顺带改返回 Promise<boolean>（既有调用方忽略返回值零破坏）。② UI=AssistantMessage 操作行（复制/沉淀旁）新增「重新生成」（RefreshCw），显形=当前会话最后一条 assistant 且非运行中非流式——**半截取消的回复同样可重发**（rewind 把半截正文与 warn notice 一并清掉后干净重来，正是 cancel 后的高频动作）；props 用 canRegenerate 布尔+稳定 onRegenerateTurn 回调不击穿 TurnBlock/AssistantMessage memo 链。**测试**=vitest +7（store.t74 +4=最后一轮编排 Rewind(1,"conversation")+Send(原文)/rewind 失败不重发且可见/非最后一轮拒绝/运行中拒绝；Message +3=按钮渲染按轮号回调/无资格不渲染/流式中不渲染）。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.232.0。**欠账**：任意轮重生成（其后轮次级联重跑=连续 LLM 调用+费用）按需另刀；SubmitDisplay（display≠submit）形态消息重发以显示文本为准（附件形态诚实降级）；壳内真机走查挂观察池。

## v4.231.0 · AI 输出渲染对齐同类产品第二刀：聊天线（伴侣/流式）代码块高亮（2026-09-12）
> 接 v4.230 同方向续刀。**缺口**：ChatPage 两模式代码块全无着色（plain 终态 ChatMarkdown 纯文本；companion/流式经 genuiAdapter 后非 genui 代码只落裸 `<code>`），且 `.hljs-*/--hl-*` 规则都在 gaea/styles.css 里而**该文件不在 ChatPage 加载范围**（仅四个 lazy chunk 引用）。**落地**：① hljs 三块自 styles.css 迁出为新 `gaea/hljs-theme.css`（色值单一真源跨板块共享），重构为双常量组：`--hl-dark-*/--hl-light-*` → :root 兜底暗色 + [data-hl=light] 跟主应用明暗；迁出前核 tailwind 桥接零页面消费方、全仓无第二处 .hljs 定义。② HlCode 自 Markdown.tsx 抽出为 `gaea/components/HlCode.tsx`（css 随组件 import），vite 自然 hoist 成办公/聊天共享异步块。③ 新 `components/ChatCodeBlock.tsx`（两模式共用）=复制头+暗色面板+高亮；面板维持「行业标准暗色专用色不随主题」在册决定，根节点挂新 `.hl-scope-dark` 元素级钉死暗色组——**令牌随面板不随应用主题**。④ 接线两处：ChatMarkdown 块级分支换 ChatCodeBlock；genuiAdapter 非 genui 块级代码同走（data-genui-host 透传不落 md-content pre 默认底）——companion/流式与 plain 终态同款。⑤ 白名单 22→27（+ruby/php/kotlin/perl/lua）。**测试**=vitest +4（plain 暗面板+异步 hljs 令牌/companion 同面板/白名单直通）。**坑**=①python heredoc 往源文件追加代码时反引号的十六进制转义在非 raw 串里提前变成真反引号，产出语法错的 JS——追加源码一律走 Edit 工具；②eslint no-raw-hex 豁免标记必须与 hex 同行，提行注释不豁免。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.231.0。**欠账**：NovelSettingPage 直用 MarkdownContent 未着色（非聊天面按需另刀）；重新生成/点赞点踩按反馈另刀；壳内真机走查挂观察池。

## v4.230.0 · AI 输出渲染对齐同类产品首刀：聊天代码块语法高亮（2026-09-11）
> **缺口**：办公板块聊天代码块一直是纯 `<pre><code>` 零着色（lang.ts 头注「no highlight.js dependency」是当年留的缝），对标 ChatGPT/Claude 类产品输出观感的最大可见短板；styles.css 的 hljs 令牌主题（`.hljs-*` 十组规则 + `--hl-*` 八色）**早已预铺却一直没有生产者**。**落地**（懒加载缝，入口体积零增）：① 新依赖 highlight.js ^11.12.0 + 新 `gaea/lib/codeHighlight.ts`——core+22 语言白名单（go/bash/sql/yaml/powershell/ts/py/c 系/杂项）全走动态 import 独立 async chunk（mermaid 566KB 同款先例；构建审计 `registerLanguage` 入口 chunk 0 命中）；未注册语言回退纯文本不硬造；围栏语言别名归一复用 lang.ts ALIASES（编辑器/工具卡同源表），toml 无官方语法借 ini 着色（展示标签不变）；hljs 产出（自带转义，仅 hljs-* span）插入前再过统一消毒层 DOMPurify 兜底。② Markdown.tsx 代码块接 HlCode：首帧纯文本立现、高亮就位整体换 HTML；调用点=稳定分段（MemoMarkdown 段签名缓存），流式增长尾部走简易 HTML 不经此处——无逐 delta 重高亮。③ 明暗双调色板：暗色=既有 One Dark 系（styles.css :root 兜底不动），浅色 `:root[data-hl="light"]` 同色相深阶（白底对比度 ≥4.5:1）；data-hl 由主 App 主题 effect 内联挂载（**刻意不 import gaea/lib——sanitize 链会拖进主入口 chunk**；darkMode 为 store 派生实际明暗，system 档跟随 OS 实时切换）。**测试**=vitest +11（codeHighlight 9=归一/别名/明示纯文本与未知回退/toml→ini/go·py 着色/script 转义不成活/data-hl 挂钩；Markdown 2=go 块异步着色+首帧纯文本先行/未知语言不硬造）。**坑**=hljs 类名是运行时 `hljs-`+scope 拼接，产物 JS 无 "hljs-keyword" 字面量，bundle 归属审计不能 grep 类名，用 `registerLanguage` 特征串。**门禁**：tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.230.0。**欠账**：白名单外语言（kotlin/php/ruby 等）纯文本降级按需扩 REGISTER；重新生成/点赞点踩等消息级交互属交互面不属输出渲染，按反馈另刀；ChatPage 伴侣线自有渲染链（ChatMarkdown/MarkdownContent）本刀不动。

## v4.229.0 · 任务表会话维度结构刀 + gongwen 技能页码/红头字色检测（2026-09-11）
> 并行双线一版销两项（文件面零交叠）。**线 A=任务表会话维度（v4.180 既定欠账「TaskCenter 会话关联需任务表加 session 维度」）**：**SchemaV20** `tasks.session_id`（ADD COLUMN NOT NULL DEFAULT ''，旧行零成本回填；迁移链 V19 终点+1 先查后定）；Task.SessionID（`json:"session_id,omitempty"`）+ `SubmitSpaceSession`（Submit/SubmitSpace 签名不变委托，INSERT/5 处 SELECT/scanTask 全链带列）。**诚实口径**：全仓扫描 6 个任务创建点（文件索引×3/价格抓取×3）全为 cron/设置类、无会话上下文，子代理运行与 DAG 节点是独立数据系统不写任务表——**全部如实留空，停在能力层不造假接线**；前端 TaskCenter `sessionPath`「本会话/全部」过滤面（渲染层 useMemo 过滤零请求、事件增量天然生效）+三语键+4 用例，**chip 暂不显形**（TasksWorkbench 不传 sessionPath，等首个会话上下文创建点再点亮，避免恒空过滤的半成品面）。零新绑定 624。**线 B=gongwen-9704 技能余项（v4.223 立项「页码与红头字色需页脚/字色层，按需另刀」）**：docx_layout.py 增页脚解析（`word/footer*.xml`+rels 反查归位 ref_types；PAGE 字段用 `\bPAGE\b` 防 NUMPAGES 误认；页脚字号+一字线文本）与字色采集（run 级 w:color 按值聚合+最大字号+样例，`auto` 不采），顶层新增 `footers`/`colors` 旧键零动、缺数据=空数组不报错；SKILL.md 排版细则表 6→8 项（页码=4 号半角宋体一字线包数字、红头字色=红色系大字；判定纪律=版心位置/单双页空一字取不到注「不适用」不误报）；新增 test_docx_layout.py（stdlib zipfile 现场构造 docx 夹具）4 用例全过。规范知识全程留技能侧零内核改动（红线）。**顺带**：AGENTS.md 五迁（v4.208~v4.203 四条 4770B 入 archive，回落 WARN 线下 56.5KB）。**门禁**：ci.ps1 全绿/绑定 624 零变更/版本三处 4.229.0。

## v4.228.0 · 办公板块输出列左对齐 + 存量 lint 清障（2026-09-11）
> **缺口**：办公板块对话输出列 `.v3-reading` 基础规则 `margin: 0 auto` 整列水平居中——前次「通用办公重设计」只把列宽覆写到 `--maxw`、**没去 auto margin**，宽屏下输出仍悬在主区正中，与工作台/产物面板/轨迹等满宽左对齐区域不一致（用户反馈「输出部分还是居中」）。**落地**（全部 `.gaea-app-layout` 作用域覆写，零删基础规则）：① `.gaea-app-layout .v3-reading` 去 auto margin——输出列 `--maxw` 宽度贴左缘；② `.composer-glow` 去 auto margin（该类仅办公 App 使用；进度计划的输入框居中来自 Composer 内部 `mx-auto`，不受影响）——输入框与输出列同一左缘；③ `.gaea-app-layout .phase` 阶段小标左对齐（**坑**：SchedulePage/MemoryHubPage/CostLibraryPage 都导入 gaea/styles.css，共享规则必须作用域限定，`.phase` 基础居中保留给进度计划的 Claude 式居中流）；④ Transcript 注释同步。**顺带清障**：`schedule/monte.ts` 存量 `prefer-const` error（v4.217.0 入库，此后发布未跑全量 lint 故未暴露）`let idx`→`const idx`。**测试**=目检前后对照（真实 CSS+还原 DOM 静态挂载，Edge 无头截图：before 输出列/输入框居中悬空，after 同一左缘对齐工作台）；vitest src/gaea 184 文件 1541 全绿。**门禁**：ci.ps1 全绿（go build/vet/test + eslint + vite build + vitest 全量 + E 系守卫 + 仓库卫生）/绑定 624 零变更/版本三处 4.228.0。

## v4.227.0 · 规范知识出内核第四刀：造价判定参数数据化（拍板池 §5.5 形态 a）（2026-09-11）
> 红线第四落地（造价域；拍板池 §5.5 选形态 a=**Apply 确认留痕流不动**，仅行业判定参数出码）。**缺口**：组价校验的行业判定参数（R1 金额容差 1%/绝对下限 0.01、R3 超推荐价 1.05/低于 0.5、含量对照最少样本 3/退化带容差 5%）硬编码在 composecheck.go/contentband.go，不同地区/企业口径想调就得改代码发版。**落地**（校验引擎=机制留码，判定参数=知识出码）：① `internal/gaea/cost/checkparams.json` go:embed（六参数内置默认，genui/novelstyle 先例）；② `params.go`：快照 atomic 换出 + `LoadCheckParams` **部分字段覆盖**（数值参数逐项覆盖比整表替换语义自然；未覆盖保默认；≤0 字段忽略，坏 JSON 显式报错不带病运行；缺失文件静默）；③ 引擎接线：composecheck R1/R3 与 contentband 最少样本/退化带全走 `currentCheckParams()`——**CheckComposeComponents/CheckContentBaseline 签名零变更**（零 app 流改动，确认留痕语义不变）；`MinContentSamples` 导出常量删除防双源漂移；④ app `ensureCostCheckParams` 每进程懒加载（composeChecks 入口；惯例路径 `.gaea/skills/cost-compose/params.json`，.gaea>.agents>.agent>.claude）；⑤ 技能文档 `.gaea/skills/cost-compose/SKILL.md`（R1-R3+含量带判定口径表×JSON 字段映射+覆盖指引+判定纪律=warn/info 语义、已留痕结论不受参数变更影响、带内静默）。**测试**=Go +2（默认表漂移守卫逐字段；覆盖全链=部分覆盖生效+未覆盖保默认+R1 容差抬升 3% 偏差翻转不触发+minContentSamples=5 后 3 例池静默+≤0 忽略+坏 JSON 报错+缺失静默+cleanup 恢复内置）。**门禁**：Go cost+app 全量 0 FAIL/绑定 624 零变更/版本三处 4.227.0。**拍板池清空（§5.4/5.5 全销）——「规范知识出内核」审计线至此收官**：四刀三形态（Go 植入撤销转技能 gongwen-9704 / Go 词表数据资产 novel-deslop / 前端阈值资产 schedule-dcma / Go 参数资产 cost-compose）；遗留观察=standard 包 Redhead/CostTable 文本级 checker 同属规范在码，迁移低优先挂观察池。

## v4.226.0 · 规范知识出内核第三刀：DCMA 阈值表数据化（拍板池 §5.4 形态 a）（2026-09-11）
> 「规范知识→数据资产」红线第三落地（进度域；拍板池 §5.4 选形态 a=质量体检视图保留、阈值表出码——不退任何已发布产品面）。**缺口**：DCMA 14 点的五个行业标准阈值（高浮 44 天/长工期 44/超标占比 5%/FS 占比 90%/CPLI·BEI 0.95）硬编码在 dcma.ts，不同业主/机构口径想调阈值就得改代码发版。**落地**（计算=机制留码，阈值=知识出码）：① `frontend/src/schedule/dcma-thresholds.json`（默认表随构建内置，resolveJsonModule）；② `dcmaAudit` 增第 4 参 `thresholds?: Partial<DcmaThresholds>`——**部分字段覆盖**，缺省字段用内置默认；判定线与体检单阈值展示串全部走生效值 th（覆盖后 UI 口径同步）；③ 兼容导出 DCMA_* 常量保留（默认值派生，历史消费方零破坏；顺带清 v4.216 起的 naCheck 死函数=eslint warning 清零）；④ DcmaView 挂载时经 ReadFile 懒加载覆盖文件 `.gaea/skills/schedule-dcma/thresholds.json`（部分字段即可，缺失/解析失败静默用内置默认）；⑤ 技能文档 `.gaea/skills/schedule-dcma/SKILL.md`（14 点×默认阈值对照表+字段映射+覆盖指引+诚实口径=CPM 断裂 score null/n/a 不计分）。**测试**=vitest +2（默认表漂移守卫逐字段；覆盖全链=长工期阈值抬升判定翻转+展示串同步+未覆盖字段保默认+超标占比归零）。**坑**=①ts 文件全局 perl 替换会把兼容导出行一起改坏（DCMA_RATIO_LIMIT→th.ratioLimitPct 出现 `export const th.x` 语法错），批量替换后必须 tsc 立验；②覆盖 ratioLimitPct 后断言「未覆盖字段」不能挑被覆盖字段（自相矛盾）；③实测占比 value 随阈值变化，不能断言前后相等。**门禁**：tsc -b 0/eslint 0/vitest 329 文件 2847 例全绿/绑定 624 零变更/版本三处 4.226.0。**拍板池余项**：§5.5 造价 R1-R3/含量带判定口径技能化（内嵌 Apply 留痕链路，涉确认流语义）。

## v4.225.0 · 规范知识出内核第二刀：小说 AI 味词表外置为可覆盖数据资产（2026-09-11）
> 「规范知识→技能/数据资产」红线的第二处落地（审计后首刀，小说域）。**缺口**：AI 味词表（27 组替换映射 + 19 词黑名单）硬编码在 novelstyle 的 Go 源里，用户想加一个词就得改代码重发版。**落地**（引擎=机制留码，词表=知识出码）：① `internal/novelstyle/words.json` go:embed（数据与逻辑分离，genui handbook 先例）——默认表随构建内置，开箱即用；② `words.go`：词表快照 atomic 换出 + `LoadWordsFile` 覆盖 API（**整表替换不合并**，语义可预测；覆盖文件 `.gaea/skills/novel-deslop/words.json`，与 gongwen-9704 同惯例目录，.gaea>.agents>.agent>.claude 优先序）；③ 引擎三处接线：rewrite 替换表/segment 分词词表（**vocab 改原子快照可重建**——tokenize 无锁读旧快照安全）/score 规则 9 黑名单，全部走 `currentWords()`；④ app `ensureNovelStyleWords` 每进程懒加载一次（生成去味 + 手动一键去味两个入口；覆盖文件缺失/解析失败静默保内置默认=增强面不挡主功能）；⑤ 技能文档 `.gaea/skills/novel-deslop/SKILL.md`（机制三层说明+改词表指引+判定纪律「宁漏勿误」）。**测试**=Go +2（words.json 漂移守卫 27/19 断言；覆盖文件全链=替换生效+黑名单进打分+分词重建+缺失静默；cleanup 恢复内置防同包污染）。**坑**=①词表快照持有者必须包级变量初始化（先于一切 init——segment 的 vocab 构建在 init 里读它，init 函数间顺序不可依赖）；②app 包 config 名被 internal/config 占用（双 config 坑再证），gaea/config 需别名 gaeaconfig；③ConventionDirs 变量 .gaea 在首位即优先序，无 ConventionDirsAt 函数。**门禁**：Go 全量 65 包 0 FAIL/绑定 624 零变更/版本三处 4.225.0。**审计余项见拍板池**：DCMA 14 点形态、造价 R1-R3 技能化——涉产品面变更待拍板。

## v4.224.0 · 架构修正：规范知识出内核——GB/T 9704 排版细则改技能按需加载（2026-09-11）
> 用户拍板的架构红线（本刀即刻执行）：**「公文写作规范这类领域知识不应植入代码，应以 skill 技能按需加载，防止 gaea 臃肿」**。内核只留通用机制，规范知识一律走技能包。**动作**：① **撤销 v4.223 的 Go 侧植入式 checker**（删 internal/office/standard/format.go+test；registry 回退 LintDocument；gaea_lint.go 回退 layout 接线；docxedit.ReadDocumentXML 无消费者一并撤）——解析/检查逻辑整体出内核；② **改为技能包 .gaea/skills/gongwen-9704/**（与 docx/pptx 技能同域同形态，按需加载零内核成本）：SKILL.md 承载全部规范知识（红头要素表七项+排版细则表六项+换算备查+判定纪律=容差内即符合/不适用≠缺失/实测值进建议/宁缺勿误），scripts/docx_layout.py（stdlib zipfile+ElementTree）承载排版事实提取——解析也是技能资产非内核代码；已实测最小 docx 取数正确。③ 红头要素/造价表式两个既有 Go checker 同属「规范在码」，列观察池候选，迁移与否待拍板（涉及既有 GaeaDocumentLint 行为变更，不随本刀擅动）。**门禁**：Go 全量 0 FAIL/绑定 624 零变更/版本三处 4.224.0/技能脚本最小 docx 实测通过。**后续铁律**：领域规范/行业模板/检测规则类需求一律先问「能否做成技能」，代码只进通用机制。

## v4.223.0 · 办公·中文文书规范包第二刀：GB/T 9704 排版细则 lint（2026-09-11）
> 板块并行池开工（roadmap 办公#5「中文文书规范包做成可校验 lint+模板」；并行池非阶段旗）。**缺口**：v4.1c 红头要素只查文本层（发文机关/发文字号/日期/盖章…），GB/T 9704-2012 的**排版层**（页边距/字体字号/行距/缩进/层级标题字体）无人管。**落地**=internal/office/standard 新增 FormatChecker（「GB/T 9704 排版细则」规范包，注册表机制自然聚合，**零新绑定零前端改动**——LintReport 走既有 GaeaDocumentLint/前端规范包分组）：① ParseDocxLayout 纯函数从 word/document.xml 提取排版事实（pgMar 四边/逐段东亚字体/字号半磅/行距 line+lineRule/首行缩进 firstLineChars/对齐；容错子集解析，run 级 rPr 优先段级缺省）；② 六项检查——页边距 上37 下35 左28 右26 mm（±2mm 容差聚一条）、公文标题字号二号 22pt（小标宋字体名变体多只作建议不判定）、正文字体字号三号 16pt 仿宋（多数票）、行距 28~30 磅固定值（多数符合即符合）、首行缩进 2 字符（多数票）、层级标题「一、」黑体「（一）」楷体（字体名缺失不判宁缺勿误）；数据不足/要素不适用=「不适用」注记不误报为缺失，违规项实测值写进修复建议。**取数口**=docxedit 新导出 ReadDocumentXML（薄包装）；gaea_lint.go .docx 分支解析失败 fail-open（排版包跳过，文本层检查器照常）。**测试**=Go +4（真实 XML 片段解析事实/合规 6 项全过/违规六项带实测建议/数据不足不适用+无边距判缺失）。**门禁**：Go 全量 0 FAIL（纯 Go 刀，vitest 以 v4.222 基线为准）/版本三处 4.223.0/绑定面零变更。**规范包余项**：页码（需页脚 XML）与红头字色/盖章位图形检测（需 drawingml 层）按需另刀；「模板」半边（GB/T 9704 公文 docx 模板内置）另刀。

## v4.222.0 · 6.3 余项：dag_plan 增量改图——run_id 编辑模式原地调和（2026-09-11）
> 设计 docs/gaea-office-dag-63-design-2026-09.md §6 又清一条（纯 Go 零绑定零前端改动）。**缺口**：改单节点指令（「把报告节点换成季度口径」）也要整链重新规划——run id 换、既有状态与验收全丢。**落地**：dag_plan 增 `run_id` 可选参=编辑模式，核心=internal/gaea/dag `ApplyEdit` 纯函数（新图对既有 run 原地调和）：节点按 id 对账，title/prompt/dependsOn 全等→**原样保留**（状态/产物/运行痕迹延续）；有变或新增→回 pending 全新节点；运行中拒绝（先终止再改）；新图必须自身合法——被移除节点仍被保留节点依赖=悬空依赖 fail-closed，不静默断链；**改已验收节点=撤销验收**（产物是旧指令的产出，新指令下不再成立），RevokedAcc 计数透出人读文案，不静默。工具描述同步注明两态语义。**测试**=Go +2（dag：调和四态=保留/变更/新增/移除+撤销验收+运行中/成环/悬空三守卫；app：编辑路径全流程=跑一轮→改图→状态断言+run 不在场/运行中/悬空拒绝）。**门禁**：Go 全量 0 FAIL（纯 Go 刀，vitest 以 v4.221 全绿基线为准）/版本三处 4.222.0/drift@624 零变更。**§6 余项剩两条**：运行中 steer 直穿+危险操作分级审批（等审批闸分级面）/产物登记 DeliverableRegistry（现侧通道够用）——**DAG 线自然收尾**。

## v4.221.0 · 6.3 余项：子代理证据落账 + 按会话归因 + 波内并行（2026-09-11）
> 三合一刀（设计 docs/gaea-office-dag-63-design-2026-09.md §6 首条清掉）。**起因=真机归因链核查发现结构性缺口：子代理写盘从不落 Journal**——子代理 Options 不带 JournalDir/SessionID，flushJournal 直接跳过；v4.219 的「窗口归因」在真机上恒为空集（fake-runner 测试靠手工塞卡绿，壳内真机走查恰挂观察池未跑）。**① 子代理证据落账**：TaskTool 增 journalDir（SetSubagentJournalDir，boot 注入与主执行器同目录）+ Options.SessionID=run.Ref，经 runSubSession 下发——task 工具/RunNew/RunFollowUp 三路径全覆盖；子代理写盘回合收尾落证据卡、按会话分文件（sa_ ref 名下）；顺带收口「task 子代理编辑对版本时间线/回滚不可见」的既有审计缺口（6.1 可审计默认化同向；子代理因此开始建回滚基线）。**② 按会话归因**：DAG 节点 outputs=SessionID==ref 的证据卡 Target（精确归因，主对话同期写盘不再并入；ref 空=ephemeral 回退窗口增量口径）；UI 口径注三语同步。**③ 波内并行**：归因不再依赖窗口不重叠，dagExecute 波内 goroutine 并发起跑（失败级联/终止级联/互斥落盘语义不变）。**测试**=Go +3（agent：落账往返+SessionID=ref+二次运行独立会话文件；app：独立根同波互等信号证并发+并发窗口下会话归因不串；生命周期 fake-runner 改模拟真实落账形态）。**门禁**：Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.221.0。**§6 余项剩三条**：运行中 steer 直穿+分级审批（等审批闸分级面）/dag_plan 增量改图/产物登记 DeliverableRegistry。

## v4.220.0 · 6.3 余项：流水线模板库——「月度报告」存模板一键重建（2026-09-11）
> 阶段六收官后续刀（设计 docs/gaea-office-dag-63-design-2026-09.md §6 余项之二）。模板只取**图形状**（goal+节点指令+依赖），状态/产物/ref/运行痕迹一律剥净：internal/gaea/dag/template.go 纯函数包（FromRun 剥痕迹+空名回退 goal 截断 24 字/Instantiate 全新草稿 run/TemplateStore 落 `.gaea/work/dag/templates/`——run 档同域子目录，Store.List 只读顶层 *.json 互不混；Save 前全量 Validate=成环/悬空依赖拒入库，删档即弃）。**绑定 620→624**（GaeaDagTemplateSave/List/New/Delete，Office 门面）：一键重建=模板→全新草稿 run，**不自动起跑**（起跑仍是人拍板，与整链首跑同闸）；重建即模板当前形状快照，改模板不影响已重建 run；删除只删模板档。前端 DagPanel 模板区（折叠条默认收起，有模板才显形）：逐条「新建」=一键重建、「删」=删档即弃；run 卡「存模板」→内联输入（默认带出 goal 截断，可清空交后端回退）。?mock=1 预置「月度经营报告」模板可走查。测试 Go +6（template_test 剥痕迹/空名回退与限长/实例化全新/存储往返+坏形状拒入库+穿越拒绝/列表倒序+与 run 档隔离；app 层 TestDagTemplateFlow=存→列→重建→重建链 fake-runner 跑通全 done→删→再删显式报错）、vitest +4（模板区显隐折叠/新建重拉/存模板内联输入提交/删模板+拉取失败静默不挡主列表）。**门禁**：Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.220.0。§6 余项剩：波内并行（等按 ref 归因定案）/运行中 GaeaSteer 直穿+危险操作分级审批（等审批闸分级面）/dag_plan 增量改图/产物登记 DeliverableRegistry。

## v4.219.0 · 阶段六 6.3 首刀：办公多文件 DAG——委托式文件流水线（2026-09-11）
> 阶段六收官刀（办公#3「多文件任务图（DAG）编排」；对标 roadmap §12.4「任务中心可视化为可插话的 DAG」）。设计基线 docs/gaea-office-dag-63-design-2026-09.md（§6 列余项：波内并行/运行中 GaeaSteer 直穿/危险操作分级审批/DAG 模板库）。**架构五层**：dag_plan 规划工具（work 空间，PersistWrite 标记=子代理注册表自动剔除防嵌套派生）→ 存储 .gaea/work/dag/<id>.json（workspace 本地与 journal 同域，删档即弃）→ 执行器（拓扑分波、波内顺序=产物归因窗口单调不重叠；每节点=一次全新 TaskTool.RunNew 子代理会话：headless 闸+过滤工具集+transcripts 落盘，自动进既有子代理树/tab，文本增量走 gaea-subagent-text 与追问同路）→ 控制 7 绑定（GaeaDagList/Get/Run/NodeRun/NodeSteer/NodeAccept/Cancel，613→620，Office 门面）→ 前端 TasksWorkbench「办公流水线」区（run 卡+节点卡：状态点/产物 chips 点击进预览/steer 输入框/重跑/验收；running 轮询自校正；?mock=1 带示例 4 节点链可走查）。**关键口径**：节点产物=运行窗口前后 Journal 证据卡 ID 集合差、Target 过滤工作区路径（主对话同期写盘会并入，UI 口径注诚实不造精确归因）；验收=人拍板→hubOfficeStore.Save（save 路径自动落 memory_events 成 5.1 图谱实体，6.2 项目本体自然吃到），重跑把 accepted 退回 done=验收失效诚实降级；终止级联（GaeaDagCancel→ctx 贯穿子代理→未起跑 skipped）；runner 未接线/依赖成环/悬空依赖/空链全部 fail-closed；应用重启 running 懒清扫=诚实置 failed。**测试**：Go +6（dag 包校验四态/分波确定性菱形+子集/派生态五判序/中断清扫/存储往返+穿越拒绝+SaveNew 防覆盖；app 层 fake-runner 3 节点链全生命周期=顺序+产物归因+上游产物进下游 prompt+验收回流记忆断言+重跑降验收 RunCount=2/上游失败下游 skipped/Cancel 级联/未接线 fail-closed/中断懒清扫/steer 三守卫/dag_plan 校验拒绝）、vitest +9（DagPanel 空态/首拉失败重试/按钮显隐矩阵/验收/起跑/steer/预览/轮询起停）。**门禁**：Go 全量 0 FAIL/drift PASS@620/tsc -b 0/eslint 0/vitest 329 文件 2840 例全绿/版本三处 4.219.0。**修复入库前既有漂移**：spaceBindings.test 锁 433 实为 435（v4.215 +2 漏更，satisfies 编译期真值兜底为准）本刀顺带归正 442=435+7。**判据「一条 ≥3 节点链全节点可控（跑/重跑/steer/验收）」满足——阶段六 6.1~6.6 出口判据全满足，阶段六收官**（规划 §3 旗）。

## v4.218.0 · 阶段六 6.6 首刀：跨域 EVM——进度×造价挣值三数两指数（2026-09-11）
> 6.6「三数+两指数出且口径注全」首刀（施工债级优先可穿插；gaea 独有跨域：进度+造价同在本机一个库）。TS 纯函数 computeEvm（evm.ts，零绑定零 Go 改动）：PV=任务预算按基线起止线性分摊到数据日期（里程碑基线完成点整额兑现）；EV=任务预算×完成率（progress）；AC=手工录入（来源口径诚实标注，缺失→CPI/CV n/a 不伪造）；SPI=EV/PV、CPI=EV/AC、SV/CV 偏差同出；BAC=任务预算合计（固定+资源分配，复用 computeCosts v4.122 成本 rollup，CPM 未过 fail-closed 同口径）；evTop=挣值来源 Top5。无基线/CPM 断裂 ok=false 带原因。进度页新增「挣值分析」视图七档（三数大字+两指数+SV/CV+口径注列表+EV Top5；数据日期与 AC 录入面）。测试 vitest +6（PV 分摊/EV-SPI-SV/落后场景/CPI 两侧/里程碑整额兑现/无基线与循环依赖 fail-closed）。**阶段六 6.1/6.2/6.4/6.5/6.6 判据全满足，余 6.3 DAG 一刀**。

## v4.217.0 · 阶段六 6.5 首刀：进度·蒙特卡洛工期带（2026-09-11）
> 6.5「三档+稳定度本地可跑（无云依赖）」首刀（nPlan 对位，复用造价 P25-P75 哲学：「工期也给你三档」）。TS 纯函数 monteCarlo（monte.ts，零绑定零 Go 改动）：种子化 RNG（mulberry32，同种子同结果跨会话可比）+ 三角分布采样（[d·(1−u/2), d, d·(1+u)]，默认不确定度 30%=工期偏乐观的不对称先验，可调 0~60%）+ 每次模拟对采样后计划重算 CPM（默认 200 次，planFinish/日历与页面同语义）；输出 P25/中位/P75 三档+min/max/均值+直方图桶+关键路径稳定度（逐任务关键出现频率降序+当前关键任务平均频率=稳定度指标，换线风险可见：当前关键但频率低=容易被顶掉）。SchedulePage 新增「工期模拟」视图（三档大字+直方图+频率条+不确定度滑杆，菜单/Segmented/侧栏六档）。测试 vitest +8（确定性/三档单调/零不确定度退化/线性链稳定度=1/并行链换线频率<100%/直方图频数守恒/CPM 断裂诚实失败/RNG 序列）。

## v4.216.0 · 阶段六 6.4 首刀：进度·DCMA 14 点计划质量体检（2026-09-11）
> 6.4「DCMA 14 点逐项+总分，纯函数，吃现有 CPM+多基线数据」首刀（对标 Deltek Acumen Fuse）。TS 纯函数 dcmaAudit（frontend/src/schedule/dcma.ts，吃 SchedulePage 前端自算的 CPM 与基线——与 deadline/drift 同一镜像口径，零绑定零 Go 改动）：缺逻辑（首末豁免收紧=孤立任务不得因 es 并列豁免）/负搭接/正搭接/FS 占比/硬约束（模型无约束字段恒过+note 诚实标注）/高浮时/负浮时/长工期/无效日期/资源缺失（未启用资源维度= n/a）/错过基线/关键路径连续性（连通分量计数）/CPLI/BEI（后两者需基线+数据日期，progress≥100 作完成口径的诚实近似）14 点逐项，超标样本点名上限 5，每点带阈值与口径 note；通过率评分在 CPM 断裂时置 null（整体分不可信，逐点结论仍在）。SchedulePage 新增「质量体检」视图（菜单+Segmented+侧栏口径，DcmaView 纯展示）。测试 vitest +11（十四点逐形态：健康基线/孤立任务/负正搭接/FS 占比/高浮长工期/负浮时/资源 n-a 与点名/错过基线/单链双链/CPLI-BEI n-a 与可评估/循环依赖全 n-a+score null/阈值常量），SchedulePage 菜单用例同步五档。判据「十四点纯函数全测」满足。

## v4.215.0 · 阶段六 6.2 首刀：记忆驱动项目本体——项目本体注入（2026-09-11）
> 6.2 出口判据「Plan 注入带决策引用且可关闭」首刀（依赖 5.1 图谱底座已就位）。BuildProjectBrief 纯函数把记忆投影成项目事实表：固化全收+project/feedback 按衰减评分降序，每行带 [MEM:] 稳定引用键（采纳→句末引用→触达/徽标同源）+决策来源归因（「依据 session-x 会话 · turn N」），600 rune 预算截行不截半句，无候选零注入。注入点=work 空间会话装配（晨报预载同位，缓存稳定前缀，跨会话项目连续性），BootOptions.ProjectBrief 穿管（CLI/TUI 不读配置维持原行为）。可关闭=config project_brief 键（默认开）+GaeaMemoryBrief/GaeaSetMemoryBrief 绑定（写后重建即时生效）+MemoryPanel「项目本体」胶囊。绑定 611→613。测试 Go +2 / vitest +1。

## v4.214.0 · 阶段六 6.1 首刀：可审计默认化——文件预览默认版本条（2026-09-11）
> 阶段旗交阶段六（书斋纵深·办公主角），6.1「Verify→Journal→一键回滚从能力升为文件工作台默认 UI」首刀。纯前端、零新绑定、零 Go 改动。缺口：VersionTimeline（v4.28 B1 逐版本预览/恢复）此前只藏在交付面板的 vN 徽标里。落地=FileVersionStrip 新组件默认可见地挂进文件预览标题栏下（三件套通用）：版本数+最近 AI 改动时间+复核状态+「可恢复」标注，无版本时显示「AI 改动会自动成为版本点」说明条；展开为该文件的逐版本时间线（按路径过滤的 VersionTimeline：预览基线/一键恢复，恢复=新版本不丢历史，恢复后父级静默重读预览）。pptx 页码归因来自证据卡摘要（p3 格式）。「AI 改动自动成版本点」=证据链既有能力（v4.1 起 Apply→Journal+基线快照）。测试 vitest +4（默认可见/展开时间线/恢复贯通 RollbackRecord+onRestored/空态与降级）。6.1 判据满足。

## v4.213.0 · 5.3 首刀：记忆生命周期三态（固化/衰减/归档）（2026-09-11）
> 阶段五 5.3 首刀。「90 天一刀切」的替代：固化（SchemaV19 facts.pinned）=用户明示保留——免疫衰减归档、**豁免保留期硬删**（CleanupArchived 跳过 pinned，替代核心）、晨报/预载排序加权，与 archived 正交、Save 覆盖不丢；衰减=纯函数评分（DecayScore 半衰期 30 天指数、下限 0.01、固化恒 1、无时间戳不造数；LifecycleOf 阈值 60 天），评分输入（save/touch）自 v4.210 全在 memory_events、状态动作（pin/unpin）同日志留痕；归档=既有保留期+purge-audit。三态可查=GaeaMemoryLifecycle 总览（固化/衰减列表带评分与闲置天数）+ GaeaMemoryPin 切换；前端 OfficeMemoryLibrary 统计行 + FactCard 固化锁/「固化」徽标。「蒸馏 no-op 转真实合并」（DistillMerge 做梦 2.0）与「预取可关闭」（晨报预载开关）此前已落地不重建。绑定 609→611。测试 Go +5 / vitest +1。**5.3 出口判据全满足**。

## v4.212.0 · 5.2 首刀：上下文编译 dump/diff + 前缀稳定证明（2026-09-11）
> 阶段五 5.2 首刀（纯 Go，零绑定，前端零改动）。意图分类与预算调度底座已有（TCCA 四层内核 L3 SkillLayer.Route=意图分类；doc 预算/compaction/tail 注入=预算调度），本刀补**证据链**缺口：CacheShape（V5.10）只看 system+tools 头部，消息历史的前缀稳定性无人判定。新增 agent 消息级编译摘要（canonical role+长度前缀+content+toolcalls 的逐消息 SHA256 链）+ DiffMessages 相邻请求判定（Stable/PrefixBytes/AppendCount/RewriteCount，>0=压缩/rewind 归因）；判定随 RequestHeader 事件落会话日志（追加列）——日志即装配 dump、相邻记录即 diff、ReadEntriesFor 即回放；contextview 趋势柱携带 RequestRecord.Prefix 与 CacheHitTokens 配对=「相邻两轮前缀字节级稳定（缓存命中可证）」完整证据链。测试 Go +4（纯函数五形态+拼接歧义防护/同会话 4 请求跨 2 回合全稳定+摘要链接续/改写诚实判漂移+RewriteCount 归因/contextview 折叠贯通+旧日志 nil）。前端渲染判定按需另刀。

## v4.211.0 · 5.1 收口：回复发出前剥离悬空 [MEM:] 引用（2026-09-11）
> v4.210.0 的 5.1 余项收口刀（纯 Go，前端零改动）。闸点=agent stream() 收尾：`Options.FinalizeText`/`SetFinalizeText` 定稿钩子在 Message 全文事件发出前改写答案文本，改写值同时进 session 历史与 TurnResult 摘要——模型幻觉的引用键到不了持久层，前端收 Message 事件整泡替换（既有语义），流式增量原样透传由重渲染收敛。boot 按记忆总闸+库可用性注入剥离闭包（`memory.Store.StripDanglingCitations` 纯函数：命中键保留、悬空键含紧邻空白整处剥离、**不 Touch**——触达职责仍在回合收尾 touchMemoryCitations；被剥离键当场落 dangling cite 事件，v4.210 的悬空可观测性不丢）。子代理/headless executor nil=不改写。空间语义同读端隔离器（work 会话剥 play 键）。测试 Go +5（memory 三例：命中保留+悬空剥离+空间隔离/不触达+零值 Store 跳过；agent 两例：Message/Summary/session 三路改写+nil no-op）。**5.1 出口判据至此全满足**，阶段五下一刀=5.2 上下文编译。

## v4.210.0 · 记忆语义图谱：事件日志投影出图（2026-09-11）
> 阶段五「记忆OS+上下文编译」首刀（规划 gaea-next-stage-plan-2026-09 §2，底座#1）。记忆此前只有存储没有结构：新增 memory_events 事件日志（SchemaV18，追加式，一切记忆写路径——remember 工具/桌面面板/做梦/蒸馏合并/引用触达——落库成功即留痕），ProjectEvents 纯函数投影出实体/事件/来源三向图（produces/affects/references），物化进 mem_graph_*（含 embedding 向量占位列，5.2 启用）。日志即真相：图可删可重建（读前水位对账懒重建，物化=投影逐字段一致有测试锁死）。[MEM:name] 图节点寻址；悬空互引不入图但 Dangling 留痕不静默；回合收尾引用解析（命中+悬空）落 cite 事件。绑定 608→609（GaeaMemorySemanticGraph）；MemoryHub 图谱页新增「关联图/事件图谱」切换（GraphView source prop，默认 cloud 旧行为零变化）。Go +9 用例（确定性/三向边/悬空拒写/删库重建/BFS 遍历/写路径挂钩/cite 事件），vitest +1（semantic 数据源切换）。设计基线：docs/gaea-memory-graph-51-design-2026-09.md（5.1 余项=真·发送前悬空引用剥离，需流式层另刀）。

## v4.209.0 · 组价含量对照：拆解含量 vs 同类条目分位带（2026-09-10）
> 造价刀路池 §6 收官（survey 六项全清）。AI 拆解的人材机含量此前只有金额自洽校验，含量本身离不离谱没人管——含量 420kg 对同类 300kg 也照样过。新增 `cost.CheckContentBaseline` 纯函数：以相似条目池的同一资源含量分布（标题+单位归一精确匹配）做 P25/P75 分位带对照，带外 warn（同值退化带 ±5% 容差）、带内静默、样本 <3 不比对、单位一方缺失宁缺勿误。接进 `GaeaCostCompose` Checks 与金额自洽结论同列，留痕自动承载。零新结构零新绑定（608 不变）、纯 Go、前端零改动。外部「行业含量区间」数据源明确不引入——基线源=用户自己的库。Go 全量 116 包 0 FAIL（+5 用例）。详见 releases/v4.209.0.md。

## v4.208.0 · 首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」（2026-09-10）
> v6 的廉价感来自「等大圆角卡片 + 描边」无限重复。v7 结构收敛为三种形态（仪表条 / 账页 / 海报墙）+ 四档排印 + 三级色调面，删极光斑改版式记号；信息零删除、契约 testid 全保留。修复重写时遗失的 `.ml-avatar-ai`/`.ml-avatar-user` 规则（气泡左右曾退化为同底色），清掉空挂 `.p-foot-wide`，删除误入的 pnpm 锁/工作区桩文件。附带门禁可信化：CI 由「无条件重试一次」改为「只在册 flaky 隔离复跑」（known-flaky 当前为空 + 分类器自测）、worker 上限跟物理核走、testTimeout 30s。绑定 608 零变更、Go 零改动。详见 releases/v4.208.0.md。

## v4.207.0 · 深色主题硬编码深灰：关系图 overlay + 进度里程碑（2026-09-10）
> v4.206 对称刀。关系图缩放/提示/图例写死 #555/#ddd，深色画布上看不见；进度计划里程碑菱形 #1f2937、标签 #374151 同病。改走 --color-text / on-surface 令牌。绑定 608 零变更。详见 releases/v4.207.0.md。

## v4.206.0 · 浅主色上的字：accent-fg 取代硬编码白（2026-09-10）
> 办公 gaea 主色按钮写死 `text-white`，暗色主题 / 浅强调色时白字糊在浅底上。33 处 `bg-accent text-white` → `text-accent-fg`（= on-primary：暗色深字、亮色白字）；App 补注入 `--color-on-primary` / `--color-surface`。绑定 608 零变更。详见 releases/v4.206.0.md。

## v4.205.0 · 小说书房工坊收口：书架画廊接线 + 阅读检查器可见（2026-09-10）
> v4.198 CSS 已写画廊头/正在编辑横条/书脊，JSX 未接；阅读 tab 把属性检查器 display:none，章节体检入口随之消失。本刀纯前端接线：书架头+正在编辑横条+书脊/角标；阅读保留目录且检查器默认可折叠（章节体检回来）；目录侧栏去掉与头栏重复的身份卡。绑定 608 零变更。详见 releases/v4.205.0.md。

## v4.204.0 · 组价多方案对照：P25/中位/P75 当场切（2026-09-10）
> 造价刀路池 §2 半刀。ComposeModal 三档格可点，应用价随档（默认中位数）；cost_compose 增 mode 参数+三档对照行。不拆逐步盖章。绑定 608 零变更。详见 releases/v4.204.0.md。

## v4.203.0 · 空间策略其余功能域键：总闸当场切（2026-09-10）
> v4.190 欠账收刀。空间策略卡其余六键（对话/轻语/小说/办公文档/角色库/例行）常显可编，与 gaea 主控同款闭环；空键灰字「未配置」，空串清除。纯前端，绑定 608 零变更。详见 releases/v4.203.0.md。

## v4.202.0 · 上下文页按会话读取：时间线/节点详情跟 UI 会话（2026-09-10）
> v4.181 欠账收刀。GaeaContextView/GaeaContextNodeDetail 变参 sessionPath（显式优先/缺省回落内核），ContextView 时间线与 inspector 节点详情按当前 UI 会话读；切会话清详情缓存。绑定名 608 零变更。Go +1、vitest 318 文件 2782 例、tsc/eslint 0。详见 releases/v4.202.0.md。

## v4.201.0 · 办公会话排队：拖拽排序（2026-09-10）
> v4.200 欠账：排队顺序只能入队先后。行左握把 HTML5 拖拽重排发送顺序；sending 钉住不可拖出，pending 可拖到任意槽。点握把不触发撤回编辑。纯前端，绑定 608 零变更。vitest 318 文件 2781 例、tsc/eslint 0。详见 releases/v4.201.0.md。

## v4.200.0 · 办公会话排队：单条/全部插话与停止 + 发送中锁定（2026-09-10）
> 排队项补齐单条/全部 Steer 与全部取消；队首派出显示「发送中」，期间整列不可编辑、删除、插话。队列升级为 id+pending/sending 状态（composerQueue 纯函数）。Esc/停止仍停当前回合并清队列。纯前端，绑定面 608 零变更。vitest 318 文件 2777 例 exit 0、tsc/eslint 0、e-check OK。详见 releases/v4.200.0.md。

## v4.199.0 · 场景元数据编辑面：POV/地点/时间/情感（2026-09-10，DSH 代收后解锁首刀）
> 小说域最后一个有名欠账。SceneMeta 的 POV/地点/时间/情感/标签字段从 v4 场景制起就在、GetChapterScenes 也早回传——唯独没有写路径（SaveScene 只存正文）。**Go SaveSceneMeta**（writingState，play）：线格式与读侧同构（camelCase）、身份字段 ID/Slug/Order 不可变、非法状态整单拒绝、标题空串保持原值、元数据不触发 blob 同步（投影不变有测试锁）。**绑定面 607→608** 七处接线。**前端 ChapterEditor** 每场景 ⓘ 钮 → 元数据弹窗：POV 下拉=本书角色（与 POV 圣经同源，作者显式指定「这一场用谁的眼睛看」）、时间六档、标签逗号分隔、状态三档。Go 全量 exit 0（+TestSaveSceneMeta 五段）、vitest 316 文件 2768 例 exit 0、tsc/eslint 0、e-check OK、drift PASS@608。详见 releases/v4.199.0.md。

## v4.198.0 · 小说工作台「书房工坊」改版 + 规划文档入库（2026-09-10，代收批次）
> 代收发布:DSH 停机确认后由本会话验收其在途批次,门禁独立全跑(tsc/eslint/vitest 316 文件 2768 例 exit 0/e-check 全绿)。**小说工作台改版**:「世界构建工作台」→「书房工坊」——NovelPage 身份头栏(书房 kicker+书名+「N 章·M 字」meta)+模式轨+分区按页显隐;NovelSidebar「世界大纲」收敛为「目录」(图标语义化);CreatePage 动作轨语义 class 化去内联;novel-workspace.css +503 行承载新视觉语言。**品牌资产**四件刷新(appicon/favicon/logo/logo-light)。**规划文档入库**:未来展望契约(含 09-10「两套空间拆开」拍板)+长期规划两份治理权威落库,docs/README 注册,design-system 规范页随更。绑定面 607 零变更、Go 零改动。详见 releases/v4.198.0.md。

## v4.197.0 · office 枢纽解耦：审计收刀（2026-09-10）
> 收敛计划 W3-4 优先刀。**读档修正**：并行子代理全量盘点实证 whisper→office 与 modelengine→office 两条边在 import 层面均为 0（internal/whisper 零 office 引用且间接边 0；internal/modelengine 依赖仅 fileutil/strutil/netclient）——「272 入/70 出枢纽」度量口径过时（疑似把 app 接线层与 gaea 侧误计入 whisper，且内核能力早经 internal/core 下沉、office 根包已是别名门面）。**内核侧真耦合仅 docmd 一族**：6 文件 9 处 100% 文件工具类（refs@引用/fileindex/知识导入/大文件摘要/format_convert/wssearch），零会话记忆、零误用。**收刀**：`internal/office/docmd → internal/docmd` 纯路径搬迁（包名不变，20 行 import 更新零逻辑改动；docmd 仅依赖 proc 无环），internal/gaea 内核对 office import 边清零，office 剩余消费面全部在 app 绑定层（编辑/预览/检查/联动本体）。遗留候选挂观察池：别名门面删除（需动生成签名）、archive.go 死代码候选、依赖度度量应排除 app 层。Go 全量 exit 0（116 包零 FAIL，纯 Go 搬迁前端零改动）。详见 docs/gaea-office-hub-decouple-audit-2026-09.md。

## v4.196.0 · 语义索引显形与显式补齐（2026-09-10）
> 刀路池第 5 项，读档修正刀型：GaeaSemanticIndexStatus（D3-1，v2.18）早已显形各 kind 向量条数，真缺口=覆盖无内容口径+成本类无显式补齐。**Go**：semantic.Store.Coverage（与 Ensure 缺失判定同口径，正文变了算未覆盖）；SemanticIndexStatus 扩展 costTotal/costIndexed（omitempty 旧消费零破坏）；新绑定 GaeaSemanticIndexBackfill（MemoryB，共用 Ensure 幂等，模型未配置拒绝并说人话，10 分钟预算）。**绑定面 606→607** 七处接线。**前端 CostLibraryView** 索引 chip 三态：全覆盖绿禁用/部分覆盖琥珀可点补齐/模型未配置灰「未启用」。**测评集**补 13-15 清单级综合单价查询（表+JSON 同步，零新种子）。Go 全量 exit 0（+2）、vitest 316 文件 2768 例 exit 0（+2）、tsc/eslint 0、e-check OK、drift PASS@607。详见 releases/v4.196.0.md。

## v4.195.0 · 造价询价库级异常扫描（2026-09-10）
> 刀路池第 4 项（survey：异常检测只作用于单点调差建议）。**Go costinquiry/scan.go（只读）**：ScanAnomalies 四类体检全量扫+内存检查，复用既有 MatchTitle/sortableDate/parsePriceDate 零新建基建——①同标题离散（max/min≥1.5 关注/≥2.0 异常）②相邻期跳变（≥30% 关注/≥50% 异常）③有效期已过仍「现行」（逐条）④最新期数超一年（陈旧）；异常优先排序，每条带 severity/人话 detail/refIds。**绑定面 605→606**（+GaeaCostInquiryScan，CostB work）七处接线全同步——v4.195 新踩第七处=bridge mappings.ts gaeaToGaea 映射（drift.ts 双向断言抓的，历史刀只记六处）。**前端 CostInquiryPanel**「⑥ 库级扫描」折叠区：挂载随元数据一并扫+重扫钮+severity 徽标+空态诚实自洽。Go 全量 0 FAIL（scan_test +3）、vitest 316 文件 2766 例全绿（+2）、tsc/eslint 0、e-check OK、drift PASS@606。详见 releases/v4.195.0.md。

## v4.194.0 · 造价五算贯通：阶段值从版本快照带入（2026-09-10）
> 造价刀路池第 3 项（survey：阶段值手填，不从预算版快照带入；规划明确「五算贯通可以」）。**零新绑定**：带入三件套 GaeaCostProjectList/GaeaCostEstimateVersions/GaeaCostStageSave 全部既有，纯前端贯通。FiveCalcPanel 每阶段行新增「带入」钮：选择器项目下拉默认当前项目（五算 projectId 与测算项目同域）可切参照项目，版本列表新→旧（v 号+合计+备注+日期）；确认带入=快照合计走 CostStageSave UPSERT 该阶段+备注自动写溯源「带入:〈项目名〉v〈n〉」+草稿回填+三路刷新+诚实 toast，阶段日期不代填（业务口径≠快照时间）；空态诚实引导（无项目/无版本两态确认禁用）。不做后台自动同步（显式动作，带入后仍可手改）。组价结果带入不做：单条目单价与阶段总量不同量纲，进项目走既有明细引用链路。vitest 316 文件 2764 例全绿（+带入全链/+空态）、tsc/eslint 0、e-check OK、drift PASS@605、Go 零改动。详见 releases/v4.194.0.md。

## v4.193.0 · 角色回写欠账刀：非空字段逐项确认覆盖（2026-09-10）
> v4.192 欠账收口：副本→库回写从「只安全空补」到「预览→逐字段确认→覆盖」两段式。**Go**：characterlib 新增 descFields 描述性字段表（预览/空补/覆盖三路共用，字段集不再各自漂移）+ 只读 PreviewImport（import/fill 计数 + 非空冲突清单含中文段名与双方值，零写入）；ImportProjectCharacters 签名扩展 +overwrites（{角色ID:[字段键]}，勾选才生效、表外键忽略、空值不写=不允许清空、同值幂等零覆盖）+overwritten 计数，nil=v4.192 行为不变。**绑定面 604→605**：+CharacterImportPreview、CharacterImportProject +overwritesJSON 返回 {imported,filled,overwritten}；六处接线全同步（门面重生成/bindingNames/bridge 断言/wailsjs/spaceBindings play/dev mock）。**前端**：角色面板头部新增常驻「回写」钮（迁移后非空差异从此有出口）+确认弹窗按角色分组逐字段「库内值 vs 副本值」复选默认全不勾；回执诚实化三计数；CharacterLibraryPage 导入回执修 [object Object] 遗留 bug。RoleType/Arc/Status 任何路径不进库（关联即快照）、无后台自动同步边界照旧。Go 全量 0 FAIL（+OverwriteConfirmed/+ReadOnlyConflicts）、vitest 316 文件 2762 例全绿（数量锁 426）、tsc/eslint 0、drift PASS@605、e-check OK。详见 releases/v4.193.0.md。

## v4.192.0 · 阶段四三小刀：角色同一人设（2026-09-10）
> 阶段四出口落地（读档 docs/gaea-character-domain-survey-2026-09.md：一套资产+项目工作副本，缺口=外观锚点分裂与演化漂移）。**①外观锚点单一来源**：绘梦参考槽在库内参考图为空时回退读 Appearance 文案并入提示词（可编辑、提示补图后可用图生图），不再静默空槽，无图也无文案才提示去角色库补全（useImageGenConfig applyRefCharacter）。**②副本→库显式回写**：ImportProjectCharacters 在既有单向约束内追加**描述性空字段补全**（库内为空而副本有值才写；非空一律不动，RoleType/Arc/Status 不补=关联即快照），返回 {imported,filled} 双计数；绑定回执改 map（604 零变更），CharacterPage 迁入按钮回执诚实化「新迁入 X、补全 Y（已有设定未覆盖）」；不覆盖语义由既有 TestImportProjectCharacters_IdempotentAndOneWay 继续锁 + 新增 EmptyFill 五段测试。**③关联即快照进 UI**：本书局部设定标签精化「仅本小说生效 · 与角色库互不影响」。不做后台自动同步（显式桥足够，维持不做批量迁移）。Go 全量 0 FAIL（characterlib/app +3 处断言改双计数）、前端 tsc/eslint 0、drift PASS@604、e-check OK。详见 releases/v4.192.0.md。

## v4.191.0 · 阶段三首刀：cost_compose 开口——管家会话里组价（2026-09-10）
> 长期规划阶段三「造价开口」首刀（阶段二已出）：开口 → 对上历史 → 确认进工程，不必先把造价页当目的地。**缺口**（survey+contract 已读）：agent 工具表有 cost_search/cost_save（hardAsk），组价全链只在造价页（GaeaCostCompose→Apply）——会话里说不出一句「帮我组个价」。**新工具 cost_compose**（只读，不落库）：清单描述→相似条目检索（SQL+语义补召回+本地精排，与 cost_search 同款组合，检索基建零新建）→价格带推荐（P25/中位/P75+样本数+置信度+离群标注）→证据链（溯源五元组表格，含离群甄别提示）；**设计取舍=嵌套 LLM 拆解不做**——管家本身就是 LLM，人材机拆解由它基于证据自行给出（也不把组价拆成逐步盖章）；采用后经 cost_save 确认沉淀（既有落盘门=确认进库，不新造门）。spacetags 钉 work（隔离红线：组价=工地话题不进会客厅）；compact 表登记；V16 留痕快照仍归 UI 全链（会话路的沉淀以 cost_save 条目为准）。Go 全量 0 FAIL（+TestCostComposeBandAndEvidence/+TestCostComposeEmptyAndEdge/+TestCostComposeSpaceTag）、绑定面 604 零变更、前端零改动（tsc/eslint 0）、drift PASS@604、e-check OK。详见 releases/v4.191.0.md。

## v4.190.0 · 空间策略写路径：总闸当场切（2026-09-10）
> v4.189 欠账收刀：space_profiles 从只读显形到可编辑。**Go**（603→604，+GaeaSpaceProfileSet，显式映射 core）：applySpaceProfileEdit 纯函数校验+落段——ref 空=清除该键（全空段删除不留 TOML 噪音）；非空必须经 ResolveModel 同链解析，解析不了宁拒不写，错误消息带已配置 provider 清单（坏引用写进去只会骗人——boot 告警后现状继续，UI 不制造这种状态）；非法空间/未知键拒绝；持久化走 gaeaSetSessionSpace 同款锁+Save，生效时机=下次引擎重建/重启（boot 装配世界，运行中引擎不动，与 GaeaSpaceActivate 同口径诚实）。**前端**：空间策略卡 gaea 行加编辑钮——带出当前值→provider/model 输入（留空=维持现状）→保存调绑定、视图随返回刷新、诚实 message「已写入，下次引擎重建/重启生效」；后端校验失败原样 message.error 不吞。**mock**：模块级状态支撑浏览器演示态编辑闭环。Go 全量 0 FAIL（+TestApplySpaceProfileEdit 五段）、vitest +2（编辑闭环/写入失败说人话）、tsc/eslint 0、drift PASS@604、e-check OK。详见 releases/v4.190.0.md。

## v4.189.0 · 阶段二首刀：模型中心「空间策略」分区——总闸画面显形（2026-09-10）
> 长期规划阶段二首刀（1A/1B 已全出口）：打开模型中心能看见空间绑的策略。**缺口盘点**：本机/云端（引擎管理 is_local 徽标）与功能域绑定（功能绑定分区+生效路由）本就在，唯空间装配 profile（[space_profiles]，boot 消费 prof.Gaea 覆写办公 agent 模型）零 UI 显形。**Go**：新增只读绑定 GaeaSpaceProfiles（602→603，CoreB 门面+gen_bindings 显式映射 core）——SpaceProfileView 逐空间汇报「配了什么+生效成什么」：gaea 覆写原值+ResolveModel 解析结果（坏引用诚实报「无法解析，现状模型继续生效」=失败说人话）、其余功能域覆写（仅非空项）、生效权限模式+强制审批（按空间 N 项/包级默认）、play 护栏生效态、space.mode 开关；引擎未初始化读盘兜底（假空防御）。**前端**：模型中心新「空间策略」分区（Category strategy，功能绑定与引擎管理之间）——双空间卡+当前生效徽标（GaeaSpaceActive，失败降级不拖垮整区）、mode=off 警示条、读取失败诚实错误卡+重试、底部固定注明壳层=导航与本区两套（1B 拍板）。**接线**：wailsjs 重生成/types wire/bridge facade/mock（浏览器演示态双空间样例）/spaceBindings shared/bindingNames 603。Go 全量 0 FAIL（+TestBuildSpaceProfileViews 四段）、vitest +3（渲染三态/mode off/失败重试）、tsc/eslint 0、drift PASS@603、e-check OK。详见 releases/v4.189.0.md。

## v4.188.0 · 1B 出口：拍板「拆开」落档 + 切换零扰契约锁（2026-09-10）
> 用户拍板：切换时不影响现在的工作——办公板块在跑时切到小说，办公照常。即两套空间开关**拆开定局**：壳层=导航，引擎空间=办公侧栏另一件事；合并方案（切空间写 session.space+安全点重建）作废，「一年后是一个开关」画面从展望删除。**实证链**：switchSpace 纯 localStorage+React 态零桥接；keepAlive 剪枝卸载只退订流不 cancel（useChatStream 卸载收尾仅解订阅/收尾 Promise/停打字机）；GaeaCancel 只挂显式停止按钮（controller.ts）——切换对后端回合零影响，行为本就满足拍板，零行为变更。**落档**：展望契约修订（书斋「同一场活、同一套空间」句改写/迭代清单该条收口/「机器现在」注明两套是定局）、长期规划 1B 标记出口、一年后验句改为「切的是导航，管家与在跑的活不受打扰」。**守卫**：e-check 新增 E26——switchSpace 函数体零桥接调用（app./wailsApp(/GaeaSpaceActivate/noteSpaceActivated/.Cancel(/GaeaSend），违反即红。零绑定 602、零 Go 改动（版本三处外）、零 locale 变更。vitest 目标 10/10、tsc/eslint 0、e-check OK（E26 PASS）、drift PASS@602。详见 releases/v4.188.0.md。

## v4.187.0 · 1B 前置：两套空间开关 UI 说清楚（2026-09-10）
> 长期规划阶段一 1B：合并 vs 拆开待拍板，未拍板前不改行为，唯一许可动工=把两套开关在 UI 上说清楚。首页顶栏 SpaceSwitch 两钮 title 追加「仅切换界面与导航，不影响办公引擎空间」（新增 home.spaceSwitchHint×3 语，aria 精化为「界面空间切换（书斋/闲庭）」），书斋 masthead 空间 chip 同 title；办公侧栏 SpaceChip 的 sidebar.spaceHint 开头点名「办公引擎空间（区别于首页顶栏的书斋/闲庭界面切换）」。壳层=导航（switchSpace 只改 appStore.space），引擎=GaeaSpaceActivate 写 session.space 下次重建生效——两开关同名同值正是 1B 点名混淆源。零绑定 602、零行为变更、零 Go 改动（版本三处外）。vitest +1、tsc/eslint 0、e-check OK、drift PASS@602。详见 releases/v4.187.0.md。

## v4.186.0 · 场景重排接 UI：上移/下移 + ReorderScenes 落盘（2026-09-10）
> v4.185 欠账收刀：场景化后场景顺序=阅读顺序与 blob 投影顺序，ReorderScenes 绑定此前零消费方。ChapterEditor 场景框头部增上移/下移（aria-label 键盘可达，首位/末位/非场景制章禁用）；moveScene 乐观换位（scenes/sceneIds 同步 swap 经 onUpdate 回喂）+ ReorderScenes 落盘（blob 投影 Go 侧同调用同步），失败回滚还原诚实提示；缺 id 框只换本地不调绑定防错位。零新绑定 602、零 Go 改动（版本三处外）。vitest +3、tsc/eslint 0、e-check OK、drift PASS@602。详见 releases/v4.186.0.md。

## v4.185.0 · 阅读页场景化：读场景拼装 + 逐场景保存 + blob 双向同步（2026-09-09）
> v4.184 阶段一 1A 欠账收刀，展望验收面「小说新章在阅读页是场景」落地。**Go 双向同步**：`syncBlobFromScenes` 把场景库投影回整章 blob（空段拼接，导出/搜索/统计口径不变），挂 SaveScene/ReorderScenes/RestoreSnapshot/GenerateScene 四条场景写路径；`rebuildScenesFromBlob` 在 CreateChapter 主线完成点把整章重写后的场景重置为单场景（旧拆分/POV 元数据不再对应新正文，诚实可预期）。**前端**：V4 主线章载入走 GetChapterScenes 逐场景框（sceneIds/sceneBacked 进 tab，IsProjectV4 ref 探测无竞态；分支/V3/mock 保持 blob 模式），保存逐场景 SaveScene（blob 同调用内同步），ChapterEditor ids 改由 tab 携带。绑定面 602 零变更。去味/重写经核查本就场景感知。Go 全量 0 FAIL（+3）、vitest +2、tsc/eslint 0、e-check OK、drift PASS@602。详见 releases/v4.185.0.md。

## v4.184.0 · 小说主路径接通：blob 章物化场景 + 加场景走 CreateScene + 章节体检进检查器（2026-09-09）
> 长期规划阶段一 1A 出口刀（docs/gaea-outlook-longterm-plan-2026-09.md）。①**blob 章物化**：CreateChapter 主路径写整章 blob，V4 下 GetChapterScenes 只认场景文件——创作生成的章逐场景生成全部「无绑定 ID」禁用；新 ensureBlobChapterScene 在纯 blob 章首次触碰场景 API 时把 blob 全文物化为首场景（slug=chapter/大纲章标题/done，幂等：已有场景或无正文不动盘；失败只降级不阻断），GetChapterScenes/CreateScene 双接线。②**加场景真落盘**：ChapterEditor.addScene 从只改 React 数组改为 CreateScene→重拉 id 序→文本框对齐（物化自动增框）；分支章与失败降级本地加框不丢输入。③**章节体检**：RunChapterGate 从未被消费的假门面挂进 NovelInspector 阅读页分区——「章节体检」单按钮手动触发，合并分析/审查/一致性/AI味四路报告（单路失败诚实显「未启用」，换章弃旧报告），非每章盖章向导。④chapter-active 事件载荷补 chapterNum。绑定面 602 零变更（RunChapterGate 本就在 NovelB 门面属 legacy 直调面）；Go 全量 0 FAIL（+4 物化用例）、vitest +4（ChapterEditor：id 拉取/空表禁用/CreateScene 落盘对齐/生成写回）、tsc/eslint 0、e-check OK、drift PASS@602、零新增 locale 键。详见 releases/v4.184.0.md。

## v4.183.0 · 双空间首页重排版：书斋文书台/闲庭游园画廊 编辑部级排印（2026-09-09）
> 用户反馈 v4.182 版式「太 low」——同质卡片海（8 等大 Bento 瓦片+右舷五连盒）是模板感根源。重排版（信息零删除，ui-ux-pro-max 双查询定方向：书斋=Premium refined/Liquid Glass，闲庭=Bold gallery/density3）：**书斋「文书台」**=编辑部排印——竖排空间名书脊 masthead+大字渐变标题+Hero 命令条（AI 打字/语音直启，focus-within 辉光）+最近文档卷宗流水（hairline 报表行+序号 hover 点亮）+旗舰横带（accent 竖条+细网格纹）+**目录式索引行**（等宽序号 01-05+行 hover 辉光，两列报表，替代等大瓦片）+右栏单一仪表纵栏（遥测/写作/会话/记忆/晨报 hairline 分节，去五连盒）；**闲庭「游园画廊」**=画廊排印——居中大字（clamp 42px+宽字距）+空间名小签两侧渐隐线+会客厅旗舰横幅（**月洞门圆环母题**+圆形徽记+CTA 胶囊）+**竖版海报大卡**（圆徽记+底部环形箭头 hover 滑入）+园底单条信息带（进度/继续话题/记忆/遥测 hairline 分节）。**结构修复**：M3 tertiary 角色全仓从未定义（v4.182 切换器暖色恒走 fallback）——appStore 增 TERTIARY 暖伴侣色表（6 预设×明暗）+getThemeTokens 合并+App.tsx 注入 CSS 变量，闲庭暖金母题与书斋青绿真实分版；契约测试锁定。无新增 locale 键；testid/aria 全保留（ModuleLauncher.test 3 例不变绿）；vitest 2746（+1 tertiary 契约）、tsc/eslint 0、e-check OK、零绑定 602、零 Go 改动（版本三处外）、build strip+冒烟 200。目检=esbuild 隔离挂载+Edge 无头截图（window.go.app 打桩走真机路径绕开 dev mock）双空间+窄档三张通过。详见 releases/v4.183.0.md。

## v4.182.0 · 双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏（2026-09-09）
> 用户拍板：切换按钮迁首页顶栏；两首页重设计各有特色。定名=书斋（work，Study）/闲庭（play，Lounge）三语覆盖 shell.space.*/shell.search.scope.*。切换器=首页顶栏 SpaceSwitch 胶囊组（aria-pressed+模型徽标，MainLayout.switchSpace 直连），rail 顶部改竖排空间指示徽标（非交互，分域导航不变）。书斋「文书台」=三栏效率台（Hero 命令条+最近文档流水主角面板+能力矩阵+右舷遥测/写作/会话/记忆/晨报）；闲庭「游园画廊」=全幅画廊（会客厅旗舰横幅+板块大卡两列+园底信息带：进度环/继续话题 chips/记忆/遥测细条）——信息零删除形态分化。ui-ux-pro-max 双查询定 dial（书斋密度7/Flat，闲庭密度4/Showcase）。新增 ModuleLauncher.test 3 例、CommandRail.test 更新、space.test 断言同步。vitest 2745、tsc/eslint 0、e-check OK、零绑定、build strip+冒烟 200。详见 releases/v4.182.0.md。

## v4.181.0 · 上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊（2026-09-09）
> 用户反馈 Agent 网络不按当前会话。根因=GaeaTrajectory/GaeaAgentNetwork 无参绑定恒读内核 ga.ctrl 单例会话，UI 切历史会话后不跟。修复=两绑定变参 sessionPath（显式路径优先/缺省回落内核兼容），新 resolveGaeaSessionPath；前端 bridge 契约（JS 数组形态）+mock+agentNetworkStore 路径声明（跨会话清快照宁空勿错/同会话宁旧勿断）+六消费方接线（ContextView/SubagentsPanel/TasksWorkbench/TrajectoryView 新 prop/AgentNetworkCard；BrowserPanel 保持内核兜底）。附带 GLM 429 分诊：编码套餐资源包挂 coding 端点计费域，标准端点 429 时提示切换「编码套餐」端点（用户实测根因）。绑定名 602 零变更；Go 全量绿（+1 显式路径用例）、vitest 2742、tsc/eslint 0、build strip+冒烟 200。详见 releases/v4.181.0.md。

## v4.180.0 · 办公任务管理会话关联刀：会话待办区入任务管理页（2026-09-09）
> 用户反馈任务管理「没与当前会话关联、全是固定内容」。定位=页面主体是 TaskCenter 全局后台任务表（价格抓取/语义索引 cron 持续在列，天然会话无关），而会话真任务（todo_write 待办）无入口。修复=TasksWorkbench 首区新增「会话待办」：直订全局 controller store items（随会话切换，零 props 穿层，ContextModal 弹层同源正确），复用 useTodoExtractor 提取最新 todo_write（与聊天流待办卡同规则），三态条目+进度 n/m，无待办不占位；页面四区语义=会话待办/子代理/本地模型工具（均会话）+任务中心（全局）。零绑定 602、TaskCenter 本体零改动。vitest 2739 首跑全绿（+1）、tsc/eslint 0、e-check OK、零 Go 改动、build strip+冒烟 200。欠账=TaskCenter 会话关联需任务表加 session 维度（结构刀另立版本）。详见 releases/v4.180.0.md。

## v4.179.0 · 瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点（2026-09-09）
> P0-P4 收官后清点刀：①knip 首扫三发现——死文件 13 全删（App/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip/PromptShelf/office 死链对/memoryhub 两件/typesGenerationCheck/m3-palette，全仓零 import 甄别）、幽灵依赖 13 包显式化（jszip×8 处/katex/dayjs/unified/hast-util-sanitize/@lezer×8 共 20 处 import 靠传递依赖侥幸工作，按 lock 现版本钉死）、顺带清死传递依赖 @codemirror/search；三存疑依赖（gsap/docx-preview/unist-util-visit）验活全部在用结案。②冷启动基线打点：Go New+Startup 七段耗时落长期日志（日常启动自动积累）+前端 gaea:boot→gaea:interactive 两 rAF 口径（CDP 可读）；初始化链审计结案=重活均已异步，懒初始化无需立项。零绑定 602、零行为变化。vitest 2738（5 例并发 flaky 单独复跑绿）、Go 116 包 0 FAIL、tsc/eslint 0、build strip+冒烟 200、exe 46.16MB 持平、knip 复扫死文件/幽灵依赖双清零。欠账=Unused exports 169 项挂下版+壳内真机池不变。详见 releases/v4.179.0.md。

## v4.178.0 · 造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI（2026-09-09）
> 造价域缺口 §1 首刀销账：SchemaV17 cost_entries/cost_estimate_items 增 code 列+索引；cost.NormalizeCode 全角/空白/大小写归一化；导入表头「定额编号/清单编码」列映射+编码优先匹配（带码未命中即新增，同标题不同编码不误覆盖）；costref 归因对标 matchEntry 编码精确优先（matchedBy=code 溯源，宁漏勿误配）；沉淀/引用继承编码；cost_save/cost_search 工具面+四组件 UI 贯通。零绑定 602；附带修复 e-check 三处陈旧守卫（v4.174 bridge/v4.176 lazy 甩下的直调/静态 import 断言）。Go 全量绿 +9、vitest 2738、tsc/eslint 0、build strip+冒烟过。详见 releases/v4.178.0.md。

## v4.163.0 · gaea 长期日志机制（2026-09-08）
> 此前日志=单文件 whisper_data/gaea.log（目录误植/无轮转/无上限），排障无从下手。长期机制：①按日分文件 <DataRoot>/logs/gaea-YYYYMMDD.log（天然轮转+归档友好）；②启动清理过期（保留 365 天）+总量兜底（>1GB 从最旧删）；③旧 gaea.log 一次性迁入 logs/gaea-legacy.log；④启动头写版本+Go 版本、Shutdown 记运行时长；⑤新绑定 GaeaOpenLogsDir（599→603 系），设置→数据加「打开日志目录」按钮。**附带修复 v4.162 缺陷：GaeaReadFileB64/GaeaSaveFileAs 漏加 facade 委托行致运行时不可达（只有 bindings_*.go 门面方法才被 Wails 暴露；gen_bindings -names 只扫 Go 方法名，两层面不同步）——OfficeB 补三行委托，壳内 CDP 实测通过。**绑定 601→603；vitest 2677、tsc/eslint 0、build+冒烟过。详见 releases/v4.163.0.md。

## v4.162.0 · 进度计划导入导出壳内修复：原生对话框链路（2026-09-08）
> 用户实测：进度计划「导入导出点击没有反应，包括菜单栏这些也是」。CDP 挂 WebView2 远程调试逐层实测定位：菜单/下拉/点击全部正常，断点在文件通道——Wails 壳内 `<input type=file>.click()` 不弹文件对话框、`<a download>` 下载不落盘（浏览器 dev/prod 均正常，故此前测试全绿不可见）。修复：新绑定 `GaeaReadFileB64`（读所选文件）+ `GaeaSaveFileAs`（系统另存为对话框+写盘），导入四路（XML/Excel/MPP/新工程）改走 `GaeaPickFiles` 系统对话框+路径读取还原 File 喂原解析器（浏览器回退原 input.click 通路，jsdom 用例口径不变）；导出五路（XML/Excel/PNG/PDF）统一 saveExportBlob（壳内另存为/浏览器下载）。绑定 599→601；vitest 2671→2677、tsc/eslint 0、build+冒烟过；壳内 CDP 受信任点击实测=原生「选择文件」对话框正常弹出。详见 releases/v4.162.0.md。

## v4.161.0 · 双代号对齐标杆二期：工程标尺 + 图面语言（2026-09-08）
> 继续对齐重庆干休所网络计划标杆件：①时间坐标框架进视图——顶部工程日刻度/月/日+底部星期/工程周（新纯函数 aoaRuler.ts，与导出标尺同口径：列=工作日、步进按总工期 5/10/20、月标签落本月首个工作日、工程周每 7 列一档），月界竖线贯通图面，随缩放/滚动与图面对齐，手动布局（x 与时间解耦）不画；②波形线拆独立路径着绿色（segsToPathSplit，segsToPath 保留旧输出兼容），视图/导出同口径；③关键事件红圈（es=ls）、工作名/工期标注改蓝色系、非关键工作线加深，图例同步。③修 v4.160 带内打包阶梯化 bug：按节点序打包把同列同行道的合并链拆成阶梯——改按全局重心行道（gRow）压缩打包，链保持一条水平线（目检实锤后修正）。零绑定 599；vitest 2671→2677、tsc/eslint 0、build+冒烟过。详见 releases/v4.161.0.md。

## v4.160.0 · 双代号分级横幅：一级/二级分区布局（对标上报件画法）（2026-09-08）
> 用户以重庆干休所网络计划 PDF 为标杆反馈：双代号「一级、二级没有分级展示，比较混乱」。解析标杆图提炼画法=每个分部工程一条横幅：分部汇总线走横幅顶部通长线（分部名标线上）、二级工作只在横幅内排布、跨分部一律虚工作竖向衔接、交替底纹分带。本刀照此重构双代号布局：①buildAoa 新增 band 归属（一级分组行=横幅，事件归成员最小 band）+band 内保序打包（沿用全局重心序），汇总箭线改横幅顶通长线（summarySegs 三段正交：竖起→横贯→竖落）；②跨横幅不合并事件（FS/SS 合并限同带）+非首横幅无前置任务自设开始事件——跨分部关系全部由虚工作衔接，杜绝任务边跨带穿行压字（实测 26 事件→34，时差/关键判定不变）；③视图与导出同口径接横幅底纹（交替浅色垫底）；行距自适应接横幅行数。零绑定 599；vitest 2665→2671、tsc/eslint 0、build+冒烟过。详见 releases/v4.160.0.md。

## v4.159.0 · 进度计划画布手感：滚轮缩放 + 双代号行距自适应（2026-09-08）
> 用户实测反馈两处：画布无法用鼠标滚轮缩放（只有按钮）、双代号网络图「行距太小全挤在一起、又矮又长严重不协调」。①三块画布补滚轮缩放：双代号/单代号=普通滚轮以光标为锚缩放（新共用钩子 useWheelZoom，layout effect 回写锚点滚动量防旧尺寸钳制）、Shift+滚轮放行原生横移；横道=Ctrl+滚轮缩日宽（MS Project 口径，普通滚轮保持滚动行，光标下日期锚定）。②双代号布局松绑：AOA_ROW_H 74→112（节点含上下时间标注占 56px，旧 74px 行只剩 18px 空隙），且行距按整图宽高比自适应放大（时标宽度被工期锁定，低并行度长计划天然扁平——宽高比超 7 时放大行距、上限 2×）；同行长边通道层距 12→16 治名称/工期标注叠压；PdmView 自动适配补 clientWidth≤0 防护（对齐 v4.143「jsdom 0 宽不误适配」纪律）。零绑定 599；vitest 2658→2665、tsc/eslint 0、build+冒烟过。详见 releases/v4.159.0.md。

## v4.158.0 · AI 组价复核闭环：确认留痕 + 拆解合理性校验（2026-09-09）
> 前置调研校正：AI 组价 v4.2 已在产，roadmap §15 部分造价欠账描述过时（校正档 docs/gaea-cost-domain-survey-2026-09.md）。本刀销真缺口：①SchemaV16 cost_compose_records 确认即留痕（完整视图快照；无确认不落库红线不变；留痕尽力而为不阻断回写）+GaeaCostComposeRecords 回看（条目详情「组价依据」折叠区，证据链表抽共享 ComposeEvidenceTable）；②cost.CheckComposeComponents 纯函数（金额一致性/非正值/合计偏差 warn+info 两档，恰阈值不触发）进 Compose 响应，ComposeModal 行级徽标+全局提示，校验随留痕入档。绑定 598→599；Go 全量绿 +18、vitest 2650→2658、tsc/eslint 0、build+冒烟过。详见 releases/v4.158.0.md。

## v4.157.0 · Office 编辑链一致性小刀：docx 证据链补齐 + pptx 面板字符级对比（2026-09-08）
> 两个小项收口三件套编辑线：①docx_apply 补快照+Journal（对齐 pptx_apply/xlsx_apply；ApplyTrackedReplace 语义零改动；accept=true 同款、拒绝分支天然回滚不加）——docx 编辑可经版本时间线回滚；②PptxEditPanel 对比区升级 ChangesDiff（changed=相邻 del/add 对改蓝配对+字符级高亮；ctx 折叠不伪造全量；删句级降级标注）。零新绑定 598；Go +3、vitest 2650、tsc/eslint 0、build+冒烟过。详见 releases/v4.157.0.md。

## v4.156.0 · pptx 真编辑刀2+刀3：编辑面板 + 版本对比（2026-09-08）
> docs/gaea-pptx-edit-design-2026-09.md 刀2/刀3 并行交付（刀1=v4.109.0）——docx/xlsx/pptx 三件套编辑能力闭环。绑定 597→598（+GaeaPptxSlideText）。
- 刀2：新绑定 `GaeaPptxSlideText`（pptxedit.LocateText 段落全文,纯 Go 零 python;大纲 texts 是 200 rune 截断预览不可当 target,apply 单段落精确匹配宁拒不误改）;PptxOutline 两级导航;PptxEditPanel(P1 Plan→Apply:段落单选→预设动作/自定义指令(短句化防溢出约束)→双栏对比→应用→新预览刷新);FilePreview pdf 分支挂面板;桥接六处(数量锁 310→311,mock ApplyEdit 未命中按后端同语义抛错)。
- 刀3：lib/pptxTextDiff.ts(JSZip 解 slides,页签名 LCS 对齐+段落级复用 diffDocxParagraphs;坏 zip 结构化降级);versionCompare kind:"pptx"(取数走 AttachmentDataURL——Preview 对 pptx 渲染成 PDF 缩略、基线 .before 扩展名不走 Preview 分派);VersionTimeline PptxCompareBody(页摘要+差异页 hunk,改蓝配对+字符高亮白得)。
- 门禁：Go 全量绿(+3)、vitest 2617→2650、tsc/eslint 0、drift PASS@598、build+冒烟过;flaky=TestCreateChapter_SameChapterConcurrentRejected TempDir 清理竞态复跑绿。真机走查挂真机池(mock Preview 无 pptx 分支,v4.28 起即如此)。详见 releases/v4.156.0.md。

## v4.155.0 · 双工期欠账放开：SS/FF/SF×cd 全搭接（§3.2.5 重推，零不动点）（2026-09-08）
> 双工期设计 §6 欠账池首项销账：v4.150 刀1 曾收窄「cd 任务仅允许 FS 搭接」，本刀实施前重推语义矩阵——原矩阵四格「需不动点」均系 wd 代数形误判，实际零迭代可解，八格全放开。TS↔Go 同批镜像，597 零变更。
- 重推手段：①新原语 `cdEarliestStart`（单调逆查：最小 s 使 cdToEf(s)≥T，与 cdToEf/cdLatestStart 互为镜像三件套）解 FF/SF-to；②SS/SF-from 直接 ls 下界（约束本就落在开始上，不经 lf+durFrom 换算）+后继 effLf=cdToEf(ls) 诚实传播。
- 落地：正推 to=cd FF/SF 走逆查；逆推 from=cd 拆双上界 `ls=min(cdLatestStart(lf), lsBound)`、lf 保留 raw（兼容「fwd(ls)≤lf」pin）；wd 路径逐位不变（FS 项无条件读 to.ls 恒等）。
- 拆闸：Go Validate「仅 FS」/引擎防御拒绝/analyze 防御 finding 三处全删，cd 余四闸不动；scheduleApply 描述、技能条款（锚点锁 37 不变）、types 注释、甘特 chip title 四处文案同步。
- 测试：锚点表×6+用例 A-E 两侧镜像；golden 62→65（cd+SS/SF/FF 成功例）；既有 fail-closed 用例按新语义原位翻转。Go 全量绿、vitest 2601→2617、tsc/eslint 0、drift PASS@597、build+冒烟过。详见 releases/v4.155.0.md。

## v4.154.0 · MPP 导入支持 Project 2013+ 变体（真机样本驱动修复）（2026-09-08）
> 用户真实工程文件（Project 2013+ 保存）此前被 ParseMpp 拒收（「缺 CompObj」），强拆后任务名/工期/层级系统性错乱。本刀字节级取证钉死 2013+ 变体五处漂移并修复——该文件现已完整导入（38 任务/8 分组/42 搭接/总工期 136 天零缺名）。纯 Go 内部引擎，597 零变更。
- 五处漂移：①无 CompObj 流→按工程目录编号判版（   114=MPP14，无 CompObj ⇒ appVer≥15）；②任务工期 @42→@84；③Var2Data 回链键 uid→ID（uid@0 是陈旧键）；④parentUID@36 失效→大纲层级 @172+层级栈重建父链；⑤里程碑位失效→零工期语义。
- 搭接：pred/succ 确认同为 ID 键空间，45→42 条全数回链；lag@14 数值合理。
- 诚实降级：2013+ 资源/分配行键位未钉死（解析出垃圾），宁缺勿错暂不解析，留观察池。
- 门控：TestMppRealSamples2013（clones/mpp2013/ 缺失 Skip）钉样本期望；旧三样本（MPP9/12/14-2010）零回归。Go 全量绿、版本三处 4.154.0、build+冒烟过。详见 releases/v4.154.0.md。

## v4.153.0 · 双工期口径刀4：mspdi 互通（DurationFormat 8 导出/导入）（2026-09-08）
> 双工期四刀收官刀：cd（日历天）任务进出 MS Project XML 不再失真（此前 ed 任务导回被静默当工作日）。纯前端，绑定面 **597 零变更**。
- 导出：cd 叶任务 `DurationFormat=8` + `PT{自然日×24}H`（wd 恒 7+PT{d×8}H 逐位不变；分组恒工作日口径）；Start/Finish 本就日历日期自动吻合。
- 导入：Format 8 → cd（分钟÷1440）；其余/未知格式维持 ÷480 工作日折算宽松容错；里程碑/汇总不标 cd；经 normalizeProject/引擎 fail-closed 双保险。
- 重构：parseDurationDays→parseDurationMinutes 按格式分流；FORMAT_*/MINUTES_PER_ELAPSED_DAY 常量显式化。
- 门禁 vitest 2596→**2601**（+5）、tsc -b/eslint 0、drift PASS@597、版本三处 4.153.0、build+冒烟过；Go 零改动。**真机池：本机未装 MS Project——Project 往返重算/PT×24 序列化钉死/WPS 斑马行为挂真机池走查（沿用「等真机」先例）；Import 端为失真修复方向性收益确定，Export 端对 Project 实际兼容性待验证。**详见 releases/v4.153.0.md。

## v4.152.0 · 双工期口径刀3：板块 UI（单位 chip + 拖拽换算 + 等效透明）（2026-09-08）
> 双工期四刀之本刀3：板块拿到 cd 的表达与编辑入口。纯前端，绑定面 **597 零变更**。
- 工期列「日历」chip：cd 行常显、wd 行 hover 浮现（静态视觉零变化）、分组行/里程碑无入口；点击循环 wd↔cd 不换算数值（拍板项 7）；工期列宽 48→84。
- 拖拽缩放：新 `resizeToNaturalDuration`（右缘=新 ef 工作日吸附，回写自然日差=所见即所得）；右缘起点 `dayNo(origEs+origDur)`→`dayNo(row.ef)`，顺带修复 cd 条形画过宽（改按真实日历跨度）；拖拽浮标「日历 X → Y 天（自然日）」。
- 口径透明：条形悬停「日历天 28（等效 20 工作日，日期区间）」；前锋线 dur 换源等效跨度；检查器工期行显等效；状态栏「（工作日）」补词；单/双代号节点箭线 cd 标「(日历)」+图例注记（计算不经视图，诚实标注）。
- 门禁 vitest 2589→**2596**（+7）、tsc -b/eslint 0、drift PASS@597、版本三处 4.152.0、build+冒烟过；Go 零改动。刀4 mspdi 互通（绑真机池）待续。详见 releases/v4.152.0.md。

## v4.151.0 · 双工期口径刀2：agent 通道（ops durationUnit + cdTasks 清单 + 技能条款）（2026-09-08）
> 双工期四刀之本刀2（设计 §3.5）：打通对话链路——AI 可用 ops 表达养护类自然日定时工作，analyze 出日历天任务清单自检，技能写入 cd 纪律。零新绑定 **597**；**Go/TS 对拍 golden +12 例（49→61）双端同吃**。
- ops：patch_task.durationUnit 指针三态（空串 no-op 同 Mode 先例；非法枚举报错原文）；单位分支先于工期分支——工期词尾随新单位，cd 态超 3650 当场拒；upsert_task 直带 durationUnit（校验同 Validate 五闸，新增回执「工期 28 日历天」）。
- 三工具：get taskView 带 durationUnit（wd 不出=旧客户端零噪音）；analyze 增 cdTasks（duration 自然日数/span 等效跨度/anchor 锚点日期）+ cd 非 FS 防御发现；apply 回执「总工期 X 天（工作日）」。
- 技能：工期分节增 cd 纪律（自然日定时才标 cd/仅 FS/成本按 ef−es=元/工日不放大；定额工期禁误标）锚点锁 35→37；汇报口径补「混排先讲 cdTasks」；compact 三工具同步。
- 门禁 Go 全量绿（+3）、vitest 2576→**2589**、tsc -b 0、drift PASS@597、版本三处 4.151.0、build+冒烟过。刀3 板块 UI/刀4 mspdi（绑真机池）按序待续。详见 releases/v4.151.0.md。

## v4.150.0 · 双工期口径刀1：任务级工期单位（模型+引擎，TS↔Go 镜像）（2026-09-08）
> 拍板池最后一项启动实施（推荐项全采纳）：养护类自然日定时工作不再被迫按工作日误录。`task.durationUnit?: 'wd'|'cd'` 缺省 wd 旧文件零迁移；cd=日历天，引擎完成边界 ceil 吸附工作日。绑定面 **597 零变更**。
- 引擎：`cdToEf`/`cdLatestStart` 换算纯函数对（ceil 吸附折叠 26/27/28cd 同 ef；逆推平段镜像回退+有界双侧校正，性质 fwd(ls)≤lf<fwd(ls+1) 钉死）；cpm 增可选 ctx（Go 双入口 ComputeCpmCal/ComputeCpmPlanCal），**快路径铁律**=无 cd 任务逐位一致（既有用例零改动全绿）；缺开工日期与 cd 涉非 FS 均 fail-closed。
- 成本/基线换源：work 分配与基线快照/漂移改用 cpm 行等效工作日跨度 ef−es——养护期周末不记工日（28cd=20 工日×费率），wd 两值恒等有 pin；倒排/摘要卡零改动。
- 校验：Go Validate 五闸（枚举/分组行/里程碑/3650 上限/FS-only）；TS normalizeProject 容错回落 wd。TS↔Go 同批镜像用例 vitest +26、Go +10。
- 门禁 tsc -b 0、vitest 2550→**2576**、Go 全量绿、版本三处 4.150.0、drift PASS@597。刀2 agent 通道/刀3 板块 UI/刀4 mspdi 按序待续。详见 releases/v4.150.0.md。

## v4.149.0 · diff 确认卡刀D：ops 通道投影模拟器 + Go/TS 对拍收官（2026-09-08）
> diff 确认闭环设计收官刀。ops 通道此前在审批卡上只是意图清单——模型说改什么就显示什么，看不到落盘后的样子。本刀交付 `simulateOps`（Go ApplyOps 逐语义 TS 镜像），审批卡在落盘前投影 ops 应用后的真实 diff（与 project 通道同管线渲染）；设计标注的最大风险「批的是 A，落的是 B」由 **Go/TS 对拍 golden fixture（49 例双端同吃，50/50 过）** 钉死。纯前端，绑定面 **597 零变更**。详见 releases/v4.149.0.md。
- 对拍主阵地：internal/schedule/ops_golden_test.go 生成并校验 fixture（21 成功+28 失败路径，覆盖全部 12 op 及守卫分支；`-update-golden` 重生成）；opsSim.golden.test.ts 双跑逐例比对「是否失败+after 状态」（canonical 消化 omitempty 表达差）。
- opsSim.ts：深拷贝顺序应用 fail-closed；指针三态/零值语义/remove_task 级联/set_links 替换缺省 FS/set_meta 日期口径与 null 不动/auto_chain 手动跳过/set_baseline 缺 savedAt 拒绝/资源级联/重复对/外来 taskId 全部钉死。
- 卡体：ops 通道缺省路径即真 diff（「ops 投影」标注），三级诚实降级（非缺省/读取失败/模拟失败→意图清单+原因）；落盘权威永远在 Go。
- 门禁 tsc -b/eslint 0、vitest **2550**（+51）、Go 全量绿、drift PASS@597、版本三处 4.149.0、build+冒烟过。**diff 确认闭环四刀（A 回滚/B diff 卡/C 引擎强制/D 投影对拍）全部收官。**

## v4.148.0 · diff 确认卡刀C：project 整量替换引擎强制逐条确认（2026-09-08）
> schedule_apply 的 project 整量通道（整计划生成/替换）毁伤半径最大——升级为引擎强制 hardAsk：任何权限级别（含 auto/yolo）前置逐条弹卡、禁会话放行；ops 通道维持现状闸门。可读 subject=「整计划替换：N 项工作，总工期 Y 天，关键 K 项」（同一 schedule 引擎口径，CPM 不过如实标注批准也将被拒）。schedule-edit 技能增「确认与回滚」分节（禁止原样重发/以回执为准/回滚入口），锚点锁 31→35；compact 同步。Go 全量绿（+5 闸门用例）、无新绑定 597、vitest 2499。详见 releases/v4.148.0.md。

## v4.147.0 · diff 确认卡刀B：schedule_apply 审批卡升级为结构化 diff 预览（2026-09-08）
> diff 确认闭环设计刀B（§2.2）：ask 级别审批卡本就弹出，但只有「schedule_apply 当前计划」一行字——有确认之形、无审阅之实。本刀卡体升级为结构化 diff：批准前先看清改了什么、工期/成本怎么动。决策/快捷键/超时/双宿主全复用既有件，零 Go 改动，绑定面 **597 零变更**。详见 releases/v4.147.0.md。
- 纯函数层 applyDiff.ts：`diffProjects(before, after)`——任务 id 键控增删改（字段级 from→to 中文 label）、搭接按 to 分组入边比对（对齐 set_links 语义）、资源删除标级联数、分配增删改、meta 四项、汇总行（总工期/关键/总成本双方 CPM+倒排校核+基线漂移预览）；after CPM 不过如实标注「引擎将拒绝」（预览层复刻 fail-closed）；两侧归一后再比不冒充改动。
- ScheduleDiffCard 卡体 + ApprovalModal 接线（新 prop scheduleApplyArgs，双宿主从 Transcript running 工具卡取 args）：缺省路径读文件实况为 before；非缺省/读取失败/ops 通道三级诚实降级（ops 投影=刀D）；「预览数字，落盘以回执为准」标注；60 行截断+展开全部。
- 门禁 tsc -b/eslint 0、vitest **2499**（+16）、drift PASS@597、版本三处 4.147.0、build+冒烟过。刀C（Go 审批闸门 hardAsk 化+锚点锁）/刀D（simulateOps+对拍）待续。

## v4.146.0 · 对话改计划回滚闭环（diff 确认卡刀A）：schedule_apply 进变更 tab + 回执卡「回滚本次」（2026-09-08）
> v4.125 落档的 diff 确认闭环设计按刀序启动，刀A=后置回滚补全（设计原文：即使确认卡不做也独立成立）。缺口：schedule_apply 证据卡一直在 Journal，但变更 tab 白名单不含它、工具卡无回滚入口——AI 改计划后用户没有任何可点的反悔入口。纯前端，绑定面 **597 零变更**。详见 releases/v4.146.0.md。
- 变更 tab：WRITE_TOOL_NAMES/WRITE_ONLY_TOOL_NAMES +schedule_apply（三态完备性锁同步满足）；extractChangedPaths 增可选 tool 参——缺省 path 回填「进度计划/当前计划.gsched.json」（与 Journal target 同口径，缺省路径整计划替换不漏记）；buildChangeDiff 显式降级说明（不伪造行级 diff）。
- 工具卡：新组件 ScheduleApplyRollback——schedule_apply 卡 done 后常驻「回滚本次」行（Journal 按 target 匹配最新带基线记录，标题带 turn；点击调既有 GaeaRollbackRecord，「已被手工修改」守卫在 Go 侧；无匹配整行不渲染）。
- 门禁 tsc -b/eslint 0、vitest **2483**（+6）、drift PASS@597、版本三处 4.146.0、build+冒烟过。刀B（diff 确认卡体）/刀C（Go 审批闸门）/刀D（ops 投影+对拍）按设计刀序待续。

## v4.145.0 · 工程复制（另存为）：管理面板第四动作 · 签证迭代底座（2026-09-08）

> 观察池「多工程欠账转移」销项：管理面板此前只有新建/改名/归档/删除/载入，缺「复制」——签证「调整1→调整2」迭代只能导入覆盖或从零重录。复制副本后任务 id 同源，基线漂移/任务对比天然可用。绑定面 **596→597**（+GaeaScheduleProjectCopy）。
- Go `GaeaScheduleProjectCopy(rel, name)`：源文件 `schedule.Load` 全套校验 → 新名 trim 非空 → SafeSlugName（冲突加序号，同 Create 口径，登记条目与游离文件都算占用）→ 深拷贝仅改 name 落新文件（任务/搭接/资源/基线/AOA 布点/日历随行）→ 索引显式登记（未归档，UpdatedAt=now）；**指针不动**（文件级动作语义，同归档——复制品是快照，用户自行切换）；回执含列表。
- store `copyProject(rel, name)`：rel 是当前工程时先冲刷在途编辑（拷的是已存盘内容）；回执列表直接入缓存不另刷；失败只落 syncError 返回 false（fail-closed）。
- 管理面板每行「复制」按钮 → 弹窗取名（默认原名+「-副本」，空名禁用确认；失败弹窗不关，syncError 由指示器诚实展示）。
- 绑定面接线全链：gen_bindings 门面重生成 + bindingNames.ts（596→597）+ bridge.ts AppBindings/gaeaToGaea + spaceBindings（数量锁 309→310）+ mock/office.ts（内存深拷贝语义）+ api.ts `copyScheduleProject`。
- 门禁 tsc -b/eslint 0、Go 全量绿（+TestGaeaScheduleProjectCopy）、vitest **2477**（+5：store 复制链/失败、页面弹窗流/空名禁用、mock 复制语义）、drift PASS@597、版本三处 4.145.0。build+冒烟过。
- 维持不做（记录在案）：跨工程资源池（多工程设计 §7 明确不做，单机形态无关）；基线跨工程直接复制（任务 id 不同源时漂移全噪音，工程复制保证 id 同源后文件内基线对比已覆盖该场景）；E1 负数对称哨兵维持「动逆推时收口」设计准绳；agent list_projects 刀3 维持等真机实测。

## v4.144.0 · 导入为新工程：三路导入安全化 · 非破坏口径（2026-09-07）
> 观察池销项：此前 MPP/XML/Excel 三路导入一律 `importProject` 原地整体替换当前工程——导入一个 146 任务的 MPP 直接顶掉当前工程内容（仅撤销兜底，防抖落盘后原文件被覆盖）。本轮把导入分流：**新工程=安全默认，替换当前=显式标注**。纯前端，绑定面 **596** 零变更。详见 releases/v4.144.0.md。
- store 新动作 `importAsProject`（importThenSwitch）：名字空兜底「导入工程」→ createThenSwitch（Go slug 去重+登记索引+自动切指针）→ importProject 内容替换 → flushDirty 立即落盘 → refreshProjectsCache 刷摘要；slug/文件内容/列表摘要三处同源一名。fail-closed：Create 失败返回 false、syncError 已置位、当前工程原状；成功则不触碰原工程文件（与 v4.139「复制=切指针」同口径）。
- 页面菜单重组：「导入为新工程…」主入口（.xml/.xlsx/.xlsm/.mpp 按扩展名分发+文件名兜底工程名+诚实报错）；三路替换入口改显式标注「导入 XML/Excel/MPP 替换当前工程…」（模板/签证覆盖场景保留，TemplatesPanel Popconfirm 口径不变）。
- 门禁 tsc -b/eslint 0、vitest **2472**（+5：importAsProject 成功链/名字兜底/Create 失败、页面 MPP 分发/不支持格式）、drift PASS@596、版本三处 4.144.0（顺手修正 versioninfo.rc FILEVERSION 逗号位——v4.140 起停在 4,139,0,0 四版未随动）。build+冒烟过。

## v4.143.0 · 双代号显示路径优化：长线分行走通道 · 可读优先适配（2026-09-07）
> 用户对照斑马进度/Project 截图批「别人优化显示路径方便查看，你直接自适应整个界面，图变多小完全不管，线重合了也不分行」。纯前端，绑定面 **596** 零变更。详见 releases/v4.143.0.md。
- 同行长箭线走行间通道：assignChannels（aoaLayout.ts 纯函数）——跨 >1.5 列的同行边从节点中心线让到行间通道（基准=行 y+ROW_H/2），x 区间重叠者依次 +12px 层号，非重叠共享层；五段正交路径+标注挂通道中线；短链箭线保持直连（链式笔画参考斑马口径）；视图/导出同源。
- 交叉削减加深：layerByTime 重心松弛 2→4 轮（左右交替）。
- 可读优先适配：自动适配下限 0.5（装不下横向滚动，绝不无脑缩成蚂蚁）；全览按钮下限 0.3；缩放下手动布点拖拽位移 ÷zoom 折算回图面坐标。
- 门禁 tsc -b/eslint 0、vitest **2467**（+2）、drift PASS@596、版本三处 4.143.0、build+冒烟过。走查：示例工程 50% 可读缩放整网/通道让道/汇总箭线 5 条嵌入。

## v4.142.0 · 分级双代号网络图（一级/二级界点衔接）· 横道列头错位修复 · 双代号全览（2026-09-07）
> 用户批「双代号 UI 太差没考虑布局、分级的一级和二级居然是分开的、整个图都是错的」+「横道画布滑动时列标题被压缩彻底错位」。纯前端，绑定面 **596** 零变更。详见 releases/v4.142.0.md。
- 分级双代号重做：分组行（一级）不再画成挂在 START/END 的零工期假箭线（与二级网络割裂的根因），改=**一条汇总箭线**从其二级子网络的开始界点事件连到完成界点事件——共享事件嵌入同一张图；汇总箭线不进 raws，正逆推/浮时/编号全部只由二级决定，恒非关键深色粗线；空分组无子级不出箭线；导出图面同步（深色粗线+图例项）。
- 横道列头错位：列头行此前无显式宽度且 cell 可收缩——容器窄于列合计时被 flex 压缩与数据行永久错位；修=列头行显式 width=leftW+`.sched-gantt-cell{flex-shrink:0}`，收纳=裁剪隐藏绝不挤压。
- 双代号缩放/全览：首帧自动适配（长计划 110px/天 上万像素此前根本读不了）+−/＋ ×1.2 步进+全览按钮+百分比读数；缩放下手动布点拖拽位移按 zoom 折算回图面坐标。
- 门禁 tsc -b/eslint 0、vitest **2465**（+4）、drift PASS@596、版本三处 4.142.0、build+冒烟过。走查：示例工程双代号 5 条汇总箭线（前期准备/基础工程/主体结构/机电与装修/竣工阶段）嵌入同一网络+自动适配 10% 整网入窗 DOM 断言。

## v4.141.0 · 进度计划导出预览 · 画布窗口内全览 · 周刻度（2026-09-07）
> 用户点名「导出要有预览、画布全览防无限拉长、日月之间加周刻度（参考 Project）」。纯前端，绑定面 **596** 零变更。详见 releases/v4.141.0.md。
- 导出预览：ExportDialog 内嵌预览区——构建器 SVG 注入 viewBox（svgWithViewBox）等比缩放进弹窗，350ms 防抖随图面类型/签署信息/标注开关实时刷新，导出前所见即所得；弹窗加宽 520→720。
- 画布全览：横道工具栏「全览」=整计划适配画布可视宽并回卷到起点；日宽由步进档改连续缩放（zoom×1.35，钳 2~40px），缩放按钮加 testid。
- 周刻度：底层刻度随缩放切换——dayW≥12 逐日号、<12 自然周（周一始、跨月不断开），周段标注起始日（≥44px「M/D」/≥18px「D」）、title=工程周序号+起止；月层恒在顶行。
- 门禁 tsc/eslint 0、vitest **2461**（+3）、drift PASS@596、版本三处 4.141.0、build+冒烟过。走查：预览 SVG viewBox/缩放切周层/全览 800px 入窗 DOM 断言全过。

## v4.140.0 · 进度计划工作台专业化：菜单栏+工具栏 · 双层时标水平化 · MPP 导入（2026-09-07）
> UI/UX 对标 MS Project/ProjectLibre 重排（用户点名「随意平铺不正规、时标应从左到右」）；**MPP 二进制导入**主刀（蒸馏 ProjectLibre MPXJ 只读零搬运，MPP9 真实样本逐字节实证）。绑定面 **595→596**（drift PASS@596）。详见 releases/v4.140.0.md。
- 命令区三行化：菜单栏（文件/编辑/视图/任务——导入 XML/Excel/MPP、导出 XML/Excel/图面、撤销重做、增删任务/分组、里程碑/检查器、四视图、示例/清空/工程管理 全量归属）+工具栏（高频命令分组竖线快捷区，导入/导出收拢下拉）+工程信息条（工程名/开工/目标竣工/日历/基线/资源 | 视图档位）；旧平铺零删减收编，工程切换器/管理/AI 栏/办公预览收进菜单栏右端。
- 双层时标：横道图时标自左向右水平排——上行月份跨列段（「2026年9月」居中，窄段降级「9月」/数字）、下行逐日号；根因=`.sched-gantt-datehead` 误用 `flex-direction:column` 致日列纵向堆叠。
- MPP 导入：`internal/schedule/mpp.go`——OLE2/CFB（mscfb）+CompObj 版本探测+Props/VarMeta/Var2Data/FixedMeta/FixedData 五原语；任务（名称/大纲/工期/里程碑/进度）、搭接（TBkndCons 四类型+有符号时距）、资源/分配、周工作制；口令保护 fail-closed；导入后 auto 模式按搭接 CPM 重排（约束日期/日历例外/费率不映射，提示如实告知）。
- 门禁 tsc/eslint 0、vitest **2458**（+3）、go +7（mpp 合成流表+真实样本门控）全量 116 包绿、drift PASS@596、版本三处 4.140.0。走查：三行命令区/时标水平/文件菜单/导入导出下拉/示例工程 DOM 断言+浏览器截图过。

## v4.139.0 · 进度计划多工程：每文件一工程+索引 · 切换器 · 管理动作 · agent 跟随（2026-09-07）
> **16 项差距清单最后一项 #15 收官**（v4.110 挂账清偿）。绑定面 590→595+Save 签名扩展（drift PASS@595）。详见 releases/v4.139.0.md。
- 每文件一工程：进度计划/*.gsched.json+索引 .gaea/schedule/index.json（缓存指针非权威，扫描重建+轻扫收编）；index.go 原子写/指针校验/安全 slug。
- 绑定 +5：Load 读索引 current、Save 显式 rel（消解切换竞态）、Projects 摘要、ProjectOpen 切指针、ProjectCreate/Archive/Delete（级联切剩余未归档）；agent 缺省 path 跟随指针（三工具零扩容）。
- 前端：store currentPath+openProject（冲刷→切换→重水合清历史，fail-closed）；页头切换器（单工程收窄零感知）+管理面板；办公卡「设为当前」复制改切指针。
- 门禁 tsc/eslint 0、vitest **2455**（+27）、go +15 全量 116 包绿、drift PASS@595、版本三处 4.139.0、build+冒烟过。走查：新建/切换/归档/删除级联/单工程收窄全过。

## v4.138.0 · 进度计划自定义包：自定义字段 · 编制说明注 · 项目模板 · 签证台账（2026-09-07）
> 差距 #14+PDF 余项全清+#10 模板自设计；16 项差距仅剩 #15 待拍板。绑定面 **590** 零变更。详见 releases/v4.138.0.md。
- 自定义字段：5 槽注册表（文本×3+数值×2）+列名项目覆盖+横道 20 列直编（缺省收起/改名铅笔入 undo）+Go xlsx 往返（有值槽出列、缺省名与覆盖名都认、非法数值静默丢弃）。
- 编制说明注：exportMeta.notes+导出弹窗多行输入+三图面图签上方说明块（≤6 行截断，空则输出逐字节不变）。
- 项目模板：本机 10 槽另存/载入（Popconfirm+可 undo）/删除；签证台账：基线面板一键 CSV（BOM+漂移列）。
- 门禁 tsc/eslint 0、vitest **2428**（+29）、go +4 全量绿、drift PASS@590、版本三处 4.138.0、build+冒烟过。走查：改名/填值/出图/模板/台账全过。

## v4.137.0 · 进度计划资源包：资源使用视图+超载检测 · 资源级日历 · 多基线槽（2026-09-07）
> 差距 #12/#13/#11 一版销三项。纯前端，绑定面 **590** 零变更（活跃基线指针口径不变，Go/AI 工具零漂移）。详见 releases/v4.137.0.md。
- 使用视图（第四档）：computeUsage 逐工作日负载 vs 可用性（工时资源；个人日历收紧可用性），UsageView 热力格+超载红标+tooltip 归因+汇总行；材料/成本不参与按天负载（诚实口径）。
- 资源级日历：SchedResource.calendar + 资源面板日历编辑器；个人休假只影响可用性与超载判定，不改任务排程。
- 多基线槽：baselines[] 上限 3 FIFO+活跃指针；BaselinesPanel 槽位切换/更新/删除/漂移摘要；槽动作入 undo 历史。
- 门禁 tsc/eslint 0、vitest **2399**（+31）、go 绿、drift PASS@590、版本三处 4.137.0、build+冒烟过。走查：超载 5 格归因/双槽切换/日历入口/空态。

## v4.136.0 · 进度计划小刀包：任务路径分析 · 任务检查器 · 排序与多级分组 · 导出标注旗标（2026-09-07）
> 一版销五项（差距 #8/#9/#16+PDF 余项 bar 标注/里程碑旗标，顺带大纲折叠）。纯前端，绑定面 **590** 零变更，机制蒸馏 ProjectLibre。详见 releases/v4.136.0.md。
- pathDriver.ts：逐依赖自由时差（复用 freeFloatPart），ff=0 且后继 auto=驱动依赖；drivingChain 沿驱动边 BFS；GanttView「路径｜前驱链｜后继链」——链上依赖线描红、链外行/线淡化。
- TaskInspector.tsx：右键「任务检查器」Drawer——六时参日期换算+「开始日期由谁决定」结论行+前驱/后继表（可点跳转、驱动 Tag）。
- ganttGroup.ts buildGanttRows：可见行序列唯一状态源（筛选→排序→多级分组→折叠，缺省恒等）；合成组头行+汇总跨度条；WBS 组头获得折叠；排序/分组持久化 chatPrefs。
- ganttExport barLabels 叶条尾「任务名（N天）」+三图面里程碑旗标+ExportDialog 开关（缺省开）。
- 门禁 tsc/eslint 0、vitest **2368**（+53）、go 绿、drift PASS@590、版本三处 4.136.0、build+冒烟过。走查：路径 4 边描红/折叠往返/分组 8+8/检查器结论行/标注开关。

## v4.135.0 · 进度计划刀C 收口：多级撤消/重做 · 任务筛选 · 行右键菜单 · 进度编辑入口（2026-09-07）
> 刀C 余项四件全清（官方「多级撤消/筛选视图/更新进度/上下文菜单」）。纯前端，绑定面 **590** 零变更。详见 releases/v4.135.0.md。
- store.ts：快照式 undo/redo（上限 50 不入持久化；**文本编辑 800ms 合并窗口**——同 kind+id 连续编辑只压一次栈；整体替换强制压栈；外部回读 setState 直改天然不入史）+工具栏按钮+Ctrl+Z/Y/Shift+Z（输入聚焦不抢）。
- ganttFilter.ts 两遍法行筛选（全部/关键/手动+文本；分组名命中强制展示整支，否则分组任一直接子孙可见才保留）；表格/条形/依赖线/前锋线同源联动，仅横道参与。
- GanttView 行右键菜单（添加任务/分组/设里程碑/删除，表格+条形两处）；ganttCols +进度列（15 列可隐藏，缺省隐藏 chatPrefs ganttHide=['progress']，存量数据回落缺省集）。
- 门禁 tsc/eslint 0、vitest **2315**（+14）、go test 绿、drift PASS@590、版本四处 4.135.0、build 2m10。走查：按钮状态机/筛选 20→18→2→20/右键里程碑+撤销/进度格组件级钉死。

## v4.134.0 · 进度计划刀D3：Excel 导入导出——上报口径往返（2026-09-07）
> 差距清单 #2（官方「Excel 导入/导出」项）。Go excelize，绑定面 **588→590**。详见 releases/v4.134.0.md。
- internal/schedule/xlsx.go：ExportXlsx（序号/WBS/任务名称/工期/开始/完成/前置，CPM fail-closed，分组跨度，前置 `2FS+3` 引用）+ ImportXlsx（表头别名归一识别、WBS 深度/空工期行分级、工期文本提取、前置引用大小写不敏感回链去重、日期仅取开工日、Validate fail-closed——排程交回 CPM 重算与 mspdi 同律）。
- 绑定 GaeaScheduleExportXlsx/ImportXlsx（base64）+ bridge/spaceBindings(+2 work，数量锁 303)/mock 直通/api.ts/SchedulePage「导入 Excel/导出 Excel」按钮。
- 门禁：go test 绿（+7 用例）、drift PASS@590、vitest 2301、tsc -b/eslint 0、版本四处 4.134.0、build 1m4。走查：导出回执+往返 20 行保名。

## v4.133.0 · 进度计划刀D2：网络图上报件 + AOA 逆推锚点修正（2026-09-07）
> 差距清单 #3（对标 .gzp 时标网络上报件）。纯前端，绑定面 **588** 零变更。详见 releases/v4.133.0.md。
- **引擎修正（aoa.ts）**：total（计算工期）在正推前取事件初始 es，长链下逆推整体平移出假负时差（实证 办公楼样例 min(ls) −111）且刀G AOA 侧 planFinish 锚点从未真实生效——total 移至正推后；+3 回归用例。
- **networkExport.ts**：双代号时标网络（工程标尺四行：工程日/月/日/星期+总工期红刻度，月界竖线，日宽 36~110px/天自适应）+ 单代号（六格盒/绑定红链/虚拟 S/T）；ganttExport 抽共享标题带/图签/调色板；ExportDialog 图面类型 Segmented（默认跟随当前视图）。
- 门禁 tsc -b/eslint 0、vitest **2301**（+10）、go test 绿、drift PASS@588、版本四处 4.133.0。走查：三图面 PNG/PDF 回执+AOA 标尺/波形/LS 诚实+PDM 红链贯通目检全对。

## v4.132.0 · 进度计划刀D1：图面导出——横道图上报件 PNG/PDF/打印（2026-09-07）
> 差距清单第一项（对标真实上报件+MS Project 官方「PDF/XPS 输出」）。纯前端，绑定面 **588** 零变更。详见 releases/v4.132.0.md。
- ganttExport.ts 纯函数构建器：标题带+上报 6 列表格+双行时标/周末底纹+关键红/汇总黑/里程碑菱形/时差尾/四型依赖线/竣工线/今日线+图例+图签（空值留白签章位）；fitText CJK 截断、XML 转义、循环依赖 fail-closed。
- imgPdf.ts 最小 PDF 封装（单页 PDF 1.4+JPEG DCTDecode 直嵌+xref 偏移对齐，零依赖）；exportArtifact.ts 管线（2 倍超采样 canvas→PNG/PDF/iframe 打印）；exportMeta.ts 签署字段持久化；SchedulePage「导出图面」弹窗三出口。
- 测试 TS +18（用例抓出 imgPdf 初稿偏移表错位）；门禁 tsc/eslint 0、vitest **2291**、go test 绿、drift PASS@588、版本四处 4.132.0。走查：PNG/PDF 真实管线出回执+图面 20 行 14 条 12 关键目检全对。ganttExport.ts 入 eslint hex-exempt（图面调色板是数据）。

## v4.131.0 · 进度计划刀C：工作台双栏——表格｜画布可拖分栏 + 列显隐 + AI 栏折叠入口（2026-09-07）
> 用户点名整改：14 列表格挤占画布/时间刻度竖排/AI 助手折叠。纯前端。绑定面 **588** 零变更。详见 releases/v4.131.0.md。
- 横道重构「表格窗格｜分隔条｜画布窗格」真双栏：分隔条拖动收纳表格（最小=行号+名称）、双击复位、纵向滚动同步（画布 scroll→表格体 translateY 直改 DOM）；列显隐菜单 12 列单列可藏（行号/名称固定）；ganttCols 纯函数+chatPrefs 扩展 ganttTableW/ganttHide 持久化。
- 时间刻度竖排修复：.sched-day-cell 补 nowrap+overflow hidden（dayW=8 两位日期折行元凶），<10px 档只画格不写数。
- AI 栏折叠第二入口进栏头（新会话旁 ‹，i18n +3 键）；刀11 页头按钮与持久化保持。
- 走查：拖窄 300→604 落盘+重挂载恢复、双击复位、全藏列钳 194；隐藏「最迟开始」宽 838；dayW=8 全格无字不竖排；AI 折叠/展开/持久化往返。门禁 tsc/eslint 0、vitest **2274**（+12，用例抓出 visibleCols 初版滤掉固定列真 bug）、go test 绿、drift PASS@588、版本四处 4.131.0。flaky 复证：ContextView+FilePreviewModal 两文件合跑互踩（各自单跑绿，既有环境 flaky 非本刀引入）。

## v4.130.0 · 进度计划基本功刀H：图例语言补全——标注归位 + 波形线 + 过桥法（2026-09-07）
> 审计渲染层三项缺陷（G2/G4/G1）销项，双代号时标网络补齐 JGJ/T 121 图例语言；单代号共用过桥。纯前端。绑定面 **588** 零变更。详见 releases/v4.130.0.md。
- G2：edgeSegs（edgeGeom 迁 aoaLayout 纯函数+段序列化）四类几何统一名称在箭线上、工期在下；波形存在时标注居中于实体段。
- G4：AOA auto 改真时标轴（x=最早时间×列宽，时距空洞成比例）；水平段超出实体工期终点的尾段画波形，虚工作整段波形，手动布局不画波形；图例新增波形线/标注口径两项。
- G1：findBridgeArcs 竖段垂直穿越他边横段处画半圆过桥（端距/间距护栏、端点相触不算）；PdmView 连线改段序列共用（首次拥有组件测试）。
- 走查（?mock=demo）：双代号 35 箭线 10 条波形全非关键、2 处过桥、20 组标注 0 错、27 节点真时标全对；单代号 2 处过桥、红搭接 7 条贯通。门禁：tsc/eslint 0、vitest **2262**（+21）、go test ./... 绿（零改动）、drift PASS@588、版本四处 4.130.0。新坑：图例小 svg 的 d 同样含 q/A，波形断言必须按连线类名取。

## v4.129.0 · 进度计划基本功刀G：引擎诚实化——FF 搭接口径 + 计划工期锚点（2026-09-07）
> 修审计确认的两处引擎缺陷：E1=freeFloatPart SS/SF 型漏减前置 ES 锚点（自由时差虚高，违反 TF=0⇒FF=0 定理）；G3=目标竣工作计划工期锚点进逆推（超期时负时差诚实呈现、关键工作=TF 最小集；manual 驱动场景维持经典口径）。TS/Go 镜像。绑定面 **588** 零变更。详见 releases/v4.129.0.md。
- E1：freeFloatPart 增 esFrom 参数并导出，四型公式单测钉死；实证 管道铺设 FF 11→0、沥青面层 2→0。
- G3：computeCpm/ComputeCpmPlan 增 planFinish 锚点（PlanFinish/planFinishOf=deadlineWorkdays 换算）；接线 SchedulePage/buildAoa/setBaseline/gschedSummary/mock analyze/Project.Analyze；AOA 事件逆推同步收紧、箭线关键=浮时最小。
- 测试：TS +11、Go +4（镜像）；实证 deadline 收紧后绑定链 TF=−2、关键集 8 项不变、状态栏「超 2 天」。门禁：tsc/eslint 0、vitest **2241**、go test ./... 绿、drift PASS@588、版本四处 4.129.0。坑复证：vitest 与 go test 并发跑必 flaky，门禁分开跑。

## v4.128.0 · 进度计划基本功刀F：单代号网络图合规整改（2026-09-06）
> 修「整网一条直线/无分支/关键线路不可辨」：布局从时标分层改拓扑分层（并行分支同列并列）、多起点/终点增虚拟 S/T 节点、关键线路=绑定中的临界搭接红色贯通、节点行号角标。引擎零变更，绑定面 **588** 零变更。详见 releases/v4.128.0.md。
- layerByTopology 纯函数（列=最长路径深度+三轮重心松弛；layerByTime 原样保留供 AOA 时标口径）；isLinkBinding 四型绑定判定（FS/SS/FF/SF 的关键搭接不再漏标）。
- 走查：16 节点 2 行/13 层级/红搭接 12 条贯通/S+T 虚拟节点在位/自动适配 30%。测试 TS +6；门禁 tsc/eslint 0、vitest **2230**、drift PASS@588、版本四处 4.128.0、build 冒烟 200。

## v4.127.0 · 进度计划基本功刀A+E+B：呈现层合规整改（2026-09-06）
> 双代号箭线正交画法合规（禁斜线）+ 视觉纪律（分组降饱和/对比度根修/图例吸顶/单代号缩放适配）+ 横道表格补齐 Project 口径（六时参列/前置后续引用/依赖线四型锚点）。引擎零变更，绑定面 **588** 零变更。详见 releases/v4.127.0.md。
- **刀A 双代号正交**：edgeGeom 重写（前进 H-V-H/同列竖直/后退下方通道；平行边错位 10px；标注随水平段归位）；走查 39 条箭线 0 斜线。
- **刀E 视觉纪律**：分组六色 16~18% 淡底+左色条（黑汇总条重见天日）；次级文字 outline（暗夜主题实为 15% 透明青——「看不清」元凶）统一切 on-surface-variant；坐标纸降强度；图例吸顶；单代号缩放/适配全图（16 节点自动 30% 一屏）。
- **刀B 横道补齐**：14 列（+最迟开始/最迟完成/总时差/自由时差/后续）；前置/后续 Project 式「3FS+2」引用文本（纯函数 fmtLinkRefs）；依赖线按 FS/SS/FF/SF 取锚点（纯函数 ganttLinkPath，含反向与短桩绕行）；手动任务时差诚实显 —。
- 测试：TS +8（ganttLinks 6+引用 2）；门禁 tsc/eslint 0、vitest **2224**、drift PASS@588、版本四处 4.127.0；?mock=demo 走查 DOM 断言+三视图截图复核。欠账：刀C 交互包（列宽/折叠/undo/进度入口/右键）、刀D 周/月刻度+打印导图。

## v4.126.0 · 本地模型使用：切换器模型粒度化 + 换模预估按目标模型全覆盖（2026-09-06）
> 办公模型切换器从引擎粒度到模型粒度；换模预估从「只支持 herdsman 且按错模型」到按目标模型覆盖全部本地引擎（宁 unknown 勿假 hot）。绑定面 **588** 零变更（GaeaModelSwitchEstimate 仅扩展签名）。详见 releases/v4.126.0.md。
- **刀1 切换器模型粒度**：GaeaModels 按引擎展开全部 Kind=llm 模型（本地组置前/ModelInfo 增 local·status·label/空清单回退 "(默认)"）；GaeaSetModel("engine/model") 先 SetDefaultModel 再切引擎（清单外报错 fail-closed）；前端分组+「本地」标+运行中徽标+展示名优先；附带修 GaeaModels nil-client 空指针风险。
- **刀2 预估诚实化**：herdsman 按目标模型查目录（修旧版只查 DefaultModel）；ollama 原生 /api/ps+/api/tags（:latest 归一，ps 挂 tags 活保守 cold）；modelhub 复用 ModelHubModelLoaded；未加载 cold 不假报秒数（前端「需要一些时间」）；i18n 三语 +2 键；mock 契约对齐。
- 测试：Go +新预估表驱动/GaeaModels/GaeaSetModel；TS +2；?mock=1 走查 DOM 断言六项全过（分组序/组名标/徽标/展示名/确认弹层/云端直切）。
- 门禁：tsc/eslint 0、vitest 2214→**2216**、Go 全量 0 FAIL、drift PASS@588、版本四处 4.126.0、build.bat 冒烟 200。

## v4.125.0 · 联动候选：摘要卡「设为当前计划」+ 拍板池三设计文档落档（2026-09-06）
> 非当前计划的 .gsched.json 摘要卡新增「设为当前计划」一键（复制为当前计划+板块即时回读，循环依赖不提供入口）；拍板池三项（diff 确认卡/双工期口径/多工程）并行子代理完成拍板用设计文档。绑定面 **588** 零变更。详见 releases/v4.125.0.md。
- 「设为当前计划」：走既有 ScheduleSave fail-closed 通道+notifyScheduleFileChanged 即时回读；i18n 三语 +4 键。
- 三份拍板文档：diff 确认卡（混合分级确认，零绑定可行已核实，4 刀）/双工期口径（durationUnit 标记，9 拍板项，4 刀）/多工程管理（每文件一工程+索引，绑定 588→590 或零新增备选，8 拍板项）。
- 门禁：tsc/eslint 0、vitest 2213→**2214**、Go 例行绿、drift PASS@588、版本四处 4.125.0、build.bat 冒烟 200。

## v4.124.0 · 资源成本刀2+刀3：agent 通道 + 板块 UI（2026-09-06）
> 并行双线（Go 侧/前端侧互不相交，子代理分头实施+主代理合并走查），绑定面 **588** 零变更。详见 releases/v4.124.0.md。
- **刀2 agent 通道**：ops 扩 upsert_resource/patch_resource/remove_resource（级联删分配）/set_assignments/patch_task.fixedCost；apply 回执强制带 totalCost+taskCosts；get 增 resources/assignments/costs；analyze 增成本叙事（Top5/byResource/未定价 finding）；compact 同步；技能「资源与成本」分节（锚点锁 23→31）。
- **刀3 板块 UI**：资源工作表弹层（单位显式标注/类型切换联动清理/删除级联）；甘特「成本」列（叶明细+分组滚动求和，无数据留空）；任务行资源 Popover（勾选建分配/三类字段/整体替换）；状态栏总成本段（无数据不显示）；removeTask 级联删分配（防 Validate 拒收）。
- 测试：TS +19、Go +18（ops 8/叙事 4/回执 4/技能 2）；?mock=1 全链走查贯通（录入→挂分配→成本列→状态栏→自动保存）。
- 门禁：tsc/eslint 0、vitest 2194→**2213**、Go 全量绿、drift PASS@588、版本四处 4.124.0、build.bat 冒烟 200。

## v4.123.0 · AOA 手动布局刀1：锚点化 + 拖拽布点 + 双模式开关（2026-09-06）
> 按 docs/gaea-schedule-aoa-manual-layout-design-2026-09.md 推荐拍板项实施：混合锚定/pins 内嵌/mode 不入文件/半格吸附/agent 不暴露布局。布局=展示层状态，buildAoa 引擎口径零变更。绑定面 **588** 零变更。详见 releases/v4.123.0.md。
- **锚点键**（aoa.ts）：AoaNode.anchor=事件业务身份（S/T/成员键字典序最小者），跨拓扑变更稳定，5 例单测钉死。
- **纯函数 aoaLayout.ts**：snapPt 半格吸附（55×37）/normalizeAoaLayout 结构归一/applyPins 混合共存/prunePins 提交剪枝。
- **持久化**：gsched.json 内嵌 aoaLayout.pins（TS↔Go 镜像；Go 结构校验 pass-through 坐标/键数；维持无 aoa.go）；旧文件零迁移。
- **AoaView**：布局 Segmented 自动/手动+重置布局；拖拽布点（window 三件套/未移动不提交/Esc 取消/自动拖拽=转手动/提交剪枝）；手动模式时间参数不受影响明示。
- 测试 +15（纯函数 9+组件 6）；门禁全绿（CostProjectsView 负载 flaky 单跑复跑绿入册）。

## v4.122.0 · 资源成本刀1：模型 + 成本 rollup 引擎 + 持久化（2026-09-06）
> 按拍板方案实施（docs/gaea-schedule-resource-cost-design-2026-09.md 推荐项生效：元/工日口径、fixed-units 简化公式、材料固定总量、分组行禁 fixedCost/分配）。数据层先行，行为对老用户零变化。绑定面 **588** 零变更。详见 releases/v4.122.0.md。
- **数据模型**：三类资源（work/material/cost）+ 分配（(taskId,resourceId) 唯一）+ 任务 fixedCost；SchedProject 增 resources/assignments——全 optional，旧文件零迁移可读。
- **成本 rollup 引擎**（cost.ts/cost.go 镜像）：work=工期×units×费率+每次使用；material=总量×单价+每次使用；cost=金额；rows 只含叶任务；CPM 循环→ok=false fail-closed；悬空分配跳过；金额到分舍入。
- **两道闸**：前端 normalize 补空数组+坏形条目逐条丢弃；Go Validate fail-closed（资源 id 唯一/类型合法/数值非负、分配引用存在+对唯一+分组行禁挂、FixedCost 非负）。
- 测试：TS +13（cost 10+normalize 3）、Go +14（cost 9+validate/roundtrip 5），镜像纪律同批场景同期望。
- 门禁：tsc/eslint 0、vitest 2166→**2179**、Go 全量绿、drift PASS@588、版本四处 4.122.0、build.bat 冒烟 200。

## v4.121.0 · 进度计划刀12：办公联动三件套——计划摘要卡 · 板块互跳 · AI 周报一键（2026-09-06）
> 办公板块与进度板块联动收口：方向拍板**不并板、做轻联动**（数据/AI 层已是「单资产、双入口、事件互通」终态，本刀补交互层）。绑定面 **588** 零变更。详见 releases/v4.121.0.md。
- **办公侧计划摘要卡**：文件预览打开 `.gsched.json` 显示摘要（总工期/工作/关键/搭接/里程碑 chips+开竣工日期+基线漂移+倒排校核+循环依赖告警），原始 JSON 开关保留文本视图，解析失败回落原文本视图。
- **板块互跳**：摘要卡「在进度计划中打开」；进度板块页头「办公侧查看」按钮反向跳办公并直开计划预览（store 预置，未挂载也落地）。跳转/周报仅对当前计划文件提供，其余 .gsched.json 只读摘要。
- **AI 进度周报一键**：摘要卡按钮把周报指令发进共享工作线程（运行中 Steer 插话/空闲落气泡 Submit，审批挂起禁用，失败 toast）。
- 纯函数 gschedSummary.ts（路径识别+摘要解析，+8 例）、ScheduleFileCard 组件（+7 例）、i18n 三语 +27 键（schedCard.*）。
- 门禁：tsc -b/eslint 0、vitest 2151→**2166**（+15，flaky 复跑全绿）、Go 零改动例行绿、drift PASS@588、版本四处 4.121.0、build.bat 冒烟 /api/health 200。

## v4.120.0 · 进度计划刀11：左栏细节完善——拖宽分隔条 · 偏好持久化 · 快捷指令条（2026-09-06）
> 刀10 双栏版面的细节收口（绑定面 **588** 零变更）。详见 releases/v4.120.0.md。
- **宽度拖拽调节**：对话栏与工作台之间 5px 分隔条（hover 显色），拖动实时调宽 320~680 钳位，松手持久化；**双击复位 420**。
- **偏好持久化**（schedule/chatPrefs.ts 纯函数，+5 例）：折叠态+宽度存 localStorage gaea.schedule.chatPrefs，字段级容错（坏 width 不拖累 collapsed、坏 JSON 回缺省、写失败静默）。
- **新会话二次确认**：Popconfirm「新建会话？当前线程仍保留在办公板块历史中」——防误触清屏。
- **常用指令快捷条**（4 chip，点击即发送、运行中禁用）：分析当前计划与关键线路 / 检查缺前置任务并推荐逻辑关系 / 生成本周进度汇报 / 评估能否提前 5 天竣工。
- **头部副标题**：「与办公板块共享同一条工作线程」常驻说明。i18n 三语 +6 键（schedChat.*）。
- 门禁：tsc -b/eslint 0、vitest 2146→**2151**（+5，一次全绿）、Go 零改动例行绿、drift PASS@588、版本四处 4.120.0、build.bat 冒烟 /api/health 200；?mock=1 走查：拖宽→钳位/持久化/重载恢复、双击复位、快捷 chip 直发（running 徽标亮起）、新会话确认流（弹出/确认/清空）全过。
- 走查环境备注：合成事件拖宽期间物理鼠标真实 mousemove 会混入（读数漂移属环境噪声，真实用户拖拽即其本人输入，非产品缺陷）。
- 欠账：双代号手动布局、资源成本（需先出设计方案）；拍板池：diff 确认卡/双工期口径/多工程；真机池：真机端到端+手感。

## v4.119.0 · 进度计划刀10：左 AI 对话 · 右工作台——双栏版面 · 共享工作线程（2026-09-06）
> 用户点名版面（绑定面 **588** 零变更）：进度计划板块改为左侧 AI 对话、右侧工作台双栏；与办公板块**共享同一条工作线程**——左栏聊的就是办公那条会话，schedule_* 工具审批/回执/证据卡全部同源。详见 releases/v4.119.0.md。
- **架构关键（store.ts 事件绑定提升恰好一次）**：gaea/lib/store 是模块单例，原 useController 每挂载订阅一次 gaea-event——keepAlive 树下 GaeaApp 与新对话 pane 同时挂载会双订阅，text/reasoning 增量重复 dispatch（气泡翻倍）。提升为模块级 ensureEventsBound（首个宿主绑定、卸载不退订：隐藏期间事件继续入库，回到板块即热态）；看门狗保持每宿主（只读校准+localCancel 幂等）。**回归测试钉死：双宿主挂载+流式增量→只入库一份**。
- **ScheduleChatPane**（schedule/ChatPane.tsx）：LocaleProvider+useController+Transcript+Composer+ApprovalModal+AskCard 装配；头部=标题+执行中脉冲徽标+新会话；schedule_apply done→notifyScheduleFileChanged 即时回读（与 App.tsx 刀5 同款）；composer disabled 门禁对齐（meta 未就绪/审批打开）。
- **SchedulePage 双栏**：左 420px 对话栏（可折叠，页头切换按钮）+右工作台（原横道/单代号/双代号+全部工具原样保留，零功能删除）；i18n 三语 +4 键（schedChat.*）。
- 门禁：tsc -b/eslint 0、vitest 2145→**2146**（+1 双宿主回归，一次全绿）、Go 例行构建绿、drift PASS@588、版本四处 4.119.0、build.bat 冒烟 /api/health 200；?mock=1 走查：双栏渲染（左 420px 对话+右 gantt）→左栏发消息（流式回合+工具卡+最终答案+输入框清空）→折叠/展开→**切办公板块同线程互认**（左栏发的消息办公对话区可见，往返保留）。
- 欠账：对话栏宽度拖拽调节、双代号手动布局、资源成本（需先出设计方案）；拍板池：diff 确认卡/双工期口径/多工程；真机池：真机端到端+拖拽/双栏手感。

## v4.118.0 · 进度计划刀9：横道拖拽调排程——拖移即定位锁定 · 右缘缩放改工期 · 直接操作闭环（2026-09-06）
> v4.110/112 欠账池收刀（绑定面 **588** 零变更）；「一表多图」的直接操作编辑补完——对话即排程（刀4~6）之外，鼠标也能改计划。详见 releases/v4.118.0.md。
- **语义钉死（诚实口径）**：拖移 auto 任务=**转手动并锁定开始到落点**（Project 拖动加约束同源）——auto 位置由搭接决定，拖完会被 CPM 拉回，转手动是唯一不骗人的做法；manual 任务拖移=更新 manualStart；右缘缩放=改工期（es 不动，auto/manual 皆可）；里程碑可拖移不可缩放；分组行不可拖。落点按**最近工作日**吸附（跨周末/节假日吸附相邻工作日，等距取左），Esc 取消、未移动=纯点击不产生修改。
- **drag.ts 纯函数（+6 例）**：workdayOffsets（工作日→自然日偏移反查表）/nearestWd（钳位+二分+等距取左）/dropToWd/resizeToDuration——像素位移全部经反查表换算，日历口径与 CPM 一致。
- **GanttView 交互**：条形 mousedown 拖移、右缘 7px 手柄（hover 显现）缩放；拖拽中条形虚线描边+浮动提示（「第 5 → 8 工作日（转手动）」/「工期 15 → 16 天」）；时差尾拖拽中隐藏防错位；提交走既有 updateTask→防抖自动保存→agent 可见链路；循环依赖（CPM 不通过）整体禁拖。
- **GanttView 首次拥有组件测试（+5 例，fireEvent 鼠标序列）**：拖移转手动锁定、缩放改工期不动模式、未移动无操作、Esc 取消、循环依赖禁拖。
- 测试：vitest 2135→**2145**（+11）。门禁：tsc -b/eslint 0、Go 例行构建回归绿、drift PASS@588、版本四处 4.118.0、build.bat 冒烟 /api/health 200；?mock=1 真浏览器走查：拖移预览→提交（条形 +60px 精确吸附、模式翻「手动」、tip 消失）→缩放（工期 15→16、模式不变、已落盘）。
- 走查坑入册：**overflow:auto 容器边缘的元素会被裁剪致 elementFromPoint 命中落空**（合成事件派发落点到 MAIN）——缩小刻度（zoom-out）让目标离开边缘即可，非产品缺陷。
- 欠账：拍板池余——对话式调整 diff 确认卡、双工期口径、多工程管理；非拍板余——双代号手动布局、资源成本；真机池——真机端到端+基线/倒排/拖拽手感（原生手势需真人复核）。

## v4.117.0 · 进度计划刀8：倒排工期——目标竣工约束 · 引擎可行性裁决 · 横道竣工线（2026-09-06）
> v4.111/112 欠账池收刀（绑定面 **588** 零变更）；对齐 Project「必须完成期限」：目标是引擎**裁决**合同/定额竣工约束的可行性，不自动改排程——压缩方案由 AI 建议、用户拍板。详见 releases/v4.117.0.md。
- **数据模型**（双侧同构）：SchedProject.deadline（YYYY-MM-DD 日历日期，开工日调整后语义稳定）；引擎按工作日换算 targetWorkdays（「第 N 个工作日竣工」，竣工日为非工作日自然回落；早于开工=0 必然不可达）。
- **TS deadline.ts（纯函数，6 例）+ calendar.deadlineWorkdays（3 例）**：checkDeadline 返回可行性/超期量（正=超期，负=富余）/关键工作清单（手动不计）。
- **Go 镜像 deadline.go + ops set_meta{deadline}（指针三态：nil 不动/空串清除/日期设置）+ Validate 格式校验**（+9 例，测试与 TS 同批场景）：非法长度拒绝；DeadlineWorkdays/CheckDeadline 与 TS 逐字段镜像。
- **agent 三工具接线**：get 带 deadline；apply 回执带 deadlineCheck；analyze 输出 deadlineCheck+checks 倒排发现（不可达=超期量+压缩对象；富余≤2 提示触线；富余充足不出发现不噪声化）。工具链测试：设 09-10 超 1 天→压缩 C 达标→清除。
- **板块 UI**：页头「目标竣工」日期输入（Tooltip 口径说明）；状态栏倒排段（第 N 工作日+超期红/富余绿/压线达成）；横道图紫虚线竣工线（画在竣工日列末=当日完工口径，超出画布不画）。
- **schedule-edit 技能编制流程第 3 步改写**（锚点锁 +3）：约束场景先 set_meta deadline，analyze 倒排校核裁决，manual 仅作个别任务定位手段。
- 测试：vitest 2126→**2135**（+9）；Go internal/schedule +9（含 ops set_meta deadline 三态）。门禁：tsc -b/eslint 0、Go 全量一次绿、drift PASS@588、版本四处 4.117.0、build.bat 冒烟 /api/health 200；?mock=1 走查：超期红（第 29 工作日超 121 天）/富余绿（217 工作日富余 67 天）/清除/重载持久化/紫线渲染全过。
- 欠账：进度计划拍板池余——对话式调整 diff 确认卡、双工期口径（定额日历天 vs 工作日）、多工程管理；非拍板——横道拖拽调排程、资源成本；真机池——真机端到端+基线/倒排 UI 手感。

## v4.116.0 · 进度计划刀7：基线对比——快照固排程 · 漂移量化 · 板块/agent 同口径（2026-09-06）
> v4.111/112 欠账池收刀（绑定面 **588** 零变更）；对齐 Project「设置基线」：把保存时的排程结果固化，之后每次调整量化「较基线」偏差。详见 releases/v4.116.0.md。
- **基线数据模型**（types 同构双侧）：SchedProject.baseline={name,savedAt,duration,rows}，行=叶任务快照 {name,es,ef,dur,critical}（工作日序号，口径=相对开工日偏移，不受开工日调整影响；分组行不入基线）。
- **TS 引擎 baseline.ts（纯函数，11 例）**：snapshotBaseline（循环依赖/无叶任务 fail-closed 拒绝）+ computeBaselineDrift（总工期漂移、推移/新增/移除、关键链进出 criticalGained/Lost；rows 只含有偏差行，移除行按 id 排序与 Go 镜像一致；单关键标记翻转也计推移）。
- **Go 镜像 internal/schedule/baseline.go + ops set_baseline/clear_baseline**（+8 例，测试与 TS 同批场景同批期望）：set_baseline 缺 savedAt 拒绝（引擎纯函数不取时钟，工具层盖时间戳）；clear 无基线报错（无事可做≠静默成功）。
- **agent 三工具接线**：schedule_get 带基线元信息；schedule_apply 回执带 baselineDrift 摘要（模型汇报先讲偏差）；schedule_analyze 输出 baselineDrift 全量+checks「较基线」发现（漂移≠0 才出现）。
- **板块 UI**：页头「基线」弹层（保存/更新/清除+漂移摘要+偏差行清单）；横道图基线条开关（灰描边细条垫行底，悬停看基线 vs 当前）；状态栏基线段（基线 X → 当前 Y 天（±N），拖后红/提前绿）。store setBaseline/clearBaseline 动作，文件化自动随存（零新绑定）。
- **schedule-edit 技能增「基线对比」节**（锚点锁 +3）：调整前建议 set_baseline 固化、回执/analyze 先讲偏差再讲措施、结构性重排后更新基线。
- 测试：vitest 2115→**2126**（baseline 11 例）；Go internal/schedule 21→32 测试函数（baseline 11，与 TS 同批场景镜像）+ 工具链用例 TestScheduleBaselineDriftChain（保存→调整→analyze 报 9→11（+2）→更新基线清零→清除后消失）。门禁：tsc -b/eslint 0、Go 全量绿（app 包 1 例 TempDir 清理竞态 flaky 复跑绿）、drift PASS@588、版本四处 4.116.0、build.bat 冒烟 /api/health 200。
- 欠账：进度计划拍板池余——对话式调整 diff 确认卡、双工期口径（定额日历天 vs 工作日）、多工程管理、真机端到端（真机池）；横道图基线滑条样式细化观察项。

## v4.115.0 · 进度计划刀6：推荐逻辑关系——auto_chain 确定性原语 · 无前置质检 · 报告模板（2026-09-06）
> 拍板池 P2 首件（绑定面 **588** 零变更）；「AI 产建议、引擎裁决」分工的编制侧落地。详见 releases/v4.115.0.md。
- **auto_chain 增量操作**（internal/schedule/ops.go，+1 例）：推荐逻辑关系的确定性缺省步——仅为无前置叶任务按 WBS 顺序补 FS 串联；分组行/已有逻辑/手动任务不动（手动=有意定位，忽略入边故不作链源）；无可补拒绝（无事可做≠静默成功）。
- **质检「无前置」发现**（schedule_analyze checks）：无前置搭接的叶任务计数（首叶=计划起点不计、手动不计），发现文案直接指路「先 auto_chain 补缺省，再按工艺逻辑 set_links 覆写例外」——与工具测试闭环验证（补后清零）。
- **schedule-edit 技能增两节**：推荐逻辑关系工作流（auto_chain 缺省→工艺例外 set_links 逐条覆写+给理由+可回滚）；分析报告四段模板（进度/风险/盲点/建议——Asana Smart Status 口径，建议给预期总工期收益）。锚点锁 +4。
- 测试：Go schedule 33 例（auto_chain 1+工具无前置闭环 1）；门禁 tsc -b/eslint 0、vitest 2115/2115、Go 全量绿、drift PASS@588、版本四处 4.115.0、build.bat 冒烟 /api/health 200。
- 欠账（拍板池余）：对话式调整 diff 确认卡、双工期口径（定额日历天 vs 工作日）、多工程管理、真机端到端（真机池）。

## v4.114.0 · 进度计划刀5：对话即排程——schedule-edit 技能 · agent 写板块即回读 · 工具卡摘要（2026-09-06）
> 刀4 地基上的智能层首刀（绑定面 **588** 零变更）；真机端到端（真模型生成整计划）入真机池。详见 releases/v4.114.0.md。
- **schedule-edit 内联技能**（internal/gaea/skill/builtins.go，+1 注册锁测试）：编制与调整纪律总纲——先读后写、「成功≠正确」回执核对、引擎裁决不绕校验、JGJ/T 121-2015 先分解后编网（WBS 两级→逻辑关系→工期→日历→project 通道写入→analyze 质检清零→汇报总工期/关键线路变化）、ops 增量调整口径（压缩非关键不缩总工期要点破）、工作日口径与手动模式锁定的适用边界。
- **agent 写计划→板块即时回读**：App 工具事件喂入——schedule_apply 成功回执触发 notifyScheduleFileChanged()（绕过 15s 轻扫；防抖/未保存守卫在 store 侧宁守勿冲），对话调计划板块秒级可见。
- **工具卡可读化**（tools.ts subjectOf/summarize + 用例）：schedule_apply 卡=「总工期 X 天 · 关键 Y 项 · N 处修改」、get/analyze 卡=总工期一行/质检计数，行首主体缺省回退「当前计划」；信封剥离兜底对齐 chart_gen 先例。
- 测试：vitest 2110→**2115**（tools 5 例+store.sync 全链 2 例——迁移→防抖自动保存→外部写回读，走 bridge mock+window.__mockScheduleFile）；Go schedule-edit 锚点锁 1 例。门禁：tsc -b/eslint 0、Go 全量绿、drift PASS@588、?mock=1 健康走查（无控制台错误/板块无回归）、版本四处 4.114.0、build.bat 冒烟 /api/health 200。
- 欠账：真机端到端（真模型一句话生成整计划→板块实时可见→对话式调整 diff 确认卡）、推荐逻辑关系、分析报告模板化、双工期口径（合成版拍板池）。

## v4.113.0 · 进度计划刀4：AI 原生通道地基——计划文件化 · agent 工具三件套（2026-09-06）
> 响应「进度计划必须依托 AI 平台」拍板；绑定面 586→**588**（+GaeaScheduleLoad/Save）。详见 releases/v4.113.0.md。
- **internal/schedule 新包（Go，28 例）**：前端 schedule 引擎的忠实移植——types/cpm（FS/SS/FF/SF+时距、手动/自动、环 fail-closed）/calendar（工作日↔日期互逆）/project（校验+临时文件原子写）/ops（upsert_task/patch_task/remove_task/set_links/set_meta 增量操作集）/analysis（关键链/近关键/里程碑叙事）。Go 与 TS 测试互为镜像，口径改动必须两侧同步。
- **agent 工具三件套（schedule_get/apply/analyze）**：计划文件=板块与 agent 共享资产。get 返回任务表+CPM 全景；apply 双通道（project 整计划 / ops 局部调整）——快照（StageBaseline）+证据卡（RecordChange tool=schedule_apply）+写根 confine，环依赖/悬空引用 fail-closed 拒绝、错误原文回传，回执强制带写入后总工期（「成功≠正确」verify 纪律）；analyze=质检 findings（孤立任务/空分组/无收口/缺里程碑）+关键链/近关键/里程碑。三工具入 Workspace.Tools()（workDir+写根绑定）。
- **板块文件化**：`进度计划/当前计划.gsched.json`（默认计划，多工程留后续刀）——GaeaScheduleLoad/Save 轻量 IO（无证据卡，UI 自动保存不刷 Journal）；前端 store 水合（文件为准，localStorage 自动迁移）+防抖 800ms 自动保存+focus/15s 可见轻扫回读 agent 改动；状态栏同步指示器（改动待保存/已保存 HH:MM/保存失败）。mock 同步走查态（window.__mockScheduleFile 钩子可模拟 agent 外部写）。
- **调研落档**：docs/market-research-2026-09-06.md（15 竞品原始稿 docs/research-2026-09-06/）——AI 产建议/引擎裁决双市场验证、对话式调整安全空白 gaea 自定义标准（diff 确认+快照回滚）、斑马 AI 11 项功能=analyze 需求清单、行内 AI 建议必须可全局关闭；勘误 JGJ/T 121-2015。
- 门禁：tsc -b/eslint 0、vitest **2110/2110**、Go 全量绿（schedule 28 例+工具 4 例）、drift PASS@**588**、分类锁 299→301、版本四处 4.113.0、?mock=1 全链走查（迁移→自动保存→agent 写入回读）、build.bat 冒烟 /api/health 200。
- 欠账（刀5 候选，见合成版拍板池）：对话式调整 diff 确认闭环、整计划生成三确认口径、推荐逻辑关系、分析报告模板化、双工期口径（定额日历天 vs 工作日）。

## v4.112.0 · 进度计划刀3：斑马 UI 对齐——六色分组横道 · 网络图统计牌 · 底部状态栏（2026-09-06）
> 视觉/交互对齐斑马进度真机参考截图；绑定面 586 零变更（纯前端展示层）。详见 releases/v4.112.0.md。
- **横道图斑马化**：分组行六色饱和循环（橙/青/粉/黄/蓝/绿实底+深色左条+▲组名前缀，全宽贯通表格与条形区，组行次级文字黑字对比度修正）；时间轴改**自然日列**（每天一列，周末/节假日底纹+表头红字呼应，CPM 仍按工作日计算、条形跨非工作日自然变宽）；新增行号列；开始/完成列按日历换算 M/D；月份标签 nowrap 防窄列折行；里程碑菱形黑化对齐斑马。
- **网络图统计牌**：单代号/双代号图例行右端红字大数字牌（总工期/关键工作/工作总数或虚工作）——初版画布右上浮层实锤遮挡首行节点，走查后改挂图例行（bounding box 断言与首节点零叠压）。
- **底部状态栏常驻**：视图/共 N 项工作·总工期 N 天（斑马口径措辞）/关键工作数/工作制+节假日/时间单位说明；工具栏分组竖分隔线（ribbon 感）。
- 走查（?mock=1 DOM+计算样式断言+截图）：三视图全过；走查新坑入册——vite 对同文件连续两次快速编辑会留下中间版 transform 缓存（后半编辑丢失），touch 强制重转换即恢复。
- 门禁：tsc -b/eslint 0、vitest **2110/2110**、Go build/test/vet 全量绿、drift PASS@586、版本四处 4.112.0、build.bat 冒烟 /api/health 200。附带补账：README 版本表补 v4.104.0~v4.111.0 八行（此前八版漏同步）。

## v4.111.0 · 进度计划刀2：Project/斑马对齐——工作日历 · 手动/自动模式 · WBS · MS Project XML 互通 · 前锋线（2026-09-06）
> 对齐清单五项全落（工作制/任务模式/WBS/mspdi 互通/前锋线）；绑定面 586 零变更（导入导出纯前端）。详见 releases/v4.111.0.md。
- **calendar.ts（8 例）**：工作日历纯函数——工作日序号↔日历日期双向换算（周末工作制+节假日例外、开工日非工作日顺延、3650 天扫描防呆）；工期/时距口径从自然日修正为**工作日**（刀1 口径修正），CPM 数值模型不变、三视图口径首次统一。
- **cpm.ts 手动/自动双模式（+3 例）**：manual 锁定开始（正推忽略入边）、逆推不回传约束（前置 LF 不被收小）、不标关键；后继自动任务仍以其 EF 为正向约束（Project 口径可用子集）。
- **mspdi.ts MS Project XML 导入导出（5 例）**：Calendars（DayType 1=周日..7=周六+Exceptions）/Tasks（WBS/IsSummary/Milestone/Duration ISO8601/Manual/PredecessorLink Type+LinkLag 十分之一分钟）务实子集；宽松容错 fail-closed；样例工程 roundtrip 逐项一致。真机 Project/斑马互开验证入真机池。
- **视图**：横道图日期轴工作日历化+WBS 编码列+模式列+**前锋线**（按完成进度取点连折线+检查日期线，DatePicker 可调）+手动条灰；工具栏导入/导出 XML（file input+Blob 下载，零绑定）+日历弹层（周工作制+节假日增删）；persist merge 兼容 v4.110 旧数据（自动补 calendar/mode 缺省）。
- 门禁：tsc -b/eslint 0、vitest **2110/2110**、Go 零改动例行回归绿、绑定 586 零变更、版本四处 4.111.0、build.bat 冒烟 /api/health 200。

## v4.110.0 · 进度计划一级板块刀1：CPM 引擎 + 横道图/单代号/双代号三视图自由切换（2026-09-06）
> 「一表多图实时联动」口径（对齐斑马进度）；绑定面 586 零变更（刀1 纯前端自治，localStorage 持久化）。详见 releases/v4.110.0.md。
- **引擎层纯函数 27 例钉死**：`schedule/cpm.ts` 关键路径法（FS/SS/FF/SF+时距、环检测 fail-closed、正逆推 ES/EF/LS/LF、总/自由时差、关键工作、里程碑 0 工期）15 例；`schedule/aoa.ts` 双代号转换（PDM→AOA 教材口径：唯一 FS(lag0) 安全合并免虚工作、其余按搭接类型自动插虚箭线+去重、一始一终、i<j 拓扑编号、事件时间正逆推、关键虚工作红虚线、时标分层布局）11 例；`schedule/layout.ts` 时标分层（列=最早时间+重心松弛）单双代号共用。
- **三视图**：SchedulePage 一级板块（工位，menuOrder 12）Segmented 自由切换。横道图=任务表（大纲缩进/行内编辑/前置 Popover 编辑器）+ 时间轴（月日双行/周末底纹/今日线/缩放/关键红条/分组汇总条/时差尾巴/里程碑菱形/搭接箭线）；单代号=六格节点（ES/工期/EF·LS/总时差/LF）+搭接标注；双代号=圆圈事件（编号+早迟时间）+实工作/虚工作箭线+关键线路红粗。
- **板块接线双侧**：Go builtins.go schedule manifest（SpaceWork）+CanonicalIDs；前端 manifests.ts+图标注册表 ScheduleOutlined、launcher 卡描述、PageRegistry 注册；boards 快照测试同步（菜单 12 板块）。
- 门禁：tsc -b/eslint 0、vitest **2094/2094**、Go board 包+全量绿、绑定 586 零变更、build.bat 冒烟 /api/health 200。

## v4.109.0 · pptx 真编辑刀1 数据层：GaeaPptxApplyEdit（run 级替换 · rPr 原字节继承 · 快照+Journal）（2026-09-06）
> 设计文档六项拍板全部按其推荐口径落地（技术路径 A=Go 自研同构 docxedit 等），绑定面 585→**586**（+GaeaPptxApplyEdit）。详见 releases/v4.109.0.md。
- **internal/office/pptxedit 新包**：slide 序映射（presentation.xml sldIdLst → rels → part 路径，放映顺序）+ DrawingML 段落解析（a:p/a:r/a:t/a:rPr 原字节区间；blocker=a:br/a:fld/a:tab；a:fld 内 a:t 无 a:r 包装的真实形状已覆盖——文本可见、进拼接文本、记 blocker 不可编辑）+ `LocateText`（按页码定位段落文本）+ `ApplyTextReplace`（页码+目标+替换，首个命中；跨 run 拼接可命中；新文本继承最后受影响 run 的 rPr 原字节=天然保格式；完全覆盖的非最后 run 整体省略；zip 条目序保留+临时文件原子替换，母版/主题/动画字节零扰动）。
- **App 层 GaeaPptxApplyEdit**：落盘前基线快照（rollback 目录，Verifier 通道 B 可复核）+ Journal 证据卡（Tool=pptx_apply，对齐 xlsx_apply；play 空间不落）+ 返回 GaeaPreview 新预览。
- **测试**：pptxedit 6 例（手工最小 pptx 夹具：slide 映射/跨 run rPr 继承/表格内命中/a:br+fld 拒绝/未命中越界/仅动目标页）。门禁：Go 全量绿、vitest 2068/2068、tsc -b/eslint 0、drift PASS@586、分类锁 299。
- 欠账：刀2 编辑面（大纲扩展+编辑面板）、备注页/图表文本边界外、`sz` 之外字体族等 rPr 细粒度保真以真机 PowerPoint 兼容性走查为准。

## v4.108.1 · ?mock=1 真浏览器走查：条件格式/看板/导图编辑三链全过 · mock 升级走查态（2026-09-06）
> 收「新交互组件刀必走 ?mock=1」欠账（v4.96 教训）；零产品缺陷，mock 层升级 + 走查结论入档；绑定面 **585** 零变更。
- **走查实锤（DOM+计算样式断言）**：①条件格式——金额 120.50 命中 >100 规则，计算样式=红底 rgb(255,199,206)+红字 rgb(156,0,6)+加粗 600（三属性覆盖基础样式）；②看板——4 泳道 4 卡渲染正确，拖卡片跨泳道 → XlsxSetCell 真实改 mock 工作簿 C2 → 预览回读泳道重算（数据驱动泳道：空泳道不保留，记 B2 后半候选）；③导图编辑——双击改名/Tab/Delete/保存条全链通，保存落盘=规范大纲（H1/H2/嵌套列表与设计一致），改名持久化。
- **mock 升级（走查基建）**：office.ts 静态样例升级会话内可变态（mockXlsxState/mockFileBodies）——XlsxSetCell 真实写内存工作簿、WriteFile 会话内记忆、Preview 保存优先回读；新增「阶段看板」board 视图样例、预算表阶段列+condRules 样例、docs/项目大纲.md 纯大纲样例、FileSearch 索引同步。
- **走查工具结论**：CUA 真实鼠标输入在后台 IAB 不达页面（document 级监听零捕获）——原生 HTML5 DnD/双击手势的手感验证仍需真人；处理链语义经合成事件（分 tick 派发，同 tick 连发会撞 React 批处理陈旧态——新坑入册）实锤。
- 门禁：tsc -b/eslint 0、vitest 2068/2068（mock 契约测试全绿）。

## v4.108.0 · 导图画布编辑 M2 自研 v1：改名/增删/拖拽挂载 · 规范大纲回写（2026-09-06）
> 已批刀序 M2，选型按蒸馏纪律定为自研（与 M1 自研树同构，不引 mind-elixir，决策可逆）；绑定面 **585** 零变更。详见 releases/v4.108.0.md。
- **纯函数层（lib/mindmapEdit.ts 新增）**：不可变树操作 addChild/addSibling/rename/remove/move（守卫=根不可删拖/不挂自身/目标在拖动子树内拒绝成环，文本去换行截 120）+ `serializeMindmapOutline` 规范序列化（root→H1、depth1→H2、depth≥2→嵌套列表），parse∘serialize 结构幂等（单测钉死）。
- **MindMapView 编辑态**：onSave 传入且纯大纲时启用——单击选中（带子节点节点保留 M1 折叠切换）、双击改名、Tab 加子/Enter 加同级/Delete 删、HTML5 拖拽挂载；脏态保存条（Ctrl+S/保存/放弃）回写规范大纲，FilePreview save(override) 复用既有 WriteFile+Preview 回读状态机。
- **防丢内容闸**：parseMindmapOutline 上报 skipped（未进大纲的非空行：段落/代码块/围栏标记）；混合内容或截断解析时编辑闸关闭并如实提示——回写会丢非大纲内容，宁禁勿损。未传 onSave 保持 M1 只读行为零变化（M1 测试原样全绿）。
- 测试：mindmapEdit 7 例（操作守卫/序列化/幂等）+ MindMapView 4 例（改名回写/增删/混合内容闸/只读兼容）。门禁：Go 全量绿（零 Go 改动例行回归）、vitest 2068/2068、tsc -b/eslint 0、drift PASS@585。
- 欠账：M2 后半（拖拽挂载的真机手感、节点折叠态在编辑后的保持策略）；真机走查待样例。

## v4.107.0 · 多维表 B2 首刀：看板视图（泳道=select 列 · 拖拽改值直编）（2026-09-06）
> 已批刀序 M1→B1→M2/B2 的余件；绑定面 **585** 零变更。详见 releases/v4.107.0.md。
- **看板视图（GbaseBoardView 新组件）**：.gbase.json 视图 `type:"board"` 放行（须带 groupBy=泳道列）——groupBy 列值分泳道、行=卡片（cardFields 首字段=卡标题、其余卡面行，缺省取前 4 字段），复用 B1 既有 filter/sort/colorRules 计算（applyGbaseView 同源）。
- **拖拽改值**：卡片拖到目标泳道 = 该行 groupBy 列单元格写值（「（空）」泳道=清空），走 XlsxSetCell 直编通道（与表格视图双击直编同口径；设计文初稿的 Plan→Apply 表述按实际通道修正成文）；原地放下不写盘；moving 期间禁用拖拽。
- **数据模型**：gbase.ts 放行 board 类型 + cardFields（去重截断 8）+ GbaseSheetModel.fieldCols（字段名→列号，回写定位）；gbaseMissingColumns 纳入 cardFields 缺列降级；未知 type 仍拒绝。
- 测试：gbase 3 例（board 解析/fieldCols/cardFields 失配）+ GbaseBoardView 4 例（泳道渲染/拖拽触发/原地不写/moving 禁用）。门禁：Go 全量绿、vitest 2056/2056、tsc -b/eslint 0、drift PASS@585。
- 欠账：B2 后半（字段类型面板/画廊/可选 validate 工具）与 M2（自研回写 vs mind-elixir 未决）待拍板；看板真机走查待真实样例。

## v4.106.0 · 办公 U4 缺口收口：xlsx 条件格式预览 · herdsman HS-obs 实测销账（2026-09-06）
> 收 U4 渲染证据盘点缺口清单 #2（Go excelize 读条件格式 + 前端渲染分支，零绑定）；绑定面 **585** 零变更。详见 releases/v4.106.0.md。
- **xlsx 条件格式预览（零绑定）**：后端 `xlsxpreview/condfmt.go` 经 f.Pkg 原始 XML 解析（workbook rels → worksheet cfRule → styles.xml dxfs），提取 CellIs 静态可判定子集（range+常量阈值+dxf 底色/字色/加粗）进既有预览 JSON（`condRules`/`condSkipped`，omitempty）；expression/色阶/数据条/图标集/引用单元格阈值等不可静态判定类型诚实跳过并计数。前端 `lib/xlsxCondFmt.ts` 纯函数判定（range 包含/数值文本比较/between 上下限无序归一/文本大小写不敏感/priority 小者整体胜出），XlsxPreview 单元格渲染命中即覆盖基础样式（Excel 语义）。
- **测试**：Go 2 例（Render 端到端提取含 dxf 样式/skipped 计数/无条件格式 omitempty；夹具注入真实 Excel 形状 dxfs——excelize 写侧只写 cfRule 不落 dxfs 表的局限实测成文）；前端 7 例（解析/覆盖/匹配/优先级胜出）。门禁：Go 全量绿、vitest 2049/2049、tsc -b/eslint 0、drift PASS@585。
- **HS-obs 实测销账（蒸馏规划 §六-1 回填）**：herdsman v0.6.4 `/props` 404=采样默认不可外部观测，无判定性实验设计，坐实不立项；`/api/benchmarks` 自报 `avg_ttft_ms=119`（vs §七实测真实聊天固定 2.8s）→ HS-obs-2 具象化为「bench 路径与聊天路径的网关/调度差异」，herdsman 侧观察。

## v4.105.0 · Model Hub 蒸馏 MH4 常驻预热 · ComfyUI 预热 CU1+量化档位 CU2（2026-09-06）
> 蒸馏 unsloth（规划 §三 MH4 + §六 CU1/CU2）；绑定面 584→**585**（+WarmComfyUI）。详见 releases/v4.105.0.md。
- **MH4 modelhub 常驻预热（Go）**：启动自动预载扩到 modelhub——活跃引擎就是 modelhub 且钉选默认模型+Key 已配置时，后台 StartModelHubModel 并轮询收敛到 loaded（5s/6 分钟上限），首聊免等冷加载。四重门控：显式武装位（Startup 置位，同 imageHubRuntimeArmed 先例）/ auto_preload 总闸 / 活跃引擎跟随（§四-4 显存互斥）/ herdsman 预载将在途时让路；预热前查 /v1/models 已含目标即跳过（§四-2 幂等语义未证实，从不重复 load）；Studio 不可达静默降级只记日志。
- **CU1 ComfyUI 绘梦预热（Go+前端，新绑定 WarmComfyUI）**：绘梦页首入触发一次 64×64/1 步空跑（复用既有 workflow 构建器，预热的就是下次真实生成要加载的模型文件），后台把默认模型预加载进显存——首图免几十秒惰性加载（§六-2）。门控：引擎=comfyui/模型已登记/ComfyUI 运行中/无生成在途/武装位+进程内一次闸；任何跳过返回 reason 静默降级，前端零弹错。
- **CU2 fp8 量化档位引导（纯前端）**：comfy 模型 ↔ checkpoint 文件名映射（与 Go 构建器同源锚定），fp8/scaled 判定为量化高效档——krea2 显示「fp8」徽标（显存省/加载快）并置顶；bf16/无标记模型诚实不猜。
- **CU3 显存释放对偶维持观察项**：真机前置确认未做（本轮 ComfyUI 未运行，连接拒绝实锤），POST /free 端点支持性未证实前不动代码。
- 测试：Go 新增 10 例（MH4 六分支+端到端收敛 6 例、CU1 空跑三要素/未知拒绝/门控/超时有界 4 例）；前端 +2 例（CU2 判定与稳定置顶）；分类锁 297→298。门禁：go vet/全量绿、vitest 2042/2042、tsc -b/eslint 0、drift PASS@585。
- **真机口径如实记录**：本轮 unsloth Studio（8888）与 ComfyUI（8188）均未运行（连接拒绝），MH4/CU1 的 Studio 侧真机端到端验证待服务可用时补（httptest 假端已覆盖契约面）。

## v4.104.0 · Model Hub 蒸馏 MH1–MH3：采样让位+默认关思考 · 加载状态收敛 · UD 档位引导（2026-09-06）
> 蒸馏 unsloth（docs/gaea-unsloth-modelhub-distill-plan-2026-09.md §三刀序 MH1→MH2→MH3）；绑定面 584 零变更。详见 releases/v4.104.0.md。
- **MH1 采样让位（Go）**：modelhub 引擎下用户未显式配置 temperature 时不再兜底 0.7（omitempty 丢弃不传），把按模型自动采样调优让给 Studio；其余引擎维持 0.7 兜底（`internal/ai/client.go` ChatStreamChunks 单点收敛，copilot 显式温度不受影响）。
- **MH1 思考默认关（Go）**：modelhub 请求默认显式 `chat_template_kwargs.enable_thinking=false`（实测 unsloth 默认开思考，token 全烧 reasoning、正文 0 字）；显式开启时传 true 并抬 max_tokens≥4096 守护。只走 A/B 实验证实的 kwarg 通道，不发顶层 enable_thinking。
- **MH2 加载状态收敛（前端）**：一键加载 stopped 模型后不再谎报「已启动」——挂「加载中」徽标（按钮 loading+禁用）轮询模型列表直到 running（5s 一拍/3 分钟上限），成功才设默认模型；超时诚实提示（Studio 冷加载实测 50s+）。
- **MH3 UD 档位引导（前端）**：modelhub 组 UD-* 动态量化变体稳定置顶 + 「推荐」徽标（title 说明：关键层保高精度/体积小/加载快/显存省）；手动置顶优先于自动推荐（排序组合语义有测试锚定）。
- 测试：Go 新增 `client_modelhub_test` 3 例（temperature 省略/透传、思考开关与预算守护、非 modelhub 0.7 兜底守护）；前端新增 6 例（UD 判定/置顶/组合排序 + 推荐/加载中/作用域守护）。门禁全量：Go 全量绿、vitest 2040/2040（dot 复跑；首跑 28 例负载 flaky 复跑通过）、tsc/eslint 0、drift PASS@584。
- 顺带收 v4.103.0 产物欠账：`build.bat` 冒烟 + SHA256SUMS（v4.103.0 exe 无法从 HEAD 产出，欠账转为 v4.104.0 产物一步到位）。

## v4.103.0 · 办公搜索对齐 unsloth：web_search url 直取整页 · 结果域名策略过滤（2026-09-05）
> 蒸馏 unsloth 搜索工具行为（`clones/unsloth`，`tools.py 的 _web_search`/`web_access_policy`）；绑定面 584 零变更。详见 releases/v4.103.0.md。
- **web_search 新增 url 直取整页模式**：复用 web_fetch 的 SSRF/域名策略/HTML→文本 `doFetch`，搜索→取全文→引用收敛为一次调用（对齐 unsloth direct URL fetch；url 优先）。
- **结果域名策略过滤**：`[search] allow_domains/deny_domains` 作用于搜索结果（`filterSearchResults`，deny 优先/allow 非空须匹配/丢弃非法非 http(s) 结果）；策略受限时过度抓取（`topK×3`）再过滤，避免“前几条被拒→结果空”。
- **日期钉定 + 引导取全文**：url 结果头部注入 `as of <date>`；`Schema`/`Description`/`compact` 同步（去掉 `required:["query"]`、新增 `url`）。
- 测试：新增单测 6 例（`websearch_policy_test`，域名过滤/受限判定/非法 url）+ 联网端到端实测（`websearch_live_test`，`GAEA_LIVE_TEST=1` 才出网，默认 SKIP）；Go build/vet/builtin 全包 test 绿（绑定 584 不变）。
- 门禁：本轮纯 Go 内部改动、未触及前端与绑定；vitest/tsc/drift 未实跑（按惯例发布前跑全量；理论上 drift PASS@584）。

## v4.102.0 · Hub 落库 + 围栏 slot 口径统一 · 图像域 17 绑定转正 · Model Hub mock（2026-09-05）
> Model Hub（Unsloth）并行会话线完结落库（6cd891df）后解禁的收尾三线；绑定面 584（+3 为 Hub 线绑定，本刀零新增）。详见 releases/v4.102.0.md。
- **附带修复**：v4.101 提交事故补录——第一批 git add 因错误路径整批失败（报错被吞）、第二批漏列围栏接线 5 文件 + 2 测试文件，已补提交（efe80310）；教训入册（多步 add 后必须 git diff --cached --stat 对照预期清单核数）。
- **GenUI 审计 §8-5 resume 槽位口径统一**（最后一项遗留，跨层收口）：实时 assistant 取号优先事件日志序（assistantSlot，logSeq 缺省回退计数器=旧后端逐字节兼容）；resume/回退/启动三路装载统一「GaeaResyncEvents(0) 折叠快照优先（id=日志序）+ History+rebuildHistoryItems 保底」——genui 交互状态跨重启可命中（旧 h<index>/旧 seq key 任其 LRU 逐出不迁移）；空数组=空会话短路。
- **图像域 17 绑定转正 AppBindings/spaceBindings**（GenerateFreeImage/CancelImageGeneration/GenerateMedia/GenerateDiagram/GetImageBackendInfo/GetPortraitConfig/SetPortraitConfig/GetComfyUIStatus/GetComfyUILoras/GetComfyUITaskProgress/StartComfyUI/StopComfyUI/GetSystemStats/OpenImageSaveDir/OpenNovelImagesDir=play，GetCharacters/SetCharacterPortrait=shared；分类锁 280→297）；api/image.ts 全部直调迁 appFacade 三态回退（wailsjsCompat import 清零）；dev mock 补 mock/imagegen.ts 17 方法（查询中性空态/动作诚实失败/GetCharacters 最小样例）+ 契约测试。
- **Model Hub / OpenCode Key dev mock 补齐**：SetModelHubKey（内存态联动）/GetModelHubKeyStatus（脱敏对齐 maskKeyStatus）/StartModelHubModel（无本地引擎诚实失败）+ OpenCode Go/Zen Key 四方法同族欠账销账 + 契约测试；?mock=1 modelcenter 全区块零 is-not-a-function 报错。
- 门禁：Go 全量 0 FAIL、tsc -b/eslint 0、vitest 256 文件/2034 用例全绿（净增 2 文件/10 用例）、drift PASS（584）、?mock=1 模型中心走查 CLEAN。

## v4.101.0 · 三线并行：GenUI 围栏 Go 侧收口 · 画室模型目录 · 深检误报缓解（2026-09-05）
> 「继续」惯例三线并行（避开并行会话 Model Hub 线禁区）；**绑定面 581 零变更**。详见 releases/v4.101.0.md。
- **GenUI 围栏纪律 Go 侧收口**（审计 §8 六项落地，第 5 项 resume 槽位口径另行排期）：新共享剥离 helper internal/gaea/genui/fence.go（行扫描语义照搬前端 splitGenuiFences，围栏折叠为 `[genui 组件]` 占位、正文逐字保留）；做梦 dreamInput / BuildCompactSummary pendingItems / whisper companion_reply 摘要三处剥离（JSON 不再占 1500 字符窗口与入记忆）；压缩 summarySystemPrompt 增补「UI 围栏 JSON 不原样收录」指引；办公面板 append 去重键改 spec 指纹（resync 不再重复追加）；会话删除按 stateKey 前缀清理 genui 交互状态与面板（App 删除点接线）。
- **画室模型目录**：imagegen 新增「模型目录」tab——读 HerdsmanModelCatalog（mock 契约不变），按目录能力字段分能力族（生图/改图/视频/识图），状态/参数量/量化/文件大小/运行时等实有字段展示；目录无档位/成本字段→「未定价/目录未标注」不伪装（T0 口径）；当前生成台模型「当前使用」高亮（创作语境，不做管理功能）。
- **小说深检误报缓解**：文本归一化（全半角/空白/包裹符）+ 角色别名归一（名单推断+称谓剥离，歧义即放弃）+ 告警置信度/原因分类（冲突/疑似/提示三档 UI，措辞差异与粗粒度时间降级不吞真冲突）+ 单条忽略记忆（项目级 localStorage 上限 500，横幅保持可见可恢复）。
- 门禁：Go 全量 0 FAIL（含并行会话在途 Model Hub 改动如实注明）、tsc -b/eslint 0、vitest 254 文件/2024 用例全绿（净增 4 文件/31 用例）、drift PASS（581 口径）、?mock=1 模型目录走查 PASS。

## v4.100.0 · 三线并行：画室 T1 余件 · 办公 U4 前半 · GenUI 围栏审计（2026-09-05）
> 「继续」惯例三线并行（文件所有权互斥子代理，避开并行会话 Model Hub 线禁区）；**绑定面 581 零变更**。详见 releases/v4.100.0.md。
- **画室 T1 余件**：①识图「读/懂」画室试用入口（粘贴/选图→SavePastedImage 落盘→GaeaRecognizeImage/GaeaOCRText，原语 Tag+诚实错误原文+localStorage 最近 5 条，零新绑定）；②画室「创作资产」面板（角色槽 CharacterList 分页带出立绘摘要、模板槽复用绘梦模板库、近期作品读 ledger；选中经既有 applyTemplate/applyRefCharacter 回填生成台，不硬拼）。**走查实锤并修复**：三视图 tab 互斥缺陷（onClick 只开不关，aria 双 selected 且内容不切换）。
- **办公 U4 前半**：①渲染证据通道盘点 docs/gaea-u4-render-evidence-inventory-2026-09.md（6 通道对照：soffice 技能源/Verifier 成对缩略/GaeaPreview 直读（缓存键含 mtime 纳秒自动失配重转）/Preview 探测/ReadFile/排除项；结论=写后反馈首选 GaeaPreview 单口刷新零绑定）；②写后预览实时跟随（写类 result 成功→800ms 防抖→isOpen 判定→paneTabs.reloadTicks 刷新总线→FilePreview 静默重载不清屏不重挂、DocxPreview 保滚动位且同文件不作废修改队列；协同 U2：刷新绝不置前、已关闭不刷新不弹、编辑态有草稿跳过；「已自动刷新」badge 三语）。
- **GenUI 记忆/压缩围栏审计**：docs/gaea-genui-memoryfence-audit-2026-09.md 五链路逐条结论（压缩/子代理投影/回放/导出=原样无害或防线有效；记忆链 3 个 Go 侧低风险吸收点登记待裁决）；**genui 内修复 1 项**：password 输入 action 只回传长度信号不再回传明文（堵「交互值→会话日志→做梦记忆链」泄漏）。
- 门禁：Go 全量 0 FAIL（树内含并行会话在途 Model Hub 改动，drift PASS@584 其中 +3 为其未提交绑定，本刀口径 581 零变更）、tsc -b/eslint 0、vitest 250 文件/1993 用例全绿（净增 2 文件/39 用例；首轮 1 例负载 flaky 复跑绿）、?mock=1 走查 PASS（创作资产三槽/识图试用全要素/三视图互斥修复验证）。

## v4.99.0 · 三线并行：图像域 T1 收口 · 办公 U2 回合投影 · GaeaListDir 硬化（2026-09-05）
> 「继续」惯例三线并行（文件所有权互斥子代理）；**绑定面 581 零变更**（ImageHubAssets/ChapterArtList 前端转正 AppBindings，Go 侧不变）。详见 releases/v4.99.0.md。
- **图像域 T1 收口**：①GenerateDiagram 登记接线（media.diagram 第二试点：成功后 .mmd 落盘 .gaea/uploads + 写 Asset Ledger，受 Startup 武装位闸控，失败只 warn，model/cost 诚实留空）；②绘梦任务中心素材库 tab 轻扫（激活重读 + 30s 可见性门控轮询，缩略按 path 缓存）；③画室素材库独立页 AssetLibrary（grid 缩略/懒加载分页/原语与来源筛选/点选溯源详情/play+work 双空间切换，零新绑定）。
- **办公 U2 回合投影与审阅收口**（蒸馏规划 §4.2，绑定面 0）：①officeTurnProjection 前端纯函数（tool_dispatch/result 按 callId 配对，写入/验证/生命周期三类归约，失败不提交转换，乱序/孤儿/重复容错）；②统一 Office 回合卡 draft/ready 徽标（DeliverableCards/VersionTimeline 同判定函数；登记表/证据链/置前行为不变，regress 锁原样全绿）；③预览浮窗语义状态机（写弹读不弹、关闭优先不复活、写意图跨回合、终态清理，App 接线+挂载/恢复只建基线）；④草稿轻量版（证据链首写=draft、Plan→Apply 批准=ready，不新增实体与按钮）。
- **Go 小刀 GaeaListDir 硬化**（v4.96 登记）：补 IsAbs 分支 + 结构化错误码 GAEADIR_NOT_FOUND/NOT_DIR/READ_FAILED（签名 []DirEntry→([]DirEntry, error)，osStat/osReadDir 注入缝使权限用例全平台确定性）；前端 verifyArtifacts 切「错误码优先+文案匹配兜底」，四态诚实降级语义不变。
- **wailsjsCompat 消费族欠账部分销账**：ImageHubAssets/ChapterArtList 转正 AppBindings/spaceBindings（shared/play，分类锁 278→280）；api/image.ts 三态回退（window.go.app.App ?? bridgeApp）；dev mock 补 imagehub 中性样例（生图/图示/视频三形态，未定价诚实留空）。
- 门禁：Go 全量 0 FAIL、tsc -b/eslint 0、vitest 248 文件/1954 用例全绿（净增 3 文件/56 用例）、drift PASS（581）、?mock=1 真浏览器走查（AssetLibrary 筛选/详情/空间切换/返回 + 任务中心素材库 tab）通过。

## v4.98.0 · 图像域 T0 契约落地：登记底座 · 素材库 · 参考槽 v0（2026-09-05）
> 图像能力域第一步（设计 docs/gaea-image-domain-t0-contract-design-2026-09.md）；**绑定面 579→581**（+ImageHubAssets/ChapterArtList 只读）。详见 releases/v4.98.0.md。
- **T0 契约与试点**：五原语能力注册表（识图-读/懂、生图、改图=未实现 fail-closed、图示）+ Asset Ledger v1（按空间 append-only JSONL，play=.gaea/play/imagehub/assets.jsonl，只存路径+溯源元数据，坏行容错、上限折叠不删文件）+ 模型中心目录只读静态视图（档位/成本，未知模型诚实留空）；三试点接线：书封/绘梦落盘登记（失败只 warn 不拖垮生成）、GaeaOCRText/GaeaRecognizeImage 入口经注册表校验（行为不变）。
- **T1 提前量**：章节配图落盘 play exports + 登记 + 项目清单（.gaea/play/art/chapter-art.json，ChapterArtList 只读取）；绘梦任务中心新增「素材库」tab（读 ledger 缩略卡：模型/成本/来源/时间）；角色剧照保存后进溯源（characterlib）。
- **T2 参考槽 v0**：绘梦左栏选角色库角色 → 自动取参考图（≤4 张，首张作图生图种子）载入图生图；ComfyUI 文生图带参考自动转图生图，ipadapter/pulid 诚实报错未实现；GLM/开放端点不支持参考图时诚实拒绝；character_id 随生成进登记。
- **办公纪律两刀**：U1 工具错误码——format_convert 全出口改 `Error [FORMAT_*]` 结构化错误（INVALID_ARGS/SOURCE_MISSING/UNSUPPORTED/CONVERT_FAILED/OUTPUT_WRITE_FAILED），模型按 code 路由恢复；内置 office-edit 技能（先读后写、写后回读验证、宁拒不误改、逐 run 编辑防丢格式、soffice 渲染取证、错误码路由、.gbase.json/思维导图纪律）。
- **修复**：图像域登记运行态闸测试进程污染——闸改为 `App.Startup` 显式武装位（原 gaeaCfgSnapshot()!=nil 在 app 包测试恒真，登记写进源码树 internal/app/.gaea/），测试进程恒不落盘，附回归钉。
- 门禁：Go 全量 0 FAIL、tsc -b/eslint 0、vitest 245 文件/1898 用例全绿（净增 4 文件/38 用例）、drift PASS（581）、冒烟 200；v4.97.0 tag 树隔离 worktree 验证（Go/tsc）补做通过。

## v4.97.0 · GenUI 蒸馏：回答即 UI · 办公会话面板（2026-09-05）
> 蒸馏 dsh-genui（MIT，快照 680693e）；**绑定面 579 零变更**。详见 releases/v4.97.0.md。
- **共享内核 frontend/src/genui**：34 种白名单组件（布局/展示/轻图表/代码/交互/quiz），
  守卫（未知 type 丢弃、节点 200/深度 8/字符串/表格/选项上限）、流式部分解析、
  状态按会话+内容指纹持久化（LRU 200）、本地判卷/排序/折叠零往返、action 300ms 防抖、
  password 永不落库/不收集。
- **双板块渲染缝**：办公 MemoMarkdown/Markdown、聊天 ChatMarkdown/MarkdownContent 均
  识别 ```genui（兼容 dsh-ui）围栏；action 办公走 steer/send、聊天回合结束后追问。
- **模型侧**：内置 genui inline 技能（词汇手册 embed）+ genui_validate 只读工具；
  办公系统提示词指针、聊天 plain 与轻语人格统一注入规则（人格带门控：仅复盘/清单/数据
  回顾类使用）。
- **办公 UI 面板**：panel:true 规格经 Markdown 缝投递右栏第六 Tab「UI」；
  genuiPanel store REPLACE/APPEND、来源+内容指纹去重、会话隔离；/panel open|clear|<指令>。
- 门禁：vitest 241 文件/1860 用例全绿（净增 35）、Go genui/skill/tool·builtin/boot/app/
  whisper 全绿、tsc -b/eslint 0、drift PASS（579）、npm run build 通过；
  办公验收走查以 jsdom DOM 等价路径覆盖（真机视觉与真模型端到端待用户复验，如实记录）。


## v4.96.0 · 三线并行：下钻联动收尾 · Verifier 逐页缩略图 · pptx 真编辑设计（2026-09-05）
> 「继续」惯例三线并行；**绑定面 579 零变更**。详见 releases/v4.96.0.md。
- **刀A 三级下钻收尾**（调研中期候选第 4 条，四条全落地）：TokenCard「查看趋势」入口 → jumpToTrend tick 锚点（与 brief 跳转同款机制）→ 趋势卡 scrollIntoView + accent 强调 3s；不碰选中态；contextview.* +2 键 ×三语。
- **刀B Verifier 逐页缩略图**（老账）：「视觉复核」行内 before/after 逐页缩略卡（页码标注/缺页诚实占位/懒加载/并发≤4）；零绑定变更——verifyArtifactRelPath 标记截取 + Preview 探测定性绕 GaeaListDir 无 IsAbs/吞错缺口（Go 小刀候选登记）；四态诚实降级；deliverPanel.* +12 键 ×三语。**走查实锤修复两个 jsdom 测不出的真缺陷**：setState updater 内副作用被父级轮询竞争吞掉（副作用移回事件处理器）、liveRef 只写 cleanup 在 StrictMode double-invoke 下恒 false（改 effect 本体重置标准写法）。
- **刀C pptx 真编辑设计文档**（待拍板，不动代码）：现状=docx 自研字节级 OOXML 手术/xlsx excelize 直编/pptx 编辑为零且格式无修订制；推荐 Go 自研 pptxedit 与 docxedit 同构（排除 gooxml AGPL/unioffice 商业）；分期四刀；待拍板 6 项成文 docs/gaea-pptx-edit-design-2026-09.md。
- 门禁：Go 全量 0 FAIL、tsc -b/eslint 0、vitest **1825/1825**（净增 39；一轮 31 败为负载 flaky 漂移，复跑全绿——如实记录）、drift PASS（579）；走查：缩略图全链路 PASS（6 卡成对+页码标注）、Token 卡→趋势定位+强调 PASS。


## v4.95.0 · 三线并行收欠账：版本对比入统一查看器 · README 内嵌 HTML 白名单 · 子代理歧义选择器（2026-09-05）
> 「继续」惯例：欠账池挑互不相交线并行子代理；**绑定面 579 零变更**（纯前端三线）。详见 releases/v4.95.0.md。
- **刀A docx/xlsx 对比迁移 ChangesDiff**（v4.87 成文计划最后一项）：DiffRow 增可选 marker 列（docx 段号/xlsx 单元格 ref，pairModifications 整行 spread 防丢）；VersionTimeline 对比体改经 ChangesDiff——xlsx 每 sheet 一个 hunk，change 单元格→相邻 del+add 对（自动获得改蓝配对+字符级高亮），formula 后缀/截断提示/diffstat/展开全部全保留；ChangesPanel/GitPanel 零变化。
- **刀B README 级内嵌 HTML 白名单**（3b 遗留收口）：rehype-raw + rehype-sanitize 显式 schema（46 白名单标签/27 strip/className 按值放行 KaTeX 与 language-*）；协议收口走渲染层（mdUrlTransform+classifyExternalLink，schema.protocols 浅合并坑实测踩中）；文件预览链专用，MemoMarkdown 聊天流不动；收口修正 GFM 任务列表 checkbox 误伤（input 按值受限白名单）。
- **刀C 子代理歧义选择器**（老账）：matchRunningCandidates 纯函数 + 歧义两槽位（resolver/handler）；ToolCard 歧义卡可点（aria「多个运行中任务，点击选择」）→ App 居中选择器人工挑一个跳转；宁缺勿错不变（绝不自动跳，0/1 候选与现状一致）；taskpick.* 三语 +7 键。
- 测试：vitest **1786/1786**（净增 28；ContextView 全量负载三连 5s 超时与刀零接触，文件级放宽 20s 后全绿——处置成文）、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（579）；?mock=1 README 内嵌 HTML DOM 断言六项过（白名单渲染+消毒剥净），VersionTimeline/taskpick 以单测为验收面（A1 先例）。新增依赖 rehype-raw/rehype-sanitize。

## v4.94.0 · Agent 网络会话跳转：子代理上下文按会话查看（2026-09-05）

> 双源蒸馏规划阶段二.5 2.5e 后半收口；**绑定面 578→579**（+GaeaSubagentContextView）。详见 releases/v4.94.0.md。
- **Go**：GaeaSubagentContextView(sessionPath, ref)——校验 sa_ ref → 定位 <会话目录>/subagents/<ref>.jsonl → ReadLogRepaired 读入 → contextview.FoldTimeline 折叠（与主上下文同一管线）；ref 非法/缺失诚实报错或空快照。
- **前端**：ContextView 增 fetchTimeline 可选数据源（缺省仍走当前会话）+ onViewSubagentContext 回调透传 AgentRadial；AgentRadial sa_ 节点渲染「查看上下文」可点入口（回调在场才渲染）；ContextModal 增 fetchTimeline/title 透传；App 接线——主上下文页径向图节点「查看上下文」→ 打开该子代理的上下文弹层（标题带 ref）。
- 测试：Go +1（合法 ref 折叠/缺失空快照/非法 ref 报错，-count=2 绿）+ AgentRadial +2（入口渲染与回调/无回调不渲染）；vitest 231/1758、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（579）。

## v4.93.0 · 失败子代理恢复入口 + diff 行内语法着色（2026-09-05）
> 双小刀合并版（每版 1-2 刀惯例）；**绑定面 578 零变更**。详见 releases/v4.93.0.md。
- **刀A 失败恢复入口**（调研回填 opencode「失败≠终点」）：SubagentThread 失败态新增恢复提示条（status=failed 时显示「输入新指令可基于已有上下文续跑」）——RunFollowUp 本就支持 failed ref 续跑（v4.64 管道），此前入口不可发现；completed/running 不显示。i18n 三语 +1 键。
- **刀B diff 行内语法着色**（2c 遗留项，CodeMirror 依赖 3a 就位）：diffHighlight.ts——按路径选 Lezer parser（ts/js/jsx/tsx/py/json/css/html/md；模块级缓存），highlightLine 单行 token 切分输出 tok-* 类片段（600 字上限防退化，未知语言整行原样），配色 changesdiff-tok.css（语法语义色板 hex-exempt）；ChangesDiff 增可选 path prop，ChangesPanel/GitPanel 接线；配对行维持字符级高亮（语义优先），非配对行与 ctx 行走语法着色。
- 测试：diffHighlight +5（ts 关键字/数字片段、py 注释、未知语言原样、超长防退化、parser 缓存）+ SubagentThread +2（failed 提示条/completed 不显示）；vitest 231/1756、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（Git 面板 ts diff 12 个 tok 片段 DOM 断言；截图通道沿例故障如实记录）。

## v4.92.0 · Mermaid strict 安全线成文 + MemoMarkdown 消毒层（2026-09-05）
> 双源蒸馏规划阶段三 3b；**绑定面 578 零变更**（纯前端刀）。详见 releases/v4.92.0.md。
- **结论先行**：mermaid SVG **不追加**外层 DOMPurify 再消毒——mermaid v11 在 securityLevel:"strict"（本项目显式配置）下已内置 DOMPurify 消毒输出；实测（vite 页面配置矩阵 + ?mock=1 三轮走查）外层 pass 的 svg profile 必然剥离 foreignObject 内 html 标签（节点文字全失），ADD_TAGS 补救无效（svg 命名空间校验拒绝）。功能性破坏 > 边际防御收益，strict 上游为正解，决策过程成文于 lib/sanitize.ts 头注。
- **落地**：新增 sanitize.ts（DOMPurify）+ MemoMarkdown 流式尾部 renderPending 输出接 sanitizeHtml 消毒（各分支已逐段转义，此层兜底未来回归）；sanitize.test +3（文件 chip data-file-preview 保留/转义输出惰性断言/事件属性剥离兜底）；mermaid 走查样例入 mock README.md（flowchart 渲染、labels「输入/处理/输出」完整、无 script 注入）。
- 门禁：Go 全量 0 FAIL、tsc -b/eslint 0、vitest 229/1753、drift PASS（578）、?mock=1 走查通过（mermaid svg+labels+无脚本三断言；截图通道沿例故障如实记录）。

## v4.91.0 · CodeMirror 语法高亮编辑器（2026-09-05）
> 双源蒸馏规划阶段三 3a（拍板项——经用户「继续」指令按列序推进，采纳懒加载方案）；**绑定面 578 零变更**（纯前端刀）。详见 releases/v4.91.0.md。
- **依赖**：新增 codemirror@6 元包 + lang-markdown/javascript/python/json/css/html（MIT，均按需进懒加载 chunk；不引主题包——中性透明底 + defaultHighlightStyle，明暗主题均可读，无主题检测 seam）。
- **前端**：CodeEditor 组件（EditorState/lineNumbers/history/drawSelection + cmLanguageFor 按扩展名选语言：md/js·ts·jsx·tsx/py/json/css/html，未知扩展纯文本）；FilePreview 编辑态改挂懒加载编辑器——React.lazy chunk + Suspense（加载中回落 textarea）+ EditorBoundary（渲染错误回落 textarea），编辑能力永不丢失；value 由 CM 内部维护避免光标回跳，换文件经 key 重挂。onViewReady 回调供测试/高级用法。
- 测试：CodeEditor +3（jsdom 挂载与初值/onChange 经 dispatch 事务回传/换路径重建）+ cmLanguage +2（常见扩展映射/未知回退）；FilePreview 既有编辑流不破（23 用例同文件全绿）；vitest 228/1750、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（go.mod 编辑态 .cm-editor/.cm-content/.cm-gutters 三断言；截图通道沿例故障如实记录）。

## v4.90.0 · 终止级联：父任务中止连带终止派生后代（2026-09-05）
> 2026-09-05 调研回填中期候选（claude-code/cline「终止即级联」共识）；**绑定面 578 零变更**（Job.ParentID 为 Go 内部字段，FE 视图未动）。详见 releases/v4.90.0.md。
- **Go（jobs 包）**：`StartIn(caller, kind, label, run)`——从调用方 ctx 检出 job ID（复用既有 jobIDKey 注入）登记父子链（children 表 + Job.ParentID 字段）；`Kill(id)` 在原语义上追加 BFS 级联：全部存活后代连带取消（跨多层；已终态中间节点继续下钻），bash job 的 ctx watcher 随即强杀进程树；`Kill` 子任务不影响父任务（单向向下）；Close 全局取消原语义不变。
- 两个派生点改道 `StartIn`：后台 task 工具（agent/task.go）与后台 bash 工具（tool/builtin/bash.go）——嵌套派生自动挂父，主回合派生无父=原行为。会话 Close/回合取消传播路径不变。
- 测试：jobs +3（跨层级联+无关任务不受波及/杀子不动父/无父等价 Start，-count=2 绿）；vitest 227/1740、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）。**勘误**：v4.88/v4.89 发布说明的 vitest 计数（1743/1742）含测量口径误差，复跑稳定值 1740，本刀起以复跑稳定值为准。级联为引擎语义，mock UI 不可驱动，以 jobs 包单测为验收面（A1 先例）。

## v4.89.0 · 成本费率 hover：单价快照 · 三档明细 · 诚实降级（2026-09-05）
> 2026-09-05 调研回填中期候选（langfuse/ccusage 费率口径对齐）；**绑定面 578 零变更**（Timeline 增可选 JSON 字段）。详见 releases/v4.89.0.md。
- **Go**：fold 跟踪最近一次 usage 事件上报的非零单价（input/output/cacheHitPrice——本就是每 1M tokens 口径，费用公式 ÷1e6 证实），ContextTimeline 透出 `rate{inputPer1M,outputPer1M,cacheHitPer1M,currency}`；无定价上报时 rate=nil。
- **前端**：SummaryBar 成本单元格 hover 展示五居中明细——来源口径说明 + 未缓存输入/输出/缓存命中三档费率（¥X / 1M tok，USD 用 $）+ 累计费用；无费率诚实显示「供应商未上报费率，费用未估算」；costHoverTitle 纯函数（可测）；i18n 三语 +5 键。
- 测试：fold +1（费率透传/无定价 nil/-count=2 绿）+ cards costHoverTitle +2（CNY 明细/USD 符号/无费率降级）；vitest 227/1742、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（hover title 五行断言；截图通道沿例故障如实记录）。

## v4.88.0 · /context 居中弹层 + 常驻「剩余上下文%」徽标（2026-09-05）
> 双源蒸馏规划阶段二.5 2.5e（与 2026-09-05 调研回填「codex 式常驻上下文徽标」合并为一刀）；**绑定面 578 零变更**（纯前端刀）。详见 releases/v4.88.0.md。
- **ContextPill 常驻徽标**：Composer 上方右侧常驻「剩余 N%」胶囊（codex 式 context left 语义）——数据来自会话 store 的 ContextUsage（加载/回合末/usage 事件自动刷新），迷你进度条三档配色（≥90% err / ≥75% warning / 常规），title 带占用与总窗口；win≤0 不渲染。点击打开居中弹层。
- **ContextModal 居中弹层**：dsh「/context 弹层」同款语义——不离开对话查看当前上下文构成；内容复用 ContextView（与主区上下文 tab 同一组件，打开挂载拉最新快照、关闭即卸载 destroyOnHidden），centered 1080px。斜杠命令 /context 从「切主区上下文 tab」改道打开弹层；主区 tab 手动切换路径保留。Agent Network 会话跳转留后续（需 Go 侧按会话参数化）。
- 测试：ContextModal/Pill +5（百分比与 title/点击回调/警示色与 win=0 不渲染/弹层挂载与关闭回调/关闭卸载）；vitest 227/1743、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（徽标渲染、点击开弹层、/context 两段 Enter 全链路；截图通道沿例故障如实记录）。

## v4.87.0 · 统一 diff 渲染升级：改蓝配对 · 行内字符高亮 · 上下文折叠（2026-09-05）
> 双源蒸馏规划阶段二 2c；**绑定面 578 零变更**（纯前端刀）。详见 releases/v4.87.0.md。
- **diffRender.ts 新**（纯函数展示模型）：①改蓝配对——相邻删块+增块按行两两配对（min(删,增) 对），配对行蓝底替代红/绿并带 data-pair 标记，余量保持红/绿，交错输入先规范化为先删后增；②行内字符高亮——配对行做字符级 LCS（240 字上限防撑爆），变化片段独立着色、未变片段正常；③上下文折叠——连续 ctx 超过 keep×2（3×2）行时收起中段为「已折叠 N 行（点击展开）」占位，fold 项携带被收起行、展开后就地渲染。语法着色留阶段三 CodeMirror（边界成文）。
- **ChangesDiff 升级**：配对 → 折叠 → 渲染管线接入；三个数据源（变更面板 LCS、Git 面板 unified diff、后续归入的 docx/xlsx 对比）共用同一查看器。既有行为兼容：独立增删行红绿令牌不变、300 行截断与 content/none 降级语义不变。
- 测试：diffRender +8（配对/余量/纯增删/交错规范化/字符分段/超长退化/折叠阈值/透传）+ ChangesDiff +2（data-pair 蓝染替代红绿断言更新、折叠占位点击展开）；vitest 226/1741、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（折叠占位/展开后行渲染/配对标记三断言；截图通道沿例故障如实记录）。

## v4.86.0 · Git 面板最小集：status/diff/stage/commit/history（2026-09-05）
> 双源蒸馏规划阶段二 2b；决策门 D3 采纳推荐默认（单仓库、无 push/pull/fetch，v4.78 采纳 D1 推荐默认同款先例）；**绑定面 571→578**（+GaeaGitStatus/GaeaGitDiff/GaeaGitStage/GaeaGitUnstage/GaeaGitDiscard/GaeaGitCommit/GaeaGitLog，默认落 office 门面）。详见 releases/v4.86.0.md。
- **Go**（gaea_git.go 新）：执行 git CLI（exec 列表无 shell 注入面，仓库锚定 gaeaCwd）；status=porcelain v1（分支/ahead/behind/X|Y 两列展开为 staged/untracked/deleted/modified/renamed）；diff=unified 文本（--no-color，staged 走 --cached）；commit 只提交暂存区（不代 add），空说明拒绝，返回短 hash；discard=checkout --（破坏性，前端两击确认）；log=NUL 分隔 pretty format；非仓库/git 缺失诚实错误。
- **前端**：workspaceTabs 追加 git 一级 Tab（v4.53 合并后首个新增）；GitPanel——三分组（已暂存/未暂存/未跟踪）+ 状态字母徽标 + 暂存/取消暂存/丢弃（两击 3s 确认态）+ 行内 diff（buildGitDiff 解析 unified diff 复用 ChangesDiff 红绿渲染，未跟踪文件诚实提示无 diff 语义）+ 提交区（暂存计数/说明必填/按钮解禁逻辑）+ 提交历史折叠懒加载；非仓库空态；GaeaGit* 调用可选守卫（无后端环境不抛）。绑定三件套同步（bindingNames/spaceBindings 全 work + 数量锁 270→277/bridge）。
- 测试：Go +3（真实临时仓库走 status→stage→diff→unstage→commit→log→discard 全链路、非仓库错误、-count 含复跑）；planDiff +2（unified diff 解析/空降级）；GitPanel +6（三组渲染/非仓库态/暂存 diff/提交门禁/两击确认/历史懒加载）；spaceBindings 数量锁更新。vitest 225/1723、Go 全量 0 FAIL、tsc -b/eslint 0、drift PASS（578）、?mock=1 走查通过（DOM 断言四项；截图通道沿例故障如实记录）。

## v4.85.0 · 本轮文件三态折叠：写入/编辑/读取独立成层 + 类型筛选（2026-09-05）
> 双源蒸馏规划阶段二 2a（对标源 better-sidebar v0.18「统一文件变动」）；**绑定面 571 零变更**（纯前端刀）。详见 releases/v4.85.0.md。
- **三态独立折叠层**：ChangesPanel 由单一写类列表重构为 写入（write_file/move_file）/ 编辑（edit_file/edit_lines/multi_edit/notebook_edit/delete_*）/ 读取（read_file/grep/vision/format_convert，对齐后端 fileActionByTool read 白名单）三层——同一文件跨层独立出现（读后又被写=两层各一条，独立语义）；读取层轻量行（无 diff，点击直接开预览）默认收起降噪；写/编辑层保留行级 diff 展开+证据链回滚，工具集参数化（buildChangeCalls/buildSessionChanges 传集合，默认值不变零破坏）。
- **类型筛选 chips**：全部/文档/表格/图片/代码/其他按扩展名分桶（categoryOf），横贯三层过滤，chip 计数=并集文件数；过滤后全空显示诚实提示。
- items 为权威源；prop changes 在 items 为空时回退为「变更」合并层（历史恢复场景兼容）。diff 展开与回滚链路逐行保真搬运未改语义。
- 测试：changes +3（reads 聚合/分桶/工具集划分）+ 面板 +3（三态层头与默认折叠/读取层轻量行回调/分类过滤与空提示）+ 空状态用例补 store 清空；vitest 224/1720、Go 全量 0 FAIL、tsc/tsc -b/eslint 0、drift PASS（571）、?mock=1 走查通过（DOM 断言：三层计数/读取展开/分类过滤/空提示；截图通道沿例故障如实记录）。

## v4.84.0 · HTML 沙箱预览 · 外链协议分流（2026-09-05）
> 双源蒸馏规划阶段一 1c（阶段一 1a/1c 收口，1b 已勘误销账）；**绑定面 571 零变更**。详见 releases/v4.84.0.md。
- **HTML 沙箱预览**：GaeaPreview 增 `.html/.htm` kind=html（原文截断读，此前落 textExts 当纯文本）；前端 SandboxedHtml 组件——独立 iframe `sandbox="allow-scripts"`（刻意无 allow-same-origin=不透明源，无法触宿主 DOM/存储）+ Chromium csp 属性 `default-src 'none'`（禁一切网络外链，只放内联样式/脚本与 data:/blob: 图）双保险，顶条如实标注沙箱语义；FilePreview/FilePreviewModal 双消费点接线；i18n 三语 +1 键（非响应式 t()，沿 DocxPreview 先例防未包 Provider 挂载点抛错）。
- **外链协议分流**：browserPolicy 增 `classifyExternalLink` 纯函数——http/https 放行系统浏览器（loopback 拒：渲染文档不得探测本机服务）、mailto/tel 交系统处理器、javascript:/data:/file:/相对路径等一律 blocked；Markdown 渲染链接与价格源两处点击点接线（此前对任意 href 直接 openExternal）。
- 测试：Go +1（html kind）；browserPolicy +5、Markdown 分流 +2（loopback 拦截/https 放行）、FilePreview 沙箱 +2（sandbox/csp/srcdoc 断言+截断提示）；vitest 224/1712、Go 全量 0 FAIL、tsc/tsc -b/eslint 0、drift PASS（571）、?mock=1 走查通过（DOM 断言：iframe sandbox/csp/srcdoc+标注条；截图通道沿例故障如实记录）。

## v4.83.0 · 工具结果图片缩略卡：官方 patch 口径 token 估算（2026-09-05）
> 双源蒸馏规划阶段二.5 2.5b 后半（前半=v4.80 深读面板）；**绑定面 571 零变更**（NodeDetail 增可选 JSON 字段）。详见 releases/v4.83.0.md。
- **Go**：imgtoken.go 按官方 patch 口径估算（28×28px=1 token，⌈w/28⌉×⌈h/28⌉，先档位缩放再封顶：标准档 1568px/1568 tok、高分辨率档 2576px/4784 tok；官方例 1000×1000→1296 有测试锚定）；imgrefs.go 从参数 JSON/自由文本确定性提取图片引用（去重保序上限 4，非法 JSON 诚实不猜）；detail 层透出 imageRefs，App 层 resolveNodeImages 解析绝对路径+仅解码头部取尺寸+估 token（缺失/不可解码诚实降级）；fold 修 stats.Images 恒 0 死字段（按引用出现次数计数）。
- **前端**：详情面板图片缩略卡——缩略图（AttachmentDataURL 懒加载、点击开预览）+「原始尺寸→标准档缩放尺寸」+「≈N tok · 标准档」成对显示（悬停给完整口径与高分辨率档值）；缺失灰态/尺寸未知诚实标注；i18n 三语 +6 键；dev mock 补 vision 节点缩略卡样例。
- 测试：imgtoken/detail/fold +6（-count=2 绿）+ App resolveNodeImages 真实 PNG 编解码 +1 + 组件 +2；门禁：Go 全量 0 FAIL、tsc/tsc -b/eslint 0、vitest 224/1700、drift PASS（571）、?mock=1 走查通过（DOM 断言；截图通道沿 v4.82 故障如实记录）。

## v4.82.0 · 上下文趋势跳转浏览器：brief 行锚点 · 偏好持久化 · 设置中心默认（2026-09-05）
> 双源蒸馏规划阶段二.5 2.5d；**绑定面 571 零变更**（锚点搭 GaeaContextView 既有返回结构）。详见 releases/v4.82.0.md。
- **Go**：fold 跟踪 brief 文本来源事件 seq（user=消息节点；工具交换结果到达时锚到结果节点、未到退化 assistant 消息节点），RequestRecord 透出 briefUserSeq/briefRespSeq；header/usage 关闭/turn_done 估算关闭三处同拍刷新。
- **前端**：趋势卡步骤详情「输入/回复」brief 行带锚点时整行可点 → 上下文浏览器对应组自动展开（含分页全量）+ 滚动进视图 + 高亮 3s；锚点无节点诚实不跳、旧数据无锚点保持纯文本；趋势粒度/模式、浏览器分类内排序、文件活动排序初值改读 localStorage 偏好（gaea.context.prefs）变更写回；设置中心办公分组新增「上下文页偏好」卡（选项文案复用 contextview.* 既有键）；i18n 三语 +11 键。
- 测试：fold +1（-count=2 绿）+ contextPrefs 5 用例 + 组件 +5；门禁：Go 全量 0 FAIL、tsc/tsc -b/eslint 0、vitest 224/1698、drift PASS（571）、?mock=1 走查通过（DOM 断言：跳转展开/高亮/aria-expanded、四项偏好落盘+刷新回读、设置卡渲染；IAB 截图管道本会话故障，实拍缺席如实记录）。

## v4.81.0 · 文件活动行级增量：±徽标 · 操作日志展开 · 详情跳转（2026-09-05）
> 双源蒸馏规划阶段二.5 2.5c；**绑定面 571 零变更**（详情跳转复用 v4.80 GaeaContextNodeDetail）。详见 releases/v4.81.0.md。
- **Go**：FileActivity 增 Added/Removed/Hits；fileDeltaFromArgs 从写类工具参数确定性提取 ±行（write_file/edit_file/multi_edit/edit_lines 四形状，取不到诚实留零）；recordFile 增 dispatchID 回填 grep 命中数（LRU 重建时映射作废）。
- **前端**：文件行 ±徽标（聚合合计）；「N 次操作」展开逐次操作行；操作行「完整调用」懒加载跳转（NodeDetailPanel 自 v4.80 抽出共享 + useNodeDetails hook）；i18n 三语 +2 键。**勘误**：规划 1b「右键菜单/@悬浮引用」经 grep 核实早已落地（v4.25-4.31），过期条目订正、1b 销账。
- 测试：fold +1（-count=2）+ 组件 +1；门禁：Go 全量除 1 个既有 TempDir flaky（复验绿）、tsc/eslint 0、vitest 223/1688、drift PASS（571）、?mock=1 走查通过（截图存 artifacts）。

## v4.80.0 · 上下文工具结果深读：完整调用懒加载 · 来源 chip · 分类内排序（2026-09-04）
> 双源蒸馏规划阶段二.5 2.5b 前半；**绑定面 570→571**（+GaeaContextNodeDetail）。详见 releases/v4.80.0.md。
- **Go**：SurfaceNode 增 Tool/Err（fold 零成本透出）；detail.go `NodeDetailFor` 纯函数读端——tool_result 全文+同 id dispatch 参数回读（live/legacy 双 kind）、user/assistant 全文、1MB 截断标 Clamped（UTF-8 安全）；App.GaeaContextNodeDetail 懒加载绑定。
- **前端**：浏览器 tool 行来源 chip + ✗error 语义点；「完整调用」面板（OK/error 徽标、行数、截断双来源提示、参数行、26rem 有界滚动、原文/渲染切换默认原文）；分类内 时间序/大小序 chips；i18n 三语 +10 键。
- 测试：detail.go +3（-count=2）+ 组件 +2；门禁：Go 全量 0 FAIL、tsc/eslint 0、vitest 223/1687、drift PASS（571）、?mock=1 走查通过（截图存 artifacts）。图片缩略卡留 2.5b 后半。

## v4.79.0 · 上下文对比上一步：逐类 signed delta · 跨压缩近似标注（2026-09-04）
> 双源蒸馏规划阶段二.5 2.5a（收 v4.68 起三版连记 P1 欠账）；**绑定面 570 零变更**（delta 搭 GaeaContextView 既有返回结构）。详见 releases/v4.79.0.md。
- **Go fold**：request_header 组装时聚合活节点 surface 快照（与 Category 同拍自洽；system/tools 取最新 header 整体估算，其余逐条聚合）→ `RequestDelta{Items,Tokens,ByCat,Approx,First}`：ByCat 按降幅稳定排序、跨压缩 Approx=true、首请求 First=true（基线=空）。
- **前端**：趋势卡请求详情新增「较上一步」delta 条（合计 signed tokens + 逐类徽标 `+N项·±Nk` + `≈ 跨压缩，近似`）；i18n 三语 +5 键；dev mock 补 delta 样例。边界：历史步仅聚合级 delta，不做 dsh 式逐步节点回放（wire 载荷 + Go 权威折中，蓝图成文）。
- 测试：fold +3（-count=2 绿）+ 组件 +2；门禁：tsc -b/eslint 0、vitest 223/1685、drift PASS（570）、?mock=1 实机走查通过（截图存 artifacts）。

## v4.78.0 · 任务强制终止收口：进程树击杀 · 两击确认（2026-09-04）
> 双源蒸馏规划（docs/gaea-dsh-better-sidebar-long-term-distill-plan-2026.md）
> 阶段一 1a 刀；决策门 D1 采纳推荐默认（只做任务输出+强杀，交互终端不进队列）。
> **绑定面 569→570**（+GaeaTaskKill）。详见 releases/v4.78.0.md。
- **Go 进程级强杀**：`tasks.Progress.OnForceKill(fn)` 进程类任务登记强杀钩子（典型=闭包 proc.KillTree/KillTracked），尝试结束三处同拍统一清理；`tasks.Manager.Kill(id)`：queued 原子取消（message=已强制终止）／running 锁内快照 cancel+钩子并记 cancelReq（与 Cancel 同纪律），锁外先杀进程树再传播 ctx 取消，幂等，无钩子诚实降级等价 Cancel。
- **前端两击确认**（对齐源任务设计文档防误杀）：TaskCenter running「强制终止」首击进入 3s 确认态（红描边+「再击确认终止」），再击才调 GaeaTaskKill；queued/stopping 保持单击；bridge/spaceBindings/mock/i18n 三语 +3 键。
- **窄屏自动激活策略审计**（1a 第三项）：CSS 1239px 隐藏断点与 openTasksAuto <1240 判定一致，零改码审计通过。
- 测试：tasks_kill_test 4 用例（真实子进程强杀 -count=2 绿）+ app 透传 +1 + TaskCenter 两击确认 2 用例；门禁：go vet 0、go test 全量除 1 个既有 TempDir 清理 flaky（单跑复验绿）、tsc -b/eslint 0、vitest 223/1683、drift PASS（570）。**绑定面计数勘误**：v4.67~v4.77 所记 561 为过期口径（小说并行会话未提交绑定所致），本刀起以 drift 实际输出为准。

## v4.77.0 · 小说板块革命：场景制生成 + 确定性去 AI 味闭环（2026）
> 用户点名「彻底、革命性、解决长文写作和 AI 味」。先做竞品调研（商业 Sudowrite/NovelCrafter/
> NovelAI + 中文 马良写作 硬数据「20万字后设定矛盾率 83%→23%」+ 社区 anti-ai-checklist/anti-ai-polish），
> 再并行子代理建三大新包（novelstyle / novelcontext / narrative），接入生成管线。详见
> `docs/gaea-novel-revolution-2026.md`（逐文件审计 + 可动刀架构 + 落地状态）。**绑定面 561→566**。
- **三大新后端包**（全离线可测）：
  - `internal/novelstyle`：确定性文风指纹（函数词 z 向量 + Burrows Delta + 句/段长 + 1000字TTR+n-gram）+ `ScoreText`(0-100 + rune span 定位, 9 条规则) + `DeSlopRewrite`(AI 黑名单词→平实替换 + 标点归一, 只改命中词、不碰剧情)；11 测试全绿。
  - `internal/novelcontext`：场景圣经编译器 `CompileSceneBible/BuildSceneBibleFromChapter/Render`，按 `SceneMeta.POVCharID` 做视角掩码（公开可见/秘密进 HiddenFacts）+ 子图检索 + 未回收伏笔 + 时间锚点 + 文风；4 测试全绿。
  - `internal/narrative`：确定性叙事状态机 + append-only 状态补丁账本；`ApplyPatch`(纯函数) + `ValidateStatePatch` + `AuthorizeAndSettle`（`approved=false` 不入账本）+ 可回放；5 测试全绿。
- **接入生成管线**（`internal/app/create_chapter_handler.go`）：`CreateChapter` 注入 POV 场景圣经；`done` 事件携带 `novelstyle` AI 味分；`story-deslop` 启用时生成即确定性去味（分数+改动入 `deSlop`）。
- **前端反馈**（`CreatePage.tsx`/`chapterStreamTypes.ts`）：生成完成展示 AI 味分 + 命中问题；vitest 14/14 + `tsc -b` 0。
- **场景级生成 + 叙事状态结算绑定**（绑定面 566→**568**）：新增 `CreateScene`/`GenerateScene`（POV 感知逐场景生成落 v4 scenes/）+ `GetNovelState`/`BuildNovelStatePatch`/`SettleNovelState`（narrative 审批制结算）+ `DeSlopChapterAiTaste`（手动一键去味）+ `RewriteChapterAiTaste`（**LLM 受限重写**：只改打分命中句，复测分数下降才落盘，安全闸）；`bindingNames.ts` 重生成 + bridge `LegacySurfaceNames` 同步；drift PASS（568）。
- **前端**：AI 味分+去味结果展示、叙事状态账本+审批 Modal、「一键去味」「**高级去味**」按钮；vitest 14/14 + `tsc -b` 0 + eslint 0。
- 验证：`go build ./...` 0、`go test`（app + 3 新包）绿、`go vet`/`gofmt` 干净、前端 `vitest`/`tsc -b` 0、绑定 drift PASS。

## v4.77.0 · 任务页自动激活补全：后台任务也自动开 · 宽窄屏差异化 · 强制终止（2026-09-04）
> 用户验收规格：实时拓扑/批量实时预览（已有）、后台任务退出码/实时输出/强制
> 终止、新子代理/新任务自动激活任务页（宽屏展开、窄屏不强制、可关）。
> **绑定面 561 零变更**（纯前端）。详见 releases/v4.77.0.md。
- 新增后台任务事件监听：新 queued/running/stopping 任务自动激活「任务」视图；
- 自动激活统一收敛到 openTasksAuto：宽屏（≥1240px）才展开右栏，窄屏不强制，
  沿用设置中心 gaea.tasks.autoOpenSubagent 开关（可关，默认开）；
- TaskCenter 运行中任务按钮明确为「强制终止」（queued 仍为取消、stopping 停止中）；
- 子代理树实时拓扑 / 批量实时预览 / 退出码 / 实时输出 dock 审计确认已在；
- vitest 223/1681（净增 1：强制终止按钮用例）、tsc/eslint 0、冒烟 200。

## v4.76.0 · 任务面板修正：整棵子代理可折叠 · 圆角收小 · 树形线（2026-09-04）
> 用户三点反馈：圆角太大要改小、增加树形线、点击要能整体折叠「子代理」区块
> （不是单卡）。**绑定面 561 零变更**（纯前端）。详见 releases/v4.76.0.md。
- 右侧「子代理」分组头改为可点击折叠开关（整棵 AgentTree 显隐，默认展开）；
- AgentTree 卡圆角 rounded-lg(16px 主题解析) → 显式 rounded-[8px]；
- 子树前加左侧树形线（text-secondary 45%），层级一眼可读；
- 单卡仍可折叠（有子树时点卡片标题 = 展开/收起），打开子代理对话收敛为
  卡片上的独立按钮；新增 i18n subagent.openThread 三语；
- vitest 223/1680（净增 2：区块折叠 + 单卡标题折叠用例）、tsc/eslint 0、冒烟 200。

## v4.75.0 · 右侧任务面板卡片化：子代理独立卡 · 运行状态胶囊（2026-09-04）
> 用户点名「右侧任务面板现在像文件树，参考图里每个子代理有标题/卡片/运行
> 状态」。ds-vision 读参考图 + redesign 审计后重构 AgentTree。
> **绑定面 561 零变更**（纯前端）。详见 releases/v4.75.0.md。
- 子代理节点从单行树行升级为独立圆角卡片（surface-container-high + 语义描边）：
  标题 + 状态胶囊（进行中/已完成/失败）+ 模型/工具数/耗时/token 指标行；
- 运行中卡片：glow 描边 + 实时动作预览（正在…/⚙ lastTool）；已完成卡片：
  展示分工回答摘要；主 agent 根卡主色底；卡片间距 1.5→2（gap-2）；
- AgentTree 既有交互不变（展开/收起、点卡打开子代理对话）；测试等价改写；
- 视觉模型验收 92/100（卡片独立成型）；vitest 223/1678、tsc/eslint 0、冒烟 200。

## v4.74.0 · 记忆界面技能重设计：hero + 统计小卡 + 视图大卡（2026-09-04）
> 用户点名「使用技能重新设计记忆界面」。依 redesign-existing-projects 审计
> （多个同尺寸卡片堆叠缺层级）与 ui-ux-pro-max 检索（hero/统计/主内容层级）
> 重构。**绑定面 561 零变更**（纯前端）。详见 releases/v4.74.0.md。
- 记忆页收敛为 4 段递进结构：头部 hero 卡（标题+摘要+开关）→ 三枚统计小卡
  （事实/文档/建议）→ 快速添加卡 → 单张「视图大卡」（分段+搜索+筛选+内容）；
- TabButton 改紧凑胶囊（去掉 flex-1 拉伸），与搜索/内容同卡承载；
- 事实搜索框改内嵌卡片底；fact-card 补卡片底色；视觉模型验收 9/10；
- vitest 223/1678、tsc/eslint 0、冒烟 200。

## v4.73.0 · 记忆迁入主区标签页：左侧入口移除 · 记忆界面卡片化（2026-09-04）
> 用户点名：删左侧面板「记忆」入口，在「上下文」旁新增「记忆」tab，并整体
> 重设计记忆界面。**绑定面 561 零变更**（纯前端，Go 零改动）。
> 详见 releases/v4.73.0.md。
- 记忆由抽屉迁入主区 ChatTabs（对话/轨迹/上下文/**记忆**）；左侧记忆按钮
  （展开态 + 折叠态）删除，Ctrl+K「记忆面板」与 `/memory` 命令改为切主区 tab；
- MemoryPanel 改主区卡片墙：头部卡（标题+摘要+记忆/晨报开关）、快速添加卡、
  事实/文档/建议分段控件（TabButton 改胶囊分段）、事实条目独立小卡；
- 抽屉路由收敛（useDrawers 不再管理记忆）；历史 localStorage 记忆视图记忆
  行为不依赖抽屉；vitest 223/1678、tsc/eslint 0、冒烟 200。

## v4.72.0 · 删除主区「概览」标签页（2026-09-04）
> 用户点名删除；概览承载的 Token/费用/命中率统计已由上下文页的 Token 统计 /
> 预估费用 / 会话信息卡承载，入口移除不产生信息缺口。**绑定面 561 零变更**
> （纯前端，Go 零改动）。详见 releases/v4.72.0.md。
- ChatTabs 由 4 tab 收敛为 3 tab（对话/轨迹/上下文）；删除 OverviewPanel
  容器组件与命令面板「概览面板」条目；历史 localStorage 选中 overview 自动
  回落「对话」；底层 StatsPanel 统计模块暂留未接线（彻底清理另行处理）；
- tsc -b/eslint 0、vitest 223/1677、冒烟 200。

## v4.71.0 · 上下文页卡片化：8 统计小卡 · 行卡化（2026-09-04）
> 用户点名「上下文统计是 8 个小卡片，不是 1 张大卡」；对照参考图后不逐像素
> 复刻外部插件，按自研规划落地（redesign-existing-projects + ui-ux-pro-max +
> ds-vision 读图）。**绑定面 561 零变更**（纯前端，Go 零改动）。
> 详见 releases/v4.71.0.md。
- 8 个统计项各自成独立小卡（`.ctx-tile`，2→4→8 列响应式），删除「1 张大卡 8 格」；
- Token/耗时/会话信息三大仪表卡带图标章卡头；事件/文件活动/浏览器节点行
  全部改独立圆角卡行（`.ctx-row`），细分隔线移除；
- 卡片表面抬升 surface-container + 顶部内高光，零硬编码色值；三语死键
  contextview.statsTitle/statsHint 删除；
- tsc -b/eslint 0、vitest 224/1683、冒烟 200。

## v4.70.0 · 上下文页视觉重设计：数字层级 · 构成强调 · 仪表化（2026-09-04）
> v4.69 交互精修静态不可见（承认），本轮改为肉眼可见的视觉重设计，逐项对齐
> dsh-context styles.css 实测数值。**绑定面 561 零变更**（纯前端，Go 零改动）。
> 详见 releases/v4.70.0.md。
- 数字层级放大：统计格 value 15/17px、当前上下文大数字与百分比 26px bold、
  环心 16px、各行 11.5px——关键数据一眼可见；
- 当前上下文宽卡强调：分段条 h-3→h-4 rounded-md、卡头 12.5px semibold、
  图例 11.5px；卡头「标题左+副注右」统一（dsh lc-card-sub 同构）；
- 环形图 Donut 84→92px、stroke 10px；事件多选 chips、趋势卡内联详情
  （v4.69 交互）保留；
- tsc -b/eslint 0、vitest 224/1683、drift PASS（561）、冒烟 200。

## v4.69.0 · 上下文页第三轮精修：图例联动 · 趋势卡内联详情 · 事件多选（2026-09-04）
> 继续对齐 dsh-context v0.41.3（用户实拍两图 + 源码复核）。**绑定面 561
> 零变更**（纯前端，Go 零改动）。详见 releases/v4.69.0.md。
- 当前上下文宽卡：图例 chip hover 联动——悬停某分类该分段保持、其余段
  150ms 淡出（dsh hover-key 同款），chip 悬停加微高亮；
- 趋势卡 master-detail 单元：请求详情内联进趋势卡（删独立 StepDetail 行），
  并默认选中最新请求——打开页面即有「第 N 轮·第 M 步」内容，点柱钉住旧
  选择、请求消失自动回退最新；
- 事件筛选对齐 dsh 多选 chips：全亮点一类=单选、多选点掉=排除、单选再点=
  恢复全选（任一时刻 ≥1 选中，去「全部」单选项）；
- 四仪表卡头行统一「标题左 + 副注右」单行排版；Token 环心命中率两位小数；
  死键 contextview.pickHint 三语删除（点击引导已由趋势图例行承载）；
- tsc -b/eslint 0、vitest 224/1683（+2 用例）、drift PASS（561）、冒烟 200。

## v4.68.0 · 上下文页对齐 dsh-context：网格仪表 + 耗时折叠 + 径向 Agent 图（2026-09-04）
> 用户展示 dsh-context v0.41.3 并点名对齐（先读其源码）。**绑定面 561 零
> 变更**（struct 字段级）。详见 releases/v4.68.0.md。
- Go：FoldTimeline 单趟折叠 ContextTiming（wall/ttft/gen/tools 时长+每工具
  排行，诚实近似+全零省略）；
- UI：v4.67 双栏 tab → dsh 全宽网格仪表（四仪表卡/当前上下文宽卡含空闲段/
  浏览器分类折叠组/文件活动聚合树/Agent 径向树/底部汇总条+口径页脚），
  i18n 三语 +47 键，功能零删减；
- judge 三图 pass 零 must_fix；Go 全量（定向包）、tsc -b/eslint 0、
  vitest 224/1681、冒烟 200。

## v4.67.0 · 上下文标签页重设计：驾驶舱三区布局（2026-09-04）
> 用户点名「使用技能重新设计上下文标签页」（ui-ux-pro-max）。**绑定面 561
> 零变更**。详见 releases/v4.67.0.md 与 design-system/gaea/pages/context.md。
- 单列 8 卡堆叠 → 驾驶舱三区：顶部总览条（统计 chips+六分类分段条+图例+
  水位，融合原三卡）/ 左过程轴（趋势→详情就地联动→事件流）/ 右 inspector
  三 tab（浏览器/文件活动/Agent，<1100px 回落单列）；功能零删减；
- judge 视觉验收两图 pass 无 must_fix；tsc -b/eslint 0、vitest 221/1647、
  冒烟 200。

## v4.66.0 · 子代理会话提升：保存为新会话 · TasksWorkbench 轮询收敛（2026-09-04）
> dsh Side Chat promote 语义补完。**绑定面 560→561（+GaeaPromoteSubagent）**。
> 详见 releases/v4.66.0.md。
- 后端忠实投影子代理 transcript 为独立新顶层会话（写前写后投影往返双
  校验，不等价不落盘；原运行逐字节不动；每次提升新副本）；诚实降级
  （system 提示不随迁/孤立工具记录丢弃）；ref 仅 sa_、running 拒绝；
- SubagentThread 头部「保存为新会话」按钮（busy/running 守卫 + 三语 +3 键）；
- TasksWorkbench net+runs 双源轮询迁两 store（四消费点全部共享单轮询），
  失败态不再静默吞错；+11 FE 用例。
- Go 全量 exit 0、tsc -b/eslint 0、vitest 221/1647、drift PASS（561）、
  冒烟 200。

## v4.65.1 · 三线并行：追问失败感知 · 任务退出码 · AgentNetwork 轮询收敛（2026-09-04）
> 三并行子代理分线 + 主代理收口。**绑定面 560 零变更**（struct 字段级）。
> 详见 releases/v4.65.1.md。
- 线A 任务退出码：Task 加 ExitCode（指针区分未上报/0），TaskCenter 失败行
  常显 `· exit N`；**「强杀」欠账过期销账**（3 类任务均为纯函数 handler，
  协作取消已足够）；取消竞态用例 ×3 复跑无回归；
- 线B 追问后台失败前端感知：meta 加 FollowUpError 经轮询带出，失败气泡
  显示真实原因（失败判定先于快照增长）；**顺手修 v4.64.0 真回归**——
  RunFollowUp 的 defer stop() 晚于终态写致 meta 永久卡 running；
- 线C 新 lib/agentNetworkStore.ts 收敛 SubagentsPanel 的 net 轮询（模式
  对齐 runs store）；失败文案统一 subagent.runsLoadFail 三语新键；
  TasksWorkbench 双源轮询为下一消费点（已收档）。
- Go 全量 exit 0；tsc -b/eslint 0；vitest 220/1636；drift PASS（560）；冒烟 200。

## v4.65.0 · 三线并行收欠账：追问失败重试 · 工作台偏好设置卡 · 子代理轮询收敛（2026-09-04）
> 三并行子代理分线 + 主代理收口。**绑定面 560 零变更**。详见
> releases/v4.65.0.md。
- 线A 追问失败诚实化：tab 内联错误条 + 失败气泡保留原文 + 一键重试/撤销
  （快照接管守卫修正：失败态不误清、不空转轮询）；SubagentThread 12/12；
- 线B 「办公工作台偏好」设置卡（设置中心 → 办公置顶）：四个自动展开类
  偏好获得开关+说明+即时生效标注；新 lib/tasksPrefs.ts；澄清 autoOpenJobs
  键不存在的过期欠账；默认值零改动；
- 线C SubagentsPanel/Sidebar 迁共享 subagentRunsStore：同屏 3 路重复轮询
  收敛为 1 路共享；store 扩展 loading/ready/error 状态机 + reload + 失败
  保留旧快照自愈（向后兼容，App 零改动）；失败态不再静默白板；
- 线D releases/README.md 从主 README 错位副本重写为归档索引（历史文件
  零删除）。
- tsc -b/eslint 0（收口抓 1 处测试类型收窄）；vitest 219/1624（净增 19）；
  drift PASS（560）；冒烟 200；实机走查（设置新卡+任务面板）PASS。

## v4.64.3 · 任务管理树呼吸感优化（2026-09-04）
> 用户点名：任务管理树过于紧凑。**绑定面 560 零变更**（纯前端呈现）。
> 详见 releases/v4.64.3.md。
- AgentTree 节点行行高/内边距/间距放大（py-1.5、gap-1、缩进 depth×12）、
  标题 12px、统计 10px、状态点 8px；节点卡间距 gap-1.5；运行活动预览行
  与「本地模型工具」区块同步放宽；信息零删减。tsc -b/eslint 0、
  vitest 217/1605、冒烟通过、实机走查 PASS。

## v4.64.2 · 修复：任务管理树丢失零工具子代理（2026-09-04）
> 用户实机报告：任务管理面板不出现新派发的子代理。**绑定面 560 零变更**。
> 详见 releases/v4.64.2.md。
- 根因：FoldAgentNetwork 以「拥有子工具记录」定义子代理节点——纯调研型
  （禁用工具、零工具调用）在事件日志无子记录、永远成不了节点，enrich 只
  富化不补挂，零工具子代理整批从树消失（running 计数正常所以仅列表缺行）。
- 修复：enrichAgentNetwork 两段式——富化同时记录 run 承载情况，未被承载且
  非 model_tool 的 run 补挂合成节点（id=sa_ ref，前端可开转录）。
- 同日完成 v4.62.2～v4.64.1 六版实机验收（六项全 PASS，证据见发布说明）。
  新增回归 3 例；Go 全量 0 FAIL、vitest 217/1605、drift PASS（560）、冒烟通过。

## v4.64.1 · 修复：mt_ 信封双层嵌套转义墙（2026-09-04）
> 热修复：mt_ 输出信封双层嵌套，v4.62.2 只拆一层仍留转义墙；历史转录已
> 落盘无法自愈。**绑定面 560 零变更**。详见 releases/v4.64.1.md。
- 写端 unwrapModelToolOutput 改递归拆包（4 层上限）；读端
  unwrapEnvelopeText 显示侧同语义拆包救历史数据；双侧回归测试。
  vitest 217/1605、drift PASS（560）、冒烟通过。

## v4.64.0 · Side Chat 式追问：子代理会话 tab 内可持续提问（2026-09-04）
> 用户点名补上 dsh 的 Side Chat 式追问。**绑定面 559→560**
> （+GaeaSubagentFollowUp）。详见 releases/v4.64.0.md。
- 子代理 tab 底部新增追问输入框：复用 continue_from 管道带着完整工作记忆
  继续运行；乐观用户气泡 + 专用通道流式打字 + ~1s 快照轮询；守卫（running/
  mt_/主回合运行中拒绝）双侧对齐。vitest 217/1603、冒烟通过。

## v4.63.4 · mt_/长文本输出 Codex 式有界渲染（2026-09-04）
> 用户点名。**绑定面 559 零变更**。详见 releases/v4.63.4.md。
- mt_ 标签页/超 4000 字输出默认限高滚动 +「展开全部（N 字）/收起」+ 字数
  标注；流式实时行保持跟随。i18n 三语 +3 键。vitest 217/1602、drift PASS
  （559）、冒烟通过。

## v4.63.3 · 对标 dsh：GaeaSubagentRuns 单轮询聚合 + 新子代理自动切任务视图（2026-09-04）
> 用户点名借鉴 dsh-better-sidebar。**绑定面 559 零变更**。详见
> releases/v4.63.3.md。
- 共享单轮询 store（每会话单定时器/单在途/不可见门控，App 两处轮询并入）；
- 新子代理 0→N 自动切右栏任务视图（500ms 去抖重臂 + 偏好开关默认开）。
- vitest 217/1601、drift PASS（559）、冒烟通过；Side Chat 式追问列为后续刀。

## v4.63.2 · 并行子代理：批量派发不再串行排队（2026-09-04）
> 用户实测发现三路子代理串行执行。**绑定面 559 零变更**。详见
> releases/v4.63.2.md。
- 根因：getConflictKey 的全局冲突键 "!spawn" 把同回合 N 路派发拆成 N 个
  串行批。修复=task/run_skill 改每调用唯一键（同批并行，runParallel ≤8），
  install_skill 保持串行；TaskTool 用量改 usageMu 互斥合并（并行安全）。
- 如实说明：本地模型推理在服务端仍可能排队，工具段真实重叠。新增分区
  回归测试 3 例。vitest 216/1597、drift PASS（559）、冒烟通过。

## v4.63.1 · 主对话子代理卡片整卡可点（2026-09-04）
> 用户点名：单击子代理卡片直接打开会话 tab。**绑定面 559 零变更**。详见
> releases/v4.63.1.md。
- task / run_skill 卡解析出 ref 整卡可点；空 ref 回退唯一 running 命中
  （宁缺勿错）；tab 预填+5s 轮询自校正；活动行同入口。i18n 三语 +1 键。
  vitest 216/1597、drift PASS（559）、冒烟通过。

## v4.63.0 · 子代理会话 tab 输出对齐主对话 Codex 式渲染（2026-09-04）
> 用户点名。**绑定面 559 零变更**。详见 releases/v4.63.0.md。
- assistant 正文/思考走主对话 AssistantMessage、工具走主对话 ToolCard
  （toolCallId 配对映射层 subagentRender.ts，孤儿降级、运行中标 running）；
  流式实时行同款。vitest 214/1594、drift PASS（559）、冒烟通过。

## v4.62.2 · 修复：对话标签页实时输出失聪（2026-09-04）
> 热修复 v4.61.0 引入的 EventsOff 连坐炸订 + 两个实机报告问题。**绑定面 559
> 零变更**。详见 releases/v4.62.2.md。
- **主因**：SubagentThread 卸载时 EventsOff("gaea-event") 注销该通道全部
  监听者，主对话订阅被连带炸掉→实时过程全灭（切过一次子代理标签页即触发）。
  修复=按监听者精确注销（wails EventsOn 返回值），前端禁用 EventsOff，
  3 个回归测试钉死。
- **附带**：mt_ transcript 落盘前拆 JSON 信封（消灭字面 
 转义墙）；
  GaeaTaskList 变参必填修正（任务中心恒空）。vitest 214/1590、drift PASS
  （559）、冒烟通过。

## v4.62.1 · 修复：子代理流式打断对话窗过程可见性（2026-09-04）
> 热修复 v4.62.0 线 A 回归。**绑定面 559 零变更**。详见 releases/v4.62.1.md。
- **根因**：SubagentText 挂 gaea-event 消费 seq 但 wire-only 不落账本，破坏
  v4.26「seq↔日志 1:1、丢件可 resync 补拉」前提——密集流丢一件即不可愈合
  缺口，前端反复整体重建对话视图，过程可见性被打断。
- **修复**：SubagentText 分道专用通道 gaea-subagent-text（无 seq）；死映射
  移除；forwarder 不变量成文；bridge 新增 onSubagentText；回归测试钉死
  seq 无断号。vitest 214/1587、drift PASS（559）、冒烟通过。

## v4.62.0 · 办公板块：子代理逐 token 流式 · 交付验收闭环 A2（2026-09-04）
> 欠账池三线（两刀快照）。**绑定面 559 零变更**。详见 releases/v4.62.0.md。
- **子代理逐 token 流式（P1 销账）**：新增 SubagentText 事件（wire-only，
  EventLogSink 免落盘）——持久化子代理（task + run_skill 派生）运行中的助手
  文本增量实时打到对应会话 tab，「正在打出的字」即时可见，不再等 ~1s 快照/3s
  轮询；快照追上后缓冲让位（reconcile），断流由既有补拉兜底。
- **Word 修改队列（Genspark Send N edits 式）**：框选「加入队列」攒批（同摘录
  同指令去重合并）→「执行全部」串行走既有 AI 生成→修订写回→自动接受通道；
  每条执行前对最新全文再定位，定位不到诚实跳过（绝不错位替换），单条失败继续，
  结束给成功/失败/跳过汇总，失败可单独重试；修订制兜底不变。
- **版本结构化对比（收 unsupported 欠账）**：.docx 段级红绿 diff（附段落序号
  列）；.xlsx sheet 对齐 + 单元格级差异表（公式串当文本如实比较，截断计数
  不失真）；pdf/图片维持「并排预览」降级。
- i18n 三语 +35 键（docxQueue.* / vcompare.*）。
- 验证：go vet/test 全量绿；tsc -b / eslint 0；vitest 214 文件/1586 用例；
  drift PASS（559）；构建产物冒烟通过。

## v4.61.0 · 子代理会话闭环 · Word 目录侧栏（2026-09-04）
> 四个未发版快照（eb84c82c/5c52a5b8/1f70e06d/aa57784c）合并发布，主线
> 「子代理从壳到芯对齐 Codex 口径」。**绑定面 559 零变更**。详见
> releases/v4.61.0.md。
- **A · Word 预览目录侧栏**：docxOutline 纯解析（outlineLvl/标题样式/
  basedOn 链，TOC 与页眉页脚排除）→ 工具栏「目录」侧栏 → 点击定位高亮 +
  铅笔按钮生成「修改《章节》」composer 模板。
- **B · 子代理 tab 对齐主代理**：SubagentThread 正文 MemoMarkdown 化；
  ChatTabs 会话 tab 状态点 + 完整 title（任务｜状态·模型），App 5s 轮询同步。
- **C · transcript 真机接线 + 本地模型工具同 UI**：惰性 SubagentStore 接线
  task/run_skill 全部真实落盘 + ~1s 快照实时化（顺带修复后台子代理从不落盘）；
  ModelBacked 标记（vision/summarize_file）→ 主执行器开 mt_ 变相子代理记录，
  同一左栏行/任务区块/会话 tab 展示；读端 kind/tool/title 扩展 + 事件驱动补拉。
- **D · 子代理入口收敛两级**：移除三办公顶层包装工具与 explore/research/
  review/security_review 分类残留；入口 = task + run_skill（技能/模板前缀
  与 per-skill 模型保留），系统提示同步收敛。
- 验证：Go 逐包全量绿（test-all.ps1）；vitest 211/1535；tsc -b/eslint 0；
  drift PASS（559）；构建产物冒烟通过。

## v4.60.0 · better-sidebar pane 化三刀 · 文件打开统一开 tab（2026-09-03）
> 三个未发版快照（d856353e/cf1bf35/4a0cae7a/6ca03e4d）合并发布 + 并发子代理
> 习惯固化。**绑定面 559 零变更**。详见 releases/v4.60.0.md。
- **右栏 pane 工作台（better-sidebar 同构）**：欢迎卡 → 视图 tab → 文件 tab
  单 tab 条、按会话持久化；产物/任务/浏览器补齐视图 tab（浏览器=地址栏+沙箱
  iframe）；任务页重做紧凑「子代理拓扑+后台任务」单页；点子代理 → 主区独立
  会话 tab；删除旧 WorkspaceTabs/WorkspacePanel/EditorTabs。
- **左栏子代理会话入口**：Sidebar 父会话行展开 → 子代理子行（状态点/任务
  摘要/状态·模型·时间，loading/空态/失败重试），点击复用独立子代理会话 tab；
  复用 GaeaSubagentRuns，当前会话运行中 5s 刷新。
- **文件打开统一开 pane 文件 tab**：产物/权威登记行、正文交付卡/行内附件/
  Markdown 文件引用、变更「打开文件预览」全部改为右栏文件 tab（可并存、
  去重激活；sidebarRegistry.openPaneFileTab + paneFileOpen 注入，未注册回落
  大预览）。
- **习惯固化**：并发子代理固化为默认执行纪律（AGENTS.md「执行纪律」）。
- 验证：go 全量测试绿、tsc -b 0、eslint 0/0、vitest 210/1525、drift PASS
  （559）、构建产物冒烟通过。

## v4.59.0 · 继续：i18n 二批 · 搜索落划线 · 自定义引擎用户价目（2026-09-03）
> 欠账池三线并行子代理（i18n/小说/模型中心域）+ 主代理收口。**绑定面 559
> 零变更**。详见 releases/v4.59.0.md。
- **A 线·i18n**：设置五面板+SettingsSection 入三语字典 +192 键/语言
  （682→874），设置中心九面板全量 i18n；zh 保真，0 重复键。
- **B 线·小说**：搜索命中「落为划线」一键永久标注（searchHitAnnotation 纯
  函数适配命中区间→摘录口径，保留原文大小写）；拆分第四批 ReadingPrefsPanel
  净减 47 行；+13 用例。
- **C 线·模型中心**：自定义引擎用户价目 v1——EngineConfig 加
  user_price_in/out 指针三态字段（nil=不动/正数=设置/<=0=清除），折算插
  最高优先层，零值语义与现状一致（回归锁）；绑定面 559 不变，wails models
  已重生成。
- **收口抓雷**：设置→聊天分组 mock 下整板块白屏——ChatPanel
  GetVoicePipelineConfig 直调 wailsjsCompat 在浏览器同步抛（`?.` 只防导出
  缺失不防执行），潜伏既有雷非本刀回归；try 兜底修复+全设置图审计。
- 验证：go 全绿、tsc/tsc -b 0、eslint 0/0、vitest 204/1457、drift PASS（559）、
  ?mock=1 九个设置分组零错误。

## v4.58.0 · 继续：三线并行收欠账 · 同章搜索重定位缺陷修复（2026-09-03）
> 欠账池三线并行子代理（novel/dev mock/i18n 域，所有权互斥）+ 主代理收口。
> **绑定面 559 零变更**。详见 releases/v4.58.0.md。
- **A 线·小说**：同章搜索重定位缺陷根因实锤（定位 effect 依赖 `[readMode,
  readNodeId]` 缺命中序号，同章命中三依赖全不变 effect 不重跑）——最小修复
  `searchLocateSeq` 入依赖，回归测试反向验证；拆分第三批 1352→1285 净减
  67 行（readingAnnotation/readingBookmark/readingScrollMemory/chapterTabData
  四新文件+两扩充），+41 用例。
- **B 线·dev mock**：补 GaeaBenchmark 五方法（查询类中性空态/动作类诚实
  失败）+ GetModelMonitor，契约 7 用例。
- **C 线·i18n**：设置三面板（绘梦/小说/关于）文案入三语字典 34 键/语言，
  zh 逐字保真（SettingsPage.test 不改全绿即验证），键总量 648→682。
- **收口**：GetModelMonitor 三消费点（ResourceMonitor/MainLayout/Module-
  Launcher）从 wailsjsCompat 直读迁 `getModelMonitor()` 三态回退（直读绕过
  bridge mock——欠账真身）；mock 补 GetEngines 空态（走查新抓缺口）。
  ?mock=1 走查零横幅：资源块 0% 空态/引擎管理空表/「暂无测评记录」。
- 验证：go 全绿、tsc/tsc -b 0、eslint 0/0、vitest 202/1442（+5 文件 +49
  用例）、drift PASS（559）。

## v4.57.0 · 设置中心化繁为简：删四补一 · 界面语言入口（2026-09-03）
> 用户点名刀：删除不要的（全部 grep 核实零交互/重复/零消费，零功能损失），
> 增加需要的。**绑定面 559 零变更**。详见 releases/v4.57.0.md。
- **删四**：绘梦「当前绘梦后端」纯展示卡（与下拉信息完全重复）；小说「角色
  剧照」零交互说明卡（一行并入存储目录 desc）；关于「系统信息」收成「存储
  路径」（引擎/API/推理强度三行与模型分组及模型中心重复）；api/settings.ts
  七个零消费死导出（TTS 五兄弟 + migrateProjectToV4 + voiceHealth）。
- **增一**：通用分组「界面语言」切换（跟随系统/简体/繁體/English）——i18n
  三语字典与 setPref 早已就绪、全应用却无切换入口；即时生效整树重渲染，
  desc 如实注明「各板块面板暂以中文为主」。
- **修一坑**：ImageGenPanel 补 comfyui_url/image_save_dir 回填——此前 comfyui
  后端直接保存会把已存 URL 清空。
- 验证：go 全绿、tsc/tsc -b 0、eslint 0/0、vitest 197/1393、drift PASS（559）、
  ?mock=1 走查语言双向切换（导航整树切换 + localStorage 三方核对）。

## v4.56.0 · 继续完善：拆分第二步 · mock 补 Herdsman 族 · 并行 task 卡关联（2026-09-03）
> 欠账池三线并行子代理（task 卡空 ref 欠账先 grep 核实为真）。**绑定面 559
> 零变更**。详见 releases/v4.56.0.md。
- **A 线**：ChapterPage 拆分第二步——applyTextHighlight/paraOf/textAtScrollTop
  搬 chapter/readingHighlight.ts（累计净减 ~111 行），ref 包装保留论证
  「可简化≠应简化」（3.6s 定时器窗口旧闭包误高亮）；+14 用例。
- **B 线**：mock 补 Herdsman 七方法（中性空态 + 生命周期诚实 ok:false），
  引擎管理「模型目录不可用/运行中不可用」横幅消除；契约 5 用例。
- **C 线**：task 卡 provider 升级 `(ref, args?)`，matchRunningRun 纯函数——
  并行多 running 时 args↔run.task 唯一命中才绑定，0/≥2 命中宁缺勿错；
  +13 用例，签名向后兼容。
- **收口**：主代理补 mock GetModelCallStats 空聚合（统计段横幅消除）。
- 验证：go 全绿、tsc -b 0、eslint 0/0、vitest 197/1392、drift PASS（559）、
  ?mock=1 走查无报错横幅。

## v4.55.0 · 继续完善：拆分补测起步 · mock 补面 · failover 文案 label 化（2026-09-03）
> 欠账池三线并行子代理 + 一条欠账 DOM 核实销账。**绑定面 559 零变更**。
> 详见 releases/v4.55.0.md。
- **销账**：右舷「虚线交叠」DOM 核实证伪（无 dashed 元素、文字无交叠）。
- **A 线**：ChapterPage 首次拥有测试——阅读搜索高亮三函数搬入
  chapter/readingHighlight.ts（净 -49 行，行为零变更）+ 15 用例（12 纯函数
  + 3 冒烟）；拆分第二步留待后续。
- **B 线**：dev mock 补 Get/SetEngineFailover（state+契约 4 用例）；收口补
  engines.ts App() 浏览器回退（?? bridgeApp，未实现方法照旧抛错走 catch）。
  走查：mock 下调度三开关全点亮，CUA 真实点击三方同步切换。
- **C 线**：failover toast 接入统一 engineLabel 解析（三级回退语义不变），
  新建 failover 测试锁 4 用例。
- 验证：go 全绿、tsc -b 0、eslint 0/0、vitest 196/1361（+4 文件 +24 用例）、
  drift PASS（559）。

## v4.54.0 · 继续完善：三线并行收欠账（2026-09-03）
> 欠账池互不相交三线并行子代理（文件所有权互斥）。全部存量清偿，无新功能。
> **绑定面 559 零变更**。详见 releases/v4.54.0.md。
- **A 线·办公面板**：任务中心空态副文案去造价语境占位；产物/变更次区分隔线
  去透明减弱（MergedPanel 测试锁保留）。
- **B 线·首页矩阵**：末行空位收整——「编程」升格 span 4×1 宽瓦片（横排门廊
  形态），30 单位=5 行整除满铺；四响应档单位数核算表入蓝图；宽窄档纯 CSS 降级，
  键盘/aria 体系零改动。
- **C 线·eslint 清零**：ConsistencyPanel 死状态收口、ChapterPage 三函数链式
  useCallback 稳定化补依赖（禁直接加防每帧重跑）、ImageGenPage 去模块级依赖
  ——**eslint 全量首次 0 error 0 warning**。
- 验证：go 全绿、tsc -b 0、vitest 192/1337、drift PASS（559）、judge 2/2 pass
  无 must_fix。

## v4.53.0 · 办公欢迎界面化繁为简：四点降噪 · 右栏 6→4（2026-09-03）
> 用户四点拍板。零功能删除，重复入口收敛；合并=上下分区直接并成一个面板
> （非二级标签）。**绑定面 559 零变更**。详见 releases/v4.53.0.md。
- **双删**：chat 头部上下文横条（ContextBar 组件连文件删除，上下文走主区
  标签页）；侧栏底部模型卡（FeatureModelBar 唯一消费方移除，组件/死 mock/
  死 CSS 规则一并清）；模型入口统一 chat 头部+顶栏 pill。
- **合并**：右栏 6→4（文件/产物/任务/浏览器）——产物=会话产物(上)+文件变更(下)、
  任务=任务中心(上)+分工(下)，MergedPanel 上下分区同屏全可见；旧持久化 id
  changes/subagents 三路径别名收敛；命令面板/设置卡随清单自动派生。
- 验证：go 全绿、tsc -b/eslint 0、vitest 192/1337（三层锁同步+MergedPanel
  新增）、drift PASS（559）、judge 视觉验收 3/3 pass 无 must_fix。

## v4.52.0 · 首页重设计「星枢港 · 双舷驾驶舱」（2026-09-03）
> 用户点名「使用技能重新设计首页界面」（ui-ux-pro-max AI-Native UI 方向 + 星枢
> 令牌体系）。零功能删除，纯重组降噪。**绑定面 559 零变更**。详见 releases/v4.52.0.md。
- **双舷骨架**：左舷 = 紧凑 Hero（左对齐工作台化，标题 40→27px）+ 命令条五要素
  （orb/打字/语音/发送/⌘K）+ 能力矩阵 Bento；右舷 300px = 内核遥测（模型/引擎/
  CPU·MEM·GPU 三表 + ComfyUI 徽标）· 写作进度环 · 最近会话 · 记忆脉搏 · 晨报。
- **v3 五段并三段（零删除）**：快捷 chips 收编进 Bento 一级面；状态细条+底部
  信息条合并为右舷面板；门廊编程（独立窗口徽标）/设置瓦片化编入矩阵尾部。
- **响应式**：≤1180 侧舷下落双列 + 旗舰单行横向排布（judge must_fix 修复）；
  降级链（焦点环/reduced-motion/raf-degraded）全保留。
- **i18n**：三语各 648 键——新增 3（kernel/sideAria/comfyRunning）、更新 3
  （title/sub/capSub）、删 22（含 9 个 v3 遗留死键，删前 grep 核实）。
- 验证：go 全绿、tsc -b/eslint 0、vitest 1332/1332（并行负载 flaky 两轮复跑
  全绿，失败集与本刀零接触）、drift PASS（559）、judge 视觉验收两轮 overall
  pass（4 图 + 复核）。

## v4.51.0 · 壳层左缘修复 · 创建青鸟助手深链直通绑定（2026-09-03）
> 「继续」欠账池双线并行（文件所有权互斥子代理）。**绑定面 559 零变更**。
> 详见 releases/v4.51.0.md。
- **A 线·壳层左缘**：根因=.v3-rail-dock 为 fixed 浮层热区、右列容器 x=0 起
  铺；右列预留 padding-left var(--v3-rail-w)（48px），悬停展开浮层语义不变。
  全板块最左列首字符裁切修复。
- **B 线·深链跳绑定**：创建青鸟助手成功 → sessionStorage 焦点（wxFocus.ts，
  读后即清）→ NAVIGATE(crossSpace) 跨空间落青鸟 → 选中未绑定新助手 → 直开
  扫码绑定。收口修复真集成缺口：跨空间被 S2.1 同空间守卫静默丢弃，MainLayout
  新增显式 crossSpace 分支走 navigateBoard 换空间（默认语义不变）。
- **附带**：造价概览「最近更新」行内 mini 条空态长文案窄列叠压修复（judge
  走查发现）。
- 验证：go 全量绿、tsc/tsc -b/eslint 0、vitest 1332/1332（+5 深链用例）、
  drift PASS（559）、judge 视觉验收两轮 overall pass（5/5 图）。

## v4.50.0 · 造价数据库化繁为简：8→6 模块 · 询价库升格 · 概览收镜头（2026-09-03）
> 用户点名「造价数据库优化，核心是化繁为简」。简化≠删功能：归并 + 降级，
> 每个旧入口都可达。**绑定面 559 零变更**。详见 releases/v4.50.0.md。
- **IA 重组 8→6**：价格源/价格仓库/询价库归并为「价格数据」三段同域
  （role=tablist）；知识图谱降为概览的「关联图谱」镜头（导航右端双胶囊，
  渐进披露）；模块剩 概览/成本条目/测算项目/价格数据/造价参考/复盘笔记。
- **询价库升格**：从「成本条目」第三个无文字 icon 视图移入价格数据一等子页
  （询价飞轮四源归一本就是价格数据域）；成本条目回归纯库管理（列表/表格）。
- **概览降噪**：快捷入口 7→4；「回到概览」纯文字（judge 指出 RefreshCw 读作
  「刷新」有歧义）。
- **CostLibraryView 瘦身**：拆 inquiry 模式；拆 compact/onInsert 死管线
  （零调用方核实后删）；删不可达 memoryhub/CostLibrary 包装层 + import test。
- **dev mock 补询价/五算九方法**（走查抓到真缺口：CostInquiryExpiring 缺失
  直接崩）；种子含到期预警 + 调差演示。
- 验证：go 全量绿、tsc/tsc -b/eslint 0、vitest 1327/1327（CostLibraryPage
  9 场景新增）、drift PASS（559）、judge 视觉验收两轮 overall pass（7/7 重拍）。

## v4.49.0 · 青鸟助手生命周期补全：编辑助手 · 角色卡一键创建 · 镜像守卫（2026-09-02）
> 「继续」欠账池双线并行（文件所有权互斥子代理）。**绑定面 559 零变更**。
> 详见 releases/v4.49.0.md。
- **① 编辑已有助手**：通道详情新增「编辑」（改名/换人格，复用人格选择器抽
  出的 PersonaPickerPanel，新增/编辑两流共用）；编辑流人格预填当前值、自定
  义角色 id 不在选项里时保留原值不重置；保存以 viewOf 完整对象叠加，立绘仅
  在新选带立绘选项时覆写（契约：空值不覆写）；核心锚点 gaea 不开放编辑。
- **② 角色卡一键创建青鸟助手**：角色库 custom 角色卡新增「创建青鸟助手」
  （Popconfirm 确认）→ WhisperAssistantSave 直建未绑定助手（wxToken 空），
  提示到青鸟板块扫码绑定；assistant/builtin 卡不显示。
- **③ 镜像守卫（真隐患修复）**：EnsureAssistants 对 Kind=custom 的命中行
  整行跳过——v4.48 起助手 personalityId 可指向 custom 角色，原逻辑会在下次
  启动把该角色名字/kind 镜像覆写冲掉；现在 custom 角色绝不被镜像触碰，
  builtin/assistant 命中与未命中新建行为不变。
- 验证：go 全量绿、tsc/tsc -b/eslint 0、vitest 1319/1319（WeixinPage 10 场
  景 + CharacterCard 14 用例 + store 守卫用例）、drift PASS（559）、judge 视
  觉验收 2/2（编辑弹窗默认态/换人格切换）。

## v4.48.0 · 青鸟：板块更名 + 人格选择器打通角色库（2026-09-02）
> 用户拍板：①微信助手更名（太直白）→「青鸟」——青鸟传信，与绘梦/轻语板名
> 同气质；②新增助手选人格时显示详细信息方便选择。**绑定面 559 零变更**。
> 详见 releases/v4.48.0.md。
- **板块更名**：manifest label 前后端「微信助手」→「青鸟」（board id/page/
  绑定名不动）；意图别名表加「青鸟」（原名兼容）；页内文案同步（新增青鸟
  助手/青鸟导航等），启动器描述改「青鸟传信 · 微信遥控器」。
- **人格选择器**：新增助手弹窗从「手填人格 ID 文本框」升级为双栏选择器——
  左侧搜索 + 分组列表（轻语预设 + 角色库可聊天角色，18+ 人格不列出），右侧
  详情预览（立绘/名字/来源与性别 Tag/标签/口吻或人设背景）。
- **立绘回显**：选中角色库人物后 portraitUrl 随暂存对象落库，轨道与详情
  头像自动用真立绘（portraitUrl 字段既有，此前 UI 不喂）。
- **配套**：CharacterList/WhisperGetPersonalities 从 legacy wailsjsCompat
  直调转正 AppBindings（登记 work 分面，浏览器 dev mock 可达）；dev mock 补
  预设/角色数据；WeixinPage 测试 9 场景（新增选择器场景 + 更名适配）。
- 验证：go 全量绿、tsc/tsc -b/eslint 0、vitest 1316/1316、drift PASS（559）、
  judge 视觉验收 5/5（更名/选择器默认态/角色详情/搜索过滤/扫码弹窗）。

## v4.47.0 · 微信助手星枢化：通讯枢纽工作台（2026-09-02）
> UI 重构刀——三卡堆叠管理台升级为 Constellation OS「通讯枢纽」工作台，
> **功能零删减、绑定面 559 零变更**。详见 releases/v4.47.0.md。
- **三分区布局**：玻璃细条（板块名 + 通道遥测 meta + 刷新）/ 左通道轨道
  （每助手一条 rail item：头像+状态字+状态点，激活=主色容器+左缘光条）/
  主区三视图（通道详情 · 离线提醒 · 使用指南）；整卡介绍收敛为一句副标语。
- **通道状态三重传达**：状态点四态（运行中发光脉冲，reduced-motion 降级/
  会话过期/已停止/未绑定虚线）+ 轨道状态字 + 详情键值卡（通道状态/微信绑定/
  启停）同口径；会话过期警示内联到对应通道详情。
- **扫码绑定流**：三步指示（扫码→确认→完成）+ 二维码容器化；新增助手确认
  后主区直接落在新通道。
- **配套**：dev mock 补微信域（mock/weixin.ts，伪二维码+扫码相位推进，
  WeixinPage 离线可开发）；设计蓝图 design-system/gaea/pages/weixin.md 落地，
  misc.md 过时定位拨正。
- 验证：go 全量绿（110 包）、tsc/tsc -b/eslint 0、vitest 1315/1315
  （WeixinPage 8 场景）、drift PASS（559）、judge 视觉验收 6/6（实机走查）。

## v4.46.0 · 小说板块第二轮：伏笔闭环 · 导出扩展 · 伴读多轮 · 一致性 AI 深检（2026-09-02）
> 接 v4.43.0 欠账池四条互不相交线，全为既有功能闭环与生成质量。**绑定面
> 557→559（+SaveForeshadows、+CheckConsistencyDeep）**。详见 releases/v4.46.0.md。
- **① 伏笔登记表闭环**：SaveForeshadows 写回+校验；syncForeshadows 按 ID 合并
  （手工条目永不被冲）；面板登记/流转/编辑/删除（乐观更新+回滚）。
- **② 导出扩展**：EPUB 真封面（cover-*.png 自动探测+SetCover，无则回退）；
  onlyMainline 参数（默认含分支兼容）；DOCX 导出落地（gooxml，formats 3→4）。
- **③ 伴读多轮化**：historyJSON 最近 6 轮；划线窗口优先 12000 rune；弹窗
  会话化（按章保留/切章清空/失败回滚）。
- **④ 一致性 AI 深检 v0**：逐章状态卡提取+本地跨章比对（死者复现/物品凭空/
  时间倒流等五类），与规则层合并带 source 徽标；诚实降级 ai_available=false。
- 验证：go 全量绿（110 包）、tsc/tsc -b/eslint 0、vitest 1312/1312、
  drift PASS（559）。

## v4.45.0 · 百炼全量下线：引擎 + 配置 + UI + 微信改图链（2026-09-02）
> 用户拍板「把百炼删除干净」——v4.40.0 对话式改图的百炼(DashScope)链路全量
> 退役，img2img 保留 ComfyUI / Herdsman 本地后端。**绑定面 557 零变更**。
> 详见 releases/v4.45.0.md。
- **引擎与配置**：image_dashscope.go 退出注册表；DashScopeAPIKey 字段/迁移/
  initImageBackend 分支全删；SetImageBackend 第 5 参彻底移除（4 参，方法名不动）。
- **imagegen UI**：引擎枚举去 dashscope（meta 单源）+ 门禁文案更新；ResultStage
  新增「改图」动作（结果图作参考图发起 img2img）；回归锁断言 dashscope 不在枚举。
- **微信改图链全删**（v4.9 起）：ActionEditImage 意图解析/exec/seam/入站图旁路
  缓存（wx_image_cache 整文件）/wx_agent edit_image 工具（7→6）/
  editImageFromCard；copyFileBounded 迁 wx_file_handler.go。
- **基建**：gen_bindings 空白参数名合成占位名（`f(_)` 编译错误根因）。
- 验证：go 全量绿（110 包）、tsc/tsc -b/eslint 0、vitest 1312/1312、drift PASS（557）。

## v4.44.0 · 绘梦专项三刀：百炼模型残留修复 · 引擎枚举单源化 · 模板画幅落地（2026-09-02）
> 绘梦板块专项摸底后收三条互不相交线，全为真实缺陷与体验收口，绑定面
> **557 零变更**、零新增绑定。详见 releases/v4.44.0.md。
- **① 百炼模型残留修复（真 bug）**：dashscope 后端 GetImageBackendInfo /
  SetImageBackend 空或残留模型（grok-imagine-* / krea2）归位 qwen-image-edit-plus
  （手填官方编辑系保留）；前端 modelOptions 固定三档官方编辑模型 + 切后端归位
  默认；queue 提交前 backendSupportsMode 拦截引擎固有模式残留（百炼仅改图 /
  GLM 仅文生图），不再点击后才被后端拒收。
- **② 引擎枚举单源化 + 补 GLM**：meta.ts 收敛唯一 BACKEND_OPTIONS（能力位
  img2imgOnly/txt2imgOnly）+ backendLabel/isLocalBackend/backendSupportsMode；
  ControlPanel 下拉、ImageGenPage 顶条状态、useImageGenConfig 启动消息、
  GenerationBar 门禁统一走单源（修复「xAI 云端 云端」拼接重复）；引擎下拉新增
  GLM（txt2imgOnly，非文生图禁用+残留专属警告）。
- **③ 模板推荐画幅落地**：templateSizeToPreset 纯函数把模板 size 比例标签
  （1:1/16:9/…/2:3）映射到实际画幅（2:3 立绘→自定义 768×1152，仅 txt2img 生效）；
  applyTemplate 同步画幅、TemplatePickerModal 不再丢弃 size 字段。

## v4.43.0 · 小说板块优化四刀：生成上下文 · 分支收账 · 搜索定位 · 一致性修复（2026-09-02）
> 小说板块自 v4.3.1 后首次专项投入：四条互不相交线全为生成质量与既有功能
> 闭环，零 UI 结构变更。**绑定面 557 零变更**。详见 releases/v4.43.0.md。
- **① 章节生成上下文增强**：prompt 追加「未回收伏笔（创作约束）」「世界观要点」
  区段（读失败静默跳过不阻断）；角色卡性格 20→60 rune + 身份/目标/关系；
  摘要 100→200、续写尾部 500→1500 rune；4000 rune 总预算双层截断。
- **② 分支链路收账**：分支结果持久化 branches.json，ApplyBranch 读存储应用
  主路径零 AI 重调（旧数据回退+注明可能不一致）；syncCharactersFromOutline
  从 no-op 变真同步（幂等物化角色卡）。
- **③ 全文搜索升级**：每章全部命中（20/章、300 总，统计不受上限影响）+ 段
  级位置字段；前端「共 N 处 · M 章」+ 跳章滚段定位临时高亮（降级全文首中）。
- **④ 一致性两 bug**：分支章节纳入扫描且告警带 Branch 标记（分线统计不混判）；
  章节断档不再 break 停扫。
- **死代码清理**（无 UI 变化）：删零引用未挂载组件 4 件 + 孤儿 api/search.ts。
- 验证：go 全量绿、tsc/tsc -b/eslint 0、vitest 186 文件 1274/1274（+15）、
  drift PASS（557）。

## v4.42.0 · 微信智能体 v1：LLM 工具调用派发（2026-09-02）
> 定位用户拍板：微信消息由模型自己派发任务给各板块、结果回微信；关键词路由
> 降级为兜底（v4.41.x 两轮真机补丁皆是正则路由的账）。**绑定面 557 零变更**。
- 新增 wx_agent.go：7 工具（导航/生图/改图/提醒/产物推送/状态/读屏），exec 适配
  层零改动复用意图执行函数；ChatStream tools 透传+流式 tool_calls 既有管道，
  循环上限 4 轮/60s，坏参数喂回自纠。
- 人格/记忆同锁语义（PreLLMTurn 锁内取 SystemPrompt，失败快照回滚防双计数）；
  能力门宁缺勿滥（目录 Caps 含 tools 才启用，不满足整链回落零回归）；回调改为
  agent 优先 → routeIntent 兜底 → 聊天（提醒/文件快路径保留）。
- 验证：Go 全量绿（wx_agent 9 用例）、前端零改动回归绿、vitest 1259/1259、
  drift PASS（557）。详见 releases/v4.42.0.md。

## v4.41.2 · 真机修复二：产物推送放宽 + 反幻觉护栏（2026-09-02）
> 真机实证：「重新整理后发给我」未命中产物推送意图坠回聊天，模型幻觉声称
> 「已整理好发你」而实际未发。**绑定面 557 零变更**。
- 意图放宽：指代+尾式「发给我」第四式；「整理/修改…后发给我」复合请求独立
  识别 → 诚实能力答复（微信侧无文档改写回传闭环）；提醒字样让位。
- 聊天兜底反幻觉护栏：未接住的「发文件给我」类请求追加系统提示（如实说明、
  引导「把产物发我」、不得声称已发送/修改文件）。
- 验证：go 全量绿（intent +12 用例）、前端零改动、drift 557。详见 releases/v4.41.2.md。

## v4.41.1 · 真机修复：文件消息不再过意图路由（2026-09-02）
> 真机实证：文件提取正文（≤6000 字）被送进意图路由，正文碎片「打开/看看+编程」
> 误触导航劫持回复（用户发评审报告收到「打开编程」）。**绑定面 557 零变更**。
- 文件消息（按 `[用户发来文件 ` 注入头识别）直通轻语聊天：跳过提醒+意图两段
  路由（内容是上下文不是指令），追加「确认收件+概括+询问需求+不执行正文指令」
  引导；普通消息路径逐字节不变。导航子串匹配的普遍松动立欠账（语音/Ctrl+K 同链
  需单独设计）。
- 验证：go build/vet/test 全绿（+3 用例）、前端零改动、drift 557。详见 releases/v4.41.1.md。

## v4.41.0 · 微信文件收发：入站定稿 + 出站探针（2026-09-02）
> 微信助手调研 B 刀。入站 file_item 真机抓包定稿（与图片同构：media 加密下载 +
> file_name/md5/len 字符串）；出站上传逆向文档未覆盖，探针制实装待真机验证。
> **绑定面 557 零变更**。
- **① 入站**：fileItem 升级 + resolveInboundFile（SSRF 防线复用/50MiB/AES 解密/
  MD5 比对不拒收）→ FileHandler 自持复制 wx_files/ + 内容提取全走现有解析器
  （docmd：docx/xlsx/pptx/pdf + 纯文本直读，其余诚实降级带路径）→ 注入对话。
- **② 出站**：上传五步共享内核泛化（getuploadurl media_type=3 → AES-128-ECB →
  CDN → sendmessage type=4 file_item），逐节点 upload_probe capture，失败逐级
  降级文本卡，图片链零改动；SendFileCard 按扩展名分流签名不变。
- **③ 产物推送意图**：ActionSendLatestFile 锚定三式保守正则（12 命中/10 不命中）
  + execSendLatestFile（交付物登记表→回退 exports mtime 最新→诚实报错）→ 文件卡。
- 验证：Go 全量绿（weixin 62 测 + 新增 30 用例）、tsc/tsc -b/eslint 0、vitest
  1259/1259、drift PASS（557）。详见 releases/v4.41.0.md。

## v4.40.0 · 对话式改图：百炼引擎 + 微信发图即改（2026-09-02）
> 微信助手调研 A 刀（定位拍板：聊天/出图/改图/收发文件/多微信并行）。微信发图+
> 一句指令→编辑出图→图片卡回推。官方契约核实后实装（禁止 OpenAI 习惯外推）。
> **绑定面 557 零变更**。
- **① 百炼引擎**：DashScope 多模态生成端点同步实装（单图单文/官方字段/改图不传
  size 保原图比例/24h URL 自动下载转 data URL），默认 qwen-image-edit-plus，
  kind=dashscope 注册表自注册，仅 img2img。
- **② Key**：config `dashscope_api_key` 密文落盘+旧明文迁移；SetImageBackend 追加
  第 5 参（空=保留存量）。
- **③ 意图+接线**：ActionEditImage 动词∧指代双门槛保守正则（宁漏勿误）；入站图
  旁路 hook（OCR 识图链路零改动）→助手级图片缓存（自持副本/TTL 10min/只留最新）；
  routeIntentForAssistant 内部变体；未命中不接管回落聊天；产物走 CardPath 图片卡。
- **④ 前端**：绘梦引擎「百炼改图」选项（txt2img disabled+Tooltip）+img2img 门禁
  三处白名单+设置页 Key 框（只写不读）。
- 验证：Go 全量绿、tsc / tsc -b / eslint 0、vitest 1259/1259（+11，ProgrammingPage
  一例负载 flaky 单跑/整包复跑绿）、drift PASS（557）。详见 releases/v4.40.0.md。

## v4.39.0 · 微信助手管理台：多微信并行 + 并发正确性两修（2026-09-02）
> 微信助手专项调研（docs/market-research-2026-09-02b.md，定位用户拍板纠偏：聊天/出图/
> 改图/收发文件/多微信并行）C 刀——收「多微信并行」：后端 CRUD 齐备前端无管理台的
> 历史缺口 + 两处并发/持久化真缺陷。**绑定面 557 零变更**。
- **① 管理台**：WeixinPage 连接卡升级助手管理台——助手卡（Avatar portraitUrl/首字
  回退 + 人格 Tag + 通道状态徽标）+ 启停 Switch + 删除 + **逐助手扫码绑定/重绑**
  （修 confirmBinding 硬编码 id:'gaea'）+ 新增微信助手表单（wx_ 前缀动态 id）；
  gaea 核心助手禁删禁停；Status+List 双路 merge。
- **② Update 写回扩展**：manager.Update 补回写 WxBotID/PortraitURL/VoiceGuide/
  Gender/Tags/Dims——空值保留现值（防部分保存清空）。
- **③ 凭据防御**：WhisperAssistantSave 更新路径空 wxToken/wxUserId 保留旧值
  （启停切换零凭据风险），补全凭据同步用于通道重启。
- **④ 并发正确性**：同人格多助手共享 orchestrator 的 AssistantName 锁外直写（数据
  竞争+互相覆盖）修复——聊天链重构为内部 whisperChatWithSearch/whisperChat（透传
  assistantName），注入移入 LockTurn 持锁窗口；微信回调改走 whisperChatAsAssistant；
  绑定签名零变更。
- 验证：Go build/vet/test 全量绿（+5 用例）、tsc / tsc -b / eslint 0、vitest
  1248/1248（+5）、drift PASS（557）。详见 releases/v4.39.0.md。

## v4.38.0 · 目录通用化：DeepSeek/xAI/Zen 官方元数据入册（2026-09-02）
> 用户指出 v4.36.0 目录只覆盖 GLM——通用化到 deepseek/xai/opencode-zen（25 条目，
> 官方页多页互证核实）。内置表（CCSwitch 预设）过时暴露：deepseek-chat/reasoner 官方
> 2026-07-24 停用、grok-4/3 系已下架、deepseek 现行价差 3 倍+。**绑定面 557 零变更**。
- **① 通用目录**：model_catalog.json v1（per-engine 分组）+ catalog_models.go loader；
  **opencode-go 不进目录（拍板）**：订阅制无按量售价，展示参考价会误导。
- **② 估算修正**：estimatePrice 目录优先层扩展；deepseek-v4-flash 3.0→12.672 CNY（官方
  USD 峰价）、grok-4.5/4.6 129.6→57.6、zen 8 模型从未计价→计价；claude/gpt/gemini/kimi
  旧条目走内置表逐位一致（回归锁）。
- **③ 动态列表 enrich**：fetchModels 后按 id 归一化匹配补元数据（只填空不覆盖），
  deepseek/xai/zen 模型卡徽标自动点亮（前端零改动，B 刀链路）。
- 验证：Go 全量 test exit 0（filewatch 一例全量超时为负载 flaky，单跑/整包复跑绿）、
  tsc -b/eslint 0、vitest 1243/1243、drift PASS（557）。详见 releases/v4.38.0.md。

## v4.37.1 · D 刀收口：模型库卸载确认带释放大小（2026-09-02）
> 模型中心调研 D 刀核伪后剩余增量（绑定面 557 零变更）：卸载确认 file_size 存在时
> 显示「释放 X GB」；磁盘占用展示与 /health 透出两项经核实已存在（E1-4 已落地、
> TestConnection 已探 herdsman），剔除不入刀。
- 验证：go build/vet、tsc -b 0、vitest 模型中心目录绿、drift PASS（557）。详见 releases/v4.37.1.md。

## v4.37.0 · 健康巡检 + 故障转移 v0（2026-09-02）
> 模型中心调研 C 刀：巡检/转移在桌面与 Web 端竞品中全部空白，gaea 以既有 Status
> 持久化与 fallback 路由语义低成本补位。**绑定面 555→557（+2）**。
- **① 健康巡检**：goroutine 10 分钟周期探已启用非本地引擎（GET /models 8s 超时），
  Status 持久化+「连续 N 次探测失败」前缀+状态变化事件 engine-health-changed；
  Error 永不含 Key（对抗回显测试锚定）。
- **② 故障转移 v0**：开关 engine_failover_enabled（默认关=现状逐字节）；网络类/408/429/5xx
  才转移（401 等配置错误不转移），候选=已连接 llm 引擎按 order 取首、用其默认模型重试一次，
  流式仅首字节前转移；emit model-failover + 逐笔记账。
- **③ 前端**：「本地调度」第三开关卡「故障转移」（降级「未知」禁用）+ 双事件订阅 +
  总览引擎卡连续失败原文/巡检时间 title。收口注记：gen_bindings explicitOverrides
  登记教训（新方法先登记再生成，否则被前缀规则误归）。
- 验证：Go 全量 test exit 0、tsc -b/eslint 0、vitest **1243/1243**（+6）、drift PASS
  （**557**）。详见 releases/v4.37.0.md。

## v4.36.0 · GLM 目录 v2：能力/价格元数据 + 远程热更新（2026-09-02）
> 模型中心调研 B 刀（=「计费三件套」本体升级版）。官方价格页动态渲染抓不到正文，
> 查不到的绝对价一律不编数，估算沿用内置 z.ai USD 价口径。**绑定面 555 零变更**。
- **① 目录 schema v2**：glm_catalog.json 44 条目（legacy 22+官方新增 22），条目带
  context_length/max_output/price/currency/unit/free/caps/price_note/coding 积分系数
  （仅 glm-5.3 与 5.3-flash）；免费档 8 个；官方核实国内价 6 条；解析兼容旧裸数组。
- **② 远程热更新 v0**：config 键 glm_catalog_url（默认空禁用）+ 24h 周期拉取 + version
  比对 + 本地缓存兜底；优先级 覆盖文件>远程>内嵌；仅影响展示与估算，不碰路由/alias/鉴权。
- **③ 估算单源化**：estimatePrice GLM 分支目录优先、内置表兜底——GLM 价格更新只动目录
  不发版；估算值零回归锁；ModelStatsSummary 透传 catalog_version/source 三态。
- **④ 前端**：模型卡上下文/能力/价格徽标（免费绿标），StatsSection coding 行官方公式
  估算积分（含缓存命中通道）+ 价格目录来源小注。
- 验证：Go 全量 test exit 0、tsc -b/eslint 0、vitest **1237/1237**（+18）、drift PASS
  （555 零变更）。详见 releases/v4.36.0.md。

## v4.35.0 · 自定义引擎：OpenAI 兼容服务商任意添加（2026-09-02）
> 模型中心专项调研（docs/market-research-2026-09-02.md）A 刀：自定义服务商是桌面客户端
> 4/5 家标配，gaea 8 引擎硬编码为最大硬缺口。**绑定面 552→555（+3）**。
- **① 引擎类型与生命周期（Go）**：新类型 EngineCustom="custom"（OpenAI 兼容）+ Manager
  六方法（Add/Update/Remove CustomEngine + customKeys 注入/取用）；engineID=custom-前缀
  +slug 冲突追加序号；Key 存 config 新键 custom_engine_keys（JSON map 加密值，
  saveSetters 登记+显式往返测试），不落 engines.json 不下发前端；baseURL 校验 http(s)
  +host（v4.9.1 Key 粘错框防线延伸，Key 当地址粘入被拒）；LoadState 恢复 custom 条目
  （type+地址合法性双校验防伪造）。
- **② 聊天路径**：BuildChatURL/resolveChatEndpoint（流式+非流式）custom 分支——
  自定义引擎真正可聊天/设活跃/绑功能；空 Key 不发 Authorization 头（无鉴权本地服务可用）。
- **③ 前端**：引擎管理分区「添加自定义引擎」表单（前后端同口径校验双保险）+ custom 卡
  地址框/编辑（Key 留空=不改）/删除确认/「自定义」徽标；内置云端引擎地址框防线一字未动
  （新增回归锁）。
- 验证：Go 全量 test exit 0、tsc -b 0（bridge.ts LegacySurfaceNames 登记 3 名）、
  eslint 0、vitest **1219/1219**（+9）、drift PASS（**555**）。详见 releases/v4.35.0.md。

## v4.34.0 · 子代理气泡恢复：恢复会话不再丢失子代理答复（2026-09-02）
> 收 v4.26 沿旧欠账「子代理气泡恢复暂缺」。根因=ProjectMessages 投影无 subagent_message
> case 整条忽略；模型面投影不可动（恢复后模型上下文须与实时语义一致）→ UI 侧并行投影。
> **绑定面 552 零变更**。
- **① UI 侧锚点投影（Go）**：session 新导出 KindSubagentMessage 常量 + ProjectSubagentAnchors
  （与 ProjectMessages 逐 case 同拍的游标镜像，subagent_message 记「插在第 K 条消息后」
  锚点，projection.go 一字未动）；GaeaHistory 读磁盘事件日志按锚点合并子代理气泡
  （mergeSubagentAnchors 纯函数 + logOffset 校正检查点 system 提示导致的系统性偏移，
  负位/越界锚点宁漏勿误丢弃）；HistoryMessage 加 subagentRef 字段（golden 不变）；
  GaeaResumeSession 零改动自动生效。
- **② 恢复徽标消费（前端）**：rebuildHistoryItems assistant 分支透传 subagentRef（空串
  归一），复用实时「子代理」徽标渲染；HistoryMessage 类型交叉扩展注明可回收。
- 验证：Go 全量 test exit 0（线A -count=2，投影/Restore/golden 回归全绿）、tsc -b/eslint 0、
  vitest **1210/1210**（+3）、drift PASS（552）、wails generate module 已刷新、版本四处
  4.34.0。详见 releases/v4.34.0.md。

## v4.33.0 · 细节收口第三刀：回滚守卫统一 / pdf 占位比精确化 / 主区预览懒加载对齐（2026-09-02）
> 欠账池三线并行子代理 + 主代理集成。**绑定面 552 零变更**。
- **① 回滚守卫统一 + write_file >8KB 恒误报修复（真 bug）**：rollback 卡接入「恢复后已被
  手工修改」守卫（撤销恢复前校验，防覆盖编辑）；write_file 守卫从精确比较改
  `evidence.ClampSummary` 同口径截断比较——原精确比较对 >8KB 未手改文件必然误报拒绝；
  截断单点化（RecordChange/app 复用）。已知边界：8KB 摘要窗口外手改不可检（宁漏勿误）。
- **② pdf 占位比按实测精确化**：pageLazy 新 nextPageAspect/placeholderAspect——页图
  onLoad 读 naturalWidth/Height，首个有效测量为整档比例（不被后续页推翻），无测量回落
  A4 估计；占位→真身交换不再跳高、滚动条比例修正。弹窗 FilePreviewModal 已接线。
- **③ 主区预览 pdf 懒加载对齐弹窗**：FilePreview pdf 分支接入同款 IO 单向懒加载（初始
  4 页/800px 预挂/不卸载/大纲跳转强制渲染/ref 回调登记即补 observe），无 IO 全量降级；
  主代理集成追加测量比例接线，占位表现与弹窗一致。
- 核实：v4.26「TrajectoryView 未消费 subagent 记录」欠账已过期剔除；「子代理气泡恢复」
  留作独立刀（GaeaHistory×事件日志跨源对齐）。
- 验证：Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest **1207/1207**（+11）、
  drift PASS（552）、版本四处 4.33.0。详见 releases/v4.33.0.md。

## v4.32.0 · 细节收口第二刀：回滚可撤销 / 产物自动弹出 / 弹窗 pdf 懒加载 / 预览最大化持久化（2026-09-02）
> 用户点名「继续优化完善 gaea」——欠账池挑四条互不相交线，三并行子代理 + 主代理 App.tsx 接线集成。**绑定面 552 零变更**。
- **① 回滚先快照当前态**（收 v4.28 B1 欠账）：GaeaRollbackRecord 恢复前把目标当前内容
  快照到原基线同目录（evidence 新导出 StageBaselineTo，命名逻辑单点化），rollback 记录
  升级为完整证据卡（Before/After 原文+BaselinePath）——**恢复动作本身成为时间线里可再
  恢复的版本（撤销恢复=对 rollback 卡再点恢复）**；目标缺失/快照失败降级不阻断恢复。
- **② 产物自动弹出 + 偏好**（收 v4.30 欠账）：新 deliverablePrefs（gaea.deliverableAutoOpen，
  **默认关** opt-in）+ DeliverablesPanel 头部胶囊；App 新产物 diff 时偏好开且 tab 未停用
  →亮右栏切「产物」tab（激活即清零角标，不动 FilePreview）；单版本「版本」徽标 title
  细化为「有 N 个历史快照，可预览/恢复」（收 v4.31 欠账）。
- **③ 弹窗 pdf 逐页懒加载**（收 v4.31 欠账）：新 lib/pageLazy 纯函数 + IntersectionObserver
  单向懒加载（初始 4 页、进视口 800px 预挂、已挂载不卸载杜绝滚动跳动），大纲跳转目标页
  强制即时渲染，无 IO 环境全量降级；顺带修 preview/loading 两次提交致 IO 观察集为空、
  懒加载永不触发的真 bug（ref 回调登记即补 observe）。
- **④ 预览最大化持久化**（收 v4.30 欠账）：gaea.previewMaximized 独立键（writePrefs 只落
  sizes 数字 map），懒初始化 + toggle/拖拽退出三处落盘；半幅宽度本就落盘，还原仍回上次
  半幅。
- 验证：Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest **1196/1196**（+30）、
  drift PASS（552）、版本四处 4.32.0。详见 releases/v4.32.0.md。

## v4.31.1 · -count>1 全量绿化：测试全局态 -count 不兼容根治 + whisper 末气泡真 bug 修复（2026-09-02）
> v4.31.0 线 D 收尾延伸：全量 go test -count=2 ./... 从 FAIL → 全绿。**绑定面 552 零变更**。
- **根因（统一）**：测试写进程级全局状态（provider/billing/boot/app 注册表 kind、
  whisperSessions 会话缓存），-count 多次运行不兼容；whisper 10m 超时为**真 bug**。
- **修法**：注册 kind 改 testKind(prefix)（进程级 atomic 单调计数，任意 -count 唯一，19 注册
  点）；app whisper 会话隔离改唯一会话 ID + t.Cleanup 清理缓存（12 调用点）；whisper
  PacedStreamEmitter.pump streamDone 分支收尾末气泡（+3 生产行，修 MarkDone 挂起/末气泡
  OnBubbleEnd 永不触发）。
- 验证：五包 -count=2/-count=5 全绿、发射器 -count=300 全绿、tasks -count=20 仍全绿、**全量
  go test -count=2 ./... exit 0**；前端零改动；drift PASS（552）；版本四处 4.31.1。详见
  releases/v4.31.1.md。
## v4.31.0 · 细节收口四线并行：单版本入口 / 弹窗 pdf 预览 / 历史轮耗时 / tasks 竞态根治（2026-09-02）
> 用户点名「并行使用子代理」——四线足迹互斥并行落地 + 主代理集成。**绑定面 552 零变更**。
- **① 产物版本时间线单版本入口**（收 v4.28 B1 欠账）：徽标条件从 {rev && …} 放宽为
  {(rev || journalEntry) && …}——versions>1 按现状 vN 徽标（旧锁不破），versions≤1 但有
  journal 快照（baselinePath）的产物渲染「版本」入口徽标（title 区分「更新 N 次」与「有版本
  历史」），无快照保持空态；VersionTimeline 本体零改动。
- **② FilePreviewModal pdf/pptx 逐页预览**（收 v4.28 欠账）：弹窗 kind="pdf" 分支补齐逐页
  缩略（data-pptx-page 锚点）+ dataUrl 整本回退 + 诚实空态 + PptxOutline 大纲卡（页锚点滚动/
  「针对第 N 页修改」composer 插入）；FilePreview.tsx 本体零改动，非 pdf 分支逐字节未动。
- **③ 轨迹历史轮耗时**（收 v4.26 欠账）：后端 Turn.DurationMs（fold turn_done 分支
  Ts>StartedAt 时 =差值×1000，omitempty 向后兼容）+ 前端 TrajectoryTurn.durationMs +
  TrajectoryView 轮次头「用时 Ns」（复用 formatElapsed）；零新增绑定（结构字段级）。
- **④ TestCancelConcurrentStress flaky 根治（实现层真竞态）**：根因=pickNext 不做任务级预留
  →多 worker 同时 execute 同一 queued 任务，claim 落选者无条件 unregisterCancel 删掉
  cancelReq（用户取消意图）→ Cancel 已成功返回的任务终态被 succeeded 吞掉。修复=tasks.go
  新 clearStaleCancel（只清残留预注册、绝不删 cancelReq）+ claim 成功后胜者重登记 cancel；
  测试改事件驱动等待（50 终态事件到齐），断言不削弱且双向加固（Cancel==nil ⇒ cancelled /
  未取消 ⇒ succeeded / 终态事件不重不漏）+ 确定性契约单测 TestClearStaleCancelOwnership。
- 验证：Go 全量 0 FAIL（tasks -count=20/100 两轮全绿、-count=3 全绿）；tsc/tsc -b/eslint 0；
  vitest **1166/1166**（+9：A3+B3+C3）；drift PASS（552）；版本四处 4.31.0。详见
  releases/v4.31.0.md。
- 发布后补充（-count>1 全量绿化）：billing/boot/provider duplicate kind → testKind 唯一化；
  app whisper 会话缓存 t.Cleanup 清理 + 隔离测试唯一会话；whisper PacedStreamEmitter
  MarkDone 末气泡收尾（+3 生产行，修末气泡 OnBubbleEnd 永不触发的真 bug）；**全量
  `go test -count=2 ./...` 与 `-count=5 ./...` 均 FAIL → 全绿（exit 0）**；tasks
  `-shuffle=on -count=10` 无顺序依赖。
## v4.30.0 · 办公 UI 化繁为简第二刀：产物置前 / 行级降噪 / 命令面板视图重排 / 预览两档（2026-09-02）
> 用户点名「继续优化完善 gaea」，收 v4.29.0 欠账四项，红线不变：简化界面不是删除功能。
> **绑定面 552 零变更**（纯前端呈现重组）。
- **产物生成自动置前/角标**（Devin Auto-open 式）：App diff 会话内新产物路径 → 产物 tab
  角标（未查看数，激活即清零）+ 产物面板行「新」徽标与高亮（data-fresh 锚点可测）；
  会话切换重置基线，恢复会话不误标「新」。
- **面板行级降噪**（Cowork 一行式）：产物/变更/任务三列表次级信息（路径/相对路径/时间/
  重试计数）改悬停次行显现（group-hover opacity 过渡），title 全保留；主行断言零改动。
- **命令面板按当前视图重排**（Linear 式）：新 lib/paletteRank 纯函数——当前激活右栏面板
  cmd 置顶、chatTab=overview 时概览置顶、其余稳定保序；CommandPalette 零改动。
- **预览半幅↔最大化两档**（VS Code Toggle Maximized Panel）：FilePreview 头部新增最大化/
  还原按钮（icons 补 Maximize2/Minimize2），最大化占满可用宽度、还原回半幅（ref 记忆）、
  拖拽分割条自动退出最大化；不传回调不渲染按钮向后兼容。
- 验证：Go build/vet 0 FAIL（零 Go 变更）；tsc/tsc -b/eslint 0；vitest **1157/1157**（+10）；
  drift PASS（552）；版本四处 4.30.0。详见 releases/v4.30.0.md。
# gaea · 多功能 AI 助手

## v4.29.0 · 办公 UI 化繁为简：顶栏收拢 / 自适应标签 / 预览降噪（2026-09-02）
> 用户点名主轴「UI 界面化繁为简，参考市场同类产品」，红线 **简化≠删除功能**。
> 弹药：模块制调研两线（AI 代理工作台 / 办公文档工具）→
> docs/market-research-2026-09-01b.md（原始稿 docs/research-2026-09-01b/）。
> **绑定面 552 零变更**（纯前端呈现重组）。
- **顶栏导出收拢**：新 ExportMenu——「导出 / Word / PDF」三个常驻文字钮收进
  单钮「导出 ⌄」下拉（对标 Devin/Linear「新动作只进菜单不加按钮」+ VS Code
  顶栏单点溢出）；md/Word/PDF 三出口与统一交付管线原样保留，仅呈现收敛。
  顶栏常驻操作钮 7→5。
- **右栏 tab 窄栏自适应图标化**：容器 <420px 时 6 tab 文字 CSS 隐藏、只显
  图标（aria-label/title/角标全保留，对标 Notion 视图 tab Icon only/Text
  only）；宽栏恢复文字。6 tab 集合与数量锁不动（WorkspaceTabs compact 受控
  覆盖可测）；340px 基线宽下 6 带字 tab 拥挤问题根治。
- **预览头部降噪**：FilePreview 头部「打开/定位」图标化（title/aria-label
  保留，测试按 title 锁）+ 全部头部按钮去边框（无边框+悬停浅底）；「编辑/
  保存/取消」等状态语义动作文字保留（编辑能力保留红线，测试钉住）。
- 验证：Go 110 包 0 FAIL；tsc / tsc -b / eslint 0；vitest **1147/1147**
  （+9：ExportMenu 5、WorkspaceTabs compact 3、FilePreview 图标化 1）；drift
  PASS（552 零变更）；版本四处 4.29.0。详见 releases/v4.29.0.md。

## v4.28.0 · 浏览器与版本：观察窗 / 版本时间线 / pptx 交互（2026-09-01）
> 规划「浏览器与版本」刀（A2+B1+B2/C3）。**绑定面 550 → 552（+2：
> GaeaPptxOutline / GaeaBrowserObserve）**。三并行子代理分线+主代理集成。
- **A2 浏览器观察窗**：右栏新「浏览器」tab——CDP 截图步进流（captureScreenshot
  jpeg ≤1280 缩放；未运行 Available=false 绝不拉起）+ URL/标题 + 操作时间线
  （browser_* 倒序上限 20，对标 Trace Viewer Actions）+ 权限静态行 + 自动弹出
  胶囊（gaea.browserAutoOpen，App 接线新 browser_* 工具自动切 tab，2.5s 可见
  门控轮询）。实时帧流/人工接管远期。
- **B1 文件版本时间线**：产物 vN 徽标可点 → 内联时间线（时间/工具/轮次/状态）
  + 基线预览（GaeaPreview abs）+ 恢复（RollbackRecord，恢复=新增证据卡不丢
  历史）——**零 Go 改动**，完全长在证据链上（对标 Notion 版本史/Artifacts
  rewind，预览即护栏）。
- **B2/C3 pptx 交互**：新绑定 GaeaPptxOutline（python-pptx 结构化大纲）+
  GaeaPreview .pptx 分支（soffice→PDF 缓存 7 天 TTL + poppler 逐页缩略 ≤60
  页）→ 前端逐页预览+大纲侧栏+页锚点滚动+「针对第 N 页修改」指令插入；
  python 缺失降级诚实。
- 验证：Go 110 包 0 FAIL（stress flaky 沿旧）；tsc -b/eslint 0；vitest
  **1138/1138**（+43）；drift PASS（552）；版本四处 4.28.0。详见
  releases/v4.28.0.md。

## v4.27.4 · todo 持久化改名：.gaea/progress.md 撞名根治（2026-09-01）
> **勘误**：此前三次把 `.gaea/progress.md` 被覆写归因于「并行会话」——错误。
> 真凶是 gaea 自己的 `todo_write` 工具：计划进度持久化写 `<工作区根>/.gaea/
> progress.md`，办公代理在以 wubigrok 仓库为工作区跑任务时，每次 todo_write
> 都覆盖同名发布进度文件（一天四次，内容即任务 todo 表）。
- **修复**：todo_write 持久化改名 **`.gaea/todos.md`**（todo.go）；compaction
  读取端 `readProgressFile` 优先 todos.md、**回退旧名 progress.md**（存量工作
  区兼容）。项目记忆文件 `.gaea/progress.md` 从此不再被运行时覆盖。
- 测试 +2：saveProgressMarkdown 写 todos.md 且不碰 progress.md；读取端优先/
  回退语义（walk-up 设计使「均缺失」断言在真实机器不成立，已注明）。
- Go 110 包 0 FAIL（TestCancelConcurrentStress 负载型 flaky 沿旧）；前端零
  改动；drift PASS（550）；版本四处 4.27.4。详见 releases/v4.27.4.md。

## v4.27.3 · markdown 包裹符：交付卡片路径修复（2026-09-01）
> 用户报告「交付卡片点击无法打开、定位打开的不是文件位置」→ 真实会话实锤：
> 模型用反引号包裹路径，匹配把开头反引号吞进路径 → 预览「文件不存在」、
> 定位错位。**绑定面 550 零变更**。
- **根因**：fileLinks 路径字符集不排除 markdown 包裹符 `` ` `` 与 `*`——两者
  恰是 Windows 文件名非法字符，应作路径边界（v4.26.1 全角括号盲区第二弹）。
- **修复**：PATH_BODY/FIRST_SEG 排除 `` ` `` 与 *，PATH_BOUNDARY 纳入为边界，
  BARE_FILE_RE 分隔符后允许包裹符前缀；下划线等合法字符不受影响；存量消息
  渲染时实时重提取，重启即恢复可点。
- 测试 +5（真实会话文件名四形态+下划线守卫）；tsc -b/eslint 0；
  vitest 1095/1095；版本四处 4.27.3。详见 releases/v4.27.3.md。

## v4.27.2 · 细节收口：subagent_message 端到端 / 轨迹子代理记录 / 目录定位（2026-09-01）
> 细节打磨刀。**绑定面 550 零变更**。
- **subagent_message 端到端收口**（v4.26 回投特性此前实际未通——后端发
  kind=subagent_message、前端无消费整条被丢）：wire 层转译 kind="message"+
  subagentRef（磁盘日志仍按原始 kind 落），前端既有 message subagentRef 语义
  接管，「子代理」徽标气泡真实生效；补拉折叠同步（GaeaResyncItem.subagentRef
  恒全键、fold subagent_message→独立条目+closePending 防误续写）。
- **轨迹面板子代理记录**：TrajectoryRecordKind 加 "subagent"，徽标/Bot 图标/
  折叠行（答复摘要+ref）/详情全文/搜索命中，turns 与 betweenTurns 双落点。
- **sidebar_open 目录定位**（收 v4.25 欠账）：directory → FileTree 树中定位；
  顺带修 FileTree 目录行无 data-path 锚点导致 reveal 静默失效的暗坑。
- 验证：Go 110 包 0 FAIL（TestCancelConcurrentStress 负载型 flaky 单跑稳定）；
  tsc -b/eslint 0；vitest 1090/1090（+5）；drift PASS（550）；版本四处 4.27.2。
  详见 releases/v4.27.2.md。

## v4.27.1 · seq 防线 omitempty 失配修复：对话窗运行中只显示读秒的根因收口（2026-09-01）
> 用户报告「运行中只有一个思考读秒，没有交替出现过程卡/文本卡（只有轨迹面板
> 有显示）」。**绑定面 550 零变更**。
- **根因**：v4.26 seq 补拉防线前后端形状契约失配——Go GaeaResyncItem 全字段
  omitempty（流式 assistant 空 reasoning、写类工具 readOnly:false 的键被序列化
  省略），前端 parseResyncItems 严格校验缺键即整快照判坏 → 补拉快照 100% 被拒、
  防线静默失效；Wails 吞件期间对话窗无物可渲染（WorkHeader 是 store tick 驱动
  所以活着，轨迹面板读盘不受害）。
- **修复**：①Go 全字段去 omitempty + TestGaeaResyncItemWireAllKeys 锁「序列化
  恒全键」契约；②前端缺省键宽容（缺键→零值，类型错/kind/id/status 校验不变）。
- **真机验证**：真实应用发只读任务——对话窗 WorkHeader「已完成 · 用时 15s ·
  7 步」+ 阶段行 + 思考块 + ls 工具卡 + 正文交替（elapsed 3s→8s→14s 运行中
  逐个渲染）；v4.26.1 交付卡片同屏确认。
- 验证：Go 110 包 0 FAIL；tsc -b/eslint 0；vitest 1085/1085（+5）；版本四处
  4.27.1。详见 releases/v4.27.1.md。

## v4.27.0 · 右侧面板对齐 Codex：子代理对话实时下钻 / 对话与上下文完善（2026-09-01）
> 延续 v4.26「对齐 Codex」：右栏文件工作台、标签扁平化、对话输出、子代理实时
> 对话、上下文标签五面打磨。**绑定面 550 零变更（纯前端）**。
- **右栏文件工作台**：点文件后预览占满右栏（原顶部 3/5 小窗 + 底部文件树）；
  文件树收敛为「文件」按钮切换的 260px 侧栏；宽度上限 720→1600（视口 − 侧栏 −
  400 对话区动态钳制），首次打开文件自动抬升 560；编辑器 tab 加文件类型图标
  （lib/fileIcon 单源）；树内高亮当前编辑文件。
- **标签扁平化**：删「资料/成本库」，取消二级标签 → 文件/产物/变更/任务/分工
  一级平铺；运行角标按任务/分工下发；旧存储值自动收敛。
- **对话输出**：用户消息去气泡；第 2 轮起「第 N 轮」分隔线；助手消息复制按钮；
  编辑类工具 +N−N diffstat 芯片。
- **子代理实时下钻**：点击子代理 → 全面板对话（SubagentThread），运行中 3s
  轮询 + 事件驱动实时刷新、自动跟随底部；消息流 Codex 式（思考折叠/tool 卡）。
- **上下文标签**：总览头部水位分色（≥70% 琥珀/≥90% 红）+ 缓存/费用/刷新；空态
  引导；文件活动行点击打开预览；步骤详情「占窗口 %」；趋势图悬停构成详情。
- 验证：vitest 1082/1082（169 文件）；tsc -b/eslint 0；Go 零改动；drift PASS
  （550）；版本四处 4.27.0。详见 releases/v4.27.0.md。

## v4.26.1 · 全角括号文件名：交付卡片失配修复（2026-09-01）
> 用户报告「看不见完工交付卡片、无法点击查看文件」→ 真实会话实证：正文有
> 「交付文件：C:\…\开工筹备计划（修订）.docx」但卡片未渲染。**绑定面 550 零变更**。
- **根因**：fileLinks 的 PATH_BODY 把全角括号（）当路径终止符——文件名含（）
  （中文办公常态：（修订）（终稿））时正则截断、扩展名拼不上，交付卡片与
  内联文件链接整体失配。
- **修复**：路径体允许全角括号；扩展名仍锚定匹配末尾（「报告.docx（三份）」
  不吞补语）。+5 匹配用例 + 组件级回归守卫（DeliverableCards.regress）。
- 验证：tsc -b/eslint 0；vitest 1080/1080；Go 无改动；版本四处 4.26.1。
- 详见 releases/v4.26.1.md。

## v4.26.0 · 对话流式重造：对齐 Codex（2026-09-01）
> 用户报告「发送后对话窗静默而轨迹在动」→ 根因六连（子代理文字有意不进主聊天/
> 预处理窗零事件/Wails 吞件/Retrying 隐身/phase 空 seam/TTFT 静默）逐一对账。
> **绑定面 549 → 550（+1：GaeaResyncEvents）**。
- **工作态头部行（WorkHeader）**：turn 激活期常驻（spinner+阶段文本+已用时+
  步数，items 为空也渲染——发送那一帧起窗口不空）；完成转 Codex 式「已完成 ·
  用时 · N 步」耗时行；StreamingIndicator 收敛为兜底。
- **后端 phase 事件接线**：预处理各阶段（启动引擎/解析 @引用/装配上下文/检索
  记忆/思考中）+ Retrying/compaction 转译 phase（磁盘日志格式不变，200ms 节流）；
  phase 收编过程卡+头部。
- **子代理活动回投主回合**（Codex 2026-08 同款）：新事件 subagent_message 回投
  子代理最终答复（完成态，中途不回投防刷屏），主区消息「子代理」徽标；task 卡
  running 实时 lastText/lastTool 预览 + 完成结果摘要。
- **事件序号防线**：gaea-event 全量带 seq（会话切换归零），跳号→GaeaResyncEvents
  从磁盘日志折叠全量快照整体替换（5s 冷却/在途去重/坏快照保底/streaming 续接；
  golden 逐字节不变）。
- **重复工具折叠**「已调用 X · N 次」（Claude Code 式）；顺带修复
  weixin_reminder_test 时间炸弹。
- 验证：Go 全量 0 FAIL（golden/fold 原样通过）；tsc -b/eslint 0；vitest
  1072/1072（净增 71）；drift PASS（550）；版本四处 4.26.0。三并行子代理分线
  +主代理集成；调研 docs/research-2026-09-01/codex-streaming-ux.md。欠账与
  v4.27 顺延详见 releases/v4.26.0.md。

## v4.25.0 · 文件工作台：编辑器 tab 化 / 变更 diff / 选区联动 / 模型主动打开（2026-09-01）
> 规划 docs/gaea-office-upgrade-plan-2026-09.md 第三刀：A3 文件工作台 +
> B3 选区联动。**绑定面 549 → 549（零新增：sidebar_open 走内置工具事件管线）**。
- **编辑器 tab 化（EditorTabs）**：文件树点开 → 右栏内多文件编辑器 tab
  （lib/editorTabs 外部 store：上限 12 LRU/关闭激活相邻/localStorage 持久化
  坏值兜底）；FilePreview 新增 embedded 模式，docx/xlsx/md/图片/PDF 能力原样
  随迁（换壳不换芯红线）；双入口保留（树行点击=右栏内开 tab，右键=主区预览
  pane）；产物行「树中定位」→ FileTree 展开父链+滚动+闪烁（reveal）。
- **变更 tab diff 化（Git 面板式）**：文件行可展开 → 行级红绿 diff（lib/planDiff
  三态：edit_file/multi_edit 真 before/after；write_file/edit_lines 写入内容
  预览+原因；其余诚实不伪造）+ 回滚接证据链 Journal 最近基线（无基线诚实标注）。
- **B3 选区联动**：xlsx 选中单元格→浮动「引用到对话」；docx 框选工具栏补
  「引用到对话」；docx 渲染失败降级纯文本视图（docxText 提取正文段落+提示条）。
- **模型主动打开（sidebar_open）**：新内置 Go 工具（work 空间/ReadOnly 直允许/
  防穿越/envelope data path_rel）+ 前端解析器 + App 按事件 id 去重接线——模型
  把关键产物推到右栏编辑器 tab，file 开 tab/directory 亮文件 tab。
- 验证：Go build/vet/test 0 FAIL（+20 用例）；tsc -b/eslint 0；vitest 1001/1001
  （净增 74）；drift PASS（549）；版本四处 4.25.0。三并行子代理分线+主代理
  集成。欠账与下一刀 v4.26 详见 releases/v4.25.0.md。

## v4.24.0 · 子代理工作台：树拓扑 / 实时动态 / 产物登记表（2026-09-01）
> 规划 docs/gaea-office-upgrade-plan-2026-09.md 第二刀：A1 分工/子代理拓扑
> tab（AgentNetworkCard+SubagentsPanel 合体进化）+ C1 后端权威产物登记表。
> **绑定面 548 → 549（+1：GaeaDeliverableRegistry）**。
- **树形实时拓扑（AgentTree）**：嵌套 Children 全量渲染（此前只画两层）；
  root 折叠为「主 agent」行、更深层默认收起可展开；**新节点自动展开父链**；
  节点量化（状态色点/任务摘要/工具数/模型徽标/耗时——running 实时已用 1s
  tick/错误数）；下钻链：节点 → 详情卡 → 完整 transcript → **工具调用行点击
  定位结果消息**（收 v4.21 欠账）。
- **合并活动流（Devin 式单列 feed）**：running 子代理 lastText/lastTool 按
  updatedAt 倒序合并、上限 20、空态收起；树内行预览并存。
- **新子代理自动展开（可关，默认开）**：新 ref 出现 → App 亮出右栏切「分工」
  tab（停用时尊重停用态）；偏好键 gaea.subagentAutoOpen，损坏值回落默认。
- **C1 权威产物登记表**：trajectory.FoldDeliverables 从事件日志折叠写类 8 +
  生成导出类 3 工具的落盘登记（路径/工具/轮次/时间/次数，上限 200，Total 去重
  全量）+ 新绑定 GaeaDeliverableRegistry；DeliverablesPanel「权威产物登记」
  只读区（tool 徽标+路径+轮次+次数+时间，点击预览），补启发式漏登。
- 验证：tsc/eslint 0；vitest 927/927（净增 16）；Go build/vet/test 0 FAIL；
  drift PASS（549）；版本六处 4.24.0。详见 releases/v4.24.0.md。

## v4.23.0 · 工作台框架：右栏对标 DSH-better-sidebar 工作台化第一刀（2026-08-31）
> 用户拍板：右面板重造为 DSH-better-sidebar/Codex 式「运行工作台」（子代理/
> 浏览器/文件编辑器等实时操作面），状态显示类迁主区轨迹/上下文旁边。规划稿
> docs/gaea-office-upgrade-plan-2026-09.md（v2）。**绑定面 548 → 548（零新增）**。
- **Tab 注册表（lib/sidebarRegistry.ts）**：元数据复用清单 + render 接线单一
  数据源，右栏渲染与命令面板全派生；新增面板 = 清单 + RENDERERS 各一条，
  面板组件本体零改动（框架/内容解耦，为浏览器观察窗/编辑器 tab 留挂载点）。
- **工作台外壳三件套（学 better-sidebar）**：全局宽度键（左缘拖拽 280–720、
  最后一次拖拽胜出跨会话跟随）；声明式设置（齿轮→侧边卡片，每 tab 独立开关，
  停用即隐藏、至少保留一个、停用不进命令面板，启用集全局键）；会话记录 v2
  （JSON {v,tab,enabled,width}，v1 裸 id 兼容可读，坏值逐项兜底、失效指针修正）。
- **主区「概览」tab（统计迁移）**：ChatTabs 第 4 tab + OverviewPanel 承载原
  StatsPanel；右栏统计下线（union 全量移除，v4.22 旧 tab:"stats" 宽容收敛回
  「文件」并钉回归用例），右栏收敛 3 主 Tab×7 面板；命令面板新增概览入口。
- 验证：tsc/eslint 0；vitest 911/911（净增 38）；Go build/vet/test 0 FAIL；
  drift PASS（548）；版本四处 4.23.0。两并行子代理分线实现 + 主代理集成。
- 欠账：Tab 拆分/底部面板/自由窗口、设置二级弹窗、注册表懒加载 chunk 化；
  下一刀 v4.24「子代理工作台」。详见 releases/v4.23.0.md。

## v4.22.0 · 一次性收官：真虚拟化 / transcript 定位 / 晨报预载 UI（2026-08-31）
> 用户要求「一次性做完」：办公板块剩余本地可做欠账一次清完并整理提交收尾。
> **绑定面 546 → 548（+2：GaeaMorningPreload / GaeaSetMorningPreload）**。
- **轨迹真虚拟化（react-window v2 动态行高）**：扁平行流按视口窗口渲染
  （±overscan 12），超长会话 DOM 恒定，v4.21「首批+加载更多」分批机制退役；
  useDynamicRowHeight + ResizeObserver 实测展开行高自动重排；概览跳转走
  listRef.scrollToRow、搜索回顶、收起/展开照常；test/setup 补 ResizeObserver
  stub。
- **transcript 消息定位**：消息序号 #N + 搜索命中自动滚动到第一条命中。
- **晨报预载 UI 开关（+2 绑定）**：GaeaMorningPreload/GaeaSetMorningPreload
  （internal/config.Save 持久化 + 内存更新 + 重建引擎即时生效）；记忆面板
  「晨报预载 开/关」胶囊按钮（同款记忆开关交互）。
- 验证：Go 全量 0 FAIL（绑定面 548 PASS）；tsc/eslint 0；vitest 873/873
  （+3）；drift PASS（548）；版本四处统一 4.22.0。收尾：六轮改动作为一次
  合并发布提交 + tag v4.22.0；剩余欠账仅外部资源/官方数据项（Realtime 真机、
  自动路由、浏览器下载上传/headless UI/Windows UIA、iLink 真机窗口）。详见
  releases/v4.22.0.md。

## v4.21.0 · 长会话与 transcript：增量渲染 / 消息搜索（2026-08-31）
> 续 v4.20.0 剩余两条欠账：轨迹超长会话渲染量 + transcript 只读无搜索。
> **零新增绑定（546 不变，纯前端）**。
- **轨迹增量渲染（DOM 有界）**：轨迹改扁平行流（轮次头+展开记录+Between
  turns）按批渲染——首批 250 行，滚动到底自动续载或「加载更多（剩余 N 条）」，
  搜索词变化回首批；概览跳转同步扩可见区（不会再「跳过去了但没渲染」）；
  收起全部/展开全部在平行流上照常生效。
- **子代理 transcript 消息搜索**：查看器头部搜索框按正文/推理/工具名/参数/
  结果过滤，显示「命中/总数」，无匹配空态。
- **注释清理**：ChatTabs「轨迹暂占位」更新为 v4.17-v4.21 实际能力。
- 验证：tsc/eslint 0；vitest 872/872（+2）；Go 全量 cached 绿、drift PASS
  （546）；版本四处统一 4.21.0。欠账：增量渲染为分批 DOM 而非 react-window
  真虚拟化（渲染量有界但已渲染部分仍为真实 DOM）；transcript 只读无跳转/
  引用定位。详见 releases/v4.21.0.md。

## v4.20.0 · 剩余收官：子代理 transcript / 轨迹概览 / 旧会话趋势补齐（2026-08-31）
> 清掉 v4.17-v4.19 三刀之后的剩余欠账。**绑定面 545 → 546（+1：
> GaeaSubagentTranscript）**。
- **子代理完整 transcript 查看器（+1 绑定）**：新绑定 GaeaSubagentTranscript
  （sessionPath, ref）读取 `<sessionDir>/subagents/<ref>.jsonl` 全量消息
  （role/content/reasoning/toolCalls），ref 安全字符校验防穿越；前端 Agent
  网络详情面板增「查看完整 transcript」→ 消息流（角色徽标+推理+工具调用+
  结果，可收起）。
- **轨迹 Overview 投影 + 轮次跳转 + 折叠控制**：轨迹标签顶部概览条（每轮一
  根柱，柱高∝记录密度，工具调用高亮/报错标红，hover 明细，点击平滑跳转并
  展开）+「收起全部/展开全部」（长会话折叠成轮次索引；新回合默认展开）。
- **迁移/兜底会话趋势补齐（诚实估算）**：ToLogEntries 每回合合成
  request_header（system=真实 system 消息拼接，tools=该轮实际工具名集合，
  顺序与运行期一致）——旧会话从此有系统/工具分类与 header 轨迹记录；
  contextview 回合末估算关闭（turn_done 未见 usage 时用当前估算构成落
  estimated 记录），前端步骤详情显示「估算构成（无用量记录）」，不伪造用量。
- 验证：Go 全量 0 FAIL（+2 用例）；tsc/eslint 0；vitest 870/870（+3）；drift
  PASS（546）；版本四处统一 4.20.0。欠账：轨迹虚拟滚动未做（以收起全部+概览
  跳转缓解）；子代理 transcript 只读无搜索。详见 releases/v4.20.0.md。

## v4.19.0 · 看板收官：上下文浏览器 / /context 命令 / 子代理节点详情（2026-08-31）
> 续 v4.17.0+v4.18.0 的第三刀「继续完善」：上下文标签最后一个页脚占位收掉。
> **零新增绑定（545 不变）**。
- **上下文浏览器（surface 节点 + 归档）**：后端 contextview 折叠补全系统/
  工具节点（request_header 的 system prompt 与工具集合只在构成变化时入
  nodes，初版+变化版，每步重复不刷屏；文本=预览，全文在日志）；前端
  ContextBrowserCard——活跃/归档双页签（归档=被压缩移出节点，带「已压缩」
  标记）+ 六分类过滤 + 节点行（分类色点+≈tokens+文本预览，超长可展开）；
  页脚占位整行移除。
- **`/context` 命令**：GaeaCommands 内置 + i18n（zh/en），斜杠菜单可发现；
  classifyComposerCommand 增 context 分类，App.handleSend 拦截 → 切上下文
  标签（不发给模型）；CLI 未拦截路径走未知斜杠 Notice。
- **Agent 网络节点点击 → 子代理详情**：AgentNetworkCard 增 sessionPath
  （App 注入 currentSessionPath），点击子代理节点 → SubagentRuns 按任务前缀
  匹配（与后端 enrichAgentNetwork 同口径）→ 固定详情面板（状态/模型/工具
  调用数/更新时间 + lastText/lastTool + 最后回答摘要）；无匹配回退节点统计。
- 验证：Go 全量 0 FAIL（+1 用例）；tsc -b 0；eslint 0；vitest 867/867（+4）；
  drift PASS（545）；版本四处统一 4.19.0。欠账：子代理完整 transcript 查看
  器；轨迹 Overview 投影与虚拟滚动；迁移会话系统/工具分类与趋势柱（诚实不
  造数）。详见 releases/v4.19.0.md。

## v4.18.0 · 看板补全：文件活动 / 增量模式 / 实时刷新（2026-08-31）
> 续 v4.17.0（事件日志默认开启，数据源接通）之后的「继续完善」：收掉两个
> 看板剩余的占位尾巴。**零新增绑定（545 不变）**。
- **文件活动时间线（上下文标签新卡）**：后端 contextview 折叠新增 FileActivity
  ——工具参数确定性提取路径（path/rel/source/destination/image_path/output 键）
  + 工具→动作白名单（read_file/grep/vision/format_convert=读；write_file/
  edit_file/multi_edit/edit_lines/chart_gen/diagram_gen/screen_capture=写；
  move_file=移；ls=目录），screen_capture 从结果输出补记，bash 等无法确定性
  取路径者诚实不造数；同轮同步骤同路径合并、上限 200、空切片非 nil。前端
  文件活动卡（动作徽标+工具+路径+时间，倒序最近 40 条），页脚改为「上下文
  浏览器将在后续阶段接入」。
- **增量（Delta）模式启用**：趋势图「增量」按钮去掉灰置与 Phase B 标题，
  切换后展示每步相对上一步的净变化（绿=净增·红=净减，随模式显示图例），
  柱色改全站一致的可视化语义色。
- **运行中实时刷新**：新 hook useLiveReload 订阅 gaea 事件流——运行中节流
  刷新（1200ms）+ turn_done 立即刷新 + 整轮完成刷新；轨迹/上下文/Agent
  网络三处统一接入（替换「仅回合结束刷新」effect）。
- 验证：Go 全量 0 FAIL（+2 用例）；tsc -b 0；eslint 0；看板+mock-contract
  vitest 22/22（+2）；drift PASS（545）；版本四处统一 4.18.0。欠账：上下文
  浏览器（surface 节点浏览/归档）仍占位；Agent 节点点击跳子代理会话；轨迹
  Overview 投影与虚拟滚动；/context 命令；迁移会话系统/工具分类与趋势柱
  （旧消息无 request_header/usage，诚实不造数）。详见 releases/v4.18.0.md。

## v4.17.0 · 轨迹上下文接通：事件日志默认开启（2026-08-31）
> 用户反馈办公板块「轨迹」「上下文」标签是空壳——根因不是 UI，而是数据源
> （事件日志）缺省关闭：`session.log_format` 缺省 legacy → sink 不接线 →
> 看板恒读空日志。本刀把事件日志改为**缺省开启**并补旧会话读端兜底。
> **零新增绑定（545 不变）**。
- **事件日志默认开启（数据源接通）**：`config.EffectiveLogFormat()` 缺省
  "event"，仅显式 `log_format = "legacy"` 退回旧行为；`gaea_handler.go`
  注入生效值、boot 同源创建 EventLogSink——轨迹/上下文/Agent 网络三看板
  从下一轮对话起即有真实数据。
- **旧会话读端兜底**：`session.ReadEntriesFor` 优先事件日志、缺失时从旧
  `<id>.jsonl` 投影折叠条目（纯读不落盘）；`GaeaTrajectory`/`GaeaContextView`/
  `GaeaAgentNetwork` 统一改用，存量 legacy 会话也能看板。
- **迁移产物带回合边界**：`ToLogEntries` 每条 user 消息前写 turn_started、
  流尾写 turn_done（ProjectMessages 忽略边界，恢复投影逐字节不变）；轨迹
  折叠兼容 `assistant_message`（内嵌工具调用展开为 tool 记录并与结果合并）。
- **资源释放**：boot 把 EventLogSink.Close 挂进 Controller.Cleanup——缺省
  event 后 Windows 上会话目录可删除/迁移（文件句柄泄漏面一并修掉）。
- 验证：Go 全量 0 FAIL（+6 用例）；tsc -b 0；看板组件 vitest 12/12；drift
  PASS（545）；版本四处统一 4.17.0。欠账：迁移/兜底会话无 request_header/
  usage（系统/工具分类 0、趋势无柱，新会话完整）；Agent 网络对 legacy 会话
  仅 root；上下文增量模式/浏览器/File activity/SSE 增量刷新仍为既定欠账。
  详见 releases/v4.17.0.md。

## v4.16.0 · 四刀并行：离线收口 / 浏览器键盘与 iframe / 复核可视化 / 晨报预装配（2026-08-31）
> 用户拍板「全部并行处理」：v4.15.0 欠账清单四个可离线方向由四个并行子代理
> 同步落地（足迹隔离，主控全绿门禁）。**零新增绑定（545 不变）**。Realtime
> 真机验证排除（需用户真 key+麦克风）。
- **①persona 侧离线裂缝收口（真 bug）**：gaea_whisper_causal/gaea_whisper_retell/
  whisper_handler 三处 `featureModel("chat")` → `routeModel("chat")`——全局离线
  过滤对 persona（轻语）链路生效（此前因果解释/记忆重述/WhisperChat 绑云端照样
  发云端），用户功能绑定语义不变（同源）；+2 离线回归测试。
- **②浏览器键盘级 Input + iframe（v4.13/14 欠账，零绑定零前端）**：新工具
  `browser_press`（第 11 工具）——Input.dispatchKeyEvent 键盘级输入（key 别名表
  + ctrl/alt/shift/meta 组合 + 可选 text 真实输入，Enter 补 `\r` 触发 keypress 真机
  踩坑修复）；browser_read/click/type 加可选 `frame` 参数——getFrameTree→
  createIsolatedWorld→contextId 执行（**iframe 内交互完整实现**，真 headless Edge
  真机验证 Read/Click/Type 全通）；snapshot 不下钻 iframe 诚实拒。
- **③Verifier 通道 B 结果进前端（v4.14 欠账）**：Verdict 增 channelBRatio/
  channelBPages/channelBArtifacts（omitempty 旧卡兼容）；证据卡 verdict 内联区
  追加「视觉复核：像素差异率 x.x% · N 页」+「查看复核产物」按钮（打开产物目录
  before/after PDF + 逐页 PNG）；无通道 B 旧 verdict 不渲染。
- **④晨报深度预装配（v4.14 欠账，零绑定零前端）**：memory.BuildMorningPreloadBlock
  纯函数（复用 BuildMorningBrief 排序口径，≤600 rune 确定性零 LLM）→ sysprompt
  装配点注入「【工作记忆晨报】」块（门控 Memory.Enabled && morning_preload &&
  space==work，play/mode=off 不注入=双空间红线）；config 键 morning_preload
  （默认 true，仅配置文件可控）。
- 验证：Go 全量 0 FAIL（+20）；**vitest 861/861**（+2）；tsc/eslint 0/0；drift
  PASS（545）；build.bat 冒烟 200。欠账：Realtime 真机（需用户资源）；自动路由
  本体（待官方逐模型数字）；浏览器 snapshot 不下钻 iframe/下载上传/headless UI/
  Windows UIA；通道 B 逐页缩略图；晨报预载无 UI 开关。详见 releases/v4.16.0.md。

## v4.15.0 · 聊天路由归位：plain 聊天离线过滤修复 + 「由谁回答」回显（2026-08-31）
> v4.14.0 欠账「自动路由 v1」经用户拍板收缩为最小价值刀——砍成本档位机制/
> 开关/UI（缓存价/峰谷价无官方逐模型数字，诚实不入表），只留两块真实价值：
> ①plain 聊天离线过滤裂缝修复（真 bug）②消息级「由谁回答/为何/花了多少」回显。
> **零新增绑定（545 不变）**。
- **聊天路由归位（bug 修复）**：chat_service.go:68/:105 + chat_handler.go:9 三处
  `featureModel("chat")` → `routeModel("chat")`——用户功能绑定语义逐字节不变
  （routeModel 步骤 1 与 featureModel 同源）；新增收益=全局离线模式对 plain 聊天
  生效（修复「总闸不总」裂缝，此前绑云端照样发云端、persona 却被滤）+ 无绑定时
  全局活跃/兜底与 persona 一致 + `model.route` 事件补齐（模型中心「当前生效」可
  展示 chat）。featureModel 保留（展示用），routeModel 零改动。
- **「由谁回答/为何/花了多少」回显**：modelengine 导出 `EstimateCostCNY`（本地/
  未知恒 0、USD 按汇率折算 CNY、非法汇率回退 7.2）；chat done 帧/ChatSend 返回
  加 `answered_by{engine,model,source,cost_cny}`（流式按 chunk.Usage 实算，usage
  不可达诚实记 0）；前端 `AnsweredByLine` 消息底部小字「由 X 回答 · 标签[ · 约
  ¥x.xx]」（费用 ≤0 隐藏费用段，不虚报）+ `useChatStream` 解析（旧事件静默跳过，
  向后兼容）。
- 验证：Go 全量 0 FAIL（+7：EstimateCostCNY 表驱动 5 + plain 聊天离线回归 +
  done 帧 SSE 断言）；**vitest 859/859**（+7）；tsc/eslint 0/0；drift PASS（545）；
  build.bat 冒烟 200。欠账：自动路由本体未做（待官方逐模型缓存/峰谷数字）；
  persona 侧 gaea_whisper_causal/retell 同类离线裂缝=观察项；plain 费用口径
  usage 不可达恒 0（诚实降级）。详见 releases/v4.15.0.md。

## v4.14.0 · 三箭并行：晨报预取 + 浏览器续刀 + 复核产品化（2026-08-31）
> 用户拍板「多刀并行」：三个互不相交的小刀由三个并行子代理同步落地（文件
> 足迹隔离），主控全绿门禁收口——①浏览器欠账（空闲 TTL 自动关停 + 多标签页）
> ②路线图 T0 欠账「做梦 2.0 主动预取 MVP」（纯本地晨报）③市场调研 ★★☆
> 「Verifier 产品化」（证据链翻成 UI，可审计护城河先发占位）。绑定面
> 544→545（+1：GaeaMemoryMorningBrief）。
- **浏览器续刀（internal/gaea/browser，零新增绑定、前端零改动）**：空闲 TTL
  自动关停（Options.IdleTTL 默认 10min、GAEA_BROWSER_IDLE_TTL env 覆盖；Ensure
  成功路径刷新 lastActive，once 守护 watcher 到期调 teardownLocked 自动回收，
  Shutdown 幂等停 watcher，到期后 browser_* 自动重拉闭环）；多标签页（Manager
  重构 conn+pageID → tabs map + activePageID，/json/list 全量 target 为真源；
  ListTabs/NewTab/SwitchTab/CloseTab，切换/新建置 epoch=0 旧 refs 诚实失效，
  关 active 自动切剩余、最后一个整体回收）；新工具 ×3（browser_tabs 只读 /
  browser_new_tab / browser_switch_tab）+ browser_close 可选 tab_id（缺省保持
  现语义逐字节不变）；compact 两表同步。+8 测试（TTL 回收/零禁用/env/列表/
  新标签/ref 失效/关标签/整体回收）。
- **做梦 2.0 主动预取 MVP（纯本地晨报）**：memory.BuildMorningBrief 纯函数
  （零 LLM/零 IO/确定性：max(UpdatedAt,LastUsedAt) 降序 top5 user/project 优先 +
  procedural/rule ≤3 条 + rune 边界截断 120 + 空输入非 nil 空数组）；新绑定
  GaeaMemoryMorningBrief() (string, error)（JSON 串对齐 GaeaCostGraph 先例：
  ListInSpace("work") 只读 + 近 24h dream 审计计数，零写库零落审计，play 红线
  安全）；前端 MorningBriefCard（首页 ml-info 记忆脉搏旁，仅 work 空间渲染，
  失败/空静默隐藏，全 token 样式）+ i18n home.morningBrief.* 三语；gen_bindings
  重生成（bindingNames 545）、spaceBindings 分类 work。Go +12、vitest +4。
- **Verifier 产品化（纯前端、零新增绑定、后端零改动）**：证据卡「三步展开」——
  卡面（无 baselinePath → 回滚禁用 + 「可复核明细」徽标 + 整卡点击展开）→
  第 1 层声明↔实况 diff（opsJson 单格 op × GaeaPreview 现取实况，口径同后端
  数值容差 1e-9/去空白/公式归一，✓/✗/跳过标注 + 近似比对脚注，预览不可用降级
  仅声明回放）→ 第 2 层操作回放时间线（序号 + type 徽标 + applyOne 风格中文
  描述 + 批量 op 折叠计数，旧卡无 opsJson 回退 beforeSummary）；lib/verifyDiff.ts
  纯函数层 + types.ts 补 baselinePath/opsJson + XlsxOpView/VerifyDiffRow；
  mock/office.ts 补证据域三绑定（此前零 mock）。vitest +23（verifyDiff 16 +
  证据区 7）。
- 验证：Go 全量 0 FAIL（+25 测试）；**vitest 852/852**（150 文件）；tsc/eslint 0；
  drift PASS（545）；版本四处统一 4.14.0；build.bat 冒烟 /api/health 200。
  欠账：晨报深度预装配（进 agent 上下文）列第二刀；浏览器 iframe/键盘级 Input/
  下载上传/headless UI/Windows UIA；Verifier 通道 B 结果未进前端、复核明细绑定
  留待真实需求；本地-云端自动路由 v1 顺延下一刀。详见 releases/v4.14.0.md。

## v4.13.0 · 自动操作·浏览器：CDP 控制 Edge + 7 工具面（2026-08-31）
> 「自动操作」四柱唯一空柱的第一块砖（调研后刀序④）：gaea 获得结构化浏览器
> 自动化——CDP（Chrome DevTools Protocol）控制 Edge，权限门 + 事件留痕第一
> 天挂上。零新增绑定（544 不变），前端零改动（工具经 Registry 自动进能力
> 面板与过程卡轨迹）。
- **internal/gaea/browser 包（新）**：msedge 三段式定位（GAEA_BROWSER_EXE
  env → Program Files 候选 → LookPath）；独立临时 profile 启动（绝不碰用户
  主 profile，Job Object 绑定 gaea 进程，父死子收）；页面级 CDP WebSocket
  会话（复用 gorilla/websocket：写串行 + 超时 + 幂等关，仿 realtime 范式）；
  Ensure 幂等 + 失联自愈重拉；URL 白名单只放行 http/https。默认有头（看得
  见=可信任），测试可 headless。
- **7 个 browser_* 内置工具（work 空间）**：browser_navigate / browser_read
  / browser_snapshot / browser_click / browser_type / browser_scroll /
  browser_close。snapshot 用 ref 机制（data-gaea-ref + 代数守门，页面跳转即
  失效诚实报 stale_refs），用法=「先 snapshot 拿 ref 再 click/type」；type
  用 React 兼容原生 setter + input/change 事件派发；结构化 envelope 返回
  （ok/timeout/not_found/stale_refs/validation_error 程序化码）。
- **权限门**：browser_read/snapshot 只读档（ReadOnly 恒放行）；其余五工具写
  档（交互 ask 档弹卡一次、可记忆规则）；permission subjectKeys 追加
  "url"——授权可固化为 browser_navigate(<url-glob>) 窄规则。不进 hardAsk
  （MVP 弹卡成本控制）；play 空间物理过滤天然不含。
- **留痕零改动全通用**：工具调用经既有 ToolDispatch/ToolResult 事件进会话
  JSONL（逐行带 space）→ trajectory 折叠 → 前端过程卡/Agent 网络自动展示
  ——全链对工具名零特判。
- 验证：**真机实测 PASS**（GAEA_LIVE_BROWSER_TEST=1：真 headless Edge 导航
  httptest 页 → 读文本 → snapshot 拿 ref → ref+selector 双路点击 → type
  联动回显 → file: 拒绝 → 跳转后旧 ref 失效，1.16s）；Go 全量 0 FAIL（+25
  测试：browser 包 19 + builtin meta 3 + permission url 3）；前端零改动、
  tsc/eslint 0、vitest 825/825（全量首跑 2 例负载型 flaky、复跑 0 failed，
  既有先例）；drift PASS（544）；build.bat 冒烟 200。

## v4.12.0 · 成本透亮：GLM 计价真实性 + 编码套餐积分口径 + 目录数据驱动（2026-08-31）
> 模块制市场调研（docs/market-research-2026-08-31.md）指出的「计费快变」风险
> 落地第一刀，兼收审计 T0 缺口②③：GLM Coding Plan 已改积分制（旧模型名
> 自动切换），静态目录与 token 计价都会失真——本刀把「花多少钱」做成真的。
> **零新增绑定（544 不变）**。
- **GLM 价格表补全（modelengine/stats.go）**：原表仅 glm-4.7 一条，GLM 用量
  实际未被计价；现按官方定价页（docs.z.ai，2026-08-31 核实，USD/百万 token，
  折 CNY 走既有 usd_cny_rate）补 glm-5.3/5.2/5.1/5/4.7/4.6/4.5-air/flash 系
  与 glm-4.6v，免费档（4.7-flash/4.5-flash/4.6v-flash）计 0；无法核实的诚实
  不入表（glm-5-turbo 显式置空挡板防 glm-5 前缀误匹配；cogview 系按张计费
  非 token 口径）。
- **编码套餐积分口径（billing_mode="coding_points"）**：glm 引擎走
  /api/coding/ 端点的调用不再按 token 估算费用（EstimatedCost=0、不进
  TotalCost），聚合以 glm@coding 单列（Tokens 计入、费用 0）——套餐内计
  「积分」不按 token 扣费；同桶混合窗口以最近一次调用口径为准（注释说明）。
- **模型别名注记（仅 coding 家族）**：官方 coding-plan 概览核实 4 条自动
  切换（glm-5.2/5.1→glm-5.3、glm-5-turbo/4.7→glm-5.3-flash）；ModelInfo 加
  alias_of 下发，前端模型卡「自动切换」标记 + title 说明；std 端点不注记
  （旧名独立计价）；记账归一让 glm-5.2 的用量落 glm-5.3 价格桶。
- **GLM 目录数据驱动（内嵌 JSON + 覆盖文件热更新）**：glmStaticModels 22 模
  型迁入 glm_catalog.json（//go:embed，新测试逐字锁定一个不增不减）；覆盖
  文件经 config 新键 `glm_catalog_path`（照 usd_cny_rate 先例启动注入，非密
  钥项）注入，mtime 变更自动重读（同 ID 替换 + 新 ID 追加），坏 JSON 静默
  回退内嵌——智谱目录快变时（调研：Kimi V1 已全平台下线）无需重编译。
- **前端**：engines.ts 类型同步三字段（alias_of/billing_mode/engines）；
  EngineSection GLM 卡 coding 家族追加「积分制计费，费用估算不含该端点用
  量」说明；StatsSection 明细行积分口径标签（费用列显示「积分内」非 ¥0）+
  按引擎小计区（glm@coding 显示「glm（编码套餐）」，旧数据无 engines 字段
  整块不渲染，向后兼容）。
- 验证：go build/vet/test 全量绿（新增 5 组 Go 测试：目录逐字锁定 / 覆盖
  mtime 热重载 + 坏 JSON 回退 / 别名注记 coding 有 std 无 / coding_points
  计费门控 + glm-5.2 归一落 glm-5.3 桶 / 价格表断言）、tsc 0 / eslint 0、
  vitest 825/825（+4）、drift PASS（544）、build.bat 冒烟 200。

## v4.11.0 · GLM 全模态纵深：生图后端 + 官方双端点（2026-08-30）
> 续 GLM 引擎主线：聊天已真机打通，本刀把 GLM 从「只能对话」补成全模态
> （生图）+ 支持编码套餐（官方双端点），并修复一处模型分类误判。
- **生图后端 `ai.GLMImageBackend`（kind=glm）**：按官方「图像生成」API 实现
  ——`POST /api/paas/v4/images/generations`、Bearer 认证。关键差异：官方
  schema 只收 model/prompt/size（**无 response_format**，仅回 URL；negative/
  seed 等 OpenAI 扩展字段也不收），故不复用通用 OpenAI 后端、只发官方字段；
  响应 URL 统一下载转 data URL（复用前端显示/落盘链路）；官方错误体
  `{"error":{code,message}}` 原样透出；200 无图提示「可能触发内容审核」；
  img2img 诚实拒绝（官方端点无图生图参数）。缺省模型 cogview-4-250304。
- **App 三处接线**：initImageBackend / SetImageBackend / 角色剧照
  buildPortraitClient 均支持 glm；GLM Key 经 `Manager.GLMKey()` 与 chat 同源
  取用（不读 EngineConfig.APIKey）；size 参数保留（官方接受）。
- **官方双端点切换 `SetGlmEndpoint`（绑定面 543→544）**：std=`/api/paas/v4`
  （按量付费）/ coding=`/api/coding/paas/v4`（编码套餐额度，官方
  coding-plan/quick-start 核实——填错端点会 404 或误扣费）。后端只收两个
  官方常量（GLMBaseURLStd/GLMBaseURLCoding），不透传自由地址；GLM 引擎卡
  Segmented 切换并落盘持久化；LoadState 脏地址防线兼容。
- **生图模型目录补全 + 误分类修复**：静态目录补 glm-image/cogview-4-250304/
  cogview-4/cogview-3-flash（锚定官方图像生成 API 枚举，18→22）；修
  glm-5-turbo 被通用 turbo 关键词误判为生图（GLM 引擎先按官方目录判型再落
  通用关键词表，回归测试锁死）。
- **前端**：模型中心「图片生成」加 GLM 云端选项；设置页绘梦引擎标签诚实化
  （此前云端引擎也标「本地引擎」）；classifyModel 补 cogview；新增
  glmEndpointFamily 端点家族判定。
- 验证：Go 全量绿（+15 测试）、vitest **821/821**、tsc/eslint 0、
  drift PASS（544）。
## v4.10.0 · 修复 GLM Key 保存被拒（2026-08-30）
> 真机实测：GLM 卡片保存 Key 报「不支持的配置项: glm_api_key」——config.go
> 加了 Key 常量与字段，但漏登记 Save 白名单 saveSetters，保存被拒。
- **修复**：saveSetters 登记 glm_api_key → configFile.GLMAPIKey。
- **防回归**：TestSaveSetters_CoverAllAPIKeyFields 用反射断言 configFile 所有
  `*_api_key` 字段都有 Save setter——以后新增任何密钥类配置漏登记即测试失败。
- 验证：Go 全量绿（+1）、vitest 818/818（ProgrammingPage 1 例既有负载 flaky
  单跑复绿）、绑定面 543 不变。

## v4.10.0 · GLM 按官方文档重写：无 /models 端点（2026-08-30）
> 用户实测两轮仍不可用后核对 docs.bigmodel.cn 官方文档，发现根本性误判：
> **智谱官方没有模型列表端点**（文档仅有 chat/completions 等），此前用
> GET /models 做测试连接/刷新模型永远失败（真机 401「内部错误」）。
- **静态模型目录**：glmStaticModels 锚定官方「模型概览」（2026-08-30）——
  glm-5.3/5.2/5.1/5/5-turbo、glm-4.7 系（4.7-flash 免费）、glm-4.6/
  4.5-air/4-long、多模态 5.3-flash/4.6v、glm-tts/glm-asr-2512/embedding-3/
  rerank；Kind 经 ClassifyModelKind 统一分类。
- **Key 校验改走 chat ping**：TestConnection 对 GLM 发最小 chat 请求
  （max_tokens=1）真实验证 Bearer Key；错误体按官方形态
  {"error":{code,message}} 原样透出（如「令牌无效」），不再出现凭空 401。
- **默认模型 glm-4.6 → glm-5.3**（官方文档全部示例所用旗舰）。
- 验证：Go 全量绿（+2 测试：静态目录分类断言 / httptest ping 401「令牌
  无效」与 200 两态 + Bearer 头断言）；vitest 818/818；tsc/eslint 0；
  绑定面 543 不变。

## v4.10.0 · GLM 引擎地址防呆修复 (2026-08-30)
> 真机实测：GLM 卡片对云端引擎露出了地址编辑框，用户把 API Key 粘进地址框
> 保存——base_url 变成 Key 本体，此后所有请求报 `unsupported protocol
> scheme ""`，且 Go 原生错误把 Key 原文回显到界面（二次泄漏）。
- **UI**：地址编辑框改为仅本地引擎（ollama/herdsman/cosyvoice）显示——云端
  地址是预置常量，本就不该可编辑（此前靠黑名单排除，新增引擎易漏）。
- **后端三道防线**：SaveEngine 拒绝无 http(s) 前缀的地址（错误信息不回显
  原值，防 Key 二次泄漏）；LoadState 忽略存量脏地址（保留预置——已中招的
  engines.json 重启应用即自愈）；fetchModels 对无效地址给不回显原值的友好
  错误。
- **用户侧善后**：重启应用即恢复 GLM 预置地址；因 Key 已出现在报错浮层与
  engines.json 明文里，建议到 open.bigmodel.cn 重新生成密钥后再填入「保存
  Key」框。
- 验证：Go 全量绿（+1 回归测试：保存拒绝/载入自愈/友好错误三断言，且断言
  错误信息不回显原值）；vitest 818/818；tsc/eslint 0；绑定面 543 不变。

## v4.10.0 · 模型中心新增 GLM 引擎 (2026-08-30)
> 智谱 GLM 云端引擎（OpenAI 兼容 `https://open.bigmodel.cn/api/paas/v4`），
> 照 DeepSeek/OpenCode 模式全链路接入。端点真实性已验证（/models 与
> /chat/completions 无 key 均 401=存在且要求鉴权）。
- **modelengine**：`EngineGLM` 类型 + 预置引擎卡（Label「GLM 云端」，默认
  模型 glm-4.6，展示顺序在 DeepSeek 之后）+ `UpdateGLMKey` + fetchModels
  认证/401 文案 + BuildChatURL key 注入。云端属性（IsLocal=false）——全局
  离线模式自动跳过、路由/用量统计按既有数据面自动归类（cloudEngineSet +glm）。
- **key 全链路**：config `glm_api_key`（DPAPI 加密落盘 + 旧明文一次性迁移）
  → 启动注入 → SetGlmKey/GetGlmKeyStatus 绑定（CoreB，绑定面 541→543）→
  GaeaSetProviderKey 支持 glm/zhipu/bigmodel 环境变量映射。
- **前端**：模型中心引擎卡自动渲染（engineIcons/Colors/Labels +glm），Key
  输入卡（脱敏回显/保存/状态刷新），api/engines.ts 类型与包装函数。
- 验证：Go 全量绿（modelengine 预置 7→8 引擎断言同步 + GLM key/URL/云端
  属性测试）；vitest 818/818（148 文件）；tsc/eslint 0；drift PASS（543）。
- 使用：模型中心 → 引擎管理 → GLM (智谱) 卡片填入 open.bigmodel.cn 的
  API Key → 测试连接 → 刷新模型（glm-4.6 / glm-4.5-air / glm-4.5-flash 等）
  → 绑定到各功能域或设为活跃引擎。

## v4.10.0 · Herdsman CLI 错误透明化 (2026-08-30)
> 真机诊断：模型中心「模型库」报「模型目录不可用，herdsman CLI 调用失败:
> exit status 3」——CLI 其实把结构化错误写在 stdout，旧代码失败路径丢弃
> stdout 只回显裸退出码，真实原因全被吞掉。
- **根因（本机三路实证）**：Herdsman 桌面端本次以**管理员身份**运行——非
  提权调用方查其 MainModule 被拒、`\\.\pipe\Herdsman-skill-v1` 连
  READ_CONTROL（Get-Acl）都拒绝；提权进程创建的命名管道 DACL 只允许提权
  令牌，普通权限的 gaea 打开即 Access is denied → CLI exit 3。旧提示
  「请确认桌面端已启动」完全误导（桌面端明明在跑）。
- **修复**：runHerdsmanCLI 失败路径捕获 stdout/stderr，优先解析 CLI 的
  JSON 结构化错误（error 字符串/对象两态兼容），裸退出码只作最后兜底，
  stderr 摘录一并透出；parseHerdsmanOpResult 同步容忍对象态 error。
- **定向提示**：Access is denied → 追加「疑似 Herdsman 以管理员权限运行，
  普通权限的 gaea 无权连接其控制管道，请用普通方式重启 Herdsman 桌面端」。
- **用户侧解法**：普通方式重启 Herdsman（若快捷方式/兼容性设置勾选了
  「以管理员身份运行」请取消）；或以管理员运行 gaea（不推荐，常驻提权）。
- 验证：Go 全量绿（+3 测试：假 CLI 端到端——非零退出透出结构化错误+定向
  提示且不再只见 exit status / 两态 error 解析+BOM 兼容 / 提示文案）；
  前端零改动；绑定面 541 不变（drift PASS）。

## v4.10.0 · 工作人设收口：办公秘书 + 节奏豁免 + 出口净化 (2026-08-30)
> 用户拍板：gaea 是工作助理，应专业、严谨的办公秘书，不是文艺女青年。
> 微信/语音实测「[SPLIT] 裸漏 + 答非所问」三根因一并收口。
- **节奏引擎专业人格豁免**：preset 带 professional tag 的人格（gaea、新增
  secretary）在 DecideRhythm 永久豁免碎碎念/独白拆分——PAD 标尺
  （-100..100）下 chatter 阈值（aro>0 && aff>3）形同虚设，办公通道每轮
  回复都被压成 ≤30 字碎片，这是「答非所问感」的第一根因；乐园陪伴人格
  （genki/tsundere 等 29 个）的碎碎念节奏保持原设计不动。
- **新增「办公秘书」人格**（PersonalityPresets 30→31 + 详细模板）：
  结论先行、要点分明、完整句；禁碎碎念/撒娇/文艺腔/情绪化/客服腔/
  网感用语；角色中心可选。语音与微信默认人格仍为 gaea（VoiceGuide 本就
  专业向，从此不再被节奏引擎带偏）。
- **[SPLIT] 出口净化**：该标记是 chatter 模式的内部格式协议，全仓库此前
  **零消费方**（节奏发射器只在 GUI 气泡路径生效）——WhisperChat 是
  GUI/微信/语音三出口的共同上游，在此把标记归一为换行（SplitOnMarker
  分条、去空条），任何出口不再见到裸标记。
- **搜索触发收窄（宁漏勿误）**：删除「什么/怎么/为什么/在哪/什么时候/
  最新/最近/实时/介绍一下/告诉我/帮我找/多少钱/是谁」等对话高频子串——
  朴素匹配把无关网页摘要灌进角色回复，是「答非所问」的第二根因；保留
  显式动词（帮我搜/查一下/搜索/上网查…）+ 硬时效词（天气/股价/汇率/
  金价/油价/比分/新闻）。身份守卫保留为纵深防御。
- 验证：Go 全量绿（+8 测试：professional 豁免（含计数器强制切换前级）/
  陪伴人格原设计保持 / secretary 注册 / 预设计数 30→31 / SplitOnMarker /
  触发词收窄矩阵（误触发 5 例 + 应触发 4 例））；前端零改动（vitest 818/818
  沿用）；绑定面 541 不变（drift PASS）。
- 说明：terse 镜像（≤15 字）维持 v4.8.3 口径（短疑问豁免已在）；情绪系统
  在工作通道只影响 TTS 韵律（Mood→TTS 既有设计），不进措辞。

## v4.10.0 · 做梦 2.0 蒸馏真实合并 (2026-08-30)
> 路线图 T0「做梦 2.0」第一刀：自动做梦按 name upsert 只增不减，确定性重复
> 记忆从此有了**非破坏**合并通道。
- **检测（memory.DistillMergeCandidates 纯函数）**：同空间内 ①归一化同名异写
  （存储后端自身已归一 kebab-case，本规则为导入数据兜底）②异名+同
  type+kind+描述逐字相同（重复沉淀主场景）；跨空间一律不成候选（双空间
  红线）；组内 UpdatedAt 降序取保留条，同输入同输出，候选封顶 8 条。
- **执行（control.DistillMerge）**：锁内重算候选集校验配对——绑定面不可被
  用于归档任意记忆，越权配对拒绝；Store.Archive 归档较旧条（**可逆**，不删
  数据）+ Touch 较新条 + refreshMemoryLocked；dream-audit.jsonl 落
  source=distill_merge 审计行。
- **前端**：记忆面板「建议」新增「重复记忆合并」卡区（保留/归档对 + 理由 +
  一键合并），i18n 三语 2 键；GaeaAcceptMergeSuggestion 新绑定（540→541，
  OfficeB）。既有办公事实库的 GaeaMemoryDuplicates/Merge（模糊相似度+硬
  删除）不受影响——两个库两条通道，互补不重叠。
- 验证：Go 全量绿（+7 测试：slug 变体 / 同类型同描述 / 跨空间排除 / 封顶与
  稳定 / 执行器与检测器一致性 / 越权拒绝 / 视图映射）；vitest 818/818
  （148 文件）；tsc/eslint 0；drift PASS（541）。
- 说明：file 后端 UpdatedAt 不落盘（语义为空），检测与执行均以存储实况为准；
  大组（同描述多成员）按设计一次只出 1 个候选，合并后再次扫描渐进收账；
  「做梦 2.0 主动预取」留后续刀。

## v4.10.0 · Verifier 通道 A 引用级深化 (2026-08-30)
> 审计欠账收口：「公式重算+引用/摘要比对」中的引用级比对此前未实装——通道 A
> 只做重算零错误。
- **声明↔实况比对**：ChangeRecord 新增可选 opsJson（xlsx_apply 落卡时随卡
  携带精确 op 载荷；截断保护=超限不落，宁勿落坏 JSON）；复核时逐条回读目标
  工作簿——set_value 比值（数值浮点等值容差）、set_formula 比公式（剥 =/
  空白归一）、replace 比替换如实落盘；批量/样式类诚实跳过并计数；不符即
  通道 A fail 并给出预期/实际示例（≤3 条）。旧证据卡无 opsJson →「声明比对
  不适用」，宁漏勿误。
- **接线**：GaeaVerifyRecord xlsx_apply 重算零错误后追加引用级复核，fail
  前缀升级整条通道 A；全簿公式评估错误（含断链 #REF! 求值）仍由 recalc
  承担，本层不做字符串级全簿扫描（防误报）。
- 验证：Go 全量绿（+9 测试：声明通过 / 值不符 / 公式不符 / 数值归一 / 替换
  通过与不符 / 旧卡不适用 / 批量跳过 / 等值与归一单测）；前端零改动
  （vitest 818/818 沿用——本轮出现 1 例 ProgrammingPage 负载型超时，系既有
  flaky 先例，单跑 12/12 绿，与改动无关）；tsc/eslint 0；绑定面 540 不变
  （drift PASS）。
- 勘误（欠账清单对账）：v4.9.0 清单中「锚点策略刻度对齐」已在 90ab160 交付
  （阈值 0-1 标尺化：weight≥0.9 / selfRelevance≥0.8 等），一并移除。

## v4.10.0 · 多跳因果链 (2026-08-30)
> v4.9.0「跨事实因果推断」纵深：证据从单跳升级为 ≤2 跳因果链（一因之因）。
- **因果链收集**：buildCausalChains 以实体为起点在 KG「导致」边上有界 DFS
  （双向：溯源找根源 + 顺藤找后果），收集 ≥2 边的链并按因果序渲染
  「A → 导致 → B → 导致 → X」；防环双保险（路径内边不重走 + 链内节点不
  重访）+ 链数封顶 4 条；被链覆盖的单跳三元组去重不重复出现。
- **证据面次序**：链优先 → 未覆盖单跳 → event_chain 关联（总上限 8 行不变）；
  系统提示词补「记忆链 = 跨多步因果链，解释时可串成完整故事」。
- 验证：Go 全量绿（新增 4 测试：两跳链 / 环路终止 / 深度上限 / 链数封顶；
  另修两跳测试的种子三元组重复）；vitest 818/818 沿用（前端零改动）；
  tsc/eslint 0；绑定面 540 不变（drift PASS）。
- 说明：跳数上限 2 是诚实边界——子串式实体匹配下更深链路噪声放大（远因
  关联度骤降），语义锚定 + 置信衰减的多跳图推理留后续设计。

## v4.9.0 · 跨事实因果推断「为什么」 (2026-08-30)
> 审计 §C「推理仅邻接遍历」深度补口：不只展示因果边，还能解释因果链。
- **GaeaWhisperCausalExplain(entity, personalityID)**（绑定面 539→540，
  MemoryB/play）：用户问「为什么<entity>」时，确定性收集证据——KG「导致」
  三元组（实体出现在因/果侧）+ event_chain 关联（涉及实体的事实对，各截断、
  上限 8 条），用当前人格口吻（Label + VoiceGuide）让 LLM 解释因果链（只用
  证据、不编造、证据不足诚实说明、≤200 字）；无证据时不调 LLM，直接返回诚实
  回退文案。5 个 Go 测试（有证据 / 无证据回退 / 空实体 / 证据构建 / 无关实体
  无证据）。
- **前端**：图谱面板新增「解释因果」按钮（查询后可用，琥珀色），解释结果展示
  在图形下方；vitest +1。
- 验证：Go 全量绿；vitest **818/818**（+1）；tsc/eslint 0；绑定面漂移
  PASS（540）。
- 说明：证据收集为确定性规则，LLM 只负责把证据讲成人话——不编造由 prompt
  约束 + 无证据零调用双保险；深度「因果图推理」（多跳推导）仍列后续。

## v4.9.0 · 首页重构（AI 多功能平台 · 星枢指挥所）+ 启动动画 (2026-08-30)
> 启动默认首页 + 两段式启动动画 + 未来感 AI 多功能平台首页（ui-ux-pro-max /
> design-taste-frontend 技能驱动，沿用 3.0「星枢 Constellation OS」令牌体系，
> 零硬编码色值）。
- **启动默认首页**：MainLayout 每次启动从首页（manifest.isHome）落地，不再恢复
  上次页面；会话内跨空间切换仍按空间恢复最后页面（原持久化机制保留）。
- **启动动画（两段式）**：index.html 静态启动屏（JS 加载期遮罩，纯 CSS）→
  BootSplash React 组件接管（旋转光环 + gaea 徽记 + 分步状态文案：核心→引擎→
  记忆→就绪 + 实时进度条）；进度/卸载全部由 timer 驱动，WebView2 rAF 节流
  （gaea-raf-degraded）与 prefers-reduced-motion 全降级。
- **首页重构**：Hero 中轴（公告 pill + 巨幅标题 + 副标题 + 中央命令条
  [AI 内核 orb / 打字 / 语音 / 发送 / ⌘K] + 语音状态行 + 快捷 chips）+ AI 状态
  细条（真实遥测 4 列）+ 能力矩阵 Bento（办公 4×2 旗舰大卡 + 板块瓦片，
  manifest 驱动，断点 6/4/2/1 列；旗舰卡 `.ml-bento.ml-bento--featured` 双类
  提权防同特异性覆盖）+ 门廊（编程独立窗口 / 设置）+ 底部信息条（会话 / 记忆 /
  系统）；全部交互元素补齐焦点环（2px 主色 outline，与 v3 壳层一致）。
- **i18n**：en/zh/zh-TW 三语同步新增 home.* 与 boot.* 共 17 键（DictKey
  编译期锁死，三语缺一即构建失败）。
- 验证：tsc / vite build 0 错误；eslint 0；vitest **809/809**（148 文件全绿；
  曾现 2 例 jsdom 并行环境抖动，单独/复跑均绿，与改动无关）；12 主题浅色抽查
  对比度正常；桌面 6 列 Bento 跨度断言正确；wails build 产物 gaea.exe 冒烟
  启动正常。
- 说明：首页视觉为「AI 多功能平台」范式（对标 Poe / Perplexity / DeepSeek
  官网），语音晶核收进命令条 orb（呼吸 / 波纹 / 辉光三态）；设计规格已同步
  design-system/gaea/pages/home.md（v3）。

## v4.9.0 · 关联入图（event_chain 因果链可见） (2026-08-30)
> 审计 §C「推理仅邻接遍历」补口：记忆关联（含 event_chain）此前只活在索引里，
> 图谱面板不可见——数据已有，只差展示。
- **子图并入关联边**：GaeaWhisperGraphSubgraph 在 KG 子图基础上并入 AssocIndex
  关联——事实 Subject 映射为实体节点，关联类型映射中文标签（event_chain→因果、
  temporal→时间、entity→同实体、emotion_peak→情绪相似、self_reference→自我、
  thematic→主题），边权重 = strength；只并入至少一端已在子图内的关联（保持以
  查询实体为中心），与 KG 边去重。只读、无副作用、绑定面不变。
- **前端**：因果关联边用琥珀色虚线描边（与普通/情绪边区分），图例补「因果」。
- 验证：Go 全量绿（新增 2 测试：关联并入 / 不连通跳过）；vitest **817/817**（+1）；
  tsc/eslint 0；绑定面 539 不变（GaeaWhisperGraphSubgraph 签名未变）。
- 说明：关联数据源 = 记忆整合（LLM consolidation 的 event_chain）+ 冷启动启发式
  （同子类标 event_chain，语义近似因果前后）；LLM 深度跨事实因果推断仍留后续。

## v4.9.0 · 图谱因果维度 (2026-08-30)
> 审计 §C 欠账收口：「图谱上无因果维度」——确定性因果模式入图，无需 LLM。
- **因果三元组提取**：extractCausalTriples 从事实摘要提取 {因, 导致, 果}——
  因为/由于…所以…、X导致/引发/造成Y、X让我/使我Y 四类模式（正则锁定 + 测试），
  直接式匹配时剥离「因为/由于」前缀；情绪经 attachEmotion 随事实落图。
- **接线**：ingest 逐事实提取（extractTriples）与文档导入（fact_landing）两条
  路径都产因果边；图谱面板无需改动（「导致」谓词天然渲染 + 情绪着色已生效）。
- 验证：Go 全量绿（新增 6 测试：因为所以 / 直接式剥离 / 让我式 / 无模式 / 情绪
  携带 / 摄入链路）；vitest 816/816 沿用（前端零改动）；tsc/eslint 0；绑定面
  539 不变。tasks 压力测试在全量并发下再现已知 flake（CI 重试语义，单跑绿）。
- 说明：本版为确定性启发式因果；LLM 深度因果抽取（跨事实推断「为什么」）留
  后续设计；关联表 event_chain 已存在但图谱面板未展示，列观察项。

## v4.9.0 · 图谱情绪维度（轻语关系图谱做深） (2026-08-30)
> 审计 §C 欠账收口：「三元组主语几乎全为「用户」+ 情绪活在图外 EmotionState」。
- **三元组情绪维度**：Triple 增 EmotionLabel / EmotionalIntensity / Valence；事实
  提取时经 attachEmotion 把情感快照落进三元组（效价 >0.15 正面 / <-0.15 负面 /
  其余中性，确定性派生）；新增 AddTriple 保留情绪（Add 兼容旧调用）；子图边带
  EmotionLabel 供前端着色。
- **主语实体化**：BASIC_PROFILE 不再硬编码「用户」→ 用档案键（如「生日」）作
  主语、谓词「属性」；关系类（赞赏/表达脆弱/关系）保持「用户」主语语义。
- **持久化**：whisper hermes.db 迁移链 V13→V14（knowledge_triples 增 3 列，
  ALTER ADD COLUMN 带默认值，存量安全）；repo 读写带情绪；schema_v13 测试版本
  断言同步 V13→V14。
- **前端**：图谱边按情绪着色（正面绿 / 负面红 / 中性灰）+ 情绪图例；
  WhisperGraphEdge 增 emotionLabel。
- 验证：Go 全量绿（新增 5 测试：实体化+情绪 / 负面标签 / AddTriple 保留 / 子图
  边情绪 / 落库读回）；vitest **816/816**（+1 图谱情绪）；tsc/eslint 0；绑定面
  539 不变。全量并发下 tasks 压力测试偶发 flake（CI 同款重试语义，单跑绿）。
- 欠账延续：因果维度（「因为…所以…」边）需 LLM/事件链，未做；时空维度暂以
  事实 CreatedAt 承担，未建专门时间索引。

## v4.9.0 · 记忆重述 + 锚点刻度对齐 (2026-08-30)
> 记忆回放收尾两刀：LLM 叙事重述 + 锚点策略阈值刻度修正。
- **GaeaWhisperMemoryRetell（绑定面 538→539，MemoryB/play）**：让 gaea 以当前
  人格口吻把情节/锚点记忆「重述成故事」——输入复用确定性回放（摘要 + 情绪 +
  原文对话），系统提示要求第一人称、称对方「你」、≤300 字、不复述字段名、结尾
  带当下感受；模型绑定沿用轻语 chat 功能绑定（引擎经 opts 显式传入）。4 个 Go
  测试（episode / anchor / 未知类型 / 上下文组装）。
- **锚点策略刻度对齐（偏离 ackem 原值，已注释说明）**：factExtractionPrompt 的
  weight/selfRelevance 标尺是 0-1，原阈值（weight≥2、selfRelevance≥4.0/4.5）
  在标尺上不可达，导致里程碑/关系分支永不触发。对齐为 0.9/0.8/0.9（0-1 标尺
  语义等价），补策略测试 2 组；旧夹具按新语义修正。
- **前端**：情节/纪念日弹窗新增「让 gaea 重述这段记忆」按钮（加载/失败/叙事块
  三态），MemoryRetell 组件两处共用。vitest +2（episode / anchor 重述）。
- 验证：Go 全量绿；vitest **815/815**（+2）；tsc/eslint 0；绑定面漂移
  PASS（539）。

## v4.9.0 · 时间锚点「重访那一天」 (2026-08-30)
> 记忆回放续刀（审计 §C 欠账收口）：锚点策略接线 + 纪念日入口 + 锚点→情节回放。
- **写路径接线（审计同类骨架欠账：ShouldWriteTemporalAnchor/BuildTemporalAnchor
  有定义无生产调用）**：MemoryWritePayload/IngestTurnArgs 增 TemporalAnchorSink →
  extractFactsViaLLM 逐事实评估（IsNew = FactStore 前后计数差，去重不重复锚点）→
  Orchestrator.AddTemporalAnchor（持回合锁，防与状态快照竞态）落
  State.TemporalAnchors 随 companion_state 持久化。命中条件沿用 ackem 策略
  （周期纪念日/里程碑/关系/高情绪）。
- **读路径**：GaeaWhisperAnchors（锚点列表，日期降序）+ GaeaWhisperAnchorReplay
  （锚点 → linked 事实的 (session, turn) → 覆盖情节 → buildEpisodeReplay 重建
  原始对话；未命中情节 Replayable=false 回退锚点摘要 + 事实摘要）。绑定面
  536→538，MemoryB/play。
- **前端**：记忆库新增「纪念日」tab（日期 + 类型徽标 + 情绪）；点击打开「纪念日
  回放」弹窗（锚点摘要 + 关联事实 chips + 回放气泡，与情节回放共用
  ReplayDialogue 组件）。
- 验证：Go 全量绿（新增 8 测试：写路径 4 + 读路径 4）；vitest **813/813**（+2）；
  tsc/eslint 0；绑定面漂移 PASS（538）。
- 观察项/欠账：锚点策略阈值（weight≥2、selfRelevance≥4）与 LLM 抽取标尺（0-1）
  刻度不一致，实际常命中「周期纪念日」分支，刻度对齐留后续决策；「重访」目前为
  确定性原文回放，LLM 重述（把摘要讲成故事）未做。

## v4.9.0 · 轻语记忆回放 (2026-08-30)
> 审计 §C 乐园做深欠账收口（「记忆回放」零代码 → 确定性重建原始对话）。
- **后端 GaeaWhisperEpisodeReplay**（绑定面 535→536，MemoryB/play）：按情节 ID
  从 hermes.db 读情节，再按 SourceSessionID + [StartTurn, EndTurn] 从
  chat_history 重建原始对话——纯确定性、不调 LLM；过旧情节因 chat_history
  裁剪（最近 2000 行）无原始对话时 Replayable=false 回退为仅摘要。3 个 Go 测试
  （轮次范围重建 / 无历史回退 / 未找到与空 ID）。
- **前端记忆库情节弹窗「回放原始对话」**：用户/gaea 对话气泡 + 轮次标注，
  加载/失败/不可回放三态；WhisperMemoryLibrary +2 vitest（回放渲染/摘要回退）。
- 验证：Go 全量绿（whisper 包在全量并发下偶发环境挂起，单跑 1.5s 绿——CI 同款
  flaky 重试语义）；vitest **811/811**（148 文件）；tsc/eslint 0；绑定面漂移
  PASS（536）。
- 欠账延续：「重访雨夜」的时间锚点入口（anchor→episode 索引）与图谱情感/因果
  维度仍列后续；本版先把「原始对话可回放」落地。

## v4.9.0 · 构建冒烟自动化 (2026-08-30)
> v4.8.3 教训收口（构建链路，不增绑定）：build.bat 不再无条件打印成功块。
- **build.bat 真实退出码 + 产物新鲜度守卫**：构建前删除旧产物（被常驻实例
  锁定时显式报错），`call wails build` 失败即停（errorlevel 检查），构建后
  必须重新生成 build\bin\gaea.exe 否则报错——「判断构建成败唯一可信标准 =
  真实退出码 + 新产物」。
- **自动冒烟**：构建成功后默认复制到 .tmp\smoke-gaea.exe（临时副本，规避
  常驻实例/AV 锁）跑 scripts/smoke.ps1（127.0.0.1:18999 /api/health 200 +
  status=ok 响应体校验），失败即停并提示勿发布；`build.bat skip-smoke` 可
  跳过（快速迭代，发布前不得跳过）。
- **smoke.ps1 增强**：Start-Process 失败显式报错、响应体 JSON 校验
  status=ok、finally 判空回收。
- 验证：默认路径实跑（wails build 44.5s + 冒烟通过 + 冒烟进程回收）；失败
  路径用假 wails 桩实测 [FAIL] wails build failed + exit 1；桌面复制失败仅
  告警不阻断（SAC 可能拦截，产物仍在 build\bin）。

## v4.9.0 · 欠账收尾小步 (2026-08-30)
> v4.8.3 发布后的代码级欠账收口（不增绑定，535 不变）：VoiceStart 门小修
> + 持久化原子写统一 + XlsxPreview 大表格虚拟滚动。Go 全量绿、vitest
> 809/809、tsc/eslint 0。
- **VoiceStart realtime 门小修**：端到端实时模式的回复走服务端 response
  事件（事件泵不经 whisperChatFn），VoiceStart 在 realtime 在位时不再要求
  whisper 对话回调——whisperChatFn=nil 也能启动；拼接管线双门（ASRReady +
  WhisperReady）逐字节保留；新增两回归测试。
- **持久化套件统一收尾**：desktop_session SaveModes 走 fileutil.AtomicWrite
  （临时文件 + rename，崩溃不留半截 JSON）；gaea/archive JSONL 单次 Write
  落整行（数据 + 换行同缓冲，消除双写撕裂窗）；新增 7 回归测试。
- **XlsxPreview 大表格行虚拟滚动**（观察项收账）：300 行以上只渲染可见窗口
  ±10 overscan + spacer 行保滚动条总高，冻结行常驻；后端预览上限
  2000×100 不再整表 20 万 td；小表全量渲染逐字节不变；2 新 vitest。

## v4.8.3「微信图片双向」(2026-08-30)
> v4.8.2 发布当日真机实测复盘 + 微信图片双向真协议实装。零新增绑定（535
> 不变），协议经三方印证（本机抓包实测解密 + hermes-agent weixin.py 生产
> 实现 + openilink SDK 导出符号）。Go 全量绿、前端零改动（vitest 807/807
> 沿用）。详见 releases/v4.8.3.md。
- **识图排障（慢+乱字母复盘）**：身份类问题（你是谁/你会什么…）跳过联网
  搜索——「你是谁」误触发搜索把无关英文网页摘要注入提示词，角色扮演模型
  把片段混进回复（乱字母根因）；WhisperChat 日志预览按字节切汉字出
  \xe6\x81 伪影修复（rune 化）。
- **微信出图回推（真机 delivered）**：getuploadurl（filekey/aeskey/md5/
  PKCS7 filesize）→ CDN 密文直传（x-encrypted-param 票据）→ sendmessage
  image_item 图片卡片 + caption 独立补发；任何失败降级文本卡片（逐字节
  不变）。真机验证 stage=delivered。
- **微信发图识别（真机两连发通过）**：入站图片 type=2 + aeskey/
  media{full_url, encrypt_query_param, aes_key} 防御解析；DownloadImage
  Encrypted（dial-time SSRF/20MiB → AES-128-ECB 解密 → 魔数终审才落盘）；
  file:// 分支限 TempDir。
- **识图模型升级**：多模态 Qwen3.6-35B 主模型优先（真机探针实测视觉链路
  完好；手写体显著强于 OCR 专线；与聊天同体常驻显存零额外开销），
  PaddleOCR → MinerU → OvisOCR2 三级链降为兜底。
- **关键坑锁死**：入站图片 type=2（非 3）；aes_key 必须 base64(hex字符串)
  （base64 原始字节 = 接收端灰框）；上传域与扫码 baseurl/redirect_host
  无关（无需重扫）；media_crypt.go 手写 AES-128-ECB + PKCS7（Go 无 ECB）。
- **抓包基建留存**：wx_capture.jsonl（qr_status/inbound_media/upload_probe
  三类）——探针时代产物，排障可用。

## v4.8.2「欠账收尾」(2026-08-30)
> v4.8.1 欠账收口：权限升级请求（v3.7.0 挂账独立一刀）+ 竞态/flake 全治理
> （含 Cancel 被 succeeded 吞掉的真生产竞态）+ Realtime S2 事件环骨架
> （16k→24k 重采样/TurnControl/事件泵/五重降级护栏，真机欠账如实记账）。
> Go 全量绿、vitest 807/807、tsc/eslint 0、drift PASS。详见 releases/v4.8.2.md。
- **权限升级请求**：request_permission 工具（reason 必填/headless 降级）+
  硬纪律三闸（deny 先行/hardAsk 拒升级/批准只写 glob 规则表不绕闸门）+
  五决策接线 + 审批卡 request 形态（reason 原文块）+ 会话 glob 规则表补全。
- **竞态/flake 治理**：Cancel 收尾窗竞态修复（×10 压力绿）+ stubGate 测试桩
  加锁 + filewatch 风暴测试时序根治（全量实战零 FAIL）+ ProgrammingPage
  显式 5s 超时。
- **Realtime S2**：Resample16kTo24k 纯函数（0.0077% 误差）+ 事件常量 +7 +
  TurnControl 可选接口 + voice_manager 事件泵（barge-in 三联/24k WAV 冲洗/
  降级拼接）+ 前端 PCM 推送死门打通；未配置=逐字节零变化（守护测试）。

## v4.8.1「欠账清尾」(2026-08-30)
> v4.8.0 欠账两刀收口：全局离线模式设置 UI（绑定面 533→535）+ Realtime S1
> （配置落盘 + DPAPI key + VoiceSettingsPanel 入口）。Go 全量绿、vitest
> 803/803、tsc/eslint 0、drift PASS。详见 releases/v4.8.1.md。
- **离线模式设置 UI**：SecurityPanel「全局离线模式」总闸段（回填/即存/失败
  回滚）+ GaeaGet/SetOfflineMode 绑定 + shelf 内存同步。
- **Realtime S1**：realtime 三键落盘（provider 仅 openai/Key 走 DPAPI 密文）+
  initVoice 注入（接线位置测试守护）+ realtimeReady = 配置且构造成功 +
  VoiceApplySettings/GetSettings 扩三键（明文 Key 永不出后端）+ 面板入口段。

## v4.8.0「全面铺开 · 触点纵深」(2026-08-30)
> 六线并行调研（多子代理分工、文件足迹不相交）→ 七刀实现。意图内核纵深
> （读屏多显示器/LLM 兜底/生图产物回推）+ 微信通道离线收敛 + 全局离线模式
> 总开关 + 成本知识图谱可视化（绑定面 532→533）+ 实时语音 Realtime S0 铺底。
> Go 全量绿、vitest 800/800（146 文件）、tsc/eslint 0、drift PASS。详见
> releases/v4.8.0.md。
- **读屏纵深**：`screen.Monitors()`/`CaptureArea` 多显示器枚举；intent 序数
  解析「第N(块)屏/主屏/副屏」（动词锚定窄规则，越界诚实报错）；OCR 文本
  >300 字本地摘要朗读（只走 Herdsman，失败退截断）；截图留档默认关。
- **intent LLM 兜底**（默认关）：规则未命中的受控冷路径——白名单
  navigate/status/read_screen + 0.75 置信门 + 2s 硬超时 + manifest 校验；
  dryRun 恒不调用；命中复用既有执行层。
- **生图 CardPath 接通**：勘误「生图异步」——实为同步阻塞，首图 FilePath
  即 CardPath；微信入口「（产物：路径）」从此有真实数据。
- **iLink 离线收敛**：per-peer 限频 20 条/分 + 4KB 截断 + 多媒体上限 5；
  图片→vision 识别管线（SSRF/20MiB/魔数三重防线，OCR 注入式接线）；防御
  解析矩阵（多态 JSON 降级不炸整批）；SendFileCard seam + 真机抓包清单。
- **全局离线模式**（跨版欠账清账，默认关）：`EngineType.IsLocal()` +
  routeModel 三步云过滤；无本地可用走既有「模型不可用」降级。
- **成本知识图谱**：costref.BuildGraph 纯函数组图器（7 节点/6 边、树聚合/
  条目展开双视角、EntryName 精确优先、截断与悬挂边防护）；CostGraphView
  零依赖 SVG 双视角 + Modal 明细；成本库第 8 模块。
- **Realtime S0**：internal/realtime seam（RealtimeSession/注册表/openai 实现）
  + VoiceHealth realtimeReady + 优雅降级；14 离线测试；S1/S2 留欠账。

## v4.7.0「命令面板接内核 · 读屏」(2026-08-30)
> 路线图 §10.4a S4.6 完整收口：桌面命令面板接统一意图路由内核（语音/微信之后
> 的第三个入口）+ 屏幕感知能力（读一下屏幕）纳入能力面。GaeaRouteIntent(text,
> dryRun) 绑定（531→532，dry-run 预览-确认制防搜索词误触发）+ SearchModal 指令
> 预览卡 + 真·Ctrl+K（办公板块让位工作台面板）+ read_screen 三入口免费受益
> （截屏→OCR→TTS 朗读/内联回执，临时文件即用即删）。Go 全量绿、vitest 796/796
> （145 文件）、tsc/eslint 0。详见 releases/v4.7.0.md。
- **GaeaRouteIntent**：dryRun=true 零副作用预览（校验口径与执行层一致——未知
  板块/媒体域缺失按未命中）；false 真执行。S4.6 显式豁免旧「零新增绑定」纪律
  （面板是前端入口，绑定即其回传通道，intent_router.go 头注已记录）。
- **预览-确认制**：SearchModal 命中出「指令」卡（动作标签 + 预览语 + 执行按钮），
  点执行才真跑；执行回执内联，导航类 emit gaea-intent-navigate 复用 S4.4 切板块
  后收面板；未命中检索行为零变化。
- **真·Ctrl+K**：MainLayout 全局快捷键落地（tooltip 此前名不副实）；gaea 工作台
  内让位自有 CommandPalette 防双面板。
- **读一下屏幕**：intent.ActionReadScreen 窄规则（读/念/看/识别+屏幕、屏幕上有
  什么、读屏——不含裸读/看）+ execReadScreen（screen.Capture → 临时 PNG →
  GaeaOCRText 既有 OCR 链 → 300 字截断回传）；语音经 TTS 朗读、面板内联展示、
  微信走文本回推；失败诚实回执不坠聊天。

## v4.6.1「微信统一路由 · 规范包机制 · 归因对标」(2026-08-30)
> 审计补课续刀：S4.5 微信消息接统一路由（routeIntentWithResult 产物感知版本，
> 提醒特例之外 navigate/生图/状态/提醒全命中即执行，未命中才走聊天）+ iLink
> 图片消息协议第一刀（image_item/file_item 防御性解析 + 非文本转模型提示行，
> 未知空项宁漏勿误）；规范包机制化（Checker 注册表 + 红头/造价工程表式双检查
> 器，GaeaDocumentLint 聚合，OfficePanel 按规范包分组）；成本归因对标（明细
> vs 参考指标 P25/P75 带宽，差幅等级/贡献金额/主因 TopDrivers，参考池排除
> 本项目，绑定面 530→531，FiveCalcPanel 归因区）。Go 全量绿、vitest 791/791、
> tsc/eslint 0。详见 releases/v4.6.1.md。

## v4.6.0「双空间收尾 · 纵深补课」(2026-08-30)
> 执行审计（docs/audit-2026-08-30-v4-execution-review.md）后的第一轮补课：
> 红线缺口三条（记忆注入跨空间 / 任务分账未启用 / 事件过滤仅 1 处）全部接线，
> 前端治理收尾（keepAlive 裸轮询 8 处门控 + CSS 真硬编码 token 化），C 类纵深
> 按价值排序落地三件——Mood→TTS 闭环 / Verifier 通道 B 真视觉 diff + 失败回
> Plan / 询价异常检测 + 价格预测 + OCR 报价单自动入询价库。每刀带「纵深检查」。
> 验证：Go 全量绿（无 FAIL）；vitest **791/791**（144 文件）；tsc/eslint 0；
> 绑定面不变（变参向后兼容）。详见 releases/v4.6.0.md。
- **红线 ①记忆注入按空间收窄**：`boot/sysprompt.go` + `controller_memory.go`
  `refreshMemoryLocked` 传 `Options.Space` → `InSpace` 读端视图——work 会话只
  注入 work 记忆、play 只注入 play；mode=off 旧行为零变化。
- **红线 ②任务分账生产启用**：`[tasks]` 配置段（max_concurrent/per_space/
  priority）→ `startTaskScheduler` 落默认 {work=1, play=1} + 价格抓取优先；
  显式空表可关分账回退全局 sem。
- **红线 ③事件过滤推广**：`onTaskEvent(cb, space?)` 订阅层过滤，任务中心/
  运行角标/价格源/索引面板全按 work；MainLayout 主事件流走 subscribeForSpace。
- **keepAlive 轮询门控**：TaskCenter/SubagentsPanel/FeatureModelBar/
  ProgrammingPage/BenchmarkSection/useStatsState/useImageGenQueue/useBridgeWatch
  八处裸轮询接入 usePollingGate（后台空转归零）。
- **Mood→TTS 闭环**：长期心境 4D EWMA → 连续韵律中文指令（低沉/温暖/不安/
  平缓/轻快…），中性轮次由心境主导「听得出她今天低落」，强情绪标签仍主导。
- **Verifier 通道 B 真 diff**：soffice 转 PDF + pdftoppm 逐页渲染 + 纯 Go 像素
  差异率（页数联合判定 pass/warn/fail），审计产物落 journal/verify/<id>/；
  失败回 Plan：证据卡内联结论 + xlsx_apply 一键「重新规划」。
- **询价飞轮反向 + 异常检测 + 预测**：OCR/图片报价单确认导入自动幂等写入询价库
  （source=OCR报价）；调差建议带 正常/关注/异常 分级；同标题询价序列线性回归
  预测下期价。
- **欠账清单**（如实）：规范包机制化 / 成本知识图谱+归因 排下轮；生命库可写化
  评估结论 = 不做盲写 Herdsman 库（锁/Schema/竞争三类风险），gaea 侧角色资产
  表另案评审。

## v4.5.0「指令中枢」· 统一意图路由内核 + 语音指令 (2026-08-30)
> 路线图 §10.4a（2026-08-30 规划修订插入）第一刀：落地触点层「同内核多入口」
> 的架构承诺——一层「意图 → 能力 → 结果回传」的统一路由内核，语音 / 微信 /
> 命令面板共用；语音指令（JARVIS 一档）首发接线。「打开绘梦」「画一张赛博朋克
> 城市」「现在用什么模型」「提醒我 30分钟后 喝水」——任何模态，唤起同一个 gaea。
> 验证：Go **108/108 包**；vitest **789/789**；tsc/eslint 0；绑定面 **530 方法**
> 零新增（执行结果走事件 + TTS 回传，漂移防线不动）。详见 releases/v4.5.0.md。
- **intent 解析包（S4.1）**：纯函数规则引擎（导航/生图/状态/提醒四类意图 +
  板块别名表贪婪匹配）；「宁漏勿误」纪律——「画得不错」「画了半天」绝不触发生图；
  LLM 兜底分类器留 Capability 接口位。
- **能力执行层（S4.2）**：`routeIntent` 统一入口——navigate（manifest 校验 +
  `gaea-intent-navigate` 事件）/ generate_image（mediaState 自由生图）/ status
  （引擎摘要）/ reminder（复用离线代办解析与持久化）。
- **语音通路（S4.3）**：voice 对话回调内先过路由——命中即能力执行、回复经同一
  TTS 流程播报，未命中透传原轻语对话；voice 包零改动。
- **前端导航（S4.4）**：MainLayout 订阅 `INTENT_NAVIGATE` → `navigateBoard`
  （语音「打开绘梦」自动切空间，复用 v4.3.2c 机制）。

## v4.4.0「触点」· 微信遥控器一期：离线代办 (2026-08-30)
> 路线图 §10.4 v4.4「触点」第一刀：微信从「能聊天」升级为「能接活的遥控器」。
> 在微信里对助手说「提醒我 …」→ 桌面端中文时间解析建提醒 → 到点经微信回推——
> 官方元宝做不了的桌面端「离线代办」差异化主打。WeixinPage 书房板块页落地。
> 验证：Go **107/107 包**；vitest **789/789**；tsc/eslint 0；绑定面 **530 方法**
> 漂移 PASS（+5）；spaceBindings **247 方法**全覆盖；版本五处统一 4.4.0。
> 详见 releases/v4.4.0.md。
- **微信主动推送通路**：`weixin.Server` 记录最近活跃会话（fromUser/contextToken），
  新增 `Push` 向其回推文本（httptest 校验目标与 item）。
- **离线代办提醒域**：中文时间解析（相对时长 / 日期前缀+段词 / 裸时刻；中文数字
  含「十」进位；无段词按字面，确认文案带完整时间供纠正）→ `wxReminder` JSON
  持久化（重启恢复）→ 微信消息任务路由（提醒类接管，失败回格式提示）→
  20s ticker 到点回推（失败重试 ≤5 次标 failed）。
- **WeixinPage 落地（书房板块）**：扫码绑定流（QR 轮询 / 配对码 / confirmed 落
  WhisperAssistantSave 自动重拉通道）+ 通道状态徽标（运行/过期/未绑定）+ 提醒
  列表（手动新建 / 删除 / 回推开关）+ 指令说明。weixin 板块 inMenu=true 进
  rail 与首页左翼书房格；`WhisperWeixin*`/`WhisperAssistant*`/`WeixinReminder*`
  共 12 个绑定自 LegacySurface 转正或新增，spaceBindings 全归 work（235→247）。

## v4.3.2「双翼·中庭」· 首页重构 + 空间导航收敛 (2026-08-30)
> 首页体验重构：双翼·中庭三区布局（左书房 / 中语音对话 / 右庭院）+ 空间导航
> 收敛（移除一级导航空间切换，首页双翼即空间入口，按板块自动切空间）。
> 设计遵循 design-taste-frontend（不对称 / 磁吸核心 / 避免等宽栏）；沿用星枢
> 深空玻璃拟态 + gaea-glow。验证：Go 全量绿；vitest **789/789**；tsc/eslint 0；
> 版本五处统一 4.3.2。详见 releases/v4.3.2.md。
- **首页「双翼·中庭」**：中庭 = 语音 + 打字一体对话条（VoiceChatText 共用管道 +
  放大 orb 磁吸锚点，hero 让位细眉）；左翼「书房」2×2 紧凑格（办公/造价/记忆/模型）；
  右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊 = 编程（独立窗口徽标）+ 设置。
- **命名**：工位→**书房**、乐园→**庭院**（三语字典 zh/zh-TW/en 同步）。
- **空间导航收敛**：移除 rail 顶部空间切换按钮；`navigateBoard` 按板块
  manifest.space 自动切空间（书房板块→work / 庭院板块→play / 编程→independent）；
  rail 展示全部板块；搜索 scope 文案同步 书房/庭院。
- **桌面端**：gaea-v4.3.2.exe（33MB，SHA256 6a0486db）+ 冒烟 /api/health 200。

## v4.3.1「乐园」后续小步 (2026-08-30)
> v4.3 后续小步收口：主动关心定时推送频控 + 创作间世界模型面板 + 角色参考图 +
> 朗读情绪 UI。设计沿用 `docs/gaea-v43-play-deepen-design.md`（v4.3c/e/f/g 补完）。
> 验证：Go 全量 **118/118 包**；vitest **789/789**（144 文件，+20）；tsc -b / eslint 0；
> 绑定面 **525 方法**漂移 PASS（+3）；spaceBindings **235 方法**全覆盖断言；
> 版本五处统一 4.3.1。详见 releases/v4.3.1.md。
- **主动关心定时推送频控（v4.3c 补完）**：app 层 ticker 四信号评估（AttentionManager
  频控 ≤3 条/小时 → MatchHabits 作息尊重 → DetectSpecialDatesV2 生日祝福（每天首条、
  人格感知提示词）→ 门控+合成器）→ `gaea-whisper-proactive` 事件推前端；
  新绑定 `GaeaWhisperProactiveConfig/SetProactiveConfig`（开关/上限/间隔/免打扰时窗）；
  前端 WhisperGraphPanel 订阅显示推送气泡（含 birthday 徽标）。play 红线零落盘。
- **创作间世界模型面板（v4.3e/f 落地）**：设定页「维度化」模式（6 维度卡片就地编辑
  整存）+ 伏笔登记表面板（状态流转/回收率）+ 一致性检查面板（三类规则告警/重新检查）。
- **角色参考图 + 生图参考槽（v4.3g 补完）**：characterlib SchemaV2 迁移（reference/
  gallery_images 两列幂等）+ `CharacterGeneratePortraitWithRef`（img2img 参考槽
  denoise 0.55 + 模型门禁）+ 前端角色库参考图管理（以参考图生成剧照/删除/添加）。
- **文本朗读情绪 UI（v4.3d 收尾）**：朗读情绪选择器（9 标签对齐 EmotionVoiceMap）+
  会话情绪自动跟随 + 朗读携带 `TTSSpeakBase64WithParams` 情绪参数（无情绪回退原路径）。

## v4.3.0「乐园」娱乐做深 (2026-08-29)
> 阶段 3+ 领域包第二发：会客厅关系记忆 + 主动关心 + 情感语音；创作间图文联动。
> 设计 `docs/gaea-v43-play-deepen-design.md`（4 份只读调研：后端骨架约 70% 已存在，
> 本版以「接线 + 参数扩展」为主，与工位零交叉）。
- **会客厅·关系记忆**：`memory_associations`/`user_habits`/`temporal_anchors` 三表
  补 repo 闭环（此前有 schema 无 repos，重启关联全空）+ `ReseedAssociationGraph` 打通
  + hermes.db 外键延迟检查实证修复；`QuerySubgraph` 多跳邻接子图召回；前端
  WhisperGraphPanel（零依赖 SVG 环形邻接图、节点点击重查）。
- **会客厅·主动关心**：`GaeaWhisperProactiveNow` 评估绑定（门控+合成器现成复用，
  时段感知）+ 前端「轻语先开口」按钮（类型徽标/提示词）；定时推送留后续小步。
- **会客厅·情感语音**：`TTSProvider.SynthesizeWithParams` 参数扩展（speed/style/
  emotion；cosyvoice 工厂不再丢弃 voiceDescription；edge SSML 参数化）；
  情绪→参数映射 `GetEmotionVoiceParams`；长期心境维 `EmotionState.Mood`（EWMA α=0.01
  持久化）——「听得出她今天低落」原料就绪；`TTSSpeakBase64WithParams` 绑定。
- **创作间·图文联动**：章节配图复活死绑定（ChapterPage「配图」按钮 +
  ChapterIllustration 弹窗）；书封生成 `GaeaGenerateBookCover`（3:4 落 play exports
  + NovelPage「生成封面」按钮；修 Windows 盘符卷文件名清洗 bug）。
- **验证**：Go 全量绿；vitest **769/769**（144 文件，+10）；tsc -b / eslint 0
  （顺带修 v4.2 遗留 TS2488/spaceBindings 键）；绑定面 **522 方法**漂移 PASS（+5）；
  spaceBindings **233 方法**全覆盖断言；版本五处统一 4.3.0。
  详见 releases/v4.3.0.md。

## v4.2.0「智慧」工位造价包 (2026-08-29)
> 阶段 3+ 领域包第一发：AI 组价 + 询价飞轮 + 五算对比（垂直蓝海率先变现）。
> 设计 `docs/gaea-v42-cost-ai-design.md`；「无确认不落库」纪律贯穿三支柱。
- **AI 组价**：`cost.PriceBand` 价格带推荐纯函数（P25/P50/P75 + 离散度 + 离群 +
  置信度 + 证据链 BandSource，分位数 R-7 与 costref 同口径）；`GaeaCostCompose`
  绑定（相似清单检索：关键词 + 本地 bge-m3 语义召回 + rerank 精排 → 价格带 →
  **LLM 人材机拆解**（`routeSensitiveLocal("office")` 敏感域本地化，失败规则降级）
  → 建议视图 + `GaeaCostComposeApply` 一键回写）；前端 ComposeModal（测算明细行
  「AI 组价」按钮：价格带卡 + 证据链 8 列表 + 人材机行编辑 → 应用为明细行）。
- **询价飞轮**：`costinquiry` 包四源归一数据点（信息价/OCR报价/供应商比价/手动
  询价，`cost_inquiry_records`）+ 到期预警（valid_until）+ 调差建议（标题归一化
  匹配，|差幅|>2%）；前端询价视图（数据点 CRUD + 预警横幅 + 一键更新成本库）。
- **五算对比**：`coststage` 包估/概/预/结/决阶段值（`cost_stage_values` UPSERT）+
  对比计算（环比/累计差）+ 偏差特征（正常/关注/异常三档 + 规则诊断文案）；前端
  FiveCalcPanel（项目详情五算区：输入保存 + 对比表着色 + 偏差卡片）。
- **验证**：Go 全量绿；vitest **759/759**（142 文件，+21）；tsc/eslint 0；
  绑定面 **517 方法**漂移 PASS（+11）；spaceBindings **229 方法**全覆盖断言；
  版本五处统一 4.2.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；CHANGELOG / README / releases / AGENTS / progress 同步。
  详见 releases/v4.2.0.md。

## v3.9.0「双空间壳 + 办公信任链」(2026-08-29)
> 阶段 2 双空间壳（S2.1–S2.3/S2.3b + 页面迁入 P1）+ v4.1 办公信任链（证据链→复核→
> 回滚→规范体检）一次收官；「审阅后」护城河从设计到端到端闭环。
- **双空间壳（阶段 2）**：S2.1 壳层两视图 + 空间切换持久化（`gaea.shell.space` /
  `gaea.shell.page.<space>`）+ 删旧 10 板块导航 + 双首页（工位任务工作台 / 乐园会客厅
  创作间）+ 事件订阅空间过滤 + Ctrl+K scope；S2.2 工作台 localStorage 空间分键
  （`gaea.work.*` 旧 key 只读迁移）+ keepAlive 跨空间卸载（性能门控）+ **i18n 决策**
  （壳层 chrome+设置三语，页面 zh-only，审计「诚实 zh-only」选项）+ 页面迁入 P1
  （chat→play 对话流）；S2.3 bridge 三门面（spaceBindings 214 方法显式分类，
  workApp/playApp/sharedApp 类型级红线）+ types 全量迁移（WireShape 55 别名 +
  typesGenerationCheck 漂移校验）。
- **151 hex token 化（S0.7 遗留）**：VoiceSettingsPanel 浅色主题不可读真 bug 修复；
  语义色全 token；图表/品牌/覆盖层显式 `hex-exempt`；eslint `local/no-raw-hex` 防回归。
- **v4.1 办公信任链**：证据链（ChangeRecord 原文摘要 8KB / ChangeLedger / JournalStore
  JSONL+markdown，play 不落盘；六类写盘工具 + xlsx_apply 接入）；Verifier 双通道
  （结构/引用完整性 + 基线 PDF 渲染对比）+ 基线快照回滚（手工编辑冲突保护零覆盖）；
  GB/T 9704 红头 7 要素规范体检（GaeaDocumentLint + OfficePanel 入口）；
  前端「证据」入口（DeliverablesPanel 复核/回滚）。
- **验证**：Go 全量绿；vitest **738/738**（139 文件）；tsc/eslint 0；vite build；
  绑定面 **506 方法**漂移 PASS（新增 4 绑定）；spaceBindings 218 全覆盖；
  版本五处统一 3.9.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；CHANGELOG / README / releases / AGENTS / progress 同步。
  详见 releases/v3.9.0.md。

## v3.8.0「双空间内核 · 工位/乐园 + 质量地基」(2026-08-29)
> 双空间（工作/娱乐隔离）从规划到内核落地：会话/记忆/任务/产物/模型/工具/权限/护栏全按
> 空间装配、互不干扰；同步完成审计 P0/P1 质量地基（并发/安全/编辑脊柱）与长期规划定稿。
- **双空间内核（阶段 1）**：SchemaV14（facts/tasks space_id，旧数据回填 work）+
  会话目录分区 `sessions/work|play` + 事件日志/checkpoint 空间自描述 + `space.mode` 开关；
  记忆写侧盖章（remember/dream 指纹含 space/审计加列/play notes 不写 work AGENTS.md）+
  读端隔离（GetInSpace/citations 限定/UnifiedSearch scope 四组）+ 前端 hub/面板 scope 切换；
  `[space_profiles]` 模型按空间 + 工具装配期过滤（work 33 / play 1 / shared 13 + MCP spec 层滤）；
  任务 per-space 并发/优先级防饥饿 + cron 显式 work；权限策略按空间（play 默认不弹审批卡）+
  hardAsk 参数化 + persist_allow 分段回写 + play 内容护栏（5 处生成点钳制）。
- **质量地基（阶段 0）**：S0.1 回合级并发加固（turnMu，临时 worktree 实证修复前必崩）；
  S0.2 Registry 锁 + 幽灵名修复；S0.3 gate 原子化（撕裂换闸）；S0.4 retry_until 门控
  （堵审批绕过 shell）；S0.6 **edit_file 工具层**（grep/edit_file/multi_edit/edit_lines/
  move_file 五工具全落地）；knowledge 索引缓存 / office 原子写 / secure 非 Win AES-GCM /
  tasks 输出 LRU；前端聊天 memo+尾部窗口 / keepAlive 轮询门控；CI 新增 `-race` 并发门禁。
- **长期规划**：`docs/gaea-nextgen-roadmap-2026.md` 定稿（双空间版本重定义 + 四层落地 +
  执行计划 + 二次审核缺口清单）；99 个过时文档归档 `docs/archive/`；**用户拍板：编程板块
  保持独立 DSH 窗口、不并入工位/不共享工具面（防工具膨胀）**。
- **验证**：Go 全量 **115 包** + vet；vitest 全绿；eslint 0/0；tsc 0；绑定面 **502 方法**
  漂移 PASS；版本五处统一 3.8.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；wails build + 冒烟。详见 releases/v3.8.0.md。

## v3.7.0「办公蒸馏 codex 收官 · 引用可追溯 + 审批决策族 + 输出事件化」(2026-08-29)
> 办公蒸馏 codex 清单第二/三刀收官（C1 方案模式已随 v3.6.0 回退）。
- **C2 记忆引用可追溯**：RecallBlock 注入行带引用键 `[MEM:name]` + 句末标注纪律 +
  陈旧记忆（90 天）时效提示；回合结束解析回传并 Touch 命中记忆（未知键静默）；
  前端 remarkMemCitations + MemCitationChip 点击弹层展示记忆详情/沉淀来源
  （零新增绑定面）。
- **C4 审批决策语义族**：拒绝三分（deny 继续 / abort 终止本轮）；审批等待超时
  （`[agent] approval_timeout_secs`，默认 0=等待）；「始终允许」（persist_allow）
  策略文件回写——`ToolName(subject)` 规则写入 `[permissions].allow`，重启不失，
  hardAsk 完全降级不回写；`GaeaApprove` 重构为决策串五值，审批卡快捷键 1-5。
- **C9 任务输出事件化**：输出变更/终态经 gaea-task 事件推送整尾回放（独立节流
  与进度互不挤占），任务中心输出 dock 事件即推，2s 轮询降级为兜底。
- **C5 上下文占用状态行**：RunStatus 窗口占用迷你进度条 + 百分比；≥75%「接近
  自动压缩」/≥90%「即将强制压缩」两档预警（对齐 80%/90% 引擎线）。
- **C6 项目说明文件**：单文件 32KB 注入预算（超限 UTF-8 边界截断留标记）+
  `.gaea/AGENTS.md` 子目录约定发现（更具体者后注入，自动进记忆面板可编辑）。
- **C3 自动做梦 no-op**：dream 输入 sha256 指纹，与上次成功处理的轮次一致直接
  跳过 LLM 提炼（指纹仅在完整处理后记录）；排查确认三层渐进披露已存在，LLM
  滚动摘要层列观察项。
- **验证**：Go 全量 **114/114 包** + vet；vitest **681/681（130 文件）**；eslint 0/0；
  tsc 0；绑定面 **499 方法**漂移 PASS；版本四处统一 3.7.0；wails build + 冒烟 200。
  详见 releases/v3.7.0.md。

## v3.6.0「办公文件编辑审阅制 · 本地优先 · 对话面减负」(2026-08-29)
> xlsx AI 编辑改两段式审阅（规划不落盘 → 批准才应用）+ 原生图表嵌入工作簿 +
> PDF 统一出口 + 办公功能级 AI 本地优先 + 运行中插话；回退方案模式 v1、
> 撤下任务目标/验收清单，用户消息 Codex 式收敛。
- **xlsx AI 编辑审阅制（Plan → Apply 两段式）**：`GaeaXlsxPlanEdit`（AI 操作集在
  原文件临时副本试运行 + 逐单元格 diff 变更清单，不落盘）→ 用户批准 →
  `GaeaXlsxApplyEdit`（excelize 执行 + LibreOffice 重算）；新增 `set_style`（叠加
  样式不丢现有填充色）/`merge_cells`/`unmerge_cells`/`set_col_width`；XlsxPreview
  规划审阅卡（对标 Copilot Plan/Show Changes 范式）。
- **xlsx 原生图表**：excelize 原生图表对象嵌入工作簿（Excel/WPS 打开即可见、可继续
  编辑，非图片截图），返回锚点 + 数据供前端迷你预览。
- **PDF 统一出口**：`GaeaConvertToPdf`（LibreOffice 无头转换 + 独立 UserInstallation
  profile 防锁冲突；md/markdown 经 docx 中转），顶栏「导出 PDF」同管线；缩略图/预览
  支持 pdf/docx 文本提取。
- **办公本地优先路由**：`routeOfficeLocal`——办公功能级 AI 调用（Word/Excel 编辑、
  资料摘要、知识导入、记忆整理）默认走本地 Herdsman（数据不出本机、省 token），
  不可用回退常规路由；聊天主 agent 不受影响；`GetOfficeLocal/SetOfficeLocal` +
  安全设置面板开关（默认开）。
- **运行中插话（GaeaSteer）**：运行中消息作为当前回合 guidance 注入（不开新回合、
  不打断工具执行），未运行走 GaeaSend 排队兜底；`event.Steer` → notice 回显。
- **对话面减负**：回退 C1 方案模式 v1（mode/shouldPlan/planGate、Ask.Plan 计划卡、
  composer 模式切换器、GaeaAgentMode/GaeaSetAgentMode、AutoPlan 配置与评分字段）；
  撤下任务目标/验收清单（GoalCard + GaeaRequirement 系 8 绑定）；用户消息 Codex 式
  无气泡 + Kimi Work 式超长消息折叠（240 字符 3 行截断）。
- **修复**：whisper 关机排水丢任务（pending WaitGroup 等到排队任务执行完）；
  trajectory/contextview 空切片 null 崩溃（绑定层 Empty* 非 nil 保证 + 前端归一化）；
  TrajectoryView 测试负载 flaky 加固（显式 5s 超时）。
- **验证**：Go 全量 **114/114 包** + vet；vitest **669/669（127 文件）**；eslint 0/0；
  tsc 0；绑定面 **499 方法**漂移 PASS；版本四处统一 3.6.0；wails build + 冒烟 200。
  详见 releases/v3.6.0.md。

## v3.5.0「办公对话区标签页 · dsh-context Go 移植」(2026-08-28)
> 对话窗口上方新增 [对话 | 轨迹 | 上下文] 三标签：上下文 = 逐请求上下文构成看板，
> 轨迹 = 对齐 DSH ui-trajectory 的扁平事件账本，Agent 网络 = 主 agent + 子代理树。
- **request_header 事件（日志不变量补位）**：`event.RequestHeader` + `request_header` 日志行，
  每次模型请求前记录实际 system prompt 与工具 schema——上下文/轨迹 system/tools 分类的
  精确数据源；旧日志无此事件按估算降级。
- **上下文标签（新包 internal/gaea/contextview + GaeaContextView）**：`FoldTimeline` 纯函数
  折叠——六分类当前组成（system/tools/user/inject/assistant/tool，usage 实际 promptTokens
  等比锚定，与顶栏 ContextBar 同源；`Referenced context:` 前缀拆 inject）+ 原生 SVG 趋势图
  （步数|轮次|全局|增量四钮，增量 Phase D；压缩 ✂；点击联动步骤详情卡：输入→回复/实际
  prompt/输出/缓存）+ 事件流（注入|压缩|剪枝筛选，压缩节点 gone + 负 delta + 归档）；
  7 个折叠黄金测试。
- **轨迹标签（新包 internal/gaea/trajectory + GaeaTrajectory）**：对齐 DSH ui-trajectory
  事件账本——扁平记录表（user/request-header/assistant/tool/compact/ask/approval，带
  ts/durationMs/step；header change 检测 initial|system|tools|system-and-tools；tool
  dispatch+result 按 ID 合并、parentId 嵌套根、running/error、截断、耗时；轮间压缩
  Between-turns 区段；turn-end 错误）；前端 TrajectoryView = Duration/Turns/Calls chips +
  搜索 + 类型徽标（ASSISTANT 紫/TOOL 橙/提问 深蓝/REQUEST HEADER/COMPACTION）+ 点击展开
  检查器；9 个折叠黄金测试 + 5 个 vitest。
- **Agent 网络（FoldAgentNetwork + GaeaAgentNetwork + AgentNetworkCard）**：主 agent 根 +
  子代理树（拥有子记录的元工具调用，不写死工具名），节点聚合子树工具数/错误/估算 token/
  状态 running|error|completed；subagents/ meta 任务摘要匹配富化状态与模型；前端 SVG 树 +
  节点 token 占比环 + running 绿脉冲 + 悬停详情条；3+3 测试。
- **随版并入（并行工作流）**：记忆统一层第二刀前端收尾（GaeaMemoryUnarchiveBatch /
  GaeaMemorySetRetentionDays 的 bridge 类型、批量恢复/保留期 UI 与 mock、生命周期测试补强）。
- **验证**：Go 全量 **114/114 包** + vet；vitest **668/668（127 文件）**；eslint 0/0；
  tsc 0；绑定面 **503 方法**漂移 PASS；版本四处统一 3.5.0；wails build + 冒烟 /api/health 200。
  详见 releases/v3.5.0.md。

## v3.4.0「记忆统一层第一刀 · 统一检索收口 + 生命周期产品化」(2026-08-27)
> 路线图 V4（记忆统一层）首发：hub 搜索 4 绑定前端拼装 → 1 绑定后端聚合（GaeaUnifiedSearch
> 增三脑/文件语义两组）；归档 tab 从「永远空白」到「分页可浏览 + 一键恢复」（新增
> GaeaMemoryUnarchive）；保留期下发展示；修复漂移脚本单条差异静默放行 bug。
- **统一检索后端收口**：`GaeaUnifiedSearch` 视图扩展四组——keyword（工作区全文）+
  semantic（跨库语义）+ **brain（三脑命中，新增，a.brain==nil 时空数组不报错）** +
  **files（文件语义，新增，复用 GaeaFileSemanticSearch 抽出的私有实现）**；hub 搜索
  （MemoryHubPage.runSearch）由「4 绑定 Promise.all 前端拼装」收敛为「单次 app.UnifiedSearch」，
  四组映射回原 HubSearchHit 渲染（徽标/预览/@ 引用零变化）；WorkspaceSearchPanel 跨库模式
  零改动，语义徽标补 file kind（后端本就返回，前端类型漏声明）。测试：Combined 扩展 +
  新增 BrainNil；空 query 四组空数组。
- **归档 tab 永远空白（缺陷修复）**：前端归档 tab 读 `view.archives`，但后端 `GaeaMemory()`
  结构体无 archives 字段 → 列表永远空白；改 `GaeaMemoryArchivedList` 分页加载（每页 50 +
  加载更多 + total）。
- **恢复能力补齐（Unarchive）**：memory 包此前只有 Archive（软删）无恢复路径（注释声称
  「90 天可恢复」但实际不存在）；补双后端 `Unarchive`（sqlite 置 archived=0；file 从
  `.archive/<ts>-<name>.md` 移回主目录 + 重建索引；未归档/已清理报错）+ 新绑定
  `GaeaMemoryUnarchive`（绑定面 497→**498**）+ 归档 tab「恢复」按钮（Rollback 图标/恢复中态/
  成功后刷新提示）。
- **保留期下发展示**：`MemoryArchivedPage` 增 `retentionDays`（= ArchivedRetention 90 天），
  归档 tab 顶部「归档保留 N 天，超期可清理」，清理确认弹窗文案跟随真实保留期（此前硬编码）。
- **修复漂移脚本单条差异静默放行**：`check-bindings-drift.ps1` 判 `$diff.Count -gt 0` 但
  PS 5.1 下单条差异 `$diff` 是单个 PSCustomObject（无 .Count）→ `$null -gt 0` 为 False 静默
  放行（实测复现：新增绑定后脚本仍报 OK）；`@()` 强制数组化修复 + 脚本恢复 UTF-8 带 BOM
  （AGENTS.md 编码规范）。
- **验证**：Go 全量 **112/112 包**（+6 测试：Unarchive 双后端/app 绑定/RetentionDays/BrainNil）、
  eslint **0/0**、tsc 0 errors、vitest **654/654（124 文件，+2）**、绑定面 **498 方法**漂移
  PASS（含负向验证：单条漂移现在能红）、版本四处统一 3.4.0、wails build + 冒烟 /api/health 200

## v3.3.0「质量收敛 · eslint 存量 warnings 清零 + flaky 治理」(2026-08-27)
> v3.2.1 后的工程质量刀：366 条存量 eslint warnings 归零（配置显式化 + 死代码清理 +
> exhaustive-deps 补全 + 混合导出显式声明 + 冗余 @ts-ignore 移除）、CI/测试 flaky 治理、
> releases/README.md 历史乱码恢复、前端性能体检。纯工程质量，零功能变更。
- **eslint 366 → 0（errors 0 / warnings 0）**：
  - 配置显式化：`no-unused-vars` 加 `^_` 前缀 ignore patterns（下划线 = 显式「故意不用」，
    社区标准）、`no-empty` 开 `allowEmptyCatch`（空 catch 为降级吞错的有意设计）、
    react-refresh 开 `allowConstantExport`（纯常量导出不破坏 Fast Refresh）
  - 死代码清理 56 处（未用 import/const/函数/catch 参数/解构成员，跨 40 文件）
  - exhaustive-deps 40 处：稳定依赖补全（useCallback/store 方法/setter）+ 不稳定依赖
    显式 disable 注释（含 GhostText/useVoiceChat 两处 TDZ 陷阱的 useCallback 定义上移
    重排、GraphView/Composer 两处 disable 注释位置修正）+ 复杂表达式提取变量 +
    每渲染重建数组 wrap useMemo + ref cleanup 竞态局部变量化
  - react-refresh/only-export-components 25 处：14 个混合导出文件加文件级显式声明
    （Provider+hook 同文件、工具函数供测试/复用等设计使然）
  - 移除 10 处冗余 `@ts-ignore`/`@ts-expect-error`（wails.d.ts 已生成类型，注释多余）
- **flaky 治理**：filewatch 测试事件等待超时 3s→5s（沙箱/CI 高负载下 fsnotify 投递 +
  debounce 延迟曾致首跑假红复跑绿）；CI 后端测试失败后整体重试一次（重试后仍失败
  正常红，不掩盖真实缺陷）；确认 CI 已排除 internal/tts、test-all.ps1 已有 AV 锁重试
- **releases/README.md 乱码恢复**：v2.40.0 及更早 98 行 GBK 损坏（U+FFFD 不可逆）从
  git 历史（v3.0.1 提交 7c53db8 干净版本）逐行重建，0 残留；版本索引历史行完整可读
- **前端性能体检**：大组件 memo 复查（Transcript/CostLibraryView/Message 已 memo，
  页面级组件 memo 收益有限不额外加）；唯一热点 = XlsxPreview Excel 网格全量渲染
  （maxRow×maxCol `<td>`），修复需虚拟滚动重构、收益/风险比低——按「先体检再决定」
  纪律记录待真实卡顿反馈
- **验证**：eslint **0 errors / 0 warnings**（366→0）、tsc 0 errors、vitest
  **652/652（124 文件）**零回归、Go 全量 **112/112 包**、filewatch 5 测试绿、
  绑定面 **497 方法**漂移 PASS、版本四处统一 3.3.0、wails build + 冒烟 /api/health 200

## v3.2.1「工作区内联编辑 · C5 文本文件直接编辑保存」(2026-08-26)
> v3.2.0 第二刀（蒸馏候选清单第 9 项收尾）：工作区文本文件在预览中直接编辑保存。
- **C5 工作区内联编辑（GaeaWriteFile）**：新绑定 `GaeaWriteFile(rel, content) error`
  （绑定面 497，+1）——安全四重校验：相对路径（拒绝绝对/`..` 穿越）+ 必须落在可写根
  （WriteRoots：工作区 + allow_write）内 + 文本扩展名白名单（md/txt/csv/json/toml/
  脚本/源码等 30 种）+ 内容 ≤2MB + 仅允许编辑已存在文件；**原子写**（同目录临时文件 +
  fsync + rename，失败保留原文件）。用户显式保存视为用户意图（非 agent 自动写，不走
  审批；agent 写仍受工具权限面约束）
- **FilePreview 编辑模式**（markdown/text 且未截断才可编辑——截断内容不完整，写回会
  丢数据）：标题栏「编辑」切换 → 等宽 textarea + 脏标记（琥珀色脉冲点）+ 保存状态机
  （保存中/失败可重试）+ **Ctrl/Cmd+S 保存** + 保存成功后自动重读预览；脏状态下退出
  编辑弹内联「放弃修改/继续编辑」确认条（不用 antd 静态弹窗，测试确定性强）
- **验证**：Go 全量测试绿（新增 TestGaeaWriteFile：正常写回 + 穿越/绝对路径/非文本/
  不存在/超大 五类拒绝 + 拒绝不改动原文件）；前端 tsc/eslint 0 errors、vitest
  **652/652（124 文件，+5：FilePreview 编辑模式 5 用例）**；绑定面 **497 方法**漂移
  PASS（+1：GaeaWriteFile）；wails build 发布版 + 冒烟 /api/health 200

## v3.2.0「任务可见性 · C1 任务实时输出 + C2 子代理活动行」(2026-08-26)
> v3.2.0 第一刀（蒸馏收尾第 3 轮，按候选清单推进顺序）：办公长任务与子代理的
> 「看得见在干什么」。后端零侵入扩展（tasks 输出环形缓冲 + stopping 结束态细分 +
> SubagentRunView 活动行字段），前端任务中心输出 dock + 分工面板活动行。
- **C1 任务实时输出（GaeaTaskOutput）**：tasks 包新增 `Progress.Output(line)` 输出
  环形缓冲（200 行 / 64KB 上限，超限截断标注，仅回放不消费游标）；三个消费者
  （价格抓取/批量抓取/语义索引）逐源逐批输出时间戳行；新绑定 `GaeaTaskOutput(taskID)
  → { tail, truncated }`（绑定面 496 方法）。任务中心：任务行可点击选中 → 底部共享
  输出 dock（pre 等宽回放、运行中 2s 轮询 + 自动尾随滚动、截断标注、可关闭）
- **C1 结束态细分（stopping）**：取消运行中任务先条件置 `stopping`（正在停止…，
  WHERE status='running' 防覆盖终态竞态）再传播取消，handler 退出后终态 cancelled；
  前端「停止中」琥珀色旋转徽标 + 取消按钮文案变化；重启续跑把 stopping 一并恢复为
  queued（取消途中崩溃不丢任务）
- **C2 子代理活动行（SubagentRunView + lastText/lastTool）**：`summarizeSubagentTranscript`
  从 transcript 尾部派生 最后 assistant 文本（截断 160 字）与 最后一次工具调用摘要
  （name + 结果首行，截断 80 字）；分工面板运行中卡片显示「正在：…」+「⚙ 工具」两行
  活动线（脉冲指示点，随 5s 轮询刷新）；父子拓扑按候选清单建议暂不做（meta 无父子
  记录，退化为活动行 + 扁平列表）
- **验证**：Go 全量测试绿（tasks 新增输出缓冲/stopping 竞态 2 用例 + app 活动行派生
  1 用例）；前端 tsc/eslint 0 errors（360 存量 warnings）、vitest **647/647（123 文件，
  +4：TaskCenter 3 + SubagentsPanel 活动行 1）**；绑定面 **496 方法**漂移 PASS（+1：
  GaeaTaskOutput）；wails build 发布版 + 冒烟 /api/health 200

## v3.1.1「造价数据库闭环补齐 · 测算项目 UI + 造价参考 + 复盘笔记 + 选区转对话」(2026-08-26)
> 承接 v3.1.0 发布后盘点：costproject/costref 后端与 15 个绑定已就绪但前端无入口，
> 本版补齐三个前端 UI + C4 选区转对话 + 仓储卫生。纪律延续：不做新板块、不堆功能。
- **测算项目 UI（新组件 CostProjectsView）**：造价数据库板块新增「测算项目」导航——
  左列项目列表（名称/类型/状态徽标/条目数/合计/版本数 + 新建/删除，级联）；右区详情：
  ① 项目信息编辑（名称/类型/规模/工艺/备注）；② 工程量清单表格（标题/单位/数量/单价
  行内编辑、失焦自动保存、金额=数量×单价实时计算、「引用成本库单价」搜索下拉回填
  title/unit/price/categoryPath/entryName、新增/删除行）；③ 保存版本（不可变 JSON 快照 +
  备注，列表按版本号倒序、点击查看快照明细表、可「恢复此版本」前端编排重建明细）；
  ④ **沉淀选中行回成本库**（勾选 → UPSERT cost_entries，成功后刷新库概览统计）。
  复用 v3.1.0 已就绪的 GaeaCostProject* / GaeaCostEstimate* 绑定，零后端改动
- **造价参考 UI（新组件 CostIndicatorsView）**：「造价参考」导航——按科目/按一级分类
  分组切换，表格展示 样本数/最小值/P25/中位数/均值/P75/最大值/单位（实时聚合不落表）；
  空态引导「保存版本或沉淀后自动成为对标样本」
- **复盘笔记 UI（新组件 CostNotesView）**：「复盘笔记」导航——搜索 + 状态过滤 + 新建/
  编辑弹窗（标题/结论/适用边界/风险提示/证据来源/可信度 高中低/复核状态 草稿·已确认/
  成本分类/项目类型/有效期至），卡片展示结论摘要 + 引用次数，删除经 Modal.confirm 确认；
  复用 GaeaCostNote* 绑定
- **板块导航同步**：board cost manifest Nav children 增补 测算项目/造价参考/复盘笔记
  （概览/成本条目/测算项目/造价参考/复盘笔记/价格源/价格仓库），与页面 MODULES 对齐
- **C4 选区转对话（新组件 SelectionToComposer，纯前端）**：办公板内选中任意正文
  （对话文本/文件预览/过程输出）→ 选区上方浮出「转为提问」浮动条 → 点击把选中文本
  以 `> 引用` 块插入输入框（可编辑后发送）；忽略输入框/文本域/下拉/弹窗内选区，
  不干扰既有交互；portal 渲染 + 视口边缘收敛
- **仓储卫生**：删除根目录临时脚本 `.go` / `.split.go`（历史 stage 复制脚本）与旧版
  `gaea.exe`（2026-08-14 残留产物）；releases/README.md 补 v3.0.8 索引行 + 修复
  v3.0.1/v3.0.0 两行历史乱码（其余更深历史乱码保留，源自历史编码事故）
- **验证**：前端 tsc 0 errors、eslint 0 errors（359 存量 warnings）、vitest **643/643
  （122 文件，+13：测算项目 4 / 复盘笔记 2 / 造价参考 3 / 选区转对话 4）**；Go
  build/vet 干净 + board/bindings 测试绿；绑定面 **495 方法**漂移 PASS（零新增绑定）；
  wails build 发布版 + 冒烟 /api/health 200

## v3.1.0「造价数据库 · 一级板块 + 办公蒸馏 + 死锁修复」(2026-08-26)
> 主线 = 参照 zaojia-database 蒸馏造价数据库并提升为一级板块：综合单价=一级、
> 人材机=二级组成；按用户定调「数据库就是数据库」收口——测算引用/对标由办公
> agent 工具承担，造价数据库只做数据沉淀与管理。并行落地办公蒸馏
> （dsh-better-sidebar 右侧面板 C3/C6/C7 + 文件工作台资源管理器）与 UI 紧凑化；
> 修复办公板块初始化死锁。
- **一级板块「造价数据库」**：成本库从记忆中枢二级分类提升为独立板块（board
  cost，`CostLibraryPage`，MenuOrder 5，AccountBookOutlined，导航：概览/成本条目/
  价格源/价格仓库），复用 CostB 门面；记忆中枢成本二级入口移除（避免双入口，
  记忆图谱节点仍保留琥珀色）
- **架构定稿（市场调研驱动）**：专项调研陕西蜘蛛网工程成本平台，确认其公开内容
  全部为「综合单价分析」（人工费+材料费+机械费+管理/利润/税金→综合单价），与
  市政手册同构；三个决策点定稿：记录=综合单价子目、资源库层保留不强制关联、
  费率入库仅展示追溯。调研文档
  `docs/market-research-2026-08-cost-architecture-zonghe-danjia.md`
- **数据模型（SchemaV9/V10/V12/V13）**：`cost_entries` 增加 地区（region）/价格
  时间期数（price_date）/价格口径（price_type）/有效期（valid_until）/导入原始
  行号（source_row），`cost_price_history` 同步记录地区与口径（价格快照可追溯
  「哪个地区、什么口径、哪一期」）；再增加 人工费/材料费/机械费 三个合计与
  管理费/利润/垫资/税率（仅展示不参与计算）；新建 `cost_entry_components` 组成
  行表（kind/title/unit/quantity/price/amount/note/sort）。`cost.Store` 的
  Save/Get/Delete/Search 与导入事务写库全链路接通，Save 组成行整组替换；
  `Summary` 增 `componentCount` 支撑面板统计
- **默认分类树重构**：综合单价 → 专业（道路/交通/绿化/电力/给水/暖气/雨污/照明/
  其他）→ 分部；人材机不再平级成类，资源库层由价格源模块承载
- **《市政成本测算手册》整本导入**：任一 sheet 含「综合单价分析+人工费/材料费/
  机械费」表头即触发专有解析（多专业表 → 分部行 → 子目行）；「综合单价分析」
  文本解析为人材机组成行（段头切 kind、行尾金额、原始表达式入 note 保留损耗系数），
  同名子目按项目特征片段去重；实测 8 张专业表 234 条全部命中，通用报价单导入不受影响
- **测算项目与沉淀闭环（新包 costproject，SchemaV10）**：`cost_projects` /
  `cost_estimate_items` / `cost_estimate_versions` 三表——测算项目容器（类型/规模/
  工艺/状态）+ 明细行（引用成本库单价或手动填价，数量×单价自动算金额）+ 不可变
  版本快照（回看/对比/恢复）；「沉淀」把明细行 UPSERT 回 cost_entries，
  「沉淀即调用」闭环；App 绑定 GaeaCostProjectSave/List/Get、GaeaCostEstimate* 系列
- **造价参考与复盘笔记（新包 costref）**：对「已保存版本/已沉淀」测算项目明细行
  实时聚合 样本数/极值/P25/P75/中位数/均值（指标不落表避免双写），供下次报价对标；
  复盘笔记沉淀「判断」（结论/适用边界/风险/证据/可信度/有效期/复核状态 + 引用计数）；
  新增 `cost_indicators` 办公 agent 工具（只读聚合，测算前对标与引用依据）
- **成本库数据自愈（cost/repair.go）**：修复历史遗留平铺分类路径——1420 条里 201 条
  category_path 在分类树上无法解析（树上看不到、统计对不上），规则引擎幂等映射回
  合法路径 + 从来源字符串保守回填地区/期数（不臆造）；`Store.Open` 自动执行防再次漂移
- **造价数据库面板重设计**：概览改为 库规模 hero（条目/专业/分部/组成行 + 累计
  人材机构成占比条）+ 数据健康（缺单价/草稿/引用完备度）+ 最近更新（行内人材机
  mini 条）+ 空库三步引导 + 骨架屏；模块导航由圆角胶囊改分段下划线；条目列表加
  人材机占比条、表格加「组成」列；沿用 v3 玻璃面板/辉光卡设计语言
- **办公联动收口**：`cost_search`/`cost_save`/`cost_indicators` 三个 agent 工具
  承载测算引用、单价沉淀与分位数对标；交付物面板「沉淀到成本库」一键生成
  `cost_save` 指令；`cost_save` 与测算沉淀更新既有子目时保留人材机组成/费率，
  避免改价抹掉二级明细
- **办公蒸馏（dsh-better-sidebar，2026-08-20/26 两轮）**：C3 会话级右侧面板布局
  持久化（`gaea.rightPanel.v1:<sessionKey>`）+ C6 运行域活动角标（useRunningBadge
  计数徽标，99+ 封顶）+ C7 预览队列 chip 化（点击切换/×关闭/中键关闭 VS Code 语义）；
  FileTree → 资源管理器（行悬浮 @引用 / 右键菜单：预览·外部打开·在文件夹中显示·
  复制相对路径 / 展开态按 cwd 持久化 / 树内搜索接 GaeaFileSearch），纯前端零后端；
  删除完成轮「大过程卡」（Transcript 统一交替语义，正文独立显示）；GoalCard/TodoCard
  默认折叠紧凑化（用户：太占地方）；修复 Tailwind v4 `max-w-[--maxw]` 方括号语法
  不生成 CSS → 统一 `max-w-(--maxw)` 括号语法（四处卡片宽度对齐输入框）
- **成本库入口接线（用户决策，2026-08-26）**：办公右侧面板「文件」组新增「成本库」
  子 Tab（CostLibraryPanel：条目列表/价格源/价格仓库三态 + 一键插入输入框），
  4 主 Tab 收敛不变；workspaceTabs 单源声明 + App 渲染分支 + 测试断言同步
- **修复办公板块初始化死锁（工作空间/新建会话不可用的根因）**：`GaeaInit` 持
  `ga.mu` 初始化时会走 `resumeLastSession → syncGoalForSession`，而后者内部经
  `gaeaCtrl()` 对同一把非重入互斥锁二次加锁，导致工作区存在会话时办公板块首次
  打开即永久卡死（界面停在「连接中…」、侧栏看不到任何项目/会话、无法新建会话、
  无法切换工作空间）。修复：`syncGoalForSession` 改为显式接收控制器，不再内部
  取锁；新增两个回归测试（持锁上下文直接调用 + 工作区含会话的完整 `GaeaInit`
  场景），并让 `persistWorkspaceLocked` 同步更新磁盘上的
  `sandbox.workspace_root`，避免切换工作空间后配置文件残留旧工作区路径。
- **决策：V4.0 dsh化 验证失败，正式废弃**（2026-08-26）：曾把 gaea 改造成 DSH 插件
  体系（独立工作空间 `C:\AI\gaea-v4`），用户验证后判定失败；删除工作空间内 V4.0
  文档（v4-blueprint 蓝图 / phase4 两份实施计划 / CHANGELOG v4.0.0 章节），
  `C:\AI\gaea-v4` 与 `~/.dsh*` 工作空间外文件一律不动；继续 V3 迭代
- **验证**：`go build ./...` + vet 干净 + 全量 `go test` 通过（filewatch 首跑为
  环境抖动，单独重跑全绿）；前端 `tsc -b` 0 errors、eslint 0 errors、vitest
  **630/630（118 文件）**；绑定面 **495 方法**漂移检查 PASS；`wails build` 发布版
  成功 + 冒烟测试（/api/health 200），产物 `build/bin/gaea.exe` 复制到 releases

## v3.0.8「办公板块 · 表格可交付 + 会话产物打包 + 多智能体分工」(2026-08-17)
> 按调研 docs/market-research-2026-08-office-table-agent-and-package.md 落地 P0 三项
> （表格选中区域→一键图表、会话产物一键打包 Zip、产物缩略图增强）+ P2 首项
> （多智能体分工可见），并按用户决策做界面收敛（不做 PPT、聚焦 Word/Excel、
> 不堆功能）：右侧面板 Tab 收敛为 4 个主标签、Excel 工具栏按上下文收敛。
- **P0-1 会话产物一键打包 Zip**：会话产物面板新增「打包下载」→
  `.gaea/exports/gaea-会话产物-<stamp>.zip`；只接受工作区相对路径（拒绝绝对路径/
  `..` 穿越）、缺失/目录静默跳过、zip 内保留相对路径防同名覆盖；后端
  `GaeaZipDeliverables` + 前端 Archive 按钮（对标 Kimi 工作空间 / WorkBuddy）
- **P0-2 表格「选中区域 → 一键图表」**：XlsxPreview「图表 ▾」菜单（柱状/折线/饼图
  PNG + 图表→Word/→PPT）——选中区域/单单元格/自动前两列数据提取 → matplotlib PNG
  → 预览队列；后端 `GaeaXlsxChart`（对标千问表格 Agent / ChatExcel）
- **P1 产物缩略图增强**：FileThumb 升级为内容缩略图（xlsx 迷你表格 / md 文本摘要 /
  图片 dataURL，失败回退类型图标），接入交付卡与会话产物面板，零新后端绑定
- **P2-1 多智能体分工可见**：右侧「运行」组新增「分工」子面板（SubagentsPanel）——
  子代理状态徽标/任务摘要/模型/工具范围/耗时/回答摘要，运行中 5 秒轮询；后端
  `GaeaSubagentRuns`（对标 WorkSwarm 蜂群 / QClaw V2 多 Agent）
- **右侧面板 Tab 收敛为 4 个主标签**：文件（文件/资料）/ 成果（产物/变更）/ 运行
  （任务/分工）/ 分析（统计），第一级组 + 第二级子 Tab；workspaceTabs 重构为
  WORKSPACE_GROUPS 分组清单，App.tsx 渲染分支与命令面板零改动
- **Excel 编辑器工具栏收敛**：10 个常驻按钮 → 行操作（选中才显示）+ 重算公式 +
  图表 ▾ 下拉；选中单元格布局重排为「公式栏在上、AI 编辑（单行紧凑）在下」，
  两个输入框逻辑分层；预设点击回填指令并有激活态
- **验证**：Go `internal/app` 全量 ok（12 个新测试）；前端 tsc 0 errors、vitest
  **605 通过**（新增 18 用例）、vite build 通过；绑定面 **480 方法**漂移检查 PASS；
  版本四处统一 3.0.8；wails build 发布版 + 烟雾测试（/api/health 200）

## v3.0.7「办公板块文件交互体验 · 内置 prompt 模板兜底」(2026-08-17)
> 调研 docs/2026-08-16-office-file-interaction-research.md 的 P0+P1+P2（纯前端部分）
> 全部落地：文件从「附件/路径文本」升级为一等公民交互对象——非图片附件 chip、
> 行内文件 chip 视觉统一、最近文件快捷区、多文件预览队列、产物版本时间线、
> 大工具输出有界预览、附件上下文占用透明化。
- **P0-1 非图片附件 chip 化**：拖入/选择的 docx/xlsx/pdf 等非图片文件不再注入裸
  `@路径` 文本，而是进 Composer 附件栏渲染为「图标+文件名+扩展名 badge」chip
  （点击预览/移除）；提交仍按附件数组统一注入 `@路径`，行为零变化
- **P0-2 行内文件 chip 视觉统一**：新 FileChip 组件 + lib/fileBadge 扩展名单源
  （BADGE_EXTS 从 FileMenu 收敛）；FileLinkText / Markdown 文件链接 / 流式尾部
  htmlFileLinks 全部升级为「图标+文件名+badge」，与 @ 菜单同视觉
- **P0-3 最近文件快捷区**：lib/recentFiles localStorage 单源（@ 引用与预览共用，
  去重置顶 20 条）；文件面板顶部「最近」快捷条一键回看，预览过的文件自动记录
- **P1-1 多文件预览队列**：preview store 扩展 previewList/index/navPreview 单源
  （兼容 previewFile；已在队列去重跳转、上限 50、close 清空）；App 预览状态全部
  改由 store 驱动（消除局部副本双写不一致）；预览底部 ← index/total → 导航条
- **P1-2 产物版本时间线**：sessionDeliverables 记录同一文件会话内出现次数
  （versions），产物面板对多次更新的文件显示「vN」徽标（对标 Hermes 版本步进器）
- **P2-2 大工具输出有界预览**：boundedOutput 纯函数（>60 行折叠为头部+「已折叠
  N 行」提示）；ToolCard「展开全部 N 行/收起输出」开关，超长输出不再撑爆卡片
  （对标 QwenPaw 超长输出折叠）
- **P2-4 附件上下文占用透明化**：附件 chip 显示「4.0 KB」占用（图片 base64 估算 /
  文件 File.size / PickFiles 后端 size 补前端类型），title 注明「附件占用（进入
  上下文的体量）」（对标 QwenPaw context-usage）
- **内置 prompt 模板兜底（SetPromptFS）**：main.go go:embed 内置 prompts/ 模板，
  exe 单文件分发（旁边无 prompts/ 目录）时由内置模板兜底，磁盘 prompts/ 仍优先
  （开发期直接改模板生效）；prompt 引擎 6 个新测试
- **验证**：vitest 587 通过（新增办公文件交互 40+ 用例）、绑定面 477 方法漂移
  PASS、wails build 发布版 35.2MB、冒烟 /api/health 200

## v3.0.6「编程板块工作台 · 办公会话回退分叉 · 顶栏工具栏迁移」(2026-08-16)
> 办公板块会话能力闭环（回退/分叉/回退点 + 右侧 Tab 清单化 + 会话统计回填 +
> mock 场景补全）与编程板块桌面内嵌工作台（iframe 内嵌 DeepSeek Harness Web +
> 启动引导 + 运行中工具栏移入顶栏）。
- **办公板块 UI+功能优化**：ProcessCard 状态四态 / 思考深度三档接线 / useDrawers
  收敛 / 恢复会话状态还原 / 右侧 tabs 激活态修复
- **会话回退/分叉/回退点**：后端实现会话 rewind 链路（此前为前端空实现）——
  会话级派生统计绑定 GaeaSessionStats + 前端恢复回填，根治恢复会话成本展示不完整
- **右侧面板 Tab 体系清单化**：workspaceTabs 单源声明 + WorkspaceTabs 组件 +
  装配层测试；同步清理死绑定（移除 GaeaWorkspaceChanges/GaeaSelectTab/GaeaTabMeta）
- **mock 场景补全**：审批/提问/压缩卡三类事件流 + initBridge `?mock=` 优先——
  浏览器离线开发可用
- **命令面板任务模板 / 领域色单源 / 模板缓存去重**：办公板块体验三件套
- **编程板块桌面内嵌工作台**：DeepSeek Harness Web（http://127.0.0.1:3080）以
  iframe 内嵌桌面窗口，运行中显示工作台工具栏（运行徽标/时长/URL/刷新/浏览器
  打开/停止），未运行时提供一键启动引导视图
- **前置条件真实检查**：启动引导从静态使用说明升级为逐项清单——新增
  GetProgrammingWebPreflight（harness 目录有效 / pnpm 可用 / node_modules 已装 /
  apps/web/dist 构建就绪 / 端口 3080 空闲 + all_ready 汇总），每项绿/红呈现 + 修复
  提示 + 「重新检查」按钮；未全部就绪时启动按钮禁用，杜绝「点完没反应」
- **启动日志可查看**：新增 ProgrammingWebLogTail（自启日志尾部，n 钳制 [1,200]），
  启动视图新增可展开日志面板（读取/刷新/空态提示），排障不再需要去临时目录翻文件
- **运行时长**：GetProgrammingWebStatus 新增 uptime_s；运行中工具栏显示
  「已运行 X 小时 Y 分」芯片，外部实例显示警示芯片（不可停止，防误杀）
- **顶栏工具栏迁移**：运行中工具栏（「Harness Web 运行中」徽标 / URL / 刷新 /
  浏览器打开 / 停止）整体移入顶栏 v3-strip（portal 进 v3-prog-host 宿主，与聊天
  模式条同款模式），仅编程板块激活时自动显示、其他板块自动隐藏；iframe 独占
  工作区全高展示；宿主缺失兜底保持原布局；新增 portal 渲染用例
- **启动动画视图**：点击「启动」到端口就绪之间显示启动动画——纯 CSS 双虚线环反向
  旋转 + 发光 orb 呼吸脉冲（WebView2 下不依赖 rAF，与粒子星云回退同理）+ 「已等待
  X 秒」实时计时 + 内嵌日志折叠面板；启动失败自动展开日志并回到引导视图
- **数据源 seam 化**：ProgrammingPage 从 wailsjsCompat 直调改为 bridge app seam
  （§5.3 前端侧模式）——Wails 原生走门面代理、浏览器 mock 走 makeMockApp，两种
  环境同一套代码；mock 的 StartProgrammingWeb 延迟 3s 报错，保留启动动画演示窗口
- **修复 useVoiceChat cleanup 崩溃**：wailsjsCompat 直调在 mock/未就绪时同步 throw，
  effect cleanup 的 App.VoiceStop().catch() 拦不住同步异常导致首页渲染崩（错误边界
  兜底）——两处调用加 try/catch，浏览器 mock 模式恢复可用
- **后端可测性**：端口探测/tasklist/taskkill/LookPath/cmd.Start/日志路径/等待超时
  全部经 probe* 探针注入；新增 programming_web_test.go 16 用例（状态归属/幂等启动/
  外部占用守卫/目录与 pnpm 缺失/超时/停止三态/日志尾读/前置检查全绿与全红），
  零外部依赖（不碰真实 3080 与进程）
- **绑定面**：CoreB 468→470 方法（GetProgrammingWebPreflight / ProgrammingWebLogTail），
  绑定完备性与漂移检查 PASS（476 方法一致）；wailsjs 绑定已重新生成
- 验证：Go 全量 internal/app ok（首轮 TestGaeaPriceFetchAllTask UNIQUE 冲突为偶发
  环境抖动——与 vitest 全量并行时的时序/资源竞争，单独重跑与全量重跑均通过）；
  前端 tsc 0 errors、eslint 0 errors、vitest 544 通过（编程板块 12 用例，新增
  顶栏 portal 渲染 1 用例）、vite build 通过；浏览器 mock 实机走查
  （引导视图 → 启动动画 → 失败展开日志 → 前置条件全绿）；wails build 发布版
  通过 + 冒烟测试（/api/health 200）

## v3.0.5「首页任务指挥中心改版」(2026-08-16)
> 参照 DeepSeek 首页风格重做桌面端首页：Hero 左文右卡 + 透明 AI 状态细条 +
> 「全部模块」办公大卡 + 8 卡 4×2 网格；修复设置卡被底部信息条遮挡、编程板块
> 在首页不可见；语音晶核动效经 WebView2 实机验证后取消粒子星云，恢复发光球。
- **Hero 左文右卡**：公告 pill（在线呼吸点 + 悬停箭头）+ 大标题（clamp 30-46px
  balance）+ 副标题 + 双行动卡「开始创作 / 和 gaea 对话」——整卡可点、黑色描边、
  悬停发光 + 图标放大 + 箭头滑入（位移 ≤2px）；内容收进 1240px 居中容器
- **右侧 AI 视觉卡**：深色渐变底 + 细网格纹理（径向渐隐）+ 右上星云流光；左侧
  语音晶核（呼吸环 + 静态发光球）+ 「AI 内核」状态标题 + 活跃模型 pill
  （本地/云端模型名 + 成功绿点）
- **AI 状态细条**：4 列透明无边框信息行（活跃模型/已启用引擎/资源占用/项目写作
  进度），不再呈现卡片感，保留图标发光与等宽数字
- **全部模块 Bento 改版**：新增「全部模块」分区标题 + 副文案；左侧办公大卡
  （280px 固定列、2×2 视觉高度、与右网格等高），右侧其余 8 卡 4×2 网格——
  角色库/设置与其他卡同排，不再单独成行；模块区 `flex:0 0 auto` 不被压缩，
  根治「设置卡被下方横框挡住」
- **编程板块回归**：PageRegistry 补注册 ProgrammingPage（此前遗漏导致首页编程
  卡不可见），编程卡恢复显示
- **语音晶核动效回退**：canvas 粒子星云在 WebView2 下 requestAnimationFrame 被
  节流只剩首帧（呈现为静止图片），setTimeout 驱动实测仍不稳定，最终取消粒子
  星云组件，恢复呼吸环 + 静态发光球（纯 CSS 动画，不再依赖 JS 循环）
- **响应式**：1120px 以下 Hero 转单列；920px 以下办公大卡上移整行、其余卡 2×2；
  720px 行动卡单列、视觉卡纵向；520px 网格单列、状态单列
- 验证：前端 tsc 0 errors、vite build 通过；wails build 桌面版 35.2MB；冒烟测试
  （HTTP 桥接 /api/health 200）通过；发布 gaea-v3.0.5.exe + SHA256SUMS +
  v3.0.5.md + CHANGELOG-v3.0.5.txt + README 索引更新（v3.0.0 归档）

## v3.0.4「办公能力加强 · 小说阅读体验重构 · 角色库闭环」(2026-08-16)
> 本会话两大主线：① 办公板块——任务目标（需求→验收）升级为目标工作流，目标卡/
> 待办卡拆分重设计，文件交互市场调研；② 小说板块——阅读体验 P0+P1 全量落地
> （排版/主题/书签/划线/AI 伴读/朗读同步/全文搜索）、书架网格统一与成品小说导入、
> 角色库补齐/加入项目闭环修复、导出合并进阅读面板、项目上下文不跨板块残留。
- **验收清单**：任务目标支持逐条验收标准（添加/勾选/双击编辑/删除），全部勾选自动
  推导为「已验收」，一键验收 = 全选/全不选
- **目标卡与待办卡拆分**：底部提示区从单一抽屉拆为两张独立卡片——任务目标卡
  （会话锚点，始终展开：目标文本 + 验收清单 + 自动追踪开关）与待办卡（todo_write
  提取，**默认折叠**，折叠态显示当前任务摘要）
- **待办卡重设计**：展开后按「未完成在前 / 已完成收尾」分组（已完成项加分组头、
  置灰删除线）、阶段行小标题化、当前任务高亮 + 左缘光条、进度条全圆角令牌渐变、
  全部完成态全绿
- **自动追踪（治理下自主）**：会话级「自动追踪」开关开启后，恢复会话时把任务目标
  写入 agent goal gate——回合结束未达标会自动继续工作（由模型判定验收），关闭或
  新建会话时清空，避免跨会话残留；默认关闭，不改变既有行为
- **兼容性**：RequirementView 增量字段（items/autoPursue），旧数据零迁移；文本变更
  视为新目标重置清单、自动追踪保留
- **后端**：新增 GaeaAddRequirementItem / GaeaSetRequirementItem / GaeaRemoveRequirementItem
  / GaeaSetRequirementItemDone / GaeaSetRequirementAutoPursue 5 个绑定（office 门面
  135→140 方法），绑定漂移检查同步通过
- **测试**：Go +2 用例（验收清单闭环、goal gate 接线、新会话清目标），前端 GoalCard
  +7 / TodoCard +4 用例（勾选/增删/双击编辑/自动追踪/验收态/默认折叠/分组/摘要）
- 验证：internal/app 全量 Go 测试通过；前端 tsc 0 errors、eslint 0 errors、vitest
  519 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：角色库文件交互修复（2026-08-16）
- **补齐时随机五维人格**：一键补齐/随机补齐现在会按补全后的性格生成五维人格
  （T/I/S/O/R，0-100），不再停留在默认 50/50/50/50/50；编辑器五维人格骰子改为
  AI 按性格随机（后端不可用时本地兜底），「全部随机」同样包含五维人格
- **加入项目可选小说**：「加入项目」从「写入当前打开的小说」改为弹窗选择任意小说
  （书架项目列表，含当前项目快捷项），不再需要先打开目标小说；新增
  CharacterAssociateTo 绑定（charlib 门面 15→16 方法），后端校验目标必须在书架
  目录内且为有效项目
- **加入项目后小说面板可见**：修复「已加入项目的角色在小说角色面板看不见」——
  加入时同步物化进目标小说 characters.json（按 ID 幂等合入、保留项目内既有角色/
  组织/关系）；小说面板读取角色列表时自愈，旧版本只写关联表未物化的角色自动补齐
- **测试**：Go +2 用例（指定项目关联 + 目录校验、小说面板读取自愈物化），前端
  CharacterLibEditor +2 用例（dims 骰子调用 generateRandom(dims)、补齐默认五维计入填充数）
- 验证：internal/app 全量 Go 测试通过；前端 tsc 0 errors、eslint 0 errors、vitest
  521 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读体验优化（2026-08-16）
- **排版可调**：阅读模式新增排版面板（Aa）——字号 A−/A+（14–24）、行距
  紧凑/标准/宽松、版宽 窄/标准/铺满，偏好全局持久化，默认铺满（延续上一轮修复）
- **段落排版**：正文按空行分段、每段首行缩进 2em、两端对齐；章节标题居中大字
- **进度与位置记忆**：阅读区顶部细进度条显示本章阅读百分比；每章滚动位置按章记忆，
  切章/退出/重开自动恢复上次位置
- **键盘导航**：阅读模式 ←/→ 上一章/下一章（Esc 退出阅读，F11 专注不变）
- **测试**：新增 readingSettings 3 用例（默认值/往返/非法回退）；前端 tsc 0 errors、
  eslint 0 errors、vitest 524 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读 P0（主题 / 书签 / 自动滚屏，2026-08-16）
- **阅读主题**：Aa 面板新增 主题（跟随/米黄/护眼绿/夜间）+ 亮度滑杆（70-120%），
  阅读区背景与文字色按主题切换，全局持久化
- **书签**：阅读栏图钉按钮 = 书签面板；滚动到目标位置「＋ 此处」添加（带章节内
  百分比 + 段落摘录），列表点击跳回、可删除；按项目持久化
- **自动滚屏**：阅读栏播放按钮开启/暂停自动滚动（40ms 步进），速度在 Aa 面板
  慢 1-5 快 五档可调；滚轮手动干预自动暂停；章节到底自动停止
- **测试**：readingSettings 扩展到 主题/亮度/滚屏（夹紧 + 回退），新增
  readingBookmarks 3 用例（往返/损坏数据/空项目）；前端 tsc 0 errors、eslint 0
  errors、vitest 527 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读 P1-划线/高亮/想法（2026-08-16）
- **划词工具条**：阅读模式拖动选中正文 → 选区上方浮动工具条（黄/绿/蓝/粉四色
  高亮 + 「想法」），仅限单段落内选择，滚动自动收起
- **高亮回渲染**：按摘录文本在段落中重新定位（支持多色、重叠冲突跳过），点击
  高亮可编辑/删除；章节内容变更后失效的摘录自然不显示
- **本章批注面板**：阅读栏高亮按钮列出本章全部划线/想法（颜色点 + 摘录 + 想法
  标记），点击滚动定位，可删除
- **想法弹窗**：摘录引用 + 多行想法编辑，保存后随高亮展示
- **测试**：新增 readingAnnotations 3 用例（往返/颜色映射/损坏数据过滤）；前端
  tsc 0 errors、eslint 0 errors、vitest 530 通过（2 个既有 launcher.test.ts 基线
  失败，零回归）

### 追加：小说阅读 P1-AI 伴读（章节摘要 + 划线问书，2026-08-16）
- **AI 摘要**：章节正文顶部新增「AI 摘要」折叠卡——展开即生成 3-5 条要点，按章
  缓存（同章重复展开不重调），失败可重试；只使用本章本地文本
- **划线问书**：划词工具条新增「问书」——针对摘选原文提问，弹窗内展示摘录引用 +
  问题输入 + AI 回答（可反复提问），答案基于原文、信息不足时明确说明
- **后端**：新增 NovelReadingAsk 绑定（novel 门面 67→68 方法）：summary/ask 两类，
  正文按 rune 截断 9000 字，模型走 novel 功能路由（GetFeatureModel），内联提示词
  不新增模板文件；绑定漂移检查 469 个方法一致
- **测试**：Go +1 用例（守卫分支：未初始化/空正文/空问题/空摘选/未知类型）；
  前端 tsc 0 errors、eslint 0 errors、vitest 530 通过（2 个既有 launcher.test.ts
  基线失败，零回归）

### 追加：小说阅读 P1-朗读同步 + 全文搜索（2026-08-16）
- **朗读同步**：阅读页脚新增听书入口（复用 TTSPlayer），朗读时逐句高亮当前句
  （蓝色）并平滑滚动居中跟随；停止/重播自动清除高亮；后端 tts-stream 已携带
  sentence 文本，前端按句在段落 DOM 中回定位（支持跨高亮标记）
- **全文搜索**：阅读栏新增放大镜——输入即搜全书（标题 + 正文，大小写不敏感），
  一章最多一个命中、正文取首次出现的上下文片段（±40 字）；点击结果打开目标
  章节并切到阅读模式，正文渲染后自动定位并黄色高亮首个命中
- **后端**：新增 NovelSearch 绑定（novel 门面 68→69 方法）：按大纲顺序扫描章节
  （分支章节走 ReadChapterBranch），上限 100 条、损坏章节跳过；绑定漂移检查
  470 个方法一致
- **测试**：Go +1 用例（无项目/大纲未初始化/空查询守卫 + 片段截取）；前端
  tsc 0 errors、eslint 0 errors（TTSPlayer 既有 ts-ignore 警告保留）、vitest 530
  通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：项目上下文不再跨板块残留（2026-08-16）
- **修复**：打开小说项目后切到绘梦/办公/聊天等其他板块，顶栏面包屑与底栏遥测
  仍显示小说标题/进度/章数/字数的残留问题
- **规则**：项目上下文（标题、写作进度、章数、字数）只在「项目锚点板块」
  （manifest.breadcrumb.anchorTo = 小说）显示；其他板块顶栏只显示当前板块名，
  底栏保留引擎/CPU/内存/GPU 遥测但不显示小说信息；小说板块内行为不变
- 验证：前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530 通过
  （2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：导出标签页合并进阅读面板（2026-08-16）
- **删除**：小说板块「导出」标签页（子导航/类型/懒加载全部移除），原
  ExportPage.tsx 删除，导出逻辑抽为可复用组件 ExportPanel
- **合并**：导出入口移到阅读面板页脚（导出图标按钮）——点击弹出「导出小说」
  弹窗，一键导出 TXT + Markdown + EPUB 到小说目录 export/ 文件夹，结果列表
  与空态保留；旧 localStorage 里残留的 export 激活值自动回退首页
- **同步清理**：小说板块清单 NOVEL_NAV 移除 export 子项；NovelInspector 的
  「导出」说明区移除
- 验证：前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530 通过
  （2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：书架网格统一 + 导入成品小说（2026-08-16）
- **书架网格统一**：移除「最近打开」首卡双列放大（hero）效果，所有卡片固定
  等高（288px），封面条/信息区/操作区对齐，不再大大小小；封面与信息布局不变
- **导入成品小说**：书架工具条与空态新增「导入小说」——原生文件选择
  （TXT / Markdown / EPUB），弹窗填写书名/题材/文风，确认后后端解析章节、
  新建项目（outline + chapters/）并自动打开；按「第X章/Chapter N/序章/楔子/
  Markdown 标题」切分章节，TXT 支持 GBK/GB18030 编码兜底，EPUB 按 spine 顺序
  读取正文（剥离 HTML），无章节标记时归为单章「全文」
- **后端**：新增 ImportNovelBook 绑定（novel 门面 69→70 方法），返回
  NovelImportResult（路径/书名/章节数/字数）；同名项目目录自动加序号避免覆盖；
  绑定漂移检查 471 个方法一致
- **测试**：Go +4 用例（章节切分/无标记单章/GBK 解码/目录名清洗/导入端到端/
  格式守卫）；前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530
  通过（2 个既有 launcher.test.ts 基线失败，零回归）

## v3.0.3「小说/绘梦/模型中心/角色剧照体验修复」(2026-08-16)
> 迭代发布：设定页手动应用 AI 输出、创作字号调节与默认 story-deslop、剧情后台构思；
> 模型中心右侧统计与资源修复（内存采集/CPU/AMD GPU/趋势时间轴）；绘梦模板库重设计
> （合并 herdsman 12 类共 231 个模板）+ ComfyUI 实时进度 + 单图铺满；角色剧照远程 URL
> 本地化防裂图。详见 releases/v3.0.3.md。
- **小说**：设定 Agent 回复新增「应用到设定」手动按钮（代码块提取兜底）；创作面板
  字号 A−/A+ 调节（12–24px 记忆）；写作技能默认 story-deslop；剧情构思改后台执行、
  完成后弹窗确认
- **模型中心**：修复右侧「统计与资源」——GlobalMemoryStatusEx 结构尺寸错误导致内存恒 0、
  CPU 短间隔轮询清零、AMD/Intel 显存识别（跳过虚拟显示器）、趋势时间轴（今日取当天、
  7/30 天按天聚合、标签修正）、加载失败可见 + 重试 + 30s 常驻刷新
- **绘梦**：模板库重设计（内置 7 类 + herdsman 12 类 = 19 类 231 个，scripts/
  gen-herdsman-templates.mjs 生成，参考 herdsman「灵感示例」卡片与分类页签）；
  ComfyUI WebSocket 实时生成进度（百分比 + 当前节点中文名）；单张结果铺满画布
- **角色剧照**：远程剧照（xAI 临时图）保存时下载本地 portraits 目录 + 启动迁移历史
  远程 URL；助手记录与小说项目角色同步；前端裂图占位兜底
- 验证：Go 全量测试通过；前端 vitest 507/509（2 个既有 launcher.test.ts 基线失败）；
  tsc 0 errors、eslint 干净；wails build 发布构建通过，版本三处统一 3.0.3。

## v3.0.1「小说板块 UX/UI 重构 · 版本漂移修复」(2026-08-15)
> 用户指令「对 gaea 的小说板块 UX/UI 进行重构」——ui-ux-pro-max skill 驱动，保持 6-tab 信息架构，
> 重构书架/阅读/创作/控制台的视觉与交互。详见 releases/v3.0.1.md。
- **书架重构**：卡片重设计（题材→令牌 tone 封面渐变条 + 阅读进度条 + 标签/统计/相对时间）；
  搜索 + 排序工具条（最近打开/总字数/章节数/书名）；「继续阅读」主操作（新增 `novel:goto-tab`
  事件联动阅读 tab）；空态/无结果态；首本宽屏 span 2 锚点；hover 位移收敛 ≤2px
- **阅读页重构**：三层堆叠 chrome（tab/信息/工具栏）收敛为单条 `.novel-chrome`；新增阅读模式
  （居中限宽 46rem 衬线排版 17px×2 + 场景分隔符 + 页脚前后章导航）；编辑/阅读分离，
  F11 专注模式 + Esc 逐级退出
- **创作页令牌化**：EditorPanel 状态 Tag 与生成进度条统一 `--gaea-glow`（novel-gen-progress-*）；
  分支色板令牌化（usePlotBranch toneColors → 语义令牌；BranchSelector/NextChapter/PlotBranch/
  NewCharacters 选中态改 `--color-primary/-warning` 派生）
- **AI 控制台面板化**：内联样式 → v3 玻璃面板（`.ai-console-*`），字号 9-10px → 12px、
  antd Tag 预设色 → 令牌 Tag、FAB 30px 不与轨道条重叠、条目键盘可达
- **令牌化清理**：NovelSettingPage 导入/导出、ChapterEditor 重试条与右键菜单、NovelInspector 保存态、
  OrganizationEditModal、RelationGraph 画布色板（挂载时 getComputedStyle 解析令牌）；
  novel-workspace.css +566 行重构层（零硬编码 hex），新元素进 reduced-motion 降级
- **版本漂移修复**：app_info.go `AppVersion` 停在 2.40.0（v3.0.0 发布漏改）——
  sync-version.ps1 统一三处（app_info.go / wails.json / versioninfo.rc）→ 3.0.1，frontend package.json 同步
- **基建**：vite.config.ts `/api` 代理支持 `GAEA_PROXY_PORT` 环境变量（本地桥接端口冲突可指路）
- 验证：tsc 0 errors + vitest 455 通过（2 个既有失败在 launcher.test.ts，改动前基线相同）+
  `wails build` 重建通过 + `scripts/smoke.ps1` 冒烟通过 + 浏览器端到端实测（书架→大纲→阅读模式→
  创作→AI 控制台）。发布 gaea-v3.0.1.exe。

## v3.0.0「星枢 Constellation OS · UI 革命性重设计」(2026-08-15)
> 用户指令「革命性重设计整个 UI，适配 V3.0，面板布局可调整」——ui-ux-pro-max skill 驱动 +
> 12 个子代理并行实施。设计规格见 `docs/2026-08-15-gaea3-ui-constellation-os.md`，
> 令牌见 `design-system/gaea/MASTER.md`（v2）。
- **壳层革命**：顶栏横向菜单 → 左侧**指挥轨道**（icon dock，hover 展开标签、键盘方向键导航、
  激活项 = 主色容器 + 左缘光条 + 呼吸 orb）+ 顶部**轨道条**（面包屑 / 模型 pill / 主题点 / ⌘K 搜索 / 设置）
  + 底部**遥测轨道**（CPU/内存/GPU 实时面积 sparkline + 引擎 pods + 写作进度，可折叠展开）；
  快捷键升级 Ctrl+1~9（MainLayout.tsx + v3/foundation.css）
- **板块工作台革命**：10 板块统一 3 分区工作台（侧栏 zone | 主区 zone | inspector zone），
  12 个子代理并行改造：home Mission Control（Bento 网格 + AI 状态卡）/ chat 对话驾驶舱（新增
  上下文·人格 inspector）/ novel 世界构建工作台（轨道子导航 + 大纲树侧栏 + 属性检查器）/
  imagegen 画廊工作台（新增历史·队列 inspector）/ gaea 办公 3 分区（会话栏|过程卡流|工具交付物）/
  memoryhub 记忆图谱舰桥（左分类轨道 + 右详情检查器）/ modelcenter 引擎控制台（左导航 + 右统计检查器）/
  characterlib 角色档案库（左检索栏 + 右详情检查器）/ settings 控制室（左分类导航 + 帮助面板）/
  knowledge 知识舰桥（列表|详情|引用 3 分区）
- **视觉升级 Luminous Glass 2.0**：`--v3-*` 令牌派生（高光线/柔光/分区线/遥测色），卡片顶部 1px
  高光线、hover 柔光位移 ≤2px、状态三重传达；App.tsx 补齐容器色令牌 shim；零硬编码 hex、antd icons only、
  焦点环/aria 齐全、reduced-motion 双路径降级
- **修复**：modelcenter InspectorPanel 注释 `*/` 语法错误、ChapterPage activeTab TDZ、CharacterLibraryPage
  回调参数类型（build 模式 tsc -b）、test/setup.ts 补 localStorage/sessionStorage polyfill（Node 25 基线）
- **发布前打磨（T6-10.2 收官）**：
  - 聊天：模式切换条（普通对话/角色对话）从聊天窗口上方移入全局顶栏轨道条（v3-strip）——
    MainLayout 提供 `#v3-chatmode-host` 宿主容器（仅聊天板块显示），ChatPage 经 createPortal 渲染，
    ChatModeBar 新增 `variant='strip'`（去横条外壳，由轨道条统一玻璃/分割线）
  - 聊天：输入框两级重设计——工具行（搜索/深度思考/语音，激活态胶囊 + 键盘提示）独立于输入卡，
    输入卡仅 textarea + 发送（ChatComposer 新增 ComposerTool 子组件）
  - **根因修复（重要）**：7 个 CSS 文件头注释内 `--md-sys-*/--gaea-*/--color-*/--v3-*` 的 `*/` 提前闭合
    CSS 注释，解析器吞掉各文件首个规则（`.novel-hub`/`.ml`/`.v3-*` 等）——表现为小说板块面板
    「只有上半截」（高度塌缩为内容高）、首页布局约束失效；全部改 `* /` 修复（novel-workspace /
    module-launcher / foundation / imagegen / character-library / settings-page / hub）
  - 首页全屏适配：移除 `.ml-shell` 1320px 宽度上限（全屏 Bento 网格随宽扩展），卡片最小宽 176→216px，
    宽屏（≥1600px）语音卡内容居中 + 侧栏限宽
  - 验证：tsc/vite build 通过；浏览器 2560×1440 全屏 + 小说三栏撑满截图走查；`wails build` 重建通过
- 验证：tsc 0 errors + vitest 全量 **91 文件 457 用例通过** + `npm run build` 成功；
  浏览器全板块截图视觉走查（home/chat/novel/imagegen/办公/memoryhub/modelcenter/characterlib/settings）
- 版本：wails.json / versioninfo.rc / frontend package.json → 3.0.0

## v2.40.0「3.0 架构主线 · Wave 4：Step 3 收官」（2026-08-15）
> 3.0 架构改造 Wave 4：Step 3（Provider Seam）遗留收官——semantic_search 工具注册、
> BalanceKind 从 ProviderEntry 贯通、ModuleLauncher 清单化，双轨并行实施 + 父代理集成。
> 详见 releases/v2.40.0.md。
- **semantic_search 工具注册**：决策纳入——实现完整且有 E2E 测试，从死代码恢复为办公 agent
  可用工具；gaeaSpecialistTools 集中注册 ocr + semantic_search，测试断言防回归
- **BalanceKind 贯通**：ProviderEntry 新增 `balance_kind`（空 = 历史默认 deepseek 形状）；
  boot→control.Options→controller.Balance 全链路透传，改走 billing.FetchByKind——切换余额
  后端只改配置、未知 kind fail-closed，补齐 Step 3d #8 消费端贯通；config/control/boot 三层测试
- **ModuleLauncher 清单化**：新增 boards/launcher.ts 纯函数派生（deriveLauncherModules +
  LAUNCHER_DESC）；ModuleLauncher 改 useSyncExternalStore 订阅活动清单，删除静态
  canonicalBoards 引用——后端合并清单（含 knowledge）变化后首页启动器自动跟随；
  launcher.test.ts 7 用例 + manifests.test.ts 36（43/43 过）
- 验证：go build/vet 干净 + test-all.ps1 全量（见发布说明）+ 前端 tsc/eslint 0 errors +
  vitest 全量（见发布说明）+ TestBindingsCompleteness PASS（464）。发布 gaea-v2.40.0.exe。

## v2.39.0「3.0 架构主线 · Wave 3」（2026-08-15）
> 3.0 架构改造 Wave 3：Step 3b LLM Seam + Step 3c OCR/ASR/TTS Seam + Step 3d 分类统一与 8 处硬编码注册表化 +
> 前端 GetBoardManifests 接线，四路并行实施 + 父代理集成。详见 releases/v2.39.0.md。
- **Step 3b LLM Seam**：LLMProvider{Provider;Chat} + ChatFromStream 聚合；bridge 互斥自注册（DefaultLLMKind=wubigrok）；
  boot.NewProvider 经 NewLLM（providers[].kind 驱动 + fail-closed）；agent 聊天/子代理只依赖 seam 接口（19 测试）
- **Step 3c OCR/ASR/TTS Seam**：OCRProvider（ovis/tesseract，GAEA_OCR_ENGINE 驱动）+ TTSProvider（edge/sapi/herdsman/xai
  自注册，四级回退与合成器链注册表化）+ ASRProvider（herdsman，voice.Manager 接口注入）；isSTTModel 委托分类单源
- **Step 3d 分类统一 + 8 处注册表化**：modelengine 导出 ClassifyModelKind/ClassifyModelByName（六桶单源）；websearch 6 引擎
  注册表（engine_order 可配序）/embed/rerank/vision/markitdown/billing 注册表化（弃 HERDSMAN_BASE_URL/GAEA_VISION_*
  环境变量绑死）；OCR 工具补注册；image_gen 裸字符串改常量
- **前端接线**：loadBoardManifests 改调 CoreB.GetBoardManifests + normalize 差集（knowledge/home/weixin）+
  KnowledgePage 注册；45/45 测试
- **父集成**：gaea.toml 新增 [retrieval]/[vision]/[markdown_converter] 段 + [search] engine_order；boot 装配四组
  Set*Runtime；app 层 5 处 provider.New→NewLLM
- 验证：go build/vet 干净 + test-all.ps1 110/110 + 前端 tsc/eslint 0 errors + vite build 42.9s + vitest 420 过
  （27 基线失败零回归）+ TestBindingsCompleteness PASS（464）。发布 gaea-v2.39.0.exe。

## v2.38.0「3.0 架构主线 · Wave 2」（2026-08-15）
> 3.0 架构改造 Wave 2：Step 1 app 层接线 + Step 2 板块 Manifest（后端 board 包 + 前端 PageRegistry）+
> Step 3a Image Seam，四路并行实施，每 Step 独立提交。详见 releases/v2.38.0.md。
- **Step 1 app 层接线**（会话事件日志「日志即真相」运行时闭环）：Resume→Restore（DetectLegacy 迁移 →
  checkpoint+log tail 重放）、Save→日志（事件模式双写）、模型调用前 flush 检查点（fail-closed，失败中止回合）、
  压缩→checkpoint（回合后 Snapshot 刷新压缩投影 + 已消费 seq）；session.log_format 回退开关，legacy 零行为变化
- **Step 2 板块 Manifest**：board 包（Board 接口 + Manifest 16 字段 + Validate 缺陷 2 防复发）+ 10 板块 canonical
  清单（9 业务 + knowledge D7）；module_registry manifest 驱动装配（intent 无 handler 启动显式报错）；
  GetBoardManifests 挂 CoreB（绑定面 464 方法）；前端 PageRegistry + MainLayout 附 B 12 硬编码点清单化 +
  events.ts 常量表（21 后端 + 4 前端）+ ModuleLauncher 清单驱动；label 单一来源统一菜单文案（用户决策）
- **Step 3a Image Seam**：图片后端注册表化（openai 兼容/comfyui 各自 init 自注册，互斥注册 panic、未知 kind
  fail-closed）；generateImageXAI 走注册表 + 401 刷新 token 单次重试守卫（retried 防无限递归）+
  imagine:content-moderated 友好提示；config 驱动选择零代码切换
- 验证：go build/vet 干净 + test-all.ps1 110/110 包 + internal/app 26.5s + 前端 tsc/eslint 0 errors +
  vite build 通过 + vitest 404 过（27 个 jsdom localStorage 基线失败，与 v2.37.0 一致零回归）+
  TestBindingsCompleteness PASS（464）+ check-bindings-drift OK。发布 gaea-v2.38.0.exe。

## v2.37.0「正确性纵深 · 收官」（2026-08-15）
> 阶段 7 第二~四刀（T7-2 可见性收口 / T7-3 名实相符 / T7-4 前端性能收尾）并行实施 +
> 3.0 架构主线 Step 0 修债 + Step 1 会话事件日志。详见 releases/v2.37.0.md。
- **T7-2 可见性收口**：qrlogin/chatWebSearch/SaveConfig/LocalTranslate 吞错清零；成本进料截断 6000 字 +
  整批事务；测评参数钳制 + 基地址 engineMgr；token 明文清理 + 剧照 ID 哈希防穿越（41 测试）
- **T7-3 名实相符**：PDF FlateDecode 压缩流还原 + OCR 单页容错 + OvisOCR2 4096/截断检测；语义检索按需
  + search 上限 + BM25 缓存 + dashboard mtime 真实聚合 + WatchErr 回退轮询（约 33 测试）
- **T7-4 前端性能收尾**：写路径静默清零 + 三态错误重试 + Transcript/MarkdownContent memo + Toast role +
  reconcileFinalAnswer 完整文本比较（41 用例）
- **Step 0 修债**：office 模块注册 GaeaSend + MainBrainChat 全链路测试 + 版本源同步脚本（搭车）
- **Step 1 会话事件日志**（3.0 地基）：append-only 日志 + 投影 + checkpoint + 迁移 + 派生 API +
  GaeaHistory 黄金测试逐字节一致（机制层；app 接线留待 gen_bindings）
- 验证：go build/vet 干净 + 逐包测试全绿（含 session 67 / T7-2 41 / T7-3 约 33 / T7-4 41）+ 前端
  tsc/eslint 0 errors + vite build 通过 + 冒烟 /api/health 200。发布 gaea-v2.37.0.exe（34.5MB，
  SHA256=37A56F54DF653E3D9E8A5751EA282CEB34BF5BBCA2672D26439BF7BAEBA7A62B）。

## v2.34.0「正确性纵深 · 并发正确性」（2026-08-15）
> 阶段 7 第一刀（T7-1）：轻语会话并发安全、任务调度器竞态、TCCA 指标聚合收敛、AI 客户端状态与重试。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段7-正确性纵深.md；详见 releases/v2.34.0.md。
- T7-1.1 **轻语会话并发安全**（internal/whisper + whisper_handler + app.go Shutdown）：Orchestrator
  per-instance Mutex 串行化三并发入口（GUI/微信/语音）；修复 CloneFullState 浅拷贝竞态（-race 实证）→
  逐指针/切片/map 深拷贝；WorkingMemory/AssociationIndex/HabitsStore/ActiveRecall 加 RWMutex；修复
  forSession 读路径惰性写 map（-race 实证）→ 只读；跨会话持久化走 persistStateSync（回合锁内快照 +
  persistMu 落库）+ drainAndPersistAll 挂 Shutdown（末轮先 drain 再 persist）；rhythm 包级计数器移入
  Orchestrator 实例（Reset 只清自己的）；新增 12 测试（并发访问/回合串行/节奏隔离/末轮落库）；
- T7-1.2 **任务调度器竞态修复**（internal/gaea/tasks）：markTerminal 进度语义（succeeded 才置 100，
  SQL CASE WHEN）；取消优先于 succeeded（handler 返回 nil 也不吞取消）；Cancel 与出队原子化
  （WHERE status='queued' 条件 UPDATE + 出队前注册取消）；runNext 包 defer recover（handler panic →
  failed 不重试，worker 存活）；新增 10 测试（进度/取消优先/竞态 50 并发/panic 恢复），22/22 -race 全绿；
- T7-1.3 **TCCA 指标聚合收敛**（internal/gaea/context/metrics.go）：MergeChild 与 Report 同字段集
  （补齐 CacheHitTokens/CacheMissTokens/BreakCount/CompactionCount 四条漏项）；merged 标记移入 child.mu
  临界区 + 数据快照走 child.Report()（子锁，锁序 父→子→孙 无死锁）；ForkCount +1 每 child 恰好一次
  （children 移除 + merged 标记防重）；新增 6 测试（Merge 前后 Report 一致/全字段/防重/并发 -race）；
- T7-1.4 **AI 客户端状态与重试**（internal/ai/client.go）：非流式 Chat 复用流式退避（连接错误/5xx
  重试 1s/2s；401 仅 xAI 同函数内刷新重发一次，不递归不占双槽）；activeEngineID/imageBackend/token
  加 RWMutex + GetToken single-flight 刷新；修复 vet 错误（Sprintf %w→%v）；新增 7 测试
  （连接/5xx/401 刷新/不占双槽/状态并发/20 并发单飞刷新）；
- 验证：go build/vet 干净 + scripts/test-all.ps1 **109/109 包 ok** + 并发门禁 C（whisper/tasks/context/ai
  go test -race 全绿，-race 需 cgo：CC=C:/msys64/ucrt64/bin/gcc.exe）；前端零改动（tsc 0 errors、
  eslint 0 errors、359 存量 warnings 与基线一致）；TestBindingsCompleteness 兜底（无新绑定）；
  冒烟通过（/api/health 200）。

## v2.33.0「质量收敛 · 前端收敛」（2026-08-14）
> 阶段 6 第十刀（T6-10）贯穿收官：巨型文件拆分、any 清零、漂移检查恢复、mock 契约化、桥接归一、性能与补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.33.0.md。
- T6-10.1 **巨型文件拆分**（8 个巨型文件全部收敛，拆分全程行为测试基线先行）：
  - ChatPage.tsx 1022→370 行（pages/chat/{constants,types,utils}.ts + components/chat/{ChatComposer,
    ChatModeBar,ChatPersonaBar,MessageList,SuggestionCard,WelcomeScreen}.tsx + hooks/useChatStream/
    useChatTopics/useChatVoice/useCustomTemplates.ts）；
  - ImageGenPage.tsx 911→310 行（hooks/useImageGenConfig/useImageGenHistory/useImageGenQueue +
    components/imagegen/meta.ts）；
  - CapabilitiesPanel.tsx 803→178 行（capabilities/{ServersSection,SkillsSection,ToolsSection}.tsx +
    useCapabilitiesData.ts）；
  - Composer.tsx 786→406 行（composer/ 7 组件 + useComposer{Attachments,Menus,Workspace}.ts）；
  - mock.ts 1563→50 行（按域拆 mock/{chat,core,cost,memory,model,office,retrieval,settings,shared,state}.ts
    10 文件，11 个 no-op 逐条落实并注释）；
- T6-10.2 **any 清零**：eslint.config.js no-explicit-any warn→error 进 CI；315 处显式 any 消灭（→0），
  新增 any 即 lint 失败；历史 as any 逃生口由类型化兼容层替代（wails.d.ts 注释成文）；
- T6-10.3 **绑定漂移检查恢复（双向）**：gen_bindings 新增 -names 模式（只输出方法名稳定排序）；
  bindingNames.ts（462 方法清单）；bridge.ts 类型级双向守卫 _CheckAppBindingsHasNoStray +
  _CheckAppBindingsCoversAll（任一方向漂移 tsc 即红）；scripts/check-bindings-drift.ps1 + CI 步骤；
- T6-10.4 **mock 契约对齐**：mock-contract-e5.test.ts RetrievalEvalRun 改契约校验（total=12 真实查询集、
  threshold=0.8、passed 与 recallAt10 自洽、kind:name 形式、首条锚点「打桩设备 台班价」），不再锁定
  虚构 0.85；CostImportVisionPreview/CostCompare/UnifiedSearch 补结构断言；
- T6-10.5 **虚拟化与性能**：Sidebar 会话列表 react-window List 虚拟滚动 + 过滤防抖；CostLibraryView
  memo/useCallback/useMemo 化；新增 useDebouncedValue（空串即时同步）+ 6 测试，Composer 外 2 处消费；
- T6-10.6 **桥接归一**：删除 api/bridge.ts（123 行旧代理）；initBridge 并入 gaea/lib/bridge.ts（+478）；
  wails.d.ts 同步收敛；新增绑定单处注册；
- T6-10.7 **测试补强**：Sidebar.test +40 / CostLibraryView.test +56 / GraphView.test +47 / ChatPage.test +26 /
  CharacterLibEditor.test +8 / BindSection.test +8 / useDebouncedValue.test 新 6 用例；vitest 354→361；
- 验证：go build/vet 干净 + go test ./... **109/109 包 ok** + TestBindingsCompleteness PASS（462 方法，
  无绑定变更）+ 漂移检查 OK；tsc 0 errors、eslint 0 errors（359 存量 warnings）、vitest **361/361**
  （80 文件）、vite build 14.46s；冒烟通过（/api/health 200）。发布 gaea-v2.33.0.exe（32.8MB，
  SHA256=8FADBB7385D794DB69842171D4F95E678849FA8481686F646A4BBA6E94F4E92F）。

## v2.32.0「质量收敛 · 辅助合集·名实相符」（2026-08-14）

> 阶段 6 第九刀（T6-9）：微信生命周期与凭据、OCR 兜底名实相符、配置原子写、路径端口可配置、token 改 header。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.32.0.md。
- T6-9.1 **微信生命周期与失效自愈**（internal/channels/weixin/clawbot.go + whisper_state.go）：
  - Stop 幂等（stopMu + stopCh 关闭即置 nil，二次 Stop 不 panic）；Start 支持重启（running.Swap 幂等
    防重复拉起 + stopCh 重建 + sessionExpired 重置，Stop→Start 轮询真正恢复）；
  - 会话过期（errcode=-14）触发 OnSessionExpired 回调后退出轮询（删除 5 分钟空转）；app 层注入回调
    emit notice「微信助手 X 会话过期，请重新扫码绑定」；getUpdatesFn/notifyStartFn/notifyStopFn 测试注入点；
  3 测试；
- T6-9.2 **凭据与表治理**：
  - wxToken DPAPI 加密（assistant/manager.go：save() 落盘加密 dpapi: 前缀、Load 解密/旧明文一次性
    迁移、解密失败返回含助手 ID 明确错误；内存保持明文 List 回显不变）；
  - weixin_* 4 张死表（grep 核实零读写）SchemaV13 DROP 追加迁移链 + ClearStructuredData 移除；3 测试；
- T6-9.3 **OCR 兜底名实相符**（office/docmd/ocr.go + single_prompt.go:43）：
  - 超时杀进程树：proc.StartTracked（Job Object）+ 超时 KillTracked + 同步 Wait 回收，失败零孤儿；
  - 单图降级：OCRImageText OvisOCR2 不可用 → tesseract 降级（同 PDF 参数），双不可用才报安装提示；
  - 文案删除名不副实的「Windows 原生 OCR」（RapidOCR 核实存在保留）；4 测试；
- T6-9.4 **配置原子写**（internal/config）：saveConfigFile 临时文件+fsync+rename 原子覆盖（失败保留
  原文件）；Load 损坏备份 .gaea_config.json.corrupt-<ts> 后默认值继续；4 测试；
- T6-9.5 **CosyVoice 路径端口可配置 + 退避重试**：新配置键 cosyvoice_dir/cosyvoice_port（默认与历史
  一致，端口校验）；tts_service.go 写死常量全删由配置推导；启动失败 1s/2s/4s 退避重试 3 次；4 测试；
- T6-9.6 **token 改 header**：服务端 tokenOK 删除 ?token= 查询兜底（仅 Bearer/X-Gaea-Token，常量时间
  比较不变）；前端弃 EventSource 改 fetch 流式 SSE 带 Authorization 头（parseSSEFrame/parseSSEStream
  纯函数：跨 chunk/CRLF/keep-alive/流尾 flush）；httpToken.ts 删 URL query 读取；Go 3 + 前端 6 测试；
- 验证：改动包全绿 + vet 干净 + TestBindingsCompleteness PASS（462 方法）；tsc 0 errors、
  vitest **354/354**（79 文件）、eslint 0 errors（72 存量 warnings）、vite build 15.17s；
  全量 go test 中 hook/skill/tts 3 包 AV 锁 test.exe（单独重跑全绿，环境抖动先例）与 docmd GBK 编码
  类失败（c426d3f 基线 worktree 复现同款，非回归）。

## v2.31.0「质量收敛 · 记忆·生命周期与审计」（2026-08-14）
> 阶段 6 第八刀（T6-8）：dream 写入审计、facts 归档生命周期清理、索引截断按边界、记忆组件补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.31.0.md。
- T6-8.1 **dream 路径决策与审计**（internal/gaea/control/controller_memory.go + internal/app/gaea_dream.go）：
  - 审批决策成文（docs/DREAM_WRITE_POLICY.md + 代码注释）：dream 写入不纳入 hardAskTools 逐条审批
    （后台异步 90s 超时无法等人工确认；显式路径本身即用户触发），补偿 = 每次实际写入落审计日志；
  - SaveDreamFacts 签名改 (source string, facts)（source=auto_dream|explicit），审计行
    {ts, source, saved, names} 追加到 <userDir>/dream-audit.jsonl（JSONL，尽力而为不阻断写入）；
  - DreamAuditEntries 读取入口（倒序最近 N 条）；新增 3 测试（自动/显式各断言 1 条审计行 + 记忆未配置跳过）；
- T6-8.2 **facts 生命周期清理**（internal/gaea/memory + 新绑定 + 前端按钮）：
  - 保留策略：归档超 90 天（ArchivedRetention）硬删；sqliteBackend/fileBackend 双后端实现
    CleanupArchived（返回被删行含溯源字段：名称/描述/正文/归档时间/来源会话）；
  - ListArchivedPaged(limit, offset) 总量+分页（limit 钳制 [1,200]/默认 50），防全量返回；
  - 新绑定 GaeaMemoryCleanupArchived / GaeaMemoryArchivedList（gen_bindings 462 方法 + 完备性 PASS）；
    清理逐条 slog 日志 + 溯源审计 purge-audit.jsonl（GAEA_DATA_ROOT 可隔离，测试不触真实用户库）；
  - 前端「归档」tab「清理超期归档」按钮（Modal.confirm + message.success + 刷新）；
  - 新增测试 7：memory 包 5（SQLite 清理/无超期不误删/分页/文件后端清理/文件后端分页）+ app 2
    （清理幂等+审计落盘、分页绑定）；
- T6-8.3 **索引截断按边界**（internal/gaea/memory/memory.go）：
  - 预算口径统一：memoryIndexBudget（3000 runes）→ memoryIndexBudgetBytes = 4096，与 Block() 的
    4096 字节全块阈值同口径（注释成文）；
  - capMemoryIndex 改 truncateIndexByLines 纯函数：预算内最后一个 '\n' 处按行边界截断（不切半行），
    markdown 链接保护（未闭合 "[" 或 "](url" 的 ")" 被截时回退到链接起始行之前整体舍弃），
    只在 ASCII '\n' 处切 → UTF-8 安全不产生半个 rune；单行超预算宁丢整行不切半字；
  - 截断提示文案与旧实现逐字一致；预算内原样返回不追加提示；
  - 新增 6 个测试函数（行边界/字节预算/rune≠byte 中文 emoji/链接完整性 5 子测/单行超预算），
    既有 recall 预算用例全绿；
- T6-8.4 **前端组件补测**（memoryhub，新增 13 vitest 用例）：
  - GraphView 5 用例：链式 stub 替身 3d-force-graph（vi.hoisted），断言工具条/节点边计数/类型过滤
    重构图/节点点击详情 Modal/空数据/variant=home 隐藏工具条；
  - WhisperMemoryLibrary 8 用例：domain 分组（含未知归「其他」）、三关键词搜索过滤、情节 tab
    （emoji/强度条/关键词/轮次）、事实/情节详情 Modal、导出归档链路（PickDirectory +
    WhisperExportArchive 调用）、事实/情节双空态；
  - 组件实现 0 改动（纯测试）；桥接经 vi.mock("../../lib/bridge") 注入确定性数据；
- 验证：改动面 Go 包（control/memory/app）全绿 + vet 干净、TestBindingsCompleteness PASS（462 方法）；
  tsc 0 errors、vitest **348/348**（78 文件）、eslint 0 errors（存量 warnings）；
  internal/app 全量存在 2 个与 v2.30.0 基线一致的既有环境失败（TestOfficeFullPipeline GBK 编码提取、
  TestSemanticSearchTool_EndToEnd 需本地嵌入模型），已用基线 worktree 复现同失败，非本刀回归。

## v2.30.0「质量收敛 · 小说·导出与原子性」（2026-08-14）
> 阶段 6 第七刀（T6-7）：export 整改、生成中断与互斥、落盘原子化、模板占位符、CreatePage 拆分。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.30.0.md。
- T6-7.1 **export 整改与测试**（internal/export，新增 13 测试）：
  - 章节读取失败静默 continue → slog.Warn + FailedChapters 计数（四格式观测一致）；
  - 作者取自项目元数据（ProjectMeta.Author，无配置回退 "gaea"）；EPUB 与 HTML 统一 markdownToHTML
    单一分段器（删除 chapterToHTML）；TXT/MD 世界观对齐；TXT/MD/EPUB 写前 MkdirAll；
  - sanitizeFilename 加固（Windows 保留名 CON/PRN/AUX/NUL/COM1-9/LPT1-9 含 CON.txt 形式 + 尾部点/空格）；
  - EPUB AddSection 错误不再丢弃；失败分支测试因沙箱无 SeCreateSymbolicLinkPrivilege 以 Skip 降级（逻辑保留）；
- T6-7.2 **生成中断与互斥**（internal/app/create_chapter_handler.go）：
  - 请求级 context.WithCancel（取消传播到 ChatStream，双路径落盘已生成部分）；
  - 新绑定 CancelCreateChapter(chapterNum, branch) bool（幂等；gen_bindings 460 方法 + 完备性 PASS）；
  - 按章节互斥（同章节并发生成拒绝明确错误，不同章节并行）；取消未生成内容不写空文件；
- T6-7.3 **落盘原子化**（internal/project）：writeFileAtomic（CreateTemp→fsync→Rename）覆盖
  writeJSON 全部 JSON + WriteChapter/WriteChapterBranch/WriteWorldview；失败清理临时文件保留旧文件；5 测试；
- T6-7.4 **模板替换精确化**：prompts/create-chapter.json "5000" 字面量 → {word_count} 占位符，
  substituteWordCount 只替换占位符杜绝误伤；2 测试；
- T6-7.5 **前端拆分与停止按钮**：CreatePage 791→288 行（拆 8 文件：chapterStreamTypes 判别联合 +
  useChapterStream hook + ChapterTreePanel/EditorPanel/CreateInspector/NewCharactersModal/BranchWizardModal +
  characterStatus 枚举单源）；「停止生成」按钮（CancelCreateChapter 接线，false 本地兜底不悬挂）；
  cancelled 事件三路收尾保留部分正文；新增 18 vitest 用例；
- 验证：export/project/types/app 包 ok（app 24.3s）、vet 干净、TestBindingsCompleteness PASS（460 方法）；
  tsc 0 errors、vitest **334/334**（76 文件）、eslint 0 errors（存量 warnings）。

## v2.29.0「质量收敛 · 模型中心·密钥与 UI」（2026-08-14）
> 阶段 6 第六刀（T6-6）：refresh_token DPAPI 加密、汇率配置化、probe 告警修复、UI 拆分与竞态守卫。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.29.0.md。
- T6-6.1 **refresh_token 密钥一致性**（internal/auth/token.go）：Save 敏感字段（access_token/refresh_token）
  经 secure.EncryptString DPAPI 加密落盘（dpapi: 前缀，JSON 结构不变）；Load 有前缀解密、解密失败返回含字段名
  的明确错误（绝不静默 nil）；无前缀旧明文自动一次性重写为加密（迁移幂等，同一把锁内完成）；非 Windows 降级
  分支 round-trip 完整；新增 3 测试（落盘无明文/旧明文迁移/解密失败报错）；
- T6-6.2 **汇率配置化**（internal/modelengine/stats.go + internal/config + 绑定）：
  - usdToCny 写死 7.2 → 配置键 usd_cny_rate（默认 7.2，saveSetters 拒绝 <=0/NaN/Inf）；
  - 新绑定 GaeaGetUsdCnyRate/GaeaSetUsdCnyRate（gen_bindings 459 方法，TestBindingsCompleteness PASS）；
  - 注入式缓存（启动注入 statsRecorder 内存副本，修改双写即时生效，record/summary 零 IO）；
  - 前端 ModelPanel 汇率输入（回填/保存/正数校验）+ engines.ts 包装；新增 4 测试；
- T6-6.3 **probe 告警文案**（internal/herdsman/probe.go:221）：目录缺失告警改打印真实目录路径
  （filepath.Join(rootDir, name) 与 checkDir 口径一致），不再误打印 config.yaml 路径；新增 1 测试；
- T6-6.4 **UI 拆分与单源**：ModelCenterPage 顶层 useState 42→3（5 分类状态下沉到 5 个 hooks：
  useEngineState/useStatsState/useImageState/useVoiceState/useBindState）；XAI_VOICES 全前端仅 utils.tsx
  一处定义（VoiceSettingsPanel/ChatPanel 改 import 单源）；
- T6-6.5 **竞态修复**：refreshLocalModels 请求序号守卫（refreshSeq，过期结果丢弃）+ 5s 定时器随 category
  重置（effect 开头作废上一分类在途刷新）；新增 2 竞态测试；
- 验证：auth/modelengine/config/herdsman/app 5 包 ok（24.4s app）、vet 干净；tsc 0 errors、
  vitest **316/316**（72 文件）、eslint 0 errors（762 存量 warnings）。

## v2.28.0「质量收敛 · 轻语·测试与可观测」（2026-08-14）
> 阶段 6 第五刀（T6-5）：补测试 146 用例、错误可见化、异步写可观测、成人模式决策成文、db 收敛、占位清理。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.28.0.md。
- T6-5.1 **补测试**（仅新增 9 个测试文件，零实现改动）：146 个用例（要求 45+）；emotion_fusion 18/18 函数
  100% 语句覆盖；memory_consolidator 98.5% / memory_contradiction 99.3% / memory_self_editor 98.5% /
  vector_store 96.5% / dispatch_router 96.3% / agent_loop_runner 92.3% / canon 全 100% / desktop 96.8%+；
  测试暴露 3 个真实缺陷（记录未改）：normalizePath %ENV% 展开不生效（Go os.ExpandEnv 不支持 %VAR%）、
  mergeProhibitions 道歉过滤漏带引号"对不起"、self_editor log 裁剪后可重新增长至 200；
- T6-5.2 **错误可见化**（T6-1.3 已改 4 处 + 本刀补齐）：whisper_handler 记忆写协程错误全部经
  recordMemoryWriteError 汇聚（slog.Error + 计数）；
- T6-5.3 **异步写可观测**：whisperWriteErrors 计数器（count/最近错误摘要/时间）+ MemoryWriteErrorSink
  透传（LLM 失败 llm_extract / JSON 解析 json_parse / panic / persist 落库失败四类 phase 全覆盖）；
  persist 协程提取 persistStateAsync，persist 系列函数改返回 error（errors.Join 聚合，不再 _ = 吞错）；
  新增 5 测试；
- T6-5.4 **成人模式决策成文**：新增 docs/ADULT_MODE.md（六节：决策/理由/接口现状/前端实证/商用化恢复
  5 步方案/代码位置）；orchestrator.go:91 注释引用文档；**删除 WhisperSetAdultMode 死接口**
  （前端零引用、实现静默忽略参数；gen_bindings 457 方法重新生成 + TestBindingsCompleteness PASS）；
- T6-5.5 **db 细节收敛**：GetDatabase 签名改 (*sql.DB, error)（12 个 repos 文件 58 处调用点适配）；
  PRAGMA 单一来源（删除重复循环，DSN 唯一，grep 断言各恰 1 次）；V11 FTS 全量重建失败补 slog.Error；
  新增 4 测试；
- T6-5.6 **陈旧占位清理**：删除 4 个占位文件（plan_document_intent/paper_card_companion/
  desktop_mode_policy/desktop_opening——目标文件已核实或不存在），grep 零引用；
- 验证：internal/whisper 3 包 + internal/app 23.9s ok、vet 干净；tsc 0 errors、vitest 312/312、
  eslint 0 errors（存量 warnings）。

## v2.27.0「质量收敛 · 绘梦·链路真实生效」（2026-08-14）
> 阶段 6 第四刀（T6-4）：取消真实生效、flux 名实相符、历史图片可恢复、格式/注入面修复、核心链路测试补全。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.27.0.md。
- T6-4.1 **取消真实生效**（image_handler.go/image_comfyui.go）：
  - CancelImageGeneration 在 cancel context 之外调用 POST /interrupt 中断 ComfyUI 当前任务；本地取消标记拒绝后续排队提交
    （ComfyUI 无删除排队任务 API，/queue 仅查询——注释说明）；checkHistory 携带 ctx、取消后轮询即刻退出；取消幂等；
- T6-4.2 **flux 名实相符**：文生图改显式映射表 txt2imgWorkflows（krea2/z-image-turbo/flux），未知模型返回中文错误
  （静默降级已消除）；flux 实现真实工作流（UNETLoader flux1-schnell + DualCLIPLoader type=flux + ae VAE +
  4 步 KSampler）；img2img 白名单化；
- T6-4.3 **历史图片可恢复**：imageItem 增 file_path 字段，生成流程把落盘路径写入历史；前端历史分级存储
  （>200k base64 只存 path）、挂载时经 GaeaAttachmentDataURL 回填恢复、下载/剧照优先 file_path；
- T6-4.4 **尺寸解析校验**：parseSize 弃 Sscanf 改 strconv.Atoi 严格解析，非法输入中文报错，钳制 64–2048；
- T6-4.5 **端口命令注入修复**：findProcessByPort 弃 cmd/findstr 拼接改 exec.Command("netstat","-ano") 参数数组
  + 输出解析；端口白名单 1–65535；
- T6-4.6 **核心链路测试补全**：ComfyUI 客户端 httpClient/pollInterval 可注入；httptest 五链路（提交 3/轮询 4/
  取消 4/上传 3/下载 2）+ 全链路端到端；Go 新增 31 用例；前端 media/historyMeta/queue 纯逻辑模块 + 19 用例；
- 验证：internal/ai 4.2s + internal/app 23s ok、vet 干净；tsc 0 errors、vitest **312/312**（70 文件）、
  eslint 0 errors（761 存量 warnings）。

## v2.26.0「质量收敛 · 对话·流可靠」（2026-08-14）
> 阶段 6 第三刀（T6-3）：流订阅竞态与超时、落库错误透传、语音持久化、迁移一次性、导出转义。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.26.0.md。
- T6-3.1 **流订阅竞态与超时**（ChatPage.tsx）：
  - runID 一到即在同一微任务注册 EventsOn（零异步间隙，首帧不丢）；30s 无帧超时（STREAM_SILENCE_TIMEOUT_MS）→ sending 复位 + 错误展示 + finally 必执行；
  - finish 幂等收尾覆盖 done/error/超时/启动拒绝/卸载五路；新增 12 个组件测试（fake timers）。
- T6-3.2 **落库错误透传**（internal/app/chat_service.go）：
  - appendChatExchange 返回 error；流式路径落库失败 emit error 终态而非 done（前端可见失败）；
  - ChatTopicsList/ChatMessagesList 签名改返回 error（绑定签名变更，前端 try/catch + LogFrontendError 同步）。
- T6-3.3 **语音持久化**：新增绑定 ChatAppendMessages（单事务批量落库）+ 前端语音识别/回复落库（不走 ChatSend 无重复）；朗读 URL revoke（onended/onerror/play 失败/卸载）；打字循环取消标志（切话题/卸载中止）。
- T6-3.4 **迁移一次性**：migrateLegacyTopics 加持久化标记（gaea_chat_migration_v1），成功才写；失败记日志不静默、保留旧键、会话内不无限重试；ChatImportTopic 改单事务（ImportTopicTx 全成或全回滚）。
- T6-3.5 **导出转义**：ChatTopicExportMarkdown 消息原文转义 Markdown 敏感字符（行首井号/反引号/尖括号/竖线）；sanitizeChatFilename 加固（Windows 保留名 CON/PRN/AUX/NUL/COM1-9/LPT1-9、尾部点号、截断 40、空→chat）；ChatGeneral 补主脑派发依赖注释（不删除）。
- 验证：test-all.ps1 全量包 ok、vet 干净；tsc 0 errors、vitest **290/290**（67 文件）、eslint 0 errors（762 存量 warnings）。

## v2.25.0「质量收敛 · 办公引擎·正确性」（2026-08-14）
> 阶段 6 第二刀（T6-2）：docmd 分页修复、TurnResult 语义、后端看门狗、Send 排队、禁写注册表化、TCCA/evidence 补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.25.0.md。
- T6-2.1 **PDF 页数统计与分页过滤修复**（internal/office/docmd/docmd.go）：
  - 页数统计改 countPDFPages 精确匹配（排除 /Type /Pages 页树干扰，修复总页数恒多 ≥1）；
  - BT..ET 文本按页对象归类（页码由页对象决定而非 BT 块自增），页范围过滤不再错位；
  - OCR 循环改绝对页码（修复 pdftoppm 从 first>1 渲染时范围错位一页），OCR 范围与"已截断"提示同源；
  - 新增 pdf_pages_test.go 7 测试（构造最小合法 PDF fixture，含 /Pages 干扰/无空格/页树边界/页内多 BT 块）。
- T6-2.2 **运行链路结果语义修复**（internal/gaea/agent/agent_run.go、agent_stream.go）：
  - TurnResult 的 blocked/precheck blocked/suppressed/tool panic 计入 Errors，Success 仅整轮无错误为 true；
  - 终止流错误路径先写已收部分文本入会话再返回 err（不丢已生成内容）；
  - step-- 加下限 0，杜绝负 step 与 grace 边界组合出额外模型轮；新增 10 测试（Success 语义收紧不影响上层：controller 丢弃 TurnResult、前端无引用）。
- T6-2.3 **TCCA 与 evidence 补测**（internal/gaea/context、internal/gaea/evidence，仅新增测试零实现改动）：
  - 新增 58 个测试：context 覆盖率 39.2%→**97.0%**、evidence 91.1%（要求 ≥60%）；
  - 记录两处观察（不改实现）：MergeChild 的 ForkCount +1 语义存疑；CacheReport 不聚合子项 CacheHitTokens/CacheMissTokens/BreakCount（子代理全会话命中统计在父报告丢失）。
- T6-2.4 **落地后端看门狗**（internal/gaea/control/watchdog.go 新增）：
  - v2.13.0 声称的看门狗此前未落地（仅前端 30s 定时器）——本实现为进程内运行态看门狗；
  - 墙钟 10min / 停滞 30s 默认阈值（Options.Watchdog 可配置，==0 默认、<0 禁用维度）；
  - 触发走该回合 cancel（与用户 Cancel 同一中断链路）→ Emit TurnDone(Err) + 用户可见 Notice；
  - watchdogSink 观察推进：工具执行在途（ToolDispatch→ToolResult）与审批/提问等待豁免停滞，不误杀长任务；
  - 新增 8 测试 + 3 子测试；与 Send 队列共存回归通过（修复跨回合 channel 复用竞态）。
- T6-2.5 **Send 排队 + 禁写清单注册表化**（internal/gaea/control/controller.go、internal/gaea/tool、internal/gaea/agent/task.go）：
  - 运行中 Send 改限长队列（8 条），回合结束按 FIFO 排空（running 保持 true）；队满拒绝并发明确错误 notice；
  - 子代理禁写清单由工具注册表 PersistWrite 标记自动推导（删除手写 6 项 map），新持久化写工具加标记即自动纳入；
  - 禁写集合与 hardAskTools 完全一致（测试断言）；新增 8 测试 + 更新 2 测试。
- T6-2.6 **docmd.go 拆分**（1521→56 行，拆为 office.go/pdf.go/ocr.go/pagespec.go）：
  - 按职责纯搬迁、行为零变化（声明段重拼与原文逐字节一致验证）；23 测试全绿，覆盖率 39.2% 与拆分前一致（OCR/外部工具路径无环境跳过为既有状态）。
- 验证：test-all.ps1 **109/109 包 ok**、go vet 干净；tsc 0 errors；eslint 0 errors；vitest 274/274。

## v2.24.0「质量收敛 · 基础层·可靠性」（2026-08-14）
> 阶段 6 第一刀（T6-1）：SSE 流式加固、前端错误可见性、后端吞错清理。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.24.0.md。
- T6-1.1 **SSE 流式加固**（internal/ai/client.go）：
  - 行上限 64KB → bufio.Reader 任意长行（1.2MB 单行实测不断流、逐字一致）；
  - 连接错误/5xx 指数退避重试（默认 2 次 1s/2s；200 流开始后不重试防重复生成；401 走刷新、include_usage 400 降级整体重试）；
  - 空闲超时 60s（每次读重置计时；取消作用于 streamCtx 解除阻塞读）；
  - 代理接入：与 web_fetch/web_search 同源读取代理配置（netclient.NewHTTPClient），localhost/回环强制直连（herdsman/ComfyUI 不走代理）；
  - 修复解析协程 send 守卫用调用方 ctx（防超时后丢失错误块）；新增 client_reliability_test.go 8 测试。
- T6-1.2 **前端错误可见性**（bridge.ts/store.ts）：
  - BridgeError/normalizeError/invoke 统一入口/logFrontendError；app proxy 全部方法包 invoke 层（LogFrontendError 防递归）；
  - store.ts 8 集群 14 处静默 .catch(()=>{}) 改 logBridgeError（状态逻辑零改动）；
  - 新增 bridge.test.ts 6 用例 + store.test.ts +3。
- T6-1.3 **后端吞错清理 + 日志脱敏**：
  - ChatTopicsList/ChatMessagesList 读错记日志（签名未改）；whisper_handler 两处 _= 记日志；
  - memory_ingest/memory_consolidator LLM/解析失败 slog.Error；config.Load 坏 JSON slog.Error（签名未改）；
  - main.go 桥接 token 日志脱敏 maskToken（尾 4 位）；
  - 全库 _= 扫描补 task_plan_store/characterlib_handler/gaea_ui 三组；新增 13 测试。
- 验证：test-all.ps1 109/109 包 ok、go vet 干净；tsc 0 errors、vitest 274/274、eslint 0 errors（749 存量 warnings）。
- 明确不做（留后续刀）：绑定签名变更（T6-3）、emit done 语义（T6-3）、config 原子写（T6-9.4）、非流式重试/useController 降级 catch（T6-10）。

## v2.23.0「运行纵深 · 进料与质量」（2026-08-14）
> 阶段 5 第三刀（T5-5 + T5-6）：成本库进料闭环、检索统一与质量回归。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.23.0.md。
- T5-5 成本库进料闭环（唯一能力补全项，属既有成本库模块内）：
  - **PDF/图片报价单本地识别入表**（GaeaCostImportVisionPreview）：PDF 文本提取（docmd）→
    扫描件本地 OCR（OvisOCR2→Windows OCR 兜底）→ 表格线启发式解析（名称/规格/单位/价格表头，
    回退整行解析）；AI 字段归一化走本地通道（sensitive_local 强制本地，不可用降级规则解析并注明）；
    复用候选预览确认流程（无确认不落库不变），Preview.source 标记识别来源（pdf_text/pdf_scan/image）；
  - **供应商比价**（GaeaCostCompare）：库内现价/价格源抓取候选/历史快照三源聚合 + 相对现价
    跳幅 diffPct（复用 DetectAnomalies 算法）；前端比价弹层（CostCompareModal：来源/期数/价格/
    跳幅着色 ≥20% 红 / >5% 琥珀，空态提示）；
- T5-6 检索统一与质量回归：
  - **统一检索入口**（GaeaUnifiedSearch）：一次调用同时出关键词全文 + 跨库语义（已跨
    cost/knowledge/office/file）两组结果；办公搜索面板「跨库」模式单框两段展示；
    原绑定收敛为共享实现委托（单一来源）；
  - **检索质量受控测评**（GaeaRetrievalEvalRun）：真实业务查询集（docs/retrieval-eval-set.md，
    12 条造价/工程域查询 + 19 个预期命中标注）→ 逐条 Recall@10 → 汇总平均，门槛 0.8；
    模型中心「检索质量」区一键运行 + 逐查询明细表；补上「受控测评只测速度不测召回」缺口；
- 测试：检索测评 4 + 统一检索 3 + 识别/比价多组 + 既有检索回归 8/8；Go 全量 90/90 包 ok、
  vet 干净、gen_bindings 457 方法 → 10 门面；前端 tsc/eslint 0 errors、vitest 265/265；冒烟通过

## v2.22.0「运行纵深 · 速度与韧性」（2026-08-14）
> 阶段 5 第二刀（T5-3 + T5-4）：本地模型调度纵深、中断续跑。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.22.0.md。
- T5-3 本地模型调度纵深：
  - **保活 keep-warm**：每 5 分钟对 catalog Running 的本地模型发轻量 SSE 探针（max_tokens=8，
    防 herdsman 卸载空闲模型）；探针失败自动降级跳过直至重新运行；开关持久化
    （keep_warm_enabled，模型中心「本地调度」设置区）；
  - **启动自动预载**：启动后后台预载功能绑定（gaea→office→chat 优先级）第一个 herdsman 模型
    （installed 且未 running 时 start --wait），首次对话免冷启动；开关 auto_preload；
  - **换模预计等待**：GaeaModelSwitchEstimate（hot/cold/download/unknown 四态，cold 提示实测
    约 15-20 秒），前端模型切换器选本地模型时弹确认；
  - **KV 缓存命中率 KPI**：gaea 侧调用记录上报 cache hit/miss token（DeepSeek/OpenAI 两种
    usage 风格归一，未上报时不污染命中率），「本地 vs 云端」统计卡新增缓存命中率
    （全局 + 云端/本地拆分）；
- T5-4 中断续跑：
  - 任务级（v2.21.0 已交付）；**agent 会话级**：每轮 turn 在 session sidecar
    （<session>.state.json）标记 running/完成 + 最后进度摘要；进程被杀残留 running=true →
    会话列表「未完成」徽标；恢复会话自动注入「上次中断于 <摘要>，请先总结进度再继续」
    并清除标记（含启动自动恢复路径）；
  - **轻语长流程**：任务计划持久化到 whisper_data/task_plan.json（重启不丢），
    WhisperTaskPlanStatus/Resume 恢复入口；
- 测试：session state 5 + controller 中断 4 + app 注入 5 + schedule 14 + stats 缓存 3 +
  overview 命中率 2 + whisper taskplan 10 + config 开关 2；Go 全量 90/90 包 ok、vet 干净、
  gen_bindings 453 方法 → 10 门面；前端 tsc/eslint 0 errors、vitest 255/255；冒烟通过

## v2.21.0「运行纵深 · 调度与异步化」（2026-08-14）
> 阶段 5 第一刀（T5-1 + T5-2）：通用任务调度器 + 批处理队列、实时文件监听。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.21.0.md。
- T5-1 通用任务调度器 + 批处理队列：新包 internal/gaea/tasks（Hephaestus.db SchemaV8 tasks 表）——
  状态机 queued→running→succeeded|failed|cancelled、进度 0-100 + 消息、取消（context 传播）、
  自动重试（指数退避）、手动重试、**重启续跑**（Startup 恢复 running→queued 重新排队）；
  进度事件经 gaea-task 通道实时推送（节流 400ms，终态必达）；
  App 绑定 GaeaTaskList/Cancel/Retry（446 方法 → 10 门面，gen_bindings 重新生成 + 完备性测试）；
  **价格抓取全异步化**（单源/一键全部/30 分钟定时 cron 全部走任务队列，同源去重、逐源进度、
  失败明细在任务结果；顺带修复 SaveFetch 按值拷贝致返回记录 ID 恒为空的历史缺陷）；
  **文件索引重建异步化**（分批 Ensure 进度、末批 Stale 清理、手动/轮询/监听共用队列去重）；
  办公右栏新增「任务」Tab（任务中心：活动/历史分组、进度条、取消/重试、失败原因）；
- T5-2 实时文件监听：新包 internal/gaea/filewatch（fsnotify 监听工作区目录树，2s 去抖合并输出
  变更/删除批次；目录级变更与事件风暴>50 标记全量重建；监听异常记录 WatchErr 回退轮询）；
  增量索引（删除直接清向量 semantic.Remove、变更内容感知重嵌、失败自愈全量重建）；
  10 分钟轮询降级兜底（监听健康时跳过）——**新文件秒级可搜**；
- 测试：tasks 包 13 组 + filewatch 包 5 组 + App 层 6 组（单源任务流/同源去重/一键抓取/
  List-Cancel-Retry 链路/定时到期跳过/cron 去重）；Go 全量 90/90 包 ok、vet 干净；
  前端 TaskCenter/价格面板/搜索面板用例 + tsc/eslint/vitest 全绿

## v2.20.1「数据可迁移·独立审查修复」（2026-08-14）
> 对 v2.20.0 变更面做独立子代理代码审查，修复 3 高危 + 4 中危 + 多项低危缺陷。
> 详见 releases/v2.20.1.md。
- 高危 #1：ApplyPending 部分失败后重试必失败 → 重构为两阶段幂等（先移走当前数据、再应用 staging，src 缺失视为已应用跳过），重试可成功且不破坏数据
- 高危 #2：home-config 从不恢复（被主循环 rename 走）→ 主循环排除 home-config 单独处理；修复 HomeConfigRel 缺 . 前缀导致的恢复到错误文件名
- 高危 #3：SQLite 快照连接无 busy_timeout + 静默回退复制缺 WAL → 加 _busy_timeout=5000 + 重试；回退改为 checkpoint 后复制；manifest 增 Warnings 告警字段
- 中危 #4：恢复失败不可见 → 失败路径也写 .restore-result.json（前端失败告警可达）
- 中危 #5：已有 pending 时再次 Restore 可堆叠/覆盖 → 拒绝并提示先取消；staging 加随机后缀防同秒撞名；Cancel 清理孤儿 staging
- 中危 #6：dirSize 全量递归阻塞 UI → 目录大小缓存（mtime + TTL 失效）
- 中危 #7：宣称可回滚但无回滚代码 → 新增 GaeaDataBackupRollback（.restore-before 移回）+ 前端失败告警「回滚到恢复前」按钮
- 低危：#8 盘符路径拒绝、#9 恢复二次确认 + zip 校验、#10 备份文件名防同秒覆盖、#11 before 保留 2 份、
  #12 WritePending 原子写、#13 Extract 两阶段、#15 shouldSkip 精确化、#16 pending 错误透出、#17 entries 数组校验
- 测试：backup 包 9 组（新增重试幂等/home-config 恢复/盘符拒绝）、App 层 5 组（新增已有 pending 拒绝/Rollback）；
  go 全量 ok、tsc/eslint 0 errors、vitest 251/251

## v2.20.0「个人使用收口·数据可迁移」（2026-08-14）
> 长期规划阶段 4 按「个人使用、不商用」重新定标（用户 2026-08-14 决策）：
> 删除商用分发项（安装器/自动更新/代码签名），聚焦个人使用最需要的
> 数据可迁移（一键备份/恢复）、模块收口（微信 beta/移动端冻结）、磁盘治理（保留 5 版约定）。
> 详见 releases/v2.20.0.md。
- P4-3 数据可迁移：设置页新增「数据」分类——一键备份（Hephaestus.db 记忆/知识/成本/语义向量 +
  whisper_data 轻语/办公/角色库/聊天 + config.toml + sessions + home 配置 → zip + manifest；
  SQLite 用 VACUUM INTO 一致性快照，运行中备份安全）；从备份恢复（两阶段：校验解压 staging +
  写 pending 标记 → 重启后自动应用，应用前先自动备份当前数据到 .restore-before-<时间>，失败可找回）；
  恢复结果提示 + 待应用告警与取消
- P4-1 模块收口：微信通道标注「个人使用实验功能（beta）」；移动端访问标注「已冻结」
- P4-2 发布形态简化：删除安装器/自动更新/代码签名（SAC/SmartScreen）等商用项；升级 = 替换 exe +
  数据备份先行（数据在用户目录不受影响）
- P4-4 磁盘治理：releases 保留最近 5 版约定明确化（README 版本表）
- 测试：Go backup 包 6 组（打包/解压往返、VACUUM INTO 快照数据完整性、manifest 校验、
  zip-slip 防穿越、pending 应用往返、SHA256）+ App 层 3 组（Info/Create/Restore/Pending/Cancel 链路、
  拒绝非法 zip、Startup 钩子）；前端 DataPanel 4 组 + SettingsPage 更新（新增「数据」分组断言）
- 个人使用声明：数据全部在本机；API 凭证（DPAPI 加密）跨机器不可解密，换机需重填

## v2.19.0「数据与成本纵深·补测评缺口」（2026-08-14）
> 长期规划阶段 3 第二刀（D3-4）：受控测评补上「看得见的缺口」——
> 报告增加每模型/长上下文/缓存复用/显存参数专项分析；新增压力专项任务
> 预设与「快速流式探针」（断流/卡顿观察）。详见 releases/v2.19.0.md。
- D3-4 报告模板化：Markdown 报告新增 5 个专项段落——每模型对比（同任务横向可比）、
  长上下文专项（TTFT vs context_size）、缓存复用专项（first vs second TTFT +
  prefill 加速比）、显存相关启动参数（effective_launch_params 关键字段）、并发专项说明
- D3-4 压力预设：受控测评任务集新增「压力·长上下文 / 压力·长输出 / 压力·显存
  （长上下文+长输出）」3 项（配合上下文 4K~32K / 并发 1/2/4 / max_tokens 使用）
- D3-4 流式探针：GaeaBenchmarkStreamProbe 对模型发起真实 SSE 请求，观察 TTFT、
  分块数、最大/平均分块间隔（卡顿指示）、是否正常 [DONE] 收尾（断流检测）；
  模型中心「受控测评」新增「快速流式探针」区（每已安装模型一键探测）
- 测试：Go 新增流式探针 3 组（SSE mock/参数校验/HTTP 错误）+ 报告专项分析 1 组；
  前端 BenchmarkSection 新增探针用例（4/4）；全量 vitest 247/247
- 真实端到端：对运行中 herdsman 的模型探测成功（冷启动 TTFT 15.2s、60 块、
  max_gap 83ms、正常收尾）

## v2.18.0「数据与成本纵深·首轮」（2026-08-14）
> 长期规划阶段 3（D3-1 ~ D3-3）：跨库统一语义检索补齐「资料」+ 索引状态、
> 本地 vs 云端分流统计与节省对比、Herdsman 受控测评产品化（一键发起/明细/报告导出）。
> 详见 releases/v2.18.0.md。
- D3-1 持久化向量索引：跨库统一语义检索（GaeaSemanticSearch）并入工作区资料（kind=file，
  复用文件索引定时维护的持久化向量，检索不扫描）；新增 GaeaSemanticIndexStatus（各库向量
  条数，记忆中枢可见索引健康度）；semantic.Store 新增 Counts
- D3-2 分流统计面板：GaeaUsageOverview 打通 gaea 侧调用记录（含费用估算）与 herdsman
  events.jsonl 本地遥测——本地 token 口径 = events 全量 + 其他本地引擎（herdsman 不重复计）；
  模型中心「调用统计」新增「本地 vs 云端·节省对比」卡（云端实际混合单价折算，无云端用量
  回退 deepseek-v4-flash 官价）
- D3-3 测评产品化：复用 herdsman /api/benchmarks（多模型 × 变体 × 上下文长度 × 并发，
  逐 case TTFT/TPS/token）——GaeaBenchmarkList/Start/Detail/Export + 模型中心「受控测评」
  分类（任务预设蒸馏自 120 组对照测评方法学、上下文 4K~32K 覆盖长上下文、并发 1/2/4、
  Markdown 报告导出含逐用例明细）；真实 herdsman 端到端验证（发起 202 + 完成 succeeded）
- 测试：Go 新增分流口径 3 组 + 测评 runs 解析/HTTP 列表发起/导出 4 组 + semantic Counts；
  前端 BenchmarkSection vitest 3/3；internal/app 全量 ok、tsc/eslint 0 errors、vitest 246/246

## v2.17.0「安全与架构收敛」（2026-08-14）
> 长期规划阶段 2（S2-1 ~ S2-4）：安全收敛与绑定面架构拆分——
> LAN 暴露告警上墙、WebView2 远程调试默认关闭、HTTP 桥接一次性 token、
> 敏感数据本地通道、429 个导出方法按板块拆 10 个绑定门面。
> 详见 releases/v2.17.0.md。
- S2-1 LAN 风险处置：全局安全横幅（启动即检测 herdsman api.lan_accessible，暴露时醒目告警 +
  中文处置指引 + 重新检测/本次忽略）；设置页新增「安全」分类（同面板可复核）
- S2-2 gaea 自身安全开关：WebView2 远程调试（9333）改 `GAEA_WEBVIEW_DEBUG=1` 才开启，默认关闭；
  HTTP 调试桥接加一次性 token（`GAEA_HTTP_TOKEN` 或每进程自动生成并打日志，/api/rpc 与
  /api/stream 须携带 Bearer/X-Gaea-Token/?token=，/api/health 保持开放），前端桥接自动透传
- S2-3 App 绑定面拆分：429 个导出方法按板块拆 10 个绑定门面（CoreB/OfficeB/MemoryB/CostB/
  ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB），方法体零改动纯委托，脚本生成
  （scripts/gen_bindings）+ 反射完备性测试兜底（App 方法集与门面并集全等）；
  前端 gaea/lib/bridge.ts 与 api/bridge.ts 单点路由，旧调用路径经 wailsjsCompat 兼容层零改动
- S2-4 敏感数据本地通道：新增「敏感域本地化」开关（默认开启，~/.gaea_config.json
  sensitive_local 持久化）——成本/报价类 AI 操作（GaeaCostImportAIParse）默认强制路由本地
  Herdsman（数据不出本机），引擎不可用自动回退常规路由；设置页「安全」分类可切换回云端
- 测试：internal/app 全量 19.8s ok（含绑定完备性）；config/httpbridge 新增 token 鉴权、
  sensitive_local 往返用例；tsc -b 通过、eslint 0 errors、vitest 243/243、vite build 通过

## v2.16.1「模型中心资源协同 + 磁盘治理」（2026-08-14）
> 长期规划 E1-4：对齐 herdsman `model_scheduling.local_concurrency=1` 的调度现实，
> 生命周期操作串行化（批量启停天然变有序队列）；模型库新增磁盘 KPI
> （已装占用 + 数据目录所在卷余量）。详见 releases/v2.16.1.md。
- 后端：`herdsmanOpMu` 串行化 Start/Stop/Download/Uninstall（下载最长 60 分钟、冷启动 20 分钟，
  并发发起会互相冲突）；`herdsmanDiskInfo`（x/sys/windows GetDiskFreeSpaceEx，可注入替身测试）+
  HerdsmanCatalog 新增 installed_bytes/disk_total/disk_free/disk_error，探测失败不阻塞目录
- 前端：模型库 KPI 新增「已装空间」「磁盘余量」（余量/总量，含探测失败提示）；fmtSize 补 TB 档
  （此前 1TB 显示为 1024.0 GB）
- 测试：磁盘解析/汇总/降级 3 组 + 操作串行化并发验证（8 goroutine × 4 操作，最大在飞必须为 1）；
  模型库 vitest 5/5；Go 变更面全绿

## v2.16.0「Herdsman 底座加固 + 工程门禁」（2026-08-14）
> 长期规划首轮（docs/superpowers/plans/2026-08-14-gaea长期规划-herdsman底座加固与工程门禁.md）：
> gaea 的本地能力链（聊天/视觉/embedding/rerank/OCR/文档解析/ASR/TTS/生图/翻译）
> 全部挂在 Herdsman 服务上，本轮把它变成「可探测、可探活、可告警」的受管底座，
> 同时修复前端 CI 门禁（此前 lint 因插件缺失直接崩溃、continue-on-error 形同虚设）。
> 详见 releases/v2.16.0.md。
- H0-1 环境探测与兼容契约：`internal/herdsman/probe.go` + `App.HerdsmanProbe`——一次探测
  config.yaml（api 段）、herdsman.exe CLI 可找到性、/v1/models 可达性（HERDSMAN_PROBE_LIVE=1 时真实探测）、
  四个数据契约（launch_records/model_stats events.jsonl/skill-operations.json/models），
  输出结构化 Probe + 中文告警清单（含 LAN 暴露、端口漂移、契约缺失）
- H0-2 服务健康检查：`internal/herdsman/health.go` + `App.HerdsmanHealth`——端口拨测（1s）+ API 存活（3s）+
  按能力归类已装模型（chat/vision/embedding/rerank/ocr/parse/asr/tts/imagegen/translation），
  Healthy=端口+API+聊天模型齐备，Summary 中文问题清单
- H0-3 TTS 默认模型动态解析：`voice.ResolveHerdsmanTTSModel`——配置值已装则用配置值，
  否则按优先级（voxcpm2 第一，本机实测唯一可用本地 TTS）从已装列表解析，
  VoiceGetSettings 返回回退标记供前端提示；修复「默认 qwen3-tts-customvoice 未安装必然先失败」
- H0-4 LAN 暴露检测与告警：`internal/herdsman/lancheck.go` + `App.HerdsmanSecurityCheck`——解析
  herdsman config.yaml 的 api.lan_accessible/port（逐行 YAML 解析，零依赖），暴露时返回中文处置指引；只提示不改配置
- H0-5 模型用途建议 + 思考模式守护：模型库卡片新增「用途建议」（`herdsmanModelHint`，依据 120 组受控测评：
  HauhauCS 日常/识图首选、LynnStyle 可审计推理、Hermes 勿开思考、voxcpm2 冷启动 50s 等）；
  本地引擎思考模式 max_tokens <4096 自动抬到 4096（bridge 与聊天流式两路径），杜绝「只有推理、无正文」
- E1-1 前端 CI 门禁修复：eslint 配置修复（react-hooks v5 flat 配置失效 + 缺 eslint-plugin-react-hooks/refresh、
  新增 lint script、globalIgnores 生成目录）；28 个硬错误清零——含 Lightbox 12 处条件调用 Hook 的真实隐患
  （顺带修复该场景下 React 运行时崩溃风险）、CreatePage case 声明、Markdown 控制字符正则等；
  存量高频风格规则（no-explicit-any 等 6 项）降为 warn 随迭代清理；CI 移除 continue-on-error、
  npm ci→npm install（仓库不提交 lockfile）、新增 vitest 步骤；修复 ComfyUI 路径文案反斜杠丢失（用户可见）
- E1-2 发布冒烟脚本：`scripts/smoke.ps1`——产物启动 + HTTP 桥接 /api/health 探活 + 进程存活 + 自动回收
- E1-3 版本节奏：本轮起转周版本，v2.15.7 后直接 v2.16.0
- 测试：internal/herdsman 全包单测（probe/lancheck/health/归类函数）；Go 相关包全绿（internal/app 22s 全量、
  bridge/ai/voice/tts）；tsc -b 通过、eslint 0 errors（725 warnings 为存量风格项）、模型库卡片 vitest 5/5

## v2.15.7「通用办公 P0 · 开工前计划卡片结构化」（2026-08-13）
> 接回通用办公优化路线图，把 v2.10 的「开工前计划确认」升级为结构化计划卡片：
> 计划生成改走严格 JSON，后端解析为「任务理解 / 步骤（资料·工具·产出物）/ 待确认」，
> 前端渲染专属计划卡片，解析失败自动回退纯文本。详见 releases/v2.15.7.md。
- 后端：planSystemPrompt 改为严格 JSON；新增 agent.ParsePlan / RenderPlanMarkdown（容错代码围栏、清洗空字段）；
  Ask 事件新增可选 Plan 结构化载荷（controller_plan 下发、gaeaEventMap 序列化），答案协议不变
- 前端：WireAsk 新增 plan 载荷；AskCard 渲染 PlanBody（目标高亮卡 + 步骤编号卡 + 资料/工具/产出物芯片 + 待确认琥珀提示），无 plan 回退 Markdown
- 测试：Go 新增 ParsePlan 4 例 + Ask.Plan 随事件下发 1 例；前端 AskCard 2 例（结构化/回退）；go vet + go test 全绿，Vitest 241→243，tsc/vite build 通过

## v2.15.6「Herdsman 深挖 P5 · 数字生命记忆联动 + 最近操作」（2026-08-13）
> 完成 Herdsman 深挖路线图收尾：把 digital-life 虚拟人格记忆（角色/关系/记忆摘要/
> 时间线/世界事件）只读接进记忆中枢，并展示 Herdsman 最近异步操作。
> 详见 releases/v2.15.6.md。
- 后端：App.HerdsmanDigitalLife（只读 life.sqlite3：角色×关系×记忆摘要合并、计数、最近时间线/世界事件）+ App.HerdsmanOperations（skill-operations.json 最近 20 条）
- 前端：记忆中枢新增「数字生命」库（角色卡片：亲密度/信任/安全条 + 摘要/高亮/强化值；最近时间线/世界事件；最近 Herdsman 操作列表）
- 测试：Go 新增数字生命/操作解析用例；前端 DigitalLifeLibrary 2 例，Vitest 239→241 全绿

## v2.15.5「Herdsman 深挖 P4 · 检索升级 + 调用统计」（2026-08-13）
> 承接 v2.15.4，落地 P4：语义检索动态升级到 qwen3-embedding-4b / qwen3-reranker-4b
> （装了自动用，没装回退 bge），并把 Herdsman 逐请求遥测接进模型中心。
> 详见 releases/v2.15.5.md。
- 检索升级：resolveHerdsmanSearchModel 动态选模型（env > qwen3 系已装 > bge 回退），覆盖成本库/知识库/办公记忆/工作区文件语义索引
- 调用统计：App.HerdsmanModelStats 聚合 model_stats/events.jsonl（调用/成功失败/token/耗时/TTFT/TPS），模型中心「模型库」新增本地统计面板（KPI + 明细表）
- 本机：qwen3-embedding-4b（2.5GB）与 qwen3-reranker-4b（2.7GB）已下载并启动；发现 Hy-MT1.5:1.8B 翻译模型已装，translate_text 自动切专用模型
- 测试：Go 新增 stats 解析/动态选型用例；前端统计面板断言；go vet + go test + tsc + Vitest 239 全绿

## v2.15.4「Herdsman 深挖 P3 · 本地翻译」（2026-08-13）
> 承接 v2.15.3，落地 P3：本地翻译能力——优先 Hunyuan-MT / Hy-MT 翻译模型
> （capability=translation），未安装时回退「常规办公」模型，本地/免费优先。
> 详见 releases/v2.15.4.md。
- 后端 `App.LocalTranslate`：翻译模型发现（hunyuan-mt/hy-mt）+ 显式 model + 回退常规办公模型（used_fallback 标注）+ 可读错误引导；文本翻译走 /v1/chat/completions（/v1/translations 是语音翻译）
- 办公专业工具 `translate_text` 注入 ExtraTools，能力面板「本地专业模型」新增入口
- 测试：Go 新增 6 例（模型命中/显式/回退/空文本/发现/工具执行），go vet + go test 全绿；tsc + Vitest 239 全绿

## v2.15.3「Herdsman 深挖 P2 · 模型生命周期管理」（2026-08-13）
> 承接 v2.15.2 模型库，本轮把「看」升级为「管」：模型卡片直接启动/停止/下载/卸载
> Herdsman 模型，并读取 launch_records 生成这台机器的启动参数预设。
> 详见 releases/v2.15.3.md。
- 后端：HerdsmanModelStart/Stop/Download/Uninstall（skill models 子命令，--wait 长超时）+ HerdsmanLaunchPresets（读 launch_records 取最近成功启动参数）+ HerdsmanOpResult 统一结果
- 前端：模型库卡片操作（运行中→停止；已安装→启动+卸载二次确认；未安装→下载），操作中 loading + 完成后自动刷新；有 launch_records 的模型显示「启动预设」徽标（悬停见参数明细）
- 测试：Go 新增操作结果/预设解析与生命周期 handler 用例；前端新增生命周期 1 例，Vitest 238→239 全绿

## v2.15.2「Herdsman 深挖 P1 · 模型库」（2026-08-13）
> 启动 Herdsman 深挖路线图（docs/superpowers/plans/2026-08-13-herdsman-deep-dive.md），
> 本轮把 Herdsman 完整本地模型目录（90 个已知模型）接进模型中心：从「只能连已存在
> 的模型」升级为「可浏览全部可安装模型与能力」。详见 releases/v2.15.2.md。
- 后端 `App.HerdsmanModelCatalog()`：调 `herdsman.exe skill models list --json`（RPC），解析 90 模型目录（能力/安装/运行/量化/变体/大小/MoE），汇总计数；HERDSMAN_EXE 环境变量优先，回退默认安装路径；CLI 缺失或 Herdsman 未运行返回可读错误，不阻塞模型中心
- 前端模型中心新增「模型库」分类：KPI（已知/已安装/运行中）+ 搜索（名称/能力）+ 状态过滤 + 类型下拉 + 模型卡片（中文名/能力/量化/大小/参数/MoE/状态），复用统一卡片与视觉 token
- 测试：Go 新增解析/排序/汇总/错误/CLI 定位用例 + 真实 90 模型回归校验；前端新增模型库 4 例，Vitest 234→238 全绿，tsc 通过

## v2.15.1「通用办公 · 产物与资料体验收口」（2026-08-13）
> 承接 v2.15.0 模型中心，回头收口通用办公的产物展示一致性与入口兜底：
> 右侧「会话产物」面板补齐图片缩略图与一键复制全部路径，Ctrl+K 命令面板补齐
> 资料/产物/变更跳转，欢迎页任务模板在命令库为空或加载失败时回退内置模板。
- 会话产物面板：图片类交付物渲染缩略图（与对话内交付卡共用 FileThumb，加载失败回退图标）；头部新增「复制全部文件路径」，一次拿到本次会话全部交付物清单
- 命令面板：新增「资料面板」「产物面板」「变更面板」跳转（与已有 文件/统计 面板项对齐）
- 欢迎页：任务模板新增内置兜底（周报/会议纪要/成本测算/方案大纲/数据分析/文档转换/报告拼装/演示文稿），首启或离线不再空白
- 测试：前端新增 5 例（产物缩略图/复制全部/模板兜底），Vitest 229→234 全绿，tsc -b 通过

## v2.15.0「模型中心 P0/P1/P2 + UI 重设计」（2026-08-13）
> 按市场调研（Open WebUI / Cherry Studio / Dify / Jan / Ollama 生态）系统优化模型中心，
> 并做浅色/深色双主题重设计。详见 releases/v2.15.0.md。
- 模型中心 P0：引擎状态→模型可见性联动（未连接模型置灰/禁用动作）；测试连接诊断（延迟+失败原因）；功能绑定回退态 + 一键重置
- 模型中心 P1：模型网格搜索 + 收藏置顶（localStorage 持久化）；本地资源占用可视化（CPU/内存/GPU/显存 + 本地引擎状态）；模型选择器统一按后端过滤 + 兜底
- 模型中心 P2：引擎批量启停 + 隐藏已停用引擎；调用统计收进抽屉
- UI 重设计（redesign-existing-projects / ui-ux-pro-max 审计）：模型卡片/空状态/面板背景与阴影改用 gaea 主题 token，适配浅色/深色
- 测试：前端 Vitest 229/229；tsc -b、vite build 通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea-v2.15.0.exe 同步桌面与 releases/

## v2.14.12「绘梦 UI 重构落地 + herdsman 生图能力修复」（2026-08-13）
> 完成绘梦板块 UI 全量重构（设计文档 Phase 1-2 + 视觉统一），选项改下拉并修复
> WebView2 弹层卡首帧；herdsman 生图链路修复与图生图接入。详见 releases/v2.14.12.md。
- 绘梦：左栏可折叠分区（基础设置/模型与引擎/画幅与输出/高级参数）、底部常驻生成栏、右侧任务中心三 Tab、玻璃 HUD 视觉统一
- 绘梦：引擎/模型（可搜索）/画幅/时长帧率改下拉；WebView2 下拉弹层兜底（popupClassName 禁用弹层动画 + getPopupContainer=body）
- herdsman：size 契约对齐（文档支持 size）、URL 响应转 data URL、图生图接入 `/v1/images/img2img`（JSON + image 字段）
- 测试：前端 Vitest 204/204；tsc -b、vite build 通过；Playwright 无头断言 32 项通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea-v2.14.12.exe 同步桌面与 releases/

## v2.14.11「小说板块后端 + 绘梦生成链路闭环」（2026-08-13）
> 小说创作补齐章节流式/状态/书架/世界观/统计/导出后端；绘梦补齐生成队列/取消、历史元数据持久化、
> ComfyUI 任务进度实时反馈、模板与绘照分配、LoRA 动态加载/重试；绘梦 UI 重构已立项，下个会话执行。
> 详见 releases/v2.14.11.md。
- 小说：章节流式创作/状态流转、书架、世界观（含回归测试）、统计、导出收敛
- 绘梦：生成队列/取消、历史元数据持久化、ComfyUI 实时进度、模板/绘照、LoRA 动态加载与重试
- 测试：前端 Vitest 200/200；tsc -b、vite build 通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea.exe 同步桌面

## v2.14.10「修复办公模型改绑不生效（仍沿用旧模型）」（2026-08-13）
> 定位：办公主 agent 的模型由 bridge provider 在 GaeaInit 时注入并缓存，运行时
> 在模型中心改绑「办公」后，配置已更新、前端也显示新模型，但没触发重新注入 +
> 重建 controller，因此继续用旧的 deepseek。详见 releases/v2.14.10.md。
- `App.SetFeatureModel` / `App.SetFeatureModelEnabled` 覆盖：feature=="gaea" 时重新注入 bridge 模型
- 办公引擎已初始化时同步重建 controller；未初始化则仅更新注入，下次 GaeaInit 生效
- 测试：新增 `TestAppSetFeatureModel_GaeaBindingApplies`
- 验证：go build/vet、go test（chat/app）全绿；wails build 通过（前端未改动）

## v2.14.9「聊天板块后端：原子落库 + AppendMessage 收敛」（2026-08-13）
> 继续收敛聊天后端写入路径：用户/助手消息改为单事务原子落库，AppendMessage 用
> RETURNING 一次拿回 id+seq，失败不再静默忽略。详见 releases/v2.14.9.md。
- `AppendMessage`：用 `INSERT ... RETURNING id, seq` 替代「先查 MAX(seq)+1 再插入」，消除竞态窗口
- 新增 `chat.Store.AppendExchange`：单事务原子写入用户 + 助手消息并刷新 updated_at
- `appendChatExchange` 改走事务，落库失败记录错误日志（不再静默丢弃）
- 测试：新增 `TestStore_AppendExchange`（含不存在话题回滚校验）
- 验证：go build/vet、go test（chat/app）全绿；前端未改动

## v2.14.8「聊天板块后端：会话列表查询收敛 + GetTopic」（2026-08-13）
> 收敛聊天板块后端存储的查询路径：会话列表预览从 N+1 改为单条相关子查询，
> 新增 `GetTopic` 供创建/导入/导出直接按 ID 读取，不再全表列举。详见 releases/v2.14.8.md。
- `chat.Store.ListTopics` 用相关子查询一次取回所有话题的预览，消除 N+1
- 新增 `chat.Store.GetTopic(id)`；`ChatTopicCreate` / `ChatImportTopic` / `ChatTopicExportMarkdown` 改走按 ID 读取
- 测试：新增 `TestStore_GetTopic`
- 验证：go build/vet、go test（chat/app）全绿；前端未改动

## v2.14.7「聊天板块交互收尾：清空确认 + 切换聚焦 + 标题生成收敛」（2026-08-13）
> 收尾几处日常交互细节：清空对话加二次确认防误触，切换话题后自动聚焦输入框，
> 会话标题生成抽成纯函数并补测试。详见 releases/v2.14.7.md。
- 清空当前对话加 Popconfirm 二次确认（避免误清空不可恢复）
- 选中话题后自动聚焦输入框，切过去即可输入
- 会话标题生成抽纯函数 `autoTopicTitle`，前端测试 194→196
- 验证：wails build（含 tsc + vite）通过；vitest 196；go build/vet、go test 全绿

## v2.14.6「聊天板块会话导出为 Markdown」（2026-08-13）
> 补上聊天板块的内容出口：把当前会话导出为 Markdown 文件，落盘到用户数据目录，
> 前端一键导出并复制文件路径。详见 releases/v2.14.6.md。
- 后端 `ChatTopicExportMarkdown`：按话题标题 + 用户/AI 分段导出全部消息为 .md
- 文件名安全规整（`sanitizeChatFilename`），写到用户数据目录 exports/chat 下
- 前端模式栏新增「导出」按钮，成功后提示路径并复制到剪贴板
- 测试：新增 `TestChatTopicExportMarkdown`
- 验证：wails build（含 tsc + vite）；vitest 194；go build/vet、go test（app）全绿

## v2.14.5「聊天板块真实流式输出（普通对话）」（2026-08-13）
> 把普通对话从「整段返回 + 前端模拟打字流」升级为后端逐块流式下发，首字更快、
> 停顿更自然；思考链也随流式下发。角色模式保持整段返回。详见 releases/v2.14.5.md。
- 普通对话真实流式：`ChatStreamPlain` 立即返回 runID，经 `chat-stream:<runID>` 下发 delta/reasoning/done/error
- 角色模式保持整段返回（沿用原有模拟打字流），两种模式统一走发送收尾（自动命名/置顶/预览同步）
- AI 客户端新增 `ChatStreamChunks`：复用同一套请求准备，但暴露底层 SSE 分块供逐块消费
- 测试：新增 `TestChatStreamChunks_EmitsDeltas`、`TestChatStreamPlain_StreamsAndPersists`
- 验证：wails build（含 tsc + vite）通过；vitest 194 例全过；go build/vet、go test（ai/app）全绿

## v2.14.4「聊天板块收口：联网搜索污染修复 + 回到底部 + 侧栏预览同步」（2026-08-13）
> 收口聊天板块最后一处数据正确性问题与一个滚动体验缺口：联网搜索注入只进模型
> 上下文、不再污染用户历史；上翻阅读时提供「回到底部」悬浮入口；侧栏预览随发送/
> 清空即时同步。详见 releases/v2.14.4.md。
- 修复：联网搜索注入不再写入用户消息历史（原实现会把搜索结果一起落库，污染话题预览与历史）
- 回到底部：上翻阅读时出现悬浮按钮，一键回底并恢复自动跟随
- 侧栏预览同步：发送首条消息 / 清空对话后即时更新会话预览，不再等重新进入
- 测试：新增 Go 用例 `TestChatSend_Plain_SearchKeepsOriginalUserMessage`
- 验证：tsc + vite build 通过；vitest 194 例全过；go build/vet、go test（含新用例）全绿

## v2.14.3「聊天板块补强：输入法防误发 + 快速切话题竞态修复 + 模式栏收敛」（2026-08-13）
> 延续聊天板块优化：修掉中文输入法候选确认时 Enter 误发消息、快速切换话题时
> 旧话题响应覆盖当前视图两个高频隐患，并收敛模式栏重复 JSX。前端用例 190→194。
> 详见 releases/v2.14.3.md。
- 输入法防误发：中文/日文 IME 组合态下的 Enter 不再触发发送（纯函数 `shouldSubmitOnEnter`）
- 快速切话题竞态修复：话题消息载入用序号令牌，过期响应直接丢弃，避免旧消息覆盖当前视图
- 模式栏收敛：普通/角色两分支合并为单一操作区，去除重复 JSX
- 测试补强：新增 `shouldSubmitOnEnter` 用例
- 验证：tsc + vite build 通过；vitest 190→194；go build/vet、go test 保持全绿

## v2.14.2「聊天板块优化：会话搜索 + 最近活跃排序 + 智能滚动 + 重开加载修复」（2026-08-13）
> 在通用办公会话化升级之后，回头把「聊天板块」的日常交互体验补到同一档：会话按
> 最近活跃排序并显示相对时间、侧栏新增会话搜索、流式/生成时不再强制吸底、重进聊天
> 板块正确载入历史消息。前端用例 182→190。
> 详见 releases/v2.14.2.md。
- 会话列表：最近活跃优先（新会话/回复会话自动置顶），会话行显示相对时间
- 会话搜索：侧栏新增搜索框，按标题/预览/模式标签过滤（纯函数 `filterChatTopics`）
- 智能滚动：生成/流式输出时用户上翻阅读不再被强制吸底，贴近底部才恢复跟随（`isNearBottom`）
- 修复：重新进入聊天板块时，已选中话题的历史消息未载入（此前显示欢迎屏而非对话）
- 缺陷收口：会话重命名失败提示（不再静默失败）
- 测试补强：新增 `filterChatTopics` / `sortByUpdatedAtDesc` / `isNearBottom` 纯函数用例
- 验证：tsc + vite build 通过；vitest 182→190；go build/vet、go test（chat/app）全绿

## v2.14.1「办公板块缺陷收口 + 测试补强 + 结构收敛」（2026-08-13）
> 承接 v2.14.0：收口会话恢复/重命名失败提示、归档删除二次确认、欢迎页跨项目
> 最近会话、变更面板汇总排序、归档会话搜索；前端用例 138→179，办公前端做
> 首轮结构收敛，并收敛 bridge 动态/静态 import 混用告警。详见 releases/v2.14.1.md。
- 缺陷收口：会话恢复/重命名失败提示；归档会话永久删除二次确认；欢迎页最近会话跨项目；
  变更面板「N 个文件 · M 次」汇总与最近排序；侧栏搜索覆盖归档；会话删除三处注册表全量清理
- 测试补强：前端 138→179 例（store/ChangesPanel/useSessionManager/Sidebar/命令与 @ 解析/相对时间/变更汇总/项目分组搜索/能力面板摘要）
- 结构收敛：App 状态组件、buildSessionChanges、Composer 斜杠与 @ 解析、Sidebar 相对时间与搜索过滤、CapabilitiesPanel 摘要、bridge 静态 import
- 文档：herdsman API 文档更新（08-08 → 08-13）

## v2.14.0「办公板块会话化升级：项目分组 + 会话生命周期 + 任务目标 + 变更面板」（2026-08-13）
> 把通用办公从扁平会话列表升级为按项目聚合的会话工作台，补齐会话生命周期、
> 任务目标（需求→验收）与文件变更可观察性，并修复记忆图谱三元组被成本节点
> 挤掉、会话删除注册表清理不完整、App 层 toast 静默失效等问题。
> 详见 releases/v2.14.0.md。
- 侧边栏「项目」分组：当前工作区置顶，其余为最近打开且有会话的工作区；会话按“当前→置顶→最近”排序
- 会话生命周期：置顶（.pinned.json）、归档（移入 <sessions>/archive/，可恢复）、侧边栏「已归档」分组恢复
- 任务目标：会话首条用户消息自动锚定，随会话持久化，待办栏展示进行中/已验收并支持一键切换
- 文件变更面板：汇总写/改工具实际改动的文件及次数，覆盖 path/file_path/paths/edits/source+destination 等形态
- 恢复会话保留工具过程：GaeaHistory 还原 tool/tool_result，恢复后过程卡与变更面板仍可见
- 专注模式：Aim 按钮 / Ctrl+Shift+F 收起侧栏与右侧面板（状态持久化）
- 修复：会话删除三处注册表全量清理；记忆图谱轻语三元组提前于成本条目入图；App 层 toast 静默失效；恢复历史待办收尾；重复快捷键与 WRITE_TOOL_NAMES 去重

## v2.13.22「修复整轮结束后大过程卡折叠 / 小过程卡合并误展开」（2026-08-12）
> 根因：整轮结束后的合并「大过程卡」复用了运行中首段「小过程卡」的同一组件
> 实例（key 相同），小卡初始为折叠，导致大卡跟着折叠；上一版改为把该实例
> 撑开，又等于「小卡展开成大卡」，违背「小卡默认折叠」。
> 修复：合并大卡改用独立 key 全新挂载（天然默认展开），小卡实例始终折叠，
> 两者互不干扰。
> 详见 releases/v2.13.22.md。
- Transcript：非运行态（整轮结束）的分段改用 `done-` 前缀 key，合并大卡全新挂载
- ProcessCard：移除小卡→大卡实例复用的展开分支（小卡始终默认折叠）
- 回归测试：小卡挂载折叠、大卡挂载展开
- 验证：tsc + 前端 129 例全过；wails build 通过；产物 gaea-v2.13.22.exe

## v2.13.21「办公板块安全审计：封堵子代理绕过持久化写入审批」（2026-08-12）
> 审计发现最严重漏洞：默认 task 子代理继承全部工具（cost_save/remember/
> knowledge_add/promote_session_facts/install_skill）但运行在 headless 审批
> 通道上，可绕过主代理的逐条确认静默写入成本库/记忆/知识库/技能。
> 修复：子代理注册表剔除全部持久化写入工具；主代理的 forget/install_skill
> 一并纳入硬性逐条审批。其余复查（弹窗、上下文注入）无新问题。
> 详见 releases/v2.13.21.md。
- agent/task.go：FilterRegistry 剔除 cost_save/remember/forget/knowledge_add/promote_session_facts/install_skill
- control：hardAskTools 补 forget / install_skill
- 回归测试：子代理注册表不含持久化写入工具；默认子代理工具集更新
- 验证：go build/agent/control/permission 测试通过；wails build 通过；产物 gaea-v2.13.21.exe

## v2.13.20「记忆/知识库写入强制确认 + 记忆索引注入预算」（2026-08-12）
> 与成本库同源问题：remember / knowledge_add / promote_session_facts 由 AI 直接
> 落盘、无确认，写入一堆杂乱记忆并整体注入系统提示词占用上下文。现在这三个
> 工具与 cost_save 一样进入硬性逐条审批（任何权限级别含 yolo 都弹卡、批准仅
> 本次生效）；「Saved memories」索引注入增加预算（3000 runes 截断，其余用
> memory_search 按需查询），控制上下文占用。
> 详见 releases/v2.13.20.md。
- control：hardAskTools 扩展 remember / knowledge_add / promote_session_facts，审批摘要含条目名/描述/分类/来源等
- memory：capMemoryIndex 限制注入系统提示词的记忆索引长度（3000 runes）
- ApprovalModal：硬性审批工具统一隐藏「本会话允许」
- 回归测试：yolo 下记忆/知识写入仍审批、会话放行不记忆、审批摘要格式
- 验证：go build/control/memory 测试通过；前端 tsc + 128 例全过；wails build 通过

## v2.13.19「成本库写入强制逐条确认，AI 不再直接入库」（2026-08-12）
> 修复成本库被 AI 生成的虚高价格污染：此前权限级别为 auto/yolo 时所有工具
> 自动放行，cost_save 直接把数据写进成本库。现在 cost_save 成为硬性审批项——
> 任何权限级别（含 yolo）都必须弹审批卡，逐条确认条目名称/单价/单位/规格/
> 来源后才写入；不提供「本会话允许」，批准仅本次生效。
> 详见 releases/v2.13.19.md。
- permission.Gate：新增 AlwaysAsk 硬门，cost_save 无视权限级别/放行规则/会话记忆
- control：gateApprover 对 cost_save 强制 requestApproval（alwaysPrompt），审批摘要含条目名称/单价/单位/规格/来源
- SetPermLevel：auto/yolo 均保留 cost_save 硬门（yolo 改用空策略 gate 替代 nil gate）
- ApprovalModal：cost_save 隐藏「本会话允许」，显示逐条确认提示
- 回归测试：yolo 下仍触发审批、会话放行不被记忆、审批摘要格式
- 验证：go build/control/permission 测试通过；前端 tsc + 128 例全过；wails build 通过

## v2.13.18「通用办公左侧面板重设计（参考 Codex）」（2026-08-12）
> 左侧面板按 Codex 会话栏风格重排：紧凑头部（logo + 名称 + 新建/折叠图标按钮）、
> 搜索框前置放大镜、会话行改为「标题 + 时间 + 预览」结构（悬停操作、左色条选中态）、
> 区块头统一安静小字，功能与折叠/拖拽行为全部保留。
> 详见 releases/v2.13.18.md。
- Sidebar.tsx：头部精简、会话行重构、搜索框带图标、分组头/区块头统一，删除大号胶囊新建按钮
- 保留：会话搜索/重命名/删除/历史、后台任务、事实底座（导出/沉淀/复制/清空）、记忆/知识库/技能导航、折叠与拖拽调整宽度
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.18.exe

## v2.13.17「修复运行中已完成的小过程卡不折叠」（2026-08-12）
> 修复 v2.13.9 引入的回归：当时为让「大过程卡（整轮合并、包含文本卡的卡）」
> 保持展开，把 ProcessCard 初始状态改成了默认展开，导致运行中不含文本的
> 分段「小过程卡」在段完成后也不再收起。现在小过程卡默认折叠、段完成后
> 收起；含文本的大过程卡保持展开；正在流式的最后一段保持展开。
> 详见 releases/v2.13.17.md。
- Transcript.ProcessCard：新增 small 标记（运行中的分段小卡）；初始折叠、完成/历史段自动收起
- 大过程卡（consolidated，含中间文本）默认展开逻辑不变
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.17.exe

## v2.13.16「办公板块铺满窗口 + 顶部标签删除 + 底栏只显示本地模型」（2026-08-12）
> 通用办公三个显示细节优化：① 删除办公板块顶部「办公」二级标签，模块直接由
> 通用办公工作台承载；② 底部状态栏的已启动模型列表/超载报警只统计本地模型，
> 云端引擎（xai/deepseek/opencode）不再展示与计入；③ 全屏/最大化时通用办公
> 铺满整个窗口（此前内容区对办公页保留了 8/16px 内边距）。
> 详见 releases/v2.13.16.md。
- MainLayout：办公页路由直接挂 GaeaPage，删除 OfficeHubPage 与 office-hub/office-page 样式；内容区对办公页去内边距、铺满
- StatusBar：已启用模型列表按 isLocal 过滤，只显示本地模型；超载报警维持本地统计
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.16.exe

## v2.13.15「删除方案编写，办公板块收敛为单一入口」（2026-08-12）
> 办公板块下线「方案编写」二级分支：删除整个方案编写模块（前端页面、
> proposal/db 后端包、Proposal* 绑定、模块注册、方案库），去除办公二级导航，
> 办公板块直接承载通用办公工作台。左脑记忆源从方案库改为办公记忆 facts，
> 三脑检索与记忆中枢不受影响。
> 详见 releases/v2.13.15.md。
- 前端：删除 OfficePage，OfficeHubPage 去除二级导航（单一通用办公入口）；模块启动器与设置面板清理「方案编写」文案
- 后端：删除 internal/office/proposal 与 internal/office/db，移除全部 Proposal* 绑定与 office 模块注册
- 左脑：办公记忆源改为 Hephaestus facts（三脑检索/记忆图谱继续可用）
- 验证：go build/定向测试通过；前端 tsc + 128 例全过；wails build 通过；产物 gaea-v2.13.15.exe

## v2.13.14「通用办公工具/技能面板显示 Word·Excel·PDF 技能」（2026-08-12）
> 通用办公的「技能面板」看不到 docx/xlsx/pdf/pptx：技能只装在仓库 .gaea/skills，
> 而技能索引只扫「当前工作区 .gaea/skills + 用户级 ~/.gaea/skills」，其它工作区
> 均为空。已把四个 ModelScope 文档技能安装到用户级全局目录（任意工作区可见），
> 并在「工具面板」新增「文档技能」分组展示这四个技能。
> 详见 releases/v2.13.14.md。
- 安装：docx / xlsx / pdf / pptx 复制到 ~/.gaea/skills（全局技能，所有工作区可见）
- CapabilitiesPanel：工具面板新增「文档技能」分组（docx/xlsx/pdf/pptx 卡片）
- 验证：全局技能在任意工作区可见（金具厂实测通过）；tsc + 前端 128 例全过；wails build 通过

## v2.13.13「聊天面板左侧栏可折叠 + 悬浮绑定模型卡随面板隐藏」（2026-08-12）
> 聊天页面左侧会话栏原本固定 264px 宽、无法折叠；左下角「聊天」绑定模型卡
> 悬浮在其上方也不随面板变化。新增折叠窄栏（52px，保留展开/新建按钮），
> 折叠态持久化；折叠时悬浮绑定模型卡一并隐藏。
> 详见 releases/v2.13.13.md。
- ChatTopicSidebar：新增 collapsed/onToggle，折叠为窄栏，展开态头部增加折叠按钮
- ChatPage：折叠状态存 localStorage；折叠时隐藏左下角绑定模型卡
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.13.exe

## v2.13.12「通用办公布局优化：精简右侧边栏 + 绑定模型卡随面板折叠」（2026-08-12）
> 通用办公右侧工作区面板删除「成本库」「搜索」两个标签页；左下角「绑定模型卡」
> 原为绝对定位悬浮，会盖住左侧栏底部的「知识库」「MCP 与技能」按钮，且不随
> 面板折叠。改为放入左侧栏底部（与后台任务/事实底座同规则），折叠时随面板隐藏。
> 详见 releases/v2.13.12.md。
- App.tsx：右侧边栏删除成本库/搜索标签与对应面板，同步移除命令面板的搜索入口
- Sidebar.tsx：绑定模型卡移入左侧栏底部，折叠时隐藏，不再遮挡底部导航
- GaeaPage.tsx：移除原来的绝对定位悬浮层
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.12.exe

## v2.13.11「修复记忆中枢·用户画像打不开」（2026-08-12）
> 记忆中枢「用户画像」页打开即白屏：后端无冲突时返回 nil 切片，序列化成
> JSON null，前端直接读 `conflicts.length` 抛 TypeError，被 ErrorBoundary
> 拦截。后端统一保证空数组，前端对 conflicts/tags 做 null 兜底，并加固
> 图谱与聊天记忆页同型风险。
> 详见 releases/v2.13.11.md。
- GaeaProfileConflicts：无冲突时返回 [] 而非 null（DetectConflicts 改 []string{}）
- ProfileLibrary：conflicts/tags 加 ?? [] 兜底（崩溃根因）
- 顺带加固：GaeaWhisperEpisodes 错误路径、GaeaMemoryGraph 空图、GraphView/WhisperMemoryLibrary null 防护
- 回归测试：Go DetectConflicts 非 nil 空切片 + 前端 ProfileLibrary null 用例
- 验证：go build/测试通过；前端 128 例全过；wails build 通过；产物 gaea-v2.13.11.exe

## v2.13.10「修复办公文档处理反复弹 cmd 黑窗」（2026-08-12）
> 通用办公后台做 docx/xlsx 转换、公式重算、报告导出、绘图、桌面自动化时，
> 子进程没加隐藏窗口参数，每执行一次就闪一个 cmd/conhost 黑窗；批量转换时
> 会"不停弹窗"。统一补上 CREATE_NO_WINDOW，全部静默执行。
> 详见 releases/v2.13.10.md。
- docmd.markitdown：`python -m markitdown` 补 HideWindow（弹窗主因，已实测抓到）
- xlsxedit.Recalc、gaea_export.runPython、diagram_gen、whisper 桌面操作（powershell）一并补上
- 验证：go build 通过；前端 127 例全过；wails build 通过；产物 gaea-v2.13.10.exe

## v2.13.9「每一轮的大过程卡默认展开 · 内部卡默认折叠」（2026-08-12）
> 调整通用办公输出显示：输出完成后每一轮的外层大过程卡都默认保持展开，
> 不再只保留最新回合；大过程卡内部的思考卡改为默认折叠，工具卡/工具组
> 保持默认折叠，只留过程文本直接可见。
> 详见 releases/v2.13.9.md。
- ProcessCard：移除 isLatest 限制，每轮大过程卡完成后均保持展开（用户手动折叠过仍尊重）
- InlineReasoning：思考卡默认折叠，点开才看推理内容；工具卡/工具组保持默认折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.9.exe

## v2.13.8「仅最新回合的大过程卡默认展开」（2026-08-12）
> 修正过程卡展开策略：只有最新回合的外层大过程卡默认展开，旧回合自动折叠，
> 避免“所有过程卡全部摊开”；手动展开过的旧卡仍保留。
> 详见 releases/v2.13.8.md。
- ProcessCard 增加 isLatest：完成时仅最新回合保持展开，新回合开始后旧卡折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.8.exe

## v2.13.7「办公上下文窗口修正 + 大上下文处理提示」（2026-08-12）
> 修复“长任务只有执行中、迟迟不出流式”的根因：办公 provider 上下文窗口此前
> 写死 1M，自动压缩阈值高达 80 万 token，会话膨胀到十几万也从不压缩，模型
> 首字要等几分钟，看起来像没输出。窗口调至 256k 让压缩生效，并加处理提示。
> （v2.13.6 的最终回答兜底补渲染一并包含在本版）
> 详见 releases/v2.13.7.md。
- gaea 办公 provider ContextWindow 1_000_000 → 256_000（80% 阈值≈204k，超限自动压缩）
- 前端 RunStatus：运行 ≥20s 且上下文 ≥4 万 token 时显示「处理大上下文中 · N」，不再误判卡死
- 最终回答兜底：turn_done / 看门狗恢复时校验 History 补渲染缺失的最终回答
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.7.exe

## v2.13.5「运行中强制跟随底部，修复“卡住没输出”假象」（2026-08-12）
> 长任务其实一直在输出，但智能滚动一旦被布局变化/折叠动画置为非跟随状态，
> 视图就冻在旧位置，看起来像卡死。运行中改为强制跟随底部，结束恢复智能滚动。
> 详见 releases/v2.13.5.md。
- Transcript：running 时内容变化强制滚到底部；运行结束后保持原智能滚动
  （用户上翻浏览时不再强拉）
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.5.exe

## v2.13.4「外层过程卡完成后默认展开」（2026-08-12）
> 修正 v2.13.3 的方向：要展开的是包裹「过程文本 + 过程卡」的最外层大过程卡
> （ProcessCard），输出完成后默认保持展开；内部工具卡仍默认折叠。
> 详见 releases/v2.13.4.md。
- ProcessCard：running → 完成后不再 setOpen(false)，外层大过程卡默认保持展开
  （用户手动折叠过仍尊重）；v2.13.3 误加的内部卡自动展开已撤回
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.4.exe

## v2.13.3「过程卡完成后默认展开」（2026-08-12）
> 输出完成后的大过程卡（bash/python 等长输出卡片）默认展开，用户手动折叠后
> 不再干预；连续同名工具组含大卡时组也默认展开。
> 详见 releases/v2.13.3.md。
- ToolCard：输出 ≥10 行或输出/参数 ≥2000 字符的大卡，完成（done/error）后自动展开
- ToolGroup：组内任一成员为大卡且完成时，组默认展开
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.3.exe

## v2.13.2「办公过程文件落盘规范：统一 .gaea/work/」（2026-08-12）
> 办公 agent 执行纪律新增「文件落盘规范」：过程/中间文件（提取文本、OCR 页图、
> 脚本、临时图表等）统一写入 .gaea/work/<任务名>/，交付物进 .gaea/exports/，
> 不再与源文件混在工作空间根目录；启动时自动创建这两个目录。
> 详见 releases/v2.13.2.md。
- 办公 agent 提示词新增「文件落盘规范」章节（single_prompt.go）
- GaeaInit 自动创建 .gaea/work/ 与 .gaea/exports/；.gitignore 增加 .gaea/work/
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.2.exe

## v2.13.1「修复 @PDF 引用注入二进制导致办公输出不可见」（2026-08-12）
> 补丁版：修复通用办公里 @引用 PDF 时，文本提取失败会把 %PDF-1.4 原始二进制
> 塞进模型上下文，导致任务只剩读秒、看不到任何输出。
> 详见 releases/v2.13.1.md。
- @引用 office 文档（PDF/docx）转换失败时注入占位提示（format_convert / summarize_file），
  绝不回退成原始字节；二进制检测扩展到整个读取窗口（此前只查前 8KB，PDF 头无 NUL 会漏检）
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.1.exe

## v2.13.0「通用办公打磨 + 模型中心优化 + 本地模型实测」（2026-08-12）
> 承接 v2.12.0，本轮聚焦通用办公稳定性与方案写作质量：修复方案分节 100 字
> 截断（流式与批量两处）、docx 读取乱码、前端“运行中”假卡死；配套模型中心
> 优化与 Herdsman/Ollama 本地模型双模式实测。
> 详见 releases/v2.13.0.md。
- 通用办公：方案「生成该节」与批量生成按章节字数目标续写（WordTarget，缺省 800），
  修复原硬编码 100 字截断；docx 读取统一走 ConvertToMarkdown（MarkItDown / 内置
  转换 / 扫描件提示），修复直接读 zip 字节乱码；前端 turn 结束事件丢失兜底
  （停止按钮无条件复位 + 30s GaeaRunning 看门狗），任务完成不再卡「执行中」；
  待办从未推进时 turn_done 自动收尾，不再残留 0/N
- 工具：format_convert 输出父目录自动创建（与 write_file 对齐）+ 回归测试；
  docmd 新增 MarkItDown 通道（docx/pptx），PDF 页数上限与扫描件 OCR 回退保持
- 模型中心：功能绑定/板块细节优化；本地引擎思考参数（enable_thinking /
  chat_template_kwargs）与路由增强
- 语音/角色库：whisper web_search、角色画像文件化等累积修复
- 实测：Herdsman 三模型 20 任务 × 思考/非思考双模式 + Ollama GLM 实测，
  脚本 scripts/test_herdsman_models.go 可复现；详见
  docs/2026-08-12-herdsman-models-evaluation-report.md
- 验证：go build/vet 干净，方案包等测试全绿；前端 127 例全过，tsc + vite build 通过

## v2.12.0「稳定工程 + 成本库多级分类重设计」（2026-08-11）
> 系统根治 WebView2 rAF 冻结引发的「界面卡死 / 点不了 / 角色卡看不见」，
> 成本库升级为按分类分级保存 + 列表/表格双视图，剧照文件化防 IPC 撑爆。
> 详见 releases/v2.12.0.md。
- 稳定性：全应用 30+ Modal 关闭即卸载 + 禁用过渡动画（弹窗关闭后不再残留遮罩卡死）；
  rAF 持续探测 + 8s 冻结看门狗（运行中晚发节流也能降级）；角色卡/启动器/聊天气泡
  入场动画纳入降级名单（修复角色库卡片全看不见）；前端错误/长任务/心跳诊断进 gaea.log
- 成本库：cost_categories 多级分类树 + 条目 category_path 多级保存；默认按信息价体系
  分类（材料细分三级）；分类增删改、改名自动重写子树路径、按路径含子树过滤；
  面板重设计为分类树 + 列表/表格双视图（排序/批量/价格历史）；cost_search/cost_save
  支持分类路径
- 剧照文件化：base64 剧照落盘存路径，>300KB 内联头像截断，根治巨型 base64 撑爆 IPC
- 记忆中枢/办公：成本导入（预览确认 + AI 归一化）、价格源仓库、价格历史与跳幅识别、
  知识导入/版本历史/查重合并、画像库、办公记忆查重合并、记忆图谱增强；后端导入链路
  加固与批量测试覆盖
- 验证：go build/vet 干净，go test 相关包全绿；前端 127 例全过，tsc + vite build 通过

## v2.11.0「通用办公大优化：四大库能力闭环 + 本地语义检索栈」（2026-08-10）
> 承接 v2.10.1 开工前 P0，落地调研蒸馏出的 P1 三项：工作区全文搜索、
> 常用资料固定装配、任务模板库——把「开工前」从「能搜到文件」升级为
> 「能搜到内容、能钉住常用资料、能一键发起常见任务」。
- 工作区全文搜索（轻量 RAG）：右侧新增「搜索」标签页 + 命令面板入口，在
  docx/xlsx/pdf/md/txt/csv 等正文里定位关键词（中文 bigram 分词 + TF-IDF 打分），
  返回命中片段，可预览或一键 @ 引用；正文按 (路径, mtime, size) 缓存，重复搜索
  不重复解析；.gaea/sessions|archive|cache 等噪音跳过，.gaea/exports 交付产物可索引
- 常用资料固定装配（P1-②）：资料面板每行图钉「固定为常用资料」，清单持久化到
  <工作区>/.gaea/pinned.json（去重、限长、防路径逃逸）；新会话启动时自动把固定
  清单装进系统提示词——文本类附正文摘要、办公文档列名按需读取（装配而非灌输）
- 任务模板库（P1-③）：欢迎页新增「任务模板」区（周报/会议纪要/成本测算/方案大纲/
  数据分析/文档转换/报告拼装/演示文稿），一键填入结构化指令；同源模板同时落盘为
  .gaea/commands/*.md（幂等、不覆盖用户文件），/ 菜单与 Submit 直接可用
- 修复：办公引擎命令发现改为基于工作区根（CommandDirsAt），切换工作空间后
  模板与自定义命令跟随新工作区，与技能/记忆的发现口径一致
- 记忆中枢关联（本轮打通）：记忆中枢首页「三脑检索」并入工作区全文搜索——
  文件命中以「文件」徽标同框展示，可预览 / 一键 @ 引用（回到办公板块自动插入
  输入框）；首页新增「项目资料」卡片（固定常用文件计数）与对应库面板（已固定/
  可固定资料管理、预览、固定/取消，与办公面板共用 .gaea/pinned.json）；固定
  资料作为 material 节点进入记忆 3D 图谱（天蓝色，图谱过滤器可单独开关）
- 记忆生命周期与可控（P1-④/⑤）：facts 新增 last_used_at / source_session /
  source_message（SchemaV3 迁移，修订不重置使用时间）；memory_get 读取即 Touch，
  逐轮记忆注入改为「关键词 + 时间 + 高频」排序 + 预算压缩（RecallBlock 默认
  800 rune，画像 600 rune），procedural 常驻、episodic 按标签触发、相关事实按
  命中带入；记忆面板顶部新增「记忆开关」（持久化配置 + 引擎重建立即生效），
  事实卡片展示来源会话与最近使用时间
- 方法论自动候选（P1-⑥）：记忆面板「建议」标签接入真实后端——从 procedural
  记忆按主题词聚类（≥2 条同主题 → 提议 workflow-<主题> 技能，附证据与正文），
  接受后经既有技能沉淀通道固化并热加载；记忆候选保持为空（自动做梦已直接入库，
  宁缺毋滥）
- 验证：后端新增 wssearch/pins 包测试 + boot 集成测试（工作区命令发现），
  以及记忆中枢关联测试（固定数统计 + 图谱 material 节点）、记忆生命周期/压缩
  注入/技能聚类测试；前端新增 WorkspaceSearchPanel 2 例 + 资料固定 1 例 +
  项目资料库 1 例，vitest 93 例全过，tsc 与 vite build 通过

### 本轮追加：成本库/知识库/办公记忆/工作区文件大优化
- **成本库打通与功能深化**：cost_search/cost_save 内置工具（读/写 Hephaestus.db
  cost_entries，同名 UPSERT）；cost-estimate 模板「先查后沉淀」；办公右侧成本库 Tab
  （浏览/搜索/一键结构化引用）；产物面板「沉淀到成本库」；导入 xlsx/csv + AI 归一化
  （预览确认，无确认不落库）；编辑/删除/批量操作；价格源订阅 + 手动/定时抓取
  （四川造价信息网实测 24 行解析）+ 价格历史 + 单期跳幅 ±20% 异常识别
- **知识库大优化**：md/txt/docx/pdf/xlsx/csv 导入 + AI 多主题拆分（预览确认）；
  批量管理 + 审核流（草稿→现行、审核人留档）+ 版本历史（内容变化自动留档）+ 批量
  导出 Markdown；查重（CJK 二元组 Dice）与相似条目一键合并
- **办公记忆**：两两查重 + 一键合并（textsim 共享相似度），自动做梦产生的同名变体
  可一键归并
- **工作区文件语义索引**：fileindex 扫描 md/txt/csv/docx/xlsx/pdf/pptx 提取正文
  （pptx 零依赖解 zip 读 slide XML），bge-m3 向量化入库（semantic_vectors kind=file，
  内容感知增量重建 + 10 分钟自动维护），搜索面板「语义」模式 + 记忆中枢统一搜索
  并入「语义·文件」命中
- **本地语义检索栈（四库统一）**：子串召回 →（不足时）bge-m3 语义召回 → BM25 排序 →
  （>8 条）bge-reranker-v2-m3 精排，任一层失败自动降级；跨库统一语义检索
  （GaeaSemanticSearch，成本/知识/办公记忆）；Herdsman bge-m3 / bge-reranker-v2-m3
  实测端到端通过，全部本地零 token
- 验证：go test 全量 + go vet 干净；前端 111 例全过，tsc 与 vite build 通过；
  真实 Herdsman 语义召回/精排与四川价格源抓取实测通过；产物 gaea-v2.11.0.exe

## v2.10.1「开工前 · 交付收尾 · 记忆自进化」（2026-08-10）
> 把办公体验从「下完指令等结果」补成完整闭环：开工前先出计划、资料一键引用；
> 对话里所有文件引用可点击预览，交付物收尾成卡片；会话结束后自动「做梦」整理记忆。
- 输出文件全量可点击预览：mdast（remark 插件）识别绝对/相对/裸文件名（参考
  openclaw/llama.cpp 的 AST 插件架构），聊天正文、流式尾部、工具输出、ask 提问均可点击，
  代码块/已有链接/公式天然跳过；后端支持裸文件名自动解析到 exports 等常见输出目录
- 交付收尾：助手消息尾部「交付文件」卡片（缩略图/图标 + 文件名 + 扩展名，预览/复制路径/
  外部打开/定位）；右侧「会话产物」面板（本次会话交付文件清单 + 溯源跳转到生成消息）；
  预览内编辑后自动标「已更新」并刷新文件树；粘贴表格即数据（CSV/TSV 识别 → Markdown 表格）
- 记忆自进化（自动做梦）：轮次成功后后台归纳对话 → 提炼稳定事实/偏好/笔记写入长期记忆
  （按 name 去重、Kimi 二问纪律、单飞、失败静默）；整理结果在聊天内通知
- 开工前：@ 文件引用增强（工作区跨目录搜索 + 最近使用 + 扩展名徽标）；右侧「资料」面板
  （docx/xlsx/pptx/pdf/md/csv 按类型分组、最新在前、一键 @ 引用进输入框）；开工前计划卡片
  （非简单任务先生成计划，用户「确认执行 / 先调整」，auto_plan 可配置，默认开启）
- 三轮竞品优点蒸馏文档：交付收尾 UX、记忆与自进化、开工前上下文；调用链核对文档
  （docs/gaea2/office-model-tool-call-chains.md）
- 发布说明：产物 gaea-v2.10.1.exe（Windows x64），桌面端同步；go test（agent/control/boot/
  app 全量）+ vitest 89 例全过，tsc 与 wails build 通过

## v2.10.0「正式发布 · 通用办公三阶段闭环」（2026-08-09）
> 自 v2.6.9 之后最大一次发布：把「办公前期的文件解析、中期的 Word/Excel
> 编辑、后期的文件输出与预览」串成一条完整闭环，并完成桌面端体验重构与
> 真实模型端到端验收。本版本包含 v2.7.0–v2.9.4 的全部累积特性。
- 前期解析：docx/xlsx/pdf → Markdown 提速（docx 约 12x / PDF 约 4x）；
  扫描件 OCR 四级管线（OvisOCR2 常驻服务 → RapidOCR → WinRT → 本地视觉模型），
  粘贴图片「提取文字/识图」双入口；事实底座（fact_add/list/clear + 侧栏面板 +
  一键沉淀长期记忆，后续对话自动加载）
- 中期编辑：Word 框选即改 + 修订模式逐条接受/拒绝 + diff 回看；Excel 单元格级
  预览/编辑（双击改值/写公式、fx 栏、AI 指令编辑、插入/删除行与列、LibreOffice
  公式重算、数字格式/冻结表头/合并单元格）；跨应用联动（Excel 数据 → 图表 →
  嵌入 Word/PPT 并随数据同步刷新）
- 后期输出：统一交付出口（对话成果一键导出 docx/pptx/xlsx/md）、模板与样式体系、
  成本测算模板（市政道路改造工程，公式全联动 + 原生图表，真实 DeepSeek 绑定
  自主生成端到端验收通过）
- 桌面端体验：Codex 式文件预览布局（右侧文件树、主区域可拖宽预览、Esc/按钮
  快捷收起）、文件就地编辑不跳出对话
- 发布说明：产物 gaea-v2.10.0.exe（Windows x64，约 40MB），桌面端同步；
  go test / vitest 59 例全过，tsc 类型检查与 wails build 通过

## v2.9.4「表格列操作：插入/删除列」（2026-08-09）
> 按用户要求：排序/筛选/条件格式不在 gaea 内做预览层实现（不写回文件），
> 后期交给 Excel/WPS 专业软件处理；本次仅落地会真实写文件的列操作。
- 插入/删除列：点击列字母选中整列，工具栏出现「← 插列 / → 插列 / 删除列」
  （删除二次确认），后端 excelize InsertCols/RemoveCol + 重算刷新，改动落盘
- 新增后端 GaeaXlsxColOps；列选择与单元格选择互斥
- 清理：移除本轮试验性的排序宏（LibreOffice）、AutoFilter 写入与条件格式渲染
- 验证：Go 单测（插列/删列错位、非法操作/非法引用）+ 前端列操作用例；
  go test / vitest 59 例全过；桌面 gaea.exe 已重建同步

## v2.9.3「表格行操作：插入/删除行」（2026-08-09）
> 按用户要求补齐高频表格操作——行级插入与删除。
- 预览头部新增「↑ 插行 / ↓ 插行 / 删除行」：基于选中单元格所在行，
  插入空行或删除整行；删除需二次确认（点击后变为「确认删除」）
- 后端新增 GaeaXlsxRowOps（excelize InsertRows/RemoveRow 平移同表公式与合并区，
  随后 LibreOffice 重算刷新结果并重渲染预览）
- 验证：Go 单测（插入错位/删除上移/非法操作/非法引用）+ 前端插行用例；
  go test / vitest 58 例全过；桌面 gaea.exe 已重建同步

## v2.9.2「公式结果显示 + 表格功能补强」（2026-08-09）
> 修复公式格只显示 fx 无结果的问题，并补上两类高频表格能力。
- 公式结果：根因是 openpyxl 生成的文件公式没有缓存值（未重算），预览渲染时
  只剩 fx 标记。现在 GaeaPreview 自动检测「无缓存值公式」→ LibreOffice 重算后
  再渲染；预览头部新增「重算公式」按钮可手动刷新；新增 GaeaXlsxRecalc 接口
- 数字格式：NumFmt 已提取但前端未应用，现按格式显示千分位/百分比/货币/小数位
  （内置编号 1-11、44-47 及常见自定义格式），公式结果也按单元格格式呈现
- 冻结表头：提取 sheet 冻结窗格（GetPanes），预览中冻结行 sticky 固定，
  滚动数据时表头不跑
- 验证：新增 NeedsRecalc 单测 + 冒烟回归（真实成本表 51 个公式全部有结果，
  预览自动重算 1.5s）+ 前端数字格式用例；go test / vitest 57 例全过；
  桌面 gaea.exe 已重建同步

## v2.9.1「预览拖拽修复 + 表格就地编辑」（2026-08-09）
> 修复 Codex 式预览的两个实测问题：拖拽分割条失效、表格只能看不能改。
- 修复拖拽：预览宽度上下限计算写反导致宽度卡死（恒为最大），改回
  320–1100px 自由拖拽；分割条命中区 6px→10px，悬停/拖拽高亮
- 表格 Excel 式就地编辑：双击单元格直接输入（Enter 保存 / Esc 取消），
  上方 fx 公式栏也可输入值或 `=公式` 回车写回；纯数字按数值写入保持可计算，
  等号开头写公式，写回后 LibreOffice 重算并即时刷新预览
- 新增后端 GaeaXlsxSetCell 直写接口（excelize 写值/公式 + 重算 + 预览）
- 验证：新增 Go 单测（写值/写公式/非法引用）+ 前端双击编辑用例；
  go test / tsc / vitest 56 例全过；桌面 gaea.exe 已重建同步

## v2.9.0「Codex 式文件预览布局」（2026-08-09）
> 参照 Codex 桌面端交互改造办公文件查看：右侧面板收敛为「增强文件树」，
> 点击文件后原右侧面板隐藏，预览在聊天区右侧主区域展开（全高、宽度可拖），
> 聊天与预览互不遮挡，可边对话边看文件。
- 布局：点文件 → 树收起 → 主区域出现预览面板，聊天区保留在左侧；
  预览与聊天之间加 6px 拖拽分割条，宽度 320–1100px 自由调整并持久化（localStorage）
- 预览头部：新增「文件」返回按钮（回到文件树）、文件名/大小、定位/外部打开/关闭；
  Esc 或工具栏面板按钮均可收起预览回到树
- 文件树增强：docx 蓝 / xlsx 绿 / pptx 橙 / pdf 红 / 图片紫 按扩展名着色，
  文件夹优先 + 自然排序；新增手动刷新按钮（强制重载目录）
- 清理：移除已被树内嵌预览取代的「最大化面板」逻辑与重复的“查看文件变更”按钮
- 验证：tsc --noEmit 通过；vitest 55 例全过；wails build 成功，桌面 gaea.exe 已同步

## v2.8.9「自主生成验收 · gaea 自己生成成本测算表」（2026-08-09）
> 让 gaea 智能体用真实模型（办公绑定 deepseek/deepseek-v4-flash）自主生成
> 成本测算表：无头驱动 Controller.Run + bridge provider + 完整工具链，验证
> 产物含公式与原生图表——桌面端已构建，办公功能绑定已生效。
- 新增 TestGaeaSelfGenerateCost（GAEA_SELFGEN 门控）：读取 ~/.gaea_config.json
  办公功能绑定路由引擎/模型；照桌面端接线 engineMgr + ai.Client + bridge；
  提示词要求 openpyxl 建表、公式联动、原生饼图/柱状图、LibreOffice 重算
- 实测结果（900s）：智能体自主完成环境检查→写脚本→生成→重算→自检；
  产物 3 工作表（成本测算/费用汇总/编制说明）、35 个公式、2 个原生图表，
  openpyxl 断言全过；路由日志「来源 feature」确认走了用户绑定
- 顺手修复既有测试全局污染：TestGaeaBootBuild 注入 gaeaConfig.SetLoader
  未恢复，会影响同包后续 boot.Build（真实模型跑测试时被 agent 诊断发现）；
  已 defer 恢复；自生成测试产物自动复制到 .gaea/exports/
- 验证：go vet/test ./internal/app 全绿；桌面端 gaea.exe 已重新构建并
  同步桌面（含全部办公能力与绑定）

## v2.8.8「成本测算模板 · 真实样例交付」（2026-08-09）
> 新增可复用成本测算生成脚本 cost_estimate.py（openpyxl）：市政道路改造工程
> 成本测算工作簿——按费用构成（人工/材料/机械/企管/规费/利润/税金）建表，
> 公式全联动（数量×单价=合价、小计、占比、含税总价、综合单价），原生饼图+
> 柱状图，LibreOffice 重算后带缓存值。
- 交付物：.gaea/exports/市政道路改造工程成本测算.xlsx（含税总价
  9,992,806.15 元，综合单价 462.63 元/m²，53 个公式 0 错误，2 个原生图表）
- 验证：openpyxl 数值/公式/图表断言、PDF 渲染文本核验（表/图表标题/编制说明
  完整）、gaea GaeaPreview 单元格预览管线读取正常（GAEA_SMOKE_COST 门控）

## v2.8.7「P2 跨应用联动 · Excel 数据 → 图表 → 嵌入 Word/PPT」（2026-08-09）
> 联动闭环：xlsx 数据一键生成 matplotlib 图表并嵌入 docx（报告模板：标题+
> 图表+数据表）或 pptx（图表页+数据明细页）；数据源更新后重新导出即刷新图表，
> 实现「图表与数据保持同步」（避免脆弱的 OLE 活链接）。
- 新增 internal/office/crosslink：xlsx 数据提取（自动/显式区域 A1:B6，表头+
  数值列解析、千分位清洗）、matplotlib 图表生成（bar/line/pie/scatter，中文
  字体、无窗口执行）、docx/pptx 嵌入 spec 构建（docx 含数据表，pptx 图表页+
  数据明细页）
- create_pptx.py 新增每页 image 支持（图表页居中嵌入）；脚本镜像同步
- 新增 GaeaCrossEmbed 绑定：格式/图表类型校验、产物与图表落到 .gaea/exports/
- 前端 XlsxPreview 新增「图表→Word」「图表→PPT」按钮：取当前工作表数据一键
  联动，成功后自动定位产物
- 验证：crosslink 单测（数据提取/区域裁剪/spec 构建）+ 真实 matplotlib 图表
  冒烟（23KB PNG，0.5s）+ docx/pptx 嵌入冒烟（产物合法 zip 且含 media 图片）；
  go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 55 例全过

## v2.8.6「打磨联调 · 边界加固 + 端到端走查」（2026-08-09）
> 把前几轮的能力串成一条全流程验证：解析 → 修订式编辑 → 接受修订 →
> 提取成果 → 模板化多形态交付，并补齐三处常见边界。
- 边界加固：
  - docxedit：表格单元格（w:tbl>w:tr>w:tc>w:p）内修订与接受全链路（合同场景）；
  - xlsxedit：transform 公式保留 $ 绝对引用、跳过函数名（LOG10/ROUND）误判，
    行引用调整专项用例；
  - 交付出口：标题含 Windows 非法字符（< > : " / ? 等）时清洗且保留中文
- 端到端：
  - 新增 TestOfficeFullPipeline（纯 Go）：编辑→接受→提取→导出 xlsx/md；
  - 新增真实文档全链路走查（GAEA_SMOKE_PIPELINE 门控）：26MB 方案文档
    编辑→接受→提取→报告模板导出 docx，全程 4s 完成、产物内容正确
- 验证：go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 54 例全过

## v2.8.5「后期输出 · 模板与样式体系」（2026-08-09）
> P1 模板库落地：统一交付出口支持 通用/公文/报告/合同 四套版式预设——
> 字体、标题色/标题字体、表头底色按模板切换，事实底座一键出对应样式的 Word。
- create_docx.py 新增 --template 预设：通用（宋体+深蓝标题）、公文（仿宋正文+
  黑体标题）、报告（微软雅黑+蓝标题）、合同（宋体+黑色标题）；封面标题色、
  标题 run、表头底色全部参数化贯穿
- GaeaExportDeliverable 新增 Template 字段（默认通用，非法值明确报错），
  docx 出口透传 --template
- 事实底座侧栏「导出报告」升级为「模板选择 + 导出」：报告（封面+目录）/
  公文 / 合同 / 通用，导出后自动定位文件
- 验证：四套模板实测生成（颜色/字体抽查：通用 1F3864/宋体、报告 2E74B5/
  微软雅黑、公文 000000/仿宋）；docx 导出冒烟带模板通过；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过；技能镜像 ~/.codex/skills 已同步

## v2.8.4「中期编辑 · 修订接受/拒绝 + diff 回看」（2026-08-09）
> P1 信任机制落地：框选即改的修订不再只能去 Word 里处理——预览里直接
> 「接受修订 / 拒绝修订」一键扁平化（按作者 gaea AI，不动他人修订），
> 修订样式本身即 diff 回看（原文划除 + 新文插入）。
- docxedit 新增 AcceptChanges / RejectChanges：字节级扁平化 w:del/w:ins
  （接受=删 w:del 留 w:ins；拒绝=删 w:ins 还原 delText→t 并保留 run 格式），
  只处理指定作者、跳过嵌套/重叠、无修订时明确报错
- 新增 GaeaDocxAcceptChanges 绑定（accept=true/false），返回更新预览
- 前端 DocxPreview：渲染后自动检测 ins/del 修订，标题栏出现「接受修订 /
  拒绝修订」按钮（加载中禁用），操作后重渲染并提示
- 验证：docxedit 接受/拒绝/无修订/他人修订不动四类单测 + 真实 26MB 文档
  「修订→接受」冒烟（1.7s 完成、XML 合法、标记清空）；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过

## v2.8.3「后期输出 · 事实底座 → 多形态交付统一出口」（2026-08-09）
> P0-④ 落地：统一交付管线「受控 Markdown → docx / pptx / xlsx / md」——
> 事实底座与对话成果一稿多用，多形态基于同一底座生成、彼此一致。
- 新增 GaeaExportDeliverable 绑定：格式校验、标题自动推导、时间戳防重名、
  非法字符清洗（保留中文）、交付到 .gaea/exports/ 并返回相对路径
- docx：复用 create_docx.py（python-docx，封面/目录/页眉页脚/页码参数化）；
  pptx：Markdown → slides spec（# 标题开新页、要点/表格行转要点）→ create_pptx.py；
  xlsx：Go 侧 excelize 直接提取 Markdown 表格为工作表（无表格时正文入 Sheet1）；
  md：直写；python 子进程带超时与 stderr 捕获
- 前端两个入口：会话工具栏新增「导出 Word」（对话成果一键交付，成功后自动定位）；
  事实底座侧栏新增「导出报告」（封面+目录，基于同一底座）
- 验证：md/xlsx/校验单测（纯 Go）+ docx/pptx 真实脚本冒烟（GAEA_SMOKE_EXPORT，
  实测通过、产物为合法 zip）；go test ./... 全绿；tsc + vite build 通过；
  vitest 54 例全过

## v2.8.2「中期编辑 · Excel 单元格级操作」（2026-08-09）
> P0-③ 闭环：在单元格预览上「选中单元格 → 指令（求和/平均/拆分列/清洗/替换/
> 自定义）→ AI 规划操作 → excelize 执行 → LibreOffice 重算公式 → 重渲染」，
> 公式可校验、结果可检查。
- 新增 internal/office/xlsxedit：AI 操作集（set_formula/set_value/fill_range/
  transform/replace/split_column/clean）校验与执行；transform 逐行公式自动调整
  相对行引用（跳过 $ 绝对引用与函数名）；操作数量/单元格数上限防滥用
- 新增 ai.XlsxEditOps：表格上下文（工作表/表头/抽样数据）+ 指令 → 严格 JSON 操作集；
  GaeaXlsxEdit 绑定走 office 功能绑定路由，闭环后返回更新预览与逐条摘要
- 公式重算：复用技能环境 recalc.py（LibreOffice 宏，自动探测 soffice），
  best-effort——重算不可用时编辑仍生效并提示；真实重算冒烟 1.75s 通过
- 前端 XlsxPreview 新增选中编辑工具栏：四预设（求和/平均值/拆分列/清洗）+ 自定义
  指令，执行后重渲染并展示应用摘要
- 验证：xlsxedit 单测（公式/逐行变换/填充/替换/拆分/清洗/错误分支）+ app 校验
  单测 + 真实 LibreOffice 重算冒烟；go test ./... 全绿；tsc + vite build 通过；
  vitest 53 例全过

## v2.8.1「中期编辑 · Excel 单元格级预览」（2026-08-09）
> P0-③ 第一步：xlsx 预览从「转 Markdown 弱表格」升级为「单元格级保真视图」——
> sheet 切换、公式标识与公式栏、样式近似还原、合并单元格与列宽，为后续
> 「选中区域 → 指令（公式/清洗/透视）→ 写回」的单元格编辑打底。
- 新增 internal/office/xlsxpreview：excelize v2.11.0（已修复 2026 年三个 CVE）
  提取结构化单元格 JSON（值/公式/类型/样式/合并/列宽）；超大表格截断到
  2000 行 × 100 列并标记；图表/宏表自动跳过；.xls 保持原 markdown 兜底
- GaeaPreview 对 .xlsx 返回 kind=xlsx（body 为 JSON），wire 契约复用原字段
- 前端新增 XlsxPreview 组件：sheet 标签切换、公式栏（点击 fx 单元格显示公式）、
  表头行列冻结、样式（加粗/填充/对齐/边框/颜色）、合并单元格 colspan/rowspan、
  列宽近似；侧栏与弹层两个预览入口接入
- 验证：xlsxpreview 单测（样式/公式/合并/列宽/多 sheet/截断）+ app 层
  GaeaPreview 端到端；go test ./... 全绿；tsc + vite build 通过；vitest 52 例全过

## v2.8.0「中期编辑 · Word 框选即改 / 修订模式」（2026-08-09）
> P0「中期编辑 · Word 框选即改」落地：在保真预览底座上，选中文字 → 指令
> （AI 四预设 + 自定义）→ AI 生成替换 → 以 Word 修订模式（w:del + w:ins）
> 就地写入并重渲染，其余内容与版式零扰动。
- 新增 internal/office/docxedit：字节级 XML 手术（不重建 OOXML）定位选中文本并执行
  修订式替换；保留 run rPr 格式、应用实体转义、空白折叠匹配兜底、优先修订非 hyperlink
  段落（TOC 目录项不被 docx-preview 映射 ins/del）；选区包含图片/制表符/换行等特殊
  格式时明确拒绝
- 新增 GaeaOfficeEditText 绑定（办公向改写提示词：关键信息不变、纯文本输出，走 office
  功能绑定路由）+ GaeaDocxApplyEdit（修订写入并返回更新预览，标记作者「gaea AI」）
- 前端 DocxPreview 新增「框选即改」工具栏：选中文字后出现，四类快捷指令
  （润色/精简/翻译中文/扩写）+ 自定义指令输入，diff 预览（原文 → AI 替换）后
  「应用修订 / 放弃」；应用后修订样式重渲染
- 验证：docxedit 单测（单 run 拆分/跨 run 切分/XML 转义/空白折叠兜底/超链接优先/
  特殊格式拒绝）+ 真实 26MB 方案文档冒烟（0.9s 修订、合法回写、docx-preview 映射
  <ins>/<del>）；go test ./... 全绿；tsc + vite build 通过；vitest 49 例全过

## v2.7.9「通用办公 · 粘贴图片一键提取文字」（2026-08-09）

> 便捷入口闭环：截图/扫描件贴进对话后，点图片附件的「提取文字」即可用本地 OvisOCR2
> 常驻服务抽出图中文字，作为用户消息发给助手——不用再靠模型读图，离线、秒级。
- 新增 GaeaOCRText 绑定（docmd.OCRImageText）：复用常驻 llama-server（自动拉起/共享端口），
  图片不存在或服务不可用时给出明确错误
- Composer 图片附件新增「提取文字」按钮（FileText 图标，位于「识图」旁），识别中显示 spinner，
  完成后以【图片文字提取：文件名】发送给助手；与既有「识图」（本地视觉模型）互补——
  识图给描述、提取文字给原文
- 验证：docmd 新增 OCRImageText 端到端测试（GAEA_TEST_OCR_IMAGE 门控，实测 0.76s 返回
  「项目周报…营收 120 万元…」）；go test ./... 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.8「扫描件 OCR · 常驻服务 + format_convert 闭环」（2026-08-09）

> 多页扫描提速 + 内置转换器闭环：llama-server 常驻（按需拉起、共享 8137 端口），
> 多页 PDF 不再每页重载模型（单页约 5s → 整页 2.4s）；format_convert/文件预览的扫描件
> 兜底从「提示装 tesseract」改为直连本地 OvisOCR2 服务，tesseract 降为兜底。
- 常驻服务：ocr_local.py 与 docmd 共用同一端口（GAEA_OCR_URL/GAEA_OCR_PORT 可覆盖），
  健康检查发现未运行即按需拉起 llama-server（Vulkan，隐藏窗口），多页逐页复用；
  路径用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL / GAEA_OCR_MMPROJ 覆盖
- format_convert 闭环：docmd.ocrPDF 先试 OvisOCR2 服务（按页 POST），不可用时退回
  tesseract；findPdftoppm 探测真实 poppler exe（本机 PATH 里的 .cmd 包装器指向
  不存在路径，直接执行会失败，回退 codex 运行时自带的 pdftoppm.exe）
- 修复扫描件被误当文本：extractRawText 曾把 PDF 对象字典/trailer/ICC/图像流的可打印
  垃圾当正文返回，OCR 永远不触发——新增 stripNonTextStreams（关键字感知流扫描，只保留
  含 BT/ET/Tj 的文本流）+ 内容质量门槛（按空白分词、拒绝高熵符号串），扫描件正确落入 OCR
- 文本提取增强：extractPDFText 支持 TJ 数组的十六进制字符串（UTF-16BE，Word/LibreOffice
  中文 PDF 常见），带 BOM/无 BOM/ASCII 自动判别
- 验证：docmd 新增 httptest 客户端、hex 解码、流剥离、端到端扫描件（env 门控）测试；
  go test ./... 全绿；pdf 技能已同步 ~/.codex/skills 镜像

## v2.7.7「扫描件 OCR · OvisOCR2 本地文档解析」（2026-08-09）

> 按用户建议安装专用本地 OCR 模型：OvisOCR2（0.8B 端到端文档解析，Qwen3.5-0.8B 后训练，
> OmniDocBench 96.58，GGUF + llama.cpp Vulkan，Apache-2.0）成为扫描件 OCR 首选通道，
> 直接输出 Markdown（含 LaTeX 公式与表格），中文印刷体/表格识别显著强于 WinRT。
- 安装：llama.cpp b10333（Vulkan，适配 Radeon 8060S 核显）+ OvisOCR2-Q5_K_M（578MB）+
  mmproj-F16（205MB），落位 C:\AI\gaea-ocr（约 850MB）
- ocr_local.py 新增 --mode ovis 与自动链路：auto = OvisOCR2 → RapidOCR → WinRT → 本地视觉模型，
  文本过短/为空时逐级降级；路径可用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL /
  GAEA_OCR_MMPROJ 环境变量覆盖
- 顺带补齐 RapidOCR（PP-OCR 转 ONNX，隔离 venv C:\AI\gaea-ocr-env）作为 Ovis 缺失时的离线兜底
- 实测：真实扫描 PDF（微软雅黑渲染后嵌图）→ Ovis 识别「项目周报：本周完成 8 项需求 /
  营收 120 万元，同比增长 18% / 修复目标：砷 ≤ 60 mg/kg（GB 36600-2018）」，
  公式转 LaTeX、单页约 5s（含模型加载）；WinRT / RapidOCR 通道同步验证
- 验证：pdf 技能与 ~/.codex/skills 镜像同步；go build/test 不受影响

## v2.7.6「性能专项 · 大文件转换提速」（2026-08-09）

> P1「性能专项」落地：先用基准量化 format_convert 热点再针对性优化——
> docx→Markdown 百页级文件提速约 12 倍（69ms→5.8ms），PDF 提速约 4 倍（5.0ms→1.2ms）。
- 根因：docxToMarkdown 主循环每次迭代都对剩余全文搜索 `<w:p>` 与 `<w:p ` 两种形式，
  带属性形式在多数文档中不存在，导致 O(n²) 全文扫描（CPU profile 显示 findOpenTag 占 83%）
- 修复：新增 nextOpenTag 单趟扫描（按 `<` 推进 + 标签名边界判断），一次定位最早的段落/表格标签；
  PDF 页数统计从逐 rune 转 string 的热循环改为 strings.Count("/Type /Page")；
  xlsx 工作表按名建索引替代逐 sheet 全文件扫描，sharedStrings 索引解析改用 strconv.Atoi
- 基准（新增 docmd bench_test.go：gooxml 构造百页 docx / 千行 xlsx / 1500 页 PDF）：
  docx 300 段 + 20 表 69ms→5.8ms（约 12x）；PDF 1500 页 5.0ms→1.2ms（约 4x）；xlsx 1000x10 微调
- 验证：docmd 既有用例全部通过（表格 round-trip / 属性段落），go test ./... 全绿

## v2.7.5「方案校验 · 原文定位 + 整改建议」（2026-08-09）

> P1「方案校验规则引擎」增强：全面检查从「报问题」升级为「指出在哪、怎么改」——
> 每条发现带整改建议与原文定位（章节 + 上下文摘录 + 一键跳转）。
- CheckItem 新增 Suggestion（整改建议）与 Locations（原文定位：sectionId/excerpt/offset）：
  废标条款响应、数据一致性（缺失事实/工期冲突/暗标单位）、跨章节重复、暗标格式（加粗/删除线/emoji）、
  规范引用（未引用/编号存疑）逐条补齐可照做的整改建议
- 原文定位：locateNeedle 忽略空白差异在章节内定位命中，excerptAround 按 rune 截取上下文摘录
  （不切坏 UTF-8，前后省略号）；重复检测给出主章节与重复章节两处定位；AI 评分覆盖检查的
  suggestion 同步透传到整改建议
- 前端：校验报告新增「整改建议」行与原文定位 chips（摘录 + 定位按钮，点击跳转对应章节）
- 验证：新增 check_advice 测试（建议/定位/双章节/覆盖建议透传/摘录 UTF-8 安全）；
  go test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.4「事实底座 · 长期记忆沉淀」（2026-08-09）

> P0「记忆与自进化」落地：事实底座不再止步于会话内——一键把交付前沉淀的事实写入长期记忆，
> 后续对话自动加载，越用越懂你、不用反复交代。
- 新增 GaeaFactBasePromote 绑定 + 侧栏「沉淀为长期记忆」按钮：把当前会话事实底座逐条写入
  memory 存储（kind=semantic），按稳定 ASCII slug 去重（中文 key 走 hash 兜底），
  重复沉淀同 key 原位更新不产生重复条目；preference 分类映射为用户画像（type=user），其余为项目事实
- 面板按钮点击后 toast 反馈沉淀条数；事实底座本身保留，可继续编辑/清空
- 验证：新增 promote 单测（写入/去重更新/中文 slug 稳定性/分类映射/单行摘要）；go test 全绿、
  tsc + vite build 通过、vitest 47 例全过

## v2.7.3「事实底座 · 一稿多用」（2026-08-09）

> P0「多形态成果交付」核心闭环：通用办公引入 任务 → 事实底座 → 多形态交付 的默认工作流，
> 交付物（docx/pptx/xlsx/图表）基于同一事实底座生成，跨交付物保持一致、可回看、可复制。
- 新增 fact_add / fact_list / fact_clear 三个会话级事实底座工具（ExtraTool）：
  fact_add 按 key 沉淀/修正事实（含来源与分类，空值即删除），fact_list 输出 Markdown 表格，
  fact_clear 开启新任务时清空；事实按会话持久化到 .gaea/sessions/<session>-facts.json
- 侧栏新增「事实底座」面板：事实列表（key/值/来源）+ 复制 Markdown + 清空，随 fact_* 工具结果
  与回合结束实时刷新；删除会话时事实底座同步清理
- 办公提示词新增「事实底座（一稿多用）」纪律：交付任务先沉淀后交付，交付物基于同一底座生成，
  事实有更新先 fact_add 修正再重新生成，新任务先 fact_clear 隔离上一任务事实
- 验证：新增 factbase 包单测（upsert/清空/Markdown 转义/round-trip/路径）+ app 侧工具与绑定测试；
  go build/test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.1「办公三件套核心闭环修复」（2026-08-09）

> 触及核心：修复 Word/Excel/PDF 三个工具的真实断点，端到端实测打通。
> 此前通用办公的 docx 技能依赖未安装的 node docx-js 与 pandoc，Excel 公式重算在 Windows
> 上因 soffice 不在 PATH 而失败——这三个断点全部修复并补回归测试。
- format_convert（docmd）：修复 docx→Markdown 表格解析把 `<w:tcPr>` 等 XML 当文本的 bug；
  `findWt` 偏移错位导致段落/单元格文本错乱；兼容带属性段落（w14:paraId）与 `<w:tabs>` 内嵌标签
  （新增 TestConvertDocxTableRoundTrip / TestConvertDocxAttrParagraphs 回归测试）
- docx 技能：新增 scripts/create_docx.py（python-docx，环境已装）——Markdown/JSON spec → 排版 Word
  （A4/Letter、封面、目录域、标题层级、表格、列表、页眉页脚、页码、图片），替代未装的 docx-js 成为主路径；
  读取兜底改为 gaea 内置 format_convert；SKILL.md 依赖清单同步更新
- xlsx 技能：scripts/office/soffice.py 增加 Windows 安装路径自动探测（Program Files / (x86) / LOCALAPPDATA），
  recalc.py 不再因 soffice 不在 PATH 失败；SKILL.md 补批量合并/统一格式流程
- docx 技能 soffice.py 同步路径探测（docx→PDF 转换同样受益）；pdf 技能补 PDF→Word 与表格→xlsx 闭环流程
- 实测：Markdown→docx→format_convert 表格 round-trip 正确；openpyxl 写公式→LibreOffice 重算 4 公式 0 错误
  （SUM/差额/利润率读回 220/130/90/40.9%）；reportlab PDF→pdfplumber 表格提取→xlsx 结构完整
- 验证：go test ./... 全绿（docmd 新增 2 例）；技能修改已同步镜像 ~/.codex/skills

## v2.7.2「本地模型优势落地 · 扫描件本地 OCR」（2026-08-09）

> 善用 gaea 本地模型资产：扫描件 PDF 的 OCR 不再依赖未安装的 pytesseract，
> 改走本地双通道——Windows 原生 OCR（WinRT，离线零成本）+ 本地视觉模型（herdsman Qwen3.6）兜底。
- pdf 技能新增 scripts/ocr_local.py：PyMuPDF 渲染 PDF 页（300 DPI）→ windows-ocr.ps1（WinRT zh-Hans）
  → 文本过短/为空时降级本地视觉模型（GAEA_VISION_BASE_URL/GAEA_VISION_MODEL 可覆盖）；
  输出 UTF-8 文本或 JSON（每页 + 工具来源），支持 --mode auto|winrt|local
- pdf 技能自带 scripts/windows-ocr.ps1（WinRT OcrEngine 离线调用，与 ds-vision-skill 同源）
- SKILL.md：扫描件 OCR 节改为本地 OCR 主路径，pytesseract 降为可选外部方案
- 办公提示词新增「本地能力优先」纪律：扫描件/图片文字提取优先本地 OCR 或 vision，敏感文档本地处理不出机
- 实测：中文 docx→PDF→本地 OCR 识别出「项目周报 / 本周完成 8 项需求 / 营收 120 万元，同比增长 18%」；
  表格 PDF→OCR 数字与表头全部识别；英文/中文双链路均离线跑通
- 验证：技能修改已同步镜像 ~/.codex/skills

## v2.7.0「通用办公强化 · PPT 交付 + 技能沉淀 + 便捷入口 + 后台任务」（2026-08-09）

> 按《市场调研：通用办公 AI 智能体》P0 结论落地：新增 pptx 演示文稿技能（python-pptx 一键成稿）、
> 欢迎页新增「演示文稿」入口、一次成功对话一键沉淀为可复用技能、粘贴剪贴板图片即转图片附件上下文、
> 侧栏后台任务面板（运行中任务 + 完成 toast）、办公提示词增强（PPT 交付纪律 + 偏好主动记忆沉淀）。
- pptx 技能：新增 `.gaea/skills/pptx` 并镜像 `~/.codex/skills/pptx`（SKILL.md + scripts/create_pptx.py，
  16:9 封面/章节要点/演讲备注，缺 python-pptx 时自动 pip 安装，生成后回读校验页数）
- 前端入口：欢迎页核心能力新增「演示文稿」卡 + 内置技能新增 pptx chip；icons 兼容层新增 FilePpt
- 提示词：SingleModelPrompt 执行纪律加入 pptx 技能与演示大纲流程；新增「记忆沉淀」检查点
  （用户偏好/项目事实/踩坑经验主动 remember，避免重复交代）
- 便捷入口：Composer 粘贴剪贴板图片（截图/网页图）不再静默丢弃，转为图片附件上下文（复用 SavePastedImage）
- 后台任务：侧栏新增「后台作业」面板（运行中 bash/task 列表 + 状态点 + 相对时间，仅展开态显示）；
  JobDoneNotifier 在任务从运行列表消失时自动 toast「后台任务已完成」
- 技能沉淀：助手消息新增「沉淀为技能」操作，弹窗预填本次任务/回答，确认后写入 .gaea/skills/<name>/SKILL.md
  并镜像 ~/.codex/skills，热加载后立即以 /技能名 调用（GaeaCaptureSkill + skill.RenderSkillFile，含校验/覆盖/测试）
- 验证：go build/test 全绿（含 skill capture 新用例）、tsc + vite build 通过、vitest 47 例全过、
  pptx --demo 端到端生成 3 页演示文稿

## v2.6.9「搜索修复 · 人格重设计 · 移除 VoxCPM2」（2026-08-09）

> 通用办公搜索修复（新增 Bing/DDG 兜底 + 代理接入，不再报「所有搜索引擎失败」）；
> 聊天板块联网搜索接通（ChatSend searchEnabled + 开关生效）；gaea 人格与提示词整体重设计
> （统一 gaea 身份，清除 Ackem/Hermes/大地女神旧文案，清空 config.toml 土壤修复旧覆盖）；
> 首页语音固定 gaea；按用户要求移除实测不达标的 VoxCPM2，本地 TTS 保留 CosyVoice2。
- web_search：Bing/DuckDuckGo Lite 无 key 兜底、代理复用 web_fetch 链路、修复 cancel bug 与 429 重试、UA 换浏览器标识
- 聊天搜索：ChatSend(searchEnabled) 普通+角色对话自动联网注入结果；前端开关生效（普通模式也显示）；whisper WebSearch Bing 优先约 0.3s 带标题/链接
- 人格：personality/canon/product_identity/main_chat 提示词重写，gaea 预设去 goddess 标签；20+ 处 Hermes/Ackem → gaea；AboutPanel/首页语音标签同步
- 首页语音：固定 gaea，删除聊天人格自动同步语音的副作用
- 移除 VoxCPM2：引擎/前端 6 文件删除，本地 C:\AI\voxcpm、C:\AI\llama-omni 清理，释放约 14GB
- 验证：go build/test 全绿、tsc+vite build 通过、wails build 成功（releases/gaea-v2.6.9.exe + SHA256SUMS）

## v2.6.8「模型中心一键启动本地 TTS 服务」（2026-08-09）

> 本地 TTS 不再需要手动运行启动脚本：gaea 启动时自动保活 CosyVoice2 / VoxCPM2；
> 模型中心点击模型的「启动」按钮也会即时拉起对应服务；「测试连接」与语音合成前兜底同样自动拉起。
- 新增 internal/app/tts_service.go：core.ensureLocalTTSService（幂等）+ mediaState.StartLocalTTSService（Wails 绑定）
- 探测：CosyVoice2 http://127.0.0.1:8010/v1/models；VoxCPM2 http://127.0.0.1:8020/v1/status
- 拉起（隐藏窗口 CREATE_NO_WINDOW，不阻塞 UI）；异步轮询就绪（cosyvoice ≤10s / voxcpm ≤180s），就绪/失败通过 tts-service-status 事件通知前端
- 验证：go build/test 全绿、tsc + vite build 通过、wails build 成功（releases/gaea-v2.6.8.exe + SHA256SUMS）

## v2.6.7「VoxCPM2 Vulkan GGUF 加速 · 4 音色替换」（2026-08-09）

> 实测 VoxCPM2 三层架构落地：8030 llama-tts-server（llama.cpp-omni + Vulkan，Q8_0+F16 GGUF）为主后端；
> 8021 ROCm PyTorch 备胎；8020 adapter.py 统一入口（gaea 零改动）。
- 关键修复：SSLServer 空证书导致 bind 失败（改普通 httplib::Server）；AudioVAE 参考特征 frame-major 布局对齐 Python（克隆从近静音恢复）
- 性能：短句克隆 RTF 0.65–0.84（6 步/CFG 1.5），语音设计 0.57–0.60；对比 ROCm 5 步 RTF ≈0.06–0.12，整体快 1.5–8 倍
- 音色：CosyVoice / VoxCPM2 统一替换为火山引擎 4 音色（中文女/男、英文女/男）；参考音频 ≤3s/16kHz，适配器自动音量归一
- 验证：go build/test 全绿、tsc + vite build 通过、四音色端到端实测通过；wails build 成功（releases/gaea-v2.6.7.exe + SHA256SUMS）
## v2.6.6「VoxCPM2 本地语音引擎接入」（2026-08-09）

> 模型中心新增「VoxCPM2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8020），
> 2B 扩散式多语种 TTS、48kHz 高保真输出，内置 7 音色 + 参考音频零样本克隆 + 声音设计；
> ROCm 驱动 Radeon 8060S 核显 + TunableOp 调优，实测 RTF ≈ 2.0。
- 引擎接入：modelengine 新增 `voxcpm` 类型与内置引擎，启动自动补齐 `VoxCPM2` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 服务端（`C:\AI\voxcpm\server.py`）：OpenAI 兼容 `/v1/audio/speech`、
  `/v1/audio/info`、`/v1/models`、`/v1/voices`（参考音频注册克隆音色，持久化）；
  Python 3.12 venv + torch 2.9.1+rocm7.2.1（AMD ROCm Windows）+ voxcpm 2.0.3
- 性能：ROCm 核显推理 + TunableOp GEMM 调优缓存（4~5 倍）；CFG 1.5 避免长文本跑飞重试；
  实测 7.7s 中文约 16s（RTF ≈ 2.0），短句 RTF ≈ 1.35；启动预热后首个请求即达稳态
- 音质：48kHz 输出；内置 7 个与 CosyVoice2 同源参考音色（中文女/男、英文女/男、日语男、粤语女、韩语女）；
  支持声音设计（自然语言描述音色）与参考音频克隆
- 验证：go build/test 全绿、tsc 通过；服务端直连合成 / 克隆 / 音色列表实测通过
  （详细记录：`docs/2026-08-09-voxcpm2-integration.md`）

## v2.6.5「CosyVoice2 LLM 核显加速」(2026-08-09)

> CosyVoice2 LLM 环节从 PyTorch CPU 切换到 llama.cpp GGUF + Vulkan（Radeon 8060S 核显），
> 整条合成管线提速约 8–10 倍：短句 6.5s→~1.5s，长句 24.5s→~2.8s。
> 默认使用 f16 GGUF（音质最接近原始权重），q8 为更快备选。
- 调研：Tinysoft/Cosyvoice2-0.5B-GGUF + llama.cpp PR #14711（Qwen2 bias 支持）
- 引擎：`cosyvoice/llm/gguf_engine.py`（token 直喂、KV cache、复刻 ras_sampling），
  `Qwen2LM.load_gguf()` 接入，实测词表布局 logits 相关性 0.9999
- 服务：server.py 默认加载 `gguf\cosyvoice_f16.gguf`（环境变量 `COSYVOICE_LLM_GGUF` 可切），
  启动预热 shader；失败自动回退 torch
- 修复：`mask_to_bias` fp16 下 -1e10 溢出 -inf 导致 ONNX Softmax NaN（fp16 改用 -3e4）
- 否决：fp16 flow estimator（误差累积，波形相关性 0.016）、LLM 单步 ONNX（链式发散）、
  bf16（EOS 失效）、flow 4 步（无稳定收益）
- 验证：直连 `/v1/audio/speech` 短句 ~1.4–1.8s、长句 ~2.8s；7 音色/中英日正常
  （详细记录：`docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`）
- 发布：`go build/vet/test` + `tsc/vite build` + E 系列回归全绿；`wails build` 成功，
  `releases/gaea-v2.6.5.exe` + `SHA256SUMS-v2.6.5.txt` + `v2.6.5.md`，桌面 `gaea.exe` 已同步

# gaea · 多功能 AI 助手

## v2.6.4「CosyVoice2 提速 + 模型中心音色选择」(2026-08-09)

> CosyVoice2 合成提速约 35%（短句 6.5s → 4.1s）：flow 解码器切 ONNX + DirectML（AMD 核显），
> 采样步数 10→5，音质与 torch 路径相关性 1.0000；
> 「功能绑定 → 聊天语音」卡片新增音色选择（xAI / CosyVoice2 / Herdsman）。
- CosyVoice2 提速：`flow.decoder.estimator.fp32.onnx` + `DmlExecutionProvider` 替换 torch estimator，
  flow 步数 5；服务端 server.py 内置，重启服务即生效
- LLM 优化探索：Qwen2 单步 ONNX 导出成功（fp32 CPU 28.8ms/步、DML 10ms/步），
  但链式推理数值发散导致音频失真，暂不启用；bf16 破坏 EOS 采样，弃用
- 模型中心：聊天语音绑定卡新增「音色」下拉（xAI 26 音色 / CosyVoice2 7 音色 / Herdsman 服务端列表），
  选择即写 `tts_voice` 持久化
- 验证：直连 4.1s/0.92s wav；go test 全绿；tsc + vite build 通过
  （releases/gaea-v2.6.4.exe，SHA256SUMS-v2.6.4.txt）

## v2.6.3「CosyVoice2-0.5B 本地音色克隆接入」(2026-08-08)

> 模型中心新增「CosyVoice2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8010），
> 聊天语音可绑定 CosyVoice2-0.5B，内置 7 个音色并支持参考音频零样本克隆。
- 引擎接入：modelengine 新增 `cosyvoice` 类型与内置引擎，启动/刷新自动补齐 `CosyVoice2-0.5B` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 音色支持：`/v1/audio/info` 拉取音色列表（中文女/男、英文女/男、日语男、粤语女、韩语女），
  设置面板/聊天设置新增 CosyVoice 选择器；无效音色回退「中文女」，
  `defaultVoiceForModel`/`ttsVoiceForModel` 补齐 cosyvoice 默认音色
- 服务端：`POST /v1/audio/speech`（OpenAI 兼容）+ `POST /v1/voices`（参考音频注册新音色，持久化到 spk2info.pt）
- 实测：直连合成 6.5s/0.92s 音频（RTF 6.8x，PyTorch CPU）；gaea 客户端端到端 38KB wav；
  go test 全绿、tsc + vite build 通过
  （releases/gaea-v2.6.3.exe，SHA256SUMS-v2.6.3.txt）

## v2.6.2「xAI Grok TTS 接入」(2026-08-08)

> xAI 引擎内置 `grok-tts` 云端语音模型：模型中心「功能绑定 → 聊天语音」可绑定 xAI，
> 语音对话/朗读优先走 Grok 音色（Eve/Ara/Rex/Sal/Leo + 旗舰 21 个），
> 实测合成约 1.4~2.4 秒，快于本地 qwen3-tts（声码器 CPU）的 2.7~3.3 秒。
- xAI TTS 路由：语音管道新增 `tryEngineTTS`，xAI 走 `POST /v1/tts`（复用 OAuth token），
  聊天语音绑定 → 全局 TTS → 自动路由均生效；流式朗读选中 xAI 时同样优先云端
- 模型中心：xAI 引擎保活内置 `grok-tts`（启动/刷新均自动补齐），
  功能绑定与「语音模型」页可直接选择；设置面板/聊天设置新增 xAI 音色选择器，
  无效音色自动回退 `eve`，`tts_voice` 持久化
- xAI TTS 客户端完善：`SynthesizeWithMime` 返回真实 MIME，音色列表静态校验 + 单测
- 验证：`go build/test ./internal/{tts,modelengine,app}/...` 全绿；`tsc` + `vite build` 通过；
  直连 `api.x.ai/v1/tts` 实测 HTTP 200 / audio/mpeg；`wails build` 成功
  （releases/gaea-v2.6.2.exe，SHA256SUMS-v2.6.2.txt）

## v2.6.1「语音交互打通 · 音色可配置 · 首页语音角色与聊天一致」(2026-08-08)

> 按 Herdsman 新版接口重新打通语音对话：AI 回复后 TTS 朗读恢复正常，不再卡在“正在聆听”；
> 语音音色可在设置面板直接选择并持久化；首页语音角色与聊天板块保持一致。
- 语音交互打通：TTS `audio_url` 支持 data URI / 相对路径 / 绝对 URL 三种形态并返回真实 MIME；
  合成前动态查询 `/v1/audio/info` 音色列表，`Cherry` 等已移除音色自动回退可用音色；
  修复 ASR `/v1/v1/audio/transcriptions` 双重前缀 404；TTS 播放期间保持“AI 回复中”状态
- 音色设置：聊天面板「语音设置」新增 Herdsman 音色选择器（实时拉取服务端支持列表），
  写入 `~/.gaea_config.json` 的 `tts_voice` 持久化；设置页聊天设置同步
- 首页语音角色：不再写死 gaea，读取聊天板块保存的同一角色键并动态显示标签；
  后端新增 `voice_personality` 持久化
- 普通对话联动：聊天板块切换「普通对话」时语音回复使用中性助手口吻，
  首页语音同步为「普通对话」；切换人格时首页语音跟随该人格
- 模型中心「功能绑定」新增聊天语音模型：单独绑定聊天语音的引擎+模型（持久化），
  未绑定回退全局 TTS；语音模型列表随引擎自动刷新，便于后续扩展
- 验证：go build/vet 全绿；tsc + vite build OK；前端 E 系列回归守卫 OK；
  直连 Herdsman 端到端验证合成 122KB 真实语音；wails build 成功
  （releases/gaea-v2.6.1.exe，SHA256SUMS-v2.6.1.txt）

## v2.6.0「小说创作工作台重设计 · 角色卡补齐/剧照/合并」(2026-08-08)

> 小说创作页改为三栏写作工作台（章节树 / 编辑器 / 创作设置），写作技能与生成参数
> 常驻可调；章节捕获的角色卡补齐 AI 补档、剧照生成与同名合并；修复设定未注入、
> 办公目录被误判为代码工程等问题。
- 小说创作界面：三栏写作工作台——左章节树（未写/生成中/已写入状态点）、中纸面化
  编辑器（标题/状态/字数/重写/保存/空状态引导）、右创作设置面板（设定摘要/技能/
  生成参数/剧情方向/统计）；技能选择常驻并显示说明与适用场景
- 生成设置真实生效：目标字数（不足自动续写）与生成温度由界面传入后端
  （CreateChapter 新增参数，默认 5000 字/服务端默认温度）
- 正文标题约束：提示词禁止 AI 自拟章节标题或编号行，章节编号由系统管理，
  杜绝「第七章…这是第一章」错乱
- 小说设定注入修复：创作页在挂载/切书/开向导/生成前均重读最新设定，提示词必注入
- 章节角色捕获：新角色只写本书 characters.json，需手动「一次性迁移」进全局库；
  角色页未入库标记 + 刷新联动
- 角色卡能力：未入库角色支持 AI 补齐空缺字段、生成剧照（自动补全关键描述）、
  合并同一人的不同称呼（引用与关系重定向去重）
- 通用办公：Profile.Scan 语言改为按工程清单真实检测，办公目录不再被标成 Go 工程；
  项目画像/技能/记忆基于工作区扫描；开发 mock 的 coding agent 提示词改为办公助手
- 技能加载：修复 SKILL.md 多行 applies_to 解析为空的问题
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 47 例全过、
  wails build 成功（releases/gaea-v2.6.0.exe，SHA256SUMS-v2.6.0.txt）

## v2.5.0「全界面科幻视觉重设计 · 绘梦图生图/文生视频 · 角色库模型绑定」(2026-08-08)

> 设置、首页、聊天欢迎屏、小说、绘梦五大板块统一为「玻璃 HUD + 单一强调色」设计语言；
> 绘梦新增图生图与文生视频（ComfyUI 后端）；模型中心功能绑定实时同步并新增角色库绑定；
> 角色库剧照支持独立后端/模型；本地视觉识别管线修复 UTF-8 编码并切换到 Qwen 视觉模型。
- 设置面板：左侧导航改为顶部平铺分类磁贴（吸顶可切换），统一 HUD 角标/聚焦/按压反馈；
  修复 flex + margin auto 导致容器收缩为内容宽度的问题（宽度 641px → 1080px）
- 首页：语音中枢升级为视觉主角——HUD 角标框、雷达脉冲环、声谱均衡条、等宽遥测读数，
  聆听/回复时扫描光带掠过面板；新增 380px 语音球与实时音量展示
- 聊天欢迎屏：普通模式用 VoiceChatOrb、角色模式用 CompanionAvatar（随语音状态变色），
  配 HUD 角标、脉冲环、遥测行；建议卡升级为键盘可达（role/tabIndex/focus-visible）；
  角色模式空状态隐藏重复人格条，修复 orb 被 flex 压缩的问题
- 小说板块：默认 Tabs 改为顶部二级导航平铺（书架/设定/角色/创作/阅读/导出），子页保持挂载；
  新增统一设计层 novel-workspace.css——玻璃面板、HUD 面板头、章节节点树、纸面化衬线编辑面、
  antd Tabs 玻璃化；大纲面板硬编码紫色改为 var(--gaea-glow)；设定页改为左右双栏
  （左设定编辑器 + 右设定 Agent 常驻，修复 flex 宽度收缩）
- 绘梦：顶部三模式导航（文生图 / 图生图 / 文生视频）；图生图支持上传/拖拽参考图 + 重绘幅度；
  文生视频支持分辨率/时长帧率预设；后端新增 ComfyUI 图生图工作流（LoadImage+VAEEncode+低 denoise）
  与 LTX-Video 文生视频工作流，输出解析扩展到 images/gifs/videos，结果舞台/历史胶片/灯箱支持视频；
  新增 GenerateMedia 绑定；修复引擎列表为空时页面崩溃
- 模型中心：功能绑定面板监听 feature-model-changed 事件实时同步（其他页面改绑定即时刷新）；
  聊天合并轻语、办公合并方案（office+gaea 双写），删除重复行；新增角色库 LLM 绑定
  （func_characterlib_*，角色生成/补全走独立绑定）
- 角色库剧照：新增独立图片后端/模型绑定（portrait_backend/portrait_model，空=跟随绘梦），
  模型中心新增「角色库剧照」卡；删除小说卡上的剧照模型标签
- 视觉识别管线：vision 技能默认模型切换为 Qwen3.6-35B-A3B-Uncensored（布局识别准确率明显提升）；
  修复 PowerShell 5.1 下 UTF-8 编码问题（脚本转 BOM + 改用 .NET HttpClient），中文输出不再乱码
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 37 例全过、
  Edge headless 各界面实测 + 视觉模型复核

## v2.4.5「通用办公欢迎页重设计」(2026-08-08)

> 通用办公对话窗口的欢迎界面从土壤修复时代的旧版，重设计为贴合当前定位的通用办公入口。

- 内容更新：移除场地调查/投标标书/修复方案/污染风险/成本测算等土壤修复专项卡片，
  改为 6 个通用办公核心能力（文档撰写/表格处理/格式转换/图表生成/报告拼装/知识沉淀）
- 新增内置技能区：format-convert / chart-builder / doc-assemble / docx / xlsx / pdf 六个技能 chips，
  点击即填入对应任务提示词
- 视觉升级：主视觉改为「GAEA OFFICE + 今天想做什么？」大标题层级，logo 带光晕角标，
  能力卡 hover 顶部高光线 + 箭头提示，新增渐次入场动画（尊重 prefers-reduced-motion）
- 保留工作区/模型 pill、最近会话区并同步优化样式
- 文案清理：welcome.tagline 与加载页 skeleton 描述从「土壤修复引擎」改为通用办公，
  删除 locale 中无人引用的土壤修复快捷卡片条目
- 验证：tsc + vite build OK；vitest 37 例全过；Edge headless 实拍确认布局正常

## v2.4.4「文件预览与右侧面板精简」(2026-08-07)

> 右侧面板去掉「消息/报告」标签并默认折叠；对话内可直接点击文件路径打开全新的预览阅览面板。

- 右侧面板：删除「消息」与「报告」标签页（MessageNavigator / ReportPreviewPanel），保留「文件」「统计」；
  面板改为默认折叠，点工具栏按钮展开
- 对话内文件预览：Markdown 渲染中的本地文件路径（.md/.docx/.xlsx/.pdf/.png 等）渲染为可点击的文件 chip，
  用户消息里的 @ 附件同样可点击；点击打开居中大尺寸预览弹层（Esc/遮罩关闭，支持定位与外部打开）
- 预览面板重设计：后端新增 GaeaPreview 统一预览接口——图片返回 dataURL、Markdown/文本原文渲染、
  docx/xlsx/pdf 经 docmd 转 Markdown 内联预览（含 OCR 回退），不支持格式给出外部打开入口；
  工作区「文件」面板的预览同步升级为 Markdown 渲染
- 转换引擎抽取：format_convert 的 docx/xlsx/pdf→Markdown 逻辑迁至 internal/office/docmd 包，
  工具与预览面板共用一份实现；修复 openpyxl 内联字符串单元格（inlineStr）丢表头的问题
- 测试：新增 FilePreviewModal 3 例 + Markdown 本地文件链接 2 例，前端 37 例全过；go test 全绿

## v2.4.3「精简内置工具集」(2026-08-07)

> 按市场调研结论（同类产品 10~20 个核心工具、文档/表格用技能扩展）把内置工具从 38 个精简到 17 个核心工具。

- 删除 21 个冗余内置工具：计算器（calc_math/calc_stats/calc_unit）、压缩（archive）、
  电脑操作（computer_use）、甘特图（gantt_gen）、项目初始化（project_init）、工具链模板（run/save_template）、
  方案 agent 工具（proposal_list/write/export），以及被 ModelScope docx/pdf/xlsx 技能覆盖的文档专项工具
  （docx_read/docx_write/pdf_create/pdf_extract/pptx_create/xlsx_read/xlsx_write/doc_merge/csv_parse）
- 删除与 run_skill 重叠的 parallel_skills（保留 RunDAG 管道基础设施）
- 保留 17 个核心工具：文件与命令（read_file/write_file/ls/bash/bash_output/kill_shell/wait）、
  网络（web_search/web_fetch）、任务（todo_write/complete_step）、记忆与知识（memory_search/knowledge_add/knowledge_search）、
  技能（read_skill）、办公引擎（format_convert/chart_gen）
- 内置技能去工具依赖：chart-builder 改为 bash + python 读表、doc-assemble 改为 format_convert + bash 拼装，
  不再依赖已删除的 xlsx_read/csv_parse/doc_merge/docx_write
- 系统提示词与单模型执行纪律改写为「文档创建/编辑交给 docx/xlsx/pdf 技能，格式转换用 format_convert」
- 修复 chart_gen 在 Windows 被 python3 商店别名劫持的问题，并抑制 matplotlib 字体告警污染 JSON 输出
- 前端能力抽屉工具列表重建为实际 7 组 29 个工具（compact 模式对模型隐藏 kill_shell/wait），
  清理 git/notebook/edit 等死条目与报告来源旧映射
- 验证：go build + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过；format_convert 与 chart_gen 端到端验证通过

## v2.4.2「通用办公改造 · ModelScope 文档技能」(2026-08-07)

> 「智能办公」精简为「通用办公」：删除土壤修复专项工具与技能，安装 ModelScope 的 docx/pdf/xlsx 文档技能。

- 办公定位：DefaultSystemPrompt 与单模型执行纪律改写为通用办公助手（文档/表格/图表/演示/检索/方案报告/任务跟踪）
- 删除 6 个土壤修复专项工具：survey_report/bid_proposal/imple_plan/spec_query/spec_judge/material_query，
  并从 compact 描述/模式表中清理
- 内置子代理技能收敛为 3 个通用技能：format-convert / chart-builder / doc-assemble（含 wrapper 工具与测试）
- 删除项目内 5 个土壤办公技能（site-survey/risk-assessment/remed-design/bid-package/data-report），保留 skill-creator
- 安装 ModelScope 技能：docx / pdf / xlsx（SKILL.md + 完整 scripts/schemas/templates），
  同时写入 ~/.codex/skills 与 .gaea/skills，供 Codex 与 gaea 通用办公使用
- 前端文案统一：智能办公→通用办公，报告类型映射改为通用办公类型，移除 Hephaestus 残留
- 验证：go vet + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过

## v2.4.1「设置面板重设计」(2026-08-07)

> 按当前功能板块重构设置中心：左侧模块导航 + 全局搜索，清除死代码与重复面板。

- 布局重设计：顶部 Tab 改为左侧功能分组导航（通用 / 聊天 / 小说 / 绘梦 / 办公 / 模型 / 关于），
  窄屏自动转横向滚动 chips；保留全局搜索并可过滤分组
- 聊天：合并「AI 伴侣 + 默认人格 + 语音核心项」（语音对话/朗读回复/合成音色），
  移除无人读取的「主动搭话」开关与重复的「清除全部会话」；完整语音面板仍在聊天板块
- 办公：并入「方案编写模型」绑定（原方案 Tab 唯一有效内容），删除纯文档的「方案生成」说明
- 模型（新增）：找回此前丢失入口的全局「推理强度」，展示当前模型并提供「前往模型中心」跳转
- 关于：版本卡片 + 系统信息（配置路径）+ 可折叠更新日志，替换原系统 Tab
- 清理：删除死代码 EnginePanel、重复面板 VoicePanel/ProposalPanel、遗留 WhisperPanel/SystemPanel
- 验证：新增 SettingsPage 4 例回归测试（分组渲染/默认分组/搜索过滤/切换），
  `npm run test` 32 例全过，`tsc` + `vite build` OK，`scripts/ci.ps1` CI OK

## v2.4.0「网页调试桥接 · 办公引擎热加载」(2026-08-07)

> 新增 `GAEA_HTTP_PORT` 网页调试桥接：浏览器/手机直接驱动同一个 Go 内核（RPC + SSE），
> 办公引擎支持从磁盘热加载，技能/工具/插件变更无需重启桌面端即可生效。

- HTTP 调试桥接（`internal/httpbridge`）：`POST /api/rpc` 反射分发全部 Wails 绑定方法、
  `GET /api/stream?id=` SSE 事件推送（15s keep-alive）、`/api/health` 存活探针；
  `core.emit` 统一发布到桥接订阅者（无 Wails 上下文也推送）
- 前端 HTTP 模式：`runtimePolyfill` 补齐 EventsOn/EventsOff/EventsOnMultiple/EventsEmit，
  所有事件经 SSE 对齐桌面端——修复并发订阅只连首个事件的竞态，探测失败后桥接就绪可自动重连；
  `bridge.ts` 将 `window.go.app.App.*` 代理到 `/api/rpc`；Vite 将 `/api` 代理到桥接
- 办公引擎热加载：`GaeaReload` 重新读取磁盘持久化配置并重建 controller
  （Agent 参数/权限/沙箱/技能路径/插件），成功后广播 gaea-ready 令前端 store 重新拉取；
  失败时保持旧引擎继续运行，不替换任何状态
- 前端入口：能力抽屉（MCP/工具/技能）新增「热加载」按钮并展示重建后的工具/技能数量；
  设置→办公新增「从磁盘热加载」；三语 i18n 同步
- 验证：新增 httpbridge RPC/SSE 2 例 + GaeaReload 热加载 1 例；`go vet` + `go test ./...` 全绿、
  `tsc` + `vite build` OK、`scripts/ci.ps1` CI OK

## v2.3.0「界面焕新 · 办公整合」(2026-08-07)

> 在 v2.2.0 基础上迭代：角色库/首页/办公板块全面重设计，办公双模块合并为独立二级导航，随机生成能力补全（含人格）。
- 角色库界面重设计：档案墙（立绘主导 + 身份覆盖 + 档案眉），加载骨架/空状态/深浅色适配，去掉多色硬编码统一主题强调色
- 角色命名：28 个内置角色配中文人名（白霜/温言/苏晚晴/林小满/顾清霜/陆寒川/姜棠/谢临川/叶绵绵/许一诺/秦挽月/阮慈/周栗/季如烟/陈恪/沈墨白/程暖阳/席晚棠/霍承渊/虞栀/江野/晏观澜/裴聿修/夏知微/乐桃/顾衍/卫昭），gaea 保留原名
- 角色相卡面板重设计 + 随机生成完善：新增 `CharacterGenerateRandom`（all 全量随机/按字段随机），字段旁 ↻ 骰子单独随机、五维本地随机、性别/定位/状态/年龄即时随机
- 首页 AI 中枢启动器重设计：单一主题强调色、玻璃档案化卡片、加载/焦点/降级动效处理
- 移除窗口顶部流光扫描条（scanline-top / scanSweep）
- 办公板块整合：合并「办公 + 方案编写」为独立二级导航（智能办公/方案编写），`OFFICE_MODULES` 注册表可扩展，移除顶部重复「方案编写」入口
- 智能办公整层重设计：对话区铺满面板（修复全屏缩成一团）、正文行宽约束、清爽玻璃面层（侧栏/顶栏/消息/输入框/滚动条）
- 方案编写整层重设计：胶囊式标签导航、左侧项目导航唯一入口（移除重复「方案列表」标签页）、隐藏重复步骤条，消除层层标题
- 验证：28 项回归测试通过；`go build` + `tsc` + `vite build` OK；`wails build` 成功（build/bin/gaea.exe ~38MB）

## v2.2.0「统一角色库」(2026-08-07)

> 在 v2.1.0 基础上迭代六轮：全局统一角色库、小说单向引用+抽卡、聊天内选角色、角色记忆隔离、状态/记忆/追踪归集角色库、取消「轻语」称谓。

- 角色库架构重构：新增 `characterlib.db` 全局统一角色存储（小说字段 + 聊天字段同一行），内置人格种子化进库；项目只引用角色并携带项目内弧线状态；导入只增不改，杜绝小说反向污染角色
- 小说角色面板改为「抽卡」：从角色库随机抽取（性别/标签/可聊天过滤），不再自行生成角色；面板只编辑项目内定位/弧线/状态，全局设定去角色库
- 聊天只选角色：聊天内 PersonaPicker 选择器（搜索/头像/类型标签），角色管理/编辑/状态/记忆/追踪全部归集角色库
- 角色记忆隔离：事实/情节/图谱按会话恢复与合并写回，A 角色记忆不串 B 角色；追踪新增会话归属列并按角色持久化
- 取消「轻语」称谓，统称聊天；首页启动器、设置、模型中心、记忆中枢文案全面统一
- 验证：新增 30+ 回归测试；`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## v2.1.0「二代完善」(2026-08-07)

> 在 v2.0.1 基础上迭代六轮：模型中心完善、功能级模型启停、Cmd+K 引擎路由、OAuth 回归验证、前端 E 系列核对、小说链路审计。

- 模型中心：引擎状态持久化（`whisper_data/engines.json`）+ 稳定顺序 + 连接状态缓存；修复 `active_engine_id` 只存不读；DeepSeek 脱敏与真实活跃模型展示
- 模型路由：FeatureModelBar 启停改为功能级开关（`func_*_enabled`），不再误关全局引擎；Cmd+K 编辑走 novel 绑定；5 个 agent 未绑定回退按引擎解析默认模型
- OAuth：discovery 配置化（`OIDCDiscoveryURL` 生效）+ 换 token 超时客户端；E04/E13 回归
- 前端：E16/E22/E23/E24 核对 + 静态回归守卫 `scripts/frontend-e-check.mjs` 接入 CI
- 小说：角色注入剥离剧照 base64（`types.PromptView`）；E02/E03 回归
- 验证：go build/vet/test + tsc + vite build + 前端守卫全绿（`scripts/ci.ps1` CI OK）

## v2.0.1「三脑底座」(2026-08-07)

> 在 v1.21.0 基础上迭代：模型路由、三脑记忆、主脑可选编排、基线加固。

- 基线：frontend/package.json 纳入版本控制；scripts/ci.ps1 闸门；E01-E24 评估集 + 首批回归
- 模型：routeModel 降级链（功能绑定→全局→首个可用）；novel/whisper 调用收敛；模型中心"当前生效"
- 记忆：BrainStore 三脑统一访问 + brain_links 跨脑关联；记忆中枢三脑检索
- 编排：ModuleRegistry + RunModule + MainBrainChat（可选入口，不经由模块直达路径）
- 互联：方案生成自动注入跨脑记忆；3.0 模块协议文档预留
- 验证：go build/vet/test 全绿 + tsc + vite build + wails build（见 releases/v2.0.1.md）

## v1.21.0「UI 工作台」(2026-08-05)

> 方案编写板块重构为三栏工作台：左侧项目/方案树（检查分数徽章）＋ 中间文档工作区 ＋ 右侧 AI 上下文面板（招标要点/检查摘要/单人复核清单）；去除 Office 页 emoji 标签与 whisper-theme 依赖。

- 后端：Proposal 增加 CheckSummary（failed/warn/total）与 ReviewChecklist（复核项 Done），SchemaV7 落库；CheckAll 自动写入检查摘要并补默认复核清单（废标逐条/工期一致/评分覆盖/暗标合规/规范引用/签字盖章）；toMap/fromMap 透传
- 前端三栏：左栏项目与方案树（点击过滤/选中，方案卡显示模板、检查分数红/橙徽章）；中栏保留流程步骤条 + 6 个 Tab；右栏 AI 上下文 Drawer（招标要点、检查摘要、复核清单勾选保存、来源定位提示）
- 视觉：移除 Office 页全部 emoji 标签与 `whisper-theme.css` 引用，统一走主题令牌
- 验证：新增 1 测试（摘要与清单持久化）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.20.0「工作流编排」(2026-08-05)

> 方案编写板块引入四阶段流水线（招标解读→投标生成→投标检查→排版导出）与阶段闸门；一键流水线；办公 agent（Hephaestus）新增方案读写/导出工具。

- 阶段模型：Proposal.Stage（parse/generate/check/format，SchemaV6 proposals.stage 列）；解析成功→parse、大纲→generate、全面检查→check、导出→format 自动推进
- 阶段闸门：有招标文件但未解析时，生成大纲/章节/批量生成返回明确提示（空白方案不拦截）
- 一键流水线：RunPipeline 按序执行 解析→大纲→批量生成→全面检查，进度事件 proposal-pipeline-progress
- 办公 agent 打通：proposal 全局服务单例（测试可注入）+ 内置工具 proposal_list / proposal_write / proposal_export，Hephaestus 可直接读写方案需求/章节并导出
- 规范索引抽离共享包 specdata（builtin 工具与 proposal 模块共用，解除导入环）
- 前端：顶部 5 步步骤条（解析/大纲/编制/检查/导出）+ 一键流水线按钮 + 下一步引导（按阶段跳 Tab）
- 验证：新增 6 测试（闸门/阶段推进/流水线/办公工具）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.19.0「排版导出」(2026-08-05)

> 方案编写板块 docx 导出升级为可配置排版引擎：封面/目录/标题层级/页眉页脚页码/Markdown 表格与代码块；按章节导出；暗标规则库（内置土壤修复通用规则，导出自动清理）。

- docx 排版引擎：A4 页边距、页眉（方案标题）、页脚页码（PAGE 字段）、封面页、静态目录、章节三级标题（22/16/14 磅）、正文 12 磅宋体；Markdown 块解析（段落/标题/列表/表格→docx 表格/代码块等宽字体）
- 按章节导出：ExportSectionDocx 单章（含子章节）渲染为独立 docx；按章节 MD 已有
- 暗标规则库：office.db SchemaV5 dark_rules 表；内置“土壤修复通用暗标规则”（无加粗/斜体/下划线/彩色/emoji/特殊符号/压缩空行）；规则可增删改；导出选项选择规则后自动清理内容且标题不加粗
- 绑定：ProposalExportDocxWithOptions（封面/目录/暗标规则）、ProposalExportSectionDocx、ProposalDarkRulesList/Save/Delete
- 前端：导出设置 Modal（封面/目录开关 + 暗标规则选择）、单章导出（章节下拉）、暗标规则库管理 Modal（列表/编辑/选项 Checkbox/删除）
- 验证：新增 8 测试（渲染封面目录表格、单章导出隔离、暗标 seed/清理/CRUD、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.18.0「校验引擎」(2026-08-05)

> 方案编写板块检查能力升级：可插拔 CheckRule 引擎 + 结构化规则（废标响应/数据一致性/跨章重复/暗标格式/规范引用）+ AI 语义覆盖规则，统一检查报告前端逐项处理。

- 校验引擎框架：CheckRule 接口 + ruleFunc 适配器 + RunChecks 运行器（规则错误转 error item）
- 结构化规则（确定性，无需 LLM）：废标条款响应（未明确回应 warn）、项目事实一致性（未体现 warn/工期冲突 fail/暗标单位名 fail）、跨章重复（20 字 n-gram 交并比，>50% fail / >30% warn）、暗标格式（加粗/删除线/emoji）、规范引用（无引用 warn/编号不在知识库 warn）
- AI 语义覆盖规则：对照招标评分标准 full/partial/none 检查（LLM）
- CheckAll 聚合全部规则输出统一检查报告；既有 CheckCoverage 内部复用并保持绑定兼容
- 绑定 ProposalCheckAll；前端导出 Tab「全面检查」报告面板（规则/状态着色/证据/章节定位跳转）
- 验证：新增 15+ 测试（框架/规则/聚合/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

### v1.18.0 补充「转换诊断与扫描件 OCR」(2026-08-05)

> 修复桌面实测问题：招标解析失败根因多为第一步转换未成功且不可见。新增扫描件 OCR、转换结果阅览、AI 工作台。

- 根因修复：convertPdfToMD 无页面文字时不再返回“仅页眉”的伪成功（此前扫描件被误判已转换）→ 返回明确错误并触发 OCR；ParseBidFile 全文件转换失败时错误信息包含每个文件名与原因
- 扫描件 OCR：OCRProvider 可插拔接口 + Python 管线（PyMuPDF 渲染 300dpi 页面 → rapidocr_onnxruntime 识别，中文支持）；自动检测，无 OCR 引擎时给出安装指引；FileDoc 增加 error/ocrStatus 字段，OCR 结果标记“OCR 转换”
- 转换结果阅览：文件列表状态细化（已转换/OCR 转换/转换失败+原因 tooltip/待转换）+「查看」抽屉预览转换后 Markdown 或失败原因
- AI 工作台：转换与 AI 分析全过程事件（proposal-ai-progress：start/parse-file/parse-request/parse-reply/done/error），前端阶段动画（Spin+阶段文案）与右侧日志面板（自动滚动）；AI 分析按钮在“任一文件转换成功”时可用（不再被失败文件卡死）
- 验证：新增 6 测试（OCR 调用/OCR 不可用报错/全文件失败明确错误/进度事件）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.17.0「记忆中枢知识资产」(2026-08-05)

> 方案编写板块知识资产统一集成 gaea 记忆中枢：规范条文/素材/历史方案全部入库 Hephaestus.db knowledge 表，spec_query 改读知识库，方案可归档回写并提供同类型参考；模板库落库。

- 规范知识入库：内置土壤修复规范索引（GB 36600/15618、HJ 25.x、HJ 682、HJ 1185 等 15+ 条文）与「土壤修复常用技术」幂等写入记忆中枢（Category=规范标准/经验总结），SearchSpecs 按查询令牌重排（标题 3 分/正文 1 分）
- 素材库：业绩/人员/设备/常用段落以 Category=素材库 入库（AddAsset/ListAssets/SearchAssets/RemoveAsset），名称纳秒级唯一
- 历史方案：ArchiveProposal 将装配全文归档（Category=设计方案 + tag legacy-proposal），SearchLegacyProposals 检索；SectionContext 自动注入同类型历史方案参考摘要（≤600 字，注明不得抄袭）
- spec_query 改读记忆中枢：优先检索知识库规范条文（Top 5），知识库不可用时回退内置索引
- 模板库落库：DefaultTemplates 幂等 seed 到 office.db templates 表，ListTemplates 优先读库（失败回退默认）
- 前端：导出 Tab「归档到记忆中枢」按钮；右上角「素材库」弹窗（搜索/新增/删除）
- 验证：新增 15+ 测试（规范入库/检索重排、素材 CRUD、归档与参考注入、模板 seed、spec_query 知识库优先、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.16.0「大纲与撰写引擎」(2026-08-05)

> 方案编写板块长篇生成落地：目录策略 + 字数预算分解（以招标文件要求为准）+ 大纲重排/目录导入 + 项目事实基线 + 统一章节上下文 + 批量生成队列（断点续写/合并装配）+ 工艺流程图/组织架构图预设。

- 大纲策略与字数预算：三种目录策略（严格按评标办法/严格按格式要求/参考两者）；解析层提取招标 totalWords，预算按招标要求分配（未要求兜底 10 万、用户可改），递归分配到章→节→小节（叶子合计严格等于总数）
- 大纲重排与目录导入：同级上移/下移自动重编号；Markdown 标题（#/##/###）解析导入替换大纲
- 项目事实基线：office.db SchemaV3 project_facts（工期/业主/修复目标/人员等跨方案共享），SchemaV4 sections 增加 word_target/words 列
- 章节上下文 v2：统一 SectionContext（大纲/评分/废标/格式/暗标/项目事实/字数目标/前章摘要/后节锚点），单章生成与流式生成共用
- 批量生成与合并装配：RunBatch 按叶子单元顺序生成、跳过已完成（断点续写）、失败继续、进度回调（proposal-batch-progress）；Assemble 自动编号（第N章/N.M/N.M.K）并用于导出
- 前端：大纲 Tab 策略/总字数/导入目录/上移下移/字数目标；文本编制 Tab 批量生成/停止/进度条；方案列表右上角项目事实编辑；图表 Tab 工艺流程图/组织架构图/横道图
- 验证：新增 20+ 测试（预算分配/策略提示词/重排/导入/事实往返/上下文注入/批量/装配）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.15.0「招标解析管线」(2026-08-05)

> 方案编写板块招标解析升级：结构化字段提取（概况/工期/资质/评分/废标/格式/暗标）+ 原文来源定位 + parse_results 落库 + 前端结构化卡片与原文抽屉。OCR 仅定义可插拔接口，文字型 PDF 优先。

- 文档分页文本提取（doctext.go）：PDF 按页返回真实页码，DOCX/TXT 单页（Page=0），供来源定位使用
- office.db SchemaV2：新增 parse_results 表（字段/页码/Markdown 偏移/摘录/置信度），Store 提供 SaveParseResults/ListParseResults；AddFile 返回文件 ID
- 来源定位器（locate.go）：AI 摘录 quote 先精确匹配，失败后做忽略空白匹配，映射回原文偏移与页码
- 招标解析管线 v2（parse.go）：逐文件/分块 LLM 提取字段+原文摘录 → 后端确定性定位 → 结果写入 parse_results 与 BidSummary（qualification/format/darkRules/redLineItems/parseStatus，旧字段全兼容）
- 绑定序列化：btm/bsf 改为 JSON 往返，新字段自动透传前端
- 前端：招标解析 Tab 由 JSON 文本框改为结构化字段卡（可编辑保存）+ 来源 chip + 原文预览抽屉（滚动高亮摘录）
- 验证：新增 20+ 测试（分页提取/定位器/解析结果 CRUD/解析管线/序列化往返）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.14.0「方案数据底座」(2026-08-05)

> 方案编写板块数据底座重建：JSON 文件存储迁移 SQLite（office.db），引入项目（标段）层级、版本快照与旧数据无损迁移；方案列表按项目分组。

- office.db 网关（internal/office/db）：SchemaV1 六表（projects/proposals/sections/files/versions/templates）+ schema_meta 迁移链，纯 Go SQLite 驱动，与主脑库同模式
- proposal.Store 迁移 SQLite：项目 CRUD、方案/章节树持久化（含子章节递归归一化）、版本快照（每次更新 +1）、级联删除、附件登记
- Service 接线：启动自动建库 + 确保「未归档项目」+ 旧 JSON 无损迁移（幂等、只读不删原文件）；导出/上传目录迁至 office/exports、office/files；应用退出关闭 office.db
- 后端绑定：新增 ProposalProjectList/Create/Delete，ProposalCreate 支持 projectId；方案/章节数据带 projectId/version/sources
- 前端项目化：方案列表按项目分组（含未分组）、新建方案可选已有项目或新建项目
- 验证：新增 30+ 测试（网关/Store CRUD/树形持久化/版本/迁移/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.13.0「记忆检索升级」(2026-08-04)

> 轻语记忆检索体系升级：检索双轨收敛为单轨（buildTierBBlock）+ FTS 全文检索接线（触发词之外的摘要词召回）+ 语音链路测试补齐（voice/tts）。
> tag v1.13.0

- 记忆检索双轨收敛为单轨：删除零调用孤岛 PrepareTurnContext → MemoryRetriever.Retrieve（重复实现整套检索但从未被消费），统一到 orchestrator.buildTierBBlock 单一主流程；精简 memory_retrieve/types，删除 RelevanceHint/RetrievalResult 类型与 7 个 TestRetrieve_* 死路径测试（净删 353 行）
- FTS5 全文索引修复（此前建而不用）：V2 外部内容表列名与主表不匹配（fact_id vs id）导致 rebuild 必失败 → V11 迁移独立表；rebuild 改为显式全量同步（修复 MaxOpenConns(1) 下 rows 未关时 Exec 死锁）；SearchFactIDsFTS 修复 MATCH 成功但空结果不降级 bug
- 中文全文检索：LIKE 降级升级为 2-gram 多模式（整句 + 相邻两字）——用户说「咖啡」能命中摘要「她喜欢喝美式咖啡」，解决触发词之外摘要词无法召回的问题
- FTS 全文检索接入 TierB 记忆上下文：Orchestrator 新增 FTSSearch 回调（app 层注入 repos 实现，避免循环依赖）；buildTierBBlock 把 FTS 命中事实补入候选（×1.3 加权）；persist 写回后自动重建索引（RebuildFactsFTS/RebuildEpisodesFTS 从零调用变为活跃）
- 语音链路测试补齐：voice 包 0% → 54.2%（31 测试，情绪→TTS 映射/配置校验/VAD 状态机/打断检测，核心状态机 100% 覆盖）；tts 包 0% → 20.9%（26 测试，分句/SSML 转义/RFC6455 握手向量/引擎回退链）
- 验证：新增 60+ 测试（FTS 重建/中文降级/2-gram 模式/引擎回退/语音状态机）；go vet clean + go build ./... 全过 + go test ./... 60 包全绿 + frontend tsc 0 错误

## v1.12.0「轻语记忆贯通」(2026-08-03)

> 轻语记忆系统与 hermes.db 全链路贯通（事实/情节/知识图谱三表持久化）+ TierB 记忆上下文补全（情节/触发词/关联扩散/记忆回声）+ 记忆中枢展示（情节 Tab + 三元组入图）。
> tag v1.12.0，构建 37,743,616 字节。

- 轻语记忆贯通 hermes.db（右脑落地的关键缺口）：Orchestrator 新增 EpisodicStore 运行时实例（原 handler 硬编码 nil，情节从未生成）；restoreWhisperState 从 hermes.db 恢复事实库/情节库/图谱（重启不丢记忆）；persistWhisperState 写回——事实合并写回（本会话以内存为准含退役态，保留其他会话事实），情节/图谱全量写回
- 修复 restoreWhisperState 早期返回缺陷：companion_state/chat_history 无行时提前 return，阻断记忆恢复（首次使用或清空历史后记忆永不加载）
- 修复数据竞争：KnowledgeGraph/EpisodicStore 无锁——extractTriples/情节生成在异步 goroutine 写 + PreLLMTurn 主流程读，Go map 并发写读会 fatal / slice append 实测丢数据 → 加 sync.RWMutex
- 修复 KnowledgeGraph.Query 评分 bug：原遍历全局 entityIdx 给所有三元组加分（图谱越大误命中越多）→ 改为匹配三元组自身 subject/object；单字实体可命中
- TierB 记忆上下文补全：情节检索（EpisodicStore.Search → 【相关记忆片段】）+ 触发词命中事实 boost 1.5x + 关联扩散（Top5 事实经 AssocIndex → 【关联记忆】）
- 记忆回声接线：buildTierBBlock 用检索事实 EmotionalContext 聚合记忆回声（Aff/Sec/Aro/Dom），PreLLMTurn 叠加到状态情绪（ApplyMemoryEcho/ComputeMemoryEchoFacts 从零调用孤岛变为生效，clamp ±100）
- 记忆中枢展示：轻语库新增「事实/情节」Tab（情节时间线流：情绪 emoji 角标 + 强度渐变条 + 关键词 chips + 轮次范围，按 AI 伴侣记忆库调研）；记忆图谱新增轻语三元组（实体节点 t: 复用 whisper 色 + predicate 关系边，共享实体去重）
- FactStore 新增 Restore（保留原 ID/退役态）/ListAll；KnowledgeGraph 新增 Restore/ListAll；影响面：记忆中枢轻语库/总览/归档/图谱首次读到真实数据
- 验证：新增 12+ 测试（FactStore Restore/ListAll、KG 并发/Restore/Query、EpisodicStore 并发、TierB 情节注入/记忆回声/关联扩散、集成往返、图谱三元组）；go test ./... 全绿 + go vet clean + go build 全过 + tsc 0 错误 + vite build 成功

## v1.11.0「界面体验深化 · 全站重设计」(2026-08-02)

> 设置中心外观细化升华（实时预览/三态显示/字体/密度/动效/强调色）+ 聊天 Markdown 消息体验 + 轻语面板 UI 重设计（角色状态头/气泡/情绪回复）+ 虚拟助手面板与角色卡详情重设计 + 轻语测试深化（21.8%）+ P3 archiveExporter 记忆归档导出。
> tag v1.11.0，构建 37,676,544 字节。

- 轻语测试深化：新增 33 个测试覆盖 memory_ingest 管线（LLM 抽取/自动退役/三元组/情节生成/隐私透传）、association_cold_start（文本重叠批量建边/孤儿链接/边去重）、paced_stream（气泡分隔/流排空）、memory 检索路径（触发词 boost/隐私过滤/budget 截断/情节检索/关联扩散/记忆回声）；覆盖率 16.8% → 21.8%
- 修复 paced_stream 真实 bug：流结束发完末段不 finishBubble → MarkDone 等待循环死锁（中文无断点文本必现）；FirstDisplayUnitLen 返回 rune 索引但调用方按字节切片 → 中文句号处切乱码（现转字节偏移）
- P3 archiveExporter：记忆归档导出（对齐 ackem archiveExporter）——README 索引（事实/核心/情节统计 + 领域分布表）+ 每个领域/子类一个 Markdown 文件；退役事实不入档；whisper.WriteArchive 写盘；绑定 GaeaWhisperExportArchive（hermes.db 数据源）+ GaeaPickDirectory 目录选择；记忆中枢轻语库「导出归档」按钮
- 模型中心：LLM 主卡片补三态状态徽章（● 运行中 / ○ 就绪 / ○ 已停止），对齐行业「模型状态三态」基准
- 设置中心：全局搜索（tab 关键词索引 + 实时过滤 + 自动切换到匹配分组 + 匹配计数）+ 即时生效统一徽章（SettingsSection instant prop，5 面板统一）
- 外观设置细化升华：外观实时预览区（主题+模式组合微缩界面，hover 主题卡即时联动「👆 预览中」）+ 主题色系大预览卡（氛围渐变 + 霓虹光晕 + 选中发光对勾）+ 显示模式三卡（暗色/亮色/跟随系统，matchMedia 实时派生，darkMode 保持 boolean 兼容现有消费者零改动）
- 外观设置扩展非颜色维度：字体设置（5 预置字体 + 字号 12-20 带预览）、界面密度（标准/紧凑，ConfigProvider token + .ui-compact CSS）、动效强度（完整/减弱，.ui-reduced-motion 全局禁用动画对齐 macOS 减弱动态）、强调色自定义（取色器覆盖 --gaea-glow/primary 令牌链，留空跟随主题）
- 聊天板块消息体验升级：Markdown 渲染（ChatMarkdown：代码块深色底+复制按钮/表格/列表，完成态渲染）+ 消息分组（同角色连续紧凑 + 组首头像）+ hover 操作栏（复制/朗读/重新生成显隐）+ 重新生成 + 建议卡点击即发送 + 头部元信息（你/gaea AI）+ 错误态红色气泡 + 玻璃内高光
- 轻语面板 UI 重设计：角色状态头（头像 + 关系阶段徽章 + L2 情绪 emoji 徽章 + 信任霓虹进度条 + 对话轮数）+ 消息气泡化（AI 琥珀玻璃气泡 / 用户粉紫渐变）+ 等待期 typing dots 修复（原空白光标）+ 快捷情绪回复 chips + 虚拟助手管理中心按钮卡片化（暖色玻璃入口卡）
- 虚拟助手面板 + 角色卡详情重设计：列表卡视觉区 TisorRadar → CompanionAvatar 粒子光球 + 性格标签 chips + 迷你五维条；详情卡大视觉区粒子球 + 五维区「左 TisorRadar 大雷达 + 右条形列表」并排
- 验证：go vet clean + go build ./... 全过 + go test whisper/app 全绿 + frontend tsc 0 错误（全流程每步验证）

## v1.10.0「科幻记忆中枢 · 架构归拢」(2026-08-02)

> 记忆中枢科幻感首页（中央 3D 图谱 + 霓虹玻璃卡片）+ 3D 图谱白屏修复（three-forcegraph → 3d-force-graph）+ AI 控制台归小说专用 + 小说专属代码归拢 components/novel/。
> tag v1.10.0，构建 37,617,152 字节。

- 记忆中枢科幻首页：中央 3D 图谱 + 四周霓虹玻璃模块卡片（hub.css 玻璃拟态 + 扫描线 + stagger），点击切库面板
- 3D 图谱白屏修复：误装底层库 three-forcegraph（class 需 new）→ 换 3d-force-graph（Kapsule 可调用）；图谱渲染时序修复（数据到达即构图）
- AI 控制台：默认关闭 + 记忆中枢隐藏（面板/按钮双层排除）+ 从 MainLayout 抽出为 components/novel/AIConsole.tsx 仅小说页挂载
- 小说代码归拢：16 组件 + hooks/api 全部移入 components/novel/（git 识别 18 rename），MainLayout 删 ~380 行小说控制台代码
- 界面配色跟随系统主题：hub.css 硬编码深色 → gaea 令牌（--bg/--fg/--accent），图谱背景/label 走令牌
- 验证：go test 57 包 0 失败（后端零改动）+ tsc -b + vite + wails build 全绿

「记忆中枢」(2026-08-02)

> 记忆体系三脑架构落地（命名 Hephaestus/Hermes + 主脑 Hephaestus.db + 左脑办公 SQLite + 右脑 hermes.db + 调度路由 + 知识库 RAG）+ 记忆中枢板块（七库统一管理）+ 3D 记忆图谱 + 成本库。
> tag v1.9.0，构建 36,552,704 字节。

- 命名体系：办公 agent → Hephaestus（火神），轻语 agent → Hermes（信使），gaea 之子女；AIgaea 产品类型统一
- 三脑架构：新建 Hephaestus.db（facts/profile/knowledge 三表 + 迁移链）；memory.Store 后端抽象（文件/SQLite 双后端，调用方零改动）
- 左脑接通：办公记忆 Markdown → Hephaestus.db 幂等迁移 + memory_get 工具 + boot/controller 默认 SQLite
- 右脑更名：whisper.db → hermes.db（首次打开自动迁移，保留备份）
- 调度路由：remember type=user → 主脑画像（profile 跨 agent 共享）+ DetectConflicts 冲突检测
- 知识库 RAG：迁入 Hephaestus.db + 共享向量层 internal/gaea/search（TF-IDF + 中文 bigram 余弦）混合排序
- 记忆中枢板块：知识库入口升级为多库面板（知识/成本/画像/办公三 tab/轻语只读/图谱），12+ 新绑定
- 3D 记忆图谱：three-forcegraph（type 着色/按库过滤/hover/点击详情），后端预计算 nodes+links
- 成本库：cost_entries 表（schema V2）+ cost 包 + CostLibrary（基础版）
- 画像冲突一键裁决（以画像为准 / 以 facts 为准）+ 轻语详情弹窗
- 验证：go test 全量 57 包 0 失败 + tsc + vite + wails build 全绿

「单模型架构 · 知识库板块」(2026-08-02)

> 办公板块删除双模型架构（Hermes/Hephaestus → 单模型）+ 知识库整合为独立板块（统一服务层 + 全文检索）+ AI 聊天双会话面板修复。
> tag v1.8.0，构建 36,084,224 字节。

- 办公板块：删除 Hermes/Hephaestus 双模型 agent（hermes.go 645 行 + 测试），runner 直接用 executor 单 Agent；
  config 删 planner_model/temperature/effort；前端删 Planner 配置/统计列/RunStatus 简化
- 单模型工作流梳理：系统提示词分层（DefaultSystemPrompt=领域知识 / SingleModelPrompt=执行纪律），
  boot 拼接保执行纪律不丢；删除 PlanCard 计划确认死链路（AskQuestion.Plan 字段全链清理）
- 知识库独立板块：knowledge.Service 进程级单例（工具/UI 统一走 ~/.gaea/knowledge），
  新增 GaeaKnowledgeSearch 全文检索（含正文）；KnowledgePage 板块页 + 导航五处注册；
  与记忆系统明确区分（显式知识 vs 隐式事实）
- AI 聊天：删除 ChatTopicSidebar 重复渲染（双会话面板修复）
- 验证：go test 全量 0 失败 + tsc + vite + wails build 全绿

## v1.7.0「设置中心重构 · 模型统一」(2026-08-02)

> 设置中心全面重构（小说/方案/办公/轻语/更新信息）+ 模型绑定统一左下角卡片 + 代码审查修复 9 项并发与绑定 bug。
> tag v1.7.0，构建 36,140,032 字节。

- 审查修复：轻语引擎 per-call 覆盖（删全局切换竞态）、小说 9 处绑定模型生效、ASR 校验、config.Save 加锁、
  manager 副本防 race、轮询进程合并、删 VoiceSetChatTarget 死绑定、finalTranscript 修复
- ComfyUI：findPython 加 standalone-env 兜底（ROCm）+ windows-standalone-build 标志 + 工作流结构测试
- 绑定模型卡：5 板块统一左下角浮动卡片（三态 + 启停），聊天页补渲染；预警只统计本地模型
- 模型中心：功能模型绑定独立标签页（2 列卡片网格）
- 设置中心 6 标签：小说（目录 C:\AI\xiaoshuo）/ 方案（新）/ 办公（完整可编辑）/ 轻语（设置合并）/
  系统（更新信息 + 删迁移）/ 语音
- 轻语界面移除设置弹窗，左栏 240 防卡片遮挡；删空态多余标题
- 验证：go test 全量 + tsc + vite + wails build 全绿

## v1.6.4「设置瘦身」(2026-08-02)

> 设置页删除与模型中心重复的「模型引擎」配置 tab，引擎/模型管理统一收敛到模型中心。
> tag v1.6.4，构建 36,106,240 字节。

- 移除设置页 EnginePanel tab（import/图标/副标题文案同步清理）
- 设置页保留：外观 / 工作空间 / 语音 / 绘梦 / 办公 / 系统
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.3「剧照模型可选」(2026-08-02)

> 补丁：小说/轻语角色剧照生成可选手模型（含 ComfyUI 本地 krea2/z-image-turbo/flux）。
> tag v1.6.3，构建 36,110,848 字节。

- 后端 GetImageBackendConfig：availableModels 恒含 ComfyUI 本地模型（无论全局后端），
  小说角色剧照弹窗即可选本地模型
- 轻语角色详情：生成剧照按钮旁加「出图模型」Select（自动加载可用模型，默认当前全局模型），
  handleGeneratePortrait 用所选模型
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.2「方案模型条 + 底栏常驻」(2026-08-02)

> 补丁：方案编写模型条不可见修复 + 底栏常驻（无项目时也显示模型监控与资源）。
> tag v1.6.2，构建 36,109,824 字节。

- OfficePage 顶部插入 FeatureModelBar（feature="office"）
- MainLayout 底栏 Footer 去掉 projectOpen 条件 → 常驻显示已启用模型 + CPU/内存/GPU
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.1「小说统一模型」(2026-08-02)

> 补丁：小说板块章节/角色/世界观 agent 统一接入 func_novel，整部小说用一个 LLM 模型（v1.6.0 已知限制 1 修复）。
> tag v1.6.1，构建 36,108,288 字节。

- worldview / character / chapter / analysis / outline 5 个 agent 全部加 featureModel + chat（带 func_novel 引擎覆盖），
  替换全部 `ChatSimpleStream(a.cfg.Model)` 调用 → `chat(ctx, ...)` + `ChatSimpleStreamWithOptions(EngineID: FuncNovelEngine)`
- 运行中切换小说模型即时生效（各 agent 动态读 cfg.FuncNovelEngine/Model）
- 验证：go test ./... 53 包全绿 + tsc -b + vite + wails build

## v1.6.0「语音交互 · 角色中心 · 功能模型」(2026-08-02)

> 语言交互全面落地：首页语音对话 + 核心 AI 助手 gaea（大地女神）+ 角色中心（助手即角色）+ 各功能板块独立模型绑定与资源监控。
> 23 提交（v1.5.0 后），tag v1.6.0，构建 36,108,288 字节。详见 releases/v1.6.0.md

### 首页语言交互（全新）
- 深空虚空首页：400px 大语言粒子球（连线网络+双环绕轨道+音量驱动）悬浮，8 透明玻璃卡片浮游，虚空微尘背景
- 「进入语音对话」本页直启麦克风：轻语 voiceManager 管道 → gaea 对话 → 识别/回复气泡 + TTS 朗读
- 布局响应式（粒子球随视口缩放）+ 语音卡片边框透明

### 核心 AI 助手 gaea = 大地女神盖亚
- 新增人格预设 gaea（首位全局默认）：大地之母，温厚沉稳包容滋养（五维 T85/I55/S20/O80/R50，goddess 标签）
- 启动确保 gaea 始终存在（修复旧数据缺失）；唯一「AI 助手」，其余标「角色」

### 角色中心（助手即角色）
- 管理中心改铺满主界面角色中心（角色卡网格 + 详情弹窗）；AI 角色剧照；小说↔轻语互传（参数统一）
- gaea 排第一 > 当前对话 > 启用 > 禁用

### 功能级模型绑定
- ai.Client per-call 引擎覆盖；config 10 键（chat/whisper/novel/office/gaea 引擎+模型）持久化
- 接入：聊天/轻语（含语音）/小说大纲；模型中心「功能模型绑定」UI
- 各窗口 FeatureModelBar（三态状态+一键启停）；底栏资源监控（CPU/内存/GPU+已启用模型）；超载弹窗警告

### 模型中心完善
- 恢复 ComfyUI 引擎（启停/状态/本机路径写死）；ComfyUI 图片模型（krea2/z-image-turbo）入墙
- 语音模型 STT/TTS 三段选择 + 持久化

### 已知限制（范围控制）
- 小说 agent 绑定仅大纲；办公 agent 绑定仅 UI；语音端到端依赖本地 herdsman 未 GUI 自动化实测

## v1.5.0「微信接通」(2026-08-02)

> 微信 ClawBot 通道全链路打通（微信发消息 → AI 自动回复），办公板块 v1.4.0 回归修复。
> 11 提交（v1.4.0 后），tag v1.5.0，构建 36,011,520 字节。
> 详见 releases/v1.5.0.md

### 微信 ClawBot 通道接通（核心）
- 认证修复：`Authorization: Bearer <token>` + `iLink-App-Id` + `iLink-App-ClientVersion`（数字编码 132099）
  + getUpdates 端点全小写 → 消除"会话过期"（errcode -14）
- 消息字段对齐**腾讯官方 openclaw-weixin**：`item_list[].type`（type=1 文本）+ client_id `{prefix}:{ts}-{hex}`
  （此前对齐社区 Rust SDK 误用 `item_type`，消息被静默丢弃 → 微信无回复的最终根因）
- 扫码绑定流程补全：need_verifycode 配对码二次查询（新绑定 WhisperWeixinQRStatusWithCode）、
  scaned_but_redirect/verify_code_blocked 处理；confirmed 返回完整 token + botId
- 会话过期状态透出：UI 显示"微信会话过期 · 需重新绑定"替代虚假"在线"
- 防脱敏 Token：前端含 `*` 用原值、后端含 `*` 拒绝
- 助手名字注入：BuildAckemCanonBlock 名字参数化 + Orchestrator.AssistantName，微信 AI 自称助手名
  （如"峨嵋"）而非默认"轻语"

### 办公板块修复（v1.4.0 回归）
- 右侧面板 Drawer 内联渲染（getContainer=false）+ 改为布局内 grid 列（340px）——不再跨页面残留/遮挡导航栏/凸出遮挡对话区
- 恢复对话区样式（修复 eb7d5e6 误删 chat-pane/transcript/markdown 样式族，输出框消失）
- 移除办公设置 + 明暗配色按钮（与系统重复，由主应用设置中心接管）

### 验证
- go test ./... 53 包全绿；tsc + vite build；wails build 产物 36,011,520 字节
- 微信全链路用户实测通过（收到消息 → AI 回复 → 发送成功）

## v1.4.0「架构统一」(2026-08-02)

> 后端四类重复收敛 + 前端两套 UI 体系统一。净删 6874 行（+555/-7429），10 提交，tag v1.4.0。
> 详见 releases/v1.4.0.md

### 后端架构重构（-526 行）
- HTTP client 统一：netclient 提升共享层，22 处裸 `http.Client` 收敛到 `NewSimpleClient`
- 工具函数收敛：office 死代码 + asr/whisper 手写字符串函数替换标准库
- 配置系统整合：删除 gaea config 零消费者死域（ComfyUI/Statusline/Notify/Serve/LSP）
- AI 通道唯一化：删除 provider/stream_client.go，通道=前端→agent→bridge→ai.Client→modelengine

### 前端两套 UI 体系统一（-6300+ 行）
- 死代码清理：老栈 29 废弃组件 + gaea 7 死代码 + 4 死 hook/util + highlight.js（-5800 行）
- 令牌层统一：gaea 颜色引用老栈 M3 令牌，删独立主题系统，暗亮双向联动
- 图标体系统一：66 个 lucide → @ant-design/icons，lucide 依赖移除
- 布局壳 antd 化：Layout/Drawer/Tooltip 用 antd，功能面板按分层原则保留自绘
- 死样式清理：.app/workspace-panel 样式族删除（-690 行）

### 验证
- go test 77 包全绿；tsc + vite build 通过；wails build 产物 gaea.exe

## v1.3.0「未来感 UI」(2026-08-02)

> 未来感 AI 多功能助手平台：深空星云 × 玻璃拟态 × 霓虹光效全链路统一 + 设置中心整合全部参数。
> 五批 UI 改造（+718/-268 前端），tag v1.3.0，构建 36.1MB。

### 设计系统
- 12 套主题新增 glow/glassBg/auroraBg 三令牌（App.tsx 注入 CSS 变量）
- 未来感 CSS 层：赛博网格 + 星点 + 漂浮光球背景、玻璃拟态、霓虹描边卡片、发光状态点、流光扫描线

### 框架与界面
- 顶栏玻璃化 + logo 光晕 + 菜单霓虹选中态；AI 中枢首页（真实模型状态 + 霓虹卡片墙）
- 模态框/弹层全局玻璃化；主题色块发光胶囊；页面切换过渡 + 霓虹加载态
- 聊天界面：玻璃气泡 + 渐变用户气泡 + 霓虹输入栏；侧栏玻璃化
- 绘梦面板玻璃化 + 画廊 hover 发光；底栏霓虹状态条

### 设置中心（整合全部设置参数）
- SettingsPage 重写为 7 分组 Tabs（外观/工作空间/模型引擎/语音/绘梦/办公/系统）
- 新增 settings/ 8 组件：主题发光选择器、图片保存目录、推理强度、引擎编辑、语音健康检测与阈值、绘梦后端、办公摘要、v4 迁移
- api/settings.ts 补 5 封装；设置页隐藏 AI 控制台

### 验证
- go test ./... 全绿（54 包）；go build / go vet 通过
- npm run build + wails build 产物 gaea.exe（36.1MB）
## v1.2.0「架构瘦身」(2026-08-01)

> 后端臃肿治理：绑定层按域拆分（197 方法迁移）+ 死代码清理 + 办公板块设置写路径接通
> + staticcheck U1000 全仓库清零 + 原子写实现去重。5 批提交，净删 ~600 行。

### 绑定层拆分（App 聚合 + 嵌入提升）
- App 结构体从 20+ 平铺字段收敛为 core + 4 域 State 嵌入引用（writing/media/whisper/office）
- 197 个方法按域归位；Go 嵌入提升保证 Wails 绑定不变，前端零改动
- 子服务持 app 反向指针协调跨域调用

### 办公板块设置写路径接通（20 个存根）
- 新增配置持久化（~/.config/gaea/config.toml 原子写）+ 改配置 → 保存 → 重建 controller
- Agent 参数 / 权限 / 沙箱 / 模型设置真实生效；Provider 增删改转发模型中心
- 回退 / 更新类改为明确错误语义，前端隐藏回退菜单

### 死代码清理（staticcheck U1000 全仓库清零）
- gaea 引擎：notify 包（2 文件）+ filteredSchemas + incompleteCanonicalTodos + runTurn
- whisper：candidate_collector.go 整文件（179 行孤岛）+ emotion_fusion 7 函数 + 3 常量
- ai：buildFluxWorkflow（已被 Z-Image/Krea 取代）

### 原子写去重
- 重建 fileutil.AtomicWrite，5 处手写重复实现收敛，净删 41 行

### 验证
- go build / go vet / 53 包测试全绿；staticcheck U1000 清零；前端 tsc 通过
- 新增测试：配置持久化往返 + 嵌入提升编译期断言

## v1.1.0「质量工程」(2026-08-01)
## v1.1.0「质量工程」(2026-08-01)

> 稳定性加固 + 架构瘦身 + 测试防线建立：21 处 goroutine recover、SSE 阻塞修复、
> 死代码清理 902 行、四个核心模块测试覆盖大幅提升（modelengine 97.1% / office 35.6% / ai 24% / whisper 16.8%）。

### 稳定性（3 批提交）
- 21 处 goroutine 无 recover 防护：桌面应用 panic 崩溃问题根治，最严重为
  controller.runGuarded turn 执行 panic 导致前端永久"生成中"，修复后复位状态 + Emit TurnDone
- SSE 流式发送 select+ctx.Done 保护：取消后 goroutine + HTTP 连接阻塞泄漏根治（hang 类问题根因）

### 架构整理（5 批提交，净删 902 行）
- gaea 模块：管理命令组 / 模糊编辑移植 / 宪法文件 / 7 处未使用字段别名
- 其他模块：剧照旧实现 / 世界观上下文 / 章节迁移等
- staticcheck U1000 84→25 处（剩余为 whisper 对齐 ackem + ComfyUI 迭代相关）

### 测试防线（5 批提交，+1900 行测试）
- modelengine 0%→97.1%、office/proposal 0%→35.6%、ai 7%→24%、whisper 13.1%→16.8%
- 修复 3 处测试暴露缺陷：GenerateSection 漏设 completed、UpdateSection/RemoveRawFile 静默失败

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`，完整说明见 `releases/v1.1.0.md`

## v1.0.0「品牌重塑 · 盖亚」(2026-08-01)

> wubigrok 正式更名为 **gaea**（盖亚，大地女神）——从「小说创作 Agent」升级为「多功能 AI 助手」。
> 全新品牌视觉：翡翠球体 + 破土嫩芽 + 灵感星芒，favicon / appicon 全量替换。

### 品牌重塑
- 全产品品牌替换：窗口标题「gaea · 多功能 AI 助手」、应用名/产物（gaea.exe）、UI 显示、文档、导出署名、图片下载前缀、日志文件
- Go module 重命名 `github.com/wubigork/wubigork` → `github.com/gaea/gaea`（233 个文件 import 同步）
- 新 logo 三件套：`frontend/public/favicon.svg` + `build/appicon.svg` + `build/appicon.png`（1024x1024）
- 版本号从 v5.x 重新起算为 **V1.0.0**（versioninfo.rc / wails.json / CHANGELOG）

### 数据兼容（老用户零丢失）
- 配置文件 `~/.gaea_config.json`（回退读取 `~/.wubigork_config.json`）
- 登录 token 回退读取 `.wubigork_token.json`（免重新登录）
- 项目标记目录 `.gaea/`（识别旧 `.wubigork/` 项目，v4 检测双向兼容）
- localStorage 键 `gaea_*`（回退读取旧键：聊天记录/人格/主题/绘梦模板保留）
- 内部 provider 注册名 `wubigrok` 保留（bridge provider 引擎兼容），UI 显示名全部为 gaea

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`

## v5.76.0「工程办公」(2026-07-31)

> gaeaW（土壤修复工程办公 AI 助手）完整移植：47 个工程工具 + Hermes/Hephaestus 双模型 agent + 6 个工程技能 + gaeaW 原生 UI，模型统一走 gaea 模型中心。

### 新板块：办公
- 后端移植 `internal/gaea/` 30 包（agent/tool/control/skill/command/plugin/knowledge/memory/boot/config），模型经 `provider/bridge` 接入模型中心（空模型动态跟随引擎切换）
- ai 包扩展工具调用支持（OpenAI 兼容 + SSE tool_calls 分片拼装，向后兼容）
- gaeaW 原生 UI 完整移植至 `frontend/src/gaea/`（App + 70 组件 + Tailwind v4），GaeaPage 渲染 gaeaW App
- 适配层：bridge.ts 90+ 方法名映射（Submit→GaeaSend 等），wubigrok 补齐 80+ Gaea* 绑定方法，事件格式精确对齐 WireEvent
- `.gaea/skills/` 6 个工程技能（场地调查/风险评估/修复设计/投标/数据报告）

### 发布
- 完整说明见 `releases/v5.76.0.md`

## v5.73.1「方案编写完善」(2026-07-31)

### 关键修复
- 嵌套章节（AI 大纲的 2/3 级）后端查找/更新全部改为递归展平，子章节可正常撰写/润色/改图/重命名
- 流式撰写内容真正落盘（原为副本指针，刷新即丢）
- 前端自动保存与图表插入支持子章节（updTree 递归更新）
- Word/MD 导出、覆盖/规范检查包含全部层级章节
- 浏览器上传招标文件改为 base64 落盘后转换（原路径为空必然失败）

### 新能力
- 大纲手工编辑：新增子章节/重命名/删除（含确认与编号重排），三个章节树均可用
- 流式撰写上下文补齐：需求、评分标准、废标条款、完整大纲、前一章节
- 失败时向前端发送 error 事件，不再卡死「生成中」

### 发布
- 完整说明见 `releases/v5.73.1.md`

## v5.20.0「精炼」(2025-07-26)

> 提示词全面重设计 + 死代码大清理 + 编译修复。净删 ~3000 行，prompts 23→15。

### 提示词重设计（4 个核心 prompt）
- **create-chapter**：硬编码字符串 → 模板化，「正在写这本书的作者」
- **chapter-generate**：「出版级」→「作者」，去 AI 味交由技能注入
- **plot-branch-browser**：「剧情策划人」→「story breaker」，固定 3 分支
- **worldview-agent**：「设定顾问」→「设定编辑」，代码块输出替换原文

### 死代码大清理（16 文件删除 + 后端精简）
- **删除 9 个废弃 prompt JSON**：brainstorm-ideas, chapter-review, outline-generate-detail, story-thread-chat, story-thread-generate, worldview-chat-section, worldview-check-consistency, worldview-generate-all, bootstrap-reference-summarize
- **删除 6 个前端组件**：AIAssistSheet, BrainstormModal, DialogueModal, StoryBibleModal, StoryBibleSteps, BeatToProse
- **删除 brainstorm_handler.go**（59 行）
- **Go handler 死代码清理**：chapter_handler (-178), copilot_handler (-237), create_chapter_handler (-82), outline_handler (-108), plot_branch_handler (-80), project_handler (-107), worldview_handler (-42)
- **核心模块精简**：chapter.go (-301), outline.go (-203), worldview.go (-215)
- **前端页面清理**：MainLayout (-60), CreatePage (-46)，其他页面移除死引用

### 编译修复
- chapter_handler.go 缺失函数闭合 `}` 修复
- copilot_handler.go 6 个未使用 import 清理
- chapter_handler.go 未使用 "strings" import 清理

## v5.17.0「重塑」(2025-07-26)

> 从 v5.7.1 分支重建。移除移动端代码，新增「创作」面板，章节节点树 + 分支系统。

### 移除
- 删除 `mobile/` 独立移动端应用
- 删除 `internal/mobile/` Go HTTP 服务
- 删除 MobileTabBar、MobileDrawer、useIsMobile 等全部移动端组件

### 创作面板（全新）
- **三栏布局**：左侧节点树 | 中间编辑器 | 右侧 AI 控制台（全局复用）
- **CreateChapter API**：一步到位，跳过对话和大纲阶段，直接生成正文
- **三分支构思**：AI 读取设定 + 前文摘要 → 生成 3 个剧情方向 → 用户选择 → 生成正文
- **节点树系统**：每章永久节点（ID、摘要、parent_id），支持分支/覆盖/删除/重生成
- **摘要注入**：每章生成后自动提取摘要存入节点树，前文注入改为节点摘要拼接

### AI 控制台增强
- 展开 REQ 查看完整 SYSTEM/USER prompt
- 展开 OK 查看 AI 完整响应（修复 response 事件缺失 content 字段）
- 固定宽度 380px + alignSelf: stretch

### 后端
- `internal/app/create_chapter_handler.go` — CreateChapter 一步生成+保存
- 正文末尾 `---CHAPTER_SUMMARY---` 标记自动分离摘要
- `GenerateOutlineWithDialogue` prompt 限制章节数，避免浪费 tokens

## v5.7.0「凝练」(2026-07-06)

> 7 轮组件化拆分迭代：42 文件变更，+3,244 / -1,946，(window as any) 和 @ts-ignore 全面清零

### ⚛️ 组件化重构（7 轮）

#### R3 — 去重 & 组件化
- **BranchSelectorPanel**: 提取 PlotBranchModal + NextChapterModal 共享分支选择 UI（187 行）
- **StoryBibleSteps**: StoryBibleModal 5 步子组件拆分，主组件 489→288 行
- **净效果**: +701/-448，两 Modal 各减少 ~90 行重复代码

#### R4 — 书架模块
- **提取 4 子组件**: WelcomePage / BrainstormModal / CreateNovelModal / ProjectCardItem
- **HomePage**: 640→280 行 (-56%)，Ctrl+N 快捷键 + 骨架屏
- **Utility**: 提取 formatRelativeTime + delay 到 utils/time.ts
- **类型化**: 新增 BrainstormIdea 接口，消除 any
- **移动端**: 统一 btnBase 样式，移除无效 CSS animation

#### R5 — 世界观模块
- **API 抽象层**: api/worldview.ts，消除 6 处 @ts-ignore
- **SectionNav**: 共享维度导航（桌面固定侧栏 / 移动端下拉面板），消除 ~90 行重复
- **ConsistencyReport / MapFullscreen**: 提取行内子组件
- **WorldviewPage**: 490→328 行 (-33%)

#### R6 — 角色模块
- **API 抽象层**: api/character.ts，消除 11 处 @ts-ignore
- **RelationshipModal / OrganizationEditModal / PortraitLightbox**: 提取行内 Modal
- **OrgField**: 移入 CharacterFormHelpers.tsx
- **renderCharEditor()**: 消除桌面/移动端 ~40 行重复
- **CharacterPage**: 489→396 行 (-19%)

#### R7 — 写作模块
- **Wails 类型导入**: 消除 7 处 (window as any)
- **ReviewResult**: 审稿组件（评分圆环 + 优势/不足/改进建议）
- **OutlinePanel**: 大纲面板独立组件（折叠/展开 + 卷/章树形渲染）
- **超长行拆分**: 最长行 620→314 字符，提取 createTabData / handleFinalize / handleNextChapterGenerate

#### R8 — 绘梦模块
- **API 抽象层**: api/image.ts，消除 12 处 (window as any)
- **ResultGallery 增强**: 添加 onDelete 支持
- **PromptBar / CustomTemplateModal**: 提取底部输入栏和模板编辑弹窗
- **ImageGenPage**: 665→521 行 (-22%)

#### R9 — 设置面板
- **API 抽象层**: api/settings.ts，消除 ~15 处 (window as any) + @ts-ignore
- **SettingsCard**: 统一卡片组件，消除 7 处重复 cardStyle
- **SettingField**: 统一「标签 + Input/Select」模式
- **handleToggleMobile / handleMigrate**: 提取命名函数
- **SettingsPage**: 475→356 行 (-25%)

### 🧹 全局清理
- **(window as any) 清零**: 前端所有页面全部替换为 wailsjs/go/app/App 类型导入
- **@ts-ignore 清零**: 通过 API 抽象层消除所有类型跳过
- **catch(_) 清零**: 全部替换为 catch(err) + console.error
- **mountedRef 移除**: React 18 自动批处理已解决

### 📦 构建
```
wails build -o build/bin/wubigork-v5.7.0.exe
```
- 新增 28 文件，修改 14 文件

## v5.6.0「移动端远程操控」(2026-07-03)

> 移动端完整功能实现：RPC 桥接、SSE 流式、静态文件服务、CORS、QR 码、IP 检测

### 📱 移动端功能
- **HTTP RPC 调度器** (`internal/mobile/rpc.go`) — 泛型反射调用 App 方法，黑名单过滤生命周期方法
- **SSE 流式推送** (`internal/mobile/sse.go`) — StreamHub 事件广播 + EventSource 频道
- **前端桥接层** (`frontend/src/api/bridge.ts`) — 非 Wails 环境自动创建 `window.go.app.App` HTTP 代理
- **Runtime 多填充** (`frontend/src/api/runtimePolyfill.ts`) — `EventsOn/Off/Emit` + SSE EventSource 多填充
- **静态文件服务** — embed.FS 生产模式 + `dist/` 前缀兼容 + SPA fallback
- **CORS 中间件** — 手机浏览器跨域访问支持
- **IP 检测** — 虚拟网卡过滤（Docker/WSL/Hyper-V 等），优先 WiFi 真实 IP
- **端口预检** — `net.Listen` 预检端口可用性，占用时立即报错

### 🐛 修复
- **移动端页面白屏** — `serveStaticOrSPA` 路径前导斜杠导致 embed.FS 拒绝读取
- **事件名不一致** — 前端 `prose-stream` → `beat-prose-stream`
- **render 崩溃** — HomePage `card.title` null 守卫
- **IP 错误** — UDP 拨号法回退到 Docker IP，接口名过滤法替代

### 📦 构建
```
wails build -o build/bin/wubigork-v5.6.0.exe
```
- 新增 4 文件，修改 12 文件

## v5.5.0「精炼」— 全量代码优化 + 工程质量加固 (2026-07-03)

> 两轮迭代：后端去重 + 前端巨型组件拆分集成 + 死代码清理 + 类型安全加固

### 🏗 后端重构（12项）
- **EstimateTokens 去重**: context+memory→util，修复中文检测用 unicode.Is
- **ExtractJSON 去重**: style→util
- **401 重试合并**: ai/client.go 三合一 → refreshAndRetry()
- **config.Save 重构**: 18-case switch→map 注册模式
- **marker 解析抽象**: ParseMarkedSections()，worldview/character 共用
- **for i:=1;;i++→ForEachChapter**: 8处死循环统一，缺失章节 continue 而非 break
- **chapter.Generate 拆分**: 200行→3函数(buildGenerateContext/streamAndRetry/postProcess)
- **ChatStream 拆分**: 请求构建+SSE 解析分离
- **mobile 死代码清理**: 删除 SPAHandler，修复 _ = path
- **app.go 类型安全**: comfyUICmd 删除，distFS interface{}→fs.FS
- **skill YAML 加固**: Windows \r\n 兼容，修正 scan() 注释
- **slog.Warn 统一**: 35+处→Agent.warnRead()

### ⚛️ 前端重构（11项）
- **ChapterPage**: 1170→227 行 (-80%)，集成 ChapterEditor/AIAssistSheet/useChapterStream
- **OutlinePage**: 1188→350 行 (-70%)，集成 OutlinePanel/ThreadPanel/DialogueModal/useOutlines
- **StoryBibleModal**: 488行→5步子组件+路由壳
- **PlotBranchModal+NextChapterModal**: 共享 usePlotBranch hook
- **4 新 hooks**: useChapterStream/useOutlines/useWailsEvent/usePlotBranch
- **errorHandler**: handleError+wrapAsync 统一错误处理
- **OutlineNode 类型冲突修复**: api/outlines.ts 删除重复定义
- **StepCreate.tsx**: onChange 死代码修复
- **HomePage Creating**: mount guard 防 unmount 后 setState
- **LorebookModal/SkillModal**: AbortController cleanup
- **TTSPlayer**: onStatusChange ref 冻结

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.5.0.exe -ldflags="-s -w" .
wails build -o build/bin/wubigork-v5.5.0.exe
```
- 新增 16 文件，修改 ~25 文件，删除 ~900 行
- 零新依赖

## v5.4.0「锻造」— 稳定性修复 + 本地生图增强 + 功能打磨 (2026-07-03)

> 修复 13 项 Bug，新增 ComfyUI 一键启停、系统状态监控、角色 Agent、图片管理增强

### 🐛 Bug 修复（13项）
- **xAI 默认模型名**: `"flux"` → `"grok-imagine-image-quality"`，修复云端生图失败
- **xAI API size 参数**: API 不再接受 `size`，请求中清除该字段
- **ComfyUI 编码崩溃**: Python 3.13 + GBK 控制台无法编码 emoji → 补丁 logger.py + prestartup_script.py
- **Login() 后 ComfyUI 丢失**: 重新登录后自动恢复后端配置
- **小说切换数据不同步**: 5 页面 (世界观/角色/大纲/章节/画布) 监听 projectPath 重新加载
- **Token 刷新无超时**: `RefreshAccessToken` 用 `http.DefaultClient` → 15s 超时，防止永久卡死
- **生成失败消息**: 返回具体原因而非笼统的「所有生成尝试均失败」
- **ComfyUI 执行错误**: 错误解析索引 msgArr[2]→msgArr[1]，正确提取异常信息
- **右侧面板隐藏**: 绘梦右侧栏改为始终显示（文件夹按钮在无历史时也可见）
- **CMD 弹窗**: `getCPUUsage` 调用 wmic 时隐藏窗口
- **防重复提交**: 绘梦生成按钮加 useRef 锁

### ✨ 新功能
- **ComfyUI 一键启停**: 绘梦顶部 🟢/⚫ 状态 + 启动/停止按钮，设置页配置安装路径
- **系统状态监控**: 绘梦右侧栏显示 CPU% + GPU 显存占用，3s 刷新
- **角色 Agent**: 角色页底部 ChatPanel，AI 对话自动创建/编辑角色
- **图片单张删除**: 画廊和历史缩略图支持逐张删除
- **图片文件夹快捷打开**: 右侧栏 📁小说图片 / 🖼生成图片 目录
- **世界观地图双击全屏**: AI 地图和 3D 势力图支持双击放大
- **图片自动保存**: 未配置专用目录时自动存到 `<小说>/images/`

### 🎨 UI/UX
- 导航 `项目` → `书架`，其余文案 `项目` → `小说`
- 大纲页卷结构面板拉长填满窗口
- 清理冗余文件回收 ~79MB

### 📦 构建
- `build/bin/wubigork-v5.4.0.exe` — 16MB

---
## v5.3.0「暗夜」— 全局配色重设计 (2026-07-03)

> 暗夜系列 6 套主题 — 手工调色，表面色与强调色分离，完全替换 M3 Tonal Palette

### 🎨 主题系统重做
- **6 套暗夜主题**: 暗夜青（默认）/ 暗夜紫 / 暗夜玫 / 暗夜金 / 暗夜苔 / 暗夜墨
- **表面色与强调色分离**: 不再从单一 seed 派生所有颜色，每套有独立的表面色温
- **手工调色**: 替换 M3 `generateTonalPalette` 自动生成，消除同色系单调问题
- **亮色模式**: 6 套暗夜主题各自对应一套亮色变体
- **零破坏**: 旧 localStorage 中的主题名自动回退到默认暗夜青

### 🔧 累积修复 (v5.2.1 → v5.3.0)
- Z-Image sampler 节点引用 14→13
- 大纲 prompt 矛盾约束（恰好5卷 vs 保留不修改）
- Lightbox z-index 1000→1050（剧照全屏被遮挡）
- AI 绘梦模板系统：类别下拉 + 自定义 + 去重风格

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.3.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **3 文件变更**: +70/-188 行
- **零新依赖**

---
## v5.2.0「凝形」— Bug修复+工程质量+性能优化 (2026-07-03)

> 基于 v5.1.0 审计，修复 24 项问题：10 Bug修复 + 8 工程质量 + 6 性能与体验

### 🐛 Bug 修复（10项）
- **Z-Image 尺寸参数**: width/height 正确映射到 Z-Image ratio（1:1/3:2/4:3/16:9/2:1），不再被忽略
- **Context cancel 泄漏**: `copilot.go` `_ = cancel` → `defer cancel()`，防止 goroutine 泄漏
- **io.ReadAll 错误忽略**: ComfyUI `queuePrompt`/`checkHistory` 两处错误现在正确返回
- **os.MkdirAll 错误忽略**: `export.go` 目录创建失败不再静默继续
- **NovelsDir 硬编码**: `D:\AI\xiaoshuo` → `~/wubigork-novels`，跨系统兼容
- **CancelGhost 空实现**: 真正取消进行中的 Ghost 补全 goroutine，前端收到 `cancelled` 事件
- **事件监听器清理**: `EventsOn('xai-output')` 添加 useEffect cleanup 防止内存泄漏
- **静默吞异常**: 全站 `catch (_) {}` → 带标签的 `console.error`
- **loadOutlines 重复**: ChapterPage/OutlinePage 重复实现 → 提取到 `api/outlines.ts`

### 🏗 工程质量（8项）
- **API 抽象层**: 新建 `frontend/src/api/outlines.ts`，封装后端调用
- **Wails 类型声明**: 新建 `frontend/src/types/wails.d.ts`，`AppAPI` 接口声明 100+ 方法，消除 `@ts-ignore`
- **TTS 配置持久化**: `config.go` 新增 `tts_port`/`tts_backend`/`tts_speed` 的 Load/Save
- **Save() 常量化**: 定义 19 个 `Key*` 常量替代硬编码字符串，防止拼写错误
- **SafeGo 工具**: `util/util.go` 新增 `SafeGo(fn)` — goroutine + panic recover + 日志
- **requirePM() 辅助**: `app.go` 新增读锁获取项目的方法，消除 handler 重复的 nil 检查
- **废弃文件清理**: 删除 `internal/ai/context.go`

### ⚡ 性能与体验
- **移动端 API 对接**: `mobile/handlers.go` `ProjectsProvider` 回调，替换硬编码 JSON 占位
- **统一日志**: 新建 `frontend/src/utils/logger.ts`，四级日志（debug/info/warn/error），生产环境可关闭
- **页面保活**: `MainLayout` `visitedPages` Set 机制，切 tab 不再销毁组件丢失状态
- **大纲五阶段固定卷**: `ContinueOutline(5)` 全量替换（起承转合终），简化 AI 生成流程
- **Z-Image NSFW LoRA**: 工作流新增 `LoraLoaderModelOnly` 节点 (strength 0.7)
- **世界地图图片**: `SaveWorldMapImage`/`GetWorldMapImage` Wails API，base64 存取

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.2.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **27 文件变更**: +1095/-346 行
- **新增 3 文件**: api/outlines.ts / types/wails.d.ts / utils/logger.ts
- **删除 1 文件**: internal/ai/context.go
- **零新依赖**

---
## v5.1.0「绘梦师」— Z-Image-Turbo 极速生图 + AI 绘梦工作台 + 角色剧照打通 (2026-07-03)

> 本地 Z-Image-Turbo 8 步生图（48s）、AI 绘梦双栏工作台重做、角色剧照与 AI 绘梦打通、角色详卡三段式布局

### ⚡ Z-Image-Turbo 本地生图集成
- **新模型支持**: ComfyUI-ZImagePowerNodes + GGUF Q5_K_M (~5.2GB) + Qwen3-4B Q4_K_M (~2.5GB) text encoder
- **8 步极速**: ZSamplerTurbo2Simple，48 秒出图（Flux 需 60-90 秒 20 步）
- **双模型切换**: Flux Dev / Z-Image-Turbo，前/后端完整支持，配置持久化
- **VRAM 优化**: UNet partial offload ~4GB 加载 + 1.4GB 卸载，8GB 显存刚好够用
- **v5.0.0 补全**: 下载脚本 + 工作流测试验证 + 节点参数自动探测

### 🎨 AI 绘梦工作台重做
- **双栏布局**: 左栏 320px 控制面板 / 右栏自适应结果区，桌面端专业工作台体验
- **负向 Prompt**: 可折叠输入区，Flux 工作流节点 8 注入
- **种子控制**: InputNumber + 🎲 随机，精准复现同一张图
- **批量生成**: 一次 1-4 张，循环提交 + 每张独立计时
- **20 个预设模板**: 创作/写实/风格/构图 4 大类，点击自动填入正/负向 prompt
- **全屏灯箱**: 键盘导航（← → Esc），显示种子/模型/尺寸/耗时，一键下载/重用参数
- **会话历史**: 底部横向缩略图画廊，点击回溯，支持清空
- **进度预估**: 按钮下方显示「预计 ~90s」/「上次 48s」，根据模型和数量动态计算
- **6 个新组件**: PromptPanel / GenControls / GenButton / ResultGallery / Lightbox / HistoryStrip
- **移动端适配**: 单栏控制面板 + 底部 Drawer 结果抽屉

### 👤 角色剧照与 AI 绘梦打通
- **剧照后端统一**: GeneratePortrait 改用 `cfg.ImageModel`（支持本地 Z-Image-Turbo/Flux），中文 prompt
- **AI 绘梦→角色剧照**: Lightbox 新增「设为剧照」下拉选择器，选角色自动保存到 `portraits/<id>.png` 并写回 `characters.json`
- **角色卡片缩略图**: 卡片列表圆形头像显示 `portrait_url`（有图时），无图时回退图标
- **SetCharacterPortrait API**: Go `character.Agent.SetPortrait()` + Wails 绑定

### 📋 角色详卡三段式重做
- **概览区**: 剧照 160×160 左侧主视觉 + 姓名/类型/状态/性格标签 + 操作按钮
- **Tab 分组**: 「📋 档案」（基本信息+性格+外貌+身材+动机+弧光+背景）/「🔗 关系」（组织+人物关系+出场章节）
- **剧照三态**: 有图悬浮重生成 / 生成中 Spin / 无图虚线框 CTA
- **删除按钮**: 移至右上角，减小误触

### 🔧 修复
- **config.go**: `comfyui_url` Save case 空白无赋值（严重bug）→ 正确赋值 `ComfyUIURL`
- **config.go**: `image_save_dir` Save case 错误写入了 `ComfyUIURL` → 修正为写入 `ImageSaveDir`
- **config.go**: `ImageModel` 配置字段新增 + Load/Save 支持

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.1.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **新增 7 文件**: 6 个前端组件 + 模板数据
- **修改 9 文件**: Go 后端 5 + 前端 4
- **零新 npm 依赖**

---

## v5.0.0「织梦者·移动」— 移动端远程操控 + M3 设计语言 (2026-07-02)

> 手机远程操控桌面端、Material Design 3 设计迁移、ComfyUI LoRA 生图链、20 commits

### 🎨 Material Design 3 全站迁移
- **M3 Tonal Palette**: 轻量内联实现，5 套主题从种子色自动生成 13 级色调调色板
- **Ant Design Token 驱动**: ConfigProvider 模拟 M3 视觉（surface/surfaceContainer/elevation/outline）
- **CSS 变量标准化**: `--md-sys-color-*` / `--md-sys-elevation-*` M3 命名体系
- **Layered Glass → M3 Surface**: `.glass-panel` → `.md-surface-container`，移除全站 backdrop-filter
- **M3 交互**: CSS-only ripple 涟漪效果、`.md-card` elevation hover lift、触控 44px 最小区域
- **25 变量兼容填充**: 旧 CSS 变量名 → M3 token 映射，零破坏迁移

### 📱 移动端远程操控
- **HTTP 服务**: Go `internal/mobile/` 包 — LAN IP 检测 + 二维码生成 + SPA fallback
- **响应式双导航**: 桌面端 Sidebar / 移动端 Bottom TabBar + Drawer + AppBar
- **Container Query 布局**: `useMediaQuery` hook + 3 级断点（compact/medium/expanded）
- **通用移动组件**: MobileSheet（底部滑出面板）、LongPressable（长按 500ms + 震动反馈）
- **全页面适配**: 9 个页面 + 5 个组件 — 3D 关系图→2D SVG 降级、Canvas→Grid、编辑器→全屏+Sheet
- **设置页**: Switch 开关 + 二维码面板，手机扫码即连

### 🎨 ComfyUI 集成增强
- **内嵌 Web UI**: ImageGenPage 一键切换 iframe 控制台，无需另开浏览器
- **LoRA 链**: Flux.1 Dev → Realism (0.8) → NSFW (0.6)，LoraLoaderModelOnly 级联
- **URL 持久化**: `GetConfig('comfyui_url')` 自动加载

### 🔧 工程质量
- **零新依赖**: 前端无新 npm 包，Go 纯 `net/http` 标准库
- **Wails 绑定**: 3 个新方法 `StartMobileServer` / `StopMobileServer` / `GetMobileServerStatus`
- **20 commits**: 每个 Task 独立审查（Spec ✅ + Quality Approved），2 个 fix round
- **前后端双编译**: `tsc -b && vite build` + `go build` 零错误

### 📦 构建
```bash
wails build                              # 生产包 (build/bin/wubigork.exe, 16MB)
go build -o build/bin/wubigork-v5.0.0.exe .  # 开发包 (不含嵌入 dist)
```

---

## v3.1.0 — Layered Glass 视觉重设计 (2026-06-21)

### 🎨 UI 全面升级 — "Layered Glass"
- **设计系统**: 17 个新 CSS 设计令牌 (accent-rgb, shadow-sm/md/lg/glow, border-subtle, bg-deep/base/elevated/glass, radius-sm/md/lg/xl, transition-fast/normal/slow)，4 套主题统一渐变+辉光
- **玻璃态全站**: 所有面板/卡片 `backdrop-filter: blur(8px)` + 半透明背景 + 柔和边框替代 thin 1px 实线
- **导航重设计**: 顶栏 sticky glass + 菜单去下划线/pill 选中态 + 底栏玻璃态
- **XAI 控制台**: 从等宽字体调试面板 → 圆角玻璃浮层 + 系统字体 + 彩色左边线
- **首页**: 项目卡片 glass-card 悬停上浮 + 空状态品牌水印 + 按钮辉光
- **写作页**: 玻璃侧边栏 + pill 标签页 + 编辑器内凹 inset 阴影 + 工具栏统一玻璃按钮
- **对话框**: Ant Design Modal 全局玻璃覆盖 + 弹簧入场动画 + 遮罩 blur
- **骨架屏**: 三处 loading 从裸 Spin 升级为 Ant Design Skeleton 骨架屏
- **无障碍**: `prefers-reduced-motion` 全局尊重

### ✨ 功能增强
- **TTS 语音朗读**: VoxCPM 本地神经网络合成 + Edge TTS 在线自然语音 + Windows SAPI 零延迟
- **写作页标签页优化**: 可横向滚动 + 关闭未保存确认弹窗
- **设置页**: 工作空间目录可配置

### 🔧 代码质量
- **DRY 清理**: 消除 9 处 `C()` 重复定义 → 统一 `import { C } from utils/theme`；消除 RelationGraph 颜色映射重复；提取 `sortNodes` 到 `utils/outline.ts`
- **错误处理**: 消除 15+ 处 `json.Marshal`/`io.ReadAll` 静默吞错误
- **死代码清理**: 删除 `internal/xai/` + `pkg/novel/` 重复函数 ~350 行

### 🛠 修复
- Edge TTS WebSocket 重写 + 引擎链降级 (SAPI → Edge → VoxCPM)
- VoxCPM 编译为静态链接，消除 DLL 依赖
- exec.Command 隐藏控制台窗口，消除朗读时弹 cmd
- NovelsDir 默认值不再依赖配置文件

---

## v3.0.0 — 革命性跃升 (2026-06-20)

### 🚀 核心创新
- **剧情分支选择器**: AI 推理 3-5 个下一章方向，用户选用或手工录入，自动写入大纲并同步角色/世界观
- **全书发展编辑**: AI 审读全书生成 1500 字编辑信 + 5 维评分 + 角色弧光诊断，对标 $3500 人工编辑
- **自我演化引擎**: 每章生成后自动分析 → 建议 Lorebook 词条 → 追加世界观空维度 → 记录伏笔变化
- **跨模块协作**: 大纲角色点击跳转角色页 + 角色出场章节查询 + 世界观一致性一键编辑

### 🎯 v2.0.0 功能 (2026-06-20)
- Story Bible 引导式创建（5 步向导：灵感 → 一键生成 → 角色优化 → 大纲优化 → 开写）
- 情节画布（水平时间线可视化全文章节 + 角色色条 + 品质情绪标注）
- Skill 管理面板（浏览/导入/创建自定义写作风格 Markdown）

### 💡 v1.5.0 功能 (2026-06-20)
- Lorebook 词条系统（定义概念 → AI 写作时自动注入相关上下文）
- 写作统计仪表盘（每章字数柱状图 + 品质趋势进度条）
- AI 审稿（5 维评分环形图 + 改进建议高亮卡片）
- Brainstorm 脑暴面板（输入题材 → AI 生成 6 个核心点子 → 选用创建）

### 🔧 v1.4.0 功能 (2026-06-20)
- 右键 AI 操作（选中正文 → 丰富描写/扩展场景/重写此段）
- 多章节标签页（同时编辑多个章节，独立状态）
- 大纲拖拽排序（Ant Design Tree draggable + 批量保存）
- 专注写作模式（全屏沉浸，隐藏侧栏和 Agent）

### 🛠 v1.3.2 修复 (2026-06-20)
- 大素材归纳压缩（先 AI 归纳关键信息再注入 prompt，防止上下文溢出）

---

**技术栈**: Go + Wails v2 + React 19 + Ant Design 6 + Three.js + d3-force-3d  
**构建**: `go build -o build/bin/wubigork-v4.0.0.exe .`  
**许可证**: MIT
## v4.164.0 · 收敛计划 W1 卫生刀：供应链可复现 + 测试抗抖 + 过期导航 id 修复（2026-09-08）
> 收敛计划立项（docs/gaea-convergence-plan-2026-09.md）后首轮四刀：①`frontend/package-lock.json` 入库（.gitignore 放行）+ CI `npm install`→`npm ci`——供应链可复现，CI 与本地同树（npm 11.6.2 的 `--package-lock-only` 产物有缺漏，改用 npm 10.9.9 生成后 `npm ci --dry-run` 通过；lockfileVersion 3 / 701 包）；②vitest 抗抖——`testTimeout` 5s→15s + CI 前端 job 加一次 flaky retry（对齐 Go job 形态），动因=同日实测双测试套件同机满并发 32 例超时假红、隔离复跑全绿；③修 DeliverablesPanel「回办公面板重新规划」按钮失效——根因=NAVIGATE 载荷沿用历史模块名 `page:"office"`，而 manifest 办公板块稳定 id 已是 `gaea`，MainLayout 白名单 fail-closed 静默丢弃（v4.121 走查遗留潜伏 bug，先失败用例钉死后最小修复，+1 回归锁）；④WebView2 壳内残留面只读审计落档 docs/webview2-shell-audit-2026-09.md（P0×2：BaselinesPanel 签证台账 CSV 导出/CharacterLibEditor 参考图；P1×7；修复刀序 A-D；print/FileSystemAPI 复核=唯一打印路径 iframe print 待真机、FileSystem API 全库零命中）。零绑定变更（drift PASS@602，v4.163 文案 603 系口径出入，以生成器实数为准）；Go 全量绿（116 包 0 FAIL）、vitest 2677→2678（全量首跑 2677/2678，ProgrammingPage 1 例负载 flaky 隔离复跑绿；整轮复跑 2678/2678 全绿——反过来印证刀②必要性）、tsc/eslint 0、build+冒烟过。欠账：nanoid<high 审计项（GHSA-2v37-7h3g-55p8 传递依赖，已锁版本待升级刀另立）；审计刀 A-C（P0/P1 修复）与刀 D（真机取证）待续；LICENSE/性质路线待拍板。详见 releases/v4.164.0.md。
## v4.165.0 · 拍板落档（路线A+私有许可）+ 壳内残留审计刀A：P0×2 修复（2026-09-08）
> 收敛计划 §0 两项拍板落档：**性质路线=A「终极个人工具」**（产品化降为期权不作承诺，不为想象中的用户写代码）+ **LICENSE=私有 All Rights Reserved**（根目录 LICENSE，未来开源须另行发布开源许可证覆盖对应模块并与私有部分区隔；README 加「许可」段）。审计刀A（docs/webview2-shell-audit-2026-09.md §5）：①新中立层 `gaea/lib/pickFile.ts`——`inShellEnv()` + `pickFileAsFile(accept)`（GaeaPickFiles 无过滤器→扩展名后置校验 fail-closed 抛错）+ `pickImageAsDataUrl()`（mime 映射+octet-stream 诚实降级），只 import ./bridge 防域倒挂；②P0-1 BaselinesPanel 导出签证台账 CSV 改 `saveExportBlob`（壳内系统另存为，浏览器回退不变）——v4.162 修了 SchedulePage 主链路漏了同域邻居；③P0-2 角色库「添加参考图」壳内改 `pickImageAsDataUrl` 读回 dataURL（此前点击完全无反应且无粘贴/拖拽替代，下游 img2img 生立绘被卡死），浏览器回退原生 input、hidden input 与既有用例口径原样保留。红→绿全链取证（修复前 GaeaPickFiles 0 次调用=点击无反应本体复现）；vitest 2678→**2684**（+6：pickFile 5+壳内用例，全量**首跑全绿**——v4.164 抗抖当场见效）、Go 全量 0 FAIL（零 Go 变更）、tsc/eslint 0、drift PASS@602（零绑定）、build+冒烟过。欠账：审计刀B（下载类×4）/刀C（上传类×3）/刀D（真机取证+PickFiles Filters）待续。详见 releases/v4.165.0.md。
## v4.166.0 · 瘦身 P1 快赢：轮子四刀全收 + 审计刀B/C + locale 死键 + 依赖验活（2026-09-09）
> 瘦身执行层首版（规划=docs/gaea-slim-masterplan-2026-09.md，P1=低风险机械刀热身，零功能损失）。**W1 字节+双门收口**：新中立层 `gaea/lib/bytes.ts`（b64ToBytes 唯一解码规范）+ `gaea/lib/saveFile.ts`（downloadBlob/dataUrlToBlob/saveExportBlob 从 schedule/exportArtifact 迁入，schedule 侧 re-export 零改动防域倒挂）；全仓 b64 手搓解码 ~20 处收口（DocxPreview/docxOutline/docxText/pptxTextDiff/pickFile/schedule/api/exportArtifact svgToPdf/TTSPlayer/ChatPage/SchedulePage/useVoiceChat/useImageGenHistory/ImageGenPage 等）；inShell 判定合一=pickFile.inShellEnv 全仓唯一规范（exportArtifact.inShell 薄委托）。**W2 slug×5 收敛**：新 `internal/gaea/strutil/title_slug.go`（TitleSlug：小写/保留 Unicode 字母数字/其余折叠 '-'/60 rune 截断/entry 兜底）；cost.SlugName（保留 "cost" 兜底）/knowledgeimport.slugName/app.slugFromTitle/modelengine.customSlug（保留 ASCII 白名单前置）/controller_memory.slugifyName（界面文案）并入；legacy golden matrix 测试钉死五旧实现逐项对照。**W3 novel diff 收口**：DiffReview 手搓弱贪心行 diff → `lib/diff.ts` LCS（贪心 3 行窗口漂移边界用例锁死；空串=0 行显示不劣化）。**审计刀B（下载类×4）**：ImageGen 两处（useImageGenHistory/ImageGenPage）壳内 dataURL→Blob 走 saveExportBlob、浏览器回退 a.download；NovelSetting 导入（pickFileAsFile）导出（saveExportBlob）双门；办公 md 分支并入 exportConversation 统一交付管线（App.export.test 源级回归锁）。**审计刀C（上传类×3）**：ControlPanel/VisionTrial/SkillModal 壳内走 GaeaPickFiles 系统对话框+扩展名后置校验 fail-closed，浏览器回退原 input（各 +3 用例）。**locale 死键清零**：scripts/slim-locale-deadkeys.mjs（dry-run 默认/--write 落盘；1445 键 0 死，en/zh/zh-TW 同步删）。**依赖验活**：26 个运行时直依赖逐个 import 计数——存疑名单 gsap(2)/docx-preview(1)/unist-util-visit(2) 全部洗清；**codemirror 顶包死重移除**（源码零引用、lockfile 零依赖者），CodeEditor 直接使用的 @codemirror/state|view|commands|language 四子包显式进 dependencies（npm 11.6.2 uninstall 再次写坏 lockfile 缺 wasi-threads/tslib 条目——git restore+手动删条目+npm ci --dry-run 验证）。门禁=Go 115 包全量绿（复跑清 flaky 同族）/vitest 2684→**2712**（+28，311 文件）/tsc 0/eslint 0/drift PASS@602（零绑定）/vite build 过。欠账=审计刀D（真机取证+Filters 可选参数）、P0 基线快照表入档、nanoid 升级刀、downloadMarkdown 死代码下轮清。详见 releases/v4.166.0.md。
## v4.167.0 · 瘦身 P2 刀0（白名单解耦）+ P0 基线落档 + 审计刀D Filters + nanoid 升级（2026-09-09）
> 瘦身执行层第二版，三线并发子代理（足迹互斥）+ 主代理第四线 + 主代理收口。**P2 刀0 白名单解耦**：`deriveNavigateWhitelist` filter 由 `inMenu && !isHome` → `!isHome`——**注册即可导航**，inMenu 只控菜单不再派生可导航性（v4.164 前科根治：藏板块 NAVIGATE 不再被白名单静默丢弃；刀1 schedule inMenu=false 依赖本语义）；settings 进白名单（有隐式入口）、fixture weixin 进白名单（角色库→青鸟 crossSpace 深链依赖），菜单 menuBoards 展示层零改动。**P0 基线落档**：`docs/gaea-slim-baseline-2026-09.md`（基线证据层）——§1 七面四表实测（dist 10.0MB/227、entry 1192.79kB、MemoryHub 1436.11kB 解剖实证 3d-force-graph 在页/cytoscape+cynefin 独立 chunk、mermaid/katex/CodeEditor 仅宿主视图、exe 46.22MB、npm 直依赖 29、>50KB 源文件 16）、§2 功能等价快照（13 板块×核心动作，纠正 knowledge 非一级导航=后端 D7 被过滤、KnowledgePage 孤儿页入 P2 待证）、§3 IA 走查（rail 12 板块平铺不分空间+foot code 区**勘出 code 双入口疑似项**、默认空间=work、首页 Bento 无空间过滤、**双空间并列未表达实证成立**仅 3 处空间感知）。**审计刀D Filters 可选参数**（代码化部分）：GaeaPickFiles 新增可选扩展名过滤（Go 转 Wails FileFilter 前置系统对话框，兼容 Windows）+ 前端 bridge/mock/pickFile 传参（accept 逗号拼接），扩展名后置校验保留 = fail-closed 双保险；pick_file_filter_test.go 7 例 + pickFile.test.ts 2 例。**nanoid 升级**：<3.3.18 审计实证（修正 <3.3.8 口径过时），3.3.17→**3.3.18**（纯传递依赖 vite→postcss，范围天生含免 overrides；npm 11 重写 lockfile 写坏→git restore+手动单条目+dry-run，diff 恰好 3 行）；`npm audit` total=0，**GHSA-2v37-7h3g-55p8 清零**。**downloadMarkdown 死代码清理**：export.ts 删除零调用函数，App.export.test 源级断言强化（无 downloadMarkdown/无 createObjectURL，md 走 exportConversation 意图保持）。门禁=Go 115 包全量 0 FAIL/vitest 2684→**2715**（311 文件）/tsc 0/eslint 0/drift PASS@602（Filters 签名变化方法名零变更）/vite build+wails build+冒烟 200。欠账=审计刀D 真机取证（printSvg print/拖拽/粘贴）、P2 走查待证项（rail code 双入口/后端 nav 子项差异/knowledge 孤儿页）、P2 刀1/刀2 待续、双空间并列落地随 P2 主干。详见 releases/v4.167.0.md。
## v4.168.0 · 瘦身 P2 刀1：schedule 并入办公文档面 + 命令面板入口 + 刀2 补验落档（2026-09-09）
> 瘦身执行层第三版：P2「schedule→办公文档化」第一刀落地（刀0 白名单解耦承前），三线并发子代理+主代理收口。**刀1a**（inMenu:false，前端 manifests.ts schedule `inMenu:true→false`：藏出顶栏菜单，白名单仍含 schedule=注册即可导航；manifests.test 菜单三断言 12→11 项+`not.toContain('schedule')` 锁、navigateWhitelist 保持 12 项含 schedule）。**刀1b**（办公「进度计划」入口）：App.tsx paletteItems 新增 cmd-schedule（Ctrl+K 直达，title 复用 schedCard.tag 三语键/icon=LineChart/run=NAVIGATE{page:schedule} 与 ScheduleFileCard 同通道）；办公文件面可达性核验=RecentFilesBar 单源含 .gsched→FilePreview 摘要卡既有链路，零新增数据源；App.palette.test 源级回归锁 3 例。**刀2 补验审计**：docs/gaea-slim-knife2-audit-2026-09.md（证据层）——刀2 主体已提前在产（v4.121 刀12 ScheduleFileCard+v4.139 多工程），三要素 2.5/3 实证（摘要卡+打开工作台✅/最近文档可见✅/.mpp 走导入✅），勘出 G-1 前后端分裂。**主代理收口**：G-1 后端同步翻 `internal/app/board/builtins.go` schedule InMenu:false（normalizeManifests 后端字段优先防真机/浏览器分裂）+ board_manifest_test 补断言；launcher.test 静态清单 12→11 卡同步；**v4.167 遗留绑定签名修复** DataPanel `GaeaPickFiles()`→`GaeaPickFiles('zip')`（wails 生成 OfficeB.d.ts 必填签名 TS2554，顺带恢复备份对话框前置过滤 zip）；**G-5 补验测试** FilePreview.test 新增 .gsched 分发 2 例（合法→摘要卡/坏 JSON→回落文本，LocaleProvider 包裹先例）。门禁=Go 115 包全量 0 FAIL/vitest 2715→**2720**（312 文件，首跑 1 flaky 复跑清同族）/tsc 0（基线 TS2554 随 DataPanel 修复清零）/eslint 0/drift PASS@602（零绑定）/vite+wails+冒烟 200。欠账=审计刀D 真机取证、P2 走查待证项（rail code 双入口/nav 子项差异/knowledge 孤儿页）、刀2 G-2 基线计数口径关闭/G-3 真机走查、观察池刀3（工作台内嵌办公）待一周评估、双空间并列落地+home 空间感知化随 P2 主干。详见 releases/v4.168.0.md。

## v4.169.0 · 瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口（2026-09-09）
> 瘦身执行层第四版：P2 形态主干收官——**工位/乐园并列结构在 UI 上成立**（masterplan §轨道一·2/3；审计=docs/gaea-slim-p2-dualspace-2026-09.md）。三线并发子代理（线A rail 双空间分域 / 线B home 空间感知 / 线C 走查项收口）+ 主代理收口。**线A**：CommandRail 新增 space/onSwitchSpace props + rail 顶部**工位/乐园切换器**（SHELL_SPACES 驱动，labelKey 三语，aria-pressed 激活态）；rail 主体改 `getActiveMenuBoardsForSpace`（shared 恒在/independent 剔除/settings inMenu:false 不在 rail）——**基线 §3.1 rail code 双入口缺陷闭合**（independent 仅 foot 单列，CommandRail.test 断言全 rail 编程入口=1）；manifests.test +4（work/play 派生序精确锁定）新增 CommandRail.test 5 例。**线B**（home 空间感知化）：Bento 按当前空间过滤（shared 保留、编程不再上首页、settings 两空间皆在）+ `LAUNCHER_FEATURED`（work=gaea 办公旗舰/play=chat 会客厅旗舰）+ hero 空间 chip + **工位右舷「最近文档」面板**（localStorage 单源 loadRecentFiles 前 5，点击→办公，零新 binding）；launcher.test +3。**线C**：**knowledge 孤儿页处置**（main.tsx 移除注册+删除 pages/KnowledgePage.tsx，全仓 grep 实证零引用；知识库=记忆中枢子面等价，后端 D7 保留导航侧过滤）；**刀2 G-2 基线数关闭**（GschedSummary 增 baselineCount + ScheduleFileCard「基线 N」StatChip，测试 +5）；**后端 nav 子项差异取证**（cost 后端 7 vs 前端静态 4、novel 后端 6 vs 5=有意的双源回退，后端字段优先无需改码）。**主代理收口**：三语字典契约键统一（shell.space.* 书房/庭院→**工位/乐园**+home.recentDocs×3+schedCard.baselineCount；progEntry 死键随编程瓦片移除清零，1448 键 0 死）；审计/基线/刀2 三文档回写；版本三处 4.169.0。门禁=vitest 2720→**2737**（313 文件全绿）/Go 127 包 0 FAIL（首跑 1 包并发 flake 复跑清同族）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 死键 0。欠账=壳内真机走查（rail 切换器/home 最近文档/.gsched「基线 N」/knowledge 无残留）、审计刀D 真机取证、观察池刀3 待一周评估。详见 releases/v4.169.0.md。

## v4.170.0 · 瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5）（2026-09-09）
> 瘦身执行层第五版：P3 结构面第一刀（masterplan 轨道三；出口判据=绑定零变更 drift PASS + >50KB 9→5），四线并发子代理（足迹互斥）纯结构拆分=行为零变化/公开导出面逐一同。**线B App.tsx** 87.3→46.6KB：palette 构建/exportConversation/JSX 布局因 App.export.test/App.palette.test src 级断言刻意留壳内；11 组逻辑迁新 `gaea/app/`（useSubagentTabs/useTaskCards/usePreviewAutoFront/useDeliverables/useWorkspaceLayout/usePreviewPanel/useSessionHandlers/useAppKeyboard/useTasksAutoOpen/sessionCleanup/layout），deps 数组与执行顺序逐项原样（35→16 组 effect，原 2 处 eslint-disable 保留）。**线C bridge.ts** 85.1→1.3KB：按 bindingNames 域切 `gaea/lib/bridge/` 16 文件（10 分域接口+`interface AppBindings extends …` 组合+mappings/proxy/events/http/drift），Proxy/懒探测/mock 回退/事件时序原样，方法名 parity 315、bindingNames 602 零变更。**线D types.ts** 66.4→1.2KB：纯类型按域拆 `gaea/lib/types/` 24 文件（220 声明零增减重名），入口纯 type re-export，135 处消费方零改动。**线E GanttView.tsx** 64.8→25.2KB：纯函数/常量/子组件拆 `schedule/gantt/` 8 文件（ganttUtil+PredEditor+CustomFieldCell+GanttToolbar+GanttTable+GanttTimeline+GanttBars+GanttOverlay），classNames/data-testid 逐字节照搬。门禁=Go 127 包 0 FAIL/vitest 2737（313 文件全绿，纯拆分零新增测试）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死。欠账=P3 版2 待续（office 抽核 journal/evidence/文件读写→internal/core + bridge 双轨退役启动（wailsjsCompat 57 引用）+ 次批 5 巨文件 config/KnowledgePanel/DeliverablesPanel/XlsxPreview/ChapterPage）、壳内真机池/审计刀D 不变。详见 releases/v4.170.0.md。
## v4.171.0 · 瘦身 P3 结构版2：office 抽核 + 次批巨文件拆分（>50KB 首拆池 9→0）（2026-09-09）
> 瘦身执行层第六版：P3 结构面第二刀（masterplan 轨道三；出口判据=绑定零变更 drift PASS + >50KB 首拆池 9→0）。承接 v4.170.0 首批 4 拆的次批收口 + 轨道三「office 抽核」，纯结构拆分=行为零变化。**线A office 抽核**：internal/office 顶层 8 文件（executor/job_manager/job_routing/session_mode/desktop_session/audit_log/types + 测试）→ 新包 internal/core（fs/jobs/journal/modes/routing/sessionmode/types），新 aliases.go 类型别名 ExecResult/AgentJobState 保 bindings_office.go 生成签名（JSON/反射零变化），app 层 import 收口——绑定面 602 零变更。**线B 次批 5+1 拆**：KnowledgePanel 62.2→40.8（knowledge/ 8 文件）/DeliverablesPanel 57.4→23.2（deliverables/ 4）/XlsxPreview 56.4→35.5（xlsxpreview/ 4）/ChapterPage 55.9→35.6（chapter/ 5）/herdsmanTemplates 55.7→1.6（data/templates/ 13）/SchedulePage 超额（schedule/ 3）；classNames/data-testid 逐字节照搬。收口修复=ExportDialog react-refresh warning×2（非组件导出仅内用却带 export→去 export 本地化，eslint 0 error 0 warning 恢复）。⚠接手会话收口：工作树由前一会话完成遗留未提交，本会话全量门禁核验+修复 lint+补齐发布。门禁=Go 128 包 0 FAIL/vitest 2737（313 文件全绿，首跑 4 文件 31 例与 Go 门禁同机并发负载假红复跑两次全绿，在册先例）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死。欠账=P3 版3 候选（bridge 双轨退役启动 wailsjsCompat 57 引用、config.go 58.7/engine.go 55.2/store.ts 54.3 二次拆分、entry 懒加载）、壳内真机池/审计刀D 不变。详见 releases/v4.171.0.md。
## v4.172.0 · 瘦身 P3 版3：bridge 双轨退役批次一（wailsjsCompat 首批 12 文件迁 bridge）（2026-09-09）
> 瘦身执行层第七版：轨道四「双轨」渐进退役第一刀（出口判据=绑定零变更 drift PASS + 迁移面行为零变化/错误归一 BridgeError）。调研先行（39 非测试文件=2A/24B/13C 三分类，权威「③ 不在 bridge」清单=drift.ts LegacySurfaceNames 292 项），本版消 A+B 首批 12 调用方。**线1 契约扩展 16 新方法**：CoreBindings+4（GetAppInfo/GetBoardManifests/GetFeatureModel(feature)/GetFeatureModelEnabled(feature)）/ModelBindings+3（SetFeatureModel 3 参/SetFeatureModelEnabled/GetActiveModel）/ImageBindings+1（StartLocalTTSService）/VoiceBindings+6（VoiceApplySettings+Whisper 读写族 5，Go 侧实挂 VoiceB 纠偏）/NovelBindings+1（NovelSearch 单参）/OfficeBindings+1（SaveSettings+mappings）；drift 摘除 16 名、spaceBindings +16 分类锁 315→331、mock 补齐含新建 voice/novel 模块。**线2A 零依赖直迁**：PersonaPicker/CharacterCard（WhisperAssistantSave Partial 零断言）。**线2B 10 文件 4 路并发**：CharacterMemoryModal/WhisperMemoryModal/manifests(+tests)/AboutPanel/useChatVoice/useFeatureModel/BindSection/useEngineState(+test)/novelSearchUtils/OfficePanel（gaeaApp.SaveSettings/Reload，AppModels.SettingsView→gaea SettingsView）。**主代理收口**：连带测试 ChatPage/ChapterPage 补 bridge vi.mock 与 wailsjsCompat 共享同一 vi.fn（bindingBridge via vi.hoisted+Proxy）——首跑 5 例断言打空转绿 22/22；**坑=vi.mock 相对路径必须与真实 import 绝对解析一致**（pages/ 下 bridge 是 ../gaea/lib/bridge 非 ../../，错路径 mock 静默失效断言 calls=0）。门禁=vitest 2737/2737/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=C9B020CE9B0257FDC21FE00D9C0417D0CCE4D47767BBE1BF2310B9AB1799AFE1 wailsjsCompat 引用 57→45。欠账=批次二（api/settings 一次扩解锁四面板/useChatTopics 元组重构/useChatStream mock 事件/Novel 面板族/ModuleLauncher）、批次三（语音族直调语义重评/cast 族先入类型/角色库族/ChatPage 五直调/ChapterPage 四直调/useBindState/DataPanel）、全部完成后 wailsjsCompat.ts 退役。详见 releases/v4.172.0.md。
## v4.173.0 · 瘦身 P3 版3：bridge 双轨退役批次二（chat/novel/settings 三族 22 文件迁 bridge + 元组契约修正）（2026-09-09）
> 瘦身执行层第八版：轨道四「双轨」渐进退役第二刀。六线并发（1 契约扩展 + 1 契约修正 + 4 业务线，足迹互斥）+ 主代理收口；批次一延续，wailsjsCompat 引用 45→35、LegacySurfaceNames 276→247。**线1 契约扩展 29 新方法**：CoreBindings+5（GetConfig/SaveConfig/GetStats/ListSkills/ExportAll——ExportAll 实测在 bindings_core.go:23）/ChatBindings+9（ChatTopicCreate/Delete/Rename/SetMode/Clear/ImportTopic/Send 6 参/StreamPlain 5 参/ExportMarkdown）/NovelBindings+12（GetWorldview/SaveWorldview/**ChatWorldview 契约纠偏=bindings_chat.go:35 返回 map 非 string**/GetWorldviewSections/SaveAllWorldviewSections/CheckConsistency/CheckConsistencyDeep/GetForeshadows/SaveForeshadows/SaveCharactersBatch/NovelReadingAsk/GenerateSceneIllustration）/VoiceBindings+2（VoiceGetSettings/VoiceChatText）/ImageBindings+1（SetImageBackend 4 参）；已核实跳过 WhisperGetPersonalities/ListSessions/MemoryHubOverview；drift 摘除 29（误删 MigrateProjectToV4 已恢复）、spaceBindings +29 锁 331→360、mock 补齐。**线1b 元组契约错位修正**：bridge/chat.ts 原声明 [T[],unknown] 元组系 T6-3.2 历史误读（注释自认），真实 Wails 对 Go ([]T,error) 成功返回数组失败 reject——修正为数组契约+mock 返回 []+mock-contract-t63 断言改 Array.isArray，下游零消费者（tsc 全绿证明）；useChatTopics 迁移免去调研预测的元组解构。**业务线×4（22 文件）**：chat 组（useChatTopics 11 处+ChatTopicCreate as unknown as 双断言/useChatStream Record unknown 两处最小 cast/chat.utils LogFrontendError 映射/ChatPage.test bindingsBridge 扩展 11 共享 fn——22/22 绿）；novel 组（10 源+4 测试，NovelSettingPage.test app 用 Proxy 回落真代理保审计刀B b 三例，ChatWorldview unknown 收窄——25/25 绿）；settings 组（api/settings.ts 8 调用全命中迁移，ChatPanel/DataPanel/VoiceSettingsPanel/useVoiceChat 宁少勿多递批次三）；ModuleLauncher（4 处迁移无遗留）。门禁=vitest 2737/2737 首跑全绿（批次一曾 5 例连带失败本次零失败）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=C3DFA08F1A56C2389C2E4D5515A0E7F09D017B0DFB763DD0CDC459EBE66750CE 版本三处 4.173.0。欠账=批次三（语音族 useVoiceChat/VoiceSettingsPanel 六方法转正+直调同步 throw 语义重评、ChatPage 五直调、WhisperClearSession/GetVoicePipelineConfig/GetTTSSpeakers 转正解锁 ChatPanel、DataPanel GaeaDataBackup×6、cast 族 CreatePage/ChapterEditor 先入类型、角色库族 api/characterlib+novel/api/character、ChapterPage 四直调、useBindState/DataPanel、wailsjsCompat.ts 退役+悬空注释清理）。详见 releases/v4.173.0.md。
## v4.174.0 · 瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat.ts shim 删除，全仓生产代码零引用）（2026-09-09）
> 瘦身执行层第九版：轨道四「双轨」渐进退役收官刀。批次三全景收口——语音族/表情/章节族/小说状态族/场景生成族/角色库族/设置面板族全部迁 bridge，frontend/src/wailsjsCompat.ts shim 删除；里程碑=全仓生产代码 wailsjsCompat import 清零、LegacySurfaceNames 292→184（累计转正 108 方法）。**契约扩展批次三（四线共 42 方法）**：3a +18（VoiceBindings 9 语音/WhisperClearSession/TTS×2 + ImageBindings 2（GetTTSSpeakers/GetVoicePipelineConfig 实挂 ImageB）+ OfficeBindings 6 DataBackup* + Core 1 ListProjects）；3b +38（CharLibBindings 16 角色库族 + NovelBindings 22 章节族/叙事状态族/场景族/项目角色族，**GetCharacters 同名核对**=charlib.ts 已有与 Go NovelB 同签名直接可用，CancelCreateChapter→boolean，**v4.7x 小说革命注释段更新 RunChapterGate 保留**）；3c +3（QuickBrainstormBranches/CreateChapter 8 参/DeleteOutlineNode）；3d +4（SetActiveASRModel/SetActiveTTSModel/SetChatVoiceModel ImageB + VoiceHealth VoiceB）；spaceBindings 360→378→416→419→423。**业务迁移 8 线并发**：组1 语音族（useVoiceChat 8 方法 VoicePushAudio Array→base64 对齐 Go []byte 契约 + VoiceSettingsPanel 4 + ChatPage 5 直调 + ChatPage.test bindingsBridge 扩 18 fn——22/22）；组2 cast 族（CreatePage/ChapterEditor 的 App as unknown as cast 对象全删——绑定再生成后 cast 编译期冗余，走 NovelB 门面具名——9/9）；组3 角色库（api/characterlib 18 方法 6 处 as unknown as，组件层 adapter 零改动——44/44）；组4 章节/小说角色（ChapterPage 4 + novel/api/character 9——7/7）；组5 收尾（useBindState 4 + DataPanel 9 + 2 测试）；CreatePage 收尾（7 处+契约 3c——9/9）；终局 3d（useVoiceState 5 + VoiceHealth + ChatPanel 4 保留整体 try/catch——47/47）；主代理收尾（ChatPage.test/ChapterPage.test 别名改 bindingsBridge 直取、SettingsPage.test 死 vi.mock 改 bridge、**wailsjsCompat.ts 删除**）。门禁=vitest 2737/2737 首跑全绿（shim 删除后零回归）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=AD7C7CED1066F7BFF077B2416D41145BEE207619868D411F9A39FDF518737E2C 版本三处 4.174.0。收官意义：S2-3 兼容层退役全部前端调用统一 gaea/lib/bridge 代理（?mock=1 dev mock 回退+BridgeError 错误归一），masterplan 轨道四「双轨」目标达成；CharacterList 分类遗留裁定=work（v4.48 微信触点先例）有意保留。后续=P4（config.go/engine.go/store.ts 拆分、entry 懒加载、exe strip）+ 壳内真机池/审计刀D。详见 releases/v4.174.0.md。
## v4.175.0 · 瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）（2026-09-09）
> 瘦身执行层第十版：P4 性能版结构前置刀——P3 遗留 Go 侧两个 >50KB + frontend 一个 >50KB 源文件纯结构拆分。三线并发子代理（足迹互斥）+ 主代理收口，行为零变化（导出面/函数签名/JSON tag/默认值逐字节保留）。**线A config.go 58.7→16.1KB**（7 文件同包：config.go 核心骨架+config_keys 8.6/config_types 14.7/config_features 2.7/config_prefs 4.9/config_realtime 0.9/config_save 12.5；逐段 Contains 断言 8 区逐字节命中）。**线B engine.go 56.5→13.3KB**（8 文件同包：engine.go 核心+engine_keys 3.0/engine_custom 7.0/engine_crud 3.7/engine_connect 3.8/engine_models 12.6/engine_modelhub 10.9/engine_state 4.0；核实无 herdsman 专属函数未建多余文件；45 func 无重复无丢失）。**线C store.ts 54.3→0.4KB**（聚合入口 export *×3 + store/controller 50.2 + store/preview 3.3 + store/commonts 1.5；18 导出面逐一同 63 消费方零改动；963/963 行逐字节对账；controller 相对 import 深度调整 4 行）。主代理收口：足迹 17 项全在目标域零越界；Go >50KB 源文件 2→0；**mock/office.ts 51.2KB 浮现**（下轮目标）；store/controller.ts 50.2KB 裁定=单 store 自然边界保留（再拆超行为零变化红线）；entry chunk 1190.55→1176.62kB。门禁=Go 128/128 包 0 FAIL/vitest 2737/2737 首跑全绿/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/冒烟 200/SHA256=729C942B1A21B4FE0DB242565808B8CE86DB6F5B812810C836DDEBE4065F9DB8 版本三处 4.175.0。欠账=P4 续（mock/office.ts 51.2 拆分、entry 懒加载 MemoryHubPage 1.4MB/GaeaPage 622KB/cynefin 690KB 按 tab 动态 import、exe strip -ldflags -s -w 对靶 46.22MB）、壳内真机池/审计刀D 不变。详见 releases/v4.175.0.md。
## v4.176.0 · 瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）（2026-09-09）
> 瘦身执行层第十一版：P4 性能版第二刀。**结构刀2 mock/office.ts 51.2→1.1KB 入口**（6 文件：types/state/schedule 10.5/methods_xlsx 4.9/methods_office 32.7/build；**TS2632 教训**=let mockScheduleCurrent 被 Create/Delete 直接重赋值禁跨模块，schedule 状态与方法必须同文件，与 weixin.ts 先例一致）。**运行刀 entry 懒加载**（调研修正 P0 两处：cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页非页面 lazy）：H6 mock 异步 chunk（proxy.ts startMockChunk 单例+真机门控零加载、events 走 mockEventSharedSync/waitMockReady；**index 1149→967.23KB −182KB**）；H2-H5 MemoryHubPage 8 组件全页内 React.lazy（**1402.5→15.95KB 页壳**、GraphView/three.js 1,354.97KB 独立 chunk）；H1 mermaid 动态 import 递下一轮（mermaidPng.ts footprint 外 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。连带测试时序修复×2（mock 异步化使 window.__mockScheduleFile 与 SearchModal RouteIntent 演示规则未就绪——补 beforeAll waitMockReady，bridge.ts 入口加 re-export；首跑 7 例失败修复后全绿）。门禁=vitest 2737/2737/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F 版本三处 4.176.0。欠账=P4 续（H1 mermaid 动态 import、H7 locales 按需、H8 SearchModal lazy、exe strip wails -ldflags -s -w -trimpath 发布版）、壳内真机池/审计刀D 不变。详见 releases/v4.176.0.md。
## v4.177.0 · 瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）（2026-09-09）
> 瘦身执行层第十二版：P4 运行刀收官。两线并发（H1 + H7/H8，足迹互斥）+ 主代理收口。**H1 mermaid 动态 import**：Markdown.tsx/mermaidPng.ts 删静态 import、ensureMermaid 改异步 await import（模块级缓存，initialize 配置原样 strict/loose 各自保留）；MermaidBlock 渲染迁 useEffect async IIFE（取消/失败语义不变）；**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用（基线有静态引用）、566.9KB 独立 async chunk 按需**；测试零改动（5 个 Markdown 测试均不渲染 mermaid 围栏）。**H7 locales 按需**：zh 静态保留 + en/zh-TW 动态 import 显式分支（Vite 静态解析独立 chunk）；useT/t/setPref 签名零变动；Provider 切换先同步渲染回退→chunk 就绪 forceRender，alive 防过期。**主代理收口关键修复**：translate 回退链 DICTS[locale]→DICTS.zh→DICTS.en→key（zh 静态恒可用兜底——非 React 调用方 tools.ts 摘要 currentLocale 默认 en 在 en chunk 未加载时裸键永不外泄）；5 测试补 beforeAll(loadLocale('en')) + ModelSwitcher en 用例 await。**H8 SearchModal lazy**：React.lazy + Suspense fallback null；SearchModal.test 不经 MainLayout 零改动。门禁=vitest 2737/2737（首跑 13 例 i18n 时序/裸键失败 zh 兜底+适配后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15绿）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0 版本三处 4.177.0。度量=entry 1193→755.48KB（−37%），en/zh-TW/SearchModal 独立 chunk。欠账=壳内真机池/审计刀D/观察池刀3 不变（**exe strip 已随本版固化进 build.bat 并实测销账**：-ldflags "-s -w" -trimpath，46.2→46.15MB 减量甚微——wails 生产构建默认已剥符号；SHA256SUMS 更正为 strip 后可复现产物 D027D29B…，tag 不动）。详见 releases/v4.177.0.md。
