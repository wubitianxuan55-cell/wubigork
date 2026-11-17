import { useEffect, useRef, useState } from "react";
import { firstFittingLine, lineHintsWidth, wrapHintRows, type HintItem } from "../../lib/hintLayout";

export interface HintStripProps {
  variants: readonly (readonly HintItem[])[];
  className?: string;
}

export function HintStrip({ variants, className = "" }: HintStripProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const node = ref.current;
    if (!node || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver((entries) => {
      setWidth(Math.floor(entries[0]?.contentRect.width ?? node.getBoundingClientRect().width));
    });
    observer.observe(node);
    setWidth(Math.floor(node.getBoundingClientRect().width));
    return () => observer.disconnect();
  }, []);

  const full = variants[0] ?? [];
  const selected = width > 0 ? firstFittingLine(variants, width) : full;
  const rows = selected
    ? [selected]
    : wrapHintRows(variants[variants.length - 1] ?? [], Math.max(width, 1));

  return (
    <div ref={ref} className={className} data-hint-width={lineHintsWidth(full)}>
      {rows.map((row, rowIndex) => (
        <span key={rowIndex} className="inline-flex flex-wrap items-center gap-x-3 gap-y-1">
          {row.map((hint) => (
            <span key={hint.id} className="whitespace-nowrap">
              {hint.keys.map((key) => <kbd key={key}>{key}</kbd>)}
              {" "}
              {hint.label}
            </span>
          ))}
        </span>
      ))}
    </div>
  );
}
