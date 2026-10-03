# 全仓审计第 48 批 · 留池小件三件（outline 同型收敛 / Media 链 comfyui 重试 / 第四围栏样板单源）· 2026-10-03

> 接续 [round-49（批次四十七）](round-49-p1-batch47.md)。收批 19/21 留池尾三件：两件收敛 + 一件**申报行为新增**。**不抬版本**。
> 快照：开工时 HEAD = `a1a07a1e`（批次四十七）；工作树干净。

---

## 一、三件处置

- **findOutlineNodeByNumAny / findOutlineNodeByID 同型收敛**（AP1-06 超清单残留，批 19 登记收口）：两者与 `findOutlineNode(nodes, pred)` 前序遍历骨架完全同型——包装保留原位（调用点/文档注释不动），函数体一行委托，「不限分支」「按 ID 认」语义进谓词显式化（与 findOutlineNodeByNum 的 Branch=="" 收窄同族）。命中序逐位等价（同前序、同首中即返）。
- **Media 链补 comfyui 重试**（round-21 申报项实施，**申报行为新增**）：media 生成链 `comfyRetries: false→true`——Errno22 孤儿实例 recover + 未运行自动拉起（各一次/整轮），与 internal 链同后端同失败模式同语义；spec 字段注释与调用点注释同步改写（「只申报不实施」→「批 48 两链同开」）。无单测：media 链 client 为具体类型无注入缝，编造测试需真客户端——由 app 全包 124s 回归背书既有行为不变。
- **MarkdownContent 第四围栏样板**（round-21 白名单外登记）：实测**仍在**（v4.233 重写后样板转移到 react-markdown 的 code 组件内联体）——`language- 提取/去尾换行/块级判定`三步与 `extractFence` 逐字节等价，接单源（第四缝入列，FE6-05 白名单外残留清零）；`ChatCodeBlock` 入参由 `match?.[1]` 改语义等价的 `lang`。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / **app 全包 123.9s 绿**（outline 收敛+media 重试过全包回归）/ tsc -b 绿 / **全量 vitest 435 文件 3762 例全绿**（MarkdownContent.test+t74 双面在列）/ 改动文件 eslint 0 错 0 警。

## 三、留池台账（批 18~21 登记项清算）

- 已清：compose 检索路径（批 47）/MPP9 门控（批 47）/:54 暖启判据（批 47）/findOutlineNodeByNumAny·ByID（本批）/Media 链 comfyui 重试（本批）/MarkdownContent 第四样板（本批）/45 条 NOT_MOCKED（批 43~46）。
- 证伪/移交：xlsx 不接 ooxml（证伪留池维持）/FE6-05 DOM 统一（产品级=拍板）/IN1-04 rubric 接入（单独立刀）。
- **批 18~21 留池全部清算完毕**。余池：coupling ~20（拍板）、零散死码（等 109 绑定拍板）、上报未动池（fsync 决策=拍板、app 8 处非同构 CreateTemp）、观察池（WhisperSubgraph 大小写疑似真机 bug）。

## 四、下一批候选

- 对账地图已无可独立执行的免拍板活池——**审计修复主线在此收束**，后续进入拍板池裁决或新审计周期。候选：请用户对拍板池十项+109 绑定删除批量裁决；或零散 CreateTemp 非同构八处续收（同批 41/42 配方，可免拍板）。
