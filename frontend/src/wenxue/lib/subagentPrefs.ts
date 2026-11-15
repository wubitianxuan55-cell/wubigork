// subagentPrefs — 分工面板（v4.24 A1「子代理工作台」）的轻量用户偏好。
//
// 「新子代理自动展开」：默认关（2026-09-27 用户拍板「右侧面板启动默认隐藏，
// 不要打开办公板块就打开」——v4.24 的默认开在实机上表现为进板块/切会话右栏
// 被自动撑开，用户明确否决）。设置中心「办公工作台偏好」卡显式开启后恢复
// 派发子代理时自动亮出拓扑；关闭时出现新子代理只更新数据不抢焦点
// （不触发 onSubagentStarted 联动）。
//
// 模式对齐 lib/layoutPreferences：localStorage 直读 + try/catch 静默降级
// （隐私模式/存储禁用时不影响面板主功能），读不到或值损坏时回落默认关。
// 2026-10-04 起读写样板收敛到 lib/booleanPref 工厂，本文件只保留域语义。

import { createBooleanPref } from "./booleanPref";

const pref = createBooleanPref("gaea.subagentAutoOpen", false);

/** 读取「新子代理自动展开」偏好；未设置/损坏/无 window 时默认关。 */
export function loadSubagentAutoOpen(): boolean {
  return pref.load();
}

/** 持久化「新子代理自动展开」偏好（"1"/"0"，可读可手改）。 */
export function saveSubagentAutoOpen(enabled: boolean): void {
  pref.save(enabled);
}
