package app

// office_com_tools_test.go — dsh-office COM 实时协同档工具层测试（波1）：
//   - 工具面形状：八件注册名/只读性/schema 合法 JSON；office_edit 非只读；
//     open/save/close 实现 WritePathResolver，read 族不实现（读边界由
//     officeComResolvePath 单点收口）；
//   - WritePathResolver：open 模板/新建声明、save path/当前文档、close
//     save:true；工作区外路径单点拒绝；
//   - 会话级操作清单发放（首次含、二次缺）+ current/own 会话态；
//   - save 落 Journal 证据卡（与四类写同账）+ Word 保存前 fill_gaps；
//   - BUSY 透传；render 产物落工作区 .gaea/renders/office。
//
// helper 调用全部走 stubCaller（协议真进程测试在 comhelper 包）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
	comhelper "github.com/gaea/gaea/internal/office/comhelper"
)

// stubCaller 桩：按 cmd 回预设结果并记录调用。
type stubCaller struct {
	mu    sync.Mutex
	calls []stubCall
	reply func(cmd string, args map[string]any) (json.RawMessage, error)
}

type stubCall struct {
	cmd  string
	args map[string]any
}

func (s *stubCaller) Call(_ context.Context, cmd string, args map[string]any, _ time.Duration) (json.RawMessage, error) {
	s.mu.Lock()
	s.calls = append(s.calls, stubCall{cmd: cmd, args: args})
	reply, err := s.reply(cmd, args)
	s.mu.Unlock()
	return reply, err
}

func (s *stubCaller) count(cmd string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if c.cmd == cmd {
			n++
		}
	}
	return n
}

// stubJSON 便捷构造成功回复。
func stubJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

// resetOfficeComForTest 单例复位 + 注入桩（包级单例仅测试操纵；进程生命周期
// 由管道关闭语义兜底）。
func resetOfficeComForTest(stub officeComCaller) {
	officeCom.mu.Lock()
	officeCom.caller = stub
	officeCom.currentApp, officeCom.currentDoc = "", ""
	officeCom.own = nil
	officeCom.mu.Unlock()
	officeCom.toldMu.Lock()
	officeCom.told = map[string]map[string]bool{}
	officeCom.toldMu.Unlock()
}

func docInfoStub(how string, ws string) comhelper.DocInfo {
	path := filepath.Join(ws, "报告.docx")
	return comhelper.DocInfo{App: "word", Name: "报告.docx", Path: &path, Saved: false, How: how}
}

func TestOfficeComRegistryShape(t *testing.T) {
	isolateOfficeWorkspace(t)
	type entry struct {
		tl       tool.Tool
		readOnly bool
	}
	eight := []entry{
		{(officeComOpenTool{}), false},
		{(officeComStatusTool{}), true},
		{(officeComReadTool{}), true},
		{(officeComEditTool{}), false},
		{(officeComRenderTool{}), true},
		{(officeComSaveTool{}), false},
		{(officeComCloseTool{}), false},
		{(officeComHelpTool{}), true},
	}
	for _, e := range eight {
		name := e.tl.Name()
		if e.tl.ReadOnly() != e.readOnly {
			t.Fatalf("%s ReadOnly=%v, want %v", name, e.tl.ReadOnly(), e.readOnly)
		}
		schema := e.tl.Schema()
		if !json.Valid(schema) {
			t.Fatalf("%s schema 非 JSON: %s", name, schema)
		}
		if strings.TrimSpace(e.tl.Description()) == "" {
			t.Fatalf("%s description 为空", name)
		}
		_, hasResolver := e.tl.(tool.WritePathResolver)
		wantResolver := name == "office_open" || name == "office_save" || name == "office_close"
		if hasResolver != wantResolver {
			t.Fatalf("%s WritePathResolver=%v, want %v", name, hasResolver, wantResolver)
		}
	}
	if (officeComOpenTool{}).Name() != "office_open" || (officeComEditTool{}).Name() != "office_edit" ||
		(officeComSaveTool{}).Name() != "office_save" || (officeComHelpTool{}).Name() != "office_help" {
		t.Fatal("工具名与上游协议名不一致")
	}
}

func TestOfficeComResolvePathWorkspaceOnly(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	got, err := officeComResolvePath("报告/总结.docx")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(ws, "报告", "总结.docx") {
		t.Fatalf("got=%s want=%s", got, filepath.Join(ws, "报告", "总结.docx"))
	}
	// 工作区外绝对路径：单点拒绝（波1 口径）。
	outside := filepath.Join(filepath.Dir(ws), "外部.docx")
	if _, err := officeComResolvePath(outside); err == nil || !strings.Contains(err.Error(), "工作区外") {
		t.Fatalf("工作区外应拒绝: %v", err)
	}
	if _, err := officeComResolvePath(""); err == nil {
		t.Fatal("空路径应拒绝")
	}
}

func TestOfficeComOpenGuideOnceAndOwnTracking(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		if cmd != "open" {
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
		if _, ok := args["path"].(string); !ok {
			return nil, fmt.Errorf("open 缺 path")
		}
		return stubJSON(docInfoStub("created", ws)), nil
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()

	// 首次 open：回执带 Word 操作清单（会话级发放）。
	out1, err := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s1"}, json.RawMessage(`{"path":"报告.docx"}`))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !strings.Contains(out1, "operations for office_edit") {
		t.Fatalf("首次 open 应附操作清单: %.200s", out1)
	}
	if !strings.Contains(out1, "Created") {
		t.Fatalf("created 回执: %.200s", out1)
	}
	// 同会话二次：清单不再发放。
	out2, err := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s1"}, json.RawMessage(`{"path":"报告.docx"}`))
	if err != nil {
		t.Fatalf("open2: %v", err)
	}
	if strings.Contains(out2, "operations for office_edit") {
		t.Fatal("第二次 open 不应重复发放清单")
	}
	// 新会话再发一次。
	out3, _ := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s2"}, json.RawMessage(`{"path":"报告.docx"}`))
	if !strings.Contains(out3, "operations for office_edit") {
		t.Fatal("新会话应重新发放清单")
	}
	// own 收编 + helper 收到绝对路径 + current 跟随。
	officeCom.mu.Lock()
	inOwn := len(officeCom.own) == 1
	current := officeCom.currentDoc
	officeCom.mu.Unlock()
	if !inOwn {
		t.Fatal("created 文档应收编进 own")
	}
	if current != filepath.Join(ws, "报告.docx") {
		t.Fatalf("current 应为打开路径: %q", current)
	}
	if stub.count("open") != 3 {
		t.Fatalf("open 调用次数=%d", stub.count("open"))
	}
}

func TestOfficeComSaveEvidenceAndFillGaps(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		switch cmd {
		case "open":
			return stubJSON(docInfoStub("opened", ws)), nil
		case "edit": // 保存前的 fill_gaps 补空
			return stubJSON(comhelper.EditResult{Done: []string{"no page needed filling"}}), nil
		case "save":
			return stubJSON(map[string]any{"path": filepath.ToSlash(filepath.Join(ws, "报告.docx")), "bytes": 1234}), nil
		default:
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()

	// 先 open（Word 文档）建立 current，再 save。
	if _, err := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-save"}, json.RawMessage(`{"path":"报告.docx"}`)); err != nil {
		t.Fatalf("open: %v", err)
	}
	out, err := (officeComSaveTool{}).Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !strings.Contains(out, "Saved") {
		t.Fatalf("save 回执: %.200s", out)
	}
	// Word 保存前自动 fill_gaps（edit 命令一次）。
	if stub.count("edit") != 1 {
		t.Fatalf("Word 保存前应跑一次 fill_gaps（edit 调用=%d）", stub.count("edit"))
	}
	// Journal 证据卡（office_save，无基线快照）。
	recs := readOfficeJournal(t, ws)
	var found bool
	for _, rec := range recs {
		if rec.Tool == "office_save" {
			found = true
			if strings.Contains(rec.AfterSummary, "1234") {
				break
			}
			t.Fatalf("证据卡缺落盘信息: %+v", rec)
		}
	}
	if !found {
		t.Fatalf("office_save 证据卡未落账: %+v", recs)
	}
}

func TestOfficeComSaveWritePaths(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	// 显式 path。
	got, err := (officeComSaveTool{}).WritePaths(json.RawMessage(`{"path":"交付/报告.docx"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(ws, "交付", "报告.docx") {
		t.Fatalf("WritePaths=%v", got)
	}
	// 既无 path 也无 current：拒绝。
	resetOfficeComForTest(&stubCaller{})
	if _, err := (officeComSaveTool{}).WritePaths(json.RawMessage(`{}`)); err == nil {
		t.Fatal("无落盘路径应拒绝")
	}
	// current 为绝对路径时缺省可用。
	officeCom.mu.Lock()
	officeCom.currentApp, officeCom.currentDoc = "word", filepath.Join(ws, "在案.docx")
	officeCom.mu.Unlock()
	got, err = (officeComSaveTool{}).WritePaths(json.RawMessage(`{}`))
	if err != nil || got[0] != filepath.Join(ws, "在案.docx") {
		t.Fatalf("current 缺省 WritePaths=%v err=%v", got, err)
	}
}

func TestOfficeComCloseOwnGuard(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	saved := true
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		switch cmd {
		case "open":
			return stubJSON(docInfoStub("created", ws)), nil
		case "status":
			docs := []comhelper.DocInfo{}
			if !saved {
				p := filepath.Join(ws, "报告.docx")
				docs = append(docs, comhelper.DocInfo{App: "word", Name: "报告.docx", Path: &p, Saved: false})
			}
			return stubJSON([]comhelper.AppStatus{{App: "word", Installed: true, Running: true, Documents: docs}}), nil
		case "close":
			return stubJSON("ok"), nil
		default:
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()

	// 非 own 文档：拒绝关闭。
	if _, err := (officeComCloseTool{}).Execute(ctx, json.RawMessage(`{"doc":"报告.docx"}`)); err == nil {
		t.Fatal("非 own 文档应拒绝关闭")
	}
	// open 收编 own 后：未保存未表态 → 要求明确 save。
	if _, err := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-close"}, json.RawMessage(`{"path":"报告.docx"}`)); err != nil {
		t.Fatalf("open: %v", err)
	}
	saved = false
	if _, err := (officeComCloseTool{}).Execute(ctx, json.RawMessage(`{"doc":"报告.docx"}`)); err == nil || !strings.Contains(err.Error(), "save") {
		t.Fatalf("未保存修改应要求明确 save: %v", err)
	}
	// save:false 放弃修改关闭。
	if out, err := (officeComCloseTool{}).Execute(ctx, json.RawMessage(`{"doc":"报告.docx","save":false}`)); err != nil {
		t.Fatalf("close save:false: %v", err)
	} else if !strings.Contains(out, "Closed") {
		t.Fatalf("close 回执: %.200s", out)
	}
	if officeCom.own != nil && len(officeCom.own) != 0 {
		t.Fatal("关闭后 own 应移除")
	}
}

func TestOfficeComEditOpsPassthroughAndGuard(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	var gotOps string
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		if cmd != "edit" {
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
		if raw, ok := args["ops"].(string); ok {
			gotOps = raw
		} else if raw, mErr := json.Marshal(args["ops"]); mErr == nil {
			gotOps = string(raw)
		}
		return stubJSON(comhelper.EditResult{Done: []string{"set_text OK"}, Total: 1}), nil
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()

	out, err := (officeComEditTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-edit"},
		json.RawMessage(`{"doc":"报告.docx","ops":[{"op":"set_text","para":3,"expect":"原开头","text":"新文本"}]}`))
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !strings.Contains(out, "1 operation(s) applied") {
		t.Fatalf("edit 回执: %.200s", out)
	}
	if !strings.Contains(gotOps, "set_text") || !strings.Contains(gotOps, "原开头") {
		t.Fatalf("ops 未透传: %s", gotOps)
	}
	// 空数组/超限拒绝。
	if _, err := (officeComEditTool{}).Execute(ctx, json.RawMessage(`{"ops":[]}`)); err == nil {
		t.Fatal("空 ops 应拒绝")
	}
	many := "["
	for i := 0; i < 201; i++ {
		if i > 0 {
			many += ","
		}
		many += `{"op":"noop"}`
	}
	many += "]"
	if _, err := (officeComEditTool{}).Execute(ctx, json.RawMessage(`{"doc":"x.docx","ops":`+many+`}`)); err == nil {
		t.Fatal("超 200 ops 应拒绝")
	}
	_ = ws
}

func TestOfficeComBusyPassesThrough(t *testing.T) {
	isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		return nil, &comhelper.ComError{Code: "BUSY", Message: "The Office app is busy: try again in a moment."}
	}}
	resetOfficeComForTest(stub)
	_, err := (officeComStatusTool{}).Execute(context.Background(), json.RawMessage(`{}`))
	if !comhelper.IsBusy(err) {
		t.Fatalf("BUSY 应透传: %v", err)
	}
}

func TestOfficeComRenderPathsUnderWorkspace(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		if cmd != "render" {
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
		outPath, _ := args["out"].(string)
		if err := os.WriteFile(outPath, []byte("png"), 0o644); err != nil { // helper 会写文件
			return nil, err
		}
		return stubJSON(map[string]any{"what": "Word page 1", "total": 3, "width": 1100.0, "height": 1500.0}), nil
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()

	out, err := (officeComRenderTool{}).Execute(ctx, json.RawMessage(`{"doc":"报告.docx","page":1}`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var payload struct {
		Pages  []string `json:"pages"`
		Vision string   `json:"vision"`
	}
	if json.Unmarshal([]byte(out), &payload) != nil || len(payload.Pages) != 1 {
		t.Fatalf("render 回执: %.400s", out)
	}
	for _, p := range payload.Pages {
		if !strings.HasPrefix(p, filepath.Join(ws, ".gaea", "renders", "office")) {
			t.Fatalf("渲染产物应落工作区 .gaea/renders/office: %s", p)
		}
		if _, statErr := os.Stat(p); statErr != nil {
			t.Fatalf("渲染产物不存在: %s", p)
		}
	}
	if !strings.Contains(payload.Vision, "vision") {
		t.Fatalf("缺 vision 交接提示: %s", payload.Vision)
	}
}

func TestOfficeComTargetFallbackAndAppOf(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	officeCom.mu.Lock()
	officeCom.currentApp, officeCom.currentDoc = "", ""
	officeCom.mu.Unlock()
	// 无 current 无 doc：报错（不是 panic）。
	if _, _, err := officeCom.target("", ""); err == nil {
		t.Fatal("无目标应报错")
	}
	// 扩展名推断 app。
	kind, target, err := officeCom.target("", filepath.Join(ws, "表.xlsx"))
	if err != nil || kind != "excel" {
		t.Fatalf("appOf=%s err=%v", kind, err)
	}
	if target != filepath.Join(ws, "表.xlsx") {
		t.Fatalf("target=%s", target)
	}
	// current 复用：doc 缺省回退。
	officeCom.mu.Lock()
	officeCom.currentApp, officeCom.currentDoc = "word", filepath.Join(ws, "报告.docx")
	officeCom.mu.Unlock()
	kind, target, err = officeCom.target("", "")
	if err != nil || kind != "word" || target != filepath.Join(ws, "报告.docx") {
		t.Fatalf("current 复用 kind=%s target=%s err=%v", kind, target, err)
	}
}
