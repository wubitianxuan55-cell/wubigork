package builtin

import (
	"strings"
	"testing"
)

// saveMarkdownRuntime 快照并恢复包级 markdownConverterRuntime（config 注入）。
func saveMarkdownRuntime(t *testing.T) {
	t.Helper()
	old := markdownConverterRuntime
	t.Cleanup(func() { markdownConverterRuntime = old })
}

// TestMarkdownConverterRegistry_AllKinds "cli" kind 经注册表构建为
// *markitdownCLIConverter，kind 列表完整且排序。
func TestMarkdownConverterRegistry_AllKinds(t *testing.T) {
	kinds := MarkdownConverterKinds()
	if len(kinds) != 1 || kinds[0] != MarkdownConverterKindCLI {
		t.Fatalf("MarkdownConverterKinds = %v, want [cli]", kinds)
	}
	c, err := NewMarkdownConverter(MarkdownConverterKindCLI, MarkdownConverterConfig{})
	if err != nil {
		t.Fatalf("NewMarkdownConverter(cli): %v", err)
	}
	if _, ok := c.(*markitdownCLIConverter); !ok {
		t.Fatalf("kind=cli 应返回 *markitdownCLIConverter, got %T", c)
	}
}

// TestMarkdownConverterRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同实现：
// 切换文档转换后端只改 kind，消费方（MarkdownConverter 接口）零改动。
func TestMarkdownConverterRegistry_ConfigRouting(t *testing.T) {
	var consumer func(kind string) (MarkdownConverter, error)
	consumer = func(kind string) (MarkdownConverter, error) {
		return NewMarkdownConverter(kind, MarkdownConverterConfig{})
	}
	c, err := consumer(MarkdownConverterKindCLI)
	if err != nil {
		t.Fatalf("consumer(cli): %v", err)
	}
	if _, ok := c.(*markitdownCLIConverter); !ok {
		t.Errorf("consumer(cli) 应返回 *markitdownCLIConverter, got %T", c)
	}
}

// TestMarkdownConverterRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestMarkdownConverterRegistry_UnknownKindError(t *testing.T) {
	_, err := NewMarkdownConverter("no-such-converter", MarkdownConverterConfig{})
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), MarkdownConverterKindCLI) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestMarkdownConverterRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestMarkdownConverterRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterMarkdownConverter(MarkdownConverterKindCLI, func(cfg MarkdownConverterConfig) (MarkdownConverter, error) { return nil, nil })
}

// TestMarkdownConverterRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestMarkdownConverterRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterMarkdownConverter("", func(cfg MarkdownConverterConfig) (MarkdownConverter, error) { return nil, nil })
}

// TestMarkdownConverterKind_DefaultAndConfigured kind 选择：空配置回落默认 cli
// （与旧 read_file 行为一致），注入后走注入 kind。
func TestMarkdownConverterKind_DefaultAndConfigured(t *testing.T) {
	saveMarkdownRuntime(t)
	if got := markdownConverterKind(); got != MarkdownConverterKindCLI {
		t.Errorf("默认 kind = %q, want cli", got)
	}
	SetMarkdownConverterRuntime(MarkdownConverterRuntime{Kind: "other-kind"})
	if got := markdownConverterKind(); got != "other-kind" {
		t.Errorf("注入 kind = %q, want other-kind", got)
	}
	SetMarkdownConverterRuntime(MarkdownConverterRuntime{Kind: ""})
	if got := markdownConverterKind(); got != MarkdownConverterKindCLI {
		t.Errorf("清空后 kind = %q, want cli", got)
	}
}

// TestMarkdownCLIConverter_ExtensionGate 扩展名白名单与旧实现一致：
// 支持格式进入转换流程（无 markitdown 时返回"未安装"错误而非"不支持格式"），
// 不支持格式直接拒绝。
func TestMarkdownCLIConverter_ExtensionGate(t *testing.T) {
	conv := &markitdownCLIConverter{}
	// 支持格式：即使 markitdown 未安装，错误也应是"未安装"而非"不支持格式"。
	if _, err := conv.Convert(t.Context(), "doc.pdf"); err == nil || strings.Contains(err.Error(), "unsupported") {
		t.Errorf("支持格式应进入转换流程（markitdown 未安装 → 安装错误），got %v", err)
	}
	// 不支持格式：直接拒绝。
	if _, err := conv.Convert(t.Context(), "file.zzz"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("不支持格式应报 unsupported, got %v", err)
	}
}
