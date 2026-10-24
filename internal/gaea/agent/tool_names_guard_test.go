package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestToolNameMentionsSubsetOfRegistry 注册面守卫（v4.520 幽灵工具名第二刀防复发）：
// agent/evidence 源码里以 `case "name"` 与预算表键形态硬编码的工具名，必须落在
// 真实注册面内——builtin（tool.Builtins）∪ 子代理元工具（subagentMetaTools）∪
// ModelBacked 动态面 ∪ 非工具名白名单。历史病灶：delete_range/notebook_edit/glob/
// run_command 等从未注册的名字散布 8+ 处 switch（v4.519/v4.520 两刀清零），lint
// 对「字符串名单」零感知，只有源级断言能锁。
func TestToolNameMentionsSubsetOfRegistry(t *testing.T) {
	known := map[string]bool{}
	for _, bt := range tool.Builtins() {
		known[bt.Name()] = true
	}
	for _, name := range subagentMetaTools {
		known[name] = true
	}
	// ModelBacked 工具按需动态注册（model_tool_recorder/agent_stream 消费面），
	// 无静态清单可引用，在此显式登记。
	known["summarize_file"] = true
	// 非工具名的合法 case（压缩原因/语言标签等枚举），逐个注明归属。
	for _, n := range []string{
		"zh", "cn", "en", "chinese", "english", // 语言标签（compact/翻译域）
		"length", "content_filter", "repetition_truncation", // 压缩原因枚举
		// A11 完成裁决枚举（complete_subtask.go：报告状态/判据状态/证据种类）
		"complete", "partial", "blocked", "failed",
		"satisfied", "unsatisfied",
		"verification", "diff", "files", "review", "manual",
	} {
		known[n] = true
	}

	files := []string{}
	matches, _ := filepath.Glob("*.go")
	files = append(files, matches...)
	files = append(files, "../evidence/evidence.go")

	caseLine := regexp.MustCompile(`^\s*case\s`)
	strLit := regexp.MustCompile(`"([a-z_][a-z0-9_]*)"`)
	budgetKey := regexp.MustCompile(`^\t"([a-z_][a-z0-9_]*)":\s*\{`)

	var bad []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		isBudgetMap := strings.HasSuffix(path, "compress.go")
		for i, line := range strings.Split(string(data), "\n") {
			var names []string
			if caseLine.MatchString(line) {
				for _, m := range strLit.FindAllStringSubmatch(line, -1) {
					names = append(names, m[1])
				}
			} else if isBudgetMap {
				if m := budgetKey.FindStringSubmatch(line); m != nil {
					names = append(names, m[1])
				}
			}
			for _, n := range names {
				if !known[n] {
					bad = append(bad, filepath.Base(path)+":"+itoa(i+1)+" "+n)
				}
			}
		}
	}
	if len(bad) > 0 {
		t.Fatalf("源码提及未注册的工具名（幽灵名回潮或注册面清单漏登）：\n%s",
			strings.Join(bad, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
