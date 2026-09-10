package docmd

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// markItDownTimeout caps a single markitdown conversion. Office files are
// small enough that a slow conversion means the toolchain is wedged; fail
// fast and let callers fall back to the built-in parser.
const markItDownTimeout = 60 * time.Second

// markitdownAvailable lazily probes `python -m markitdown` once per process and
// caches the result, so conversions don't pay exec overhead every call.
var (
	markitdownOnce     sync.Once
	markitdownReady    bool
)

func markitdownAvailable() bool {
	markitdownOnce.Do(func() {
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("python"); err != nil {
				return
			}
		} else {
			if _, err := exec.LookPath("python3"); err != nil {
				return
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := markItDownCmd(ctx, "--help").Run(); err == nil {
			markitdownReady = true
		}
	})
	return markitdownReady
}

// convertViaMarkItDown converts docx/xlsx/pptx to Markdown with Microsoft's
// MarkItDown library. Returns an error when the toolchain is missing or the
// conversion fails; callers fall back to the built-in parser.
func convertViaMarkItDown(path string) (string, error) {
	if !markitdownAvailable() {
		return "", errMarkItDownUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), markItDownTimeout)
	defer cancel()
	var out, errBuf bytes.Buffer
	cmd := markItDownCmd(ctx, path)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	md := strings.TrimSpace(out.String())
	if md == "" {
		return "", errMarkItDownEmpty
	}
	return md, nil
}

func markItDownCmd(ctx context.Context, arg string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "python", "-m", "markitdown", arg)
	} else {
		cmd = exec.CommandContext(ctx, "python3", "-m", "markitdown", arg)
	}
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	return cmd
}

type markItDownError string

func (e markItDownError) Error() string { return string(e) }

const (
	errMarkItDownUnavailable = markItDownError("markitdown 不可用（python -m markitdown 未安装或不可用）")
	errMarkItDownEmpty       = markItDownError("markitdown 转换结果为空")
)
