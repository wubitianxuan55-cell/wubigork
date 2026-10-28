# 板块瘦身双刀纪要:模型中心 + 小说(2026-10-27)

> 用户指令「瘦身/重组模型中心自身」与「瘦身/重组小说板块,大道至简」。两处幅度
> 拍板点(AskUserQuestion)均未获即时应答,按推荐档执行——与既往拍板风格一致
> (实质收敛、简化≠删功能、同批顺手收口)。提交锚点:`43fa62d9`(模型中心)/
> `56b21897`(小说),随本批 30 提交整体推送 origin,**未发版**(发版随池另拍)。

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

- 真机目检:模型中心四节标视觉/「模型一览」三段纵长体感;小说 SettingHub 子导航
  视觉/状态条展开手感(可与首页线走查同场)。
- 模型中心:CreatePage 同款深拆不在册;小说:创作页 18 弹层交互与阅读页双 chrome
  未动(下轮候选)、CreatePage 剩 ~41 态因与生成流逻辑纠缠未拆。
- 发版:随池另拍(本批 30 提交含首页方向一重设计/运行监测 HUD/工具化线等三线)。
