# gaea-codex-distill-batch3-20261117-d30cb5d2

- **打包点**：`d30cb5d230164cdd05aaba6832061e674b56db13`
  （`feat(gaea): 完成 codex 蒸馏第三批双板块收口`，94 files changed）。
- **构建树**：独立 detached worktree `.worktrees/pkg-d30cb5d2`，固定在
  `d30cb5d2`；`frontend/node_modules` 通过 junction 复用主树依赖。
- **包面**：codex 蒸馏第三批 W1–W9 全清，并同步落地第四批 P2 输出预算同源：
  双壳审批/Composer/工具逻辑单源化，审批焦点与终止键、回退判据、异步乱序门、
  会话去重；前后端共用输出预算与 UTF-8 安全头尾截断；Esc 阶梯、显示宽度度量、
  渲染闸门、表格 GFM 复制和 GenUI 2 MiB 源门。第三批之外的剩余第四批事项、
  F 批与 B 批不包含在本包。
- **构建态**：前端 `npm.cmd run build` 总耗时 81.412s，其中 Vite 34.85s；
  仅有既有 CSS minify 与大 chunk 告警。随后使用 UCRT64 GCC、
  `CGO_ENABLED=1`、系统默认 TMP/TEMP 执行
  `wails build -s -ldflags "-s -w" -trimpath`，总耗时 49.665s
  （编译 49.387s），日志确认 `Skip Bindings=false`。
- **构建树一致性**：Wails 再生后 `frontend/wailsjs/**` 与 `go.mod` 仅显示
  CRLF 状态噪声；`git diff --ignore-space-at-eol` 内容漂移为空。
- **功能门禁（提交时）**：Vitest 494 文件 / 4453 例绿，tsc + Vite 绿，
  ESLint 0 error；Go `strutil/agent/archive/control` 四包、vet 与
  `go build ./...` 通过。
- **版本**：源版本 `wails.json:info.productVersion = 4.544.0`，不抬版本；
  当前及上一包 exe 的 raw `FileVersion`/`ProductVersion` 字段均为空，
  因此不把源版本伪记为 PE 资源版本 `4.544.0.0`。
- **运行时 DLL**：三件 DLL 保持上一已验收基线的字节内容不变，从
  `releases/gaea-codex-distill-batch12-20261117-4be0ddc9/payload/` 原样复制。
- **payload SHA256**：
  - `gaea.exe` `E058DB7E3CFDD1CA3E6B38A3D6D569A1EC9D2F57B76B7011421031412C080119`
  - `onnxruntime.dll` `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B`
  - `sherpa-onnx-c-api.dll` `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8`
  - `sherpa-onnx-cxx-api.dll` `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE`
- **payload 大小**：`gaea.exe` 54,916,608 B；三件 DLL 合计 23,847,736 B。
- **冒烟**：从正式交付路径
  `payload/gaea.exe` 启动，`/api/health` 返回 200
  （`{"status":"ok"}`），进程 PID 49028 已回收。
- **源码包**：`gaea-d30cb5d2-source.tar.gz`，由 `git archive` 从同一提交生成，
  前缀为 `gaea-d30cb5d2/`；未跟踪的工作区文件不会混入。
- **投递包**：`gaea-codex-distill-batch3-20261117-d30cb5d2.zip`，包含
  `payload/`、本说明及首版 `SHA256SUMS-d30cb5d2.txt`。
- **工作区排除项**：未跟踪文档
  `docs/codex-distill/hotspot-07-decompile-pipeline-and-c-findings.md`
  不属于本提交，未纳入源码包、构建或记录提交。
- **未装机**：本包仅完成桌面构建、正式路径冒烟与归档，未运行
  `gaea_install.ps1`，未覆盖 `C:\AI\gaea-app`。
