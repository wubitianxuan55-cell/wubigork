package app

import "testing"

// pickFileFilter 契约锁（审计刀D：GaeaPickFiles 可选扩展名过滤）：
// 逗号分隔扩展名 → Wails FileFilter（DisplayName 人类可读、Pattern 分号
// 拼接通配符）；空/空白/带点/大小写混入归一；空输入 → nil（不过滤）。
func TestPickFileFilter(t *testing.T) {
	tests := []struct {
		name    string
		filters string
		want    *struct{ display, pattern string }
	}{
		{"empty", "", nil},
		{"whitespace only", "   , ,", nil},
		{"single ext", "png", &struct{ display, pattern string }{"*.png", "*.png"}},
		{"multi ext", "png,jpg", &struct{ display, pattern string }{"*.png; *.jpg", "*.png;*.jpg"}},
		{"leading dot stripped", ".MD,.txt", &struct{ display, pattern string }{"*.md; *.txt", "*.md;*.txt"}},
		{"case folded", "PNG,WebP", &struct{ display, pattern string }{"*.png; *.webp", "*.png;*.webp"}},
		{"mixed empty segments", "png,,jpg, ", &struct{ display, pattern string }{"*.png; *.jpg", "*.png;*.jpg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickFileFilter(tt.filters)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("pickFileFilter(%q) = %+v, want nil", tt.filters, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("pickFileFilter(%q) = nil, want %+v", tt.filters, tt.want)
			}
			if got.DisplayName != tt.want.display || got.Pattern != tt.want.pattern {
				t.Fatalf("pickFileFilter(%q) = {DisplayName:%q Pattern:%q}, want {%q %q}",
					tt.filters, got.DisplayName, got.Pattern, tt.want.display, tt.want.pattern)
			}
		})
	}
}