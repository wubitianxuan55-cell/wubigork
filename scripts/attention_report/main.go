// attention_report 是「注意力事务」阶段 0 的只读基线统计器（2026-11-02，
// 众神殿共识产物）。它不改变任何运行时行为：只读既有 <id>.gaea-log.jsonl
// 事件日志，数出三件事——
//
//  1. 每轮打断次数：approval_request / ask_request 各几张卡（口径=一次打断一张卡，
//     ask 按卡不按问题数）；
//  2. 解决率：同一 id 是否等到 approval_resolved / ask_resolved（新日志才有，
//     旧日志此处恒为「未解决」，这正是阶段 1 要补的洞）；
//  3. 结果分布与等待时长：allowed / denied / aborted / timeout / cancelled /
//     answered，以及 latency_ms 均值、来源 source（旧日志无此字段，计 legacy）。
//
// 用法：
//
//	go run ./scripts/attention_report                    # 默认扫 .gaea/sessions 及 work/play 分区
//	go run ./scripts/attention_report <dir-or-file> ...  # 指定目录或日志文件
//	go run ./scripts/attention_report -json              # 机器可读
//
// 退出码恒 0（统计不是断言）；读不了的单个文件在 stderr 提示并跳过。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 日志 kind 字面量。请求侧复用 session 包的持久化字符串（与写端同源），
// 结果侧是本轮新增的成对事件。
const (
	kindApprovalRequest = "approval_request"
	kindAskRequest      = "ask_request"
	kindApprovalResolve = "approval_resolved"
	kindAskResolve      = "ask_resolved"
	kindTurnStarted     = "turn_started"
	kindTurnDone        = "turn_done"
)

// approvalPayload / askPayload 是请求侧的最小字段面（旧日志缺 source/reason）。
type approvalPayload struct {
	ID      string `json:"id"`
	Tool    string `json:"tool"`
	Subject string `json:"subject"`
	Source  string `json:"source"`
	Reason  string `json:"reason"`
	Request bool   `json:"request"`
}

// approvalResolvedPayload 是审批结果侧。
type approvalResolvedPayload struct {
	ID        string `json:"id"`
	Tool      string `json:"tool"`
	Subject   string `json:"subject"`
	Source    string `json:"source"`
	Decision  string `json:"decision"`
	Outcome   string `json:"outcome"`
	LatencyMS int64  `json:"latency_ms"`
}

// askPayload / askResolvedPayload 是提问侧。Ask 按卡计数，questions 只用于
// 「卡内问题数」诊断列。
type askPayload struct {
	ID        string `json:"id"`
	Questions []struct {
		ID     string `json:"id"`
		Prompt string `json:"prompt"`
	} `json:"questions"`
}

type askResolvedPayload struct {
	ID        string `json:"id"`
	Outcome   string `json:"outcome"`
	Answered  int    `json:"answered"`
	LatencyMS int64  `json:"latency_ms"`
}

// sessionStat 是一份日志的统计结果。
type sessionStat struct {
	Path          string          `json:"path"`
	Session       string          `json:"session"`
	Turns         int             `json:"turns"`
	Approvals     int             `json:"approvals"`
	Asks          int             `json:"asks"`
	Cards         int             `json:"cards"`
	Unresolved    int             `json:"unresolved"`
	UnresolvedIDs []string        `json:"unresolvedIds,omitempty"`
	Outcomes      map[string]int  `json:"outcomes"`
	Sources       map[string]int  `json:"sources"`
	LatencyN      int             `json:"latencySamples"`
	LatencyMS     int64           `json:"latencyMsTotal"`
	Questions     int             `json:"questions"`
	OpenIDs       []string        `json:"-"`
	openSet       map[string]bool `json:"-"`
	legacyIDs     map[string]bool `json:"-"`
}

// main 解析参数并汇总所有输入。
func main() {
	jsonOut := flag.Bool("json", false, "以 JSON 输出（机器可读）")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "attention_report：注意力事务基线统计（只读）\n\n用法: go run ./scripts/attention_report [-json] [目录或日志文件 ...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	paths, err := collectLogs(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "attention_report: %v\n", err)
		os.Exit(0) // 统计器不因缺目录而失败
	}

	var stats []sessionStat
	for _, p := range paths {
		st, err := scanLog(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "attention_report: 跳过 %s: %v\n", p, err)
			continue
		}
		stats = append(stats, st)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(stats)
		return
	}
	printMarkdown(stats)
}

// collectLogs 把参数展开成日志文件列表：目录取其下的 *.gaea-log.jsonl，
// 文件直接采用；无参数时扫默认会话目录（含 work/play 分区）。
func collectLogs(args []string) ([]string, error) {
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		base := filepath.Join(cwd, ".gaea", "sessions")
		args = []string{base, filepath.Join(base, "work"), filepath.Join(base, "play")}
	}
	seen := map[string]bool{}
	var out []string
	for _, a := range args {
		fi, err := os.Stat(a)
		if err != nil {
			continue // 不存在的分区不是错误
		}
		if !fi.IsDir() {
			if strings.HasSuffix(a, ".gaea-log.jsonl") && !seen[a] {
				seen[a] = true
				out = append(out, a)
			}
			continue
		}
		entries, err := os.ReadDir(a)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".gaea-log.jsonl") {
				continue
			}
			p := filepath.Join(a, e.Name())
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// scanLog 单文件统计：一遍流式读取，用 id 把请求与结果配对。
func scanLog(path string) (sessionStat, error) {
	entries, err := session.ReadLogRepaired(path)
	if err != nil {
		return sessionStat{}, err
	}
	st := sessionStat{
		Path:      path,
		Session:   strings.TrimSuffix(filepath.Base(path), ".gaea-log.jsonl"),
		Outcomes:  map[string]int{},
		Sources:   map[string]int{},
		openSet:   map[string]bool{},
		legacyIDs: map[string]bool{},
	}

	// legacy：旧日志的 approval_request 没有 source 字段，统一记 legacy，
	// 避免把「未记录」误读成「gate 触发」。
	for _, e := range entries {
		switch e.Kind {
		case kindTurnStarted:
			st.Turns++
		case kindApprovalRequest:
			var p approvalPayload
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			st.Approvals++
			st.Cards++
			st.markOpen(p.ID)
			src := p.Source
			if src == "" {
				// legacy：旧日志的 approval_request 没有 source 字段，先记 legacy，
				// 若同 id 有结果侧事件再按其 source 精确归位（见下方 resolve 分支）。
				src = "legacy"
				st.legacyIDs[p.ID] = true
			}
			st.Sources[src]++
		case kindAskRequest:
			var p askPayload
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			st.Asks++
			st.Cards++
			st.Questions += len(p.Questions)
			st.markOpen(p.ID)
		case kindApprovalResolve:
			var p approvalResolvedPayload
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			st.resolve(p.ID, p.Outcome, p.LatencyMS)
			// 结果侧携带同一张卡的 source：旧请求行缺字段时据此精确归位；
			// 新日志两侧都有值，请求侧已计过，不重复计数。
			if p.Source != "" && st.legacyIDs[p.ID] {
				delete(st.legacyIDs, p.ID)
				st.Sources["legacy"]--
				st.Sources[p.Source]++
			}
		case kindAskResolve:
			var p askResolvedPayload
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			st.resolve(p.ID, p.Outcome, p.LatencyMS)
		}
	}

	for id := range st.openSet {
		st.UnresolvedIDs = append(st.UnresolvedIDs, id)
	}
	sort.Strings(st.UnresolvedIDs)
	st.Unresolved = len(st.UnresolvedIDs)
	// legacy 归位后可能归零：删掉零值键，报告里不留 legacy: 0 噪声。
	for k, v := range st.Sources {
		if v == 0 {
			delete(st.Sources, k)
		}
	}
	if st.LatencyN > 0 {
		st.LatencyMS = st.LatencyMS / int64(st.LatencyN)
	}
	return st, nil
}

// markOpen 登记一张尚未解决的卡。
func (s *sessionStat) markOpen(id string) {
	if s.openSet[id] {
		return // 同一 id 重复请求（日志修复/重放）只算一张卡
	}
	s.openSet[id] = true
	s.OpenIDs = append(s.OpenIDs, id)
}

// resolve 按 id 关闭一张卡：同一 id 只计一次（重复结果事件忽略）。
func (s *sessionStat) resolve(id, outcome string, latencyMS int64) {
	if !s.openSet[id] {
		return
	}
	delete(s.openSet, id)
	if outcome == "" {
		outcome = "unknown"
	}
	s.Outcomes[outcome]++
	s.LatencyMS += latencyMS
	s.LatencyN++
}

// printMarkdown 渲染人读报告：先逐会话一行，再给分项合计与结论提示。
func printMarkdown(stats []sessionStat) {
	if len(stats) == 0 {
		fmt.Println("# 注意力事务基线（阶段 0）")
		fmt.Println()
		fmt.Println("未找到 *.gaea-log.jsonl。请确认会话目录：`.gaea/sessions` 或指定路径。")
		return
	}

	total := sessionStat{Outcomes: map[string]int{}, Sources: map[string]int{}}
	var latencyTotal int64
	var latencyN int
	fmt.Println("# 注意力事务基线（阶段 0）")
	fmt.Println()
	fmt.Println("口径：一次打断=一张卡（ask 按卡不按问题数）；「已解决」= 同 id 等到 *_resolved 事件。")
	fmt.Println()
	fmt.Println("| 会话 | 轮次 | 打断卡 | 审批 | 提问 | 已解决 | 未解决 | 均等待 |")
	fmt.Println("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
	for _, s := range stats {
		resolved := s.Cards - s.Unresolved
		fmt.Printf("| %s | %d | %d | %d | %d | %d | %d | %s |\n",
			shortSession(s.Session), s.Turns, s.Cards, s.Approvals, s.Asks,
			resolved, s.Unresolved, formatLatency(s.LatencyN, s.LatencyMS))
		total.Turns += s.Turns
		total.Cards += s.Cards
		total.Approvals += s.Approvals
		total.Asks += s.Asks
		total.Unresolved += s.Unresolved
		total.Questions += s.Questions
		latencyTotal += s.LatencyMS * int64(s.LatencyN)
		latencyN += s.LatencyN
		for k, v := range s.Outcomes {
			total.Outcomes[k] += v
		}
		for k, v := range s.Sources {
			total.Sources[k] += v
		}
	}
	meanMS := int64(0)
	if latencyN > 0 {
		meanMS = latencyTotal / int64(latencyN)
	}
	fmt.Printf("| **合计** | %d | %d | %d | %d | %d | %d | %s |\n",
		total.Turns, total.Cards, total.Approvals, total.Asks,
		total.Cards-total.Unresolved, total.Unresolved, formatLatency(latencyN, meanMS))
	fmt.Println()

	fmt.Printf("提问卡内问题数合计：%d（仅诊断，不计入打断次数）\n", total.Questions)
	fmt.Println()

	fmt.Println("## 结果分布")
	if len(total.Outcomes) == 0 {
		fmt.Println("无已解决记录——旧日志只有请求没有结果侧事件，这是阶段 1 要补的洞。")
	} else {
		for _, k := range sortedKeys(total.Outcomes) {
			fmt.Printf("- %s: %d\n", k, total.Outcomes[k])
		}
	}
	fmt.Println()

	fmt.Println("## 触发来源")
	if len(total.Sources) == 0 {
		fmt.Println("无来源字段记录。")
	} else {
		for _, k := range sortedKeys(total.Sources) {
			fmt.Printf("- %s: %d\n", k, total.Sources[k])
		}
		if total.Sources["legacy"] > 0 {
			fmt.Println()
			fmt.Println("注：`legacy` = 旧日志没有 `source` 字段（改动前的写入），不等于 gate 触发；")
			fmt.Println("    新日志起每一张卡都带 source，可与 gate/hard_ask/project_plan/permission_request 分项统计。")
		}
	}

	var orphans []string
	for _, s := range stats {
		for _, id := range s.UnresolvedIDs {
			orphans = append(orphans, s.Session+"/"+id)
		}
	}
	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println("## 未解决的卡（可重放置信面）")
		for _, o := range orphans {
			fmt.Printf("- %s\n", o)
		}
	}
}

// shortSession 缩短会话文件名以便表格宽度可控（保留尾部唯一区分段）。
func shortSession(name string) string {
	r := []rune(name)
	if len(r) <= 22 {
		return name
	}
	return "…" + string(r[len(r)-21:])
}

// formatLatency 渲染均值等待；无样本时返回 "-"。
func formatLatency(n int, meanMS int64) string {
	if n == 0 {
		return "-"
	}
	if meanMS < 1000 {
		return fmt.Sprintf("%dms", meanMS)
	}
	return fmt.Sprintf("%.1fs", float64(meanMS)/1000)
}

// sortedKeys 稳定输出分项，避免 map 随机序。
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
