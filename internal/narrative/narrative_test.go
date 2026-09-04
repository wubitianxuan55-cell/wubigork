package narrative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mkPatch builds a structurally valid patch with the given upserts.
func mkPatch(id string, chapter int, upserts ...EntityState) StatePatch {
	return StatePatch{
		ID:         id,
		Chapter:    chapter,
		Branch:     "main",
		Reason:     "test patch " + id,
		Upserts:    upserts,
		ProposedBy: "ai",
	}
}

func mkEntity(id, name, status string) EntityState {
	return EntityState{
		ID:         id,
		Name:       name,
		Type:       "character",
		Status:     status,
		Items:      []string{"sword"},
		KnownBy:    []string{"narrator"},
		Properties: map[string]string{"trait": "brave"},
	}
}

func TestApplyPatch_CloneIsSafe(t *testing.T) {
	base := &StateSnapshot{
		Version: 0,
		Entities: map[string]EntityState{
			"char-a": mkEntity("char-a", "阿廖沙", "alive"),
		},
	}

	patch := mkPatch("p-1", 1,
		mkEntity("char-a", "阿廖沙", "dead"), // update
		mkEntity("char-b", "芙兰", "alive"), // insert
	)

	newSnap, err := base.ApplyPatch(patch)
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}

	// Source snapshot must be untouched.
	src := base.Entities["char-a"]
	if src.Status != "alive" {
		t.Fatalf("source mutated: status=%q want alive", src.Status)
	}
	if len(src.Items) != 1 || src.Items[0] != "sword" {
		t.Fatalf("source items mutated: %v", src.Items)
	}
	if len(src.KnownBy) != 1 || src.KnownBy[0] != "narrator" {
		t.Fatalf("source known_by mutated: %v", src.KnownBy)
	}
	if src.Properties["trait"] != "brave" {
		t.Fatalf("source properties mutated: %v", src.Properties)
	}

	// New snapshot reflects the patch and version moved forward.
	if newSnap.Version != 1 {
		t.Fatalf("version = %d, want 1", newSnap.Version)
	}
	if got := newSnap.Entities["char-a"].Status; got != "dead" {
		t.Fatalf("new entity status = %q, want dead", got)
	}
	if _, ok := newSnap.Entities["char-b"]; !ok {
		t.Fatalf("new entity char-b missing")
	}

	// Deep-copy independence: mutating the returned snapshot must not reach the source.
	newSnap.Entities["char-a"].Items[0] = "XX"
	newSnap.Entities["char-a"].KnownBy[0] = "YY"
	newSnap.Entities["char-a"].Properties["trait"] = "ZZ"
	if base.Entities["char-a"].Items[0] != "sword" {
		t.Fatalf("source items shared backing array with result")
	}
	if base.Entities["char-a"].KnownBy[0] != "narrator" {
		t.Fatalf("source known_by shared backing array with result")
	}
	if base.Entities["char-a"].Properties["trait"] != "brave" {
		t.Fatalf("source properties map shared with result")
	}

	// Mutating the patch upserts after apply must not reach the snapshot.
	patch.Upserts[0].Items[0] = "YY"
	if newSnap.Entities["char-a"].Items[0] != "XX" {
		t.Fatalf("snapshot shares reference with patch upserts")
	}
}

func TestValidateStatePatch(t *testing.T) {
	valid := mkPatch("p-1", 1, mkEntity("char-a", "阿廖沙", "alive"))
	if err := ValidateStatePatch(valid); err != nil {
		t.Fatalf("valid patch rejected: %v", err)
	}

	cases := []struct {
		name  string
		patch StatePatch
	}{
		{"empty upsert id", func() StatePatch {
			p := mkPatch("p-1", 1, mkEntity("", "阿廖沙", "alive"))
			return p
		}()},
		{"empty upsert name", func() StatePatch {
			p := mkPatch("p-1", 1)
			p.Upserts = []EntityState{{ID: "char-a", Name: "", Status: "alive"}}
			return p
		}()},
		{"negative chapter", func() StatePatch {
			return mkPatch("p-1", -3, mkEntity("char-a", "阿廖沙", "alive"))
		}()},
		{"zero chapter", func() StatePatch {
			return mkPatch("p-1", 0, mkEntity("char-a", "阿廖沙", "alive"))
		}()},
		{"bad foreshadow action", func() StatePatch {
			p := mkPatch("p-1", 1, mkEntity("char-a", "阿廖沙", "alive"))
			p.Foreshadow = []PatchForeshadow{{Category: "plot", Description: "x", Action: "buried"}}
			return p
		}()},
		{"empty proposed by", func() StatePatch {
			p := mkPatch("p-1", 1, mkEntity("char-a", "阿廖沙", "alive"))
			p.ProposedBy = ""
			return p
		}()},
		{"empty delete id", func() StatePatch {
			p := mkPatch("p-1", 1, mkEntity("char-a", "阿廖沙", "alive"))
			p.Deletes = []string{""}
			return p
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateStatePatch(tc.patch); err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}
		})
	}
}

func TestJournalAppendReplayRoundTrip(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	p1 := mkPatch("p-1", 1, mkEntity("char-a", "阿廖沙", "alive"))
	p2 := mkPatch("p-2", 2, mkEntity("char-b", "芙兰", "missing"))
	p3 := mkPatch("p-3", 3, mkEntity("char-a", "阿廖沙", "dead"))

	for _, p := range []StatePatch{p1, p2, p3} {
		if err := j.Append(p); err != nil {
			t.Fatalf("Append %q: %v", p.ID, err)
		}
	}

	replay, err := j.Replay()
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(replay) != 3 {
		t.Fatalf("Replay length = %d, want 3", len(replay))
	}
	for i, want := range []string{"p-1", "p-2", "p-3"} {
		if replay[i].ID != want {
			t.Fatalf("Replay[%d].ID = %q, want %q", i, replay[i].ID, want)
		}
	}
	if replay[2].Upserts[0].Status != "dead" {
		t.Fatalf("last patch upsert status = %q, want dead", replay[2].Upserts[0].Status)
	}

	snap, err := j.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Version != 3 {
		t.Fatalf("Snapshot version = %d, want 3", snap.Version)
	}
	if got := snap.Entities["char-a"].Status; got != "dead" {
		t.Fatalf("char-a status = %q, want dead", got)
	}
	if got := snap.Entities["char-b"].Status; got != "missing" {
		t.Fatalf("char-b status = %q, want missing", got)
	}
}

func TestAuthorizeAndSettle_RequiresApproval(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	p := mkPatch("p-1", 1, mkEntity("char-c", "列夫", "alive"))

	// Not approved: AI suggestion must not reach the ledger or state.
	snap, err := AuthorizeAndSettle(j, p, false)
	if err != nil {
		t.Fatalf("AuthorizeAndSettle(false): %v", err)
	}
	replay, err := j.Replay()
	if err != nil {
		t.Fatalf("Replay after reject: %v", err)
	}
	if len(replay) != 0 {
		t.Fatalf("journal length after reject = %d, want 0", len(replay))
	}
	if _, ok := snap.Entities["char-c"]; ok {
		t.Fatalf("rejected patch leaked into snapshot")
	}
	if _, serr := os.Stat(filepath.Join(j.dir, stateName)); !os.IsNotExist(serr) {
		t.Fatalf("state.json should not exist after reject")
	}

	// Approved: only now does it settle.
	snap2, err := AuthorizeAndSettle(j, p, true)
	if err != nil {
		t.Fatalf("AuthorizeAndSettle(true): %v", err)
	}
	replay2, err := j.Replay()
	if err != nil {
		t.Fatalf("Replay after approve: %v", err)
	}
	if len(replay2) != 1 {
		t.Fatalf("journal length after approve = %d, want 1", len(replay2))
	}
	if got := snap2.Entities["char-c"].Status; got != "alive" {
		t.Fatalf("char-c status = %q, want alive", got)
	}

	// state.json must now exist and match the snapshot.
	raw, err := os.ReadFile(filepath.Join(j.dir, stateName))
	if err != nil {
		t.Fatalf("state.json not written: %v", err)
	}
	var onDisk StateSnapshot
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("state.json unmarshal: %v", err)
	}
	if onDisk.Version != snap2.Version {
		t.Fatalf("state.json version = %d, want %d", onDisk.Version, snap2.Version)
	}
	if _, ok := onDisk.Entities["char-c"]; !ok {
		t.Fatalf("state.json missing char-c")
	}
}

func TestSnapshot_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	snap, err := j.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot on empty dir: %v", err)
	}
	if snap.Version != 0 {
		t.Fatalf("version = %d, want 0", snap.Version)
	}
	if len(snap.Entities) != 0 {
		t.Fatalf("entities len = %d, want 0", len(snap.Entities))
	}

	// A base dir that does not yet exist is also fine: Open creates the tree.
	missing := filepath.Join(t.TempDir(), "not", "created", "yet")
	j2, err := Open(missing)
	if err != nil {
		t.Fatalf("Open on missing tree: %v", err)
	}
	snap2, err := j2.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot on created tree: %v", err)
	}
	if snap2.Version != 0 || len(snap2.Entities) != 0 {
		t.Fatalf("expected empty snapshot, got version=%d entities=%d", snap2.Version, len(snap2.Entities))
	}
}
