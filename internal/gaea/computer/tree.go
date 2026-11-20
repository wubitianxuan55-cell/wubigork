package computer

// 无障碍树文本渲染（平台无关；施工单 W2）。
//
// 文法与 codex 逐字对齐（asset-62 §4，gaea 侧唯一定义处）：
//   首行： Window: "<窗口标题>", App: <displayName>.
//   节点行：<\t×深度><序号> <角色>[(状态)] <名字>[ Description: …][ Value: …][ ID: …][ Secondary Actions: …]
//   - 序号全树连续递增（深度先序），角色取 UIA LocalizedControlType（不硬编码英文表）；
//   - 长文本中段截断：头 75 + "... [truncated]" + 尾 73（合计 163 字符）；
//   - 整树超 maxTreeBytes 按行截断，追加 "... [accessibility tree truncated]"。
// UIA 遍历在 tree_windows.go 产出 treeNode；本文件负责渲染与截断几何。

import (
	"fmt"
	"strings"
)

// 状态/后缀渲染所需的常量（文法几何，golden 测试钉死）。
const (
	truncHead           = 75                // 长文本保留头长
	truncTail           = 73                // 长文本保留尾长
	truncMarker         = "... [truncated]" // 中段标记（15 字符）
	maxTreeBytes        = 32 * 1024         // 整树字节帽（超限按行截断）
	treeTruncatedMarker = "... [accessibility tree truncated]"
)

// treeNode 树节点中间结构：UIA 遍历层产出，渲染层消费（golden 夹具同构）。
type treeNode struct {
	Role        string      `json:"role"`                  // UIA LocalizedControlType
	Name        string      `json:"name,omitempty"`        // UIA Name
	States      []string    `json:"states,omitempty"`      // (collapsed) (settable, string) 等
	Description string      `json:"description,omitempty"` // UIA HelpText
	Value       string      `json:"value,omitempty"`       // UIA Value 值
	ID          string      `json:"id,omitempty"`          // UIA AutomationId
	Secondary   []string    `json:"secondary,omitempty"`   // Secondary Actions: 后缀
	Children    []*treeNode `json:"children,omitempty"`
}

// trunc163 长文本中段截断：>163 字符（rune）时 头75 + 标记15 + 尾73。
func trunc163(s string) string {
	r := []rune(s)
	if len(r) <= truncHead+len([]rune(truncMarker))+truncTail {
		return s
	}
	return string(r[:truncHead]) + truncMarker + string(r[len(r)-truncTail:])
}

// DisplayNameFromApp 从 process:<exe 路径> 提取展示名（exe 基名去扩展名，
// 对齐 codex list_windows 的 displayName）。无 process: 前缀时原样返回。
func DisplayNameFromApp(app string) string {
	p := strings.TrimPrefix(app, "process:")
	if p == "" {
		return app
	}
	p = strings.ReplaceAll(p, "\\", "/")
	base := p[strings.LastIndex(p, "/")+1:]
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	return base
}

// RenderTree 渲染整树文本（施工单 W2：唯一实现处，单函数产出）。
func RenderTree(w Window, root *treeNode) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Window: %q, App: %s.", w.Title, DisplayNameFromApp(w.App))
	if root == nil {
		return b.String()
	}
	idx := 0
	over := false
	var walk func(n *treeNode, depth int) bool
	walk = func(n *treeNode, depth int) bool {
		if n == nil {
			return true
		}
		var line strings.Builder
		fmt.Fprintf(&line, "%s%d %s", strings.Repeat("\t", depth), idx, trunc163(n.Role))
		idx++
		if len(n.States) > 0 {
			fmt.Fprintf(&line, " (%s)", trunc163(strings.Join(n.States, ", ")))
		}
		if n.Name != "" {
			line.WriteString(" ")
			line.WriteString(trunc163(n.Name))
		}
		if n.Description != "" {
			fmt.Fprintf(&line, " Description: %s", trunc163(n.Description))
		}
		if n.Value != "" {
			fmt.Fprintf(&line, " Value: %s", trunc163(n.Value))
		}
		if n.ID != "" {
			fmt.Fprintf(&line, " ID: %s", trunc163(n.ID))
		}
		if len(n.Secondary) > 0 {
			fmt.Fprintf(&line, " Secondary Actions: %s", trunc163(strings.Join(n.Secondary, ", ")))
		}
		// 字节帽：放不下这行就停（含换行），追加截断标记。
		if b.Len()+line.Len()+1 > maxTreeBytes {
			over = true
			return false
		}
		b.WriteByte('\n')
		b.WriteString(line.String())
		for _, c := range n.Children {
			if !walk(c, depth+1) {
				return false
			}
		}
		return true
	}
	walk(root, 0)
	if over {
		b.WriteString("\n" + treeTruncatedMarker)
	}
	return b.String()
}
