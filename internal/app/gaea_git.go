package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// Git 面板最小集（蒸馏规划 2b，决策门 D3 采纳推荐默认）：单仓库
// status / diff / stage / unstage / discard / commit / history——
// 无 push/pull/fetch（与源 better-sidebar git tab 一致）；执行 git CLI，
// 参数走 exec 列表（无 shell 注入面），仓库锚定 gaea 工作区 cwd。
// 非 Git 仓库 / git 未安装 → 诚实错误，前端显示为面板空态。

// GitFileStatus 是一条文件状态（porcelain v1 的 X/Y 两列展开）。
type GitFileStatus struct {
	Path      string `json:"path"`               // 相对仓库根（原样，含 / 分隔）
	X         string `json:"x"`                  // 暂存区状态（' '、A、M、D、R…）
	Y         string `json:"y"`                  // 工作区状态
	Staged    bool   `json:"staged"`             // 已暂存在改动（X 非 ' ' 且非 ? 且非冲突——冲突不可提交）
	Untracked bool   `json:"untracked"`          // 未跟踪（??）
	Deleted   bool   `json:"deleted"`            // 工作区删除（Y=D）
	Modified  bool   `json:"modified"`           // 工作区修改（Y=M）
	Renamed   bool   `json:"renamed"`            // 重命名（R）
	Conflict  bool   `json:"conflict,omitempty"` // 未合并冲突（任一 U，或 AA/DD/AU/UA/DU/UD）
	OrigPath  string `json:"origPath,omitempty"` // 重命名/复制来源路径（R/C 的 porcelain -z 第二字段，工作区内才带）
}

// GitStatus 是仓库状态快照。
type GitStatus struct {
	IsRepo           bool            `json:"isRepo"`
	Branch           string          `json:"branch,omitempty"`
	Ahead            int             `json:"ahead,omitempty"`
	Behind           int             `json:"behind,omitempty"`
	Files            []GitFileStatus `json:"files"`
	Error            string          `json:"error,omitempty"` // 非 repo/git 缺失等诚实错误
	IsLinkedWorktree bool            `json:"isLinkedWorktree,omitempty"`
	MainWorktreePath string          `json:"mainWorktreePath,omitempty"`
}

// GitCommitInfo 是一条历史提交。
type GitCommitInfo struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author,omitempty"`
	Ts      int64  `json:"ts,omitempty"` // Unix 秒
}

// gitWorkspace 返回 git 面的工作区锚：gaeaCwd 优先，回退进程启动目录。
func gitWorkspace() string {
	cwd := gaeaCwd()
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return cwd
}

// Git 子进程预算（G1 深化，验收欠账收口：超时+输出预算）：读面 20s、写面
// 60s（commit 可能跑 pre-commit 钩子）；stdout 预算读面 8MB、diff 4MB、写面
// 1MB。var 而非 const：测试可收窄预算实测溢出路径。
var (
	gitReadTimeout  = 20 * time.Second
	gitWriteTimeout = 60 * time.Second
	gitReadBudget   = 8 << 20
	gitDiffBudget   = 4 << 20
	gitWriteBudget  = 1 << 20
)

// errGitOutputBudget 由 boundedWriter 溢出时返回；调用方决定诚实报错
// （status/log/worktree list）还是降级为截断展示（diff）。
var errGitOutputBudget = errors.New("git 输出超出预算")

// boundedWriter 收满 limit 后拒绝继续写入：os/exec 的拷贝 goroutine 收到
// 错误即停读并关闭管道读端，git 写端 EPIPE 终止；进程滞留由 ctx 超时强杀兜底。
type boundedWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	room := w.limit - w.buf.Len()
	if room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		w.buf.Write(p)
	}
	if w.buf.Len() >= w.limit {
		return len(p), errGitOutputBudget
	}
	return len(p), nil
}

// gitRunBudgeted 是全部 git 子命令的唯一出口：exec 列表传参（无 shell 注入
// 面）+ 超时 + stdout 预算 + stderr 拼进 error。
func gitRunBudgeted(cwd string, timeout time.Duration, budget int, args ...string) (string, error) {
	// v4.374（Reasonix gitcmd 蒸馏）：中和仓库本地 .git/config 里的
	// filter.<driver>——diff 是唯一会在工作树内容上跑 clean filter 的子命令，
	// 而 .gitattributes 是不可信输入（克隆来的仓库可借查看 diff 执行任意
	// clean 程序）。置空 clean/process 并关 required 后 filter 是恒等直通，
	// diff 仍渲染真实变更。失败尽力而为（非 repo 等场景无 filter 可中和）。
	// v4.375：-c 硬化对（fsmonitor/maintenance 关后台钩子）注入每次调用。
	argv := make([]string, 0, 4+len(args)+len(cleanFilterOverrides(cwd)))
	argv = append(argv, "-C", cwd)
	argv = append(argv, gitHardeningArgs()...)
	argv = append(argv, cleanFilterOverrides(cwd)...)
	argv = append(argv, args...)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", argv...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框（git 面板每次调用）
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // 永不交互式要凭据（面板场景无 TTY 可答）
		"GIT_OPTIONAL_LOCKS=0",  // 只读面不写 index.lock 副锁
		"GIT_CONFIG_NOSYSTEM=1", // 系统级 gitconfig 不参与（不可信面收敛）
	)
	var stderr strings.Builder
	stdout := &boundedWriter{limit: budget}
	cmd.Stdout = stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if errors.Is(err, errGitOutputBudget) {
			return stdout.buf.String(), err
		}
		if ctx.Err() == context.DeadlineExceeded {
			return "", errGit("git 超时（" + timeout.String() + "）：仓库过大或盘区响应缓慢，可稍后重试")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errGit(msg)
	}
	return stdout.buf.String(), nil
}

// gitRunAt 读面子命令（status/diff/log/worktree list）：20s 超时 + 8MB 预算。
func gitRunAt(cwd string, args ...string) (string, error) {
	return gitRunBudgeted(cwd, gitReadTimeout, gitReadBudget, args...)
}

// gitRunWrite 写面子命令（add/reset/checkout/commit）：60s 超时（pre-commit
// 钩子可合法地跑 lint/测试）+ 1MB 预算。
func gitRunWrite(cwd string, args ...string) (string, error) {
	return gitRunBudgeted(cwd, gitWriteTimeout, gitWriteBudget, args...)
}

// gitRun 在工作区 cwd 执行 git 子命令（读面默认档）。
func gitRun(args ...string) (string, error) {
	return gitRunAt(gitWorkspace(), args...)
}

// sameDirPath 目录路径等值：先 Clean 归一分隔符，Windows 大小写不敏感，
// 其余平台逐字节（同一仓库两条仅大小写不同的 worktree 路径属病理场景）。
func sameDirPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func gitRepoRoot(start string) (string, error) {
	out, err := gitRunAt(start, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(strings.Split(out, "\n")[0])
	if root == "" {
		return "", errGit("当前工作区不是 Git 仓库")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// workspaceStatusPath maps a repo-root-relative status path into the current
// workspace. Files outside a nested workspace are intentionally hidden: the
// file tree cannot safely open or mutate them.
func workspaceStatusPath(repoRoot, workspace, statusPath string) (string, bool) {
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(repoRoot, workspaceAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	statusPath = filepath.ToSlash(statusPath)
	if rel == "." {
		return strings.TrimPrefix(statusPath, "./"), true
	}
	prefix := strings.TrimSuffix(rel, "/") + "/"
	if !strings.HasPrefix(statusPath, prefix) {
		return "", false
	}
	return strings.TrimPrefix(statusPath, prefix), true
}

// gitHardeningArgs 返回每次 git 调用前注入的 -c 硬化对（须位于子命令前）：
// fsmonitor/maintenance 关掉后台钩子与守护进程，后台线程不因面板查询被拉起。
func gitHardeningArgs() []string {
	return []string{"-c", "core.fsmonitor=false", "-c", "maintenance.auto=false"}
}

// cleanFilterOverrides enumerates the repo-local filter.<name> sections and
// returns `-c` overrides that neutralize each one (clean/process emptied,
// required off). Best-effort: any error (not a repo, git missing, no
// filters) yields nil and the caller proceeds without overrides.
func cleanFilterOverrides(cwd string) []string {
	probe := exec.Command("git", "-C", cwd, "config", "--local", "--name-only", "--get-regexp", `^filter\.`)
	proc.HideWindow(probe) // Windows: 防止弹出 cmd 黑框
	list, err := probe.Output()
	if err != nil {
		// git config 在无匹配时以 exit 1 退出——输出可能仍为空，二者都视为
		// 「无 filter」。解析失败同样静默放弃（fail-open：不阻塞正常 git 面）。
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(string(list), "\n") {
		line = strings.TrimSpace(line)
		// 形如 filter.<name>.<key>；name 可含点（section 子级），取首段后全段。
		if !strings.HasPrefix(line, "filter.") {
			continue
		}
		rest := strings.TrimPrefix(line, "filter.")
		dot := strings.Index(rest, ".")
		if dot <= 0 {
			continue
		}
		name := rest[:dot]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out,
			"-c", "filter."+name+".clean=",
			"-c", "filter."+name+".process=",
			"-c", "filter."+name+".required=false",
		)
	}
	return out
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

// GaeaGitStatus 返回仓库状态：分支 + ahead/behind + 文件列表（porcelain v1），
// 并标注工作区是否位于 linked worktree（G1 深化）。
func (a *App) GaeaGitStatus() GitStatus {
	workspace := gitWorkspace()
	repoRoot, err := gitRepoRoot(workspace)
	if err != nil {
		return GitStatus{Files: []GitFileStatus{}, Error: err.Error()}
	}
	out, err := gitRunAt(repoRoot, "status", "--porcelain=v1", "-z", "-b")
	if err != nil {
		return GitStatus{Files: []GitFileStatus{}, Error: err.Error()}
	}
	st := GitStatus{Files: []GitFileStatus{}}
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		line := strings.TrimRight(records[i], "\r\n")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			st.Branch, st.Ahead, st.Behind = parseGitStatusBranch(line)
			continue
		}
		if len(line) < 4 {
			continue
		}
		x, y := string(line[0]), string(line[1])
		path := line[3:]
		var origPath string
		if x == "R" || x == "C" || y == "R" || y == "C" {
			// porcelain -z stores the destination first, followed by the source
			// path as its own NUL-delimited field（G1.7 起保留来源供前端溯源）。
			if i+1 < len(records) {
				i++
				origPath = strings.TrimRight(records[i], "\r\n")
			}
		}
		path, ok := workspaceStatusPath(repoRoot, workspace, path)
		if !ok {
			continue
		}
		f := GitFileStatus{Path: path, X: x, Y: y}
		f.Untracked = x == "?" && y == "?"
		f.Conflict = isConflictXY(x, y)
		f.Staged = !f.Untracked && !f.Conflict && x != " "
		f.Deleted = y == "D"
		f.Modified = y == "M"
		f.Renamed = x == "R" || y == "R"
		// 来源路径走同一工作区归一/过滤口径：工作区外来源不上树（不溯源）。
		if origPath != "" {
			if o, ok2 := workspaceStatusPath(repoRoot, workspace, origPath); ok2 {
				f.OrigPath = o
			}
		}
		st.Files = append(st.Files, f)
	}
	st.IsRepo = true
	// linked worktree 判定（G1 深化）：--absolute-git-dir 与
	// --git-common-dir 不同 → 工作区在 linked worktree（非主检出）。
	// --path-format=absolute 需 git ≥2.31，取不到时尽力而为不标注，不阻塞状态面。
	if gitDir, err := gitRunAt(workspace, "rev-parse", "--absolute-git-dir"); err == nil {
		if commonDir, err2 := gitRunAt(workspace, "rev-parse", "--path-format=absolute", "--git-common-dir"); err2 == nil {
			gitDir = strings.TrimSpace(strings.Split(gitDir, "\n")[0])
			commonDir = strings.TrimSpace(strings.Split(commonDir, "\n")[0])
			if gitDir != "" && commonDir != "" && !sameDirPath(gitDir, commonDir) {
				st.IsLinkedWorktree = true
				st.MainWorktreePath = filepath.Dir(filepath.FromSlash(commonDir))
			}
		}
	}
	return st
}

// isConflictXY 未合并冲突形态（porcelain v1 X/Y）：任一 U，或双方改写的
// AA/DD 与「我方已暂存/对方工作区」组合 AU/UA/DU/UD。与前端 gitStatusInk
// 同一规则（六形态）。
func isConflictXY(x, y string) bool {
	if x == "U" || y == "U" {
		return true
	}
	switch x + y {
	case "AA", "DD", "AU", "UA", "DU", "UD":
		return true
	}
	return false
}

func parseGitStatusBranch(line string) (branch string, ahead, behind int) {
	head := strings.TrimSpace(strings.TrimPrefix(line, "## "))
	if strings.HasPrefix(head, "No commits yet on ") {
		return strings.TrimPrefix(head, "No commits yet on "), 0, 0
	}
	if strings.HasPrefix(head, "HEAD (no branch)") {
		head = "HEAD"
	}
	if i := strings.Index(head, "..."); i >= 0 {
		head = head[:i]
	}
	if i := strings.IndexAny(head, " \t"); i >= 0 {
		head = head[:i]
	}
	if i := strings.Index(line, "ahead "); i >= 0 {
		if f := strings.Fields(line[i+6:]); len(f) > 0 {
			ahead, _ = strconv.Atoi(strings.TrimSuffix(f[0], ","))
		}
	}
	if i := strings.Index(line, "behind "); i >= 0 {
		if f := strings.Fields(line[i+7:]); len(f) > 0 {
			behind, _ = strconv.Atoi(strings.TrimSuffix(f[0], "]"))
		}
	}
	return head, ahead, behind
}

// GaeaGitDiff 返回单个已跟踪文件的 unified diff 文本（staged=true 取暂存区
// 与 HEAD 的差异）。未跟踪文件没有 diff 语义，前端走内容预览。
func (a *App) GaeaGitDiff(path string, staged bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errGit("path 为空")
	}
	// --no-ext-diff/--no-textconv（v4.375，Reasonix gitcmd 基线）：外置 diff
	// 驱动与 textconv 会执行 .gitattributes 配置的外部程序，查看面板 diff
	// 时一律关闭（clean filter 由 gitRun 层统一中和）。
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, err := gitRunBudgeted(gitWorkspace(), gitReadTimeout, gitDiffBudget, args...)
	if errors.Is(err, errGitOutputBudget) {
		// 展示层诚实截断：保留已收内容 + 截断标记，不当错误炸面板。
		return out + "\n\n[diff 输出超过 4MB，已截断]", nil
	}
	return out, err
}

// GaeaGitStage 把文件加入暂存区（新增文件同样适用）。
func (a *App) GaeaGitStage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRunWrite(gitWorkspace(), append([]string{"add", "--"}, clean...)...)
	return err
}

// GaeaGitUnstage 把文件移出暂存区（不动工作区内容；首条提交前仓库无 HEAD
// 时 git 会报错，原样透出为诚实错误）。
func (a *App) GaeaGitUnstage(paths []string) error {
	clean := nonEmptyPaths(paths)
	if len(clean) == 0 {
		return errGit("未选择文件")
	}
	_, err := gitRunWrite(gitWorkspace(), append([]string{"reset", "-q", "HEAD", "--"}, clean...)...)
	return err
}

// GaeaGitDiscard 丢弃单个文件的工作区改动（危险操作，前端两击确认；
// 只接受已跟踪文件，未跟踪文件走删除语义不在本刀）。
func (a *App) GaeaGitDiscard(path string) error {
	if strings.TrimSpace(path) == "" {
		return errGit("path 为空")
	}
	_, err := gitRunWrite(gitWorkspace(), "checkout", "-q", "--", path)
	return err
}

// GaeaGitCommit 提交暂存区（不代 add：暂存什么提交什么，与源一致）；
// 返回新提交短 hash。
func (a *App) GaeaGitCommit(message string) (string, error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "", errGit("提交说明为空")
	}
	if _, err := gitRunWrite(gitWorkspace(), "commit", "-m", msg); err != nil {
		return "", err
	}
	out, err := gitRunAt(gitWorkspace(), "rev-parse", "--short", "HEAD")
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

// GitWorktree 是一条 worktree（git worktree list --porcelain 展开）。
type GitWorktree struct {
	Path       string `json:"path"`             // worktree 本地路径（原生分隔符，Clean 过）
	Head       string `json:"head,omitempty"`   // HEAD 短 hash（bare 无）
	Branch     string `json:"branch,omitempty"` // 已去掉 refs/heads/ 前缀（detached 无）
	IsMain     bool   `json:"isMain"`           // 主检出（列表第一条）
	IsBare     bool   `json:"isBare,omitempty"`
	IsDetached bool   `json:"isDetached,omitempty"`
	IsLocked   bool   `json:"isLocked,omitempty"` // git worktree lock
	LockedNote string `json:"lockedNote,omitempty"`
	IsCurrent  bool   `json:"isCurrent"` // 工作区 cwd 所在的 worktree
}

// GaeaGitWorktrees 列出仓库全部 worktree（只读展示）。gaea 全应用单一
// workspace 锚（ga.cfg.Workspace），FileTree/Git/Agent 共用它——worktree
// 「切换」= 工作区切换器（G2/W1 级产品门），不在本刀；也不做 add/remove/prune。
func (a *App) GaeaGitWorktrees() ([]GitWorktree, error) {
	workspace := gitWorkspace()
	repoRoot, err := gitRepoRoot(workspace)
	if err != nil {
		return nil, err
	}
	out, err := gitRunAt(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseGitWorktrees(out, repoRoot), nil
}

// parseGitWorktrees 解析 `git worktree list --porcelain`：记录以空行分隔，
// 字段行前缀 worktree/HEAD/branch/bare/detached/locked（locked 可带原因）。
// 路径含空格安全；含换行的路径是病理场景不做承诺。isCurrent 与 repoRoot
// （工作区自身 toplevel）等值判定，主检出=第一条记录。
func parseGitWorktrees(out, current string) []GitWorktree {
	list := []GitWorktree{}
	var cur *GitWorktree
	flush := func() {
		if cur != nil {
			list = append(list, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur = &GitWorktree{
				Path: filepath.Clean(filepath.FromSlash(strings.TrimSpace(strings.TrimPrefix(line, "worktree ")))),
			}
		case cur == nil:
			continue
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "branch ")), "refs/heads/")
		case line == "bare":
			cur.IsBare = true
		case line == "detached":
			cur.IsDetached = true
		case strings.HasPrefix(line, "locked"):
			cur.IsLocked = true
			if note := strings.TrimSpace(strings.TrimPrefix(line, "locked")); note != "" {
				cur.LockedNote = note
			}
		}
	}
	flush()
	for i := range list {
		if i == 0 {
			list[i].IsMain = true
		}
		if sameDirPath(list[i].Path, current) {
			list[i].IsCurrent = true
		}
	}
	return list
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
