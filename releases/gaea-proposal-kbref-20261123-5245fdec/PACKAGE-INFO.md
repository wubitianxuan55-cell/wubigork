# gaea-proposal-kbref-20261123-5245fdec

- 交付提交：`5245fdec`
  （`feat(gaea): point proposal mode at knowledge library references`）
- 分支：`main`（ahead origin，未 push）
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-5245fdec`，detached HEAD
  （干净树，主工作树 wenxue 隐私线等在途改动零混入）
- 前端：pnpm `11.21.0` install `27s` + `pnpm run build` Vite `28.36s`
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc、PATH 前置
  `build/bin`（DLL 三件套）；`wails build -s -ldflags "-s -w" -trimpath`，
  bindings 未跳过
- 主树对照构建：前端 Vite `32.49s` + Wails `37.73s`；
  `build\bin\gaea.exe`（同源）冒烟 `200`（PID 53632 已回收）
- 漂移：Wails 再次生成 wailsjs 27 文件 + go.mod，忽略 EOL 后内容差异文件数 `0`
  （纯 CRLF 噪音）
- 冒烟：正式 payload `/api/health` 返回 `200`（HTTP 桥 `127.0.0.1:18999`
  避让活壳），PID `70992` 已回收
- 三 DLL 与既有基线一致（onnxruntime `68D0D5F5…`、sherpa c-api `408F9AB6…`、
  sherpa cxx-api `DBAC55CE…`，与 gaea-bloom029-final-20261123-555eda3e 相同）

## 本批内容（方案模式·参照知识库接线）

- 方案档运行模式行补句：知识库『设计方案』分类已导入方案可作参照——
  动笔前用 `knowledge_search` 按关键词检索，命中后对照章节骨架与评分点口径
  （standard/cost 档文本不动）
- `proposal-longform` 技能阶段①（建清单）前置检索参照步：命中多篇挑最相近
  1~3 篇读全文对照骨架/评分点，无命中不阻塞，再落 `longdoc_manifest` 清单
- 锚测试 `TestProposalRunModeKnowledgeRef`：钉死两处接线，standard/cost 档
  恒不含 `knowledge_search` 句
- 零新存储/零新绑定/零新 UI——方案库=知识库『设计方案』分类视图，
  导入链（AI 解析/查重/INDEX.md 索引）与 `knowledge_search`/`unified_search`
  召回全走现成链
- 定向验证：`go test ./internal/app -run 'RunMode|ComposeModeLine|OfficeMode|KnowledgeRef'`
  **6/6 绿**（含确定性/矩阵/配置归一/身份稿/工具面注册/新锚六例）

## 已知边界

- CDP 实机走查被活壳挡（9333 被正在运行的旧壳占用）：切方案档新会话→上下文
  面板确认模式行含参照句→`knowledge_search` 命中已导入条目，待壳关闭后补
- 未装机、未 push；安装按 `scripts\gaea_install.ps1`（会杀现有壳）

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `22CD2CECC41ECF73FDEAB853078C129F262460EDE59CF735FA845750658593DE` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-proposal-kbref-20261123-5245fdec.zip | `82854F306089013FC54160C466722806C51FC04FF8B377B5DEB775FC0620D729` |
| gaea-proposal-kbref-20261123-5245fdec-src.zip | `D1AF3FFF3F4814F6620A27508146E9F9F694B27F56865C48AE383DA409CF2826` |
