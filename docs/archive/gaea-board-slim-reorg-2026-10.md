# 板块瘦身双刀纪要:模型中心 + 小说(2026-10-27)

> 用户指令「瘦身/重组模型中心自身」与「瘦身/重组小说板块,大道至简」。两处幅度
> 拍板点(AskUserQuestion)均未获即时应答,按推荐档执行——与既往拍板风格一致
> (实质收敛、简化≠删功能、同批顺手收口)。提交锚点:`43fa62d9`(模型中心)/
> `56b21897`(小说),随本批 30 提交整体推送 origin,**未发版**(发版随池另拍)。
>
> **2026-10-28 续刀**(用户「继续大道至简」,提交 `本批`):上表余池「创作页 18 弹层
> 交互与阅读页双 chrome」盘底定刀,见文末「续刀(2026-10-28)」节。

## 刀一:模型中心瘦身重组(43fa62d9)

**现状盘底**:左 rail 13 分类平铺无分组,`pages/modelcenter/` 44 文件 10268 行,
约 128 个 useState;热点 EngineSection(537 行/17 态)、HerdsmanCatalogSection
(395 行/14 态)、utils.tsx(~60 导出)、modelcenter.css(1376 行)。

**刀法(导航 13→9 温和合并 + 代码热点同批)**:

- **合并三组**(照 engine tab「一 tab 多段」既有先例,Section 组件全保留零删除):
  llm+specialty+recommend→「模型一览」三段;benchmark+retrieval→「测评中心」;
  strategy+attribution→「策略与成本」。rail 按概览/模型/质量与成本/接入四节分组
  (`.mc-rail-group`,对齐微信 rail 同款)。
- `Category` 类型 14→9 值(无消费 `'stats'` 残留一并清除)——tsc 编译期兜底:
  生产代码零断点,仅 2 处测试参数随迁。
- **EngineSection 拆件**:自定义引擎添加/编辑表单(含价目 v1 三态)→
  `CustomEngineForm.tsx` 内聚;单卡渲染+managedRunning 探活→`EngineCard.tsx`;
  主文件 17→5 态留编排(savedPrices 会话级价目记忆留编排层防跨开合丢)。
- RecommendSection 裸 inline style→SectionHead 体系;指路文案随迁
  (「在『语言模型』启动」→「模型一览」)。
- 新增 `ModelCenterPage.tabs.test.tsx` 5 例钉死合并编排契约。

**删除史核对(动手前)**:「详细统计」钮上轮拍板不动 ✓;不复用「模型目录」命名
(v4.502 绘梦退役史)✓;InspectorPanel 已并入运行监测不回提 ✓;外部入口核实均为
裸 `setPage('modelcenter')` 无 tab 参数 ✓;utils.tsx 因外部共享面
(RuntimePage/VoiceSettingsPanel/ChatPanel)不拆 ✓。

## 刀二:小说板块瘦身·大道至简(56b21897)

**现状盘底**:六页签(书架/设定/角色/创作/阅读/原罪),页面 5312 行+组件 ~24700 行;
复杂度断层 CreatePage(1568 行/67 态/18 弹层/4 横幅),次 ChapterPage(1210 行);
上轮 v4.470/471 五刀只做工具轨表面收纳,深层结构未动;CSS 卫生良好(死类≈0)。

**刀法(页签 6→5 + 创作页收敛)**:

- **角色页并入设定**:新薄壳 `pages/SettingHub.tsx`(Segmented「世界观/人物」+
  双 pane 常驻 display 切换 + `active = hubActive && 当前视图` 转发保 Ctrl+S
  窗口级门控链 + localStorage `gaea.novel.settingView` 记忆 + 旧值 `'character'`
  迁移直达人物视图)。CharacterPage/NovelSettingPage 组件零改动只挪挂载。
  终态心智:书架/设定/创作/阅读/原罪 = 入口/资料/生成/精修/另玩。
- **创作页四横幅→一条写作状态条**(`WritingStatusBar`):error 恒居首不被折叠
  吞掉、「N 条反馈」展开全部、**全部条目恒在 DOM 保 testid**——CreatePage.test
  45 例行为锁零改动实锤。
- **67 useState 分域拆四 hook**(`pages/create/`:useCreatePanels 15 弹层开关/
  useCreateBanners 5 反馈源/useCreateWizard 向导族/useCreateGraph 脑图族),
  页面同名解构=引用点零改动。

## 两刀通用配方(可复用)

1. **合并前必查事件面/入口面**:grep 跳转事件发射方与 setPage 调用,零带参
   跳转=合并改面趋零(novel:goto-tab 全书只有 chapter/create 两类目标)。
2. **拆件保测试绿**:aria-label/按钮文案/testid 逐字保留,行为锁测试零改动
   即等价证明;「条目恒在 DOM + CSS 控制可见性」是收纳类改造的 testid 契约手法。
3. **类型收窄用编译器兜底**:导航值域先改类型定义,让 tsc 暴露全部比较点,
   优于人肉 grep。
4. **纯态拆 hook 用同名解构**:域 hook 返回与原 useState 同名字段,页面全部
   引用点零改动。

## 坑清单

- **setter 出 useState 后 exhaustive-deps 豁免失效**:域 hook 返回的 setter 被
  react-hooks 插件视为不稳定,13 处补 deps(setter 实际恒稳,行为零变)。
- **CSS 类删除先 grep 全消费面**:`novel-create-alert` 以为死类删除,
  ForeshadowDueBanner 仍在用,当场复原。
- **迁移断言方向**:旧 tab 值迁移写 settingView=chars 后激活的是人物子 pane
  (pane-character=true),世界观 pane=false——直觉会写反。
- **测试过≠语义对**:NovelPage.test 在 6→5 后「碰巧」仍绿(真 SettingHub 穿透
  渲染两个 mock 子页,pane 计数巧合不变),已随迁注释+补 hub 转发链/旧值迁移两例。
- react-refresh 组件文件禁导出函数(第四次命中):`customFormError` 无外部消费
  改模块私有即消。
- 测试 getByText 撞名收窄:模型中心「功能绑定」tab label 与 rail-foot 摘要标题
  天然撞名,须 `{ selector: '.mc-rail-label' }`。

## 余池(未随本批)

- ~~真机目检:模型中心四节标视觉/「模型一览」三段纵长体感;小说 SettingHub 子导航
  视觉/状态条展开手感~~ → **2026-10-28 已走查**(与首页线同场,见下「真机走查」节):
  模型中心/小说 SettingHub/首页工具链/运行监测 HUD 全过;唯一遗留=创作页状态条
  「N 条反馈」展开手感需实弹生成流(渲染条件=有反馈条目,干净新书返回 null 属设计)。
- ~~小说:创作页 18 弹层交互与阅读页双 chrome~~ → **2026-10-28 续刀**(见文末)。
  盘底结论:阅读页双 chrome(ReadingChrome/EditChrome)实为已收敛态——单行 chrome+
  弹层组,无可收面,候选销号;创作页弹层面 19 个均为「菜单一次点击可达」的独立
  功能面,合面板内容(如对标拆解+技法库)收益小语义风险大,不做;真余量在状态族。
- 模型中心:CreatePage 同款深拆不在册;CreatePage 剩 ~24 态(章节正文域 10/生成流域 13
  与生成流逻辑纠缠,拆之无益——同族已尽)。
- 发版:随池另拍(本批 30 提交含首页方向一重设计/运行监测 HUD/工具化线等三线)。

## 真机走查(2026-10-28,沙箱壳=重建后桌面 exe,GAEA_WALKTHROUGH 隔离核验过)

- **首页线**:工具下拉落屏内(§8.7 缺陷未回潮)+CAD/进度 ToolSheet 双开合+示例工程
  载入(15 项/141 天/横道 57 SVG)全过;最近对象组实时记账(开书即上首页)。
- **模型中心**:rail 9 分类四节组;「模型一览」三段(语言模型/专项模型卡阵/选型参考)、
  「测评中心」两段(受控测评/检索质量测评)、「策略与成本」两段(空间策略/成本归因)
  纵长滚览全过。
- **小说**:5 页签;SettingHub 世界观/人物双 pane display 切换+`gaea.novel.settingView`
  持久化;书架登录态完整形态(正在编辑横条/书卡封套);创作页三栏。**状态条三项
  testid 仅在有反馈条目时渲染**(items.length===0 return null=设计),展开手感留实弹池。
- **运行监测 HUD 轮四**(上轮 SSE 过代理崩 vite 未竟,本轮真壳补上):LIVE 徽标/时钟/
  编号节标 01-04/FUI 角标引擎卡/按模型明细空态(轮三 nil 兜底未回潮)/最近请求终端
  化表头;「按模型明细→」跳钮确认已撤(反断言 0)。亮色档同场目检(首页/聊天/运行
  监测三面暖纸形态正常)。
- **走查修一刀**:硬件遥测「内存 31.8/68.340748288 GB」分母裸浮点——RuntimePage
  ramTotal/vramTotal 补 toFixed(1),测试 '12.2/96' 随迁 '12.2/96.0'。
- **走查工艺新坑**(入册):①CDP 反复 reload 会撞**冷启动 BootSplash 静态屏**——
  静态屏长得和真首页一模一样且 window.go 已注入,`document.body.textContent.length`
  停在 ~1232 即静态屏信号,断言前必须等真 app 就绪(如 `.w-boards` 胶囊数>5 轮询);
  ②`window.go.app.App` 对象**自定义属性写不进读不回**(方法覆写可以),stub 旗标要挂
  window 侧;③走查壳登录态=沙箱 home 的 `.gaea_token.json`(第三数据面,非 appdata),
  明文合成 token(expires_in:0)即可过 EnsureToken 零网络;④页内 location.reload 保
  Wails 桥,CDP Page.reload 后 evaluate 可能打到旧执行上下文残影。

## 续刀(2026-10-28):创作页状态族清账 + 质检菜单分组

用户「继续大道至简」。盘底(见余池销号注):双 chrome 已收敛无面可收;19 弹层面
合并内容不做;真余量=状态族与菜单 IA。两刀:

- **质检菜单 8 项平铺→三组**(全书评估: 全书体检/平台评审/文风指纹;
  写法参考: 故事骨架/对标拆解/技法库;本章工具: 收敛修补/上下文清单)——
  antd Menu `type:'group'`,组名即导览,成员/key/handler 零变化;测试从「6 项在册」
  升级为「8 成员+3 组名」全量钉死(旧断言漏了 2 成员,顺带补齐)。
- **状态族两域出页,域 hook 4→6**:
  `useCreateNarrativeOps`(叙事账本 loadState/proposeState/approveState+去味
  deslopChapter/llmDeslop+AI 反推 reconstructOutlines 族——三者共用 rail 消息通道
  stateMsg/stateBusy 故同族;7 态+6 动作+反推轮询/取消/`novel:auto-reconstruct`
  串联事件,`active` 经参数入 hook 走本地 ref 镜像)与
  `useCreateFingerprintReview`(指纹 fp*4+评审 review*5+5 动作,纯自包含)。
  CreatePage 1467→1258 行/40→24 态。
- **B5a 切书失效下沉**:评审/指纹的切书复位内聚为 hook 的 projectPath effect
  (原 CreatePage 切书 effect 内联语句逐字迁入,挂载首跑全为初值=无操作等价),
  页面切书 effect 只管正文/在途生成;换平台弃报告收为 `changeReviewPlatform`。
- 等价证明=CreatePage.test 43/43 **零编辑**复绿(行为锁含 B5a/菜单成员/去味禁用);
  新增断言仅菜单三组名。坑复训:hook 内事件处理器读 `active` 须本地 ref 镜像
  (闭包捕获 prop 滞后一拍);`useXxx` 引用必须先于解构使用点,切书 effect 引用
  后置 hook 的返回值会吃 TS2448——复位职责索性整体下沉,别让页面当二传手。
- **NovelPage pane 断言竞态根修**(全量并发下两连失败,单跑恒绿——按纪律不盖
  flaky 章):「默认书架 tab」用例只 findBy pane-home 一个惰性 chunk,其余四 pane
  **同步断言**——负载下设定 hub chunk 慢半拍即炸(「全量并发超时单跑绿」家族的
  潜伏形态,这次凑齐两连败证据链)。根修=等 pane 计数到 5 再断言,等待语义增强、
  断言零变化。
