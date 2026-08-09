import { memo, useCallback, useEffect, useState } from "react";
import {
  ExternalLink,
  File,
  FilePpt,
  FileSpreadsheet,
  FileText,
  Paperclip,
  RefreshCw,
} from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import type { FileSearchHit } from "../lib/types";
import { useToast } from "./Toast";

// 资料分组：文档 / 表格 / 演示 / PDF
const GROUPS: { key: string; label: string; re: RegExp; icon: typeof FileText }[] = [
  { key: "docs", label: "文档", re: /\.(docx?|md|markdown|txt)$/i, icon: FileText },
  { key: "sheets", label: "表格", re: /\.(xlsx?|csv)$/i, icon: FileSpreadsheet },
  { key: "slides", label: "演示", re: /\.(pptx?)$/i, icon: FilePpt },
  { key: "pdf", label: "PDF", re: /\.pdf$/i, icon: File },
];

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

// MaterialsPanel — 右侧「资料」视图：工作区 office/文本文件按类型分组、
// 最新在前；每条可预览，或一键 @ 引用为对话上下文（对标千问办公/aily 的
// 开工前资料盘点）。
export const MaterialsPanel = memo(function MaterialsPanel({
  onOpenFile,
}: {
  onOpenFile: (path: string) => void;
}) {
  const [items, setItems] = useState<FileSearchHit[]>([]);
  const [loading, setLoading] = useState(false);
  const requestAt = useComposerInsertStore((s) => s.requestAt);
  const toast = useToast();

  const load = useCallback(() => {
    setLoading(true);
    app.Materials(120)
      .then((h) => setItems(h ?? []))
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => { load(); }, [load]);

  const reference = useCallback((path: string) => {
    requestAt(path);
    toast.show(`已引用 @${path}`, "info");
  }, [requestAt, toast]);

  const total = items.length;
  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <Paperclip size={13} className="text-accent" />
          资料
        </span>
        <div className="flex items-center gap-1">
          {total > 0 && (
            <span className="text-[10px] text-fg-faint border border-border-soft/60 rounded-full px-1.5 py-px">
              {total}
            </span>
          )}
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={load}
            title="刷新资料列表"
          >
            <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
          </button>
        </div>
      </div>

      {total === 0 ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center text-fg-faint/50">
          <FileText size={24} className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            工作区暂无资料
            <br />
            （docx / xlsx / pptx / pdf / md / csv）
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-2">
          {GROUPS.map((g) => {
            const rows = items.filter((f) => g.re.test(f.name));
            if (rows.length === 0) return null;
            const Icon = g.icon;
            return (
              <div key={g.key}>
                <div className="text-[10px] uppercase tracking-wider text-fg-faint/60 font-medium px-1 mb-1">
                  {g.label} · {rows.length}
                </div>
                <div className="flex flex-col gap-0.5">
                  {rows.slice(0, 20).map((f) => (
                    <div
                      key={f.path}
                      className="group flex items-center gap-2 px-2 py-1 rounded-md border border-border-soft/60 bg-bg-soft/25 hover:border-accent/30 hover:bg-bg-soft/60 transition-colors"
                    >
                      <span className="shrink-0 w-6 h-6 rounded-md bg-accent/10 text-accent flex items-center justify-center">
                        <Icon size={12} />
                      </span>
                      <button
                        type="button"
                        onClick={() => onOpenFile(f.path)}
                        title={`点击预览 ${f.path}`}
                        className="min-w-0 flex-1 text-left cursor-pointer"
                      >
                        <span className="block truncate text-[12px] text-fg font-medium leading-tight">
                          {f.name}
                        </span>
                        <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">
                          {f.path}
                          {f.size ? ` · ${fmtSize(f.size)}` : ""}
                        </span>
                      </button>
                      <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          type="button"
                          className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                          onClick={() => reference(f.path)}
                          title="一键 @ 引用为对话上下文"
                        >
                          <Paperclip size={12} />
                        </button>
                        <button
                          type="button"
                          className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                          onClick={() => void app.OpenWorkspacePath(f.path).catch(() => {})}
                          title="在外部程序中打开"
                        >
                          <ExternalLink size={12} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
});
