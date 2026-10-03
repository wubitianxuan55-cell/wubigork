# 全仓审计第 52 批 · 拍板执行首批（estimateTokens 同型收敛 / compact 三件套摘除 / God 类型冻结约定）· 2026-10-03

> 接续 [round-52（批次五十）](round-52-p1-batch50.md)与 [decision-brief](decision-brief-2026-10-03.md)。用户「继续」视为按建议执行可安全项（批 52=B7①+B6①+A3 约定；A1 大刀与 B1/B2 新功能面仍待明确拍板）。A4/A5/A6/B3/C1/C2 六项=维持现状零动作（钉/注释已在册，无改动）。**B7 收敛零行为变化；B6① 摘除=申报行为移除（前端绑定面）。不抬版本。**
> 快照：开工时 HEAD = `de5b31ea`（批次五十）；工作树干净。

---

## 一、B7 · estimateTokens 口径收敛（普查实为五处三家族）

- **bytes÷4 floor 族三处 → `util.EstimateTokensLen` 单源**：contextview/estimate.go（连带删 fallbackTokPerChar 常量）、trajectory/fold.go（agentnet/fold 四调用点经本地包装受益）、app/sin_insight.go（sinEstTokens）——包装保留、调用点零改、字节等价（×0.25 与 ÷4 对 int64 严格相等，0.25 是 2 的负幂）。
- **两处方言钉死不并入**：agent/cache_shape.go `(len+3)/4` = **ceil**（缓存预留上限保守口径）；util.EstimateTokens = **语言学口径**（context 引擎预算，中文 1.5/字+英 1.3/词）——三家族服务不同面，互钉注释「跨型不许顺手对齐」。
- **自伤一次被编译器当场抓获**：重写 contextview/estimate.go 时只读了头 10 行整文件覆盖，同文件的 scaleCategory/briefOf/indexByte 丢失 → build 红（undefined: briefOf）→ 从 git show HEAD 补回全量。教训入册：**整文件重写前必须读全文件**（Read limit 是给自己挖坑）。

## 二、B6① · compact 三件套摘除（六面+两处编译器追捕）

- 摘除面：bridge/core.ts 声明 / bridge/proxy.ts 逐名理由 / bridge/drift.ts MOCK_ONLY_NAMES（**清单暂空，机制保留**）/ mock/chat.ts 方法+Pick / controller_actions_workspace.ts 回调 / controller.ts 解构+return。
- **编译器追捕两处漏网**：spaceBindings.ts 分类表 Compact 行（双向漂移锁当即红）+ proxy.mockonly.test.ts（钉的正是「Compact 在册」前提）——后者重写为**摘除后态防回潮钉**（MOCK_ONLY_NAMES 暂空 + 四名死成员在 facets/mock 不得复现；未来新登记须随本测试一起更新=fail-closed 登记）。
- 摘除依据：批 20 实证零 UI 调用者；真机压缩由后端会话事件自动执行；mock-only fail-fast 机制（FE3-04）保留供未来名字登记。
- 申报：前端绑定面 -1（AppBindings）；全量 vitest 3762→3760（摘除的两个 Compact 用例，账平）。

## 三、A3 · God 类型冻结约定

- `.gaea/AGENTS.md` 技术栈约定节新增：历史巨结构体维持现状不拆，**新增字段必须挂子结构体**，不得往顶层铺平（round-31 判定引用 + decision-brief §A3 指针）。字节数 56011→56446（预算 65536 内 ✓）。

## 四、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / contextview+trajectory+agent 三包全绿 / app Sin 族定向绿 / tsc -b 绿 / contract+spaceBindings+mockonly+store 金样 72 例绿 / **全量 vitest 435 文件 3760 例全绿** / 改动文件 eslint 0 错 0 警。提交后树干净 + HEAD 双侧可编译。

## 五、拍板执行台账

- 已执行：B7①（本批）/B6①（本批）/A3 冻结约定（本批）。
- 零动作确认：A4（IN2-07 维持）/A5（AP5-06 冻结）/A6（GA6-05 方言钉）/B3（DocumentLint 保留）/C1（FE6-05 不做）/C2（IN1-04 随评测）——钉与注释均已在册。
- 仍待明确拍板：**A1 109 绑定删除**（大刀，建议已给）、**A2**（AP4-01 统一任务身份小刀，建议已给）、**A7 fsync**（brief 建议②文字与选项表述有出入需您定夺：默认开还是默认关）、**B1/B2**（新功能面立项）。
