# <search_results> · 1 个版本（按长度降序）


## 版本 1（194 字符）

```text
{{ range .refs }}
标题：{{ .Title }}
{{ if .SiteName }}来源：{{ .SiteName }}
{{ end }}{{ if .SourceType }}数据类型：{{ .SourceType }}
{{ end }}URL：{{ .Url }}
日期：{{ .TimeDesc }}
摘要：{{ .Summary }}

{{ end }}
```
