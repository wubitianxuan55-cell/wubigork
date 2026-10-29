import { useEffect, useState } from "react";
import { onEvent } from "../lib/bridge";
import type { WireEvent } from "../lib/types";

// usePreviewProgress 订阅扫描件 PDF 预览的 OCR 逐页进度（preview_progress 事件）。
// 仅在 OCR 真正逐页识别时收到事件；文本型 PDF 全程无进度，返回 null。
// elapsedSec=自解析起点已耗时（整秒，可与 done 同显给时间感；老事件无该字段为 undefined）。
export interface PreviewProgress {
  done: number;
  total: number;
  elapsedSec?: number;
}

export function usePreviewProgress(relPath: string | null) {
  const [progress, setProgress] = useState<PreviewProgress | null>(null);

  useEffect(() => {
    setProgress(null);
    if (!relPath) return;
    const off = onEvent((e: WireEvent) => {
      if (e.kind === "preview_progress" && e.progress?.path === relPath) {
        setProgress({
          done: e.progress.done,
          total: e.progress.total,
          elapsedSec: e.progress.elapsedSec,
        });
      }
    });
    return off;
  }, [relPath]);

  return progress;
}
