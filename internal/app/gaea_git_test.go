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
			// G1.7：porcelain -z 第二字段（来源路径）须保留并归一为工作区相对。
			if f.OrigPath != "inside.txt" {
				t.Fatalf("rename source path = %q, want inside.txt", f.OrigPath)
			}
		}
	}
	if !foundRename {
		t.Fatalf("rename destination not parsed from porcelain -z: %+v", st.Files)
	}
}

// G1.7：未合并冲突——真仓制造 UU（两分支改同一文件后 merge），状态须标
// conflict 且不误判为「已暂存」（冲突不可提交）。
func TestGaeaGitStatusConflict(t *testing.T) {
	dir := initTempRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	// 容忍冲突退出码的 runner（merge 冲突时 git 以非零退出）。
	runTolerant := func(args ...string) {
		t.Helper()
		_ = exec.Command("git", append([]string{"-C", dir}, args...)...).Run()
	}
	run("checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "tracked.txt", "feature side\n")
	run("commit", "-q", "-am", "feature edit")
	runTolerant("checkout", "-q", "-")
	writeRepoFile(t, dir, "tracked.txt", "main side\n")
	run("commit", "-q", "-am", "main edit")
	runTolerant("merge", "feature")

	t.Chdir(dir)
	st := (&App{}).GaeaGitStatus()
	var conflict *GitFileStatus
	for i := range st.Files {
		if st.Files[i].Path == "tracked.txt" {
			conflict = &st.Files[i]
		}
	}
	if conflict == nil {
		t.Fatalf("冲突文件应上树: %+v", st.Files)
	}
	if !conflict.Conflict {
		t.Fatalf("tracked.txt 应标记 conflict: %+v", conflict)
	}
	if conflict.Staged {
		t.Fatalf("冲突文件不应计入已暂存（不可提交）: %+v", conflict)
	}
}

// G1.7：二进制文件 diff——clean filter 中和与预算层不得改变 git 的二进制
// 诚实输出（"Binary files ... differ"），前端按前缀显示占位而非假红绿行。
func TestGaeaGitDiffBinary(t *testing.T) {
	dir := initTempRepo(t)
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte{0x00, 0x01, 0x02, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("add", "blob.bin")
	run("commit", "-q", "-m", "binary base")
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte{0x00, 0x09, 0x09, 0x00, 0xFF}, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (&App{}).GaeaGitDiff("blob.bin", false)
	if err != nil {
		t.Fatalf("binary diff: %v", err)
	}
	if !strings.Contains(out, "Binary files") {
		t.Fatalf("二进制 diff 应为 Binary files 行: %q", out)
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

// G1.8：空仓库（无提交 HEAD）降级矩阵——status 诚实报未诞生分支、log/unstage/
// staged-diff 诚实报错（前端走空态/toast）、worktree list 正常（仅主检出）。
func TestGaeaGitUnbornHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("环境无 git CLI")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.name", "t")
	run("config", "user.email", "t@example.com")

	a := &App{}
	t.Chdir(dir)

	// status：IsRepo=true，分支名从「No commits yet on」解析，无文件无 ahead/behind。
	st := a.GaeaGitStatus()
	if !st.IsRepo || st.Branch == "" || st.Ahead != 0 || st.Behind != 0 || len(st.Files) != 0 {
		t.Fatalf("空仓 status = %+v, want isRepo+branch 无文件", st)
	}

	// log：无提交 → 诚实报错（前端 history .catch → 「暂无提交」空态）。
	if _, err := a.GaeaGitLog(5); err == nil {
		t.Fatal("空仓 log 应报错")
	}

	// unstage：无 HEAD——git ≥2.26 把未诞生 HEAD 当空树回退（exit 0，撤销暂存），
	// 是合法语义而非错误；断言成功并确实清空了暂存条目。
	if err := a.GaeaGitUnstage([]string{"x.txt"}); err != nil {
		t.Fatalf("空仓 unstage（未诞生 HEAD=空树回退）不应报错: %v", err)
	}
	st2 := a.GaeaGitStatus()
	for _, f := range st2.Files {
		if f.Path == "x.txt" && f.Staged {
			t.Fatalf("unstage 后 x.txt 不应留在暂存区: %+v", st2.Files)
		}
	}

	// worktree list：空仓正常返回主检出一条（无 HEAD 字段，isCurrent=工作区所在）。
	wts, err := a.GaeaGitWorktrees()
	if err != nil || len(wts) != 1 || !wts[0].IsMain || !wts[0].IsCurrent {
		t.Fatalf("空仓 worktrees = %+v, %v", wts, err)
	}

	// staged diff（未诞生 HEAD）：git ≥2.26 按空树比较——首提前的暂存 diff
	// 全部呈现为新增行（对「首提交」场景正是用户想要的语义），不报错不崩溃。
	writeRepoFile(t, dir, "x.txt", "new\n")
	run("add", "x.txt")
	out, err := a.GaeaGitDiff("x.txt", true)
	if err != nil || !strings.Contains(out, "+new") {
		t.Fatalf("空仓 staged diff 应按空树显示新增: %q, %v", out, err)
	}
}

// G2：提交图——真仓分支+合并拓扑（parents/装饰解析）、分页、分支筛选、
// 分支清单；分支注入拒绝；提交详情（numstat 含二进制 -1、diff、合并诚实空 diff）。
func TestGaeaGitGraphAndDetail(t *testing.T) {
	dir := initTempRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	// 拓扑：init(b1) → feat 分支(b2,b3) → main(b4) → merge feat(b5, 两父)。
	// feature 臂追加 f.txt/新增 g.txt，main 臂新增 h.txt——双臂无交集保干净合并。
	writeRepoFile(t, dir, "f.txt", "b1\n")
	run("add", "f.txt")
	run("commit", "-q", "-m", "b1")
	run("checkout", "-q", "-b", "feature")
	writeRepoFile(t, dir, "f.txt", "b1\nb2\n")
	run("commit", "-q", "-am", "b2")
	writeRepoFile(t, dir, "g.txt", "b3\n")
	run("add", "g.txt")
	run("commit", "-q", "-m", "b3")
	run("checkout", "-q", "-")
	writeRepoFile(t, dir, "h.txt", "b4\n")
	run("add", "h.txt")
	run("commit", "-q", "-m", "b4")
	run("merge", "-q", "--no-ff", "-m", "b5 merge feature", "feature")
	writeRepoFile(t, dir, "blob.bin", "\x00\x01")
	run("add", "blob.bin")
	run("commit", "-q", "-m", "b6 binary")
	t.Chdir(dir)
	a := &App{}

	page, err := a.GaeaGitGraph(0, 10, "")
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if len(page.Commits) != 7 { // initTempRepo 的 init + b1..b6
		t.Fatalf("commits = %d, want 7: %+v", len(page.Commits), page.Commits)
	}
	// 分支清单含两条本地分支（master/main 由 git 版本决定 + feature）。
	foundFeature := false
	for _, b := range page.Branches {
		if b == "feature" {
			foundFeature = true
		}
	}
	if !foundFeature {
		t.Fatalf("branches 应含 feature: %+v", page.Branches)
	}
	// 合并提交（b5）两父。
	var mergeNode *GitGraphNode
	for i := range page.Commits {
		if page.Commits[i].Subject == "b5 merge feature" {
			mergeNode = &page.Commits[i]
		}
	}
	if mergeNode == nil || len(mergeNode.Parents) != 2 {
		t.Fatalf("合并提交应两父: %+v", mergeNode)
	}
	// HEAD 装饰：b6 应带 HEAD -> 当前分支 装饰。
	head := page.Commits[0]
	if len(head.Refs) == 0 || !strings.Contains(strings.Join(head.Refs, ","), "HEAD") {
		t.Fatalf("HEAD 提交应有装饰: %+v", head.Refs)
	}

	// 分页：skip=1 limit=2 → 跳过 HEAD 的 b6，取 b5/b4。
	page2, err := a.GaeaGitGraph(1, 2, "")
	if err != nil {
		t.Fatalf("graph skip: %v", err)
	}
	if len(page2.Commits) != 2 || page2.Commits[0].Subject != "b5 merge feature" || page2.Commits[1].Subject != "b4" {
		t.Fatalf("分页不符: %+v", page2.Commits)
	}

	// 分支筛选：按 feature 取历史，b1 是祖先链尾（无 merge 提交）。
	pageF, err := a.GaeaGitGraph(0, 10, "feature")
	if err != nil {
		t.Fatalf("graph branch: %v", err)
	}
	for _, c := range pageF.Commits {
		if c.Subject == "b5 merge feature" || c.Subject == "b4" {
			t.Fatalf("feature 历史不应含 main 侧提交: %+v", pageF.Commits)
		}
	}

	// 分支注入：'-' 开头与含 '..' 均拒绝。
	if _, err := a.GaeaGitGraph(0, 10, "--all"); err == nil {
		t.Fatal("--all 选项注入应拒绝")
	}
	if _, err := a.GaeaGitGraph(0, 10, "a..b"); err == nil {
		t.Fatal("范围注入应拒绝")
	}

	// 详情：b6 二进制文件 numstat=-1，diff 走预算管道。
	bin := page.Commits[0]
	d, err := a.GaeaGitCommitDetail(bin.Hash)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(d.Files) != 1 || d.Files[0].Add != -1 || d.Files[0].Del != -1 {
		t.Fatalf("二进制 numstat 应 -1: %+v", d.Files)
	}
	if !strings.Contains(d.Diff, "Binary files") {
		t.Fatalf("二进制 diff 应为 Binary files 行: %q", d.Diff)
	}

	// 详情：合并提交（b5）——diff-tree 默认无输出 → Diff 空串诚实提示。
	d5, err := a.GaeaGitCommitDetail(mergeNode.Hash)
	if err != nil {
		t.Fatalf("detail merge: %v", err)
	}
	if d5.Diff != "" {
		t.Fatalf("合并提交默认应无 diff: %q", d5.Diff)
	}
	if len(d5.Parents) != 2 {
		t.Fatalf("详情应含两父: %+v", d5.Parents)
	}

	// hash 注入：非十六进制拒绝。
	if _, err := a.GaeaGitCommitDetail("../../etc/passwd"); err == nil {
		t.Fatal("非 hash 应拒绝")
	}
	if _, err := a.GaeaGitCommitDetail("HEAD"); err == nil {
		t.Fatal("符号 ref 应拒绝（只收十六进制）")
	}
}

// G2：gitRefSafe/gitHashSafe 纯函数边界。
func TestGitRefSafe(t *testing.T) {
	if !gitRefSafe("feature/x") || !gitRefSafe("release-1.2") || !gitRefSafe("中文分支") {
		t.Fatal("常规/中文分支名应放行")
	}
	for _, bad := range []string{"-all", "a..b", "x@{0}", "y.lock", "z.", "a b", ""} {
		if gitRefSafe(bad) {
			t.Fatalf("%q 应拒绝", bad)
		}
	}
	if !gitHashSafe("0123456789abcdef") || gitHashSafe("HEAD") || gitHashSafe("abc") || gitHashSafe("0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Fatal("hash 校验边界不符")
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
