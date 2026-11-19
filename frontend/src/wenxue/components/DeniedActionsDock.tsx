import { useCallback, useEffect, useState } from "react";
import { app } from "../lib/bridge";
import type { WireDeniedAction } from "../lib/types";
import { truncateToWidth } from "../../gaea/lib/textMetrics";

// 文学板块待追认账本（codex 蒸馏 T3/T11）：数据源和追认动作均映射到
// WenxueB，绝不误读办公 controller 的拒绝账本。
export function DeniedActionsDock({ onReinstated }: { onReinstated?: () => void }) {
  const [actions, setActions] = useState<WireDeniedAction[]>([]);

  const refresh = useCallback(() => {
    app.PendingDeniedActions()
      .then((list) => setActions(list ?? []))
      .catch(() => {
        /* 会话未就绪/桥接抖动：下一个事件时点重试 */
      });
  }, []);

  useEffect(() => {
    refresh();
    const off = onWenxueEventKind((kind) => {
      if (kind === "turn_done" || kind === "approval_auto" || kind === "approval_resolved") refresh();
    });
    return off;
  }, [refresh]);

  if (actions.length === 0) return null;

  const reinstate = (id: string) => {
    app.ApproveDenied(id)
      .then(() => {
        onReinstated?.();
        refresh();
      })
      .catch(() => refresh());
  };

  return (
    <div className="office-denied-dock" data-testid="denied-actions-dock" aria-live="polite">
      <p className="office-denied-dock__title">
        有 {actions.length} 条被拒/超时动作待追认（追认将重放审批卡，由你裁决）
      </p>
      <ul className="office-denied-dock__list">
        {actions.map((a) => (
          <li key={a.id} className="office-denied-dock__item" data-decision={a.decision} data-source={a.source}>
            <span className="office-denied-dock__tool">
              {a.decision === "timeout" ? "⏱ " : "✕ "}
              {a.tool}
            </span>
            {a.subject && <span className="office-denied-dock__subject">{clip(a.subject, 80)}</span>}
            {a.reason && <span className="office-denied-dock__reason" title={a.reason}>{clip(a.reason, 60)}</span>}
            <button
              type="button"
              className="office-denied-dock__reinstate"
              onClick={() => reinstate(a.id)}
            >
              追认
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function clip(text: string, max: number): string {
  return truncateToWidth(text, max);
}

function onWenxueEventKind(cb: (kind: string) => void): () => void {
  let off: (() => void) | null = null;
  let cancelled = false;
  void import("../lib/bridge")
    .then((m) => {
      if (cancelled) return;
      off = m.onEvent((e) => cb(e.kind));
    })
    // 同 gaea 副本：teardown 打断动态 chunk 的 rejection 必须收编（防 vitest
    // unhandled error 噪音翻牌同 worker 用例），生产侧降级为告警。
    .catch((err) => {
      if (!cancelled) console.warn("[DeniedActionsDock] bridge 动态加载失败，事件订阅未建立", err);
    });
  return () => {
    cancelled = true;
    off?.();
  };
}
