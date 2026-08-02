import React from 'react'
import { KnowledgePanel } from '../gaea/components/KnowledgePanel'
import { LocaleProvider } from '../gaea/lib/i18n'
import FeatureModelBar from '../components/FeatureModelBar'
import '../gaea/styles.css'
import '../gaea/tailwind.css'

// 知识库板块：独立页面，显式工程知识条目（规范/案例/经验）的管理与检索。
// 与记忆系统（隐式跨会话事实，侧边栏 MemoryPanel）明确区分——
// 知识库是可编辑、可分类、可全文检索的显式知识，服务于办公/方案/轻语等板块
// 通过 knowledge_search/knowledge_add 工具（MCP 式）按需调用。
// LocaleProvider 必须包在最外层（组件自身调用 useT）。
function KnowledgePage() {
  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', position: 'relative' }}>
      <div style={{ flex: 1, minHeight: 0 }}>
        <LocaleProvider>
          <KnowledgePanel variant="page" onClose={() => {}} />
        </LocaleProvider>
      </div>
      {/* 绑定模型卡（左下角浮动） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="gaea" label="知识库" />
      </div>
    </div>
  )
}

export default KnowledgePage
