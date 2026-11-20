import React, { useId } from 'react'
import { ThunderboltOutlined } from '@ant-design/icons'
import { useT } from '../../gaea/lib/i18n'

/** SettingsSection — 设置中心统一信息卡。
 * instant 标记设置项是否即时生效；icon 为当前任务域的识别符。
 * 标题/说明/控件改为稳定语义结构，具体视觉由 settings-page.css 统一管理。 */
const SettingsSection: React.FC<{ title: React.ReactNode; desc?: string; instant?: boolean; icon?: React.ReactNode; children: React.ReactNode; noMargin?: boolean }> = ({ title, desc, instant, icon, children, noMargin }) => {
  const t = useT()
  const titleId = useId()
  return (
  <div
    className={`v3-card settings-section${noMargin ? ' is-flush' : ''}`}
    role="group"
    aria-labelledby={titleId}
  >
    <div className="settings-section__head">
      {icon && (
        <span className="settings-section__icon" aria-hidden="true">{icon}</span>
      )}
      <div className="settings-section__heading">
        <h3 id={titleId} className="settings-section__title"><strong>{title}</strong></h3>
        {desc && <p className="settings-section__desc">{desc}</p>}
      </div>
      {instant && (
        <span className="settings-section__instant">
          <ThunderboltOutlined aria-hidden="true" style={{ fontSize: 10 }} />
          <span>{t('settings.instantBadge')}</span>
        </span>
      )}
    </div>
    <div className="settings-section__body">{children}</div>
  </div>
  )
}

export default SettingsSection
