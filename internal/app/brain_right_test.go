package app

import (
	"testing"

	whisperdb "github.com/gaea/gaea/internal/whisper/db"
)

func TestRightBrainWriteSearch(t *testing.T) {
	dir := t.TempDir()
	if whisperdb.GetDatabase(dir) == nil {
		t.Fatal("whisper db unavailable")
	}
	defer whisperdb.CloseDatabase(dir)

	rb := &rightBrain{dataRoot: dir}
	if err := rb.Write("甲方A", "偏好", "保守报价"); err != nil {
		t.Fatal(err)
	}
	hits, err := rb.Search("报价")
	if err != nil || len(hits) == 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if hits[0].Entity != "甲方A" {
		t.Fatalf("entity = %q", hits[0].Entity)
	}
}
