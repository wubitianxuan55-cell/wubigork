package app

// 长篇刀5测试：风格学习回灌（指纹→可执行指令→生成注入）。
// 覆盖：指令编译矩阵 / 构建往返与生命周期 / 注入区段（CreateChapter HTTP 断言）/
// 零 digest 零注入。

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/types"
)

// digestFixture 测试摘要档（含可断言指令片段）。
func digestFixture() *types.StyleDigestFile {
	return &types.StyleDigestFile{
		BuiltAt: "2026-09-30T00:00:00Z", Chapters: 3, Chars: 3600,
		Instructions: "- 句子平均约 18 字，长短交错明显\n- 四字格克制（作者密度 1.0/千字）：避免成语连用，一段至多一处\n- 连接词克制：少用「然而/此外」开头",
	}
}

// outlineFixtureNodes 大纲夹具。
func outlineFixtureNodes() types.OutlineFile {
	return types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Title: "第一章", Summary: "开头", KeyPoints: []string{"引入"}},
	}}
}

// TestDigestOf 指令编译矩阵：nil 空 / 句长离散 vs 均匀 / 对话高占比 /
// 低密度避免项 / 签名词。
func TestDigestOf(t *testing.T) {
	if got := novelstyle.DigestOf(nil); got != "" {
		t.Fatalf("nil 指纹应空指令：%q", got)
	}
	// 长短交错（sd > 0.7×mean）+ 高对话 + 低四字格/连接词/形副
	loose := &novelstyle.Fingerprint{
		SentenceLen: novelstyle.LenDist{Mean: 18, Sd: 14},
		ParaLen:     novelstyle.LenDist{Mean: 60},
		DialogRatio: 0.55, FourCharRatio: 1.0, ConnectiveDensity: 1.5, AdjAdvDensity: 10,
		TTRSd: 9, AuthorSignWords: []string{"眼底", "喉结滚动"},
	}
	got := novelstyle.DigestOf(loose)
	for _, want := range []string{
		"长短交错明显", "一段一个镜头", "对话驱动", "四字格克制", "连接词克制",
		"用动作与名词说话", "惯用表达", "眼底、喉结滚动",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("指令应含 %q：\n%s", want, got)
		}
	}
	// 均匀句长 + 低对话 → 均匀口径
	even := &novelstyle.Fingerprint{
		SentenceLen: novelstyle.LenDist{Mean: 20, Sd: 5},
		DialogRatio: 0.08,
	}
	got = novelstyle.DigestOf(even)
	if !strings.Contains(got, "节奏均匀") || !strings.Contains(got, "叙述为主") {
		t.Fatalf("均匀指纹应给均匀口径：\n%s", got)
	}
}

// TestNovelStyleDigestLifecycle 构建→读→注入区段→清除→零注入（样本门槛 3 章 3000 字）。
func TestNovelStyleDigestLifecycle(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)
	// 未构建：Get exists:false + 注入零
	dg, err := a.NovelStyleDigestGet()
	if err != nil || dg["exists"] != false {
		t.Fatalf("未构建应 exists:false：%v err=%v", dg, err)
	}
	if sec := a.styleDigestSection(pm); sec != "" {
		t.Fatalf("零 digest 应零注入：%q", sec)
	}
	// 造样本：3 章各 1200+ 字（作者风格=短句少对话低密度）
	for i := 1; i <= 3; i++ {
		body := strings.Repeat("他推门进来，雨还没停。她没抬头，手里的针线也没停。灯芯短了一截，屋里暗下来。", 40)
		if err := pm.WriteChapter(i, body); err != nil {
			t.Fatal(err)
		}
	}
	res, err := a.NovelStyleDigestBuild()
	if err != nil {
		t.Fatalf("构建: %v", err)
	}
	instr, _ := res["instructions"].(string)
	if instr == "" {
		t.Fatal("构建结果应含指令集")
	}
	// 读回一致
	dg2, err := a.NovelStyleDigestGet()
	if err != nil || dg2["exists"] != true {
		t.Fatalf("读回：%v err=%v", dg2, err)
	}
	// 注入区段含标题与指令
	sec := a.styleDigestSection(pm)
	if !strings.Contains(sec, "作者风格约束") || !strings.Contains(sec, instr[:12]) {
		t.Fatalf("注入区段应含标题与指令：%s", sec)
	}
	// 清除 → 零注入
	if err := a.NovelStyleDigestClear(); err != nil {
		t.Fatalf("清除: %v", err)
	}
	if sec := a.styleDigestSection(pm); sec != "" {
		t.Fatalf("清除后应零注入：%q", sec)
	}
	// 样本不足：新项目 1 章短文 → 诚实拒绝
	a2 := newGateEmptyApp()
	pm2 := newGateProject(t)
	a2.setPM(pm2)
	if err := pm2.WriteChapter(1, "太短。"); err != nil {
		t.Fatal(err)
	}
	if _, err := a2.NovelStyleDigestBuild(); err == nil || !strings.Contains(err.Error(), "样本不足") {
		t.Fatalf("样本不足应拒绝：%v", err)
	}
}

// TestCreateChapterDigestInjection 整章生成 prompt 含作者风格约束（HTTP body 断言）。
func TestCreateChapterDigestInjection(t *testing.T) {
	a, pm, requests := newChapterGateLLMAppReply(t, "正文。")
	a.setPM(pm)
	if err := pm.WriteStyleDigest(digestFixture()); err != nil {
		t.Fatal(err)
	}
	of := outlineFixtureNodes()
	if err := pm.WriteOutlines(&of); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateChapterWithOverride("设定", "", "写第一章", 1, "", "", 10, 0.8, true); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	// Windows unlinkat 竞态（在册）：生成协程尾步写盘 vs TempDir 清理——收尾必等。
	waitGensDone(t, a)
	select {
	case body := <-requests:
		// 刀6：digest 并入「文风与表达」合并区段（去重单标题）。
		if !strings.Contains(string(body), "文风与表达") || !strings.Contains(string(body), "四字格克制") {
			t.Fatalf("prompt 应含文风合并区段与学习指令（回灌验收），捕获体未见")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("未捕获生成请求")
	}
}
