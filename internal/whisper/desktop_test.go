package whisper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── T6-5.1: desktop_router.go / desktop_executor.go / desktop_capability_routing.go ──

func TestActionLabel(t *testing.T) {
	if got := ActionLabel(ActionListFolder); got != "列出目录内容" {
		t.Errorf("已知 action 应返回中文标签, got %q", got)
	}
	if got := ActionLabel("zzz"); got != "zzz" {
		t.Errorf("未知 action 应返回原文, got %q", got)
	}
}

func TestCheckActionSettings(t *testing.T) {
	ctx := RouterContext{}
	if got := checkActionSettings(ActionOpenApp, ctx); got != "应用控制未开启" {
		t.Errorf("open_app 应被应用控制拦截, got %q", got)
	}
	if got := checkActionSettings(ActionWriteText, ctx); got != "文件写入未开启" {
		t.Errorf("write_text 应被写入开关拦截, got %q", got)
	}
	if got := checkActionSettings(ActionDeletePath, ctx); got != "删除操作未开启" {
		t.Errorf("delete_path 应被删除开关拦截, got %q", got)
	}
	if got := checkActionSettings(ActionDownloadFile, ctx); got != "下载未开启" {
		t.Errorf("download_file 应被下载开关拦截, got %q", got)
	}
	if got := checkActionSettings(ActionDownloadAndInstall, ctx); got != "安装操作未开启" {
		t.Errorf("download_and_install 应优先被安装开关拦截, got %q", got)
	}
	if got := checkActionSettings(ActionReadDocument, ctx); got != "文档读取未开启" {
		t.Errorf("read_document 应被文档读取开关拦截, got %q", got)
	}
	allOn := RouterContext{AllowAppControl: true, AllowFileWrite: true, AllowDelete: true, AllowDownload: true, AllowInstall: true, AllowDocumentRead: true}
	if got := checkActionSettings(ActionWriteText, allOn); got != "" {
		t.Errorf("全部开启时应放行, got %q", got)
	}
}

func TestIsBlockedCloseTarget(t *testing.T) {
	blocked := []string{"explorer.exe", "EXPLORER.EXE", "lsass", "winlogon.exe", "services"}
	for _, target := range blocked {
		if !isBlockedCloseTarget(target) {
			t.Errorf("%q 应被判定为禁止关闭", target)
		}
	}
	safe := []string{"chrome.exe", "notepad", "myapp.exe", ""}
	for _, target := range safe {
		if isBlockedCloseTarget(target) {
			t.Errorf("%q 不应被禁止关闭", target)
		}
	}
}

func TestEvaluatePathPolicy(t *testing.T) {
	got := evaluatePathPolicy(ActionOpenApp, "", "", "")
	if !got.OK {
		t.Errorf("open_app 应免路径检查: %+v", got)
	}
	got = evaluatePathPolicy(ActionWriteText, "", "", "")
	if got.OK || !strings.Contains(got.HardBlockReason, "缺少路径") {
		t.Errorf("缺路径应硬阻断: %+v", got)
	}
	got = evaluatePathPolicy(ActionOpenFolder, "", "", "")
	if !got.OK {
		t.Errorf("open_folder 应允许空路径: %+v", got)
	}
	got = evaluatePathPolicy(ActionWriteText, "C:\\Windows\\System32\\evil.exe", "", "")
	if got.OK || !strings.Contains(got.HardBlockReason, "系统目录") {
		t.Errorf("System32 写入应硬阻断: %+v", got)
	}
	got = evaluatePathPolicy(ActionWriteText, "C:\\Windows\\Temp\\x.txt", "", "")
	if got.OK == false || got.SensitiveWarning == "" {
		t.Errorf("系统目录应给敏感警告: %+v", got)
	}
}

func TestNormalizePath(t *testing.T) {
	if got := normalizePath("C:\\a\\..\\b", ""); got != "C:\\b" {
		t.Errorf("路径规范化错误: %q", got)
	}
	got := normalizePath("rel.txt", "C:\\base")
	if !strings.Contains(got, "C:\\base") || !strings.Contains(got, "rel.txt") {
		t.Errorf("相对路径应拼接 cwd: %q", got)
	}
	// 实测：os.ExpandEnv 仅展开 $var/${var}，%var%（如 %TEMP%）保持原样（已知限制，见 T6-5.1 报告）
	t.Setenv("DSH_TEST_EXPAND_DIR", "C:\\expanded")
	got = normalizePath("%DSH_TEST_EXPAND_DIR%\\x.txt", "")
	if got != "%DSH_TEST_EXPAND_DIR%\\x.txt" {
		t.Errorf("%%var%% 不被 os.ExpandEnv 展开: %q", got)
	}
	got = normalizePath("$DSH_TEST_EXPAND_DIR\\x.txt", "")
	if !strings.HasSuffix(got, "expanded\\x.txt") {
		t.Errorf("$var 应被展开为绝对路径: %q", got)
	}
	home, _ := os.UserHomeDir()
	got = normalizePath("~/Desktop", "")
	if !strings.Contains(strings.ToLower(got), strings.ToLower(filepath.Base(home))) {
		t.Errorf("~ 应展开到用户主目录: %q", got)
	}
	if got := normalizePath("", "C:\\x"); got != "" {
		t.Errorf("空路径应返回空: %q", got)
	}
}

func TestIsSensitivePath(t *testing.T) {
	if !isSensitivePath("C:\\Windows\\System32") {
		t.Error("Windows 目录应判定为敏感")
	}
	if !isSensitivePath("C:\\Program Files\\App") {
		t.Error("Program Files 应判定为敏感")
	}
	if isSensitivePath("C:\\Users\\wubi\\Desktop") {
		t.Error("用户目录不应判定为敏感")
	}
}

func TestIsHardBlockedWritePath(t *testing.T) {
	if !isHardBlockedWritePath("C:\\Windows\\System32\\x.dll") {
		t.Error("System32 写入应硬阻断")
	}
	if !isHardBlockedWritePath("C:\\Windows\\SysWOW64\\x.dll") {
		t.Error("SysWOW64 写入应硬阻断")
	}
	if isHardBlockedWritePath("C:\\Temp\\x.txt") {
		t.Error("普通路径不应硬阻断")
	}
}

func TestExecuteUseComputer_SettingBlock(t *testing.T) {
	ctx := RouterContext{DataRoot: t.TempDir()}
	res := ExecuteUseComputer(UseComputerArgs{Action: ActionOpenApp, Target: "notepad"}, ctx)
	if res.Success || res.Content != "应用控制未开启" {
		t.Errorf("设置拦截应失败: %+v", res)
	}
}

func TestExecuteUseComputer_BlockedCloseTarget(t *testing.T) {
	ctx := RouterContext{DataRoot: t.TempDir(), AllowAppControl: true, AllowFileWrite: true}
	res := ExecuteUseComputer(UseComputerArgs{Action: ActionCloseApp, Target: "explorer.exe"}, ctx)
	if res.Success || !strings.Contains(res.Content, "系统关键进程") {
		t.Errorf("关闭系统进程应被拦截: %+v", res)
	}
}

func TestExecuteUseComputer_ConfirmDenied(t *testing.T) {
	dir := t.TempDir()
	ctx := RouterContext{
		DataRoot: t.TempDir(), AllowFileWrite: true, CWD: dir,
		RequestConfirm: func(actionLabel, path, target, sensitiveWarning string) bool { return false },
	}
	path := filepath.Join(dir, "a.txt")
	res := ExecuteUseComputer(UseComputerArgs{Action: ActionWriteText, Path: path, Options: &UseComputerOptions{Content: "x"}}, ctx)
	if res.Success || !strings.Contains(res.Content, "未允许") {
		t.Errorf("用户拒绝应失败: %+v", res)
	}
	if res.MemoryHint == "" {
		t.Error("拒绝时应给记忆提示")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("拒绝后不应写入文件")
	}
}

func TestExecuteUseComputer_WriteSuccessAndAudit(t *testing.T) {
	dir := t.TempDir()
	dataRoot := t.TempDir()
	ctx := RouterContext{DataRoot: dataRoot, AllowFileWrite: true, CWD: dir}
	path := filepath.Join(dir, "hello.txt")
	res := ExecuteUseComputer(UseComputerArgs{Action: ActionWriteText, Path: path, Options: &UseComputerOptions{Content: "你好世界"}}, ctx)
	if !res.Success || !strings.Contains(res.Content, "已写入") {
		t.Fatalf("写入应成功: %+v", res)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "你好世界" {
		t.Errorf("文件内容错误: %q err=%v", string(data), err)
	}
	entries, err := ReadAuditEntriesSince(dataRoot, "2020-01-01T00:00:00Z")
	if err != nil || len(entries) != 1 || entries[0].Result != "allowed" {
		t.Errorf("应写入 1 条 allowed 审计: %v err=%v", entries, err)
	}
}

func TestExecuteDesktopAgentAction_Files(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if res := ExecuteDesktopAgentAction(ActionMkdir, sub, "", "", "", "", "", DesktopExecContext{}); !res.OK {
		t.Fatalf("mkdir 失败: %+v", res)
	}
	file := filepath.Join(sub, "a.txt")
	if res := ExecuteDesktopAgentAction(ActionWriteText, file, "", "", "", "", "内容A", DesktopExecContext{}); !res.OK {
		t.Fatalf("write 失败: %+v", res)
	}
	if res := ExecuteDesktopAgentAction(ActionListFolder, sub, "", "", "", "", "", DesktopExecContext{}); !res.OK || !strings.Contains(res.Content, "a.txt") {
		t.Errorf("list 应含 a.txt: %+v", res)
	}
	if res := ExecuteDesktopAgentAction(ActionReadText, file, "", "", "", "", "", DesktopExecContext{}); !res.OK || res.Content != "内容A" {
		t.Errorf("read 内容错误: %+v", res)
	}
	if res := ExecuteDesktopAgentAction(ActionStatFile, file, "", "", "", "", "", DesktopExecContext{}); !res.OK {
		t.Errorf("stat 失败: %+v", res)
	}
	dst := filepath.Join(dir, "b.txt")
	if res := ExecuteDesktopAgentAction(ActionCopyPath, file, dst, "", "", "", "", DesktopExecContext{}); !res.OK {
		t.Fatalf("copy 失败: %+v", res)
	}
	if data, _ := os.ReadFile(dst); string(data) != "内容A" {
		t.Errorf("复制内容错误: %q", string(data))
	}
	moved := filepath.Join(dir, "c.txt")
	if res := ExecuteDesktopAgentAction(ActionMovePath, dst, moved, "", "", "", "", DesktopExecContext{}); !res.OK {
		t.Fatalf("move 失败: %+v", res)
	}
	if _, err := os.Stat(moved); err != nil {
		t.Error("移动后目标应存在")
	}
}

func TestExecuteDesktopAgentAction_Errors(t *testing.T) {
	dir := t.TempDir()
	if res := ExecuteDesktopAgentAction(ActionReadDocument, filepath.Join(dir, "x.docx"), "", "", "", "", "", DesktopExecContext{}); res.OK {
		t.Error("docx 应提示暂不支持")
	}
	if res := ExecuteDesktopAgentAction(ActionReadImage, filepath.Join(dir, "none.png"), "", "", "", "", "", DesktopExecContext{}); res.OK {
		t.Error("不存在的图片应失败")
	}
	if res := ExecuteDesktopAgentAction("no_such_action", "", "", "", "", "", "", DesktopExecContext{}); res.OK {
		t.Error("未知 action 应失败")
	}
	if res := ExecuteDesktopAgentAction(ActionReadText, filepath.Join(dir, "missing.txt"), "", "", "", "", "", DesktopExecContext{}); res.OK {
		t.Error("不存在的文本应失败")
	}
}

func TestExecuteDesktopAgentAction_SearchAndGrep(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "project")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "report.md"), []byte("关键词 内文"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// search_files
	res := ExecuteDesktopAgentAction(ActionSearchFiles, dir, "", "", "report", "", "", DesktopExecContext{})
	if !res.OK || !strings.Contains(res.Content, "report.md") {
		t.Errorf("search_files 应命中 report.md: %+v", res)
	}
	// grep_text
	res = ExecuteDesktopAgentAction(ActionGrepText, dir, "", "", "关键词", "", "", DesktopExecContext{})
	if !res.OK || !strings.Contains(res.Content, "report.md") {
		t.Errorf("grep_text 应命中 report.md: %+v", res)
	}
	// grep 缺关键词
	res = ExecuteDesktopAgentAction(ActionGrepText, dir, "", "", "", "", "", DesktopExecContext{})
	if res.OK {
		t.Errorf("grep 缺关键词应失败: %+v", res)
	}
}

func TestExecuteDesktopAgentAction_ReadImageAndImport(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(img, []byte("fake-png"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// read_image 仅定位文件
	res := ExecuteDesktopAgentAction(ActionReadImage, img, "", "", "", "", "", DesktopExecContext{})
	if !res.OK || !strings.Contains(res.Summary, "pic.png") {
		t.Errorf("read_image 应定位文件: %+v", res)
	}
	// import_to_ackem 复制到 DataRoot/imports
	dataRoot := t.TempDir()
	res = ExecuteDesktopAgentAction(ActionImportToAckem, img, "", "", "", "", "", DesktopExecContext{DataRoot: dataRoot})
	if !res.OK {
		t.Fatalf("import 失败: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "imports", "pic.png")); err != nil {
		t.Errorf("导入后文件应存在于 imports 目录: %v", err)
	}
}

func TestDownloadHTTPS_NonHTTPS(t *testing.T) {
	res := downloadHTTPS("http://example.com/x.exe", filepath.Join(t.TempDir(), "x.exe"))
	if res.OK || !strings.Contains(res.Content, "仅支持 HTTPS") {
		t.Errorf("非 HTTPS 应拒绝: %+v", res)
	}
}

func TestDefaultDownloadDir(t *testing.T) {
	if got := defaultDownloadDir(""); !strings.HasSuffix(got, "LightWhisperDownloads") {
		t.Errorf("空配置应回退默认下载目录: %q", got)
	}
	if got := defaultDownloadDir("C:\\dl"); got != "C:\\dl" {
		t.Errorf("已配置目录应原样返回: %q", got)
	}
}

func TestCoalesce(t *testing.T) {
	if coalesce("a", "b") != "a" || coalesce("", "b") != "b" {
		t.Error("coalesce 逻辑错误")
	}
}

// ─── desktop_capability_routing.go ──────────────────────────────

func TestListRoutableCapabilities(t *testing.T) {
	caps := ListRoutableCapabilities()
	if len(caps) != 9 {
		t.Fatalf("应有 9 个能力, got %d", len(caps))
	}
	seen := map[string]bool{}
	for _, c := range caps {
		if !c.Enabled || c.ID == "" || c.Handler == "" {
			t.Errorf("能力定义不完整: %+v", c)
		}
		if seen[c.ID] {
			t.Errorf("能力 ID 重复: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestResolveDesktopCapabilityEnhanced_Keyword(t *testing.T) {
	m := ResolveDesktopCapabilityEnhanced("列出桌面文件")
	if m == nil || m.Handler != "list_folder" || m.Source != "keyword" {
		t.Errorf("关键词路由失败: %+v", m)
	}
	m = ResolveDesktopCapabilityEnhanced("你能做什么")
	if m == nil || m.Handler != "capability_help" {
		t.Errorf("能力帮助路由失败: %+v", m)
	}
	m = ResolveDesktopCapabilityEnhanced("整理桌面")
	if m == nil || m.Handler != "organize_files" {
		t.Errorf("整理路由失败: %+v", m)
	}
}

func TestResolveDesktopCapabilityEnhanced_CleanupOverride(t *testing.T) {
	m := ResolveDesktopCapabilityEnhanced("整理一下游戏文件夹")
	if m == nil || m.Handler != "organize_files" || m.Score != 0.55 || m.Source != "regex_fallback" {
		t.Errorf("清理意图覆盖失败: %+v", m)
	}
}

func TestResolveDesktopCapabilityEnhanced_RegexFallback(t *testing.T) {
	m := ResolveDesktopCapabilityEnhanced("整理")
	if m == nil || m.Handler != "organize_files" || m.Source != "regex_fallback" {
		t.Errorf("「整理」应走正则兜底: %+v", m)
	}
	m = ResolveDesktopCapabilityEnhanced("我想玩游戏")
	if m == nil || m.Handler != "investigate_games" {
		t.Errorf("游戏意图应命中: %+v", m)
	}
	m = ResolveDesktopCapabilityEnhanced("有没有pdf文档")
	if m == nil || m.Handler != "investigate_documents" {
		t.Errorf("文档意图应命中: %+v", m)
	}
}

func TestResolveDesktopCapabilityEnhanced_NoMatch(t *testing.T) {
	if m := ResolveDesktopCapabilityEnhanced("你好呀"); m != nil {
		t.Errorf("无关消息应返回 nil, got %+v", m)
	}
	if m := ResolveDesktopCapabilityEnhanced("   "); m != nil {
		t.Errorf("空白消息应返回 nil, got %+v", m)
	}
}

func TestScoreByExamples(t *testing.T) {
	if got := scoreByExamples("列出桌面文件", []string{"列出桌面文件", "查看目录"}); got != 1.0 {
		t.Errorf("精确匹配应得 1.0, got %v", got)
	}
	got := scoreByExamples("找一下文件", []string{"找一下", "找一下文件", "找文件"})
	if got <= 0 || got > 1.0 {
		t.Errorf("部分匹配分数异常: %v", got)
	}
	if got := scoreByExamples("完全不相关", []string{"abc", "xyz"}); got != 0 {
		t.Errorf("无匹配应 0, got %v", got)
	}
}

func TestPartialMatch(t *testing.T) {
	if !partialMatch("吃辣", "喜欢吃辣") {
		t.Error("共享 bigram 应部分匹配")
	}
	if partialMatch("abc", "xzy") {
		t.Error("无共享 bigram 不应匹配")
	}
}

func TestIsCleanupIntent(t *testing.T) {
	for _, msg := range []string{"整理", "清理垃圾", "收拾一下", "归类文件", "organize my files", "分类"} {
		if !isCleanupIntent(msg) {
			t.Errorf("%q 应为清理意图", msg)
		}
	}
	if isCleanupIntent("帮我查天气") {
		t.Error("无关消息不应是清理意图")
	}
}

func TestMatchByRegexFallback(t *testing.T) {
	if m := matchByRegexFallback("整理桌面"); m == nil || m.Handler != "organize_files" {
		t.Errorf("清理兜底失败: %+v", m)
	}
	if m := matchByRegexFallback("装了哪些游戏"); m == nil || m.Handler != "investigate_games" {
		t.Errorf("游戏兜底失败: %+v", m)
	}
	if m := matchByRegexFallback("我的文档在哪里"); m == nil || m.Handler != "investigate_documents" {
		t.Errorf("文档兜底失败: %+v", m)
	}
	if m := matchByRegexFallback("怎么用"); m == nil || m.Handler != "capability_help" {
		t.Errorf("帮助兜底失败: %+v", m)
	}
	if m := matchByRegexFallback("随便聊聊"); m != nil {
		t.Errorf("无关消息兜底应为 nil, got %+v", m)
	}
}

func TestAuditLogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	entry := DesktopAgentAuditEntry{TS: "2026-01-01T00:00:00Z", Action: "list_folder", Path: "C:\\x", Result: "allowed", Summary: "ok"}
	if err := AppendDesktopAgentAudit(dir, entry); err != nil {
		t.Fatalf("追加审计失败: %v", err)
	}
	entries, err := ReadAuditEntriesSince(dir, "2020-01-01T00:00:00Z")
	if err != nil || len(entries) != 1 || entries[0].Action != "list_folder" {
		t.Fatalf("读取审计失败: %v err=%v", entries, err)
	}
	if got, err := ReadAuditEntriesSince(dir, "garbage"); err != nil || got != nil {
		t.Errorf("非法时间应返回 nil,nil: %v %v", got, err)
	}
}
