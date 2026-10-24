# gaea 办公板块 CAD 图纸识别功能·市场调研（2026-10）

- 日期：2026-10-23
- 方法：四路并行网络调研（国内竞品 / 海外竞品 / 技术路线 / VLM 图纸理解能力），WebSearch + 厂商官网/GitHub 抓取，时间窗 2024~2026-10，关键结论均挂来源 URL（见各节与文末索引）。
- 需求方拍板前提：办公板块**可选使用**的功能模块（opt-in 卡片），不默认启用。
- 前置约束（沿用既有拍板）：不做 MCP/平台化/跨设备同步；领域规范知识一律技能化不进内核（CAD 领域知识同理）。
- 现有资产基线：识图管线 `internal/gaea/vision`（Provider 注册表，OpenAI 兼容视觉端点；本地 herdsman Qwen3.6-35B 多模态主力，模型中心「识图」绑定可动态切云端引擎）；造价域清单/定额/组价库；AI 组价 v4.2 在产。

---

## 一、结论摘要（给拍板用）

1. **赛道热度已被验证**：海外「AI takeoff/图纸识别」2025 下半年起融资提速（Beam AI $30.5M B 轮、xBuild $19M 且 a16z 入场、Bluebeam 集成 Claude）；国内广联达 GTJ2026 宣称识别率 90%/AI 提量 98%，并把图纸解析能力独立成商品（IQcad 睿图 API）。方向本身不缺市场。
2. **能力分界线清晰（关键情报）**：VLM 读图纸**文字/表格/标题栏/说明可信**（准确率 0.85~0.95），**计数/尺寸读取不可信**（门 39%/窗 34%，尺寸有幻觉实证）。所以 gaea 能做且值得做的是「图纸信息提取 + 图纸问答」，不是「看图算量」。
3. **差异化空档**：国内三强（广联达/鲁班/品茗）全部押在「重建模算量」，年租 6,700~9,000+ 元、只租不卖；「轻量提取表格→直接进清单组价」「对着图纸问」这类碎片场景无人覆盖，恰好接 gaea 的清单/定额库与识图管线。**避坑红线：不做全构件算量建模**——GTJ 的护城河是 20 年规则库+校核体系，识别只是入口。
4. **技术路线可行**：Go 生态 2026 年出现变量 `unitedrhino/go-cad`（Apache-2.0，纯 Go 读写 DWG R9~R2018/DXF + PNG/SVG 输出），需 PoC 验证、LibreDWG 子进程兜底（GPL 仅可子进程调用，严禁 cgo 链接）；前端渲染用 vagran/dxf-viewer（MPL-2.0，WebGL，可嵌 React）；语义层复用现有 vision Provider 管线。混合路线一期量级约 2~4 人月。
5. **桌面本地处理是卖点**：房企/设计院图纸不愿上第三方云（万翼/广联达均为云端或私有化部署），Wails 单机本地解析+可选本地引擎推理，天然契合数据敏感诉求。

---

## 二、需求侧：gaea 用户与场景契合

gaea 的双场景用户（办公文档人群 + 造价人员）对「CAD 图纸」的真实高频需求，按调研证据排序：

| 场景 | 说明 | 竞品覆盖度 |
|---|---|---|
| 图纸查看 | 造价/办公人群常需打开 DWG/DXF 但未装 CAD | 免费查看器多（ Autodesk TrueView 等），非痛点 |
| 图纸文字/表格提取 | 门窗表、钢筋表、做法表、设计说明 → 结构化数据 → 进清单/组价 | 广联达 IQcad 做成 API 商品但面向企业；**桌面轻量工具空档** |
| 图纸问答 | 「这张图什么结构」「这页做法表说了什么」「找一下三层平面图」 | Togal.CHAT/Kreo Caddie/Bluebeam Max(Claude) 正成为海外标配；国内 AecGPT 等刚起步、无高准确率商品 |
| 图纸归档/整理 | 图号、标题栏、图名提取，批量重命名归档 | smartscissor（广联达预览版）等零散；轻量可做 |
| 看图算量（计数/尺寸→工程量） | 海外 AI takeoff 主战场 | **VLM 能力不达标 + GTJ 类护城河深，不建议碰** |

---

## 三、国内竞品格局

### 3.1 造价算量类（识别能力最成熟的一极）

| 厂商/产品 | 识别对象 | 技术路线 | 价格/模式 |
|---|---|---|---|
| 广联达 GTJ2025/2026 | 轴网、柱（大样/柱表）、墙、门窗洞、梁（集中/原位标注）、板筋、基础钢筋、预制构件；GTJ2026 新增地连墙；格式 dwg/dxf/pdf/cadi2/gad | 人机交互「提取图层→识别→校核」的 CAD 实体规则解析为主，2026 版叠加 AI | **网络锁年租约 9,000 元/年，只租不卖**；宣称导入率 99%/识别率 90%/土建 AI 提量 98% |
| 广联达 IQcad 睿图 | 建筑平面图构件级 AI 识别（墙/门/窗/洞/楼梯/电梯）、房间识别、设计说明文本与表格结构化、dwg↔json | AI+规则混合，云服务+API（6 步流程）；要求图纸先「规范化预处理」 | 未公开，联系商务 |
| 广联达 QuantifAI | 图纸→算量报表一站式 AI（2026-09 服贸会发布，面向东盟/全球） | 自研 BIM+产业 AI | 未公开；宣传提效 6-8 倍 |
| 鲁班软件 | 钢筋平法 G101、土建构件；「智能算量大师 2026」主打 AI 一键识别图元 | CAD 转化（规则解析）为主，2026 融合 AI | 未公开标价；旧口径安装算量年费约 6,700 元 |
| 品茗 HiBIM/茗算 | HiBIM：Revit 上 CAD 翻模（V4.3.2 起免提取图层一键 AI 识别）；茗算：Rcad 识别图形/文字/表格 | AI 识别 + 自研 Rcad 规则解析 | 未公开（询价制） |
| 神机妙算/未来软件 | CAD 底图描图+模板宏为主 / 计价软件 | 无成体系 CAD 自动识别 | — |

**成熟度判读**：二维结构建模算量识别已到平台期（20 年打磨、90% 官方口径），2025-2026 新竞争焦点是「AI 一键翻模免提取图层」与「AI 提量」。**表格/文字结构化已被广联达单独商品化（IQcad API）**，证明该能力有独立付费价值。

### 3.2 AI 审图/图纸理解类

| 厂商 | 能力边界 | 规模/商业模式 |
|---|---|---|
| 万翼科技 AI 审图（万科系） | 住宅建筑/结构专业规范审查（300+ 规则）、跨文件一致性、结构计算书参数审查；机电专业覆盖弱（第三方横评） | 累计审图 32 万+ 张、58 万+ 问题、90+ 家房企/设计院/政府；效率为人工 8.7 倍；询价制 |
| 广联达 AecGPT + AI 审图 | 2024-05 发布 320 亿参数行业大模型；审图覆盖 5 大专业 160+ 标准 2000+ 条文；AI 图像叠图比对 | 与产品线捆绑，政企项目制 |
| 品茗晓筑 | 「查造价」（定额/清单秒查）、「智能审图」（平面图↔节点详图关联）、AI 图模管理（图纸识别/分类/版本） | 未公开 |
| 云工涌现（上海建工孵化） | 云工·图 3.0 工程 AI-CAD 引擎；规范/图集问答、方案审核 | 集团孵化，未公开 |
| 小库科技 XKool | 设计前端（强排/日照/出图），不做施工图算量识别 | 免费试用+企业版 |

**判读**：国内图纸大模型 2024-2026 才起步，多为「知识问答+局部审图」，**开放图纸问答无高准确率商品**；万翼证明「识别+规则+大模型」混合管线的商业闭环（纯 VLM 路线无人敢上生产）。

### 3.3 国内痛点（= gaea 机会）

- **图纸质量敏感**：天正格式、外部参照、嵌套块导致识别失败是最高频痛点（B 站大量「分解到底」救图教程）；连广联达 IQcad 都要求先规范化预处理。
- **价格门槛**：广联达 9,000/年只租不卖，个人造价师/小事务所负担重（催生盗版锁灰色市场）→ 一次性买断/低价桌面授权有差异化空间。
- **机电/安装/装修专业**识别深度不足。

---

## 四、海外竞品格局

### 4.1 AI takeoff 新锐

| 公司 | AI 能力 | 定价（月费） | 融资信号 |
|---|---|---|---|
| Togal.AI | 自动检测/测量/对比（自报 98%）、AI Search 同类符号计数、Togal.CHAT 问图 | Growth **$299/人/月** | 2025-08 $7M A 轮，累计 ~$17.5-22.65M |
| Kreo | Auto Measure 3.0（轮廓 94%/面积 99.3% 自报）、房间/门窗检测、比例尺识别、Caddie AI agent + 无代码 Skill Builder、语音 | Pro **$175**、Business $250 | 公开融资仅 ~$3M，客户含 Willmott Dixon |
| Beam AI（ibeam.ai） | 全自动 takeoff+估算（自报 99%+）、修订自动检测、航拍算量；DIY 与 Done-For-You（人工 QA 24-72h 交付）两档 | DIY $8K/年起/工种；DFY $25K/年起 | **2025-11 $30.5M B 轮（Insight 领投，赛道最大单笔）** |
| Handoff.ai | 住宅「AI 队友」：图纸→估算+报价，按 ZIP code 本地化价格，10 万+估算训练 | Flex $119 / Pro $239 | 2025-07 $5.8M |
| xBuild | 「vibe estimating」：照片/测量→完整报价，屋顶估算 <15 分钟 | Token 计费 | **2026-01 $19M A 轮（a16z 参投）** |

**技术路线观察**：各家均支持 PDF（含扫描件）/CAD/图片混合输入，但公开材料极少披露底层——主流是**图像/CV 管线**（Togal「给一个符号找全部同类」是典型目标检测路线），矢量解析不是公开卖点。第三方指南（ConstructConnect 2026-08）明确提示**多数 AI takeoff 在 MEP/市政图上仍不可靠**。

### 4.2 传统厂商 AI 化

- **Autodesk**：AutoCAD 2026 Markup Assist（AI 识别 PDF 批注并建议应用）、Activity Insights；Autodesk Construction Cloud 更名 Forma Takeoff，含 AI Symbol Detection。打法=「AI 增强现有工作流」。
- **Bluebeam Max**（2025-10 发布、2026 年初上线）：**集成 Anthropic Claude「与图纸对话」**+ AI-REVIEW/AI-MATCH（收购 Firmus AI）——传统巨头亲自把「图纸对话」做成 2026 主打，是最强方向信号。
- PlanSwift Takeoff Boost（$2,000/年）、STACK FloorPlan AI（$249-299/用户/月 + $899 附加）、RIB CostX（询价）。

### 4.3 P&ID/工业图纸数字化

SymphonyAI、Azati 等做流程工厂 P&ID 数字化（目标检测+OCR+关系抽取三元管线；Roboflow 在封闭符号集上做到 99.2% mAP）。与建筑造价是两个付费场景，对 gaea 是**技术可借鉴**（切块→检测→OCR→结构化的管线结构）而非直接竞品。

### 4.4 市场信号

- 无权威机构单独测算 AI takeoff 细分市场；「AI in Construction」大口径 $3.9B~$11.1B（2025），共识 CAGR 20-25%，estimating/takeoff 被普遍列为增长最快子赛道。
- 融资时间线：2024-02 Beam A 轮 → 2025-07 Handoff → 2025-08 Togal A 轮 → 2025-10 Bluebeam Max 发布 → 2025-11 Beam B 轮 $30.5M → 2026-01 xBuild $19M。**2025 下半年起赛道升温且有顶级 VC+传统巨头双重背书**。

---

## 五、技术路线盘点（面向 Wails：Go + React）

### 5.1 DWG/DXF 解析与转换

| 组件 | 语言 | 许可证 | 覆盖 | 评估 |
|---|---|---|---|---|
| **unitedrhino/go-cad** | Go | **Apache-2.0** | DWG 读 R9~R2018、写 R2000~R2018；DXF ASCII/二进制 R12~R2018 双向；DXF↔JSON；内置 PNG 渲染（含中文字形）、SVG 输出、智能分图、标题栏图号识别 | 纯 Go 零 cgo，恰好覆盖 Wails 痛点；自称与 LibreDWG 37 张真实图纸 100% 值级对齐。**但仅 59 stars/18 commits，社区验证不足，必须 PoC**；无 ACIS 内核（SAT/SAB 只存 blob） |
| LibreDWG | C | **GPL-3.0** | DWG 读全版本（R2010+ 少数高级对象跳过）；写仅 R2000 前可靠 | `dwgread`/`dwg2dxf`/`dwg2SVG` 工具链成熟、活跃维护；**只能子进程调用（arm's length），严禁 cgo 链接**；FreeCAD 曾因 GPLv3 传染移除先例 |
| ODA File Converter | 专有 | 免费但**不可随产品再分发** | DWG/DXF R12~R2018 互转 | 集成方式只能是「检测未装→引导用户官网自装」；ODA SDK 商用首年 $7,500 起 |
| DWG TrueView | 专有 | 免费 | 只能 DWG 版本间转换，**不能导出 DXF** | 排除 |
| ezdxf | Python | MIT | DXF 读写 R12~R2018，drawing add-on 出 PNG/SVG | 活跃（1.4.5，2026-10），适合做离线测试基准 |

### 5.2 前端渲染与 JS/Go 生态

- **mlightcad/cad-viewer**（MIT，1195★，日活维护，**2026-10-23 二轮调研头号发现**）：纯浏览器 DXF/DWG 查看+编辑器（three.js/WebGL），自带图层管理/测量（距离/面积/角度）/对象捕捉/编辑命令；核心 `@mlightcad/cad-simple-viewer` 是框架无关 plain-DOM 组件，可直接嵌 React（不引入其 Vue 全套 UI）；另有 cad-agent-plugin（查看器内 AI 助手）。注意其 DWG 直渲走 LibreDWG WASM（GPL 传染）或商业解析器——我们用 Go 侧先转 DXF 即绕开。首选。
- vagran/dxf-viewer（MPL-2.0）：活跃、高性能 WebGL、自带解析器、有 React 嵌入示例；MPL 文件级 copyleft 商用安全。备选（仅渲染，无测量/图层树等完整查看器能力）。
- dxf-parser + three-dxf（MIT）：事实标准但 2021/2024 起停更，需 fork 维护。
- Go 侧其余 DXF 库（yofu/dxf、flywave/go-dxf、rpaloschi/dxf-go）陈旧，不可用——**go-cad 之外 Go 没有能打的库**。

### 5.3 矢量化与符号识别（研究面）

- 扫描图→CAD 矢量化：Scan2CAD $89/月、WinTopo 免费版可商用；Print2CAD 公司 2025-09 已解散。深度学习矢量化集中在户型图（Raster2Seq 等），机械/电气图纸**无成熟开源方案**——「扫描件重建矢量」不作为目标。
- 符号识别研究资源：FloorplanCAD 数据集（1 万+ 真实图纸，panoptic symbol spotting 基准）、ArchCAD-400k、CADSpotting/LS-CAD（2024-12）；PaddleOCR PP-StructureV3 表格识别带单元格坐标（对标题栏/明细表定位有用），但对密集线条图纸无官方适配，社区实践=先子图分割再理解。

### 5.4 三条可行路线与量级

- **A 矢量管线**：go-cad（兜底 LibreDWG 子进程）解析 → 实体/图层/文本 JSON → 前端 vagran/dxf-viewer 渲染。几何/图层/文本精确可得，支持拾取测量，无网络依赖。MVP 约 2~4 人月。
- **B 光栅化+VLM 语义层**：矢量/PDF/扫描件统一光栅化 → 复用 vision 管线做问答/表格提取。对扫描件一视同仁，但无矢量几何、计数尺寸不可信。MVP 约 3~6 人月。
- **C 混合（推荐）**：A 为骨架、B 为语义。一期=A（导入/渲染/图层树/文本提取）+光栅化出图接现有识图卡问答；二期=表格结构化提取→造价域联动；三期视需求做符号 spotting（本地引擎部署）。

---

## 六、VLM 图纸理解能力证据（决定产品边界的关键数据）

### 6.1 横向实测

- **AECV-bench**（2026-01，10 模型 120 张平面图）：计数均值 Gemini 3 Pro 51% > GPT-5.2 49% > Claude Opus 4.5 42% > GLM-4.6V 39%（开源最佳）；分项卧室 91%（靠读文字标签）/**门 39%/窗 34%**；能力梯度 OCR≈0.95 > 比较推理 0.80 > 空间推理 0.70 > 计数 0.40-0.55。结论原文：AI 是合格「文档助手」但缺乏「图纸素养」。
- **Businessware 机械图基准**（2025-08）：表格单元格 Gemini 2.5 Pro 94.2% 第一；尺寸提取 Gemini 2.5 Pro ~80%、Claude ~40%、GPT-o3 ~20%、Qwen VL Plus <8%；有编造行列/虚构公差实证。
- **其他**：Claude Opus 4.7 高分辨率档读图综合最佳但「没有任何模型能做工程精度测量，用于决策的数字必须人工复核」；GPT-5 计数 10 题对 4、直尺测量失败；三层平面图实测门宽 1.80m 读成 1.00m。
- **学术**：DesignQA（Autodesk）GPT-4 系几何/公差提取仅 40-62%；GD&T 微调小模型 F1 97.3%、幻觉降到 5.23%（微调专用小模型是生产级提取主流）；工程示意图基准 Enginuity Token F1 仅 0.03-0.18（数值逐字正确是硬伤）。通用基准高分≠工程图纸可靠（领域鸿沟）。

### 6.2 分界线（产品设计的铁律）

| 能力 | 证据准确率 | 可否投产 |
|---|---|---|
| 图纸文字/说明/标题栏 OCR | ~0.95 | ✅ 可（含本地 Qwen3.6-35B 级别） |
| 表格提取（门窗表/钢筋表/做法表） | 0.85-0.94 | ✅ 可，需结构化 JSON+对账校验 |
| 语义问答/概念检查/归档整理 | 0.80+ | ✅ 可 |
| 构件计数（门/窗/符号） | 0.34-0.55 | ❌ 不可直接信；要走矢量实体计数或专用检测模型 |
| 尺寸/数值读取 | 0.20-0.80 且有幻觉 | ❌ 不可直接信；必须展示依据+人工复核 |

### 6.3 计价（2026-10 实时）

单张高分辨率图折合约 600~5,000 token：OpenAI ~$0.0011、Claude ~$0.0065、Gemini ~$0.001-0.002、**qwen3-vl-plus ~$0.0002、GLM-4.6V ¥1/M（有免费 Flash 档）**。批量图纸问答边际成本可忽略，瓶颈是准确率不是钱。本地 herdsman 多模态走现有管线零边际成本。

---

## 七、对 gaea 的建议

### 7.1 定位（一句话）

**「桌面本地的图纸信息提取与问答卡」**——导入 DWG/DXF（及 PDF/截图），看得见（WebGL 渲染+图层树）、问得出（复用识图管线 VLM 问答）、提得走（表格/文字结构化导出，二期接清单组价）。**明确不做**看图算量/计数/尺寸读取直出工程量。

### 7.2 红线（调研证据支撑）

1. 不做全构件算量建模（GTJ 护城河=规则库+校核+用户锁定，识别只是入口）。
2. 计数/尺寸类数值不直接信 VLM 输出；矢量实体计数走 DXF 实体层（块参照计数天然精确——这是矢量管线的独有优势）。
3. LibreDWG 仅子进程调用（GPL 严禁链接）；ODA File Converter 只引导自装不分发。
4. CAD 领域知识（图例、平法、规范）按既有红线技能化，内核只留通用机制。
5. 提取结果须「展示依据」（高亮来源区域/原始文本），对标广联达/万翼的可追溯校核范式。

### 7.3 分期

- **P0（可选卡上线，约 2~4 人月）**：DXF 原生解析 + DWG 经 go-cad（PoC 通过后）/LibreDWG 子进程转换 → mlightcad/cad-simple-viewer（MIT，自带图层树/测量/捕捉）或 vagran/dxf-viewer 渲染 + 文本/标注提取（go-cad 自带标题栏图号识别与 caddocling 结构化管线，见第十章）→ 光栅化切块接现有 vision 识图问答。绑定面遵循 VisionRoute 只读不记账红线。
- **P1（造价域联动，+2~3 人月）**：门窗表/钢筋表/做法表结构化（PP-StructureV3 管线或 VLM+坐标对账）→ 导出/联动清单组价（复用 AI 组价与 MatchResourceByTitleSpec 宁缺勿误管线）。
- **P2（观察池，不立项）**：符号 spotting 本地模型（FloorplanCAD 路线）、扫描件理解（B 路线已覆盖）、批量归档整理。
- **前置 PoC（1~2 周）**：go-cad 用真实用户图纸（含天正导出件）验证解析完备度，对照 LibreDWG 输出；不过则 P0 降级为「DXF/PDF 优先 + DWG 引导用户转格式」。

### 7.4 风险清单

- go-cad 社区验证不足（59 stars，单一企业主导）——PoC 把关 + LibreDWG 兜底双保险。
- 天正代理实体/外部参照是行业性坑（广联达都要预处理）——首版明示支持边界。
- 本地 Qwen3.6-35B 对图纸理解的准确率未实测——P0 前用 AECV-bench（开源）+ 自建金标图集跑一轮本地/云端对比，确定默认路由。
- Windows 分发 LibreDWG 二进制需附 GPL 合规（源码链接），法务过目。

---

## 八、查不到/存疑项（诚实清单）

- 广联达 GCCP/数字造价云「AI 组价」官方一手功能细节与价格：未找到官方 URL。
- 万翼/品茗/鲁班/小库公开价目表：均不存在，全部询价制。
- 鲁班「CAD 转化 95%」：出自第三方下载站，官网未见此数字。
- Togal 2025 A 轮无公开新闻稿（仅 PitchBook 记录）；TrueBuilt 定价未公开。
- AI takeoff 独立细分市场规模：无权威数据（只有 AI in Construction 大口径）。
- ODA File Converter EULA 再分发逐字条款：需下载后审阅。
- 机电审图横评结论（万翼电气弱）：正文 403，仅取搜索摘要，未核实全文。

---

## 九、来源索引（节选）

**国内**：广联达 GTJ2025 CAD 识别文档 aecore.glodon.com/doc/GTJ2025/0fa9a428570440b39ab1a8067d384607｜GTJ2026 产品页 glodon.com/product/145.html｜IQcad aecore.glodon.com/doc/iqcad｜QuantifAI 人民日报 2026-09-18｜年租 9,000 口径 m.fwxgx.com/questions/3972578｜万翼 vanyitech.com（新华网转载+深圳住建局应用）｜品茗 pinming.cn/product_cont_6078.html｜云工涌现 yungongai.com｜AecGPT glodon.com/news/1386.html

**海外**：Togal.AI togal.ai｜Kreo kreo.ai｜Beam AI ibeam.ai + Insight Partners 2025-11-12 新闻稿｜Handoff handoff.ai/blog（2025-07-02）｜xBuild PR Newswire（2026-01-20）｜Bluebeam Max press.bluebeam.com（2025-10-01）｜ConstructConnect AI takeoff 指南（2026-08）｜Autodesk AutoCAD 2026 博客（2025-03-25）｜STACK newswire.com（2024-01）

**技术**：go-cad github.com/unitedrhino/go-cad｜LibreDWG github.com/LibreDWG/libredwg + GNU GPL FAQ｜ODA opendesign.com/pricing｜ezdxf ezdxf.readthedocs.io｜vagran/dxf-viewer github.com/vagran/dxf-viewer｜FloorplanCAD floorplancad.github.io｜PP-StructureV3 arxiv.org/html/2507.05595v1

**VLM**：AECV-bench aecfoundry.com/blog（2026-01-10，2026-10-20 更新）+ github.com/AECFoundry/AECV-Bench｜Businessware businesswaretech.com（2025-08）｜AI Consulting Network（2026-05-13）｜Roboflow GPT-5 vision（2025-08-07）｜ArchitectsWhoCode（2026-04-21）｜DesignQA arxiv.org/abs/2404.07917｜GD&T 微调 arxiv.org/abs/2411.03707｜Enginuity arxiv.org/abs/2606.03410｜各厂计价页（OpenAI/Anthropic/Google/阿里云/智谱/DeepSeek 2026-10 抓取）

---

# 附录（2026-10-23 二轮）：GitHub 与社区同类开源项目盘点

> 专答「GitHub/社区有没有同类产品」。star/许可证/最近推送均经 GitHub API 或仓库页核实；分级：**A=可直接复用（活跃+许可友好+工程级）、B=只可借鉴设计（传染许可/栈不匹配）、C=玩具/弃坑/营销号**。

## A 级：可直接复用

| 项目 | Stars/许可/状态 | 做什么 | 对 gaea 的价值 |
|---|---|---|---|
| [mlightcad/cad-viewer](https://github.com/mlightcad/cad-viewer) | 1195★/MIT/2026-10-22 日活 | 浏览器 DXF/DWG 查看+编辑（three.js/WebGL，图层/测量/捕捉/编辑命令） | **前端查看模块首选**：核心 @mlightcad/cad-simple-viewer 框架无关可嵌 React；cad-agent-plugin=查看器内 AI 助手可对标 |
| [unitedrhino/go-cad](https://github.com/unitedrhino/go-cad) | 59★/Apache-2.0/2026-10 | 纯 Go DWG/DXF 解析渲染（详见 5.1） | 二轮补充：公司主体（联犀/ThingsPanel）非个人玩票；**自带 caddocling CLI 输出图框/文字/图像结构化清单并与 docling 集成做 RAG——直接命中图纸×AI**；SVG 输出带图层组/handle 元数据 |
| [jeremylongshore/cad-ai-agent](https://github.com/jeremylongshore/cad-ai-agent) | 45★/Apache-2.0/**当天仍在提交**，345 commits | DXF/矢量 PDF 分析+安全编辑；LLM 只产出结构化 ChangeSet 不直接改图，模型无关，20+ 工具 agent 循环，mock 模式离线可跑 | **「LLM×图纸理解」架构最佳蓝本**（Apache 友好）：规划→结构化变更→确定性执行，与我们「AI 不直接信、结构化+校核」红线同构 |
| CAD MCP 三件（借设计不借协议——gaea 拍板不做 MCP） | | | |
| ├ [daobataotie/CAD-MCP](https://github.com/daobataotie/CAD-MCP) | 632★/MIT/2026-10 活跃 | 35 工具「绘制→检查→纠正」闭环；**支持 AutoCAD/浩辰/中望** | 与国内造价用户 CAD 环境完全对口 |
| ├ [puran-water/autocad-mcp](https://github.com/puran-water/autocad-mcp) | 586★/MIT | AutoCAD LT 自动化；**ezdxf 无头 DXF 双后端（免装 CAD）** | 无头路线与 ezdxf 技术栈衔接 |
| └ [U-C4N/Autocad-MCP](https://github.com/U-C4N/Autocad-MCP) | 173★/MIT/2026-09 | 247 工具：drawing_understand 一键读图、房间面积、**门窗明细表、管线/电缆算量** | 算量类工具集的划分设计直接抄 |
| [FloorPlanCAD 数据集](https://floorplancad.github.io) + [floorcad-yolov8n-detect 现成模型](https://huggingface.co/mudasir13cs/floorcad-yolov8n-detect) | 1 万+ 真实 CAD 图纸细粒度标注 | 矢量 CAD 符号 spotting 基准 | P2 符号识别若立项，训练底座免费现成 |

## B 级：只可借鉴设计（传染许可或仅参考）

| 项目 | Stars/许可 | 借鉴点 |
|---|---|---|
| [Kentucky-ai/opentakeoff](https://github.com/Kentucky-ai/opentakeoff) | 652★/**AGPL-3.0**/2026-10-22 非常活跃，721 commits | 浏览器 PDF takeoff（React18 同栈）：测量交互、工程量**溯源**、AI 结果「铅笔态」须人工盖章才生效——与我们「展示依据+人工校核」设计完全同构；AGPL 传染**不能抄代码**，交互设计可看 |
| [OpenConstructionERP](https://github.com/datadrivenconstruction/OpenConstructionERP) | 1020★/AGPL/极活跃 8487 commits | 自托管建筑 ERP：LLM（20 家接入）→带置信度 BOQ→CWICR 12 万条价格库匹配；CV 管线 PaddleOCR+YOLOv11——AI 估价管线与数据模型参考；同厂 [CWICR 开放成本库](https://github.com/datadrivenconstruction/OpenConstructionEstimate-DDC-CWICR)（55K+ 工作项，许可未声明用前确认） |
| [Azure P&ID digitization](https://github.com/Azure-Samples/digitization-of-piping-and-instrument-diagrams) | 143★/MIT/低频 | 符号检测+OCR+关系抽取→图结构→图数据库的**管线设计文档**是最好的教材（强绑 Azure 不搬码） |
| [VITA-Group/CADTransformer](https://github.com/VITA-Group/CADTransformer) | 137★/MIT/归档 | 矢量 CAD 图上的符号 spotting（CVPR'22） |
| [grebtsew/FloorplanToBlender3d](https://github.com/grebtsew/FloorplanToBlender3d) | 632★/GPL-3.0/停滞 | 检测→3D 的工程化管线组织（Swagger/Docker/教程） |
| [Edwardhehe/batchPrintZWCAD](https://github.com/Edwardhehe/batchPrintZWCAD) | 107★/MIT/2026-10 活跃 | 图框块识别+批量打印（中望平台）——图纸整理周边唯一活跃开源实现 |
| [skymakerolof/dxf](https://github.com/skymakerolof/dxf)（原 bjnortier） | 402★/MIT/2025-09 | DXF→SVG 轻量路线（不依赖 three.js） |
| Go 侧配套 | — | 渲染 draw2d（BSD-2）/几何 go-geom（BSD-2）；**Go 无主流多边形布尔库**（测量类自算够用） |

## C 级：玩具/弃坑/营销号（明确避开）

- [FloorplanTransformation](https://github.com/art-programmer/FloorplanTransformation)（682★/MIT）与 [DeepFloorplan](https://github.com/zlzeng/DeepFloorplan)（418★/GPL）：学术名作但 Python2/TF1/Torch7 停更，仅历史参考。
- [CubiCasa5k](https://github.com/CubiCasa/CubiCasa5k)：数据集 **CC BY-NC 非商业**，只能当研究基线。
- **VisionMillionDataStudio / Vision-Studios 系「CAD 门窗识别数据集」仓库**：营销号矩阵引流付费毕设，数据来路不明，禁用。
- CSDN「AI小怪兽」YOLO11 CAD 检测系统：源码在付费墙内，mAP50 仅 0.543。
- Bakkopi/engineering-drawing-extractor（无许可证 4 commits 弃坑）、BlueprintParser_OS（无许可证半玩具）、ypj658/CadPlot（2013 弃坑）。
- **kkFileView 的 DWG 预览**：底层 aspose-cad，**商用必须另购授权**——别以为套 kkFileView 就白嫖了 DWG。
- 桌面 CAD 三家（LibreCAD 6.5k★/GPLv2、QCAD CE/GPLv3、FreeCAD 34.7k★/LGPL）：全部「DXF 原生+DWG 靠外部转换」，GPL 传染不可嵌入闭源模块，只能读源码学处理逻辑。

## 空白区（确认无活跃开源项目）

- **标题栏/图框识别**：论文有（arXiv 2504.08645）代码无；业界现状=VLM+表格抽取自研（go-cad 自带该能力是稀缺点）。
- **产品级 symbol spotting**：只有数据集+论文代码。
- **Gitee 全域**：site: 定向检索确认，**没有独立维护的中文工程图纸识别开源项目**——Gitee 结果全是通用 CV 仓库/kkFileView 系 fork/无关项目。
- **宽松许可的浏览器 DWG 直渲**：不存在（mlightcad 的 DWG 也走 LibreDWG WASM=GPL 或商业包），唯一安全路径=后端先转 DXF。

## 中文社区信号（知乎/V2EX/B站）

- 知乎实测《AI 识图算量：4 款工具对比》（2026-05，物流园真实项目）：广联达 AI 算量最准但贵；**DeepSeek 识图是唯一免费方案但结果不可编辑，「生产不可用」**；元启 BeesFPD 机电准；品茗 D+ 建筑结构强。行业共识=「半自动+AI」。
- **V2EX 从业者共识（2023~2025 三帖）：「别走视觉路线，直接读原始 CAD 矢量数据」信息量更大更精确；「AI 无法保证准确性在工程中是致命的」**——与我们混合路线（矢量骨架+VLM 语义层）独立得出相同结论，方向印证。
- **天正事实标准**：天正构件=ACAD_PROXY_ENTITY 代理实体，无任何开源解析方案（LibreDWG 也不行）；业界做法=引导用户在天正里 TXDC 批量导出 T3 后再进软件（免费的快速看图/飞图/浩辰云转换均为商业免费非开源）。另有 2★ 萌芽项目 [hwdemtv/tianzheng-dwg-parse](https://github.com/hwdemtv/tianzheng-dwg-parse)（2026-09，天正施工图解析）可观察。
- MxDraw/vjmap/中望/浩辰 SDK 全是商业授权（「免费下载」只是试用；中望专业版 10,998 元/套），不作为依赖。

## 二轮结论修订

1. **前端查看器首选升级**：vagran/dxf-viewer → **mlightcad/cad-simple-viewer**（MIT、图层/测量/捕捉齐备、日活维护、框架无关可嵌 React；DWG 由 Go 侧先转 DXF 绕开其 GPL/商业解析器）。
2. **图纸×AI 的两个免费蓝本**：cad-ai-agent（Apache-2.0，结构化 ChangeSet 架构）+ go-cad caddocling（Go 原生 RAG 管线）；交互范式对标 opentakeoff 的「铅笔态+人工盖章」（AGPL 只看不抄）。
3. **中文开源侧无竞品无积木**：Gitee 空白，说明该方向国内没有开源先行者——既是机会（无人占位）也再次说明难点在工程而非算法。
4. V2EX「读矢量不走视觉」从业者共识与我们的混合路线互证，P0 立项的技术信心增强。
