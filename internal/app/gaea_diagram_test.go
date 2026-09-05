package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// TestDiagramToolMeta 校验画图工具元信息与 Schema。
func TestDiagramToolMeta(t *testing.T) {
	tool := diagramTool{}
	if tool.Name() != "diagram" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) || !json.Valid(tool.CompactSchema()) {
		t.Fatal("Schema 非法")
	}
	if tool.ReadOnly() {
		t.Error("diagram 不应为只读工具")
	}
}

// TestExtractDiagramMermaid 兼容围栏/裸代码提取。
func TestExtractDiagramMermaid(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"```mermaid\nflowchart LR\nA-->B\n```", "flowchart LR\nA-->B"},
		{"```\nflowchart TD\nA-->B\n```", "flowchart TD\nA-->B"},
		{"flowchart LR\nA-->B", "flowchart LR\nA-->B"},
		{"请参考：\n```mermaid\npie title T\n\"A\": 1\n```\n以上", "pie title T\n\"A\": 1"},
	}
	for _, c := range cases {
		if got := extractDiagramMermaid(c.raw); got != c.want {
			t.Errorf("extractDiagramMermaid(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// TestValidMermaidStart 首行关键字校验。
func TestValidMermaidStart(t *testing.T) {
	ok := []string{"flowchart LR\nA-->B", "sequenceDiagram\nA->>B", "%% comment\nmindmap\n root", "gantt\ntitle X"}
	for _, s := range ok {
		if !validMermaidStart(s) {
			t.Errorf("validMermaidStart(%q) = false, want true", s)
		}
	}
	bad := []string{"随便一句话", "graphql query", ""}
	for _, s := range bad {
		if validMermaidStart(s) {
			t.Errorf("validMermaidStart(%q) = true, want false", s)
		}
	}
}

// TestSaveDiagramMermaid 落盘 .mmd 文件。
func TestSaveDiagramMermaid(t *testing.T) {
	t.Chdir(t.TempDir())
	rel, err := saveDiagramMermaid(".", "flowchart LR\nA-->B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, ".gaea/uploads/diagram-") || !strings.HasSuffix(rel, ".mmd") {
		t.Fatalf("路径 = %q", rel)
	}
	data, err := os.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if !strings.Contains(string(data), "flowchart") {
		t.Error("文件内容不正确")
	}
}

// TestGenerateDiagramGuard 绘梦图表模式的空提示词 / 未初始化保护。
func TestGenerateDiagramGuard(t *testing.T) {
	// 未初始化客户端
	a := &mediaState{}
	res, err := a.GenerateDiagram("订单处理流程图")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["error"] == nil {
		t.Fatal("未初始化客户端应返回错误")
	}

	// 空提示词（不触发 LLM 调用）
	a = &mediaState{core: &core{client: &ai.Client{}}}
	res, err = a.GenerateDiagram("   ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["error"] == nil {
		t.Fatal("空提示词应返回错误")
	}
}

// armImageHubGateWithWorkspace 为登记用例模拟运行态：武装位置位 + 工作区指向
// 独立临时目录（ga.cfg 全局态先存后还，避免污染同包其他测试）。
func armImageHubGateWithWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	imageHubRuntimeArmed.Store(true)
	t.Cleanup(func() {
		imageHubRuntimeArmed.Store(false)
		ga.cfg = oldCfg
	})
	return ws
}

// TestRegisterDiagramAssetSuccess 图示登记接线（media.diagram）：武装态下落盘
// .mmd 并写 ledger，登记字段（原语/来源/类型/提示词/AI 标记/params）正确。
func TestRegisterDiagramAssetSuccess(t *testing.T) {
	ws := armImageHubGateWithWorkspace(t)

	registerDiagramAsset("flowchart LR\nA-->B", "订单审批流程")

	// .mmd 已落盘且内容为 Mermaid 代码。
	entries, err := os.ReadDir(filepath.Join(ws, ".gaea", "uploads"))
	if err != nil {
		t.Fatalf("读取 uploads: %v", err)
	}
	var mmdPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mmd") {
			mmdPath = filepath.Join(ws, ".gaea", "uploads", e.Name())
		}
	}
	if mmdPath == "" {
		t.Fatal("图示 .mmd 未落盘")
	}
	data, err := os.ReadFile(mmdPath)
	if err != nil || !strings.Contains(string(data), "flowchart LR") {
		t.Fatalf("落盘内容错误: %v %q", err, string(data))
	}

	// 登记写入 work 空间 ledger（测试工作区无空间分区，gaeaEffectiveSpace()=""),
	// 字段与试点口径一致。
	recs := newImageHubLedger(ws).list("work", 0)
	if len(recs) != 1 {
		t.Fatalf("应登记 1 条，got %d", len(recs))
	}
	rec := recs[0]
	if rec.Meta.Capability != string(CapabilityMediaDiagram) {
		t.Errorf("capability = %q, want media.diagram", rec.Meta.Capability)
	}
	if rec.Meta.SourceBoard != "imagegen" {
		t.Errorf("source_board = %q", rec.Meta.SourceBoard)
	}
	if rec.Asset.Kind != ImageHubAssetKindDiagram {
		t.Errorf("kind = %q, want diagram", rec.Asset.Kind)
	}
	if rec.Asset.Path != mmdPath {
		t.Errorf("asset path = %q, want %q", rec.Asset.Path, mmdPath)
	}
	if rec.Meta.Prompt != "订单审批流程" || !rec.Meta.AIFlag {
		t.Errorf("溯源字段缺失: %+v", rec.Meta)
	}
	// 图示走活跃对话引擎，无显式模型名 → 模型/成本诚实留空（不伪装 0）。
	if rec.Meta.Model != "" || rec.Meta.Cost != "" {
		t.Errorf("model/cost 应诚实留空: model=%q cost=%q", rec.Meta.Model, rec.Meta.Cost)
	}
	if rec.Meta.Params["format"] != "mermaid" {
		t.Errorf("params.format = %v", rec.Meta.Params["format"])
	}
}

// TestRegisterDiagramAssetLedgerFailureWarnOnly 登记失败只 warn：ledger 文件被
// 目录占位导致写入失败时，.mmd 仍落盘、无 panic，生成返回不受影响
// （registerDiagramAsset 无 error 出口，失败被吞成 warn）。
func TestRegisterDiagramAssetLedgerFailureWarnOnly(t *testing.T) {
	ws := armImageHubGateWithWorkspace(t)
	// 占位：把 work ledger 路径变成目录 → record 打开文件必失败。
	ledgerPath := imageHubLedgerPath(ws, "work")
	if err := os.MkdirAll(ledgerPath, 0o755); err != nil {
		t.Fatalf("占位 ledger 目录: %v", err)
	}

	registerDiagramAsset("mindmap\nroot", "产品脑图") // 不得 panic

	entries, err := os.ReadDir(filepath.Join(ws, ".gaea", "uploads"))
	if err != nil {
		t.Fatalf("读取 uploads: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mmd") {
			found = true
		}
	}
	if !found {
		t.Fatal("登记失败时 .mmd 仍应落盘（登记是旁路）")
	}
}

// TestRegisterDiagramAssetDisarmedNoDisk 未武装（测试进程缺省态）不写盘：
// 既不落 .mmd 也不写登记文件——复刻书封/绘梦试点的运行态闸语义。
func TestRegisterDiagramAssetDisarmedNoDisk(t *testing.T) {
	ws := armImageHubGateWithWorkspace(t)
	imageHubRuntimeArmed.Store(false) // 回到未武装（闸 = 武装位 && 配置非 nil）

	registerDiagramAsset("flowchart TD\nA-->B", "不应落盘")

	if _, err := os.Stat(filepath.Join(ws, ".gaea", "uploads")); !os.IsNotExist(err) {
		t.Fatalf("未武装不应创建 uploads 目录: %v", err)
	}
	if _, err := os.Stat(imageHubLedgerPath(ws, "work")); !os.IsNotExist(err) {
		t.Fatalf("未武装不应写登记文件: %v", err)
	}
}

// TestGenerateDiagramGuardArmedNoSideEffects 武装态不改变既有错误返回：
// 客户端未初始化时 GenerateDiagram 仍返回 error 视图（生成路径行为不变），
// 且不产生任何落盘/登记（登记只在生成成功后经 registerDiagramAsset 旁路发生）。
func TestGenerateDiagramGuardArmedNoSideEffects(t *testing.T) {
	ws := armImageHubGateWithWorkspace(t)
	a := &mediaState{}
	res, err := a.GenerateDiagram("订单流程图")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res["error"] == nil {
		t.Fatal("未初始化客户端应返回错误视图")
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "uploads")); !os.IsNotExist(err) {
		t.Fatalf("生成失败不应落盘: %v", err)
	}
	if _, err := os.Stat(imageHubLedgerPath(ws, "work")); !os.IsNotExist(err) {
		t.Fatalf("生成失败不应登记: %v", err)
	}
}
