# 当日总账（2026-11-09）：dsh-wubi 优化迭代两刀（工具箱补满 15/15 + 子会话独立预设）

> 用户口径「优化迭代 dsh-wubi」，拍板选两线：功能欠账两件 + 工具箱补满 15/15。
> dsh-wubi 仓（C:\AI\wubigrok\dsh-wubi，junction 装机至
> C:\AI\dsh\home\profiles\web\node_modules\dsh-wubi）两笔：**6c5f7ea**（worker 预设+
> 接线+identity+Scripts 解析）、**5ae0ed8**（v2.1.0 版本行）。均未 push（仓内 ahead 2）。
> 隔离宿主 3091 由本会话冷启复验（PID 1400）。

## 刀一：工具箱补满 15/15（config.json 四空缺键全落）

| 工具 | 版本 | 装法 | 冒烟 |
|---|---|---|---|
| nmap | 7.993 | 官网 setup.exe（NSIS）→ 7-Zip 解包便携化 toolkit\nmap\ | --version 过（Npcap 缺走 connect() 模式，如实） |
| masscan | 1.3.2 | 社区构建 Arryboom/MasscanForWindows masscan64.exe（MZ 头核）→ toolkit\masscan\masscan.exe | --version 过（2022-01-11 官方 1.3.2 编译） |
| ksubdomain | v0.7 | release 仅 rar → winget 装 7-Zip 26.06 → 解包 toolkit\ksubdomain\ | --help 过 |
| impacket | v0.13.1 | Python 3.13 venv pip 装 toolkit\impacket\venv（.py 直落 Scripts/ 无 exe 包装） | secretsdump.py --help 过 |

- 代码面一刀：`findInDir` 嵌套目录补 **Scripts/**（venv 脚本层此前探不到 impacket）。
- config.json 四键写真实路径后 `resolveAllTools` **15/15**；子会话 E2E preflight 同步实
  证 15/15。
- **Npcap 未装**（nmap 退化 connect 扫、masscan 原始发包不可用）——系统级驱动装
  置挂用户拍板，未擅自装。
- Defender 白名单 toolkit+dl 两目录沿用 11-08 既有范围，未新增。

## 刀二：子会话 agentPreset 完整迁移（挂后续项落地）

- 机制实证：rc.1 spawn provider `capabilities.agentPreset=true` 且
  `inheritsParentContext=false`（fork 不支持，保持继承）；`dsh-subagent` index.js:349
  缺省继承父预设=证实此前子会话挂全套指挥预设。
- 新增包内预设 `presets/redteam-worker/`（14 行）：叶子纪律人设+anchor-gate+
  redteam-tools+pwsh/fs/jobs/skills/compaction/todo/web；**无指挥人设、无委派工具行、
  无 plan/goal/ask-user、无 REA MCP**（229 schema ≈60k tokens 不再进每个子会话；
  exploit/内网子会话若高频用 REA，后续可拆 per-role 预设，挂拍板池）。
- 插件 apply() 追加 worker 注册 effect：成功→部署副本 tool-subagent 行写入
  `agentPreset/allowedAgentPresets` 接线（文本手术带标记幂等，锚=maxDepth: 1 行）；
  失败/broken→主动摘除接线回退继承父预设，两侧不留断链。接线只写部署副本（注册
  真源），包内模板不带——新装环境 worker 注册失败时不炸子会话派活。
- `identity.js isRedteamPresetId` 扩识别 `redteam-worker`：不扩则子会话换挂后角色稿
  注入/落库链/会话绑定继承全失明（本刀唯一的隐性雷，盘底时逮住）。

### E2E 实锤（浏览器驱动真 UI 派活）

- recon 子会话 header：`"origin":"subagent","delegationDepth":1,"agentPreset":
  "redteam-worker"` + `agent-preset/selected` 事件（session.v4.jsonl.zstd 解压读）。
- 子会话工具目录：**零委派工具**（subagent/workflow 仅剩 host 注入的审批策略文本与
  工具描述字样），redteam_* 54 工具席全在，preflight 报 **15/15**。
- preset-declare.json：redteam `ok=true broken=null`（20 行）+ `workerOk=true`（14 行）。
- 解压配方：DSH runtime node 24 的 `node:zlib zstdDecompressSync`（node:zstd 模块
  不存在，zstd 1.5.7 编译进 zlib）。

## 刀三：办公识图根修（ZCode 执行体，d3577dc3）

2026-11-02 诊断定罪的「办公读图片不自动用 uocr2」落地（候选拍板①并联注入+②描述分工组合）：

- **刀A refs.go appendImageBlock 双腿并联**：识图（画面理解，1200 上限）与全文 OCR
  （`docmd.OCRImageText`，auto=uocr2→ovis→tesseract）goroutine 并行互不阻塞；OCR 腿
  15s 预算（包级 var `visionOCRBudget`，外层 select 强制——`docmd.OCRImageText` 无 ctx
  参数），冷启动不拖死本轮，放弃的 goroutine 后台继续拉起服务、就绪后下轮秒回；
  空/噪声（<12 字符）/失败静默降级，超时给「可调用 ocr 工具」提示行；识图失败回退
  占位块也带 OCR 全文（腿倒了另一腿活着照样值钱）。
- **顺手修真 bug**：`truncateVision` 字节硬切劈中文→改用现成 `truncateUTF8`（对齐
  sin 线口径）；注入标签【图片识别】→【画面理解】/【全文 OCR】分岗（全仓无其他消费方）。
- **刀B 描述分工三处+前端一行**：ocr 工具 Description 补触发时机（用户发图要文字/
  表格/票据内容时主动用，识图块只含画面理解）；系统提示「本地能力优先」补 ocr 工具
  点名（原只提 pdf 技能+vision）；read_file 读图片改给 ocr/vision 分工引导（原通用
  binary/markitdown 提示）；附件按钮 title 过时 OvisOCR2→uocr2。
- **坑**：single_prompt.go 是 raw string 字面量，prompt 文本里写反引号会把 Go 字符串
  劈出 syntax error（当场踩中当场修，工具名不加反引号）；refs_test 既有 3 例识图
  测试必须补 `ocrImageText` seam stub，否则单测真拉起 C:\AI\uocr2-llama（同 internal/app
  测试语义管线地雷形态）。
- 门禁：go build 全仓（ucrt64+CGO）+ control/builtin 全包 + agent prompt + app 定向族
  全绿 + tsc 0；新增测试 7 例（双注入/OCR 失败零回归/预算超时/UTF8 截断/门槛 gate/
  readfile 图片分支×2）。
- 遗留挂拍板池：sin 线（sin_file_refs.go）同构双注入；read_file 图片 OCR converter
  （刀A 落地后冗余，降级为错误信息引导已做）。

## 刀四：sin 线同构双注入（同日拍板即落，d1bef807）

刀三遗留拍板池项当场点掉——原罪 @图片附件补双腿并联（硬隔离两线最小同构，
实现独立于 control）：

- `sinImageBlock` 双腿：识图【画面理解】（rune 截断口径不变）+全文 OCR【全文 OCR】
  （`sinOCRImageText` seam=docmd auto 链，8KB 上限用既有 `sinTrimPartialUTF8`
  保边界）；`sinVisionOCRBudget` 15s 包级 var；噪声/失败静默降级、超时提示行；
  识图失败占位块也带全文。标签【图片识别】→【画面理解】对齐办公线分岗。
- 测试：既有例补 `stubSinOCR`（防真拉起 uocr2）+新增双注入/预算超时/门槛 gate
  3 例；门禁 go build 全仓+app Sin 定向族全绿。
- 至此办公+原罪两线 @图片注入语义对齐：画面理解与文字全文分岗双块。

## 刀五：扫描件 PDF fallback（候选③收官，d82f78c7+435cb236）

2026-11-02 候选③落地。**调研修正前提**：@引用其实已隐式带 OCR 腿
（ConvertLimit→pdfToMarkdownLimit→ocrPDFRange 文本层空自动 OCR），真缺口是
漏检闸、零预算、误导文案三处：

- **刀① docmd 漏检闸根修（d82f78c7）**：扫描仪页码戳类文本层（纯数字/符号，
  全行≤3 字符且无字母词且无 CJK）此前被当正文返回、永不触发 OCR；新
  `stampNoiseText` 严格谓词闸落 OCR 兜底。**教训**：首版用 meaningfulText
  整档闸，6 例既有测试当场翻车（甲/乙/丙/丁、单行 9 词等稀疏合法正文被推进
  真 pdftoppm OCR）——从严不误伤，收窄为严格谓词；CJK/词形戳记仍漏检，挂后续。
  附 pdftoppmLookup seam+过时注释重写。
- **刀② @引用预算+诚实文案（435cb236）**：办公分支 30s 预算（officeConvBudget
  包级 var，超时放弃后台继续）——此前 20 页扫描件 OCR（GPU 2-2.5min）同步
  阻塞消息发送零进度；失败文案 convErr 真因透传替换固定误导句。
- **刀③ read_file 扫描件指引（同笔）**：PDF 转不出文本改指 format_convert
  （扫描件自动 OCR）/summarize_file，不再误导 install markitdown；
  tryMarkItDownFn seam（本机装没装 markitdown 不 hermetic 的教训）。
- 门禁：go build 全仓+docmd/control/builtin 三包全绿+app PDF/OCR 族绿；
  新增测试 12 例。
- 坑：假 PDF 字节 `%PDF`+双 NUL 撞 BOM-less UTF-16 探测/本机 markitdown 真装
  ——合成件须配合 seam 才 hermetic。
- 遗留挂拍板池：混排 PDF 逐页判定（大手术）；ObjStm 页数=1 绕过页上限；
  成本导入 OCRImageText 直喂 PDF 字节（gaea_cost_import_vision.go:126）；
  docmd 无 OCR 结果缓存（wssearch textCache 形态可借）。

## 当日环境坑

- **双启撞 port=死亡进程遗言假象**：第一次 bash `&` 后台启宿主其实活了（占 3091），
  第二次 run_in_background 启动撞 EADDRINUSE 死亡，其上下文拆卸中途 tools 行
  `ctx.effect` 抛 `cannot create effect on inactive context` → preset-declare.json 写下
  registered-broken。**该 broken 是死亡进程的遗言，非代码回归**；清场单启后全绿。
  排障口径：broken 读数先核「该进程是否还活着+端口归属」，再查代码。
- DSH Web UI 真机走查：**对话标签页未激活时 composer 为 0×0**（内容可编辑 div 在
  DOM 但不可见），必须先点 tablist 里的「对话」；展开底部面板的按钮 label 在
  展开/折叠间互换，aria-label 会漂；Playwright locator 对该 UI 的 actionability 判定
  偏严（快照可见但 click 超时），PointerEvent 全链 dispatch 可靠。
- Python venv 的 impacket 0.12+ 不再生成 `impacket-*.exe` 包装，`secretsdump.py` 等
  直落 `venv/Scripts/`——检测面要认 .py 形态。

## 收尾与移交

- token 脱敏核实：全盘（dsh home+docs+AGENTS.md）零真 token；gaea 捕获层
  programming_web.go:292 落盘前已替换 `<redacted>`，11-08 commit 注记的警示闭环。
- 测试会话「红队演练台就绪确认」内新增连通性验证一轮（recon 子会话 inactive 已收），
  随旧测试会话一并可删。
- 拍板池（挂用户）：①Npcap 装否（masscan 实弹前置）；②exploit/内网子会话 REA 需求
  （决定是否拆 per-role 预设）；③模式下拉出现「红队执行（子会话）」属预期（registry
  无 hidden 位），如嫌碍眼需上游支持。
- dsh-wubi 仓 ahead 2 未 push；主仓本批仅本 day-log（引擎线在途文件属另一执行体，
  未触碰）。
