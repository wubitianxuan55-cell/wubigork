package pricefeed

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func newTestStoreDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return Open(gdb)
}

func TestStoreSourcesCRUD(t *testing.T) {
	s := newTestStoreDB(t)
	if !s.Available() {
		t.Fatal("store unavailable")
	}
	src := Source{
		ID: "src-1", Name: "四川信息价", URL: "http://x/pricelist.aspx?period=758",
		Parser: "sc_table", FrequencyHours: 24, Area: "成都市区",
		Headers: map[string]string{"Cookie": "a=b"}, Enabled: true,
	}
	if err := s.SaveSource(src); err != nil {
		t.Fatal(err)
	}
	got, ok := s.GetSource("src-1")
	if !ok || got.Name != "四川信息价" || !got.Enabled || got.FrequencyHours != 24 || got.Headers["Cookie"] != "a=b" {
		t.Errorf("GetSource = %+v ok=%v", got, ok)
	}
	_ = s.TouchSource("src-1", "2026-08-10T00:00:00Z")
	if got, _ := s.GetSource("src-1"); got.LastFetchAt == "" {
		t.Error("last_fetch_at not updated")
	}
	if len(s.ListSources()) != 1 {
		t.Error("ListSources len != 1")
	}
	if err := s.DeleteSource("src-1"); err != nil {
		t.Fatal(err)
	}
	if len(s.ListSources()) != 0 {
		t.Error("source not deleted")
	}
}

func TestStoreFetchAndHistory(t *testing.T) {
	s := newTestStoreDB(t)
	rec := FetchRecord{
		ID: "fetch-1", SourceID: "src-1", SourceName: "四川信息价",
		URL: "http://x/list?period=758", Period: "758", Status: "pending",
		Candidates: []Candidate{{Title: "螺纹钢", Price: 3420, Status: "更新", ExistingName: "rebar"}},
	}
	if err := s.SaveFetch(rec); err != nil {
		t.Fatal(err)
	}
	list := s.ListFetches(10)
	if len(list) != 1 || list[0].Candidates[0].Title != "螺纹钢" {
		t.Fatalf("ListFetches = %+v", list)
	}
	if err := s.SetFetchStatus("fetch-1", "applied"); err != nil {
		t.Fatal(err)
	}
	if s.ListFetches(10)[0].Status != "applied" {
		t.Error("status not updated")
	}

	if err := s.AddHistory(History{Name: "rebar", Title: "螺纹钢", Unit: "t", Price: 3420, Source: "四川信息价", Period: "758"}); err != nil {
		t.Fatal(err)
	}
	h := s.ListHistory("rebar", 10)
	if len(h) != 1 || h[0].Price != 3420 || h[0].Period != "758" {
		t.Errorf("ListHistory = %+v", h)
	}
}

func TestStoreUnavailable(t *testing.T) {
	s := Open(nil)
	if s.Available() {
		t.Error("nil db should be unavailable")
	}
	if s.ListSources() != nil {
		t.Error("ListSources on nil db should be nil")
	}
	if err := s.SaveSource(Source{URL: "http://x"}); err == nil {
		t.Error("SaveSource on nil db should error")
	}
}
