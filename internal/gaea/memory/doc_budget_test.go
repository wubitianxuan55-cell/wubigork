package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// C6 项目说明文件（AGENTS.md 模式，蒸馏 codex agents_md.rs）：
//   - .gaea/AGENTS.md 子目录约定自动发现（项目层）
//   - 单文件字节预算（对齐 codex project_doc_max_bytes 默认 32KB），超长截断留标记

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverGaeaSubdirDoc(t *testing.T) {
	cwd := t.TempDir()
	write(t, filepath.Join(cwd, ".git"), "gitdir: x")
	write(t, filepath.Join(cwd, ".gaea", "AGENTS.md"), "gaea 子目录项目约定")

	sources := discoverDocs(cwd, "")
	found := false
	for _, s := range sources {
		if s.Scope == ScopeProject && strings.HasSuffix(filepath.ToSlash(s.Path), "/.gaea/AGENTS.md") {
			if !strings.Contains(s.Body, "gaea 子目录项目约定") {
				t.Fatalf(".gaea/AGENTS.md 内容缺失: %q", s.Body)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("应发现 .gaea/AGENTS.md（项目层），实际 %d 个来源", len(sources))
	}
}

func TestDiscoverPrecedenceFlatThenGaeaSubdir(t *testing.T) {
	cwd := t.TempDir()
	write(t, filepath.Join(cwd, ".git"), "gitdir: x")
	write(t, filepath.Join(cwd, "AGENTS.md"), "平铺约定")
	write(t, filepath.Join(cwd, ".gaea", "AGENTS.md"), "子目录约定")

	sources := discoverDocs(cwd, "")
	flat, sub := -1, -1
	for i, s := range sources {
		if s.Scope != ScopeProject {
			continue
		}
		name := filepath.ToSlash(s.Path)
		if strings.HasSuffix(name, "/AGENTS.md") && !strings.Contains(name, "/.gaea/") && flat < 0 {
			flat = i
		}
		if strings.Contains(name, "/.gaea/AGENTS.md") && sub < 0 {
			sub = i
		}
	}
	if flat < 0 || sub < 0 {
		t.Fatalf("两个文档都应被发现: flat=%d sub=%d", flat, sub)
	}
	if sub < flat {
		t.Fatalf("子目录约定应排在平铺名之后（更具体者后注入）: flat=%d sub=%d", flat, sub)
	}
}

func TestTruncateDocBudget(t *testing.T) {
	// 未超限：原样返回
	short := strings.Repeat("字", 100)
	if got := truncateDoc(short, "AGENTS.md"); got != short {
		t.Fatal("预算内不应截断")
	}

	// 超限：在字节边界内截断 + 标记注释；不得出现半个 UTF-8 字符
	long := strings.Repeat("办", docMaxBytes/3+10) // 每个「办」3 字节，超限
	got := truncateDoc(long, "AGENTS.md")
	if len(got) > docMaxBytes+200 {
		t.Fatalf("截断后应接近预算: %d", len(got))
	}
	if !strings.Contains(got, "truncated") || !strings.Contains(got, "32KB") {
		t.Fatalf("截断后应带标记注释: %q", got[len(got)-120:])
	}
	// 截断点必须是合法 UTF-8（Go string 按合法 rune 遍历不报错即为证）
	for _, r := range got[:len(got)-len("\n\n<!-- truncated: 文档超出 32KB 注入预算，已截断（AGENTS.md） -->")] {
		if r == 0xFFFD {
			t.Fatal("截断点出现了无效 UTF-8 字符")
		}
	}
}
