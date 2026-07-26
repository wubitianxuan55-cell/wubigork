package graph

import (
	"strings"
	"testing"
)

func TestParseLinks(t *testing.T) {
	content := "[[Elara]] walked into [[青云宗]] and met [[Kael]]."
	links := ParseLinks(content)

	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(links))
	}
	if links[0] != "Elara" || links[1] != "青云宗" || links[2] != "Kael" {
		t.Fatalf("unexpected links: %v", links)
	}
}

func TestParseLinks_Deduplication(t *testing.T) {
	content := "[[Elara]] saw [[Elara]] in the mirror."
	links := ParseLinks(content)
	if len(links) != 1 {
		t.Fatalf("expected 1 unique link, got %d", len(links))
	}
}

func TestParseLinks_Chinese(t *testing.T) {
	content := "[[青云宗]]坐落于[[苍山]]之巅，[[Elara]]是[[青云宗]]的弟子。"
	links := ParseLinks(content)
	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d: %v", len(links), links)
	}
}

func TestParseLinksWithContext(t *testing.T) {
	content := "She entered [[青云宗]] with hope.\nThe next day, [[Elara]] arrived."
	result := ParseLinksWithContext(content, "chapters/001.md")

	if len(result) != 2 {
		t.Fatalf("expected 2 links, got %d", len(result))
	}
	if result[0].SourceFile != "chapters/001.md" {
		t.Fatal("source file not set")
	}
	if result[0].LineNumber != 1 {
		t.Fatalf("expected line 1, got %d", result[0].LineNumber)
	}
	if result[1].LineNumber != 2 {
		t.Fatalf("expected line 2, got %d", result[1].LineNumber)
	}
}

func TestFindUnlinkedMentions(t *testing.T) {
	content := "Elara walked through 青云宗. [[Kael]] was waiting at 苍山."
	entities := []string{"Elara", "青云宗", "Kael", "苍山"}

	unlinked := FindUnlinkedMentions(content, entities)
	// Elara and 青云宗 should be unlinked (bare text), Kael is linked ([[Kael]]), 苍山 is bare
	if len(unlinked) != 3 {
		t.Fatalf("expected 3 unlinked, got %d: %v", len(unlinked), unlinked)
	}
}

func TestEntityDB_Query(t *testing.T) {
	db := &EntityDB{
		Entities: []Entity{
			{ID: "1", Name: "Elara", Type: EntityCharacter},
			{ID: "2", Name: "青云宗", Type: EntityLocation},
			{ID: "3", Name: "Kael", Type: EntityCharacter},
		},
	}

	chars := db.Query(EntityCharacter)
	if len(chars) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(chars))
	}

	locs := db.Query(EntityLocation)
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locs))
	}

	all := db.Query("")
	if len(all) != 3 {
		t.Fatalf("expected 3 total, got %d", len(all))
	}
}

func TestEntityDB_GetByName(t *testing.T) {
	db := &EntityDB{
		Entities: []Entity{
			{ID: "1", Name: "Elara", Type: EntityCharacter},
		},
	}

	e := db.GetByName("elara") // case insensitive
	if e == nil {
		t.Fatal("GetByName should be case-insensitive")
	}
	if e.ID != "1" {
		t.Fatal("wrong entity returned")
	}

	e2 := db.GetByName("nonexistent")
	if e2 != nil {
		t.Fatal("expected nil for nonexistent")
	}
}

func TestConsistencyIssues_Structure(t *testing.T) {
	issue := ConsistencyIssue{
		Severity:    "error",
		Category:    "attribute",
		EntityName:  "Elara",
		Description: "眼睛颜色不一致",
		Location:    "第3章",
		Evidence:    "蓝眼睛 vs 绿眼睛",
		Suggestion:  "统一描述",
	}

	if issue.Severity != "error" {
		t.Fatal("severity mismatch")
	}
}

func TestBacklinkIndex(t *testing.T) {
	idx := make(BacklinkIndex)
	idx["Elara"] = []Link{
		{Target: "Elara", SourceFile: "chapters/001.md", LineNumber: 5},
		{Target: "Elara", SourceFile: "chapters/003.md", LineNumber: 12},
	}

	backlinks := idx.GetBacklinks("Elara")
	if len(backlinks) != 2 {
		t.Fatalf("expected 2 backlinks, got %d", len(backlinks))
	}

	entities := idx.GetAllEntities()
	if len(entities) != 1 || entities[0] != "Elara" {
		t.Fatalf("unexpected entities: %v", entities)
	}
}

func TestExtractProperty(t *testing.T) {
	content := "Elara的蓝色眼睛闪烁着光芒，她站在大殿中央。"

	result := extractProperty(content, "Elara", []string{"眼睛", "眼眸"})
	if result == "" {
		t.Fatal("expected to extract eye property")
	}
	if !strings.Contains(result, "蓝色") {
		t.Fatalf("expected 蓝色 in result, got: %s", result)
	}
}
