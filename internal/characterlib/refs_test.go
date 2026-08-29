package characterlib

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrateSchemaV2_IdempotentAndLegacyCompat v4.3g 迁移契约：
//   - 旧库（仅 SchemaV1）打开后自动补 reference_images / gallery_images 列；
//   - 旧行读回两字段为默认空数组（'[]'），不报错；
//   - 迁移幂等：重复执行（直接调用 + 重新打开）不报错、不重复加列。
func TestMigrateSchemaV2_IdempotentAndLegacyCompat(t *testing.T) {
	dir := t.TempDir()

	// 1. 构造纯 SchemaV1 旧库 + 一条旧行（绕过 GetDatabase 的迁移）
	raw, err := sql.Open("sqlite", filepath.Join(dir, "characterlib.db"))
	if err != nil {
		t.Fatalf("打开旧库: %v", err)
	}
	if _, err := raw.Exec(SchemaV1); err != nil {
		t.Fatalf("建旧表: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO characters (id, name, kind, portrait_url, created_at, updated_at)
		 VALUES ('legacy_1', '旧角色', 'custom', '', 'x', 'x')`); err != nil {
		t.Fatalf("插入旧行: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("关闭旧库: %v", err)
	}

	// 2. GetDatabase 触发迁移
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("Store 初始化失败")
	}
	defer s.Close()

	for _, col := range []string{"reference_images", "gallery_images"} {
		has, err := hasColumn(s.db, "characters", col)
		if err != nil || !has {
			t.Fatalf("迁移后缺少列 %s: has=%v err=%v", col, has, err)
		}
	}

	// 3. 旧行兼容：两字段读回空数组（非 nil，序列化为 []）
	c, err := s.Get("legacy_1")
	if err != nil || c == nil {
		t.Fatalf("读取旧行: %v %v", c, err)
	}
	if c.ReferenceImages == nil || len(c.ReferenceImages) != 0 {
		t.Fatalf("旧行 referenceImages 应为空数组: %#v", c.ReferenceImages)
	}
	if c.GalleryImages == nil || len(c.GalleryImages) != 0 {
		t.Fatalf("旧行 galleryImages 应为空数组: %#v", c.GalleryImages)
	}
	// JSON 往返一致：空数组 omitempty 下省略键，回读为空（无数据损坏）
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back Character
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(back.ReferenceImages) != 0 || len(back.GalleryImages) != 0 {
		t.Fatalf("旧行 JSON 往返后参考图应保持为空: %#v %#v", back.ReferenceImages, back.GalleryImages)
	}

	// 4. 幂等：直接再跑一次迁移 + 重新打开数据库再跑
	if err := migrateSchemaV2(s.db); err != nil {
		t.Fatalf("重复迁移报错: %v", err)
	}
	if err := CloseDatabase(dir); err != nil {
		t.Fatalf("关闭: %v", err)
	}
	s2 := NewStore(dir)
	if s2 == nil || s2.db == nil {
		t.Fatal("重新打开 Store 失败")
	}
	defer s2.Close()
	for _, col := range []string{"reference_images", "gallery_images"} {
		has, err := hasColumn(s2.db, "characters", col)
		if err != nil || !has {
			t.Fatalf("重开后缺少列 %s: has=%v err=%v", col, has, err)
		}
	}
	// 旧行仍在且字段兼容
	c2, err := s2.Get("legacy_1")
	if err != nil || c2 == nil || len(c2.ReferenceImages) != 0 {
		t.Fatalf("重开后旧行异常: %+v %v", c2, err)
	}
}

// TestReferenceGalleryImages_RoundTrip 参考图/画廊图保存→读取往返一致：
//   - data URL 本地化为 portraits 目录文件路径；
//   - 远程 URL 下载为本地文件路径；
//   - 已本地化路径原样保留；
//   - JSON 往返（Marshal → Unmarshal）两字段内容一致；
//   - List / DrawRandom 同样带出两字段。
func TestReferenceGalleryImages_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("Store 不可用")
	}
	defer s.Close()

	// 1x1 红色 PNG
	const tinyPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

	// 远程图片服务
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(strings.TrimPrefix(tinyPNG, "data:image/png;base64,")))
	}))
	defer srv.Close()
	remoteURL := srv.URL + "/ref.png"

	localPath := filepath.Join(dir, "portraits", "existing.png")
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("建 portraits 目录: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("写本地路径: %v", err)
	}

	c := &Character{
		ID:              "rt1",
		Name:            "往返测试",
		Kind:            KindCustom,
		ReferenceImages: []string{tinyPNG, remoteURL, localPath},
		GalleryImages:   []string{tinyPNG},
	}
	if err := s.Upsert(c); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := s.Get("rt1")
	if err != nil || got == nil {
		t.Fatalf("Get: %v %v", got, err)
	}
	if len(got.ReferenceImages) != 3 {
		t.Fatalf("referenceImages 数量 = %d, want 3: %#v", len(got.ReferenceImages), got.ReferenceImages)
	}
	// data URL → 文件路径（_ref_0）；远程 URL → 下载文件（_ref_1）；本地路径原样
	if strings.HasPrefix(got.ReferenceImages[0], "data:") {
		t.Fatalf("data URL 参考图应本地化为文件: %s", got.ReferenceImages[0])
	}
	if !strings.HasPrefix(got.ReferenceImages[0], dir) {
		t.Fatalf("参考图应存文件路径: %s", got.ReferenceImages[0])
	}
	if _, err := os.Stat(got.ReferenceImages[0]); err != nil {
		t.Fatalf("参考图文件不存在: %v", err)
	}
	if !strings.HasPrefix(got.ReferenceImages[1], dir) {
		t.Fatalf("远程参考图应下载为本地文件: %s", got.ReferenceImages[1])
	}
	if got.ReferenceImages[2] != localPath {
		t.Fatalf("本地路径参考图应原样保留: %s", got.ReferenceImages[2])
	}
	if len(got.GalleryImages) != 1 || strings.HasPrefix(got.GalleryImages[0], "data:") {
		t.Fatalf("画廊图应本地化: %#v", got.GalleryImages)
	}

	// JSON 往返
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back Character
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(back.ReferenceImages) != 3 || back.ReferenceImages[0] != got.ReferenceImages[0] ||
		back.ReferenceImages[1] != got.ReferenceImages[1] || back.ReferenceImages[2] != got.ReferenceImages[2] {
		t.Fatalf("JSON 往返后 referenceImages 不一致: %#v vs %#v", back.ReferenceImages, got.ReferenceImages)
	}
	if len(back.GalleryImages) != 1 || back.GalleryImages[0] != got.GalleryImages[0] {
		t.Fatalf("JSON 往返后 galleryImages 不一致: %#v vs %#v", back.GalleryImages, got.GalleryImages)
	}

	// List / DrawRandom 带出两字段
	items, _, err := s.List("", "", false, 50, 0)
	if err != nil || len(items) != 1 {
		t.Fatalf("List: %d %v", len(items), err)
	}
	if len(items[0].ReferenceImages) != 3 || len(items[0].GalleryImages) != 1 {
		t.Fatalf("List 应带出参考图/画廊图: %#v", items[0])
	}
	drawn, err := s.DrawRandom(5, "", "", false)
	if err != nil || len(drawn) != 1 || len(drawn[0].ReferenceImages) != 3 {
		t.Fatalf("DrawRandom 应带出参考图: %#v %v", drawn, err)
	}
}

// TestMarshalStringList_NilBecomesEmptyArray nil 列表序列化为 "[]" 而非 "null"
// （保证列值语义与 NOT NULL DEFAULT '[]' 一致，旧行/空角色读回空数组）。
func TestMarshalStringList_NilBecomesEmptyArray(t *testing.T) {
	if got := marshalStringList(nil); got != "[]" {
		t.Fatalf("nil 应序列化为 [], got %q", got)
	}
	if got := marshalStringList([]string{}); got != "[]" {
		t.Fatalf("空数组应序列化为 [], got %q", got)
	}
	if got := marshalStringList([]string{"a", "b"}); got != `["a","b"]` {
		t.Fatalf("列表序列化错误: %s", got)
	}
}

// TestUpsertRefsWriteEmptyArrayToDB 无参考图时库里存 '[]'（非 null）。
func TestUpsertRefsWriteEmptyArrayToDB(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("Store 不可用")
	}
	defer s.Close()

	if err := s.Upsert(&Character{ID: "e1", Name: "空参考", Kind: KindCustom}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	var refs, gallery string
	if err := s.db.QueryRow(`SELECT reference_images, gallery_images FROM characters WHERE id='e1'`).Scan(&refs, &gallery); err != nil {
		t.Fatalf("查询: %v", err)
	}
	if refs != "[]" || gallery != "[]" {
		t.Fatalf("空参考应落库为 '[]', got refs=%q gallery=%q", refs, gallery)
	}
}
