package whisper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── 线3（IN4-05 / IN4-08 现场复核）正式回归用例 ─────────────────────
//
// 1) 关闭进程黑名单的通配符绕过（判据侧 + 执行侧双侧同源）
// 2) 路径前缀判据的形态归一（\\?\ / UNC / 正斜杠 / 尾随空格点）
// 3) copy_path / move_path 缺 pathTo 的静默落点
// 4) download_and_install 的策略盲区与「已确认」纵深防御

// TestIsBlockedCloseTarget_WildcardBypass 黑名单必须拦住通配符与路径形态。
// 实测（2026-10-02）：修复前 `explorer*`/`*` 在判据侧 blockedByMap=false，
// 而执行侧 `Get-Process -Name '*'` 返回本机全部进程名（通配符确实展开）。
func TestIsBlockedCloseTarget_WildcardBypass(t *testing.T) {
	blocked := []string{
		"explorer.exe", "EXPLORER.EXE", "Lsass", " services",
		// 通配符形态（修复前全部不命中）
		"explorer*", "EXPLORER*", "*", "explo?er", "explo?er.exe",
		"explorer*.exe", "*e", "[a-z]xplorer", "explorer[.]exe",
		// 路径形态（修复前不命中）
		`C:\Windows\explorer.exe`, `C:\Windows\System32\lsass.exe`, `explorer* `,
	}
	for _, target := range blocked {
		if !isBlockedCloseTarget(target) {
			t.Errorf("%q 应被判定为禁止关闭（通配符/路径形态必须拒绝）", target)
		}
	}
	safe := []string{"chrome.exe", "notepad", "myapp.exe", "", "   ", "code", "wechat"}
	for _, target := range safe {
		if isBlockedCloseTarget(target) {
			t.Errorf("%q 不应被禁止（不得过度收紧）", target)
		}
	}
}

// TestCloseAppTarget_RejectsWildcard 执行侧必须与判据侧同源：不再把用户可控名字
// 交给支持通配符的接口。含通配符/路径分隔符的名字在执行前即被拒绝（不拉起 PowerShell）。
func TestCloseAppTarget_RejectsWildcard(t *testing.T) {
	for _, target := range []string{"*", "explorer*", "EXPLORER*", "explo?er", "[a-z]xplorer", `C:\Windows\explorer.exe`, ""} {
		res := closeAppTarget(target)
		if res.OK {
			t.Errorf("closeAppTarget(%q) 不应成功: %+v", target, res)
		}
		if res.Summary == "" {
			t.Errorf("closeAppTarget(%q) 应给出明确原因", target)
		}
	}
	// 不存在的普通进程名：走到 PowerShell 但找不到进程 → 失败（证明未被形状拒绝误伤）
	if res := closeAppTarget("zz_no_such_process_name"); res.OK {
		t.Errorf("不存在的进程不应成功: %+v", res)
	}
}

// TestExecuteUseComputer_WildcardCloseBlocked 全链路：通配符目标在第 2 层被拦，
// 审计留痕为 blocked，且执行侧从未被调用。
func TestExecuteUseComputer_WildcardCloseBlocked(t *testing.T) {
	for _, target := range []string{"explorer*", "*", "explo?er", `C:\Windows\explorer.exe`} {
		dataRoot := t.TempDir()
		ctx := RouterContext{DataRoot: dataRoot, AllowAppControl: true}
		res := ExecuteUseComputer(UseComputerArgs{Action: ActionCloseApp, Target: target}, ctx)
		if res.Success {
			t.Errorf("target=%q 应被拦截: %+v", target, res)
		}
		entries, err := ReadAuditEntriesSince(dataRoot, "2020-01-01T00:00:00Z")
		if err != nil || len(entries) != 1 {
			t.Fatalf("target=%q 应有 1 条审计: %v err=%v", target, entries, err)
		}
		if entries[0].Result != "blocked" {
			t.Errorf("target=%q 审计结果应为 blocked, got %q", target, entries[0].Result)
		}
	}
}

// TestPathShapeCanonicalizationIsHardBlocked \\?\ / \\.\ / UNC / 正斜杠 / 尾随空格点
// 形态必须在判据侧归一到同一个「盘符 + 反斜杠」形态后命中硬阻断。
func TestPathShapeCanonicalizationIsHardBlocked(t *testing.T) {
	blockedForms := []string{
		`C:\Windows\System32\x.dll`,
		`c:\windows\system32\x.dll`,
		`C:\WINDOWS\SYSTEM32\x.dll`,
		`C:/Windows/System32/x.dll`,
		`\\?\C:\Windows\System32\x.dll`,
		`\\?\c:\windows\system32\x.dll`,
		`\\?\C:/Windows/System32/x.dll`,
		`\\.\C:\Windows\System32\x.dll`,
		`\\localhost\C$\Windows\System32\x.dll`,
		`C:\Windows\System32\ `,
		`C:\Windows\System32.`,
		`C:\Windows\SysWOW64\x.dll`,
		`\\?\C:\Windows\System32\..\System32\x.dll`,
	}
	for _, p := range blockedForms {
		// 判据侧（不依赖 normalizePath，任何调用方传入的形态都要能判）
		if !isHardBlockedWritePath(p) {
			t.Errorf("isHardBlockedWritePath(%q) 应为 true", p)
		}
		// 策略层全链路（写入动作）
		if got := evaluatePathPolicy(ActionWriteText, p, "", `C:\`); got.OK {
			t.Errorf("evaluatePathPolicy(write_text, %q) 应硬阻断, got %+v", p, got)
		}
		// 敏感告警（读类也应有告警）
		if !isSensitivePath(p) {
			t.Errorf("isSensitivePath(%q) 应为 true（系统目录应至少告警）", p)
		}
	}
}

// TestPathPrefixBoundaryNotOverTightened 边界用例：不得把 `System32foo` 当成
// System32，也不得把既有设计允许的 `C:\Windows\Temp` 变成硬阻断
// （desktop_test.go:79-82 钉死了「允许 + 敏感告警」）。
func TestPathPrefixBoundaryNotOverTightened(t *testing.T) {
	if isHardBlockedWritePath(`C:\Windows\System32foo\x.dll`) {
		t.Error("System32foo 落点不在 System32 内，不应硬阻断")
	}
	if isHardBlockedWritePath(`C:\Windows\Temp\x.txt`) {
		t.Error(`C:\Windows\Temp 属既有设计「允许 + 敏感告警」，不应硬阻断`)
	}
	got := evaluatePathPolicy(ActionWriteText, `C:\Windows\Temp\x.txt`, "", "")
	if !got.OK || got.SensitiveWarning == "" {
		t.Errorf(`C:\Windows\Temp 应放行但给敏感告警: %+v`, got)
	}
	if !isSensitivePath(`C:\Program Files (x86)\App\x.dll`) {
		t.Error("Program Files (x86) 应判为敏感目录")
	}
	if isSensitivePath(`C:\Users\u\Desktop\x.txt`) {
		t.Error("用户目录不应判为敏感目录")
	}
}

// TestEvaluatePathPolicy_CopyMoveMissingPathTo copy/move 缺 pathTo 必须硬阻断：
// 修复前 OK=true 且 normalizedPathTo=""，执行侧 filepath.Dir("")=="." 会把文件
// 静默写到进程当前目录。
func TestEvaluatePathPolicy_CopyMoveMissingPathTo(t *testing.T) {
	src := `C:\Users\u\docs\a.txt`
	for _, action := range []DesktopAgentAction{ActionCopyPath, ActionMovePath} {
		got := evaluatePathPolicy(action, src, "", `C:\Users\u\docs`)
		if got.OK {
			t.Errorf("%s 缺 pathTo 应硬阻断: %+v", action, got)
		}
		if !strings.Contains(got.HardBlockReason, "目标路径") {
			t.Errorf("%s 阻断原因应指明缺少目标路径, got %q", action, got.HardBlockReason)
		}
		// 只给空白的 pathTo 同样阻断
		if got := evaluatePathPolicy(action, src, "   ", `C:\Users\u\docs`); got.OK {
			t.Errorf("%s 空白 pathTo 应硬阻断: %+v", action, got)
		}
		// 正常 pathTo 不误伤
		if got := evaluatePathPolicy(action, src, `C:\Users\u\docs\b.txt`, `C:\Users\u\docs`); !got.OK {
			t.Errorf("%s 正常 pathTo 应放行: %+v", action, got)
		}
	}
}

// TestEvaluatePathPolicy_DownloadSink 下载类动作的落点同样受系统目录硬阻断约束
// （修复前 ActionDownloadAndInstall 走 AppActions 早退，实测 OK=true 且无告警）。
func TestEvaluatePathPolicy_DownloadSink(t *testing.T) {
	for _, p := range []string{
		`C:\Windows\System32\evil.exe`,
		`\\?\C:\Windows\System32\evil.exe`,
		`\\localhost\C$\Windows\System32\evil.exe`,
	} {
		got := evaluatePathPolicy(ActionDownloadAndInstall, p, "", "")
		if got.OK {
			t.Errorf("download_and_install 落点 %q 应硬阻断, got %+v", p, got)
		}
		if !strings.Contains(got.HardBlockReason, "系统目录") {
			t.Errorf("阻断原因应指明系统目录, got %q", got.HardBlockReason)
		}
	}
	// 默认下载目录（AppData）不得被误拦
	home, _ := os.UserHomeDir()
	def := filepath.Join(home, "Downloads", "LightWhisperDownloads", "x.exe")
	if got := evaluatePathPolicy(ActionDownloadAndInstall, def, "", ""); !got.OK {
		t.Errorf("默认下载目录不应被硬阻断: %+v", got)
	}
}

// TestExecuteDesktopAgentAction_InstallRequiresConfirm 执行器纵深防御：
// 未经确认层批准的下载并安装 / 运行安装包必须被拒绝（且不发起下载）。
func TestExecuteDesktopAgentAction_InstallRequiresConfirm(t *testing.T) {
	dir := t.TempDir()
	res := ExecuteDesktopAgentAction(ActionDownloadAndInstall, "", "", "", "", "https://example.invalid/x.exe", "", DesktopExecContext{DownloadDir: dir})
	if res.OK || !strings.Contains(res.Content, "确认") {
		t.Errorf("未确认的 download_and_install 应被拒绝: %+v", res)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("未确认时不应产生任何下载物: %v", entries)
	}
	res = ExecuteDesktopAgentAction(ActionRunInstaller, filepath.Join(dir, "x.exe"), "", "", "", "", "", DesktopExecContext{})
	if res.OK || !strings.Contains(res.Content, "确认") {
		t.Errorf("未确认的 run_installer 应被拒绝: %+v", res)
	}
}

// TestExecuteToolBatch_RouterNilHonestReceipt Router 未接线时过去什么都不追加：
// 模型拿到空结果集 → shouldContinue=false → 直接 AllPassed 收尾，用户侧看不到
// 「动作根本没执行」。本用例钉死诚实回执（反向证据：删掉 else 分支即变红）。
func TestExecuteToolBatch_RouterNilHonestReceipt(t *testing.T) {
	r := &AgentLoopRunner{MaxRounds: 1} // Router == nil
	results := r.executeToolBatch([]AgentAction{{Name: "use_computer", Args: map[string]string{"action": "list_folder", "path": `C:\`}}})
	if len(results) != 1 {
		t.Fatalf("Router==nil 必须产出 1 条诚实回执，got %d 条: %+v", len(results), results)
	}
	if results[0].Name != "use_computer" {
		t.Errorf("回执必须走既有 use_computer 通道（不新造通道），got %q", results[0].Name)
	}
	for _, want := range []string{"未启用", "未执行", "list_folder"} {
		if !strings.Contains(results[0].Content, want) {
			t.Errorf("回执应含 %q，got %q", want, results[0].Content)
		}
	}
	// 接线后仍是原有结果（不得把正常路径也降级）
	r2 := &AgentLoopRunner{MaxRounds: 1, Router: &RouterContext{DataRoot: t.TempDir(), AllowFileWrite: true}}
	res2 := r2.executeToolBatch([]AgentAction{{Name: "use_computer", Args: map[string]string{"action": "list_folder", "path": t.TempDir()}}})
	if len(res2) != 1 || strings.Contains(res2[0].Content, "未启用") {
		t.Errorf("Router 非空时应执行真实动作，got %+v", res2)
	}
}

// TestExecuteUseComputer_ConfirmedReachesExecutor 生产唯一入口必须把「已过确认层」
// 传给执行器，否则上一条的纵深防御会把正常链路也挡掉。
func TestExecuteUseComputer_ConfirmedReachesExecutor(t *testing.T) {
	dir := t.TempDir()
	ctx := RouterContext{
		DataRoot: t.TempDir(), AllowInstall: true, AllowDownload: true,
		DownloadDir: dir, CWD: t.TempDir(),
		RequestConfirm: func(actionLabel, path, target, sensitiveWarning string) bool { return true },
	}
	// 非 HTTPS 会被 downloadHTTPS 拒绝；这里只断言「没有落到 '未经确认' 那条早退」。
	res := ExecuteUseComputer(UseComputerArgs{Action: ActionDownloadFile, URL: "https://example.invalid/x.exe", Path: filepath.Join(dir, "x.exe")}, ctx)
	if strings.Contains(res.Content, "未经确认层批准") {
		t.Errorf("生产入口不应被 Confirmed 门挡住: %+v", res)
	}
}
