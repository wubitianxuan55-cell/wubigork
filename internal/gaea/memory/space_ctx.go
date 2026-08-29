package memory

import (
	"context"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// S1.2 记忆空间隔离器 · A 步（写侧空间化）：ctx 空间管线仿 WithQueue /
// WithSessionSaver（queue.go）——agent 运行链路把会话空间盖章到工具调用
// ctx（注入点 agent.executeOne），remember / memory_get 等记忆工具经
// SpaceFromContext 读取，写读都限定本空间。
//
// 设计权威：docs/gaea-memory-isolation-design.md §方案 A + §勘误与关键锚点
// （「ctx 空间管线仿 memory.WithQueue/WithSessionSaver，注入点
// execute_one.go:147-150」）。私有 struct key 防与其他 ctx key 冲突。

type memorySpaceKey struct{}

// WithSpace 把会话空间盖章到 ctx（记忆工具的空间管线）。值经 spaces.Normalize
// 归一：空值/非法值 → work（与 agent.WithSpace 同一读端降级语义）。
func WithSpace(ctx context.Context, space string) context.Context {
	return context.WithValue(ctx, memorySpaceKey{}, spaces.Normalize(space))
}

// SpaceFromContext 返回 ctx 上盖章的会话空间；无标注（headless 直调、后台
// job 重建 ctx、测试直构造）时缺省 work——写侧落库缺省与 S1.1 的
// sqliteBackend.Save 空值缺省一致，既有调用零行为变化。
func SpaceFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(memorySpaceKey{}).(string); ok {
		return spaces.Normalize(s)
	}
	return spaces.SpaceWork
}
