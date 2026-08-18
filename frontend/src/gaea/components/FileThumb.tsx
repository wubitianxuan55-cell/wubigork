import { memo, useEffect, useState } from "react";
import { File, FileImage, FilePpt, FileSpreadsheet, FileText } from "../icons";
import { app } from "../lib/bridge";
import type { XlsxPreview } from "../lib/types";

// 交付物图片扩展名：命中后优先渲染缩略图，其余回退类型图标。
export const IMAGE_EXT_RE = /\.(png|jpe?g|gif|webp|svg|bmp|ico)$/i;

export function FileTypeIcon({ ext, size }: { ext: string; size: number }) {
  if (/\.(xlsx?|csv|et|ods)$/i.test(ext)) return <FileSpreadsheet size={size} />;
  if (/\.(pptx?|dps|odp)$/i.test(ext)) return <FilePpt size={size} />;
  if (IMAGE_EXT_RE.test(ext)) return <FileImage size={size} />;
  if (/\.(docx?|pdf|md|markdown|txt|odt|rtf|wps|ofd|html?)$/i.test(ext)) return <FileText size={size} />;
  return <File size={size} />;
}

// 迷你内容缩略图（P1，对标豆包 PPT 卡片 / AutoGPT MIME 插图）：产物列表里
// 「看得见内容」比「看得见图标」更接近成果。
// - xlsx/csv/et/ods → 渲染前 3 行 × 前 3 列的迷你表格（复用 GaeaPreview 结构化数据）
// - md/txt → 渲染前几行文本摘要
// - 图片 → 保持 dataURL 缩略图；其余回退类型图标；任何加载失败静默回退类型图标。
export const FileThumb = memo(function FileThumb({
  path,
  ext,
  imgClassName,
}: {
  path: string;
  ext: string;
  imgClassName?: string;
}) {
  const [dataUrl, setDataUrl] = useState<string | null>(null);
  const [grid, setGrid] = useState<string[][] | null>(null);
  const [textLines, setTextLines] = useState<string[] | null>(null);

  useEffect(() => {
    let live = true;
    setDataUrl(null);
    setGrid(null);
    setTextLines(null);
    if (IMAGE_EXT_RE.test(ext)) {
      app.AttachmentDataURL(path).then((url) => { if (live) setDataUrl(url); }).catch(() => {});
    } else if (/\.(xlsx?|csv|et|ods)$/i.test(ext)) {
      // 复用 GaeaPreview 的结构化单元格 JSON：取首表前 3 行 × 前 3 列
      app.Preview(path).then((p) => {
        if (!live || p.kind !== "xlsx" || !p.body) return;
        try {
          const parsed = JSON.parse(p.body) as XlsxPreview;
          const rows = parsed.sheets?.[0]?.rows ?? [];
          const sample = rows.slice(0, 3).map((r) => r.slice(0, 3).map((c) => c.value ?? ""));
          if (sample.length > 0) setGrid(sample);
        } catch { /* 回退类型图标 */ }
      }).catch(() => {});
    } else if (/\.(md|markdown|txt|log|json|toml|yaml|yml|csv|tsv)$/i.test(ext)) {
      app.Preview(path).then((p) => {
        if (!live || (p.kind !== "markdown" && p.kind !== "text") || !p.body) return;
        const lines = p.body.split("\n").map((l) => l.trim()).filter(Boolean).slice(0, 4);
        if (lines.length > 0) setTextLines(lines);
      }).catch(() => {});
    }
    return () => { live = false; };
  }, [path, ext]);

  if (dataUrl) {
    return (
      <img
        src={dataUrl}
        alt=""
        className={imgClassName ?? "w-10 h-8 object-cover rounded-[5px] border border-border-soft bg-bg"}
      />
    );
  }
  if (grid) {
    return (
      <div
        className="grid gap-px p-0.5 overflow-hidden rounded-[5px] border border-border-soft bg-bg"
        style={{ gridTemplateColumns: `repeat(${grid[0].length}, minmax(0, 1fr))`, width: "100%", height: "100%" }}
        aria-label="表格内容缩略图"
      >
        {grid.flat().map((cell, i) => (
          <span
            key={i}
            className="truncate text-[7px] leading-[1.15] px-0.5 py-px text-(color:--md-sys-color-text-secondary)"
            style={{ background: "color-mix(in srgb, var(--md-sys-color-outline-variant) 30%, transparent)" }}
            title={cell}
          >
            {cell || " "}
          </span>
        ))}
      </div>
    );
  }
  if (textLines) {
    return (
      <div
        className="flex flex-col gap-0.5 p-1 overflow-hidden rounded-[5px] border border-border-soft bg-bg"
        aria-label="文本内容缩略图"
      >
        {textLines.map((line, i) => (
          <span key={i} className="truncate text-[7px] leading-[1.2] text-(color:--md-sys-color-text-secondary)">
            {line}
          </span>
        ))}
      </div>
    );
  }
  return <FileTypeIcon ext={ext} size={14} />;
});
