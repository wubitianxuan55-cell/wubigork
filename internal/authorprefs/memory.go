package authorprefs

import (
	"fmt"
	"strings"
)

type Memory struct {
	AP *levelStore
	BP *levelStore
}

func Open(workspaceRoot, bookRoot string) (*Memory, error) {
	ap, err := openLevel(workspaceRoot, "AP")
	if err != nil {
		return nil, err
	}
	var bp *levelStore
	if strings.TrimSpace(bookRoot) != "" {
		bp, err = openLevel(bookRoot, "BP")
		if err != nil {
			return nil, err
		}
	}
	return &Memory{AP: ap, BP: bp}, nil
}

func (m *Memory) Apply(ev Event) (Preference, bool, error) {
	if m == nil || m.AP == nil {
		return Preference{}, false, fmt.Errorf("author memory is not open")
	}
	if ev.Action == "decide" || ev.Action == "replace" || ev.Action == "forget" {
		store := m.AP
		if strings.HasPrefix(ev.TargetID, "BP-") && m.BP != nil {
			store = m.BP
		}
		return store.apply(ev)
	}
	scope := ev.Preference.Scope
	store := m.AP
	if (scope == "book" || scope == "") && m.BP != nil {
		store = m.BP
	}
	return store.apply(ev)
}

func (m *Memory) Query(kind, scope string) (QueryResult, error) {
	ap, err := m.AP.load()
	if err != nil {
		return QueryResult{}, err
	}
	bp := Snapshot{}
	if m.BP != nil {
		bp, err = m.BP.load()
		if err != nil {
			return QueryResult{}, err
		}
	}
	return renderQuery(mergeAccepted(ap, bp, kind, scope)), nil
}

func (m *Memory) View() (View, error) {
	ap, err := m.AP.load()
	if err != nil {
		return View{}, err
	}
	if m.BP == nil {
		return viewOf(ap), nil
	}
	bp, err := m.BP.load()
	if err != nil {
		return View{}, err
	}
	return viewOf(ap, bp), nil
}
