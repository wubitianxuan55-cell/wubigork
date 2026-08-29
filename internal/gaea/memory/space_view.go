package memory

import "time"

// spaceView 是按空间收窄**读路径**的 backend 视图（S1.2 B 读端隔离器，
// docs/gaea-memory-isolation-design.md §方案 B「Store.Load 加 Space 选项」）：
// 包一层现有 backend，读路径（List/Index/Get/Touch）经空间谓词收窄，写路径
// （Save/Archive/…）原样透传——落库归属仍以 Memory.Space 为准（缺省 work），
// 视图不强制改写。
//
// space 语义与 listInSpace 一致：空 = 不过滤（旧行为），非空 = 仅该空间。
// InSpace("") 返回原 Store，不包视图，既有调用零行为变化。

type spaceView struct {
	inner backend
	space string
}

func (v spaceView) Index() string {
	return renderIndex(v.inner.ListInSpace(v.space))
}

func (v spaceView) Path(name string) string { return v.inner.Path(name) }

// Save 透传：写入归属由 Memory.Space 决定（sqliteBackend 空值缺省 work），
// 视图只收窄读、不改写写侧语义。
func (v spaceView) Save(m Memory) (string, error) { return v.inner.Save(m) }

func (v spaceView) Archive(name string) (string, error)       { return v.inner.Archive(name) }
func (v spaceView) Unarchive(name string) error               { return v.inner.Unarchive(name) }
func (v spaceView) Delete(name string) error                  { return v.inner.Delete(name) }
func (v spaceView) ChangeType(name string, t Type) error      { return v.inner.ChangeType(name, t) }
func (v spaceView) ListArchived() []ArchivedMemory            { return v.inner.ListArchived() }
func (v spaceView) ListArchivedPaged(l, o int) ([]ArchivedMemory, int, error) {
	return v.inner.ListArchivedPaged(l, o)
}
func (v spaceView) CleanupArchived(cutoff time.Time) ([]ArchivedMemory, error) {
	return v.inner.CleanupArchived(cutoff)
}

// List 视图内「全部」= 视图空间的全部（收窄后的全量）。
func (v spaceView) List() []Memory { return v.inner.ListInSpace(v.space) }

// ListInSpace 在视图空间内再按 space 收窄：空 = 视图空间全量，非空 = 交集
//（即内层谓词；视图空间与请求空间不一致时结果为空集，读端降级语义一致）。
func (v spaceView) ListInSpace(space string) []Memory {
	if space == "" {
		return v.inner.ListInSpace(v.space)
	}
	return v.inner.ListInSpace(space)
}

func (v spaceView) Get(name string) (Memory, bool) { return v.inner.GetInSpace(name, v.space) }

func (v spaceView) GetInSpace(name, space string) (Memory, bool) {
	if space == "" {
		return v.inner.GetInSpace(name, v.space)
	}
	return v.inner.GetInSpace(name, space)
}

func (v spaceView) Touch(name string) error { return v.inner.TouchInSpace(name, v.space) }

func (v spaceView) TouchInSpace(name, space string) error {
	if space == "" {
		return v.inner.TouchInSpace(name, v.space)
	}
	return v.inner.TouchInSpace(name, space)
}
