package app

// gaea 自动做梦（空闲自整理）：会话轮次成功结束后，后台把本轮对话归纳为
// 可长期记忆的事实/笔记，写入主脑记忆（memory.Store，按 name 去重）。
// 对标 WPS 灵犀「自动做梦」、Codex 后台会话总结。
//
// 写入纪律（对齐调研结论）：
//   - 只提炼稳定事实与偏好，不做实时逐句记录（Kimi 二问思路：先过滤再入库）；
//   - 同轮只跑一次、有实质内容才跑、单飞（并发安全）；
//   - 归纳失败静默跳过，不打扰用户。
//
// 审批决策（T6-8.1）：后台自动做梦**不**走 hardAskTools 逐条审批——异步
// goroutine 无法等待人工确认，且 /dream extract 与记忆建议接受本身就是用户
// 显式触发。补偿机制：每次实际写入都经 SaveDreamFacts(source, …) 落审计
// 日志（source=auto_dream|explicit，条数+名称），全程可追溯。详见
// docs/DREAM_WRITE_POLICY.md。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// gaeaDreamState 单飞状态：同一时刻最多一个整理任务在跑。
var gaeaDreamState struct {
	sync.Mutex
	running bool
	last    time.Time
	// lastHash 是上次**成功处理**的轮次输入指纹（C3 no-op 优化，对齐 codex
	// memories/write phase2 的「无变化即 no-op 成功，省一次 LLM 调用」）：内容
	// 相同的轮次（重试/重入/同内容连问）不再重复提炼。仅内存态——跨重启不
	// 持久化，恢复会话后的首次整理宁可多跑一次（记忆库可能已变化）。
	lastHash string
}

// dreamSystemPrompt 记忆整理提示词：只输出 JSON，控制条数与类型。
const dreamSystemPrompt = `你是 gaea 的记忆整理员。给你一段刚结束的对话，请从中提炼**值得长期记住**的信息。

判定标准（Kimi 二问）：
1. 这条信息以后还会用到吗？（用户身份/偏好/项目事实/踩过的坑/可复用方法 → 值得记）
2. 它稳定吗？（一次性、临时、会话内细节 → 不记）

输出规则：
- 只输出一个 JSON 对象，不要任何解释或代码围栏。
- facts：每条 {name(英文短横线 slug), type(user|project|feedback|reference), kind(semantic|episodic|procedural), description(一句话摘要), body(事实正文，Markdown)}。
- notes：把不适合进事实但值得留档的零散经验写成 {scope(local|user|project), note(一行)}。
- 事实最多 5 条、笔记最多 3 条；宁缺毋滥，没有就输出 {"facts":[],"notes":[]}。

JSON 示例：
{"facts":[{"name":"user-unit","type":"user","kind":"semantic","description":"用户所在单位","body":"用户单位为 XX 公司，负责成本测算"}],"notes":[{"scope":"local","note":"成本测算常用口径：税前"} ]}`

type dreamFact struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

type dreamNote struct {
	Scope string `json:"scope"`
	Note  string `json:"note"`
}

type dreamResult struct {
	Facts []dreamFact `json:"facts"`
	Notes []dreamNote `json:"notes"`
}

// gaeaSessionSpace 返回当前办公会话的空间（S1.2 A dream 空间化的取值单点）：
//   - 引擎已初始化 → ctrl.SessionSpace()（会话路径归属的写入侧空间自描述：
//     "work"/"play"；space.mode=off 平铺形态为 ""=无空间维度）；
//   - 引擎未初始化（事件早于构建完成等）→ 回退配置生效空间 gaeaEffectiveSpace()
//     （mode=off 同样为 ""）。
//
// 返回值保持原样不归一：off 模式的 "" 必须原样传给指纹/写入层（指纹退化为
// 纯内容形态、SaveDreamFacts 写侧统一 Normalize 兜底 work），严格等价旧行为。
func gaeaSessionSpace() string {
	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl != nil {
		return ctrl.SessionSpace()
	}
	return gaeaEffectiveSpace()
}

// maybeDreamAfterTurn 在轮次成功后触发后台「自动做梦」（单飞）。space 是触发
// 会话的空间（gaeaSessionSpace()）：整轮整理（指纹、落库、notes 分流）都限定
// 在该空间内进行，防 play 会话内容写进 work 记忆。
func (a *App) maybeDreamAfterTurn(space string) {
	gaeaDreamState.Lock()
	if gaeaDreamState.running {
		gaeaDreamState.Unlock()
		return
	}
	gaeaDreamState.running = true
	gaeaDreamState.Unlock()

	go func() {
		defer func() {
			gaeaDreamState.Lock()
			gaeaDreamState.running = false
			gaeaDreamState.last = time.Now()
			gaeaDreamState.Unlock()
		}()
		if err := a.runDream(space); err != nil {
			slog.Debug("gaea dream skipped", "err", err)
		}
	}()
}

// runDream 执行一轮记忆整理：取最后一轮对话 → 模型提炼 → 写入长期记忆。
// space 为触发会话的空间（""=space.mode=off 平铺形态，行为与改造前一致）。
func (a *App) runDream(space string) error {
	c := gaeaCtrl()
	if c == nil || c.Memory() == nil {
		return fmt.Errorf("memory unavailable")
	}
	msgs := dreamTurnMessages(c.History())
	if !dreamWorthwhile(msgs) {
		return fmt.Errorf("no worthwhile content")
	}
	input := dreamInput(msgs)
	// S1.2 A：指纹键含会话空间——同内容跨空间不共用指纹（play 会话内容与
	// work 相同时不得误命中 work 指纹而跳过提炼）；space 为空（mode=off）时
	// 键内无空间成分，等价纯内容指纹（旧行为）。
	hash := dreamInputHash(space, input)
	gaeaDreamState.Lock()
	dup := hash == gaeaDreamState.lastHash
	gaeaDreamState.Unlock()
	if dup {
		// C3 no-op：本轮内容与上次成功整理的完全一致，提炼结论不会变化，
		// 直接跳过（省一次 LLM 调用；codex phase2「无变化即成功」同型）。
		return fmt.Errorf("dream no-op: 本轮内容已整理过")
	}
	if a.client == nil {
		return fmt.Errorf("ai client unavailable")
	}

	// 2026-08-28 本地优先强化：记忆整理属办公功能级调用，优先本地 Herdsman。
	featEng, featModel, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	out, err := a.client.ChatSimpleStreamWithOptions(ctx, featModel, dreamSystemPrompt, input,
		ai.ChatSimpleOptions{EngineID: featEng, Temperature: 0.2, MaxTokens: 1200})
	if err != nil {
		return fmt.Errorf("dream summarize: %w", err)
	}
	res, err := parseDreamOutput(out)
	if err != nil {
		return err
	}

	saved, err := c.SaveDreamFacts(space, "auto_dream", toDreamMemories(res.Facts))
	if err != nil {
		return err
	}
	notes := dreamWriteNotes(c, space, res.Notes)
	if saved > 0 || notes > 0 {
		a.emit("gaea-event", gaeaEventMap(event.Event{
			Kind:  event.Notice,
			Level: event.LevelInfo,
			Text:  fmt.Sprintf("已自动整理记忆：新增 %d 条事实、%d 条笔记", saved, notes),
		}))
	}
	// 完整处理成功（含「模型判定无可记内容」）才记录指纹：写入失败不记录，
	// 下次同内容重试不受影响。
	gaeaDreamState.Lock()
	gaeaDreamState.lastHash = hash
	gaeaDreamState.Unlock()
	return nil
}

// dreamInputHash 返回整理输入的内容指纹（sha256 hex，sha256 无空串歧义）。
// S1.2 A 双空间：指纹键含会话空间（space+"\x00"+input）——同内容跨空间不
// 共用指纹，防「play 会话内容与 work 相同被误判 no-op 跳过提炼」。space 为
// 空（space.mode=off 平铺形态）时键内无空间成分，退化为纯内容指纹，与旧行
// 为严格等价（off 无空间维度）。
func dreamInputHash(space, input string) string {
	sum := sha256.Sum256([]byte(space + "\x00" + input))
	return hex.EncodeToString(sum[:])
}

// dreamNoteWriter 是 dream notes 落盘的最小接口（*control.Controller 满足；
// 测试注入 fake 验证空间分流，无需真实控制器）。
type dreamNoteWriter interface {
	QuickAdd(scope memory.Scope, note string) (string, error)
}

// dreamWriteNotes 把本轮提炼的零散经验笔记落盘，返回写入条数。
//
// 空间分流（S1.2 A，方案决策见 docs/gaea-memory-isolation-design.md §勘误）：
// notes 走 QuickAdd → 项目级 doc 记忆（AGENTS.md 等）——该文件属 **work 项目
// 说明**，play 会话的 dream 不得写它（验收红线「play 会话 dream 不写入 work
// 记忆」）。play 侧直接**丢弃 notes 仅落 facts** 并 slog.Debug 留痕：facts 表
// 已带 space_id 全量保留 dream 提炼内容，零散笔记在 play 域没有对应说明文件，
// 宁丢不跨空间污染 work AGENTS.md。work 路径行为逐字不变。
func dreamWriteNotes(w dreamNoteWriter, space string, notes []dreamNote) int {
	written := 0
	for _, n := range notes {
		note := strings.TrimSpace(n.Note)
		if note == "" {
			continue
		}
		if space == spaces.SpacePlay {
			slog.Debug("dream note dropped (play space, not writing work AGENTS.md)", "note", note)
			continue
		}
		scope := memory.ScopeLocal
		switch n.Scope {
		case "user":
			scope = memory.ScopeUser
		case "project":
			scope = memory.ScopeProject
		}
		if _, err := w.QuickAdd(scope, note); err == nil {
			written++
		}
	}
	return written
}

// dreamTurnMessages 取会话历史最后一轮（最后一个 user 消息起）的 user/assistant 消息。
func dreamTurnMessages(hist []provider.Message) []provider.Message {
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == provider.RoleUser {
			out := make([]provider.Message, 0, len(hist)-i)
			for _, m := range hist[i:] {
				if m.Role == provider.RoleUser || m.Role == provider.RoleAssistant {
					out = append(out, m)
				}
			}
			return out
		}
	}
	return nil
}

// dreamWorthwhile 判断本轮对话是否值得整理：至少一问一答，且助手输出够实质。
func dreamWorthwhile(msgs []provider.Message) bool {
	if len(msgs) < 2 {
		return false
	}
	assistantLen := 0
	for _, m := range msgs {
		if m.Role == provider.RoleAssistant {
			assistantLen += len(m.Content)
		}
	}
	return assistantLen >= 100
}

// dreamInput 把消息拼成整理输入（截断超长内容，控制 token）。
func dreamInput(msgs []provider.Message) string {
	var b strings.Builder
	b.WriteString("以下是刚结束的一轮对话，请提炼值得长期记住的信息。\n\n")
	for _, m := range msgs {
		content := m.Content
		if len(content) > 1500 {
			content = content[:1500] + "…"
		}
		b.WriteString("【" + string(m.Role) + "】\n" + content + "\n\n")
	}
	return b.String()
}

// parseDreamOutput 解析模型输出：剥离代码围栏/前后缀，容错 JSON。
func parseDreamOutput(out string) (dreamResult, error) {
	var res dreamResult
	s := strings.TrimSpace(out)
	if i := strings.Index(s, "{"); i >= 0 {
		s = s[i:]
	}
	if j := strings.LastIndex(s, "}"); j >= 0 {
		s = s[:j+1]
	}
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return res, fmt.Errorf("dream: 解析记忆提炼结果失败: %w", err)
	}
	if len(res.Facts) > 5 {
		res.Facts = res.Facts[:5]
	}
	if len(res.Notes) > 3 {
		res.Notes = res.Notes[:3]
	}
	clean := res.Facts[:0]
	for _, f := range res.Facts {
		f.Name = strings.TrimSpace(f.Name)
		f.Description = strings.TrimSpace(f.Description)
		f.Body = strings.TrimSpace(f.Body)
		if f.Name == "" || (f.Description == "" && f.Body == "") {
			continue
		}
		clean = append(clean, f)
	}
	res.Facts = clean
	return res, nil
}

func toDreamMemories(facts []dreamFact) []memory.Memory {
	out := make([]memory.Memory, 0, len(facts))
	for _, f := range facts {
		out = append(out, memory.Memory{
			Name:        f.Name,
			Type:        memory.NormalizeType(f.Type),
			Kind:        memory.NormalizeKind(f.Kind),
			Description: f.Description,
			Body:        f.Body,
		})
	}
	return out
}
