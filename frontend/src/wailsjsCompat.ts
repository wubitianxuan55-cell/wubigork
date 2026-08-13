/**
 * wailsjsCompat — S2-3「App 绑定面拆分」兼容层
 *
 * 后端绑定面已从单一 App 对象拆为 10 个板块门面（CoreB/OfficeB/MemoryB/
 * CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，见 internal/app/
 * bindings_*.go）。wails 重新生成 wailsjs/go/app/<门面>.js 后，旧代码
 * `import * as App from '<...>/src/wailsjsCompat'` 不再存在。
 *
 * 本模块把新门面的导出重新合并为旧命名空间形态：旧 import 路径统一改为
 * `'<...>/src/wailsjsCompat'`（等价深度），`App.GetXxx()` 调用不变。
 *
 * 注意：本模块依赖 wails 生成的 wailsjs 模块（wails build 时再生成），
 * 若缺失请先执行 `wails build -s` 重新生成绑定。
 */

export * from '../wailsjs/go/app/CoreB'
export * from '../wailsjs/go/app/OfficeB'
export * from '../wailsjs/go/app/MemoryB'
export * from '../wailsjs/go/app/CostB'
export * from '../wailsjs/go/app/ModelB'
export * from '../wailsjs/go/app/VoiceB'
export * from '../wailsjs/go/app/ChatB'
export * from '../wailsjs/go/app/NovelB'
export * from '../wailsjs/go/app/ImageB'
export * from '../wailsjs/go/app/CharlibB'
