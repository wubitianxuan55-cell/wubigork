package xlsxedit

// save_atomic_test.go — A 案原子写回归（2026-11-19 用户拍板；pan_1795058214142
// action-2 占用 A/B 挂账）：ApplyOps 落盘走同目录临时文件+RenameWithRetry——
// ①成功腿：目标更新+临时文件零残留；②失败腿（Windows 只读目标挡 rename）：
// 原文件逐字节未动+临时文件零残留。占用态（Excel/WPS 打开中）真机行为由
// internal/app/xlsx_occupy_ab_test.go（env 门控）复验，不在此重复。

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xuri/excelize/v2"
)

func atomicFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "atomic.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", 100)
	f.SetCellValue("Sheet1", "A2", 200)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func shaOfFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func assertNoTempLeft(t *testing.T, dir string) {
	t.Helper()
	if left, _ := filepath.Glob(filepath.Join(dir, ".gaea-xlsxedit-*")); len(left) > 0 {
		t.Fatalf("临时文件残留: %v", left)
	}
}

func TestApplyOpsAtomicSuccessAndCleanup(t *testing.T) {
	path := atomicFixture(t)
	summary, err := ApplyOps(path, []Op{{Type: "set_value", Sheet: "Sheet1", Target: "A3", Value: 300}})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 {
		t.Fatalf("summary=%v", summary)
	}
	// 目标真被更新且可解析。
	fr, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("重开失败: %v", err)
	}
	v, _ := fr.GetCellValue("Sheet1", "A3")
	fr.Close()
	if v != "300" {
		t.Fatalf("A3=%q want 300", v)
	}
	assertNoTempLeft(t, filepath.Dir(path))
}

func TestApplyOpsAtomicFailureLeavesOriginalIntact(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("只读目标挡 rename 是 Windows 语义；linux rename 可替换只读目标")
	}
	path := atomicFixture(t)
	before := shaOfFile(t, path)
	// Windows 只读属性 → RenameWithRetry 替换失败（ACCESS_DENIED），模拟
	// 「临时文件已写好但收尾被挡」的中途失败形态。
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o644) }() // 还原可写，防 TempDir 清理失败

	_, err := ApplyOps(path, []Op{{Type: "set_value", Sheet: "Sheet1", Target: "A3", Value: 300}})
	if err == nil {
		t.Fatal("只读目标应失败")
	}
	if after := shaOfFile(t, path); after != before {
		t.Fatalf("失败后原文件被改动（半写）：before=%s after=%s", before[:12], after[:12])
	}
	assertNoTempLeft(t, filepath.Dir(path))
}
