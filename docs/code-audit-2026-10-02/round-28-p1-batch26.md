# 全仓审计第 26 批 · god-file 续拆（GA5-08 browser manager 四分 + GA5-04 fold.go 耗时域分家）· 2026-10-03

> 接续 [round-27（批次二十五）](round-27-p1-batch25.md)。god-file 同配方续拆两线，均为**同包纯文件拆分**（原位搬移零逻辑改动），主代理直做。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `486959f7`（批次二十五）；工作树干净。

---

## 〇、选点与预核

- **GA5-08** browser/manager.go 1068 行：文件自带四段分节注释（核心生命周期/多标签页/页面操作/Runtime.evaluate 与 JS 片段），切口是文件自己声明的——最低风险的 god-file。测试网：manager_test/browser_test/screenshot_test（+live 门控）。
- **GA5-04** contextview/fold.go 1041 行：耗时折叠域（:707-873）自带域头注且 **timing_test.go 是专属测试文件**——分家后测试与实现同域对位。测试网：fold_test 668 行 + timing_test 265 行。

## 一、线 1 · GA5-08 manager.go → 四文件（1068 → ~320+196+370+196）

- `manager.go`（~320）：核心生命周期——语义化错误/Options/NewManager/idleTTLFromEnv/Ensure/ensureRealLocked/attachLocked/teardownLocked/Shutdown/ClosePage/startIdleWatcher。
- `manager_tabs.go`（196）：多标签页——TabInfo/ListTabs/NewTab/SwitchTab/CloseTab/findPageTarget/ValidateURL。
- `manager_actions.go`（370）：页面操作——Navigate/Read/Snapshot/Click/Type/Scroll + frame 变体 + resolveTarget/guardEpoch。
- `manager_evaluate.go`（196）：Runtime.evaluate 封装（evaluate/evaluateIn/okField/jsErr/jsString/boolJS）+ 六段页面 JS 常量。
- **import 图谱靠两轮 grep**（别名调用 `url.`/`atomic.`/`json.` 与注释内包名都要分清）：evaluate 段首版误带 `time`（匹配进注释），**编译器裁决**裁掉——import 重算的终审是 build 不是 grep。

## 二、线 2 · GA5-04 fold.go → fold.go 874 + fold_timing.go 180

- `fold_timing.go`（180）：耗时折叠域整体（域头注+timingStepStart/TurnStart/Token/Assistant/ToolDispatch/ToolResult/Usage/TurnDone/Close/addWall/timingStepBase/timingToolsRanked/ContextTiming.nonZero）。
- `fold.go`（874）：其余（FoldTimeline/folding 结构/apply 族/文件活动/usage/compaction/surface 对比）。**import 零改动**（json/sort 双侧共用、time 从未真用——先 grep 证伪再动手省一步）。
- 状态字段仍挂 folding 结构体（fold.go 定义），方法按域分文件——**结构不动、方法分家**，比拆结构体低一档风险。

## 三、等价证明与门禁（两线同配方）

- **双证零漂移**：符号清单 before/after diff 空 + `go doc -all` 全包逐字节一致（browser/contextview 两包各验）。
- 行数对账闭合：manager 1047 内容行+18 旧头+3 seam 空行=1068；fold 167+167+874+1+... = 1041。
- 包测试：browser 5.3s 绿、contextview 0.4s 绿（timing_test 专属用例直接覆盖搬移代码）。
- 前台 ci **exit 0**：golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例全绿 / 卫生守卫绿。
- 新文件四枚进显式 add 清单；提交后复验「树干净 + HEAD 可编译」。

## 四、god-file 余量与下一批

- 已销：GA2-01、IN2-12（批 25）+ GA5-08、GA5-04（本批）。god-file 余 ~17（App.tsx 1041/mpp.go 1035/project.go 1011/controller.go 963/agent.go 975/memory_hub.go/CostLibraryView 1132 等前端组件与 god 包类）。
- 下一批候选：同配方续拆（IN3-03 mpp.go 版本布局分家 / IN1-01 project.go 八类关注点——后者动核心需更厚预核）；coupling ~20 与零散死码继续等拍板。
