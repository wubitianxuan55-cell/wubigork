package memory

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestSQLiteLifecycleAndTouch(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	used := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	if _, err := s.Save(Memory{
		Name:          "cost-rule",
		Title:         "成本测算规则",
		Description:   "成本测算先对科目再汇总",
		Type:          TypeProject,
		Kind:          KindProcedural,
		Body:          "1. 对齐科目；2. 金额用公式；3. 出图表。",
		LastUsedAt:    used,
		SourceSession: "session-20260810-demo.jsonl",
		SourceMessage: "turn 3",
	}); err != nil {
		t.Fatal(err)
	}

	m, ok := s.Get("cost-rule")
	if !ok {
		t.Fatal("Get 失败")
	}
	if m.SourceSession != "session-20260810-demo.jsonl" || m.SourceMessage != "turn 3" {
		t.Fatalf("来源字段未持久化: %+v", m)
	}
	if !m.LastUsedAt.Equal(used) {
		t.Fatalf("last_used_at 未持久化: %v", m.LastUsedAt)
	}

	// Touch 更新最近使用时间
	if err := s.Touch("cost-rule"); err != nil {
		t.Fatal(err)
	}
	if m2, _ := s.Get("cost-rule"); m2.LastUsedAt.Before(used) || m2.LastUsedAt.Equal(used) {
		t.Fatalf("Touch 未更新 last_used_at: %v", m2.LastUsedAt)
	}

	// 覆盖保存不重置 last_used_at（修订内容 ≠ 使用）
	if _, err := s.Save(Memory{Name: "cost-rule", Description: "修订", Type: TypeProject, Kind: KindProcedural, Body: "新版"}); err != nil {
		t.Fatal(err)
	}
	if m3, _ := s.Get("cost-rule"); m3.LastUsedAt.Before(used) {
		t.Fatalf("修订不应重置 last_used_at: %v", m3.LastUsedAt)
	}
}

func TestRecallBlockRanking(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	// 相关 procedural（成本）
	if _, err := s.Save(Memory{
		Name: "cost-estimate-rule", Title: "成本测算规则",
		Description: "成本测算先对科目再汇总，金额用公式",
		Type:        TypeProject, Kind: KindProcedural,
		Body: "科目 → 数量×单价 → 汇总 → 图表",
	}); err != nil {
		t.Fatal(err)
	}
	// 不相关 semantic
	if _, err := s.Save(Memory{
		Name: "user-hobby", Title: "用户爱好",
		Description: "用户喜欢户外运动",
		Type:        TypeUser, Kind: KindSemantic,
		Body: "周末常去爬山。",
	}); err != nil {
		t.Fatal(err)
	}

	set := &Set{Store: s}
	block := set.RecallBlock("帮我做一份成本测算表", 800)
	if block == "" {
		t.Fatal("RecallBlock 应返回注入块")
	}
	if !strings.Contains(block, "成本测算规则") {
		t.Fatalf("相关 procedural 应注入: %s", block)
	}
	if strings.Contains(block, "户外运动") {
		t.Fatalf("不相关事实不应注入: %s", block)
	}
	if !strings.HasPrefix(block, "## 记忆上下文") {
		t.Fatalf("块头异常: %s", block)
	}
}

func TestRecallBlockBudget(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 8; i++ {
		if _, err := s.Save(Memory{}); err == nil {
			t.Fatal("空名不应保存")
		}
		if _, err := s.Save(Memory{
			Name: "rule-x" + string(rune('a'+i)), Title: "规则",
			Description: strings.Repeat("长描述", 60),
			Type:        TypeProject, Kind: KindProcedural,
			Body: strings.Repeat("方法论正文", 120),
		}); err != nil {
			t.Fatal(err)
		}
	}
	set := &Set{Store: s}
	block := set.RecallBlock("规则 方法论 帮助", 800)
	if utf8.RuneCountInString(block) > 900 {
		t.Fatalf("注入超出预算: %d", utf8.RuneCountInString(block))
	}
	if block == "" {
		t.Fatal("预算内应有注入")
	}
}

func TestProfileBlockBudget(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 6; i++ {
		if _, err := s.Save(Memory{
			Name: "user-pref" + string(rune('a'+i)), Title: "偏好",
			Description: strings.Repeat("用户偏好", 40),
			Type:        TypeUser, Kind: KindSemantic,
			Body: "b",
		}); err != nil {
			t.Fatal(err)
		}
	}
	set := &Set{DB: gdb, Store: s}
	block := set.ProfileBlock()
	if utf8.RuneCountInString(block) > profileBudget+80 {
		t.Fatalf("画像注入超出预算: %d", utf8.RuneCountInString(block))
	}
}

// ─── 召回 ×固化块 emitted 级去重（众神殿 2026-11-28 论证 action-4）──────────
// 已完整进入前缀固化块（Emitted 且未截断）的 pinned 不再逐轮召回；被挤出
// （Omitted）、正文截断（Truncated）、正文空的 pinned 仍走召回——不能按
// m.Pinned 全量跳过，否则固化块溢出时两处皆失。

func newPinnedDedupTestSet(t *testing.T) *Set {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	fix := []Memory{
		{Name: "aaa-emit", Title: "已入块", Kind: KindSemantic, Type: TypeProject,
			Description: "已完整入块固化", Body: "固化正文 alpha-kw 已随前缀在场"},
		{Name: "bbb-trunc", Title: "被截断", Kind: KindSemantic, Type: TypeProject,
			Description: "正文截断固化", Body: "beta-kw " + strings.Repeat("长", pinnedFactsBodyCap)},
		{Name: "ccc-omit", Title: "被挤出", Kind: KindSemantic, Type: TypeProject,
			Description: "块预算溢出固化", Body: strings.Repeat("gamma-kw 固化正文。", 60)},
		{Name: "delta-empty", Title: "空正文固化", Kind: KindSemantic, Type: TypeProject,
			Description: "正文空的固化条 delta-kw", Body: "   "},
	}
	for _, m := range fix {
		if _, err := s.Save(m); err != nil {
			t.Fatalf("save %s: %v", m.Name, err)
		}
		// 固化态是独立操作（Save 不落 pinned 列，与面板「固化」动作同口径）
		if err := s.Pin(m.Name); err != nil {
			t.Fatalf("pin %s: %v", m.Name, err)
		}
	}
	// 预算 700：aaa-emit 与 bbb-trunc（400 rune 截断）入块；ccc-omit 整行放
	// 不下出局；delta-empty 本就不进块（无正文可装配）。
	set := &Set{Store: s}
	res := BuildPinnedFacts(s.List(), 700)
	if !equalStrings(res.Emitted, []string{"aaa-emit", "bbb-trunc"}) ||
		!equalStrings(res.Truncated, []string{"bbb-trunc"}) ||
		!equalStrings(res.Omitted, []string{"ccc-omit"}) {
		t.Fatalf("fixture 发射面不符预期: %+v", res)
	}
	set.SetPinnedFactsState(PinnedFactsState{
		Emitted: res.Emitted, Truncated: res.Truncated, Omitted: res.Omitted,
	})
	return set
}

func TestRecallBlockPinnedDedupByEmitted(t *testing.T) {
	set := newPinnedDedupTestSet(t)
	block := set.RecallBlock("alpha-kw beta-kw gamma-kw delta-kw", 0)
	if block == "" {
		t.Fatal("应有召回块")
	}
	if strings.Contains(block, "[MEM:aaa-emit]") {
		t.Fatalf("已完整入块的 pinned 不应重复召回: %s", block)
	}
	for _, want := range []string{"[MEM:bbb-trunc]", "[MEM:ccc-omit]", "[MEM:delta-empty]"} {
		if !strings.Contains(block, want) {
			t.Fatalf("截断/挤出/空正文 pinned 仍应召回，缺 %s: %s", want, block)
		}
	}
	// 零状态 Set（boot 未装配/直接 Load）：不排任何 pinned——去重只认发射面。
	fresh := &Set{Store: set.Store}
	if block := fresh.RecallBlock("alpha-kw beta-kw gamma-kw", 0); !strings.Contains(block, "[MEM:aaa-emit]") {
		t.Fatalf("零状态 Set 不应去重: %s", block)
	}
}
