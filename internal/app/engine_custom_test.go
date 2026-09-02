package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/modelengine"
)

// ── A 刀：自定义引擎三绑定方法（App 门面 → core → engineMgr 委托链）────

// newCustomEngineTestApp 构造最小 App（engineMgr 就绪）并把 HOME 指到临时目录
// ——Add/Update/Remove 会经 saveCustomEngineKeys 写 ~/.gaea_config.json，
// 不隔离会污染真实用户配置。
func newCustomEngineTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	c := &core{engineMgr: modelengine.NewManager("", ""), cfg: &config.Config{}}
	return &App{
		core:         c,
		whisperState: &whisperState{core: &core{engineMgr: c.engineMgr}},
	}
}

// TestCustomEngineBindings_NilManager 模型引擎管理器未初始化时三方法全部报错。
func TestCustomEngineBindings_NilManager(t *testing.T) {
	a := &App{core: &core{}}
	if _, err := a.AddCustomEngine("X", "https://x.example.com/v1", "k"); err == nil {
		t.Error("AddCustomEngine(nil mgr) 应报错")
	}
	if err := a.UpdateCustomEngine("custom-x", "X", "https://x.example.com/v1", ""); err == nil {
		t.Error("UpdateCustomEngine(nil mgr) 应报错")
	}
	if err := a.RemoveCustomEngine("custom-x"); err == nil {
		t.Error("RemoveCustomEngine(nil mgr) 应报错")
	}
}

// TestAddCustomEngine_Delegation 委托链：App → core → Manager；Key 只以密文
// 落 config，Manager 内存持明文，EngineConfig.APIKey 恒空。
func TestAddCustomEngine_Delegation(t *testing.T) {
	a := newCustomEngineTestApp(t)
	const secret = "sk-app-secret-42"
	id, err := a.AddCustomEngine("My Relay", "https://relay.example.com/v1", secret)
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if !strings.HasPrefix(id, "custom-") {
		t.Fatalf("engineID = %q, want custom- 前缀", id)
	}

	// Manager 内存明文（聊天路径取用）
	if a.engineMgr.CustomEngineKey(id) != secret {
		t.Errorf("CustomEngineKey = %q, want 明文 %q", a.engineMgr.CustomEngineKey(id), secret)
	}
	// EngineConfig 不带 Key
	if eng, ok := a.engineMgr.GetEngine(id); !ok || eng.APIKey != "" {
		t.Errorf("GetEngine = (%+v, %v), want APIKey 空", eng, ok)
	}
	// config 落盘为密文：文件含键名、不含明文
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".gaea_config.json"))
	if err != nil {
		t.Fatalf("读取配置文件: %v", err)
	}
	if !strings.Contains(string(raw), "custom_engine_keys") {
		t.Error("配置文件缺少 custom_engine_keys")
	}
	if strings.Contains(string(raw), secret) {
		t.Error("配置文件泄漏自定义引擎 Key 明文")
	}
	// 内存 cfg 同步密文
	if a.cfg.CustomEngineKeys[id] == "" || a.cfg.CustomEngineKeys[id] == secret {
		t.Errorf("cfg.CustomEngineKeys[%s] = %q, want 非空密文", id, a.cfg.CustomEngineKeys[id])
	}
}

// TestUpdateCustomEngine_Delegation 空 apiKey = 保留原 Key；新 apiKey = 替换。
func TestUpdateCustomEngine_Delegation(t *testing.T) {
	a := newCustomEngineTestApp(t)
	id, err := a.AddCustomEngine("Relay", "https://old.example.com/v1", "sk-first")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}

	// 空 Key = 保留
	if err := a.UpdateCustomEngine(id, "Relay 2", "https://new.example.com/v1", ""); err != nil {
		t.Fatalf("UpdateCustomEngine(空Key): %v", err)
	}
	eng, _ := a.engineMgr.GetEngine(id)
	if eng.Label != "Relay 2" || eng.BaseURL != "https://new.example.com/v1" {
		t.Errorf("名称/地址未更新: %q/%q", eng.Label, eng.BaseURL)
	}
	if _, key, _ := a.engineMgr.BuildChatURL(id); key != "sk-first" {
		t.Errorf("空 apiKey 应保留原 Key, got %q", key)
	}

	// 新 Key 替换
	if err := a.UpdateCustomEngine(id, "Relay 2", "https://new.example.com/v1", "sk-second"); err != nil {
		t.Fatalf("UpdateCustomEngine(新Key): %v", err)
	}
	if _, key, _ := a.engineMgr.BuildChatURL(id); key != "sk-second" {
		t.Errorf("新 Key 未生效, got %q", key)
	}

	// 校验拒绝路径经委托链原样返回
	if err := a.UpdateCustomEngine("xai", "X", "https://x.example.com/v1", ""); err == nil {
		t.Error("更新内置引擎应报错")
	}
	if err := a.UpdateCustomEngine("custom-ghost", "X", "https://x.example.com/v1", ""); err == nil {
		t.Error("更新不存在的引擎应报错")
	}
	if err := a.UpdateCustomEngine(id, "Relay 2", "sk-key-as-url", ""); err == nil {
		t.Error("Key 当地址应报错")
	}
}

// TestRemoveCustomEngine_Delegation 删除经委托链生效；内置引擎拒绝。
func TestRemoveCustomEngine_Delegation(t *testing.T) {
	a := newCustomEngineTestApp(t)
	if err := a.RemoveCustomEngine("xai"); err == nil {
		t.Error("删除内置引擎应报错")
	}
	id, err := a.AddCustomEngine("Doomed", "https://doom.example.com/v1", "sk-doom")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := a.RemoveCustomEngine(id); err != nil {
		t.Fatalf("RemoveCustomEngine: %v", err)
	}
	if _, ok := a.engineMgr.GetEngine(id); ok {
		t.Error("删除后引擎仍在 Manager")
	}
	if _, ok := a.cfg.CustomEngineKeys[id]; ok {
		t.Error("删除后 cfg.CustomEngineKeys 仍含该引擎")
	}
	if a.engineMgr.CustomEngineKey(id) != "" {
		t.Error("删除后 Manager 内存 Key 未清")
	}
}

// TestDecryptCustomEngineKeys 启动注入前置步骤：密文 map → 明文 map，
// 解密失败/空值条目丢弃（保守不注入半截 Key）。
func TestDecryptCustomEngineKeys(t *testing.T) {
	enc1, err := secure.EncryptString("sk-plain-1")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	got := decryptCustomEngineKeys(map[string]string{
		"custom-a": enc1,
		"custom-b": "dpapi:not-a-valid-blob!!", // 解密失败 → 丢弃
		"custom-c": "",                        // 空 → 丢弃
	})
	if got["custom-a"] != "sk-plain-1" {
		t.Errorf("custom-a = %q, want sk-plain-1", got["custom-a"])
	}
	if _, ok := got["custom-b"]; ok {
		t.Error("解密失败条目应被丢弃")
	}
	if _, ok := got["custom-c"]; ok {
		t.Error("空值条目应被丢弃")
	}
	if decryptCustomEngineKeys(nil) == nil || len(decryptCustomEngineKeys(nil)) != 0 {
		t.Error("nil 输入应返回空 map（非 nil），供 SetCustomEngineKeys 清空语义")
	}
}
