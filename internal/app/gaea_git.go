package app

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Git 面板最小集（蒸馏规划 2b，决策门 D3 采纳推荐默认）：单仓库
// status / diff / stage / unstage / discard / commit / history——
// 无 push/pull/fetch（与源 better-sidebar git tab 一致）；执行 git CLI，
// 参数走 exec 列表（无 shell 注入面），仓库锚定 gaea 工作区 cwd。
// 非 Git 仓库 / git 未安装 → 诚实错误，前端显示为面板空态。

// GitFileStatus 是一条文件状态（porcelain v1 的 X/Y 两列展开）。
type GitFileStatus struct {
	Path      string `json:"path"`      // 相对仓库根（原样，含 / 分隔）
	X         string `json:"x"`         // 暂存区状态（' '、A、M、D、R…）
	Y         string `json:"y"`         // 工作区状态
	Staged    bool   `json:"staged"`    // 已暂存在改动（X 非 ' ' 且非 ?）
	Untracked bool   `json:"untracked"` // 未跟踪（??）
	Deleted   bool   `json:"deleted"`   // 工作区删除（Y=D）
	Modified  bool   `json:"modified"`  // 工作区修改（Y=M）
	Renamed   bool   `json:"renamed"`   // 重命名（R）
}

// GitStatus 是仓库状态快照。
type GitStatus struct {
	IsRepo bool            `json:"isRepo"`
	Branch string          `json:"branch,omitempty"`
	Ahead  int             `json:"ahead,omitempty"`
	Behind int             `json:"behind,omitempty"`
	Files  []GitFileStatus `json:"files"`
	Error  string          `json:"error,omitempty"` // 非 repo/git 缺失等诚实错误
}

// GitCommitInfo 是一条历史提交。
type GitCommitInfo struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author,omitempty"`
	Ts      int64  `json:"ts,omitempty"` // Unix 秒
}

// gitRun 在工作区 cwd 执行 git 子命令，返回 stdout。stderr 拼进 error。
func gitRun(args ...string) (string, error) {
	cwd := gaeaCwd()
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errGit(msg)
	}
	return string(out), nil
}

// errGit 把 git/环境错误包装为面板可读文案（git 未安装时 CombinedOutput
// 之前就失败，stderr 为空）。
func errGit(detail string) error {
	return &gitError{detail: detail}
}

type gitError struct{ detail string }

func (e *gitError) Error() string {
	d := e.detail
	if strings.Contains(d, "executable file not found") || strings.Contains(d, "system cannot find the file") {
		return "未找到 git 命令，请确认已安装 Git 并加入 PATH"
	}
	if strings.Contains(d, "not a git repository") {
		return "当前工作区不是 Git 仓库"
	}
	return d
}

// GaeaGitStatus 返回仓库状态：分支 + ahead/behind + 文件列表（porcelain v1）。
func (a *App) GaeaGitStatus() GitStatus {
	out, err := gitRun("status", "--porcelain=v1", "-b")
	if err != nil {
		return GitStatus{Files: []GitFileStatus{}, Error: err.Error()}
	}
	st := GitStatus{Files: []GitFileStatus{}}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			// ## branch...upstream [ahead N, behind M]（无 upstream 时无 ...）
			head := strings.TrimPrefix(line, "## ")
			if i := strings.Index(head, "..."); i >= 0 {
				head = head[:i]
			}
			if i := strings.IndexAny(head, " \t"); i >= 0 {
				head = head[:i]
			}
			st.Branch = head
			if i := strings.Index(line, "ahead "); i >= 0 {
				if f := strings.Fields(line[i+6:]); len(f) > 0 {
					st.Ahead, _ = strconv.Atoi(f[0])
				}
			}
			if i := strings.Index(line, "behind "); i >= 0 {
				if f := strings.Fields(line[i+7:]); len(f) > 0 {
					st.Behind, _ = strconv.Atoi(f[0])
				}
			}
			continue
		}
		if len(line) < 4 {
			continue
		}
		x, y := string(line[0]), string(line[1])
		path := line[3:]
		f := GitFileStatus{Path: path, X: x, Y: y}
		f.Untracked = x == "?" && y == "?"
		f.Staged = !f.Untracked && x != " "
		f.Deleted = y == "D"
		f.Modified = y == "M"
		f.Renamed = x == "R" || y == "R"
		st.Files = append(st.Files, f)
	}
	st.IsRepo = true
	return st
}

// GaeaGitDiff 返回单个已跟踪文件的 unified diff 文本（staged=true 取暂存区
// 与 HEAD 的差异）。未跟踪文件没有 diff 语义，前端走内容预览。
func (a *App) GaeaGitDiff(path string, staged bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errGit("path 为空")
	}
	args := []string{"diff", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	return gitRun(args...)
}

// GaeaGitStage 把文件加入暂存区（新增文件同样适用）。
func (a *App) GaeaGitStage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRun(append([]string{"add", "--"}, clean...)...)
	return err
}

// GaeaGitUnstage 把文件移出暂存区（不动工作区内容；首条提交前仓库无 HEAD
// 时 git 会报错，原样透出为诚实错误）。
func (a *App) GaeaGitUnstage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRun(append([]string{"reset", "-q", "HEAD", "--"}, clean...)...)
	return err
}

// GaeaGitDiscard 丢弃单个文件的工作区改动（危险操作，前端两击确认；
// 只接受已跟踪文件，未跟踪文件走删除语义不在本刀）。
func (a *App) GaeaGitDiscard(path string) error {
	if strings.TrimSpace(path) == "" {
		return errGit("path 为空")
	}
	_, err := gitRun("checkout", "-q", "--", path)
	return err
}

// GaeaGitCommit 提交暂存区（不代 add：暂存什么提交什么，与源一致）；
// 返回新提交短 hash。
func (a *App) GaeaGitCommit(message string) (string, error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "", errGit("提交说明为空")
	}
	if _, err := gitRun("commit", "-m", msg); err != nil {
		return "", err
	}
	out, err := gitRun("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", nil // 提交已成功，短 hash 拿不到不回滚提交（诚实降级）
	}
	return strings.TrimSpace(out), nil
}

// GaeaGitLog 返回最近提交历史（不含 push 相关语义）。
func (a *App) GaeaGitLog(limit int) ([]GitCommitInfo, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	out, err := gitRun("-c", "core.quotepath=false", "log", "-n", strconv.Itoa(limit),
		"--pretty=format:%h%x1f%s%x1f%an%x1f%at%x1e")
	if err != nil {
		return nil, err
	}
	commits := []GitCommitInfo{}
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.Split(rec, "\x1f")
		c := GitCommitInfo{}
		if len(parts) > 0 {
			c.Hash = parts[0]
		}
		if len(parts) > 1 {
			c.Subject = parts[1]
		}
		if len(parts) > 2 {
			c.Author = parts[2]
		}
		if len(parts) > 3 {
			c.Ts, _ = strconv.ParseInt(parts[3], 10, 64)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func nonEmptyPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
