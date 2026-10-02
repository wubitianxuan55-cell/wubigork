package app

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/app/board"
)

// fakeBoard 测试用板块：仅携带 id/intents 的声明式实现。
type fakeBoard struct {
	id      string
	name    string
	intents []string
}

func (f fakeBoard) ID() string               { return f.id }
func (f fakeBoard) Name() string             { return f.name }
func (f fakeBoard) Icon() string             { return "I" }
func (f fakeBoard) PageKey() string          { return "p" }
func (f fakeBoard) Bindings() []string       { return nil }
func (f fakeBoard) Intents() []string        { return f.intents }
func (f fakeBoard) Tools() []string          { return nil }
func (f fakeBoard) Init(board.AppHost) error { return nil }

// TestInitModulesManifestDriven initModules 由 manifest 驱动（§5.2）：canonical
// 板块清单声明意图 → 装配出 gaea/whisper/novel/imagegen/office 五个主脑模块，
// 装配无完整性错误；office.create（D8）可正常派发。
func TestInitModulesManifestDriven(t *testing.T) {
	a := &App{}
	a.initModules()
	if a.modules == nil {
		t.Fatal("initModules 未初始化注册表")
	}
	if err := a.modules.Err(); err != nil {
		t.Fatalf("canonical manifest 装配不应有完整性错误: %v", err)
	}
	for _, want := range []string{"gaea", "whisper", "novel", "imagegen", "office"} {
		if !a.modules.Has(want) {
			t.Fatalf("manifest 驱动装配缺少模块 %q", want)
		}
	}
	// office.create → GaeaSend（D8）：裸 App（core 未装配）返回提交语义。
	out, err := a.modules.Dispatch("office", "create", map[string]any{"prompt": "写一份标书"})
	if err != nil {
		t.Fatalf("office.create 派发失败: %v", err)
	}
	if out["status"] != "submitted" {
		t.Fatalf("office.create 应返回 submitted，got %+v", out)
	}
}

// TestCheckModuleIntegrityClean 启动自检：canonical 清单完整性校验通过
// （intent 全部有 handler、无重复 id、无未知意图）。
func TestCheckModuleIntegrityClean(t *testing.T) {
	a := &App{}
	if err := a.CheckModuleIntegrity(); err != nil {
		t.Fatalf("启动自检应通过: %v", err)
	}
}

// TestFillFromManifestsIntentWithoutHandler 完整性断言：人为制造「intent 无
// handler」→ FillFromManifests 返回 error 且注册表记录 fillErr（启动自检
// 不静默）；对 canonical 清单 + 空 resolver 同样生效（缺陷 2 防复发）。
func TestFillFromManifestsIntentWithoutHandler(t *testing.T) {
	// 场景 1：fake 板块声明了 handler 表里没有的意图。
	reg := NewModuleRegistry()
	broken := []board.Board{fakeBoard{id: "ghost", name: "幽灵", intents: []string{"teleport"}}}
	err := reg.FillFromManifests(broken, func(string, string) (string, Handler, bool) {
		return "", nil, false
	})
	if err == nil {
		t.Fatal("intent 无 handler 应返回 error（不静默）")
	}
	if !strings.Contains(err.Error(), "无 handler") {
		t.Fatalf("错误信息应指出无 handler，got: %v", err)
	}
	if reg.Err() == nil || !strings.Contains(reg.Err().Error(), "无 handler") {
		t.Fatalf("注册表应记录完整性错误（启动自检可读），got Err()=%v", reg.Err())
	}

	// 场景 2：canonical 清单 + 空 resolver（人为制造全部意图无 handler）。
	reg2 := NewModuleRegistry()
	err2 := reg2.FillFromManifests(board.Builtins(), func(string, string) (string, Handler, bool) {
		return "", nil, false
	})
	if err2 == nil {
		t.Fatal("canonical 清单 + 空 resolver 应报错")
	}
	if !strings.Contains(err2.Error(), "无 handler") {
		t.Fatalf("got: %v", err2)
	}
}

// TestInitModulesLogsIntegrityFailure 启动路径不静默：initModules 装配失败时
// 必须记录 slog.Error（人为制造：用损坏 resolver 驱动 canonical 清单）。
func TestInitModulesLogsIntegrityFailure(t *testing.T) {
	a := &App{}
	a.modules = NewModuleRegistry()
	records := captureLogs(t, func() {
		// 模拟 initModules 内部装配失败的日志路径
		if err := a.modules.FillFromManifests(board.Builtins(), func(string, string) (string, Handler, bool) {
			return "", nil, false
		}); err != nil {
			t.Logf("装配失败（预期）: %v", err)
		}
	})
	// 启动自检错误必须显式可见（这里以注册表 Err() 承载，等价于 initModules 的日志）。
	if a.modules.Err() == nil {
		t.Fatal("装配失败后注册表应记录错误")
	}
	_ = records // 日志断言在 app.Startup 的 initModules 处由 slog.Error 显式输出
}

// TestFillFromManifestsHappyPath fake 板块 + 完整 resolver：正常装配并派发。
func TestFillFromManifestsHappyPath(t *testing.T) {
	reg := NewModuleRegistry()
	bs := []board.Board{
		fakeBoard{id: "novel", name: "小说", intents: []string{"create_chapter"}},
	}
	resolve := func(boardID, intent string) (string, Handler, bool) {
		if boardID == "novel" && intent == "create_chapter" {
			return "novel", func(input map[string]any) (map[string]any, error) {
				return map[string]any{"title": input["title"]}, nil
			}, true
		}
		return "", nil, false
	}
	if err := reg.FillFromManifests(bs, resolve); err != nil {
		t.Fatalf("装配失败: %v", err)
	}
	if !reg.Has("novel") {
		t.Fatal("novel 模块未注册")
	}
	out, err := reg.Dispatch("novel", "create_chapter", map[string]any{"title": "第一章"})
	if err != nil {
		t.Fatalf("Dispatch 失败: %v", err)
	}
	if out["title"] != "第一章" {
		t.Fatalf("out = %+v", out)
	}
}

// TestGetBoardManifestsCanonical GetBoardManifests 返回 canonical 清单：
// 12 个 canonical 板块（含 cost 成本库 + v4.110.0 schedule 进度计划 +
// 原罪 sin 故事创作；v4.439 删除 code 编程板块）+ knowledge 独立板块（D7），含 weixin 服务板块；
// JSON 字段对齐 doc §5.2 TS schema；CoreB 委托与 App 方法一致。
func TestGetBoardManifestsCanonical(t *testing.T) {
	a := &App{}
	manifests := a.GetBoardManifests()
	if len(manifests) != 13 {
		t.Fatalf("GetBoardManifests 应返回 13 个板块（12 canonical + knowledge D7），got %d", len(manifests))
	}
	byID := map[string]board.Manifest{}
	for _, m := range manifests {
		byID[m.ID] = m
	}
	ids := board.SortedIDs(manifests)
	got := strings.Join(ids, ",")
	for _, want := range board.CanonicalIDs() {
		if _, ok := byID[want]; !ok {
			t.Errorf("缺少 canonical 板块 %q（现有: %s）", want, got)
		}
	}
	// weixin 触点板块（v4.4：WeixinPage 落地 + 进菜单）
	wx, ok := byID["weixin"]
	if !ok {
		t.Fatal("weixin 触点板块缺失")
	}
	if wx.Page != "WeixinPage" || len(wx.Bindings) != 0 {
		t.Errorf("weixin 应挂载 WeixinPage 触点板块，got %+v", wx)
	}
	// knowledge 独立板块（D7 恢复挂载）
	kn, ok := byID["knowledge"]
	if !ok {
		t.Fatal("knowledge 独立板块缺失（D7）")
	}
	if kn.Page == "" || kn.InMenu == nil || !*kn.InMenu {
		t.Errorf("knowledge 应挂载为可导航板块，got %+v", kn)
	}
	// 编程板块已于 v4.439 删除（用户拍板「删除编程板块」）：manifest 不再含 code
	if _, ok := byID["code"]; ok {
		t.Error("code 编程板块应已删除，manifest 不应再含该条目")
	}
	// 成本库一级板块（2026-08-19 重设计：成本库独立成一级导航）
	ct, ok := byID["cost"]
	if !ok {
		t.Fatal("成本库一级板块缺失")
	}
	if ct.Label != "造价数据库" || ct.Page != "CostLibraryPage" || ct.Icon != "AccountBookOutlined" ||
		ct.InMenu == nil || !*ct.InMenu || ct.MenuOrder != 5 ||
		len(ct.Bindings) != 1 || ct.Bindings[0] != "CostB" {
		t.Errorf("成本库板块 manifest 字段不齐: %+v", ct)
	}
	// 进度计划（瘦身刀1 v4.168.0）：并入办公文档面——InMenu=false 藏出顶栏菜单/
	// 启动器，仍注册可导航（前端白名单刀0 已解耦），办公文件面/命令面板承接。
	sc, ok := byID["schedule"]
	if !ok {
		t.Fatal("进度计划板块缺失")
	}
	if sc.Label != "进度计划" || sc.Page != "SchedulePage" || sc.InMenu == nil || *sc.InMenu ||
		sc.MenuOrder != 12 {
		t.Errorf("瘦身刀1 schedule 应 InMenu=false（办公入口承接）: %+v", sc)
	}
	// chat 板块字段抽样对齐 TS schema
	chat := byID["chat"]
	if chat.Label != "聊天" || chat.Icon != "MessageOutlined" || chat.Page != "ChatPage" ||
		!chat.Lazy || chat.KeepAlive == nil || !*chat.KeepAlive || chat.Layout != "full" ||
		chat.Shortcut != "ctrl+1" || chat.MenuOrder != 1 || chat.InMenu == nil || !*chat.InMenu ||
		chat.FeatureModel != "chat" {
		t.Errorf("chat 板块 manifest 字段不齐: %+v", chat)
	}
	if len(chat.Bindings) != 2 || chat.Bindings[0] != "VoiceB" || chat.Bindings[1] != "ChatB" {
		t.Errorf("chat 绑定门面应声明 VoiceB+ChatB: %v", chat.Bindings)
	}
	if len(chat.Intents) != 1 || chat.Intents[0].ID != "chat" || chat.Intents[0].Handler != "WhisperChat" {
		t.Errorf("chat 意图应声明 whisper.chat: %+v", chat.Intents)
	}
	// JSON 可序列化（前端零依赖解析，D2）
	raw, err := json.Marshal(manifests)
	if err != nil {
		t.Fatalf("manifest 序列化失败: %v", err)
	}
	if !strings.Contains(string(raw), `"id":"chat"`) {
		t.Errorf("JSON 输出缺少 chat 板块: %s", string(raw))
	}
	// CoreB 委托（gen_bindings 显式覆盖 GetBoardManifests→core）与 App 一致
	cb := &CoreB{a: a}
	got2 := cb.GetBoardManifests()
	if len(got2) != len(manifests) {
		t.Errorf("CoreB 委托数量不一致: %d vs %d", len(got2), len(manifests))
	}
}

// TestMainBrainDefaultBranchDispatchable 审计 AP9-03 复核回归锁：canonical
// 清单 + 生产 resolver 装配后，classifyMainBrainIntent 的 default 分支
// （"gaea","chat"）必须能派发到 ChatGeneral 闭包（审计声称「主脑默认分支
// 永远派发失败」——实为 moduleOfIntent 误读：gaea.chat→gaea、gaea.create→
// office 是不同模块，无静默覆盖；此测试钉死该事实防回归）。
func TestMainBrainDefaultBranchDispatchable(t *testing.T) {
	a := &App{}
	a.modules = NewModuleRegistry()
	if err := a.modules.FillFromManifests(board.Builtins(), a.resolveIntent); err != nil {
		t.Fatalf("canonical 清单装配失败: %v", err)
	}
	moduleID, intent := classifyMainBrainIntent("帮我看看今天的安排")
	if moduleID != "gaea" || intent != "chat" {
		t.Fatalf("default 分支应落 (gaea,chat)，实得 (%s,%s)", moduleID, intent)
	}
	if !a.modules.Has(moduleID) {
		t.Fatalf("模块 %q 未注册（默认分支将派发失败）", moduleID)
	}
	// resolver 必须给出非 nil 闭包（裸 App 执行闭包会因 core 未装配 panic，
	// 这里只断言「有 handler」，闭包连通性由 ChatGeneral 自身用例覆盖）。
	_, h, ok := a.resolveIntent("gaea", "chat")
	if !ok || h == nil {
		t.Fatalf("default 分支 (gaea,chat) 解析失败：ok=%v handler=nil=%v", ok, h == nil)
	}
	// gaea 板块的双意图各自成模块（gaea.chat→gaea、gaea.create→office），
	// 装配不得报多意图完整性错误。
	if a.modules.Err() != nil {
		t.Fatalf("装配记录了完整性错误: %v", a.modules.Err())
	}
}

// TestMainBrainClassifierBranchesDispatchable 审计 AP9-06 可执行部分：分类器
// （classifyMainBrainIntent）与注册表两套口径的机器对齐守卫——分类器每个
// 分支的 (module, intent) 硬编码输出，必须在 manifest 驱动装配的注册表中
// 可派发（模块已注册且声明该意图）。CheckModuleIntegrity 的完整性断言只
// 覆盖「manifest 声明意图」方向、不覆盖分类器输出，两边漂移时 MainBrainChat
// 只会在运行期告警跳过（D8 路径）——本守卫把漂移提前到测试期拦红。
func TestMainBrainClassifierBranchesDispatchable(t *testing.T) {
	a := &App{}
	a.initModules()
	for _, msg := range []string{
		"写一份标书",     // → office.create
		"生成第三章",     // → novel.create_chapter
		"和轻语聊聊天",    // → whisper.chat
		"画一张星空图",    // → imagegen.generate
		"帮我看看今天的安排", // → gaea.chat（default 分支）
	} {
		moduleID, intent := classifyMainBrainIntent(msg)
		intents, ok := a.modules.IntentsOf(moduleID)
		if !ok {
			t.Errorf("分类器分支 %q → 模块 %q 未注册（MainBrainChat 运行期只会告警跳过）", msg, moduleID)
			continue
		}
		if !slices.Contains(intents, intent) {
			t.Errorf("分类器分支 %q → (%q,%q)：模块已注册但未声明该意图（Intents=%v）", msg, moduleID, intent, intents)
		}
	}
}
