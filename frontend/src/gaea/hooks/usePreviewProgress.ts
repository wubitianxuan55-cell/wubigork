import { useEffect, useState } from "react";
import { onEvent } from "../lib/bridge";
import type { WireEvent } from "../lib/types";

// usePreviewProgress 订阅扫描件 PDF 预览的 OCR 逐页进度（preview_progress 事件）。
// 仅在 OCR 真正逐页识别时收到事件；文本型 PDF 全程无进度，返回 null。
export function usePreviewProgress(relPath: string | null) {
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null);

  useEffect(() => {
    setProgress(null);
    if (!relPath) return;
    const off = onEvent((e: WireEvent) => {
      if (e.kind === "preview_progress" && e.progress?.path === relPath) {
        setProgress({ done: e.progress.done, total: e.progress.total });
      }
    });
    return off;
  }, [relPath]);

  return progress;
}
