// tasksPrefs — v4.63「新任务自动切任务视图」的轻量用户偏好。
//
// 当前会话出现新子代理/本地模型工具运行时，App 自动把右栏切到「任务」视图
// （对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂）。**默认关**
// （2026-09-27 用户拍板「右侧面板启动默认隐藏，不要打开办公板块就打开」——
// v4.63 的默认开在实机上表现为进板块/切会话右栏被自动撑开，用户明确否决）。
// 设置中心「办公工作台偏好」卡显式开启后恢复自动切换；关闭时只记角标语义，
// 不抢右栏焦点。
//
// 存储键 gaea.tasks.autoOpenSubagent 由 v4.63 落地时即存在于 App.tsx（触发
// 时机处 inline localStorage 直读，2026-09-27 起与 App 侧同口径：只认显式
// 开启值 "1"/"true" 为开，其余一律默认关）。
//
// 2026-10-04 起读写样板收敛到 lib/booleanPref 工厂，本文件只保留域语义。

import { createBooleanPref } from "./booleanPref";

const pref = createBooleanPref("gaea.tasks.autoOpenSubagent", false);

/** 读取「新任务自动切任务视图」偏好；未设置/损坏/无 window 时默认关。 */
export function loadTasksAutoOpenSubagent(): boolean {
  return pref.load();
}

/** 持久化「新任务自动切任务视图」偏好（"1"/"0"，可读可手改）。 */
export function saveTasksAutoOpenSubagent(enabled: boolean): void {
  pref.save(enabled);
}
