package app

// office_com_settle_test.go — COM 档波3 settle 收尾钩子测试：
//   - officeComSettings 单点读取：ga.cfg==nil 回上游默认值（测试态安全）、
//     cfg 值透传、非法 suite/越界宽度归一（config.OfficeComNormalized 口径）；
//   - settle 收尾：静默 open 记账 → TurnDone 槽函数直接调 maybeOfficeComSettle
//     → 发 settle+card 命令、Journal 落 office_save 证据卡、emit office_file_saved
//    （测试态 ctx nil 走 core.emit 的空守卫，断言不炸即可）、记账清空；
//   - 无活动时零 helper 命令（stub.calls 为空——TurnDone 每回合触发，绝不空 ping）；
//   - settle 说明行解析（与 OfficeHelper.cs Settle() 实测形态逐字对齐）。
//
// helper 调用全部走 stubCaller（协议真进程测试在 comhelper 包）。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// withOfficeComCfg 在 ga.cfg 注入 [office_com] 段并注册恢复（isolateOfficeWorkspace
// 已装 Workspace cfg，这里只改 OfficeCom 子段，同一指针随 cleanup 还原）。
func withOfficeComCfg(t *testing.T, oc gaeaConfig.OfficeComConfig) {
	t.Helper()
	ga.mu.Lock()
	if ga.cfg == nil {
		ga.cfg = &gaeaConfig.Config{}
	}
	ga.cfg.OfficeCom = oc
	ga.mu.Unlock()
}

// TestOfficeComSettings 波3 设置单点：nil 回默认、值透传、非法归一。
func TestOfficeComSettings(t *testing.T) {
	ga.mu.Lock()
	saved := ga.cfg
	ga.cfg = nil
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.cfg = saved
		ga.mu.Unlock()
	})

	// ga.cfg==nil（测试态/启动早期）：上游 DEFAULTS 口径。
	s := officeComSettings()
	if s.suite != "auto" || s.silent || !s.follow || !s.typing || s.renderWidth != 1100 {
		t.Fatalf("nil cfg 应回上游默认值: %+v", s)
	}

	// cfg 值透传（合法段原样）。
	ga.mu.Lock()
	ga.cfg = &gaeaConfig.Config{OfficeCom: gaeaConfig.OfficeComConfig{
		Suite: "wps", Silent: true, Follow: false, Typing: false, RenderWidth: 900,
	}}
	ga.mu.Unlock()
	s = officeComSettings()
	if s.suite != "wps" || !s.silent || s.follow || s.typing || s.renderWidth != 900 {
		t.Fatalf("cfg 值应透传: %+v", s)
	}

	// 非法 suite 回 auto；越界宽度夹 [600,2000]；布尔原样透传（裸 Config 零值
	// = 显式关闭——生产路径恒经 Default() 播种 follow/typing=true）。
	ga.mu.Lock()
	ga.cfg = &gaeaConfig.Config{OfficeCom: gaeaConfig.OfficeComConfig{Suite: "libre", RenderWidth: 5000}}
	ga.mu.Unlock()
	s = officeComSettings()
	if s.suite != "auto" || s.silent || s.follow || s.typing || s.renderWidth != 2000 {
		t.Fatalf("非法值应归一: %+v", s)
	}
}

// TestOfficeComSettleSavesSilentDocs 静默 open 后 TurnDone 槽直接调
// maybeOfficeComSettle：发 settle+card、Journal 落 office_save 账、记账清空、
// 二连调零命令（单飞节流）。
func TestOfficeComSettleSavesSilentDocs(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	withOfficeComCfg(t, gaeaConfig.OfficeComConfig{Suite: "auto", Silent: true})
	var (
		openSilent, openShow, openCard bool
		cardArgs                       map[string]any
	)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		switch cmd {
		case "open":
			openSilent, _ = args["silent"].(bool)
			openShow, _ = args["show"].(bool)
			openCard, _ = args["card"].(bool)
			return stubJSON(docInfoStub("created", ws)), nil
		case "settle":
			// helper Settle() 实测回执形态：纯文本说明行数组（无结构化路径字段）。
			return stubJSON([]string{
				"saved and closed " + filepath.Join(ws, "报告.docx"),
				"closed " + filepath.Join(ws, "已关.docx"),
				"left open " + filepath.Join(ws, "别处.docx") + " (the user has it on screen)",
			}), nil
		case "card":
			cardArgs = args
			return stubJSON("ok"), nil
		default:
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
	}}
	resetOfficeComForTest(stub)

	// 静默模式 open（不传 show）：args 应 silent=true/show=false，文档记入
	// silentOpen，open 成功 stamp 活动时刻。
	if _, err := (officeComOpenTool{}).ExecuteWithContext(context.Background(),
		tool.ToolContext{SessionID: "s-settle"}, json.RawMessage(`{"path":"报告.docx"}`)); err != nil {
		t.Fatalf("open: %v", err)
	}
	if !openSilent || openShow || openCard {
		t.Fatalf("静默 open args 应 silent=true/show=false/card=false: got %v/%v/%v", openSilent, openShow, openCard)
	}
	officeCom.mu.Lock()
	silenced := len(officeCom.silentOpen) == 1
	officeCom.mu.Unlock()
	if !silenced {
		t.Fatal("静默 open 应记入 silentOpen")
	}

	// TurnDone 槽函数直接调（成功回合末同款）。emit 走 core.emit 的 ctx nil
	// 守卫（httpbridge.Publish 无订阅者 no-op），断言不炸即可。
	a := &App{}
	a.maybeOfficeComSettle()

	if stub.count("settle") != 1 || stub.count("card") != 1 {
		t.Fatalf("应收发 settle+card 各一次: settle=%d card=%d", stub.count("settle"), stub.count("card"))
	}
	if text, _ := cardArgs["text"].(string); text != "AI 已完成" {
		t.Fatalf("card text 应为「AI 已完成」: %v", cardArgs["text"])
	}
	if hold, _ := cardArgs["hold"].(bool); hold {
		t.Fatalf("card hold 应为 false: %v", cardArgs["hold"])
	}

	// Journal 证据卡：office_save 同账，Target=静默文档 rel，回执行整行落
	// afterSummary（helper settle 回执无结构化路径字段）。
	recs := readOfficeJournal(t, ws)
	var found bool
	for _, rec := range recs {
		if rec.Tool == "office_save" && rec.Target == "报告.docx" &&
			strings.Contains(rec.AfterSummary, "saved and closed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("settle 保存应落 office_save 证据卡: %+v", recs)
	}

	// 收尾完成：静默记账清空；二连调零新命令（lastSettle 已 stamp + 无静默文档）。
	officeCom.mu.Lock()
	left := len(officeCom.silentOpen)
	officeCom.mu.Unlock()
	if left != 0 {
		t.Fatalf("settle 后 silentOpen 应清空: %d", left)
	}
	stub.mu.Lock()
	stub.calls = nil
	stub.mu.Unlock()
	a.maybeOfficeComSettle()
	if stub.count("settle") != 0 || stub.count("card") != 0 {
		t.Fatalf("紧随的二连调应零命令（守卫节流）: settle=%d card=%d", stub.count("settle"), stub.count("card"))
	}
}

// TestOfficeComSettleIdleZeroCommands 无活动会话（call 空+无静默文档）：TurnDone
// 槽触发也不发任何 helper 命令（含不创建 client、不拉起 helper 进程）。
func TestOfficeComSettleIdleZeroCommands(t *testing.T) {
	isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		return stubJSON("ok"), nil
	}}
	resetOfficeComForTest(stub)
	a := &App{}
	a.maybeOfficeComSettle()
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.calls) != 0 {
		t.Fatalf("无活动应零命令: %+v", stub.calls)
	}
}

// TestParseOfficeComSettleLine 说明行解析与 OfficeHelper.cs Settle() 实测形态
// 逐字对齐（前缀+路径拼接；移交用户行剥固定后缀）。
func TestParseOfficeComSettleLine(t *testing.T) {
	cases := []struct {
		line  string
		saved bool
		path  string
	}{
		{"saved and closed C:\\w\\报告.docx", true, `C:\w\报告.docx`},
		{"saved C:\\w\\别处.docx (the user has it on screen)", true, `C:\w\别处.docx`},
		{"closed C:\\w\\已关.docx", false, `C:\w\已关.docx`},
		{"left open C:\\w\\别处.docx (the user has it on screen)", false, `C:\w\别处.docx`},
		{"could not close C:\\w\\卡住.docx: RPC 失败", false, ""},
		{"", false, ""},
		{"totally unknown", false, ""}, // helper 未来格式变化的不认行
	}
	for i, tc := range cases {
		got := parseOfficeComSettleLine(tc.line)
		if got.saved != tc.saved || got.path != tc.path {
			t.Fatalf("case %d (%q): got %+v want saved=%v path=%q", i, tc.line, got, tc.saved, tc.path)
		}
	}
}
