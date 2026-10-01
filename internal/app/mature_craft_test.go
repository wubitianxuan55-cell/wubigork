package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// TestNormalizeMatureLevel 档位白名单归一：合法值原样通过，其余（含用户手改
// project.json 的坏值）一律落 ""——不允许半开半关。
func TestNormalizeMatureLevel(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"sensual":       MatureSensual,
		"explicit":      MatureExplicit,
		"SENSUAL":       "", // 大小写敏感：写盘走白名单，读盘坏值不猜测
		"explicit ":     "", // 带空格的坏值不洗白
		"成人向":           "",
		"r18":           "",
		"explicit;drop": "",
	}
	for in, want := range cases {
		if got := normalizeMatureLevel(in); got != want {
			t.Errorf("normalizeMatureLevel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBuildMatureCraftSection 正文向工艺区段矩阵：空档位零注入；两档各带档位
// 专属工艺；硬线（成年/不美化胁迫）与功能闸两档都在。
func TestBuildMatureCraftSection(t *testing.T) {
	if got := buildMatureCraftSection(""); got != "" {
		t.Errorf("非成人向必须零注入，got:\n%s", got)
	}
	if got := buildMatureCraftSection("坏值"); got != "" {
		t.Errorf("坏值必须零注入，got:\n%s", got)
	}

	sensual := buildMatureCraftSection(MatureSensual)
	for _, want := range []string{"含蓄", "张力优先", "留白不是黑幕", "功能闸", "必须是成年人", "不把胁迫包装成浪漫"} {
		if !strings.Contains(sensual, want) {
			t.Errorf("含蓄档缺「%s」\n%s", want, sensual)
		}
	}
	if strings.Contains(sensual, "正面直书") {
		t.Errorf("含蓄档不得携带直白档口径")
	}

	explicit := buildMatureCraftSection(MatureExplicit)
	for _, want := range []string{"直白档", "正面直书", "感官纪律", "事后节拍", "同意节拍", "AI 腔限流", "功能闸", "必须是成年人", "不把胁迫包装成浪漫"} {
		if !strings.Contains(explicit, want) {
			t.Errorf("直白档缺「%s」\n%s", want, explicit)
		}
	}
	if strings.Contains(explicit, "留白不是黑幕") {
		t.Errorf("直白档不得携带含蓄档专属条目")
	}
}

// TestBuildMaturePlanSection 计划向纪律矩阵：空档位零注入；非空档位要求计划把
// 亲密戏当事件排（节拍+功能），并禁止空账表述。
func TestBuildMaturePlanSection(t *testing.T) {
	if got := buildMaturePlanSection(""); got != "" {
		t.Errorf("非成人向计划纪律必须零注入，got:\n%s", got)
	}
	plan := buildMaturePlanSection(MatureExplicit)
	for _, want := range []string{"成人向计划纪律", "节拍", "不得只写「两人关系升温」", "永久改变了什么"} {
		if !strings.Contains(plan, want) {
			t.Errorf("计划纪律缺「%s」\n%s", want, plan)
		}
	}
}

// TestUpdateProjectMeta 元信息更新绑定：合法更新落盘可回读；非法档位拒绝且不落盘；
// 空书名拒绝。
func TestUpdateProjectMeta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "旧书名", "玄幻", "热血", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a := &App{core: &core{}}
	// setPM 挂在 writingState 上：裸 &App{} 是 nil writingState（在册坑，v4.423 同款）
	a.writingState = &writingState{core: a.core, app: a, mu: sync.RWMutex{}}
	a.setPM(pm)

	if err := a.UpdateProjectMeta("新书名", "都市", "甜宠", MatureExplicit); err != nil {
		t.Fatalf("UpdateProjectMeta: %v", err)
	}
	info := a.GetProjectInfo()
	if info["title"] != "新书名" || info["mature"] != MatureExplicit {
		t.Fatalf("内存元信息未更新: %v", info)
	}
	// 落盘验证：重新 Open 读盘
	pm2, err := project.Open(dir)
	if err != nil {
		t.Fatalf("重新打开: %v", err)
	}
	if pm2.Meta.Title != "新书名" || pm2.Meta.Mature != MatureExplicit || pm2.Meta.Genre != "都市" {
		t.Fatalf("project.json 未落盘: %+v", pm2.Meta)
	}

	// 非法档位拒绝且不落盘
	if err := a.UpdateProjectMeta("新书名", "都市", "甜宠", "r18"); err == nil {
		t.Fatalf("非法档位必须拒绝")
	}
	pm3, _ := project.Open(dir)
	if pm3.Meta.Mature != MatureExplicit {
		t.Fatalf("被拒绝的更新不得落盘: %+v", pm3.Meta)
	}

	// 空书名拒绝
	if err := a.UpdateProjectMeta("  ", "都市", "甜宠", ""); err == nil {
		t.Fatalf("空书名必须拒绝")
	}

	// 未打开项目时拒绝
	a2 := &App{core: &core{}}
	a2.writingState = &writingState{core: a2.core, app: a2, mu: sync.RWMutex{}}
	if err := a2.UpdateProjectMeta("x", "", "", ""); err == nil {
		t.Fatalf("未打开项目必须拒绝")
	}
}

// TestCreateChapterInjectsMatureCraft 整章生成链注入断言（HTTP body 级）：
// 成人向项目 user prompt 必带工艺区段与档位口径；非成人向项目逐字节零出现。
func TestCreateChapterInjectsMatureCraft(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	gotRequests := make(chan []byte, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotRequests <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"正文"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	runCreate := func(t *testing.T, mature string) string {
		t.Helper()
		dir := filepath.Join(t.TempDir(), "novel")
		pm, err := project.Create(dir, "测试小说", "都市", "", "")
		if err != nil {
			t.Fatalf("创建项目: %v", err)
		}
		pm.Meta.Mature = mature
		a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
		a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
		a.setPM(pm)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		a.ctx = ctx
		defer cancel()

		gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
		gateSeedOutline(t, pm, gateOutlineNode(1, []string{"重逢"}, "张力"))
		if _, err := a.CreateChapter("设定文本", "", "主角重逢旧识", 1, "", "", 1200, 0); err != nil {
			t.Fatalf("CreateChapter(mature=%q): %v", mature, err)
		}
		select {
		case body := <-gotRequests:
			var req struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("解析请求体失败: %v", err)
			}
			for _, m := range req.Messages {
				if m.Role == "user" {
					return m.Content
				}
			}
			t.Fatalf("请求中缺少 user 消息")
			return ""
		case <-time.After(10 * time.Second):
			t.Fatalf("等待生成请求超时")
			return ""
		}
	}

	userPrompt := runCreate(t, MatureExplicit)
	for _, want := range []string{"成人向写作纪律", "直白档", "必须是成年人"} {
		if !strings.Contains(userPrompt, want) {
			t.Errorf("成人向项目 user prompt 缺「%s」", want)
		}
	}

	plainPrompt := runCreate(t, "")
	if strings.Contains(plainPrompt, "成人向") {
		t.Errorf("非成人向项目 user prompt 不得出现工艺区段:\n%s", plainPrompt)
	}
}
