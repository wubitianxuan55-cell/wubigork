import { memo, useCallback, useState } from "react";
import { Button, Collapse, Typography } from "antd";
import { Loader } from "../icons";
import { useT } from "../lib/i18n";
import type { SkillDistillCandidateView, SkillDistillView } from "../lib/types";
import { EmptyState } from "./EmptyState";

const { Text } = Typography;

// SkillDistillSection — 流程蒸馏分区（阶段七 7.2-2 journal 历史蒸馏）：
// 确定性挖掘出的跨会话重复流程候选 → 建议卡只提示不打扰。每卡=模式步骤序
// （PatternLine 行，工具名 code 样式）+「重复 N 次」徽章 + 最近日期 + 证据
// 折叠（sessions/evidence 行）+ 动作行〔结晶为技能〕（LLM 蒸馏→7.2-1 审阅
// 弹窗，由容器层驱动）〔不再提示〕（Decide ignore，静默不再出现）。
// 空态两分：view=null/available=false → 不可用；candidates 空 → 暂无重复流程。

// lastAt 兼容秒/毫秒两种 int64 时间戳，统一转毫秒后本地化日期（面板原生
// Date 先例，不引 dayjs）。
function formatLastAt(at: number): string {
  const ms = at > 0 ? (at > 1e12 ? at : at * 1000) : 0;
  return ms > 0 ? new Date(ms).toLocaleDateString() : "";
}

// PatternLine 步骤行（"edit_file .md"）：首段工具名 code 样式，其余为形状。
const PatternStepLine = memo(function PatternStepLine({ line }: { line: string }) {
  const sp = line.indexOf(" ");
  const tool = sp === -1 ? line : line.slice(0, sp);
  const shape = sp === -1 ? "" : line.slice(sp + 1);
  return (
    <div className="flex items-baseline gap-1.5 leading-relaxed">
      <Text code className="!text-[11px]">
        {tool}
      </Text>
      {shape && (
        <Text type="secondary" className="!text-[11px]">
          {shape}
        </Text>
      )}
    </div>
  );
});

const CandidateCard = memo(function CandidateCard(p: {
  item: SkillDistillCandidateView;
  busy: boolean;
  action: "draft" | "ignore" | null;
  onDraft: (id: string) => void;
  onIgnore: (id: string) => void;
}) {
  const t = useT();
  const { item, busy, action } = p;
  const lines = item.evidence.length > 0 ? item.evidence : item.sessions;
  return (
    <div
      className="border border-border-soft rounded-xl p-3.5 bg-bg-soft/60 hover:bg-bg-soft transition-colors"
      data-testid={`distill-card-${item.id}`}
    >
      <div className="flex items-center gap-1.5 mb-1.5">
        <span className="text-accent text-[10px] font-semibold uppercase tracking-wider bg-accent/10 px-1.5 py-0.5 rounded">
          {t("memory.distill.repeatBadge", { n: item.repeat })}
        </span>
        {item.lastAt > 0 && (
          <span className="text-fg-faint text-[10px]">
            {t("memory.distill.lastAt")}
            {formatLastAt(item.lastAt)}
          </span>
        )}
      </div>
      <div className="flex flex-col gap-0.5">
        {item.pattern.map((line, i) => (
          <PatternStepLine key={`${i}-${line}`} line={line} />
        ))}
      </div>
      {lines.length > 0 && (
        <Collapse
          ghost
          size="small"
          className="distill-evidence"
          items={[
            {
              key: "ev",
              label: (
                <span className="text-[11px] text-fg-faint select-none">
                  {t("memory.distill.evidence")}
                </span>
              ),
              children: (
                <div className="flex flex-col gap-1">
                  {lines.map((e, i) => (
                    <div key={`${i}-${e}`} className="text-fg-faint/70 text-[10.5px] leading-relaxed font-mono break-all">
                      {e}
                    </div>
                  ))}
                </div>
              ),
            },
          ]}
        />
      )}
      <div className="flex items-center gap-2 mt-1">
        <Button
          size="small"
          type="primary"
          loading={busy && action === "draft"}
          disabled={busy}
          onClick={() => p.onDraft(item.id)}
        >
          {busy && action === "draft" ? t("memory.distill.drafting") : t("memory.distill.draft")}
        </Button>
        <Button
          size="small"
          disabled={busy}
          loading={busy && action === "ignore"}
          onClick={() => p.onIgnore(item.id)}
        >
          {t("memory.distill.ignore")}
        </Button>
      </div>
    </div>
  );
});

export const SkillDistillSection = memo(function SkillDistillSection(p: {
  view: SkillDistillView | null;
  loading?: boolean;
  onDraft: (id: string) => Promise<void> | void;
  onIgnore: (id: string) => Promise<void> | void;
}) {
  const t = useT();
  const [busy, setBusy] = useState<{ id: string; action: "draft" | "ignore" } | null>(null);

  const run = useCallback(
    async (id: string, action: "draft" | "ignore", fn: (id: string) => Promise<void> | void) => {
      if (busy) return;
      setBusy({ id, action });
      try {
        await fn(id);
      } finally {
        setBusy(null);
      }
    },
    [busy],
  );

  const onDraft = useCallback((id: string) => void run(id, "draft", p.onDraft), [run, p.onDraft]);
  const onIgnore = useCallback((id: string) => void run(id, "ignore", p.onIgnore), [run, p.onIgnore]);

  const candidates = p.view?.candidates ?? [];

  return (
    <section className="flex flex-col gap-1.5" data-testid="distill-section">
      <div className="flex items-center gap-1.5">
        <span className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider">
          {t("memory.distill.title")}
        </span>
        {p.loading && <Loader size={11} className="animate-spin text-fg-faint" aria-label={t("memory.distill.loading")} />}
      </div>
      <div className="text-fg-faint/60 text-[10.5px]">{t("memory.distill.hint")}</div>
      {p.loading && candidates.length === 0 ? (
        <div className="py-4 text-center text-fg-faint text-[12px]" data-testid="distill-loading">
          {t("memory.distill.loading")}
        </div>
      ) : !p.view || !p.view.available ? (
        <EmptyState message={t("memory.distill.unavailable")} />
      ) : candidates.length === 0 ? (
        <EmptyState message={t("memory.distill.empty")} />
      ) : (
        <div className="flex flex-col gap-2">
          {candidates.map((c) => (
            <CandidateCard
              key={c.id}
              item={c}
              busy={busy?.id === c.id}
              action={busy?.id === c.id ? busy.action : null}
              onDraft={onDraft}
              onIgnore={onIgnore}
            />
          ))}
        </div>
      )}
    </section>
  );
});
