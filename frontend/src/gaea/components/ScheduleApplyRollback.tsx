import { useEffect, useState } from "react";
import { Loader2, Rollback } from "../icons";
import { app } from "../lib/bridge";
import type { JournalChangeRecord } from "../lib/types";
import { useToast } from "./Toast";

// ScheduleApplyRollback — schedule_apply 回执卡「回滚本次」行（diff 确认闭环刀A）。
//
// Why: schedule_apply 的证据卡一直落在证据链 Journal 里，但变更 tab 白名单此前
// 不含它，进度计划侧没有任何可点的回滚入口（设计
// docs/gaea-schedule-diff-confirm-design-2026-09.md §1.5/§2.3 刀A——先把
// 「easy revert」补齐，独立于事前确认卡成立）。
//
// 匹配口径：target（args.path，缺省=Go DefaultRelPath 镜像常量）对 Journal 里
// tool==="schedule_apply" 的**最新一条带基线快照**的记录——与 ChangesPanel 的
// 「按 target 匹配最新」同款语义。多次 apply 同一路径时 Journal 无调用级关联键，
// 不逐卡区分（设计 §5 欠账另案）：按钮标题显式带 turn，回滚的总是该路径最近
// 一次 apply 的写盘前基线；「已被手工修改」守卫与恢复前再快照都在 Go 侧
// GaeaRollbackRecord 内置，误触不静默。
//
// 常驻可见（对齐 TaskLiveRow 位置，不进折叠区）：回滚是后置闭环的主入口，
// 藏进展开体等于没有。无匹配记录（Journal 未就绪/旧会话无快照）整行不渲染。

/** 与 schedule/gschedSummary 的 SCHEDULE_FILE_PATH、Go DefaultRelPath 同源镜像
 *  （changes.ts DEFAULT 同款先例）：避免本组件拉起整个进度计划 store 图谱。 */
const SCHEDULE_DEFAULT_REL = "进度计划/当前计划.gsched.json";

function resolveTarget(args: string): string {
  try {
    const parsed = JSON.parse(args || "{}") as { path?: unknown };
    if (typeof parsed.path === "string" && parsed.path.trim() !== "") return parsed.path.trim();
  } catch { /* 缺省路径兜底 */ }
  return SCHEDULE_DEFAULT_REL;
}

export function ScheduleApplyRollback({ args }: { args: string }) {
  const toast = useToast();
  const [record, setRecord] = useState<JournalChangeRecord | null>(null);
  const [rollingBack, setRollingBack] = useState(false);

  const target = resolveTarget(args);
  useEffect(() => {
    let alive = true;
    Promise.resolve(app.GaeaJournalList?.(60) ?? [])
      .then((rs) => {
        if (!alive) return;
        const hit = (rs as JournalChangeRecord[])
          .filter((r) => r.tool === "schedule_apply" && r.target === target && !!r.baselinePath && !!r.id)
          .at(-1);
        setRecord(hit ?? null);
      })
      .catch(() => { /* Journal 未就绪（旧会话/绑定缺位）→ 整行不渲染，诚实无入口 */ });
    return () => { alive = false; };
  }, [target]);

  if (!record) return null;

  const doRollback = async () => {
    if (rollingBack) return;
    setRollingBack(true);
    try {
      await app.RollbackRecord(record.id);
      toast.show(`已回滚 ${target}（基线快照恢复，板块回读即见）`, "info");
    } catch (e) {
      toast.show(`回滚失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    } finally {
      setRollingBack(false);
    }
  };

  return (
    <div className="px-3 py-1 flex items-center gap-2 border-b border-border-soft">
      <span className="text-[10px] text-fg-faint truncate">
        本次修改已留基线快照，可整体还原（turn {record.turn}）
      </span>
      <button
        className="ml-auto shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-err/40 bg-transparent text-err text-[10px] cursor-pointer hover:bg-err/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        onClick={() => void doRollback()}
        disabled={rollingBack}
        title={`回滚到写盘前基线快照（turn ${record.turn} · schedule_apply）`}
        data-testid="sched-apply-rollback"
      >
        {rollingBack ? <Loader2 size={10} className="animate-spin" /> : <Rollback size={10} aria-hidden />}
        回滚本次
      </button>
    </div>
  );
}
