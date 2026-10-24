package authorprefs

import "strings"

func RenderQuery(q QueryResult) string {
	if strings.TrimSpace(q.Text) == "" {
		return ""
	}
	return "## 作者偏好（仅显式确认项）\n" +
		"以下偏好只约束对应创作维度；未确认的冲突项不得生效：\n" +
		q.Text
}

func RenderCompact(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return "## 作者偏好（仅显式确认项）\n" + strings.Join(lines, "\n")
}
