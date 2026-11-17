# gaea-codex-distill-batch12-20261117-4be0ddc9

- **打包点**：`4be0ddc9`（功能提交 `f416e223` + Wails 模块再生提交）
- **构建树**：独立 detached worktree `.worktrees/pkg-4be0ddc9`，构建时工作树内容零漂移；
  `wailsjs` 仅显示 CRLF 状态噪声，`git diff --ignore-space-at-eol` 为空。
- **包面**：codex 蒸馏前两批 T1–T15 的办公/文学双板块收口，包括审批理由与追认、
  Guardian 双板块路由、章节差分会话边界、模型降档预压缩、工具环境与安全闸补件、
  角色级 `config_layer`。后续 F 批与 B 批未落地。
- **前端构建**：`npm.cmd run build`，Vite 27.94s；5 条既有 CSS minify 语法告警与
  大 chunk 告警不阻断构建。
- **Wails 构建**：UCRT64 GCC、`CGO_ENABLED=1`、系统默认 TMP/TEMP；
  `wails build -s -ldflags "-s -w" -trimpath` 56.045s，`Skip Bindings=false`。
- **版本**：4.544.0；exe raw FileVersion/ProductVersion = 4.544.0.0（未抬版本）。
- **payload SHA256**：
  - `gaea.exe` `03455E79F7560AD8F8DD862635691A908D0074BC378D13CFFCA50F5040FA54D2`
  - `onnxruntime.dll` `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B`
  - `sherpa-onnx-c-api.dll` `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8`
  - `sherpa-onnx-cxx-api.dll` `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE`
- **冒烟**：独立 worktree 的 `build/bin/gaea.exe` 受本机策略拦截（WebView2
  `800700aa`）；同一构建四件复制到正式 payload 路径后，
  `scripts/smoke.ps1` 返回 `/api/health` 200，PID 55368 已回收；同一四件同步到
  主树 `build/bin` 后再次冒烟 200，PID 24580 已回收。
- **源码包**：`gaea-4be0ddc9-source.tar.gz`，由 `git archive` 从同一提交生成。
- **投递包**：`gaea-codex-distill-batch12-20261117-4be0ddc9.zip`。
- **未装机**：仅完成桌面构建与正式路径冒烟，不覆盖 `C:\AI\gaea-app`。
