// useCharFilters — 角色页筛选面：四筛选状态 + 按项目持久化（restore/write
// 两 effect）。2026-10-04 自 CharacterPage.tsx hook 分域首刀逐字搬迁。
import { useEffect, useState } from 'react'
import { useAppStore } from '../../stores/appStore'

const CHAR_FILTER_KEY = 'gaea.novel.charFilters.'

interface CharFilterState {
  gender: string
  role: string
  status: string
  org: string
}

function readCharFilters(projectPath: string): CharFilterState {
  try {
    const raw = localStorage.getItem(CHAR_FILTER_KEY + projectPath)
    if (!raw) return { gender: '', role: '', status: '', org: '' }
    const value = JSON.parse(raw) as CharFilterState
    return value || { gender: '', role: '', status: '', org: '' }
  } catch {
    return { gender: '', role: '', status: '', org: '' }
  }
}

function writeCharFilters(projectPath: string, state: CharFilterState) {
  try {
    if (projectPath) localStorage.setItem(CHAR_FILTER_KEY + projectPath, JSON.stringify(state))
  } catch { /* ignore */ }
}

export function useCharFilters() {
  const projectPath = useAppStore(s => s.projectPath)
  const [filterGender, setFilterGender] = useState<string>('')
  const [filterRole, setFilterRole] = useState<string>('')
  const [filterStatus, setFilterStatus] = useState<string>('')
  const [filterOrg, setFilterOrg] = useState<string>('')

  // 项目切换时恢复/重置筛选条件
  useEffect(() => {
    const saved = readCharFilters(projectPath)
    setFilterGender(saved.gender || '')
    setFilterRole(saved.role || '')
    setFilterStatus(saved.status || '')
    setFilterOrg(saved.org || '')
  }, [projectPath])

  // 筛选变化按项目记忆
  useEffect(() => {
    if (!projectPath) return
    writeCharFilters(projectPath, {
      gender: filterGender,
      role: filterRole,
      status: filterStatus,
      org: filterOrg,
    })
  }, [projectPath, filterGender, filterRole, filterStatus, filterOrg])

  return {
    filterGender, setFilterGender,
    filterRole, setFilterRole,
    filterStatus, setFilterStatus,
    filterOrg, setFilterOrg,
  }
}
