package app

import (
	"fmt"
	"log/slog"
)

// 图像生成内存压力预检（v4.409，观察池「iGPU 压力预检」）：iGPU 共享内存
// 架构下系统 RAM 就是本地模型权重的实际载体——提交时可用内存偏低=大概率
// 触发权重换出+冷载（v4.404.1 实录：服务端 20.5 分钟后才出图，客户端误报
// 超时）。本预检在 ComfyUI 提交口如实设预期：命中阈值发 imagegen:pressure
// 事件（前端节流提示）+WARN 落日志（分诊对齐）；只提示不拦截。

// systemMemoryReader 内存读取 seam（测试替换；生产恒为 readSystemMemoryMB）。
var systemMemoryReader = readSystemMemoryMB

// readSystemMemoryMB 读系统物理内存（MB）——复用 image_handler 侧既有的
// GlobalMemoryStatusEx 原生 API 采集（winMemoryStatusEx，弃 wmic 教训在案）。
// 读数不可信 ok=false，调用方静默跳过预检。
func readSystemMemoryMB() (availMB, totalMB uint64, ok bool) {
	totalGB, usedGB := getMemoryStats()
	if totalGB <= 0 {
		return 0, 0, false
	}
	total := uint64(totalGB * 1024)
	avail := uint64((totalGB - usedGB) * 1024)
	return avail, total, true
}

// imageGenPressureNote 压力判定纯函数：可用内存 < 总量 15%（换出/冷载高发
// 区）或绝对值 < 4GB（连日常负载都紧张）即命中。返回面向用户的提示文案；
// 不命中或 totalMB=0（读数不可信）返回空串。
func imageGenPressureNote(availMB, totalMB uint64) string {
	if totalMB == 0 {
		return ""
	}
	lowRatio := availMB*100 < totalMB*15
	lowAbs := availMB < 4096
	if !lowRatio && !lowAbs {
		return ""
	}
	return fmt.Sprintf("系统可用内存偏低（%.1f GB / 共 %.1f GB）——本地模型可能需冷载，本次生成等待会明显变长",
		float64(availMB)/1024, float64(totalMB)/1024)
}

// noteImageGenMemoryPressure 压力预检入口：读系统内存→命中则 WARN+事件。
// 读不到内存（非 Windows/读失败）静默跳过——预检是增益信息，不是硬依赖。
func (c *core) noteImageGenMemoryPressure() {
	availMB, totalMB, ok := systemMemoryReader()
	if !ok {
		return
	}
	note := imageGenPressureNote(availMB, totalMB)
	if note == "" {
		return
	}
	slog.Warn("图像生成内存压力预检命中", "availMB", availMB, "totalMB", totalMB)
	c.emit("imagegen:pressure", map[string]interface{}{"note": note})
}
