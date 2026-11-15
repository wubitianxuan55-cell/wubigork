import { useCallback, useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Check, X } from "../icons";

interface CropRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

// ScreenCropOverlay 截图裁剪浮层：全屏遮罩 + 拖拽选区 + 画布裁剪。
// 选区坐标基于图片显示区域，确认时按原图尺寸缩放裁剪。
export function ScreenCropOverlay({
  src,
  onCancel,
  onConfirm,
}: {
  src: string;
  onCancel: () => void;
  onConfirm: (dataUrl: string) => void;
}) {
  const imgRef = useRef<HTMLImageElement>(null);
  const dragRef = useRef<{ x: number; y: number } | null>(null);
  const scaleRef = useRef(1);
  const [rect, setRect] = useState<CropRect | null>(null);
  const [imgPos, setImgPos] = useState({ left: 0, top: 0 });
  const [busy, setBusy] = useState(false);

  const onPointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    const img = imgRef.current;
    if (!img || e.button !== 0) return;
    const imgRect = img.getBoundingClientRect();
    scaleRef.current = img.naturalWidth / Math.max(1, imgRect.width);
    setImgPos({ left: imgRect.left, top: imgRect.top });
    const x = e.clientX - imgRect.left;
    const y = e.clientY - imgRect.top;
    dragRef.current = { x, y };
    setRect({ x, y, w: 0, h: 0 });
  };

  const onPointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    const start = dragRef.current;
    const img = imgRef.current;
    if (!start || !img) return;
    const imgRect = img.getBoundingClientRect();
    const cx = e.clientX - imgRect.left;
    const cy = e.clientY - imgRect.top;
    setRect({
      x: Math.min(start.x, cx),
      y: Math.min(start.y, cy),
      w: Math.abs(cx - start.x),
      h: Math.abs(cy - start.y),
    });
  };

  const endDrag = () => {
    dragRef.current = null;
  };

  const confirm = useCallback(() => {
    const img = imgRef.current;
    if (!img || !rect || rect.w < 2 || rect.h < 2 || busy) return;
    setBusy(true);
    try {
      const s = scaleRef.current;
      const sx = Math.max(0, Math.round(rect.x * s));
      const sy = Math.max(0, Math.round(rect.y * s));
      const sw = Math.min(img.naturalWidth - sx, Math.round(rect.w * s));
      const sh = Math.min(img.naturalHeight - sy, Math.round(rect.h * s));
      if (sw < 2 || sh < 2) return;
      const canvas = document.createElement("canvas");
      canvas.width = sw;
      canvas.height = sh;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      ctx.drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh);
      onConfirm(canvas.toDataURL("image/png"));
    } finally {
      setBusy(false);
    }
  }, [rect, busy, onConfirm]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
      if (e.key === "Enter") confirm();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onCancel, confirm]);

  const sel = rect && rect.w >= 2 && rect.h >= 2 ? rect : null;

  return (
    <div
      className="fixed inset-0 z-[120] flex items-center justify-center bg-black/60 backdrop-blur-[1px] select-none no-drag"
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={endDrag}
      onPointerLeave={endDrag}
    >
      <img
        ref={imgRef}
        src={src}
        alt=""
        className="max-w-[calc(100vw-32px)] max-h-[calc(100vh-72px)] rounded-lg shadow-2xl pointer-events-none"
        draggable={false}
      />

      {/* 选区高亮 + 遮罩开孔 */}
      {sel && (
        <div
          className="absolute border-2 border-accent pointer-events-none"
          style={{
            left: imgPos.left + sel.x,
            top: imgPos.top + sel.y,
            width: sel.w,
            height: sel.h,
            boxShadow: "0 0 0 9999px rgba(0,0,0,0.55)",
          }}
        />
      )}

      {/* 底部操作栏 */}
      <div
        className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-2 px-3 py-2 rounded-xl bg-bg-elev border border-border-soft shadow-lg"
        onPointerDown={(e) => e.stopPropagation()}
      >
        <span className="text-xs text-fg-dim mr-1 select-none">
          {sel ? `${Math.round(sel.w)} × ${Math.round(sel.h)}` : "拖拽选择区域"}
        </span>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border-0 bg-accent text-accent-fg text-xs cursor-pointer hover:brightness-110 disabled:opacity-50 disabled:cursor-default"
          onClick={confirm}
          disabled={!sel || busy}
        >
          <Check size={13} /> 确定
        </button>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border-0 bg-bg-elev-2 text-fg-dim text-xs cursor-pointer hover:text-fg"
          onClick={onCancel}
        >
          <X size={13} /> 取消
        </button>
      </div>
    </div>
  );
}
