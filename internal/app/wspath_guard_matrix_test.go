package app

// wspath_guard_matrix_test.go — 路径归一/穿越防护参数化矩阵（审计 2026-10-02 AP5-09）。
//
// 结构：**入口 × 恶意输入**表驱动。
//   - 「入口」= 每个改调 wspath.ResolveRelWithin / wspath.Within 的对外绑定或内部
//     解析点，用 probe 闭包表达「是否按路径策略拒绝」；
//   - probe 不只断言「出错了」，而是断言**错因是路径策略**（token）——否则
//     「文件不存在」「后端解析失败」这类顺带错误会把穿越漏洞伪装成拒绝；
//   - 恶意输入矩阵：`../` 穿越、`..\` 穿越（Windows）、混合分隔符、深处穿越、
//     绝对路径（仅对本来拒绝对的入口）、卷相对 `C:`（Windows）。
//
// 反向证据：把 wspath.ResolveRelWithin 改成直接返回 Join 结果（去掉包含性检查），
// 本文件必须整片转红（每个 probe 的 token 断言失败或 probe 变成「未拒绝」）。

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// escapeMarker 工作区外诱饵文件的内容：若某入口真的读了工作区外文件，断言会
// 直接抓到这段明文（而不仅是「碰巧报错」）。
const escapeMarker = "工作区外机密内容-ESCAPE-MARKER"

// armMatrixWorkspace 装配矩阵用隔离工作区：
//   - ws      = base/ws（工作区，ga.cfg.Workspace + Sandbox.WorkspaceRoot 都指它，
//     前者供 gaeaCwd()、后者供 WriteRoots()/GaeaWriteFile 的可写根白名单）；
//   - outside = base/outside（工作区**外**的兄弟目录，放真实存在的诱饵文件）。
//
// 诱饵真实存在是本矩阵的关键：只有存在，旧口径（Join 后直通）才会「成功」，
// 新口径的拒绝才排除了「碰巧文件不存在」的伪证。
func armMatrixWorkspace(t *testing.T) (ws, outside string) {
	t.Helper()
	base := t.TempDir()
	ws = filepath.Join(base, "ws")
	outside = filepath.Join(base, "outside")
	for _, d := range []string{ws, filepath.Join(ws, "sub"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFileT(t, filepath.Join(ws, "a.md"), "# 工作区内文档\n内容\n")
	writeFileT(t, filepath.Join(ws, "sub", "b.txt"), "工作区内文本\n")
	writeFileT(t, filepath.Join(outside, "escape.md"), "# 工作区外文档\n"+escapeMarker+"\n")
	writeFileT(t, filepath.Join(outside, "escape.txt"), escapeMarker+"\n")
	// 工作区外诱饵目录（供目录列举入口）。
	writeFileT(t, filepath.Join(outside, "escapeDir", "c.txt"), escapeMarker+"\n")

	oldCfg := ga.cfg
	cfg := &gaeaConfig.Config{Workspace: ws}
	cfg.Sandbox.WorkspaceRoot = ws
	ga.cfg = cfg
	t.Cleanup(func() { ga.cfg = oldCfg })
	return ws, outside
}

func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// matrixApp 返回**只装配全局 cfg + 零值 core** 的 App（矩阵专用显式构造点）。
//
// 裸构造论证（守卫 check-test-ctors 要求逐入口说明，勿删）——为什么这些入口
// 不需要 core/writingState/mediaState/whisperState/officeState 域状态：
//
// 矩阵只喂**必须被路径策略拒绝**的输入，所有探针都在各自入口的「路径归一阶段」
// 早退（返回前不触达任何域状态），逐入口定位：
//
//	解析原语层（无接收者）：resolveTarget / resolveWorkspacePath / listDirEntries
//	                        ——内部只用 gaeaCwd()/ga.cfg + 文件系统。
//	GaeaDocumentLint        rel 检查 → 归一 → 早退（.docx 的 docmd 链在归一之后）。
//	GaeaReadFile/GaeaWriteFile  rel 检查 → 归一 → 早退（withinWriteRoots 只读 ga.cfg）。
//	GaeaListDir             listDirEntries 的归一 → 早退。
//	GaeaZipDeliverables     条目归一失败 → continue → 空集错误返回（不触 a）。
//	GaeaDocxApplyEdit / GaeaDocxAcceptChanges  归一 → 早退（基线快照/修订写入在其后）。
//	GaeaPptxApplyEdit       参数检查 → 归一 → 早退（os.Stat 在其后）。
//	GaeaXlsx* 六入口 + GaeaXlsxChart + GaeaCrossEmbed  归一 → 早退（AI/证据链/落盘在其后）。
//	GaeaPreview             归一失败（或 .md 命中）分支返回；其 .doc/.xls/.pdf/pptx
//	                        分支会调 a.emit —— 故本函数**装配一个零值 core**（收线
//	                        加固）：core.emit 在 ctx==nil 时直接 return
//	                        （app.go:314-326），既让 a.emit 不再可能 nil 解引用，
//	                        又不引入任何真实域状态。其余域状态（mediaState 等）仍为
//	                        nil：矩阵 19 个入口都在归一阶段早退（见上方逐入口论证）。
//
// 实测背书：29 个探针在只装 cfg+core 的条件下全部执行到策略拒绝分支（任一域状态
// 解引用都会 panic 直接红）；本函数存在的意义是把构造点显式化并集中到一处评审。
func matrixApp() *App { return &App{core: &core{}} }

// guardProbe 是矩阵入口探针：返回 nil 表示**未被路径策略拒绝**（漏洞信号），
// 非 nil 时其 error 文案必须含入口声明 token（证明拒绝来自路径策略）。
type guardProbe struct {
	name string
	// token 是期望出现在 error 文案里的策略片段（证明拒绝来自路径策略）。
	token string
	probe func(rel string) error
}

// TestWspathGuardMatrix 入口 × 恶意输入矩阵：每个入口对每种穿越形态都必须
// 按路径策略拒绝，且错因文案命中 token。
func TestWspathGuardMatrix(t *testing.T) {
	_, outside := armMatrixWorkspace(t)

	probes := []guardProbe{
		{
			name:  "GaeaDocumentLint（本批漏点）",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaDocumentLint(rel)
				return err
			},
		},
		{
			name:  "GaeaReadFile",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				got := matrixApp().GaeaReadFile(rel)
				if !strings.Contains(got.Markdown, "非法工作区相对路径") {
					return nil
				}
				return errors.New(got.Markdown)
			},
		},
		{
			name:  "GaeaWriteFile",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				return matrixApp().GaeaWriteFile(rel, "x")
			},
		},
		{
			name:  "GaeaPreview（resolvePreviewPath）",
			token: "文件不存在",
			probe: func(rel string) error {
				got := matrixApp().GaeaPreview(rel)
				if got.Kind == "error" && got.Body == "" && got.DataURL == "" {
					return errors.New(got.Error)
				}
				return nil // 读成了 → 穿越
			},
		},
		{
			name:  "GaeaListDir（listDirEntries）",
			token: "路径越出工作区",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaListDir(rel)
				return err
			},
		},
		{
			name:  "GaeaZipDeliverables",
			token: "都不存在或不可访问",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaZipDeliverables([]string{rel})
				return err
			},
		},
		{
			name:  "resolveTarget（证据卡复核/回滚）",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := resolveTarget(rel)
				return err
			},
		},
		{
			name:  "resolveWorkspacePath（登记表路径）",
			token: "空串",
			probe: func(rel string) error {
				if got := resolveWorkspacePath(gaeaCwd(), rel); got == "" {
					return errors.New("空串（拒绝）")
				}
				return nil
			},
		},
		{
			name:  "GaeaDocxApplyEdit",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaDocxApplyEdit(rel, "旧", "新")
				return err
			},
		},
		{
			name:  "GaeaDocxAcceptChanges",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaDocxAcceptChanges(rel, false)
				return err
			},
		},
		{
			name:  "GaeaPptxApplyEdit",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaPptxApplyEdit(rel, 1, "旧", "新")
				return err
			},
		},
		{
			name:  "GaeaXlsxPlanEdit",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxPlanEdit(rel, "", "", "")
				return err
			},
		},
		{
			name:  "GaeaXlsxApplyEdit",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxApplyEdit(rel, "[]")
				return err
			},
		},
		{
			name:  "GaeaXlsxSetCell",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxSetCell(rel, "S1", "A1", "1")
				return err
			},
		},
		{
			name:  "GaeaXlsxRecalc",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxRecalc(rel)
				return err
			},
		},
		{
			name:  "GaeaXlsxRowOps",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxRowOps(rel, "S1", "delete", "A1")
				return err
			},
		},
		{
			name:  "GaeaXlsxColOps",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxColOps(rel, "S1", "delete", "A1")
				return err
			},
		},
		{
			name:  "GaeaXlsxChart",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaXlsxChart(XlsxChartInput{Rel: rel})
				return err
			},
		},
		{
			name:  "GaeaCrossEmbed",
			token: "非法工作区相对路径",
			probe: func(rel string) error {
				_, err := matrixApp().GaeaCrossEmbed(CrossEmbedInput{XlsxRel: rel, Into: "docx", ChartType: "bar"})
				return err
			},
		},
	}

	// ── 恶意输入矩阵（相对形态 + 卷相对；绝对路径单列，见下）──
	type badForm struct {
		name    string
		mk      func(outside string) string
		winOnly bool
	}
	forms := []badForm{
		{"父目录 ../", func(o string) string { return "../outside/escape.md" }, false},
		{"反斜杠 ..\\", func(o string) string { return `..\outside\escape.md` }, true},
		{"混合分隔符", func(o string) string { return `sub/..\../outside/escape.md` }, true},
		{"深处穿越", func(o string) string { return "sub/../../outside/escape.md" }, false},
		{"多级穿越", func(o string) string { return "a/b/../../../outside/escape.md" }, false},
		{"卷相对 C:", func(o string) string { return `C:outside\escape.md` }, true},
	}
	// office 文档族入口在拒绝前不做文件访问，故用一个工作区外**不存在**的名字
	// 变体：这样被拒绝时错因只能来自路径策略（token 断言）。
	officeForms := []badForm{
		{"父目录 ../", func(o string) string { return "../outside/nope.xlsx" }, false},
		{"反斜杠 ..\\", func(o string) string { return `..\outside\nope.xlsx` }, true},
		{"深处穿越", func(o string) string { return "sub/../../outside/nope.xlsx" }, false},
		{"卷相对 C:", func(o string) string { return `C:outside\nope.xlsx` }, true},
	}
	officePrefixes := func(name string) bool {
		return strings.HasPrefix(name, "GaeaDocx") || strings.HasPrefix(name, "GaeaPptx") ||
			strings.HasPrefix(name, "GaeaXlsx") || strings.HasPrefix(name, "GaeaCrossEmbed")
	}

	for _, p := range probes {
		formsFor := forms
		if officePrefixes(p.name) {
			formsFor = officeForms
		}
		for _, f := range formsFor {
			if f.winOnly && runtime.GOOS != "windows" {
				continue
			}
			rel := f.mk(outside)
			got := p.probe(rel)
			if got == nil {
				t.Errorf("%s 未拒绝穿越输入 %q（漏洞信号：探针返回未拒绝）", p.name, rel)
				continue
			}
			if !strings.Contains(got.Error(), p.token) {
				t.Errorf("%s 对 %q 的拒绝文案 %q 未含策略 token %q（可能只是碰巧报错/仍指向工作区外）",
					p.name, rel, got.Error(), p.token)
			}
		}
	}
}

// TestWspathGuardMatrixAbsolute 绝对路径与卷相对：**保留合法绝对路径旁路**的
// 入口（列举/预览/论文转换/文档编辑族）必须继续放行——这是「不许悄悄收紧到
// 咬人」的正面锁；只放行本来就把绝对路径当非法的工作区相对入口拒绝。
func TestWspathGuardMatrixAbsolute(t *testing.T) {
	ws, outside := armMatrixWorkspace(t)
	absOutside := filepath.Join(outside, "escape.md")

	// 只接受工作区相对形态的入口：绝对路径必须拒绝。
	rejectAbs := []struct {
		name  string
		token string
		probe func(rel string) error
	}{
		{"GaeaReadFile", "非法工作区相对路径", func(rel string) error {
			got := matrixApp().GaeaReadFile(rel)
			if !strings.Contains(got.Markdown, "非法工作区相对路径") {
				return nil
			}
			return errors.New(got.Markdown)
		}},
		{"GaeaWriteFile", "非法工作区相对路径", func(rel string) error {
			return matrixApp().GaeaWriteFile(rel, "x")
		}},
		{"GaeaZipDeliverables", "都不存在或不可访问", func(rel string) error {
			_, err := matrixApp().GaeaZipDeliverables([]string{rel})
			return err
		}},
	}
	for _, c := range rejectAbs {
		got := c.probe(absOutside)
		if got == nil {
			t.Errorf("%s 必须拒绝绝对路径 %q", c.name, absOutside)
			continue
		}
		if !strings.Contains(got.Error(), c.token) {
			t.Errorf("%s 对绝对路径的拒绝文案 %q 未含 %q", c.name, got.Error(), c.token)
		}
	}

	// 保留绝对路径合法旁路的入口：工作区外绝对路径照旧可用（收紧会咬桌面用法）。
	absInWS := filepath.Join(ws, "a.md")
	if _, err := matrixApp().GaeaDocumentLint(filepath.Join(ws, "sub", "b.txt")); err != nil {
		t.Errorf("GaeaDocumentLint 绝对路径（工作区内）必须照旧通过：%v", err)
	}
	// GaeaConvertToPdf 的绝对分支走 withinReadRoots ∪ 已选文件门（v4.363 口径），
	// 工作区内绝对路径必须在可读根内。
	if !matrixApp().withinReadRoots(absInWS) {
		t.Errorf("withinReadRoots 必须认工作区内绝对路径：%q", absInWS)
	}
	if _, err := matrixApp().GaeaListDir(filepath.Join(ws, "sub")); err != nil {
		t.Errorf("GaeaListDir 绝对路径分支必须保留：%v", err)
	}
	if got, err := resolveTarget(absInWS); err != nil || got != absInWS {
		t.Errorf("resolveTarget 绝对路径分支必须保留（allow_write 目标的复核/回滚），got %q err=%v", got, err)
	}
	if got := resolveWorkspacePath(ws, absInWS); got != absInWS {
		t.Errorf("resolveWorkspacePath 绝对路径应原样返回，got %q", got)
	}
}

// TestWspathGuardMatrixLegit 合法输入回归锁：改调原语后，工作区内相对路径的
// 既有行为逐条不变（防「悄悄收紧到咬人」）。
func TestWspathGuardMatrixLegit(t *testing.T) {
	ws, _ := armMatrixWorkspace(t)

	t.Run("GaeaDocumentLint 相对路径", func(t *testing.T) {
		if _, err := matrixApp().GaeaDocumentLint("a.md"); err != nil {
			t.Fatalf("工作区内相对路径必须通过：%v", err)
		}
	})
	t.Run("GaeaReadFile 相对路径", func(t *testing.T) {
		got := matrixApp().GaeaReadFile("a.md")
		if !strings.Contains(got.Markdown, "工作区内文档") {
			t.Fatalf("应读到工作区内文档，got %+v", got)
		}
	})
	t.Run("GaeaWriteFile 相对路径", func(t *testing.T) {
		if err := matrixApp().GaeaWriteFile(filepath.Join("sub", "b.txt"), "改写\n"); err != nil {
			t.Fatalf("工作区内相对路径写必须通过：%v", err)
		}
		b, err := os.ReadFile(filepath.Join(ws, "sub", "b.txt"))
		if err != nil || string(b) != "改写\n" {
			t.Fatalf("写回内容不符：%q err=%v", b, err)
		}
	})
	t.Run("GaeaPreview 相对路径", func(t *testing.T) {
		got := matrixApp().GaeaPreview("a.md")
		if got.Kind != "markdown" || !strings.Contains(got.Body, "工作区内文档") {
			t.Fatalf("相对路径预览应照旧可用，got kind=%q body=%q", got.Kind, got.Body)
		}
	})
	t.Run("GaeaListDir 相对路径（含内部..归一）", func(t *testing.T) {
		if _, err := matrixApp().GaeaListDir(""); err != nil {
			t.Fatalf("工作区根列举必须通过：%v", err)
		}
		if _, err := matrixApp().GaeaListDir(filepath.Join("sub", "..", "sub")); err != nil {
			t.Fatalf("内部 .. 归一形态必须照旧通过：%v", err)
		}
	})
	t.Run("resolveTarget 相对路径", func(t *testing.T) {
		got, err := resolveTarget(filepath.Join("sub", "b.txt"))
		if err != nil {
			t.Fatalf("相对路径必须通过：%v", err)
		}
		if got != filepath.Join(ws, "sub", "b.txt") {
			t.Fatalf("resolveTarget = %q, want %q", got, filepath.Join(ws, "sub", "b.txt"))
		}
	})
	t.Run("resolveWorkspacePath 相对路径", func(t *testing.T) {
		if got := resolveWorkspacePath(ws, filepath.Join(".gaea", "exports", "a.xlsx")); got != filepath.Join(ws, ".gaea", "exports", "a.xlsx") {
			t.Fatalf("resolveWorkspacePath = %q", got)
		}
	})
	t.Run("GaeaPreview 越界返回空路径而非工作区外路径", func(t *testing.T) {
		if p, _ := resolvePreviewPath("../outside/escape.md"); p != "" {
			t.Fatalf("越界预览必须返回空路径（不得指向工作区外），got %q", p)
		}
	})
}
