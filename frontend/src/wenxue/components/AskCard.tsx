import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { Check, MessageSquare } from "../icons";
import { useT } from "../lib/i18n";
import type { QuestionAnswer, WireAsk, WireAskQuestion } from "../lib/types";
import { Markdown } from "./Markdown";

/** 卡片至少保留在屏幕内的边距 (px) */
const DRAG_MARGIN = 40;

export function AskCard({
  ask,
  onAnswer,
  onDismiss,
}: {
  ask: WireAsk;
  onAnswer: (id: string, answers: QuestionAnswer[]) => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const idPrefix = useId();
  const titleId = `${idPrefix}-title`;
  const [sel, setSel] = useState<Record<string, string[]>>({});
  const [custom, setCustom] = useState<Record<string, string>>({});

  // ── 拖拽：位置状态 ──────────────────────────────────────────────
  const cardRef = useRef<HTMLDivElement>(null);
  const cardSize = useRef({ w: 0, h: 0 });
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);
  const dragging = useRef(false);
  const dragStart = useRef({ x: 0, y: 0 });
  const posStart = useRef({ x: 0, y: 0 });

  /** 将坐标约束在可视区域内 */
  const clamp = useCallback((x: number, y: number) => {
    const { w, h } = cardSize.current;
    const M = DRAG_MARGIN;
    return {
      x: Math.min(window.innerWidth - M, Math.max(-w + M, x)),
      y: Math.min(window.innerHeight - M, Math.max(-h + M, y)),
    };
  }, []);

  // 初始居中
  useLayoutEffect(() => {
    const card = cardRef.current;
    if (!card) return;
    const r = card.getBoundingClientRect();
    cardSize.current = { w: r.width, h: r.height };
    setPos(clamp((window.innerWidth - r.width) / 2, (window.innerHeight - r.height) / 2));
  }, [clamp]);

  // 窗口 resize 时重新约束
  useEffect(() => {
    const onResize = () => {
      setPos((p) => {
        if (!p) return p;
        const r = cardRef.current?.getBoundingClientRect();
        if (r) cardSize.current = { w: r.width, h: r.height };
        return clamp(p.x, p.y);
      });
    };
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, [clamp]);

  // ── 拖拽事件 ──────────────────────────────────────────────────
  const startDrag = useCallback(
    (e: React.PointerEvent) => {
      if (e.button !== 0) return;
      e.preventDefault();
      dragging.current = true;
      dragStart.current = { x: e.clientX, y: e.clientY };
      setPos((p) => {
        posStart.current = p ?? { x: 0, y: 0 };
        return p;
      });

      const onMove = (me: PointerEvent) => {
        if (!dragging.current) return;
        setPos(
          clamp(
            posStart.current.x + (me.clientX - dragStart.current.x),
            posStart.current.y + (me.clientY - dragStart.current.y),
          ),
        );
      };
      const onUp = () => {
        dragging.current = false;
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
        window.removeEventListener("pointercancel", onUp);
      };
      document.body.style.cursor = "grabbing";
      document.body.style.userSelect = "none";
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
      window.addEventListener("pointercancel", onUp);
    },
    [clamp],
  );

  // ── 问题交互 ────────────────────────────────────────────────
  const toggle = (q: WireAskQuestion, label: string) => {
    setCustom((c) => ({ ...c, [q.id]: "" }));
    setSel((s) => {
      const cur = s[q.id] ?? [];
      if (q.multi) {
        return { ...s, [q.id]: cur.includes(label) ? cur.filter((x) => x !== label) : [...cur, label] };
      }
      return { ...s, [q.id]: [label] };
    });
  };

  const setTyped = (q: WireAskQuestion, text: string) => {
    setCustom((c) => ({ ...c, [q.id]: text }));
    if (text.trim()) setSel((s) => ({ ...s, [q.id]: [] }));
  };

  const answered = (q: WireAskQuestion) =>
    (sel[q.id]?.length ?? 0) > 0 || (custom[q.id]?.trim() ?? "") !== "";
  const allAnswered = ask.questions.every(answered);
  const answeredCount = ask.questions.filter(answered).length;

  const submit = () => {
    onAnswer(
      ask.id,
      ask.questions.map((q) => ({
        questionId: q.id,
        selected: custom[q.id]?.trim() ? [custom[q.id].trim()] : (sel[q.id] ?? []),
      })),
    );
  };

  // 浮卡首次定位后接管焦点，避免键盘用户仍停在被遮住的输入框里。
  const focusedRef = useRef(false);
  useEffect(() => {
    focusedRef.current = false;
  }, [ask.id]);
  useEffect(() => {
    if (!pos || focusedRef.current) return;
    focusedRef.current = true;
    cardRef.current
      ?.querySelector<HTMLElement>(".office-ask-option, .office-ask-custom, button")
      ?.focus();
  }, [ask.id, pos]);

  // 卡片标题：优先用 ask.title，否则用第一个问题的 header
  const cardTitle = ask.questions[0]?.header || "";
  const dialogTitle = ask.questions.length === 1 ? (cardTitle || t("ask.title")) : t("ask.title");

  return (
    <div className="office-ask-layer animate-[fadeIn_.15s_ease-out]">
      <div
        ref={cardRef}
        className="office-ask-card animate-[scaleIn_.2s_ease-out]"
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        tabIndex={-1}
        style={{
          ...(pos
            ? { position: "absolute", left: pos.x, top: pos.y }
            : { visibility: "hidden" })
        }}
      >
        {/* 拖拽只负责摆放，不是完成问题的必经操作。 */}
        <div
          className="office-ask-handle"
          onPointerDown={startDrag}
          title={t("ask.dragHint")}
          aria-hidden
        >
          <span />
        </div>

        <header className="office-ask-header">
          <div className="office-decision-head">
            <span className="office-decision-head__icon" aria-hidden>
              <MessageSquare size={14} />
            </span>
            <span className="office-decision-head__copy">
              <span className="office-decision-head__kicker">{t("ask.title")}</span>
              <h2 id={titleId} className="office-decision-head__title">
                {dialogTitle}
              </h2>
            </span>
          </div>
        </header>

        <div className="office-ask-progress">
          <div
            className="office-ask-progress__track"
            role="progressbar"
            aria-label={t("ask.answered", { n: answeredCount, total: ask.questions.length })}
            aria-valuemin={0}
            aria-valuemax={ask.questions.length}
            aria-valuenow={answeredCount}
          >
            <div
              className="office-ask-progress__fill"
              style={{ width: `${(answeredCount / Math.max(ask.questions.length, 1)) * 100}%` }}
            />
          </div>
          <span className="office-ask-progress__label">
            {t("ask.answered", { n: answeredCount, total: ask.questions.length })}
          </span>
        </div>

        <div className="office-ask-body">
        {ask.questions.map((q, questionIndex) => {
          const questionTitleId = `${idPrefix}-question-${questionIndex}`;
          return (
          <section className="office-ask-question" key={q.id} role="group" aria-labelledby={questionTitleId}>
            {q.header && ask.questions.length > 1 && (
              <h3 id={questionTitleId} className="office-ask-question__title">{q.header}</h3>
            )}
            {(!q.header || ask.questions.length === 1) && (
              <span id={questionTitleId} className="sr-only">{q.header || dialogTitle}</span>
            )}
            <div className="office-ask-question__prompt">
              <Markdown text={q.prompt} autoExportMermaid={false} />
            </div>
            <div className="office-ask-options">
              {q.options.map((o) => {
                const on = (sel[q.id] ?? []).includes(o.label);
                return (
                  <button
                    type="button"
                    key={o.label}
                    className="office-ask-option"
                    aria-pressed={on}
                    onClick={() => toggle(q, o.label)}
                  >
                    <span className="office-ask-option__mark" data-multi={q.multi ? "true" : "false"}>
                      {on && (
                        q.multi
                          ? <Check size={10} className="text-accent-fg" />
                          : <span className="w-1.5 h-1.5 rounded-full bg-accent-fg" />
                      )}
                    </span>
                    <span className="office-ask-option__copy">
                      <span className="office-ask-option__title">{o.label}</span>
                      {o.description && (
                        <span className="office-ask-option__hint">{o.description}</span>
                      )}
                    </span>
                  </button>
                );
              })}
            </div>
            <input
              className="office-ask-custom"
              placeholder={t("ask.customPlaceholder")}
              aria-label={t("ask.customPlaceholder")}
              value={custom[q.id] ?? ""}
              onChange={(e) => setTyped(q, e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey && allAnswered) {
                  e.preventDefault();
                  submit();
                }
              }}
            />
          </section>
          );
        })}
        </div>

        <footer className="office-ask-footer">
          <span className="office-ask-footer__hint">{t("ask.enterHint")}</span>
          <button
            type="button"
            className="office-ask-footer__dismiss px-4 py-2"
            onClick={onDismiss}
          >
            {t("ask.justChat")}
          </button>
          <button
            type="button"
            className="office-ask-footer__submit px-4 py-2"
            onClick={submit}
            disabled={!allAnswered}
          >
            {t("common.submit")}
          </button>
        </footer>
      </div>
    </div>
  );
}
