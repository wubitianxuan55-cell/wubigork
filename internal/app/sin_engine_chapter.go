package app

// ── 原罪章节协议（2026-11-15 升格刀3）──────────────────────────────────
//
// 拍板（用户）：章节标记协议——模型输出 `@@章节|标题@@` 独占一行显式开新章
// （对齐 @@插图|描述@@ 协议族）；一回可多章、一章可跨回合；回合结束后扫描
// **本回合最后一条助手消息**，逐章落盘到书目录 chapters/NNN.md（编号自动续接
// MaxChapterBodyNum+1），并在 outline.json 追加/更新节点（Title/ChapterFile/
// Status=done）。
//
// 边界：
//   - 只服务升格引擎路径（sinAfterTurn 挂 sin 控制器）；legacy 管线不感知
//     （sin_chapter 只进引擎注册表，不进 sinToolOrder/legacy 提示词锁）；
//   - 落盘走 internal/project Manager（书目录直操，不碰 getPM() 全局当前项目
//     ——与办公/小说板块零互扰）；
//   - 未绑定书（bookPath 空）= 只解析不落盘（标记仍在正文，前端导航可见）；
//   - 标记行本身不进章节文件（标题归 outline 节点，文件=纯正文）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// sinChapterMarkRe 章节标记行：@@章节|标题@@ 独占一行（行首尾允许空白）。
var sinChapterMarkRe = regexp.MustCompile(`(?m)^[ \t]*@@章节\|([^@]{1,80}?)@@[ \t]*$`)

// sinToolChapter sin_chapter 工具名（章节交付工具）。
const sinToolChapter = "sin_chapter"

// sinChapterBackupDir 章节重写备份目录（书/chapters/.history/）。
const sinChapterBackupDir = ".history"

func init() {
	registerSinTool(sinToolChapter, func(c sinToolContext) sinTool {
		return &sinChapterTool{app: c.app, storeKey: c.storeKey(), bookPath: c.bookPath}
	})
}

// sinChapterMark 一处章节标记（行边界为字节偏移，含整行）。
type sinChapterMark struct {
	Title string
	Start int // 标记行起点（字节）
	End   int // 标记行终点（字节，含行尾换行）
}

// sinScanChapterMarks 扫描章节标记行（独占一行口径；零标记=空切片）。
func sinScanChapterMarks(text string) []sinChapterMark {
	locs := sinChapterMarkRe.FindAllStringSubmatchIndex(text, -1)
	out := make([]sinChapterMark, 0, len(locs))
	for _, loc := range locs {
		out = append(out, sinChapterMark{
			Title: strings.TrimSpace(text[loc[2]:loc[3]]),
			Start: loc[0],
			End:   loc[1],
		})
	}
	return out
}

// sinSplitChapters 按标记切章：每章 = 标记行之后到下一标记行/文本结尾。
// 返回 (标题, 正文) 序列；首个标记前的内容不属于任何章（回合旁白，不落盘）。
func sinSplitChapters(text string) [][2]string {
	marks := sinScanChapterMarks(text)
	if len(marks) == 0 {
		return nil
	}
	out := make([][2]string, 0, len(marks))
	for i, m := range marks {
		end := len(text)
		if i+1 < len(marks) {
			end = marks[i+1].Start
		}
		body := strings.TrimSpace(text[m.End:end])
		out = append(out, [2]string{m.Title, body})
	}
	return out
}

// sinWrittenChapter 单章落盘回执（前端导航/交付物消费）。
type sinWrittenChapter struct {
	Num         int    `json:"num"`
	Title       string `json:"title"`
	ChapterFile string `json:"chapterFile"`
}

// sinPersistChapters 把本回合章节写入书项目（编号续接 + outline 节点）。
// 幂等语义：每回合只扫自己的消息（TurnDone 一次）；重写既有章走 sin_chapter
// 工具（带备份）。书未就绪/无标记 = 零动作。
func (a *App) sinPersistChapters(bookPath, text string) []sinWrittenChapter {
	chapters := sinSplitChapters(text)
	if len(chapters) == 0 || strings.TrimSpace(bookPath) == "" {
		return nil
	}
	m, err := project.Open(bookPath)
	if err != nil {
		slog.Warn("sin: 章节落盘失败（书项目打开）", "book", bookPath, "error", err)
		return nil
	}
	base, err := m.MaxChapterBodyNum()
	if err != nil {
		base = 0
	}
	of, err := m.ReadOutlines()
	if err != nil || of == nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	maxOrder := 0
	for i := range of.Nodes {
		if of.Nodes[i].OrderIndex > maxOrder {
			maxOrder = of.Nodes[i].OrderIndex
		}
	}
	written := make([]sinWrittenChapter, 0, len(chapters))
	for i, ch := range chapters {
		title := ch[0]
		body := ch[1]
		if title == "" {
			title = fmt.Sprintf("第%d回", base+i+1)
		}
		if strings.TrimSpace(body) == "" {
			continue // 空章不落盘（标记后无正文=未写完，下一回合续）
		}
		num := base + i + 1
		if err := m.WriteChapter(num, body); err != nil {
			slog.Warn("sin: 章节写入失败", "num", num, "error", err)
			continue
		}
		chapterFile := fmt.Sprintf("%03d.md", num)
		node := types.OutlineNode{
			ID:          fmt.Sprintf("sin-%03d", num),
			Title:       title,
			Status:      types.OutlineDone,
			ChapterFile: chapterFile,
			OrderIndex:  maxOrder + i + 1,
		}
		of.Nodes = sinUpsertOutlineNode(of.Nodes, node)
		written = append(written, sinWrittenChapter{Num: num, Title: title, ChapterFile: chapterFile})
	}
	if len(written) > 0 {
		if err := m.WriteOutlines(of); err != nil {
			slog.Warn("sin: outline 回写失败（章节文件已落盘）", "error", err)
		}
		slog.Info("sin: 章节已落盘", "book", filepath.ToSlash(bookPath), "count", len(written))
	}
	return written
}

// sinUpsertOutlineNode 按 ChapterFile 幂等合并节点（同章重扫不堆积）。
func sinUpsertOutlineNode(nodes []types.OutlineNode, node types.OutlineNode) []types.OutlineNode {
	for i := range nodes {
		if nodes[i].ChapterFile == node.ChapterFile {
			nodes[i].Title = node.Title
			nodes[i].Status = node.Status
			return nodes
		}
	}
	return append(nodes, node)
}

// sinAfterTurn 回合成功后钩子：扫描本回合助手消息，章节落盘。
// 失败只记日志（不阻断回合；章节标记仍在正文，前端导航可见）。
func (a *App) sinAfterTurn() {
	sinEng.mu.Lock()
	ctrl, book := sinEng.ctrl, sinEng.bookPath
	sinEng.mu.Unlock()
	if ctrl == nil || book == "" {
		return
	}
	msgs := ctrl.History()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" {
			a.sinPersistChapters(book, msgs[i].Content)
			return
		}
	}
}

// ── sin_chapter 工具（章节交付：list/read/write；写护栏限 chapters/NNN.md）──

type sinChapterTool struct {
	app      *App
	storeKey string // 书 slug（引擎路径）/故事 id（legacy，工具仅引擎注册）
	bookPath string // 引擎装配时注入的书目录
}

func (t *sinChapterTool) Name() string   { return sinToolChapter }
func (t *sinChapterTool) ReadOnly() bool { return false }
func (t *sinChapterTool) Description() string {
	return "章节交付工具：list 列出本书章节（编号/标题/状态）；read 读指定章正文；write 写/改指定章（num 必填，title 更新大纲节点标题，正文整章替换，重写自动备份到 chapters/.history/）。写文件仅限本书 chapters/NNN.md，越界拒绝。"
}

func (t *sinChapterTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "action":{"type":"string","enum":["list","read","write"],"description":"list=列章节；read=读单章；write=写/改单章"},
    "num":{"type":"integer","description":"章节号（read/write 必填）"},
    "title":{"type":"string","description":"章标题（write 时更新大纲节点）"},
    "content":{"type":"string","description":"章正文（write 必填，整章替换）"}
  },
  "required":["action"]
}`)
}

// sinChapterEnginePath 引擎路径的书目录（未装配=空）。
func sinChapterEnginePath() string {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	return sinEng.bookPath
}

func (t *sinChapterTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action  string `json:"action"`
		Num     int    `json:"num"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	book := t.bookPath
	if book == "" {
		book = sinChapterEnginePath()
	}
	if strings.TrimSpace(book) == "" {
		return "", fmt.Errorf("未绑定书工作空间：先在书架选择一本书（章节落盘不可用）")
	}
	m, err := project.Open(book)
	if err != nil {
		return "", fmt.Errorf("书项目打开失败: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(p.Action)) {
	case "list":
		base, _ := m.MaxChapterBodyNum()
		of, err := m.ReadOutlines()
		if err != nil || of == nil {
			return fmt.Sprintf("本书暂无章节（chapters/ 目录最大章号=%d）", base), nil
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("本书章节（最大章号 %d）：\n", base))
		for _, n := range of.Nodes {
			if n.ChapterFile == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("- %s（%s，%s）\n", n.Title, n.ChapterFile, n.Status))
		}
		if b.Len() == 0 {
			b.WriteString("本书暂无章节\n")
		}
		return b.String(), nil
	case "read":
		if p.Num <= 0 {
			return "", fmt.Errorf("read 需要 num")
		}
		content, err := m.ReadChapter(p.Num)
		if err != nil {
			return "", fmt.Errorf("第 %d 章读取失败: %w", p.Num, err)
		}
		return content, nil
	case "write":
		if p.Num <= 0 {
			return "", fmt.Errorf("write 需要 num")
		}
		if strings.TrimSpace(p.Content) == "" {
			return "", fmt.Errorf("write 需要 content（整章正文）")
		}
		// 重写备份：chapters/.history/NNN-<ts>.md（原文件存在才备份）
		chPath := m.ChapterPath(p.Num)
		if _, statErr := os.Stat(chPath); statErr == nil {
			histDir := filepath.Join(filepath.Dir(chPath), sinChapterBackupDir)
			if mkErr := os.MkdirAll(histDir, 0o755); mkErr == nil {
				stamp := time.Now().Format("20060102-150405")
				_ = os.WriteFile(filepath.Join(histDir, fmt.Sprintf("%03d-%s.md", p.Num, stamp)), []byte(mustRead(chPath)), 0o600)
			}
		}
		if err := m.WriteChapter(p.Num, p.Content); err != nil {
			return "", fmt.Errorf("第 %d 章写入失败: %w", p.Num, err)
		}
		// outline 节点同步（标题更新/缺失补节点）
		of, err := m.ReadOutlines()
		if err == nil && of != nil {
			title := strings.TrimSpace(p.Title)
			if title == "" {
				title = fmt.Sprintf("第%d回", p.Num)
			}
			of.Nodes = sinUpsertOutlineNode(of.Nodes, types.OutlineNode{
				ID:          fmt.Sprintf("sin-%03d", p.Num),
				Title:       title,
				Status:      types.OutlineDone,
				ChapterFile: fmt.Sprintf("%03d.md", p.Num),
			})
			_ = m.WriteOutlines(of)
		}
		return fmt.Sprintf("第 %d 章已写入 %s（%d 字）", p.Num, filepath.ToSlash(chPath), len([]rune(p.Content))), nil
	default:
		return "", fmt.Errorf("未知 action %q（list/read/write）", p.Action)
	}
}

// mustRead 尽力读文件（备份用；读失败返回空串，不阻断写主流程）。
func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
