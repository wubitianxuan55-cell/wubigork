// cadExt — CAD 图纸扩展名判定（叶模块，零依赖）。
//
// Why: paneTabs 的快照净化需要判「dwg/dxf 文件 tab 是坏预览」（2026-11-14 起
// 这类文件统一落右栏 CAD 面板），而 lib/cadOpen.ts 依赖 paneTabs（openView）
// ——再从 cadOpen 反向导入会成环；把扩展名判定抽成叶模块，cadOpen 与 paneTabs
// 各自从这里取，公共 API 由 cadOpen 再导出保持不变。

/** 右栏 CAD 面板承接的图纸扩展名（pdf 走既有文档预览链，不在此列）。 */
export const CAD_FILE_EXTS = ["dwg", "dxf"] as const;

function extOf(name: string): string {
  const i = name.lastIndexOf(".");
  return i >= 0 ? name.slice(i + 1).toLowerCase() : "";
}

/** 该路径是否为右栏 CAD 面板承接的图纸文件。 */
export function isCadFilePath(rel: string): boolean {
  const name = rel.split(/[\\/]/).pop() || rel;
  return (CAD_FILE_EXTS as readonly string[]).includes(extOf(name));
}
