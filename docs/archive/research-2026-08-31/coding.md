# 5. 编程板块（v4.11.0 基线复扫 · 2026-08-31）

> 调研方法说明：本轮通过 GitHub API 实测仓库活跃度（星标/最后推送日期）、直接抓取官网与 CHANGELOG 核实，中文搜索引擎结果受合规过滤较多，个别项标注「未核实」。调研日期 2026-08-31。

## 市场格局 · 最新动态

- **收敛为「一个底座、多个表面」**：终端 CLI agent（Claude Code、Codex CLI、OpenCode、Gemini CLI 等）成为事实底座，IDE（Cursor、Devin Desktop、Trae）与异步云端围绕其分层。GitHub 于 2026-02-04 起 Agent HQ public preview 直接接入 Claude 与 Codex，2026 年 6-7 月又追加 agent apps、Issues 内 agent 自动化控制——平台枢纽开始聚合第三方 agent（[1][2]）。
- **异步云端常态化**：OpenAI Codex 已五面一体（ChatGPT 桌面/Web、CLI、IDE 扩展、cloud、Remote），支持长任务、通知、由 Gmail/Slack/GitHub 事件触发的定时任务（2026-08），agent harness 开源（[7]）。Cognition 方面，windsurf.com 已 308 重定向至 devin.ai/desktop（2026-08-31 实测），Devin Desktop 定位「coding agent 之家」、可导入 VS Code/Cursor 配置；Windsurf 独立品牌是否完全退役**未核实**（[3][4]）。
- **中国侧**：Trae 拆分为 TraeCode + TraeWork（AI 办公平台），积分制四档约 ¥45-699/月，内置 Seed/GLM-5.2 模型，云端任务并行 2-20 个（[20][21]）；通义灵码免费、Lingma IDE 全面公测、未见 CLI 形态（[22]）；Kimi 推出 Kimi Code 订阅；围绕 Claude Code/Codex 的 API 中转站生态（约官方价 10-38%）成为壳窗软件的主要赞助来源（[9]）。

## 范式迁移（上轮调研以来的变化）

1. **「壳窗 + 外部 CLI」从民间黑箱走向官方协议**。Claude Code v2.1.x CHANGELOG（2026-08 在查）显示官方提供 `claude attach / logs / stop / respawn / rm` 后台会话管理、Remote Control 客户端实时流式工具调用、SessionStart 钩子返回会话 staleness 与重缓存成本、`/usage` 花费限额条、Claude Desktop 跨会话消息投递（[8]）。第三方壳窗的定位从「破解式包装」变为「官方接口的桌面客户端」，技术风险显著下降。
2. **壳窗供给繁荣但商业模型脆弱**。活跃样本：cc-switch（13.0 万星，Tauri，管理 8 种引擎）、claudecodeui/CloudCLI（1.35 万星，多引擎+手机/Web）、CodePilot（归藏，6.4 千星，17+ 供应商、手机遥控）、desktop-cc-gui、codeg、Nimbalyst（Crystal 后继，个人免费+iOS 伴侣）、Happy、Conductor（Mac 专属，$50/月 SaaS 化+企业版）（[9]-[17]）。收缩样本：opcode（2.24 万星）自 2025-10 停更；Crystal 停更；Vibe Kanban 公司运营关停、转社区开源（[15][16][23]）。存活路径共性：多引擎聚合、免费开源靠赞助、并入更大的个人助手（OpenClaw 38.8 万星，[18]），或 SaaS 化收云费用。
3. **「管家式」体验有直接对标**：会话恢复（Nimbalyst 原生 resume 全历史 [14]；claude-mem 9.3 万星做跨会话持久上下文 [19]）；完成通知/远程遥控（Happy [17]、Nimbalyst iOS、CodePilot 手机控制 [11]、Conductor mobile 官网标注 coming soon [16]）；健康总览（Conductor「一眼看到各 agent 在做什么」、Codex 通知与长任务 [7]）。

## 对 gaea 的机会与威胁

- **决策验证**：2026 年格局下「编程板块保持 DSH 壳窗、不并入工位、不共享工具面、不做原生工作台」仍然成立且更稳——官方协议化让壳窗更可靠；同时 opcode/Crystal/Vibe Kanban 三个停更案例证明独立壳窗的商业与维护风险真实存在。gaea 只做体验层、不承担引擎维护成本，符合收敛后的产业分工。
- **机会**：现有壳窗都在卷「多引擎并行与切换」，面向个人用户的「低焦虑管家层」——健康徽标、断连自愈、完成通知进入通知中心、与个人记忆弱联结——仍是空位；gaea 的中文办公场景与「工位/乐园」隔离可延伸出「编程会话不污染办公空间」的差异点。
- **威胁**：①官方客户端上移，Claude Desktop 已能跨会话投递/回复 Claude Code 消息、Codex 与 ChatGPT 桌面一体化，官方壳会挤压第三方壳的存在感（[7][8]）；②注意力分流，中文圈 10+ 免费壳窗与 OpenClaw 类个人助手直接竞争「个人助手管编程」的心智（[9][11][18]）；③中转站 + cc-switch 组合使用成本极低，编程板块若无管家价值难以留人（[9]）。

## 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **现在 0-3 月（体验加固对标官方语义）**：断连自愈对齐 `claude attach/respawn` 与 `--resume` 行为；外链唤起打通 deeplink→终端/CLI；健康徽标读取进程存活 + SessionStart staleness 钩子；复述并冻结「不并轨、不共享工具面」决策（[8]）。
- **下个 3-6 月**：任务完成系统级通知（对标 Happy / Nimbalyst iOS 伴侣）；只读会话列表与一键恢复入口；适配清单扩展至 Codex CLI、OpenCode、Gemini CLI 等多引擎（[9][14][17]）。
- **愿景 6-12 月**：完成事件进入 gaea 通知中心并与个人记忆弱联结；观望 Remote Control 协议是否成为壳窗标准接口；仍不做原生编程工作台。

### 未核实项
- GitHub Copilot Agent HQ 的 2026 年采用数据；Claude Code 是否有官方桌面 GUI（未搜到，但 CHANGELOG 已见 Claude Desktop 与其会话互通）；Windsurf 品牌退役程度；CodeBuddy 2026 现状（官网抓取为空）。

## 参考来源

1. https://github.blog/news-insights/company-news/pick-your-agent-use-claude-and-codex-on-agent-hq/ （2026-02-04）
2. https://github.blog/news-insights/company-news/welcome-home-agents/ （2025-10-28）
3. https://devin.ai/desktop （另：windsurf.com → devin.ai/desktop 308 重定向，2026-08-31 实测）
4. https://docs.devin.ai/zh/desktop/getting-started
5. https://claude.com/pricing （Pro $17-20/月、Max $100 起、Claude Code 全付费计划含、5 小时滚动+周限额、Managed Agents $0.08/session-hour）
6. https://cursor.com/pricing （Hobby 免费 / Pro $20 / Pro+ / Ultra 20x / Teams $40）
7. https://learn.chatgpt.com/docs （Codex 五表面、事件触发定时任务、开源 harness）
8. https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md （v2.1.251）
9. https://github.com/farion1231/cc-switch （13.0 万星，ccswitch.io，API 中转站赞助生态）
10. https://github.com/siteboon/claudecodeui （CloudCLI，多引擎，pushed 2026-08-27）
11. https://github.com/op7418/CodePilot （6.4 千星，17+ 供应商，BSL-1.1）
12. https://github.com/zhukunpenglinyutong/desktop-cc-gui （Tauri 多引擎桌面端）
13. https://github.com/xintaofei/codeg （多 agent 会话聚合工作区）
14. https://nimbalyst.com/crystal/ （Crystal 停更、Nimbalyst 后继，个人免费）
15. https://www.vibekanban.com/ （公司运营关停、转社区开源，2.8 万星）
16. https://conductor.build/ 及 https://conductor.build/pricing （Free/$50 Pro/$60 Teams/企业）
17. https://happy.engineering/ （Claude Code & Codex Remote Control）
18. https://github.com/openclaw/openclaw （38.8 万星，pushed 2026-08-31）
19. https://github.com/thedotmack/claude-mem （9.3 万星，跨会话持久上下文）
20. https://docs.trae.cn/ide_plans-and-billing （积分制 ¥45-699/月）
21. https://work.trae.cn/ （TraeWork AI 办公平台）
22. https://lingma.aliyun.com/ （通义灵码免费、Lingma IDE 公测）
23. https://github.com/winfunc/opcode （2.24 万星，pushed 2025-10-16，停更）
