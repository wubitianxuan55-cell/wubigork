/**
 * saveFile.ts — 导出文件保存中立层（瘦身 P1 线1「双门+字节收口刀」③）
 *
 * Wails 壳内 <a download> 不落盘（v4.162 实证）→ 壳内走 GaeaSaveFileAs
 * 系统「另存为」对话框写入；浏览器回退 <a download> 原机制。实现体自
 * schedule/exportArtifact.ts 迁入（schedule 侧 re-export 保既有 import 零
 * 改动），供任意域（图生图下载、小说设定导出…）复用，防 schedule 域依赖
 * 倒挂。壳判定唯一规范 = pickFile.inShellEnv。本模块只 import ./bridge、
 * ./pickFile、./bytes（中立层内零倒挂）。
 */
import { app } from './bridge'
import { inShellEnv } from './pickFile'
import { b64ToBytes } from './bytes'

/** 触发浏览器下载（与导出 XML 同一 <a download> 机制） */
export function downloadBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}

/**
 * data URL → Blob（壳内另存为前置转换）：mime 取 dataURL 前缀（未知降级
 * application/octet-stream），负载经 b64ToBytes 精确还原。
 */
export function dataUrlToBlob(dataUrl: string): Blob {
  const comma = dataUrl.indexOf(',')
  const b64 = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl
  const mime = /^data:([^;,]+)/.exec(dataUrl)?.[1] ?? 'application/octet-stream'
  return new Blob([b64ToBytes(b64)], { type: mime })
}

/**
 * 保存导出文件（双门）：壳内走系统「另存为」（GaeaSaveFileAs，返回实际
 * 写入路径；用户取消返回空串）；浏览器回退 <a download>。
 * 返回 true=已保存，false=用户取消。
 */
export async function saveExportBlob(blob: Blob, name: string): Promise<boolean> {
  if (inShellEnv()) {
    const b64 = await blobToB64(blob)
    const path = await app.SaveFileAs(name, b64)
    return path !== ''
  }
  downloadBlob(blob, name)
  return true
}

/** Blob → base64（FileReader dataURL，去掉 dataURL 头部） */
function blobToB64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const fr = new FileReader()
    fr.onload = () => resolve(String(fr.result).slice(String(fr.result).indexOf(',') + 1))
    fr.onerror = () => reject(new Error('读取导出内容失败'))
    fr.readAsDataURL(blob)
  })
}
