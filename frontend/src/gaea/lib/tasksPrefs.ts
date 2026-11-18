// tasksPrefs — v4.63「新任务自动切任务视图」的轻量用户偏好。
//
// 当前会话出现新子代理/本地模型工具运行时，主区自动切「编队」视图（对标
// dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂）。
//
// 默认值历史：v4.63 默认开 → 2026-09-27 用户拍板改关（实机表现为进板块/
// 切会话右栏被自动撑开）→ **2026-11-18 子代理实弹反馈「前端看不到子代理卡」
// 翻回默认开**。当初反感的是「打开板块就撑开」；触发面如今已被基线机制
// 收敛为「新任务/子代理首次出现」才切（≥1240px + 可关），打开板块本身
// 不触发。设置中心仍可显式关。
//
// 存储键 gaea.tasks.autoOpenSubagent 由 v4.63 落地时即存在于 App.tsx（触发
// 时机处 inline localStorage 直读，2026-09-27 起与 App 侧同口径：只认显式
// 开启值 "1"/"true" 为开，其余一律默认关）。
//
// 2026-10-04 起读写样板收敛到 lib/booleanPref 工厂，本文件只保留域语义。

import { createBooleanPref } from "./booleanPref";

const pref = createBooleanPref("gaea.tasks.autoOpenSubagent", true);

/** 读取「新任务自动切任务视图」偏好；未设置/损坏/无 window 时默认开（实弹修复）。 */
export function loadTasksAutoOpenSubagent(): boolean {
  return pref.load();
}

/** 持久化「新任务自动切任务视图」偏好（"1"/"0"，可读可手改）。 */
export function saveTasksAutoOpenSubagent(enabled: boolean): void {
  pref.save(enabled);
}
