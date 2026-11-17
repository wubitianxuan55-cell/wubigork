# gaea-vision-ocr-gallery-20261118-b3d994dc

**打包点**：`b3d994dc`（detached 干净工作树 `.worktrees/pkg-b3d994dc`，junction node_modules）
**版本**：4.544.0（wails.json 不抬号——功能随下个发版点统一抬）

## 本包内容（2026-11-17~18 治理线全量）

- **办公识图本地 llama 档根修**：llamaEnsure 按 MMProj 装载 --mmproj（qwen35 档
  此前纯文本服务，识图跟随链把图喂给它必坏）+ managed 目录 caps=vision 能力
  标注 + 跟随链能力闸（纯文本档永不接图，自我愈合）
- **OCR 专属大卡**：docmd 进度钩子（三链路插桩：图片链/缓存命中/PDF 页粒度）
  → gaea-event ocr_progress → 转录流大卡（引擎拉起中/识别中 N/M 页/缓存命中/
  完成 N 字/失败原因；预览面板专用通道不受影响）
- **文学「插图」标签（第九 tab）**：WenxueGalleryList 图册（.gaea/uploads 生成
  图+工作空间其他图片，mtime 倒序 400 上限）+ WenxueGalleryHub（网格懒加载/
  来源筛选/lightbox/filetree 自动刷新）+ ChatTabs 九 tab 契约
- **U5 触发源徽标**：TurnStarted.Trigger 下发，WorkHeader 非用户回合亮归属
  （后台唤醒/子代理回合）
- **DSH 子代理蒸馏首批 D1-D6**：编队拓扑会话化（parentRef/spawnCallId meta
  直写+会话骨架+matchRun 退役）、折行布局、折叠聚合、aborted 结算词汇、
  单闸补齐（RunNew/runFollowUp）、运行中独立 Stop（GaeaSubagentStop）
- **编队 UI 完善批**：D8 双视图（FleetTree）、深度色带、节点详情浮层；
  走查两处几何 bug 根修（判列逐列迭代/步进口径/测量 content-box）

## payload（四件）

| 文件 | 字节 | SHA256 |
|---|---|---|
| gaea.exe | 55,064,576 | b9024b6d78deb6bb6ea432916fe2b8f91ae4484374f087130764ec1c9df47313 |
| onnxruntime.dll | 19,288,888 | 68d0d5f502be4509e2a16f932627a95137d748b2fae519027f80125c0b1f550b |
| sherpa-onnx-c-api.dll | 4,294,656 | 5dd36dbd3572846b72c4a40a4017c5a770160ee8e00da5053bebc8b2632501e8 |
| sherpa-onnx-cxx-api.dll | 264,192 | 4c56327796cdb980dcbf90025c28093d89ca286088fb74c3b107f2f633e67ffe |

（三 DLL 与基线哈希一致，未重编）

## 构建与冒烟

- 前端 `npm.cmd run build` 30.03s；UCRT64 GCC + `CGO_ENABLED=1` + 系统
  TMP/TEMP 裸 `wails build -s -ldflags "-s -w" -trimpath` 42.803s
  （Skip Bindings=false）；wailsjs 仅 CRLF 噪音（`--ignore-space-at-eol`
  内容漂移为空——再生物已随 b3d994dc 物化）
- 冒烟：正式 payload 路径 `scripts\smoke.ps1 -ExePath` → `/api/health` 200
  （PID 50912 已回收）
- 装机：`gaea_install.ps1 -SourceDir payload -Launch`（结果见 progress 记录）
- 未入库存放：源码 TAR/ZIP 留盘（P2-3 纪律）
- gaea-b3d994dc-source.tar.gz 43,727,334 B 3997df78ec98e059fb2fefc4c31a01740a828b8c9581963f01daa16934769f95
- gaea-b3d994dc-source.zip 12,821,240 B 5dc4f634f6066f33dde66fd60a3e7afdbbd178148a8d91337008bc01760831c7
