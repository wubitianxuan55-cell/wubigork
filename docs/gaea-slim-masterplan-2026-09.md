# gaea 瘦身长期总规划（2026-09-08 立项，权威）

> **状态：🔄 活跃权威（长期）**——瘦身「瘦什么/怎么瘦/瘦到哪」的唯一权威。
> 治理规则（周度 release train / 拍板池过期 / 零功能周 / 不做清单 / 拍板结论）仍以
> `gaea-convergence-plan-2026-09.md` 为准；壳内点位底稿见 `webview2-shell-audit-2026-09.md`；
> 三者关系：收敛计划=治理层，本文=瘦身执行层，审计=证据层。
>
> **总原则（四条，全程不变）**：①零功能损失（功能等价验收表）；②藏≠删、合≠砍、懒≠去；
> ③先测后动、每刀带前后数字；④引擎零改动（调度/造价/文档引擎在形态合并中一律不动）。
> 前提拍板：路线 A=终极个人工具 · LICENSE=私有 All Rights Reserved。
> **拍板（2026-09-08，用户纠正）：工位与乐园是并列平级关系**——办公不是乐园的上位核心，
> 瘦身不是「收乐园保办公」，而是让双空间并列结构在 UI 上成立。认知面问题重定义为：
> **13 板块平铺单一导航面，并列结构未被表达**。
>
> **执行进度（2026-09-09）**：P0-P4 全部收官（v4.166~v4.177）；v4.179.0 清点刀=
> 轨道二 knip 落地（死文件 13 清零+幽灵依赖 13 包显式化；Unused exports 169 挂账）+
> 轨道五冷启动基线打点就位（Go 七段+前端两 rAF 口径，数据随日常启动积累）+
> 初始化链审计结案（重活均已异步，懒初始化无需立项——轨道五余项收敛为真机采集）。

> **巨文件四件套复核（2026-09-10 第六会话）**：轨道三「9 个 >50KB 巨文件」
> 现状已变——W3-4 四件套实质完成：bridge.ts 已拆为 lib/bridge/*（index 仅
> 1.4KB）、GanttView 25KB（aoaRuler 先例拆分）、config.go 16.5KB、App.tsx
> 降至 47.8KB。当前 >50KB 源文件仅剩：locales en/zh-TW/zh 三件（纯数据，
> 拆分=低价值churn，挂起）+ gaea/lib/store/controller.ts 51KB +
> gaea/control/controller.go 51KB（两件新挂账，拆分刀序待排）。

## 0. 臃肿全景（七面体检，2026-09-08 实测）

| # | 面 | 现状实测 | 目标态（6 周） | 长期态 |
|---|---|---|---|---|
| 一 | 认知 | 13 一级板块平铺单一导航面（work5/play4/shared3/indep1），**双空间并列结构未表达**（拍板：工位/乐园平级，非核心+附属） | 双空间切换一级可见、各自导航：工位 ≤6（schedule 并入办公文档面）、乐园 ≤4，shared/independent 归各自壳层 | 每空间板块数只减不增；新域先问「属于哪个空间、能否是文档」 |
| 二 | 资产 | dist 10.03MB/224 chunk；entry 1.19MB；MemoryHub chunk 1.4MB | entry 与 MemoryHub 双拆；locale 死键清零 | 首屏恒 ≤ 靶值；重库恒按需 |
| 三 | 结构 | office 包 272 入/70 出枢纽；9 个 >50KB 巨文件 | 抽公共内核；巨文件首批 4 个拆掉 | >50KB 源文件 ≤3；枢纽入度 <100 |
| 四 | 轮子 | title-slug×5、b64 解码×18、行 diff 漏网×1、下载出口×5、选取点×8、双轨 bridge | 重复清单清零或归入保留清单 | 防复发规约+eslint 门禁 |
| 五 | 运行 | 冷启动/常驻内存**无基线**；app.New() 全模块初始化 | 基线+懒初始化改造；可交互 -30% | 季度复测防回归 |
| 六 | 产物 | exe 46.2MB；npm lock 701 包（运行时直依赖 26） | 发布版 strip 实测；依赖逐个验活 | exe ≤40MB；直依赖零死重 |
| 七 | 数据/知识 | progress.md 244KB；持久化四范式并存；clones/ 占仓目录 | 知识外化三件套；范式观察池立案 | 文档/数据总量与信噪比纳入季度审计 |

## 1. 七条轨道（内容权威）

### 轨道一 · 认知面：信息架构

1. **schedule→办公文档化（三刀，已论证）**：
   - 刀0（前置小刀）：导航白名单与 `inMenu` 解耦——现状 `filter(inMenu && !isHome)` 派生白名单
     （manifests.ts:162-164），藏板块会静默丢导航（v4.164 同款坑）。改为注册即可导航，
     `inMenu` 只控菜单。
   - 刀1：schedule `inMenu:false`（settings 先例）+ 办公「进度计划」入口（最近工程列表+
     打开工作台+命令面板项）。引擎零改动。
   - 刀2：`.gsched` 注册进办公文件面（最近文档可见；FilePreview 摘要卡：任务数/总工期/
     关键路径/基线数+「打开工作台」）；`.mpp` 走既有「导入为新工程」不做内嵌预览。
   - 刀3（观察池）：工作台全幅内嵌办公。刀1/2 用一周再拍板。
2. **双空间并列落地（非折叠）**：chat/novel/imagegen/characterlib 在**乐园空间内一级可见**
   （不是藏起来），manifests `space:'play'` 已就绪；要做的是空间切换器与分域导航，让
   工位/乐园平级共存于同一壳。
3. **home 空间感知化**：按当前空间给对应工作台——工位=最近工程/文档/待办/模板；
   乐园=最近会话/小说/创作；不再用单一首页平铺 13 板块。
4. **cost 候选（远期观察）**：schedule 落稳一版后再议「造价工程文件化」，一次只动一个域。

### 轨道二 · 资产面：前端体积

- entry 1.19MB 拆解（antd/react/bridge/types/共享组件中非首屏必需下放）；
- MemoryHubPage 1.4MB chunk 解剖：3d-force-graph/cytoscape/cynefin 疑似同包——按 tab 动态 import；
- 确认 mermaid(607KB)/katex(255KB)/CodeEditor(310KB) 仅宿主视图加载；
- locale 死键清理（en+zh-TW 188KB 源，i18n 已拍板内容层 zh-only）；
- 26 个运行时直依赖逐个验活（gsap/docx-preview/unist-util-visit 存疑名单）+ knip 扫 unused exports。

### 轨道三 · 结构面：架构止血

- **office 抽核**：272 入调用归类（whisper→office 96、modelengine→office 62），journal/
  evidence/文件读写抽 `internal/core`，三板块平级依赖——此刀降低后续每刀碰撞面，结构刀中优先；
- **巨文件拆分池（9→≤3）**：首批 App.tsx(85KB)/bridge.ts(85KB)/types.ts(66KB)/GanttView.tsx(65KB)；
  次批 config.go(59KB)/KnowledgePanel(62KB)/DeliverablesPanel(57KB)/XlsxPreview(56KB)/ChapterPage(56KB)；
- 拆法沿用先例：bridge 按 bindingNames 域切、GanttView 三分（aoaRuler 先例）、config 同包多文件。

### 轨道四 · 轮子面：重复收敛

| 类 | 项 | 处置 |
|---|---|---|
| 真重复 | title-slug Go×5（cost.SlugName/knowledgeimport.slugName/app.slugFromTitle/modelengine.customSlug/controller_memory.slugifyName） | 并 `strutil.TitleSlug`；schedule.SafeSlugName（文件名安全）与 memory.slugify（路径）**语义不同不并** |
| 真重复 | b64→bytes×18 处 | `b64ToBytes()` util 全收 |
| 真重复 | novel DiffReview 手搓行 diff（弱贪心算法） | 迁 `lib/diff.ts` LCS（收口即提质） |
| 真重复 | 下载出口×5 / PickFiles 调用点×8 / inShell×2 | 双门收口：进=pickFile.ts、出=saveExportBlob（刀B/C 主体） |
| 双轨 | bridge.ts 手工路由(602) vs wailsjs 生成门面 | 渐进退役 wailsjsCompat（跨版本） |
| 评估池 | XlsxPreview 手搓虚拟滚动 vs react-window（已在依赖） | 冻结行/spacer 特殊需求评估后定，不盲迁 |
| 保留（写明理由） | CPM Go/TS 镜像（golden 对拍）/ imgPdf 手搓 PDF（避重依赖）/ mock 后端面复刻（服务测试与走查） | 只补 tradeoff 注释 |

### 轨道五 · 运行面：启动与内存

- 基线：冷启动两档（首帧/可交互）+ 常驻内存 + 首屏 chunk 网络清单；
- `app.New()` 初始化链审计：prompt 引擎/Herdsman 探测/whisper DB/模型中心——重活首访懒启+骨架占位；
- 壳内行为债（审计刀B/C/D）随阶段 1 收尾，属同一批文件面。

### 轨道六 · 产物面：二进制与依赖

- 发布版 `-ldflags "-s -w"` 实测（dev 保留符号）；upx 记录不做（杀软误报）；
- go.mod 逐项验活；npm 已 lock（v4.164），nanoid high 升级刀排队。

### 轨道七 · 数据与知识面

- 知识外化三件套（收敛计划 W2）：30 分钟上手文档 / progress.md 244KB 瘦身归档 / 个人路径清洗；
- 持久化范式观察池：SQLite(facts)/JSON(工程)/JSONL(journal)/index 缓存四范式——不合并，立案防第五种诞生；
- 磁盘：clones/（蒸馏参考库）移出仓库目录；.gaea/work、backups 容量策略（日志轮转 v4.163 先例）。

## 2. 阶段计划（周度 train，每阶段≈1 版，共 6 版+长期维持）

| 阶段 | 版本预算 | 内容 | 出口判据 |
|---|---|---|---|
| **P0 基线** | 0.5 天（随 P1 版附带） | 七面基线四表+功能等价快照表（13 板块×核心动作可达性）+ IA 现状走查（rail 实际显示/默认空间/首页陈列） | 基线数字入档，P2-P4 靶值据此定 |
| **P1 快赢** | 2 版 | 轮子 W1（b64×18+inShell 合一）/W2（slug×5）/W3（novel diff）+ 审计刀B/C + locale 死键 + 依赖验活 | 重复清单清零；vitest/Go 全绿零改动 |
| **P2 形态** | 2 版 | 白名单解耦刀0→schedule 合并刀1/刀2→双空间并列落地（切换器+分域导航）→home 空间感知化→三路可达走查（壳内真机） | 工位 ≤6 且乐园 ≤4 一级导航；等价快照表全绿 |
| **P3 结构** | 2 版 | office 抽核 + 巨文件首批（App/bridge/types/GanttView）+ bridge 双轨退役启动 | 绑定零变更 drift PASS；>50KB 文件 9→5 |
| **P4 性能** | 1 版 | 懒初始化 + exe strip + MemoryHub chunk 拆解 + entry 拆解 | 可交互 -30%；exe ≤40MB（以 P0 实测定靶） |
| **P5 维持** | 长期 | 季度复测七面；观察池审判（cost 文档化/虚拟滚动迁移/工作台内嵌）；零功能周执行 | 度量看板不劣化 |

> 排序依据：P1 全是低风险机械刀先热身并清最痛的重复；P2 是用户感知最大的一刀但依赖刀0 前置；
> P3 碰骨架放中段（前面刀已把碰撞面压小）；P4 必须有 P0 基线才能对靶。

## 3. 度量看板（每版六行，进发布说明）

```
每空间一级板块数（工位/乐园） ｜ entry chunk(gz KB) ｜ 冷启动可交互(ms) ｜ 常驻内存(MB) ｜ exe(MB) ｜ >50KB 源文件数
```

### P5 季度复测（2026-09-25，距 P4 收官基线 v4.177 六个版本季）

| 面 | v4.177 基线 | v4.412 实测 | Δ | 判读 |
|----|----|----|----|------|
| entry chunk | 755.48KB | 969.67KB | +214KB（+28%） | 无单点元凶：react-dom 129KB/zh locale 56KB 不变，增量=ModuleLauncher/MainLayout/TaskInboxPanel/spaceBindings 等新面板长尾+antd form/tabs 族——245 版功能 proportional，**不立项刀** |
| dist/assets 全量 | 10.0MB | 11.1MB | +1.1MB | 大 chunk 均懒加载面（GraphView 1.38MB=GaeaPage 内 three.js、mermaid 640KB、cynefin 690KB 独立 chunk 按需） |
| exe | 46.15MB | 51.44MB | +5.3MB（+11%） | Go 侧 245 版功能增重（whisper 死码刀 9.9k 行已削）；无嵌入资产异常 |
| Go >50KB 源文件 | 0 | 4（image_comfyui 58K/image_handler 55K/controller 51.7K/config 50.9K；另 >40KB 共 9） | 回升 | T3 编辑力/角色资产 v2/sin 看板连发所致；拆分留观察池勿顺手做（P3 纪律=专项） |
| 冷启动/常驻内存 | 未采 | Go 同步链 43–110ms（09-24/25 十二次自然启动样本，中位 ~62ms；大头 charlib+stores 40–47ms）；**前端 gaea:interactive=497ms**（v4.413.0 壳 CDP 实测，gaea:boot 91ms→interactive 588ms，DCL 139ms） | — | 常驻内存仍挂真机窗口 |

判读：回涨全部属功能比例（每版均值 entry +0.9KB/exe +22KB），P4 懒加载结构未破防（页面级 lazy 完整、entry 无重依赖回归）；唯 Go >50KB 0→4 破防，挂观察池待专项。下季复测随 P5 维持。

## 4. 防复发规约（P1 起写入 AGENTS，P2 起上 eslint 门禁）

1. 新下载出口一律 `saveExportBlob`（eslint 自定义规则禁裸 `a.download`，豁免=exportArtifact 回退层本体）；
2. 新文件选取一律 `pickFile.ts`（壳内/浏览器双路径免费）；
3. 新文本 diff 一律复用 `lib/diff.ts`（novel 漏网教训）；
4. 新 slug 一律 `strutil.TitleSlug`；
5. 新 base64 解码一律 `b64ToBytes()`（eslint 禁 `Uint8Array.from(atob`）；
6. NAVIGATE 载荷一律 manifest 板块 id（v4.164 规约）；
7. 新板块冻结：新域先问「能否是某板块的文档/子面」，答案为否才立板块。

## 5. 红线与风险

| 风险 | 对策 |
|---|---|
| 白名单解耦漏做导致导航静默失效 | 刀0 是 P2 第一刀，先测后藏（v4.164 前科） |
| lazy/隐藏在 Wails 壳内行为异常 | P2 全量壳内真机走查（v4.162 前科：浏览器全绿壳内失效） |
| 形态合并伤引擎 | 引擎零改动铁律；schedule/cost 引擎文件在合并 PR 中禁触 |
| 虚拟滚动/react-window 盲迁 | 评估池制度：先写评估记录再动 |
| 瘦身变功能退化 | 功能等价快照表每版复验；测试数只增不减 |

## 6. 观察池（评估后再定，默认不做）

工作台全幅内嵌（轨道一刀3）· cost 文档化 · XlsxPreview 虚拟滚动迁移 · 持久化范式合并 ·
模型中心/青鸟归属调整 · upx 压缩。

## 7. 与既有文档关系

- `gaea-convergence-plan-2026-09.md`：治理层（节奏/拍板/不做清单）——其 §3 架构止血已并入本文轨道三，§5.2 模块可藏并入轨道一；
- `webview2-shell-audit-2026-09.md`：证据层（刀B/C/D 随 P1 执行）；
- `.gaea/todos.md`：本文阶段项的执行账。
