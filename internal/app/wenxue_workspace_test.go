package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateWenxueRuntime(t *testing.T) (*App, string) {
	t.Helper()
	restore := workspaceTestIsolate(t)
	t.Cleanup(restore)
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", root)

	oldCfg := ga.cfg
	ga.cfg = nil
	ga.mu.Lock()
	oldCtrl := ga.ctrl
	ga.ctrl = nil
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		if ga.ctrl != nil {
			ga.ctrl.Close()
		}
		ga.ctrl = oldCtrl
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})
	return &App{}, root
}

func TestWenxueCreateNovelStaysUnderTuwenRoot(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	created, err := a.WenxueCreateNovel("长夜灯火", "")
	if err != nil {
		t.Fatalf("WenxueCreateNovel: %v", err)
	}
	want := filepath.Join(root, "长夜灯火")
	if created.Path != want || created.Name != "长夜灯火" || !created.Current {
		t.Fatalf("created = %+v, want path=%q current", created, want)
	}
	if info, err := os.Stat(filepath.Join(want, ".gaea")); err != nil || !info.IsDir() {
		t.Fatalf("novel .gaea directory missing: info=%v err=%v", info, err)
	}

	list := a.WenxueListWorkspaces()
	if len(list) != 1 || list[0].Path != want || !list[0].Current {
		t.Fatalf("WenxueListWorkspaces = %+v, want the current novel only", list)
	}

	if _, err := a.WenxueCreateNovel("长夜灯火", ""); err == nil {
		t.Fatal("duplicate novel name should fail")
	}
	if _, err := a.WenxueCreateNovel(`..\逃逸`, ""); err == nil {
		t.Fatal("path-like novel name should fail")
	}
	for _, bad := range []string{".隐藏", "CON", "NUL.md", "星夜."} {
		if _, err := a.WenxueCreateNovel(bad, ""); err == nil {
			t.Fatalf("invalid Windows directory name %q should fail", bad)
		}
	}
}

func TestWenxueSwitchWorkspaceRejectsOutsideRoot(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	novel := filepath.Join(root, "甲小说")
	if err := os.MkdirAll(novel, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := a.WenxueSwitchWorkspace(novel); got != novel {
		t.Fatalf("WenxueSwitchWorkspace = %q, want %q", got, novel)
	}

	outside := t.TempDir()
	if got := a.WenxueSwitchWorkspace(outside); got != novel {
		t.Fatalf("outside switch = %q, want current %q", got, novel)
	}
	if got := wenxueWorkspace(); got != novel {
		t.Fatalf("wenxueWorkspace = %q, want %q", got, novel)
	}
}

func TestWenxueProjectGroupsOnlyContainNovelDirectories(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	for _, name := range []string{"乙小说", "甲小说"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "不是小说.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	current := filepath.Join(root, "甲小说")
	if got := a.WenxueSwitchWorkspace(current); got != current {
		t.Fatalf("WenxueSwitchWorkspace = %q, want %q", got, current)
	}
	groups := a.WenxueListProjectSessions()
	if len(groups) != 2 {
		t.Fatalf("WenxueListProjectSessions = %+v, want 2 novel groups", groups)
	}
	if groups[0].Path != current || !groups[0].Current {
		t.Fatalf("first group = %+v, want current 甲小说", groups[0])
	}
	if groups[0].Name == ".hidden" || groups[1].Name == ".hidden" {
		t.Fatalf("hidden directory leaked into groups: %+v", groups)
	}
	if sessions := a.WenxueListSessions(); len(sessions) != 0 {
		t.Fatalf("new novel sessions = %+v, want empty", sessions)
	}
}

func TestWenxueDeleteNovelOnlyDirectChildren(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	first, err := a.WenxueCreateNovel("旧城来信", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.WenxueCreateNovel("长夜灯火", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Path == second.Path {
		t.Fatal("created novels should have distinct paths")
	}
	if err := a.WenxueDeleteNovel(first.Path); err != nil {
		t.Fatalf("delete non-current novel: %v", err)
	}
	if _, err := os.Stat(first.Path); !os.IsNotExist(err) {
		t.Fatalf("deleted novel still exists: %v", err)
	}
	if got := wenxueWorkspace(); got != second.Path {
		t.Fatalf("current novel changed after deleting another: got %q want %q", got, second.Path)
	}

	if err := a.WenxueDeleteNovel(root); err == nil {
		t.Fatal("deleting tuwen root should fail")
	}
	if err := a.WenxueDeleteNovel(t.TempDir()); err == nil {
		t.Fatal("deleting outside path should fail")
	}
	nested := filepath.Join(second.Path, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := a.WenxueDeleteNovel(nested); err == nil {
		t.Fatal("deleting nested path should fail")
	}

	if err := a.WenxueDeleteNovel(second.Path); err != nil {
		t.Fatalf("delete current novel: %v", err)
	}
	if got := wenxueWorkspace(); got != root {
		t.Fatalf("current workspace after deleting current novel = %q, want root %q", got, root)
	}
}

// TestWenxueCreateNovel_BookInfoAndPin 关键信息刀（2026-11-16）：infoJSON 落
// 书籍信息.md（含成人等级+分级说明）并自动固定（pinned.json）；非无限制档
// 附恒守线，无限制档零安全底线；非法成人等级报错并列出合法值；空 infoJSON
// 走全年龄缺省不落空文件。
func TestWenxueCreateNovel_BookInfoAndPin(t *testing.T) {
	a, _ := isolateWenxueRuntime(t)
	created, err := a.WenxueCreateNovel("雷雨长街", `{"genre":"都市悬疑","intro":"一个雨夜引发的连环旧案。","targetWords":"80 万","adultLevel":"成人（限制级）"}`)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(created.Path, "书籍信息.md"))
	if err != nil {
		t.Fatalf("书籍信息.md 缺失: %v", err)
	}
	card := string(raw)
	for _, want := range []string{
		"# 书籍信息", "- 书名：雷雨长街", "- 题材：都市悬疑", "- 目标字数：80 万",
		"- 成人等级：成人（限制级）", "## 简介", "一个雨夜引发的连环旧案。",
		"可直接描写成人内容与暴力", "唯一恒守线：涉未成年角色的性内容一律禁止",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("书籍信息缺 %q", want)
		}
	}
	pinRaw, err := os.ReadFile(filepath.Join(created.Path, ".gaea", "pinned.json"))
	if err != nil {
		t.Fatalf("pinned.json 缺失: %v", err)
	}
	if !strings.Contains(string(pinRaw), "书籍信息.md") {
		t.Fatalf("书籍信息.md 应自动固定: %s", pinRaw)
	}

	t.Run("无限制档零安全底线", func(t *testing.T) {
		created3, err := a.WenxueCreateNovel("无底线书", `{"adultLevel":"深度成人（无限制）"}`)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		raw3, err := os.ReadFile(filepath.Join(created3.Path, "书籍信息.md"))
		if err != nil {
			t.Fatalf("读书籍信息: %v", err)
		}
		card3 := string(raw3)
		if !strings.Contains(card3, "- 成人等级：深度成人（无限制）") {
			t.Fatalf("缺成人等级行: %s", card3)
		}
		for _, want := range []string{"无任何安全底线", "无禁止题材", "无限制档执行契约", "零拒绝", "冲突裁决"} {
			if !strings.Contains(card3, want) {
				t.Errorf("无限制档缺「%s」: %s", want, card3)
			}
		}
		for _, ban := range []string{"唯一恒守线", "涉未成年角色的性内容一律禁止"} {
			if strings.Contains(card3, ban) {
				t.Errorf("无限制档不得写恒守线「%s」: %s", ban, card3)
			}
		}
	})

	t.Run("非法成人等级报错列合法值", func(t *testing.T) {
		_, err := a.WenxueCreateNovel("坏等级书", `{"adultLevel":"NC-17"}`)
		if err == nil || !strings.Contains(err.Error(), "成人等级必须是") {
			t.Fatalf("err = %v, want 白名单报错", err)
		}
	})
	t.Run("空信息缺省全年龄", func(t *testing.T) {
		created2, err := a.WenxueCreateNovel("无信息书", "")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		raw2, err := os.ReadFile(filepath.Join(created2.Path, "书籍信息.md"))
		if err != nil {
			t.Fatalf("读书籍信息: %v", err)
		}
		if !strings.Contains(string(raw2), "- 成人等级：全年龄") {
			t.Fatalf("空等级应缺省全年龄: %s", raw2)
		}
	})
}
