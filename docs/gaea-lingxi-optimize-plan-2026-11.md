# gaea 办公板块优化计划（灵犀蒸馏落地，2026-11）

> 依据：`docs/gaea-lingxi-re-distill-2026-11.md`（灵犀事实源）、`docs/gaea-dsh-office-distill-research-2026-11.md`（COM helper 来源）。
> 原则：**不动 COM 实时编辑主线**，只补灵犀证明有效的缺口；每项独立可回滚；不做无法本地化的云依赖。
> 构建纪律：任何改动按 `docs/gaea-build-guide.md` 执行（前端先 `npm.cmd run build`，`CGO_ENABLED=1` + ucrt64，`wails build -s -ldflags "-s -w" -trimpath`，`scripts\smoke.ps1` 冒烟）。

> **落地状态（2026-11-13 P0 批收官）**：P0-1/P0-2/P0-3/P0-4 已全部落地并装机（exe 装机位冒烟 200）。
> 偏差与余项：① P0-2 取 **A 档受控宏**（`office_macro` 白名单→既有 ops），VBA/JSAPI 任意脚本路线弃用；
> ② P0-3 的「前端已捕获选区胶囊」不做——改为 agent 按需调 `office_selection`，无面板捕获态；
> ③ P0-5 steer 已有实锤（`steerQueue`+`MidTurnSteerPrefix`+intake 收尾闸），不立项、不加提示词；
> ④ #12 参数错误三要素已补完（2026-11-13 续批：办公 COM 族 8 工具+通用文件族 10 工具统一「工具名+参数解析失败+json 字段路径」+双守卫测试；非高频 ~19 处留池）；
> ⑤ deliver_result IM 语义仍留 P2 文档预留。P1 阻塞点=Python 运行时决策（等拍板）。

---

## 0. 目标与边界

**目标**：让 gaea 的办公板块在三件事上对齐灵犀的实测水平——① 交付卡片稳定不翻车；② 模型有逃生舱不因缺 op 卡死；③ 用户划词即改。

**不做**：替换 COM 为 JSAPI、引入云文档依赖、复刻 5 套主题、patch wpscli 签名校验。

---

## 1. 总览

| 阶段 | 项 | 价值 | 成本 | 依赖 |
|---|---|---|---|---|
| P0-1 | 交付契约硬化（协议优先 + 提示词三条） | 高（用户天天遇到） | 0.5–1 人日 | 无 |
| P0-2 | `office_script` 逃生舱 | 高 | 1–2 人日 | 无 |
| P0-3 | 选区上下文 `<office_selection>` | 高 | 2–3 人日 | P0-2 可选并行 |
| P1-1 | pdf 技能移植 | 中高 | 2–3 人日 | Python 运行时决策 |
| P1-2 | xlsx 质量门禁（recalc/audit） | 中高 | 1–2 人日 | 同上 |
| P1-3 | pptx 模板复刻流水线 | 中 | 3–5 人日 | 同上 |
| P1-4 | 右侧面板打开本机 WPS（dock → embed） | 中高（体验） | 1–2（dock）/ +3（embed） | Win32 钩子 |
| P2-1 | office 工具 MCP 化 | 中（生态） | 3–4 人日 | P0-2 |
| P2-2 | 技能市场/安装契约 | 低中 | 3–5 人日 | skills 落地 |

**里程碑**：M1 = P0 三项完成（约一周）→ M2 = P1 全部（约两周）→ M3 = P2 择机。

---

## 2. P0（先做，收益/成本比最高）

### P0-1 交付契约硬化

| 任务 | 落点 | 验收 |
|---|---|---|
| 协议优先解析：先认 `file:///`、`local-file:///`、`sandbox:///`，命中即直取路径生成卡片；正则启发式降级兜底 | `frontend/src/gaea/lib/fileLinks.ts` | 新增单测：三种协议、带空格路径、带 `#`/`?`、反引号包裹的链接均能出卡 |
| 提示词补三条约束（链接禁反引号/禁代码块；伪协议只用于用户可见链接；工具参数用真实绝对路径） | `prompts/` 办公系统提示词段 | 对话实测：模型输出链接不再被渲染成代码片段 |
| 预留 IM 交付语义（`deliver_result(ref, 单文件单次, 文字写完再调)`）到工具契约文档 | `docs/` + `internal/gaea/tool/builtin/` 注释 | 文档评审通过；实现留给 P2 |

**为什么先做**：这是用户每次交付都会碰到的路径，改动面最小、可完全用测试兜住。

### P0-2 `office_script` 逃生舱

| 任务 | 落点 | 验收 |
|---|---|---|
| 工具 schema：`{doc, app, lang: vba\|jsapi, code, ui, timeout_ms}` | `internal/app/office_com_tools.go` | schema 通过现有工具注册测试 |
| VBA 路径：受控执行（预置宏库 + 参数，或写临时 `.bas` 后 `Application.Run`），失败明确报「该动作不支持」而非静默 | `internal/office/comhelper/OfficeHelper.cs` 新 `Script` 分支 | 用 ops 覆盖不到的动作（如「全表格斑马纹」）跑通，`office_render` + vision 复核 |
| 审计与门控：代码 sha256 入日志、默认关闭、`ui:true` 需前端确认、沙箱会话才放行 | `internal/app/` + 设置页 | 关闭时调用返回明确拒绝；日志可追踪单次脚本 |

**风险**：VBA 工程信任。Plan：优先「预置宏 + 参数」路线，不行再评估 `VBE` 注入。

### P0-3 选区上下文

| 任务 | 落点 | 验收 |
|---|---|---|
| 读 COM 选区（Word `Selection.Range` / Excel `Selection.Address` / PPT `SelectedShape`）+ 前后各 200 字 | `internal/app/office_com_tools.go` 新 `office_selection` | 单测 + 真机：选中一句返回 `{doc, app, para, start, end, text, before, after}` |
| 注入格式 `<office_selection …>` 与提示词约定「有选区时做最小改动，用 `replace_text` + `expect`」 | `prompts/` | 划词说「改成更正式的说法」，只改选中句，`expect` 命中 |
| 前端：面板焦点/快捷键触发捕获，显示「已捕获选区」胶囊 | `frontend/src/gaea/components/` | 捕获/取消/清空三态可测 |

---

### P0-4 提示词级补齐（零排期，随 P0 一起提交）

来源：`docs/gaea-lingxi-advantages-2026-11.md` 附录「最缺的 5 条」。**全部是文案改动，不动逻辑**。

| 任务 | 落点 | 验收 |
|---|---|---|
| 清单截断通知：技能/连接器目录被截断时，显式告知「已展示 N/Total」并给出补全路径（`skill list`），同时说明 location 在执行域内的差异 | `prompts/` 技能段 | 构造 >50 个技能的目录，模型不会声称「没有该技能」 |
| 参数错误三要素：高频 10 个工具的错误文案统一成「字段路径 + 期望 + 合法值」 | `internal/gaea/tool/builtin/*.go` + `internal/app/office_com_tools.go` | 单测断言错误串含字段路径 |
| 压缩交接单三要素：强制纯文本、保留「进行中 + 下一个具体动作」、校验前序摘要继承 | `internal/gaea/agent/compact*.go` 的摘要模板 | 连续压缩两轮，第二轮不丢第一轮的关键决策 |
| 渲染白名单约束：`office_render` / 文档工具补「路径必须在工作区且为绝对路径」 | 工具 schema 描述 | 越界路径被拒绝且有明确提示 |
| 回复收尾三条：先结果后证据、陈述未完成项、禁模板化结束语 + 结束前自检清单 | `prompts/` 全局段 | 抽 10 条真实回复人工过一遍 |

### P0-5 steer 能力确认（先查后做）

`run/steer` + `agent:steerRun` 是灵犀体验的分水岭（长任务中途补话不重跑）。先确认 `internal/gaea/agent` 是否已有等价物：
- 已有 → 补一条「中途注入后如何向用户确认」的提示词；
- 没有 → 立项，评估在 run 生命周期里加增量输入通道的成本（涉及流式与取消语义，不进 P0）。

## 3. P1（补齐能力缺口）

### P1-1 pdf 技能移植

> **B 批证据更新（2026-11-13，见 re-distill §11/§12）**：灵犀 `python-env` 实际只预装 33 个包（349.9 MB），
> `pypdf`/`pdfplumber`/`python-docx`/`python-pptx`/`formulas`/`requests` **都不在预装里**，由客户端按 `service\requirements.txt` 首启 pip 联网装。
> 故本项决策要点从「装不装 Python」改为「预置白名单 + 断网降级」：`pdf_cli` 17 子命令只依赖 `pypdf`+`pdfplumber`+预装的 `pypdfium2`，
> 体积远小于整套 python-env；`pdf_to_md` 依赖金山云（365.kdocs.cn/api.wps.cn），移植时剔除。

- 范围：`skills/pdf`（17 子命令：info/page-count/extract-text/extract-table/to-md/merge/split/delete-pages/rotate/crop/watermark/compress/encrypt/decrypt/render/extract-images/batch）。
- 决策点：**Python 运行时**——① 内置精简 Python（pypdf+pdfplumber+pypdfium2，约 40 MB）② 复用系统 Python ③ Go 原生重写（pdfcpu/unipdf）。
- 建议：①，因为它同时解锁 P1-2/P1-3；若不做 Python，则 P1-2 用 Go 的 excelize 部分替代、P1-3 降级。
- 验收：对一份扫描件跑 `to-md` 出 Markdown；`render` 出 PNG 与 `office_render` 结果一致性抽查。

### P1-2 xlsx 质量门禁

- 加 `xlsx_recalc` / `xlsx_audit` 工具（Go 侧 excelize 可做公式审计的最小版：扫描错误值 + 循环引用；完整重算走 Python `formulas`）。
- 提示词：把「公式优先、零错误、动态数组默认」三条写进 xlsx 技能段。
- 验收：生成一张含公式的表，注入错误数据能报出 `#DIV/0!` 位置。

### P1-3 pptx 模板复刻流水线

- 抄 `flatten → shell-report → make-skeleton → apply-flat → assemble`，落到 Python 脚本层（python-pptx + lxml）。
- 字体白名单 ∩ 系统字体，空交集回退微软雅黑；仅 16:9。
- 验收：给一份母版 pptx + 一段文案，产出保留母版版式的 5 页 deck，逐页 PNG 无跑版。

### P1-4 右侧面板打开本机 WPS

- 已有探针 `tools/wpsdock`（list/dock/embed/restore，已实测本机 `XLMAIN` 窗口 dpi=96）。
- 第一步 dock：`internal/office/wpshost/` + `bindings_office.go` 三个绑定 + `frontend/src/gaea/components/OfficePanel.tsx`（ResizeObserver 上报矩形）。
- 第二步 embed（`SetParent`）默认关闭，开关 `office.panel.embed`。
- 验收：面板折叠/切换 tab/窗口缩放时窗口跟随；关闭面板 restore 后 WPS 恢复正常顶层窗口。

---

## 4. P2（生态与结构）

- **P2-1 office 工具 MCP 化**：把 `office_*` 暴露成 MCP server（stdio），让外部 agent（含 dsh）复用；参考灵犀「wpscli 即 MCP server」的形态。
- **P2-2 技能市场/安装契约**：`installFromUrl` / `packSkillZip` / `setEnabled` 三个动作 + SKILL.md 校验（frontmatter name/description、脚本可执行、无绝对路径）。

---

## 5. 验收与测试策略

| 层 | 手段 |
|---|---|
| 单测 | fileLinks 协议解析、office_selection 结构、xlsx audit 错误值扫描 |
| 真机 | 一份 docx / xlsx / pptx 各跑「生成 → 编辑 → 渲染 → vision 复核」全链 |
| 回归 | 现有 office 测试套（`internal/app/office_*_test.go`）不得红；新增工具默认关闭，不影响旧路径 |
| 冒烟 | `scripts\smoke.ps1 -ExePath build\bin\gaea.exe`，`/api/health` 200 |

---

## 6. 工期（人日，单人）

| 阶段 | 乐观 | 悲观 |
|---|---|---|
| P0 合计 | 4 | 6 |
| P1 合计 | 7 | 12 |
| P2 合计 | 6 | 9 |

阻塞点：Python 运行时决策（影响 P1 全部）；VBA 信任策略（影响 P0-2 路线）。

---

## 7. 风险与回滚

| 风险 | 应对 | 回滚 |
|---|---|---|
| `office_script` 被滥用 | 默认关 + 沙箱会话放行 + 审计日志 | 开关置 false，工具返回「未启用」 |
| 选区误改（改了不该改的） | `expect` 校验 + 只允许选区范围内 `replace_text` | 文档级 undo（COM `Document.Undo`） |
| 面板停靠导致 WPS 异常 | 先 dock（不改从属关系） | `wpsdock restore` 一键恢复样式 |
| Python 运行时体积/杀软误报 | 精简依赖、不进主安装包（首次按需下载） | 不启用 Python 时工具集降级为 Go 原生 |
| 移植 skills 带来云依赖 | 逐脚本标注外部依赖 | 单技能开关关闭 |

---

## 8. 文档联动

- 本文落地后，更新 `docs/gaea-build-guide.md`（若新增构建产物）与 `.gaea/AGENTS.md` 构建入口，禁止两处漂移。
- 与 `docs/gaea-dsh-office-distill-research-2026-11.md` 的关系：那份定义 COM helper 的来源与坑册，本份定义「围绕它补什么」，不重复。
