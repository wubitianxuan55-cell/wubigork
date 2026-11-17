package memory

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// T12（codex 蒸馏）验收五条。

// ① 12 条真实形态假密钥脱敏后按原文零命中；② 5 条普通文本不误伤。
func TestSanitizeRedactsSecrets(t *testing.T) {
	secrets := []string{
		"sk-abcdefghijklmnopqrstuvwxyz012345",                 // sk- 长密钥
		"sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZ123456",            // sk-proj 变体
		"AKIAIOSFODNN7EXAMPLE",                                // AWS AccessKeyId（16 位数字大写）
		"AKIA1234567890ABCDEF",                                // AWS 形态 2
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test.sig", // Bearer JWT
		"Bearer abcdefghijklmnopqrstuvwxyz123456",             // Bearer token 体
		`api_key = "supersecretvalue123"`,                     // 赋值形态带引号
		"api-key: 0123456789abcdef",                           // 连字符键
		"API_KEY=abcdefgh12345678",                            // 大写键无空格
		`"token": "ghp_abcdefghijklmnopqrstuvwxyz"`,           // JSON 形态
		"password: hunter2secretvault",                        // password 键
		`secret='my-super-secret-value'`,                      // 单引号形态
	}
	for i, s := range secrets {
		got, n := Sanitize("前缀 " + s + " 后缀")
		if n == 0 {
			t.Fatalf("样例 %d 未脱敏: %s", i, s)
		}
		if strings.Contains(got, s) {
			t.Fatalf("样例 %d 原文仍可命中: %s", i, s)
		}
		if !strings.Contains(got, redactedPlaceholder) {
			t.Fatalf("样例 %d 缺占位符: %s", i, got)
		}
	}
}

func TestSanitizeLeavesNormalText(t *testing.T) {
	normal := []string{
		"小说里的 token 概念指的是语言的最小单位",                      // token 词面无赋值
		"The API key lives in ~/.config/gaea/api.key.", // 键词无值
		"用户偏好 key=value 形式的配置说明",                       // 值太短（<8）
		"sk 板块与 sk- 短前缀不是密钥",                           // sk- 后不足 20 位
		"bearer token 的规范在 RFC 6750 文档里",               // bearer 后无 token 体
	}
	for i, s := range normal {
		got, n := Sanitize(s)
		if n != 0 || got != s {
			t.Fatalf("样例 %d 被误伤: %q → %q (n=%d)", i, s, got, n)
		}
	}
}

// 赋值形态保留键名与分隔符（信息量纪律：看得出有字段，拿不到值）。
func TestSanitizeKeepsKeyNames(t *testing.T) {
	got, _ := Sanitize(`api_key = "supersecretvalue123"`)
	if !strings.Contains(got, "api_key") || !strings.Contains(got, "=") {
		t.Fatalf("键名/分隔符应保留: %q", got)
	}
	if !strings.Contains(got, redactedPlaceholder) {
		t.Fatalf("值应替换: %q", got)
	}
}

// ③ 背压门四条拒绝路径 + 限流拿不到按通过（务实默认）。
func TestGateCheck(t *testing.T) {
	cases := []struct {
		in     GateInput
		wantOK bool
		reason string
	}{
		{GateInput{Ephemeral: true, Enabled: true}, false, "skipped: ephemeral"},
		{GateInput{Subagent: true, Enabled: true}, false, "skipped: subagent"},
		{GateInput{Enabled: false}, false, "skipped: disabled"},
		{GateInput{Enabled: true, RateLimited: boolPtr(false)}, false, "skipped: rate_limited"},
		{GateInput{Enabled: true, RateLimited: boolPtr(true)}, true, ""},
		{GateInput{Enabled: true}, true, ""}, // nil 限流 = 拿不到 → 通过
	}
	for i, tc := range cases {
		got := GateCheck(tc.in)
		if got.OK != tc.wantOK || got.Reason != tc.reason {
			t.Fatalf("样例 %d = %+v, want ok=%v reason=%q", i, got, tc.wantOK, tc.reason)
		}
	}
	// 会话类型门优先于限流。
	if d := GateCheck(GateInput{Ephemeral: true, RateLimited: boolPtr(true)}); d.Reason != "skipped: ephemeral" {
		t.Fatalf("会话门应优先: %+v", d)
	}
}

func boolPtr(b bool) *bool { return &b }

// ④ 基线差分三类计数 + 阈值拦截（删除>10% / 修改>30% 不应用）。
func TestBaselineDiffAndThreshold(t *testing.T) {
	mk := func(i int, body string) Memory {
		return Memory{Name: "mem-" + string(rune('a'+i)), Body: body}
	}
	before := SnapshotBaseline([]Memory{mk(0, "v1"), mk(1, "v1"), mk(2, "v1"), mk(3, "v1"), mk(4, "v1")})
	// 修改 1 条（20%>10% 但<30% ✓）+ 新增 1 条：放行。
	after := SnapshotBaseline([]Memory{mk(0, "v1"), mk(1, "CHANGED"), mk(2, "v1"), mk(3, "v1"), mk(4, "v1"), mk(5, "new")})
	d := DiffBaseline(before, after)
	if len(d.Added) != 1 || len(d.Modified) != 1 || len(d.Removed) != 0 {
		t.Fatalf("差分类别不符: %+v", d)
	}
	if !DiffAllowed(d, len(before)) {
		t.Fatalf("20%% 修改应放行")
	}
	// 删除 1/5 = 20% > 10% → 拦截。
	after2 := SnapshotBaseline([]Memory{mk(0, "v1"), mk(1, "v1"), mk(2, "v1"), mk(3, "v1")})
	d2 := DiffBaseline(before, after2)
	if DiffAllowed(d2, len(before)) {
		t.Fatalf("20%% 删除应拦截")
	}
	// 修改 2/5 = 40% > 30% → 拦截。
	after3 := SnapshotBaseline([]Memory{mk(0, "X"), mk(1, "Y"), mk(2, "v1"), mk(3, "v1"), mk(4, "v1")})
	if DiffAllowed(DiffBaseline(before, after3), len(before)) {
		t.Fatalf("40%% 修改应拦截")
	}
	// 空基线：任何删除/修改都超线（防清库形态——基线 5 条被清到 0）。
	empty := DiffBaseline(before, map[string]string{})
	if DiffAllowed(empty, len(before)) {
		t.Fatalf("清库形态应拦截")
	}
}

// ⑤ 差分渲染有界且按字符边界截断（中文不出半个字，UTF-8 合法）。
func TestRenderBaselineDiffBoundedUTF8(t *testing.T) {
	var mems []Memory
	for i := 0; i < 200; i++ {
		// 名字带序号保证快照键唯一——200 行差分足以超过 300 字节上限。
		mems = append(mems, Memory{Name: "记忆条目-" + strings.Repeat("长", 20) + "-" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Body: "x"})
	}
	d := DiffBaseline(map[string]string{}, SnapshotBaseline(mems))
	if len(d.Added) != 200 {
		t.Fatalf("快照键应唯一: %d", len(d.Added))
	}
	out := RenderBaselineDiff(d, 300)
	if len(out) > 300 {
		t.Fatalf("渲染超上限: %d", len(out))
	}
	if !utf8.ValidString(out) {
		t.Fatalf("渲染切出非法 UTF-8")
	}
	if !strings.Contains(out, "已截断") {
		t.Fatalf("截断应标注: %q", out)
	}
}

// Save 单一写口脱敏常开：入库后按原文检索零命中。
func TestSaveSanitizesOnWrite(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	secret := "sk-abcdefghijklmnopqrstuvwxyz012345"
	_, err := store.Save(Memory{
		Name: "leak-test", Title: "泄漏测试", Type: TypeReference, Kind: KindSemantic,
		Description: "包含密钥 " + secret, Body: "密钥原文 " + secret,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok := store.Get("leak-test")
	if !ok {
		t.Fatalf("读取失败")
	}
	if strings.Contains(got.Body, secret) || strings.Contains(got.Description, secret) {
		t.Fatalf("入库后原文仍可命中: %+v", got)
	}
}
