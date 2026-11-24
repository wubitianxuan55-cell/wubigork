package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/proposal"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// proposalBindingEnv 把方案库落到临时目录。MemoryUserDir 走 os.UserConfigDir，
// Windows 上读的是 %AppData%——只改 USERPROFILE 拦不住（2026-11-23 的旧测试因此把
// 夹具写进了正式库），这里改 APPDATA 并 fail-closed 核对。本地模型客户端指向
// 不可达端口：零网络，检索降级为全文路。
func proposalBindingEnv(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	dir := config.MemoryUserDir()
	if !strings.HasPrefix(strings.ToLower(dir), strings.ToLower(home)) {
		t.Fatalf("方案库未隔离：MemoryUserDir=%s 不在临时目录 %s 下，拒绝继续", dir, home)
	}
	// Windows 上库文件被连接池占着时临时目录删不掉：先关库（后注册的 cleanup 先跑）。
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	dead := "http://127.0.0.1:1"
	builtin.SetRetrievalRuntime(builtin.RetrievalRuntime{
		EmbedKind: "openai", EmbedBaseURL: dead, EmbedModel: "test-embed",
		RerankKind: "openai", RerankBaseURL: dead, RerankModel: "test-rerank",
	})
	t.Cleanup(func() { builtin.SetRetrievalRuntime(builtin.RetrievalRuntime{}) })
	return &App{}
}

func TestProposalAssetBinding(t *testing.T) {
	a := proposalBindingEnv(t)
	st, err := a.hubProposalStore()
	if err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()
	raw, err := base64.StdEncoding.DecodeString(tinyProposalPNG)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "images", "scene.png"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := st.Save(proposal.SaveInput{
		Meta:         proposal.Proposal{Title: "图片绑定方案"},
		AssetBaseDir: sourceDir,
		Sections: []proposal.Section{{
			Level: 1, Title: "第一章",
			Body: "![现场图](images/scene.png)",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Assets != 1 {
		t.Fatalf("Assets=%d", res.Assets)
	}
	tree, err := a.GaeaProposalOutline("图片绑定方案")
	if err != nil {
		t.Fatal(err)
	}
	body, err := a.GaeaProposalRead(tree[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(body.Body, proposal.AssetURIPrefix)
	if start < 0 {
		t.Fatalf("正文缺少资产引用: %s", body.Body)
	}
	ref := body.Body[start : start+len(proposal.AssetURIPrefix)+64]
	dataURL, err := a.GaeaProposalAssetDataURL(ref)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("data URL=%q", dataURL)
	}
	if err := a.GaeaProposalDelete("图片绑定方案"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaProposalAssetDataURL(ref); err == nil {
		t.Fatal("删除方案后资产绑定应失效")
	}
}

const tinyProposalPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="

// TestProposalBindingsListOutlineReadSearchDelete 方案库板块的端到端
// （与界面同一条调用链）：清单带规模 → 标题树带父子 → 读节 → 检索回小节与
// 命中摘要（本地模型不可达时如实降级）→ 删除级联。
func TestProposalBindingsListOutlineReadSearchDelete(t *testing.T) {
	a := proposalBindingEnv(t)
	st, err := a.hubProposalStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Save(proposal.SaveInput{
		Meta: proposal.Proposal{Title: "技术标 甲", ProjectType: "技术标", Year: "2026"},
		Sections: []proposal.Section{
			{Level: 1, Title: "第一章 施工降水", Body: "基坑降水采用管井降水，降水期间监测周边建筑物沉降。"},
			{Level: 2, Title: "1.1 沉降控制", Body: "如发现沉降超限，立即回灌并停止降水。"},
			{Level: 1, Title: "第二章 安全保证", Body: "边坡失稳的防控措施：分层开挖、及时支护。"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	list, err := a.GaeaProposalList()
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v %+v", err, list)
	}
	if it := list[0]; it.Title != "技术标 甲" || it.Sections != 3 || it.Chars == 0 || it.Status != proposal.StatusActive || it.UpdatedAt == "" {
		t.Fatalf("清单行: %+v", it)
	}

	tree, err := a.GaeaProposalOutline("技术标 甲")
	if err != nil || len(tree) != 3 {
		t.Fatalf("Outline: %v %+v", err, tree)
	}
	if tree[1].ParentID != tree[0].ID || tree[1].TitlePath != "第一章 施工降水 › 1.1 沉降控制" || tree[1].Level != 2 || tree[0].Chars == 0 {
		t.Fatalf("标题树: %+v", tree)
	}
	if _, err := a.GaeaProposalOutline("没有的方案"); err == nil || !strings.Contains(err.Error(), "没有") {
		t.Fatalf("不存在的方案应报错: %v", err)
	}

	body, err := a.GaeaProposalRead(tree[0].ID)
	if err != nil || body.Body != "基坑降水采用管井降水，降水期间监测周边建筑物沉降。" || body.ProposalTitle != "技术标 甲" ||
		body.TitlePath != "第一章 施工降水" || len(body.Children) != 1 || body.Children[0].Title != "1.1 沉降控制" {
		t.Fatalf("Read: %v %+v", err, body)
	}
	if _, err := a.GaeaProposalRead(99999); err == nil {
		t.Fatal("不存在的小节应报错")
	}

	res, err := a.GaeaProposalSearch("边坡 失稳", "", 0, "", "", "", 10)
	if err != nil || res == nil || len(res.Hits) != 1 {
		t.Fatalf("Search: %v %+v", err, res)
	}
	// 两个相邻的命中词连成一个【】标记段
	if h := res.Hits[0]; h.SectionID != tree[2].ID || !strings.Contains(h.Snippet, "【边坡失稳】") || len(h.Matched) != 2 {
		t.Fatalf("命中: %+v", h)
	}
	// 本地模型不可达：如实说明降级，不报错、不吞
	if res.Mode == "hybrid" && !strings.Contains(res.Note, "语义路无命中") {
		t.Fatalf("降级说明缺失: mode=%q note=%q", res.Mode, res.Note)
	}
	// 过滤：层级 / 限定方案 / 标题树范围
	if res, _ = a.GaeaProposalSearch("沉降", "", 2, "", "", "", 10); res == nil || len(res.Hits) != 1 || res.Hits[0].Level != 2 {
		t.Fatalf("层级过滤: %+v", res)
	}
	if res, _ = a.GaeaProposalSearch("沉降", "", 0, "", "", "别的方案", 10); res == nil || len(res.Hits) != 0 {
		t.Fatalf("限定不存在的方案应为空: %+v", res)
	}
	// 标题树范围：标题或标题路径含词即命中（子节的路径里带父节标题），标题本身命中者排前
	if res, _ = a.GaeaProposalSearch("降水", "title", 0, "", "", "", 10); res == nil || res.Mode != "title" || len(res.Hits) != 2 ||
		res.Hits[0].SectionID != tree[0].ID || res.Hits[1].SectionID != tree[1].ID ||
		!strings.Contains(res.Hits[0].Snippet, "【降水】") {
		t.Fatalf("标题树范围: %+v", res)
	}

	if err := a.GaeaProposalDelete("技术标 甲"); err != nil {
		t.Fatal(err)
	}
	if list, err = a.GaeaProposalList(); err != nil || len(list) != 0 {
		t.Fatalf("删除后清单应空: %v %+v", err, list)
	}
	if err := a.GaeaProposalDelete("技术标 甲"); err == nil {
		t.Fatal("重复删除应报错")
	}
}
