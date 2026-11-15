package app

// wenxue_tools.go — 图文专属 agent 工具四族（2026-11-15 刀3）。
//
// 章节 / 角色 / 时间线 / 统计，命名 wenxue_* 前缀，仅装配进图文独立引擎
//（wenxueExtraTools，办公引擎 ExtraTools 不含 → 办公会话零出现）。
// 目录约定见 wenxue-workbench 技能（宽容匹配中文/英文目录名）：
//   章节/|chapters/、角色/|characters/、时间线/|timeline/、设定/|worldview/。
// 路径安全：全部写读经 wspath.ResolveRelWithin（工作区内解析，拒 .. 穿越）；
// 写侧带 overwrite 显式闸（缺省 false，已存在即拒——宁拒不误改）。
// 契约行：RegistryMap 十二行（extra-app/work——刀3 原九件+定稿刀三件；
// 2026-11-15 众神殿首刀收口批改口径归一），装配面守卫=wenxue_surface_probe_test.go
// census 三例。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/tool"
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
		wenxueCharacterListTool(),
		wenxueCharacterReadTool(),
		wenxueCharacterWriteTool(),
		wenxueTimelineReadTool(),
		wenxueTimelineAppendTool(),
		wenxueStatsTool(),
		wenxueFinalListTool(),
		wenxueFinalReadTool(),
		wenxueFinalWriteTool(),
	}
}

// wenxueBaseTools 图文引擎通用 extra 工具单点（= gaeaExtraTools 媒体根分档版，
// 2026-11-15 众神殿首刀，pan_1794754879290 裁决）：image_gen/ocr 媒体根注入
// wenxueWorkspace——图文会话生成图与 OCR 相对路径锚当前小说目录，不再落办公
// 工作区；vision 不动（builtin-workspace 随 Build Cwd 注入，本就正确）。
// 办公装配走 gaeaExtraTools 原样（nil 缝=gaeaCwd 现状逐字节，红线）。
func wenxueBaseTools(a *App) []tool.Tool {
	return extraToolsWithMediaRoot(a, wenxueWorkspace)
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

// wenxueReadBody 读工作区文件全文（非法路径/缺文件返回错误）。
func wenxueReadBody(root, rel string) (string, os.FileInfo, error) {
	abs, err := wspath.ResolveRelWithin(root, rel)
	if err != nil {
		return "", nil, fmt.Errorf("非法工作区相对路径: %s", rel)
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
				"dir":        wenxueFindDir(root, "chapters"),
				"count":      len(chapters),
				"totalWords": total,
				"chapters":   chapters,
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
			if wenxueIsUnderFinalDir(root, rel) {
				return "", fmt.Errorf("定稿目录（%s）请用 wenxue_final_write——章节工具不写定稿，防草稿入库", wenxueFindDir(root, "final"))
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
				"hint": "回写配套：更新相关角色卡「当前状态」与时间线事件（wenxue_character_write / wenxue_timeline_append）",
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

func wenxueCharacterListTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_list",
		desc:     "列出角色卡清单（名字/字数/摘要）。写新章前用它确认出场人物。",
		schema:   `{"type":"object","additionalProperties":false}`,
		readOnly: true,
		run: func(args json.RawMessage) (string, error) {
			root := wenxueWorkspace()
			cards := wenxueListCharacters(root)
			return wenxueJSON(map[string]any{"dir": wenxueFindDir(root, "characters"), "count": len(cards), "characters": cards})
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
			return wenxueJSON(map[string]any{"path": rel, "content": body, "words": wenxueCountWords(body)})
		},
	}
}

func wenxueCharacterWriteTool() *wenxueTool {
	return &wenxueTool{
		name:     "wenxue_character_write",
		desc:     "建/更新角色卡（全量正文）。overwrite=false 时已存在即拒。写后回执带卡面字数。",
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
				"hint": "章节推进后记得更新「当前状态」小节（再次调用本工具 overwrite=true）",
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
				"dir":        wenxueFindDir(root, "final"),
				"count":      len(chapters),
				"totalWords": total,
				"chapters":   chapters,
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
				"delta": meta.Words - prevWords,
				"hint":  "定稿已入库：阅读标签页刷新即见；阅读 tab 与 wenxue_final_list/read 共享此目录",
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
