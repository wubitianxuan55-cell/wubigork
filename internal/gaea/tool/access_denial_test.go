package tool

import (
	"context"
	"strings"
	"testing"
)

// A7 诚实读拒绝（Reasonix 2.x target_access 蒸馏）测试：拒绝消息列出具体
// 允许根 / 空集合明说 / 超量根诚实省略 / 出路指引在场。

func TestDenialListsConcreteRoots(t *testing.T) {
	p := NewPathAccessPolicy("/gws-a", []string{"/gws-b", "/gws-c"}, []string{"/gws-out"}, nil)
	err := p.CheckRead(context.Background(), "/etc/passwd")
	if err == nil {
		t.Fatal("越界读应拒绝")
	}
	msg := err.Error()
	for _, want := range []string{"/etc/passwd", "gws-a", "gws-b", "gws-c", "allow_read", "select the file in the UI"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("拒绝消息应含 %q: %s", want, msg)
		}
	}
}

func TestDenialWriteListsWritableRoots(t *testing.T) {
	p := NewPathAccessPolicy("/gws-a", []string{"/gws-b"}, []string{"/gws-out"}, nil)
	err := p.CheckWrite(context.Background(), "/etc/hosts")
	if err == nil {
		t.Fatal("越界写应拒绝")
	}
	msg := err.Error()
	if !strings.Contains(msg, "gws-out") || !strings.Contains(msg, "gws-a") {
		t.Fatalf("写拒绝应列出可写根: %s", msg)
	}
	if strings.Contains(msg, "gws-b") {
		t.Fatalf("只读根不应混进可写根清单: %s", msg)
	}
}

func TestDenialNoRootsConfiguredSaysSo(t *testing.T) {
	p := NewPathAccessPolicy("", nil, nil, nil)
	err := p.CheckRead(context.Background(), "/anywhere")
	if err == nil {
		t.Fatal("无根配置应拒绝一切")
	}
	if !strings.Contains(err.Error(), "(none configured)") {
		t.Fatalf("空根清单应明说: %s", err.Error())
	}
}

func TestRenderRootsTruncatesHonestly(t *testing.T) {
	roots := []string{"/a", "/b", "/c", "/d", "/e", "/f", "/g", "/h", "/i", "/j"}
	out := renderRoots(roots)
	if !strings.Contains(out, `"/h"`) || strings.Contains(out, `"/i"`) {
		t.Fatalf("应列前 8 个: %s", out)
	}
	if !strings.Contains(out, "+2 more") {
		t.Fatalf("省略应带计数: %s", out)
	}
}
