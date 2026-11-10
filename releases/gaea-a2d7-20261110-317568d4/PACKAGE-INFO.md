# Gaea dsh-ops 全清收官预览包（A2+D7 上桌）

**包标识：** `gaea-a2d7-20261110-317568d4`

**性质：** 功能提交预览，不是正式版本号发布。

## 构建来源

- 提交：`317568d4c4f3f8b8e9fa8b7252bc72c12bec3c9b`（短哈希 `317568d4`）
- 提交说明：`docs(distill): A2+D7 销项收官——A2 两刀落地+D7 审计一刀两项不立项，候选池全清`
- 构建方式：独立干净工作树检出上述提交（`.tmp/pkg-a2d7`，构建后已回收），
  前端依赖 junction 复用主树 `node_modules`（同提交同 lock、主树零未提交
  改动），先 `npm run build` 再 `wails build -s -ldflags "-s -w" -trimpath`。
- 构建工具：Wails `2.13.0`、Go `1.26.10 windows/amd64`、Node.js `25.2.1`、
  UCRT64 GCC（`C:\msys64\ucrt64\bin`）、系统默认 TMP/TEMP。
- 构建时间：2026-11-10 18:44:44（Asia/Shanghai）。
- 冒烟在正式路径 `build\bin` 执行（AGENTS.md 纪律：不从 `.tmp` 运行 exe）。

## 本包增量（自上次装机 DE44AB5A400158A2 起）

- **A2 动态工具目录**：`[agent] dynamic_tools` 三态装配。缺省 `auto`——
  仅当 MCP 工具 schema 超 32KB 时把 `mcp__*` 转为可发现（经新工具
  `discover_tools` 词法匹配激活，下一模型步生效，可见性≠授权）；标准
  会话零变化（`off` 关闭，`on` 激进两级供 A/B 评测）。
- **D7-B 取消执行双闸**：用户按停止后，已流式收到的工具调用不再被派发
  执行（旧缺口：bridge 取消时干净关流被当正常收尾→写盘照发）；未执行
  调用回 `NOT_EXECUTED` 零效应可重发信封。
- D7 其余两项（作用域仲裁/变更归属传播）经对位审计判同构不立项，证据
  记录在 `docs/gaea-dsh-ops-distill-2026-11-10.md` §8.6。

## 使用方式

解压后 `gaea.exe` 与三枚 DLL 同级运行。缺任一 DLL 时语音链路不可用。
装机态同款已在 `C:\AI\gaea-app`（PID 36260，2026-11-10 18:46）。

## 文件校验

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `gaea.exe` | 61,903,360 | `A23083061DA4B394617CCD396F46BB997A7FCB56A6D5EB3D11D29E0F93A80F63` |
| `onnxruntime.dll` | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| `sherpa-onnx-c-api.dll` | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| `sherpa-onnx-cxx-api.dll` | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| `gaea-317568d4-preview.zip` | 31,636,929 | `11165E366B6B4BFE97AFE98049AF07662ACA71D4482ED8B8BC0C1449C53EF6F5` |
| `gaea-317568d4-source.tar.gz` | 38,803,097 | `E31208ECA6FFBCEBC2373561A71E72DDFB19D20208785CEA174E6CD109290CC7` |

统一校验见 `../SHA256SUMS-317568d4.txt`。

## 验证

- 干净提交的前端生产构建通过（23.9s）。
- Wails 桌面构建通过，bindings 未跳过。
- exe+三 DLL 在正式 `build\bin` 执行 `scripts/smoke.ps1`：
  `/api/health` 返回 `200`，响应 `{"status":"ok"}`。
- 装机：`gaea_install.ps1 -Launch` → `INSTALL OK`；新 PID 36260 启动
  18:46:28 > 构建完成 18:44:44（PID AFTER BUILD OK）；
  `C:\AI\gaea-app\gaea.exe` 哈希与构建产物一致。

## 版本边界

工作区既有版本源漂移（`app_info.go=4.531.0 / wails.json=4.535.0 /
versioninfo.rc=4.533.0`）未动；本包以功能提交号标识，不冒充正式版本号。
