import { useCallback, useEffect, useState } from "react";
import { app } from "../lib/bridge";
import type { WireDeniedAction } from "../lib/types";
import { truncateToWidth } from "../lib/textMetrics";

// DeniedActionsDock 待追认账本（codex 蒸馏 T3/T11 前端面）：拒绝/超时挂账的
// 动作列表——「无人值守回来后还有 N 条待审」的落点。追认 = 按原参数重放
// 审批卡（弹出常规 ApprovalModal，由人裁决）；guardian 拒绝的追认同走人工卡。
//
// 数据节奏：挂载时拉一次 + 每次 gaea 事件流里出现 turn_done / approval_auto
// / approval_resolved 时刷新（挂账发生在这些时点：guardian deny 落账、
// 用户拒绝落账、追认放行出账）。列表为空时零渲染。

export function DeniedActionsDock({ onReinstated }: { onReinstated?: () => void }) {
  const [actions, setActions] = useState<WireDeniedAction[]>([]);

  const refresh = useCallback(() => {
    app.PendingDeniedActions()
      .then((list) => setActions(list ?? []))
      .catch(() => {
        /* 拉取失败静默（会话未就绪/桥接抖动），下个事件时点重试 */
      });
  }, []);

  useEffect(() => {
    refresh();
    // 事件驱动刷新：回合结束（guardian 拒绝可能刚落账）、自动评审账本更新、
    // 审批卡解决（用户拒绝落账 / 追认放行出账）。
    const off = onGaeaEventKind((kind) => {
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
            {a.subject && <span className="office-denied-dock__subject">{clipSubject(a.subject)}</span>}
            {a.reason && <span className="office-denied-dock__reason" title={a.reason}>{clipReason(a.reason)}</span>}
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

function clipSubject(s: string): string {
  return truncateToWidth(s, 80);
}

function clipReason(s: string): string {
  return truncateToWidth(s, 60);
}

// onGaeaEventKind：gaea 事件流的 kind 过滤薄封装（组件内联，避免新公共 API）。
function onGaeaEventKind(cb: (kind: string) => void): () => void {
  // 动态 import 防循环依赖：bridge/events 已被 App 大量引用，本组件被 App
  // 引用——静态环会炸 dev 加载序。事件订阅 API 见 lib/bridge/events.ts。
  let off: (() => void) | null = null;
  let cancelled = false;
  void import("../lib/bridge")
    .then((m) => {
      if (cancelled) return;
      off = m.onEvent((e) => cb(e.kind));
    })
    // 测试环境 teardown 可打断在途动态 chunk（bridge→mock 全图），拒绝若不收编
    // 即 vitest unhandled error 噪音且会翻牌同 worker 用例；生产侧降级为仅告警
    // （订阅未建立= dock 收不到实时刷新，权限面只读不受影响）。
    .catch((err) => {
      if (!cancelled) console.warn("[DeniedActionsDock] bridge 动态加载失败，事件订阅未建立", err);
    });
  return () => {
    cancelled = true;
    off?.();
  };
}
