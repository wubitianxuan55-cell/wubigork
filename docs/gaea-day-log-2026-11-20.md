# gaea Day Log · 2026-11-20 · 设置中枢全线收官 + 设置板块整板二次重设计

> 接 11-19 深夜设置中枢七批+尾批（装机 PID 38556）。今日用户连发「继续」驱动：
> ①模型中心直达回摆（拍板推翻批 7 tab 退役）②UI 重设计四刀（点名用技能）③进度/
> 记忆中枢索引卡（批 7 挂账收口）④装机+CDP 走查+74 笔列车推送⑤键盘可达⑥竞态
> 根修⑦走查配方转正+快捷键可发现性⑧亮色主题验证⑨搜索直达项。全日 12 提交
> 全推 origin（ahead=0）+装机同步（终态 PID 26064）。

## 1. 模型中心「接入」节直达回摆（6cd9f0a3）

用户拍板「模型中心打开可以直接进行模型引擎管理和功能绑定」——回摆批 7 的 tab 退役：
ModelCenterPage rail 恢复「接入」节两 tab（功能绑定=BindSection 单段；引擎管理=
SchedulingSection+EngineSection 两段，引擎域操作含保活/预载/故障转移先例同构）。
**单源组件双宿主**：设置页 model 组经 ModelSettingsProvider 消费同组件（批 3 抽出的
useModelCenterBundle 双宿主共用），非复制漂移源；图片生成/语音模型仍留设置页
imagegen/chat 组不回摆；rail-foot 绑定摘要卡改回 setCategory('bind') 本页直达。
tabs 契约测试 5→7 tab/3→4 节+三断言。modelcenter 全族 120 例绿。

## 2. UI 重设计四刀（9fdb3d78，ui-ux-pro-max 技能）

技能流程跑通（--design-system 生成参考后**只取结构性建议**，外部配色/字体否决——
保持仓内 md-sys 令牌+玄墨语言一致性）。IA/批 1 骨架/测试契约零动，重画：
- **内容头**（settings-head）：中栏激活组头部（40px accent 图标章+17px 标题+描述+
  zone 徽标 全局/板块）——「当前在哪」不依赖左导航记忆；底色对齐 v3-card 信息层
  无 hover 光效（静态方位块）。
- **搜索零死端**：无匹配态升级为图标圆章+文案+高频组建议胶囊（SUGGEST_KEYS=
  通用/模型/办公/聊天，点击直达清搜索，与锚点直达同语义）。
- **导航精修**：zone 标题 accent 小 tick、激活项图标章 accent 着色、计数胶囊过滤态
  切 n/total、搜索框 250→300px、底部提示行与激活 orb 噪音退役。
- **inspector 升级**：「相关分组」快跳卡（注册表关键词交集计算，纯前端零绑定，
  slice 6 空则隐）+当前组图标章 accent 化；locales 三语 +settings.related。
令牌纪律全保持（零硬编码色值）。全量 4572 绿。

## 3. 进度计划/记忆中枢索引卡（e9a5ae2c，批 7 拍板挂账收口）

「使用中即时调」型显示偏好**边界留域**（偏好面留驻板块=单一事实源），设置页给索引卡：
新组件 BoardIndexSection（卡题+留域说明+可调项胶囊+跳转钮 NAVIGATE 直达板块页）。
板块区两组入驻（现七组）：schedule 组=网络图图上内容/关系线布局/工作日历/检视·
基线·资源与成本/项目设置索引直达；memoryhub 组=星图显示设置（轨道环/星尘/自动
旋转）+交互提示索引直达。图标复用 boardIcons（currentColor stroke 随激活态着色）。
locales 三语 +settings.cat.schedule/memoryhub 四键。坑入册：antd 图标 span 带
role=img aria-label=right 参与按钮可访问名，getByRole 精确名炸→走正则。

## 4. 装机+CDP 走查 17/17+74 笔列车推送

- **装机**：wails build 52s（ucrt64/CGO/系统 TMP 纪律）→ 冒烟 200 → INSTALL OK
  （exe 7CC3D49BFA1A，PID 24072>构建时刻验真）。
- **走查**：杀实盘壳释放 9333 → GAEA_WALKTHROUGH=1 沙箱壳 CDP 驱动 → 17 断言全绿
  （内容头/索引卡+NAVIGATE 直达 schedule/零死端/相关分组/模型中心 7tab+引擎管理
  两段）+双截图目检。首跑 2 FAIL 均系断言口径（__navSeen 历史累积/SectionHead 空
  span 噪音）非产品缺陷。
- **新认知入册**：gaea 壳因 `--disable-features` 被 Wails **连带默认开启**
  127.0.0.1:9333（非仅 GAEA_WEBVIEW_DEBUG 才开）=实盘壳恒占口，CDP 走查前必先收壳。
- **推送**：设置线 10 笔随 **74 笔列车** 04ad05f2..e9a5ae2c 推 origin（ahead=0；
  构成核对=设置 10+文学 P0+办公 R2+众神殿账本+DSH 子代理 11-18 各批+codex 蒸馏
  11-17 各波+发版/bindings 同步，无 WIP）。

## 5. 键盘可达批（c06bfb20）

ui-ux-pro-max Priority 1/2 补课：导航 ↑/↓ 环绕移焦/Home/End 跳端（roving focus，
Enter 原生激活，过滤态现场查询可见项不缓存句柄）；页面级 `/` 聚焦搜索（INPUT/
TEXTAREA/contentEditable 打字态不劫持，全仓查无全局 `/` 监听零冲突）；搜索框 Esc
有词清词无词退焦；antd Input ref 走 InputRef 类型。+3 断言。

## 6. 竞态根修（d4c2892e）——历批「mock.ts teardown 竞态噪音」定性+根治

**定性**：DeniedActionsDock（gaea/wenxue 双副本）防循环依赖的
`void import(bridge)` 无 .catch——测试环境 teardown 打断在途 bridge→mock 全图
chunk 加载时 rejection 落成 vitest unhandled error（历批文档噪音本体，voice/weixin
chunk 换位出现），**且会翻牌同 worker 兄弟用例**（全量挂数每轮漂移 1/0/1/2、挂文件
漂移、与改动零交集、单跑全绿）。
**修**：四处火后不理动态 import（双 dock+双 Insights saveRange）拒绝路径收编
console.warn（cancelled 静默；生产降级=订阅/偏好未保存仅告警）。修后两连跑 4577
全绿零翻牌。runner 级 EnvironmentTeardownError 日志仍存（不翻结果）——vitest v4
已知局限挂观察池不追（根修需动 pool/隔离配置，分片/isolate 实验均已证伪过）。

## 7. 走查配方转正+快捷键可发现性（77261987）

走查驱动从 .tmp 升入 scripts/gaea-settings-walk.mjs（17 断言 CDP 回归配方+9333
杀壳红线头注，设置域动刀复跑即回归）；搜索框 placeholder 三语追加「按 / 快速聚焦」。

## 8. 亮色主题验证（零修补）

重设计后目检首次补亮色：沙箱壳 CDP 切亮色（antd Segmented label 层点击；
data-theme 只挂主题名不挂明暗，明暗=App.tsx 运行时注入 --md-sys-* 令牌，验真用
getComputedStyle 读 --md-sys-color-surface-container/text=#EDEAE3/#1E1C18）→
三张截图目检（内容头+导航激活态+accent 图标章/索引卡胶囊/模型中心接入节 rail）
全部成立对比度充足——**token 纪律红利实锤**（color-mix 派生随 12 主题自动成立）。

## 9. 搜索直达项（c65c8a77）

搜索从「直达组」升到「直达项」：过滤态切组后短窗轮询（150ms×13≈2s 命中即停）扫描
已挂面板段标题（strong/h2~h4/mc-section-title/mc-bind-title，各面板头部异构故 DOM
扫描零面板耦合），首个命中查询词段卡 scrollIntoView(block:start)+settings-sec-flash
一闪 accent 环（1.6s ease-out 淡出，reduced-motion 全停）；无命中静默回落组级；
纯切组不触发。**轮询必要性实锤**：段标题随 bridge 数据异步挂载（OfficeMemorySection
等），单次扫描必扑空。实机验证：搜索「记忆」→2/15→办公组→面板滚至办公记忆卡
置顶+flash 按时淡出（亮色沙箱一并过目）。

## 10. 大坑与教训

- **测试文件漏 import waitFor → ReferenceError 189ms 秒抛，被误读成「2000ms 超时
  挂」→ 连锁误诊「假定时器/act 杀 interval」五轮探针全追幽灵**。教训=失败先看
  完整报错原文再立论，时长≠超时。
- 装机 KILL FAIL/Copy-Item IOException=句柄残留瞬时竞态，显式 Stop-Process 后
  重试即过（脚本重试语义已知）。
- Git Bash 内联 python 反斜杠降级（build\bin 字面量被吃）——按坑册走 .py 文件
  +chr(92) 配方。

## 11. 终态

- 12 提交全推 origin（487f3333..c65c8a77 含 74 笔列车+2+1，ahead=0）；全量 504 文件
  4578 用例+tsc+lint 绿；装机同步（PID 26064）；挂账池空（余=TTS 死键不收编〔拍板
  不做〕+双入口单源保留〔设计如此〕+EnvironmentTeardownError 观察池）。
- 板块验证面：暗色+亮色+键盘+4578 用例+CDP 17 断言，全在案。板块进入纯实弹期。

## 12. 设置板块整板 UI 二次重设计（工作树，未提交）

用户追加反馈「整个设置板块的 UI」「模型把以前多个标签的内容挤在一起」。本轮不重做
底层配置与绑定契约，重画 15 个分类共用的控制台外壳，并把多任务分类的旧纵向堆叠
改成任务页签。

- **整板外壳**：命令栏（控制台标识/搜索/帮助）+ 两级分类轨道 + 当前分区画布；
  分类轨道改为紧凑单行标签，分类说明集中到内容头；统一实底卡、标题/说明/状态、
  ISO 键盘焦点、skip link、reduced-motion。亮暗主题全部消费现有 md-sys/gaea 令牌。
- **任务分栏**：通用 3 / 聊天 4 / 绘梦 2 / 办公 4 / 模型 5 / 安全 2 / 数据 2 /
  文学 2 / 高级 2；关于、代码图谱、微信、众神殿、进度计划、记忆中枢保持单页。
  访问过的页签保持挂载，未保存草稿切页不丢；标准 tablist 支持左右方向键/Home/End。
- **模型专项**：全局概览、功能绑定、空间策略、本地调度、引擎管理分栏；1280×720
  mock 实页连续滚动高度从 3155px 收到 801px。补设置页 `--mc-*` 令牌作用域，修掉
  此前依赖模型中心懒加载 CSS 时空间策略等区域可能退回无样式纯文本的问题。
- **搜索/深链**：分类搜索可驱动内部任务页签，再在目标面板内锚定段卡闪示；语音入口
  经 `openSettings(category, tab)` 直达「朗读引擎」，不再落到聊天首页。
- **导航连续性续刀**：切换分类回内容顶部；分类轨道无论竖排还是窄窗横排都自动把
  激活项滚入视野；离开分类后记住上次任务页签；430px 横栏自动把当前任务页签带回
  可视区。页签记忆走 ref 写入，避免 effect→父 state→受控页签的重复渲染环（定向
  测试曾实锤整组用例挂起，修后 27/27）。
- **零引擎空态**：模型「引擎管理」在无引擎或筛选无结果时显示原因和添加入口，
  不再出现大面积空白；新增空态测试，EngineSection 11/11。
- **可访问性/响应式**：tablist 的 `aria-selected`/`aria-controls`/tabpanel 关系
  完整；1280 亮/暗、1024、720、390 实页走查无页面级横向溢出；亮暗正文对比度
  ≥14.16:1，次级文字 ≥6.59:1（浏览器计算值）。
- **验证**：定向 7 文件 76/76 绿；全量 503/504 文件、4583/4585 用例通过。失败仅
  `DatabaseLibrary` 两例全量并发 flaky，单跑 6/6 绿；另命中既有
  `wenxue/components/ToolCard.test.tsx` `EnvironmentTeardownError`，单跑 2/2 绿，
  均与设置域零交集。`tsc -b` 0；改动面 ESLint 0 error（ChatPage 两条既有
  exhaustive-deps warning）；`npm run build` 36.28s 绿（既有 Wenxue CSS syntax/chunk
  体积 warning）。`scripts/gaea-settings-walk.mjs` 扩至 24 断言并通过 node syntax
  check。
- **产物**：`C:/AI/wubigrok/.tmp/settings-redesign/settings-model-dark.png`、
  `settings-model-light.png` 与 `settings-engine-empty-dark.png`。未提交、未 Wails
  构建、未装机。
