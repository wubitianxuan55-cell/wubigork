# gaea 未来展望（迭代方向，2026-09-09）

> **状态：🔄 活跃权威** — 一年后打开时的样子，不是欠账清单。
>
> 选题顺序 = `docs/gaea-outlook-longterm-plan-2026-09.md`。
> 路线 A=终极个人工具。书斋 / 闲庭硬隔离（壳层旧称工位 / 乐园）。
> **拍板**：模型中心是云端×本地混合的一级节点，不降为设置。
>
> 下文「机器现在」只写本轮读过的主路径。没打开的文件不写进结论。

---

## 一年后的 gaea

一台电脑里两间房，共用一个模型总闸。

它不是「中国版 Copilot + 猫箱 + 广联达」，也不是又一个板块全能工作站。它是本机上把一场活干完的管家，外加两样市场不卖的东西：你自己的造价记忆，和一间绝不提工地的闲庭。

你带着活进来，不带着板块进来。书斋里你开口，它取本机资料、改文件、算造价、出进度图——问、读、算、起草当场做，工具自己串。闲庭里你写、画、聊，它记得角色和你们的关系。编程在隔壁自己的房间。微信和语音是门铃：在别的地方喊一声，还是这个管家出来干。

云端和本地不是两种产品。这句话走本机，下一句走云，在模型中心一眼看见、当场切，书斋和闲庭可以各绑各的。

迭代只朝这个画面收。新能力问：它是让「带着活来」更真，还是又多了一个要先点进去的房间。

---

## 机器现在（主路径，读过才写）

Wails 桌面，一个 Go 内核，十个绑定门面。嘴有两张：

- **办公管家** = `GaeaSend` → `control.Controller`（`internal/app/gaea_handler.go`）。内置工具 + ExtraTools（`image_gen` / `diagram` / `ocr` / `semantic_search` / 事实三件）。`cost_search` / `cost_save` / `schedule_get|apply|analyze` 在内置表里。xlsx/docx 的 agent 纪律是技能 `office-edit`：bash + python，不是预览面板那条 Plan→Apply。
- **会客厅** = `ChatPage` 人格流，不是 `GaeaSend`。

空间也是两套，**且拍板（2026-09-10）就是两套，不是过渡态**：

- **壳层书斋/闲庭** = `MainLayout.switchSpace`，只改 `appStore.space` 和页面，不调 `GaeaSpaceActivate`（`frontend/src/layouts/MainLayout.tsx`、`frontend/src/boards/space.ts`）。
- **办公引擎空间** = 配置 `session.space`。`GaeaSpaceActivate` 写配置、**不重建**当前 controller；工具过滤和会话目录在**下次** `gaeaBuildController` 才生效（`internal/app/gaea_spaces.go`）。办公侧栏 `SpaceChip` 才走这条。

落盘门（办公 agent）：`cost_save` / 记忆写入等在 `hardAskTools` 里，yolo 也要确认；`schedule_apply` 的 **project 整量**通道同样强制确认，ops 增量跟权限级别（`internal/gaea/control/controller_approval.go`）。对话本身没有 GoalCard。

组价：`GaeaCostCompose` / `Apply` 是 CostB 绑定，确认才进库（`internal/app/gaea_cost_compose.go`）。本轮未见同名 agent 工具。

进度：`schedule` 板块 `inMenu: false`，页还在；agent 三工具仍注册。`schedule_*` 未进 `spaceTags` 显式 work 表，缺省 shared（`internal/gaea/tool/builtin/spacetags.go`）。

做梦：办公回合结束后 `maybeDreamAfterTurn(gaeaSessionSpace())`，play 的 dream notes 不写 work 的 AGENTS.md（`internal/app/gaea_dream.go`）。这是办公会话空间，不是壳层闲庭。

小说：创作 `CreateChapter` 仍 `WriteChapter` 整章 blob；阅读 `GenerateScene` 走 `scene.Manager` + POV 圣经。`CreateScene` 绑定在、创作/阅读页不加调。`RunChapterGate` 不在 `NovelBindings`（见小说专项读档）。

青鸟：`wxAgentTools` 为切板块 / 生图 / 提醒 / 发已有文件 / 查状态 / 读屏；指令写明不能改文档（`internal/app/wx_agent.go`）。不是 `GaeaSend`。

首页书斋语音：`VoiceChatText` + 人格 `gaea`（`ModuleLauncher.tsx` DeskHome），不是 `GaeaSend`。

模型中心 UI：按功能域 `chat/novel/office/gaea/characterlib/routine` 绑定（`useBindState.ts`）。引擎侧 `SpaceProfile.Gaea` 在 TOML，装配时覆盖办公 agent 模型。不是「两房策略一张画面」。

编程：`space: 'independent'`，不进两房 rail。

市场坐标仍成立：巨头抢第一入口；gaea 站本地把活干完。跟混模总闸，不跟 MCP 广场、跨端、积分云。

---

## 书斋

舞台是办公会话。磁盘上的文件是活本身。

对话没有仪式。Steer 改方向。唯一硬门在磁盘：覆盖真文件、`cost_save`、整计划 `schedule_apply`、造价库确认回写。试算、检索、会话内起草不设门。Journal 是事后保险。落盘形态按施工。

以后往这收：书斋里这场活就是办公管家那场活（`GaeaSend`），壳层切去闲庭不打断它——壳层是导航，引擎空间是办公侧栏另一件事，切换对在跑的活零扰（2026-09-10 拍板：两套拆开，「一年后是一个开关」的画面作废）；造价组价和进度图作为这场活的材料被调用，不必先把造价/进度当成上班打卡点。进度已经够专业，不再养成 Project。造价是唯一加厚的垂直：开口对上你的历史，确认才进库。

记忆：干活时被记得。做梦按办公会话空间分流。中枢若还在，是检修口。

---

## 闲庭

隔离是产品也是结构。会客厅不提工地。

聊天保持人格连续、主动开口极稀。

小说 · 绘梦 · 角色库往「同一资产、文图互生」收。小说不是空房间：场景制、POV 圣经、去味、叙事账本已经写过，主路径还在写 blob。先接通，再谈新家具。没在写、没在画，不加空转功能。

---

## 模型中心

一级节点。混合可见、失败说人话、板块不私藏引擎表。接供应商是中心增量。不折进设置。

书斋敏感可强制本地、闲庭可走更会写的云——配置里已有 `SpaceProfile`，画面还没有。

---

## 隔壁与门铃

编程继续隔壁，工具面不和办公混注册。

青鸟和语音的未来是唤起**同一张办公嘴**（`GaeaSend` 那场活），不是再造一个微信里的 gaea，也不是只切板块、发已有文件。

首页：书斋=最近工程和文件；闲庭=最近会话和创作。设置放家底。模型不放设置。

---

## 迭代时怎么用

往画面靠的，做：

- 壳层书斋/闲庭与办公引擎空间：已拍板两套拆开（2026-09-10）——壳层=导航、引擎空间=办公侧栏另一件事，切换不打断在跑的活；守卫=e-check E26（switchSpace 零桥接）。本条收口，后续只维持不再动
- 让小说创作和阅读共用 `scene.Manager`
- 让对话保持直接；磁盘硬门维持 hardAsk，不铺成每轮计划卡
- 让造价开口能用上历史（组价链在 CostB，进不进管家是施工拍板）
- 让模型中心能看见两房策略
- 让门铃最终能驱动管家，而不是只能 navigate

往画面反的，不做：

- 再开一级房间
- 把办公对话做成审阅制皮肤
- 进度养成 Project、造价养成广联达
- 记忆养成参观图谱
- 模型中心折进设置
- 编程和办公工具倒进一个 agent
- 空闲庭加家具
- 第一入口 / MCP 广场 / 跨端多人积分云

瘦身 P5 是维持，不是主题。真机债、会话身份、落盘形态按施工。未来仍是：带着活来，对话直接干，云和本地一个闸，两间房互不串门。
