package app

// wenxue_tools.go — 图文专属 agent 工具四族（2026-11-15 刀3）。
//
// 章节 / 角色 / 时间线 / 统计，命名 wenxue_* 前缀，仅装配进图文独立引擎
//（wenxueExtraTools，办公引擎 ExtraTools 不含 → 办公会话零出现）。
// 目录约定见 wenxue-workbench 技能（宽容匹配中文/英文目录名）：
//   章节/|chapters/、角色/|characters/、时间线/|timeline/、设定/|worldview/。
// 路径安全：全部写读经 wspath.ResolveRelWithin（工作区内解析，拒 .. 穿越）；
// 写侧带 overwrite 显式闸（缺省 false，已存在即拒——宁拒不误改）。
// 契约行：RegistryMap 十五行（extra-app/work——刀3 原九件+定稿刀三件+角色库
// 导入刀一件+候选稿两件；2026-11-15 众神殿首刀收口批改口径归一），装配面守卫=
// wenxue_surface_probe_test.go census 三例。

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// ── 工具骨架 ────────────────────────────────────────────────────

// wenxueTool 图文专属工具通用骨架：自声明 work 空间（SpaceTaggedTool），
// exec 在 Execute 时解析当前小说工作空间（切换工作空间即新锚）。
type wenxueTool struct {
	name     string
	desc     string
	schema   string
	readOnly bool
	run      func(args json.RawMessage) (string, error)
}

func (t *wenxueTool) Name() string        { return t.name }
func (t *wenxueTool) Description() string { return t.desc }
func (t *wenxueTool) Schema() json.RawMessage {
	return json.RawMessage(t.schema)
}
func (t *wenxueTool) ReadOnly() bool { return t.readOnly }
func (t *wenxueTool) SpaceTag() string {
	return "work" // 图文板块 work 锚（与其余 work 工具同空间装配语义）
}
func (t *wenxueTool) CachePaths(json.RawMessage) []string {
	if t.readOnly {
		return nil
	}
	if root := strings.TrimSpace(wenxueWorkspace()); root != "" {
		// 图文写族可能同时改动章节/角色/时间线；按整本工作区失效最稳妥，
		// 且每回合缓存规模很小。
		return []string{root}
	}
	return nil
}
func (t *wenxueTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	return t.run(args)
}

// wenxueExtraTools 图文引擎专属工具装配单点（只进 wx boot.Build 的 ExtraTools）。
func wenxueExtraTools(a *App) []tool.Tool {
	_ = a // 预留：后续需要应用服务（如模型回调）时经 a 取
	return []tool.Tool{
		wenxueChapterListTool(),
		wenxueChapterReadTool(),
		wenxueChapterWriteTool(),
		wenxueCandidateWriteTool(),
		wenxueCandidatePromoteTool(),
		wenxueCharacterListTool(a),
		wenxueCharacterReadTool(),
		wenxueCharacterWriteTool(),
		wenxueCharacterImportTool(a),
		wenxueTimelineReadTool(),
		wenxueTimelineAppendTool(),
		wenxueStatsTool(),
		wenxueQualityCheckTool(),
		wenxueFinalListTool(),
		wenxueFinalReadTool(),
		wenxueFinalWriteTool(),
	}
}

// wxWorkSpaceTool 图文装配空间适配器：只改写 SpaceTag，Execute 原样委托。
// image_gen 在 spacetags 名表是 play 域（办公/通用语义不动），图文（work
// 空间）装配期会被物理过滤——众神殿媒体根分档裁决「图文会话生成图锚小说
// 目录」的意图落空（2026-11-16 实测：装配面缺 image_gen；件数不写数字，
// 装配面以 census 守卫测试为准）。本适配器仅在
// 图文装配单点把实例标成 work；gaeaExtraTools 原样（办公 play 语义不变）。
type wxWorkSpaceTool struct{ tool.Tool }

func (t wxWorkSpaceTool) SpaceTag() string { return "work" }

// wenxueBaseTools 图文引擎通用 extra 工具单点（= gaeaExtraTools 媒体根分档版，
// 2026-11-15 众神殿首刀，pan_1794754879290 裁决）：image_gen/ocr 媒体根注入
// wenxueWorkspace——图文会话生成图与 OCR 相对路径锚当前小说目录，不再落办公
// 工作区；vision 不动（builtin-workspace 随 Build Cwd 注入，本就正确）。
// image_gen 经 wxWorkSpaceTool 标 work 过空间闸（见上）。
// 办公装配走 gaeaExtraTools 原样（nil 缝=gaeaCwd 现状逐字节，红线）。
func wenxueBaseTools(a *App) []tool.Tool {
	tools := extraToolsWithMediaRoot(a, wenxueWorkspace)
	for i, t := range tools {
		if t.Name() == "image_gen" {
			tools[i] = wxWorkSpaceTool{Tool: t}
			break
		}
	}
	return tools
}

// ── 目录约定与共享 helper ────────────────────────────────────────

// wenxueDirCandidates 目录约定宽容匹配（中/英文目录名，按优先序）。
func wenxueDirCandidates(root, kind string) []string {
	switch kind {
	case "chapters":
		return []string{"章节", "chapters", "正文"}
	case "characters":
		return []string{"角色", "characters", "人物"}
	case "timeline":
		return []string{"时间线", "timeline"}
	case "settings":
		return []string{"设定", "worldview", "设定集"}
	case "final":
		// 定稿域（2026-11-15 阅读刀）：只放发布稿不放草稿；agent 经
		// wenxue_final_* 读写，阅读标签页据此生成目录与阅读窗。
		return []string{"定稿", "final", "finalized"}
	}
	return nil
}

// wenxueFindDir 返回工作区内第一个存在的约定目录（相对路径；无则空串）。
func wenxueFindDir(root, kind string) string {
	for _, name := range wenxueDirCandidates(root, kind) {
		rel := filepath.ToSlash(name)
		if abs, err := wspath.ResolveRelWithin(root, rel); err == nil {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				return rel
			}
		}
	}
	return ""
}

// wenxueEnsureDir 取约定目录（无则建首选中文名），返回工作区相对路径。
func wenxueEnsureDir(root, kind string) string {
	if rel := wenxueFindDir(root, kind); rel != "" {
		return rel
	}
	rel := filepath.ToSlash(wenxueDirCandidates(root, kind)[0])
	abs, err := wspath.ResolveRelWithin(root, rel)
	if err == nil {
		_ = os.MkdirAll(abs, 0o755)
	}
	return rel
}

// wenxueCountWords 中文语感字数：非空白 rune 计数。
func wenxueCountWords(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

const wenxueWordCountRule = "字数=Unicode 非空白字符数（rune；按写入文件全文，标题若在 content 中则计入；含标点，不含空格/换行/制表符）；contentChars=全文含空白的 Unicode 字符数"

// wenxueIsUnderFinalDir 报告 rel 是否落在定稿域目录内（域隔离判据：章节
// 工具不写定稿、定稿工具只写定稿——草稿/定稿分离由工具域强制）。
func wenxueIsUnderFinalDir(root, rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	rel = strings.TrimSuffix(rel, "/") + "/"
	for _, cand := range wenxueDirCandidates(root, "final") {
		if strings.HasPrefix(rel, cand+"/") || rel == cand+"/" {
			return true
		}
	}
	return false
}

// wenxueIsUnderCandidateDir 候选稿域（<章节目录>/候选/）判据——chapter_write
// 对称定稿域闸（2026-11-17 审查批3）：否则 overwrite=true 可覆写候选文件，
// 绕过 wenxue_candidate_write 的 overwrite 闸。
func wenxueIsUnderCandidateDir(root, rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	rel = strings.TrimSuffix(rel, "/") + "/"
	for _, cand := range wenxueDirCandidates(root, "chapters") {
		if strings.HasPrefix(rel, cand+"/候选/") {
			return true
		}
	}
	return false
}

// wenxueChapterMeta 解析章节文件名：前导数字=章号，其后分隔符后=标题。
type wenxueChapterMeta struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Words   int    `json:"words"`
	ModTime int64  `json:"modTime"`
}

// wenxueStemNumber 从文件名 stem 提取章号与标题余部：前导 ASCII 数字=章号，
// 其后分隔符剥离=标题；无前导数字 → 章号 0、标题=整个 stem（ok=false）。
// 共享章号 helper（2026-11-15 众神殿首刀）：章节文件解析与告警单一来源——
// 中文序号（第一章.md）不可解析，落 Number=0，由 wenxueChapterWarnings 显式告警。
func wenxueStemNumber(stem string) (number int, title string, ok bool) {
	i := 0
	for i < len(stem) && stem[i] >= '0' && stem[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, strings.TrimSpace(stem), false
	}
	n, err := strconv.Atoi(stem[:i])
	if err != nil {
		return 0, strings.TrimSpace(stem), false
	}
	return n, strings.TrimSpace(strings.TrimLeft(stem[i:], "-_ .：:")), true
}

// wenxueParseChapterFile 解析单个章节文件（缺文件/目录/非法路径返回 ok=false）。
func wenxueParseChapterFile(root, rel string) (wenxueChapterMeta, bool) {
	abs, err := wspath.ResolveRelWithin(root, rel)
	if err != nil {
		return wenxueChapterMeta{}, false
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return wenxueChapterMeta{}, false
	}
	ext := strings.ToLower(filepath.Ext(abs))
	if ext != ".md" && ext != ".markdown" && ext != ".txt" {
		return wenxueChapterMeta{}, false
	}
	meta := wenxueChapterMeta{Path: filepath.ToSlash(rel), Name: filepath.Base(abs), ModTime: info.ModTime().UnixMilli()}
	stem := strings.TrimSuffix(meta.Name, filepath.Ext(meta.Name))
	meta.Number, meta.Title, _ = wenxueStemNumber(stem)
	if b, err := os.ReadFile(abs); err == nil {
		meta.Words = wenxueCountWords(string(b))
	}
	return meta, true
}

// wenxueListChaptersWithFallback 枚举约定章节目录全部章节（无约定目录时回退
// 工作区根一层的 md/txt），按 章号→文件名 排序；第二返回值报告是否走了根
// 目录回退（wenxue_chapter_list 告警判据）。
func wenxueListChaptersWithFallback(root string) ([]wenxueChapterMeta, bool) {
	out := []wenxueChapterMeta{}
	dir := wenxueFindDir(root, "chapters")
	fellBack := dir == ""
	scanDirs := []string{}
	if dir != "" {
		scanDirs = append(scanDirs, dir)
	} else {
		scanDirs = append(scanDirs, ".")
	}
	for _, d := range scanDirs {
		abs, err := wspath.ResolveRelWithin(root, d)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			rel := e.Name()
			if d != "." {
				rel = d + "/" + e.Name()
			}
			if m, ok := wenxueParseChapterFile(root, rel); ok {
				out = append(out, m)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].Name < out[j].Name
	})
	return out, fellBack
}

// wenxueListChapters 枚举章节（不关心回退判据的调用面）。
func wenxueListChapters(root string) []wenxueChapterMeta {
	out, _ := wenxueListChaptersWithFallback(root)
	return out
}

// wenxueChapterWarnings 章节清单窄告警（2026-11-15 众神殿首刀，列表级两条
// 上限、不逐文件重复；纯函数，挂 wenxue_chapter_list 回执 warnings 字段）：
// ① 存在章号未解析文件（前导无阿拉伯数字，含「第一章.md」式中文序号）→ 一条；
// ② 无约定目录走了根目录回退 → 一条。
func wenxueChapterWarnings(chapters []wenxueChapterMeta, fellBack bool) []string {
	var out []string
	unnumbered := 0
	for _, c := range chapters {
		if c.Number == 0 {
			unnumbered++
		}
	}
	if unnumbered > 0 {
		out = append(out, fmt.Sprintf("有 %d 个章节文件章号未解析（文件名需以阿拉伯数字开头，如 001-标题.md；中文序号如「第一章」不可解析）；这些文件按文件名排序", unnumbered))
	}
	if fellBack {
		out = append(out, "未找到约定章节目录（章节/chapters/正文），已回退扫描工作区根一层")
	}
	return out
}

// wenxueJSON 统一 JSON 回执（工具结果文本形态）。
func wenxueJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// wenxueReadBody 读工作区文件全文（非法路径/缺文件返回错误）。会话件围栏：
// .gaea/sessions/ 子树结构化拒绝（历史会话走 @会话/session_search；同
// builtin 会话件围栏 2026-11-16）。
func wenxueReadBody(root, rel string) (string, os.FileInfo, error) {
	abs, err := wspath.ResolveRelWithin(root, rel)
	if err != nil {
		return "", nil, fmt.Errorf("非法工作区相对路径: %s", rel)
	}
	if builtin.SessionDirFenceRule(abs) {
		return "", nil, builtin.SessionDirDenied("图文读族工具", rel)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", nil, fmt.Errorf("文件不存在: %s", rel)
	}
	if info.IsDir() {
		return "", nil, fmt.Errorf("目标是目录: %s", rel)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, err
	}
	return string(b), info, nil
}

func wenxueRevision(body string) string {
	sum := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%x", sum[:])
}

// wenxueWriteFence 写侧会话件围栏（2026-11-17 审查批2 刀2，读侧 wenxueReadBody
// 同源对称面）：写族自由 path 参数不得落 .gaea/sessions/ 内部事件流。
// 候选/提升两工具路径由章节清单派生（有界）不过此闸。
func wenxueWriteFence(toolName, abs string) error {
	if builtin.SessionDirFenceRule(abs) {
		return builtin.SessionDirDeniedWrite(toolName, abs)
	}
	return nil
}

func wenxueLocateChapter(root, rawPath string, number int) (wenxueChapterMeta, string, error) {
	rel := filepath.ToSlash(strings.TrimSpace(rawPath))
	for _, c := range wenxueListChapters(root) {
		if (rel != "" && c.Path == rel) || (rel == "" && number > 0 && c.Number == number) {
			body, _, err := wenxueReadBody(root, c.Path)
			return c, body, err
		}
	}
	return wenxueChapterMeta{}, "", fmt.Errorf("未定位到章节（path=%q number=%d）；先用 wenxue_chapter_list 查看清单", rawPath, number)
}

// ── 章节族 ──────────────────────────────────────────────────────

func wenxueChapterListTool() *wenxueTool {
	return &wenxueTool{
		name: "wenxue_chapter_list",
		desc: "列出当前小说全部章节（章号/标题/字数/修改时间，按序号排序）。写作前用它把握全局进度。",
		// schema 无参数（众神殿首刀删假 dir：执行体从不解析输入目录，清单
		// 恒按约定目录自动探测；回执 dir 字段=探测结果）。
		schema:   `{"type":"object","additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			chapters, fellBack := wenxueListChaptersWithFallback(root)
			total := 0
			for _, c := range chapters {
				total += c.Words
			}
			out := map[string]any{
				"dir":           wenxueFindDir(root, "chapters"),
				"count":         len(chapters),
				"totalWords":    total,
				"chapters":      chapters,
				"wordCountRule": wenxueWordCountRule,
			}
			if w := wenxueChapterWarnings(chapters, fellBack); len(w) > 0 {
				out["warnings"] = w // 仅告警非空时携带（低影响契约）
			}
			return wenxueJSON(out)
		},
	}
}

func wenxueChapterReadTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_chapter_read",
		desc:     "读一章正文：内容+章号/标题/字数+上一章结尾（衔接用）。按 path 或 number 定位。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"章节文件工作区相对路径（优先）"},"number":{"type":"integer","description":"章号（path 缺省时按号定位）"},"prevTail":{"type":"integer","description":"附上一章结尾字符数，缺省 200"}},"additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path     string `json:"path"`
				Number   int    `json:"number"`
				PrevTail int    `json:"prevTail"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			root := wenxueWorkspace()
			chapters := wenxueListChapters(root)
			idx := -1
			if in.Path != "" {
				for i, c := range chapters {
					if c.Path == filepath.ToSlash(strings.TrimSpace(in.Path)) {
						idx = i
						break
					}
				}
			} else if in.Number > 0 {
				for i, c := range chapters {
					if c.Number == in.Number {
						idx = i
						break
					}
				}
			}
			if idx < 0 {
				return "", fmt.Errorf("未定位到章节（path=%q number=%d）；先用 wenxue_chapter_list 查看清单", in.Path, in.Number)
			}
			body, _, err := wenxueReadBody(root, chapters[idx].Path)
			if err != nil {
				return "", err
			}
			out := map[string]any{"chapter": chapters[idx], "content": body}
			if in.PrevTail <= 0 {
				in.PrevTail = 200
			}
			if idx > 0 {
				prevBody, _, err := wenxueReadBody(root, chapters[idx-1].Path)
				if err == nil {
					r := []rune(prevBody)
					if len(r) > in.PrevTail {
						r = r[len(r)-in.PrevTail:]
					}
					out["prevTail"] = map[string]any{"chapter": chapters[idx-1].Number, "title": chapters[idx-1].Title, "text": string(r)}
				}
			}
			return wenxueJSON(out)
		},
	}
}

func wenxueChapterWriteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_chapter_write",
		desc:     "写/建一章正文：全量覆盖或新建。overwrite=false 时目标已存在即拒（宁拒不误改）；回执带字数与前后章衔接提示。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"目标文件工作区相对路径（缺省按 number+title 在约定章节目录生成 序号-标题.md）"},"number":{"type":"integer","description":"章号（用于定位/命名）"},"title":{"type":"string","description":"章节标题（新建命名用）"},"content":{"type":"string","description":"章节正文全文"},"overwrite":{"type":"boolean","description":"允许覆盖已存在文件，缺省 false"}},"required":["content"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path      string `json:"path"`
				Number    int    `json:"number"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				Overwrite bool   `json:"overwrite"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			if strings.TrimSpace(in.Content) == "" {
				return "", fmt.Errorf("content 为空（清稿请明确 overwrite=true 且 content 给出占位说明）")
			}
			root := wenxueWorkspace()
			rel := filepath.ToSlash(strings.TrimSpace(in.Path))
			if rel == "" {
				dir := wenxueEnsureDir(root, "chapters")
				num := in.Number
				title := strings.TrimSpace(in.Title)
				if title == "" {
					if num <= 0 {
						return "", fmt.Errorf("path 缺省时需提供 number 或 title（否则章节文件无法命名）")
					}
					title = "未命名"
				}
				// 同章号已有文件→沿用其文件名（保住原命名）；否则生成 序号-标题.md
				if num > 0 {
					for _, c := range wenxueListChapters(root) {
						if c.Number == num {
							rel = c.Path
							break
						}
					}
				}
				if rel == "" {
					rel = fmt.Sprintf("%s/%03d-%s.md", dir, num, sanitizeFileName(title))
				}
			}
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", fmt.Errorf("非法工作区相对路径: %s", rel)
			}
			if ferr := wenxueWriteFence("wenxue_chapter_write", abs); ferr != nil {
				return "", ferr
			}
			if wenxueIsUnderFinalDir(root, rel) {
				return "", fmt.Errorf("定稿目录（%s）请用 wenxue_final_write——章节工具不写定稿，防草稿入库", wenxueFindDir(root, "final"))
			}
			if wenxueIsUnderCandidateDir(root, rel) {
				return "", fmt.Errorf("候选稿域（章节/候选/）请用 wenxue_candidate_write——章节工具不写候选，防绕过候选 overwrite 闸")
			}
			if ext := strings.ToLower(filepath.Ext(abs)); ext != ".md" && ext != ".markdown" && ext != ".txt" {
				return "", fmt.Errorf("章节正文仅支持 md/markdown/txt，收到 %s", ext)
			}
			if info, err := os.Stat(abs); err == nil && !in.Overwrite {
				return "", fmt.Errorf("目标已存在（%d 字节，%s）；确认覆盖请传 overwrite=true", info.Size(), info.ModTime().Format("15:04:05"))
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			prevWords := 0
			if b, err := os.ReadFile(abs); err == nil {
				prevWords = wenxueCountWords(string(b))
			}
			if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
				return "", err
			}
			meta, _ := wenxueParseChapterFile(root, rel)
			chapters := wenxueListChapters(root)
			next := map[string]any{}
			prev := map[string]any{}
			for i, c := range chapters {
				if c.Path != meta.Path {
					continue
				}
				if i > 0 {
					prev["path"] = chapters[i-1].Path
					prev["number"] = chapters[i-1].Number
					prev["title"] = chapters[i-1].Title
				}
				if i+1 < len(chapters) {
					next["path"] = chapters[i+1].Path
					next["number"] = chapters[i+1].Number
					next["title"] = chapters[i+1].Title
				}
				break
			}
			return wenxueJSON(map[string]any{
				"saved": meta, "words": meta.Words, "prevWords": prevWords,
				"delta": meta.Words - prevWords, "prev": prev, "next": next,
				"contentChars": utf8.RuneCountInString(in.Content), "wordCountRule": wenxueWordCountRule,
				"hint": "回写配套：更新相关角色卡「当前状态」与时间线事件（wenxue_character_write / wenxue_timeline_append）",
			})
		},
	}
}

func wenxueCandidateDir(root, chapterPath string) (string, error) {
	if wenxueIsUnderFinalDir(root, chapterPath) {
		return "", fmt.Errorf("候选稿只服务章节草稿，不能基于定稿目录生成")
	}
	stem := strings.TrimSuffix(filepath.Base(chapterPath), filepath.Ext(chapterPath))
	rel := filepath.ToSlash(filepath.Join(filepath.Dir(chapterPath), "候选", stem))
	if _, err := wspath.ResolveRelWithin(root, rel); err != nil {
		return "", err
	}
	return rel, nil
}

func wenxueCandidateWriteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_candidate_write",
		desc:     "把同章候选稿写入 章节/候选/<章名>/ 下，供 A/B/C 方案比较；目标章须已在章节清单（新章先 wenxue_chapter_write 落正文占位）；不覆盖正文、不写定稿。label 建议 A/B/C 或简短方案名。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"目标章节相对路径（优先）"},"number":{"type":"integer","description":"章号（path 缺省时按号定位）"},"label":{"type":"string","description":"候选标签，如 A / B / 冷雨版"},"content":{"type":"string","description":"候选稿全文"},"overwrite":{"type":"boolean","description":"同名候选已存在时允许覆盖，缺省 false"}},"required":["label","content"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path      string `json:"path"`
				Number    int    `json:"number"`
				Label     string `json:"label"`
				Content   string `json:"content"`
				Overwrite bool   `json:"overwrite"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			label := strings.TrimSpace(in.Label)
			if label == "" || strings.TrimSpace(in.Content) == "" {
				return "", fmt.Errorf("label 与 content 都必须非空")
			}
			root := wenxueWorkspace()
			chapter, _, err := wenxueLocateChapter(root, in.Path, in.Number)
			if err != nil {
				return "", err
			}
			dir, err := wenxueCandidateDir(root, chapter.Path)
			if err != nil {
				return "", err
			}
			rel := dir + "/" + sanitizeFileName(label) + ".md"
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", err
			}
			if info, err := os.Stat(abs); err == nil && !in.Overwrite {
				return "", fmt.Errorf("候选已存在（%d 字节，%s）；换 label 或传 overwrite=true", info.Size(), info.ModTime().Format("15:04:05"))
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
				return "", err
			}
			return wenxueJSON(map[string]any{
				"chapter": chapter.Path, "candidate": filepath.ToSlash(rel), "label": label,
				"words": wenxueCountWords(in.Content), "contentChars": utf8.RuneCountInString(in.Content),
				"revision": wenxueRevision(in.Content), "wordCountRule": wenxueWordCountRule,
				"hint": "候选稿不会自动进入正文；用户选定后用 wenxue_candidate_promote 提升",
			})
		},
	}
}

func wenxueCandidatePromoteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_candidate_promote",
		desc:     "把章节目录下的候选稿提升为正文：先自动归档当前正文到同章候选目录，再写入选定候选。不会写定稿。",
		schema:   `{"type":"object","properties":{"candidatePath":{"type":"string","description":"wenxue_candidate_write 返回的候选路径"},"expectedRevision":{"type":"string","description":"可选：候选稿 revision；与当前候选不一致时拒绝"}},"required":["candidatePath"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				CandidatePath    string `json:"candidatePath"`
				ExpectedRevision string `json:"expectedRevision"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			root := wenxueWorkspace()
			candidateRel := filepath.ToSlash(strings.TrimSpace(in.CandidatePath))
			if candidateRel == "" || wenxueIsUnderFinalDir(root, candidateRel) {
				return "", fmt.Errorf("candidatePath 必须位于章节草稿的候选目录")
			}
			ext := strings.ToLower(filepath.Ext(candidateRel))
			if ext != ".md" && ext != ".markdown" && ext != ".txt" {
				return "", fmt.Errorf("候选稿仅支持 md/markdown/txt，收到 %s", ext)
			}
			_, err := wspath.ResolveRelWithin(root, candidateRel)
			if err != nil {
				return "", fmt.Errorf("非法候选路径: %s", candidateRel)
			}
			candidateStem := filepath.Base(filepath.Dir(candidateRel))
			var chapter wenxueChapterMeta
			for _, c := range wenxueListChapters(root) {
				if strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path)) == candidateStem {
					chapter = c
					break
				}
			}
			if chapter.Path == "" {
				return "", fmt.Errorf("候选路径无法关联章节（期望 <章节目录>/候选/<章名>/<候选>.md）")
			}
			dir, err := wenxueCandidateDir(root, chapter.Path)
			if err != nil {
				return "", err
			}
			if !strings.HasPrefix(candidateRel, dir+"/") {
				return "", fmt.Errorf("candidatePath 必须位于 %s/ 下", dir)
			}
			candidateBody, _, err := wenxueReadBody(root, candidateRel)
			if err != nil {
				return "", err
			}
			candidateRevision := wenxueRevision(candidateBody)
			if in.ExpectedRevision != "" && in.ExpectedRevision != candidateRevision {
				return "", fmt.Errorf("REVISION_CONFLICT：候选稿在读取后已变化，请重新确认")
			}
			currentBody, _, err := wenxueReadBody(root, chapter.Path)
			if err != nil {
				return "", err
			}
			archiveRel := fmt.Sprintf("%s/原稿-%s-%s.md", dir, time.Now().Format("20060102-150405"), randomSuffix(3))
			archiveAbs, err := wspath.ResolveRelWithin(root, archiveRel)
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(archiveAbs), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(archiveAbs, []byte(currentBody), 0o644); err != nil {
				return "", err
			}
			chapterAbs, err := wspath.ResolveRelWithin(root, chapter.Path)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(chapterAbs, []byte(candidateBody), 0o644); err != nil {
				return "", err
			}
			saved, _ := wenxueParseChapterFile(root, chapter.Path)
			return wenxueJSON(map[string]any{
				"saved": saved, "candidate": candidateRel, "archivedPrevious": filepath.ToSlash(archiveRel),
				"previousRevision": wenxueRevision(currentBody), "revision": wenxueRevision(candidateBody),
				"wordCountRule": wenxueWordCountRule,
				"hint":          "候选已提升为正文，旧稿已归档为候选；定稿目录未改动",
			})
		},
	}
}

// ── 角色族 ──────────────────────────────────────────────────────

// wenxueListCharacters 枚举角色卡（首行一级标题=角色名；摘要=正文前 80 字）。
func wenxueListCharacters(root string) []map[string]any {
	out := []map[string]any{}
	dir := wenxueFindDir(root, "characters")
	if dir == "" {
		return out
	}
	abs, err := wspath.ResolveRelWithin(root, dir)
	if err != nil {
		return out
	}
	entries, _ := os.ReadDir(abs)
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".md" && ext != ".markdown" && ext != ".txt" {
			continue
		}
		rel := dir + "/" + e.Name()
		body, info, err := wenxueReadBody(root, rel)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if line := wenxueFirstLine(body); strings.HasPrefix(line, "# ") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		r := []rune(strings.TrimSpace(body))
		if len(r) > 80 {
			r = r[:80]
		}
		out = append(out, map[string]any{
			"path": rel, "name": name, "words": wenxueCountWords(body),
			"excerpt": string(r), "modTime": info.ModTime().UnixMilli(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["name"].(string) < out[j]["name"].(string)
	})
	return out
}

func wenxueCharacterListTool(a *App) *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_list",
		desc:     "列出角色卡清单（名字/字数/摘要）+ 角色库同名命中（charlib_hits，可经 wenxue_character_import 全字段导入）。写新章前用它确认出场人物。",
		schema:   `{"type":"object","additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			cards := wenxueListCharacters(root)
			res := map[string]any{"dir": wenxueFindDir(root, "characters"), "count": len(cards), "characters": cards, "wordCountRule": wenxueWordCountRule}
			// 库内同名命中（2026-11-16 导入刀）：提示 agent 库里有现成角色，
			// 不要只拿名字凭空造人。库未初始化/查询失败不拦清单——附带错误字段。
			if a != nil && a.charLib != nil {
				cardNames := map[string]bool{}
				for _, c := range cards {
					if n, _ := c["name"].(string); n != "" {
						cardNames[strings.ToLower(strings.TrimSpace(n))] = true
					}
				}
				items, _, err := a.charLib.List("", "", false, 500, 0)
				if err != nil {
					res["charlib_error"] = err.Error()
				} else {
					hits := []map[string]any{}
					for i := range items {
						c := &items[i]
						if cardNames[strings.ToLower(strings.TrimSpace(c.Name))] {
							hits = append(hits, map[string]any{"name": c.Name, "id": c.ID, "kind": c.Kind})
						}
					}
					res["charlib_hits"] = hits
					if len(hits) > 0 {
						res["charlib_hint"] = "以上角色在角色库有完整档案，建卡/补卡用 wenxue_character_import 全字段导入，不要只按名字重写"
					}
				}
			}
			return wenxueJSON(res)
		},
	}
}

func wenxueCharacterReadTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_read",
		desc:     "读一张角色卡全文。按 path 或 name 定位。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"角色卡相对路径（优先）"},"name":{"type":"string","description":"角色名（path 缺省时按名匹配）"}},"additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path string `json:"path"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			root := wenxueWorkspace()
			rel := filepath.ToSlash(strings.TrimSpace(in.Path))
			if rel == "" {
				for _, c := range wenxueListCharacters(root) {
					if c["name"] == strings.TrimSpace(in.Name) {
						rel = c["path"].(string)
						break
					}
				}
			}
			if rel == "" {
				return "", fmt.Errorf("未定位到角色卡（path=%q name=%q）；先用 wenxue_character_list", in.Path, in.Name)
			}
			body, _, err := wenxueReadBody(root, rel)
			if err != nil {
				return "", err
			}
			return wenxueJSON(map[string]any{"path": rel, "content": body, "words": wenxueCountWords(body), "contentChars": utf8.RuneCountInString(body), "wordCountRule": wenxueWordCountRule})
		},
	}
}

func wenxueCharacterWriteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_write",
		desc:     "建/更新角色卡（全量正文）。建新角色前先用 wenxue_character_import 查角色库——同名命中即全字段导入再本地化，未命中才从零撰写，不要只拿名字凭空造人。overwrite=false 时已存在即拒。写后回执带卡面字数。",
		schema:   `{"type":"object","properties":{"name":{"type":"string","description":"角色名（卡文件名与首行标题）"},"content":{"type":"string","description":"角色卡全文（建议分 设定/外貌/性格/口头禅/关系/当前状态 小节）"},"path":{"type":"string","description":"可选：自定义卡文件相对路径"},"overwrite":{"type":"boolean","description":"允许覆盖已存在卡，缺省 false"}},"required":["name","content"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Name      string `json:"name"`
				Content   string `json:"content"`
				Path      string `json:"path"`
				Overwrite bool   `json:"overwrite"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			name := strings.TrimSpace(in.Name)
			if name == "" || strings.TrimSpace(in.Content) == "" {
				return "", fmt.Errorf("name 与 content 必填")
			}
			root := wenxueWorkspace()
			rel := filepath.ToSlash(strings.TrimSpace(in.Path))
			if rel == "" {
				rel = wenxueEnsureDir(root, "characters") + "/" + sanitizeFileName(name) + ".md"
			}
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", fmt.Errorf("非法工作区相对路径: %s", rel)
			}
			if ferr := wenxueWriteFence("wenxue_character_write", abs); ferr != nil {
				return "", ferr
			}
			if info, err := os.Stat(abs); err == nil && !in.Overwrite {
				return "", fmt.Errorf("角色卡已存在（%s，%d 字节）；确认覆盖请传 overwrite=true", rel, info.Size())
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			content := in.Content
			if !strings.HasPrefix(strings.TrimSpace(content), "# ") {
				content = "# " + name + "\n\n" + content
			}
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return "", err
			}
			return wenxueJSON(map[string]any{
				"path": rel, "name": name, "words": wenxueCountWords(content),
				"contentChars": utf8.RuneCountInString(content), "wordCountRule": wenxueWordCountRule,
				"hint": "章节推进后记得更新「当前状态」小节（再次调用本工具 overwrite=true）",
			})
		},
	}
}

// ── 角色库导入（2026-11-16）────────────────────────────────────
//
// 角色工具此前与角色库零关联：agent 只能拿用户口述的名字凭空造人，库内
// 定位/背景/外貌/性格/动机/台词样例全丢。本刀补 wenxue_character_import——
// 库卡全字段映射六节落盘（卡面契约与前端 WenxueCharacterHub.cardFromLibraryChar
// 同源：设定←定位/标签/背景/动机，外貌←外貌/身材，性格←性格/弧线，
// 口头禅←台词样例，关系留白，当前状态←状态/备注），设定节末记来源 id。
// 同步语义=单向一次性快照：卡面「当前状态」随章节演进、库侧档案独立维护，
// 双向自动合并必打架——库侧更新要进卡须显式重导入（overwrite=true 整卡
// 覆盖，「当前状态」演进内容会丢，工具回执已提示）。库只读不回写（红线）。

// wenxueRoleTypeZh / wenxueCharStatusZh 库字段中文化（与前端 ROLE_TYPE_ZH /
// STATUS_ZH 同源对齐，改两处同批）。
var wenxueRoleTypeZh = map[string]string{
	"protagonist": "主角", "antagonist": "反派", "supporting": "配角", "minor": "龙套",
}

var wenxueCharStatusZh = map[string]string{
	"Alive": "存活", "Dead": "死亡", "Missing": "失踪", "Transformed": "转变",
}

func wenxueZhOf(m map[string]string, key string) string {
	if key == "" {
		return ""
	}
	if v, ok := m[key]; ok {
		return v
	}
	return key // 未知枚举如实透传（前端同口径 ?? c.roleType）
}

// wenxueCardFromLibraryChar 库卡 → 六节角色卡 md（与前端 cardFromLibraryChar
// 逐节同源；追加「来源」行记库 id，供对账与显式重导入）。
func wenxueCardFromLibraryChar(c *characterlib.Character) string {
	meta := []string{wenxueZhOf(wenxueRoleTypeZh, c.RoleType), c.Gender, c.Age}
	kept := meta[:0]
	for _, s := range meta {
		if strings.TrimSpace(s) != "" {
			kept = append(kept, strings.TrimSpace(s))
		}
	}
	setting := []string{}
	if len(kept) > 0 {
		setting = append(setting, "定位："+strings.Join(kept, " · "))
	}
	if len(c.Tags) > 0 {
		setting = append(setting, "标签："+strings.Join(c.Tags, "、"))
	}
	if strings.TrimSpace(c.Background) != "" {
		setting = append(setting, c.Background)
	}
	if strings.TrimSpace(c.Motivation) != "" {
		setting = append(setting, "动机："+c.Motivation)
	}
	setting = append(setting, "来源：角色库（id "+c.ID+"，单向快照不自动同步）")
	personality := []string{}
	if strings.TrimSpace(c.Personality) != "" {
		personality = append(personality, c.Personality)
	}
	if strings.TrimSpace(c.Arc) != "" {
		personality = append(personality, "弧线："+c.Arc)
	}
	status := []string{}
	if zh := wenxueZhOf(wenxueCharStatusZh, c.Status); zh != "" {
		status = append(status, "状态："+zh)
	}
	if strings.TrimSpace(c.Notes) != "" {
		status = append(status, c.Notes)
	}
	samples := make([]string, 0, len(c.DialogueSamples))
	for _, s := range c.DialogueSamples {
		if strings.TrimSpace(s) != "" {
			samples = append(samples, "- "+strings.TrimSpace(s))
		}
	}
	section := func(title, body string) string { return "## " + title + "\n\n" + body }
	bodyOf := func(lines []string, fallback string) string {
		filled := make([]string, 0, len(lines))
		for _, l := range lines {
			if t := strings.TrimSpace(l); t != "" {
				filled = append(filled, t)
			}
		}
		if len(filled) == 0 {
			return fallback
		}
		return strings.Join(filled, "\n")
	}
	samplesBody := "（待补）"
	if len(samples) > 0 {
		samplesBody = strings.Join(samples, "\n")
	}
	parts := []string{
		"# " + c.Name,
		section("设定", bodyOf(setting, "（待补）")),
		section("外貌", bodyOf([]string{c.Appearance, c.Figure}, "（待补）")),
		section("性格", bodyOf(personality, "（待补）")),
		section("口头禅", samplesBody),
		section("关系", "（留待书中发展）"),
		section("当前状态", bodyOf(status, "（待补）")),
	}
	return strings.Join(parts, "\n\n") + "\n"
}

// wenxueCharlibByID / wenxueCharlibByName 角色库精确取卡（import 数据面）。
// name 路径：List LIKE 粗筛后整名唯一匹配；零同名带近似提示，多同名要求 id。
func (a *App) wenxueCharlibByID(id string) (*characterlib.Character, error) {
	if a == nil || a.charLib == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	c, err := a.charLib.Get(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("角色库读取失败: %w", err)
	}
	if c == nil {
		return nil, fmt.Errorf("角色库无 id=%s 的角色", strings.TrimSpace(id))
	}
	return c, nil
}

func (a *App) wenxueCharlibByName(name string) (*characterlib.Character, error) {
	if a == nil || a.charLib == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name 不能为空")
	}
	items, _, err := a.charLib.List(name, "", false, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("角色库查询失败: %w", err)
	}
	var exact []characterlib.Character
	for i := range items {
		if strings.EqualFold(strings.TrimSpace(items[i].Name), name) {
			exact = append(exact, items[i])
		}
	}
	switch len(exact) {
	case 1:
		return &exact[0], nil
	case 0:
		if len(items) == 0 {
			return nil, fmt.Errorf("角色库无「%s」——先在角色面板建库卡，或用 wenxue_character_write 从零撰写", name)
		}
		near := make([]string, 0, min(len(items), 5))
		for i, c := range items {
			if i >= 5 {
				break
			}
			near = append(near, c.Name)
		}
		return nil, fmt.Errorf("角色库无同名「%s」，近似命中：%s——可改传 id 精确导入", name, strings.Join(near, "、"))
	default:
		ids := make([]string, 0, len(exact))
		for _, c := range exact {
			ids = append(ids, c.ID)
		}
		return nil, fmt.Errorf("角色库命中多张同名卡（id: %s）——请传 id 精确导入", strings.Join(ids, "、"))
	}
}

func wenxueCharacterImportTool(a *App) *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_import",
		desc:     "从角色库导入角色为工作空间角色卡（全字段落卡：定位/标签/背景/动机/外貌/性格/弧线/台词样例/状态→六节，设定节记来源 id；库只读不回写）。建新角色前先用本工具查库——同名命中即导入再本地化，未命中才用 wenxue_character_write 从零撰写。",
		schema:   `{"type":"object","properties":{"name":{"type":"string","description":"按名查角色库（须唯一同名命中）"},"id":{"type":"string","description":"角色库精确 id（与 name 二选一，优先级更高）"},"overwrite":{"type":"boolean","description":"允许覆盖已存在卡，缺省 false"}},"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Name      string `json:"name"`
				ID        string `json:"id"`
				Overwrite bool   `json:"overwrite"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			var (
				c   *characterlib.Character
				err error
			)
			if strings.TrimSpace(in.ID) != "" {
				c, err = a.wenxueCharlibByID(in.ID)
			} else {
				c, err = a.wenxueCharlibByName(in.Name)
			}
			if err != nil {
				return "", err
			}
			root := wenxueWorkspace()
			rel := wenxueEnsureDir(root, "characters") + "/" + sanitizeFileName(c.Name) + ".md"
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", fmt.Errorf("非法工作区相对路径: %s", rel)
			}
			if info, err := os.Stat(abs); err == nil && !in.Overwrite {
				return "", fmt.Errorf("角色卡已存在（%s，%d 字节）；确认整卡覆盖请传 overwrite=true（「当前状态」章节演进内容会丢）", rel, info.Size())
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			content := wenxueCardFromLibraryChar(c)
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return "", err
			}
			return wenxueJSON(map[string]any{
				"path": rel, "name": c.Name, "words": wenxueCountWords(content), "source_id": c.ID,
				"contentChars": utf8.RuneCountInString(content), "wordCountRule": wenxueWordCountRule,
				"hint": "单向快照：此后卡面随剧情演进、库侧档案独立；库侧有更新时再次本工具 overwrite=true 可整卡刷新（当前状态会丢，慎用）",
			})
		},
	}
}

// ── 时间线族 ────────────────────────────────────────────────────

// wenxueTimelineEntry 一条时间线事件（章节锚 + 文本 + 来源文件）。
type wenxueTimelineEntry struct {
	Chapter int    `json:"chapter"`
	Text    string `json:"text"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

// wenxueAnchorNumber 从行文本提取「第N章/话」章节锚（ASCII 数字）；无锚 → 0。
// 共享章号 helper（2026-11-15 众神殿首刀）：时间线解析/告警单一来源——中文
// 数字（第十二章）不可解析落 0，由 wenxueTimelineWarnings 显式告警，workbench
// 回写句已钉阿拉伯数字口径。
func wenxueAnchorNumber(text string) int {
	text = strings.TrimSpace(text)
	text = strings.TrimLeft(text, "-•*· ")
	if i := strings.Index(text, "第"); i >= 0 {
		// rune 索引扫描（byte 索引比较多字节 rune 会溢出）
		rs := []rune(text)
		start := len([]rune(text[:i]))
		j := start + 1
		for j < len(rs) && rs[j] >= '0' && rs[j] <= '9' {
			j++
		}
		if j > start+1 && j < len(rs) && (rs[j] == '章' || rs[j] == '话') {
			if n, err := strconv.Atoi(string(rs[start+1 : j])); err == nil {
				return n
			}
		}
	}
	return 0
}

// wenxueParseTimelineLine 从行文本提取章节锚与事件文本（宽容格式）。
func wenxueParseTimelineLine(line string) (int, string) {
	text := strings.TrimSpace(line)
	text = strings.TrimLeft(text, "-•*· ")
	return wenxueAnchorNumber(text), text
}

// wenxueTimelineWarnings 时间线窄告警（2026-11-15 众神殿首刀：聚合一条、附
// 文件:行号封顶 8 处；纯函数，挂 wenxue_timeline_read 回执 warnings 字段）。
func wenxueTimelineWarnings(entries []wenxueTimelineEntry) []string {
	var refs []string
	for _, e := range entries {
		if e.Chapter == 0 {
			refs = append(refs, fmt.Sprintf("%s:%d", e.File, e.Line))
			if len(refs) >= 8 {
				break
			}
		}
	}
	if len(refs) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("有事件缺章号锚点（写法需「第12章 · 一句话事件」，章号用阿拉伯数字）：%s", strings.Join(refs, "、"))}
}

// wenxueTimelineEntries 全量时间线事件（按章节→文件→行序）。
func wenxueTimelineEntries(root string) []wenxueTimelineEntry {
	out := []wenxueTimelineEntry{}
	dir := wenxueFindDir(root, "timeline")
	if dir == "" {
		return out
	}
	abs, err := wspath.ResolveRelWithin(root, dir)
	if err != nil {
		return out
	}
	files, _ := os.ReadDir(abs)
	names := []string{}
	for _, f := range files {
		if !f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext == ".md" || ext == ".markdown" || ext == ".txt" {
				names = append(names, f.Name())
			}
		}
	}
	sort.Strings(names)
	for _, name := range names {
		rel := dir + "/" + name
		body, _, err := wenxueReadBody(root, rel)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				// 空行与 Markdown 标题行（# 时间线）不是事件——2026-11-15
				// 众神殿首刀裁决：跳过，标题不进 events。
				continue
			}
			ch, text := wenxueParseTimelineLine(line)
			out = append(out, wenxueTimelineEntry{Chapter: ch, Text: text, File: rel, Line: i + 1})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Chapter != out[j].Chapter {
			return out[i].Chapter < out[j].Chapter
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

func wenxueTimelineReadTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_timeline_read",
		desc:     "读时间线全部事件（章节序）。开新章/查连续性/排伏笔前先读。",
		schema:   `{"type":"object","properties":{"sinceChapter":{"type":"integer","description":"可选：只返回该章之后（含）的事件"}},"additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				SinceChapter int `json:"sinceChapter"`
			}
			_ = json.Unmarshal(args, &in)
			root := wenxueWorkspace()
			entries := wenxueTimelineEntries(root)
			if in.SinceChapter > 0 {
				filtered := entries[:0:0]
				for _, e := range entries {
					if e.Chapter == 0 || e.Chapter >= in.SinceChapter {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}
			out := map[string]any{"dir": wenxueFindDir(root, "timeline"), "count": len(entries), "events": entries}
			if w := wenxueTimelineWarnings(entries); len(w) > 0 {
				out["warnings"] = w // 仅告警非空时携带（低影响契约）
			}
			return wenxueJSON(out)
		},
	}
}

func wenxueTimelineAppendTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_timeline_append",
		desc:     "追加一条时间线事件（章节锚+一句话）。写完一章的固定回写动作；文件缺省 时间线/时间线.md（自动建）。",
		schema:   `{"type":"object","properties":{"chapter":{"type":"integer","description":"事件发生章节号"},"text":{"type":"string","description":"一句话事件（主语+动作+结果/影响）"},"file":{"type":"string","description":"可选：目标文件相对路径，缺省 时间线/时间线.md"}},"required":["text"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Chapter int    `json:"chapter"`
				Text    string `json:"text"`
				File    string `json:"file"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			text := strings.TrimSpace(in.Text)
			if text == "" {
				return "", fmt.Errorf("text 必填（一句话事件）")
			}
			root := wenxueWorkspace()
			rel := filepath.ToSlash(strings.TrimSpace(in.File))
			if rel == "" {
				dir := wenxueEnsureDir(root, "timeline")
				rel = dir + "/时间线.md"
			}
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", fmt.Errorf("非法工作区相对路径: %s", rel)
			}
			if ferr := wenxueWriteFence("wenxue_timeline_append", abs); ferr != nil {
				return "", ferr
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			line := "- "
			if in.Chapter > 0 {
				line += fmt.Sprintf("第%d章 · ", in.Chapter)
			}
			line += text + "\n"
			f, err := os.OpenFile(abs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return "", err
			}
			_, werr := f.WriteString(line)
			_ = f.Close()
			if werr != nil {
				return "", werr
			}
			out := map[string]any{"appended": strings.TrimRight(line, "\n"), "file": rel}
			if in.Chapter == 0 {
				// 非章节事件（如全书纲要注意）合法保留弹性，但章节事件缺锚会
				// 在按章排序时失位——回执提示而非强制必填（众神殿首刀裁决）。
				out["hint"] = "本章事件应传 chapter（章号锚点用阿拉伯数字，落盘形如「第12章 · 事件」），否则时间线按章排序时无法归位"
			}
			return wenxueJSON(out)
		},
	}
}

// ── 统计族 ──────────────────────────────────────────────────────

func wenxueStatsTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_stats",
		desc:     "小说工作空间统计快照：总字数/章节字数分布/角色卡数/设定文件数/时间线事件数/最近修改。日更复盘与节奏检查用。",
		schema:   `{"type":"object","additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			chapters := wenxueListChapters(root)
			total := 0
			dist := make([]map[string]any, 0, len(chapters))
			var lastMod int64
			for _, c := range chapters {
				total += c.Words
				if c.ModTime > lastMod {
					lastMod = c.ModTime
				}
				dist = append(dist, map[string]any{"number": c.Number, "title": c.Title, "words": c.Words})
			}
			characters := wenxueListCharacters(root)
			timeline := wenxueTimelineEntries(root)
			// 设定面：文件数与字数
			settingFiles, settingWords := 0, 0
			if dir := wenxueFindDir(root, "settings"); dir != "" {
				if abs, err := wspath.ResolveRelWithin(root, dir); err == nil {
					entries, _ := os.ReadDir(abs)
					for _, e := range entries {
						if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
							continue
						}
						rel := dir + "/" + e.Name()
						if body, _, err := wenxueReadBody(root, rel); err == nil {
							settingFiles++
							settingWords += wenxueCountWords(body)
						}
					}
				}
			}
			wsInfo, _ := os.Stat(root)
			var wsMod int64
			if wsInfo != nil {
				wsMod = wsInfo.ModTime().UnixMilli()
			}
			if lastMod == 0 {
				lastMod = wsMod
			}
			return wenxueJSON(map[string]any{
				"workspace":      filepath.Base(root),
				"totalWords":     total,
				"chapterCount":   len(chapters),
				"avgWords":       avgOf(total, len(chapters)),
				"chapters":       dist,
				"characterCards": len(characters),
				"settingFiles":   settingFiles,
				"settingWords":   settingWords,
				"timelineEvents": len(timeline),
				"lastModified":   lastMod,
				"generatedAt":    time.Now().Format(time.RFC3339),
				"wordCountRule":  wenxueWordCountRule,
			})
		},
	}
}

// ── 定稿族（阅读刀 2026-11-15：只放发布稿不放草稿）──────────────

// wenxueListFinalChapters 枚举定稿目录章节（结构同 wenxueListChapters，域不同）。
func wenxueListFinalChapters(root string) []wenxueChapterMeta {
	out := []wenxueChapterMeta{}
	dir := wenxueFindDir(root, "final")
	if dir == "" {
		return out
	}
	abs, err := wspath.ResolveRelWithin(root, dir)
	if err != nil {
		return out
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		rel := dir + "/" + e.Name()
		if m, ok := wenxueParseChapterFile(root, rel); ok {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func wenxueFinalListTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_final_list",
		desc:     "列出定稿目录章节（发布稿，非草稿；章号/标题/字数）。写审校/前情回顾/发布对照前用它。",
		schema:   `{"type":"object","additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			chapters := wenxueListFinalChapters(root)
			total := 0
			for _, c := range chapters {
				total += c.Words
			}
			return wenxueJSON(map[string]any{
				"dir":           wenxueFindDir(root, "final"),
				"count":         len(chapters),
				"totalWords":    total,
				"chapters":      chapters,
				"wordCountRule": wenxueWordCountRule,
			})
		},
	}
}

func wenxueFinalReadTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_final_read",
		desc:     "读一章定稿全文（发布稿口径）。按 path 或 number 定位；审校对照、前情摘要、风格参考用。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"定稿文件工作区相对路径（优先）"},"number":{"type":"integer","description":"章号（path 缺省时按号定位）"}},"additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path   string `json:"path"`
				Number int    `json:"number"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			root := wenxueWorkspace()
			chapters := wenxueListFinalChapters(root)
			idx := -1
			if in.Path != "" {
				for i, c := range chapters {
					if c.Path == filepath.ToSlash(strings.TrimSpace(in.Path)) {
						idx = i
						break
					}
				}
			} else if in.Number > 0 {
				for i, c := range chapters {
					if c.Number == in.Number {
						idx = i
						break
					}
				}
			}
			if idx < 0 {
				return "", fmt.Errorf("未定位到定稿章节（path=%q number=%d）；先用 wenxue_final_list", in.Path, in.Number)
			}
			body, _, err := wenxueReadBody(root, chapters[idx].Path)
			if err != nil {
				return "", err
			}
			out := map[string]any{"chapter": chapters[idx], "content": body}
			if idx+1 < len(chapters) {
				out["next"] = map[string]any{"path": chapters[idx+1].Path, "number": chapters[idx+1].Number, "title": chapters[idx+1].Title}
			}
			return wenxueJSON(out)
		},
	}
}

func wenxueFinalWriteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_final_write",
		desc:     "写/更新定稿章节（发布稿专用；只允许写定稿目录）。用户明确说「定稿」「入库」「发布稿」时才用；草稿请用 wenxue_chapter_write。",
		schema:   `{"type":"object","properties":{"path":{"type":"string","description":"定稿文件相对路径（须在定稿目录内；缺省按 number+title 生成 序号-标题.md）"},"number":{"type":"integer","description":"章号"},"title":{"type":"string","description":"章节标题（新建命名用）"},"content":{"type":"string","description":"定稿全文"},"overwrite":{"type":"boolean","description":"允许覆盖已存在定稿，缺省 false"}},"required":["content"],"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var in struct {
				Path      string `json:"path"`
				Number    int    `json:"number"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				Overwrite bool   `json:"overwrite"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			if strings.TrimSpace(in.Content) == "" {
				return "", fmt.Errorf("content 为空")
			}
			root := wenxueWorkspace()
			rel := filepath.ToSlash(strings.TrimSpace(in.Path))
			if rel == "" {
				dir := wenxueEnsureDir(root, "final")
				title := strings.TrimSpace(in.Title)
				if title == "" {
					num0 := in.Number
					if num0 <= 0 {
						return "", fmt.Errorf("path 缺省时需提供 number 或 title（否则定稿文件无法命名）")
					}
					title = "未命名"
				}
				num := in.Number
				if num > 0 {
					for _, c := range wenxueListFinalChapters(root) {
						if c.Number == num {
							rel = c.Path
							break
						}
					}
				}
				if rel == "" {
					rel = fmt.Sprintf("%s/%03d-%s.md", dir, num, sanitizeFileName(title))
				}
			}
			if !wenxueIsUnderFinalDir(root, rel) {
				return "", fmt.Errorf("wenxue_final_write 只允许写定稿目录内（%s）；草稿请用 wenxue_chapter_write", strings.Join(wenxueDirCandidates(root, "final"), "/"))
			}
			abs, err := wspath.ResolveRelWithin(root, rel)
			if err != nil {
				return "", fmt.Errorf("非法工作区相对路径: %s", rel)
			}
			if info, err := os.Stat(abs); err == nil && !in.Overwrite {
				return "", fmt.Errorf("定稿已存在（%d 字节，%s）；确认覆盖请传 overwrite=true", info.Size(), info.ModTime().Format("15:04:05"))
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			prevWords := 0
			if b, err := os.ReadFile(abs); err == nil {
				prevWords = wenxueCountWords(string(b))
			}
			if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
				return "", err
			}
			meta, _ := wenxueParseChapterFile(root, rel)
			return wenxueJSON(map[string]any{
				"saved": meta, "words": meta.Words, "prevWords": prevWords,
				"delta":        meta.Words - prevWords,
				"contentChars": utf8.RuneCountInString(in.Content), "wordCountRule": wenxueWordCountRule,
				"hint": "定稿已入库：阅读标签页刷新即见；阅读 tab 与 wenxue_final_list/read 共享此目录",
			})
		},
	}
}

// ── 小 helper ───────────────────────────────────────────────────

// wenxueFirstLine 返回文本首个非空行（截 120 rune；gaea_export.firstLine 同名避开）。
func wenxueFirstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			r := []rune(line)
			if len(r) > 120 {
				return string(r[:120])
			}
			return line
		}
	}
	return ""
}

func avgOf(total, n int) int {
	if n <= 0 {
		return 0
	}
	return total / n
}

// sanitizeFileName 文件名消毒：去非法字符与首尾空白（Windows 保留名兜底交给
// 系统 Create 报错；长度钳 60 rune）。
func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		if strings.ContainsRune(`\/:*?"<>|`, r) || r < 32 {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	r := []rune(out)
	if len(r) > 60 {
		out = string(r[:60])
	}
	if out == "" {
		out = "未命名"
	}
	return out
}

// ── 质检族（2026-11-17 审查批8 开工，原挂池项）────────────────────────
// 全书确定性体检（零 LLM）：按图文 md 工作区约定全新自建——不背 novelgate
// 的 JSON 项目数据模型（随小说板退役删除），LLM 评审归 novel-review 技能
// （分工与技能文档一致）。分组回执仅非空携带（窄告警口径），items 封顶。

// wenxueQualityItem 单条发现（path=工作区相对路径或文件:行）。
type wenxueQualityItem struct {
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// wenxueQualityGroup 分组发现集（仅非空携带进回执）。
type wenxueQualityGroup struct {
	Name     string              `json:"name"`
	Severity string              `json:"severity"` // warn | info
	Count    int                 `json:"count"`
	Items    []wenxueQualityItem `json:"items"`
}

// wenxueQualityItemCap 单组回执封顶（窄告警口径：全量在文件里，回执给信号）。
const wenxueQualityItemCap = 8

// wenxueCardSections 角色卡六节骨架（与 character 卡契约/前端 hub 同源）。
var wenxueCardSections = []string{"设定", "外貌", "性格", "口头禅", "关系", "当前状态"}

// wenxueQualityCheckTool 全书确定性体检。
func wenxueQualityCheckTool() *wenxueTool {
	const name = "wenxue_quality_check"
	return &wenxueTool{
		name:     name,
		desc:     "全书确定性体检（零 LLM）：章节编号连续性（跳号/重号/未编号）、角色卡六节完整性、时间线未锚与中文序号行、设定空文档/无标题、孤儿候选目录；分组回执仅非空携带，items 封顶 8。LLM 评审归 novel-review 技能。",
		schema:   `{"type":"object","properties":{},"additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			var groups []wenxueQualityGroup
			add := func(gName, severity string, items []wenxueQualityItem) {
				if len(items) == 0 {
					return
				}
				g := wenxueQualityGroup{Name: gName, Severity: severity, Count: len(items), Items: items}
				if len(g.Items) > wenxueQualityItemCap {
					g.Items = g.Items[:wenxueQualityItemCap]
				}
				groups = append(groups, g)
			}

			// 1) 章节编号：跳号/重号/未编号（清单按章号升序对照）。
			chapters := wenxueListChapters(root)
			{
				var items []wenxueQualityItem
				seen := map[int]string{}
				prev := 0
				for _, c := range chapters {
					if c.Number == 0 {
						items = append(items, wenxueQualityItem{Path: c.Path, Message: "未编号章（文件名无前导序号）"})
						continue
					}
					if first, dup := seen[c.Number]; dup {
						items = append(items, wenxueQualityItem{Path: c.Path, Message: fmt.Sprintf("章号 %d 重号（与 %s）", c.Number, first)})
						continue
					}
					seen[c.Number] = c.Path
					if prev > 0 && c.Number > prev+1 {
						items = append(items, wenxueQualityItem{Path: c.Path, Message: fmt.Sprintf("章号跳号：%d 之后直接 %d（缺 %d-%d）", prev, c.Number, prev+1, c.Number-1)})
					}
					prev = c.Number
				}
				add("章节编号", "warn", items)
			}

			// 2) 角色卡：六节骨架缺节（全无节头=自由格式卡，info 如实）。
			{
				var items []wenxueQualityItem
				for _, m := range wenxueListCharacters(root) {
					p, _ := m["path"].(string)
					if p == "" {
						continue
					}
					body, _, err := wenxueReadBody(root, p)
					if err != nil {
						continue
					}
					has := map[string]bool{}
					anySection := false
					for _, line := range strings.Split(body, "\n") {
						t := strings.TrimSpace(line)
						if !strings.HasPrefix(t, "## ") {
							continue
						}
						anySection = true
						has[strings.TrimSpace(strings.TrimPrefix(t, "## "))] = true
					}
					if !anySection {
						items = append(items, wenxueQualityItem{Path: p, Message: "自由格式卡（无节结构，hub 按原文回退展示）"})
						continue
					}
					var missing []string
					for _, sec := range wenxueCardSections {
						if !has[sec] {
							missing = append(missing, sec)
						}
					}
					if len(missing) > 0 {
						items = append(items, wenxueQualityItem{Path: p, Message: "缺节：" + strings.Join(missing, "、")})
					}
				}
				add("角色卡", "warn", items)
			}

			// 3) 时间线：未锚/中文序号行（机器不可排序；回写纪律=阿拉伯数字）。
			{
				var items []wenxueQualityItem
				for _, e := range wenxueTimelineEntries(root) {
					if e.Chapter == 0 {
						items = append(items, wenxueQualityItem{
							Path:    fmt.Sprintf("%s:%d", e.File, e.Line),
							Message: "未锚/中文序号事件行（第N章锚缺失或中文数字不可解析）",
						})
					}
				}
				add("时间线", "warn", items)
			}

			// 4) 设定：空文档/无一级标题。
			{
				var items []wenxueQualityItem
				dir := wenxueFindDir(root, "settings")
				if dir != "" {
					if abs, err := wspath.ResolveRelWithin(root, dir); err == nil {
						entries, _ := os.ReadDir(abs)
						for _, e := range entries {
							if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
								continue
							}
							rel := filepath.ToSlash(filepath.Join(dir, e.Name()))
							body, _, err := wenxueReadBody(root, rel)
							if err != nil {
								continue
							}
							if strings.TrimSpace(body) == "" {
								items = append(items, wenxueQualityItem{Path: rel, Message: "空文档"})
							} else if !strings.Contains(body, "\n# ") && !strings.HasPrefix(body, "# ") {
								items = append(items, wenxueQualityItem{Path: rel, Message: "无一级标题（大纲/渲染按首行标题兜底）"})
							}
						}
					}
				}
				add("设定", "info", items)
			}

			// 5) 孤儿候选：章节/候选/<stem>/ 的 stem 不在任何章节清单里。
			{
				var items []wenxueQualityItem
				stems := map[string]bool{}
				for _, c := range chapters {
					stems[strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))] = true
				}
				for _, dir := range wenxueDirCandidates(root, "chapters") {
					candRel := filepath.ToSlash(filepath.Join(dir, "候选"))
					abs, err := wspath.ResolveRelWithin(root, candRel)
					if err != nil {
						continue
					}
					entries, _ := os.ReadDir(abs)
					for _, e := range entries {
						if !e.IsDir() {
							continue
						}
						if !stems[e.Name()] {
							items = append(items, wenxueQualityItem{
								Path:    filepath.ToSlash(filepath.Join(candRel, e.Name())),
								Message: "孤儿候选目录（对应章不在章节清单）",
							})
						}
					}
				}
				add("孤儿候选", "warn", items)
			}

			checked := map[string]int{
				"chapters": len(chapters), "characters": len(wenxueListCharacters(root)),
				"timeline": len(wenxueTimelineEntries(root)),
			}
			return wenxueJSON(map[string]any{"ok": true, "checked": checked, "groups": groups})
		},
	}
}
