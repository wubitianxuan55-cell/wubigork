# gaea t4-C3 余项 · partial 选段局部重写 · 实施规格（v4.321.0）

> 来源：todos 小说线「C3 余项=partial 局部重写」；规格依据 docs/distill/04-plot-analysis.md
> §5.3（MuMu 路径 B 蒸馏：±50 模糊重锚/±500 上下文/长度模式四档/max_tokens 公式）。
> **gaea 升级项**（相对 MuMu）：partial 也走 v4.304 版本库（全文快照可回滚——
> MuMu 局部重写无快照无 undo，F5 缺陷）；rune 口径（MuMu 字符=Python len）。
> 三线并行（A 引擎纯函数 / B handler 接线 / C 前端），主代理收口。
> **零新绑定**：NovelChapterRewrite reqJSON 已带 mode（types.RewriteRequest
> StartPos/EndPos/LengthMode/TargetWordCount 契约字段已先行），绑定面 696 不动。

## 0. 论点

整章重写对「改一段」太重：全章重新生成会波及未选段落。局部重写=选区 +
指令 + 长度模式，只重写选中片段，前后文原样拼接；版本库照常留全文快照
（回滚=整章恢复，与 whole 一致）。

## 1. 线 A · 引擎纯函数（internal/rewrite/partial.go 新文件 + partial_test.go）

```go
// 长度模式常量
const (
    LengthModeSimilar  = "similar"  // 缺省
    LengthModeExpand   = "expand"
    LengthModeCondense = "condense"
    LengthModeCustom   = "custom"
)

// ResolveSelection rune 口径选段校验与 ±50 模糊重锚（MuMu L4959-4986 rune 版）。
// start/end 为 rune 偏移；selected 空 = 跳过重锚只做边界校验。
// 错误文案：起始位置超出内容范围 / 结束位置超出内容范围 / 起始位置必须小于结束位置 /
// 选中的文本与章节内容不匹配，请刷新后重试。
func ResolveSelection(content string, start, end int, selected string) (int, int, error)

type PartialLengthSpec struct { MinRunes, MaxRunes int; Hint string }
// 四档（MuMu L5054-5071）：similar 0.8×~1.2× / expand 1.2×~2.0× / condense 0.5×~0.8×
// / custom target±20%（target 钳 [50,10000]）；未知 mode → similar。
// Hint：「保持与原文相近的字数（约 N 字，允许 min-max 字浮动）」等对应措辞。
func PartialLengthSpec(mode string, selRunes, targetWords int) PartialLengthSpec

// MuMu L5100：max(500, min(MaxRunes*3, 8000))。
func PartialMaxTokens(spec PartialLengthSpec) int

// BuildPartialInstruction 局部重写指令，四段固定顺序：
// ①上下文（ctxBefore/ctxAfter 为调用方截好的 ±500 rune 片段；空→「（这是章节
//   开头/结尾）」；两段都注明「仅参考，不得改动」）
// ②选中文本（「只重写以下【选段】，选段外一字不动」）
// ③用户指令（trim 非空；>1000 rune 截断）
// ④长度模式提示（spec.Hint）
func BuildPartialInstruction(instr, selectedText, ctxBefore, ctxAfter string, spec PartialLengthSpec) string
```

NormalizeRequest（engine.go）加 partial 分支：Mode=partial 时——Source 必须为
custom 或空（空归 custom；其余报「局部重写仅支持自定义指令」）；CustomInstructions
trim 空 → 报「重写指令不能为空」；LengthMode 空 → similar；custom 且
TargetWordCount<=0 → 报「自定义长度需提供目标字数」；StartPos<0 或
EndPos<=StartPos → 报「选区非法」（内容长度校验在 handler）。

**A 测试**（表驱动）：ResolveSelection 全分支 / PartialLengthSpec 四档数学+未知回
similar / PartialMaxTokens 下限 500 上限 8000 / BuildPartialInstruction 四段序与
空上下文文案 / NormalizeRequest partial 分支六断言。

## 2. 线 B · handler（internal/app/novel_rewrite_handler.go + _test）

NovelChapterRewrite 在 NormalizeRequest 后加 `req.Mode == types.RewriteModePartial`
分支（整章路径零变化）：

1. 建议读取段跳过（partial 恒 custom）。
2. original=ReadChapter 后：`runes := []rune(original)`；`start,end,err :=
   rewrite.ResolveSelection(original, req.StartPos, req.EndPos, req.SelectedText)`
   ——**注意**：RewriteRequest 需加 `SelectedText string json:"selected_text,omitempty"`
   （types/plot_v2.go，线 B 足迹内）；selected := string(runes[start:end])；
   ctxBefore := string(runes[max(0,start-500):start])；ctxAfter :=
   string(runes[end:min(len(runes),end+500)])。
3. spec := rewrite.PartialLengthSpec(req.LengthMode, len([]rune(selected)),
   req.TargetWordCount)；instruction := rewrite.BuildPartialInstruction(
   req.CustomInstructions, selected, ctxBefore, ctxAfter, spec)。
4. 同一 rewrite-chapter 模板：systemPrompt = substituteWordCount(..., spec.MaxRunes)；
   userPrompt 的 chapter_content = **selected（只给选段）**，prev_summary 同 whole；
   ChatSimpleOptions.MaxTokens = rewrite.PartialMaxTokens(spec)（whole 固定 8192
   不动，仅本分支）。
5. newSelected := CleanRewriteOutput(raw)；空报错不落盘；
   newFull := string(runes[:start]) + newSelected + string(runes[end:])；
   diff := ComputeDiff(selected, newSelected)。
6. 版本落盘：Mode=partial、StartPos/EndPos、LengthMode=req.LengthMode、
   TargetWords（custom 时=req.TargetWordCount）、Original/NewContent=全文、
   Similarity=diff。返回 map 在 whole 键集上加：mode="partial"、selectedWordCount、
   newSelectedWordCount、lengthMode、startPos、endPos（newContent=newFull）。

**B 测试**：按既有 whole 测试先例（novel_rewrite_handler_test.go 查实况）——
fake client 注入：partial 请求往返（选段拼接正确/版本字段/返回键）+ 选区不匹配
报错 + custom 无字数报错。**只跑定向** `go test ./internal/app -run TestNovelChapterRewrite -count=1`。

## 3. 线 C · 前端（EditorPanel/CreatePage/新 PartialRewriteModal + 测试）

- **EditorPanel**（components/novel/create/EditorPanel.tsx）：TextArea 加 ref；
  新可选 prop `onPartialRewrite?: (sel: { start: number; end: number; text: string }) => void`
  （rune 偏移——code-unit selectionStart/End 经 `[...content.slice(0, s)].length`
  换算，text=选中文本）；「局部重写」按钮（activeNode 且非 generating 时显示，
  重写按钮旁）：无选区（start===end）→ message.warning('请先选中要重写的文字
  段落')；有 → 回调。**该文件 147 行现无任何 state/ref——引入 useRef/按钮即
  从函数式组件升级最小必要 hooks，风格对齐。**
- **CreatePage**（pages/CreatePage.tsx）：pSel state + 挂 PartialRewriteModal
  （onApplied 同 RewriteModal→selectChapter 刷新正文）。
- **PartialRewriteModal**（components/novel/ 新文件，antd Modal 硬编码中文）：
  - 表单态：选段预览（只读折叠面板+字数）+ 指令 TextArea（必填 ≤1000，空禁用
    提交）+ 长度模式 RadioGroup（相近/扩展/精简/自定义；custom 时 InputNumber
    50~10000）+〔开始重写〕。
  - 提交：`NovelChapterRewrite(chapterNum, JSON.stringify({mode:'partial',
    source:'custom', custom_instructions, start_pos, end_pos, selected_text,
    length_mode, target_word_count?}))`。
  - 运行态：Spin+禁关闭。
  - 结果态：新选段预览+统计行（选段 N 字→新 M 字·相似度 X%）+〔应用并写回〕
    （NovelApplyRewriteVersion(versionId)→onApplied）+〔放弃版本〕
    （NovelDiscardRewriteVersion）。
- **测试**：PartialRewriteModal 4 例（提交载荷形状含 rune 偏移与 selected_text/
  结果统计渲染/应用调用/指令空禁用）；EditorPanel 2 例（无选区 warning/有选区
  回调 rune 换算——用含代理对样例「𝒜文」验证 code-unit≠rune 时换算正确）。
  **mock 不动**（NovelChapterRewrite mock 抛错=浏览器端诚实）。
- **不碰** bridge/mock/locales/bindingNames/spaceBindings（主代理收口统一复核）。

## 4. 主代理收口（v4.321.0）

线间对账 → tsc -b/eslint/定向 go+vitest → ci.ps1 恰一次 → 版本三处 → build →
exe/SUMS/桌面副本/冒烟 → releases/v4.321.0.md+CHANGELOG/README×2 → .gaea
AGENTS 迁 1 插 1+progress/todos → commit+tag。drift 复核 696（零绑定变化）。

## 5. 足迹互斥表

| 线 | 独占足迹 |
|---|---|
| A | internal/rewrite/partial.go(+partial_test.go)、engine.go(+engine_test 增例) |
| B | internal/app/novel_rewrite_handler.go(+_test 增例)、internal/types/plot_v2.go（RewriteRequest +SelectedText 一行） |
| C | frontend/src/components/novel/create/EditorPanel.tsx(+test 增例)、pages/CreatePage.tsx、components/novel/PartialRewriteModal.tsx(+test) |
| 主 | 本规格书、门禁、版本、releases/CHANGELOG/README/.gaea |

## 6. 出口对照与观察池

- 判据：选中一段→只重写该段（前后文零变化，拼接断言钉死）✅；版本库可回滚
  （partial 版本 Apply/Restore 走既有链路）✅；长度模式四档区间进指令 ✅。
- 观察池：选段高亮回写编辑器（应用后选中新区段）；流式输出（V1 同步等待）；
  deslop 模式入口（类型已留）。
