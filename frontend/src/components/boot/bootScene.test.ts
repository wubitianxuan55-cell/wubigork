import { describe, expect, it } from 'vitest'
import {
  BOOT_FINISH_S,
  BOOT_MOTION_FULL,
  BOOT_MOTION_REDUCED,
  BOOT_SCENE_AT_S,
  EXIT_MS,
  REDUCED_EXIT_MS,
  SKIP_EXIT_MS,
  bootHudAt,
  bootPose,
  bootProgressAt,
} from './bootScene'

const poseAt = (t: number, exit = 0) => bootPose(t, exit, BOOT_MOTION_FULL)
const stillAt = (t: number, exit = 0) => bootPose(t, exit, BOOT_MOTION_REDUCED)
const between = (from: number, to: number, step = 0.01) => {
  const out: number[] = []
  for (let t = from; t <= to + 1e-9; t += step) out.push(Number(t.toFixed(4)))
  return out
}

describe('bootScene 时间轴', () => {
  it('保留语音仪式：换景点、收束点与三档退出时长不变', () => {
    expect(BOOT_SCENE_AT_S).toEqual([0, 1.596])
    expect(BOOT_FINISH_S).toBe(4)
    expect([EXIT_MS, SKIP_EXIT_MS, REDUCED_EXIT_MS]).toEqual([520, 420, 180])
  })

  it('俯冲前停在星际幕，穿日到达人间幕，全片只白闪一次', () => {
    expect(poseAt(0).scene).toBe(0)
    expect(poseAt(1.6).scene).toBe(0)
    expect(poseAt(2.45).scene).toBeCloseTo(1, 5)
    expect(poseAt(4).scene).toBe(1)

    const scenes = between(1.6, 2.45).map((t) => poseAt(t).scene)
    expect(scenes.every((v, i) => i === 0 || v >= scenes[i - 1])).toBe(true)

    const flash = between(1.6, 2.45).map((t) => poseAt(t).flash)
    expect(poseAt(1.6).flash).toBeLessThan(0.001)
    expect(poseAt(2.45).flash).toBeLessThan(0.001)
    expect(Math.max(...flash)).toBeGreaterThan(0.9)
    // one rise and one fall: no strobing
    const rising = flash.map((v, i) => i > 0 && v > flash[i - 1])
    const falling = flash.map((v, i) => i > 0 && v < flash[i - 1])
    expect(rising.indexOf(true)).toBeGreaterThan(-1)
    expect(falling.lastIndexOf(true)).toBe(flash.length - 1)
    expect(falling.indexOf(true)).toBeGreaterThan(rising.lastIndexOf(true))
  })

  it('镜头只在推进与俯冲时变化，停稳后人间幕缓慢漂移', () => {
    expect(poseAt(0).cam1[0]).toBe(1)
    expect(poseAt(1.6).cam1[0]).toBeGreaterThan(1)
    expect(poseAt(0.5).cam2[0]).toBe(poseAt(0).cam2[0])
    expect(poseAt(2.45).cam2[0]).toBeGreaterThan(poseAt(3.0).cam2[0])
    expect(poseAt(4).cam2[0]).toBeGreaterThan(poseAt(3.0).cam2[0])
  })

  it('进度沿 68 → 72 → 100 的节奏推进且永不回退', () => {
    const values = between(0, 4).map((t) => bootProgressAt(t, 0))
    expect(values.every((v, i) => i === 0 || v >= values[i - 1])).toBe(true)
    expect(Math.round(bootProgressAt(1.1, 0))).toBe(68)
    expect(Math.round(bootProgressAt(2.3, 0))).toBe(72)
    expect(bootProgressAt(4, 0)).toBe(94)
    const leaving = between(0, 1).map((e) => bootProgressAt(3, e))
    expect(leaving.every((v, i) => i === 0 || v >= leaving[i - 1])).toBe(true)
    expect(bootProgressAt(3, 1)).toBe(100)
    expect(bootProgressAt(0, 0)).toBe(0)
  })

  it('减弱动效保留同样的时序，但不推镜头、不白闪、不闪烁', () => {
    expect(stillAt(1.2).cam1[0]).toBe(1)
    expect(stillAt(2.2).cam1[0]).toBe(1)
    expect(stillAt(2.2).cam2[0]).toBe(1)
    expect(stillAt(2.04).flash).toBe(0)
    expect(stillAt(1.3).amb).toBe(0)
    expect(stillAt(1.5).scene).toBe(0)
    expect(stillAt(1.8).scene).toBe(1)
    // fades still carry the state change
    expect(stillAt(1.3).logoA).toBe(1)
    expect(stillAt(0).logoA).toBe(0)
  })

  it('离场时标与轨道环先退，画面随后交给工作台', () => {
    expect(poseAt(3, 0).logoA).toBe(1)
    expect(poseAt(3, 0).ringA).toBeGreaterThan(0.99)
    expect(poseAt(3, 0.6).logoA).toBe(0)
    expect(poseAt(3, 1).ringA).toBe(0)
    expect(poseAt(3, 1).logoA).toBe(0)
  })

  it('HUD 文案按序出现：角标 → 字标收字距 → 标语一 → 标语二', () => {
    expect(bootHudAt(0, 0).chrome).toBe(0)
    expect(bootHudAt(0.6, 0).chrome).toBe(1)
    expect(bootHudAt(0.5, 0).wordmark).toBe(0)
    expect(bootHudAt(0.56, 0).track).toBeGreaterThan(0.9)
    expect(bootHudAt(1.4, 0).wordmark).toBe(1)
    expect(bootHudAt(1.4, 0).track).toBeCloseTo(0, 5)
    expect(bootHudAt(1.2, 0).tagline1).toBeGreaterThan(0.5)
    expect(bootHudAt(1.9, 0).tagline1).toBe(0)
    expect(bootHudAt(1.9, 0).tagline2).toBe(0)
    expect(bootHudAt(2.8, 0).tagline2).toBe(1)
    expect(bootHudAt(3, 1).chrome).toBe(0)
    expect(bootHudAt(3, 1).wordmark).toBe(0)
    expect(bootHudAt(3, 1).tagline2).toBe(0)
  })
})
