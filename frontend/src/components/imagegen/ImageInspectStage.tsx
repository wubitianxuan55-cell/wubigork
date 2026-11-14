/**
 * components/imagegen/ImageInspectStage.tsx — 识图工作台的图像检视画布
 *
 * 对标通用图像查看器（Figma / 截图工具）的基本手感：
 *  - 双标尺（上/左像素刻度，随缩放与平移实时重算）；
 *  - 滚轮以光标为锚点缩放、拖拽平移、双击在「适合窗口 ⇄ 100%」间切换；
 *  - 工具条：缩小 / 缩放读数 / 放大 / 适合窗口 / 1:1 / 全屏；
 *  - 状态读数：原图尺寸 · 缩放 · 光标图像坐标（十进制像素）。
 *
 * 布局：标尺各占 20px，图像区 clamp 在可用空间里（不撑破容器）；空图时给引导语。
 * 纯视图层：缩放/刻度数学在 `imageViewport.ts`，本组件只管事件与 DOM。
 */
import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Button, Tooltip } from 'antd'
import {
  FullscreenExitOutlined, FullscreenOutlined, MinusOutlined, OneToOneOutlined,
  PlusOutlined, ReloadOutlined,
} from '@ant-design/icons'
import { useT } from '../../gaea/lib/i18n'
import {
  ZOOM_MIN, centerView, clampZoom, fitZoom, formatZoom, imagePointAt, stepZoom, ticksFor, zoomAtPoint,
  type ViewState,
} from './imageViewport'
import './image-viewport.css'

export interface ImageInspectStageProps {
  /** 图片地址（data URL 或本地路径读回的 data URL） */
  src: string
  alt: string
  /** 原图尺寸（若已知；未知时等 onLoad 量） */
  width?: number
  height?: number
}

const RULER = 20

export const ImageInspectStage: React.FC<ImageInspectStageProps> = ({ src, alt, width, height }) => {
  const t = useT()
  const wrapRef = useRef<HTMLDivElement | null>(null)
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const [box, setBox] = useState({ w: 0, h: 0 })
  const [natural, setNatural] = useState({ w: width ?? 0, h: height ?? 0 })
  const [view, setView] = useState<ViewState>({ zoom: 1, x: 0, y: 0 })
  const [cursor, setCursor] = useState<{ x: number; y: number } | null>(null)
  const [fullscreen, setFullscreen] = useState(false)
  const [drag, setDrag] = useState<{ px: number; py: number; x: number; y: number } | null>(null)
  /** 是否已被用户手动缩放/平移：是则窗口变化不再强制 fit（尊重用户当前视角） */
  const userMoved = useRef(false)

  // 量视口（ResizeObserver；jsdom 有 stub，真机跟随面板拖拽实时更新）
  useLayoutEffect(() => {
    const el = viewportRef.current
    if (!el) return
    const measure = () => setBox({ w: el.clientWidth, h: el.clientHeight })
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  const doFit = useCallback(() => {
    if (natural.w <= 0 || natural.h <= 0 || box.w <= 0 || box.h <= 0) return
    const zoom = fitZoom(natural.w, natural.h, box.w, box.h)
    setView(centerView(natural.w, natural.h, box.w, box.h, zoom))
  }, [natural, box])

  // 首次量到原图/容器时自动适合窗口；之后只有点「适合窗口」才重新 fit
  useEffect(() => {
    if (userMoved.current) return
    doFit()
  }, [doFit])

  useEffect(() => {
    if (width && height) setNatural({ w: width, h: height })
  }, [width, height])

  const onWheel = (e: React.WheelEvent<HTMLDivElement>) => {
    e.preventDefault()
    const rect = e.currentTarget.getBoundingClientRect()
    const px = e.clientX - rect.left
    const py = e.clientY - rect.top
    const factor = Math.exp(-e.deltaY * 0.0015)
    userMoved.current = true
    setView((v) => zoomAtPoint(v, v.zoom * factor, px, py))
  }

  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return
    ;(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId)
    setDrag({ px: e.clientX, py: e.clientY, x: view.x, y: view.y })
  }
  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const p = imagePointAt(view, e.clientX - rect.left, e.clientY - rect.top)
    setCursor({ x: Math.round(p.x), y: Math.round(p.y) })
    if (!drag) return
    userMoved.current = true
    setView((v) => ({ ...v, x: drag.x + (e.clientX - drag.px), y: drag.y + (e.clientY - drag.py) }))
  }
  const onPointerUp = () => setDrag(null)

  const zoomBy = (dir: 1 | -1) => {
    userMoved.current = true
    setView((v) => zoomAtPoint(v, stepZoom(v.zoom, dir), box.w / 2, box.h / 2))
  }
  const actualSize = () => {
    userMoved.current = true
    setView(centerView(natural.w, natural.h, box.w, box.h, 1))
  }
  const fitNow = () => {
    userMoved.current = false
    doFit()
  }
  const toggleFullscreen = async () => {
    const el = wrapRef.current
    if (!el) return
    try {
      if (document.fullscreenElement) await document.exitFullscreen()
      else await el.requestFullscreen?.()
    } catch {
      // 浏览器拒绝（权限/嵌套）时退化为 CSS 全屏，保证按钮有反馈
      setFullscreen((v) => !v)
    }
  }

  useEffect(() => {
    const onChange = () => setFullscreen(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [])

  const hTicks = natural.w > 0 ? ticksFor(view, natural.w, Math.max(0, box.w)) : { step: 1, ticks: [] }
  const vTicks = natural.h > 0 ? ticksFor(view, natural.h, Math.max(0, box.h)) : { step: 1, ticks: [] }

  return (
    <div
      ref={wrapRef}
      className={`ig-stage${fullscreen ? ' is-fullscreen' : ''}`}
      data-testid="vision-stage"
    >
      <div className="ig-stage-toolbar" role="toolbar" aria-label={t('imagehubT1.visionStageToolbar')}>
        <Tooltip title={t('imagehubT1.visionZoomOut')}>
          <Button size="small" type="text" icon={<MinusOutlined />} aria-label={t('imagehubT1.visionZoomOut')}
            disabled={!src || view.zoom <= ZOOM_MIN} onClick={() => zoomBy(-1)} data-testid="vision-zoom-out" />
        </Tooltip>
        <span className="ig-stage-zoom" data-testid="vision-zoom-readout">{formatZoom(view.zoom)}</span>
        <Tooltip title={t('imagehubT1.visionZoomIn')}>
          <Button size="small" type="text" icon={<PlusOutlined />} aria-label={t('imagehubT1.visionZoomIn')}
            disabled={!src} onClick={() => zoomBy(1)} data-testid="vision-zoom-in" />
        </Tooltip>
        <span className="ig-stage-sep" aria-hidden />
        <Tooltip title={t('imagehubT1.visionZoomFit')}>
          <Button size="small" type="text" icon={<ReloadOutlined />} aria-label={t('imagehubT1.visionZoomFit')}
            disabled={!src} onClick={fitNow} data-testid="vision-zoom-fit" />
        </Tooltip>
        <Tooltip title={t('imagehubT1.visionZoomActual')}>
          <Button size="small" type="text" icon={<OneToOneOutlined />} aria-label={t('imagehubT1.visionZoomActual')}
            disabled={!src} onClick={actualSize} data-testid="vision-zoom-actual" />
        </Tooltip>
        <span className="ig-stage-grow" />
        <span className="ig-stage-size" data-testid="vision-size">
          {natural.w > 0 ? `${natural.w} × ${natural.h}` : '—'}
        </span>
        <Tooltip title={fullscreen ? t('imagehubT1.visionFullscreenExit') : t('imagehubT1.visionFullscreen')}>
          <Button size="small" type="text"
            icon={fullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            aria-label={fullscreen ? t('imagehubT1.visionFullscreenExit') : t('imagehubT1.visionFullscreen')}
            disabled={!src} onClick={() => void toggleFullscreen()} data-testid="vision-fullscreen" />
        </Tooltip>
      </div>

      {!src ? (
        <div className="ig-stage-empty" data-testid="vision-stage-empty">
          {t('imagehubT1.visionStageEmpty')}
        </div>
      ) : (
        <div className="ig-stage-body">
          <div className="ig-stage-corner" aria-hidden />
          {/* 上标尺：水平刻度 */}
          <div className="ig-stage-ruler ig-stage-ruler-h" aria-hidden>
            {hTicks.ticks.map((tk) => (
              <span key={`h${tk.label}`} className="ig-stage-tick" style={{ left: RULER + tk.pos }}>
                <i />
                <em>{tk.label}</em>
              </span>
            ))}
          </div>
          {/* 左标尺：垂直刻度 */}
          <div className="ig-stage-ruler ig-stage-ruler-v" aria-hidden>
            {vTicks.ticks.map((tk) => (
              <span key={`v${tk.label}`} className="ig-stage-tick" style={{ top: RULER + tk.pos }}>
                <i />
                <em>{tk.label}</em>
              </span>
            ))}
          </div>
          <div
            ref={viewportRef}
            className={`ig-stage-viewport${drag ? ' is-panning' : ''}`}
            onWheel={onWheel}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerLeave={() => { setCursor(null); setDrag(null) }}
            onDoubleClick={() => (view.zoom === 1 ? fitNow() : actualSize())}
            data-testid="vision-stage-viewport"
          >
            <img
              src={src}
              alt={alt}
              draggable={false}
              onLoad={(e) => {
                const img = e.currentTarget
                if (natural.w <= 0 && img.naturalWidth > 0) setNatural({ w: img.naturalWidth, h: img.naturalHeight })
              }}
              style={{
                transform: `translate(${view.x}px, ${view.y}px) scale(${view.zoom})`,
                transformOrigin: '0 0',
                width: natural.w > 0 ? natural.w : undefined,
                height: natural.h > 0 ? natural.h : undefined,
              }}
            />
          </div>
        </div>
      )}

      <div className="ig-stage-status" data-testid="vision-status">
        <span>{t('imagehubT1.visionZoomLabel', { zoom: formatZoom(appliedZoom(view.zoom)) })}</span>
        <span>{cursor ? t('imagehubT1.visionCursor', { x: cursor.x, y: cursor.y }) : t('imagehubT1.visionCursorIdle')}</span>
        <span className="ig-stage-status-right">{t('imagehubT1.visionStageHint')}</span>
      </div>
    </div>
  )
}

/** 读数用：把极小/极大值也钳在同一域（纯展示，不写回状态） */
function appliedZoom(z: number): number {
  return clampZoom(z)
}
