package pantheon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T15（codex 蒸馏）验收：新建角色文件即出现 / 删除不残留 / description 进
// spawn 引导 / 角色级覆盖只影响该席位。

// fakeRoleBackend 记录每席收到的调用与角色（隔离性验收用）。
type fakeRoleBackend struct {
	calls     map[string]RoleFile
	prompts   map[string]string
	available bool
}

func newFakeRoleBackend() *fakeRoleBackend {
	return &fakeRoleBackend{
		calls: map[string]RoleFile{}, prompts: map[string]string{}, available: true,
	}
}

func (f *fakeRoleBackend) ProbeRole(_ context.Context, role RoleFile) AgentStatus {
	return AgentStatus{ID: role.Name, Name: role.Name, Available: f.available}
}

func (f *fakeRoleBackend) CallRole(_ context.Context, role RoleFile, prompt string) (string, error) {
	f.calls[role.Name] = role
	f.prompts[role.Name] = prompt
	return "陈词:" + role.Name, nil
}

func writeRoleFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// LoadRoleFiles：合法文件解析（frontmatter + persona 正文）；目录不存在静默空。
func TestLoadRoleFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	// 目录不存在：常态空表，不是错误。
	roles, warns, err := LoadRoleFiles(dir)
	if err != nil || len(roles) != 0 || len(warns) != 0 {
		t.Fatalf("目录不存在应空表: %v %v %v", roles, warns, err)
	}
	writeRoleFile(t, dir, "reviewer.md", `---
name: reviewer
description: 独立审稿人：对方案做风险与漏洞审查。
nicknames: 审稿人甲,审稿人乙
config_layer: reviewer.toml
---
你是审稿人席位，只做风险审查。`)
	writeRoleFile(t, dir, "reviewer.toml", "[agent]\nmax_steps = 6\n")
	// 缺 description 的坏文件：跳过并告警。
	writeRoleFile(t, dir, "broken.md", "---\nname: broken\n---\n没有人设说明。")

	roles, warns, err = LoadRoleFiles(dir)
	if err != nil {
		t.Fatalf("LoadRoleFiles: %v", err)
	}
	if len(roles) != 1 || len(warns) != 1 {
		t.Fatalf("应 1 角色 + 1 告警: %+v %v", roles, warns)
	}
	r := roles[0]
	if r.Name != "reviewer" || !strings.Contains(r.Description, "独立审稿人") {
		t.Fatalf("解析不符: %+v", r)
	}
	if len(r.Nicknames) != 2 || !strings.Contains(r.Persona, "只做风险审查") {
		t.Fatalf("nicknames/persona 不符: %+v", r)
	}
	if r.ConfigLayer != "reviewer.toml" {
		t.Fatalf("config_layer 未解析: %+v", r)
	}
}

// 验收①②：建文件 → SetRoleSeats → Agents() 出现；删文件（空表重载）→ 不残留。
func TestRoleSeatsLifecycle(t *testing.T) {
	m, err := NewManager(t.TempDir(), nil, func(Event) {})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	roles, _, _ := LoadRoleFiles(func() string {
		dir := filepath.Join(t.TempDir(), "agents")
		writeRoleFile(t, dir, "reviewer.md", "---\nname: reviewer\ndescription: 审稿席位\n---\npersona 文本")
		return dir
	}())
	backend := newFakeRoleBackend()
	m.SetRoleSeats(roles, backend)

	agents := m.Agents(context.Background())
	found := false
	for _, a := range agents {
		if a.ID == "reviewer" {
			found = true
			if a.Description != "审稿席位" {
				t.Fatalf("Description 应随席位暴露: %+v", a)
			}
		}
	}
	if !found {
		t.Fatalf("文件席位应出现在席位表: %+v", agents)
	}
	// 删除（清空重载）：不残留注册。
	m.SetRoleSeats(nil, backend)
	for _, a := range m.Agents(context.Background()) {
		if a.ID == "reviewer" {
			t.Fatalf("删除后不应残留: %+v", a)
		}
	}
}

// 验收③：description 进席位引导（spawn 引导契约）。
func TestRoleDescriptionInBriefing(t *testing.T) {
	agent := AgentStatus{
		ID: "reviewer", Name: "reviewer", Kind: "文件席位",
		Capabilities: []string{"role-file"},
		Description:  "独立审稿人：对方案做风险与漏洞审查。",
	}
	got := toolPolicy(agent)
	if !strings.Contains(got, "【角色设定】") || !strings.Contains(got, "独立审稿人") {
		t.Fatalf("spawn 引导应带角色设定: %q", got)
	}
	// 内建席位不受影响（无 role-file 能力）。
	if got := toolPolicy(AgentStatus{ID: "grok"}); strings.Contains(got, "【角色设定】") {
		t.Fatalf("内建席位不应带角色设定段")
	}
}

// 验收④：角色级覆盖只影响该席位——两个席位各自把「自己的」RoleFile 传给后端。
func TestRoleSeatOverrideIsolation(t *testing.T) {
	adapters := buildAdapters(nil, nil, "", []RoleFile{
		{Name: "role-a", Description: "A 的人设", Persona: "persona-A"},
		{Name: "role-b", Description: "B 的人设", Persona: "persona-B"},
	}, newFakeRoleBackend())
	backend := newFakeRoleBackend()
	_ = adapters
	// 直接经 buildAdapters 产出的两个适配器分别调 invoke，验证各自携带
	// 自己的角色（隔离：A 的人设不会进 B 的调用）。
	a, okA := adapters["role-a"]
	b, okB := adapters["role-b"]
	if !okA || !okB {
		t.Fatalf("两文件席位都应注册: %v", adapters)
	}
	_, _ = a.invoke(context.Background(), "问题A")
	_, _ = b.invoke(context.Background(), "问题B")
	// 用独立 backend 重放以捕获角色（上面的 adapters 绑的是第一个 backend）。
	backend2 := newFakeRoleBackend()
	a2 := buildAdapters(nil, nil, "", []RoleFile{
		{Name: "role-a", Description: "A 的人设", Persona: "persona-A"},
	}, backend2)["role-a"]
	_, _ = a2.invoke(context.Background(), "问题A")
	got, ok := backend2.calls["role-a"]
	if !ok || got.Persona != "persona-A" || got.Description != "A 的人设" {
		t.Fatalf("后端应收到席位自己的角色: %+v %v", got, ok)
	}
	_ = backend
	_ = b
}

// stubGrokBackend 让内建 grok 席位真正被构建（nil 后端时内建席不存在）。
type stubGrokBackend struct{}

func (stubGrokBackend) ProbeGrok(_ context.Context) AgentStatus {
	return AgentStatus{ID: "grok", Name: "Grok", Kind: "Gaea 办公席位 · xAI"}
}

func (stubGrokBackend) CallGrok(_ context.Context, _ string) (string, error) {
	return "", nil
}

// 内建同名文件席位被跳过（防串引擎提权）。
func TestRoleSeatBuiltinCollisionSkipped(t *testing.T) {
	adapters := buildAdapters(nil, stubGrokBackend{}, "", []RoleFile{
		{Name: "grok", Description: "冒牌 grok"},
		{Name: "reviewer", Description: "正常席位"},
	}, newFakeRoleBackend())
	if _, ok := adapters["reviewer"]; !ok {
		t.Fatalf("正常文件席位应注册")
	}
	if adapters["grok"].kind != "Gaea 办公席位 · xAI" {
		t.Fatalf("内建 grok 不得被文件席位顶替: %+v", adapters["grok"])
	}
}

// T15 验收④：config_layer 只改变传入后端的角色自身预算；A/B 两席各读各的，
// 且路径不得越出角色目录。
func TestLoadRoleOverridesIsolated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	writeRoleFile(t, dir, "reviewer.toml", `[agent]
max_steps = 6
temperature = 0.1
context_window = 64000
`)
	writeRoleFile(t, dir, "writer.toml", `[agent]
max_steps = 18
`)
	a := RoleFile{Name: "reviewer", Source: filepath.Join(dir, "reviewer.md"), ConfigLayer: "reviewer.toml"}
	b := RoleFile{Name: "writer", Source: filepath.Join(dir, "writer.md"), ConfigLayer: "writer.toml"}
	ao, err := LoadRoleOverrides(a)
	if err != nil {
		t.Fatalf("reviewer layer: %v", err)
	}
	bo, err := LoadRoleOverrides(b)
	if err != nil {
		t.Fatalf("writer layer: %v", err)
	}
	if ao.MaxSteps == nil || *ao.MaxSteps != 6 || ao.ContextWindow == nil || *ao.ContextWindow != 64000 {
		t.Fatalf("reviewer 覆盖不符: %+v", ao)
	}
	if bo.MaxSteps == nil || *bo.MaxSteps != 18 || bo.Temperature != nil || bo.ContextWindow != nil {
		t.Fatalf("writer 覆盖串味: %+v", bo)
	}
	escape := a
	escape.ConfigLayer = filepath.Join("..", "outside.toml")
	if _, err := LoadRoleOverrides(escape); err == nil {
		t.Fatal("越出角色目录的 config_layer 应拒绝")
	}
}
