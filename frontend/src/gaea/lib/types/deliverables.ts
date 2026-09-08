// ── 权威产物登记表（v4.24 C1：写类工具落盘登记，替代正文扩展名白名单启发式）──
export interface DeliverableEntry {
  path: string; // 产物路径（工具参数原样，trim 后）
  tool: string; // 最近一次写入该路径的工具
  turn: number; // 最近一次写入的来源轮次（1-based；0=轮外）
  updatedAt: number; // 最近一次写入时间（日志 ts，unix 秒）
  touches: number; // 该路径累计写入次数
}

export interface DeliverableRegistryView {
  available: boolean; // false = 会话无事件日志（legacy 未迁移/路径非法），前端显示空态
  entries: DeliverableEntry[]; // 按 updatedAt 倒序、上限 200 条
  total: number; // 去重后完整登记数（可能 > len(entries)）
}
