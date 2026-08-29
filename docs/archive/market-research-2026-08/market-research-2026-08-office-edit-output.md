# 市场调研：AI 办公「中期编辑 · 后期输出与预览」，2026-08

> 目的：为 gaea 通用办公的「中期 Word/Excel 编辑」与「后期文件输出/预览」提供设计依据。
> 背景：上一份《市场调研：通用办公 AI 智能体》已把 gaea 的「前期解析」闭环（format_convert /
> OCR / 事实底座），本调研覆盖竞品如何做 ① 原地编辑（框选即改、修订模式、Excel 单元格级操作）
> 与 ② 原生可编辑交付 + 所见即所得预览，并给出 gaea 的落地清单。

## 0. 一句话结论

2026 年 AI 办公竞争已从「生成一份文件」转向「在文件里干活」：竞品普遍用**框选即改 + 修订模式
（逐条接受/拒绝）+ 原生可编辑交付 + 保真预览**来兑现「少复制粘贴」。gaea 目前缺的正是这三块：
中期没有原地编辑体验（只能解压改 XML），后期输出是「技能级生成」而非「事实底座一稿多用」的统一
出口，预览是「转 Markdown 弱预览」而非保真渲染。中期编辑与后期预览是同一枚硬币：**谁先把
Word/Excel 的「可编辑 + 可预览」底座做好，谁就能承载 AI 编辑动作**。

## 1. 竞品矩阵

### 1.1 中期 Word 编辑（原地编辑，主战场）
| 产品 | 做法 | 关键细节 |
|---|---|---|
| 腾讯 WorkBuddy V5.3.5「人机双写」（2026-07-29 上线） | 联合腾讯文档，在 Word/Excel/PPT/Markdown 内框选内容 + 自然语言指令，AI 就地修改/生成/排版，改动实时落在原文件 | 业内唯一同时兼容本地 Office 文件与在线文档；框选区域局部修改、保持其余内容与版式不变；可随时接手微调或加批注让 AI 继续改；AI 共创内容可一键上传腾讯文档多人协同 |
| Microsoft Copilot Agent Mode（2026-04 全量） | Word 内 draft / rewrite / restructure / adjust tone，多步 app-native 动作直接改文档 | Excel 内直接加公式、表格、图表；Word/Excel/PPT 三件套打通，Agent 默认开启 |
| WPS 灵犀专业版（2026-07-15 发布） | 基于 WPS 三十年文档技术，原生级处理格式排版、表格公式、PPT 动效 | 文档内起草/润色/结构调整同步完成，结果留存于文件本身，格式不变形、版本可追溯；AI 改文字以修订模式呈现，用户逐条接受/拒绝 |
| Claude in Word（2026-04） | 右侧边栏给修改建议，每处改动以 Word 原生修订模式呈现（原文划掉、新内容标插入） | 用户逐条接受或拒绝；典型场景：合同审阅（NDA） |
| Praxim（YC 2026-08） | Agentic Word 编辑器，单条 prompt 全文档编辑，无需手动选中 | 输出真实 Word 格式（列表/表格/样式），不靠复制粘贴 |
| docx-editor（开源 1.0，2026-06） | 浏览器内 Word 编辑器（React/Vue），纯客户端 | 内置 AI Agent SDK：大模型直接操作文档（批注、修改追踪、自动审阅），流式输出同时修改落进编辑器 |

### 1.2 中期 Excel 编辑（单元格级操作）
| 产品 | 做法 |
|---|---|
| Microsoft Excel Copilot | 自然语言创建/修复公式、清理数据（重命名/拆分列）、生成透视表与图表、Python 统计预测、分析师代理跨文件模式 |
| WPS 灵犀表格智能体 | 输出**原生公式**（SUMIFS/XLOOKUP 等），推理过程嵌入可编辑结构，用户可检查/修改/扩展计算逻辑；数据勾稽关系清晰可校验；透视表/公式保留 |
| DuMate | Excel 数据透视表自动生成、Word 智能排版、PPT 模板化设计建议；Excel 图表嵌入 PPT 保持动态更新（跨应用联动） |
| 豆包专业版/办公任务模式 | 复杂表格自动处理、动态图表、长任务后台运行、产物可继续调整 |

### 1.3 后期输出（原生可编辑交付 / 一稿多用）
| 产品 | 做法 |
|---|---|
| WPS 灵犀 | 打通 WPS 底层原生 API，交付物都是**原生可编辑 Office 文件**（公式、透视表、动效保留），不是图片或文本框堆叠 |
| DuMate | 「统一事实底稿 → 四形态交付」：先研究形成事实底稿，再基于同一底稿输出 Word/PPT/网页/Excel，多形态彼此一致、可留存可复用 |
| GenOffice（Genspark，开源 2026-08） | 上层让 AI 生成 Markdown/HTML，下层**中间层转换引擎**把内容变成可编辑可交付的 Word/PPT——不要求模型啃最复杂的 OOXML |
| 模板化生成（社区技能生态） | Pandoc + python-docx + reference.docx 品牌模板；受控 Markdown/JSON → python-docx 确定性公文排版（字体/字号/行距/标题层级/页码）；Jinja2 模板目录 + headless Chrome 渲染 |

### 1.4 后期预览（所见即所得 / WYSIWYG 底座）
| 方案 | 定位 | 成本/约束 |
|---|---|---|
| docx-preview（纯前端库） | docx → HTML 浏览器渲染，轻量 | 只读预览为主，复杂版式有损；可 DOM 操作做高亮/批注 |
| Mammoth.js | docx → HTML 内容提取（CMS 场景） | 版式弱保真 |
| @vue-office/docx 等组件 | 前端 docx 预览组件 | 同上 |
| OnlyOffice Docs（开源自托管） | 实时协同编辑 + 强 Office 格式兼容 + 批注/修订 | 社区版个人免费（AGPL，≤20 用户）；可接 Ollama/LM Studio/OpenAI 兼容 API 做 AI 写文档；可当桌面/网页 WYSIWYG 编辑与预览底座 |
| Word Online / Office Online / Aspose | 官方/商用渲染 | 依赖微软云或商业授权 |
| docx-editor（开源） | 浏览器内 Word 编辑器 + AI Agent SDK | 纯客户端，无后端，文档不出浏览器 |

## 2. 行业共识：AI 文档编辑的六条关键范式

1. **框选即改、保持版式**：WorkBuddy/WPS 都强调「选中区域让 AI 局部修改，其余内容与版式不变」——
   原地编辑的前提是**范围限定 + 非目标区域零扰动**。
2. **修订模式 + 逐条接受/拒绝**：Claude in Word / WPS AI / docx-editor 都把 AI 改动落到原生
   Tracked Changes（原文划掉、新内容插入），用户逐条审阅。这是「AI 敢改、用户敢信」的信任机制，
   也是 AI 编辑与人工编辑无缝接力的核心（Adeu/changex 已把「按 ID 接受/拒绝/回复」做成库）。
3. **原生可编辑交付**：灵犀/DuMate 反复强调「不是图片堆叠、不是文本块拼凑」，公式、透视表、动效
   都要保留并可继续编辑——「可编辑」本身就是交付质量的一部分。
4. **表格公式可校验、可追溯**：灵犀表格智能体把推理过程嵌入可编辑结构，SUMIFS/XLOOKUP 原生落表，
   勾稽关系清晰可校验；Copilot 同样主张「公式建议 + 可检查」。
5. **事实底座一稿多用**：DuMate「统一事实底稿 → 多形态交付」已验证为行业范式，与 gaea 已落地的事实
   底座方向一致——差异在 gaea 还缺「从底座一键产出可编辑 docx/xlsx/pptx」的统一出口。
6. **中间层转换引擎，AI 不啃 OOXML**：GenOffice 明确「AI 生成 Markdown/HTML，中间层转可编辑
   Office」；gaea 的 create_docx.py（受控 Markdown → python-docx）已是同思路，缺的是**模板/样式
   体系与双向（编辑回写）能力**。

## 3. gaea 现状盘点（基于代码，2026-08-09）

### 3.1 前期解析（已闭环）
- format_convert（docmd）：docx/xlsx/pdf → Markdown，大文件已提速（docx 12x / PDF 4x）；
- 扫描件 OCR：OvisOCR2 常驻服务（llama-server）→ RapidOCR → WinRT → 本地视觉模型；
- 事实底座：fact_add/list/clear + 侧栏面板 + 一键沉淀长期记忆；
- 粘贴图片「提取文字/识图」双入口（Composer 附件按钮）。

### 3.2 中期编辑（近乎空白，只有原始手段）
- docx 技能：**解压 → 改 XML → 重新打包**（`Editing Existing Documents`），保真但极脆弱，
  无 UI、无范围限定、无修订模式；
- create_docx.py（python-docx）：**从零创建**能力强（封面/目录/标题层级/表格/页眉页脚/页码/图片），
  但不支持「打开已有文档原地改」；
- xlsx 技能：openpyxl 写单元格/公式 + LibreOffice 重算（recalc.py 已修 Windows 路径探测），
  但同样无 UI、无「选中区域 → 指令」的交互；
- 方案编写模块：结构化章节编辑（ProposalSection）是**结构化编辑器**，不是 docx 原地编辑。

### 3.3 后期输出（技能级生成，缺统一出口）
- 方案导出：ProposalExportDocx / ExportMD（gooxml 排版，封面/目录开关、暗标规则）；
- 通用办公：对话交付物以 Markdown 为主，导出按钮仅 `downloadMarkdown`；
- 技能级生成：docx（create_docx.py）、pptx（create_pptx.py）、xlsx（openpyxl+recalc）、pdf——
  各自独立，与事实底座没有统一的「一稿多用」编排入口；
- 无模板库（reference.docx 风格模板、品牌样式）、无「事实底座 → 多形态交付」的一键管线。

### 3.4 后期预览（弱预览，无 WYSIWYG）
- GaeaPreview：docx/xlsx/pdf → Markdown/文本的**弱渲染**，图片/表格内联，版式不保真；
- 对话内文件预览弹层、mermaid PNG 渲染、图片预览已有；
- 无 docx-preview / OnlyOffice / docx-editor 这类保真渲染底座，无 Excel 单元格级渲染，无轻编辑能力。

## 4. 差距分析（gaea vs 竞品）
| 能力维度 | 行业标杆做法 | gaea 现状 | 差距等级 |
|---|---|---|---|
| 原地编辑（Word） | 框选即改 + 修订模式逐条接受/拒绝，其余内容版式不变 | 无 UI；仅解压改 XML | **P0** |
| 原地编辑（Excel） | 选中区域/列 → 公式/清洗/透视表，原生公式可校验 | openpyxl 写入 + recalc，无 UI 无指令式操作 | **P0** |
| 原生可编辑交付 | WPS/DuMate：公式、透视表、动效保留 | 技能级生成可编辑文件，但无模板/样式体系 | P1 |
| 一稿多用统一出口 | DuMate 事实底稿 → Word/PPT/网页/Excel | 事实底座已建，缺统一交付编排与模板 | **P0** |
| 保真预览 | docx-preview/OnlyOffice/docx-editor WYSIWYG | 转 Markdown 弱预览 | **P0** |
| AI 审阅信任机制 | 修订模式 + 按 ID 接受/拒绝（Adeu/changex 已成库） | 无 | P1 |
| 跨应用联动 | DuMate Excel 图表嵌入 PPT 动态更新 | 无 | P2 |
| 协同/多人 | WorkBuddy 人机多端协同 | 明确不做（个人工具） | 不做 |

## 5. 设计决策与强化清单（映射 gaea）

### P0 中期编辑与后期预览（同一底座，先做）
1. **保真预览/轻编辑底座**：引入 docx-preview（纯前端，docx → HTML 保真渲染）或
   OnlyOffice 社区版（自托管 WYSIWYG + 批注/修订），xlsx 用 SheetJS/exceljs 转 HTML 渲染
   单元格与公式；对话/方案里的文件预览从「Markdown 弱渲染」升级为「版式保真 + 可框选」。
2. **框选即改（Word）**：在保真预览上支持选中段落/表格区域 → 自然语言指令 → AI 就地改。
   落点：先做「选中 → 改写/扩写/润色/翻译」的确定性子集（走本地/在线模型 + 结构化 diff），
   改动以**修订样式**呈现，逐条接受/拒绝；避免一开始就啃完整 OOXML。
3. **Excel 单元格级操作**：选中区域/列 → 指令（生成公式、清洗去重、拆分列、透视表、图表）；
   复用 openpyxl 写原生公式 + LibreOffice recalc 校验（已具备），把「结果可检查、公式可编辑」
   作为交付标准。
4. **一稿多用统一出口**：事实底座 → 一键交付 docx/xlsx/pptx 的编排管线（模板化）：
   建 `reference.docx` 风格模板库（公文/报告/合同），create_docx.py 从「受控 Markdown/JSON」
   确定性排版（已有雏形，补模板与页眉页脚/目录/页码参数化）；对话成果默认支持 docx 导出。

### P1 深化
5. **修订模式信任机制**：把 AI 编辑落到 Tracked Changes（参考 Adeu/changex 的按 ID 接受/拒绝），
   与人工编辑无缝接力；diff 可回看。
6. **模板与样式体系**：品牌模板（reference.docx + 样式映射表）+ 素材库联动（业绩/人员/设备表）。
7. **跨应用联动**（远期 P2）：Excel 图表/数据嵌入 Word/PPT 保持动态更新。

### 明确不做
- 多人实时协同、共享智能体、组织权限（个人工具定位，同上一份调研）；
- 云端 SaaS 常驻与商业授权绑定（OnlyOffice 企业版授权、Office Online 依赖微软云均不引入）；
- 让模型直接生成/编辑原生 OOXML（保留「中间层转换引擎」思路，AI 只出受控 Markdown/JSON/指令）。

## 6. 落地优先级建议
- **本轮优先（P0）**：① 保真预览底座（docx-preview 纯前端先行，OnlyOffice 自托管作为后续可选）；
  ② 框选即改（Word 选中 → 指令 → 修订式就地改）；③ Excel 单元格级操作（公式/清洗/透视，公式可校验）；
  ④ 事实底座 → 多形态交付统一出口（模板库 + 一键导出 docx/xlsx/pptx）。
- **其次（P1）**：修订模式接受/拒绝 + diff 回看；模板与样式体系；对话成果默认 docx 导出。
- **明确不做**：多人协同、云常驻、OOXML 直出。

## 7. 结论

gaea 前期解析已形成护城河（本地优先、离线 OCR、事实底座）；下一阶段的价值全部集中在
「**在文件里干活**」：中期框选即改 + 修订式信任机制，后期保真预览 + 原生可编辑交付 +
事实底座一稿多用统一出口。技术选型上，纯前端 docx-preview 先行、OnlyOffice 自托管为进阶、
openpyxl+LibreOffice 承接 Excel 原生公式与重算、受控 Markdown→python-docx 承接模板化输出——
与 gaea「本地优先、零外部依赖」的定位一致。
