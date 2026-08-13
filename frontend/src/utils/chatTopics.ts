/** 聊天会话列表的纯函数工具（搜索过滤 / 最近活跃排序）。 */

/** 会话列表搜索：命中标题、预览或模式标签，大小写不敏感；空查询返回原列表。 */
export function filterChatTopics<T extends { title: string; preview?: string; modeLabel?: string }>(
  topics: T[],
  query: string,
): T[] {
  const q = query.trim().toLowerCase();
  if (!q) return topics;
  return topics.filter(
    (t) =>
      (t.title || "").toLowerCase().includes(q) ||
      (t.preview || "").toLowerCase().includes(q) ||
      (t.modeLabel || "").toLowerCase().includes(q),
  );
}

/** 最近活跃优先（按 key 取出的时间戳降序），稳定排序，不修改原数组。 */
export function sortByUpdatedAtDesc<T>(topics: T[], key: (t: T) => number): T[] {
  return [...topics].sort((a, b) => key(b) - key(a));
}

/** 用首条用户消息生成会话标题：超过 20 字截断并加省略号。 */
export function autoTopicTitle(text: string): string {
  return text.length > 20 ? `${text.slice(0, 20)}…` : text;
}
