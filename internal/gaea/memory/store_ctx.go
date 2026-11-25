package memory

import (
	"context"
	"fmt"
	"strings"
)

// store_ctx.go — 按空间供给记忆存储的会话钩子（2026-11-25 办公工具梳理批）。
//
// memory_get 退役后并入 memory_search（给 name 即按名读正文）。memory_search 是
// 无状态的 builtin 自注册件，不绑定 Store——与会话控制器的索引钩子同构：会话
// 控制器实现 StoreProvider，经工具 ctx 上已盖章的会话钩子取用（生产链路里
// SessionSaver / SessionFactPromoter 都是同一个 *control.Controller）。

// StoreProvider 是「按空间供给记忆存储」的可选能力接口：返回 false = 本控制器
// 不供（记忆未启用），调用方如实报不可用。
type StoreProvider interface {
	MemoryStoreForSpace(space string) (Store, bool)
}

// StoreFromContext 取本次调用的会话记忆存储（已按空间与运行模式收窄的视图）。
func StoreFromContext(ctx context.Context) (Store, bool) {
	space := SpaceFromContext(ctx)
	if ss, ok := SessionSaverFromContext(ctx); ok {
		if p, ok := ss.(StoreProvider); ok {
			return p.MemoryStoreForSpace(space)
		}
	}
	if pr, ok := PromoterFromContext(ctx); ok {
		if p, ok := pr.(StoreProvider); ok {
			return p.MemoryStoreForSpace(space)
		}
	}
	return Store{}, false
}

// RenderMemory 把一条记忆渲染成模型可读的正文（标题 + 元信息 + 正文）。
func RenderMemory(m Memory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", displayTitle(m.Title, m.Name))
	fmt.Fprintf(&b, "- name: %s\n- type: %s\n- kind: %s\n", m.Name, m.Type, m.Kind)
	if strings.TrimSpace(m.Mode) != "" {
		fmt.Fprintf(&b, "- mode: %s\n", ModeLabel(m.Mode))
	}
	if len(m.Tags) > 0 {
		fmt.Fprintf(&b, "- tags: %s\n", strings.Join(m.Tags, ", "))
	}
	b.WriteString("\n")
	b.WriteString(m.Body)
	return b.String()
}
