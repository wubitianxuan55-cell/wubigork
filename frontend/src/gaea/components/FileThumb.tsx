import { memo, useEffect, useState } from "react";
import { File, FileImage, FilePpt, FileSpreadsheet, FileText } from "../icons";
import { app } from "../lib/bridge";

// 交付物图片扩展名：命中后优先渲染缩略图，其余回退类型图标。
export const IMAGE_EXT_RE = /\.(png|jpe?g|gif|webp|svg|bmp|ico)$/i;

export function FileTypeIcon({ ext, size }: { ext: string; size: number }) {
  if (/\.(xlsx?|csv|et|ods)$/i.test(ext)) return <FileSpreadsheet size={size} />;
  if (/\.(pptx?|dps|odp)$/i.test(ext)) return <FilePpt size={size} />;
  if (IMAGE_EXT_RE.test(ext)) return <FileImage size={size} />;
  if (/\.(docx?|pdf|md|markdown|txt|odt|rtf|wps|ofd|html?)$/i.test(ext)) return <FileText size={size} />;
  return <File size={size} />;
}

// FileThumb 加载本地图片的 data URL；加载失败或非图片时回退类型图标。
// 供对话内交付卡与右侧「会话产物」面板复用，保证两处缩略图表现一致。
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
  useEffect(() => {
    let live = true;
    setDataUrl(null);
    app.AttachmentDataURL(path).then((url) => { if (live) setDataUrl(url); }).catch(() => {});
    return () => { live = false; };
  }, [path]);
  if (dataUrl) {
    return (
      <img
        src={dataUrl}
        alt=""
        className={imgClassName ?? "w-10 h-8 object-cover rounded-[5px] border border-border-soft bg-bg"}
      />
    );
  }
  return <FileTypeIcon ext={ext} size={14} />;
});
