import { useMemo, useState } from "react";
import { parseJsonlRecords } from "../lib/jsonlRecords";

export function JsonlPreview({ text }: { text: string }) {
  const [mode, setMode] = useState<"cards" | "raw">("cards");
  const records = useMemo(() => parseJsonlRecords(text), [text]);
  const invalid = records.filter((record) => !record.valid).length;
  return (
    <div className="flex min-h-full flex-col" data-testid="jsonl-preview">
      <div className="sticky top-0 z-10 flex items-center gap-2 border-b border-border-soft bg-bg/95 px-3 py-1.5 text-[10.5px] text-fg-faint backdrop-blur">
        <span>JSONL · {records.length} 条记录</span>
        {invalid > 0 && <span className="text-amber-500">{invalid} 条无效</span>}
        <span className="ml-auto flex rounded-md border border-border-soft p-0.5">
          <button
            type="button"
            className={`rounded px-2 py-0.5 border-0 cursor-pointer ${mode === "cards" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint"}`}
            onClick={() => setMode("cards")}
          >
            记录卡
          </button>
          <button
            type="button"
            className={`rounded px-2 py-0.5 border-0 cursor-pointer ${mode === "raw" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint"}`}
            onClick={() => setMode("raw")}
          >
            原文
          </button>
        </span>
      </div>
      {mode === "raw" ? (
        <pre className="p-3 font-mono text-[11px] leading-relaxed whitespace-pre-wrap text-fg-dim">{text}</pre>
      ) : records.length === 0 ? (
        <div className="px-3 py-8 text-center text-[11px] text-fg-faint">没有可显示的 JSONL 记录</div>
      ) : (
        <div className="flex flex-col gap-1.5 p-2.5">
          {records.map((record) => (
            <details
              key={record.line}
              open={!record.valid}
              data-testid={`jsonl-record-${record.line}`}
              className={`rounded-md border ${record.valid ? "border-border-soft bg-bg-soft/20" : "border-amber-500/30 bg-amber-500/5"}`}
            >
              <summary className="cursor-pointer list-none px-2.5 py-1.5 text-[10.5px]">
                <span className="mr-2 font-mono text-fg-faint">L{record.line}</span>
                <span className={record.valid ? "text-fg-dim" : "text-amber-500"}>{record.summary}</span>
              </summary>
              <pre className="m-0 max-h-[34vh] overflow-auto border-t border-border-soft/60 px-2.5 py-2 font-mono text-[10.5px] leading-relaxed whitespace-pre-wrap text-fg-dim">
                {record.formatted}{record.truncated ? "\n…（记录过长，已预览截断）" : ""}
              </pre>
            </details>
          ))}
          {records.length >= 500 && (
            <div className="py-1 text-center text-[10px] text-fg-faint">仅展示前 500 条记录，可切换原文查看完整内容</div>
          )}
        </div>
      )}
    </div>
  );
}
