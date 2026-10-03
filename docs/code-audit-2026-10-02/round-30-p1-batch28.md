# 全仓审计第 28 批 · god-file 续拆（IN1-01 project.go 五分：核心/固定资产/章节/迁移/逐章存储）· 2026-10-03

> 接续 [round-29（批次二十七）](round-29-p1-batch27.md)。god-file 同配方单线，**同包纯文件拆分**（原位搬移零逻辑改动），主代理直做。预核判定：虽是核心 Manager，但**同包文件分家对消费方零影响**（符号仍包内可见、API 不动）——与「接口分域两步走」相比，本批先做零风险的第一步。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `0492001f`（批次二十七）；工作树干净。

---

## 〇、预核与切口

- IN1-01 审计指认复核成立且**已恶化**：1011 → 1194 行（审计后继续增长）。八类关注点中，本包已有 plan_store.go/story_spine_store.go 分家先例——Manager 方法按域落兄弟文件是本包已确立的组织形态。
- 五文件方案（切口=文件自带分节注释+域起止）：
  - `project.go`（~222）：头+Manager 结构+生命周期（Create/Open/Close/WriteMeta）+上下文构建（LoadContext/findNode/findParentVolume）。
  - `project_files.go`（~229）：固定资产读写——世界观（md/json 双形态）/角色/大纲/伏笔/风格指纹·摘要·StyleProfile（含 legacyBrandDir 同源）/Lorebook。
  - `project_chapters.go`（~348）：章节正文主线+branch 变体、章节摘要族（mainline 口径/before 查询）、MaxChapter 探测、ParseChapterFileName 单源（批 21 保留站点口径对照表随行）。
  - `project_migrate.go`（~160）：writeJSON/loadJSON 泛型原子读写（全包共用，落点纯属组织性——头注写明）、DefaultSections、IsV4、MigrateV3ToV4+finalize。
  - `project_stores.go`（~290）：SceneManager/SnapshotStore 子存储、ReadChapterAsStitch/ForEachChapter、AnalysisV2/StoryMemory/RewriteVersion 族（含索引同步）/Annotation。

## 一、落地与等价证明

- sed 行号字节精确抽取 + 手写文件头拼接；行数对账闭合（1170 内容行 + 5 seam 空行 + 19 旧头 = 1194）。build/vet 一次过（别名感知的两轮 import 普查：json/types/slog/fileutil/scene/snapshot/sync/filepath 按段重算）。
- **双证零漂移**：符号清单 before/after diff 空（**对称口径教训复训：两侧 glob 都要含测试文件**，首轮 after 侧漏 `*_test.go` 出现 27 行假删除）+ `go doc -all` 逐字节一致（project.go 无包级 doc 注释，无批 27 的 doc 复位问题）。
- 包测试：project 0.6s 绿（章节摘要/迁移 fail-closed/RewriteVersion/ParseChapterFileName 等价钉全走公共入口，直接覆盖搬移代码）。
- 前台 ci **exit 0**：golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例全绿 / 卫生守卫绿。
- 新文件四枚进显式 add 清单；提交后复验「树干净 + HEAD 可编译」。

## 二、god-file 余量与下一批

- 已销：GA2-01、IN2-12、GA5-08、GA5-04、IN3-03、IN1-01。god-file 余 ~15，重心转入**前端组件族**（CostLibraryView 1132 行×2 条/CostProjectsView 862/controller.ts 963/MockOffice 81 mock/App.tsx 1041/ModuleLauncher 1038）——组件拆分=状态上提+prop 穿线，与 Go 文件分家不同型，需先立前端拆分配方（小组件试水→金样 DOM 断言先行）。
- 下一批候选：前端试水（FE6-07 ModuleLauncher 或 FE2-04 CostProjectsView）或 Go 侧续拆（GA4-02 controller.go 上帝对象 40 字段/GA1-03 agent.go 975 行）。coupling ~20 与零散死码继续等拍板。
