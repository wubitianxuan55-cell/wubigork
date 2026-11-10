# Gaea 代码图谱预览包（设置面板神经网络形态上桌）

**包标识：** `gaea-codeatlas-20261110-b27066b9`

**性质：** 功能提交预览，不是正式版本号发布。

## 构建来源

- 提交：`b27066b9`（`feat(settings): 代码图谱神经网络形态重设计——前馈拓扑分层+神经元/突触/信号三件套`）
- 构建方式：独立干净工作树检出上述提交（`.tmp/pkg-atlas`，构建后已回收），
  前端依赖 junction 复用主树 `node_modules`（同提交同 lock、主树零未提交
  改动），先 `npm run build` 再 `wails build -s -ldflags "-s -w" -trimpath`。
- 构建工具：Wails `2.13.0`、Go `1.26.10 windows/amd64`、Node.js `25.2.1`、
  UCRT64 GCC（`C:\msys64\ucrt64\bin`）、系统默认 TMP/TEMP。
- 构建时间：2026-11-10 22:43（Asia/Shanghai）。
- 冒烟在正式路径 `build\bin` 执行（AGENTS.md 纪律：不从 `.tmp` 运行 exe）。

## 本包增量（自上次装机 7DFE7DC4239BBA44 起为同源重建；自 a2d7 包 317568d4 起）

- **设置面板「代码图谱」标签页（第十分类）**：`frontend/src/data/codeAtlas.json`
  仓内快照（162 Go 包/467 内部依赖边/十层口径），由
  `go run scripts/code_atlas_doc.go -json > frontend/src/data/codeAtlas.json`
  单源再生成；快照日期与再生成命令在页面上屏（防「以为是实时」）。
- **神经网络形态可视化**：真拓扑前馈分层——DAG 最长路径 height（h0 基础库
  49 神经元 → … → h10=`internal/app` 独苗 → h11=`(main)` 唯一输出神经元），
  467 条贝塞尔突触全部严格左→右（前馈不变量测试钉死）；神经元三件套
  （辉光/胞体/核点）+层色 inkColor 两态自适应+78 条流动信号+选中点火环；
  `prefers-reduced-motion` 下动效全停；零新依赖。
- **交互**：十层 CheckableTag 图例=过滤（含计数）、包名搜索淡显/高亮、
  点选神经元高亮一跳突触+检视条（源码/测试/被引/出边）、重置、全隐藏空态。
- i18n 三语（en/zh/zh-TW）；测试 7/7（含前馈不变量）。
- 前情增量（317568d4 已载）：A2 动态工具目录三态+D7-B 取消执行双闸。

## 使用方式

解压后 `gaea.exe` 与三枚 DLL 同级运行。缺任一 DLL 时语音链路不可用。
「代码图谱」入口：设置 → 代码图谱（「数据」与「关于」之间）。
装机态同款已在 `C:\AI\gaea-app`（PID 18360，2026-11-10 22:44，exe 哈希与本包一致）。

## 文件校验

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `gaea.exe` | 61,966,336 | `0E5D29D80F0CA55CB83141F7D58BA87CAA5343796C9FA16BC81B368D503CEFF0` |
| `onnxruntime.dll` | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| `sherpa-onnx-c-api.dll` | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| `sherpa-onnx-cxx-api.dll` | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| `gaea-b27066b9-preview.zip` | 31,650,776 | `0F0E410971402D60536EC144AC3CC6E4136AB203ABBCBE3EE789AF14FC4BAEBA` |
| `gaea-b27066b9-source.tar.gz` | 38,843,790 | `3664A9ED0543BE86AE236A874BD10E66E8D912937E7EFA3448FEF7634E4FF509` |

统一校验见 `SHA256SUMS-b27066b9.txt`。

## 验证

- 干净提交的前端生产构建通过（23.1s）。
- Wails 桌面构建通过，bindings 未跳过。
- exe+三 DLL 在正式 `build\bin` 执行 `scripts/smoke.ps1`：
  `/api/health` 返回 `200`，响应 `{"status":"ok"}`。
- 代码图谱面板：vitest 7/7（快照口径/层过滤/点选/搜索/重置/空态/前馈不变量）；
  视觉代理明暗双态验收 pass（探针：467 path/162 节点/78 信号/令牌落位）。

## 版本边界

三处版本源已对齐 `4.535.0`（wails.json 真源，`sync-version.ps1` 同步，提交
4665c267）；本包以功能提交号标识，不另抬版本号。
