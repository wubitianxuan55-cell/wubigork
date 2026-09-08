/**
 * pickFile.ts — Wails 壳内文件选取共享 util（审计刀A，v4.162 实证背景）
 *
 * Wails 壳内 <input type=file>.click() 不弹文件对话框（浏览器一切正常），
 * 壳内选取必须走 GaeaPickFiles 系统对话框 + GaeaReadFileB64 读内容还原 File。
 * 本模块是中立基础层：只许 import ./bridge（types 经 bridge 返回类型带入）
 * 与同级零依赖 util（./bytes），供任意域（角色库参考图、进度计划导入…）
 * 复用，防 schedule 域依赖倒挂。
 * 注意：GaeaPickFiles 无文件类型过滤器（审计刀D 项）——调用方传 accept
 * 扩展名数组由本层后置校验，不合法抛错由调用方提示（fail-closed 不静默吞）。
 * GaeaReadFileB64 后端上限 64MB（os.Stat 拒绝，防误用），超限报错透传。
 */
import { app } from './bridge'
import { b64ToBytes } from './bytes'

/**
 * 是否运行在 Wails 壳内——全仓唯一规范判定（瘦身 P1 线1 ②合一）：
 * typeof window !== 'undefined' && 'go' in window。schedule/exportArtifact
 * 的 inShell 已薄委托至此（v4.162 实证：壳内 input[type=file] 不弹框、
 * <a download> 不落盘，浏览器一切正常）。
 */
export function inShellEnv(): boolean {
  return typeof window !== 'undefined' && 'go' in window
}

/** 图片扩展名白名单（GaeaPickFiles 无过滤器，本层按文件名后缀后置校验） */
const IMAGE_EXTS = ['png', 'jpg', 'jpeg', 'webp', 'gif', 'bmp']

/** 扩展名 → MIME 映射（未知映射诚实降级 application/octet-stream） */
const MIME_BY_EXT: Record<string, string> = {
  png: 'image/png',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  webp: 'image/webp',
  gif: 'image/gif',
  bmp: 'image/bmp',
}

/** 取小写扩展名（不含点；无扩展名返回空串） */
function extOf(name: string): string {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

/**
 * 壳内选取单个文件并还原为 File（浏览器路径由调用方走原生 input，本函数
 * 不做壳判断）。GaeaPickFiles 空=用户取消→返回 null。accept 传小写扩展名
 * 数组（如 ['png','jpg']），按文件名后缀判定（大小写不敏感）；不合法抛
 * Error('仅支持 …')由调用方 toast——fail-closed 不静默吞。内容经
 * GaeaReadFileB64 读回（上限 64MB，超限报错透传）。
 */
export async function pickFileAsFile(accept?: string[]): Promise<File | null> {
  const picked = await app.PickFiles()
  if (!picked?.length) return null
  const f = picked[0]
  if (accept?.length) {
    const allowed = accept.map((e) => e.toLowerCase().replace(/^\./, ''))
    if (!allowed.includes(extOf(f.name))) {
      throw new Error(`仅支持 ${allowed.map((e) => `.${e}`).join('/')} 文件`)
    }
  }
  const b64 = await app.ReadFileB64(f.path)
  const bytes = b64ToBytes(b64)
  return new File([bytes], f.name)
}

/**
 * 壳内选取一张图片并读成 data URL（基于 pickFileAsFile）：图片扩展名白名单
 * 后置校验（不合法抛错由调用方 toast），取消→null。返回
 * `data:<mime>;base64,<b64>`——mime 按扩展名映射，未知映射诚实降级
 * application/octet-stream。GaeaReadFileB64 上限 64MB（超限报错透传）。
 */
export async function pickImageAsDataUrl(): Promise<string | null> {
  const file = await pickFileAsFile(IMAGE_EXTS)
  if (!file) return null
  const b64 = await fileToB64(file)
  return `data:${MIME_BY_EXT[extOf(file.name)] ?? 'application/octet-stream'};base64,${b64}`
}

/** File → 裸 base64（FileReader dataURL 去头部） */
function fileToB64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const fr = new FileReader()
    fr.onload = () => resolve(String(fr.result).slice(String(fr.result).indexOf(',') + 1))
    fr.onerror = () => reject(new Error('读取图片失败'))
    fr.readAsDataURL(file)
  })
}
