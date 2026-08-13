# 任务进度

> 最后更新: 2026-08-14

| 状态 | 任务 |
|------|------|
| ✅ | 长期规划定稿（docs/superpowers/plans/2026-08-14-gaea长期规划-herdsman底座加固与工程门禁.md） |
| ✅ | H0-1 环境探测与兼容契约（internal/herdsman/probe.go + App.HerdsmanProbe） |
| ✅ | H0-2 服务健康检查（internal/herdsman/health.go + App.HerdsmanHealth） |
| ✅ | H0-3 TTS 默认模型动态解析（voice.ResolveHerdsmanTTSModel + voice_handler 接入） |
| ✅ | H0-4 LAN 暴露检测与告警（internal/herdsman/lancheck.go + App.HerdsmanSecurityCheck） |
| ✅ | H0-5 模型用途建议（模型库卡片 Hint）+ 思考模式 max_tokens 守护（bridge/ai 两路径） |
| ✅ | E1-1 前端 CI 修复（eslint 配置/插件、28 硬错误清零、CI 加 vitest、ComfyUI 路径文案） |
| ✅ | E1-2 发布冒烟脚本 scripts/smoke.ps1 |
| ✅ | E1-3 版本节奏转周更：v2.16.0 |
| ✅ | 发布文档：CHANGELOG / README / releases/v2.16.0.md / wails.json productVersion=2.16.0 |
| ✅ | 变更面逐包验证全绿（herdsman/app/bridge/ai/voice/tts；tsc；eslint 0 errors；模型库卡片 vitest 5/5） |
| ⏳ | wails build → gaea-v2.16.0.exe + SHA256SUMS + smoke.ps1 冒烟（本会话沙箱 vite 编译挂起，移交本机执行：`cmd /c build.bat`） |
