# gaea 上游蒸馏与安全加固执行计划（2026-10-23）

> **状态：B0–B4 DONE / B5 DONE（代码+mock）/ B5 真机验证挂池（2026-10-24）**
>
> **审计基线**：`ce56c783`，产品版本 v4.523.0，工作树含未提交的 CAD/工作台改动。
>
> **当前入口**：B0–B5 的代码与自动化门禁已收口；后续是否提交、拆提交或发版由用户决定。
> B5 真机已切换为官方 Qwen Image 2.1 Turbo Edit（内部 ID 保持 `qwen-image-2.1-edit`）：
> 单图产品级生成通过；4 图参考在 5 分钟门控内未完成，中文图上文字真机失败（输出噪声）。
> 旧 2511 已退役并从产品链路删除，不再回退；不要自动下载权重或改只读上游。
>
> **来源**：2026-10-23 全仓代码审计 + Reasonix v1.39.10、DeepSeek Harness
> 0.2.1-rc.2、oh-story 0.9.0、ComfyUI 0.42.0 上游同步。
>
> **纪律**：只重推导机制，不整包搬上游代码；`clones/` 与
> `C:\AI\ComfyUI\ComfyUI` 只读；保留工作树已有改动；未经用户明确要求不 commit/push。

## 0. 下个会话直接开始

执行顺序固定如下，不跳批：

1. `git status --short --branch`，确认 HEAD 仍是 `ce56c783` 或记录新 HEAD。
2. 若目标文件自 2026-10-23 21:20 后被并发改动，先重读该文件，再按本计划施加改动；
   不 stash、不 reset、不覆盖其他会话的改动。
3. 执行 **B0 发布阻断清零**。
4. B0 全绿后，直接进入 **B1 文件访问边界 + 子代理 hooks**，不再询问。
5. B1 全绿后，按同一会话的时间预算继续 **B2 CAD 收口**、**B3 附件与 provider**。
6. B4/B5 是产品增量，必须在前三批全绿后另起独立提交/发版节奏。

本计划默认：

- 每批先写失败测试，再实现；不得用“全量测试绿”代替缺失场景。
- 定向测试先跑，批次收口跑一次全量；不重复同一全量测试。
- 结构改造和依赖升级不混在同一提交。
- 每个批次结束时按 §8 更新本文件的状态行与实测数字。

## 1. 已锁定的架构决策

| 决策 | 锁定口径 |
|---|---|
| 安全边界 | workspace 应同时约束读、预览、写；不能再只是“写隔离” |
| 外部读 | 默认读根 = workspace + `allow_write`；新增 `allow_read` 只扩展读；UI 对话框已选文件视为显式授权 |
| 子代理 | 用户 hooks 必须继承；权限 Gate 仍是独立层，不得用 Gate 代替 hooks |
| 附件 | 前端按元数据先拒；后端在 base64 解码前拒；固定安全上限优先，用户可配置放到后续刀 |
| 默认附件上限 | 普通文件 **25 MiB**；图片维持现有 **10 MiB**，不照抄上游 64 MiB 的 Wails 桥接负载 |
| 文件落盘 | 上传目录使用目录句柄与独占创建；失败按句柄清理，不使用可被替换的二次路径删除 |
| Provider | 区分发送阶段、响应头阶段、流阶段；只对确定未完成发送的请求重发一次 |
| CAD | 当前分支必须先收口数据串台和资源上限，不能把新卡直接带病发布 |
| 小说 | 不重做已收官的 oh-story T1~T6；只吸收 v0.9.0 的作者记忆与三层去味增量 |
| 图像 | 先接 Qwen Image 2.1 Turbo Edit；MOSS-Image 另批；不自动下载权重 |
| 上游排除 | Unsloth Studio、DSH 插件市场/远程 RBAC/CUA、ComfyUI Partner/API Nodes 不进本计划 |

## 2. B0：发布阻断清零

**目标**：让当前工作树重新满足仓库自己的发布前门禁。

### B0.1 依赖安全

处理文件：

- `go.mod`
- `go.sum`
- `frontend/package.json`
- `frontend/pnpm-lock.yaml`

执行：

1. Go toolchain 升到 **1.26.10**：
   - `go 1.26.10`
   - `toolchain go1.26.10`
2. `golang.org/x/net` 至少升到 **v0.60.0**；若依赖图允许，直接取最新修复版。
3. `github.com/xuri/excelize/v2` 升到包含以下全部修复的版本：
   - GO-2026-6689 CFB 内存耗尽
   - GO-2026-6688 数组公式栈溢出
   - GO-2026-6685 `Rows.Columns` 超长循环
   - GO-2026-6684 畸形加密工作簿 panic
   - 首选固定伪版本：`v2.11.1-0.20260930021559-01a9ff32fb3c` 或更新版本；
     升级后用 `govulncheck` 证明四项消失。
4. `dompurify` 升到 **>=3.4.17**。
5. `katex` 升到 **>=0.18.2**；验证与 `rehype-katex` 的兼容，不做忽略 peer warning 的假绿。

验收：

```powershell
go version
govulncheck ./...
cd frontend
pnpm audit --prod
```

期望：`govulncheck` 与 `pnpm audit --prod` 均无可达/生产漏洞。

### B0.2 清 lint 阻断

文件：`internal/gaea/cad/count.go`

- 删除未使用字段 `sheetBoxs`。
- 三处 `reflect.Ptr` 改为 `reflect.Pointer`。
- 保留现有 `Count` 行为，不顺手重构反射面。

验收：

```powershell
golangci-lint run --timeout=10m
```

期望：0 issues。

### B0.3 文档门禁

- `docs/README.md` 登记：
  - `gaea-cad-poc-report-2026-10.md`
  - `gaea-cad-recognition-market-survey-2026-10.md`
  - `gaea-cad-tool-master-plan-2026-10.md`
- `docs/gaea-cad-tool-master-plan-2026-10.md` 中不存在的
  `docs/gaea-cad-poc-report-2026-11.md` 改为实际文件 `2026-10`。
- 若 CAD 总计划状态与新实现不一致，只修事实与链接，不重写已完成内容。

验收：

```powershell
node scripts/check-docs.mjs
```

期望：文档登记 0 缺失、悬空引用 0。

### B0.4 当前正确性回归

- CAD 换图纸必须清空上一份 `count`；见 B2。
- `frontend/package.json` 版本仍是 4.128.0，与当前 4.523.0 漂移。
  B0 只在即将发版时用统一版本脚本同步；不要现在手工制造第四个版本源。
- `CadPanel.tsx` ESLint 的 `rasterWidth` 依赖 warning 在 B2 一并修。

### B0 完成门禁

```powershell
go build ./...
go vet ./...
golangci-lint run --timeout=10m
go test ./... -count=1
cd frontend
pnpm exec tsc -b --noEmit --force
pnpm run lint
pnpm run knip
pnpm run build
pnpm exec vitest run --reporter=dot
cd ..
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-bindings-drift.ps1
node scripts/check-contract-drift.mjs
node scripts/frontend-e-check.mjs
node scripts/check-docs.mjs
govulncheck ./...
```

### B0 执行记录（2026-10-23）

- **HEAD**：`ce56c783`（未 commit/push）；保留执行前 CAD/工作台未提交改动。
- **依赖**：Go 1.26.10 + `toolchain go1.26.10`；x/net 0.61.0；
  excelize `v2.11.1-0.20261020033641-611b222fd8f7`（计划指定版本允许的更新修复版）；
  DOMPurify 3.4.17、KaTeX 0.18.2，pnpm 11 的传递依赖经
  `pnpm-workspace.yaml` overrides 统一。`govulncheck` 可达漏洞 0；
  `pnpm audit --prod` 漏洞 0。
- **兼容**：excelize 新版的 `SetConditionalFormat` 只接受
  `NewConditionalStyle` 产生的 dxfs；条件格式测试夹具改用官方 API，
  净删 61 行脆弱的手工 zip/dxfs 注入代码。Markdown/KaTeX/Mermaid 定向 38 例通过。
- **Lint/文档**：CAD `sheetBoxs` 删除、`reflect.Pointer` 迁移；
  `golangci-lint` 0 issues。三份 CAD 文档已登记，PoC 链接改回实际的
  `2026-10` 文件；文档守卫 0 孤儿 / 0 悬空。
- **全量门禁**：Go build/vet/test 全绿；前端 tsc/build/knip 通过；
  vitest **454 文件 / 3951 用例全绿**；绑定 drift PASS@675、契约无新增漂移、
  E 系列守卫通过。前端 lint 0 error / 7 既有 warning（CAD `rasterWidth`
  一条按计划留到 B2）。
- **绑定面**：675，不变；未改公共接口或产品版本。

## 3. B1：文件访问边界与子代理 hooks

**上游依据**

- Reasonix `11b8ceb38`：TargetAccess、preview confinement、读写分根、无策略 fail-closed。
- Reasonix `896faf36d`：子代理通过独立 session runner 继承 hooks。

**gaea 已证实缺口**

- `internal/gaea/permission/permission.go:122`：只读工具默认 Allow。
- `internal/gaea/tool/builtin/readfile.go:99`：可打开任意绝对路径。
- `internal/gaea/agent/tool_precheck.go:40`：写入预检查先读文件；第 54 行回显前 200 字节。
- `internal/gaea/boot/boot.go:296`：TaskTool 装配未传 hooks。
- `internal/gaea/agent/task.go:722`：子代理 Options 未设置 Hooks。

### B1.1 AccessPolicy 抽象

新增 `internal/gaea/tool/access.go`：

```go
type AccessPolicy interface {
    CheckRead(ctx context.Context, path string) error
    CheckPreview(ctx context.Context, path string) error
    CheckWrite(ctx context.Context, path string) error
}

type ReadTargeter interface {
    ReadTarget(args json.RawMessage) (string, bool)
}

type ReadPathResolver interface {
    ReadPaths(ctx context.Context, args json.RawMessage) ([]string, error)
}

type WritePathResolver interface {
    WritePaths(args json.RawMessage) ([]string, error)
}
```

路径规范化复用 `internal/gaea/wspath`，禁止另写字符串前缀判断。

### B1.2 配置

文件：

- `internal/gaea/config/config.go`
- `internal/gaea/config/render.go`
- `internal/gaea/config/render_preserve_test.go`

在 `SandboxConfig` 增加：

```toml
[sandbox]
allow_read = []
```

规则：

- 默认读根 = workspace + `allow_write`。
- `allow_read` 只扩展读，不扩展写。
- UI 已选文件通过 `ReadPathAllowed func(string) bool` 回调放行；不要让 agent 包 import app。
- 生产策略缺失必须 fail-closed；测试显式传 TempDir 根。

### B1.3 Agent 接入

文件：

- `internal/gaea/agent/agent_config.go`
- `internal/gaea/agent/agent.go`
- `internal/gaea/agent/execute_one.go`
- `internal/gaea/agent/tool_precheck.go`
- `internal/gaea/tool/builtin/readfile.go`
- `internal/gaea/tool/builtin/editfile.go`
- `internal/gaea/tool/builtin/multiedit.go`
- `internal/gaea/tool/builtin/writefile.go`
- `internal/gaea/tool/builtin/movefile.go`
- `internal/gaea/tool/builtin/grep.go`
- `internal/gaea/tool/builtin/glob.go`

顺序：

1. `executeOne` 在 `precheckTool` 之前调用 AccessPolicy。
2. 文件工具实现路径解析接口，禁止 agent 按工具名维护新白名单。
3. `readFileForPrecheck` 只接受已通过 `CheckPreview` 的路径；拒绝结果不含文件正文。
4. 写执行仍使用现有 writer confinement，AccessPolicy 是执行前统一入口，不替代工具自身防线。
5. TaskTool 的 `subOpts.WriteRoots/ReadRoots` 与 AccessPolicy 同步继承。

### B1.4 子代理 hooks

文件：

- `internal/gaea/agent/task.go`
- `internal/gaea/boot/boot.go`
- `internal/gaea/hook/runner.go`
- `internal/gaea/hook/runner_test.go`
- `internal/gaea/agent/task_test.go` 或新增 hooks 子域测试

实现：

1. `TaskTool` 增加 `hooksForSession func(string) ToolHooks`。
2. 新增 `TaskTool.WithHooksForSession`。
3. `runSubSession` 用 `run.Ref` 取 child hooks，写入 `subOpts.Hooks`。
4. 生产 boot 为 TaskTool 接线。
5. 过渡版可以复用同一不可变 runner，但最终必须给子会话独立 session id。
6. `retry_until` 的 bash 检查同样走 child hooks，不保留测试专用接线。

### B1 必测

- `read_file` 读 workspace 外文件 → 拒绝。
- `edit_file` 预检查 workspace 外文件 → 不调用 `os.ReadFile`，结果无正文。
- `allow_write` 目录可读可写；`allow_read` 目录可读不可写。
- UI 已选文件可读，其他绝对路径仍拒绝。
- symlink/junction、大小写变体、跨卷 `..` 逃逸全部拒绝。
- 父级 `PreToolUse` 禁 bash → `task` 子代理与 `retry_until` 都被拒。
- 子代理 hook payload 的 session id 为 `sa_...`，不污染父会话。

## 4. B2：CAD 当前版本收口

**目标**：先让已落地的 CAD 一期可发，不新增二期功能。

文件：

- `frontend/src/gaea/components/cad/CadPanel.tsx`
- `frontend/src/gaea/components/cad/CadPanel.test.tsx`
- `internal/gaea/cad/cad.go`
- `internal/gaea/cad/pdf.go`
- `internal/gaea/cad/cad_test.go`
- `internal/app/gaea_cad.go`
- `internal/app/gaea_cad_test.go`

### B2.1 前端状态隔离

1. 打开新图纸时清空 `count/countBusy/texts/textsBusy/raster/rasterWidth/answer/question/copied`。
2. 增加 `requestSeq`；`load`、`fetchRaster`、`loadTexts`、`loadCount`、`ask` 返回时校验：
   - 当前 file path 未变；
   - 当前 sheet 未变；
   - seq 仍是最新。
3. `import_` 在 `busy` 时禁用；recent 按钮同样禁用。
4. `selectSheet` 在 `askBusy` 时不切；或回答必须绑定发起时的 sheet。
5. `handoffToAI` 只允许使用当前 file + sheet 代际的 count/texts。

必测：

- A 图打开并统计，再打开 B 图：B 图绝不显示/发送 A 图 count。
- B 的慢 Open 晚于 A 返回：A 结果不能覆盖 B。
- 快速切 sheet，旧 sheet 的 raster/answer 不能覆盖新 sheet。
- ask 未完成时换图/图框：答案丢弃并提示或绑定原图，不串到新图。

### B2.2 后端资源上限

1. `Open` 前 `Stat`：
   - 文件类型正确；
   - 普通文件；
   - `DWG/DXF` 默认上限 **32 MiB**，PDF **128 MiB**；值集中常量。
2. `os.ReadFile` 改为有界读取；超限返回明确错误。
3. `RenderSheet/RenderSheets` width 上限 **8000 px**，下限 64 px。
4. 图框数量上限与总输出预算：
   - 单图纸最多 100 个图框；
   - 预计总 PNG 内存/磁盘超预算时拒绝或分批渲染。
5. `OpenPDF`：
   - 用 `pdftoppm -f 1 -l maxPages` 只渲染前 N 页；
   - 不要先全量渲染后截断；
   - N 默认 50。
6. `docmd.RenderPDFPages` 不顺手改公共行为；CAD 新增带页范围与 context 的窄接口。
7. `exec.CommandContext` + 超时；取消/超时杀进程树。
8. PNG 与缓存元数据使用原子写（temp+rename），并发重复渲染至少不产生半文件。

### B2.3 UI 诚实性

- 展示 `count.warnings`；无块引用时说明“散线或天正对象不适用块统计”。
- 错误文案区分打开、统计、渲染、问图纸。
- 修 `ask` 的 `rasterWidth` 依赖 warning。

### B2 验收

- 当前 CAD 测试 + 新增竞态/超限/PDF 页数测试全绿。
- 生成一个 51 页合成 PDF，证明 `pdftoppm` 只产出 50 页中间文件。
- 大/伪造扩展名/损坏文件不上报 panic。
- `pnpm run lint` 无 CAD 新 warning，`pnpm run build` 通过。

## 5. B3：附件准入与 Provider 发送保护

### B3.1 附件单一安全实现

**已证实问题**

- `internal/gaea/control/attachments.go` 的安全函数仅被测试调用，生产无接线。
- `frontend/src/gaea/hooks/useComposerAttachments.ts:73` 先读文件，10MB 只是确认框。
- `internal/app/gaea_ui_extra.go:867/900` 自己重复实现，完整 base64 先解码，无硬上限。

新增叶子包 `internal/gaea/attachment`：

- `limits.go`：普通文件 25 MiB、图片 10 MiB、base64 上界。
- `store.go`：`OpenRoot` 固定上传目录，独占创建，成功后原子可见。
- `image.go`：MIME 探测与图片限制。
- `file.go`：非图片附件。

接线：

- `GaeaSavePastedImage`、`GaeaSaveAttachmentFile` 委托单一服务。
- `GaeaReadFileB64` 保持现有 64 MiB，但检查移到读取前。
- `useComposerAttachments` 增加 `splitOversizeAttachments`；超限文件不进入 `FileReader`。
- 粘贴、拖拽、截图三条路径共用限制。

必测：

- 超限文件前端未发生 `readAsDataURL/readAsArrayBuffer`。
- 超限 base64 后端未发生 `DecodeString`。
- 伪造 MIME、上传目录 symlink、半途失败无残文件、同毫秒路径不碰撞。
- 现有图片流程、预览、识图、OCR 零回归。

### B3.2 Provider 发送阶段保护

文件：

- `internal/ai/client.go`
- `internal/ai/client_stream.go`
- `internal/ai/client_failover.go`
- 新增 `internal/ai/send_guard.go`

实现要点：

1. 用 `httptrace.GetConn/WroteRequest` 建发送阶段 guard。
2. 发送预算 300 秒；成功后交给现有 `ResponseHeaderTimeout`。
3. 只取消 attempt context，不设置共享 socket deadline，不关闭 HTTP/2 连接。
4. 用户 cancel / outer deadline 永远优先。
5. 只有“确定未完成发送”或“复用 idle 连接已死”允许一次重发。
6. 5xx 重试保持现状；响应头超时不重发，避免重复生成。
7. 诊断只记录白名单原因码：`request_send_timeout`、`eof`、`reset`、
   `timeout`、`dns`、`http2_protocol`。

必测：

- 上传阶段卡住 → 300 秒超时 + 最多重发一次。
- 完整写完后的响应头超时 → 不重发。
- 健康 SSE 持续超过 300 秒 → 不中断。
- 用户取消优先于发送超时。
- redirect / Transport retry 每次重新 arm。
- HTTP/2 sibling stream 不受影响。

## 6. B4：小说作者记忆与三层去味

**上游依据**：oh-story v0.9.0（2026-10-17 tag；main `c3fbbbe`）。

### B4.1 作者记忆

新增 `internal/authorprefs`：

```text
model.go       // preference/scope/evidence/status
store.go       // AP/BP 两级、revision、幂等 event、migration
query.go       // kind+scope 选路、importance 排序、2048B 上限
views.go       // 作者画像/待确认派生视图
render.go      // 生成上下文最小注入
```

存储：

- workspace：`<workspace>/.gaea/author-memory/`
- book：`<book>/.gaea/author-memory/`

契约：

- 项目级 ID `AP`，书级 ID `BP`。
- 只接受 explicit user / accepted suggestion / manual。
- 同一 event 幂等；revision 不倒退。
- global/genre/workflow 与 book 分开存储。
- book 例外覆盖 global，不算冲突。
- 冲突进 pending/conflict，未确认不得生效。
- query 只返回注入 lines 和 omitted IDs；不得把完整画像注入模型。
- 写入成功才有 receipt；注入超预算不否定写入。

接入：

- `internal/app/create_chapter_context.go`：按阶段注入。
- `internal/app/style_digest_handler.go`：显式偏好优先，学习 digest 跟随。
- 新工具 `author_memory`：`remember/decide/replace/forget/query`。
- 不在每条用户消息后自动推断偏好。

### B4.2 三层去味

不重写 `novelstyle` 引擎，只加层级：

- expression：套词/句式/声口/模板复现。
- narrative：详略/关键过程/细节功能/情绪承接/收尾。
- story：行为依据/因果/立场/结果/知情顺序。

修改：

- `.gaea/skills/novel-deslop/gates.json`
- `.gaea/skills/novel-deslop/SKILL.md`
- `internal/novelstyle` issue 增加 layer 元数据
- `internal/novelgate` 报告增加 layer

铁律：

- 表达可自动改。
- 用笔只做有原文依据的压缩/展开。
- 故事新增证据、改动机、改胜负、改揭露顺序必须另行授权。
- 脚本零命中不能证明后两层合格。

### B4 验收

- 两级 store 往返、幂等重放、冲突、替换、遗忘、迁移。
- 2048B 查询硬上限和 omitted IDs。
- 开关为零时章节生成逐字节不变。
- 三层各至少 3 个正例/3 个反例；干净文本零误报。

## 7. B5：ComfyUI Qwen Image 2.1 Turbo Edit

**上游依据**：ComfyUI 0.42.0 / workflow_templates v0.11.96，
`image_qwen_image_2_1_image_edit`。

现有资产：

- gaea 已有 Qwen2.1 T2I：`qwen_image_2.1_turbo_int8_convrot`、
  `qwen3vl_8b_int8_convrot`、`qwen_image_2.1_vae_bf16`。
- 2026-10-24 用户已删除旧 Qwen-Image-Edit 2511；基础 2.1 Edit UNET 未安装，执行口径改为
  官方 Turbo Edit，复用现有 `qwen_image_2.1_turbo_int8_convrot.safetensors`。

实现：

1. 新增 `qwen-image-2.1-edit` 模型/编辑引擎。
2. 新增 `buildQwen21EditWorkflow`：
   - UNETLoader + CLIPLoader(type=qwen_image) + VAELoader。
   - `TextEncodeQwenImage21` 使用 `images.image_1..N`、prompt、negative、resolution。
   - `QwenImage21Cache -> CFGGuider(cfg=8) -> SamplerCustomAdvanced`
     + `ManualSigmas` 按官方 Turbo 图接线。
   - 输出 VAEDecode -> SaveImage。
3. 参考图上限从 3 扩到 **10**；第 11 张明确拒绝。
4. 提示词增强器默认关闭；开启时单独登记额外权重与耗时。
5. 缺权重提示三件套；普通 500/网络错误不误挂下载提示。
6. 2511 已按 2026-10-24 用户指令删除，不保留回退；历史任务仅保留数据记录。
7. 模型中心、image_handler、comfyAlways、前端模型列四处同步。

必测：

- 单图/四图/十图 golden workflow。
- 第 11 图拒绝。
- qwen2.1 三件权重文件名、CLIP type、CFG=1、resolution。
- 缺增强器但 enhancer=off 可运行。
- 缺三件核心权重时错误含准确文件名与目录。
- 真机至少验证单图改衣、4 图角色一致性、中文图上文字。

### B1–B5 执行记录（2026-10-23）

> 执行期间 HEAD 始终为 `ce56c783`，未 commit/push/发版。相对 HEAD 的累计规模为
> tracked `+2030 / -1359`，另有 58 个未跟踪文件 `+8907` 行，合计 `+10937 / -1359`
> （净 `+9578`，本次快照）。该数字包含执行前已有的 CAD/工作台改动与并发新增文件，
> 无法可靠按批次归因，
> 故以下记录以文件面、定向场景和门禁结果为准。

**B1 DONE：文件访问边界 + 子代理 hooks**

- 新增 `internal/gaea/tool/access.go`、agent 侧 AccessPolicy 接线，以及
  `wspath.RealPath` 的跨平台真实路径解析；配置新增只读 `allow_read`，
  `allow_write` 自动可读，UI 已选文件通过显式回调仅放行读。
- file/grep/ls/vision/format/chart/diagram/schedule/sidebar/largefile/OCR 等路径工具
  统一执行前校验；subagent 继承同一策略。用户 hooks 使用独立 `sa_` session view，
  `retry_until` 同样经过 child hooks。
- 定向覆盖 symlink/junction、跨卷与大小写路径、越界正文不泄漏、读根不可写、
  hooks 阻断 bash；新增测试合计 19 个场景。未改公共绑定面。

**B2 DONE：CAD 当前版本收口**

- `CadPanel` 使用 generation + request sequence 隔离 Open/Raster/Texts/Count/Ask 的迟到结果，
  新图纸清空旧状态，ask 期间禁止换图框，handoff 仅使用当前代际数据。
- CAD 后端新增 DWG/DXF 32 MiB、PDF 128 MiB、100 图框、512 MiB PNG 总预算、
  64–8000 px 渲染宽度边界；PDF 仅渲染前 50 页，pdftoppm 支持 context 取消并杀进程树；
  PNG/cache 元数据原子写。
- `internal/gaea/cad` 11 个测试、app 3 个测试、前端 CAD 14 个用例通过；
  51 页合成 PDF 实测只产出 50 页。前端 lint 0 error / 6 个既有 warning，无 CAD 新 warning。

**B3 DONE：附件准入 + Provider 发送保护**

- 新增 `internal/gaea/attachment` 单一安全实现：普通文件 25 MiB、图片 10 MiB，
  base64 解码前拒绝超限；`os.Root` 固定目录、独占临时文件、原子改名、真实路径与 MIME 校验。
  粘贴图片与附件绑定已接入，旧 control 重复实现删除；前端三类附件入口均在 FileReader 前分拣。
- 新增 `internal/ai/send_guard.go`：发送阶段 300 秒预算、请求写完才启动响应头计时、
  仅取消 attempt context；只对确定未写完发送安全重发一次，响应头超时不重发、不触发 failover，
  已建连 SSE 持续超过预算仍保持。
- attachment 7 个后端场景、send guard 6 个场景、附件前端 12 个场景通过；未改公共绑定面。

**B4 DONE：作者记忆 + 三层去味**

- 新增 `internal/authorprefs`：AP/BP 两级存储、幂等 event、单调 revision、
  conflict/pending/accepted、book 覆盖 global、replace/forget/query、2048 字节注入上限与
  omitted IDs；新增 play 空间 `author_memory` 工具并接入章节与风格上下文。
- `novelstyle.TasteIssue`、`novelgate.Issue` 增加 expression/narrative/story layer；
  novel-deslop 新增 story 层 H/I/J 授权门禁，明确脚本零命中不能证明后两层合格。
- authorprefs 6 个场景、novelstyle/novelgate layer 测试通过；开关为零时原生成链保持。
  新工具属于 agent 内部工具，不增加 Wails 产品绑定。

**B5 DONE（代码+mock）：Qwen Image 2.1 Turbo Edit**

- 保持 `qwen-image-2.1-edit` 内部 ID 兼容已有配置，UI 如实显示 “Qwen Image 2.1 Turbo Edit”；
  工作流改用官方 Turbo Edit：现有三件核心权重、`TextEncodeQwenImage21` 的
  `images.image_1..N`、`QwenImage21Cache → CFGGuider(cfg=8) → SamplerCustomAdvanced`
  （euler + 官方 8 步 `ManualSigmas`）、1/4/10 参考图、第 11 图拒绝；增强器默认关闭。
- 模型目录、图片模型下拉、catalog、comfyAlways 与配置注释已同步；缺核心权重提示包含准确文件名。
- 合规面：Qwen Image 2.1 沿用项目既有红线 `Qwen Research License（禁商用）`；catalog
  license hint、模型选择页警示与编辑弹窗说明已显形，并补目录/UI 回归测试。
  旧 2511 不再作为商用回退，商业用途需另选具备相应许可的模型。
- **2026-10-24 真机复核**：ComfyUI 0.42.0 / templates 0.11.96 下，产品级
  `GenerateImage` 单图指令编辑成功（1024×1024 RGBA、172,229 种颜色）；Turbo Edit
  所需 UNET/CLIP/VAE 均已就位，不再等待基础 Edit UNET。
- 2.1 Turbo 真机文生图成功（1024×1024 RGBA、274,349 种颜色、49.59 秒）；其本机
  SHA-256 与 HF tree API 的 `lfs.oid` 完全一致，证明 2.1 CLIP/VAE/采样节点栈健康。
- 2511 在 0.42.0 下仍然黑图；同实例 Krea2 对照正常出图（158,876 种颜色），排除 VAE 与
  全局 GPU 故障。本机 2511 文件大小与官方同为 20,533,762,817 B，但 SHA-256 为
  `d21c31e4…d081`，官方 LFS SHA-256 为 `c9fdc158…0a4e`，确认本地权重损坏。
  该结论仅保留为退役依据，不再重新获取 2511。
- 两次 2511 实跑均出现 VAE 解码非有限值转换警告并输出纯黑图；诊断证据已随 2511 退役清理。
- **2511 退役（2026-10-24 用户指令）**：删除 `qwen-image-edit` 的 workflow、权重提示、
  Kontext 缩放辅助与测试；媒体编辑、角色设定卡、原罪 qedit 统一改走
  `qwen-image-2.1-edit`。本地 2.1 Edit 不支持蒙版，局部重绘/扩图明确提示改用云端。
  实际 2511 权重 25 个文件 / 51.31GB 已移入 Windows 回收站（可恢复），其他模型未动。
- golden/模型测试通过；单图真机通过。4 图真机在 5 分钟门控内停在 0/8、未产出，已中断并挂池；
  中文图上文字真机输出绿色噪声、无可读文字，判失败并挂池。不自动下载权重。

**最终门禁**

- Go build、vet、golangci-lint、govulncheck 通过；最终 `go test ./... -count=1` 全绿。
  期间曾有一次 Windows 临时测试进程 `Access is denied` 启动失败，失败包单独复跑通过，
  随后全量复跑无失败。
- 前端 tsc/build/knip、生产依赖审计、lint（0 error）、绑定 drift、契约/E 系列/文档守卫通过；
  vitest 当前全量 **456 文件 / 3968 用例全绿**；2511 退役与 Turbo Edit 改造后 Go 全量复跑通过。
- 未抬产品版本；当前绑定面 `PASS@679`（含并行小说功能增量）。工作树保留执行前改动与本轮全部增量，待用户决定提交方式。

## 8. 状态记录与收口

每批完成后更新本文件顶部状态：

```text
> 状态：B0–B4 DONE / B5 DONE（代码+mock）/ B5 真机验证挂池
```

并在批次末尾记录：

- HEAD
- 改动文件
- 新增/删除行
- 定向测试
- 全量测试数字
- 已知留池
- 是否改变绑定面/契约面

不自动抬产品版本；是否发版由用户另行决定。

## 9. 硬阻断与回报条件

只有以下情况停下向用户回报：

1. 依赖升级需要改写超过 3 个公共接口，无法保持行为兼容。
2. 读边界收紧后，核心办公流程必须访问未登记外部根；需要用户决定是否加入
   `allow_read` 默认值。
3. ComfyUI 真机缺少核心权重或显存不足；代码与 mock 测试继续，真机项挂池。
4. 并发会话改写同一目标文件，无法在不覆盖用户改动的前提下合并。
5. 上游 API 实际矩阵再次自相矛盾；保留已通过测试的版本，不追最新。

## 10. 明确不纳入

- Reasonix 全量 MCP Apps、远程 RBAC、插件市场、持久 bash PTY。
- DSH Agent Team、CUA、插件市场、跨设备。
- DSH token-meter 持久 projection 重构：先 profile，无证据不做。
- Unsloth Studio/MLX/training：modelhub 已退役。
- ComfyUI 云端 Partner/API Nodes。
- MOSS-Image、FLUX 2/3、FIBO 等新模型：Qwen2.1 Turbo Edit 之后另批。
- CAD 交互式矢量查看器：一期 PNG 稳定后再做 mlightcad 1.7.7 PoC。
- so-novel AGPL 规则 JSON 搬运；只同步自有解析器的站点数据与测试。

## 11. 参考上游

| 上游 | 快照/目标 | 采用机制 |
|---|---|---|
| Reasonix | v1.39.10 / main `7aa0342` | TargetAccess、子代理 hooks、附件准入、provider send guard |
| DeepSeek Harness | 0.2.1-rc.2 / `687f282` | plan dismiss 语义；token-meter 仅 profile 后评估 |
| oh-story | v0.9.0 / main `c3fbbbe` | 两级作者记忆、三层去味 |
| ComfyUI | 0.42.0 / templates v0.11.96 | Qwen Image 2.1 Turbo Edit |
| mlightcad | npm 显式 1.7.7 | 二期查看器 PoC |

关键提交：

- Reasonix `11b8ceb38`：TargetAccess 与 preview confinement。
- Reasonix `896faf36d`：子代理 hooks 继承。
- Reasonix `694b096ff` / `6bd9fa74c` / `e045489e4`：附件准入、流式限制、目录句柄。
- Reasonix `1019b2811` / `83a9b0056`：provider typed diagnostics / send guard。
- Reasonix `8d2df779c` / `2467b3d7d`：reasoning-only 拒答与单次恢复。
- DSH `18a4bc4093`：关闭计划审批即结束回合。
- oh-story `79b2970` / `ebc17af`：作者记忆与三层去味。

