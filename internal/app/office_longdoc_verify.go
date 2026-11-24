package app

// office_longdoc_verify.go — 方案档专属交付验证工具（2026-11-21 用户拍板：沿江
// 1公里技术标会话倒逼出的验证体系脚本收编为工具）。五脚本通用内核：
//
//   - chapter（verify_ch.py）：单章四对账——题注↔引用↔章末索引三方同步、图引
//     存在性、docx 内嵌图 md5 逐一比对+占位残留、禁用词（否定语境排除）；
//   - cross（cross_check.py）：跨章——图表编号唯一、禁用词全册、口径词×各章
//     矩阵、各章字数；
//   - book（verify_v1.py/acceptance_all.py）：整册 docx——内嵌数/占位文本、
//     章标题在册、styles.xml 字面量读数、页眉页脚域、分节数、源 md 完整性；
//   - ink（ink_color.py）：渲染页 PNG 暗部像素平均色+饱和度判色。
//
// 项目耦合的东西（期望图表数/禁用词/口径词）全部调用时传参，内核确定性零 LLM。
// 未收编：qc_geom（matplotlib 活体拦截进不了 Go，留技能纪律）、CPM 自算交叉
// 验证（引擎为权威，纪律在 schedule-edit 技能）。
//
// 装配/契约与 longdoc_manifest 同族：toolgroups 方案组+gaea_handler 装配+
// registrymap 契约行+spacetags 静态表四处同批；只读（无 WritePaths）。

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── 编号形态（与 md_validate 同思想，收窄到「章-序」两段项目规范形态）──────

var ldvCaptionRe = regexp.MustCompile(`^(?:>\s*)?(图|表)\s*(\d+)[-–—](\d+)[A-Za-z]?\s`)
var ldvRefRe = regexp.MustCompile(`(图|表)\s*(\d+)[-–—](\d+)[A-Za-z]?`)
var ldvIndexRowRe = regexp.MustCompile(`\|\s*(图|表)\s*(\d+)[-–—](\d+)[A-Za-z]?\s*\|`)
var ldvImageRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
var ldvIndexHeadingRe = regexp.MustCompile(`图表索引`)
var ldvHanRe = regexp.MustCompile(`\p{Han}`)

// ldvNegPrefixes 否定语境前缀（命中词前 30 字含其一=反面引用，不计禁用）。
var ldvNegPrefixes = []string{"不以", "避免", "禁止", "不得", "不能", "不宜", "无须", "无需"}

// ldvDefaultPlaceholders docx 占位残留默认词（外加调用方自定义）。
var ldvDefaultPlaceholders = []string{"图片加载失败", "TODO", "【待补充】", "【占位"}

// ldvBand ink 采样条带（y 比例 0-1）；缺省带来自 ink_color.py 实战取位。
type ldvBand struct {
	Label  string  `json:"label"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

var ldvDefaultBands = []ldvBand{
	{"页眉", 0.04, 0.07}, {"标题行", 0.093, 0.115}, {"正文上", 0.135, 0.16},
	{"正文中", 0.30, 0.33}, {"页脚", 0.93, 0.97},
}

// ldvParams 四门共用入参（解耦形态：字段按 action 取用）。
type ldvParams struct {
	Action       string    `json:"action"`
	Path         string    `json:"path"`
	Paths        []string  `json:"paths"`
	Docx         string    `json:"docx"`
	PNG          string    `json:"png"`
	Bands        []ldvBand `json:"bands"`
	Forbidden    []string  `json:"forbidden"`
	Keys         []string  `json:"keys"`
	BadKeys      []string  `json:"bad_keys"`
	ExpectFigs   *int      `json:"expect_figs"`
	ExpectTabs   *int      `json:"expect_tabs"`
	Placeholders []string  `json:"placeholders"`
}

type longdocVerifyTool struct{ a *App }

func (longdocVerifyTool) Name() string   { return "longdoc_verify" }
func (longdocVerifyTool) ReadOnly() bool { return true }

func (longdocVerifyTool) Description() string {
	return "方案档专属交付验证（只读、确定性、零 LLM；技术标实测验证体系工具化）。五门：chapter=单章验收（题注↔引用↔章末索引三方同步+图引存在性+docx 内嵌图 md5 逐一比对+占位残留+禁用词否定语境排除+口径词计数）；cross=跨章核查（图表编号跨章唯一+引用↔题注并集对账+跨章引用语义初筛〔引用句↔被引标题 bigram 重合，低重合=引用错表高疑标记，不判 fail〕+禁用词全册+口径词×各章出现矩阵+各章字数）；book=整册终验（docx 内嵌数/占位文本/章标题在册/styles.xml 颜色字号字面量读数/页眉页脚域/分节数/源 md 完整）；ink=渲染页像素判色（doc_render 出图后暗部平均色，饱和度差>25 判彩色——标题改色是否真生效用它）；layout=版面机检（空白页/墨覆盖两口径〔整页面积与已占用版面分母同给〕/图片有效 ppi/文本越界 bbox——图文占比不再靠手搓脚本三版口径打架）。期望图表数/禁用词/口径词等项目耦合项全部调用时传参。清单结构机检另有 longdoc_manifest check；编号撞号跳号单章内形态用 md_validate（paths 多文件=全册并集对账）。"
}

func (longdocVerifyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object","required":["action"],
"properties":{
  "action":{"type":"string","enum":["chapter","cross","book","ink","layout"],"description":"chapter=单章验收；cross=跨章核查；book=整册终验；ink=像素判色；layout=版面机检（空白页/墨覆盖两口径/图片有效dpi/文本越界——docx 或 pdf 输入）"},
  "path":{"type":"string","description":"chapter：章节 md 路径"},
  "paths":{"type":"array","items":{"type":"string"},"description":"cross/book：md 清单（book=各章源文件，用于章标题在册与源文件完整）"},
  "docx":{"type":"string","description":"chapter/book：对照 docx（chapter=单章导出件，book=整册）"},
  "png":{"type":"string","description":"ink：doc_render 渲染出的页 PNG 路径"},
  "bands":{"type":"array","items":{"type":"object","properties":{"label":{"type":"string"},"top":{"type":"number"},"bottom":{"type":"number"}}},"description":"ink：水平采样条带（y 比例 0-1）；缺省 页眉/标题行/正文两段/页脚"},
  "forbidden":{"type":"array","items":{"type":"string"},"description":"禁用词（真命中计 fail；前 30 字含否定语境——不以/避免/禁止/不得等——不计）"},
  "keys":{"type":"array","items":{"type":"string"},"description":"口径词（各文件出现次数矩阵，只报不判）"},
  "bad_keys":{"type":"array","items":{"type":"string"},"description":"旧口径残留词（出现即 fail）"},
  "expect_figs":{"type":"integer","description":"chapter：期望图题注数（缺省只查连续性与对账不查总数）"},
  "expect_tabs":{"type":"integer","description":"chapter：期望表题注数"},
  "placeholders":{"type":"array","items":{"type":"string"},"description":"docx 占位残留词（默认 图片加载失败/TODO/【待补充】/【占位）"}
}
}`)
}

// ReadTarget 主读目标（访问策略面）：按 action 取 path/docx/png/paths[0]。
func (longdocVerifyTool) ReadTarget(args json.RawMessage) (string, bool) {
	var p ldvParams
	if err := json.Unmarshal(args, &p); err != nil {
		return "", false
	}
	primary := ""
	switch p.Action {
	case "chapter":
		primary = p.Path
	case "cross":
		if len(p.Paths) > 0 {
			primary = p.Paths[0]
		}
	case "book":
		primary = p.Docx
	case "ink":
		primary = p.PNG
	case "layout":
		primary = p.Docx
	}
	if strings.TrimSpace(primary) == "" {
		return "", false
	}
	abs, err := longdocResolvePath(primary)
	if err != nil {
		return "", false
	}
	return abs, true
}

func (t longdocVerifyTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p ldvParams
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("longdoc_verify 参数解析失败: %w", err)
	}
	switch p.Action {
	case "chapter":
		if strings.TrimSpace(p.Path) == "" {
			return "", fmt.Errorf("chapter 门需要 path（章节 md）")
		}
		return t.runChapter(&p)
	case "cross":
		if len(p.Paths) == 0 {
			return "", fmt.Errorf("cross 门需要 paths（≥2 个章节 md 才有跨章意义）")
		}
		return t.runCross(&p)
	case "book":
		if strings.TrimSpace(p.Docx) == "" {
			return "", fmt.Errorf("book 门需要 docx（整册文件）")
		}
		return t.runBook(&p)
	case "ink":
		if strings.TrimSpace(p.PNG) == "" {
			return "", fmt.Errorf("ink 门需要 png（doc_render 渲染页）")
		}
		return t.runInk(&p)
	case "layout":
		if strings.TrimSpace(p.Docx) == "" {
			return "", fmt.Errorf("layout 门需要 docx 或 pdf（整册文件路径）")
		}
		return t.runLayout(ctx, &p)
	default:
		return "", fmt.Errorf("未知 action %q（chapter/cross/book/ink）", p.Action)
	}
}

// ldvReceipt 回执聚合器。
type ldvReceipt struct {
	lines  []string
	issues []string
}

func (r *ldvReceipt) line(format string, a ...any) {
	r.lines = append(r.lines, "  "+fmt.Sprintf(format, a...))
}
func (r *ldvReceipt) fail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	r.issues = append(r.issues, msg)
	r.lines = append(r.lines, "  ✗ "+msg)
}

func (r *ldvReceipt) String(title string) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	b.WriteString(strings.Join(r.lines, "\n"))
	b.WriteString("\n")
	if len(r.issues) > 0 {
		fmt.Fprintf(&b, "✗ 验证未通过 %d 项（修复后重跑本工具确认清零）：\n", len(r.issues))
		for _, is := range r.issues {
			b.WriteString("  - " + is + "\n")
		}
		return b.String()
	}
	b.WriteString("✓ 全部通过\n")
	return b.String()
}

// ldvNormSeg 编号段归一（–/— → -）。
func ldvNormSeg(s string) string {
	s = strings.ReplaceAll(s, "–", "-")
	return strings.ReplaceAll(s, "—", "-")
}

// ── chapter：单章四对账 ─────────────────────────────────────────────────────

func (t longdocVerifyTool) runChapter(p *ldvParams) (string, error) {
	mdPath, err := longdocResolvePath(p.Path)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("读取章节失败: %w", err)
	}
	text := ldvNormSeg(string(raw))
	r := &ldvReceipt{}
	r.line("验收：%s（%d 字节，%d 汉字，%d 行）", mdPath, len(raw),
		len(ldvHanRe.FindAllString(text, -1)), strings.Count(text, "\n")+1)

	// ① 题注↔引用↔章末索引三方对账（索引区/题注行不算正文引用）。
	captions := map[string]int{} // key→行号（1-based）
	captionOrder := []string{}
	lines := strings.Split(text, "\n")
	indexZone := -1
	for i, ln := range lines {
		if indexZone < 0 && strings.HasPrefix(strings.TrimSpace(ln), "#") && ldvIndexHeadingRe.MatchString(ln) {
			indexZone = i
		}
		if m := ldvCaptionRe.FindStringSubmatch(strings.TrimSpace(ln)); m != nil {
			k := m[1] + m[2] + "-" + m[3]
			if prev, dup := captions[k]; dup {
				r.fail("题注撞号：%s（L%d 与 L%d）", k, prev, i+1)
			} else {
				captions[k] = i + 1
				captionOrder = append(captionOrder, k)
			}
		}
	}
	refs := map[string][]int{}
	for i, ln := range lines {
		if indexZone >= 0 && i >= indexZone {
			continue
		}
		if ldvCaptionRe.MatchString(strings.TrimSpace(ln)) {
			continue
		}
		for _, m := range ldvRefRe.FindAllStringSubmatch(ln, -1) {
			refs[m[1]+m[2]+"-"+m[3]] = append(refs[m[1]+m[2]+"-"+m[3]], i+1)
		}
	}
	nFig, nTab := 0, 0
	for _, k := range captionOrder {
		if strings.HasPrefix(k, "图") {
			nFig++
		} else {
			nTab++
		}
	}
	ldvContinuity(r, captions, captionOrder)
	r.line("[1] 题注 图 %d、表 %d；正文引用 图 %d 种、表 %d 种", nFig, nTab,
		ldvCountPrefix(refs, "图"), ldvCountPrefix(refs, "表"))
	if p.ExpectFigs != nil && *p.ExpectFigs != nFig {
		r.fail("图题注 %d ≠ 期望 %d", nFig, *p.ExpectFigs)
	}
	if p.ExpectTabs != nil && *p.ExpectTabs != nTab {
		r.fail("表题注 %d ≠ 期望 %d", nTab, *p.ExpectTabs)
	}
	// 引用二分：本章号引用=严格对账；跨章引用（引其他章的表/图）本章无法
	// 裁决——列信息不判 fail，对账责任在 cross 门（全册题注并集裁决）。
	localChapters := map[string]bool{}
	for _, k := range captionOrder {
		runes := []rune(k)
		var ch int
		if n, err := fmt.Sscanf(string(runes[1:]), "%d", &ch); err == nil && n == 1 {
			localChapters[string(runes[0])+fmt.Sprint(ch)] = true
		}
	}
	localRefs, crossRefs := map[string][]int{}, map[string][]int{}
	for k, at := range refs {
		runes := []rune(k)
		isLocal := false
		if len(runes) >= 3 {
			var ch int
			if n, err := fmt.Sscanf(string(runes[1:]), "%d", &ch); err == nil && n == 1 {
				isLocal = localChapters[string(runes[0])+fmt.Sprint(ch)]
			}
		}
		if isLocal {
			localRefs[k] = at
		} else {
			crossRefs[k] = at
		}
	}
	for k, at := range localRefs {
		if _, ok := captions[k]; !ok {
			r.fail("孤立引用：正文引 %s（L%v）但本章无此题注", k, at)
		}
	}
	if len(crossRefs) > 0 {
		keys := make([]string, 0, len(crossRefs))
		for k := range crossRefs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		r.line("[1+] 跨章引用 %d 种（%s…）——本章不裁决，对账走 cross 门全册并集", len(crossRefs), strings.Join(keys[:min(3, len(keys))], "、"))
	}
	for _, k := range captionOrder {
		if _, ok := refs[k]; !ok {
			r.line("[1+] 题注 %s 本章正文未引用（可能被其他章引用——严格判定走 cross 门并集）", k)
		}
	}
	if indexZone >= 0 {
		idxSet := map[string]bool{}
		for i := indexZone; i < len(lines); i++ {
			if m := ldvIndexRowRe.FindStringSubmatch(lines[i]); m != nil {
				idxSet[m[1]+m[2]+"-"+m[3]] = true
			}
		}
		for _, k := range captionOrder {
			if !idxSet[k] {
				r.fail("章末索引缺条目：%s", k)
			}
		}
		for k := range idxSet {
			if _, ok := captions[k]; !ok {
				r.fail("章末索引多出正文没有的条目：%s", k)
			}
		}
		r.line("[1+] 章末图表索引 %d 条与题注对账", len(idxSet))
	} else {
		r.line("[1+] 无章末图表索引（按项目约定决定是否补——索引与题注同源）")
	}

	// ② 图引存在性（相对路径按章节目录解析；缺失=导出必然缺图）。
	chDir := filepath.Dir(mdPath)
	imgRefs := []string{}
	for _, m := range ldvImageRe.FindAllStringSubmatch(text, -1) {
		imgRefs = append(imgRefs, strings.TrimSpace(m[1]))
	}
	for _, ref := range imgRefs {
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			r.fail("图引是外链（导出链不拉取）：%s", ref)
			continue
		}
		img := ref
		if !filepath.IsAbs(img) {
			img = filepath.Join(chDir, ref)
		}
		if _, serr := os.Stat(img); serr != nil {
			r.fail("图引文件缺失：%s（按 %s 解析）", ref, img)
		}
	}
	r.line("[2] 图引 %d 处（相对 %d）；缺失见上", len(imgRefs), ldvCountRelative(imgRefs))

	// ③ docx 对照（给了才跑）：内嵌 md5 逐一比对+drawing 对账+占位残留。
	if strings.TrimSpace(p.Docx) != "" {
		dxPath, derr := longdocResolvePath(p.Docx)
		if derr != nil {
			return "", derr
		}
		media, drawing, docText, zerr := ldvDocxFacts(dxPath)
		if zerr != nil {
			return "", fmt.Errorf("docx 读取失败: %w", zerr)
		}
		bad := 0
		for _, ref := range imgRefs {
			img := ref
			if !filepath.IsAbs(img) {
				img = filepath.Join(chDir, ref)
			}
			b, ferr := os.ReadFile(img)
			if ferr != nil {
				continue // 缺失已由 ② 报
			}
			sum := md5.Sum(b)
			if !media[hex.EncodeToString(sum[:])] {
				bad++
			}
		}
		r.line("[3] docx：内嵌媒体 %d、drawing %d；源图 md5 不符 %d（create_docx 原样内嵌口径）",
			len(media), drawing, bad)
		if bad > 0 {
			r.fail("docx 内嵌图与源文件 md5 不符 %d 处（导出件不是当前源图，重出）", bad)
		}
		if drawing != len(imgRefs) {
			r.fail("docx drawing %d ≠ 图引 %d", drawing, len(imgRefs))
		}
		ldvPlaceholders(r, docText, p.Placeholders)
	} else {
		r.line("[3] 未给 docx（单章 md 验收；导出后再跑可加 docx 对照）")
	}

	// ④ 禁用词（否定语境排除）+口径词+旧口径残留。
	ldvWordChecks(r, text, p.Forbidden, p.Keys, p.BadKeys)
	return r.String("longdoc_verify · chapter"), nil
}

// ldvContinuity 同 kind+章内序号 1..max 连续（跳号点名缺口）。
func ldvContinuity(r *ldvReceipt, captions map[string]int, order []string) {
	type seg struct{ no int }
	groups := map[string][]seg{}
	var gorder []string
	for _, k := range order {
		runes := []rune(k)
		if len(runes) < 3 {
			continue
		}
		var ch, no int
		// rune 切片：首 rune 是 图/表，其余形如 "3-1"。字节切片会把多字节
		// 汉字劈开（图=3 字节，k[1:] 是乱码前缀——实测踩过）。
		if _, err := fmt.Sscanf(string(runes[1:]), "%d-%d", &ch, &no); err != nil {
			continue
		}
		g := string(runes[0]) + fmt.Sprint(ch)
		if _, ok := groups[g]; !ok {
			gorder = append(gorder, g)
		}
		groups[g] = append(groups[g], seg{no})
	}
	for _, g := range gorder {
		maxNo, seen := 0, map[int]bool{}
		for _, s := range groups[g] {
			if s.no > maxNo {
				maxNo = s.no
			}
			seen[s.no] = true
		}
		for i := 1; i <= maxNo; i++ {
			if !seen[i] {
				r.fail("%s 章内跳号：缺 %s-%d（现有 1..%d）", g, g, i, maxNo)
			}
		}
	}
}

func ldvCountPrefix(refs map[string][]int, prefix string) int {
	n := 0
	for k := range refs {
		if strings.HasPrefix(k, prefix) {
			n++
		}
	}
	return n
}

func ldvCountRelative(refs []string) int {
	n := 0
	for _, r := range refs {
		if !filepath.IsAbs(r) && !strings.HasPrefix(r, "http://") && !strings.HasPrefix(r, "https://") {
			n++
		}
	}
	return n
}

// ldvPlaceholders docx 正文占位残留（默认词+自定义词）。
func ldvPlaceholders(r *ldvReceipt, docText string, extra []string) {
	words := append([]string{}, ldvDefaultPlaceholders...)
	words = append(words, extra...)
	found := 0
	for _, w := range words {
		if n := strings.Count(docText, w); n > 0 {
			r.fail("docx 正文残留「%s」%d 处", w, n)
			found += n
		}
	}
	if found == 0 {
		r.line("[3+] 占位残留 0（默认词+%d 自定义词）", len(extra))
	}
}

// ldvWordChecks 禁用词（否定语境排除）+口径词计数+旧口径残留。
func ldvWordChecks(r *ldvReceipt, text string, forbidden, keys, badKeys []string) {
	for _, w := range forbidden {
		hits, excluded := 0, 0
		for _, at := range ldvAllIndexes(text, w) {
			if ldvNegContext(text, at) {
				excluded++
				continue
			}
			hits++
		}
		if hits+excluded > 0 {
			r.line("[4] 禁用词「%s」：真命中 %d（否定语境排除 %d）", w, hits, excluded)
		}
		if hits > 0 {
			r.fail("禁用词「%s」真命中 %d 次", w, hits)
		}
	}
	for _, k := range keys {
		r.line("[4] 口径词「%s」出现 %d 次", k, strings.Count(text, k))
	}
	for _, b := range badKeys {
		if n := strings.Count(text, b); n > 0 {
			r.fail("旧口径残留「%s」%d 处", b, n)
		}
	}
}

// ldvNegContext 命中词前 30 字（rune）是否含否定语境前缀。
func ldvNegContext(text string, byteAt int) bool {
	ctxRunes := []rune(text[:byteAt])
	start := len(ctxRunes) - 30
	if start < 0 {
		start = 0
	}
	ctx := string(ctxRunes[start:])
	for _, n := range ldvNegPrefixes {
		if strings.Contains(ctx, n) {
			return true
		}
	}
	return false
}

func ldvAllIndexes(s, sub string) []int {
	var out []int
	for i := 0; ; {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			return out
		}
		out = append(out, i+j)
		i += j + len(sub)
	}
}

// ldvDocxFacts 打开 docx：media md5 集、document.xml 的 drawing 数、去标签正文。
func ldvDocxFacts(path string) (map[string]bool, int, string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, 0, "", err
	}
	defer func() { _ = zr.Close() }()
	media := map[string]bool{}
	doc := ""
	for _, f := range zr.File {
		switch {
		case strings.HasPrefix(f.Name, "word/media/"):
			if rc, oerr := f.Open(); oerr == nil {
				b, _ := io.ReadAll(rc)
				_ = rc.Close()
				sum := md5.Sum(b)
				media[hex.EncodeToString(sum[:])] = true
			}
		case f.Name == "word/document.xml":
			if rc, oerr := f.Open(); oerr == nil {
				b, _ := io.ReadAll(rc)
				_ = rc.Close()
				doc = string(b)
			}
		}
	}
	return media, strings.Count(doc, "<w:drawing"), ldvStripTags(doc), nil
}

var ldvTagRe = regexp.MustCompile(`<[^>]+>`)

func ldvStripTags(s string) string { return ldvTagRe.ReplaceAllString(s, "") }

// ── cross：跨章核查 ─────────────────────────────────────────────────────────

func (t longdocVerifyTool) runCross(p *ldvParams) (string, error) {
	type chText struct {
		base string
		text string
	}
	var chs []chText
	for _, raw := range p.Paths {
		abs, err := longdocResolvePath(raw)
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", raw, err)
		}
		chs = append(chs, chText{filepath.Base(abs), ldvNormSeg(string(b))})
	}
	r := &ldvReceipt{}
	// ① 图表编号跨章唯一 + 全册引用↔题注并集双向对账（chapter 门不裁决的
	// 跨章引用与仅被他章引用的题注，在这里闭合）。
	seen := map[string]string{}
	capTitle := map[string]string{} // 题注 key → 题注行全文（语义初筛用）
	allNums := 0
	unionRefs := map[string]string{} // 引用 key → 首现文件
	for _, c := range chs {
		indexZone := false
		for _, ln := range strings.Split(c.text, "\n") {
			trimmed := strings.TrimSpace(ln)
			if strings.HasPrefix(trimmed, "#") && ldvIndexHeadingRe.MatchString(ln) {
				indexZone = true
			}
			if m := ldvCaptionRe.FindStringSubmatch(trimmed); m != nil {
				k := m[1] + m[2] + "-" + m[3]
				allNums++
				if prev, dup := seen[k]; dup {
					r.fail("图表编号跨章重复：%s（%s 与 %s）", k, prev, c.base)
				} else {
					seen[k] = c.base
					capTitle[k] = trimmed
				}
				continue
			}
			if indexZone {
				continue // 索引区不算正文引用
			}
			for _, m := range ldvRefRe.FindAllStringSubmatch(ln, -1) {
				k := m[1] + m[2] + "-" + m[3]
				if _, ok := unionRefs[k]; !ok {
					unionRefs[k] = c.base
				}
			}
		}
	}
	r.line("[1] 图表编号共 %d 个，重复 %d", allNums, len(r.issues))
	for k, where := range unionRefs {
		if _, ok := seen[k]; !ok {
			r.fail("孤立引用（全册并集）：%s 引用 %s 但全册无此题注", where, k)
		}
	}
	for k := range seen {
		if _, ok := unionRefs[k]; !ok {
			r.fail("题注全册未被引用：%s", k)
		}
	}
	r.line("[1+] 引用↔题注并集对账：引用 %d 种、题注 %d 个", len(unionRefs), len(seen))
	// ①半 语义初筛（2026-11-21 反馈 P0-3）：「引用错表」=编号存在/连续/对账齐
	// 但被引内容已换（并行子代理沿用重排前旧号）——三道存在性门全绿仍漏。对每处
	// 跨章引用输出被引标题+引用句上下文+汉字 bigram 重合初筛；低重合=标记请核对
	// （不判 fail——初筛有误报率，裁决留给人/AI 复核）。
	type crossRefCase struct {
		key, file, seg string
	}
	var crossRefs []crossRefCase
	for _, c := range chs {
		indexZone := false
		for _, ln := range strings.Split(c.text, "\n") {
			trimmed := strings.TrimSpace(ln)
			if strings.HasPrefix(trimmed, "#") && ldvIndexHeadingRe.MatchString(ln) {
				indexZone = true
			}
			if ldvCaptionRe.MatchString(trimmed) || indexZone {
				continue
			}
			for _, m := range ldvRefRe.FindAllStringSubmatch(ln, -1) {
				k := m[1] + m[2] + "-" + m[3]
				if where, ok := seen[k]; ok && where != c.base {
					crossRefs = append(crossRefs, crossRefCase{k, c.base, ldvRefSentence(ln, m[0])})
				}
			}
		}
	}
	if len(crossRefs) > 0 {
		flagged, passed := 0, 0
		var lines []string
		for _, cr := range crossRefs {
			shared := ldvSharedBigrams(cr.seg, capTitle[cr.key])
			line := fmt.Sprintf("    %s ← %s「%s」 ↔ 被引「%s」", cr.key, cr.file, ldvShort(cr.seg, 42), ldvShort(capTitle[cr.key], 28))
			if len(shared) > 0 {
				passed++
				shown := shared
				if len(shown) > 3 {
					shown = shown[:3]
				}
				lines = append(lines, line+fmt.Sprintf(" 重合%d（%s）", len(shared), strings.Join(shown, "·")))
			} else {
				flagged++
				lines = append(lines, line+" 重合0【低重合·请核对：引用语境与被引标题疑似不符】")
			}
		}
		r.line("[1s] 跨章引用语义初筛：%d 处（重合≥1 过筛 %d、低重合标记 %d，不判 fail）", len(crossRefs), passed, flagged)
		for _, l := range lines {
			r.line("%s", l)
		}
	}
	// ② 禁用词全册（否定语境排除）。
	for _, w := range p.Forbidden {
		tot, exc := 0, 0
		per := map[string]int{}
		for _, c := range chs {
			for _, at := range ldvAllIndexes(c.text, w) {
				if ldvNegContext(c.text, at) {
					exc++
					continue
				}
				tot++
				per[c.base]++
			}
		}
		r.line("[2] 禁用词「%s」全册真命中 %d（否定语境排除 %d）：%s", w, tot, exc, ldvFmtCounts(per))
		if tot > 0 {
			r.fail("禁用词「%s」全册真命中 %d 次", w, tot)
		}
	}
	// ③ 口径词矩阵+旧口径残留。
	if len(p.Keys) > 0 {
		r.line("[3] 口径词 × 各文件出现矩阵：")
		for _, k := range p.Keys {
			per := map[string]int{}
			for _, c := range chs {
				per[c.base] = strings.Count(c.text, k)
			}
			r.line("    %-16s %s", k, ldvFmtCounts(per))
		}
	}
	for _, b := range p.BadKeys {
		for _, c := range chs {
			if n := strings.Count(c.text, b); n > 0 {
				r.fail("「%s」残留于 %s %d 处（旧口径）", b, c.base, n)
			}
		}
	}
	// ④ 各章字数。
	r.line("[4] 各章规模：")
	tot := 0
	for _, c := range chs {
		n := len(ldvHanRe.FindAllString(c.text, -1))
		tot += n
		r.line("    %-30s %6d 汉字", c.base, n)
	}
	r.line("    合计 %d 汉字（%d 文件）", tot, len(chs))
	return r.String("longdoc_verify · cross"), nil
}

// ldvRefSentence 取引用所在的句片段（句读边界 [。；;！？!?] 内含引用词的段，
// 截 60 rune）——语义初筛的引用语境展示。
func ldvRefSentence(ln, ref string) string {
	at := strings.Index(ln, ref)
	if at < 0 {
		return ldvShort(strings.TrimSpace(ln), 60)
	}
	start, end := at, at+len(ref)
	for start > 0 && !strings.ContainsRune("。；;！？!?", rune(ln[start-1])) {
		start--
	}
	for end < len(ln) && !strings.ContainsRune("。；;！？!?", rune(ln[end])) {
		end++
	}
	return ldvShort(strings.TrimSpace(ln[start:end]), 60)
}

// ldvShort 显示截断（rune 边界）。
func ldvShort(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

// ldvCJKBigrams 汉字 bigram 集合（语义初筛的重合度量——无分词依赖，共享双字
// 词（里程碑/监测/频次）即计重合）。
func ldvCJKBigrams(s string) map[string]bool {
	var cjk []rune
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			cjk = append(cjk, r)
		}
	}
	set := make(map[string]bool, len(cjk))
	for i := 0; i+1 < len(cjk); i++ {
		set[string(cjk[i:i+2])] = true
	}
	return set
}

// ldvSharedBigrams 引用句与被引标题的共享 bigram（确定性序输出）。
func ldvSharedBigrams(sentence, title string) []string {
	sb, tb := ldvCJKBigrams(sentence), ldvCJKBigrams(title)
	var shared []string
	for k := range sb {
		if tb[k] {
			shared = append(shared, k)
		}
	}
	sort.Strings(shared)
	return shared
}

func ldvFmtCounts(per map[string]int) string {
	names := make([]string, 0, len(per))
	for k := range per {
		names = append(names, k)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, k := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", k, per[k]))
	}
	if len(parts) == 0 {
		return "全部 0"
	}
	return strings.Join(parts, " ")
}

// ── book：整册终验 ───────────────────────────────────────────────────────────

func (t longdocVerifyTool) runBook(p *ldvParams) (string, error) {
	dxPath, err := longdocResolvePath(p.Docx)
	if err != nil {
		return "", err
	}
	media, drawing, docText, zerr := ldvDocxFacts(dxPath)
	if zerr != nil {
		return "", fmt.Errorf("docx 读取失败: %w", zerr)
	}
	r := &ldvReceipt{}
	fi, _ := os.Stat(dxPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}
	r.line("整册：%s（%.1f MB；内嵌媒体 %d，drawing %d，正文 %d 汉字）",
		dxPath, float64(size)/1024/1024, len(media), drawing,
		len(ldvHanRe.FindAllString(docText, -1)))
	// ① 占位残留。
	ldvPlaceholders(r, docText, p.Placeholders)
	// ② 各章源文件：在盘+章标题（首个一级标题）在整册正文中。
	for _, raw := range p.Paths {
		abs, aerr := longdocResolvePath(raw)
		if aerr != nil {
			return "", aerr
		}
		b, ferr := os.ReadFile(abs)
		if ferr != nil {
			r.fail("源 md 缺失：%s", raw)
			continue
		}
		title := ""
		for _, ln := range strings.Split(string(b), "\n") {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(ln, "# "))
				break
			}
		}
		if title == "" {
			r.fail("%s 无一级标题（清单漂移风险）", filepath.Base(abs))
			continue
		}
		if !strings.Contains(docText, title) {
			// 编号联动（2026-11-24）：docx 标题编号改样式层多级列表后，标题
			// 文字不再携带字面编号——字面全文不中时按裸题名对账。
			bare := longdocStripHeadingNumber(title)
			if bare == "" || !strings.Contains(docText, bare) {
				r.fail("章标题未出现在整册正文：%s（%s）", title, filepath.Base(abs))
			} else {
				r.line("[2] 章标题在册：%s（字面编号已由多级编号引擎取代，按裸题名对账）", bare)
			}
		} else {
			r.line("[2] 章标题在册：%s", title)
		}
	}
	// ③ styles.xml 字面量读数（heading 系颜色/字号/另起页——证据非判定）。
	if styles := ldvReadStyles(dxPath); len(styles) > 0 {
		r.line("[3] 样式表读数（解包字面量，最终事实源——run 级直接格式仍可能覆盖，判色用 ink 门）：")
		for _, s := range styles {
			r.line("    %s", s)
		}
	}
	// ④ 页眉页脚与分节（存在性+域指令；分节数=页码连续性口径）。
	hasHdr, ftr, sects := ldvHeaderFooterSections(dxPath)
	r.line("[4] 页眉 %s；页脚 %s；分节数 %d（单节=页码连续口径）",
		ldvYesNo(hasHdr), ldvYesNo(ftr != ""), sects)
	if ftr != "" {
		r.line("    页脚域：%s", ftr)
	}
	return r.String("longdoc_verify · book"), nil
}

func ldvYesNo(b bool) string {
	if b {
		return "有"
	}
	return "无"
}

var ldvStyleBlockRe = regexp.MustCompile(`(?s)<w:style [^>]*w:styleId="([^"]+)"[^>]*>(.*?)</w:style>`)
var ldvColorRe = regexp.MustCompile(`<w:color w:val="([^"]+)"`)
var ldvSzRe = regexp.MustCompile(`<w:sz w:val="(\d+)"`)
var ldvStyleNameRe = regexp.MustCompile(`<w:name w:val="([^"]+)"`)

// ldvPageBreakBeforeRe 另起页元素（含可选 w:val）。OOXML 语义：元素缺省=开，
// w:val 取 0/false/off=关、其余（含缺省/1/true/on）=开（2026-11-21 反馈 P1-5：
// 此前按元素出现即读「是」，val="0" 被反向读成另起页，驱动了错误写法）。
var ldvPageBreakBeforeRe = regexp.MustCompile(`<w:pageBreakBefore(?:\s+w:val="([^"]*)")?\s*/>`)
var ldvInstrRe = regexp.MustCompile(`<w:instrText[^>]*>([^<]*)</w:instrText>`)

// ldvReadZipEntry 读 zip 内单个条目（缺件返回 ""）。
func ldvReadZipEntry(zr *zip.ReadCloser, name string) string {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		if rc, err := f.Open(); err == nil {
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			return string(b)
		}
	}
	return ""
}

// ldvReadStyles heading 系样式的字面量读数（颜色/字号/另起页）。
func ldvReadStyles(path string) []string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil
	}
	defer func() { _ = zr.Close() }()
	src := ldvReadZipEntry(zr, "word/styles.xml")
	var out []string
	for _, m := range ldvStyleBlockRe.FindAllStringSubmatch(src, -1) {
		name := m[1]
		if nm := ldvStyleNameRe.FindStringSubmatch(m[2]); nm != nil {
			name = nm[1]
		}
		if !strings.Contains(strings.ToLower(name), "heading") && !strings.Contains(name, "标题") {
			continue
		}
		color, size, pb := "-", "-", "否"
		if c := ldvColorRe.FindStringSubmatch(m[2]); c != nil {
			color = c[1]
		}
		if s := ldvSzRe.FindStringSubmatch(m[2]); s != nil {
			size = s[1]
		}
		if pbm := ldvPageBreakBeforeRe.FindStringSubmatch(m[2]); pbm != nil {
			switch strings.ToLower(pbm[1]) {
			case "0", "false", "off":
				// 显式关：元素在但 val 否定——读「否」。
			default:
				pb = "是"
			}
		}
		out = append(out, fmt.Sprintf("%-14s 颜色=%-8s 字号(半磅)=%-4s 另起页=%s", name, color, size, pb))
	}
	return out
}

// ldvHeaderFooterSections 页眉/页脚存在与页脚域指令、分节数。
func ldvHeaderFooterSections(path string) (bool, string, int) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return false, "", 0
	}
	defer func() { _ = zr.Close() }()
	hdr, ftr := false, ""
	sects := strings.Count(ldvReadZipEntry(zr, "word/document.xml"), "<w:sectPr")
	for _, f := range zr.File {
		switch {
		case strings.HasPrefix(f.Name, "word/header"):
			hdr = true
		case strings.HasPrefix(f.Name, "word/footer"):
			if rc, oerr := f.Open(); oerr == nil {
				b, _ := io.ReadAll(rc)
				_ = rc.Close()
				var instr []string
				for _, m := range ldvInstrRe.FindAllStringSubmatch(string(b), -1) {
					instr = append(instr, strings.TrimSpace(m[1]))
				}
				ftr = "有"
				if len(instr) > 0 {
					ftr = strings.Join(instr, "；")
				}
			}
		}
	}
	return hdr, ftr, sects
}

// ── ink：像素判色 ───────────────────────────────────────────────────────────

func (t longdocVerifyTool) runInk(p *ldvParams) (string, error) {
	pngPath, err := longdocResolvePath(p.PNG)
	if err != nil {
		return "", err
	}
	f, err := os.Open(pngPath)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("解码图片失败（ink 门吃 doc_render 出的 PNG）: %w", err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	bands := p.Bands
	if len(bands) == 0 {
		bands = ldvDefaultBands
	}
	r := &ldvReceipt{}
	r.line("判色：%s（%d×%d，暗部阈值 lum≤110，饱和度差>25 判彩色）", pngPath, w, h)
	for _, b := range bands {
		y0 := int(float64(h) * b.Top)
		y1 := int(float64(h) * b.Bottom)
		if b.Top < 0 || b.Bottom > 1 || y1 <= y0 {
			r.fail("条带 %s 比例区间非法（%v-%v，须 0-1 且下大于上）", b.Label, b.Top, b.Bottom)
			continue
		}
		var rs, gs, bs float64
		n := 0
		for y := y0; y < y1; y++ {
			for x := 0; x < w; x++ {
				cr, cg, cb, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				r8, g8, b8 := float64(cr>>8), float64(cg>>8), float64(cb>>8)
				if 0.299*r8+0.587*g8+0.114*b8 <= 110 {
					rs += r8
					gs += g8
					bs += b8
					n++
				}
			}
		}
		if n == 0 {
			r.line("%-6s 无暗部像素（该带没有文字/线条？确认条带取位）", b.Label)
			continue
		}
		avg := []float64{rs / float64(n), gs / float64(n), bs / float64(n)}
		sat := avg[0]
		minv := avg[0]
		for _, v := range avg {
			if v > sat {
				sat = v
			}
			if v < minv {
				minv = v
			}
		}
		sat -= minv
		verdict := "黑色"
		if sat > 25 {
			verdict = "蓝/彩色"
		}
		r.line("%-6s 平均色 RGB=(%.0f,%.0f,%.0f) 饱和度差 %.0f 暗像素 %d → %s",
			b.Label, avg[0], avg[1], avg[2], sat, n, verdict)
	}
	r.line("判读：标题改色是否真生效看「标题行」带；与 book 门样式表读数交叉印证。")
	return r.String("longdoc_verify · ink"), nil
}

// 接口钉（编译期锁 family 形态）。
var _ tool.Tool = longdocVerifyTool{}

// ── layout：版面机检（P1-6，2026-11-21 沿江反馈：图文占比全靠自写脚本量，
// 三版口径 43%→19.2%→24.6% 能差一倍）──────────────────────────────────
//
// 确定性面四件（零 LLM，poppler 三件套）：①空白页（72dpi 墨像素<0.5%）②墨
// 覆盖两口径（整页面积分母=默认公布口径；已占用版面分母=墨 bbox——两数同给
// 不打架）③图片清单+有效 ppi（<96 warn=插入尺寸放大致糊）④文本越界（bbox
// xMax>页宽=物理裁切）。表格断表/图题同页等语义面留技能层（qc_geom）。

func (t longdocVerifyTool) runLayout(ctx context.Context, p *ldvParams) (string, error) {
	src, err := longdocResolvePath(p.Docx)
	if err != nil {
		return "", err
	}
	// docx → 临时 PDF（pdf 直读）。
	pdfPath := src
	cleanup := ""
	switch strings.ToLower(filepath.Ext(src)) {
	case ".pdf":
	case ".docx", ".doc":
		tmp, terr := os.CreateTemp("", "gaea-layout-*.pdf")
		if terr != nil {
			return "", terr
		}
		cleanup = tmp.Name()
		_ = tmp.Close()
		defer func() { _ = os.Remove(cleanup) }()
		if cerr := docmd.ConvertToPdfFile(src, cleanup); cerr != nil {
			return "", fmt.Errorf("layout 门 docx→PDF 失败: %w", cerr)
		}
		pdfPath = cleanup
	default:
		return "", fmt.Errorf("layout 门输入须为 docx 或 pdf（当前 %s）", filepath.Ext(src))
	}
	r := &ldvReceipt{}

	// ① 像素面：72dpi 渲染逐页墨覆盖。
	pngs, perr := docmd.RenderPDFPageRange(ctx, pdfPath, filepath.Join(os.TempDir(), "gaea-layout-page"), 72, 0, 0)
	type pageInk struct {
		Page                  int
		InkPctPage, InkPctOcc float64 // 整页面积分母 / 已占用版面（墨 bbox）分母
	}
	var inks []pageInk
	if perr == nil {
		defer func() {
			for _, png := range pngs {
				_ = os.Remove(png)
			}
		}()
		for i, png := range pngs {
			ink, pctPage, pctOcc := ldvPageInk(png)
			if ink < 0 {
				continue // 解码失败页跳过（如实不猜）
			}
			inks = append(inks, pageInk{Page: i + 1, InkPctPage: pctPage, InkPctOcc: pctOcc})
			if ink < 0.005 {
				r.fail("空白页：第 %d 页墨覆盖 %.2f%%（<0.5%%）——成册出现空白页=分页/空段问题", i+1, pctPage*100)
			}
		}
	} else {
		r.line("[1] 像素面不可用（%v）——空白页/墨覆盖跳过", perr)
	}

	// ② 文本面：bbox 越界。
	pages, terr := docmd.PDFTextBBoxes(pdfPath)
	if terr == nil {
		overflow := 0
		for pi, pg := range pages {
			for _, w := range pg.Words {
				if w.XMax > pg.Width+0.5 || w.XMin < -0.5 {
					overflow++
					if overflow <= 8 {
						r.fail("文本越界：第 %d 页词 %q xMax=%.1f 超页宽 %.1f（物理裁切，渲染才能看到）", pi+1, ldvShort(w.Text, 16), w.XMax, pg.Width)
					}
				}
			}
		}
		if overflow > 8 {
			r.fail("文本越界共 %d 处（上列前 8）", overflow)
		}
	} else {
		r.line("[2] 文本 bbox 面不可用（%v）——越界检查跳过", terr)
	}

	// ③ 图片面：有效 ppi。
	imgs, ierr := docmd.PDFImagesInfo(pdfPath)
	if ierr == nil {
		low := 0
		var imgAreaTotal float64
		var pageArea float64
		if len(pages) > 0 {
			pageArea = pages[0].Width * pages[0].Height
		}
		for _, im := range imgs {
			// 插入尺寸（pt）=像素/ppi；页面积占比（图号占页折算的数据源）。
			if pageArea > 0 && im.XPPI > 0 && im.YPPI > 0 {
				imgAreaTotal += (float64(im.Width) / im.XPPI) * (float64(im.Height) / im.YPPI)
			}
			if im.XPPI < 96 || im.YPPI < 96 {
				low++
				if low <= 8 {
					r.line("    [warn] 第 %d 页图片 %d×%d 有效 %0.fppi（<96=插入尺寸放大，打印可能糊）", im.Page, im.Width, im.Height, (im.XPPI+im.YPPI)/2)
				}
			}
		}
		r.line("[3] 图片 %d 张（有效 ppi<96 共 %d 张）；图面折算占版 %.1f%%", len(imgs), low, pct(imgAreaTotal, pageArea))
	} else {
		r.line("[3] 图片面不可用（%v）", ierr)
	}

	// ④ 墨覆盖两口径总览（默认公布口径=整页面积分母；已占用版面分母同给）。
	if len(inks) > 0 {
		r.line("[4] 逐页墨覆盖（页面积%% / 已占用版面%%——对外报数注明口径）：")
		for i, k := range inks {
			if i >= 40 {
				r.line("    …余 %d 页略", len(inks)-i)
				break
			}
			r.line("    P%-3d %5.1f%%  %5.1f%%", k.Page, k.InkPctPage*100, k.InkPctOcc*100)
		}
	}
	r.line("layout 门说明：空白页/文本越界=硬伤；图片低 ppi=warn；表格断表/图题同页等语义面留技能层。")
	return r.String("longdoc_verify · layout"), nil
}

// ldvPageInk 单页墨分析：返回 (墨像素占比, 整页分母覆盖, 已占用分母覆盖)；
// 解码失败返回 -1。
func ldvPageInk(png string) (float64, float64, float64) {
	f, err := os.Open(png)
	if err != nil {
		return -1, 0, 0
	}
	defer func() { _ = f.Close() }()
	img, err := pngDecode(f)
	if err != nil {
		return -1, 0, 0
	}
	b := img.Bounds()
	total := 0
	ink := 0
	minX, minY, maxX, maxY := b.Dx(), b.Dy(), 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			rr, gg, bb, _ := img.At(x, y).RGBA()
			total++
			// 墨判定：任一通道低于 75%亮度（灰框/淡色都算墨——版面占用语义）。
			if rr>>8 < 192 || gg>>8 < 192 || bb>>8 < 192 {
				ink++
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if total == 0 || ink == 0 {
		return 0, 0, 0
	}
	pctPage := float64(ink) / float64(total)
	bbox := (maxX - minX + 1) * (maxY - minY + 1)
	pctOcc := float64(ink) / float64(bbox)
	return pctPage, pctPage, pctOcc
}

// pngDecode image.Decode 的 png 特化（ink 门同款管线）。
func pngDecode(f *os.File) (image.Image, error) {
	return png.Decode(f)
}

func pct(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b * 100
}
