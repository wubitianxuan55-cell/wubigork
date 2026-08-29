package board

import (
	"reflect"
	"strings"
	"testing"
)

// TestManifestCompleteness manifest 完整性：canonical 清单通过 ValidateAll，
// 且包含 §3.1 的 9 个 canonical 板块 + D7 的 knowledge 独立板块。
func TestManifestCompleteness(t *testing.T) {
	manifests := BuiltinManifests()
	if err := ValidateAll(manifests); err != nil {
		t.Fatalf("canonical manifest 完整性校验失败: %v", err)
	}
	ids := map[string]bool{}
	for _, m := range manifests {
		ids[m.ID] = true
		if m.Page == "" && m.ID != "weixin" {
			t.Errorf("板块 %q 缺 page（weixin 除外）", m.ID)
		}
		for _, b := range m.Bindings {
			if !strings.HasSuffix(b, "B") {
				t.Errorf("板块 %q 绑定门面 %q 命名不符合 *B 约定", m.ID, b)
			}
		}
		for _, it := range m.Intents {
			if it.ID == "" || it.Handler == "" {
				t.Errorf("板块 %q 意图 %+v 缺 id/handler", m.ID, it)
			}
		}
	}
	for _, want := range CanonicalIDs() {
		if !ids[want] {
			t.Errorf("canonical 清单缺少板块 %q", want)
		}
	}
	if !ids["knowledge"] {
		t.Errorf("D7 决策：knowledge 应恢复挂载为独立板块，但清单缺失")
	}
	// weixin 服务板块必须在列（§3.1 层间不一致 #3）
	if !ids["weixin"] {
		t.Errorf("weixin 服务板块缺失")
	}
}

// TestManifestDuplicateIDRejected 重复 id 拒绝：ValidateAll 必须报错。
func TestManifestDuplicateIDRejected(t *testing.T) {
	dup := []Manifest{
		{ID: "chat", Label: "AI 聊天", Icon: "I", Page: "p"},
		{ID: "chat", Label: "另一个聊天", Icon: "I", Page: "p"},
	}
	err := ValidateAll(dup)
	if err == nil || !strings.Contains(err.Error(), "重复 id") {
		t.Fatalf("重复 id 应被拒绝，got err = %v", err)
	}
}

// TestManifestSpaceAssignments S2.1 双空间壳：canonical 板块空间归属表
// （docs/gaea-space-shell-design.md §3）逐项断言，防止新增板块漏标空间。
func TestManifestSpaceAssignments(t *testing.T) {
	want := map[string]string{
		"chat":         SpaceShared,
		"novel":        SpacePlay,
		"imagegen":     SpacePlay,
		"gaea":         SpaceWork,
		"cost":         SpaceWork,
		"code":         SpaceIndependent,
		"memoryhub":    SpaceWork,
		"modelcenter":  SpaceShared,
		"characterlib": SpacePlay,
		"settings":     SpaceShared,
		"weixin":       SpaceWork,
		"knowledge":    SpaceWork,
	}
	got := map[string]string{}
	for _, m := range BuiltinManifests() {
		got[m.ID] = m.Space
	}
	for id, space := range want {
		if got[id] != space {
			t.Errorf("板块 %q space = %q, want %q", id, got[id], space)
		}
	}
	if err := ValidateAll(BuiltinManifests()); err != nil {
		t.Fatalf("space 赋值后清单校验失败: %v", err)
	}
}

// TestManifestSpaceValidation 非法 space 拒绝；空串（旧数据兼容）放行。
func TestManifestSpaceValidation(t *testing.T) {
	for _, bad := range []string{"home", "WORK", "office", "garden"} {
		m := Manifest{ID: "x", Label: "X", Icon: "I", Page: "p", Space: bad}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "space") {
			t.Errorf("space %q 应报错，got err = %v", bad, err)
		}
	}
	for _, ok := range []string{"", SpaceWork, SpacePlay, SpaceShared, SpaceIndependent} {
		m := Manifest{ID: "x", Label: "X", Icon: "I", Page: "p", Space: ok}
		if err := m.Validate(); err != nil {
			t.Errorf("space %q 应放行，got err = %v", ok, err)
		}
	}
}

// TestManifestIntentWithoutHandlerRejected 意图无 handler 必须报错
// （缺陷 2 的机器保证：intent 无 handler 启动即报错，不静默）。
func TestManifestIntentWithoutHandlerRejected(t *testing.T) {
	m := Manifest{ID: "x", Label: "X", Icon: "I", Page: "p",
		Intents: []IntentDecl{{ID: "teleport"}}}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "无 handler") {
		t.Fatalf("意图无 handler 应报错，got err = %v", err)
	}
}

// TestBuiltinBoardsConform Board 接口一致性：Builtins 返回的每个板块，
// 其访问器必须与 manifest 单一数据源一致，且 Init 为声明式空装配。
func TestBuiltinBoardsConform(t *testing.T) {
	manifests := BuiltinManifests()
	boards := Builtins()
	if len(boards) != len(manifests) {
		t.Fatalf("Builtins 数量 %d != BuiltinManifests 数量 %d", len(boards), len(manifests))
	}
	byID := map[string]Manifest{}
	for _, m := range manifests {
		byID[m.ID] = m
	}
	for _, b := range boards {
		m, ok := byID[b.ID()]
		if !ok {
			t.Fatalf("板块 %q 不在 manifest 清单中", b.ID())
		}
		if b.Name() != m.Label {
			t.Errorf("板块 %q Name=%q != manifest.Label=%q", b.ID(), b.Name(), m.Label)
		}
		if b.Icon() != m.Icon {
			t.Errorf("板块 %q Icon=%q != manifest.Icon=%q", b.ID(), b.Icon(), m.Icon)
		}
		if b.PageKey() != m.Page {
			t.Errorf("板块 %q PageKey=%q != manifest.Page=%q", b.ID(), b.PageKey(), m.Page)
		}
		if !reflect.DeepEqual(b.Bindings(), m.Bindings) {
			t.Errorf("板块 %q Bindings=%v != manifest.Bindings=%v", b.ID(), b.Bindings(), m.Bindings)
		}
		if !reflect.DeepEqual(b.Tools(), m.Tools) {
			t.Errorf("板块 %q Tools=%v != manifest.Tools=%v", b.ID(), b.Tools(), m.Tools)
		}
		if !reflect.DeepEqual(b.Intents(), m.IntentIDs()) {
			t.Errorf("板块 %q Intents=%v != manifest.IntentIDs=%v", b.ID(), b.Intents(), m.IntentIDs())
		}
		if err := b.Init(nil); err != nil {
			t.Errorf("板块 %q Init 应为空装配: %v", b.ID(), err)
		}
	}
}

// TestBuiltinBoardsHaveIntentHandlers 每个声明意图的板块，其 handler 方法名非空
// 且指向 App 上真实存在的方法（gen_bindings 生成的 TestManifestIntentHandlersExist
// 会在 app 包做反射兜底；这里先做数据层断言）。
func TestBuiltinBoardsHaveIntentHandlers(t *testing.T) {
	for _, m := range BuiltinManifests() {
		for _, it := range m.Intents {
			if it.Handler == "" {
				t.Errorf("板块 %q 意图 %q 无 handler", m.ID, it.ID)
			}
		}
	}
}
