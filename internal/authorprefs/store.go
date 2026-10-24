package authorprefs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

const storeVersion = 1

var idSeq atomic.Uint64

type levelStore struct {
	mu     sync.Mutex
	root   string
	prefix string
}

func openLevel(root, prefix string) (*levelStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("author-memory root is empty")
	}
	dir := filepath.Join(root, ".gaea", "author-memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &levelStore{root: dir, prefix: prefix}, nil
}

func (s *levelStore) file() string { return filepath.Join(s.root, "memory.json") }

func (s *levelStore) load() (Snapshot, error) {
	snap := Snapshot{Version: storeVersion, Preferences: []Preference{}, AppliedEvents: map[string]string{}}
	raw, err := os.ReadFile(s.file())
	if errors.Is(err, os.ErrNotExist) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("decode author memory: %w", err)
	}
	if snap.Version <= 0 {
		snap.Version = storeVersion
	}
	if snap.AppliedEvents == nil {
		snap.AppliedEvents = map[string]string{}
	}
	if snap.Preferences == nil {
		snap.Preferences = []Preference{}
	}
	return snap, nil
}

func (s *levelStore) update(fn func(*Snapshot) (Preference, bool, error)) (Preference, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.load()
	if err != nil {
		return Preference{}, false, err
	}
	next := cloneSnapshot(current)
	pref, changed, err := fn(&next)
	if err != nil || !changed {
		return pref, changed, err
	}
	next.Version = storeVersion
	next.Revision = current.Revision + 1
	if pref.ID != "" {
		for i := range next.Preferences {
			if next.Preferences[i].ID == pref.ID {
				next.Preferences[i].Revision = next.Revision
				next.Preferences[i].UpdatedAt = nowRFC3339()
				pref = next.Preferences[i]
				break
			}
		}
	}
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return Preference{}, false, err
	}
	if err := fileutil.AtomicWrite(s.file(), raw, 0o644); err != nil {
		return Preference{}, false, err
	}
	return pref, true, nil
}

func cloneSnapshot(in Snapshot) Snapshot {
	out := in
	out.Preferences = append([]Preference(nil), in.Preferences...)
	for i := range out.Preferences {
		out.Preferences[i].Evidence = append([]string(nil), out.Preferences[i].Evidence...)
	}
	out.AppliedEvents = make(map[string]string, len(in.AppliedEvents))
	for k, v := range in.AppliedEvents {
		out.AppliedEvents[k] = v
	}
	return out
}

func (s *levelStore) apply(ev Event) (Preference, bool, error) {
	if strings.TrimSpace(ev.ID) == "" {
		return Preference{}, false, fmt.Errorf("event id is required")
	}
	return s.update(func(snap *Snapshot) (Preference, bool, error) {
		if id, ok := snap.AppliedEvents[ev.ID]; ok {
			for _, p := range snap.Preferences {
				if p.ID == id {
					return p, false, nil
				}
			}
			return Preference{}, false, nil
		}
		switch ev.Action {
		case "remember":
			return rememberInto(s, snap, ev)
		case "decide":
			return decideInto(snap, ev)
		case "replace":
			return replaceInto(snap, ev)
		case "forget":
			return forgetInto(snap, ev)
		default:
			return Preference{}, false, fmt.Errorf("unknown author memory action %q", ev.Action)
		}
	})
}

func rememberInto(s *levelStore, snap *Snapshot, ev Event) (Preference, bool, error) {
	p := ev.Preference
	p.Text = strings.TrimSpace(p.Text)
	if p.Text == "" || strings.TrimSpace(p.Kind) == "" {
		return Preference{}, false, fmt.Errorf("kind and text are required")
	}
	if !validSource(p.Source) {
		return Preference{}, false, fmt.Errorf("author memory only accepts explicit user, accepted suggestion, or manual source")
	}
	if p.Scope == "" {
		p.Scope = "global"
	}
	if p.Importance <= 0 {
		p.Importance = 50
	}
	if p.ID == "" {
		p.ID = newID(s.prefix)
	}
	p.Status = StatusAccepted
	if conflictKey(snap, p.Key, p.Text, "") {
		p.Status = StatusConflict
	}
	p.Revision = snap.Revision + 1
	p.UpdatedAt = nowRFC3339()
	snap.Preferences = append(snap.Preferences, p)
	snap.AppliedEvents[ev.ID] = p.ID
	return p, true, nil
}

func decideInto(snap *Snapshot, ev Event) (Preference, bool, error) {
	p := findPreference(snap, ev.TargetID)
	if p == nil {
		return Preference{}, false, fmt.Errorf("preference %q not found", ev.TargetID)
	}
	if !ev.Accept {
		p.Status = StatusPending
	} else {
		p.Status = StatusAccepted
		for i := range snap.Preferences {
			other := &snap.Preferences[i]
			if other.ID != p.ID && other.Status == StatusAccepted && p.Key != "" && other.Key == p.Key {
				other.Status = StatusPending
			}
		}
	}
	snap.AppliedEvents[ev.ID] = p.ID
	return *p, true, nil
}

func replaceInto(snap *Snapshot, ev Event) (Preference, bool, error) {
	p := findPreference(snap, ev.TargetID)
	if p == nil {
		return Preference{}, false, fmt.Errorf("preference %q not found", ev.TargetID)
	}
	text := strings.TrimSpace(ev.Text)
	if text == "" {
		return Preference{}, false, fmt.Errorf("replacement text is required")
	}
	if conflictKey(snap, p.Key, text, p.ID) {
		return Preference{}, false, fmt.Errorf("replacement conflicts with another accepted preference for key %q", p.Key)
	}
	p.Text = text
	p.Status = StatusAccepted
	snap.AppliedEvents[ev.ID] = p.ID
	return *p, true, nil
}

func forgetInto(snap *Snapshot, ev Event) (Preference, bool, error) {
	p := findPreference(snap, ev.TargetID)
	if p == nil {
		return Preference{}, false, fmt.Errorf("preference %q not found", ev.TargetID)
	}
	out := *p
	kept := snap.Preferences[:0]
	for _, item := range snap.Preferences {
		if item.ID != p.ID {
			kept = append(kept, item)
		}
	}
	snap.Preferences = kept
	snap.AppliedEvents[ev.ID] = p.ID
	return out, true, nil
}

func conflictKey(snap *Snapshot, key, text, ignoreID string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, p := range snap.Preferences {
		if p.ID != ignoreID && p.Status == StatusAccepted && p.Key == key && p.Text != text {
			return true
		}
	}
	return false
}

func findPreference(snap *Snapshot, id string) *Preference {
	for i := range snap.Preferences {
		if snap.Preferences[i].ID == id {
			return &snap.Preferences[i]
		}
	}
	return nil
}

func validSource(s Source) bool {
	return s == SourceExplicitUser || s == SourceAcceptedSuggestion || s == SourceManual
}

func newID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d-%d", prefix, idSeq.Add(1), os.Getpid())
	}
	return fmt.Sprintf("%s-%d-%d-%s", prefix, idSeq.Add(1), os.Getpid(), hex.EncodeToString(buf))
}
