package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestRegisterCharacterPortraitAsset(t *testing.T) {
	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	cwd := t.TempDir()
	img := filepath.Join(cwd, "c1.png")
	if err := os.WriteFile(img, []byte("png"), 0o644); err != nil {
		t.Fatalf("write img: %v", err)
	}
	registerCharacterPortraitAsset(cwd, []types.Character{
		{ID: "c1", PortraitURL: img},
		{ID: "c2", PortraitURL: "data:image/png;base64,AAAA"},
	}, "c1")

	got := newImageHubLedger(cwd).list("play", 0)
	if len(got) != 1 {
		t.Fatalf("应登记 1 条剧照，got %d", len(got))
	}
	rec := got[0]
	if rec.Meta.SourceBoard != "characterlib" || rec.Asset.Path != img {
		t.Fatalf("剧照登记错误: %+v", rec)
	}
	if rec.Meta.Params["character_id"] != "c1" {
		t.Fatalf("缺 character_id: %+v", rec.Meta.Params)
	}
}

func TestRegisterCharacterPortraitAssetSkipsRemoteOrMissing(t *testing.T) {
	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	cwd := t.TempDir()
	registerCharacterPortraitAsset(cwd, []types.Character{
		{ID: "c1", PortraitURL: "data:image/png;base64,AAAA"},
		{ID: "c2", PortraitURL: filepath.Join(cwd, "missing.png")},
		{ID: "c3", PortraitURL: ""},
	}, "c1")
	registerCharacterPortraitAsset(cwd, []types.Character{{ID: "c1", PortraitURL: "data:image/png;base64,AAAA"}}, "c2")
	if got := newImageHubLedger(cwd).list("play", 0); len(got) != 0 {
		t.Fatalf("远程/缺失/空路径不应登记: %+v", got)
	}
}
