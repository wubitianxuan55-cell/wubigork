package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// proposalTestEnv 把方案库落到临时目录。注意 MemoryUserDir 走 os.UserConfigDir，
// Windows 上读的是 %AppData%——只改 USERPROFILE 拦不住（2026-11-23 的旧测试因此
// 把夹具写进了正式库）。这里改 APPDATA/XDG_CONFIG_HOME，并 fail-closed 核对解析
// 出来的库路径确实在临时目录里。
func proposalTestEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	userDir := config.MemoryUserDir()
	if userDir == "" {
		t.Skip("本环境解析不出用户配置目录")
	}
	if !strings.HasPrefix(strings.ToLower(userDir), strings.ToLower(home)) {
		t.Fatalf("方案库未隔离：MemoryUserDir=%s 不在临时目录 %s 下，拒绝继续", userDir, home)
	}
	if db.GetDatabase(userDir) == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(userDir) })
}

// fakeGroups 假 embedding 的语义分组：文本里出现组内关键词即点亮该组维度。
// 查询与段落只要讲到同一件事就会靠近，测试因此确定且零网络。
var fakeGroups = [][]string{
	{"降水", "管井", "回灌"},
	{"沉降", "房子", "建筑物", "沉"},
	{"扬尘", "噪声", "边坡", "支护"},
	{"监测", "预警", "测点"},
}

func fakeEmbedText(s string) []float32 {
	v := make([]float32, 64)
	for gi, kws := range fakeGroups {
		for _, k := range kws {
			if strings.Contains(s, k) {
				v[gi]++
			}
		}
	}
	return v
}

// fakeModelServer 假的本地模型服务（OpenAI 兼容 /v1/models、/v1/embeddings、
// /v1/rerank），替身 Herdsman。
type fakeModelServer struct {
	*httptest.Server
	embedCalls  atomic.Int32
	rerankCalls atomic.Int32
}

func newFakeModelServer(t *testing.T) *fakeModelServer {
	t.Helper()
	f := &fakeModelServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"fake-embed"},{"id":"fake-rerank"}]}`))
	})
	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		f.embedCalls.Add(1)
		var in struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data := make([]map[string]any, 0, len(in.Input))
		for i, s := range in.Input {
			data = append(data, map[string]any{"index": i, "embedding": fakeEmbedText(s)})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
	mux.HandleFunc("/v1/rerank", func(w http.ResponseWriter, r *http.Request) {
		f.rerankCalls.Add(1)
		var in struct {
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		type item struct {
			idx   int
			score float64
		}
		items := make([]item, 0, len(in.Documents))
		for i, d := range in.Documents {
			var s float64
			for _, c := range in.Query {
				if strings.ContainsRune(d, c) {
					s++
				}
			}
			items = append(items, item{i, s})
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
		res := make([]map[string]any, 0, len(items))
		for _, it := range items {
			res = append(res, map[string]any{"index": it.idx, "relevance_score": it.score})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": res})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// useFakeModels 把工具面的本地模型客户端指到假服务（测试后还原）。
func useFakeModels(t *testing.T, f *fakeModelServer) {
	t.Helper()
	oldE, oldR, oldRT := embedderOverride, rerankerOverride, retrievalRuntime
	embedderOverride = retrieval.NewEmbedder(f.URL, "fake-embed")
	rerankerOverride = retrieval.New(f.URL, "fake-rerank")
	retrievalRuntime.EmbedModel = "fake-embed"
	t.Cleanup(func() {
		embedderOverride, rerankerOverride, retrievalRuntime = oldE, oldR, oldRT
	})
}

// useDeadModels 本地模型服务不可用（连接即拒），验证降级。
func useDeadModels(t *testing.T) {
	t.Helper()
	oldE, oldR, oldRT := embedderOverride, rerankerOverride, retrievalRuntime
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	embedderOverride = retrieval.NewEmbedder(dead.URL, "fake-embed")
	rerankerOverride = retrieval.New(dead.URL, "fake-rerank")
	retrievalRuntime.EmbedModel = "fake-embed"
	t.Cleanup(func() {
		embedderOverride, rerankerOverride, retrievalRuntime = oldE, oldR, oldRT
		dead.Close()
	})
}

// toolMessage 解出工具回执正文（WrapText 信封里的 Message）。
func toolMessage(t *testing.T, out string) string {
	t.Helper()
	env, ok := tool.ParseEnvelope(out)
	if !ok {
		t.Fatalf("回执不是信封格式: %q", out)
	}
	return env.Message
}

var proposalMD = `包3技术标 第三版

目录
第1章 项目实施方案 …… 3
1.1 总体部署 …… 4

# 第1章 项目实施方案
本章说明总体部署。

## 1.1 总体部署
分 A、B、C 三个施工区，先北后南。

## 1.2 施工降水
基坑降水采用管井降水，降水期间监测周边建筑物沉降。
如发现沉降超限，立即回灌并停止降水。

# 第4章 安全保障方案
安全管理体系与应急预案。

## 4.3 危险源辨识
基坑边坡失稳、扬尘、噪声为主要危险源。
边坡失稳的防控措施：分层开挖、及时支护。

### 4.3.1 危险源清单

| 序号 | 危险源 | 控制措施 |
|---|---|---|
| 1 | 边坡失稳 | 分层开挖 |
| 2 | 扬尘 | 洒水覆盖 |

## 4.4 监测
监测点布置与预警值。
`

func TestProposalSaveOutlineBatchesSearchRead(t *testing.T) {
	proposalTestEnv(t)
	useFakeModels(t, newFakeModelServer(t))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bao3.md"), []byte(proposalMD), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	save := proposalSaveTool{workDir: dir}
	search := proposalSearchTool{}
	read := proposalReadTool{}

	// 第一批：相对路径按工作区解析；不收尾
	out, err := save.Execute(ctx, toJSON(t, map[string]any{
		"title": "包3技术标 第三版", "project_type": "技术标", "year": "2026",
		"source_file": "bao3.md",
		"outline": []map[string]any{
			{"level": 1, "title": "第1章 项目实施方案"},
			{"level": 2, "title": "1.1 总体部署"},
			{"level": 2, "title": "1.2 施工降水"},
		},
	}))
	if err != nil {
		t.Fatalf("第一批: %v", err)
	}
	msg := toolMessage(t, out)
	for _, want := range []string{"录入中", "新增 3 个标题节点", "已录到第 13 行", "首个标题前 6 行", "已向量化", "mode=append"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("第一批回执缺 %q:\n%s", want, msg)
		}
	}
	// 第二批：续录 + 收尾（title 不变，元信息不丢）
	out, err = save.Execute(ctx, toJSON(t, map[string]any{
		"title": "包3技术标 第三版", "mode": "append", "source_file": filepath.Join(dir, "bao3.md"), "finalize": true,
		"outline": []map[string]any{
			{"level": 1, "title": "第4章 安全保障方案"},
			{"level": 2, "title": "4.3 危险源辨识"},
			{"level": 3, "title": "危险源清单"},
			{"level": 2, "title": "监测"},
		},
	}))
	if err != nil {
		t.Fatalf("第二批: %v", err)
	}
	msg = toolMessage(t, out)
	for _, want := range []string{"已录入方案库：", "累计 7 个节点", "1 级 2 / 2 级 4 / 3 级 1"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("第二批回执缺 %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "过长小节") {
		t.Fatalf("短小节不应报过长:\n%s", msg)
	}

	// 检索：关键词（两字词兜底）
	out, err = search.Execute(ctx, toJSON(t, map[string]any{"query": "扬尘"}))
	if err != nil {
		t.Fatalf("search 扬尘: %v", err)
	}
	msg = toolMessage(t, out)
	for _, want := range []string{"方案库命中 2 条", "4.3 危险源辨识", "【扬尘】", "proposal_read section_id=", "已本地精排"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("检索回执缺 %q:\n%s", want, msg)
		}
	}
	// 检索：换了说法（全文路零命中，语义路补上）
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{"query": "降水的时候旁边房子沉了怎么办"}))
	msg = toolMessage(t, out)
	if !strings.Contains(msg, "1.2 施工降水") || !strings.Contains(msg, "语义相似度") {
		t.Fatalf("语义路应召回 1.2 施工降水:\n%s", msg)
	}
	// 检索：标题树、方案清单、过滤、空结果
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第三版"}))
	msg = toolMessage(t, out)
	if !strings.Contains(msg, "标题树（共 7 节）") || !strings.Contains(msg, "] 第1章 项目实施方案（9 字）") || !strings.Contains(msg, "] 4.3.1 危险源清单") {
		t.Fatalf("标题树:\n%s", msg)
	}
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{"proposal": "包3技术标 第三版", "level": 3}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "共 1 节") {
		t.Fatalf("层级过滤的标题树:\n%s", msg)
	}
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "方案库概览（1 份）") || !strings.Contains(msg, "7/7") {
		t.Fatalf("方案清单:\n%s", msg)
	}
	// 标题路：只看标题树（骨架参照），标题本身命中者优先
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{"query": "危险源", "scope": "title"}))
	msg = toolMessage(t, out)
	if !strings.Contains(msg, "方案库命中 2 条（title）") || !strings.Contains(msg, "4.3.1 危险源清单") ||
		strings.Index(msg, "1. 4.3 危险源辨识") > strings.Index(msg, "4.3.1 危险源清单") {
		t.Fatalf("标题路:\n%s", msg)
	}
	out, _ = search.Execute(ctx, toJSON(t, map[string]any{"query": "完全查不到的词"}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "未命中") {
		t.Fatalf("空结果:\n%s", msg)
	}
	if _, err := search.Execute(ctx, toJSON(t, map[string]any{"query": "x", "scope": "bogus"})); err == nil {
		t.Fatal("非法 scope 应报错")
	}

	// 读取：整节 / 命中段落 / 子树 / 分页
	st, err := proposalStore()
	if err != nil {
		t.Fatal(err)
	}
	p, _ := st.Resolve("包3技术标 第三版")
	secs, _ := st.Outline(p.ID)
	byTitle := map[string]int64{}
	for _, s := range secs {
		byTitle[s.Title] = s.ID
	}
	out, err = read.Execute(ctx, toJSON(t, map[string]any{"section_id": byTitle["4.3 危险源辨识"]}))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	msg = toolMessage(t, out)
	for _, want := range []string{"## 4.3 危险源辨识", "**路径**: 第4章 安全保障方案 › 4.3 危险源辨识", "基坑边坡失稳、扬尘、噪声为主要危险源。", "[", "] 4.3.1 危险源清单（", "共 40 字，本次返回第 1~40 字。"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("阅读回执缺 %q:\n%s", want, msg)
		}
	}
	out, _ = read.Execute(ctx, toJSON(t, map[string]any{"section_id": byTitle["4.3 危险源辨识"], "query": "扬尘"}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "取到 1 处") || !strings.Contains(msg, "【扬尘】") {
		t.Fatalf("按词取段落:\n%s", msg)
	}
	out, _ = read.Execute(ctx, toJSON(t, map[string]any{"section_id": byTitle["4.3 危险源辨识"], "scope": "subtree", "query": "洒水"}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "连子节 1 个") || !strings.Contains(msg, "（出自子节 4.3.1 危险源清单）") {
		t.Fatalf("子树取段落:\n%s", msg)
	}
	out, _ = read.Execute(ctx, toJSON(t, map[string]any{"section_id": byTitle["第1章 项目实施方案"], "limit": 5}))
	if msg = toolMessage(t, out); !strings.Contains(msg, "未完，继续读：offset=5") {
		t.Fatalf("分页:\n%s", msg)
	}
	if _, err := read.Execute(ctx, toJSON(t, map[string]any{"section_id": 99999})); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("不存在的小节: %v", err)
	}
	if _, err := read.Execute(ctx, toJSON(t, map[string]any{})); err == nil {
		t.Fatal("缺 section_id 应报错")
	}
}

func TestProposalSaveLongSectionHinted(t *testing.T) {
	proposalTestEnv(t)
	useFakeModels(t, newFakeModelServer(t))
	body := strings.Repeat("工艺说明。", 2000) // 1 万字
	out, err := (proposalSaveTool{}).Execute(context.Background(), toJSON(t, map[string]any{
		"title": "长节方案",
		"sections": []map[string]any{
			{"level": 1, "title": "第一章 施工工艺", "body": body},
			{"level": 1, "title": "第二章 保证措施", "body": "简述。"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	msg := toolMessage(t, out)
	if !strings.Contains(msg, "过长小节") || !strings.Contains(msg, "第一章 施工工艺（1.0 万字）") {
		t.Fatalf("过长小节应提示:\n%s", msg)
	}
}

func TestProposalSaveGuards(t *testing.T) {
	proposalTestEnv(t)
	useFakeModels(t, newFakeModelServer(t))
	ctx := context.Background()
	save := proposalSaveTool{}
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{
		"title":    "技术标 甲",
		"sections": []map[string]any{{"level": 1, "title": "第一章", "body": "正文。"}},
	})); err != nil {
		t.Fatal(err)
	}
	// 同名未确认：拒，并说清三条出路
	_, err := save.Execute(ctx, toJSON(t, map[string]any{
		"title": "技术标 甲", "sections": []map[string]any{{"level": 1, "title": "别的", "body": "x"}},
	}))
	if err == nil || !strings.Contains(err.Error(), "已有同名方案") || !strings.Contains(err.Error(), "overwrite=true") {
		t.Fatalf("同名未确认应拒: %v", err)
	}
	// 跳级 / 空 / 非法 mode / 两形态同给
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{"title": "跳级", "sections": []map[string]any{
		{"level": 1, "title": "第一章", "body": "x"}, {"level": 3, "title": "直接三级", "body": "y"}}})); err == nil || !strings.Contains(err.Error(), "跳级") {
		t.Fatalf("跳级应 fail-closed: %v", err)
	}
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{"title": "空", "sections": []map[string]any{}})); err == nil {
		t.Fatal("空录入应报错")
	}
	if _, err := save.Execute(ctx, toJSON(t, map[string]any{"title": "x", "mode": "merge", "sections": []map[string]any{
		{"level": 1, "title": "a", "body": "b"}}})); err == nil || !strings.Contains(err.Error(), "mode 只能是") {
		t.Fatalf("非法 mode: %v", err)
	}
	// 源文件：缺、类型不对、找不到标题
	dir := t.TempDir()
	word := filepath.Join(dir, "方案.docx")
	if err := os.WriteFile(word, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = save.Execute(ctx, toJSON(t, map[string]any{"title": "x", "outline": []map[string]any{{"level": 1, "title": "a"}}}))
	if err == nil || !strings.Contains(err.Error(), "source_file") {
		t.Fatalf("缺源文件: %v", err)
	}
	_, err = save.Execute(ctx, toJSON(t, map[string]any{"title": "x", "source_file": word, "outline": []map[string]any{{"level": 1, "title": "a"}}}))
	if err == nil || !strings.Contains(err.Error(), "format_convert") {
		t.Fatalf("Word 应指引先转 Markdown: %v", err)
	}
	md := filepath.Join(dir, "a.md")
	if err := os.WriteFile(md, []byte("一、总则\n正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = save.Execute(ctx, toJSON(t, map[string]any{"title": "x", "source_file": md, "outline": []map[string]any{{"level": 1, "title": "没有的标题"}}}))
	if err == nil || !strings.Contains(err.Error(), "找不到") {
		t.Fatalf("找不到标题应报错: %v", err)
	}
	// 引擎会话件不能当来源
	_, err = save.Execute(ctx, toJSON(t, map[string]any{"title": "x", "source_file": filepath.Join(dir, ".gaea", "sessions", "x.md"),
		"outline": []map[string]any{{"level": 1, "title": "a"}}}))
	if err == nil || !strings.Contains(err.Error(), "SESSION_DENIED") {
		t.Fatalf("会话件围栏: %v", err)
	}
}

// 本地模型服务不可用：录入照常落库，检索降级为全文路，都不报错；模型恢复后
// 检索时自动补齐向量。
func TestProposalDegradesWhenLocalModelsDown(t *testing.T) {
	proposalTestEnv(t)
	useDeadModels(t)
	ctx := context.Background()
	save, search := proposalSaveTool{}, proposalSearchTool{}
	out, err := save.Execute(ctx, toJSON(t, map[string]any{
		"title": "降级方案",
		"sections": []map[string]any{
			{"level": 1, "title": "第一章", "body": "基坑降水采用管井降水，降水期间监测周边建筑物沉降。"},
			{"level": 1, "title": "第二章", "body": "边坡失稳的防控措施。"},
		},
	}))
	if err != nil {
		t.Fatalf("模型不可用时录入不应失败: %v", err)
	}
	if msg := toolMessage(t, out); !strings.Contains(msg, "向量化中断") {
		t.Fatalf("回执应如实说明向量化未完成:\n%s", msg)
	}
	out, err = search.Execute(ctx, toJSON(t, map[string]any{"query": "边坡失稳"}))
	if err != nil {
		t.Fatalf("模型不可用时检索不应失败: %v", err)
	}
	msg := toolMessage(t, out)
	if !strings.Contains(msg, "方案库命中 1 条") || !strings.Contains(msg, "全文路") {
		t.Fatalf("应降级为全文路并如实说明:\n%s", msg)
	}
	// 模型恢复：检索时补齐向量，语义路随之可用
	srv := newFakeModelServer(t)
	useFakeModels(t, srv)
	out, err = search.Execute(ctx, toJSON(t, map[string]any{"query": "降水的时候旁边房子沉了怎么办"}))
	if err != nil {
		t.Fatal(err)
	}
	msg = toolMessage(t, out)
	if !strings.Contains(msg, "第一章") || srv.embedCalls.Load() == 0 {
		t.Fatalf("模型恢复后应自动补齐并走语义路:\n%s", msg)
	}
}
