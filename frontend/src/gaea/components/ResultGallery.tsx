// ResultGallery.tsx — 生成图片缩略图网格（Tailwind 纯 class，零内联样式）
import { Eye, Download, Copy, Trash2, Loader2, Sparkles } from "lucide-react";
import type { ImageGenResult } from "../lib/types";

interface ResultGalleryProps {
  results: ImageGenResult[];
  generating: boolean;
  onPreview: (index: number) => void;
  onReuse: (result: ImageGenResult) => void;
  onDelete?: (index: number) => void;
  onDownload: (index: number) => void;
}

function getAspectClass(size: string): string {
  if (size === "576x1024" || size === "768x1024") return "aspect-[9/16]";
  if (size === "1024x576" || size === "1280x544") return "aspect-[16/9]";
  return "aspect-square";
}

export function ResultGallery({ results, generating, onPreview, onDownload, onReuse, onDelete }: ResultGalleryProps) {
  // 生成中空状态
  if (generating && results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[300px] gap-4">
        <div className="animate-[pulse-glow_2s_ease-in-out_infinite] rounded-2xl p-4">
          <Loader2 className="w-10 h-10 text-accent animate-spin" />
        </div>
        <span className="text-sm text-fg-faint">AI 正在绘制中...</span>
      </div>
    );
  }

  // 空结果
  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[300px] gap-3">
        <Sparkles className="w-10 h-10 text-fg-faint" />
        <span className="text-sm text-fg-faint">输入描述，点击生成</span>
        <span className="text-xs text-fg-faint/60">支持中文提示词，描述越详细效果越好</span>
      </div>
    );
  }

  return (
    <div className="grid gap-3 grid-cols-[repeat(auto-fill,minmax(200px,1fr))]">
      {results.map((r, i) => (
        <div
          key={i}
          tabIndex={0}
          data-result-card
          className="relative group rounded-xl overflow-hidden border border-border bg-bg-elev cursor-pointer hover:shadow-lg transition-shadow animate-[fadeIn_0.2s_ease-out] outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2"
          onClick={() => onPreview(i)}
        >
          <img
            src={r.image}
            alt={r.prompt?.slice(0, 60) || `生成图片 #${i + 1}`}
            className={`w-full block object-cover ${getAspectClass(r.size)}`}
            loading="lazy"
          />

          {/* 右上角：生成耗时 */}
          {r.time != null && (
            <div className="absolute top-1.5 right-1.5 rounded-md px-1.5 py-0.5 text-xs text-white bg-black/60 backdrop-blur-sm">
              {r.time}s
            </div>
          )}

          {/* 底部操作栏 */}
          <div className="absolute bottom-0 left-0 right-0 bg-black/60 backdrop-blur-md px-2 py-1.5 flex items-center justify-center gap-3">
            <button
              onClick={(e) => { e.stopPropagation(); onPreview(i); }}
              className="text-white/80 hover:text-white transition"
              title="预览"
            >
              <Eye className="w-3 h-3" />
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onDownload(i); }}
              className="text-white/80 hover:text-white transition"
              title="下载"
            >
              <Download className="w-3 h-3" />
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onReuse(r); }}
              className="text-white/80 hover:text-white transition"
              title="复用提示词"
            >
              <Copy className="w-3 h-3" />
            </button>
            {onDelete && (
              <button
                onClick={(e) => { e.stopPropagation(); onDelete(i); }}
                className="text-white/80 hover:text-red-400 transition"
                title="删除"
              >
                <Trash2 className="w-3 h-3" />
              </button>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
