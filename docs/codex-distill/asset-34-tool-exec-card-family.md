# 资产 34 · 工具/执行卡片族：一张卡的七种投影 + 三层输出预算

> 证据根（只读）：`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs\tui\src\`
> 本篇用到 `history_cell/{mod,activity_preview,base}.rs`、`exec_cell/{model,render,compact,transcript,live_output}.rs`。
> 对照根（只读）：`frontend/src/gaea/{components/ToolCard.tsx,components/ToolGroup.tsx,components/Transcript.tsx,lib/tools.ts}`
> 与 `frontend/src/wenxue/components/{ToolCard.tsx}`（第二份拷贝）。
>
> 本篇只出**机制 + 落点 + 验收口径**，不改 gaea 任何源码。

---

## 一、`HistoryCell` 不是「一个渲染函数」，是七种投影的契约

`history_cell/mod.rs:199-354` 的 trait 把「一张卡」拆成**同一份数据的多种表述**，
每个投影各有明确用途，互不代用：

| 投影 | 签名 | 用途 | 证据 |
|---|---|---|---|
| `display_lines(width)` | 主视口富文本 | 聊天主流 | `mod.rs:222` |
| `raw_lines()` | 纯文本逻辑行 | 复制/回滚模式 | `mod.rs:225` |
| `live_raw_lines()` | 可省掉仅转录期诊断 | 流式中的原始态 | `mod.rs:233` |
| `display_hyperlink_lines(width)` | 富文本 + 链接元数据 | 主视口（可点） | `mod.rs:238` |
| `compact_hyperlink_lines(width)` | **紧凑**表述 | 独占转录/折叠态 | `mod.rs:244` |
| `retained_hyperlink_lines(w, detailed)` | 紧凑/详细二选一 | 可重绘的历史区 | `mod.rs:249-255` |
| `expanded_hyperlink_lines(width)` | **展开**表述 | 就地展开 | `mod.rs:264-266` |

外加四条**旁路**，回答的不是"长什么样"而是"别的东西怎么拿"：
`copy_source()` 语义复制源（`mod.rs:217`，**Markdown 原文，与呈现无关**）、
`image_path()` 可预览的本地产物（`mod.rs:228`）、
`warning_entries()/warning_keys()` 诊断原文与稳定身份（`mod.rs:207-214`，注释点明 keys「独立于换行与重复投递」）、
`activity_ids()` 折叠身份（`mod.rs:257-261`，「namespaced 成员身份，跨分组与重放保留」）。

两个**动态性**钩子：`transcript_animation_tick()`（`mod.rs:341-353`，返回 `None` 表示稳定，`Some(tick)` 表示有时基动画必须重算——注释直说「若动画卡恒返回 None，`Ctrl+T` 会冻在第一帧」）；
`has_stable_transcript_height()`（`mod.rs:329-335`，外态驱动的卡返回 `false`，共享视口就每帧重排）。
`is_stream_continuation()`（`mod.rs:337`）让「同一段正文的第二块」能接排。

> **一句话**：codex 的卡片契约里，"渲染"只是七分之一；剩下六项分别服务复制、展开、身份、动画与高度。
> gaea 的 `ToolCard` 把"渲染 + 展开 + 复制（靠 DOM 选区）"混在一个组件里，**没有 copy_source 这一层**——
> 于是复制出来的永远是"被 `…已折叠 N 行` 污染过的呈现文本"。

## 二、高度 = 实际换行后的行数，不是逻辑行数

`mod.rs:292-309`：默认 `desired_height` 用 `Paragraph::line_count(width)`（`Wrap { trim: false }`）测，
注释把理由写死：**逻辑行数会低估**——「含超宽 URL 的行比终端还宽」。具体卡只在"超出 Paragraph 测得的布局"时才覆写。

配套 `mod.rs:356-372`：`Box<dyn HistoryCell>` 的 render **先 `Clear.render(area)` 再画**，
理由写在注释里：活动卡在 resize/流式更新时会剧烈重排，"上一帧的字形不许残留"。

> 对 Web 的映射：这不是"估算高度"，而是**用与渲染同参的度量再测一次**。
> gaea 的 `content-visibility` + 尾部窗口（`components/chat/MessageList.tsx`）用的是"跳过渲染换性能"，
> 两者冲突点就在这里——见 §八、§九。

## 三、执行卡的归组是**语义**，不是同名

`exec_cell/model.rs:233-244`：

```
is_exploring_call(call) = !UserShell && !parsed.is_empty() && parsed.iter().all(Read|ListFiles|Search)
```

- `add_call`（`model.rs:92-116`）：只有「本卡已是探索组」且「新 call 也是探索类」才并入，否则返回 `false` 让上层另起一卡。
- `append_completed`（`model.rs:119-130`）：历史合并走同一判据，且 `should_flush()` 或对方非探索时**拒绝合并**（返回 `Err(自身)`）。
- `should_flush`（`model.rs:163-166`）：探索组**永不满**（"相邻调用即使读失败也继续留着"`model.rs:164` 注释）；非探索组在全部 call 都拿到 `duration` 后才可冲掉。
- `append_reasoning`（`mod.rs:202-204` + `render.rs:213-216`）：**推理片段被收进探索组**（`group.push_detail`），
  注释说明「call 位置固定不因在飞调用完成而变」，这样紧凑态不会被推理撑开，展开态又保留原位。

> gaea 的对比物是 `ToolGroup.tsx:86-115` `scanGroups`：**同名连续 ≥2 才折叠**，`skipRunning` 时运行中不折。
> 差别不是"谁好看"，而是**归组的判据不同**：同名是"字面重复"，语义是"这段在读代码"。
> 一次任务里 `read_file → grep → ls` 三个不同名调用，codex 合成一张"探索卡"，gaea 是三个独立卡。

## 四、`call_id` 路由失败是**信号**，不是异常

`model.rs:137-161` `complete_call`：按 `call_id` **反向**找最近的匹配 call，找不到返回 `false`。
注释把这个返回值定成契约（`model.rs:138-141`）：

> 「调用方应把 `false` 当**路由失配**而不是静默忽略。widget 用它避免把孤儿 `exec_end` 挂到无关的活动探索卡上——
> 那会把两条转录项错误地collapse 成一条。」

- `mark_failed`（`model.rs:168-182`）：卡被判定失败时，给**所有未完成**的 call 补 `duration=elapsed` 与 `exit_code=1`——
  即"没结果"与"失败"在渲染层不区分，界面不给假绿。
- `append_output`（`model.rs:212-231`）：空 chunk 直接 `false`，不建 live buffer。

## 五、live 输出的三层预算：字节 → 行 → 单行头尾

`exec_cell/live_output.rs:5-11`：

| 常量 | 值 | 含义 |
|---|---|---|
| `MAX_BYTES` | `1 MiB` | 总量闸门 |
| `MAX_LINES` | `50` | 头/尾各保留 50 行 |
| `MAX_LINE_BYTES` | `MAX_BYTES/(2*50+2)` = 10,280 B | 单行预算（分母的 `+2` 是给"未完成行"留位） |
| `HEAD` / `TAIL` | 5,140 / 5,140 B | 单行再对半切头尾 |

机制链（`live_output.rs:32-97`）：
1. 未超预算 → 全留（`full_output`），同时累计 `completed_lines` 与 `has_partial_line`；
2. 一旦超限 → **把已积的全量重放进有界模式**（`push_truncated_str(&full_output)`），此后只走有界路径；
3. 完成行：头 50 行进 `head`，之后进 `tail`（`VecDeque`，满 50 就 `pop_front`）；
4. 单行超 `MAX_LINE_BYTES` → `LiveCommandOutputLine` 只留头 5KB + 尾 5KB，中间记 `omitted_bytes`（`live_output.rs:180-223`，切点全部回退到 `is_char_boundary`）；
5. 渲染省略标记前**先关掉可能悬空的 ANSI 序列**（`live_output.rs:238-251`：OSC 补 `\x07`、CSI 补 `m`、再补 `\x1b[0m`），否则"半个颜色序列 + 省略标记"会把后续整屏染色。

两个计数必须分开（`live_output.rs:99-113`）：`total_lines()`（真实总数，含未完成行）与 `retained_lines()`（实际渲染行数）。
`lines()`（`116-139`）给**预览**（有损但尾向可迭代），`transcript_lines()`（`142-162`）给**转录**（未截断时无损，截断时在头尾之间插 `… +N lines`）。

> 这就是"诚实截断"的最小实现：**总数在、保留数在、差多少说清楚**。

## 六、披露是**量化**的：`ActivityDisclosure`

`history_cell/activity_preview.rs:7` `DETAIL_PREVIEW_LINES = 3`；
`activity_preview.rs:9-15` `ActivityDisclosure = Generic | OutputLines(n)`（"保留输出中未完全可见的行数，**不含存储层省略**"）；
`activity_preview.rs:17-19` `clipped_line` 注释：**剪裁后的行不许被选择复制**（"别教会选区去复制当前隐藏的文本"）。

`exec_cell/compact.rs:20-66` 的算法值得逐句抄：

```
retained   = output.line_counts().1                       // 保留行
clipped    = 最近 3 行里「ANSI 宽度 + 4 > 卡宽」的条数      // 宽度溢出的行
hidden     = max(retained - 3, 0) + clipped
if hidden > 0 -> OutputLines(hidden)
else if 组内有额外 details -> Generic
else if 命令本身多行或首行超宽 -> Generic
else if 输出超 3 行或有超宽行 -> Generic
else None                                                  // 什么都不给
```

> **判据是"确实藏着东西"才给展开入口**，而不是"这个工具类型通常能展开"。
> 第六档返回 `None` 这一条最容易被忽略：gaea 里 `expandable` 是**按工具能力**算的
> （`ToolCard.tsx:186`：有 command / 有 diff / 有 args / 有 output 就可展开），
> 于是"一条 `Ran ls` + 0 行输出"也会亮出可展开箭头，点开只有空块。

## 七、状态词与符号由状态派生，不手写

`compact.rs:80-101`：`marker` 与 `title` 都是**四态派生**：

| 状态 | 符号 | 标题 |
|---|---|---|
| 非零退出 | `•` 粗 | `Exited (status {code})` |
| 在飞 | `activity_indicator(start_time, motion, StaticBullet)` | `Running` |
| 统一执行交互 + 已结束 | `•` 绿粗 | `Background output` |
| 其它已完成 | `•` 绿粗 | `Ran` |

`transcript.rs:42-56`：展开态每 call 尾部再给一行结果行——`✓` 绿粗 / `✗ 红 + (code)`，有 duration 时追加 `• 1.2s` 暗色。
另有两处细节：命令前缀 `$ ` 用洋红、续行缩进 4 空格（`transcript.rs:29-36`）；
统一执行交互的输出**故意不进展开转录**（`compact.rs:24-25` 注释："Interaction output is intentionally absent"）。

## 八、gaea 现状核对（只读实测）

| 位置 | 现状 | 与 codex 的差 |
|---|---|---|
| `gaea/lib/tools.ts:449-482` `boundedOutput` | 预览 = **头部 N 行**（`lines.slice(0, maxPreviewLines)`），尾部只追加一行 `… 已折叠 N 行` | codex 是**头 + 尾**双向保留：一条 200 行的构建日志，关键信息在尾部（错误），gaea 恰好丢掉 |
| `gaea/lib/tools.ts:447` `TOOL_OUTPUT_LINE_CAP=160` | 按**字符数**截断单行 | codex 按**显示宽度**（`ansi_escape_line().width()`）算，CJK/emoji 差一倍；且 gaea 的 `lineFolded`（`:461`）**没进 `hiddenLines`**，折叠计数少报 |
| `gaea/lib/tools.ts:424/442` | 通用 60 行、卡片 12 行两档 | 数值可留（Web 侧按实测重定，见 §十一），缺的是"总量/保留量/省略量"三件同时可读 |
| `gaea/components/ToolCard.tsx:186` `expandable` | 按**能力**算（command/diff/args/output 任一） | codex 按**实际隐藏内容**算（§六），空输出也会亮箭头 |
| `ToolCard.tsx:364-377` | 「展开全部」直接把 `bounded.full` 塞进 `<pre>` | 十万行输出 = 十万行 DOM；codex 的转录**无损但不进主视口**（主视口恒 5 行，`render.rs:43` `TOOL_CALL_MAX_LINES=5`） |
| `ToolCard.tsx:363` `tool.outputHeader {n}` | 抬头写"输出 n 行"，n 取自 `item.output.split("\n").length`（`:199`） | 与 `bounded.totalLines` 同值但**各自算一遍**，两处口径迟早分叉 |
| `ToolGroup.tsx:86-115` `scanGroups` | 同名连续折叠 | 见 §三：语义探索组不存在 |
| `Transcript.tsx:282,641,723` + `SubagentThread.tsx:503` | 同一 `ToolCard` 在四处消费，`subcalls` 递归 | 与 codex 的"卡契约 + 多层投影"结构差一层抽象，但消费面已经收口到一处组件，改造成本可控 |
| `gaea/components/ToolCard.tsx` vs `wenxue/components/ToolCard.tsx` | **两份拷贝**（19.6KB / 18.4KB），已开始漂移 | 本批任何卡片改动必须单源化，否则改一份、另一份静默退化 |

## 九、三档判定

**档一（照搬语义，值得动）**

1. **披露量化**：`ActivityDisclosure::OutputLines(hidden)` → gaea 的 `hiddenLines` 必须**含宽度裁掉的条数**；
   `hidden == 0` 时不给展开入口（消灭"点开是空块"）。
2. **诚实省略标记**：`… +N lines` 之外要有"去哪儿看"的指引（codex 的 `transcript_hint()`，`render.rs:309-315`）。
3. **状态词派生**：非零退出必须是 `Exited (status N)` 而不是通用的"失败"；运行中/后台输出/正常结束三态分开。
4. **路由失配不吞**：gaea 的 tool_result 按 `toolCallId` 配对（`SubagentThread.tsx:34,83`），
   配不上时应留**孤儿卡**并标 `orphan`，不要挂到上一张同名卡上。
5. **组语义**：`scanGroups` 增加"探索组"判据（read_file/ls/glob/grep 相邻且全非 running 即可归组），
   推理项保留组内原位（`append_reasoning` 的等价物）。

**档二（改造搬）**

6. **三层预算**：gaea 的 live 输出（SSE 累积）目前只有"到头截断"。应改为 head+tail + 单行头尾 + `omittedBytes`，
   且 `totalLines`/`retainedLines` 分开暴露给 UI（不是 UI 再 `split("\n").length` 一遍）。
7. **显示宽度裁断**：`TOOL_OUTPUT_LINE_CAP` 从字符数改成 `Intl.Segmenter` 分段 + East-Asian 宽度累计，
   或直接交给 CSS（`text-overflow` + `ch` 单位）——但**折叠计数**仍需宽度口径，否则数字是假的。

**档三（不搬）**

8. `Clear.render` 重绘、`HyperlinkLine` 元数据、转录缓存键 `active_cell_cache_key`（终端/帧模型专属）；
9. `LIVE_COMMAND_OUTPUT_MAX_LINE_BYTES=10280` 这类数值（1MiB/102 是终端常量，Web 侧按实测重定）。

## 十、施工卡（拍板后另开批次）

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F34-A 披露量化** | `gaea/lib/tools.ts`（`BoundedOutput` 增 `hiddenWidthClipped`）→ `ToolCard.tsx:186,365-377` | `expandable = hidden > 0 \|\| command \|\| diffs.length > 0`；`hiddenLines = total - preview + widthClipped` | 单测：0 行输出 ⇒ 无展开；12 行内 ⇒ 无展开；200 行 ⇒ `hiddenLines=188`；含 3 条超宽行 ⇒ `+3` |
| **F34-B head+tail 预览** | `boundedOutput` 内部分支 | `maxPreviewLines` 拆成 head 6 / tail 6（或 4/8，按观感定），中间插省略行 | 夹具：200 行构建日志，折叠态**同时**看到第 1 行与最后一行 |
| **F34-C 三层预算** | `gaea/lib/tools.ts` 新 `liveOutputBuffer`（纯函数）+ `gaea/lib/store` 累积点 | 字节闸门→行闸门→单行头尾；`totalLines/retainedLines` 上抛 | 单测：单行 1MB base64 ⇒ 卡片高度恒定；`retained < total` 时省略行数字等于差额 |
| **F34-D 探索组** | `ToolGroup.tsx:86-115` `scanGroups` | 增加 `isExploring(name)` 集合 + 相邻归组（全非 running） | 单测：`read_file,grep,ls` 合成一组；其中一个是 running ⇒ 不组；`bash,bash` ⇒ 仍走同名组 |
| **F34-E 单源化** | `wenxue/components/ToolCard.tsx` | 改为 `export { ToolCard } from "../../gaea/components/ToolCard"`（或反向），差异部分抽 props | 两壳渲染同一夹具输出逐像素一致；knip 无死导出 |

**开关**：F34-A/B/C/D 全部走"纯函数先行"，UI 层用 `FEATURE_TOOL_CARD_V2` 单闸；关闸时 `boundedOutput` 行为与现状逐字节等价。
**回退**：五项均可在 `lib/tools.ts` 内以旧实现并存（`boundedOutputLegacy`），无需回滚组件。

## 十一、风险

1. **流式路径**：`ToolCard` 在 `Transcript.tsx` 的尾部窗口内，卡高变化会牵动滚动锚定（本批 F28 的坐标系问题）。
   卡片改预算前先确认 `useFollowScroll` 的重算触发点是"内容变化"而非"高度变化"。
2. **`content-visibility` 与展开态冲突**：展开一张位于 `content-visibility:auto` 容器内的卡，浏览器可能不重算尺寸——
   验收必须包含"展开第 50 张卡后向上滚动无跳变"。
3. **不要照搬 5 行/3 行常量**：`TOOL_CALL_MAX_LINES=5`、`DETAIL_PREVIEW_LINES=3` 是终端行数；
   Web 卡片有内边距与字号差异，按"折叠态可读三段"重定，数值写进常量而不是散落。
4. **两份拷贝**：任何一项落地前先做 F34-E，否则改动会分叉。

## 十二、证据自查

- 七投影契约：`history_cell/mod.rs:199-354`（trait 定义处逐行可对）。
- 高度度量：`mod.rs:292-309`；重绘清理：`mod.rs:356-372`。
- 归组：`exec_cell/model.rs:92-130,163-166,233-244`；推理入组：`mod.rs:202-204`、`render.rs:213-216`。
- 路由失配：`model.rs:137-161`（含注释原文）；失败补齐：`model.rs:168-182`。
- 三层预算：`live_output.rs:5-11,32-97,99-162,180-255`。
- 披露量化：`activity_preview.rs:7-19`、`compact.rs:20-66`；宽度裁断口径：`compact.rs:33,62`。
- 状态派生：`compact.rs:80-101`；结果行：`transcript.rs:42-56`；主视口行数：`render.rs:43-45`。
- gaea 侧：`lib/tools.ts:424-482`、`ToolCard.tsx:186-199,341-384`、`ToolGroup.tsx:86-115`、`Transcript.tsx:282,641,723`。
