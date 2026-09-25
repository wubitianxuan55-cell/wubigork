// imagegenPressure.ts — 图像生成内存压力预检的前端提示（v4.409）。
// 后端在 comfyui 提交口命中低内存时发 imagegen:pressure 事件；这里全局
// 订阅一次（App 级）+2 分钟节流——绘梦/角色库/sin 全部提交口共用，连续
// 提交不刷屏。只提示不拦截，如实设预期（冷载实录见 v4.404.1）。

import { message } from 'antd'

const THROTTLE_MS = 120_000

let lastWarnAt = 0

/** 事件处理：payload { note: string }；note 空或节流窗口内则忽略。 */
export function handleImageGenPressureEvent(payload: unknown): void {
  const note = (payload as { note?: string } | null | undefined)?.note?.trim()
  if (!note) return
  const now = Date.now()
  if (now - lastWarnAt < THROTTLE_MS) return
  lastWarnAt = now
  message.warning(note)
}

/** 测试专用：重置节流基准。 */
export function resetImageGenPressureThrottleForTest(): void {
  lastWarnAt = 0
}
