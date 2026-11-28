package agent

// stale_paths.go — 文件闸（同回合 stale 闸 / 跨轮版本观察 / 编辑预检查）用的路径
// 与锚点判定（2026-11-27 AI 反馈 #5 根修）。
//
// 此前三处都拿参数里的**原始路径串**：模型写相对路径（.gaea/work/a.py）时，闸用
// os.ReadFile 按进程 cwd（程序安装目录）去读，读不到就当成「锚点失配」——
// write_file 之后紧跟的 edit_file 必被拦（包4 会话日志逐调用回放：old_string 与
// 磁盘逐字一致，仍被拦三次，读一遍才放行）；版本观察也从未记上；编辑预检查
// 同理读不到文件而整体失效。单测全用绝对路径，一直没测出来。
//
// 工具自己会把相对路径解析到工作区（ReadTarget / WritePaths）。闸必须用同一个
// 解析结果——路径口径只有一处：问工具。

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// staleGuardPaths 闸与观察用的目标路径：读族走 ReadTarget，写族走 WritePaths
// （move_file 给出 [源, 目标]）；都不实现时退回参数里的原始路径串。
func staleGuardPaths(t tool.Tool, name, args string) []string {
	raw := json.RawMessage(args)
	if rt, ok := t.(tool.ReadTargeter); ok {
		if p, ok := rt.ReadTarget(raw); ok && p != "" {
			return []string{p}
		}
	}
	if wp, ok := t.(tool.WritePathResolver); ok {
		if ps, err := wp.WritePaths(raw); err == nil {
			out := make([]string, 0, len(ps))
			for _, p := range ps {
				if p != "" {
					out = append(out, p)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	if p := extractFilePath(name, args); p != "" {
		return []string{p}
	}
	return nil
}

// logicalLines 按行拆并剥掉行尾符（三档行匹配的输入口径）。
func logicalLines(s string) []string {
	parts := strings.Split(s, "\n")
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

// anchorPresent 报告 old 在 content 里是否还找得到：精确命中、行尾换算后精确命中
// （与 edit_file 的行尾无关匹配同口径，2026-11-28 黄甲反馈），或只是空白漂了
// （edit_file 工具自己的三档行匹配能命中）——判定口径与工具一致，闸不比工具更严。
func anchorPresent(content, old string) bool {
	if old == "" || content == "" {
		return false
	}
	if strings.Contains(content, old) {
		return true
	}
	if strings.Contains(content, "\r\n") && !strings.Contains(old, "\r\n") &&
		strings.Contains(content, strings.ReplaceAll(old, "\n", "\r\n")) {
		return true
	}
	if strings.Contains(old, "\r\n") && !strings.Contains(content, "\r\n") &&
		strings.Contains(content, strings.ReplaceAll(old, "\r\n", "\n")) {
		return true
	}
	pat := logicalLines(old)
	// 锚点首尾的空行只是「这里有个换行」，不是要匹配一个空行：去掉再比。
	for len(pat) > 1 && pat[len(pat)-1] == "" {
		pat = pat[:len(pat)-1]
	}
	for len(pat) > 1 && pat[0] == "" {
		pat = pat[1:]
	}
	idx, _ := tool.FindSequence(logicalLines(content), pat, 0)
	return idx >= 0
}

// editMissHint old_string 在文件里真找不到时的诊断：最接近的候选位置与那里的
// 现行文本——模型照着改锚点直接重试，不必为此单独 read_file 一轮。
func editMissHint(content, old string) string {
	var sb strings.Builder
	preview := old
	if r := []rune(preview); len(r) > 100 {
		preview = string(r[:100]) + "..."
	}
	fmt.Fprintf(&sb, "  searched for: %q\n", preview)
	lines, pat := logicalLines(content), logicalLines(old)
	near := tool.NearestMatches(lines, pat, 3)
	if len(near) == 0 {
		head := content
		if r := []rune(head); len(r) > 200 {
			head = string(r[:200]) + "..."
		}
		fmt.Fprintf(&sb, "  file content (first 200 chars): %q\n", head)
		sb.WriteString("  suggestion: use read_file to see the current content, then retry with the exact string.")
		return sb.String()
	}
	at := near[0] - 1
	from, to := at-1, at+len(pat)+1
	if from < 0 {
		from = 0
	}
	if to > len(lines) {
		to = len(lines)
	}
	fmt.Fprintf(&sb, "  最接近的位置：第 %d 行起（文件已被改过，锚点对不上）。那一带的现行文本：\n", near[0])
	for i := from; i < to; i++ {
		l := lines[i]
		if r := []rune(l); len(r) > 160 {
			l = string(r[:160]) + "…"
		}
		fmt.Fprintf(&sb, "    %d→%s\n", i+1, l)
	}
	if len(near) > 1 {
		others := make([]string, 0, len(near)-1)
		for _, n := range near[1:] {
			others = append(others, strconv.Itoa(n))
		}
		fmt.Fprintf(&sb, "  其他候选行：%s\n", strings.Join(others, ", "))
	}
	fmt.Fprintf(&sb, "  suggestion: 按上面的现行文本重写 old_string 后直接重试；要看更多上下文用 read_file（offset=%d limit=%d）。", from+1, to-from)
	return sb.String()
}
