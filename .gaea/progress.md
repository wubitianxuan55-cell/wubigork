# 任务进度

> 最后更新 2026-08-12（v2.13.1 发布）

## 已发布：v2.13.1「修复 @PDF 引用注入二进制」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | @引用 office 文档转换失败时注入占位提示，不再把 %PDF-1.4 原始字节塞进模型上下文 |
| ✅ | 二进制检测扩展到整个读取窗口（此前只查前 8KB，PDF 头无 NUL 会漏检） |
| ✅ | 发布：v2.13.1 版本号、releases/v2.13.1.md、CHANGELOG、gaea-v2.13.1.exe、git tag v2.13.1 |

## 已发布：v2.13.0「通用办公打磨 + 模型中心优化 + 本地模型实测」（2026-08-12）

| 状态 | 任务 |
|------|------|
| ✅ | 方案分节字数修复：流式「生成该节」+ 批量生成按 WordTarget 续写（原硬编码 100 字截断） |
| ✅ | docx 读取乱码修复：ReadTextFile 统一走 ConvertToMarkdown |
| ✅ | 前端运行态兜底：停止按钮无条件复位 + 30s GaeaRunning 看门狗（turn_done 丢失不再卡「执行中」） |
| ✅ | 待办自动收尾：从未推进的列表 turn_done 后显示已完成 |
| ✅ | format_convert 输出父目录自动创建 + 回归测试 |
| ✅ | docmd MarkItDown 通道（docx/pptx）+ PDF 页数上限保持 |
| ✅ | 模型中心功能绑定/板块细节优化；本地引擎思考参数透传（enable_thinking / chat_template_kwargs） |
| ✅ | whisper web_search、角色画像文件化等累积修复 |
| ✅ | 本地模型实测：Herdsman 三模型 20 任务×思考/非思考双模式 + Ollama GLM；脚本 scripts/test_herdsman_models.go 可复现 |
| ✅ | 发布：v2.13.0 版本号、releases/v2.13.0.md、CHANGELOG、gaea-v2.13.0.exe、git tag v2.13.0 |

## 验证

- go build / go vet 干净；方案包（proposal）测试全绿
- 前端 127 例全过，tsc + vite build 通过
- 产物 gaea-v2.13.0.exe（约 39.6MB），桌面端同步

## 后续候选

- 成本库导入行分类映射到多级路径（当前导入落到一级节点，可在编辑时细分）
- 成本条目补充「地区 + 期数」字段（信息价三要素：规格 + 地区 + 时间）
- 发布版移除 WebView2 渲染调试端口（main.go 诊断配置，本地保留即可）
- exe 瘦身：Wails 构建压缩 / 前端大 chunk 按需加载
- 待办工具：任务完成时自动置 completed（当前靠模型回写状态，前端 turn_done 已兜底）
- tesseract 缺失时 OCR 错误提示更友好（当前靠本地视觉模型兜底）
