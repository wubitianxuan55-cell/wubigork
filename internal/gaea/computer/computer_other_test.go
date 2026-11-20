//go:build !windows

package computer

// W1 验收 2：非 Windows 平台 Observe 必须回 ErrUnsupported。
import (
	"errors"
	"testing"
)

func TestObserveUnsupportedOnNonWindows(t *testing.T) {
	s := NewSession(Options{})
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("非 Windows 必须 ErrUnsupported: %v", err)
	}
}
