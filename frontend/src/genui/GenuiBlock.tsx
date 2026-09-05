/* eslint-disable react-refresh/only-export-components -- 常量/渲染函数与组件同文件导出 */
// GenuiBlock — 渲染一个已修复的 GenUI 规格。
// 状态归属：块级 answers/fields/meta/locked；本地判卷与持久化都在本层完成。
import { memo, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { type GenuiSpec } from "./spec";
import { useGenuiAction, type GenuiActionHandler } from "./GenuiActionContext";
import { loadBlockState, saveBlockState } from "./interaction";
import { renderNode } from "./renderNode";
import { BlockApiContext, type BlockApi, type QuestionMeta } from "./blocks/state";
import "./styles.css";

export const GENUI_ACTION_DEBOUNCE_MS = 300;

export interface GenuiBlockProps {
  spec: GenuiSpec;
  /** 有值时启用持久化（会话+消息槽+指纹）；无值 = 流式 volatile 实例。 */
  stateKey?: string;
}

function specEquivalent(a: GenuiSpec, b: GenuiSpec): boolean {
  if (a === b) return true;
  return JSON.stringify(a) === JSON.stringify(b);
}

function useDebouncedAction(
  onAction: GenuiActionHandler | undefined,
): ((action: string, payload: Record<string, unknown>) => void) | undefined {
  const pending = useRef(new Map<string, ReturnType<typeof setTimeout>>());
  const actionRef = useRef(onAction);
  actionRef.current = onAction;

  useEffect(() => {
    const timers = pending.current;
    return () => {
      for (const timer of timers.values()) clearTimeout(timer);
      timers.clear();
    };
  }, []);

  const debounced = useCallback((action: string, payload: Record<string, unknown>): void => {
    const existing = pending.current.get(action);
    if (existing !== undefined) clearTimeout(existing);
    pending.current.set(
      action,
      setTimeout(() => {
        pending.current.delete(action);
        actionRef.current?.(action, payload);
      }, GENUI_ACTION_DEBOUNCE_MS),
    );
  }, []);
  return onAction === undefined ? undefined : debounced;
}

function GenuiBlockInstance({ spec, stateKey }: GenuiBlockProps) {
  const onAction = useGenuiAction();
  const emitRaw = useDebouncedAction(onAction);
  const [persisted] = useState(() => (stateKey === undefined ? null : loadBlockState(stateKey)));
  const [answers, setAnswers] = useState<Record<string, string>>(persisted?.answers ?? {});
  const [fields, setFields] = useState<Record<string, string>>(persisted?.fields ?? {});
  const [meta, setMeta] = useState<Record<string, QuestionMeta>>({});
  const [locked, setLocked] = useState(persisted?.locked === true);
  const [round, setRound] = useState(0);
  const [secretFields, setSecretFields] = useState<ReadonlySet<string>>(() => new Set());

  const setAnswer = useCallback((group: string, label: string): void => {
    setAnswers((prev) => (prev[group] === label ? prev : { ...prev, [group]: label }));
  }, []);

  const setField = useCallback((id: string, value: string): void => {
    setFields((prev) => {
      if (value.trim() === "") {
        if (!(id in prev)) return prev;
        const next = { ...prev };
        delete next[id];
        return next;
      }
      return prev[id] === value ? prev : { ...prev, [id]: value };
    });
  }, []);

  const registerMeta = useCallback((group: string, m: QuestionMeta): void => {
    setMeta((prev) => {
      const existing = prev[group];
      if (
        existing !== undefined &&
        existing.answer === m.answer &&
        existing.explanation === m.explanation
      ) {
        return prev;
      }
      return { ...prev, [group]: m };
    });
  }, []);

  const registerSecret = useCallback((id: string): void => {
    setSecretFields((prev) => (prev.has(id) ? prev : new Set(prev).add(id)));
  }, []);

  const isSecret = useCallback((id: string): boolean => secretFields.has(id), [secretFields]);

  const reset = useCallback((notifyAction?: string): void => {
    setAnswers({});
    setLocked(false);
    setRound((r) => r + 1);
    if (notifyAction !== undefined && onAction !== undefined) onAction(notifyAction, {});
  }, [onAction]);

  const api = useMemo<BlockApi>(
    () => ({
      answers,
      fields,
      meta,
      locked,
      round,
      hasAction: onAction !== undefined,
      setAnswer,
      setField,
      registerMeta,
      registerSecret,
      isSecret,
      lock: () => setLocked(true),
      reset,
      emit: (action, payload) => {
        emitRaw?.(action, payload);
      },
    }),
    [answers, fields, meta, locked, round, onAction, setAnswer, setField, registerMeta, registerSecret, isSecret, reset, emitRaw],
  );

  useEffect(() => {
    if (stateKey === undefined) return;
    const timer = setTimeout(() => {
      const safeFields = Object.fromEntries(
        Object.entries(fields).filter(([id]) => !secretFields.has(id)),
      );
      saveBlockState(stateKey, {
        answers,
        locked,
        ...(Object.keys(safeFields).length > 0 ? { fields: safeFields } : {}),
      });
    }, 300);
    return () => clearTimeout(timer);
  }, [stateKey, answers, locked, fields, secretFields]);

  const gap = spec.gap ?? 14;
  return (
    <div className="gui-block" data-genui>
      {spec.title !== undefined && <div className="gui-banner">{spec.title}</div>}
      <BlockApiContext.Provider value={api}>
        <div className="gui-col" style={{ gap }}>
          {spec.items.map((node, i) => (
            <div className="gui-item" key={i}>
              {renderNode(node, 0)}
            </div>
          ))}
        </div>
      </BlockApiContext.Provider>
    </div>
  );
}

/** 同一 stateKey 下持久化状态只在实例间迁移一次；流式 volatile 共享同一实例。 */
export const GenuiBlock = memo(function GenuiBlock(props: GenuiBlockProps) {
  const instanceKey = props.stateKey === undefined ? "volatile" : `durable:${props.stateKey}`;
  return <GenuiBlockInstance key={instanceKey} {...props} />;
}, (prev, next) => prev.stateKey === next.stateKey && specEquivalent(prev.spec, next.spec));

export function renderGenuiSpec(spec: GenuiSpec): ReactNode {
  return <GenuiBlock spec={spec} />;
}
