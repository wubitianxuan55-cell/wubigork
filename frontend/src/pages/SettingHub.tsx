import React, { useState } from 'react'
import { Segmented } from 'antd'
import { FileTextOutlined, UserOutlined } from '@ant-design/icons'

// SettingHub — 「设定」页签薄壳（2026-10-27 大道至简重组：角色页并入设定，
// 页签 6→5——书架/设定/创作/阅读/原罪 = 入口/资料/生成/精修/另玩）。
// 双 pane **常驻挂载 display 切换**（沿用 NovelPage 五 pane 契约语义：隐藏 ≠ 卸载，
// CharacterPage 的 hooks 数据与草稿不因切视图丢失）；`active` 按「hub 激活且当前
// 视图」转发子页——NovelSettingPage 的 Ctrl+S 等窗口级监听仍只在真正可见时生效。
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage')) as React.ComponentType<{ active?: boolean }>
const CharacterPage = React.lazy(() => import('./CharacterPage')) as React.ComponentType<{ active?: boolean }>

export const SETTING_VIEW_KEY = 'gaea.novel.settingView'
export type SettingView = 'world' | 'chars'

function loadSettingView(): SettingView {
  try {
    const v = localStorage.getItem(SETTING_VIEW_KEY)
    if (v === 'world' || v === 'chars') return v
  } catch { /* ignore */ }
  return 'world'
}

const SettingHub: React.FC<{ active?: boolean }> = ({ active = true }) => {
  const [view, setView] = useState<SettingView>(loadSettingView)

  const changeView = (v: SettingView) => {
    setView(v)
    try { localStorage.setItem(SETTING_VIEW_KEY, v) } catch { /* ignore */ }
  }

  return (
    <div className="novel-setting-hub" data-setting-view={view}>
      <div className="novel-setting-hub-nav">
        <Segmented
          size="small"
          value={view}
          onChange={(v) => changeView(v as SettingView)}
          options={[
            { value: 'world', label: '世界观', icon: <FileTextOutlined /> },
            { value: 'chars', label: '人物', icon: <UserOutlined /> },
          ]}
        />
      </div>
      <div className="novel-setting-hub-body">
        <div className={`novel-setting-hub-pane${view === 'world' ? ' is-active' : ''}`}>
          <React.Suspense fallback={<div className="novel-tab-skeleton" aria-hidden />}>
            <NovelSettingPage active={active && view === 'world'} />
          </React.Suspense>
        </div>
        <div className={`novel-setting-hub-pane${view === 'chars' ? ' is-active' : ''}`}>
          <React.Suspense fallback={<div className="novel-tab-skeleton" aria-hidden />}>
            <CharacterPage active={active && view === 'chars'} />
          </React.Suspense>
        </div>
      </div>
    </div>
  )
}

export default SettingHub
