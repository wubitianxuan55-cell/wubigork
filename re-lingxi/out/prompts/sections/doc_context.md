# <doc_context> · 1 个版本（按长度降序）


## 版本 1（675 字符）

```text
{{- if .DocContext.Path }}
  <path>{{xmlText .DocContext.Path}}</path>
{{- end }}
{{- if .DocContext.Name }}
  <name>{{xmlText .DocContext.Name}}</name>
{{- end }}
{{- if .DocContext.MIMEType }}
  <mime_type>{{xmlText .DocContext.MIMEType}}</mime_type>
{{- end }}
{{- if .DocContext.Selections }}
  <selections>
{{- range $index, $sel := .DocContext.Selections }}
    <selection index="{{index1 $index}}">
{{- if $sel.Selector }}
      <selector>{{xmlText $sel.Selector}}</selector>
{{- end }}
{{- if $sel.TagName }}
      <tag_name>{{xmlText $sel.TagName}}</tag_name>
{{- end }}
      <text>{{xmlText $sel.Text}}</text>
    </selection>
{{- end }}
  </selections>
{{- end }}
```
