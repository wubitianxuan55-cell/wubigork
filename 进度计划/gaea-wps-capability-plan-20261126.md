# WPS 能力开发施工单（大项 · 已记录待开工）

2026-11-26 · 依据用户当日拍板（原话要点：「我的 WPS 是年会员，额度没有问题，转换解析优先用 WPS，
另外 WPS 的许多能力我们要再开发，用来帮助我们 gaea」「WPS 能力是大项，先做好记录，下个会话进行」）
状态：**只记录，未开工**。下个会话从这里起步，不要重新摸底。

## 一、事实源（开工先读这两份）

| 文件 | 内容 |
|---|---|
| `C:\AI\re\wps-12.1.0.29690\docs\WPSCLI-CAPABILITIES.md` | **主事实源**：入口与门禁勘误、30 个子命令分组、扫描件在线 OCR 实测数字、WPS 输出的已知瑕疵、每项能力开工前的实测清单、复跑脚本 |
| `C:\AI\re\wps-12.1.0.29690\evidence\kpdfcli_help.json` | CLI 帮助全文快照（kpdfcli 3.1.0.3809，30 个子命令的参数与退出码）；摘要版 `work\wpscli-summary.txt` |

数据与证据一律放 `C:\AI\re\wps-12.1.0.29690\`，仓库只放文档与代码（仓根 `AGENTS.md` 强制默认）。

## 二、已经落定的前置（本批随包）

- 用户拍板：年会员、额度不设限；**转换解析优先用 WPS**。
- 已落地：提不出文字的 PDF 先走 WPS 在线 OCR（`pdf2word --scanned true --range`，100 页分段）；
  `format_convert` 对文字型 PDF 也先经 WPS；WPS 不可用才退回本地提取 / 本地 OCR（限 30 页）。
- 实测：174 页三明治扫描件整条链 85 秒（之前本地 OCR 堵死 40 分钟）。
- 代码入口：`internal/docmd/wps.go`（`findWpsCLI` / `wpsRun` / `wpsConvert` / `wpsScannedPDFMarkdown`）、
  `internal/docmd/pdf.go`（分流）、`internal/gaea/tool/builtin/format_convert.go`。

## 三、能力清单（三梯队，对应 gaea 现状）

### 第一梯队 —— 造价 / 办公直接受益

| 能力 | gaea 现状 | 开工前要实测什么 |
|---|---|---|
| `pdf2excel`（支持 `--scanned` `--multi-sheet`） | 成本导入靠模型从 Markdown 文本里抠表 | 清单 / 定额表 PDF（文字型 + 扫描件各一）：sheet 切分、合并单元格、数字是否仍被拆成 `3. 1`、耗时 |
| `photo2excel` / `photo2word` | 图片只能出纯文本（`@图片` 的 OCR 腿） | 手机拍的报价单、手写单据：表格还原率、数字准确率、`--merge` 行为 |
| `ofd2pdf`（本地引擎）/ `caj2pdf` | gaea 不认 OFD / CAJ | 找真实 OFD（电子发票、证照、电子招标文件）与 CAJ 样本：是否本地完成、版式、是否要会员 |
| `pdf2txt` | 自写提取器 `internal/docmd/pdf.go`（乱码、页数、不认 `<hex>Tj` 等问题的根源） | 233 页文字型 PDF 测速；对比页码 / 表格 / 乱码 → 决定是否替换自写提取 |

### 第二梯队 —— 替换 LibreOffice

| 能力 | gaea 现状 | 开工前要实测什么 |
|---|---|---|
| `word2pdf` / `excel2pdf` / `ppt2pdf` | `internal/docmd/soffice.go` `ConvertToPdfFile`（LibreOffice） | 与 LO 比速度；GB/T 9704 公文、带公式 / 图表文档的中文字体保真 |
| `word2photo` / `excel2photo` / `ppt2photo` / `pdf2photo` | `doc_render` = LibreOffice 转 PDF + poppler 转 PNG | 同上；另看 `--long-image`、`--dpi` 是否够视觉验收用 |
| `pdf2cad`（产出 zip，须带 `--range`） | CAD 板块只读识别（`cad_ask`） | PDF 图纸 → 解 zip 看 DXF / DWG 版本、图层、文字是否可编辑 |

### 第三梯队 —— 办公杂务（适合并成一个工具的若干动作）

`pdfmerge` `pdfsplit` `pdfcompress` `pdfencrypt` `pdfwatermark` `pdfremovewatermark`
`photo2pdf` `txt2pdf` `pdf2ofd` `pdf2imgpdf` `pdf2ppt` `photo2ppt` `cad2pdf`。
（`pdf2md` 不可用：报 success 但产出 0 字节，11-24 四次实测。）

## 四、建议开工顺序

1. `pdf2excel` + `photo2excel`（造价最痛：清单、报价单进 Excel）。
2. `ofd2pdf` / `caj2pdf`（补 gaea 完全不认的两种格式）。
3. `pdf2txt` 评估（成立则替换自写提取器，能删掉一大块易错代码）。
4. 导出 / 渲染改走 WPS（先对测，LO 链保留为无 WPS 机器的兜底）。
5. `pdf2cad` 与杂务类，按需。

每项**先各做一次实测再立项**；实测清单见事实源 §4。

## 五、约束与坑（开工必读）

- **入口**：一律用 `<安装根>\clitool\wpscli.exe`（无门禁）。`office6\wpscli.exe` 有父进程签名门禁，
  11-24 报告只测了后者——不要再去研究绕门禁。**别跑 `wpscli install`**（会覆盖各 agent 目录的技能文件）。
- **账号级失败**：退出码 100（未登录）/ 101（无会员）/ 429（配额）→ gaea 进 10 分钟全局静默，不逐份重撞。
- **`--scanned true` 必须带 `--range`**（单输入只收一段）；gaea 按 100 页分段串行。
- **wpscli 一次只跑一个**：`wpsCLIMu` 串行锁，新能力复用同一把锁。
- **「success」不可信**：一律核对产物（文件存在且非空、内容有文字）。
- **单测不得真调 WPS**：`internal/docmd` 的 `TestMain` 默认 `GAEA_WPS_PDF=0`；要验 WPS 行为用
  `stubWpsBoth` 打桩，或让测试二进制子进程扮假 wpscli（`GAEAPROBE_WPS_STUB=1`）；真调实弹用
  `TestLiveWPS`（耗会员额度，默认跳过）。
- **WPS 输出瑕疵**（事实源 §3）：表格里小数被拆（已加 `rejoinSplitDecimals` 保守并回）、正文汉字间有空格、
  无标题样式、跨页断表、表头可能在首个数据行。新能力接进造价取数前，数字必须归一化并抽检。
- **没装 WPS / 非会员的机器**：每项新能力都要有本地兜底或如实报错，不能静默失败。
- 项目红线：领域规范进技能不进内核；简化界面不等于删功能；新增工具按「三处同批」纪律
  （注册 / `spacetags` / `compact.go` 紧凑描述，守卫测试会拦）；行为规矩走记忆不走代码门。

## 六、验收口径

- 每项能力：实测数字入事实源 §2；单测（桩）+ 一次真实样本实弹；失败路径有真因透传的报错。
- 涉及造价取数的：数字逐格抽检（对照原件），不只看「转出来了」。
- 涉及替换现有链路（LO、自写提取）的：新旧并跑同一批样本，记录速度与保真差异后再切。

## 七、未决点（开工时向用户确认）

1. 第一梯队四项是否按建议顺序，还是先做某一项。
2. 样本：OFD / CAJ / PDF 图纸 / 拍照报价单由用户提供，还是从现有工作区（`D:\新工作\…`）里找。
3. `--ai-fix true` 是否启用（在线 AI 修复扫描件；未实测，怕改动数字）。
