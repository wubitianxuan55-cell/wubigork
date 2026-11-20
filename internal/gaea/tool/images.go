package tool

// W0 多模态旁路的工具侧出口（电脑操控施工单 2026-11-19 W0-4）。
// 工具 Execute 期间把产出的本地图片路径塞进本次调用的 ImageSink；
// 执行器（agent.executeOneTimed）调用前注入、调用后收集进 toolOutcome，
// 最终随 tool 结果消息进入主对话请求（bridge 层统一编码+限量）。

import (
	"context"
	"sync"
)

// ImageSink 收集单次工具调用产出的本地图片路径。零值可用；所有方法
// nil 安全（未注入 ctx 的工具无需判空即可 AddImage）。
type ImageSink struct {
	mu    sync.Mutex
	paths []string
}

// AddImage 登记一张本调用产出的图片（本地绝对路径）。空串忽略。
func (s *ImageSink) AddImage(path string) {
	if s == nil || path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paths = append(s.paths, path)
}

// Paths 返回已收集路径的快照（调用方持有副本，工具可继续追加）。
func (s *ImageSink) Paths() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.paths))
	copy(out, s.paths)
	return out
}

type imageSinkKey struct{}

// WithImageSink 把本次调用的图片收集器盖进 ctx（执行器专用注入点）。
func WithImageSink(ctx context.Context, sink *ImageSink) context.Context {
	return context.WithValue(ctx, imageSinkKey{}, sink)
}

// ImageSinkFromContext 取当前调用的图片收集器；未注入时返回 nil。
func ImageSinkFromContext(ctx context.Context) *ImageSink {
	sink, _ := ctx.Value(imageSinkKey{}).(*ImageSink)
	return sink
}
