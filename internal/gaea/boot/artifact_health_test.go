package boot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sessionquery"
	"github.com/xuri/excelize/v2"
)

// 交接卡体检钩子（2026-11-27 AI 反馈 #14）：xlsx 打不开、公式格没有缓存值都要
// 点名；别的类型不给结论（交接卡只报存在与否）。
func TestArtifactHealthHook(t *testing.T) {
	wireArtifactHealth()
	dir := t.TempDir()

	junk := filepath.Join(dir, "broken.xlsx")
	if err := os.WriteFile(junk, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := artifactHealthOf(junk); !strings.Contains(got, "打不开") {
		t.Errorf("坏文件应报打不开：%q", got)
	}

	f := excelize.NewFile()
	_ = f.SetCellFormula(f.GetSheetName(0), "A1", "1+1")
	nocache := filepath.Join(dir, "nocache.xlsx")
	if err := f.SaveAs(nocache); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if got := artifactHealthOf(nocache); !strings.Contains(got, "没有缓存值") || !strings.Contains(got, "xlsx_recalc") {
		t.Errorf("缺缓存值应点名并指路重算：%q", got)
	}

	plain := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := artifactHealthOf(plain); got != "" {
		t.Errorf("非 xlsx 不该给结论：%q", got)
	}
}

// artifactHealthOf 取当前接线的钩子；没接线时如实返回。
func artifactHealthOf(path string) string {
	if sessionquery.ArtifactHealth == nil {
		return "(未接线)"
	}
	return sessionquery.ArtifactHealth(path)
}
