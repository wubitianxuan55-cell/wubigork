# gaea-pantheon-subagent-ui-20261118-fc120738

**打包点**：`fc120738eac3c461d0d4448558c1541a249452f2`
**提交标题**：`feat(office): unify Pantheon and subagent conversation UI`
**版本**：4.544.0（不抬号；日常构建）

## 本包内容

- 众神殿独立 Agent 窗口：观点默认折叠，支持批量收起/展开；轨迹合并工具记录，
  增加统计、轮次概览、搜索、轮次分组与失败状态文字。
- 众神殿新增上下文标签页：上下文账本、Prompt 构成、会话信息、逐轮趋势与账目；
  未上报 usage 时使用单一诚实空态。
- 标签栏使用标准 tablist/tab/tabpanel 语义，支持方向键、Home/End 切换。
- 子代理对话复用父代理的阅读列、用户消息、分段、过程卡与工具卡结构。
- 父代理对话过滤后台 task/bash 启动通知及 `subagentRef` 完成回执，避免与
  委派卡/父代理汇总重复。
- 委派卡按会话派发顺序显示 `子代理 A/B/C…`，同时保留任务摘要和既有跳转。

## 源码文件

- `frontend/src/gaea/components/SubagentThread.tsx`
- `frontend/src/gaea/components/SubagentThread.test.tsx`
- `frontend/src/gaea/components/Transcript.tsx`
- `frontend/src/gaea/components/Transcript.test.ts`
- `frontend/src/gaea/components/ToolCard.tsx`
- `frontend/src/gaea/lib/taskActivity.ts`
- `frontend/src/gaea/lib/taskActivity.test.tsx`
- `frontend/src/pages/PantheonPage.test.tsx`
- `frontend/src/pages/pantheon.css`
- `frontend/src/pages/pantheon/PantheonRecord.tsx`
- `frontend/src/pages/pantheon/PantheonRecord.streaming.test.tsx`
- `frontend/src/pages/pantheon/seatInsights.ts`
- `frontend/src/pages/pantheon/seatInsights.test.ts`

## 构建与验收

- 干净工作树：`.worktrees/pkg-fc120738`（detached）
- 前端：`npm.cmd run build`，Vite 20.95s，总前端门禁 46.400s
- Wails：UCRT64 GCC、`CGO_ENABLED=1`、系统 TMP/TEMP，
  `wails build -s -ldflags "-s -w" -trimpath`，44.2s，
  `Skip Bindings=false`
- bindings：仅行尾格式变化；`git diff --ignore-space-at-eol` 内容漂移为 0
- 正式 payload 冒烟：`/api/health` 200，冒烟 PID 41464 已回收
- 装机：`gaea_install.ps1 -SourceDir payload -Launch`，
  `INSTALL OK`，新 PID 59568（启动时间 23:46:45 晚于构建时间 23:46:15）
- 相关前端测试：5 文件 110/110
- TypeScript：`tsc -b` 通过
- ESLint：改动面 0 error

## payload

| 文件 | 字节 | SHA256 |
|---|---:|---|
| gaea.exe | 55,193,088 | 86052955E1F5E290817F01B3D8BD11D8FCC9FB657763C36BC1CA1CF55CB86CEF |
| onnxruntime.dll | 19,288,888 | 68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B |
| sherpa-onnx-c-api.dll | 4,294,656 | 5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8 |
| sherpa-onnx-cxx-api.dll | 264,192 | 4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE |

三份 DLL 与既有基线哈希一致，未重编。

## 归档

- 二进制预览 ZIP：`gaea-pantheon-subagent-ui-20261118-fc120738.zip`
  - 30,177,708 B
  - SHA256 `81D053BE0D6566817E26E4E1DC553EC4E34783815BD2FE8AEC0CC0EEB5FF6539`
- 源码 TAR：`gaea-fc120738-source.tar.gz`
  - 43,815,614 B
  - SHA256 `C1243CE7276B5C2EAB542C106FAB347570B571312DCC1C38EF4AA1DFBD90F7D1`
- 源码 ZIP：`gaea-fc120738-source.zip`
  - 47,969,713 B
  - SHA256 `8FFFA1BC20DB52C16904E15ABF7669B20D5E5F0548400F1805494F3746B0E670`

源码 TAR/ZIP 与二进制 ZIP 留盘，不纳入源码提交；`PACKAGE-INFO.md` 与
`SHA256SUMS.txt` 入库。walkthrough 截图保留在 `screenshots/`。
