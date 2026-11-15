import WenxueApp from '../wenxue/App'
import { LocaleProvider } from '../wenxue/lib/i18n'
import '../gaea/styles.css'
import '../gaea/tailwind.css'
import '../gaea/redesign.css'
import '../wenxue/wenxue-ui.css'
import '../wenxue/wenxue-conversation.css'
import '../wenxue/wenxue-lingxi-skin.css'
import '../wenxue/wenxue-sidebar.css'

// 文学板块（2026-11-15）：办公板块的完整前端拷贝（frontend/src/wenxue/），全共享
// 引擎与会话流（同一批 OfficeB 等绑定），仅 UI 层独立——4 个皮肤 css 拷贝件经
// .wx-root 作用域隔离（keyframes 同步改名 wx-*），keepAlive 双挂载互不污染；
// 二次改造一律在本目录内进行，不得反向影响办公板块。
// LocaleProvider 必须包在 WenxueApp 外层：App 顶层自身调用 useT()，Provider 若只
// 放在其 return JSX 内盖不住组件自身（同 GaeaPage 的运行时抛错教训）。
function WenxuePage() {
  return (
    <div
      className="wx-root"
      style={{ height: '100%', display: 'flex', flexDirection: 'column', position: 'relative' }}
    >
      <div className="v3-zone" style={{ flex: 1, minHeight: 0 }}>
        <LocaleProvider>
          <WenxueApp />
        </LocaleProvider>
      </div>
    </div>
  )
}

export default WenxuePage
