package app

// office_longdoc_tools.go — 方案运行模式专用工具两件（2026-11-21 众神殿
// pan_1795202763988 终案：长文方案编制的失败面拆分——清单机检廉价高频 /
// 整册构建昂贵事务）。
//
//   - longdoc_manifest：章节清单维护（JSON 常驻工作区 .gaea/work/longdoc/）
//     + 确定性机检：缺章/空章/重复章节/悬空来源/未映射评分点/未闭合占位符/
//     清单漂移（正文与清单脱节）。复用 md_validate 的正则骨架思想（确定性
//     判定），但锚清单结构而非单文件修订。
//   - longdoc_build：按清单分章组装整册 → 复用 GaeaExportDeliverable 模板链
//     （create_docx.py 等，不重写生成器）→ SHA256 Go 自算入回执。构建前强制
//     机检（带硬检问题拒绝组装——「清单漂移可检出」验收断言的执行位）；页数
//     由模型用 doc_render 渲染取（终案口径：本工具不渲染）。
//
// 装配点=extraToolsWithMediaRoot；空间=work（spacetags.go 静态表同批）；
// 模式面=方案档可见（standard/cost 经 boot.Options.ToolHide 隐藏）。
// 实现 WritePathResolver（DAG write_paths 声明面可用）；不标 PersistWrite
// （工作区工作文件非持久外部库，office_generate 族同判）。

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// longdocManifestFile 是清单的磁盘形态（唯一事实源；正文漂移由 check 闸）。
type longdocManifestFile struct {
	Title  string `json:"title"`
	Format string `json:"format,omitempty"`
	// Requirements 是评分点/硬性要求登记（id+原文）；check 断言每条都被至少
	// 一个章节映射（未映射=交付缺口）。
	Requirements []longdocRequirement `json:"requirements"`
	Chapters     []longdocChapter     `json:"chapters"`
}

type longdocRequirement struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type longdocChapter struct {
	No           int      `json:"no"`
	Title        string   `json:"title"`
	Path         string   `json:"path"`
	Sources      []string `json:"sources"`
	Requirements []string `json:"requirements"`
	Words        int      `json:"words"`
	// Figs/Tables 期望图表数（2026-11-21 反馈 P2-9：配额验收进 check，0=不校验）。
	Figs   int `json:"figs,omitempty"`
	Tables int `json:"tables,omitempty"`
}

// longdocFinding 是机检发现（hard=阻断构建）。
type longdocFinding struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Hard   bool   `json:"hard"`
}

// longdocImageLineRe 整行图引（与 create_docx.py 的解析规则同款：图引必须独占
// 一行，行内混排图不识别）。2026-11-21 复盘 A3：相对图引按输出目录解析全部
// 落空、18 张图静默降级为提示文字——机检+绝对化双闸后此路封死。
var longdocImageLineRe = regexp.MustCompile(`^!\[[^\]]*\]\(([^)]+)\)\s*$`)

// longdocResolvePath 归一工作区路径（绝对=原样交 WriteScopePolicy 校验；
// 相对=ResolveRelWithin，officeResolveWritePath 同判据）。
func longdocResolvePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少路径")
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}
	p, err := wspath.ResolveRelWithin(gaeaCwd(), raw)
	if err != nil {
		return "", fmt.Errorf("非法工作区相对路径: %s", raw)
	}
	return p, nil
}

// loadLongdocManifest 读清单文件（不存在=可读错误，set/init 场景可建）。
func loadLongdocManifest(manifestPath string) (*longdocManifestFile, error) {
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var m longdocManifestFile
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("清单 JSON 解析失败（%s）: %w", manifestPath, err)
	}
	return &m, nil
}

// longdocCheck 清单机检（确定性，零 LLM）。返回发现列表；hard 项存在时
// longdoc_build 拒绝组装。
func longdocCheck(m *longdocManifestFile, manifestPath string) []longdocFinding {
	var out []longdocFinding
	add := func(code, detail string, hard bool) {
		out = append(out, longdocFinding{Code: code, Detail: detail, Hard: hard})
	}
	baseDir := filepath.Dir(manifestPath)
	rel := func(p string) string {
		if filepath.IsAbs(p) {
			if r, err := filepath.Rel(gaeaCwd(), p); err == nil {
				return filepath.ToSlash(r)
			}
			return p
		}
		return filepath.ToSlash(p)
	}
	// 结构面：空清单/章节号重复/标题重复/路径重复。
	if len(m.Chapters) == 0 {
		add("missing_chapter", "清单无章节——先 set 登记章节再写作", true)
		return out
	}
	seenNo, seenTitle, seenPath := map[int]string{}, map[string]int{}, map[string]int{}
	for _, ch := range m.Chapters {
		if prev, ok := seenNo[ch.No]; ok && ch.No != 0 {
			add("duplicate_chapter", fmt.Sprintf("章节号 %d 重复（%s 与 %s）", ch.No, prev, ch.Title), true)
		}
		seenNo[ch.No] = ch.Title
		if ch.Title != "" {
			if prev, ok := seenTitle[strings.ToLower(ch.Title)]; ok {
				add("duplicate_chapter", fmt.Sprintf("章节标题重复：%q（第 %d 与 %d 章）", ch.Title, prev, ch.No), true)
			}
			seenTitle[strings.ToLower(ch.Title)] = ch.No
		}
		if ch.Path == "" {
			add("missing_chapter", fmt.Sprintf("第 %d 章 %q 未登记正文路径", ch.No, ch.Title), true)
		} else {
			if prev, ok := seenPath[ch.Path]; ok {
				add("duplicate_chapter", fmt.Sprintf("正文路径重复：%s（第 %d 与 %d 章）", ch.Path, prev, ch.No), true)
			}
			seenPath[ch.Path] = ch.No
		}
	}
	// 章节文件面：缺章/空章/占位符/漂移 + 来源悬空。
	for _, ch := range m.Chapters {
		if ch.Path == "" {
			continue
		}
		p := ch.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			add("missing_chapter", fmt.Sprintf("第 %d 章 %q 正文不存在：%s", ch.No, ch.Title, rel(p)), true)
		} else {
			text := strings.TrimSpace(string(b))
			if text == "" {
				add("empty_chapter", fmt.Sprintf("第 %d 章 %q 正文为空文件", ch.No, ch.Title), true)
			} else {
				// 图引面（A3）：整行图引逐条解析，相对路径按章节目录解析；缺失或
				// 外链=硬伤——导出链对本地文件才嵌图，问题在机检层带路径语义暴露，
				// 不再等 python 端（该端已改硬失败）。
				chDir := filepath.Dir(p)
				for _, line := range strings.Split(text, "\n") {
					m := longdocImageLineRe.FindStringSubmatch(strings.TrimSpace(line))
					if m == nil {
						continue
					}
					ref := strings.TrimSpace(m[1])
					if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
						add("image_missing", fmt.Sprintf("第 %d 章 %q 图引 %s 是外链——导出链不拉取网络图，先落盘本地文件（绝对路径或相对本章目录）", ch.No, ch.Title, ref), true)
						continue
					}
					img := ref
					if !filepath.IsAbs(img) {
						img = filepath.Join(chDir, ref)
					}
					if _, serr := os.Stat(img); serr != nil {
						add("image_missing", fmt.Sprintf("第 %d 章 %q 图引文件不存在：%s（图引 %s，相对路径按章节目录 %s 解析；也可写绝对路径）", ch.No, ch.Title, img, ref, chDir), true)
					}
				}
				// 未闭合占位符：模板插值 {{ }} 计数不平衡，或已知占位标记在正文。
				if strings.Count(text, "{{") != strings.Count(text, "}}") {
					add("unclosed_placeholder", fmt.Sprintf("第 %d 章 %q 的 {{ }} 插值不闭合（{{ %d 处 vs }} %d 处）", ch.No, ch.Title, strings.Count(text, "{{"), strings.Count(text, "}}")), true)
				}
				for _, marker := range []string{"TODO", "【待补充】", "【占位", "XXX占位"} {
					if strings.Contains(text, marker) {
						add("placeholder_found", fmt.Sprintf("第 %d 章 %q 含占位标记 %q", ch.No, ch.Title, marker), true)
						break
					}
				}
				// 清单漂移：正文首个一级标题 ≠ 清单标题（正文/清单脱节显形）。
				for _, line := range strings.Split(text, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "# ") {
						h := strings.TrimSpace(strings.TrimPrefix(line, "# "))
						if ch.Title != "" && h != ch.Title {
							add("drift", fmt.Sprintf("第 %d 章清单标题 %q 与正文一级标题 %q 不一致（改结构先改清单）", ch.No, ch.Title, h), true)
						}
						break // 只看首个一级标题
					}
				}
			}
		}
		for _, src := range ch.Sources {
			sp := src
			if !filepath.IsAbs(sp) {
				sp = filepath.Join(baseDir, sp)
			}
			if _, err := os.Stat(sp); err != nil {
				add("dangling_source", fmt.Sprintf("第 %d 章来源文件缺失：%s", ch.No, rel(sp)), true)
			}
		}
	}
	// 评分点映射面：登记的 requirement 没有任何章节映射 = 未覆盖缺口。
	if len(m.Requirements) > 0 {
		mapped := map[string]bool{}
		for _, ch := range m.Chapters {
			for _, id := range ch.Requirements {
				mapped[id] = true
			}
		}
		for _, r := range m.Requirements {
			if !mapped[r.ID] {
				add("unmapped_requirement", fmt.Sprintf("评分点 %s（%s）未被任何章节映射", r.ID, r.Text), true)
			}
		}
		for _, ch := range m.Chapters {
			for _, id := range ch.Requirements {
				known := false
				for _, r := range m.Requirements {
					if r.ID == id {
						known = true
						break
					}
				}
				if !known {
					add("unmapped_requirement", fmt.Sprintf("第 %d 章映射的评分点 %s 未在清单 requirements 登记", ch.No, id), true)
				}
			}
		}
	}
	return out
}

// formatLongdocFindings 机检回执文本（manifest check 与 build 共用）。
func formatLongdocFindings(fs []longdocFinding) (string, int) {
	hard := 0
	var b strings.Builder
	for _, f := range fs {
		tag := "INFO"
		if f.Hard {
			tag = "HARD"
			hard++
		}
		fmt.Fprintf(&b, "- [%s] %s: %s\n", tag, f.Code, f.Detail)
	}
	return b.String(), hard
}

// compactLongdocFindings 硬伤单行摘要（build 拒绝路径用——多行错误会被下游
// 按首行截断，2026-11-21 复盘 B9：拒绝理由「冒号后为空」，模型只能盲调
// longdoc_manifest check 找明细）。
func compactLongdocFindings(fs []longdocFinding) string {
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.Hard {
			parts = append(parts, fmt.Sprintf("[%s] %s", f.Code, f.Detail))
		}
	}
	s := strings.Join(parts, "；")
	if len(s) > 1200 {
		s = s[:1200] + fmt.Sprintf("…（共 %d 项硬伤，全量明细 longdoc_manifest action=check）", len(parts))
	}
	return s
}

// longdocAbsolutizeImages 把整行图引的相对路径改写为按 dir 解析的绝对路径
// （正斜杠形态，python 端开文件两种斜杠都认），返回改写数。绝对路径与外链
// 不动；解析不到的相对路径不改写（机检闸已拦，此处不二次报错）。
func longdocAbsolutizeImages(text *string, dir string) int {
	lines := strings.Split(*text, "\n")
	n := 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		m := longdocImageLineRe.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		ref := strings.TrimSpace(m[1])
		if ref == "" || filepath.IsAbs(ref) || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			continue
		}
		abs := filepath.ToSlash(filepath.Join(dir, ref))
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		lines[i] = strings.Replace(line, "("+m[1]+")", "("+abs+")", 1)
		n++
	}
	if n > 0 {
		*text = strings.Join(lines, "\n")
	}
	return n
}

// longdocCountImageRefs 统计整行图引数（组装后文本；对账基准）。
func longdocCountImageRefs(text string) int {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		if longdocImageLineRe.MatchString(strings.TrimSpace(line)) {
			n++
		}
	}
	return n
}

// longdocCountEmbeddedImages 数 docx 里 word/document.xml 的 <w:drawing 出现
// 次数（每张内嵌图一个 drawing；同一图片文件被引两次是两个 drawing，与媒体
// 文件去重无关——比数 word/media 条目可靠）。
func longdocCountEmbeddedImages(docxPath string) (int, error) {
	r, err := zip.OpenReader(docxPath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, oerr := f.Open()
		if oerr != nil {
			return 0, oerr
		}
		b, rerr := io.ReadAll(rc)
		_ = rc.Close()
		if rerr != nil {
			return 0, rerr
		}
		return strings.Count(string(b), "<w:drawing"), nil
	}
	return 0, fmt.Errorf("docx 内无 word/document.xml：%s", docxPath)
}

// ── longdoc_manifest：清单维护 + 机检 ─────────────────────────────────────

type longdocManifestTool struct{ a *App }

func (t longdocManifestTool) Name() string   { return "longdoc_manifest" }
func (t longdocManifestTool) ReadOnly() bool { return false }
func (t longdocManifestTool) Description() string {
	return "长文方案章节清单维护与确定性机检：init/set 登记章节（标题/正文路径/来源文件/评分点映射/目标字数），check 跑缺章/空章/重复章节/悬空来源/未映射评分点/未闭合占位符/清单漂移/图引缺失八类机检（图引=整行 ![]() 图片引用，相对路径按章节目录解析，缺失或外链=硬伤）。清单是整册的唯一事实源——先清单后正文，改结构先改清单。整册组装用 longdoc_build。"
}
func (t longdocManifestTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action","manifest"],
"properties":{
  "action":{"type":"string","enum":["init","set","get","check"],"description":"init=建空清单（登记标题）；set=整册覆盖写清单（清单是唯一事实源，整册提交）；get=读清单；check=跑确定性机检"},
  "manifest":{"type":"string","description":"清单文件路径（工作区相对，默认放 .gaea/work/longdoc/<文档名>/manifest.json）"},
  "title":{"type":"string","description":"整册标题（init/set）"},
  "requirements":{"type":"array","description":"评分点/硬性要求登记（set）","items":{"type":"object","properties":{"id":{"type":"string"},"text":{"type":"string"}}}},
  "chapters":{"type":"array","description":"章节清单（set 整册覆盖）","items":{"type":"object","properties":{
    "no":{"type":"integer","description":"章节序号"},
    "title":{"type":"string","description":"章节标题（须与正文首个一级标题一致，否则机检报漂移）"},
    "path":{"type":"string","description":"章节正文 md 路径（相对清单目录或绝对路径）"},
    "sources":{"type":"array","items":{"type":"string"},"description":"本章来源文件（悬空来源会被机检拦截）"},
    "requirements":{"type":"array","items":{"type":"string"},"description":"本章映射的评分点 id"},
    "words":{"type":"integer","description":"目标字数"},
    "figs":{"type":"integer","description":"期望图数（0=不校验）"},
    "tables":{"type":"integer","description":"期望表数（0=不校验）"}
  }}}
}
}`)
}

func (t longdocManifestTool) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Manifest string `json:"manifest"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}
	abs, err := longdocResolvePath(p.Manifest)
	if err != nil {
		return nil, err
	}
	return []string{abs}, nil
}

func (t longdocManifestTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action   string `json:"action"`
		Manifest string `json:"manifest"`
		Title    string `json:"title"`
		Reqs     []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"requirements"`
		Chapters []longdocChapter `json:"chapters"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("longdoc_manifest 参数解析失败: %w", err)
	}
	manifestPath, err := longdocResolvePath(p.Manifest)
	if err != nil {
		return "", err
	}
	switch p.Action {
	case "init":
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
			return "", err
		}
		if _, err := os.Stat(manifestPath); err == nil {
			return "", fmt.Errorf("清单已存在（%s）：沿用或 set 覆盖，不要 init 冲掉既有清单", manifestPath)
		}
		m := longdocManifestFile{Title: strings.TrimSpace(p.Title), Chapters: []longdocChapter{}}
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(manifestPath, b, 0o644); err != nil {
			return "", err
		}
		return fmt.Sprintf("清单已建立：%s（title=%q）。下一步 set 登记章节，随后分章写作。", manifestPath, m.Title), nil
	case "set":
		m := longdocManifestFile{
			Title:        strings.TrimSpace(p.Title),
			Chapters:     p.Chapters,
			Requirements: []longdocRequirement{},
		}
		for _, r := range p.Reqs {
			m.Requirements = append(m.Requirements, longdocRequirement{ID: strings.TrimSpace(r.ID), Text: strings.TrimSpace(r.Text)})
		}
		for i := range m.Chapters {
			if m.Chapters[i].Sources == nil {
				m.Chapters[i].Sources = []string{}
			}
			if m.Chapters[i].Requirements == nil {
				m.Chapters[i].Requirements = []string{}
			}
		}
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(manifestPath, b, 0o644); err != nil {
			return "", err
		}
		findings := longdocCheck(&m, manifestPath)
		body, hard := formatLongdocFindings(findings)
		if hard > 0 {
			return fmt.Sprintf("清单已写入（%d 章）但机检发现 %d 项硬伤：\n%s先修复再进入 longdoc_build。", len(m.Chapters), hard, body), nil
		}
		return fmt.Sprintf("清单已写入（%d 章），机检 %d 项硬伤 0。可分章写作；写完再 check 复检后 longdoc_build。", len(m.Chapters), hard), nil
	case "get":
		m, err := loadLongdocManifest(manifestPath)
		if err != nil {
			return "", fmt.Errorf("读取清单失败: %w", err)
		}
		b, _ := json.MarshalIndent(m, "", "  ")
		return fmt.Sprintf("清单（%d 章，评分点 %d 条）：\n%s", len(m.Chapters), len(m.Requirements), b), nil
	case "check":
		m, err := loadLongdocManifest(manifestPath)
		if err != nil {
			return "", fmt.Errorf("读取清单失败: %w", err)
		}
		findings := longdocCheck(m, manifestPath)
		body, hard := formatLongdocFindings(findings)
		quota := longdocQuotaSummary(m, manifestPath) // P2-9：配额面恒附（结构化 JSON 供外挂脚本）
		if hard > 0 {
			return fmt.Sprintf("机检发现 %d 项硬伤：\n%s%s全部修复后复检，绿了才能 longdoc_build。", hard, body, quota), nil
		}
		return fmt.Sprintf("机检全绿（%d 章，硬伤 0）：\n%s%s可 longdoc_build 组装。", len(m.Chapters), body, quota), nil
	default:
		return "", fmt.Errorf("未知 action %q（init/set/get/check）", p.Action)
	}
}

// ── longdoc_build：按清单组装整册 ─────────────────────────────────────────

type longdocBuildTool struct{ a *App }

func (t longdocBuildTool) Name() string   { return "longdoc_build" }
func (t longdocBuildTool) ReadOnly() bool { return false }
func (t longdocBuildTool) Description() string {
	return "按清单组装整册并导出交付物（docx/pdf/md）：构建前强制机检（带硬检问题拒绝组装，拒绝理由单行列明），章节按清单序拼接（整行图引相对路径自动按章节目录绝对化）后走交付模板链（不重写生成器）。docx 回执含图片对账（引用数/嵌入数，缺图即错）与 SHA256（哈希 Go 自算）；页数用 doc_render 渲染取。output_path 为必填声明写路径。模板缺省=标书（宋体正文/黑体黑标题/章另起页/页眉下框线；封面五要素 subtitle=包号/author=投标人/bid_copy/bid_date，页眉缺省取投标人）。"
}
func (t longdocBuildTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["manifest","format","output_path"],
"properties":{
  "manifest":{"type":"string","description":"清单文件路径（longdoc_manifest 维护）"},
  "format":{"type":"string","enum":["docx","pdf","md"],"description":"交付格式"},
  "output_path":{"type":"string","description":"声明写路径（工作区相对路径）；扩展名须与 format 一致"},
  "template":{"type":"string","description":"docx 模板：内置名 通用|公文|报告|合同|标书（缺省 标书：宋体/黑标题/章另起页），或登记格式模板名（工作区 .gaea/templates/docx/<名>.json 优先，其次技能 format_templates/；make_template.py 从格式样例登记）"},
  "cover":{"type":"boolean","description":"生成封面"},
  "toc":{"type":"boolean","description":"生成目录（长文默认建议开）"},
  "subtitle":{"type":"string","description":"标书封面包号（如 第 3 包）"},
  "author":{"type":"string","description":"标书封面投标人；页眉缺省时自动作为页眉"},
  "bid_copy":{"type":"string","enum":["正本","副本"],"description":"标书封面正本/副本标记，缺省 正本"},
  "bid_date":{"type":"string","description":"标书封面落款日期，缺省=今天"},
  "header":{"type":"string","description":"页眉文字（标书带下框线；缺省取投标人）"},
  "footer":{"type":"string","description":"页脚文字，缺省 第 {page} 页"}
}
}`)
}

func (t longdocBuildTool) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Manifest   string `json:"manifest"`
		OutputPath string `json:"output_path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}
	out := []string{}
	if abs, err := longdocResolvePath(p.Manifest); err == nil {
		out = append(out, abs)
	}
	if strings.TrimSpace(p.OutputPath) != "" {
		abs, err := longdocResolvePath(p.OutputPath)
		if err != nil {
			return nil, err
		}
		out = append(out, abs)
		return out, nil
	}
	return nil, fmt.Errorf("缺少写路径参数（output_path）")
}

func (t longdocBuildTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Manifest   string `json:"manifest"`
		Format     string `json:"format"`
		OutputPath string `json:"output_path"`
		Template   string `json:"template"`
		Cover      bool   `json:"cover"`
		TOC        bool   `json:"toc"`
		Subtitle   string `json:"subtitle"`
		Author     string `json:"author"`
		BidCopy    string `json:"bid_copy"`
		BidDate    string `json:"bid_date"`
		Header     string `json:"header"`
		Footer     string `json:"footer"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("longdoc_build 参数解析失败: %w", err)
	}
	if strings.TrimSpace(p.OutputPath) == "" {
		return "", fmt.Errorf("缺少 output_path（声明写路径）：整册交付必须声明落点")
	}
	format := strings.ToLower(strings.TrimPrefix(p.Format, "."))
	switch format {
	case "docx", "pdf", "md":
	default:
		return "", fmt.Errorf("不支持的格式 %q（docx/pdf/md；xlsx 交付走 xlsx 能力，pptx 走 pptx 技能）", format)
	}
	manifestPath, err := longdocResolvePath(p.Manifest)
	if err != nil {
		return "", err
	}
	m, err := loadLongdocManifest(manifestPath)
	if err != nil {
		return "", fmt.Errorf("读取清单失败: %w", err)
	}
	// 构建前强制机检（「清单漂移可检出」断言的执行位）：硬伤即拒组装。
	findings := longdocCheck(m, manifestPath)
	_, hard := formatLongdocFindings(findings)
	if hard > 0 {
		return "", fmt.Errorf("机检发现 %d 项硬伤，拒绝组装：%s（全量明细 longdoc_manifest action=check；修复后复检再 build）", hard, compactLongdocFindings(findings))
	}
	// 按清单序拼接（章节号升序；同号按登记序稳定）。整册标题在册时置顶——
	// 各章自带一级标题，模板按两级标题排版。
	chapters := append([]longdocChapter(nil), m.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool { return chapters[i].No < chapters[j].No })
	baseDir := filepath.Dir(manifestPath)
	var sb strings.Builder
	if t := strings.TrimSpace(m.Title); t != "" {
		sb.WriteString("# " + t + "\n\n")
	}
	for i, ch := range chapters {
		cp := ch.Path
		if !filepath.IsAbs(cp) {
			cp = filepath.Join(baseDir, cp)
		}
		b, err := os.ReadFile(cp)
		if err != nil {
			return "", fmt.Errorf("组装中止：第 %d 章正文读取失败（%s）——清单与磁盘在机检后发生了漂移，请复检", i+1, ch.Path)
		}
		// 图引绝对化（A3）：相对图引按章节目录解析改写——此前按输出目录解析
		// 全部落空。机检已保证引用文件存在，这里只改写不校验。
		text := string(b)
		_ = longdocAbsolutizeImages(&text, filepath.Dir(cp))
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(strings.TrimSpace(text))
	}
	if strings.TrimSpace(sb.String()) == "" {
		return "", fmt.Errorf("整册内容为空：清单无有效章节")
	}
	// 复用交付模板链（GaeaExportDeliverable：create_docx.py 等，不重写生成器）。
	// 模板缺省=标书（2026-11-21 反馈 P0-2：报告主题的微软雅黑直接改变页数口径
	// ——实测同一内容 472→419 页，页数是投标显性指标；标书口径=宋体/黑标题/
	// 章另起页）。封面五要素与页眉参数透传；页眉缺省取投标人/bid_copy 缺省
	// 正本/日期缺省今天——三处缺省单源在 create_docx.py（模板语义归模板）。
	in := ExportDeliverableInput{
		Markdown:   sb.String(),
		Format:     format,
		OutputPath: p.OutputPath,
		Template:   p.Template,
		Cover:      p.Cover,
		TOC:        p.TOC,
		Subtitle:   p.Subtitle,
		Author:     p.Author,
		BidCopy:    p.BidCopy,
		BidDate:    p.BidDate,
		Header:     p.Header,
		Footer:     p.Footer,
	}
	if in.Template == "" {
		in.Template = "标书"
	}
	// 静态目录两遍法（2026-11-21 反馈 P0-2 建议③）：toc=true 且 docx 时不留需要
	// Ctrl+A→F9 的域目录——占位目录出 pass-1 → soffice→PDF + pdftotext 逐页文本
	// 定位每个标题的渲染实测页 → 回填重建。占位与终稿目录行数一致（页码宽度不
	// 改行数），两遍分页稳定。渲染链（LibreOffice/poppler）不可用→回落 TOC 域
	// 并在回执如实标注回落原因，不静默降级。
	tocNote := ""
	if p.TOC && format == "docx" {
		if entries := longdocTocEntries(sb.String()); len(entries) > 0 {
			in.TocEntries = entries
			measured, merr := t.a.longdocMeasureTocPages(in)
			switch {
			case merr != nil:
				in.TocEntries = nil
				tocNote = fmt.Sprintf("；目录=TOC 域回落（静态页码不可用：%v；打开后 Ctrl+A→F9 刷新页码）", merr)
			default:
				in.TocEntries = measured
				tocNote = fmt.Sprintf("；目录=TOC 域+实测缓存（%d 条渲染实测回填；打开即准确页码，WPS/Word 里 F9 或打印预览全联动重排；LibreOffice 口径 Word 打印可能 ±1~2 页）", len(measured))
			}
		}
	}
	res, err := t.a.GaeaExportDeliverable(in)
	if err != nil {
		return "", err
	}
	// SHA256 Go 自算（终案口径：哈希不以渲染链为准）；页数留给 doc_render。
	var hash string
	if outAbs, err := longdocResolvePath(res.Path); err == nil {
		if data, rerr := os.ReadFile(outAbs); rerr == nil {
			sum := sha256.Sum256(data)
			hash = hex.EncodeToString(sum[:])
		}
	}
	// Journal 证据卡（office_generate 同款单点记账）。
	appendOfficeEvidence("longdoc_build", res.Path,
		"按清单组装整册（机检通过后构建）",
		fmt.Sprintf("format=%s 章节=%d size=%d bytes sha256=%s", res.Format, len(chapters), res.Size, shortHash(hash)),
		"", "")
	// 图片对账（C3 准出门槛的执行位）：docx 交付时数 document.xml 的 drawing
	// 与组装文本的图引数——嵌入数 < 引用数 = 交付物缺图，按错误回（文件虽已
	// 产出但不得交付；机检+python 硬失败双闸下到不了这里，到=新缺陷面）。
	imgNote := ""
	if imgRefs := longdocCountImageRefs(sb.String()); imgRefs > 0 {
		if format == "docx" {
			if outAbs, aerr := longdocResolvePath(res.Path); aerr == nil {
				if embedded, derr := longdocCountEmbeddedImages(outAbs); derr == nil {
					if embedded < imgRefs {
						return "", fmt.Errorf("导出后图片对账失败：图引 %d 处仅嵌入 %d 处——交付物 %s 缺图不得交付，修复图源后重新 build", imgRefs, embedded, res.Path)
					}
					imgNote = fmt.Sprintf("；图片对账：引用 %d / 嵌入 %d 一致", imgRefs, embedded)
				} else {
					imgNote = fmt.Sprintf("；图片对账未完成（%v）——交付前抽 doc_render 核图", derr)
				}
			}
		} else {
			imgNote = fmt.Sprintf("；图引 %d 处（%s 渲染核图走 doc_render）", imgRefs, format)
		}
	}
	if hash == "" {
		return fmt.Sprintf("整册已组装导出：%s（%d 章，%d bytes；sha256 未读回）。机检 0 硬伤%s%s；页数用 doc_render 渲染取，交付面板可复核。", res.Path, len(chapters), res.Size, imgNote, tocNote), nil
	}
	return fmt.Sprintf("整册已组装导出：%s（%d 章，%d bytes，sha256=%s）。机检 0 硬伤%s%s；页数用 doc_render 渲染取，交付面板可复核。", res.Path, len(chapters), res.Size, shortHash(hash), imgNote, tocNote), nil
}

// longdocTocHeadingRe 组装文本的 H1/H2 标题行（目录条目源）。
var longdocTocHeadingRe = regexp.MustCompile(`^(#{1,2})\s+(.+)$`)

// longdocTocEntries 从组装文本抽目录条目（跳过围栏代码块内伪标题行）。
func longdocTocEntries(md string) []ExportTocEntry {
	var out []ExportTocEntry
	inFence := false
	for _, line := range strings.Split(md, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := longdocTocHeadingRe.FindStringSubmatch(s); m != nil {
			out = append(out, ExportTocEntry{Text: strings.TrimSpace(m[2]), Level: len(m[1]), Page: "—"})
		}
	}
	return out
}

// longdocMeasureTocPages 两遍法的量页腿（office 薄壳）：导出/路径解析走本板
// 链，量页核心 measureTocEntries 板间共用（图文 wenxue_book_build 同款复用）。
func (a *App) longdocMeasureTocPages(in ExportDeliverableInput) ([]ExportTocEntry, error) {
	return measureTocEntries(in, a.GaeaExportDeliverable, longdocResolvePath)
}

// 量页 PDF 腿 seam（2026-11-24 裸题名兜底回归钉用；生产=docmd 真链）。
var (
	longdocConvertToPdf = docmd.ConvertToPdfFile
	longdocPDFPageTexts = docmd.PDFPageTexts
)

// measureTocEntries 量页核心（板间共用）：按占位目录出 pass-1 docx → soffice 转
// PDF → pdftotext 逐页文本 → 每条目标题定位首个含它的页（空白全剥的归一口径，
// PDF 内换行/字距不影响匹配）。匹配不到的条目页码保持「—」（如实可见，不猜）。
// 任何一步失败返回 error，调用方回落 TOC 域。
func measureTocEntries(in ExportDeliverableInput,
	export func(ExportDeliverableInput) (ExportDeliverableResult, error),
	resolve func(string) (string, error)) ([]ExportTocEntry, error) {
	res, err := export(in)
	if err != nil {
		return nil, fmt.Errorf("pass-1 导出失败: %w", err)
	}
	abs, err := resolve(res.Path)
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "gaea-toc-measure-*.pdf")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := longdocConvertToPdf(abs, tmpPath); err != nil {
		return nil, err
	}
	pages, err := longdocPDFPageTexts(tmpPath)
	if err != nil {
		return nil, err
	}
	normPages := make([]string, len(pages))
	for i, p := range pages {
		normPages[i] = longdocNormForMatch(p)
	}
	entries := append([]ExportTocEntry(nil), in.TocEntries...)
	for i := range entries {
		key := longdocNormForMatch(entries[i].Text)
		if key == "" {
			continue
		}
		matched := false
		for pi, np := range normPages {
			if strings.Contains(np, key) {
				entries[i].Page = fmt.Sprintf("%d", pi+1)
				matched = true
				break
			}
		}
		// 编号联动兜底（2026-11-24）：docx 标题编号改样式层多级列表后，渲染
		// 文本里的编号由引擎生成，md 字面前缀异写（第一章 vs 第1章）即字面
		// key 不中——退裸题名再试，题名是编号无关的稳定锚。
		if !matched {
			if bare := longdocNormForMatch(longdocStripHeadingNumber(entries[i].Text)); bare != "" && bare != key {
				for pi, np := range normPages {
					if strings.Contains(np, bare) {
						entries[i].Page = fmt.Sprintf("%d", pi+1)
						break
					}
				}
			}
		}
	}
	return entries, nil
}

// longdocNormForMatch 标题↔页文本匹配归一：剥全部空白（PDF 抽取的换行/字距
// 与 markdown 原文不可逐字节比对）。
func longdocNormForMatch(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f', '　', 0x00A0, 0x200B:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// longdocHeadingPrefixRes 标题字面编号前缀（2026-11-24 编号联动：docx 标题
// 编号改样式层多级列表后，渲染文本里的编号由引擎生成，可能与 md 字面前缀
// 异写——对账用裸题名兜底。与 create_docx.py HEADING_PREFIX_RES 同源镜像，
// 两处同批改）。裸数字空格层（「1 概述」）不走正则：Go RE2 无先行断言，
// 与 python 侧同款「剥后看首字」过滤（见 longdocBareNumKeep）。
var longdocHeadingPrefixRes = []*regexp.Regexp{
	regexp.MustCompile(`^第\s*[0-9０-９一二三四五六七八九十百]+\s*[章篇部]\s*`),
	regexp.MustCompile(`^[0-9０-９]+(?:\s*[.、]\s*[0-9０-９]+)+\s+`),
	regexp.MustCompile(`^[0-9０-９]+(?:\s*[.、]\s*[0-9０-９]+)*[、]\s*`),
	regexp.MustCompile(`^[（(]\s*[0-9０-９一二三四五六七八九十]+\s*[)）]\s*`),
	regexp.MustCompile(`^[一二三四五六七八九十百]+\s*[、.]\s*`),
}

var longdocBareNumRe = regexp.MustCompile(`^[0-9０-９]{1,3}\s+`)

// longdocBareNumKeep 裸数字后接这些字=日期/序次语义（「2026 年度」「12 月进度」），
// 不剥——数字开头的非编号标题。
var longdocBareNumKeep = func() map[rune]bool {
	m := map[rune]bool{}
	for _, r := range "年月日号度第周" {
		m[r] = true
	}
	return m
}()

// longdocStripHeadingNumber 剥标题字面编号前缀（最多两轮，吃「第1章 1.1」），
// 只留纯题名。对账专用：markdown 源文件本身不动。
func longdocStripHeadingNumber(s string) string {
	for i := 0; i < 2; i++ {
		t := s
		matched := false
		for _, rx := range longdocHeadingPrefixRes {
			if loc := rx.FindStringIndex(t); loc != nil {
				t = t[loc[1]:]
				matched = true
				break
			}
		}
		if !matched {
			if loc := longdocBareNumRe.FindStringIndex(t); loc != nil {
				rest := t[loc[1]:]
				if rest != "" {
					r, _ := utf8.DecodeRuneInString(rest)
					if !longdocBareNumKeep[r] {
						t = rest
						matched = true
					}
				}
			}
		}
		if !matched {
			break
		}
		s = t
	}
	return strings.TrimSpace(s)
}

// longdocQuotaSummary 配额面（2026-11-21 反馈 P2-9：check 八类机检不比对字数
// 配额与图表数，验收要自写四套脚本）。逐章实测 汉字数/图题注数/表题注数 vs
// 清单目标：words 偏差>30%（figs/tables 期望>0 时不等）出 warn 级提示（配额
// 是目标不是硬约束，不拦组装）；恒附结构化 JSON 行供外挂脚本消费。
func longdocQuotaSummary(m *longdocManifestFile, manifestPath string) string {
	type chQuota struct {
		No      int    `json:"no"`
		Title   string `json:"title"`
		Words   int    `json:"words"`
		Target  int    `json:"words_target,omitempty"`
		Dev     int    `json:"dev_pct,omitempty"`
		Figs    int    `json:"figs"`
		FigExp  int    `json:"figs_expect,omitempty"`
		Tables  int    `json:"tables"`
		TablExp int    `json:"tables_expect,omitempty"`
	}
	baseDir := filepath.Dir(manifestPath)
	var rows []chQuota
	var warns []string
	for _, ch := range m.Chapters {
		q := chQuota{No: ch.No, Title: ch.Title, Target: ch.Words, FigExp: ch.Figs, TablExp: ch.Tables}
		p := ch.Path
		if p != "" && !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		if b, err := os.ReadFile(p); err == nil {
			text := string(b)
			q.Words = len(ldvHanRe.FindAllString(text, -1))
			for _, ln := range strings.Split(text, "\n") {
				if mm := ldvCaptionRe.FindStringSubmatch(strings.TrimSpace(ln)); mm != nil {
					if mm[1] == "图" {
						q.Figs++
					} else {
						q.Tables++
					}
				}
			}
			if q.Target > 0 {
				q.Dev = (q.Words - q.Target) * 100 / q.Target
				if q.Dev > 30 || q.Dev < -30 {
					warns = append(warns, fmt.Sprintf("第 %d 章字数偏差 %d%%（目标 %d，实际 %d）——配额超差，复核篇幅规划", ch.No, q.Dev, q.Target, q.Words))
				}
			}
			if ch.Figs > 0 && q.Figs != ch.Figs {
				warns = append(warns, fmt.Sprintf("第 %d 章图数期望 %d，实际题注 %d", ch.No, ch.Figs, q.Figs))
			}
			if ch.Tables > 0 && q.Tables != ch.Tables {
				warns = append(warns, fmt.Sprintf("第 %d 章表数期望 %d，实际题注 %d", ch.No, ch.Tables, q.Tables))
			}
		}
		rows = append(rows, q)
	}
	if len(rows) == 0 {
		return ""
	}
	b, _ := json.Marshal(rows)
	var sb strings.Builder
	if len(warns) > 0 {
		sb.WriteString("配额 warn（不拦组装）：\n")
		for _, w := range warns {
			sb.WriteString("  - " + w + "\n")
		}
	}
	fmt.Fprintf(&sb, "配额摘要 JSON：%s\n", b)
	return sb.String()
}

// shortHash 哈希展示截断（回执可读；全值可由文件复算）。
func shortHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "…"
	}
	return h
}
