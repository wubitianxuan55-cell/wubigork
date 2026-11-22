# gaea-workbench-tabs-20261122-4d23703e

- 功能提交：`cf422218`（feat(ui): redesign right workbench tabs and panels）
- 最终提交：`4d23703e`（chore(wails): regenerate bindings for current DTOs）
- 独立干净工作树：`.worktrees/pkg-4d23703e`（检出 `4d23703e`）
- 前端依赖：`pnpm install --frozen-lockfile`；最终干净树复用同一依赖快照，不读取主工作树业务改动
- 前端构建：`npm run build`，`tsc -b` + Vite，38.31s
- 桌面构建：`CGO_ENABLED=1`、`CC=C:\msys64\ucrt64\bin\gcc.exe`、UCRT64 PATH 前置，系统默认 TMP/TEMP；`wails build -s -ldflags "-s -w" -trimpath`，52.275s，**Skip Bindings=false**
- 冒烟：正式 payload 路径 `/api/health` **200**，响应 `{"status":"ok"}`，PID 37464 已回收
- 交付内容：办公与文学右侧工作台页签/各面板 UX 重设计、文学插图页签图表与占位修正
- 隔离边界：未带入同一主工作树内的能力面板重构、F 批 Go 内核改动、临时试验脚本等未提交业务
- 生成物说明：`4d23703e` 已物化真实 DTO 漂移；构建后 Wails 包装文件与 `go.mod` 仅有 CRLF 噪音，内容零漂移
- 未装机：本轮只构建并打包，不覆盖 `C:\AI\gaea-app`

## SHA256

| 文件 | 字节 | SHA256 |
| --- | ---: | --- |
| payload/gaea.exe | 61,400,064 | `24817586DEB667DFC80BE75ECFE15B55ABD42DAC699D1C5642649DC1E88F9833` |
| payload/onnxruntime.dll | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| payload/sherpa-onnx-cxx-api.dll | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| gaea-workbench-tabs-20261122-4d23703e.zip | 35,123,360 | `CEBBCD58402C63EF93AFC27677D3EC36CF2642122C8BE097EF1B1967F5015ADA` |
| gaea-workbench-tabs-20261122-4d23703e-src.zip | 828,931,977 | `1829AE7C4D5A486ED000D7253E3A59C7DE5A4F78236A607B4B79805C3FC88F8E` |

三枚 DLL 与 `gaea-20261122-a4b5b0a7` 基线一致。源码 ZIP 由
`git archive --format=zip 4d23703e` 生成，共 **6,916** 个条目。
