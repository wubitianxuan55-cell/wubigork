// Package office — executor.go
package office

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ExecResult struct {
	Success bool   `json:"success"`
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Summary string `json:"summary"`
	Error   string `json:"error,omitempty"`
}

func Execute(action DesktopAgentAction, path, target, query, url, content string) ExecResult {
	switch action {
	case ActionReadText: return execReadText(path)
	case ActionListFolder: return execListFolder(path)
	case ActionStatFile: return execStatFile(path)
	case ActionSearchFile: return execSearchFile(path, query)
	case ActionOpenFile: return execOpenFile(path)
	case ActionCopyFile: return execCopyFile(path, target)
	case ActionMoveFile: return execMoveFile(path, target)
	case ActionCreateDir: return execCreateDir(path)
	case ActionWriteFile: return execWriteFile(path, content)
	case ActionDeleteFile: return execDeleteFile(path)
	case ActionWebSearch: return execWebSearch(query)
	case ActionWebFetch: return execWebFetch(url)
	default: return ExecResult{Success: false, Action: string(action), Error: "unknown: " + string(action)}
	}
}

func execReadText(path string) ExecResult {
	data, err := os.ReadFile(path)
	if err != nil { return ExecResult{Success: false, Action: "read_text", Path: path, Error: err.Error()} }
	text := string(data)
	if len(text) > 5000 { text = text[:5000] + "\n…(truncated)" }
	return ExecResult{Success: true, Action: "read_text", Path: path, Content: text, Summary: fmt.Sprintf("read %s (%d bytes)", filepath.Base(path), len(data))}
}

func execListFolder(path string) ExecResult {
	if path == "" { path = "." }
	entries, err := os.ReadDir(path)
	if err != nil { return ExecResult{Success: false, Action: "list_folder", Path: path, Error: err.Error()} }
	var lines []string
	for i, e := range entries {
		if i >= 200 { lines = append(lines, "…(more)"); break }
		name := e.Name()
		if e.IsDir() { name += "/" }
		info, _ := e.Info()
		size := ""
		if info != nil && !e.IsDir() { size = fmt.Sprintf("  %s", formatSize(info.Size())) }
		lines = append(lines, name+size)
	}
	return ExecResult{Success: true, Action: "list_folder", Path: path, Content: strings.Join(lines, "\n"), Summary: fmt.Sprintf("%s: %d entries", path, len(entries))}
}

func execStatFile(path string) ExecResult {
	info, err := os.Stat(path)
	if err != nil { return ExecResult{Success: false, Action: "stat_file", Path: path, Error: err.Error()} }
	lines := []string{
		fmt.Sprintf("Name: %s", info.Name()),
		fmt.Sprintf("Size: %s", formatSize(info.Size())),
		fmt.Sprintf("Modified: %s", info.ModTime().Format("2006-01-02 15:04")),
		fmt.Sprintf("IsDir: %v", info.IsDir()),
	}
	return ExecResult{Success: true, Action: "stat_file", Path: path, Content: strings.Join(lines, "\n"), Summary: fmt.Sprintf("stat %s", filepath.Base(path))}
}

func execSearchFile(dir, query string) ExecResult {
	if dir == "" { dir = "." }
	if query == "" { return ExecResult{Success: false, Action: "search_file", Error: "missing query"} }
	var found []string
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || len(found) >= 50 { return filepath.SkipDir }
		if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(query)) { found = append(found, p) }
		return nil
	})
	if len(found) == 0 { return ExecResult{Success: true, Action: "search_file", Summary: fmt.Sprintf("no match for '%s' in %s", query, dir)} }
	return ExecResult{Success: true, Action: "search_file", Content: strings.Join(found, "\n"), Summary: fmt.Sprintf("found %d matches for '%s'", len(found), query)}
}

func execOpenFile(path string) ExecResult {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	if err := cmd.Start(); err != nil { return ExecResult{Success: false, Action: "open_file", Path: path, Error: err.Error()} }
	return ExecResult{Success: true, Action: "open_file", Path: path, Summary: "opened " + filepath.Base(path)}
}

func execCopyFile(src, dst string) ExecResult {
	sf, err := os.Open(src)
	if err != nil { return ExecResult{Success: false, Action: "copy_file", Path: src, Error: err.Error()} }
	defer sf.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil { return ExecResult{Success: false, Action: "copy_file", Error: err.Error()} }
	df, err := os.Create(dst)
	if err != nil { return ExecResult{Success: false, Action: "copy_file", Path: dst, Error: err.Error()} }
	defer df.Close()
	if _, err := io.Copy(df, sf); err != nil { return ExecResult{Success: false, Action: "copy_file", Error: err.Error()} }
	return ExecResult{Success: true, Action: "copy_file", Path: dst, Summary: fmt.Sprintf("copied %s → %s", filepath.Base(src), dst)}
}

func execMoveFile(src, dst string) ExecResult {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil { return ExecResult{Success: false, Action: "move_file", Error: err.Error()} }
	if err := os.Rename(src, dst); err != nil { return ExecResult{Success: false, Action: "move_file", Error: err.Error()} }
	return ExecResult{Success: true, Action: "move_file", Path: dst, Summary: fmt.Sprintf("moved %s → %s", filepath.Base(src), dst)}
}

func execCreateDir(path string) ExecResult {
	if err := os.MkdirAll(path, 0755); err != nil { return ExecResult{Success: false, Action: "create_dir", Path: path, Error: err.Error()} }
	return ExecResult{Success: true, Action: "create_dir", Path: path, Summary: "created dir " + path}
}

func execWriteFile(path, content string) ExecResult {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return ExecResult{Success: false, Action: "write_file", Path: path, Error: err.Error()} }
	if err := os.WriteFile(path, []byte(content), 0644); err != nil { return ExecResult{Success: false, Action: "write_file", Path: path, Error: err.Error()} }
	return ExecResult{Success: true, Action: "write_file", Path: path, Summary: fmt.Sprintf("wrote %s (%d bytes)", filepath.Base(path), len(content))}
}

func execDeleteFile(path string) ExecResult {
	if err := os.Remove(path); err != nil { return ExecResult{Success: false, Action: "delete_file", Path: path, Error: err.Error()} }
	return ExecResult{Success: true, Action: "delete_file", Path: path, Summary: "deleted " + filepath.Base(path)}
}

func execWebSearch(query string) ExecResult {
	if query == "" { return ExecResult{Success: false, Action: "web_search", Error: "missing query"} }
	cmd := exec.Command("cmd", "/c", "start", "", "https://www.bing.com/search?q="+strings.ReplaceAll(query, " ", "+"))
	if err := cmd.Start(); err != nil { return ExecResult{Success: false, Action: "web_search", Error: err.Error()} }
	return ExecResult{Success: true, Action: "web_search", Summary: "searching: " + query}
}

func execWebFetch(url string) ExecResult {
	if url == "" { return ExecResult{Success: false, Action: "web_fetch", Error: "missing URL"} }
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil { return ExecResult{Success: false, Action: "web_fetch", Error: err.Error()} }
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10000))
	if err != nil { return ExecResult{Success: false, Action: "web_fetch", Error: err.Error()} }
	return ExecResult{Success: true, Action: "web_fetch", Content: string(body), Summary: fmt.Sprintf("fetched %s (%d bytes)", url, len(body))}
}

func formatSize(n int64) string {
	const unit = 1024
	if n < unit { return fmt.Sprintf("%d B", n) }
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit { div *= unit; exp++ }
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
