package tool

import "encoding/json"

// CachePathResolver exposes paths whose cached reads may become stale after a
// successful tool call. It is deliberately separate from WritePathResolver:
// todo_write persists host state while remaining read-only from an approval
// standpoint, and domain writers may conservatively invalidate a workspace.
type CachePathResolver interface {
	CachePaths(args json.RawMessage) []string
}
