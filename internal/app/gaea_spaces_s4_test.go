package app

// S4 产物路径分区 + 绑定面（设计 docs/gaea-space-dimension-design.md §5/§6）：
//   - 各 exports 写死点（export/zip/pdf/crosslink/knowledge_meta）缺省 work
//     路径不变（兼容红线）+ play 落 .gaea/play/exports；
//   - preview 白名单加 .gaea/play/exports；
//   - GaeaSpaceList/Active/Activate 三绑定往返 + 非法 space 拒绝；
//   - 任务模板按空间渲染，work 缺省输出逐字不变。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// s4SpaceIsolate 隔离配置/工作目录并把 ga.cfg 指向临时工作区（space 为空 =
// work 缺省），返回工作区路径与恢复函数。
func s4SpaceIsolate(t *testing.T, space string) (string, func()) {
	t.Helper()
	restoreIsolate := workspaceTestIsolate(t)
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	ga.cfg, ga.ctrl = nil, nil
	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	if space != "" {
		ga.cfg.Session.Space = space
	}
	return ws, func() {
		ga.cfg, ga.ctrl = oldCfg, oldCtrl
		restoreIsolate()
	}
}

// ── 导出交付物（gaea_export.go）─────────────────────────────────

func TestExportDeliverableWorkPathUnchanged(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, "")
	defer restore()
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: exportMarkdown, Format: "md"})
	if err != nil {
		t.Fatal(err)
	}
	// 结果路径是工作区相对形态（slash），缺省恒落现状目录
	if !strings.HasPrefix(got.Path, ".gaea/exports/") {
		t.Errorf("缺省导出路径 = %q, want 前缀 .gaea/exports/（现状路径不得改变）", got.Path)
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(got.Path))); err != nil {
		t.Fatalf("缺省导出未落盘: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "play")); !os.IsNotExist(err) {
		t.Errorf("work 缺省不应创建 play 分区目录: %v", err)
	}
}

func TestExportDeliverablePlayPath(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, spaces.SpacePlay)
	defer restore()
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: exportMarkdown, Format: "md"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Path, ".gaea/play/exports/") {
		t.Errorf("play 导出路径 = %q, want 前缀 .gaea/play/exports/", got.Path)
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(got.Path))); err != nil {
		t.Fatalf("play 导出未落盘: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "exports")); !os.IsNotExist(err) {
		t.Errorf("play 空间不应写 work 现状 exports 目录: %v", err)
	}
}

// ── 交付 zip（gaea_deliverable_zip.go）──────────────────────────

func TestZipDeliverablesSpacePaths(t *testing.T) {
	for _, tc := range []struct {
		space    string
		wantRoot string // 工作区相对的 zip 落点根段
	}{
		{"", ".gaea/exports"},
		{spaces.SpacePlay, ".gaea/play/exports"},
	} {
		ws, restore := s4SpaceIsolate(t, tc.space)
		src := filepath.Join(ws, "docs", "报告.docx")
		if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(src, []byte("docx"), 0o644); err != nil {
			t.Fatal(err)
		}
		a := &App{}
		res, err := a.GaeaZipDeliverables([]string{"docs/报告.docx"})
		if err != nil {
			t.Fatalf("space=%q 打包失败: %v", tc.space, err)
		}
		if !strings.HasPrefix(res.Path, tc.wantRoot+"/") {
			t.Errorf("space=%q zip 路径 = %q, want 前缀 %q/", tc.space, res.Path, tc.wantRoot)
		}
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(res.Path))); err != nil {
			t.Fatalf("space=%q zip 未落盘: %v", tc.space, err)
		}
		restore()
	}
}

// ── PDF（gaea_pdf.go）与交叉嵌入（gaea_crosslink.go）────────────
// 测试环境没有 soffice/matplotlib：转换/绘图本身以依赖缺失结束（既有测试
// 同款容错），但 exportsDir 的 MkdirAll 在依赖调用之前执行——目录落点即
// 分区断言点。

func TestPdfAndCrosslinkPlayExportsDir(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, spaces.SpacePlay)
	defer restore()

	// PDF：txt 可直接转换（依赖缺失在 mkdir 之后才发生）
	txt := filepath.Join(ws, ".gaea", "uploads", "doc.txt")
	if err := os.MkdirAll(filepath.Dir(txt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txt, []byte("内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaConvertToPdf(".gaea/uploads/doc.txt"); err == nil {
		t.Log("soffice 可用：PDF 转换成功（目录断言仍成立）")
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "play", "exports")); err != nil {
		t.Errorf("play 空间 PDF 应建 .gaea/play/exports: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "exports")); !os.IsNotExist(err) {
		t.Errorf("play 空间不应建 work 现状 exports 目录: %v", err)
	}

	// 交叉嵌入：合法 xlsx → exportsDir mkdir 后才可能因绘图依赖失败
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "月份")
	f.SetCellValue("Sheet1", "B1", "销售额")
	f.SetCellValue("Sheet1", "A2", "一月")
	f.SetCellValue("Sheet1", "B2", 120) // 自动模式需表头行 + 至少一行数值，否则 ExtractChartData 先失败（mkdir 之前）
	xlsx := filepath.Join(ws, ".gaea", "uploads", "chart.xlsx")
	if err := os.MkdirAll(filepath.Dir(xlsx), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsx); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := a.GaeaCrossEmbed(CrossEmbedInput{XlsxRel: ".gaea/uploads/chart.xlsx", Into: "docx", ChartType: "bar"}); err == nil {
		t.Log("绘图依赖可用：交叉嵌入成功（目录断言仍成立）")
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "play", "exports")); err != nil {
		t.Errorf("play 空间交叉嵌入应建 .gaea/play/exports: %v", err)
	}
}

// TestCrosslinkWorkExportsDirUnchanged 缺省 work：交叉嵌入建现状目录。
func TestCrosslinkWorkExportsDirUnchanged(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, "")
	defer restore()
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "月份")
	f.SetCellValue("Sheet1", "B1", "销售额")
	f.SetCellValue("Sheet1", "A2", "一月")
	f.SetCellValue("Sheet1", "B2", 120) // 自动模式需表头行 + 至少一行数值，否则 ExtractChartData 先失败（mkdir 之前）
	xlsx := filepath.Join(ws, ".gaea", "uploads", "chart.xlsx")
	if err := os.MkdirAll(filepath.Dir(xlsx), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsx); err != nil {
		t.Fatal(err)
	}
	f.Close()
	a := &App{}
	if _, err := a.GaeaCrossEmbed(CrossEmbedInput{XlsxRel: ".gaea/uploads/chart.xlsx", Into: "docx"}); err == nil {
		t.Log("绘图依赖可用：交叉嵌入成功")
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "exports")); err != nil {
		t.Errorf("缺省 work 交叉嵌入应建现状 .gaea/exports: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "play")); !os.IsNotExist(err) {
		t.Errorf("缺省 work 不应建 play 分区目录: %v", err)
	}
}

// ── 知识导出（gaea_knowledge_meta.go）：knowledge 属 work 固定 ──

func TestKnowledgeExportAlwaysWorkDir(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, spaces.SpacePlay)
	// 先关闭隔离 APPDATA 下的 Hephaestus.db 单例（Windows 文件锁会卡住
	// t.TempDir 清理），再恢复环境——顺序反了会关到真实用户目录的句柄。
	defer func() {
		_ = db.CloseDatabase(gaeaConfig.MemoryUserDir())
		restore()
	}()
	a := &App{}
	n, err := a.GaeaKnowledgeExport("")
	if err != nil {
		t.Fatalf("知识导出失败: %v", err)
	}
	t.Logf("知识导出 %d 条（空库可为 0）", n)
	want := filepath.Join(ws, ".gaea", "exports", "knowledge-"+time.Now().Format("20060102"))
	if _, err := os.Stat(want); err != nil {
		t.Errorf("knowledge 导出目录应固定在 work 现状路径 %s: %v", want, err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "play")); !os.IsNotExist(err) {
		t.Errorf("knowledge 导出不随会话空间分区（play 目录不应创建）: %v", err)
	}
}

// ── 预览白名单（gaea_preview.go）───────────────────────────────

func TestPreviewSearchDirsIncludePlayExports(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, "")
	defer restore()
	playDir := filepath.Join(ws, ".gaea", "play", "exports")
	if err := os.MkdirAll(playDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playDir, "成本测算.xlsx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, _ := resolvePreviewPath("成本测算.xlsx")
	if want := filepath.Join(playDir, "成本测算.xlsx"); found != want {
		t.Errorf("裸文件名解析 = %q, want play 分区产物 %q", found, want)
	}
}

// ── GaeaSpace* 三绑定（CoreB）──────────────────────────────────

func TestGaeaSpaceBindingsRoundTrip(t *testing.T) {
	_, restore := s4SpaceIsolate(t, "")
	defer restore()
	a := &App{}

	// List：work/play 静态枚举（顺序稳定）
	opts := a.GaeaSpaceList()
	if len(opts) != 2 || opts[0].ID != spaces.SpaceWork || opts[1].ID != spaces.SpacePlay {
		t.Fatalf("GaeaSpaceList = %+v, want [work play]", opts)
	}

	// Active：缺省 work（现状路径落点）
	v := a.GaeaSpaceActive()
	if v.Space != spaces.SpaceWork || !v.ModeOn {
		t.Fatalf("GaeaSpaceActive = %+v, want work/on", v)
	}
	if v.ExportsDir != ".gaea/exports" || v.WorkDir != ".gaea/work" {
		t.Fatalf("缺省落点 = %+v, want 现状路径", v)
	}

	// 非法 space 拒绝（严格小写，不做大小写归一）
	for _, bad := range []string{"", "Bogus", "Play", "PLAY", "session", "sub"} {
		if _, err := a.GaeaSpaceActivate(bad); err == nil {
			t.Errorf("GaeaSpaceActivate(%q) 应拒绝", bad)
		}
	}

	// 往返：激活 play → 视图 + 磁盘持久化 + 读取端一致
	v2, err := a.GaeaSpaceActivate(spaces.SpacePlay)
	if err != nil {
		t.Fatalf("GaeaSpaceActivate(play): %v", err)
	}
	if v2.Space != spaces.SpacePlay || v2.ExportsDir != ".gaea/play/exports" || v2.WorkDir != ".gaea/play/work" {
		t.Fatalf("激活 play 后视图 = %+v", v2)
	}
	if got := a.GaeaSpaceActive(); got.Space != spaces.SpacePlay {
		t.Fatalf("激活后 Active = %+v, want play", got)
	}
	disk, err := gaeaLoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := disk.SessionSpace(); got != spaces.SpacePlay {
		t.Fatalf("磁盘配置 session.space = %q, want play（持久化缺失）", got)
	}

	// 往返回 work
	v3, err := a.GaeaSpaceActivate(spaces.SpaceWork)
	if err != nil || v3.Space != spaces.SpaceWork {
		t.Fatalf("GaeaSpaceActivate(work) = %+v, %v", v3, err)
	}
}

// TestGaeaSpaceActiveModeOff mode=off：分区整体关闭，视图恒报 work + 现状
// 落点（即使 session.space 配置为 play），产物路径单点同样回退。
func TestGaeaSpaceActiveModeOff(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, spaces.SpacePlay)
	defer restore()
	ga.mu.Lock()
	ga.cfg.Space = gaeaConfig.SpaceConfig{Mode: "off"}
	ga.mu.Unlock()

	if got := gaeaEffectiveSpace(); got != "" {
		t.Fatalf("mode=off gaeaEffectiveSpace = %q, want 空（整体回退）", got)
	}
	v := (&App{}).GaeaSpaceActive()
	if v.Space != spaces.SpaceWork || v.ModeOn {
		t.Fatalf("mode=off Active = %+v, want work/off", v)
	}
	if v.ExportsDir != ".gaea/exports" {
		t.Fatalf("mode=off ExportsDir = %q, want 现状路径", v.ExportsDir)
	}
	if got := spaces.ExportsDir(ws, gaeaEffectiveSpace()); got != filepath.Join(ws, ".gaea", "exports") {
		t.Fatalf("mode=off ExportsDir(cwd, eff) = %q, want 现状路径", got)
	}
}

// ── 任务模板空间渲染（gaea_templates.go）───────────────────────

func TestTaskTemplatesSpaceRender(t *testing.T) {
	// work 缺省：与数据源逐字一致（测试锚定）
	for _, space := range []string{"", spaces.SpaceWork, "bogus"} {
		got := renderTaskTemplates(space)
		if len(got) != len(taskTemplates) {
			t.Fatalf("renderTaskTemplates(%q) 长度 = %d, want %d", space, len(got), len(taskTemplates))
		}
		for i := range got {
			if got[i].Prompt != taskTemplates[i].Prompt || got[i].Name != taskTemplates[i].Name {
				t.Errorf("renderTaskTemplates(%q)[%d] 与数据源不一致（缺省必须逐字不变）", space, i)
			}
		}
	}
	// play：根段替换，其余逐字不变
	play := renderTaskTemplates(spaces.SpacePlay)
	playCount := 0
	for i := range play {
		if strings.Contains(play[i].Prompt, templateExportsRootWork) {
			t.Errorf("play 模板 %s 仍含现状根段", play[i].Name)
		}
		playCount += strings.Count(play[i].Prompt, templateExportsRootPlay)
		want := strings.ReplaceAll(taskTemplates[i].Prompt, templateExportsRootWork, templateExportsRootPlay)
		if play[i].Prompt != want {
			t.Errorf("play 模板 %s 除根段外被改动", play[i].Name)
		}
	}
	if playCount == 0 {
		t.Fatal("play 渲染未发生任何根段替换（数据源可能已不含 .gaea/exports/，需同步测试）")
	}
}

// TestGaeaTaskTemplatesFollowsSpace 绑定入口按当前生效空间渲染。
func TestGaeaTaskTemplatesFollowsSpace(t *testing.T) {
	_, restore := s4SpaceIsolate(t, spaces.SpacePlay)
	defer restore()
	for _, tm := range (&App{}).GaeaTaskTemplates() {
		if strings.Contains(tm.Prompt, ".gaea/exports/") {
			t.Errorf("play 空间模板 %s 仍含现状根段: %.60s", tm.Name, tm.Prompt)
		}
	}
}
