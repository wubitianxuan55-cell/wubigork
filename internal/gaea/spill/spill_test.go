package spill

import (
	"context"
	"strings"
	"testing"
)

func sizedData(n int) string { return strings.Repeat("x", n) }

// Save/Retrieve 全链路：分页、剩余字节、offset 越界、未知 id 如实报错。
func TestStoreSaveRetrievePaging(t *testing.T) {
	s := NewStore()
	data := sizedData(30 * 1024)
	id := s.Save("bash", data)
	if id == "" {
		t.Fatal("Save must return a locator for in-range data")
	}

	chunk, total, remaining, err := s.Retrieve(id, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != len(data) || len(chunk) != ReadDefaultBytes {
		t.Fatalf("first page: got %d bytes of %d total, want default page", len(chunk), total)
	}
	if remaining != len(data)-len(chunk) {
		t.Fatalf("remaining = %d, want %d", remaining, len(data)-len(chunk))
	}
	if chunk != data[:len(chunk)] {
		t.Fatal("first page content mismatch")
	}

	// 中段分页：offset+remaining 收尾页。
	tail, _, rem2, err := s.Retrieve(id, len(data)-100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rem2 != 0 || tail != data[len(data)-100:] {
		t.Fatalf("tail page mismatch (rem=%d)", rem2)
	}

	// offset 越界与未知 id 都是显式错误。
	if _, _, _, err := s.Retrieve(id, len(data), 0); err == nil {
		t.Fatal("offset == total must error")
	}
	if _, _, _, err := s.Retrieve("sp-999999", 0, 0); err == nil {
		t.Fatal("unknown id must error")
	}
}

// 总预算 FIFO 驱逐：最旧条目先出，locator 过期如实报错。
func TestStoreEvictsOldestOnBudget(t *testing.T) {
	old := TotalBudget
	TotalBudget = 40 * 1024
	defer func() { TotalBudget = old }()

	s := NewStore()
	id1 := s.Save("bash", sizedData(30*1024))
	id2 := s.Save("bash", sizedData(30*1024))
	if id1 == "" || id2 == "" {
		t.Fatal("both saves must succeed")
	}
	// id1 已被驱逐（30K+30K > 40K 预算，FIFO 出最旧）。
	if _, _, _, err := s.Retrieve(id1, 0, 0); err == nil {
		t.Fatal("oldest entry must be evicted")
	}
	if _, _, _, err := s.Retrieve(id2, 0, 0); err != nil {
		t.Fatalf("newest entry must survive: %v", err)
	}
}

// 单条超限拒收：不泄洪（返回 ""），既有截断管线照旧。
func TestStoreRefusesOversizedEntry(t *testing.T) {
	s := NewStore()
	if id := s.Save("bash", sizedData(MaxEntryBytes+1)); id != "" {
		t.Fatalf("oversized entry must be refused, got %q", id)
	}
	if id := s.Save("bash", sizedData(MaxEntryBytes)); id == "" {
		t.Fatal("entry exactly at the cap must be stored")
	}
}

// ctx 盖章往返；nil 库不盖章。
func TestStoreContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	if _, ok := FromContext(ctx); ok {
		t.Fatal("unstamped ctx must yield no store")
	}
	if _, ok := FromContext(WithStore(ctx, nil)); ok {
		t.Fatal("nil store must not be stamped")
	}
	s := NewStore()
	stamped, ok := FromContext(WithStore(ctx, s))
	if !ok || stamped != s {
		t.Fatal("stamped store round-trip failed")
	}
}
