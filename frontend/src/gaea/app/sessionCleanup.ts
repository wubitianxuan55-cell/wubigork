// 会话删除成功后的 GenUI 状态清理（审计 2026-09 #7）：localStorage 交互状态
// 按 stateKey 前缀清理，内存面板内容按会话键清除。幂等；只应在删除成功路径
// 调用（删除失败时保留现场，交互状态丢失只会退化为干净默认渲染）。
import { clearBlockStatesForSession } from "../../genui/interaction";
import { clearGenuiPanel, sanitizeSessionKey } from "../lib/genuiPanel";

export function purgeDeletedSessionGenui(path: string): void {
  clearBlockStatesForSession(sanitizeSessionKey(path));
  clearGenuiPanel(path);
}
