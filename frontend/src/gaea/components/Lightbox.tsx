// Lightbox.tsx — 全屏大图查看器，支持键盘翻页、下载
import { useEffect, useCallback } from "react";
import { X, ChevronLeft, ChevronRight, Download } from "lucide-react";

export interface LightboxImage {
  dataUrl: string; // base64 data URL
  prompt?: string;
  seed?: number;
}

interface LightboxProps {
  images: LightboxImage[];
  index: number;
  onClose: () => void;
  onNavigate: (index: number) => void;
}

export function Lightbox({ images, index, onClose, onNavigate }: LightboxProps) {
  const current = images[index];
  const hasPrev = index > 0;
  const hasNext = index < images.length - 1;

  const goPrev = useCallback(() => {
    if (hasPrev) onNavigate(index - 1);
  }, [hasPrev, index, onNavigate]);

  const goNext = useCallback(() => {
    if (hasNext) onNavigate(index + 1);
  }, [hasNext, index, onNavigate]);

  // 下载当前图片
  const download = useCallback(() => {
    if (!current) return;
    const a = document.createElement("a");
    a.href = current.dataUrl;
    a.download = `gaea-${current.seed || Date.now()}.png`;
    a.click();
  }, [current]);

  // 键盘事件
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      switch (e.key) {
        case "Escape":
          onClose();
          break;
        case "ArrowLeft":
          goPrev();
          break;
        case "ArrowRight":
          goNext();
          break;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, goPrev, goNext]);

  if (!current) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm animate-[fadeIn_0.2s_ease-out]"
      onClick={onClose}
    >
      {/* 关闭按钮 */}
      <button
        onClick={onClose}
        className="absolute top-4 right-4 z-10 rounded-full bg-black/40 backdrop-blur-md p-2 text-white/80 hover:bg-black/60 hover:text-white transition"
        title="关闭 (Esc)"
      >
        <X className="w-6 h-6" />
      </button>

      {/* 下载按钮 */}
      <button
        onClick={download}
        className="absolute top-4 left-4 z-10 rounded-full bg-black/40 backdrop-blur-md p-2 text-white/80 hover:bg-black/60 hover:text-white transition"
        title="下载"
      >
        <Download className="w-6 h-6" />
      </button>

      {/* 上一张 */}
      {hasPrev && (
        <button
          onClick={(e) => { e.stopPropagation(); goPrev(); }}
          className="absolute left-4 top-1/2 -translate-y-1/2 z-10 rounded-full bg-black/40 backdrop-blur-md p-3 text-white/80 hover:bg-black/60 hover:text-white transition"
          title="上一张 (←)"
        >
          <ChevronLeft className="w-6 h-6" />
        </button>
      )}

      {/* 下一张 */}
      {hasNext && (
        <button
          onClick={(e) => { e.stopPropagation(); goNext(); }}
          className="absolute right-4 top-1/2 -translate-y-1/2 z-10 rounded-full bg-black/40 backdrop-blur-md p-3 text-white/80 hover:bg-black/60 hover:text-white transition"
          title="下一张 (→)"
        >
          <ChevronRight className="w-6 h-6" />
        </button>
      )}

      {/* 图片 */}
      <img
        src={current.dataUrl}
        alt={current.prompt || "生成图片"}
        className="max-h-[90vh] max-w-[90vw] object-contain rounded-lg animate-[scaleIn_0.2s_ease-out]"
        onClick={(e) => e.stopPropagation()}
      />

      {/* 底栏信息 + 键盘提示 */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg bg-black/60 backdrop-blur-md px-4 py-2 text-white/70 text-sm flex items-center gap-4">
        <span>{index + 1} / {images.length}</span>
        {current.seed != null && <span>Seed: {current.seed}</span>}
        <span className="text-white/40 text-xs">← → 翻页 · Esc 关闭</span>
      </div>
    </div>
  );
}
