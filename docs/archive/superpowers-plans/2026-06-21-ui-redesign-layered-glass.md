# wubigork UI Redesign — "Layered Glass" Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform wubigork from bare Ant Design default look into a premium "Layered Glass" desktop app — glassmorphism panels, backdrop-blur, gradient accents, shadow elevation system, unified spacing

**Architecture:** Extend the existing CSS-variable + ConfigProvider theming system with new design tokens (backdrop-blur, shadows, border-weights, radius scale). Apply tokens globally via index.css utility classes + inline style replacements across all 8 pages. Zero new dependencies.

**Tech Stack:** React 19, Ant Design 6, TypeScript 6, Zustand 5, Vite 8, Wails v2

## Global Constraints

- 保留全部 4 套现有主题 (nightGreen / starPurple / warmAmber / minimalGray)
- 不引入新 npm 依赖
- 所有功能行为不变，仅改变视觉呈现
- 明暗模式均需正常工作
- 文本对比度 ≥ 4.5:1
- `prefers-reduced-motion` 需被尊重
- TypeScript 编译零错误

---

### Task 1: Extend ThemeTokens with new design variables

**Files:**
- Modify: `frontend/src/stores/appStore.ts`

**Interfaces:**
- Produces: Extended `ThemeTokens` interface with `accentRgb`, `shadowSm`, `shadowMd`, `shadowLg`, `shadowGlow`, `borderSubtle`, `bgDeep`, `bgBase`, `bgElevated`, `bgGlass`, `radiusSm`, `radiusMd`, `radiusLg`, `radiusXl`, `transitionFast`, `transitionNormal`, `transitionSlow`

- [ ] **Step 1: Extend ThemeTokens interface**

In `appStore.ts`, replace the existing `ThemeTokens` interface:

```typescript
export interface ThemeTokens {
  colorPrimary: string
  colorBgContainer: string
  colorBgLayout: string
  colorText: string
  colorTextSecondary: string
  colorBorder: string
  colorSuccess: string
  colorWarning: string
  // 新增 Layered Glass 令牌
  accentRgb: string            // e.g. "74, 222, 128" for rgba() 构建
  shadowSm: string
  shadowMd: string
  shadowLg: string
  shadowGlow: string
  borderSubtle: string
  bgDeep: string
  bgBase: string
  bgElevated: string
  bgGlass: string
  radiusSm: string
  radiusMd: string
  radiusLg: string
  radiusXl: string
  transitionFast: string
  transitionNormal: string
  transitionSlow: string
}
```

- [ ] **Step 2: Update all 4 dark theme presets with new tokens**

Replace `themePresets` object. Each preset gets the shared glass/depth tokens + unique accentRgb:

```typescript
// 共享的深色基底令牌
const sharedDarkTokens = {
  shadowSm: '0 1px 3px rgba(0,0,0,0.4)',
  shadowMd: '0 4px 12px rgba(0,0,0,0.5)',
  shadowLg: '0 8px 24px rgba(0,0,0,0.6)',
  borderSubtle: 'rgba(255,255,255,0.06)',
  bgDeep: '#08080a',
  bgBase: '#0d0d12',
  bgElevated: 'rgba(255,255,255,0.04)',
  bgGlass: 'rgba(255,255,255,0.06)',
  radiusSm: '4px',
  radiusMd: '8px',
  radiusLg: '12px',
  radiusXl: '16px',
  transitionFast: '100ms ease-out',
  transitionNormal: '150ms ease-out',
  transitionSlow: '200ms ease-out',
}

const themePresets: Record<ThemePreset, ThemeTokens> = {
  nightGreen: {
    colorPrimary: '#4ade80',
    colorBgContainer: '#1a1a1a',
    colorBgLayout: '#0d0d0d',
    colorText: '#e0e0e0',
    colorTextSecondary: '#9ca3af',
    colorBorder: '#333',
    colorSuccess: '#4ade80',
    colorWarning: '#f59e0b',
    accentRgb: '74, 222, 128',
    shadowGlow: '0 0 20px rgba(74, 222, 128, 0.15)',
    ...sharedDarkTokens,
  },
  starPurple: {
    colorPrimary: '#a78bfa',
    colorBgContainer: '#1b1a23',
    colorBgLayout: '#121118',
    colorText: '#e2e0e7',
    colorTextSecondary: '#8b8a9a',
    colorBorder: '#2d2b3a',
    colorSuccess: '#6ee7b7',
    colorWarning: '#fbbf24',
    accentRgb: '167, 139, 250',
    shadowGlow: '0 0 20px rgba(167, 139, 250, 0.15)',
    ...sharedDarkTokens,
  },
  warmAmber: {
    colorPrimary: '#f59e0b',
    colorBgContainer: '#1f1b15',
    colorBgLayout: '#14110c',
    colorText: '#e8dcc8',
    colorTextSecondary: '#a6987a',
    colorBorder: '#3d3528',
    colorSuccess: '#a3e635',
    colorWarning: '#fb923c',
    accentRgb: '245, 158, 11',
    shadowGlow: '0 0 20px rgba(245, 158, 11, 0.15)',
    ...sharedDarkTokens,
  },
  minimalGray: {
    colorPrimary: '#9ca3af',
    colorBgContainer: '#171717',
    colorBgLayout: '#0a0a0a',
    colorText: '#d4d4d4',
    colorTextSecondary: '#737373',
    colorBorder: '#262626',
    colorSuccess: '#a3a3a3',
    colorWarning: '#d6d3d1',
    accentRgb: '156, 163, 175',
    shadowGlow: '0 0 20px rgba(156, 163, 175, 0.12)',
    ...sharedDarkTokens,
  },
}
```

- [ ] **Step 3: Update all 4 light theme presets with new tokens**

Replace `lightPresets` similarly — share common light tokens, vary accentRgb + shadowGlow:

```typescript
const sharedLightTokens = {
  shadowSm: '0 1px 3px rgba(0,0,0,0.06)',
  shadowMd: '0 4px 12px rgba(0,0,0,0.08)',
  shadowLg: '0 8px 24px rgba(0,0,0,0.12)',
  borderSubtle: 'rgba(0,0,0,0.06)',
  bgDeep: '#f5f5f7',
  bgBase: '#fafafa',
  bgElevated: 'rgba(255,255,255,0.8)',
  bgGlass: 'rgba(255,255,255,0.6)',
  radiusSm: '4px',
  radiusMd: '8px',
  radiusLg: '12px',
  radiusXl: '16px',
  transitionFast: '100ms ease-out',
  transitionNormal: '150ms ease-out',
  transitionSlow: '200ms ease-out',
}

const lightPresets: Record<ThemePreset, ThemeTokens> = {
  nightGreen: {
    colorPrimary: '#16a34a',
    colorBgContainer: '#ffffff',
    colorBgLayout: '#f8fafc',
    colorText: '#1e293b',
    colorTextSecondary: '#64748b',
    colorBorder: 'rgba(0,0,0,0.08)',
    colorSuccess: '#22c55e',
    colorWarning: '#f59e0b',
    accentRgb: '22, 163, 74',
    shadowGlow: '0 0 20px rgba(22, 163, 74, 0.1)',
    ...sharedLightTokens,
  },
  starPurple: {
    colorPrimary: '#7c3aed',
    colorBgContainer: '#ffffff',
    colorBgLayout: '#faf8ff',
    colorText: '#1e1b2e',
    colorTextSecondary: '#6b6480',
    colorBorder: 'rgba(0,0,0,0.08)',
    colorSuccess: '#10b981',
    colorWarning: '#f59e0b',
    accentRgb: '124, 58, 237',
    shadowGlow: '0 0 20px rgba(124, 58, 237, 0.1)',
    ...sharedLightTokens,
  },
  warmAmber: {
    colorPrimary: '#d97706',
    colorBgContainer: '#fffbeb',
    colorBgLayout: '#fffdf5',
    colorText: '#292524',
    colorTextSecondary: '#78716c',
    colorBorder: 'rgba(0,0,0,0.08)',
    colorSuccess: '#65a30d',
    colorWarning: '#ea580c',
    accentRgb: '217, 119, 6',
    shadowGlow: '0 0 20px rgba(217, 119, 6, 0.1)',
    ...sharedLightTokens,
  },
  minimalGray: {
    colorPrimary: '#6b7280',
    colorBgContainer: '#ffffff',
    colorBgLayout: '#f5f5f5',
    colorText: '#262626',
    colorTextSecondary: '#737373',
    colorBorder: 'rgba(0,0,0,0.08)',
    colorSuccess: '#84cc16',
    colorWarning: '#d6d3d1',
    accentRgb: '107, 114, 128',
    shadowGlow: '0 0 20px rgba(107, 114, 128, 0.08)',
    ...sharedLightTokens,
  },
}
```

- [ ] **Step 4: Verify TypeScript compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

Expected: no errors.

---

### Task 2: Sync new CSS variables to :root in App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

**Interfaces:**
- Consumes: Extended `ThemeTokens` from Task 1

- [ ] **Step 1: Add new CSS variable sync in useEffect**

Replace the `useEffect` in `App.tsx`:

```tsx
useEffect(() => {
  const root = document.documentElement
  root.style.setProperty('--color-primary', tokens.colorPrimary)
  root.style.setProperty('--color-bg-container', tokens.colorBgContainer)
  root.style.setProperty('--color-bg-layout', tokens.colorBgLayout)
  root.style.setProperty('--color-text', tokens.colorText)
  root.style.setProperty('--color-text-secondary', tokens.colorTextSecondary)
  root.style.setProperty('--color-border', tokens.colorBorder)
  root.style.setProperty('--color-success', tokens.colorSuccess)
  root.style.setProperty('--color-warning', tokens.colorWarning)
  // 新增 Layered Glass 令牌
  root.style.setProperty('--accent-rgb', tokens.accentRgb)
  root.style.setProperty('--shadow-sm', tokens.shadowSm)
  root.style.setProperty('--shadow-md', tokens.shadowMd)
  root.style.setProperty('--shadow-lg', tokens.shadowLg)
  root.style.setProperty('--shadow-glow', tokens.shadowGlow)
  root.style.setProperty('--border-subtle', tokens.borderSubtle)
  root.style.setProperty('--bg-deep', tokens.bgDeep)
  root.style.setProperty('--bg-base', tokens.bgBase)
  root.style.setProperty('--bg-elevated', tokens.bgElevated)
  root.style.setProperty('--bg-glass', tokens.bgGlass)
  root.style.setProperty('--radius-sm', tokens.radiusSm)
  root.style.setProperty('--radius-md', tokens.radiusMd)
  root.style.setProperty('--radius-lg', tokens.radiusLg)
  root.style.setProperty('--radius-xl', tokens.radiusXl)
  root.style.setProperty('--transition-fast', tokens.transitionFast)
  root.style.setProperty('--transition-normal', tokens.transitionNormal)
  root.style.setProperty('--transition-slow', tokens.transitionSlow)
}, [tokens])
```

- [ ] **Step 2: Update ConfigProvider theme.token with new color values**

Extend the `ConfigProvider` `theme.token` to also pass the light-mode `colorBorder` if needed (already done implicitly via token merging):

```tsx
<ConfigProvider
  theme={{
    algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      colorPrimary: tokens.colorPrimary,
      colorBgContainer: tokens.colorBgContainer,
      colorBgLayout: tokens.colorBgLayout,
      colorText: tokens.colorText,
      colorTextSecondary: tokens.colorTextSecondary,
      colorBorder: tokens.colorBorder,  // 现在可能是 rgba 字符串
    },
  }}
>
```

- [ ] **Step 3: Verify compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 3: Add global utility CSS for glass panels, transitions, and reduced motion

**Files:**
- Modify: `frontend/src/index.css`

- [ ] **Step 1: Append Layered Glass utility styles**

After the existing scrollbar styles, add:

```css
/* ═══ Layered Glass 全局工具样式 ═══ */

/* 玻璃面板基类 */
.glass-panel {
  background: var(--bg-glass);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

/* 玻璃面板悬停 */
.glass-card {
  background: var(--bg-glass);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition: transform var(--transition-slow), box-shadow var(--transition-slow);
}
.glass-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

/* 内凹编辑器区 */
.inset-editor {
  background: rgba(0, 0, 0, 0.15);
  border-radius: var(--radius-md);
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.3);
  padding: 16px;
}

/* 分区标题 */
.section-title {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-subtle);
}

/* 主题按钮（带发光） */
.btn-primary-glow {
  background: var(--color-primary) !important;
  border-color: var(--color-primary) !important;
  box-shadow: var(--shadow-glow);
  transition: box-shadow var(--transition-slow);
}
.btn-primary-glow:hover {
  box-shadow: 0 0 30px rgba(var(--accent-rgb), 0.25);
}

/* 链接/导航高亮条 */
.nav-indicator {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 2px;
  height: 20px;
  background: var(--color-primary);
  border-radius: 1px;
  opacity: 0;
  transition: opacity var(--transition-fast);
}
.nav-active .nav-indicator {
  opacity: 1;
}

/* 页面背景渐变 */
.page-bg {
  background: linear-gradient(180deg, var(--bg-deep) 0%, var(--bg-base) 100%);
}

/* reduced-motion 尊重 */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

- [ ] **Step 2: Verify the CSS file is valid**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

(No CSS syntax checks needed — Vite handles it at build time.)

---

### Task 4: Redesign MainLayout — glass header, navigation, footer, XAI console

**Files:**
- Modify: `frontend/src/layouts/MainLayout.tsx`

- [ ] **Step 1: Apply page background gradient**

Wrap the outermost `<Layout>` with the gradient class. Change line 208 `style`:

```tsx
<Layout style={{ minHeight: '100vh', background: 'linear-gradient(180deg, var(--bg-deep) 0%, var(--bg-base) 100%)' }}>
```

- [ ] **Step 2: Redesign Header as glass bar**

Replace the `<Header>` div (line 210-228) style:

```tsx
<Header style={{
  display: 'flex', alignItems: 'center', height: 48, padding: '0 16px',
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(12px)',
  WebkitBackdropFilter: 'blur(12px)',
  borderBottom: '1px solid var(--border-subtle)',
  lineHeight: '48px',
  position: 'sticky', top: 0, zIndex: 100,
}}>
```

Update logo area — scale up favicon slightly and add subtle text shadow:

```tsx
<div style={{ display: 'flex', alignItems: 'center', gap: 8, marginRight: 24 }}>
  <img src="/favicon.svg" alt="wubigork" style={{ width: 26, height: 26 }} />
  <Typography.Text strong style={{
    color: 'var(--color-primary)', fontSize: 16,
    textShadow: '0 0 12px rgba(var(--accent-rgb), 0.3)',
  }}>
    wubigork
  </Typography.Text>
</div>
```

- [ ] **Step 3: Redesign horizontal Menu — glass style, no underline**

Replace the `<Menu>` component (lines 222-228). Use Ant Design Menu `theme="dark"` is not ideal; instead use inline styling on items:

```tsx
<Menu
  mode="horizontal"
  selectedKeys={[page]}
  items={menuItems.map((m) => ({
    key: m.key,
    icon: m.icon,
    label: m.label,
    style: {
      borderRadius: 'var(--radius-md)',
      margin: '0 2px',
      transition: 'background var(--transition-fast)',
    },
  }))}
  onClick={({ key }) => setPage(key as Page)}
  style={{
    flex: 1, minWidth: 0, background: 'transparent', borderBottom: 'none',
  }}
  // Override Ant Design selected style via theme
  theme="dark"
/>
```

Actually, to better control the selected state without Ant Design's dark theme overriding our tokens, use a custom approach with CSS. Add to the `Menu` style an override:

```tsx
<Menu
  mode="horizontal"
  selectedKeys={[page]}
  items={menuItems.map((m) => ({
    key: m.key,
    icon: m.icon,
    label: m.label,
  }))}
  onClick={({ key }) => setPage(key as Page)}
  style={{
    flex: 1, minWidth: 0, background: 'transparent', borderBottom: 'none',
  }}
/>
```

And add this CSS in `index.css`:

```css
/* 导航菜单玻璃风格 */
.ant-menu-horizontal {
  border-bottom: none !important;
}
.ant-menu-horizontal .ant-menu-item {
  border-radius: var(--radius-md);
  margin: 0 2px !important;
  padding: 0 12px !important;
  transition: background var(--transition-fast);
  color: var(--color-text-secondary);
}
.ant-menu-horizontal .ant-menu-item:hover {
  background: var(--bg-elevated) !important;
  color: var(--color-text) !important;
}
.ant-menu-horizontal .ant-menu-item-selected {
  background: rgba(var(--accent-rgb), 0.12) !important;
  color: var(--color-primary) !important;
}
.ant-menu-horizontal .ant-menu-item-selected::after {
  display: none !important; /* 去掉 Ant Design 默认下划线 */
}
```

Append this to `index.css` and add the class `glass-nav` to the Menu or use the global selectors.

- [ ] **Step 4: Redesign theme dots — enlarge slightly**

Change theme dot size from 16px to 18px (line 239):

```tsx
style={{
  width: 18, height: 18, borderRadius: '50%',
  // ... rest unchanged
}}
```

- [ ] **Step 5: Redesign XAI console panel as glass floating panel**

Replace the console panel div (line 317-385). Key changes: rounded corners, glass background, system font, no Consolas:

```tsx
{consoleOpen && (
  <div style={{
    width: 280,
    margin: '8px 8px 8px 0',
    background: 'var(--bg-glass)',
    backdropFilter: 'blur(16px)',
    WebkitBackdropFilter: 'blur(16px)',
    border: '1px solid var(--border-subtle)',
    borderRadius: 'var(--radius-lg)',
    boxShadow: 'var(--shadow-md)',
    display: 'flex', flexDirection: 'column', fontSize: 11,
    fontFamily: 'system-ui, sans-serif',
    overflow: 'hidden',
    animation: 'slideInRight 0.25s cubic-bezier(0.16, 1, 0.3, 1)',
  }}>
    {/* Header */}
    <div style={{
      padding: '8px 12px',
      borderBottom: '1px solid var(--border-subtle)',
      display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    }}>
      <Space size={6}>
        <span style={{
          width: 6, height: 6, borderRadius: '50%',
          background: logs.length > 0 ? 'var(--color-primary)' : 'var(--color-text-secondary)',
          display: 'inline-block',
          boxShadow: logs.length > 0 ? '0 0 6px var(--color-primary)' : 'none',
        }} />
        <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 10, fontWeight: 500 }}>
          AI 控制台
        </Typography.Text>
      </Space>
      <Space size={4}>
        <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 9, opacity: 0.6 }}>
          {logs.length}
        </Typography.Text>
        <Button type="text" size="small" onClick={() => setConsoleOpen(false)}
          style={{
            color: 'var(--color-text-secondary)', fontSize: 12, padding: 0,
            width: 20, height: 20, borderRadius: '50%',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
          ✕
        </Button>
      </Space>
    </div>
    {/* Log list */}
    <div style={{ flex: 1, overflow: 'auto', padding: '8px 12px' }}>
      {logs.length === 0 ? (
        <div style={{ color: 'var(--color-text-secondary)', textAlign: 'center', marginTop: 40, opacity: 0.5 }}>
          <div style={{ fontSize: 24, marginBottom: 8 }}>◉</div>
          <div>等待 AI 调用...</div>
        </div>
      ) : (
        logs.map((l) => (
          <div key={l.id} style={{
            marginBottom: 6, padding: '4px 8px',
            background: 'var(--bg-elevated)',
            borderRadius: 'var(--radius-sm)',
            lineHeight: 1.5,
            borderLeft: l.type === 'error' ? '2px solid #f87171'
              : l.type === 'chunk' ? '2px solid var(--color-primary)'
              : '2px solid transparent',
          }}>
            <span style={{ color: 'var(--color-text-secondary)', fontSize: 9, opacity: 0.5 }}>{l.time}</span>{' '}
            {l.type === 'request' && (
              <span>
                <span style={{ color: '#60a5fa', fontWeight: 600, fontSize: 9 }}>REQ</span>{' '}
                <span style={{ color: 'var(--color-text)', fontSize: 10 }}>{l.model}</span>
              </span>
            )}
            {l.type === 'response' && (
              <span>
                <span style={{ color: 'var(--color-primary)', fontWeight: 600, fontSize: 9 }}>OK</span>{' '}
                <span style={{ color: 'var(--color-text-secondary)', fontSize: 10 }}>{l.length} 字</span>
              </span>
            )}
            {l.type === 'error' && (
              <span style={{ color: '#f87171', fontSize: 10 }}>{l.error?.slice(0, 60)}</span>
            )}
            {l.type === 'chunk' && (
              <span style={{ color: 'var(--color-primary)', fontSize: 9 }}>▌</span>
            )}
          </div>
        ))
      )}
      <div ref={logEnd} />
    </div>
  </div>
)}
```

- [ ] **Step 6: Update collapsed console toggle button**

Replace the collapsed console button (lines 387-397) with a small glass circle:

```tsx
{!consoleOpen && (
  <Button
    type="text" size="small"
    onClick={() => setConsoleOpen(true)}
    style={{
      position: 'fixed', right: 12, top: 56, zIndex: 1000,
      width: 28, height: 28, borderRadius: '50%',
      background: 'var(--bg-glass)',
      backdropFilter: 'blur(8px)',
      border: '1px solid var(--border-subtle)',
      color: 'var(--color-text-secondary)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      boxShadow: 'var(--shadow-sm)',
    }}
    title="AI 控制台"
  >
    <ConsoleSqlOutlined style={{ fontSize: 12 }} />
  </Button>
)}
```

- [ ] **Step 7: Update footer StatusBar to glass style**

Replace the `statusBarStyle` constants (lines 64-69):

```typescript
const statusBarStyle = {
  display: 'flex' as const, alignItems: 'center' as const, justifyContent: 'space-between' as const,
  padding: '0 16px', height: 32, fontSize: 12,
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(12px)',
  WebkitBackdropFilter: 'blur(12px)',
  borderTop: '1px solid var(--border-subtle)',
  color: 'var(--color-text-secondary)',
}
```

- [ ] **Step 8: Verify compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 5: Redesign HomePage — glass cards, hover effects, empty state

**Files:**
- Modify: `frontend/src/pages/HomePage.tsx`

- [ ] **Step 1: Apply page background**

Wrap the outer container with the gradient class. Find the outermost `<div>` and add the background. (HomePage mostly uses inline styles via Ant Design Card — rely on global CSS.)

- [ ] **Step 2: Redesign project cards as glass-card elements**

Find the project card rendering section. The cards use Ant Design `<Card>`. Wrap them and apply the glass class:

Change the card rendering to use a custom wrapper with the `glass-card` class from `index.css`:

```tsx
<div
  className="glass-card"
  style={{ padding: 0, overflow: 'hidden' }}
  onClick={() => openProject(p.path, p.title)}
>
  <Card
    hoverable={false}
    style={{
      background: 'transparent',
      border: 'none',
      borderRadius: 0,
    }}
    bodyStyle={{ padding: '16px 20px' }}
  >
    {/* existing card content */}
  </Card>
</div>
```

- [ ] **Step 3: Add hover transition to cards**

The `glass-card` CSS class already handles hover. Verify the existing card has `cursor: pointer`.

- [ ] **Step 4: Add empty state branding when no projects**

When `projects.length === 0`, replace the current empty div with:

```tsx
<div style={{
  display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
  height: '60vh', opacity: 0.3, userSelect: 'none', pointerEvents: 'none',
}}>
  <img src="/favicon.svg" alt="" style={{ width: 80, height: 80, marginBottom: 16 }} />
  <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 24, fontWeight: 200, letterSpacing: '0.1em' }}>
    wubigork
  </Typography.Text>
  <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 12, marginTop: 8 }}>
    Ctrl+N 新建项目
  </Typography.Text>
</div>
```

- [ ] **Step 5: Update new project button to primary-glow style**

Find the "新建项目" button and add the `btn-primary-glow` class (or inline the glow shadow):

```tsx
<Button type="primary" icon={<PlusOutlined />} onClick={() => setNewModal(true)}
  style={{
    background: 'var(--color-primary)',
    borderColor: 'var(--color-primary)',
    boxShadow: 'var(--shadow-glow)',
    borderRadius: 'var(--radius-md)',
  }}>
  新建项目
</Button>
```

- [ ] **Step 6: Style brainstorm modal sections with glass**

In the brainstorm section, wrap the ideas list in a glass panel:

```tsx
<div style={{
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  padding: 16,
  marginBottom: 16,
}}>
```

- [ ] **Step 7: Verify compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 6: Redesign ChapterPage — sidebar glass, pill tabs, inset editor, toolbar

**Files:**
- Modify: `frontend/src/pages/ChapterPage.tsx`
- Modify: `frontend/src/index.css` (for Ant Design tab overrides)

- [ ] **Step 1: Redesign left sidebar (outline tree) as glass panel**

Find the outline sidebar container. Replace its style:

```tsx
<div style={{
  width: 220, flexShrink: 0,
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  WebkitBackdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  overflow: 'auto',
  marginRight: 12,
}}>
```

Update the selected node indicator: replace the current selected state (around line 387 in the original file) with:

```tsx
const isSelected = n.id === activeKey
// ... in the node div:
style={{
  ...,
  borderLeft: isSelected ? '3px solid var(--color-primary)' : '3px solid transparent',
  background: isSelected ? 'rgba(var(--accent-rgb), 0.08)' : 'transparent',
  borderRadius: '0 var(--radius-sm) var(--radius-sm) 0',
  transition: 'background var(--transition-fast), border-color var(--transition-fast)',
}}
```

- [ ] **Step 2: Redesign tabs as pill-style (custom Ant Design override)**

Add CSS overrides in `index.css` to transform the editable-card tabs into pill style:

```css
/* Pill 风格标签页 */
.chapter-tabs .ant-tabs-nav {
  margin-bottom: 0 !important;
}
.chapter-tabs .ant-tabs-tab {
  border-radius: 20px !important;
  padding: 4px 14px !important;
  margin: 4px 2px !important;
  border: 1px solid var(--border-subtle) !important;
  background: transparent !important;
  color: var(--color-text-secondary) !important;
  transition: all var(--transition-fast) !important;
  font-size: 12px !important;
}
.chapter-tabs .ant-tabs-tab-active {
  background: var(--color-primary) !important;
  border-color: var(--color-primary) !important;
  color: #fff !important;
}
.chapter-tabs .ant-tabs-tab-active .ant-tabs-tab-btn {
  color: #fff !important;
}
.chapter-tabs .ant-tabs-tab:hover {
  background: var(--bg-elevated) !important;
  border-color: var(--border-subtle) !important;
}
.chapter-tabs .ant-tabs-tab-active:hover {
  background: var(--color-primary) !important;
}
.chapter-tabs .ant-tabs-tab-remove {
  color: var(--color-text-secondary) !important;
  font-size: 10px !important;
}
.chapter-tabs .ant-tabs-tab-active .ant-tabs-tab-remove {
  color: rgba(255,255,255,0.7) !important;
}
```

- [ ] **Step 3: Apply inset editor style to the textarea**

Find the main textarea(s) in the chapter content area. Apply the `inset-editor` class from index.css, or inline equivalent:

```tsx
<Input.TextArea
  value={scene}
  onChange={(e) => ...}
  style={{
    background: 'rgba(0,0,0,0.15)',
    borderRadius: 'var(--radius-md)',
    border: '1px solid var(--border-subtle)',
    color: 'var(--color-text)',
    padding: '16px',
    fontSize: 15,
    lineHeight: 1.8,
    boxShadow: 'inset 0 2px 8px rgba(0,0,0,0.3)',
    resize: 'vertical',
    minHeight: 200,
  }}
/>
```

- [ ] **Step 4: Redesign toolbar buttons**

Apply consistent styling to toolbar buttons. Replace the existing `<Button>` styles in the toolbar row:

Each small button:
```tsx
<Button size="small" icon={<SaveOutlined />} onClick={handleSave} disabled={!content}
  style={{
    background: 'var(--bg-elevated)',
    border: '1px solid var(--border-subtle)',
    color: 'var(--color-text-secondary)',
    borderRadius: 'var(--radius-md)',
    transition: 'all var(--transition-fast)',
  }}>
  保存
</Button>
```

For the primary generate button, use glow style:
```tsx
<Button type="primary" size="small" icon={<PlayCircleOutlined />} onClick={handleGenerate}
  loading={activeTab.generating}
  style={{
    background: 'var(--color-primary)',
    borderColor: 'var(--color-primary)',
    boxShadow: 'var(--shadow-glow)',
    borderRadius: 'var(--radius-md)',
  }}>
  生成
</Button>
```

- [ ] **Step 5: Redesign stats/action buttons row**

Update the full toolbar row (review, stats, book review buttons) with consistent small-button glass style:

```tsx
<Button size="small" icon={<SearchOutlined />} onClick={handleReview}
  style={{
    background: 'var(--bg-elevated)',
    border: '1px solid rgba(96, 165, 250, 0.3)',
    color: '#60a5fa',
    borderRadius: 'var(--radius-md)',
  }}>
  审稿
</Button>
```

- [ ] **Step 6: Verify compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 7: Apply global glass styles to remaining pages

**Files:**
- Modify: `frontend/src/pages/WorldviewPage.tsx`
- Modify: `frontend/src/pages/CharacterPage.tsx`
- Modify: `frontend/src/pages/OutlinePage.tsx`
- Modify: `frontend/src/pages/CanvasPage.tsx`
- Modify: `frontend/src/pages/SettingsPage.tsx`
- Modify: `frontend/src/pages/ExportPage.tsx`

This task applies consistent glass-card, input, and button patterns across all remaining pages.

- [ ] **Step 1: WorldviewPage — glass panels for dimension cards**

Replace `Section` component inline style background with glass:

```tsx
const Section: React.FC<...> = ({ title, color, extra, children }) => (
  <div style={{
    background: 'var(--bg-glass)',
    backdropFilter: 'blur(8px)',
    WebkitBackdropFilter: 'blur(8px)',
    borderRadius: 'var(--radius-lg)',
    border: '1px solid var(--border-subtle)',
    padding: '12px 16px', marginBottom: 8,
    boxShadow: 'var(--shadow-sm)',
  }}>
    {/* existing content */}
  </div>
)
```

Replace `Block` component similarly:

```tsx
const Block: React.FC<{ label: string; children: React.ReactNode }> = ({ label, children }) => (
  <div style={{
    marginBottom: 10, padding: '10px 14px',
    background: 'var(--bg-elevated)',
    borderRadius: 'var(--radius-md)',
    border: '1px solid var(--border-subtle)',
  }}>
    <Typography.Text strong style={{
      color: 'var(--color-text-secondary)', fontSize: 10,
      display: 'block', marginBottom: 6,
      textTransform: 'uppercase', letterSpacing: '0.05em',
    }}>{label}</Typography.Text>
    {children}
  </div>
)
```

- [ ] **Step 2: CharacterPage — glass cards and input fields**

Update character card styles — change Card components to use glass background:

```tsx
<Card
  style={{
    background: 'var(--bg-glass)',
    backdropFilter: 'blur(8px)',
    border: '1px solid var(--border-subtle)',
    borderRadius: 'var(--radius-lg)',
    boxShadow: 'var(--shadow-sm)',
    cursor: 'pointer',
    transition: 'box-shadow var(--transition-slow), transform var(--transition-slow)',
  }}
  hoverable
  // ...
>
```

Update input fields throughout the modal forms:
```tsx
<Input
  style={{
    background: 'rgba(255,255,255,0.05)',
    border: '1px solid var(--border-subtle)',
    borderRadius: 'var(--radius-md)',
    color: 'var(--color-text)',
  }}
/>
```

- [ ] **Step 3: OutlinePage — consistent glass panels**

Update the `Section` and `Block` components in OutlinePage (mirror WorldviewPage changes). Also update the story thread textarea:

```tsx
<Input.TextArea
  style={{
    background: 'rgba(0,0,0,0.15)',
    border: '1px solid var(--border-subtle)',
    borderRadius: 'var(--radius-md)',
    color: 'var(--color-text)',
    boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)',
    minHeight: 120,
  }}
/>
```

- [ ] **Step 4: CanvasPage — glass timeline cards**

Update the chapter card style in the horizontal timeline:

```tsx
<div style={{
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-sm)',
  padding: '12px 16px',
  minWidth: 180,
  transition: 'box-shadow var(--transition-slow)',
  cursor: 'pointer',
}}>
```

- [ ] **Step 5: SettingsPage — glass panels for config sections**

Update the settings sections with glass panel background:

```tsx
<div style={{
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  padding: '16px 20px',
  marginBottom: 12,
}}>
```

Apply glass input styles to all inputs. Apply `btn-primary-glow` to primary action buttons.

- [ ] **Step 6: ExportPage — glass export panel**

Update the export button area:

```tsx
<div style={{
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  padding: 24,
  textAlign: 'center',
}}>
```

- [ ] **Step 7: Verify compilation across all files**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 8: Redesign modals — glass dialogs with spring entry

**Files:**
- Modify: `frontend/src/index.css` (modal animation keyframes)
- Modify: `frontend/src/components/BookReviewModal.tsx`
- Modify: `frontend/src/components/SearchModal.tsx`
- Modify: `frontend/src/components/StoryBibleModal.tsx`
- Modify: `frontend/src/components/PlotBranchModal.tsx`
- Modify: `frontend/src/components/SkillModal.tsx`
- Modify: `frontend/src/components/LorebookModal.tsx`
- Modify: `frontend/src/components/StatsModal.tsx`

- [ ] **Step 1: Add modal glass animation keyframes to index.css**

```css
/* 玻璃模态框弹簧入场 */
@keyframes modalGlassIn {
  from {
    opacity: 0;
    transform: scale(0.92) translateY(12px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

/* 全局 Ant Design Modal 玻璃覆盖 */
.ant-modal-content {
  background: var(--bg-glass) !important;
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle) !important;
  border-radius: var(--radius-xl) !important;
  box-shadow: var(--shadow-lg) !important;
}

.ant-modal-header {
  background: transparent !important;
  border-bottom: 1px solid var(--border-subtle) !important;
}

.ant-modal-footer {
  border-top: 1px solid var(--border-subtle) !important;
}

.ant-modal-close {
  color: var(--color-text-secondary) !important;
}

/* Modal 遮罩 */
.ant-modal-mask {
  backdrop-filter: blur(2px);
  background: rgba(0, 0, 0, 0.6) !important;
}
```

- [ ] **Step 2: Update each modal component for consistency**

For each modal file (BookReviewModal, SearchModal, StoryBibleModal, PlotBranchModal, SkillModal, LorebookModal, StatsModal):

Pattern: The Ant Design `<Modal>` component already gets the glass styling from the global CSS override. But ensure each Modal has no conflicting inline styles. Remove any hardcoded background/border styles from Modal props.

Additionally, update any internal cards/panels in these modals to use `var(--bg-elevated)` instead of solid backgrounds, and inputs to use the glass input style.

Example for BookReviewModal:
- Keep existing Modal props
- Replace internal `<Card>` backgrounds with `background: 'var(--bg-elevated)'`
- Replace internal `<Input>` styles with `background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)'`

- [ ] **Step 3: Verify compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

---

### Task 9: Final verification — full build and visual review

**Files:**
- No new changes. Verification only.

- [ ] **Step 1: Full TypeScript compilation**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 2: Vite production build**

```bash
cd D:/AI/wubigork/frontend && npx vite build
```

Expected: build succeeds, dist files generated.

- [ ] **Step 3: Wails dev build (optional, if Go/Wails toolchain available)**

```bash
cd D:/AI/wubigork && wails build
```

- [ ] **Step 4: Manual visual checklist**

Open the app and verify:
- [ ] 4 theme color dots all work
- [ ] Dark/light toggle works
- [ ] Header is glass with backdrop-blur
- [ ] Navigation menu has no underline, selected item has colored background
- [ ] XAI console has rounded glass panel, system font, no monospace
- [ ] Footer has glass style
- [ ] HomePage cards have glass background + hover lift
- [ ] Empty state shows wubigork watermark
- [ ] ChapterPage sidebar is glass panel
- [ ] ChapterPage tabs are pill-shaped
- [ ] ChapterPage editor has inset shadow
- [ ] All modals have glass panel + blur backdrop
- [ ] All text contrast ≥ 4.5:1
- [ ] No `@ts-ignore` errors from new code
- [ ] Reduced-motion respected (test with OS setting)
