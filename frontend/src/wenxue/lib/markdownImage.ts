/** Resolve one Markdown image URL into a local workspace path, or null for direct/data URLs. */
export function resolveMarkdownImagePath(src: string, basePath?: string): string | null {
  let value = String(src || "").trim();
  if (!value) return null;
  if (/^(data:image\/|https?:\/\/|blob:)/i.test(value)) return null;
  if (/^data:/i.test(value)) return null;
  value = value.replace(/[?#].*$/, "");
  try {
    value = decodeURIComponent(value);
  } catch {
    // Keep malformed percent encoding literal; the Go read path will fail visibly.
  }

  if (/^file:/i.test(value)) {
    const raw = value.replace(/^file:\/\//i, "");
    let p = raw.startsWith("/") ? raw : `//${raw}`;
    if (/^\/\/localhost\//i.test(p)) p = p.slice("//localhost".length);
    if (/^\/localhost\//i.test(p)) p = p.slice("/localhost".length);
    if (/^\/[A-Za-z]:\//.test(p)) p = p.slice(1);
    return normalizePath(p);
  }
  if (/^[A-Za-z]:[\\/]/.test(value) || /^\/\//.test(value)) return normalizePath(value);
  if (/^[A-Za-z][A-Za-z0-9+.-]*:/.test(value)) return null;

  const relative = value.replace(/^[\\/]+/, "");
  const dir = (basePath ?? "").replace(/\\/g, "/").replace(/\/[^/]*$/, "");
  return normalizePath(dir ? `${dir}/${relative}` : relative);
}

function normalizePath(path: string): string {
  const slash = path.replace(/\\/g, "/");
  const unc = slash.startsWith("//");
  const drive = slash.match(/^[A-Za-z]:/)?.[0] ?? "";
  const absolute = drive !== "" || slash.startsWith("/");
  let body = drive ? slash.slice(drive.length) : slash;
  const parts: string[] = [];
  for (const part of body.split("/")) {
    if (!part || part === ".") continue;
    if (part === "..") {
      if (parts.length > 0 && parts[parts.length - 1] !== "..") parts.pop();
      else if (!absolute) parts.push("..");
      continue;
    }
    parts.push(part);
  }
  body = parts.join("/");
  if (drive) return `${drive}/${body}`;
  if (unc) return `//${body}`;
  if (absolute) return `/${body}`;
  return body;
}
