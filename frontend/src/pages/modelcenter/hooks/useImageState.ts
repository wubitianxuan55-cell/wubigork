/**
 * useImageState — 模型中心「图片生成」状态 Hook（T6-6.4 UI 拆分）
 *
 * 归集图片后端（xAI/ComfyUI）的全部状态：后端选择、ComfyUI URL/路径、
 * 保存目录、图片模型、启停状态与保存/切换处理。
 */
import { useCallback, useEffect, useState } from 'react'
import { message } from 'antd'
import { getImageBackendInfo, setImageBackend as setImageBackendAPI } from '../../../api/settings'
import { startComfyUI, stopComfyUI, getComfyUIStatus } from '../../../api/image'

export interface ImageState {
  imageBackend: string
  setImageBackend: (v: string) => void
  comfyUIURL: string
  comfyUIPath: string
  comfyUIPythonPath: string
  imageModel: string
  setImageModel: (v: string) => void
  imageSaveDir: string
  setImageSaveDir: (v: string) => void
  imageBackendSaving: boolean
  comfyStatus: { running: boolean; port: number }
  comfyBusy: boolean
  handleToggleComfy: () => Promise<void>
  handleSaveImageBackend: () => Promise<void>
}

export function useImageState(): ImageState {
  const [imageBackend, setImageBackend] = useState('xai')
  const [comfyUIURL, setComfyUIURL] = useState('http://127.0.0.1:8188')
  const [imageSaveDir, setImageSaveDir] = useState('')
  const [imageModel, setImageModel] = useState('krea2')
  const [comfyUIPath, setComfyUIPath] = useState('')
  const [comfyUIPythonPath, setComfyUIPythonPath] = useState('')
  const [imageBackendSaving, setImageBackendSaving] = useState(false)
  const [comfyStatus, setComfyStatus] = useState<{ running: boolean; port: number }>({ running: false, port: 0 })
  const [comfyBusy, setComfyBusy] = useState(false)

  const loadImageBackend = useCallback(async () => {
    try {
      const cfg: any = await getImageBackendInfo()
      if (cfg?.backend) setImageBackend(cfg.backend)
      if (cfg?.image_model || cfg?.model) setImageModel(cfg.image_model || cfg.model)
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (_) {}
    try {
      const st: any = await getComfyUIStatus()
      if (st) {
        const port = typeof st.port === 'number'
          ? st.port
          : st.url ? Number(String(st.url).split(':').pop()) || 8188 : 0
        setComfyStatus({ running: !!st.running, port })
        if (st.url) setComfyUIURL(st.url)
      }
    } catch (_) {}
  }, [])

  useEffect(() => { void loadImageBackend() }, [loadImageBackend])

  const handleToggleComfy = async () => {
    setComfyBusy(true)
    try {
      if (comfyStatus.running) { await stopComfyUI(); setComfyStatus({ running: false, port: 0 }) }
      else { await startComfyUI(); setComfyStatus({ running: true, port: 8188 }) }
      const st: any = await getComfyUIStatus()
      if (st) {
        const port = typeof st.port === 'number' ? st.port : 8188
        setComfyStatus({ running: !!st.running, port })
        message.success(st.running ? 'ComfyUI 已启动' : 'ComfyUI 已停止')
      } else {
        message.success(comfyStatus.running ? 'ComfyUI 已停止' : 'ComfyUI 已启动')
      }
    } catch (err: any) { message.error(err?.message || '操作失败') }
    finally { setComfyBusy(false) }
  }

  const handleSaveImageBackend = async () => {
    setImageBackendSaving(true)
    try { await setImageBackendAPI(imageBackend, comfyUIURL, imageModel, imageSaveDir); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
    finally { setImageBackendSaving(false) }
  }

  return {
    imageBackend, setImageBackend,
    comfyUIURL, comfyUIPath, comfyUIPythonPath,
    imageModel, setImageModel,
    imageSaveDir, setImageSaveDir,
    imageBackendSaving, comfyStatus, comfyBusy,
    handleToggleComfy, handleSaveImageBackend,
  }
}
