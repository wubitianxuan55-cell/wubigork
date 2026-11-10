package agent

// A2 动态工具目录（dsh-ops 蒸馏 2026-11-10，上游 discovery/controller.js）。
//
// 工具面两级：resident（常驻 schema）+ discoverable（注册在册但不出 schema）。
// discover_tools 对未常驻工具的元数据做**确定性词法匹配**（不调模型），命中
// 即加入会话激活集——**下一模型步**生效（stream 每步重读 currentSchemas）。
//
// 上游纪律原样继承：
//   - 可见性≠授权：激活只把 schema 挂进工具面，权限/审批/写作用域闸照常；
//   - 每次激活 1–5 个、query ≤512 字符、同分按名字固定序；
//   - in-history 模式：会话历史里出现过的工具调用视为已激活（模型已见过其
//     schema）——重启/续会话后不必重新发现，也不因 schema 面收缩让历史调用
//     变成「未激活」。
//
// 如实缩小（记档 distill §4 A2）：目录文本走 discover 回执而非常驻 prompt 段
// （不为目录付常驻 token）；不做「只读作用域不得激活写工具」（gaea 权限闸
// 在执行期拦，激活不扩权）；激活集进程内存驻留（重启即失，对齐
// controller.granted 语义，in-history 播种兜底）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// DiscoverToolName 是发现入口工具名（目录开启时恒常驻，否则模型无从发现）。
const DiscoverToolName = "discover_tools"

const (
	discoverMaxActivate = 5   // 每次激活上限（上游 1–5）
	discoverMaxQuery    = 512 // query 长度帽（上游同值）
	discoverCatalogCap  = 20  // 目录清单条数上限（token 纪律）
)

// toolActive 报告工具是否在当前会话工具面（目录关闭恒 true）。turnMu 保护
// （resident 可经 SetResidentTools 于装配期改写，activated 并发写）。
func (a *AgentRunner) toolActive(name string) bool {
	a.turnMu.Lock()
	defer a.turnMu.Unlock()
	return a.resident == nil || a.resident[name] || a.activated[name]
}

// toolResident 报告工具是否在常驻名单（目录关闭恒 false）。
func (a *AgentRunner) toolResident(name string) bool {
	a.turnMu.Lock()
	defer a.turnMu.Unlock()
	return a.resident != nil && a.resident[name]
}

// activateTools 把工具加入会话激活集，返回本次**新**激活的名字（已常驻/已
// 激活的不重复报）。目录关闭时 no-op。
func (a *AgentRunner) activateTools(names []string) []string {
	if !a.DynamicToolsOn() {
		return nil
	}
	var newly []string
	a.turnMu.Lock()
	for _, n := range names {
		if a.resident[n] || a.activated[n] {
			continue
		}
		a.activated[n] = true
		newly = append(newly, n)
	}
	a.turnMu.Unlock()
	return newly
}

// seedActivatedFromHistory in-history 播种：会话历史里 assistant 发起过的
// 工具调用视为已激活（每轮开头调用，幂等；激活集只增不减）。
func (a *AgentRunner) seedActivatedFromHistory() {
	if !a.DynamicToolsOn() || a.session == nil || a.tools == nil {
		return
	}
	var names []string
	for _, m := range a.session.Messages {
		if m.Role != provider.RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if _, ok := a.tools.Get(tc.Name); ok {
				names = append(names, tc.Name)
			}
		}
	}
	a.activateTools(names)
}

// discoverCandidates 返回可发现工具（在册、非隐藏、未常驻）的 schema 元数据，
// 按名字序（Schemas 已排序）。
func (a *AgentRunner) discoverCandidates() []provider.ToolSchema {
	all := a.tools.Schemas()
	a.turnMu.Lock()
	resident := a.resident
	a.turnMu.Unlock()
	if resident == nil {
		return nil
	}
	out := make([]provider.ToolSchema, 0, len(all))
	for _, s := range all {
		if resident[s.Name] {
			continue
		}
		out = append(out, s)
	}
	return out
}

// discoverTerms 把 query 切成匹配词：小写；字母数字连续段为一词；汉字连续段
// 为一词，超过 2 字再补全部二元组（容忍措辞差异）。单个 ASCII 字符丢弃（噪声）。
func discoverTerms(q string) []string {
	q = strings.ToLower(strings.TrimSpace(q))
	var (
		terms  []string
		cur    []rune
		curHan bool
		seen   = map[string]bool{}
	)
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		terms = append(terms, s)
	}
	flush := func() {
		if len(cur) == 0 {
			return
		}
		if curHan || len(cur) > 1 {
			add(string(cur))
		}
		if curHan && len(cur) > 2 {
			for i := 0; i+2 <= len(cur); i++ {
				add(string(cur[i : i+2]))
			}
		}
		cur = cur[:0]
	}
	for _, r := range q {
		isHan := unicode.Is(unicode.Han, r)
		if !isHan && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush() // 分隔符（含下划线：chart_gen → chart + gen 两词）
			continue
		}
		if len(cur) > 0 && isHan != curHan {
			flush()
		}
		curHan = isHan
		cur = append(cur, r)
	}
	flush()
	return terms
}

// discoverScore 名字命中 +3、描述命中 +1（逐词累加）。确定性，无模型调用。
func discoverScore(terms []string, name, desc string) int {
	ln, ld := strings.ToLower(name), strings.ToLower(desc)
	score := 0
	for _, t := range terms {
		if strings.Contains(ln, t) {
			score += 3
		}
		if strings.Contains(ld, t) {
			score++
		}
	}
	return score
}

// discoverToolsTool 是发现入口（agent 包工具：经 ctx 取宿主 runner，无装配
// 顺序依赖；子代理/测试 runner 目录关闭时如实答复「全量常驻」）。
type discoverToolsTool struct{}

// NewDiscoverToolsTool 构造发现入口工具（boot 装配单点 reg.Add）。
func NewDiscoverToolsTool() tool.Tool { return discoverToolsTool{} }

func (discoverToolsTool) Name() string { return DiscoverToolName }

func (discoverToolsTool) Description() string {
	return "发现并激活未常驻的工具(工具面=常驻+已激活;query关键词词法匹配或select精确名,每次≤5,下一模型步生效;空参数列目录;可见性≠授权)"
}

func (discoverToolsTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"关键词(工具名/用途)"},"select":{"type":"array","items":{"type":"string"},"description":"精确工具名"}}}`)
}

// ReadOnly：不产生宿主副作用（只改本会话 schema 面），计划模式可用。
func (discoverToolsTool) ReadOnly() bool { return true }

func (discoverToolsTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query  string   `json:"query"`
		Select []string `json:"select"`
	}
	if len(strings.TrimSpace(string(args))) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", &tool.Failure{Code: tool.CodeValidationError,
				Err: fmt.Errorf("invalid args: %w", err), EffectState: tool.EffectNone}
		}
	}
	a, ok := RunnerFromContext(ctx)
	if !ok {
		return "", &tool.Failure{Code: tool.CodeValidationError,
			Err: fmt.Errorf("discover_tools 只能在 agent 回合内调用"), EffectState: tool.EffectNone}
	}
	if !a.DynamicToolsOn() {
		return "当前会话工具全量常驻，无需发现（动态工具目录未开启）。", nil
	}
	if len(p.Query) > discoverMaxQuery {
		return "", &tool.Failure{Code: tool.CodeValidationError,
			Err:         fmt.Errorf("query 过长（%d > %d 字符）", len(p.Query), discoverMaxQuery),
			EffectState: tool.EffectNone}
	}
	cands := a.discoverCandidates()
	byName := make(map[string]provider.ToolSchema, len(cands))
	for _, c := range cands {
		byName[c.Name] = c
	}

	// 空参数=列目录（不激活）。
	if strings.TrimSpace(p.Query) == "" && len(p.Select) == 0 {
		return discoverCatalog(a, cands, "可发现工具目录"), nil
	}

	var (
		picked  []string
		unknown []string
		seen    = map[string]bool{}
	)
	pick := func(n string) {
		if !seen[n] && len(picked) < discoverMaxActivate {
			seen[n] = true
			picked = append(picked, n)
		}
	}
	for _, n := range p.Select {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := byName[n]; ok {
			pick(n)
		} else if a.toolResident(n) {
			seen[n] = true // 已常驻：无需激活，不占激活额度
		} else {
			unknown = append(unknown, n)
		}
	}
	if terms := discoverTerms(p.Query); len(terms) > 0 {
		type hit struct {
			name  string
			score int
		}
		var hits []hit
		for _, c := range cands {
			if s := discoverScore(terms, c.Name, c.Description); s > 0 {
				hits = append(hits, hit{c.Name, s})
			}
		}
		// 同分按名字固定序（确定性）。
		sort.SliceStable(hits, func(i, j int) bool {
			if hits[i].score != hits[j].score {
				return hits[i].score > hits[j].score
			}
			return hits[i].name < hits[j].name
		})
		for _, h := range hits {
			pick(h.name)
		}
	}

	if len(picked) == 0 {
		head := "无匹配工具"
		if len(unknown) > 0 {
			head += "（未知或不可发现：" + strings.Join(unknown, ", ") + "）"
		}
		return discoverCatalog(a, cands, head+"。可发现工具目录"), nil
	}
	newly := a.activateTools(picked)
	newSet := make(map[string]bool, len(newly))
	for _, n := range newly {
		newSet[n] = true
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[discover] 已激活 %d 个工具（下一模型步生效；本次调用零工具动作）：\n", len(newly))
	for _, n := range picked {
		mark := ""
		if !newSet[n] {
			mark = "（此前已激活）"
		}
		fmt.Fprintf(&b, "  - %s%s — %s\n", n, mark, byName[n].Description)
	}
	if len(unknown) > 0 {
		fmt.Fprintf(&b, "未知或不可发现：%s\n", strings.Join(unknown, ", "))
	}
	b.WriteString("可见性≠授权：激活只把 schema 挂进工具面，权限/审批闸照常。")
	return b.String(), nil
}

// discoverCatalog 目录清单（≤discoverCatalogCap 条；已激活的标注）。
func discoverCatalog(a *AgentRunner, cands []provider.ToolSchema, head string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s（共 %d 个，显示前 %d）：\n", head, len(cands), min(len(cands), discoverCatalogCap))
	for i, c := range cands {
		if i >= discoverCatalogCap {
			fmt.Fprintf(&b, "  … +%d 个未列出（用更具体的 query 缩小范围）\n", len(cands)-discoverCatalogCap)
			break
		}
		mark := ""
		if a.toolActive(c.Name) {
			mark = " [已激活]"
		}
		fmt.Fprintf(&b, "  - %s%s — %s\n", c.Name, mark, c.Description)
	}
	b.WriteString("用 query 关键词或 select 精确名激活（每次≤5，下一模型步生效）。")
	return b.String()
}
