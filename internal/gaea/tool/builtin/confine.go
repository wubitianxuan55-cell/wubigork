package builtin

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// ConfineBash returns the bash built-in bound to an OS-sandbox spec, overriding
// the unconfined instance registered at init. When the spec enforces, bash runs
// each command through the sandbox (see package sandbox). The instance carries
// a session state anchor (shellstate.go) so cwd/env persist across calls.
func ConfineBash(spec sandbox.Spec) tool.Tool {
	return bash{sb: spec, shell: sandbox.ResolveShell(), st: newShellState()}
}

// ConfineWriters returns the file-writing built-ins (write_file, edit_file,
// multi_edit, edit_lines, move_file) bound to roots — the only directories
// they may modify. The composition root adds these to the per-run registry to
// override the unconfined instances registered at init time, so writes stay
// inside the workspace by default. roots may be relative; they are resolved to
// absolute, symlink-free paths once here. An empty roots slice yields
// unconfined writers.
func ConfineWriters(roots []string) []tool.Tool {
	rs := realRoots(roots)
	return []tool.Tool{
		writeFile{roots: rs},
		editFile{roots: rs},
		multiEdit{roots: rs},
		editLines{roots: rs},
		moveFile{roots: rs},
	}
}

// realRoots resolves each root to an absolute, symlink-free path, dropping any
// that cannot be made absolute. Resolving here (once) means the per-call check
// only has to resolve the target.
func realRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		if real, err := realPath(r); err == nil {
			out = append(out, real)
		}
	}
	return out
}

// confine reports an error when target resolves outside every root. An empty
// roots slice is unconfined (returns nil) — the safe default for the built-in
// templates before a run configures the workspace. The error text is written
// for the model: it names the boundary and how the user can widen it.
//
// 双侧归一：root 也过 RealPath。构造方可能传 8.3 短名/junction 形态的根
// （GH windows runner 的 TEMP=RUNNER~1 实锤：root 短名 vs target 经 RealPath
// 解析成长名，Within 误判 outside）；解析失败退回原形——不确定即拒
// （fail-closed），绝不因解析不了而放行。
func confine(roots []string, target string) error {
	if len(roots) == 0 {
		return nil
	}
	abs, err := realPath(target)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", target, err)
	}
	for _, r := range roots {
		real, err := realPath(r)
		if err != nil {
			real = filepath.Clean(r)
			if !filepath.IsAbs(real) {
				return fmt.Errorf("resolve root %s: %w", r, err)
			}
		}
		if within(real, abs) {
			return nil
		}
	}
	return fmt.Errorf("path %q is outside the workspace (writes are confined to %s); "+
		"write inside it, or widen [sandbox] workspace_root / allow_write in gaea.toml",
		target, strings.Join(roots, ", "))
}

// realPath resolves path to an absolute, symlink-free form. Because a write
// target need not exist yet (write_file creates it), it resolves the deepest
// existing ancestor with EvalSymlinks and re-appends the not-yet-existing tail.
// This stops a symlinked directory from smuggling a write outside a root.
func realPath(path string) (string, error) {
	return wspath.RealPath(path)
}

// within reports whether path is at or below root. Both must be absolute,
// cleaned, symlink-free. It is a thin alias of the single-source judgment
// wspath.Within (审计 2026-10-02 AP5-09): filepath.Rel semantics, correct
// across volumes, never fooled by a prefix that only matches a partial path
// component (e.g. /work-other is not within /work), and case-insensitive on
// Windows like the filesystem itself.
func within(root, path string) bool { return wspath.Within(root, path) }

// resolveIn resolves a path relative to workDir if the path is relative.
// workDir defaults to cwd when empty.
func resolveIn(workDir, p string) string {
	if workDir == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(workDir, p)
}
