package xlsxedit

// recalc_test.go — LibreOffice 重算链（内嵌脚本 + 独立配置目录）的离线单测。
// 脚本调用被替换成桩，不拉 LibreOffice、不需要 python；真拉 LibreOffice 的
// 实弹用例在 xlsxedit_test.go 的 TestSmokeRecalc*（默认跳过）。

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// macroMarker 是宏文件里必然出现的标识：配置目录里有它 = 宏已就位。
const macroMarker = "RecalculateAndSave"

// macroPath 是宏在 LibreOffice 配置目录里的位置（配置目录根下）。
func macroPath(profile string) string {
	return filepath.Join(profile, "user", "basic", "Standard", "Module1.xba")
}

// recalcCall 是一次脚本调用的现场。
type recalcCall struct {
	path    string
	profile string
	copied  bool // 进脚本前配置目录里已有宏（= 从模板复制来的）
}

// recalcSandbox 把重算链的注入口指到临时目录并替换脚本调用，用例结束还原。
type recalcSandbox struct {
	cache string
	tmp   string
	calls []recalcCall
}

func sandboxRecalc(t *testing.T, fn func(c *recalcCall) (RecalcReport, error)) *recalcSandbox {
	t.Helper()
	sb := &recalcSandbox{cache: t.TempDir(), tmp: t.TempDir()}
	oldRoot, oldTmp, oldRun, oldSleep := recalcCacheRootOverride, recalcTempDir, runRecalcScript, recalcSleep
	t.Cleanup(func() {
		recalcCacheRootOverride, recalcTempDir, runRecalcScript, recalcSleep = oldRoot, oldTmp, oldRun, oldSleep
	})
	recalcCacheRootOverride = func() string { return sb.cache }
	recalcTempDir = func() string { return sb.tmp }
	recalcSleep = func(time.Duration) {}
	runRecalcScript = func(path, profile string) (RecalcReport, error) {
		c := recalcCall{path: path, profile: profile}
		if b, err := os.ReadFile(macroPath(profile)); err == nil {
			c.copied = strings.Contains(string(b), macroMarker)
		}
		sb.calls = append(sb.calls, c)
		return fn(&c)
	}
	return sb
}

// okRunner 模拟脚本跑成：全新配置目录里留下宏（LibreOffice 初始化 + 脚本放宏
// 的副产物），正是真实链路里会被留作模板的形态。
func okRunner(c *recalcCall) (RecalcReport, error) {
	if err := os.MkdirAll(filepath.Dir(macroPath(c.profile)), 0o755); err != nil {
		return RecalcReport{}, err
	}
	if err := os.WriteFile(macroPath(c.profile), []byte(macroMarker), 0o644); err != nil {
		return RecalcReport{}, err
	}
	return RecalcReport{Status: "success", TotalFormulas: 1}, nil
}

// seedTemplate 预置一份模板（模拟此前跑成留下的那份）。
func seedTemplate(t *testing.T, body string) {
	t.Helper()
	dir := recalcTemplateDir()
	if err := os.MkdirAll(filepath.Dir(macroPath(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(macroPath(dir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// recalcFixture 造一份「有公式、无缓存值」的工作簿。
func recalcFixture(t *testing.T) string {
	t.Helper()
	path := buildBase(t)
	if _, err := ApplyOps(path, []Op{{Type: "set_formula", Sheet: "数据", Target: "B5", Formula: "SUM(B2:B4)"}}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRecalcDisabledByEnv(t *testing.T) {
	t.Setenv(recalcDisableKey, "1")
	sb := sandboxRecalc(t, okRunner)
	if _, err := Recalc(recalcFixture(t)); !errors.Is(err, ErrRecalcUnavailable) || len(sb.calls) != 0 {
		t.Fatalf("关闭开关下应直接判不可用且不调脚本：err=%v calls=%d", err, len(sb.calls))
	}
}

// TestRecalcTemplateLifecycle 钉住配置模板的来去：首次从零建、跑成留模板；
// 此后每次复制一份独立的、用完即删。
func TestRecalcTemplateLifecycle(t *testing.T) {
	sb := sandboxRecalc(t, okRunner)
	path := recalcFixture(t)

	if _, err := Recalc(path); err != nil {
		t.Fatalf("首次重算失败：%v", err)
	}
	if len(sb.calls) != 1 || sb.calls[0].copied {
		t.Fatalf("首次调用应是全新配置目录：calls=%+v", sb.calls)
	}
	if b, err := os.ReadFile(macroPath(recalcTemplateDir())); err != nil || !strings.Contains(string(b), macroMarker) {
		t.Fatalf("首次跑成后应把配置目录留作模板：err=%v body=%q", err, b)
	}

	if _, err := Recalc(path); err != nil {
		t.Fatalf("第二次重算失败：%v", err)
	}
	if len(sb.calls) != 2 || !sb.calls[1].copied {
		t.Fatalf("第二次调用应复制模板（宏已就位）：calls=%+v", sb.calls)
	}
	if sb.calls[0].profile == sb.calls[1].profile {
		t.Fatalf("每次调用的配置目录必须各自独立：%s", sb.calls[0].profile)
	}
	for _, c := range sb.calls {
		if _, err := os.Stat(c.profile); !os.IsNotExist(err) {
			t.Errorf("用完的配置目录应已删除：%s（err=%v）", c.profile, err)
		}
	}
	if entries, _ := os.ReadDir(sb.tmp); len(entries) != 0 {
		t.Fatalf("临时目录不应留下配置目录残骸：%d 项", len(entries))
	}
}

// TestRecalcRetriesTransientRead 回读类错误是瞬时的：自动重试一次；仍失败才
// 报回读失败，并说明已经重试过。
func TestRecalcRetriesTransientRead(t *testing.T) {
	var sb *recalcSandbox
	sb = sandboxRecalc(t, func(c *recalcCall) (RecalcReport, error) {
		if len(sb.calls) == 1 {
			return RecalcReport{}, fmt.Errorf("%w：Error -3 while decompressing data: invalid distance too far back", ErrRecalcRead)
		}
		return okRunner(c)
	})
	if _, err := Recalc(recalcFixture(t)); err != nil || len(sb.calls) != 2 {
		t.Fatalf("回读失败应重试一次后成功：err=%v calls=%d", err, len(sb.calls))
	}

	sb2 := sandboxRecalc(t, func(*recalcCall) (RecalcReport, error) {
		return RecalcReport{}, fmt.Errorf("%w：boom", ErrRecalcRead)
	})
	_, err := Recalc(recalcFixture(t))
	if !errors.Is(err, ErrRecalcRead) || !strings.Contains(err.Error(), "已自动重试 1 次") || len(sb2.calls) != 2 {
		t.Fatalf("两次回读都失败应如实报回读失败并注明已重试：err=%v calls=%d", err, len(sb2.calls))
	}
}

// TestRecalcNoRetryWhenUnavailableOnFreshProfile 全新配置目录都跑不起来 =
// 环境问题（缺 LibreOffice 等），重试同一环境没有意义。
func TestRecalcNoRetryWhenUnavailableOnFreshProfile(t *testing.T) {
	sb := sandboxRecalc(t, func(*recalcCall) (RecalcReport, error) {
		return RecalcReport{}, fmt.Errorf("%w：LibreOffice recalculation timed out after 60s", ErrRecalcUnavailable)
	})
	_, err := Recalc(recalcFixture(t))
	if !errors.Is(err, ErrRecalcUnavailable) || len(sb.calls) != 1 {
		t.Fatalf("环境不可用不该重试：err=%v calls=%d", err, len(sb.calls))
	}
}

// TestRecalcHealsBrokenTemplate 模板复制件跑不起来（模板损坏 / 版本不配）时
// 丢弃模板、从零重建再试一次——重算不能因为一份坏模板永久不可用。
func TestRecalcHealsBrokenTemplate(t *testing.T) {
	sb := sandboxRecalc(t, func(c *recalcCall) (RecalcReport, error) {
		if c.copied {
			return RecalcReport{}, fmt.Errorf("%w：LibreOffice exited with code 1", ErrRecalcUnavailable)
		}
		return okRunner(c)
	})
	seedTemplate(t, macroMarker+" (but the rest of the profile is broken)")

	if _, err := Recalc(recalcFixture(t)); err != nil {
		t.Fatalf("坏模板应自愈后重算成功：%v", err)
	}
	if len(sb.calls) != 2 || !sb.calls[0].copied || sb.calls[1].copied {
		t.Fatalf("应先是复制件失败、再是全新配置成功：calls=%+v", sb.calls)
	}
	if b, err := os.ReadFile(macroPath(recalcTemplateDir())); err != nil || string(b) != macroMarker {
		t.Fatalf("坏模板应已被重建的替换：err=%v body=%q", err, b)
	}
}

// TestRecalcRejectsNonWorkbook 打不开的文件在入口秒退，不交给 LibreOffice
// （交给它要挂满 60 秒超时）。
func TestRecalcRejectsNonWorkbook(t *testing.T) {
	sb := sandboxRecalc(t, okRunner)
	bad := filepath.Join(t.TempDir(), "bad.xlsx")
	if err := os.WriteFile(bad, []byte("PK\x03\x04 not really a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Recalc(bad)
	if !errors.Is(err, ErrRecalcNotWorkbook) || len(sb.calls) != 0 {
		t.Fatalf("非 zip 文件应判 ErrRecalcNotWorkbook 且不调脚本：err=%v calls=%d", err, len(sb.calls))
	}
}

func TestRemoveStaleRecalcTemplates(t *testing.T) {
	sb := sandboxRecalc(t, okRunner)
	stale := filepath.Join(sb.cache, recalcProfilePrefix+"000000000000")
	other := filepath.Join(sb.cache, "unrelated")
	for _, d := range []string{recalcTemplateDir(), stale, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	removeStaleRecalcTemplates(recalcTemplateDir())
	for d, wantGone := range map[string]bool{recalcTemplateDir(): false, stale: true, other: false} {
		_, err := os.Stat(d)
		if gone := os.IsNotExist(err); gone != wantGone {
			t.Errorf("%s 存在性=%v，期望删除=%v", filepath.Base(d), !gone, wantGone)
		}
	}
}

func TestRecalcCacheRootDefault(t *testing.T) {
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "Local"))
	want := filepath.Join(os.Getenv("LOCALAPPDATA"), "gaea", "lo-recalc")
	if got := RecalcCacheRoot(); got != want {
		t.Fatalf("RecalcCacheRoot = %q，期望 %q", got, want)
	}
	t.Setenv("LOCALAPPDATA", "")
	if got := RecalcCacheRoot(); got != filepath.Join(os.TempDir(), "gaea", "lo-recalc") {
		t.Fatalf("LOCALAPPDATA 缺失应回退系统临时目录：%q", got)
	}
}

// TestParseRecalcOutput 脚本回执 → 报告 / 归类错误。
func TestParseRecalcOutput(t *testing.T) {
	cases := []struct {
		name   string
		out    string
		runErr error
		want   RecalcReport
		wantIs error
		wantIn string
	}{
		{
			name: "成功回执",
			out:  `{"status": "errors_found", "total_errors": 1, "error_summary": {"#REF!": {"count": 1}}, "total_formulas": 7}`,
			want: RecalcReport{Status: "errors_found", TotalErrors: 1, TotalFormulas: 7, ErrorSummary: map[string]interface{}{"#REF!": map[string]interface{}{"count": float64(1)}}},
		},
		{
			name: "JSON 之前混了告警行（stdout/stderr 合并）",
			out: `UserWarning: Workbook contains no default style
{"status": "success", "total_errors": 0, "error_summary": {}, "total_formulas": 3}`,
			want: RecalcReport{Status: "success", TotalFormulas: 3},
		},
		{
			name: "JSON 之后混了告警行",
			out: `{"status": "success", "total_errors": 0, "error_summary": {}, "total_formulas": 4}
UserWarning: something harmless`,
			want: RecalcReport{Status: "success", TotalFormulas: 4},
		},
		{
			name:   "回读阶段失败（宁丰 #1 的 zlib 报错）",
			out:    `{"error": "Error -3 while decompressing data: invalid distance too far back", "stage": "read"}`,
			wantIs: ErrRecalcRead,
		},
		{
			name:   "LibreOffice 超时",
			out:    `{"error": "LibreOffice recalculation timed out after 60s", "stage": "recalc"}`,
			wantIs: ErrRecalcUnavailable,
		},
		{
			name:   "失败回执没带环节（保守归环境问题，不重试）",
			out:    `{"error": "something failed"}`,
			wantIs: ErrRecalcUnavailable,
		},
		{
			name:   "输出不是 JSON",
			out:    `Traceback: nonsense`,
			wantIs: ErrRecalcUnavailable,
		},
		{
			name:   "脚本自己崩了（如缺 openpyxl）",
			out:    `ModuleNotFoundError: No module named 'openpyxl'`,
			runErr: errors.New("exit status 1"),
			wantIs: ErrRecalcUnavailable,
			wantIn: "openpyxl",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := parseRecalcOutput([]byte(tc.out), tc.runErr)
			if tc.wantIs == nil {
				if err != nil {
					t.Fatalf("应成功：%v", err)
				}
				if rep.Status != tc.want.Status || rep.TotalErrors != tc.want.TotalErrors || rep.TotalFormulas != tc.want.TotalFormulas {
					t.Fatalf("报告不符：got %+v want %+v", rep, tc.want)
				}
				return
			}
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("应归类 %v，实际：%v", tc.wantIs, err)
			}
			if tc.wantIn != "" && !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("错误应带脚本输出 %q：%v", tc.wantIn, err)
			}
		})
	}
}

// TestEmbeddedRecalcScript 内嵌脚本与 Go 层的契约对得上，且语法没坏——脚本是
// 字符串，编译期查不了。
func TestEmbeddedRecalcScript(t *testing.T) {
	for _, want := range []string{
		`PROFILE_ENV = "` + recalcProfileKey + `"`,
		"-env:UserInstallation=",
		`"stage": "read"`,
		`"stage": "recalc"`,
		macroMarker,
	} {
		if !strings.Contains(recalcScript, want) {
			t.Fatalf("内嵌脚本缺 %q（Go 与脚本的契约漂移）", want)
		}
	}
	py, err := exec.LookPath("python")
	if err != nil {
		t.Skip("本机无 python，跳过语法检查")
	}
	f := filepath.Join(t.TempDir(), "recalc.py")
	if err := os.WriteFile(f, []byte(recalcScript), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(py, "-c", "import ast,sys; ast.parse(open(sys.argv[1], encoding='utf-8').read())", f).CombinedOutput()
	if err != nil {
		t.Fatalf("内嵌 recalc.py 语法错误：%v\n%s", err, out)
	}
}
