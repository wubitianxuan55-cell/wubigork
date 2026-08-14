// ChatPage 拆分产物：欢迎屏建议卡（行为零变化，T6-10.1）
import React from 'react'

/** 欢迎屏建议卡：键盘可达 + 焦点可见 */
export const SuggestionCard: React.FC<{ s: { icon: React.ReactNode; label: string; desc: string }; onClick: () => void }> = ({ s, onClick }) => (
  <div
    role="button"
    tabIndex={0}
    className="chat-suggestion-card"
    onClick={onClick}
    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
  >
    <div className="chat-suggestion-icon">{s.icon}</div>
    <div className="chat-suggestion-label">{s.label}</div>
    <div className="chat-suggestion-desc">{s.desc}</div>
  </div>
)
