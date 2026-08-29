package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

// decodeSpaceProfiles 把 TOML 片段解码到 Default() 配置上（模拟 mergeFile 的
// 项目配置叠加路径），返回可直接断言的 *Config。
func decodeSpaceProfiles(t *testing.T, tomlSrc string) *Config {
	t.Helper()
	cfg := Default()
	if _, err := toml.Decode(tomlSrc, cfg); err != nil {
		t.Fatalf("解码 TOML: %v", err)
	}
	return cfg
}

// TestSpaceProfilesParse 断言 [space_profiles.<space>] 段解析：模型键按
// feature_model_handler 功能域命名，permissions 子段随段解析。
func TestSpaceProfilesParse(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.work]
chat  = "deepseek-flash/deepseek-v4-flash"
office = "mimo-pro"

[space_profiles.play]
chat    = "deepseek-pro/deepseek-v4-pro"
whisper = "mimo-flash"
novel   = "deepseek-pro"
permissions.mode = "allow"
permissions.hard_ask = []
permissions.approval_timeout_secs = 42
permissions.allow = ["bash(go test*)"]
`)
	wp, err := cfg.SpaceProfile("work")
	if err != nil {
		t.Fatalf("SpaceProfile(work): %v", err)
	}
	if wp.Chat != "deepseek-flash/deepseek-v4-flash" || wp.Office != "mimo-pro" {
		t.Errorf("work profile = %+v", wp)
	}
	pp, err := cfg.SpaceProfile("play")
	if err != nil {
		t.Fatalf("SpaceProfile(play): %v", err)
	}
	if pp.Whisper != "mimo-flash" || pp.Novel != "deepseek-pro" || pp.Chat != "deepseek-pro/deepseek-v4-pro" {
		t.Errorf("play profile = %+v", pp)
	}
	if pp.Permissions == nil {
		t.Fatal("play permissions 段未解析")
	}
	if pp.Permissions.Mode != "allow" || pp.Permissions.ApprovalTimeoutSecs != 42 {
		t.Errorf("play permissions = %+v", pp.Permissions)
	}
	// 显式空数组必须保真为非 nil 空切片（play 不弹审批卡的语义字段）。
	if pp.Permissions.HardAsk == nil || len(pp.Permissions.HardAsk) != 0 {
		t.Errorf("hard_ask = %#v, want 非 nil 空切片", pp.Permissions.HardAsk)
	}
}

// TestSpaceProfileDefaultZero 缺省零值：段缺失/空配置 → 零值 profile +
// nil error（现状逐字节回退）；非法空间 → error；大小写不敏感匹配。
func TestSpaceProfileDefaultZero(t *testing.T) {
	cfg := Default()
	p, err := cfg.SpaceProfile("work")
	if err != nil || p == nil {
		t.Fatalf("SpaceProfile(work) = (%v, %v), want 零值+nil", p, err)
	}
	if *p != (SpaceProfile{}) {
		t.Errorf("缺省 profile 应为零值: %+v", *p)
	}
	if p, err := cfg.SpaceProfile(""); err != nil || *p != (SpaceProfile{}) {
		t.Fatalf("mode=off 空间应返回零值 profile: (%v, %v)", p, err)
	}
	if _, err := cfg.SpaceProfile("nope"); err == nil {
		t.Fatal("非法空间应报错")
	}
	cfg.SpaceProfiles = map[string]SpaceProfile{"PLAY": {Chat: "x"}}
	if p, err := cfg.SpaceProfile(" play "); err != nil || p.Chat != "x" {
		t.Fatalf("空间键应大小写不敏感并 trim: (%+v, %v)", p, err)
	}
}

// TestPermissionsForSpaceDefaults 缺省=现状单 Policy：
//   - mode=off（""）与 work 未配置段 → 顶层 [permissions] 原样（HardAsk nil）；
//   - play 未配置段 → 产品默认 mode=allow + hard_ask 空集，规则列表继承顶层
//     （deny 硬拒绝仍生效）。
func TestPermissionsForSpaceDefaults(t *testing.T) {
	cfg := Default() // Permissions: mode=ask, allow=[run_skill]
	cfg.Permissions.Deny = []string{"bash(rm -rf*)"}

	off := cfg.PermissionsForSpace("")
	if off.Mode != "ask" || len(off.Allow) != 1 || off.Allow[0] != "run_skill" {
		t.Errorf("mode=off 应原样回退顶层: %+v", off)
	}
	if off.HardAsk != nil || off.ApprovalTimeoutSecs != 0 {
		t.Errorf("mode=off HardAsk/Timeout 应为未配置: %+v", off)
	}

	work := cfg.PermissionsForSpace("work")
	if work.Mode != "ask" || work.Allow[0] != "run_skill" || work.HardAsk != nil {
		t.Errorf("work 缺省应=现状单 Policy: %+v", work)
	}

	play := cfg.PermissionsForSpace("play")
	if play.Mode != "allow" {
		t.Errorf("play 缺省 mode = %q, want allow（不弹审批卡）", play.Mode)
	}
	if play.HardAsk == nil || len(play.HardAsk) != 0 {
		t.Errorf("play 缺省 hard_ask 应为显式空集: %#v", play.HardAsk)
	}
	if len(play.Deny) != 1 || play.Deny[0] != "bash(rm -rf*)" {
		t.Errorf("play 应继承顶层 deny（硬拒绝仍生效）: %v", play.Deny)
	}
	if len(play.Allow) != 1 || play.Allow[0] != "run_skill" {
		t.Errorf("play 应继承顶层 allow: %v", play.Allow)
	}
}

// TestPermissionsForSpaceSection 段配置按字段生效：mode 显式覆盖 play 缺省、
// 规则列表顶层+段叠加、hard_ask 显式值优先、approval_timeout_secs 贯通。
func TestPermissionsForSpaceSection(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play.permissions]
mode = "deny"
hard_ask = ["cost_save"]
approval_timeout_secs = 42
allow = ["bash(go test*)"]
deny = ["bash(sudo*)"]

[space_profiles.work.permissions]
allow = ["bash(ls*)"]
`)
	play := cfg.PermissionsForSpace("play")
	if play.Mode != "deny" {
		t.Errorf("显式段 mode 应覆盖 play 缺省: %q", play.Mode)
	}
	if len(play.HardAsk) != 1 || play.HardAsk[0] != "cost_save" {
		t.Errorf("hard_ask 段值应生效: %v", play.HardAsk)
	}
	if play.ApprovalTimeoutSecs != 42 {
		t.Errorf("approval_timeout_secs = %d, want 42", play.ApprovalTimeoutSecs)
	}
	// 顶层 allow（Default: run_skill）+ 段 allow 叠加。
	if len(play.Allow) != 2 || play.Allow[0] != "run_skill" || play.Allow[1] != "bash(go test*)" {
		t.Errorf("allow 应顶层+段叠加: %v", play.Allow)
	}
	if len(play.Deny) != 1 || play.Deny[0] != "bash(sudo*)" {
		t.Errorf("段 deny 应生效: %v", play.Deny)
	}

	work := cfg.PermissionsForSpace("work")
	if work.Mode != "ask" {
		t.Errorf("work 段未写 mode 应回退顶层: %q", work.Mode)
	}
	if work.HardAsk != nil {
		t.Errorf("work 段未写 hard_ask 应回退默认集（nil）: %#v", work.HardAsk)
	}
	if len(work.Allow) != 2 || work.Allow[1] != "bash(ls*)" {
		t.Errorf("work allow 应顶层+段叠加: %v", work.Allow)
	}
}

// TestSpaceProfilesRenderRoundTrip RenderTOML → 重新解码往返保真（persist_allow
// 回写依赖 Save() 走 RenderTOML，丢段即丢策略）。
func TestSpaceProfilesRenderRoundTrip(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play]
gaea = "deepseek-pro"
permissions.mode = "allow"
permissions.hard_ask = []

[space_profiles.work]
chat = "deepseek-flash/deepseek-v4-flash"
`)
	out := RenderTOML(cfg)
	got := decodeSpaceProfiles(t, out)
	wp, err := got.SpaceProfile("work")
	if err != nil || wp.Chat != "deepseek-flash/deepseek-v4-flash" {
		t.Fatalf("work 往返失真: (%+v, %v)", wp, err)
	}
	pp, err := got.SpaceProfile("play")
	if err != nil || pp.Gaea != "deepseek-pro" {
		t.Fatalf("play 往返失真: (%+v, %v)", pp, err)
	}
	if pp.Permissions == nil || pp.Permissions.Mode != "allow" {
		t.Fatalf("play permissions 往返失真: %+v", pp.Permissions)
	}
	if pp.Permissions.HardAsk == nil || len(pp.Permissions.HardAsk) != 0 {
		t.Fatalf("hard_ask 显式空集应往返保真: %#v", pp.Permissions.HardAsk)
	}
	// 渲染确定性（两次输出逐字节一致）。
	if again := RenderTOML(got); again != out {
		t.Fatalf("RenderTOML 应确定性输出")
	}
}

// TestAddPermissionRuleForSpace persist_allow 按空间分段回写：有段写段内
// （幂等去重），无段/空空间写顶层，非法规则报错。
func TestAddPermissionRuleForSpace(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play.permissions]
mode = "ask"
`)
	if err := cfg.AddPermissionRuleForSpace("play", "allow", "bash(go build*)"); err != nil {
		t.Fatalf("段内回写失败: %v", err)
	}
	if err := cfg.AddPermissionRuleForSpace("play", "allow", "bash(go build*)"); err != nil {
		t.Fatalf("重复回写应幂等: %v", err)
	}
	if got := cfg.SpaceProfiles["play"].Permissions.Allow; len(got) != 1 || got[0] != "bash(go build*)" {
		t.Fatalf("段内 allow = %v, want 恰一条", got)
	}
	if len(cfg.Permissions.Allow) != 1 || cfg.Permissions.Allow[0] != "run_skill" {
		t.Fatalf("顶层 allow 不应被段内回写污染: %v", cfg.Permissions.Allow)
	}

	// 无段 → 顶层（现状路径）。
	if err := cfg.AddPermissionRuleForSpace("work", "allow", "bash(ls*)"); err != nil {
		t.Fatalf("顶层回写失败: %v", err)
	}
	if last := cfg.Permissions.Allow[len(cfg.Permissions.Allow)-1]; last != "bash(ls*)" {
		t.Fatalf("无段空间应写顶层 allow: %v", cfg.Permissions.Allow)
	}
	// mode=off（空空间）恒顶层。
	if err := cfg.AddPermissionRuleForSpace("", "deny", "bash(rm*)"); err != nil {
		t.Fatalf("空空间回写失败: %v", err)
	}
	if last := cfg.Permissions.Deny[len(cfg.Permissions.Deny)-1]; last != "bash(rm*)" {
		t.Fatalf("空空间应写顶层 deny: %v", cfg.Permissions.Deny)
	}
	// 非法规则报错（段内路径）。
	if err := cfg.AddPermissionRuleForSpace("play", "allow", "  "); err == nil {
		t.Fatal("空规则应报错")
	}
	// 未知列表报错。
	if err := cfg.AddPermissionRuleForSpace("play", "nope", "bash(x)"); err == nil {
		t.Fatal("未知列表应报错")
	}
}
