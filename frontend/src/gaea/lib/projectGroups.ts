import type { ProjectGroup } from "./types";

// 侧边栏「项目」分组的搜索过滤：命中标题/预览或路径的会话保留在所属分组内，
// 空查询返回原分组。
export function filterProjectGroups(groups: ProjectGroup[], query: string): ProjectGroup[] {
  const q = query.trim().toLowerCase();
  if (!q) return groups;
  const out: ProjectGroup[] = [];
  for (const g of groups) {
    const hits = g.sessions.filter((s) =>
      (s.title || s.preview || "").toLowerCase().includes(q) ||
      s.path.toLowerCase().includes(q),
    );
    if (hits.length) out.push({ ...g, sessions: hits });
  }
  return out;
}
