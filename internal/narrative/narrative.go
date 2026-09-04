// Package narrative implements a deterministic narrative state machine plus an
// append-only patch ledger. LLM agents may only *propose* StatePatch values; no
// proposal reaches the canonical state until the author approves it via
// AuthorizeAndSettle. This is the single source of truth for the novel board's
// entity state, and its KnownBy field is the data source for POV masking in the
// later novelcontext stage.
//
// The package is deliberately pure-file and pure-function: it never calls an LLM
// or a database, and every behavior is verifiable offline via `go test`.
package narrative

import (
	"fmt"
	"time"
)

// EntityState is the tracked state of one narrative entity (character, location,
// item, organization, event, or concept). KnownBy records which entities (usually
// characters) are aware of this entity's key information, so later prompt
// assembly can withhold what a viewpoint character would not know.
type EntityState struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"` // character/location/item/org/event/concept
	Status     string            `json:"status"`
	Location   string            `json:"location"`
	Items      []string          `json:"items"`
	KnownBy    []string          `json:"known_by"`
	Properties map[string]string `json:"properties"`
}

// StateSnapshot is the full canonical state at a point in time. Version is the
// authoritative ordering token: it increments by one per successfully applied
// patch. UpdatedAt is informational; a pure ApplyPatch never rewrites it, so a
// replay is deterministic and comparisons stay exact.
type StateSnapshot struct {
	Version   int                    `json:"version"`
	Entities  map[string]EntityState `json:"entities"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// StatePatch is a proposed state change. An AI can only emit this structure; it
// can never mutate state directly. ProposedBy is the suggesting side and is
// always "ai" in normal operation.
type StatePatch struct {
	ID         string            `json:"id"`
	Chapter    int               `json:"chapter"`
	Branch     string            `json:"branch"`
	Reason     string            `json:"reason"`     // why the change is proposed (AI explanation)
	Upserts    []EntityState     `json:"upserts"`    // new / updated entity states
	Deletes    []string          `json:"deletes"`    // entity IDs to remove
	Foreshadow []PatchForeshadow `json:"foreshadow"` // foreshadow mutations
	Canon      []string          `json:"canon"`      // new canon facts (one sentence each)
	ProposedBy string            `json:"proposed_by"`
}

// PatchForeshadow is a foreshadowing mutation attached to a patch.
type PatchForeshadow struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Action      string `json:"action"` // planted / hinted / revealed
}

// Clone returns a deep copy of the snapshot. Maps and slices are duplicated so
// that the clone shares no backing arrays with the source; this protects against
// replay and concurrency pollution.
func (snap *StateSnapshot) Clone() *StateSnapshot {
	if snap == nil {
		return nil
	}
	c := &StateSnapshot{
		Version:   snap.Version,
		Entities:  make(map[string]EntityState, len(snap.Entities)),
		UpdatedAt: snap.UpdatedAt,
	}
	for k, v := range snap.Entities {
		c.Entities[k] = cloneEntity(v)
	}
	return c
}

// cloneEntity deep-copies an EntityState so slices and the Properties map are
// independent of the source.
func cloneEntity(e EntityState) EntityState {
	c := e
	if e.Items != nil {
		c.Items = append([]string(nil), e.Items...)
	}
	if e.KnownBy != nil {
		c.KnownBy = append([]string(nil), e.KnownBy...)
	}
	if e.Properties != nil {
		c.Properties = make(map[string]string, len(e.Properties))
		for k, v := range e.Properties {
			c.Properties[k] = v
		}
	}
	return c
}

// ApplyPatch applies a validated patch to a clone of the receiver and returns the
// new snapshot. The receiver is never mutated (pure function). An invalid patch
// returns an error and changes nothing.
func (snap *StateSnapshot) ApplyPatch(patch StatePatch) (*StateSnapshot, error) {
	if err := ValidateStatePatch(patch); err != nil {
		return nil, err
	}
	if snap == nil {
		snap = &StateSnapshot{}
	}
	newSnap := snap.Clone()
	if newSnap.Entities == nil {
		newSnap.Entities = map[string]EntityState{}
	}
	// Upserts win over nothing, but an explicit Delete for the same ID on the
	// same patch is authoritative (applied after upserts).
	for _, up := range patch.Upserts {
		newSnap.Entities[up.ID] = cloneEntity(up)
	}
	for _, id := range patch.Deletes {
		delete(newSnap.Entities, id)
	}
	newSnap.Version++
	return newSnap, nil
}

// ValidateStatePatch checks the structural legality of a patch: a positive
// chapter, a non-empty upsert ID/Name, an allowed foreshadow action, and a
// non-empty ProposedBy. A patch with ProposedBy == "ai" is treated as pending
// author approval (it does not take effect on its own); this function only
// checks shape, it never settles anything.
func ValidateStatePatch(patch StatePatch) error {
	if patch.Chapter <= 0 {
		return fmt.Errorf("narrative: chapter must be > 0, got %d", patch.Chapter)
	}
	if patch.ProposedBy == "" {
		return fmt.Errorf("narrative: proposed_by must be non-empty")
	}
	for _, up := range patch.Upserts {
		if up.ID == "" {
			return fmt.Errorf("narrative: upsert with empty id")
		}
		if up.Name == "" {
			return fmt.Errorf("narrative: upsert %q has empty name", up.ID)
		}
	}
	for _, id := range patch.Deletes {
		if id == "" {
			return fmt.Errorf("narrative: delete with empty id")
		}
	}
	for _, f := range patch.Foreshadow {
		switch f.Action {
		case "planted", "hinted", "revealed":
			// valid
		default:
			return fmt.Errorf("narrative: invalid foreshadow action %q (want planted/hinted/revealed)", f.Action)
		}
	}
	return nil
}
