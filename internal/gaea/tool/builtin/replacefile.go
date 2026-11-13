package builtin

// replace — 跨文件字面量批量替换（D4，dsh-ops ops_replace 蒸馏，2026-11-10）。
//
// 上游纪律（distill §8.3）：
//   - 全计划溢出安全预检：每文件替换后 >32MiB → 整次 PLAN_SIZE_LIMIT 拒绝，
//     **零变更**（dry_run 也计入）——先算术后动手；
//   - 逐文件原子发布**非事务**：A 成 B 败=保留 A 回执+B 失败+pending 清单
//     （不回滚已发布文件）；任一失败→结构化失败（effect=unknown：已发布
//     文件清单进 detail，模型可核实）；
//   - changed 计**替换次数**非文件数；每文件发布前在写锁内重读比对内容
//     哈希（计划哈希不证明现状）。
//
// gaea 方言如实缩小（记档）：pattern=字面量非正则；include=单串 glob（与
// grep 同方言，复用 grepIncludeMatch）；不带 caller revision（自扫描+发布前
// 哈希重核）；凭据围栏**写向**跳过（写凭据比读后果重）。

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

func init() { tool.RegisterBuiltin(replaceTool{}) }

type replaceTool struct {
	workDir string
	roots   []string
}

const (
	replaceDefaultMax  = 100      // max_replacements 缺省
	replaceMaxCap      = 100000   // max_replacements 上限
	replaceFileSizeCap = 32 << 20 // 每文件替换后 32MiB 上限（上游同值）
	replaceReceiptCap  = 30       // 回执逐文件清单上限（token 纪律）
)

func (replaceTool) Name() string { return "replace" }

func (replaceTool) Description() string {
	return "Replace a literal string across many files atomically per-file (NOT a transaction: already-published files stay published if a later file fails; the receipt lists published/failed/pending). Pre-checks the whole plan first: any file that would exceed 32MiB after replacement fails the entire call with zero changes. changed counts replacements, not files. Re-reads each file under a write lock and compares content hash before publishing (plan hash does not prove current state). Skips noise dirs, .gaea host subtrees, credential-shaped files and binary files (skips are reported). Use dry_run to preview the plan without writing."
}

func (replaceTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "pattern":{"type":"string","description":"Literal text to find (not a regex)"},
  "replacement":{"type":"string","description":"Replacement text (empty string deletes the pattern)"},
  "path":{"type":"string","description":"File or directory to search (default \".\" — the working directory)"},
  "include":{"type":"string","description":"Glob to filter files by name, e.g. \"*.go\""},
  "max_replacements":{"type":"integer","description":"Total replacement count cap (default 100, cap 100000); occurrences beyond it are listed as pending","minimum":1},
  "dry_run":{"type":"boolean","description":"Preview the plan without writing (the plan hash does not prove anything was written)"}
},
"required":["pattern","replacement"]
}`)
}

func (replaceTool) ReadOnly() bool { return false }

// WritePaths 声明写作用域=搜索根（CheckWrite 前缀包含判定；逐文件另有锁内
// 哈希重核）。
func (r replaceTool) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Path) == "" {
		p.Path = "."
	}
	return []string{resolveIn(r.workDir, p.Path)}, nil
}

func (replaceTool) CompactDescription() string     { return compactDesc["replace"] }
func (replaceTool) CompactSchema() json.RawMessage { return compactSchema["replace"] }
func (replaceTool) SpaceTag() string               { return spaces.SpaceWork }

func (r replaceTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Pattern         string `json:"pattern"`
		Replacement     string `json:"replacement"`
		Path            string `json:"path"`
		Include         string `json:"include"`
		MaxReplacements int    `json:"max_replacements"`
		DryRun          bool   `json:"dry_run"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", &tool.Failure{Code: tool.CodeValidationError, Err: fmt.Errorf("replace 参数解析失败: %w", err), EffectState: tool.EffectNone}
	}
	if p.Pattern == "" {
		return "", &tool.Failure{Code: tool.CodeValidationError, Err: fmt.Errorf("pattern is required and must be non-empty"), EffectState: tool.EffectNone}
	}
	max := p.MaxReplacements
	if max <= 0 {
		max = replaceDefaultMax
	}
	if max > replaceMaxCap {
		max = replaceMaxCap
	}
	root := resolveIn(r.workDir, p.Path)
	if root == "" {
		root = "."
	}
	if err := confine(r.roots, root); err != nil {
		return "", err
	}

	// ── 阶段一：扫描+全计划预检（零变更）──
	var (
		plan     []replaceFilePlan
		fenced   int
		binarySk int
		pendingF []string
		budget   = max
	)
	werr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // 不可读条目跳过（回执记 gap 属 grep 语义；写向保守跳过）
		}
		if d.IsDir() {
			if path != root && wspath.IsSkippedIn(d.Name(), wspath.GrepExtra) {
				return fs.SkipDir
			}
			return nil
		}
		if rel, rerr := filepath.Rel(root, path); rerr == nil && wspath.IsGaeaHostNoiseRel(filepath.ToSlash(rel)) {
			return nil
		}
		if _, hit := credentialFenceRule(path); hit {
			fenced++
			return nil
		}
		if p.Include != "" && !grepIncludeMatch(p.Include, root, path) {
			return nil
		}
		content, enc, err := readFileEncoded(path)
		if err != nil {
			return nil
		}
		if hasNULPeek(content) {
			binarySk++
			return nil
		}
		occ := strings.Count(content, p.Pattern)
		if occ == 0 {
			return nil
		}
		take := occ
		if take > budget {
			take = budget
		}
		if take == 0 {
			if rel, rerr := filepath.Rel(root, path); rerr == nil {
				pendingF = append(pendingF, filepath.ToSlash(rel))
			}
			return nil
		}
		newLen := len(content) - take*len(p.Pattern) + take*len(p.Replacement)
		if newLen > replaceFileSizeCap {
			// 全计划预检失败：整次零变更（上游纪律——先算术后动手）。
			return &tool.Failure{
				Code:        tool.CodeValidationError,
				Subcode:     "PLAN_SIZE_LIMIT",
				Err:         fmt.Errorf("replace 计划超限：%s 替换后将达 %d 字节（上限 %d）；缩小 pattern 范围或拆批", path, newLen, replaceFileSizeCap),
				EffectState: tool.EffectNone,
			}
		}
		rel, _ := filepath.Rel(root, path)
		plan = append(plan, replaceFilePlan{
			path: path, rel: filepath.ToSlash(rel), count: take, newLen: newLen,
			content: content, enc: enc, revision: RevisionOfBytes([]byte(content)),
		})
		budget -= take
		return nil
	})
	// WalkDir 回调返回 *tool.Failure 时 WalkDir 会把它包在 *PathError 里——
	// 拆包还原结构化失败。
	if werr != nil {
		if f, ok := tool.AsFailure(werr); ok {
			return "", f
		}
		if pe, ok := werr.(*fs.PathError); ok {
			if f, ok2 := tool.AsFailure(pe.Err); ok2 {
				return "", f
			}
		}
		return "", fmt.Errorf("replace %s: %v", root, werr)
	}

	changed := 0
	for _, it := range plan {
		changed += it.count
	}
	head := fmt.Sprintf("[replace] pattern=%q files=%d changed=%d（计替换次数）", p.Pattern, len(plan), changed)
	if fenced > 0 {
		head += fmt.Sprintf(" — credential-fence skipped %d file(s)", fenced)
	}
	if binarySk > 0 {
		head += fmt.Sprintf(" — binary skipped %d file(s)", binarySk)
	}
	if p.DryRun {
		return head + "\n" + replacePlanReceipt(plan, pendingF) +
			"\n(dry_run：未写盘；计划哈希不证明任何内容已写入——执行时逐文件重读比对)", nil
	}

	// ── 阶段二：逐文件原子发布（非事务）──
	var (
		published []string
		failed    []string
	)
	for _, it := range plan {
		pubErr := r.publishItem(ctx, it, p.Pattern, p.Replacement)
		if pubErr != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", it.rel, pubErr))
			continue
		}
		published = append(published, fmt.Sprintf("%s: %d 处", it.rel, it.count))
	}

	receipt := head + "\n" + replacePublishReceipt(published, failed, pendingF)
	if len(failed) > 0 {
		// 非事务：已发布文件保留（effect=unknown——模型须核实已发布部分）。
		return receipt, &tool.Failure{
			Code:        tool.CodeExecError,
			Subcode:     "PARTIAL_PUBLISH",
			Err:         fmt.Errorf("replace 部分失败：%d 文件已发布 / %d 失败 / %d pending（已发布不回滚）", len(published), len(failed), len(pendingF)),
			EffectState: tool.EffectUnknown,
			NextAction:  "按回执核实现状后重跑失败项（文件级原子，重跑安全）",
		}
	}
	return receipt, nil
}

// publishItem 阶段二单文件发布：写锁内 bind→重读→哈希重核→原子写。
// 抽为方法是为了对「计划哈希 vs 现状」竞态窗口做确定性单测（阶段一→阶段二
// 的毫秒级间隙在整调用里不可注入）。
func (replaceTool) publishItem(ctx context.Context, it replaceFilePlan, pattern, replacement string) error {
	return WithFileLock(it.path, func() error {
		bind, berr := bindFileLockTarget(it.path)
		if berr != nil {
			return berr
		}
		content, enc, rerr := readFileEncoded(it.path)
		if rerr != nil {
			return rerr
		}
		if RevisionOfBytes([]byte(content)) != it.revision {
			return &tool.Failure{
				Code:        tool.CodeConflict,
				Err:         fmt.Errorf("REVISION_CONFLICT: %s 自扫描后已被修改（计划哈希不匹配），本文件跳过", it.rel),
				Retryable:   true,
				EffectState: tool.EffectNone,
				NextAction:  "该文件已被外部修改：重跑 replace 或单独处理",
			}
		}
		if verr := bind.verify(it.path); verr != nil {
			return verr
		}
		newContent := strings.Replace(content, pattern, replacement, it.count)
		if werr := atomicWriteEncoded(it.path, newContent, enc); werr != nil {
			return werr
		}
		evidence.RecordChange(ctx, evidence.ChangeRecord{
			Tool:          "replace",
			Target:        it.path,
			BeforeSummary: truncateForPlan(pattern),
			AfterSummary:  truncateForPlan(replacement),
		})
		return nil
	})
}

// replaceFilePlan 一个文件的替换计划（阶段一产物；content+revision 供阶段二
// 锁内哈希重核，enc 保真回写）。
type replaceFilePlan struct {
	path     string
	rel      string
	count    int
	newLen   int
	content  string
	enc      fileenc.Kind
	revision string
}

func truncateForPlan(s string) string {
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}

func replacePlanReceipt(plan []replaceFilePlan, pending []string) string {
	var b strings.Builder
	for i, it := range plan {
		if i >= replaceReceiptCap {
			fmt.Fprintf(&b, "  … +%d more file(s)\n", len(plan)-replaceReceiptCap)
			break
		}
		fmt.Fprintf(&b, "  - %s: %d 处（plan %s…）\n", it.rel, it.count, it.revision[:12])
	}
	if len(pending) > 0 {
		fmt.Fprintf(&b, "pending（超出 max_replacements）：%d file(s)\n", len(pending))
	}
	return b.String()
}

func replacePublishReceipt(published, failed, pending []string) string {
	var b strings.Builder
	for i, l := range published {
		if i >= replaceReceiptCap {
			fmt.Fprintf(&b, "  … +%d more published\n", len(published)-replaceReceiptCap)
			break
		}
		b.WriteString("  ✓ " + l + "\n")
	}
	for _, l := range failed {
		b.WriteString("  ✗ " + l + "\n")
	}
	if len(pending) > 0 {
		fmt.Fprintf(&b, "pending（超出 max_replacements）：%d file(s)\n", len(pending))
	}
	return strings.TrimRight(b.String(), "\n")
}

func hasNULPeek(s string) bool {
	peek := s
	if len(peek) > 8192 {
		peek = peek[:8192]
	}
	return strings.ContainsRune(peek, 0)
}
