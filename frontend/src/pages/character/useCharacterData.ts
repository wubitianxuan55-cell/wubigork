// useCharacterData — 角色页数据面（characters/organizations/relationships/projectRefs
// 加载、代际守卫、旧项目同步事件、同步按钮）。2026-10-04 自 CharacterPage.tsx
// hook 分域首刀逐字搬迁（controller.ts 配方：逻辑/顺序/deps 零改动）。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { message } from 'antd'
import { getCharacters } from '../../components/novel/api/character'
import { listProjectCharacters, syncProjectCharacters } from '../../api/characterlib'
import { errText } from '../../utils/errText'
import { useAppStore } from '../../stores/appStore'
import type { CharacterData, OrganizationData, RelationshipData } from '../../types'

export function useCharacterData() {
  const projectPath = useAppStore(s => s.projectPath)
  const [characters, setCharacters] = useState<CharacterData[]>([])
  const [organizations, setOrganizations] = useState<OrganizationData[]>([])
  const [relationships, setRelationships] = useState<RelationshipData[]>([])
  const [projectRefs, setProjectRefs] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [syncing, setSyncing] = useState(false)
  const dataLoadToken = useRef(0)
  const refsLoadToken = useRef(0)

  const loadData = useCallback(async () => {
    const token = ++dataLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    setLoading(true)
    try {
      const data = await getCharacters()
      if (token !== dataLoadToken.current || requestedPath !== useAppStore.getState().projectPath) return
      setCharacters(data.characters || [])
      setOrganizations(data.organizations || [])
      setRelationships(data.relationships || [])
      setLoadError('')
    } catch (err) {
      console.error('[CharacterPage] loadData:', err)
      if (token === dataLoadToken.current) setLoadError(errText(err, '角色数据读取失败'))
    } finally {
      if (token === dataLoadToken.current) setLoading(false)
    }
  }, [])

  const loadRefs = useCallback(async () => {
    const token = ++refsLoadToken.current
    if (!projectPath) { setProjectRefs(new Set()); return }
    try {
      const refs = await listProjectCharacters()
      if (token !== refsLoadToken.current || projectPath !== useAppStore.getState().projectPath) return
      setProjectRefs(new Set(refs.map(r => r.characterId)))
    } catch (_) {
      if (token === refsLoadToken.current && projectPath === useAppStore.getState().projectPath) setProjectRefs(new Set())
    }
  }, [projectPath])

  useEffect(() => {
    setCharacters([]); setOrganizations([]); setRelationships([])
    setLoadError('')
    if (projectPath) { loadData(); loadRefs() }
  }, [projectPath, loadData, loadRefs])

  // 旧项目数据检测：characters.json 里有未入库角色（未关联）
  const unimported = useMemo(
    () => characters.filter(c => !projectRefs.has(c.id)),
    [characters, projectRefs],
  )

  const refreshAll = useCallback(async () => {
    await loadData()
    await loadRefs()
  }, [loadData, loadRefs])

  // 角色库侧移出本书后，已挂载的本面板保持同步刷新（MainLayout 不销毁组件）
  useEffect(() => {
    const handler = () => { refreshAll() }
    window.addEventListener('gaea-project-chars-changed', handler)
    return () => window.removeEventListener('gaea-project-chars-changed', handler)
  }, [refreshAll])

  const handleSync = async () => {
    setSyncing(true)
    try {
      await syncProjectCharacters()
      await loadData()
      message.success('已把本书引用的角色同步到 characters.json')
    } catch (err: unknown) {
      message.error(errText(err, '同步失败'))
    } finally {
      setSyncing(false)
    }
  }

  return {
    characters, setCharacters,
    organizations, setOrganizations,
    relationships, setRelationships,
    projectRefs, loading, setLoading, loadError, unimported, syncing,
    loadData, loadRefs, refreshAll, handleSync,
  }
}
