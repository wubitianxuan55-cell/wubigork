package builtin

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// proposal_maintain_tools_test.go — 方案库维护三工具 + proposal_save 的进度/演练/落点回执
// + 清单目录列。全部走工具面（与模型调用同一条路径），不直连存储层。

func maintainSetup(t *testing.T) (dir string, save proposalSaveTool, update proposalUpdateTool, del proposalDeleteTool, export proposalExportTool, search proposalSearchTool) {
	t.Helper()
	proposalTestEnv(t)
	useFakeModels(t, newFakeModelServer(t))
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bao3.md"), []byte(proposalMD), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, proposalSaveTool{workDir: dir}, proposalUpdateTool{workDir: dir}, proposalDeleteTool{}, proposalExportTool{workDir: dir}, proposalSearchTool{}
}

func mustExec(t *testing.T, msg string, out string, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
	return toolMessage(t, out)
}

func mustContain(t *testing.T, msg, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("%s：回执缺 %q\n%s", msg, w, got)
		}
	}
}

func sectionID(t *testing.T, proposalName, title string) int64 {
	t.Helper()
	st, err := proposalStore()
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.Resolve(proposalName)
	if err != nil || p == nil {
		t.Fatalf("resolve %s: %v", proposalName, err)
	}
	secs, err := st.Outline(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range secs {
		if s.Title == title {
			return s.ID
		}
	}
	t.Fatalf("没有小节 %q", title)
	return 0
}

func TestProposalMaintainToolsLifecycle(t *testing.T) {
	dir, save, update, del, export, search := maintainSetup(t)
	ctx := context.Background()
	const title = "包3技术标 第三版"

	// 进度查询：库里还没有
	out, err := save.Execute(ctx, toJSON(t, map[string]any{"action": "status", "title": title}))
	mustContain(t, "status(空库)", mustExec(t, "status", out, err), "方案库里没有")
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{"action": "status", "title": title, "outline": []map[string]any{{"level": 1, "title": "第1章"}}})); err == nil {
		t.Fatal("action=status 带录入内容应报错")
	}
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{"action": "bogus", "title": title})); err == nil {
		t.Fatal("未知 action 应报错")
	}

	// 演练：校验照做、不写库
	batch := []map[string]any{
		{"level": 1, "title": "第1章 项目实施方案"},
		{"level": 2, "title": "1.1 总体部署"},
		{"level": 2, "title": "1.2 施工降水"},
	}
	out, err = save.Execute(ctx, toJSON(t, map[string]any{"title": title, "dry_run": true, "source_file": "bao3.md", "outline": batch}))
	mustContain(t, "dry-run", mustExec(t, "dry-run", out, err), "演练通过（未写库", "将新增 3 个标题节点", "正式写入后 1 份", "去掉 dry_run=true")
	out, _ = save.Execute(ctx, toJSON(t, map[string]any{"action": "status", "title": title}))
	mustContain(t, "status(演练后)", mustExec(t, "status", out, err), "方案库里没有")

	// 正式首批：回执带落点与库内总数
	out, err = save.Execute(ctx, toJSON(t, map[string]any{
		"title": title, "project_type": "技术标", "year": "2026", "source_file": "bao3.md", "outline": batch,
	}))
	first := mustExec(t, "首批", out, err)
	mustContain(t, "首批", first, "已录入方案库", "**落点**: 工作区 "+dir, "方案库共 1 份")

	// 进度：已录到第几行、下一批从哪里接
	out, err = save.Execute(ctx, toJSON(t, map[string]any{"action": "status", "title": title}))
	mustContain(t, "status(录入中)", mustExec(t, "status", out, err),
		"录入中", "已录到第 13 行", "已录**: 3 个标题节点", "2 级「1.2 施工降水」", "下一步", "dry_run=true")

	// 清单：带目录列（末级目录名）
	out, err = search.Execute(ctx, toJSON(t, map[string]any{}))
	mustContain(t, "清单", mustExec(t, "清单", out, err), "| 标题 | 类型 | 年度 | 状态 | 标题节点 | 字数 | 语义索引 | 目录 | 来源 |", "| "+filepath.Base(dir)+" |")

	// 收尾
	out, err = save.Execute(ctx, toJSON(t, map[string]any{"title": title, "mode": "append", "finalize": true, "source_file": "bao3.md", "outline": []map[string]any{
		{"level": 1, "title": "第4章 安全保障方案"},
		{"level": 2, "title": "4.3 危险源辨识"},
		{"level": 3, "title": "4.3.1 危险源清单"},
		{"level": 2, "title": "4.4 监测"},
	}}))
	mustExec(t, "收尾", out, err)
	out, err = save.Execute(ctx, toJSON(t, map[string]any{"action": "status", "title": title}))
	mustContain(t, "status(完成)", mustExec(t, "status", out, err), "现行", "7 个标题节点", "proposal_update")

	// 改元数据
	out, err = update.Execute(ctx, toJSON(t, map[string]any{"proposal": title, "title": "包3技术标 第四版", "project_type": "投标技术标", "workspace": `D:\新工作\沿江`}))
	mustContain(t, "update(meta)", mustExec(t, "update", out, err), "已修改方案信息", "包3技术标 第四版", "原名「"+title+"」", "投标技术标", `工作区 D:\新工作\沿江`)
	if _, err := update.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版"})); err == nil {
		t.Fatal("update 没有改动项应报错")
	}
	if _, err := update.Execute(ctx, toJSON(t, map[string]any{"proposal": "不存在", "year": "2027"})); err == nil || !strings.Contains(err.Error(), "没有《不存在》") {
		t.Fatalf("update 不存在的方案应报错，得到 err=%v", err)
	}

	// 增量修正某一节：改标题（录错一个字不必整册重录）
	sid := sectionID(t, "包3技术标 第四版", "1.1 总体部署")
	out, err = update.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "section_id": sid, "section_title": "1.1 施工总体部署"}))
	mustContain(t, "update(section)", mustExec(t, "update", out, err),
		"已修正方案库小节", "「1.1 总体部署」→「1.1 施工总体部署」", "**路径**: 第1章 项目实施方案 › 1.1 施工总体部署")
	if _, err := update.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "section_id": sid})); err == nil {
		t.Fatal("update 给了 section_id 却没给改动应报错")
	}

	// 整节替换正文
	out, err = update.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "section_id": sid, "body": "分东、西两个施工区，先深后浅。\n"}))
	mustContain(t, "update(body)", mustExec(t, "update", out, err), "正文已整节替换")
	out, err = search.Execute(ctx, toJSON(t, map[string]any{"query": "先深后浅"}))
	mustContain(t, "search(新正文)", mustExec(t, "search", out, err), "方案库命中 1 条", "先深后浅")

	// 导出：按标题层级还原
	outPath := filepath.Join("交付", "包3技术标 第四版.md")
	out, err = export.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "output": outPath}))
	mustContain(t, "export", mustExec(t, "export", out, err), "已导出《包3技术标 第四版》", filepath.Join(dir, outPath), "7 个标题节点")
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(outPath)))
	if err != nil {
		t.Fatal(err)
	}
	md := string(raw)
	mustContain(t, "export 文件", md, "> 方案库导出：包3技术标 第四版（投标技术标 / 2026）", "\n# 第1章 项目实施方案\n", "\n## 1.1 施工总体部署\n", "分东、西两个施工区，先深后浅。", "\n### 4.3.1 危险源清单\n", "| 1 | 边坡失稳 | 分层开挖 |")
	if _, err := export.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "output": "x.docx"})); err == nil {
		t.Fatal("export 非 .md 应报错")
	}
	if _, err := export.Execute(ctx, toJSON(t, map[string]any{"proposal": "不存在", "output": "a.md"})); err == nil {
		t.Fatal("export 不存在的方案应报错")
	}
	got, err := (proposalExportTool{workDir: dir}).WritePaths(toJSON(t, map[string]any{"output": "a/b.md"}))
	if err != nil || len(got) != 2 || got[0] != filepath.Join(dir, "a", "b.md") || got[1] != filepath.Join(dir, "a", "b-assets") {
		t.Fatalf("WritePaths = %v err=%v", got, err)
	}

	// 删除
	out, err = del.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版", "reason": "录错"}))
	mustContain(t, "delete", mustExec(t, "delete", out, err), "已从方案库删除《包3技术标 第四版》", "7 个标题节点", "方案库现有 0 份")
	if _, err := del.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第四版"})); err == nil {
		t.Fatal("delete 不存在的方案应报错")
	}
	if _, err := export.Execute(ctx, toJSON(t, map[string]any{"output": "a.md"})); err == nil {
		t.Fatal("export 缺 proposal 应报错")
	}
}

func TestProposalExportCopiesManagedImages(t *testing.T) {
	proposalTestEnv(t)
	useFakeModels(t, newFakeModelServer(t))
	ctx := context.Background()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fig.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	save := proposalSaveTool{workDir: dir}
	out, err := save.Execute(ctx, toJSON(t, map[string]any{
		"title": "带图方案",
		"sections": []map[string]any{
			{"level": 1, "title": "一 平面布置", "body": "说明文字。\n\n![平面图](fig.png)\n"},
		},
	}))
	mustContain(t, "save", mustExec(t, "save", out, err), "已托管 1 张")

	out, err = (proposalExportTool{workDir: dir}).Execute(ctx, toJSON(t, map[string]any{"proposal": "带图方案", "output": "导出/带图方案.md"}))
	msg := mustExec(t, "export", out, err)
	mustContain(t, "export", msg, "图片 1 张存到", filepath.Join(dir, "导出", "带图方案-assets"))
	raw, err := os.ReadFile(filepath.Join(dir, "导出", "带图方案.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "![平面图](带图方案-assets/") || strings.Contains(string(raw), "gaea-asset://") {
		t.Fatalf("导出文件里的图片应改为相对路径：%s", raw)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "导出", "带图方案-assets"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("应落 1 个图片文件：entries=%v err=%v", entries, err)
	}
}
