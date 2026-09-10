// Composer 发送队列纯函数：pending 可编辑/删除/插话；sending = 已派出
// 等当前回合接手。发送中整列锁定（不可编辑、删除、Steer）。

export type ComposerQueueStatus = "pending" | "sending";

export interface ComposerQueueItem {
  id: string;
  text: string;
  status: ComposerQueueStatus;
}

let seq = 0;

export function makeQueueItem(text: string): ComposerQueueItem {
  seq += 1;
  return { id: `q${seq}`, text, status: "pending" };
}

export function isQueueBusy(items: readonly ComposerQueueItem[]): boolean {
  return items.some((q) => q.status === "sending");
}

export function firstPending(items: readonly ComposerQueueItem[]): ComposerQueueItem | undefined {
  return items.find((q) => q.status === "pending");
}

export function pendingItems(items: readonly ComposerQueueItem[]): ComposerQueueItem[] {
  return items.filter((q) => q.status === "pending");
}

export function markSending(items: readonly ComposerQueueItem[], id: string): ComposerQueueItem[] {
  return items.map((q) => (q.id === id ? { ...q, status: "sending" } : q));
}

export function revertSending(items: readonly ComposerQueueItem[], id: string): ComposerQueueItem[] {
  return items.map((q) => (q.id === id && q.status === "sending" ? { ...q, status: "pending" } : q));
}

export function dropSending(items: readonly ComposerQueueItem[]): ComposerQueueItem[] {
  return items.filter((q) => q.status !== "sending");
}

export function removeAt(items: readonly ComposerQueueItem[], index: number): ComposerQueueItem[] {
  if (index < 0 || index >= items.length) return [...items];
  return items.filter((_, i) => i !== index);
}

export function insertAt(items: readonly ComposerQueueItem[], index: number, item: ComposerQueueItem): ComposerQueueItem[] {
  const next = [...items];
  next.splice(Math.max(0, Math.min(index, next.length)), 0, item);
  return next;
}
