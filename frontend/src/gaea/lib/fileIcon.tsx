// FileTypeIcon — 文件类型图标（按扩展名着色，办公文件优先）。
//
// Why: 文件树行与编辑器 tab 需要同一套文件类型图标语言（Codex 式：tab 上
// 也显示文件类型小图标）。此前 fileIcon 内联在 FileTree，编辑器 tab 无图标；
// v4.27 抽到 lib 共享，树与 tab 单源一致，避免两处各自维护映射漂移。
import { File, FileText, Folder, Image } from "../icons";

/** 文件类型图标：isDir 渲染文件夹；按扩展名给办公/图片/代码文件着色。 */
export function FileTypeIcon({
  name,
  isDir = false,
  size = 14,
  className = "",
}: {
  name: string;
  isDir?: boolean;
  size?: number;
  className?: string;
}) {
  const hidden = { "aria-hidden": true as const };
  if (isDir) return <Folder size={size} className={`text-accent shrink-0 ${className}`} {...hidden} />;
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  if (["doc", "docx"].includes(ext))
    return <FileText size={size} className={`text-sky-400 shrink-0 ${className}`} {...hidden} />;
  if (["xls", "xlsx", "csv"].includes(ext))
    return <FileText size={size} className={`text-emerald-400 shrink-0 ${className}`} {...hidden} />;
  if (["ppt", "pptx"].includes(ext))
    return <FileText size={size} className={`text-orange-400 shrink-0 ${className}`} {...hidden} />;
  if (ext === "pdf")
    return <FileText size={size} className={`text-red-400 shrink-0 ${className}`} {...hidden} />;
  if (["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg"].includes(ext))
    return <Image size={size} className={`text-violet-400 shrink-0 ${className}`} {...hidden} />;
  if (["md", "txt", "json", "toml", "yaml", "yml", "xml", "html", "css", "js", "ts", "tsx", "jsx", "go", "py"].includes(ext))
    return <FileText size={size} className={`text-fg-dim shrink-0 ${className}`} {...hidden} />;
  return <File size={size} className={`text-fg-faint shrink-0 ${className}`} {...hidden} />;
}
