package agent

import "github.com/gaea/gaea/internal/gaea/agent/textutils"

// Re-exported helpers from the textutils sub-package.
var (
	truncateToolOutput     = textutils.TruncateToolOutput
	truncateToolOutputWith = textutils.TruncateToolOutputWith
	firstLine              = textutils.FirstLine
)

func streamedRows(s string, w int) int { return textutils.StreamedRows(s, w) }
