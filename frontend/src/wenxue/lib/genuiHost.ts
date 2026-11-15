// 办公面板/消息区的 GenUI action 全局宿主（App 注册，面板 Tab 复用同一出口）。
import type { GenuiActionHandler } from "../../genui/GenuiActionContext";

let actionHandler: GenuiActionHandler | undefined;

export function setGenuiActionHandler(handler: GenuiActionHandler | undefined): void {
  actionHandler = handler;
}

export function getGenuiActionHandler(): GenuiActionHandler | undefined {
  return actionHandler;
}
