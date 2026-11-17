package tool

// P8 能力声明（第四批）：网络能力名单与前缀规则的防漂移钉。

import "testing"

func TestNetworkCapableList(t *testing.T) {
	for _, name := range []string{
		"web_search", "web_fetch",
		"browser_navigate", "browser_read", "browser_snapshot", "browser_click",
		"browser_type", "browser_press", "browser_scroll", "browser_close",
		"browser_tabs", "browser_new_tab", "browser_switch_tab", "browser_run_js",
		"mcp__github__create_issue",
	} {
		if !NetworkCapable(name) {
			t.Errorf("NetworkCapable(%q) = false, want true", name)
		}
	}
	// 本地工具（含 bash：有自己的子进程网络开关，不整体拒绝）。
	for _, name := range []string{"bash", "read_file", "edit_file", "write_file", "todo_write", "task", "grep", "websearchx"} {
		if NetworkCapable(name) {
			t.Errorf("NetworkCapable(%q) = true, want false", name)
		}
	}
}
