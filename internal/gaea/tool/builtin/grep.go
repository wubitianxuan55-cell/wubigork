package builtin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

func init() { tool.RegisterBuiltin(grepTool{}) }

// grepTool searches file contents for a regular expression and prints each
// match as `path:line: content` — the exact format the agent layer's
// compressGrep (agent/compress.go) parses, so results compress losslessly on
// their way back to the model. It is read-only: no confinement is needed
// (matching read_file's posture), but noise directories and binary files are
// skipped so a workspace sweep stays useful.
type grepTool struct{ workDir string }

const (
	grepDefaultMaxResults = 200 // matches returned when max_results is unset
	grepMaxResultsCap     = 1000
	grepBinaryPeek        = 8 * 1024 // bytes scanned for a NUL to flag binary
)

func (grepTool) Name() string { return "grep" }

func (grepTool) Description() string {
	return "Search file contents with a regular expression. Returns matches as `path:line: content` lines. Skips binary files, noise directories (node_modules, .git, dist, build, ...) and credential-shaped files (credential fence). Use `include` (e.g. \"*.go\") to filter by file name glob. Use `offset` to page through large match sets (the head line reports the next offset)."
}

func (grepTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "pattern":{"type":"string","description":"Regular expression to search for"},
  "path":{"type":"string","description":"File or directory to search (default \".\" — the working directory)"},
  "include":{"type":"string","description":"Glob to filter files by name, e.g. \"*.go\""},
  "max_results":{"type":"integer","description":"Maximum number of matching lines to return (default 200, cap 1000)","minimum":1},
  "offset":{"type":"integer","description":"Skip the first N matches (continuation cursor for paging through large result sets)","minimum":0}
},
"required":["pattern"]
}`)
}

func (grepTool) ReadOnly() bool { return true }

func (g grepTool) ReadPaths(_ context.Context, args json.RawMessage) ([]string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Path) == "" {
		p.Path = "."
	}
	return []string{resolveIn(g.workDir, p.Path)}, nil
}

func (grepTool) CompactDescription() string     { return compactDesc["grep"] }
func (grepTool) CompactSchema() json.RawMessage { return compactSchema["grep"] }

// 噪声目录口径见 internal/gaea/wspath（唯一真相源）：本工具用 GrepExtra 增量，
// 与 agent/compress.go 的树压缩器逐项一致（收敛前两份 15 项清单逐字相同）。
// 此处不再自建清单——builtin 不能 import agent，但两者都可以 import 叶子包 wspath。

func (g grepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Include    string `json:"include"`
		MaxResults int    `json:"max_results"`
		Offset     int    `json:"offset"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	re, err := regexp.Compile(p.Pattern)
	if err != nil {
		// Validation-style error: the pattern never became a regex, so no
		// execution happened — tell the model exactly what was wrong.
		return "", fmt.Errorf("invalid pattern %q: %v", p.Pattern, err)
	}
	maxResults := p.MaxResults
	if maxResults <= 0 {
		maxResults = grepDefaultMaxResults
	}
	if maxResults > grepMaxResultsCap {
		maxResults = grepMaxResultsCap
	}
	root := resolveIn(g.workDir, p.Path)
	if root == "" {
		root = "." // path omitted entirely → cwd
	}

	var out strings.Builder
	total := 0       // 本页已发射命中数
	skip := p.Offset // A1' 续页游标（dsh-ops 蒸馏）：跳过前 N 条命中
	if skip < 0 {
		skip = 0
	}
	fenced := 0 // B1 凭据围栏跳过数
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir // unreadable dir — skip silently, keep walking
			}
			return nil // unreadable file — skip
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if path != root && wspath.IsSkippedIn(d.Name(), wspath.GrepExtra) {
				return fs.SkipDir
			}
			return nil
		}
		// B1 凭据围栏（dsh-ops 蒸馏）：凭据形态文件跳过不读，头部如实记账。
		if _, hit := credentialFenceRule(path); hit {
			fenced++
			return nil
		}
		if p.Include != "" && !grepIncludeMatch(p.Include, root, path) {
			return nil
		}
		// A1'：skip 在文件中间耗尽时同文件剩余命中接着发射（单遍 skip+emit，
		// 页边界不错切）。
		n, err := grepFile(ctx, re, path, &skip, maxResults-total, &out)
		if err != nil {
			return nil // skip unreadable/undecodable file, keep searching
		}
		total += n
		if total >= maxResults {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && err != fs.SkipAll && ctx.Err() == nil {
		return "", fmt.Errorf("grep %s: %v", root, err)
	}
	if total == 0 && fenced == 0 {
		return fmt.Sprintf("(no matches for %q under %s)", p.Pattern, filepath.ToSlash(root)), nil
	}
	// 头部行（截断生存性结论同 spillNotice：头部行在两种截断形态下必活）——
	// 游标续页指引与围栏记账共用一行。matchedTotal 语义如实：扫描在预算帽
	// 提前止步时 total 是「至少」，不给伪造总数。
	if total >= maxResults || p.Offset > 0 || fenced > 0 {
		var head strings.Builder
		fmt.Fprintf(&head, "[grep page] offset=%d shown=%d", p.Offset, total)
		if total >= maxResults {
			fmt.Fprintf(&head, " — more matches may remain; re-run with offset=%d", p.Offset+total)
		}
		if fenced > 0 {
			fmt.Fprintf(&head, " — credential-fence skipped %d file(s)", fenced)
		}
		head.WriteString("\n")
		out.WriteString(head.String())
	}
	if total == 0 {
		return fmt.Sprintf("[grep page] offset=%d shown=0%s\n(no matches for %q under %s)",
			p.Offset, fenceNote(fenced), p.Pattern, filepath.ToSlash(root)), nil
	}
	if total >= maxResults {
		fmt.Fprintf(&out, "[truncated at %d results — refine the pattern, narrow the path, raise max_results, or re-run with offset=%d]", maxResults, p.Offset+total)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// fenceNote 围栏记账尾注（零跳过静默）。
func fenceNote(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf(" — credential-fence skipped %d file(s)", n)
}

// grepFile scans one file and appends its matches as `path:line: content`.
// Returns the number of matches emitted. Binary files (NUL byte in the first
// peek window) are skipped. skip 为页游标（A1'）：跳过前 N 条命中后才计入
// budget 并发射——skip 在文件中间耗尽时该文件剩余命中接着发射，页边界不错切；
// skip 指针跨文件累计递减。
func grepFile(ctx context.Context, re *regexp.Regexp, path string, skip *int, budget int, out *strings.Builder) (int, error) {
	if budget <= 0 {
		return 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	peek := data
	if len(peek) > grepBinaryPeek {
		peek = peek[:grepBinaryPeek]
	}
	if bytes.IndexByte(peek, 0) >= 0 {
		return 0, nil // binary — never a text hit
	}
	enc, _ := fileenc.Detect(data)
	content := string(fileenc.Decode(data, enc))

	slashed := filepath.ToSlash(path)
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if re.MatchString(line) {
			if skip != nil && *skip > 0 {
				*skip--
				continue
			}
			n++
			fmt.Fprintf(out, "%s:%d: %s\n", slashed, lineNo, line)
			if n >= budget {
				break
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	return n, nil
}

// grepIncludeMatch reports whether path passes the include glob. A glob
// without a separator matches the file's base name ("*.go"); one with a
// separator matches the slash path relative to the search root ("cmd/**").
func grepIncludeMatch(include, root, path string) bool {
	if !strings.ContainsAny(include, "/\\") {
		ok, err := filepath.Match(include, filepath.Base(path))
		return err == nil && ok
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	ok, err := filepath.Match(filepath.ToSlash(include), filepath.ToSlash(rel))
	return err == nil && ok
}
