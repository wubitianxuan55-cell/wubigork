package narrative

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/util"
)

// journalName is the append-only patch ledger within the narrative directory.
const journalName = "journal.jsonl"

// stateName is the latest snapshot file written at settlement time.
const stateName = "state.json"

// Journal is an append-only ledger of state patches backed by files under
// <dir>/.gaea/narrative/. It supports replaying the ledger to rebuild the latest
// StateSnapshot.
type Journal struct {
	dir string
}

// Open creates the .gaea/narrative/ directory under dir and returns a Journal
// bound to it. The base dir is created (recursively) if needed. Open never fails
// merely because the journal files do not exist yet.
func Open(dir string) (*Journal, error) {
	if dir == "" {
		return nil, errors.New("narrative: open: empty dir")
	}
	journalDir := filepath.Join(dir, ".gaea", "narrative")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		return nil, fmt.Errorf("narrative: mkdir %s: %w", journalDir, err)
	}
	return &Journal{dir: journalDir}, nil
}

// journalPath returns the ledger path for this journal.
func (j *Journal) journalPath() string {
	return filepath.Join(j.dir, journalName)
}

// Append adds one validated patch to the ledger. It writes the whole ledger
// atomically (temp file + rename) so a reader never sees a partial line, and
// best-effort syncs the file and directory.
func (j *Journal) Append(patch StatePatch) error {
	if j == nil {
		return errors.New("narrative: append: nil journal")
	}
	if err := ValidateStatePatch(patch); err != nil {
		return err
	}
	line, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("narrative: marshal patch %q: %w", patch.ID, err)
	}
	return j.appendLine(line)
}

// appendLine appends a single JSON line (already marshaled) to the ledger using
// an atomic rewrite of the whole file.
func (j *Journal) appendLine(line []byte) error {
	path := j.journalPath()
	var existing []byte
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		existing = b
	case errors.Is(err, os.ErrNotExist):
		existing = nil
	default:
		return fmt.Errorf("narrative: read journal for append: %w", err)
	}
	if n := len(existing); n > 0 && existing[n-1] != '\n' {
		existing = append(existing, '\n')
	}
	data := append(existing, line...)
	if len(line) > 0 && line[len(line)-1] != '\n' {
		data = append(data, '\n')
	}
	return writeFileAtomic(path, data, 0o644)
}

// Replay reads all patches from the ledger in order, skipping corrupt lines with
// a warning.
func (j *Journal) Replay() ([]StatePatch, error) {
	return j.Collect(j.dir)
}

// Collect is the directory-based alias for Replay: it reads every patch stored as
// a JSON line under dir. When dir is empty it defaults to this journal's own
// directory. A missing ledger yields nil (empty), never an error.
func (j *Journal) Collect(dir string) ([]StatePatch, error) {
	if dir == "" {
		dir = j.dir
	}
	if j == nil {
		return nil, nil
	}
	path := filepath.Join(dir, journalName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("narrative: read journal %s: %w", path, err)
	}
	var patches []StatePatch
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 {
			continue
		}
		var p StatePatch
		if uerr := json.Unmarshal(line, &p); uerr != nil {
			slog.Warn("narrative: skip corrupt journal line",
				"error", uerr,
				"line", util.Truncate(string(line), 200))
			continue
		}
		patches = append(patches, p)
	}
	return patches, nil
}

// Snapshot rebuilds the latest StateSnapshot by replaying the journal in order.
// A missing or empty ledger yields an empty snapshot (Version 0, Entities {})
// with no error.
func (j *Journal) Snapshot() (*StateSnapshot, error) {
	patches, err := j.Replay()
	if err != nil {
		return nil, err
	}
	snap := &StateSnapshot{Version: 0, Entities: map[string]EntityState{}}
	for _, p := range patches {
		next, aerr := snap.ApplyPatch(p)
		if aerr != nil {
			slog.Warn("narrative: skip patch during replay",
				"patch_id", p.ID,
				"error", aerr)
			continue
		}
		snap = next
	}
	return snap, nil
}

// AuthorizeAndSettle is the only path that lets a patch into the canonical state.
// An approved patch (approved == true) is appended to the ledger and state.json
// is rewritten. A rejected patch (approved == false) is dropped: it is only
// logged, the ledger is untouched, and the returned snapshot is unchanged. This
// enforces the project discipline that an AI has suggestion rights only.
func AuthorizeAndSettle(j *Journal, patch StatePatch, approved bool) (*StateSnapshot, error) {
	if j == nil {
		return nil, errors.New("narrative: settle: nil journal")
	}
	if err := ValidateStatePatch(patch); err != nil {
		return nil, err
	}
	if !approved {
		slog.Info("narrative: patch NOT authorized, dropped without settling",
			"patch_id", patch.ID,
			"chapter", patch.Chapter,
			"proposed_by", patch.ProposedBy)
		return j.Snapshot()
	}
	if err := j.Append(patch); err != nil {
		return nil, err
	}
	snap, err := j.Snapshot()
	if err != nil {
		return nil, err
	}
	payload, merr := marshalSnapshot(snap)
	if merr != nil {
		return nil, merr
	}
	if werr := writeFileAtomic(filepath.Join(j.dir, stateName), payload, 0o644); werr != nil {
		return nil, werr
	}
	return snap, nil
}

// marshalSnapshot renders the snapshot as indented JSON for state.json.
func marshalSnapshot(s *StateSnapshot) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("narrative: marshal snapshot: %w", err)
	}
	return b, nil
}

// writeFileAtomic writes data to path atomically: it writes to a temporary
// sibling file and renames it over the target. Readers never see a partial
// write. This mirrors the project-wide fileutil.AtomicWrite pattern without
// importing that package (the narrative package is intentionally dependency
// minimal). File and directory sync are best-effort.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("narrative: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("narrative: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("narrative: write temp: %w", err)
	}
	flushErr := tmp.Sync() // best-effort fsync
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("narrative: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("narrative: rename %s: %w", path, err)
	}
	if d, derr := os.Open(dir); derr == nil && flushErr == nil {
		_ = d.Sync() // best-effort directory sync
		_ = d.Close()
	}
	return nil
}
