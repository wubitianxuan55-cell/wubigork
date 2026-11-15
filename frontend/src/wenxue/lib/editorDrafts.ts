import { Modal } from "antd";
import { normalizePreviewPath } from "./officeTurnProjection";

const owners = new Map<string, Set<symbol>>();
let beforeUnloadBound = false;

function onBeforeUnload(event: BeforeUnloadEvent): void {
  if (owners.size === 0) return;
  event.preventDefault();
  event.returnValue = "";
}

function syncBeforeUnload(): void {
  if (typeof window === "undefined") return;
  if (owners.size > 0 && !beforeUnloadBound) {
    window.addEventListener("beforeunload", onBeforeUnload);
    beforeUnloadBound = true;
  } else if (owners.size === 0 && beforeUnloadBound) {
    window.removeEventListener("beforeunload", onBeforeUnload);
    beforeUnloadBound = false;
  }
}

export function registerEditorDraft(path: string, owner: symbol): void {
  const key = normalizePreviewPath(path);
  if (!key) return;
  const set = owners.get(key) ?? new Set<symbol>();
  set.add(owner);
  owners.set(key, set);
  syncBeforeUnload();
}

export function unregisterEditorDraft(path: string, owner: symbol): void {
  const key = normalizePreviewPath(path);
  const set = owners.get(key);
  if (!set) return;
  set.delete(owner);
  if (set.size === 0) owners.delete(key);
  syncBeforeUnload();
}

export function isEditorDirty(path: string): boolean {
  return owners.has(normalizePreviewPath(path));
}

export function hasDirtyEditor(): boolean {
  return owners.size > 0;
}

/** Guard a destructive navigation/close with one explicit discard confirmation. */
export function confirmDiscardEditor(path: string | null | undefined, action: () => void): void {
  if (!path || !isEditorDirty(path)) {
    action();
    return;
  }
  Modal.confirm({
    title: "有未保存的修改",
    content: "继续将关闭当前编辑页并丢弃草稿。需要保留时请先取消并保存。",
    okText: "放弃修改",
    cancelText: "继续编辑",
    okButtonProps: { danger: true },
    centered: true,
    onOk: action,
  });
}
