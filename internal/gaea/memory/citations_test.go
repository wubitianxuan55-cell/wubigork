package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestExtractCitationNames(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"", nil},
		{"没有引用的回答", nil},
		{"按 [MEM:cost-rule] 汇总，再按 [MEM:cost-rule] 校验", []string{"cost-rule"}},
		{"[mem:a] 与 [MEM:b] 大小写都收", []string{"a", "b"}},
		{"[MEM:a][MEM:b][MEM:a] 去重保序", []string{"a", "b"}},
		{"普通 [链接](x) 与 [MEM:x-y]（连字符键）", []string{"x-y"}},
		{"[MEM:x-y_z] 下划线不合 kebab-case 不匹配", nil},
		{"[MEM:] 空键与 [MEM:-] 非法键不匹配", nil},
		{"[MEM:Cost-Rule] 大写键归一", []string{"cost-rule"}},
	}
	for _, tc := range cases {
		got := ExtractCitationNames(tc.text)
		if len(got) != len(tc.want) {
			t.Fatalf("ExtractCitationNames(%q) = %v, want %v", tc.text, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("ExtractCitationNames(%q) = %v, want %v", tc.text, got, tc.want)
			}
		}
	}
}

func newCitationTestSet(t *testing.T) *Set {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	set := &Set{Store: s}
	if _, err := s.Save(Memory{
		Name: "cost-rule", Title: "成本测算规则",
		Description: "先对科目再汇总", Type: TypeProject, Kind: KindProcedural,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{
		Name: "stale-fact", Title: "陈旧事实",
		Description: "很久没有修订", Type: TypeProject, Kind: KindSemantic,
		UpdatedAt: time.Now().Add(-120 * 24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	return set
}

func TestResolveCitationsTouch(t *testing.T) {
	set := newCitationTestSet(t)

	if _, ok := set.Store.Get("cost-rule"); !ok {
		t.Fatal("cost-rule 未保存")
	}
	before, _ := set.Store.Get("cost-rule")

	resolved := set.ResolveCitations("按 [MEM:cost-rule] 汇总，参考 [MEM:missing-key]。", "")
	if len(resolved) != 1 || resolved[0] != "cost-rule" {
		t.Fatalf("应只命中真实存在的记忆: %v", resolved)
	}
	after, _ := set.Store.Get("cost-rule")
	if !after.LastUsedAt.After(before.LastUsedAt) {
		t.Fatalf("命中记忆应 Touch 更新 last_used_at: %v -> %v", before.LastUsedAt, after.LastUsedAt)
	}

	// 未知键与空文本静默
	if got := set.ResolveCitations("[MEM:missing-key]", ""); got != nil {
		t.Fatalf("未知键应静默丢弃: %v", got)
	}
	if got := set.ResolveCitations("", ""); got != nil {
		t.Fatalf("空文本应返回 nil: %v", got)
	}
	var nilSet *Set
	if got := nilSet.ResolveCitations("[MEM:a]", ""); got != nil {
		t.Fatalf("nil Set 应返回 nil: %v", got)
	}
}

// S1.2 B 读端隔离器：citations 解析限定本空间——本空间键命中并 Touch，跨空间
// 键等同未知键静默不命中不 Touch，space="" 全空间（旧行为）。
func TestResolveCitationsSpaceIsolation(t *testing.T) {
	set := newCitationTestSet(t)
	// 额外落一条 play 空间事实
	if _, err := set.Store.Save(Memory{
		Name: "play-note", Title: "乐园笔记",
		Description: "游戏偏好", Type: TypeProject, Kind: KindSemantic, Space: "play",
	}); err != nil {
		t.Fatal(err)
	}

	// play 空间：play-note 命中并 Touch；cost-rule（work）跨空间不命中不 Touch
	resolved := set.ResolveCitations("玩 [MEM:play-note]，另见 [MEM:cost-rule]。", "play")
	if len(resolved) != 1 || resolved[0] != "play-note" {
		t.Fatalf("play 空间应只命中 play-note: %v", resolved)
	}
	playAfter, _ := set.Store.GetInSpace("play-note", "play")
	if playAfter.LastUsedAt.IsZero() {
		t.Fatal("play 空间命中应 Touch 更新 last_used_at")
	}
	costBefore, _ := set.Store.Get("cost-rule")
	if !costBefore.LastUsedAt.IsZero() {
		t.Fatalf("跨空间键不应被 Touch，cost-rule last_used_at = %v", costBefore.LastUsedAt)
	}

	// work 空间：cost-rule 命中，play-note 跨空间不命中
	workResolved := set.ResolveCitations("按 [MEM:cost-rule] 与 [MEM:play-note] 汇总。", "work")
	if len(workResolved) != 1 || workResolved[0] != "cost-rule" {
		t.Fatalf("work 空间应只命中 cost-rule: %v", workResolved)
	}
	costAfter, _ := set.Store.Get("cost-rule")
	if costAfter.LastUsedAt.IsZero() {
		t.Fatal("work 空间命中 cost-rule 应 Touch")
	}

	// space=""：全空间旧行为，两键都命中
	allResolved := set.ResolveCitations("[MEM:cost-rule] 与 [MEM:play-note]。", "")
	if len(allResolved) != 2 {
		t.Fatalf("space=\"\" 应全空间命中: %v", allResolved)
	}
}

func TestRecallBlockCitationTokens(t *testing.T) {
	set := newCitationTestSet(t)

	block := set.RecallBlock("成本测算 汇总", 800)
	if block == "" {
		t.Fatal("应有注入块")
	}
	if !strings.Contains(block, "[MEM:引用键]") {
		t.Fatalf("块头应带引用纪律: %s", block)
	}
	if !strings.Contains(block, "[MEM:cost-rule]") {
		t.Fatalf("注入行应带引用键: %s", block)
	}
	// 注入行内出现的引用键应能被回传解析器识别（格式同构）
	if names := ExtractCitationNames(block); len(names) == 0 {
		t.Fatalf("注入块引用键应可被解析器识别: %s", block)
	}
}

func TestFormatRecallLineStaleHint(t *testing.T) {
	// 陈旧记忆（120 天未修订）：行尾带时效提示与引用键
	stale := formatRecallLine(Memory{
		Name: "stale-fact", Title: "陈旧事实", Description: "很久没有修订",
		Type: TypeProject, Kind: KindSemantic,
		UpdatedAt: time.Now().Add(-120 * 24 * time.Hour),
	})
	if !strings.Contains(stale, "（已 120 天未修订，注意时效）") {
		t.Fatalf("陈旧记忆应有时效提示: %s", stale)
	}
	if !strings.Contains(stale, "[MEM:stale-fact]") {
		t.Fatalf("注入行应带引用键: %s", stale)
	}
	// 新鲜记忆（89 天，阈值内）与零值时间：无时效提示
	fresh := formatRecallLine(Memory{
		Name: "fresh-fact", Title: "新事实", Description: "d",
		Type: TypeProject, Kind: KindSemantic,
		UpdatedAt: time.Now().Add(-89 * 24 * time.Hour),
	})
	if strings.Contains(fresh, "未修订") {
		t.Fatalf("阈值内不应提示时效: %s", fresh)
	}
	unknown := formatRecallLine(Memory{Name: "no-time", Type: TypeProject, Kind: KindSemantic, Description: "d"})
	if strings.Contains(unknown, "未修订") {
		t.Fatalf("未知修订时间不应提示时效: %s", unknown)
	}
}
