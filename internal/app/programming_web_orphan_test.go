package app

// programming_web_orphan_test.go — DSH Web 遗孤回收行为锁（2026-11-23 根修
// 「外部浏览器打不开/窗口被占用」）。分支：
//   1. 端口被占 + 占用者=自家 dsh web（命令行=本根入口+本端口）→ 树杀回收、
//      端口随后释放 → 不再报「占用」，走后续 preflight（临时根必失败=退出锚）；
//   2. 端口被占 + 占用者=异进程 → 诚实拒绝，错误附 PID；
//   3. netstat 解析纯函数（监听行/形态噪声）。

import (
	"strings"
	"testing"
)

// orphanSeam 包一层四缝替换（portOpen 用可控开关：killTree 调用后翻 false
// 模拟回收生效）。
func orphanSeam(t *testing.T, our bool, portAlways bool) *int {
	t.Helper()
	oldOpen := programmingWebPortOpenFn
	oldOwner := programmingWebPortOwnerFn
	oldIsOur := programmingWebIsOurPIDFn
	oldKill := programmingWebKillTreeFn
	open := true
	killedPID := 0
	programmingWebPortOpenFn = func() bool { return portAlways || open }
	programmingWebPortOwnerFn = func() int { return 9999 }
	programmingWebIsOurPIDFn = func(pid int) bool { return our && pid == 9999 }
	programmingWebKillTreeFn = func(pid int) error {
		killedPID = pid
		open = false // 回收成功 → 端口释放
		return nil
	}
	t.Cleanup(func() {
		programmingWebPortOpenFn = oldOpen
		programmingWebPortOwnerFn = oldOwner
		programmingWebIsOurPIDFn = oldIsOur
		programmingWebKillTreeFn = oldKill
	})
	return &killedPID
}

func TestProgrammingWebOrphanReclaimed(t *testing.T) {
	t.Setenv("GAEA_DSH_ROOT", t.TempDir()) // 空根：preflight 必失败=回收后退出锚
	resetProgrammingWebTestState(t)
	t.Cleanup(func() { resetProgrammingWebTestState(t) })
	killed := orphanSeam(t, true, false)

	err := (&App{}).StartProgrammingWeb()
	if err == nil {
		t.Fatal("临时空根 preflight 应失败（退出锚），不应真启动")
	}
	if strings.Contains(err.Error(), "占用") {
		t.Fatalf("自家遗孤应被回收而不是报占用: %v", err)
	}
	if *killed != 9999 {
		t.Fatalf("遗孤未被树杀: killed=%d", *killed)
	}
}

func TestProgrammingWebForeignHolderStillRefused(t *testing.T) {
	t.Setenv("GAEA_DSH_ROOT", t.TempDir())
	resetProgrammingWebTestState(t)
	t.Cleanup(func() { resetProgrammingWebTestState(t) })
	killed := orphanSeam(t, false, true) // 异进程：端口恒占

	err := (&App{}).StartProgrammingWeb()
	if err == nil || !strings.Contains(err.Error(), "已被进程 9999 占用") {
		t.Fatalf("异进程占用应诚实拒绝并附 PID: %v", err)
	}
	if *killed != 0 {
		t.Fatalf("异进程不得树杀: killed=%d", *killed)
	}
}

func TestParseNetstatListenPID(t *testing.T) {
	out := strings.Join([]string{
		"  TCP    0.0.0.0:135          0.0.0.0:0              LISTENING       1234",
		"  TCP    127.0.0.1:3091       0.0.0.0:0              LISTENING       57984",
		"  TCP    127.0.0.1:3091       127.0.0.1:52300        ESTABLISHED     999",
		"  TCP    [::1]:3091           [::]:0                 LISTENING       777",
		"",
	}, "\n")
	if got := parseNetstatListenPID(out, 3091); got != 57984 {
		t.Fatalf("IPv4 监听行应命中 57984，got %d（IPv6 行与 ESTABLISHED 行不得干扰）", got)
	}
	if got := parseNetstatListenPID(out, 135); got != 0 {
		t.Fatalf("0.0.0.0 绑定非环回，不在判据内应 0（dsh web 恒绑 127.0.0.1）: %d", got)
	}
	if got := parseNetstatListenPID(out, 9999); got != 0 {
		t.Fatalf("未监听端口应 0: %d", got)
	}
}
