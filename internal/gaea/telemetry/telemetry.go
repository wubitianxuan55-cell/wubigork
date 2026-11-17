// Package telemetry 是本地观测最小集（codex 蒸馏 B10 折算，第四批 P7）。
//
// 机制同构 codex analytics 事实流的「环形缓冲 + 去重上限 + 落盘」，不引
// otel（归档决策：本地事实流已能回答关键问题，otel 全套是净负债）：
//   - 环形缓冲 256 条事实（内存查询面，Snapshot/失败率/p95 的数据源）；
//   - 去重计数表上限 4096 条目（同名同维度事实只累计计数，不重复进环）；
//   - 落盘 whisperDataRoot/telemetry/telemetry-YYYYMMDD.ndjson，按日滚动，
//     启动清扫 >14 天旧文件（拍板 2026-11-17：14 天保留期）；
//   - consent：[telemetry] enabled 默认 false——关闭时 Record 是单布尔
//     短路（零成本），不写盘不进环。
//
// 事实模型刻意最小：name + dims（低基数维度）+ value（毫秒/计数）+ ts。
// 维度值超长截断、条目数超环即滚动——观测数据自己不能变成故障源。
package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 上限常量（上游生产值同源：ring 256 / 去重 4096）。
const (
	RingCapacity   = 256
	DedupCapacity  = 4096
	RetentionDays  = 14
	dimValueMax    = 64
	factNameMax    = 64
	dayFileLayout  = "20060102"
	exportKindFact = "fact"
)

// Fact 是一条观测事实。
type Fact struct {
	Name string            `json:"name"`
	Dims map[string]string `json:"dims,omitempty"`
	Val  float64           `json:"val,omitempty"`
	Ts   int64             `json:"ts"`
	// N 是去重合并的次数（同名同维度第 2 次起不再进环，只累加 N 与 Val）。
	N int `json:"n,omitempty"`
}

// Recorder 是环形+去重+落盘的观测记录器。零值可用（关闭态：Record 短路），
// Enable 后开启。并发安全。
type Recorder struct {
	mu      sync.Mutex
	enabled bool
	dir     string
	day     string
	f       *os.File
	w       *bufio.Writer
	ring    []Fact
	dedup   map[string]int // fact key → ring 下标；超容量后仅计数丢弃
	dropped int            // 去重表满后被丢弃的合并次数（自观测）
	count   int64
}

// defaultRecorder 是进程级单例（agent/control 低摩擦记账面；默认关闭）。
var defaultRecorder = &Recorder{}

// Default 返回进程级记录器。
func Default() *Recorder { return defaultRecorder }

// Enabled 报告开关态。
func (r *Recorder) Enabled() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled
}

// Enable 开启观测并绑定落盘目录（立即清扫过期文件；目录创建失败仍开启
// 内存面——观测不能因为落盘问题反过来阻断宿主）。
func (r *Recorder) Enable(dir string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = true
	r.dir = dir
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			sweepExpiredLocked(dir)
		}
	}
}

// Disable 关闭（内存面保留可查，落盘停止）。
func (r *Recorder) Disable() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
	r.closeFileLocked()
}

// Record 记一条事实。关闭态零成本；同名同维度第二次起合并计数（N/Val
// 累加）不再占环位；去重表满后丢弃只计 dropped。
func (r *Recorder) Record(name string, dims map[string]string, val float64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.enabled {
		return
	}
	if r.dedup == nil {
		r.dedup = make(map[string]int, 64)
	}
	now := time.Now()
	f := Fact{Name: truncate(name, factNameMax), Dims: normalizeDims(dims), Val: val, Ts: now.UnixMilli()}
	key := factKey(f)
	if idx, ok := r.dedup[key]; ok && idx >= 0 && idx < len(r.ring) {
		hit := &r.ring[idx]
		if hit.Name == f.Name && factKey(*hit) == key { // 下标可能被环滚动挤掉，双保险
			hit.N++
			hit.Val += val
			hit.Ts = f.Ts
			r.writeLocked(*hit)
			return
		}
	}
	if len(r.dedup) >= DedupCapacity {
		r.dropped++
		return
	}
	if len(r.ring) >= RingCapacity {
		// 滚动：挤掉最旧的一条，其去重下标作废（下次同名事实重新进环）。
		oldest := r.ring[0]
		if idx, ok := r.dedup[factKey(oldest)]; ok && idx == 0 {
			delete(r.dedup, factKey(oldest))
		}
		r.ring = r.ring[1:]
		fixupDedupIndexes(r.dedup)
	}
	r.ring = append(r.ring, f)
	r.dedup[key] = len(r.ring) - 1
	r.count++
	r.writeLocked(f)
}

// Snapshot 返回环形缓冲副本（时间正序）。
func (r *Recorder) Snapshot() []Fact {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Fact, len(r.ring))
	copy(out, r.ring)
	return out
}

// Stats 返回自观测计数（总记录/丢弃合并数/当前环长）。
func (r *Recorder) Stats() (recorded int64, dropped int, ringLen int) {
	if r == nil {
		return 0, 0, 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count, r.dropped, len(r.ring)
}

// Flush 把缓冲落盘收尾（进程退出路径用；关闭态 no-op）。
func (r *Recorder) Flush() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeFileLocked()
}

// writeLocked 把一条事实追加到当日 ndjson（跨日滚动文件；写失败静默——
// 观测尽力而为，绝不反向阻断宿主）。
func (r *Recorder) writeLocked(f Fact) {
	if r.dir == "" {
		return
	}
	day := time.UnixMilli(f.Ts).Format(dayFileLayout)
	if r.f == nil || day != r.day {
		r.closeFileLocked()
		path := filepath.Join(r.dir, "telemetry-"+day+".ndjson")
		fh, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return // 本日写不进就只留内存面
		}
		r.f, r.w, r.day = fh, bufio.NewWriter(fh), day
	}
	b, err := json.Marshal(map[string]any{"kind": exportKindFact, "fact": f})
	if err != nil {
		return
	}
	if _, err := r.w.Write(append(b, '\n')); err != nil {
		return
	}
	// 低频事实：每条即 flush（进程被强杀也不丢当日已记事实；无 fsync——
	// 观测数据不值得为持久性付出 fsync 成本）。
	_ = r.w.Flush()
}

func (r *Recorder) closeFileLocked() {
	if r.w != nil {
		_ = r.w.Flush()
		r.w = nil
	}
	if r.f != nil {
		_ = r.f.Close()
		r.f = nil
	}
}

// sweepExpiredLocked 清扫超过保留期的当日文件（启动时一次）。
func sweepExpiredLocked(dir string) {
	cutoff := time.Now().AddDate(0, 0, -RetentionDays).Format(dayFileLayout)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "telemetry-") || !strings.HasSuffix(name, ".ndjson") {
			continue
		}
		day := strings.TrimSuffix(strings.TrimPrefix(name, "telemetry-"), ".ndjson")
		if len(day) == len(dayFileLayout) && day < cutoff {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// factKey 是去重键：name + 排序后的维度对。
func factKey(f Fact) string {
	dims := make([]string, 0, len(f.Dims))
	for k, v := range f.Dims {
		dims = append(dims, k+"="+v)
	}
	// 小维度集：插入排序足够（维度数是个位数）。
	for i := 1; i < len(dims); i++ {
		for j := i; j > 0 && dims[j] < dims[j-1]; j-- {
			dims[j], dims[j-1] = dims[j-1], dims[j]
		}
	}
	return f.Name + "|" + strings.Join(dims, ",")
}

// normalizeDims 归一维度：小写键、值截断、nil → 空。
func normalizeDims(dims map[string]string) map[string]string {
	if len(dims) == 0 {
		return nil
	}
	out := make(map[string]string, len(dims))
	for k, v := range dims {
		out[truncate(strings.ToLower(k), 32)] = truncate(v, dimValueMax)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// fixupDedupIndexes 环滚动后整体下标减一。
func fixupDedupIndexes(dedup map[string]int) {
	for k, v := range dedup {
		if v > 0 {
			dedup[k] = v - 1
		} else {
			delete(dedup, k)
		}
	}
}

// String 给诊断日志一行摘要。
func (r *Recorder) String() string {
	if r == nil {
		return "telemetry(off)"
	}
	recorded, dropped, ringLen := r.Stats()
	return fmt.Sprintf("telemetry(enabled=%v recorded=%d dropped=%d ring=%d)", r.Enabled(), recorded, dropped, ringLen)
}
