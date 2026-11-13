# <runtime> · 4 个版本（按长度降序）


## 版本 1（478 字符）

```text
当前日期：{{ .Now }}
支持环境：
{{ if eq .ClientType "web" -}}
- 操作系统：{{ .OsType }}（具有互联网访问权限）
  - 用户：root（具有 sudo 权限，无需密码）
{{- else -}}
- 操作系统：{{ .OsType }}
{{- end -}}
- 命令执行环境：Git Bash
- 浏览器：Chromium 已就绪，可通过 browser SKILL 操作
- Python：可运行脚本，并可按需安装缺失的第三方依赖
- Node.js + npm：可运行脚本和按需安装包

预装依赖：
- Python：已安装常用第三方包（如 pandas、numpy、matplotlib、requests、beautifulsoup4、python-docx、python-pptx、openpyxl、Pillow 等），直接导入使用即可
{{ if eq .ClientType "web" }}- Node.js：docx、jszip、@xmldom/xmldom
{{ end }}
```

## 版本 2（474 字符）

```text
当前日期：{{ .Now }}
支持环境：
{{ if eq .ClientType "web" -}}
- 操作系统：{{ .OsType }}（具有互联网访问权限）
  - 用户：root（具有 sudo 权限，无需密码）
{{- else -}}
- 操作系统：{{ .OsType }}
{{- end -}}
- 命令执行环境：Git Bash
- 浏览器：Chromium 已就绪，可通过 browser SKILL 操作
- Python：已安装常用依赖，可按需安装缺失包
- Node.js + npm：可运行脚本和按需安装包

预装依赖：
- Python：已安装常用第三方包（如 pandas、numpy、matplotlib、requests、beautifulsoup4、python-docx、python-pptx、openpyxl、Pillow 等），直接导入使用即可
{{ if eq .ClientType "web" }}- Node.js：docx、jszip、@xmldom/xmldom
{{ end }}
```

## 版本 3（156 字符）

```text
当前日期：{{ .CurrentDate }}
{{ if .ProjectName -}}
项目名称：{{ .ProjectName }}
{{ end -}}
{{ if .ProjectInstructions -}}
项目说明：
{{ .ProjectInstructions }}
{{ end -}}
```

## 版本 4（89 字符）

```text
当前日期：{{ .CurrentDate }}
用户：{{ .UserName }}（ID: {{ .UserID }}），wiki/profile/ 下的画像页面归属于此用户。
```
