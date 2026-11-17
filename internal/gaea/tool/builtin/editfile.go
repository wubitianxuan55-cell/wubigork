package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(editFile{}) }

// editFile replaces an exact old_string with new_string inside a file.
// roots, when non-empty, confines the target to the workspace (see confine);
// the zero value registered at init is unconfined and is overridden per run by
// ConfineWriters. workDir, when non-empty, is the directory a relative path
// resolves against (see resolveIn).
type editFile struct {
	roots   []string
	workDir string
}

func (editFile) Name() string { return "edit_file" }

func (editFile) Description() string {
	return "Replace an exact old_string with new_string in a file. new_string may be an empty string to delete. Fails when old_string is missing or matches more than once (pass replace_all:true for all occurrences). Encoding (GB18030/UTF-16) and file permissions are preserved."
}

func (editFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "old_string":{"type":"string","description":"Exact text to replace (must be non-empty and unique unless replace_all)"},
  "new_string":{"type":"string","description":"Replacement text (empty string deletes old_string)"},
  "replace_all":{"type":"boolean","description":"Replace every occurrence (default false)"},
  "expected_revision":{"type":"string","description":"Optional CAS guard: sha256 from the read_file trailer. Mismatch = REVISION_CONFLICT (file changed since you read it)."}
},
"required":["path","old_string","new_string"]
}`)
}

func (editFile) ReadOnly() bool { return false }

func (e editFile) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	return []string{resolveIn(e.workDir, p.Path)}, nil
}

func (editFile) CompactDescription() string     { return compactDesc["edit_file"] }
func (editFile) CompactSchema() json.RawMessage { return compactSchema["edit_file"] }

func (e editFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path             string `json:"path"`
		OldString        string `json:"old_string"`
		NewString        string `json:"new_string"`
		ReplaceAll       bool   `json:"replace_all"`
		ExpectedRevision string `json:"expected_revision,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("edit_file 参数解析失败: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if p.OldString == "" {
		return "", fmt.Errorf("old_string is required and must be non-empty (pass new_string:\"\" to delete text, or use write_file to create a file)")
	}
	p.Path = resolveIn(e.workDir, p.Path)
	if err := confine(e.roots, p.Path); err != nil {
		return "", err
	}
	// D4 写锁（dsh-ops bind-before-publish 蒸馏）：锁包住「读→CAS→写」整段——
	// CAS 防陈旧读（opt-in），锁防跨进程「CAS 通过→rename 前」TOCTOU；publish
	// 前 bind 重核 inode 身份。
	var (
		n        int
		baseline string
		seekNote string
	)
	if err := WithFileLock(p.Path, func() error {
		bind, berr := bindFileLockTarget(p.Path)
		if berr != nil {
			return fmt.Errorf("stat %s: %w", p.Path, berr)
		}
		content, enc, rerr := readFileEncoded(p.Path)
		if rerr != nil {
			return fmt.Errorf("read %s: %w", p.Path, rerr)
		}
		// A3 revision CAS（dsh-ops 蒸馏）：内容在手即刻比对（""=零行为变化）。
		if p.ExpectedRevision != "" {
			if err := checkExpectedRevision(p.Path, []byte(content), true, p.ExpectedRevision); err != nil {
				return err
			}
		}
		if verr := bind.verify(p.Path); verr != nil {
			return verr
		}
		// v4.1b：写盘前整文件基线快照（Verifier 视觉 diff / 回滚原料；未配置则 ""）。
		baseline = evidence.StageBaseline(ctx, p.Path, []byte(content))
		newContent, cnt, note, nerr := replaceInContent(content, p.OldString, p.NewString, p.ReplaceAll)
		if nerr != nil {
			return fmt.Errorf("edit %s: %w", p.Path, nerr)
		}
		n = cnt
		seekNote = note
		if werr := atomicWriteEncoded(p.Path, newContent, enc); werr != nil {
			return fmt.Errorf("write %s: %w", p.Path, werr)
		}
		return nil
	}); err != nil {
		return "", err
	}
	// v4.1 证据链：记录本次替换的原文/新文摘要（old_string/new_string 即变更
	// 区域的原文摘要，非展示截断文本）。ctx 无台账时静默跳过。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "edit_file",
		Target:        p.Path,
		BeforeSummary: p.OldString,
		AfterSummary:  p.NewString,
		BaselinePath:  baseline,
	})
	msg := fmt.Sprintf("edited %s: %d occurrence(s) replaced (+%d/-%d bytes)",
		p.Path, n, n*len(p.NewString), n*len(p.OldString))
	if seekNote != "" {
		// 三档兜底命中：告诉模型「没逐字节对上也找到了」，下次可继续给上下文块。
		msg += "\n" + seekNote
	}
	return msg, nil
}

// ── shared replacement / write kernel (edit_file + multi_edit + edit_lines) ──

// replaceInContent replaces old with new inside content and returns the new
// content, the number of replacements made, and a locator note (非空=走了三档
// 兜底定位，调用方原样追加进工具结果). With replaceAll=false it demands exactly
// one match: zero occurrences falls back to three-tier line matching (T1)，
// and more than one is an error telling the model to widen the context or opt
// into replace_all. edit_file and multi_edit both route through here so the
// two tools cannot drift apart on match semantics.
func replaceInContent(content, old, new string, replaceAll bool) (string, int, string, error) {
	n := strings.Count(content, old)
	if n > 1 && !replaceAll {
		return "", 0, "", fmt.Errorf(
			"old_string matches %d locations; include more surrounding context to make it unique, "+
				"or pass replace_all:true to replace all %d occurrences", n, n)
	}
	if n >= 1 {
		if replaceAll {
			return strings.ReplaceAll(content, old, new), n, "", nil
		}
		return strings.Replace(content, old, new, 1), 1, "", nil
	}
	// 精确零命中：CRLF 病因优先诊断（既有行为，三档兜底不抢——裸 \n 对 CRLF
	// 文件仍报错，指引模型抄对行尾或改用 edit_lines；静默改写 CRLF 文件会混行尾）。
	if strings.Contains(content, strings.ReplaceAll(old, "\n", "\r\n")) {
		return "", 0, "", fmt.Errorf(
			"old_string not found: the file uses CRLF (Windows) line endings — " +
				"your old_string has bare \\n. Re-read the file and copy the text exactly, " +
				"or edit line-wise with edit_lines")
	}
	// 三档行匹配兜底（codex 蒸馏 T1）：文件被无关改动后按上下文仍能定位。
	return seekReplace(content, old, new, replaceAll)
}

// seekReplace 三档行匹配兜底：把 old/new 拆成逻辑行，tool.FindSequence 逐档
// 定位后按行首偏移整段拼接替换。未触及区域保留原始字节（含各自行尾符），
// 替换块用文件的行尾风格（CRLF 文件探测到 \r\n 即整块用 CRLF）。
// 未命中时错误附「最接近的候选」行号（首行匹配度前 3）。
func seekReplace(content, old, new string, replaceAll bool) (string, int, string, error) {
	pat := splitLogicalLines(old)
	lines := splitLinesKeepEnds(content)
	logical := make([]string, len(lines))
	for i, l := range lines {
		logical[i] = strings.TrimRight(l, "\r\n")
	}
	// 反复定位收集全部非重叠命中（FindSequence 恒取最低档位，同输入同结果）。
	type hit struct {
		idx, tier int
	}
	var hits []hit
	for at := 0; ; {
		idx, tier := tool.FindSequence(logical, pat, at)
		if idx < 0 {
			break
		}
		hits = append(hits, hit{idx, tier})
		at = idx + len(pat)
	}
	if len(hits) == 0 {
		hint := ""
		if cands := tool.NearestMatches(logical, pat, 3); len(cands) > 0 {
			parts := make([]string, len(cands))
			for i, c := range cands {
				parts[i] = strconv.Itoa(c)
			}
			hint = " 三档匹配（忽略行尾/两侧空白）仍未命中；最接近的候选在第 " +
				strings.Join(parts, "、") + " 行"
		}
		return "", 0, "", fmt.Errorf(
			"old_string not found in file (0 occurrences). Re-read the file with read_file "+
				"and retry with the exact current content.%s", hint)
	}
	if len(hits) > 1 && !replaceAll {
		return "", 0, "", fmt.Errorf(
			"old_string matches %d locations under whitespace-tolerant matching; include more "+
				"surrounding context to make it unique, or pass replace_all:true to replace all %d occurrences",
			len(hits), len(hits))
	}
	// 行首偏移表：starts[i] 是第 i 行在 content 里的字节偏移。
	starts := make([]int, len(lines)+1)
	for i, l := range lines {
		starts[i+1] = starts[i] + len(l)
	}
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	block := strings.ReplaceAll(strings.ReplaceAll(new, "\r\n", "\n"), "\n", eol)
	out := content
	// 从后往前替换，前面的偏移不受影响。
	for i := len(hits) - 1; i >= 0; i-- {
		last := hits[i].idx + len(pat) - 1
		start, end := starts[hits[i].idx], starts[last]+len(logical[last])
		out = out[:start] + block + out[end:]
	}
	note := ""
	if tier := hits[0].tier; tier > 0 {
		name := "忽略行尾空白"
		if tier == 2 {
			name = "忽略两侧空白"
		}
		note = fmt.Sprintf("（按第 %d 档[%s]定位到第 %d 行）", tier, name, hits[0].idx+1)
	}
	return out, len(hits), note, nil
}

// splitLogicalLines 把模型给的块拆成逻辑行（剥 \r 行尾；尾随空行保留，
// 它承载「删到行尾」的语义）。
func splitLogicalLines(s string) []string {
	parts := strings.Split(s, "\n")
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

// splitLinesKeepEnds 按行拆分但保留每行行尾符（末行可能无行尾符）。
func splitLinesKeepEnds(s string) []string {
	var lines []string
	for len(s) > 0 {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			lines = append(lines, s)
			break
		}
		lines = append(lines, s[:i+1])
		s = s[i+1:]
	}
	return lines
}

// atomicWriteEncoded encodes content back to the file's original encoding and
// writes it atomically (temp file + rename), preserving the original file's
// permission bits. Missing files get UTF-8 / 0644. readFileEncoded/Encode stay
// paired here so a GB18030 or UTF-16 file is never silently re-coded.
func atomicWriteEncoded(path, content string, enc fileenc.Kind) error {
	perm := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	return fileutil.AtomicWrite(path, fileenc.Encode(content, enc), perm)
}
