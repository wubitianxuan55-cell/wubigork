package agent

import (
	"fmt"
	"os"
)

// ─── 刀2: Live file observations（v4.375，Reasonix fileops/observation.go
// 蒸馏）────────────────────────────────────────────────────────────────────
//
// host 持有的「当前版本观察」：模型每次真实读到文件内容，host 记下该内容的
// 版本指纹；后续写该文件前先比对磁盘当前版本——不一致说明文件在模型观察之后
// 被外部（用户/其他进程）改过，模型持有的锚点/全文已过期，放行即静默覆盖
// 外部修改（数据丢失）。gaea 原有的 stale-anchor 守卫只覆盖「同轮已写须先读」，
// 检测不了跨轮外部修改；本观察是其超集，两者并存（anchor 规则管锚点新鲜度，
// 版本观察管外部篡改）。
//
// 边界（对齐上游语义）：
//   - 版本指纹只取元数据（size + mtime 纳秒），大文件零内容扫描。
//   - 观察从不序列化、从不进模型上下文——纯 host 侧事实。
//   - 从未观察过的路径不拦（模型没依赖过它的内容，无「过期」可言）。
//   - read_file 命中进程内缓存时不记录观察——缓存内容版本未知，记录当前
//     磁盘版本反而会掩盖过期。
//   - 自身写成功后刷新观察（后续编辑是站在自己刚写的内容上）。

// fileVersionFingerprint returns the current version fingerprint of path
// ("v1:<size>:<mtimeNano>") and true; ok=false when the file does not exist.
func fileVersionFingerprint(path string) (string, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("v1:%d:%d", st.Size(), st.ModTime().UnixNano()), true
}

// observeFile records the current version of path as what the model has seen.
// Caller must hold no assumptions about locking — uses turnMu (executeOne's
// goroutines and the run loop both touch the map).
func (a *AgentRunner) observeFile(path string) {
	if path == "" {
		return
	}
	fp, ok := fileVersionFingerprint(path)
	if !ok {
		return // 文件不存在：无可观察内容
	}
	a.turnMu.Lock()
	if a.fileObs == nil {
		a.fileObs = make(map[string]string)
	}
	a.fileObs[path] = fp
	a.turnMu.Unlock()
}

// checkFileStale reports a blocked-message when path's on-disk version differs
// from the version the model last observed. "" means proceed.
func (a *AgentRunner) checkFileStale(path string) string {
	if path == "" {
		return ""
	}
	a.turnMu.Lock()
	observed, has := a.fileObs[path]
	a.turnMu.Unlock()
	if !has {
		return "" // 从未观察：无过期依赖，放行
	}
	cur, exists := fileVersionFingerprint(path)
	if !exists {
		return fmt.Sprintf(
			"blocked: [stale version] %q was observed earlier but no longer exists on disk (deleted or moved externally). Re-check with ls/read_file before recreating it, or confirm the removal in your final answer.",
			path)
	}
	if cur != observed {
		return fmt.Sprintf(
			"blocked: [stale version] %q was modified outside this session since you last read it (observed %s, now %s). Your anchors/content are outdated — re-read it with read_file and redo the change against the fresh content.",
			path, observed, cur)
	}
	return ""
}
