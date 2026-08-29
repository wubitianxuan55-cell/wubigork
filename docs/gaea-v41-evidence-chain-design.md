# v4.1「信任」办公包 · 证据链设计

> 依据：`docs/gaea-nextgen-roadmap-2026.md` §10.4（v4.1）、§1 通用办公革命性跃升 1/2/5、
> §15 补丁（证据卡存原文摘要）。
> 前置：阶段 0–2 已收官（edit_file 工具层 / 审批决策族 / Plan→Apply / 事件日志空间化 /
> 双空间壳）。v4.1 是工位办公包第一刀——「审阅后」护城河：**每次变更可追溯、可复核、可回滚**。

---

## 1. 目标与红线

- **目标**：把工位（work）的「AI 改文件」从不可审计的魔法，变成
  **证据链（Apply→Verify→Journal 三段式）**：每次变更绑定证据卡（来源/摘要/工具/模型/
  时间戳），自动复核闭环（pass/warn/fail），一键回滚，审计导出。
- **红线**：
  1. 证据卡必须存**原文摘要**（非 truncateToolOutput 截断文本）——§15 补丁；
  2. 证据链只覆盖**工位**（play 是创作空间，不做审计）；
  3. 回滚只回滚本回合 AI 变更，绝不覆盖用户后续手工编辑；
  4. 每 Step 独立提交可回退；旧数据只读兼容。

## 2. 现状资产盘点（全部可复用）

| 资产 | 位置 | 复用点 |
|---|---|---|
| 事件日志（含 space） | internal/gaea/event + 会话目录 | Journal 投影的事实来源 |
| 审批决策族（allow/deny/abort/persist_allow） | controller_approval + 前端 ApprovalModal | Apply 前的闸门已具备 |
| Plan→Apply 两段式（xlsx） | GaeaXlsxPlanEdit/ApplyEdit | 工具侧已有「规划→批准→执行」形态 |
| edit_file 工具层（五工具） | internal/gaea/tool/builtin | Apply 统一入口（后续工具接入证据卡） |
| 产物路径分区 | spaces.ExportsDir（.gaea/work/exports） | Journal 导出落点 |
| LibreOffice 无头渲染 | office/（ConvertToPdf 同管线） | Verifier 通道 B 视觉 diff |

## 3. 证据卡模型 `evidence.Record`

```go
type Record struct {
  ID        string    // 证据卡 id（回合内稳定，可被 Journal 引用）
  SessionID string
  Space     string    // 恒 "work"（play 不落证据链）
  Turn      int       // 回合号（事件日志对齐）
  Tool      string    // 变更工具（edit_file / xlsx_apply / docx_apply / …）
  Target    string    // 变更目标（工作区相对路径 / 单元格引用 / …）
  // 原文摘要（§15 补丁）：变更前内容的关键片段（非截断展示文本）；
  // 由工具在变更前捕获（ReadFile 原文 → 摘要上限如 8KB）。
  BeforeSummary string
  AfterSummary  string
  // 证据来源：模型 + 时间戳 + 检索/文件来源（对齐「可追溯证据链」）
  Model     string
  At        int64    // unix ms
  Sources   []Source // 来源文件/检索片段（可选）
  // 回滚信息：反向补丁或基线快照引用（见 §6）
  Rollback  *RollbackInfo
  Status    string   // "pending_verify" | "verified" | "warned" | "failed" | "reverted"
}
```

- 落点：独立 `internal/gaea/evidence` 包 + 会话事件日志旁（`.gaea/work/journal/` 按会话分文件，
  JSONL，追加写；旧会话无 journal 时只读兼容缺省）。
- **摘要来源**：Apply 工具在写盘前捕获 Before（已有 ReadFile/编辑前快照路径），写盘后捕获
  After；不依赖展示层截断。

## 4. Apply→Verify→Journal 三段式

1. **Apply**：审批通过后，变更工具执行并产出 `evidence.Record`（含 Before/After 摘要）。
   现有工具逐步接入（xlsx_apply 先行——已有 Plan/Apply 形态；edit_file 次之）。
2. **Verify**：`Verifier` 对 Record 做双通道复核（§5），产出 verdict（pass/warn/fail）。
   - pass → 状态 verified；warn → 状态 warned（可见可继续）；fail → 变更被标记 failed，
     回合提示用户可一键回滚（不自动回滚——保留人裁决）。
3. **Journal**：回合结束（TurnDone）将本回合全部 Record + verdict 投影为
   `JournalEntry`（可读摘要 + 导出 markdown/JSON），落 `.gaea/work/exports/journal/`。

## 5. Verifier 双通道

- **通道 A（结构/引用完整性，本地无头）**：
  - xlsx：excelize 重算公式（复用 xlsxedit recalc）+ 单元格引用完整性 + 行/列/表名存在性；
  - docx：引用/书签/超链接目标存在性；
  - 文本：目标路径存在、无乱码标记（UTF-8 边界）。
- **通道 B（视觉 diff）**：LibreOffice 无头把 Before/After 各自渲染 PDF → 页面对齐
  diff（像素变化率 + 差异页截图路径），超出阈值判 warn/fail。复用 ConvertToPdf 管线
  （独立 UserInstallation profile，防锁冲突）。
- 判定：A fail 或 B 大改且 A 无法解释 → fail；B 中改 → warn；其余 pass。

## 6. 回滚语义

- **基线快照 + 反向补丁**：Apply 前把目标文件复制为临时基线（`.gaea/work/rollback/<id>/`），
  回滚 = 用基线覆盖目标。**冲突保护**：回滚前比对目标当前内容与 AfterSummary——
  不一致（用户已手工改过）→ 拒绝回滚并提示差异，绝不覆盖用户编辑。
- 回滚后 Record.Status = "reverted"，Journal 追加一条回滚记录（可审计）。

## 7. 中文规范包（第一刀：GB/T 9704 红头子集）

- 范围收敛：**红头文件版式 lint**（标题二号小标宋/红头字色、版记要素、成文日期与
  盖章位占位、页边距档位），输出体检报告（不合格项 + 定位 + 一键修复建议）；
  造价/工程表式规范包放 v4.2 随造价包走。
- 形态：`internal/office/standard` 纯函数校验器 + 前端「体检报告」面板
  （挂 OfficePanel 下，不新增板块）。

## 8. Step 拆解（每步独立提交可回退）

- **v4.1a 证据链**：evidence.Record + 落库 + xlsx_apply/edit_file 接入（Before/After 摘要）
  + 回合 Journal 投影 + 前端「证据」入口（复用 DeliverablesPanel 挂载点）。
  验收：改 xlsx 后 Journal 含可读证据卡与原文摘要；旧会话零报错。
- **v4.1b Verifier**：通道 A（xlsx 重算/引用）+ 通道 B（PDF 视觉 diff）+
  verdict 状态机 + 失败提示回滚入口。验收：构造坏公式/引用 → fail；改版式 → warn。
- **v4.1c 规范包**：GB/T 9704 红头 lint + 体检报告 + 修复建议。验收：样例红头
  检出缺版记/错字色，建议可执行。

## 9. 不做（边界）

- 多文件任务图 DAG 编排 / 项目本体记忆（v4.1+，另行排期）。
- play 空间任何审计（红线：乐园不落证据链）。
- 自动回滚（fail 只提示，人裁决）；自动撤销审批（沿用现有审批闸门）。

## 10. 验收红线（纳入每 Step）

1. 证据卡摘要 = 原文片段（非截断展示文本）——单测断言 Before/After 摘要与文件实况一致。
2. 回滚冲突保护：目标被手工改过 → 拒绝回滚，零覆盖。
3. 旧会话/旧产物只读兼容，无迁移报错。
4. play 会话不产生任何 evidence.Record。

## 11. v4.1a 落地记录（2026-08-29）

- `internal/gaea/evidence/journal.go`：`ChangeRecord`（Before/After 原文摘要，8KB 上限）、
  `ChangeLedger`（回合内存台账，ctx 盖章）、`JournalStore`（按会话 JSONL 追加 +
  turn markdown 投影）。
- 接线：`AgentRunner` 持有台账 + `journalDir`（Options.JournalDir，boot 注入
  `.gaea/work/journal`）；回合收尾 `flushJournal` 落盘 JSONL + 导出
  `.gaea/work/exports/journal/<session>/turn-<n>.md`；**play 回合整体不落盘**（红线）。
- 工具接入：edit_file / write_file / move_file 成功后经 `evidence.RecordChange`
  上报（ctx 无台账静默——直调/dev/旧后端兼容）。
- 验证：evidence/builtin/agent 全量 Go 测试绿。
- **未做（v4.1a2）**：multi_edit/edit_lines 逐条摘要、xlsx_apply 接入（App 层 Apply
  入口）、前端「证据」入口（复用 DeliverablesPanel）、Verifier（v4.1b）。
