# <connector_changes> · 1 个版本（按长度降序）


## 版本 1（613 字符）

```text
{{- if .ConnectorChanges.Open }}
  <用户新启用的连接器>
{{- range $c := .ConnectorChanges.Open }}
    <connector>
      <id>{{ $c.ID }}</id>
      <name>{{ $c.Name }}</name>
      <type>{{ $c.Type }}</type>
      <description>{{ $c.Description }}</description>
    </connector>
{{- end }}
  </用户新启用的连接器>
{{ end -}}
{{- if .ConnectorChanges.Close }}
  <用户新关闭的连接器>
{{- range $c := .ConnectorChanges.Close }}
    <connector>
      <id>{{ $c.ID }}</id>
      <name>{{ $c.Name }}</name>
      <type>{{ $c.Type }}</type>
      <description>{{ $c.Description }}</description>
    </connector>
{{- end }}
  </用户新关闭的连接器>
{{ end -}}
```
