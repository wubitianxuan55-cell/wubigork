package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
