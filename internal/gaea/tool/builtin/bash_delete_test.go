package builtin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A6 递归删除 guard（Reasonix 2.x shellsafe/delete 蒸馏）测试：
// 灾难面四族硬拒绝 / 字面性 fail-closed / 合法递归删除放行 / PowerShell 词法。

func testGuard(t *testing.T, command, workDir string, ps bool) deleteVerdict {
	t.Helper()
	return guardRecursiveDelete(command, workDir, ps)
}

func TestGuardRecursiveDeleteRejectsCatastrophic(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	wd := filepath.Join(home, "proj", "ws")
	cases := []struct {
		name    string
		command string
		workDir string
	}{
		{"根", "rm -rf /", wd},
		{"home 本身", "rm -rf " + home, wd},
		{"home 波浪号", "rm -rf ~", wd},
		{"home 祖先", "rm -rf " + filepath.Dir(home), wd},
		{"工作目录本身", "rm -rf " + wd, wd},
		{"工作目录祖先", "rm -rf " + filepath.Dir(filepath.Dir(wd)), wd},
		{"相对路径上级", "rm -rf ../..", wd},
		{"相对路径点", "rm -rf .", wd},
		{"链式尾部灾难", "go build ./... && rm -rf /", wd},
		{"rm 单独 -r", "rm -r /", wd},
		{"长旗标", "rm --recursive /", wd},
		{"find -delete 起点=工作目录", "find " + wd + " -name '*.tmp' -delete", wd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := testGuard(t, tc.command, tc.workDir, false)
			if !v.reject {
				t.Fatalf("应拒绝: %s", tc.command)
			}
			if v.reason == "" {
				t.Fatal("拒绝必须带诚实说明")
			}
		})
	}
	// 卷根（Windows 形态）
	if runtime.GOOS == "windows" {
		for _, cmd := range []string{"rm -rf C:\\", "rm -rf C:", "rm -rf C:/"} {
			if v := testGuard(t, cmd, wd, false); !v.reject {
				t.Fatalf("应拒绝卷根: %s", cmd)
			}
		}
	}
}

func TestGuardRecursiveDeleteAllowsLegit(t *testing.T) {
	home, _ := os.UserHomeDir()
	wd := filepath.Join(home, "proj", "ws")
	allow := []string{
		"rm -rf ./build",                      // 工作区内子目录
		"rm -rf " + filepath.Join(wd, "dist"), // 工作区内绝对路径
		"rm -rf /tmp/gaea-walkthrough",        // temp 清理（工作区外普通递归删除=刻意偏差，放行）
		"rm file.txt",                         // 非递归
		"rm -f file.txt",                      // -f 不含 r
		"ls -r",                               // 递归旗标但不是 rm
		"grep -r foo .",                       // 同上
		"git clean -fdx",                      // git 面不在本 guard 范围
		"find " + wd + " -name '*.go' -print", // find 无 -delete
		"go test ./... && go vet ./...",       // 无删除
		"rm " + filepath.Join(wd, "a") + " " + filepath.Join(wd, "b"), // 多目标非递归
	}
	for _, cmd := range allow {
		if v := testGuard(t, cmd, wd, false); v.reject {
			t.Fatalf("应放行 %q，被拒: %s", cmd, v.reason)
		}
	}
}

func TestGuardRecursiveDeleteNonliteralFailClosed(t *testing.T) {
	wd := "/home/u/ws"
	for _, cmd := range []string{
		"rm -rf $TMPDIR/x",
		"rm -rf build/*",
		"rm -rf ${OUT}",
		"rm -rf ~/$DIR",
	} {
		if v := testGuard(t, cmd, wd, false); !v.reject {
			t.Fatalf("字面性不足应 fail-closed: %s", cmd)
		}
		if !strings.Contains(testGuard(t, cmd, wd, false).reason, "literal") {
			t.Fatalf("拒绝说明应指出字面性: %s", cmd)
		}
	}
}

func TestGuardRecursiveDeletePowerShell(t *testing.T) {
	wd := `C:\Users\u\ws`
	reject := []string{
		"Remove-Item -Recurse -Force C:\\Users\\u", // home
		"Remove-Item -Path C:\\ -Recurse",          // 卷根
		"rm -Recurse C:\\Users",                    // PS 的 rm 别名 + home 祖先
		"ri -r C:\\Users\\u\\ws",                   // ri 别名 + 工作目录
	}
	for _, cmd := range reject {
		if v := testGuard(t, cmd, wd, true); !v.reject {
			t.Fatalf("PS 应拒绝: %s", cmd)
		}
	}
	allow := []string{
		"Remove-Item -Recurse -Force .\\build",        // 工作区内
		"Remove-Item -Recurse C:\\Users\\u\\ws\\dist", // 工作区内
		"Get-ChildItem -Recurse",                      // 非删除
	}
	for _, cmd := range allow {
		if v := testGuard(t, cmd, wd, true); v.reject {
			t.Fatalf("PS 应放行 %q: %s", cmd, v.reason)
		}
	}
}

func TestGuardRecursiveDeleteFindXargs(t *testing.T) {
	wd := "/home/u/ws"
	// find 起点=工作目录喂给 xargs rm -rf → 拒
	if v := testGuard(t, "find "+wd+" -name '*.tmp' -print0 | xargs -0 rm -rf", wd, false); !v.reject {
		t.Fatal("find|xargs rm -rf 应拒")
	}
	// find 工作区内子目录 → 放行
	if v := testGuard(t, "find "+wd+"/logs -name '*.log' -print0 | xargs -0 rm -f", wd, false); v.reject {
		t.Fatalf("find 子目录普通删除应放行: %s", v.reason)
	}
}

func TestGuardRecursiveDeleteQuotesSurvive(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	wd := "/home/u/ws"
	// 引号内路径照常分类（用平台真实绝对路径——Windows 下 POSIX 形不是 IsAbs）
	if v := testGuard(t, `rm -rf "`+home+`"`, wd, false); !v.reject {
		t.Fatal("引号包裹的灾难目标应拒")
	}
	// 引号内的分号不切段（整体仍是一个 rm——目标含分号形态按字面分类）
	if v := testGuard(t, `rm -rf "/tmp/a;b"`, wd, false); v.reject {
		t.Fatalf("合法带分号路径应放行: %s", v.reason)
	}
}

// ── v1.1（对源复核后补齐的上游形态）─────────────────────────────────────────

func TestGuardRecursiveDeleteWrappers(t *testing.T) {
	wd := "/home/u/ws"
	reject := []string{
		"sudo rm -rf /",             // 包装解包
		"sudo rm -rf /home/u",       // sudo + home
		"env rm -rf /",              // env 直包
		"env FOO=1 rm -rf /",        // env 带赋值
		"nohup rm -rf /home/u",      // nohup
		"sudo -k rm -rf /",          // 白名单外 sudo 旗标 → 解包失败 → 递归参数扫描
		"sudo -u root rm -rf /home", // -u 取值跳过后仍是灾难目标
		"xargs rm -rf",              // 裸 xargs：删除树根未知
		"echo x | xargs rm -rf",     // 管道 xargs（起点不可静态定）
	}
	for _, cmd := range reject {
		if v := testGuard(t, cmd, wd, false); !v.reject {
			t.Fatalf("应拒: %s", cmd)
		}
	}
	allow := []string{
		"echo rm -rf file",    // 无关基座不放删除语义（上游 default 分支）
		"sudo rm -rf ./build", // sudo 但目标合法
		"env TMPDIR=/x rm -rf ./build",
	}
	for _, cmd := range allow {
		if v := testGuard(t, cmd, wd, false); v.reject {
			t.Fatalf("应放行 %q: %s", cmd, v.reason)
		}
	}
}

func TestGuardRecursiveDeleteFindExecAndOptions(t *testing.T) {
	wd := "/home/u/ws"
	reject := []string{
		"find /home/u/ws -exec rm -rf {} ;",        // find -exec 嵌套递归
		"find /home/u/ws -execdir rm -rf {} +",     // -execdir：树根按命中项漂移
		"find /home/u/ws -L -name '*.tmp' -delete", // -follow/-L ⇒ unknownOption fail-closed
		"rm -rf --unknown-flag /home/u",            // 未知长旗标 fail-closed
		"rm -Z -rf /home/u",                        // 未知短旗标 fail-closed
	}
	for _, cmd := range reject {
		if v := testGuard(t, cmd, wd, false); !v.reject {
			t.Fatalf("应拒: %s", cmd)
		}
	}
	allow := []string{
		"find /home/u/ws/logs -exec rm {} ;", // -exec 非递归 rm
		"rm -rf -- /home/u/ws/build",         // -- 后目标照常分类
		"rm -rfv ./build",                    // 组合短旗标含已知字符
		"rm --rec ./build",                   // 长旗标前缀缩写（上游 HasPrefix 口径）
	}
	for _, cmd := range allow {
		if v := testGuard(t, cmd, wd, false); v.reject {
			t.Fatalf("应放行 %q: %s", cmd, v.reason)
		}
	}
}

func TestGuardRecursiveDeletePowerShellV11(t *testing.T) {
	wd := `C:\Users\u\ws`
	reject := []string{
		"Remove-Item -Path:C:\\ -Recurse",              // 冒号内联 Path + 卷根（v1 漏）
		"Remove-Item -Path C:\\ -Recurse",              // 分离式 Path
		"rd -Recurse C:\\Users\\u",                     // rd 别名（v1 漏）
		"rmdir -Recurse C:\\Users",                     // rmdir 别名
		"Remove-Item -Weird -Recurse C:\\Users\\u\\ws", // 未知旗标 fail-closed
	}
	for _, cmd := range reject {
		if v := testGuard(t, cmd, wd, true); !v.reject {
			t.Fatalf("PS 应拒: %s", cmd)
		}
	}
	allow := []string{
		"Remove-Item -Recurse:$false -Path C:\\Users",             // 显式关递归（v1 会误伤）
		"Remove-Item -Confirm:SilentlyContinue -Recurse .\\build", // 内联取值+合法目标
	}
	for _, cmd := range allow {
		if v := testGuard(t, cmd, wd, true); v.reject {
			t.Fatalf("PS 应放行 %q: %s", cmd, v.reason)
		}
	}
}

func TestGuardRecursiveDeleteCmdShell(t *testing.T) {
	wd := "C:\\Users\\u\\ws"
	reject := []string{
		"cmd /c rd /s /q C:\\Users\\u", // cmd.exe rd /s
		`cmd /c "rd /s /q C:\"`,        // 引号负载卷根
	}
	for _, cmd := range reject {
		if v := testGuard(t, cmd, wd, false); !v.reject {
			t.Fatalf("cmd 应拒: %s", cmd)
		}
	}
	if v := testGuard(t, "cmd /c go build ./...", wd, false); v.reject {
		t.Fatalf("cmd 非删除应放行: %s", v.reason)
	}
}
