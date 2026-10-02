package wspath

// resolve_test.go — 路径归一原语用例（审计 2026-10-02 AP5-09）。
//
// 表驱动矩阵：正反两面都钉死——恶意输入必须被拒（可证明「改坏能红」），
// 合法桌面输入（工作区内相对路径、`sub/../sub` 形态、大小写差异、root 本身）
// 必须照旧通过（防「悄悄收紧到咬人」）。

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// windowsOnly 标记仅在 Windows 上有意义的用例（盘符/UNC/卷相对/反斜杠分隔符）。
func windowsOnly() bool { return runtime.GOOS == "windows" }

// TestWithin 落在根内判据：正面/反面/边界（root 本身、跨盘符、前缀兄弟目录、
// 大小写、空串、未解析形态）。
func TestWithin(t *testing.T) {
	root := filepath.Join(string(filepath.Separator)+"ws", "proj")

	cases := []struct {
		name    string
		root    string
		target  string
		want    bool
		winOnly bool
	}{
		{"root本身", root, root, true, false},
		{"根内文件", root, filepath.Join(root, "a.txt"), true, false},
		{"根内深路径", root, filepath.Join(root, "a", "b", "c.txt"), true, false},
		{"根内带..归一仍在内", root, filepath.Join(root, "a", "..", "b.txt"), true, false},
		// 合法文件「..foo」：Rel 返回 "..foo"，不是越界——这是 filewatch 旧判据
		// strings.HasPrefix(rel, "..") 误丢合法文件的根因（改调 Within 后修正）。
		{"根内..前缀文件", root, filepath.Join(root, "..foo"), true, false},
		{"父目录", root, filepath.Dir(root), false, false},
		{"穿越两级", root, filepath.Join(root, "..", "..", "x"), false, false},
		{"前缀兄弟目录", root, filepath.Join(filepath.Dir(root), "proj-other", "x"), false, false},
		{"空target", root, "", false, false},
		{"空root", "", filepath.Join(root, "a.txt"), false, false},
		{"未解析穿越形态", root, filepath.Join(root, "..", "x"), false, false},
		// Windows 专有：大小写不敏感（Rel 承担）、跨盘符不可比、反斜杠穿越。
		{"大小写不敏感", `C:\ws\proj`, `c:\WS\PROJ\a.txt`, true, true},
		{"跨盘符", `C:\ws\proj`, `D:\ws\proj\a.txt`, false, true},
		{"反斜杠穿越", `C:\ws\proj`, `C:\ws\proj\..\..\x`, false, true},
	}
	for _, c := range cases {
		if c.winOnly && !windowsOnly() {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			if got := Within(c.root, c.target); got != c.want {
				t.Fatalf("Within(%q, %q) = %v, want %v", c.root, c.target, got, c.want)
			}
		})
	}
}

// TestResolveRelWithin 工作区相对路径解析矩阵：合法输入逐条给出期望绝对路径，
// 恶意输入逐条给出期望错因（errors.Is 分辨三类）。
func TestResolveRelWithin(t *testing.T) {
	root := t.TempDir()

	ok := []struct {
		name string
		rel  string
		want string // 相对于 root 的期望落点（"." = root 本身）
	}{
		{"空串=工作区根", "", "."},
		{"点=工作区根", ".", "."},
		{"普通文件名", "a.txt", "a.txt"},
		{"子目录", "sub/b.txt", filepath.Join("sub", "b.txt")},
		{"内部..归一", "sub/../b.txt", "b.txt"},
		{"嵌套清理（既有用例形态）", filepath.Join("sub", "..", "sub"), "sub"},
		{"尾部斜杠", "sub/", "sub"},
		{"点前缀", "./a.txt", "a.txt"},
	}
	for _, c := range ok {
		t.Run("合法/"+c.name, func(t *testing.T) {
			got, err := ResolveRelWithin(root, c.rel)
			if err != nil {
				t.Fatalf("ResolveRelWithin(%q) 不应报错：%v", c.rel, err)
			}
			want := root
			if c.want != "." {
				want = filepath.Join(root, c.want)
			}
			if got != want {
				t.Fatalf("ResolveRelWithin(%q) = %q, want %q", c.rel, got, want)
			}
			if !Within(root, got) {
				t.Fatalf("解析结果必须落在 root 内：%q", got)
			}
		})
	}

	bad := []struct {
		name    string
		rel     string
		wantErr error
		winOnly bool
	}{
		{"父目录", "..", ErrPathEscape, false},
		{"单级穿越", "../x.txt", ErrPathEscape, false},
		{"两级穿越", "../../x.txt", ErrPathEscape, false},
		{"深处穿越", "a/b/../../../x", ErrPathEscape, false},
		{"穿越到兄弟前缀目录", "../proj-other/x", ErrPathEscape, false},
		{"反斜杠穿越", `..\x.txt`, ErrPathEscape, true},
		{"混合分隔符穿越", `a/..\../x`, ErrPathEscape, true},
	}
	for _, c := range bad {
		if c.winOnly && !windowsOnly() {
			continue
		}
		t.Run("非法/"+c.name, func(t *testing.T) {
			got, err := ResolveRelWithin(root, c.rel)
			if err == nil {
				t.Fatalf("ResolveRelWithin(%q) 应拒绝，却返回 %q", c.rel, got)
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("ResolveRelWithin(%q) 错因 = %v, want errors.Is %v", c.rel, err, c.wantErr)
			}
			if got != "" {
				t.Fatalf("拒绝时必须返回空串，got %q", got)
			}
		})
	}

	abs := []struct {
		name    string
		rel     string
		wantErr error
		winOnly bool
	}{
		{"盘符绝对路径", `C:\x\y.txt`, ErrPathAbsolute, true},
		{"正斜杠盘符", `C:/x/y.txt`, ErrPathAbsolute, true},
		{"UNC", `\\srv\share\x`, ErrPathAbsolute, true},
		{"设备路径", `\\?\C:\x`, ErrPathAbsolute, true},
		{"卷相对", `C:foo`, ErrPathVolume, true},
		{"卷相对带穿越", `C:..\..\x`, ErrPathVolume, true},
	}
	for _, c := range abs {
		if c.winOnly && !windowsOnly() {
			continue
		}
		t.Run("非法/"+c.name, func(t *testing.T) {
			if _, err := ResolveRelWithin(root, c.rel); !errors.Is(err, c.wantErr) {
				t.Fatalf("ResolveRelWithin(%q) 错因 = %v, want errors.Is %v", c.rel, err, c.wantErr)
			}
		})
	}

	// 无卷根路径（Windows `\x`）：不是 IsAbs，被 Join 视作工作区相对 → 落在 root 内。
	if windowsOnly() {
		t.Run("无卷根路径落在root内", func(t *testing.T) {
			got, err := ResolveRelWithin(root, `\x.txt`)
			if err != nil {
				t.Fatalf("不应报错：%v", err)
			}
			if !Within(root, got) {
				t.Fatalf("必须落在 root 内：%q", got)
			}
		})
	}
}

// TestResolveRelWithinEmptyRoot fail-closed：根为空/不可解析一律拒绝（空 root
// 不得退化成「无约束」）。
func TestResolveRelWithinEmptyRoot(t *testing.T) {
	if _, err := ResolveRelWithin("", "a.txt"); err == nil {
		t.Fatal("空 root 必须拒绝")
	}
	if _, err := ResolveRelWithin("   ", "a.txt"); err == nil {
		t.Fatal("空白 root 必须拒绝")
	}
}

// TestResolveRelWithinCaseInsensitiveRoot Windows 上大小写不同的 root 仍判为同根
// （filepath.Rel 口径），否则「工作区内合法路径」会被误拒（咬人）。
func TestResolveRelWithinCaseInsensitiveRoot(t *testing.T) {
	if !windowsOnly() {
		t.Skip("仅 Windows")
	}
	root := t.TempDir()
	upper := strings.ToUpper(root)
	if upper == root {
		t.Skip("临时目录路径无字母")
	}
	got, err := ResolveRelWithin(upper, filepath.Join("sub", "a.txt"))
	if err != nil {
		t.Fatalf("大小写不同的 root 不应报错：%v", err)
	}
	if !Within(root, got) {
		t.Fatalf("解析结果必须落在原 root 内：%q", got)
	}
}
