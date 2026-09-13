# t1 共享契约只读审查（2026-09-13 18:46，Codex 会话产物）

> 背景：`mumu-impl` 团队正按 `docs/distill/09-impl-handoff.md` 实施六个小说域，t1（共享契约）
> 进行中。本报告为 **只读审查**：未修改任何源码，未触碰该团队文件，仅读取 + 全仓 grep +
> `go build ./...`（exit 0，7.1s）。
> 结论：契约层主体成型（6 态并集 / 水位守卫 / 派生值 / 接口倒置齐备），但存在 **1 个 P0 口径冲突**
> 与 3 个集成观察项，建议在 t2~t7 大规模消费前拍板定死。

## P0 伏笔回收 wire 值被反向归一化（写入口径 = `resolved`）

**权威口径**（`docs/distill/09-impl-handoff.md` §2-3 与 §4-7，18:42 更正版）：
存储与 JSON 的规范值仍是 `"revealed"`；「任何『归一化为 resolved』的改法都是破坏性变更」。

**当前 t1 产物**（与上述相反）：

- `internal/types/foreshadow_v2.go:40-41`：`ForeshadowResolved = "resolved"`（注释写「写入统一用本值」）
- `internal/types/foreshadow_v2.go:78-81`：`NormalizeForeshadowStatus` 把 `revealed → resolved`
- `internal/types/types.go`：`ForeshadowChange.Action` 注释同步写「写入统一用 resolved」

**受影响消费方**（当前树实测，均为硬编码字符串/常量比较；写 `resolved` 后这些路径静默读空）：

| 消费方 | 位置 | 反转后的实际后果 |
|---|---|---|
| 统计 | `internal/stats/stats.go:72` + `internal/app/stats_handler.go:47-48` | `foreshadow_revealed` 计数与回收率归零 |
| 伏笔体检 | `internal/app/novel_foreshadow_lint_handler.go:81/85/129/178` | lint 计数/状态流转判定失配 |
| 分析写回 | `internal/analysis/analysis.go:155/160-161/301`、`evolution.go:136` | 状态推进与统计失配 |
| 章节上下文 | `internal/app/create_chapter_handler.go:655` | 已回收伏笔被**重新注入** prompt |
| 保存校验 | `internal/app/foreshadow_handler.go:30-34` | validStatus 白名单拒收 `resolved` 与新增三态 |
| 叙事 | `internal/narrative/narrative.go:154` | 只识别 planted/hinted/revealed |
| 前端 | `frontend/src/components/novel/ForeshadowPanel.tsx`（12 处，含 `:164/:167` 回收率） | 回收率与状态流转按钮失效 |

**两条出路（需 captain 拍板，二选一）**：

- **A（推荐，零破坏）**：写入口径维持 `"revealed"`，`resolved` 仅作读取别名。
  `NormalizeForeshadowStatus` 方向反转为 `resolved → revealed`；`IsResolvedStatus` 覆盖两者。
  存量 JSON 与上述 7 类消费方**零改动**。
- **B（接受反转）**：必须同时指定 owner 修上表全部 7 处。其中
  `internal/stats/**`、`internal/narrative/**` 在 handoff §4.5 归属表里**没有 owner（孤儿）**；
  `create_chapter_handler.go:655` 属 t6（其 prompt 只提 substituteWordCount，未提此项）；
  前端 12 处属 t7。

**无论 A/B 都必须同步**：新增三态（`pending`/`partially_resolved`/`abandoned`）要同时进
`foreshadow_handler.go` 白名单与 `ForeshadowPanel` 状态流转，否则新态「存不进去 / 点不动」。

## 观察项（不阻塞）

1. **章号解析双实现**：`internal/types/interfaces.go:348 ChapterNumOf` 与
   `internal/app/novel_foreshadow_lint_handler.go:45 chapterNumOf` 逻辑等价（新实现多一层空串保护）。
   建议 t3 改为委托，避免日后分叉。
2. **模板契约镜像**：`internal/types/interfaces.go:28-73` 是 `internal/prompt.Template/InputDef`
   的镜像（+ `version`/`category`/`description`/`order`），字段名与 JSON tag 一致。
   t6 落 `Order` 时须保持 15 份 `prompts/*.json` 原样可读，并明确镜像与 `internal/prompt` 的单一真源。
3. **规避效果靠消费方**：水位守卫 / 派生 member_count / 亲密钳制 helper 已就位
   （`internal/types/character_state.go:170-227`），但真正规避 MuMu 缺陷取决于 t3/t5 **是否真的调用**。
   建议 t8/t9 反向抽查（把 helper 调用删掉后测试是否仍绿——仍绿即测试无效）。

## 与 Codex 侧的交接（工作区并发提示）

- 工作区另有**未提交的在制品 v4.278.0**（伏笔体检 `LintForeshadows`：`internal/app/novel_foreshadow_lint_handler.go`
  + `ForeshadowPanel` + 绑定名/mock/bridge/spaceBindings，版本三处已置 4.278.0），其文件与 t3/t7 足迹重叠。
  建议由集成方在 mumu 落地后统一收口发版；**发成一次还是两次版本需拍板**。
- 生成物（`bindingNames.ts` / `spaceBindings.ts` / `mock/novel.ts` / `bridge/novel.ts` / `wailsjs`）
  按纪律由单一负责人（集成方）在全部后端域落地后统一再生。

---

## 处置结果（2026-09-13，v4.278.0 收口，Codex 侧）

用户反馈实施团队已停摆、工作区由 Codex 侧接管收口，**按上表 A 方案执行并已落库**：

- `internal/types/foreshadow_v2.go`：写入口径回到 `revealed`；`NormalizeForeshadowStatus` 方向反转为
  `resolved → revealed`；`AllForeshadowStatuses` / `IsForeshadowStatusValid` 回到 6 态（含 revealed、
  不含 resolved）；新增 `IsResolvedStatus`（两套 wire 值都判「已回收」）。
- `internal/types/compat_test.go`：同步翻转断言（别名=resolved、合法写入=revealed），并新增
  `TestIsResolvedStatus_CoversBothWireValues` 两套 wire 值覆盖用例。
- 消费方接线（对存量数据行为等价，只增不减）：`internal/stats/stats.go`、
  `internal/app/novel_foreshadow_lint_handler.go`（状态一致性检查 / 状态标签 / 报告计数）、
  `internal/app/create_chapter_handler.go`（已回收不再注入）、`internal/analysis/analysis.go`（聚合统计）。
- `internal/app/foreshadow_handler.go`：SaveForeshadows 白名单从 3 态扩到并集 6 态，写入前先
  `NormalizeForeshadowStatus`（别名 resolved 落库即归一为 revealed）。
- 观察项 1/2/3 维持原建议，留给 t2~t7 实施时处理。
