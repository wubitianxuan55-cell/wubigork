// 交互状态持久化：答案/锁定/字段值，按 stateKey 存取，LRU 有界。
// 与上游语义一致：只存交互值，不存正文/spec；password 值永不落库。

const STORE_KEY = "gaea.genui.interaction";
const MAX_BLOCKS = 200;

export interface BlockInteractionState {
  /** radio group → 选中项 label。 */
  answers?: Record<string, string>;
  /** submit 本地判卷后锁定。 */
  locked?: boolean;
  /** 带 id 的输入值（password 已剥离）。 */
  fields?: Record<string, string>;
}

interface StoreShape {
  order: string[];
  blocks: Record<string, BlockInteractionState>;
}

function emptyStore(): StoreShape {
  return { order: [], blocks: {} };
}

function readStore(): StoreShape {
  try {
    const raw = window.localStorage.getItem(STORE_KEY);
    if (raw === null) return emptyStore();
    const parsed = JSON.parse(raw) as Partial<StoreShape>;
    if (!Array.isArray(parsed.order) || typeof parsed.blocks !== "object" || parsed.blocks === null) {
      return emptyStore();
    }
    return {
      order: parsed.order.filter((k): k is string => typeof k === "string"),
      blocks: parsed.blocks as Record<string, BlockInteractionState>,
    };
  } catch {
    return emptyStore();
  }
}

function writeStore(store: StoreShape): void {
  try {
    window.localStorage.setItem(STORE_KEY, JSON.stringify(store));
  } catch {
    // 隐私模式/配额失败：状态只在内存中存活，不致命。
  }
}

export function loadBlockState(stateKey: string): BlockInteractionState | null {
  if (stateKey === "") return null;
  const store = readStore();
  return store.blocks[stateKey] ?? null;
}

export function saveBlockState(stateKey: string, state: BlockInteractionState): void {
  if (stateKey === "") return;
  const store = readStore();
  const order = store.order.filter((k) => k !== stateKey);
  order.unshift(stateKey);
  store.blocks[stateKey] = state;
  while (order.length > MAX_BLOCKS) {
    const evicted = order.pop();
    if (evicted !== undefined) delete store.blocks[evicted];
  }
  store.order = order;
  writeStore(store);
}

export function clearBlockState(stateKey: string): void {
  if (stateKey === "") return;
  const store = readStore();
  if (!(stateKey in store.blocks)) return;
  delete store.blocks[stateKey];
  store.order = store.order.filter((k) => k !== stateKey);
  writeStore(store);
}

export function resetInteractionStore(): void {
  try {
    window.localStorage.removeItem(STORE_KEY);
  } catch {
    // noop
  }
}
