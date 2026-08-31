package app

// ── 今日晨报（做梦 2.0 主动预取 MVP：纯本地晨报）──────────────────────
// GaeaMemoryMorningBrief 返回「今日晨报」JSON 串（前端 JSON.parse 后渲染，
// 对齐 GaeaCostGraph 的 JSON 串绑定先例——避免绑定层结构映射开销）。
//
// 读取面（零副作用）：
//   - 办公记忆：hubOfficeStore().ListInSpace("work") 只读当前空间活跃事实
//     （双空间红线：晨报只呈现 work 记忆，play 不渲染）；
//   - 近 24h dream 沉淀计数：DreamAuditEntries(userDir, 大N) 过滤
//     TS≥now-24h 且 Source∈{auto_dream,distill_merge} 的 Saved 计数；
//     审计文件缺失/损坏/解析失败一律降级 0，不阻断晨报生成。
//
// 纯函数 BuildMorningBrief（memory 包）做排序/截断/组装；本绑定只负责
// 取数 + 计数 + 序列化。零 LLM、零写库、零落审计。

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// morningAuditWindow 是「近 24h」的统计时窗。
const morningAuditWindow = 24 * time.Hour

// morningAuditMax 是 DreamAuditEntries 的读取上限（大 N）：24h 内的 dream
// 写入远低于此值，且函数按最近优先返回，足够覆盖时窗内的全部审计行。
const morningAuditMax = 5000

// morningAuditUserDir 返回 dream 写入审计所在用户目录（GAEA_DATA_ROOT
// 优先，测试隔离用；生产等同 config.MemoryUserDir——对齐 gaea_dream.go
// 的 dreamAuditPath(userDir) 用法，审计文件位于 <userDir>/dream-audit.jsonl）。
func morningAuditUserDir() string {
	if d := os.Getenv("GAEA_DATA_ROOT"); d != "" {
		return d
	}
	return config.MemoryUserDir()
}

// countDreamed24h 统计近 24h 的 dream 沉淀条数：过滤 TS≥now-24h 且
// Source∈{auto_dream,distill_merge} 的审计行，累加 Saved（行 TS 解析失败
// 跳过；整表缺失返回 0——统计失败降级不阻断）。
func countDreamed24h(entries []control.DreamAuditEntry, now time.Time) int {
	cutoff := now.Add(-morningAuditWindow).UTC()
	n := 0
	for _, e := range entries {
		if e.Source != "auto_dream" && e.Source != "distill_merge" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, e.TS)
		if err != nil {
			continue
		}
		if ts.Before(cutoff) {
			continue
		}
		n += e.Saved
	}
	return n
}

// GaeaMemoryMorningBrief 生成今日晨报 JSON 串。无记忆/无审计均正常返回
// 空结构（items/rules 为非 nil 空数组），永不报错阻断。
func (a *App) GaeaMemoryMorningBrief() (string, error) {
	now := time.Now()
	store := a.hubOfficeStore()
	b := memory.BuildMorningBrief(
		store.ListInSpace("work"),
		countDreamed24h(control.DreamAuditEntries(morningAuditUserDir(), morningAuditMax), now),
		now,
	)
	raw, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
