# <connectors> · 1 个版本（按长度降序）


## 版本 1（206 字符）

```text
{{ range $c := .Connectors }}  <connector>
    <id>{{ $c.ID }}</id>
    <name>{{ $c.Name }}</name>
    <type>{{ $c.Type }}</type>
    <description>{{ $c.Description }}</description>
  </connector>
{{ end }}
```
