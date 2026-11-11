# Gaea 聊天板块 GenUI 根修预览包

**包标识：** `gaea-chatgenui-20261111-4d66818b`

**性质：** 功能提交预览，不是正式版本号发布（版本三源维持 4.541.0 对齐，
`wails.json` / `app_info.go` / `versioninfo.rc` 无漂移）。

## 构建来源

- 提交：`4d66818b`（含 `b2cf0117` GenUI 根修全量）
  - `b2cf0117` `fix(genui): 聊天板块 GenUI 不可用根修——单组件根+字段方言归一（对齐上游 dsh-genui）+围栏降级面`
  - `4d66818b` `fix(build): wails.json blob 修复——f60cd0af 抬号把 comments 写成 mojibake 并吞掉闭合引号，干净工作树检出即炸 Wails CLI JSON 解析`
- 构建方式：独立干净工作树检出 `4d66818b`（`.tmp/pkg-chatgenui`），前端依赖
  junction 复用主树 `node_modules`（package.json/lock 两提交间零差异），先
  `npm run build`（23.1s）再裸 `wails build -s -ldflags "-s -w" -trimpath`
  （1m06.6s），bindings 未跳过。
- 构建工具：Wails `2.13.0`、UCRT64 GCC（`C:\msys64\ucrt64\bin` 前置）、
  `CGO_ENABLED=1`、系统默认 TMP/TEMP。
- 构建完成：2026-11-11 23:13:43（Asia/Shanghai）；冒烟在正式路径
  `build\bin` 执行（AGENTS.md 纪律：不从 `.tmp` 运行 exe）。

## 本包增量（自上次装机 A23083061DA4B394 起）

- **聊天板块 GenUI 根修**：本地模型把单个组件直接当围栏根、条目用
  `{label,content}` 自然方言，旧 guard 信封分支早退 null → 围栏原地蒸发。
  修复=根带白名单 type 即按单组件根解释 + 字段方言归一（上游 dsh-genui
  SKILL 口径：label/name→title、description/content/text/body/detail→desc、
  仅正文→title）+ list `{title,desc}` 记录形态 + 围栏解析失败降级为可折叠
  原文（不再静默消失）+ Go genui_validate 同步（记录行不再误报）。
- **chatrule.md 提示词加固**：聊天系统提示注入根信封模板与高频条目字段名
  （本包已含，模型侧漂移源头收敛）。
- **wails.json blob 修复**：`f60cd0af` 抬号提交把 comments 写成 mojibake 并
  吞闭合引号，任何干净工作树检出该提交即炸 Wails CLI（`invalid character
  '\r' in string literal`）；逐提交二分实锤 5e2960e8 OK → f60cd0af BAD，
  本包回填磁盘已验证内容。打包纪律注：新工作树由 autocrlf=true 物化 CRLF，
  合法 JSON 下 CRLF 仅是行间空白，Wails CLI 正常解析。

## 使用方式

解压后 `gaea.exe` 与三枚 DLL 同级运行。缺任一 DLL 时语音链路不可用。
装机态同款已在 `C:\AI\gaea-app`（见「验证」节 PID）。

## 文件校验

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `gaea.exe` | 62,046,208 | `3FADA201E7950B187D6CE42765A692E2672C08F425F328650DA7382748013AED` |
| `onnxruntime.dll` | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| `sherpa-onnx-c-api.dll` | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| `sherpa-onnx-cxx-api.dll` | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| `gaea-4d66818b-preview.zip` | 31,679,010 | `547CEA0EBC3FAA207B3EAB1F576EB8AAFC7EA55522537CC5F7CCB2398DFB4F00` |
| `gaea-4d66818b-source.tar.gz` | 38,875,315 | `8C01AD02EB43EB720C2FA7789E4F5DF880E068B1EDBAF1368497BD94B41BAAD5` |

统一校验见同目录 `SHA256SUMS-4d66818b.txt`。

## 验证

- 干净提交的前端生产构建通过（23.1s）。
- Wails 桌面构建通过，bindings 未跳过。
- exe+三 DLL 在正式 `build\bin` 执行 `scripts/smoke.ps1`：
  `/api/health` 返回 `200`，响应 `{"status":"ok"}`，冒烟进程已回收（PID 48372）。
- 三枚 DLL 与上包（A23083061DA4B394）SHA256 逐字节一致，语音链路零变化。

## 版本边界

版本三源（`wails.json` / `internal/app/app_info.go` / `versioninfo.rc`）
均为 `4.541.0` 且 JSON/常量合法；本包以功能提交号标识，不冒充正式版本号。
`Same-Turn Fence Steering`（上游围栏校验失败同轮回喂）挂池未立项。
