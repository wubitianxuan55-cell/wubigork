package sessionquery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// handoff_test.go — 2026-11-27 AI 反馈 #14 回归钉：会话交接卡按需取（session_search
// handoff=<会话 id>）——首条任务、最后一条助手成稿、经工具写过的文件及其现状。
// 实时日志（独立 tool_dispatch）与旧格式（内嵌 tool_calls）都认；只读工具不算；
// bash 只认带重定向的；体检经可注入的钩子。

func dispatchLine(seq, ts int64, id, name, args string, readOnly, partial bool) logLine {
	return logLine{Seq: seq, Ts: ts, Kind: "tool_dispatch", Payload: mustJSON(map[string]any{
		"id": id, "name": name, "args": args, "readOnly": readOnly, "partial": partial,
	})}
}

func resultLine(seq, ts int64, id, errText string) logLine {
	return logLine{Seq: seq, Ts: ts, Kind: "tool_result", Payload: mustJSON(map[string]string{"id": id, "err": errText})}
}

func handoffWorkspace(t *testing.T) (ws, sessions string) {
	t.Helper()
	ws = t.TempDir()
	sessions = filepath.Join(ws, ".gaea", "sessions", "work")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"scripts", "exports", "交付"} {
		if err := os.MkdirAll(filepath.Join(ws, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return ws, sessions
}

func TestHandoffCardFromLiveLog(t *testing.T) {
	ws, sessions := handoffWorkspace(t)
	for _, p := range []string{"scripts/fill.py", "exports/明细.csv", "交付/测算.xlsx"} {
		if err := os.WriteFile(filepath.Join(ws, filepath.FromSlash(p)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 体检钩子：只对 xlsx 给结论（真实实现见 boot 的 wireArtifactHealth）。
	SetArtifactHealth(func(path string) string {
		if strings.HasSuffix(path, ".xlsx") {
			return "⚠ 公式 3 格，其中 3 格没有缓存值"
		}
		return ""
	})
	t.Cleanup(func() { SetArtifactHealth(nil) })

	const id = "20261127-210000.000000000-gaea"
	writeSessionLog(t, sessions, id, []logLine{
		userLine(1, 2_000, "在模板文件上编制包2测算"),
		dispatchLine(2, 2_001, "w1", "write_file", `{"path":"scripts/fill.py","content":"print(1)"}`, false, false),
		resultLine(3, 2_002, "w1", ""),
		dispatchLine(4, 2_003, "r1", "read_file", `{"path":"参考.md"}`, true, false),
		resultLine(5, 2_004, "r1", ""),
		dispatchLine(6, 2_005, "b1", "bash", `{"command":"python scripts/fill.py > exports/明细.csv 2>&1"}`, false, false),
		resultLine(7, 2_006, "b1", ""),
		dispatchLine(8, 2_007, "e1", "edit_file", `{"path":"exports/表.xlsx","old":"a","new":"b"}`, false, false),
		resultLine(9, 2_008, "e1", "stale file"),
		dispatchLine(10, 2_009, "m1", "move_file", `{"source":".gaea/work/a.xlsx","destination":"交付/测算.xlsx","overwrite":true}`, false, false),
		resultLine(11, 2_010, "m1", ""),
		dispatchLine(12, 2_011, "p1", "write_file", `{"path":"half.txt"}`, false, true), // 半截派发不算
		liveAssistantLine(13, 2_100, "已生成 模板版.xlsx，明细溢出问题待核。"),
	})
	s := NewStore(dbForTest(t), sessions)

	card, err := s.Handoff(id)
	if err != nil {
		t.Fatal(err)
	}
	if card.FirstPrompt != "在模板文件上编制包2测算" || !strings.Contains(card.LastReply, "模板版.xlsx") {
		t.Errorf("任务与收尾回复不对：%+v", card)
	}
	var paths []string
	for _, f := range card.Files {
		paths = append(paths, f.Path)
	}
	// 只读工具的文件不算；move_file 只算终点；重定向目标算。
	if got := strings.Join(paths, "|"); got != "scripts/fill.py|exports/明细.csv|exports/表.xlsx|交付/测算.xlsx" {
		t.Fatalf("动过的文件不对：%s", got)
	}
	if card.Files[0].OK != true || card.Files[2].OK != false || card.Files[2].Calls != 1 {
		t.Errorf("回执成败应带到文件上：%+v", card.Files)
	}
	health := map[string]string{}
	for _, f := range card.Files {
		health[f.Path] = f.Health
	}
	if !strings.HasPrefix(health["scripts/fill.py"], "在（") || !strings.HasPrefix(health["交付/测算.xlsx"], "⚠ 公式 3 格") ||
		!strings.Contains(health["exports/表.xlsx"], "已不存在") {
		t.Errorf("文件现状不对：%v", health)
	}

	out := FormatHandoff(card)
	for _, want := range []string{"## 会话交接卡", "2026-11-27 21:00", "**任务**：在模板文件上编制包2测算", "**收尾回复**",
		"已生成 模板版.xlsx", "动过的文件**（4 个", "- scripts/fill.py（write_file ×1，有成功回执）：在（1 B）",
		"- exports/表.xlsx（edit_file ×1，未见成功回执）：已不存在", "交接提示"} {
		if !strings.Contains(out, want) {
			t.Errorf("交接卡缺 %q：\n%s", want, out)
		}
	}

	// 清单里带收尾一行；工具层 handoff 取卡；没有的会话如实报。
	list, err := s.Recent(0)
	if err != nil || len(list) != 1 || !strings.Contains(list[0].LastReply, "模板版.xlsx") {
		t.Fatalf("最近会话应带收尾回复：%+v %v", list, err)
	}
	if recent := FormatRecent(list); !strings.Contains(recent, "收尾：已生成 模板版.xlsx") {
		t.Errorf("清单应有一行收尾：\n%s", recent)
	}
	cardOut, err := (searchTool{store: s}).Execute(context.Background(), mustJSON(map[string]any{"handoff": id}))
	if err != nil || !strings.Contains(cardOut, "## 会话交接卡") {
		t.Fatalf("handoff 应返回交接卡：%v\n%s", err, cardOut)
	}
	if _, err = (searchTool{store: s}).Execute(context.Background(), mustJSON(map[string]any{"handoff": "20260101-000000.000000000-gaea"})); err == nil || !strings.Contains(err.Error(), "没有") {
		t.Fatalf("不存在的会话应如实报：%v", err)
	}
}

func TestHandoffCardFromLegacyLog(t *testing.T) {
	ws, sessions := handoffWorkspace(t)
	const id = "20261120-090000.000000000-gaea"
	writeSessionLog(t, sessions, id, []logLine{
		userLine(1, 1_000, "旧格式的会话"),
		{Seq: 2, Ts: 1_100, Kind: "assistant_message", Payload: mustJSON(map[string]any{
			"text": "写好了",
			"tool_calls": []map[string]any{
				{"id": "c1", "name": "write_file", "args": `{"path":"legacy.md","content":"x"}`},
				{"id": "c2", "name": "read_file", "args": `{"path":"only-read.md"}`},
			},
		})},
		resultLine(3, 1_101, "c1", ""),
		resultLine(4, 1_102, "c2", ""),
	})
	s := NewStore(dbForTest(t), sessions)
	card, err := s.Handoff(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Files) != 1 || card.Files[0].Path != "legacy.md" || !card.Files[0].OK || !card.LegacyCalls {
		t.Fatalf("旧格式内嵌调用应抽出写过的文件、排除只读工具：%+v", card.Files)
	}
	if card.LastReply != "写好了" {
		t.Errorf("收尾回复：%q", card.LastReply)
	}
	if out := FormatHandoff(card); !strings.Contains(out, "旧格式日志里读写只能按工具名区分") {
		t.Errorf("旧格式应注明读写按名字区分：\n%s", out)
	}
	_ = ws
}

func TestBashRedirectPaths(t *testing.T) {
	got := bashRedirectPaths(`echo a > out.csv && python x.py >> "logs/run.txt" 2>&1; cat > notes.md <<EOF; y>/dev/null; curl -o page.html http://x`)
	if strings.Join(got, "|") != "out.csv|logs/run.txt|notes.md" {
		t.Fatalf("重定向目标不对：%v", got)
	}
	if bashRedirectPaths("python script.py") != nil {
		t.Fatal("没有重定向就不该有文件")
	}
}
