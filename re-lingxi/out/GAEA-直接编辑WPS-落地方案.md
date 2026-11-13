# gaea「直接编辑 WPS」落地方案（基于灵犀逆向）

> 前提事实（见 `REPORT-灵犀逆向分析.md` 附录 E、`lingxi-appserver-re-report.md`）：
> 灵犀不是用 COM，而是 `wpscli`(Rust MCP) → `\\.\pipe\lingxi-wps-broker-*` → WPS **JSAPI 引擎**；且每个脚本类工具都有 `ui:true` 能开 WPS 窗口。
> gaea 现状：`office_com_tools.go` + `OfficeHelper.cs`（8600+ 行）走 COM，op 面含 word 21 / excel 24 / ppt 19 类动作，**比灵犀的 20 个语义 action 更宽**。

## 0. 结论修正（重要）

上一轮的判断「抄灵犀的 action 词表」**不成立**——反向核对 `OfficeHelper.cs` 的 `type == "..."` 分发后可见 gaea 覆盖面更广：

| 域 | gaea 已有 op（节选） |
|---|---|
| Word | `replace_text` `insert_paragraphs` `set_text` `format_text` `set_style` `delete_range` `insert_table` `set_cell` `insert_image` `insert_diagram/insert_flowchart` `insert_references` `style_format` `page_setup` `page_numbers` `insert_toc` `page_break` `header` `fill_gaps` `update_fields` `format_table` `set_image` `footnote_settings/insert_footnote/edit_footnote/delete_footnote` |
| Excel | `add_sheet` `write_range` `format_range` `autofit` `rename_sheet` `delete_sheet` `insert_rows/delete_rows` `sort` `filter/clear_filter` `remove_duplicates` `replace` `clean` `fill_blanks` `fill_formula` `add_column` `insert_columns/delete_columns` `copy_range` `pivot` `table` `conditional_format` `freeze` `validation` `add_chart/set_chart/delete_chart` |
| PPT | `add_slide` `theme` `slide` `reuse_slide` `delete_slides/delete_slide` `move_slide` `set_notes` `add_textbox` `add_image` `duplicate_slide` `set_layout` `set_background` `add_shape` `add_table` `add_smartart` `format_text` `set_text` `set_shape` `delete_shape` |

真正缺的只有三样：**逃生舱**、**选区上下文**、**进程/权限分层**。

---

## 1. 三条可选路线

| 路线 | 机制 | 优点 | 代价 | 建议 |
|---|---|---|---|---|
| A 扩 COM（现状） | `KWPS.Application` / `Ket.Application` / `KWPP.Application` IDispatch | 稳定、跨版本、无需授权；op 面已足够 | 进程外调用，拿不到 WPS 新特性；无「任意脚本」能力 | 主线，继续 |
| B 自研 WPS 加载项 | `%APPDATA%\kingsoft\wps\jsaddons` 放加载项（JS + `publish.xml`/`jsplugins.xml`），加载项内 WebSocket 连 gaea | **与灵犀等价的 JSAPI 能力**；官方支持的第三方开发方式；进程内执行、性能好 | 需用户安装加载项；加载项在 WPS 进程内，权限=用户权限 | P1 增强 |
| C 复用 wpscli | 直接调灵犀的 wpscli MCP | 零开发 | `parent_verify` + `WinVerifyTrust` 签名校验 + 沙箱账户 broker，产品化不可行 | 仅实验室 |
| D WPS 加载项 RPC SDK | `wpsjs-rpc-sdk` 走通「外部进程 ↔ 加载项」RPC | 有现成 SDK 形态 | 仍需加载项落地 | B 的实现子选项 |

参考：[WPS 加载项集成业务系统开发](http://open.wps.cn/documents/app-integration-dev/wps365/client/wpsoffice/wps-integration-mode/wps-addin-development/wps-addin-integration-business-system-development)、[wpsjs-rpc-sdk](https://npmmirror.com/package/wpsjs-rpc-sdk-new)。

---

## 2. P0：`office_script` 逃生舱（最高性价比）

**目标**：当 21+ 个高层 op 覆盖不到时，允许模型执行一段受控脚本，而不是报「不支持」。

**落点**：`internal/app/office_com_tools.go` 新增工具 `office_script`，`OfficeHelper.cs` 新增 `Script` 分支。

```json
{
  "name": "office_script",
  "properties": {
    "doc":   {"type": "string", "description": "文档路径或窗口名，缺省=最近一次 open/read/edit 的文档"},
    "app":   {"type": "string", "enum": ["word","excel","ppt"]},
    "lang":  {"type": "string", "enum": ["vba","jsapi"], "default": "vba"},
    "code":  {"type": "string", "description": "脚本正文；vba 模式为宏体，jsapi 模式需 WPS 加载项通道"},
    "ui":    {"type": "boolean", "default": false},
    "timeout_ms": {"type": "integer", "default": 15000}
  },
  "required": ["code"]
}
```

实现要点（按优先级）：
1. **vba 路径**（今天就能做）：COM 侧 `Application.Run` 无法直接跑匿名代码，需两步——把代码写入临时 `.bas`/`.vbs`，用 `Application.VBE`（WPS 需信任访问 VBA 工程）或走 `WScript` 注入；更稳的是在 `OfficeHelper.cs` 里 `doc.VBProject` 校验失败时回退到「预置宏库 + 参数」模式。
2. **jsapi 路径**：等 P1 加载项通道就绪后接入。
3. **安全边界**：只在 `sandbox` 会话放行（`LINGXI_SANDBOX_POLICY_MODE` 的等价物），默认关闭；`ui:true` 需前端显式确认；脚本执行写审计日志（代码哈希 + 结果摘要）。

**验收**：给一份 docx，让模型用 `office_script` 完成一个 ops 覆盖不到的动作（如「给所有表格加斑马纹」），`office_render` 出图后 vision 复核通过。

---

## 3. P0：选区上下文注入（「划词即改」）

**目标**：用户在 WPS 里选中一段文字/一个单元格，gaea 能拿到「哪份文档、哪一段、选了什么、上下文前后 N 字」，模型不必自己找位置。

**落点**：
- 后端：`internal/app/` 新增 `office_selection` handler，读 COM `Selection`（`Range.Start/End/Text`、`Word.Selection.Range`、`Excel.Selection.Address`），补 `前后文各 200 字`。
- 前端：选区变化时（`shell` 拦截或用户快捷键）注入 `<office_selection>` 结构体，形如灵犀的 `<wps_docs_selection>`：
  ```xml
  <office_selection doc="报告.docx" app="word" page="3" para="12" start="1450" end="1520">
    <text>选中的原文</text>
    <before>上文…</before><after>下文…</after>
  </office_selection>
  ```
- 提示词：在系统提示里约定「有 `<office_selection>` 时优先做最小改动，用 `replace_text` + `expect` 校验」。

**验收**：选中一句话说「改成更正式的说法」，模型只动这一句，`expect` 校验命中，`office_render` 可见改动。

---

## 4. P1：WPS 加载项 JSAPI 通道（等价 wpscli）

**目标**：拿到与灵犀同级的「WPS 引擎内 JSAPI」能力，不依赖金山签名授权。

```
gaea(Wails/Go)  ──WS(localhost:随机端口 + token)──  gaea-wps-addin (JS, 跑在 WPS 内)
                                                      │  wps.WpsApplication().ActiveDocument
                                                      ▼
                                                   WPS 引擎
```

- 加载项骨架：`jsaddons/gaea-wps-addin/{publish.xml, index.html, main.js, ribbon.xml}`；`main.js` 启动即连 `ws://127.0.0.1:<port>?t=<token>`，收到 `{id, code}` 后 `eval` 并回 `{id, ok, result}`。
- 端口/token 由 gaea 启动时生成，写 `%USERPROFILE%\.gaea\wps-addin.port`（仿灵犀 `serverdir/nativehost.port`）。
- 能力：`wps.Application.ActiveDocument` 全 API + `Application.Run` JSAPI；gaea 侧暴露成 `office_script(lang:"jsapi")`。
- 与灵犀的差别：灵犀是原生 exe 直连，我们是加载项桥；对外表现一致（都能开 `ui`、都能跑脚本）。

**风险**：WPS 加载项需用户首次启用（WPS 会提示第三方加载项）；跨版本 API 差异需探测 `wps.EtApplication()` 等入口。

---

## 5. P2：broker 分层（可选）

灵犀 `app-server → wpscli-proxy(命名管道 + LINGXI_WPS_BROKER_TOKEN) → wpscli` 的分层值得抄，但要等我们有多 worker/沙箱需求时再做：

- 进程：`gaea-wps-broker.exe`（Go），监听 `\\.\pipe\gaea-wps-broker-<pid>-<rand>`，校验 token 后转发到 COM/加载项通道。
- 好处：技能脚本（未来 Python 侧）不必各自持 COM，连接可复用、可审计、可限流。
- 不建议现在做：当前 gaea 的 COM 调用在同一进程内，加一层只会增加复杂度。

---

## 6. 实施顺序与验收

| 阶段 | 内容 | 验收标准 |
|---|---|---|
| P0-1 | `office_script`（vba 路径 + 审计 + 沙箱门控） | 完成一个 ops 覆盖不到的编辑，render+vision 通过 |
| P0-2 | 选区上下文 + 提示词约定 | 划词指令只改选中内容，`expect` 校验命中 |
| P1 | 加载项 JSAPI 通道 + `office_script(lang:"jsapi")` | 同一脚本在加载项里跑通，且能 `ui` 展示过程 |
| P2 | broker 分层 | 多会话并发编辑不互相阻塞 |

## 7. 不做的事

- 不 patch `wpscli` 的 `parent_verify`（签名校验是金山的产品边界，且版本一升级就失效）。
- 不照搬灵犀的 20 个 action 词表（gaea 已有更宽的 op 面）。
- 不引入 AppContainer（灵犀自己也没用，用的是 restricted token + Job Object + 私有桌面）。
