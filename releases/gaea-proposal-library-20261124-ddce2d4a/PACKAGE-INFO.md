# gaea-proposal-library-20261124-ddce2d4a

- 交付提交：`ddce2d4a`（`feat(gaea): retire the knowledge base and build the
  proposal library`）
- 前置提交：`9b5b8724`（`feat(wenxue): feedback batch — hanzi gate, chapter
  deliver, reading typography; retire five novel skills`，11-23 文学实测反馈批，
  另一会话完成、本次工作树收口入库）
- 基础提交：`aa2f4360`（`docs(gaea): record xai login fixes`）
- 产品版本：`4.544.0`（日常装机包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-ddce2d4a`，detached HEAD，
  node_modules 为主树 junction
- 前端：`npm.cmd run build`，Vite `35.04s`，6608 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；PATH 前置主树
  `build/bin`（DLL 三件套）；`wails build -s -ldflags "-s -w" -trimpath`，
  bindings 未跳过，Wails `1m5.284s`
- 漂移：Wails 再生仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol` 内容差 `0`
- 冒烟：正式 payload `/api/health` 返回 **200**（PID 7496 已回收）

## 交付内容

- 知识库整体退役：两包、两工具、13 个绑定下导出，前端面板 / 导入弹窗 /
  记忆中枢入口 / 快捷键 / 三语键全清；造价导入用文本提取迁至
  `internal/app/gaea_import_text.go`
- 方案库存储层 `internal/gaea/proposal`（SchemaV35 / V36）：标题树 1~4 级 /
  小节逐字正文 / 语义块三层；按源文件标题行切正文的分批续录；全文路 +
  语义路 → 名次融合 → 本地 reranker 精排，模型不可用逐级降级
- 工具 `proposal_save` / `proposal_search` / `proposal_read` 重写；绑定
  `ProposalB` 五方法（List / Outline / Read / Search / Delete，绑定面 724）
- 一级板块「方案库」`ProposalLibraryPage`：方案清单 / 命中 → 标题树 → 阅读窗，
  检索条与类型 / 年度 / 层级筛选，四态与空库引导，窄窗页签；menuOrder=6
  紧随造价库，众神殿及其后板块序号 +1
- 方案模式 ModeLine / 身份提示词 / proposal-longform 技能同步录入与取材流程
- 文学实测反馈批（前置提交 `9b5b8724`，《软件不足与建议》，64 文件）：
  工具层五刀（汉字口径回执 / 字数硬门禁 / 角色卡匹配 /
  `wenxue_chapter_deliver` 单章交付 / 定稿内联体检）；阅读室排版偏好（字体、
  字号、Ctrl+滚轮缩放，存 localStorage）；story-long-write / story-deslop /
  novel-agents / novel-deslop / novel-review 五个技能按用户拍板整批退役

## 验证

- Go 全量（ucrt64、cgo 开启、APPDATA 重定向临时目录）：**131 包通过**；
  仅 `internal/docmd`（uocr2 环境）与 `internal/voice/realtime`（实弹探针）
  两包为既有环境红，均不依赖本次改动
- 前端全量：517 文件 4734 例，本次改动相关全绿（导航栏测试同步 13 板块）；
  安全面板 1 例、数据库存档 2 例负载期失败，单跑全绿
- `tsc -b` 0；改动面 ESLint 0；vet 0；绑定漂移 OK@724
- 真实技术标（27.3 万字 / 419 页 / 241 标题）：录入 0.2s、954 块 bge-m3
  向量化 24s、检索 0.5~1.3s 含精排，精确词与自然语言问法均命中正确小节
- 真实程序经 HTTP 桥调用五绑定：返回字段形状与前端类型一致；借此抓到并
  修复「层级 / 类型 /年度过滤对语义路不生效」
- 无头浏览器走查：亮色主界面 / 检索命中 / 删除确认 / 暗色 / 窄窗页签 /
  空库引导，桌面与窄窗均无横向溢出（截图 `temp/proposal-walk/`）
- 最终 payload 已覆盖安装，`INSTALL OK`，PID `32336`，启动时间
  `07:04:08` > 构建时间 `07:03:00`；装机 exe 与 payload 同哈希；安装目录
  仅 exe + 三 DLL，桌面仅 `gaea.lnk`

## 须知情

- 正式库已为 SchemaV36：工具包全量测试有既有用例会打开正式库（未定位到
  具体用例），V36 挂进迁移链后跑测试即被应用。只加列加表，旧版兼容。
- 正式库留有 11-23 测试写入的夹具方案「包3技术标 第三版」（3 节、每节
  十余字），未动；可在方案库界面删除。
- `node scripts/check-docs.mjs` 既有 FAIL：`docs/README.md` 未登记
  `gaea-agent-memory-nebula-distill-2026-11.md`、`gaea-day-log-2026-11-22.md`
  （他线文档）。
- 未验证：真实 LLM 在标题稀疏文档上自行定级录入；真实壳内页面渲染
  （WebView2 157 下调试端口未监听，改以 HTTP 桥核对绑定）。

## 包布局

- `payload/gaea.exe`
- `payload/onnxruntime.dll`
- `payload/sherpa-onnx-c-api.dll`
- `payload/sherpa-onnx-cxx-api.dll`
- `gaea-proposal-library-20261124-ddce2d4a.zip`：上述四件运行文件
- `gaea-proposal-library-20261124-ddce2d4a-src.zip`：
  `git archive` 自交付提交的源码包

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `71205F079B9D6796D93756E3A198AE248169F1E0B72B0581D6DF566201B6309D` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-proposal-library-20261124-ddce2d4a.zip | `59AC02D6E31BC1AC3D4640F84E92B85D6870CEF9CA57BB1FDEE95BD8EBB80226` |
| gaea-proposal-library-20261124-ddce2d4a-src.zip | `71F2D2359EBD76E220E00CB0AC37A29F35B6E273A7AE486F78825004C001B3FA` |

## 边界

- 未 push（本地 main 领先 origin）。
- 日常装机包，不抬产品版本。
- ZIP / 源码包留盘不入库；`PACKAGE-INFO.md` 与 `SHA256SUMS.txt` 入库。

## 记录提交前复核（2026-11-24 07:35 起，新会话续做）

上一会话完成构建、冒烟、装机并写就本文件后中断，记录未提交。此时 HEAD 仍为
`ddce2d4a`，故**未重建**——同提交重建只会得到同名包，还会打断装机位与包的
同哈希关系——改为独立复核：

- 六项 SHA256 逐项重算，与本文件一致
- 源码包与 `git archive --format=zip ddce2d4a` 重生成件逐字节相同（6548 文件）
- 运行包四项解包哈希 = payload
- 装机位 `C:\AI\gaea-app` 四件 = payload = 构建工作树 `build\bin\gaea.exe`
- 构建工作树：29 个状态改动文件去 CR 逐个比对，内容差 0；无非忽略的未跟踪
  文件；`package-lock.json` 与提交一致
- 装机实例 PID `32336`（启动 `07:04:08`）仍在运行，诊断口
  `127.0.0.1:18123/healthz` 返回 `ok`（200）
- **冒烟未复跑**：`scripts/smoke.ps1` 不隔离工作区，会拉起与用户在用壳同名的
  第二实例。冒烟 200 沿用上一会话对同一哈希 payload 的记录，非复核会话实测。
- 桌面另有用户图片 `gaea-1794829783263-seed50000.jpg`，不是装机散件，未动。
