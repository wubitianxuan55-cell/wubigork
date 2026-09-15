# gaea t4-C3 收官 · 场景工程整章重写 · 实施规格（v4.322.0）

> 来源：todos 小说线「C3 余项=场景工程」（v4.304 首刀范围注记「v4 场景工程与
> partial 下刀放开」，partial 已于 v4.321 落地）；单线小刀主代理直做。
> **产品裁决（已拍板=单场景替换）**：整章重写产出的是整章新全文，与原场景边界
> 无法自动对齐——重写整章本就意味着重划结构，应用=新全文写回该章唯一场景
> （保留首场景 id/title，其余删除）；Restore 同语义（原文写回唯一场景）。
> LLM 智能分场景回写与场景粒度重写入观察池。
> **partial 对场景章仍拒绝**（选区拼接回写会破坏结构且无法对齐，V1 不做）。

## 0. 论点

NovelChapterRewrite 现对 v4 场景章直接拒绝（「整章重写暂未开放」）——场景工程
项目整章重写不可达。根因=版本库 Apply/Restore 只写 blob，而 v4 的真源是
scenes/（scene→blob 单向同步 `syncBlobFromScenes`）。本刀补 v4 感知的写回：
**单场景替换**（ReplaceAll）+ blob 同步，重写/应用/恢复全链对场景章闭环。
**零新绑定**（RewriteModal/RewriteHistoryPanel 全 chapterNum 泛型，696 不动）。

## 1. Go 三处

### ① internal/scene/scene.go：`ReplaceAll(content string) error`

- List 按序；0 个 → Create（slug="whole"，title="整章"）后 Write content；
  ≥1 个 → 首场景 Write(content)（保留 id/title/meta），其余逐个 Delete。
- 原子性：逐场景操作非事务——中断残留多场景属可恢复脏态（blob 未同步前
  stitch 仍读旧态），文档注明；调用方紧随 syncBlobFromScenes。
- 测试：0/1/3 场景三态（剩 1 个、内容=新全文、顺序保留首场景、其余目录清除）。

### ② internal/app/chapter_handler.go：`writeChapterV4Aware(pm, chapterNum, content) error`

- pm.IsV4() 且 SceneManager.List() 非空 → sm.ReplaceAll(content) +
  syncBlobFromScenes；否则 WriteChapter（v3/v4 无场景章原路径）。
- 放 syncBlobFromScenes 旁（同文件内聚）。

### ③ internal/app/novel_rewrite_handler.go

- **移除** whole 入口的 v4 拒绝段（L43-48）；original 改 `ReadChapterAsStitch`
  （v3 走 fallback 零变化）。
- NovelApplyRewriteVersion / NovelRestoreRewriteVersion：`WriteChapter` →
  `writeChapterV4Aware`（各一行；partial 的 Apply 同享——见下）。
- partial 方法开头加守卫：pm.IsV4() 且场景非空 → 报「场景工程章暂不支持局部
  重写（选段与场景边界无法对齐）」。
- 测试：①v4 场景章 whole 往返（多场景 fixture→重写→版本快照=stitch→Apply→
  场景收敛为 1 且内容=新全文+blob 同步=ReadChapter 读到新文）②Restore 回原文
  ③partial 对 v4 拒绝④v3 章回归（既有用例不动即证）。

## 2. 前端一处：ChapterPage（v4 编辑面）

- sceneBacked 活动页工具栏加「整章重写」「重写历史」两按钮（非 sceneBacked
  章已有 CreatePage 入口，不重复挂）。
- 挂 RewriteModal + RewriteHistoryPanel（chapterNum=activeTab.chapterNum；
  onApplied → loadChapterIntoTab 重载该 tab——按现有 tab key 机制调用）。
- 测试：ChapterPage.test 增例——sceneBacked tab 渲染两按钮；v3 tab 不渲染。

## 3. 收口（v4.322.0）

定向（scene/app/rewrite go + ChapterPage/ChapterPage 单测 + RewriteModal 既有）
→ tsc/eslint → ci.ps1 恰一次 → 版本三处（**含 app_info.go 入 commit**）→
build/SUMS/桌面副本/冒烟 → releases/v4.322.0.md+CHANGELOG/README×2 → AGENTS
迁 1 插 1（三十七迁）+progress/todos → commit+tag。drift 复核 696。

## 4. 足迹

| 文件 | 动作 |
|---|---|
| internal/scene/scene.go(+scene_test 增例) | ReplaceAll |
| internal/app/chapter_handler.go(+test 增例) | writeChapterV4Aware |
| internal/app/novel_rewrite_handler.go(+test 增例) | 拒绝移除/stitch 读/Apply·Restore 换写/partial 守卫 |
| frontend/src/pages/ChapterPage.tsx(+test 增例) | 两按钮+两弹窗挂载 |
| 主 | 规格书/门禁/版本/releases/CHANGELOG/README/.gaea |

## 5. 出口对照与观察池

- 判据：v4 场景章整章重写全链可达（重写→版本→应用=单场景替换→恢复原文）✅
  测试钉死；C3 余项**全清**（whole v4.304/partial v4.321/历史面板 v4.320/场景 v4.322）。
- 观察池：LLM 智能分场景回写；场景粒度重写（对齐去味逐场景先例）；partial
  对场景章的选段→场景映射。
