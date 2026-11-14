# Gaea 进度板块 × 斑马 ZPert8 复刻 + DSH 批代收编预览包

**包标识：** `gaea-schedule-zpert8-20261114-1052adec`

**性质：** 功能提交预览，不是正式版本号发布（版本源 `wails.json` = 4.543.0，本包不抬号、以提交号标识）。

## 构建来源

- 提交：`1052adec`（`fix(schedule): 补齐命令台缺失文件——HEAD 检出不完整修复`）；
  打包范围 = 上一个包 `aa60faa8`（v2）之后的 **15 笔提交**，`git merge-base --is-ancestor`
  已验证 dsh 系全部在包内：
  - **DSH 批代系**：`18fb5dcb`（align context and fleet with dsh-context source）、
    `2eb8d9fd`（dsh-context 0.84.1 上下文交互续刀，用户拍板代提交）——上包 v2 已含，
    本次包延续包含；`9fd5296c`/`b4c0d120` 记档随行。
  - **进度板块线（本包主体，14 笔）**：板块分离一级 `2a564426` → S1~S4 收编 `feda9c04` →
    S5 预警 `8bd52a64` → 后端 agent 面 `77ad3d31`（ExecutionNarrative+AskFinish *int）→
    xlsx 要求完成列 `110f8553` → MPP 判定 `cd1e119e` → get 读面 `72753f7e` →
    视图库 S6.2+codex UX 首刀收编 `284c34b5` → 一次点亮批1~4 `64a2f632` →
    批5 格式组/工作分类 `83820741` → 检出修复 `1052adec`。
- 构建方式：独立干净工作树检出 `1052adec`（`.tmp/pkg-zpert8`，构建后回收），
  前端依赖 junction 复用主树 `node_modules`（同 lock、主树源码改动不进包）；
  先 `npm.cmd run build` 再 `wails build -s -ldflags "-s -w" -trimpath`（bindings 未跳过）。
- 构建工具：Wails 2.13.0、Go 1.26.10 windows/amd64、UCRT64 GCC、系统默认 TMP/TEMP。
- 构建时间：2026-11-14 16:30（Asia/Shanghai）。

## 检出完整性修复（本包首发发现）

`83820741` 的 `SchedulePage.tsx` 已 import `commandRegistry` / `ScheduleCommandPalette`，
但四个命令台文件（registry+测试+Palette+测试）一直在树面**未跟踪**——干净工作树
构建即炸 TS2307（「磁盘能构建≠提交能检出」老坑再现）。`1052adec` 批代入库修复，
本包以修复提交为构建基线。

## 本包增量（进度板块线为主）

- **板块归属**：进度计划从办公板块分离为一级板块（双端 manifest inMenu=true）。
- **斑马复刻 S1~S5**：承载位（reviews/aoa/networkStyle/views）→ N1 点-箭持久化 →
  检查器 18 条 → 检视+拉直+ReviewPanel+横道前锋线 → 预警状态机（红绿灯/两算法/五档/
  askFinish 双端）→ xlsx 上报件「要求完成」列 → agent 面 schedule_analyze/get 执行期
  治理叙事。
- **视图库 S6.2**：保存/管理/应用命名视图（斑马同名校验原文），schedule_get 出清单。
- **一次点亮**：ribbon 20/22 占位启用——大纲五操作（C13 预演）、关系、查找、分组…、
  复制/粘贴、智能布图、**时标网络图/逻辑网络图**（buildAoa mode=logic x=拓扑深度）、
  格式组四项（Task.style 双端 schema）+工作分类字段。
- 详见 `进度计划/gaea-schedule-zpert8-*-20261114.md` 三档判定书与 progress.md 行级记档。

## SHA256（详见 `SHA256SUMS-1052adec.txt`）

| 文件 | 说明 |
|---|---|
| `payload/gaea.exe` | `08b513d4859fe8b6…`（64,387,844 B，-ldflags "-s -w" -trimpath） |
| `payload/onnxruntime.dll` | `68d0d5f502be4509…` |
| `payload/sherpa-onnx-c-api.dll` | `5dd36dbd3572846b…` |
| `payload/sherpa-onnx-cxx-api.dll` | `4c56327796cdb980…` |
| `gaea-1052adec-preview.zip` | `8ff627600ecfe923…`（payload 打包，磁盘留存不入库） |
| `gaea-1052adec-source.tar.gz` | `2fa938c52b3e43c4…`（git archive 1052adec，磁盘留存不入库） |

## 验证

- 干净工作树前端生产构建通过（39.25s，检出修复后）。
- Wails 桌面构建通过（1m0.2s），bindings 未跳过。
- exe+三 DLL 在正式路径 `releases/…/payload/` 执行 `scripts/smoke.ps1`：
  `/api/health` 返回 `200`，响应 `{"status":"ok"}`（PID 48808 已回收）。
- 主树装机另跑（日常装机 exe `FA7A4F2EB5D8CD8D`，与本包提交号一致 lineage）。

## 版本边界

版本源 `wails.json` = 4.543.0 未动（本包以功能提交号标识，不冒充正式版本号）；
`frontend/src/pages/SchedulePage.test.tsx`、`schedule.css`、`.gaea/*` 与命令面板
续刀为 codex 在途改动，**不进本包**（干净工作树纪律）。
