package app

// 技能调用计数（阶段七 7.2-2 判据②：结晶技能调用 ≥5 次且成功率可见）。
// 纯累计与视图在 internal/skillstats（零 IO 表驱动）；本文件只做状态文件
// 持久化（<DataRoot>/skill_stats.json，route_suggestions 同款容错读+原子写）
// 与 boot 回调接线。诚实口径：成功率=工具级（read_skill 交付正文/run_skill
// 管线无错=ok），回合级成功率留观察池。
import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/skillstats"
)

// skillStatsMu 串行化读改写（read_skill/run_skill 可能并发于多回合）。
var skillStatsMu sync.Mutex

func skillStatsPath(dataRoot string) string {
	return filepath.Join(dataRoot, "skill_stats.json")
}

// loadSkillStats 读计数状态；文件缺失/损坏/version≠1 一律回空表
// （计数丢得起，不该为一个坏文件挡住技能通道）。
func loadSkillStats(dataRoot string) skillstats.File {
	b, err := os.ReadFile(skillStatsPath(dataRoot))
	if err != nil {
		return skillstats.File{}
	}
	var f skillstats.File
	if err := json.Unmarshal(b, &f); err != nil || f.Version != 1 {
		return skillstats.File{}
	}
	return f
}

// saveSkillStats 原子写（temp+rename，route_suggestions 同款手法）。
func saveSkillStats(dataRoot string, f skillstats.File) error {
	f.Version = 1
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	p := skillStatsPath(dataRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "skill-stats-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// recordSkillUse boot OnSkillUse 回调（引擎 goroutine 调用，无 App 状态依赖）：
// 累计一次调用并落盘。落盘失败只告警不阻断工具结果。
func recordSkillUse(name string, ok bool) {
	skillStatsMu.Lock()
	defer skillStatsMu.Unlock()
	root := config.DataRoot()
	f := skillstats.Record(loadSkillStats(root), name, ok, time.Now().Unix())
	if err := saveSkillStats(root, f); err != nil {
		slog.Warn("技能调用计数落盘失败", "skill", name, "error", err)
	}
}

// GaeaSkillStats 技能调用计数视图（能力面板技能行消费）：calls 降序，
// 损坏回空表恒非 nil。
func (a *App) GaeaSkillStats() []skillstats.StatView {
	skillStatsMu.Lock()
	defer skillStatsMu.Unlock()
	return skillstats.View(loadSkillStats(config.DataRoot()))
}
