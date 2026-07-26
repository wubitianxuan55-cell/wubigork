# 书架功能 — 项目浏览与管理

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 HomePage 升级为书架视图 — 扫描 `novels/` 目录展示所有项目卡片，支持打开、新建、删除项目。

**Architecture:** Go 后端新增 `ListProjects`（扫描目录 + 计算字数/章节数）和 `DeleteProject`（安全删除）两个 Wails 绑定方法。前端 HomePage 从双按钮升级为响应式卡片网格，每张卡片显示标题/题材/字数/章节数/最后打开时间，并内联新建项目表单。

**Tech Stack:** Go 1.26 + Wails v2.12 + React 18 + Ant Design 5 (Card, Modal, Popconfirm, Row/Col, Empty, Tag)

## Global Constraints

- 文件即真相：不引入数据库，直接扫描文件系统
- 项目目录统一为 `./novels/<项目名>/`
- 不引入新依赖
- 保持现有暗色主题风格
- 遵循架构原则：不引入 SQLite / EventBus / 全局索引

---

### Task 1: Go 后端 — `ListProjects` 方法

**Files:**
- Create: `internal/app/shelf.go` — 书架相关方法（扫描 + 删除）
- Modify: `internal/app/app.go:192-199` — 在 `initAgents` 附近注册

**Interfaces:**
- Produces: `func (a *App) ListProjects() ([]ProjectCard, error)` — Wails 绑定，前端通过 `window.go.app.App.ListProjects()` 调用
- Produces: `type ProjectCard` struct — 返回给前端的项目摘要
- Produces: `func (a *App) DeleteProject(dir string) error` — Wails 绑定

- [ ] **Step 1: 创建 `internal/app/shelf.go`**

```go
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/wubigork/wubigork/internal/types"
)

// ProjectCard 书架上的项目卡片数据
type ProjectCard struct {
	Title        string `json:"title"`
	Genre        string `json:"genre"`
	Style        string `json:"style"`
	Path         string `json:"path"`          // 相对于 novels/ 的目录名
	WordCount    int    `json:"word_count"`
	ChapterCount int    `json:"chapter_count"`
	CreatedAt    string `json:"created_at"`     // ISO8601 格式化
	LastOpenedAt string `json:"last_opened_at"` // ISO8601 格式化
}

// ListProjects 扫描 novels/ 目录，返回所有小说项目摘要
func (a *App) ListProjects() ([]ProjectCard, error) {
	novelsDir := "./novels"
	entries, err := os.ReadDir(novelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProjectCard{}, nil
		}
		return nil, fmt.Errorf("读取 novels 目录失败: %w", err)
	}

	var cards []ProjectCard
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(novelsDir, entry.Name())
		metaPath := filepath.Join(dirPath, "project.json")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue
		}

		meta, err := loadProjectMeta(metaPath)
		if err != nil {
			continue // 跳过损坏的项目
		}

		// 快速扫描章节数和字数
		chapterCount, wordCount := scanChapterStats(dirPath)

		cards = append(cards, ProjectCard{
			Title:        meta.Title,
			Genre:        meta.Genre,
			Style:        meta.Style,
			Path:         dirPath,
			WordCount:    wordCount,
			ChapterCount: chapterCount,
			CreatedAt:    meta.CreatedAt.Format(time.RFC3339),
			LastOpenedAt: meta.LastOpenedAt.Format(time.RFC3339),
		})
	}

	return cards, nil
}

// loadProjectMeta 读取 project.json 的元信息（轻量，只取需要的字段）
func loadProjectMeta(path string) (*types.ProjectMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// 只解析需要的字段，忽略不认识的字段
	var meta struct {
		Title        string    `json:"title"`
		Genre        string    `json:"genre"`
		Style        string    `json:"style"`
		CreatedAt    time.Time `json:"created_at"`
		LastOpenedAt time.Time `json:"last_opened_at"`
	}
	if err := jsonUnmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &types.ProjectMeta{
		Title:        meta.Title,
		Genre:        meta.Genre,
		Style:        meta.Style,
		CreatedAt:    meta.CreatedAt,
		LastOpenedAt: meta.LastOpenedAt,
	}, nil
}

// scanChapterStats 快速扫描章节目录的字数和章节数
func scanChapterStats(projectDir string) (chapterCount int, totalWords int) {
	chaptersDir := filepath.Join(projectDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return 0, 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 只统计 NNN.md 文件，跳过 NNN-summary.json
		if filepath.Ext(name) != ".md" {
			continue
		}
		// 确保是纯数字前缀（如 001.md）
		if len(name) < 7 || name[3] != '.' {
			continue
		}
		content, err := os.ReadFile(filepath.Join(chaptersDir, name))
		if err != nil {
			continue
		}
		chapterCount++
		totalWords += utf8.RuneCountInString(string(content))
	}
	return
}
```

- [ ] **Step 2: 在 `shelf.go` 中添加 `jsonUnmarshal` 辅助和 `DeleteProject`**

在 `shelf.go` 末尾追加：

```go
import "encoding/json"

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// DeleteProject 删除整个项目目录
func (a *App) DeleteProject(dir string) error {
	// 安全检查：必须在 novels/ 目录下
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("路径解析失败: %w", err)
	}
	absNovels, _ := filepath.Abs("./novels")
	if !strings.HasPrefix(absDir, absNovels+string(filepath.Separator)) {
		return fmt.Errorf("出于安全考虑，只能删除 novels/ 下的项目")
	}

	// 如果该项目当前已打开，先关闭
	if a.pm != nil && a.pm.Dir == dir {
		a.pm.Close()
		a.pm = nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("删除项目失败: %w", err)
	}
	return nil
}
```

并更新 import，添加 `"encoding/json"` 和 `"strings"`。

- [ ] **Step 3: 验证编译**

```bash
cd D:/AI/wubigork && go build ./internal/app/
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/shelf.go
git commit -m "feat: add ListProjects and DeleteProject shelf methods"
```

---

### Task 2: 前端 — 更新 appStore

**Files:**
- Modify: `frontend/src/stores/appStore.ts`

**Interfaces:**
- Produces: `projects: ProjectCard[]`, `loadProjects()`, `deleteProject(path: string)`
- Consumes: `window.go.app.App.ListProjects()`, `window.go.app.App.DeleteProject(dir)`

- [ ] **Step 1: 在 `appStore.ts` 中添加书架状态**

在 `interface AppState` 中新增字段和方法，完整替换文件：

```typescript
import { create } from 'zustand'

export interface ProjectCard {
  title: string
  genre: string
  style: string
  path: string
  word_count: number
  chapter_count: number
  created_at: string
  last_opened_at: string
}

interface AppState {
  loggedIn: boolean
  projectOpen: boolean
  projectPath: string
  projectTitle: string
  projects: ProjectCard[]
  login: () => Promise<void>
  checkLogin: () => Promise<void>
  setLoggedIn: (v: boolean) => void
  openProject: (path: string, title: string) => void
  closeProject: () => void
  loadProjects: () => Promise<void>
  deleteProject: (path: string) => Promise<void>
}

export const useAppStore = create<AppState>((set, get) => ({
  loggedIn: false,
  projectOpen: false,
  projectPath: '',
  projectTitle: '',
  projects: [],

  login: async () => {
    try {
      // @ts-ignore — Wails Go binding
      const result = await window.go.app.App.Login()
      set({ loggedIn: true })
    } catch (err: any) {
      console.error('login failed:', err)
      throw err
    }
  },

  checkLogin: async () => {
    try {
      // @ts-ignore
      const status = await window.go.app.App.GetLoginStatus()
      set({ loggedIn: status })
    } catch (_) {
      // Go 绑定未就绪时静默忽略
    }
  },

  setLoggedIn: (v: boolean) => set({ loggedIn: v }),

  openProject: (path: string, title: string) =>
    set({ projectOpen: true, projectPath: path, projectTitle: title }),

  closeProject: () =>
    set({ projectOpen: false, projectPath: '', projectTitle: '' }),

  loadProjects: async () => {
    try {
      // @ts-ignore
      const cards: ProjectCard[] = await window.go.app.App.ListProjects()
      set({ projects: cards || [] })
    } catch (err) {
      console.error('loadProjects failed:', err)
    }
  },

  deleteProject: async (path: string) => {
    try {
      // @ts-ignore
      await window.go.app.App.DeleteProject(path)
      // 从本地列表移除
      const projects = get().projects.filter((p) => p.path !== path)
      set({ projects })
    } catch (err: any) {
      console.error('deleteProject failed:', err)
      throw err
    }
  },
}))
```

- [ ] **Step 2: 验证 TypeScript 编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit src/stores/appStore.ts
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/appStore.ts
git commit -m "feat: add bookshelf state to appStore"
```

---

### Task 3: 前端 — 重写 HomePage 为书架视图

**Files:**
- Modify: `frontend/src/pages/HomePage.tsx` — 完整重写

**Interfaces:**
- Consumes: `useAppStore` — projects, loadProjects, openProject, deleteProject, projectOpen
- Consumes: `window.go.app.App.OpenProject(dir)`, `window.go.app.App.CreateProject(...)`, `window.go.app.App.BootstrapProject(...)`

- [ ] **Step 1: 重写 `HomePage.tsx`**

完整替换文件内容：

```tsx
import React, { useState, useEffect } from 'react'
import {
  Typography, Button, Space, Card, Input, Modal, Switch, Progress,
  Row, Col, Tag, Popconfirm, Empty, message,
} from 'antd'
import {
  PlusOutlined, ThunderboltOutlined, BookOutlined,
  DeleteOutlined, EditOutlined, FileTextOutlined, ClockCircleOutlined,
} from '@ant-design/icons'
import { useAppStore, type ProjectCard } from '../stores/appStore'

const HomePage: React.FC = () => {
  const { loggedIn, login, projectOpen, projectTitle, projects, loadProjects, openProject, deleteProject } = useAppStore()
  const [newModal, setNewModal] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newGenre, setNewGenre] = useState('')
  const [newStyle, setNewStyle] = useState('')
  const [reference, setReference] = useState('')
  const [importName, setImportName] = useState('')
  const [autoBootstrap, setAutoBootstrap] = useState(true)
  const [creating, setCreating] = useState(false)
  const [progress, setProgress] = useState(0)
  const [progressStep, setProgressStep] = useState('')

  useEffect(() => {
    if (loggedIn && !projectOpen) {
      loadProjects()
    }
  }, [loggedIn, projectOpen])

  const handleCreate = async () => {
    if (!newTitle.trim()) return
    setCreating(true)
    setProgress(0)
    try {
      const dir = `./novels/${newTitle.replace(/[/\\:*?"<>|]/g, '_')}`

      if (autoBootstrap) {
        setProgressStep('正在创建项目...')
        setProgress(10)
        // @ts-ignore
        await window.go.app.App.BootstrapProject(
          dir, newTitle, newGenre || '未分类', newStyle || '默认', reference
        )
        setProgressStep('✅ 世界观已生成')
        setProgress(40)
        await delay(300)
        setProgressStep('✅ 角色已生成')
        setProgress(70)
        await delay(300)
        setProgressStep('✅ 大纲已生成')
        setProgress(90)
        await delay(300)
        setProgressStep('全部完成！')
        setProgress(100)
        await delay(500)
      } else {
        // @ts-ignore
        await window.go.app.App.CreateProject(dir, newTitle, newGenre || '未分类', newStyle || '默认')
      }

      openProject(dir, newTitle)
      setNewModal(false)
      resetForm()
    } catch (err: any) {
      message.error(err.message || '创建失败')
    } finally {
      setCreating(false)
      setProgress(0)
    }
  }

  const handleOpen = async (card: ProjectCard) => {
    try {
      // @ts-ignore
      await window.go.app.App.OpenProject(card.path)
      openProject(card.path, card.title)
    } catch (err: any) {
      message.error(err.message || '打开失败')
    }
  }

  const handleDelete = async (card: ProjectCard) => {
    try {
      await deleteProject(card.path)
      message.success(`已删除「${card.title}」`)
    } catch (err: any) {
      message.error(err.message || '删除失败')
    }
  }

  const resetForm = () => {
    setNewTitle('')
    setNewGenre('')
    setNewStyle('')
    setReference('')
    setImportName('')
    setAutoBootstrap(true)
  }

  // --- 未登录 ---
  if (!loggedIn) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '70vh' }}>
        <Card style={{ maxWidth: 480, textAlign: 'center', background: '#1a1a1a', borderColor: '#333' }}>
          <Typography.Title level={2} style={{ color: '#e0e0e0' }}>🚀 欢迎使用 wubigork</Typography.Title>
          <Typography.Paragraph style={{ color: '#9ca3af', fontSize: 16 }}>
            基于 xAI Grok 的桌面端小说创作 Agent。<br />一键导入参考，AI 自动生成世界观、角色和大纲。
          </Typography.Paragraph>
          <Button type="primary" size="large" onClick={login}
            style={{ background: '#4ade80', borderColor: '#4ade80', marginTop: 16 }}>
            登录 xAI 开始创作
          </Button>
        </Card>
      </div>
    )
  }

  // --- 已打开项目 ---
  if (projectOpen) {
    return (
      <div>
        <Typography.Title level={3} style={{ color: '#e0e0e0' }}>
          📖 {projectTitle}
        </Typography.Title>
        <Typography.Paragraph style={{ color: '#9ca3af' }}>
          左侧菜单：世界观 → 角色 → 大纲 → 写作
        </Typography.Paragraph>
      </div>
    )
  }

  // --- 书架视图 ---
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Typography.Title level={3} style={{ color: '#e0e0e0', margin: 0 }}>
          📚 我的书架
        </Typography.Title>
        <Button type="primary" size="large" icon={<PlusOutlined />} onClick={() => setNewModal(true)}
          style={{ background: '#4ade80', borderColor: '#4ade80' }}>
          新建小说
        </Button>
      </div>

      {projects.length === 0 ? (
        <Empty
          description={<span style={{ color: '#555' }}>书架空空，创建你的第一本小说吧！</span>}
          style={{ marginTop: 80 }}
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setNewModal(true)}
            style={{ background: '#4ade80', borderColor: '#4ade80' }}>
            开始创作
          </Button>
        </Empty>
      ) : (
        <Row gutter={[16, 16]}>
          {projects.map((card) => (
            <Col key={card.path} xs={24} sm={12} lg={8} xl={6}>
              <Card
                hoverable
                onClick={() => handleOpen(card)}
                style={{
                  background: '#1a1a1a',
                  borderColor: '#333',
                  borderRadius: 8,
                  height: '100%',
                  cursor: 'pointer',
                }}
                styles={{ body: { padding: 16 } }}
                actions={[
                  <Popconfirm
                    key="delete"
                    title="确定删除？"
                    description={`「${card.title}」的所有数据将被永久删除`}
                    onConfirm={(e) => { e?.stopPropagation(); handleDelete(card) }}
                    onCancel={(e) => e?.stopPropagation()}
                    okText="删除"
                    cancelText="取消"
                    okButtonProps={{ danger: true }}
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(e) => e.stopPropagation()}
                    >
                      删除
                    </Button>
                  </Popconfirm>,
                ]}
              >
                <div onClick={(e) => e.stopPropagation()}>
                  {/* 标题行 */}
                  <Typography.Title level={5} style={{ color: '#e0e0e0', marginBottom: 8, marginTop: 0 }}>
                    <BookOutlined style={{ marginRight: 6, color: '#4ade80' }} />
                    {card.title}
                  </Typography.Title>

                  {/* 标签 */}
                  <Space size={4} wrap style={{ marginBottom: 12 }}>
                    {card.genre && card.genre !== '未分类' && (
                      <Tag color="#60a5fa" style={{ fontSize: 11 }}>{card.genre}</Tag>
                    )}
                    {card.style && card.style !== '默认' && (
                      <Tag color="#c084fc" style={{ fontSize: 11 }}>{card.style}</Tag>
                    )}
                  </Space>

                  {/* 统计信息 */}
                  <Space direction="vertical" size={2} style={{ width: '100%' }}>
                    <Typography.Text style={{ color: '#9ca3af', fontSize: 12 }}>
                      <FileTextOutlined style={{ marginRight: 4 }} />
                      {card.chapter_count > 0
                        ? `${card.word_count.toLocaleString()} 字 · ${card.chapter_count} 章`
                        : '尚未开始写作'}
                    </Typography.Text>
                    <Typography.Text style={{ color: '#555', fontSize: 11 }}>
                      <ClockCircleOutlined style={{ marginRight: 4 }} />
                      {formatRelativeTime(card.last_opened_at)}
                    </Typography.Text>
                  </Space>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {/* 新建项目 Modal — 保持不变 */}
      <Modal
        title="📝 新建小说项目"
        open={newModal}
        onOk={handleCreate}
        onCancel={() => { setNewModal(false); resetForm() }}
        confirmLoading={creating}
        okText={autoBootstrap ? '一键生成' : '创建'}
        cancelText="取消"
        width={520}
        styles={{ body: { background: '#1a1a1a' }, header: { background: '#1a1a1a' } }}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Input placeholder="小说标题（必填）" value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            style={{ background: '#0d0d0d', borderColor: '#333', color: '#e0e0e0' }} />
          <Input placeholder="题材（玄幻、科幻、言情...）" value={newGenre}
            onChange={(e) => setNewGenre(e.target.value)}
            style={{ background: '#0d0d0d', borderColor: '#333', color: '#e0e0e0' }} />
          <Input placeholder="文风（细腻温情、热血战斗...）" value={newStyle}
            onChange={(e) => setNewStyle(e.target.value)}
            style={{ background: '#0d0d0d', borderColor: '#333', color: '#e0e0e0' }} />

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ color: '#9ca3af', fontSize: 13 }}>
              <ThunderboltOutlined style={{ color: '#f59e0b' }} /> AI 引导生成世界观+角色+大纲
            </span>
            <Switch checked={autoBootstrap} onChange={setAutoBootstrap} />
          </div>

          {autoBootstrap && (
            <div>
              <Space style={{ marginBottom: 8 }}>
                <input
                  type="file"
                  accept=".txt,.md,.json,.csv"
                  style={{ display: 'none' }}
                  id="ref-file-input"
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (!file) return
                    const reader = new FileReader()
                    reader.onload = () => {
                      setReference(reader.result as string)
                      setImportName(file.name)
                    }
                    reader.readAsText(file)
                  }}
                />
                <Button size="small" onClick={() => document.getElementById('ref-file-input')!.click()}>
                  📁 导入文件
                </Button>
                {importName && (
                  <span style={{ color: '#4ade80', fontSize: 12 }}>已导入: {importName}</span>
                )}
              </Space>
              <Input.TextArea
                placeholder="📎 参考素材（可选）&#10;粘贴已有设定、灵感片段，或点击上方按钮导入 .txt/.md 文件..."
                value={reference}
                onChange={(e) => setReference(e.target.value)}
                rows={5}
                style={{ background: '#0d0d0d', borderColor: '#333', color: '#e0e0e0' }}
              />
            </div>
          )}

          {creating && (
            <div style={{ textAlign: 'center', padding: 8 }}>
              <Progress percent={progress} size="small" strokeColor="#4ade80" />
              <Typography.Text style={{ color: '#9ca3af', fontSize: 12 }}>{progressStep}</Typography.Text>
            </div>
          )}
        </Space>
      </Modal>
    </div>
  )
}

/** 格式化为相对时间描述 */
function formatRelativeTime(iso: string): string {
  if (!iso) return '未知'
  const date = new Date(iso)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  if (days < 7) return `${days} 天前`
  if (days < 30) return `${Math.floor(days / 7)} 周前`
  return date.toLocaleDateString('zh-CN')
}

function delay(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}

export default HomePage
```

- [ ] **Step 2: 验证前端编译**

```bash
cd D:/AI/wubigork/frontend && npx tsc --noEmit
```

- [ ] **Step 3: 完整构建验证**

```bash
cd D:/AI/wubigork && wails build
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/HomePage.tsx
git commit -m "feat: upgrade HomePage to bookshelf with project cards and delete"
```

---

### Task 4: 边界处理 — Go 后端修复 import 和空书架初始化

**Files:**
- Modify: `internal/app/shelf.go`

- [ ] **Step 1: 确保 novels 目录不存在时自动创建**

在 `ListProjects` 开头，`os.ReadDir` 之前插入目录创建逻辑：

```go
// 确保 novels 目录存在
if err := os.MkdirAll(novelsDir, 0755); err != nil {
	return nil, fmt.Errorf("创建 novels 目录失败: %w", err)
}
```

- [ ] **Step 2: 完整编译验证**

```bash
cd D:/AI/wubigork && go build ./internal/app/ && go vet ./internal/app/
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/shelf.go
git commit -m "fix: auto-create novels dir in ListProjects"
```

---

## 验证清单

- [ ] `ListProjects` 返回空数组时前端显示 Empty 占位
- [ ] 有项目时卡片网格正确渲染
- [ ] 点击卡片 → `OpenProject` → 跳转到已打开项目视图
- [ ] 删除项目 → Popconfirm → 卡片从列表移除
- [ ] 新建项目 → 创建后自动打开并跳转
- [ ] 当前打开的项目在删除列表中时自动关闭
- [ ] `wails build` 编译成功
