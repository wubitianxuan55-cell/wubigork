export type FileCardKind =
  | "word"
  | "excel"
  | "ppt"
  | "pdf"
  | "image"
  | "html"
  | "markdown"
  | "txt"
  | "archive"
  | "other";

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

export function fileCardKind(path: string): FileCardKind {
  const ext = extOf(path).slice(1);
  if (/^(docx?|rtf|odt|wps)$/.test(ext)) return "word";
  if (/^(xlsx?|csv|tsv|et|ods)$/.test(ext)) return "excel";
  if (/^(pptx?|dps|odp)$/.test(ext)) return "ppt";
  if (ext === "pdf") return "pdf";
  if (/^(png|jpe?g|gif|webp|svg|bmp|ico|tiff?|heic|psd)$/.test(ext)) return "image";
  if (/^(html?|htm)$/.test(ext)) return "html";
  if (/^(md|markdown)$/.test(ext)) return "markdown";
  if (/^(txt|log)$/.test(ext)) return "txt";
  if (/^(zip|7z|rar|tar|gz)$/.test(ext)) return "archive";
  return "other";
}

export function fileBadge(path: string, kind: FileCardKind): string {
  const ext = extOf(path).slice(1).toUpperCase();
  if (ext.length > 0 && ext.length <= 4) return ext;
  switch (kind) {
    case "markdown":
      return "MD";
    case "image":
      return "IMG";
    case "html":
      return "HTML";
    case "archive":
      return "ZIP";
    case "word":
      return "DOC";
    case "excel":
      return "XLS";
    case "ppt":
      return "PPT";
    case "pdf":
      return "PDF";
    case "txt":
      return "TXT";
    default:
      return "FILE";
  }
}
