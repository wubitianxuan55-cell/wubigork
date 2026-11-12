package contextview

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 跨会话洞察（对标 dsh-context Context Insights 页）：KPI 栏、逐日用量、
// 活动热力图、技能/工具/命令三统计、会话卡片。BuildInsights 是纯函数——
// 输入单会话折叠结果（FoldTimeline 产物）+元数据，输出聚合快照；逐日归因
// 按「请求发起日」（dsh 同口径：不是会话最后活跃日）。I/O 与缓存在对称的
// App 绑定层（GaeaContextInsights）。

const (
	InsightsToday = "today"
	Insights7d    = "7d"
	Insights30d   = "30d"
	InsightsAll   = "all"
	// InsightsCustomPrefix 是自定义日期范围前缀（custom:起:止，含止日全天；
	// 1:1 对齐上游日期范围选择器）。
	InsightsCustomPrefix = "custom:"
)

// maxInsightTallies 是三统计行的展示上限（按次数取 Top）。
const maxInsightTallies = 12

// ContextInsights 是洞察页的整值快照（GaeaContextInsights 返回值）。
type ContextInsights struct {
	Ok          bool              `json:"ok"`
	Range       string            `json:"range"`
	GeneratedAt int64             `json:"generatedAt"`
	Kpi         InsightsKpi       `json:"kpi"`
	Days        []InsightsDay     `json:"days"`   // 按日升序，仅含范围内有活动的日子
	Skills      []InsightsTally   `json:"skills"` // 技能加载 Top（run_skill）
	Tools       []InsightsTally   `json:"tools"`  // 工具调用 Top
	Programs    []InsightsTally   `json:"programs"`
	Sessions    []InsightsSession `json:"sessions"`
	// Scanned 是扫描的会话总数，Truncated 是因上限未折叠的会话数（诚实边界）。
	Scanned   int `json:"scanned"`
	Truncated int `json:"truncated,omitempty"`
}

// InsightsKpi 是 KPI 栏。Tokens/Cost/Requests/缓存命中率按范围内请求归因；
// Turns 按范围内请求的轮次去重；ToolCalls/Skills 按节点事件日归因；
// ToolsMs/WallMs 是会话生命周期合计（日志不支持按日切分，dsh 同口径：
// 「统计这些会话及其子 Agent 的全部生命周期」）。
type InsightsKpi struct {
	Sessions        int     `json:"sessions"`
	Tokens          int64   `json:"tokens"`
	Requests        int     `json:"requests"`
	Turns           int     `json:"turns"`
	ToolCalls       int     `json:"toolCalls"`
	Skills          int     `json:"skills"`
	Cost            float64 `json:"cost,omitempty"`
	CacheHitPercent float64 `json:"cacheHitPercent,omitempty"`
	ToolsMs         int64   `json:"toolsMs,omitempty"`
	WallMs          int64   `json:"wallMs,omitempty"`
	// 1:1 补齐（对标上游 KPI 卡副行）：Archived=归档会话数、Billed=已计价
	// 会话数（有 usage 的）、Calls=模型调用次数、ToolMsNote 同 ToolsMs。
	Archived int `json:"archived,omitempty"`
	Billed   int `json:"billed,omitempty"`
	Calls    int `json:"calls,omitempty"`
	Steps    int `json:"steps,omitempty"`
	// 耗时统计卡行数据（1:1 上游 Timing Stats：模型等待/模型生成）。
	TTFTMs int64 `json:"ttftMs,omitempty"`
	GenMs  int64 `json:"genMs,omitempty"`
	// Token 统计卡三拆合计（模型思考/回复/调用参数——1:1 对齐上游 Token
	// Stats 的 Reasoning/Answers 行；Tool Calls 行=tool+callArgs 由前端并）。
	AssistantReasoning int64 `json:"assistantReasoning,omitempty"`
	AssistantAnswers   int64 `json:"assistantAnswers,omitempty"`
	AssistantCallArgs  int64 `json:"assistantCallArgs,omitempty"`
	OutputTokens       int64 `json:"outputTokens,omitempty"`
	// EstimatedTokens 是无 usage 提供方的估算构成合计（诚实分列，不混入
	// 计费 Tokens——两口径混排会让「计费」失真）。
	EstimatedTokens int64 `json:"estimatedTokens,omitempty"`
}

// InsightsDay 是一天的活动聚合（7 天柱图/8 周热力图数据源）。
type InsightsDay struct {
	Date      string  `json:"date"` // 本地时区 YYYY-MM-DD
	Tokens    int64   `json:"tokens"`
	Requests  int     `json:"requests"`
	Turns     int     `json:"turns,omitempty"`
	ToolCalls int     `json:"toolCalls,omitempty"`
	Skills    int     `json:"skills,omitempty"`
	Cost      float64 `json:"cost,omitempty"`
	// Sessions 是当日活跃会话数（活跃度卡「会话|Token 消耗」toggle 的会话档）。
	Sessions int `json:"sessions,omitempty"`
}

// InsightsTally 是一条统计行（名称+次数+最近一次时间，点击可过滤会话）。
type InsightsTally struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Last  int64  `json:"last,omitempty"`
}

// InsightsSession 是会话卡片（范围内数字；无范围内活动不入选）。
type InsightsSession struct {
	Path       string `json:"path"`
	Title      string `json:"title,omitempty"`
	Preview    string `json:"preview,omitempty"`
	SpaceID    string `json:"spaceId,omitempty"`
	Archived   bool   `json:"archived,omitempty"`
	ModTime    int64  `json:"modTime"`
	LastActive int64  `json:"lastActive"`
	Tokens     int64  `json:"tokens"`
	Requests   int    `json:"requests"`
	Turns      int    `json:"turns"`
	// LastMessage 是最后一条用户消息 brief（会话卡「最后消息」行；对齐上游）。
	LastMessage string `json:"lastMessage,omitempty"`
	// Steps 是范围内步数（会话卡「N 轮 M 步」的 M）。
	Steps           int      `json:"steps,omitempty"`
	Cost            float64  `json:"cost,omitempty"`
	CacheHitPercent float64  `json:"cacheHitPercent,omitempty"`
	Composition     Category `json:"composition"`
	// 名称清单（v4.542 刀8，去重保序；洞察页统计行点击过滤会话的依据）：
	// Skills=run_skill 技能名、Tools=产生过工具结果的工具名、Programs=命令
	// 程序名。空=该会话在范围内无对应事件。
	Skills   []string `json:"skills,omitempty"`
	Tools    []string `json:"tools,omitempty"`
	Programs []string `json:"programs,omitempty"`
	// Estimated 是该会话在范围内全部请求均为估算关闭（无 usage 提供方/旧
	// 日志）——前端标「估算」角标。
	Estimated bool `json:"estimated,omitempty"`
}

// InsightsSessionFold 是聚合输入：单会话折叠结果 + 列表元数据（App 层组装，
// 缓存单元）。
type InsightsSessionFold struct {
	Path     string
	Title    string
	Preview  string
	SpaceID  string
	Archived bool
	ModTime  int64
	Timeline ContextTimeline
}

// insightCutoff 返回 rangeKey 的起点时间戳（秒）。today=本地今日零点；
// 7d/30d=整日粒度回溯（含今日共 N 天）；all=0；custom:YYYY-MM-DD:YYYY-MM-DD
// =自定义起止（1:1 对齐上游日期范围选择器，终点日含当日全天）。未知 key 按 30d 兜底。
func insightCutoff(rangeKey string, now time.Time) (int64, int64) {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if strings.HasPrefix(rangeKey, InsightsCustomPrefix) {
		parts := strings.SplitN(strings.TrimPrefix(rangeKey, InsightsCustomPrefix), ":", 2)
		from, err1 := time.ParseInLocation("2006-01-02", parts[0], now.Location())
		to := now
		if len(parts) == 2 {
			if t2, err2 := time.ParseInLocation("2006-01-02", parts[1], now.Location()); err2 == nil {
				to = t2.AddDate(0, 0, 1) // 终点日含全天
			}
		}
		if err1 == nil {
			return from.Unix(), to.Unix()
		}
		return midnight.AddDate(0, 0, -29).Unix(), 1 << 62
	}
	switch rangeKey {
	case InsightsToday:
		return midnight.Unix(), 1 << 62
	case Insights7d:
		return midnight.AddDate(0, 0, -6).Unix(), 1 << 62
	case InsightsAll:
		return -1 << 62, 1 << 62 // 语义起点=-∞（首条请求也入选）
	default: // Insights30d 与未知兜底
		return midnight.AddDate(0, 0, -29).Unix(), 1 << 62
	}
}

// dayKey 是本地时区日期键（YYYY-MM-DD）。
func dayKey(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02")
}

// BuildInsights 聚合跨会话洞察。now 由调用方注入（可测试）；cutoff 之前的
// 请求不参与任何数字。会话按范围内 token 降序（并列按最近活动新→旧）。
func BuildInsights(rangeKey string, now time.Time, folds []InsightsSessionFold) ContextInsights {
	cutoff, until := insightCutoff(rangeKey, now)
	out := ContextInsights{
		Ok:          true,
		Range:       rangeKey,
		GeneratedAt: now.Unix(),
		Days:        []InsightsDay{},
		Skills:      []InsightsTally{},
		Tools:       []InsightsTally{},
		Programs:    []InsightsTally{},
		Sessions:    []InsightsSession{},
	}
	type dayAcc struct {
		day    InsightsDay
		turns  map[int]bool
		sesses map[string]bool
	}
	days := map[string]*dayAcc{}
	skillTally := map[string]*InsightsTally{}
	toolTally := map[string]*InsightsTally{}
	progTally := map[string]*InsightsTally{}
	hit, hitMiss := int64(0), int64(0)

	addDay := func(ts int64) *dayAcc {
		k := dayKey(ts)
		d := days[k]
		if d == nil {
			d = &dayAcc{day: InsightsDay{Date: k}, turns: map[int]bool{}, sesses: map[string]bool{}}
			days[k] = d
		}
		return d
	}
	addTally := func(m map[string]*InsightsTally, name string, ts int64) {
		if name == "" {
			return
		}
		t := m[name]
		if t == nil {
			t = &InsightsTally{Name: name}
			m[name] = t
		}
		t.Count++
		if ts > t.Last {
			t.Last = ts
		}
	}

	for _, f := range folds {
		out.Scanned++
		tl := f.Timeline
		sess := InsightsSession{
			Path:     f.Path,
			Title:    f.Title,
			Preview:  f.Preview,
			SpaceID:  f.SpaceID,
			Archived: f.Archived,
			ModTime:  f.ModTime,
		}
		sessTurns := map[int]bool{}
		inRange := false
		estOnly := true
		for i := range tl.Requests {
			r := tl.Requests[i]
			if r.Ts < cutoff || r.Ts >= until {
				continue
			}
			inRange = true
			sessTurns[r.Turn] = true
			sess.Requests++
			sess.Composition = addCategory(sess.Composition, r.Category)
			// 计费口径：有真实 usage 用 prompt+output；否则退估算构成
			// （诚实分列，Kpi.EstimatedTokens 不混入计费 Tokens）。
			if r.PromptTokens > 0 {
				sess.Tokens += r.PromptTokens + r.OutputTokens
				estOnly = false
			} else {
				sess.Tokens += r.Category.Total()
			}
			sess.Cost += r.Cost
			if r.Ts > sess.LastActive {
				sess.LastActive = r.Ts
			}
			if r.BriefUser != "" {
				sess.LastMessage = r.BriefUser
			}
			d := addDay(r.Ts)
			if r.PromptTokens > 0 {
				d.day.Tokens += r.PromptTokens + r.OutputTokens
			} else {
				d.day.Tokens += r.Category.Total()
			}
			d.day.Requests++
			d.day.Cost += r.Cost
			d.turns[r.Turn] = true
			d.sesses[f.Path] = true
			out.Kpi.OutputTokens += r.OutputTokens
			out.Kpi.Requests++
			if r.PromptTokens > 0 {
				out.Kpi.Tokens += r.PromptTokens + r.OutputTokens
			} else {
				out.Kpi.EstimatedTokens += r.Category.Total()
			}
			out.Kpi.Cost += r.Cost
			hit += r.CacheHitTokens
			hitMiss += r.CacheHitTokens + r.CacheMissTokens
		}
		if !inRange {
			continue
		}
		sess.Turns = len(sessTurns)
		sess.Steps = sess.Requests
		sess.Estimated = estOnly
		if f.Archived {
			out.Kpi.Archived++
		}
		if !estOnly {
			out.Kpi.Billed++
		}
		if tl.Timing != nil {
			out.Kpi.Calls += tl.Timing.Calls
			out.Kpi.TTFTMs += tl.Timing.TTFTMs
			out.Kpi.GenMs += tl.Timing.GenMs
		}
		out.Kpi.Steps += sess.Requests
		if tl.Stats.CacheHitPercent > 0 {
			sess.CacheHitPercent = tl.Stats.CacheHitPercent
		}
		// 工具/技能统计：KPI 用 stats 全生命周期计数（dsh 口径：「工具执行
		// 与活跃时长统计会话全部生命周期」）；逐日与 Top 统计行按事件日
		// 归因。旧日志节点无时间戳（Ts=0）：不进逐日，也不进范围过滤的
		// 统计行（无法归因，诚实缺失），range=all 时才计入统计行。
		out.Kpi.ToolCalls += tl.Stats.ToolCalls
		out.Kpi.Skills += tl.Stats.Skills
		inAll := rangeKey == InsightsAll
		sessSkillSet := map[string]bool{}
		sessToolSet := map[string]bool{}
		sessProgSet := map[string]bool{}
		for i := range tl.Nodes {
			n := tl.Nodes[i]
			inRange := n.Ts >= cutoff && n.Ts < until
			switch n.Cat {
			case catTool:
				if inRange && n.Ts > 0 {
					addDay(n.Ts).day.ToolCalls++
				}
				if inRange || inAll {
					addTally(toolTally, n.Tool, n.Ts)
					sessToolSet[n.Tool] = true
				}
			case catSkill:
				if inRange && n.Ts > 0 {
					addDay(n.Ts).day.Skills++
				}
				if inRange || inAll {
					addTally(skillTally, n.Name, n.Ts)
					sessSkillSet[n.Name] = true
				}
			}
		}
		for _, c := range tl.Commands {
			if c.Time < cutoff || c.Time >= until {
				continue
			}
			prog := commandProgram(c.Cmd)
			addTally(progTally, prog, c.Time)
			sessProgSet[prog] = true
		}
		sess.Skills = sortedNameSet(sessSkillSet)
		sess.Tools = sortedNameSet(sessToolSet)
		sess.Programs = sortedNameSet(sessProgSet)
		if tl.Timing != nil {
			out.Kpi.ToolsMs += tl.Timing.ToolsMs
			out.Kpi.WallMs += tl.Timing.WallMs
		}
		out.Kpi.Turns += sess.Turns
		out.Kpi.Sessions++
		out.Kpi.AssistantReasoning += tl.Stats.AssistantReasoning
		out.Kpi.AssistantAnswers += tl.Stats.AssistantAnswerTokens
		out.Kpi.AssistantCallArgs += tl.Stats.AssistantCallTokens
		out.Sessions = append(out.Sessions, sess)
	}

	// 逐日轮次/会话去重计数（聚合完成后统一回填）。
	for _, acc := range days {
		acc.day.Turns = len(acc.turns)
		acc.day.Sessions = len(acc.sesses)
		out.Days = append(out.Days, acc.day)
	}
	sort.Slice(out.Days, func(i, j int) bool { return out.Days[i].Date < out.Days[j].Date })
	out.Skills = topTallies(skillTally)
	out.Tools = topTallies(toolTally)
	out.Programs = topTallies(progTally)
	if hitMiss > 0 {
		out.Kpi.CacheHitPercent = float64(hit) * 100 / float64(hitMiss)
	}
	sort.SliceStable(out.Sessions, func(i, j int) bool {
		if out.Sessions[i].Tokens != out.Sessions[j].Tokens {
			return out.Sessions[i].Tokens > out.Sessions[j].Tokens
		}
		return out.Sessions[i].LastActive > out.Sessions[j].LastActive
	})
	return out
}

// sortedNameSet 去重集 → 字典序名称清单（会话卡过滤依据，稳定输出）。
func sortedNameSet(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// addCategory 逐分类累加（聚合用，不走指针）。
func addCategory(a, b Category) Category {
	a.System += b.System
	a.Tools += b.Tools
	a.User += b.User
	a.Inject += b.Inject
	a.Skill += b.Skill
	a.Assistant += b.Assistant
	a.Tool += b.Tool
	return a
}

// topTallies 按次数降序（并列按名称稳定排序）取 Top。
func topTallies(m map[string]*InsightsTally) []InsightsTally {
	out := make([]InsightsTally, 0, len(m))
	for _, t := range m {
		out = append(out, *t)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > maxInsightTallies {
		out = out[:maxInsightTallies]
	}
	return out
}

// commandProgram 从命令行提取程序名（首个空白分隔 token 的 basename；
// KEY=v 环境变量前缀与 sudo/nice/nohup 包装跳过）。提取不到返回 "other"。
func commandProgram(cmd string) string {
	for _, tok := range strings.Fields(cmd) {
		if strings.Contains(tok, "=") && !strings.HasPrefix(tok, "/") && !strings.HasPrefix(tok, "-") {
			continue
		}
		if tok == "sudo" || tok == "nice" || tok == "nohup" || tok == "command" {
			continue
		}
		return filepath.Base(strings.Trim(tok, `"'`))
	}
	return "other"
}
