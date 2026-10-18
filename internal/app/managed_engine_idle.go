package app

// managed_engine_idle.go — 外部引擎空闲自动退出（model_idle_exit_minutes）。
//
// 天璇/Strata 是 gaea 代启的本地模型进程（吃 UMA/VRAM 的大户）：连续 N 分钟
// 无真实使用即自动停止，不占机器。判定依据是 modelengine 的活跃记账
//（TouchEngine：LLM 请求端点解析 / 引擎启动 / 识图绑定解析）；探活刻意不
// 记账。只收 gaea 本会话拉起的实例（proc registered）——外部手动启动的引擎
// 尊重其独立性，不代管。gaea 自身退出另有 Job Object 兜底（kill-on-close），
// 两者合起来覆盖「运行中空转」与「gaea 退了模型还挂着」两种占用。

import (
	"context"
	"log/slog"
	"time"
)

// managedIdleCheckInterval 空闲巡检周期：1 分钟粒度足够（分钟级阈值）。
const managedIdleCheckInterval = time.Minute

// managedIdleTarget 一个可收割目标（生产指向真实探活/停止；测试注入桩）。
type managedIdleTarget struct {
	id    string
	proc  *localEngineProc
	probe func() bool // true=引擎在运行
	stop  func()      // 幂等停止（复用引擎卡的停止语义）
}

// startManagedEngineIdleReaper 启动空闲巡检循环（ctx 随 app 生命周期）。
func (a *App) startManagedEngineIdleReaper(ctx context.Context) {
	if a.core == nil || a.cfg == nil || a.engineMgr == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(managedIdleCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.reapIdleManagedEngines()
			}
		}
	}()
}

// managedIdleExitMinutes 当前生效阈值（分钟；<=0=机制关闭）。
func (a *App) managedIdleExitMinutes() int {
	if a.cfg == nil {
		return 0
	}
	return a.cfg.ModelIdleExitMinutes
}

// reapIdleManagedEngines 单轮巡检：逐引擎判「gaea 拉起 + 有使用记录 + 空闲
// 超阈值 + 确在运行」→ 停止并广播。阈值关闭（<=0）直接返回。
func (a *App) reapIdleManagedEngines() {
	minutes := a.managedIdleExitMinutes()
	if minutes <= 0 {
		return
	}
	a.reapManagedEngineIdleOnce(time.Now(), time.Duration(minutes)*time.Minute, a.managedIdleTargets())
}

// managedIdleTargets 生产目标表（天璇 8094 / Strata 8091）。
func (a *App) managedIdleTargets() []managedIdleTarget {
	return []managedIdleTarget{
		{
			id:   "tianxuan",
			proc: &a.tianxuanProc,
			probe: func() bool {
				_, ok := tianxuanProbe()
				return ok
			},
			stop: func() { _ = a.TianxuanEngineStop() },
		},
		{
			id:   "strata",
			proc: &a.strataProc,
			probe: func() bool {
				_, ok := strataProbe()
				return ok
			},
			stop: func() { _ = a.StrataEngineStop() },
		},
	}
}

// reapManagedEngineIdleOnce 单轮判定（now/idle/targets 注入便于测试）：
// 返回本轮被收割的引擎 id。四道闸逐个过：
//  1. gaea 本会话拉起（proc registered）——外部实例不代管；
//  2. 有使用记录（lastActivity 非零——零值说明本会话从未经手，别碰）；
//  3. 空闲超阈值；
//  4. 探活确认在运行（不在运行无需动作；探活失败也不算「可停」——可能是
//     冷启动中，下一轮再看）。
func (a *App) reapManagedEngineIdleOnce(now time.Time, idle time.Duration, targets []managedIdleTarget) []string {
	var stopped []string
	for _, t := range targets {
		if !t.proc.registered() {
			continue
		}
		last := a.engineMgr.EngineLastActivity(t.id)
		if last.IsZero() {
			continue
		}
		if idleUsed := now.Sub(last); idleUsed < idle {
			continue
		}
		if !t.probe() {
			continue
		}
		slog.Info("外部引擎空闲超阈值，自动退出", "engine", t.id,
			"idle_minutes", now.Sub(last).Round(time.Second).String(),
			"threshold", idle.String())
		t.stop()
		stopped = append(stopped, t.id)
		a.emit("managed-engine-idle-stopped", map[string]interface{}{
			"engine": t.id, "idle_minutes": int(idle.Minutes()),
		})
	}
	return stopped
}
