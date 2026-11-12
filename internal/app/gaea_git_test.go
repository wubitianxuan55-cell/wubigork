package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// initTempRepo 建一个真实临时 git 仓库（testGit 依赖环境装有 git CLI，
// 与运行时依赖一致；git 缺失时跳过）。
func initTempRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("环境无 git CLI，跳过 Git 面板测试")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.name", "t")
	run("config", "user.email", "t@example.com")
	run("config", "core.autocrlf", "false") // Windows 上行为确定化
	writeRepoFile(t, dir, "tracked.txt", "l1\n")
	run("add", "tracked.txt")
	run("commit", "-q", "-m", "init")
	return dir
}

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGaeaGitStatusAndDiff(t *testing.T) {
	dir := initTempRepo(t)
	t.Chdir(dir)
	writeRepoFile(t, dir, "tracked.txt", "l1\nl2\n")
	writeRepoFile(t, dir, "new.txt", "brand new")

	a := &App{}
	st := a.GaeaGitStatus()
	if !st.IsRepo || st.Branch == "" {
		t.Fatalf("status = %+v, want isRepo+branch", st)
	}
	var tracked *GitFileStatus
	for i := range st.Files {
		if st.Files[i].Path == "tracked.txt" {
			tracked = &st.Files[i]
		}
	}
	if tracked == nil || !tracked.Modified || tracked.Staged {
		t.Fatalf("tracked.txt 应为工作区修改: %+v", st.Files)
	}

	// stage 后：tracked.txt 入暂存区，new.txt 为未跟踪
	if err := a.GaeaGitStage([]string{"tracked.txt"}); err != nil {
		t.Fatalf("stage: %v", err)
	}
	st = a.GaeaGitStatus()
	var stagedFile *GitFileStatus
	for i := range st.Files {
		if st.Files[i].Path == "tracked.txt" {
			stagedFile = &st.Files[i]
		}
	}
	if stagedFile == nil || !stagedFile.Staged {
		t.Fatalf("tracked.txt 应已暂存: %+v", st.Files)
	}

	diff, err := a.GaeaGitDiff("tracked.txt", true)
	if err != nil || !strings.Contains(diff, "+l2") {
		t.Fatalf("staged diff = %q, %v", diff, err)
	}

	// unstage 后回到工作区修改
	if err := a.GaeaGitUnstage([]string{"tracked.txt"}); err != nil {
		t.Fatalf("unstage: %v", err)
	}
	st = a.GaeaGitStatus()
	still := false
	for _, f := range st.Files {
		if f.Path == "tracked.txt" && f.Staged {
			still = true
		}
	}
	if still {
		t.Fatal("unstage 后不应留在暂存区")
	}
}

func TestGaeaGitStatusNestedWorkspaceAndRename(t *testing.T) {
	repo := initTempRepo(t)
	ws := filepath.Join(repo, "workspace")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, "workspace/inside.txt", "base\n")
	writeRepoFile(t, repo, "outside.txt", "base\n")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("add", "workspace/inside.txt", "outside.txt")
	run("commit", "-q", "-m", "nested baseline")

	writeRepoFile(t, repo, "workspace/inside.txt", "changed\n")
	writeRepoFile(t, repo, "workspace/new.txt", "new\n")
	writeRepoFile(t, repo, "outside.txt", "changed outside\n")
	writeRepoFile(t, repo, "outside-new.txt", "outside\n")
	t.Chdir(ws)

	st := (&App{}).GaeaGitStatus()
	got := map[string]GitFileStatus{}
	for _, f := range st.Files {
		got[f.Path] = f
	}
	if _, ok := got["inside.txt"]; !ok {
		t.Fatalf("workspace change missing after root-relative normalization: %+v", st.Files)
	}
	if _, ok := got["new.txt"]; !ok {
		t.Fatalf("workspace untracked file missing: %+v", st.Files)
	}
	if _, ok := got["outside.txt"]; ok {
		t.Fatalf("outside tracked file leaked into nested workspace: %+v", st.Files)
	}
	if _, ok := got["outside-new.txt"]; ok {
		t.Fatalf("outside untracked file leaked into nested workspace: %+v", st.Files)
	}

	run("mv", "workspace/inside.txt", "workspace/renamed.txt")
	st = (&App{}).GaeaGitStatus()
	foundRename := false
	for _, f := range st.Files {
		if f.Path == "renamed.txt" && f.Renamed {
			foundRename = true
		}
	}
	if !foundRename {
		t.Fatalf("rename destination not parsed from porcelain -z: %+v", st.Files)
	}
}

func TestGaeaGitCommitLogDiscard(t *testing.T) {
	dir := initTempRepo(t)
	t.Chdir(dir)
	writeRepoFile(t, dir, "tracked.txt", "v1\n")
	a := &App{}
	if err := a.GaeaGitStage([]string{"tracked.txt"}); err != nil {
		t.Fatal(err)
	}
	hash, err := a.GaeaGitCommit("首次提交")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if hash == "" {
		t.Fatal("应返回短 hash")
	}
	if _, err := a.GaeaGitCommit("  "); err == nil {
		t.Fatal("空说明应报错")
	}

	log, err := a.GaeaGitLog(10)
	if err != nil || len(log) < 2 {
		t.Fatalf("log = %+v, %v", log, err)
	}
	if log[0].Subject != "首次提交" || log[0].Hash == "" || log[0].Ts == 0 {
		t.Fatalf("最新提交 = %+v", log[0])
	}

	// discard：工作区改动被还原
	writeRepoFile(t, dir, "tracked.txt", "v2\n")
	if err := a.GaeaGitDiscard("tracked.txt"); err != nil {
		t.Fatalf("discard: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "tracked.txt"))
	if strings.ReplaceAll(string(b), "\r\n", "\n") != "v1\n" {
		t.Fatalf("discard 后内容 = %q, want v1", b)
	}
}

// gitTopLevelOf 返回 dir 所在 git 仓库根（git 向上寻根语义）；非仓库返回 ""。
func gitTopLevelOf(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestGaeaGitNotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("环境无 git CLI")
	}
	dir := t.TempDir()
	// 沙箱选址防「外层仓库」假红：test-all.ps1 为规避沙箱 SAC/AV 扫描把 TMP/TEMP
	// 重定向到仓库内 .tmp（build.bat 同款），t.TempDir() 随之落在外层 git 仓库内——
	// git status 向上寻根会命中外层仓库，「非仓库」测试前提被环境破坏（无论产品代码
	// 是否改动都确定性失败）。此时改在系统用户缓存区（天然在仓库外）另建沙箱；
	// 仍不可得时诚实跳过（前提不可满足，不假装断言）。
	if gitTopLevelOf(dir) != "" {
		if cache, err := os.UserCacheDir(); err == nil {
			if d, err2 := os.MkdirTemp(cache, "gaea-git-nonrepo-"); err2 == nil {
				dir = d
				t.Cleanup(func() { _ = os.RemoveAll(d) })
			}
		}
		if gitTopLevelOf(dir) != "" {
			t.Skip("TMP 落入 git 仓库内且无法选址仓库外沙箱：非仓库断言前提不可满足")
		}
	}
	t.Chdir(dir)
	a := &App{}
	st := a.GaeaGitStatus()
	if st.IsRepo || st.Error == "" {
		t.Fatalf("非仓库应 isRepo=false 且带诚实错误: %+v", st)
	}
	if _, err := a.GaeaGitLog(5); err == nil {
		t.Fatal("非仓库 log 应报错")
	}
}

// v4.374（Reasonix gitcmd 蒸馏）：clean-filter 中和——仓库本地 .git/config
// 里的 filter.<driver> 必须在 git 面调用前置空（.gitattributes 是不可信
// 输入，查看 diff 不得执行攻击者配置的 clean 程序）。
func TestCleanFilterOverrides(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run("init", "-q"); err != nil {
		t.Skipf("git init 不可用: %v (%s)", err, out)
	}
	if out, err := run("config", "--local", "filter.big.clean", "sed s/a/b/g"); err != nil {
		t.Fatalf("配置 filter: %v (%s)", err, out)
	}
	if _, err := run("config", "--local", "filter.big.required", "true"); err != nil {
		t.Fatalf("配置 required: %v", err)
	}

	got := cleanFilterOverrides(dir)
	want := []string{
		"-c", "filter.big.clean=",
		"-c", "filter.big.process=",
		"-c", "filter.big.required=false",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanFilterOverrides = %v, want %v", got, want)
	}
}

// 非 git 仓库：尽力而为返回空（fail-open，不阻塞正常 git 面）。
func TestCleanFilterOverridesNonRepo(t *testing.T) {
	if got := cleanFilterOverrides(t.TempDir()); got != nil {
		t.Fatalf("non-repo should yield nil, got %v", got)
	}
}

// G1 深化：porcelain 解析纯函数——bare/detached/locked（带原因）/branch 前缀
// 剥离/主检出/当前工作区标记。
func TestParseGitWorktrees(t *testing.T) {
	out := "" +
		"worktree /repo/main\n" +
		"HEAD abc12345\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/.wt/bare\n" +
		"bare\n" +
		"\n" +
		"worktree /repo/.wt/det\n" +
		"HEAD def67890\n" +
		"detached\n" +
		"\n" +
		"worktree /repo/.wt/locked\n" +
		"HEAD ghi13579\n" +
		"branch refs/heads/wip\n" +
		"locked 使用中（测试）\n"
	list := parseGitWorktrees(out, "/repo/main")
	if len(list) != 4 {
		t.Fatalf("len = %d, want 4: %+v", len(list), list)
	}
	if !list[0].IsMain || !list[0].IsCurrent || list[0].Branch != "main" || list[0].Head != "abc12345" {
		t.Fatalf("main = %+v", list[0])
	}
	if !list[1].IsBare || list[1].Head != "" || list[1].Branch != "" {
		t.Fatalf("bare = %+v", list[1])
	}
	if !list[2].IsDetached || list[2].Branch != "" || list[2].Head != "def67890" {
		t.Fatalf("detached = %+v", list[2])
	}
	if !list[3].IsLocked || list[3].LockedNote != "使用中（测试）" || list[3].Branch != "wip" || list[3].IsMain {
		t.Fatalf("locked = %+v", list[3])
	}
	if list[1].IsCurrent || list[2].IsCurrent || list[3].IsCurrent {
		t.Fatalf("非当前 worktree 不应标记 isCurrent: %+v", list)
	}
}

// G1 深化：真仓 worktree 面板——主检出/linked/detached 三条，isCurrent 随
// 工作区切换，status 标注 IsLinkedWorktree + MainWorktreePath。
func TestGaeaGitWorktrees(t *testing.T) {
	dir := initTempRepo(t)
	linkedDir := filepath.Join(dir, ".worktrees", "linked")
	detachedDir := filepath.Join(dir, ".worktrees", "det")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("worktree", "add", linkedDir, "-b", "feature/x")
	run("worktree", "add", "--detach", detachedDir, "HEAD")

	// 主检出视角：第一条主检出且当前，linked 有分支，detached 无分支。
	t.Chdir(dir)
	a := &App{}
	list, err := a.GaeaGitWorktrees()
	if err != nil {
		t.Fatalf("worktrees: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3: %+v", len(list), list)
	}
	if !list[0].IsMain || !list[0].IsCurrent {
		t.Fatalf("主检出 = %+v", list[0])
	}
	// linked worktree 不按位置取（git 按路径排序，.worktrees/det 先于 linked）。
	var linked, detached *GitWorktree
	for i := range list {
		switch {
		case list[i].IsDetached:
			detached = &list[i]
		case list[i].Branch == "feature/x":
			linked = &list[i]
		}
	}
	if linked == nil || linked.IsDetached || linked.Branch != "feature/x" || linked.IsMain {
		t.Fatalf("linked = %+v", linked)
	}
	if detached == nil || !detached.IsDetached || detached.Branch != "" || detached.Head == "" {
		t.Fatalf("detached = %+v", detached)
	}
	if st := a.GaeaGitStatus(); st.IsLinkedWorktree {
		t.Fatalf("主检出不应标注 linked worktree: %+v", st)
	}

	// linked 视角：isCurrent 翻到 linked，status 标注 linked + 主检出路径。
	t.Chdir(linkedDir)
	list, err = a.GaeaGitWorktrees()
	if err != nil {
		t.Fatalf("worktrees(linked): %v", err)
	}
	current := 0
	for i := range list {
		if list[i].IsCurrent {
			current++
			if !sameDirPath(list[i].Path, linkedDir) {
				t.Fatalf("isCurrent 指向 %q, want %q", list[i].Path, linkedDir)
			}
		}
	}
	if current != 1 {
		t.Fatalf("isCurrent 应恰好一条: %+v", list)
	}
	st := a.GaeaGitStatus()
	if !st.IsLinkedWorktree || st.MainWorktreePath == "" || !sameDirPath(st.MainWorktreePath, dir) {
		t.Fatalf("linked 状态标注 = %+v, want main=%s", st, dir)
	}
}

// G1 深化验收欠账收口：子进程超时与输出预算（var 预算测试内收窄，defer 还原）。
func TestGitRunTimeoutAndBudget(t *testing.T) {
	dir := initTempRepo(t)
	t.Chdir(dir)

	origBudget, origTimeout := gitReadBudget, gitReadTimeout
	t.Cleanup(func() { gitReadBudget, gitReadTimeout = origBudget, origTimeout })

	// 输出预算：8B 装不下 branch 行（干净仓无 -b 时 porcelain 零输出，故带 -b）
	// → 预算错误（诚实报错，不静默截断）。
	gitReadBudget = 8
	if _, err := gitRunAt(dir, "status", "--porcelain", "-b"); !errors.Is(err, errGitOutputBudget) {
		t.Fatalf("缩预算 status 应报预算错误, got %v", err)
	}

	// 超时：1ms 内启不完 git → 超时诚实文案。
	gitReadTimeout = 1 * time.Millisecond
	_, err := gitRunAt(dir, "status", "--porcelain", "-b")
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("缩超时应报超时, got %v", err)
	}
}

// G1 深化：diff 超预算走展示层截断（保留内容 + 标记），不当错误炸面板。
func TestGaeaGitDiffTruncation(t *testing.T) {
	dir := initTempRepo(t)
	t.Chdir(dir)
	orig := gitDiffBudget
	t.Cleanup(func() { gitDiffBudget = orig })
	gitDiffBudget = 64 // diff 头部就超过 64B

	writeRepoFile(t, dir, "tracked.txt", "l1\nl2\nl3\nl4\n")
	a := &App{}
	out, err := a.GaeaGitDiff("tracked.txt", false)
	if err != nil {
		t.Fatalf("diff 截断不应报错: %v", err)
	}
	if !strings.Contains(out, "已截断") || !strings.Contains(out, "diff --git") {
		t.Fatalf("截断 diff 应含内容与标记: %q", out)
	}
}
