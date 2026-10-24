package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

// ErrAccessDenied is returned by AccessPolicy implementations when a path is
// outside the configured boundary. Callers may expose the wrapped message to
// the model, but must never attach file contents to a denial.
var ErrAccessDenied = errors.New("path access denied")

// AccessPolicy is the execution-time path boundary for tools that expose local
// files. Tool implementations declare how to resolve their model-controlled
// paths; the agent applies this policy before any precheck or execution.
type AccessPolicy interface {
	CheckRead(ctx context.Context, path string) error
	CheckPreview(ctx context.Context, path string) error
	CheckWrite(ctx context.Context, path string) error
}

// ReadTargeter exposes the single local read target of a tool.
type ReadTargeter interface {
	ReadTarget(args json.RawMessage) (string, bool)
}

// ReadPathResolver exposes every local read target of a tool. It is for tools
// such as grep/ls whose read scope can be a directory tree.
type ReadPathResolver interface {
	ReadPaths(ctx context.Context, args json.RawMessage) ([]string, error)
}

// WritePathResolver exposes every local path a tool may modify, including both
// endpoints of a move.
type WritePathResolver interface {
	WritePaths(args json.RawMessage) ([]string, error)
}

// PathAccessPolicy is the default filesystem policy. Read roots are workspace
// roots plus allow_write and allow_read; write roots are workspace plus
// allow_write only. An explicitly selected file may be read (and previewed) but
// never becomes writable through that selection.
type PathAccessPolicy struct {
	readRoots  []string
	writeRoots []string
	allowRead  func(string) bool
}

// NewPathAccessPolicy builds a canonical path policy. workspace and writeRoots
// are always readable; readRoots adds read-only boundaries. Roots that cannot
// be resolved are dropped, so a missing production configuration denies all
// path-tool calls instead of degrading to an unconfined mode.
func NewPathAccessPolicy(workspace string, readRoots, writeRoots []string, allowRead func(string) bool) *PathAccessPolicy {
	write := canonicalRoots(append([]string{workspace}, writeRoots...))
	read := canonicalRoots(append(append([]string{workspace}, writeRoots...), readRoots...))
	return &PathAccessPolicy{readRoots: read, writeRoots: write, allowRead: allowRead}
}

// NewDenyAccessPolicy returns a policy that denies every local path. Agent
// construction uses it when no production policy was supplied.
func NewDenyAccessPolicy() AccessPolicy { return denyAccessPolicy{} }

type denyAccessPolicy struct{}

func (denyAccessPolicy) CheckRead(context.Context, string) error    { return denied("read") }
func (denyAccessPolicy) CheckPreview(context.Context, string) error { return denied("preview") }
func (denyAccessPolicy) CheckWrite(context.Context, string) error   { return denied("write") }

func denied(kind string) error {
	return fmt.Errorf("%w: no %s policy is configured", ErrAccessDenied, kind)
}

func (p *PathAccessPolicy) CheckRead(_ context.Context, path string) error {
	return p.check(path, false)
}

func (p *PathAccessPolicy) CheckPreview(ctx context.Context, path string) error {
	return p.CheckRead(ctx, path)
}

func (p *PathAccessPolicy) CheckWrite(_ context.Context, path string) error {
	return p.check(path, true)
}

func (p *PathAccessPolicy) check(path string, write bool) error {
	if p == nil || strings.TrimSpace(path) == "" {
		return denied(map[bool]string{true: "write", false: "read"}[write])
	}
	canonical, err := wspath.RealPath(path)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve path %q: %v", ErrAccessDenied, path, err)
	}
	if !write && p.allowRead != nil {
		if p.allowRead(canonical) || p.allowRead(path) || p.allowRead(filepath.Clean(path)) {
			return nil
		}
	}
	roots := p.readRoots
	if write {
		roots = p.writeRoots
	}
	for _, root := range roots {
		if wspath.Within(root, canonical) {
			return nil
		}
	}
	if write {
		return fmt.Errorf("%w: %q is outside writable roots %s; write inside one of these or widen [sandbox] allow_write", ErrAccessDenied, path, renderRoots(roots))
	}
	// 诚实拒绝（Reasonix 2.x target_access 蒸馏，v4.524 刀 A7）：消息只依赖
	// 策略与路径——列出具体允许根让模型能自我纠正，但不指认文件是否存在
	// （解析失败与越界是两种错误，前者才提存在性）。
	return fmt.Errorf("%w: %q is outside readable roots %s; read inside one of these, widen [sandbox] allow_read, or select the file in the UI", ErrAccessDenied, path, renderRoots(roots))
}

// renderRoots 允许根清单的可读形态：最多列 8 个（配置序=确定性），超出诚实
// 省略并计数；空集合明说（不装作有根）。
func renderRoots(roots []string) string {
	if len(roots) == 0 {
		return "(none configured)"
	}
	const maxList = 8
	shown := roots
	if len(shown) > maxList {
		shown = shown[:maxList]
	}
	quoted := make([]string, 0, len(shown))
	for _, r := range shown {
		quoted = append(quoted, fmt.Sprintf("%q", r))
	}
	out := "(" + strings.Join(quoted, ", ")
	if len(roots) > maxList {
		out += fmt.Sprintf(", … +%d more", len(roots)-maxList)
	}
	return out + ")"
}

func canonicalRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	seen := map[string]bool{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		real, err := wspath.RealPath(root)
		if err != nil {
			continue
		}
		key := real
		if runtime.GOOS == "windows" {
			key = strings.ToLower(real)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, real)
	}
	return out
}
