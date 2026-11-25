package memory

// modeView 是按运行模式收窄读路径的 backend 视图（仿 spaceView）：包一层现有
// backend，读路径只放行「通用 + 本模式」的记忆，写路径先按默认规则盖归属标记
// 再透传。详见 mode.go 文件头。
//
// 按名写（Archive/Delete/ChangeType/Pin）先过可见性：本模式看不见的记忆，
// 猜名字也动不到——造价会话删不掉方案档的记忆。

import (
	"fmt"
	"time"
)

type modeView struct {
	inner backend
	mode  string // 会话运行模式；视图存在即启用（非空）
}

var _ interface{ sessionMode() string } = modeView{}

func (v modeView) sessionMode() string { return v.mode }

func (v modeView) visible(m Memory) bool { return modeVisible(v.mode, m) }

func (v modeView) filter(in []Memory) []Memory {
	out := in[:0:0]
	for _, m := range in {
		if v.visible(m) {
			out = append(out, m)
		}
	}
	return out
}

// Index 视图内的索引：带模式标签，让模型看得出哪几条是本模式专属。
func (v modeView) Index() string { return renderIndexTagged(v.filter(v.inner.List()), v.mode) }

func (v modeView) Path(name string) string { return v.inner.Path(name) }

func (v modeView) Save(m Memory) (string, error) {
	normalizeModeForSave(&m, v.mode)
	// 同名条目属于本模式看不见的记忆：拒绝，不覆盖——否则造价会话按名「新建」
	// 一条，会悄悄把方案档那条记忆改归属、换正文。
	if prev, ok := v.inner.Get(slug(m.Name)); ok && !v.visible(prev) {
		return "", fmt.Errorf("memory %q 已被「%s」运行模式的记忆占用：换个名称，或用 scope 把这条也归入该模式", slug(m.Name), ModeLabel(prev.Mode))
	}
	return v.inner.Save(m)
}

func (v modeView) Archive(name string) (string, error) {
	if _, ok := v.Get(name); !ok {
		return "", nil // 不可见=不存在：与「归档缺失条不是错误」同语义
	}
	return v.inner.Archive(name)
}

func (v modeView) Unarchive(name string) error {
	// 归档区不按模式收窄（面板统一管理）；恢复出来的条目若属别的模式，本会话
	// 仍然看不到——如实返回，不替调用方改归属。
	return v.inner.Unarchive(name)
}

func (v modeView) Delete(name string) error {
	if _, ok := v.Get(name); !ok {
		return nil
	}
	return v.inner.Delete(name)
}

func (v modeView) ChangeType(name string, t Type) error {
	if _, ok := v.Get(name); !ok {
		return errModeHidden(name)
	}
	return v.inner.ChangeType(name, t)
}

func (v modeView) ListArchived() []ArchivedMemory { return v.inner.ListArchived() }

func (v modeView) ListArchivedPaged(l, o int) ([]ArchivedMemory, int, error) {
	return v.inner.ListArchivedPaged(l, o)
}

func (v modeView) CleanupArchived(cutoff time.Time) ([]ArchivedMemory, error) {
	return v.inner.CleanupArchived(cutoff)
}

// errModeHidden 是按名写一条本模式看不见的记忆时的报错。
func errModeHidden(name string) error {
	return fmt.Errorf("memory %q 不在当前运行模式的记忆里（它属于别的模式或是通用记忆之外的名字）", name)
}

// List 视图内「全部」= 通用 + 本模式。
func (v modeView) List() []Memory { return v.filter(v.inner.List()) }

func (v modeView) ListInSpace(space string) []Memory { return v.filter(v.inner.ListInSpace(space)) }

func (v modeView) Get(name string) (Memory, bool) {
	m, ok := v.inner.Get(name)
	return m, ok && v.visible(m)
}

func (v modeView) GetInSpace(name, space string) (Memory, bool) {
	m, ok := v.inner.GetInSpace(name, space)
	return m, ok && v.visible(m)
}

func (v modeView) Touch(name string) error {
	if _, ok := v.Get(name); !ok {
		return nil
	}
	return v.inner.Touch(name)
}

func (v modeView) TouchInSpace(name, space string) error {
	if _, ok := v.GetInSpace(name, space); !ok {
		return nil
	}
	return v.inner.TouchInSpace(name, space)
}

// Pin/Unpin 固化态：先过可见性再委托内层（固化是写路径全局属性，不随视图改写）。
func (v modeView) Pin(name string) error {
	if _, ok := v.Get(name); !ok {
		return errModeHidden(name)
	}
	return v.inner.Pin(name)
}

func (v modeView) Unpin(name string) error {
	if _, ok := v.Get(name); !ok {
		return errModeHidden(name)
	}
	return v.inner.Unpin(name)
}

// AppendFeedbackEvent 事件行透传内层后端（反馈事件不带模式归属）。
func (v modeView) AppendFeedbackEvent(e Event) error { return v.inner.AppendFeedbackEvent(e) }
