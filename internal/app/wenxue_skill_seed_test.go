package app

// wenxue_skill_seed_test.go — 播种分发链行为锁（2026-11-17 审查批5 刀2；
// 2026-11-23 随五件写作域技能退役收敛为无限制档专属件单件）。

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
	for _, name := range wenxueSeedSkillNamesUnrestricted {
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

// TestWenxueSeedBookSkills 播种语义（2026-11-23 收敛后）：非无限制书零落盘/
// 无限制书只落专属件/幂等/用户改动保留/空根拒绝。
func TestWenxueSeedBookSkills(t *testing.T) {
	// 非无限制书：什么都不落（全档位三件已退役，播种链只携带专属件）。
	root := t.TempDir()
	if n := wenxueSeedBookSkills(root, false); n != 0 {
		t.Fatalf("非无限制书不应播种任何文件，got %d", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".gaea", "skills")); !os.IsNotExist(err) {
		t.Fatalf("非无限制书不应建 .gaea/skills: %v", err)
	}

	// 无限制书：专属件落盘且幂等。
	nuRoot := t.TempDir()
	if nU := wenxueSeedBookSkills(nuRoot, true); nU == 0 {
		t.Fatal("无限制书首播应落盘文件")
	}
	skillMd := filepath.Join(nuRoot, ".gaea", "skills", "novel-chapter-branches", "SKILL.md")
	if _, err := os.Stat(skillMd); err != nil {
		t.Fatalf("无限制书未播种 novel-chapter-branches: %v", err)
	}
	// 已退役三件不得回潜播种。
	for _, retired := range []string{"novel-agents", "novel-deslop", "novel-review"} {
		if _, err := os.Stat(filepath.Join(nuRoot, ".gaea", "skills", retired)); !os.IsNotExist(err) {
			t.Fatalf("已退役技能 %s 不应再播种: %v", retired, err)
		}
	}
	if nU2 := wenxueSeedBookSkills(nuRoot, true); nU2 != 0 {
		t.Fatalf("无限制书二次播种应幂等（count=0），got %d", nU2)
	}

	// 用户改动保留：改专属件后再播种不回滚。
	if err := os.WriteFile(skillMd, []byte(`{"user":"edit"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = wenxueSeedBookSkills(nuRoot, true)
	body, _ := os.ReadFile(skillMd)
	if string(body) != `{"user":"edit"}` {
		t.Fatal("用户改动被播种覆盖（应先到先得）")
	}

	// 空根诚实拒绝。
	if n3 := wenxueSeedBookSkills("   ", true); n3 != 0 {
		t.Fatalf("空根应拒绝播种，got %d", n3)
	}
	// 目录足迹：只允许落在 .gaea/skills 子树内。
	if err := filepath.WalkDir(nuRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(nuRoot, p)
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "novel-") && !strings.HasPrefix(rel, ".gaea/skills/") {
			t.Errorf("播种越界足迹: %s", rel)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
