package app

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// TestGaeaBootBuild 验证办公引擎核心链路：bridge provider 注册 + 配置装载 +
// boot.Build 构建单模型控制器。
// 对应 GaeaInit 的核心逻辑（跳过 Wails runtime 的 emit 路径，避免
// runtime.EventsEmit 在非 Wails 上下文 log.Fatal）。
// chdir 到临时目录，避免会话/归档污染仓库（cwd/.gaea）。
func TestGaeaBootBuild(t *testing.T) {
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	_ = os.Chdir(t.TempDir())

	bridge.SetClient(ai.NewClient(config.Load()))
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:          "gaea",
		Kind:          "wubigrok",
		Model:         "",
		ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	// 恢复全局 loader，避免污染同包后续测试（如 boot.Build 读到注入配置而失败）
	defer gaeaConfig.SetLoader(nil)

	// 与 GaeaInit 保持一致：SessionDir 指向工作区会话目录（cwd/.gaea/sessions）
	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		Model:      "gaea",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		MaxSteps:   0,
		SessionDir: gaeaConfig.WorkspaceSessionDir(""),
	})
	// GaeaInit 同样在构建后启用交互审批（工具审批/提问走前端）
	ctrl.EnableInteractiveApproval()
	if err != nil {
		t.Fatalf("办公引擎构建失败: %v", err)
	}
	if ctrl == nil {
		t.Fatal("办公控制器为 nil")
	}
	defer ctrl.Close()

	// 控制器必须能通过 NewSession 创建会话（对应办公会话持久化链路）
	if err := ctrl.NewSession(); err != nil {
		t.Fatalf("办公控制器 NewSession 失败: %v", err)
	}
	if ctrl.SessionPath() == "" {
		t.Fatal("办公控制器会话路径仍为空")
	}
	// 会话必须落在工作区会话目录（与 GaeaListSessions 读取路径一致）
	wd, _ := os.Getwd()
	wantDir := gaeaConfig.WorkspaceSessionDir(wd)
	if gotDir := filepath.Dir(ctrl.SessionPath()); gotDir != wantDir {
		t.Fatalf("会话目录不一致: got=%s want=%s（历史面板将无法看到会话）", gotDir, wantDir)
	}
	t.Logf("办公控制器构建成功: session=%s", ctrl.SessionPath())
}

// TestGaeaAttachmentRoundTrip 验证粘贴图片/附件的保存与读取链路
// （Composer 粘贴图 → SavePastedImage → AttachmentDataURL 渲染）。
func TestGaeaAttachmentRoundTrip(t *testing.T) {
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	_ = os.Chdir(t.TempDir())

	a := &App{}

	// 粘贴图片：dataURL 保存后应能原样读回
	pngB := []byte("fake-png-bytes")
	pngURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngB)
	path, err := a.GaeaSavePastedImage(pngURL)
	if err != nil {
		t.Fatalf("保存粘贴图失败: %v", err)
	}
	if !strings.HasSuffix(path, ".png") {
		t.Fatalf("粘贴图扩展名错误: %s", path)
	}
	gotURL, err := a.GaeaAttachmentDataURL(path)
	if err != nil {
		t.Fatalf("读取粘贴图失败: %v", err)
	}
	if gotURL != pngURL {
		t.Fatalf("粘贴图往返不一致: got=%s want=%s", gotURL, pngURL)
	}

	// 附件：base64 内容保存后应能按 mime 类型读回
	txt := "hello 附件"
	apath, err := a.GaeaSaveAttachmentFile("note.txt", base64.StdEncoding.EncodeToString([]byte(txt)))
	if err != nil {
		t.Fatalf("保存附件失败: %v", err)
	}
	aurl, err := a.GaeaAttachmentDataURL(apath)
	if err != nil {
		t.Fatalf("读取附件失败: %v", err)
	}
	if !strings.HasPrefix(aurl, "data:text/plain;base64,") {
		t.Fatalf("附件 mime 错误: %s", aurl)
	}
	dec, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(aurl, "data:text/plain;base64,"))
	if err != nil || string(dec) != txt {
		t.Fatalf("附件内容不一致: %q err=%v", dec, err)
	}
}
