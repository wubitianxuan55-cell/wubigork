package app

// xlsx_occupy_ab_test.go — action-2 打开中 xlsx 观察 A/B（众神殿
// pan_1795058214142 裁决，用户签注 accepted；依赖 action-1 R2 已落地）。
//
// 门控：GAEA_XLSX_OCCUPY_AB=1 才跑（真 Excel/WPS COM + 真工具层，分钟级；
// CI/缺省恒 skip，同 GAEA_AB_EVAL/PANTHEON_LIVE_SMOKE 纪律）。
// 可选：GAEA_XLSX_OCCUPY_AB_ARMS=closed,excel,wps 选臂（缺省全跑）；
//       GAEA_XLSX_OCCUPY_AB_OUT 指定产出目录（缺省 .tmp/xlsx-occupy-ab-<ts>/）。
//
// 臂（每臂 ≥3 次，裁决验收口径）：
//   closed  = 文件关闭态 xlsx_apply（对照，验证操作集本身可落盘）
//   excel   = Excel.Application 打开中 xlsx_apply
//   wps     = Ket.Application（WPS 表格）打开中 xlsx_apply
//
// 每次运行采集：回执/错误原文（判定谓词 P-a 占用归因、P-b 指路）＋落盘前后
// sha256＋事后 excelize 重开可解析性（半写金标=改坏/截断即立即升级 P1 补丁，
// 裁决 risks #3）＋基线快照是否生成。夹具全用可丢弃临时目录（裁决风险条款），
// COM 实例由专用 powershell 进程持有（marker 文件握手收尾），超时杀树不碰
// 用户自己的 Office 进程。

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/xuri/excelize/v2"
)

type occupyRun struct {
	Arm       string // closed / excel / wps
	Run       int
	Receipt   string // 工具回执或错误原文（逐字）
	ErrText   string
	ShaBefore string
	ShaAfter  string
	SizeAfter int64
	ReopenOK  bool
	Outcome   string // success / clean-fail / half-write / open-failed
	Baseline  bool   // .gaea/work/rollback 基线快照是否生成
}

func TestXlsxOccupyABLive(t *testing.T) {
	if os.Getenv("GAEA_XLSX_OCCUPY_AB") == "" {
		t.Skip("设 GAEA_XLSX_OCCUPY_AB=1 跑打开中 xlsx 占用 A/B（真 Excel/WPS，分钟级）")
	}
	arms := []string{"closed", "excel", "wps"}
	if v := os.Getenv("GAEA_XLSX_OCCUPY_AB_ARMS"); v != "" {
		arms = strings.Split(v, ",")
	}
	outDir := os.Getenv("GAEA_XLSX_OCCUPY_AB_OUT")
	if outDir == "" {
		outDir = filepath.Join(".tmp", fmt.Sprintf("xlsx-occupy-ab-%d", time.Now().Unix()))
	}
	// 先转绝对：臂内 t.Chdir 到临时夹具目录后相对路径会指错地方。
	if abs, aerr := filepath.Abs(outDir); aerr == nil {
		outDir = abs
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var runs []occupyRun
	for _, arm := range arms {
		for i := 1; i <= 3; i++ {
			r := runOccupyArm(t, arm, i)
			runs = append(runs, r)
			t.Logf("[%s #%d] outcome=%s receipt=%.300s", arm, i, r.Outcome, r.Receipt+r.ErrText)
		}
	}
	writeOccupyReport(t, outDir, runs)
	summarizeOccupy(t, runs)
}

// runOccupyArm 一次运行：可丢弃夹具 →（按臂）COM 持锁 → 工具层全链 → 采集。
func runOccupyArm(t *testing.T, arm string, run int) occupyRun {
	t.Helper()
	ws := t.TempDir()
	t.Chdir(ws) // gaeaCwd 跟随（同 gaea_xlsx_edit_test 先例）
	rel := filepath.ToSlash(filepath.Join(".gaea", "uploads", fmt.Sprintf("ab-occupy-%s-%d.xlsx", arm, run)))
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "项目")
	f.SetCellValue("Sheet1", "B1", "数量")
	f.SetCellValue("Sheet1", "C1", "单价")
	f.SetCellValue("Sheet1", "A2", "甲")
	f.SetCellValue("Sheet1", "B2", 3)
	f.SetCellValue("Sheet1", "C2", 10)
	f.SetCellValue("Sheet1", "A3", "乙")
	f.SetCellValue("Sheet1", "B3", 5)
	f.SetCellValue("Sheet1", "C3", 8)
	f.SetCellFormula("Sheet1", "D2", "SUM(B2:C2)")
	if err := f.SaveAs(filepath.FromSlash(rel)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	path := filepath.FromSlash(rel)
	before := fileSha(t, path)

	r := occupyRun{Arm: arm, Run: run, ShaBefore: before}

	// 打开臂：专用 powershell 进程持 COM 锁（marker 握手），结束必收尾。
	if arm != "closed" {
		progID := "Excel.Application"
		if arm == "wps" {
			progID = "Ket.Application"
		}
		abs, aerr := filepath.Abs(path)
		if aerr != nil {
			t.Fatal(aerr)
		}
		release, ok := holdWorkbookViaCOM(t, progID, abs)
		if !ok {
			r.Outcome = "open-failed"
			return r
		}
		defer release()
	}

	// 工具层全链（agent 工具面，非内部直调——回执形态与模型所见一致）。
	args, _ := json.Marshal(map[string]string{
		"path":     rel,
		"ops_json": `[{"type":"set_value","sheet":"Sheet1","target":"E2","value":"AB合计"}]`,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := (officeXlsxApplyTool{a: &App{core: &core{}}}).Execute(ctx, args)
	r.Receipt, r.ErrText = out, ""
	if err != nil {
		r.ErrText = err.Error()
	}

	r.ShaAfter = fileSha(t, path)
	if fi, serr := os.Stat(path); serr == nil {
		r.SizeAfter = fi.Size()
	}
	if fr, oerr := excelize.OpenFile(path); oerr == nil {
		r.ReopenOK = true
		fr.Close()
	}
	// 基线快照（xlsx-<ns>.before）是否落 rollback 目录。
	if matches, _ := filepath.Glob(filepath.Join(".gaea", "work", "rollback", "xlsx-*.before")); len(matches) > 0 {
		r.Baseline = true
	}

	switch {
	case err == nil && r.ShaAfter != r.ShaBefore && r.ReopenOK:
		r.Outcome = "success"
	case err != nil && r.ShaAfter == r.ShaBefore && r.ReopenOK:
		r.Outcome = "clean-fail" // 失败但文件原样未动
	case err != nil && (!r.ReopenOK || r.SizeAfter == 0):
		r.Outcome = "half-write" // 反例金标：写坏/截断 → 裁决=立即升级 P1
	case err == nil && r.ShaAfter == r.ShaBefore:
		r.Outcome = "success-nochange" // 声称成功但未改（异常态，如实记）
	default:
		r.Outcome = "other"
	}
	return r
}

// holdWorkbookViaCOM 起一个 powershell 进程用 COM 打开工作簿并保持，返回
// release 收尾函数。握手：stdout 出 OPENED 后才算持锁；Go 侧写 marker 文件
// 通知收尾；超时 taskkill /T 杀该 powershell 树（不碰用户自己的 Office）。
func holdWorkbookViaCOM(t *testing.T, progID, path string) (release func(), ok bool) {
	t.Helper()
	script := filepath.Join(t.TempDir(), "hold.ps1")
	marker := filepath.Join(t.TempDir(), "release.marker")
	ps := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
try {
  $app = New-Object -ComObject %s
  $app.Visible = $false
  try { $app.DisplayAlerts = $false } catch {}
  $wb = $app.Workbooks.Open('%s')
  Write-Output 'OPENED'
  while (-not (Test-Path '%s')) { Start-Sleep -Milliseconds 200 }
  try { $wb.Close($false) } catch {}
  try { $app.Quit() } catch {}
  [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($wb)
  [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($app)
  Write-Output 'CLOSED'
} catch {
  Write-Output ("OPEN-ERR: " + $_.Exception.Message)
}
`, progID, path, marker)
	if err := os.WriteFile(script, []byte(ps), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script)
	proc.HideWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Logf("%s COM 启动失败: %v", progID, err)
		return func() {}, false
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()

	lineCh := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lineCh <- sc.Text()
		}
	}()
	select {
	case line := <-lineCh:
		if !strings.HasPrefix(line, "OPENED") {
			t.Logf("%s 打开失败: %s", progID, line)
			_ = os.WriteFile(marker, []byte{1}, 0o644)
			<-done
			return func() {}, false
		}
	case <-time.After(45 * time.Second):
		t.Logf("%s 打开超时", progID)
		_ = cmd.Process.Kill()
		<-done
		return func() {}, false
	}
	time.Sleep(500 * time.Millisecond) // 等 Excel 把文件锁吃稳

	release = func() {
		_ = os.WriteFile(marker, []byte{1}, 0o644)
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			kill := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid))
			proc.HideWindow(kill)
			_ = kill.Run()
			<-done
			t.Logf("%s COM 收尾超时，已杀 powershell 树 PID=%d", progID, cmd.Process.Pid)
		}
	}
	return release, true
}

func sha12(s string) string {
	if len(s) < 12 {
		return strings.TrimSpace(s)
	}
	return s[:12]
}

func fileSha(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeOccupyReport(t *testing.T, outDir string, runs []occupyRun) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "# 打开中 xlsx 占用 A/B 报告（action-2，pan_1795058214142）\n\n")
	fmt.Fprintf(&b, "- 时间：%s\n- 载体：officeXlsxApplyTool 全链（agent 工具面）\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "- 操作集：set_value E2（单点写）\n\n")
	for _, r := range runs {
		fmt.Fprintf(&b, "## [%s #%d] %s\n\n", r.Arm, r.Run, r.Outcome)
		if r.ErrText != "" {
			fmt.Fprintf(&b, "- 错误原文：`%s`\n", r.ErrText)
		} else {
			fmt.Fprintf(&b, "- 回执原文：`%s`\n", r.Receipt)
		}
		fmt.Fprintf(&b, "- sha before/after：%s / %s（%d bytes after）\n", sha12(r.ShaBefore), sha12(r.ShaAfter), r.SizeAfter)
		fmt.Fprintf(&b, "- 重开可解析：%v；基线快照：%v\n\n", r.ReopenOK, r.Baseline)
	}
	path := filepath.Join(outDir, "report.md")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.MarshalIndent(runs, "", " ")
	_ = os.WriteFile(filepath.Join(outDir, "runs.json"), raw, 0o644)
	t.Logf("报告落档：%s", path)
}

// summarizeOccupy 判定谓词快报（P-a 占用归因文案/P-b 指路；P-c 模型两步内
// 动作不属本层，由实弹或 abeval 补）。
func summarizeOccupy(t *testing.T, runs []occupyRun) {
	t.Helper()
	counts := map[string]int{}
	for _, r := range runs {
		counts[r.Arm+"/"+r.Outcome]++
	}
	for k, v := range counts {
		t.Logf("汇总：%s × %d", k, v)
	}
	for _, r := range runs {
		if r.Outcome == "half-write" {
			t.Errorf("反例金标命中：[%s #%d] 半写/写坏——按裁决立即升级 P1 补丁", r.Arm, r.Run)
		}
	}
	var failTexts []string
	for _, r := range runs {
		if r.ErrText != "" && r.Outcome == "clean-fail" {
			failTexts = append(failTexts, r.ErrText)
		}
	}
	if len(failTexts) > 0 {
		joined := strings.Join(failTexts, " | ")
		pa := strings.Contains(joined, "used by another process") || strings.Contains(joined, "被另一") || strings.Contains(joined, "占用")
		t.Logf("P-a（失败文案含占用归因）：%v", pa)
		t.Logf("P-b（失败文案指路 office_open/关闭后重试）：%v", strings.Contains(joined, "office_open"))
	}
}
