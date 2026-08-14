// Composer 拆分产物：附件/粘贴/截图/识图/OCR 状态机（行为零变化，T6-10.1）
import { useRef, useState } from 'react'
import { Modal } from 'antd'
import { app } from '../lib/bridge'
import { useToast } from '../components/Toast'

export interface Attachment { path: string; previewUrl: string; type: "image" | "file" }

export interface UseComposerAttachmentsOptions {
  text: string
  setText: React.Dispatch<React.SetStateAction<string>>
  taRef: React.RefObject<HTMLTextAreaElement>
  running: boolean
  onSend: (displayText: string, submitText?: string) => void
}

export function useComposerAttachments({ text, setText, taRef, running, onSend }: UseComposerAttachmentsOptions) {
  const toast = useToast()
  const [attachments, setAttachments] = useState<Attachment[]>([])
  const [pendingPaste, setPendingPaste] = useState(0)
  const [cropSrc, setCropSrc] = useState<string | null>(null)
  const [captureBusy, setCaptureBusy] = useState(false)
  const [recognizingPath, setRecognizingPath] = useState<string | null>(null)
  const [ocrPath, setOcrPath] = useState<string | null>(null)
  // 与 Composer.submit 一致：onSend 每次渲染更新，避免异步回调读到过期闭包
  const onSendRef = useRef(onSend)
  onSendRef.current = onSend

  // 附加文件：图片走已有流程，非图片用 base64 上传
  const attachDroppedFiles = async (files: File[]) => {
    const images: File[] = []
    const others: File[] = []
    for (const f of files) {
      if (f.type.startsWith("image/")) images.push(f)
      else others.push(f)
    }
    // 大文件提示
    const bigFiles = files.filter((f) => f.size > 10 * 1024 * 1024)
    if (bigFiles.length > 0) {
      const names = bigFiles.map((f) => f.name).join(", ")
      // 原生 confirm 会同步阻塞 WebView2 主线程导致界面卡死，改用异步弹窗。
      const ok = await new Promise<boolean>((resolve) => {
        Modal.confirm({
          title: "大文件提示",
          content: `以下文件超过 10MB，可能上传较慢：\n${names}\n\n确定要继续吗？`,
          okText: "继续上传",
          cancelText: "取消",
          onOk: () => resolve(true),
          onCancel: () => resolve(false),
        })
      })
      if (!ok) return
    }
    // 处理图片
    for (const file of images) {
      setPendingPaste((n) => n + 1)
      try {
        const dataUrl = await new Promise<string>((res, rej) => { const r = new FileReader(); r.onload = () => res(String(r.result)); r.onerror = () => rej(r.error); r.readAsDataURL(file) })
        const path = await app.SavePastedImage(dataUrl)
        const previewUrl = await app.AttachmentDataURL(path)
        setAttachments((prev) => [...prev, { path, previewUrl, type: "image" }])
      } catch {} finally { setPendingPaste((n) => Math.max(0, n - 1)) }
    }
    // 处理非图片文件
    for (const file of others) {
      setPendingPaste((n) => n + 1)
      try {
        const buf = await new Promise<ArrayBuffer>((res, rej) => { const r = new FileReader(); r.onload = () => res(r.result as ArrayBuffer); r.onerror = () => rej(r.error); r.readAsArrayBuffer(file) })
        const bytes = new Uint8Array(buf)
        let bin = ""
        for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
        const b64 = btoa(bin)
        const path = await app.SaveAttachmentFile(file.name, b64)
        const atRef = `@${path}`
        setText((prev) => prev + (prev.endsWith(" ") || prev === "" ? "" : " ") + atRef + " ")
        if (taRef.current) { taRef.current.focus(); taRef.current.selectionStart = taRef.current.selectionEnd = (text + " " + atRef + " ").length }
      } catch {} finally { setPendingPaste((n) => Math.max(0, n - 1)) }
    }
  }

  // 导入文件：通过原生对话框选择文件
  const handlePickFiles = async () => {
    try {
      const files = await app.PickFiles()
      if (!files || files.length === 0) return
      for (const f of files) {
        if (f.type === "image") {
          setAttachments((prev) => [...prev, { path: f.path, previewUrl: f.previewUrl ?? "", type: "image" as const }])
        } else {
          const atRef = `@${f.path}`
          setText((prev) => prev + (prev.endsWith(" ") || prev === "" ? "" : " ") + atRef + " ")
          if (taRef.current) { taRef.current.focus(); taRef.current.selectionStart = taRef.current.selectionEnd = (text + " " + atRef + " ").length }
        }
      }
    } catch {
      // 静默处理（旧后端不支持）
    }
  }

  // ── 截图：整屏捕获 → 裁剪浮层 → 复用图片附件流程 ──
  const handleScreenshot = async () => {
    if (running || captureBusy) return
    setCaptureBusy(true)
    try {
      const dataUrl = await app.CaptureScreen()
      setCropSrc(dataUrl)
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setCaptureBusy(false)
    }
  }

  const handleCropConfirm = async (dataUrl: string) => {
    setCropSrc(null)
    setPendingPaste((n) => n + 1)
    try {
      const path = await app.SavePastedImage(dataUrl)
      const previewUrl = await app.AttachmentDataURL(path)
      setAttachments((prev) => [...prev, { path, previewUrl, type: "image" }])
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setPendingPaste((n) => Math.max(0, n - 1))
    }
  }

  // ── 识图：本地视觉模型识别附件图片，把结果作为用户消息发给助手 ──
  const handleRecognize = async (att: Attachment) => {
    if (running || recognizingPath) return
    setRecognizingPath(att.path)
    try {
      const desc = await app.RecognizeImage(
        att.path,
        "请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。",
      )
      const name = att.path.split(/[/\\]/).pop() || att.path
      const msg = `【图片识图：${name}】\n${desc}`
      setText("")
      setAttachments((prev) => prev.filter((x) => x.path !== att.path))
      onSendRef.current(msg, msg)
      toast.show("识别完成，已发送给助手")
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn")
    } finally {
      setRecognizingPath(null)
    }
  }

  // ── 提取文字：本地 OvisOCR2 常驻服务识别图片中的文字，结果作为用户消息发给助手 ──
  const handleOCRText = async (att: Attachment) => {
    if (running || ocrPath) return
    setOcrPath(att.path)
    try {
      const text = await app.OCRText(att.path)
      const name = att.path.split(/[/\\]/).pop() || att.path
      const msg = `【图片文字提取：${name}】\n${text}`
      setText("")
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
    attachments, setAttachments, pendingPaste,
    cropSrc, setCropSrc, captureBusy, recognizingPath, ocrPath,
    attachDroppedFiles, handlePickFiles, handleScreenshot,
    handleCropConfirm, handleRecognize, handleOCRText,
  }
}
