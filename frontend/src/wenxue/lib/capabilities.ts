// 能力面板（MCP / Skills）的纯字符串摘要函数——板块无关，直接复用办公实现。
// 2026-11-26 同步审计：此前是分叉态拷贝，办公 11-17 改成按显示宽度截断后未镜像
// （中文摘要按字符数截、偏长）；改转发后两板块不再漂移。
export { summarizeServerError, summarizeSkillDescription } from "../../gaea/lib/capabilities";
