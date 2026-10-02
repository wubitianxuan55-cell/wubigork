package config

import (
	"os"
	"strconv"
	"strings"
)

// render_preserve.go — P0-1 刀：保留式落盘渲染。
//
// RenderTOML 是有损的归一化渲染，只写渲染器建模的段与键。直接拿它整文件重写会把
// 两类内容从磁盘上静默删掉：
//  1. 渲染器完全不负责的顶层段/标量（[memory]/[dream]/[network]/[tasks]…、
//     workspace = "..."）——一次「始终允许」审批的 persist_allow 回写就会触发；
//  2. 渲染器负责的段内部、但渲染器忘了写的键（[agent] effort/subagent_effort/
//     approval_timeout_secs、[tools] compact、[permissions] hard_ask、
//     [[providers]] thinking/effort/prices、[sandbox] 与 plugins 的未渲染键…）。
//
// 保留式渲染对两类都保留：
//   - 顶层块：渲染器不负责的整块逐字保留（追加在末尾）。
//   - 根标量（workspace 之类）：前置到文件最前面。TOML 的 key=value 归属上一个
//     表头，实测 BurntSushi 会把 [sandbox] 之后的 workspace 吞进 sandbox，
//     读回来是空串——所以根标量必须写在第一张表之前。
//   - 已知段内部：键级合并。existing 里该段（含子段）/该数组表条目的键，凡渲染
//     结果没有同名键的，逐字合并回渲染结果对应块内（渲染键在前、保留键在后）。
//     这是泛化兜底：将来 Config 新增字段而渲染器忘了渲染，也不会丢。
//
// existing 为空或解析不到内容时，逐字节等价 RenderTOML（零回归）。

// rendererOwnedTopLevel 是 RenderTOML 真正会重新渲染的顶层表名。这里名字下的旧块
// 在顶层不做整块保留（否则写出重复表头 = 非法 TOML），其内部键由键级合并补回。
// 注意 plugins/sandbox 也在 RenderTOML 职责内（任务清单的 8 段列表没列全）。
var rendererOwnedTopLevel = map[string]bool{
	"agent":          true,
	"providers":      true,
	"tools":          true,
	"permissions":    true,
	"space_profiles": true,
	"sandbox":        true,
	"plugins":        true,
}

// rendererOwnedScalars 是渲染器负责的顶层标量键（旧行丢弃，改由渲染结果给出）。
var rendererOwnedScalars = map[string]bool{
	"default_model": true,
	"language":      true,
}

// preserve*Note 是保留内容前的来源注释（防手删 + 可追溯）。
const (
	preserveScalarNote  = "# ── 以下顶层标量由 RenderTOMLPreserving 原样保留（渲染器不负责，勿手删）──"
	preserveSectionNote = "# ── 以下段由 RenderTOMLPreserving 原样保留（渲染器不负责，勿手删）──"
	mergedKeyNote       = "# ── 以下键为渲染器未覆盖、按原样保留（RenderTOMLPreserving）──"
)

// rendererHeaderComments 是 RenderTOML 自己写在文件头部的注释（逐字）。
// 保留文件头注释时跳过它们，否则每次保存都会把这几行再堆一遍。
var rendererHeaderComments = map[string]bool{
	"# Tianxuan configuration.": true,
	"# Resolution order: flag > ./gaea.toml > ~/.config/gaea/config.toml > built-in defaults.": true,
	"# Secrets come from the environment via api_key_env; never put keys here.":                true,
}

// RenderTOMLPreserving 渲染 c 并保留 existing 里 RenderTOML 不负责的内容。
// 落盘（Save/SaveTo/WriteFile）请用它，不要直接用 RenderTOML。
func RenderTOMLPreserving(c *Config, existing string) string {
	rendered := RenderTOML(c)
	existing = strings.ReplaceAll(existing, "\r\n", "\n")
	if strings.TrimSpace(existing) == "" {
		return rendered // 无既有内容 = 与 RenderTOML 逐字节等价（零回归）
	}

	lines := strings.Split(existing, "\n")
	head, blocks := parseBlocks(lines)
	scalarLines, sections, extra := splitPreservable(head, blocks)
	// 文件头里的用户注释也带进保留区（渲染器头部注释照旧由 RenderTOML 给出）。
	extra = dropBlank(append(extra, headCommentLines(head)...))
	// P0#2 复核修正：键级合并必须无条件执行。此前在「无顶层保留内容」时提前
	// return，纯 owned 段配置（如只有 [space_profiles]/[[plugins]]、无未建模段
	// 与注释）会整段跳过合并——owned 段内渲染器未建模的键照丢。

	out := mergeOwnedSections(strings.Split(rendered, "\n"), lines)
	if len(scalarLines) > 0 {
		out = prependScalars(out, scalarLines)
	}
	if len(sections) > 0 || len(extra) > 0 {
		out = append(out, preserveSectionNote)
		for _, s := range sections {
			out = append(out, s.pre...)
			out = append(out, s.header)
			out = append(out, s.body...)
		}
		out = append(out, extra...)
	}
	return joinLines(out)
}

// prependScalars 把保留的根标量插到第一张表之前。TOML 的 key=value 归属上一个
// 表头（实测 [sandbox] 之后的 workspace 会被 BurntSushi 吞进 sandbox，读回来是
// 空串），所以根标量必须落在第一张表之前。
func prependScalars(rendered []string, scalars []string) []string {
	at := len(rendered)
	for i, ln := range rendered {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		at = i // 第一个非注释非空行（通常是 default_model = ...）之前
		break
	}
	lines := make([]string, 0, len(scalars)+2)
	lines = append(lines, preserveScalarNote)
	lines = append(lines, scalars...)
	lines = append(lines, "")
	out := make([]string, 0, len(rendered)+len(lines))
	out = append(out, rendered[:at]...)
	out = append(out, lines...)
	out = append(out, rendered[at:]...)
	return out
}

// dropBlank 去掉空行（保留区里堆空行没意义）。
func dropBlank(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// ── 原始块解析 ────────────────────────────────────────────────────────

// rawBlock 是 existing 里的一个原始表块：表头行 + 到下一个表头之前的全部原始行。
type rawBlock struct {
	key       string   // 配对键：普通表 = 规范化表名；数组表 = 表名#name 字段值
	array     bool     // [[name]] 数组表
	header    string   // 表头行原文
	body      []string // 正文行原文（含缩进/块内注释/子表头）
	tableName string   // 不含方括号的表名原文（如 "network.proxy"）
	pre       []string // 表头之前紧邻的注释/空行（整块保留时一并带走）
}

// parseBlocks 把文本拆成「文件头 + 顶层原始块」。首个表头之前的行（渲染器注释 +
// 根标量）作为 head 单独返回——它们是根级内容，若挂在首个 owned 块上会随该块
// 一起被丢弃，丢数据（P0-1 实现踩过一次：workspace 反向复活成旧值）。
func parseBlocks(lines []string) ([]string, []rawBlock) {
	var blocks []rawBlock
	var cur *rawBlock
	var pre []string
	var head []string // 首个表头之前的全部行（文件头）
	flush := func() {
		if cur != nil {
			blocks = append(blocks, *cur)
			cur = nil
		}
	}
	for _, ln := range lines {
		if name, ok := tableHeaderName(ln); ok {
			flush()
			b := &rawBlock{
				header:    ln,
				tableName: name,
				pre:       pre,
				key:       normalizeTableName(name),
				array:     isArrayTableHeader(ln),
			}
			if head == nil && len(blocks) == 0 {
				head = pre  // 首个表头：此前攒下的就是文件头
				b.pre = nil // 文件头不重复算成块的前置注释
			}
			pre = nil
			cur = b
			continue
		}
		if cur != nil {
			cur.body = append(cur.body, ln)
			continue
		}
		pre = append(pre, ln)
	}
	flush()
	if head == nil {
		head = pre // 文件里一张表都没有：整份内容都是文件头
	}
	return head, blocks
}

// headCommentLines 返回文件头里剥掉标量、渲染器头部注释与空行之后的行（用户注释）。
// 它们随保留段一起落到文件末尾的保留区，不会静默消失。
func headCommentLines(head []string) []string {
	var out []string
	for _, ln := range head {
		if _, ok := parseKeyLine(ln); ok {
			continue // 标量行另有处理（前置保留）
		}
		if strings.TrimSpace(ln) == "" || rendererHeaderComments[strings.TrimSpace(ln)] {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// isArrayTableHeader 报告表头是否为数组表（[[name]]）。
func isArrayTableHeader(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]")
}

// tableHeaderName 解析表头行，返回不含方括号的表名（原文，未规范化）。
// 必须以 '[' 开头、以 ']' 结尾才算表头——否则 "key = value"、注释等普通行
// 会被误判成表头（P0-1 实现踩过一次）。
func tableHeaderName(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return "", false
	}
	t = strings.TrimPrefix(t, "[[")
	t = strings.TrimSuffix(t, "]]")
	t = strings.TrimPrefix(t, "[")
	t = strings.TrimSuffix(t, "]")
	if t == "" || strings.ContainsAny(t, "[]") {
		return "", false
	}
	return strings.TrimSpace(t), true
}

// normalizeTableName 去掉表名里的空白与引号，用于配对
// （渲染器写 [space_profiles.work]，用户可能写 [ space_profiles . work ]）。
func normalizeTableName(name string) string {
	r := strings.NewReplacer(" ", "", "\t", "", `"`, "", `'`, "")
	return r.Replace(name)
}

// topLevelName 取规范化表名的第一段（network.proxy → network）。
func topLevelName(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return name
}

// keyLine 是一行 "key = value" 的解析结果（只保留键名——保留区按整行搬运，不需要值）。
type keyLine struct {
	key string
}

// parseKeyLine 解析一行 "key = value"（含缩进行；带行尾注释也算），返回键名与值原文。
// 注释行、表头行、空行不算。
func parseKeyLine(line string) (keyLine, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "[") {
		return keyLine{}, false
	}
	eq := strings.IndexByte(t, '=')
	if eq <= 0 {
		return keyLine{}, false
	}
	key := strings.TrimSpace(t[:eq])
	if key == "" {
		return keyLine{}, false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return keyLine{}, false
		}
	}
	return keyLine{key: key}, true
}

// ── 顶层保留判定（根标量 + 渲染器不负责的整段）────────────────────────

// section 是一段顶层保留内容：前置行 + 首行（表头或标量）+ 正文（全部原文）。
type section struct {
	pre    []string
	header string
	body   []string
}

// splitPreservable 返回要前置保留的根标量，以及渲染器不负责的顶层整段。
// 「被丢弃的 owned 块」的表头注释不丢：记进 extra 附到下一段保留内容前
// （顺序会挪位，但用户注释不会静默消失）。
func splitPreservable(head []string, blocks []rawBlock) ([]string, []section, []string) {
	var scalars []string
	for _, ln := range head {
		f, ok := parseKeyLine(ln)
		if !ok || rendererOwnedScalars[f.key] {
			continue // 非标量行/渲染器负责的标量：不当前置标量
		}
		scalars = append(scalars, ln)
	}
	var sections []section
	var extra []string
	for _, b := range blocks {
		if rendererOwnedTopLevel[topLevelName(b.tableName)] {
			// 该块整块丢弃，但用户写在表头前的注释留下。
			for _, ln := range b.pre {
				if strings.TrimSpace(ln) != "" {
					extra = append(extra, ln)
				}
			}
			continue
		}
		s := section{header: b.header, body: b.body}
		if len(extra) > 0 {
			// 前置注释 = 用户自己的表头注释 + 上一张 owned 表丢下的注释。
			s.pre = append(s.pre, b.pre...)
			s.pre = append(s.pre, extra...)
			extra = nil
		} else {
			s.pre = b.pre
		}
		sections = append(sections, s)
	}
	return scalars, sections, extra
}

// ── 已知段的键级合并 ─────────────────────────────────────────────────

// mergeOwnedSections 把 existing 里「渲染器负责的顶层段」中渲染器没写的键/条目，
// 合并进渲染结果对应块内（渲染键在前、保留键在后，带来源注释）。
//
// 配对口径：
//   - 普通表/子段（[agent]/[tools]/[permissions]/[sandbox]/[space_profiles.X…]）：
//     按规范化表名配对。
//   - 数组表（[[providers]]/[[plugins]]）：同一数组内按位次配对（第 N 条对第 N 条）。
//     必须按位次而不是按 name——Load 让渲染结果的条目与用户文件条目同序，
//     按 name 配会把「渲染器写的条目」与「用户文件条目」搅在一起（实测把
//     用户 p1 的 thinking/effort 补到了另一个条目上）。
//
// 配不上的 existing 块（含渲染结果里根本没有的子段）整块（连表头）作为遗留块保留。
func mergeOwnedSections(rendered []string, existing []string) []string {
	rblocks := pairBlocks(rendered)
	matched, legacies := pairByTable(rblocks, pairBlocks(existing))

	familyAnchor := map[string]int{} // 族 → 该族最后一块「正文末行」的下标（遗留块锚点）
	familyKeys := map[string]map[string]bool{}
	renderedKeys := map[string]map[string]bool{}
	for _, b := range rblocks {
		top := topLevelName(normalizeTableName(mustTableName(b.header)))
		if end, ok := blockBodyEnd(b); ok && end > familyAnchor[top] {
			familyAnchor[top] = end
		}
		if familyKeys[top] == nil {
			familyKeys[top] = map[string]bool{}
		}
		ks := map[string]bool{}
		for _, ln := range b.body {
			if f, ok := parseKeyLine(ln); ok {
				ks[f.key] = true
				familyKeys[top][f.key] = true
			}
		}
		renderedKeys[b.identity()] = ks
	}

	// 补录行按「锚定行」分组：配对上的块 → 锚在该块自己的正文末尾；
	// 遗留块 → 锚在该族最后一块的正文末尾；族不存在 → 追加到文件末尾。
	afterLine := map[int][]string{}
	var tail []string

	// 配对上的块：补它缺的键（含它内部未渲染的子段）。
	for _, b := range rblocks {
		fam := topLevelName(normalizeTableName(mustTableName(b.header)))
		if !rendererOwnedTopLevel[fam] {
			continue
		}
		ex, ok := matched[b.identity()]
		if !ok {
			continue
		}
		ks := renderedKeys[b.identity()]
		var keys []string
		var extra []section
		for j := 0; j < len(ex.body); j++ {
			ln := ex.body[j]
			if name, isHeader := tableHeaderName(ln); isHeader {
				sub := normalizeTableName(mustTableName(ex.header) + "." + name)
				end := j + 1
				for end < len(ex.body) {
					if _, next := tableHeaderName(ex.body[end]); next {
						break
					}
					end++
				}
				if !renderedHasTable(rblocks, sub) {
					extra = append(extra, section{header: ln, body: stripTrailingBlank(ex.body[j+1 : end])})
				}
				j = end - 1
				continue
			}
			f, ok := parseKeyLine(ln)
			if !ok || ks[f.key] {
				continue
			}
			if b.array && f.key == "name" {
				continue // name 是数组表的身份键：渲染器必然写了，别用旧值盖新值
			}
			keys = append(keys, ln)
		}
		if len(keys) == 0 && len(extra) == 0 {
			continue
		}
		anchor, ok := blockBodyEnd(b)
		if !ok {
			anchor = b.start // 空块（无正文）：补录挂在表头后
		}
		afterLine[anchor] = append(afterLine[anchor], mergedKeyNote)
		afterLine[anchor] = append(afterLine[anchor], keys...)
		for _, lg := range extra {
			afterLine[anchor] = append(afterLine[anchor], lg.header)
			afterLine[anchor] = append(afterLine[anchor], lg.body...)
		}
	}

	// 配不上的 existing 块：整块（连表头）保留。
	for _, b := range legacies {
		fam := topLevelName(normalizeTableName(mustTableName(b.header)))
		if !rendererOwnedTopLevel[fam] {
			continue // 未知顶层段由 splitPreservable 整块保留
		}
		lines := []string{mergedKeyNote, b.header}
		lines = append(lines, stripTrailingBlank(b.body)...)
		if anchor, ok := familyAnchor[fam]; ok {
			afterLine[anchor] = append(afterLine[anchor], lines...)
			continue
		}
		tail = append(tail, lines...)
	}

	if len(afterLine) == 0 && len(tail) == 0 {
		return rendered
	}
	var out []string
	for idx := 0; idx < len(rendered); idx++ {
		out = append(out, rendered[idx])
		out = append(out, afterLine[idx]...)
	}
	out = append(out, tail...)
	return out
}

// renderedHasTable 报告渲染结果里是否已有该规范化表名的块。
func renderedHasTable(rblocks []rblock, table string) bool {
	for _, b := range rblocks {
		if b.key == table {
			return true
		}
	}
	return false
}

// rblock 是归一化后的一个表块。
type rblock struct {
	key    string
	header string
	body   []string
	start  int // 表头行在原文里的下标
	array  bool
	idx    int // 在所在列表里的位次（配对用）
}

// blockBodyEnd 返回块「正文末行」在原文里的下标；块没有正文时返回 false。
func blockBodyEnd(b rblock) (int, bool) {
	if len(b.body) == 0 {
		return 0, false
	}
	return b.start + 1 + len(b.body) - 1, true
}

// identity 是块在配对表里的查找键（同一列表内唯一）。
func (b rblock) identity() string {
	return b.header + "\x00" + strconv.Itoa(b.idx)
}

// pairByTable 把渲染结果与 existing 的块配对：
//   - 数组表：同一数组内按位次配对（第 N 条对第 N 条）；
//   - 普通表/子段：按规范化表名配对。
//
// 返回「渲染块 identity → 对应 existing 块」与「配不上的 existing 块（遗留）」。
func pairByTable(rendered, existing []rblock) (map[string]rblock, []rblock) {
	byKey := map[string]int{}   // 普通表：表名 → rendered 下标
	byOrd := map[string][]int{} // 数组表：表名 → rendered 下标序
	for i, b := range rendered {
		if b.array {
			byOrd[b.key] = append(byOrd[b.key], i)
			continue
		}
		byKey[b.key] = i
	}
	ordUsed := map[string]int{}
	used := map[int]bool{}
	matched := map[string]rblock{}
	for _, b := range existing {
		var i int
		var ok bool
		if b.array {
			n := ordUsed[b.key]
			ordUsed[b.key] = n + 1
			list := byOrd[b.key]
			if n < len(list) {
				i, ok = list[n], true
			}
		} else {
			i, ok = byKey[b.key]
		}
		if !ok || used[i] {
			continue
		}
		used[i] = true
		matched[rendered[i].identity()] = b
	}
	// 配不上的 existing 块 = 遗留（重扫一遍，用同样的位次分配）。
	var legacies []rblock
	ordUsed = map[string]int{}
	for _, b := range existing {
		var i int
		var ok bool
		if b.array {
			n := ordUsed[b.key]
			ordUsed[b.key] = n + 1
			list := byOrd[b.key]
			if n < len(list) {
				i, ok = list[n], true
			}
		} else {
			i, ok = byKey[b.key]
		}
		if ok && used[i] {
			continue
		}
		legacies = append(legacies, b)
	}
	return matched, legacies
}

// pairBlocks 按行扫描文本，切出带表头的块列表（数组表条目标记 array）。
func pairBlocks(lines []string) []rblock {
	type pending struct {
		start  int
		header string
		name   string
		array  bool
	}
	var out []rblock
	var cur *pending
	flush := func(end int) {
		if cur == nil {
			return
		}
		body := stripTrailingBlank(lines[cur.start+1 : end])
		out = append(out, rblock{
			key:    normalizeTableName(cur.name),
			header: cur.header,
			body:   body,
			start:  cur.start,
			array:  cur.array,
			idx:    len(out),
		})
		cur = nil
	}
	for i, ln := range lines {
		name, isHeader := tableHeaderName(ln)
		if isHeader {
			flush(i)
			cur = &pending{start: i, header: ln, name: name, array: isArrayTableHeader(ln)}
		}
	}
	flush(len(lines))
	return out
}

// mustTableName 返回表头行的表名（调用方已确认是表头）。
func mustTableName(line string) string {
	name, _ := tableHeaderName(line)
	return name
}

// stripTrailingBlank 去掉末尾空行（统一由 joinLines 补换行）。
func stripTrailingBlank(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[:end]
}

// joinLines 用换行拼回文本（以换行收尾；入参已以 "\n" 收尾——即末元素为空串
// ——时不再追加，保证 split→join 对渲染结果逐字节稳定）。
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	s := strings.Join(lines, "\n")
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

// readFileOrEmpty 读现有配置内容；不存在/不可读返回空串
// （等价「无既有内容」→ RenderTOMLPreserving 退化为 RenderTOML）。
func readFileOrEmpty(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
