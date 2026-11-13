# <ask_question_result> · 1 个版本（按长度降序）


## 版本 1（211 字符）

```text
{{- if .AskQuestion.Skipped }}
用户跳过了选择，请继续执行任务。
{{- else }}
{{- range .AskQuestion.Questions }}
问题：{{ .Question }}
用户选择：{{ range $i, $s := .Selected }}{{ if $i }}、{{ end }}{{ $s }}{{ end }}
{{- end }}
{{- end }}
```
