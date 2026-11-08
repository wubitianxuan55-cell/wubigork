package tool

// 节点写作用域策略（2026-11-08 众神殿办公板块子代理裁决刀4：「逐调用路径
// 包含校验」前置件——并行写放开的裁决条件之一）。在父策略（工作区/allow_write
// 边界）之上把写收窄到本节点声明路径；读/预览透传父策略不收窄（节点要读
// 上游产物与材料，读面=父边界）。声明根由调用方解析成绝对路径后传入；逐
// 调用比对经 RealPath 规范化（符号链接/8.3 短名/junction 归一，Windows
// 大小写不敏感）——与 PathAccessPolicy 同一基线，声明面与执行面共用一套
// 归一（裁决风险条：任务声明与工具 workDir 相对基准必须统一）。

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

type WriteScopePolicy struct {
	parent AccessPolicy
	roots  []string
}

// NewWriteScopePolicy 包裹父策略并收窄写面到声明根（绝对路径；解析失败的
// 根丢弃——与 NewPathAccessPolicy 同纪律：宁缺勿放宽）。parent 为 nil 时写
// 全拒（无边界可继承=无授权）。
func NewWriteScopePolicy(parent AccessPolicy, declaredRoots []string) *WriteScopePolicy {
	roots := make([]string, 0, len(declaredRoots))
	seen := make(map[string]bool, len(declaredRoots))
	for _, r := range declaredRoots {
		if strings.TrimSpace(r) == "" {
			continue
		}
		real, err := wspath.RealPath(r)
		if err != nil {
			continue
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		roots = append(roots, real)
	}
	return &WriteScopePolicy{parent: parent, roots: roots}
}

func (p *WriteScopePolicy) CheckRead(ctx context.Context, path string) error {
	return p.parent.CheckRead(ctx, path)
}

func (p *WriteScopePolicy) CheckPreview(ctx context.Context, path string) error {
	return p.parent.CheckPreview(ctx, path)
}

// CheckWrite 先过父策略（工作区边界是节点作用域的底座），再验声明根包含。
func (p *WriteScopePolicy) CheckWrite(ctx context.Context, path string) error {
	if p.parent != nil {
		if err := p.parent.CheckWrite(ctx, path); err != nil {
			return err
		}
	} else {
		return denied("write")
	}
	canonical, err := wspath.RealPath(path)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve path %q: %v", ErrAccessDenied, path, err)
	}
	for _, root := range p.roots {
		if wspath.Within(root, canonical) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q is outside this node's declared write paths %s; write only what the node declares (adjust write_paths in the plan if the scope is wrong)", ErrAccessDenied, path, renderRoots(p.roots))
}
