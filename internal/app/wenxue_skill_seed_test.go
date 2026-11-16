package app

// wenxue_skill_seed_test.go — 播种分发链行为锁（2026-11-17 审查批5 刀2）。

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWenxueSkillSeedMirrorDrift 资产双源防漂移：internal/app/wenxue_skill_seed
// 快照必须与仓内 .gaea/skills 源逐字节一致（双向：仓源有→嵌入有且同文；
// 嵌入有→仓源有）。改技能资产时两处同批（同「新增工具三处同批」纪律）。
func TestWenxueSkillSeedMirrorDrift(t *testing.T) {
	for _, name := range wenxueSeedSkillNames {
		repoDir := filepath.Join("..", "..", ".gaea", "skills", name)
		embedded := map[string][]byte{}
		srcRoot := "wenxue_skill_seed/" + name
		err := fs.WalkDir(wenxueSkillSeedFS, srcRoot, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(srcRoot, p)
			if err != nil {
				return err
			}
			body, err := fs.ReadFile(wenxueSkillSeedFS, p)
			if err != nil {
				return err
			}
			embedded[filepath.ToSlash(rel)] = body
			return nil
		})
		if err != nil {
			t.Fatalf("读嵌入资产 %s: %v", name, err)
		}
		seen := map[string]bool{}
		err = filepath.WalkDir(repoDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(repoDir, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			seen[rel] = true
			body, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			emb, ok := embedded[rel]
			if !ok {
				t.Errorf("%s/%s 在仓源有、嵌入快照缺——重新拷贝进 internal/app/wenxue_skill_seed", name, rel)
				return nil
			}
			if string(emb) != string(body) {
				t.Errorf("%s/%s 嵌入快照与仓源漂移（字节不一致）——两处同批更新", name, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("走仓源 %s: %v", name, err)
		}
		for rel := range embedded {
			if !seen[rel] {
				t.Errorf("%s/%s 在嵌入快照有、仓源缺——孤儿资产", name, rel)
			}
		}
		if len(embedded) == 0 {
			t.Errorf("%s 嵌入快照为空", name)
		}
	}
}

// TestWenxueSeedBookSkills 播种语义：只补缺失/幂等/用户改动保留/空根拒绝。
func TestWenxueSeedBookSkills(t *testing.T) {
	root := t.TempDir()

	n := wenxueSeedBookSkills(root)
	if n == 0 {
		t.Fatal("首播应落盘文件")
	}
	skillMd := filepath.Join(root, ".gaea", "skills", "novel-agents", "SKILL.md")
	if _, err := os.Stat(skillMd); err != nil {
		t.Fatalf("novel-agents SKILL.md 未播种: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gaea", "skills", "novel-agents", "agents", "story-architect.md")); err != nil {
		t.Fatalf("派发契约伴随件（角色卡）未播种: %v", err)
	}

	if n2 := wenxueSeedBookSkills(root); n2 != 0 {
		t.Fatalf("二次播种应幂等（count=0），got %d", n2)
	}

	// 用户改动保留：改词表覆盖点后再播种不回滚。
	overwritten := filepath.Join(root, ".gaea", "skills", "novel-deslop", "gates.json")
	if err := os.WriteFile(overwritten, []byte(`{"user":"edit"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = wenxueSeedBookSkills(root)
	body, _ := os.ReadFile(overwritten)
	if string(body) != `{"user":"edit"}` {
		t.Fatal("用户改动被播种覆盖（应先到先得）")
	}

	// 空根诚实拒绝。
	if n3 := wenxueSeedBookSkills("   "); n3 != 0 {
		t.Fatalf("空根应拒绝播种，got %d", n3)
	}
	// 目录足迹：只允许落在 .gaea/skills 子树内。
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "novel-") && !strings.HasPrefix(rel, ".gaea/skills/") {
			t.Errorf("播种越界足迹: %s", rel)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
