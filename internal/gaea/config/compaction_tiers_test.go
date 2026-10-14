package config

// 阶段九 9.1「双阈值可配置」：三层压缩水位的配置暴露与钳制。

import "testing"

func TestCompactionTiers_Defaults(t *testing.T) {
	// 零值（未配置）= 内置缺省 0.5/0.8/0.9（Reasonix/dsh 蒸馏实证值）
	soft, ratio, force := AgentConfig{}.CompactionTiers()
	if soft != 0.5 || ratio != 0.8 || force != 0.9 {
		t.Fatalf("缺省档: %v %v %v", soft, ratio, force)
	}
}

func TestCompactionTiers_Configured(t *testing.T) {
	// 调研建议的 75%/88%（规划口径）经配置达成
	soft, ratio, force := AgentConfig{CompactionSoftRatio: 0.6, CompactionRatio: 0.75, CompactionForceRatio: 0.88}.CompactionTiers()
	if soft != 0.6 || ratio != 0.75 || force != 0.88 {
		t.Fatalf("配置档: %v %v %v", soft, ratio, force)
	}
}

func TestCompactionTiers_InvalidFallsBackAsGroup(t *testing.T) {
	cases := []AgentConfig{
		{CompactionRatio: 1.5}, // 越界
		{CompactionSoftRatio: 0.9, CompactionRatio: 0.8, CompactionForceRatio: 0.7},    // 顺序破坏
		{CompactionRatio: 0.8, CompactionForceRatio: 0.8},                              // 等值（水位重叠）
		{CompactionSoftRatio: -0.1, CompactionRatio: 0.75, CompactionForceRatio: 0.88}, // 负值
	}
	for i, c := range cases {
		soft, ratio, force := c.CompactionTiers()
		if soft != 0.5 || ratio != 0.8 || force != 0.9 {
			t.Fatalf("case %d 非法应整组回落缺省: %v %v %v", i, soft, ratio, force)
		}
	}
}
