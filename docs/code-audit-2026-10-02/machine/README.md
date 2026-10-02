# machine/ —— 2026-10-02 全仓审计的机器守卫

## scripts/check-primitives.mjs · 原语平行实现守卫（06 分册刀1；warn 档，暂未接 CI）

- `node scripts/check-primitives.mjs`（**须在仓库根执行**）：默认 warn 档——打印三项计数表 + 与基线对比 + 新增位置清单，**恒 exit 0**。
- `node scripts/check-primitives.mjs --strict`：有超出基线的新增项则 **exit 1**（以后接 CI 用）；无新增 exit 0。`--json` 输出机读 JSON（可叠加 `--strict`，退出码一致）；exit 2 = 用法错 / 基线缺失或损坏。
- 基线更新：确认新增项合理或完成一轮收敛后，跑 `--write-baseline` 覆写 `docs/code-audit-2026-10-02/machine/primitive-baseline.json`，与刀一并提交、在版本条目里注明计数变化。
- 认键规则：项以「文件 + 形式/API + 该文件内序号」为键，**行号只作展示**——无关编辑造成的行号漂移不会误报新增。
- 口径差异（以本脚本口径为准，可 grep 复核）：截断原语顶层函数 **41** = `grep -rIniE '^func[[:space:]]+(truncate|clip)' internal --include='*.go'`；同一命令去掉 `-i` 只有 **37**（差 4 处大写 `Truncate`/`TruncateRef`/`TruncateToolOutput(With)`），审计的 41 对应 `-i` 口径。方法形式另计 1（`(a *AgentRunner) truncateRescue`，只收小写开头的未导出原语；`context.Store.Truncate` 等语义是截断存储条目、不收）。
- 接 CI 前先拍板验收线：截断原语 **41 → ≤5**（审计附录 A / 06 分册刀4）、原子写旁路 `internal/app` **13 → 0**（全走 `fileutil.AtomicWrite`）、前端旁路 **5 → 0**（`a.download=` 走 `saveFile.downloadBlob`、`atob(` 走 `bytes.b64ToBytes`）；达标后钉死新基线再上 `--strict` 门禁。
- 待拍板：前端 `a.download =` 的第 3 处是 `frontend/src/gaea/lib/saveFile.test.ts:47` 的测试断言字符串命中（生产码 2 处，与 06 分册现场复核一致），接 CI 前定是否豁免 `*.test.ts(x)`。
