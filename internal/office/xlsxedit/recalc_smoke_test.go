package xlsxedit

// recalc_smoke_test.go — 重算链实弹用例：真拉 LibreOffice 与内嵌脚本，默认跳过。
//
//	GAEA_SMOKE_RECALC=1 go test ./internal/office/xlsxedit -run TestSmokeRecalc -v

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/xuri/excelize/v2"
)

// smokeRecalcReady 两个实弹用例的共同前置。
func smokeRecalcReady(t *testing.T) {
	t.Helper()
	if os.Getenv("GAEA_SMOKE_RECALC") == "" {
		t.Skip("未设置 GAEA_SMOKE_RECALC")
	}
	if os.Getenv(recalcDisableKey) != "" {
		t.Skip("重算已被 GAEA_RECALC_DISABLE 关闭")
	}
}

// assertRecalculated 断言 B5 的 SUM 公式已被 LibreOffice 算出并写回缓存值——
// 「文件被真正重算」的唯一硬证据（2026-11-26 的缺陷形态正是脚本回成功而缓存值
// 仍为空）。
func assertRecalculated(t *testing.T, path string) {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("重算后打开失败：%v", err)
	}
	defer func() { _ = f.Close() }()
	if got, _ := f.GetCellValue("数据", "B5"); got != "600" {
		t.Fatalf("B5 缓存值 = %q，期望 600（100+200+300）——文件没有被真正重算", got)
	}
	audit, err := Audit(path)
	if err != nil || len(audit.Findings) != 0 {
		t.Fatalf("重算后审计应零发现：err=%v findings=%+v", err, audit.Findings)
	}
}

// TestSmokeRecalc 连跑两次：第一次从零建配置模板，第二次走「复制模板」的稳态路径。
func TestSmokeRecalc(t *testing.T) {
	smokeRecalcReady(t)
	path := recalcFixture(t)
	rep, err := Recalc(path)
	if err != nil {
		t.Fatalf("重算失败：%v", err)
	}
	if rep.Status != "success" || rep.TotalFormulas != 1 {
		t.Fatalf("重算回执不符：%+v", rep)
	}
	assertRecalculated(t, path)
	if entries, err := os.ReadDir(RecalcCacheRoot()); err != nil || len(entries) == 0 {
		t.Fatalf("跑成后应留下配置模板：%v", err)
	}

	path2 := recalcFixture(t)
	if _, err := Recalc(path2); err != nil {
		t.Fatalf("复制模板路径重算失败：%v", err)
	}
	assertRecalculated(t, path2)
}

// findSofficeForTest 实弹用例定位 LibreOffice（与内嵌脚本同一组常见安装位置）。
func findSofficeForTest() string {
	if p := os.Getenv("GAEA_TEST_SOFFICE"); p != "" {
		return p
	}
	if p, err := exec.LookPath("soffice"); err == nil {
		return p
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(os.Getenv("ProgramFiles"), "LibreOffice", "program", "soffice.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "LibreOffice", "program", "soffice.exe"))
	} else {
		candidates = append(candidates, "/Applications/LibreOffice.app/Contents/MacOS/soffice")
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// TestSmokeRecalcIsolatedUnderForeignInstance 回归 2026-11-26 实测的缺陷：机器上
// 另有一个 LibreOffice 实例（默认配置目录）活着时，走默认目录的调用会被转交给
// 它并立刻返回，文件没被重算却回「成功」。独立配置目录后不得受它影响。
func TestSmokeRecalcIsolatedUnderForeignInstance(t *testing.T) {
	smokeRecalcReady(t)
	soffice := findSofficeForTest()
	if soffice == "" {
		t.Skip("本机未找到 LibreOffice")
	}
	const addr = "127.0.0.1:2083"
	foreign := exec.Command(soffice, "--headless", "--norestore", "--nologo", "--nodefault",
		"--accept=socket,host=localhost,port=2083;urp;StarOffice.ComponentContext")
	proc.HideWindow(foreign)
	if err := foreign.Start(); err != nil {
		t.Skipf("起不了外部 LibreOffice 实例：%v", err)
	}
	defer proc.KillTree(foreign)

	// 等它把 accept 端口监听起来（端口已被别的进程占着时这里会提前通过，
	// 那种情况下本用例退化成普通重算，不会假红）。
	deadline := time.Now().Add(30 * time.Second)
	for {
		if conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("外部 LibreOffice 实例 30 秒内没起来")
		}
		time.Sleep(250 * time.Millisecond)
	}
	time.Sleep(1500 * time.Millisecond)

	path := recalcFixture(t)
	rep, err := Recalc(path)
	if err != nil {
		t.Fatalf("外部实例在场时重算失败：%v", err)
	}
	if rep.TotalFormulas != 1 {
		t.Fatalf("重算回执不符：%+v", rep)
	}
	assertRecalculated(t, path)
}
