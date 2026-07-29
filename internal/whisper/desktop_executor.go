// Package whisper — desktop_executor.go
// 100% 对齐 ackem desktop-agent/adapters/win/executor.ts
// 桌面动作执行器：22 种 FS/进程操作
package whisper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	textReadLimit = 512_000
	listLimit     = 200
	searchLimit   = 100
	maxDownloadMB = 200
)

// ExecuteResult 执行结果
type ExecuteResult struct {
	OK      bool   `json:"ok"`
	Content string `json:"content"`
	Summary string `json:"summary"`
}

// DesktopExecContext 执行上下文
type DesktopExecContext struct {
	DataRoot    string
	DownloadDir string
	CWD         string
}

// ExecuteDesktopAgentAction 主分发入口
func ExecuteDesktopAgentAction(action DesktopAgentAction, path, pathTo, target, query, url, content string, ctx DesktopExecContext) ExecuteResult {
	cwd := ctx.CWD
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}

	switch action {
	case ActionListFolder:
		return listFolder(path)
	case ActionStatFile:
		return statFile(path)
	case ActionReadText:
		return readTextFile(path)
	case ActionReadDocument:
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" || ext == ".log" {
			return readTextFile(path)
		}
		return ExecuteResult{OK: false, Content: "V1 暂不支持解析该文档格式全文；若为纯文本可改用 read_text", Summary: "文档格式暂未解析"}
	case ActionReadImage:
		if _, err := os.Stat(path); err == nil {
			return ExecuteResult{OK: true, Content: statLineStr(path), Summary: "已定位图片 " + filepath.Base(path)}
		}
		return ExecuteResult{OK: false, Content: "文件不存在", Summary: "图片不存在"}
	case ActionSearchFiles:
		return searchFiles(path, query, cwd)
	case ActionGrepText:
		return grepText(path, query, cwd)
	case ActionOpenFolder, ActionOpenFile:
		return shellOpen(path)
	case ActionOpenApp, ActionFocusApp:
		return openAppTarget(coalesce(target, path))
	case ActionCloseApp, ActionCloseFile:
		return closeAppTarget(coalesce(target, filepath.Base(path)))
	case ActionCopyPath:
		os.MkdirAll(filepath.Dir(pathTo), 0755)
		if err := copyFile(path, pathTo); err != nil {
			return ExecuteResult{OK: false, Content: err.Error(), Summary: "复制失败"}
		}
		return ExecuteResult{OK: true, Content: "已复制到 " + pathTo, Summary: "已复制 " + filepath.Base(path)}
	case ActionMovePath:
		os.MkdirAll(filepath.Dir(pathTo), 0755)
		if err := os.Rename(path, pathTo); err != nil {
			return ExecuteResult{OK: false, Content: err.Error(), Summary: "移动失败"}
		}
		return ExecuteResult{OK: true, Content: "已移动到 " + pathTo, Summary: "已移动 " + filepath.Base(path)}
	case ActionMkdir:
		if err := os.MkdirAll(path, 0755); err != nil {
			return ExecuteResult{OK: false, Content: err.Error(), Summary: "创建目录失败"}
		}
		return ExecuteResult{OK: true, Content: "已创建 " + path, Summary: "已创建目录 " + filepath.Base(path)}
	case ActionWriteText:
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return ExecuteResult{OK: false, Content: err.Error(), Summary: "写入失败"}
		}
		return ExecuteResult{OK: true, Content: "已写入 " + path, Summary: "已写入 " + filepath.Base(path)}
	case ActionDeletePath:
		// 移入回收站（Windows 下通过 PowerShell）
		return trashItem(path)
	case ActionDownloadFile:
		dest := path
		if dest == "" {
			dest = filepath.Join(defaultDownloadDir(ctx.DownloadDir), filepath.Base(url))
		}
		return downloadHTTPS(url, dest)
	case ActionDownloadAndInstall:
		dir := defaultDownloadDir(ctx.DownloadDir)
		os.MkdirAll(dir, 0755)
		fileName := filepath.Base(url)
		if fileName == "" || fileName == "." {
			fileName = "installer.exe"
		}
		dest := filepath.Join(dir, fileName)
		dl := downloadHTTPS(url, dest)
		if !dl.OK {
			return dl
		}
		shellOpen(filepath.Dir(dest))
		run := shellOpen(dest)
		return ExecuteResult{OK: run.OK, Content: dl.Content + "\n" + run.Content, Summary: "已下载并开始安装 " + fileName}
	case ActionRunInstaller:
		return shellOpen(path)
	case ActionImportToAckem:
		importsDir := filepath.Join(ctx.DataRoot, "imports")
		os.MkdirAll(importsDir, 0755)
		dest := filepath.Join(importsDir, filepath.Base(path))
		if err := copyFile(path, dest); err != nil {
			return ExecuteResult{OK: false, Content: err.Error(), Summary: "导入失败"}
		}
		return ExecuteResult{OK: true, Content: "已复制到 " + dest, Summary: "已导入 " + filepath.Base(path)}
	default:
		return ExecuteResult{OK: false, Content: "未知 action: " + string(action), Summary: "执行失败"}
	}
}

// ─── 文件操作 ────────────────────────────────────────────────────

func statLineStr(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "路径不存在"
	}
	kind := "文件"
	if info.IsDir() {
		kind = "目录"
	}
	return fmt.Sprintf("%s · %d 字节 · 修改于 %s", kind, info.Size(), info.ModTime().Format("2006-01-02 15:04:05"))
}

func statFile(path string) ExecuteResult {
	if _, err := os.Stat(path); err != nil {
		return ExecuteResult{OK: false, Content: "路径不存在", Summary: "stat 失败：" + path}
	}
	return ExecuteResult{OK: true, Content: statLineStr(path), Summary: "已查看 " + filepath.Base(path) + " 信息"}
}

func listFolder(path string) ExecuteResult {
	entries, err := os.ReadDir(path)
	if err != nil {
		return ExecuteResult{OK: false, Content: "路径不存在", Summary: "目录不存在：" + path}
	}

	var lines []string
	count := 0
	for _, e := range entries {
		if count >= listLimit {
			break
		}
		prefix := "[FILE]"
		if e.IsDir() {
			prefix = "[DIR]"
		}
		lines = append(lines, prefix+" "+e.Name())
		count++
	}

	suffix := ""
	if len(entries) >= listLimit {
		suffix = fmt.Sprintf("\n…（仅显示前 %d 项）", listLimit)
	}
	return ExecuteResult{OK: true, Content: strings.Join(lines, "\n") + suffix, Summary: fmt.Sprintf("已列出 %s（%d 项）", filepath.Base(path), count)}
}

func readTextFile(path string) ExecuteResult {
	info, err := os.Stat(path)
	if err != nil {
		return ExecuteResult{OK: false, Content: "文件不存在", Summary: "读取失败：" + path}
	}
	if info.IsDir() {
		return ExecuteResult{OK: false, Content: "路径是目录", Summary: "无法以文本读取目录"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ExecuteResult{OK: false, Content: err.Error(), Summary: "读取失败"}
	}

	truncated := len(data) > textReadLimit
	if truncated {
		data = data[:textReadLimit]
	}

	text := string(data)
	summary := "已读取 " + filepath.Base(path)
	if truncated {
		text += fmt.Sprintf("\n…（仅显示前 %d 字节）", textReadLimit)
		summary += "（截断）"
	}
	return ExecuteResult{OK: true, Content: text, Summary: summary}
}

func searchFiles(root, query, cwd string) ExecuteResult {
	if root == "" {
		root = cwd
	}
	if _, err := os.Stat(root); err != nil {
		return ExecuteResult{OK: false, Content: "路径不存在", Summary: "搜索失败"}
	}

	q := strings.ToLower(query)
	if q == "" {
		q = strings.ToLower(filepath.Base(root))
	}

	var hits []string
	walkLimit := 0
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || walkLimit >= searchLimit {
			return nil
		}
		// 深度限制
		rel, _ := filepath.Rel(root, p)
		if depth := strings.Count(rel, string(os.PathSeparator)); depth > 6 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), q) {
			hits = append(hits, p)
			walkLimit++
		}
		return nil
	})

	if len(hits) == 0 {
		return ExecuteResult{OK: true, Content: "（未找到匹配文件）", Summary: fmt.Sprintf("搜索「%s」找到 0 项", query)}
	}
	return ExecuteResult{OK: true, Content: strings.Join(hits, "\n"), Summary: fmt.Sprintf("搜索「%s」找到 %d 项", query, len(hits))}
}

func grepText(root, query, cwd string) ExecuteResult {
	if root == "" {
		root = cwd
	}
	q := strings.ToLower(query)
	if q == "" {
		return ExecuteResult{OK: false, Content: "缺少搜索关键词", Summary: "grep 失败"}
	}

	textExts := map[string]bool{".txt": true, ".md": true, ".json": true, ".csv": true, ".log": true, ".js": true, ".ts": true, ".tsx": true, ".py": true, ".go": true}

	var hits []string
	walkLimit := 0
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || walkLimit >= 50 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if depth := strings.Count(rel, string(os.PathSeparator)); depth > 3 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !textExts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if len(data) > 64000 {
			data = data[:64000]
		}
		if strings.Contains(strings.ToLower(string(data)), q) {
			hits = append(hits, p)
			walkLimit++
		}
		return nil
	})

	if len(hits) == 0 {
		return ExecuteResult{OK: true, Content: "（未找到包含该文本的文件）", Summary: fmt.Sprintf("grep「%s」0 个文件", query)}
	}
	return ExecuteResult{OK: true, Content: strings.Join(hits, "\n"), Summary: fmt.Sprintf("grep「%s」%d 个文件", query, len(hits))}
}

// ─── Shell / 进程操作 ────────────────────────────────────────────

func shellOpen(path string) ExecuteResult {
	var cmd *exec.Cmd
	if _, err := os.Stat(path); err == nil {
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	} else {
		return ExecuteResult{OK: false, Content: "路径不存在", Summary: "打开失败：" + path}
	}
	if err := cmd.Start(); err != nil {
		return ExecuteResult{OK: false, Content: err.Error(), Summary: "打开失败：" + path}
	}
	return ExecuteResult{OK: true, Content: "已打开 " + path, Summary: "已打开 " + filepath.Base(path)}
}

func openAppTarget(target string) ExecuteResult {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("Start-Process '%s'", strings.ReplaceAll(target, "'", "''")))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ExecuteResult{OK: false, Content: string(out), Summary: "未能打开 " + target}
	}
	return ExecuteResult{OK: true, Content: "已启动 " + target, Summary: "已打开 " + target}
}

func closeAppTarget(target string) ExecuteResult {
	name := strings.TrimSuffix(strings.ToLower(target), ".exe")
	script := fmt.Sprintf(
		"$p = Get-Process -Name '%s' -ErrorAction SilentlyContinue; if (-not $p) { exit 2 }; $p | ForEach-Object { $_.CloseMainWindow() | Out-Null }; exit 0",
		strings.ReplaceAll(name, "'", "''"),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ExecuteResult{OK: false, Content: strings.TrimSpace(string(out)), Summary: "未能关闭 " + target}
	}
	return ExecuteResult{OK: true, Content: "已请求关闭 " + target, Summary: "已关闭 " + target}
}

func trashItem(path string) ExecuteResult {
	// 使用 PowerShell 将文件移入回收站
	script := fmt.Sprintf(
		"$shell = New-Object -ComObject Shell.Application; $folder = $shell.Namespace(0); $item = $folder.ParseName('%s'); if ($item) { $item.InvokeVerb('delete') } else { exit 1 }",
		strings.ReplaceAll(path, "'", "''"),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ExecuteResult{OK: false, Content: strings.TrimSpace(string(out)), Summary: "删除失败"}
	}
	return ExecuteResult{OK: true, Content: "已移入回收站：" + path, Summary: "已删除 " + filepath.Base(path)}
}

// ─── 下载 ────────────────────────────────────────────────────────

func downloadHTTPS(url, destPath string) ExecuteResult {
	if !strings.HasPrefix(url, "https://") {
		return ExecuteResult{OK: false, Content: "仅支持 HTTPS 下载", Summary: "下载被拒绝"}
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)

	resp, err := http.Get(url)
	if err != nil {
		return ExecuteResult{OK: false, Content: err.Error(), Summary: "下载失败"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ExecuteResult{OK: false, Content: fmt.Sprintf("HTTP %d", resp.StatusCode), Summary: "下载失败"}
	}

	// 限制 200MB
	limitReader := io.LimitReader(resp.Body, maxDownloadMB*1024*1024)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return ExecuteResult{OK: false, Content: err.Error(), Summary: "下载失败"}
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return ExecuteResult{OK: false, Content: err.Error(), Summary: "保存失败"}
	}

	return ExecuteResult{OK: true, Content: fmt.Sprintf("已下载到 %s（%d 字节）", destPath, len(data)), Summary: "已下载 " + filepath.Base(destPath)}
}

func defaultDownloadDir(settingsDir string) string {
	if strings.TrimSpace(settingsDir) != "" {
		return settingsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "LightWhisperDownloads")
}

// ─── 辅助 ────────────────────────────────────────────────────────

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
