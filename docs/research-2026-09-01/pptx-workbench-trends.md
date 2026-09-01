# PPT 工作台与 AI 侧栏动向调研（2026-09-01）

> 服务规划 B2「pptx 最小交互」（大纲 + 每页缩略图预览，点页 →「针对第 N 页修改」）。基线 v4.24.0 / dsh-better-sidebar v0.18。方法：在线检索核实（本环境 Google 域、DDG、Brave 均不可达，以官方页直抓 + cn.bing 为主）；未能核实处如实标注，未编造。

## 一、AI 生成/编辑 PPT 的交互先例（Q1）

- **Copilot in PowerPoint**：入口固定在右下角 Copilot 图标弹出的聊天面板；新建走「Add Content → Agent Mode」，先对话澄清受众/风格、细化**大纲**再生成，生成后"继续聊天编辑"（[support.microsoft.com](https://support.microsoft.com/en-us/powerpoint/copilot/create-a-new-presentation-with-copilot-in-powerpoint)）。编辑指令的页级定位两种句式**并存**：相对引用"this slide"（先选中页）与显式页码——官方示例句"Update slides 4–6 for a UK audience""Add … to slides 3, 5, and 7"；另有 Skills 菜单一键命令（Review this presentation / Visualize this slide）（[support.microsoft.com](https://support.microsoft.com/en-US/PowerPoint/edit-with-copilot-in-powerpoint)）。加页同样是聊天描述而非操作缩略图（[support.microsoft.com](https://support.microsoft.com/en-US/PowerPoint/add-a-slide-with-copilot)）。
- **Beautiful.ai**：标准 outline-first——生成设计前必须"Review and refine a slide-by-slide outline"（逐页大纲确认），迭代时"Rewrite/tighten/expand… without breaking slide structure"；未描述对单页发指令的聊天（[beautiful.ai](https://www.beautiful.ai/)）。
- **Gamma**：卡片制（一卡=一页，付费按"卡片"计量），大纲是输入源之一；行内"+++"触发 AI 续写、"AI 一键改写、扩写、翻译、生成图片"；缩略图导航与单卡对话修改**未获官方页证实**（官方站不可达，取自二手页：[gamma.gongke.net](https://gamma.gongke.net/)、[ai-bot.cn](https://ai-bot.cn/sites/995.html)，未核实项）。
- **Kimi**：Agent 侧栏一级入口"PPT/文档/表格"，宣称"一键生成文档/PPT/表格"（[kimi.com/zh/agent](https://www.kimi.com/zh/agent)）；逐页修改流程**未核实**。
- **WPS AI（灵犀）**：「PPT大纲生成」"根据给定的主题一键生成PPT大纲"+四大助手（写作/设计/阅读/数据）；无逐页对话修改描述；**"网页版仅开放基础功能"——完整 AI 能力在客户端**（[ai.wps.cn](https://ai.wps.cn)）。
- Gemini in Slides：Google 域不可达，**未核实**。Tome：官网 404，现状**未核实**。

**对 gaea v4.26+ 的启示**：B2"大纲+缩略图双视图、点页→指令"踩中行业主线；页级指令应同时支持**显式页码（官方句式"改第 3–5 页"）与"当前选中页"相对引用**两条通道，并把"生成前大纲确认"做成硬门槛。

## 二、python-pptx 侧预览技术路线（Q2）

- **路线 A（渲染像素）**：LibreOffice headless `--convert-to pdf:… --outdir`（官方命令行参数，[help.libreoffice.org](https://help.libreoffice.org/latest/en-US/text/shared/guide/start_parameters.html)）→ PyMuPDF 逐页光栅化（`for page in doc:` + `Page.get_pixmap()`，"create a page image in raster format"，[pymupdf.readthedocs.io](https://pymupdf.readthedocs.io/en/latest/page.html)）。社区已知坑：缺中文字体致版式漂移、soffice 冷启动秒级开销与单实例 profile 锁、Windows 下服务化难——**社区经验，本次未在线核实到具体帖子**；直接 `--convert-to png` 是否只出首页同样**未核实**。
- **路线 B（不渲染像素，更轻）**：python-pptx 原生读结构——Working with Slides / Understanding Shapes / Understanding Placeholders / Working with text 等章节覆盖逐页抽标题+正文+形状树（[python-pptx.readthedocs.io](https://python-pptx.readthedocs.io/en/stable/)），零外部依赖、毫秒级、无字体风险，代价是无视觉保真。
- OSS 先例：orcastor/addon-previewer（Go，本地多模态预览 docx/xlsx/pptx，60★）（[api.github.com](https://api.github.com/search/repositories?q=pptx+thumbnail+preview)）。

**对 gaea v4.26+ 的启示**：v4.26 先做路线 B 的"结构化大纲卡"（标题+要点+形状/图表统计+备注），像素缩略图做成可选增强：soffice→PDF 一次转换，PyMuPDF 按页懒渲染 + 磁盘缓存，避免逐页重启 soffice。

## 三、「AI 工作台/侧边栏」2026 新动向（Q3）

- **Claude Cowork（实锤）**：桌面 App（macOS/Windows/ChromeOS/Linux），定位"hand Claude real work"——文件夹级读写授权、内置浏览器进**侧栏**（side panel）、直接处理 pptx/xlsx/docx、定时任务、插件=Skills+Connectors+Sub-agents、"shows each step" 步骤可视；2026-02→08 滚动更新（[claude.com/product/cowork](https://claude.com/product/cowork)）。
- **ChatGPT 桌面端**：Microsoft Store 官方条目宣称"Bring in context from your apps, files, browser, screenshots, email, and screen"（[apps.microsoft.com](https://apps.microsoft.com/detail/9plm9xgg6vks)）；用户提到的"desktop workapps"**未查到**。
- **Manus**：官网确认 slides 创建、browser operator、桌面 App（[manus.im](https://manus.im)）；"侧栏改版"**未查到实据**；2026 年公司层面动荡（收购被否→8 月恢复独立运营，[zhihu.com](https://www.zhihu.com/question/2070808774667921064)，二手）。

**对 gaea v4.26+ 的启示**：侧栏正从"聊天侧栏"进化为"代理工作台"（授权范围可见+步骤可视+定时+并行）；dsh-better-sidebar 的对标升级点是 Cowork 式**文件授权范围声明 + 执行步骤逐条展示**，而非更多聊天入口。

## 四、办公秘书/助理品类动静（Q4）

- **M365 Copilot**：2026-04-22 Word/Excel/PPT **agentic 能力 GA**（"take multi-step, app-native actions directly in your documents"、尊重企业模板、理解动画；自列 next = "more transparency/preview of changes"）（[microsoft.com](https://www.microsoft.com/en-us/microsoft-365/blog/2026/04/22/copilots-agentic-capabilities-in-word-excel-and-powerpoint-are-generally-available/)）。**2026-05-28 全新设计**（"5 月新设计"的后续即此线）：左导航可展开收起、提示行升级为"task-aware workspace"、capability-focused agents（Designer/Researcher/Word/Excel/PowerPoint）、应用内 side pane 定位为"会建议也会直接改"的 editing partner、可在"within a paragraph, cell, or slide"就地唤起、"progressive disclosure"；上线后 PowerPoint 使用 +43%（[microsoft.com](https://www.microsoft.com/en-us/microsoft-365/blog/2026/05/28/introducing-a-new-design-for-microsoft-365-copilot/)）。后续：06-02 Work IQ APIs + 常驻个人代理 Microsoft Scout、06-16 微软自家 **Copilot Cowork GA**（与 Anthropic 撞名，"Cowork"成品类词）（[microsoft.com](https://www.microsoft.com/en-us/microsoft-365/blog/)）。
- **Google Workspace 侧栏**：本环境不可达，**未核实**。
- **国内**：飞书官网标题已改定位为"字节跳动旗下AI工作平台"（[feishu.cn](https://www.feishu.cn)，标题级证据）；钉钉 AI 助理 2026 具体动向**未核实**（搜索仅返回下载页）；WPS AI 见第一节——桌面/客户端是 AI 完整能力载体。

**对 gaea v4.26+ 的启示**：头部共识 = PPT 编辑放在应用内 side pane + 就地唤起 + 代理直接改，gaea 右栏工作台与之同构；微软自列的"变更透明/预览"恰是 gaea Verify→Journal→回滚的先发优势，应趁 B2 把「改了哪几页、可一键还原」做成 PPT 工作台的默认 UI。

## 五、未核实清单（诚实声明）

Gamma 官方站/帮助中心、Tome、Gemini in Slides、Google Workspace 侧栏、Kimi PPT 逐页流程、钉钉 AI 助理细节、Manus 侧栏改版、ChatGPT"workapps"、LibreOffice 单页导出坑的原始帖——均未核实；文中相关结论已降级为"未核实"或改用二手来源标注。
