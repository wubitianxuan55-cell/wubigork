package dag

// 计划期写路径闸（2026-11-08 众神殿办公板块子代理裁决刀2，pan_1794121254456/
// pan_1794117515203）：节点用 WritePaths 声明各自的写作用域（协调元数据，非
// 安全边界）。裁决条件逐字落约——
//   - 同波已声明写路径必须两两不重叠，重叠在计划期拒绝（新建与改图两分支）；
//   - 缺失或无法判断的写路径默认串行（执行期该波降宽 1）。
// 重叠判定口径：相等、目录前缀（目录含其下全部）、Windows 大小写不敏感；
// 相对路径基准=工作区根（统一 ToSlash 归一后比较）。glob 与 ".." 逃逸段属
// 「无法判断」：不参与重叠比对，但令该波降宽串行。

import (
	"fmt"
	"path/filepath"
	"strings"
)

// IsLiteralWritePath 报告 p 是否为可判定的字面写路径（无 glob 元字符、无
// ".." 逃逸段、非空）。glob 与逃逸属「无法判断」——不参与重叠比对。
func IsLiteralWritePath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	if strings.ContainsAny(p, "*?[]") {
		return false
	}
	slash := filepath.ToSlash(p)
	for _, seg := range strings.Split(slash, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// normalizeWritePath 写路径归一（仅用于比较，不改写声明原文）：ToSlash、去
// 首部 "./"、去尾 "/"、Windows 大小写不敏感统一小写。盘符/绝对路径不特殊
// 处理——声明与比对同走本归一，两边一致即判定一致。
func normalizeWritePath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(p, "/")
	return strings.ToLower(p)
}

// WritePathsOverlap 报告两条已归一（或未归一均可）的字面写路径是否重叠：
// 相等、任一为另一的目录前缀（"/" 边界对齐，a 与 a.txt.bak 不重叠）。
func WritePathsOverlap(a, b string) bool {
	a, b = normalizeWritePath(a), normalizeWritePath(b)
	if a == b {
		return true
	}
	return strings.HasPrefix(b, a+"/") || strings.HasPrefix(a, b+"/")
}

// waveWriteConflicts 对一个波内的已声明字面写路径做两两重叠检查，返回人读
// 冲突描述（空=无冲突）。无法判定的条目（glob/逃逸）跳过比对——它们的代价
// 由 WaveFullyDeclared 的降宽承担，不在这里拒绝。
func waveWriteConflicts(wave []Node) []string {
	type decl struct{ node, path string }
	var all []decl
	for _, n := range wave {
		for _, p := range n.WritePaths {
			if IsLiteralWritePath(p) {
				all = append(all, decl{node: n.ID, path: p})
			}
		}
	}
	var conflicts []string
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if WritePathsOverlap(all[i].path, all[j].path) {
				conflicts = append(conflicts, fmt.Sprintf(
					"节点 %s 与 %s 同波写路径重叠（%s ↔ %s）",
					all[i].node, all[j].node, all[i].path, all[j].path))
			}
		}
	}
	return conflicts
}

// WaveWriteConflicts 全图分波后逐波检查已声明写路径的两两重叠（依赖分波使
// 上下游节点天然异波，不受本闸约束）。返回人读冲突列表（空=无冲突）；调用
// 方 fail-closed 拒绝规划。Order 失败（成环等）原样上抛。
func WaveWriteConflicts(nodes []Node) ([]string, error) {
	waves, err := Order(nodes, nil)
	if err != nil {
		return nil, err
	}
	var conflicts []string
	for _, wave := range waves {
		conflicts = append(conflicts, waveWriteConflicts(wave)...)
	}
	return conflicts, nil
}

// WaveFullyDeclared 报告波内每个节点都声明了至少一条可判定的字面写路径。
// 否则该波执行期降宽 1（裁决：缺失或无法判断的写路径默认串行）。
func WaveFullyDeclared(wave []Node) bool {
	for _, n := range wave {
		ok := false
		for _, p := range n.WritePaths {
			if IsLiteralWritePath(p) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return len(wave) > 0
}
