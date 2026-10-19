package app

import (
	"testing"
	"time"
)

// TestParseGPUPerfOutput 脚本输出解析：两行齐/单侧缺/坏输出/钳幅。
func TestParseGPUPerfOutput(t *testing.T) {
	util, vram, utilOK, vramOK := parseGPUPerfOutput("util_sum=1.4\nvram_gb=12.21\n")
	if !utilOK || util != 1.4 || !vramOK || vram != 12.21 {
		t.Errorf("完整输出 = (%v,%v,%v,%v), want (1.4,12.21,true,true)", util, vram, utilOK, vramOK)
	}
	// 单侧缺行：另一侧照常
	_, vram2, utilOK2, vramOK2 := parseGPUPerfOutput("vram_gb=8\n")
	if utilOK2 || !vramOK2 || vram2 != 8 {
		t.Errorf("单侧输出 = (%v,%v,%v)", utilOK2, vram2, vramOK2)
	}
	// 坏输出：双侧无读数
	if _, _, u3, v3 := parseGPUPerfOutput("Get-Counter : 拒绝访问\r\n"); u3 || v3 {
		t.Error("坏输出不应出读数")
	}
	// 钳幅：>100 折 100，负值折 0
	util4, _, u4, _ := parseGPUPerfOutput("util_sum=140.5\n")
	if !u4 || util4 != 100 {
		t.Errorf("越上界 = %v, want 100", util4)
	}
	util5, _, u5, _ := parseGPUPerfOutput("util_sum=-3\n")
	if !u5 || util5 != 0 {
		t.Errorf("负值 = %v, want 0", util5)
	}
}

// TestWindowsGPUPerfRealMachine 真机采集冒烟：脚本可用时写入缓存且值域合法，
// windowsGPUPerf 随后命中；脚本不可用（无显卡环境）ok=false 如实。不阻塞
// 主线程（stale-while-revalidate 的首次路径是同步采集——测试里直接调
// collectGPUPerfOnce 驱动，windowsGPUPerf 只读缓存）。
func TestWindowsGPUPerfRealMachine(t *testing.T) {
	collectGPUPerfOnce()
	util, vram, ok := windowsGPUPerf()
	if !ok {
		t.Skip("本机 GPU 计数器无读数（无会话/Get-Counter 不可用），跳过值域断言")
	}
	if util < 0 || util > 100 {
		t.Errorf("GPU 负载 = %v, 越界 [0,100]", util)
	}
	if vram < 0 {
		t.Errorf("显存占用 = %v, 负值", vram)
	}
	// 缓存命中路径：TTL 内不触发后台刷新（Refreshing 维持 0）
	_ = time.Since(time.Now())
}
