// deliverableStatus — 产物验收状态机数据层（A1 交付验收闭环，调研 A#9/O#10）。
//
// 语义（对标 M365 Copilot 交付确认 / Devin Approve）：
//   open     待查看（默认）——agent 产出/新版本落盘后的初始态；
//   confirmed已验收——用户明确确认，交付闭环完成；
//   redo     要求修改——用户打回，回到 agent 继续加工。
// 新版本重置：登记表 updatedAt（unix 秒）大于标记时刻记录的版本戳 → 视为
// 「新版本落盘」，confirmed/redo 一律回到 open（旧验收不覆盖新内容）。
//
// 持久化：localStorage（本地优先、单用户；与 loadDeliverableAutoOpen 同模式），
// 键 gaea.deliverableAcceptance.v1，值为 map：`${sessionKey}::${pathKey}` → 记录。
// 纯函数（acceptanceOf/setAcceptance/acceptanceResetOnNewVersion）可单测；
// load/save 是薄 IO 壳。

export type DeliverableAcceptance = "open" | "confirmed" | "redo";

export interface DeliverableStatusRec {
  status: Exclude<DeliverableAcceptance, "open">; // open 是缺省态，不落存储
  at: number; // 标记时刻（unix ms）
  versionAt: number; // 标记时所见登记表 updatedAt（unix 秒，0=未知）
}

export type DeliverableStatusMap = Record<string, DeliverableStatusRec>;

const STORAGE_KEY = "gaea.deliverableAcceptance.v1";

/** 状态键：会话路径 + 产物路径归一（反斜杠→/、小写，与交付卡去重键同口径）。 */
export function statusKeyOf(sessionPath: string, path: string): string {
  const sp = sessionPath.replace(/\\/g, "/").toLowerCase();
  const pp = path.replace(/\\/g, "/").toLowerCase();
  return `${sp}::${pp}`;
}

/**
 * 读验收状态。currentUpdatedAt 传登记表最新 updatedAt（unix 秒）：
 * 已标记产物出现更新版本（updatedAt > versionAt）→ 重置为 open；
 * versionAt=0（标记时未知版本）不重置——宁保持旧判断，勿误回待查看。
 */
export function acceptanceOf(
  map: DeliverableStatusMap,
  sessionPath: string,
  path: string,
  currentUpdatedAt?: number,
): DeliverableAcceptance {
  const rec = map[statusKeyOf(sessionPath, path)];
  if (!rec) return "open";
  if (currentUpdatedAt && rec.versionAt > 0 && currentUpdatedAt > rec.versionAt) return "open";
  return rec.status;
}

/** 标记：返回新 map（不改入参）；status=open 等价于清除记录。 */
export function setAcceptance(
  map: DeliverableStatusMap,
  sessionPath: string,
  path: string,
  status: DeliverableAcceptance,
  now: number,
  versionAt = 0,
): DeliverableStatusMap {
  const key = statusKeyOf(sessionPath, path);
  const next: DeliverableStatusMap = { ...map };
  if (status === "open") {
    delete next[key];
    return next;
  }
  next[key] = { status, at: now, versionAt };
  return next;
}

/** 头部汇总计数：paths 中各产物当前验收态各归其位（total=paths.length）。
 *  currentUpdatedAtOf 提供各路径登记表最新 updatedAt（unix 秒；缺省视为未知，
 *  不触发新版本重置）——与逐行 acceptanceOf 同口径，新版本回落后不再计入。 */
export function acceptanceSummary(
  map: DeliverableStatusMap,
  sessionPath: string,
  paths: readonly string[],
  currentUpdatedAtOf?: (path: string) => number | undefined,
): { total: number; confirmed: number; redo: number } {
  let confirmed = 0;
  let redo = 0;
  for (const p of paths) {
    const status = acceptanceOf(map, sessionPath, p, currentUpdatedAtOf?.(p));
    if (status === "confirmed") confirmed += 1;
    else if (status === "redo") redo += 1;
  }
  return { total: paths.length, confirmed, redo };
}

// ── 薄 IO 壳（浏览器 localStorage；异常静默返回缺省）──────────────────

export function loadAcceptanceMap(): DeliverableStatusMap {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as DeliverableStatusMap;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

export function saveAcceptanceMap(map: DeliverableStatusMap): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch {
    /* 私密模式/配额满：内存态仍本会话有效 */
  }
}
