# 生产化评估集（1.x 真实修复点与踩坑样本）

> 用途：2.0 迭代的能力验收基线；样本来自 1.x 真实修复点与代码注释。
> 采集：2026-08-07 重建（源自 1.x git log 与代码注释）。状态列说明该样本在 v1.21.0 的落地方式。

| 编号 | 来源 commit | 现象/踩坑 | 修复要点 | v1.21.0 状态 |
|------|-------------|-----------|----------|--------------|
| E01 | 8851864（不可达） | ComfyUI 工作流参数顺序 flaky | 参数顺序稳定 | 行为在 `internal/ai/image_comfyui.go`，已转回归测试 |
| E02 | a52c6ce（不可达） | 角色注入 prompt 夹带剧照 base64，摘要抽取"我"等代词 | 剔除 base64、过滤代词 | 待后续阶段在 1.x 对应代码验证 |
| E03 | da2d5b8（不可达） | 章节创作回退全局模型而非功能绑定模型 | 使用功能级模型绑定 | 待 P1 模型路由阶段验证 |
| E04 | c8a1372（不可达） | xAI OAuth loopback 换 token 返回 500 | 明确 500/403 错误 + 超时客户端 | 已转回归测试（oauth_flow_test.go：500/403/verifier 缺失/discovery 配置化） |
| E05 | fee1eaa | 扫描件/转换诊断无法走 OCR | OCR + AI 工作台 | 已有实现，待后续阶段补回归 |
| E06 | 9b33966 | 招标解析合并/数组解析缺陷 | 解析合并与数组解析修复 | `internal/office/proposal/parse.go`，已转回归测试 |
| E07 | 9c8a2e1 | ComfyUI 孤儿实例 Errno 22 | 孤儿实例自动恢复 | 已有实现，待后续阶段补回归 |
| E08 | e03e63b | ComfyUI 生成失败（LoRA 文件名硬编码） | LoRA 列表动态读取 | 已有实现，待后续阶段补回归 |
| E09 | 18cc74d | 方案章节生成误用全局模型 | office_handler 接入功能级绑定 | 待 P1 模型路由阶段验证 |
| E10 | 6449121 | 办公 chat 误走活跃引擎导致 404 | bridge 注入功能级绑定 | 待 P1 模型路由阶段验证 |
| E11 | c4c703f | 工具操作目录不跟随工作空间 | 工具目录跟随工作空间 | `internal/gaea/tool/builtin/workspace.go`，已转回归测试 |
| E12 | 47a7571 | 工作空间无法新建/切换 | 系统目录对话框 + 路径持久化 | 已有实现，待后续阶段补回归 |
| E13 | ecc3573 | xAI OAuth 登录失败（referrer 误改） | referrer 改回 wubigork | 已有实现 + 回归（TestBuildAuthURL 断言 referrer=wubigork） |
| E14 | 25bcbb9 | FTS5 索引与事实表不同步 | 独立表 + 显式全量 rebuild | 已有实现，待后续阶段补回归 |
| E15 | efd6f9d | KG 三元组持久化缺失、Query 误命中 | KG 贯通 hermes.db + 并发安全 | 已有实现，待后续阶段补回归 |
| E16 | 1c1c746 | QUICK_REPLIES 常量误插组件体内 | 常量移至模块顶层 | 已核实修复 + 回归守卫（scripts/frontend-e-check.mjs） |
| E17 | bd95f58 | 微信助手固定叫"轻语"不跟随自定义名字 | 使用自定义名字 | 待后续阶段验证 |
| E18 | ea6fc2e | 微信消息字段不匹配官方协议 | 字段对齐 openclaw-weixin | 待后续阶段验证 |
| E19 | 41d2a4a | 微信无回复且 token 被脱敏 | 会话过期透出 + 防脱敏保存 | 待后续阶段验证 |
| E20 | 7ed7124 | 微信无回复（BotID 未传递） | 透传 BotID | 待后续阶段验证 |
| E21 | 2b6cc9b | 工作空间对话框因配置测试污染静默失败 | 测试隔离修复 | 已修复同类问题（app_info.go findChangelogPath cwd 优先） |
| E22 | b195473 | AI 控制台打不开（降级态 CSS 入场动画） | 降级态禁用自定义 CSS 动画 | 已核实修复 + 回归守卫（index.css .ai-console-panel 规则） |
| E23 | 2e08a0d | 功能绑定下拉打不开（WebView2 节流） | rAF/CSS 动画节流降级 | 已核实修复 + 回归守卫（main.tsx ensureRAF + index.css antd motion） |
| E24 | 93b4ae3 | 记忆中枢白屏（3D 图谱误装底层库） | 换用 3d-force-graph | 已核实修复 + 回归守卫（GraphView 用 3d-force-graph） |

## 可达性说明

- E01/E02/E03/E04 的来源 commit（`8851864`、`a52c6ce`、`da2d5b8`、`c8a1372`）不在 v1.21.0 可达历史中（2.0 期 commit），按现象保留；对应行为在 v1.21.0 代码中验证。
- 其余来源 commit 均在 v1.21.0 可达历史中，可用 `git cat-file -e <commit>^{commit}` 复核。
