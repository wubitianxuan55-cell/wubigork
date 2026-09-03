// wxFocus.ts — 角色库「创建青鸟助手」→ 青鸟页的深链焦点传递（v4.51 B 线）。
//
// 为什么用 sessionStorage 而不是内存单例：导航后 WeixinPage lazy 挂载，事件
// （NAVIGATE）先于订阅——写入方（CharacterCard）与消费方（WeixinPage）生命
// 周期错位；且壳层 keepAlive 页面只切 display 不重挂载，内存单例在模块重求值
// （HMR/刷新）时即丢失。sessionStorage 把「待聚焦助手 id」挂在标签页会话上，
// 写入方先落盘、消费方挂载/复明后再取，时序彻底解耦。
//
// 链路：CharacterCard 创建成功 → setWxFocusAssistant(新助手 id) →
// emitFrontendEvent(NAVIGATE, { page: 'weixin' }) → WeixinPage 挂载读取 +
// NAVIGATE 监听两条路 takeWxFocusAssistant() → 命中列表则选中；未绑定则直接
// 打开其扫码绑定流（读后即清，两路天然互斥不重复消费）。

/** sessionStorage 键（导出供测试对齐，避免两处魔法串漂移） */
export const WX_FOCUS_KEY = 'gaea-wx-focus-assistant'

/** 写入待聚焦的青鸟助手 id（创建成功后、派发 NAVIGATE 前调用）。 */
export function setWxFocusAssistant(id: string): void {
  try {
    sessionStorage.setItem(WX_FOCUS_KEY, id)
  } catch {
    /* 存储不可用（隐私模式等）时静默：只损失自动聚焦，不影响创建本身 */
  }
}

/** 取出并清除待聚焦助手 id（读后即清，防陈旧焦点影响后续正常进入）。 */
export function takeWxFocusAssistant(): string | null {
  try {
    const id = sessionStorage.getItem(WX_FOCUS_KEY)
    if (id !== null) sessionStorage.removeItem(WX_FOCUS_KEY)
    return id
  } catch {
    return null
  }
}
