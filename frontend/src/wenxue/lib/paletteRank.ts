// v4.30 命令面板按当前视图重排（Linear 式：命令菜单数百动作按当前视图重排优先级）。
//
// Why: CommandPalette 的命令项（含右栏面板项 cmd-*）原按固定构造顺序排列；
// Linear 的「按当前视图重排」先例是——用户正处在哪个视图，与其相关的命令
// 就排最前，降低检索成本。v4.72 删除「概览」tab 后主区无 tab 专属命令，
// 当前视图 = rightTab（右栏激活面板）。
//
// How: 纯函数 rankPaletteItems(items, view)——把「与当前视图相关」的命令稳定
// 移到组首，其余保持原顺序（稳定排序，不改变无关项相对位置）。命令 id 约定：
//   cmd-files/cmd-deliverables/... ↔ rightTab === 对应面板 id
// 相关度：与 rightTab 直接匹配 > 无关。同相关度按原序。
//
// 测试：lib/paletteRank.test.ts（纯函数，无 DOM）。

export interface PaletteView {
  /** 主区 tab（保留字段；当前无主区 tab 专属命令）。 */
  chatTab?: string;
  /** 右栏激活面板 id（files/deliverables/changes/tasks/subagents/browser）。 */
  rightTab?: string;
}

/** 相关度：0 = 与当前右栏面板直接匹配（最高优先）；2 = 无关。值越小越靠前。 */
function rankOf(id: string, view: PaletteView): number {
  // 右栏面板命令：cmd-<panelId>，命中当前激活面板 → 最高优先
  if (view.rightTab && id === `cmd-${view.rightTab}`) return 0;
  // 其余右栏面板命令（未激活）：次优先于无关项，便于 Ctrl+K 直达面板
  if (id.startsWith("cmd-") && !id.startsWith("cmd-sess") && !id.startsWith("cmd-tpl")) return 2;
  return 3;
}

/** 按当前视图重排命令项：稳定地把相关命令移到组首（组内相关项置顶），
 *  无关项保持原相对顺序。返回新数组（不改入参）。 */
export function rankPaletteItems<T extends { id: string }>(items: T[], view: PaletteView): T[] {
  // 稳定排序：相关度升序，同相关度保持原序（index 保序）
  return items
    .map((it, index) => ({ it, index, rank: rankOf(it.id, view) }))
    .sort((a, b) => (a.rank !== b.rank ? a.rank - b.rank : a.index - b.index))
    .map(({ it }) => it);
}
