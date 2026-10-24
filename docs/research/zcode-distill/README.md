# ZCode 蒸馏 · 子报告索引

> 2026-10-24 · 对象 `zai-org/ZCode@3c5b227`（v3.16.2）
> 主文档：[`../../zcode-office-distill-2026-10.md`](../../zcode-office-distill-2026-10.md) · 分析篇：[`../../zcode-office-distill-analysis-2026-10.md`](../../zcode-office-distill-analysis-2026-10.md)
> 全部只读调研，未改 ZCode 与 gaea 任何代码。

8 份子报告是"证据层"，主文档是"结论层"。子报告允许冗长与不确定（含"未找到"章节），主文档只留收敛后的判断。

| # | 文件 | 覆盖 | 一句话 |
|---|---|---|---|
| S1 | [s1-agent-runtime.md](s1-agent-runtime.md) | ZCode 运行时内核（821 行 / 33 机制） | 权限 15 层判定链与 compact 四件套值得看；shell/git/多宿主是负债 |
| S2 | [s2-tooling.md](s2-tooling.md) | 工具层契约（561 行） | 价值在"工具是执行单元"四件套，不在文件 IO 实现；Office 的 stale 必须改成投影版本 |
| S3 | [s3-genui-ux.md](s3-genui-ux.md) | GenUI 与交互面（15 条候选） | 上下文条/审批 dock/交付物打开三处空白；工具卡与排队净增量很小 |
| S4 | [s4-extension.md](s4-extension.md) | 扩展模型（skills/agents/plugins/MCP/hooks） | `agents/*.md` 声明式子代理是最大缺口；清单→组件投影可抄，市场不可抄 |
| S5 | [s5-gaea-office-baseline.md](s5-gaea-office-baseline.md) | gaea 办公后端基线（353 行） | 双轨写盘 + 写盘不入证据链 + 模型无版式验证工具，三个严重缺口 |
| S6 | [s6-gaea-office-ux.md](s6-gaea-office-ux.md) | gaea 办公前端基线（411 行） | 呈现件成熟度 ≫ 事件契约成熟度；补三个字段能解锁四类交互 |
| S7 | [s7-docx-pptx-plugin.md](s7-docx-pptx-plugin.md) | ZCode docx/pptx 插件逐文件拆解 | 管线=docx-js 生成 + Python 后处理 + 机检 + 视觉验收；postcheck 宣称 14 实跑 9 |
| S8 | [s8-xlsx-pdf-plugin.md](s8-xlsx-pdf-plugin.md) | ZCode xlsx/pdf 插件逐文件拆解 | 值得搬的是判据与阈值表（纯文本/纯算术），渲染重栈不要 |

## 证据强度约定

- 子报告内每条结论标 `[实读]` / `[文档]` / `[推断]`；未找到的写进各报告"未找到 / 未覆盖"章节，不补全。
- 主文档只引用 `[实读]` 或双源一致的结论；被证伪的初判（例如 `render_preserve.go` 与 docx 样式无关）已在 S7 报告中更正。
