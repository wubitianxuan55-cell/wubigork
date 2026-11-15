// 会话/任务时间的相对显示：刚刚 / N 分钟前 / N 小时前 / 昨天 / M-D。
// 提供可注入的 now 便于确定性测试。
// FE4-03：文案唯一源——此前本函数与 utils/time.formatRelativeTime 两套档位
// （后者多「N 天前/N 周前」档）并存且各自硬编码中文；现统一为本档位并走 i18n
// （非响应式 t，见 lib/i18n.tsx：React 外可调用，未加载语言回退 zh 兜底）。
import { t } from "./i18n";

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
  if (min < 1) return t("time.justNow");
  if (min < 60) return t("time.minutesAgo", { n: min });
  const days = Math.round((startOfDay(new Date(now)) - startOfDay(new Date(ms))) / 86_400_000);
  if (days <= 0) {
    const h = Math.floor(min / 60);
    return h < 24 ? t("time.hoursAgo", { n: h }) : new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  if (days === 1) return t("time.yesterday");
  const d = new Date(ms);
  return `${d.getMonth() + 1}-${d.getDate()}`;
}
