package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// rememberTool lets the model persist a durable fact to the auto-memory store.
// It is stateful (bound to one project's Store), so boot constructs it and adds
// it to the registry — the same pattern as the task tool — rather than
// self-registering as a stateless built-in.
type rememberTool struct {
	store   Store
	profile *ProfileStore // 主脑全局画像（type=user 时路由到这里）
}

// NewRememberTool returns the `remember` tool bound to store. When profile is
// non-nil, Type=user facts are routed to the main-brain profile table
// (cross-agent shared) instead of the project facts. A zero/disabled store
// yields a tool that reports the store is unavailable rather than silently
// dropping saves.
func NewRememberTool(store Store, profile *ProfileStore) tool.Tool {
	return rememberTool{store: store, profile: profile}
}

func (rememberTool) Name() string { return "remember" }

func (rememberTool) Description() string {
	return "Save a durable fact to project memory so it survives across sessions. " +
		"Use for things worth remembering long-term: who the user is and their preferences (type \"user\"); " +
		"guidance on how to work, including the why (type \"feedback\"); ongoing goals or constraints not " +
		"derivable from the code (type \"project\"); or pointers to external resources (type \"reference\"). " +
		"For feedback/project, structure the body with a \"**Why:**\" line and a \"**How to apply:**\" line so the fact is actionable later; " +
		"link related memories inline with [[their-name]]. " +
		"Do NOT save what the repo already records (code structure, git history) or facts that only matter to the current conversation; " +
		"if asked to remember one of those, save instead the non-obvious point behind it. " +
		"Before saving, check the loaded memory index for an entry that already covers this — reuse that name to update it rather than create a near-duplicate, and use `forget` to drop one that is now wrong. " +
		"The saved index loads into context at the start of each session. " +
		"Set session=true to save tentatively — the fact is visible this session but not persisted to disk until you call promote_session_facts. " +
		"Kind: \"semantic\" (default, facts/prefs), \"episodic\" (past experiences with trigger tags), \"procedural\" (always-active rules)."
}

func (rememberTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type": "object",
"properties": {
  "name": {"type": "string", "description": "Short kebab-case slug identifying the fact, e.g. \"prefers-tabs\". Reusing a name overwrites that memory — do that to update an existing fact. Omit to derive one from the description."},
  "title": {"type": "string", "description": "Short human-readable label shown in the memory index, e.g. \"Prefers tabs\". Omit to derive one from the name."},
  "description": {"type": "string", "description": "One-line hook shown in the index — the phrase a future session reads to decide whether to open this memory. Make it specific."},
  "type": {"type": "string", "enum": ["user", "feedback", "project", "reference"], "description": "Category of the fact."},
  "kind": {"type": "string", "enum": ["semantic", "episodic", "procedural"], "description": "Cognitive function. semantic (default): facts/prefs searchable. episodic: past experiences with trigger tags. procedural: always-active rules."},
  "tags": {"type": "array", "items": {"type": "string"}, "description": "Trigger keywords for episodic memories. When user input matches, the memory is injected as a few-shot example."},
  "body": {"type": "string", "description": "The fact itself (Markdown). For episodic, use pattern: observation -> action -> result."},
  "session": {"type": "boolean", "description": "If true, save to session-only memory (not permanent)."}
},
"required": ["description", "body"]
}`)
}

func (t rememberTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Name        string   `json:"name"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Type        string   `json:"type"`
		Kind        string   `json:"kind"`
		Tags        []string `json:"tags"`
		Body        string   `json:"body"`
		Session     bool     `json:"session"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Description == "" || in.Body == "" {
		return "", fmt.Errorf("description and body are required")
	}
	name := in.Name
	if name == "" {
		name = in.Title // Save slugifies; the title (or, below, the description) makes a serviceable slug
	}
	if name == "" {
		name = in.Description
	}
	// S1.2 A 写侧空间化：从调用 ctx 取会话空间（executeOne 注入点盖章；无
	// 标注缺省 work）——play 会话的 remember 落 play、work 落 work，修复
	// 「play 会话记忆默认落 work」的写侧泄漏（设计 §写侧漏洞 2）。
	m := Memory{
		Name:        name,
		Title:       in.Title,
		Description: in.Description,
		Type:        NormalizeType(in.Type),
		Kind:        NormalizeKind(in.Kind),
		Tags:        in.Tags,
		Body:        in.Body,
		Space:       SpaceFromContext(ctx),
	}

	// Session-only save: store in-memory, not to disk.
	if in.Session {
		ss, ok := SessionSaverFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("session memory unavailable")
		}
		note := ss.SaveSession(m)
		if q, ok := QueueFromContext(ctx); ok {
			q.QueueMemory(note)
		}
		return fmt.Sprintf("Saved to session memory (\"%s\"). It applies this session. Call promote_session_facts to make it permanent.", slug(name)), nil
	}

	// Permanent save: Type=user routes to the main-brain profile (cross-agent
	// shared user portrait); everything else stays in the project facts store.
	if t.profile != nil && NormalizeType(in.Type) == TypeUser {
		if err := t.profile.Save(m); err != nil {
			return "", err
		}
		if q, ok := QueueFromContext(ctx); ok {
			q.QueueMemory("Saved user profile \"" + slug(name) + "\": " + oneLine(in.Description))
		}
		return fmt.Sprintf("Saved to global user profile (\"%s\", shared across agents): %s", slug(name), oneLine(in.Description)), nil
	}
	path, err := t.store.Save(m)
	if err != nil {
		return "", err
	}
	if q, ok := QueueFromContext(ctx); ok {
		q.QueueMemory("Saved memory \"" + slug(name) + "\": " + oneLine(in.Description))
	}
	return fmt.Sprintf("Saved memory to %s (it applies now and loads automatically in future sessions).", path), nil
}

func (rememberTool) ReadOnly() bool    { return false }
func (rememberTool) PersistWrite() bool { return true }
