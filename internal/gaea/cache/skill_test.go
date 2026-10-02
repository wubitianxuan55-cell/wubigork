package cache

import (
	"reflect"
	"testing"
)

// TestProfilesSingleVersionPerKind pins the post-Learner (V5.0) shape of
// Profiles: the adaptive-promotion machinery is deleted, so nothing can ever
// select a version other than 1. A v2/v3 entry reappearing in this map would
// be unreachable dead weight (Route hard-pins version 1) — this test fails if
// one comes back.
func TestProfilesSingleVersionPerKind(t *testing.T) {
	if len(Profiles) == 0 {
		t.Fatal("Profiles must not be empty")
	}
	for kind, versions := range Profiles {
		if len(versions) != 1 {
			t.Errorf("Profiles[%q] defines %d versions, want exactly 1 (Learner removed, v2/v3 must stay dead)", kind, len(versions))
			continue
		}
		if _, ok := versions[1]; !ok {
			t.Errorf("Profiles[%q] must define version 1", kind)
		}
	}
	if _, ok := Profiles[KindDefault][1]; !ok {
		t.Error("Profiles[default][1] must exist — it is Route's ultimate fallback")
	}
}

// TestRouteAlwaysSelectsVersionOne pins that Route is pinned to version 1 for
// every classifiable intent: CurrentVersion() stays 1 and the returned profile
// is exactly Profiles[kind][1]. If a version>1 selection path is ever
// reintroduced (e.g. the old per-kind version resolver brought back), this
// goes red because Profiles no longer carries those versions.
func TestRouteAlwaysSelectsVersionOne(t *testing.T) {
	inputs := []string{
		"fix the login bug",
		"add a new feature",
		"review this pull request",
		"explain how this works",
		"research the best web frameworks",
		"analyze this csv of sales",
		"write a blog post about go",
		"help me plan a trip",
		"what is the meaning of this error?",
		"hello",
	}
	seen := map[TaskKind]bool{}
	for _, in := range inputs {
		l := NewSkillLayer()
		p := l.Route(in)
		seen[p.Kind] = true
		if got := l.CurrentVersion(); got != 1 {
			t.Errorf("Route(%q): CurrentVersion = %d, want 1 (no version>1 path exists)", in, got)
		}
		want, ok := Profiles[p.Kind][1]
		if !ok {
			t.Errorf("Route(%q) selected kind %q which has no version-1 profile", in, p.Kind)
			continue
		}
		if !reflect.DeepEqual(p, want) {
			t.Errorf("Route(%q) = %+v, want Profiles[%q][1] = %+v", in, p, p.Kind, want)
		}
		if !reflect.DeepEqual(l.CurrentProfile(), want) {
			t.Errorf("Route(%q): CurrentProfile = %+v, want the same version-1 profile", in, want)
		}
	}
	// Guard the battery itself: it must exercise more than one kind, otherwise
	// the assertions above silently narrow.
	if len(seen) < 3 {
		t.Errorf("input battery covered only %d kinds (%v), want a spread", len(seen), seen)
	}
}
