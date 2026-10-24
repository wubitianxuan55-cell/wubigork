// cadRecent.ts — CAD 卡最近图纸（master-plan §2.1：内存态不落库，避免数据
// 膨胀）。模块级数组跨面板卸载存活（pane 关卡再开仍可一键重开）；应用重启
// 即清。去重按路径，最多 MAX 条，最近在前。

export interface CadRecent {
  path: string;
  name: string;
}

const MAX_RECENT = 5;

let recents: CadRecent[] = [];

export function cadRecentList(): CadRecent[] {
  return [...recents];
}

export function cadRecentPush(path: string, name: string): void {
  if (!path) return;
  recents = [{ path, name }, ...recents.filter((r) => r.path !== path)].slice(0, MAX_RECENT);
}

/** 测试辅助：清空（vitest 用例间隔离；模块级状态跨用例存活）。 */
export function cadRecentResetForTest(): void {
  recents = [];
}
