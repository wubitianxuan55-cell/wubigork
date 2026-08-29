package config

import (
	"reflect"
	"strings"
	"testing"
)

// TestPlayGuardrailsParse 断言 [space_profiles.<space>.guardrails] 段解析：
// 五字段随段解析进 SpaceProfile.Guardrails，未配置段的空间 Guardrails 为 nil。
func TestPlayGuardrailsParse(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play.guardrails]
enabled = true
temperature_max = 0.7
max_output_tokens = 4096
image_safe_mode = true
persona_lock = true
`)
	pp, err := cfg.SpaceProfile("play")
	if err != nil {
		t.Fatalf("SpaceProfile(play): %v", err)
	}
	if pp.Guardrails == nil {
		t.Fatal("play guardrails 段未解析")
	}
	want := PlayGuardrails{
		Enabled:         true,
		TemperatureMax:  0.7,
		MaxOutputTokens: 4096,
		ImageSafeMode:   true,
		PersonaLock:     true,
	}
	if *pp.Guardrails != want {
		t.Errorf("play guardrails = %+v, want %+v", *pp.Guardrails, want)
	}

	// 未配置 guardrails 的空间：nil = 未配置（零钳制）。
	wp, err := cfg.SpaceProfile("work")
	if err != nil {
		t.Fatalf("SpaceProfile(work): %v", err)
	}
	if wp.Guardrails != nil {
		t.Errorf("work guardrails 应为 nil（未配置），got %+v", wp.Guardrails)
	}
}

// TestPlayGuardrailsForSpace 取值点语义：非 play 空间 / mode=off / 段未配置 /
// enabled=false → 零值（零钳制 = 现状逐字节）；配置且 enabled=true → 段值原样。
func TestPlayGuardrailsForSpace(t *testing.T) {
	zero := PlayGuardrails{}

	// 缺省（Default 无段）：全部零值。
	d := Default()
	for _, space := range []string{"play", "work", "", "nope"} {
		if g := d.PlayGuardrails(space); g != zero {
			t.Errorf("缺省 PlayGuardrails(%q) = %+v, want 零值", space, g)
		}
	}

	// 段配置且 enabled=true → 段值原样。
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play.guardrails]
enabled = true
temperature_max = 0.7
max_output_tokens = 2048
image_safe_mode = true
persona_lock = true
`)
	want := PlayGuardrails{
		Enabled:         true,
		TemperatureMax:  0.7,
		MaxOutputTokens: 2048,
		ImageSafeMode:   true,
		PersonaLock:     true,
	}
	if g := cfg.PlayGuardrails("play"); g != want {
		t.Errorf("PlayGuardrails(play) = %+v, want %+v", g, want)
	}
	// 大小写/空白不敏感。
	if g := cfg.PlayGuardrails(" PLAY "); g.TemperatureMax != 0.7 {
		t.Errorf("PlayGuardrails( PLAY ) 应大小写不敏感: %+v", g)
	}
	// 非 play 空间恒零值（护栏只由 play 域生成点消费），即使 work 段也写了。
	cfg.SpaceProfiles["work"] = SpaceProfile{Guardrails: &want}
	if g := cfg.PlayGuardrails("work"); g != zero {
		t.Errorf("PlayGuardrails(work) 应为零值（非 play 域不消费）: %+v", g)
	}

	// enabled=false（总开关关）→ 零值。
	off := decodeSpaceProfiles(t, `
[space_profiles.play.guardrails]
enabled = false
temperature_max = 0.7
`)
	if g := off.PlayGuardrails("play"); g != zero {
		t.Errorf("enabled=false 应零值: %+v", g)
	}

	// space.mode=off → 零值（空间维度整体关闭，恒等现状）。
	moff := decodeSpaceProfiles(t, `
[space]
mode = "off"

[space_profiles.play.guardrails]
enabled = true
temperature_max = 0.7
`)
	if g := moff.PlayGuardrails("play"); g != zero {
		t.Errorf("space.mode=off 应零值: %+v", g)
	}
}

// TestPlayGuardrailsRenderRoundTrip RenderTOML → 重新解码往返保真（Save() 走
// RenderTOML，丢段即丢护栏），且 PlayGuardrails("play") 生效值贯通、渲染确定。
func TestPlayGuardrailsRenderRoundTrip(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play]
chat = "deepseek-pro/deepseek-v4-pro"

[space_profiles.play.guardrails]
enabled = true
temperature_max = 0.7
max_output_tokens = 4096
image_safe_mode = true
persona_lock = true
`)
	out := RenderTOML(cfg)
	got := decodeSpaceProfiles(t, out)
	pp, err := got.SpaceProfile("play")
	if err != nil || pp.Chat != "deepseek-pro/deepseek-v4-pro" {
		t.Fatalf("play 往返失真: (%+v, %v)", pp, err)
	}
	want := PlayGuardrails{
		Enabled:         true,
		TemperatureMax:  0.7,
		MaxOutputTokens: 4096,
		ImageSafeMode:   true,
		PersonaLock:     true,
	}
	if pp.Guardrails == nil || *pp.Guardrails != want {
		t.Fatalf("guardrails 往返失真: %+v, want %+v", pp.Guardrails, want)
	}
	if g := got.PlayGuardrails("play"); g != want {
		t.Errorf("PlayGuardrails(play) 往返后 = %+v, want %+v", g, want)
	}
	// 渲染确定性（两次输出逐字节一致）。
	if again := RenderTOML(got); again != out {
		t.Fatalf("RenderTOML 应确定性输出")
	}
	// 未配置 guardrails 的 profile 不渲染 guardrails 段（缺省 = 无输出）。
	plain := decodeSpaceProfiles(t, `
[space_profiles.work]
chat = "deepseek-flash"
`)
	if out := RenderTOML(plain); containsActiveGuardrailsSection(out) {
		t.Errorf("未配置 guardrails 不应渲染段:\n%s", out)
	}
}

// TestPlayGuardrailsRenderDisabledFidelity enabled=false（含显式零值字段）
// 恒活动渲染保真：Save 往返不丢段、不丢显式配置（hard_ask 空集同款处理）。
func TestPlayGuardrailsRenderDisabledFidelity(t *testing.T) {
	cfg := decodeSpaceProfiles(t, `
[space_profiles.play.guardrails]
enabled = false
temperature_max = 0.7
`)
	out := RenderTOML(cfg)
	if !containsActiveGuardrailsSection(out) {
		t.Fatalf("guardrails 段应恒活动渲染（往返不丢段）:\n%s", out)
	}
	got := decodeSpaceProfiles(t, out)
	pp, err := got.SpaceProfile("play")
	if err != nil || pp.Guardrails == nil {
		t.Fatalf("enabled=false 段往返丢失: (%+v, %v)", pp.Guardrails, err)
	}
	want := PlayGuardrails{Enabled: false, TemperatureMax: 0.7}
	if !reflect.DeepEqual(*pp.Guardrails, want) {
		t.Errorf("往返值 = %+v, want %+v（显式配置不丢）", *pp.Guardrails, want)
	}
	if g := got.PlayGuardrails("play"); g != (PlayGuardrails{}) {
		t.Errorf("enabled=false 生效值应零值: %+v", g)
	}
}

// containsActiveGuardrailsSection 判断渲染输出中是否存在活动
// [space_profiles.*.guardrails] 段头（注释行不算）。
func containsActiveGuardrailsSection(rendered string) bool {
	for _, line := range strings.Split(rendered, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[space_profiles.") && strings.HasSuffix(line, ".guardrails]") {
			return true
		}
	}
	return false
}
