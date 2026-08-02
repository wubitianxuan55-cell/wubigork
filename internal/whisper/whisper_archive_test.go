// Package whisper — whisper_archive_test.go
// P3 archiveExporter：记忆归档导出测试（对齐 ackem memory/archiveExporter.ts）

package whisper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildArchive_EmptyStore(t *testing.T) {
	fs := NewFactStore()
	files := BuildArchive(fs, nil)

	if len(files) != 1 {
		t.Fatalf("空库应只有 README, got %d 个文件", len(files))
	}
	if files[0].Path != "README.md" {
		t.Errorf("空库首个文件应为 README.md, got %s", files[0].Path)
	}
	if !strings.Contains(files[0].Content, "事实总数") {
		t.Errorf("README 应含统计信息")
	}
}

func TestBuildArchive_OrganizesByDomainAndSubcategory(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "最好的朋友叫阿珍",
		Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, Tier: "core",
		CreatedAt: time.Now(),
	})
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FAMILY", Subject: "用户", Summary: "家里养了一只猫",
		Weight: 2.5, Confidence: 0.85, SelfRelevance: 0.7,
		CreatedAt: time.Now(),
	})
	fs.Add(MemoryFact{
		Domain: "IDENTITY", Subcategory: "BASIC_PROFILE", Subject: "用户", Summary: "叫小明",
		Weight: 3, Confidence: 0.9, SelfRelevance: 1,
		CreatedAt: time.Now(),
	})

	files := BuildArchive(fs, nil)

	byPath := map[string]string{}
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	// README 存在
	if _, ok := byPath["README.md"]; !ok {
		t.Fatal("缺少 README.md")
	}
	// 领域/子类分目录
	if _, ok := byPath["SOCIAL/FRIENDS.md"]; !ok {
		t.Errorf("缺少 SOCIAL/FRIENDS.md, 实际路径: %v", keys(byPath))
	}
	if _, ok := byPath["SOCIAL/FAMILY.md"]; !ok {
		t.Errorf("缺少 SOCIAL/FAMILY.md")
	}
	if _, ok := byPath["IDENTITY/BASIC_PROFILE.md"]; !ok {
		t.Errorf("缺少 IDENTITY/BASIC_PROFILE.md")
	}
	// 内容包含事实摘要
	if !strings.Contains(byPath["SOCIAL/FRIENDS.md"], "阿珍") {
		t.Errorf("SOCIAL/FRIENDS.md 应含事实摘要")
	}
	// README 统计包含领域分布（展示名 Social）
	if !strings.Contains(byPath["README.md"], "Social") {
		t.Errorf("README 应含领域统计")
	}
}

func TestBuildArchive_ExcludesRetired(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "活跃事实",
		Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, CreatedAt: time.Now(),
	})
	retired := fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "已退役事实",
		Weight: 0.2, Confidence: 0.2, SelfRelevance: 0.8, CreatedAt: time.Now(),
	})
	fs.RetireFact(retired.ID)

	files := BuildArchive(fs, nil)
	for _, f := range files {
		if strings.Contains(f.Content, "已退役事实") {
			t.Errorf("退役事实不应进入归档: %s", f.Path)
		}
	}
}

func TestBuildArchive_IncludesEpisodes(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "和阿珍常去爬山",
		Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, CreatedAt: time.Now(),
	})
	es := NewEpisodicStore()
	es.Add(Episode{Summary: "一起喝了咖啡", Keywords: []string{"咖啡"}, EmotionalIntensity: 0.7, CreatedAt: time.Now()})

	files := BuildArchive(fs, es)
	readme := files[0].Content
	if !strings.Contains(readme, "情节") {
		t.Errorf("README 应含情节统计, got: %s", readme)
	}
	if !strings.Contains(readme, "1") {
		t.Errorf("README 应含情节数量 1")
	}
}

func TestWriteArchive_WritesFiles(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "最好的朋友叫阿珍",
		Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, CreatedAt: time.Now(),
	})

	dir := t.TempDir()
	n, err := WriteArchive(fs, nil, dir)
	if err != nil {
		t.Fatalf("WriteArchive 失败: %v", err)
	}
	if n < 2 {
		t.Errorf("应写入 README + 1 个子类文件, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Errorf("README.md 未写入: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SOCIAL", "FRIENDS.md")); err != nil {
		t.Errorf("SOCIAL/FRIENDS.md 未写入: %v", err)
	}
}

func TestWriteArchive_NoFactNoPanic(t *testing.T) {
	dir := t.TempDir()
	n, err := WriteArchive(NewFactStore(), nil, dir)
	if err != nil {
		t.Fatalf("空库导出不应报错: %v", err)
	}
	if n != 1 {
		t.Errorf("空库应只写 README, got %d", n)
	}
}

// keys 辅助：列出 map 键
func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
