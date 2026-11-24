package docmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalizeDataImages(t *testing.T) {
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="
	root := t.TempDir()
	mdDir := filepath.Join(root, "exports")
	assetDir := filepath.Join(mdDir, "方案-assets")
	md := "前文\n\n![工艺流程图](data:image/png;base64," + png + ")\n\n后文"

	out, count, err := ExternalizeDataImages(md, mdDir, assetDir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d, want 1", count)
	}
	if strings.Contains(out, "data:image/") {
		t.Fatalf("data URI 未外置: %s", out)
	}
	if !strings.Contains(out, "![工艺流程图](方案-assets/") {
		t.Fatalf("相对引用未改写: %s", out)
	}
	entries, err := os.ReadDir(assetDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("sidecar 文件: %v %+v", err, entries)
	}
}

func TestExternalizeDataImagesMalformedFailsClosed(t *testing.T) {
	root := t.TempDir()
	_, _, err := ExternalizeDataImages(
		"![坏图](data:image/png;base64,not-base64!)",
		root,
		filepath.Join(root, "assets"),
	)
	if err == nil {
		t.Fatal("损坏 data URI 应报错")
	}
}
