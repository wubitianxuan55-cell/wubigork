package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

// ── A10 子代理写范围声明（Reasonix 2.x writeclaim/fleet 预检蒸馏，v4.524）──
//
// task 调用可用 write_paths 声明本子代理要改哪些文件/目录；声明使两件事成立：
//  1. 冲突键（batch_executor.getConflictKey）：声明集相同的调用同批必串行；
//  2. 本注册表：执行期与**正在运行**的其它声明做两两重叠检查，目录重叠其下
//     所有路径，重叠的后到者当场拒绝——上游 fleet「目录重叠串行化」的 gaea
//     形态取 fail-fast（无审批 UX，等排队位次不如让模型显式重派）。
//
// 诚实边界：声明是并行协调元数据，不是安全边界——越出声明范围的写不在此拦
// （上游有 extend_write_paths 能力卡把越界路由给人，gaea 无审批面）；文件系
// 统安全的执行期边界仍是 AccessPolicy。未声明调用的行为零变化。

// normalizeWritePaths 校验并归一 write_paths：glob（*?[]）与路径逃逸（.. 越
// 出根）拒绝（上游 task_schema 同口径）；相对路径对进程 cwd 归一（同一运行
// 内所有 task 调用共享同一基准，重叠比较才有意义）。
func normalizeWritePaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	cwd, _ := os.Getwd()
	out := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			return nil, fmt.Errorf("write_paths contains an empty entry")
		}
		if strings.ContainsAny(p, "*?[]") {
			return nil, fmt.Errorf("write_paths entry %q is a glob: declare literal file or directory paths", p)
		}
		// Windows 宿主的 POSIX 绝对形态先归位：MSYS 盘符挂载（/c/x → C:\x）
		// 翻译后走常规检查；其余（/、/home/…）join 会退化成本 cwd，直接拒。
		if runtime.GOOS == "windows" && strings.HasPrefix(p, "/") && !filepath.IsAbs(p) {
			if len(p) >= 3 && p[2] == '/' && isASCIILetter(p[1]) {
				p = strings.ToUpper(p[1:2]) + ":" + filepath.FromSlash(p[2:])
			} else {
				return nil, fmt.Errorf("write_paths entry %q is not a usable Windows path", raw)
			}
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		p = filepath.Clean(p)
		// 逃逸=归一后不在 cwd 子树内（../outside、a/../../x 都拦；..foo 是
		// 合法名不受影响）。cwd 即本运行所有 task 调用共享的基准。
		if cwd != "" && !wspath.Within(cwd, p) {
			return nil, fmt.Errorf("write_paths entry %q escapes the workspace", raw)
		}
		// 卷根/家目录同样是逃逸（写入根集合的钳制交给 AccessPolicy，这里只
		// 拒绝「声明了无意义的大范围」）。
		if runtime.GOOS == "windows" {
			if len(p) >= 2 && p[1] == ':' && len(p) <= 3 {
				return nil, fmt.Errorf("write_paths entry %q is a drive root", raw)
			}
		}
		key := p
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// pathsOverlap 两声明集是否重叠：任一路径对相等，或一方是另一方目录前缀
// （目录重叠其下所有路径——上游 fleet 口径）。比较在 Windows 下大小写不敏感，
// 分隔符统一 ToSlash（声明可能来自 POSIX 形态的模型输出）。
func pathsOverlap(a, b []string) bool {
	norm := func(p string) string {
		p = filepath.ToSlash(filepath.Clean(p))
		if runtime.GOOS == "windows" {
			return strings.ToLower(p)
		}
		return p
	}
	for _, x := range a {
		for _, y := range b {
			xn, yn := norm(x), norm(y)
			if xn == yn || strings.HasPrefix(yn, xn+"/") || strings.HasPrefix(xn, yn+"/") {
				return true
			}
		}
	}
	return false
}

// isASCIILetter ASCII 字母判定（MSYS 盘符形态识别用）。
func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// acquireWriteClaim 登记一个活动声明；与任何活动声明重叠即拒绝（fail-fast）。
// id 用子代理会话引用（run ref 或 ephemeral id）。
func (t *TaskTool) acquireWriteClaim(id string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	t.claimsMu.Lock()
	defer t.claimsMu.Unlock()
	if t.writeClaims == nil {
		t.writeClaims = map[string][]string{}
	}
	for otherID, other := range t.writeClaims {
		if otherID == id || !pathsOverlap(paths, other) {
			continue
		}
		return fmt.Errorf("write_paths overlap with a concurrently running task (%s): %v vs %v; "+
			"dispatch sequentially, or give the tasks disjoint write_paths so they can run in parallel",
			otherID, paths, other)
	}
	t.writeClaims[id] = paths
	return nil
}

// releaseWriteClaim 释放声明（执行结束/取消都要走——defer 配对）。
func (t *TaskTool) releaseWriteClaim(id string) {
	t.claimsMu.Lock()
	defer t.claimsMu.Unlock()
	delete(t.writeClaims, id)
}

// declaredWritePaths 从 task 调用参数里取声明的写范围（冲突键用；解析失败
// 视为未声明——执行期的 normalizeWritePaths 会如实报错，键层不重复报）。
func declaredWritePaths(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}
	var p struct {
		WritePaths []string `json:"write_paths"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return nil
	}
	return p.WritePaths
}
