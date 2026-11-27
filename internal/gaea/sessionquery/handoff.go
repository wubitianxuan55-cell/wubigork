package sessionquery

// handoff.go — 会话交接卡（2026-11-27 AI 反馈 #14）。
//
// 同一项目连续几场会话接力时，上一场的产物带着缺陷交棒，下一场开场只能翻文件
// 现场重推状态。交接卡按需取（session_search handoff=<会话 id>）：这场会话的首条
// 任务、最后一条助手成稿、经工具写过的文件及体检现状。不自动注入——要不要读
// 由模型判断（2026-11-28 用户拍板）。
//
// 日志两种形态都认：实时日志里工具调用是独立的 tool_dispatch / tool_call 条目
// （带 readOnly 标记）；迁移来的旧日志把工具调用内嵌在 assistant_message 里
// （没有标记，只读工具只能按名字排除）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// ArtifactHealth 交接卡逐件体检的钩子：给一个存在的文件的绝对路径，返回一句现状
// （如 xlsx 的打开性与缓存值）；返回空串表示没有类型专属结论。由装配层接线
// （boot 用 xlsxedit），本包不认识具体文件类型；没接线时只报存在与否。
var ArtifactHealth func(path string) string

// SetArtifactHealth 接上体检钩子（nil = 撤掉）。
func SetArtifactHealth(fn func(path string) string) { ArtifactHealth = fn }

// HandoffFile 交接卡里的一个文件：这场会话经工具写过它。
type HandoffFile struct {
	Path   string   // 工具参数里的原样路径
	Tools  []string // 写过它的工具（去重，最多 4 个）
	Calls  int
	OK     bool   // 至少有一次成功回执
	Health string // 现状：存在与否 + 体检结论
}

// HandoffCard 一场会话的交接卡。
type HandoffCard struct {
	ID          string
	Title       string
	Started     time.Time
	LastActive  time.Time
	FirstPrompt string
	LastReply   string
	Files       []HandoffFile
	BadLines    int
	LegacyCalls bool // 日志里有内嵌形态的工具调用（读写只能按工具名区分）
}

// handoffMaxFiles 交接卡最多列多少个文件。
const handoffMaxFiles = 40

// readOnlyToolNames 旧格式日志的内嵌调用没有 readOnly 标记，按名字排除只读工具。
var readOnlyToolNames = map[string]bool{
	"read_file": true, "read_spill": true, "list_dir": true, "grep": true, "glob": true,
	"find": true, "search": true, "xlsx_audit": true, "xlsx_profile": true, "md_validate": true,
	"session_search": true, "proposal_search": true, "proposal_read": true, "cost_project": true,
	"cost_audit": true, "web_fetch": true, "web_search": true, "doc_render": true,
}

// writtenPathKeys 写类工具参数里放目标路径的键，按这个顺序取第一个非空字符串。
var writtenPathKeys = []string{"path", "output", "output_path", "destination", "dest", "target", "file"}

var (
	// reBashRedirect bash 命令里的重定向目标：> 或 >> 后面的路径。
	reBashRedirect = regexp.MustCompile(`>{1,2}\s*["']?([^\s"'|;&<>()]+)`)
	// reFileWithExt 像文件路径：带 1~6 位字母数字扩展名。
	reFileWithExt = regexp.MustCompile(`^[^\s/\\]+\.[A-Za-z0-9]{1,6}$|[/\\][^\s/\\]+\.[A-Za-z0-9]{1,6}$`)
	// reMsysDrive bash 方言的盘符路径（/d/foo → D:\foo）。
	reMsysDrive = regexp.MustCompile(`^/([a-zA-Z])(/|$)`)
)

// Handoff 生成一场会话的交接卡：读它的日志（只认完整行），抽首条任务、最后一条
// 助手成稿、经工具写过的文件及其现状。id 用清单里的会话 id。
func (s *Store) Handoff(id string) (*HandoffCard, error) {
	path, sid, err := s.locateSession(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	entries, badLines, err := readLogTolerant(path)
	if err != nil {
		return nil, fmt.Errorf("read session log %s: %w", path, err)
	}
	card := &HandoffCard{ID: sid, BadLines: badLines, Title: loadTitles(s.dir)[sid]}
	if st, ok := startTimeFromID(sid); ok {
		card.Started = st
	}
	// 回执排在调用之后：先扫一遍「哪些调用成功了」，再扫调用。
	succeeded := map[string]bool{}
	for _, e := range entries {
		if e.Kind != session.KindToolResult {
			continue
		}
		var r struct {
			ID  string `json:"id"`
			Err string `json:"err"`
		}
		if json.Unmarshal(e.Payload, &r) == nil && r.ID != "" {
			succeeded[r.ID] = strings.TrimSpace(r.Err) == ""
		}
	}
	files := map[string]*HandoffFile{}
	var order []string
	add := func(p, name string, ok bool) {
		f := files[p]
		if f == nil {
			f = &HandoffFile{Path: p}
			files[p] = f
			order = append(order, p)
		}
		f.Calls++
		if ok {
			f.OK = true
		}
		if len(f.Tools) < 4 {
			for _, t := range f.Tools {
				if t == name {
					return
				}
			}
			f.Tools = append(f.Tools, name)
		}
	}
	take := func(c struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Args     string `json:"args"`
		ReadOnly bool   `json:"readOnly"`
		Partial  bool   `json:"partial"`
	}, legacy bool) {
		if c.Partial || strings.TrimSpace(c.Args) == "" || c.ReadOnly || (legacy && readOnlyToolNames[c.Name]) {
			return
		}
		ok := succeeded[c.ID]
		if c.Name == "bash" {
			var a struct {
				Command string `json:"command"`
			}
			if json.Unmarshal([]byte(c.Args), &a) != nil {
				return
			}
			for _, p := range bashRedirectPaths(a.Command) {
				add(p, c.Name, ok)
			}
			return
		}
		if p := writtenPath(c.Args); p != "" {
			add(p, c.Name, ok)
		}
	}
	for _, e := range entries {
		if e.Ts > 0 {
			card.LastActive = time.Unix(e.Ts, 0)
		}
		switch e.Kind {
		case session.KindToolCall, "tool_dispatch":
			var c struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Args     string `json:"args"`
				ReadOnly bool   `json:"readOnly"`
				Partial  bool   `json:"partial"`
			}
			if json.Unmarshal(e.Payload, &c) == nil {
				take(c, false)
			}
		case session.KindAssistantMessage:
			// 旧格式：工具调用内嵌在助手消息里（正文另由 indexableContent 抽）。
			var p struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Args     string `json:"args"`
					ReadOnly bool   `json:"readOnly"`
					Partial  bool   `json:"partial"`
				} `json:"tool_calls"`
			}
			if json.Unmarshal(e.Payload, &p) == nil {
				for _, c := range p.ToolCalls {
					if !c.ReadOnly && !c.Partial {
						card.LegacyCalls = true
					}
					take(c, true)
				}
			}
		}
		if role, content, ok := indexableContent(e); ok {
			if role == "user" && card.FirstPrompt == "" {
				card.FirstPrompt = content
			}
			if role == "assistant" {
				card.LastReply = content
			}
		}
	}
	for _, p := range order {
		if len(card.Files) >= handoffMaxFiles {
			break
		}
		f := files[p]
		f.Health = s.fileHealth(p)
		card.Files = append(card.Files, *f)
	}
	return card, nil
}

// locateSession 按会话 id 找到它的日志路径。
func (s *Store) locateSession(id string) (path, sid string, err error) {
	if id == "" {
		return "", "", errors.New("handoff 需要会话 id（session_search 不带 query 列出的那些）")
	}
	if _, err := s.Refresh(); err != nil {
		return "", "", err
	}
	rows, err := s.db.Query(`SELECT DISTINCT session_id, path FROM session_fts`)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid, cpath string
		if err := rows.Scan(&cid, &cpath); err != nil {
			return "", "", err
		}
		if cid == id {
			return cpath, cid, nil
		}
	}
	return "", "", fmt.Errorf("本工作区的会话索引里没有 %q（先 session_search 不带 query 列出最近的会话）", id)
}

// writtenPath 从写类工具的参数里取目标路径；取不到或不像文件路径返回空串。
func writtenPath(args string) string {
	var m map[string]any
	if json.Unmarshal([]byte(args), &m) != nil {
		return ""
	}
	for _, k := range writtenPathKeys {
		v, ok := m[k].(string)
		if !ok || strings.TrimSpace(v) == "" || strings.ContainsAny(v, "\n\r") {
			continue
		}
		return strings.TrimSpace(v)
	}
	return ""
}

// bashRedirectPaths bash 命令里重定向写出的文件（> 与 >>）。脚本内部写的文件
// 看不出来，这是清单的已知盲区。
func bashRedirectPaths(command string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range reBashRedirect.FindAllStringSubmatch(command, -1) {
		p := m[1]
		if p == "" || strings.HasPrefix(p, "&") || strings.HasPrefix(p, "/dev/") || strings.Contains(p, "://") ||
			!reFileWithExt.MatchString(p) || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// workspaceRoot 会话目录所属的工作区根（向上找到 .gaea 的上一级）；找不到返回空。
func (s *Store) workspaceRoot() string {
	for dir := s.dir; ; {
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		if filepath.Base(dir) == ".gaea" {
			return parent
		}
		dir = parent
	}
}

// fileHealth 一件产物的现状：在不在、多大、有没有类型专属的体检结论。
func (s *Store) fileHealth(path string) string {
	abs := path
	if runtime.GOOS == "windows" {
		if m := reMsysDrive.FindStringSubmatch(abs); m != nil {
			abs = strings.ToUpper(m[1]) + ":" + filepath.FromSlash(abs[len(m[1])+1:])
		}
	}
	if !filepath.IsAbs(abs) {
		root := s.workspaceRoot()
		if root == "" {
			return "相对路径，但定不了工作区根"
		}
		abs = filepath.Join(root, filepath.FromSlash(abs))
	}
	fi, err := os.Stat(abs)
	if err != nil || fi.IsDir() {
		return "已不存在（被移走、改名或删了）"
	}
	if ArtifactHealth != nil {
		if note := ArtifactHealth(abs); note != "" {
			return note
		}
	}
	return fmt.Sprintf("在（%s）", humanSize(fi.Size()))
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// FormatHandoff 渲染交接卡。
func FormatHandoff(c *HandoffCard) string {
	var b strings.Builder
	title := c.Title
	if title == "" {
		title = "（无标题）"
	}
	fmt.Fprintf(&b, "## 会话交接卡：%s\n\n%s｜%s → %s\n\n", title, c.ID,
		c.Started.Format("2006-01-02 15:04"), c.LastActive.Format("2006-01-02 15:04"))
	task := oneLine(c.FirstPrompt, 200)
	if strings.TrimSpace(task) == "" {
		task = "（没有用户消息）"
	}
	fmt.Fprintf(&b, "**任务**：%s\n\n", task)
	if strings.TrimSpace(c.LastReply) == "" {
		b.WriteString("**收尾回复**：这场会话没有助手成稿。\n\n")
	} else {
		fmt.Fprintf(&b, "**收尾回复**（最后一条助手消息的开头；要全文用 session_search 带关键词检索）：\n%s\n\n", clipBlock(c.LastReply, 800))
	}
	if len(c.Files) == 0 {
		b.WriteString("**动过的文件**：这场会话没有经工具直接写出的文件（bash 脚本内部写出的看不出来）。\n\n")
	} else {
		fmt.Fprintf(&b, "**动过的文件**（%d 个，去重；路径是工具参数里的原样写法", len(c.Files))
		if c.LegacyCalls {
			b.WriteString("；旧格式日志里读写只能按工具名区分，个别只读的可能混入")
		}
		b.WriteString("）：\n")
		for _, f := range c.Files {
			state := "未见成功回执"
			if f.OK {
				state = "有成功回执"
			}
			fmt.Fprintf(&b, "- %s（%s ×%d，%s）：%s\n", f.Path, strings.Join(f.Tools, " / "), f.Calls, state, f.Health)
		}
		b.WriteString("\n")
	}
	if c.BadLines > 0 {
		fmt.Fprintf(&b, "⚠ 这份日志有 %d 行解不出来，已跳过。\n\n", c.BadLines)
	}
	b.WriteString("交接提示：收尾回复是上一场的自述，不是复核。交付件接着用之前先看上面的体检，再用对应的审计工具过一遍（xlsx_audit / cost_library 导入演练 / md_validate）。\n")
	return b.String()
}

// clipBlock 多行文本截到 n 个 rune，超了注明。
func clipBlock(s string, n int) string {
	s = strings.TrimSpace(s)
	if rs := []rune(s); len(rs) > n {
		return string(rs[:n]) + "\n…（回复较长，已截）"
	}
	return s
}
