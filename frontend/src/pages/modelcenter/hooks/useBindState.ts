/**
 * useBindState — 模型中心「功能绑定/角色库剧照」状态 Hook（T6-6.4 UI 拆分）
 *
 * 归集功能模型绑定（chat/novel/office/gaea/characterlib/routine）、生效路由
 * 与角色库剧照（后端/模型）的全部状态与保存处理。engines 由引擎 Hook 传入，
 * 用于计算剧照的可用模型选项（跨分类共享）。
 */
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { message } from 'antd'
import * as App from '../../../wailsjsCompat'
import { getPortraitConfig, setPortraitConfig } from '../../../api/image'
import { FEATURES, imageModelOptionsFor } from '../utils'
import type { EngineConfig } from '../../../api/engines'

export interface BindState {
  featureCfg: Record<string, { engine: string; model: string }>
  featureDraft: Record<string, { engine: string; model: string }>
  setFeatureDraft: Dispatch<SetStateAction<Record<string, { engine: string; model: string }>>>
  featureEnabled: Record<string, boolean>
  modelRoutes: Record<string, { engine: string; model: string; source: string }>
  portraitCfg: { backend: string; model: string }
  portraitDraft: { backend: string; model: string }
  setPortraitDraft: Dispatch<SetStateAction<{ backend: string; model: string }>>
  portraitModelOptions: { label: string; value: string }[]
  portraitSaving: boolean
  loadFeatureCfg: () => Promise<void>
  refreshRoutes: () => Promise<void>
  handleSaveFeature: (key: string) => Promise<void>
  handleToggleFeatureEnabled: (key: string, enabled: boolean) => Promise<void>
  handleSavePortrait: () => Promise<void>
}

export function useBindState(engines: EngineConfig[]): BindState {
  const [featureCfg, setFeatureCfg] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureDraft, setFeatureDraft] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureEnabled, setFeatureEnabled] = useState<Record<string, boolean>>({})
  const [modelRoutes, setModelRoutes] = useState<Record<string, { engine: string; model: string; source: string }>>({})
  const [portraitCfg, setPortraitCfg] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitDraft, setPortraitDraft] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitSaving, setPortraitSaving] = useState(false)

  // 当前生效路由（后端 routeModel 降级链结果：feature / global / fallback）
  const refreshRoutes = useCallback(async () => {
    const bind = (window as any).go?.app?.App
    if (!bind?.GetModelRoute) return
    const next: Record<string, { engine: string; model: string; source: string }> = {}
    for (const key of ['chat', 'novel', 'office', 'gaea', 'characterlib', 'routine']) {
      try {
        next[key] = JSON.parse(await bind.GetModelRoute(key))
      } catch { /* 单功能失败忽略 */ }
    }
    setModelRoutes(next)
  }, [])

  const loadFeatureCfg = useCallback(async () => {
    try {
      const cfg: Record<string, { engine: string; model: string }> = {}
      const en: Record<string, boolean> = {}
      for (const f of FEATURES) {
        const keys = [f.key, ...(f.mergeKeys || [])]
        let engine = ''
        let model = ''
        let on = true
        for (const k of keys) {
          const r: any = await App.GetFeatureModel(k)
          if (!engine && r?.engine) { engine = r.engine; model = r.model || '' }
          let e = true
          try { e = !!(await App.GetFeatureModelEnabled(k)) } catch (_) { e = true }
          on = on && e
        }
        cfg[f.key] = { engine, model }
        en[f.key] = on
      }
      setFeatureCfg(cfg)
      setFeatureEnabled(en)
      setFeatureDraft(JSON.parse(JSON.stringify(cfg)))
    } catch (_) {}
  }, [])

  useEffect(() => { void refreshRoutes() }, [refreshRoutes])
  useEffect(() => { void loadFeatureCfg() }, [loadFeatureCfg])

  // 角色库剧照配置读取
  useEffect(() => {
    (async () => {
      try {
        const p = await getPortraitConfig()
        setPortraitCfg(p)
        setPortraitDraft(p)
      } catch (err: any) {
        message.error(err?.message || '读取剧照配置失败')
      }
    })()
  }, [])

  const handleSaveFeature = async (key: string) => {
    const d = featureDraft[key]
    if (!d?.engine || !d?.model) { message.warning('请先选择引擎和模型'); return }
    const f = FEATURES.find(x => x.key === key)
    try {
      await App.SetFeatureModel(key, d.engine, d.model)
      for (const k of f?.mergeKeys || []) {
        await App.SetFeatureModel(k, d.engine, d.model)
      }
      message.success(`${f?.label || key}模型已绑定并持久化`)
      loadFeatureCfg()
      refreshRoutes()
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    }
  }

  const handleToggleFeatureEnabled = async (key: string, enabled: boolean) => {
    const f = FEATURES.find(x => x.key === key)
    try {
      await App.SetFeatureModelEnabled(key, enabled)
      for (const k of f?.mergeKeys || []) {
        await App.SetFeatureModelEnabled(k, enabled)
      }
      message.success(`${f?.label || key}功能模型已${enabled ? '启用' : '停用'}`)
      setFeatureEnabled(prev => ({ ...prev, [key]: enabled }))
      loadFeatureCfg()
    } catch (err: any) {
      message.error(err?.message || '操作失败')
    }
  }

  const handleSavePortrait = async () => {
    setPortraitSaving(true)
    try {
      await setPortraitConfig(portraitDraft.backend, portraitDraft.model)
      setPortraitCfg({ ...portraitDraft })
      message.success(
        portraitDraft.backend
          ? `角色库剧照已绑定：${portraitDraft.backend} / ${portraitDraft.model || '跟随绘梦'}`
          : '角色库剧照已恢复为跟随绘梦',
      )
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    } finally {
      setPortraitSaving(false)
    }
  }

  // 角色库剧照独立后端/模型选项（空 = 跟随绘梦）
  const portraitModelOptions = useMemo(() => {
    const b = portraitDraft.backend
    if (!b) return [{ label: '跟随绘梦', value: '' }]
    return [
      { label: '跟随绘梦', value: '' },
      ...imageModelOptionsFor(b, engines, portraitDraft.model),
    ]
  }, [portraitDraft.backend, portraitDraft.model, engines])

  return {
    featureCfg,
    featureDraft, setFeatureDraft,
    featureEnabled,
    modelRoutes,
    portraitCfg,
    portraitDraft, setPortraitDraft,
    portraitModelOptions,
    portraitSaving,
    loadFeatureCfg,
    refreshRoutes,
    handleSaveFeature,
    handleToggleFeatureEnabled,
    handleSavePortrait,
  }
}
