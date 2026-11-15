import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronDown, Clock } from "../icons";
import { app } from "../lib/bridge";
import { usePreviewStore } from "../lib/store";
import { useToast } from "./Toast";
import type { JournalChangeRecord } from "../lib/types";
import {
  groupVersionsByPath,
  normalizeVersionPath,
  versionLabel,
  versionStatusText,
  versionTimeText,
} from "../lib/versionTimeline";
import { VersionTimeline } from "./VersionTimeline";

/** FileVersionStrip — 6.1 可审计默认化：文件工作台的版本时间线一级入口。
 *  默认可见的版本条（三件套通用，任何有证据卡的文件都有）：显示该文件的
 *  版本数、最近 AI 改动时间与复核状态；展开为该文件的逐版本时间线
 *  （VersionTimeline：预览基线 / 一键恢复）。无版本时显示「AI 改动会自动
 *  成为版本点」的说明条——回滚入口从「能力」（交付面板徽标）升为默认 UI。
 *  数据源与交付面板同源：GaeaJournalList 证据卡（挂载/换文件时拉一次，
 *  恢复成功后重拉——恢复动作自身也生成新证据卡）。 */
export function FileVersionStrip({
  relPath,
  onRestored,
}: {
  relPath: string;
  /** 恢复成功后回调（父级静默重读预览——基线写回目标，内容已变化）。 */
  onRestored?: () => void;
}) {
  const toast = useToast();
  const openFilePreview = usePreviewStore((s2) => s2.openFilePreview);
  const [records, setRecords] = useState<JournalChangeRecord[] | null>(null);
  const [open, setOpen] = useState(false);

  const load = useCallback(async () => {
    try {
      const recs = await app.GaeaJournalList(200);
      setRecords(recs ?? []);
    } catch {
      setRecords([]); // 静默降级：证据链不可用时仅无版本数据，不影响预览
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, relPath]);

  const versions = useMemo(
    () => groupVersionsByPath(records ?? []).get(normalizeVersionPath(relPath)) ?? [],
    [records, relPath],
  );
  const latest = versions[0] ?? null;

  const doRestore = useCallback(
    async (r: JournalChangeRecord) => {
      try {
        await app.RollbackRecord(r.id);
        toast.show(`已恢复到 ${versionLabel(r)}（基线快照写回）`, "info");
        void load();
        onRestored?.();
      } catch (e) {
        toast.show(`恢复失败：${e instanceof Error ? e.message : String(e)}`, "warn");
      }
    },
    [load, onRestored, toast],
  );

  return (
    <div className="shrink-0 border-b border-border-soft" data-testid="file-version-strip">
      <button
        className="w-full flex items-center gap-2 px-3 py-1.5 bg-transparent border-0 text-left cursor-pointer hover:bg-bg-soft/50 transition-colors"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        title="本文件的 AI 改动版本时间线（证据链自动快照，可预览/恢复）"
        type="button"
      >
        <Clock size={11} className="text-fg-faint shrink-0" />
        {versions.length > 0 ? (
          <span className="text-fg-dim text-[10.5px] truncate">
            版本 <span className="text-accent font-medium">{versions.length}</span>
            <span className="text-fg-faint">
              {" "}· 最近 AI 改动 {versionTimeText(latest!)} · {versionStatusText(latest!.status)}
            </span>
            {latest!.baselinePath && <span className="text-fg-faint"> · 可恢复</span>}
          </span>
        ) : (
          <span className="text-fg-faint text-[10.5px]">
            AI 改动此文件会自动成为版本点（可预览 / 一键恢复）
          </span>
        )}
        <ChevronDown
          size={11}
          className={`ml-auto text-fg-faint transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open && (
        <div className="px-3 pb-2">
          <VersionTimeline
            path={relPath}
            records={records === null ? null : versions}
            onPreview={(baselinePath) => openFilePreview(baselinePath)}
            onRestore={doRestore}
          />
        </div>
      )}
    </div>
  );
}

