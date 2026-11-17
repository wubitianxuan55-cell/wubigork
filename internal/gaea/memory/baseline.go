package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// 记忆基线差分（codex 蒸馏 T12 护栏三，baseline diff 同构）：整理/批量写入
// 前先快照「名 → 正文哈希」，落成后与基线比出 added/modified/removed 三类。
// 删除超 10% 或修改超 30% 视为不可信整理（模型大规模重写记忆是事故级信号），
// 结果不应用并标「待确认」。gaea 用内存快照即可，不起 git（上游用 GitBaselineDiff
// 是因为它的记忆在文件树里）。

// BaselineDiff 是一次整理相对基线的差分。
type BaselineDiff struct {
	Added    []string // 新增的记忆名（稳定排序）
	Modified []string // 正文变化的记忆名
	Removed  []string // 消失的记忆名
}

// SnapshotBaseline 对记忆列表做快照：名 → Body 的 sha256（无效条目按名跳过）。
func SnapshotBaseline(ms []Memory) map[string]string {
	out := make(map[string]string, len(ms))
	for _, m := range ms {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		sum := sha256.Sum256([]byte(m.Body))
		out[name] = hex.EncodeToString(sum[:])
	}
	return out
}

// DiffBaseline 比较整理前后的两个快照，三类差分各自稳定排序（同名同输入同结果）。
func DiffBaseline(before, after map[string]string) BaselineDiff {
	var d BaselineDiff
	for name := range after {
		h, ok := before[name]
		if !ok {
			d.Added = append(d.Added, name)
		} else if h != after[name] {
			d.Modified = append(d.Modified, name)
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok {
			d.Removed = append(d.Removed, name)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Modified)
	sort.Strings(d.Removed)
	return d
}

// baselineMaxRemoved / baselineMaxModified 是阈值保护线（上游同值）：
// 删除超过基线的 10%、修改超过 30%，即视为不可信整理。
const (
	baselineMaxRemoved  = 0.10
	baselineMaxModified = 0.30
)

// DiffAllowed 报告差分是否在阈值内（可应用）。基线为空时任何删除都超线
// （0 条基线被删到 0 = 分母 0 按超线处理，防「清库」形态）。
func DiffAllowed(d BaselineDiff, baselineSize int) bool {
	if baselineSize <= 0 {
		return len(d.Removed) == 0 && len(d.Modified) == 0
	}
	if float64(len(d.Removed))/float64(baselineSize) > baselineMaxRemoved {
		return false
	}
	if float64(len(d.Modified))/float64(baselineSize) > baselineMaxModified {
		return false
	}
	return true
}

// baselineRenderMaxBytes 是差分渲染的字节上限（有界渲染，超限按字符边界截断）。
const baselineRenderMaxBytes = 4096

// RenderBaselineDiff 把差分渲染成可审文本（有界：超上限按 UTF-8 字符边界截断，
// 中文不出半个字）。
func RenderBaselineDiff(d BaselineDiff, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = baselineRenderMaxBytes
	}
	var b strings.Builder
	fmt.Fprintf(&b, "整理差分：新增 %d / 修改 %d / 删除 %d\n", len(d.Added), len(d.Modified), len(d.Removed))
	for _, name := range d.Added {
		fmt.Fprintf(&b, "+ %s\n", name)
	}
	for _, name := range d.Modified {
		fmt.Fprintf(&b, "~ %s\n", name)
	}
	for _, name := range d.Removed {
		fmt.Fprintf(&b, "- %s\n", name)
	}
	out := b.String()
	if len(out) <= maxBytes {
		return out
	}
	keep := maxBytes - len("…\n（差分过长，已截断）")
	if keep <= 0 {
		return "差分过长，无法渲染\n"
	}
	// 回退到 UTF-8 字符边界（不留半个 rune）。
	for keep > 0 && out[keep]&0xC0 == 0x80 {
		keep--
	}
	return out[:keep] + "…\n（差分过长，已截断）"
}
