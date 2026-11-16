# gaea-wenxue-v3-20261116-7e0dffde

- **源提交**：`7e0dffde` feat(wenxue): 图文五批——本地引擎预检/角色库导入/工具瘦身/记忆中枢接入/记忆面板恢复+生图面
- **前置提交**：`666f239d` 三标签页重设计（另一执行体批次代提交）
- **打包方式**：独立干净工作树（detached @7e0dffde，junction node_modules），
  前端 Vite 44.65s → `wails build -s -ldflags "-s -w" -trimpath`（bindings 未跳过，
  ucrt64 gcc + CGO_ENABLED=1）
- **内容**：payload/ = gaea.exe + onnxruntime.dll + sherpa-onnx-c-api.dll +
  sherpa-onnx-cxx-api.dll（同目录四件）
- **冒烟**：`scripts\smoke.ps1 -ExePath releases\...\payload\gaea.exe` →
  `/api/health 200 {"status":"ok"}`，冒烟进程已回收
- **批次内容速览**：每轮本地引擎预检+10min 图文租约 / wenxue_character_import
  角色库全字段落卡 / 工具面板 55→43（boot.addGated 全注册点过闸）/ 记忆中枢
  第九库「图文记忆」（只读跨书）/ 图文脑记忆面板第八 tab 恢复 + image_gen
  生图面修复（wxWorkSpaceTool）
- zip/tar 留盘不入库（惯例）；SHA256SUMS 覆盖 payload 四件+zip+tar
