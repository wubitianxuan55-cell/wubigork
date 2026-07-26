# wubigork 移动端远程操控 — 设计文档

> 日期：2026-07-02 | 版本：v1.0  
> 定位：手机作为桌面端的**远程遥控器/第二屏**，桌面端仍是主控工作台

---

## 1. 架构概览

```
┌─ 桌面端 (Wails WebView2) ───────────────────────────────────┐
│  React 19 + Vite   (dev: localhost:5173 / prod: embed dist/) │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ M3 Theme   │  │ Zustand      │  │ Responsive Layout    │ │
│  │ System     │  │ Store        │  │ (Container Queries)  │ │
│  └────────────┘  └──────────────┘  └──────────────────────┘ │
│                              │                                │
│                     window.go.app.App.*()                     │
└──────────────────────────────┼────────────────────────────────┘
                               │
┌─ Go Backend ─────────────────┼────────────────────────────────┐
│  internal/app/ (22 handlers) ┘                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  NEW: internal/mobile/                                    │ │
│  │  - HTTPServer  :8080 (独立开关，SettingsPage 控制)       │ │
│  │  - ImageProxy  (AI 图片路径直出)                          │ │
│  │  - QRCode      (局域网 IP:Port 二维码)                    │ │
│  │  - API 适配层  (复用现有 app.App 方法)                    │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬─────────────────────────────────┘
                               │ HTTP (局域网)
                               ▼
┌─ 手机浏览器 ─────────────────────────────────────────────────┐
│  同一套 React SPA (dist/ 静态资源)                            │
│  · 底部 TabBar + Drawer 导航                                  │
│  · M3 触控优化 (min 44px 触控区, 涟漪反馈)                   │
│  · 渐进增强 (3D → 2D, Canvas → Grid)                         │
│  · 视觉与桌面端共享 M3 主题                                   │
└──────────────────────────────────────────────────────────────┘
```

### 核心原则

| 原则 | 说明 |
|------|------|
| **一套代码，两种形态** | 不 fork 项目，同一个 React app 通过 Container Queries + 少量 useMediaQuery 适配两端 |
| **桌面端是主机** | 数据在桌面端，AI 调用在桌面端，手机只是远程浏览器。手机不存状态，关掉重开即可 |
| **Go 后端不动业务逻辑** | Wails 绑定原样工作；新增 `internal/mobile/` 只做 HTTP 暴露和薄 API 适配 |
| **静态资源零额外构建** | `dist/` 同时被 Wails embed 和 HTTP FileServer 使用 |
| **图片直通** | AI 生图存本地 `novels/<project>/images/`，HTTP serve 目录，不转 base64 |

---

## 2. M3 主题系统

### 2.1 Tonal Palette 算法

Material Design 3 的核心：一个种子色 → 算法生成完整 13 级色调调色板。轻量内联实现，不引入外部 npm 包。

```
种子色 (Primary Hue)  →  Tonal Palette (0, 10, 20, ... 95, 99, 100)
                      →  映射到 Ant Design ConfigProvider theme token
                      →  同步为 CSS 变量 → :root → 全局生效
```

### 2.2 5 套主题

| 主题名 | 种子色 | M3 调性 | 场景 |
|--------|--------|---------|------|
| `nightGreen` | #4CAF50 | Tertiary 冷绿 | 深夜写作 |
| `starPurple` | #7C4DFF | Primary 紫 | 创意灵感 |
| `midnightInk` (默认) | #6750A4 | M3 Default | 日常 |
| `warmAmber` | #FF9800 | Secondary 暖金 | 温馨阅读 |
| `minimalGray` | #64748B | Neutral 灰蓝 | 专注 |

### 2.3 代码变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/theme/m3-palette.ts` | **新建** | Tonal Palette 生成函数（纯函数，~80 行） |
| `frontend/src/store/appStore.ts` | **重写 ThemeTokens** | 对齐 M3 token 命名（surface, surfaceContainer, onSurface, outline 等） |
| `frontend/src/App.tsx` | **重写** ConfigProvider | 用 M3 tokens 驱动 Ant Design |
| `frontend/src/index.css` | **重写** 全局变量 | `--md-sys-color-*` M3 标准命名，替换 Layered Glass 变量 |
| `frontend/src/index.css` | **删除** | `.glass-panel`, `.glass-card`, `.inset-editor` 等 Layered Glass 工具类 |
| `frontend/src/index.css` | **新增** | `.md-surface`, `.md-surface-container`, `.md-elevation-1/2/3` 等 M3 Surface 工具类 |

### 2.4 不做

- ❌ 不引入 `@material/material-color-utilities`
- ❌ 不做 Dynamic Color（壁纸取色）
- ❌ 不改 Ant Design 组件源码 — 纯 token 驱动

---

## 3. 响应式布局

### 3.1 断点系统（移动优先，Container Query 为主）

```typescript
const breakpoints = {
  compact:  '(max-width: 599px)',                              // 手机竖屏
  medium:   '(min-width: 600px) and (max-width: 899px)',       // 手机横屏 / 小平板
  expanded: '(min-width: 900px)',                               // 桌面端（现有）
} as const;
```

**Container Query 优先的理由**：桌面端 Sidebar 折叠、右侧面板打开都会改变内容区宽度。`@container` 让组件自适应自身容器——同一组件在桌面端侧边被挤压后自动切换为移动端布局，无需重复判断。

### 3.2 导航体系

```
桌面端 (≥900px)                    移动端 (<900px)
┌──────────────────────┐          ┌──────────────────────────────┐
│ Sidebar   ┆  内容区   │          │  AppBar (页面标题 + Drawer)  │
│ (230px)   ┆           │          ├──────────────────────────────┤
│ 首页      ┆           │          │                              │
│ 世界观    ┆           │          │         内容区               │
│ 角色      ┆           │          │                              │
│ 大纲      ┆           │          ├──────────────────────────────┤
│ 章节      ┆           │          │ 首页 │大纲│章节│角色│世界观  │
│ Canvas    ┆           │          │         Bottom TabBar        │
│ 导出 ─    ┆           │          └──────────────────────────────┘
│ 设置 ═    ┆           │
└──────────────────────┘
```

- **桌面端**：保留左侧 230px Sidebar（改造为 M3 NavigationRail 视觉风格）
- **移动端**：
  - **Bottom TabBar**：5 个核心标签 — 首页 / 大纲 / 章节 / 角色 / 世界观
  - **顶部 AppBar**：当前页面标题 + 右侧汉堡按钮打开 Drawer
  - **Drawer**：次要入口 — Canvas / 导出 / 图表分析 / 设置 / 图片生成
- **实现**：`MainLayout.tsx` 用 `useMediaQuery` 选择渲染 Sidebar 或 TabBar+Drawer

### 3.3 各页面适配

| 页面 | 桌面端 | 移动端 |
|------|--------|--------|
| **HomePage** | 3 列 Bento Grid | 单列堆叠，关键指标 2×2 grid |
| **ChapterPage** | 左编辑区 / 右 AI 面板 (SplitPane) | 全屏编辑器 + 底部 Sheet 拉出 AI 面板；工具栏吸附键盘上方 |
| **OutlinePage** | 树形列表 + 右侧详情面板 | 全屏列表 → 点击进入详情；长按触发拖拽排序 |
| **CharacterPage** | 表格 + 侧边详情抽屉 | 卡片列表 + 底部 Sheet 编辑表单 |
| **WorldviewPage** | 左分类树 + 右条目列表 | 可折叠 Accordion 分类 + 条目卡片 |
| **CanvasPage** | 无限画布，拖拽/缩放 | 固定网格布局，不可拖动，双指缩放查看 |
| **StoryGraph** (3D) | WebGL 3D 力导向图 (Three.js + d3-force-3d) | 降级为 2D DOM/SVG 关系图 |
| **TTSPlayer** | 迷你播放条 (footer) | 全宽底部播放条，更大的控制按钮 |
| **SettingsPage** | 居中表单 600px | 全宽表单，「移动端访问」开关 + 二维码面板 |
| **ExportPage** | 多选项表单 | 单列表单 |
| **ImageGenPage** | 左参数 / 右预览 | 全屏参数 → 底部 Sheet 预览 |

### 3.4 全局交互适配

| 规则 | 实现 |
|------|------|
| **触控区 ≥ 44×44px** | 覆盖所有按钮、列表项、Tab 标签的 `min-height`/`min-width` + `padding` |
| **长按 = 右键** | 列表/卡片上 `onTouchStart`/`onTouchEnd` 计时 500ms → 触发上下文菜单 |
| **点击反馈** | M3 涟漪：`::after` 伪元素 + `@keyframes ripple`，CSS only |
| **键盘适配** | `visualViewport` API 监听 → 编辑器区域高度自动调整 |
| **横屏写作** | `screen.orientation` 检测横屏时提示旋转，编辑器进入沉浸全屏模式 |
| **滚动条隐藏** | 移动端 `scrollbar-width: none` + `::-webkit-scrollbar { display: none }` |

---

## 4. Go 后端：`internal/mobile/` 包

### 4.1 文件结构

```
internal/mobile/
├── server.go       # HTTP 服务器 + 生命周期管理
├── handlers.go     # 现有 app.App 方法的 HTTP 适配层
├── qrcode.go       # 局域网 IP 检测 + 二维码生成
└── server_test.go  # 单元测试
```

### 4.2 HTTP 路由

```
GET  /api/health          → {"status":"ok","ip":"192.168.1.5","port":8080}
GET  /api/qrcode          → 返回二维码 PNG (当前局域网 IP:Port)
GET  /api/projects        → 项目列表
GET  /api/project/:id     → 项目详情
GET  /api/chapters/:pid   → 章节列表
GET  /api/chapter/:id     → 章节内容
POST /api/*               → 写操作（通过 Wails 绑定间接调用，确保桌面端状态同步）
GET  /api/images/*        → serve novels/<project>/images/ 目录的 AI 生图
GET  /*                   → SPA fallback → dist/index.html
```

### 4.3 关键设计

- **API 复用**：`handlers.go` 薄封装 `app.App` 方法到 HTTP handler，**不重写业务逻辑**。读操作直接调用方法返回 JSON；写操作通过 Wails 事件通知桌面端同步
- **CORS**：`Access-Control-Allow-Origin: *`（局域网安全边界内）
- **IP 检测**：遍历 `net.InterfaceAddrs()`，取第一个非 loopback 的 /24 IPv4 地址
- **图片路径**：FileServer 直接 serve `novels/<project>/images/`
- **开关控制**：`Start(port) / Stop() / Status()` 三个方法，桌面端 SettingsPage 调用

### 4.4 Wails 绑定扩展

```typescript
// 新增 3 个 Go 方法暴露给前端：
StartMobileServer(port: number): Promise<{ip: string, port: number}>
StopMobileServer(): Promise<void>
GetMobileServerStatus(): Promise<{running: boolean, ip: string, port: number}>
```

### 4.5 设置页 UI 原型

```
┌─ 设置 ──────────────────────────────┐
│  ...                                 │
│  ─── 移动端远程访问 ───              │
│                                      │
│  [Switch] 启用移动端访问             │
│  端口: [8080               ]         │
│                                      │
│  ┌──────────────────────┐            │
│  │                      │            │
│  │    ████████████      │            │
│  │    ██        ██      │            │
│  │    ██ QR Code ██     │  ← 二维码  │
│  │    ██        ██      │            │
│  │    ████████████      │            │
│  │                      │            │
│  │ http://192.168.1.5:8080 │          │
│  └──────────────────────┘            │
│                                      │
│  ⚠ 请确保手机和电脑在同一局域网      │
└──────────────────────────────────────┘
```

---

## 5. 组件改造原则

每个页面遵循 3 步改造法：

1. **包裹响应式容器**：最外层 `<div>` 加 `container-type: inline-size`，子组件用 `@container` 查询切换布局
2. **拆分大文件**：ChapterPage (52KB) → `ChapterPageDesktop.tsx` + `ChapterPageMobile.tsx`，共用逻辑抽到 `useChapterPage.ts` hook
3. **条件渲染降级路径**：`useMediaQuery('(max-width: 899px)')` 在关键分支 switch 3D→2D、Canvas→Grid

### 文件拆分约定

```
frontend/src/pages/
├── ChapterPage.tsx           # 入口：判断设备，选择渲染
├── ChapterPageDesktop.tsx    # 桌面端组件（现有代码移入）
├── ChapterPageMobile.tsx     # 移动端组件（全新）
└── useChapterPage.ts         # 共享逻辑 hook（状态、API 调用、事件处理）

frontend/src/components/
├── MobileTabBar.tsx           # 底部 TabBar
├── MobileDrawer.tsx           # 侧滑 Drawer
├── MobileSheet.tsx            # 底部 Sheet 通用组件
├── LongPressable.tsx          # 长按检测 HOC
└── AppBar.tsx                 # 顶部标题栏
```

---

## 6. 实施阶段

| Phase | 内容 | 预估工作量 |
|-------|------|-----------|
| **Phase 1** | M3 主题系统：palette 生成 + token 重写 + CSS 全局替换 | 1-2 天 |
| **Phase 2** | 响应式布局基础：导航体系 (TabBar/Drawer/AppBar) + 容器工具类 + 全局交互（涟漪、长按） | 2-3 天 |
| **Phase 3** | Go 后端 `internal/mobile/`：HTTP 服务 + 二维码 + API 适配 | 1 天 |
| **Phase 4** | 页面适配：HomePage / ChapterPage / OutlinePage（核心流程） | 2-3 天 |
| **Phase 5** | 页面适配：CharacterPage / WorldviewPage / CanvasPage / StoryGraph 降级 | 1-2 天 |
| **Phase 6** | 剩余页面 + 图片流 + TTS + 设置页移动端开关 | 1 天 |
| **Phase 7** | 全量测试 + Bug 修复 + 横屏/键盘适配打磨 | 1-2 天 |

**总计**：约 9-14 天

---

## 7. 不做的事

| 项目 | 原因 |
|------|------|
| Service Worker / 离线缓存 | PWA 留到后续迭代 |
| 推送通知 | 远程操控场景手机在线才有效，离线无意义 |
| 新的 npm 依赖 | 二维码 Go 端生成；涟漪 CSS only；不引入 @mui, react-router |
| Go 核心业务修改 | 移动端纯 HTTP 薄适配层 |
| WebSocket 实时同步 | HTTP 请求即可满足远程操控需求 |
| 手机端独立登录 | 复用桌面端已登录状态，无额外认证 |
| iOS/Android 原生包 | 纯 Web，浏览器访问 |
