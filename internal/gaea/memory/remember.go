package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
	// 口径回调（2026-11-21 用户拍板：auto-dream 删除后 remember 是唯一模型侧
	// 记忆入口，「只有用户明确要求才调用」过紧——模型实际停止主动记任何
	// 东西。恢复「值得记就调」，确认制改由确认卡承载（work 空间执行时弹
	// 用户确认卡，用户点确认才落盘=「未经确认不写」红线仍在）；瞬时状态
	// 与用户明令不保存的内容依旧不记。
	return "Save a durable fact to project memory so it survives across sessions. " +
		"When you learn something worth keeping long-term, call this proactively — a user confirmation card " +
		"is shown before anything is written (the user's approval IS the consent step), so do not withhold " +
		"memories waiting to be asked. Do not log transient state or anything the user asked not to save. " +
		"Use for things worth remembering long-term: who the user is and their preferences (type \"user\"); " +
		"guidance on how to work, including the why (type \"feedback\"); ongoing goals or constraints not " +
		"derivable from the code (type \"project\"); or pointers to external resources (type \"reference\"). " +
		"For feedback/project, structure the body with a \"**Why:**\" line and a \"**How to apply:**\" line so the fact is actionable later; " +
		"link related memories inline with [[their-name]]. " +
		"Do NOT save what the repo already records (code structure, git history) or facts that only matter to the current conversation; " +
		"if asked to remember one of those, save instead the non-obvious point behind it. " +
		"Before saving, check the loaded memory index for an entry that already covers this — reuse that name to update it rather than create a near-duplicate, and use `forget` to drop one that is now wrong. " +
		"For single-valued facts (one right answer per question, like the package manager or the release branch), set subject_key — saving against a held subject is rejected so you update the holder instead of contradicting it. " +
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
  "subject_key": {"type": "string", "description": "Optional dotted key naming the single-valued question this fact answers, e.g. \"project.package_manager\" or \"user.response_style\". One active value per subject per space: saving against a held subject is rejected with the holder's name — update that memory instead. Omit for narrative facts that are not a single right answer."},
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
		SubjectKey  string   `json:"subject_key"`
		Body        string   `json:"body"`
		Session     bool     `json:"session"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Description == "" || in.Body == "" {
		// 缺参回显收到的键（铜陵反馈 #16/#19 同款：模型反复缺同一批键时，
		// 键面让它立刻看出哪个键没发出去）。
		var top map[string]json.RawMessage
		_ = json.Unmarshal(args, &top)
		keys := make([]string, 0, len(top))
		for k := range top {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		missing := "description"
		if in.Description != "" {
			missing = "body"
		}
		return "", fmt.Errorf("%s is required（本次收到键：%s——缺 %s 没送达，多半没发出去或发进了别的键；description=索引里的一行钩子）", missing, strings.Join(keys, ", "), missing)
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
		SubjectKey:  NormalizeSubjectKey(in.SubjectKey),
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

func (rememberTool) ReadOnly() bool     { return false }
func (rememberTool) PersistWrite() bool { return true }

// KeepSchemaDescriptions 记忆字段（description 一行钩子 / body Why-How 结构 /
// subject_key 单值语义）的语义只写在字段描述里（铜陵反馈 #19：剥掉描述后模型
// 连发 4 次「只有 body 没有 description」——它看不见每个字段该装什么）。
func (rememberTool) KeepSchemaDescriptions() bool { return true }

// UsageHint 缺参时给一张正确骨架（loop guard 附加值，与 cost_library 同款；
// 2026-11-25 铜陵反馈 #19 实测缺 description 连发 3 次原样撞）。
func (rememberTool) UsageHint(args json.RawMessage) string {
	var got struct {
		Description string `json:"description"`
		Body        string `json:"body"`
	}
	_ = json.Unmarshal(args, &got)
	hint := `remember 正确参数骨架：{"name":"kebab-case-slug","description":"一行钩子（索引里给未来会话看的那句）","body":"事实本体（feedback/project 用 **Why:** 与 **How to apply:** 结构）","type":"feedback"}`
	if got.Body != "" && got.Description == "" {
		hint += "——本次只差 description：把这条记忆的一句话钩子补上（越具体越好，别复述 body）"
	}
	return hint
}
