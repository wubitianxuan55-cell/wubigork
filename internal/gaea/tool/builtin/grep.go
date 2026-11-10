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
	return "Search file contents with a regular expression. Returns matches as `path:line: content` lines. Skips binary files, noise directories (node_modules, .git, dist, build, ...), .gaea host-runtime subtrees and credential-shaped files (credential fence). The head line reports scan completeness (scan=complete|partial — partial with unreadable-file gaps means coverage is a lower bound) and the next offset when more matches may remain. Use `include` (e.g. \"*.go\") to filter by file name glob."
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

	// 刀0（dsh-ops 第二波 2026-11-10）：命中行攒进 body，头行组装后前置——
	// 旧实现头行写在匹配行之后（与「头部行截断生存性」注释相悖），且
	// compressGrep 只保留可解析行，头/尾控制行必须靠后置回放前置才活得下来。
	var body strings.Builder
	total := 0       // 本页已发射命中数
	skip := p.Offset // A1' 续页游标（dsh-ops 蒸馏）：跳过前 N 条命中
	if skip < 0 {
		skip = 0
	}
	pageOffset := skip // 本页起始 offset（回显与续页指引用；skip 游标在遍历中递减）
	fenced := 0        // B1 凭据围栏跳过数（excluded：选择不看）
	gaps := 0          // D5 跳过缺口（没看到：不可读文件/目录、超长行截断）——
	// 「没看到」与「选择不看」（噪声/围栏/include/二进制）分开记账：缺口在
	// 则本页覆盖是下界，续读指引不发（dsh-ops 蒸馏 D5，2026-11-10）。
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			gaps++ // unreadable dir/file — 如实记缺口，不再静默
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
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
		// 刀0：.gaea 宿主自管子树（sessions/journal/quarantine 等运行时产物）
		// 不是工作区正文——按相对路径跳（名称型跳过表达不了「.gaea 下部分
		// 子树」，exports/play 保留）。
		if rel, rerr := filepath.Rel(root, path); rerr == nil && wspath.IsGaeaHostNoiseRel(filepath.ToSlash(rel)) {
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
		n, err := grepFile(ctx, re, path, &skip, maxResults-total, &body)
		if err != nil {
			gaps++ // 不可读/超长行截断——记缺口继续搜
			return nil
		}
		total += n
		if total >= maxResults {
			return fs.SkipAll
		}
		return nil
	})
	cancelled := ctx.Err() != nil
	if err != nil && err != fs.SkipAll && !cancelled {
		return "", fmt.Errorf("grep %s: %v", root, err)
	}
	// D5 取消诚实：被取消的部分扫描不得伪装成完整「无命中」——旧实现预取消
	// ctx 走到 ``(no matches ...)`` 是假完备。
	if total == 0 && fenced == 0 && gaps == 0 && !cancelled {
		return fmt.Sprintf("(no matches for %q under %s)", p.Pattern, filepath.ToSlash(root)), nil
	}
	// 头部行：游标续页指引与围栏/缺口记账共用一行，前置。matchedTotal 语义
	// 如实：扫描在预算帽提前止步时 total 是「至少」，不给伪造总数。
	// scan 二元完备性（D5，不照抄上游三元组——gaea 单遍全扫无页概念）：
	// complete=全树扫完且零缺口；partial=预算止步/有缺口/被取消，附原因。
	var out strings.Builder
	pageFull := total >= maxResults
	if pageFull || p.Offset > 0 || fenced > 0 || gaps > 0 || cancelled {
		scan := "complete"
		if pageFull || gaps > 0 || cancelled {
			scan = "partial"
		}
		fmt.Fprintf(&out, "[grep page] offset=%d shown=%d scan=%s", pageOffset, total, scan)
		// 续读指引只在「预算止步且零缺口且未被取消」时发——缺口/取消意味着
		// 覆盖是下界，续页会静默跳过没看到的文件。
		if pageFull && gaps == 0 && !cancelled {
			fmt.Fprintf(&out, " — more matches may remain; re-run with offset=%d", pageOffset+total)
		}
		if gaps > 0 {
			fmt.Fprintf(&out, " — %d path(s) unreadable (coverage is a lower bound; narrow the path or fix permissions)", gaps)
		}
		if cancelled {
			out.WriteString(" — scan interrupted")
		}
		if fenced > 0 {
			fmt.Fprintf(&out, " — credential-fence skipped %d file(s)", fenced)
		}
		out.WriteByte('\n')
	}
	out.WriteString(strings.TrimRight(body.String(), "\n"))
	if total == 0 {
		if gaps > 0 || cancelled {
			fmt.Fprintf(&out, "\n(no matches in scanned files for %q under %s)", p.Pattern, filepath.ToSlash(root))
		} else {
			fmt.Fprintf(&out, "\n(no matches for %q under %s)", p.Pattern, filepath.ToSlash(root))
		}
	}
	if total >= maxResults {
		// 续读建议与头行同口径：缺口/取消时覆盖是下界，只给收窄指引。
		if gaps == 0 && !cancelled {
			fmt.Fprintf(&out, "\n[truncated at %d results — refine the pattern, narrow the path, raise max_results, or re-run with offset=%d]", maxResults, pageOffset+total)
		} else {
			fmt.Fprintf(&out, "\n[truncated at %d results — refine the pattern, narrow the path, or raise max_results]", maxResults)
		}
	}
	return strings.TrimRight(out.String(), "\n"), nil
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
	// D5：超长行（>1MB 缓冲）会让 scanner 截断——该文件余下命中静默丢失是
	// 旧实现的真缺口，如实报错让调用方记账（gap），不再吞掉。
	if serr := scanner.Err(); serr != nil {
		return n, fmt.Errorf("scan %s: %w", path, serr)
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
