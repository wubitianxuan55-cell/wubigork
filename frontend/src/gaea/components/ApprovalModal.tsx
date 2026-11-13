import { useEffect, useId, useRef } from "react";
import { Shield } from "../icons";
import { useT } from "../lib/i18n";
import type { WireApproval } from "../lib/types";
import { ScheduleDiffCard } from "../../schedule/ScheduleDiffCard";
import { scheduleApplyIsProjectChannel } from "../../schedule/applyDiff";

function PlanBtn({ num, tone = "default", title, hint, onClick, decision }: {
  num: number;
  tone?: "default" | "primary" | "danger";
  title: string;
  hint: string;
  onClick: () => void;
  decision: ApprovalDecision;
}) {
  return (
    <button
      type="button"
      className="office-decision-option"
      data-tone={tone}
      data-decision={decision}
      onClick={onClick}
      aria-keyshortcuts={String(num)}
    >
      <span className="office-decision-key" aria-hidden>{num}</span>
      <span className="office-decision-option__copy">
        <span className="office-decision-option__title">{title}</span>
        <span className="office-decision-option__hint">{hint}</span>
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
  scheduleApplyArgs,
}: {
  approval: WireApproval;
  onAnswer: (decision: ApprovalDecision) => void;
  onRevisePlan?: (text: string) => void;
  /** schedule_apply 专用：宿主从 Transcript 取该次调用的完整 args（弹卡期间
   *  工具卡 status=running 已携带，§1.3）→ diff 确认卡（刀B）事前预览用。 */
  scheduleApplyArgs?: string | null;
}) {
  const t = useT();
  const idPrefix = useId();
  const titleId = `${idPrefix}-title`;
  const noteId = `${idPrefix}-note`;
  const cardRef = useRef<HTMLDivElement | null>(null);
  // 持久化写入（成本库/记忆/知识库）必须逐条确认：
  // 不提供「本会话内允许」「始终允许」，批准仅本次生效。
  const HARD_ASK_TOOLS = ["cost_save", "remember", "knowledge_add", "promote_session_facts"];
  const isHardAsk = HARD_ASK_TOOLS.includes(approval.tool);
  // diff 确认闭环刀C：schedule_apply project 整量通道（args.project 非空）
  // 同样按三钮形态渲染（禁会话记忆，拍板项 2）——Go 闸门 alwaysPrompt 是
  // 强制权威，这里只是让按钮形态与后端语义一致；ops 增量通道维持五钮。
  const isProjectHard =
    approval.tool === "schedule_apply" && scheduleApplyIsProjectChannel(scheduleApplyArgs ?? null);
  const hardConfirm = isHardAsk || isProjectHard;

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
      if (hardConfirm && (decision === "allow_session" || decision === "persist_allow")) return;
      event.preventDefault();
      onAnswer(decision);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onAnswer, hardConfirm]);

  return (
    <div className="office-approval-dock" aria-live="polite">
      <section
        ref={cardRef}
        className="office-approval-card"
        role="dialog" aria-modal="true" tabIndex={-1}
        aria-labelledby={titleId}
        aria-describedby={noteId}
        data-permission-request={approval.request ? "true" : "false"}
        data-hard-confirm={hardConfirm ? "true" : "false"}
      >
        <header className="office-approval-head">
          <div className="office-decision-head">
            <span className="office-decision-head__icon" aria-hidden>
              <Shield size={14} />
            </span>
            <span className="office-decision-head__copy">
              <span className="office-decision-head__kicker">{t("approval.decisionLabel")}</span>
              <h2 id={titleId} className="office-decision-head__title">
                {approval.request ? t("approval.requestTitle") : t("approval.toolTitle")}
              </h2>
              <p id={noteId} className="office-decision-head__note">
                {approval.request ? t("approval.requestNote") : t("approval.toolNote")}
              </p>
            </span>
          </div>
        </header>

        <div className="office-approval-body">
        <div className="office-decision-context">
          <span className="office-decision-context__label">
            {approval.request ? t("approval.ruleLabel") : t("approval.toolLabel")}
          </span>
          <span className="office-decision-context__value">
            {approval.request && approval.subject ? `${approval.tool}(${approval.subject})` : approval.tool}
          </span>
        </div>

        {approval.request && (
          <div className="office-approval-request">
            <div className="office-approval-request__label">
              {t("approval.reasonLabel")}
            </div>
            {/* 模型给出的申请理由原文：必须可见后才谈得上采信，缺省时明确标注 */}
            <div className="office-approval-request__text">
              {approval.reason?.trim() ? approval.reason : t("approval.reasonMissing")}
            </div>
          </div>
        )}

        {approval.tool === "schedule_apply" ? (
          /* diff 确认闭环刀B：schedule_apply 卡体升级为结构化 diff 预览
             （决策钮/快捷键/超时语义不变，只换卡体内容）。 */
          <ScheduleDiffCard args={scheduleApplyArgs ?? null} />
        ) : (
          !approval.request && approval.subject && (
            <pre className="office-approval-pre">
              {approval.subject}
            </pre>
          )
        )}
        </div>

        <footer className="office-approval-actions">
          <div className="office-approval-actions__primary">
            <PlanBtn num={1} decision="deny" title={t("approval.deny")} hint={t("approval.denyHint")} onClick={() => onAnswer("deny")} />
            <PlanBtn num={2} decision="allow_once" tone="primary" title={t("approval.allowOnce")} hint={t("approval.allowOnceHint")} onClick={() => onAnswer("allow_once")} />
          </div>
          <div className="office-approval-actions__secondary">
            {!hardConfirm && (
              <PlanBtn num={3} decision="allow_session" title={t("approval.allowSession")} hint={t("approval.allowSessionHint")} onClick={() => onAnswer("allow_session")} />
            )}
            {!hardConfirm && (
              <PlanBtn num={4} decision="persist_allow" title={t("approval.persistAlways")} hint={t("approval.persistAlwaysHint")} onClick={() => onAnswer("persist_allow")} />
            )}
            <PlanBtn num={hardConfirm ? 3 : 5} decision="abort" tone="danger" title={t("approval.abort")} hint={t("approval.abortHint")} onClick={() => onAnswer("abort")} />
          </div>
          {isProjectHard && (
            <p className="office-approval-note">
              整计划替换需逐条确认：批准仅本次生效，不会自动放行后续写入；拒绝后可让模型按你的意见修改后重新提交。
            </p>
          )}
          {isHardAsk && (
            <p className="office-approval-note">
              持久化写入需逐条确认：批准仅本次生效，不会自动放行后续写入。
            </p>
          )}
        </footer>
      </section>
    </div>
  );
}
