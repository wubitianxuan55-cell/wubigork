package plugin

// B8（codex 蒸馏第四批 P9）目录缓存验收：
// 指纹=spec 身份（键序归一、改配置换键）；缓存 TTL/容量逐出；
// 命中重建的 remoteTool 绑定新连接（不残留旧 client）。

import (
	"testing"
	"time"
)

func TestSpecFingerprintStableAndSensitive(t *testing.T) {
	base := Spec{Name: "srv", Type: "stdio", Command: "npx", Args: []string{"-y", "x"},
		Env: map[string]string{"A": "1", "B": "2"}}
	// env 键序不同 = 同一 spec（语义相同命中同键）。
	shuffled := Spec{Name: "srv", Type: "stdio", Command: "npx", Args: []string{"-y", "x"},
		Env: map[string]string{"B": "2", "A": "1"}}
	if SpecFingerprint(base) != SpecFingerprint(shuffled) {
		t.Error("env 键序不同应命中同一指纹")
	}
	// args 保序：顺序不同 = 不同 spec。
	reordered := Spec{Name: "srv", Type: "stdio", Command: "npx", Args: []string{"x", "-y"},
		Env: map[string]string{"A": "1", "B": "2"}}
	if SpecFingerprint(base) == SpecFingerprint(reordered) {
		t.Error("args 顺序语义不同，指纹应不同")
	}
	// 改任一字段（command/url/headers/dir）换键。
	for _, mutate := range []func(*Spec){
		func(s *Spec) { s.Command = "other" },
		func(s *Spec) { s.URL = "http://x" },
		func(s *Spec) { s.Dir = "C:/proj" },
		func(s *Spec) { s.Headers = map[string]string{"k": "v"} },
	} {
		m := base
		mutate(&m)
		if SpecFingerprint(base) == SpecFingerprint(m) {
			t.Error("spec 字段变化未换指纹键")
		}
	}
	// Name 不参与指纹：改名重连=同目录形状（服务器还是那个进程形态）。
	renamed := base
	renamed.Name = "srv2"
	if SpecFingerprint(base) != SpecFingerprint(renamed) {
		t.Error("改名不应换指纹（目录形状相同）")
	}
}

func TestCatalogTTLAndCapacity(t *testing.T) {
	catalogReset()
	fp := SpecFingerprint(Spec{Command: "a"})
	tools := []mcpTool{{Name: "t1", Description: "d"}}
	catalogPut(fp, tools)
	if got, ok := catalogGet(fp); !ok || len(got) != 1 {
		t.Fatalf("fresh entry missing: %v %v", got, ok)
	}
	// 过期（伪造 at）。
	catalog.mu.Lock()
	e := catalog.m[fp]
	e.at = time.Now().Add(-catalogTTL - time.Minute)
	catalog.m[fp] = e
	catalog.mu.Unlock()
	if _, ok := catalogGet(fp); ok {
		t.Error("过期条目应 miss")
	}
	// 容量逐出：塞满 32+ 再放一条，最旧被逐。
	catalogReset()
	for i := 0; i < catalogMaxEntries+5; i++ {
		catalogPut(SpecFingerprint(Spec{Command: "c", Args: []string{string(rune('a' + i%26)), string(rune('a' + i%26)), string(rune('0' + i%10))}}),
			[]mcpTool{{Name: "x"}})
		time.Sleep(time.Millisecond) // 保证 at 可区分
	}
	if got := CatalogStats(); got > catalogMaxEntries {
		t.Errorf("entries = %d, want <= %d（满逐出）", got, catalogMaxEntries)
	}
}

// 命中缓存重建：remoteTool 名字/描述/schema 与缓存形状一致，且 client 是
// 新连接（不残留旧 client 指针）。
func TestToolsFromCatalogRebindsNewClient(t *testing.T) {
	c1 := &Client{name: "srv"}
	c2 := &Client{name: "srv"}
	raw := []mcpTool{
		{Name: "echo", Description: "回声",
			InputSchema: []byte(`{"type":"object"}`)},
	}
	tools := c1.toolsFromCatalog(raw)
	if len(tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(tools))
	}
	rt, ok := tools[0].(*remoteTool)
	if !ok {
		t.Fatalf("tool type = %T", tools[0])
	}
	if rt.client != c1 {
		t.Error("rebind 失败：client 指针不符")
	}
	// 同一目录形状给新连接重建 → 绑定新 client。
	tools2 := c2.toolsFromCatalog(raw)
	rt2 := tools2[0].(*remoteTool)
	if rt2.client != c2 {
		t.Error("第二次重建仍绑旧 client")
	}
	if rt2.name != rt.name || rt2.desc != rt.desc {
		t.Errorf("重建后形状漂移: %v vs %v", rt2.name, rt.name)
	}
}
