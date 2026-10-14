package memory

// 阶段九 9.3 测试：失效标记生命周期（不删原文/注入排除/可恢复/编辑复活）+
// 冲突比对（同题自动失效候选/BM25 疑似/排除同名与已失效）。

import (
	"strings"
	"testing"
	"time"
)

func invalidateTestStore(t *testing.T) Set {
	t.Helper()
	// 真实 SQLite 库（GetDatabase 自带迁移链，临时目录隔离；graphTestDB 同款）：
	// 生命周期持久化口径的适用前提。
	gdb := graphTestDB(t)
	s := Load(Options{CWD: t.TempDir(), UserDir: t.TempDir(), DB: gdb})
	return *s
}

func mustSave(t *testing.T, s Set, m Memory) {
	t.Helper()
	if _, err := s.Store.Save(m); err != nil {
		t.Fatalf("save %s: %v", m.Name, err)
	}
}

func TestInvalidateLifecycle_OriginalKept(t *testing.T) {
	s := invalidateTestStore(t)
	mustSave(t, s, Memory{Name: "project-用-pnpm", Title: "包管理器", Description: "构建工具选型", Type: TypeProject, Body: "本项目使用 pnpm 管理依赖。", Space: "work"})

	// 失效：不删原文
	if err := s.Store.Invalidate("project-用-pnpm", "被 npm 条目推翻"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	got, ok := s.Store.Get("project-用-pnpm")
	if !ok {
		t.Fatal("失效后原条目必须仍可查（不删红线）")
	}
	if got.Body != "本项目使用 pnpm 管理依赖。" {
		t.Fatalf("原文不得改动: %q", got.Body)
	}
	if got.InvalidAt.IsZero() {
		t.Fatal("InvalidAt 应非空")
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt 应回填")
	}

	// 注入面排除：RecallBlock 不含失效条目
	mustSave(t, s, Memory{Name: "unrelated", Title: "无关条目", Description: "别的话题", Type: TypeProject, Body: "完全无关的内容 pnpm 关键词命中测试。", Space: "work"})
	block := s.RecallBlock("pnpm", 0)
	if strings.Contains(block, "被 npm 条目推翻") || strings.Contains(block, "project-用-pnpm") {
		t.Fatalf("失效条目不得注入:\n%s", block)
	}

	// 恢复：重新参与注入
	if err := s.Store.RestoreInvalidate("project-用-pnpm"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	got, _ = s.Store.Get("project-用-pnpm")
	if !got.InvalidAt.IsZero() {
		t.Fatalf("恢复后 InvalidAt 应清空: %+v", got)
	}
	block = s.RecallBlock("pnpm", 0)
	// 渲染行=Title+引用键（[MEM:...]），按 Title 断言
	if !strings.Contains(block, "包管理器") {
		t.Fatalf("恢复后应重新注入:\n%s", block)
	}
}

func TestInvalidate_RevisionRevives(t *testing.T) {
	s := invalidateTestStore(t)
	mustSave(t, s, Memory{Name: "fact-a", Title: "事实甲", Description: "d", Type: TypeProject, Body: "旧内容。", Space: "work"})
	if err := s.Store.Invalidate("fact-a", "测试"); err != nil {
		t.Fatal(err)
	}
	// 同名修订（编辑复活）：invalid_at 清空
	mustSave(t, s, Memory{Name: "fact-a", Title: "事实甲", Description: "d", Type: TypeProject, Body: "新内容（作者重申）。", Space: "work"})
	got, _ := s.Store.Get("fact-a")
	if !got.InvalidAt.IsZero() {
		t.Fatalf("修订应复活（清失效）: %+v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("修订不得重置 CreatedAt（首次入库时间）")
	}
}

func TestInvalidate_FileBackendHonestError(t *testing.T) {
	// file 后端：生命周期不持久化口径——诚实报错不假装成功
	s := Set{Store: Store{Dir: t.TempDir()}}
	if err := s.Store.Invalidate("x", "r"); err == nil {
		t.Fatal("file 后端应诚实报错")
	}
}

func TestDetectConflicts_SameTitleConfirmed(t *testing.T) {
	s := invalidateTestStore(t)
	mustSave(t, s, Memory{Name: "old-take", Title: "部署流程", Description: "旧版部署说明", Type: TypeProject, Body: "手工部署：打包后 FTP 上传。", Space: "work"})
	mustSave(t, s, Memory{Name: "unrelated", Title: "别的主题", Description: "无关", Type: TypeProject, Body: "不相干内容。", Space: "work"})

	rep := s.Store.DetectConflicts(Memory{Name: "new-take", Title: "部署流程", Description: "新版部署", Type: TypeProject, Body: "CI/CD 自动部署。"})
	if len(rep.Confirmed) != 1 || rep.Confirmed[0] != "old-take" {
		t.Fatalf("同题旧条目应进 Confirmed: %+v", rep)
	}
	if strings.Contains(rep.Report(), "unrelated") {
		t.Fatalf("无关条目不得出现: %q", rep.Report())
	}
	// 应用层语义：Confirmed 自动失效
	for _, name := range rep.Confirmed {
		if err := s.Store.Invalidate(name, "同题推翻"); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.Store.Get("old-take")
	if got.InvalidAt.IsZero() || got.Body != "手工部署：打包后 FTP 上传。" {
		t.Fatalf("自动失效应保留原文: %+v", got)
	}
	// 重跑比对：已失效条目不再出现（裁决过不重复处理）
	rep2 := s.Store.DetectConflicts(Memory{Name: "new-take-2", Title: "部署流程", Description: "再版", Type: TypeProject, Body: "部署 v3。"})
	if len(rep2.Confirmed) != 0 {
		t.Fatalf("已失效条目不得重复进 Confirmed: %+v", rep2)
	}
}

func TestDetectConflicts_SuspectedBM25(t *testing.T) {
	s := invalidateTestStore(t)
	mustSave(t, s, Memory{Name: "build-notes", Title: "构建备忘", Description: "前端构建", Type: TypeProject, Body: "前端构建使用 vite，产物在 dist 目录。", Space: "work"})

	rep := s.Store.DetectConflicts(Memory{Name: "build-notes-v2", Title: "构建备忘二", Description: "前端构建补充", Type: TypeProject, Body: "前端构建使用 vite，产物目录改为 build。"})
	if len(rep.Suspected) == 0 {
		t.Fatalf("BM25 相似应进 Suspected: %+v", rep)
	}
	// Suspected 不改状态（宁漏勿误——裁决归作者）
	got, _ := s.Store.Get("build-notes")
	if !got.InvalidAt.IsZero() {
		t.Fatalf("Suspected 不得自动失效: %+v", got)
	}
}

func TestDetectConflicts_ExcludesSelfAndInvalid(t *testing.T) {
	s := invalidateTestStore(t)
	now := time.Now()
	mustSave(t, s, Memory{Name: "same-name", Title: "同题", Description: "d", Type: TypeProject, Body: "同名修订不被自己命中。", Space: "work"})
	mustSave(t, s, Memory{Name: "already-invalid", Title: "同题", Description: "d2", Type: TypeProject, Body: "已裁决过的。", Space: "work"})
	_ = s.Store.Invalidate("already-invalid", "先前裁决")
	_ = now

	rep := s.Store.DetectConflicts(Memory{Name: "same-name", Title: "同题", Description: "d3", Type: TypeProject, Body: "修订内容。"})
	for _, c := range rep.Confirmed {
		if c == "same-name" {
			t.Fatal("同名（Save 覆盖语义的修订）不得进 Confirmed")
		}
		if c == "already-invalid" {
			t.Fatal("已失效条目不得进 Confirmed")
		}
	}
}
