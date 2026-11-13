// OfficeLiveBadge — COM 实时编辑状态胶囊（波3「壳内迷你卡替代」展示位）。
//
// 数据源 lib/officeLiveStore（hooks/useOfficeLive 喂入）。两个渲染变体：
//   - 本文件变体（relPath 与 activeDoc 归一化相同）：「此文件正在被 AI 实时
//     编辑」——FilePreview 正打开的文件就是 AI 手里的那份，进度一目了然；
//   - 他文件变体：预览着 A 文件、AI 正改 B 文件——胶囊显示 B 的文件名 +
//     进度，点击经 openPaneFileOrPreview（U2 置前同入口）打开 B 的预览。
//
// 新鲜度：activeDoc 存在且 updatedAt 在 5 分钟内才渲染（过期即消失）；秒级
// 复查走全局共享时钟 useNow（1s 单例 interval，订阅者归零自动停摆），无本
// 组件私有的定时器。文案 zh 内联（FilePreview 头部既有 zh-only 惯例）。

import { useEffect } from "react";
import { isOfficeLiveFresh, useOfficeLiveStore } from "../lib/officeLiveStore";
import { normalizePreviewPath } from "../lib/officeTurnProjection";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { useNow } from "../lib/useNow";

/** 应用中文名（胶囊 title 用；word/excel/ppt 与 Go 工具枚举一致）。 */
const APP_LABEL: Record<string, string> = { word: "Word", excel: "Excel", ppt: "演示文稿" };

export function OfficeLiveBadge({ relPath }: { relPath?: string | null }) {
  const activeDoc = useOfficeLiveStore((s) => s.activeDoc);
  const activeApp = useOfficeLiveStore((s) => s.activeApp);
  const step = useOfficeLiveStore((s) => s.step);
  const updatedAt = useOfficeLiveStore((s) => s.updatedAt);
  // 秒级时钟：只有挂着活动文档时才订阅（无活动时组件本就不渲染，时钟停摆）。
  const nowSec = useNow(activeDoc !== null);
  const now = nowSec * 1000;

  // 惰性过期：越过 TTL 的活动文档就地清空 store（无定时器的收尾），清完
  // activeDoc=null → 时钟退订，组件消失。
  useEffect(() => {
    useOfficeLiveStore.getState().expireIfStale(now);
  }, [now]);

  if (!isOfficeLiveFresh({ activeDoc, activeApp, step, updatedAt }, now)) return null;
  const doc = activeDoc as string;

  // 归一化比较（normalizePreviewPath：反斜杠→斜杠 + 小写，与刷新总线同键）。
  const thisFile = relPath ? normalizePreviewPath(relPath) === normalizePreviewPath(doc) : false;
  const fileName = doc.split(/[\\/]/).pop() || doc;
  const appText = activeApp ? (APP_LABEL[activeApp] ?? activeApp) : "";
  const stepText = step ? `编辑中 · 第 ${step.done}/${step.total} 步` : "实时编辑中";

  return (
    <button
      type="button"
      data-testid="office-live-badge"
      onClick={() => openPaneFileOrPreview(doc)}
      title={`${appText ? `AI 正在 ${appText} 窗口中` : "AI 正在"}实时编辑：${doc}\n点击打开预览`}
      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border border-accent/30 bg-accent/10 text-accent text-[10px] shrink-0 cursor-pointer hover:bg-accent/20"
    >
      <span aria-hidden className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
      <span className="max-w-[160px] truncate">{thisFile ? "此文件正在被 AI 实时编辑" : fileName}</span>
      <span className="opacity-80 shrink-0">{stepText}</span>
    </button>
  );
}
