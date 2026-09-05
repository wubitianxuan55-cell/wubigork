package app

// gaea_listdir_test.go — GaeaListDir 小刀用例（v4.96 登记销账）：
// 相对路径回归（原行为不变）/ 绝对路径 / 目录不存在 / 非目录 / 权限透传，
// 外加结构化错误码形态断言（`Error [CODE]: message`，对齐 errcode.go 口径）。

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// armListDirWorkspace 指向独立临时工作区（ga.cfg 全局态先存后还，避免污染
// 同包其他测试——先例见 gaea_diagram_test.go armImageHubGateWithWorkspace）。
func armListDirWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = oldCfg })
	return ws
}

// writeListDirFixture 在 dir 下落一个文件与一个子目录（子目录内再落一文件），
// 返回（文件大小，用于 Size 断言）。
func writeListDirFixture(t *testing.T, dir string) int64 {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return 10
}

// requireDirCode 断言错误带指定结构化码且形态为 `Error [CODE]: message`。
func requireDirCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("期望错误(%s)，实际 nil", code)
	}
	prefix := "Error [" + code + "]: "
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("错误应带 %s 前缀，实际：%s", prefix, err.Error())
	}
	if err.Error() == prefix {
		t.Fatalf("错误码后应有 message，实际：%s", err.Error())
	}
}

// TestListDirRelativeUnchanged 相对路径回归：""=工作区根、子目录、嵌套清理、
// 条目三字段（name/isDir/size）与旧实现一致——行为完全不变。
func TestListDirRelativeUnchanged(t *testing.T) {
	ws := armListDirWorkspace(t)
	size := writeListDirFixture(t, ws)

	entries, err := (&App{}).GaeaListDir("")
	if err != nil {
		t.Fatalf("根目录列举不应出错：%v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("根目录应有 2 条（a.txt + sub），实际 %d：%+v", len(entries), entries)
	}
	byName := map[string]DirEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	a, ok := byName["a.txt"]
	if !ok || a.IsDir || a.Size != size {
		t.Fatalf("a.txt 条目不符（应 isDir=false size=%d）：%+v", size, a)
	}
	sub, ok := byName["sub"]
	if !ok || !sub.IsDir {
		t.Fatalf("sub 条目不符（应 isDir=true）：%+v", sub)
	}

	// 子目录相对列举
	subEntries, err := (&App{}).GaeaListDir("sub")
	if err != nil {
		t.Fatalf("子目录列举不应出错：%v", err)
	}
	if len(subEntries) != 1 || subEntries[0].Name != "b.png" || subEntries[0].IsDir {
		t.Fatalf("sub 列举不符：%+v", subEntries)
	}

	// 嵌套/清理与 Join 语义一致（sub/../sub → sub）
	nested, err := (&App{}).GaeaListDir(filepath.Join("sub", "..", "sub"))
	if err != nil {
		t.Fatalf("嵌套相对路径不应出错：%v", err)
	}
	if len(nested) != 1 || nested[0].Name != "b.png" {
		t.Fatalf("嵌套相对路径列举不符：%+v", nested)
	}
}

// TestListDirAbsPath 绝对路径分支（v4.96 缺口①）：不再 Join 工作区根，
// 工作区外的绝对目录直接列举成功；正斜杠写法（前端 ToSlash 口径）同样可用。
func TestListDirAbsPath(t *testing.T) {
	ws := armListDirWorkspace(t)
	writeListDirFixture(t, ws)
	abs := filepath.Join(ws, "sub")

	for _, rel := range []string{abs, filepath.ToSlash(abs)} {
		entries, err := (&App{}).GaeaListDir(rel)
		if err != nil {
			t.Fatalf("绝对路径 %q 列举不应出错：%v", rel, err)
		}
		if len(entries) != 1 || entries[0].Name != "b.png" {
			t.Fatalf("绝对路径 %q 列举不符：%+v", rel, entries)
		}
	}
	// 反证内含于上：若 IsAbs 分支缺失（旧实现 Join(root, abs)），工作区下会拼出
	// 不存在/非法的奇异路径 → 必然报错；两形态都成功即证明分支生效。
}

// TestListDirMissingStructuredCode 目录不存在 → GAEADIR_NOT_FOUND（相对与绝对同码）。
func TestListDirMissingStructuredCode(t *testing.T) {
	ws := armListDirWorkspace(t)

	_, err := (&App{}).GaeaListDir("nope/deeper")
	requireDirCode(t, err, errCodeDirNotFound)

	_, err = (&App{}).GaeaListDir(filepath.Join(ws, "gone"))
	requireDirCode(t, err, errCodeDirNotFound)
}

// TestListDirNotDirStructuredCode 是文件不是目录 → GAEADIR_NOT_DIR。
func TestListDirNotDirStructuredCode(t *testing.T) {
	ws := armListDirWorkspace(t)
	writeListDirFixture(t, ws)

	_, err := (&App{}).GaeaListDir("a.txt")
	requireDirCode(t, err, errCodeDirNotDir)
}

// TestListDirPermissionPassthrough 权限错误透传：码 GAEADIR_READ_FAILED +
// 底层 os 错误原文保留（不再吞成空切片）。Windows 的 chmod 只置只读位、
// 不剥夺目录读权限（真实 chmod 在该平台只能 Skip），故经系统调用缝注入
// fs.ErrPermission 假错误全平台确定性覆盖 READ_FAILED 分支。
func TestListDirPermissionPassthrough(t *testing.T) {
	ws := armListDirWorkspace(t)
	writeListDirFixture(t, ws)

	oldStat, oldReadDir := osStat, osReadDir
	osReadDir = func(string) ([]os.DirEntry, error) {
		return nil, &fs.PathError{Op: "readdir", Path: filepath.Join(ws, "sub"), Err: fs.ErrPermission}
	}
	t.Cleanup(func() { osStat, osReadDir = oldStat, oldReadDir })

	_, err := (&App{}).GaeaListDir("sub")
	requireDirCode(t, err, errCodeDirRead)
	if !strings.Contains(err.Error(), "读取目录失败") {
		t.Fatalf("读失败 message 应保留语境，实际：%s", err.Error())
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("底层 os 错误原文应透传（permission denied），实际：%s", err.Error())
	}

	// Stat 阶段的非 NotExist 失败（父目录无权限等）同口径：READ_FAILED 透传
	osStat = func(string) (fs.FileInfo, error) {
		return nil, &fs.PathError{Op: "stat", Path: filepath.Join(ws, "sub"), Err: fs.ErrPermission}
	}
	_, err = (&App{}).GaeaListDir("sub")
	requireDirCode(t, err, errCodeDirRead)
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("Stat 阶段 os 错误原文应透传，实际：%s", err.Error())
	}
}
