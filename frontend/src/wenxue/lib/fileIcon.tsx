// FileTypeIcon — 文件类型图标（按扩展名着色，办公文件优先）。
//
// Why: 文件树行与编辑器 tab 需要同一套文件类型图标语言（Codex 式：tab 上
// 也显示文件类型小图标）。此前 fileIcon 内联在 FileTree，编辑器 tab 无图标；
// v4.27 抽到 lib 共享，树与 tab 单源一致，避免两处各自维护映射漂移。
import {
  Archive,
  Command,
  Database,
  File,
  FileImage,
  FilePdf,
  FilePpt,
  FileSpreadsheet,
  FileText,
  FileWord,
  Folder,
  FolderOpen,
  Settings,
} from "../icons";

const WORD_EXTS = new Set(["doc", "docx", "rtf", "odt"]);
const SHEET_EXTS = new Set(["xls", "xlsx", "csv", "tsv", "ods"]);
const SLIDE_EXTS = new Set(["ppt", "pptx", "odp"]);
const IMAGE_EXTS = new Set(["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg", "ico", "heic", "avif"]);
const CODE_EXTS = new Set([
  "js", "mjs", "cjs", "ts", "tsx", "jsx", "go", "py", "rs", "java", "kt", "kts",
  "c", "h", "hpp", "cpp", "cc", "cs", "php", "rb", "swift", "scala", "sh",
  "bash", "ps1", "bat", "cmd", "lua", "vue", "svelte", "html", "htm", "css",
  "scss", "less",
]);
const CONFIG_EXTS = new Set(["json", "jsonc", "jsonl", "ndjson", "yaml", "yml", "toml", "ini", "xml", "env", "conf"]);
const ARCHIVE_EXTS = new Set(["zip", "7z", "rar", "tar", "gz", "bz2", "xz", "zst"]);
const DATABASE_EXTS = new Set(["db", "sqlite", "sqlite3", "mdb", "sql"]);
const TEXT_EXTS = new Set(["md", "markdown", "txt", "log"]);

/** 文件类型图标：目录区分开合；文件按办公/代码/配置/压缩/数据库类型着色。 */
export function FileTypeIcon({
  name,
  isDir = false,
  open = false,
  size = 14,
  className = "",
}: {
  name: string;
  isDir?: boolean;
  open?: boolean;
  size?: number;
  className?: string;
}) {
  const hidden = { "aria-hidden": true as const };
  if (isDir) {
    const FolderIcon = open ? FolderOpen : Folder;
    return <FolderIcon size={size} className={`text-accent shrink-0 ${className}`} data-file-kind={open ? "folder-open" : "folder"} {...hidden} />;
  }
  const lower = name.toLowerCase();
  const ext = lower.includes(".") ? lower.split(".").pop() ?? "" : "";
  if (WORD_EXTS.has(ext))
    return <FileWord size={size} className={`text-sky-400 shrink-0 ${className}`} data-file-kind="word" {...hidden} />;
  if (SHEET_EXTS.has(ext))
    return <FileSpreadsheet size={size} className={`text-emerald-400 shrink-0 ${className}`} data-file-kind="sheet" {...hidden} />;
  if (SLIDE_EXTS.has(ext))
    return <FilePpt size={size} className={`text-orange-400 shrink-0 ${className}`} data-file-kind="slides" {...hidden} />;
  if (ext === "pdf")
    return <FilePdf size={size} className={`text-red-400 shrink-0 ${className}`} data-file-kind="pdf" {...hidden} />;
  if (IMAGE_EXTS.has(ext))
    return <FileImage size={size} className={`text-violet-400 shrink-0 ${className}`} data-file-kind="image" {...hidden} />;
  if (CODE_EXTS.has(ext))
    return <Command size={size} className={`text-cyan-400 shrink-0 ${className}`} data-file-kind="code" {...hidden} />;
  if (CONFIG_EXTS.has(ext) || lower === ".env")
    return <Settings size={size} className={`text-amber-400 shrink-0 ${className}`} data-file-kind="config" {...hidden} />;
  if (ARCHIVE_EXTS.has(ext))
    return <Archive size={size} className={`text-yellow-500 shrink-0 ${className}`} data-file-kind="archive" {...hidden} />;
  if (DATABASE_EXTS.has(ext) || lower === "sqlite.db")
    return <Database size={size} className={`text-indigo-400 shrink-0 ${className}`} data-file-kind="database" {...hidden} />;
  if (TEXT_EXTS.has(ext) || ["readme", "license", "makefile", "dockerfile"].includes(lower))
    return <FileText size={size} className={`text-fg-dim shrink-0 ${className}`} data-file-kind="text" {...hidden} />;
  return <File size={size} className={`text-fg-faint shrink-0 ${className}`} data-file-kind="file" {...hidden} />;
}
