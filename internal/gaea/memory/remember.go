package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// rememberTool 是模型侧的记忆写入口。2026-11-25 办公工具梳理批把五个记忆
// 工具收成两个：本工具（存 / 删 / 转正，action 区分）+ memory_search（检索 /
// 按名读正文）。forget、promote_session_facts 退役——68 个真实办公会话里
// 两者 0 次调用，却各占一份常驻 schema。
//
// 它是有状态的（绑定一个工作区的 Store 视图），由 boot 构造后加进注册表——
// 与 task 工具同一模式，不走无状态 builtin 自注册。
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
	//
	// 2026-11-25 办公工具梳理批：描述从 1782 字符的英文全文换成中文要点
	// （造价/方案档的身份提示词要求全中文，记忆工具却是最大的一段英文）；
	// 行为纪律（Why/How 结构、近重复检查）留在正文与技能里，这里只留字段
	// 该装什么。
	return "记忆的存、删、转正（本项目工作区；带运行模式归属）。" +
		"默认 action=save：存一条跨会话的事实——值得长期保留的才记，瞬时状态和用户明令不存的不要记；执行时弹确认卡，用户确认才落盘。" +
		"type：user=用户是谁与偏好；feedback=工作方式（正文写清 **Why:** 与 **How to apply:**）；project=项目目标与约束；reference=外部资源指针。" +
		"kind：semantic=事实；episodic=带触发词的过往经验；procedural=常驻规则（每轮注入，只写真正不变的纪律）。" +
		"description 是索引里的一行钩子，要具体。name 用 kebab-case，沿用已有名称=更新那一条，不要另建近重复；单值事实给 subject_key。" +
		"scope=归属：general=通用（各运行模式都注入）；mode=仅当前模式；不给则按默认（user/feedback 通用，其余归当前模式；按名修订已有记忆不改归属）。" +
		"session=true 只在本会话有效，之后用 action=promote 转正。" +
		"action=forget（给 name）：删掉已过时或错误的一条；action=promote：把本会话的临时记忆转正。"
}

func (rememberTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type": "object",
"properties": {
  "action": {"type": "string", "enum": ["save", "forget", "promote"], "description": "save（默认）=存一条；forget=按 name 删一条；promote=把本会话的临时记忆转正。"},
  "scope": {"type": "string", "enum": ["general", "mode"], "description": "归属：general=各运行模式通用；mode=仅当前运行模式。不给=按默认规则（user/feedback 通用，其余归当前模式）。"},
  "name": {"type": "string", "description": "Short kebab-case slug identifying the fact, e.g. \"prefers-tabs\". Reusing a name overwrites that memory — do that to update an existing fact. Omit to derive one from the description. forget 时必填。"},
  "title": {"type": "string", "description": "Short human-readable label shown in the memory index, e.g. \"Prefers Tabs\". Omit to derive one from the name."},
  "description": {"type": "string", "description": "One-line hook shown in the index — the phrase a future session reads to decide whether to open this memory. Make it specific."},
  "type": {"type": "string", "enum": ["user", "feedback", "project", "reference"], "description": "Category of the fact."},
  "kind": {"type": "string", "enum": ["semantic", "episodic", "procedural"], "description": "Cognitive function. semantic (default): facts/prefs searchable. episodic: past experiences with trigger tags. procedural: always-active rules."},
  "tags": {"type": "array", "items": {"type": "string"}, "description": "Trigger keywords for episodic memories. When user input matches, the memory is injected as a few-shot example."},
  "subject_key": {"type": "string", "description": "Optional dotted key naming the single-valued question this fact answers, e.g. \"project.package_manager\" or \"user.response_style\". One active value per subject per space: saving against a held subject is rejected with the holder's name — update that memory instead. Omit for narrative facts that are not a single right answer."},
  "body": {"type": "string", "description": "The fact itself (Markdown). For episodic, use pattern: observation -> action -> result."},
  "session": {"type": "boolean", "description": "If true, save to session-only memory (not permanent)."}
}
}`)
}

func (t rememberTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Action      string   `json:"action"`
		Scope       string   `json:"scope"`
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
	switch strings.ToLower(strings.TrimSpace(in.Action)) {
	case "", "save":
		return t.save(ctx, rememberSaveArgs{
			Scope: in.Scope, Name: in.Name, Title: in.Title, Description: in.Description,
			Type: in.Type, Kind: in.Kind, Tags: in.Tags, SubjectKey: in.SubjectKey,
			Body: in.Body, Session: in.Session,
		}, args)
	case "forget":
		if strings.TrimSpace(in.Name) == "" {
			return "", fmt.Errorf("action=forget 需要 name（要删的记忆名，取自索引里的方括号名称）")
		}
		if _, err := t.store.Archive(in.Name); err != nil {
			return "", err
		}
		if q, ok := QueueFromContext(ctx); ok {
			q.QueueMemory("Deleted memory \"" + slug(in.Name) + "\" — disregard its line still shown in the saved-memories index until next session.")
		}
		return fmt.Sprintf("已删除记忆 %q（不再生效，之后的会话不再加载）。", in.Name), nil
	case "promote":
		p, ok := PromoterFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("session promote unavailable in this context")
		}
		n, err := p.PromoteSessionFacts()
		if err != nil {
			return "", fmt.Errorf("promote failed: %w", err)
		}
		if n == 0 {
			return "本会话没有待转正的临时记忆。", nil
		}
		return fmt.Sprintf("已把 %d 条临时记忆转正为永久记忆。", n), nil
	default:
		return "", fmt.Errorf("未知 action %q（save / forget / promote）", in.Action)
	}
}

type rememberSaveArgs struct {
	Scope       string
	Name        string
	Title       string
	Description string
	Type        string
	Kind        string
	Tags        []string
	SubjectKey  string
	Body        string
	Session     bool
}

func (t rememberTool) save(ctx context.Context, in rememberSaveArgs, raw json.RawMessage) (string, error) {
	if in.Description == "" || in.Body == "" {
		// 缺参回显收到的键（铜陵反馈 #16/#19 同款：模型反复缺同一批键时，
		// 键面让它立刻看出哪个键没发出去）。
		var top map[string]json.RawMessage
		_ = json.Unmarshal(raw, &top)
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
	// 运行模式归属（mode.go）：显式 scope 照办；没给 scope 时，按名修订已有
	// 记忆不改它的归属，新记忆按默认规则。
	sessionMode := t.store.SessionMode()
	if strings.TrimSpace(in.Scope) != "" {
		m.Mode = resolveRememberMode(in.Scope, m.Type, sessionMode)
	} else if prev, ok := t.store.Get(slug(name)); ok {
		if strings.TrimSpace(prev.Mode) == "" {
			m.Mode = ModeGeneral
		} else {
			m.Mode = prev.Mode
		}
	} else {
		m.Mode = resolveRememberMode("auto", m.Type, sessionMode)
	}
	// 模式维度关闭的存储没有视图替我们归一：这里把「显式通用」的内存态标记
	// 清成落库值。维度开启时留给 modeView.Save——提前清掉会让它把「显式通用」
	// 当成「没说」而重新盖当前模式。
	if sessionMode == "" {
		normalizeModeForSave(&m, sessionMode)
	}

	// Session-only save: store in-memory, not to disk.
	if in.Session {
		normalizeModeForSave(&m, sessionMode)
		ss, ok := SessionSaverFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("session memory unavailable")
		}
		note := ss.SaveSession(m)
		if q, ok := QueueFromContext(ctx); ok {
			q.QueueMemory(note)
		}
		return fmt.Sprintf("已存为临时记忆（\"%s\"），只在本会话生效；要跨会话保留用 action=promote 转正。", slug(name)), nil
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
		return fmt.Sprintf("已存入全局用户画像（\"%s\"，各处共享）：%s", slug(name), oneLine(in.Description)), nil
	}
	path, err := t.store.Save(m)
	if err != nil {
		return "", err
	}
	if q, ok := QueueFromContext(ctx); ok {
		q.QueueMemory("Saved memory \"" + slug(name) + "\": " + oneLine(in.Description))
	}
	where := "之后各会话自动加载"
	if strings.TrimSpace(m.Mode) != "" {
		where = "只在「" + ModeLabel(m.Mode) + "」运行模式加载"
	}
	return fmt.Sprintf("已存记忆（%s；%s）。", path, where), nil
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
		Action      string `json:"action"`
		Description string `json:"description"`
		Body        string `json:"body"`
	}
	_ = json.Unmarshal(args, &got)
	switch strings.ToLower(strings.TrimSpace(got.Action)) {
	case "forget":
		return `remember 正确参数骨架：{"action":"forget","name":"要删的记忆名"}`
	case "promote":
		return `remember 正确参数骨架：{"action":"promote"}`
	}
	hint := `remember 正确参数骨架：{"name":"kebab-case-slug","description":"一行钩子（索引里给未来会话看的那句）","body":"事实本体（feedback/project 用 **Why:** 与 **How to apply:** 结构）","type":"feedback"}`
	if got.Body != "" && got.Description == "" {
		hint += "——本次只差 description：把这条记忆的一句话钩子补上（越具体越好，别复述 body）"
	}
	return hint
}
