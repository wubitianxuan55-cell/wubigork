// Session-scoped Composer drafts. Text and attachment paths survive session
// switches (and app restarts); queued sends stay in memory so a restart never
// replays a request that may already have reached the backend.

export type ComposerDraftSpace = "work" | "play";

export interface ComposerDraftAttachment {
  path: string;
  previewUrl: string;
  type: "image" | "file";
  size?: number;
}

export interface ComposerDraftQueueItem {
  id: string;
  text: string;
  status: "pending" | "sending";
}

export interface ComposerDraft {
  text: string;
  attachments: ComposerDraftAttachment[];
  queue: ComposerDraftQueueItem[];
}

const STORAGE_KEY = "gaea.composerDrafts.v1";
const EMPTY_DRAFT: ComposerDraft = { text: "", attachments: [], queue: [] };

let hydrated = false;
let drafts = new Map<string, ComposerDraft>();

function nonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value.trim() !== "";
}

function normalizeDraft(value: unknown): ComposerDraft {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return EMPTY_DRAFT;
  }
  const raw = value as Record<string, unknown>;
  const text = typeof raw.text === "string" ? raw.text : "";
  const attachments = Array.isArray(raw.attachments)
    ? raw.attachments.flatMap((item): ComposerDraftAttachment[] => {
        if (item === null || typeof item !== "object" || Array.isArray(item)) return [];
        const a = item as Record<string, unknown>;
        if (!nonEmptyString(a.path) || (a.type !== "image" && a.type !== "file")) return [];
        return [{
          path: a.path,
          previewUrl: typeof a.previewUrl === "string" ? a.previewUrl : "",
          type: a.type,
          ...(typeof a.size === "number" && Number.isFinite(a.size) && a.size >= 0 ? { size: a.size } : {}),
        }];
      }).filter((item, index, all) => all.findIndex((other) => other.path === item.path) === index)
    : [];
  const queue = Array.isArray(raw.queue)
    ? raw.queue.flatMap((item): ComposerDraftQueueItem[] => {
        if (item === null || typeof item !== "object" || Array.isArray(item)) return [];
        const q = item as Record<string, unknown>;
        if (!nonEmptyString(q.id) || !nonEmptyString(q.text)) return [];
        return [{
          id: q.id,
          text: q.text,
          // A persisted "sending" row is ambiguous after restart. Treat every
          // restored row as pending; persistence strips queue rows entirely.
          status: "pending",
        }];
      })
    : [];
  return { text, attachments, queue };
}

function hydrate(): void {
  if (hydrated || typeof window === "undefined") return;
  hydrated = true;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as unknown;
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) return;
    drafts = new Map<string, ComposerDraft>();
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (!nonEmptyString(key)) continue;
      const draft = normalizeDraft(value);
      if (draft.text || draft.attachments.length > 0) drafts.set(key, draft);
    }
  } catch {
    drafts = new Map<string, ComposerDraft>();
  }
}

function persist(): void {
  if (typeof window === "undefined") return;
  try {
    const serializable: Record<string, ComposerDraft> = {};
    for (const [key, draft] of drafts) {
      serializable[key] = {
        ...draft,
        // Image previews may be multi-megabyte data URLs. Restored image chips
        // still work through their saved path and simply omit the thumbnail.
        attachments: draft.attachments.map((item) => ({ ...item, previewUrl: "" })),
        queue: [],
      };
    }
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(serializable));
  } catch {
    // Draft persistence is best-effort; the in-memory map remains authoritative.
  }
}

export function makeComposerSessionKey(
  space: ComposerDraftSpace,
  sessionPath: string | undefined,
  cwd: string | undefined,
  newSessionNonce: number,
): string {
  const identity = sessionPath
    ? `session:${sessionPath}`
    : `new:${cwd ?? ""}:${Math.max(0, Math.floor(newSessionNonce))}`;
  return `${space}:${identity}`;
}

export function getComposerDraft(sessionKey: string | undefined): ComposerDraft {
  if (!sessionKey) return EMPTY_DRAFT;
  hydrate();
  return drafts.get(sessionKey) ?? EMPTY_DRAFT;
}

export function saveComposerDraft(sessionKey: string | undefined, value: ComposerDraft): void {
  if (!sessionKey) return;
  hydrate();
  const draft = normalizeDraft(value);
  if (!draft.text && draft.attachments.length === 0 && draft.queue.length === 0) {
    drafts.delete(sessionKey);
  } else {
    drafts.set(sessionKey, draft);
  }
  persist();
}

export function resetComposerDraftsForTests(clearStorage = true): void {
  drafts = new Map<string, ComposerDraft>();
  hydrated = false;
  if (clearStorage && typeof window !== "undefined") {
    window.localStorage.removeItem(STORAGE_KEY);
  }
}
