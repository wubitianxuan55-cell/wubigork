# <user_referenced_cloud_docs> · 1 个版本（按长度降序）


## 版本 1（593 字符）

```text
{{- range $index, $doc := .CloudDocs }}
  <cloud_doc index="{{$index}}">
    <file_id>{{$doc.FileID}}</file_id>
{{- if $doc.Error }}
    <error>{{$doc.Error}}</error>
{{- else }}
    <name>{{$doc.Name}}</name>
    <drive_id>{{$doc.DriveID}}</drive_id>
    <link_url>{{$doc.LinkURL}}</link_url>
    <size>{{$doc.Size}}</size>
    <ctime>{{$doc.Ctime}}</ctime>
    <mtime>{{$doc.Mtime}}</mtime>
    <created_by id="{{$doc.CreatedBy.ID}}">{{$doc.CreatedBy.Name}}</created_by>
    <modified_by id="{{$doc.ModifiedBy.ID}}">{{$doc.ModifiedBy.Name}}</modified_by>
{{- end }}
  </cloud_doc>
{{- end }}
```
