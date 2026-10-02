import io

p = r'C:\AI\wubigrok\.gaea\progress.md'
s = io.open(p, encoding='utf-8', newline='').read()

entry = '''## 非版本刀：第一轮 P0 止血刀（7 条并行线）· 2026-10-02

- **刀型**：接续同日全仓代码审计（见下一条），按建议顺序「**守卫 → 止血**」落地第一批：7 条并行子代理线（4 槽位），主代理定契约、独立验收（含真实配置端到端探针）、跑全量快闸收口。
- **交付**：`docs/code-audit-2026-10-02/round-1-fixes.md`（逐刀红→绿证据 + 口径更正）＋ `round-1-p0-status.md`（25 条 P0 在当前工作树的复核）＋ `scripts/check-primitives.mjs`（原语守卫 warn 档 + 真实基线 `machine/primitive-baseline.json`）。
- **七刀**：①**配置保存丢段**（`RenderTOMLPreserving` 段级＋键级保留；真实配置副本端到端「16 段→5 段」的静默丢失已消除，且 `[agent] effort`/`[tools] compact`/`[[providers]] thinking` 这类段内未渲染键一并保住）②任务收件箱读-改-写加锁（进程内）③`ExtractJSON` 改括号配平（21 处调用点零改动）④GenUI 校验器区分「节点数组/数据数组」⑤进度契约（多基线 + `Quantity/Amount` 三态 + 资源 `calendar`；前端零改动、golden fixture 字节不变）⑥原语守卫脚本（截断 41 / `internal/app` 原子写 13 / 前端旁路 5；warn 档未接 CI）⑦`bookimport` 第二份 `ExtractJSON` 收敛到 util。
- **复核更正报告口径三处**：P0#17 **撤销**（`setAllSelected`/`clearSelection` 定义在同文件 1126-1131，函数声明提升，原「未定义函数」判据不成立 → 降 P2「缺表头全选用例」）；P0#21 方向更正为「**显式 0 → 缺键**」（原写反）；P0#22 生产调用点更正为 **21** 处。另：报告 25 行里 #19/#20/#21 与 #1/#2/#8 重复 → **独立 P0 = 22 条**，本轮**已修 7 条独立、仍开放 10 条**（清单在 `round-1-p0-status.md`）。
- **坑（进在册）**：① **TOML 根标量按原位置写会被后续表头吞掉**（`workspace` 读写回空串）→ 保留式渲染必须把根标量前置到第一张表之前；② `Load()` 对数组表是**整片替换**，保留必须**按位次**配对（按 name 会错配 thinking/effort）；③ `sandbox`/`plugins` 也在渲染器职责内（清单漏列），当未知段整段保留会写出**重复表头 = 非法 TOML**；④ **并发线同包编译互相挡**（`render_preserve.go` 在途时 `internal/app` build failed）——集成门必须等所有写者停笔再跑；⑤ 子代理留下的临时探针测试文件必须收尾删除并 `Test-Path` 复核（本次 `zz_probe_save_test.go` 已删）。
- **未做（有意）**：守卫脚本仍是 **warn 档**（接 CI 前先拍板验收线：截断 41→≤5 / 原子写 13→0 / 前端旁路 5→0）；未抬版本、未动 CHANGELOG（发版仪式归用户）；仍开放的 10 条独立 P0（后台链登记纪律 / 检索吞错 / 编排三合一 / 微信语音并发 / 绘梦单槽进度 / 三类复发型缺陷守卫）待排下一批。

'''

anchor = '## 非版本刀：全仓并行代码审计（27 单元 + 6 分册 + 2 附录）· 2026-10-02'
if anchor in s:
    s = s.replace(anchor, entry + anchor, 1)
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
    print('inserted')
else:
    print('MISS anchor')
