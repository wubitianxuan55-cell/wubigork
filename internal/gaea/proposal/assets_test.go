package proposal

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="

func newAssetStore(t *testing.T) (*Store, string) {
	t.Helper()
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(userDir) })
	st, err := OpenWithAssets(gdb, userDir)
	if err != nil {
		t.Fatal(err)
	}
	return st, userDir
}

func writeTinyPNG(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProposalAssetsExternalizeAndDelete(t *testing.T) {
	st, _ := newAssetStore(t)
	sourceDir := t.TempDir()
	raw := writeTinyPNG(t, filepath.Join(sourceDir, "images", "flow.png"))

	res, err := st.Save(SaveInput{
		Meta:         Proposal{Title: "图片方案"},
		AssetBaseDir: sourceDir,
		Sections: []Section{{
			Level: 1, Title: "第一章 工艺",
			Body: "工艺路线如下：\n\n![工艺流程图](images/flow.png)",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Assets != 1 || len(res.MissingAssets) != 0 {
		t.Fatalf("保存回执: %+v", res)
	}
	secs, err := st.Outline(res.ProposalID)
	if err != nil || len(secs) != 1 {
		t.Fatalf("Outline: %v %+v", err, secs)
	}
	read, err := st.Read(secs[0].ID, ReadOpt{Limit: ReadAll})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read.Text, "![工艺流程图](gaea-asset://") || strings.Contains(read.Text, "images/flow.png") {
		t.Fatalf("正文引用未改写: %s", read.Text)
	}
	start := strings.Index(read.Text, AssetURIPrefix)
	if start < 0 {
		t.Fatalf("缺少资产引用: %s", read.Text)
	}
	ref := read.Text[start : start+len(AssetURIPrefix)+64]
	dataURL, err := st.AssetDataURL(ref)
	if err != nil {
		t.Fatal(err)
	}
	got, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, "data:image/png;base64,"))
	if err != nil || string(got) != string(raw) {
		t.Fatalf("回读图片不一致: %v", err)
	}
	hash := strings.TrimPrefix(ref, AssetURIPrefix)
	assetPath := st.assetPath(hash, "image/png")
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("资产文件未落盘: %v", err)
	}
	// 已托管正文再次整册覆盖时，应识别并保留 gaea-asset 引用，不要求回源图片。
	res, err = st.Save(SaveInput{
		Meta:      Proposal{Title: "图片方案"},
		Overwrite: true,
		Sections:  []Section{{Level: 1, Title: "第一章 工艺", Body: read.Text}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Assets != 1 || len(res.MissingAssets) != 0 {
		t.Fatalf("覆盖已托管正文: %+v", res)
	}
	if _, err := st.AssetDataURL(ref); err != nil {
		t.Fatalf("覆盖后图片应可读: %v", err)
	}

	if err := st.Delete("图片方案"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(assetPath); !os.IsNotExist(err) {
		t.Fatalf("删除方案后资产文件应回收: %v", err)
	}
}

func TestProposalAssetsOutlineFormAndMissing(t *testing.T) {
	st, _ := newAssetStore(t)
	sourceDir := t.TempDir()
	writeTinyPNG(t, filepath.Join(sourceDir, "figure.png"))
	lines := []string{
		"# 第一章 现场",
		"![现场照片](figure.png)",
		"![缺图](missing.png)",
	}
	res, err := st.Save(SaveInput{
		Meta:         Proposal{Title: "outline 图片"},
		Outline:      []OutlineEntry{{Level: 1, Title: "第一章 现场", Line: 1}},
		Lines:        lines,
		SourceFile:   filepath.Join(sourceDir, "source.md"),
		AssetBaseDir: sourceDir,
		Finalize:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Assets != 1 || len(res.MissingAssets) != 1 || !strings.Contains(res.MissingAssets[0], "缺图") {
		t.Fatalf("资产/缺图回执: %+v", res)
	}
	secs, err := st.Outline(res.ProposalID)
	if err != nil {
		t.Fatal(err)
	}
	read, err := st.Read(secs[0].ID, ReadOpt{Limit: ReadAll})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read.Text, "gaea-asset://") || !strings.Contains(read.Text, "missing.png") {
		t.Fatalf("应改写成功图、保留缺图原引用: %s", read.Text)
	}
}

func TestProposalAssetsDedupAndSharedLifetime(t *testing.T) {
	st, _ := newAssetStore(t)
	sourceDir := t.TempDir()
	writeTinyPNG(t, filepath.Join(sourceDir, "shared.png"))
	save := func(title string) string {
		t.Helper()
		res, err := st.Save(SaveInput{
			Meta:         Proposal{Title: title},
			AssetBaseDir: sourceDir,
			Sections:     []Section{{Level: 1, Title: "第一章", Body: "![共用图](shared.png)"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		secs, err := st.Outline(res.ProposalID)
		if err != nil || len(secs) != 1 {
			t.Fatalf("Outline: %v %+v", err, secs)
		}
		read, err := st.Read(secs[0].ID, ReadOpt{Limit: ReadAll})
		if err != nil {
			t.Fatal(err)
		}
		start := strings.Index(read.Text, AssetURIPrefix)
		if start < 0 {
			t.Fatalf("正文缺少资产引用: %s", read.Text)
		}
		return read.Text[start : start+len(AssetURIPrefix)+64]
	}
	first := save("方案甲")
	second := save("方案乙")
	if first != second {
		t.Fatalf("同图应共用 sha 引用: %s != %s", first, second)
	}
	hash := strings.TrimPrefix(first, AssetURIPrefix)
	path := st.assetPath(hash, "image/png")
	if err := st.Delete("方案甲"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("另一方案仍引用时不得删文件: %v", err)
	}
	if _, err := st.AssetDataURL(second); err != nil {
		t.Fatalf("另一方案图片应继续可读: %v", err)
	}
	if err := st.Delete("方案乙"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("最后一个引用删除后应回收文件: %v", err)
	}
}
