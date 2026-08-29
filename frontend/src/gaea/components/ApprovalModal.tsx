import { useEffect, useRef } from "react";
import { useT } from "../lib/i18n";
import type { WireApproval } from "../lib/types";

const btnBase = "grid grid-cols-[28px_1fr] items-center gap-2.5 w-full min-h-[46px] rounded-lg text-fg p-1.5 px-2 text-left cursor-pointer transition-all duration-[var(--dur-fast)]";

function PlanBtn({ num, active, title, hint, onClick }: {
  num: number; active?: boolean; title: string; hint: string; onClick: () => void;
}) {
  return (
    <button
      className={`${btnBase} border ${
        active
          ? "border-[color-mix(in_srgb,var(--accent)_55%,var(--border))] bg-accent-soft hover:border-accent hover:scale-[1.01]"
          : "border-border-soft bg-bg-soft hover:border-fg-faint hover:bg-bg-elev-2 hover:scale-[1.01]"
      }`}
      onClick={onClick}
    >
      <span className={`inline-flex items-center justify-center w-[26px] h-[26px] border rounded-lg font-mono text-xs font-bold ${
        active ? "border-accent bg-accent text-accent-fg" : "border-border text-fg-dim bg-bg"
      }`}>
        {num}
      </span>
      <span className="flex min-w-0 flex-col gap-px">
        <span className="text-fg text-[13px] font-semibold leading-[1.25]">{title}</span>
        <span className="text-fg-faint text-[11.5px] leading-[1.3]">{hint}</span>
      </span>
    </button>
  );
}

// 审批决策（C4，对齐 control.ApproveDecision 与 codex ReviewDecision 子集）：
// deny=拒绝但继续 / allow_once=允许一次 / allow_session=本会话内允许 /
// persist_allow=始终允许（回写策略文件）/ abort=拒绝并终止本轮。
export type ApprovalDecision = "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort";

// 快捷键固定映射（与按钮序号一致；hardAsk 隐藏的按钮对应按键不响应）。
const KEY_DECISIONS: Record<string, ApprovalDecision> = {
  "1": "deny",
  "2": "allow_once",
  "3": "allow_session",
  "4": "persist_allow",
  "5": "abort",
};

export function ApprovalModal({
  approval,
  onAnswer,
}: {
  approval: WireApproval;
  onAnswer: (decision: ApprovalDecision) => void;
  onRevisePlan?: (text: string) => void;
}) {
  const t = useT();
  const cardRef = useRef<HTMLDivElement | null>(null);
  // 持久化写入（成本库/记忆/知识库）必须逐条确认：
  // 不提供「本会话内允许」「始终允许」，批准仅本次生效。
  const HARD_ASK_TOOLS = ["cost_save", "remember", "knowledge_add", "promote_session_facts"];
  const isHardAsk = HARD_ASK_TOOLS.includes(approval.tool);

  useEffect(() => { cardRef.current?.focus(); }, [approval.id]);

  useEffect(() => {
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.isContentEditable) return;
      if (event.key === "Escape") {
        event.preventDefault();
        onAnswer("deny");
        return;
      }
      const decision = KEY_DECISIONS[event.key];
      if (!decision) return;
      if (isHardAsk && (decision === "allow_session" || decision === "persist_allow")) return;
      event.preventDefault();
      onAnswer(decision);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onAnswer, isHardAsk]);

  return (
    <div className="plan-approval-dock" aria-live="polite">
      <div
        ref={cardRef}
        className="border border-border rounded-xl bg-bg-elev p-4 outline-none focus-visible:border-accent transition-shadow duration-200"
        style={{boxShadow: "var(--ds-shadow-card)"}}
        onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.boxShadow = "var(--ds-shadow-card-hover)"}
        onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.boxShadow = "var(--ds-shadow-card)"}
        role="dialog" aria-modal="false" tabIndex={-1}
        aria-labelledby="tool-approval-title"
      >
        <div className="mb-3">
          <div id="tool-approval-title" className="text-fg font-semibold leading-[1.35] text-[14px]">
            {t("approval.toolTitle")}
          </div>
          <div className="text-fg-dim text-[12.5px] leading-[1.45] mt-0.5">
            {t("approval.toolNote")}
          </div>
        </div>

        <div className="flex items-center gap-2 min-h-[34px] mb-3 border border-border-soft rounded-lg bg-bg-soft py-[5px] px-2">
          <span className="shrink-0 text-fg-faint font-mono text-[11px] uppercase tracking-[0.04em]">{t("approval.toolLabel")}</span>
          <span className="font-mono text-xs font-medium text-fg">{approval.tool}</span>
        </div>

        {approval.subject && (
          <pre className="m-0 mb-3 px-[11px] py-[9px] bg-bg-soft border border-border-soft rounded-lg font-mono text-[12.5px] whitespace-pre-wrap break-words max-h-[140px] overflow-auto">
            {approval.subject}
          </pre>
        )}

        <div className="flex flex-col gap-1.5">
          <PlanBtn num={1} title={t("approval.deny")} hint={t("approval.denyHint")} onClick={() => onAnswer("deny")} />
          <PlanBtn num={2} active title={t("approval.allowOnce")} hint={t("approval.allowOnceHint")} onClick={() => onAnswer("allow_once")} />
          {!isHardAsk && (
            <PlanBtn num={3} title={t("approval.allowSession")} hint={t("approval.allowSessionHint")} onClick={() => onAnswer("allow_session")} />
          )}
          {!isHardAsk && (
            <PlanBtn num={4} title={t("approval.persistAlways")} hint={t("approval.persistAlwaysHint")} onClick={() => onAnswer("persist_allow")} />
          )}
          <PlanBtn num={isHardAsk ? 3 : 5} title={t("approval.abort")} hint={t("approval.abortHint")} onClick={() => onAnswer("abort")} />
        </div>
        {isHardAsk && (
          <div className="mt-2 text-[11px] text-fg-faint leading-snug">
            持久化写入需逐条确认：批准仅本次生效，不会自动放行后续写入。
          </div>
        )}
      </div>
    </div>
  );
}
