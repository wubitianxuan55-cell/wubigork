# gaea-wps-ocr-20261126-7f8ea22b

- 交付提交：`ac3a60bd`（`feat(docmd,office): PDF 转换 WPS 优先 + 本地 OCR 精简（2026-11-26）`）
- 同包提交：`7f8ea22b`（`docs(gaea): WPS 能力开发施工单记档 + PDF 转换 WPS 优先 / 本地 OCR 精简进度`）
- 产品版本：日常装机包，不抬版本（wails.json productVersion 未动）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-wps-ocr`，detached HEAD @ 7f8ea22b
  （主树零未提交业务改动），frontend/node_modules 为主树 junction
- 前端：`npm run build`（tsc -b + vite build），Vite `22.27s`
- 桌面：系统 `TMP/TEMP`（显式 `C:\Users\wubi\AppData\Local\Temp`）、`CGO_ENABLED=1`、
  ucrt64 gcc；PATH 前置 ucrt64 bin 与主树 `build/bin`；仓根 `wails build -s -ldflags
  "-s -w" -trimpath`，bindings 未跳过，26.5s
- wailsjs 零漂移：干净树构建后 `git diff --quiet -- frontend/wailsjs` 过
  （status M 全 CRLF 假象）；go.mod / go.sum 同样零内容差异
- 冒烟：正式 payload `/api/health` 返回 200（`{"status":"ok"}`，冒烟进程 PID 40788 已回收）
- 装机：包版四件先回写主树 `build/bin`（哈希对账一致），再 `gaea_install.ps1 -Launch`
  （默认 `-SourceDir`=主树 build/bin）→ INSTALL OK，装机位 `C:\AI\gaea-app`，
  新壳 PID 49952（11:58:27 启动，晚于构建时刻 11:55:26）
- 回写：包版 exe 已回写主树 `build/bin`；三个 DLL 与上一个包
  （gaea-wenxue-sync-20261126-e6d018b6）逐字节相同
- exe SHA256 `ca9b9d41038691663de2c5c698390ce5e93fc93684d027904e6e04aae324523b`
  （60,085,248 B）
- zip `8843f57820a9582d…`（33,132,919 B）；src.zip `48bf3ebe40fe7e3f…`（56,422,218 B，
  `git archive 7f8ea22b -- . ':(exclude)releases'`——排除 releases/ 防历届包嵌套雪球）；
  全量见 SHA256SUMS.txt

## 交付内容

### PDF 转换 WPS 优先（ac3a60bd）
起因：造价模式读 PDF 走 OCR 堵死，按停止后下一轮一直「排队中」。病根 = 事发 PDF 是
三明治件（JPEG 图像页 + 字形号文字层），自写提取整本提空，「空文本」分支直接落本地
OCR，WPS 只挂在「有文字但乱码」分支；`format_convert` 不看取消信号，代理循环死等。

- 提不出文字的 PDF 先走 WPS 在线 OCR（`pdf2word --scanned true --range`，100 页分段），
  所有消费方生效；WPS 不可用或失败才落本地 OCR（限 30 页）。
- `format_convert` 对文字型 PDF 也先经 WPS（真表格 + 图片），失败退回本地提取。
- 退出码 100/101/429 进 10 分钟全局静默；WPS 与本地 OCR 都失败时报错带两头真因。
- `rejoinSplitDecimals`：WPS OCR 把表格里小数拆成「3. 1」「115953. 12t」，保守并回。
- `format_convert` 收到取消即返回、不落盘，本轮立即收尾，下一条消息不再排队。
- 实测：174 页扫描件经真实 Go 链路 85 秒（103 图 / 77040 汉字 / 37 表）；
  之前 = 堵死 40 分钟直到重启。

### 本地 OCR 精简（ac3a60bd，用户确认只留 uocr2 llama 后端）
- 删 OvisOCR2、tesseract 两套 provider 与 uocr2 python 后端；`GAEA_OCR_ENGINE`
  空 / auto / uocr2 → uocr2，显式配置已退役引擎如实报「已退役」并指路，不静默换挡。
- 模型中心本地 OCR 三张卡 → 一张（`local:uocr2`）；配置残留的已退役定点值启动时清掉。
- pdf 技能脚本 `ocr_local.py` 只剩 WinRT + 本地视觉模型；前端文案与 mock 同步
  （办公 + 文学镜像）。

### 记录（7f8ea22b）
WPS 能力开发大项施工单（只记录未开工，下个会话进行）：
`进度计划/gaea-wps-capability-plan-20261126.md`；事实源
`C:\AI\re\wps-12.1.0.29690\docs\WPSCLI-CAPABILITIES.md`（含勘误：`clitool\wpscli.exe`
无门禁，11-24 报告测的「门禁」是 `office6\` 那个）。`.gaea/AGENTS.md` 新增
「转换解析优先用 WPS」约定。

## 门禁（打包前）
`go vet ./...` 0；go test docmd / tool / builtin / control / fileindex / largefile /
wssearch / config / internal/ocr 全绿；app 包 OCR / 本地服务卡 / 成本导入 / 工具面
预算 / 绑定完整性全绿，整包 148s 仅 `TestGaeaPriceFetchAllTask` 偶红（既有缺陷：
`pricefeed.SaveFetch` 用纳秒时间戳生成 ID 会撞号，单跑在改动前后都过，本批未动）；
前端 tsc -b 0、vitest 全量 4780 例、改动文件 eslint 0；check-bindings-drift OK；
node scripts/check-docs.mjs OK。顺手清两条既有假红（`TestBuildUocr2LlamaServerCmd`
夹具路径、`SettingsPage.test.tsx` 过期的「晨报预载」断言）。

## 未做 / 余项
- 装机后真机走查未做（要在用户工作区的会话里发消息，未替用户发）：造价模式
  `format_convert` 那份 174 页扫描件；转换中按停止再发消息不排队。
- 退役引擎的模型目录：打包时未删；**2026-11-26 12:09 经用户确认后已送入回收站**
  （`C:\AI\gaea-OCR\unlimited` ≈ 14.9 GB、`C:\AI\gaea-OCR\ovis` ≈ 0.8 GB；可恢复性、
  归档与重建链见 `C:\AI\re\unlimited-ocr-2\README.md`，过程见 `.gaea/progress.md` 补记 2）。
  原清单：`unlimited`（70,305 文件 / 15,964,598,671 B）、`ovis`（22 文件 / 860,116,904 B）。
- agent 循环层对「不看 ctx 的慢工具」仍会 `wg.Wait()` 死等（本批只在 format_convert
  工具级解决）；正文汉字间 OCR 空格未归一化；WPS `--ai-fix`、在线引擎单次页数上限
  未验证。
