# 全仓审计第 29 批 · god-file 对账收窄（GA1-03/GA4-02 价值判定降级）+ 留池 gofmt 清账 · 2026-10-03

> 接续 [round-30（批次二十八）](round-30-p1-batch28.md)。god-file Go 侧六刀后对本批候选做**拆分价值判定**，两条降级出池；顺手清掉留池一项。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `2f6947c3`（批次二十八）；工作树干净。

---

## 一、GA1-03 agent.go：判定「拆分价值低」，降级出可拆池

- 复核：agent 包**已高度分解**（80+ 文件：agent_run/agent_stream/agent_config/compact 族/tool_dispatch/execute_one/stop_gate/plan_mode/fold_ladder/spill/task/tokenmeter… 各域独立成文件，测试 40+ 件）。
- `agent.go` 971 行的实际构成：接口族（Asker/PermissionRequester/Gate/ToolHooks/Runner/TurnResult）+ ctx 装饰器（:30-130）+ **AgentRunner 结构体本体 280 行**（:212-495）+ 一行式访问器/注入器 ~200 行（:496-695，已是琐碎单行）+ New（:696-794）+ 中途转向/背景循环判定两个实质方法。
- 判定：再拆只能搬「一行式访问器」和 ctx 装饰器——**搬移收益接近零**（无 diff 冲突面收益：访问器极少改）；真正的「God 类型」是 40+ 字段结构体本体，收敛它属于**设计重构（拍板项）**而非文件组织。原审计「孤儿注释残骸」在历批（GA4-09 等）已清。
- 处置：GA1-03 从 god-file 可拆池移入「拆分价值低；God 类型收敛=拍板候选」。

## 二、GA4-02 controller.go：同判定降级

- 复核：control 包已分解 30+ 文件（controller_approval/memory/mcp/rewind/session/submit + input/slash/refs/dream/attachments…）。controller.go 963 行剩核心结构（**40 字段上帝对象本体**）+New/装配——与 GA1-03 同型。
- 处置：同上降级；字段收敛（分组子结构）=设计拍板候选。

## 三、留池清账：test_image_models.go gofmt

- `scripts/test_image_models.go` 补现代构建标签 `//go:build ignore`（与既有 `// +build ignore` 成对；scripts/ 不在 golangci 覆盖内故存量为假阴性，在册五批）。
- gofmt -d 预览仅此一行差异；gofmt 解析通过=语法完好；`gofmt -l scripts/` 归零。池项关账。

## 四、god-file 对账后余量

- Go 侧：**可拆池清零**（六刀销六条 + 两条判定降级）。前端组件族 ~13 条在册（CostLibraryView 1132×2/CostProjectsView 862/controller.ts 963/App.tsx 1041/ModuleLauncher 1038/methods_office.ts 1029/mock 81 绑定等）——组件/TS 模块拆分=状态上提+import 穿线，**与 Go 同包分家不同型**，须先立前端配方（金样 DOM/vitest 契约断言先行，小组件试水），列下批优先。
- 下一批候选：前端拆分配方试水（FE3-15 methods_office.ts 的 DAG 前言+域分组最接近机械形态，contract 744 绑定测试=现成金样）。coupling ~20 与零散死码继续等拍板。
