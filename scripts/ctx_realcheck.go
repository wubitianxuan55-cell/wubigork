//go:build ignore

// ctx_realcheck.go — 上下文折叠真实数据探针（v4.542 完善批）：对本工作区
// .gaea/sessions 的全部会话日志跑 FoldTimeline + BuildInsights，打印逐会话
// 摘要与跨会话洞察，验证折叠管线在真实日志上的行为（HTTP 桥对变参绑定有
// 反射限制，桌面端不受影响；本探针补「真实数据端到端」验证面）。
//
// 运行：go run ./scripts/ctx_realcheck.go [-dir <sessions 目录>] [-space work|play|all]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

func main() {
	dir := flag.String("dir", ".gaea/sessions", "会话目录（默认本工作区）")
	flag.Parse()
	infos, err := session.List(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list:", err)
		os.Exit(1)
	}
	if len(infos) == 0 {
		fmt.Println("无会话")
		return
	}
	folds := make([]contextview.InsightsSessionFold, 0, len(infos))
	for _, s := range infos {
		entries, err := session.ReadEntriesFor(s.Path)
		if err != nil {
			fmt.Printf("跳过 %s: %v\n", filepath.Base(s.Path), err)
			continue
		}
		tl := contextview.FoldTimeline(entries, 0, 0)
		// 家族事件流（刀10）：subagents 目录枚举+折叠
		subs := map[string][]session.LogEntry{}
		if matches, _ := filepath.Glob(filepath.Join(filepath.Dir(s.Path), "subagents", "*.jsonl")); len(matches) > 0 {
			for _, m := range matches {
				if fe, err := session.ReadLogRepaired(m); err == nil {
					subs[strings.TrimSuffix(filepath.Base(m), ".jsonl")] = fe
				}
			}
		}
		fam := trajectory.FoldFamilyEvents(entries, subs)
		famKinds := map[string]int{}
		for _, fe := range fam {
			famKinds[fe.Kind]++
		}
		folds = append(folds, contextview.InsightsSessionFold{
			Path: s.Path, Preview: s.Preview, SpaceID: s.Space,
			ModTime: s.ModTime.Unix(), Timeline: tl,
		})
		b, _ := json.Marshal(tl.Stats)
		cats := map[string]int{}
		for _, n := range tl.Nodes {
			cats[n.Cat]++
		}
		catsJSON, _ := json.Marshal(cats)
		fmt.Printf("== %s（%d 条日志）\n   stats=%s\n   节点=%s 命令=%d 请求=%d 事件=%d\n   家族流=%v transcripts=%d\n",
			filepath.Base(s.Path)+" · "+s.Preview, len(entries), b, catsJSON, len(tl.Commands), len(tl.Requests), len(tl.Events), famKinds, len(subs))
		if len(tl.Requests) > 0 {
			q := tl.Requests[len(tl.Requests)-1]
			cj, _ := json.Marshal(q.Category)
			fmt.Printf("   末请求: T%d·S%d prompt=%d cost=%f cat=%s delta=%d\n",
				q.Turn, q.Step, q.PromptTokens, q.Cost, cj, deltaOf(q.Delta))
		}
	}
	out := contextview.BuildInsights(contextview.InsightsAll, time.Now(), folds)
	b, _ := json.MarshalIndent(out.Kpi, "", "  ")
	fmt.Printf("\n── 洞察 KPI（all）──\n%s\n", b)
	fmt.Printf("days=%d tools=%s skills=%s programs=%s sessions=%d\n",
		len(out.Days), names(out.Tools), names(out.Skills), names(out.Programs), len(out.Sessions))
}

func deltaOf(d *contextview.RequestDelta) int64 {
	if d == nil {
		return 0
	}
	return d.Tokens
}

func names(ts []contextview.InsightsTally) string {
	parts := make([]string, 0, len(ts))
	for i, t := range ts {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s×%d", t.Name, t.Count))
	}
	return strings.Join(parts, ",")
}

