// VisionTrial（图像域 T1）：识图「读/懂」画室试用视图——挂在「生成模式」轨道的
// 独立 tab（与 v4.99 素材库同模式，见 ImageGenPage 接线）。
//   - 载图：paste 事件或本机选图 → readFileAsDataURL → SavePastedImage 落盘
//     （既有绑定 GaeaSavePastedImage，零新绑定；mock 样例见 mock/office.ts）；
//   - 动作：「识别内容」走 GaeaRecognizeImage（识图-懂 vision.understand，
//     提示词可自填、缺省给通用识别 prompt）；「提取文字」走 GaeaOCRText
//     （识图-读 vision.read）；
//   - 结果诚实呈现：识别/OCR 文本或错误原文；原语标注（识图-懂 / 识图-读）+
//     模型名（仅当返回里确实携带）。登记口径 = 无需前端介入——识图原语
//     ProducesAsset=false（internal/app/image_domain.go 注册表），不产图不入库，
//     只在结果区如实标注；
//   - 历史：localStorage 最近 5 条（含路径/动作/提示词/文本截断），缩略按路径
//     懒读复用 readFileAsDataURL，结果缓存不重读。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Input, Spin, Tag, Typography, message } from 'antd'
import {
  ArrowLeftOutlined, EyeOutlined, FileImageOutlined, ScanOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import {
  readFileAsDataURL, savePastedImage, visionRead, visionUnderstand,
  type VisionCallResult,
} from '../../api/image'
import { useT } from '../../gaea/lib/i18n'

type VisionAction = 'understand' | 'read'

interface VisionHistoryItem {
  id: string
  path: string
  action: VisionAction
  prompt: string
  text: string
  model?: string
  createdAt: string
}

const HISTORY_KEY = 'imagehubT1.visionHistory'
const HISTORY_MAX = 5
/** 历史文本截断上限（轻量保留：只存摘要，原图按路径懒读）。 */
const HISTORY_TEXT_LIMIT = 600

function errText(e: unknown): string {
  return (e instanceof Error && e.message) || String(e)
}

function fileToDataUrl(file: File): Promise<string> {
  return new Promise((res, rej) => {
    const r = new FileReader()
    r.onload = () => res(String(r.result))
    r.onerror = () => rej(r.error)
    r.readAsDataURL(file)
  })
}

/** 读历史（损坏/越界数据一律丢弃回空，不阻断视图）。 */
function loadHistory(): VisionHistoryItem[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((x): x is VisionHistoryItem =>
        !!x && typeof x === 'object' && typeof (x as VisionHistoryItem).path === 'string')
      .slice(0, HISTORY_MAX)
  } catch {
    return []
  }
}

function persistHistory(items: VisionHistoryItem[]): void {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(items))
  } catch { /* 隐私模式等写不进：历史退化为本次组件态 */ }
}

export const VisionTrial: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const t = useT()
  const fileRef = useRef<HTMLInputElement | null>(null)
  const thumbCacheRef = useRef<Map<string, string>>(new Map())

  const [path, setPath] = useState('')
  const [preview, setPreview] = useState('')
  const [saving, setSaving] = useState(false)
  const [prompt, setPrompt] = useState('')
  const [running, setRunning] = useState<VisionAction | ''>('')
  const [result, setResult] = useState<(VisionCallResult & { action: VisionAction }) | null>(null)
  const [errMsg, setErrMsg] = useState('')
  const [history, setHistory] = useState<VisionHistoryItem[]>(() => loadHistory())
  const [histThumbs, setHistThumbs] = useState<Record<string, string>>({})

  // 载图统一漏斗：dataURL → SavePastedImage 落盘 → 读回预览（失败诚实呈现）。
  const handleImageDataUrl = useCallback(async (dataUrl: string) => {
    if (!dataUrl) return
    setSaving(true)
    setErrMsg('')
    try {
      const savedPath = await savePastedImage(dataUrl)
      if (!savedPath) throw new Error('empty path')
      setPath(savedPath)
      setPreview(await readFileAsDataURL(savedPath).catch(() => ''))
    } catch (e: unknown) {
      setErrMsg(t('imagehubT1.visionSaveFailed', { msg: errText(e) }))
    } finally {
      setSaving(false)
    }
  }, [t])

  // paste 事件：剪贴板里的图片直接进漏斗（窗口级监听，组件卸载时移除）。
  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      const items = e.clipboardData?.items
      if (!items) return
      for (const item of Array.from(items)) {
        if (item.type.startsWith('image/')) {
          const f = item.getAsFile()
          if (f) {
            e.preventDefault()
            void fileToDataUrl(f).then(handleImageDataUrl)
          }
          return
        }
      }
    }
    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [handleImageDataUrl])

  // 历史缩略懒读（按路径缓存，已读不重读；失败 = 空串占位）。
  useEffect(() => {
    let cancelled = false
    void (async () => {
      for (const h of history) {
        if (thumbCacheRef.current.has(h.path)) continue
        const url = await readFileAsDataURL(h.path).catch(() => '')
        if (cancelled) return
        thumbCacheRef.current.set(h.path, url || '')
        setHistThumbs((prev) => ({ ...prev, [h.path]: url || '' }))
      }
    })()
    return () => { cancelled = true }
  }, [history])

  const pushHistory = useCallback((item: { path: string; action: VisionAction; prompt: string; text: string; model?: string }) => {
    setHistory((prev) => {
      const rest = prev.filter((h) => !(h.path === item.path && h.action === item.action))
      const next = [{
        ...item,
        text: item.text.slice(0, HISTORY_TEXT_LIMIT),
        id: `vh-${Date.now()}`,
        createdAt: new Date().toISOString(),
      }, ...rest].slice(0, HISTORY_MAX)
      persistHistory(next)
      return next
    })
  }, [])

  const runAction = useCallback(async (action: VisionAction) => {
    if (!path) {
      message.warning(t('imagehubT1.visionNoImage'))
      return
    }
    if (running) return
    setRunning(action)
    setErrMsg('')
    try {
      const r = action === 'understand'
        ? await visionUnderstand(path, prompt.trim() || t('imagehubT1.visionPromptDefault'))
        : await visionRead(path)
      setResult({ ...r, action })
      pushHistory({ path, action, prompt: prompt.trim(), text: r.text, model: r.model })
    } catch (e: unknown) {
      setResult(null)
      setErrMsg(errText(e))
    } finally {
      setRunning('')
    }
  }, [path, prompt, running, pushHistory, t])

  // 历史回放：把该条目的图片/提示词/结果装回工作区（可原地重跑）。
  const restoreHistory = useCallback(async (h: VisionHistoryItem) => {
    setPath(h.path)
    setPrompt(h.prompt || t('imagehubT1.visionPromptDefault'))
    setResult({ text: h.text, model: h.model, action: h.action })
    setErrMsg('')
    const cached = thumbCacheRef.current.get(h.path)
    const url = cached !== undefined ? cached : await readFileAsDataURL(h.path).catch(() => '')
    thumbCacheRef.current.set(h.path, url || '')
    setPreview(url || '')
  }, [t])

  const clearHistory = useCallback(() => {
    setHistory([])
    try { localStorage.removeItem(HISTORY_KEY) } catch { /* 忽略 */ }
  }, [])

  const busyText = running === 'understand'
    ? t('imagehubT1.visionRecognizeBusy')
    : t('imagehubT1.visionOcrBusy')

  return (
    <div className="ig-vision-trial v3-zone" aria-label={t('imagehubT1.visionTitle')}
      style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: '10px 12px' }}>
      {/* 顶条：返回 + 标题 + 原语徽标 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0, flexWrap: 'wrap' }}>
        <Button size="small" icon={<ArrowLeftOutlined />} onClick={onClose}
          style={{ borderRadius: 999, fontSize: 12 }}>
          {t('imagehubT1.libraryBack')}
        </Button>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}>
          {t('imagehubT1.visionTitle')}
        </Typography.Text>
        <Tag color="geekblue" style={{ marginInlineEnd: 0 }}>{t('imagehubT1.visionCapUnderstand')}</Tag>
        <Tag color="cyan" style={{ marginInlineEnd: 0 }}>{t('imagehubT1.visionCapRead')}</Tag>
      </div>
      <Typography.Text type="secondary" style={{ fontSize: 11, marginTop: 6, flexShrink: 0 }}>
        {t('imagehubT1.visionSubtitle')}
      </Typography.Text>

      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', marginTop: 10, display: 'flex', flexDirection: 'column', gap: 10 }}>
        {/* 载图区：粘贴 / 本机选图，统一走 SavePastedImage 落盘 */}
        <input ref={fileRef} type="file" accept="image/*" style={{ display: 'none' }}
          onChange={(e) => {
            const f = e.target.files?.[0]
            e.target.value = ''
            if (f) void fileToDataUrl(f).then(handleImageDataUrl)
          }} />
        <button type="button" onClick={() => fileRef.current?.click()}
          style={{
            border: '1px dashed var(--border-subtle)', borderRadius: 12, padding: '18px 12px',
            background: 'rgba(255,255,255,0.03)', cursor: 'pointer', width: '100%',
            display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, flexShrink: 0,
          }}>
          {saving ? <Spin size="small" /> : <FileImageOutlined style={{ fontSize: 22, color: C('color-text-secondary'), opacity: 0.7 }} />}
          <span style={{ fontSize: 12, color: C('color-text-secondary') }}>
            {saving ? t('imagehubT1.visionPasteBusy') : t('imagehubT1.visionPasteHint')}
          </span>
        </button>

        {/* 已载图片预览 + 路径（诚实展示落盘路径） */}
        {path && (
          <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start', flexShrink: 0 }}>
            {preview ? (
              <img src={preview} alt={path}
                style={{ maxWidth: 240, maxHeight: 180, objectFit: 'contain', borderRadius: 8, border: '1px solid var(--border-subtle)' }} />
            ) : (
              <div style={{ width: 120, height: 90, borderRadius: 8, border: '1px solid var(--border-subtle)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <FileImageOutlined style={{ fontSize: 20, color: C('color-text-secondary'), opacity: 0.6 }} />
              </div>
            )}
            <div style={{ minWidth: 0, fontSize: 11, display: 'flex', flexDirection: 'column', gap: 3 }}>
              <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                {t('imagehubT1.visionPathLabel')}
              </Typography.Text>
              <Typography.Text style={{ fontSize: 11, wordBreak: 'break-all' }}>{path}</Typography.Text>
            </div>
          </div>
        )}

        {/* 提示词 + 两动作（识图-懂 / 识图-读） */}
        <div style={{ flexShrink: 0 }}>
          <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
            {t('imagehubT1.visionPromptLabel')}
          </Typography.Text>
          <Input.TextArea rows={2} value={prompt}
            placeholder={t('imagehubT1.visionPromptDefault')}
            onChange={(e) => setPrompt(e.target.value)} style={{ marginTop: 4 }} />
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <Button type="primary" size="small" icon={<EyeOutlined />}
              loading={running === 'understand'} disabled={running === 'read'}
              onClick={() => void runAction('understand')}>
              {t('imagehubT1.visionActRecognize')}
            </Button>
            <Button size="small" icon={<ScanOutlined />}
              loading={running === 'read'} disabled={running === 'understand'}
              onClick={() => void runAction('read')}>
              {t('imagehubT1.visionActOcr')}
            </Button>
          </div>
        </div>

        {/* 结果区：原语标注 + 模型名（若返回里有）+ 文本/错误原文 */}
        <div style={{
          border: '1px solid var(--border-subtle)', borderRadius: 10,
          background: 'rgba(255,255,255,0.03)', padding: '10px 12px', flexShrink: 0, minWidth: 0,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
            {result && (
              <Tag color={result.action === 'understand' ? 'geekblue' : 'cyan'} style={{ marginInlineEnd: 0 }}>
                {result.action === 'understand' ? t('imagehubT1.visionCapUnderstand') : t('imagehubT1.visionCapRead')}
              </Tag>
            )}
            {result?.model && <Tag color="blue" style={{ marginInlineEnd: 0 }}>{result.model}</Tag>}
          </div>
          {running && (
            <div className="ig-task-empty" style={{ marginTop: 8 }}>
              <Spin size="small" />
              <span>{busyText}</span>
            </div>
          )}
          {!running && result && (
            <Typography.Paragraph style={{ marginTop: 8, marginBottom: 0, fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
              {result.text}
            </Typography.Paragraph>
          )}
          {!running && errMsg && (
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="danger" style={{ fontSize: 12, fontWeight: 600 }}>
                {t('imagehubT1.visionErrorTitle')}
              </Typography.Text>
              <Typography.Paragraph type="danger" style={{ marginTop: 4, marginBottom: 0, fontSize: 11, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                {errMsg}
              </Typography.Paragraph>
            </div>
          )}
          {!running && !result && !errMsg && (
            <div className="ig-task-empty" style={{ marginTop: 8 }}>
              <FileImageOutlined style={{ fontSize: 18, color: C('color-text-secondary'), opacity: 0.6 }} />
              <span>{t('imagehubT1.visionResultEmpty')}</span>
            </div>
          )}
          <Typography.Text type="secondary" style={{ display: 'block', fontSize: 10, marginTop: 8 }}>
            {t('imagehubT1.visionResultNote')}
          </Typography.Text>
        </div>

        {/* 历史记录（最近 5 条，含缩略；点击回放，可原地重跑） */}
        <div style={{ flexShrink: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
              {t('imagehubT1.visionHistoryTitle')}
            </Typography.Text>
            {history.length > 0 && (
              <Button size="small" type="text" onClick={clearHistory} style={{ fontSize: 11 }}>
                {t('imagehubT1.visionHistoryClear')}
              </Button>
            )}
          </div>
          {history.length === 0 ? (
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('imagehubT1.visionHistoryEmpty')}
            </Typography.Text>
          ) : (
            <div style={{ display: 'flex', gap: 8, marginTop: 6, flexWrap: 'wrap' }}>
              {history.map((h) => (
                <button key={h.id} type="button" onClick={() => void restoreHistory(h)}
                  title={`${h.action === 'understand' ? t('imagehubT1.visionActRecognize') : t('imagehubT1.visionActOcr')} · ${h.text.slice(0, 80)}`}
                  style={{
                    width: 96, padding: 4, cursor: 'pointer', textAlign: 'left',
                    border: '1px solid', borderColor: path === h.path ? 'var(--color-primary)' : 'var(--border-subtle)',
                    borderRadius: 10, background: 'rgba(255,255,255,0.03)',
                  }}>
                  {histThumbs[h.path] ? (
                    <img src={histThumbs[h.path]} alt={h.path}
                      style={{ width: '100%', aspectRatio: '1 / 1', objectFit: 'cover', borderRadius: 7, display: 'block' }} />
                  ) : (
                    <div style={{ width: '100%', aspectRatio: '1 / 1', borderRadius: 7, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(255,255,255,0.04)' }}>
                      <FileImageOutlined style={{ fontSize: 16, color: C('color-text-secondary'), opacity: 0.6 }} />
                    </div>
                  )}
                  <Tag color={h.action === 'understand' ? 'geekblue' : 'cyan'}
                    style={{ marginInlineEnd: 0, marginTop: 4, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>
                    {h.action === 'understand' ? t('imagehubT1.visionCapUnderstand') : t('imagehubT1.visionCapRead')}
                  </Tag>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
