import { useMemo, useState } from "react";
import type { ToolSource } from "../../types/evidence";

function hostOf(raw: string): string {
  try {
    return new URL(raw).hostname.replace(/^www\./, "");
  } catch {
    return "";
  }
}

/** Compact evidence drawer shared by office and literature tool cards. */
export function ToolSources({ sources }: { sources?: ToolSource[] }) {
  const [open, setOpen] = useState(false);
  const unique = useMemo(() => {
    const seen = new Set<string>();
    return (sources ?? []).filter((source) => {
      if (!source.url || seen.has(source.url)) return false;
      seen.add(source.url);
      return true;
    });
  }, [sources]);
  if (unique.length === 0) return null;
  return (
    <div className="border-t border-border-soft/60 px-2 py-1" data-testid="tool-sources">
      <button
        type="button"
        className="inline-flex items-center gap-1 rounded border-0 bg-transparent px-1 py-0.5 text-[10.5px] text-fg-faint hover:bg-bg-soft hover:text-fg-dim"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span aria-hidden="true">◫</span>
        来源 {unique.length}
        <span aria-hidden="true">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className="mt-1 grid gap-1" data-testid="tool-source-list">
          {unique.map((source, index) => (
            <a
              key={source.url}
              href={source.url}
              target="_blank"
              rel="noreferrer"
              className="block rounded-md border border-border-soft/70 bg-bg-soft/50 px-2 py-1.5 no-underline hover:border-accent/30 hover:bg-bg-soft"
            >
              <div className="flex items-center gap-1.5 text-[9.5px] text-fg-faint">
                <span className="font-mono tabular-nums">{index + 1}</span>
                <span className="truncate">{source.source || hostOf(source.url)}</span>
              </div>
              <div className="mt-0.5 line-clamp-2 text-[11px] leading-snug text-fg-dim">{source.title}</div>
              {source.snippet && (
                <div className="mt-0.5 line-clamp-3 text-[10.5px] leading-relaxed text-fg-faint">{source.snippet}</div>
              )}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
