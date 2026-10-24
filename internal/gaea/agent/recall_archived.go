package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── A5 折叠区召回（Reasonix 2.x compact_search/compact_index 蒸馏，v4.524）──
//
// 压缩把旧对话折叠进 archive JSONL（路径附在 digest 尾部——gaea 的「fold
// index」），但找回=读整个 archive 文件：多代压缩后 archive 可以很大，读全文
// 找一段历史反而炸上下文预算。本工具提供**预算内检索**：确定性计分（关键词
// 与 CJK 双元命中），命中片段带出处（archive 文件+种类+所属调用），输出封顶。
// 与上游的刻意差异：上游检索 canonical 转录（投影在 DB），gaea 的折叠区在
// archive 文件里——检索面=archive 目录。

const (
	recallDefaultLimit = 8
	recallMaxLimit     = 20
	recallSnippetRunes = 240
	recallNoiseRatio   = 0.25
	// recallOutputBudget 输出总预算（rune）——超限如实截断并计数。
	recallOutputBudget = 4000
)

type archiveDirKey struct{}

// WithArchiveDir 注入 archive 目录（runDirect 组装 ctx 时；空=工具如实报
// 「未配置」）。子代理 ctx 继承（只读面）。
func WithArchiveDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, archiveDirKey{}, dir)
}

func archiveDirFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(archiveDirKey{}).(string); ok {
		return v
	}
	return ""
}

// recallArchivedTool 检索已折叠归档的会话段（只读）。
type recallArchivedTool struct{}

func init() { tool.RegisterBuiltin(recallArchivedTool{}) }

func (recallArchivedTool) Name() string { return "recall_archived" }

func (recallArchivedTool) Description() string {
	return "Search archived conversation segments that were compacted out of the working context (folded to disk before summarization). Deterministic keyword scoring over the archive files; returns ranked snippets with their source (archive file, kind, owning tool call). Use when you need a fact from earlier in the session that no longer appears in context and the digest's archive pointer is too large to read whole."
}

func (recallArchivedTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"What to look for. Keywords, identifiers, paths, or a phrase from the earlier conversation."},"limit":{"type":"integer","description":"Max snippets to return (default 8, max 20)."}},"required":["query"]}`)
}

func (recallArchivedTool) ReadOnly() bool { return true }
func (recallArchivedTool) CompactDescription() string {
	return "检索已折叠归档的会话段(压缩前原文落盘的 JSONL)——确定性关键词计分,命中片段带出处,输出封顶"
}
func (recallArchivedTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},"required":["query"]}`)
}

// recallDoc 一个检索单元（上游 recallDoc 同构）：tool_output 由其调用寻址——
// 召回片段连「命令/调用」一起返回，只有结果没有来路的片段没法用。
type recallDoc struct {
	archive string // 文件名（出处）
	pos     int    // 消息序号（出处）
	kind    string // user/assistant/tool_input/tool_output
	caller  string // tool_output 的所属调用名
	text    string
	score   int
}

func (recallArchivedTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	dir := archiveDirFromContext(ctx)
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("no archive directory is configured for this session (compaction archives disabled)")
	}
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	query := strings.TrimSpace(p.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	limit := p.Limit
	if limit <= 0 {
		limit = recallDefaultLimit
	}
	if limit > recallMaxLimit {
		limit = recallMaxLimit
	}

	docs, err := loadArchiveDocs(dir)
	if err != nil {
		return "", err
	}
	terms := recallTerms(query)
	if len(terms) == 0 {
		return "", fmt.Errorf("query produced no searchable terms")
	}
	for i := range docs {
		lower := strings.ToLower(docs[i].text)
		for _, t := range terms {
			docs[i].score += strings.Count(lower, t)
		}
	}
	// 只留命中，噪声比截断（远低于最高分的尾命中不返回——查询词太常见时
	// 不至于刷二十条近零行，上游 recallSearchNoise 同口径）。
	var hits []recallDoc
	for _, d := range docs {
		if d.score > 0 {
			hits = append(hits, d)
		}
	}
	if len(hits) == 0 {
		return "no matches in archived conversation segments", nil
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	best := hits[0].score
	kept := hits[:0]
	for _, d := range hits {
		if float64(d.score) >= float64(best)*recallNoiseRatio && len(kept) < limit {
			kept = append(kept, d)
		}
	}

	var b strings.Builder
	used := 0
	for _, d := range kept {
		line := fmt.Sprintf("[%s #%d %s%s] %s",
			d.archive, d.pos, d.kind, recallCallerSuffix(d.caller),
			html.EscapeString(recallSnippet(d.text, terms)))
		if used+len(line) > recallOutputBudget {
			fmt.Fprintf(&b, "... %d more hit(s) omitted (output budget)\n", len(kept)-countLines(b.String()))
			break
		}
		b.WriteString(line + "\n")
		used += len(line)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func recallCallerSuffix(caller string) string {
	if caller == "" {
		return ""
	}
	return ":" + caller
}

func countLines(s string) int { return strings.Count(s, "\n") }

// recallSnippet 取首个命中词附近的窗口（≤240 rune）。
func recallSnippet(text string, terms []string) string {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	best := -1
	for _, t := range terms {
		if i := strings.Index(lower, t); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	if best < 0 {
		return clipRecoveryValue(text)
	}
	half := recallSnippetRunes / 2
	start := best - half
	if start < 0 {
		start = 0
	}
	runes := []rune(text)
	// 按 rune 对齐 start
	sr := len([]rune(text[:start]))
	end := sr + recallSnippetRunes
	if end > len(runes) {
		end = len(runes)
	}
	out := string(runes[sr:end])
	if sr > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out += "…"
	}
	return out
}

// recallTerms 查询词：空白/标点切词 + CJK 双元滑窗（中文无词边界，与实体卡
// 触发注入同口径）；≥2 rune 才参与（单字误伤面大）；去重。
func recallTerms(query string) []string {
	var terms []string
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		if len([]rune(t)) >= 2 && !seen[t] {
			seen[t] = true
			terms = append(terms, t)
		}
	}
	var run []rune
	flush := func() {
		rs := []rune(string(run))
		if hasCJK(rs) {
			for i := 0; i+1 < len(rs); i++ {
				add(string(rs[i : i+2]))
			}
		} else {
			add(string(run))
		}
		run = run[:0]
	}
	for _, r := range query {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == ',' || r == '，' || r == '。' || r == '、' || r == '：' || r == ':' || r == '"' || r == '\'':
			flush()
		default:
			run = append(run, r)
		}
	}
	flush()
	return terms
}

func hasCJK(rs []rune) bool {
	for _, r := range rs {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// loadArchiveDocs 扫 archive 目录（*.jsonl，旧→新），解析每条消息为检索文档。
func loadArchiveDocs(dir string) ([]recallDoc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no archive directory found at %s", dir)
		}
		return nil, fmt.Errorf("read archive dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var docs []recallDoc
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue // 单档损坏不拖垮检索（其余照常可搜，上游 per-source 口径）
		}
		var msgs []provider.Message
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m provider.Message
			if json.Unmarshal([]byte(line), &m) != nil {
				continue
			}
			msgs = append(msgs, m)
		}
		callAt := map[string]string{}
		for i, m := range msgs {
			switch m.Role {
			case provider.RoleUser:
				if !isCompactionSummary(m) && !isSyntheticUserTurn(m.Content) {
					docs = append(docs, recallDoc{archive: name, pos: i, kind: "user", text: m.Content})
				}
			case provider.RoleAssistant:
				docs = append(docs, recallDoc{archive: name, pos: i, kind: "assistant", text: m.Content})
				for _, tc := range m.ToolCalls {
					callAt[tc.ID] = tc.Name
					docs = append(docs, recallDoc{archive: name, pos: i, kind: "tool_input", caller: tc.Name, text: tc.Arguments})
				}
			case provider.RoleTool:
				docs = append(docs, recallDoc{archive: name, pos: i, kind: "tool_output", caller: callAt[m.ToolCallID], text: m.Content})
			}
		}
	}
	return docs, nil
}
