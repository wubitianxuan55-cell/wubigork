# <skills> · 1 个版本（按长度降序）


## 版本 1（187 字符）

```text
{{ range .Skills }}
  <skill available="true">
    <name>{{ .Name }}</name>
    <description>{{ .Description }}</description>
    <location>{{ .Location }}</location>
  </skill>
{{ end }}
```
