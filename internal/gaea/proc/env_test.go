package proc

import "testing"

// find 在环境列表里取键值；ok=false 表示不存在。
func find(env []string, key string) (string, bool) {
	for _, kv := range env {
		if k, v, ok := cut(kv); ok && k == key {
			return v, true
		}
	}
	return "", false
}

func cut(kv string) (string, string, bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return kv, "", false
}

// 注入后 PATH 保留（不删父环境，用户 PATH/代理必须存活）。
func TestCommandEnvKeepsPATH(t *testing.T) {
	t.Setenv("PATH", `C:\bin;C:\windows`)
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	env := CommandEnv()
	if _, ok := find(env, "PATH"); !ok {
		t.Fatalf("PATH 丢失: %v", env)
	}
	if v, _ := find(env, "HTTPS_PROXY"); v != "http://127.0.0.1:7890" {
		t.Fatalf("代理变量丢失: %q", v)
	}
}

// 固定覆盖生效：TERM=dumb；COLORTERM 存在且为空（不是删除）。
func TestCommandEnvDeterministicOverrides(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("PAGER", "less")
	env := CommandEnv()
	if v, _ := find(env, "TERM"); v != "dumb" {
		t.Fatalf("TERM 未覆盖: %q", v)
	}
	v, ok := find(env, "COLORTERM")
	if !ok {
		t.Fatalf("COLORTERM 必须存在（空值语义）")
	}
	if v != "" {
		t.Fatalf("COLORTERM 必须为空, got %q", v)
	}
	if v, _ := find(env, "PAGER"); v != "cat" {
		t.Fatalf("PAGER 未覆盖: %q", v)
	}
	// 同键只出现一次（父变量被剔除，无重复项）。
	n := 0
	for _, kv := range env {
		if k, _, _ := cut(kv); k == "TERM" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("TERM 出现 %d 次, 应为 1", n)
	}
}

// extra 后到者优先，且同名父变量让位（探针变量不可被固定项遮蔽）。
func TestCommandEnvExtraWins(t *testing.T) {
	t.Setenv("GAEA_STATE_OUT", "stale")
	env := CommandEnv("GAEA_STATE_OUT=probe-42")
	if v, _ := find(env, "GAEA_STATE_OUT"); v != "probe-42" {
		t.Fatalf("extra 未生效: %q", v)
	}
	n := 0
	for _, kv := range env {
		if k, _, _ := cut(kv); k == "GAEA_STATE_OUT" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("GAEA_STATE_OUT 出现 %d 次, 应为 1", n)
	}
}

// 转义阀门：GAEA_EXEC_ENV_RAW=1 跳过注入，父 TERM 原样保留。
func TestCommandEnvRawValve(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("GAEA_EXEC_ENV_RAW", "1")
	env := CommandEnv()
	if v, _ := find(env, "TERM"); v != "xterm-256color" {
		t.Fatalf("阀门未生效, TERM=%q", v)
	}
	if _, ok := find(env, "GAEA_CI"); ok {
		t.Fatalf("阀门开启时不应注入 GAEA_CI")
	}
}

// DeterministicEnv 返回固定 10 项且顺序稳定（比对依赖顺序不变）。
func TestDeterministicEnvShape(t *testing.T) {
	env := DeterministicEnv()
	if len(env) != 10 {
		t.Fatalf("固定变量应为 10 项, got %d", len(env))
	}
	want := []string{"NO_COLOR=1", "TERM=dumb", "LANG=C.UTF-8", "LC_CTYPE=C.UTF-8",
		"LC_ALL=C.UTF-8", "COLORTERM=", "PAGER=cat", "GIT_PAGER=cat", "GH_PAGER=cat", "GAEA_CI=1"}
	for i, kv := range want {
		if env[i] != kv {
			t.Fatalf("第 %d 项 = %q, want %q", i, env[i], kv)
		}
	}
}
