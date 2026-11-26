// Composer 拆分产物：附件/粘贴/截图/识图/OCR 状态机（行为零变化，T6-10.1）
import { useRef, useState } from 'react'
import { app as defaultApp } from '../lib/bridge'
import type { ComposerBridge } from '../lib/composerBridge'
import { isImagePath } from '../lib/pickFile'
import { useToast } from '../components/Toast'

export interface Attachment {
  path: string;
  previewUrl: string;
  type: "image" | "file";
  /** 附件字节数（P2-4 上下文用量透明化：图片由 base64 估算，文件取 File.size）。 */
  size?: number;
}

export interface AttachmentUploadTask {
  id: string;
  name: string;
  size?: number;
  type: "image" | "file";
  previewUrl?: string;
  status: "uploading" | "error";
  progress?: number;
  error?: string;
  retryable: boolean;
}

interface UploadSource {
  file: File;
  dataUrl?: string;
}

export const ATTACHMENT_LIMITS = {
  /** 单次桥接整包上限（SaveAttachmentFile 单调用；超过走分块上传）。 */
  file: 25 * 1024 * 1024,
  /** 分块上传总量上限（与后端 attachment.MaxChunkedFileBytes 同档）。 */
  chunkedFile: 1024 * 1024 * 1024,
  image: 10 * 1024 * 1024,
} as const

export interface OversizeAttachment {
  file: File
  limit: number
  kind: "image" | "file"
}

/**
 * Classify by metadata before any FileReader reads bytes.
 *
 * 2026-11-12 超大附件改不拒收：图片仍守 10MB（识图引擎限制面），文件只把
 * 超分块总量上限（1GiB）的挡在门外；25MB～1GiB 的文件照常 accepted，由
 * attachDroppedFiles 按体积路由到分块上传（拖拽无源路径，字节必须过桥，
 * 但改为逐片流式不再整包）。选择对话框的文件本就按路径引用（发送侧只注
 * 入 @路径 头部 64KB），不在此分类——那里任何大小都直接挂 chip。
 */
export function splitOversizeAttachments(files: File[]): {
  accepted: File[]
  rejected: OversizeAttachment[]
} {
  const accepted: File[] = []
  const rejected: OversizeAttachment[] = []
  for (const file of files) {
    const kind = file.type.startsWith("image/") || isImagePath(file.name) ? "image" : "file"
    const limit = kind === "image" ? ATTACHMENT_LIMITS.image : ATTACHMENT_LIMITS.chunkedFile
    if (file.size > limit) rejected.push({ file, limit, kind })
    else accepted.push(file)
  }
  return { accepted, rejected }
}

export interface UseComposerAttachmentsOptions {
  text: string
  setText: React.Dispatch<React.SetStateAction<string>>
  taRef: React.RefObject<HTMLTextAreaElement>
  running: boolean
  onSend: (displayText: string, submitText?: string) => void
  /** 会话草稿恢复用；缺省空数组保持既有调用方零变化。 */
  initialAttachments?: Attachment[]
  /** 双壳共用 Composer 时显式注入本壳 bridge；缺省走办公 bridge。 */
  bridge?: ComposerBridge
}

// 图片 base64 dataURL → 估算字节数（P2-4 上下文占用透明化）。
// base64 每 4 字符 ≈ 3 字节,减去 data:image/...;base64, 前缀。
function dataUrlBytes(dataUrl: string): number {
  const comma = dataUrl.indexOf(",");
  const body = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
  return Math.floor((body.length * 3) / 4);
}

function fileFromDataUrl(dataUrl: string, name: string): File {
  const match = /^data:([^;,]+);base64,(.*)$/s.exec(dataUrl)
  if (!match) throw new Error("截图数据格式无效")
  const binary = atob(match[2])
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return new File([bytes], name, { type: match[1] })
}

// 对话框级故障没有单文件任务锚点，仍以 toast 可见上报。
function attachFailText(step: string, e: unknown): string {
  const why = e instanceof Error ? (e.message || String(e)) : String(e ?? "未知错误");
  return `${step}：${why}`;
}

function uploadErrorText(step: string, e: unknown): string {
  return `${step}：${e instanceof Error ? (e.message || String(e)) : String(e ?? "未知错误")}`;
}

function isRetryableUploadError(e: unknown): boolean {
  const message = e instanceof Error ? (e.message || String(e)) : String(e ?? "");
  return !/(不支持|非法|格式|超限|超过|权限|invalid|unsupported|permission|denied|not found|no such|disk full|空间不足|磁盘已满)/i.test(message);
}

let uploadTaskSeq = 0;
function nextUploadTaskId(): string {
  uploadTaskSeq += 1;
  return `upload-${Date.now()}-${uploadTaskSeq}`;
}

// bytesToBase64 分块编码（v4.508）：此前逐字节 String.fromCharCode 拼接——
// 10MB 文件 = 千万次拼接 + 一条 10MB 巨串进 btoa，UI 卡秒级。8KB 块编码
// 拼接，行为等价（btoa 对分块边界无要求，Latin-1 逐字节语义不变）。
function bytesToBase64(bytes: Uint8Array): string {
  const CHUNK = 0x8000
  let bin = ""
  for (let i = 0; i < bytes.length; i += CHUNK) {
    bin += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return btoa(bin)
}

// ── 超大附件分块上传（2026-11-12）──
// 单次 SaveAttachmentFile 整包过桥 25MB 封顶；更大文件走 Begin/Append/Finish
// 三段绑定流式落盘。切片必须 3 字节对齐：base64 每 3 字节独立成组，对齐切
// 才能保证每片都是合法可解码的 standalone base64（8388606 = 8MiB 向下取整
// 到 3 的倍数）。失败即 Abort 清理后端临时会话，再把原始错误抛给 toast。
const CHUNK_UPLOAD_BYTES = 8 * 1024 * 1024 - (8 * 1024 * 1024) % 3

function readSlice(file: File, start: number, end: number): Promise<ArrayBuffer> {
  return new Promise((res, rej) => {
    const r = new FileReader()
    r.onload = () => res(r.result as ArrayBuffer)
    r.onerror = () => rej(r.error)
    r.readAsArrayBuffer(file.slice(start, end))
  })
}

async function uploadChunkedAttachment(file: File, api: ComposerBridge, onProgress?: (percent: number) => void): Promise<string> {
  const token = await api.AttachmentChunkBegin(file.name)
  try {
    for (let off = 0; off < file.size; off += CHUNK_UPLOAD_BYTES) {
      const buf = await readSlice(file, off, Math.min(off + CHUNK_UPLOAD_BYTES, file.size))
      await api.AttachmentChunkAppend(token, bytesToBase64(new Uint8Array(buf)))
      onProgress?.(Math.min(95, Math.round(((off + CHUNK_UPLOAD_BYTES) / file.size) * 100)))
    }
    return await api.AttachmentChunkFinish(token)
  } catch (e: unknown) {
    try { await api.AttachmentChunkAbort(token) } catch { /* 尽力清理，不掩盖原始错误 */ }
    throw e
  }
}

// appendUnique 附件去重（v4.508）：同路径重复拖入/选择不再堆叠 chip——
// 提交时同一 @路径 注入两遍只会白占上下文。
function appendUnique(prev: Attachment[], next: Attachment): Attachment[] {
  return prev.some((x) => x.path === next.path) ? prev : [...prev, next]
}

// v4.508：识图/OCR 不再清草稿后 setText 无消费——不解构（接口字段保留，
// 调用方签名不变；text/taRef 同为兼容保留）。
export function useComposerAttachments({ running, onSend, initialAttachments, bridge: bridgeOverride }: UseComposerAttachmentsOptions) {
  const bridge = bridgeOverride ?? defaultApp
  const toast = useToast()
  const [attachments, setAttachments] = useState<Attachment[]>(() => initialAttachments ?? [])
  const [uploadTasks, setUploadTasks] = useState<AttachmentUploadTask[]>([])
  const [pendingPaste, setPendingPaste] = useState(0)
  const [cropSrc, setCropSrc] = useState<string | null>(null)
  const [captureBusy, setCaptureBusy] = useState(false)
  const [recognizingPath, setRecognizingPath] = useState<string | null>(null)
  const [ocrPath, setOcrPath] = useState<string | null>(null)
  // 与 Composer.submit 一致：onSend 每次渲染更新，避免异步回调读到过期闭包
  const onSendRef = useRef(onSend)
  onSendRef.current = onSend
  const uploadSourcesRef = useRef(new Map<string, UploadSource>())

  const updateTask = (id: string, patch: Partial<AttachmentUploadTask>) => {
    setUploadTasks((prev) => prev.map((task) => task.id === id ? { ...task, ...patch } : task))
  }

  const removeTask = (id: string) => {
    uploadSourcesRef.current.delete(id)
    setUploadTasks((prev) => prev.filter((task) => task.id !== id))
  }

  const addErrorTask = (
    file: { name: string; size?: number },
    type: "image" | "file",
    reason: string,
    retryable: boolean,
  ): AttachmentUploadTask => {
    const task: AttachmentUploadTask = {
      id: nextUploadTaskId(), name: file.name, size: file.size, type,
      status: "error", error: reason, retryable,
    }
    setUploadTasks((prev) => [...prev, task])
    return task
  }

  const performUpload = async (taskId: string, file: File, type: "image" | "file") => {
    const source = uploadSourcesRef.current.get(taskId) ?? { file }
    uploadSourcesRef.current.set(taskId, source)
    setPendingPaste((n) => n + 1)
    try {
      if (type === "image") {
        let dataUrl = source.dataUrl
        if (!dataUrl) {
          dataUrl = await new Promise<string>((res, rej) => {
            const r = new FileReader()
            r.onload = () => res(String(r.result))
            r.onerror = () => rej(r.error)
            r.readAsDataURL(file)
          })
          source.dataUrl = dataUrl
          updateTask(taskId, { previewUrl: dataUrl, progress: 45 })
        }
        const path = await bridge.SavePastedImage(dataUrl)
        setAttachments((prev) => appendUnique(prev, {
          path, previewUrl: dataUrl, type: "image", size: file.size || dataUrlBytes(dataUrl),
        }))
      } else {
        const path = file.size > ATTACHMENT_LIMITS.file
          ? await uploadChunkedAttachment(file, bridge, (progress) => updateTask(taskId, { progress }))
          : await (async () => {
              const buf = await new Promise<ArrayBuffer>((res, rej) => {
                const r = new FileReader()
                r.onload = () => res(r.result as ArrayBuffer)
                r.onerror = () => rej(r.error)
                r.readAsArrayBuffer(file)
              })
              updateTask(taskId, { progress: 55 })
              return bridge.SaveAttachmentFile(file.name, bytesToBase64(new Uint8Array(buf)))
            })()
        setAttachments((prev) => appendUnique(prev, { path, previewUrl: "", type: "file", size: file.size }))
      }
      removeTask(taskId)
    } catch (e: unknown) {
      const step = type === "image" ? "图片上传失败" : "文件上传失败"
      updateTask(taskId, {
        status: "error",
        error: uploadErrorText(step, e),
        retryable: isRetryableUploadError(e),
        progress: undefined,
      })
    } finally {
      setPendingPaste((n) => Math.max(0, n - 1))
    }
  }

  // 附加文件：图片走已有流程，非图片用 base64 上传
  const attachDroppedFiles = async (files: File[]) => {
    const split = splitOversizeAttachments(files)
    for (const rejected of split.rejected) {
      addErrorTask(
        rejected.file,
        rejected.kind,
        `超过 ${Math.round(rejected.limit / (1024 * 1024))}MB 上限`,
        false,
      )
    }
    for (const file of split.accepted) {
      const type = file.type.startsWith("image/") || isImagePath(file.name) ? "image" : "file"
      const task: AttachmentUploadTask = {
        id: nextUploadTaskId(), name: file.name, size: file.size, type,
        status: "uploading", progress: 0, retryable: true,
      }
      uploadSourcesRef.current.set(task.id, { file })
      setUploadTasks((prev) => [...prev, task])
      await performUpload(task.id, file, type)
    }
  }

  const retryUploadTask = async (id: string) => {
    const task = uploadTasks.find((item) => item.id === id)
    const source = uploadSourcesRef.current.get(id)
    if (!task || task.status !== "error" || !task.retryable || !source) return
    updateTask(id, { status: "uploading", error: undefined, progress: 0 })
    await performUpload(id, source.file, task.type)
  }

  // 导入文件：通过原生对话框选择文件
  const handlePickFiles = async () => {
    // FE4-12：旧后端/HTTP 桥不暴露 PickFiles 时才静默（绑定不存在 ≠ 调用失败）。
    // 其余错误（对话框异常、后端拒绝）必须可见。
    if (typeof bridge.PickFiles !== "function") return
    try {
      // filters 显式空串=不过滤（Go 侧口径）。此前无参调用在 Wails v2.13 下被
      // 拒「received 0 arguments, expected 1」（2026-10-22/23 gaea.log 实录）。
      const files = await bridge.PickFiles("")
      if (!files || files.length === 0) return
      for (const f of files) {
        // v4.508 bug 根修：FilePickResult 无 MIME type 字段，此前按 f.type
        // 判断对话框选的图片恒 false——图片全被当普通文件（无缩略图/无识图
        // OCR）。改按扩展名（isImagePath 单源，与 pickImageAsDataUrl 白名单
        // 同源）；previewUrl 空由附件条图标兜底。
        const image = isImagePath(f.path)
        // 2026-11-12：对话框选的文件不再设大小闸——文件附件只按路径引用
        // （提交注入 @路径，发送侧仅注入头部 64KB，AI 用 read_file 补读），
        // 大小只受磁盘约束，任何体积都能挂 chip。图片仍守 10MB（识图链
        // 限制面，与拖拽同口径）。
        if (image && (f.size ?? 0) > ATTACHMENT_LIMITS.image) {
          addErrorTask(f, "image", `超过 ${Math.round(ATTACHMENT_LIMITS.image / (1024 * 1024))}MB 上限`, false)
          continue
        }
        if (image) {
          setAttachments((prev) => appendUnique(prev, { path: f.path, previewUrl: f.previewUrl ?? "", type: "image" as const, size: f.size }))
        } else {
          // P0-1 chip 化：非图片附件进 attachments（提交时统一注入 @路径）
          setAttachments((prev) => appendUnique(prev, { path: f.path, previewUrl: "", type: "file" as const, size: f.size }))
        }
      }
    } catch (e: unknown) {
      toast.show(attachFailText("选择附件失败", e), "warn")
    }
  }

  // ── 截图：整屏捕获 → 裁剪浮层 → 复用图片附件流程 ──
  const handleScreenshot = async () => {
    if (running || captureBusy) return
    setCaptureBusy(true)
    try {
      const dataUrl = await bridge.CaptureScreen()
      if (dataUrlBytes(dataUrl) > ATTACHMENT_LIMITS.image) {
        toast.show(`截图超过 ${Math.round(ATTACHMENT_LIMITS.image / (1024 * 1024))}MB 上限`, "warn")
        return
      }
      setCropSrc(dataUrl)
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setCaptureBusy(false)
    }
  }

  const handleCropConfirm = async (dataUrl: string) => {
    setCropSrc(null)
    if (dataUrlBytes(dataUrl) > ATTACHMENT_LIMITS.image) {
      addErrorTask(
        { name: `截图-${Date.now()}.png`, size: dataUrlBytes(dataUrl) },
        "image",
        `超过 ${Math.round(ATTACHMENT_LIMITS.image / (1024 * 1024))}MB 上限`,
        false,
      )
      return
    }
    try {
      const file = fileFromDataUrl(dataUrl, `截图-${Date.now()}.png`)
      const task: AttachmentUploadTask = {
        id: nextUploadTaskId(), name: file.name, size: file.size, type: "image",
        previewUrl: dataUrl, status: "uploading", progress: 45, retryable: true,
      }
      uploadSourcesRef.current.set(task.id, { file, dataUrl })
      setUploadTasks((prev) => [...prev, task])
      await performUpload(task.id, file, "image")
    } catch (e: unknown) {
      addErrorTask({ name: "截图", size: dataUrlBytes(dataUrl) }, "image", uploadErrorText("截图上传失败", e), true)
    }
  }

  // ── 识图：本地视觉模型识别附件图片，把结果作为用户消息发给助手 ──
  // v4.508：不再 setText("") 清空输入框——用户贴图后常先写半句要求再点识图，
  // 识别结果独立成一条消息发送，草稿必须保留（此前被整段清掉=输入丢失）。
  const handleRecognize = async (att: Attachment) => {
    if (running || recognizingPath) return
    setRecognizingPath(att.path)
    try {
      const desc = await bridge.RecognizeImage(
        att.path,
        "请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。",
      )
      const name = att.path.split(/[/\\]/).pop() || att.path
      const msg = `【图片识图：${name}】\n${desc}`
      setAttachments((prev) => prev.filter((x) => x.path !== att.path))
      onSendRef.current(msg, msg)
      toast.show("识别完成，已发送给助手")
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setRecognizingPath(null)
    }
  }

  // ── 提取文字：本地 Unlimited-OCR-2 常驻服务识别图片中的文字，结果作为用户消息发给助手 ──
  const handleOCRText = async (att: Attachment) => {
    if (running || ocrPath) return
    setOcrPath(att.path)
    try {
      const text = await bridge.OCRText(att.path)
      const name = att.path.split(/[/\\]/).pop() || att.path
      const msg = `【图片文字提取：${name}】\n${text}`
      setAttachments((prev) => prev.filter((x) => x.path !== att.path))
      onSendRef.current(msg, msg)
      toast.show("文字提取完成，已发送给助手")
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setOcrPath(null)
    }
  }

  return {
    attachments, setAttachments, uploadTasks, pendingPaste,
    cropSrc, setCropSrc, captureBusy, recognizingPath, ocrPath,
    attachDroppedFiles, handlePickFiles, handleScreenshot,
    handleCropConfirm, handleRecognize, handleOCRText,
    retryUploadTask, removeUploadTask: removeTask,
  }
}
