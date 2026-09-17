# gaea 小说·刀7续收官：POV 视图（场景圣经消费面）规格书

> 2026-09-17 立项。来源：用户指令「继续」。规格 docs/gaea-novel-revolution-
> 2026-09.md 刀7续未勾项收尾。前置核对：**场景生成按钮已落**（ChapterEditor
> handleAiGenerate 逐场景 GenerateScene+sceneIds 追踪，v4 场景工程线）；
> 全文脑图/文风指纹面板在位——**刀7续仅剩 POV 视图**。本刀合入+勾选对账后
> revolution 只剩刀8（验收演练/race 门禁，CI 侧）。
> 目标版本：v4.330.0。

## 1. 论点

POV 视角掩码（刀2 场景圣经编译器的核心差异化：POVView=POV 已知 /
HiddenFacts=不知情且不得泄露）只在生成链内部消费——用户看不到「这个场景
通过谁的眼睛在看、哪些信息被刻意瞒住了」，视角纪律不可审视。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | +1 绑定 `NovelSceneBibleView(chapterNum, sceneID)`（706→707，NovelB） |
| Q2 | sceneID 非空=该场景编译（SceneManager.Read→CompileSceneBible；不存在报错）；空=整章合成（BuildSceneBibleFromChapter，v3/纯 blob 章也可用） |
| Q3 | 视图 camelCase typed（新绑定纪律）：pov/title/location/timeOfDay/mood/tags + characters[]{name,roleType,status,location,currentState,careerMain,careerSub,items,knownBy} + **povView（已知）/hiddenFacts（不知情）对照为核心区** + foreshadows/memories/timeAnchor/style/thread |
| Q4 | 前端=ChapterEditor 场景操作行「视角」按钮（EyeOutlined，sceneBacked 门控同信息按钮）→ SceneBibleDrawer：POV 头 + 已知（绿）/不知情（红）两栏对照 + 角色卡 + 伏笔/时间锚点/主线折叠 |
| Q5 | 编译读取全部静默降级（novelcontext 契约）——空区段如实空展示，不报错不编造 |

## 3. 落地

- **Go** `internal/app/novel_scene_bible.go`：视图类型+双路编译绑定。
- **前端** `SceneBibleDrawer.tsx`（novel 组件）+ ChapterEditor 按钮 + bridge/
  mock + spaceBindings（锁 529→530）+ bindingNames 再生（707）。
- **revolution 文档勾选对账**：GenerationGate 闭环（v4.326 实落）与刀7续
  （场景生成/脑图/指纹核对已落+本刀 POV）勾选补记。
- 测试：Go（整章路视图形状/未知场景报错）+前端（Drawer 渲染对照区/空态）。

## 4. 观察池（本刀不做）

Render 全文预览（prompt 注入形态）；POV 切换模拟（改 POVCharID 重编译的
沙盒）；多场景圣经并排；隐藏事实来源溯源（哪章埋的）。

## 5. 门禁

go build/vet + 触面包测试、tsc -b、eslint、vitest 全量、drift OK@707、
计数锁 530、ci.ps1、版本三处 4.330.0。
