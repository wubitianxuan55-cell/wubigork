package app

// office_longdoc_verify_layout_test.go — P1-6 版面机检回归钉：空白页（双分页
// 断造）=硬伤 fail；墨覆盖两口径表恒出；渲染链缺席 skip（office 侧守真链）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/docmd"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestLongdocVerifyLayoutBlankPage(t *testing.T) {
	if docmd.FindSoffice() == "" {
		t.Skip("LibreOffice 缺席，layout 门不可测")
	}
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ormRepoRootForSkills(t)}
	defer func() { ga.cfg = oldCfg }()
	dir := t.TempDir()
	spec := map[string]any{
		"title": "版面测试", "font": "宋体", "fontSize": 12,
		"blocks": []map[string]any{
			{"type": "paragraph", "text": "第一页正文内容，占位。"},
			{"type": "pagebreak"},
			{"type": "pagebreak"},
			{"type": "paragraph", "text": "末页正文内容。"},
		},
	}
	sb, _ := json.Marshal(spec)
	specPath := filepath.Join(dir, "spec.json")
	os.WriteFile(specPath, sb, 0o644)
	dx := filepath.Join(dir, "layout.docx")
	if err := runPython([]string{findSkillScript("docx", "scripts", "create_docx.py"), specPath, dx, "--spec"}, 60); err != nil {
		t.Skipf("python 链不可用：%v", err)
	}
	out, err := ldvRun(t, `{"action":"layout","docx":`+ldvJSON(dx)+`}`)
	if err != nil {
		// 渲染链（poppler 三件）缺席=依赖缺失，如实跳过。
		if strings.Contains(err.Error(), "未找到") || strings.Contains(err.Error(), "不可用") {
			t.Skipf("poppler 工具缺席：%v", err)
		}
		t.Fatal(err)
	}
	if !strings.Contains(out, "空白页") {
		t.Fatalf("双分页断造的空白页应 fail：\n%s", out)
	}
	if !strings.Contains(out, "逐页墨覆盖") || !strings.Contains(out, "已占用版面") {
		t.Fatalf("墨覆盖两口径表应恒出：\n%s", out)
	}
	if strings.Contains(out, "✓ 全部通过") {
		t.Fatalf("有空白页不得全过：\n%s", out)
	}
}
