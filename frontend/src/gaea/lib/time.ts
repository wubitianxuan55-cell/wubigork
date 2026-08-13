// 会话/任务时间的相对显示：刚刚 / N 分钟前 / N 小时前 / 昨天 / M-D。
// 提供可注入的 now 便于确定性测试。
export function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
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
