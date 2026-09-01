// 会话/任务时间的相对显示：刚刚 / N 分钟前 / N 小时前 / 昨天 / M-D。
// 提供可注入的 now 便于确定性测试。
export function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

// formatElapsed 已用时段格式化（v4.26 工作态头部行）：<60s 显示「42s」，
// 否则「1m23s」。WorkHeader / task 卡 live 行共用同一口径。
export function formatElapsed(sec: number): string {
  if (sec < 60) return `${Math.max(0, Math.floor(sec))}s`;
  return `${Math.floor(sec / 60)}m${Math.floor(sec % 60)}s`;
}


export function relativeTime(ms: number, now = Date.now()): string {
  const diff = now - ms;
  const min = Math.floor(diff / 60_000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min} 分钟前`;
  const days = Math.round((startOfDay(new Date(now)) - startOfDay(new Date(ms))) / 86_400_000);
  if (days <= 0) {
    const h = Math.floor(min / 60);
    return h < 24 ? `${h} 小时前` : new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  if (days === 1) return "昨天";
  const d = new Date(ms);
  return `${d.getMonth() + 1}-${d.getDate()}`;
}
