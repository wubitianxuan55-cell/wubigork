package app

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestMarkNodeWritten(t *testing.T) {
	root := types.OutlineNode{
		OrderIndex: 1,
		Status:     types.OutlinePlanned,
		Children: []types.OutlineNode{
			{
				OrderIndex: 2,
				Branch:     "a",
				Status:     types.OutlineWriting,
			},
		},
	}

	if !markNodeWritten(&root, 2, "a") {
		t.Fatal("markNodeWritten should find branch node")
	}
	if root.Children[0].Status != types.OutlineDone {
		t.Fatalf("branch status = %q, want %q", root.Children[0].Status, types.OutlineDone)
	}
	if root.Status != types.OutlinePlanned {
		t.Fatalf("root status should stay planned, got %q", root.Status)
	}
}
