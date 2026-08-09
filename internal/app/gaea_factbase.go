package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/factbase"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// FactView is one fact in the fact base, as seen by the frontend panel.
type FactView struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Source    string `json:"source,omitempty"`
	Category  string `json:"category,omitempty"`
	UpdatedAt int64  `json:"updatedAt"`
}

// FactBaseView is the whole sidebar panel view: facts + copy-ready Markdown.
type FactBaseView struct {
	Facts    []FactView `json:"facts"`
	Markdown string     `json:"markdown"`
	Count    int        `json:"count"`
	Path     string     `json:"path"`
}

// gaeaFactBaseStore returns the fact-base store for the current session, or
// nil when the office engine is not ready.
func gaeaFactBaseStore() *factbase.Store {
	c := gaeaCtrl()
	if c == nil {
		return nil
	}
	return factbase.NewStore(factbase.PathFor(c.SessionPath()))
}

// factBasePath returns the absolute path of the current session fact base.
func factBasePath() string {
	c := gaeaCtrl()
	if c == nil {
		return ""
	}
	return factbase.PathFor(c.SessionPath())
}

// factBaseSnapshot reads the current session fact base into a frontend view.
func factBaseSnapshot() FactBaseView {
	view := FactBaseView{Facts: []FactView{}}
	st := gaeaFactBaseStore()
	if st == nil {
		return view
	}
	b, err := st.Snapshot()
	if err != nil {
		return view
	}
	for _, f := range b.Sorted() {
		view.Facts = append(view.Facts, FactView{
			Key: f.Key, Value: f.Value, Source: f.Source, Category: f.Category,
			UpdatedAt: f.UpdatedAt.UnixMilli(),
		})
	}
	view.Markdown = b.Markdown()
	view.Count = len(view.Facts)
	view.Path = factBasePath()
	return view
}

// factAddTool records one settled fact (upsert by key). It is an ExtraTool so
// it resolves the current session's fact-base file at call time.
type factAddTool struct{}

func (factAddTool) Name() string { return "fact_add" }

func (factAddTool) Description() string {
	return "Record one settled fact into the conversation's fact base (upsert by key, empty value removes). Use for every deliverable task (report/plan/PPT/spreadsheet): collect the confirmed numbers, units, deadlines, scope and sources FIRST, then build outputs from fact_list so every deliverable stays consistent. Optional category: project/data/preference/other."
}

func (factAddTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "key":{"type":"string","description":"Fact name, e.g. construction-period"},
  "value":{"type":"string","description":"Fact value, e.g. 90 calendar days"},
  "source":{"type":"string","description":"Where it came from, e.g. tender doc P3"},
  "category":{"type":"string","enum":["project","data","preference","other"],"description":"Optional category; default project"}
},
"required":["key","value"]
}`)
}

func (factAddTool) ReadOnly() bool { return false }

func (factAddTool) CompactDescription() string {
	return "Record one settled fact into the fact base (upsert by key)."
}

func (factAddTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"},"value":{"type":"string"},"source":{"type":"string"},"category":{"type":"string"}},"required":["key","value"]}`)
}

func (factAddTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		Source   string `json:"source"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Key) == "" {
		return "", fmt.Errorf("key is required")
	}
	st := gaeaFactBaseStore()
	if st == nil {
		return "", fmt.Errorf("office engine not ready")
	}
	if err := st.Add(p.Key, strings.TrimSpace(p.Value), strings.TrimSpace(p.Source), strings.TrimSpace(p.Category)); err != nil {
		return "", err
	}
	b, _ := st.Snapshot()
	return fmt.Sprintf("Fact base now has %d facts. Deliverables must be built from the fact base (see fact_list).", len(b.Facts)), nil
}

// factListTool renders the current fact base for the model and the user.
type factListTool struct{}

func (factListTool) Name() string { return "fact_list" }

func (factListTool) Description() string {
	return "List the conversation's fact base as a Markdown table (key, value, source, category). Read it before generating any deliverable so every output uses the same settled facts."
}

func (factListTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func (factListTool) ReadOnly() bool { return true }

func (factListTool) CompactDescription() string {
	return "List the fact base as a Markdown table."
}

func (factListTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (factListTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	st := gaeaFactBaseStore()
	if st == nil {
		return "", fmt.Errorf("office engine not ready")
	}
	md, err := st.Markdown()
	if err != nil {
		return "", err
	}
	if md == "" {
		return "The fact base is empty: no settled facts yet. Record facts with fact_add before producing deliverables.", nil
	}
	return md, nil
}

// factClearTool resets the conversation's fact base for a new task.
type factClearTool struct{}

func (factClearTool) Name() string { return "fact_clear" }

func (factClearTool) Description() string {
	return "Clear the conversation's fact base. Use when starting a fresh task that should not inherit facts from the previous one."
}

func (factClearTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func (factClearTool) ReadOnly() bool { return false }

func (factClearTool) CompactDescription() string {
	return "Clear the fact base."
}

func (factClearTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (factClearTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	st := gaeaFactBaseStore()
	if st == nil {
		return "", fmt.Errorf("office engine not ready")
	}
	if err := st.Clear(); err != nil {
		return "", err
	}
	return "Fact base cleared.", nil
}

// GaeaFactBase returns the current session fact base for the sidebar panel.
func (a *App) GaeaFactBase() FactBaseView {
	return factBaseSnapshot()
}

// GaeaFactBaseClear empties the current session fact base from the panel.
func (a *App) GaeaFactBaseClear() error {
	st := gaeaFactBaseStore()
	if st == nil {
		return nil
	}
	return st.Clear()
}

// GaeaFactBasePromote writes the current session fact base into permanent
// memory, so settled facts survive across sessions. Deduplicated by name:
// re-promoting the same key updates the existing memory instead of duplicating.
// Returns the number of facts promoted.
func (a *App) GaeaFactBasePromote() (int, error) {
	c := gaeaCtrl()
	if c == nil {
		return 0, fmt.Errorf("office engine not ready")
	}
	mem := c.Memory()
	if mem == nil || mem.UserDir == "" {
		return 0, fmt.Errorf("memory store not available")
	}
	st := gaeaFactBaseStore()
	if st == nil {
		return 0, fmt.Errorf("office engine not ready")
	}
	b, err := st.Snapshot()
	if err != nil {
		return 0, err
	}
	return promoteFactsToMemory(mem.Store, b.Sorted())
}

// promoteFactsToMemory saves fact-base facts into a memory store as semantic
// memories. Name is a stable ASCII slug (hash fallback for CJK keys) so
// re-promoting the same key updates in place; Type follows the category
// (preference → user, everything else → project).
func promoteFactsToMemory(store memory.Store, facts []factbase.Fact) (int, error) {
	n := 0
	for _, f := range facts {
		name := factMemoryName(f.Key)
		body := fmt.Sprintf("**%s**：%s", f.Key, strings.TrimSpace(f.Value))
		if s := strings.TrimSpace(f.Source); s != "" {
			body += "\n\n来源：" + s
		}
		m := memory.Memory{
			Name:        name,
			Title:       f.Key,
			Description: oneLineSummary(f.Value),
			Type:        factMemoryType(f.Category),
			Kind:        memory.KindSemantic,
			Body:        body,
		}
		if _, err := store.Save(m); err != nil {
			return n, fmt.Errorf("promoting fact %q: %w", f.Key, err)
		}
		n++
	}
	return n, nil
}

// factMemoryName maps a fact key to a filesystem-safe kebab name. Pure-CJK keys
// (e.g. 工期) fall back to a stable hash so the memory keeps a valid stem.
func factMemoryName(key string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(key)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name != "" {
		return name
	}
	sum := sha1.Sum([]byte(strings.TrimSpace(key)))
	return "fact-" + hex.EncodeToString(sum[:4])
}

// factMemoryType maps a fact-base category to a memory type.
func factMemoryType(category string) memory.Type {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "preference", "user":
		return memory.TypeUser
	default:
		return memory.TypeProject
	}
}

// oneLineSummary collapses whitespace and caps a value for the memory index.
func oneLineSummary(v string) string {
	s := strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if r := []rune(s); len(r) > 80 {
		s = string(r[:80]) + "…"
	}
	return s
}
