// taskinbox 表驱动测试（规格 进度计划/gaea-task-inbox-7-3-1-20260915.md §1
// 测试清单逐项落地）：状态机全矩阵 / ParseStatus 未知值与大小写 / NormalizeTitle
// 空白与 rune 截断 / ValidSpace·ValidSource 边界 / ParseTaskID 残渣 /
// FilterBySpace 三口径 / Sort 组序+组内时间+同时间 ID 序 / JSON camelCase 标签
// 钉死。先例 skilldistill/distill_test.go：白盒同包、零 IO、纯表驱动。
package taskinbox

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestParseStatus 四合法值归一 + 未知值/大小写敏感/带空白拒绝。
func TestParseStatus(t *testing.T) {
	cases := []struct {
		in   string
		want Status
		ok   bool
	}{
		{"pending", statusPending, true},
		{"doing", statusDoing, true},
		{"done", statusDone, true},
		{"abandoned", statusAbandoned, true},
		// 大小写敏感且不 trim：状态是程序写入的枚举，不是用户输入
		{"Pending", "", false},
		{"PENDING", "", false},
		{" Done", "", false},
		{"done ", "", false},
		{"", "", false},
		{"frozen", "", false},
		{"doing,done", "", false},
	}
	for _, c := range cases {
		got, ok := ParseStatus(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("ParseStatus(%q)=(%q,%v) want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
	// 常量与 wire 值逐字钉死（防常量漂移）
	if string(statusPending) != "pending" || string(statusDoing) != "doing" ||
		string(statusDone) != "done" || string(statusAbandoned) != "abandoned" {
		t.Fatal("状态常量必须与 wire 字符串逐字一致")
	}
}

// TestCanTransition 状态机全矩阵：合法恰 4 条（pending→doing、doing→done、
// pending→abandoned、doing→abandoned），其余含终态回退/复活/跳级/同值/未知全 false。
func TestCanTransition(t *testing.T) {
	// 显式点验：规格 §1「pending→doing→done；pending|doing→abandoned」
	legal := []struct{ from, to Status }{
		{statusPending, statusDoing},
		{statusDoing, statusDone},
		{statusPending, statusAbandoned},
		{statusDoing, statusAbandoned},
	}
	for _, p := range legal {
		if !CanTransition(p.from, p.to) {
			t.Fatalf("合法迁移 %s→%s 应放行", p.from, p.to)
		}
	}
	illegal := []struct{ from, to Status }{
		{statusDone, statusPending},    // 终态回退
		{statusAbandoned, statusDoing}, // 终态复活
		{statusPending, statusDone},    // 跳级：完成必须经 doing
		{statusDone, statusAbandoned},  // 终态互切
		{statusPending, statusPending}, // 同值迁移（无变化）恒 false
		{statusDoing, statusDoing},
		{statusDone, statusDone},
		{statusAbandoned, statusAbandoned},
	}
	for _, p := range illegal {
		if CanTransition(p.from, p.to) {
			t.Fatalf("非法迁移 %s→%s 不得放行", p.from, p.to)
		}
	}
	// 4×4 全矩阵兜底：合法集之外一律 false
	all := []Status{statusPending, statusDoing, statusDone, statusAbandoned}
	legalSet := map[Status]map[Status]bool{
		statusPending:   {statusDoing: true, statusAbandoned: true},
		statusDoing:     {statusDone: true, statusAbandoned: true},
		statusDone:      {},
		statusAbandoned: {},
	}
	for _, from := range all {
		for _, to := range all {
			if got, want := CanTransition(from, to), legalSet[from][to]; got != want {
				t.Fatalf("矩阵 CanTransition(%s→%s)=%v want %v", from, to, got, want)
			}
		}
	}
	// 空/未知残渣：任何方向都不得放行
	targets := append([]Status{""}, all...)
	for _, from := range []Status{"", "Pending", "frozen"} {
		for _, to := range targets {
			if CanTransition(from, to) {
				t.Fatalf("未知 from %q 不得放行到 %q", from, to)
			}
		}
	}
	for _, to := range []Status{"", "unknown"} {
		if CanTransition(statusPending, to) {
			t.Fatalf("未知 to %q 不得可达", to)
		}
	}
}

// TestNormalizeTitle 空/纯空白报错；120 rune 恰好不截；121 rune 截为 120
// （[]rune 口径，中文不得按字节腰斩）；中英混合去首尾空白。
func TestNormalizeTitle(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"空串", "", "", true},
		{"纯空白（空格/换行/tab/全角空格）", " \n\t　", "", true},
		{"中英混合去首尾空白", "  你好 gaea 任务 Inbox\t　", "你好 gaea 任务 Inbox", false},
		{"恰 120 rune 中文不截断", strings.Repeat("任", MaxTitleRunes), strings.Repeat("任", MaxTitleRunes), false},
		{"121 rune 中文截为 120", strings.Repeat("任", MaxTitleRunes+1), strings.Repeat("任", MaxTitleRunes), false},
		{"中英混合超长按 rune 截断", strings.Repeat("ab任", MaxTitleRunes), strings.Repeat("ab任", 40), false},
	}
	for _, c := range cases {
		got, err := NormalizeTitle(c.in)
		if c.wantErr {
			if err == nil || !errors.Is(err, ErrEmptyTitle) {
				t.Fatalf("%s: 期望 ErrEmptyTitle，得到 (%q,%v)", c.name, got, err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Fatalf("%s: 得到 (%q,%v) want (%q,nil)", c.name, got, err, c.want)
		}
	}
	// 截断长度口径复核：任意超长输入必恰 MaxTitleRunes rune
	got, err := NormalizeTitle(strings.Repeat("任", MaxTitleRunes+5))
	if err != nil || len([]rune(got)) != MaxTitleRunes {
		t.Fatalf("超长截断应恰 %d rune，得到 %d（err=%v）", MaxTitleRunes, len([]rune(got)), err)
	}
}

// TestValidSpace 合法恰 work|play；大小写、带空白、其他词一律拒绝。
func TestValidSpace(t *testing.T) {
	for _, ok := range []string{"work", "play"} {
		if !ValidSpace(ok) {
			t.Fatalf("ValidSpace(%q) 应 true", ok)
		}
	}
	for _, bad := range []string{"", "Work", "WORK", "work ", " home", "home", "both"} {
		if ValidSpace(bad) {
			t.Fatalf("ValidSpace(%q) 不应通过", bad)
		}
	}
}

// TestValidSource 合法恰五来源；大小写、近义残渣一律拒绝。
func TestValidSource(t *testing.T) {
	for _, ok := range []string{"ctrlk", "palette", "voice", "weixin", "inbox"} {
		if !ValidSource(ok) {
			t.Fatalf("ValidSource(%q) 应 true", ok)
		}
	}
	for _, bad := range []string{"", "CtrlK", "panel", "command", "ctrlk ", "wechat", "manual"} {
		if ValidSource(bad) {
			t.Fatalf("ValidSource(%q) 不应通过", bad)
		}
	}
}

// TestParseTaskID 残渣防御：合法 "ti-"+12 小写 hex；错前缀/长短偏差/大写/
// 非 hex 字符/尾部空白一律拒绝（先例 ParseSuggestionID 口径）。
func TestParseTaskID(t *testing.T) {
	for _, ok := range []string{
		"ti-0123456789ab",
		"ti-000000000000",
		"ti-ffffffffffff",
		"ti-deadbeefcafe",
	} {
		if !ParseTaskID(ok) {
			t.Fatalf("ParseTaskID(%q) 应 true", ok)
		}
	}
	for _, bad := range []string{
		"",                 // 空串
		"ti-",              // 零 hex
		"ti-0123456789a",   // 11 hex
		"ti-0123456789abc", // 13 hex
		"ti-0123456789AB",  // 大写 hex 拒绝
		"TI-0123456789ab",  // 大写前缀
		"tx-0123456789ab",  // 错前缀
		"it-0123456789ab",  // 错前缀（倒序残渣）
		"ti0123456789ab",   // 缺连字符
		"ti-0123456789ag",  // 非 hex 字符 g
		"ti-0123456789aZ",  // 非 hex 字母
		"ti-0123456789ab ", // 尾部空白
	} {
		if ParseTaskID(bad) {
			t.Fatalf("ParseTaskID(%q) 不应通过", bad)
		}
	}
}

// TestFilterBySpace 三口径：work/play 严格相等保原序；"" 全量（GaeaTaskList
// 变参先例口径）；另验未知空间与 nil 入参的防御行为。
func TestFilterBySpace(t *testing.T) {
	tasks := []Task{
		{ID: "ti-000000000001", Title: "写周报", Space: "work"},
		{ID: "ti-000000000002", Title: "看电影", Space: "play"},
		{ID: "ti-000000000003", Title: "改需求", Space: "work"},
		{ID: "ti-000000000004", Title: "拼乐高", Space: "play"},
	}
	got := FilterBySpace(tasks, "work")
	if len(got) != 2 || got[0].ID != "ti-000000000001" || got[1].ID != "ti-000000000003" {
		t.Fatalf("work 过滤应 [001 003]：%v", idsOf(got))
	}
	got = FilterBySpace(tasks, "play")
	if len(got) != 2 || got[0].ID != "ti-000000000002" || got[1].ID != "ti-000000000004" {
		t.Fatalf("play 过滤应 [002 004]：%v", idsOf(got))
	}
	got = FilterBySpace(tasks, "")
	if len(got) != len(tasks) {
		t.Fatalf("空 space 应全量，得到 %d/%d", len(got), len(tasks))
	}
	for i := range tasks {
		if got[i].ID != tasks[i].ID {
			t.Fatalf("全量应保原序：第 %d 位 %q ≠ %q", i, got[i].ID, tasks[i].ID)
		}
	}
	if got := FilterBySpace(tasks, "home"); len(got) != 0 {
		t.Fatalf("未知空间应空表，得到 %d 条", len(got))
	}
	if got := FilterBySpace(nil, "work"); len(got) != 0 {
		t.Fatalf("nil 入对应空表，得到 %d 条", len(got))
	}
}

// TestSort 跨状态组序（pending<doing<done<abandoned，与时间新旧无关）+ 组内
// UpdatedAt 降序 + 同时间 ID 升序；入参不被改写；未知状态垫底；nil 安全。
func TestSort(t *testing.T) {
	// 输入乱序：ID 均为合法 12hex 形状（与真实数据一致）
	input := []Task{
		{ID: "ti-000000000009", Status: statusAbandoned, UpdatedAt: 999}, // 全场最新但组序垫底
		{ID: "ti-00000000000e", Status: statusDoing, UpdatedAt: 400},
		{ID: "ti-00000000000c", Status: statusPending, UpdatedAt: 200},
		{ID: "ti-00000000000f", Status: statusDone, UpdatedAt: 100}, // 全场最旧仍排 abandoned 前
		{ID: "ti-00000000000b", Status: statusPending, UpdatedAt: 300},
		{ID: "ti-00000000000d", Status: statusDoing, UpdatedAt: 500},
		{ID: "ti-00000000000a", Status: statusPending, UpdatedAt: 300}, // 与 00b 同时间 → ID 升序在前
	}
	want := []string{
		"ti-00000000000a", // pending 300（同时间 ID 小者先）
		"ti-00000000000b", // pending 300
		"ti-00000000000c", // pending 200（组内时间降序）
		"ti-00000000000d", // doing 500
		"ti-00000000000e", // doing 400
		"ti-00000000000f", // done 100
		"ti-000000000009", // abandoned 999（组序垫底，与时间无关）
	}
	got := Sort(input)
	if len(got) != len(want) {
		t.Fatalf("排序后应 %d 条，得到 %d", len(want), len(got))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("排序第 %d 位应为 %s，得到 %s（全序 %v）", i, id, got[i].ID, idsOf(got))
		}
	}
	// 纯函数纪律：入参保持原序不被改写
	if input[0].ID != "ti-000000000009" || input[1].ID != "ti-00000000000e" || input[2].ID != "ti-00000000000c" {
		t.Fatalf("Sort 不得改写入参：%v", idsOf(input))
	}
	// 未知状态残渣垫底（比 abandoned 更后，不干扰合法组序）
	withJunk := append(append([]Task{}, input...), Task{ID: "ti-000000000001", Status: "frozen", UpdatedAt: 1000})
	got2 := Sort(withJunk)
	if got2[len(got2)-1].ID != "ti-000000000001" {
		t.Fatalf("未知状态应垫底：%v", idsOf(got2))
	}
	if got := Sort(nil); len(got) != 0 {
		t.Fatalf("nil 排序应空表，得到 %d 条", len(got))
	}
}

// TestTaskJSONShape JSON 标签逐字钉死（与状态文件契约一致，防漂移）：全字段
// 键名/顺序/camelCase；action·target·session·note 四键 omitempty 缺省不出。
func TestTaskJSONShape(t *testing.T) {
	full := Task{
		ID: "ti-0123456789ab", Title: "写周报", Space: "work", Status: statusDoing,
		Source: "ctrlk", Action: "navigate", Target: "desk-recent-docs",
		Session: "sessions/2026-09-15.md", Note: "周五前交",
		CreatedAt: 1757932800000, UpdatedAt: 1757932801000,
	}
	b, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("marshal 失败：%v", err)
	}
	want := `{"id":"ti-0123456789ab","title":"写周报","space":"work","status":"doing","source":"ctrlk","action":"navigate","target":"desk-recent-docs","session":"sessions/2026-09-15.md","note":"周五前交","createdAt":1757932800000,"updatedAt":1757932801000}`
	if string(b) != want {
		t.Fatalf("全字段 JSON 形状漂移：\n got  %s\n want %s", b, want)
	}
	minimal := Task{
		ID: "ti-000000000000", Title: "拼乐高", Space: "play", Status: statusPending,
		Source: "inbox", CreatedAt: 1, UpdatedAt: 1,
	}
	b2, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("marshal 失败：%v", err)
	}
	want2 := `{"id":"ti-000000000000","title":"拼乐高","space":"play","status":"pending","source":"inbox","createdAt":1,"updatedAt":1}`
	if string(b2) != want2 {
		t.Fatalf("最小卡 JSON 形状漂移（omitempty 四键不得出现）：\n got  %s\n want %s", b2, want2)
	}
}

// idsOf 测试辅助：取 ID 序列，便于报错信息可读。
func idsOf(tasks []Task) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}
