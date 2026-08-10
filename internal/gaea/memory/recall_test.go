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
