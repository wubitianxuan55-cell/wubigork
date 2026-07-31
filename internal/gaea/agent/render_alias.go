package agent

import (
	"io"

	"github.com/gaea/gaea/internal/gaea/agent/render"
	"github.com/gaea/gaea/internal/gaea/event"
)

// Renderer is an alias for render.Renderer — kept for external consumers (cli).
type Renderer = render.Renderer

// TextSink is the legacy name for render.Sink.
type TextSink = render.Sink

// Internal aliases for stream batcher.
var newStreamBatcher = render.NewBatcher

// Exported functions.
var (
	NewTextSink     = func(out io.Writer, r Renderer, w int) *TextSink { return render.NewSink(out, r, w) }
	FormatUsageLine = render.FormatUsageLine
	CompactArgs     = render.CompactArgs
)

// Verify at compile time that event.Sink is compatible with render.Batcher's sink.
var _ = render.NewBatcher((event.Sink)(nil))
