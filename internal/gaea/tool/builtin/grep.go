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
	return "Search file contents with a regular expression. Returns matches as `path:line: content` lines. Skips binary files and noise directories (node_modules, .git, dist, build, ...). Use `include` (e.g. \"*.go\") to filter by file name glob."
}

func (grepTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "pattern":{"type":"string","description":"Regular expression to search for"},
  "path":{"type":"string","description":"File or directory to search (default \".\" — the working directory)"},
  "include":{"type":"string","description":"Glob to filter files by name, e.g. \"*.go\""},
  "max_results":{"type":"integer","description":"Maximum number of matching lines to return (default 200, cap 1000)","minimum":1}
},
"required":["pattern"]
}`)
}

func (grepTool) ReadOnly() bool { return true }

func (grepTool) CompactDescription() string     { return compactDesc["grep"] }
func (grepTool) CompactSchema() json.RawMessage { return compactSchema["grep"] }

// grepNoiseDirs mirrors agent/compress.go noiseDirs — the directories whose
// contents are never useful grep hits (dependencies, build output, VCS
// internals). Duplicated here because builtin must not import the agent
// package (agent sits above tool in the layering).
var grepNoiseDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	"target": true, "__pycache__": true, ".next": true, ".nuxt": true,
	".cache": true, ".venv": true, "venv": true, "coverage": true,
	"out": true, ".turbo": true, ".devenv": true,
}

func (g grepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Include    string `json:"include"`
		MaxResults int    `json:"max_results"`
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
	total := 0
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
			if path != root && grepNoiseDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if p.Include != "" && !grepIncludeMatch(p.Include, root, path) {
			return nil
		}
		n, err := grepFile(ctx, re, path, maxResults-total, &out)
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
	if total == 0 {
		return fmt.Sprintf("(no matches for %q under %s)", p.Pattern, filepath.ToSlash(root)), nil
	}
	if total >= maxResults {
		fmt.Fprintf(&out, "[truncated at %d results — refine the pattern, narrow the path, or raise max_results]", maxResults)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// grepFile scans one file and appends its matches as `path:line: content`.
// Returns the number of matches emitted. Binary files (NUL byte in the first
// peek window) are skipped.
func grepFile(ctx context.Context, re *regexp.Regexp, path string, budget int, out *strings.Builder) (int, error) {
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
