# gaea-codex-distill-batch4-20261117-2a11f08a

codex 0.165.0 → gaea 蒸馏**第四批（B 批 · 后端波）收官包**。

## 交付点

- 功能提交：`2a11f08a`（`feat(gaea): codex 蒸馏第四批 Wave 2+3`）+
  Wave 1 `0d42c3fb`（P1/P3/P4）；两笔同批发布。
- Bindings 再生：`c393f847`（`chore(bindings)`——干净树构建再生出
  `trajectory.ToolRec` 的 `model/turn` 两字段，B5 归因的正当漂移）。
- 本包只含第四批（P1–P9）；第三批前端与前三批已各自发包
  （`gaea-codex-distill-batch3-20261117-d30cb5d2` /
  `gaea-codex-distill-batch12-20261117-4be0ddc9`）。

## 第四批内容（9 项全落地，归档 8 项在册）

- P1 B1 优雅关闭：controller draining/admitted/BeginDrain + Shutdown 步超时
  + stderr 尾截 4KiB（`[agent] drain_enabled` 默认关）。
- P2 B12 截断口径（随第三批 Wave 2 提前落地，共享 `output_budget.json`）。
- P3 B5 工具归因：工具事件 Model/Turn 盖戳→会话日志→轨迹折叠；
  退役零消费 auditFunc。
- P4 B9/B7 出网安全：`netclient/outbound.go` 四态拒因+许可撤销+≤10 跳+
  跨源剥凭据头；构造即验证。
- P5 B4 step 快照：回合内权限/窗口变更三态拒+理由透传。
- P6 B3 持久化守卫：投影超前丢弃（灰度 warn）。
- P7 B10 观测最小集：`internal/gaea/telemetry`（默认关，14 天）。
- P8 B6 断网闸+能力声明：`[sandbox] network=false` 拒 web/浏览器/MCP。
- P9 B8 MCP 目录缓存+启动 deadline：指纹键 32 条/30min + 15s fail-soft。

## 构建

- 独立干净工作树 `.worktrees/pkg-2a11f08a`（detached @`2a11f08a`，
  frontend node_modules junction）。
- `npm.cmd run build` 23.91s（Vite 绿）→ UCRT64 GCC + `CGO_ENABLED=1` +
  系统 TMP/TEMP，`wails build -s -ldflags "-s -w" -trimpath` 49.061s
  （Skip Bindings=false）。wailsjs 仅 models.ts +4 行内容漂移
  （ToolRec model/turn，已随 `c393f847` 物化回主树）。
- 版本：`wails.json` 4.544.0（不抬号——后端治理批，无前端可感面）。

## 冒烟

- 正式 payload 路径 `scripts\smoke.ps1 -ExePath` → `/api/health` 200
  （`{"status":"ok"}`，PID 40852 已回收）；构建/冒烟后无孤儿进程。

## 边界

- 未装机（`gaea_install.ps1` 未运行，`C:\AI\gaea-app` 未覆盖）。
- 默认关开关（drain/telemetry/outbound restricted）待实弹后启用；
  P6 超前断言灰度 warn 一版后升 error。
