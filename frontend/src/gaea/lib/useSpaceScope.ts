// useSpaceScope.ts — S1.2-C 检索 scope 的「当前空间」来源
// （docs/gaea-memory-isolation-design.md C 步：复用 SpaceChip 的 GaeaSpaceActive）。
//
// 模块级缓存 + 订阅：SpaceChip 激活新空间后经 noteSpaceActivated 广播，
// 已挂载的检索面（keepAlive 的记忆中枢页 / 搜索面板）随之刷新默认 scope，
// 避免一次性 fetch 在空间切换后拿到陈旧的「当前空间」。
// 绑定不可用（旧后端）时 active 恒为 null，调用方按 work 兜底（mode=off 时
// SpaceActiveView 本就恒报 work，不违反双空间红线）。
import { useEffect, useState } from "react";
import { app } from "./bridge";
import type { SearchScope, SpaceActiveView } from "./types";

let cached: SpaceActiveView | null = null;
let inflight: Promise<void> | null = null;
const listeners = new Set<(v: SpaceActiveView) => void>();

/** SpaceChip 激活成功后调用：更新缓存并广播给所有检索面。 */
export function noteSpaceActivated(v: SpaceActiveView): void {
  cached = v;
  inflight = null;
  for (const l of listeners) l(v);
}

/** 拉取当前生效空间（模块级去重：hub 页/搜索面板/后续调用者共享一次 RPC）。 */
function ensureActive(): void {
  if (cached || inflight) return;
  inflight = app
    .GaeaSpaceActive()
    .then((v) => {
      noteSpaceActivated(v);
    })
    .catch(() => {
      // 绑定不可用（旧后端/异常）：保持 null，调用方按 work 兜底；下次挂载重试。
      inflight = null;
    });
}

/** 检索面用：返回当前生效空间视图（未解析到时为 null）。 */
export function useSpaceScope(): { active: SpaceActiveView | null } {
  const [active, setActive] = useState<SpaceActiveView | null>(cached);
  useEffect(() => {
    let cancelled = false;
    const push = (v: SpaceActiveView) => {
      if (!cancelled) setActive(v);
    };
    listeners.add(push);
    if (cached) push(cached);
    else ensureActive();
    return () => {
      cancelled = true;
      listeners.delete(push);
    };
  }, []);
  return { active };
}

// scope 切换的三档选项（工位/乐园/全部）：默认选中项=当前生效空间，
// 「全部」=scope ""（旧行为，仅显式选择时使用）。
export const SCOPE_OPTIONS: { value: SearchScope; label: string; title: string }[] = [
  { value: "work", label: "工位", title: "工位（办公空间）——只搜工位记忆与资料" },
  { value: "play", label: "乐园", title: "乐园（娱乐空间）——只搜乐园记忆" },
  { value: "", label: "全部", title: "全部——跨工位与乐园检索（显式选择，默认不跨空间）" },
];
